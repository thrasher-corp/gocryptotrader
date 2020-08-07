package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (o Order) ID() int {
	return o.id
}

func (o *Order) SetID(id int) {
	o.id = id
}

func (o Order) Direction() order.Side {
	if o.orderType == order.Limit || o.orderType == order.Market {
		return order.Buy
	} else {
		return order.Sell
	}
}

func (o *Order) SetOrderType(orderType order.Type) {
	o.orderType = orderType
}
func (o *Order) GetOrderType() (orderType order.Type) {
	return o.orderType
}

func (o Order) Amount() float64 {
	return o.amount
}

func (o *Order) SetAmount(i float64) {
	o.amount = i
}

func (o Order) Status() order.Status {
	return o.status
}

func (o *Order) Cancel() {
	o.status = order.Cancelled
}

func (o *Order) Update(_ OrderEvent) {

}

func (o Order) GetAmountFilled() float64 {
	return o.amountFilled
}

func (o Order) GetAvgFillPrice() float64 {
	return o.avgFillPrice
}

func (o Order) Price() float64 {
	return o.avgFillPrice
}

func (o Order) ExchangeFee() float64 {
	return o.fee
}

func (o Order) Cost() float64 {
	return o.cost
}

func (o Order) Value() float64 {
	return o.amountFilled * o.avgFillPrice
}

func (o *Order) NetValue() float64 {
	if o.Direction() == order.Buy {
		return o.amountFilled*o.avgFillPrice + o.cost
	}

	return  o.amountFilled*o.avgFillPrice - o.cost
}
