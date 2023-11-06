package clocks

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestDefaultClock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := DefaultClock()
	base := c.Now()

	// Sleep for a millisecond as some clocks only have ms resolution
	if v := c.SleepFor(ctx, time.Millisecond); !v {
		t.Errorf("unexpected return value: %t; expected true", v)
	}

	if negElapsed := c.Until(base); negElapsed < -time.Hour {
		t.Errorf("c.Until returned a much more negative duration than expected: %s; expected at least %s",
			negElapsed, time.Millisecond)
	} else if negElapsed > -time.Millisecond {
		t.Errorf("c.Until returned a much less negative duration than expected: %s; expected at least %s",
			negElapsed, time.Millisecond)
	}

	// we'll cancel our context to awaken our sleeper.
	wakeUpTime := base.Add(time.Hour * 24)
	ch := make(chan bool)
	go func() {
		ch <- c.SleepUntil(ctx, wakeUpTime)
	}()

	// Give the goroutine a chance to run
	runtime.Gosched()

	cancel()

	if v := <-ch; v {
		t.Errorf("unexpected return value for SleepUntil (and by extension SleepFor): %t; expected false", v)
	}

	{
		afCh := make(chan struct{})
		c.AfterFunc(time.Millisecond, func() { close(afCh) })
		<-afCh
	}
}

func TestDefaultClockContext(t *testing.T) {
	c := DefaultClock()

	t.Run("ContextWithDeadlineExceeded", func(t *testing.T) {
		base := c.Now()

		ctx, cancel := c.ContextWithDeadline(context.Background(), base.Add(time.Millisecond))
		t.Cleanup(cancel)

		if v := c.SleepUntil(ctx, base.Add(time.Second)); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
		}
	})

	t.Run("ContextWithDeadlineNotExceeded", func(t *testing.T) {
		base := c.Now()

		ctx, cancel := c.ContextWithDeadline(context.Background(), base.Add(3*time.Second))
		t.Cleanup(cancel)

		if v := c.SleepUntil(ctx, base.Add(time.Millisecond)); !v {
			t.Errorf("unexpected return value: %t; expected true", v)
		} else {
			if ctx.Err() != nil {
				t.Errorf("unexpected error: %v; expected nil", ctx.Err())
			}
		}
	})

	t.Run("ContextWithTimeoutExceeded", func(t *testing.T) {
		ctx, cancel := c.ContextWithTimeout(context.Background(), time.Millisecond)
		t.Cleanup(cancel)

		if v := c.SleepFor(ctx, time.Second); v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
		}
	})

	t.Run("ContextWithTimeoutNotExceeded", func(t *testing.T) {
		ctx, cancel := c.ContextWithTimeout(context.Background(), 3*time.Second)
		t.Cleanup(cancel)

		if v := c.SleepFor(ctx, time.Millisecond); !v {
			t.Errorf("unexpected return value: %t; expected false", v)
		} else {
			if ctx.Err() != nil {
				t.Errorf("unexpected error: %v; expected nil", ctx.Err())
			}
		}
	})
}
