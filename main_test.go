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
	// Create test binary
	testbin := path.Join(t.TempDir(), "go-instrument-testbin")
	cmd := exec.Command("go", "build", "-cover", "-o", testbin, "main.go")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}

	// Create a Go file with a function that panics on a known line
	testGoFile := `package main

import "context"

func TestFunc(ctx context.Context) error {
	// This is line 6
	// This is line 7 
	panic("test panic on line 8") // Line 8
}

func main() {
	TestFunc(context.Background())
}
`

	t.Run("panic line numbers preserved after instrumentation", func(t *testing.T) {
		// Create a temporary directory with proper Go module setup
		tempDir := t.TempDir()
		
		// Copy go.mod and go.sum to temp directory so we can build instrumented code
		goModBytes, _ := os.ReadFile("go.mod")
		goSumBytes, _ := os.ReadFile("go.sum")
		os.WriteFile(path.Join(tempDir, "go.mod"), goModBytes, 0644)
		os.WriteFile(path.Join(tempDir, "go.sum"), goSumBytes, 0644)
		
		// Run go mod download to ensure dependencies are available
		modCmd := exec.Command("go", "mod", "download")
		modCmd.Dir = tempDir
		modCmd.Run() // Ignore errors, as we might not need all deps

		// Create original file
		origFile := path.Join(tempDir, "test_panic.go")
		if err := os.WriteFile(origFile, []byte(testGoFile), 0644); err != nil {
			t.Fatalf("Failed to write original file: %v", err)
		}

		// Build and run original version to get panic line number
		origBinary := path.Join(tempDir, "original_panic")
		buildCmd := exec.Command("go", "build", "-o", origBinary, "test_panic.go")
		buildCmd.Dir = tempDir
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build original binary: %v", err)
		}

		runCmd := exec.Command(origBinary)
		origOutput, _ := runCmd.CombinedOutput() // We expect it to fail with panic
		origLineNum := extractPanicLineNumber(string(origOutput))
		if origLineNum == 0 {
			t.Fatalf("Could not extract line number from original panic output: %s", string(origOutput))
		}

		// Create instrumented file  
		instrumentedFile := path.Join(tempDir, "test_panic_instrumented.go")
		if err := os.WriteFile(instrumentedFile, []byte(testGoFile), 0644); err != nil {
			t.Fatalf("Failed to write instrumented file: %v", err)
		}

		// Instrument the file
		instrumentCmd := exec.Command(testbin, "-w", "-filename", instrumentedFile)
		instrumentCmd.Env = append(instrumentCmd.Environ(), "GOCOVERDIR=./coverage")
		if err := instrumentCmd.Run(); err != nil {
			t.Fatalf("Failed to instrument file: %v", err)
		}

		// Run go mod tidy to ensure dependencies are available for instrumented file
		modCmd = exec.Command("go", "mod", "tidy")
		modCmd.Dir = tempDir
		if err := modCmd.Run(); err != nil {
			t.Fatalf("Failed to run go mod tidy: %v", err)
		}

		// Build and run instrumented version to get panic line number
		instrumentedBinary := path.Join(tempDir, "instrumented_panic")
		buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "test_panic_instrumented.go")
		buildCmd.Dir = tempDir
		if err := buildCmd.Run(); err != nil {
			// Get detailed build error
			output, _ := buildCmd.CombinedOutput()
			t.Fatalf("Failed to build instrumented binary: %v\nOutput: %s", err, string(output))
		}

		runCmd = exec.Command(instrumentedBinary)
		instrumentedOutput, _ := runCmd.CombinedOutput() // We expect it to fail with panic
		instrumentedLineNum := extractPanicLineNumber(string(instrumentedOutput))
		if instrumentedLineNum == 0 {
			t.Fatalf("Could not extract line number from instrumented panic output: %s", string(instrumentedOutput))
		}

		// Compare line numbers - they should be the same
		if origLineNum != instrumentedLineNum {
			t.Errorf("Panic line numbers don't match: original=%d, instrumented=%d", origLineNum, instrumentedLineNum)
			t.Logf("Original output: %s", string(origOutput))
			t.Logf("Instrumented output: %s", string(instrumentedOutput))
		}
	})
}

// extractPanicLineNumber extracts the line number from a panic stack trace
func extractPanicLineNumber(output string) int {
	// Look for pattern like "test_panic.go:8" in the stack trace
	re := regexp.MustCompile(`test_panic.*\.go:(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		if lineNum, err := strconv.Atoi(matches[1]); err == nil {
			return lineNum
		}
	}
	return 0
}
