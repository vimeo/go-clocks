package fake

import (
	"context"
	"runtime"
	"sort"
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

func TestFakeClockWithTimer(t *testing.T) {
	t.Parallel()

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
	timer := fc.NewTimer(time.Hour * 2)
	ch := make(chan struct{})
	go func() {
		<-*timer.Ch()
		ch <- struct{}{}
	}()

	fc.AwaitAggExtractedChans(1)

	if sl := fc.NumSleepers(); sl != 0 {
		t.Errorf("unexpected sleeper-count: %d; expected 0", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 0 {
		t.Errorf("unexpected sleeper-count: %d; expected 0", len(sl))
	}

	if as := fc.NumAggExtractedChans(); as != 1 {
		t.Errorf("unexpected number of aggregate aggregate extracted channels: %d; expected 1", as)
	}

	if regTimers := fc.RegisteredTimers(); len(regTimers) != 1 {
		t.Errorf("unexpected registered timer-count %d; %v", len(regTimers), regTimers)
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

	// make sure we're still sleeping after the SetClock call
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	fc.awaitExtractedChans(1)
	// verify that our one sleeper is still sleeping
	if sl := fc.numExtractedChans(); sl != 1 {
		t.Errorf("unexpected extracted channel-count: %d; ", sl)
	}

	if regTimers := fc.RegisteredTimers(); len(regTimers) != 1 {
		t.Errorf("unexpected registered timer-count %d; %v", len(regTimers), regTimers)
	}

	// advance to our wakeup point
	expectedTime = sleeperWake
	if wakers := fc.SetClock(sleeperWake); wakers != 1 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 1", wakers)
	}

	// wait for our sleeper to wake and return (expected true)
	<-ch

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected sleep abort-count: %d; expected 0", sa)
	}

	if sa := fc.NumTimerAborts(); sa != 0 {
		t.Errorf("unexpected timer abort-count: %d; expected 0", sa)
	}
}

// Test that cancels a timer and advances the clock in parallel to guarantee that timer operations
// are correctly synchronized. (mostly useful when run with -race)
func TestFakeClockWithTimerStopRace(t *testing.T) {
	t.Parallel()

	baseTime := time.Now()
	fc := NewClock(baseTime)
	// Setup a channel for us to close when we're done and wake the sleeping goroutine.
	testWakeCh := make(chan struct{})

	sleeperWake := baseTime.Add(time.Hour * 2)
	timer := fc.NewTimer(time.Hour * 2)
	ch := make(chan struct{})
	go func() {
		select {
		case <-*timer.Ch():
		case <-testWakeCh:
		}
		ch <- struct{}{}
	}()

	fc.AwaitAggExtractedChans(1)

	go timer.Stop()
	go fc.SetClock(sleeperWake)

	close(testWakeCh)

	<-ch

}

func TestFakeClockWithTimerWithStop(t *testing.T) {
	t.Parallel()

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

	// Setup a channel for us to close when we're done and wake the sleeping goroutine.
	testWakeCh := make(chan struct{})

	sleeperWake := expectedTime.Add(time.Hour * 2)
	timer := fc.NewTimer(time.Hour * 2)
	ch := make(chan bool)
	go func() {
		select {
		case <-*timer.Ch():
			ch <- true
		case <-testWakeCh:
			ch <- false
		}
	}()

	fc.AwaitAggExtractedChans(1)

	if sl := fc.NumSleepers(); sl != 0 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", sl)
	}
	if sl := fc.Sleepers(); len(sl) != 0 {
		t.Errorf("unexpected sleeper-count: %d; expected 1", len(sl))
	}

	if as := fc.NumAggExtractedChans(); as != 1 {
		t.Errorf("unexpected number of aggregate aggregate extracted channels: %d; expected 1", as)
	}

	if regTimers := fc.RegisteredTimers(); len(regTimers) != 1 {
		t.Errorf("unexpected registered timer-count %d; %v", len(regTimers), regTimers)
	}

	// make sure we're still sleeping
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}

	// Set up a goroutine to awaken once we have an aborted timer (our call to .Stop() further down).
	stopWaitRunning := make(chan struct{})
	stopWaitCh := make(chan struct{})
	go func() {
		close(stopWaitRunning)
		fc.AwaitTimerAborts(1)
		stopWaitCh <- struct{}{}
	}()
	// Make sure that the goroutine calling AwaitTimerAborts is running before proceeding
	<-stopWaitRunning

	expectedTime = expectedTime.Add(time.Hour)
	if wakers := fc.SetClock(expectedTime); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	// make sure we're still sleeping after the SetClock call
	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early")
	default:
	}
	select {
	case <-stopWaitCh:
		t.Errorf("timer abort watching goroutine awoke unexpectedly early (SetClock should not wake AwaitTimerAborts)")
	default:
	}

	// verify that our one sleeper is still sleeping
	if sl := fc.numExtractedChans(); sl != 1 {
		t.Errorf("unexpected extracted channel-count: %d; ", sl)
	}

	if regTimers := fc.RegisteredTimers(); len(regTimers) != 1 {
		t.Errorf("unexpected registered timer-count %d; %v", len(regTimers), regTimers)
	}

	if !timer.Stop() {
		t.Errorf("Stop indicated it didn't prevent firing (false), the clock hasn't advanced far enough")
	}

	select {
	case <-ch:
		t.Errorf("sleeper finished unexpectedly early (Stop should not wake)")
	default:
	}

	// wait for the timer aborts to wake up and finish
	<-stopWaitCh

	// advance to our wakeup point
	expectedTime = sleeperWake
	if wakers := fc.SetClock(sleeperWake); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 hour(%d); expected 0", wakers)
	}

	select {
	case <-ch:
		t.Errorf("stopped timer-based sleeper finished unexpectedly early")
	default:
	}

	close(testWakeCh)
	// wait for our sleeper to wake and return (expected true)
	if wokeByTimer := <-ch; wokeByTimer {
		t.Errorf("unexpected wake reason: timer fired, expected close of testWakeCh")
	}

	if wu := fc.Wakeups(); wu != 0 {
		t.Errorf("unexpected wakeup-count: %d; expected 0", wu)
	}
	if sa := fc.NumSleepAborts(); sa != 0 {
		t.Errorf("unexpected sleep abort-count: %d; expected 0", sa)
	}

	if sa := fc.NumTimerAborts(); sa != 1 {
		t.Errorf("unexpected timer abort-count: %d; expected 1", sa)
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

	// Register a few extra afterfuncs to repro a bug in setClockLocked
	// where we were capturing a loop variable
	for z := 0; z < 20; z++ {
		st := fc.AfterFunc(time.Hour*3, func() {})
		defer st.Stop()
	}

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

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 20 {
		t.Errorf("unexpected registered callbacks: %d; expected 20", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 20 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 20", regCBs)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}
	cbRun := make(chan struct{})
	timerHandle := fc.AfterFunc(time.Hour, func() { close(cbRun) })

	fc.WaitAfterFuncs()
	<-aggCallbackWaitCh
	<-regCallbackWaitCh

	if regCBs := fc.NumRegisteredCallbacks(); regCBs != 21 {
		t.Errorf("unexpected registered callbacks: %d; expected 21", regCBs)
	}
	if regCBs := fc.NumAggCallbacks(); regCBs != 21 {
		t.Errorf("unexpected aggregate registered callbacks: %d; expected 21", regCBs)
	}

	if wakers := fc.Advance(time.Minute); wakers != 0 {
		t.Errorf("unexpected wakers from advancing 1 minute(%d); expected 0", wakers)
	}
	if cbExecs := fc.NumCallbackExecs(); cbExecs != 0 {
		t.Errorf("unexpected executed callbacks: %d; expected 0", cbExecs)
	}

	cbWakes := fc.RegisteredCallbacks()
	sort.Slice(cbWakes, func(i, j int) bool { return cbWakes[i].Before(cbWakes[j]) })
	if len(cbWakes) != 21 || !cbWakes[0].Equal(
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

	fc.WaitAfterFuncs()
	<-aggCallbackWaitCh
	<-regCallbackWaitCh
	fc.WaitAfterFuncs()

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
	fc.WaitAfterFuncs()
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

func TestFakeClockContext(t *testing.T) {
	t.Run("ContextDeadline", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		deadline := base.Add(1)
		ctx, cancel := c.ContextWithDeadline(context.Background(), deadline)
		t.Cleanup(cancel)

		ctxDeadline, isSet := ctx.Deadline()
		if !isSet {
			t.Errorf("context deadline not set")
		}
		if !ctxDeadline.Equal(deadline) {
			t.Errorf("unexpected context deadline: %v; expected %v", ctxDeadline, deadline)
		}
	})

	t.Run("ContextWithDeadlineExceeded", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		ctx, cancel := c.ContextWithDeadline(context.Background(), base.Add(1))
		t.Cleanup(cancel)

		c.Advance(1)

		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithDeadlineNotExceeded", func(t *testing.T) {
		base := time.Now()
		c := NewClock(base)

		ctx, cancel := c.ContextWithDeadline(context.Background(), base.Add(1))
		t.Cleanup(cancel)

		select {
		case <-ctx.Done():
			t.Errorf("context should not be done")
		default:
			if ctx.Err() != nil {
				t.Errorf("unexpected error: %v; expected nil", ctx.Err())
			}
		}
	})

	t.Run("ContextWithTimeoutExceeded", func(t *testing.T) {
		c := NewClock(time.Now())
		ctx, cancel := c.ContextWithTimeout(context.Background(), 1)
		t.Cleanup(cancel)

		c.Advance(1)

		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v; expected %v", ctx.Err(), context.DeadlineExceeded)
			}
		case <-time.After(time.Second):
			t.Errorf("context not done after 1 second")
		}
	})

	t.Run("ContextWithTimeouteNotExceeded", func(t *testing.T) {
		c := NewClock(time.Now())
		ctx, cancel := c.ContextWithTimeout(context.Background(), 1)
		t.Cleanup(cancel)

		select {
		case <-ctx.Done():
			t.Errorf("context should not be done")
		default:
			if ctx.Err() != nil {
				t.Errorf("unexpected error: %v; expected nil", ctx.Err())
			}
		}
	})
}
