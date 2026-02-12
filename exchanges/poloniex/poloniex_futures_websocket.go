package poloniex

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	futuresWebsocketPrivateURL = "wss://ws.poloniex.com/ws/v3/private"
	futuresWebsocketPublicURL  = "wss://ws.poloniex.com/ws/v3/public"
)

const (
	// Public channels
	channelFuturesSymbol           = "symbol"
	channelFuturesOrderbookLvl2    = "book_lv2"
	channelFuturesOrderbook        = "book"
	channelFuturesTickers          = "tickers"
	channelFuturesTrades           = "trades"
	channelFuturesIndexPrice       = "index_price"
	channelFuturesMarkPrice        = "mark_price"
	channelFuturesFundingRate      = "funding_rate"
	channelFuturesMarkPriceCandles = "mark_price_candles"
	channelFuturesMarkCandles      = "mark_candles"
	channelFuturesCandles          = "candles"
	channelFuturesIndexCandles     = "index_candles"
	channelFuturesLimitPrice       = "limit_price"
	channelFuturesLiquidationPrice = "liquidation_orders"
	channelFuturesOpenInterest     = "open_interest"

	// Authenticated channels
	channelFuturesPrivatePositions = "positions"
	channelFuturesPrivateOrders    = "orders"
	channelFuturesPrivateTrades    = "trade"
	channelFuturesAccount          = "account"
)

var (
	futuresDefaultSubscriptions = subscription.List{
		{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.FiveMin},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	}

	futuresPrivateDefaultSubscriptions = subscription.List{
		{Enabled: true, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.MyTradesChannel, Authenticated: true},
	}

	futuresSubscriptionNames = map[string]string{
		subscription.CandlesChannel:   channelFuturesCandles,
		subscription.AllTradesChannel: channelFuturesTrades,
		subscription.TickerChannel:    channelFuturesTickers,
		subscription.OrderbookChannel: channelFuturesOrderbookLvl2,
		subscription.MyOrdersChannel:  channelFuturesPrivateOrders,
		subscription.MyAccountChannel: channelFuturesAccount,
		subscription.MyTradesChannel:  channelFuturesPrivateTrades,
	}
)

// authenticateFuturesAuthConn authenticates a futures websocket connection
func (e *Exchange) authenticateFuturesAuthConn(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	timestamp := time.Now().UnixMilli()
	signatureStrings := "GET\n/ws\nsignTimestamp=" + strconv.FormatInt(timestamp, 10)

	var hmac []byte
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signatureStrings),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	e.futuresSubMtx.Lock()
	data, err := conn.SendMessageReturnResponse(ctx, fWebsocketPrivateEPL, "auth", &SubscriptionPayload{
		Event:   "subscribe",
		Channel: []string{"auth"},
		Params: map[string]any{
			"key":           creds.Key,
			"signTimestamp": timestamp,
			"signature":     base64.StdEncoding.EncodeToString(hmac),
		},
	})
	e.futuresSubMtx.Unlock()
	if err != nil {
		return err
	}
	var resp *AuthenticationResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if !resp.Data.Success {
		return fmt.Errorf("authentication failed with status code: %s", resp.Data.Message)
	}
	return nil
}

func (e *Exchange) generateFuturesSubscriptions() (subscription.List, error) {
	return futuresDefaultSubscriptions.ExpandTemplates(e)
}

func (e *Exchange) generateFuturesPrivateSubscriptions() (subscription.List, error) {
	return futuresPrivateDefaultSubscriptions.ExpandTemplates(e)
}

func (e *Exchange) wsFuturesHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var result *SubscriptionResponse
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.Event != "" {
		switch result.Event {
		case "pong":
			return nil
		case "subscribe", "unsubscribe", "error":
			return conn.RequireMatchWithData("subscription", respRaw)
		default:
			return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
		}
	}
	switch result.Channel {
	case channelAuth:
		return conn.RequireMatchWithData(channelAuth, respRaw)
	case channelFuturesSymbol:
		var resp []*WSProductDetail
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesOrderbookLvl2:
		return e.processFuturesOrderbookLevel2(result.Data, result.Action)
	case channelFuturesOrderbook:
		return e.processFuturesOrderbook(result.Data)
	case channelFuturesTickers:
		return e.processFuturesTickers(ctx, result.Data)
	case channelFuturesTrades:
		return e.processFuturesTrades(ctx, result.Data)
	case channelFuturesIndexPrice:
		var resp []*WSInstrumentIndexPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesMarkPrice:
		var resp []*FuturesWebsocketMarkPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesFundingRate:
		return e.processFuturesFundingRate(ctx, result.Data)
	case channelFuturesPrivatePositions:
		var resp []*WsFuturesPosition
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesPrivateOrders:
		return e.processFuturesOrders(ctx, result.Data)
	case channelFuturesPrivateTrades:
		return e.processFuturesTradeFills(ctx, result.Data)
	case channelFuturesAccount:
		return e.processFuturesAccountData(ctx, result.Data)
	case channelFuturesLimitPrice:
		var resp []*FuturesLimitPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesLiquidationPrice:
		var resp []*FuturesLiquidationOrder
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	case channelFuturesOpenInterest:
		var resp []*FuturesOpenInterest
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, resp)
	default:
		if strings.Contains(result.Channel, "_") {
			channel, interval, err := channelToIntervalSplit(result.Channel)
			if err != nil {
				return err
			}
			switch channel {
			case channelFuturesMarkPriceCandles, channelFuturesMarkCandles, channelFuturesIndexCandles:
				return e.processFuturesMarkAndIndexPriceCandlesticks(ctx, result.Data, interval)
			case channelFuturesCandles:
				return e.processFuturesCandlesticks(ctx, result.Data, interval)
			}
		}
		return fmt.Errorf("%s %s %s", e.Name, websocket.UnhandledMessage, string(respRaw))
	}
}

