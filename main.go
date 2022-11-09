package main

import (
	"flag"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"

	"github.com/nikolaydubina/go-instrument/instrument"
	"github.com/nikolaydubina/go-instrument/processor"
)

func main() {
	var (
		fileName   string
		overwrite  bool
		tracerName string
		verbosity  int
	)
	flag.StringVar(&fileName, "file", "", "go file to instrument")
	flag.StringVar(&tracerName, "tracer-name", "app", "name of tracer")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.IntVar(&verbosity, "v", 0, "verbositry of STDERR logs")
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
		TracerName:  tracerName,
		ContextName: "ctx",
		ErrorName:   "error",
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

	printer.Fprint(os.Stdout, fset, file)
}
