package fake

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type timerTracker struct {
	// backpointer to the parent clock
	fc *Clock

	timers map[*fakeTimer]time.Time

	mu sync.Mutex
}

type wakeRes struct {
	notified int
	awoken   int
}

// Returns the number of timers that were notified, followed by how many were
// actually awoken (or at least a best-guess).
func (t *timerTracker) wakeup(now time.Time) wakeRes {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := wakeRes{}
	for ft, ttim := range t.timers {
		// use After to reflect <= rather than <.
		if ttim.After(now) {
			continue
		}
		wres := ft.wake(now)
		if wres.wasStopped {
			continue
		}
		if wres.signaled {
			out.awoken++
			out.notified++
		}
		delete(t.timers, ft)
	}
	return out
}

func (t *timerTracker) registerTimer(ft *fakeTimer, wakeTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timers[ft] = wakeTime
}

// returns true if the timer was previously present
func (t *timerTracker) remove(ft *fakeTimer) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, present := t.timers[ft]
	delete(t.timers, ft)
	return present
}

func (t *timerTracker) registeredTimers() []time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]time.Time, 0, len(t.timers))
	for _, t := range t.timers {
		out = append(out, t)
	}
	return out
}

type refCnt struct {
	i atomic.Int32
}

func (r *refCnt) inc() {
	r.i.Add(1)
}

func (r *refCnt) dec() {
	if v := r.i.Add(-1); v < 0 {
		panic("negative refcount")
	}
}

func (r *refCnt) val() int32 {
	return r.i.Load()
}

type fakeTimer struct {
	// backreference to tracker
	tracker *timerTracker

	ch chan time.Time

	fired   atomic.Bool
	stopped atomic.Bool

	ptrExtracted refCnt
}

type ftWakeState struct {
	ptrWasExtracted bool
	wasStopped      bool
	signaled        bool
}

func (f *fakeTimer) wake(now time.Time) ftWakeState {
	stopped := f.stopped.Load()
	f.fired.Store(true)
	if stopped {
		return ftWakeState{
			ptrWasExtracted: false,
			wasStopped:      stopped,
			signaled:        false,
		}
	}

	select {
	case f.ch <- now:
		return ftWakeState{
			ptrWasExtracted: f.ptrExtracted.val() > 0,
			wasStopped:      false,
			signaled:        true,
		}
	default:
		return ftWakeState{
			ptrWasExtracted: f.ptrExtracted.val() > 0,
			wasStopped:      false,
			signaled:        false,
		}
	}
}

// Stop attempts to prevent a timer from firing, returning true if it
// succeeds in preventing the timer from firing, and false if it
// already fired.
func (f *fakeTimer) Stop() bool {
	f.tracker.remove(f)
	f.stopped.Store(true)
	fired := f.fired.Load()

	f.tracker.fc.mu.Lock()
	defer f.tracker.fc.mu.Unlock()
	f.tracker.fc.timerAborts++
	f.tracker.fc.cond.Broadcast()
	return !fired
}

// Ch returns a pointer to the channel for the timer
func (f *fakeTimer) Ch() *<-chan time.Time {
	f.ptrExtracted.inc()

	// Define this callback before chWrapper so it doesn't hold a reference
	// and prevent the finalizer from running
	chFin := func(any) {
		f.ptrExtracted.dec()
		f.tracker.fc.mu.Lock()

		defer f.tracker.fc.mu.Unlock()
		f.tracker.fc.extractedChans--
		f.tracker.fc.cond.Broadcast()
	}

	chWrapper := struct {
		ch <-chan time.Time
	}{
		ch: f.ch,
	}

	runtime.SetFinalizer(&chWrapper, chFin)

	f.tracker.fc.mu.Lock()
	defer f.tracker.fc.mu.Unlock()
	f.tracker.fc.extractedChans++
	f.tracker.fc.extractedChansAggregate++

	f.tracker.fc.cond.Broadcast()
	return &chWrapper.ch
}
