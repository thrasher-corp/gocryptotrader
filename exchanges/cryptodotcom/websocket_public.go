package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

// The channels Crypto.com names in the pushes of its book, ticker, trade and candlestick channels; a delta book
// subscription's deltas come from the book.update channel
const (
	channelBook        = "book"
	channelBookUpdate  = "book.update"
	channelTicker      = "ticker"
	channelTrade       = "trade"
	channelCandlestick = "candlestick"
)

// A book channel unsubscribed for a new snapshot that cannot be subscribed again is retried up to
// bookResubscribeAttempts times, defaultBookResubscribeDelay after the failure, then after twice as long as the last
const (
	bookResubscribeAttempts     = 5
	defaultBookResubscribeDelay = time.Second
)

var (
	errBookDeltaHasNoUpdate = errors.New("book delta has no update")
	errBookOutOfSync        = errors.New("order book out of sync")
)

// websocketCandleTimeframes are the intervals the candlestick channel serves, by the timeframe Crypto.com names each by
var websocketCandleTimeframes = map[string]kline.Interval{
	"1m":  kline.OneMin,
	"5m":  kline.FiveMin,
	"15m": kline.FifteenMin,
	"30m": kline.ThirtyMin,
	"1h":  kline.OneHour,
	"2h":  kline.TwoHour,
	"4h":  kline.FourHour,
	"12h": kline.TwelveHour,
	"1D":  kline.OneDay,
	"7D":  kline.SevenDay,
	"14D": kline.TwoWeek,
	"1M":  kline.OneMonth,
}

