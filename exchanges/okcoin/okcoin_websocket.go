package okcoin

import (
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
	// wsTickers,
	// wsCandle1D,
	// wsTrades,
	wsOrderbooks,
	// wsStatus,
}

func isAuthenticatedChannel(channel string) bool {
	switch channel {
	case wsAccount, wsOrder, wsOrdersAlgo, wsAlgoAdvance:
		return true
	default:
		return false
	}
}

// WsConnect initiates a websocket connection
func (o *OKCoin) WsConnect() error {
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

	o.Websocket.Wg.Add(2)
	go o.funnelWebsocketConn(o.Websocket.Conn)
	go o.WsReadData()

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
func (o *OKCoin) WsLogin(ctx context.Context, dialer *websocket.Dialer) error {
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
	go o.funnelWebsocketConn(o.Websocket.AuthConn)
	o.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		Delay:       time.Second * 25,
		Message:     []byte("ping"),
		MessageType: websocket.TextMessage,
	})
	systemTime, err := o.GetSystemTime(context.Background())
	if err != nil {
		systemTime = time.Now().UTC()
	}
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(systemTime.UTC().Unix(), 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64 := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: "login",
		Arguments: []map[string]string{
			{
				"apiKey":     creds.Key,
				"passphrase": creds.ClientID,
				"timestamp":  strconv.FormatInt(systemTime.UTC().Unix(), 10),
				"sign":       base64,
			},
		},
	}
	_, err = o.Websocket.AuthConn.SendMessageReturnResponse("login", request)
	if err != nil {
		return err
	}
	o.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

var messageChan = make(chan stream.Response)

func (o *OKCoin) funnelWebsocketConn(conn stream.Connection) {
	defer o.Websocket.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		messageChan <- resp
	}
}

