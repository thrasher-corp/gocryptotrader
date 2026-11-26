package bingx

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// wsHandleSpotData routes a spot market data frame
func (e *Exchange) wsHandleSpotData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	symbol, stream, data, err := wsMarketFrame(respRaw)
	if err != nil {
		return err
	}
	switch {
	case stream == "trade":
		return e.wsProcessSpotTrade(data)
	case stream == "ticker":
		return e.wsProcessSpotTicker(ctx, data)
	case stream == WebsocketLastPriceChannel:
		return wsRelay[WebsocketSpotLastPrice](ctx, e, data)
	case stream == WebsocketBookTickerChannel:
		return wsRelay[WebsocketSpotBookTicker](ctx, e, data)
	case strings.HasPrefix(stream, "kline_"):
		return e.wsProcessSpotKline(ctx, data)
	case strings.HasPrefix(stream, "depth"):
		return e.wsProcessSpotOrderbook(ctx, conn, symbol+"@"+stream, respRaw, data)
	}
	return fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
}

// wsProcessSpotTrade relays a spot trade of an enabled pair
func (e *Exchange) wsProcessSpotTrade(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketSpotTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.Spot)
	if err != nil || !enabled {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		TID:          resp.TradeID.String(),
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Side:         wsTakerSide(resp.IsBuyerMaker),
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.TradeTime.Time(),
	})
}

// wsProcessSpotTicker stores a spot 24 hour ticker of an enabled pair and relays it
func (e *Exchange) wsProcessSpotTicker(ctx context.Context, data []byte) error {
	var resp WebsocketSpotTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.Spot)
	if err != nil || !enabled {
		return err
	}
	return e.wsProcessTicker(ctx, &ticker.Price{
		Last:                resp.LatestTradePrice.Float64(),
		High:                resp.HighestPrice.Float64(),
		Low:                 resp.LowestPrice.Float64(),
		Bid:                 resp.BestBidPrice.Float64(),
		BidSize:             resp.BestBidQuantity.Float64(),
		Ask:                 resp.BestAskPrice.Float64(),
		AskSize:             resp.BestAskQuantity.Float64(),
		BaseVolume:          resp.Volume.Float64(),
		QuoteVolume:         resp.Turnover.Float64(),
		Open:                resp.OpenPrice.Float64(),
		PercentChange24Hour: resp.PriceChangePercentage.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.Spot,
		LastUpdated:         resp.EventTime.Time(),
	})
}

// wsProcessSpotKline relays a spot candle of an enabled pair, partial while the event precedes the candle's end. Spot
// candles cover UTC+8 periods, as REST candles requested in that time zone do
func (e *Exchange) wsProcessSpotKline(ctx context.Context, data []byte) error {
	var resp WebsocketSpotKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Kline.Symbol, asset.Spot)
	if err != nil || !enabled {
		return err
	}
	interval, err := wsParseInterval(resp.Kline.Interval, spotWebsocketIntervalToString)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:             resp.Kline.StartTime.Time(),
			Open:             resp.Kline.Open.Float64(),
			High:             resp.Kline.High.Float64(),
			Low:              resp.Kline.Low.Float64(),
			Close:            resp.Kline.Close.Float64(),
			Volume:           resp.Kline.Volume.Float64(),
			QuoteVolume:      resp.Kline.Turnover.Float64(),
			ValidationIssues: wsCandleValidation(resp.EventTime.Time(), resp.Kline.StartTime.Time(), interval, websocketUTCPlus8),
		}},
	})
}

