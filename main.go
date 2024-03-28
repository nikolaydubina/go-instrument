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
		defaultSelect bool
		skipGenerated bool
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.StringVar(&app, "app", "app", "name of application")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.BoolVar(&defaultSelect, "all", true, "instrument all by default")
	flag.BoolVar(&skipGenerated, "skip-generated", false, "skip generated files")
	flag.Parse()

	if err := process(fileName, app, overwrite, defaultSelect, skipGenerated); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func process(fileName, app string, overwrite, defaultSelect, skipGenerated bool) error {
	if fileName == "" {
		return errors.New("missing arg: file name")
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

	// format already ensures that file is parsable
	file, _ := parser.ParseFile(fset, fileName, formattedSrc, parser.ParseComments)
	if skipGenerated && ast.IsGenerated(file) {
		return errors.New("skipping generated file")
	}

	directives := processor.GoBuildDirectivesFromFile(*file)
	for _, q := range directives {
		if q.SkipFile() {
			return nil
		}
	}

	commands, err := processor.CommandsFromFile(*file)
	if err != nil {
		return err
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
