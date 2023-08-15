package okcoin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// pong message
	pongBytes = "pong"

	// Public endpoint subscriptions
	wsInstruments = "instruments"
	wsTickers     = "tickers"
	wsCandle3M, wsCandle1M, wsCandle1W, wsCandle1D, wsCandle2D, wsCandle3D, wsCandle5D,
	wsCandle12H, wsCandle6H, wsCandle4H, wsCandle2H, wsCandle1H, wsCandle30m, wsCandle15m,
	wsCandle5m, wsCandle3m, wsCandle1m, wsCandle3Mutc, wsCandle1Mutc, wsCandle1Wutc, wsCandle1Dutc,
	wsCandle2Dutc, wsCandle3Dutc, wsCandle5Dutc, wsCandle12Hutc, wsCandle6Hutc = "candle3M", "candle1M", "candle1W", "candle1D", "candle2D",
		"candle3D", "candle5D", "candle12H", "candle6H", "candle4H",
		"candle2H", "candle1H", "candle30m", "candle15m", "candle5m",
		"candle3m", "candle1m", "candle3Mutc", "candle1Mutc", "candle1Wutc",
		"candle1Dutc", "candle2Dutc", "candle3Dutc", "candle5Dutc", "candle12Hutc", "candle6Hutc"
	wsTrades                  = "trades"
	wsOrderbooks              = "books"
	wsOrderbooksL5            = "book5"
	wsOrderbookL1             = "bbo-tbt"
	wsOrderbookTickByTickL400 = "books-l2-tbt"
	wsOrderbookTickByTickL50  = "books50-l2-tbt"
	wsStatus                  = "status"
	// Private subscriptions
	wsAccount     = "account"
	wsOrder       = "orders"
	wsOrdersAlgo  = "orders-algo"
	wsAlgoAdvance = "algo-advance"
)

var defaultSubscriptions = []string{
	wsTickers,
	wsOrderbooks,
	wsStatus,
}

func isAuthenticatedChannel(channel string) bool {
	switch channel {
	case wsAccount, wsOrder, wsOrdersAlgo, wsAlgoAdvance:
		return true
	}
	return false
}

// WsConnect initiates a websocket connection
func (o *Okcoin) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	err := o.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if o.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			o.Websocket.GetWebsocketURL())
	}
	o.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Delay:       time.Second * 25,
		Message:     []byte("ping"),
		MessageType: websocket.TextMessage,
	})

	o.Websocket.Wg.Add(1)
	go o.WsReadData(o.Websocket.Conn)

	if o.IsWebsocketAuthenticationSupported() {
		err = o.WsLogin(context.TODO(), &dialer)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				o.Name,
				err)
		}
	}
	return nil
}

