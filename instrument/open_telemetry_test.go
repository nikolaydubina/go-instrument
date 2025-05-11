package instrument_test

import (
	"bytes"
	_ "embed"
	"go/printer"
	"go/token"
	"go/types"
	"maps"
	"testing"

	"github.com/nikolaydubina/go-instrument/instrument"
)

//go:embed testdata/open_telemetry_error.go
var expOpenTelemetryError string

//go:embed testdata/open_telemetry.go
var expOpenTelemetry string

func TestOpenTelemetry_Error(t *testing.T) {
	p := instrument.OpenTelemetry{
		TracerName:             "app",
		ErrorStatusDescription: "error",
	}
	c := p.PrefixStatements("myClass.MyFunction", "ctx", true, "err")

	var out bytes.Buffer
	printer.Fprint(&out, token.NewFileSet(), c)

	if s := out.String(); s != expOpenTelemetryError {
		t.Error(s)
	}

	imports := p.Imports()

	expImportPaths := map[string]bool{
		"go.opentelemetry.io/otel ":                true,
		"go.opentelemetry.io/otel/codes otelCodes": true,
	}
	importPaths := importPathsFromImports(imports)

	if !maps.Equal(expImportPaths, importPaths) {
		t.Error(importPaths)
	}
}

func TestOpenTelemetry(t *testing.T) {
	p := instrument.OpenTelemetry{
		TracerName:             "app",
		ErrorStatusDescription: "error",
	}
	c := p.PrefixStatements("myClass.MyFunction", "ctx", false, "err")

	var out bytes.Buffer
	printer.Fprint(&out, token.NewFileSet(), c)

	if s := out.String(); s != expOpenTelemetry {
		t.Error(s)
	}

	imports := p.Imports()

	expImportPaths := map[string]bool{
		"go.opentelemetry.io/otel ": true,
	}
	importPaths := importPathsFromImports(imports)

	if !maps.Equal(expImportPaths, importPaths) {
		t.Error(importPaths)
	}
}

func importPathsFromImports(imports []*types.Package) map[string]bool {
	importPaths := make(map[string]bool, len(imports))
	for _, pkg := range imports {
		importPaths[pkg.Path()+" "+pkg.Name()] = true
	}
	return importPaths
}
