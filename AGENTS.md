# AGENTS.md — Technical Specification & Guidelines for `pre`

`pre` is a lightweight, general-purpose source code preprocessor CLI tool written in Go. It processes self-contained `.pre` files structured with a pure YAML frontmatter header and a body powered by the **Jet** template engine (`CloudyKit/jet/v6`).

---

## 1. Project Philosophy & Core Requirements

1. **Language & Codebase**: The entire codebase, CLI messages, comments, and documentation **must be strictly in English**.
2. **Strict Syntax Separation**: 
   - **Frontmatter**: Plain, valid YAML only between the top `---` delimiters. No template logic or Jet tags inside the YAML block.
   - **Template Body**: The content below the second `---` is treated as the Jet template.
3. **Deterministic Output Location**: Output files are resolved relative to the source `.pre` file's directory. When the source is read from stdin (`pre -`), the current working directory is used instead. Subdirectories are allowed, but the path is not sandboxed: `..` segments are not rejected.
4. **Relative Path Resolution**: Any included assets or partial templates must resolve paths relative to the source `.pre` file's directory, ignoring the current terminal working directory. With `pre -` the current working directory is used instead, because there is no source file.
5. **Context Injection (`output`)**: The target filename must be automatically injected into Jet's variable scope under the `output` key so it can be referenced inside the body template.
6. **Output & Unix Discipline**:
   - **Silent on success**: The program outputs **nothing** to `stdout` upon successful execution and exits with status code `0`, except in `--stdout` mode, where `stdout` carries the generated content (and nothing else).
   - **Errors to `stderr`**: All error messages, configuration issues, or template execution failures must be written exclusively to `stderr` with a non-zero exit code. Diagnostics such as the `--verbose` progress messages are written to `stderr` as well.

---

## 2. CLI Design & POSIX Conventions

The CLI interface must follow standard POSIX conventions using standard Go packages (`flag` or similar):

- **Usage**: `pre [options] <file.pre>`
- **Standard input**: A `-` file argument reads the `.pre` source from stdin; includes and generated files are then resolved relative to the current working directory. There is no `--stdin` flag and options are not auto-detected from a piped stdin.
- **Flags**:
  - `-h`, `--help`: Prints usage information, options, and examples to `stdout` and exits with `0`.
  - `-V`, `--version`: Prints the binary version to `stdout` and exits with `0`.
  - `-v`, `--verbose`: Prints the files being generated to `stderr`.
  - `--stdout`: Prints the generated content to `stdout` instead of writing files. It requires a single selected target (use `--target` when the frontmatter defines several).
  - `--target <output>`: Processes only the target with the given output name.
- **Deterministic target index**: `--target` only skips targets, it never renumbers them. `target_index` is always the zero based position in the frontmatter, so the generated content does not depend on the selection. Output names must therefore be unique in the frontmatter.

---

## 3. Project Architecture

```text
pre/
├── go.mod
├── go.sum
├── build.go                # Build script (static, dynamic, cross, dev, vendor, clean)
├── Makefile                # Developer entrypoints (build, test, examples)
├── README.md
├── cmd/
│   └── pre/
│       └── main.go          # CLI entrypoint, argument parsing, error exits
├── internal/
│   ├── engine/
│   │   └── engine.go        # Core preprocessor engine: parses YAML, runs Jet, writes files
│   └── helpers/
│       ├── helpers.go        # Main registry function for custom Jet helpers
│       ├── string_helpers.go # String helper function implementations
│       ├── list_helpers.go   # Collection helper function implementations
│       ├── random_helpers.go # Random number and text helper function implementations
│       └── time_helpers.go   # Time helper function implementations
├── examples/               # Self-contained example projects (see section 8)
│   ├── README.md           # Index of the examples and how to run them
│   ├── examples_test.go    # Renders every example in an isolated temporary directory
│   └── check.sh            # Validates the generated output with local toolchains
├── vendor/                 # Generated dependencies (go mod vendor), never edited by hand
```

---

## 4. Modular Helper Functions (`internal/helpers`)

All custom Jet template helper functions must be isolated in the `internal/helpers` package to make extending the engine straightforward.

### Automatic Variables

The engine injects two variables in the template scope of every target:

| Variable | Description |
| --- | --- |
| `output` | Name of the file being generated |
| `target_index` | Zero based position of the target in the frontmatter |

Both are exposed as context values (`{{ .output }}`) and as plain variables
(`{{ output }}`), and they always take precedence over the variables defined in
the frontmatter.

`target_index` is always the zero based position of the target in the
frontmatter. Selecting targets with `--target` only skips the other ones, so it
never renumbers them and the generated content never depends on the selection.
Output names must be unique for the same reason.

### Registration Example (`internal/helpers/helpers.go`):
```go
package helpers

import "github.com/CloudyKit/jet/v6"

// RegisterHelpers binds all custom helper functions to the provided Jet Set.
//
// The registrations are grouped by domain, with one registerXHelpers function
// per helper file: string, list, random and time helpers.
func RegisterHelpers(set *jet.Set) {
	registerStringHelpers(set)
	registerListHelpers(set)
	registerRandomHelpers(set)
	registerTimeHelpers(set)
}
```

