package coinbase

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Gateway book notification types and level actions
const (
	derivativesBookSnapshot = "snapshot"
	derivativesBookChange   = "change"
	derivativesBookDelete   = "delete"
)

var (
	errDerivativesNoTickerSnapshot       = errors.New("gateway incremental ticker change received before its snapshot")
	errDerivativesOrderbookOutOfSequence = errors.New("gateway order book change out of sequence")
	errDerivativesMalformedNotification  = errors.New("malformed gateway notification")
)

// wsHandleDerivativesNotification passes a gateway subscription notification to its channel's handler
func (e *Exchange) wsHandleDerivativesNotification(ctx context.Context, conn websocket.Connection, channel string, respRaw []byte) error {
	switch derivativesNotificationChannel(channel) {
	case ChannelDerivativesBook:
		return e.wsHandleDerivativesOrderbook(ctx, conn, channel, respRaw)
	case ChannelDerivativesIncrementalTicker:
		return e.wsHandleDerivativesIncrementalTicker(ctx, channel, respRaw)
	case ChannelDerivativesChartTrades:
		return e.wsHandleDerivativesCandle(ctx, channel, respRaw)
	case ChannelDerivativesTicker:
		return e.wsHandleDerivativesTicker(ctx, respRaw)
	case ChannelDerivativesTrades:
		return e.wsHandleDerivativesTrades(ctx, respRaw)
	case ChannelDerivativesUserOrders, ChannelDerivativesUserIsolatedOrders:
		return e.wsHandleDerivativesOrders(ctx, respRaw)
	case ChannelDerivativesUserTrades, ChannelDerivativesUserComboTrades, ChannelDerivativesUserIsolatedTrades:
		return e.wsHandleDerivativesUserTrades(respRaw)
	case ChannelDerivativesPerpetual:
		return wsDerivativesRelay[DerivativesWebsocketPerpetualInterest](ctx, e, respRaw)
	case ChannelDerivativesQuote:
		return wsDerivativesRelay[DerivativesWebsocketQuote](ctx, e, respRaw)
	case ChannelDerivativesPriceIndex:
		return wsDerivativesRelay[DerivativesWebsocketPriceIndex](ctx, e, respRaw)
	case ChannelDerivativesPriceRanking:
		return wsDerivativesRelay[[]DerivativesWebsocketPriceRanking](ctx, e, respRaw)
	case ChannelDerivativesPriceStatistics:
		return wsDerivativesRelay[DerivativesWebsocketPriceStatistics](ctx, e, respRaw)
	case ChannelDerivativesVolatilityIndex:
		return wsDerivativesRelay[DerivativesWebsocketVolatilityIndex](ctx, e, respRaw)
	case ChannelDerivativesMarkPriceOptions:
		return wsDerivativesRelay[[]DerivativesWebsocketMarkPriceOption](ctx, e, respRaw)
	case ChannelDerivativesEstimatedExpirationPrice:
		return wsDerivativesRelay[DerivativesWebsocketEstimatedExpirationPrice](ctx, e, respRaw)
	case ChannelDerivativesInstrumentState:
		return wsDerivativesRelay[DerivativesWebsocketInstrumentState](ctx, e, respRaw)
	case ChannelDerivativesInstrumentCreation:
		return wsDerivativesRelay[DerivativesWebsocketInstrumentCreation](ctx, e, respRaw)
	case ChannelDerivativesPlatformState:
		return wsDerivativesRelay[DerivativesWebsocketPlatformState](ctx, e, respRaw)
	case ChannelDerivativesPublicMethodsState:
		return wsDerivativesRelay[DerivativesWebsocketPublicMethodsState](ctx, e, respRaw)
	case ChannelDerivativesAnnouncements:
		return wsDerivativesRelay[DerivativesWebsocketAnnouncement](ctx, e, respRaw)
	case ChannelDerivativesUserPortfolio:
		return wsDerivativesRelay[DerivativesAccountSummary](ctx, e, respRaw)
	case ChannelDerivativesUserIsolatedPortfolio:
		return wsDerivativesRelay[DerivativesWebsocketIsolatedPortfolio](ctx, e, respRaw)
	case ChannelDerivativesUserChanges, ChannelDerivativesUserIsolatedChanges:
		return wsDerivativesRelay[DerivativesWebsocketUserChanges](ctx, e, respRaw)
	case ChannelDerivativesUserAccessLog:
		return wsDerivativesRelay[DerivativesAccessLogEntry](ctx, e, respRaw)
	case ChannelDerivativesUserLock:
		return wsDerivativesRelay[DerivativesWebsocketUserLock](ctx, e, respRaw)
	case ChannelDerivativesUserPositionLock:
		return wsDerivativesRelay[DerivativesWebsocketPositionLock](ctx, e, respRaw)
	case ChannelDerivativesUserIsolatedLiquidation:
		return wsDerivativesRelay[DerivativesWebsocketUserLiquidation](ctx, e, respRaw)
	case ChannelDerivativesUserTrailingOrdersTriggerPrice:
		return wsDerivativesRelay[DerivativesWebsocketTrailingTriggerPrices](ctx, e, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// decodeDerivativesNotification decodes a subscription notification's data into the value data points to
func decodeDerivativesNotification(respRaw []byte, data any) error {
	notification := derivativesNotification{Params: derivativesNotificationParams{Data: data}}
	if err := json.Unmarshal(respRaw, &notification); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesMalformedNotification, err)
	}
	return nil
}

// wsDerivativesRelay decodes a notification's data into its exported type and relays it
func wsDerivativesRelay[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	data := new(T)
	if err := decodeDerivativesNotification(respRaw, data); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

// wsHandleDerivativesOrderbook loads a book snapshot and applies a book change, checking each change follows the last one
// applied, or loads a grouped book, which every notification sends whole
func (e *Exchange) wsHandleDerivativesOrderbook(ctx context.Context, conn websocket.Connection, channel string, respRaw []byte) error {
	// book.{instrument}.{group}.{depth}.{interval} has 5 parts, book.{instrument}.{interval} 3
	if strings.Count(channel, ".") == 4 {
		var data DerivativesWebsocketGroupedOrderbook
		if err := decodeDerivativesNotification(respRaw, &data); err != nil {
			return err
		}
		a, pair, err := derivativesInstrumentPair(data.InstrumentName)
		if err != nil {
			return err
		}
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             a,
			Bids:              derivativesBookLevels(data.Bids),
			Asks:              derivativesBookLevels(data.Asks),
			LastUpdated:       data.Timestamp.Time(),
			LastUpdateID:      int64(data.ChangeID), //nolint:gosec // The gateway's change IDs fit int64
			ValidateOrderbook: e.ValidateOrderbook,
		})
	}

	var data DerivativesWebsocketOrderbook
	if err := decodeDerivativesNotification(respRaw, &data); err != nil {
		return err
	}
	a, pair, err := derivativesInstrumentPair(data.InstrumentName)
	if err != nil {
		return err
	}
	bids, asks := derivativesBookChangeLevels(data.Bids), derivativesBookChangeLevels(data.Asks)
	switch data.Type {
	case derivativesBookSnapshot:
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             a,
			Bids:              bids,
			Asks:              asks,
			LastUpdated:       data.Timestamp.Time(),
			LastUpdateID:      int64(data.ChangeID), //nolint:gosec // The gateway's change IDs fit int64
			ValidateOrderbook: e.ValidateOrderbook,
		})
	case derivativesBookChange:
		lastChangeID, err := e.Websocket.Orderbook.LastUpdateID(pair, a)
		if err == nil && uint64(lastChangeID) != data.PreviousChangeID { //nolint:gosec // The gateway's change IDs fit int64
			err = fmt.Errorf("%w: %s change %d follows %d, not the last applied %d", errDerivativesOrderbookOutOfSequence, data.InstrumentName, data.ChangeID, data.PreviousChangeID, lastChangeID)
		}
		if err == nil {
			err = e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
				Pair:       pair,
				Asset:      a,
				Bids:       bids,
				Asks:       asks,
				UpdateID:   int64(data.ChangeID), //nolint:gosec // The gateway's change IDs fit int64
				UpdateTime: data.Timestamp.Time(),
				// A change can leave both sides as they were, but still advances the change ID the next one follows
				AllowEmpty: true,
			})
		}
		// A book that misses a change, or fails one, stays invalid until resubscribing sends a new snapshot; a full data
		// handler is no reason to, as the book is applied before it is relayed
		if errors.Is(err, errDerivativesOrderbookOutOfSequence) || errors.Is(err, orderbook.ErrOrderbookInvalid) || errors.Is(err, orderbook.ErrDepthNotFound) {
			if invalidateErr := e.Websocket.Orderbook.InvalidateOrderbook(pair, a); invalidateErr != nil && !errors.Is(invalidateErr, orderbook.ErrDepthNotFound) {
				err = common.AppendError(err, invalidateErr)
			}
			e.wsDerivativesResubscribe(ctx, conn, channel)
		}
		return err
	default:
		return fmt.Errorf("%w: %s book notification type %q", errDerivativesMalformedNotification, data.InstrumentName, data.Type)
	}
}

