package processor_test

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"testing"

	"github.com/nikolaydubina/go-instrument/instrument"
	"github.com/nikolaydubina/go-instrument/processor"
)

func TestProcessor(t *testing.T) {
	app := "app"
	fileName := "../internal/example/basic.go"
	fileNameExp := "../internal/example/instrumented/basic.go.out"

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
		t.Errorf("can not parse input file: %s", err)
	}
	commands, err := processor.CommandsFromFile(*fileWithComments)
	if err != nil {
		t.Error(err)
	}
	p.ApplyCommand(commands...)

	// process without comments
	file, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil || file == nil {
		t.Errorf("can not parse input file: %s", err)
	}
	if err := p.Process(fset, *file); err != nil {
		t.Error(err)
	}

	// output
	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, file); err != nil {
		t.Error(err)
	}

	// compare
	expFile, err := os.ReadFile(fileNameExp)
	if err != nil {
		t.Error(err)
	}
	if string(expFile) != out.String() {
		t.Error("diff output")
	}
}
