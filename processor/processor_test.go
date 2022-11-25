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
	tests := []struct {
		app           string
		fileName      string
		fileNameExp   string
		defaultSelect bool
	}{
		{
			app:           "app",
			fileName:      "../internal/example/basic.go",
			fileNameExp:   "../internal/example/instrumented/basic.go.exp",
			defaultSelect: true,
		},
		{
			app:           "app",
			fileName:      "../internal/example/basic_include_only.go",
			fileNameExp:   "../internal/example/instrumented/basic_include_only.go.exp",
			defaultSelect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.fileName, func(t *testing.T) {
			fset := token.NewFileSet()

			// extract all commands from file comments
			fileWithComments, err := parser.ParseFile(fset, tc.fileName, nil, parser.ParseComments)
			if err != nil || fileWithComments == nil {
				t.Errorf("can not parse input file: %s", err)
			}
			commands, err := processor.CommandsFromFile(*fileWithComments)
			if err != nil {
				t.Error(err)
			}
			functionSelector := processor.NewMapFunctionSelectorFromCommands(tc.defaultSelect, commands)

			var instrumenter processor.Instrumenter = &instrument.OpenTelemetry{
				TracerName:  tc.app,
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
			file, err := parser.ParseFile(fset, tc.fileName, nil, 0)
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
			expFile, err := os.ReadFile(tc.fileNameExp)
			if err != nil {
				t.Error(err)
			}
			if string(expFile) != out.String() {
				t.Error("diff output", out.String())
			}
		})
	}
}
