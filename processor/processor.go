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

// Processor traverses AST, collects details on funtions and methods, and invokes Instrumenter
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
		// poitner receiver
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

func (p *Processor) functionName(spec ast.FuncDecl) string {
	if spec.Name == nil {
		return ""
	}
	return spec.Name.Name
}

func (p *Processor) isContext(e ast.Field) bool {
	// anonymous arg
	// multilple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
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

func (p *Processor) isError(e ast.Field) bool {
	// anonymous arg
	// multilple symbols
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

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		if c == nil {
			return true
		}

		fn, ok := c.Node().(*ast.FuncDecl)
		if !ok || fn == nil {
			return true
		}

		fname := p.functionName(*fn)
		if !p.FunctionSelector.AcceptFunction(fname) {
			return true
		}

		spanName := p.SpanName(p.methodReceiverTypeName(*fn), fname)

		hasContext := false
		hasError := false

		if t := fn.Type; t != nil {
			if ps := t.Params; ps != nil {
				for _, q := range ps.List {
					if q == nil {
						continue
					}
					hasContext = hasContext || p.isContext(*q)
				}
			}

			if rs := t.Results; rs != nil {
				for _, q := range rs.List {
					if q == nil {
						continue
					}
					hasError = hasError || p.isError(*q)
				}
			}
		}

		if !hasContext {
			return true
		}

		ps := p.Instrumenter.PrefixStatements(spanName, hasError)
		patches = append(patches, patch{pos: fn.Body.Pos(), stmts: ps})

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
