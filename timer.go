package clocks

import "time"

// StopTimer exposes a `Stop()` method for an equivalent object to time.Timer
// (in the defaultClock case, it may be an actual time.Timer).
type StopTimer interface {
	// Stop attempts to prevent a timer from firing, returning true if it
	// succeeds in preventing the timer from firing, and false if it
	// already fired.
	Stop() bool
}

// Timer exposes methods for an equivalent object to time.Timer
// (in the defaultClock case, it may be an actual time.Timer)
type Timer interface {
	StopTimer
	// Ch returns the channel for the timer
	// For the fake clock's tracking to work, one must always dereference
	// the channel returned by this method directly, as it relies on a
	// finalizer to figure out when a listening goroutine woke up.
	Ch() *<-chan time.Time
}

type defaultTimer struct {
	*time.Timer
}

func (d *defaultTimer) Ch() *<-chan time.Time {
	return &d.C
}
