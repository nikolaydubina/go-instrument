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

func TestPanicLineNumbers(t *testing.T) {
	testbin := path.Join(t.TempDir(), "go-instrument-testbin")
	if err := exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run(); err != nil {
		t.Fatal(err)
	}

	t.Run("simple panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "testdata/internal/panic1")
	})

	t.Run("nested panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "testdata/internal/panic2")
	})

	t.Run("complex function panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "testdata/internal/panic3")
	})
}

func testPanicScenario(t *testing.T, testbin, testDir string) {
	tempDir := t.TempDir()

	// Copy the static test file
	sourceFile := path.Join(testDir, "main.go")
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}

	// Build original (non-instrumented) binary
	originalFile := path.Join(tempDir, "main.go")
	if err := os.WriteFile(originalFile, sourceBytes, 0644); err != nil {
		t.Fatal(err)
	}

	originalBinary := path.Join(tempDir, "original_panic")
	buildCmd := exec.Command("go", "build", "-o", originalBinary, "main.go")
	buildCmd.Dir = tempDir
	if err := buildCmd.Run(); err != nil {
		t.Fatal(err)
	}

	origOutput, _ := exec.Command(originalBinary).CombinedOutput()

	// Build instrumented binary
	instrumentedFile := path.Join(tempDir, "main_instrumented.go")
	if err := os.WriteFile(instrumentedFile, sourceBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(testbin, "-w", "-filename", instrumentedFile).Run(); err != nil {
		t.Fatal(err)
	}

	modCmd := exec.Command("go", "mod", "init", "test_panic_instrumented")
	modCmd.Dir = tempDir
	modCmd.Run()

	modTidyCmd := exec.Command("go", "mod", "tidy")
	modTidyCmd.Dir = tempDir
	modTidyCmd.Run()

	instrumentedBinary := path.Join(tempDir, "instrumented_panic")
	buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "main_instrumented.go")
	buildCmd.Dir = tempDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}

	instrumentedOutput, _ := exec.Command(instrumentedBinary).CombinedOutput()

	originalLines := extractLineNumbers(string(origOutput))
	instrumentedLines := extractLineNumbers(string(instrumentedOutput))

	if !slices.Equal(originalLines, instrumentedLines) {
		t.Error(originalLines, instrumentedLines, string(origOutput), string(instrumentedOutput))
	}
}

func extractLineNumbers(output string) []int {
	var lines []int
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
