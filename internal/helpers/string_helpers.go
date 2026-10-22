package helpers

import (
	"strings"
	"unicode"
)

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
