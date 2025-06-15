package cryptodotcom

import (
	"context"
	"encoding/hex"
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
	userOrderCnl       = "user.order" // user.order.{instrument_name}
	userTradeCnl       = "user.trade" // user.trade.{instrument_name}
	userBalanceCnl     = "user.balance"
	positionBalanceCnl = "user.position_balance"
	accountRiskCnl     = "user.account_risk"
	userPositionsCnl   = "user.positions"

	// public subscription channels
	instrumentOrderbookCnl = "book"        // book.{instrument_name} or book.{instrument_name}.{depth}user.position_balance
	tickerCnl              = "ticker"      // ticker.{instrument_name}
	tradeCnl               = "trade"       // trade.{instrument_name}
	candlestickCnl         = "candlestick" // candlestick.{time_frame}.{instrument_name}
	otcBooksCnl            = "otc_book"    // otc_book.{instrument_name}
	fundingCnl             = "funding"     // funding.{instrument_name}
	settlementCnl          = "settlement"
	markCnl                = "mark"
	indexCnl               = "index"
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
	req.Signature = hex.EncodeToString(hmac)
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
			tradeCnl,
			otcBooksCnl,
			fundingCnl,
			settlementCnl,
			markCnl,
			indexCnl:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
				Pairs:   enabledPairs,
			})
		case positionBalanceCnl,
			accountRiskCnl,
			userPositionsCnl,
			userOrderCnl,
			userTradeCnl,
			userBalanceCnl:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
			})
		case candlestickCnl:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:  channels[x],
				Pairs:    enabledPairs,
				Interval: kline.FiveMin,
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
		case instrumentOrderbookCnl,
			tickerCnl,
			tradeCnl,
			otcBooksCnl,
			fundingCnl,
			settlementCnl,
			markCnl,
			indexCnl:
			for p := range subscription[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel + "." + subscription[x].Pairs[p].String()}}
			}
		case positionBalanceCnl,
			accountRiskCnl,
			userPositionsCnl,
			userOrderCnl,
			userTradeCnl,
			userBalanceCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel}}
		case candlestickCnl:
			interval, err := intervalToString(subscription[x].Interval)
			if err != nil {
				return nil, err
			}
			for p := range subscription[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel + "." + interval + "." + subscription[x].Pairs[p].String()}}
			}
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
		switch resp.Result.Channel {
		case userBalanceCnl:
			return cr.processUserBalance(resp.Result)
		case instrumentOrderbookCnl:
			return cr.processOrderbook(resp.Result)
		case tickerCnl:
			return cr.processTicker(resp.Result)
		case tradeCnl:
			return cr.processTrades(resp.Result)
		case candlestickCnl:
			return cr.processCandlestick(resp.Result)
		case otcBooksCnl:
			return cr.processOTCOrderbook(resp.Result)
		case positionBalanceCnl:
			return cr.processPositionBalance(resp.Result)
		case accountRiskCnl:
			return cr.processAccountRisk(resp.Result)
		case userPositionsCnl:
			return cr.processUserPosition(resp.Result)
		case fundingCnl:
			return cr.processFundingRate(resp.Result)
		case settlementCnl, markCnl, indexCnl:
			cr.Websocket.DataHandler <- resp
			return nil
		default:
			if strings.HasPrefix(resp.Result.Channel, userOrderCnl) {
				return cr.processUserOrders(resp.Result)
			} else if strings.HasPrefix(resp.Result.Channel, userTradeCnl) {
				if !cr.IsFillsFeedEnabled() {
					return nil
				}
				return cr.processUserTrade(resp.Result)
			}
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

func (cr *Cryptodotcom) processFundingRate(resp *WsResult) error {
	var data []ValueAndTimestamp
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.InstrumentName)
	if err != nil {
		return err
	}
	for d := range data {
		cr.Websocket.DataHandler <- websocket.FundingData{
			Timestamp:    data[d].Timestamp.Time(),
			CurrencyPair: cp,
			AssetType:    asset.PerpetualSwap,
			Exchange:     cr.Name,
			Rate:         data[d].Value.Float64(),
		}
	}
	return nil
}

func (cr *Cryptodotcom) processAccountRisk(resp *WsResult) error {
	var data []UserAccountRisk
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	for x := range data {
		changes := make([]account.Change, len(data[x].Balances))
		for b := range data[x].Balances {
			changes[b] = account.Change{
				AssetType: asset.PerpetualSwap,
				Balance: &account.Balance{
					Currency: currency.NewCode(data[x].Balances[b].Currency),
					Total:    data[x].Balances[b].Quantity.Float64(),
					Hold:     data[x].Balances[b].ReservedQty.Float64(),
					Free:     data[x].Balances[b].Quantity.Float64() - data[x].Balances[b].ReservedQty.Float64(),
				},
			}
		}
		positions := make([]order.Detail, len(data[x].Positions))
		for p := range data[x].Positions {
			cp, err := currency.NewPairFromString(data[x].Positions[p].InstrumentName)
			if err != nil {
				return err
			}
			positions[p] = order.Detail{
				Leverage:       data[x].Positions[p].TargetLeverage.Float64(),
				Price:          data[x].Positions[p].MarkPrice.Float64(),
				Amount:         data[x].Positions[p].Quantity.Float64(),
				ContractAmount: data[x].Positions[p].Quantity.Float64(),
				Cost:           data[x].Positions[p].Cost.Float64(),
				Exchange:       cr.Name,
				AccountID:      data[x].Positions[p].AccountID,
				AssetType:      asset.PerpetualSwap,
				LastUpdated:    data[x].Positions[p].UpdateTimestampMs.Time(),
				Pair:           cp,
			}
		}
		cr.Websocket.DataHandler <- positions
		cr.Websocket.DataHandler <- changes
	}
	return nil
}

