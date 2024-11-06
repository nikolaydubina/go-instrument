package instrument_test

import (
	"bytes"
	_ "embed"
	"go/printer"
	"go/token"
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
		ContextName:            "ctx",
		ErrorStatusDescription: "error",
	}
	c := p.PrefixStatements("myClass.MyFunction", true, "err")

	var out bytes.Buffer
	printer.Fprint(&out, token.NewFileSet(), c)

	if s := out.String(); s != expOpenTelemetryError {
		t.Errorf("%s", s)
	}

	expImports := map[string]bool{
		"go.opentelemetry.io/otel ":                true,
		"go.opentelemetry.io/otel/codes otelCodes": true,
	}
	imports := p.Imports()
	for _, pkg := range imports {
		if !expImports[pkg.Path()+" "+pkg.Name()] {
			t.Errorf("wrong import")
		}
	}
	if len(imports) != len(expImports) {
		t.Error("wrong imports")
	}
}
func TestOpenTelemetry(t *testing.T) {
	p := instrument.OpenTelemetry{
		TracerName:             "app",
		ContextName:            "ctx",
		ErrorStatusDescription: "error",
	}
	c := p.PrefixStatements("myClass.MyFunction", false, "err")

	var out bytes.Buffer
	printer.Fprint(&out, token.NewFileSet(), c)

	if s := out.String(); s != expOpenTelemetry {
		t.Errorf("got(%v) != exp(%v)", s, expOpenTelemetry)
	}

	expImports := map[string]bool{
		"go.opentelemetry.io/otel": true,
	}
	imports := p.Imports()
	for _, pkg := range imports {
		if !expImports[pkg.Path()+pkg.Name()] {
			t.Errorf("wrong import")
		}
	}
	if len(imports) != len(expImports) {
		t.Error("wrong imports")
	}
}