// WsLogin sends a login request to websocket to enable access to authenticated endpoints
func (o *Okcoin) WsLogin(ctx context.Context, dialer *websocket.Dialer) error {
	o.Websocket.SetCanUseAuthenticatedEndpoints(false)
	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return err
	}
	err = o.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	o.Websocket.Wg.Add(1)
	go o.WsReadData(o.Websocket.AuthConn)
	o.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		Delay:       time.Second * 25,
		Message:     []byte("ping"),
		MessageType: websocket.TextMessage,
	})
	systemTime := time.Now()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(systemTime.Unix(), 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64 := crypto.Base64Encode(hmac)
	authRequest := WebsocketEventRequest{
		Operation: "login",
		Arguments: []map[string]string{
			{
				"apiKey":     creds.Key,
				"passphrase": creds.ClientID,
				"timestamp":  strconv.FormatInt(systemTime.Unix(), 10),
				"sign":       base64,
			},
		},
	}
	_, err = o.Websocket.AuthConn.SendMessageReturnResponse("login", authRequest)
	if err != nil {
		return err
	}
	o.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

// WsReadData receives and passes on websocket messages for processing
func (o *Okcoin) WsReadData(conn stream.Connection) {
	defer o.Websocket.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := o.WsHandleData(resp.Raw)
		if err != nil {
			o.Websocket.DataHandler <- err
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (o *Okcoin) WsHandleData(respRaw []byte) error {
	if bytes.Equal(respRaw, []byte(pongBytes)) {
		return nil
	}
	var dataResponse WebsocketDataResponse
	err := json.Unmarshal(respRaw, &dataResponse)
	if err != nil {
		return err
	}
	if dataResponse.ID != "" {
		if !o.Websocket.Match.IncomingWithData(dataResponse.ID, respRaw) {
			return fmt.Errorf("couldn't match incoming message with id: %s and operation: %s", dataResponse.ID, dataResponse.Operation)
		}
		return nil
	}
	if len(dataResponse.Data) > 0 {
		switch dataResponse.Arguments.Channel {
		case wsInstruments:
			return o.wsProcessInstruments(respRaw)
		case wsTickers:
			return o.wsProcessTickers(respRaw)
		case wsCandle3M, wsCandle1M, wsCandle1W, wsCandle1D, wsCandle2D, wsCandle3D, wsCandle5D,
			wsCandle12H, wsCandle6H, wsCandle4H, wsCandle2H, wsCandle1H, wsCandle30m, wsCandle15m,
			wsCandle5m, wsCandle3m, wsCandle1m, wsCandle3Mutc, wsCandle1Mutc, wsCandle1Wutc, wsCandle1Dutc,
			wsCandle2Dutc, wsCandle3Dutc, wsCandle5Dutc, wsCandle12Hutc, wsCandle6Hutc:
			return o.wsProcessCandles(respRaw)
		case wsTrades:
			return o.wsProcessTrades(respRaw)
		case wsOrderbooks,
			wsOrderbooksL5,
			wsOrderbookL1,
			wsOrderbookTickByTickL400,
			wsOrderbookTickByTickL50:
			return o.wsProcessOrderbook(respRaw, dataResponse.Arguments.Channel)
		case wsStatus:
			var resp WebsocketStatus
			err = json.Unmarshal(respRaw, &resp)
			if err != nil {
				return err
			}
			for x := range resp.Data {
				systemStatus := fmt.Sprintf("%s %s on system %s %s service type From %s To %s", systemStateString(resp.Data[x].State), resp.Data[x].Title, resp.Data[x].System, systemStatusServiceTypeString(resp.Data[x].ServiceType), resp.Data[x].Begin.Time().String(), resp.Data[x].End.Time().String())
				if resp.Data[x].Href != "" {
					systemStatus = fmt.Sprintf("%s Href: %s\n", systemStatus, resp.Data[x].Href)
				}
				if resp.Data[x].RescheduleDescription != "" {
					systemStatus = fmt.Sprintf("%s Rescheduled Description: %s", systemStatus, resp.Data[x].RescheduleDescription)
				}
				log.Warnf(log.ExchangeSys, systemStatus)
			}
			o.Websocket.DataHandler <- resp
			return nil
		case wsAccount:
			return o.wsProcessAccount(respRaw)
		case wsOrder:
			return o.wsProcessOrders(respRaw)
		case wsOrdersAlgo:
			return o.wsProcessAlgoOrder(respRaw)
		case wsAlgoAdvance:
			return o.wsProcessAdvancedAlgoOrder(respRaw)
		}
		o.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: o.Name + stream.UnhandledMessage + string(respRaw),
		}
		return nil
	}

	var errorResponse WebsocketErrorResponse
	err = json.Unmarshal(respRaw, &errorResponse)
	if err == nil && errorResponse.ErrorCode > 0 {
		return fmt.Errorf("%v error - %v message: %s ",
			o.Name,
			errorResponse.ErrorCode,
			errorResponse.Message)
	}
	var eventResponse WebsocketEventResponse
	err = json.Unmarshal(respRaw, &eventResponse)
	if err == nil && eventResponse.Event != "" {
		switch eventResponse.Event {
		case "login":
			if o.Websocket.Match.IncomingWithData("login", respRaw) {
				o.Websocket.SetCanUseAuthenticatedEndpoints(eventResponse.Code == "0")
			}
		case "subscribe", "unsubscribe":
			o.Websocket.DataHandler <- eventResponse
		case "error":
			if o.Verbose {
				log.Debugf(log.ExchangeSys,
					o.Name+" - "+eventResponse.Event+" on channel: "+eventResponse.Channel)
			}
		}
	}
	return nil
}

func (o *Okcoin) wsProcessAdvancedAlgoOrder(respRaw []byte) error {
	var resp WebsocketAdvancedAlgoOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	algoOrders := make([]order.Detail, len(resp.Data))
	for d := range resp.Data {
		cp, err := currency.NewPairFromString(resp.Data[d].InstrumentID)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(resp.Data[d].OrderType)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[d].AlgoID,
				Err:      err,
			}
		}
		oSide, err := order.StringToOrderSide(resp.Data[d].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[d].AlgoID,
				Err:      err,
			}
		}
		// this code block introduces two order states
		// 1. 'effective' - equivalent to filled, and
		// 2. 'order_failed' == equivalent to failed
		oStatus, err := order.StringToOrderStatus(resp.Data[d].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[d].AlgoID,
				Err:      err,
			}
		}
		algoOrders[d] = order.Detail{
			Leverage:        resp.Data[d].Lever.Float64(),
			Price:           resp.Data[d].OrderPrice.Float64(),
			Amount:          resp.Data[d].Size.Float64(),
			LimitPriceUpper: resp.Data[d].PriceLimit.Float64(),
			TriggerPrice:    resp.Data[d].TriggerPrice.Float64(),
			RemainingAmount: resp.Data[d].ActualSz.Float64(),
			Exchange:        o.Name,
			OrderID:         resp.Data[d].AlgoID,
			ClientOrderID:   resp.Data[d].ClOrdID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Spot,
			Date:            resp.Data[d].CreationTime.Time(),
			LastUpdated:     resp.Data[d].PushTime.Time().UTC(),
			Pair:            cp,
		}
	}
	o.Websocket.DataHandler <- algoOrders
	return nil
}

