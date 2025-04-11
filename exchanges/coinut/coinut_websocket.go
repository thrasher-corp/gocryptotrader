package coinut

import (
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/internal/exchange/websocket"
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
func (c *COINUT) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	c.Websocket.Wg.Add(1)
	go c.wsReadData()

	if !c.instrumentMap.IsLoaded() {
		_, err = c.WsGetInstruments()
		if err != nil {
			return err
		}
	}

	if c.IsWebsocketAuthenticationSupported() {
		if err = c.wsAuthenticate(context.TODO()); err != nil {
			c.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorln(log.WebsocketMgr, c.Name+" "+err.Error())
		}
	}

	// define bi-directional communication
	channels = make(map[string]chan []byte)
	channels["hb"] = make(chan []byte, 1)

	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *COINUT) wsReadData() {
	defer c.Websocket.Wg.Done()

	ctx := context.TODO()

	for {
		resp := c.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		if strings.HasPrefix(string(resp.Raw), "[") {
			var incoming []wsResponse
			err := json.Unmarshal(resp.Raw, &incoming)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}
			for i := range incoming {
				if incoming[i].Nonce > 0 {
					if c.Websocket.Match.IncomingWithData(incoming[i].Nonce, resp.Raw) {
						break
					}
				}
				var individualJSON []byte
				individualJSON, err = json.Marshal(incoming[i])
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				err = c.wsHandleData(ctx, individualJSON)
				if err != nil {
					c.Websocket.DataHandler <- err
				}
			}
		} else {
			var incoming wsResponse
			err := json.Unmarshal(resp.Raw, &incoming)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}
			err = c.wsHandleData(ctx, resp.Raw)
			if err != nil {
				c.Websocket.DataHandler <- err
			}
		}
	}
}

