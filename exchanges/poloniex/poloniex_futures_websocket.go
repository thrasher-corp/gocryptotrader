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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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

func (e *Exchange) generateFuturesSubscriptions() (subscription.List, error) {
	return futuresDefaultSubscriptions.ExpandTemplates(e)
}

func (e *Exchange) generateFuturesPrivateSubscriptions() (subscription.List, error) {
	return futuresPrivateDefaultSubscriptions.ExpandTemplates(e)
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
			log.Debugf(log.ExchangeSys, "Unexpected event message futures %s", string(respRaw))
		}
		return nil
	}
	switch result.Channel {
	case channelAuth:
		return conn.RequireMatchWithData("auth", respRaw)
	case channelFuturesSymbol:
		var resp []*ProductDetail
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesOrderbookLvl2,
		channelFuturesOrderbook:
		return e.processFuturesOrderbook(result.Data, result.Action)
	case channelFuturesCandles:
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
		var resp []*InstrumentIndexPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesMarkPrice:
		var resp []*FuturesMarkPrice
		if err := json.Unmarshal(result.Data, &resp); err != nil {
			return err
		}
		e.Websocket.DataHandler <- resp
		return nil
	case channelFuturesFundingRate:
		return e.processFuturesFundingRate(result.Data)
	case channelFuturesPrivatePositions:
		var resp []*FuturesPosition
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
		channel, interval, err := channelToIntervalSplit(result.Channel)
		if err != nil {
			return err
		}
		switch channel {
		case channelFuturesMarkPriceCandles, channelFuturesMarkCandles, channelFuturesIndexCandles:
			return e.processFuturesMarkAndIndexPriceCandlesticks(result.Data, interval)
		case channelFuturesCandles:
			return e.processFuturesCandlesticks(result.Data, interval)
		}
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", e.Name, string(respRaw))
	}
}

func (e *Exchange) processFuturesAccountData(data []byte) error {
	var resp []*FuturesAccountBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	var accChanges []accounts.Change
	for i := range resp {
		for j := range resp[i].Details {
			accChanges = append(accChanges, accounts.Change{
				AssetType: asset.Futures,
				Balance: accounts.Balance{
					Currency:  resp[i].Details[j].Currency,
					Total:     resp[i].Details[j].Available.Float64(),
					Hold:      resp[i].Details[j].TrdHold.Float64(),
					Free:      resp[i].Details[j].Available.Float64() - resp[i].Details[j].TrdHold.Float64(),
					UpdatedAt: resp[i].Details[j].UpdateTime.Time(),
				},
			})
		}
	}
	e.Websocket.DataHandler <- accChanges
	return nil
}

func (e *Exchange) processFuturesTradeFills(data []byte) error {
	var resp []*FuturesTradeFill
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tFills := make([]fill.Data, len(resp))
	for i := range resp {
		oSide, err := order.StringToOrderSide(resp[i].Side)
		if err != nil {
			return err
		}
		tFills[i] = fill.Data{
			Side:          oSide,
			Exchange:      e.Name,
			AssetType:     asset.Futures,
			CurrencyPair:  resp[i].Symbol,
			OrderID:       resp[i].OrderID,
			ID:            resp[i].TradeID,
			TradeID:       resp[i].TradeID,
			ClientOrderID: resp[i].ClientOrderID,
			Timestamp:     resp[i].UpdateTime.Time(),
			Price:         resp[i].FillPrice.Float64(),
			Amount:        resp[i].FillQuantity.Float64(),
		}
	}
	e.Websocket.DataHandler <- tFills
	return nil
}

func (e *Exchange) processFuturesOrders(data []byte) error {
	var resp []*FuturesOrderDetails
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	orders := make([]order.Detail, len(resp))
	for i := range resp {
		oType, err := order.StringToOrderType(resp[i].OrderType)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp[i].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(resp[i].State)
		if err != nil {
			return err
		}
		orders[i] = order.Detail{
			ReduceOnly:           resp[i].ReduceOnly,
			Leverage:             resp[i].Leverage.Float64(),
			Price:                resp[i].Price.Float64(),
			Amount:               resp[i].Size.Float64(),
			TriggerPrice:         resp[i].TakeProfitTriggerPrice.Float64(),
			AverageExecutedPrice: resp[i].AveragePrice.Float64(),
			ExecutedAmount:       resp[i].ExecQuantity.Float64(),
			RemainingAmount:      resp[i].Size.Float64() - resp[i].ExecQuantity.Float64(),
			Fee:                  resp[i].FeeAmount.Float64(),
			FeeAsset:             resp[i].FeeCurrency,
			Exchange:             e.Name,
			OrderID:              resp[i].OrderID,
			ClientOrderID:        resp[i].ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 resp[i].CreationTime.Time(),
			Pair:                 resp[i].Symbol,
		}
	}
	e.Websocket.DataHandler <- orders
	return nil
}

func (e *Exchange) processFuturesFundingRate(data []byte) error {
	var resp []*FuturesFundingRate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	for i := range resp {
		e.Websocket.DataHandler <- websocket.FundingData{
			CurrencyPair: resp[i].Symbol,
			Timestamp:    resp[i].Timestamp.Time(),
			AssetType:    asset.Futures,
			Exchange:     e.Name,
			Rate:         resp[i].FundingRate.Float64(),
		}
	}
	return nil
}

