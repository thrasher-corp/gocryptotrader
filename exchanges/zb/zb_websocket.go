package zb

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	zbWebsocketAPI       = "wss://api.zb.cn:9999/websocket"
	zWebsocketAddChannel = "addChannel"
	zbWebsocketRateLimit = 20
)

// WsConnect initiates a websocket connection
func (z *ZB) WsConnect() error {
	if !z.Websocket.IsEnabled() || !z.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
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
			resp, err := z.WebsocketConn.ReadMessage()
			if err != nil {
				z.Websocket.ReadMessageErrors <- err
				return
			}
			z.Websocket.TrafficAlert <- struct{}{}
			fixedJSON := z.wsFixInvalidJSON(resp.Raw)
			var result Generic
			err = common.JSONDecode(fixedJSON, &result)
			if err != nil {
				z.Websocket.DataHandler <- err
				continue
			}
			if result.No > 0 {
				z.WebsocketConn.AddResponseWithID(result.No, fixedJSON)
				continue
			}
			if result.Code > 0 && result.Code != 1000 {
				z.Websocket.DataHandler <- fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, result.Message, wsErrCodes[result.Code])
				continue
			}
			switch {
			case strings.Contains(result.Channel, "markets"):
				var markets Markets
				err := common.JSONDecode(result.Data, &markets)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

			case strings.Contains(result.Channel, "ticker"):
				cPair := strings.Split(result.Channel, "_")
				var ticker WsTicker
				err := common.JSONDecode(fixedJSON, &ticker)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				z.Websocket.DataHandler <- wshandler.TickerData{
					Exchange:  z.Name,
					Close:     ticker.Data.Last,
					Volume:    ticker.Data.Volume24Hr,
					High:      ticker.Data.High,
					Low:       ticker.Data.Low,
					Last:      ticker.Data.Last,
					Bid:       ticker.Data.Buy,
					Ask:       ticker.Data.Sell,
					Timestamp: time.Unix(0, ticker.Date),
					AssetType: asset.Spot,
					Pair:      currency.NewPairFromString(cPair[0]),
				}

			case strings.Contains(result.Channel, "depth"):
				var depth WsDepth
				err := common.JSONDecode(fixedJSON, &depth)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				var asks []orderbook.Item
				for i := range depth.Asks {
					ask := depth.Asks[i].([]interface{})
					asks = append(asks, orderbook.Item{
						Amount: ask[1].(float64),
						Price:  ask[0].(float64),
					})
				}

				var bids []orderbook.Item
				for i := range depth.Bids {
					bid := depth.Bids[i].([]interface{})
					bids = append(bids, orderbook.Item{
						Amount: bid[1].(float64),
						Price:  bid[0].(float64),
					})
				}

				channelInfo := strings.Split(result.Channel, "_")
				cPair := currency.NewPairFromString(channelInfo[0])
				var newOrderBook orderbook.Base
				newOrderBook.Asks = asks
				newOrderBook.Bids = bids
				newOrderBook.AssetType = asset.Spot
				newOrderBook.Pair = cPair

				err = z.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
					true)
				if err != nil {
					z.Websocket.DataHandler <- err
					continue
				}

				z.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
					Pair:     cPair,
					Asset:    asset.Spot,
					Exchange: z.GetName(),
				}

			case strings.Contains(result.Channel, "trades"):
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

				channelInfo := strings.Split(result.Channel, "_")
				cPair := currency.NewPairFromString(channelInfo[0])
				z.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    time.Unix(0, t.Date),
					CurrencyPair: cPair,
					AssetType:    asset.Spot,
					Exchange:     z.GetName(),
					EventTime:    t.Date,
					Price:        t.Price,
					Amount:       t.Amount,
					Side:         t.TradeType,
				}
			default:
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
	enabledCurrencies := z.GetEnabledPairs(asset.Spot)
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
		log.Error(log.ExchangeSys, err)
		return ""
	}
	hmac := crypto.GetHMAC(crypto.HashMD5,
		jsonResponse,
		[]byte(crypto.Sha1ToHex(z.API.Credentials.Secret)))
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
	capturedInvalidZBJSON := strings.Replace(string(matchingResults), "\"", "", 1)
	// Remove last quote character
	fixedJSON := capturedInvalidZBJSON[:len(capturedInvalidZBJSON)-1]
	return []byte(strings.Replace(string(json), string(matchingResults), fixedJSON, 1))
}