func (c *COINUT) wsHandleData(_ context.Context, respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "[") {
		var orders []wsOrderContainer
		err := json.Unmarshal(respRaw, &orders)
		if err != nil {
			return err
		}
		for i := range orders {
			o, err2 := c.parseOrderContainer(&orders[i])
			if err2 != nil {
				return err2
			}
			c.Websocket.DataHandler <- o
		}
		return nil
	}

	var incoming wsResponse
	err := json.Unmarshal(respRaw, &incoming)
	if err != nil {
		return err
	}
	if c.Websocket.Match.IncomingWithData(incoming.Nonce, respRaw) {
		return nil
	}

	format, err := c.GetPairFormat(asset.Spot, true)
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
		c.Websocket.DataHandler <- &order.Detail{
			Exchange:    c.Name,
			OrderID:     strconv.FormatInt(cancel.OrderID, 10),
			Status:      order.Cancelled,
			LastUpdated: time.Now(),
			AssetType:   asset.Spot,
		}
	case "cancel_orders":
		var cancels WsCancelOrdersResponse
		err := json.Unmarshal(respRaw, &cancels)
		if err != nil {
			return err
		}
		for i := range cancels.Results {
			c.Websocket.DataHandler <- &order.Detail{
				Exchange:    c.Name,
				OrderID:     strconv.FormatInt(cancels.Results[i].OrderID, 10),
				Status:      order.Cancelled,
				LastUpdated: time.Now(),
				AssetType:   asset.Spot,
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
				c.instrumentMap.Seed(k, v2.InstrumentID)
			}
		}
	case "inst_tick":
		var wsTicker WsTicker
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		pairs, err := c.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}
		currencyPair := c.instrumentMap.LookupInstrument(wsTicker.InstID)
		p, err := currency.NewPairFromFormattedPairs(currencyPair,
			pairs,
			format)
		if err != nil {
			return err
		}

		c.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: c.Name,
			Volume:       wsTicker.Volume24,
			QuoteVolume:  wsTicker.Volume24Quote,
			Bid:          wsTicker.HighestBuy,
			Ask:          wsTicker.LowestSell,
			High:         wsTicker.High24,
			Low:          wsTicker.Low24,
			Last:         wsTicker.Last,
			LastUpdated:  time.Unix(0, wsTicker.Timestamp),
			AssetType:    asset.Spot,
			Pair:         p,
		}
	case "inst_order_book":
		var orderbookSnapshot WsOrderbookSnapshot
		err := json.Unmarshal(respRaw, &orderbookSnapshot)
		if err != nil {
			return err
		}
		err = c.WsProcessOrderbookSnapshot(&orderbookSnapshot)
		if err != nil {
			return err
		}
	case "inst_order_book_update":
		var orderbookUpdate WsOrderbookUpdate
		err := json.Unmarshal(respRaw, &orderbookUpdate)
		if err != nil {
			return err
		}
		err = c.WsProcessOrderbookUpdate(&orderbookUpdate)
		if err != nil {
			return err
		}
	case "inst_trade":
		if !c.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeSnap WsTradeSnapshot
		err := json.Unmarshal(respRaw, &tradeSnap)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range tradeSnap.Trades {
			pairs, err := c.GetEnabledPairs(asset.Spot)
			if err != nil {
				return err
			}
			currencyPair := c.instrumentMap.LookupInstrument(tradeSnap.InstrumentID)
			p, err := currency.NewPairFromFormattedPairs(currencyPair,
				pairs,
				format)
			if err != nil {
				return err
			}

			tSide, err := order.StringToOrderSide(tradeSnap.Trades[i].Side)
			if err != nil {
				c.Websocket.DataHandler <- order.ClassificationError{
					Exchange: c.Name,
					Err:      err,
				}
			}

			trades = append(trades, trade.Data{
				Timestamp:    time.Unix(0, tradeSnap.Trades[i].Timestamp*1000),
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Exchange:     c.Name,
				Price:        tradeSnap.Trades[i].Price,
				Side:         tSide,
				Amount:       tradeSnap.Trades[i].Quantity,
				TID:          strconv.FormatInt(tradeSnap.Trades[i].TransID, 10),
			})
		}
		return trade.AddTradesToBuffer(trades...)
	case "inst_trade_update":
		if !c.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeUpdate WsTradeUpdate
		err := json.Unmarshal(respRaw, &tradeUpdate)
		if err != nil {
			return err
		}

		pairs, err := c.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}
		currencyPair := c.instrumentMap.LookupInstrument(tradeUpdate.InstID)
		p, err := currency.NewPairFromFormattedPairs(currencyPair,
			pairs,
			format)
		if err != nil {
			return err
		}

		tSide, err := order.StringToOrderSide(tradeUpdate.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				Err:      err,
			}
		}

		return trade.AddTradesToBuffer(trade.Data{
			Timestamp:    time.Unix(0, tradeUpdate.Timestamp*1000),
			CurrencyPair: p,
			AssetType:    asset.Spot,
			Exchange:     c.Name,
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
		o, err := c.parseOrderContainer(&orderContainer)
		if err != nil {
			return err
		}
		c.Websocket.DataHandler <- o
	default:
		c.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: c.Name + websocket.UnhandledMessage + string(respRaw)}
		return nil
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

func (c *COINUT) parseOrderContainer(oContainer *wsOrderContainer) (*order.Detail, error) {
	var oSide order.Side
	var oStatus order.Status
	var err error
	orderID := strconv.FormatInt(oContainer.OrderID, 10)
	if oContainer.Side != "" {
		oSide, err = order.StringToOrderSide(oContainer.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
	} else if oContainer.Order.Side != "" {
		oSide, err = order.StringToOrderSide(oContainer.Order.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
	}

	oStatus, err = stringToOrderStatus(oContainer.Reply, oContainer.OpenQuantity)
	if err != nil {
		c.Websocket.DataHandler <- order.ClassificationError{
			Exchange: c.Name,
			OrderID:  orderID,
			Err:      err,
		}
	}
	if oContainer.Status[0] != "OK" {
		return nil, fmt.Errorf("%s - Order rejected: %v", c.Name, oContainer.Status)
	}
	if len(oContainer.Reasons) > 0 {
		return nil, fmt.Errorf("%s - Order rejected: %v", c.Name, oContainer.Reasons)
	}

	o := &order.Detail{
		Price:           oContainer.Price,
		Amount:          oContainer.Quantity,
		ExecutedAmount:  oContainer.FillQuantity,
		RemainingAmount: oContainer.OpenQuantity,
		Exchange:        c.Name,
		OrderID:         orderID,
		Side:            oSide,
		Status:          oStatus,
		Date:            time.Unix(0, oContainer.Timestamp),
		Trades:          nil,
	}
	if oContainer.Reply == "order_filled" {
		o.Side, err = order.StringToOrderSide(oContainer.Order.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		o.RemainingAmount = oContainer.Order.OpenQuantity
		o.Amount = oContainer.Order.Quantity
		o.OrderID = strconv.FormatInt(oContainer.Order.OrderID, 10)
		o.LastUpdated = time.Unix(0, oContainer.Timestamp)
		o.Pair, o.AssetType, err = c.GetRequestFormattedPairAndAssetType(c.instrumentMap.LookupInstrument(oContainer.Order.InstrumentID))
		if err != nil {
			return nil, err
		}
		o.Trades = []order.TradeHistory{
			{
				Price:     oContainer.FillPrice,
				Amount:    oContainer.FillQuantity,
				Exchange:  c.Name,
				TID:       strconv.FormatInt(oContainer.TransactionID, 10),
				Side:      oSide,
				Timestamp: time.Unix(0, oContainer.Timestamp),
			},
		}
	} else {
		o.Pair, o.AssetType, err = c.GetRequestFormattedPairAndAssetType(c.instrumentMap.LookupInstrument(oContainer.InstrumentID))
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}

// WsGetInstruments fetches instrument list and propagates a local cache
func (c *COINUT) WsGetInstruments() (Instruments, error) {
	var list Instruments
	req := wsRequest{
		Request:      "inst_list",
		SecurityType: strings.ToUpper(asset.Spot.String()),
		Nonce:        getNonce(),
	}
	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, req.Nonce, req)
	if err != nil {
		return list, err
	}
	err = json.Unmarshal(resp, &list)
	if err != nil {
		return list, err
	}
	for curr, data := range list.Instruments {
		c.instrumentMap.Seed(curr, data[0].InstrumentID)
	}
	if len(c.instrumentMap.GetInstrumentIDs()) == 0 {
		return list, errors.New("instrument list failed to populate")
	}
	return list, nil
}

// WsProcessOrderbookSnapshot processes the orderbook snapshot
func (c *COINUT) WsProcessOrderbookSnapshot(ob *WsOrderbookSnapshot) error {
	bids := make([]orderbook.Tranche, len(ob.Buy))
	for i := range ob.Buy {
		bids[i] = orderbook.Tranche{
			Amount: ob.Buy[i].Volume,
			Price:  ob.Buy[i].Price,
		}
	}

	asks := make([]orderbook.Tranche, len(ob.Sell))
	for i := range ob.Sell {
		asks[i] = orderbook.Tranche{
			Amount: ob.Sell[i].Volume,
			Price:  ob.Sell[i].Price,
		}
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.VerifyOrderbook = c.CanVerifyOrderbook

	pairs, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := c.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	newOrderBook.Pair, err = currency.NewPairFromFormattedPairs(
		c.instrumentMap.LookupInstrument(ob.InstID),
		pairs,
		format)
	if err != nil {
		return err
	}

	newOrderBook.Asset = asset.Spot
	newOrderBook.Exchange = c.Name
	newOrderBook.LastUpdated = time.Now() // No time sent

	return c.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessOrderbookUpdate process an orderbook update
func (c *COINUT) WsProcessOrderbookUpdate(update *WsOrderbookUpdate) error {
	pairs, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	format, err := c.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}

	p, err := currency.NewPairFromFormattedPairs(
		c.instrumentMap.LookupInstrument(update.InstID),
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
		bufferUpdate.Bids = []orderbook.Tranche{{Price: update.Price, Amount: update.Volume}}
	} else {
		bufferUpdate.Asks = []orderbook.Tranche{{Price: update.Price, Amount: update.Volume}}
	}
	return c.Websocket.Orderbook.Update(bufferUpdate)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *COINUT) GenerateDefaultSubscriptions() (subscription.List, error) {
	channels := []string{"inst_tick", "inst_order_book", "inst_trade"}
	var subscriptions subscription.List
	enabledPairs, err := c.GetEnabledPairs(asset.Spot)
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
func (c *COINUT) Subscribe(subs subscription.List) error {
	var errs error
	for _, s := range subs {
		if len(s.Pairs) != 1 {
			return subscription.ErrNotSinglePair
		}
		fPair, err := c.FormatExchangeCurrency(s.Pairs[0], asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}

		subscribe := wsRequest{
			Request:      s.Channel,
			InstrumentID: c.instrumentMap.LookupID(fPair.String()),
			Subscribe:    true,
			Nonce:        getNonce(),
		}
		err = c.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, subscribe)
		if err == nil {
			err = c.Websocket.AddSuccessfulSubscriptions(c.Websocket.Conn, s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *COINUT) Unsubscribe(channelToUnsubscribe subscription.List) error {
	var errs error
	for _, s := range channelToUnsubscribe {
		if len(s.Pairs) != 1 {
			return subscription.ErrNotSinglePair
		}
		fPair, err := c.FormatExchangeCurrency(s.Pairs[0], asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}

		subscribe := wsRequest{
			Request:      s.Channel,
			InstrumentID: c.instrumentMap.LookupID(fPair.String()),
			Subscribe:    false,
			Nonce:        getNonce(),
		}
		resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, subscribe.Nonce, subscribe)
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
				err = common.AppendError(errs, fmt.Errorf("%v unsubscribe failed for channel %v", c.Name, s.Channel))
			default:
				err = c.Websocket.RemoveSubscriptions(c.Websocket.Conn, s)
			}
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

func (c *COINUT) wsAuthenticate(ctx context.Context) error {
	creds, err := c.GetCredentials(ctx)
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
	r.Hmac = crypto.HexEncodeToString(hmac)

	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, r.Nonce, r)
	if err != nil {
		return err
	}

	respKey, err := jsonparser.GetUnsafeString(resp, "api_key")
	if err != nil || respKey != creds.Key {
		return errors.New("failed to authenticate")
	}

	c.Websocket.SetCanUseAuthenticatedEndpoints(true)

	return nil
}

func (c *COINUT) wsGetAccountBalance() (*UserBalance, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}
	accBalance := wsRequest{
		Request: "user_balance",
		Nonce:   getNonce(),
	}
	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, accBalance.Nonce, accBalance)
	if err != nil {
		return nil, err
	}
	var response UserBalance
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Status[0] != "OK" {
		return &response, fmt.Errorf("%v get account balance failed", c.Name)
	}
	return &response, nil
}

func (c *COINUT) wsSubmitOrder(o *WsSubmitOrderParameters) (*order.Detail, error) {
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to submit order", c.Name)
	}

	curr, err := c.FormatExchangeCurrency(o.Currency, asset.Spot)
	if err != nil {
		return nil, err
	}

	var orderSubmissionRequest WsSubmitOrderRequest
	orderSubmissionRequest.Request = "new_order"
	orderSubmissionRequest.Nonce = getNonce()
	orderSubmissionRequest.InstrumentID = c.instrumentMap.LookupID(curr.String())
	orderSubmissionRequest.Quantity = o.Amount
	orderSubmissionRequest.Price = o.Price
	orderSubmissionRequest.Side = o.Side.String()

	if o.OrderID > 0 {
		orderSubmissionRequest.OrderID = o.OrderID
	}
	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, orderSubmissionRequest.Nonce, orderSubmissionRequest)
	if err != nil {
		return nil, err
	}
	var incoming wsOrderContainer
	err = json.Unmarshal(resp, &incoming)
	if err != nil {
		return nil, err
	}
	var ord *order.Detail
	ord, err = c.parseOrderContainer(&incoming)
	if err != nil {
		return nil, err
	}
	return ord, nil
}

func (c *COINUT) wsSubmitOrders(orders []WsSubmitOrderParameters) ([]order.Detail, []error) {
	var errs []error
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		errs = append(errs, fmt.Errorf("%v not authorised to submit orders",
			c.Name))
		return nil, errs
	}
	orderRequest := WsSubmitOrdersRequest{}
	for i := range orders {
		curr, err := c.FormatExchangeCurrency(orders[i].Currency, asset.Spot)
		if err != nil {
			return nil, []error{err}
		}

		orderRequest.Orders = append(orderRequest.Orders,
			WsSubmitOrdersRequestData{
				Quantity:      orders[i].Amount,
				Price:         orders[i].Price,
				Side:          orders[i].Side.String(),
				InstrumentID:  c.instrumentMap.LookupID(curr.String()),
				ClientOrderID: i + 1,
			})
	}

	orderRequest.Nonce = getNonce()
	orderRequest.Request = "new_orders"
	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, orderRequest.Nonce, orderRequest)
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
		o, err := c.parseOrderContainer(&incoming[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ordersResponse = append(ordersResponse, *o)
	}

	return ordersResponse, errs
}

func (c *COINUT) wsGetOpenOrders(curr string) (*WsUserOpenOrdersResponse, error) {
	var response *WsUserOpenOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get open orders",
			c.Name)
	}
	var openOrdersRequest WsGetOpenOrdersRequest
	openOrdersRequest.Request = "user_open_orders"
	openOrdersRequest.Nonce = getNonce()
	openOrdersRequest.InstrumentID = c.instrumentMap.LookupID(curr)

	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, openOrdersRequest.Nonce, openOrdersRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get open orders failed for currency %v",
			c.Name,
			curr)
	}
	return response, nil
}

