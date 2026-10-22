package examples_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inticardo/pre/internal/engine"
)

// TestExamples renders every example inside an isolated temporary directory.
// Copying the source first means a stale generated file can never make the test
// pass: the output under test was produced by this run.
func TestExamples(t *testing.T) {
	sources, err := filepath.Glob("*/*.pre")
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("no example sources found")
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			dir := t.TempDir()
			example := filepath.Dir(source)
			if err := os.CopyFS(dir, os.DirFS(example)); err != nil {
				t.Fatalf("copy %s: %v", example, err)
			}
			if err := os.RemoveAll(filepath.Join(dir, "generated")); err != nil {
				t.Fatalf("remove generated output: %v", err)
			}

			if err := engine.Process(filepath.Join(dir, filepath.Base(source))); err != nil {
				t.Fatalf("process %s: %v", source, err)
			}

			entries, err := os.ReadDir(filepath.Join(dir, "generated"))
			if err != nil {
				t.Fatalf("generated directory: %v", err)
			}
			if len(entries) == 0 {
				t.Fatal("no files were generated")
			}
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					t.Fatal(err)
				}
				if info.Size() == 0 {
					t.Errorf("generated file %s is empty", entry.Name())
				}
			}
		})
	}
}
