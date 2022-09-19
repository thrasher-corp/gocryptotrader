package engine

import (
	"errors"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/ftxcashandcarry"
	"github.com/thrasher-corp/gocryptotrader/backtester/writer"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"testing"
	"time"
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
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.Strategy = &ftxcashandcarry.Strategy{}
	err = rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.MetaData.ID == "" {
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
}

func TestList(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	_, err := rm.GetSummary("")
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
}

func TestGetSummary(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list := rm.List()
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	bt := &BackTest{
		Strategy: &ftxcashandcarry.Strategy{},
	}
	err := rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	list = rm.List()
	if len(list) != 1 {
		t.Errorf("received '%v' expected '%v'", len(list), 1)
	}
}

func TestStopRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list := rm.List()
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	err := rm.StopRun("")
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
}

func TestStartRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	list := rm.List()
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	err := rm.StartRun("")
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
}

func TestStartAllRuns(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	startedRuns := rm.StartAllRuns()
	if len(startedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 0)
	}

	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err := rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	startedRuns = rm.StartAllRuns()
	if len(startedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 1)
	}
}

func TestClearRun(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()

	err := rm.ClearRun("")
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
	list := rm.List()
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
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
	list := rm.List()
	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}
}

func TestReportLogs(t *testing.T) {
	t.Parallel()
	rm := SetupRunManager()
	bt := &BackTest{
		Strategy:   &ftxcashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		Datas:      &data.HandlerPerCurrency{},
		shutdown:   make(chan struct{}),
	}
	err := rm.AddRun(bt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.logHolder = &writer.Writer{}
	_, err = rm.ReportLogs(bt.MetaData.ID)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = rm.ReportLogs("")
	if !errors.Is(err, errRunNotFound) {
		t.Errorf("received '%v' expected '%v'", err, errRunNotFound)
	}
}
