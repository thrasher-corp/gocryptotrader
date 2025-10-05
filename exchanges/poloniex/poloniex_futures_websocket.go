package poloniex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	futuresWebsocketPrivateURL = "wss://ws.poloniex.com/ws/v3/private"
	futuresWebsocketPublicURL  = "wss://ws.poloniex.com/ws/v3/public"
)

const (
	channelFuturesSymbol        = "symbol"
	channelFuturesOrderbookLvl2 = "book_lv2"
	channelFuturesOrderbook     = "book"
	channelFuturesTickers       = "tickers"
	channelFuturesTrades        = "trades"
	channelFuturesIndexPrice    = "index_price"
	channelFuturesMarkPrice     = "mark_price"
	channelFuturesFundingRate   = "funding_rate"

	channelFuturesPrivatePositions = "positions"
	channelFuturesPrivateOrders    = "orders"
	channelFuturesPrivateTrades    = "trade"
	channelFuturesAccount          = "account"
)

const (
	candles1Min, candles5Min, candles10Min, candles15Min, candles30Min, candles1Hr, candles2Hr,
	candles4Hr, candles6Hr, candles12Hr, candles1Day, candles3Day, candles1Week, candles1Month = "candles_minute_1", "candles_minute_5", "candles_minute_10", "candles_minute_15", "candles_minute_30", "candles_hour_1",
		"candles_hour_2", "candles_hour_4", "candles_hour_6",
		"candles_hour_12", "candles_day_1", "candles_day_3", "candles_week_1", "candles_month_1"

	markCandles1Min, markCandles5Min, markCandles10Min, markCandles15Min,
	markCandles30Min, markCandles1Hr, markCandles2Hr, markCandles4Hr, markCandles12Hr, markCandles1Day, markCandles3Day, markCandles1Week = "mark_price_candles_minute_1", "mark_price_candles_minute_5", "mark_price_candles_minute_10", "mark_price_candles_minute_15",
		"mark_candles_minute_30", "mark_candles_hour_1", "mark_candles_hour_2", "mark_candles_hour_4", "mark_candles_hour_12",
		"mark_candles_day_1", "mark_candles_day_3", "mark_candles_week_1"

	indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min, indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week = "index_candles_minute_1",
		"index_candles_minute_5", "index_candles_minute_10", "index_candles_minute_15", "index_candles_minute_30", "index_candles_hour_1", "index_candles_hour_2", "index_candles_hour_4",
		"index_candles_hour_12", "index_candles_day_1", "index_candles_day_3", "index_candles_week_1"
)

var (
	defaultFuturesChannels = []string{
		channelFuturesTickers,
		channelFuturesOrderbookLvl2,
		candles15Min,
	}

	defaultPrivateFuturesChannels = []string{
		channelFuturesPrivatePositions,
		channelFuturesPrivateOrders,
		channelFuturesPrivateTrades,
		channelFuturesAccount,
	}

	onceFuturesOrderbook map[string]bool
)

// WsFuturesConnect establishes a websocket connection to the futures websocket server.
func (e *Exchange) WsFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	onceFuturesOrderbook = make(map[string]bool)
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	setupPingHandler(conn)
	return nil
}

// futuresAuthConnect establishes a websocket and authenticates to futures private websocket
func (e *Exchange) futuresAuthConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	setupPingHandler(conn)
	return nil
}

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
	data, err := conn.SendMessageReturnResponse(ctx, request.Auth, "auth", &SubscriptionPayload{
		Event:   "subscribe",
		Channel: []string{"auth"},
		Params: map[string]any{
			"key":           creds.Key,
			"signTimestamp": timestamp,
			"signature":     base64.StdEncoding.EncodeToString(hmac),
		},
	})
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

