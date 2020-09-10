package offset

import (
	"context"
	"testing"
	"time"

	"github.com/vimeo/go-clocks/fake"
)

func TestOffsetClock(t *testing.T) {
	base := time.Now()
	inner := fake.NewClock(base)
	const o = time.Minute + 3*time.Second
	c := NewOffsetClock(inner, o)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if n := c.Now(); !base.Add(o).Equal(n) {
		t.Errorf("unexpected time from Now(): %s; expected %s",
			n, base.Add(o))
	}

	if d := c.Until(base.Add(time.Hour)); d != (time.Hour + o) {
		t.Errorf("unexpected value for c.Until(%s); got %s; expected %s",
			base.Add(time.Hour), d, (time.Hour + o))
	}

	{
		expectedWakeInner := base.Add(time.Hour)
		ch := make(chan bool)
		go func() {
			ch <- c.SleepFor(ctx, time.Hour)
		}()
		inner.AwaitSleepers(1)

		if sl := inner.Sleepers(); len(sl) != 1 || !sl[0].Equal(expectedWakeInner) {
			t.Errorf("unexpected sleepers: %v; expected 1 with %s", sl, expectedWakeInner)
		}

		select {
		case v := <-ch:
			t.Fatalf("SleepFor exited prematurely with value %t", v)
		default:
		}

		if awoken := inner.Advance(time.Hour); awoken != 1 {
			t.Errorf("unexpected number of awoken waiters: %d; expected 1", awoken)
		}

		if v := <-ch; !v {
			t.Errorf("unexpected return value from SleepFor; %t; expected true", v)
		}
	}
	{
		expectedWakeInner := base.Add(2 * time.Hour)
		ch := make(chan bool)
		go func() {
			ch <- c.SleepUntil(ctx, expectedWakeInner.Add(-o))
		}()
		inner.AwaitSleepers(1)

		if sl := inner.Sleepers(); len(sl) != 1 || !sl[0].Equal(expectedWakeInner) {
			t.Errorf("unexpected sleepers: %v; expected 1 with %s", sl, expectedWakeInner)
		}

		select {
		case v := <-ch:
			t.Fatalf("SleepFor exited prematurely with value %t", v)
		default:
		}

		if awoken := inner.Advance(time.Hour); awoken != 1 {
			t.Errorf("unexpected number of awoken waiters: %d; expected 1", awoken)
		}

		if v := <-ch; !v {
			t.Errorf("unexpected return value from SleepUntil; %t; expected true", v)
		}
	}
}