func channelToIntervalSplit(intervalString string) (string, kline.Interval, error) {
	splits := strings.Split(intervalString, "_")
	length := len(splits)
	if length < 3 {
		return intervalString, kline.Interval(0), fmt.Errorf("%w %q", kline.ErrInvalidInterval, intervalString)
	}
	intervalValue, err := stringToInterval(strings.Join(splits[length-2:], "_"))
	return strings.Join(splits[:length-2], "_"), intervalValue, err
}

func (e *Exchange) processFuturesAccountData(ctx context.Context, data []byte) error {
	var resp []*FuturesAccountBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	var accChanges []accounts.Change
	for _, r := range resp {
		for _, detail := range r.Details {
			accChanges = append(accChanges, accounts.Change{
				AssetType: asset.Futures,
				Balance: accounts.Balance{
					Currency:  detail.Currency,
					Total:     detail.Available.Float64(),
					Hold:      detail.TradeHold.Float64(),
					Free:      detail.Available.Float64() - detail.TradeHold.Float64(),
					UpdatedAt: detail.UpdateTime.Time(),
				},
			})
		}
	}
	return e.Websocket.DataHandler.Send(ctx, accChanges)
}

func (e *Exchange) processFuturesTradeFills(ctx context.Context, data []byte) error {
	var resp []*WSTradeFill
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tFills := make([]fill.Data, len(resp))
	for i, r := range resp {
		oSide, err := order.StringToOrderSide(r.Side)
		if err != nil {
			return err
		}
		tFills[i] = fill.Data{
			Side:          oSide,
			Exchange:      e.Name,
			AssetType:     asset.Futures,
			CurrencyPair:  r.Symbol,
			OrderID:       r.OrderID,
			ID:            r.TradeID,
			TradeID:       r.TradeID,
			ClientOrderID: r.ClientOrderID,
			Timestamp:     r.UpdateTime.Time(),
			Amount:        r.FillQuantity.Float64(),
			Price:         r.FillPrice.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, tFills)
}

func (e *Exchange) processFuturesOrders(ctx context.Context, data []byte) error {
	var resp []*FuturesWebsocketOrderDetails
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	orders := make([]order.Detail, len(resp))
	for i, r := range resp {
		oStatus, err := order.StringToOrderStatus(r.State)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(r.OrderType)
		if err != nil {
			return err
		}
		var marginMode margin.Type
		if r.MarginMode != "" {
			marginMode, err = margin.StringToMarginType(r.MarginMode)
			if err != nil {
				return err
			}
		}
		orders[i] = order.Detail{
			ReduceOnly:           r.ReduceOnly,
			Price:                r.Price.Float64(),
			Amount:               r.Size.Float64(),
			AverageExecutedPrice: r.AveragePrice.Float64(),
			ExecutedAmount:       r.ExecutedQuantity.Float64(),
			RemainingAmount:      r.Size.Float64() - r.ExecutedQuantity.Float64(),
			Fee:                  r.FeeAmount.Float64(),
			FeeAsset:             r.FeeCurrency,
			Exchange:             e.Name,
			OrderID:              r.OrderID,
			ClientOrderID:        r.ClientOrderID,
			Type:                 oType,
			Side:                 r.Side,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 r.CreationTime.Time(),
			LastUpdated:          r.UpdateTime.Time(),
			TimeInForce:          r.TimeInForce,
			MarginType:           marginMode,
			Pair:                 r.Symbol,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, orders)
}

func (e *Exchange) processFuturesFundingRate(ctx context.Context, data []byte) error {
	var resp []*WSFuturesFundingRate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	for _, r := range resp {
		if err := e.Websocket.DataHandler.Send(ctx, websocket.FundingData{
			CurrencyPair: r.Symbol,
			Timestamp:    r.PushTime.Time(),
			AssetType:    asset.Futures,
			Exchange:     e.Name,
			Rate:         r.FundingRate.Float64(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesMarkAndIndexPriceCandlesticks(ctx context.Context, data []byte, interval kline.Interval) error {
	var resp []*WsFuturesMarkAndIndexPriceCandle
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for i, r := range resp {
		candles[i] = websocket.KlineData{
			Timestamp:  r.PushTimestamp.Time(),
			Pair:       r.Symbol,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  r.StartTime.Time(),
			CloseTime:  r.EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  r.OpeningPrice.Float64(),
			ClosePrice: r.ClosingPrice.Float64(),
			HighPrice:  r.HighestPrice.Float64(),
			LowPrice:   r.LowestPrice.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, candles)
}

func (e *Exchange) processFuturesOrderbook(data []byte) error {
	var resp []*WSFuturesOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	for _, r := range resp {
		if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Bids:         r.Bids.Levels(),
			Asks:         r.Asks.Levels(),
			Exchange:     e.Name,
			LastUpdateID: r.ID,
			Asset:        asset.Futures,
			Pair:         r.Symbol,
			LastUpdated:  r.CreationTime.Time(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesOrderbookLevel2(data []byte, action string) error {
	var resp []*WSFuturesOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	for _, r := range resp {
		if action == "snapshot" {
			if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Bids:         r.Bids.Levels(),
				Asks:         r.Asks.Levels(),
				Exchange:     e.Name,
				Asset:        asset.Futures,
				Pair:         r.Symbol,
				LastUpdated:  r.CreationTime.Time(),
				LastUpdateID: r.LastVersionID,
			}); err != nil {
				return err
			}
			continue
		}
		if err := e.Websocket.Orderbook.Update(&orderbook.Update{
			UpdateID:   r.ID,
			UpdateTime: r.CreationTime.Time(),
			LastPushed: r.Timestamp.Time(),
			Action:     orderbook.UpdateAction,
			Asset:      asset.Futures,
			Pair:       r.Symbol,
			Asks:       r.Asks.Levels(),
			Bids:       r.Bids.Levels(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesTickers(ctx context.Context, data []byte) error {
	var resp []*FuturesTickerDetails
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tickerPrices := make([]ticker.Price, len(resp))
	for i, r := range resp {
		tickerPrices[i] = ticker.Price{
			High:         r.HighPrice.Float64(),
			Low:          r.LowPrice.Float64(),
			Bid:          r.BestBidPrice.Float64(),
			BidSize:      r.BestBidSize.Float64(),
			Ask:          r.BestAskPrice.Float64(),
			AskSize:      r.BestAskSize.Float64(),
			Volume:       r.BaseAmount.Float64(),
			QuoteVolume:  r.QuoteAmount.Float64(),
			Open:         r.OpeningPrice.Float64(),
			Close:        r.ClosingPrice.Float64(),
			MarkPrice:    r.MarkPrice.Float64(),
			Pair:         r.Symbol,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  r.Timestamp.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, tickerPrices)
}

// processFuturesTrades handles latest trading data for this product, including the latest price, trading volume, trading direction, etc.
func (e *Exchange) processFuturesTrades(ctx context.Context, data []byte) error {
	var resp []*FuturesTrades
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for i, r := range resp {
		oSide, err := order.StringToOrderSide(resp[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			TID:          strconv.FormatInt(r.ID, 10),
			Exchange:     e.Name,
			Side:         oSide,
			AssetType:    asset.Futures,
			CurrencyPair: r.Symbol,
			Price:        r.Price.Float64(),
			Amount:       r.BaseAmount.Float64(),
			Timestamp:    r.Timestamp.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, trades)
}

func (e *Exchange) processFuturesCandlesticks(ctx context.Context, data []byte, interval kline.Interval) error {
	var resp []*WsFuturesCandlestick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for i, r := range resp {
		candles[i] = websocket.KlineData{
			Timestamp:  r.PushTime.Time(),
			Pair:       r.Symbol,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  r.StartTime.Time(),
			CloseTime:  r.EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  r.OpenPrice.Float64(),
			ClosePrice: r.ClosePrice.Float64(),
			HighPrice:  r.HighestPrice.Float64(),
			LowPrice:   r.LowestPrice.Float64(),
			Volume:     r.QuoteAmount.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, candles)
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (e *Exchange) SubscribeFutures(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubs(ctx, "subscribe", conn, subs, &e.futuresSubMtx)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (e *Exchange) UnsubscribeFutures(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
	return e.manageSubs(ctx, "unsubscribe", conn, unsub, &e.futuresSubMtx)
}