func (e *Exchange) wsFuturesHandleData(_ context.Context, conn websocket.Connection, respRaw []byte) error {
	var result *FuturesSubscriptionResp
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	if result.Event != "" {
		switch result.Event {
		case "pong", "subscribe":
		case "error":
			if result.Message == "user must be authenticated!" {
				e.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Debugf(log.ExchangeSys, "authenticated websocket disabled%s", string(respRaw))
			}
			fallthrough
		default:
			log.Debugf(log.ExchangeSys, "Unexpected event message %s", string(respRaw))
		}
		return nil
	}
	switch result.Channel {
	case channelAuth:
		return conn.RequireMatchWithData("auth", respRaw)
	case channelFuturesSymbol:
		var resp []ProductInfo
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesOrderbookLvl2,
		channelFuturesOrderbook:
		return e.processFuturesOrderbook(result.Data, result.Action)
	case candles1Min, candles5Min, candles10Min, candles15Min, candles30Min, candles1Hr, candles2Hr, candles4Hr,
		candles6Hr, candles12Hr, candles1Day, candles3Day, candles1Week, candles1Month:
		interval, err := stringToInterval(strings.Join(strings.Split(result.Channel, "_")[1:], "_"))
		if err != nil {
			return err
		}
		return e.processFuturesCandlesticks(result.Data, interval)
	case channelFuturesTickers:
		return e.processFuturesTickers(result.Data)
	case channelFuturesTrades:
		return e.processFuturesTrades(result.Data)
	case channelFuturesIndexPrice:
		var resp []InstrumentIndexPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesMarkPrice:
		var resp []FuturesMarkPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case markCandles1Min, markCandles5Min, markCandles10Min, markCandles15Min,
		markCandles30Min, markCandles1Hr, markCandles2Hr, markCandles4Hr, markCandles12Hr, markCandles1Day, markCandles3Day, markCandles1Week,
		// Index Candlestick channels
		indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min,
		indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week:
		var interval kline.Interval
		var err error
		if strings.HasPrefix(result.Channel, "mark_price") {
			interval, err = stringToInterval(strings.Join(strings.Split(result.Channel, "_")[3:], "_"))
		} else {
			interval, err = stringToInterval(strings.Join(strings.Split(result.Channel, "_")[2:], "_"))
		}
		if err != nil {
			return err
		}
		return e.processFuturesMarkAndIndexPriceCandlesticks(result.Data, interval)
	case channelFuturesFundingRate:
		return e.processFuturesFundingRate(result.Data)
	case channelFuturesPrivatePositions:
		var resp []FuturesPosition
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesPrivateOrders:
		return e.processFuturesOrders(result.Data)
	case channelFuturesPrivateTrades:
		return e.processFuturesTradeFills(result.Data)
	case channelFuturesAccount:
		return e.processFuturesAccountData(result.Data)
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", e.Name, string(respRaw))
	}
}

func (e *Exchange) processFuturesAccountData(data []byte) error {
	var resp []FuturesAccountBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	accChanges := []account.Change{}
	for a := range resp {
		for b := range resp[a].Details {
			accChanges = append(accChanges, account.Change{
				AssetType: asset.Futures,
				Balance: &account.Balance{
					Currency:  resp[a].Details[b].Currency,
					Total:     resp[a].Details[b].Available.Float64(),
					Hold:      resp[a].Details[b].TrdHold.Float64(),
					Free:      resp[a].Details[b].Available.Float64() - resp[a].Details[b].TrdHold.Float64(),
					UpdatedAt: resp[a].Details[b].UpdateTime.Time(),
				},
			})
		}
	}
	e.Websocket.DataHandler <- accChanges
	return nil
}

func (e *Exchange) processFuturesTradeFills(data []byte) error {
	var resp []FuturesTradeFill
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tfills := make([]fill.Data, len(resp))
	for a := range resp {
		oSide, err := order.StringToOrderSide(resp[a].Side)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		tfills[a] = fill.Data{
			CurrencyPair:  cp,
			Side:          oSide,
			Exchange:      e.Name,
			AssetType:     asset.Futures,
			OrderID:       resp[a].OrderID,
			ID:            resp[a].TradeID,
			TradeID:       resp[a].TradeID,
			ClientOrderID: resp[a].ClientOrderID,
			Timestamp:     resp[a].UpdateTime.Time(),
			Price:         resp[a].FillPrice.Float64(),
			Amount:        resp[a].FillQuantity.Float64(),
		}
	}
	e.Websocket.DataHandler <- tfills
	return nil
}

