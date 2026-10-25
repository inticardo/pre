package helpers

import (
	"bytes"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
)

// render executes the given template body with the registered helpers.
func render(t *testing.T, body string) string {
	t.Helper()

	set := jet.NewSet(jet.NewInMemLoader())
	RegisterHelpers(set)

	template, err := set.Parse("test", body)
	if err != nil {
		t.Fatalf("cannot parse the template: %v", err)
	}

	var buffer bytes.Buffer
	if err := template.Execute(&buffer, jet.VarMap{}, nil); err != nil {
		t.Fatalf("cannot execute the template: %v", err)
	}

	return buffer.String()
}

// renderError executes the given template body expecting it to fail.
func renderError(t *testing.T, body string) error {
	t.Helper()

	set := jet.NewSet(jet.NewInMemLoader())
	RegisterHelpers(set)

	template, err := set.Parse("test", body)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer

	return template.Execute(&buffer, jet.VarMap{}, nil)
}

// renderWithVars executes the given template body with the registered helpers
// and the given variables exposed to the template, like the frontmatter does.
func renderWithVars(t *testing.T, body string, vars jet.VarMap) string {
	t.Helper()

	set := jet.NewSet(jet.NewInMemLoader())
	RegisterHelpers(set)

	template, err := set.Parse("test", body)
	if err != nil {
		t.Fatalf("cannot parse the template: %v", err)
	}

	var buffer bytes.Buffer
	if err := template.Execute(&buffer, vars, nil); err != nil {
		t.Fatalf("cannot execute the template: %v", err)
	}

	return buffer.String()
}

func TestStringHelpers(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{name: "lower", fn: lower, in: "UserRepository", want: "userrepository"},
		{name: "upper", fn: upper, in: "user_repository", want: "USER_REPOSITORY"},
		{name: "trimSpace", fn: trimSpace, in: "  padded  ", want: "padded"},
		{name: "snakeCase from camel", fn: snakeCase, in: "UserRepository", want: "user_repository"},
		{name: "snakeCase from snake", fn: snakeCase, in: "user_repository", want: "user_repository"},
		{name: "snakeCase from kebab", fn: snakeCase, in: "user-repository", want: "user_repository"},
		{name: "snakeCase from acronym", fn: snakeCase, in: "HTTPServer", want: "http_server"},
		{name: "snakeCase from spaces", fn: snakeCase, in: "User Repository", want: "user_repository"},
		{name: "snakeCase from digits", fn: snakeCase, in: "field2Name", want: "field2_name"},
		{name: "snakeCase empty", fn: snakeCase, in: "", want: ""},
		{name: "kebabCase", fn: kebabCase, in: "UserRepository", want: "user-repository"},
		{name: "camelCase", fn: camelCase, in: "user_repository", want: "userRepository"},
		{name: "camelCase from camel", fn: camelCase, in: "UserRepository", want: "userRepository"},
		{name: "pascalCase", fn: pascalCase, in: "user_repository", want: "UserRepository"},
		{name: "pascalCase from pascal", fn: pascalCase, in: "UserRepository", want: "UserRepository"},
		{name: "titleCase", fn: titleCase, in: "user_repository", want: "User Repository"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.fn(test.in); got != test.want {
				t.Errorf("%s(%q) = %q, want %q", test.name, test.in, got, test.want)
			}
		})
	}
}

func TestReplaceHelper(t *testing.T) {
	if got := replace("a.b.c", ".", "-", 2); got != "a-b-c" {
		t.Errorf("replace() = %q, want %q", got, "a-b-c")
	}

	if got := replace("a.b.c", ".", "-", -1); got != "a-b-c" {
		t.Errorf("replace() = %q, want %q", got, "a-b-c")
	}
}

func TestAffixHelpers(t *testing.T) {
	if !hasPrefix("preprocess", "pre") {
		t.Error("hasPrefix() = false, want true")
	}

	if hasSuffix("preprocess", "post") {
		t.Error("hasSuffix() = true, want false")
	}
}

func TestTrimAffixHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "trimPrefix removes the prefix", got: trimPrefix("tbl_user", "tbl_"), want: "user"},
		{name: "trimPrefix keeps other strings", got: trimPrefix("user", "tbl_"), want: "user"},
		{name: "trimPrefix keeps the whole prefix", got: trimPrefix("tbl_tbl_user", "tbl_"), want: "tbl_user"},
		{name: "trimPrefix with empty prefix", got: trimPrefix("user", ""), want: "user"},
		{name: "trimSuffix removes the suffix", got: trimSuffix("user_old", "_old"), want: "user"},
		{name: "trimSuffix keeps other strings", got: trimSuffix("user", "_old"), want: "user"},
		{name: "trimSuffix keeps the whole suffix", got: trimSuffix("user_old_old", "_old"), want: "user_old"},
		{name: "trimSuffix with empty suffix", got: trimSuffix("user", ""), want: "user"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestPadHelpers(t *testing.T) {
	tests := []struct {
		name string
		body string
		vars jet.VarMap
		want string
	}{
		{name: "padStart with digits", body: `{{ padStart("42", 6, "0") }}`, want: "000042"},
		{name: "padStart accepts numbers", body: `{{ padStart(42, 6, "0") }}`, want: "000042"},
		{name: "padStart with spaces", body: `{{ padStart("42", 6) }}`, want: `    42`},
		{name: "padStart leaves wide strings", body: `{{ padStart("1234567", 6, "0") }}`, want: "1234567"},
		{name: "padStart counts runes", body: `{{ padStart("áé", 4, "x") }}`, want: "xxáé"},
		{name: "padEnd with digits", body: `{{ padEnd("42", 5, ".") }}`, want: "42..."},
		{name: "padEnd with spaces", body: `{{ padEnd("42", 5) }}`, want: `42   `},
		{name: "padEnd counts runes", body: `{{ padEnd("áé", 4, "x") }}`, want: "áéxx"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderWithVars(t, test.body, test.vars); got != test.want {
				t.Errorf("%s = %q, want %q", test.body, got, test.want)
			}
		})
	}
}

func TestPadHelpersReportInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "negative width", body: `{{ padStart("42", -1, "0") }}`},
		{name: "empty padding", body: `{{ padStart("42", 6, "") }}`},
		{name: "non string input", body: `{{ padStart(slice("a"), 6, "0") }}`},
		{name: "non numeric width", body: `{{ padStart("42", "6", "0") }}`},
		{name: "too few arguments", body: `{{ padStart("42") }}`},
		{name: "too many arguments", body: `{{ padEnd("42", 5, " ", "x") }}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := renderError(t, test.body); err == nil {
				t.Errorf("%s: expected an error, got none", test.name)
			}
		})
	}
}

func TestIndentHelper(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "positive width", got: indent(4, "a\nb\nc"), want: "    a\n    b\n    c"},
		{name: "indents the last line without newline", got: indent(2, "a\nb"), want: "  a\n  b"},
		{name: "zero width keeps the content", got: indent(0, "a\nb"), want: "a\nb"},
		{name: "empty content", got: indent(4, ""), want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestDefaultHelper(t *testing.T) {
	tests := []struct {
		name string
		body string
		vars jet.VarMap
		want string
	}{
		{name: "empty string falls back", body: `{{ default(name, "main") }}`, vars: jet.VarMap{"name": reflect.ValueOf("")}, want: "main"},
		{name: "kept string", body: `{{ default(name, "main") }}`, vars: jet.VarMap{"name": reflect.ValueOf("server")}, want: "server"},
		{name: "zero number falls back", body: `{{ default(count, 5) }}`, vars: jet.VarMap{"count": reflect.ValueOf(0)}, want: "5"},
		{name: "kept number", body: `{{ default(count, 5) }}`, vars: jet.VarMap{"count": reflect.ValueOf(3)}, want: "3"},
		{name: "empty map falls back", body: `{{ default(items, "none") }}`, vars: jet.VarMap{"items": reflect.ValueOf(map[string]any{})}, want: "none"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderWithVars(t, test.body, test.vars); got != test.want {
				t.Errorf("%s = %q, want %q", test.body, got, test.want)
			}
		})
	}
}

func TestRandomIntHelper(t *testing.T) {
	for i := 0; i < 50; i++ {
		value, err := strconv.Atoi(strings.TrimSpace(render(t, "{{ randomInt(3, 7) }}")))
		if err != nil {
			t.Fatalf("the helper did not return an integer: %v", err)
		}

		if value < 3 || value > 7 {
			t.Fatalf("randomInt(3, 7) = %d, want a value between 3 and 7", value)
		}
	}

	if got := strings.TrimSpace(render(t, "{{ randomInt(5, 5) }}")); got != "5" {
		t.Errorf("randomInt(5, 5) = %q, want %q", got, "5")
	}
}

func TestRandomIntHelperAcceptsFloats(t *testing.T) {
	value, err := strconv.Atoi(strings.TrimSpace(render(t, "{{ randomInt(1.0, 2.0) }}")))
	if err != nil {
		t.Fatalf("the helper did not return an integer: %v", err)
	}

	if value < 1 || value > 2 {
		t.Fatalf("randomInt(1.0, 2.0) = %d, want a value between 1 and 2", value)
	}
}

func TestRandomFloatHelper(t *testing.T) {
	for i := 0; i < 50; i++ {
		value, err := strconv.ParseFloat(strings.TrimSpace(render(t, "{{ randomFloat(0, 1) }}")), 64)
		if err != nil {
			t.Fatalf("the helper did not return a float: %v", err)
		}

		if value < 0 || value >= 1 {
			t.Fatalf("randomFloat(0, 1) = %f, want a value between 0 and 1", value)
		}
	}
}

func TestRandomStringHelper(t *testing.T) {
	got := render(t, "{{ randomString(12) }}")

	if len(got) != 12 {
		t.Errorf("randomString(12) returned %d characters, want 12", len(got))
	}

	for _, r := range got {
		if !strings.ContainsRune(alphanumericChars, r) {
			t.Errorf("randomString(12) returned the invalid character %q", r)
		}
	}
}

func TestRandomAlphaHelper(t *testing.T) {
	got := render(t, "{{ randomAlpha(9) }}")

	if len(got) != 9 {
		t.Errorf("randomAlpha(9) returned %d characters, want 9", len(got))
	}

	for _, r := range got {
		if !strings.ContainsRune(alphaCharset, r) {
			t.Errorf("randomAlpha(9) returned the invalid character %q", r)
		}
	}
}

func TestRandomStringHelperWithZeroLength(t *testing.T) {
	if got := render(t, "{{ randomString(0) }}"); got != "" {
		t.Errorf("randomString(0) = %q, want an empty string", got)
	}
}

func TestRandomHelpersReportInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "minimum greater than maximum", body: "{{ randomInt(10, 1) }}"},
		{name: "float minimum greater than maximum", body: "{{ randomFloat(10, 1) }}"},
		{name: "non numeric argument", body: "{{ randomInt(\"a\", 1) }}"},
		{name: "non numeric float argument", body: "{{ randomFloat(\"a\", 1) }}"},
		{name: "missing argument", body: "{{ randomInt(1) }}"},
		{name: "too many arguments", body: "{{ randomString(1, 2) }}"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := renderError(t, test.body); err == nil {
				t.Errorf("%s: expected an error, got none", test.name)
			}
		})
	}
}

func TestRandomChoiceHelper(t *testing.T) {
	vars := jet.VarMap{
		"colors": reflect.ValueOf([]any{"red", "green", "blue"}),
	}

	seen := make(map[string]bool, 3)
	for i := 0; i < 100; i++ {
		got := renderWithVars(t, `{{ randomChoice(colors) }}`, vars)
		if got != "red" && got != "green" && got != "blue" {
			t.Fatalf("randomChoice(colors) = %q, want one of the list members", got)
		}
		seen[got] = true
	}

	if len(seen) < 2 {
		t.Errorf("randomChoice(colors) only returned %v in 100 tries", seen)
	}
}

func TestUUIDHelper(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for i := 0; i < 25; i++ {
		got := render(t, "{{ uuid() }}")
		if !pattern.MatchString(got) {
			t.Fatalf("uuid() = %q, want a version 4 UUID", got)
		}
	}

	if first, second := render(t, "{{ uuid() }}"), render(t, "{{ uuid() }}"); first == second {
		t.Errorf("two uuid() calls returned the same value %q", first)
	}
}

func TestDateHelper(t *testing.T) {
	original := clock
	defer func() { clock = original }()

	// The clock is pinned so that the test is deterministic.
	clock = func() time.Time {
		return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "year", body: `{{ date("2006") }}`, want: "2026"},
		{name: "day", body: `{{ date("2006-01-02") }}`, want: "2026-01-02"},
		{name: "time", body: `{{ date("15:04:05") }}`, want: "03:04:05"},
		{name: "now is an alias of date", body: `{{ now("2006-01-02") }}`, want: "2026-01-02"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := render(t, test.body); got != test.want {
				t.Errorf("%s = %q, want %q", test.body, got, test.want)
			}
		})
	}
}

func TestListHelpers(t *testing.T) {
	vars := jet.VarMap{
		"names":   reflect.ValueOf([]any{"Bob", "Anna", "Bob"}),
		"numbers": reflect.ValueOf([]any{1, 2, 3}),
		"fields": reflect.ValueOf([]any{
			map[string]any{"name": "Email", "index": 1},
			map[string]any{"name": "ID", "index": 0},
		}),
		"records": reflect.ValueOf([]any{
			map[string]any{"word": "a"},
			map[string]any{"word": "b"},
			map[string]any{"word": "a"},
		}),
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "join strings", body: `{{ join(names, ", ") }}`, want: "Bob, Anna, Bob"},
		{name: "join with an empty separator", body: `{{ join(names, "") }}`, want: "BobAnnaBob"},
		{name: "join numbers", body: `{{ join(numbers, "|") }}`, want: "1|2|3"},
		{name: "first element", body: `{{ first(names) }}`, want: "Bob"},
		{name: "last element", body: `{{ last(names) }}`, want: "Bob"},
		{name: "first element field", body: `{{ first(fields).name }}`, want: "Email"},
		{name: "last element field", body: `{{ last(fields).name }}`, want: "ID"},
		{name: "contains a member", body: `{{ if contains(names, "Anna") }}yes{{ else }}no{{ end }}`, want: "yes"},
		{name: "contains a missing member", body: `{{ if contains(names, "Lily") }}yes{{ else }}no{{ end }}`, want: "no"},
		{name: "contains behaves as a substring check", body: `{{ if contains("repository", "posit") }}yes{{ else }}no{{ end }}`, want: "yes"},
		{name: "sort records by string field", body: `{{ range index, field := sortBy(fields, "name") }}{{ field.name }} {{ end }}`, want: "Email ID "},
		{name: "sort records by number field", body: `{{ range index, field := sortBy(fields, "index") }}{{ field.name }} {{ end }}`, want: "ID Email "},
		{name: "sort keeps equal elements in order", body: `{{ range index, record := sortBy(records, "word") }}{{ record.word }}{{ end }}`, want: "aab"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderWithVars(t, test.body, vars); got != test.want {
				t.Errorf("%s = %q, want %q", test.body, got, test.want)
			}
		})
	}
}

func TestListHelpersReportInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "join a non slice", body: `{{ join(42, ", ") }}`},
		{name: "join a non string separator", body: `{{ join(slice("Bob"), 42) }}`},
		{name: "first of an empty list", body: `{{ first(slice()) }}`},
		{name: "last of an empty list", body: `{{ last(slice()) }}`},
		{name: "first of a non slice", body: `{{ first(42) }}`},
		{name: "last of a non slice", body: `{{ last(42) }}`},
		{name: "contains without a list or a string", body: `{{ contains(42, 42) }}`},
		{name: "contains with too few arguments", body: `{{ contains(slice("a")) }}`},
		{name: "sortBy a missing field", body: `{{ sortBy(slice(map("name", "x")), "nope") }}`},
		{name: "sortBy a non slice", body: `{{ sortBy(1, "name") }}`},
		{name: "sortBy with a non string field", body: `{{ sortBy(slice(map("name", "x")), 42) }}`},
		{name: "sortBy with too few arguments", body: `{{ sortBy(slice(map("name", "x"))) }}`},
		{name: "randomChoice of an empty list", body: `{{ randomChoice(slice()) }}`},
		{name: "randomChoice of a non slice", body: `{{ randomChoice(1) }}`},
		{name: "uuid with arguments", body: `{{ uuid(1) }}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := renderError(t, test.body); err == nil {
				t.Errorf("%s: expected an error, got none", test.name)
			}
		})
	}
}
