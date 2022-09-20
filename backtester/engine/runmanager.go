package engine

import (
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errRunNotFound         = errors.New("run not found")
	errRunAlreadyMonitored = errors.New("run already monitored")
	errAlreadyRan          = errors.New("run already ran")
	errRunHasNotRan        = errors.New("run hasn't ran yet")
	errRunIsRunning        = errors.New("run is already running")
	errCannotClear         = errors.New("cannot clear run")
)

// SetupRunManager creates a run manager to allow the backtester to manage multiple strategies
func SetupRunManager() *RunManager {
	return &RunManager{}
}

// AddRun adds a run to the manager
func (r *RunManager) AddRun(b *BackTest) error {
	if r == nil {
		return fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	if b == nil {
		return fmt.Errorf("%w BackTest", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	err := b.SetupMetaData()
	if err != nil {
		return err
	}
	for i := range r.runs {
		if r.runs[i].Equal(b) {
			return fmt.Errorf("%w %s %s", errRunAlreadyMonitored, b.MetaData.ID, b.MetaData.Strategy)
		}
	}

	r.runs = append(r.runs, b)
	return nil
}

// List details all backtesting/livestrategy runs
func (r *RunManager) List() ([]*RunSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	resp := make([]*RunSummary, len(r.runs))
	for i := range r.runs {
		sum, err := r.runs[i].GenerateSummary()
		if err != nil {
			return nil, err
		}
		resp[i] = sum
	}
	return resp, nil
}

// GetSummary returns details about a completed backtesting/livestrategy run
func (r *RunManager) GetSummary(id uuid.UUID) (*RunSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.runs {
		if !r.runs[i].MatchesID(id) {
			continue
		}
		return r.runs[i].GenerateSummary()
	}
	return nil, fmt.Errorf("%s %w", id, errRunNotFound)
}

// StopRun stops a backtesting/livestrategy run if enabled, this will run CloseAllPositions
func (r *RunManager) StopRun(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.runs {
		if !r.runs[i].MatchesID(id) {
			continue
		}
		switch {
		case r.runs[i].IsRunning():
			r.runs[i].Stop()
			return nil
		case r.runs[i].HasRan():
			return fmt.Errorf("%w %v", errAlreadyRan, id)
		default:
			return fmt.Errorf("%w %v", errRunHasNotRan, id)
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StopAllRuns stops all running strategies
func (r *RunManager) StopAllRuns() ([]*RunSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	var resp []*RunSummary
	for i := range r.runs {
		if r.runs[i].IsRunning() {
			r.runs[i].Stop()
			sum, err := r.runs[i].GenerateSummary()
			if err != nil {
				return nil, err
			}
			resp = append(resp, sum)
		}
	}
	return resp, nil
}

// StartRun executes a strategy if found
func (r *RunManager) StartRun(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.runs {
		if !r.runs[i].MatchesID(id) {
			continue
		}
		switch {
		case r.runs[i].IsRunning():
			return fmt.Errorf("%w %v", errRunIsRunning, id)
		case r.runs[i].HasRan():
			return fmt.Errorf("%w %v", errAlreadyRan, id)
		default:
			return r.runs[i].ExecuteStrategy(false)
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StartAllRuns executes all strategies
func (r *RunManager) StartAllRuns() ([]uuid.UUID, error) {
	if r == nil {
		return nil, fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	executedRuns := make([]uuid.UUID, 0, len(r.runs))
	for i := range r.runs {
		if r.runs[i].HasRan() {
			continue
		}
		executedRuns = append(executedRuns, r.runs[i].MetaData.ID)
		err := r.runs[i].ExecuteStrategy(false)
		if err != nil {
			return nil, err
		}
	}

	return executedRuns, nil
}

// ClearRun removes a run from memory
func (r *RunManager) ClearRun(id uuid.UUID) error {
	if r == nil {
		return fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.runs {
		if !r.runs[i].MatchesID(id) {
			continue
		}
		if r.runs[i].IsRunning() {
			return fmt.Errorf("%w %v, currently running. Stop it first", errCannotClear, r.runs[i].MetaData.ID)
		}
		r.runs = append(r.runs[:i], r.runs[i+1:]...)
		return nil
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// ClearAllRuns removes all runs from memory
func (r *RunManager) ClearAllRuns() (clearedRuns, remainingRuns []*RunSummary, err error) {
	if r == nil {
		return nil, nil, fmt.Errorf("%w RunManager", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	for i := 0; i < len(r.runs); i++ {
		var run *RunSummary
		run, err = r.runs[i].GenerateSummary()
		if err != nil {
			return nil, nil, err
		}
		if r.runs[i].IsRunning() {
			remainingRuns = append(remainingRuns, run)
		} else {
			clearedRuns = append(clearedRuns, run)
			r.runs = append(r.runs[:i], r.runs[i+1:]...)
			i--
		}
	}
	return clearedRuns, remainingRuns, nil
}
