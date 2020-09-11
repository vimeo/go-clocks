package fake

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestFakeClockBasic(t *testing.T) {
	t.Parallel()
	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	// make sure we get the same value a second time
	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}

	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}

	expectedTime = expectedTime.Add(time.Minute)
	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	if zeroDur := fc.Until(expectedTime); zeroDur != 0 {
		t.Errorf("expected zero duration, got %s", zeroDur)
	}

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}

}

func TestFakeClockWithAbsoluteWaiter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	expectedTime = expectedTime.Add(time.Minute)

	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	sleeperWake := baseTime.Add(time.Hour * 2)
	ch := make(chan bool)
	go func() {
		ch <- fc.SleepUntil(ctx, sleeperWake)
	}()

	fc.AwaitSleepers(1)

	if sl := fc.NumSleepers(); sl != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	if as := fc.NumAggSleepers(); as != 1 {
		t.Errorf("unexpected number of aggregate sleepers: %d; expected 1", as)
	}

	// make sure we're still sleeping
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	// verify that our one sleeper is still sleeping
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; ", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	// advance to our wakeup point
	expectedTime = sleeperWake
	if wakers := fc.SetClock(sleeperWake); wakers != 1 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 1", wakers)
	}

	// wait for our sleeper to wake and return (expected true)
	if v := <-ch; !v {
		t.Errorf("sleeper awoke with unexpected return value: %t; expected true", v)
	}

	if !fc.SleepUntil(ctx, baseTime) {
		t.Errorf("attempt to sleep until past time returned false")
	}
	if wu := fc.Wakeups(); wu != 1 {
		t.Errorf("unexpected wakeup-count: %d; expected 1", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}
}

func TestFakeClockWithRelativeWaiter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	expectedTime = expectedTime.Add(time.Minute)

	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	sleeperWake := expectedTime.Add(time.Hour * 2)
	ch := make(chan bool)
	go func() {
		ch <- fc.SleepFor(ctx, time.Hour*2)
	}()

	fc.AwaitAggSleepers(1)

	if sl := fc.NumSleepers(); sl != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	if as := fc.NumAggSleepers(); as != 1 {
		t.Errorf("unexpected number of aggregate sleepers: %d; expected 1", as)
	}

	// make sure we're still sleeping
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	// verify that our one sleeper is still sleeping
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; ", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	// advance to our wakeup point
	expectedTime = sleeperWake
	if wakers := fc.SetClock(sleeperWake); wakers != 1 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 1", wakers)
	}

	// wait for our sleeper to wake and return (expected true)
	if v := <-ch; !v {
		t.Errorf("sleeper awoke with unexpected return value: %t; expected true", v)
	}

	if !fc.SleepFor(ctx, -1*time.Minute) {
		t.Errorf("attempt to sleep until past time returned false")
	}
	if wu := fc.Wakeups(); wu != 1 {
		t.Errorf("unexpected wakeup-count: %d; expected 1", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}
}

func TestFakeClockWithRelativeWaiterWithCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	expectedTime = expectedTime.Add(time.Minute)

	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	sleeperWake := expectedTime.Add(time.Hour * 2)
	ch := make(chan bool)
	go func() {
		ch <- fc.SleepFor(ctx, time.Hour*2)
	}()

	fc.AwaitAggSleepers(1)

	if sl := fc.NumSleepers(); sl != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	if as := fc.NumAggSleepers(); as != 1 {
		t.Errorf("unexpected number of aggregate sleepers: %d; expected 1", as)
	}

	// make sure we're still sleeping
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	// verify that our one sleeper is still sleeping
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; ", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	// cancel the context
	cancel()
	fc.AwaitSleepAborts(1)

	// wait for our sleeper to wake and return (expected true)
	if v := <-ch; v {
		t.Errorf("sleeper awoke with unexpected return value: %t; expected false", v)
	}

	if !fc.SleepFor(ctx, -1*time.Minute) {
		t.Errorf("attempt to sleep until past time returned false")
	}
	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 1 {
		t.Errorf("unexpected abort-count: %d; expected 1", sa)
	}
}

