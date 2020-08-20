package backtest

import (
	"fmt"
	"time"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Execution) OnData(data DataEvent, t *Backtest) (OrderEvent, error) {
	fmt.Println("Execution OnData()")
	portfolio := t.Portfolio.(*Portfolio)
	candle := data.(*Candle)

	orders := &portfolio.OrderBook
	for i := len(*orders) - 1; i >= 0; i-- {
		v := (*orders)[i]

		order, _ := v.(*Order)
		price := 0.0

		switch order.orderType {
		case gctorder.Market:
			if order.Direction() == gctorder.Buy {
				price = candle.Price()
				break
			} else if order.Direction() == gctorder.Sell {
				if order.limitPrice < candle.Price() {
					continue
				}
				price = order.limitPrice
				break
			}
		case gctorder.Limit:
			if order.Direction() == gctorder.Buy {
				if order.limitPrice < candle.Price() {
					continue
				}
				price = order.limitPrice
				break
			} else if order.Direction() == gctorder.Sell {
				if order.limitPrice > candle.Price() {
					continue
				}
				price = order.limitPrice
				break
			}
		default:
			break
		}

		order.amountFilled = order.amount
		order.avgFillPrice = price
		order.fillTime = time.Now()
		order.status = gctorder.Filled

		fee, err := e.ExchangeFee.Calculate(order.amount, order.avgFillPrice)
		if err != nil {
			return order, err
		}
		order.fee = fee

		order.cost = +fee

		tx, err := portfolio.OnFill(order)
		if err != nil {
			continue
		}
		t.Stats.TrackTransaction(tx)

		*orders = append((*orders)[:i], (*orders)[i+1:]...)
	}

	return nil, nil
}
