package fake

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	clocks "github.com/vimeo/go-clocks"
)

// Clock implements the clocks.Clock interface, with helpful primitives for
// testing and skipping through timestamps without having to actually sleep in
// the test.
type Clock struct {
	mu      sync.Mutex
	current time.Time
	// sleepers contains a map from a channel on which that
	// sleeper is sleeping to a target-time. When time is advanced past a
	// sleeper's wakeup point, its channel should be closed and it should
	// be removed from the map.
	sleepers map[chan<- struct{}]time.Time
	// cbs contains a map from a *stopTimer containing the callback
	// function to the wakeup time. (protected by mu).
	cbs map[*stopTimer]time.Time

	// cbsWG tracks callback goroutines configured from AfterFunc (no mutex
	// protection necessary).
	cbsWG sync.WaitGroup

	// cond is broadcasted() upon any sleep or wakeup event (mutations to
	// sleepers or cbs).
	cond sync.Cond

	// counter tracking the number of wakeups (protected by mu).
	wakeups int

	// callbackExecs tracking the number of callback executions (protected by mu).
	callbackExecs int

	// counter tracking the number of canceled sleeps (protected by mu).
	sleepAborts int

	// counter tracking the number of canceled timers (including AfterFuncs) (protected by mu).
	timerAborts int

	// counter tracking the number of sleepers who have ever gone to sleep
	// (protected by mu).
	sleepersAggregate int

	// counter tracking the number of callbacks that have ever been
	// registered (via AfterFunc) (protected by mu).
	callbacksAggregate int
}

var _ clocks.Clock = (*Clock)(nil)

// NewClock returns an initialized Clock instance.
func NewClock(initialTime time.Time) *Clock {
	fc := Clock{
		current:  initialTime,
		sleepers: map[chan<- struct{}]time.Time{},
		cbs:      map[*stopTimer]time.Time{},
		cond:     sync.Cond{},
	}
	fc.cond.L = &fc.mu
	return &fc
}

// returns the number of sleepers awoken.
func (f *Clock) setClockLocked(t time.Time, cbRunningWG *sync.WaitGroup) int {
	awoken := 0
	for ch, target := range f.sleepers {
		if target.Sub(t) <= 0 {
			close(ch)
			delete(f.sleepers, ch)
			awoken++
		}
	}
	cbsRun := 0
	for s, target := range f.cbs {
		if target.Sub(t) <= 0 {
			cbRunningWG.Add(1)
			f.cbsWG.Add(1)
			go func(st *stopTimer) {
				defer f.cbsWG.Done()
				cbRunningWG.Done()
				st.f()
			}(s)
			delete(f.cbs, s)
			cbsRun++
		}
	}
	f.wakeups += awoken
	f.callbackExecs += cbsRun
	f.current = t
	f.cond.Broadcast()
	return awoken + cbsRun
}

// SetClock skips the FakeClock to the specified time (forward or backwards) The
// goroutines running newly-spawned functions scheduled with AfterFunc are
// guaranteed to have scheduled by the time this function returns.  It returns
// the number of sleepers awoken.
func (f *Clock) SetClock(t time.Time) int {
	cbsWG := sync.WaitGroup{}
	// Wait for callbacks to schedule before returning (but after the mutex
	// is unlocked)
	defer cbsWG.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.setClockLocked(t, &cbsWG)
}

// Advance skips the FakeClock forward by the specified duration (backwards if
// negative) The goroutines running newly-spawned functions scheduled with
// AfterFunc are guaranteed to have scheduled by the time this function returns.
// It returns number of sleepers awoken.
func (f *Clock) Advance(dur time.Duration) int {
	cbsWG := sync.WaitGroup{}
	// Wait for callbacks to schedule before returning (but after the mutex
	// is unlocked)
	defer cbsWG.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.current.Add(dur)
	return f.setClockLocked(t, &cbsWG)
}

