# Go API models

One model definition per target, rendered into Go structs with the JSON and
database tags already in place, plus a small validation method.

It generates `generated/user.go` and `generated/product.go`.

What it demonstrates:

- `snakeCase` for struct tags and error messages.
- Optional fields rendered as pointers with `omitempty`.
- Conditional code inside a `range` over the fields.
- Multi-line template blocks that keep the generated Go formatted.

```bash
./bin/pre examples/go-api/models.pre
gofmt -w examples/go-api/generated
go vet ./examples/go-api/generated
```

Inspect a single target without writing files:

```bash
./bin/pre --stdout --target generated/user.go examples/go-api/models.pre
```
