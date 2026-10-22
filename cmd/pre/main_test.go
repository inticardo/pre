package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// twoTargetSource defines two targets that print their frontmatter index, so
// that every test can check that the index is not affected by the selection.
const twoTargetSource = `---
targets:
  - output: "first.txt"
    vars:
      name: "first"
  - output: "second.txt"
    vars:
      name: "second"
---
{{ target_index }} {{ .name }}
`

// writeSource writes a .pre file into a fresh temporary directory and returns
// its path.
func writeSource(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "input.pre")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the source file: %v", err)
	}

	return path
}

// runCommand runs the CLI with the given arguments and an empty stdin.
func runCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	return runWithStdin(t, "", args...)
}

// runWithStdin runs the CLI with the given arguments and stdin content.
func runWithStdin(t *testing.T, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errOut bytes.Buffer
	err = run(args, strings.NewReader(input), &out, &errOut)

	return out.String(), errOut.String(), err
}

// assertContent checks the content of a generated file.
func assertContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %q: %v", path, err)
	}

	if string(got) != want {
		t.Errorf("unexpected content of %q\ngot:  %q\nwant: %q", path, got, want)
	}
}

// assertNotExist checks that a file has not been generated.
func assertNotExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file %q should not exist (stat error: %v)", path, err)
	}
}

func TestRunPrintsUsage(t *testing.T) {
	for _, flag := range []string{"-h", "--help", "-help"} {
		stdout, _, err := runCommand(t, flag)
		if err != nil {
			t.Fatalf("run(%q) returned an error: %v", flag, err)
		}

		if !strings.Contains(stdout, "pre [options] <file.pre>") {
			t.Errorf("run(%q) did not print the usage information: %q", flag, stdout)
		}
	}
}

func TestRunPrintsVersion(t *testing.T) {
	for _, flag := range []string{"-V", "--version", "-version"} {
		stdout, _, err := runCommand(t, flag)
		if err != nil {
			t.Fatalf("run(%q) returned an error: %v", flag, err)
		}

		if want := "pre " + version + "\n"; stdout != want {
			t.Errorf("run(%q) printed %q, want %q", flag, stdout, want)
		}
	}
}

func TestRunValidatesArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no arguments", args: nil},
		{name: "too many arguments", args: []string{"a.pre", "b.pre"}},
		{name: "unknown flag", args: []string{"--unknown"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, _, err := runCommand(t, test.args...)
			if err == nil {
				t.Fatal("expected an error, got none")
			}

			if stdout != "" {
				t.Errorf("unexpected output: %q", stdout)
			}
		})
	}
}

func TestRunProcessesTheFile(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      name: \"world\"\n---\nHello {{ .name }}!\n")

	stdout, stderr, err := runCommand(t, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "" || stderr != "" {
		t.Errorf("unexpected output: stdout %q, stderr %q", stdout, stderr)
	}

	assertContent(t, filepath.Join(filepath.Dir(path), "out.txt"), "Hello world!\n")
}

