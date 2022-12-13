package common

import (
	"context"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Requirement defines the base requirements for managing the operation of the
// strategy so most of the internals can be abstracted away from individual
// definitions.
type Requirement struct {
	registered time.Time
	strategy   string
	Activities
	wg               sync.WaitGroup
	shutdown         chan struct{}
	running          bool
	OperateBeyondEnd bool
	mtx              sync.Mutex
}

// Run oversees the deployment of the current strategy adhering to policies,
// limits, signals and schedules.
func (r *Requirement) Run(ctx context.Context, strategy Requirements) error {
	if r == nil {
		return errRequirementIsNil
	}

	if strategy == nil {
		return ErrIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.running {
		return ErrAlreadyRunning
	}

	r.shutdown = make(chan struct{})
	r.running = true
	r.wg.Add(1)
	go r.deploy(ctx, strategy)
	return nil
}

// deploy is the core routine that handles strategy functionality and lifecycle
func (r *Requirement) deploy(ctx context.Context, strategy Requirements) {
	defer func() { r.wg.Done(); _ = r.Stop() }()
	strategy.ReportStart(strategy.GetDescription())
	for {
		select {
		case signal := <-strategy.GetSignal():
			complete, err := strategy.OnSignal(ctx, signal)
			if err != nil {
				log.Errorf(log.Strategy, "ID: [%s] has failed %v handling signal %T", strategy.GetID(), err, signal)
				strategy.ReportFatalError(err)
				return
			}
			if complete {
				strategy.ReportComplete()
				return
			}
			strategy.ReportWait(strategy.GetNext())
		case end := <-strategy.GetEnd(strategy.CanContinuePassedEnd()):
			strategy.ReportTimeout(end)
			return
		case <-ctx.Done():
			log.Warnf(log.Strategy, "ID: [%s] context has finished: %v", strategy.GetID(), ctx.Err())
			strategy.ReportContextDone(ctx.Err())
			return
		case <-r.shutdown:
			strategy.ReportShutdown()
			return
		}
	}
}

// Stop stops the strategy and releases routine
func (r *Requirement) Stop() error {
	if r == nil {
		return errRequirementIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if !r.running {
		return ErrNotRunning
	}

	close(r.shutdown)
	r.running = false
	r.wg.Wait()
	return nil
}

// GetState returns the strategy details
func (r *Requirement) GetDetails() (*Details, error) {
	if r == nil {
		return nil, errRequirementIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()
	return &Details{r.id, r.registered, r.running, r.strategy}, nil
}

// GetReporter returns a channel that allows the broadcast of activity from a
// specific strategy.
func (r *Requirement) GetReporter(verbose bool) (<-chan *Report, error) {
	if r == nil {
		return nil, errRequirementIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()
	return r.getReporter(verbose)
}

// LoadID loads an externally generated uuid for tracking.
func (r *Requirement) LoadID(id uuid.UUID) error {
	if r == nil {
		return errRequirementIsNil
	}

	if id.IsNil() {
		return ErrInvalidUUID
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if !r.id.IsNil() {
		return errIDAlreadySet
	}
	r.id = id
	return nil
}

// GetID returns the ID for the loaded strategy
func (r *Requirement) GetID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return r.id
}

// CanContinuePassedEnd returns if the strategy can continue to operated passed
// an end date/time.
func (r *Requirement) CanContinuePassedEnd() bool {
	if r == nil {
		return false
	}
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return r.OperateBeyondEnd
}
