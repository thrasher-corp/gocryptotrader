package zb

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	zbWebsocketAPI       = "wss://api.zb.cn:9999/websocket"
	zWebsocketAddChannel = "addChannel"
)

// WsConnect initiates a websocket connection
func (z *ZB) WsConnect() error {
	if !z.Websocket.IsEnabled() || !z.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	z.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName: z.Name,
		URL:          z.Websocket.GetWebsocketURL(),
		Verbose:      z.Verbose,
		RateLimit:    20,
	}
	var dialer websocket.Dialer
	err := z.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	go z.WsHandleData()
	z.GenerateDefaultSubscriptions()

	return nil
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (z *ZB) WsReadData() (wshandler.WebsocketResponse, error) {
	resp, err := z.WebsocketConn.ReadMessage()
	if err != nil {
		return wshandler.WebsocketResponse{}, err
	}

	z.Websocket.TrafficAlert <- struct{}{}
	return resp, nil
}

// WsHandleData handles all the websocket data coming from the websocket
// connection
func (z *ZB) WsHandleData() {
	z.Websocket.Wg.Add(1)

	defer func() {
		z.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-z.Websocket.ShutdownC:
			return
		default:
			resp, err := z.WsReadData()
			if err != nil {
				z.Websocket.DataHandler <- err
				return
			}
			fixedJSON := z.wsFixInvalidJSON(resp.Raw)
			var result Generic
			err = common.JSONDecode(fixedJSON, &result)
			if err != nil || !result.Success {
				z.Websocket.DataHandler <- err
				continue
			}
			switch {
			case common.StringContains(result.Channel, "markets"):
				if !result.Success {
					z.Websocket.DataHandler <- fmt.Errorf("zb_websocket.go error - unsuccessful market response %s", wsErrCodes[result.Code])
					continue
				}

				var markets Markets
				err := common.JSONDecode(result.Data, &markets)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

			case common.StringContains(result.Channel, "ticker"):
				cPair := common.SplitStrings(result.Channel, "_")

				var ticker WsTicker

				err := common.JSONDecode(fixedJSON, &ticker)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				z.Websocket.DataHandler <- wshandler.TickerData{
					Timestamp:  time.Unix(0, ticker.Date),
					Pair:       currency.NewPairFromString(cPair[0]),
					AssetType:  "SPOT",
					Exchange:   z.GetName(),
					ClosePrice: ticker.Data.Last,
					HighPrice:  ticker.Data.High,
					LowPrice:   ticker.Data.Low,
				}

			case common.StringContains(result.Channel, "depth"):
				var depth WsDepth
				err := common.JSONDecode(fixedJSON, &depth)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				var asks []orderbook.Item
				for _, askDepth := range depth.Asks {
					ask := askDepth.([]interface{})
					asks = append(asks, orderbook.Item{
						Amount: ask[1].(float64),
						Price:  ask[0].(float64),
					})
				}

				var bids []orderbook.Item
				for _, bidDepth := range depth.Bids {
					bid := bidDepth.([]interface{})
					bids = append(bids, orderbook.Item{
						Amount: bid[1].(float64),
						Price:  bid[0].(float64),
					})
				}

				channelInfo := common.SplitStrings(result.Channel, "_")
				cPair := currency.NewPairFromString(channelInfo[0])

				var newOrderBook orderbook.Base
				newOrderBook.Asks = asks
				newOrderBook.Bids = bids
				newOrderBook.AssetType = "SPOT"
				newOrderBook.Pair = cPair

				err = z.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
					z.GetName(),
					true)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				z.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
					Pair:     cPair,
					Asset:    "SPOT",
					Exchange: z.GetName(),
				}

			case common.StringContains(result.Channel, "trades"):
				var trades WsTrades
				err := common.JSONDecode(fixedJSON, &trades)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				// Most up to date trade
				if len(trades.Data) == 0 {
					continue
				}
				t := trades.Data[len(trades.Data)-1]

				channelInfo := common.SplitStrings(result.Channel, "_")
				cPair := currency.NewPairFromString(channelInfo[0])

				z.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    time.Unix(0, t.Date),
					CurrencyPair: cPair,
					AssetType:    "SPOT",
					Exchange:     z.GetName(),
					EventTime:    t.Date,
					Price:        t.Price,
					Amount:       t.Amount,
					Side:         t.TradeType,
				}
			default:
				if result.No > 0 {
					if z.WebsocketConn.IDResponses == nil {
						z.WebsocketConn.IDResponses = make(map[int64][]byte)
					}
					z.WebsocketConn.IDResponses[result.No] = fixedJSON
					continue
				}
				z.Websocket.DataHandler <- errors.New("zb_websocket.go error - unhandled websocket response")
				continue
			}
		}
	}
}

var wsErrCodes = map[int64]string{
	1000: "Successful call",
	1001: "General error message",
	1002: "internal error",
	1003: "Verification failed",
	1004: "Financial security password lock",
	1005: "The fund security password is incorrect. Please confirm and re-enter.",
	1006: "Real-name certification is awaiting review or review",
	1007: "Channel is empty",
	1008: "Event is empty",
	1009: "This interface is being maintained",
	1011: "Not open yet",
	1012: "Insufficient permissions",
	1013: "Can not trade, if you have any questions, please contact online customer service",
	1014: "Cannot be sold during the pre-sale period",
	2002: "Insufficient balance in Bitcoin account",
	2003: "Insufficient balance of Litecoin account",
	2005: "Insufficient balance in Ethereum account",
	2006: "Insufficient balance in ETC currency account",
	2007: "Insufficient balance of BTS currency account",
	2008: "Insufficient balance in EOS currency account",
	2009: "Insufficient account balance",
	3001: "Pending order not found",
	3002: "Invalid amount",
	3003: "Invalid quantity",
	3004: "User does not exist",
	3005: "Invalid parameter",
	3006: "Invalid IP or inconsistent with the bound IP",
	3007: "Request time has expired",
	3008: "Transaction history not found",
	4001: "API interface is locked",
	4002: "Request too frequently",
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (z *ZB) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	// Tickerdata is its own channel
	subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
		Channel: "markets",
	})
	channels := []string{"%s_ticker", "%s_depth", "%s_trades"}
	enabledCurrencies := z.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf(channels[i], enabledCurrencies[j].Lower().String()),
				Currency: enabledCurrencies[j].Lower(),
			})
		}
	}
	z.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (z *ZB) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscriptionRequest := Subscription{
		Event:   zWebsocketAddChannel,
		Channel: channelToSubscribe.Channel,
	}
	return z.WebsocketConn.SendMessage(subscriptionRequest)
}

