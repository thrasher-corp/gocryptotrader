package bingx

import (
	"context"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
		return e.wsProcessSpotOrderUpdate(ctx, data)
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
//   - the average price is Z over z, and the remaining amount q less z
//   - O dates the order and E its update
//
// A fill's fee is the fill's alone, so it is reported with the fill, from l, L, Y, t and T, as the order's trade
// rather than as the order's fee. BingX reports a charged fee as a negative amount
func (e *Exchange) wsProcessSpotOrderUpdate(ctx context.Context, data []byte) error {
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
	return e.Websocket.DataHandler.Send(ctx, d)
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
