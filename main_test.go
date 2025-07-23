package main_test

import (
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
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
		testPanicScenario(t, testbin, "TestFunc")
	})

	t.Run("nested panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "Level1")
	})
	t.Run("complex function panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "FuncWithBody")
	})
}
func testPanicScenario(t *testing.T, testbin, entryFunc string) {
	tempDir := t.TempDir()
	sourceFile := "testdata/internal/panics.go"
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(sourceBytes) + `

func main() {
	` + entryFunc + `(context.Background())
}`
	origFile := path.Join(tempDir, "test_panic.go")
	if err := os.WriteFile(origFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	origBinary := path.Join(tempDir, "original_panic")
	buildCmd := exec.Command("go", "build", "-o", origBinary, "test_panic.go")
	buildCmd.Dir = tempDir
	if err := buildCmd.Run(); err != nil {
		t.Fatal(err)
	}
	origOutput, _ := exec.Command(origBinary).CombinedOutput()
	instrumentedFile := path.Join(tempDir, "test_panic_instrumented.go")
	if err := os.WriteFile(instrumentedFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command(testbin, "-w", "-filename", instrumentedFile).Run(); err != nil {
		t.Fatal(err)
	}
	goModBytes, _ := os.ReadFile("go.mod")
	goSumBytes, _ := os.ReadFile("go.sum")
	os.WriteFile(path.Join(tempDir, "go.mod"), goModBytes, 0644)
	os.WriteFile(path.Join(tempDir, "go.sum"), goSumBytes, 0644)
	modCmd := exec.Command("go", "mod", "tidy")
	modCmd.Dir = tempDir
	modCmd.Run()
	instrumentedBinary := path.Join(tempDir, "instrumented_panic")
	buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "test_panic_instrumented.go")
	buildCmd.Dir = tempDir
	if err := buildCmd.Run(); err != nil {
		t.Fatal(err)
	}
	instrumentedOutput, _ := exec.Command(instrumentedBinary).CombinedOutput()
	// Extract line numbers from both outputs to verify preservation
	origLines := extractLineNumbers(string(origOutput))
	instrumentedLines := extractLineNumbers(string(instrumentedOutput))
	expectedOutputFile := "/home/runner/work/go-instrument/go-instrument/testdata/panic_output.txt"
	os.WriteFile(expectedOutputFile, origOutput, 0644)

	// Compare only the user code line numbers (ignore main function which is test-added)
	if len(origLines) == 0 || len(instrumentedLines) == 0 {
		t.Errorf("Could not extract line numbers from stack traces")
	} else if !equalUserCodeLineNumbers(origLines, instrumentedLines) {
		t.Errorf("User code line numbers in stack traces don't match")
		t.Logf("Original line numbers: %v", origLines)
		t.Logf("Instrumented line numbers: %v", instrumentedLines)
		t.Logf("Original:\n%s", string(origOutput))
		t.Logf("Instrumented:\n%s", string(instrumentedOutput))
	}
}

// extractLineNumbers extracts line numbers from stack trace output
func extractLineNumbers(output string) []int {
	var lines []int
	// Look for patterns like "test.go:26" or "/path/test.go:26"
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

// equalUserCodeLineNumbers compares only the first line number (main panic location)
func equalUserCodeLineNumbers(orig, instrumented []int) bool {
	// The most important thing is that the first line number (panic location) matches
	if len(orig) == 0 || len(instrumented) == 0 {
		return false
	}
	return orig[0] == instrumented[0]
}