func (z *ZB) wsGenerateSignature(request interface{}) string {
	jsonResponse, err := common.JSONEncode(request)
	if err != nil {
		log.Error(err)
	}
	hmac := common.GetHMAC(common.HashMD5,
		jsonResponse,
		[]byte(common.Sha1ToHex(z.APISecret)))
	return fmt.Sprintf("%x", hmac)
}

func (z *ZB) wsFixInvalidJSON(json []byte) []byte {
	invalidZbJSONRegex := `(\"\[|\"\{)(.*)(\]\"|\}\")`
	regexChecker := regexp.MustCompile(invalidZbJSONRegex)
	matchingResults := regexChecker.Find(json)
	if matchingResults == nil {
		return json
	}
	// Remove first quote character
	capturedInvalidZBJSON := common.ReplaceString(string(matchingResults), "\"", "", 1)
	// Remove last quote character
	fixedJSON := capturedInvalidZBJSON[:len(capturedInvalidZBJSON)-1]
	return []byte(common.ReplaceString(string(json), string(matchingResults), fixedJSON, 1))
}

func (z *ZB) wsAddSubUser(username, password string) (*WsGetSubUserListResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetSubUserListResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAddSubUserRequest{
		Memo:        "memo",
		Password:    password,
		SubUserName: username,
	}
	request.Channel = "addSubUser"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)
	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetSubUserListResponse{}, err
	}
	var response WsGetSubUserListResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsGetSubUserList() (*WsGetSubUserListResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetSubUserListResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAuthenticatedRequest{}
	request.Channel = "getSubUserList"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.No = z.WebsocketConn.GenerateMessageID(true)
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetSubUserListResponse{}, err
	}
	var response WsGetSubUserListResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsDoTransferFunds(pair currency.Code, amount float64, fromUserName, toUserName string) (*WsRequestResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsRequestResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsDoTransferFundsRequest{
		Amount:       amount,
		Currency:     pair,
		FromUserName: fromUserName,
		ToUserName:   toUserName,
		No:           z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = "doTransferFunds"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsRequestResponse{}, err
	}
	var response WsRequestResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsCreateSubUserKey(assetPerm, entrustPerm, leverPerm, moneyPerm bool, keyName, toUserID string) (*WsRequestResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsRequestResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsCreateSubUserKeyRequest{
		AssetPerm:   assetPerm,
		EntrustPerm: entrustPerm,
		KeyName:     keyName,
		LeverPerm:   leverPerm,
		MoneyPerm:   moneyPerm,
		No:          z.WebsocketConn.GenerateMessageID(true),
		ToUserID:    toUserID,
	}
	request.Channel = "createSubUserKey"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsRequestResponse{}, err
	}
	var response WsRequestResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsSubmitOrder(pair currency.Pair, amount, price float64, tradeType int64) (*WsSubmitOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsSubmitOrderResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsSubmitOrderRequest{
		Amount:    amount,
		Price:     price,
		TradeType: tradeType,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_order", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsSubmitOrderResponse{}, err
	}
	var response WsSubmitOrderResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsCancelOrder(pair currency.Pair, orderID int64) (*WsCancelOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsCancelOrderResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsCancelOrderRequest{
		ID: orderID,
		No: z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_cancelorder", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsCancelOrderResponse{}, err
	}
	var response WsCancelOrderResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsGetOrder(pair currency.Pair, orderID int64) (*WsGetOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetOrderResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrderRequest{
		ID: orderID,
		No: z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getorder", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetOrderResponse{}, err
	}
	var response WsGetOrderResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsGetOrders(pair currency.Pair, pageIndex, tradeType int64) (*WsGetOrdersResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetOrdersResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrdersRequest{
		PageIndex: pageIndex,
		TradeType: tradeType,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getorders", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)
	log.Debug(request)
	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetOrdersResponse{}, err
	}
	var response WsGetOrdersResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsGetOrdersIgnoreTradeType(pair currency.Pair, pageIndex, pageSize int64) (*WsGetOrdersIgnoreTradeTypeResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetOrdersIgnoreTradeTypeResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrdersIgnoreTradeTypeRequest{
		PageIndex: pageIndex,
		PageSize:  pageSize,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getordersignoretradetype", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.APIKey
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetOrdersIgnoreTradeTypeResponse{}, err
	}
	var response WsGetOrdersIgnoreTradeTypeResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}

func (z *ZB) wsGetAccountInfoRequest() (*WsGetAccountInfoResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return &WsGetAccountInfoResponse{}, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAuthenticatedRequest{
		Channel:   "getaccountinfo",
		Event:     zWebsocketAddChannel,
		Accesskey: z.APIKey,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return &WsGetAccountInfoResponse{}, err
	}
	var response WsGetAccountInfoResponse
	err = common.JSONDecode(resp, &response)
	return &response, err
}