func (c *COINUT) wsCancelOrder(cancellation *WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var response *CancelOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to cancel order", c.Name)
	}

	curr, err := c.FormatExchangeCurrency(cancellation.Currency, asset.Spot)
	if err != nil {
		return nil, err
	}

	var cancellationRequest WsCancelOrderRequest
	cancellationRequest.Request = "cancel_order"
	cancellationRequest.InstrumentID = c.instrumentMap.LookupID(curr.String())
	cancellationRequest.OrderID = cancellation.OrderID
	cancellationRequest.Nonce = getNonce()

	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, cancellationRequest.Nonce, cancellationRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v order cancellation failed for currency %v and orderID %v, message %v",
			c.Name,
			cancellation.Currency,
			cancellation.OrderID,
			response.Status[0])
	}
	return response, nil
}

func (c *COINUT) wsCancelOrders(cancellations []WsCancelOrderParameters) (*CancelOrdersResponse, error) {
	var err error
	var response *CancelOrdersResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, err
	}
	var cancelOrderRequest WsCancelOrdersRequest
	for i := range cancellations {
		var curr currency.Pair
		curr, err = c.FormatExchangeCurrency(cancellations[i].Currency,
			asset.Spot)
		if err != nil {
			return nil, err
		}
		cancelOrderRequest.Entries = append(cancelOrderRequest.Entries,
			WsCancelOrdersRequestEntry{
				InstID:  c.instrumentMap.LookupID(curr.String()),
				OrderID: cancellations[i].OrderID,
			})
	}

	cancelOrderRequest.Request = "cancel_orders"
	cancelOrderRequest.Nonce = getNonce()
	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, cancelOrderRequest.Nonce, cancelOrderRequest)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	return response, err
}

func (c *COINUT) wsGetTradeHistory(p currency.Pair, start, limit int64) (*WsTradeHistoryResponse, error) {
	var response *WsTradeHistoryResponse
	if !c.Websocket.CanUseAuthenticatedEndpoints() {
		return response, fmt.Errorf("%v not authorised to get trade history",
			c.Name)
	}

	curr, err := c.FormatExchangeCurrency(p, asset.Spot)
	if err != nil {
		return nil, err
	}

	var req WsTradeHistoryRequest
	req.Request = "trade_history"
	req.InstID = c.instrumentMap.LookupID(curr.String())
	req.Nonce = getNonce()
	req.Start = start
	req.Limit = limit

	resp, err := c.Websocket.Conn.SendMessageReturnResponse(context.TODO(), request.Unset, req.Nonce, req)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return response, err
	}
	if response.Status[0] != "OK" {
		return response, fmt.Errorf("%v get trade history failed for %v", c.Name, req)
	}
	return response, nil
}
