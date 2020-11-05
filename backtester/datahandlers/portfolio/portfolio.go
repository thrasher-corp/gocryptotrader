package portfolio

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	fill2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/backtester/risk"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func New(initialFunds, defaultAmount, maximumAmount, feeRate float64) (*Portfolio, error) {
	if defaultAmount == 0 && maximumAmount == 0 {
		return nil, errors.New("requires funding guidance")
	}
	if initialFunds == 0 {
		return nil, errors.New("can't hope to buy anything without money")
	}
	return &Portfolio{
		InitialFunds: initialFunds,
		RiskManager:  &risk.Risk{},
		SizeManager: &size.Size{
			DefaultSize: defaultAmount,
			MaxSize:     maximumAmount,
			FeeRate:     feeRate,
		},
	}, nil
}

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.SizeManager = size
}

func (p *Portfolio) Reset() {
	p.Funds = 0
	p.Holdings = nil
	p.Transactions = nil
}

func (p *Portfolio) OnSignal(signal signal.SignalEvent, data portfolio.DataHandler) (*order.Order, error) {
	if signal.GetDirection() == "" {
		return &order.Order{}, errors.New("invalid Direction")
	}

	currAmount := p.Holdings[signal.Pair()].Amount
	currFunds := p.GetFunds()

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && currAmount <= signal.GetAmount() {
		return &order.Order{}, errors.New("no holdings to sell")
	}

	if (signal.GetDirection() == gctorder.Buy || signal.GetDirection() == gctorder.Bid) && currFunds <= 0 {
		return &order.Order{}, errors.New("not enough funds to buy")
	}

	initialOrder := &order.Order{
		Event: event.Event{
			Exchange:     signal.GetExchange(),
			Time:         signal.GetTime(),
			CurrencyPair: signal.Pair(),
			AssetType:    signal.GetAssetType(),
		},
		Direction: signal.GetDirection(),
		Price:     signal.GetPrice(),
		Amount:    signal.GetAmount(),
		OrderType: gctorder.Market,
	}
	latest := data.Latest()
	sizedOrder, err := p.SizeManager.SizeOrder(initialOrder, latest, currFunds, p.)
	if err != nil {
		return nil, err
	}

	o, err := p.RiskManager.EvaluateOrder(sizedOrder, latest, p.Holdings)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (p *Portfolio) OnFill(fill fill2.FillEvent, _ portfolio.DataHandler) (*fill2.Fill, error) {
	if p.Holdings == nil {
		p.Holdings = make(map[currency.Pair]positions.Positions)
	}

	if pos, ok := p.Holdings[fill.Pair()]; ok {
		pos.Update(fill)
		p.Holdings[fill.Pair()] = pos
	} else {
		pos := positions.Positions{}
		pos.Create(fill)
		p.Holdings[fill.Pair()] = pos
	}

	if fill.GetDirection() == gctorder.Buy {
		p.Funds -= fill.NetValue()
	} else if fill.GetDirection() == gctorder.Sell || fill.GetDirection() == gctorder.Ask {
		p.Funds += fill.NetValue()
	}

	p.Transactions = append(p.Transactions, fill)

	return fill.(*fill2.Fill), nil
}

func (p *Portfolio) IsInvested(pair currency.Pair) (pos positions.Positions, ok bool) {
	pos, ok = p.Holdings[pair]
	if ok && (pos.Amount != 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsLong(pair currency.Pair) (pos positions.Positions, ok bool) {
	pos, ok = p.Holdings[pair]
	if ok && (pos.Amount > 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsShort(pair currency.Pair) (pos positions.Positions, ok bool) {
	pos, ok = p.Holdings[pair]
	if ok && (pos.Amount < 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) Update(d portfolio.DataEventHandler) {
	if pos, ok := p.IsInvested(d.Pair()); ok {
		pos.UpdateValue(d)
		p.Holdings[d.Pair()] = pos
	}
}

func (p *Portfolio) SetInitialFunds(initial float64) {
	p.InitialFunds = initial
}

func (p *Portfolio) GetInitialFunds() float64 {
	return p.InitialFunds
}

func (p *Portfolio) SetFunds(funds float64) {
	p.Funds = funds
}

func (p *Portfolio) GetFunds() float64 {
	return p.Funds
}

func (p *Portfolio) Value() float64 {
	holdingValue := decimal.NewFromFloat(0)
	for x := range p.Holdings {
		marketValue := decimal.NewFromFloat(p.Holdings[x].MarketValue)
		holdingValue = holdingValue.Add(marketValue)
	}

	funds := decimal.NewFromFloat(p.Funds)
	value, _ := funds.Add(holdingValue).Round(4).Float64()
	return value
}

func (p *Portfolio) ViewHoldings() map[currency.Pair]positions.Positions {
	return p.Holdings
}
