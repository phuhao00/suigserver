package utils

import "time"

// GetCurrentTimestampMS returns the current Unix timestamp in milliseconds.
func GetCurrentTimestampMS() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GetCurrentTimestampS returns the current Unix timestamp in seconds.
func GetCurrentTimestampS() int64 {
	return time.Now().Unix()
}

// FormatTimeRFC3339 formats a time.Time object into RFC3339 string format.
func FormatTimeRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTimeRFC3339 parses an RFC3339 string into a time.Time object.
func ParseTimeRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// TODO: Add more time utility functions as needed, for example:
// - Functions to calculate durations in specific units.
// - Functions to get start/end of day/week/month.
// - Timezone conversion helpers if dealing with multiple timezones.