func (e *Exchange) processFuturesOrders(data []byte) error {
	var resp []FuturesOrderDetail
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	orders := make([]order.Detail, len(resp))
	for o := range resp {
		oType, err := order.StringToOrderType(resp[o].OrderType)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp[o].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(resp[o].State)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(resp[o].Symbol)
		if err != nil {
			return err
		}
		orders[o] = order.Detail{
			ReduceOnly:           resp[o].ReduceOnly,
			Leverage:             resp[o].Leverage.Float64(),
			Price:                resp[o].Price.Float64(),
			Amount:               resp[o].Size.Float64(),
			TriggerPrice:         resp[o].TakeProfitTriggerPrice.Float64(),
			AverageExecutedPrice: resp[o].AveragePrice.Float64(),
			ExecutedAmount:       resp[o].ExecQuantity.Float64(),
			RemainingAmount:      resp[o].Size.Float64() - resp[o].ExecQuantity.Float64(),
			Fee:                  resp[o].FeeAmount.Float64(),
			FeeAsset:             currency.NewCode(resp[o].FeeCurrency),
			Exchange:             e.Name,
			OrderID:              resp[o].OrderID,
			ClientOrderID:        resp[o].ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 resp[o].CreationTime.Time(),
			Pair:                 cp,
		}
	}
	e.Websocket.DataHandler <- orders
	return nil
}

func (e *Exchange) processFuturesFundingRate(data []byte) error {
	var resp []FuturesFundingRate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	for a := range resp {
		cp, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		e.Websocket.DataHandler <- websocket.FundingData{
			CurrencyPair: cp,
			Timestamp:    resp[a].Timestamp.Time(),
			AssetType:    asset.Futures,
			Exchange:     e.Name,
			Rate:         resp[a].FundingRate.Float64(),
		}
	}
	return nil
}

