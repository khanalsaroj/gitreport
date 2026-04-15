package timeutil

import (
	"fmt"
	"time"
)

// Resolve returns a time.Time representing the start of the desired range.
// Exactly one of week, days, or month must be non-zero.
func Resolve(week, days, month int) (time.Time, error) {
	now := time.Now()

	switch {
	case week > 0:
		return now.AddDate(0, 0, -7*week), nil
	case days > 0:
		return now.AddDate(0, 0, -days), nil
	case month > 0:
		return now.AddDate(0, -month, 0), nil
	default:
		return time.Time{}, fmt.Errorf("no time range specified")
	}
}

// FormatGit formats a time.Time for use with git --since flag.
func FormatGit(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}
