// Package gemini exchange documentation can be found at
// https://docs.sandbox.gemini.com
package gemini

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	geminiWebsocketEndpoint        = "wss://api.gemini.com"
	geminiWebsocketSandboxEndpoint = "wss://api.sandbox.gemini.com/v1/"
	geminiWsMarketData             = "marketdata"
	geminiWsOrderEvents            = "order/events"
)

const (
	marketDataLevel2 = "l2"
	candlesChannel   = "candles"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneDay},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	// Authenticated connection is directly to the orders URI, so this is implicit
	// {Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.CandlesChannel:   candlesChannel,
	subscription.OrderbookChannel: marketDataLevel2,
}

// Instantiates a communications channel between websocket connections
var comms = make(chan websocket.Response)

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}

	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}

	e.Websocket.Wg.Add(2)
	go e.wsReadData()
	go e.wsFunnelConnectionData(e.Websocket.Conn)

	if e.Websocket.CanUseAuthenticatedEndpoints() {
		err := e.WsAuth(ctx, &dialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - websocket authentication failed: %v\n", e.Name, err)
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName": channelName,
		"interval":    channelInterval,
	}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubs(ctx, subs, wsSubscribeOp)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	return e.manageSubs(ctx, subs, wsUnsubscribeOp)
}

func (e *Exchange) manageSubs(ctx context.Context, subs subscription.List, op wsSubOp) error {
	req := wsSubscribeRequest{
		Type:          op,
		Subscriptions: make([]wsSubscriptions, 0, len(subs)),
	}
	for _, s := range subs {
		req.Subscriptions = append(req.Subscriptions, wsSubscriptions{
			Name:    s.QualifiedChannel,
			Symbols: s.Pairs.Strings(),
		})
	}

	if err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req); err != nil {
		return err
	}

	if op == wsUnsubscribeOp {
		return e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subs...)
	}

	return e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, subs...)
}

// WsAuth will connect to Gemini's secure endpoint
func (e *Exchange) WsAuth(ctx context.Context, dialer *gws.Dialer) error {
	if !e.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", e.Name)
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	payload := WsRequestPayload{
		Request: "/v1/" + geminiWsOrderEvents,
		Nonce:   time.Now().UnixNano(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%v sendAuthenticatedHTTPRequest: Unable to JSON request", e.Name)
	}
	wsEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	endpoint := wsEndpoint + geminiWsOrderEvents
	payloadB64 := base64.StdEncoding.EncodeToString(payloadJSON)
	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(payloadB64), []byte(creds.Secret))
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Add("Content-Length", "0")
	headers.Add("Content-Type", "text/plain")
	headers.Add("X-GEMINI-PAYLOAD", payloadB64)
	headers.Add("X-GEMINI-APIKEY", creds.Key)
	headers.Add("X-GEMINI-SIGNATURE", hex.EncodeToString(hmac))
	headers.Add("Cache-Control", "no-cache")

	err = e.Websocket.AuthConn.Dial(ctx, dialer, headers)
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", e.Name, endpoint, err)
	}
	e.Websocket.Wg.Add(1)
	go e.wsFunnelConnectionData(e.Websocket.AuthConn)
	return nil
}

// wsFunnelConnectionData receives data from multiple connections and passes it to wsReadData
func (e *Exchange) wsFunnelConnectionData(ws websocket.Connection) {
	defer e.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- websocket.Response{Raw: resp.Raw}
	}
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData() {
	defer e.Websocket.Wg.Done()
	for {
		select {
		case <-e.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := e.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case e.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr,
							"%s websocket handle data error: %v",
							e.Name,
							err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			err := e.wsHandleData(resp.Raw)
			if err != nil {
				e.Websocket.DataHandler <- err
			}
		}
	}
}

