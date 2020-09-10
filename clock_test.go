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
}
