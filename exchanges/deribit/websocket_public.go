package deribit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// wsHandleOrderbook loads a book snapshot and applies a book change, checking each change follows the last one applied,
// or loads a grouped book, which every notification sends whole
func (e *Exchange) wsHandleOrderbook(ctx context.Context, channel string, respRaw []byte) error {
	// book.{instrument}.{group}.{depth}.{interval} has 5 parts, book.{instrument}.{interval} 3
	if strings.Count(channel, ".") == 4 {
		var data WebsocketGroupedOrderbook
		if err := decodeNotification(respRaw, &data); err != nil {
			return err
		}
		a, pair, err := getAssetPairByInstrument(data.InstrumentName)
		if err != nil {
			return err
		}
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             a,
			Bids:              orderBookLevels(data.Bids),
			Asks:              orderBookLevels(data.Asks),
			LastUpdated:       data.Timestamp.Time(),
			LastUpdateID:      int64(data.ChangeID), //nolint:gosec // Deribit's change IDs fit int64
			ValidateOrderbook: e.ValidateOrderbook,
		})
	}

	var data WebsocketOrderbook
	if err := decodeNotification(respRaw, &data); err != nil {
		return err
	}
	a, pair, err := getAssetPairByInstrument(data.InstrumentName)
	if err != nil {
		return err
	}
	bids, asks := orderbookChangeLevels(data.Bids), orderbookChangeLevels(data.Asks)
	switch data.Type {
	case bookSnapshot:
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             a,
			Bids:              bids,
			Asks:              asks,
			LastUpdated:       data.Timestamp.Time(),
			LastUpdateID:      int64(data.ChangeID), //nolint:gosec // Deribit's change IDs fit int64
			ValidateOrderbook: e.ValidateOrderbook,
		})
	case bookChange:
		lastChangeID, err := e.Websocket.Orderbook.LastUpdateID(pair, a)
		if err == nil && uint64(lastChangeID) != data.PreviousChangeID { //nolint:gosec // Deribit's change IDs fit int64
			err = fmt.Errorf("%w: %s change %d follows %d, not the last applied %d", errOrderbookOutOfSequence, data.InstrumentName, data.ChangeID, data.PreviousChangeID, lastChangeID)
		}
		if err == nil {
			err = e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
				Pair:       pair,
				Asset:      a,
				Bids:       bids,
				Asks:       asks,
				UpdateID:   int64(data.ChangeID), //nolint:gosec // Deribit's change IDs fit int64
				UpdateTime: data.Timestamp.Time(),
				// A change can leave both sides as they were, but still advances the change ID the next one follows
				AllowEmpty: true,
			})
		}
		// A book that misses a change, or fails one, stays invalid until resubscribing sends a new snapshot; a full data
		// handler is no reason to, as the book is applied before it is relayed
		if errors.Is(err, errOrderbookOutOfSequence) || errors.Is(err, orderbook.ErrOrderbookInvalid) || errors.Is(err, orderbook.ErrDepthNotFound) {
			if invalidateErr := e.Websocket.Orderbook.InvalidateOrderbook(pair, a); invalidateErr != nil && !errors.Is(invalidateErr, orderbook.ErrDepthNotFound) {
				err = common.AppendError(err, invalidateErr)
			}
			e.resubscribeChannel(ctx, channel)
		}
		return err
	default:
		return fmt.Errorf("%w: %s book notification type %q", errMalformedNotification, data.InstrumentName, data.Type)
	}
}

// orderbookChangeLevels converts book changes to levels; a deleted level carries a zero amount, which removes it
func orderbookChangeLevels(changes []WebsocketOrderbookChange) orderbook.Levels {
	levels := make(orderbook.Levels, len(changes))
	for i := range changes {
		levels[i] = orderbook.Level{Price: changes[i].Price, Amount: changes[i].Amount}
		if changes[i].Action == bookDelete {
			levels[i].Amount = 0
		}
	}
	return levels
}

