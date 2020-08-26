package backtest

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.sizeManager = size
}

func (p *Portfolio) Reset() {
	p.cash = 0
	p.holdings = nil
	p.transactions = nil
}

func (p *Portfolio) OnSignal(signal SignalEvent, data DataHandler) (*Order, error) {
	var limit float64

	if signal.GetDirection() == "" {
		return &Order{}, errors.New("invalid Direction")
	}

	currAmount := p.holdings[signal.Pair().String()].amount
	currCash := p.Funds()
	currPrice := data.Latest(signal.Pair()).LatestPrice()

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && currAmount <= signal.GetAmount() {
		return &Order{}, errors.New("no holdings to sell")
	}

	if (signal.GetDirection() == gctorder.Buy || signal.GetDirection() == gctorder.Bid) && currCash <= signal.GetPrice()*currPrice {
		return &Order{}, errors.New("tot enough cash to buy")
	}

	initialOrder := &Order{
		Event: Event{
			Time:         signal.GetTime(),
			CurrencyPair: signal.Pair(),
		},
		Direction: signal.GetDirection(),
		Amount:    signal.GetAmount(),
		OrderType: gctorder.Market,
		Limit:     limit,
	}

	latest := data.Latest(signal.Pair())
	sizedOrder, err := p.sizeManager.SizeOrder(initialOrder, latest, p)
	if err != nil {
		return nil, err
	}

	order, err := p.riskManager.EvaluateOrder(sizedOrder, latest, p.holdings)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (p *Portfolio) OnFill(fill FillEvent, data DataHandler) (*Fill, error) {
	if p.holdings == nil {
		p.holdings = make(map[string]Positions)
	}

	if pos, ok := p.holdings[fill.Pair().String()]; ok {
		pos.Update(fill)
		p.holdings[fill.Pair().String()] = pos
	} else {
		pos := Positions{}
		pos.Create(fill)
		p.holdings[fill.Pair().String()] = pos
	}

	if fill.GetDirection() == gctorder.Buy {
		p.cash -= fill.NetValue()
	} else if fill.GetDirection() == gctorder.Sell || fill.GetDirection() == gctorder.Ask {
		p.cash += fill.NetValue()
	}

	p.transactions = append(p.transactions, fill)

	f := fill.(*Fill)
	return f, nil
}

func (p *Portfolio) IsInvested(symbol string) (pos Positions, ok bool) {
	pos, ok = p.holdings[symbol]
	if ok && (pos.amount != 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsLong(symbol string) (pos Positions, ok bool) {
	pos, ok = p.holdings[symbol]
	if ok && (pos.amount > 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsShort(symbol string) (pos Positions, ok bool) {
	pos, ok = p.holdings[symbol]
	if ok && (pos.amount < 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) Update(d DataEventHandler) {
	if pos, ok := p.IsInvested(d.Pair().String()); ok {
		pos.UpdateValue(d)
		p.holdings[d.Pair().String()] = pos
	}
}

func (p *Portfolio) SetInitialFunds(initial float64) {
	p.initialCash = initial
}

func (p *Portfolio) InitialFunds() float64 {
	return p.initialCash
}

func (p *Portfolio) SetFunds(cash float64) {
	p.cash = cash
}

func (p *Portfolio) Funds() float64 {
	return p.cash
}

func (p *Portfolio) Value() float64 {
	holdingValue := decimal.NewFromFloat(0)
	for x := range p.holdings {
		marketValue := decimal.NewFromFloat(p.holdings[x].marketValue)
		holdingValue = holdingValue.Add(marketValue)
	}

	cash := decimal.NewFromFloat(p.cash)
	value, _ := cash.Add(holdingValue).Round(4).Float64()
	return value
}

func (p *Portfolio) ViewHoldings() {
	fmt.Println(p.holdings)
}
