package coinbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// Level2 update sides
const (
	wsLevel2SideBid   = "bid"
	wsLevel2SideOffer = "offer"
)

var errLevel2SideUnknown = errors.New("level2 side unknown")

// websocketProduct returns the asset and pair of a product a message names, spot or US derivatives
func (e *Exchange) websocketProduct(productID string) (asset.Item, currency.Pair, error) {
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		if p, err := e.MatchSymbolWithAvailablePairs(productID, a, true); err == nil {
			return a, p, nil
		}
	}
	return asset.Empty, currency.EMPTYPAIR, fmt.Errorf("%w: %s", errWebsocketUnknownProduct, productID)
}

// websocketPairs returns the asset of a product a message names and the enabled pairs it streams for: the product's
// own pair and the USDC aliases sharing its book, which Coinbase streams under its ID
func (e *Exchange) websocketPairs(productID string) (asset.Item, currency.Pairs, error) {
	a, p, err := e.websocketProduct(productID)
	if err != nil {
		return asset.Empty, nil, err
	}
	candidates := currency.Pairs{p}
	if a == asset.Spot {
		if aliases := e.pairAliases.GetAlias(p); len(aliases) != 0 {
			candidates = aliases
		} else if p.Quote.Equal(currency.USD) && !p.Base.Equal(currency.USDT) && !p.Base.Equal(currency.EURC) {
			// Until product data loads the aliases, the documented rule names them
			candidates = append(candidates, currency.NewPairWithDelimiter(p.Base.String(), currency.USDC.String(), p.Delimiter))
		}
	}
	pairs := make(currency.Pairs, 0, len(candidates))
	for _, candidate := range candidates {
		if enabled, err := e.IsPairEnabled(candidate, a); err == nil && enabled {
			pairs = append(pairs, candidate)
		}
	}
	return a, pairs, nil
}

// wsHandleTicker stores and relays tickers from the ticker and ticker_batch channels. A batched ticker leaves out the
// best bid and ask, so it keeps those of the pair's stored ticker
func (e *Exchange) wsHandleTicker(ctx context.Context, envelope *WebsocketEnvelope) error {
	var events []WebsocketTickerEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	batched := envelope.Channel == ChannelTickerBatch
	var tickers []ticker.Price
	var errs error
	for i := range events {
		for j := range events[i].Tickers {
			t := &events[i].Tickers[j]
			a, pairs, err := e.websocketPairs(t.ProductID)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			for _, p := range pairs {
				price := ticker.Price{
					ExchangeName:        e.Name,
					Pair:                p,
					AssetType:           a,
					LastUpdated:         envelope.Timestamp,
					Last:                t.Price.Float64(),
					High:                t.High24Hour.Float64(),
					Low:                 t.Low24Hour.Float64(),
					BaseVolume:          t.Volume24Hour.Float64(),
					PercentChange24Hour: t.PricePercentageChange24Hour.Float64(),
					Bid:                 t.BestBid.Float64(),
					BidSize:             t.BestBidQuantity.Float64(),
					Ask:                 t.BestAsk.Float64(),
					AskSize:             t.BestAskQuantity.Float64(),
				}
				tickers = append(tickers, price)
			}
		}
	}
	if len(tickers) == 0 {
		return errs
	}
	// A batched ticker carries no best bid or ask, and neither carries the quote volume REST tickers do
	processed, err := e.processMergedTickers(tickers, batched, true)
	errs = common.AppendError(errs, err)
	if len(processed) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, processed))
}

// wsHandleCandles relays each pair's five minute candles; a candle is sent again each second it changes
func (e *Exchange) wsHandleCandles(ctx context.Context, envelope *WebsocketEnvelope) error {
	var events []WebsocketCandlesEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var items []kline.Item
	index := make(map[currency.Pair]int)
	var errs error
	for i := range events {
		for j := range events[i].Candles {
			c := &events[i].Candles[j]
			a, pairs, err := e.websocketPairs(c.ProductID)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			candle := kline.Candle{
				Time:   c.Start.Time(),
				Open:   c.Open.Float64(),
				High:   c.High.Float64(),
				Low:    c.Low.Float64(),
				Close:  c.Close.Float64(),
				Volume: c.Volume.Float64(),
			}
			for _, p := range pairs {
				k, ok := index[p]
				if !ok {
					k = len(items)
					index[p] = k
					items = append(items, kline.Item{Exchange: e.Name, Pair: p, Asset: a, Interval: kline.FiveMin})
				}
				items[k].Candles = append(items[k].Candles, candle)
			}
		}
	}
	if len(items) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, items))
}

// wsHandleMarketTrades relays and saves trades as the trade feed and trade saving settings allow. Coinbase sends the
// maker's side of each trade, so a trade's side is the taker's, its opposite
func (e *Exchange) wsHandleMarketTrades(ctx context.Context, envelope *WebsocketEnvelope) error {
	tradeFeed, saveTrades := e.IsTradeFeedEnabled(), e.IsSaveTradeDataEnabled()
	if !tradeFeed && !saveTrades {
		return nil
	}
	var events []WebsocketMarketTradesEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var trades []trade.Data
	var errs error
	for i := range events {
		for j := range events[i].Trades {
			t := &events[i].Trades[j]
			a, pairs, err := e.websocketPairs(t.ProductID)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			side, err := takerSide(t.Side)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			for _, p := range pairs {
				trades = append(trades, trade.Data{
					TID:          t.TradeID,
					Exchange:     e.Name,
					CurrencyPair: p,
					AssetType:    a,
					Side:         side,
					Price:        t.Price.Float64(),
					Amount:       t.Size.Float64(),
					Timestamp:    t.Time,
				})
			}
		}
	}
	if len(trades) == 0 {
		return errs
	}
	if tradeFeed {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, trades))
	}
	if saveTrades {
		errs = common.AppendError(errs, trade.AddTradesToBuffer(trades...))
	}
	return errs
}

