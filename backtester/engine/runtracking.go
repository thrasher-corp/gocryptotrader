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
			r.Runs[i].Stop()
			return nil
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StartRun executes a strategy if found
func (r *RunManager) StartRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.Runs {
		if r.Runs[i].RunMetaData.ID == id {
			return r.Runs[i].ExecuteStrategy()
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}
