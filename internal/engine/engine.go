// Package engine implements the core of pre: it reads a .pre file, splits it
// into its YAML frontmatter and its Jet template body, and renders the body
// once per configured target.
//
// Every target is rendered with two automatic variables injected into the
// template scope: "output", the name of the file being generated, and
// "target_index", the zero based position of the target in the frontmatter.
// Both are available as context values ({{ .output }}) and as plain variables
// ({{ output }}).
package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"gopkg.in/yaml.v3"

	"github.com/inticardo/pre/internal/helpers"
)

// Delimiter separates the frontmatter from the template body.
const Delimiter = "---"

// Target describes a single file to be generated.
type Target struct {
	// Output is the name of the file to generate, relative to the directory
	// of the source .pre file.
	Output string `yaml:"output"`

	// Vars holds the variables exposed to the template body.
	Vars map[string]any `yaml:"vars"`
}

// Config is the YAML frontmatter of a .pre file.
type Config struct {
	Targets []Target `yaml:"targets"`
}

// Options configures a ProcessWithOptions run.
type Options struct {
	// Stdout, when non-nil, receives the rendered content instead of writing
	// the output files. The content is emitted only after every selected
	// target has been rendered successfully.
	Stdout io.Writer

	// Log, when non-nil, receives one progress line before each target is
	// rendered.
	Log io.Writer

	// Target, when not empty, restricts the run to the target with that
	// output name.
	Target string
}

// Process reads the given .pre file and generates every configured target.
//
// The generated files are always written next to the source .pre file,
// regardless of the current working directory.
func Process(sourcePath string) error {
	return ProcessWithOptions(sourcePath, Options{})
}

// ProcessWithOptions is like Process but accepts additional options.
//
// Selecting a target only skips the other ones: the target_index variable
// keeps its zero based position in the frontmatter, so the generated content
// never depends on the selection.
func ProcessWithOptions(sourcePath string, opts Options) error {
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("cannot resolve path %q: %w", sourcePath, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	return process(absPath, data, filepath.Dir(absPath), opts)
}

// ProcessReader reads the .pre source from r instead of from a file.
//
// The name is only used in error messages. Since there is no source file, the
// included partials and the generated files are resolved relative to baseDir.
func ProcessReader(name string, r io.Reader, baseDir string, opts Options) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", name, err)
	}

	return process(name, data, baseDir, opts)
}

// process renders the source data and writes (or prints) the selected targets.
// It is shared by the file and the reader entry points.
func process(name string, data []byte, baseDir string, opts Options) error {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("cannot resolve base directory %q: %w", baseDir, err)
	}

	frontmatter, body, err := split(string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	var config Config
	if err := yaml.Unmarshal([]byte(frontmatter), &config); err != nil {
		return fmt.Errorf("%s: invalid frontmatter: %w", name, err)
	}

	if len(config.Targets) == 0 {
		return fmt.Errorf("%s: no targets defined in frontmatter", name)
	}

	for i := range config.Targets {
		config.Targets[i].Output = strings.TrimSpace(config.Targets[i].Output)
		if config.Targets[i].Output == "" {
			return fmt.Errorf("%s: target %d/%d has no output file name", name, i+1, len(config.Targets))
		}
	}

	if err := validateUniqueOutputs(config.Targets); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	selected, err := selectTarget(config.Targets, opts.Target)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	if opts.Stdout != nil && selected < 0 && len(config.Targets) > 1 {
		return fmt.Errorf("%s: --stdout requires a single target, but the frontmatter defines %d; use --target to select one", name, len(config.Targets))
	}

	// The loader is rooted at the base directory so that included partials
	// and imported blocks are resolved relative to the .pre file (or to the
	// base directory when the source is read from stdin), no matter where the
	// tool is invoked from. The safe writer is disabled because the generated
	// content is source code, not HTML.
	set := jet.NewSet(
		jet.NewOSFileSystemLoader(absBase),
		jet.WithSafeWriter(nil),
	)
	helpers.RegisterHelpers(set)

	template, err := set.Parse(filepath.Base(name), body)
	if err != nil {
		return fmt.Errorf("%s: invalid template: %w", name, err)
	}

	total := len(config.Targets)

	var rendered [][]byte
	for i, target := range config.Targets {
		if selected >= 0 && i != selected {
			continue
		}

		content, err := renderTarget(template, target, i, total, opts)
		if err != nil {
			return err
		}

		if opts.Stdout != nil {
			rendered = append(rendered, content)
			continue
		}

		if err := writeTarget(absBase, target, i, total, content); err != nil {
			return err
		}
	}

	if opts.Stdout != nil {
		for _, content := range rendered {
			if _, err := opts.Stdout.Write(content); err != nil {
				return fmt.Errorf("cannot write to stdout: %w", err)
			}
		}
	}

	return nil
}

