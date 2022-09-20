package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/ftxcashandcarry"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

func TestSetupRunManager(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	if rm == nil {
		t.Errorf("received '%v' expected '%v'", rm, "&RunManager{}")
	}
}

func TestAddRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	err := rm.AddRun(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt := &BackTest{}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, "a random ID")
	}
	if len(rm.runs) != 1 {
		t.Errorf("received '%v' expected '%v'", len(rm.runs), 1)
	}

	err = rm.AddRun(bt)
	if !errors.Is(err, errRunAlreadyMonitored) {
		t.Errorf("received '%v' expected '%v'", err, errRunAlreadyMonitored)
	}
	if len(rm.runs) != 1 {
		t.Errorf("received '%v' expected '%v'", len(rm.runs), 1)
	}

	rm = nil
	err = rm.AddRun(bt)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestGetSummary(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = rm.GetSummary(id)
	if !errors.Is(err, errRunNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errRunNotFound)
	}

	bt := &BackTest{
		Strategy: &ftxcashandcarry.Strategy{},
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	sum, err := rm.GetSummary(bt.MetaData.ID)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if sum.MetaData.ID != bt.MetaData.ID {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, bt.MetaData.ID)
	}

	rm = nil
	_, err = rm.GetSummary(id)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list, err := rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	bt := &BackTest{
		Strategy: &ftxcashandcarry.Strategy{},
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	list, err = rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 1 {
		t.Errorf("received '%v' expected '%v'", len(list), 1)
	}

	rm = nil
	_, err = rm.List()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStopRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list, err := rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = rm.StopRun(id)
	if !errors.Is(err, errRunNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errRunNotFound)
	}

	bt := &BackTest{
		Strategy: &ftxcashandcarry.Strategy{},
		shutdown: make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = rm.StopRun(bt.MetaData.ID)
	if !errors.Is(err, errRunHasNotRan) {
		t.Errorf("received '%v' expected '%v'", err, errRunHasNotRan)
	}

	bt.MetaData.DateStarted = time.Now()
	err = rm.StopRun(bt.MetaData.ID)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = rm.StopRun(bt.MetaData.ID)
	if !errors.Is(err, errAlreadyRan) {
		t.Errorf("received '%v' expected '%v'", err, errAlreadyRan)
	}

	rm = nil
	err = rm.StopRun(id)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStopAllRuns(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	stoppedRuns, err := rm.StopAllRuns()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(stoppedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(stoppedRuns), 0)
	}

	bt := &BackTest{
		Strategy: &ftxcashandcarry.Strategy{},
		shutdown: make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.MetaData.DateStarted = time.Now()
	stoppedRuns, err = rm.StopAllRuns()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(stoppedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(stoppedRuns), 1)
	}

	rm = nil
	_, err = rm.StopAllRuns()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStartRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list, err := rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = rm.StartRun(id)
	if !errors.Is(err, errRunNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errRunNotFound)
	}

	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = rm.StartRun(bt.MetaData.ID)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = rm.StartRun(bt.MetaData.ID)
	if !errors.Is(err, errRunIsRunning) {
		t.Errorf("received '%v' expected '%v'", err, errRunIsRunning)
	}

	bt.MetaData.DateEnded = time.Now()
	bt.MetaData.Closed = true

	err = rm.StartRun(bt.MetaData.ID)
	if !errors.Is(err, errAlreadyRan) {
		t.Errorf("received '%v' expected '%v'", err, errAlreadyRan)
	}

	rm = nil
	err = rm.StartRun(id)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStartAllRuns(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	startedRuns, err := rm.StartAllRuns()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(startedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 0)
	}

	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	startedRuns, err = rm.StartAllRuns()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(startedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 1)
	}

	rm = nil
	_, err = rm.StartAllRuns()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestClearRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()

	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = rm.ClearRun(id)
	if !errors.Is(err, errRunNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errRunNotFound)
	}

	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.MetaData.DateStarted = time.Now()
	err = rm.ClearRun(bt.MetaData.ID)
	if !errors.Is(err, errCannotClear) {
		t.Errorf("received '%v' expected '%v'", err, errCannotClear)
	}

	bt.MetaData.DateStarted = time.Time{}
	err = rm.ClearRun(bt.MetaData.ID)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	list, err := rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	rm = nil
	err = rm.ClearRun(id)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestClearAllRuns(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()

	clearedRuns, remainingRuns, err := rm.ClearAllRuns()
	if len(clearedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 0)
	}
	if len(remainingRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 0)
	}
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.MetaData.DateStarted = time.Now()
	clearedRuns, remainingRuns, err = rm.ClearAllRuns()
	if len(clearedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 0)
	}
	if len(remainingRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 1)
	}
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.MetaData.DateStarted = time.Time{}
	clearedRuns, remainingRuns, err = rm.ClearAllRuns()
	if len(clearedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 1)
	}
	if len(remainingRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 0)
	}
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	list, err := rm.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	rm = nil
	_, _, err = rm.ClearAllRuns()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}