func (o *Okcoin) wsProcessAlgoOrder(respRaw []byte) error {
	var resp WebsocketAlgoOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(resp.Data))
	for a := range resp.Data {
		cp, err := currency.NewPairFromString(resp.Data[a].InstrumentID)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(resp.Data[a].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[a].OrderID,
				Err:      err,
			}
		}
		var oType order.Type
		oType, err = order.StringToOrderType(resp.Data[a].OrderType)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[a].OrderID,
				Err:      err,
			}
		}
		var status order.Status
		status, err = order.StringToOrderStatus(resp.Data[a].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[a].OrderID,
				Err:      err,
			}
		}
		orderDetails[a] = order.Detail{
			LastUpdated:   resp.Data[a].CreateTime.Time(),
			Exchange:      o.Name,
			AssetType:     asset.Spot,
			Pair:          cp,
			Side:          side,
			OrderID:       resp.Data[a].OrderID,
			ClientOrderID: resp.Data[a].ClientOrderID,
			Price:         resp.Data[a].Price.Float64(),
			Amount:        resp.Data[a].Size.Float64(),
			Type:          oType,
			Status:        status,
			Date:          resp.Data[a].CreateTime.Time(),
			Leverage:      resp.Data[a].Leverage,
			TriggerPrice:  resp.Data[a].TriggerPrice.Float64(),
		}
	}
	o.Websocket.DataHandler <- orderDetails
	return nil
}

func (o *Okcoin) wsProcessOrders(respRaw []byte) error {
	var resp WebsocketOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Arg.InstrumentID)
	if err != nil {
		return err
	}

	algoOrder := make([]order.Detail, len(resp.Data))
	for x := range resp.Data {
		var side order.Side
		side, err := order.StringToOrderSide(resp.Data[x].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[x].OrderID,
				Err:      err,
			}
		}
		var oType order.Type
		oType, err = order.StringToOrderType(resp.Data[x].OrderType)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[x].OrderID,
				Err:      err,
			}
		}
		var status order.Status
		status, err = order.StringToOrderStatus(resp.Data[x].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[x].OrderID,
				Err:      err,
			}
		}
		algoOrder[x] = order.Detail{
			Exchange:             o.Name,
			AssetType:            asset.Spot,
			Pair:                 cp,
			Side:                 side,
			OrderID:              resp.Data[x].OrderID,
			ClientOrderID:        resp.Data[x].ClientOrdID,
			Amount:               resp.Data[x].Size.Float64(),
			Type:                 oType,
			Status:               status,
			Date:                 resp.Data[x].CreateTime.Time(),
			LastUpdated:          resp.Data[x].UpdateTime.Time(),
			ExecutedAmount:       resp.Data[x].FillSize.Float64(),
			ReduceOnly:           resp.Data[x].ReduceOnly,
			Leverage:             resp.Data[x].Leverage.Float64(),
			Price:                resp.Data[x].Price.Float64(),
			AverageExecutedPrice: resp.Data[x].AveragePrice.Float64(),
			RemainingAmount:      resp.Data[x].Size.Float64() - resp.Data[x].FillSize.Float64(),
			Cost:                 resp.Data[x].AveragePrice.Float64() * resp.Data[x].FillSize.Float64(),
			Fee:                  resp.Data[x].Fee.Float64(),
		}
	}
	o.Websocket.DataHandler <- algoOrder
	return nil
}

