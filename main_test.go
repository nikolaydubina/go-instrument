package main_test

import (
	"math/rand"
	"os"
	"os/exec"
	"path"
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
		f := copyFile(t, "./internal/testdata/basic.go")
		cmd := exec.Command(testbin, "-w", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		assertEqFile(t, "./internal/testdata/instrumented/basic.go.exp", f)
	})

	t.Run("skip file", func(t *testing.T) {
		t.Run("generated file", func(t *testing.T) {
			file := "./internal/testdata/skipped_generated.go"
			f := copyFile(t, file)

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
				f := copyFile(t, file)

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
		f := copyFile(t, "./internal/testdata/instrumented/basic.go.exp")

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
			t.Errorf("files are different at line(%d)\n%s\n%s\n", i, la[i], lb[i])
		}
	}
}

// normalizeLineDirective replaces the filename in //line directives with "FILE" 
// so that comparisons focus on line numbers, not temp filenames
func normalizeLineDirective(line string) string {
	if strings.HasPrefix(line, "//line ") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			return "//line FILE:" + parts[1]
		}
	}
	return line
}

func copyFile(t *testing.T, from string) string {
	f := path.Join(t.TempDir(), time.Now().Format("20060102-150405-")+strconv.Itoa(rand.Int()))
	fbytes, _ := os.ReadFile(from)
	os.WriteFile(f, fbytes, 0644)
	return f
}
