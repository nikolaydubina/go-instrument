package processor

import (
	"github.com/nikolaydubina/go-instrument/instrument"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/astutil"
)

// Instrumenter supplies ast of Go code that will be inserted and required dependencies.
type Instrumenter interface {
	Imports() []*types.Package
	PrefixStatements(spanName string, hasError bool, errName string) []ast.Stmt
}

// FunctionSelector tells if function has to be instrumented.
type FunctionSelector interface {
	AcceptFunction(functionName string) bool
}

// BasicSpanName is common notation of <class>.<method> or <pkg>.<func>
func BasicSpanName(receiver, function string) string {
	if receiver == "" {
		return function
	}
	return receiver + "." + function
}

// Processor traverses AST, collects details on functions and methods, and invokes Instrumenter
type Processor struct {
	Instrumenter     Instrumenter
	FunctionSelector FunctionSelector
	SpanName         func(receiver, function string) string
	ContextName      string
	ContextPackage   string
	ContextType      string
	ErrorType        string
}

func (p *Processor) methodReceiverTypeName(fn *ast.FuncDecl) string {
	// function
	if fn == nil || fn.Recv == nil {
		return ""
	}
	// method
	for _, v := range fn.Recv.List {
		if v == nil {
			continue
		}
		t := v.Type
		// pointer receiver
		if v, ok := v.Type.(*ast.StarExpr); ok {
			t = v.X
		}
		// value/pointer receiver
		if v, ok := t.(*ast.Ident); ok {
			return v.Name
		}
	}
	return ""
}

func (p *Processor) functionName(fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	return fn.Name.Name
}

func (p *Processor) isContext(e *ast.Field) bool {
	if e == nil || len(e.Names) == 0 {
		return false
	}

	pkg, sym := parseSelector(e.Type)
	return pkg == p.ContextPackage && sym == p.ContextType
}

func parseSelector(expr ast.Expr) (pkg, sym string) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", ""
	}
	return ident.Name, sel.Sel.Name
}

func (p *Processor) isError(e *ast.Field) (ok bool, name string) {
	if e == nil {
		return false, ""
	}
	// anonymous arg
	// multiple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false, ""
	}
	if v, ok := e.Type.(*ast.Ident); ok && v != nil {
		return v.Name == p.ErrorType, e.Names[0].Name
	}
	return false, ""
}

func (p *Processor) functionHasContext(fnType *ast.FuncType) bool {
	if fnType == nil {
		return false
	}

	if ps := fnType.Params; ps != nil {
		for _, q := range ps.List {
			if p.isContext(q) {
				return true
			}
		}
	}

	return false
}

func (p *Processor) getContextParam(fnType *ast.FuncType) (string, bool) {
	if fnType == nil {
		return "", false
	}

	if fnType.Params == nil {
		return "", false
	}

	for _, param := range fnType.Params.List {
		if p.isContext(param) {
			return param.Names[0].Name, true
		}
	}
	return "", false
}

func (p *Processor) functionHasError(fnType *ast.FuncType) (ok bool, name string) {
	if fnType == nil {
		return false, ""
	}

	if rs := fnType.Results; rs != nil {
		for _, q := range rs.List {
			if ok, name := p.isError(q); ok {
				return true, name
			}
		}
	}

	return false, ""
}

