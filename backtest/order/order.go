package order

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

func (o *Order) Cancel() {
	o.Status = order.Cancelled
}

func (o *Order) ID() uint64 {
	return o.id
}

func (o *Order) SetID(in uint64) {
	o.id = in
}

func (o *Order) Amount() float64 {
	return o.amount
}

func (o *Order) SetAmount(in float64) {
	o.amount = in
}
