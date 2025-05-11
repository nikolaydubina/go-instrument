package processor

import (
	"go/ast"
	"go/build/constraint"
	"slices"
)

// BuildConstraint is selected tags for build constraints
// https://pkg.go.dev/cmd/go#hdr-Build_constraints
type BuildConstraint uint

const (
	UnknownBuildConstraint BuildConstraint = iota
	BuildIgnore
	BuildExclude
)

func (v BuildConstraint) SkipFile() bool {
	switch v {
	case BuildIgnore, BuildExclude:
		return true
	default:
		return false
	}
}

func ParseBuildConstraint(s string) BuildConstraint {
	expr, err := constraint.Parse(s)
	if err != nil {
		return UnknownBuildConstraint
	}

	switch {
	case expr.Eval(func(tag string) bool { return tag == "ignore" }):
		return BuildIgnore
	case expr.Eval(func(tag string) bool { return tag == "exclude" }):
		return BuildExclude
	default:
		return UnknownBuildConstraint
	}
}

func BuildConstraintsFromFile(file ast.File) []BuildConstraint {
	var constraints []BuildConstraint
	for _, q := range file.Comments {
		if q == nil {
			continue
		}
		for _, c := range q.List {
			if c == nil {
				continue
			}
			if d := ParseBuildConstraint(c.Text); d != UnknownBuildConstraint {
				constraints = append(constraints, d)
			}
		}
	}
	slices.Sort(constraints)
	return slices.Compact(constraints)
}