// NumSleepers returns the number of goroutines waiting in SleepFor and SleepUntil
// calls.
func (f *Clock) NumSleepers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sleepers)
}

// NumAggSleepers returns the number of goroutines who have ever slept under
// SleepFor and SleepUntil calls.
func (f *Clock) NumAggSleepers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sleepersAggregate
}

// NumSleepAborts returns the number of calls to SleepFor and SleepUntil which
// have ended prematurely due to canceled contexts.
func (f *Clock) NumSleepAborts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sleepAborts
}

// Sleepers returns the wake-times for goroutines waiting in SleepFor and
// SleepUntil calls.
func (f *Clock) Sleepers() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]time.Time, 0, len(f.sleepers))
	for _, t := range f.sleepers {
		out = append(out, t)
	}
	return out
}

// RegisteredCallbacks returns the execution-times of registered callbacks.
func (f *Clock) RegisteredCallbacks() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]time.Time, 0, len(f.sleepers))
	for _, t := range f.cbs {
		out = append(out, t)
	}
	return out
}

// AwaitSleepers waits until the number of sleepers equals or exceeds its
// argument.
func (f *Clock) AwaitSleepers(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.sleepers) < n {
		f.cond.Wait()
	}
}

// AwaitAggSleepers waits until the aggregate number of sleepers equals or
// exceeds its argument.
func (f *Clock) AwaitAggSleepers(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.sleepersAggregate < n {
		f.cond.Wait()
	}
}

// AwaitSleepAborts waits until the number of aborted sleepers equals or exceeds
// its argument.
func (f *Clock) AwaitSleepAborts(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.sleepAborts < n {
		f.cond.Wait()
	}
}

// Wakeups returns the number of sleepers that have been awoken (useful for
// verifying that nothing was woken up when advancing time).
func (f *Clock) Wakeups() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wakeups
}

// Now implements Clock.Now(), returning the current time for this FakeClock.
func (f *Clock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current

}

// Until implements Clock.Now(), returning the time-difference between the
// timestamp argument and the current timestamp for the clock.
func (f *Clock) Until(t time.Time) time.Duration {
	return t.Sub(f.Now())
}

func (f *Clock) setAbsoluteWaiter(until time.Time) chan struct{} {
	ch := make(chan struct{})
	f.mu.Lock()
	defer f.mu.Unlock()
	if until.Sub(f.current) <= 0 {
		close(ch)
		return ch
	}
	f.sleepers[ch] = until
	f.sleepersAggregate++

	f.cond.Broadcast()
	return ch
}

func (f *Clock) removeWaiter(ch chan struct{}, abort bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// If the channel is present, and this was an abort, increment the
	// aborts counter.
	if _, ok := f.sleepers[ch]; ok && abort {
		f.sleepAborts++
	}
	delete(f.sleepers, ch)
	f.cond.Broadcast()
}

