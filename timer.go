package clocks

// StopTimer exposes a `Stop()` method for an equivalent object to time.Timer
// (in the defaultClock case, it may be an actual time.Timer)
type StopTimer interface {
	// Stop attempts to prevent a timer from firing, returning true if it
	// succeeds in preventing the timer from firing, and false if it
	// already fired.
	Stop() bool
}
