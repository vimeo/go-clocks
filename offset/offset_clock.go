package offset

import (
	"context"
	"time"

	clocks "github.com/vimeo/go-clocks"
)

// Clock wraps another clock, adjusting time intervals by adding a constant
// offset. (useful for simulating clock-skew)
// Note that relative durations are unaffected.
type Clock struct {
	inner  clocks.Clock
	offset time.Duration
}

// Now implements Clock, returning the current time (according to the captive
// clock adjusted by offset).
func (o *Clock) Now() time.Time {
	return o.inner.Now().Add(o.offset)
}

// Until implements Clock, returning the difference between the current time
// and its argument (according to the captive clock adjusted by offset).
func (o *Clock) Until(t time.Time) time.Duration {
	return o.inner.Until(t) + o.offset
}

// SleepUntil blocks until either ctx expires or until arrives.
// Return value is false if context-cancellation/expiry prompted an
// early return.
func (o *Clock) SleepUntil(ctx context.Context, until time.Time) bool {
	return o.inner.SleepUntil(ctx, until.Add(o.offset))
}

// SleepFor is the relative-time equivalent of SleepUntil.
// Return value is false if context-cancellation/expiry prompted an
// early return.
func (o *Clock) SleepFor(ctx context.Context, dur time.Duration) bool {
	// SleepFor is relative, so it doesn't need any adjustment
	return o.inner.SleepFor(ctx, dur)
}

// AfterFunc executes function f after duration d, (delegating to the wrapped
// clock, as the time-argument is relative).
func (o *Clock) AfterFunc(d time.Duration, f func()) clocks.StopTimer {
	// relative time, so nothing to do here, just delegate on down.
	return o.inner.AfterFunc(d, f)
}

// ContextWithDeadline behaves like context.WithDeadline, but it uses the
// clock to determine the when the deadline has expired.
func (o *Clock) ContextWithDeadline(ctx context.Context, t time.Time) (context.Context, context.CancelFunc) {
	return o.inner.ContextWithDeadline(ctx, t.Add(o.offset))
}

// ContextWithTimeout behaves like context.WithTimeout, but it uses the
// clock to determine the when the timeout has elapsed.
func (o *Clock) ContextWithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	// timeout is relative, so it doesn't need any adjustment
	return o.inner.ContextWithTimeout(ctx, d)
}

// NewOffsetClock creates an OffsetClock. offset is added to all absolute times.
func NewOffsetClock(inner clocks.Clock, offset time.Duration) *Clock {
	return &Clock{
		inner:  inner,
		offset: offset,
	}
}