func TestFakeClockWithAbsoluteWaiterWithCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	expectedTime = expectedTime.Add(time.Minute)

	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	sleeperWake := baseTime.Add(time.Hour * 2)
	ch := make(chan bool)
	go func() {
		ch <- fc.SleepUntil(ctx, sleeperWake)
	}()

	fc.AwaitAggSleepers(1)

	if sl := fc.NumSleepers(); sl != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	if as := fc.NumAggSleepers(); as != 1 {
		t.Errorf("unexpected number of aggregate sleepers: %d; expected 1", as)
	}

	// make sure we're still sleeping
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	// verify that our one sleeper is still sleeping
	if sl := fc.Sleepers(); len(sl) != 1 {
		t.Errorf("unexpected sleeper-count: %d; ", len(sl))
	} else if !sl[0].Equal(sleeperWake) {
		t.Errorf("solitary sleeper has an incorrect wake-time: %s; expected %s",
			sl[0], sleeperWake)
	}

	// cancel the context
	cancel()
	fc.AwaitSleepAborts(1)

	// wait for our sleeper to wake and return (expected true)
	if v := <-ch; v {
		t.Errorf("sleeper awoke with unexpected return value: %t; expected false", v)
	}

	if !fc.SleepFor(ctx, -1*time.Minute) {
		t.Errorf("attempt to sleep until past time returned false")
	}
	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 1 {
		t.Errorf("unexpected abort-count: %d; expected 1", sa)
	}
}

