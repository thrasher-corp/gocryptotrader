package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

const testStrat = "test strategy"

func TestNewActivities(t *testing.T) {
	t.Parallel()

	_, err := NewActivities("", false)
	if !errors.Is(err, errStrategyDescriptionIsEmpty) {
		t.Fatalf("received: '%v' but expected '%v'", err, errStrategyDescriptionIsEmpty)
	}

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if act.id != id {
		t.Fatalf("received: '%v' but expected '%v'", act.id, id)
	}

	if act.strategy != testStrat {
		t.Fatalf("received: '%v' but expected '%v'", act.strategy, testStrat)
	}

	if act.simulation {
		t.Fatalf("received: '%v' but expected '%v'", act.simulation, false)
	}
}

func TestReportComplete(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportComplete()

	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if report.Action != nil {
			t.Fatalf("received: '%v' but expected '%v'", report.Action, nil)
		}

		if !report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, true)
		}

		if report.Reason != Complete {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Complete)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
	}
}

func TestReportTimeout(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	tn := time.Now()

	act.ReportTimeout(tn)

	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(TimeoutAction); !ok {
			t.Fatalf("received: '%v' but expected '%T'", report.Action, TimeoutAction{})
		}

		if !report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, true)
		}

		if report.Reason != TimeOut {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, TimeOut)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
	}
}

func TestReportFatalError(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportFatalError(errors.New("test error"))

	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(ErrorAction); !ok {
			t.Fatalf("received: '%v' but expected '%T'", report.Action, ErrorAction{})
		}

		if !report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, true)
		}

		if report.Reason != FatalError {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, FatalError)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
	}
}

func TestReportContextDone(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportContextDone(context.Canceled)

	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(ErrorAction); !ok {
			t.Fatalf("received: '%v' but expected '%T'", report.Action, ErrorAction{})
		}

		if !report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, true)
		}

		if report.Reason != ContextDone {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, ContextDone)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
	}
}

func TestReportShutdown(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportShutdown()

	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if report.Action != nil {
			t.Fatalf("received: '%v' but expected '%v'", report.Action, nil)
		}

		if !report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, true)
		}

		if report.Reason != Shutdown {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Shutdown)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
	}
}

func TestReportInfo(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportInfo("surprising action")
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(MessageAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%v' but expected '%T'", report.Action, MessageAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != Info {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Info)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportOrder(t *testing.T) {
	t.Parallel()
	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportOrder(OrderAction{})
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(OrderAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, OrderAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != OrderExecution {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, OrderExecution)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportStart(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportStart("")
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(MessageAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, MessageAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != Start {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Start)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportRegister(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportRegister()
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(RegisterAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, RegisterAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != Registered {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Registered)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportWait(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportWait(time.Now())
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(WaitAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, WaitAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != Wait {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, Wait)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportAcceptedSignal(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportAcceptedSignal(nil)
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(SignalAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, SignalAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != SignalAccepted {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, SignalAccepted)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}

func TestReportRejectedSignal(t *testing.T) {
	t.Parallel()

	act, err := NewActivities(testStrat, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	reporter, err := act.getReporter(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	act.ReportRejectedSignal(nil)
	for report := range reporter {
		if report.Strategy != testStrat {
			t.Fatalf("received: '%v' but expected '%v'", report.Strategy, testStrat)
		}

		if _, ok := report.Action.(SignalAction); !ok {
			if report.Reason == Shutdown {
				continue
			}
			t.Fatalf("received: '%T' but expected '%T'", report.Action, SignalAction{})
		}

		if report.Finished {
			t.Fatalf("received: '%v' but expected '%v'", report.Finished, false)
		}

		if report.Reason != SignalRejection {
			t.Fatalf("received: '%v' but expected '%v'", report.Reason, SignalRejection)
		}

		if report.Time.IsZero() {
			t.Fatalf("received: '%v' but expected '%v'", report.Time, "non zero time")
		}
		act.ReportShutdown()
	}
}
