package bingx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// wsHandleCoinMarginedData routes a coin-M perpetual market data frame. Coin-M error replies carry no id, so one goes to
// the request in flight that it names, and is reported otherwise
func (e *Exchange) wsHandleCoinMarginedData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if handled, err := e.wsHandleControlFrame(ctx, conn, respRaw); handled {
		return err
	}
	if code, err := jsonparser.GetInt(respRaw, "code"); err == nil && code != 0 {
		return e.wsRouteCoinMarginedError(conn, respRaw, code)
	}
	symbol, stream, data, err := wsMarketFrame(respRaw)
	if err != nil {
		return err
	}
	switch {
	case stream == "trade":
		return e.wsProcessCoinMarginedTrade(data)
	case stream == "ticker":
		return e.wsProcessCoinMarginedTicker(ctx, data)
	case stream == WebsocketLastPriceChannel:
		return wsRelay[WebsocketCoinMarginedLastPrice](ctx, e, data)
	case stream == WebsocketMarkPriceChannel:
		return wsRelay[WebsocketMarkPrice](ctx, e, data)
	case stream == WebsocketBookTickerChannel:
		return wsRelay[WebsocketCoinMarginedBookTicker](ctx, e, data)
	case strings.HasPrefix(stream, "kline_"):
		return e.wsProcessCoinMarginedKline(ctx, strings.TrimPrefix(stream, "kline_"), respRaw, data)
	case strings.HasPrefix(stream, "depth"):
		return e.wsProcessCoinMarginedOrderbook(ctx, conn, symbol+"@"+stream, data)
	}
	return fmt.Errorf("%w: %s", errWebsocketUnhandledMessage, respRaw)
}

// wsProcessCoinMarginedTrade relays a coin-M perpetual trade of an enabled pair, its amount in contracts
func (e *Exchange) wsProcessCoinMarginedTrade(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketCoinMarginedTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.CoinMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		TID:          resp.TradeID.String(),
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    asset.CoinMarginedFutures,
		Side:         wsTakerSide(resp.IsBuyerMaker),
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.TradeTime.Time(),
	})
}

// wsProcessCoinMarginedTicker stores a coin-M perpetual 24 hour ticker of an enabled pair and relays it. v counts
// contracts, so the base volume is m, the amount in the base coin, and the quote volume q, the amount in USD
func (e *Exchange) wsProcessCoinMarginedTicker(ctx context.Context, data []byte) error {
	var resp WebsocketCoinMarginedTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.CoinMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	return e.wsProcessTicker(ctx, &ticker.Price{
		Last:                resp.LatestPrice.Float64(),
		LastSize:            resp.LatestTradeQuantity.Float64(),
		High:                resp.HighestPrice.Float64(),
		Low:                 resp.LowestPrice.Float64(),
		Bid:                 resp.BestBidPrice.Float64(),
		BidSize:             resp.BestBidQuantity.Float64(),
		Ask:                 resp.BestAskPrice.Float64(),
		AskSize:             resp.BestAskQuantity.Float64(),
		BaseVolume:          resp.TradeAmount.Float64(),
		QuoteVolume:         resp.QuoteTradeAmount.Float64(),
		Open:                resp.OpenPrice.Float64(),
		PercentChange24Hour: resp.PriceChangePercentage.Float64(),
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.CoinMarginedFutures,
		LastUpdated:         resp.EventTime.Time(),
	})
}

// wsProcessCoinMarginedKline relays a coin-M perpetual candle of an enabled pair, its volume in contracts. The stream
// covers UTC+8 periods, unlike REST candles, which follow UTC periods whatever time zone is requested, and sends the
// current minute as each candle's start, so a candle is dated by the start of the period it falls in. It is partial
// while the frame's timestamp, or the receipt time without one, precedes the period's end
func (e *Exchange) wsProcessCoinMarginedKline(ctx context.Context, intervalName string, respRaw, data []byte) error {
	var resp WebsocketCoinMarginedKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.CoinMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	interval, err := wsParseInterval(intervalName, intervalToString)
	if err != nil {
		return err
	}
	start := wsUTCPlus8PeriodStart(resp.StartTime.Time(), interval)
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.CoinMarginedFutures,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:             start,
			Open:             resp.Open.Float64(),
			High:             resp.High.Float64(),
			Low:              resp.Low.Float64(),
			Close:            resp.Close.Float64(),
			Volume:           resp.Volume.Float64(),
			ValidationIssues: wsCandleValidation(wsFrameTime(respRaw, "timestamp"), start, interval, websocketUTCPlus8),
		}},
	})
}

// wsProcessCoinMarginedOrderbook loads a coin-M perpetual limited depth push of an enabled pair as its book's snapshot.
// Levels are counted in contracts, v, as the coin-M REST order book, orders and trades count them; asks arrive highest
// first and are sorted. The book is dated by the push's timestamp, or the receipt time without one
func (e *Exchange) wsProcessCoinMarginedOrderbook(ctx context.Context, conn websocket.Connection, dataType string, data []byte) error {
	var resp WebsocketCoinMarginedOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.wsMatchPair(resp.Symbol, asset.CoinMarginedFutures)
	if err != nil || !enabled {
		return err
	}
	bids, asks := wsCoinMarginedLevels(resp.Bids), wsCoinMarginedLevels(resp.Asks)
	wsSortLevels(bids, asks)
	pushed := resp.Timestamp.Time()
	if pushed.IsZero() {
		pushed = time.Now()
	}
	return e.wsLoadBook(ctx, conn, dataType, &orderbook.Book{
		Bids:              bids,
		Asks:              asks,
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.CoinMarginedFutures,
		LastUpdated:       pushed,
		LastPushed:        pushed,
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsCoinMarginedLevels returns coin-M depth levels sized in contracts
func wsCoinMarginedLevels(levels []WebsocketCoinMarginedOrderbookLevel) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{Price: levels[i].Price.Float64(), Amount: levels[i].Volume.Float64()}
	}
	return l
}

// wsHandleCoinMarginedAccountData routes a coin-M perpetual account data frame
func (e *Exchange) wsHandleCoinMarginedAccountData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.wsHandlePerpetualAccountData(ctx, conn, respRaw, asset.CoinMarginedFutures)
}
