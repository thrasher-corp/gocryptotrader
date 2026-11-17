package bingx

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// wsHandleSwapData routes a USDT-M perpetual market data frame
func (e *Exchange) wsHandleSwapData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	symbol, stream, data, err := wsMarketFrame(respRaw)
	if err != nil {
		return err
	}
	switch {
	case stream == "trade":
		return e.wsProcessSwapTrades(data)
	case stream == "ticker":
		return e.wsProcessSwapTicker(ctx, data)
	case stream == WebsocketLastPriceChannel:
		return wsRelay[WebsocketSwapLastPrice](ctx, e, data)
	case stream == WebsocketMarkPriceChannel:
		return wsRelay[WebsocketMarkPrice](ctx, e, data)
	case stream == WebsocketBookTickerChannel:
		return wsRelay[WebsocketSwapBookTicker](ctx, e, data)
	case strings.HasPrefix(stream, "kline_"):
		return e.wsProcessSwapKlines(ctx, symbol, strings.TrimPrefix(stream, "kline_"), data)
	case strings.HasPrefix(stream, "depth"):
		return e.wsProcessSwapOrderbook(ctx, conn, symbol+"@"+stream, respRaw, data)
	}
	return fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
}

// wsProcessSwapTrades relays the trades of a USDT-M perpetual trade push, an array, for an enabled pair
func (e *Exchange) wsProcessSwapTrades(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp []WebsocketSwapTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(resp))
	for i := range resp {
		pair, enabled, err := e.wsMatchPair(resp[i].Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}
		trades = append(trades, trade.Data{
			TID:          resp[i].TradeID.String(),
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.USDTMarginedFutures,
			Side:         wsTakerSide(resp[i].IsBuyerMaker.Bool()),
			Price:        resp[i].Price.Float64(),
			Amount:       resp[i].Quantity.Float64(),
			Timestamp:    resp[i].TradeTime.Time(),
		})
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// wsProcessSwapTicker stores a USDT-M perpetual 24 hour ticker of an enabled pair and relays it
func (e *Exchange) wsProcessSwapTicker(ctx context.Context, data []byte) error {
	var resp WebsocketSwapTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	return e.wsProcessTicker(ctx, &ticker.Price{
		Last:                resp.LatestTransactionPrice.Float64(),
		LastSize:            resp.LatestTransactionAmount.Float64(),
		High:                resp.HighPrice.Float64(),
		Low:                 resp.LowPrice.Float64(),
		Bid:                 resp.BestBidPrice.Float64(),
		BidSize:             resp.BestBidQuantity.Float64(),
		Ask:                 resp.BestAskPrice.Float64(),
		AskSize:             resp.BestAskQuantity.Float64(),
		BaseVolume:          resp.Volume.Float64(),
		QuoteVolume:         resp.Turnover.Float64(),
		Open:                resp.FirstPrice.Float64(),
		PercentChange24Hour: resp.PriceChangePercentage.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.USDTMarginedFutures,
		LastUpdated:         resp.EventTime.Time(),
	})
}

// wsProcessSwapKlines relays the candles of a USDT-M perpetual candle push, an array, for an enabled pair. The push
// carries no time of its own, so a candle is partial while it is received before its end
func (e *Exchange) wsProcessSwapKlines(ctx context.Context, symbol, intervalName string, data []byte) error {
	pair, enabled, err := e.wsMatchPair(symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	interval, err := wsParseInterval(intervalName, intervalToString)
	if err != nil {
		return err
	}
	var resp []WebsocketSwapKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp) == 0 {
		return nil
	}
	received := time.Now()
	candles := make([]kline.Candle, len(resp))
	for i := range resp {
		candles[i] = kline.Candle{
			Time:             resp[i].Time.Time(),
			Open:             resp[i].Open.Float64(),
			High:             resp[i].High.Float64(),
			Low:              resp[i].Low.Float64(),
			Close:            resp[i].Close.Float64(),
			Volume:           resp[i].Volume.Float64(),
			ValidationIssues: wsCandleValidation(received, resp[i].Time.Time(), interval, time.UTC),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.USDTMarginedFutures,
		Interval: interval,
		Candles:  candles,
	})
}

// wsProcessSwapOrderbook loads a USDT-M perpetual partial depth push of an enabled pair as its book's snapshot, dated by
// ts, the time the server built the message. The data carries no symbol, so it comes from the dataType
func (e *Exchange) wsProcessSwapOrderbook(ctx context.Context, conn websocket.Connection, dataType string, respRaw, data []byte) error {
	symbol, _, _ := strings.Cut(dataType, "@")
	pair, enabled, err := e.wsMatchPair(symbol, asset.USDTMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	var resp WebsocketSwapOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	bids, asks := resp.Bids.Levels(), resp.Asks.Levels()
	wsSortLevels(bids, asks)
	pushed := wsFrameTime(respRaw, "ts")
	return e.wsLoadBook(ctx, conn, dataType, &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.USDTMarginedFutures,
		LastUpdated:       pushed,
		LastPushed:        pushed,
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsHandleSwapAccountData routes a USDT-M perpetual account data frame
func (e *Exchange) wsHandleSwapAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.wsHandlePerpetualAccountData(ctx, conn, respRaw, asset.USDTMarginedFutures)
}

// wsHandlePerpetualAccountData routes a USDT-M or coin-M perpetual account data frame, which BingX pushes every event to
// without a subscription. Order events arrive as ORDER_TRADE_UPDATE, or TRADE_UPDATE as the USDT-M docs example names
// them, with or without an envelope
func (e *Exchange) wsHandlePerpetualAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	event, data, err := wsAccountEvent(respRaw)
	if err != nil {
		return err
	}
	switch event {
	case "ORDER_TRADE_UPDATE", "TRADE_UPDATE":
		return e.wsProcessPerpetualOrderUpdate(ctx, a, data)
	case "ACCOUNT_UPDATE":
		return e.wsProcessPerpetualAccountUpdate(ctx, conn, a, data)
	case "ACCOUNT_CONFIG_UPDATE":
		return wsRelay[WebsocketPerpetualConfigUpdate](ctx, e, data)
	case "listenKeyExpired":
		return e.wsProcessListenKeyExpired(ctx, conn, data)
	}
	return fmt.Errorf("%w %q on the %s account data stream", errWebsocketUnhandledEvent, event, a)
}

// wsProcessPerpetualOrderUpdate relays a perpetual order event as an order detail:
//   - p, q, sp, ap and z are the order's price, quantity, trigger price, average price and executed quantity
//   - the remaining amount is q less z, and ro marks a reduce only order
//   - the event carries no creation time, so the order is dated by its update: the event's T, or E when T is unset
//   - a USDT-M order's cost is ap times z in the quote currency; a coin-M order's cost needs the contract size the event
//     lacks
//
// The event reports a fill's fee alone, while an order detail carries the order's fee, so the fee of a TRADE or
// CALCULATED fill is added to the order's running total, which the detail carries. BingX reports a charged fee as a
// negative amount
func (e *Exchange) wsProcessPerpetualOrderUpdate(ctx context.Context, a asset.Item, data []byte) error {
	var resp WebsocketPerpetualOrderUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	o := &resp.Order
	orderID := o.OrderID.String()
	pair, err := e.wsAccountPair(o.Symbol, a)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	status, err := convertOrderStatus(o.OrderStatus)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	orderType, err := convertOrderType(o.OrderType)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	side, err := convertOrderSide(o.Side)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	updated := o.TradeTime.Time()
	if updated.IsZero() {
		updated = resp.EventTime.Time()
	}
	executed := o.CumulativeFilledQuantity.Decimal()
	d := &order.Detail{
		ReduceOnly:           o.ReduceOnly.Bool(),
		Price:                o.Price.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.TriggerPrice.Float64(),
		AverageExecutedPrice: o.AverageFilled.Float64(),
		ExecutedAmount:       executed.InexactFloat64(),
		RemainingAmount:      max(o.Quantity.Decimal().Sub(executed).InexactFloat64(), 0),
		FeeAsset:             o.FeeAsset,
		Exchange:             e.Name,
		OrderID:              orderID,
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            a,
		Date:                 updated,
		LastUpdated:          updated,
		Pair:                 pair,
	}
	if a == asset.USDTMarginedFutures {
		d.Cost = o.AverageFilled.Decimal().Mul(executed).InexactFloat64()
	}
	k := perpetualOrderFeeKey{asset: a, orderID: orderID}
	if o.ExecutionType == ExecutionTypeTrade || o.ExecutionType == ExecutionTypeCalculated {
		d.Fee = e.perpetualOrderFees.add(k, executed, o.Fee.Decimal().Neg()).InexactFloat64()
	}
	if d.IsInactive() {
		e.perpetualOrderFees.forget(k)
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

// perpetualOrderFeeTotals keeps the fee of each open perpetual order's fills that the account streams have reported
type perpetualOrderFeeTotals struct {
	mu     sync.Mutex
	orders map[perpetualOrderFeeKey]perpetualOrderFeeTotal
}

// perpetualOrderFeeKey names a perpetual order by its asset and order ID
type perpetualOrderFeeKey struct {
	asset   asset.Item
	orderID string
}

// perpetualOrderFeeTotal is the fee of an order's fills up to its executed quantity
type perpetualOrderFeeTotal struct {
	executed decimal.Decimal
	fee      decimal.Decimal
}

// add counts a fill's fee once, when its event raises the order's executed quantity, which a repeated event does not,
// and returns the order's fee so far
func (t *perpetualOrderFeeTotals) add(k perpetualOrderFeeKey, executed, fee decimal.Decimal) decimal.Decimal {
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.orders[k]
	if !executed.GreaterThan(total.executed) {
		return total.fee
	}
	total.executed = executed
	total.fee = total.fee.Add(fee)
	if t.orders == nil {
		t.orders = make(map[perpetualOrderFeeKey]perpetualOrderFeeTotal)
	}
	t.orders[k] = total
	return total.fee
}

// forget drops the total of an order that is no longer open
func (t *perpetualOrderFeeTotals) forget(k perpetualOrderFeeKey) {
	t.mu.Lock()
	delete(t.orders, k)
	t.mu.Unlock()
}

// wsProcessPerpetualAccountUpdate relays a perpetual ACCOUNT_UPDATE event's balances and positions. A balance's wb is
// its wallet balance, and nothing in the event gives the margin available or held, so the balances are relayed as the
// event reports them rather than stored, and the asset's balances are refreshed over REST instead, once for a burst of
// events, for the account of the credentials the connection's listen key was created for. A connection no longer
// tracked refreshes nothing, as its session has ended. A position whose symbol cannot be read is left out
func (e *Exchange) wsProcessPerpetualAccountUpdate(ctx context.Context, conn websocket.Connection, a asset.Item, data []byte) error {
	var resp WebsocketPerpetualAccountUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp.Account.Balances) != 0 {
		if state, err := e.wsConnection(conn); err == nil {
			e.refreshBalances(accounts.DeployCredentialsToContext(ctx, state.credentials), a, state.session)
		}
		if err := e.Websocket.DataHandler.Send(ctx, resp.Account.Balances); err != nil {
			return err
		}
	}
	positions := make([]futures.Position, 0, len(resp.Account.Positions))
	for i := range resp.Account.Positions {
		p := &resp.Account.Positions[i]
		pair, err := e.wsAccountPair(p.Symbol, a)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s %s account update: leaving out a position: %v", e.Name, a, err)
			continue
		}
		positions = append(positions, wsPerpetualPosition(e.Name, a, pair, p, resp.EventTime.Time()))
	}
	if len(positions) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

// wsPerpetualPosition builds a position from a perpetual account event's: pa is its signed size, ep its entry price,
// iw its isolated margin, and up and cr its unrealised and realised profit. Hedge mode reports the side, while one-way
// mode, BOTH, signs the size; a zero size closes the position
func wsPerpetualPosition(exchangeName string, a asset.Item, pair currency.Pair, p *WebsocketPerpetualPosition, updated time.Time) futures.Position {
	amount := p.PositionAmount.Decimal()
	position := futures.Position{
		Exchange:       exchangeName,
		Asset:          a,
		Pair:           pair,
		PositionMargin: p.IsolatedPositionMargin.Decimal(),
		RealisedPNL:    p.RealisedPNL.Decimal(),
		UnrealisedPNL:  p.UnrealisedPNL.Decimal(),
		Status:         order.Open,
		OpeningPrice:   p.EntryPrice.Decimal(),
		LatestSize:     amount.Abs(),
		LastUpdated:    updated,
	}
	switch {
	case p.PositionSide == PositionSideLong, p.PositionSide != PositionSideShort && amount.IsPositive():
		position.OpeningDirection = order.Long
	case p.PositionSide == PositionSideShort, amount.IsNegative():
		position.OpeningDirection = order.Short
	}
	position.LatestDirection = position.OpeningDirection
	if amount.IsZero() {
		position.Status = order.Closed
		position.CloseDate = updated
	}
	return position
}
