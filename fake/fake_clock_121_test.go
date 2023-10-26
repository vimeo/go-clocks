//go:build go1.21

package fake

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFakeClockContext121(t *testing.T) {
	t.Run("ContextWithDeadlineCause", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), base.Add(1), errors.New("test"))
		t.Cleanup(cancel)

		c.Advance(1)

		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithDeadlineCanceled", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), base.Add(1), errors.New("test"))
		t.Cleanup(cancel)

		cancel()

		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithDeadlineCausePast", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), base.Add(-1), errors.New("test"))
		t.Cleanup(cancel)

		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithTimeoutCause", func(t *testing.T) {
		c := NewClock(time.Now())
		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), 1, errors.New("test"))
		t.Cleanup(cancel)

		c.Advance(1)

		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithTimeoutCauseCanceled", func(t *testing.T) {
		c := NewClock(time.Now())
		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), 1, errors.New("test"))
		t.Cleanup(cancel)

		cancel()

		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
			}
			if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
				t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})
}
