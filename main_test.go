package main_test

import (
	"os"
	"os/exec"
	"path"
	"testing"
)

var (
	testbin string
)

func init() {
	testbin := path.Join(os.TempDir(), "go-instrument-testbin")
	exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run()
}

func FuzzBadFile(f *testing.F) {
	f.Fuzz(func(t *testing.T, orig string) {
		t.Run("when bad go file, then error", func(t *testing.T) {
			fname := path.Join(t.TempDir(), "fuzz-test-file.go")
			os.WriteFile(fname, []byte(orig), 0644)

			cmd := exec.Command(testbin, "-app", "app", "-w", "-filename", fname)
			if err := cmd.Run(); err == nil {
				t.Fatal(err)
			}
		})
	})
}

func TestMain(t *testing.T) {
	t.Run("when basic, then ok", func(t *testing.T) {
		cmd := exec.Command(testbin, "-app", "app", "-w", "-filename", "./internal/testdata/basic.go")
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assertEqFile(t, "./internal/testdata/basic.go", "internal/testdata/instrumented/basic.go.exp")
	})

	t.Run("when include only, then ok", func(t *testing.T) {
		cmd := exec.Command(testbin, "-app", "app", "-w", "-all=false", "-filename", "./internal/testdata/basic_include_only.go")
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assertEqFile(t, "./internal/testdata/basic_include_only.go", "internal/testdata/instrumented/basic_include_only.go.exp")
	})
}

func assertEqFile(t *testing.T, a, b string) {
	fa, _ := os.ReadFile(a)
	fb, _ := os.ReadFile(b)
	if string(fa) != string(fb) {
		t.Error("files are different")
	}
}