func (p *Processor) Process(fset *token.FileSet, file *ast.File) error {
	var patches []patch

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		var receiver, fname string
		var fnType *ast.FuncType
		var fnBody *ast.BlockStmt
		var hasContextRecv bool
		var ctxName string
		var receiverName string

		switch fn := c.Node().(type) {
		case *ast.FuncLit:
			fnType, fnBody = fn.Type, fn.Body
			fname = "anonymous"
		case *ast.FuncDecl:
			fnType, fnBody = fn.Type, fn.Body
			fname = p.functionName(fn)
			receiver = p.methodReceiverTypeName(fn)

			// Check receiver context only for FuncDecl
			if fn.Recv != nil {
				receiverContext, hasContextInReceiver, contextCount := p.getContextInReceiver(fn.Recv, file)
				if hasContextInReceiver {
					if contextCount > 1 {
						return true
					}
					ctxName = receiverContext
					hasContextRecv = true

					// Get the receiver variable name for prefixing
					if len(fn.Recv.List) > 0 && len(fn.Recv.List[0].Names) > 0 {
						receiverName = fn.Recv.List[0].Names[0].Name
					}
				}
			}
		default:
			return true
		}

		// General context checks
		paramCtx, hasParam := p.getContextParam(fnType)
		if hasParam {
			ctxName = paramCtx
		}

		hasContext := isFunctionHaveContext(hasParam, hasContextRecv)
		if !hasContext {
			return true
		}

		if hasContext && p.isFunctionInstrumented(fnBody) {
			return true
		}

		// Skip functions with multiple contexts
		if p.functionHasMultipleContexts(fnType) || (hasParam && hasContextRecv) {
			return true
		}

		// Check if function should be instrumented according to function selector
		if !p.FunctionSelector.AcceptFunction(fname) {
			return true
		}

		// Update Instrumenter's context name with proper prefix if it's from a receiver
		contextVarName := ctxName
		if hasContextRecv && receiverName != "" {
			contextVarName = receiverName + "." + ctxName
		}

		// Update Instrumenter's context name
		switch instr := p.Instrumenter.(type) {
		case *instrument.OpenTelemetry:
			instr.ContextName = contextVarName
		}

		// Inject instrumentation
		hasError, errorName := p.functionHasError(fnType)
		spanName := p.SpanName(receiver, fname)
		stmts := p.Instrumenter.PrefixStatements(spanName, hasError, errorName)
		patches = append(patches, patch{pos: fnBody.Pos(), stmts: stmts})

		return true
	})

	if len(patches) > 0 {
		if err := patchFile(fset, file, patches...); err != nil {
			return err
		}
		for _, pkg := range p.Instrumenter.Imports() {
			astutil.AddNamedImport(fset, file, pkg.Name(), pkg.Path())
		}
	}

	return nil
}

func isFunctionHaveContext(hasParam bool, hasContextRecv bool) bool {
	return hasParam || hasContextRecv
}

// getStructType resolves struct type from receiver (handles pointer receivers)
func getStructType(expr ast.Expr, file *ast.File) *ast.StructType {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return findStructType(file, ident.Name)
	}
	return nil
}

// findStructType finds struct definition by name in the same file
func findStructType(file *ast.File, name string) *ast.StructType {
	for _, decl := range file.Decls { // Now 'file' is accessible
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == name {
					if st, ok := ts.Type.(*ast.StructType); ok {
						return st
					}
				}
			}
		}
	}
	return nil
}

// getContextInReceiver checks if the receiver's struct has a context.Context field
func (p *Processor) getContextInReceiver(recv *ast.FieldList, file *ast.File) (string, bool, int) {
	if recv == nil || len(recv.List) == 0 {
		return "", false, 0
	}

	recvType := recv.List[0].Type
	structType := getStructType(recvType, file)
	if structType == nil {
		return "", false, 0
	}

	contextCount := 0
	var firstName string

	for _, field := range structType.Fields.List {
		if p.isContext(field) {
			contextCount++
			if firstName == "" && len(field.Names) > 0 {
				firstName = field.Names[0].Name
			}
		}
	}

	return firstName, contextCount > 0, contextCount
}

// functionHasMultipleContexts checks if a function has more than one context
// parameter in its argument list
func (p *Processor) functionHasMultipleContexts(fnType *ast.FuncType) bool {
	if fnType == nil || fnType.Params == nil {
		return false
	}

	contextCount := 0
	for _, param := range fnType.Params.List {
		if pkg, sym := parseSelector(param.Type); pkg == p.ContextPackage && sym == p.ContextType {
			contextCount += len(param.Names) // Count all names sharing this type
			if contextCount > 1 {
				return true
			}
		}
	}
	return false
}

func (p *Processor) isFunctionInstrumented(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) < 2 {
		return false
	}

	// Check first statement: ctx, span := otel.Tracer(...).Start(...)
	assignStmt, ok := body.List[0].(*ast.AssignStmt)
	if !ok || assignStmt == nil || len(assignStmt.Lhs) != 2 || len(assignStmt.Rhs) != 1 {
		return false
	}
	ctxIdent, ok1 := assignStmt.Lhs[0].(*ast.Ident)
	spanIdent, ok2 := assignStmt.Lhs[1].(*ast.Ident)
	if !ok1 || !ok2 || ctxIdent == nil || spanIdent == nil || ctxIdent.Name != p.ContextName || spanIdent.Name != "span" {
		return false
	}

	// Check second statement: defer span.End()
	deferStmt, ok := body.List[1].(*ast.DeferStmt)
	if !ok || deferStmt == nil {
		return false
	}
	callExpr, ok := deferStmt.Call.Fun.(*ast.SelectorExpr)
	if !ok || callExpr == nil || callExpr.Sel.Name != "End" {
		return false
	}
	spanVar, ok := callExpr.X.(*ast.Ident)
	return ok && spanVar != nil && spanVar.Name == "span"
}
