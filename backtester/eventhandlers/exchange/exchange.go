package exchange

import (
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	order2 "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Exchange) SetCurrency(c Currency) {
	e.Currency = c
}

func (e *Exchange) ExecuteOrder(o internalordermanager.OrderEvent, data portfolio.DataHandler) (*fill.Fill, error) {
	fillEvent := &fill.Fill{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
		},
		Direction:   o.GetDirection(),
		Amount:      o.GetAmount(),
		Price:       data.Latest().Price(),
		ExchangeFee: e.Currency.ExchangeFee, // defaulting to just using taker fee right now without orderbook
		Why:         o.GetWhy(),
	}
	if o.GetAmount() <= 0 {
		fillEvent.Direction = common.DoNothing
		return fillEvent, nil
	}
	fillEvent.Direction = o.GetDirection()
	fillEvent.ExchangeFee = e.calculateExchangeFee(data.Latest().Price(), o.GetAmount(), e.Currency.ExchangeFee)
	u, _ := uuid.NewV4()
	o2 := &order.Submit{
		Price:       fillEvent.Price,
		Amount:      o.GetAmount(),
		Fee:         fillEvent.ExchangeFee,
		Exchange:    fillEvent.Exchange,
		ID:          u.String(),
		Side:        fillEvent.Direction,
		AssetType:   fillEvent.AssetType,
		Date:        o.GetTime(),
		LastUpdated: o.GetTime(),
		Pair:        o.Pair(),
		Type:        order.Market,
	}
	o2Response := order.SubmitResponse{
		IsOrderPlaced: true,
		OrderID:       u.String(),
		Rate:          fillEvent.Amount,
		Fee:           fillEvent.ExchangeFee,
		Cost:          fillEvent.Price,
		FullyMatched:  true,
	}

	resp, err := engine.Bot.OrderManager.SubmitFakeOrder(o2, o2Response)

	if err != nil {
		return nil, err
	}
	e.Orders.Add(&order2.Order{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
			MakerFee:     e.Currency.MakerFee,
			TakerFee:     e.Currency.TakerFee,
			FeeRate:      e.Currency.ExchangeFee,
		},
		Why:       o.GetWhy(),
		ID:        resp.OrderID,
		Direction: fillEvent.Direction,
		Status:    o.GetStatus(),
		Price:     fillEvent.Price,
		Amount:    fillEvent.Amount,
		OrderType: order.Market,
	})

	return fillEvent, nil
}

func (e *Exchange) calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