// wsProcessSpotOrderbook loads a spot partial depth push of an enabled pair as its book's snapshot. The data carries no
// symbol, so it comes from the dataType, and the time is the frame's
func (e *Exchange) wsProcessSpotOrderbook(ctx context.Context, conn websocket.Connection, dataType string, respRaw, data []byte) error {
	symbol, _, _ := strings.Cut(dataType, "@")
	pair, enabled, err := e.wsMatchPair(symbol, asset.Spot)
	if err != nil || !enabled {
		return err
	}
	var resp WebsocketSpotOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	bids, asks := resp.Bids.Levels(), resp.Asks.Levels()
	wsSortLevels(bids, asks)
	pushed := wsFrameTime(respRaw, "timestamp")
	return e.wsLoadBook(ctx, conn, dataType, &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.Spot,
		LastUpdated:       pushed,
		LastPushed:        pushed,
		LastUpdateID:      resp.LastUpdateID,
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsHandleSpotAccountData routes a spot account data frame. Balance events are stored for the account of the credentials
// the connection's listen key was created for, and those of a connection no longer tracked, whose account nothing
// names, return errWebsocketConnectionUnknown
func (e *Exchange) wsHandleSpotAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	event, data, err := wsAccountEvent(respRaw)
	if err != nil {
		return err
	}
	switch event {
	case "executionReport":
		return e.wsProcessSpotOrderUpdate(ctx, conn, data)
	case "ACCOUNT_UPDATE":
		state, err := e.wsConnection(conn)
		if err != nil {
			return err
		}
		return e.wsProcessSpotAccountUpdate(accounts.DeployCredentialsToContext(ctx, state.credentials), data)
	case "listenKeyExpired":
		return e.wsProcessListenKeyExpired(ctx, conn, data)
	}
	return fmt.Errorf("%w %q on the spot account data stream", errWebsocketUnhandledEvent, event)
}

// wsProcessSpotOrderUpdate relays an executionReport event as an order detail:
//   - p, q, Q, z and Z are the order's price, quantity, quote amount, executed quantity and cost
//   - the average price is Z over z, and the remaining amount q less z, or 0 when z exceeds q
//   - O dates the order and E its update
//
// A fill's fee is reported with the fill, from l, L, Y, t and T, as the order's trade, and BingX reports a charged fee
// as a negative amount. The order's fee is the total the connection's spotOrderFees keeps, which the detail carries
// when it reaches at least the event's executed quantity; an order whose earlier fills the connection did not stream is
// looked up by wsLookUpSpotOrderFee, which relays the fee once it covers them. A connection no longer tracked keeps no
// totals
func (e *Exchange) wsProcessSpotOrderUpdate(ctx context.Context, conn websocket.Connection, data []byte) error {
	var resp WebsocketSpotOrderUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	orderID := resp.OrderID.String()
	pair, err := e.wsAccountPair(resp.Symbol, asset.Spot)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	status, err := convertOrderStatus(resp.OrderStatus)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	orderType, err := convertOrderType(resp.OrderType)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	side, err := convertOrderSide(resp.Side)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	executed := resp.CumulativeFilledQuantity.Decimal()
	d := &order.Detail{
		Price:           resp.OriginalPrice.Float64(),
		Amount:          resp.OriginalQuantity.Float64(),
		QuoteAmount:     resp.OriginalOrderAmount.Float64(),
		ExecutedAmount:  executed.InexactFloat64(),
		RemainingAmount: max(resp.OriginalQuantity.Decimal().Sub(executed).InexactFloat64(), 0),
		Cost:            resp.CumulativeFilledAmount.Float64(),
		FeeAsset:        resp.FeeAsset,
		Exchange:        e.Name,
		OrderID:         orderID,
		ClientOrderID:   resp.ClientOrderID,
		Type:            orderType,
		Side:            side,
		Status:          status,
		AssetType:       asset.Spot,
		Date:            resp.OrderCreationTime.Time(),
		LastUpdated:     resp.EventTime.Time(),
		Pair:            pair,
	}
	if executed.IsPositive() {
		d.AverageExecutedPrice = resp.CumulativeFilledAmount.Decimal().Div(executed).InexactFloat64()
	}
	if resp.ExecutionType == ExecutionTypeTrade {
		d.Trades = []order.TradeHistory{{
			Price:     resp.LastFilledPrice.Float64(),
			Amount:    resp.LastFilledQuantity.Float64(),
			Fee:       -resp.FeeAmount.Float64(),
			Exchange:  e.Name,
			TID:       resp.TradeID.String(),
			Type:      orderType,
			Side:      side,
			Timestamp: resp.TradeTime.Time(),
			IsMaker:   resp.IsMaker,
			FeeAsset:  resp.FeeAsset.String(),
			Total:     resp.LastFilledAmount.Float64(),
		}}
	}
	state, err := e.wsConnection(conn)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, d)
	}
	var fill *spotOrderFill
	filled, filledOK := finiteDecimal(resp.LastFilledQuantity)
	if fee, feeOK := finiteDecimal(resp.FeeAmount); resp.ExecutionType == ExecutionTypeTrade && filledOK && feeOK && filled.IsPositive() {
		fill = &spotOrderFill{from: executed.Sub(filled), to: executed, fee: fee.Neg()}
	}
	// Copied for the lookup, as the receiver can keep d and update it
	event := *d
	lookUp, err := state.spotFees.relay(orderID, d, executed, fill, status.IsInactive(), func() error { return e.Websocket.DataHandler.Send(ctx, d) })
	if lookUp {
		e.wsLookUpSpotOrderFee(ctx, conn, state, &event, uint64(resp.OrderID))
	}
	return err
}