```go
package helpers

import (
	"strings"

	"github.com/CloudyKit/jet/v6"
)

// registerStringHelpers binds all string helpers to the provided Jet Set,
// implementing each helper inside the file that owns the domain.
func registerStringHelpers(set *jet.Set) {
	set.AddGlobal("lower", func(in string) string {
		return strings.ToLower(in)
	})
	set.AddGlobal("upper", func(in string) string {
		return strings.ToUpper(in)
	})
}
```

Note: `AddGlobalFunc` only accepts `jet.Func` values (`func(Arguments) reflect.Value`), so plain Go functions must be registered with `AddGlobal`.

`snakeCase`, `pascalCase`, `camelCase` and the other word helpers are
implemented in `string_helpers.go`; the snippet above only illustrates the
registration mechanism. Helper implementations are grouped by domain:
`string_helpers.go` (text), `list_helpers.go` (collections), `random_helpers.go`
(random values) and `time_helpers.go` (current time), and each group defines
its own `registerXHelpers(set)` binding function when it needs `jet.Func`
members. Helpers that accept a flexible argument list or coercible arguments
(such as `padStart` and the random helpers) must be registered with
`AddGlobalFunc`; plain fixed signature functions use `AddGlobal`.

---

## 5. Clean `.pre` File Example

The Frontmatter contains **only pure YAML data**, and the body handles template expressions.

```
---
targets:
  - output: "user_repository.go"
    vars:
      entity: "User"
      package_name: "repositories"
      fields:
        - name: "ID"
          type: "int"
        - name: "Email"
          type: "string"

  - output: "product_repository.go"
    vars:
      entity: "Product"
      package_name: "repositories"
      fields:
        - name: "ID"
          type: "int"
        - name: "Price"
          type: "float64"
---
package {{ .package_name }}

// Code generated for target file: {{ output }}

type {{ .entity }}Repository struct {
    client DatabaseClient
}

{{ range index, field := .fields }}
// Field {{ index + 1 }}
func (r *{{ .entity }}Repository) Get{{ field.name }}() {{ field.type }} {
    var val {{ field.type }}
    return val
}
{{ end }}
```

> **Jet syntax note**: Jet v6 does not implement the `$` root reference of the Go
> template engine, so `{{ $.entity }}` is a parse error. Inside a `range` block
> the parent context remains available, therefore `{{ .entity }}` is the correct
> form. The `output` key is injected both as a context value (`{{ .output }}`)
> and as a plain variable (`{{ output }}`).

---

## 6. Implementation Steps for AI Agents

1. **CLI Setup**: Implement flag parsing for `--help` / `-h` and validate that exactly one `.pre` file path argument is provided. `--target` filters the targets and `--stdout` renders to `stdout` without writing files.
2. **File Processing**: Read the source `.pre` file, locate the `---` delimiters, and split the content strictly into YAML frontmatter and template body. A `-` file argument reads the source from stdin instead.
3. **YAML Parsing**: Decode the frontmatter into target configuration structs using `gopkg.in/yaml.v3`. Reject duplicate output names.
4. **Jet Initialization**:
   - Create a Jet `Set` with a loader scoped to the source file's directory
     (or to the base directory when the source is read from stdin).
   - Disable the HTML escaping with `jet.WithSafeWriter(nil)`: the generated
     content is source code, and the default escaping would corrupt it.
   - Register custom functions using `helpers.RegisterHelpers(set)`.
5. **Target Execution Loop**:
   - Iterate the frontmatter targets keeping their original zero based index;
     `--target` only skips targets, it never renumbers them.
   - For each selected target, inject the target `output` string and the
     `target_index` directly into the target variables map.
   - Execute the template body against the variables.
   - Write the generated output file to `filepath.Join(sourceDir, target.Output)`.
     When `--stdout` is used, emit the exact same content to `stdout` instead,
     after every selected target has been rendered successfully.
6. **Error Management**: On any failure (missing arguments, malformed YAML, invalid Jet expressions, write errors), log a clear error message to `os.Stderr` and terminate with `os.Exit(1)`.

---

## 7. Build & Dependencies

- Dependencies are vendored in the `vendor/` directory of the project root, the
  same way `node_modules` or the PHP `vendor` directory work. The directory is
  **generated** (`go mod vendor`), never edited by hand, and it is ignored by
  git.
- Every build uses the `-mod=vendor` flag, so the compilation is hermetic and
  never reaches the network.
- The build script is always launched with `go run -mod=mod build.go ...`. Go
  defaults to the vendor directory when it is present, so this is the only way
  to run the script when `go.mod` and `vendor/modules.txt` are out of sync.
- After changing the dependencies, always refresh the vendor directory with
  `make vendor` (`go mod tidy && go mod vendor`).

---

## 8. Examples

`examples/` contains self-contained projects that exercise the engine in
different languages. They are part of the specification of the user experience,
so they must be kept working:

- Every example lives in its own directory with one `.pre` source and, when
  needed, a `partials/` directory and a `README.md`.
- Output is written to `examples/<name>/generated/`, which is ignored by git and
  recreated with `make examples`.
- Examples must be deterministic: never use the random or time helpers in them.
- `examples/examples_test.go` renders every example in a temporary directory, so
  `go test ./...` catches broken templates without requiring external
  toolchains.
- `make examples-check` validates the generated code with whatever toolchains
  are installed (`gcc`, `go`, `python3`, `sqlite3`, `godot`) and skips the rest.
