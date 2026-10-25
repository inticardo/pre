package helpers

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/CloudyKit/jet/v6"
)

// registerListHelpers binds the collection helpers to the provided Jet set.
//
// They are implemented with jet.Func instead of plain Go functions because the
// lists defined in the YAML frontmatter reach the template as []interface{}
// values, so signatures typed with concrete slices would never match. Working
// with reflect keeps them usable with every kind of list the frontmatter can
// define.
func registerListHelpers(set *jet.Set) {
	set.AddGlobalFunc("join", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("join", 2, 2)

		list, ok := sequence(args.Get(0))
		if !ok {
			args.Panicf("join(): the first argument must be a slice or an array")
		}

		separator, ok := stringArg(args, 1, "join")
		if !ok {
			args.Panicf("join(): the second argument must be a string")
		}

		var builder strings.Builder
		for i := 0; i < list.Len(); i++ {
			if i > 0 {
				builder.WriteString(separator)
			}
			builder.WriteString(elementToString(list.Index(i)))
		}

		return reflect.ValueOf(builder.String())
	})

	set.AddGlobalFunc("first", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("first", 1, 1)

		list, ok := sequence(args.Get(0))
		if !ok {
			args.Panicf("first(): the argument must be a slice or an array")
		}
		if list.Len() == 0 {
			args.Panicf("first(): the list is empty")
		}

		return list.Index(0)
	})

	set.AddGlobalFunc("last", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("last", 1, 1)

		list, ok := sequence(args.Get(0))
		if !ok {
			args.Panicf("last(): the argument must be a slice or an array")
		}
		if list.Len() == 0 {
			args.Panicf("last(): the list is empty")
		}

		return list.Index(list.Len() - 1)
	})

	set.AddGlobalFunc("contains", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("contains", 2, 2)

		haystack := unwrap(args.Get(0))
		needle := unwrap(args.Get(1))
		if !needle.IsValid() {
			args.Panicf("contains(): the second argument must be a valid value")
		}

		if haystack.Kind() == reflect.String {
			return reflect.ValueOf(strings.Contains(haystack.String(), elementToString(needle)))
		}

		list, ok := sequence(args.Get(0))
		if !ok {
			args.Panicf("contains(): the first argument must be a string, a slice or an array")
		}

		for i := 0; i < list.Len(); i++ {
			if same(list.Index(i), needle) {
				return reflect.ValueOf(true)
			}
		}

		return reflect.ValueOf(false)
	})

	set.AddGlobalFunc("sortBy", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("sortBy", 2, 2)

		list, ok := sequence(args.Get(0))
		if !ok {
			args.Panicf("sortBy(): the first argument must be a slice or an array")
		}

		field, ok := stringArg(args, 1, "sortBy")
		if !ok {
			args.Panicf("sortBy(): the second argument must be a string")
		}

		// The original slice is never modified: the helper returns a new
		// []interface{} with the elements reordered.
		elements := make([]any, list.Len())
		keys := make([]reflect.Value, list.Len())
		keyKind := reflect.Invalid

		for i := 0; i < list.Len(); i++ {
			elements[i] = list.Index(i).Interface()

			keys[i] = unwrap(fieldOf(list.Index(i), field))
			if !keys[i].IsValid() {
				args.Panicf("sortBy(): the element %d has no field %q", i, field)
			}

			kind := keys[i].Kind()
			if kind != reflect.String && !isNumberKind(kind) {
				args.Panicf("sortBy(): the field %q must be a string or a number", field)
			}

			if i == 0 {
				keyKind = kind
			} else if !compatibleKeyKinds(keyKind, kind) {
				args.Panicf("sortBy(): the field %q has inconsistent types across the elements", field)
			}
			keyKind = keyKindOf(keyKind, kind)
		}

		if len(elements) == 0 {
			return reflect.ValueOf(elements)
		}

		// sort.SliceStable keeps the relative order of elements with equal
		// keys, which makes the output deterministic for frontmatter data.
		sort.SliceStable(elements, func(i, j int) bool {
			if keyKind == reflect.String {
				return keys[i].String() < keys[j].String()
			}
			return numberValue(keys[i]) < numberValue(keys[j])
		})

		return reflect.ValueOf(elements)
	})
}

// sequence validates that the value is a slice or an array (unwrapping any
// interface or pointer indirection) and returns the unwrapped value.
func sequence(value reflect.Value) (reflect.Value, bool) {
	value = unwrap(value)
	if !value.IsValid() {
		return value, false
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return value, true
	}

	return reflect.Value{}, false
}

// unwrap removes interface and pointer indirections until it reaches a
// concrete value.
func unwrap(value reflect.Value) reflect.Value {
	for value.IsValid() {
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if value.IsNil() {
				return value
			}
			value = value.Elem()
		default:
			return value
		}
	}

	return value
}

// stringArg reads the argument at the given position as a string.
func stringArg(args jet.Arguments, position int, helperName string) (string, bool) {
	value := unwrap(args.Get(position))
	if value.Kind() != reflect.String {
		return "", false
	}

	return value.String(), true
}

// elementToString renders a single list element as a string: strings are used
// verbatim and any other kind is formatted with its default representation.
func elementToString(value reflect.Value) string {
	value = unwrap(value)
	if value.Kind() == reflect.String {
		return value.String()
	}

	return fmt.Sprint(value.Interface())
}

// same reports whether two list elements are equal, unwrapping any
// indirections first so that interface wrapped values compare naturally.
func same(a, b reflect.Value) bool {
	return reflect.DeepEqual(unwrap(a).Interface(), unwrap(b).Interface())
}

// fieldOf reads the given field (a map key or a struct field) from a list
// element. It returns an invalid value when the element has no such field.
func fieldOf(element reflect.Value, field string) reflect.Value {
	value := unwrap(element)
	if !value.IsValid() {
		return value
	}

	switch value.Kind() {
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return reflect.Value{}
		}
		return value.MapIndex(reflect.ValueOf(field))
	case reflect.Struct:
		found := value.FieldByName(field)
		if !found.IsValid() {
			return reflect.Value{}
		}
		return found
	}

	return reflect.Value{}
}

// isNumberKind reports whether the kind is any signed, unsigned or floating
// point integer kind.
func isNumberKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}

	return false
}

// compatibleKeyKinds reports whether a sort key of one kind may be compared
// with a sort key of another kind: strings only compare with strings, and
// every numeric kind compares with every other numeric kind.
func compatibleKeyKinds(have, new reflect.Kind) bool {
	if have == new {
		return true
	}

	return have != reflect.String && new != reflect.String && isNumberKind(have) && isNumberKind(new)
}

// keyKindOf normalizes two compatible key kinds: any mix of numeric kinds is
// treated as Float64, because string and number keys can never mix.
func keyKindOf(have, kind reflect.Kind) reflect.Kind {
	if kind == reflect.String || have == reflect.String {
		return reflect.String
	}

	return reflect.Float64
}

// numberValue converts a numeric sort key to a float for comparison.
func numberValue(value reflect.Value) float64 {
	value = unwrap(value)

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	}

	return 0
}
