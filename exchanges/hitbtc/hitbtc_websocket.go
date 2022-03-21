package hitbtc

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
	rateLimit              = 20
	errAuthFailed          = 1002
)

// WsConnect starts a new connection with the websocket API
func (h *HitBTC) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := h.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	h.Websocket.Wg.Add(1)
	go h.wsReadData()

	err = h.wsLogin(context.TODO())
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", h.Name, err)
	}

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (h *HitBTC) wsReadData() {
	defer h.Websocket.Wg.Done()

	for {
		resp := h.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		err := h.wsHandleData(resp.Raw)
		if err != nil {
			h.Websocket.DataHandler <- err
		}
	}
}

func (h *HitBTC) wsGetTableName(respRaw []byte) (string, error) {
	var init capture
	err := json.Unmarshal(respRaw, &init)
	if err != nil {
		return "", err
	}
	if init.Error.Code == errAuthFailed {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if init.ID > 0 {
		if h.Websocket.Match.IncomingWithData(init.ID, respRaw) {
			return "", nil
		}
	}
	if init.Error.Message != "" || init.Error.Code != 0 {
		return "", fmt.Errorf("code: %d, Message: %s",
			init.Error.Code,
			init.Error.Message)
	}
	if _, ok := init.Result.(bool); ok {
		return "", nil
	}
	if init.Method != "" {
		return init.Method, nil
	}
	switch resultType := init.Result.(type) {
	case map[string]interface{}:
		if reportType, ok := resultType["reportType"].(string); ok {
			return reportType, nil
		}
		// check for ids - means it was a specific request
		// and can't go through normal processing
		if responseID, ok := resultType["id"].(string); ok {
			if responseID != "" {
				return "", nil
			}
		}
	case []interface{}:
		if len(resultType) == 0 {
			h.Websocket.DataHandler <- fmt.Sprintf("No data returned. ID: %v", init.ID)
			return "", nil
		}

		data, ok := resultType[0].(map[string]interface{})
		if !ok {
			return "", errors.New("unable to type assert data")
		}
		if _, ok := data["clientOrderId"]; ok {
			return "order", nil
		} else if _, ok := data["available"]; ok {
			return "trading", nil
		}
	}
	h.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: h.Name + stream.UnhandledMessage + string(respRaw)}
	return "", nil
}

