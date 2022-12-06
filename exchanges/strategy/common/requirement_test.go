package common

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type superstrat struct {
	Requirements
}

func (s *superstrat) ReportStart(_ fmt.Stringer)    {}
func (s *superstrat) GetSignal() <-chan interface{} { return nil }
func (s *superstrat) GetEnd() <-chan time.Time      { return nil }

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

	err = req.Run(context.Background(), &superstrat{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}
}
