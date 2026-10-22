# SQL schema

A table description in YAML becomes a `CREATE TABLE` script. Adding a column or
a table never requires touching the SQL by hand.

It generates `generated/schema.sql`.

What it demonstrates:

- Nested `range` blocks over tables and columns.
- Conditional fragments for `PRIMARY KEY`, `NOT NULL`, `DEFAULT` and
  `REFERENCES`.
- `snakeCase` applied to table and column names.

```bash
./bin/pre examples/sql-schema/schema.pre
sqlite3 :memory: ".read examples/sql-schema/generated/schema.sql"
```
