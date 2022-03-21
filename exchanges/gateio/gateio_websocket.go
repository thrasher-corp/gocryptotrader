package gateio

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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	gateioWebsocketEndpoint  = "wss://ws.gateio.ws/v3/"
	gateioWebsocketRateLimit = 120
)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	g.Websocket.Wg.Add(1)
	go g.wsReadData()

	if g.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = g.wsServerSignIn(context.TODO())
		if err != nil {
			g.Websocket.DataHandler <- err
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		} else {
			var authsubs []stream.ChannelSubscription
			authsubs, err = g.GenerateAuthenticatedSubscriptions()
			if err != nil {
				g.Websocket.DataHandler <- err
				g.Websocket.SetCanUseAuthenticatedEndpoints(false)
			} else {
				err = g.Websocket.SubscribeToChannels(authsubs)
				if err != nil {
					g.Websocket.DataHandler <- err
					g.Websocket.SetCanUseAuthenticatedEndpoints(false)
				}
			}
		}
	}

	return nil
}

func (g *Gateio) wsServerSignIn(ctx context.Context) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	nonce := int(time.Now().Unix() * 1000)
	sigTemp, err := g.GenerateSignature(creds.Secret, strconv.Itoa(nonce))
	if err != nil {
		return err
	}
	signature := crypto.Base64Encode(sigTemp)
	signinWsRequest := WebsocketRequest{
		ID:     g.Websocket.Conn.GenerateMessageID(false),
		Method: "server.sign",
		Params: []interface{}{creds.Key, signature, nonce},
	}
	resp, err := g.Websocket.Conn.SendMessageReturnResponse(signinWsRequest.ID,
		signinWsRequest)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	var response WebsocketAuthenticationResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	if response.Result.Status == "success" {
		g.Websocket.SetCanUseAuthenticatedEndpoints(true)
		return nil
	}

	return fmt.Errorf("%s cannot authenticate websocket connection: %s",
		g.Name,
		response.Result.Status)
}

// wsReadData receives and passes on websocket messages for processing
func (g *Gateio) wsReadData() {
	defer g.Websocket.Wg.Done()

	for {
		resp := g.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := g.wsHandleData(resp.Raw)
		if err != nil {
			g.Websocket.DataHandler <- err
		}
	}
}