func (z *ZB) wsAddSubUser(username, password string) (*WsGetSubUserListResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAddSubUserRequest{
		Memo:        "memo",
		Password:    password,
		SubUserName: username,
	}
	request.Channel = "addSubUser"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.No = z.WebsocketConn.GenerateMessageID(true)
	request.Sign = z.wsGenerateSignature(request)
	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var genericResponse Generic
	err = common.JSONDecode(resp, &genericResponse)
	if err != nil {
		return nil, err
	}
	if genericResponse.Code > 0 && genericResponse.Code != 1000 {
		return nil, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, genericResponse.Message, wsErrCodes[genericResponse.Code])
	}
	var response WsGetSubUserListResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (z *ZB) wsGetSubUserList() (*WsGetSubUserListResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAuthenticatedRequest{}
	request.Channel = "getSubUserList"
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.No = z.WebsocketConn.GenerateMessageID(true)
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsGetSubUserListResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsDoTransferFunds(pair currency.Code, amount float64, fromUserName, toUserName string) (*WsRequestResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
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
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsRequestResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsCreateSubUserKey(assetPerm, entrustPerm, leverPerm, moneyPerm bool, keyName, toUserID string) (*WsRequestResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
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
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsRequestResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsSubmitOrder(pair currency.Pair, amount, price float64, tradeType int64) (*WsSubmitOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsSubmitOrderRequest{
		Amount:    amount,
		Price:     price,
		TradeType: tradeType,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_order", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsSubmitOrderResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsCancelOrder(pair currency.Pair, orderID int64) (*WsCancelOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsCancelOrderRequest{
		ID: orderID,
		No: z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_cancelorder", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsCancelOrderResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsGetOrder(pair currency.Pair, orderID int64) (*WsGetOrderResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrderRequest{
		ID: orderID,
		No: z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getorder", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsGetOrderResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsGetOrders(pair currency.Pair, pageIndex, tradeType int64) (*WsGetOrdersResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrdersRequest{
		PageIndex: pageIndex,
		TradeType: tradeType,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getorders", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)
	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsGetOrdersResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsGetOrdersIgnoreTradeType(pair currency.Pair, pageIndex, pageSize int64) (*WsGetOrdersIgnoreTradeTypeResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsGetOrdersIgnoreTradeTypeRequest{
		PageIndex: pageIndex,
		PageSize:  pageSize,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Channel = fmt.Sprintf("%v_getordersignoretradetype", pair.String())
	request.Event = zWebsocketAddChannel
	request.Accesskey = z.API.Credentials.Key
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsGetOrdersIgnoreTradeTypeResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}

func (z *ZB) wsGetAccountInfoRequest() (*WsGetAccountInfoResponse, error) {
	if !z.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", z.Name)
	}
	request := WsAuthenticatedRequest{
		Channel:   "getaccountinfo",
		Event:     zWebsocketAddChannel,
		Accesskey: z.API.Credentials.Key,
		No:        z.WebsocketConn.GenerateMessageID(true),
	}
	request.Sign = z.wsGenerateSignature(request)

	resp, err := z.WebsocketConn.SendMessageReturnResponse(request.No, request)
	if err != nil {
		return nil, err
	}
	var response WsGetAccountInfoResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return nil, err
	}
	if response.Code > 0 && response.Code != 1000 {
		return &response, fmt.Errorf("%v request failed, message: %v, error code: %v", z.Name, response.Message, wsErrCodes[response.Code])
	}
	return &response, nil
}