// resubscribeChannel resubscribes the subscription of a qualified channel in the background, so Deribit sends a new
// snapshot without holding up the reader, which must go on matching responses while a resubscription waits for them
func (e *Exchange) resubscribeChannel(ctx context.Context, channel string) {
	sub := e.markResubscribing(channel)
	if sub == nil {
		return
	}
	generation := e.connectionGeneration.Load()
	go func() {
		if err := e.resubscribe(ctx, sub, generation); err != nil {
			log.Errorf(log.WebsocketMgr, "%s resubscribing to %s: %s", e.Name, channel, err)
		}
	}()
}

// markResubscribing marks the held subscription of a qualified channel as resubscribing and returns it, or nil when none
// is held or it is resubscribing already, as the changes still arriving for its invalid book would each resubscribe it
// again once the resubscription under way finished
func (e *Exchange) markResubscribing(channel string) *subscription.Subscription {
	for _, s := range e.Websocket.GetSubscriptions() {
		if s.QualifiedChannel == channel {
			if s.SetState(subscription.ResubscribingState) != nil {
				return nil
			}
			return s
		}
	}
	return nil
}

// resubscribe unsubscribes a subscription marked as resubscribing and subscribes it again on the connection of a
// generation. When its unsubscribe fails or is not sent the channel is as it was, so a subscription still held is
// marked subscribed again for the next change out of sequence to retry, and one unsubscribed meanwhile is left
// unsubscribed; when the subscribe fails after it, no further notification prompts another attempt, so the subscribe is
// retried
func (e *Exchange) resubscribe(ctx context.Context, sub *subscription.Subscription, generation uint64) error {
	subs := subscription.List{sub}
	var unsubscribed bool
	err := e.onConnection(generation, sub, func() error {
		if err := e.Websocket.UnsubscribeChannels(ctx, e.Websocket.Conn, subs); err != nil {
			return err
		}
		unsubscribed = true
		return e.Websocket.SubscribeToChannels(ctx, e.Websocket.Conn, subs)
	})
	switch {
	case err == nil:
		return nil
	case !unsubscribed:
		if sub.State() == subscription.ResubscribingState {
			err = common.AppendError(err, sub.SetState(subscription.SubscribedState))
		}
		if errors.Is(err, websocket.ErrNotConnected) || errors.Is(err, subscription.ErrNotFound) {
			return nil
		}
	case sub.State() == subscription.UnsubscribedState:
		if retryErr := e.subscribeAgain(ctx, sub, generation); retryErr != nil {
			return common.AppendError(err, retryErr)
		}
		return nil
	}
	return err
}

