package exchange

import (
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Exchange) ExecuteOrder(o orders.OrderEvent, data portfolio.DataHandler) (*fill.Fill, error) {
	f := &fill.Fill{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
		},
		Direction:   "",
		Amount:      o.GetAmount(),
		Price:       data.Latest().LatestPrice(),
		ExchangeFee: 0,
	}

	f.Direction = o.GetDirection()
	f.ExchangeFee = e.calculateExchangeFee()
	u, _ := uuid.NewV4()
	o2 := &order.Submit{
		Price:       f.Price,
		Amount:      o.GetAmount(),
		Fee:         f.ExchangeFee,
		Exchange:    f.Exchange,
		ID:          u.String(),
		Side:        f.Direction,
		AssetType:   f.AssetType,
		Date:        o.GetTime(),
		LastUpdated: o.GetTime(),
		Pair:        o.Pair(),
		Type:        order.Market,
	}
	o2Response := order.SubmitResponse{
		IsOrderPlaced: true,
		OrderID:       u.String(),
		Rate:          f.Amount,
		Fee:           f.ExchangeFee,
		Cost:          f.Price,
		Trades:        nil,
	}
	_, err := engine.Bot.OrderManager.SubmitFakeOrder(o2, o2Response)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (e *Exchange) calculateExchangeFee() float64 {
	return e.ExchangeFee
}