func (cr *Cryptodotcom) processPositionBalance(resp *WsResult) error {
	var data *WsUserPositionBalance
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	positions := make([]order.Detail, len(data.Positions))
	for p := range data.Positions {
		cp, err := currency.NewPairFromString(data.Positions[p].InstrumentName)
		if err != nil {
			return err
		}
		positions[p] = order.Detail{
			Leverage:       data.Positions[p].TargetLeverage.Float64(),
			Price:          data.Positions[p].MarkPrice.Float64(),
			Amount:         data.Positions[p].Quantity.Float64(),
			ContractAmount: data.Positions[p].Quantity.Float64(),
			Cost:           data.Positions[p].Cost.Float64(),
			Exchange:       cr.Name,
			AccountID:      data.Positions[p].AccountID,
			AssetType:      asset.PerpetualSwap,
			LastUpdated:    data.Positions[p].UpdateTimestampMs.Time(),
			Pair:           cp,
		}
	}
	cr.Websocket.DataHandler <- positions
	changes := make([]account.Change, len(data.Balances))
	for b := range data.Balances {
		changes[b] = account.Change{
			AssetType: asset.PerpetualSwap,
			Balance: &account.Balance{
				Currency: currency.NewCode(data.Balances[b].CurrencyName),
				Total:    data.Balances[b].Quantity.Float64(),
			},
		}
	}
	cr.Websocket.DataHandler <- changes
	return nil
}

func (cr *Cryptodotcom) processUserPosition(resp *WsResult) error {
	var data []UserPosition
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	orders := make([]order.Detail, len(data))
	for x := range data {
		cp, err := currency.NewPairFromString(data[x].InstrumentName)
		if err != nil {
			return err
		}
		var assetType asset.Item
		if data[x].InstrumentType == "PERPETUAL_SWAP" {
			assetType = asset.Futures
		}
		orders[x] = order.Detail{
			Leverage:    data[x].TargetLeverage.Float64(),
			Price:       data[x].MarkPrice.Float64(),
			Amount:      data[x].Quantity.Float64(),
			Cost:        data[x].Cost.Float64(),
			Exchange:    cr.Name,
			OrderID:     data[x].AccountID,
			AccountID:   data[x].AccountID,
			AssetType:   assetType,
			LastUpdated: data[x].UpdateTimestampMs.Time(),
			Pair:        cp,
		}
	}
	cr.Websocket.DataHandler <- orders
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

func (cr *Cryptodotcom) processUserBalance(wsResult *WsResult) error {
	var resp []UserBalanceDetail
	err := json.Unmarshal(wsResult.Data, &resp)
	if err != nil {
		return err
	}
	accountChanges := make([]account.Change, len(resp))
	for x := range resp {
		accountChanges[x] = account.Change{
			Balance: &account.Balance{
				Currency: currency.NewCode(resp[x].InstrumentName),
				Total:    resp[x].TotalCashBalance.Float64(),
				Hold:     resp[x].TotalCashBalance.Float64() - resp[x].TotalAvailableBalance.Float64(),
				Free:     resp[x].TotalAvailableBalance.Float64(),
			},
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
			Price:        data[x].TradedPrice.Float64(),
			Amount:       data[x].TradedQuantity.Float64(),
		}
	}
	return cr.Websocket.Fills.Update(fills...)
}

func (cr *Cryptodotcom) processUserOrders(resp *WsResult) error {
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
		var tif order.TimeInForce
		switch data[x].TimeInForce {
		case "POST_ONLY":
			tif = order.PostOnly
		case "GOOD_TILL_CANCEL":
			tif = order.GoodTillCancel
		}
		ordersDetails[x] = order.Detail{
			Amount:               data[x].Quantity.Float64(),
			AverageExecutedPrice: data[x].AvgPrice.Float64(),
			RemainingAmount:      data[x].Quantity.Float64() - data[x].CumulativeExecutedQuantity.Float64(),
			ExecutedAmount:       data[x].CumulativeExecutedQuantity.Float64(),
			Cost:                 data[x].CumulativeExecutedValue.Float64(),
			FeeAsset:             currency.NewCode(data[x].FeeCurrency),
			LastUpdated:          data[x].UpdateTime.Time(),
			Date:                 data[x].CreateTime.Time(),
			Price:                data[x].Price.Float64(),
			ClientOrderID:        data[x].ClientOrderID,
			OrderID:              data[x].OrderID,
			AssetType:            asset.Spot,
			CostAsset:            cp.Quote,
			Exchange:             cr.Name,
			Side:                 oSide,
			Type:                 oType,
			Status:               status,
			TimeInForce:          tif,
			Pair:                 cp,
		}
	}
	cr.Websocket.DataHandler <- ordersDetails
	return nil
}
