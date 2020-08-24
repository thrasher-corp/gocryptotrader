package backtest

import (
	"time"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Portfolio) OnFill(order *Order) (*Order, error) {
	p.Holdings.Update(order)

	if order.Direction() == gctorder.Buy {
		p.funds = -order.NetValue()
	} else {
		p.funds = +order.NetValue()
	}

	return order, nil
}

func (p *Portfolio) OnSignal(signal SignalEvent, data DataHandler) (*Order, error) {
	var limit float64

	initialOrder := &Order{
		Event: Event{
			time: signal.Time(),
		},
		orderType:  gctorder.Market,
		orderSide: gctorder.Buy,
		limitPrice: limit,
	}

	latest := data.Latest()
	sizedOrder, err := p.sizeManager.SizeOrder(initialOrder, latest, p)
	if err != nil {
	}

	order, err := p.riskManager.EvaluateOrder(sizedOrder, latest, p.Holdings)
	if err != nil {
	}

	return order, nil
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

func (p *Portfolio) Order(price float64, amount float64, side gctorder.Side) {
	var orderType gctorder.Type
	//var orderSide gctorder.Side
	var newAmount float64


	if price < 0 {
		orderType = gctorder.Market
	} else {
		orderType = gctorder.Limit
	}

	// if p.Holdings.Amount > amount {
	// 	newAmount = p.Holdings.Amount - amount
	// } else if p.Holdings.Amount < amount {
	// 	if price < 0 {
	// 		orderType = gctorder.Market
	// 	} else {
	// 		orderType = gctorder.Limit
	// 	}
	// 	newAmount = amount - p.Holdings.Amount
	// }

	initialOrder := &Order{
		Event: Event{
			time: time.Now(),
		},
		amount:     newAmount,
		orderType:  orderType,
		orderSide:  side,
		limitPrice: price,
	}
	p.OrderBook = []OrderEvent{initialOrder}
}

func (p *Portfolio) Position() Position {
	return Position{}
}

func (p *Portfolio) Update(event DataEvent) {
	if pos, ok := p.IsInvested(); ok {
		pos.UpdateValue(event)
		p.Holdings = pos
	}
}

func (p *Portfolio) Value() float64 {
	return 0
}


func (p Portfolio) SizeManager() SizeHandler {
	return p.sizeManager
}

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.sizeManager = size
}

func (p Portfolio) RiskManager() RiskHandler {
	return p.riskManager
}

func (p *Portfolio) SetRiskManager(risk RiskHandler) {
	p.riskManager = risk
}
