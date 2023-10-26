//go:build !go1.21

package clocks

import (
	"context"
	"time"
)

func (c defaultClock) ContextWithDeadlineCause(ctx context.Context, t time.Time, cause error) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, t)
}

func (c defaultClock) ContextWithTimeoutCause(ctx context.Context, d time.Duration, cause error) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}
