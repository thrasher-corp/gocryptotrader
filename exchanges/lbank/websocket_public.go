package lbank

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// wsProcessTicker stores a pair's ticker and relays it
func (e *Exchange) wsProcessTicker(ctx context.Context, msg *websocketMessage) error {
	if msg.Ticker == nil {
		return fmt.Errorf("%w: ticker", common.ErrNoResponse)
	}
	pair, err := e.matchSpotPair(msg.Pair)
	if err != nil {
		return err
	}
	tick := &ticker.Price{
		ExchangeName:        e.Name,
		Pair:                pair,
		AssetType:           asset.Spot,
		Last:                msg.Ticker.LastPrice,
		High:                msg.Ticker.High24Hour,
		Low:                 msg.Ticker.Low24Hour,
		BaseVolume:          msg.Ticker.Volume24Hour,
		QuoteVolume:         msg.Ticker.Turnover24Hour,
		PercentChange24Hour: msg.Ticker.ChangePercentage24Hour,
		LastUpdated:         msg.Time.Time(),
	}
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// wsProcessDepth loads a pair's order book. LBank pushes the whole book to the depth subscribed to each time, so each
// push is a snapshot
func (e *Exchange) wsProcessDepth(ctx context.Context, msg *websocketMessage) error {
	if msg.Depth == nil {
		return fmt.Errorf("%w: depth", common.ErrNoResponse)
	}
	pair, err := e.matchSpotPair(msg.Pair)
	if err != nil {
		return err
	}
	lastUpdated := msg.DepthTime.Time()
	if lastUpdated.IsZero() {
		lastUpdated = msg.Time.Time()
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.Spot,
		Bids:              msg.Depth.Bids.Levels(),
		Asks:              msg.Depth.Asks.Levels(),
		LastUpdated:       lastUpdated,
		LastPushed:        msg.Time.Time(),
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsProcessTrade relays a public trade
func (e *Exchange) wsProcessTrade(msg *websocketMessage) error {
	if msg.Trade == nil {
		return fmt.Errorf("%w: trade", common.ErrNoResponse)
	}
	pair, err := e.matchSpotPair(msg.Pair)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(msg.Trade.Direction)
	if err != nil {
		return err
	}
	return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), trade.Data{
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Side:         side,
		Price:        msg.Trade.Price,
		Amount:       msg.Trade.Quantity,
		Timestamp:    msg.Trade.Time.Time(),
	})
}

// wsProcessKline relays a candle
func (e *Exchange) wsProcessKline(ctx context.Context, msg *websocketMessage) error {
	if msg.Kline == nil {
		return fmt.Errorf("%w: kline", common.ErrNoResponse)
	}
	pair, err := e.matchSpotPair(msg.Pair)
	if err != nil {
		return err
	}
	interval, err := intervalFromKlineSlot(msg.Kline.Slot)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   msg.Kline.Time.Time(),
			Open:   msg.Kline.Open,
			High:   msg.Kline.High,
			Low:    msg.Kline.Low,
			Close:  msg.Kline.Close,
			Volume: msg.Kline.Volume,
		}},
	})
}

// intervalFromKlineSlot returns the interval of a streamed candle slot
func intervalFromKlineSlot(slot string) (kline.Interval, error) {
	for interval, s := range websocketKlineSlots {
		if s == slot {
			return interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrUnsupportedInterval, slot)
}
