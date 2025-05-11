package processor

import (
	"go/parser"
	"go/token"
	"slices"
	"testing"
)

func TestBuildConstraintsFromFile(t *testing.T) {
	tests := []struct {
		fileName string
		vs       []buildConstraint
	}{
		{
			fileName: "../internal/testdata/skipped_buildignore.go",
			vs: []buildConstraint{
				buildIgnore,
			},
		},
		{
			fileName: "../internal/testdata/skipped_gobuildignore.go",
			vs: []buildConstraint{
				buildIgnore,
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

			vs := buildConstraintsFromFile(*f)

			if !slices.Equal(vs, tc.vs) {
				t.Error(vs, tc.vs)
			}
		})
	}
}

func TestParseBuildConstraint(t *testing.T) {
	tests := []struct {
		s string
		v buildConstraint
	}{
		{
			s: "//go:build ignore",
			v: buildIgnore,
		},
		{
			s: "//go:build exclude",
			v: buildExclude,
		},
		{
			s: "// +build ignore",
			v: buildIgnore,
		},
		{
			s: "// +build exclude",
			v: buildExclude,
		},
		// whitespace
		{
			s: "//go:build    ignore   ",
			v: buildIgnore,
		},
		{
			s: "//go:build     exclude    ",
			v: buildExclude,
		},
		{
			s: "//    +build    ignore    ",
			v: buildIgnore,
		},
		{
			s: "//    +build    exclude   ",
			v: buildExclude,
		},
	}
	for _, tc := range tests {
		t.Run(tc.s, func(t *testing.T) {
			if v := parseBuildConstraint(tc.s); v != tc.v {
				t.Error(v)
			}
		})
	}
}
