package stringers

import (
	"fmt"
	"time"
)

var _ fmt.Stringer = Elipsis{}

type Elipsis struct {
	Stringer fmt.Stringer
	Limit    int // todo: limit by rendered size, not character count (this exists somewhere...)
}

func (e Elipsis) String() string {
	s := e.Stringer.String()
	if len(s) > e.Limit {
		s = s[:e.Limit] + "..."
	}
	return s
}

var _ fmt.Stringer = Func(func() string { return "" })

type Func func() string

func (fs Func) String() string {
	return fs()
}

const (
	day = 24 * time.Hour
	// justNowSeconds is how recent a time has to be to read as "Just now" rather than a count.
	justNowSeconds = 3
)

// KindRelativeTime presents how long ago a time is compared to now for a human
type KindRelativeTime struct {
	T time.Time
}

func (r KindRelativeTime) String() string {
	d := time.Since(r.T)
	switch {
	case d > day:
		days := d / day
		if days == 1 {
			return "Yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case d > time.Hour:
		hours := d / time.Hour
		if hours == 1 {
			return fmt.Sprintf("%d hour ago", hours)
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d > time.Minute:
		minutes := d / time.Minute
		if minutes == 1 {
			return fmt.Sprintf("%d minute ago", minutes)
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	seconds := d / time.Second
	if seconds < justNowSeconds {
		return "Just now"
	}
	return fmt.Sprintf("%d seconds ago", seconds)
}
