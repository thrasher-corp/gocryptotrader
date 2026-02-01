package apexpro

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	apexProWebsocket        = "wss://qa-quote.omni.apex.exchange/realtime_public"
	apexProPrivateWebsocket = "wss://quote.omni.apex.exchange/realtime_private"

	chOrderbook   = "orderBook"
	chTrade       = "recentlyTrade"
	chTicker      = "instrumentInfo"
	chAllTickers  = "instrumentInfo.all"
	chCandlestick = "candle"

	chNotify      = "ws_notify_v1"
	chZKAccountV3 = "ws_zk_accounts_v3"
)

var defaultChannels = []string{
	chOrderbook, chTrade, chTicker, chCandlestick, chAllTickers,
}

func generatePingMessage() ([]byte, error) {
	return json.Marshal(&WsMessage{
		Operation: "ping",
		Args:      []string{strconv.FormatInt(time.Now().UnixMilli(), 10)},
	})
}

// WsConnect creates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.Background()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	err = e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			e.Name,
			err)
	}
	payload, err := generatePingMessage()
	if err != nil {
		return err
	}
	e.Websocket.Conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Message:           payload,
	})
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.Conn)
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		err := e.WsAuth(&dialer)
		e.Websocket.SetCanUseAuthenticatedEndpoints(err == nil)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%v", err.Error())
		}
	}
	return nil
}

// WsAuth authenticates the websocket connection
func (e *Exchange) WsAuth(dialer *gws.Dialer) error {
	ctx := context.Background()
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	err = e.Websocket.AuthConn.Dial(ctx, dialer, http.Header{})
	if err != nil {
		return err
	}
	timestamp := time.Now().UnixMilli()
	req := WsInput{
		Type:        "login",
		RequestPath: "/ws/accounts",
		Timestamp:   timestamp,
		HTTPMethod:  http.MethodGet,
		Topics:      []string{chNotify, chZKAccountV3},
		Passphrase:  creds.ClientID,
		APIKey:      creds.Key,
	}
	encodedSecret := base64.StdEncoding.EncodeToString([]byte(creds.Secret))
	var hmacSigned []byte
	messageString := strconv.FormatInt(timestamp, 10) + req.HTTPMethod + req.RequestPath
	hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(messageString),
		[]byte(encodedSecret))
	if err != nil {
		return err
	}
	signature := base64.StdEncoding.EncodeToString(hmacSigned)
	req.Signature = signature
	value, err := json.Marshal(req)
	if err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.AuthConn)
	return e.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, &struct {
		Operation string        `json:"op"`
		Arguments []interface{} `json:"args"`
	}{
		Operation: "login",
		Arguments: []interface{}{string(value)},
	})
}

// GenerateDefaultSubscriptions generates a default subscription list.
func (e *Exchange) GenerateDefaultSubscriptions() (subscription.List, error) {
	subscriptions := subscription.List{}
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return subscriptions, err
	}
	for a := range defaultChannels {
		switch defaultChannels[a] {
		case chOrderbook:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: defaultChannels[a],
				Pairs:   enabledPairs,
				Levels:  200,
			})
		case chTrade, chTicker:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:  defaultChannels[a],
				Pairs:    enabledPairs,
				Interval: kline.HundredMilliseconds,
			})
		case chCandlestick:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:  defaultChannels[a],
				Pairs:    enabledPairs,
				Levels:   200,
				Interval: kline.FiveMin,
			})
		case chAllTickers:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: defaultChannels[a],
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket channel subscription.
func (e *Exchange) Subscribe(subscriptions subscription.List) error {
	payload, err := e.handleSubscriptionPayload("subscribe", subscriptions)
	if err != nil {
		return err
	}
	err = e.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payload)
	if err != nil {
		return err
	}
	return e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, subscriptions...)
}

// Unsubscribe sends a websocket channel unsubscriptions.
func (e *Exchange) Unsubscribe(subscriptions subscription.List) error {
	payload, err := e.handleSubscriptionPayload("unsubscribe", subscriptions)
	if err != nil {
		return err
	}
	return e.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payload)
}

func (e *Exchange) handleSubscriptionPayload(operation string, subscriptions subscription.List) (*WsMessage, error) {
	susbcriptionPayload := &WsMessage{
		Operation: operation,
		Args:      []string{},
	}
	pairFormat, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	for s := range subscriptions {
		subscriptions[s].Pairs = subscriptions[s].Pairs.Format(pairFormat)
		switch subscriptions[s].Channel {
		case chOrderbook:
			if subscriptions[s].Levels == 0 {
				return nil, errOrderbookLevelIsRequired
			}
			for p := range subscriptions[s].Pairs {
				susbcriptionPayload.Args = append(susbcriptionPayload.Args, subscriptions[s].Channel+strconv.Itoa(subscriptions[s].Levels)+".H."+subscriptions[s].Pairs[p].String())
			}
		case chTrade, chTicker:
			for p := range subscriptions[s].Pairs {
				susbcriptionPayload.Args = append(susbcriptionPayload.Args, subscriptions[s].Channel+".H."+subscriptions[s].Pairs[p].String())
			}
		case chCandlestick:
			if subscriptions[s].Interval == kline.Interval(0) {
				return nil, kline.ErrInvalidInterval
			}
			intervalString, err := intervalToString(subscriptions[s].Interval)
			if err != nil {
				return nil, err
			}
			for p := range subscriptions[s].Pairs {
				susbcriptionPayload.Args = append(susbcriptionPayload.Args, subscriptions[s].Channel+"."+intervalString+"."+subscriptions[s].Pairs[p].String())
			}
		case chAllTickers:
			susbcriptionPayload.Args = append(susbcriptionPayload.Args, subscriptions[s].Channel)
		}
	}
	return susbcriptionPayload, nil
}

