package helpers

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

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
