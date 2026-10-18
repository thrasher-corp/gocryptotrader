package bybit

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
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

// Errors from decoding public stream pushes
var (
	errMalformedTopic = errors.New("malformed topic")
)

const (
	// defaultOrderbookLevels is the depth of an orderbook subscription that sets none; option books come in 25 and
	// 100 levels only, and accept but never push any other depth
	defaultOrderbookLevels       = 50
	defaultOptionOrderbookLevels = 25
)

// orderbookLevels returns the depth of an orderbook subscription to the asset's stream, or the asset's default when it
// sets none
func orderbookLevels(s *subscription.Subscription, a asset.Item) int {
	switch {
	case s.Levels != 0:
		return s.Levels
	case a == asset.Options:
		return defaultOptionOrderbookLevels
	}
	return defaultOrderbookLevels
}

// wsHandleData handles a message from the public stream of assetType
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, assetType asset.Item, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return err
	}
	if topic == "" {
		return e.handleNoTopicWebsocketResponse(ctx, conn, respRaw)
	}
	name, rest, _ := strings.Cut(topic, ".")
	switch name {
	case chanOrderbook:
		switch depth, _, _ := strings.Cut(rest, "."); depth {
		case "rpi":
			return relayWebsocketResponse[WebsocketRPIOrderbookResponse](ctx, e, respRaw)
		case "full":
			// Full depth pushes deltas only, which need a REST snapshot to build a book from, so they are relayed as is
			return relayWebsocketResponse[WebsocketOrderbookResponse](ctx, e, respRaw)
		}
		return e.wsProcessOrderbook(ctx, assetType, respRaw)
	case chanPublicTrade:
		return e.wsProcessPublicTrade(assetType, respRaw)
	case chanPublicTicker:
		return e.wsProcessPublicTicker(ctx, assetType, respRaw)
	case chanKline:
		return e.wsProcessKline(ctx, assetType, respRaw)
	case chanAllLiquidation:
		return relayWebsocketResponse[WebsocketAllLiquidationResponse](ctx, e, respRaw)
	case chanInsurancePool:
		return relayWebsocketResponse[WebsocketInsurancePoolResponse](ctx, e, respRaw)
	case chanPriceLimit:
		return relayWebsocketResponse[WebsocketPriceLimitResponse](ctx, e, respRaw)
	case chanADLAlert:
		return relayWebsocketResponse[WebsocketADLAlertResponse](ctx, e, respRaw)
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// relayWebsocketResponse decodes a push that has no common GCT type and relays it as is
func relayWebsocketResponse[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	resp := new(T)
	if err := json.Unmarshal(respRaw, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

func (e *Exchange) wsProcessOrderbook(ctx context.Context, assetType asset.Item, respRaw []byte) error {
	var resp WebsocketOrderbookResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	cp, err := e.MatchSymbolWithAvailablePairs(resp.Data.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	if resp.Type == snapshotType {
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Pair:         cp,
			Exchange:     e.Name,
			Asset:        assetType,
			LastUpdated:  resp.CreationTimestamp.Time(),
			LastUpdateID: int64(resp.Data.UpdateID), //nolint:gosec // Update IDs stay far below the int64 limit
			LastPushed:   resp.Timestamp.Time(),
			Asks:         resp.Data.Asks.Levels(),
			Bids:         resp.Data.Bids.Levels(),
		})
	}
	return e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
		Pair:       cp,
		Asks:       resp.Data.Asks.Levels(),
		Bids:       resp.Data.Bids.Levels(),
		Asset:      assetType,
		UpdateID:   int64(resp.Data.UpdateID), //nolint:gosec // Update IDs stay far below the int64 limit
		UpdateTime: resp.CreationTimestamp.Time(),
		LastPushed: resp.Timestamp.Time(),
		AllowEmpty: true,
	})
}

func (e *Exchange) wsProcessPublicTrade(assetType asset.Item, respRaw []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketPublicTradeResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(resp.Data))
	for i := range resp.Data {
		t := &resp.Data[i]
		cp, err := e.publicTradePair(t.Symbol, assetType)
		if err != nil {
			return err
		}
		if cp.IsEmpty() {
			continue
		}
		side, err := order.StringToOrderSide(t.Side)
		if err != nil {
			return err
		}
		trades = append(trades, trade.Data{
			Timestamp:    t.OrderFillTimestamp.Time(),
			CurrencyPair: cp,
			AssetType:    assetType,
			Exchange:     e.Name,
			Price:        t.Price.Float64(),
			Amount:       t.Size.Float64(),
			Side:         side,
			TID:          t.TradeID,
		})
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// publicTradePair returns the pair a public trade is for, or an empty pair for an option trade to skip: an option
// trade topic covers every contract of its base coin, including contracts that are not enabled or were not listed
// when the pairs were last updated
func (e *Exchange) publicTradePair(symbol string, a asset.Item) (currency.Pair, error) {
	if a != asset.Options {
		return e.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
	}
	cp, enabled, err := e.MatchSymbolCheckEnabled(symbol, a, true)
	if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
		return currency.EMPTYPAIR, err
	}
	if !enabled {
		return currency.EMPTYPAIR, nil
	}
	return cp, nil
}