// derivativesBookChangeLevels converts book changes to levels; a deleted level carries a zero amount, which removes it
func derivativesBookChangeLevels(changes []DerivativesWebsocketOrderbookChange) orderbook.Levels {
	levels := make(orderbook.Levels, len(changes))
	for i := range changes {
		levels[i] = orderbook.Level{Price: changes[i].Price, Amount: changes[i].Amount}
		if changes[i].Action == derivativesBookDelete {
			levels[i].Amount = 0
		}
	}
	return levels
}

// derivativesBookLevels converts book levels, which the gateway sends best first, leaving out a combo's levels at a zero
// price, which its spread can be quoted at but a book cannot hold
func derivativesBookLevels(levels []DerivativesOrderBookLevel) orderbook.Levels {
	converted := make(orderbook.Levels, 0, len(levels))
	for i := range levels {
		if levels[i].Price != 0 {
			converted = append(converted, orderbook.Level{Price: levels[i].Price, Amount: levels[i].Amount})
		}
	}
	return converted
}

// wsDerivativesResubscribe resubscribes the subscription of a qualified channel on its connection in the background, so
// the gateway sends a new snapshot without holding up the reader
func (e *Exchange) wsDerivativesResubscribe(ctx context.Context, conn websocket.Connection, channel string) {
	for _, s := range conn.Subscriptions().List() {
		if s.QualifiedChannel != channel {
			continue
		}
		go func() {
			if err := e.Websocket.ResubscribeFromConnection(ctx, conn, subscription.List{s}); err != nil && !errors.Is(err, subscription.ErrInStateAlready) {
				_ = e.Websocket.DataHandler.Send(ctx, fmt.Errorf("%s websocket could not resubscribe to %s: %w", e.Name, channel, err))
			}
		}()
		return
	}
}

