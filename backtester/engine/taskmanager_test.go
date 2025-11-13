package engine

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

func TestSetupRunManager(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	if rm == nil {
		t.Errorf("received '%v' expected '%v'", rm, "&TaskManager{}")
	}
}

func TestAddRun(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	err := rm.AddTask(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	if bt.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, "a random ID")
	}
	if len(rm.tasks) != 1 {
		t.Errorf("received '%v' expected '%v'", len(rm.tasks), 1)
	}

	err = rm.AddTask(bt)
	assert.ErrorIs(t, err, errTaskAlreadyMonitored)

	if len(rm.tasks) != 1 {
		t.Errorf("received '%v' expected '%v'", len(rm.tasks), 1)
	}

	rm = nil
	err = rm.AddTask(bt)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestGetSummary(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	id, err := uuid.NewV4()
	assert.NoError(t, err)

	_, err = rm.GetSummary(id)
	assert.ErrorIs(t, err, errTaskNotFound)

	bt := &BackTest{
		Strategy:  &binancecashandcarry.Strategy{},
		Statistic: &statistics.Statistic{},
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	sum, err := rm.GetSummary(bt.MetaData.ID)
	assert.NoError(t, err)

	if sum.MetaData.ID != bt.MetaData.ID {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, bt.MetaData.ID)
	}

	rm = nil
	_, err = rm.GetSummary(id)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestList(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	list, err := rm.List()
	assert.NoError(t, err)

	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	bt := &BackTest{
		Strategy:  &binancecashandcarry.Strategy{},
		Statistic: &statistics.Statistic{},
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	list, err = rm.List()
	assert.NoError(t, err)

	if len(list) != 1 {
		t.Errorf("received '%v' expected '%v'", len(list), 1)
	}

	rm = nil
	_, err = rm.List()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStopRun(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	list, err := rm.List()
	assert.NoError(t, err)

	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	id, err := uuid.NewV4()
	assert.NoError(t, err)

	err = rm.StopTask(id)
	assert.ErrorIs(t, err, errTaskNotFound)

	bt := &BackTest{
		Strategy:  &fakeStrat{},
		Statistic: &fakeStats{},
		Reports:   &fakeReport{},
		shutdown:  make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	err = rm.StopTask(bt.MetaData.ID)
	assert.ErrorIs(t, err, errTaskHasNotRan)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	err = rm.StopTask(bt.MetaData.ID)
	assert.NoError(t, err)

	err = rm.StopTask(bt.MetaData.ID)
	assert.ErrorIs(t, err, errAlreadyRan)

	rm = nil
	err = rm.StopTask(id)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStopAllRuns(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	stoppedRuns, err := rm.StopAllTasks()
	assert.NoError(t, err)

	if len(stoppedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(stoppedRuns), 0)
	}

	bt := &BackTest{
		Strategy:  &binancecashandcarry.Strategy{},
		Statistic: &fakeStats{},
		Reports:   &fakeReport{},
		shutdown:  make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	stoppedRuns, err = rm.StopAllTasks()
	assert.NoError(t, err)

	if len(stoppedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(stoppedRuns), 1)
	}

	rm = nil
	_, err = rm.StopAllTasks()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStartRun(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	list, err := rm.List()
	assert.NoError(t, err)

	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	id, err := uuid.NewV4()
	assert.NoError(t, err)

	err = rm.StartTask(id)
	assert.ErrorIs(t, err, errTaskNotFound)

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	err = rm.StartTask(bt.MetaData.ID)
	assert.NoError(t, err)

	err = rm.StartTask(bt.MetaData.ID)
	assert.ErrorIs(t, err, errTaskIsRunning)

	bt.m.Lock()
	bt.MetaData.DateEnded = time.Now()
	bt.MetaData.Closed = true
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()

	err = rm.StartTask(bt.MetaData.ID)
	assert.ErrorIs(t, err, errAlreadyRan)

	rm = nil
	err = rm.StartTask(id)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStartAllRuns(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()
	startedRuns, err := rm.StartAllTasks()
	assert.NoError(t, err)

	if len(startedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 0)
	}

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	startedRuns, err = rm.StartAllTasks()
	assert.NoError(t, err)

	if len(startedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(startedRuns), 1)
	}

	rm = nil
	_, err = rm.StartAllTasks()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestClearRun(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()

	id, err := uuid.NewV4()
	assert.NoError(t, err)

	err = rm.ClearTask(id)
	assert.ErrorIs(t, err, errTaskNotFound)

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	err = rm.ClearTask(bt.MetaData.ID)
	assert.ErrorIs(t, err, errCannotClear)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.m.Unlock()
	err = rm.ClearTask(bt.MetaData.ID)
	assert.NoError(t, err)

	list, err := rm.List()
	assert.NoError(t, err)

	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	rm = nil
	err = rm.ClearTask(id)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestClearAllRuns(t *testing.T) {
	t.Parallel()
	rm := NewTaskManager()

	clearedRuns, remainingRuns, err := rm.ClearAllTasks()
	if len(clearedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 0)
	}
	if len(remainingRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 0)
	}
	assert.NoError(t, err)

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = rm.AddTask(bt)
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	clearedRuns, remainingRuns, err = rm.ClearAllTasks()
	if len(clearedRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 0)
	}
	if len(remainingRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 1)
	}
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.m.Unlock()
	clearedRuns, remainingRuns, err = rm.ClearAllTasks()
	if len(clearedRuns) != 1 {
		t.Errorf("received '%v' expected '%v'", len(clearedRuns), 1)
	}
	if len(remainingRuns) != 0 {
		t.Errorf("received '%v' expected '%v'", len(remainingRuns), 0)
	}
	assert.NoError(t, err)

	list, err := rm.List()
	assert.NoError(t, err)

	if len(list) != 0 {
		t.Errorf("received '%v' expected '%v'", len(list), 0)
	}

	rm = nil
	_, _, err = rm.ClearAllTasks()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}