func (e *Exchange) wsHandleData(respRaw []byte) error {
	// only order details are sent in arrays
	if strings.HasPrefix(string(respRaw), "[") {
		var result []WsOrderResponse
		err := json.Unmarshal(respRaw, &result)
		if err != nil {
			return err
		}

		for i := range result {
			oSide, err := order.StringToOrderSide(result[i].Side)
			if err != nil {
				e.Websocket.DataHandler <- order.ClassificationError{
					Exchange: e.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = stringToOrderType(result[i].OrderType)
			if err != nil {
				e.Websocket.DataHandler <- order.ClassificationError{
					Exchange: e.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = stringToOrderStatus(result[i].Type)
			if err != nil {
				e.Websocket.DataHandler <- order.ClassificationError{
					Exchange: e.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}

			enabledPairs, err := e.GetAvailablePairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := e.GetPairFormat(asset.Spot, true)
			if err != nil {
				return err
			}

			pair, err := currency.NewPairFromFormattedPairs(result[i].Symbol, enabledPairs, format)
			if err != nil {
				return err
			}

			e.Websocket.DataHandler <- &order.Detail{
				HiddenOrder:     result[i].IsHidden,
				Price:           result[i].Price,
				Amount:          result[i].OriginalAmount,
				ExecutedAmount:  result[i].ExecutedAmount,
				RemainingAmount: result[i].RemainingAmount,
				Exchange:        e.Name,
				OrderID:         result[i].OrderID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       asset.Spot,
				Date:            result[i].TimestampMS.Time(),
				Pair:            pair,
			}
		}
		return nil
	}
	var result map[string]any
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return fmt.Errorf("%v Error: %v, Raw: %v", e.Name, err, string(respRaw))
	}
	if _, ok := result["type"]; ok {
		switch result["type"] {
		case "l2_updates":
			var l2MarketData *wsL2MarketData
			err := json.Unmarshal(respRaw, &l2MarketData)
			if err != nil {
				return err
			}
			return e.wsProcessUpdate(l2MarketData)
		case "trade":
			if !e.IsSaveTradeDataEnabled() {
				return nil
			}

			var result wsTrade
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}

			tSide, err := order.StringToOrderSide(result.Side)
			if err != nil {
				e.Websocket.DataHandler <- order.ClassificationError{
					Exchange: e.Name,
					Err:      err,
				}
			}

			enabledPairs, err := e.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := e.GetPairFormat(asset.Spot, true)
			if err != nil {
				return err
			}

			pair, err := currency.NewPairFromFormattedPairs(result.Symbol, enabledPairs, format)
			if err != nil {
				return err
			}

			tradeEvent := trade.Data{
				Timestamp:    result.Timestamp.Time(),
				CurrencyPair: pair,
				AssetType:    asset.Spot,
				Exchange:     e.Name,
				Price:        result.Price,
				Amount:       result.Quantity,
				Side:         tSide,
				TID:          strconv.FormatInt(result.EventID, 10),
			}

			return trade.AddTradesToBuffer(tradeEvent)
		case "subscription_ack":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- result
		case "initial":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- result
		case "heartbeat":
			return nil
		case "candles_1m_updates",
			"candles_5m_updates",
			"candles_15m_updates",
			"candles_30m_updates",
			"candles_1h_updates",
			"candles_6h_updates",
			"candles_1d_updates":
			var candle wsCandleResponse
			err := json.Unmarshal(respRaw, &candle)
			if err != nil {
				return err
			}
			enabledPairs, err := e.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := e.GetPairFormat(asset.Spot, true)
			if err != nil {
				return err
			}

			pair, err := currency.NewPairFromFormattedPairs(candle.Symbol, enabledPairs, format)
			if err != nil {
				return err
			}

			for i := range candle.Changes {
				if len(candle.Changes[i]) != 6 {
					continue
				}
				interval, ok := result["type"].(string)
				if !ok {
					return errors.New("unable to type assert interval")
				}
				e.Websocket.DataHandler <- websocket.KlineData{
					Timestamp:  time.UnixMilli(int64(candle.Changes[i][0])),
					Pair:       pair,
					AssetType:  asset.Spot,
					Exchange:   e.Name,
					Interval:   interval,
					OpenPrice:  candle.Changes[i][1],
					HighPrice:  candle.Changes[i][2],
					LowPrice:   candle.Changes[i][3],
					ClosePrice: candle.Changes[i][4],
					Volume:     candle.Changes[i][5],
				}
			}
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
			return nil
		}
	} else if r, ok := result["result"].(string); ok {
		switch r {
		case "error":
			if reason, ok := result["reason"].(string); ok {
				if msg, ok := result["message"].(string); ok {
					reason += " - " + msg
				}
				return errors.New(reason)
			}
			return fmt.Errorf("%v Unhandled websocket error %s", e.Name, respRaw)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
			return nil
		}
	}
	return nil
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "accepted":
		return order.New, nil
	case "booked":
		return order.Active, nil
	case "fill":
		return order.Filled, nil
	case "cancelled":
		return order.Cancelled, nil
	case "cancel_rejected":
		return order.Rejected, nil
	case "closed":
		return order.Filled, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func stringToOrderType(oType string) (order.Type, error) {
	switch oType {
	case "exchange limit", "auction-only limit", "indication-of-interest limit":
		return order.Limit, nil
	case "market buy", "market sell", "block_trade":
		// block trades are conducted off order-book, so their type is market,
		// but would be considered a hidden trade
		return order.Market, nil
	default:
		return order.UnknownType, errors.New(oType + " not recognised as order type")
	}
}

func (e *Exchange) wsProcessUpdate(result *wsL2MarketData) error {
	isInitial := len(result.Changes) > 0 && len(result.Trades) > 0
	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	pair, err := currency.NewPairFromFormattedPairs(result.Symbol, enabledPairs, format)
	if err != nil {
		return err
	}

	bids := make([]orderbook.Level, 0, len(result.Changes))
	asks := make([]orderbook.Level, 0, len(result.Changes))

	for x := range result.Changes {
		price, err := strconv.ParseFloat(result.Changes[x][1], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(result.Changes[x][2], 64)
		if err != nil {
			return err
		}
		obItem := orderbook.Level{
			Amount: amount,
			Price:  price,
		}
		if result.Changes[x][0] == "buy" {
			bids = append(bids, obItem)
			continue
		}
		asks = append(asks, obItem)
	}

	if isInitial {
		var newOrderBook orderbook.Book
		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.Asset = asset.Spot
		newOrderBook.Pair = pair
		newOrderBook.Exchange = e.Name
		newOrderBook.ValidateOrderbook = e.ValidateOrderbook
		newOrderBook.LastUpdated = time.Now() // No time is sent
		err := e.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
		if err != nil {
			return err
		}
	} else {
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}
		err := e.Websocket.Orderbook.Update(&orderbook.Update{
			Asks:       asks,
			Bids:       bids,
			Pair:       pair,
			Asset:      asset.Spot,
			UpdateTime: time.Now(), // No time is sent
		})
		if err != nil {
			return err
		}
	}

	if len(result.AuctionEvents) > 0 {
		e.Websocket.DataHandler <- result.AuctionEvents
	}

	if !e.IsSaveTradeDataEnabled() {
		return nil
	}

	trades := make([]trade.Data, len(result.Trades))
	for x := range result.Trades {
		tSide, err := order.StringToOrderSide(result.Trades[x].Side)
		if err != nil {
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				Err:      err,
			}
		}
		trades[x] = trade.Data{
			Timestamp:    result.Trades[x].Timestamp.Time(),
			CurrencyPair: pair,
			AssetType:    asset.Spot,
			Exchange:     e.Name,
			Price:        result.Trades[x].Price,
			Amount:       result.Trades[x].Quantity,
			Side:         tSide,
			TID:          strconv.FormatInt(result.Trades[x].EventID, 10),
		}
	}

	return trade.AddTradesToBuffer(trades...)
}

func channelName(s *subscription.Subscription) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return n
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

func channelInterval(i kline.Interval) string {
	switch i {
	case kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.SixHour:
		return i.Short()
	case kline.OneDay:
		return "1d"
	}
	panic(fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, i.Short()))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- with $i := $.S.Interval -}} _ {{- interval $i }}{{ end -}}
	{{- $.AssetSeparator }}
{{- end }}
`
