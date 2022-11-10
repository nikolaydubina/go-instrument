package processor

import (
	"errors"
	"go/ast"
	"strings"
)

const (
	commandIdentifier = `//go:instrument`

	operationSkipIdentifier = `-skip=`
	operationSkipSeparator  = `|`
)

// Command to change behavior of Processor or Instrumentor
type Command struct {
	Skip []string // list of functions to skip
}

var (
	// NoopCommand does not perform any action
	NoopCommand Command

	errEmptyCommand     = errors.New("empty command")
	errUnknownOperation = errors.New("unknown operation")
	errBadOperation     = errors.New("bad operation")
)

// ParseCommand from string representation
func ParseCommand(s string) (Command, error) {
	if !strings.HasPrefix(s, commandIdentifier) {
		return NoopCommand, nil
	}

	parts := strings.Fields(s[len(commandIdentifier):])
	if len(parts) == 0 {
		return NoopCommand, errEmptyCommand
	}

	var c Command
	for _, q := range parts {
		switch {
		case strings.HasPrefix(q, operationSkipIdentifier):
			vs := strings.Split(q[len(operationSkipIdentifier):], operationSkipSeparator)
			if len(vs) == 0 {
				return NoopCommand, errBadOperation
			}
			c.Skip = vs
		default:
			return NoopCommand, errUnknownOperation
		}
	}

	return c, nil
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