func (e *Exchange) wsReadData(conn websocket.Connection) {
	defer e.Websocket.Wg.Done()
	for {
		response := conn.ReadMessage()
		if response.Raw == nil {
			return
		}
		err := e.wsHandleData(response.Raw)
		if err != nil {
			log.Errorln(log.WebsocketMgr, err)
		}
	}
}

func (e *Exchange) wsHandleData(respRaw []byte) error {
	var response WsMessage
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	switch response.Operation {
	case "pong":
	case chOrderbook:
		return e.processOrderbook(respRaw)
	case chTrade:
		return e.processTrades(respRaw)
	case chTicker:
		return e.processTickerData(respRaw)
	case chCandlestick:
		return e.processCandlestickData(respRaw)
	case chAllTickers:
		return e.processAllTickers(respRaw)
	default:
		var authResp *WsAuthResponse
		err = json.Unmarshal(respRaw, &authResp)
		if err != nil {
			return err
		}
		switch authResp.Topic {
		case chZKAccountV3:
			var resp *AuthWebsocketAccountResponse
			err = json.Unmarshal(authResp.Contents, &resp)
			if err != nil {
				return err
			}
			err = e.processAccountOrders(resp.Orders)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%v", err.Error())
			}
			err = e.processAccountFills(resp.Fills)
			if err != nil {
				log.Warnf(log.ExchangeSys, "%v", err.Error())
			}
		case chNotify:
			var resp *WsAccountNotificationsResponse
			err = json.Unmarshal(authResp.Contents, &resp)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler.Send(context.Background(), resp)
		}
	}
	return nil
}

func (e *Exchange) processAccountOrders(respOrders []OrderDetail) error {
	orders := make([]order.Detail, len(respOrders))
	for o := range respOrders {
		pair, err := currency.NewPairFromString(respOrders[o].Symbol)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(respOrders[o].OrderType)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(respOrders[o].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(respOrders[o].Status)
		if err != nil {
			return err
		}
		switch respOrders[o].Status {
		case "PENDING":
			oStatus = order.Pending
		case "OPEN":
			oStatus = order.Open
		case "FILLED":
			oStatus = order.Filled
		case "CANCELED":
			oStatus = order.Cancelled
		case "EXPIRED":
			oStatus = order.Expired
		case "UNTRIGGERED":
			oStatus = order.Hidden
		}
		tif, err := order.StringToTimeInForce(respOrders[o].TimeInForce)
		if err != nil {
			return err
		}
		if respOrders[o].PostOnly {
			tif |= order.PostOnly
		}
		orders[o] = order.Detail{
			TimeInForce:        tif,
			ReduceOnly:         respOrders[o].ReduceOnly,
			Price:              respOrders[o].Price.Float64(),
			Amount:             respOrders[o].Size.Float64(),
			ContractAmount:     respOrders[o].Size.Float64(),
			TriggerPrice:       respOrders[o].TriggerPrice.Float64(),
			ExecutedAmount:     respOrders[o].Size.Float64() - respOrders[o].RemainingSize.Float64(),
			RemainingAmount:    respOrders[o].RemainingSize.Float64(),
			Cost:               respOrders[o].Size.Float64() * respOrders[o].Price.Float64(),
			Fee:                respOrders[o].Fee.Float64(),
			FeeAsset:           pair.Quote,
			Exchange:           e.Name,
			OrderID:            respOrders[o].ID,
			ClientOrderID:      respOrders[o].ClientOrderID,
			AccountID:          respOrders[o].AccountID,
			Type:               oType,
			Side:               oSide,
			Status:             oStatus,
			AssetType:          asset.Futures,
			Date:               respOrders[o].CreatedAt.Time(),
			CloseTime:          respOrders[o].ExpiresAt.Time(),
			LastUpdated:        respOrders[o].UpdatedTime.Time(),
			Pair:               pair,
			SettlementCurrency: pair.Quote,
		}
		return e.Websocket.DataHandler.Send(context.Background(), orders)
	}
	return nil
}

func (e *Exchange) processAccountFills(orderFills []WsAccountOrderFill) error {
	fillsList := make([]fill.Data, len(orderFills))
	for f := range orderFills {
		pair, err := currency.NewPairFromString(orderFills[f].Symbol)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(orderFills[f].Side)
		if err != nil {
			return err
		}
		fillsList[f] = fill.Data{
			ID:           orderFills[f].ID,
			Timestamp:    orderFills[f].UpdatedAt.Time(),
			Exchange:     e.Name,
			AssetType:    asset.Futures,
			CurrencyPair: pair,
			Side:         oSide,
			OrderID:      orderFills[f].OrderID,
			TradeID:      orderFills[f].ID,
			Price:        orderFills[f].Price.Float64(),
			Amount:       orderFills[f].Size.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(context.Background(), fillsList)
}

func (e *Exchange) processOrderbook(respRaw []byte) error {
	var resp *WsDepth
	var cp currency.Pair
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err = currency.NewPairFromString(resp.Data.Symbol)
	if err != nil {
		return err
	}
	asks := make(orderbook.Levels, len(resp.Data.Asks))
	for a := range resp.Data.Asks {
		asks[a].Price = resp.Data.Asks[a][0].Float64()
		asks[a].Amount = resp.Data.Asks[a][1].Float64()
	}
	bids := make(orderbook.Levels, len(resp.Data.Bids))
	for b := range resp.Data.Bids {
		bids[b].Price = resp.Data.Bids[b][0].Float64()
		bids[b].Amount = resp.Data.Bids[b][1].Float64()
	}
	if resp.Type == "delta" {
		return e.Websocket.Orderbook.Update(&orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       cp,
			UpdateID:   resp.Data.UpdateID,
			UpdateTime: resp.Timestamp.Time(),
			Asset:      asset.Futures,
		})
	}
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Pair:              cp,
		Asset:             asset.Spot,
		Exchange:          e.Name,
		LastUpdateID:      resp.Data.UpdateID,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdated:       time.Now(),
		Asks:              asks,
		Bids:              bids,
	})
}