func (o *Okcoin) wsProcessAccount(respRaw []byte) error {
	var resp WebsocketAccount
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	accountChanges := []account.Change{}
	for a := range resp.Data {
		for b := range resp.Data[a].Details {
			accountChanges = append(accountChanges, account.Change{
				Exchange: o.Name,
				Asset:    asset.Spot,
				Currency: currency.NewCode(resp.Data[a].Details[b].Currency),
				Amount:   resp.Data[a].Details[b].AvailableBalance.Float64()})
		}
	}
	o.Websocket.DataHandler <- accountChanges
	return nil
}

func (o *Okcoin) wsProcessOrderbook(respRaw []byte, obChannel string) error {
	var resp WebsocketOrderbookResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Arg.InstrumentID)
	if err != nil {
		return err
	}
	channel := resp.Arg.Channel
	length := len(resp.Data[0].Asks) + len(resp.Data[0].Bids)
	var snapshot bool
	if channel == wsOrderbooks && length >= 100 ||
		channel == wsOrderbooksL5 ||
		channel == wsOrderbookTickByTickL50 ||
		channel == wsOrderbookTickByTickL400 && length >= 400 ||
		channel == wsOrderbookTickByTickL50 && length >= 50 {
		snapshot = true
	}
	if snapshot {
		resp.Data[0].prepareOrderbook()
		if len(resp.Data[0].Asks)+len(resp.Data[0].Bids) == 0 {
			return nil
		}
		base := orderbook.Base{
			Asset:       asset.Spot,
			Pair:        cp,
			Exchange:    o.Name,
			LastUpdated: resp.Data[0].Timestamp.Time(),
		}
		base.Asks = make([]orderbook.Item, len(resp.Data[0].Asks))
		for a := range resp.Data[0].Asks {
			base.Asks[a].Amount = resp.Data[0].Asks[a][1].Float64()
			base.Asks[a].Price = resp.Data[0].Asks[a][0].Float64()
		}
		base.Bids = make([]orderbook.Item, len(resp.Data[0].Bids))
		for b := range resp.Data[0].Bids {
			base.Bids[b].Amount = resp.Data[0].Bids[b][1].Float64()
			base.Bids[b].Price = resp.Data[0].Bids[b][0].Float64()
		}
		var signedChecksum int32
		signedChecksum, err = o.CalculateChecksum(&resp.Data[0])
		if err != nil {
			return fmt.Errorf("%s channel: Orderbook unable to calculate orderbook checksum: %s", o.Name, err)
		}
		if int64(signedChecksum) != resp.Data[0].Checksum {
			return fmt.Errorf("%s channel: Orderbook for %v checksum invalid",
				o.Name,
				cp)
		}
		err = base.Process()
		if err != nil {
			return err
		}
		err = o.Websocket.Orderbook.LoadSnapshot(&base)
		if err != nil {
			if errors.Is(err, orderbook.ErrOrderbookInvalid) {
				err2 := o.ReSubscribeSpecificOrderbook(obChannel, base.Pair)
				if err2 != nil {
					return err2
				}
			}
			return err
		}
		return nil
	}
	if len(resp.Data[0].Asks)+len(resp.Data[0].Bids) == 0 {
		return nil
	}
	asks, err := o.AppendWsOrderbookItems(resp.Data[0].Asks)
	if err != nil {
		return err
	}
	bids, err := o.AppendWsOrderbookItems(resp.Data[0].Bids)
	if err != nil {
		return err
	}
	update := orderbook.Update{
		Asset:      asset.Spot,
		Pair:       cp,
		UpdateTime: resp.Data[0].Timestamp.Time(),
		Asks:       asks,
		Bids:       bids,
	}
	err = o.Websocket.Orderbook.Update(&update)
	if err != nil {
		if errors.Is(err, orderbook.ErrOrderbookInvalid) {
			err2 := o.ReSubscribeSpecificOrderbook(obChannel, update.Pair)
			if err2 != nil {
				return err2
			}
		}
		return err
	}
	updatedOb, err := o.Websocket.Orderbook.GetOrderbook(cp, asset.Spot)
	if err != nil {
		return err
	}
	checksum := o.CalculateOrderbookUpdateChecksum(updatedOb)
	if int64(checksum) != resp.Data[0].Checksum {
		return fmt.Errorf("checksum failed, calculated '%v' received '%v'", checksum, resp.Data)
	}
	return nil
}

