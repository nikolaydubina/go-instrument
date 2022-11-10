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

	fset := token.NewFileSet()

	// extract all commands from file comments
	fileWithComments, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil || fileWithComments == nil {
		log.Fatalf("can not parse input file: %s", err)
	}
	commands, err := processor.CommandsFromFile(*fileWithComments)
	if err != nil {
		log.Fatal(err)
	}
	p.ApplyCommand(commands...)

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