// SleepUntil blocks until either ctx expires or until arrives.
// Return value is false if context-cancellation/expiry prompted an
// early return.
func (f *Clock) SleepUntil(ctx context.Context, until time.Time) (success bool) {
	ch := f.setAbsoluteWaiter(until)
	defer func() { f.removeWaiter(ch, !success) }()
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

func (f *Clock) setRelativeWaiter(dur time.Duration) chan struct{} {
	ch := make(chan struct{})
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sleepers[ch] = f.current.Add(dur)
	f.sleepersAggregate++
	f.cond.Broadcast()
	return ch
}

// SleepFor is the relative-time equivalent of SleepUntil.
func (f *Clock) SleepFor(ctx context.Context, dur time.Duration) (success bool) {
	if dur <= 0 {
		return true
	}
	ch := f.setRelativeWaiter(dur)
	defer func() { f.removeWaiter(ch, !success) }()
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

type stopTimer struct {
	f func()
	c *Clock
}

func (s *stopTimer) Stop() bool {
	return s.c.removeAfterFunc(s)
}

func (f *Clock) removeAfterFunc(s *stopTimer) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.cbs[s]
	if ok {
		f.timerAborts++
	}
	delete(f.cbs, s)

	f.cond.Broadcast()

	return ok
}

type doaStopTimer struct{}

func (doaStopTimer) Stop() bool { return false }

// AfterFunc runs cb after time duration has "elapsed" (by this clock's
// definition of "elapsed").
func (f *Clock) AfterFunc(d time.Duration, cb func()) clocks.StopTimer {
	s := &stopTimer{f: cb, c: f}
	f.mu.Lock()
	defer f.mu.Unlock()
	defer f.cond.Broadcast()
	f.callbacksAggregate++
	// If the interval is negative, run the callback immediately and return
	// a nop StopTimer that always returns false (since the goroutine has
	// run by the time the function has returned).
	if d <= 0 {
		f.callbackExecs++
		f.cbsWG.Add(1)
		go func() {
			defer f.cbsWG.Done()
			cb()
		}()
		return doaStopTimer{}
	}
	wakeTime := f.current.Add(d)
	f.cbs[s] = wakeTime
	return s
}

// NumCallbackExecs returns the number of registered callbacks that have been
// executed due to time advancement.
func (f *Clock) NumCallbackExecs() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callbackExecs
}

// NumAggCallbacks returns the aggregate number of registered callbacks
// (via AfterFunc).
func (f *Clock) NumAggCallbacks() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callbacksAggregate
}

// NumTimerAborts returns the aggregate number of registered callbacks (and
// timers) that have been canceled.
func (f *Clock) NumTimerAborts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.timerAborts
}

// AwaitAggCallbacks waits until the aggregate number of registered callbacks
// (via AfterFunc) exceeds its argument.
func (f *Clock) AwaitAggCallbacks(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.callbacksAggregate < n {
		f.cond.Wait()
	}
}

// NumRegisteredCallbacks returns the aggregate number of registered callbacks
// (via AfterFunc).
func (f *Clock) NumRegisteredCallbacks() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.cbs)
}

// AwaitRegisteredCallbacks waits until the number of registered callbacks
// (via AfterFunc) exceeds its argument.
func (f *Clock) AwaitRegisteredCallbacks(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.cbs) < n {
		f.cond.Wait()
	}
}

// AwaitTimerAborts waits until the aggregate number of registered callbacks
// (via AfterFunc) exceeds its argument.
func (f *Clock) AwaitTimerAborts(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.timerAborts < n {
		f.cond.Wait()
	}
}

// WaitAfterFuncs blocks until all currently running AfterFunc callbacks
// return.
func (f *Clock) WaitAfterFuncs() {
	f.cbsWG.Wait()
}

type deadlineContext struct {
	context.Context
	timedOut atomic.Bool
	deadline time.Time
}

func (d *deadlineContext) Deadline() (time.Time, bool) {
	return d.deadline, true
}

func (d *deadlineContext) Err() error {
	if d.timedOut.Load() {
		return context.DeadlineExceeded
	}
	return d.Context.Err()
}

// ContextWithDeadline behaves like context.WithDeadline, but it uses the
// clock to determine the when the deadline has expired.
func (c *Clock) ContextWithDeadline(ctx context.Context, t time.Time) (context.Context, context.CancelFunc) {
	return c.ContextWithDeadlineCause(ctx, t, nil)
}

// ContextWithTimeout behaves like context.WithTimeout, but it uses the
// clock to determine the when the timeout has elapsed.
func (c *Clock) ContextWithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return c.ContextWithDeadlineCause(ctx, c.Now().Add(d), nil)
}

// ContextWithTimeoutCause behaves like context.WithTimeoutCause, but it
// uses the clock to determine the when the timeout has elapsed. Cause is
// ignored in Go 1.20 and earlier.
func (c *Clock) ContextWithTimeoutCause(ctx context.Context, d time.Duration, cause error) (context.Context, context.CancelFunc) {
	return c.ContextWithDeadlineCause(ctx, c.Now().Add(d), cause)
}