// websocketCandleTimeframe returns the timeframe the candlestick channel names an interval by
func websocketCandleTimeframe(interval kline.Interval) (string, error) {
	for timeframe, i := range websocketCandleTimeframes {
		if i == interval {
			return timeframe, nil
		}
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// websocketCandleInterval returns the interval a candlestick channel's timeframe names
func websocketCandleInterval(timeframe string) (kline.Interval, error) {
	if interval, ok := websocketCandleTimeframes[timeframe]; ok {
		return interval, nil
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrUnsupportedInterval, timeframe)
}

// instrumentAsset is an asset an instrument trades as, with the instrument's pair in that asset
type instrumentAsset struct {
	Asset asset.Item
	Pair  currency.Pair
}

// instrumentAssets returns each asset an instrument name trades as among the available pairs, in asset order, with its
// pair in that asset: a spot instrument trades as spot and as margin, on the one book, and a derivative as its one asset
func (e *Exchange) instrumentAssets(instrument string) ([]instrumentAsset, error) {
	if instrument == "" {
		return nil, errInstrumentNameRequired
	}
	assets := e.GetAssetTypes(false)
	slices.Sort(assets)
	var found []instrumentAsset
	for _, a := range assets {
		p, err := e.MatchSymbolWithAvailablePairs(instrument, a, true)
		if err != nil {
			if errors.Is(err, currency.ErrPairNotFound) {
				continue
			}
			return nil, err
		}
		// Pairs are matched without the name's delimiter, so a match must also be named the same
		name, err := e.FormatSymbol(p, a)
		if err != nil {
			return nil, err
		}
		if name == instrument {
			found = append(found, instrumentAsset{Asset: a, Pair: p})
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("%w: %s", currency.ErrPairNotFound, instrument)
	}
	return found, nil
}

// enabledInstrumentAssets returns the assets an instrument's market data is relayed to: those it trades as whose enabled
// pairs hold it
func (e *Exchange) enabledInstrumentAssets(instrument string) ([]instrumentAsset, error) {
	assets, err := e.instrumentAssets(instrument)
	if err != nil {
		return nil, err
	}
	enabled := assets[:0]
	for _, a := range assets {
		ok, err := e.IsPairEnabled(a.Pair, a.Asset)
		if err != nil {
			return nil, err
		}
		if ok {
			enabled = append(enabled, a)
		}
	}
	return enabled, nil
}

// wsHandleBook loads a book snapshot into the book of each asset its instrument trades as; a snapshot subscription pushes
// a snapshot every 500 milliseconds, and a delta subscription one on subscribing, which its deltas follow
func (e *Exchange) wsHandleBook(ctx context.Context, push *websocketChannelPush) error {
	var books []WebsocketBook
	if err := decodePushData(push, &books); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil {
		return err
	}
	var errs error
	for i := range books {
		for _, a := range assets {
			errs = common.AppendError(errs, e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
				Exchange:          e.Name,
				Pair:              a.Pair,
				Asset:             a.Asset,
				Bids:              orderbook.Levels(books[i].Bids),
				Asks:              orderbook.Levels(books[i].Asks),
				LastUpdated:       bookUpdateTime(&books[i]),
				LastPushed:        books[i].PublishTime.Time(),
				LastUpdateID:      int64(books[i].UpdateID),
				ValidateOrderbook: e.ValidateOrderbook,
			}))
		}
	}
	return errs
}

// wsHandleBookUpdate applies a book delta to the book of each asset its instrument trades as. A delta names the update it
// follows, so one that does not follow the books' last update means one was missed, and the books are invalidated and
// the channel resubscribed for a new snapshot; deltas arriving while the books await it are dropped
func (e *Exchange) wsHandleBookUpdate(ctx context.Context, conn websocket.Connection, push *websocketChannelPush) error {
	var deltas []WebsocketBook
	if err := decodePushData(push, &deltas); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil || len(assets) == 0 {
		return err
	}
	var errs error
	for i := range deltas {
		if deltas[i].Update == nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s", errBookDeltaHasNoUpdate, push.Subscription))
		}
		updateIDs := make([]int64, 0, len(assets))
		for _, a := range assets {
			if id, err := e.Websocket.Orderbook.LastUpdateID(a.Pair, a.Asset); err == nil {
				updateIDs = append(updateIDs, id)
			}
		}
		switch {
		case len(updateIDs) == 0:
			// The books are awaiting a snapshot, which is requested again in case an earlier resubscription failed
			e.resubscribeBook(ctx, conn, push.Subscription)
			return errs
		case len(updateIDs) != len(assets) || slices.ContainsFunc(updateIDs, func(id int64) bool { return id != updateIDs[0] }):
			// A book is missing, from an asset enabled since the snapshot, or out of step with the others
			e.resynchroniseBooks(ctx, conn, push.Subscription, assets)
			return common.AppendError(errs, fmt.Errorf("%w: %s books are not at the same update", errBookOutOfSync, push.Subscription))
		case int64(deltas[i].PreviousUpdateID) != updateIDs[0]:
			e.resynchroniseBooks(ctx, conn, push.Subscription, assets)
			return common.AppendError(errs, fmt.Errorf("%w: %s delta follows update %d, not the last update %d", errBookOutOfSync, push.Subscription, deltas[i].PreviousUpdateID, updateIDs[0]))
		}
		for _, a := range assets {
			if err := e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
				UpdateID:   int64(deltas[i].UpdateID),
				UpdateTime: bookUpdateTime(&deltas[i]),
				LastPushed: deltas[i].PublishTime.Time(),
				Asset:      a.Asset,
				Pair:       a.Pair,
				Bids:       orderbook.Levels(deltas[i].Update.Bids),
				Asks:       orderbook.Levels(deltas[i].Update.Asks),
				// Crypto.com pushes a delta without levels when the book has not changed, which still moves its update on
				AllowEmpty: true,
			}); err != nil {
				errs = common.AppendError(errs, err)
				if errors.Is(err, orderbook.ErrOrderbookInvalid) {
					// The delta broke this book, and the other books, having taken the same deltas, are no sounder
					e.resynchroniseBooks(ctx, conn, push.Subscription, assets)
					return errs
				}
			}
		}
	}
	return errs
}

