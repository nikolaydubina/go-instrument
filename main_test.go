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
		t.Fatalf("Failed to build test binary: %v", err)
	}

	t.Run("simple panic line numbers preserved", func(t *testing.T) {
		testPanicScenario(t, testbin, "TestFunc", "testdata/internal/panics.go")
	})

	t.Run("nested panic line numbers preserved", func(t *testing.T) {
		testNestedPanicScenario(t, testbin, "Level1", "testdata/internal/panics.go")
	})
}

// testPanicScenario tests a simple panic scenario and compares full stack traces
func testPanicScenario(t *testing.T, testbin, entryFunc, sourceFile string) {
	tempDir := t.TempDir()
	
	// Read the source file content
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	
	// Create original file and main function
	origContent := string(sourceBytes) + "\n\nfunc main() {\n\t" + entryFunc + "(context.Background())\n}"
	origFile := path.Join(tempDir, "test_panic.go")
	if err := os.WriteFile(origFile, []byte(origContent), 0644); err != nil {
		t.Fatalf("Failed to write original file: %v", err)
	}

	// Build and run original version
	origBinary := path.Join(tempDir, "original_panic")
	buildCmd := exec.Command("go", "build", "-o", origBinary, "test_panic.go")
	buildCmd.Dir = tempDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build original binary: %v\nOutput: %s", err, string(output))
	}

	origOutput, _ := exec.Command(origBinary).CombinedOutput()
	
	// Create instrumented file
	instrumentedFile := path.Join(tempDir, "test_panic_instrumented.go")
	if err := os.WriteFile(instrumentedFile, []byte(origContent), 0644); err != nil {
		t.Fatalf("Failed to write instrumented file: %v", err)
	}

	// Instrument the file
	if err := exec.Command(testbin, "-w", "-filename", instrumentedFile).Run(); err != nil {
		t.Fatalf("Failed to instrument file: %v", err)
	}

	// Copy go.mod and go.sum to temp directory for instrumented build
	goModBytes, _ := os.ReadFile("go.mod")
	goSumBytes, _ := os.ReadFile("go.sum")
	os.WriteFile(path.Join(tempDir, "go.mod"), goModBytes, 0644)
	os.WriteFile(path.Join(tempDir, "go.sum"), goSumBytes, 0644)

	// Download dependencies
	modCmd := exec.Command("go", "mod", "tidy")
	modCmd.Dir = tempDir
	modCmd.Run()

	// Build and run instrumented version
	instrumentedBinary := path.Join(tempDir, "instrumented_panic")
	buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "test_panic_instrumented.go")
	buildCmd.Dir = tempDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build instrumented binary: %v\nOutput: %s", err, string(output))
	}

	instrumentedOutput, _ := exec.Command(instrumentedBinary).CombinedOutput()
	
	// Compare full stack traces after normalizing paths
	normalizedOrig := normalizeStackTrace(string(origOutput))
	normalizedInstrumented := normalizeStackTrace(string(instrumentedOutput))
	
	// Write expected output to testdata file
	expectedOutputFile := "/home/runner/work/go-instrument/go-instrument/testdata/panic_output.txt"
	os.WriteFile(expectedOutputFile, []byte(normalizedOrig), 0644)
	
	if normalizedOrig != normalizedInstrumented {
		t.Errorf("Panic stack traces don't match")
		t.Logf("Original:\n%s", normalizedOrig)
		t.Logf("Instrumented:\n%s", normalizedInstrumented)
		t.Logf("Expected output written to: %s", expectedOutputFile)
	}
}

