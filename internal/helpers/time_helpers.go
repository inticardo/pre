package helpers

import (
	"time"

	"github.com/CloudyKit/jet/v6"
)

// clock is the function the date and now helpers read the current time from.
// It is a package level variable so the tests can pin a deterministic
// instant; production code always reads the real clock through it.
var clock = time.Now

// registerTimeHelpers binds the time helpers to the provided Jet set.
//
// They are plain Go functions with a fixed layout string argument, following
// Go's reference layout ("2006-01-02", "15:04:05", ...).
func registerTimeHelpers(set *jet.Set) {
	set.AddGlobal("date", date)
	set.AddGlobal("now", date)
}

// date formats the current time with the given Go reference layout, so that
// date("2006-01-02") renders the day and date("2006") renders the year.
//
// The helpers are registered under two names: date and now. They render the
// current time, which makes the generated content non reproducible, just like
// the random helpers.
func date(layout string) string {
	return clock().Format(layout)
}