func (e *Exchange) wsProcessPublicTicker(ctx context.Context, assetType asset.Item, respRaw []byte) error {
	var resp WebsocketTickerResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	p, err := e.MatchSymbolWithAvailablePairs(resp.Data.Symbol, assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	tick := &ticker.Price{Pair: p, ExchangeName: e.Name, AssetType: assetType}
	if resp.Type != snapshotType {
		// A delta holds only the fields that changed since the last push
		if tick, err = e.GetCachedTicker(p, assetType); err != nil {
			return err
		}
	}
	updateTicker(tick, &resp.Data)
	tick.LastUpdated = resp.Timestamp.Time()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// updateTicker applies the non-zero fields of a ticker push to tick; a delta omits the fields that did not change, so
// a zero value cannot be told apart from an omitted one
func updateTicker(tick *ticker.Price, t *WebsocketTicker) {
	setIfNonZero := func(dst *float64, v float64) {
		if v != 0 {
			*dst = v
		}
	}
	setIfNonZero(&tick.Last, t.LastPrice.Float64())
	setIfNonZero(&tick.High, t.HighPrice24Hour.Float64())
	setIfNonZero(&tick.Low, t.LowPrice24Hour.Float64())
	// Linear and inverse tickers carry the best bid and ask as bid1 and ask1, option tickers without the 1, and spot
	// tickers carry neither
	setIfNonZero(&tick.Bid, t.Bid1Price.Float64())
	setIfNonZero(&tick.BidSize, t.Bid1Size.Float64())
	setIfNonZero(&tick.Ask, t.Ask1Price.Float64())
	setIfNonZero(&tick.AskSize, t.Ask1Size.Float64())
	setIfNonZero(&tick.Bid, t.BidPrice.Float64())
	setIfNonZero(&tick.BidSize, t.BidSize.Float64())
	setIfNonZero(&tick.Ask, t.AskPrice.Float64())
	setIfNonZero(&tick.AskSize, t.AskSize.Float64())
	// volume24h counts the quote currency on the inverse category and turnover24h the base, the reverse of everywhere
	// else, so the two are mapped together rather than one field at a time
	baseVolume, quoteVolume := t.Volume24Hour.Float64(), t.Turnover24Hour.Float64()
	if tick.AssetType == asset.CoinMarginedFutures {
		baseVolume, quoteVolume = quoteVolume, baseVolume
	}
	setIfNonZero(&tick.BaseVolume, baseVolume)
	setIfNonZero(&tick.QuoteVolume, quoteVolume)
	setIfNonZero(&tick.MarkPrice, t.MarkPrice.Float64())
	setIfNonZero(&tick.IndexPrice, t.IndexPrice.Float64())
	setIfNonZero(&tick.OpenInterest, t.OpenInterest.Float64())
	setIfNonZero(&tick.OpenInterestValue, t.OpenInterestValue.Float64())
}

func (e *Exchange) wsProcessKline(ctx context.Context, assetType asset.Item, respRaw []byte) error {
	var resp WebsocketKlineResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	// Candles carry no symbol, so it comes from the kline.{interval}.{symbol} topic
	parts := strings.SplitN(resp.Topic, ".", 3)
	if len(parts) != 3 {
		return fmt.Errorf("%w: %s", errMalformedTopic, resp.Topic)
	}
	cp, err := e.MatchSymbolWithAvailablePairs(parts[2], assetType, hasPotentialDelimiter(assetType))
	if err != nil {
		return err
	}
	for i := range resp.Data {
		k := &resp.Data[i]
		interval, err := stringToInterval(k.Interval)
		if err != nil {
			return err
		}
		// As with tickers, volume counts the quote currency on the inverse category and turnover the base
		volume, quoteVolume := k.Volume.Float64(), k.Turnover.Float64()
		if assetType == asset.CoinMarginedFutures {
			volume, quoteVolume = quoteVolume, volume
		}
		if err := e.Websocket.DataHandler.Send(ctx, kline.Item{
			Pair:     cp,
			Asset:    assetType,
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:        k.Start.Time(),
				Open:        k.Open.Float64(),
				Close:       k.Close.Float64(),
				High:        k.High.Float64(),
				Low:         k.Low.Float64(),
				Volume:      volume,
				QuoteVolume: quoteVolume,
			}},
		}); err != nil {
			return err
		}
	}
	return nil
}

