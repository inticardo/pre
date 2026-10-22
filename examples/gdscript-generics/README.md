# Typed lists for GDScript

GDScript has typed arrays (`Array[int]`), but no generics: a list class has to be
written once per element type. This example generates one class per type from a
single template, so the three files always stay in sync.

It generates `generated/int_list.gd`, `generated/float_list.gd` and
`generated/vector2_list.gd` for Godot 4.

What it demonstrates:

- Several targets sharing one template body.
- `include` with a partial that reads the automatic `output` variable.
- A `class_name` derived from the frontmatter, so the classes are registered
  globally in the editor.

```bash
../../bin/pre typed_lists.pre
# or, from the repository root:
./bin/pre examples/gdscript-generics/typed_lists.pre
```

Usage in Godot:

```gdscript
var scores := IntList.new()
scores.add(10)
print(scores.first_or(0))
```
