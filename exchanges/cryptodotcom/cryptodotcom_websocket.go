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

const (
	publicMessageFilter  = "publicConnectionFilter"
	privateMessageFilter = "privateConnectionFilter"
)

// WsConnect creates a new websocket to public and private endpoints.
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	responseStream = make(chan SubscriptionRawData)
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	err := conn.Dial(ctx, &dialer, http.Header{}, nil)
	if err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsFunnelConnectionData(conn, false)

	if e.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			e.Websocket.GetWebsocketURL())
	}
	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PingMessage,
		Delay:             time.Second * 10,
	})
	return nil
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (e *Exchange) wsFunnelConnectionData(ws websocket.Connection, authenticated bool) {
	defer e.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- SubscriptionRawData{Data: resp.Raw, Authenticated: authenticated}
	}
}

func (e *Exchange) respondHeartbeat(ctx context.Context, conn websocket.Connection, resp *SubscriptionResponse) error {
	subscriptionInput := &SubscriptionInput{
		ID:     resp.ID,
		Code:   resp.Code,
		Method: publicRespondHeartbeat,
	}
	return conn.SendJSONMessage(ctx, request.UnAuth, subscriptionInput)
}

// WsAuthConnect represents an authenticated connection to a websocket server
func (e *Exchange) WsAuthConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", e.Name)
	}
	var authDialer gws.Dialer
	authDialer.ReadBufferSize = 8192
	authDialer.WriteBufferSize = 8192
	err := conn.Dial(ctx, &authDialer, http.Header{}, nil)
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", e.Name, cryptodotcomWebsocketUserAPI, err)
	}
	e.Websocket.Wg.Add(1)
	go e.wsFunnelConnectionData(conn, true)
	return nil
}

// AuthenticateWebsocketConnection authenticates the websocekt connection.
func (e *Exchange) AuthenticateWebsocketConnection(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := time.Now()
	req := &WsRequestPayload{
		ID:     e.MessageSequence(),
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
	payload, err = conn.SendMessageReturnResponse(ctx, request.UnAuth, req.ID, req)
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
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.handleSubscriptions(ctx, conn, "subscribe", subscriptions)
}

// Unsubscribe sends a websocket unsubscription to a channel message through the websocket connection handlers.
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.handleSubscriptions(ctx, conn, "unsubscribe", subscriptions)
}

// generateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (e *Exchange) generateDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := defaultSubscriptions
	if e.Websocket.CanUseAuthenticatedEndpoints() {
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
		enabledPairs, err := e.GetEnabledPairs(asset.Spot)
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

func (e *Exchange) handleSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subscriptions subscription.List) error {
	subscriptionPayloads, err := e.generatePayload(operation, subscriptions)
	if err != nil {
		return err
	}
	for p := range subscriptionPayloads {
		if subscriptionPayloads[p].Authenticated {
			if err := conn.SendJSONMessage(ctx, request.Auth, subscriptionPayloads[p]); err != nil {
				return err
			}
		} else {
			if err := conn.SendJSONMessage(ctx, request.UnAuth, subscriptionPayloads[p]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Exchange) generatePayload(operation string, subscriptions subscription.List) ([]SubscriptionPayload, error) {
	subscriptionPayloads := make([]SubscriptionPayload, len(subscriptions))
	timestamp := time.Now()
	for x := range subscriptions {
		subscriptionPayloads[x] = SubscriptionPayload{
			ID:     e.MessageSequence(),
			Method: operation,
			Nonce:  timestamp.UnixMilli(),
		}
		switch subscriptions[x].Channel {
		case instrumentOrderbookCnl,
			tickerCnl,
			tradeCnl,
			otcBooksCnl,
			fundingCnl,
			settlementCnl,
			markCnl,
			indexCnl:
			for p := range subscriptions[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscriptions[x].Channel + "." + subscriptions[x].Pairs[p].String()}}
			}
		case positionBalanceCnl,
			accountRiskCnl,
			userPositionsCnl,
			userOrderCnl,
			userTradeCnl,
			userBalanceCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {subscriptions[x].Channel}}
		case candlestickCnl:
			interval, err := intervalToString(subscriptions[x].Interval)
			if err != nil {
				return nil, err
			}
			for p := range subscriptions[x].Pairs {
				subscriptionPayloads[x].Params = map[string][]string{"channels": {subscriptions[x].Channel + "." + interval + "." + subscriptions[x].Pairs[p].String()}}
			}
		}
		switch subscriptions[x].Channel {
		case userOrderCnl, userTradeCnl, userBalanceCnl:
			subscriptionPayloads[x].Authenticated = true
		}
	}
	return subscriptionPayloads, nil
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (e *Exchange) WsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var resp *SubscriptionResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}

	if resp.ID > 0 && resp.Method == publicHeartbeat {
		return e.respondHeartbeat(ctx, conn, resp)
	}
	if resp.Result == nil {
		resp.Result = &WsResult{}
	}
	if resp.Method == "subscribe" {
		switch resp.Result.Channel {
		case userBalanceCnl:
			return e.processUserBalance(ctx, resp.Result)
		case instrumentOrderbookCnl:
			return e.processOrderbook(resp.Result)
		case tickerCnl:
			return e.processTicker(ctx, resp.Result)
		case tradeCnl:
			return e.processTrades(resp.Result)
		case candlestickCnl:
			return e.processCandlestick(ctx, resp.Result)
		case otcBooksCnl:
			return e.processOTCOrderbook(resp.Result)
		case positionBalanceCnl:
			return e.processPositionBalance(ctx, resp.Result)
		case accountRiskCnl:
			return e.processAccountRisk(ctx, resp.Result)
		case userPositionsCnl:
			return e.processUserPosition(ctx, resp.Result)
		case fundingCnl:
			return e.processFundingRate(ctx, resp.Result)
		case settlementCnl, markCnl, indexCnl:
			return e.Websocket.DataHandler.Send(ctx, resp)
		default:
			if strings.HasPrefix(resp.Result.Channel, userOrderCnl) {
				return e.processUserOrders(ctx, resp.Result)
			} else if strings.HasPrefix(resp.Result.Channel, userTradeCnl) {
				if !e.IsFillsFeedEnabled() {
					return nil
				}
				return e.processUserTrade(resp.Result)
			}
			if resp.Code == 0 {
				return nil
			}
		}
	}
	if !conn.IncomingWithData(resp.ID, respRaw) {
		return fmt.Errorf("could not match incoming message with signature: %d, and data: %s", resp.ID, string(respRaw))
	}
	return nil
}