// generatePublicDefaultSubscriptions returns the orderbook, trade and ticker subscriptions for every enabled pair of a
// public asset other than spot, which follows the configured subscriptions instead
func (e *Exchange) generatePublicDefaultSubscriptions(a asset.Item) (subscription.List, error) {
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil
		}
		return nil, err
	}
	levels := defaultOrderbookLevels
	if a == asset.Options {
		levels = defaultOptionOrderbookLevels
	}
	var subs subscription.List
	for _, p := range pairs {
		subs = append(subs,
			&subscription.Subscription{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{p}, Asset: a, Levels: levels},
			&subscription.Subscription{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{p}, Asset: a})
		if a != asset.Options {
			subs = append(subs, &subscription.Subscription{Enabled: true, Channel: chanPublicTrade, Pairs: currency.Pairs{p}, Asset: a})
		}
	}
	if a == asset.Options {
		// An option trade topic covers every contract of its base coin
		var bases []currency.Code
		for _, p := range pairs {
			if !p.Base.IsEmpty() && !slices.ContainsFunc(bases, p.Base.Equal) {
				bases = append(bases, p.Base)
			}
		}
		for _, base := range bases {
			var basePairs currency.Pairs
			for _, p := range pairs {
				if p.Base.Equal(base) {
					basePairs = append(basePairs, p)
				}
			}
			subs = append(subs, &subscription.Subscription{Enabled: true, Channel: chanPublicTrade, Pairs: basePairs, Asset: a})
		}
	}
	valid, err := e.qualifyPublicSubscriptions(a, subs)
	if err != nil {
		return nil, err
	}
	return valid, nil
}

// qualifyPublicSubscriptions sets the topic of each subscription for asset a that has none, and returns the
// subscriptions that have one, with an error naming any that cannot
func (e *Exchange) qualifyPublicSubscriptions(a asset.Item, subs subscription.List) (subscription.List, error) {
	valid := make(subscription.List, 0, len(subs))
	var errs error
	for _, s := range subs {
		if s.QualifiedChannel == "" {
			topic, err := e.publicTopic(s, a)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionFailure, s, err))
				continue
			}
			s.QualifiedChannel = topic
		}
		valid = append(valid, s)
	}
	return valid, errs
}

// publicTopic returns the topic a public subscription for asset a subscribes to
func (e *Exchange) publicTopic(s *subscription.Subscription, a asset.Item) (string, error) {
	name := channelName(s)
	switch {
	case name == chanInsurancePool, name == chanADLAlert:
		group, err := contractGroup(a)
		if err != nil {
			return "", err
		}
		return name + "." + group, nil
	case name == chanPublicTrade && a == asset.Options:
		return optionTradeTopic(s.Pairs)
	case len(s.Pairs) != 1:
		return "", subscription.ErrNotSinglePair
	}
	symbol, err := e.symbol(s.Pairs[0], a)
	if err != nil {
		return "", err
	}
	switch name {
	case chanOrderbook:
		return name + "." + strconv.Itoa(orderbookLevels(s, a)) + "." + symbol, nil
	case chanKline:
		interval, err := intervalToString(s.Interval)
		if err != nil {
			return "", fmt.Errorf("%w: %w", subscription.ErrInvalidInterval, err)
		}
		return name + "." + interval + "." + symbol, nil
	}
	return name + "." + symbol, nil
}

// contractGroup returns the contract group that the insurance pool and ADL alert topics of an asset are named for
func contractGroup(a asset.Item) (string, error) {
	switch a {
	case asset.USDTMarginedFutures:
		return "USDT", nil
	case asset.USDCMarginedFutures:
		return "USDC", nil
	case asset.CoinMarginedFutures:
		return "inverse", nil
	}
	return "", fmt.Errorf("%w %s", errContractGroupNotSupported, a)
}

// optionTradeTopic returns the trade topic of the base coin the option pairs share
func optionTradeTopic(pairs currency.Pairs) (string, error) {
	if len(pairs) == 0 {
		return "", currency.ErrCurrencyPairsEmpty
	}
	base := pairs[0].Base
	for _, p := range pairs[1:] {
		if !p.Base.Equal(base) {
			return "", fmt.Errorf("%w: %s", errOptionTradesNotOneBaseCoin, pairs)
		}
	}
	return chanPublicTrade + "." + base.Upper().String(), nil
}