// wsLookUpSpotOrderFee looks up a spot order whose events show fills its account connection did not stream, off the
// connection's reader and with the credentials of the connection's listen key. The lookup is sent once and ends
// websocketOpenOrderReadTimeout after it begins or with the connection's session, and its executed quantity and fee
// settle the order's fee on the connection. The fee is relayed once it reaches every fill the order's events reported,
// while the connection is still up, in a detail dated by the update time BingX reports for the order and holding the
// order's identity, price, amount, quote amount and date from event, the detail of the event that showed the missing
// fills, but no status or executed quantity, which the order's events report
func (e *Exchange) wsLookUpSpotOrderFee(ctx context.Context, conn websocket.Connection, state *websocketConnection, event *order.Detail, orderID uint64) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	e.wsState.mu.Lock()
	timeout := cmp.Or(e.wsState.openOrderReadTimeout, websocketOpenOrderReadTimeout)
	e.wsState.mu.Unlock()
	lookupCtx, cancelLookup := context.WithTimeout(request.WithRetryNotAllowed(accounts.DeployCredentialsToContext(ctx, state.credentials)), timeout)
	relayed := order.Detail{
		Exchange:      event.Exchange,
		OrderID:       event.OrderID,
		ClientOrderID: event.ClientOrderID,
		Pair:          event.Pair,
		AssetType:     event.AssetType,
		Type:          event.Type,
		Side:          event.Side,
		Price:         event.Price,
		Amount:        event.Amount,
		QuoteAmount:   event.QuoteAmount,
		Date:          event.Date,
	}
	e.Websocket.Wg.Add(1)
	e.wsState.background.Go(func() {
		defer e.Websocket.Wg.Done()
		defer cancel()
		defer cancelLookup()
		go func() {
			select {
			case <-state.session:
				cancel()
			case <-ctx.Done():
			}
		}()
		executed, fee, feeAsset, updated, err := e.spotOrderFee(lookupCtx, relayed.Pair, orderID)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s %s order %s fee lookup failed: %v", e.Name, asset.Spot, relayed.OrderID, err)
			_ = state.spotFees.settle(relayed.OrderID, decimal.Zero, decimal.Zero, false, nil)
			return
		}
		err = state.spotFees.settle(relayed.OrderID, executed, fee, true, func(fee decimal.Decimal) error {
			if !wsIsConnected(conn) {
				return nil
			}
			d := relayed
			d.Fee, d.FeeAsset, d.LastUpdated = fee.InexactFloat64(), feeAsset, updated
			return e.Websocket.DataHandler.Send(ctx, &d)
		})
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s %s order %s fee relay failed: %v", e.Name, asset.Spot, relayed.OrderID, err)
		}
	})
}

