package exchange

import (
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

func (e *Exchange) Reset() {
	*e = Exchange{}
}

func (e *Exchange) ExecuteOrder(o order.Event, data data.Handler, bot *engine.Engine) (*fill.Fill, error) {
	cs, _ := e.GetCurrencySettings(o.GetExchange(), o.GetAssetType(), o.Pair())
	f := &fill.Fill{
		Event: event.Event{
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
			Interval:     o.GetInterval(),
			Why:          o.GetWhy(),
		},
		Direction: o.GetDirection(),
		Amount:    o.GetAmount(),

		ClosePrice:  data.Latest().Price(),
		ExchangeFee: cs.ExchangeFee, // defaulting to just using taker fee right now without orderbook
	}

	f.Direction = o.GetDirection()
	if o.GetDirection() != gctorder.Buy && o.GetDirection() != gctorder.Sell {
		return f, nil
	}
	highStr := data.StreamHigh()
	high := highStr[len(highStr)-1]

	lowStr := data.StreamLow()
	low := lowStr[len(lowStr)-1]

	volStr := data.StreamVol()
	volume := volStr[len(volStr)-1]
	var adjustedPrice, amount float64
	var err error
	if cs.UseRealOrders {
		// get current orderbook
		var ob *orderbook.Base
		ob, err = orderbook.Get(f.Exchange, f.CurrencyPair, f.AssetType)
		if err != nil {
			return f, err
		}
		// calculate an estimated slippage rate
		slippageRate := slippage.CalculateSlippageByOrderbook(ob, o.GetDirection(), f.ClosePrice)
		adjustedPrice = f.ClosePrice * slippageRate
		amount = f.Amount
	} else {
		adjustedPrice, amount, err = e.sizeOfflineOrder(high, low, volume, &cs, f)
		if err != nil {
			switch f.GetDirection() {
			case gctorder.Buy:
				f.SetDirection(common.CouldNotBuy)
			case gctorder.Sell:
				f.SetDirection(common.CouldNotSell)
			default:
				f.SetDirection(common.DoNothing)
			}
			f.AppendWhy(err.Error())
			return f, err
		}
	}

	var orderID string
	orderID, err = e.placeOrder(adjustedPrice, amount, cs.UseRealOrders, f, bot)
	if err != nil {
		return f, err
	}
	ords, _ := bot.OrderManager.GetOrdersSnapshot("")
	for i := range ords {
		if ords[i].ID != orderID {
			continue
		}
		ords[i].Date = o.GetTime()
		ords[i].LastUpdated = o.GetTime()
		ords[i].CloseTime = o.GetTime()
		f.Order = &ords[i]
		f.PurchasePrice = ords[i].Price
	}

	if f.Order == nil {
		return nil, fmt.Errorf("placed order %v not found in order manager", orderID)
	}

	return f, nil
}

func (e *Exchange) placeOrder(price, amount float64, useRealOrders bool, f *fill.Fill, bot *engine.Engine) (string, error) {
	if f == nil {
		return "", errors.New("received nil event")
	}
	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	var orderID string
	o := &gctorder.Submit{
		Price:       price,
		Amount:      amount,
		Fee:         f.ExchangeFee,
		Exchange:    f.Exchange,
		ID:          u.String(),
		Side:        f.Direction,
		AssetType:   f.AssetType,
		Date:        f.GetTime(),
		LastUpdated: f.GetTime(),
		Pair:        f.Pair(),
		Type:        gctorder.Market,
	}

	if useRealOrders {
		resp, err := bot.OrderManager.Submit(o)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	} else {
		submitResponse := gctorder.SubmitResponse{
			IsOrderPlaced: true,
			OrderID:       u.String(),
			Rate:          f.Amount,
			Fee:           f.ExchangeFee,
			Cost:          price,
			FullyMatched:  true,
		}
		resp, err := bot.OrderManager.SubmitFakeOrder(o, submitResponse)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	}
	return orderID, nil
}

func (e *Exchange) sizeOfflineOrder(high, low, volume float64, cs *Settings, f *fill.Fill) (adjustedPrice, adjustedAmount float64, err error) {
	if cs == nil || f == nil {
		return 0, 0, errors.New("received nil arguments")
	}
	// provide history and estimate volatility
	slippageRate := slippage.EstimateSlippagePercentage(cs.MinimumSlippageRate, cs.MaximumSlippageRate)
	f.VolumeAdjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(f.ClosePrice, f.Amount, high, low, volume)
	if adjustedAmount <= 0 && f.Amount > 0 {
		return 0, 0, fmt.Errorf("amount set to 0, data may be incorrect")
	}
	adjustedPrice = applySlippageToPrice(f.GetDirection(), f.GetVolumeAdjustedPrice(), slippageRate)

	f.Slippage = (slippageRate * 100) - 100
	f.ExchangeFee = calculateExchangeFee(adjustedPrice, adjustedAmount, cs.ExchangeFee)
	return adjustedPrice, adjustedAmount, nil
}

func applySlippageToPrice(direction gctorder.Side, price, slippageRate float64) float64 {
	adjustedPrice := price
	if direction == gctorder.Buy {
		adjustedPrice = price + (price * (1 - slippageRate))
	} else if direction == gctorder.Sell {
		adjustedPrice = price * slippageRate
	}
	return adjustedPrice
}

func (e *Exchange) SetCurrency(exch string, a asset.Item, cp currency.Pair, c *Settings) {
	if c.ExchangeName == "" ||
		c.AssetType == "" ||
		c.CurrencyPair.IsEmpty() {
		return
	}

	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair == cp &&
			e.CurrencySettings[i].AssetType == a &&
			exch == e.CurrencySettings[i].ExchangeName {
			e.CurrencySettings[i] = *c
			return
		}
	}
	e.CurrencySettings = append(e.CurrencySettings, *c)
}

func (e *Exchange) GetCurrencySettings(exch string, a asset.Item, cp currency.Pair) (Settings, error) {
	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair == cp {
			if e.CurrencySettings[i].AssetType == a {
				if exch == e.CurrencySettings[i].ExchangeName {
					return e.CurrencySettings[i], nil
				}
			}
		}
	}
	return Settings{}, fmt.Errorf("no currency settings found for %v %v %v", exch, a, cp)
}

func ensureOrderFitsWithinHLV(slippagePrice, amount, high, low, volume float64) (adjustedPrice, adjustedAmount float64) {
	adjustedPrice = slippagePrice
	if adjustedPrice < low {
		adjustedPrice = low
	}
	if adjustedPrice > high {
		adjustedPrice = high
	}
	if volume <= 0 {
		return adjustedPrice, adjustedAmount
	}
	currentVolume := amount * adjustedPrice
	if currentVolume > volume {
		// hey, this order is too big here
		for currentVolume > volume {
			// reduce the volume by a fraction until it is within the candle's volume
			currentVolume *= 0.99999999
		}
	}
	// extract the amount from the adjusted volume
	adjustedAmount = currentVolume / adjustedPrice

	return adjustedPrice, adjustedAmount
}

func calculateExchangeFee(price, amount, fee float64) float64 {
	return fee * price * amount
}
