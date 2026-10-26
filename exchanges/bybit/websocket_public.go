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
	"github.com/thrasher-corp/gocryptotrader/types"
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
			return relayWebsocketResponse[WebsocketRPIOrderbookUpdate](ctx, e, respRaw)
		case "full":
			// Full depth pushes deltas only, which need a REST snapshot to build a book from, so they are relayed as is
			return relayWebsocketResponse[WebsocketOrderbookUpdate](ctx, e, respRaw)
		}
		return e.wsProcessOrderbook(ctx, assetType, respRaw)
	case chanPublicTrade:
		return e.wsProcessPublicTrade(assetType, respRaw)
	case chanPublicTicker:
		return e.wsProcessPublicTicker(ctx, assetType, respRaw)
	case chanKline:
		return e.wsProcessKline(ctx, assetType, respRaw)
	case chanAllLiquidation:
		return relayWebsocketResponse[WebsocketAllLiquidationUpdate](ctx, e, respRaw)
	case chanInsurancePool:
		return relayWebsocketResponse[WebsocketInsurancePoolUpdate](ctx, e, respRaw)
	case chanPriceLimit:
		return relayWebsocketResponse[WebsocketPriceLimitUpdate](ctx, e, respRaw)
	case chanADLAlert:
		return relayWebsocketResponse[WebsocketADLAlertUpdate](ctx, e, respRaw)
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
	var resp WebsocketOrderbookUpdate
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
	var resp WebsocketPublicTradeUpdate
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
	fields, err := tickerFields(respRaw)
	if err != nil {
		return err
	}
	var resp WebsocketTickerUpdate
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
	updateTicker(tick, &resp.Data, fields)
	tick.LastUpdated = resp.Timestamp.Time()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// tickerFields returns the fields a ticker push carries a value for: a delta carries only the fields that changed,
// which can change to zero, and an empty string carries no value
func tickerFields(respRaw []byte) (map[string]bool, error) {
	fields := make(map[string]bool)
	return fields, jsonparser.ObjectEach(respRaw, func(key, value []byte, dataType jsonparser.ValueType, _ int) error {
		if dataType != jsonparser.String || len(value) != 0 {
			fields[string(key)] = true
		}
		return nil
	}, "data")
}

// updateTicker merges the fields a ticker push carries into tick
func updateTicker(tick *ticker.Price, t *WebsocketTicker, fields map[string]bool) {
	set := func(dst *float64, field string, v types.Number) {
		if fields[field] {
			*dst = v.Float64()
		}
	}
	set(&tick.Last, "lastPrice", t.LastPrice)
	set(&tick.High, "highPrice24h", t.HighPrice24Hour)
	set(&tick.Low, "lowPrice24h", t.LowPrice24Hour)
	// Linear and inverse tickers carry the best bid and ask as bid1 and ask1, option tickers without the 1, and spot
	// tickers carry neither
	set(&tick.Bid, "bid1Price", t.Bid1Price)
	set(&tick.BidSize, "bid1Size", t.Bid1Size)
	set(&tick.Ask, "ask1Price", t.Ask1Price)
	set(&tick.AskSize, "ask1Size", t.Ask1Size)
	set(&tick.Bid, "bidPrice", t.BidPrice)
	set(&tick.BidSize, "bidSize", t.BidSize)
	set(&tick.Ask, "askPrice", t.AskPrice)
	set(&tick.AskSize, "askSize", t.AskSize)
	// volume24h counts the quote currency on the inverse category and turnover24h the base, the reverse of everywhere
	// else
	baseField, quoteField, baseVolume, quoteVolume := "volume24h", "turnover24h", t.Volume24Hour, t.Turnover24Hour
	if tick.AssetType == asset.CoinMarginedFutures {
		baseField, quoteField, baseVolume, quoteVolume = quoteField, baseField, quoteVolume, baseVolume
	}
	set(&tick.BaseVolume, baseField, baseVolume)
	set(&tick.QuoteVolume, quoteField, quoteVolume)
	set(&tick.MarkPrice, "markPrice", t.MarkPrice)
	set(&tick.IndexPrice, "indexPrice", t.IndexPrice)
	// openInterest counts both sides of each linear and inverse contract, where GCT counts one; an option ticker's
	// openInterest, its only figure, counts one side already
	if tick.AssetType == asset.Options {
		set(&tick.OpenInterest, "openInterest", t.OpenInterest)
		set(&tick.OpenInterestValue, "openInterestValue", t.OpenInterestValue)
		return
	}
	set(&tick.OpenInterest, "singleOpenInterest", t.SingleOpenInterest)
	set(&tick.OpenInterestValue, "singleOpenInterestValue", t.SingleOpenInterestValue)
}

func (e *Exchange) wsProcessKline(ctx context.Context, assetType asset.Item, respRaw []byte) error {
	var resp WebsocketKlineUpdate
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
		var validationIssues string
		if !k.Confirm {
			validationIssues = kline.PartialCandle
		}
		if err := e.Websocket.DataHandler.Send(ctx, kline.Item{
			Pair:     cp,
			Asset:    assetType,
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:             k.Start.Time(),
				Open:             k.Open.Float64(),
				Close:            k.Close.Float64(),
				High:             k.High.Float64(),
				Low:              k.Low.Float64(),
				Volume:           volume,
				QuoteVolume:      quoteVolume,
				ValidationIssues: validationIssues,
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
