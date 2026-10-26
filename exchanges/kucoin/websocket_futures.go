package kucoin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var (
	errMalformedOrderbookChange = errors.New("malformed orderbook change")
	errUnknownMarginMode        = errors.New("unknown margin mode")
)

// processFuturesTickerV2 merges a Ticker V2 push onto the stored ticker. Ticker V2 carries only the best bid and ask,
// and the ticker's other fields come from its companion topics
func (e *Exchange) processFuturesTickerV2(ctx context.Context, data []byte) error {
	var resp WebsocketFuturesTickerV2
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.Bid = resp.BestBidPrice.Float64()
		t.BidSize = float64(resp.BestBidSize)
		t.Ask = resp.BestAskPrice.Float64()
		t.AskSize = float64(resp.BestAskSize)
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// processFuturesTickerV1 merges the last fill of a Ticker V1 push onto the stored ticker. Its best bid and ask are
// those at the fill and often older than Ticker V2's, so they are left to Ticker V2
func (e *Exchange) processFuturesTickerV1(ctx context.Context, data []byte) error {
	var resp WebsocketFuturesTickerV1
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.Last = resp.Price.Float64()
		t.LastSize = float64(resp.Size)
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// processFuturesSnapshot merges the 24 hour statistics of a Symbol Snapshot push onto the stored ticker. Its last price
// is up to 5 seconds old, so the last price is left to Ticker V1
func (e *Exchange) processFuturesSnapshot(ctx context.Context, data []byte) error {
	var resp WebsocketFuturesSnapshot
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	baseVolume, quoteVolume := resp.Volume, resp.Turnover
	if isInverseContract(pair) {
		// An inverse contract is worth one unit of its quote currency, so its volume is counted in the quote currency
		baseVolume, quoteVolume = quoteVolume, baseVolume
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.High = resp.HighPrice
		t.Low = resp.LowPrice
		t.Open24Hour = resp.Price24HoursBefore
		t.BaseVolume = baseVolume
		t.QuoteVolume = quoteVolume
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// isInverseContract returns whether a futures pair is an inverse contract; KuCoin's linear contracts are quoted in
// USDTM or USDCM and its inverse ones in USDM or by delivery month
func isInverseContract(pair currency.Pair) bool {
	return !pair.Quote.Equal(currency.USDTM) && !pair.Quote.Equal(currency.USDCM)
}

// processFuturesOrderbookSnapshot loads a futures Orderbook - Level 5 or Level 50 push
func (e *Exchange) processFuturesOrderbookSnapshot(ctx context.Context, conn websocket.Connection, data []byte, symbol string) error {
	var resp WebsocketFuturesOrderbookDepth
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	sequence := orderbookUpdateID(resp.Sequence)
	return e.feedOrderbook(conn, pair, asset.Futures, sequence, orderbookSnapshot, func() error {
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:     e.Name,
			LastUpdateID: sequence,
			LastUpdated:  resp.Timestamp.Time(),
			LastPushed:   resp.PushTimestamp.Time(),
			Asset:        asset.Futures,
			Bids:         mergeRoundedOrderbookLevels(resp.Bids.Levels()),
			Asks:         mergeRoundedOrderbookLevels(resp.Asks.Levels()),
			Pair:         pair,
		})
	})
}

// processFuturesOrderbookIncrement applies a futures Orderbook - Increment push
func (e *Exchange) processFuturesOrderbookIncrement(ctx context.Context, conn websocket.Connection, data []byte, symbol string) error {
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	var resp WebsocketFuturesOrderbookIncrement
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	parts := strings.Split(resp.Change, ",")
	if len(parts) != 3 {
		return fmt.Errorf("%w: %q", errMalformedOrderbookChange, resp.Change)
	}
	price, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("%w: price %q: %w", errMalformedOrderbookChange, resp.Change, err)
	}
	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return fmt.Errorf("%w: size %q: %w", errMalformedOrderbookChange, resp.Change, err)
	}
	sequence := orderbookUpdateID(resp.Sequence)
	var bids, asks []orderbook.Level
	switch parts[1] {
	case "buy":
		bids = []orderbook.Level{{Price: price, Amount: amount, ID: sequence}}
	case "sell":
		asks = []orderbook.Level{{Price: price, Amount: amount, ID: sequence}}
	default:
		return fmt.Errorf("%w: side %q", errMalformedOrderbookChange, resp.Change)
	}
	return e.feedOrderbook(conn, pair, asset.Futures, sequence, orderbookIncrement, func() error {
		return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, sequence, &orderbook.Update{
			UpdateTime: resp.Timestamp.Time(),
			LastPushed: resp.Timestamp.Time(),
			UpdateID:   sequence,
			Pair:       pair,
			Asset:      asset.Futures,
			Asks:       asks,
			Bids:       bids,
		})
	})
}

