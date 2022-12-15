package common

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

type superstrat struct {
	Requirements
	signalNotComplete bool
	signalComplete    bool
	signalTimeout     bool
	RandoReporter     chan Reason
	protec            sync.Mutex
}

func (s *superstrat) ReportStart(Descriptor) {}

func (s *superstrat) GetNext() time.Time         { return time.Time{} }
func (s *superstrat) ReportShutdown()            { s.RandoReporter <- Shutdown }
func (s *superstrat) ReportContextDone(error)    { s.RandoReporter <- ContextDone }
func (s *superstrat) ReportTimeout(time.Time)    { s.RandoReporter <- TimeOut }
func (s *superstrat) ReportComplete()            { s.RandoReporter <- Complete }
func (s *superstrat) ReportFatalError(error)     { s.RandoReporter <- FatalError }
func (s *superstrat) ReportWait(time.Time)       { s.RandoReporter <- Wait }
func (s *superstrat) GetDescription() Descriptor { return nil }
func (s *superstrat) CanContinuePassedEnd() bool { return false }
func (s *superstrat) GetID() uuid.UUID           { return uuid.Nil }
func (s *superstrat) OnSignal(context.Context, interface{}) (bool, error) {
	s.protec.Lock()
	defer s.protec.Unlock()
	return s.signalComplete, nil
}
func (s *superstrat) GetSignal() <-chan interface{} {
	s.protec.Lock()
	defer s.protec.Unlock()
	if s.signalComplete || s.signalNotComplete {
		ch := make(chan interface{})
		close(ch)
		return ch
	}
	return nil
}
func (s *superstrat) GetEnd(bool) <-chan time.Time {
	if s.signalTimeout {
		ch := make(chan time.Time)
		close(ch)
		return ch
	}
	return nil
}
func (s *superstrat) SetSignalNotComplete(b bool) {
	s.protec.Lock()
	defer s.protec.Unlock()
	s.signalNotComplete = b
}
func (s *superstrat) SetSignalComplete(b bool) {
	s.protec.Lock()
	defer s.protec.Unlock()
	s.signalComplete = b
}
func (s *superstrat) SetSignalTimeout(b bool) {
	s.protec.Lock()
	defer s.protec.Unlock()
	s.signalTimeout = b
}

func TestRun(t *testing.T) {
	t.Parallel()

	var req *Requirement
	err := req.Run(context.Background(), nil)
	if !errors.Is(err, errRequirementIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, errRequirementIsNil)
	}

	req = &Requirement{}
	err = req.Run(context.Background(), nil)
	if !errors.Is(err, ErrIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrIsNil)
	}

	s := &superstrat{RandoReporter: make(chan Reason, 100)}
	err = req.Run(context.Background(), s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	// shutdown via strategy manager example
	err = req.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if reason := <-s.RandoReporter; reason != Shutdown {
		t.Fatalf("received: '%v' but expected '%v'", reason, Shutdown)
	}

	// context done
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	err = req.Run(ctx, s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	cancel()
	if reason := <-s.RandoReporter; reason != ContextDone {
		t.Fatalf("received: '%v' but expected '%v'", reason, ContextDone)
	}

	s.SetSignalTimeout(true)
	// get end signal
	err = req.Run(context.Background(), s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if reason := <-s.RandoReporter; reason != TimeOut {
		t.Fatalf("received: '%v' but expected '%v'", reason, TimeOut)
	}

	s.SetSignalTimeout(false)
	s.SetSignalNotComplete(true)
	// get normal signal don't complete
	err = req.Run(context.Background(), s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if reason := <-s.RandoReporter; reason != Wait {
		t.Fatalf("received: '%v' but expected '%v'", reason, TimeOut)
	}

	err = req.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

empty:
	for {
		// Throw away and empty every action that was put in.
		select {
		case <-s.RandoReporter:
		default:
			break empty
		}
	}

	s.SetSignalNotComplete(false)
	s.SetSignalComplete(true)
	// get normal signal then complete
	err = req.Run(context.Background(), s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if reason := <-s.RandoReporter; reason != Complete {
		t.Fatalf("received: '%v' but expected '%v'", reason, Complete)
	}
}

func TestStop(t *testing.T) {
	t.Parallel()

	var req *Requirement
	err := req.Run(context.Background(), nil)
	if !errors.Is(err, errRequirementIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, errRequirementIsNil)
	}

	req = &Requirement{}
	err = req.Run(context.Background(), nil)
	if !errors.Is(err, ErrIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrIsNil)
	}

	err = req.Run(context.Background(), &superstrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}
}

func TestGetDetails(t *testing.T) {
	t.Parallel()

	var req *Requirement
	_, err := req.GetDetails()
	if !errors.Is(err, errRequirementIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, errRequirementIsNil)
	}

	registeredAt, running, strategyname := time.Now(), true, "teststrat"
	req = &Requirement{registered: registeredAt, running: running, strategy: strategyname}
	deets, err := req.GetDetails()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if !deets.Registered.Equal(registeredAt) {
		t.Fatalf("received: '%v' but expected '%v'", deets.Registered, registeredAt)
	}

	if !deets.Running {
		t.Fatalf("received: '%v' but expected '%v'", deets.Running, true)
	}

	if deets.Strategy != strategyname {
		t.Fatalf("received: '%v' but expected '%v'", deets.Running, true)
	}
}

func TestGetReporter(t *testing.T) {
	t.Parallel()

	var req *Requirement
	_, err := req.GetReporter(false)
	if !errors.Is(err, errRequirementIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, errRequirementIsNil)
	}

	req = &Requirement{}
	_, err = req.GetReporter(false)
	if !errors.Is(err, ErrReporterIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrReporterIsNil)
	}
}

func TestLoadID(t *testing.T) {
	t.Parallel()

	var req *Requirement
	err := req.LoadID(uuid.Nil)
	if !errors.Is(err, errRequirementIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, errRequirementIsNil)
	}

	req = &Requirement{}
	err = req.LoadID(uuid.Nil)
	if !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrInvalidUUID)
	}

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	err = req.LoadID(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	err = req.LoadID(id)
	if !errors.Is(err, errIDAlreadySet) {
		t.Fatalf("received: '%v' but expected '%v'", err, errIDAlreadySet)
	}
}