// subscribeAgain retries subscribing a subscription a resubscription unsubscribed, backing off between attempts, until
// the attempts run out, the context ends or the connection goes, as a new connection subscribes every channel itself
func (e *Exchange) subscribeAgain(ctx context.Context, sub *subscription.Subscription, generation uint64) error {
	backoff := request.DefaultBackoff()
	var err error
	for attempt := 1; attempt < maximumRequestAttempts; attempt++ {
		timer := time.NewTimer(backoff(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		err = e.onConnection(generation, sub, func() error {
			return e.Websocket.SubscribeToChannels(ctx, e.Websocket.Conn, subscription.List{sub})
		})
		if err == nil || errors.Is(err, websocket.ErrNotConnected) {
			return err
		}
	}
	return err
}

// nextConnectionGeneration starts the generation of a new connection once no subscription change is being made on the
// last one
func (e *Exchange) nextConnectionGeneration() {
	e.connectionMtx.Lock()
	defer e.connectionMtx.Unlock()
	e.connectionGeneration.Add(1)
}

// onConnection makes a subscription change on the connection of a generation, holding a new connection back until it is
// made. It is not made once that connection has gone, and the subscription is removed again when the connection goes
// meanwhile, as a new connection subscribes its own channels and would refuse them all as duplicates of it
func (e *Exchange) onConnection(generation uint64, sub *subscription.Subscription, change func() error) error {
	e.connectionMtx.Lock()
	defer e.connectionMtx.Unlock()
	if e.connectionGeneration.Load() != generation || !e.Websocket.IsConnected() {
		return fmt.Errorf("%w: %s", websocket.ErrNotConnected, sub.QualifiedChannel)
	}
	err := change()
	if !e.Websocket.IsConnected() && e.Websocket.GetSubscription(qualifiedChannelKey{sub}) != nil {
		err = common.AppendError(err, e.Websocket.RemoveSubscriptions(e.Websocket.Conn, sub))
	}
	return err
}

// wsHandleTicker stores and relays a ticker
func (e *Exchange) wsHandleTicker(ctx context.Context, respRaw []byte) error {
	var data TickerResponse
	if err := decodeNotification(respRaw, &data); err != nil {
		return err
	}
	a, cp, err := getAssetPairByInstrument(data.InstrumentName)
	if err != nil {
		return err
	}
	tick := data.tickerPrice(e.Name, cp, a)
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// wsHandleQuote stores a quote's best prices and sizes on the instrument's stored ticker, keeping the ticker's other
// fields, which a quote does not carry, and relays the ticker
func (e *Exchange) wsHandleQuote(ctx context.Context, respRaw []byte) error {
	var data WebsocketQuote
	if err := decodeNotification(respRaw, &data); err != nil {
		return err
	}
	a, cp, err := getAssetPairByInstrument(data.InstrumentName)
	if err != nil {
		return err
	}
	tick, err := ticker.GetTicker(e.Name, cp, a)
	if err != nil {
		tick = &ticker.Price{ExchangeName: e.Name, Pair: cp, AssetType: a}
	}
	tick.Bid, tick.BidSize = data.BestBidPrice, data.BestBidAmount
	tick.Ask, tick.AskSize = data.BestAskPrice, data.BestAskAmount
	tick.LastUpdated = data.Timestamp.Time()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// wsHandleTrades relays and saves public trades as the trade feed and trade saving settings allow
func (e *Exchange) wsHandleTrades(ctx context.Context, respRaw []byte) error {
	tradeFeed, saveTrades := e.IsTradeFeedEnabled(), e.IsSaveTradeDataEnabled()
	if !tradeFeed && !saveTrades {
		return nil
	}
	var trades []PublicTrade
	if err := decodeNotification(respRaw, &trades); err != nil {
		return err
	}
	converted := make([]trade.Data, 0, len(trades))
	var errs error
	for i := range trades {
		a, cp, err := getAssetPairByInstrument(trades[i].InstrumentName)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		side, err := order.StringToOrderSide(trades[i].Direction)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		converted = append(converted, trade.Data{
			TID:          trades[i].TradeID,
			Exchange:     e.Name,
			CurrencyPair: cp,
			AssetType:    a,
			Side:         side,
			Price:        trades[i].Price,
			Amount:       trades[i].Amount,
			Timestamp:    trades[i].Timestamp.Time(),
		})
	}
	if tradeFeed {
		for i := range converted {
			errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, converted[i]))
		}
	}
	if saveTrades && len(converted) > 0 {
		errs = common.AppendError(errs, trade.AddTradesToBuffer(converted...))
	}
	return errs
}

// wsHandleIncrementalTicker merges an incremental ticker change onto the instrument's state so far, or starts it afresh
// from a snapshot, then stores and relays the merged ticker
func (e *Exchange) wsHandleIncrementalTicker(ctx context.Context, channel string, respRaw []byte) error {
	instrumentName := strings.TrimPrefix(channel, ChannelIncrementalTicker+".")
	a, cp, err := getAssetPairByInstrument(instrumentName)
	if err != nil {
		return err
	}
	update := &WebsocketIncrementalTicker{}
	if err := decodeNotification(respRaw, update); err != nil {
		return err
	}

	// A snapshot starts the instrument's state afresh. A change carries only the fields that moved,
	// and the store overwrites a pair wholesale, so it is decoded onto the state so far: each field
	// it leaves out keeps its value, and each it sends as null is cleared
	e.incrementalTickersMtx.Lock()
	state, ok := e.incrementalTickers[instrumentName]
	switch {
	case update.Type != bookChange:
		state = update
	case !ok:
		e.incrementalTickersMtx.Unlock()
		return fmt.Errorf("%w: %s", errNoTickerSnapshot, instrumentName)
	default:
		if err := decodeNotification(respRaw, state); err != nil {
			e.incrementalTickersMtx.Unlock()
			return err
		}
	}
	if e.incrementalTickers == nil {
		e.incrementalTickers = make(map[string]*WebsocketIncrementalTicker)
	}
	e.incrementalTickers[instrumentName] = state
	tick := state.tickerPrice(e.Name, cp, a)
	e.incrementalTickersMtx.Unlock()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// tickerPrice maps an instrument's merged state as a full ticker is mapped
func (t *WebsocketIncrementalTicker) tickerPrice(exchangeName string, cp currency.Pair, a asset.Item) *ticker.Price {
	value := func(v *float64) float64 {
		if v == nil {
			return 0
		}
		return *v
	}
	return &ticker.Price{
		ExchangeName: exchangeName,
		Pair:         cp,
		AssetType:    a,
		LastUpdated:  t.Timestamp.Time(),
		Bid:          value(t.BestBidPrice),
		BidSize:      value(t.BestBidAmount),
		Ask:          value(t.BestAskPrice),
		AskSize:      value(t.BestAskAmount),
		Last:         value(t.LastPrice),
		Close:        value(t.LastPrice),
		High:         value(t.Stats.High),
		Low:          value(t.Stats.Low),
		BaseVolume:   value(t.Stats.Volume),
		QuoteVolume:  quoteVolume(value(t.Stats.VolumeUSD), value(t.Stats.VolumeNotional), a),
		MarkPrice:    value(t.MarkPrice),
		IndexPrice:   value(t.IndexPrice),
		OpenInterest: value(t.OpenInterest),
	}
}

// wsHandleCandle relays a chart.trades candle with the interval its channel's resolution names
func (e *Exchange) wsHandleCandle(ctx context.Context, channel string, respRaw []byte) error {
	// chart.trades.{instrument}.{resolution}
	rest, ok := strings.CutPrefix(channel, ChannelChartTrades+".")
	separator := strings.LastIndexByte(rest, '.')
	if !ok || separator < 0 {
		return fmt.Errorf("%w: %s", errMalformedNotification, channel)
	}
	interval, err := resolutionToInterval(rest[separator+1:])
	if err != nil {
		return err
	}
	a, cp, err := getAssetPairByInstrument(rest[:separator])
	if err != nil {
		return err
	}
	var candle WebsocketCandle
	if err := decodeNotification(respRaw, &candle); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Pair:     cp,
		Asset:    a,
		Exchange: e.Name,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:        candle.Tick.Time(),
			Open:        candle.Open,
			High:        candle.High,
			Low:         candle.Low,
			Close:       candle.Close,
			Volume:      candle.Volume,
			QuoteVolume: candle.Cost,
		}},
	})
}

// resolutionToInterval returns the interval a chart resolution names
func resolutionToInterval(resolution string) (kline.Interval, error) {
	if resolution == "1D" {
		return kline.OneDay, nil
	}
	minutes, err := strconv.ParseUint(resolution, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: resolution %q", kline.ErrInvalidInterval, resolution)
	}
	interval := kline.Interval(time.Duration(minutes) * time.Minute) //nolint:gosec // Resolutions are a few minutes
	if _, err := chartResolution(interval); err != nil {
		return 0, err
	}
	return interval, nil
}