// processFuturesKline relays a futures Klines push as a candle. Live candles carry the turnover where the docs place
// the volume, which they call incorrect, and the traded contracts where they place the amount; the volume relayed is
// the contracts, as the REST klines report it
func (e *Exchange) processFuturesKline(ctx context.Context, data []byte, symbolAndInterval string) error {
	symbol, intervalString, ok := strings.Cut(symbolAndInterval, currency.UnderscoreDelimiter)
	if !ok {
		return fmt.Errorf("%w: %q", errMalformedCandleTopic, symbolAndInterval)
	}
	interval, err := IntervalFromString(intervalString)
	if err != nil {
		return err
	}
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	var resp WebsocketCandlestick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &kline.Item{
		Asset:    asset.Futures,
		Exchange: e.Name,
		Pair:     pair,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   resp.Candles.StartTime.Time(),
			Open:   resp.Candles.OpenPrice.Float64(),
			Close:  resp.Candles.ClosePrice.Float64(),
			High:   resp.Candles.HighPrice.Float64(),
			Low:    resp.Candles.LowPrice.Float64(),
			Volume: resp.Candles.TransactionAmount.Float64(),
		}},
	})
}

// processFuturesExecution saves a futures Trade push
func (e *Exchange) processFuturesExecution(data []byte) error {
	var resp WebsocketFuturesExecution
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		CurrencyPair: pair,
		Timestamp:    resp.Timestamp.Time(),
		Price:        resp.Price.Float64(),
		Amount:       float64(resp.Size),
		Side:         side,
		Exchange:     e.Name,
		TID:          resp.TradeID,
		AssetType:    asset.Futures,
	})
}

