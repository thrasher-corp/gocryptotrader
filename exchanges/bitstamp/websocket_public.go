package bitstamp

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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

// seedOrderbook loads the REST orderbook of each of an orderbook subscription's markets, before the subscription is sent
// The orderbook channels only send a book when it changes, so a quiet market would otherwise have no book until it next
// changes. Loading the book before subscribing lets the books the channel sends afterwards replace it
func (e *Exchange) seedOrderbook(ctx context.Context, s *subscription.Subscription) error {
	detail := subscriptionNames[s.Channel] == channelDetailOrderbook
	grouping := OrderbookGroupedByPrice
	if detail {
		grouping = OrderbookUngroupedWithOrderIDs
	}
	var errs error
	for _, p := range s.Pairs {
		if err := e.seedMarketOrderbook(ctx, formatMarketSymbol(p), grouping, detail); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s %s: %w", errOrderbookSeed, p, s.Asset, err))
		}
	}
	return errs
}

// seedMarketOrderbook loads the REST orderbook of a market under the pair its channel's books are loaded under
func (e *Exchange) seedMarketOrderbook(ctx context.Context, market string, grouping OrderbookGrouping, detail bool) error {
	pair, a, err := e.marketPair(market)
	if err != nil {
		return err
	}
	fPair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return err
	}
	ob, err := e.GetOrderbook(ctx, fPair, grouping)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Pair:              pair,
		LastUpdated:       ob.Microtimestamp.Time(),
		Asset:             a,
		Exchange:          e.Name,
		PriceDuplication:  detail,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	if detail {
		if book.Bids, err = orderLevels(ob.Bids); err != nil {
			return err
		}
		if book.Asks, err = orderLevels(ob.Asks); err != nil {
			return err
		}
	} else {
		book.Bids, book.Asks = orderbookLevels(ob.Bids), orderbookLevels(ob.Asks)
	}
	filterOrderbookZeroBidPrice(book)
	return e.Websocket.Orderbook.LoadSnapshot(ctx, book)
}

// isOrderbookSubscription returns whether a subscription is to one of the orderbook channels
func isOrderbookSubscription(s *subscription.Subscription) bool {
	name := subscriptionNames[s.Channel]
	return name == channelOrderbook || name == channelDetailOrderbook
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
