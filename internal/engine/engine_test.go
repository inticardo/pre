package engine_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inticardo/pre/internal/engine"
)

// copyCase copies a testdata case into a temporary directory so that the
// generated files never pollute the repository.
func copyCase(t *testing.T, name string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatalf("cannot copy testdata case %q: %v", name, err)
	}

	return dir
}

// assertGolden compares every file of the expected directory with the file
// generated in the working directory.
func assertGolden(t *testing.T, dir, name string) {
	t.Helper()

	expectedDir := filepath.Join("testdata", name, "expected")
	err := filepath.WalkDir(expectedDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(expectedDir, path)
		if err != nil {
			return err
		}

		want, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read expected file %q: %w", rel, err)
		}

		got, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return fmt.Errorf("cannot read generated file %q: %w", rel, err)
		}

		if string(got) != string(want) {
			t.Errorf("generated file %q does not match the expected one\n--- got ---\n%s\n--- want ---\n%s", rel, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cannot compare expected directory: %v", err)
	}
}

func TestProcessGeneratesEveryTarget(t *testing.T) {
	dir := copyCase(t, "basic")

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertGolden(t, dir, "basic")
}

func TestProcessResolvesIncludesRelativeToTheSourceFile(t *testing.T) {
	dir := copyCase(t, "includes")

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertGolden(t, dir, "includes")
}

func TestProcessExposesTheCustomHelpers(t *testing.T) {
	dir := copyCase(t, "helpers")

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertGolden(t, dir, "helpers")
}

func TestProcessInjectsTheTargetIndex(t *testing.T) {
	dir := copyCase(t, "index")

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertGolden(t, dir, "index")
}

func TestProcessCreatesMissingOutputDirectories(t *testing.T) {
	dir := copyCase(t, "subdirectory")

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertGolden(t, dir, "subdirectory")
}

func TestProcessWritesNextToTheSourceFile(t *testing.T) {
	dir := copyCase(t, "basic")

	other := t.TempDir()
	t.Chdir(other)

	if err := engine.Process(filepath.Join(dir, "input.pre")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "user_repository.go")); err != nil {
		t.Fatalf("the output file was not generated next to the source file: %v", err)
	}

	entries, err := os.ReadDir(other)
	if err != nil {
		t.Fatalf("cannot read the working directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("files were generated in the working directory: %v", entries)
	}
}

func TestProcessReportsErrors(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name:    "missing frontmatter",
			file:    "no_frontmatter.pre",
			wantErr: "missing frontmatter",
		},
		{
			name:    "invalid yaml",
			file:    "invalid_yaml.pre",
			wantErr: "invalid frontmatter",
		},
		{
			name:    "no targets",
			file:    "no_targets.pre",
			wantErr: "no targets defined",
		},
		{
			name:    "target without output",
			file:    "no_output.pre",
			wantErr: "has no output file name",
		},
		{
			name:    "unknown variable",
			file:    "unknown_variable.pre",
			wantErr: "no field or method 'missing'",
		},
		{
			name:    "invalid template",
			file:    "invalid_template.pre",
			wantErr: "invalid template",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := engine.Process(filepath.Join("testdata", "errors", test.file))
			if err == nil {
				t.Fatal("expected an error, got none")
			}

			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("unexpected error message\ngot:  %v\nwant: %q", err, test.wantErr)
			}
		})
	}
}