// validateUniqueOutputs rejects frontmatter definitions that generate the same
// file more than once, which would make the target selection ambiguous.
func validateUniqueOutputs(targets []Target) error {
	seen := make(map[string]int, len(targets))
	for i, target := range targets {
		if first, ok := seen[target.Output]; ok {
			return fmt.Errorf("duplicate target output %q (targets %d and %d)", target.Output, first+1, i+1)
		}
		seen[target.Output] = i
	}
	return nil
}

// selectTarget returns the zero based index of the requested target, or -1
// when every target has to be processed.
func selectTarget(targets []Target, output string) (int, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return -1, nil
	}

	for i, target := range targets {
		if target.Output == output {
			return i, nil
		}
	}

	available := make([]string, len(targets))
	for i, target := range targets {
		available[i] = target.Output
	}
	return -1, fmt.Errorf("target %q not found; available targets: %s", output, strings.Join(available, ", "))
}

// renderTarget executes the template for a single target and returns the
// rendered content.
func renderTarget(template *jet.Template, target Target, index, total int, opts Options) ([]byte, error) {
	context := make(map[string]any, len(target.Vars)+2)
	for name, value := range target.Vars {
		context[name] = value
	}
	context["output"] = target.Output
	context["target_index"] = index

	variables := make(jet.VarMap, len(target.Vars)+2)
	for name, value := range target.Vars {
		variables[name] = reflect.ValueOf(value)
	}
	variables["output"] = reflect.ValueOf(target.Output)
	variables["target_index"] = reflect.ValueOf(index)

	if opts.Log != nil {
		fmt.Fprintf(opts.Log, "generating %s\n", target.Output)
	}

	var buffer bytes.Buffer
	if err := template.Execute(&buffer, variables, context); err != nil {
		return nil, fmt.Errorf("target %d/%d (%s): %w", index+1, total, target.Output, err)
	}

	return buffer.Bytes(), nil
}

// writeTarget writes the rendered content to the output file of the target.
func writeTarget(baseDir string, target Target, index, total int, content []byte) error {
	outputPath := filepath.Join(baseDir, filepath.FromSlash(target.Output))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("target %d/%d (%s): cannot create output directory: %w", index+1, total, target.Output, err)
	}

	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("target %d/%d (%s): cannot write output file: %w", index+1, total, target.Output, err)
	}

	return nil
}

// split separates the YAML frontmatter from the template body.
//
// The frontmatter is delimited by two lines starting with "---" at column 0
// (trailing spaces and tabs are ignored); everything after the second
// delimiter is the template body.
func split(content string) (frontmatter, body string, err error) {
	content = strings.TrimPrefix(content, "\ufeff")
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	start := -1
	for i, line := range lines {
		if isDelimiter(line) {
			start = i
			break
		}
		if strings.TrimSpace(line) != "" {
			break
		}
	}
	if start < 0 {
		return "", "", errors.New("missing frontmatter: the file must start with a \"---\" line")
	}

	end := -1
	for i := start + 1; i < len(lines); i++ {
		if isDelimiter(lines[i]) {
			end = i
			break
		}
	}
	if end < 0 {
		return "", "", errors.New("unterminated frontmatter: a second \"---\" line is required")
	}

	return strings.Join(lines[start+1:end], "\n"), strings.Join(lines[end+1:], "\n"), nil
}

// isDelimiter reports whether the line is a frontmatter delimiter. The marker
// must start at column 0 so that indented "---" lines inside YAML block
// scalars are not mistaken for the end of the frontmatter.
func isDelimiter(line string) bool {
	return strings.TrimRight(line, " \t") == Delimiter
}
