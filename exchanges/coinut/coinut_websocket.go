package coinut

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	coinutWebsocketURL       = "wss://wsapi.coinut.com"
	coinutWebsocketRateLimit = 30
)

var channels map[string]chan []byte

// NOTE for speed considerations
// wss://wsapi-as.coinut.com
// wss://wsapi-na.coinut.com
// wss://wsapi-eu.coinut.com

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

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)

	if !e.instrumentMap.IsLoaded() {
		_, err = e.WsGetInstruments(ctx)
		if err != nil {
			return err
		}
	}

	if e.IsWebsocketAuthenticationSupported() {
		if err = e.wsAuthenticate(ctx); err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorln(log.WebsocketMgr, e.Name+" "+err.Error())
		}
	}

	// define bi-directional communication
	channels = make(map[string]chan []byte)
	channels["hb"] = make(chan []byte, 1)

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()

	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		if strings.HasPrefix(string(resp.Raw), "[") {
			var incoming []wsResponse
			if err := json.Unmarshal(resp.Raw, &incoming); err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
				}
				continue
			}
			for i := range incoming {
				if incoming[i].Nonce > 0 {
					if e.Websocket.Match.IncomingWithData(incoming[i].Nonce, resp.Raw) {
						break
					}
				}
				individualJSON, err := json.Marshal(incoming[i])
				if err != nil {
					if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
						log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
					}
					continue
				}
				if err := e.wsHandleData(ctx, individualJSON); err != nil {
					if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
						log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
					}
				}
			}
		} else {
			if err := e.wsHandleData(ctx, resp.Raw); err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
				}
			}
		}
	}
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "[") {
		var orders []wsOrderContainer
		if err := json.Unmarshal(respRaw, &orders); err != nil {
			return err
		}
		for i := range orders {
			o, err := e.parseOrderContainer(&orders[i])
			if err != nil {
				return err
			}
			if err := e.Websocket.DataHandler.Send(ctx, o); err != nil {
				return err
			}
		}
		return nil
	}

	var incoming wsResponse
	err := json.Unmarshal(respRaw, &incoming)
	if err != nil {
		return err
	}
	if e.Websocket.Match.IncomingWithData(incoming.Nonce, respRaw) {
		return nil
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	switch incoming.Reply {
	case "hb":
		channels["hb"] <- respRaw
	case "user_balance":
		var userBalance WsUserBalanceResponse
		err := json.Unmarshal(respRaw, &userBalance)
		if err != nil {
			return err
		}
	case "user_open_orders":
		var openOrders WsUserOpenOrdersResponse
		err := json.Unmarshal(respRaw, &openOrders)
		if err != nil {
			return err
		}
	case "cancel_order":
		var cancel WsCancelOrderResponse
		err := json.Unmarshal(respRaw, &cancel)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &order.Detail{
			Exchange:    e.Name,
			OrderID:     strconv.FormatInt(cancel.OrderID, 10),
			Status:      order.Cancelled,
			LastUpdated: time.Now(),
			AssetType:   asset.Spot,
		})
	case "cancel_orders":
		var cancels WsCancelOrdersResponse
		err := json.Unmarshal(respRaw, &cancels)
		if err != nil {
			return err
		}
		for i := range cancels.Results {
			if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
				Exchange:    e.Name,
				OrderID:     strconv.FormatInt(cancels.Results[i].OrderID, 10),
				Status:      order.Cancelled,
				LastUpdated: time.Now(),
				AssetType:   asset.Spot,
			}); err != nil {
				return err
			}
		}
	case "trade_history":
		var trades WsTradeHistoryResponse
		err := json.Unmarshal(respRaw, &trades)
		if err != nil {
			return err
		}
	case "inst_list":
		var instList wsInstList
		err := json.Unmarshal(respRaw, &instList)
		if err != nil {
			return err
		}
		for k, v := range instList.Spot {
			for _, v2 := range v {
				e.instrumentMap.Seed(k, v2.InstrumentID)
			}
		}
	case "inst_tick":
		var wsTicker WsTicker
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		pairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}
		currencyPair := e.instrumentMap.LookupInstrument(wsTicker.InstID)
		p, err := currency.NewPairFromFormattedPairs(currencyPair,
			pairs,
			format)
		if err != nil {
			return err
		}

		return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			Volume:       wsTicker.Volume24,
			QuoteVolume:  wsTicker.Volume24Quote,
			Bid:          wsTicker.HighestBuy,
			Ask:          wsTicker.LowestSell,
			High:         wsTicker.High24,
			Low:          wsTicker.Low24,
			Last:         wsTicker.Last,
			LastUpdated:  wsTicker.Timestamp.Time(),
			AssetType:    asset.Spot,
			Pair:         p,
		})
	case "inst_order_book":
		var orderbookSnapshot WsOrderbookSnapshot
		err := json.Unmarshal(respRaw, &orderbookSnapshot)
		if err != nil {
			return err
		}
		return e.WsProcessOrderbookSnapshot(&orderbookSnapshot)
	case "inst_order_book_update":
		var orderbookUpdate WsOrderbookUpdate
		err := json.Unmarshal(respRaw, &orderbookUpdate)
		if err != nil {
			return err
		}
		return e.WsProcessOrderbookUpdate(&orderbookUpdate)
	case "inst_trade":
		if !e.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeSnap WsTradeSnapshot
		err := json.Unmarshal(respRaw, &tradeSnap)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range tradeSnap.Trades {
			pairs, err := e.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}
			currencyPair := e.instrumentMap.LookupInstrument(tradeSnap.InstrumentID)
			p, err := currency.NewPairFromFormattedPairs(currencyPair,
				pairs,
				format)
			if err != nil {
				return err
			}

			tSide, err := order.StringToOrderSide(tradeSnap.Trades[i].Side)
			if err != nil {
				return err
			}

			trades = append(trades, trade.Data{
				Timestamp:    tradeSnap.Trades[i].Timestamp.Time(),
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Exchange:     e.Name,
				Price:        tradeSnap.Trades[i].Price,
				Side:         tSide,
				Amount:       tradeSnap.Trades[i].Quantity,
				TID:          strconv.FormatInt(tradeSnap.Trades[i].TransID, 10),
			})
		}
		return trade.AddTradesToBuffer(trades...)
	case "inst_trade_update":
		if !e.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeUpdate WsTradeUpdate
		err := json.Unmarshal(respRaw, &tradeUpdate)
		if err != nil {
			return err
		}

		pairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}
		currencyPair := e.instrumentMap.LookupInstrument(tradeUpdate.InstID)
		p, err := currency.NewPairFromFormattedPairs(currencyPair,
			pairs,
			format)
		if err != nil {
			return err
		}

		tSide, err := order.StringToOrderSide(tradeUpdate.Side)
		if err != nil {
			return err
		}

		return trade.AddTradesToBuffer(trade.Data{
			Timestamp:    tradeUpdate.Timestamp.Time(),
			CurrencyPair: p,
			AssetType:    asset.Spot,
			Exchange:     e.Name,
			Price:        tradeUpdate.Price,
			Side:         tSide,
			Amount:       tradeUpdate.Quantity,
			TID:          strconv.FormatInt(tradeUpdate.TransID, 10),
		})
	case "order_filled", "order_rejected", "order_accepted":
		var orderContainer wsOrderContainer
		err := json.Unmarshal(respRaw, &orderContainer)
		if err != nil {
			return err
		}
		o, err := e.parseOrderContainer(&orderContainer)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, o)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}
	return nil
}

