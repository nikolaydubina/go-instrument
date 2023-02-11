package processor

import (
	"go/ast"
	"strings"
)

// GoBuildDirective is standard Go compiler directive.
type GoBuildDirective uint

const (
	UnknownDirective GoBuildDirective = iota
	GoBuildIgnore
	GoBuildExclude
	BuildIgnore
	BuildExclude
)

func (v GoBuildDirective) SkipFile() bool {
	switch v {
	case GoBuildIgnore, GoBuildExclude, BuildIgnore, BuildExclude:
		return true
	default:
		return false
	}
}

func IsGoBuildIgnore(s string) bool {
	fs := strings.Fields(s)
	if len(fs) != 2 {
		return false
	}
	return fs[0] == "//go:build" && fs[1] == "ignore"
}

func IsGoBuildExclude(s string) bool {
	fs := strings.Fields(s)
	if len(fs) != 2 {
		return false
	}
	return fs[0] == "//go:build" && fs[1] == "exclude"
}

func IsBuildIgnore(s string) bool {
	fs := strings.Fields(s)
	if len(fs) != 3 {
		return false
	}
	return fs[0] == "//" && fs[1] == "+build" && fs[2] == "ignore"
}

func IsBuildExclude(s string) bool {
	fs := strings.Fields(s)
	if len(fs) != 3 {
		return false
	}
	return fs[0] == "//" && fs[1] == "+build" && fs[2] == "exclude"
}

func ParseGoBuildDirective(s string) GoBuildDirective {
	switch {
	case IsGoBuildIgnore(s):
		return GoBuildIgnore
	case IsGoBuildExclude(s):
		return GoBuildExclude
	case IsBuildIgnore(s):
		return BuildIgnore
	case IsBuildExclude(s):
		return BuildExclude
	default:
		return UnknownDirective
	}
}

func GoBuildDirectivesFromFile(file ast.File) (directives []GoBuildDirective) {
	for _, q := range file.Comments {
		if q == nil {
			continue
		}
		for _, c := range q.List {
			if c == nil {
				continue
			}
			// can not use ast.CommentGroup since its Text() skips directive comments
			// that do not have empty space after `//` (eg, //go:instrument or //go:generate)
			if d := ParseGoBuildDirective(c.Text); d != UnknownDirective {
				directives = append(directives, d)
			}
		}
	}
	return directives
}
