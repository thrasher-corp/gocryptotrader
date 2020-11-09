package exchange

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	order2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Exchange) SetCurrency(c Currency) {
	e.Currency = c
}

func (e *Exchange) ExecuteOrder(o orders.OrderEvent, data portfolio.DataHandler) (*fill.Fill, error) {
	var curr Currency
	f := &fill.Fill{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
		},
		Direction:   o.GetDirection(),
		Amount:      o.GetAmount(),
		Price:       data.Latest().Price(),
		ExchangeFee: curr.ExchangeFee, // defaulting to just using taker fee right now without orderbook
	}
	if o.GetAmount() <= 0 {
		f.Direction = common.DoNothing
		return f, nil
	}
	f.Direction = o.GetDirection()
	f.ExchangeFee = e.calculateExchangeFee(data.Latest().Price(), o.GetAmount(), curr.ExchangeFee)
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
	e.Orders.Add(&order2.Order{
		Event: event.Event{
			Exchange:     "",
			Time:         time.Time{},
			CurrencyPair: currency.Pair{},
			AssetType:    "",
			MakerFee:     0,
			TakerFee:     0,
			FeeRate:      0,
		},
		ID:        0,
		Direction: "",
		Status:    "",
		Price:     0,
		Amount:    0,
		OrderType: "",
		Limit:     0,
		Leverage:  0,
	})
	return f, nil
}

func (e *Exchange) calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
