package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"log"
	"os"

	"github.com/nikolaydubina/go-instrument/instrument"
	"github.com/nikolaydubina/go-instrument/processor"
)

func main() {
	var (
		fileName      string
		overwrite     bool
		app           string
		defaultSelect bool
		skipGenerated bool
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.StringVar(&app, "app", "app", "name of application")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.BoolVar(&defaultSelect, "all", true, "instrument all by default")
	flag.BoolVar(&skipGenerated, "skip-generated", false, "skip generated files")
	flag.Parse()

	if fileName == "" {
		log.Fatalln("missing arg: file name")
	}

	fset := token.NewFileSet()

	// extract all commands from file comments
	fileWithComments, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil || fileWithComments == nil {
		log.Fatalf("can not parse input file: %s", err)
	}
	if skipGenerated && ast.IsGenerated(fileWithComments) {
		log.Fatalf("skipping generated file")
	}

	directives := processor.GoBuildDirectivesFromFile(*fileWithComments)
	for _, q := range directives {
		if q.SkipFile() {
			return
		}
	}

	commands, err := processor.CommandsFromFile(*fileWithComments)
	if err != nil {
		log.Fatal(err)
	}
	functionSelector := processor.NewMapFunctionSelectorFromCommands(defaultSelect, commands)

	var instrumenter processor.Instrumenter = &instrument.OpenTelemetry{
		TracerName:  app,
		ContextName: "ctx",
		ErrorName:   "err",
	}
	p := processor.Processor{
		Instrumenter:     instrumenter,
		FunctionSelector: functionSelector,
		SpanName:         processor.BasicSpanName,
		ContextName:      "ctx",
		ContextPackage:   "context",
		ContextType:      "Context",
		ErrorName:        "err",
		ErrorType:        `error`,
	}

	// process without comments
	file, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil || file == nil {
		log.Fatalf("can not parse input file: %s", err)
	}
	if err := p.Process(fset, *file); err != nil {
		log.Fatal(err)
	}

	// output
	var out io.Writer = os.Stdout
	if overwrite {
		out, err = os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := printer.Fprint(out, fset, file); err != nil {
		log.Fatal(err)
	}
}
