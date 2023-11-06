//go:build go1.21

package fake

import (
	"context"
	"time"
)

// ContextWithDeadlineCause behaves like context.WithDeadlineCause, but it
// uses the clock to determine the when the deadline has expired. Cause is
// ignored in Go 1.20 and earlier.
func (f *Clock) ContextWithDeadlineCause(ctx context.Context, t time.Time, cause error) (context.Context, context.CancelFunc) {
	cctx, cancelCause := context.WithCancelCause(ctx)
	dctx := &deadlineContext{
		Context:  cctx,
		deadline: t,
	}
	dur := f.Until(t)
	if dur <= 0 {
		dctx.timedOut.Store(true)
		cancelCause(cause)
		return dctx, func() {}
	}
	stop := f.AfterFunc(dur, func() {
		if cctx.Err() == nil {
			dctx.timedOut.Store(true)
		}
		cancelCause(cause)
	})
	cancel := func() {
		cancelCause(context.Canceled)
		stop.Stop()
	}
	return dctx, cancel
}
