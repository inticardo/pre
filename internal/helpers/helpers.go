// Package helpers provides the custom template helper functions available to
// Jet templates processed by pre.
//
// Helpers are registered as globals on the Jet set, which means they take
// precedence over the built-in functions shipped with Jet.
package helpers

import "github.com/CloudyKit/jet/v6"

// RegisterHelpers binds all custom helper functions to the provided Jet set.
//
// The registered functions are plain Go functions, so they can be called from
// a template using the usual syntax: {{ snakeCase("UserRepository") }}.
func RegisterHelpers(set *jet.Set) {
	set.AddGlobal("lower", lower)
	set.AddGlobal("upper", upper)
	set.AddGlobal("trimSpace", trimSpace)
	set.AddGlobal("replace", replace)
	set.AddGlobal("hasPrefix", hasPrefix)
	set.AddGlobal("hasSuffix", hasSuffix)
	set.AddGlobal("snakeCase", snakeCase)
	set.AddGlobal("camelCase", camelCase)
	set.AddGlobal("pascalCase", pascalCase)
	set.AddGlobal("kebabCase", kebabCase)
	set.AddGlobal("titleCase", titleCase)

	registerRandomHelpers(set)
}
