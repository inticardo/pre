// Package helpers provides the custom template helper functions available to
// Jet templates processed by pre.
//
// Helpers are registered as globals on the Jet set, which means they take
// precedence over the built-in functions shipped with Jet.
package helpers

import "github.com/CloudyKit/jet/v6"

// RegisterHelpers binds all custom helper functions to the provided Jet set.
//
// The registrations are grouped by domain, with one registerXHelpers function
// per helper file: string, list, random and time helpers.
func RegisterHelpers(set *jet.Set) {
	registerStringHelpers(set)
	registerListHelpers(set)
	registerRandomHelpers(set)
	registerTimeHelpers(set)
}