// spotOrderFee returns the executed quantity, fee, fee currency and update time BingX reports for a spot order. The fee
// is taken as paid whatever its sign, as spotOrderDetail takes it
func (e *Exchange) spotOrderFee(ctx context.Context, pair currency.Pair, orderID uint64) (executed, fee decimal.Decimal, feeAsset currency.Code, updated time.Time, err error) {
	o, err := e.GetSpotOrderDetails(ctx, &SpotOrderDetailsRequest{Symbol: pair, OrderID: orderID})
	if err != nil {
		return decimal.Zero, decimal.Zero, currency.EMPTYCODE, time.Time{}, err
	}
	executed, executedOK := finiteDecimal(o.ExecutedQuantity)
	fee, feeOK := finiteDecimal(o.Fee)
	if !executedOK || !feeOK {
		return decimal.Zero, decimal.Zero, currency.EMPTYCODE, time.Time{}, fmt.Errorf("%w: executed quantity %v, fee %v", errSpotOrderFeeNotFinite, o.ExecutedQuantity.Float64(), o.Fee.Float64())
	}
	return executed, fee.Abs(), o.FeeAsset, o.UpdateTime.Time(), nil
}

// finiteDecimal returns n as a decimal, and whether it is a finite number, which alone a decimal holds
func finiteDecimal(n types.Number) (decimal.Decimal, bool) {
	if f := n.Float64(); math.IsNaN(f) || math.IsInf(f, 0) {
		return decimal.Zero, false
	}
	return n.Decimal(), true
}

// spotOrderFees keeps the fees of the spot orders an account connection streams. A spot order event reports the
// quantity and fee of the fill it reports beside the order's executed quantity, so an order's fee is known from its
// first fill, and each fill after it adds to it. The fills of an order whose earlier fills the connection did not
// stream are kept, while the order is open or a lookup of it is in flight, for a lookup to report the order's fee up to
// them
type spotOrderFees struct {
	mu     sync.Mutex
	orders map[string]*spotOrderFee
}

// spotOrderFee is what an account connection knows of a spot order's fee. With known set, fee is the fee of its fills
// up to executed. fills holds the fills streamed beyond what is known, latest the highest executed quantity its events
// reported, lookups the lookups of it in flight, and closed whether an event closed it
type spotOrderFee struct {
	executed, fee, latest decimal.Decimal
	known, closed         bool
	fills                 []spotOrderFill
	lookups               int
}

// spotOrderFill is a fill an order event reported: the order's executed quantity before and after it, and its fee
type spotOrderFill struct {
	from, to, fee decimal.Decimal
}

// relay relays an order event's detail d with send, carrying the order's fee when what is known of it reaches at least
// the event's executed quantity, and reports whether the order is to be looked up: the event was relayed, the order's
// events report fills the connection did not stream, and no lookup of it is in flight. fill is the fill the event
// reports, if any, and closes whether the event closes the order, whose record is dropped once no lookup of it is in
// flight. An event of an order without a record that reports no fill and nothing executed starts none, as the order's
// first fill shows as much
func (f *spotOrderFees) relay(orderID string, d *order.Detail, executed decimal.Decimal, fill *spotOrderFill, closes bool, send func() error) (lookUp bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[orderID]
	if !ok && fill == nil && executed.IsZero() {
		return false, send()
	}
	if !ok {
		o = new(spotOrderFee)
		if f.orders == nil {
			f.orders = make(map[string]*spotOrderFee)
		}
		f.orders[orderID] = o
	}
	if fill != nil {
		o.add(fill)
	}
	if executed.GreaterThan(o.latest) {
		o.latest = executed
	}
	if o.known && !o.executed.LessThan(executed) {
		d.Fee = o.fee.InexactFloat64()
	}
	err = send()
	o.closed = o.closed || closes
	if lookUp = err == nil && (!o.known || o.executed.LessThan(o.latest)) && o.lookups == 0; lookUp {
		o.lookups++
	}
	if o.closed && o.lookups == 0 {
		delete(f.orders, orderID)
	}
	return lookUp, err
}