// wsHandleDerivativesTicker stores and relays a ticker
func (e *Exchange) wsHandleDerivativesTicker(ctx context.Context, respRaw []byte) error {
	var data DerivativesTicker
	if err := decodeDerivativesNotification(respRaw, &data); err != nil {
		return err
	}
	a, pair, err := derivativesInstrumentPair(data.InstrumentName)
	if err != nil {
		return err
	}
	tick := data.tickerPrice(e.Name, pair, a)
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// tickerPrice converts a ticker; min_price and max_price bound the price an order may carry rather than the day's range
func (t *DerivativesTicker) tickerPrice(exchangeName string, pair currency.Pair, a asset.Item) *ticker.Price {
	return &ticker.Price{
		ExchangeName:        exchangeName,
		Pair:                pair,
		AssetType:           a,
		LastUpdated:         t.Timestamp.Time(),
		Bid:                 t.BestBidPrice,
		BidSize:             t.BestBidAmount,
		Ask:                 t.BestAskPrice,
		AskSize:             t.BestAskAmount,
		Last:                t.LastPrice,
		Close:               t.LastPrice,
		High:                t.Stats.High,
		Low:                 t.Stats.Low,
		BaseVolume:          t.Stats.Volume,
		QuoteVolume:         derivativesQuoteVolume(t.Stats.VolumeUSD, t.Stats.VolumeNotional),
		PercentChange24Hour: t.Stats.PriceChange,
		MarkPrice:           t.MarkPrice,
		IndexPrice:          t.IndexPrice,
		OpenInterest:        t.OpenInterest,
	}
}

// derivativesQuoteVolume returns a perpetual's 24 hour volume in its quote currency: the gateway sends volume_notional
// in it, and volume_usd, the same for the perpetuals, where it leaves the notional out
func derivativesQuoteVolume(volumeUSD, volumeNotional float64) float64 {
	if volumeNotional != 0 {
		return volumeNotional
	}
	return volumeUSD
}

// wsHandleDerivativesTrades relays and saves public trades as the trade feed and trade saving settings allow; a
// trade's direction is its taker's
func (e *Exchange) wsHandleDerivativesTrades(ctx context.Context, respRaw []byte) error {
	tradeFeed, saveTrades := e.IsTradeFeedEnabled(), e.IsSaveTradeDataEnabled()
	if !tradeFeed && !saveTrades {
		return nil
	}
	var trades []DerivativesPublicTrade
	if err := decodeDerivativesNotification(respRaw, &trades); err != nil {
		return err
	}
	converted := make([]trade.Data, 0, len(trades))
	var errs error
	for i := range trades {
		a, pair, err := derivativesInstrumentPair(trades[i].InstrumentName)
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
			CurrencyPair: pair,
			AssetType:    a,
			Side:         side,
			Price:        trades[i].Price,
			Amount:       trades[i].Amount,
			Timestamp:    trades[i].Timestamp.Time(),
		})
	}
	if len(converted) == 0 {
		return errs
	}
	if tradeFeed {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, converted))
	}
	if saveTrades {
		errs = common.AppendError(errs, trade.AddTradesToBuffer(converted...))
	}
	return errs
}

