//go:build go1.21

package offset

import (
	"context"
	"time"
)

// ContextWithDeadlineCause behaves like context.WithDeadlineCause, but it
// uses the clock to determine the when the deadline has expired. Cause is
// ignored in Go 1.20 and earlier.
func (o *Clock) ContextWithDeadlineCause(ctx context.Context, t time.Time, cause error) (context.Context, context.CancelFunc) {
	return o.inner.ContextWithDeadlineCause(ctx, t.Add(o.offset), cause)
}

// ContextWithTimeoutCause behaves like context.WithTimeoutCause, but it
// uses the clock to determine the when the timeout has elapsed. Cause is
// ignored in Go 1.20 and earlier.
func (o *Clock) ContextWithTimeoutCause(ctx context.Context, d time.Duration, cause error) (context.Context, context.CancelFunc) {
	return o.inner.ContextWithTimeoutCause(ctx, d, cause)
}
