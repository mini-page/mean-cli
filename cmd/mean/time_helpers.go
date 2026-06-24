package main

import "time"

// These thin wrappers exist so tests can override time.Now easily.
func timeNow() time.Time          { return time.Now() }
func timeDate(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }
