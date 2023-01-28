package cryptodotcom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	publicHeartbeat        = "public/heartbeat"
	publicRespondHeartbeat = "public/respond-heartbeat"
)

var websocketSubscriptionEndpointsURL = []string{
	publicAuth,
	publicInstruments,

	privateSetCancelOnDisconnect,
	privateGetCancelOnDisconnect,

	postWithdrawal,
	privateGetWithdrawalHistory,
	privateGetAccountSummary,

	privateCreateOrder,
	privateCancelOrder,
	privateCreateOrderList,
	privateCancelOrderList,
	privateCancelAllOrders,
	privateGetOrderHistory,
	privateGetOpenOrders,
	privateGetOrderDetail,
	privateGetTrades,
}

// websocket subscriptions channels list

const (
	// private subscription channels
	userOrderCnl   = "user.order.%s" // user.order.{instrument_name}
	userTradeCnl   = "user.trade.%s" // user.trade.{instrument_name}
	userBalanceCnl = "user.balance"

	// public subscription channels

	instrumentOrderbookCnl = "book.%s"           // book.{instrument_name}
	tickerCnl              = "ticker.%s"         // ticker.{instrument_name}
	tradeCnl               = "trade.%s"          // trade.{instrument_name}
	candlestickCnl         = "candlestick.%s.%s" // candlestick.{time_frame}.{instrument_name}
)

var defaultSubscriptions = []string{
	instrumentOrderbookCnl,
	tickerCnl,
	tradeCnl,
	candlestickCnl,
}

// responseStream a channel thought which the data coming from the two websocket connection will go through.
var responseStream chan SubscriptionRawData

// this field is used to notify the websocket authentication status for the running go-routine
// and the Mutex locker below is used to safely access the field websocketAuthenticationFailed
var websocketAuthenticationFailed bool
var wAuthFailure sync.Mutex

func (cr *Cryptodotcom) WsConnect() error {
	responseStream = make(chan SubscriptionRawData)
	if !cr.Websocket.IsEnabled() || !cr.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
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
	cr.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 10,
	})
	if cr.Websocket.CanUseAuthenticatedEndpoints() {
		var authDialer websocket.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = cr.WsAuthConnect(context.TODO(), authDialer)
		if err != nil {
			wAuthFailure.Lock()
			websocketAuthenticationFailed = true
			wAuthFailure.Unlock()
			cr.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (cr *Cryptodotcom) wsFunnelConnectionData(ws stream.Connection, authenticated bool) {
	defer cr.Websocket.Wg.Done()
	for {
		if authenticated {
			wAuthFailure.Lock()
			if websocketAuthenticationFailed {
				return
			}
			wAuthFailure.Unlock()

		}
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- SubscriptionRawData{stream.Response{Raw: resp.Raw}, authenticated}
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
				err := cr.WsHandleData(resp.Data.Raw, resp.Authenticated)
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
			err := cr.WsHandleData(resp.Data.Raw, resp.Authenticated)
			if err != nil {
				cr.Websocket.DataHandler <- err
			}
		}
	}
}

func (cr *Cryptodotcom) respondHeartbeat(resp *SubscriptionResponse, authConnection bool) error {
	resp.Method = publicRespondHeartbeat
	if authConnection {
		return cr.Websocket.AuthConn.SendJSONMessage(resp)
	}
	return cr.Websocket.Conn.SendJSONMessage(resp)
}

// WsAuthConnect represents an authenticated connection to a websocket server
func (cr *Cryptodotcom) WsAuthConnect(ctx context.Context, dialer websocket.Dialer) error {
	if !cr.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", cr.Name)
	}
	err := cr.Websocket.AuthConn.Dial(&dialer, http.Header{})
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
	var idInt int64
	idInt = cr.Websocket.AuthConn.GenerateMessageID(true)
	req := &WsRequestPayload{
		ID:     idInt,
		Method: publicAuth,
		Nonce:  timestamp.UnixMilli(),
	}
	var hmac, payload []byte
	signaturePayload := publicAuth + strconv.FormatInt(idInt, 10) + creds.Key + strconv.FormatInt(timestamp.UnixMilli(), 10)
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signaturePayload),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	req.APIKey = creds.Key
	req.Signature = crypto.HexEncodeToString(hmac)
	payload, err = cr.Websocket.AuthConn.SendMessageReturnResponse(req.ID, req)
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
func (cr *Cryptodotcom) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return cr.handleSubscriptions("subscribe", subscriptions)
}

// Unsubscribe sends a websocket unsubscription to a channel message through the websocket connection handlers.
func (cr *Cryptodotcom) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return cr.handleSubscriptions("unsubscribe", subscriptions)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (cr *Cryptodotcom) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	subscriptions := []stream.ChannelSubscription{}
	channels := defaultSubscriptions
	if cr.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(
			channels,

			// authenticated endpoint subscriptions.
			userBalanceCnl,
			userOrderCnl,
			userTradeCnl,
		)
	}
	for x := range channels {
		if channels[x] == userBalanceCnl {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
			})
			continue
		}
		enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
		for p := range enabledPairs {
			switch channels[x] {
			case instrumentOrderbookCnl,
				tickerCnl,
				userOrderCnl,
				userTradeCnl,
				tradeCnl:
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: enabledPairs[p],
				})
			case candlestickCnl:
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: enabledPairs[p],
					Params: map[string]interface{}{
						"interval": "5m",
					},
				})
			default:
				continue
			}
		}
	}
	return subscriptions, nil
}

