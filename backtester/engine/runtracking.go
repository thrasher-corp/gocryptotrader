package engine

import (
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"strings"
	"time"
)

var (
	errRunNotFound         = errors.New("run not found")
	errRunAlreadyMonitored = errors.New("run already monitored")
	errAlreadyRan          = errors.New("run already ran")
	errCannotClear         = errors.New("cannot clear run")
)

// GenerateSummary creates a summary of a backtesting/live strategy run
// this summary contains many details of a run
func (bt *BackTest) GenerateSummary() (*RunSummary, error) {
	return &RunSummary{
		Identifier: bt.RunMetaData,
	}, nil
}

// SetupRunManager creates a run manager to allow the backtester to manage multiple strategies
func SetupRunManager() *RunManager {
	return &RunManager{}
}

// AddRun adds a run to the manager
func (r *RunManager) AddRun(b *BackTest) error {
	r.m.Lock()
	defer r.m.Unlock()
	if b.RunMetaData.ID == "" {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		b.RunMetaData = RunMetaData{
			ID:         id.String(),
			Strategy:   b.Strategy.Name(),
			DateLoaded: time.Now(),
			// TODO set livetest & realorder after merge
		}
	}
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == b.RunMetaData.ID {
			return fmt.Errorf("%w %s %s", errRunAlreadyMonitored, b.RunMetaData.ID, b.RunMetaData.Strategy)
		}
	}
	r.Runs = append(r.Runs, b)
	return nil
}

// List details all backtesting/live strategy runs
func (r *RunManager) List() ([]RunSummary, error) {
	r.m.Lock()
	defer r.m.Unlock()
	var resp []RunSummary
	for i := range r.Runs {
		sum, err := r.Runs[i].GenerateSummary()
		if err != nil {
			return nil, err
		}
		resp = append(resp, *sum)
	}
	return resp, nil
}

// GetSummary returns details about a backtesting/live strategy run
func (r *RunManager) GetSummary(id string) (*RunSummary, error) {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			return r.Runs[i].GenerateSummary()
		}
	}
	return nil, fmt.Errorf("%s %w", id, errRunNotFound)
}

// StopRun stops a backtesting/live strategy run if enabled, this will run CloseAllPositions
func (r *RunManager) StopRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			if !r.Runs[i].RunMetaData.Closed && !r.Runs[i].RunMetaData.DateStarted.IsZero() {
				r.Runs[i].Stop()
				return nil
			} else {
				return fmt.Errorf("%w %v", errAlreadyRan, id)
			}
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StopAllRuns stops all running strategies
func (r *RunManager) StopAllRuns() ([]*RunSummary, error) {
	r.m.Lock()
	defer r.m.Unlock()
	var resp []*RunSummary
	for i := range r.Runs {
		if !r.Runs[i].RunMetaData.Closed && !r.Runs[i].RunMetaData.DateStarted.IsZero() {
			r.Runs[i].Stop()
			sum, err := r.Runs[i].GenerateSummary()
			if err != nil {
				return nil, err
			}
			resp = append(resp, sum)
		}
	}
	return resp, nil
}

// StartRun executes a strategy if found
func (r *RunManager) StartRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			switch {
			case !r.Runs[i].RunMetaData.Closed && r.Runs[i].RunMetaData.DateStarted.IsZero():
				return r.Runs[i].ExecuteStrategy()
			case r.Runs[i].RunMetaData.Closed && !r.Runs[i].RunMetaData.DateStarted.IsZero():
				return fmt.Errorf("%w %v", errAlreadyRan, id)
			}
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StartAllRuns executes all strategies
func (r *RunManager) StartAllRuns() ([]*RunSummary, error) {
	r.m.Lock()
	defer r.m.Unlock()
	var resp []*RunSummary
	for i := range r.Runs {
		if !r.Runs[i].RunMetaData.Closed && r.Runs[i].RunMetaData.DateStarted.IsZero() {
			err := r.Runs[i].ExecuteStrategy()
			if err != nil {
				return nil, err
			}
			sum, err := r.Runs[i].GenerateSummary()
			if err != nil {
				return nil, err
			}
			resp = append(resp, sum)
		}
	}

	return resp, nil
}

// ClearRun removes a run from memory
func (r *RunManager) ClearRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			if !r.Runs[i].RunMetaData.Closed && !r.Runs[i].RunMetaData.DateStarted.IsZero() {
				return fmt.Errorf("%w %v, currently running. Stop it first", errCannotClear, r.Runs[i].RunMetaData.ID)
			}
			r.Runs = append(r.Runs[:i], r.Runs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// ClearAllRuns removes all runs from memory
func (r *RunManager) ClearAllRuns() (clearedRuns, remainingRuns []*RunSummary, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.Runs {
		var run *RunSummary
		run, err = r.Runs[i].GenerateSummary()
		if err != nil {
			return nil, nil, err
		}
		if !r.Runs[i].RunMetaData.Closed && !r.Runs[i].RunMetaData.DateStarted.IsZero() {
			remainingRuns = append(remainingRuns, run)
		} else {
			clearedRuns = append(clearedRuns, run)
		}
	}
	r.Runs = []*BackTest{}
	return clearedRuns, remainingRuns, nil
}

// ReportLogs returns the full logs from a run
func (r *RunManager) ReportLogs(id string) (string, error) {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			return r.Runs[i].logHolder.String(), nil
		}
	}
	return "", fmt.Errorf("%s %w", id, errRunNotFound)
}
