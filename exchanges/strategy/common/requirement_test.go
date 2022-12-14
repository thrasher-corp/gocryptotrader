package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

type superstrat struct {
	Requirements
	SignalNotComplete bool
	SignalComplete    bool
	SignalTimeout     bool
	RandoReporter     chan struct{}
}

func (s *superstrat) ReportStart(_ Descriptor)       {}
func (s *superstrat) GetEnd(_ bool) <-chan time.Time { return nil }
func (s *superstrat) GetNext() time.Time             { return time.Time{} }
func (s *superstrat) ReportShutdown()                { s.RandoReporter <- struct{}{} }
func (s *superstrat) ReportContextDone(_ error)      { s.RandoReporter <- struct{}{} }
func (s *superstrat) ReportTimeout(_ time.Time)      { s.RandoReporter <- struct{}{} }
func (s *superstrat) ReportComplete()                { s.RandoReporter <- struct{}{} }
func (s *superstrat) ReportFatalError(_ error)       { s.RandoReporter <- struct{}{} }
func (s *superstrat) ReportWait(_ time.Time)         { s.RandoReporter <- struct{}{} }
func (s *superstrat) GetDescription() Descriptor     { return nil }
func (s *superstrat) CanContinuePassedEnd() bool     { return false }
func (s *superstrat) GetID() uuid.UUID               { return uuid.Nil }
func (s *superstrat) OnSignal(_ context.Context, _ interface{}) (bool, error) {
	if s.SignalComplete {
		return true, nil
	}
	return false, nil
}
func (s *superstrat) GetSignal() <-chan interface{} {
	if s.SignalComplete || s.SignalNotComplete {
		ch := make(chan interface{})
		close(ch)
		return ch
	}
	return nil
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

	// shutdown
	s := &superstrat{RandoReporter: make(chan struct{})}
	err = req.Run(context.Background(), s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	req.shutdown <- struct{}{}
	<-s.RandoReporter
	err = req.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	// context done
	err = req.Run(ctx, s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	cancel()
	<-s.RandoReporter

	s.SignalTimeout = true
	// get end signal
	err = req.Run(ctx, s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	<-s.RandoReporter

	s.SignalTimeout = false
	s.SignalNotComplete = true
	// get normal signal don't complete
	err = req.Run(ctx, s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	<-s.RandoReporter

	s.SignalNotComplete = false
	s.SignalComplete = true
	// get normal signal then complete
	err = req.Run(ctx, s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	<-s.RandoReporter

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