// processFuturesInstrument relays an Instrument push, naming the symbol from the topic as the push does not carry it
func (e *Exchange) processFuturesInstrument(ctx context.Context, msg *WebsocketMessage, symbol string) error {
	var resp any
	switch msg.Subject {
	case "mark.index.price":
		resp = &WebsocketFuturesMarkIndexPrice{Symbol: symbol}
	case "funding.rate":
		resp = &WebsocketFuturesFundingRate{Symbol: symbol}
	default:
		return unhandledSubject(msg)
	}
	if err := json.Unmarshal(msg.Data, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesAnnouncement relays a Funding Fee Settlement push
func (e *Exchange) processFuturesAnnouncement(ctx context.Context, msg *WebsocketMessage) error {
	switch msg.Subject {
	case "funding.begin":
		return processPayload[WebsocketFuturesFundingBegin](ctx, e, msg.Data)
	case "funding.end":
		return processPayload[WebsocketFuturesFundingEnd](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// processFuturesOrderChange relays a futures Orders push as an order update
func (e *Exchange) processFuturesOrderChange(ctx context.Context, data []byte) error {
	var resp WebsocketFuturesOrderChange
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	// The open push of an order resting on the book carries no order type, so the order keeps the type it was given
	oType := order.UnknownType
	if resp.OrderType != "" {
		if oType, err = order.StringToOrderType(resp.OrderType); err != nil {
			return err
		}
	}
	if stopType, ok := e.futuresStopOrderTypes.Load(resp.OrderID); ok {
		if t, ok := stopType.(order.Type); ok {
			oType = t
		}
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	status, err := orderChangeStatus(resp.Status, resp.Type, resp.FilledSize.Float64(), resp.RemainSize.Float64())
	if err != nil {
		return err
	}
	marginType, err := futuresMarginType(resp.MarginMode)
	if err != nil {
		return err
	}
	detail := &order.Detail{
		Price:           resp.Price.Float64(),
		Amount:          resp.Size.Float64(),
		ExecutedAmount:  resp.FilledSize.Float64(),
		RemainingAmount: resp.RemainSize.Float64(),
		Exchange:        e.Name,
		OrderID:         resp.OrderID,
		ClientOrderID:   resp.ClientOrderID,
		Type:            oType,
		Side:            side,
		Status:          status,
		AssetType:       asset.Futures,
		MarginType:      marginType,
		Date:            resp.OrderTime.Time(),
		LastUpdated:     resp.Timestamp.Time(),
		Pair:            pair,
	}
	if resp.Type == "match" {
		detail.Trades = []order.TradeHistory{{
			Price:     resp.MatchPrice.Float64(),
			Amount:    resp.MatchSize.Float64(),
			Exchange:  e.Name,
			TID:       resp.TradeID,
			Type:      oType,
			Side:      side,
			Timestamp: resp.Timestamp.Time(),
			IsMaker:   resp.Liquidity == "maker",
		}}
	}
	if err := e.Websocket.DataHandler.Send(ctx, detail); err != nil {
		return err
	}
	if resp.Status == "done" {
		// Noted once relayed, so that the stop pushes of an order whose final push the relay could not take keep their
		// status
		e.endFuturesOrder(resp.OrderID)
	}
	return nil
}

// futuresMarginType returns the margin type of a futures margin mode
func futuresMarginType(marginMode string) (margin.Type, error) {
	switch marginMode {
	case "ISOLATED":
		return margin.Isolated, nil
	case "CROSS":
		return margin.Multi, nil
	case "":
		return margin.Unset, nil
	}
	return margin.Unknown, fmt.Errorf("%w: %q", errUnknownMarginMode, marginMode)
}

// processFuturesWallet saves the balance of a futures Balance push and relays it. The order margin and withdrawal hold
// subjects, deprecated once an account first switches margin mode, are relayed as they come
func (e *Exchange) processFuturesWallet(ctx context.Context, msg *WebsocketMessage) error {
	switch msg.Subject {
	case "walletBalance.change":
		var resp WebsocketFuturesWalletBalance
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return err
		}
		// As REST saves it: the total is the equity, and the hold the position and order margins with the frozen balance
		hold := resp.CrossPositionMargin.Float64() + resp.CrossOrderMargin.Float64() + resp.IsolatedPositionMargin.Float64() + resp.IsolatedOrderMargin.Float64() + resp.HoldBalance.Float64()
		return e.saveFuturesBalance(ctx, resp.Currency, resp.Equity.Float64(), hold, resp.AvailableBalance.Float64(), resp.Timestamp.Time())
	case "availableBalance.change":
		var resp WebsocketFuturesAvailableBalance
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return err
		}
		// This subject's hold balance is the position margin, order margin and frozen funds together
		return e.saveFuturesBalance(ctx, resp.Currency, resp.AvailableBalance+resp.HoldBalance, resp.HoldBalance, resp.AvailableBalance, resp.Timestamp.Time())
	case "orderMargin.change":
		return processPayload[WebsocketFuturesOrderMargin](ctx, e, msg.Data)
	case "withdrawHold.change":
		return processPayload[WebsocketFuturesWithdrawHold](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// saveFuturesBalance saves and relays a futures balance. The save is one step against REST's futures listings, so a
// listing saved alongside neither undoes it nor is refused for it
func (e *Exchange) saveFuturesBalance(ctx context.Context, code currency.Code, total, hold, available float64, updatedAt time.Time) error {
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	subAccts[0].Balances.Set(code, accounts.Balance{
		Total:     total,
		Hold:      hold,
		Free:      available,
		UpdatedAt: updatedAt,
	})
	e.futuresBalanceMu.Lock()
	err := e.Accounts.Save(ctx, subAccts, false)
	e.futuresBalanceMu.Unlock()
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// processFuturesPosition relays a Positions push
func (e *Exchange) processFuturesPosition(ctx context.Context, msg *WebsocketMessage) error {
	switch msg.Subject {
	case "position.change":
		return processPayload[WebsocketFuturesPosition](ctx, e, msg.Data)
	case "position.settlement":
		return processPayload[WebsocketFuturesPositionSettlement](ctx, e, msg.Data)
	case "position.adjustRiskLimit":
		return processPayload[WebsocketFuturesRiskLimitAdjustment](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// rememberFuturesStopOrderType records the type of a futures stop order a push or a REST read reported, unless a push or
// REST read has reported the order ended within futuresFinalPushWait, and reports whether it did. An end noted longer ago
// is forgotten, so the type is not forgotten with it
func (e *Exchange) rememberFuturesStopOrderType(orderID string, oType order.Type) bool {
	e.futuresOrderEndsMu.Lock()
	defer e.futuresOrderEndsMu.Unlock()
	return e.rememberFuturesStopOrderTypeLocked(orderID, oType)
}

// rememberFuturesStopOrderTypeLocked is rememberFuturesStopOrderType with futuresOrderEndsMu held
func (e *Exchange) rememberFuturesStopOrderTypeLocked(orderID string, oType order.Type) bool {
	if e.futuresOrderEndedLocked(orderID) {
		return false
	}
	delete(e.futuresOrderEnds, orderID)
	e.futuresStopOrderTypes.Store(orderID, oType)
	return true
}

// rememberFuturesStopOrderRead records the type of a futures stop order a REST read reported, as
// rememberFuturesStopOrderType does, and notes the end of one the read reported ended, whose final Orders push may still
// be on its way
func (e *Exchange) rememberFuturesStopOrderRead(orderID string, oType order.Type, ended bool) {
	e.futuresOrderEndsMu.Lock()
	defer e.futuresOrderEndsMu.Unlock()
	if e.rememberFuturesStopOrderTypeLocked(orderID, oType) && ended {
		e.noteFuturesOrderEndLocked(orderID)
	}
}

// endFuturesOrder forgets the type of a futures stop order a push reported ended, noting that it ended
func (e *Exchange) endFuturesOrder(orderID string) {
	e.futuresOrderEndsMu.Lock()
	defer e.futuresOrderEndsMu.Unlock()
	e.futuresStopOrderTypes.Delete(orderID)
	e.noteFuturesOrderEndLocked(orderID)
}

// noteFuturesOrderEndLocked notes that a push or a REST read reported a futures order ended, a push's end replacing an
// earlier one. Noting an end forgets, at most once a second, the ends noted longer ago than futuresFinalPushWait, with
// the types REST reads recorded for final Orders pushes that did not come. futuresOrderEndsMu must be held
func (e *Exchange) noteFuturesOrderEndLocked(orderID string) {
	now := time.Now()
	if now.Sub(e.futuresOrderEndsPrunedAt) >= time.Second {
		for id, endedAt := range e.futuresOrderEnds {
			if now.Sub(endedAt) > futuresFinalPushWait {
				delete(e.futuresOrderEnds, id)
				e.futuresStopOrderTypes.Delete(id)
			}
		}
		e.futuresOrderEndsPrunedAt = now
	}
	if e.futuresOrderEnds == nil {
		e.futuresOrderEnds = make(map[string]time.Time)
	}
	e.futuresOrderEnds[orderID] = now
}

// futuresOrderEnded reports whether a push or a REST read reported a futures order ended within futuresFinalPushWait
func (e *Exchange) futuresOrderEnded(orderID string) bool {
	e.futuresOrderEndsMu.Lock()
	defer e.futuresOrderEndsMu.Unlock()
	return e.futuresOrderEndedLocked(orderID)
}

// futuresOrderEndedLocked is futuresOrderEnded with futuresOrderEndsMu held
func (e *Exchange) futuresOrderEndedLocked(orderID string) bool {
	endedAt, ok := e.futuresOrderEnds[orderID]
	return ok && time.Since(endedAt) <= futuresFinalPushWait
}

// processFuturesStopOrder relays a futures Stop Orders push as an order update
func (e *Exchange) processFuturesStopOrder(ctx context.Context, data []byte) error {
	var resp WebsocketFuturesStopOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := stopPushOrderType(resp.OrderType, resp.OrderPrice.Float64())
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	oType = takeProfitType(oType, resp.Stop, side)
	status, err := stopOrderStatus(resp.Type)
	if err != nil {
		return err
	}
	marginType, err := futuresMarginType(resp.MarginMode)
	if err != nil {
		return err
	}
	switch {
	case resp.Type == "cancel":
		// A cancel reports the order's end whatever came before it
	case !e.rememberFuturesStopOrderType(resp.OrderID, oType):
		// On its own topic this push can arrive after a push or a REST read reported the order ended, when its status would
		// reopen the order. It is relayed without one, which the order store leaves as it is; an order the store does not
		// hold is added with its status unknown, which the order manager reads back once the order is no longer listed
		status = order.UnknownStatus
	case resp.Type == "triggered":
		// A triggered futures stop order stays live under its order ID, and the Orders topic follows it from then on
		status = order.Open
	}
	if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Price:        resp.OrderPrice.Float64(),
		TriggerPrice: resp.StopPrice.Float64(),
		Amount:       float64(resp.Size),
		Exchange:     e.Name,
		OrderID:      resp.OrderID,
		Type:         oType,
		Side:         side,
		Status:       status,
		AssetType:    asset.Futures,
		MarginType:   marginType,
		Date:         resp.CreatedAt.Time(),
		LastUpdated:  resp.Timestamp.Time(),
		Pair:         pair,
	}); err != nil {
		return err
	}
	if resp.Type == "cancel" {
		// Noted once relayed, so that the stop pushes of an order whose cancel the relay could not take keep their status
		e.endFuturesOrder(resp.OrderID)
	}
	return nil
}
