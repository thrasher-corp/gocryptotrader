package engine

import (
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/writer"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
	"strings"
	"sync"
	"time"
)

var (
	errRunNotFound         = errors.New("run not found")
	errRunAlreadyMonitored = errors.New("run already monitored")
	errAlreadyRan          = errors.New("run already ran")
	errRunHasNotRan        = errors.New("run hasn't ran yet")
	errRunIsRunning        = errors.New("run is already running")
	errCannotClear         = errors.New("cannot clear run")
	errNoLoggerSetup       = errors.New("run is missing log storage")
)

// SetupRunManager creates a run manager to allow the backtester to manage multiple strategies
func SetupRunManager() *RunManager {
	return &RunManager{}
}

// AddRun adds a run to the manager
func (r *RunManager) AddRun(b *BackTest) error {
	if b == nil {
		return fmt.Errorf("%w BackTest", gctcommon.ErrNilPointer)
	}
	r.m.Lock()
	defer r.m.Unlock()
	if b.MetaData.ID == "" {
		if b.Strategy == nil {
			return fmt.Errorf("%w Backtest strategy not setup", gctcommon.ErrNilPointer)
		}
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		b.MetaData = RunMetaData{
			ID:         id.String(),
			Strategy:   b.Strategy.Name(),
			DateLoaded: time.Now(),
			// TODO set livetest & realorder after merge
		}
	}
	for i := range r.runs {
		if r.runs[i].MetaData.ID == b.MetaData.ID {
			return fmt.Errorf("%w %s %s", errRunAlreadyMonitored, b.MetaData.ID, b.MetaData.Strategy)
		}
	}
	var err error
	b.logHolder, err = writer.SetupWriter(b.MetaData.ID)
	if err != nil {
		return err
	}
	err = log.AddWriter(b.logHolder)
	if err != nil {
		return err
	}
	r.runs = append(r.runs, b)
	return nil
}

// List details all backtesting/live strategy runs
func (r *RunManager) List() []RunSummary {
	r.m.Lock()
	defer r.m.Unlock()
	var resp []RunSummary
	for i := range r.runs {
		sum := r.runs[i].GenerateSummary()
		resp = append(resp, *sum)
	}
	return resp
}

// GetSummary returns details about a backtesting/live strategy run
func (r *RunManager) GetSummary(id string) (*RunSummary, error) {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.runs {
		if r.runs[i].MetaData.ID == id {
			return r.runs[i].GenerateSummary(), nil
		}
	}
	return nil, fmt.Errorf("%s %w", id, errRunNotFound)
}

// StopRun stops a backtesting/live strategy run if enabled, this will run CloseAllPositions
func (r *RunManager) StopRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.runs {
		if r.runs[i].MetaData.ID == id {
			switch {
			case !r.runs[i].MetaData.Closed && !r.runs[i].MetaData.DateStarted.IsZero():
				r.runs[i].Stop()
				return nil
			case !r.runs[i].MetaData.Closed && r.runs[i].MetaData.DateStarted.IsZero():
				return fmt.Errorf("%w %v", errRunHasNotRan, id)
			default:
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
	for i := range r.runs {
		if !r.runs[i].MetaData.Closed && !r.runs[i].MetaData.DateStarted.IsZero() {
			r.runs[i].Stop()
			sum := r.runs[i].GenerateSummary()
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
	for i := range r.runs {
		if r.runs[i].MetaData.ID == id {
			switch {
			case !r.runs[i].MetaData.Closed && !r.runs[i].MetaData.DateStarted.IsZero():
				return fmt.Errorf("%w %v", errRunIsRunning, id)
			case !r.runs[i].MetaData.Closed && r.runs[i].MetaData.DateStarted.IsZero():
				var startedGo sync.WaitGroup
				startedGo.Add(1)
				go func() {
					startedGo.Done()
					err := r.runs[i].ExecuteStrategy()
					if err != nil {
						log.Error(common.Backtester, err)
					}
				}()
				startedGo.Wait()
				return nil
			default:
				return fmt.Errorf("%w %v", errAlreadyRan, id)
			}
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// StartAllRuns executes all strategies
func (r *RunManager) StartAllRuns() []*RunSummary {
	r.m.Lock()
	defer r.m.Unlock()
	var resp []*RunSummary
	var startedGo sync.WaitGroup
	for i := range r.runs {
		if !r.runs[i].MetaData.Closed && r.runs[i].MetaData.DateStarted.IsZero() {
			startedGo.Add(1)
			go func() {
				startedGo.Done()
				err := r.runs[i].ExecuteStrategy()
				if err != nil {
					log.Error(common.Backtester, err)
				}
			}()
			resp = append(resp, r.runs[i].GenerateSummary())
		}
	}
	startedGo.Wait()

	return resp
}

// ClearRun removes a run from memory
func (r *RunManager) ClearRun(id string) error {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.runs {
		if r.runs[i].MetaData.ID == id {
			if !r.runs[i].MetaData.Closed && !r.runs[i].MetaData.DateStarted.IsZero() {
				return fmt.Errorf("%w %v, currently running. Stop it first", errCannotClear, r.runs[i].MetaData.ID)
			}
			err := log.RemoveWriter(r.runs[i].logHolder)
			if err != nil && errors.Is(err, log.ErrWriterNotFound) {
				return err
			}
			r.runs = append(r.runs[:i], r.runs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%s %w", id, errRunNotFound)
}

// ClearAllRuns removes all runs from memory
func (r *RunManager) ClearAllRuns() (clearedRuns, remainingRuns []*RunSummary, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.runs {
		run := r.runs[i].GenerateSummary()
		if err != nil {
			return nil, nil, err
		}
		if !r.runs[i].MetaData.Closed && !r.runs[i].MetaData.DateStarted.IsZero() {
			remainingRuns = append(remainingRuns, run)
		} else {
			clearedRuns = append(clearedRuns, run)
			err = log.RemoveWriter(r.runs[i].logHolder)
			if err != nil && errors.Is(err, log.ErrWriterNotFound) {
				return nil, nil, err
			}
			r.runs = append(r.runs[:i], r.runs[i+1:]...)
		}
	}
	return clearedRuns, remainingRuns, nil
}

// ReportLogs returns the full logs from a run
func (r *RunManager) ReportLogs(id string) (string, error) {
	r.m.Lock()
	defer r.m.Unlock()
	id = strings.ToLower(id)
	for i := range r.runs {
		if r.runs[i].MetaData.ID == id {
			if r.runs[i].logHolder == nil {
				return "", fmt.Errorf("%s %w", id, errNoLoggerSetup)
			}
			return r.runs[i].logHolder.String(), nil
		}
	}
	return "", fmt.Errorf("%s %w", id, errRunNotFound)
}
