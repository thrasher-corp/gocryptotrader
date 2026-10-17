package bitstamp

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func (e *Exchange) handleWSOrderbook(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	var ob WebsocketOrderbook
	if err := json.Unmarshal(data, &ob); err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              ob.Bids.Levels(),
		Asks:              ob.Asks.Levels(),
		Pair:              pair,
		LastUpdated:       ob.Microtimestamp.Time(),
		Asset:             a,
		Exchange:          e.Name,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	filterOrderbookZeroBidPrice(book)
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// handleWSDetailOrderbook loads a snapshot of the top orders of an orderbook, which repeats the price of each order at
// a price level
func (e *Exchange) handleWSDetailOrderbook(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	var ob WebsocketDetailOrderbook
	if err := json.Unmarshal(data, &ob); err != nil {
		return err
	}
	bids, err := orderLevels(ob.Bids)
	if err != nil {
		return err
	}
	asks, err := orderLevels(ob.Asks)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Pair:              pair,
		LastUpdated:       ob.Microtimestamp.Time(),
		Asset:             a,
		Exchange:          e.Name,
		PriceDuplication:  true,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	filterOrderbookZeroBidPrice(book)
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// orderLevels returns orderbook levels which each hold an order
func orderLevels(levels []OrderbookLevel) (orderbook.Levels, error) {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		if levels[i].OrderID > math.MaxInt64 {
			return nil, fmt.Errorf("%w: %d", errOrderIDOverflow, levels[i].OrderID)
		}
		l[i] = orderbook.Level{Price: levels[i].Price, Amount: levels[i].Amount, ID: int64(levels[i].OrderID)}
	}
	return l, nil
}

func (e *Exchange) handleWSTrade(data []byte, pair currency.Pair, a asset.Item) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var t WebsocketTrade
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		Timestamp:    t.Microtimestamp.Time(),
		CurrencyPair: pair,
		AssetType:    a,
		Exchange:     e.Name,
		Price:        t.Price.Float64(),
		Amount:       t.Amount.Float64(),
		Side:         t.Side.Side(),
		TID:          strconv.FormatUint(t.ID, 10),
	})
}

// handleWSFundingRate relays the funding rate, and the funding update with its mark and index prices
func (e *Exchange) handleWSFundingRate(ctx context.Context, data []byte, pair currency.Pair, a asset.Item) error {
	rate := &WebsocketFundingRate{Pair: pair, AssetType: a}
	if err := json.Unmarshal(data, rate); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, websocket.FundingData{
		Timestamp:    rate.Timestamp.Time(),
		CurrencyPair: pair,
		AssetType:    a,
		Exchange:     e.Name,
		Rate:         rate.FundingRate.Float64(),
	}); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, rate)
}

func (e *Exchange) handleWSAnnouncement(ctx context.Context, event string, data []byte) error {
	announcement := &WebsocketAnnouncement{Event: event}
	if err := json.Unmarshal(data, announcement); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, announcement)
}