func TestProcessFailsOnMissingFile(t *testing.T) {
	err := engine.Process(filepath.Join("testdata", "does_not_exist.pre"))
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "cannot read file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// optionsSource defines two targets that print their frontmatter index, so
// that every option test can check that the index is never renumbered.
const optionsSource = `---
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

func TestProcessWritesToAnOutputWriter(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n---\nhello\n")

	var out bytes.Buffer
	if err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.String() != "hello\n" {
		t.Errorf("unexpected content: %q", out.String())
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "out.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the output file must not be written (stat error: %v)", err)
	}
}

func TestProcessLogsEveryTarget(t *testing.T) {
	path := writeSource(t, optionsSource)

	var log bytes.Buffer
	if err := engine.ProcessWithOptions(path, engine.Options{Log: &log}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "generating first.txt\ngenerating second.txt\n"
	if log.String() != want {
		t.Errorf("unexpected log\ngot:  %q\nwant: %q", log.String(), want)
	}
}

func TestProcessSelectsASingleTarget(t *testing.T) {
	path := writeSource(t, optionsSource)
	dir := filepath.Dir(path)

	if err := engine.ProcessWithOptions(path, engine.Options{Target: "second.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "first.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the skipped target must not be generated (stat error: %v)", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	if string(got) != "1 second\n" {
		t.Errorf("the target_index must keep its frontmatter position: %q", got)
	}
}

func TestProcessSelectionDoesNotChangeTheGeneratedContent(t *testing.T) {
	all := writeSource(t, optionsSource)
	if err := engine.Process(all); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join(filepath.Dir(all), "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	selected := writeSource(t, optionsSource)
	if err := engine.ProcessWithOptions(selected, engine.Options{Target: "second.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(filepath.Dir(selected), "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("the selection changed the generated content\ngot:  %q\nwant: %q", got, want)
	}

	var out bytes.Buffer
	viaStdout := writeSource(t, optionsSource)
	if err := engine.ProcessWithOptions(viaStdout, engine.Options{Stdout: &out, Target: "second.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out.Bytes(), want) {
		t.Errorf("stdout differs from the generated file\ngot:  %q\nwant: %q", out.Bytes(), want)
	}
}

func TestProcessStdoutRequiresASingleTarget(t *testing.T) {
	path := writeSource(t, optionsSource)

	var out bytes.Buffer
	err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out})
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "--target") {
		t.Errorf("the error should suggest --target: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("unexpected output: %q", out.String())
	}
}

func TestProcessRejectsDuplicateOutputs(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n  - output: \"out.txt\"\n---\nx\n")

	err := engine.Process(path)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "duplicate target output") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessRejectsUnknownTarget(t *testing.T) {
	path := writeSource(t, optionsSource)

	err := engine.ProcessWithOptions(path, engine.Options{Target: "missing.txt"})
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "available targets: first.txt, second.txt") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessTrimsOutputNames(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \" user.go \"\n    vars:\n      name: \"user\"\n---\n{{ output }} {{ .name }}\n")
	dir := filepath.Dir(path)

	var out bytes.Buffer
	err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out, Target: "  user.go  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.String() != "user.go user\n" {
		t.Errorf("unexpected content: %q", out.String())
	}

	if err := engine.Process(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "user.go")); err != nil {
		t.Errorf("the trimmed output file was not generated: %v", err)
	}
}

func TestProcessRejectsDuplicateOutputsAfterTrimming(t *testing.T) {
	path := writeSource(t, "---\ntargets:\n  - output: \"out.txt\"\n  - output: \" out.txt \"\n---\nx\n")

	err := engine.Process(path)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "duplicate target output") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessAcceptsUTF8BOM(t *testing.T) {
	path := writeSource(t, "\ufeff---\ntargets:\n  - output: \"out.txt\"\n---\nhello\n")

	var out bytes.Buffer
	if err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.String() != "hello\n" {
		t.Errorf("unexpected content: %q", out.String())
	}
}

func TestProcessKeepsIndentedDelimitersInTheFrontmatter(t *testing.T) {
	source := `---
targets:
  - output: "out.txt"
    vars:
      script: |-
        line one
        ---
        line two
---
script-start
{{ .script }}
script-end
`
	path := writeSource(t, source)

	var out bytes.Buffer
	if err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "script-start\nline one\n---\nline two\nscript-end\n"
	if out.String() != want {
		t.Errorf("unexpected content\ngot:  %q\nwant: %q", out.String(), want)
	}
}

func TestProcessRejectsIndentedOpeningDelimiter(t *testing.T) {
	path := writeSource(t, "  ---\ntargets:\n  - output: \"out.txt\"\n---\nbody\n")

	err := engine.Process(path)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "missing frontmatter") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessAutomaticVariablesOverrideFrontmatterVars(t *testing.T) {
	source := `---
targets:
  - output: "real.txt"
    vars:
      output: "fake.txt"
      target_index: 99
---
{{ output }} {{ .output }} {{ target_index }} {{ .target_index }}
`
	path := writeSource(t, source)

	var out bytes.Buffer
	if err := engine.ProcessWithOptions(path, engine.Options{Stdout: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "real.txt real.txt 0 0\n"
	if out.String() != want {
		t.Errorf("unexpected content\ngot:  %q\nwant: %q", out.String(), want)
	}
}

func TestProcessReaderGeneratesFilesInTheBaseDirectory(t *testing.T) {
	dir := t.TempDir()

	if err := engine.ProcessReader("<stdin>", strings.NewReader(optionsSource), dir, engine.Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	if string(got) != "1 second\n" {
		t.Errorf("unexpected content: %q", got)
	}
}

func TestProcessReaderResolvesIncludesFromTheBaseDirectory(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "partials"), 0o755); err != nil {
		t.Fatalf("cannot create the partials directory: %v", err)
	}

	header := "// Package {{ .package_name }}.\n"
	if err := os.WriteFile(filepath.Join(dir, "partials", "header.jet"), []byte(header), 0o644); err != nil {
		t.Fatalf("cannot write the partial: %v", err)
	}

	source := "---\ntargets:\n  - output: \"out.txt\"\n    vars:\n      package_name: \"repositories\"\n---\n{{ include \"./partials/header.jet\" }}"

	var out bytes.Buffer
	if err := engine.ProcessReader("<stdin>", strings.NewReader(source), dir, engine.Options{Stdout: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.String() != "// Package repositories.\n" {
		t.Errorf("unexpected content: %q", out.String())
	}
}

func TestProcessReaderNamesTheSourceInErrors(t *testing.T) {
	err := engine.ProcessReader("<stdin>", strings.NewReader(""), t.TempDir(), engine.Options{})
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	if !strings.Contains(err.Error(), "<stdin>") {
		t.Errorf("the error should name the source: %v", err)
	}
}

func TestProcessReaderMatchesTheFileOutput(t *testing.T) {
	readerDir := t.TempDir()
	if err := engine.ProcessReader("<stdin>", strings.NewReader(optionsSource), readerDir, engine.Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fromReader, err := os.ReadFile(filepath.Join(readerDir, "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	fileDir := t.TempDir()
	path := filepath.Join(fileDir, "input.pre")
	if err := os.WriteFile(path, []byte(optionsSource), 0o644); err != nil {
		t.Fatalf("cannot write the source file: %v", err)
	}

	if err := engine.Process(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fromFile, err := os.ReadFile(filepath.Join(fileDir, "second.txt"))
	if err != nil {
		t.Fatalf("cannot read the generated file: %v", err)
	}

	if !bytes.Equal(fromReader, fromFile) {
		t.Errorf("stdin and file input differ\ngot:  %q\nwant: %q", fromReader, fromFile)
	}
}
