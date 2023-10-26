//go:build go1.21

package clocks

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultClockContext121(t *testing.T) {
	c := DefaultClock()

	t.Run("ContextWithDeadlineCause", func(t *testing.T) {
		base := c.Now()

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), base.Add(time.Millisecond), errors.New("test"))
		t.Cleanup(cancel)

		if v := c.SleepUntil(ctx, base.Add(time.Second)); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
			}
		}
	})

	t.Run("ContextWithDeadlineCauseCanceled", func(t *testing.T) {
		base := c.Now()

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), base.Add(500*time.Millisecond), errors.New("test"))

		cancel()

		if v := c.SleepUntil(ctx, base.Add(time.Second)); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.Canceled {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
			}
		}
	})

	t.Run("ContextWithTimeoutCause", func(t *testing.T) {
		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), time.Millisecond, errors.New("test"))
		t.Cleanup(cancel)

		if v := c.SleepFor(ctx, time.Second); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
			}
		}
	})

	t.Run("ContextWithTimeoutCauseCanceled", func(t *testing.T) {
		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), 500*time.Millisecond, errors.New("test"))

		cancel()

		if v := c.SleepFor(ctx, time.Second); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.Canceled {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
			}
		}
	})
}
