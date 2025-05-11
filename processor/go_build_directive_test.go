package processor_test

import (
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/nikolaydubina/go-instrument/processor"
)

func TestGoBuildDirectivesFromFile(t *testing.T) {
	tests := []struct {
		fileName   string
		directives []processor.GoBuildDirective
	}{
		{
			fileName: "../internal/testdata/skipped_buildignore.go",
			directives: []processor.GoBuildDirective{
				processor.GoBuildIgnore,
				processor.BuildIgnore,
			},
		},
		{
			fileName: "../internal/testdata/skipped_gobuildignore.go",
			directives: []processor.GoBuildDirective{
				processor.GoBuildIgnore,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.fileName, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tc.fileName, nil, parser.ParseComments)
			if err != nil || f == nil {
				t.Error(err)
			}

			directives := processor.GoBuildDirectivesFromFile(*f)

			if !slices.Equal(directives, tc.directives) {
				t.Error(directives)
			}
		})
	}
}

func TestParseGoBuildDirective(t *testing.T) {
	tests := []struct {
		s string
		v processor.GoBuildDirective
	}{
		{
			s: "//go:build ignore",
			v: processor.GoBuildIgnore,
		},
		{
			s: "//go:build exclude",
			v: processor.GoBuildExclude,
		},
		{
			s: "// +build ignore",
			v: processor.BuildIgnore,
		},
		{
			s: "// +build exclude",
			v: processor.BuildExclude,
		},
		// whitespace
		{
			s: "//go:build    ignore   ",
			v: processor.GoBuildIgnore,
		},
		{
			s: "//go:build     exclude    ",
			v: processor.GoBuildExclude,
		},
		{
			s: "//    +build    ignore    ",
			v: processor.BuildIgnore,
		},
		{
			s: "//    +build    exclude   ",
			v: processor.BuildExclude,
		},
	}
	for _, tc := range tests {
		t.Run(tc.s, func(t *testing.T) {
			if v := processor.ParseGoBuildDirective(tc.s); v != tc.v {
				t.Error(v)
			}
		})
	}
}