// bookUpdateTime returns when a book was last updated, else when it was published; Crypto.com's schema gives the update
// time of a snapshot alone, although its deltas, those without levels included, carry it too
func bookUpdateTime(b *WebsocketBook) time.Time {
	if t := b.UpdateTime.Time(); !t.IsZero() {
		return t
	}
	return b.PublishTime.Time()
}

// resynchroniseBooks invalidates the books of an instrument's assets, so that nothing trades on them, and resubscribes
// their book channel for a new snapshot
func (e *Exchange) resynchroniseBooks(ctx context.Context, conn websocket.Connection, channel string, assets []instrumentAsset) {
	for _, a := range assets {
		// Invalidating errors only for a book never loaded, which has nothing to invalidate
		_ = e.Websocket.Orderbook.InvalidateOrderbook(a.Pair, a.Asset)
	}
	e.resubscribeBook(ctx, conn, channel)
}

// resubscribeBook resubscribes a book channel for a new snapshot, apart from the connection's reader, which must go on
// reading for the resubscription's responses to arrive. Nothing is done while the channel resubscribes, nor before it is
// stored against its connection, which a later delta retries, as it does a failed resubscription that leaves the channel
// subscribed; one that leaves it unsubscribed, which no delta follows, is retried here
func (e *Exchange) resubscribeBook(ctx context.Context, conn websocket.Connection, channel string) {
	s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: channel}})
	if s == nil || s.State() != subscription.SubscribedState {
		return
	}
	go func() {
		err := e.Websocket.ResubscribeFromConnection(ctx, conn, subscription.List{s})
		if err == nil || errors.Is(err, subscription.ErrInStateAlready) {
			// The subscription is already resubscribing when another delta started the resubscription first
			return
		}
		log.Errorf(log.WebsocketMgr, "%s websocket: resubscribing %s for a new book snapshot: %v", e.Name, channel, err)
		if s.State() == subscription.ResubscribingState {
			// Only a failed unsubscription leaves the state unchanged, and the channel then is still subscribed
			_ = s.SetState(subscription.SubscribedState)
			return
		}
		e.retryBookSubscription(ctx, conn, s)
	}()
}

// retryBookSubscription subscribes a book channel a failed resubscription left unsubscribed on its connection again,
// waiting twice as long before each attempt as before the last. It stops once the channel is subscribed, or held again,
// as a flush of the subscriptions holds it, once it is no longer configured or its connection is lost, as connecting
// afresh subscribes every channel, or after bookResubscribeAttempts attempts. Each attempt subscribes the one channel,
// as a flush would hold the websocket's lock, which its other requests wait on, while it waits on Crypto.com
func (e *Exchange) retryBookSubscription(ctx context.Context, conn websocket.Connection, s *subscription.Subscription) {
	delay := e.bookResubscribeDelay
	for attempt := 1; attempt <= bookResubscribeAttempts; attempt++ {
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
		if c, ok := conn.(interface{ IsConnected() bool }); ok && !c.IsConnected() {
			return
		}
		if !e.isConfiguredMarketDataChannel(s.QualifiedChannel) {
			return
		}
		err := e.Websocket.SubscribeToChannels(ctx, conn, subscription.List{s})
		switch {
		case err == nil:
			// The manager stores a subscription against its connection only when it subscribes it itself, and a later
			// delta finds its book channel there
			if err := conn.Subscriptions().Add(s); err != nil {
				log.Errorf(log.WebsocketMgr, "%s websocket: storing %s against its connection: %v", e.Name, s.QualifiedChannel, err)
			}
			return
		case errors.Is(err, subscription.ErrDuplicate):
			return
		}
		log.Errorf(log.WebsocketMgr, "%s websocket: subscribing %s again for a new book snapshot, attempt %d of %d: %v", e.Name, s.QualifiedChannel, attempt, bookResubscribeAttempts, err)
		delay *= 2
	}
}

// isConfiguredMarketDataChannel reports whether the configured market data subscriptions still name a qualified
// channel; one that cannot be told is taken as not, so that a channel removed is not subscribed again
func (e *Exchange) isConfiguredMarketDataChannel(channel string) bool {
	subs, err := e.generateMarketDataSubscriptions()
	return err == nil && slices.ContainsFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel == channel })
}