func (e *Exchange) processFuturesMarkAndIndexPriceCandlesticks(data []byte, interval kline.Interval) error {
	var resp []*WsFuturesMarkAndIndexPriceCandle
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for i := range resp {
		candles[i] = websocket.KlineData{
			Timestamp:  resp[i].PushTimestamp.Time(),
			Pair:       resp[i].Symbol,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  resp[i].StartTime.Time(),
			CloseTime:  resp[i].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[i].OpeningPrice.Float64(),
			ClosePrice: resp[i].ClosingPrice.Float64(),
			HighPrice:  resp[i].HighestPrice.Float64(),
			LowPrice:   resp[i].LowestPrice.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) processFuturesOrderbook(data []byte, action string) error {
	var resp []*WSFuturesOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	for i := range resp {
		_, okay := onceFuturesOrderbook[resp[i].Symbol.String()]
		if !okay || action == "snapshot" {
			if onceFuturesOrderbook == nil {
				onceFuturesOrderbook = make(map[string]bool)
			}
			onceFuturesOrderbook[resp[i].Symbol.String()] = true
			if err := e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Bids:         resp[i].Bids.Levels(),
				Asks:         resp[i].Asks.Levels(),
				Exchange:     e.Name,
				LastUpdateID: resp[i].ID,
				Asset:        asset.Futures,
				Pair:         resp[i].Symbol,
				LastUpdated:  resp[i].CreationTime.Time(),
			}); err != nil {
				return err
			}
			continue
		}
		if err := e.Websocket.Orderbook.Update(&orderbook.Update{
			UpdateID:   resp[i].ID,
			UpdateTime: resp[i].CreationTime.Time(),
			LastPushed: resp[i].Timestamp.Time(),
			Action:     orderbook.UpdateAction,
			Asset:      asset.Futures,
			Pair:       resp[i].Symbol,
			Asks:       resp[i].Asks.Levels(),
			Bids:       resp[i].Bids.Levels(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesTickers(data []byte) error {
	var resp []*FuturesTickerDetails
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tickerPrices := make([]ticker.Price, len(resp))
	for i := range resp {
		tickerPrices[i] = ticker.Price{
			High:         resp[i].HighPrice.Float64(),
			Low:          resp[i].LowPrice.Float64(),
			Bid:          resp[i].BestBidPrice.Float64(),
			BidSize:      resp[i].BestBidSize.Float64(),
			Ask:          resp[i].BestAskPrice.Float64(),
			AskSize:      resp[i].BestAskSize.Float64(),
			Volume:       resp[i].Quantity.Float64(),
			QuoteVolume:  resp[i].Amount.Float64(),
			Open:         resp[i].OpeningPrice.Float64(),
			Close:        resp[i].ClosingPrice.Float64(),
			MarkPrice:    resp[i].MarkPrice.Float64(),
			Pair:         resp[i].Symbol,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  resp[i].Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- tickerPrices
	return nil
}

// processFuturesTrades handles latest trading data for this product, including the latest price, trading volume, trading direction, etc.
func (e *Exchange) processFuturesTrades(data []byte) error {
	var resp []*FuturesTrades
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp))
	for i := range resp {
		oSide, err := order.StringToOrderSide(resp[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			TID:          strconv.FormatInt(resp[i].ID, 10),
			Exchange:     e.Name,
			Side:         oSide,
			AssetType:    asset.Futures,
			CurrencyPair: resp[i].Symbol,
			Price:        resp[i].Price.Float64(),
			Amount:       resp[i].Quantity.Float64(),
			Timestamp:    resp[i].Timestamp.Time(),
		}
	}
	e.Websocket.DataHandler <- trades
	return nil
}

func (e *Exchange) processFuturesCandlesticks(data []byte, interval kline.Interval) error {
	var resp []*WsFuturesCandlesctick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	candles := make([]websocket.KlineData, len(resp))
	for i := range resp {
		candles[i] = websocket.KlineData{
			Timestamp:  resp[i].PushTime.Time(),
			Pair:       resp[i].Symbol,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  resp[i].StartTime.Time(),
			CloseTime:  resp[i].EndTime.Time(),
			Interval:   interval.String(),
			OpenPrice:  resp[i].OpenPrice.Float64(),
			ClosePrice: resp[i].ClosePrice.Float64(),
			HighPrice:  resp[i].HighestPrice.Float64(),
			LowPrice:   resp[i].LowestPrice.Float64(),
			Volume:     resp[i].Amount.Float64(),
		}
	}
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) handleFuturesSubscriptions(operation string, subscs subscription.List) []*SubscriptionPayload {
	payloads := make([]*SubscriptionPayload, len(subscs))
	for i := range subscs {
		input := &SubscriptionPayload{
			Event:   operation,
			Channel: []string{subscs[i].QualifiedChannel},
		}
		if len(subscs[i].Pairs) != 0 && subscs[i].QualifiedChannel != channelFuturesAccount {
			input.Symbols = subscs[i].Pairs.Strings()
		}
		payloads[i] = input
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
