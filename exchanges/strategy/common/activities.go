package common

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Activities defines a holder for strategy activity and reportable actions.
type Activities struct {
	id         uuid.UUID
	strategy   string
	simulation bool
	reporter   chan *Report
	// TODO: Add the ability to store operations in line with backtester.
	verbose bool
}

// NewActivities returns an Activities holder (POC). NOTE: For now this is
// designed to be single threaded and methods should not be called concurrently.
func NewActivities(strategy string, simulation bool) (*Activities, error) {
	if strategy == "" {
		return nil, errStrategyDescriptionIsEmpty
	}
	return &Activities{
		uuid.Nil, // This will be set via the strategy manager.
		strategy,
		simulation,
		make(chan *Report, defaultReporterBuffer),
		false,
	}, nil
}

// getReporter allows separation of channel position so it can be nil'd out on
// activities field.
func (r *Activities) getReporter(verbose bool) (<-chan *Report, error) {
	if r == nil {
		return nil, errActivitiesIsNil
	}
	if r.reporter == nil {
		return nil, ErrReporterIsNil
	}
	r.verbose = verbose
	return r.reporter, nil
}

// Send sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver. NOTE: This should not be called
// concurrently.
func (r *Activities) send(reason Reason, action interface{}, complete bool) {
	if r == nil || r.reporter == nil {
		return
	}
	select {
	case r.reporter <- &Report{r.id, r.strategy, action, complete, reason, time.Now()}:
	default:
	}

	// TODO: Append to historical list in line with backtester operations.

	if complete {
		close(r.reporter)
		r.reporter = nil
	}
}

// ReportComplete is called when a strategy has completed sufficiently. This
// will alert a receiver that it has completed and will close the reporting
// channel.
func (r *Activities) ReportComplete() {
	r.send(Complete, nil, true)
}

// ReportTimeout is called when a strategy has timed-out and it has exceeded its
// operating time. This will alert a receiver that it has completed and will
// close the reporting channel.
func (r *Activities) ReportTimeout(end time.Time) {
	r.send(TimeOut, TimeoutAction{EndTime: end}, true)
}

// ReportFatalError is called when a strategy has errored and it cannot continue
// operations. This will alert a receiver that it has completed and will close
// the reporting channel.
func (r *Activities) ReportFatalError(err error) {
	if err == nil {
		panic("fatal error should not be nil")
	}
	r.send(FatalError, ErrorAction{Error: err}, true)
}

// ReportContextDone is called when a context has timed-out or has been
// cancelled and cannot continue operations. This will alert a receiver that it
// has completed and will close the reporting channel.
func (r *Activities) ReportContextDone(err error) {
	r.send(ContextDone, ErrorAction{Error: err}, true)
}

// ReportShutdown is called when the strategy has been shutdown and cannot
// continue operations. This will alert a receiver that it has completed and
// will close the reporting channel.
func (r *Activities) ReportShutdown() {
	r.send(Shutdown, nil, true)
}

// ReportInfo is called when the strategy wants to send relevant information to
// a reporter receiver.
func (r *Activities) ReportInfo(message string) {
	r.send(Info, MessageAction{Message: message}, false)
}

// ReportOrder is called when the strategy wants to send order execution
// information to a reporter receiver.
func (r *Activities) ReportOrder(action OrderAction) {
	var payload interface{}
	if r.verbose {
		payload = action
	} else {
		payload = MessageAction{
			Message: fmt.Sprintf("AverageOrderCost: %v Amount: %v Side: %s Status: %s Type: %s",
				action.Orderbook.AverageOrderCost,
				action.Response.Amount,
				action.Response.Side,
				action.Response.Status,
				action.Response.Type,
			),
		}
	}
	r.send(OrderExecution, payload, false)
}

// ReportStart is called when the strategy is accepted and run and is waiting
// for signals and sends a report to a reporter receiver.
func (r *Activities) ReportStart(description string) {
	if description == "" {
		return
	}
	r.send(Start, MessageAction{Message: description}, false)
}

// ReportRegister is called when the strategy is registered with the manager
// and sends a report to a reporter receiver.
func (r *Activities) ReportRegister() {
	if !r.verbose {
		return
	}
	r.send(Registered, nil, false)
}

// ReportWait is called to notify when the next signal is going to occur and
// sends a report to a reporter receiver.
func (r *Activities) ReportWait(next time.Time) {
	if next.IsZero() || !r.verbose {
		return
	}
	r.send(Wait, WaitAction{Until: time.Until(next).String()}, false)
}

// ReportAcceptedSignal is called to notify when a signal that has been
// generated is accepted and allows a strategy to execute pre-defined code and
// sends a report to a reporter receiver.
func (r *Activities) ReportAcceptedSignal(reason interface{}) {
	if !r.verbose {
		return
	}
	r.send(SignalAccepted, SignalAction{Reason: reason}, false)
}

// ReportRejectedSignal is called to notify when a signal that has been
// generated is rejected and will not allow a strategy to execute pre-defined
// code and sends a report to a reporter receiver.
func (r *Activities) ReportRejectedSignal(reason interface{}) {
	r.send(SignalRejection, SignalAction{Reason: reason}, false)
}
