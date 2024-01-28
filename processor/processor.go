package processor

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// Instrumenter supplies ast of Go code that will be inserted and required dependencies.
type Instrumenter interface {
	Imports() []string
	PrefixStatements(spanName string, hasError bool) []ast.Stmt
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

func functionLiteral(fnc *ast.FuncDecl, funcLitInst *funcTypeConditions) *ast.FuncLit {
	if funcLitInst == nil || len(fnc.Body.List) != 1 {
		return nil
	}

	returnStmt, stmtOk := fnc.Body.List[0].(*ast.ReturnStmt)
	if !stmtOk {
		return nil
	}

	funcLit, funcLitOk := returnStmt.Results[0].(*ast.FuncLit)
	if !funcLitOk || !funcLitInst.hasContext {
		return nil
	}
	funcLitInst = nil

	return funcLit
}

// funcTypeConditions collects details on functions and methods
type funcTypeConditions struct {
	Type       *ast.FuncType
	hasContext bool
	hasError   bool
}

func (p *Processor) hasPrefixConditions(fn *funcTypeConditions) {
	if t := fn.Type; t != nil {
		if ps := t.Params; ps != nil {
			for _, q := range ps.List {
				if q == nil {
					continue
				}
				fn.hasContext = fn.hasContext || p.isContext(*q)
			}
		}

		if rs := t.Results; rs != nil {
			for _, q := range rs.List {
				if q == nil {
					continue
				}
				fn.hasError = fn.hasError || p.isError(*q)
			}
		}
	}
}

// Processor traverses AST, collects details on functions and methods, and invokes Instrumenter
type Processor struct {
	Instrumenter     Instrumenter
	FunctionSelector FunctionSelector
	PackageName      string
	SpanName         func(receiver, function string) string
	ContextName      string
	ContextPackage   string
	ContextType      string
	ErrorName        string
	ErrorType        string
}

func (p *Processor) methodReceiverTypeName(spec ast.FuncDecl) string {
	// function
	if spec.Recv == nil {
		return ""
	}
	// method
	for _, v := range spec.Recv.List {
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

func (p *Processor) packageName(c *astutil.Cursor) {
	if c.Node() == nil && c.Name() == "Doc" {
		if f, ok := c.Parent().(*ast.File); ok {
			p.PackageName = f.Name.Name
		}
	}
}

func (p *Processor) functionName(spec ast.FuncDecl) string {
	if spec.Name == nil {
		return ""
	}
	return spec.Name.Name
}

func (p *Processor) isContext(e ast.Field) bool {
	// anonymous arg
	// multiple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != p.ContextName {
		return false
	}

	pkg := ""
	sym := ""

	if starExpr, starExprOk := e.Type.(*ast.StarExpr); starExprOk {
		if selectorExpr, selectorExprOk := starExpr.X.(*ast.SelectorExpr); selectorExprOk {
			pkg = selectorExpr.X.(*ast.Ident).Name
		}
		if selectorExprX, selectorExprXOk := starExpr.X.(*ast.SelectorExpr); selectorExprXOk && selectorExprX.Sel != nil {
			sym = selectorExprX.Sel.Name
		}
		if ident, identOk := starExpr.X.(*ast.Ident); identOk && ident.Obj != nil {
			sym = ident.Obj.Name
		}
	}

	if selectorExpr, selectorExprOk := e.Type.(*ast.SelectorExpr); selectorExprOk {
		if indent, indentOk := selectorExpr.X.(*ast.Ident); indentOk {
			pkg = indent.Name
		}
		if v := selectorExpr.Sel; v != nil {
			sym = v.Name
		}
	}

	return pkg == p.ContextPackage && sym == p.ContextType
}

func (p *Processor) isError(e ast.Field) bool {
	// anonymous arg
	// multiple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != p.ErrorName {
		return false
	}

	if v, ok := e.Type.(*ast.Ident); ok && v != nil {
		return v.Name == p.ErrorType
	}

	return false
}

func (p *Processor) Process(fset *token.FileSet, file *ast.File) error {
	var patches []patch
	var funcLitCond *funcTypeConditions

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		if c == nil {
			return true
		}

		p.packageName(c)

		switch c.Node().(type) {
		// original
		case *ast.FuncDecl:
			fnc, _ := c.Node().(*ast.FuncDecl)
			funcName := p.functionName(*fnc)
			if !p.FunctionSelector.AcceptFunction(funcName) {
				return true
			}

			receiverName := p.methodReceiverTypeName(*fnc)
			if receiverName == "" {
				receiverName = p.PackageName
			}

			spanName := p.SpanName(receiverName, funcName)
			funcCond := &funcTypeConditions{Type: fnc.Type}
			p.hasPrefixConditions(funcCond)

			if funcLit := functionLiteral(fnc, funcLitCond); funcLit != nil {
				ps := p.Instrumenter.PrefixStatements(spanName, funcLitCond.hasError)
				patches = append(patches, patch{pos: funcLit.Body.Pos(), stmts: ps})
			}

			if !funcCond.hasContext {
				return true
			}

			ps := p.Instrumenter.PrefixStatements(spanName, funcCond.hasError)
			patches = append(patches, patch{pos: fnc.Body.Pos(), stmts: ps})
		// anonymous
		case *ast.FuncLit:
			fnc, _ := c.Node().(*ast.FuncLit)
			funcLitCond = &funcTypeConditions{Type: fnc.Type}
			p.hasPrefixConditions(funcLitCond)
		default:
			return true
		}
		return true
	})

	if len(patches) > 0 {
		if err := p.patchFile(fset, file, patches...); err != nil {
			return err
		}

		for _, q := range p.Instrumenter.Imports() {
			astutil.AddImport(fset, file, q)
		}
	}

	return nil
}
