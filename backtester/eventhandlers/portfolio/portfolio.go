package portfolio

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Portfolio) Reset() {
	p.Funds = 0
	p.Holdings = nil
	p.Transactions = nil
}

func (p *Portfolio) OnSignal(signal signal.SignalEvent, data interfaces.DataHandler, cs *exchange.CurrencySettings) (*order.Order, error) {
	if signal.GetDirection() == "" {
		return &order.Order{}, errors.New("invalid Direction")
	}

	exchangeAssetPairHoldings := p.ViewHoldings(signal.GetExchange(), signal.GetAssetType(), signal.Pair())
	currFunds := p.GetFunds()

	if signal.GetDirection() == common.DoNothing {
		return &order.Order{
			Event: event.Event{
				Exchange:     signal.GetExchange(),
				Time:         signal.GetTime(),
				CurrencyPair: signal.Pair(),
				AssetType:    signal.GetAssetType(),
			},
			Direction: signal.GetDirection(),
			Why:       signal.GetWhy(),
		}, nil
	}

	if (signal.GetDirection() == gctorder.Sell || signal.GetDirection() == gctorder.Ask) && exchangeAssetPairHoldings.Amount <= signal.GetAmount() {
		return nil, NoHoldingsToSellErr
	}

	if (signal.GetDirection() == gctorder.Buy || signal.GetDirection() == gctorder.Bid) && currFunds <= 0 {
		return nil, NotEnoughFundsErr
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
		Why:       signal.GetWhy(),
	}
	latest := data.Latest()
	sizedOrder, err := p.SizeManager.SizeOrder(
		initialOrder,
		latest,
		currFunds,
		cs,
	)
	if err != nil {
		return nil, err
	}

	o, err := p.RiskManager.EvaluateOrder(sizedOrder, latest, exchangeAssetPairHoldings, p.Holdings)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (p *Portfolio) OnFill(fillEvent fill.FillEvent, _ interfaces.DataHandler) (*fill.Fill, error) {
	if fillEvent.GetDirection() == common.DoNothing {
		what := fillEvent.(*fill.Fill)
		what.ExchangeFee = 0
		return what, nil
	}
	holdings := p.ViewHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair())
	if !holdings.Timestamp.IsZero() {
		holdings.Update(fillEvent)
	} else {
		holdings = positions.Positions{}
		holdings.Create(fillEvent)
	}
	p.SetHoldings(fillEvent.GetExchange(), fillEvent.GetAssetType(), fillEvent.Pair(), holdings)

	if fillEvent.GetDirection() == gctorder.Buy {
		p.Funds -= fillEvent.NetValue()
	} else if fillEvent.GetDirection() == gctorder.Sell || fillEvent.GetDirection() == gctorder.Ask {
		p.Funds += fillEvent.NetValue()
	}

	p.Transactions = append(p.Transactions, fillEvent)

	return fillEvent.(*fill.Fill), nil
}

func (p *Portfolio) SetSizeManager(size SizeHandler) {
	p.SizeManager = size
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

func (p *Portfolio) Update(d interfaces.DataEventHandler) {
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
