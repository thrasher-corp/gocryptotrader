package htx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	wsSpotHost   = "api.huobi.pro"
	wsSpotURL    = "wss://" + wsSpotHost
	wsPublicPath = "/ws"

	wsCandlesChannel      = "market.%s.kline"
	wsOrderbookChannel    = "market.%s.depth"
	wsTradesChannel       = "market.%s.trade.detail"
	wsMarketDetailChannel = "market.%s.detail"
)

// wsHandleV1ping handles v1 style pings, currently only used with public connections
func (e *Exchange) wsHandleV1ping(ctx context.Context, conn websocket.Connection, pingValue int64) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	if err := conn.SendJSONMessage(ctx, request.Unset, json.RawMessage(`{"pong":`+strconv.FormatInt(pingValue, 10)+`}`)); err != nil {
		return fmt.Errorf("error sending pong response: %w", err)
	}
	return nil
}

func (e *Exchange) wsHandleCandleMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var c WsKline
	if err := json.Unmarshal(respRaw, &c); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Asset:    s.Asset,
		Pair:     s.Pairs[0],
		Interval: s.Interval,
		Candles: []kline.Candle{{
			Time:   time.Unix(c.Tick.ID, 0).UTC(),
			Open:   c.Tick.Open,
			Close:  c.Tick.Close,
			High:   c.Tick.High,
			Low:    c.Tick.Low,
			Volume: c.Tick.Volume,
		}},
	})
}

func (e *Exchange) wsHandleAllTradesMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	tradeFeed := e.IsTradeFeedEnabled()
	if !saveTradeData && !tradeFeed {
		return nil
	}
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var t WsTrade
	if err := json.Unmarshal(respRaw, &t); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(t.Tick.Data))
	for i := range t.Tick.Data {
		side := order.Buy
		if t.Tick.Data[i].Direction != "buy" {
			side = order.Sell
		}
		tradeID := t.Tick.Data[i].TradeID
		if tradeID == "" {
			tradeID = t.Tick.Data[i].ID
		}
		trades = append(trades, trade.Data{
			Exchange:     e.Name,
			AssetType:    s.Asset,
			CurrencyPair: s.Pairs[0],
			Timestamp:    t.Tick.Data[i].Timestamp.Time().UTC(),
			Amount:       t.Tick.Data[i].Amount,
			Price:        t.Tick.Data[i].Price,
			Side:         side,
			TID:          string(tradeID),
		})
	}
	if tradeFeed {
		for i := range trades {
			if err := e.Websocket.DataHandler.Send(ctx, trades[i]); err != nil {
				return err
			}
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(trades...)
	}
	return nil
}

func (e *Exchange) wsHandleTickerMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var wsTicker WsTick
	if err := json.Unmarshal(respRaw, &wsTicker); err != nil {
		return err
	}
	tickPrice := &ticker.Price{
		ExchangeName: e.Name,
		Open:         wsTicker.Tick.Open,
		Last:         wsTicker.Tick.Close,
		Close:        wsTicker.Tick.Close,
		BaseVolume:   wsTicker.Tick.Amount,
		High:         wsTicker.Tick.High,
		Low:          wsTicker.Tick.Low,
		LastUpdated:  wsTicker.Timestamp.Time(),
		AssetType:    s.Asset,
		Pair:         s.Pairs[0],
	}
	if len(wsTicker.Tick.Bid) != 0 {
		if len(wsTicker.Tick.Bid) != 2 {
			return errInvalidBidData
		}
		tickPrice.Bid = wsTicker.Tick.Bid[0]
		tickPrice.BidSize = wsTicker.Tick.Bid[1]
	}
	if len(wsTicker.Tick.Ask) != 0 {
		if len(wsTicker.Tick.Ask) != 2 {
			return errInvalidAskData
		}
		tickPrice.Ask = wsTicker.Tick.Ask[0]
		tickPrice.AskSize = wsTicker.Tick.Ask[1]
	}
	switch s.Asset {
	case asset.Spot:
		tickPrice.QuoteVolume = wsTicker.Tick.Volume
	case asset.USDTMarginedFutures:
		tickPrice.QuoteVolume = wsTicker.Tick.TradeTurnover
	}
	if err := ticker.ProcessTicker(tickPrice); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tickPrice)
}

func (e *Exchange) wsHandleOrderbookMsg(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var update WsDepth
	if err := json.Unmarshal(respRaw, &update); err != nil {
		return err
	}
	bids := make(orderbook.Levels, len(update.Tick.Bids))
	for i := range update.Tick.Bids {
		if len(update.Tick.Bids[i]) != 2 {
			return errInvalidBidData
		}
		price, ok := update.Tick.Bids[i][0].(float64)
		if !ok {
			return errBidPriceTypeAssertion
		}
		amount, ok := update.Tick.Bids[i][1].(float64)
		if !ok {
			return errBidAmountTypeAssertion
		}
		bids[i] = orderbook.Level{
			Price:  price,
			Amount: amount,
		}
	}

	asks := make(orderbook.Levels, len(update.Tick.Asks))
	for i := range update.Tick.Asks {
		if len(update.Tick.Asks[i]) != 2 {
			return errInvalidAskData
		}
		price, ok := update.Tick.Asks[i][0].(float64)
		if !ok {
			return errAskPriceTypeAssertion
		}
		amount, ok := update.Tick.Asks[i][1].(float64)
		if !ok {
			return errAskAmountTypeAssertion
		}
		asks[i] = orderbook.Level{
			Price:  price,
			Amount: amount,
		}
	}

	var newOrderBook orderbook.Book
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = s.Pairs[0]
	newOrderBook.Asset = s.Asset
	newOrderBook.Exchange = e.Name
	newOrderBook.ValidateOrderbook = e.ValidateOrderbook
	newOrderBook.LastUpdated = update.Timestamp.Time()

	return e.Websocket.Orderbook.LoadSnapshot(ctx, &newOrderBook)
}