func (h *HitBTC) wsHandleData(respRaw []byte) error {
	name, err := h.wsGetTableName(respRaw)
	if err != nil {
		return err
	}
	switch name {
	case "":
		return nil
	case "ticker":
		var wsTicker WsTicker
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		pairs, err := h.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}

		format, err := h.GetPairFormat(asset.Spot, true)
		if err != nil {
			return err
		}

		p, err := currency.NewPairFromFormattedPairs(wsTicker.Params.Symbol,
			pairs,
			format)
		if err != nil {
			return err
		}

		h.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: h.Name,
			Open:         wsTicker.Params.Open,
			Volume:       wsTicker.Params.Volume,
			QuoteVolume:  wsTicker.Params.VolumeQuote,
			High:         wsTicker.Params.High,
			Low:          wsTicker.Params.Low,
			Bid:          wsTicker.Params.Bid,
			Ask:          wsTicker.Params.Ask,
			Last:         wsTicker.Params.Last,
			LastUpdated:  wsTicker.Params.Timestamp,
			AssetType:    asset.Spot,
			Pair:         p,
		}
	case "snapshotOrderbook":
		var obSnapshot WsOrderbook
		err := json.Unmarshal(respRaw, &obSnapshot)
		if err != nil {
			return err
		}
		err = h.WsProcessOrderbookSnapshot(obSnapshot)
		if err != nil {
			return err
		}
	case "updateOrderbook":
		var obUpdate WsOrderbook
		err := json.Unmarshal(respRaw, &obUpdate)
		if err != nil {
			return err
		}
		err = h.WsProcessOrderbookUpdate(obUpdate)
		if err != nil {
			return err
		}
	case "snapshotTrades", "updateTrades":
		if !h.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeSnapshot WsTrade
		err := json.Unmarshal(respRaw, &tradeSnapshot)
		if err != nil {
			return err
		}
		var trades []trade.Data
		p, err := currency.NewPairFromString(tradeSnapshot.Params.Symbol)
		if err != nil {
			return &order.ClassificationError{
				Exchange: h.Name,
				Err:      err,
			}
		}
		for i := range tradeSnapshot.Params.Data {
			side, err := order.StringToOrderSide(tradeSnapshot.Params.Data[i].Side)
			if err != nil {
				return &order.ClassificationError{
					Exchange: h.Name,
					Err:      err,
				}
			}
			trades = append(trades, trade.Data{
				Timestamp:    tradeSnapshot.Params.Data[i].Timestamp,
				Exchange:     h.Name,
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Price:        tradeSnapshot.Params.Data[i].Price,
				Amount:       tradeSnapshot.Params.Data[i].Quantity,
				Side:         side,
				TID:          strconv.FormatInt(tradeSnapshot.Params.Data[i].ID, 10),
			})
		}
		return trade.AddTradesToBuffer(h.Name, trades...)
	case "activeOrders":
		var o wsActiveOrdersResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		for i := range o.Params {
			err = h.wsHandleOrderData(&o.Params[i])
			if err != nil {
				return err
			}
		}
	case "trading":
		var trades WsGetTradingBalanceResponse
		err := json.Unmarshal(respRaw, &trades)
		if err != nil {
			return err
		}
		h.Websocket.DataHandler <- trades
	case "report":
		var o wsReportResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		err = h.wsHandleOrderData(&o.OrderData)
		if err != nil {
			return err
		}
	case "order":
		var o wsActiveOrderRequestResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		for i := range o.OrderData {
			err = h.wsHandleOrderData(&o.OrderData[i])
			if err != nil {
				return err
			}
		}
	case
		"replaced",
		"canceled",
		"new":
		var o wsOrderResponse
		err := json.Unmarshal(respRaw, &o)
		if err != nil {
			return err
		}
		err = h.wsHandleOrderData(&o.OrderData)
		if err != nil {
			return err
		}
	default:
		h.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: h.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