func (e *Exchange) processTrades(respRaw []byte) error {
	var resp *WsTrade
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!e.IsTradeFeedEnabled() {
		return nil
	}
	trades := make([]trade.Data, len(resp.Data))
	for a := range resp.Data {
		cp, err := currency.NewPairFromString(resp.Data[a].Symbol)
		if err != nil {
			return err
		}
		trades[a] = trade.Data{
			CurrencyPair: cp,
			Timestamp:    resp.Data[a].Timestamp.Time(),
			Price:        resp.Data[a].Price.Float64(),
			Amount:       resp.Data[a].Volume.Float64(),
			Exchange:     e.Name,
			AssetType:    asset.Futures,
			TID:          resp.Data[a].OrderID,
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

func (e *Exchange) processTickerData(respRaw []byte) error {
	var resp *WsTicker
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Data.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(context.Background(), &ticker.Price{
		Last:         resp.Data.LastPrice.Float64(),
		High:         resp.Data.HighPrice24H.Float64(),
		Low:          resp.Data.LowPrice24H.Float64(),
		Volume:       resp.Data.Volume24H.Float64(),
		OpenInterest: resp.Data.OpenInterest.Float64(),
		MarkPrice:    resp.Data.OraclePrice.Float64(),
		IndexPrice:   resp.Data.IndexPrice.Float64(),
		Pair:         cp,
		ExchangeName: e.Name,
		AssetType:    asset.Futures,
	})
}

func (e *Exchange) processCandlestickData(respRaw []byte) error {
	var resp *WsCandlesticks
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	klineData := make([]websocket.KlineData, len(resp.Data))
	for a := range resp.Data {
		pair, err := currency.NewPairFromString(resp.Data[a].Symbol)
		if err != nil {
			return err
		}
		klineData[a] = websocket.KlineData{
			Timestamp:  resp.Timestamp.Time(),
			Pair:       pair,
			AssetType:  asset.Futures,
			Exchange:   e.Name,
			StartTime:  resp.Data[a].Start.Time(),
			Interval:   resp.Data[a].Interval,
			OpenPrice:  resp.Data[a].Open.Float64(),
			ClosePrice: resp.Data[a].Close.Float64(),
			HighPrice:  resp.Data[a].High.Float64(),
			LowPrice:   resp.Data[a].Low.Float64(),
			Volume:     resp.Data[a].Volume.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(context.Background(), klineData)
}

func (e *Exchange) processAllTickers(respRaw []byte) error {
	var resp *WsSymbolsTickerInformaton
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	tickerData := make([]ticker.Price, len(resp.Data))
	for a := range resp.Data {
		pair, err := currency.NewPairFromString(resp.Data[a].Symbol)
		if err != nil {
			return err
		}
		tickerData[a] = ticker.Price{
			Last:         resp.Data[a].LastPrice.Float64(),
			High:         resp.Data[a].Highest24Hr.Float64(),
			Low:          resp.Data[a].Lowest24Hr.Float64(),
			Volume:       resp.Data[a].Volume24Hr.Float64(),
			Open:         resp.Data[a].OpeningPrice.Float64(),
			OpenInterest: resp.Data[a].OpenInterest.Float64(),
			MarkPrice:    resp.Data[a].MarkPrice.Float64(),
			IndexPrice:   resp.Data[a].IndexPrice.Float64(),
			Pair:         pair,
			ExchangeName: e.Name,
			AssetType:    asset.Futures,
			LastUpdated:  resp.Timestamp.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(context.Background(), tickerData)
}
