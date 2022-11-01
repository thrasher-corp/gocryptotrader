package common

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var errReportIsNil = errors.New("activity report is nil")

// Reporter defines an initial concept broadcaster of a strategy action
type Reporter chan *Report

// Send sends a strategy activity report to a potential receiver. Will
// do nothing if there is no receiver.
func (r *Reporter) Send(rp *Report) {
	if r == nil || rp == nil {
		return // TODO: expand
	}
	*r <- rp
	if rp.Finished {
		close(*r)
	}
}

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
