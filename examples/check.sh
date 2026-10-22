#!/bin/sh
# Validate the generated example output with the toolchains available on this
# machine. Missing toolchains are reported and skipped, so the script stays
# usable on minimal setups.
set -eu

cd "$(dirname "$0")/.."

failed=0
skipped=0

skip() {
	printf 'skip: %s\n' "$1"
	skipped=$((skipped + 1))
}

check() {
	description=$1
	shift
	printf 'check: %s\n' "$description"
	if ! "$@"; then
		failed=$((failed + 1))
	fi
}

for file in examples/c-generics/generated/*.h; do
	if command -v gcc >/dev/null 2>&1; then
		check "$file" gcc -fsyntax-only -x c "$file"
	else
		skip "gcc not found (C example)"
		break
	fi
done

if command -v go >/dev/null 2>&1; then
	check "examples/go-api" go vet ./examples/go-api/generated
else
	skip "go not found (Go example)"
fi

for file in examples/python-dataclasses/generated/*.py; do
	if command -v python3 >/dev/null 2>&1; then
		check "$file" python3 -c 'import ast, sys; ast.parse(open(sys.argv[1]).read(), sys.argv[1])' "$file"
	else
		skip "python3 not found (Python example)"
		break
	fi
done

if command -v sqlite3 >/dev/null 2>&1; then
	check "examples/sql-schema" sqlite3 :memory: ".read examples/sql-schema/generated/schema.sql"
else
	skip "sqlite3 not found (SQL example)"
fi

for file in examples/gdscript-generics/generated/*.gd; do
	if command -v godot >/dev/null 2>&1; then
		check "$file" godot --headless --check-only --script "$file"
	else
		skip "godot not found (GDScript example)"
		break
	fi
done

if command -v python3 >/dev/null 2>&1 && python3 -c 'import yaml' >/dev/null 2>&1; then
	check "examples/config-yaml" python3 -c 'import sys, yaml; yaml.safe_load(open(sys.argv[1]))' examples/config-yaml/generated/docker-compose.yml
else
	skip "PyYAML not found (YAML example)"
fi

if [ "$failed" -ne 0 ]; then
	printf '%d check(s) failed\n' "$failed" >&2
	exit 1
fi

printf 'all available checks passed (%d skipped)\n' "$skipped"
