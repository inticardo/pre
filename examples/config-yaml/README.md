# Docker Compose file

A list of services and volumes becomes a Compose file. The YAML skeleton lives
in a partial template, while each section is provided by a Jet block.

It generates `generated/docker-compose.yml`.

What it demonstrates:

- `extends`: the body of `compose.pre` inherits the layout from
  `partials/compose.yml.jet`.
- `block` and `yield`: the layout yields the `services()` and `volumes()`
  sections, which the body overrides.
- Optional YAML keys: `ports` and `depends_on` only appear when they are not
  empty.

```bash
./bin/pre examples/config-yaml/compose.pre
python3 -c 'import yaml; yaml.safe_load(open("examples/config-yaml/generated/docker-compose.yml"))'
docker compose -f examples/config-yaml/generated/docker-compose.yml config
```

Note that a block is rendered where it is declared, so a reusable layout keeps
the `block` declarations in the parent and the overrides in the child. This is
the same inheritance model used by Jet layouts.

Two Jet details are worth remembering:

- `{{ if .values }}` is true for an empty list, because Jet only treats `nil` as
  false. Use `{{ if len(.values) > 0 }}` to test emptiness.
- `{{ range value := .values }}` binds the **index**; use two variables
  (`{{ range index, value := .values }}`) to get the elements.