func (g *Gateio) wsHandleData(respRaw []byte) error {
	var result WebsocketResponse
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}

	if result.ID > 0 {
		if g.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return nil
		}
	}

	if result.Error.Code != 0 {
		if strings.Contains(result.Error.Message, "authentication") {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
			return fmt.Errorf("%v - authentication failed: %v", g.Name, err)
		}
		return fmt.Errorf("%v error %s", g.Name, result.Error.Message)
	}

	switch {
	case strings.Contains(result.Method, "ticker"):
		var wsTicker WebsocketTicker
		var c string
		err = json.Unmarshal(result.Params[1], &wsTicker)
		if err != nil {
			return err
		}
		err = json.Unmarshal(result.Params[0], &c)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = currency.NewPairFromString(c)
		if err != nil {
			return err
		}

		g.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: g.Name,
			Open:         wsTicker.Open,
			Close:        wsTicker.Close,
			Volume:       wsTicker.BaseVolume,
			QuoteVolume:  wsTicker.QuoteVolume,
			High:         wsTicker.High,
			Low:          wsTicker.Low,
			Last:         wsTicker.Last,
			AssetType:    asset.Spot,
			Pair:         p,
		}

	case strings.Contains(result.Method, "trades"):
		if !g.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeData []WebsocketTrade
		var c string
		err = json.Unmarshal(result.Params[1], &tradeData)
		if err != nil {
			return err
		}
		err = json.Unmarshal(result.Params[0], &c)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = currency.NewPairFromString(c)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range tradeData {
			var tSide order.Side
			tSide, err = order.StringToOrderSide(tradeData[i].Type)
			if err != nil {
				g.Websocket.DataHandler <- order.ClassificationError{
					Exchange: g.Name,
					Err:      err,
				}
			}
			trades = append(trades, trade.Data{
				Timestamp:    convert.TimeFromUnixTimestampDecimal(tradeData[i].Time),
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Exchange:     g.Name,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Side:         tSide,
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
			})
		}
		return trade.AddTradesToBuffer(g.Name, trades...)
	case strings.Contains(result.Method, "balance.update"):
		var balance wsBalanceSubscription
		err = json.Unmarshal(respRaw, &balance)
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- balance
	case strings.Contains(result.Method, "order.update"):
		var orderUpdate wsOrderUpdate
		err = json.Unmarshal(respRaw, &orderUpdate)
		if err != nil {
			return err
		}
		if len(orderUpdate.Params) < 2 {
			return errors.New("unexpected orderUpdate.Params data length")
		}
		invalidJSON, ok := orderUpdate.Params[1].(map[string]interface{})
		if !ok {
			return errors.New("unable to type assert invalidJSON")
		}
		oStatus := order.UnknownStatus
		oType := order.UnknownType
		oSide := order.UnknownSide

		orderStatus, ok := orderUpdate.Params[0].(float64)
		if !ok {
			return errors.New("unable to type assert orderStatus")
		}
		switch orderStatus {
		case 1:
			oStatus = order.New
		case 2:
			oStatus = order.PartiallyFilled
		case 3:
			oStatus = order.Filled
		}

		orderType, ok := invalidJSON["orderType"].(float64)
		if !ok {
			return errors.New("unable to type assert orderType")
		}
		switch orderType {
		case 1:
			oType = order.Limit
		case 2:
			oType = order.Market
		}

		orderSide, ok := invalidJSON["type"].(float64)
		if !ok {
			return errors.New("unable to type assert orderSide")
		}
		switch orderSide {
		case 1:
			oSide = order.Sell
		case 2:
			oSide = order.Buy
		}

		var price, amount, filledTotal, left, fee float64
		price, err = convert.FloatFromString(invalidJSON["price"])
		if err != nil {
			return err
		}
		amount, err = convert.FloatFromString(invalidJSON["amount"])
		if err != nil {
			return err
		}
		filledTotal, err = convert.FloatFromString(invalidJSON["filledTotal"])
		if err != nil {
			return err
		}
		left, err = convert.FloatFromString(invalidJSON["left"])
		if err != nil {
			return err
		}
		fee, err = convert.FloatFromString(invalidJSON["dealFee"])
		if err != nil {
			return err
		}

		var p currency.Pair
		pairStr, ok := invalidJSON["market"].(string)
		if !ok {
			return errors.New("unable to type assert market")
		}
		p, err = currency.NewPairFromString(pairStr)
		if err != nil {
			return err
		}

		var a asset.Item
		a, err = g.GetPairAssetType(p)
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- &order.Detail{
			Price:           price,
			Amount:          amount,
			ExecutedAmount:  filledTotal,
			RemainingAmount: left,
			Fee:             fee,
			Exchange:        g.Name,
			ID:              strconv.FormatFloat(invalidJSON["id"].(float64), 'f', -1, 64),
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       a,
			Date:            convert.TimeFromUnixTimestampDecimal(invalidJSON["ctime"].(float64)),
			LastUpdated:     convert.TimeFromUnixTimestampDecimal(invalidJSON["mtime"].(float64)),
			Pair:            p,
		}
	case strings.Contains(result.Method, "depth"):
		var IsSnapshot bool
		var c string
		var data wsOrderbook

		err = json.Unmarshal(result.Params[0], &IsSnapshot)
		if err != nil {
			return err
		}

		err = json.Unmarshal(result.Params[2], &c)
		if err != nil {
			return err
		}

		err = json.Unmarshal(result.Params[1], &data)
		if err != nil {
			return err
		}

		var asks, bids []orderbook.Item
		var amount, price float64
		for i := range data.Asks {
			amount, err = strconv.ParseFloat(data.Asks[i][1], 64)
			if err != nil {
				return err
			}
			price, err = strconv.ParseFloat(data.Asks[i][0], 64)
			if err != nil {
				return err
			}
			asks = append(asks, orderbook.Item{Amount: amount, Price: price})
		}

		for i := range data.Bids {
			amount, err = strconv.ParseFloat(data.Bids[i][1], 64)
			if err != nil {
				return err
			}
			price, err = strconv.ParseFloat(data.Bids[i][0], 64)
			if err != nil {
				return err
			}
			bids = append(bids, orderbook.Item{Amount: amount, Price: price})
		}

		var p currency.Pair
		p, err = currency.NewPairFromString(c)
		if err != nil {
			return err
		}

		if IsSnapshot {
			var newOrderBook orderbook.Base
			newOrderBook.Asks = asks
			newOrderBook.Bids = bids
			newOrderBook.Asset = asset.Spot
			newOrderBook.Pair = p
			newOrderBook.Exchange = g.Name
			newOrderBook.VerifyOrderbook = g.CanVerifyOrderbook

			err = g.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
			if err != nil {
				return err
			}
		} else {
			err = g.Websocket.Orderbook.Update(&buffer.Update{
				Asks:       asks,
				Bids:       bids,
				Pair:       p,
				UpdateTime: time.Now(),
				Asset:      asset.Spot,
			})
			if err != nil {
				return err
			}
		}
	case strings.Contains(result.Method, "kline"):
		var data []interface{}
		err = json.Unmarshal(result.Params[0], &data)
		if err != nil {
			return err
		}
		open, err := strconv.ParseFloat(data[1].(string), 64)
		if err != nil {
			return err
		}
		closePrice, err := strconv.ParseFloat(data[2].(string), 64)
		if err != nil {
			return err
		}
		high, err := strconv.ParseFloat(data[3].(string), 64)
		if err != nil {
			return err
		}
		low, err := strconv.ParseFloat(data[4].(string), 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(data[5].(string), 64)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromString(data[7].(string))
		if err != nil {
			return err
		}

		g.Websocket.DataHandler <- stream.KlineData{
			Timestamp:  time.Now(),
			Pair:       p,
			AssetType:  asset.Spot,
			Exchange:   g.Name,
			OpenPrice:  open,
			ClosePrice: closePrice,
			HighPrice:  high,
			LowPrice:   low,
			Volume:     volume,
		}
	default:
		g.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: g.Name + stream.UnhandledMessage + string(respRaw),
		}
		return nil
	}
	return nil
}

