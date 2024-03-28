package processor_test

import (
	"go/parser"
	"go/token"
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
				t.Errorf("can not parse input file: %s", err)
			}
			directives := processor.GoBuildDirectivesFromFile(*f)
			if len(directives) != len(tc.directives) {
				t.Errorf("exp(%#v) != (%#v)", tc.directives, directives)
			}
			for i := range directives {
				if directives[i] != tc.directives[i] {
					t.Errorf("exp(%#v) != (%#v)", tc.directives, directives)
				}
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
				t.Errorf("exp(%v) != (%v)", tc.v, v)
			}
		})
	}
}
