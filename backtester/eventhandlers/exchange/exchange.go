package exchange

import (
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func (e *Exchange) SetCurrency(c CurrencySettings) {
	e.CurrencySettings = c
}

func (e *Exchange) GetCurrency() CurrencySettings {
	return e.CurrencySettings
}

func (e *Exchange) ExecuteOrder(o internalordermanager.OrderEvent, data interfaces.DataHandler) (*fill.Fill, error) {
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
		ExchangeFee: e.CurrencySettings.ExchangeFee, // defaulting to just using taker fee right now without orderbook
		Why:         o.GetWhy(),
	}
	if o.GetAmount() <= 0 {
		fillEvent.Direction = common.DoNothing
		return fillEvent, nil
	}
	fillEvent.Direction = o.GetDirection()
	fillEvent.ExchangeFee = e.calculateExchangeFee(data.Latest().Price(), o.GetAmount(), e.CurrencySettings.ExchangeFee)
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
	log.Debugf(log.BackTester, "submitting fake order for %v interval", o.GetTime())
	_, err := engine.Bot.OrderManager.SubmitFakeOrder(o2, o2Response)

	if err != nil {
		return nil, err
	}
	//e.Orders.Add(&order2.Order{
	//	Event: event.Event{
	//		Exchange:     o.GetExchange(),
	//		Time:         o.GetTime(),
	//		CurrencyPair: o.Pair(),
	//		AssetType:    o.GetAssetType(),
	//		MakerFee:     e.CurrencySettings.MakerFee,
	//		TakerFee:     e.CurrencySettings.TakerFee,
	//		FeeRate:      e.CurrencySettings.ExchangeFee,
	//	},
	//	Why:       o.GetWhy(),
	//	ID:        resp.OrderID,
	//	Direction: fillEvent.Direction,
	//	Status:    o.GetStatus(),
	//	Price:     fillEvent.Price,
	//	Amount:    fillEvent.Amount,
	//	OrderType: order.Market,
	//})

	return fillEvent, nil
}

func (e *Exchange) calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