// ReSubscribeSpecificOrderbook removes the subscription and the subscribes
// again to fetch a new snapshot in the event of a de-sync event.
func (o *Okcoin) ReSubscribeSpecificOrderbook(obChannel string, p currency.Pair) error {
	subscription := []stream.ChannelSubscription{{
		Channel:  obChannel,
		Currency: p,
	}}
	if err := o.Unsubscribe(subscription); err != nil {
		return err
	}
	return o.Subscribe(subscription)
}

// wsProcessInstruments converts instrument data and sends it to the datahandler
func (o *Okcoin) wsProcessInstruments(respRaw []byte) error {
	var response []WebsocketInstrumentData
	resp := WebsocketDataResponseReciever{
		Data: &response,
	}
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- response
	return nil
}

// wsProcessTickers  converts ticker data and sends it to the datahandler
func (o *Okcoin) wsProcessTickers(respRaw []byte) error {
	var response []WsTickerData
	resp := WebsocketDataResponseReciever{
		Data: &response,
	}
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	tickers := make([]ticker.Price, len(response))
	for x := range response {
		pair, err := currency.NewPairFromString(response[x].InstrumentID)
		if err != nil {
			return err
		}
		tickers[x] = ticker.Price{
			AssetType:    asset.Spot,
			Last:         response[x].Last.Float64(),
			Open:         response[x].Open24H.Float64(),
			High:         response[x].High24H.Float64(),
			Low:          response[x].Low24H.Float64(),
			Volume:       response[x].Vol24H.Float64(),
			QuoteVolume:  response[x].VolCcy24H.Float64(),
			Bid:          response[x].BidPrice.Float64(),
			BidSize:      response[x].BidSize.Float64(),
			Ask:          response[x].AskPrice.Float64(),
			AskSize:      response[x].AskSize.Float64(),
			LastUpdated:  response[x].Timestamp.Time(),
			ExchangeName: o.Name,
			Pair:         pair,
		}
	}
	o.Websocket.DataHandler <- tickers
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (o *Okcoin) wsProcessTrades(respRaw []byte) error {
	if !o.IsSaveTradeDataEnabled() {
		return nil
	}
	var response WebsocketTradeResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(response.Data))
	for i := range response.Data {
		instrument, err := currency.NewPairFromString(response.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		tSide, err := order.StringToOrderSide(response.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				Err:      err,
			}
		}
		trades[i] = trade.Data{
			Amount:       response.Data[i].Size.Float64(),
			AssetType:    asset.Spot,
			CurrencyPair: instrument,
			Exchange:     o.Name,
			Price:        response.Data[i].Price.Float64(),
			Side:         tSide,
			Timestamp:    response.Data[i].Timestamp.Time(),
			TID:          response.Data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(o.Name, trades...)
}

// wsProcessCandles converts candle data and sends it to the data handler
func (o *Okcoin) wsProcessCandles(respRaw []byte) error {
	var response WebsocketCandlesResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	candlesticks, err := o.GetCandlesData(&response)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- candlesticks
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an
// orderbook item array
func (o *Okcoin) AppendWsOrderbookItems(entries [][2]okcoinNumber) ([]orderbook.Item, error) {
	items := make([]orderbook.Item, len(entries))
	for j := range entries {
		amount := entries[j][1].Float64()
		price := entries[j][0].Float64()
		items[j] = orderbook.Item{Amount: amount, Price: price}
	}
	return items, nil
}

// CalculateChecksum alternates over the first 25 bid and ask
// entries from websocket data. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *Okcoin) CalculateChecksum(orderbookData *WebsocketOrderBook) (int32, error) {
	orderbookData.prepareOrderbook()
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice := orderbookData.Bids[i][0]
			bidAmount := orderbookData.Bids[i][1]
			checksum.WriteString(bidPrice.String() +
				currency.ColonDelimiter +
				bidAmount.String() +
				currency.ColonDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice := orderbookData.Asks[i][0]
			askAmount := orderbookData.Asks[i][1]
			checksum.WriteString(askPrice.String() +
				currency.ColonDelimiter +
				askAmount.String() +
				currency.ColonDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), currency.ColonDelimiter)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr))), nil
}

