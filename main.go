package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/nikolaydubina/go-instrument/instrument"
)

const (
	contextName    = "ctx"
	contextPackage = "context"
	contextType    = "Context"
	errorName      = "err"
	errorType      = `error`
)

func methodReceiverTypeName(spec ast.FuncDecl) string {
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

func functionName(spec ast.FuncDecl) string {
	if spec.Name == nil {
		return ""
	}
	return spec.Name.Name
}

func spanName(methodReciverName, functionName string) string {
	if methodReciverName == "" {
		return functionName
	}
	return methodReciverName + "." + functionName
}

func isContext(e ast.Field) bool {
	// anonymous arg
	// multilple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != contextName {
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

	return pkg == contextPackage && sym == contextType
}

func isError(e ast.Field) bool {
	// anonymous arg
	// multilple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != errorName {
		return false
	}

	if v, ok := e.Type.(*ast.Ident); ok && v != nil {
		return v.Name == errorType
	}

	return false
}

const (
	frameworkOpenTelemetry = "open-telemetry"
)

type Instrumenter interface {
	Imports() []string
	PrefixStatements(spanName string, hasError bool) []ast.Stmt
}

func main() {
	var (
		fileName   string
		overwrite  bool
		tracerName string
		verbosity  int
		framework  string
	)
	flag.StringVar(&fileName, "file", "", "go file to instrument")
	flag.StringVar(&tracerName, "tracer-name", "app", "name of tracer")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.IntVar(&verbosity, "v", 0, "verbositry of STDERR logs")
	flag.StringVar(&framework, "framework", frameworkOpenTelemetry, `framework for instrumentation ("`+frameworkOpenTelemetry+`")`)
	flag.Parse()

	fset := token.NewFileSet()

	if fileName == "" {
		log.Fatalln("missing arg: file name")
	}

	file, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil {
		log.Fatalf("can not parse input file: %s", err)
	}

	var instrumenter Instrumenter = &instrument.OpenTelemetry{
		TracerName:  tracerName,
		ContextName: contextName,
		ErrorName:   errorName,
	}

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		if c == nil {
			return true
		}

		fn, ok := c.Node().(*ast.FuncDecl)
		if !ok || fn == nil {
			return true
		}

		spanName := spanName(methodReceiverTypeName(*fn), functionName(*fn))

		hasContext := false
		hasError := false

		if t := fn.Type; t != nil {
			if ps := t.Params; ps != nil {
				for _, q := range ps.List {
					if q == nil {
						continue
					}
					hasContext = hasContext || isContext(*q)
				}
			}

			if rs := t.Results; rs != nil {
				for _, q := range rs.List {
					if q == nil {
						continue
					}
					hasError = hasError || isError(*q)
				}
			}
		}

		if verbosity > 0 {
			log.Printf("%s: has_context(%t) has_error(%t)\n", spanName, hasContext, hasError)
		}

		if !hasContext {
			return true
		}

		fn.Body.List = append(instrumenter.PrefixStatements(spanName, hasError), fn.Body.List...)

		c.Replace(fn)

		return true
	})

	for _, q := range instrumenter.Imports() {
		astutil.AddImport(fset, file, q)
	}

	printer.Fprint(os.Stdout, fset, file)
}
