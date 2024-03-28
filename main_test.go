package main_test

import (
	"os"
	"os/exec"
	"path"
	"testing"
)

func FuzzBadFile(f *testing.F) {
	testbin := path.Join(f.TempDir(), "testbin")
	exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run()

	f.Fuzz(func(t *testing.T, orig string) {
		t.Run("when bad go file, then error", func(t *testing.T) {
			fname := path.Join(t.TempDir(), "fuzz-test-file.go")
			os.WriteFile(fname, []byte(orig), 0644)

			cmd := exec.Command(testbin, "--type", "Color")
			cmd.Env = append(cmd.Environ(), "GOFILE="+fname, "GOPACKAGE=main")
			if err := cmd.Run(); err == nil {
				t.Fatal("must be error")
			}
		})
	})
}

func TestMain(t *testing.T) {
	testbin := path.Join(t.TempDir(), "testbin")
	exec.Command("go", "build", "-cover", "-o", os.Getenv("GOCOVERAGE"), "main.go").Run()

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
