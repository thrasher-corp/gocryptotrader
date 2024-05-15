// Package gemini exchange documentation can be found at
// https://docs.sandbox.gemini.com
package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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

// Instantiates a communications channel between websocket connections
var comms = make(chan stream.Response)

// WsConnect initiates a websocket connection
func (g *Gemini) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}

	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	g.Websocket.Wg.Add(2)
	go g.wsReadData()
	go g.wsFunnelConnectionData(g.Websocket.Conn)

	if g.Websocket.CanUseAuthenticatedEndpoints() {
		err := g.WsAuth(context.TODO(), &dialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - websocket authentication failed: %v\n", g.Name, err)
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (g *Gemini) GenerateDefaultSubscriptions() ([]subscription.Subscription, error) {
	// See gemini_types.go for more subscription/candle vars
	var channels = []string{
		marketDataLevel2,
		candles1d,
	}

	pairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	var subscriptions []subscription.Subscription
	for x := range channels {
		for y := range pairs {
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: channels[x],
				Pair:    pairs[y],
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (g *Gemini) Subscribe(channelsToSubscribe []subscription.Subscription) error {
	channels := make([]string, 0, len(channelsToSubscribe))
	for x := range channelsToSubscribe {
		if common.StringDataCompareInsensitive(channels, channelsToSubscribe[x].Channel) {
			continue
		}
		channels = append(channels, channelsToSubscribe[x].Channel)
	}

	var pairs currency.Pairs
	for x := range channelsToSubscribe {
		if pairs.Contains(channelsToSubscribe[x].Pair, true) {
			continue
		}
		pairs = append(pairs, channelsToSubscribe[x].Pair)
	}

	fmtPairs, err := g.FormatExchangeCurrencies(pairs, asset.Spot)
	if err != nil {
		return err
	}

	subs := make([]wsSubscriptions, len(channels))
	for x := range channels {
		subs[x] = wsSubscriptions{
			Name:    channels[x],
			Symbols: strings.Split(fmtPairs, ","),
		}
	}

	wsSub := wsSubscribeRequest{
		Type:          "subscribe",
		Subscriptions: subs,
	}
	err = g.Websocket.Conn.SendJSONMessage(wsSub)
	if err != nil {
		return err
	}

	g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gemini) Unsubscribe(channelsToUnsubscribe []subscription.Subscription) error {
	channels := make([]string, 0, len(channelsToUnsubscribe))
	for x := range channelsToUnsubscribe {
		if common.StringDataCompareInsensitive(channels, channelsToUnsubscribe[x].Channel) {
			continue
		}
		channels = append(channels, channelsToUnsubscribe[x].Channel)
	}

	var pairs currency.Pairs
	for x := range channelsToUnsubscribe {
		if pairs.Contains(channelsToUnsubscribe[x].Pair, true) {
			continue
		}
		pairs = append(pairs, channelsToUnsubscribe[x].Pair)
	}

	fmtPairs, err := g.FormatExchangeCurrencies(pairs, asset.Spot)
	if err != nil {
		return err
	}

	subs := make([]wsSubscriptions, len(channels))
	for x := range channels {
		subs[x] = wsSubscriptions{
			Name:    channels[x],
			Symbols: strings.Split(fmtPairs, ","),
		}
	}

	wsSub := wsSubscribeRequest{
		Type:          "unsubscribe",
		Subscriptions: subs,
	}
	err = g.Websocket.Conn.SendJSONMessage(wsSub)
	if err != nil {
		return err
	}

	g.Websocket.RemoveSubscriptions(channelsToUnsubscribe...)
	return nil
}

// WsAuth will connect to Gemini's secure endpoint
func (g *Gemini) WsAuth(ctx context.Context, dialer *websocket.Dialer) error {
	if !g.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", g.Name)
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	payload := WsRequestPayload{
		Request: "/v1/" + geminiWsOrderEvents,
		Nonce:   time.Now().UnixNano(),
	}
	PayloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%v sendAuthenticatedHTTPRequest: Unable to JSON request", g.Name)
	}
	wsEndpoint, err := g.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	endpoint := wsEndpoint + geminiWsOrderEvents
	PayloadBase64 := crypto.Base64Encode(PayloadJSON)
	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384,
		[]byte(PayloadBase64),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Add("Content-Length", "0")
	headers.Add("Content-Type", "text/plain")
	headers.Add("X-GEMINI-PAYLOAD", PayloadBase64)
	headers.Add("X-GEMINI-APIKEY", creds.Key)
	headers.Add("X-GEMINI-SIGNATURE", crypto.HexEncodeToString(hmac))
	headers.Add("Cache-Control", "no-cache")

	err = g.Websocket.AuthConn.Dial(dialer, headers)
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", g.Name, endpoint, err)
	}
	go g.wsFunnelConnectionData(g.Websocket.AuthConn)
	return nil
}

// wsFunnelConnectionData receives data from multiple connections and passes it to wsReadData
func (g *Gemini) wsFunnelConnectionData(ws stream.Connection) {
	defer g.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- stream.Response{Raw: resp.Raw}
	}
}

// wsReadData receives and passes on websocket messages for processing
func (g *Gemini) wsReadData() {
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := g.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case g.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr,
							"%s websocket handle data error: %v",
							g.Name,
							err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			err := g.wsHandleData(resp.Raw)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

func (g *Gemini) wsHandleData(respRaw []byte) error {
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
				g.Websocket.DataHandler <- order.ClassificationError{
					Exchange: g.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = stringToOrderType(result[i].OrderType)
			if err != nil {
				g.Websocket.DataHandler <- order.ClassificationError{
					Exchange: g.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = stringToOrderStatus(result[i].Type)
			if err != nil {
				g.Websocket.DataHandler <- order.ClassificationError{
					Exchange: g.Name,
					OrderID:  result[i].OrderID,
					Err:      err,
				}
			}

			enabledPairs, err := g.GetAvailablePairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := g.GetPairFormat(asset.Spot, true)
			if err != nil {
				return err
			}

			pair, err := currency.NewPairFromFormattedPairs(result[i].Symbol, enabledPairs, format)
			if err != nil {
				return err
			}

			g.Websocket.DataHandler <- &order.Detail{
				HiddenOrder:     result[i].IsHidden,
				Price:           result[i].Price,
				Amount:          result[i].OriginalAmount,
				ExecutedAmount:  result[i].ExecutedAmount,
				RemainingAmount: result[i].RemainingAmount,
				Exchange:        g.Name,
				OrderID:         result[i].OrderID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       asset.Spot,
				Date:            time.UnixMilli(result[i].Timestampms),
				Pair:            pair,
			}
		}
		return nil
	}
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return fmt.Errorf("%v Error: %v, Raw: %v", g.Name, err, string(respRaw))
	}
	if _, ok := result["type"]; ok {
		switch result["type"] {
		case "l2_updates":
			var l2MarketData *wsL2MarketData
			err := json.Unmarshal(respRaw, &l2MarketData)
			if err != nil {
				return err
			}
			return g.wsProcessUpdate(l2MarketData)
		case "trade":
			if !g.IsSaveTradeDataEnabled() {
				return nil
			}

			var result wsTrade
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}

			tSide, err := order.StringToOrderSide(result.Side)
			if err != nil {
				g.Websocket.DataHandler <- order.ClassificationError{
					Exchange: g.Name,
					Err:      err,
				}
			}

			enabledPairs, err := g.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := g.GetPairFormat(asset.Spot, true)
			if err != nil {
				return err
			}

			pair, err := currency.NewPairFromFormattedPairs(result.Symbol, enabledPairs, format)
			if err != nil {
				return err
			}

			tradeEvent := trade.Data{
				Timestamp:    time.UnixMilli(result.Timestamp),
				CurrencyPair: pair,
				AssetType:    asset.Spot,
				Exchange:     g.Name,
				Price:        result.Price,
				Amount:       result.Quantity,
				Side:         tSide,
				TID:          strconv.FormatInt(result.EventID, 10),
			}

			return trade.AddTradesToBuffer(g.Name, tradeEvent)
		case "subscription_ack":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
		case "initial":
			var result WsSubscriptionAcknowledgementResponse
			err := json.Unmarshal(respRaw, &result)
			if err != nil {
				return err
			}
			g.Websocket.DataHandler <- result
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
			enabledPairs, err := g.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}

			format, err := g.GetPairFormat(asset.Spot, true)
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
				g.Websocket.DataHandler <- stream.KlineData{
					Timestamp:  time.UnixMilli(int64(candle.Changes[i][0])),
					Pair:       pair,
					AssetType:  asset.Spot,
					Exchange:   g.Name,
					Interval:   interval,
					OpenPrice:  candle.Changes[i][1],
					HighPrice:  candle.Changes[i][2],
					LowPrice:   candle.Changes[i][3],
					ClosePrice: candle.Changes[i][4],
					Volume:     candle.Changes[i][5],
				}
			}
		default:
			g.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: g.Name + stream.UnhandledMessage + string(respRaw)}
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
			return fmt.Errorf("%v Unhandled websocket error %s", g.Name, respRaw)
		default:
			g.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: g.Name + stream.UnhandledMessage + string(respRaw)}
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

func (g *Gemini) wsProcessUpdate(result *wsL2MarketData) error {
	isInitial := len(result.Changes) > 0 && len(result.Trades) > 0
	enabledPairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := g.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	pair, err := currency.NewPairFromFormattedPairs(result.Symbol, enabledPairs, format)
	if err != nil {
		return err
	}

	bids := make([]orderbook.Tranche, 0, len(result.Changes))
	asks := make([]orderbook.Tranche, 0, len(result.Changes))

	for x := range result.Changes {
		price, err := strconv.ParseFloat(result.Changes[x][1], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(result.Changes[x][2], 64)
		if err != nil {
			return err
		}
		obItem := orderbook.Tranche{
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
		var newOrderBook orderbook.Base
		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.Asset = asset.Spot
		newOrderBook.Pair = pair
		newOrderBook.Exchange = g.Name
		newOrderBook.VerifyOrderbook = g.CanVerifyOrderbook
		newOrderBook.LastUpdated = time.Now() // No time is sent
		err := g.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
		if err != nil {
			return err
		}
	} else {
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}
		err := g.Websocket.Orderbook.Update(&orderbook.Update{
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
		g.Websocket.DataHandler <- result.AuctionEvents
	}

	if !g.IsSaveTradeDataEnabled() {
		return nil
	}

	trades := make([]trade.Data, len(result.Trades))
	for x := range result.Trades {
		tSide, err := order.StringToOrderSide(result.Trades[x].Side)
		if err != nil {
			g.Websocket.DataHandler <- order.ClassificationError{
				Exchange: g.Name,
				Err:      err,
			}
		}
		trades[x] = trade.Data{
			Timestamp:    time.UnixMilli(result.Trades[x].Timestamp),
			CurrencyPair: pair,
			AssetType:    asset.Spot,
			Exchange:     g.Name,
			Price:        result.Trades[x].Price,
			Amount:       result.Trades[x].Quantity,
			Side:         tSide,
			TID:          strconv.FormatInt(result.Trades[x].EventID, 10),
		}
	}

	return trade.AddTradesToBuffer(g.Name, trades...)
}