// GenerateAuthenticatedSubscriptions returns authenticated subscriptions
func (g *Gateio) GenerateAuthenticatedSubscriptions() ([]stream.ChannelSubscription, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	var channels = []string{"balance.subscribe", "order.subscribe"}
	var subscriptions []stream.ChannelSubscription
	enabledCurrencies, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// GenerateDefaultSubscriptions returns default subscriptions
func (g *Gateio) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"ticker.subscribe",
		"trades.subscribe",
		"depth.subscribe",
		"kline.subscribe"}
	var subscriptions []stream.ChannelSubscription
	enabledCurrencies, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		for j := range enabledCurrencies {
			params := make(map[string]interface{})
			if strings.EqualFold(channels[i], "depth.subscribe") {
				params["limit"] = 30
				params["interval"] = "0.1"
			} else if strings.EqualFold(channels[i], "kline.subscribe") {
				params["interval"] = 1800
			}

			fpair, err := g.FormatExchangeCurrency(enabledCurrencies[j],
				asset.Spot)
			if err != nil {
				return nil, err
			}

			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: fpair.Upper(),
				Params:   params,
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (g *Gateio) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generatePayload(channelsToSubscribe)
	if err != nil {
		return err
	}

	var errs common.Errors
	for k := range payloads {
		resp, err := g.Websocket.Conn.SendMessageReturnResponse(payloads[k].ID, payloads[k])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		var response WebsocketAuthenticationResponse
		err = json.Unmarshal(resp, &response)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if response.Result.Status != "success" {
			errs = append(errs, fmt.Errorf("%v could not subscribe to %v",
				g.Name,
				payloads[k].Method))
			continue
		}
		g.Websocket.AddSuccessfulSubscriptions(payloads[k].Channels...)
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (g *Gateio) generatePayload(channelsToSubscribe []stream.ChannelSubscription) ([]WebsocketRequest, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}

	var payloads []WebsocketRequest
channels:
	for i := range channelsToSubscribe {
		// Ensures params are in order
		params := []interface{}{channelsToSubscribe[i].Currency}
		if strings.EqualFold(channelsToSubscribe[i].Channel, "depth.subscribe") {
			params = append(params,
				channelsToSubscribe[i].Params["limit"],
				channelsToSubscribe[i].Params["interval"])
		} else if strings.EqualFold(channelsToSubscribe[i].Channel, "kline.subscribe") {
			params = append(params, channelsToSubscribe[i].Params["interval"])
		}

		for j := range payloads {
			if payloads[j].Method == channelsToSubscribe[i].Channel {
				switch {
				case strings.EqualFold(channelsToSubscribe[i].Channel, "depth.subscribe"):
					if len(payloads[j].Params) == 3 {
						// If more than one currency pair we need to send as
						// matrix
						_, ok := payloads[j].Params[0].(currency.Pair)
						if ok {
							var bucket = payloads[j].Params
							payloads[j].Params = nil
							payloads[j].Params = append(payloads[j].Params, bucket)
						}
					}

					payloads[j].Params = append(payloads[j].Params, params)
				case strings.EqualFold(channelsToSubscribe[i].Channel, "kline.subscribe"):
					// Can only subscribe one market at the same time, market
					// list is not supported currently. For multiple
					// subscriptions, only the last one takes effect.
				default:
					payloads[j].Params = append(payloads[j].Params, params...)
				}
				payloads[j].Channels = append(payloads[j].Channels, channelsToSubscribe[i])
				continue channels
			}
		}

		payloads = append(payloads, WebsocketRequest{
			ID:       g.Websocket.Conn.GenerateMessageID(false),
			Method:   channelsToSubscribe[i].Channel,
			Params:   params,
			Channels: []stream.ChannelSubscription{channelsToSubscribe[i]},
		})
	}
	return payloads, nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	// NOTE: This function does not take in parameters, it cannot unsubscribe a
	// single item but a full channel. i.e. if you subscribe to ticker BTC_USDT
	// & LTC_USDT this function will unsubscribe both. This function will be
	// kept unlinked to the websocket subsystem and a full connection flush will
	// occur when currency items are disabled.
	var channelsThusFar []string
	for i := range channelsToUnsubscribe {
		if common.StringDataCompare(channelsThusFar,
			channelsToUnsubscribe[i].Channel) {
			continue
		}

		channelsThusFar = append(channelsThusFar,
			channelsToUnsubscribe[i].Channel)

		unsubscribeText := strings.Replace(channelsToUnsubscribe[i].Channel,
			"subscribe",
			"unsubscribe",
			1)

		unsubscribe := WebsocketRequest{
			ID:     g.Websocket.Conn.GenerateMessageID(false),
			Method: unsubscribeText,
			Params: []interface{}{channelsToUnsubscribe[i].Currency.String()},
		}

		resp, err := g.Websocket.Conn.SendMessageReturnResponse(unsubscribe.ID,
			unsubscribe)
		if err != nil {
			return err
		}
		var response WebsocketAuthenticationResponse
		err = json.Unmarshal(resp, &response)
		if err != nil {
			return err
		}
		if response.Result.Status != "success" {
			return fmt.Errorf("%v could not subscribe to %v",
				g.Name,
				channelsToUnsubscribe[i].Channel)
		}
	}
	return nil
}

func (g *Gateio) wsGetBalance(currencies []string) (*WsGetBalanceResponse, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to get balance", g.Name)
	}
	balanceWsRequest := wsGetBalanceRequest{
		ID:     g.Websocket.Conn.GenerateMessageID(false),
		Method: "balance.query",
		Params: currencies,
	}
	resp, err := g.Websocket.Conn.SendMessageReturnResponse(balanceWsRequest.ID, balanceWsRequest)
	if err != nil {
		return nil, err
	}
	var balance WsGetBalanceResponse
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		return &balance, err
	}

	if balance.Error.Message != "" {
		return nil, fmt.Errorf("%s websocket error: %s",
			g.Name,
			balance.Error.Message)
	}

	return &balance, nil
}

func (g *Gateio) wsGetOrderInfo(market string, offset, limit int) (*WebSocketOrderQueryResult, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to get order info", g.Name)
	}
	ord := WebsocketRequest{
		ID:     g.Websocket.Conn.GenerateMessageID(false),
		Method: "order.query",
		Params: []interface{}{
			market,
			offset,
			limit,
		},
	}

	resp, err := g.Websocket.Conn.SendMessageReturnResponse(ord.ID, ord)
	if err != nil {
		return nil, err
	}

	var orderQuery WebSocketOrderQueryResult
	err = json.Unmarshal(resp, &orderQuery)
	if err != nil {
		return &orderQuery, err
	}

	if orderQuery.Error.Message != "" {
		return nil, fmt.Errorf("%s websocket error: %s",
			g.Name,
			orderQuery.Error.Message)
	}

	return &orderQuery, nil
}
