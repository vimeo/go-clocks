//go:build go1.21

package offset

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vimeo/go-clocks/fake"
)

func TestOffsetClockContext121(t *testing.T) {
	t.Run("ContextWithDeadlineCause", func(t *testing.T) {
		base := time.Now()
		inner := fake.NewClock(base)
		c := NewOffsetClock(inner, time.Hour)

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), inner.Now().Add(time.Hour), errors.New("test"))
		t.Cleanup(cancel)

		awoken := inner.Advance(2 * time.Hour)
		if awoken != 1 {
			t.Errorf("unexpected number of awoken sleepers: %d; expected 1", awoken)
		}

		<-ctx.Done()
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
		}
		if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
			t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
		}
	})

	t.Run("ContextWithDeadlineCauseCanceled", func(t *testing.T) {
		base := time.Now()
		inner := fake.NewClock(base)
		c := NewOffsetClock(inner, time.Hour)

		ctx, cancel := c.ContextWithDeadlineCause(context.Background(), inner.Now().Add(time.Hour), errors.New("test"))

		cancel()

		<-ctx.Done()
		if ctx.Err() != context.Canceled {
			t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
		}
		if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
			t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
		}
	})

	t.Run("ContextWithTimeoutCause", func(t *testing.T) {
		base := time.Now()
		inner := fake.NewClock(base)
		c := NewOffsetClock(inner, time.Hour)

		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), time.Hour, errors.New("test"))
		t.Cleanup(cancel)

		awoken := inner.Advance(time.Hour)
		if awoken != 1 {
			t.Errorf("unexpected number of awoken sleepers: %d; expected 1", awoken)
		}

		<-ctx.Done()
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
		}
		if context.Cause(ctx) == nil || context.Cause(ctx).Error() != "test" {
			t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), "test")
		}
	})

	t.Run("ContextWithTimeoutCauseCanceled", func(t *testing.T) {
		base := time.Now()
		inner := fake.NewClock(base)
		c := NewOffsetClock(inner, time.Hour)

		ctx, cancel := c.ContextWithTimeoutCause(context.Background(), time.Hour, errors.New("test"))

		cancel()

		<-ctx.Done()
		if ctx.Err() != context.Canceled {
			t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.Canceled)
		}
		if context.Cause(ctx) == nil || context.Cause(ctx) != context.Canceled {
			t.Errorf("unexpected cause: %v; expected %v", context.Cause(ctx), context.Canceled)
		}
	})
}
