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

// Time window responses
const (
	MinimumSizeResponse = "reduce end date, increase granularity (interval) or increase deployable capital requirements"
	MaximumSizeResponse = "increase end date, decrease granularity (interval) or decrease deployable capital requirements"
)

// Reason defines a simple string type on what the strategy is doing
type Reason string

var (
	// Initial check errors:

	// ErrNilSignal indicates that a signal is nil.
	ErrNilSignal = errors.New("signal is nil")
	// ErrUnhandledSignal indicates that a signal of an unknown or unhandled
	// type was encountered.
	ErrUnhandledSignal = errors.New("signal type is unhandled")
	// ErrCannotGenerateSignal indicates that it was not possible to generate
	// the required signals.
	ErrCannotGenerateSignal = errors.New("cannot generate adequate signals")
	// ErrIntervalNotSupported indicates that the specified interval is not
	// supported by the function or method.
	ErrIntervalNotSupported = errors.New("interval currently not supported")
	// ErrInvalidUUID indicates that the provided UUID (universally unique identifier)
	// is invalid.
	ErrInvalidUUID = errors.New("invalid UUID")
	// ErrIsNil indicates that a value is nil, which means it is an empty or
	// uninitialized value.
	ErrIsNil = errors.New("is nil")
	// ErrNotFound indicates that a specified item or value was not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyRunning indicates that a process or operation is already
	// running and cannot be started again.
	ErrAlreadyRunning = errors.New("already running")
	// ErrNotRunning indicates that a process or operation is not currently
	// running and cannot be stopped or interrupted.
	ErrNotRunning = errors.New("not running")
	// ErrConfigIsNil indicates that the provided configuration is nil, which
	// means it is an empty or uninitialized value.
	ErrConfigIsNil = errors.New("configuration is nil")
	// ErrExchangeIsNil indicates that the provided exchange is nil, which means
	//  it is an empty or uninitialized value.
	ErrExchangeIsNil = errors.New("exchange is nil")
	// ErrReporterIsNil indicates that the provided strategy reporter is nil,
	// which means it is an empty or uninitialized value.
	ErrReporterIsNil = errors.New("strategy reporter is nil")
	// ErrInvalidAssetType indicates that the provided asset type is not
	// currently supported for spot trading pairs.
	ErrInvalidAssetType = errors.New("non spot trading pairs not currently supported") // TODO: Open up to all asset types.

	// Value errors:

	// ErrInvalidSlippage indicates that an invalid slippage percentage was
	// provided.
	ErrInvalidSlippage = errors.New("invalid slippage percentage")
	// ErrInvalidSpread indicates that an invalid spread percentage was provided.
	ErrInvalidSpread = errors.New("invalid spread percentage")
	// ErrMaxSpreadExceeded indicates that the maximum spread percentage was
	/// exceeded.
	ErrMaxSpreadExceeded = errors.New("max spread percentage exceeded")
	// ErrMaxImpactExceeded indicates that the maximum impact percentage was
	// exceeded.
	ErrMaxImpactExceeded = errors.New("impact percentage exceeded")
	// ErrMaxNominalExceeded indicates that the maximum nominal percentage was
	// exceeded.
	ErrMaxNominalExceeded = errors.New("nominal percentage exceeded")
	// ErrInvalidAmount indicates that an invalid amount was provided.
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrCannotSetAmount indicates that a specific amount cannot be set because
	//  the full amount bool is set.
	ErrCannotSetAmount = errors.New("specific amount cannot be set, full amount bool set")
	// ErrUnderMinimumAmount indicates that the provided amount is under the
	// minimum required amount.
	ErrUnderMinimumAmount = errors.New("amount is under the minimum requirements")
	// ErrOverMaximumAmount indicates that the provided amount is over the
	// maximum allowed amount.
	ErrOverMaximumAmount = errors.New("amount is over the maximum requirments")
	// ErrInvalidPriceLimit indicates that an invalid price limit was provided.
	ErrInvalidPriceLimit = errors.New("invalid price limit")
	// ErrPriceLimitExceeded indicates that the provided price limit was exceeded.
	ErrPriceLimitExceeded = errors.New("price limit exceeded")

	// Time errors:

	// ErrInvalidOperatingWindow indicates that the provided start to end time
	// window is less than the operating interval.
	ErrInvalidOperatingWindow = errors.New("start to end time window cannot be less than the operating interval")
	// ErrEndBeforeTimeNow indicates that the provided end time is before the
	// current time.
	ErrEndBeforeTimeNow = errors.New("end time is before time now")

	// Orderbook errors:

	// ErrOrderbookIsNil indicates that the provided orderbook is nil.
	ErrOrderbookIsNil = errors.New("orderbook is nil")
	// ErrExceedsLiquidity indicates that the requested operation exceeds the
	// total orderbook liquidity.
	ErrExceedsLiquidity = errors.New("exceeds total orderbook liquidity")

	// Order execution errors:

	// ErrInvalidRetryAttempts indicates that the provided number of retry
	// attempts is invalid.
	ErrInvalidRetryAttempts = errors.New("invalid retry attempts")
	// ErrNoBalance indicates that there is no balance available for the
	// requested operation.
	ErrNoBalance = errors.New("no balance")
	// ErrExceedsFreeBalance indicates that the requested operation exceeds the
	// current free balance.
	ErrExceedsFreeBalance = errors.New("exceeds current free balance")
	// ErrSubmitOrderIsNil indicates that the provided submit order is nil.
	ErrSubmitOrderIsNil = errors.New("submit order is nil")

	// Simulation errors:

	// ErrFullAmountSimulation indicates that the full amount cannot be
	// requested in a simulation.
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
	// TimeoutAction represents an action that indicates that a time limit has
	// been reached or exceeded.
	TimeoutAction struct {
		// EndTime specifies the time at which the time limit expired or was
		// reached.
		EndTime time.Time
	}
	// ErrorAction represents an action that indicates that an error has occurred.
	ErrorAction struct {
		// Error specifies the error that occurred.
		Error error
	}
	// MessageAction represents an action that sends a message to a specified
	// recipient.
	MessageAction struct {
		// Message specifies the message that should be sent.
		Message string
	}
	// WaitAction represents an action that indicates that the process or
	// operation should wait until a specified time.
	WaitAction struct {
		// Until specifies the time at which the wait should end.
		Until string
	}
	// SignalAction represents an action that sends a signal to a specified
	// recipient.
	SignalAction struct {
		// Reason specifies the reason or purpose for the signal.
		Reason interface{}
	}
	// OrderAction represents an action that involves an order, such as
	// submitting an order or receiving a response to an order.
	OrderAction struct {
		// Submit specifies the order that should be submitted.
		Submit *order.Submit
		// Response specifies the response to a previously submitted order.
		Response *order.SubmitResponse
		// Orderbook specifies the orderbook movement that *theoretically*
		// performed.
		Orderbook *orderbook.Movement
	}
	// RegisterAction represents an action that registers a strategy for into
	// the manager.
	RegisterAction struct {
		// Time specifies the time at which the registration occurred.
		Time time.Time
	}
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
	GetEnd(suppress bool) <-chan time.Time
	// OnSignal is a strategy-defined function that handles the data that is
	// returned from `GetSignal()`. This method is defined on the `strategy`
	// type in the `specific individual _wrapper.go` file.
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
	ReportStart(description Descriptor)
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
	// GetID returns a loaded uuid. This method is defined on the `Requirement`
	// type in the `requirement.go` file.
	GetID() uuid.UUID
	// GetDescription returns a strategy defined type that defines basic
	// operating information. This method is defined on the `strategy wrapper`
	// type in the `specific individual _wrapper.go` file.
	GetDescription() Descriptor
	// CanContinuePassedEnd returns if the strategy will continue to operate
	// passed expected final date/time if the strategy for example does not
	// deplete all funds.
	CanContinuePassedEnd() bool
}

// Descriptor interface allows a strategy defined type to be passed back to rpc
// and in logger params.
type Descriptor interface {
	// Stringer functionality allows for short descriptions
	fmt.Stringer
}

// Details define base level information
type Details struct {
	ID         uuid.UUID
	Registered time.Time
	Running    bool
	Strategy   string
}
