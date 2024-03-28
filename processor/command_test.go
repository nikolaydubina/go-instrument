package processor_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
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

func FuzzBadFile(f *testing.F) {
	f.Fuzz(func(t *testing.T, file string) {
		badfile := path.Join(t.TempDir(), "bad-file")
		os.WriteFile(badfile, []byte(file), 0755)

		t.Run("when bad go file, then error", func(t *testing.T) {
			if _, err := parser.ParseFile(token.NewFileSet(), badfile, nil, parser.ParseComments); err == nil {
				t.Errorf("error expected")
			}
		})
	})
}