func (e *Exchange) processFundingRate(ctx context.Context, resp *WsResult) error {
	var data []ValueAndTimestamp
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	for d := range data {
		if err := e.Websocket.DataHandler.Send(ctx, websocket.FundingData{
			Timestamp:    data[d].Timestamp.Time(),
			CurrencyPair: resp.InstrumentName,
			AssetType:    asset.PerpetualSwap,
			Exchange:     e.Name,
			Rate:         data[d].Value.Float64(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processAccountRisk(ctx context.Context, resp *WsResult) error {
	var data []UserAccountRisk
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	for x := range data {
		changes := make([]accounts.Change, len(data[x].Balances))
		for b := range data[x].Balances {
			changes[b] = accounts.Change{
				AssetType: asset.PerpetualSwap,
				Balance: accounts.Balance{
					Currency: currency.NewCode(data[x].Balances[b].Currency),
					Total:    data[x].Balances[b].Quantity.Float64(),
					Hold:     data[x].Balances[b].ReservedQty.Float64(),
					Free:     data[x].Balances[b].Quantity.Float64() - data[x].Balances[b].ReservedQty.Float64(),
				},
			}
		}
		positions := make([]order.Detail, len(data[x].Positions))
		for p := range data[x].Positions {
			positions[p] = order.Detail{
				Exchange:       e.Name,
				AssetType:      asset.PerpetualSwap,
				Leverage:       data[x].Positions[p].TargetLeverage.Float64(),
				Price:          data[x].Positions[p].MarkPrice.Float64(),
				Amount:         data[x].Positions[p].Quantity.Float64(),
				ContractAmount: data[x].Positions[p].Quantity.Float64(),
				Cost:           data[x].Positions[p].Cost.Float64(),
				Pair:           data[x].Positions[p].InstrumentName,
				AccountID:      data[x].Positions[p].AccountID,
				LastUpdated:    data[x].Positions[p].UpdateTimestampMs.Time(),
			}
		}
		if err := e.Websocket.DataHandler.Send(ctx, positions); err != nil {
			return err
		}
		if err := e.Websocket.DataHandler.Send(ctx, changes); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processPositionBalance(ctx context.Context, resp *WsResult) error {
	var data *WsUserPositionBalance
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	positions := make([]order.Detail, len(data.Positions))
	for p := range data.Positions {
		positions[p] = order.Detail{
			Leverage:       data.Positions[p].TargetLeverage.Float64(),
			Price:          data.Positions[p].MarkPrice.Float64(),
			Amount:         data.Positions[p].Quantity.Float64(),
			ContractAmount: data.Positions[p].Quantity.Float64(),
			Cost:           data.Positions[p].Cost.Float64(),
			Exchange:       e.Name,
			AccountID:      data.Positions[p].AccountID,
			AssetType:      asset.PerpetualSwap,
			LastUpdated:    data.Positions[p].UpdateTimestampMs.Time(),
			Pair:           data.Positions[p].InstrumentName,
		}
	}
	if err := e.Websocket.DataHandler.Send(ctx, positions); err != nil {
		return err
	}
	changes := make([]accounts.Change, len(data.Balances))
	for b := range data.Balances {
		changes[b] = accounts.Change{
			AssetType: asset.PerpetualSwap,
			Balance: accounts.Balance{
				Currency: currency.NewCode(data.Balances[b].CurrencyName),
				Total:    data.Balances[b].Quantity.Float64(),
			},
		}
	}
	return e.Websocket.DataHandler.Send(ctx, changes)
}

func (e *Exchange) processUserPosition(ctx context.Context, resp *WsResult) error {
	var data []UserPosition
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	orders := make([]order.Detail, len(data))
	for x := range data {
		var assetType asset.Item
		if data[x].InstrumentType == "PERPETUAL_SWAP" {
			assetType = asset.Futures
		}
		orders[x] = order.Detail{
			Leverage:    data[x].TargetLeverage.Float64(),
			Price:       data[x].MarkPrice.Float64(),
			Amount:      data[x].Quantity.Float64(),
			Cost:        data[x].Cost.Float64(),
			Exchange:    e.Name,
			OrderID:     data[x].AccountID,
			AccountID:   data[x].AccountID,
			AssetType:   assetType,
			LastUpdated: data[x].UpdateTimestampMs.Time(),
			Pair:        data[x].InstrumentName,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, orders)
}

func (e *Exchange) processOTCOrderbook(resp *WsResult) error {
	var data []OTCBook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	for x := range data {
		book := orderbook.Book{
			Exchange:          e.Name,
			Asset:             asset.OTC,
			ValidateOrderbook: e.ValidateOrderbook,
			Pair:              resp.InstrumentName,
			LastUpdated:       resp.Timestamp.Time(),
		}
		book.Asks = make([]orderbook.Level, len(data[x].Asks))
		for i := range data[x].Asks {
			book.Asks[i].Price = data[x].Asks[i][0].Float64()
			book.Asks[i].Amount = data[x].Asks[i][1].Float64()
		}
		book.Bids = make([]orderbook.Level, len(data[x].Bids))
		for i := range data[x].Bids {
			book.Bids[i].Price = data[x].Bids[i][0].Float64()
			book.Bids[i].Amount = data[x].Bids[i][1].Float64()
		}
		book.Asks.SortAsks()
		book.Bids.SortBids()
		err = e.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processCandlestick(ctx context.Context, resp *WsResult) error {
	var data []WsCandlestickItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	interval, err := stringToInterval(resp.Interval)
	if err != nil {
		return err
	}
	candles := make([]kline.Item, len(data))
	for x := range data {
		candles[x] = kline.Item{
			Exchange: e.Name,
			Interval: interval,
			Asset:    asset.Spot,
			Pair:     resp.InstrumentName,
			Candles: []kline.Candle{
				{
					Open:   data[x].Open,
					High:   data[x].High,
					Low:    data[x].Low,
					Volume: data[x].Volume,
					Time:   data[x].UpdateTime.Time(),
				},
			},
		}
	}
	return e.Websocket.DataHandler.Send(ctx, candles)
}

func (e *Exchange) processTrades(resp *WsResult) error {
	var data []TradeItem
	err := json.Unmarshal(resp.Data, &data)
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
			CurrencyPair: resp.InstrumentName,
			Exchange:     e.Name,
			Side:         oSide,
			Timestamp:    data[i].TradeTimestamp.Time(),
			TID:          data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

func (e *Exchange) processTicker(ctx context.Context, resp *WsResult) error {
	var data []TickerItem
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	tickersDatas := make([]ticker.Price, len(data))
	for x := range data {
		tickersDatas[x] = ticker.Price{
			ExchangeName: e.Name,
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
			Pair:         resp.InstrumentName,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, tickersDatas)
}

func (e *Exchange) processOrderbook(resp *WsResult) error {
	var data []WsOrderbook
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	for x := range data {
		book := orderbook.Book{
			Exchange:          e.Name,
			Asset:             asset.Spot,
			LastUpdated:       data[x].OrderbookUpdateTime.Time(),
			LastUpdateID:      data[x].UpdateSequence,
			Pair:              resp.InstrumentName,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		book.Asks = make([]orderbook.Level, len(data[x].Asks))
		for i := range data[x].Asks {
			book.Asks[i].Price = data[x].Asks[i][0].Float64()
			book.Asks[i].Amount = data[x].Asks[i][1].Float64()
		}
		book.Bids = make([]orderbook.Level, len(data[x].Bids))
		for i := range data[x].Bids {
			book.Bids[i].Price = data[x].Bids[i][0].Float64()
			book.Bids[i].Amount = data[x].Bids[i][1].Float64()
		}
		book.Asks.SortAsks()
		book.Bids.SortBids()
		err = e.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processUserBalance(ctx context.Context, wsResult *WsResult) error {
	var resp []UserBalanceDetail
	err := json.Unmarshal(wsResult.Data, &resp)
	if err != nil {
		return err
	}
	accountChanges := make([]accounts.Change, len(resp))
	for x := range resp {
		accountChanges[x] = accounts.Change{
			Balance: accounts.Balance{
				Currency: currency.NewCode(resp[x].InstrumentName),
				Total:    resp[x].TotalCashBalance.Float64(),
				Hold:     resp[x].TotalCashBalance.Float64() - resp[x].TotalAvailableBalance.Float64(),
				Free:     resp[x].TotalAvailableBalance.Float64(),
			},
		}
	}
	return e.Websocket.DataHandler.Send(ctx, accountChanges)
}

func (e *Exchange) processUserTrade(resp *WsResult) error {
	var data []UserTrade
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(data))
	for x := range data {
		oSide, err := order.StringToOrderSide(data[x].Side)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			ID:           data[x].OrderID,
			Timestamp:    data[x].CreateTime.Time(),
			Exchange:     e.Name,
			AssetType:    asset.Spot,
			Side:         oSide,
			OrderID:      data[x].OrderID,
			TradeID:      data[x].TradeID,
			CurrencyPair: resp.InstrumentName,
			Price:        data[x].TradedPrice.Float64(),
			Amount:       data[x].TradedQuantity.Float64(),
		}
	}
	return e.Websocket.Fills.Update(fills...)
}

func (e *Exchange) processUserOrders(ctx context.Context, resp *WsResult) error {
	var data []UserOrder
	err := json.Unmarshal(resp.Data, &data)
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
		case tifPOSTONLY:
			tif = order.PostOnly
		case tifGTC:
			tif = order.GoodTillCancel
		}
		ordersDetails[x] = order.Detail{
			Amount:               data[x].Quantity.Float64(),
			AverageExecutedPrice: data[x].AvgPrice.Float64(),
			RemainingAmount:      data[x].Quantity.Float64() - data[x].CumulativeExecutedQuantity.Float64(),
			ExecutedAmount:       data[x].CumulativeExecutedQuantity.Float64(),
			Cost:                 data[x].CumulativeExecutedValue.Float64(),
			FeeAsset:             currency.NewCode(data[x].FeeCurrency),
			CostAsset:            resp.InstrumentName.Quote,
			Pair:                 resp.InstrumentName,
			LastUpdated:          data[x].UpdateTime.Time(),
			Date:                 data[x].CreateTime.Time(),
			Price:                data[x].Price.Float64(),
			ClientOrderID:        data[x].ClientOrderID,
			OrderID:              data[x].OrderID,
			AssetType:            asset.Spot,
			Exchange:             e.Name,
			Side:                 oSide,
			Type:                 oType,
			Status:               status,
			TimeInForce:          tif,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, ordersDetails)
}
