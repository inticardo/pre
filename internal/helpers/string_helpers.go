package helpers

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/CloudyKit/jet/v6"
)

// registerStringHelpers binds all string helpers to the provided Jet set.
//
// The registered functions are plain Go functions, so they can be called from
// a template using the usual syntax: {{ snakeCase("UserRepository") }}. Only
// padStart and padEnd are jet.Func values, because they take an optional
// padding argument and accept both strings and numbers.
func registerStringHelpers(set *jet.Set) {
	set.AddGlobal("lower", lower)
	set.AddGlobal("upper", upper)
	set.AddGlobal("trimSpace", trimSpace)
	set.AddGlobal("replace", replace)
	set.AddGlobal("hasPrefix", hasPrefix)
	set.AddGlobal("hasSuffix", hasSuffix)
	set.AddGlobal("trimPrefix", trimPrefix)
	set.AddGlobal("trimSuffix", trimSuffix)
	set.AddGlobal("indent", indent)
	set.AddGlobal("default", defaultValue)
	set.AddGlobalFunc("padStart", padStart)
	set.AddGlobalFunc("padEnd", padEnd)
	set.AddGlobal("snakeCase", snakeCase)
	set.AddGlobal("camelCase", camelCase)
	set.AddGlobal("pascalCase", pascalCase)
	set.AddGlobal("kebabCase", kebabCase)
	set.AddGlobal("titleCase", titleCase)
}

// lower returns the string with all Unicode letters mapped to their lower case.
func lower(in string) string {
	return strings.ToLower(in)
}

// upper returns the string with all Unicode letters mapped to their upper case.
func upper(in string) string {
	return strings.ToUpper(in)
}

// trimSpace returns the string with all leading and trailing white space removed.
func trimSpace(in string) string {
	return strings.TrimSpace(in)
}

// replace returns a copy of the string with the first n occurrences of old
// replaced by new. It mirrors the signature of the standard library function.
func replace(in, old, new string, n int) string {
	return strings.Replace(in, old, new, n)
}

// hasPrefix reports whether the string begins with the given prefix.
func hasPrefix(in, prefix string) bool {
	return strings.HasPrefix(in, prefix)
}

// hasSuffix reports whether the string ends with the given suffix.
func hasSuffix(in, suffix string) bool {
	return strings.HasSuffix(in, suffix)
}

// snakeCase converts the string to snake_case: "UserRepository" -> "user_repository".
func snakeCase(in string) string {
	return joinLower(in, "_")
}

// kebabCase converts the string to kebab-case: "UserRepository" -> "user-repository".
func kebabCase(in string) string {
	return joinLower(in, "-")
}

// camelCase converts the string to camelCase: "user_repository" -> "userRepository".
func camelCase(in string) string {
	return joinCapitalized(in, "", false)
}

// pascalCase converts the string to PascalCase: "user_repository" -> "UserRepository".
func pascalCase(in string) string {
	return joinCapitalized(in, "", true)
}

// titleCase converts the string to Title Case: "user_repository" -> "User Repository".
func titleCase(in string) string {
	return joinCapitalized(in, " ", true)
}

// trimPrefix returns the string without the given leading prefix, when
// present: "tbl_user" -> "user". When the prefix is not there the string is
// returned unchanged.
func trimPrefix(in, prefix string) string {
	return strings.TrimPrefix(in, prefix)
}

// trimSuffix returns the string without the given trailing suffix, when
// present: "user_old" -> "user". When the suffix is not there the string is
// returned unchanged.
func trimSuffix(in, suffix string) string {
	return strings.TrimSuffix(in, suffix)
}

// indent prefixes every line of the string with the given number of spaces,
// including the first line and the last one even when it has no trailing
// newline. It is meant to indent whole blocks rendered through include, block
// or exec calls.
func indent(width int, in string) string {
	if width <= 0 || in == "" {
		return in
	}

	prefix := strings.Repeat(" ", width)

	return prefix + strings.ReplaceAll(in, "\n", "\n"+prefix)
}

// defaultValue returns the fallback value when the given value is empty, nil
// or its type zero value: strings that are "", numbers that are 0, nil and
// empty lists and maps, and nil pointers. Otherwise it returns the value
// itself. It is registered as "default".
func defaultValue(value, fallback any) any {
	target := reflect.ValueOf(value)
	if !target.IsValid() || target.IsZero() || isEmpty(target) {
		return fallback
	}

	return value
}

