package common

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestNewScheduler(t *testing.T) {
	t.Parallel()
	_, err := NewScheduler(time.Time{}, time.Time{}, false, kline.Interval(time.Millisecond))
	if !errors.Is(err, ErrIntervalNotSupported) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrIntervalNotSupported)
	}

	start := time.Now().Add(time.Minute)
	end := start.Add(time.Second * 30)
	_, err = NewScheduler(start, end, false, kline.OneMin)
	if !errors.Is(err, ErrCannotGenerateSignal) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrCannotGenerateSignal)
	}

	sched, err := NewScheduler(time.Time{}, time.Time{}, false, kline.OneMin)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	<-sched.GetSignal() // This should fire immediately

	if sched.GetEnd(false) != nil {
		t.Fatalf("received: '%v' but expected '%v'", "chan", "nil chan")
	}

	nextDeploymentTime := sched.GetNext()
	if time.Until(nextDeploymentTime) > time.Minute {
		t.Fatalf("received: '%v' but expected '%v'", time.Until(nextDeploymentTime), time.Minute)
	}

	// schedule start not aligned
	start = time.Now().Add(time.Minute) // Should fire 1 minute from now
	end = start.Add(time.Minute * 5)    // Should finish 5 minutes from now

	sched, err = NewScheduler(start, end, false, kline.OneMin)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	select {
	case <-sched.GetSignal(): // This should *not* fire immediately
		t.Fatalf("received: '%v' but expected '%v'", "chan fired", "chan not to fire cause its scheduled")
	default:
	}

	if sched.GetEnd(false) == nil {
		t.Fatalf("received: '%v' but expected '%v'", "nil chan", "non-nil chan")
	}

	if !sched.end.Equal(end) {
		t.Fatalf("set: '%v' but expected '%v'", sched.end, end)
	}

	nextDeploymentTime = sched.GetNext()
	if time.Until(nextDeploymentTime) > time.Minute {
		t.Fatalf("received: '%v' but expected '%v'", time.Until(nextDeploymentTime), time.Minute)
	}

	// schedule start aligned
	start = time.Now().Add(time.Minute)
	end = start.Add(time.Minute * 5)
	sched, err = NewScheduler(start, end, true, kline.OneMin)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	select {
	case <-sched.GetSignal(): // This should *not* fire immediately
		t.Fatalf("received: '%v' but expected '%v'", "chan fired", "chan not to fire cause its scheduled")
	default:
	}

	if sched.GetEnd(false) == nil {
		t.Fatalf("received: '%v' but expected '%v'", "nil chan", "non-nil chan")
	}

	nextDeploymentTime = sched.GetNext()
	if time.Until(nextDeploymentTime) < time.Minute {
		t.Fatalf("received: '%v' but expected '%v'", time.Until(nextDeploymentTime), time.Minute)
	}
}
