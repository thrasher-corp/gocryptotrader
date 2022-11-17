package common

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var errStrategyDescriptionIsEmpty = errors.New("strategy description is empty")

// ExecutedOrder holds order execution details
type ExecutedOrder struct {
	Submit    *order.Submit         `json:"submit,omitempty"`
	Response  *order.SubmitResponse `json:"response,omitempty"`
	Orderbook *orderbook.Movement   `json:"orderbook,omitempty"`
}

// Activities defines a holder for strategy activity and reportable actions.
type Activities struct {
	id         uuid.UUID
	strategy   string
	simulation bool
	reporter   chan *Report

	// TODO: Shell out operations and histories.
}

// NewReporter returns a Activities holder.
func NewActivities(strategy string, id uuid.UUID, simulation bool) (*Activities, error) {
	if strategy == "" {
		return nil, errStrategyDescriptionIsEmpty
	}
	if id.IsNil() {
		return nil, ErrInvalidUUID
	}
	return &Activities{
		strategy:   strategy,
		id:         id,
		simulation: simulation,
		reporter:   make(chan *Report, 1000), // Buffered for reasons.
	}, nil
}

// Activity defines functionality that will store and broadcast strategy related
// actions.
type Activity interface {
	// GetReporter returns a channel that gives you a full report or summary of
	// action.
	GetReporter() (<-chan *Report, error)
	ReportComplete()
	ReportTimeout(end time.Time)
	ReportFatalError(err error)
	ReportContextDone(err error)
	ReportShutdown()
	ReportInfo(message string)
	ReportOrder(*ExecutedOrder)
	ReportStart(data fmt.Stringer)
	ReportWait(next time.Time)
}

// // Reporter defines an initial concept broadcaster of a strategy action
// // NOTE: For now will only support single routine read.
// type Reporter  struct {
// 	reporter chan *Report

// Report defines an order execution action with corresponding orderbook
// deployment details.
type Report struct {
	ID       uuid.UUID   `json:"id"`
	Strategy string      `json:"strategy"`
	Action   interface{} `json:"action,omitempty"`
	Finished bool        `json:"finished,omitempty"`
	Reason   string      `json:"reason"`
	Time     time.Time   `json:"time"`
}

// Send sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver.
func (r *Activities) send(reason string, action interface{}, complete bool) {
	if r == nil || r.reporter == nil {
		return // TODO: expand
	}
	// timeout := time.NewTimer(time.Millisecond * 200) // Wait for receiver. TODO: Implement a better policy.
	select {
	case r.reporter <- &Report{
		ID:       r.id,
		Strategy: r.strategy,
		Reason:   reason,
		Action:   action,
		Finished: complete,
		Time:     time.Now(),
	}:
	// case <-timeout.C:
	default:
	}

	if complete {
		close(r.reporter)
	}
}

// ReportComplete is called when a strategy has completed sufficiently. This will
// alert a receiver that it has completed and will close the reporting channel.
func (r *Activities) ReportComplete() {
	r.send("STRATEGY COMPLETED", nil, true)
}

// ReportTimeout is called when a strategy has timed-out and it has exceeded its
// operating time. This will alert a receiver that it has completed and will
// close the reporting channel.
func (r *Activities) ReportTimeout(end time.Time) {
	r.send("STRATEGY TIMEOUT", struct{ EndTime time.Time }{EndTime: end}, true)
}

// ReportFatalError is called when a strategy has errored and it cannot continue
// operations. This will alert a receiver that it has completed and will
// close the reporting channel.
func (r *Activities) ReportFatalError(err error) {
	r.send("FATAL ERROR", struct{ Error error }{Error: err}, true)
}

// ReportContextDone is called when a context has timed-out or has been cancelled
// and cannot continue operations. This will alert a receiver that it has
// completed and will close the reporting channel.
func (r *Activities) ReportContextDone(err error) {
	r.send("CONTEXT DONE", struct{ Error error }{Error: err}, true)
}

// ReportShutdown is called when the strategy has been shutdown and cannot continue
// operations. This will alert a receiver that it has completed and will close
// the reporting channel.
func (r *Activities) ReportShutdown() {
	r.send("STRATEGY SHUTDOWN", nil, true)
}

// ReportInfo is called when the strategy wants to send relevant information to a
// reporter receiver.
func (r *Activities) ReportInfo(message string) {
	r.send("INFO", struct{ Message string }{Message: message}, false)
}

// ReportOrder is called when the strategy wants to send order exectution
// information to a reporter receiver.
func (r *Activities) ReportOrder(exec *ExecutedOrder) {
	r.send("ORDER EXECUTION", exec, false)
}

// ReportStart is called when the strategy is accepted and run.
func (r *Activities) ReportStart(data fmt.Stringer) {
	if data == nil {
		return
	}
	r.send("STRATEGY START", struct{ Message string }{Message: data.String()}, false)
}

// ReportRegister is called when the strategy is registered with the manager.
func (r *Activities) ReportRegister() {
	r.send("STRATEGY REGISTERED", nil, false)
}

// ReportWait is called to notify when the next signal is going to occur.
func (r *Activities) ReportWait(next time.Time) {
	if next.IsZero() {
		return
	}
	r.send("STRATEGY WAITING", struct{ Until string }{Until: time.Until(next).String()}, false)
}

// ReportAcceptedSignal
func (r *Activities) ReportAcceptedSignal(obj interface{}) {
	r.send("STRATEGY ACCEPTED SIGNAL", struct{ Reason interface{} }{Reason: obj}, false)
}

// ReportRejectedSignal
func (r *Activities) ReportRejectedSignal(obj interface{}) {
	r.send("STRATEGY REJECTED SIGNAL", struct{ Reason interface{} }{Reason: obj}, false)
}
