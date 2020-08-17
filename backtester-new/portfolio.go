package backtest

import gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"

func (p *Portfolio) OnFill(order *Order) (*Order, error) {
	//p.holdings.Update(order)

	if order.Direction() == gctorder.Buy {
		p.funds = -order.NetValue()
	} else {
		p.funds = +order.NetValue()
	}

	return order, nil
}

func (p *Portfolio) OnSignal(signal SignalEvent, data DataHandler) (*Order, error) {
	return nil, nil
}

func (p Portfolio) IsInvested() (pos Position, ok bool) {
	pos = p.Holdings
	if pos.Amount != 0 {
		return pos, true
	}
	return pos, false
}

func (p Portfolio) IsLong() (pos Position, ok bool) {
	pos = p.Holdings
	if pos.Amount > 0 {
		return pos, true
	}
	return pos, false
}

func (p Portfolio) IsShort() (pos Position, ok bool) {
	pos = p.Holdings
	if pos.Amount < 0 {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) Reset() error {
	return nil
}

func (p *Portfolio) SetInitialFunds(funds float64) {
	p.initialFunds = funds
}

func (p Portfolio) InitialFunds() float64 {
	return p.initialFunds
}

func (p *Portfolio) SetFunds(funds float64) {
	p.funds = funds
}

func (p Portfolio) Funds() float64 {
	return p.funds
}

func (p *Portfolio) Order(price float64, amount float64) {
	data := Backtest{}.data
	var orderType gctorder.Type
	var orderSide gctorder.Side
	var newAmount float64

	if p.Holdings.Amount > amount {
		if price < 0 {
			orderType = gctorder.Market
		} else {
			orderType = gctorder.Limit
		}
		orderSide = gctorder.Sell
		newAmount = p.Holdings.Amount - amount
	} else if p.Holdings.Amount < amount {
		if price < 0 {
			orderType = gctorder.Market
		} else {
			orderType = gctorder.Limit
		}
		orderSide = gctorder.Buy
		newAmount = amount - p.Holdings.Amount
	}

	initialOrder := &Order{
		Event: Event{
			time: data.Latest().Time(),
		},
		amount:     newAmount,
		orderType:  orderType,
		orderSide:  orderSide,
		limitPrice: price,
	}
	p.OrderBook = []OrderEvent{initialOrder}
}

func (p *Portfolio) Position() Position {
	return Position{}
}

func (p *Portfolio) Update(event DataEvent) {

}

func (p *Portfolio) Value() float64 {
	return 0
}