// WsProcessOrderbookSnapshot processes a full orderbook snapshot to a local cache
func (h *HitBTC) WsProcessOrderbookSnapshot(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 || len(ob.Params.Ask) == 0 {
		return errors.New("no orderbooks to process")
	}

	var newOrderBook orderbook.Base
	for i := range ob.Params.Bid {
		newOrderBook.Bids = append(newOrderBook.Bids, orderbook.Item{
			Amount: ob.Params.Bid[i].Size,
			Price:  ob.Params.Bid[i].Price,
		})
	}

	for i := range ob.Params.Ask {
		newOrderBook.Asks = append(newOrderBook.Asks, orderbook.Item{
			Amount: ob.Params.Ask[i].Size,
			Price:  ob.Params.Ask[i].Price,
		})
	}

	pairs, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := h.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(ob.Params.Symbol,
		pairs,
		format)
	if err != nil {
		h.Websocket.DataHandler <- err
		return err
	}
	newOrderBook.Asset = asset.Spot
	newOrderBook.Pair = p
	newOrderBook.Exchange = h.Name
	newOrderBook.VerifyOrderbook = h.CanVerifyOrderbook

	return h.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

func (h *HitBTC) wsHandleOrderData(o *wsOrderData) error {
	var trades []order.TradeHistory
	if o.TradeID > 0 {
		trades = append(trades, order.TradeHistory{
			Price:     o.TradePrice,
			Amount:    o.TradeQuantity,
			Fee:       o.TradeFee,
			Exchange:  h.Name,
			TID:       strconv.FormatFloat(o.TradeID, 'f', -1, 64),
			Timestamp: o.UpdatedAt,
		})
	}
	oType, err := order.StringToOrderType(o.Type)
	if err != nil {
		h.Websocket.DataHandler <- order.ClassificationError{
			Exchange: h.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}
	o.Status = strings.Replace(o.Status, "canceled", "cancelled", 1)
	oStatus, err := order.StringToOrderStatus(o.Status)
	if err != nil {
		h.Websocket.DataHandler <- order.ClassificationError{
			Exchange: h.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}
	oSide, err := order.StringToOrderSide(o.Side)
	if err != nil {
		h.Websocket.DataHandler <- order.ClassificationError{
			Exchange: h.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}

	p, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		h.Websocket.DataHandler <- order.ClassificationError{
			Exchange: h.Name,
			OrderID:  o.ID,
			Err:      err,
		}
	}

	var a asset.Item
	a, err = h.GetPairAssetType(p)
	if err != nil {
		return err
	}
	h.Websocket.DataHandler <- &order.Detail{
		Price:           o.Price,
		Amount:          o.Quantity,
		ExecutedAmount:  o.CumQuantity,
		RemainingAmount: o.Quantity - o.CumQuantity,
		Exchange:        h.Name,
		ID:              o.ID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       a,
		Date:            o.CreatedAt,
		LastUpdated:     o.UpdatedAt,
		Pair:            p,
		Trades:          trades,
	}
	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (h *HitBTC) WsProcessOrderbookUpdate(update WsOrderbook) error {
	if len(update.Params.Bid) == 0 && len(update.Params.Ask) == 0 {
		// Periodically HitBTC sends empty updates which includes a sequence
		// can return this as nil.
		return nil
	}

	var bids, asks []orderbook.Item
	for i := range update.Params.Bid {
		bids = append(bids, orderbook.Item{
			Price:  update.Params.Bid[i].Price,
			Amount: update.Params.Bid[i].Size,
		})
	}

	for i := range update.Params.Ask {
		asks = append(asks, orderbook.Item{
			Price:  update.Params.Ask[i].Price,
			Amount: update.Params.Ask[i].Size,
		})
	}

	pairs, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := h.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(update.Params.Symbol,
		pairs,
		format)
	if err != nil {
		return err
	}

	return h.Websocket.Orderbook.Update(&buffer.Update{
		Asks:     asks,
		Bids:     bids,
		Pair:     p,
		UpdateID: update.Params.Sequence,
		Asset:    asset.Spot,
	})
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HitBTC) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"subscribeTicker",
		"subscribeOrderbook",
		"subscribeTrades",
		"subscribeCandles"}

	var subscriptions []stream.ChannelSubscription
	if h.Websocket.CanUseAuthenticatedEndpoints() {
		subscriptions = append(subscriptions, stream.ChannelSubscription{
			Channel: "subscribeReports",
		})
	}
	enabledCurrencies, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		for j := range enabledCurrencies {
			fpair, err := h.FormatExchangeCurrency(enabledCurrencies[j], asset.Spot)
			if err != nil {
				return nil, err
			}

			enabledCurrencies[j].Delimiter = ""
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: fpair,
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HitBTC) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToSubscribe {
		subscribe := WsRequest{
			Method: channelsToSubscribe[i].Channel,
			ID:     h.Websocket.Conn.GenerateMessageID(false),
		}

		if channelsToSubscribe[i].Currency.String() != "" {
			subscribe.Params.Symbol = channelsToSubscribe[i].Currency.String()
		}
		if strings.EqualFold(channelsToSubscribe[i].Channel, "subscribeTrades") {
			subscribe.Params.Limit = 100
		} else if strings.EqualFold(channelsToSubscribe[i].Channel, "subscribeCandles") {
			subscribe.Params.Period = "M30"
			subscribe.Params.Limit = 100
		}

		err := h.Websocket.Conn.SendJSONMessage(subscribe)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		h.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HitBTC) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToUnsubscribe {
		unsubscribeChannel := strings.Replace(channelsToUnsubscribe[i].Channel,
			"subscribe",
			"unsubscribe",
			1)

		unsubscribe := WsNotification{
			JSONRPCVersion: rpcVersion,
			Method:         unsubscribeChannel,
		}

		unsubscribe.Params.Symbol = channelsToUnsubscribe[i].Currency.String()
		if strings.EqualFold(unsubscribeChannel, "unsubscribeTrades") {
			unsubscribe.Params.Limit = 100
		} else if strings.EqualFold(unsubscribeChannel, "unsubscribeCandles") {
			unsubscribe.Params.Period = "M30"
			unsubscribe.Params.Limit = 100
		}

		err := h.Websocket.Conn.SendJSONMessage(unsubscribe)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		h.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HitBTC) wsLogin(ctx context.Context) error {
	if !h.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", h.Name)
	}
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return err
	}
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	n := strconv.FormatInt(time.Now().Unix(), 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(n),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	request := WsLoginRequest{
		Method: "login",
		Params: WsLoginData{
			Algo:      "HS256",
			PKey:      creds.Key,
			Nonce:     n,
			Signature: crypto.HexEncodeToString(hmac),
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}

	err = h.Websocket.Conn.SendJSONMessage(request)
	if err != nil {
		h.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}

	return nil
}

// wsPlaceOrder sends a websocket message to submit an order
func (h *HitBTC) wsPlaceOrder(pair currency.Pair, side string, price, quantity float64) (*WsSubmitOrderSuccessResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}

	id := h.Websocket.Conn.GenerateMessageID(false)
	fpair, err := h.FormatExchangeCurrency(pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	request := WsSubmitOrderRequest{
		Method: "newOrder",
		Params: WsSubmitOrderRequestData{
			ClientOrderID: id,
			Symbol:        fpair.String(),
			Side:          strings.ToLower(side),
			Price:         price,
			Quantity:      quantity,
		},
		ID: id,
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(id, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsSubmitOrderSuccessResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsCancelOrder sends a websocket message to cancel an order
func (h *HitBTC) wsCancelOrder(clientOrderID string) (*WsCancelOrderResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsCancelOrderRequest{
		Method: "cancelOrder",
		Params: WsCancelOrderRequestData{
			ClientOrderID: clientOrderID,
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsCancelOrderResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsReplaceOrder sends a websocket message to replace an order
func (h *HitBTC) wsReplaceOrder(clientOrderID string, quantity, price float64) (*WsReplaceOrderResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "cancelReplaceOrder",
		Params: WsReplaceOrderRequestData{
			ClientOrderID:   clientOrderID,
			RequestClientID: strconv.FormatInt(time.Now().Unix(), 10),
			Quantity:        quantity,
			Price:           price,
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsReplaceOrderResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetActiveOrders sends a websocket message to get all active orders
func (h *HitBTC) wsGetActiveOrders() (*wsActiveOrdersResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot get active orders", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "getOrders",
		Params: WsReplaceOrderRequestData{},
		ID:     h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response wsActiveOrdersResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetTradingBalance sends a websocket message to get trading balance
func (h *HitBTC) wsGetTradingBalance() (*WsGetTradingBalanceResponse, error) {
	if !h.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authenticated, cannot place order", h.Name)
	}
	request := WsReplaceOrderRequest{
		Method: "getTradingBalance",
		Params: WsReplaceOrderRequestData{},
		ID:     h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetTradingBalanceResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetCurrencies sends a websocket message to get trading balance
func (h *HitBTC) wsGetCurrencies(currencyItem currency.Code) (*WsGetCurrenciesResponse, error) {
	request := WsGetCurrenciesRequest{
		Method: "getCurrency",
		Params: WsGetCurrenciesRequestParameters{
			Currency: currencyItem,
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetCurrenciesResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (h *HitBTC) wsGetSymbols(c currency.Pair) (*WsGetSymbolsResponse, error) {
	fpair, err := h.FormatExchangeCurrency(c, asset.Spot)
	if err != nil {
		return nil, err
	}

	request := WsGetSymbolsRequest{
		Method: "getSymbol",
		Params: WsGetSymbolsRequestParameters{
			Symbol: fpair.String(),
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetSymbolsResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}

// wsGetSymbols sends a websocket message to get trading balance
func (h *HitBTC) wsGetTrades(c currency.Pair, limit int64, sort, by string) (*WsGetTradesResponse, error) {
	fpair, err := h.FormatExchangeCurrency(c, asset.Spot)
	if err != nil {
		return nil, err
	}

	request := WsGetTradesRequest{
		Method: "getTrades",
		Params: WsGetTradesRequestParameters{
			Symbol: fpair.String(),
			Limit:  limit,
			Sort:   sort,
			By:     by,
		},
		ID: h.Websocket.Conn.GenerateMessageID(false),
	}
	resp, err := h.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	var response WsGetTradesResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, fmt.Errorf("%v %v", h.Name, err)
	}
	if response.Error.Code > 0 || response.Error.Message != "" {
		return &response, fmt.Errorf("%v Error:%v Message:%v", h.Name, response.Error.Code, response.Error.Message)
	}
	return &response, nil
}