func TestRunIsVerbose(t *testing.T) {
	path := writeSource(t, twoTargetSource)
	dir := filepath.Dir(path)

	stdout, stderr, err := runCommand(t, "-v", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "" {
		t.Errorf("verbose messages must not be written to stdout: %q", stdout)
	}

	first := strings.Index(stderr, "generating first.txt")
	second := strings.Index(stderr, "generating second.txt")
	if first < 0 || second < 0 {
		t.Fatalf("verbose messages missing from stderr: %q", stderr)
	}
	if first > second {
		t.Errorf("verbose messages are out of order: %q", stderr)
	}

	assertContent(t, filepath.Join(dir, "first.txt"), "0 first\n")
	assertContent(t, filepath.Join(dir, "second.txt"), "1 second\n")
}

func TestRunPrintsToStdout(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      name: \"world\"\n---\nHello {{ .name }}!\n")

	stdout, stderr, err := runCommand(t, "--stdout", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "Hello world!\n" {
		t.Errorf("unexpected stdout: %q", stdout)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr: %q", stderr)
	}

	assertNotExist(t, filepath.Join(filepath.Dir(path), "out.txt"))
}

func TestRunPrintsTheSelectedTarget(t *testing.T) {
	path := writeSource(t, twoTargetSource)
	dir := filepath.Dir(path)

	stdout, stderr, err := runCommand(t, "--stdout", "--target", "second.txt", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "1 second\n" {
		t.Errorf("the target_index must keep its frontmatter position, got %q", stdout)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr: %q", stderr)
	}

	assertNotExist(t, filepath.Join(dir, "first.txt"))
	assertNotExist(t, filepath.Join(dir, "second.txt"))
}

func TestRunSelectsTheTarget(t *testing.T) {
	path := writeSource(t, twoTargetSource)
	dir := filepath.Dir(path)

	stdout, _, err := runCommand(t, "--target", "second.txt", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "" {
		t.Errorf("unexpected output: %q", stdout)
	}

	assertNotExist(t, filepath.Join(dir, "first.txt"))
	assertContent(t, filepath.Join(dir, "second.txt"), "1 second\n")
}

func TestRunKeepsStdoutCleanWhenVerbose(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      name: \"world\"\n---\nHello {{ .name }}!\n")

	stdout, stderr, err := runCommand(t, "-v", "--stdout", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "Hello world!\n" {
		t.Errorf("unexpected stdout: %q", stdout)
	}

	if !strings.Contains(stderr, "generating out.txt") {
		t.Errorf("verbose message missing from stderr: %q", stderr)
	}
}

func TestRunRejectsStdoutWithMultipleTargets(t *testing.T) {
	path := writeSource(t, twoTargetSource)
	dir := filepath.Dir(path)

	stdout, _, err := runCommand(t, "--stdout", path)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "--target") {
		t.Errorf("the error should suggest --target: %v", err)
	}

	if stdout != "" {
		t.Errorf("unexpected output: %q", stdout)
	}

	assertNotExist(t, filepath.Join(dir, "first.txt"))
	assertNotExist(t, filepath.Join(dir, "second.txt"))
}

func TestRunRejectsUnknownTarget(t *testing.T) {
	path := writeSource(t, twoTargetSource)

	_, _, err := runCommand(t, "--target", "missing.txt", path)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "missing.txt") || !strings.Contains(err.Error(), "first.txt, second.txt") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunReadsFromStdin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	stdout, stderr, err := runWithStdin(t, twoTargetSource, "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "" || stderr != "" {
		t.Errorf("unexpected output: stdout %q, stderr %q", stdout, stderr)
	}

	assertContent(t, filepath.Join(dir, "first.txt"), "0 first\n")
	assertContent(t, filepath.Join(dir, "second.txt"), "1 second\n")
}

func TestRunReadsFromStdinToStdout(t *testing.T) {
	source := "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      name: \"world\"\n---\nHello {{ .name }}!\n"

	dir := t.TempDir()
	t.Chdir(dir)

	stdout, stderr, err := runWithStdin(t, source, "--stdout", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "Hello world!\n" {
		t.Errorf("unexpected stdout: %q", stdout)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr: %q", stderr)
	}

	assertNotExist(t, filepath.Join(dir, "out.txt"))
}

func TestRunReadsTheSelectedTargetFromStdin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	stdout, _, err := runWithStdin(t, twoTargetSource, "--stdout", "--target", "second.txt", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "1 second\n" {
		t.Errorf("the target_index must keep its frontmatter position, got %q", stdout)
	}

	assertNotExist(t, filepath.Join(dir, "first.txt"))
	assertNotExist(t, filepath.Join(dir, "second.txt"))
}

func TestRunResolvesStdinIncludesFromTheCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(dir, "partials"), 0o755); err != nil {
		t.Fatalf("cannot create the partials directory: %v", err)
	}

	header := "// Package {{ .package_name }}.\n"
	if err := os.WriteFile(filepath.Join(dir, "partials", "header.jet"), []byte(header), 0o644); err != nil {
		t.Fatalf("cannot write the partial: %v", err)
	}

	source := "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      package_name: \"repositories\"\n---\n{{ include \"./partials/header.jet\" }}Generated for {{ output }}.\n"

	stdout, _, err := runWithStdin(t, source, "--stdout", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "// Package repositories.\nGenerated for out.txt.\n"
	if stdout != want {
		t.Errorf("unexpected stdout\ngot:  %q\nwant: %q", stdout, want)
	}
}

func TestRunRejectsEmptyStdin(t *testing.T) {
	stdout, _, err := runWithStdin(t, "", "-")
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "<stdin>") || !strings.Contains(err.Error(), "missing frontmatter") {
		t.Errorf("unexpected error message: %v", err)
	}

	if stdout != "" {
		t.Errorf("unexpected output: %q", stdout)
	}
}

// failingReader fails the test if the CLI reads from stdin.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("stdin must not be read")
}

func TestRunHelpAndVersionDoNotReadStdin(t *testing.T) {
	for _, flag := range []string{"-h", "--help", "-V", "--version"} {
		var out, errOut bytes.Buffer

		if err := run([]string{flag}, failingReader{}, &out, &errOut); err != nil {
			t.Fatalf("run(%q) returned an error: %v", flag, err)
		}

		if out.Len() == 0 {
			t.Errorf("run(%q) did not print anything", flag)
		}
	}
}