func TestFakeClockAfterFuncTimeWake(t *testing.T) {
	t.Parallel()
	baseTime := time.Now()
	fc := NewClock(baseTime)

	expectedTime := baseTime

	aggCallbackWaitCh := make(chan struct{})
	go func() {
		defer close(aggCallbackWaitCh)
		fc.AwaitAggCallbacks(1)
	}()
	regCallbackWaitCh := make(chan struct{})
	go func() {
		defer close(regCallbackWaitCh)
		fc.AwaitRegisteredCallbacks(1)
	}()
	runtime.Gosched()

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	// make sure we get the same value a second time
	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 0 {
		t.Errorf("unexpected registered callbacks: %d; expected 0", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 0 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 0", regCBs)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}
	cbRun := make(chan struct{})
	timerHandle := fc.AfterFunc(time.Hour, func() { close(cbRun) })

	<-aggCallbackWaitCh
	<-regCallbackWaitCh

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 1 {
		t.Errorf("unexpected registered callbacks: %d; expected 1", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 1 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 1", regCBs)
	}

	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}

	if cbWakes := fc.RegisteredCallbacks(); len(cbWakes) != 1 || !cbWakes[0].Equal(
		baseTime.Add(time.Hour)) {
		t.Errorf("unexpected scheduled exec time for callback: %v, expected %s",
			cbWakes, baseTime.Add(time.Hour))
	}
	select {
	case <-cbRun:
		t.Errorf("callback ran when canceled before time advanced to exec-point")
	default:
	}

	expectedTime = expectedTime.Add(time.Minute)
	if fn := fc.Now(); !fn.Equal(expectedTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", expectedTime, fn)
	}

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 1 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 1", wakers)
	}

	// Wait for the callback to complete
	<-cbRun

	if cbExecs := fc.NumCallbackExecs(); cbExecs != 1 {
		t.Errorf("unexpected executed callbacks: %d; expected 1", cbExecs)
	}

	if zeroDur := fc.Until(expectedTime); zeroDur != 0 {
		t.Errorf("expected zero duration, got %s", zeroDur)
	}

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}
	if timerHandle.Stop() {
		t.Errorf("stop returned true after callback execution")
	}

}
func TestFakeClockAfterFuncTimeAbort(t *testing.T) {
	t.Parallel()
	baseTime := time.Now()
	fc := NewClock(baseTime)

	aggCallbackWaitCh := make(chan struct{})
	go func() {
		defer close(aggCallbackWaitCh)
		fc.AwaitAggCallbacks(1)
	}()
	regCallbackWaitCh := make(chan struct{})
	go func() {
		defer close(regCallbackWaitCh)
		fc.AwaitRegisteredCallbacks(1)
	}()
	cancelCBWaitCh := make(chan struct{})
	go func() {
		defer close(cancelCBWaitCh)
		fc.AwaitTimerAborts(1)
	}()

	runtime.Gosched()

	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}
	// make sure we get the same value a second time
	if fn := fc.Now(); !fn.Equal(baseTime) {
		t.Errorf("mismatched baseTime(%s) and unincremented Now()(%s)", baseTime, fn)
	}

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 0 {
		t.Errorf("unexpected registered callbacks: %d; expected 0", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 0 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 0", regCBs)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}

	cbRun := make(chan struct{})
	timerHandle := fc.AfterFunc(time.Hour, func() { close(cbRun) })

	<-aggCallbackWaitCh
	<-regCallbackWaitCh

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 1 {
		t.Errorf("unexpected registered callbacks: %d; expected 1", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 1 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 1", regCBs)
	}

	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}

	if cbWakes := fc.RegisteredCallbacks(); len(cbWakes) != 1 || !cbWakes[0].Equal(
		baseTime.Add(time.Hour)) {
		t.Errorf("unexpected scheduled exec time for callback: %v, expected %s",
			cbWakes, baseTime.Add(time.Hour))
	}

	if !timerHandle.Stop() {
		t.Errorf("callback ran prematurely; stop returned false")
	}
	if timerHandle.Stop() {
		t.Errorf("stop returned true after previous stop")
	}

	<-cancelCBWaitCh

	select {
	case <-cbRun:
		t.Errorf("callback ran when canceled before time advanced to exec-point")
	default:
	}

	if cbWakes := fc.RegisteredCallbacks(); len(cbWakes) != 0 {
		t.Errorf("unexpected scheduled exec times for callback(s): %v, none expected",
			cbWakes)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}
	if cbAborts := fc.NumTimerAborts(); cbAborts != 1 {
		t.Errorf("unexpected aborted callbacks: %d; expected 1", cbAborts)
	}

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}

}

func TestFakeClockAfterFuncNegDur(t *testing.T) {
	t.Parallel()
	baseTime := time.Now()
	fc := NewClock(baseTime)

	aggCallbackWaitCh := make(chan struct{})
	go func() {
		defer close(aggCallbackWaitCh)
		fc.AwaitAggCallbacks(1)
	}()

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 0 {
		t.Errorf("unexpected registered callbacks: %d; expected 0", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 0 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 0", regCBs)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}

	cbRun := make(chan struct{})
	timerHandle := fc.AfterFunc(-time.Hour, func() { close(cbRun) })
	<-aggCallbackWaitCh
	<-cbRun

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 0 {
		t.Errorf("unexpected registered callbacks: %d; expected 0", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 1 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 1", regCBs)
	}

	if cbExecs := fc.NumCallbackExecs(); cbExecs != 1 {
		t.Errorf("unexpected executed callbacks: %d; expected 1", cbExecs)
	}

	if timerHandle.Stop() {
		t.Errorf("stop returned true")
	}

	if cbWakes := fc.RegisteredCallbacks(); len(cbWakes) != 0 {
		t.Errorf("unexpected scheduled exec times for callback(s): %v, none expected",
			cbWakes)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 1 {
		t.Errorf("unexpected executed callbacks: %d; expected 1", cbExecs)
	}
	if cbAborts := fc.NumTimerAborts(); cbAborts != 0 {
		t.Errorf("unexpected aborted callbacks: %d; expected 0", cbAborts)
	}

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected abort-count: %d; expected 0", sa)
	}

}