func stringToOrderStatus(status string, quantity float64) (order.Status, error) {
	switch status {
	case "order_accepted":
		return order.Active, nil
	case "order_filled":
		if quantity > 0 {
			return order.PartiallyFilled, nil
		}
		return order.Filled, nil
	case "order_rejected":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func (e *Exchange) parseOrderContainer(oContainer *wsOrderContainer) (*order.Detail, error) {
	var oSide order.Side
	var oStatus order.Status
	var err error
	orderID := strconv.FormatInt(oContainer.OrderID, 10)
	if oContainer.Side != "" {
		oSide, err = order.StringToOrderSide(oContainer.Side)
		if err != nil {
			return nil, err
		}
	} else if oContainer.Order.Side != "" {
		oSide, err = order.StringToOrderSide(oContainer.Order.Side)
		if err != nil {
			return nil, err
		}
	}

	oStatus, err = stringToOrderStatus(oContainer.Reply, oContainer.OpenQuantity)
	if err != nil {
		return nil, err
	}
	if oContainer.Status[0] != "OK" {
		return nil, fmt.Errorf("%s - Order rejected: %v", e.Name, oContainer.Status)
	}
	if len(oContainer.Reasons) > 0 {
		return nil, fmt.Errorf("%s - Order rejected: %v", e.Name, oContainer.Reasons)
	}

	o := &order.Detail{
		Price:           oContainer.Price,
		Amount:          oContainer.Quantity,
		ExecutedAmount:  oContainer.FillQuantity,
		RemainingAmount: oContainer.OpenQuantity,
		Exchange:        e.Name,
		OrderID:         orderID,
		Side:            oSide,
		Status:          oStatus,
		Date:            oContainer.Timestamp.Time(),
		Trades:          nil,
	}
	if oContainer.Reply == "order_filled" {
		o.Side, err = order.StringToOrderSide(oContainer.Order.Side)
		if err != nil {
			return nil, err
		}
		o.RemainingAmount = oContainer.Order.OpenQuantity
		o.Amount = oContainer.Order.Quantity
		o.OrderID = strconv.FormatInt(oContainer.Order.OrderID, 10)
		o.LastUpdated = oContainer.Timestamp.Time()
		o.Pair, o.AssetType, err = e.GetRequestFormattedPairAndAssetType(e.instrumentMap.LookupInstrument(oContainer.Order.InstrumentID))
		if err != nil {
			return nil, err
		}
		o.Trades = []order.TradeHistory{
			{
				Price:     oContainer.FillPrice,
				Amount:    oContainer.FillQuantity,
				Exchange:  e.Name,
				TID:       strconv.FormatInt(oContainer.TransactionID, 10),
				Side:      oSide,
				Timestamp: oContainer.Timestamp.Time(),
			},
		}
	} else {
		o.Pair, o.AssetType, err = e.GetRequestFormattedPairAndAssetType(e.instrumentMap.LookupInstrument(oContainer.InstrumentID))
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}

// WsGetInstruments fetches instrument list and propagates a local cache
func (e *Exchange) WsGetInstruments(ctx context.Context) (Instruments, error) {
	var list Instruments
	req := wsRequest{
		Request:      "inst_list",
		SecurityType: strings.ToUpper(asset.Spot.String()),
		Nonce:        getNonce(),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.Nonce, req)
	if err != nil {
		return list, err
	}
	err = json.Unmarshal(resp, &list)
	if err != nil {
		return list, err
	}
	for curr, data := range list.Instruments {
		e.instrumentMap.Seed(curr, data[0].InstrumentID)
	}
	if len(e.instrumentMap.GetInstrumentIDs()) == 0 {
		return list, errors.New("instrument list failed to populate")
	}
	return list, nil
}

// WsProcessOrderbookSnapshot processes the orderbook snapshot
func (e *Exchange) WsProcessOrderbookSnapshot(ob *WsOrderbookSnapshot) error {
	bids := make([]orderbook.Level, len(ob.Buy))
	for i := range ob.Buy {
		bids[i] = orderbook.Level{
			Amount: ob.Buy[i].Volume,
			Price:  ob.Buy[i].Price,
		}
	}

	asks := make([]orderbook.Level, len(ob.Sell))
	for i := range ob.Sell {
		asks[i] = orderbook.Level{
			Amount: ob.Sell[i].Volume,
			Price:  ob.Sell[i].Price,
		}
	}

	var newOrderBook orderbook.Book
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.ValidateOrderbook = e.ValidateOrderbook

	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	newOrderBook.Pair, err = currency.NewPairFromFormattedPairs(
		e.instrumentMap.LookupInstrument(ob.InstID),
		pairs,
		format)
	if err != nil {
		return err
	}

	newOrderBook.Asset = asset.Spot
	newOrderBook.Exchange = e.Name
	newOrderBook.LastUpdated = time.Now() // No time sent

	return e.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessOrderbookUpdate process an orderbook update
func (e *Exchange) WsProcessOrderbookUpdate(update *WsOrderbookUpdate) error {
	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(
		e.instrumentMap.LookupInstrument(update.InstID),
		pairs,
		format)
	if err != nil {
		return err
	}

	bufferUpdate := &orderbook.Update{
		Pair:       p,
		UpdateID:   update.TransID,
		Asset:      asset.Spot,
		UpdateTime: time.Now(), // No time sent
	}
	if strings.EqualFold(update.Side, order.Buy.Lower()) {
		bufferUpdate.Bids = []orderbook.Level{{Price: update.Price, Amount: update.Volume}}
	} else {
		bufferUpdate.Asks = []orderbook.Level{{Price: update.Price, Amount: update.Volume}}
	}
	return e.Websocket.Orderbook.Update(bufferUpdate)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (e *Exchange) GenerateDefaultSubscriptions() (subscription.List, error) {
	channels := []string{"inst_tick", "inst_order_book", "inst_trade"}
	var subscriptions subscription.List
	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		for j := range enabledPairs {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[i],
				Pairs:   currency.Pairs{enabledPairs[j]},
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	var errs error
	for _, s := range subs {
		if len(s.Pairs) != 1 {
			return subscription.ErrNotSinglePair
		}
		fPair, err := e.FormatExchangeCurrency(s.Pairs[0], asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}

		subscribe := wsRequest{
			Request:      s.Channel,
			InstrumentID: e.instrumentMap.LookupID(fPair.String()),
			Subscribe:    true,
			Nonce:        getNonce(),
		}
		err = e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, subscribe)
		if err == nil {
			err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(channelToUnsubscribe subscription.List) error {
	ctx := context.TODO()
	var errs error
	for _, s := range channelToUnsubscribe {
		if len(s.Pairs) != 1 {
			return subscription.ErrNotSinglePair
		}
		fPair, err := e.FormatExchangeCurrency(s.Pairs[0], asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}

		subscribe := wsRequest{
			Request:      s.Channel,
			InstrumentID: e.instrumentMap.LookupID(fPair.String()),
			Subscribe:    false,
			Nonce:        getNonce(),
		}
		resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, subscribe.Nonce, subscribe)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		var response map[string]any
		err = json.Unmarshal(resp, &response)
		if err == nil {
			val, ok := response["status"].([]any)
			switch {
			case !ok:
				err = common.GetTypeAssertError("[]any", response["status"])
			case len(val) == 0, val[0] != "OK":
				err = common.AppendError(errs, fmt.Errorf("%v unsubscribe failed for channel %v", e.Name, s.Channel))
			default:
				err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

func (e *Exchange) wsAuthenticate(ctx context.Context) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	r := WsLoginReq{
		Request:   "login",
		Username:  creds.ClientID,
		Nonce:     getNonce(),
		Timestamp: time.Now().Unix(),
	}
	payload := creds.ClientID + "|" + strconv.FormatInt(r.Timestamp, 10) + "|" + strconv.FormatInt(r.Nonce, 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Key))
	if err != nil {
		return err
	}
	r.Hmac = hex.EncodeToString(hmac)

	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, r.Nonce, r)
	if err != nil {
		return err
	}

	respKey, err := jsonparser.GetUnsafeString(resp, "api_key")
	if err != nil || respKey != creds.Key {
		return errors.New("failed to authenticate")
	}

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)

	return nil
}

func (e *Exchange) wsGetAccountBalance(ctx context.Context) (*UserBalance, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", e.Name)
	}
	accBalance := wsRequest{
		Request: "user_balance",
		Nonce:   getNonce(),
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, accBalance.Nonce, accBalance)
	if err != nil {
		return nil, err
	}
	var response UserBalance
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Status[0] != "OK" {
		return &response, fmt.Errorf("%v get account balance failed", e.Name)
	}
	return &response, nil
}

func (e *Exchange) wsSubmitOrder(ctx context.Context, o *WsSubmitOrderParameters) (*order.Detail, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", e.Name)
	}

	curr, err := e.FormatExchangeCurrency(o.Currency, asset.Spot)
	if err != nil {
		return nil, err
	}

	var orderSubmissionRequest WsSubmitOrderRequest
	orderSubmissionRequest.Request = "new_order"
	orderSubmissionRequest.Nonce = getNonce()
	orderSubmissionRequest.InstrumentID = e.instrumentMap.LookupID(curr.String())
	orderSubmissionRequest.Quantity = o.Amount
	orderSubmissionRequest.Price = o.Price
	orderSubmissionRequest.Side = o.Side.String()

	if o.OrderID > 0 {
		orderSubmissionRequest.OrderID = o.OrderID
	}
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, orderSubmissionRequest.Nonce, orderSubmissionRequest)
	if err != nil {
		return nil, err
	}
	var incoming wsOrderContainer
	err = json.Unmarshal(resp, &incoming)
	if err != nil {
		return nil, err
	}
	var ord *order.Detail
	ord, err = e.parseOrderContainer(&incoming)
	if err != nil {
		return nil, err
	}
	return ord, nil
}

func (e *Exchange) wsSubmitOrders(ctx context.Context, orders []WsSubmitOrderParameters) ([]order.Detail, []error) {
	var errs []error
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		errs = append(errs, fmt.Errorf("%v not authorised to submit orders",
			e.Name))
		return nil, errs
	}
	orderRequest := WsSubmitOrdersRequest{}
	for i := range orders {
		curr, err := e.FormatExchangeCurrency(orders[i].Currency, asset.Spot)
		if err != nil {
			return nil, []error{err}
		}

		orderRequest.Orders = append(orderRequest.Orders,
			WsSubmitOrdersRequestData{
				Quantity:      orders[i].Amount,
				Price:         orders[i].Price,
				Side:          orders[i].Side.String(),
				InstrumentID:  e.instrumentMap.LookupID(curr.String()),
				ClientOrderID: i + 1,
			})
	}

	orderRequest.Nonce = getNonce()
	orderRequest.Request = "new_orders"
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, orderRequest.Nonce, orderRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	var incoming []wsOrderContainer
	err = json.Unmarshal(resp, &incoming)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	ordersResponse := make([]order.Detail, 0, len(incoming))
	for i := range incoming {
		o, err := e.parseOrderContainer(&incoming[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ordersResponse = append(ordersResponse, *o)
	}

	return ordersResponse, errs
}

func (e *Exchange) wsGetOpenOrders(ctx context.Context, curr string) (*WsUserOpenOrdersResponse, error) {
	var response *WsUserOpenOrdersResponse
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get open orders",
			e.Name)
	}
	var openOrdersRequest WsGetOpenOrdersRequest
	openOrdersRequest.Request = "user_open_orders"
	openOrdersRequest.Nonce = getNonce()
	openOrdersRequest.InstrumentID = e.instrumentMap.LookupID(curr)

	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, openOrdersRequest.Nonce, openOrdersRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get open orders failed for currency %v",
			e.Name,
			curr)
	}
	return response, nil
}

func (e *Exchange) wsCancelOrder(ctx context.Context, cancellation *WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var response *CancelOrdersResponse
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to cancel order", e.Name)
	}

	curr, err := e.FormatExchangeCurrency(cancellation.Currency, asset.Spot)
	if err != nil {
		return nil, err
	}

	var cancellationRequest WsCancelOrderRequest
	cancellationRequest.Request = "cancel_order"
	cancellationRequest.InstrumentID = e.instrumentMap.LookupID(curr.String())
	cancellationRequest.OrderID = cancellation.OrderID
	cancellationRequest.Nonce = getNonce()

	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, cancellationRequest.Nonce, cancellationRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v order cancellation failed for currency %v and orderID %v, message %v",
			e.Name,
			cancellation.Currency,
			cancellation.OrderID,
			response.Status[0])
	}
	return response, nil
}

