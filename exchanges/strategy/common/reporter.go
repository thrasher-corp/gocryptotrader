package common

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Activity defines functionality that will store and broadcast strategy related
// actions.
type Activity interface {
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

// Reporter defines an initial concept broadcaster of a strategy action
// NOTE: For now will only support single routine read.
type Reporter chan *Report

// Report defines an order execution action with corresponding orderbook
// deployment details.
type Report struct {
	Submit     *order.Submit
	Response   *order.SubmitResponse
	Deployment *orderbook.Movement
	Error      error
	Finished   bool
	Reason     string
}

// Send sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver.
func (r *Reporter) send(rp *Report) {
	if r == nil || *r == nil || rp == nil {
		return // TODO: expand
	}
	timeout := time.NewTimer(time.Millisecond * 200) // Wait for receiver. TODO: Implement a better policy.
	select {
	case *r <- rp:
	case <-timeout.C:
	}

	if rp.Finished {
		close(*r)
	}
}

// ReportComplete is called when a strategy has completed sufficiently. This will
// alert a receiver that it has completed and will close the reporting channel.
func (r *Reporter) ReportComplete() {
	r.send(&Report{Reason: "STRATEGY COMPLETED", Finished: true})
}

// ReportTimeout is called when a strategy has timed-out and it has exceeded its
// operating time. This will alert a receiver that it has completed and will
// close the reporting channel.
func (r *Reporter) ReportTimeout(end time.Time) {
	r.send(&Report{Reason: fmt.Sprintf("TIMELAPSE: %s", end), Finished: true})
}

// ReportFatalError is called when a strategy has errored and it cannot continue
// operations. This will alert a receiver that it has completed and will
// close the reporting channel.
func (r *Reporter) ReportFatalError(err error) {
	r.send(&Report{Reason: "FATAL ERROR", Error: err, Finished: true})
}

// ReportContextDone is called when a context has timed-out or has been cancelled
// and cannot continue operations. This will alert a receiver that it has
// completed and will close the reporting channel.
func (r *Reporter) ReportContextDone(err error) {
	r.send(&Report{Reason: "CONTEXT DONE", Error: err, Finished: true})
}

// ReportShutdown is called when the strategy has been shutdown and cannot continue
// operations. This will alert a receiver that it has completed and will close
// the reporting channel.
func (r *Reporter) ReportShutdown() {
	r.send(&Report{Reason: "STRATEGY SHUTDOWN", Finished: true})
}

// ReportInfo is called when the strategy wants to send relevant information to a
// reporter receiver.
func (r *Reporter) ReportInfo(message string) {
	r.send(&Report{Reason: message})
}

// ReportOrder is called when the strategy wants to send order exectution
// information to a reporter receiver.
func (r *Reporter) ReportOrder(submit *order.Submit, resp *order.SubmitResponse, detail *orderbook.Movement) {
	r.send(&Report{
		Reason:     "ORDER EXECUTION",
		Submit:     submit,
		Response:   resp,
		Deployment: detail})
}

// ReportStart is called when the strategy is accepted and run.
func (r *Reporter) ReportStart(data fmt.Stringer) {
	if data == nil {
		return
	}
	r.send(&Report{Reason: data.String()})
}

// ReportRegister is called when the strategy is registered with the manager.
func (r *Reporter) ReportRegister(id uuid.UUID) {
	r.send(&Report{Reason: "REGISTERED: " + id.String()})
}
