# Examples

Each directory is a self-contained `.pre` project: the frontmatter describes the
targets and their variables, and the Jet body renders them into the `generated/`
directory next to the source. The generated files are not versioned; run the
examples to produce them.

| Example | Language | What it shows |
| --- | --- | --- |
| [`gdscript-generics`](gdscript-generics) | GDScript (Godot 4) | Typed containers, since GDScript has no generics |
| [`c-generics`](c-generics) | C | Header-only typed vectors built from one template |
| [`go-api`](go-api) | Go | Struct tags and validation generated from field lists |
| [`python-dataclasses`](python-dataclasses) | Python | Dataclasses, type hints and defaults |
| [`sql-schema`](sql-schema) | SQL | DDL from nested table and column ranges |
| [`config-yaml`](config-yaml) | YAML | A layout inherited with `extends`, `block` and `yield` |

```bash
make examples        # build bin/pre and generate every example
make examples-check  # validate the output with the available toolchains
```

To generate a single example:

```bash
./bin/pre --verbose examples/go-api/models.pre
```

Some examples define several targets, so `--stdout` needs `--target`:

```bash
./bin/pre --stdout --target generated/user.go examples/go-api/models.pre
```

The examples never use the random helpers, so the output is reproducible and can
be diffed between runs.
