package common

import (
	"context"
	"errors"
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

// Time window responses
const (
	MinimumSizeResponse = "reduce end date, increase granularity (interval) or increase deployable capital requirements"
	MaximumSizeResponse = "increase end date, decrease granularity (interval) or decrease deployable capital requirements"
)

// Reason defines a simple string type on what the strategy is doing
type Reason string

var (
	// Initial check errors
	ErrNilSignal            = errors.New("signal is nil")
	ErrUnhandledSignal      = errors.New("signal type is unhandled")
	ErrCannotGenerateSignal = errors.New("cannot generate adequate signals")
	ErrIntervalNotSupported = errors.New("interval currently not supported")
	ErrInvalidUUID          = errors.New("invalid UUID")
	ErrIsNil                = errors.New("is nil")
	ErrNotFound             = errors.New("not found")
	ErrAlreadyRunning       = errors.New("already running")
	ErrNotRunning           = errors.New("not running")
	ErrConfigIsNil          = errors.New("configuration is nil")
	ErrExchangeIsNil        = errors.New("exchange is nil ")
	ErrReporterIsNil        = errors.New("strategy reporter is nil")
	ErrInvalidAssetType     = errors.New("non spot trading pairs not currently supported") // TODO: Open up to all asset types.

	// Value errors
	ErrInvalidSlippage    = errors.New("invalid slippage percentage")
	ErrInvalidSpread      = errors.New("invalid spread percentage")
	ErrMaxSpreadExceeded  = errors.New("max spread percentage exceeded")
	ErrMaxImpactExceeded  = errors.New("impact percentage exceeded")
	ErrMaxNominalExceeded = errors.New("nominal percentage exceeded")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrCannotSetAmount    = errors.New("specific amount cannot be set, full amount bool set")
	ErrUnderMinimumAmount = errors.New("amount is under the minimum requirements")
	ErrOverMaximumAmount  = errors.New("amount is over the maximum requirments")
	ErrInvalidPriceLimit  = errors.New("invalid price limit")
	ErrPriceLimitExceeded = errors.New("price limit exceeded")

	// Time errors
	ErrInvalidOperatingWindow = errors.New("start to end time window cannot be less than the operating interval")
	ErrEndBeforeTimeNow       = errors.New("end time is before time now")

	// Orderbook errors
	ErrOrderbookIsNil   = errors.New("orderbook is nil")
	ErrExceedsLiquidity = errors.New("exceeds total orderbook liquidity")

	// Order execution errors
	ErrInvalidRetryAttempts = errors.New("invalid retry attempts")
	ErrNoBalance            = errors.New("no balance")
	ErrExceedsFreeBalance   = errors.New("exceeds current free balance")
	ErrSubmitOrderIsNil     = errors.New("submit order is nil")

	// Simulation errors
	ErrFullAmountSimulation = errors.New("full amount cannot be requested in simulation, for now")

	errStrategyDescriptionIsEmpty = errors.New("strategy description/name is empty")
	errRequirementIsNil           = errors.New("requirement is nil")
	errActivitiesIsNil            = errors.New("activities is nil")
	errIDAlreadySet               = errors.New("id already set")
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
	// GetNext returns the next execution time for the strategy.
	GetNext() time.Time
	// GetReporter returns a channel that gives you a full report or summary of
	// action as soon as it occurs.
	GetReporter(verbose bool) (<-chan *Report, error)
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
	ReportOrder(OrderAction)
	// ReportStart is called when the strategy is accepted and run and is
	// waiting for signals and sends a report to a reporter receiver. This
	// method is defined on the `Activities` type in the `activities.go` file.
	ReportStart(description string)
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

	GetID() uuid.UUID

	GetDescription() string
}

// Details define base level information
type Details struct {
	ID         uuid.UUID
	Registered time.Time
	Running    bool
	Strategy   string
}
