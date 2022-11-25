package processor

import (
	"errors"
	"go/ast"
	"strings"
)

const (
	commandPrefix            = `//instrument:`
	commandIncludeIdentifier = `//instrument:include`
	commandExcludeIdentifier = `//instrument:exclude`
)

// Command to change behavior of Processor or Instrumentor
type Command struct {
	acceptFunctions map[string]bool
}

// ParseCommand from string representation
func ParseCommand(s string) (Command, error) {
	command := Command{acceptFunctions: map[string]bool{}}

	if !strings.HasPrefix(s, commandPrefix) {
		return command, nil
	}

	switch {
	case strings.HasPrefix(s, commandIncludeIdentifier):
		for _, v := range strings.Split(strings.TrimSpace(s[len(commandIncludeIdentifier):]), "|") {
			command.acceptFunctions[v] = true
		}
	case strings.HasPrefix(s, commandExcludeIdentifier):
		for _, v := range strings.Split(strings.TrimSpace(s[len(commandExcludeIdentifier):]), "|") {
			command.acceptFunctions[v] = false
		}
	default:
		return command, errors.New("unkown command")
	}
	return command, nil
}

// CommandsFromFile that has been parsed by `go/parse` with comments
func CommandsFromFile(file ast.File) ([]Command, error) {
	var commands []Command

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
			c, err := ParseCommand(c.Text)
			if err != nil {
				return nil, err
			}
			commands = append(commands, c)
		}
	}

	return commands, nil
}

// NewMapFunctionSelectorFromCommands performs join of function names stored in maps of commands.
func NewMapFunctionSelectorFromCommands(defaultSelect bool, commands []Command) MapFunctionSelector {
	f := MapFunctionSelector{
		AcceptFunctions: make(map[string]bool, len(commands)),
		Default:         defaultSelect,
	}

	for _, q := range commands {
		for fname, accept := range q.acceptFunctions {
			if v, ok := f.AcceptFunctions[fname]; ok {
				f.AcceptFunctions[fname] = v && accept
			} else {
				f.AcceptFunctions[fname] = accept
			}
		}
	}

	return f
}
