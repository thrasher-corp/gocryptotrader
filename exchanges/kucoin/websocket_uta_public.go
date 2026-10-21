package kucoin

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
	"github.com/thrasher-corp/gocryptotrader/log"
)

// errUTAOrderbookSequenceGap is returned when an orderbook delta skips changes the stored orderbook has not seen
var errUTAOrderbookSequenceGap = errors.New("UTA orderbook delta does not follow on from the stored orderbook")

// processUTATicker stores a ticker push under each asset its symbol trades as and relays the stored tickers. The push
// carries no high, low or volumes, so the stored ticker keeps those
func (e *Exchange) processUTATicker(ctx context.Context, msg *UTAWebsocketMessage, market string) error {
	var resp UTAWebsocketTicker
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	pair, assets, err := e.utaPairAndAssets(resp.Symbol, market)
	if err != nil {
		return err
	}
	tickers := make([]ticker.Price, 0, len(assets))
	e.tickerMergeMu.Lock()
	for _, a := range assets {
		t, err := e.storedUTATicker(pair, a)
		if err != nil {
			e.tickerMergeMu.Unlock()
			return err
		}
		t.Last = resp.LastTradedPrice.Float64()
		t.LastSize = resp.LastTradedSize.Float64()
		t.Bid = resp.BestBidPrice.Float64()
		t.BidSize = resp.BestBidSize.Float64()
		t.Ask = resp.BestAskPrice.Float64()
		t.AskSize = resp.BestAskSize.Float64()
		t.LastUpdated = resp.MatchTimestamp.Time()
		tickers = append(tickers, *t)
	}
	processed, err := ticker.ProcessBatch(tickers)
	e.tickerMergeMu.Unlock()
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// storedUTATicker returns the stored ticker of a pair, or a new one when none is stored yet
func (e *Exchange) storedUTATicker(pair currency.Pair, a asset.Item) (*ticker.Price, error) {
	t, err := ticker.GetTicker(e.Name, pair, a)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, ticker.ErrTickerNotFound) {
		return nil, err
	}
	return &ticker.Price{ExchangeName: e.Name, Pair: pair, AssetType: a}, nil
}

// processUTAKline relays a kline push for each asset its symbol trades as
func (e *Exchange) processUTAKline(ctx context.Context, msg *UTAWebsocketMessage, market string) error {
	var resp UTAWebsocketKline
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	interval, err := utaIntervalFromString(resp.Interval)
	if err != nil {
		return err
	}
	pair, assets, err := e.utaPairAndAssets(resp.Symbol, market)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if err := e.Websocket.DataHandler.Send(ctx, &kline.Item{
			Exchange: e.Name,
			Pair:     pair,
			Asset:    a,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:        resp.StartTime.Time(),
				Open:        resp.OpenPrice.Float64(),
				High:        resp.HighPrice.Float64(),
				Low:         resp.LowPrice.Float64(),
				Close:       resp.ClosePrice.Float64(),
				Volume:      resp.TransactionVolume.Float64(),
				QuoteVolume: resp.TransactionAmount.Float64(),
			}},
		}); err != nil {
			return err
		}
	}
	return nil
}

// utaIntervalFromString returns the interval a kline push names
func utaIntervalFromString(name string) (kline.Interval, error) {
	for i, n := range utaKlineIntervals {
		if n == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, name)
}

// processUTATrade relays a public trade push for each asset its symbol trades as, when trades are saved or fed
func (e *Exchange) processUTATrade(msg *UTAWebsocketMessage, market string) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp UTAWebsocketTrade
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	pair, assets, err := e.utaPairAndAssets(resp.Symbol, market)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(assets))
	for i, a := range assets {
		trades[i] = trade.Data{
			TID:          resp.TradeID,
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    a,
			Side:         side,
			Price:        resp.Price.Float64(),
			Amount:       resp.Size.Float64(),
			Timestamp:    resp.MatchTimestamp.Time(),
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// processUTAOrderbook loads an orderbook snapshot push, or applies a delta push to the stored orderbook. An orderbook
// that missed changes is resubscribed to, as KuCoin pushes a fresh snapshot on subscribing
func (e *Exchange) processUTAOrderbook(ctx context.Context, conn websocket.Connection, msg *UTAWebsocketMessage, market string) error {
	var resp UTAWebsocketOrderbook
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	pair, assets, err := e.utaPairAndAssets(resp.Symbol, market)
	if err != nil {
		return err
	}
	var errs error
	var resync bool
	for _, a := range assets {
		switch msg.UpdateType {
		case "snapshot":
			err = e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
				Exchange:          e.Name,
				Pair:              pair,
				Asset:             a,
				Bids:              utaOrderbookLevels(resp.Bids, true),
				Asks:              utaOrderbookLevels(resp.Asks, true),
				LastUpdated:       resp.MatchTimestamp.Time(),
				LastPushed:        msg.PushTimestamp.Time(),
				LastUpdateID:      orderbookUpdateID(resp.SequenceEnd),
				ValidateOrderbook: e.ValidateOrderbook,
			})
		case "delta":
			var desynced bool
			desynced, err = e.applyUTAOrderbookDelta(ctx, msg, &resp, pair, a)
			resync = resync || desynced
		default:
			return fmt.Errorf("%w: orderbook update type %q", errUTAUnknownTopic, msg.UpdateType)
		}
		errs = common.AppendError(errs, err)
	}
	if resync {
		e.resyncUTAOrderbook(ctx, conn, pair)
	}
	return errs
}