// settle ends a lookup of an order. A lookup that succeeded reports the order's fee up to its executed quantity, which
// the fills streamed beyond it add to, and the fee is relayed with send once it reaches every fill the order's events
// reported
func (f *spotOrderFees) settle(orderID string, executed, fee decimal.Decimal, ok bool, send func(fee decimal.Decimal) error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := f.orders[orderID]
	o.lookups--
	defer func() {
		if o.closed && o.lookups == 0 {
			delete(f.orders, orderID)
		}
	}()
	if !ok || o.known && !executed.GreaterThan(o.executed) {
		return nil
	}
	o.known, o.executed, o.fee = true, executed, fee
	o.absorb()
	if o.executed.LessThan(o.latest) {
		return nil
	}
	return send(o.fee)
}

// add counts a fill that continues what is known of the order's fee, or starts it as the order's first, and keeps one
// that does not
func (o *spotOrderFee) add(fill *spotOrderFill) {
	switch {
	case o.known && !fill.to.GreaterThan(o.executed):
	case o.known && fill.from.Equal(o.executed), !o.known && fill.from.IsZero():
		o.known, o.executed, o.fee = true, fill.to, o.fee.Add(fill.fee)
		o.absorb()
	case !slices.ContainsFunc(o.fills, func(kept spotOrderFill) bool { return kept.to.Equal(fill.to) }):
		o.fills = append(o.fills, *fill)
	}
}

// absorb counts the kept fills that continue what is known of the order's fee, and drops those it already counts
func (o *spotOrderFee) absorb() {
	slices.SortFunc(o.fills, func(a, b spotOrderFill) int { return a.from.Cmp(b.from) })
	kept := o.fills[:0]
	for _, fill := range o.fills {
		switch {
		case !fill.to.GreaterThan(o.executed):
		case fill.from.Equal(o.executed):
			o.executed, o.fee = fill.to, o.fee.Add(fill.fee)
		default:
			kept = append(kept, fill)
		}
	}
	if len(kept) == 0 {
		kept = nil
	}
	o.fills = kept
}

// wsProcessSpotAccountUpdate stores the spot balances an ACCOUNT_UPDATE event carries and relays them, leaving other
// currencies as they are
func (e *Exchange) wsProcessSpotAccountUpdate(ctx context.Context, data []byte) error {
	var resp WebsocketSpotAccountUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp.Account.Balances) == 0 {
		return nil
	}
	subAccount, err := e.wsStoreSpotBalances(ctx, resp.Account.Balances)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount})
}

// wsStoreSpotBalances stores the balances of a spot account event and returns them. wb, the asset's total after the
// change, is the total, lk, its frozen amount, is held, and the rest is free. The accounts store stamps a change with
// its arrival, and neither the event nor the spot balance query carries a time to order a REST snapshot by, so the
// write is counted, and the count and the write hold the lock a snapshot's check and save hold: a snapshot requested
// before the write is requested again rather than stored over it, and one stored before it is overwritten by it. The
// write is counted for the account of the credentials ctx names when it starts, whose balances it stores although the
// exchange's default credentials change meanwhile. Spot's account events are the only stream events that store balances
func (e *Exchange) wsStoreSpotBalances(ctx context.Context, balances []WebsocketSpotBalance) (*accounts.SubAccount, error) {
	ctx, k, err := e.accountBalanceContext(ctx, asset.Spot)
	if err != nil {
		return nil, err
	}
	b := &e.balances
	b.writes.Lock()
	defer b.writes.Unlock()
	if b.generations == nil {
		b.generations = make(map[balanceKey]uint64)
	}
	b.generations[k]++
	subAccount := accounts.NewSubAccount(asset.Spot, "")
	for i := range balances {
		total, hold := balances[i].WalletBalance.Float64(), balances[i].FrozenAssets.Float64()
		balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Spot, balances[i].Asset, func(balance *accounts.Balance) {
			balance.Total = total
			balance.Hold = hold
			balance.Free = max(total-hold, 0)
		})
		if err != nil {
			return nil, err
		}
		subAccount.Balances.Set(balances[i].Asset, balance)
	}
	return subAccount, nil
}
