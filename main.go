package main

import (
	"flag"
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
		fileName  string
		overwrite bool
		app       string
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.StringVar(&app, "app", "app", "name of application")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.Parse()

	if fileName == "" {
		log.Fatalln("missing arg: file name")
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil {
		log.Fatalf("can not parse input file: %s", err)
	}

	var instrumenter processor.Instrumenter = &instrument.OpenTelemetry{
		TracerName:  app,
		ContextName: "ctx",
		ErrorName:   "err",
	}
	p := processor.Processor{
		Instrumenter:   instrumenter,
		SpanName:       processor.BasicSpanName,
		ContextName:    "ctx",
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorName:      "err",
		ErrorType:      `error`,
	}
	p.Process(fset, file)

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
