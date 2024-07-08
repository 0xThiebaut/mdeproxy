package times

import "time"

const Layout = "2006-01-02T15:04:05.9999999Z"

func Min(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