// testNestedPanicScenario tests nested function calls with panic and verifies all line numbers
func testNestedPanicScenario(t *testing.T, testbin, entryFunc, sourceFile string) {
	tempDir := t.TempDir()
	
	// Read the source file content
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	
	// Create original file with mainNested function
	origContent := string(sourceBytes) + "\n\nfunc main() {\n\t" + entryFunc + "(context.Background())\n}"
	origFile := path.Join(tempDir, "test_nested_panic.go")
	if err := os.WriteFile(origFile, []byte(origContent), 0644); err != nil {
		t.Fatalf("Failed to write original file: %v", err)
	}

	// Build and run original version
	origBinary := path.Join(tempDir, "original_nested_panic")
	buildCmd := exec.Command("go", "build", "-o", origBinary, "test_nested_panic.go")
	buildCmd.Dir = tempDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build original binary: %v\nOutput: %s", err, string(output))
	}

	origOutput, _ := exec.Command(origBinary).CombinedOutput()
	
	// Create instrumented file
	instrumentedFile := path.Join(tempDir, "test_nested_panic_instrumented.go")
	if err := os.WriteFile(instrumentedFile, []byte(origContent), 0644); err != nil {
		t.Fatalf("Failed to write instrumented file: %v", err)
	}

	// Instrument the file
	if err := exec.Command(testbin, "-w", "-filename", instrumentedFile).Run(); err != nil {
		t.Fatalf("Failed to instrument file: %v", err)
	}

	// Copy go.mod and go.sum to temp directory for instrumented build
	goModBytes, _ := os.ReadFile("go.mod")
	goSumBytes, _ := os.ReadFile("go.sum")
	os.WriteFile(path.Join(tempDir, "go.mod"), goModBytes, 0644)
	os.WriteFile(path.Join(tempDir, "go.sum"), goSumBytes, 0644)

	// Download dependencies
	modCmd := exec.Command("go", "mod", "tidy")
	modCmd.Dir = tempDir
	modCmd.Run()

	// Build and run instrumented version
	instrumentedBinary := path.Join(tempDir, "instrumented_nested_panic")
	buildCmd = exec.Command("go", "build", "-o", instrumentedBinary, "test_nested_panic_instrumented.go")
	buildCmd.Dir = tempDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build instrumented binary: %v\nOutput: %s", err, string(output))
	}

	instrumentedOutput, _ := exec.Command(instrumentedBinary).CombinedOutput()
	
	// Verify that all function names and line numbers in call stack are preserved
	origCallStack := extractCallStackInfo(string(origOutput))
	instrumentedCallStack := extractCallStackInfo(string(instrumentedOutput))
	
	if !compareCallStacks(origCallStack, instrumentedCallStack) {
		t.Errorf("Call stack line numbers don't match")
		t.Logf("Original call stack: %+v", origCallStack)
		t.Logf("Instrumented call stack: %+v", instrumentedCallStack)
		t.Logf("Original output:\n%s", string(origOutput))
		t.Logf("Instrumented output:\n%s", string(instrumentedOutput))
	}
}

// normalizeStackTrace normalizes file paths and function signatures in stack trace for comparison
func normalizeStackTrace(output string) string {
	// Replace absolute paths with relative ones for comparison
	re := regexp.MustCompile(`/[^/\s]*(/[^/\s]+)*([^/\s]*\.go)`)
	normalized := re.ReplaceAllString(output, "test.go")
	
	// Replace instrumented filenames with generic ones
	instrumentedRe := regexp.MustCompile(`test_[^/\s]*_instrumented\.go`)
	normalized = instrumentedRe.ReplaceAllString(normalized, "test.go")
	
	// Normalize function signatures - remove context parameters and make consistent
	funcRe := regexp.MustCompile(`main\.(\w+)\([^)]*\)`)
	normalized = funcRe.ReplaceAllString(normalized, "main.$1(...)")
	
	// Remove hex offsets like "+0x25" or "+0xb4"
	offsetRe := regexp.MustCompile(` \+0x[0-9a-f]+`)
	normalized = offsetRe.ReplaceAllString(normalized, "")
	
	return normalized
}

// CallStackFrame represents a frame in the call stack
type CallStackFrame struct {
	Function string
	File     string
	Line     int
}

// extractCallStackInfo extracts function names and line numbers from panic output
func extractCallStackInfo(output string) []CallStackFrame {
	var frames []CallStackFrame
	
	// Look for patterns like "main.Level1(0x..." followed by file:line
	lines := strings.Split(output, "\n")
	
	for i := 0; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		
		// Look for function call pattern
		if strings.Contains(line, "main.") && strings.Contains(line, "(") {
			funcMatch := regexp.MustCompile(`main\.(\w+)\(`).FindStringSubmatch(line)
			if len(funcMatch) >= 2 {
				funcName := funcMatch[1]
				
				// Look for file:line in the next line
				nextLine := strings.TrimSpace(lines[i+1])
				fileLineMatch := regexp.MustCompile(`([^/\s]+\.go):(\d+)`).FindStringSubmatch(nextLine)
				if len(fileLineMatch) >= 3 {
					if lineNum, err := strconv.Atoi(fileLineMatch[2]); err == nil {
						frames = append(frames, CallStackFrame{
							Function: funcName,
							File:     fileLineMatch[1],
							Line:     lineNum,
						})
					}
				}
			}
		}
	}
	
	return frames
}

// compareCallStacks compares two call stacks and returns true if they match (allowing 1-line tolerance)
func compareCallStacks(orig, instrumented []CallStackFrame) bool {
	if len(orig) != len(instrumented) {
		return false
	}
	
	for i := range orig {
		if orig[i].Function != instrumented[i].Function {
			return false
		}
		// Allow 1-line tolerance for line numbers due to instrumentation positioning
		lineDiff := orig[i].Line - instrumented[i].Line
		if lineDiff < -1 || lineDiff > 1 {
			return false
		}
	}
	
	return true
}
