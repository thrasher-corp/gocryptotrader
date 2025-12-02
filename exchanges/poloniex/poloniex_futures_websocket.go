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
	for _, r := range resp {
		for _, detail := range r.Details {
			accChanges = append(accChanges, accounts.Change{
				AssetType: asset.Futures,
				Balance: accounts.Balance{
					Currency:  detail.Currency,
					Total:     detail.Available.Float64(),
					Hold:      detail.TrdHold.Float64(),
					Free:      detail.Available.Float64() - detail.TrdHold.Float64(),
					UpdatedAt: detail.UpdateTime.Time(),
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
			Price:         r.FillPrice.Float64(),
			Amount:        r.FillQuantity.Float64(),
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
	for i, r := range resp {
		oStatus, err := order.StringToOrderStatus(r.State)
		if err != nil {
			return err
		}
		orders[i] = order.Detail{
			ReduceOnly:           r.ReduceOnly,
			Leverage:             r.Leverage.Float64(),
			Price:                r.Price.Float64(),
			Amount:               r.Size.Float64(),
			TriggerPrice:         r.TakeProfitTriggerPrice.Float64(),
			AverageExecutedPrice: r.AveragePrice.Float64(),
			ExecutedAmount:       r.ExecQuantity.Float64(),
			RemainingAmount:      r.Size.Float64() - r.ExecQuantity.Float64(),
			Fee:                  r.FeeAmount.Float64(),
			FeeAsset:             r.FeeCurrency,
			Exchange:             e.Name,
			OrderID:              r.OrderID,
			ClientOrderID:        r.ClientOrderID,
			Type:                 r.OrderType,
			Side:                 r.Side,
			Status:               oStatus,
			AssetType:            asset.Futures,
			Date:                 r.CreationTime.Time(),
			Pair:                 r.Symbol,
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

	for _, r := range resp {
		e.Websocket.DataHandler <- websocket.FundingData{
			CurrencyPair: r.Symbol,
			Timestamp:    r.Timestamp.Time(),
			AssetType:    asset.Futures,
			Exchange:     e.Name,
			Rate:         r.FundingRate.Float64(),
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
	e.Websocket.DataHandler <- candles
	return nil
}

func (e *Exchange) processFuturesOrderbook(data []byte, action string) error {
	var resp []*WSFuturesOrderbook
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	for _, r := range resp {
		_, okay := onceFuturesOrderbook[r.Symbol.String()]
		if !okay || action == "snapshot" {
			if onceFuturesOrderbook == nil {
				onceFuturesOrderbook = make(map[string]bool)
			}
			onceFuturesOrderbook[r.Symbol.String()] = true
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

func (e *Exchange) processFuturesTickers(data []byte) error {
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
			Volume:       r.Quantity.Float64(),
			QuoteVolume:  r.Amount.Float64(),
			Open:         r.OpeningPrice.Float64(),
			Close:        r.ClosingPrice.Float64(),
			MarkPrice:    r.MarkPrice.Float64(),
			Pair:         r.Symbol,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  r.Timestamp.Time(),
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
			Amount:       r.Quantity.Float64(),
			Timestamp:    r.Timestamp.Time(),
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
			Volume:     r.Amount.Float64(),
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
	for _, payload := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payload); err != nil {
			return err
		}
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (e *Exchange) UnsubscribeFutures(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
	payloads := e.handleFuturesSubscriptions("unsubscribe", unsub)
	for _, payload := range payloads {
		if err := conn.SendJSONMessage(ctx, request.UnAuth, payload); err != nil {
			return err
		}
	}
	return e.Websocket.RemoveSubscriptions(conn, unsub...)
}
