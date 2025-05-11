package processor

import (
	"go/ast"
	"go/build/constraint"
	"slices"
)

// buildConstraint is selected tags for build constraints
// https://pkg.go.dev/cmd/go#hdr-Build_constraints
type buildConstraint uint

const (
	unknownBuildConstraint buildConstraint = iota
	buildIgnore
	buildExclude
)

func (v buildConstraint) SkipFile() bool {
	switch v {
	case buildIgnore, buildExclude:
		return true
	default:
		return false
	}
}

func parseBuildConstraint(s string) buildConstraint {
	expr, err := constraint.Parse(s)
	if err != nil {
		return unknownBuildConstraint
	}

	switch {
	case expr.Eval(func(tag string) bool { return tag == "ignore" }):
		return buildIgnore
	case expr.Eval(func(tag string) bool { return tag == "exclude" }):
		return buildExclude
	default:
		return unknownBuildConstraint
	}
}

func buildConstraintsFromFile(file ast.File) []buildConstraint {
	var constraints []buildConstraint
	for _, q := range file.Comments {
		if q == nil {
			continue
		}
		for _, c := range q.List {
			if c == nil {
				continue
			}
			if d := parseBuildConstraint(c.Text); d != unknownBuildConstraint {
				constraints = append(constraints, d)
			}
		}
	}
	slices.Sort(constraints)
	return slices.Compact(constraints)
}