// applyUTAOrderbookDelta applies a delta to the stored orderbook and reports whether the orderbook needs a fresh
// snapshot: when the delta's first sequence skips past the orderbook's last, which invalidates the orderbook, and when
// the orderbook is missing or already invalid, whose deltas are dropped while its fresh snapshot is awaited. A delta
// the orderbook already holds is skipped
func (e *Exchange) applyUTAOrderbookDelta(ctx context.Context, msg *UTAWebsocketMessage, resp *UTAWebsocketOrderbook, pair currency.Pair, a asset.Item) (resync bool, err error) {
	lastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(pair, a)
	if errors.Is(err, orderbook.ErrOrderbookInvalid) || errors.Is(err, orderbook.ErrDepthNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if orderbookUpdateID(resp.SequenceStart) > lastUpdateID+1 {
		gapErr := fmt.Errorf("%w: %s %s stored sequence %d, delta starts at %d", errUTAOrderbookSequenceGap, pair, a, lastUpdateID, resp.SequenceStart)
		return true, common.AppendError(gapErr, e.Websocket.Orderbook.InvalidateOrderbook(pair, a))
	}
	err = e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
		UpdateID:   orderbookUpdateID(resp.SequenceEnd),
		UpdateTime: resp.MatchTimestamp.Time(),
		LastPushed: msg.PushTimestamp.Time(),
		Asset:      a,
		Pair:       pair,
		Bids:       utaOrderbookLevels(resp.Bids, false),
		Asks:       utaOrderbookLevels(resp.Asks, false),
		// A delta can move the sequence on without changing a level within the subscribed depth
		AllowEmpty:                 true,
		SkipOutOfOrderLastUpdateID: true,
	})
	return errors.Is(err, orderbook.ErrOrderbookInvalid), err
}

// resyncUTAOrderbook resubscribes to the orderbook subscriptions of a pair, for a fresh snapshot, unless a
// resubscription is already under way. It resubscribes in the background, as the replies arrive through the reader
// calling it
func (e *Exchange) resyncUTAOrderbook(ctx context.Context, conn websocket.Connection, pair currency.Pair) {
	var subs subscription.List
	for _, s := range conn.Subscriptions().List() {
		if s.Channel == UTAOrderbookChannel && s.State() == subscription.SubscribedState && s.Pairs.Contains(pair, true) {
			subs = append(subs, s)
		}
	}
	if len(subs) == 0 {
		return
	}
	go func() {
		if err := e.Websocket.ResubscribeFromConnection(ctx, conn, subs); err != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket: resubscribing to %s for a fresh orderbook snapshot: %v", e.Name, subs, err)
		}
	}()
}

// utaOrderbookLevels converts orderbook push levels. Retail price improvement sizes are left out, as API orders cannot
// fill against them, so a snapshot drops levels holding retail price improvement orders alone; in a delta a zero size
// removes the level
func utaOrderbookLevels(levels []UTAWebsocketOrderbookLevel, snapshot bool) orderbook.Levels {
	out := make(orderbook.Levels, 0, len(levels))
	for i := range levels {
		if snapshot && levels[i].Size == 0 {
			continue
		}
		out = append(out, orderbook.Level{Price: levels[i].Price.Float64(), Amount: levels[i].Size.Float64()})
	}
	return out
}

// processUTAMarkPrice stores the mark price, index price and open interest of a futures contract in its ticker,
// keeping the ticker's other fields, and relays the stored ticker
func (e *Exchange) processUTAMarkPrice(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketMarkPrice
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, asset.Futures, false)
	if err != nil {
		return err
	}
	e.tickerMergeMu.Lock()
	t, err := e.storedUTATicker(pair, asset.Futures)
	if err == nil {
		t.MarkPrice = resp.MarkPrice.Float64()
		t.IndexPrice = resp.IndexPrice.Float64()
		t.OpenInterest = resp.OpenInterest.Float64()
		t.LastUpdated = resp.Timestamp.Time()
		err = ticker.ProcessTicker(t)
	}
	e.tickerMergeMu.Unlock()
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, t)
}

// processUTAFundingFee relays a funding fee rate push
func (e *Exchange) processUTAFundingFee(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketFundingFee
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processUTAAllFundingFeeRates relays an all funding fee rates push
func (e *Exchange) processUTAAllFundingFeeRates(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp []UTAWebsocketFundingFee
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processUTACallAuctionInfo relays a call auction push
func (e *Exchange) processUTACallAuctionInfo(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketCallAuctionInfo
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