func (cr *Cryptodotcom) handleSubscriptions(operation string, subscriptions []stream.ChannelSubscription) error {
	subscriptionPayloads, err := cr.generatePayload(operation, subscriptions)
	if err != nil {
		return err
	}
	for p := range subscriptionPayloads {
		if subscriptionPayloads[p].Authenticated {
			err = cr.Websocket.AuthConn.SendJSONMessage(subscriptionPayloads[p])
		} else {
			err = cr.Websocket.Conn.SendJSONMessage(subscriptionPayloads[p])
		}
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (cr *Cryptodotcom) generatePayload(operation string, subscription []stream.ChannelSubscription) ([]SubscriptionPayload, error) {
	subscriptionPayloads := make([]SubscriptionPayload, len(subscription))
	timestamp := time.Now()
	for x := range subscription {
		subscriptionPayloads[x] = SubscriptionPayload{
			ID:     int(cr.Websocket.Conn.GenerateMessageID(false)),
			Method: operation,
			Nonce:  timestamp.UnixMilli(),
		}
		switch subscription[x].Channel {
		case userOrderCnl,
			userTradeCnl,
			instrumentOrderbookCnl,
			tickerCnl,
			tradeCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {fmt.Sprintf(subscription[x].Channel, subscription[x].Currency.String())}}
		case candlestickCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {fmt.Sprintf(subscription[x].Channel, subscription[x].Params["interval"].(string), subscription[x].Currency.String())}}
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
	if resp.Method == "subscribe" {
		switch resp.Result.Channel {
		case "user.order":
			return cr.processUserOrderbook(&resp.Result)
		case "user.trade":
			if !cr.IsFillsFeedEnabled() {
				return nil
			}
			return cr.processUserTrade(&resp.Result)
		case "user.balance":
			return cr.processUserBalance(&resp.Result)
		case "book":
			return cr.processOrderbook(&resp.Result)
		case "ticker":
			return cr.processTicker(&resp.Result)
		case "trade":
			return cr.processTrades(&resp.Result)
		case "candlestick":
			return cr.processCandlestick(&resp.Result)
		default:
			if !cr.Websocket.Match.IncomingWithData(resp.ID, respRaw) {
				return fmt.Errorf("can not pass push data message with signature %d with method %s", resp.ID, resp.Method)
			}
		}
	} else if !cr.Websocket.Match.IncomingWithData(resp.ID, respRaw) {
		return fmt.Errorf("can not pass push data message with signature %d with method %s", resp.ID, resp.Method)
	}
	return nil
}

func (cr *Cryptodotcom) processCandlestick(resp *WsResult) error {
	var data []CandlestickItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return nil
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	interval, err := stringToInterval(resp.Interval)
	if err != nil {
		return err
	}
	for x := range data {
		cr.Websocket.DataHandler <- stream.KlineData{
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
	return nil
}

func (cr *Cryptodotcom) processTrades(resp *WsResult) error {
	var data []TradeItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return nil
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
			Amount:       data[i].TradeQuantity,
			Price:        data[i].TradePrice,
			AssetType:    asset.Spot,
			CurrencyPair: cp,
			Exchange:     cr.Name,
			Side:         oSide,
			Timestamp:    data[i].TradeTimestamp.Time(),
			TID:          data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(cr.Name, trades...)
}

func (cr *Cryptodotcom) processTicker(resp *WsResult) error {
	var data []TickerItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return nil
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	for x := range data {
		cr.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: cr.Name,
			Volume:       data[x].TradedVolume,
			QuoteVolume:  data[x].TradedVolumeInUSD24H,
			High:         data[x].HighestTradePrice,
			Low:          data[x].LowestTradePrice,
			Bid:          data[x].BestBidPrice,
			BidSize:      data[x].BestBidSize,
			Ask:          data[x].BestAskPrice,
			AskSize:      data[x].BestAskSize,
			Last:         data[x].LatestTradePrice,
			AssetType:    asset.Spot,
			Pair:         cp,
		}
	}
	return nil
}

func (cr *Cryptodotcom) processOrderbook(resp *WsResult) error {
	var data []WsOrderbook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return nil
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
		book.Asks = make([]orderbook.Item, len(data[x].Asks))
		for i := range data[x].Asks {
			book.Asks[i].Price, err = strconv.ParseFloat(data[x].Asks[i][0], 64)
			if err != nil {
				return err
			}
			book.Asks[i].Amount, err = strconv.ParseFloat(data[x].Asks[i][1], 64)
			if err != nil {
				return err
			}
		}
		book.Bids = make([]orderbook.Item, len(data[x].Bids))
		for i := range data[x].Bids {
			book.Bids[i].Price, err = strconv.ParseFloat(data[x].Bids[i][0], 64)
			if err != nil {
				return err
			}
			book.Bids[i].Amount, err = strconv.ParseFloat(data[x].Bids[i][1], 64)
			if err != nil {
				return err
			}
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
		return nil
	}
	for x := range data {
		cr.Websocket.DataHandler <- account.Change{
			Exchange: cr.Name,
			Currency: currency.NewCode(data[x].Currency),
			Asset:    asset.Spot,
			Amount:   data[x].Balance,
		}
	}
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
	var data []UserOrderbook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	for x := range data {
		status, err := order.StringToOrderStatus(data[x].Status)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(data[x].Type)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(data[x].Side)
		if err != nil {
			return err
		}
		cr.Websocket.DataHandler <- &order.Detail{
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
	return nil
}
