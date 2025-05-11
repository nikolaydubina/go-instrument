package processor_test

import (
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/nikolaydubina/go-instrument/processor"
)

func TestBuildConstraintsFromFile(t *testing.T) {
	tests := []struct {
		fileName string
		vs       []processor.BuildConstraint
	}{
		{
			fileName: "../internal/testdata/skipped_buildignore.go",
			vs: []processor.BuildConstraint{
				processor.BuildIgnore,
			},
		},
		{
			fileName: "../internal/testdata/skipped_gobuildignore.go",
			vs: []processor.BuildConstraint{
				processor.BuildIgnore,
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

			vs := processor.BuildConstraintsFromFile(*f)

			if !slices.Equal(vs, tc.vs) {
				t.Error(vs, tc.vs)
			}
		})
	}
}

func TestParseBuildConstraint(t *testing.T) {
	tests := []struct {
		s string
		v processor.BuildConstraint
	}{
		{
			s: "//go:build ignore",
			v: processor.BuildIgnore,
		},
		{
			s: "//go:build exclude",
			v: processor.BuildExclude,
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
			v: processor.BuildIgnore,
		},
		{
			s: "//go:build     exclude    ",
			v: processor.BuildExclude,
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
			if v := processor.ParseBuildConstraint(tc.s); v != tc.v {
				t.Error(v)
			}
		})
	}
}