// isEmpty reports whether the given value is a nil or empty collection: a nil
// count alone is not enough, because a zero length list or map is already
// empty even when it is allocated.
func isEmpty(target reflect.Value) bool {
	switch target.Kind() {
	case reflect.Slice, reflect.Map:
		return target.Len() == 0
	}

	return false
}

// padStart pads the string on the left with the padding string until it
// reaches the given width counted in runes: padStart("42", 6, "0") renders
// "000042". The padding argument is optional and defaults to a space, and
// strings that are already wider are returned unchanged.
var padStart jet.Func = func(args jet.Arguments) reflect.Value {
	return pad("padStart", args, true)
}

// padEnd pads the string on the right with the padding string until it
// reaches the given width counted in runes: padEnd("ab", 5) renders "ab   ".
// It shares the padding and optional argument behavior with padStart.
var padEnd jet.Func = func(args jet.Arguments) reflect.Value {
	return pad("padEnd", args, false)
}

// pad implements the common behavior of the padStart and padEnd helpers.
func pad(name string, args jet.Arguments, onLeft bool) reflect.Value {
	args.RequireNumOfArguments(name, 2, 3)

	in := unwrap(args.Get(0))
	if !in.IsValid() || in.Kind() != reflect.String && !isNumberKind(in.Kind()) {
		args.Panicf("%s(): the first argument must be a string or a number", name)
	}
	value := elementToString(args.Get(0))

	width := toInt(args, 1)
	if width < 0 {
		args.Panicf("%s(): the width must not be negative", name)
	}

	padding := " "
	if args.NumOfArguments() == 3 {
		preset, ok := stringArg(args, 2, name)
		if !ok {
			args.Panicf("%s(): the third argument must be a string", name)
		}
		padding = preset
	}

	runes := []rune(value)
	if len(runes) >= width {
		return reflect.ValueOf(value)
	}

	if padding == "" {
		args.Panicf("%s(): the padding must not be empty", name)
	}

	filler := []rune(padding)
	filled := make([]rune, 0, width-len(runes))
	for len(filled) < width-len(runes) {
		filled = append(filled, filler...)
	}
	filled = filled[:width-len(runes)]

	if onLeft {
		return reflect.ValueOf(string(filled) + value)
	}

	return reflect.ValueOf(value + string(filled))
}

// joinLower splits the input into words, lower cases them and joins them with
// the given separator.
func joinLower(in, separator string) string {
	words := splitWords(in)
	parts := make([]string, 0, len(words))

	for _, word := range words {
		parts = append(parts, strings.ToLower(word))
	}

	return strings.Join(parts, separator)
}

// joinCapitalized splits the input into words and joins them with the given
// separator, upper casing the first letter of every word.
//
// When capitalizeAll is false the first word is kept lower cased, which is what
// distinguishes camelCase from PascalCase.
func joinCapitalized(in, separator string, capitalizeAll bool) string {
	words := splitWords(in)
	parts := make([]string, 0, len(words))

	for i, word := range words {
		if i == 0 && !capitalizeAll {
			parts = append(parts, strings.ToLower(word))
			continue
		}
		parts = append(parts, capitalize(word))
	}

	return strings.Join(parts, separator)
}

// splitWords splits the input into its constituent words, understanding
// underscores, dashes, dots, slashes, spaces and camel case boundaries.
//
// "user_repository", "UserRepository" and "user-repository" all yield the
// words ["user", "repository"], while "HTTPServer" yields ["HTTP", "Server"].
func splitWords(in string) []string {
	runes := []rune(in)
	words := make([]string, 0, len(runes))
	current := make([]rune, 0, len(runes))

	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch {
		case isSeparator(r):
			flush()
		case unicode.IsUpper(r):
			// A new word starts when the previous rune was not upper case, or
			// when the previous rune was upper case but the current one starts
			// a new word (the last rune of an acronym followed by a word).
			if len(current) > 0 {
				previous := current[len(current)-1]
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if !unicode.IsUpper(previous) || nextIsLower {
					flush()
				}
			}
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()

	return words
}

// isSeparator reports whether the rune separates two words.
func isSeparator(r rune) bool {
	switch r {
	case '_', '-', '.', '/', '\\', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// capitalize upper cases the first letter of the word and lower cases the rest.
func capitalize(word string) string {
	if word == "" {
		return word
	}

	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes[0]) + strings.ToLower(string(runes[1:]))
}