func (e *Exchange) processFuturesMarkAndIndexPriceCandlesticks(data []byte, interval kline.Interval) error {
	var resp []WsFuturesMarkAndIndexPriceCandle
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for a := range resp {
		cp, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		candles[a] = websocket.KlineData{
			Timestamp:  resp[a].PushTimestamp.Time(),
			Pair:       cp,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  resp[a].StartTime.Time(),
			CloseTime:  resp[a].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[a].OpeningPrice.Float64(),
			ClosePrice: resp[a].ClosingPrice.Float64(),
			HighPrice:  resp[a].HighestPrice.Float64(),
			LowPrice:   resp[a].LowestPrice.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) processFuturesOrderbook(data []byte, action string) error {
	var resp []FuturesOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	for x := range resp {
		cp, err := currency.NewPairFromString(resp[x].Symbol)
		if err != nil {
			return err
		}
		_, okay := onceFuturesOrderbook[resp[x].Symbol]
		if !okay || action == "snapshot" {
			if onceFuturesOrderbook == nil {
				onceFuturesOrderbook = make(map[string]bool)
			}
			onceFuturesOrderbook[resp[x].Symbol] = true
			if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Bids:         resp[x].Bids.Levels(),
				Asks:         resp[x].Asks.Levels(),
				Exchange:     e.Name,
				Pair:         cp,
				Asset:        asset.Futures,
				LastUpdated:  resp[x].CreationTime.Time(),
				LastUpdateID: resp[x].ID.Int64(),
			}); err != nil {
				return err
			}
			continue
		}
		if err := e.Websocket.Orderbook.Update(&orderbook.Update{
			UpdateID:   resp[x].ID.Int64(),
			UpdateTime: resp[x].CreationTime.Time(),
			LastPushed: resp[x].Timestamp.Time(),
			Action:     orderbook.UpdateAction,
			Asset:      asset.Futures,
			Pair:       cp,
			Asks:       resp[x].Asks.Levels(),
			Bids:       resp[x].Bids.Levels(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesTickers(data []byte) error {
	var resp []FuturesTickerDetail
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tickerPrices := make([]ticker.Price, len(resp))
	for a := range resp {
		cp, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		tickerPrices[a] = ticker.Price{
			High:         resp[a].HighPrice.Float64(),
			Low:          resp[a].LowPrice.Float64(),
			Bid:          resp[a].BestBidPrice.Float64(),
			BidSize:      resp[a].BestBidSize.Float64(),
			Ask:          resp[a].BestAskPrice.Float64(),
			AskSize:      resp[a].BestAskSize.Float64(),
			Volume:       resp[a].Quantity.Float64(),
			QuoteVolume:  resp[a].Amount.Float64(),
			Open:         resp[a].OpeningPrice.Float64(),
			Close:        resp[a].ClosingPrice.Float64(),
			MarkPrice:    resp[a].MarkPrice.Float64(),
			Pair:         cp,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  resp[a].Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- tickerPrices
	return nil
}

// processFuturesTrades handles latest trading data for this product, including the latest price, trading volume, trading direction, etc.
func (e *Exchange) processFuturesTrades(data []byte) error {
	var resp []FuturesTrades
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for t := range resp {
		oSide, err := order.StringToOrderSide(resp[t].Side)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(resp[t].Symbol)
		if err != nil {
			return err
		}
		trades[t] = trade.Data{
			TID:          trades[t].TID,
			Exchange:     e.Name,
			CurrencyPair: cp,
			AssetType:    asset.Futures,
			Side:         oSide,
			Price:        resp[t].Price.Float64(),
			Amount:       resp[t].Amount.Float64(),
			Timestamp:    resp[t].Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- trades
	return nil
}

func (e *Exchange) processFuturesCandlesticks(data []byte, interval kline.Interval) error {
	var resp []WsFuturesCandlesctick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for a := range resp {
		cp, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		candles[a] = websocket.KlineData{
			Timestamp:  resp[a].PushTime.Time(),
			Pair:       cp,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  resp[a].StartTime.Time(),
			CloseTime:  resp[a].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[a].OpenPrice.Float64(),
			ClosePrice: resp[a].ClosePrice.Float64(),
			HighPrice:  resp[a].HighestPrice.Float64(),
			LowPrice:   resp[a].LowestPrice.Float64(),
			Volume:     resp[a].Amount.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

// ------------------------------------------------------------------------------------------------

// GenerateFuturesDefaultSubscriptions adds default subscriptions to futures websockets.
func (e *Exchange) GenerateFuturesDefaultSubscriptions(authenticated bool) (subscription.List, error) {
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	channels := defaultFuturesChannels
	if authenticated {
		channels = defaultPrivateFuturesChannels
	}
	subscriptions := subscription.List{}
	for i := range channels {
		switch channels[i] {
		case channelFuturesAccount:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:       channels[i],
				Asset:         asset.Futures,
				Authenticated: true,
			})
		case channelFuturesPrivatePositions,
			channelFuturesPrivateOrders,
			channelFuturesPrivateTrades,
			channelFuturesSymbol,
			channelFuturesOrderbookLvl2,
			channelFuturesOrderbook,
			channelFuturesTickers,
			channelFuturesTrades,
			channelFuturesIndexPrice,
			channelFuturesMarkPrice,
			indexCandles1Min, indexCandles5Min, indexCandles10Min, indexCandles15Min, indexCandles30Min, indexCandles1Hr, indexCandles2Hr, indexCandles4Hr, indexCandles12Hr, indexCandles1Day, indexCandles3Day, indexCandles1Week,
			channelFuturesFundingRate:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
				Asset:   asset.Futures,
				Pairs:   enabledPairs,
			})
		}
	}
	return subscriptions, nil
}

func (e *Exchange) handleFuturesSubscriptions(operation string, subscs subscription.List) []SubscriptionPayload {
	payloads := []SubscriptionPayload{}
	for x := range subscs {
		if len(subscs[x].Pairs) == 0 {
			input := SubscriptionPayload{
				Event:   operation,
				Channel: []string{subscs[x].Channel},
			}
			payloads = append(payloads, input)
		} else {
			input := SubscriptionPayload{
				Event:   operation,
				Channel: []string{subscs[x].Channel},
			}
			input.Symbols = subscs[x].Pairs.Strings()
			payloads = append(payloads, input)
		}
	}
	return payloads
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (e *Exchange) SubscribeFutures(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	payloads := e.handleFuturesSubscriptions("subscribe", subs)
	for i := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
			return err
		}
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (e *Exchange) UnsubscribeFutures(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
	payloads := e.handleFuturesSubscriptions("unsubscribe", unsub)
	for i := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payloads[i]); err != nil {
			return err
		}
	}
	return e.Websocket.RemoveSubscriptions(conn, unsub...)
}