// CalculateOrderbookUpdateChecksum calculated the orderbook update checksum using currency pair full snapshot.
func (o *Okcoin) CalculateOrderbookUpdateChecksum(orderbookData *orderbook.Base) int32 {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			bidAmount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(bidPrice +
				currency.ColonDelimiter +
				bidAmount +
				currency.ColonDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			askAmount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(askPrice +
				currency.ColonDelimiter +
				askAmount +
				currency.ColonDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), currency.ColonDelimiter)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be
// handled by ManageSubscriptions()
func (o *Okcoin) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	pairs, err := o.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	spotFormat, err := o.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(spotFormat)
	channels := defaultSubscriptions
	if o.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(
			channels,
			wsAccount,
			wsOrder,
			wsOrdersAlgo,
			wsAlgoAdvance,
		)
	}
	for s := range channels {
		switch channels[s] {
		case wsInstruments:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[s],
				Asset:   asset.Spot,
			})
		case wsTickers, wsTrades, wsOrderbooks, wsOrderbooksL5, wsOrderbookL1, wsOrderbookTickByTickL50,
			wsOrderbookTickByTickL400, wsCandle3M, wsCandle1M, wsCandle1W, wsCandle1D,
			wsCandle2D, wsCandle3D, wsCandle5D,
			wsCandle12H, wsCandle6H, wsCandle4H, wsCandle2H, wsCandle1H, wsCandle30m, wsCandle15m,
			wsCandle5m, wsCandle3m, wsCandle1m, wsCandle3Mutc, wsCandle1Mutc, wsCandle1Wutc, wsCandle1Dutc,
			wsCandle2Dutc, wsCandle3Dutc, wsCandle5Dutc, wsCandle12Hutc, wsCandle6Hutc:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[s],
					Currency: pairs[p],
				})
			}
		case wsStatus:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[s],
			})
		case wsAccount:
			currenciesMap := map[currency.Code]bool{}
			for p := range pairs {
				if reserved, okay := currenciesMap[pairs[p].Base]; !okay && !reserved {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel: channels[s],
						Params: map[string]interface{}{
							"ccy": pairs[p].Base,
						},
					})
					currenciesMap[pairs[p].Base] = true
				}
			}
			for p := range pairs {
				if reserved, okay := currenciesMap[pairs[p].Quote]; !okay && !reserved {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel: channels[s],
						Params: map[string]interface{}{
							"ccy": pairs[p].Quote,
						},
					})
					currenciesMap[pairs[p].Quote] = true
				}
			}
		case wsOrder, wsOrdersAlgo, wsAlgoAdvance:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[s],
					Currency: pairs[p],
					Asset:    asset.Spot,
				})
			}
		default:
			return nil, fmt.Errorf("unsupported websocket channel %v", channels[s])
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (o *Okcoin) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("subscribe", channelsToSubscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (o *Okcoin) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("unsubscribe", channelsToUnsubscribe)
}

