# pre

`pre` is a lightweight, general-purpose source code preprocessor written in Go.

It processes self-contained `.pre` files: a plain YAML frontmatter that describes
what has to be generated, followed by a [Jet](https://github.com/CloudyKit/jet)
template body that renders it.

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

Running `pre user_repository.pre` writes `user_repository.go` next to the
`.pre` file.

## Installation

```bash
go install github.com/inticardo/pre/cmd/pre@latest
```

Or build it from a checkout:

```bash
make build      # static binary in bin/pre
make dev        # development binary with race detector and debug symbols
make cross      # cross-compiled binaries in bin/
```

## Usage

```
pre [options] <file.pre>
```

| Option | Description |
| --- | --- |
| `-h`, `--help` | Print the usage information and exit |
| `-V`, `--version` | Print the version and exit |
| `-v`, `--verbose` | Print the files being generated to stderr |
| `--stdout` | Print the generated content to stdout instead of writing the files |
| `--target <output>` | Process only the target with the given output name |

`pre` writes nothing to stdout on success and exits with status code `0`, except
in `--stdout` mode, where stdout carries the generated content. Every error is
reported on stderr and exits with a non-zero status code.

### Inspecting the output

`--stdout` renders the selected target and prints it to stdout without writing
any file, which keeps it usable in tests and pipelines:

```bash
pre --stdout --target user_repository.go templates/repositories.pre | diff - expected.go
```

Because `.pre` files can define several targets, `--stdout` needs exactly one
selected target: pass `--target <output>` when the frontmatter defines more than
one. Selecting a target never renumbers it: `{{ target_index }}` always keeps
its zero based position in the frontmatter, so the generated content is
byte-identical whether the target is selected or not.

Progress messages (`--verbose`) are written to **stderr**, so they never
pollute the generated content on stdout:

```bash
$ pre --verbose templates/repositories.pre
generating user_repository.go
generating product_repository.go
```

### Reading from standard input

Use `-` as the file argument to read the `.pre` source from stdin. Since there
is no source file, included partials are resolved and output files are written
relative to the **current working directory**:

```bash
cat templates/repositories.pre | pre -
```

Combine it with `--stdout` to use `pre` as a filter:

```bash
git show HEAD:templates/repositories.pre | pre --stdout --target user_repository.go -
```

Options must be given before the file argument: `pre --stdout -`, not
`pre - --stdout`. To process a real file named `-`, use `./-`.

## The `.pre` file format

A `.pre` file is made of two parts separated by two `---` lines:

```
---
<YAML frontmatter>
---
<Jet template body>
```

### Frontmatter

The frontmatter is plain YAML and must contain a `targets` list. Every target
defines the name of the file to generate and the variables exposed to the
template body:

```yaml
targets:
  - output: "user_repository.go"   # required: name of the generated file
    vars:                          # optional: variables available to the body
      entity: "User"
      package_name: "repositories"
```

The `output` name is resolved relative to the directory of the `.pre` file, and
missing intermediate directories are created automatically. Subdirectories are
allowed, but the path is not sandboxed: `..` segments are not rejected, so a
mistyped output can point outside the source directory. Keep output paths inside
the project. Output names must be unique, because they identify the target for
the `--target` option.

### Template body

Everything after the second `---` line is a Jet template. The variables of the
target are available as the current context, so they are accessed with a
leading dot:

```
{{ .package_name }}
```

The name of the generated file is injected automatically under the `output`
key, both as a context value and as a plain variable:

```
{{ output }}    {# same as {{ .output }} #}
```

The zero based position of the target in the frontmatter is injected as well,
under the `target_index` key:

```
{{ target_index }}    {# same as {{ .target_index }} #}
```

Both automatic variables take precedence over the variables defined in the
frontmatter.

> Jet does not support the `$` root reference of the Go template engine. Inside
> a `range` block the parent context stays available, so use `{{ .entity }}`
> instead of `{{ $.entity }}`.

> Helper functions are called with the usual parenthesis syntax, also when the
> argument is a context value: `{{ pascalCase(.entity) }}`.

## Output location

Generated files are written **next to the source `.pre` file**, no matter the
directory `pre` is invoked from. The same applies to the partials and blocks
referenced from the template body: they are resolved relative to the source
file, not relative to the terminal's working directory.

The engine does not sandbox these paths: an absolute include path containing
`..` can reach outside the source directory, so keep partial paths inside the
project.

When the source is read from stdin (`pre -`), there is no source file, so both
the partials and the generated files use the current working directory instead.

```
project/
├── templates/
│   ├── repositories.pre
│   └── partials/
│       └── header.jet
└── repositories/
    └── user_repository.go   # generated
```

## Examples

The [`examples/`](examples) directory contains self-contained projects that
generate code and configuration for several languages:

| Example | Language | What it shows |
| --- | --- | --- |
| [`gdscript-generics`](examples/gdscript-generics) | GDScript (Godot 4) | Typed containers, since GDScript has no generics |
| [`c-generics`](examples/c-generics) | C | Header-only typed vectors built from one template |
| [`go-api`](examples/go-api) | Go | Struct tags and validation generated from field lists |
| [`python-dataclasses`](examples/python-dataclasses) | Python | Dataclasses, type hints and defaults |
| [`sql-schema`](examples/sql-schema) | SQL | DDL from nested table and column ranges |
| [`config-yaml`](examples/config-yaml) | YAML | A layout inherited with `extends`, `block` and `yield` |

```bash
make examples        # generate every example into its generated/ directory
make examples-check  # validate the output with the available toolchains
```

The generated files are ignored by git, because they can be recreated from the
`.pre` sources at any time.

## Helpers

`pre` registers the following helpers, which take precedence over the built-in
functions of Jet:

| Helper | Description | Example |
| --- | --- | --- |
| `lower` | Lower cases the string | `lower("AbC")` → `abc` |
| `upper` | Upper cases the string | `upper("AbC")` → `ABC` |
| `trimSpace` | Removes surrounding white space | `trimSpace(" a ")` → `a` |
| `replace` | Replaces occurrences of a substring | `replace("a.b", ".", "-", -1)` → `a-b` |
| `hasPrefix` | Reports whether the string starts with a prefix | `hasPrefix("pre", "p")` → `true` |
| `hasSuffix` | Reports whether the string ends with a suffix | `hasSuffix("pre", "e")` → `true` |
| `snakeCase` | Converts to `snake_case` | `snakeCase("UserRepository")` → `user_repository` |
| `camelCase` | Converts to `camelCase` | `camelCase("user_repository")` → `userRepository` |
| `pascalCase` | Converts to `PascalCase` | `pascalCase("user_repository")` → `UserRepository` |
| `kebabCase` | Converts to `kebab-case` | `kebabCase("UserRepository")` → `user-repository` |
| `titleCase` | Converts to `Title Case` | `titleCase("user_repository")` → `User Repository` |

### Random helpers

| Helper | Description | Example |
| --- | --- | --- |
| `randomInt` | Random integer within the given range, both bounds included | `randomInt(1, 6)` → `4` |
| `randomFloat` | Random float within the given range, the upper bound excluded | `randomFloat(0, 1)` → `0.42` |
| `randomString` | Random alphanumeric string of the given length | `randomString(8)` → `a3Fz09Qk` |
| `randomAlpha` | Random string of letters of the given length | `randomAlpha(8)` → `qTzMbWnR` |

The numeric arguments accept both integers and floats, so `randomFloat(0, 1)`
and `randomFloat(0.0, 1.0)` are equivalent.

> These helpers draw from a random source, so the generated content is **not
> reproducible**: running `pre` twice over the same file produces different
> values.

## Dependencies

The dependencies are vendored into the `vendor/` directory of the project root,
the same way `node_modules` or the PHP `vendor` directory work. The directory is
generated, never edited by hand, and it is not versioned.

```bash
make vendor      # refresh the vendor directory (go mod tidy && go mod vendor)
```

Every build uses the vendored dependencies through the `-mod=vendor` flag, so
the compilation is hermetic: it never reaches the network and always uses the
exact same dependency versions.

```bash
make build       # static binary in bin/pre
make dev         # development binary with race detector and debug symbols
make cross       # cross-compiled binaries in bin/
```

> The build script is always launched with `go run -mod=mod`, because Go
> defaults to the vendor directory when it is present. That way the script can
> still run, and regenerate the vendor directory, when `go.mod` and
> `vendor/modules.txt` are out of sync.

The vendor directory is only generated when it is missing, so a build with an
up to date vendor directory needs neither the network nor the module cache. If
it becomes stale, the build fails with Go's `inconsistent vendoring` error and
`make vendor` regenerates it.

## Development

```bash
make fmt            # format the source code
make vet            # run go vet
make test           # run the test suite
make tidy           # tidy the go.mod module and refresh the vendor directory
make vendor         # refresh the vendor directory only
make examples       # generate every example
make examples-check # validate the example output with local toolchains
make clean          # remove the bin and vendor directories and the example output
```
