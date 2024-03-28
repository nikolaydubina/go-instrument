package processor_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"testing"

	"github.com/nikolaydubina/go-instrument/processor"
)

func TestParseCommand_Error(t *testing.T) {
	tests := []string{
		"//instrument:",
		"//instrument: asdf",
		"//instrument:asdf",
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := processor.ParseCommand(tc)
			if err == nil {
				t.Error("error expected")
			}
		})
	}
}

func TestParseCommandFromFile_Error(t *testing.T) {
	tests := []string{
		"testdata/bad_command_unknown.go",
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), tc, nil, parser.ParseComments)
			if err != nil || f == nil {
				t.Errorf("can not parse input file: %s", err)
			}

			c, err := processor.CommandsFromFile(*f)
			if len(c) != 0 {
				t.Errorf("no commands expected")
			}
			if err == nil {
				t.Error("error expected")
			}
		})
	}
}