func (e *Exchange) wsCancelOrders(ctx context.Context, cancellations []WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var err error
	var response *CancelOrdersResponse
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, err
	}
	var cancelOrderRequest WsCancelOrdersRequest
	for i := range cancellations {
		var curr currency.Pair
		curr, err = e.FormatExchangeCurrency(cancellations[i].Currency,
			asset.Spot)
		if err != nil {
			return nil, err
		}
		cancelOrderRequest.Entries = append(cancelOrderRequest.Entries,
			WsCancelOrdersRequestEntry{
				InstID:  e.instrumentMap.LookupID(curr.String()),
				OrderID: cancellations[i].OrderID,
			})
	}

	cancelOrderRequest.Request = "cancel_orders"
	cancelOrderRequest.Nonce = getNonce()
	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, cancelOrderRequest.Nonce, cancelOrderRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	return response, err
}

func (e *Exchange) wsGetTradeHistory(ctx context.Context, p currency.Pair, start, limit int64) (*WsTradeHistoryResponse, error) {
	var response *WsTradeHistoryResponse
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get trade history",
			e.Name)
	}

	curr, err := e.FormatExchangeCurrency(p, asset.Spot)
	if err != nil {
		return nil, err
	}

	var req WsTradeHistoryRequest
	req.Request = "trade_history"
	req.InstID = e.instrumentMap.LookupID(curr.String())
	req.Nonce = getNonce()
	req.Start = start
	req.Limit = limit

	resp, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.Nonce, req)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get trade history failed for %v", e.Name, req)
	}
	return response, nil
}
