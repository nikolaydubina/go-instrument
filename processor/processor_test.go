package processor_test

import (
	"bytes"
	"go/format"
	"go/parser"
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
			fileName:      "../internal/testdata/basic.go",
			fileNameExp:   "../internal/testdata/instrumented/basic.go.exp",
			defaultSelect: true,
		},
		{
			app:           "app",
			fileName:      "../internal/testdata/basic_include_only.go",
			fileNameExp:   "../internal/testdata/instrumented/basic_include_only.go.exp",
			defaultSelect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.fileName, func(t *testing.T) {
			src, err := os.ReadFile(tc.fileName)
			if err != nil {
				t.Errorf("can not read input file: %s", err)
			}

			src, err = format.Source(src)
			if err != nil {
				t.Errorf("can not format input file: %s", err)
			}

			fset := token.NewFileSet()

			file, err := parser.ParseFile(fset, tc.fileName, src, parser.ParseComments)
			if err != nil || file == nil {
				t.Errorf("can not parse input file: %s", err)
			}

			// extract all commands from file comments
			commands, err := processor.CommandsFromFile(*file)
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

			if err := p.Process(fset, file); err != nil {
				t.Error(err)
			}

			// output
			var out bytes.Buffer
			if err := format.Node(&out, fset, file); err != nil {
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
