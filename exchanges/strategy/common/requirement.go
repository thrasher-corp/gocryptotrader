package common

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

const SimulationTag = "SIMULATION"

var errRequirementIsNil = errors.New("requirement is nil")

// Requirements define baseline functionality for strategy management
type Requirements interface {
	// Run checks the base requirement state and generates a routine to handle
	// signals, shutdown, context done and other activities for the strategy as
	// defined in method on type 'Requirement'.
	Run(ctx context.Context, strategy Runner) error
	// Stop stops the current operating strategy as defined in method on type
	// 'Requirement'.
	Stop() error
	// GetDetails returns the base requirement detail as defined method on type
	// 'Requirement'.
	GetDetails() (*Details, error)

	Runner
}

// Runner defines baseline functionality to handle strategy activities
type Runner interface {
	// GetSignal is a strategy defined function that alerts the deploy routine
	// as defined method on type 'Requirement' to call 'OnSignal' method which
	// will handle the data/change correctly. Type 'Scheduler' implements the
	// default 'GetSignal' method.
	GetSignal() <-chan interface{}

	// GetEnd alerts the deploy routine as defined method on type 'Requirement'
	// to return and finish when the strategy is scheduled to end. See type
	// 'Scheduler' implements the default 'GetEnd' method. This can return a nil
	// map type with no consequences.
	GetEnd() <-chan time.Time

	// GetSignal is a strategy defined function that handles the data that is
	// returned from GetSignal().
	OnSignal(ctx context.Context, signal interface{}) (bool, error)

	// String is a strategy defined function that returns basic information
	String() string

	Activity
}

// Requirement defines the base requirements for managing the operation of the
// strategy so most of the internals can be abstracted away from individual
// definitions.
type Requirement struct {
	id         uuid.UUID
	registered time.Time
	strategy   string
	Activities
	wg       sync.WaitGroup
	shutdown chan struct{}
	running  bool
	mtx      sync.Mutex
}

// Run oversees the deployment of the current strategy adhering to policies,
// limits, signals and schedules.
func (r *Requirement) Run(ctx context.Context, strategy Runner) error {
	if strategy == nil {
		return ErrIsNil
	}

	if r == nil {
		return errRequirementIsNil
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
func (r *Requirement) deploy(ctx context.Context, strategy Runner) {
	defer func() { r.wg.Done(); _ = r.Stop() }()
	strategy.ReportStart(strategy)
	for {
		select {
		case signal := <-strategy.GetSignal():
			complete, err := strategy.OnSignal(ctx, signal)
			if err != nil {
				strategy.ReportFatalError(err)
				return
			}
			if complete {
				strategy.ReportComplete()
				return
			}
		case end := <-strategy.GetEnd():
			strategy.ReportTimeout(end)
			return
		case <-ctx.Done():
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

// Details define base level information
type Details struct {
	ID         uuid.UUID
	Registered time.Time
	Running    bool
	Strategy   string
}

// GetState returns the state of the strategy
func (r *Requirement) GetDetails() (*Details, error) {
	if r == nil {
		return nil, errRequirementIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	return &Details{
		ID:         r.id,
		Registered: r.registered,
		Running:    r.running,
		Strategy:   r.strategy,
	}, nil
}

// GetReporter returns a channel that allows the broadcast of activity from a
// specific strategy.
func (r *Requirement) GetReporter() (<-chan *Report, error) {
	if r == nil {
		return nil, errRequirementIsNil
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.reporter == nil {
		return nil, ErrReporterIsNil
	}
	return r.reporter, nil
}