// qualifiedChannelKey finds a subscription by its qualified channel, which names one of Crypto.com's channels
type qualifiedChannelKey struct {
	*subscription.Subscription
}

var _ subscription.MatchableKey = qualifiedChannelKey{}

// Match reports whether a subscription has the key's qualified channel
func (k qualifiedChannelKey) Match(eachKey subscription.MatchableKey) bool {
	s := eachKey.GetSubscription()
	return s != nil && s.QualifiedChannel == k.QualifiedChannel
}

// GetSubscription returns the subscription holding the qualified channel to find
func (k qualifiedChannelKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

// String returns the qualified channel to find
func (k qualifiedChannelKey) String() string {
	return k.QualifiedChannel
}

// wsHandleTicker stores an instrument's ticker for each asset it trades as, then relays the tickers stored
func (e *Exchange) wsHandleTicker(ctx context.Context, push *websocketChannelPush) error {
	var tickers []WebsocketTicker
	if err := decodePushData(push, &tickers); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil {
		return err
	}
	prices := make([]ticker.Price, 0, len(tickers)*len(assets))
	for i := range tickers {
		for _, a := range assets {
			price := ticker.Price{
				Last:                tickers[i].LastPrice.Float64(),
				High:                tickers[i].High24Hour.Float64(),
				Low:                 tickers[i].Low24Hour.Float64(),
				Bid:                 tickers[i].BestBidPrice.Float64(),
				BidSize:             tickers[i].BestBidSize.Float64(),
				Ask:                 tickers[i].BestAskPrice.Float64(),
				AskSize:             tickers[i].BestAskSize.Float64(),
				BaseVolume:          tickers[i].Volume24Hour.Float64(),
				PercentChange24Hour: percentFromFraction(tickers[i].PriceChange24Hour),
				OpenInterest:        tickers[i].OpenInterest.Float64(),
				Pair:                a.Pair,
				ExchangeName:        e.Name,
				AssetType:           a.Asset,
				LastUpdated:         tickers[i].Time.Time(),
			}
			if quotedInUSD(a) {
				price.QuoteVolume = tickers[i].VolumeValue24Hour.Float64()
			}
			prices = append(prices, price)
		}
	}
	processed, err := ticker.ProcessBatch(prices)
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// quotedInUSD reports whether an instrument is quoted in USD, the currency of a ticker's volume value; a derivative's
// underlying is named by its base and quote currencies, such as BTCUSD
func quotedInUSD(a instrumentAsset) bool {
	switch a.Asset {
	case asset.Spot, asset.Margin:
		return a.Pair.Quote.Equal(currency.USD)
	default:
		return strings.HasSuffix(a.Pair.Base.Upper().String(), currency.USD.String())
	}
}

// wsHandleTrades relays an instrument's trades through the trade feed for each asset it trades as; subscribing returns
// the last 50 trades, newest first
func (e *Exchange) wsHandleTrades(push *websocketChannelPush) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var trades []WebsocketTrade
	if err := decodePushData(push, &trades); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil {
		return err
	}
	var errs error
	data := make([]trade.Data, 0, len(trades)*len(assets))
	for i := range trades {
		side, err := order.StringToOrderSide(trades[i].Side)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s trade %s: %w", push.InstrumentName, trades[i].TradeID, err))
			continue
		}
		for _, a := range assets {
			data = append(data, trade.Data{
				TID:          trades[i].TradeID,
				Exchange:     e.Name,
				CurrencyPair: a.Pair,
				AssetType:    a.Asset,
				Side:         side,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Time.Time(),
			})
		}
	}
	return common.AppendError(errs, e.Websocket.Trade.Update(saveTradeData, data...))
}

