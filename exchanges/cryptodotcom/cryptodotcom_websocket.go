package cryptodotcom

import (
	"context"
	"errors"
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
	publicHeartbeat        = "public/heartbeat"
	publicRespondHeartbeat = "public/respond-heartbeat"
)

// websocket subscriptions channels list

const (
	// private subscription channels
	userOrderCnl   = "user.order" // user.order.{instrument_name}
	userTradeCnl   = "user.trade" // user.trade.{instrument_name}
	userBalanceCnl = "user.balance"

	// public subscription channels
	instrumentOrderbookCnl = "book"        // book.{instrument_name}
	tickerCnl              = "ticker"      // ticker.{instrument_name}
	tradeCnl               = "trade"       // trade.{instrument_name}
	candlestickCnl         = "candlestick" // candlestick.{time_frame}.{instrument_name}
	otcBooksCnl            = "otc_book"    // otc_book.{instrument_name}
)

var defaultSubscriptions = []string{
	instrumentOrderbookCnl,
	tickerCnl,
	tradeCnl,
	candlestickCnl,
}

// responseStream a channel to multiplex push data coming through the two websocket connections, Public and Authenticated, to the method wsHandleData
var responseStream chan SubscriptionRawData

// WsConnect creates a new websocket to public and private endpoints.
func (cr *Cryptodotcom) WsConnect() error {
	responseStream = make(chan SubscriptionRawData)
	if !cr.Websocket.IsEnabled() || !cr.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	err := cr.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	cr.Websocket.Wg.Add(2)
	go cr.wsFunnelConnectionData(cr.Websocket.Conn, false)
	go cr.WsReadData()
	if cr.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			cr.Websocket.GetWebsocketURL())
	}
	cr.Websocket.Conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PingMessage,
		Delay:             time.Second * 10,
	})
	if cr.Websocket.CanUseAuthenticatedEndpoints() {
		var authDialer gws.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = cr.WsAuthConnect(&authDialer)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s", cr.Name, err)
			cr.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (cr *Cryptodotcom) wsFunnelConnectionData(ws websocket.Connection, authenticated bool) {
	defer cr.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- SubscriptionRawData{Data: resp.Raw, Authenticated: authenticated}
	}
}

// WsReadData read coming messages thought the websocket connection and process the data.
func (cr *Cryptodotcom) WsReadData() {
	defer cr.Websocket.Wg.Done()
	for {
		select {
		case <-cr.Websocket.ShutdownC:
			select {
			case resp := <-responseStream:
				err := cr.WsHandleData(resp.Data, resp.Authenticated)
				if err != nil {
					select {
					case cr.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", cr.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseStream:
			err := cr.WsHandleData(resp.Data, resp.Authenticated)
			if err != nil {
				cr.Websocket.DataHandler <- err
			}
		}
	}
}

func (cr *Cryptodotcom) respondHeartbeat(resp *SubscriptionResponse, authConnection bool) error {
	subscriptionInput := &SubscriptionInput{
		ID:     resp.ID,
		Code:   resp.Code,
		Method: publicRespondHeartbeat,
	}
	if authConnection {
		return cr.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, subscriptionInput)
	}
	return cr.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, subscriptionInput)
}

// WsAuthConnect represents an authenticated connection to a websocket server
func (cr *Cryptodotcom) WsAuthConnect(dialer *gws.Dialer) error {
	if !cr.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", cr.Name)
	}
	err := cr.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", cr.Name, cryptodotcomWebsocketUserAPI, err)
	}
	cr.Websocket.Wg.Add(1)
	go cr.wsFunnelConnectionData(cr.Websocket.AuthConn, true)
	return cr.AuthenticateWebsocketConnection()
}

