package portfolio

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	fill2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.SizeManager = size
}

func (p *Portfolio) Reset() {
	p.Funds = 0
	p.Holdings = nil
	p.Transactions = nil
}

func (p *Portfolio) SetFee(exchangeName string, a asset.Item, cp currency.Pair, fee float64) {
	if p.Fees == nil {
		p.Fees = make(map[string]map[asset.Item]map[currency.Pair]float64)
	}
	if p.Fees[exchangeName] == nil {
		p.Fees[exchangeName] = make(map[asset.Item]map[currency.Pair]float64)
	}
	if p.Fees[exchangeName][a] == nil {
		p.Fees[exchangeName][a] = make(map[currency.Pair]float64)
	}
	p.Fees[exchangeName][a][cp] = fee
}

// GetFee can panic for bad requests, but why are you getting things that don't exist?
func (p *Portfolio) GetFee(exchangeName string, a asset.Item, cp currency.Pair) float64 {
	return p.Fees[exchangeName][a][cp]
}

func (p *Portfolio) OnSignal(signal signal.SignalEvent, data portfolio.DataHandler) (*order.Order, error) {
	if signal.GetDirection() == "" {
		return &order.Order{}, errors.New("invalid Direction")
	}

	holdings := p.ViewHoldings(signal.GetExchange(), signal.GetAssetType(), signal.Pair())
	currFunds := p.GetFunds()

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && holdings.Amount <= signal.GetAmount() {
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
	sizedOrder, err := p.SizeManager.SizeOrder(initialOrder, latest, currFunds, p.GetFee(signal.GetExchange(), signal.Pair(), signal.GetAssetType()))
	if err != nil {
		return nil, err
	}

	o, err := p.RiskManager.EvaluateOrder(sizedOrder, latest, holdings)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (p *Portfolio) OnFill(fill fill2.FillEvent, _ portfolio.DataHandler) (*fill2.Fill, error) {
	holdings := p.ViewHoldings(fill.GetExchange(), fill.GetAssetType(), fill.Pair())
	if !holdings.Timestamp.IsZero() {
		holdings.Update(fill)
	} else {
		holdings := positions.Positions{}
		holdings.Create(fill)
	}
	p.SetHoldings(fill.GetExchange(), fill.GetAssetType(), fill.Pair(), holdings)

	if fill.GetDirection() == gctorder.Buy {
		p.Funds -= fill.NetValue()
	} else if fill.GetDirection() == gctorder.Sell || fill.GetDirection() == gctorder.Ask {
		p.Funds += fill.NetValue()
	}

	p.Transactions = append(p.Transactions, fill)

	return fill.(*fill2.Fill), nil
}

func (p *Portfolio) IsInvested(exchangeName string, a asset.Item, cp currency.Pair) (pos positions.Positions, ok bool) {
	pos = p.ViewHoldings(exchangeName, a, cp)
	if ok && (pos.Amount != 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsLong(exchangeName string, a asset.Item, cp currency.Pair) (pos positions.Positions, ok bool) {
	pos = p.ViewHoldings(exchangeName, a, cp)
	if ok && (pos.Amount > 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) IsShort(exchangeName string, a asset.Item, cp currency.Pair) (pos positions.Positions, ok bool) {
	pos = p.ViewHoldings(exchangeName, a, cp)
	if ok && (pos.Amount < 0) {
		return pos, true
	}
	return pos, false
}

func (p *Portfolio) Update(d portfolio.DataEventHandler) {
	if pos, ok := p.IsInvested(d.GetExchange(), d.GetAssetType(), d.Pair()); ok {
		pos.UpdateValue(d)
		p.SetHoldings(d.GetExchange(), d.GetAssetType(), d.Pair(), pos)
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
	for i := range p.Holdings {
		for j := range p.Holdings[i] {
			for k := range p.Holdings[i][j] {
				marketValue := decimal.NewFromFloat(p.Holdings[i][j][k].MarketValue)
				holdingValue = holdingValue.Add(marketValue)

			}
		}
	}

	funds := decimal.NewFromFloat(p.Funds)
	value, _ := funds.Add(holdingValue).Round(4).Float64()
	return value
}

func (p *Portfolio) ViewHoldings(exchangeName string, a asset.Item, cp currency.Pair) positions.Positions {
	return p.Holdings[exchangeName][a][cp]
}

func (p *Portfolio) SetHoldings(exchangeName string, a asset.Item, cp currency.Pair, pos positions.Positions) {
	if p.Holdings == nil {
		p.Holdings = make(map[string]map[asset.Item]map[currency.Pair]positions.Positions)
	}
	if p.Holdings[exchangeName] == nil {
		p.Holdings[exchangeName] = make(map[asset.Item]map[currency.Pair]positions.Positions)
	}
	if p.Holdings[exchangeName][a] == nil {
		p.Holdings[exchangeName][a] = make(map[currency.Pair]positions.Positions)
	}

	p.Holdings[exchangeName][a][cp] = pos
}
