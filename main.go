package main

import (
	"errors"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"

	"github.com/nikolaydubina/go-instrument/instrument"
	"github.com/nikolaydubina/go-instrument/processor"
)

func main() {
	var (
		fileName      string
		overwrite     bool
		app           string
		skipGenerated bool
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.StringVar(&app, "app", "app", "name of application")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.BoolVar(&skipGenerated, "skip-generated", false, "skip generated files")
	flag.Parse()

	if err := process(fileName, app, overwrite, skipGenerated); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func process(fileName, app string, overwrite, skipGenerated bool) error {
	if fileName == "" {
		return errors.New("missing file name")
	}

	src, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	formattedSrc, err := format.Source(src)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, formattedSrc, parser.ParseComments)
	if err != nil || file == nil {
		return err
	}
	if skipGenerated && ast.IsGenerated(file) {
		return nil
	}

	p := processor.Processor{
		Instrumenter: &instrument.OpenTelemetry{
			TracerName:             app,
			ErrorStatusDescription: "error",
		},
		SpanName:       processor.BasicSpanName,
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorType:      `error`,
	}

	if err := p.Process(fset, file); err != nil {
		return err
	}

	var out io.Writer = os.Stdout
	if overwrite {
		outf, err := os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer outf.Close()
		out = outf
	}

	return format.Node(out, fset, file)
}