// AuthenticateWebsocketConnection authenticates the websocekt connection.
func (cr *Cryptodotcom) AuthenticateWebsocketConnection() error {
	creds, err := cr.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	timestamp := time.Now()
	req := &WsRequestPayload{
		ID:     cr.Websocket.AuthConn.GenerateMessageID(true),
		Method: publicAuth,
		Nonce:  timestamp.UnixMilli(),
	}
	var hmac, payload []byte
	signaturePayload := publicAuth + strconv.FormatInt(req.ID, 10) + creds.Key + strconv.FormatInt(timestamp.UnixMilli(), 10)
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signaturePayload),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	req.APIKey = creds.Key
	req.Signature = crypto.HexEncodeToString(hmac)
	payload, err = cr.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, req.ID, req)
	if err != nil {
		return err
	}
	var resp *RespData
	err = json.Unmarshal(payload, &resp)
	if err != nil {
		return err
	} else if resp == nil {
		return errors.New("no valid response from server")
	}
	if resp.Code != 0 {
		mes := fmt.Sprintf("error code: %d Message: %s", resp.Code, resp.Message)
		if resp.DetailCode != "0" && resp.DetailCode != "" {
			mes = fmt.Sprintf("%s Detail: %s %s", mes, resp.DetailCode, resp.DetailMessage)
		}
		return errors.New(mes)
	}
	return nil
}

// Subscribe sends a websocket subscription to a channel message through the websocket connection handlers.
func (cr *Cryptodotcom) Subscribe(subscriptions subscription.List) error {
	return cr.handleSubscriptions("subscribe", subscriptions)
}

// Unsubscribe sends a websocket unsubscription to a channel message through the websocket connection handlers.
func (cr *Cryptodotcom) Unsubscribe(subscriptions subscription.List) error {
	return cr.handleSubscriptions("unsubscribe", subscriptions)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (cr *Cryptodotcom) GenerateDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := defaultSubscriptions
	if cr.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(
			channels,

			// authenticated endpoint subscriptions.
			userBalanceCnl,
			userOrderCnl,
			userTradeCnl,
			otcBooksCnl,
		)
	}
	for x := range channels {
		if channels[x] == userBalanceCnl {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
			})
			continue
		}
		enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
		switch channels[x] {
		case instrumentOrderbookCnl,
			tickerCnl,
			userOrderCnl,
			userTradeCnl,
			tradeCnl,
			otcBooksCnl:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
				Pairs:   enabledPairs,
			})
		case candlestickCnl:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
				Pairs:   enabledPairs,
				Params: map[string]interface{}{
					"interval": "5m",
				},
			})
		default:
			continue
		}
	}
	return subscriptions, nil
}

func (cr *Cryptodotcom) handleSubscriptions(operation string, subscriptions subscription.List) error {
	subscriptionPayloads, err := cr.generatePayload(operation, subscriptions)
	if err != nil {
		return err
	}
	for p := range subscriptionPayloads {
		if subscriptionPayloads[p].Authenticated {
			err = cr.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, subscriptionPayloads[p])
		} else {
			err = cr.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, subscriptionPayloads[p])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (cr *Cryptodotcom) generatePayload(operation string, subscription subscription.List) ([]SubscriptionPayload, error) {
	subscriptionPayloads := make([]SubscriptionPayload, len(subscription))
	timestamp := time.Now()
	for x := range subscription {
		subscriptionPayloads[x] = SubscriptionPayload{
			ID:     cr.Websocket.Conn.GenerateMessageID(false),
			Method: operation,
			Nonce:  timestamp.UnixMilli(),
		}
		switch subscription[x].Channel {
		case userOrderCnl,
			userTradeCnl,
			instrumentOrderbookCnl,
			tickerCnl,
			tradeCnl,
			otcBooksCnl:
			for p := range subscription[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel + "." + subscription[x].Pairs[p].String()}}
			}
		case candlestickCnl:
			interval, okay := subscription[x].Params["interval"].(string)
			if !okay {
				return nil, kline.ErrInvalidInterval
			}
			for p := range subscription[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel + "." + interval + "." + subscription[x].Pairs[p].String()}}
			}
		case userBalanceCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel}}
		}
		switch subscription[x].Channel {
		case userOrderCnl, userTradeCnl, userBalanceCnl:
			subscriptionPayloads[x].Authenticated = true
		}
	}
	return subscriptionPayloads, nil
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (cr *Cryptodotcom) WsHandleData(respRaw []byte, authConnection bool) error {
	var resp *SubscriptionResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}

	if resp.ID > 0 {
		if resp.Method == publicHeartbeat {
			return cr.respondHeartbeat(resp, authConnection)
		}
	}
	if resp.Result == nil {
		resp.Result = &WsResult{}
	}
	if resp.Method == "subscribe" {
		switch {
		case strings.HasPrefix(resp.Result.Channel, userOrderCnl):
			return cr.processUserOrderbook(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, userTradeCnl):
			if !cr.IsFillsFeedEnabled() {
				return nil
			}
			return cr.processUserTrade(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, userBalanceCnl):
			return cr.processUserBalance(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, instrumentOrderbookCnl):
			return cr.processOrderbook(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, tickerCnl):
			return cr.processTicker(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, tradeCnl):
			return cr.processTrades(resp.Result)
		case strings.HasPrefix(resp.Result.Channel, candlestickCnl):
			return cr.processCandlestick(resp.Result)
		case resp.Result.Channel == otcBooksCnl:
			return cr.processOTCOrderbook(resp.Result)
		default:
			if resp.Code == 0 {
				return nil
			}
		}
	}
	if !cr.Websocket.Match.IncomingWithData(resp.ID, respRaw) {
		return fmt.Errorf("could not match incoming message with signature: %d, and data: %s", resp.ID, string(respRaw))
	}
	return nil
}