// wsHandleCandles relays an instrument's candles for each asset it trades as; subscribing returns up to the last 300
// candles, oldest first, and later pushes carry the latest candle as it changes
func (e *Exchange) wsHandleCandles(ctx context.Context, push *websocketChannelPush) error {
	interval, err := websocketCandleInterval(push.Interval)
	if err != nil {
		return err
	}
	var candles []WebsocketCandle
	if err := decodePushData(push, &candles); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil || len(candles) == 0 {
		return err
	}
	klineCandles := make([]kline.Candle, len(candles))
	for i := range candles {
		klineCandles[i] = kline.Candle{
			Time:   candles[i].Time.Time(),
			Open:   candles[i].Open.Float64(),
			High:   candles[i].High.Float64(),
			Low:    candles[i].Low.Float64(),
			Close:  candles[i].Close.Float64(),
			Volume: candles[i].Volume.Float64(),
		}
	}
	var errs error
	for _, a := range assets {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, kline.Item{
			Exchange: e.Name,
			Pair:     a.Pair,
			Asset:    a.Asset,
			Interval: interval,
			// Each asset's candles are its own, so that a receiver changing them changes no other asset's
			Candles: slices.Clone(klineCandles),
		}))
	}
	return errs
}

// wsHandleIndexPrice relays an index's price
func (e *Exchange) wsHandleIndexPrice(ctx context.Context, push *websocketChannelPush) error {
	var prices []WebsocketIndexPrice
	if err := decodePushData(push, &prices); err != nil {
		return err
	}
	var errs error
	for i := range prices {
		prices[i].IndexName = push.InstrumentName
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, &prices[i]))
	}
	return errs
}

// wsHandleMarkPrice relays an instrument's mark price for each asset it trades as
func (e *Exchange) wsHandleMarkPrice(ctx context.Context, push *websocketChannelPush) error {
	return relayInstrumentValues[WebsocketMarkPrice](ctx, e, push)
}

// wsHandleSettlementPrice relays a future's settlement price
func (e *Exchange) wsHandleSettlementPrice(ctx context.Context, push *websocketChannelPush) error {
	return relayInstrumentValues[WebsocketSettlementPrice](ctx, e, push)
}

// wsHandleFundingRate relays a perpetual swap's funding rate
func (e *Exchange) wsHandleFundingRate(ctx context.Context, push *websocketChannelPush) error {
	return relayInstrumentValues[WebsocketFundingRate](ctx, e, push)
}

// wsHandleEstimatedFundingRate relays a perpetual swap's estimated funding rate
func (e *Exchange) wsHandleEstimatedFundingRate(ctx context.Context, push *websocketChannelPush) error {
	return relayInstrumentValues[WebsocketEstimatedFundingRate](ctx, e, push)
}

// instrumentValue is a value pushed for an instrument, which is relayed for each asset the instrument trades as
type instrumentValue[T any] interface {
	*T
	setInstrumentAsset(instrumentAsset)
}

// relayInstrumentValues relays a copy of each value a push holds for each asset its instrument trades as
func relayInstrumentValues[T any, V instrumentValue[T]](ctx context.Context, e *Exchange, push *websocketChannelPush) error {
	var values []T
	if err := decodePushData(push, &values); err != nil {
		return err
	}
	assets, err := e.enabledInstrumentAssets(push.InstrumentName)
	if err != nil {
		return err
	}
	var errs error
	for i := range values {
		for _, a := range assets {
			v := values[i]
			V(&v).setInstrumentAsset(a)
			errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, &v))
		}
	}
	return errs
}

func (r *WebsocketMarkPrice) setInstrumentAsset(a instrumentAsset) {
	r.Pair, r.AssetType = a.Pair, a.Asset
}

func (r *WebsocketSettlementPrice) setInstrumentAsset(a instrumentAsset) {
	r.Pair, r.AssetType = a.Pair, a.Asset
}

func (r *WebsocketFundingRate) setInstrumentAsset(a instrumentAsset) {
	r.Pair, r.AssetType = a.Pair, a.Asset
}

func (r *WebsocketEstimatedFundingRate) setInstrumentAsset(a instrumentAsset) {
	r.Pair, r.AssetType = a.Pair, a.Asset
}
