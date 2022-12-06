package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// defaultReporterBuffer defines the default buffer level accounts for when
// there could be a slow receiver on the gRPC writing.
const defaultReporterBuffer = 1000

// Simulation tags a strategy as being in simulation mode
const Simulation = "SIMULATION"

// Consts below define consistent reasons for a report to be generated
const (
	Complete        Reason = "COMPLETE"
	TimeOut         Reason = "TIMEOUT"
	FatalError      Reason = "FATAL ERROR"
	ContextDone     Reason = "CONTEXT DONE"
	Shutdown        Reason = "SHUTDOWN"
	Info            Reason = "INFO"
	OrderExecution  Reason = "ORDER EXECUTION"
	Start           Reason = "START"
	Registered      Reason = "REGISTERED"
	Wait            Reason = "WAITING"
	SignalAccepted  Reason = "SIGNAL ACCEPTED"
	SignalRejection Reason = "SIGNAL REJECTED"
)

var (
	ErrInvalidUUID    = errors.New("invalid UUID")
	ErrIsNil          = errors.New("strategy is nil")
	ErrNotFound       = errors.New("strategy not found")
	ErrAlreadyRunning = errors.New("strategy is already running")
	ErrNotRunning     = errors.New("strategy not running")
	ErrConfigIsNil    = errors.New("strategy configuration is nil")
	ErrReporterIsNil  = errors.New("strategy reporter is nil")

	errStrategyDescriptionIsEmpty = errors.New("strategy description/name is empty")
	errRequirementIsNil           = errors.New("requirement is nil")
	errActivitiesIsNil            = errors.New("activities is nil")
)

// Report defines a strategies actions at a certain point in time for ad-hoc
// notification or broadcast.
type Report struct {
	ID       uuid.UUID   `json:"id"`
	Strategy string      `json:"strategy"`
	Action   interface{} `json:"action,omitempty"`
	Finished bool        `json:"finished,omitempty"`
	Reason   Reason      `json:"reason"`
	Time     time.Time   `json:"time"`
}

// Reason defines a simple string type on what the strategy is doing
type Reason string

// Defines reportable actions that this strategy might undertake, struct types
// used for outbound json marshalling.
type (
	TimeoutAction struct{ EndTime time.Time }
	ErrorAction   struct{ Error error }
	MessageAction struct{ Message string }
	WaitAction    struct{ Until string }
	SignalAction  struct{ Reason interface{} }
	OrderAction   struct {
		Submit    *order.Submit         `json:"submit,omitempty"`
		Response  *order.SubmitResponse `json:"response,omitempty"`
		Orderbook *orderbook.Movement   `json:"orderbook,omitempty"`
	}
	RegisterAction struct{ Time time.Time }
)

// Requirements defines the baseline functionality for managing strategies
type Requirements interface {
	// Run checks the base requirement state and generates a routine to handle
	// signals, shutdown, context done and other activities for the strategy, as
	// defined by the implementing type.
	Run(ctx context.Context, strategy Requirements) error
	// Stop stops the current operating strategy, as defined by the implementing
	// type.
	Stop() error
	// GetDetails returns the base requirement details, as defined by the
	// implementing type.
	GetDetails() (*Details, error)
	// GetSignal is a strategy-defined function that alerts the deploy routine
	// to call the `OnSignal` method, which will handle the data/change
	// correctly. The `Scheduler` type implements the default `GetSignal`
	// method.
	GetSignal() <-chan interface{}
	// GetEnd alerts the deploy routine to return and finish when the strategy
	// is scheduled to end. The `Scheduler` type implements the default
	// `GetEnd` method. This can return a `nil` channel with no consequences
	// if a strategy has no set end date.
	GetEnd() <-chan time.Time
	// OnSignal is a strategy-defined function that handles the data that is
	// returned from `GetSignal()`.
	OnSignal(ctx context.Context, signal interface{}) (bool, error)
	// String is a strategy-defined function that returns basic information.
	String() string
	// GetNext returns the next execution time for the strategy.
	GetNext() time.Time
	// GetReporter returns a channel that gives you a full report or summary of
	// action as soon as it occurs.
	GetReporter() (<-chan *Report, error)
	// ReportComplete is called when a strategy has completed sufficiently.
	// This will alert a receiver that it has completed and will close the
	// reporting channel. This method is defined on the `Activities` type in
	// the `activities.go` file.
	ReportComplete()
	// ReportTimeout is called when a strategy has timed out and exceeded its
	// operating time. This will alert a receiver that it has completed and
	// will close the reporting channel. This method is defined on the
	// `Activities` type in the `activities.go` file.
	ReportTimeout(end time.Time)
	// ReportFatalError is called when a strategy has errored and cannot
	// continue operations. This will alert a receiver that it has completed
	// and will close the reporting channel. This method is defined on the
	// `Activities` type in the `activities.go` file.
	ReportFatalError(err error)
	// ReportContextDone is called when a context has timed out or has been
	// cancelled and cannot continue operations. This will alert a receiver
	// that it has completed and will close the reporting channel. This
	// method is defined on the `Activities` type in the `activities.go` file.
	ReportContextDone(err error)
	// ReportShutdown is called when the strategy has been shutdown and cannot
	// continue operations. This will alert a receiver
	// that it has completed and will close the reporting channel. This
	// method is defined on the `Activities` type in the `activities.go` file.
	ReportShutdown()
	// ReportInfo is called when the strategy wants to send relevant information
	// to a reporter receiver. This method is defined on the `Activities` type
	// in the `activities.go` file.
	ReportInfo(message string)
	// ReportOrder is called when the strategy wants to send order execution
	// information to a reporter receiver. This method is defined on the
	// `Activities` type in the `activities.go` file.
	ReportOrder(*OrderAction)
	// ReportStart is called when the strategy is accepted and run and is
	// waiting for signals and sends a report to a reporter receiver. This
	// method is defined on the `Activities` type in the `activities.go` file.
	ReportStart(data fmt.Stringer)
	// ReportRegister is called when the strategy is registered with the manager
	// and sends a report to a reporter receiver. As defined as method on
	// 'Activities' type in activities.go.
	ReportRegister()
	// ReportWait is called to notify when the next signal is going to occur and
	// sends a report to a reporter receiver. This method is defined on the
	// `Activities` type in the `activities.go` file.
	ReportWait(next time.Time)
	// ReportAcceptedSignal is called to notify when a signal that has been
	// generated is accepted and allows a strategy to execute pre-defined code
	// and sends a report to a reporter receiver. This method is defined on the
	// `Activities` type in the `activities.go` file.
	ReportAcceptedSignal(reason interface{})
	// ReportRejectedSignal is called to notify when a signal that has been
	// generated is rejected and will not allow a strategy to execute pre-defined
	// code and sends a report to a reporter receiver. This method is defined
	// on the `Activities` type in the `activities.go` file.
	ReportRejectedSignal(reason interface{})
	// LoadID loads an externally generated uuid for tracking. This method is
	// defined on the `Requirement` type in the `requirement.go` file.
	LoadID(id uuid.UUID) error
}

// Details define base level information
type Details struct {
	ID         uuid.UUID
	Registered time.Time
	Running    bool
	Strategy   string
}