func (o *Okcoin) handleSubscriptions(operation string, subs []stream.ChannelSubscription) error {
	subscriptionRequest := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	authRequest := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	temp := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	authTemp := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	var err error
	var channels []stream.ChannelSubscription
	var authChannels []stream.ChannelSubscription
	for i := 0; i < len(subs); i++ {
		authenticatedChannelSubscription := isAuthenticatedChannel(subs[i].Channel)
		// Temp type to evaluate max byte len after a marshal on batched unsubs
		copy(temp.Arguments, subscriptionRequest.Arguments)
		copy(authTemp.Arguments, authRequest.Arguments)
		argument := map[string]string{
			"channel": subs[i].Channel,
		}
		if subs[i].Params != nil {
			if ccy, okay := subs[i].Params["ccy"]; okay {
				argument["ccy"], okay = ccy.(string)
				if !okay {
					continue
				}
			}
			if interval, okay := subs[i].Params["interval"]; okay {
				intervalString, okay := interval.(string)
				if !okay {
					continue
				}
				argument["channel"] += intervalString
			}
		}
		if subs[i].Asset != asset.Empty {
			argument["instType"] = strings.ToUpper(subs[i].Asset.String())
		}
		if !subs[i].Currency.IsEmpty() {
			argument["instId"] = subs[i].Currency.String()
		}
		if authenticatedChannelSubscription {
			authTemp.Arguments = append(authTemp.Arguments, argument)
		} else {
			temp.Arguments = append(temp.Arguments, argument)
		}
		var chunk []byte
		if authenticatedChannelSubscription {
			chunk, err = json.Marshal(authRequest)
			if err != nil {
				return err
			}
		} else {
			chunk, err = json.Marshal(subscriptionRequest)
			if err != nil {
				return err
			}
		}

		if len(chunk) > maxConnByteLen {
			// If temp chunk exceeds max byte length determined by the exchange,
			// commit last payload.
			i-- // reverse position in range to reuse channel unsubscription on
			// next iteration
			if authenticatedChannelSubscription {
				err = o.Websocket.AuthConn.SendJSONMessage(authRequest)
			} else {
				err = o.Websocket.Conn.SendJSONMessage(subscriptionRequest)
			}
			if err != nil {
				return err
			}

			if operation == "unsubscribe" {
				if authenticatedChannelSubscription {
					o.Websocket.RemoveSuccessfulUnsubscriptions(authChannels...)
				} else {
					o.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
				}
			} else {
				if authenticatedChannelSubscription {
					o.Websocket.AddSuccessfulSubscriptions(authChannels...)
				} else {
					o.Websocket.AddSuccessfulSubscriptions(channels...)
				}
			}
			// Drop prior unsubs and chunked payload args on successful unsubscription
			if authenticatedChannelSubscription {
				authChannels = nil
				authRequest.Arguments = nil
			} else {
				channels = nil
				subscriptionRequest.Arguments = nil
			}
			continue
		}
		// Add pending chained items
		channels = append(channels, subs[i])
		if authenticatedChannelSubscription {
			authRequest.Arguments = authTemp.Arguments
		} else {
			subscriptionRequest.Arguments = temp.Arguments
		}
	}
	if len(subscriptionRequest.Arguments) > 0 {
		err = o.Websocket.Conn.SendJSONMessage(subscriptionRequest)
		if err != nil {
			return err
		}
	}
	if len(authRequest.Arguments) > 0 {
		err = o.Websocket.AuthConn.SendJSONMessage(authRequest)
		if err != nil {
			return err
		}
	}
	if operation == "unsubscribe" {
		o.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
	} else {
		o.Websocket.AddSuccessfulSubscriptions(channels...)
	}
	return nil
}

// GetCandlesData represents a candlestick instances list.
func (o *Okcoin) GetCandlesData(arg *WebsocketCandlesResponse) ([]stream.KlineData, error) {
	candlesticks := make([]stream.KlineData, len(arg.Data))
	cp, err := currency.NewPairFromString(arg.Arg.InstrumentID)
	if err != nil {
		return nil, err
	}
	for x := range arg.Data {
		if len(arg.Data[x]) < 6 {
			return nil, fmt.Errorf("%w expected a kline data of length 6 but found %d", kline.ErrInsufficientCandleData, len(arg.Data[x]))
		}
		var timestamp int64
		timestamp, err = strconv.ParseInt(arg.Data[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].AssetType = asset.Spot
		candlesticks[x].Pair = cp
		candlesticks[x].Timestamp = time.UnixMilli(timestamp)
		candlesticks[x].Exchange = o.Name
		candlesticks[x].OpenPrice, err = strconv.ParseFloat(arg.Data[x][1], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].HighPrice, err = strconv.ParseFloat(arg.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].LowPrice, err = strconv.ParseFloat(arg.Data[x][3], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].ClosePrice, err = strconv.ParseFloat(arg.Data[x][4], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].Volume, err = strconv.ParseFloat(arg.Data[x][5], 64)
		if err != nil {
			return nil, err
		}
	}
	return candlesticks, nil
}

func systemStatusServiceTypeString(serviceType int64) string {
	switch serviceType {
	case 0:
		return "Websocket"
	case 1:
		return "Classic account"
	case 5:
		return "Unified account"
	case 99:
		return "Unknown"
	default:
		return ""
	}
}

func systemStateString(state string) string {
	switch state {
	case "scheduled":
		return "Scheduled"
	case "ongoing":
		return "Ongoing"
	case "pre_open":
		return "Pre-Open"
	case "completed":
		return "Completed"
	case "canceled":
		return "Canceled"
	default:
		return ""
	}
}
