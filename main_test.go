package main_test

import (
	"os"
	"os/exec"
	"path"
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

			cmd := exec.Command(testbin, "-app", "app", "-w", "-filename", fname)
			if err := cmd.Run(); err == nil {
				t.Fatal(err)
			}
		})
	})
}

func TestMain(t *testing.T) {
	testbin := path.Join(t.TempDir(), "go-instrument-testbin")
	exec.Command("go", "build", "-cover", "-o", testbin, "main.go").Run()

	t.Run("when basic, then ok", func(t *testing.T) {
		f := copyFile(t, "./internal/testdata/basic.go")
		cmd := exec.Command(testbin, "-app", "app", "-w", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Errorf(err.Error())
		}
		assertEqFile(t, "./internal/testdata/instrumented/basic.go.exp", f)
	})

	t.Run("when include only, then ok", func(t *testing.T) {
		f := copyFile(t, "./internal/testdata/basic_include_only.go")
		cmd := exec.Command(testbin, "-app", "app", "-w", "-all=false", "-filename", f)
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err != nil {
			t.Errorf(err.Error())
		}
		assertEqFile(t, "./internal/testdata/instrumented/basic_include_only.go.exp", f)
	})

	t.Run("when generated file, then err", func(t *testing.T) {
		cmd := exec.Command(testbin, "-app", "app", "-w", "-skip-generated=true", "-filename", "./internal/testdata/skipped_generated.go")
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("expected exit code 1")
		}
		if !strings.Contains(string(out), "skipping generated file") {
			t.Errorf("expected skipping generated file")
		}
	})

	t.Run("when cannot open file, then err", func(t *testing.T) {
		cmd := exec.Command(testbin, "-filename", "asdf")
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err == nil {
			t.Errorf("expected exit code 1")
		}
	})

	t.Run("when non go file, then err", func(t *testing.T) {
		cmd := exec.Command(testbin, "-filename", "README.md")
		cmd.Env = append(cmd.Environ(), "GOCOVERDIR=./coverage")
		if err := cmd.Run(); err == nil {
			t.Errorf("expected exit code 1")
		}
	})
}

func assertEqFile(t *testing.T, a, b string) {
	fa, _ := os.ReadFile(a)
	fb, _ := os.ReadFile(b)
	if string(fa) != string(fb) {
		t.Errorf("files are different: %s != %s", string(fa), string(fb))
	}
}

func copyFile(t *testing.T, from string) string {
	f := path.Join(t.TempDir(), time.Now().Format("20060102-150405-"))
	fbytes, _ := os.ReadFile(from)
	os.WriteFile(f, fbytes, 0644)
	return f
}
