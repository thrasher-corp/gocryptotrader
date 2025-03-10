package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/gofrs/uuid"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errTaskNotFound         = errors.New("task not found")
	errTaskAlreadyMonitored = errors.New("task already monitored")
	errAlreadyRan           = errors.New("task already ran")
	errTaskHasNotRan        = errors.New("task hasn't ran yet")
	errTaskIsRunning        = errors.New("task is already running")
	errCannotClear          = errors.New("cannot clear task")
)

// NewTaskManager creates a run manager to allow the backtester to manage multiple strategies
func NewTaskManager() *TaskManager {
	return &TaskManager{}
}

// AddTask adds a run to the manager
func (r *TaskManager) AddTask(b *BackTest) error {
	if r == nil {
		return fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	if b == nil {
		return fmt.Errorf("%w BackTest", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.tasks {
		if r.tasks[i].Equal(b) {
			return fmt.Errorf("%w %s %s", errTaskAlreadyMonitored, b.MetaData.ID, b.MetaData.Strategy)
		}
	}

	err := b.SetupMetaData()
	if err != nil {
		return err
	}
	r.tasks = append(r.tasks, b)
	return nil
}

// List details all strategy tasks
func (r *TaskManager) List() ([]*TaskSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	resp := make([]*TaskSummary, len(r.tasks))
	for i := range r.tasks {
		sum, err := r.tasks[i].GenerateSummary()
		if err != nil {
			return nil, err
		}
		resp[i] = sum
	}
	return resp, nil
}

// GetSummary returns details about a completed strategy task
func (r *TaskManager) GetSummary(id uuid.UUID) (*TaskSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.tasks {
		if !r.tasks[i].MatchesID(id) {
			continue
		}
		return r.tasks[i].GenerateSummary()
	}
	return nil, fmt.Errorf("%s %w", id, errTaskNotFound)
}

// StopTask stops a strategy task if enabled, this will run CloseAllPositions
func (r *TaskManager) StopTask(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.tasks {
		switch {
		case !r.tasks[i].MatchesID(id):
			continue
		case r.tasks[i].IsRunning():
			return r.tasks[i].Stop()
		case r.tasks[i].HasRan():
			return fmt.Errorf("%w %v", errAlreadyRan, id)
		default:
			return fmt.Errorf("%w %v", errTaskHasNotRan, id)
		}
	}
	return fmt.Errorf("%s %w", id, errTaskNotFound)
}

// StopAllTasks stops all running strategies
func (r *TaskManager) StopAllTasks() ([]*TaskSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	resp := make([]*TaskSummary, 0, len(r.tasks))
	for i := range r.tasks {
		if !r.tasks[i].IsRunning() {
			continue
		}
		err := r.tasks[i].Stop()
		if err != nil {
			return nil, err
		}
		sum, err := r.tasks[i].GenerateSummary()
		if err != nil {
			return nil, err
		}
		resp = append(resp, sum)
	}
	return resp, nil
}

// StartTask executes a strategy if found
func (r *TaskManager) StartTask(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.tasks {
		switch {
		case !r.tasks[i].MatchesID(id):
			continue
		case r.tasks[i].IsRunning():
			return fmt.Errorf("%w %v", errTaskIsRunning, id)
		case r.tasks[i].HasRan():
			return fmt.Errorf("%w %v", errAlreadyRan, id)
		default:
			return r.tasks[i].ExecuteStrategy(false)
		}
	}
	return fmt.Errorf("%s %w", id, errTaskNotFound)
}

// StartAllTasks executes all strategies
func (r *TaskManager) StartAllTasks() ([]uuid.UUID, error) {
	if r == nil {
		return nil, fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	executedRuns := make([]uuid.UUID, 0, len(r.tasks))
	for i := range r.tasks {
		if r.tasks[i].HasRan() {
			continue
		}
		executedRuns = append(executedRuns, r.tasks[i].MetaData.ID)
		err := r.tasks[i].ExecuteStrategy(false)
		if err != nil {
			return nil, err
		}
	}

	return executedRuns, nil
}

// ClearTask removes a run from memory, but only if it is not running
func (r *TaskManager) ClearTask(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.tasks {
		if !r.tasks[i].MatchesID(id) {
			continue
		}
		if r.tasks[i].IsRunning() {
			return fmt.Errorf("%w %v, currently running. Stop it first", errCannotClear, r.tasks[i].MetaData.ID)
		}
		r.tasks = slices.Delete(r.tasks, i, i+1)
		return nil
	}
	return fmt.Errorf("%s %w", id, errTaskNotFound)
}

// ClearAllTasks removes all tasks from memory, but only if they are not running
func (r *TaskManager) ClearAllTasks() (clearedRuns, remainingRuns []*TaskSummary, err error) {
	if r == nil {
		return nil, nil, fmt.Errorf("%w TaskManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := 0; i < len(r.tasks); i++ {
		var run *TaskSummary
		run, err = r.tasks[i].GenerateSummary()
		if err != nil {
			return nil, nil, err
		}
		if r.tasks[i].IsRunning() {
			remainingRuns = append(remainingRuns, run)
		} else {
			clearedRuns = append(clearedRuns, run)
			r.tasks = slices.Delete(r.tasks, i, i+1)
			i--
		}
	}
	return clearedRuns, remainingRuns, nil
}