// wsHandleDerivativesIncrementalTicker merges an incremental ticker change onto the instrument's state so far, or
// starts it afresh from a snapshot, then stores and relays the merged ticker
func (e *Exchange) wsHandleDerivativesIncrementalTicker(ctx context.Context, channel string, respRaw []byte) error {
	instrumentName := strings.TrimPrefix(channel, ChannelDerivativesIncrementalTicker+".")
	a, pair, err := derivativesInstrumentPair(instrumentName)
	if err != nil {
		return err
	}
	update := new(DerivativesWebsocketIncrementalTicker)
	if err := decodeDerivativesNotification(respRaw, update); err != nil {
		return err
	}

	// A snapshot starts the instrument's state afresh. A change carries only the fields that moved, and the store
	// overwrites a pair wholesale, so it is decoded onto the state so far: each field it leaves out keeps its value, and
	// each it sends as null is cleared
	e.derivativesIncrementalTickersMu.Lock()
	state, ok := e.derivativesIncrementalTickers[instrumentName]
	switch {
	case update.Type != derivativesBookChange:
		state = update
	case !ok:
		e.derivativesIncrementalTickersMu.Unlock()
		return fmt.Errorf("%w: %s", errDerivativesNoTickerSnapshot, instrumentName)
	default:
		if err := decodeDerivativesNotification(respRaw, state); err != nil {
			e.derivativesIncrementalTickersMu.Unlock()
			return err
		}
	}
	if e.derivativesIncrementalTickers == nil {
		e.derivativesIncrementalTickers = make(map[string]*DerivativesWebsocketIncrementalTicker)
	}
	e.derivativesIncrementalTickers[instrumentName] = state
	tick := state.tickerPrice(e.Name, pair, a)
	e.derivativesIncrementalTickersMu.Unlock()
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// tickerPrice maps an instrument's merged state as a full ticker is mapped
func (d *DerivativesWebsocketIncrementalTicker) tickerPrice(exchangeName string, pair currency.Pair, a asset.Item) *ticker.Price {
	value := func(v *float64) float64 {
		if v == nil {
			return 0
		}
		return *v
	}
	return &ticker.Price{
		ExchangeName:        exchangeName,
		Pair:                pair,
		AssetType:           a,
		LastUpdated:         d.Timestamp.Time(),
		Bid:                 value(d.BestBidPrice),
		BidSize:             value(d.BestBidAmount),
		Ask:                 value(d.BestAskPrice),
		AskSize:             value(d.BestAskAmount),
		Last:                value(d.LastPrice),
		Close:               value(d.LastPrice),
		High:                value(d.Stats.High),
		Low:                 value(d.Stats.Low),
		BaseVolume:          value(d.Stats.Volume),
		QuoteVolume:         derivativesQuoteVolume(value(d.Stats.VolumeUSD), value(d.Stats.VolumeNotional)),
		PercentChange24Hour: value(d.Stats.PriceChange),
		MarkPrice:           value(d.MarkPrice),
		IndexPrice:          value(d.IndexPrice),
		OpenInterest:        value(d.OpenInterest),
	}
}

// wsHandleDerivativesCandle relays a chart.trades candle with the interval its channel's resolution names
func (e *Exchange) wsHandleDerivativesCandle(ctx context.Context, channel string, respRaw []byte) error {
	// chart.trades.{instrument}.{resolution}
	rest, ok := strings.CutPrefix(channel, ChannelDerivativesChartTrades+".")
	separator := strings.LastIndexByte(rest, '.')
	if !ok || separator < 0 {
		return fmt.Errorf("%w: %s", errDerivativesMalformedNotification, channel)
	}
	interval, err := derivativesResolutionInterval(rest[separator+1:])
	if err != nil {
		return err
	}
	a, pair, err := derivativesInstrumentPair(rest[:separator])
	if err != nil {
		return err
	}
	var candle DerivativesWebsocketCandle
	if err := decodeDerivativesNotification(respRaw, &candle); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Pair:     pair,
		Asset:    a,
		Exchange: e.Name,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   candle.Tick.Time(),
			Open:   candle.Open,
			High:   candle.High,
			Low:    candle.Low,
			Close:  candle.Close,
			Volume: candle.Volume,
		}},
	})
}