// WsReadData receives and passes on websocket messages for processing
func (o *OKCoin) WsReadData() {
	defer o.Websocket.Wg.Done()
	for {
		select {
		case <-o.Websocket.ShutdownC:
			select {
			case resp := <-messageChan:
				err := o.WsHandleData(resp.Raw)
				if err != nil {
					select {
					case o.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", o.Name, err)
					}
				}
			default:
			}
			return
		case data := <-messageChan:
			err := o.WsHandleData(data.Raw)
			if err != nil {
				o.Websocket.DataHandler <- err
			}
			err = o.WsHandleData(data.Raw)
			if err != nil {
				select {
				case o.Websocket.DataHandler <- err:
				default:
					log.Errorf(log.WebsocketMgr,
						"%s websocket handle data error: %v",
						o.Name,
						err)
				}
			}
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (o *OKCoin) WsHandleData(respRaw []byte) error {
	println(string(respRaw))
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
			return o.wsProcessOrderbook(respRaw)
		case wsStatus:
			var resp WebsocketStatus
			err = json.Unmarshal(respRaw, &resp)
			if err != nil {
				return err
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
			waitingSignatureLock.Lock()
			for x := range waitingSignatures {
				if strings.Contains(dataResponse.Message, waitingSignatures[x]) {
					o.Websocket.Match.IncomingWithData(waitingSignatures[x], respRaw)
					return nil
				}
			}
			waitingSignatureLock.Unlock()
			if o.Verbose {
				log.Debugf(log.ExchangeSys,
					o.Name+" - "+eventResponse.Event+" on channel: "+eventResponse.Channel)
			}
		}
	}
	return nil
}

func (o *OKCoin) wsProcessAdvancedAlgoOrder(respRaw []byte) error {
	var resp WebsocketAdvancedAlgoOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- resp
	return nil
}

func (o *OKCoin) wsProcessAlgoOrder(respRaw []byte) error {
	var resp WebsocketAlgoOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- resp
	return nil
}

func (o *OKCoin) wsProcessOrders(respRaw []byte) error {
	var resp WebsocketOrder
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Arg.InstID)
	if err != nil {
		return err
	}

	algoOrder := make([]fill.Data, len(resp.Data))
	for x := range resp.Data {
		side, err := order.StringToOrderSide(resp.Data[x].PositionSide)
		if err != nil {
			return err
		}
		algoOrder[x] = fill.Data{
			ID:            resp.Data[x].OrderID,
			Timestamp:     resp.Data[x].CreateTime.Time(),
			Exchange:      o.Name,
			AssetType:     asset.Spot,
			CurrencyPair:  cp,
			Side:          side,
			OrderID:       resp.Data[x].OrderID,
			ClientOrderID: resp.Data[x].ClientOrdID,
			TradeID:       resp.Data[x].TradeID,
			Price:         resp.Data[x].FillPrice,
			Amount:        resp.Data[x].Size,
		}
	}
	o.Websocket.DataHandler <- algoOrder
	return nil
}

func (o *OKCoin) wsProcessAccount(respRaw []byte) error {
	var resp WebsocketAccount
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- &resp
	return nil
}

// StringToOrderStatus converts order status IDs to internal types
func StringToOrderStatus(num int64) (order.Status, error) {
	switch num {
	case -2:
		return order.Rejected, nil
	case -1:
		return order.Cancelled, nil
	case 0:
		return order.Active, nil
	case 1:
		return order.PartiallyFilled, nil
	case 2:
		return order.Filled, nil
	case 3:
		return order.New, nil
	case 4:
		return order.PendingCancel, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%v not recognised as order status", num)
	}
}

func (o *OKCoin) wsProcessOrderbook(respRaw []byte) error {
	println(" Orderbook data ")
	var resp WebsocketOrderbookResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(resp.Arg.InstID)
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
		base := orderbook.Base{
			Asset:       asset.Spot,
			Pair:        cp,
			Exchange:    o.Name,
			LastUpdated: resp.Data[0].Timestamp.Time(),
		}
		for x := range resp.Data {
			base.Asks = make([]orderbook.Item, len(resp.Data[x].Asks))
			for a := range resp.Data[x].Asks {
				base.Asks[a].Amount, err = strconv.ParseFloat(resp.Data[x].Asks[a][0], 64)
				if err != nil {
					return err
				}
				base.Asks[a].Price, err = strconv.ParseFloat(resp.Data[x].Asks[a][1], 64)
				if err != nil {
					return err
				}
			}
			base.Bids = make([]orderbook.Item, len(resp.Data[x].Bids))
			for b := range resp.Data[x].Bids {
				base.Bids[b].Amount, err = strconv.ParseFloat(resp.Data[x].Bids[b][0], 64)
				if err != nil {
					return err
				}
				base.Bids[b].Price, err = strconv.ParseFloat(resp.Data[x].Bids[b][1], 64)
				if err != nil {
					return err
				}
			}
			var signedChecksum int32
			signedChecksum, err = o.CalculatePartialOrderbookChecksum(&resp.Data[x])
			if err != nil {
				return fmt.Errorf("%s channel: Orderbook unable to calculate orderbook checksum: %s", o.Name, err)
			}
			if signedChecksum != resp.Data[0].Checksum {
				return fmt.Errorf("%s channel: Orderbook for %v checksum invalid",
					o.Name,
					cp)
			}
		}
		err = base.Process()
		if err != nil {
			return err
		}
		println(" Loading snapshot ... ")
		return o.Websocket.Orderbook.LoadSnapshot(&base)
	}

	update := orderbook.Update{
		Asset: asset.Spot,
		Pair:  cp,
	}
	for x := range resp.Data {
		update.Asks = make([]orderbook.Item, len(resp.Data[x].Asks))
		for a := range resp.Data[x].Asks {
			update.Asks[a].Amount, err = strconv.ParseFloat(resp.Data[x].Asks[a][1], 64)
			if err != nil {
				return err
			}
			update.Asks[a].Price, err = strconv.ParseFloat(resp.Data[x].Asks[a][0], 64)
			if err != nil {
				return err
			}
		}
		update.Bids = make([]orderbook.Item, len(resp.Data[x].Bids))
		for b := range resp.Data[x].Bids {
			update.Bids[b].Amount, err = strconv.ParseFloat(resp.Data[x].Bids[b][1], 64)
			if err != nil {
				return err
			}
			update.Bids[b].Price, err = strconv.ParseFloat(resp.Data[x].Bids[b][0], 64)
			if err != nil {
				return err
			}
		}
		// println(string(respRaw))
		updateChecksum := o.CalculateUpdateOrderbookChecksum(&update)
		println("Sent checksum: ", updateChecksum, "Calculated checksum: ", uint32(resp.Data[x].Checksum))
		if uint32(updateChecksum) != uint32(resp.Data[x].Checksum) {
			return fmt.Errorf("%s channel: Orderbook unable to calculate orderbook checksum: %s", o.Name, err)
		}
	}
	println(" Update orderbook ")
	return o.Websocket.Orderbook.Update(&update)
}

