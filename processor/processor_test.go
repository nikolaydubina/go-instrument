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
		testName      string
		app           string
		fileName      string
		fileNameExp   string
		defaultSelect bool
	}{
		{
			testName:      "basic with default select",
			app:           "app",
			fileName:      "../internal/example/basic.go",
			fileNameExp:   "../internal/example/instrumented/basic.go.exp",
			defaultSelect: true,
		},
		{
			testName:      "basic with include only",
			app:           "app",
			fileName:      "../internal/example/basic_include_only.go",
			fileNameExp:   "../internal/example/instrumented/basic_include_only.go.exp",
			defaultSelect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
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

			if err = p.Process(fset, file); err != nil {
				t.Error(err)
			}

			// output
			var out bytes.Buffer
			if err = format.Node(&out, fset, file); err != nil {
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

func TestCustomContextProcessor(t *testing.T) {
	tests := []struct {
		testName      string
		processor     processor.Processor
		fileName      string
		fileNameExp   string
		defaultSelect bool
	}{
		{
			testName: "basic custom pkg context with default select",
			processor: processor.Processor{
				Instrumenter: &instrument.OpenTelemetry{
					TracerName:  "app",
					ContextName: "c",
					ErrorName:   "err",
				},
				SpanName:       processor.BasicSpanName,
				ContextName:    "c",
				ContextPackage: "custom",
				ContextType:    "Context",
				ErrorName:      "err",
				ErrorType:      `error`,
			},
			fileName:      "../internal/example/custom_context.go",
			fileNameExp:   "../internal/example/instrumented/pkg_custom_context.go.exp",
			defaultSelect: true,
		},
		{
			testName: "basic custom context with default select",
			processor: processor.Processor{
				Instrumenter: &instrument.OpenTelemetry{
					TracerName:  "app",
					ContextName: "ctx",
					ErrorName:   "err",
				},
				SpanName:       processor.BasicSpanName,
				ContextName:    "ctx",
				ContextPackage: "",
				ContextType:    "exampleContext",
				ErrorName:      "err",
				ErrorType:      `error`,
			},
			fileName:      "../internal/example/custom_context.go",
			fileNameExp:   "../internal/example/instrumented/custom_context.go.exp",
			defaultSelect: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
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

			p := tc.processor
			p.FunctionSelector = processor.NewMapFunctionSelectorFromCommands(tc.defaultSelect, commands)

			if err = p.Process(fset, file); err != nil {
				t.Error(err)
			}

			// output
			var out bytes.Buffer
			if err = format.Node(&out, fset, file); err != nil {
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
