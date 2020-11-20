package exchange

import (
	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
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

func (e *Exchange) ensureOrderFitsWithinHLV(slippagePrice, amount, high, low, volume float64) (float64, float64) {
	if slippagePrice < low {
		slippagePrice = low
	}
	if slippagePrice > high {
		slippagePrice = high
	}

	if amount*slippagePrice > volume {
		// hey, this order is too big here
		for amount*slippagePrice > volume {
			amount *= 0.99999
		}
	}

	return slippagePrice, amount
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
	var slippageRate, estimatedPrice, amount float64
	if e.UseRealOrders {
		// get current orderbook
		// calculate an estimated slippage rate
		slippageRate = slippage.CalculateSlippage(nil)
		estimatedPrice = fillEvent.Price * slippageRate
	} else {
		// provide n history and estimate volatility
		slippageRate = slippage.EstimateSlippagePercentage()
		estimatedPrice = fillEvent.Price * slippageRate
		high := data.StreamHigh()
		low := data.StreamLow()
		volume := data.StreamVol()

		estimatedPrice, amount = e.ensureOrderFitsWithinHLV(estimatedPrice, o.GetAmount(), high[len(high)-1], low[len(low)-1], volume[len(volume)-1])
	}

	fillEvent.ExchangeFee = e.calculateExchangeFee(estimatedPrice, amount, e.CurrencySettings.ExchangeFee)
	u, _ := uuid.NewV4()
	o2 := &order.Submit{
		Price:       estimatedPrice,
		Amount:      amount,
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
	if e.UseRealOrders {
		_, err := engine.Bot.OrderManager.Submit(o2)
		if err != nil {
			return nil, err
		}
	} else {
		o2Response := order.SubmitResponse{
			IsOrderPlaced: true,
			OrderID:       u.String(),
			Rate:          fillEvent.Amount,
			Fee:           fillEvent.ExchangeFee,
			Cost:          estimatedPrice,
			FullyMatched:  true,
		}
		log.Debugf(log.BackTester, "submitting fake order for %v interval", o.GetTime())
		_, err := engine.Bot.OrderManager.SubmitFakeOrder(o2, o2Response)
		if err != nil {
			return nil, err
		}
	}

	//e.InternalOrderManager.Add(&order2.Order{
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