func (o *OKCoin) wsProcessOrder(respRaw []byte) error {
	var resp WebsocketSpotOrderResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for i := range resp.Data {
		var oType order.Type
		var oSide order.Side
		var oStatus order.Status
		oType, err = order.StringToOrderType(resp.Data[i].Type)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oSide, err = order.StringToOrderSide(resp.Data[i].Side)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}
		oStatus, err = StringToOrderStatus(resp.Data[i].State)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}

		pair, err := currency.NewPairFromString(resp.Data[i].InstrumentID)
		if err != nil {
			o.Websocket.DataHandler <- order.ClassificationError{
				Exchange: o.Name,
				OrderID:  resp.Data[i].OrderID,
				Err:      err,
			}
		}

		o.Websocket.DataHandler <- &order.Detail{
			ImmediateOrCancel: resp.Data[i].OrderType == 3,
			FillOrKill:        resp.Data[i].OrderType == 2,
			PostOnly:          resp.Data[i].OrderType == 1,
			Price:             resp.Data[i].Price,
			Amount:            resp.Data[i].Size,
			ExecutedAmount:    resp.Data[i].LastFillQty,
			RemainingAmount:   resp.Data[i].Size - resp.Data[i].LastFillQty,
			Exchange:          o.Name,
			OrderID:           resp.Data[i].OrderID,
			Type:              oType,
			Side:              oSide,
			Status:            oStatus,
			AssetType:         o.GetAssetTypeFromTableName(resp.Table),
			Date:              resp.Data[i].CreatedAt,
			Pair:              pair,
		}
	}
	return nil
}

// wsProcessInstruments converts instrument data and sends it to the datahandler
func (o *OKCoin) wsProcessInstruments(respRaw []byte) error {
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
func (o *OKCoin) wsProcessTickers(respRaw []byte) error {
	var response []WsTickerData
	resp := WebsocketDataResponseReciever{
		Data: &response,
	}
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for x := range response {
		pair, err := currency.NewPairFromString(response[x].InstrumentID)
		if err != nil {
			return err
		}
		o.Websocket.DataHandler <- &ticker.Price{
			AssetType:    asset.Spot,
			Last:         response[x].Last,
			Open:         response[x].Open24H,
			High:         response[x].High24H,
			Low:          response[x].Low24H,
			Volume:       response[x].Vol24H,
			QuoteVolume:  response[x].VolCcy24H,
			Bid:          response[x].BidPrice,
			BidSize:      response[x].BidSize,
			Ask:          response[x].AskPrice,
			AskSize:      response[x].AskSize,
			LastUpdated:  response[x].Timestamp.Time(),
			ExchangeName: o.Name,
			Pair:         pair,
		}
	}
	return nil
}

// wsProcessTrades converts trade data and sends it to the datahandler
func (o *OKCoin) wsProcessTrades(respRaw []byte) error {
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
			Amount:       response.Data[i].Size,
			AssetType:    asset.Spot,
			CurrencyPair: instrument,
			Exchange:     o.Name,
			Price:        response.Data[i].Price,
			Side:         tSide,
			Timestamp:    response.Data[i].Timestamp.Time(),
			TID:          response.Data[i].TradeID,
		}
	}
	return trade.AddTradesToBuffer(o.Name, trades...)
}

// wsProcessCandles converts candle data and sends it to the data handler
func (o *OKCoin) wsProcessCandles(respRaw []byte) error {
	var response WebsocketCandlesResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}

	candlesticks, err := response.GetCandlesData(o.Name)
	if err != nil {
		return err
	}
	o.Websocket.DataHandler <- candlesticks
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an
// orderbook item array
func (o *OKCoin) AppendWsOrderbookItems(entries [][]interface{}) ([]orderbook.Item, error) {
	items := make([]orderbook.Item, len(entries))
	for j := range entries {
		amount, err := strconv.ParseFloat(entries[j][1].(string), 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(entries[j][0].(string), 64)
		if err != nil {
			return nil, err
		}
		items[j] = orderbook.Item{Amount: amount, Price: price}
	}
	return items, nil
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask
// entries from websocket data. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKCoin) CalculatePartialOrderbookChecksum(orderbookData *WebsocketOrderBook) (int32, error) {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice := orderbookData.Bids[i][0]
			bidAmount := orderbookData.Bids[i][1]
			checksum.WriteString(bidPrice +
				delimiterColon +
				bidAmount +
				delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice := orderbookData.Asks[i][0]
			askAmount := orderbookData.Asks[i][1]
			checksum.WriteString(askPrice +
				delimiterColon +
				askAmount +
				delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr))), nil
}

