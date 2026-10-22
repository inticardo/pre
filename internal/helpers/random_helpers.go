package helpers

import (
	"math/rand/v2"
	"reflect"
	"strings"

	"github.com/CloudyKit/jet/v6"
)

const (
	alphaCharset      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphanumericChars = alphaCharset + "0123456789"
)

// registerRandomHelpers binds the random helpers to the provided Jet set.
//
// The helpers are implemented with jet.Func instead of plain Go functions so
// that the numeric arguments can be coerced: a template may pass either an
// integer or a float without breaking the call.
func registerRandomHelpers(set *jet.Set) {
	set.AddGlobalFunc("randomInt", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("randomInt", 2, 2)

		min, max := toInt(args, 0), toInt(args, 1)
		if min > max {
			args.Panicf("randomInt(): the minimum must not be greater than the maximum")
		}

		return reflect.ValueOf(min + rand.IntN(max-min+1))
	})

	set.AddGlobalFunc("randomFloat", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("randomFloat", 2, 2)

		min, max := toFloat(args, 0), toFloat(args, 1)
		if min > max {
			args.Panicf("randomFloat(): the minimum must not be greater than the maximum")
		}

		return reflect.ValueOf(min + rand.Float64()*(max-min))
	})

	set.AddGlobalFunc("randomString", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("randomString", 1, 1)

		return reflect.ValueOf(randomText(toInt(args, 0), alphanumericChars))
	})

	set.AddGlobalFunc("randomAlpha", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("randomAlpha", 1, 1)

		return reflect.ValueOf(randomText(toInt(args, 0), alphaCharset))
	})
}

// randomText returns a random string of the given length built with the
// given character set.
func randomText(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		builder.WriteByte(charset[rand.IntN(len(charset))])
	}

	return builder.String()
}

// toInt reads the argument at the given position as an integer.
func toInt(args jet.Arguments, position int) int {
	value := args.Get(position)
	if !value.IsValid() {
		args.Panicf("the argument %d must be a number", position+1)
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(value.Uint())
	case reflect.Float32, reflect.Float64:
		return int(value.Float())
	}

	args.Panicf("the argument %d must be a number", position+1)

	return 0
}

// toFloat reads the argument at the given position as a float.
func toFloat(args jet.Arguments, position int) float64 {
	value := args.Get(position)
	if !value.IsValid() {
		args.Panicf("the argument %d must be a number", position+1)
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	}

	args.Panicf("the argument %d must be a number", position+1)

	return 0
}
