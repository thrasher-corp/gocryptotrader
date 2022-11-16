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
		reporter:   make(chan *Report),
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
	ReportOrder(submit *order.Submit, resp *order.SubmitResponse, detail *orderbook.Movement)
	ReportStart(data fmt.Stringer)
}

// // Reporter defines an initial concept broadcaster of a strategy action
// // NOTE: For now will only support single routine read.
// type Reporter  struct {
// 	reporter chan *Report

// Report defines an order execution action with corresponding orderbook
// deployment details.
type Report struct {
	ID       uuid.UUID
	Strategy string
	Action   interface{}
	Error    error
	Finished bool
	Reason   string
}

// Send sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver.
func (r *Activities) send(reason string, action interface{}, complete bool) {
	if r == nil || r.reporter == nil || action == nil {
		return // TODO: expand
	}
	timeout := time.NewTimer(time.Millisecond * 200) // Wait for receiver. TODO: Implement a better policy.
	select {
	case r.reporter <- &Report{
		ID:       r.id,
		Strategy: r.strategy,
		Reason:   reason,
		Action:   action,
	}:
	case <-timeout.C:
	}

	if complete {
		close(r.reporter)
	}
}

// ReportComplete is called when a strategy has completed sufficiently. This will
// alert a receiver that it has completed and will close the reporting channel.
func (r *Activities) ReportComplete() {
	r.send("STRATEGY COMPLETED", struct{ Time time.Time }{Time: time.Now()}, true)
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
	r.send("STRATEGY SHUTDOWN", struct{ Time time.Time }{Time: time.Now()}, true)
}

// ReportInfo is called when the strategy wants to send relevant information to a
// reporter receiver.
func (r *Activities) ReportInfo(message string) {
	r.send("INFO", struct{ Message string }{Message: message}, false)
}

type ExecutedOrder struct {
	Submit    *order.Submit
	Response  *order.SubmitResponse
	Orderbook *orderbook.Movement
}

// ReportOrder is called when the strategy wants to send order exectution
// information to a reporter receiver.
func (r *Activities) ReportOrder(submit *order.Submit, resp *order.SubmitResponse, detail *orderbook.Movement) {
	r.send("ORDER EXECUTION",
		&ExecutedOrder{
			Submit:    submit,
			Response:  resp,
			Orderbook: detail,
		},
		false)
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
	r.send("STRATEGY REGISTERED", struct{ ID uuid.UUID }{ID: r.id}, false)
}
