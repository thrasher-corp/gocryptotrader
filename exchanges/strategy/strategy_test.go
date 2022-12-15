package strategy

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

type testStrat struct {
	strategy.Requirements
	Running bool
	id      uuid.UUID
}

func (s *testStrat) Run(ctx context.Context, runner strategy.Requirements) error {
	return nil
}
func (s *testStrat) GetReporter(_ bool) (<-chan *strategy.Report, error) {
	m := make(chan *strategy.Report)
	close(m)
	return m, nil
}
func (s *testStrat) GetDetails() (*strategy.Details, error) {
	return &strategy.Details{Running: s.Running, ID: s.id}, nil
}
func (s *testStrat) Stop() error { return nil }
func (s *testStrat) LoadID(id uuid.UUID) error {
	s.id = id
	return nil
}
func (s *testStrat) GetID() uuid.UUID {
	return s.id
}
func (s *testStrat) GetDescription() strategy.Descriptor {
	return nil
}
func (s *testStrat) ReportRegister() {}

func TestRegister(t *testing.T) {
	t.Parallel()

	var m Manager
	_, err := m.Register(nil)
	if !errors.Is(err, strategy.ErrIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrIsNil)
	}

	id, err := m.Register(&testStrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if id.IsNil() {
		t.Fatalf("received: '%v' but expected '%v'", nil, "uuid")
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	var m Manager
	err := m.Run(context.Background(), uuid.Nil)
	if !errors.Is(err, strategy.ErrInvalidUUID) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrInvalidUUID)
	}

	registeredID, err := m.Register(&testStrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if registeredID.IsNil() {
		t.Fatalf("received: '%v' but expected '%v'", nil, "uuid")
	}

	notRegisteredID, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	err = m.Run(context.Background(), notRegisteredID)
	if !errors.Is(err, strategy.ErrNotFound) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrNotFound)
	}

	err = m.Run(context.Background(), registeredID)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}
}

func TestRunStream(t *testing.T) {
	t.Parallel()

	var m Manager
	_, err := m.RunStream(context.Background(), uuid.Nil, false)
	if !errors.Is(err, strategy.ErrInvalidUUID) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrInvalidUUID)
	}

	registeredID, err := m.Register(&testStrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if registeredID.IsNil() {
		t.Fatalf("received: '%v' but expected '%v'", nil, "uuid")
	}

	notRegisteredID, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	_, err = m.RunStream(context.Background(), notRegisteredID, false)
	if !errors.Is(err, strategy.ErrNotFound) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrNotFound)
	}

	reporter, err := m.RunStream(context.Background(), registeredID, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if reporter == nil {
		t.Fatalf("received: '%v' but expected '%v'", reporter, "reporter")
	}
}

func TestGetAllStrategies(t *testing.T) {
	t.Parallel()

	var m Manager
	deets, err := m.GetAllStrategies(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if len(deets) != 0 {
		t.Fatalf("received: '%v' but expected '%v'", len(deets), 0)
	}

	id1, err := m.Register(&testStrat{Running: true})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	id2, err := m.Register(&testStrat{Running: true})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	id3, err := m.Register(&testStrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	deets, err = m.GetAllStrategies(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if len(deets) != 3 {
		t.Fatalf("received: '%v' but expected '%v'", len(deets), 3)
	}

	for x := range deets {
		if deets[x].ID != id1 && deets[x].ID != id2 && deets[x].ID != id3 {
			t.Fatalf("received: '%v' but expected '%v'", deets[x].ID, "expected to match with IDs")
		}
	}

	deets, err = m.GetAllStrategies(true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if len(deets) != 2 {
		t.Fatalf("received: '%v' but expected '%v'", len(deets), 2)
	}

	for x := range deets {
		if deets[x].ID != id1 && deets[x].ID != id2 {
			t.Fatalf("received: '%v' but expected '%v'", deets[x].ID, "expected to match with running IDs")
		}
	}
}

func TestStop(t *testing.T) {
	t.Parallel()

	var m Manager
	err := m.Stop(uuid.Nil)
	if !errors.Is(err, strategy.ErrInvalidUUID) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrInvalidUUID)
	}

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	err = m.Stop(id)
	if !errors.Is(err, strategy.ErrNotFound) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrNotFound)
	}

	id, err = m.Register(&testStrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	err = m.Stop(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}
}
