package exchange

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	order2 "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
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
		Why:       o.GetWhy(),
		ID:        0,
		Direction: "",
		Status:    "",
		Price:     0,
		Amount:    0,
		OrderType: "",
		Limit:     0,
		Leverage:  0,
	})

	return fillEvent, nil
}

func (e *Exchange) calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