// CalculateUpdateOrderbookChecksum alternates over the first 25 bid and ask
// entries of a merged orderbook. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (o *OKCoin) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Update) uint32 {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + delimiterColon + amount + delimiterColon)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), delimiterColon)
	return crc32.ChecksumIEEE([]byte(checksumStr))
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be
// handled by ManageSubscriptions()
func (o *OKCoin) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	assets := o.GetAssetTypes(true)
	for x := range assets {
		if assets[x] != asset.Spot {
			continue
		}
		pairs, err := o.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}
		spotFormat, err := o.GetPairFormat(assets[x], true)
		if err != nil {
			return nil, err
		}
		pairs = pairs.Format(spotFormat)
		channels := defaultSubscriptions
		if o.Websocket.CanUseAuthenticatedEndpoints() {
			channels = append(
				channels,
				// wsAccount,
				// wsOrder,
				// wsOrdersAlgo,
				// wsAlgoAdvance,
			)
		}
		for s := range channels {
			switch channels[s] {
			case wsInstruments:
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel: channels[s],
					Asset:   assets[x],
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
						Asset:    assets[x],
					})
				}
			default:
				return nil, errors.New("unsupported websocket channel")
			}
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (o *OKCoin) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("subscribe", channelsToSubscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (o *OKCoin) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return o.handleSubscriptions("unsubscribe", channelsToUnsubscribe)
}

func (o *OKCoin) handleSubscriptions(operation string, subs []stream.ChannelSubscription) error {
	request := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	authRequest := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	temp := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	authTemp := WebsocketEventRequest{Operation: operation, Arguments: []map[string]string{}}
	var err error
	var channels []stream.ChannelSubscription
	var authChannels []stream.ChannelSubscription
	for i := 0; i < len(subs); i++ {
		authenticatedChannelSubscription := isAuthenticatedChannel(subs[i].Channel)
		// Temp type to evaluate max byte len after a marshal on batched unsubs
		copy(temp.Arguments, request.Arguments)
		copy(authTemp.Arguments, authRequest.Arguments)
		argument := map[string]string{
			"channel": subs[i].Channel,
		}
		if subs[i].Params != nil {
			if currency, okay := subs[i].Params["ccy"]; okay {
				argument["ccy"], okay = (currency).(string)
				if !okay {
					continue
				}
			}
			if interval, okay := subs[i].Params["interval"]; okay {
				intervalString, okay := (interval).(string)
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
			chunk, err = json.Marshal(request)
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
				err = o.Websocket.Conn.SendJSONMessage(request)
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
				request.Arguments = nil
			}
			continue
		}
		// Add pending chained items
		channels = append(channels, subs[i])
		if authenticatedChannelSubscription {
			authRequest.Arguments = authTemp.Arguments
		} else {
			request.Arguments = temp.Arguments
		}
	}
	if len(request.Arguments) > 0 {
		err = o.Websocket.Conn.SendJSONMessage(request)
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

// GetWsChannelWithoutOrderType takes WebsocketDataResponse.Table and returns
// The base channel name eg receive "spot/depth5:BTC-USDT" return "depth5"
func (o *OKCoin) GetWsChannelWithoutOrderType(table string) string {
	index := strings.Index(table, "/")
	if index == -1 {
		return table
	}
	channel := table[index+1:]
	index = strings.Index(channel, ":")
	// Some events do not contain a currency
	if index == -1 {
		return channel
	}

	return channel[:index]
}

// GetAssetTypeFromTableName gets the asset type from the table name
// eg "spot/ticker:BTCUSD" results in "SPOT"
func (o *OKCoin) GetAssetTypeFromTableName(table string) asset.Item {
	assetIndex := strings.Index(table, "/")
	switch table[:assetIndex] {
	case asset.Spot.String():
		return asset.Spot
	default:
		log.Warnf(log.ExchangeSys, "%s unhandled asset type %s",
			o.Name,
			table[:assetIndex])
		return asset.Empty
	}
}
