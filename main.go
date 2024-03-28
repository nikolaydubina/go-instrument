package main

import (
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
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

	src, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("can not read input file: %s", err)
	}

	formattedSrc, err := format.Source(src)
	if err != nil {
		log.Fatalf("can not format input file: %s", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, formattedSrc, parser.ParseComments)
	if err != nil || file == nil {
		log.Fatalf("can not parse input file: %s", err)
	}
	if skipGenerated && ast.IsGenerated(file) {
		log.Fatalf("skipping generated file")
	}

	directives := processor.GoBuildDirectivesFromFile(*file)
	for _, q := range directives {
		if q.SkipFile() {
			return
		}
	}

	commands, err := processor.CommandsFromFile(*file)
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

	if err := p.Process(fset, file); err != nil {
		log.Fatal(err)
	}

	var out io.Writer = os.Stdout
	if overwrite {
		outf, err := os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			log.Fatal(err)
		}
		defer func(outf *os.File) {
			if err := outf.Close(); err != nil {
				log.Fatal(err)
			}
		}(outf)
		out = outf
	}

	if err := format.Node(out, fset, file); err != nil {
		log.Fatal(err)
	}
}