func (cr *Cryptodotcom) processOTCOrderbook(resp *WsResult) error {
	var data []OTCBook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	for x := range data {
		book := orderbook.Base{
			Exchange:        cr.Name,
			Pair:            cp,
			Asset:           asset.OTC,
			VerifyOrderbook: cr.CanVerifyOrderbook,
			LastUpdated:     resp.Timestamp.Time(),
		}
		book.Asks = make([]orderbook.Tranche, len(data[x].Asks))
		for i := range data[x].Asks {
			book.Asks[i].Price = data[x].Asks[i][0].Float64()
			book.Asks[i].Amount = data[x].Asks[i][1].Float64()
		}
		book.Bids = make([]orderbook.Tranche, len(data[x].Bids))
		for i := range data[x].Bids {
			book.Bids[i].Price = data[x].Bids[i][0].Float64()
			book.Bids[i].Amount = data[x].Bids[i][1].Float64()
		}
		book.Asks.SortAsks()
		book.Bids.SortBids()
		err = cr.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cr *Cryptodotcom) processCandlestick(resp *WsResult) error {
	var data []WsCandlestickItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	interval, err := stringToInterval(resp.Interval)
	if err != nil {
		return err
	}
	candles := make([]websocket.KlineData, len(data))
	for x := range data {
		candles[x] = websocket.KlineData{
			Pair:      cp,
			Exchange:  cr.Name,
			Timestamp: data[x].UpdateTime.Time(),
			Interval:  interval.Word(),
			AssetType: asset.Spot,
			OpenPrice: data[x].Open,
			HighPrice: data[x].High,
			LowPrice:  data[x].Low,
			Volume:    data[x].Volume,
			StartTime: data[x].EndTime.Time(), // This field represents Start Timestamp for websocket push data. and End Timestamp for REST
		}
	}
	cr.Websocket.DataHandler <- candles
	return nil
}

func (cr *Cryptodotcom) processTrades(resp *WsResult) error {
	var data []TradeItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(data))
	for i := range data {
		var oSide order.Side
		oSide, err = order.StringToOrderSide(data[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			Amount:       data[i].TradeQuantity.Float64(),
			Price:        data[i].TradePrice.Float64(),
			AssetType:    asset.Spot,
			CurrencyPair: cp,
			Exchange:     cr.Name,
			Side:         oSide,
			Timestamp:    data[i].TradeTimestamp.Time(),
			TID:          data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

func (cr *Cryptodotcom) processTicker(resp *WsResult) error {
	var data []TickerItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	tickersDatas := make([]ticker.Price, len(data))
	for x := range data {
		tickersDatas[x] = ticker.Price{
			ExchangeName: cr.Name,
			Volume:       data[x].TradedVolume.Float64(),
			QuoteVolume:  data[x].TradedVolumeInUSD24H.Float64(),
			High:         data[x].HighestTradePrice.Float64(),
			Low:          data[x].LowestTradePrice.Float64(),
			Bid:          data[x].BestBidPrice.Float64(),
			BidSize:      data[x].BestBidSize.Float64(),
			Ask:          data[x].BestAskPrice.Float64(),
			AskSize:      data[x].BestAskSize.Float64(),
			Last:         data[x].LatestTradePrice.Float64(),
			AssetType:    asset.Spot,
			Pair:         cp,
		}
	}
	cr.Websocket.DataHandler <- tickersDatas
	return nil
}

func (cr *Cryptodotcom) processOrderbook(resp *WsResult) error {
	var data []WsOrderbook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	for x := range data {
		book := orderbook.Base{
			Exchange:        cr.Name,
			Pair:            cp,
			Asset:           asset.Spot,
			LastUpdated:     data[x].OrderbookUpdateTime.Time(),
			LastUpdateID:    data[x].UpdateSequence,
			VerifyOrderbook: cr.CanVerifyOrderbook,
		}
		book.Asks = make([]orderbook.Tranche, len(data[x].Asks))
		for i := range data[x].Asks {
			book.Asks[i].Price = data[x].Asks[i][0].Float64()
			book.Asks[i].Amount = data[x].Asks[i][1].Float64()
		}
		book.Bids = make([]orderbook.Tranche, len(data[x].Bids))
		for i := range data[x].Bids {
			book.Bids[i].Price = data[x].Bids[i][0].Float64()
			book.Bids[i].Amount = data[x].Bids[i][1].Float64()
		}
		book.Asks.SortAsks()
		book.Bids.SortBids()
		err = cr.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cr *Cryptodotcom) processUserBalance(resp *WsResult) error {
	var data []UserBalance
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	accountChanges := make([]account.Change, len(data))
	for x := range data {
		accountChanges[x] = account.Change{
			Exchange: cr.Name,
			Currency: currency.NewCode(data[x].Currency),
			Asset:    asset.Spot,
			Amount:   data[x].Balance,
		}
	}
	cr.Websocket.DataHandler <- accountChanges
	return nil
}

func (cr *Cryptodotcom) processUserTrade(resp *WsResult) error {
	var data []UserTrade
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(data))
	for x := range data {
		oSide, err := order.StringToOrderSide(data[x].Side)
		if err != nil {
			cr.Websocket.DataHandler <- order.ClassificationError{
				Exchange: cr.Name,
				Err:      err,
			}
		}
		fills[x] = fill.Data{
			ID:           data[x].OrderID,
			Timestamp:    data[x].CreateTime.Time(),
			Exchange:     cr.Name,
			AssetType:    asset.Spot,
			CurrencyPair: cp,
			Side:         oSide,
			OrderID:      data[x].OrderID,
			TradeID:      data[x].TradeID,
			Price:        data[x].TradedPrice,
			Amount:       data[x].TradedQuantity,
		}
	}
	return cr.Websocket.Fills.Update(fills...)
}

func (cr *Cryptodotcom) processUserOrderbook(resp *WsResult) error {
	var data []UserOrder
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	ordersDetails := make([]order.Detail, len(data))
	for x := range data {
		status, err := order.StringToOrderStatus(data[x].Status)
		if err != nil {
			return err
		}
		oType, err := StringToOrderType(data[x].Type)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(data[x].Side)
		if err != nil {
			return err
		}
		ordersDetails[x] = order.Detail{
			Price:                data[x].Price,
			Amount:               data[x].Quantity,
			AverageExecutedPrice: data[x].AvgPrice,
			ExecutedAmount:       data[x].CumulativeExecutedQuantity,
			RemainingAmount:      data[x].Quantity - data[x].CumulativeExecutedQuantity,
			Cost:                 data[x].CumulativeExecutedValue,
			CostAsset:            cp.Quote,
			FeeAsset:             currency.NewCode(data[x].FeeCurrency),
			Exchange:             cr.Name,
			OrderID:              data[x].OrderID,
			ClientOrderID:        data[x].ClientOrderID,
			LastUpdated:          data[x].UpdateTime.Time(),
			Date:                 data[x].CreateTime.Time(),
			Side:                 oSide,
			Type:                 oType,
			AssetType:            asset.Spot,
			Status:               status,
			Pair:                 cp,
		}
	}
	cr.Websocket.DataHandler <- ordersDetails
	return nil
}
