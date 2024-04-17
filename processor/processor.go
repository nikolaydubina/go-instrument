package processor

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/ast/astutil"
)

// Instrumenter supplies ast of Go code that will be inserted and required dependencies.
type Instrumenter interface {
	Imports() []*types.Package
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

// Processor traverses AST, collects details on functions and methods, and invokes Instrumenter
type Processor struct {
	Instrumenter     Instrumenter
	FunctionSelector FunctionSelector
	SpanName         func(receiver, function string) string
	ContextName      string
	ContextPackage   string
	ContextType      string
	ErrorName        string
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
	// anonymous arg
	// multiple symbols
	// strange symbol
	if e == nil || len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != p.ContextName {
		return false
	}

	pkg := ""
	sym := ""

	if se, ok := e.Type.(*ast.SelectorExpr); ok && se != nil {
		if v, ok := se.X.(*ast.Ident); ok && v != nil {
			pkg = v.Name
		}
		if v := se.Sel; v != nil {
			sym = v.Name
		}
	}

	return pkg == p.ContextPackage && sym == p.ContextType
}

func (p *Processor) isError(e *ast.Field) bool {
	if e == nil {
		return false
	}
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

func (p *Processor) functionHasError(fnType *ast.FuncType) bool {
	if fnType == nil {
		return false
	}

	if rs := fnType.Results; rs != nil {
		for _, q := range rs.List {
			if p.isError(q) {
				return true
			}
		}
	}

	return false
}

func (p *Processor) Process(fset *token.FileSet, file *ast.File) error {
	var patches []patch

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		if c == nil {
			return true
		}

		var receiver, fname string
		var fnType *ast.FuncType
		var fnBody *ast.BlockStmt

		switch fn := c.Node().(type) {
		case *ast.FuncLit:
			fnType, fnBody = fn.Type, fn.Body
			fname = "anonymous"
		case *ast.FuncDecl:
			fnType, fnBody = fn.Type, fn.Body
			fname = p.functionName(fn)
			receiver = p.methodReceiverTypeName(fn)
		default:
			return true
		}

		if !p.FunctionSelector.AcceptFunction(fname) {
			return true
		}

		if p.functionHasContext(fnType) {
			ps := p.Instrumenter.PrefixStatements(p.SpanName(receiver, fname), p.functionHasError(fnType))
			patches = append(patches, patch{pos: fnBody.Pos(), stmts: ps})
		}

		return true
	})

	if len(patches) > 0 {
		// Patches must be applied in the ascending order, otherwise the
		// modified source file will become corrupted.
		sort.Slice(patches, func(i, j int) bool { return patches[i].pos < patches[j].pos })

		if err := patchFile(fset, file, patches...); err != nil {
			return err
		}

		for _, pkg := range p.Instrumenter.Imports() {
			astutil.AddNamedImport(fset, file, pkg.Name(), pkg.Path())
		}
	}

	return nil
}
