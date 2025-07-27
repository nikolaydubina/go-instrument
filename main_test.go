package main_test

import (
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func FuzzBadFile(f *testing.F) {
	testbin := path.Join(f.TempDir(), "go-instrument-testbin")
	exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run()

	f.Fuzz(func(t *testing.T, orig string) {
		t.Run("when bad go file, then error", func(t *testing.T) {
			fname := path.Join(t.TempDir(), "fuzz-test-file.go")
			os.WriteFile(fname, []byte(orig), 0644)

			cmd := exec.Command(testbin, "-w", "-filename", fname)
			cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
			if err := cmd.Run(); err == nil {
				t.Fatal(err)
			}
		})
	})
}

func TestApp(t *testing.T) {
	testbin := path.Join(t.TempDir(), "go-instrument-testbin")
	exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run()

	t.Run("when basic, then ok", func(t *testing.T) {
		f := randFileName(t)
		if err := copy("./internal/testdata/basic.go", f); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(testbin, "-w", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		assertEqFile(t, "./internal/testdata/instrumented/basic.go.exp", f)
	})

	t.Run("when not preserving line numbers, then not no line number directives", func(t *testing.T) {
		f := randFileName(t)
		if err := copy("./internal/testdata/basic.go", f); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(testbin, "-w", "-preserve-line-numbers=false", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		assertEqFile(t, "./internal/testdata/instrumented/basic_no_line.go.exp", f)
	})

	t.Run("skip file", func(t *testing.T) {
		t.Run("generated file", func(t *testing.T) {
			file := "./internal/testdata/skipped_generated.go"
			f := randFileName(t)
			if err := copy(file, f); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(testbin, "-w", "-skip-generated=true", "-filename", file)
			cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
			if _, err := cmd.CombinedOutput(); err != nil {
				t.Error(err)
			}

			assertEqFile(t, file, f)
		})

		skipFiles := []string{
			"./internal/testdata/skipped_buildexclude.go",
			"./internal/testdata/skipped_buildignore.go",
			"./internal/testdata/skipped_gobuildignore.go",
		}
		for _, file := range skipFiles {
			t.Run(file, func(t *testing.T) {
				f := randFileName(t)
				if err := copy(file, f); err != nil {
					t.Fatal(err)
				}

				cmd := exec.Command(testbin, "-w", "-filename", file)
				cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
				if _, err := cmd.CombinedOutput(); err != nil {
					t.Error(err)
				}

				assertEqFile(t, file, f)
			})
		}
	})

	t.Run("bad file", func(t *testing.T) {
		t.Run("cannot open file", func(t *testing.T) {
			cmd := exec.Command(testbin, "-filename", "asdf")
			cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
			if err := cmd.Run(); err == nil {
				t.Error("expected exit code 1")
			}
		})

		t.Run("non go file", func(t *testing.T) {
			cmd := exec.Command(testbin, "-filename", "README.md")
			cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
			if err := cmd.Run(); err == nil {
				t.Error("expected exit code 1")
			}
		})

		t.Run("non filename", func(t *testing.T) {
			cmd := exec.Command(testbin)
			cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
			if err := cmd.Run(); err == nil {
				t.Error("expected exit code 1")
			}
		})
	})

	t.Run("when already instrumented, then do not instrument", func(t *testing.T) {
		f := randFileName(t)
		if err := copy("./internal/testdata/instrumented/basic.go.exp", f); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(testbin, "-w", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}

		assertEqFile(t, "./internal/testdata/instrumented/basic.go.exp", f)
	})
}

func assertEqFile(t *testing.T, a, b string) {
	fa, _ := os.ReadFile(a)
	fb, _ := os.ReadFile(b)

	la := strings.Split(string(fa), "\n")
	lb := strings.Split(string(fb), "\n")

	for i := 0; i < len(la) && i < len(lb); i++ {
		// Normalize //line directives to compare only the line numbers, not filenames
		lineA := normalizeLineDirective(la[i])
		lineB := normalizeLineDirective(lb[i])

		if lineA != lineB {
			t.Errorf("files are different at line(%d)\n%s\n===\n%s\n", i, la[i], lb[i])
		}
	}
}

// normalizeLineDirective replaces the filename in line directives with "FILE"
// so that comparisons focus on line numbers, not temp filenames
func normalizeLineDirective(line string) string {
	// Handle //line directives
	if strings.HasPrefix(line, "//line ") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			return "//line FILE:" + parts[1]
		}
	}
	// Handle /*line*/ directives
	if strings.Contains(line, "/*line ") {
		// Find the pattern /*line filename:line:col*/
		re := regexp.MustCompile(`/\*line\s+[^:]+:(\d+):(\d+)\*/`)
		return re.ReplaceAllString(line, "/*line FILE:$1:$2*/")
	}
	return line
}

func TestPanicLineNumbers(t *testing.T) {
	testbin := path.Join(t.TempDir(), "go-instrument-testbin")
	if err := exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run(); err != nil {
		t.Fatal(err)
	}

	tests := []string{
		"testdata/internal/panic1/main.go",
		"testdata/internal/panic2/main.go",
		"testdata/internal/panic3/main.go",
	}
	for _, tc := range tests {
		dir := t.TempDir()

		if err := copy(tc, path.Join(dir, "main.go")); err != nil {
			t.Fatal(err)
		}

		originalBinary := path.Join(dir, "original_panic")
		buildCmd := exec.Command("go", "build", "-o", originalBinary, "main.go")
		buildCmd.Dir = dir
		if err := buildCmd.Run(); err != nil {
			t.Fatal(err)
		}

		originalOutput, _ := exec.Command(originalBinary).CombinedOutput()

		if err := copy(tc, path.Join(dir, "main_instrumented.go")); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command(testbin, "-w", "--preserve-line-numbers", "-filename", path.Join(dir, "main_instrumented.go")).Run(); err != nil {
			t.Fatal(err)
		}

		modCmd := exec.Command("go", "mod", "init", "test_panic_instrumented")
		modCmd.Dir = dir
		modCmd.Run()

		modTidyCmd := exec.Command("go", "mod", "tidy")
		modTidyCmd.Dir = dir
		modTidyCmd.Run()

		instrumentedBinary := path.Join(dir, "instrumented_panic")
		buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "main_instrumented.go")
		buildCmd.Dir = dir
		if out, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}

		instrumentedOutput, _ := exec.Command(instrumentedBinary).CombinedOutput()

		originalLines := extractLineNumbers(string(originalOutput))
		instrumentedLines := extractLineNumbers(string(instrumentedOutput))

		if !slices.Equal(originalLines, instrumentedLines) {
			t.Error(originalLines, instrumentedLines, string(originalOutput), string(instrumentedOutput))
		}
	}
}

func extractLineNumbers(output string) (lines []int) {
	re := regexp.MustCompile(`\.go:(\d+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			if lineNum, err := strconv.Atoi(match[1]); err == nil {
				lines = append(lines, lineNum)
			}
		}
	}
	return lines
}

func randFileName(t *testing.T) string {
	return path.Join(t.TempDir(), time.Now().Format("20060102-150405-")+strconv.Itoa(rand.Int())+".go")
}

func copy(from, to string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, 0644)
}