// takerSide returns the taker's side of a trade from the maker's side Coinbase reports
func takerSide(makerSide string) (order.Side, error) {
	side, err := order.StringToOrderSide(makerSide)
	if err != nil {
		return order.UnknownSide, err
	}
	switch side {
	case order.Buy:
		return order.Sell, nil
	case order.Sell:
		return order.Buy, nil
	default:
		return order.UnknownSide, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, makerSide)
	}
}

// wsHandleLevel2 loads a book snapshot or applies an update, whose levels replace the book's, a zero quantity removing
// one. A book an update cannot apply to is resubscribed, which sends a new snapshot; until it arrives its updates are
// dropped
func (e *Exchange) wsHandleLevel2(ctx context.Context, conn websocket.Connection, envelope *WebsocketEnvelope) error {
	var events []WebsocketLevel2Event
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var errs error
	for i := range events {
		ev := &events[i]
		a, pairs, err := e.websocketPairs(ev.ProductID)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		bids, asks, err := level2Levels(ev.Updates)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", ev.ProductID, err))
			continue
		}
		for _, p := range pairs {
			switch ev.Type {
			case wsEventSnapshot:
				errs = common.AppendError(errs, e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
					Exchange:          e.Name,
					Pair:              p,
					Asset:             a,
					Bids:              bids,
					Asks:              asks,
					LastUpdated:       envelope.Timestamp,
					ValidateOrderbook: e.ValidateOrderbook,
				}))
			case wsEventUpdate:
				err := e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
					Pair:       p,
					Asset:      a,
					Bids:       bids,
					Asks:       asks,
					UpdateTime: envelope.Timestamp,
				})
				if errors.Is(err, orderbook.ErrOrderbookInvalid) || errors.Is(err, orderbook.ErrDepthNotFound) {
					if e.wsResubscribeBook(ctx, conn, a, ev.ProductID) {
						// The book awaits its new snapshot
						err = nil
					}
				}
				errs = common.AppendError(errs, err)
			default:
				errs = common.AppendError(errs, fmt.Errorf("%w: %s %q", errWebsocketEventTypeUnknown, wsChannelLevel2Data, ev.Type))
			}
		}
	}
	return errs
}

// level2Levels returns an event's bid and offer levels
func level2Levels(updates []WebsocketLevel2Update) (bids, asks orderbook.Levels, err error) {
	bids = make(orderbook.Levels, 0, len(updates))
	asks = make(orderbook.Levels, 0, len(updates))
	for i := range updates {
		level := orderbook.Level{Price: updates[i].PriceLevel.Float64(), Amount: updates[i].NewQuantity.Float64()}
		switch updates[i].Side {
		case wsLevel2SideBid:
			bids = append(bids, level)
		case wsLevel2SideOffer:
			asks = append(asks, level)
		default:
			return nil, nil, fmt.Errorf("%w: %q", errLevel2SideUnknown, updates[i].Side)
		}
	}
	return bids, asks, nil
}

// wsResubscribeBook resubscribes the connection's level2 subscription streaming a product, in the background, so
// Coinbase sends a new snapshot of its book; it reports whether the subscription is resubscribing
func (e *Exchange) wsResubscribeBook(ctx context.Context, conn websocket.Connection, a asset.Item, productID string) bool {
	for _, s := range conn.Subscriptions().List() {
		if s.QualifiedChannel != wsChannelLevel2 || s.Asset != a || len(s.Pairs) != 1 || e.websocketStreamProduct(a, s.Pairs[0]) != productID {
			continue
		}
		if e.wsConnections.get(conn).isResubscribing(s) {
			return true
		}
		e.wsInvalidateBooks(s)
		e.wsResubscribe(ctx, conn, subscription.List{s}, websocketMessageEPL)
		return true
	}
	return false
}

// wsInvalidateBooks invalidates the books a level2 subscription streams, its pair's and its USDC aliases', until a new
// snapshot loads them
func (e *Exchange) wsInvalidateBooks(s *subscription.Subscription) {
	if len(s.Pairs) != 1 {
		return
	}
	_, pairs, err := e.websocketPairs(e.websocketStreamProduct(s.Asset, s.Pairs[0]))
	if err != nil {
		return
	}
	for _, p := range pairs {
		// A book not yet loaded has nothing to invalidate
		_ = e.Websocket.Orderbook.InvalidateOrderbook(p, s.Asset)
	}
}

// wsHandleStatus relays products' trading status
func (e *Exchange) wsHandleStatus(ctx context.Context, envelope *WebsocketEnvelope) error {
	var events []WebsocketStatusEvent
	if err := json.Unmarshal(envelope.Events, &events); err != nil {
		return err
	}
	var errs error
	for i := range events {
		if len(events[i].Products) != 0 {
			errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, events[i].Products))
		}
	}
	return errs
}
