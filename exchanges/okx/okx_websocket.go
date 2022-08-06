package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// OkxOrderbookMutex Ensures if two entries arrive at once, only one can be
// processed at a time
var OkxOrderbookMutex sync.Mutex

// responseStream a channel throught which the data coming from the two websocket connection will go through.
var responseStream = make(chan stream.Response)

// defaultSubscribedChannels list of chanels which are subscribed by default
var defaultSubscribedChannels = []string{
	OkxChannelTrades,
	OkxChannelOrderBooks,
	OkxChannelOrderBooks5,
	OkxChannelOrderBooks50TBT,
	OkxChannelOrderBooksTBT,
	OkxChannelCandle5m,
	OkxChannelTickers,
}

const (
	OkxOrderBookFull   = "snapshot"
	OkxOrderBookUpdate = "update"

	// To be used in validating checksum
	ColonDelimiter = ":"

	// maxConnByteLen otal length of multiple channels cannot exceed 4096 bytes.
	maxConnByteLen = 4096

	// Candlestick channels
	markPrice        = "mark-price-"
	indexCandlestick = "index-"
	candle           = "candle"

	// Candlesticks
	OkxChannelCandle1Y     = candle + "1Y"
	OkxChannelCandle6M     = candle + "6M"
	OkxChannelCandle3M     = candle + "3M"
	OkxChannelCandle1M     = candle + "1M"
	OkxChannelCandle1W     = candle + "1W"
	OkxChannelCandle1D     = candle + "1D"
	OkxChannelCandle2D     = candle + "2D"
	OkxChannelCandle3D     = candle + "3D"
	OkxChannelCandle5D     = candle + "5D"
	OkxChannelCandle12H    = candle + "12H"
	OkxChannelCandle6H     = candle + "6H"
	OkxChannelCandle4H     = candle + "4H"
	OkxChannelCandle2H     = candle + "2H"
	OkxChannelCandle1H     = candle + "1H"
	OkxChannelCandle30m    = candle + "30m"
	OkxChannelCandle15m    = candle + "15m"
	OkxChannelCandle5m     = candle + "5m"
	OkxChannelCandle3m     = candle + "3m"
	OkxChannelCandle1m     = candle + "1m"
	OkxChannelCandle1Yutc  = candle + "1Yutc"
	OkxChannelCandle3Mutc  = candle + "3Mutc"
	OkxChannelCandle1Mutc  = candle + "1Mutc"
	OkxChannelCandle1Wutc  = candle + "1Wutc"
	OkxChannelCandle1Dutc  = candle + "1Dutc"
	OkxChannelCandle2Dutc  = candle + "2Dutc"
	OkxChannelCandle3Dutc  = candle + "3Dutc"
	OkxChannelCandle5Dutc  = candle + "5Dutc"
	OkxChannelCandle12Hutc = candle + "12Hutc"
	OkxChannelCandle6Hutc  = candle + "6Hutc"

	// Ticker channel
	OkxChannelTickers                = "tickers"
	OkxChannelIndexTickers           = "index-tickers"
	OkxChannelStatus                 = "status"
	OkxChannelPublicStrucBlockTrades = "public-struc-block-trades"
	OkxChannelBlockTickers           = "block-tickers"

	// Private Channels
	OkxChannelAccount            = "account"
	OkxChannelPositions          = "positions"
	OkxChannelBalanceAndPosition = "balance_and_position"
	OkxChannelOrders             = "orders"
	OkxChannelAlgoOrders         = "orders-algo"
	OkxChannelAlgoAdvanced       = "algo-advance"
	OkxChannelLiquidationWarning = "liquidation-warning"
	OkxChannelAccountGreeks      = "account-greeks"
	OkxChannelRFQs               = "rfqs"
	OkxChannelQuotes             = "quotes"
	OkxChannelStruckeBlockTrades = "struc-block-trades"
	OkxChannelSpotGridOrder      = "grid-orders-spot"
	OkxChannelGridOrdersConstuct = "grid-orders-contract"
	OkxChannelGridPositions      = "grid-positions"
	OkcChannelGridSubOrders      = "grid-sub-orders"
	OkxChannelInstruments        = "instruments"
	OkxChannelOpenInterest       = "open-interest"
	OkxChannelTrades             = "trades"

	OkxChannelEstimatedPrice  = "estimated-price"
	OkxChannelMarkPrice       = "mark-price"
	OkxChannelPriceLimit      = "price-limit"
	OkxChannelOrderBooks      = "books"
	OkxChannelOrderBooks5     = "books5"
	OkxChannelOrderBooks50TBT = "books50-l2-tbt"
	OkxChannelOrderBooksTBT   = "books-l2-tbt"
	OkxChannelBBO_TBT         = "bbo-tbt"
	OkxChannelOptSummary      = "opt-summary"
	OkxChannelFundingRate     = "funding-rate"

	// Index Candlesticks Channels
	OkxChannelIndexCandle1Y     = indexCandlestick + OkxChannelCandle1Y
	OkxChannelIndexCandle6M     = indexCandlestick + OkxChannelCandle6M
	OkxChannelIndexCandle3M     = indexCandlestick + OkxChannelCandle3M
	OkxChannelIndexCandle1M     = indexCandlestick + OkxChannelCandle1M
	OkxChannelIndexCandle1W     = indexCandlestick + OkxChannelCandle1W
	OkxChannelIndexCandle1D     = indexCandlestick + OkxChannelCandle1D
	OkxChannelIndexCandle2D     = indexCandlestick + OkxChannelCandle2D
	OkxChannelIndexCandle3D     = indexCandlestick + OkxChannelCandle3D
	OkxChannelIndexCandle5D     = indexCandlestick + OkxChannelCandle5D
	OkxChannelIndexCandle12H    = indexCandlestick + OkxChannelCandle12H
	OkxChannelIndexCandle6H     = indexCandlestick + OkxChannelCandle6H
	OkxChannelIndexCandle4H     = indexCandlestick + OkxChannelCandle4H
	OkxChannelIndexCandle2H     = indexCandlestick + OkxChannelCandle2H
	OkxChannelIndexCandle1H     = indexCandlestick + OkxChannelCandle1H
	OkxChannelIndexCandle30m    = indexCandlestick + OkxChannelCandle30m
	OkxChannelIndexCandle15m    = indexCandlestick + OkxChannelCandle15m
	OkxChannelIndexCandle5m     = indexCandlestick + OkxChannelCandle5m
	OkxChannelIndexCandle3m     = indexCandlestick + OkxChannelCandle3m
	OkxChannelIndexCandle1m     = indexCandlestick + OkxChannelCandle1m
	OkxChannelIndexCandle1Yutc  = indexCandlestick + OkxChannelCandle1Yutc
	OkxChannelIndexCandle3Mutc  = indexCandlestick + OkxChannelCandle3Mutc
	OkxChannelIndexCandle1Mutc  = indexCandlestick + OkxChannelCandle1Mutc
	OkxChannelIndexCandle1Wutc  = indexCandlestick + OkxChannelCandle1Wutc
	OkxChannelIndexCandle1Dutc  = indexCandlestick + OkxChannelCandle1Dutc
	OkxChannelIndexCandle2Dutc  = indexCandlestick + OkxChannelCandle2Dutc
	OkxChannelIndexCandle3Dutc  = indexCandlestick + OkxChannelCandle3Dutc
	OkxChannelIndexCandle5Dutc  = indexCandlestick + OkxChannelCandle5Dutc
	OkxChannelIndexCandle12Hutc = indexCandlestick + OkxChannelCandle12Hutc
	OkxChannelIndexCandle6Hutc  = indexCandlestick + OkxChannelCandle6Hutc

	// Mark price candlesticks channel
	OkxChannelMarkPriceCandle1Y     = markPrice + OkxChannelCandle1Y
	OkxChannelMarkPriceCandle6M     = markPrice + OkxChannelCandle6M
	OkxChannelMarkPriceCandle3M     = markPrice + OkxChannelCandle3M
	OkxChannelMarkPriceCandle1M     = markPrice + OkxChannelCandle1M
	OkxChannelMarkPriceCandle1W     = markPrice + OkxChannelCandle1W
	OkxChannelMarkPriceCandle1D     = markPrice + OkxChannelCandle1D
	OkxChannelMarkPriceCandle2D     = markPrice + OkxChannelCandle2D
	OkxChannelMarkPriceCandle3D     = markPrice + OkxChannelCandle3D
	OkxChannelMarkPriceCandle5D     = markPrice + OkxChannelCandle5D
	OkxChannelMarkPriceCandle12H    = markPrice + OkxChannelCandle12H
	OkxChannelMarkPriceCandle6H     = markPrice + OkxChannelCandle6H
	OkxChannelMarkPriceCandle4H     = markPrice + OkxChannelCandle4H
	OkxChannelMarkPriceCandle2H     = markPrice + OkxChannelCandle2H
	OkxChannelMarkPriceCandle1H     = markPrice + OkxChannelCandle1H
	OkxChannelMarkPriceCandle30m    = markPrice + OkxChannelCandle30m
	OkxChannelMarkPriceCandle15m    = markPrice + OkxChannelCandle15m
	OkxChannelMarkPriceCandle5m     = markPrice + OkxChannelCandle5m
	OkxChannelMarkPriceCandle3m     = markPrice + OkxChannelCandle3m
	OkxChannelMarkPriceCandle1m     = markPrice + OkxChannelCandle1m
	OkxChannelMarkPriceCandle1Yutc  = markPrice + OkxChannelCandle1Yutc
	OkxChannelMarkPriceCandle3Mutc  = markPrice + OkxChannelCandle3Mutc
	OkxChannelMarkPriceCandle1Mutc  = markPrice + OkxChannelCandle1Mutc
	OkxChannelMarkPriceCandle1Wutc  = markPrice + OkxChannelCandle1Wutc
	OkxChannelMarkPriceCandle1Dutc  = markPrice + OkxChannelCandle1Dutc
	OkxChannelMarkPriceCandle2Dutc  = markPrice + OkxChannelCandle2Dutc
	OkxChannelMarkPriceCandle3Dutc  = markPrice + OkxChannelCandle3Dutc
	OkxChannelMarkPriceCandle5Dutc  = markPrice + OkxChannelCandle5Dutc
	OkxChannelMarkPriceCandle12Hutc = markPrice + OkxChannelCandle12Hutc
	OkxChannelMarkPriceCandle6Hutc  = markPrice + OkxChannelCandle6Hutc
)

// WsConnect initiates a websocket connection
func (ok *Okx) WsConnect() error {
	if !ok.Websocket.IsEnabled() || !ok.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	var authDialer websocket.Dialer
	authDialer.ReadBufferSize = 8192
	authDialer.WriteBufferSize = 8192
	err := ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ok.Websocket.GetWebsocketURL())
	}
	ok.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 5,
	})

	ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	if ok.IsWebsocketAuthenticationSupported() {
		go func() {
			er := ok.WsAuth(context.Background(), &authDialer)
			if er != nil {
				return
			}
			ok.Websocket.Wg.Add(1)
			go ok.wsFunnelConnectionData(ok.Websocket.AuthConn)
		}()
	}
	ok.Websocket.Wg.Add(2)
	go ok.wsFunnelConnectionData(ok.Websocket.Conn)
	go ok.WsReadData()
	return nil
}

// WsAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ok *Okx) WsAuth(ctx context.Context, dialer *websocket.Dialer) error {
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ok.Name)
	}
	var creds *account.Credentials
	err := ok.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", ok.Name, okxAPIWebsocketPrivateURL, err)
	}
	ok.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 5,
	})
	creds, err = ok.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timeUnix := time.Now()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(timeUnix.UTC().Unix(), 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64Sign := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: "login",
		Arguments: []WebsocketLoginData{
			{
				APIKey:     creds.Key,
				Passphrase: creds.ClientID,
				Timestamp:  timeUnix,
				Sign:       base64Sign,
			},
		},
	}
	go ok.Websocket.AuthConn.SendMessageReturnResponse("login", request)
	return nil
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (ok *Okx) wsFunnelConnectionData(ws stream.Connection) {
	defer ok.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- stream.Response{Raw: resp.Raw}
	}
}

// Subscribe sends a websocket subscription request to several channels to receive data.
func (ok *Okx) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	return ok.handleSubscription("subscribe", channelsToSubscribe)
}

// Unsubscribe sends a websocket unsubscription request to several channels to receive data.
func (ok *Okx) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return ok.handleSubscription("unsubscribe", channelsToUnsubscribe)
}

// handleSubscription sends a subscription and unsubscription information throught the websocket endpoint.
// as of the okex, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (ok *Okx) handleSubscription(operation string, subscriptions []stream.ChannelSubscription) error {
	request := WSSubscriptionInformations{
		Operation: operation,
		Arguments: []SubscriptionInfo{},
	}

	authRequests := WSSubscriptionInformations{
		Operation: operation,
		Arguments: []SubscriptionInfo{},
	}

	var channels []stream.ChannelSubscription
	var authChannels []stream.ChannelSubscription
	var er error
	for i := 0; i < len(subscriptions); i++ {
		arg := SubscriptionInfo{
			Channel: subscriptions[i].Channel,
		}
		var instrumentID string
		var underlying string
		var okay bool
		var instrumentType string
		var authSubscription bool
		var algoID string
		var uid string

		if strings.EqualFold(arg.Channel, OkxChannelAccount) ||
			strings.EqualFold(arg.Channel, OkxChannelOrders) {
			authSubscription = true
		}
		if strings.EqualFold(arg.Channel, "grid-positions") {
			algoID, _ = subscriptions[i].Params["algoId"].(string)
		}

		if strings.EqualFold(arg.Channel, "grid-sub-orders") || strings.EqualFold(arg.Channel, "grid-positions") {
			uid, _ = subscriptions[i].Params["uid"].(string)
		}

		if strings.HasPrefix(arg.Channel, "candle") ||
			strings.EqualFold(arg.Channel, OkxChannelTickers) ||
			strings.EqualFold(arg.Channel, OkxChannelOrderBooks) ||
			strings.EqualFold(arg.Channel, OkxChannelOrderBooks5) ||
			strings.EqualFold(arg.Channel, OkxChannelOrderBooks50TBT) ||
			strings.EqualFold(arg.Channel, OkxChannelOrderBooksTBT) ||
			strings.EqualFold(arg.Channel, OkxChannelTrades) {
			if subscriptions[i].Params["instId"] != "" {
				instrumentID, okay = subscriptions[i].Params["instId"].(string)
				if !okay {
					instrumentID = ""
				}
			} else if subscriptions[i].Params["instrumentID"] != "" {
				instrumentID, okay = subscriptions[i].Params["instrumentID"].(string)
				if !okay {
					instrumentID = ""
				}
			}
			if instrumentID == "" {
				instrumentID, er = ok.GetInstrumentIDFromPair(subscriptions[i].Currency, subscriptions[i].Asset)
				if er != nil {
					instrumentID = ""
				}
			}
		}
		if strings.EqualFold(arg.Channel, "instruments") ||
			strings.EqualFold(arg.Channel, "positions") ||
			strings.EqualFold(arg.Channel, "orders") ||
			strings.EqualFold(arg.Channel, "orders-algo") ||
			strings.EqualFold(arg.Channel, "algo-advance") ||
			strings.EqualFold(arg.Channel, "liquidation-warning") ||
			strings.EqualFold(arg.Channel, "grid-orders-spot") ||
			strings.EqualFold(arg.Channel, "grid-orders-spot") ||
			strings.EqualFold(arg.Channel, "grid-orders-contract") ||
			strings.EqualFold(arg.Channel, "estimated-price") {
			instrumentType = ok.GetInstrumentTypeFromAssetItem(subscriptions[i].Asset)
		}

		if strings.EqualFold(arg.Channel, "positions") || strings.EqualFold(arg.Channel, "orders") || strings.EqualFold(arg.Channel, "orders-algo") || strings.EqualFold(arg.Channel, "estimated-price") || strings.EqualFold(arg.Channel, "opt-summary") {
			underlying, _ = ok.GetUnderlying(subscriptions[i].Currency, subscriptions[i].Asset)
		}

		// if (!subscriptions[i].Currency.IsEmpty()) && subscriptions[i].Asset.IsValid() {
		// 	underlying, er = ok.GetUnderlying(subscriptions[i].Currency, subscriptions[i].Asset)
		// 	if er != nil {
		// 		underlying = ""
		// 	}
		// }
		arg.InstrumentID = instrumentID
		arg.Underlying = underlying
		arg.InstrumentType = instrumentType
		arg.UID = uid
		arg.AlgoID = algoID

		if authSubscription {
			authChannels = append(authChannels, subscriptions[i])
			authRequests.Arguments = append(authRequests.Arguments, arg)
			authChunk, er := json.Marshal(authRequests)
			if er != nil {
				return er
			}
			if len(authChunk) > maxConnByteLen {
				authRequests.Arguments = authRequests.Arguments[:len(authRequests.Arguments)-1]
				i--
				er = ok.Websocket.AuthConn.SendJSONMessage(authRequests)
				if er != nil {
					return er
				}
				if operation == "unsubscribe" {
					ok.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
				} else {
					ok.Websocket.AddSuccessfulSubscriptions(channels...)
				}
				authChannels = []stream.ChannelSubscription{}
				authRequests.Arguments = []SubscriptionInfo{}
			}
		} else {
			channels = append(channels, subscriptions[i])
			request.Arguments = append(request.Arguments, arg)
			chunk, er := json.Marshal(request)
			if er != nil {
				return er
			}
			if len(chunk) > maxConnByteLen {
				i--
				er = ok.Websocket.Conn.SendJSONMessage(request)
				if er != nil {
					return er
				}
				if operation == "unsubscribe" {
					ok.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
				} else {
					ok.Websocket.AddSuccessfulSubscriptions(channels...)
				}
				channels = []stream.ChannelSubscription{}
				request.Arguments = []SubscriptionInfo{}
				continue
			}
		}
	}
	if len(request.Arguments) > 0 {
		val, _ := json.Marshal(request)
		log.Debugf(log.ExchangeSys, "Subscription %s", string(val))
		er = ok.Websocket.Conn.SendJSONMessage(request)
		if er != nil {
			return er
		}
	}
	log.Debugf(log.ExchangeSys, "Can Use Authenticated Websocket Endpoints: %v", ok.Websocket.CanUseAuthenticatedEndpoints())
	if len(authRequests.Arguments) > 0 && ok.Websocket.CanUseAuthenticatedEndpoints() {
		val, _ := json.Marshal(authRequests)
		log.Debugf(log.ExchangeSys, "Auth Subscription %s", string(val))
		er = ok.Websocket.AuthConn.SendJSONMessage(authRequests)
		if er != nil {
			return er
		}
	}
	if er != nil {
		return er
	}

	if operation == "unsubscribe" {
		channels = append(channels, authChannels...)
		ok.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
	} else {
		channels = append(channels, authChannels...)
		ok.Websocket.AddSuccessfulSubscriptions(channels...)
	}
	return nil
}

// WsReadData read coming messages throught the websocket connection and process the data.
func (ok *Okx) WsReadData() {
	defer ok.Websocket.Wg.Done()
	for {
		select {
		case <-ok.Websocket.ShutdownC:
			select {
			case resp := <-responseStream:
				err := ok.WsHandleData(resp.Raw)
				if err != nil {
					select {
					case ok.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", ok.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseStream:
			err := ok.WsHandleData(resp.Raw)
			if err != nil {
				ok.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (ok *Okx) WsHandleData(respRaw []byte) error {
	var dataResponse WebsocketDataResponse
	err := json.Unmarshal(respRaw, &dataResponse)
	if err != nil {
		var resp WSLoginResponse
		er := json.Unmarshal(respRaw, &resp)
		if er == nil && (strings.EqualFold(resp.Event, "login") && resp.Code == 0) {
			ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
		} else if er == nil && (strings.EqualFold(resp.Event, "error") || resp.Code == 60006 || resp.Code == 60007 || resp.Code == 60009 || resp.Code == 60026) {
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		return er
	}
	if len(dataResponse.Data) > 0 {
		switch strings.ToLower(dataResponse.Argument.Channel) {
		case OkxChannelCandle1Y, OkxChannelCandle6M, OkxChannelCandle3M, OkxChannelCandle1M, OkxChannelCandle1W,
			OkxChannelCandle1D, OkxChannelCandle2D, OkxChannelCandle3D, OkxChannelCandle5D, OkxChannelCandle12H,
			OkxChannelCandle6H, OkxChannelCandle4H, OkxChannelCandle2H, OkxChannelCandle1H, OkxChannelCandle30m,
			OkxChannelCandle15m, OkxChannelCandle5m, OkxChannelCandle3m, OkxChannelCandle1m, OkxChannelCandle1Yutc,
			OkxChannelCandle3Mutc, OkxChannelCandle1Mutc, OkxChannelCandle1Wutc, OkxChannelCandle1Dutc,
			OkxChannelCandle2Dutc, OkxChannelCandle3Dutc, OkxChannelCandle5Dutc, OkxChannelCandle12Hutc,
			OkxChannelCandle6Hutc:
			return ok.wsProcessCandles(dataResponse)
		case OkxChannelIndexCandle1Y, OkxChannelIndexCandle6M, OkxChannelIndexCandle3M, OkxChannelIndexCandle1M,
			OkxChannelIndexCandle1W, OkxChannelIndexCandle1D, OkxChannelIndexCandle2D, OkxChannelIndexCandle3D,
			OkxChannelIndexCandle5D, OkxChannelIndexCandle12H, OkxChannelIndexCandle6H, OkxChannelIndexCandle4H,
			OkxChannelIndexCandle2H, OkxChannelIndexCandle1H, OkxChannelIndexCandle30m, OkxChannelIndexCandle15m,
			OkxChannelIndexCandle5m, OkxChannelIndexCandle3m, OkxChannelIndexCandle1m, OkxChannelIndexCandle1Yutc,
			OkxChannelIndexCandle3Mutc, OkxChannelIndexCandle1Mutc, OkxChannelIndexCandle1Wutc,
			OkxChannelIndexCandle1Dutc, OkxChannelIndexCandle2Dutc, OkxChannelIndexCandle3Dutc, OkxChannelIndexCandle5Dutc,
			OkxChannelIndexCandle12Hutc, OkxChannelIndexCandle6Hutc:
			return ok.wsProcessIndexCandles(dataResponse)
		case OkxChannelTickers:
			return ok.wsProcessTickers(respRaw)
		case OkxChannelIndexTickers:
			var response WsIndexTicker
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelStatus:
			var response WsSystemStatusResponse
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelPublicStrucBlockTrades:
			var response WsPublicTradesResponse
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelBlockTickers:
			var response WsBlockTicker
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelAccountGreeks:
			var response WsGreeks
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelAccount:
			var response WsAccountChannelPushData
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelPositions,
			OkxChannelLiquidationWarning:
			var response WsPositionResponse
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelBalanceAndPosition:
			var response WsBalanceAndPosition
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelOrders:
			return ok.wsProcessOrders(respRaw)
		case OkxChannelAlgoOrders:
			var response WsAlgoOrder
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelAlgoAdvanced:
			var response WsAdvancedAlgoOrder
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelRFQs:
			var response WsRFQ
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelQuotes:
			var response WsQuote
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelStruckeBlockTrades:
			var response WsStructureBlocTrade
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelSpotGridOrder:
			var response WsSpotGridAlgoOrder
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelGridOrdersConstuct:
			var response WsContractGridAlgoOrder
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelGridPositions:
			var response WsContractGridAlgoOrder
			return ok.wsProcessPushData(respRaw, &response)
		case OkcChannelGridSubOrders:
			var response WsGridSubOrderData
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelInstruments:
			var response WSInstrumentResponse
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelOpenInterest:
			var response WSOpenInterestResponse
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelTrades:
			return ok.wsProcessTrades(respRaw)
		case OkxChannelEstimatedPrice:
			var response WsDeliveryEstimatedPrice
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelMarkPrice,
			OkxChannelPriceLimit:
			var response WsMarkPrice
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelOrderBooks,
			OkxChannelOrderBooks5,
			OkxChannelOrderBooks50TBT,
			OkxChannelBBO_TBT,
			OkxChannelOrderBooksTBT:
			return ok.wsProcessOrderBooks(respRaw)
		case OkxChannelOptSummary:
			var response WsOptionSummary
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelFundingRate:
			var response WsFundingRate
			return ok.wsProcessPushData(respRaw, &response)
		case OkxChannelMarkPriceCandle1Y, OkxChannelMarkPriceCandle6M, OkxChannelMarkPriceCandle3M, OkxChannelMarkPriceCandle1M,
			OkxChannelMarkPriceCandle1W, OkxChannelMarkPriceCandle1D, OkxChannelMarkPriceCandle2D, OkxChannelMarkPriceCandle3D,
			OkxChannelMarkPriceCandle5D, OkxChannelMarkPriceCandle12H, OkxChannelMarkPriceCandle6H, OkxChannelMarkPriceCandle4H,
			OkxChannelMarkPriceCandle2H, OkxChannelMarkPriceCandle1H, OkxChannelMarkPriceCandle30m, OkxChannelMarkPriceCandle15m,
			OkxChannelMarkPriceCandle5m, OkxChannelMarkPriceCandle3m, OkxChannelMarkPriceCandle1m, OkxChannelMarkPriceCandle1Yutc,
			OkxChannelMarkPriceCandle3Mutc, OkxChannelMarkPriceCandle1Mutc, OkxChannelMarkPriceCandle1Wutc, OkxChannelMarkPriceCandle1Dutc,
			OkxChannelMarkPriceCandle2Dutc, OkxChannelMarkPriceCandle3Dutc, OkxChannelMarkPriceCandle5Dutc, OkxChannelMarkPriceCandle12Hutc,
			OkxChannelMarkPriceCandle6Hutc:
			return ok.wsHandleMarkPriceCandles(respRaw)
		default:
			ok.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: ok.Name + stream.UnhandledMessage + string(respRaw)}
			return nil
		}
	}
	return nil
}

// wsProcessIndexCandles processes index candlestic data
func (ok *Okx) wsProcessIndexCandles(intermediate WebsocketDataResponse) error {
	var response WSCandlestickResponse
	if len(intermediate.Data) == 0 {
		return errNoCandlestickDataFound
	}
	pair, er := ok.GetPairFromInstrumentID(intermediate.Argument.InstrumentID)
	if er != nil {
		return er
	}
	var a asset.Item
	a, _ = ok.GetAssetTypeFromInstrumentType(intermediate.Argument.InstrumentType)
	candleInterval := strings.TrimPrefix(intermediate.Argument.Channel, candle)
	for i := range response.Data {
		candles, okay := (intermediate.Data[i]).([5]string)
		if !okay {
			continue
		}
		timestamp, er := strconv.Atoi(candles[0])
		if er != nil {
			return er
		}
		candle := stream.KlineData{
			Pair:      pair,
			Exchange:  ok.Name,
			Timestamp: time.UnixMilli(int64(timestamp)),
			Interval:  candleInterval,
			AssetType: a,
		}
		candle.OpenPrice, er = strconv.ParseFloat(candles[1], 64)
		if er != nil {
			return er
		}
		candle.HighPrice, er = strconv.ParseFloat(candles[2], 64)
		if er != nil {
			return er
		}
		candle.LowPrice, er = strconv.ParseFloat(candles[3], 64)
		if er != nil {
			return er
		}
		candle.ClosePrice, er = strconv.ParseFloat(candles[4], 64)
		if er != nil {
			return er
		}
		ok.Websocket.DataHandler <- candle
	}
	return nil
}

// wsProcessOrderBooks processes "snapshot" and "update" order book
func (ok *Okx) wsProcessOrderBooks(data []byte) error {
	var response WsOrderBook
	var er error
	if er = json.Unmarshal(data, &response); er != nil {
		return er
	}
	if !(strings.EqualFold(response.Action, OkxOrderBookUpdate) ||
		strings.EqualFold(response.Action, OkxOrderBookFull) ||
		strings.EqualFold(response.Argument.Channel, OkxChannelOrderBooks5) ||
		strings.EqualFold(response.Argument.Channel, OkxChannelBBO_TBT) ||

		strings.EqualFold(response.Argument.Channel, OkxChannelOrderBooks50TBT) ||
		strings.EqualFold(response.Argument.Channel, OkxChannelOrderBooksTBT)) {
		return errors.New("invalid order book action ")
	}
	OkxOrderbookMutex.Lock()
	defer OkxOrderbookMutex.Unlock()
	var pair currency.Pair
	var a asset.Item
	a, _ = ok.GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if a == asset.Empty {
		a = ok.GuessAssetTypeFromInstrumentID(response.Argument.InstrumentID)
	}
	pair, er = ok.GetPairFromInstrumentID(response.Argument.InstrumentID)
	if er != nil {
		pair.Delimiter = currency.DashDelimiter
	}
	for i := range response.Data {
		if strings.EqualFold(response.Action, OkxOrderBookFull) ||
			strings.EqualFold(response.Argument.Channel, OkxChannelOrderBooks5) ||
			strings.EqualFold(response.Argument.Channel, OkxChannelBBO_TBT) {
			er = ok.WsProcessFullOrderBook(response.Data[i], pair, a)
			if er != nil {
				_, err2 := ok.OrderBooksSubscription("subscribe", response.Argument.Channel, a, pair)
				if err2 != nil {
					ok.Websocket.DataHandler <- err2
				}
				return er
			}
		} else {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			er := ok.WsProcessUpdateOrderbook(response.Argument.Channel, response.Data[i], pair, a)
			if er != nil {
				_, err2 := ok.OrderBooksSubscription("subscribe", response.Argument.Channel, a, pair)
				if err2 != nil {
					ok.Websocket.DataHandler <- err2
				}
				return er
			}
		}
	}
	return nil
}

// WsProcessFullOrderBook processes snapshot order books
func (ok *Okx) WsProcessFullOrderBook(data WsOrderBookData, pair currency.Pair, a asset.Item) error {
	var er error
	if data.Checksum != 0 {
		signedChecksum, er := ok.CalculatePartialOrderbookChecksum(data)
		if er != nil {
			return fmt.Errorf("%s channel: Orderbook unable to calculate orderbook checksum: %s", ok.Name, er)
		}
		if signedChecksum != data.Checksum {
			return fmt.Errorf("%s channel: Orderbook for %v checksum invalid",
				ok.Name,
				pair)
		}
	}

	if ok.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s passed checksum for pair %v",
			ok.Name, pair,
		)
	}
	var asks []orderbook.Item
	var bids []orderbook.Item
	asks, er = ok.AppendWsOrderbookItems(data.Asks)
	if er != nil {
		return er
	}
	bids, er = ok.AppendWsOrderbookItems(data.Bids)
	if er != nil {
		return er
	}
	newOrderBook := orderbook.Base{
		Asset:           a,
		Asks:            asks,
		Bids:            bids,
		LastUpdated:     data.Timestamp,
		Pair:            pair,
		Exchange:        ok.Name,
		VerifyOrderbook: ok.CanVerifyOrderbook,
	}
	return ok.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing
// orderbook
func (ok *Okx) WsProcessUpdateOrderbook(channel string, data WsOrderBookData, pair currency.Pair, a asset.Item) error {
	update := &orderbook.Update{
		Asset: a,
		Pair:  pair,
	}
	var err error
	update.Asks, err = ok.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	update.Bids, err = ok.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	switch {
	case strings.EqualFold(channel, OkxChannelOrderBooksTBT):
		update.MaxDepth = 400
	case strings.EqualFold(channel, OkxChannelOrderBooks50TBT):
		update.MaxDepth = 50
	}
	err = ok.Websocket.Orderbook.Update(update)
	if err != nil {
		return err
	}
	updatedOb, err := ok.Websocket.Orderbook.GetOrderbook(pair, a)
	if err != nil {
		return err
	}
	if data.Checksum != 0 {
		checksum := ok.CalculateUpdateOrderbookChecksum(updatedOb)
		if checksum != data.Checksum {
			log.Warnf(log.ExchangeSys, "%s checksum failure for pair %v",
				ok.Name,
				pair)
			return errors.New("checksum failed")
		}
	}
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an
// orderbook item array
func (o *Okx) AppendWsOrderbookItems(entries [][4]string) ([]orderbook.Item, error) {
	items := make([]orderbook.Item, len(entries))
	for j := range entries {
		amount, err := strconv.ParseFloat(entries[j][1], 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(entries[j][0], 64)
		if err != nil {
			return nil, err
		}
		items[j] = orderbook.Item{Amount: amount, Price: price}
	}
	return items, nil
}

// CalculatePartialOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data.
func (ok *Okx) CalculatePartialOrderbookChecksum(orderbookData WsOrderBookData) (int32, error) {
	var checksum strings.Builder
	for i := 0; i < 25; i++ {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice := orderbookData.Bids[i][0]
			bidAmount := orderbookData.Bids[i][1]
			checksum.WriteString(
				bidPrice +
					ColonDelimiter +
					bidAmount +
					ColonDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice := orderbookData.Asks[i][0]
			askAmount := orderbookData.Asks[i][1]
			checksum.WriteString(askPrice +
				ColonDelimiter +
				askAmount +
				ColonDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ColonDelimiter)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr))), nil
}

// wsHandleMarkPriceCandles processes candlestick mark price push data as a result of  subscription to "mark-price-candle*" channel.
func (ok *Okx) wsHandleMarkPriceCandles(data []byte) error {
	tempo := &struct {
		Argument SubscriptionInfo `json:"arg"`
		Data     [][5]string      `json:"data"`
	}{}
	var er error
	if er = json.Unmarshal(data, tempo); er != nil {
		return er
	}
	var candles []CandlestickMarkPrice
	var tsInt int64
	var ts time.Time
	var op float64
	var hp float64
	var lp float64
	var cp float64
	for x := range tempo.Data {
		tsInt, er = strconv.ParseInt(tempo.Data[x][0], 10, 64)
		if er != nil {
			return er
		}
		ts = time.UnixMilli(tsInt)
		op, er = strconv.ParseFloat(tempo.Data[x][1], 64)
		if er != nil {
			return er
		}
		hp, er = strconv.ParseFloat(tempo.Data[x][2], 64)
		if er != nil {
			return er
		}
		lp, er = strconv.ParseFloat(tempo.Data[x][3], 64)
		if er != nil {
			return er
		}
		cp, er = strconv.ParseFloat(tempo.Data[x][4], 64)
		if er != nil {
			return er
		}
		candles = append(candles, CandlestickMarkPrice{
			Timestamp:    ts,
			OpenPrice:    op,
			HighestPrice: hp,
			LowestPrice:  lp,
			ClosePrice:   cp,
		})
	}
	ok.Websocket.DataHandler <- candles
	return nil
}

// wsProcessTrades handles a list of trade information.
func (ok *Okx) wsProcessTrades(data []byte) error {
	var response WsTradeOrder
	var er error
	var assetType asset.Item
	if er := json.Unmarshal(data, &response); er != nil {
		return er
	}
	assetType, _ = ok.GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if assetType == asset.Empty {
		assetType = ok.GuessAssetTypeFromInstrumentID(response.Argument.InstrumentID)
	}
	trades := make([]trade.Data, len(response.Data))
	for i := range response.Data {
		var pair currency.Pair
		pair, er = ok.GetPairFromInstrumentID(response.Data[i].InstrumentID)
		if er != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				Err:      er,
			}
			return er
		}
		amount := response.Data[i].Quantity
		side := order.ParseOrderSideString(response.Data[i].Side)
		trades[i] = trade.Data{
			Amount:       amount,
			AssetType:    assetType,
			CurrencyPair: pair,
			Exchange:     ok.Name,
			Side:         side,
			Timestamp:    response.Data[i].Timestamp,
			TID:          response.Data[i].TradeID,
			Price:        response.Data[i].Price,
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
}

// wsProcessOrders handles websocket order push data responses.
func (ok *Okx) wsProcessOrders(respRaw []byte) error {
	var response WsOrderResponse
	var pair currency.Pair
	var assetType asset.Item
	var er error
	if er = json.Unmarshal(respRaw, &response); er != nil {
		return er
	}
	assetType, er = ok.GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if er != nil {
		return er
	}
	pair, er = ok.GetPairFromInstrumentID(response.Argument.InstrumentID)
	if er != nil {
		return er
	}
	for x := range response.Data {
		var orderType order.Type
		var orderStatus order.Status
		side := response.Data[x].Side
		orderType, er = order.StringToOrderType(response.Data[x].OrderType)
		if er != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      er,
			}
		}
		orderStatus, er = order.StringToOrderStatus(response.Data[x].State)
		if er != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      er,
			}
		}
		var a asset.Item
		a, er = ok.GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
		if er != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      er,
			}
			a = assetType
		}
		pair, er = ok.GetPairFromInstrumentID(response.Data[x].InstrumentID)
		if er != nil {
			return er
		}
		ok.Websocket.DataHandler <- &order.Detail{
			Price:           response.Data[x].Price,
			Amount:          response.Data[x].Size,
			ExecutedAmount:  response.Data[x].LastFilledSize,
			RemainingAmount: response.Data[x].AccumulatedFillSize - response.Data[x].LastFilledSize,
			Exchange:        ok.Name,
			OrderID:         response.Data[x].OrderID,
			Type:            orderType,
			Side:            side,
			Status:          orderStatus,
			AssetType:       a,
			Date:            response.Data[x].CreationTime,
			Pair:            pair,
		}
	}
	return nil
}

// wsProcessCandles handler to get a list of candlestick messages.
func (ok *Okx) wsProcessCandles(intermediate WebsocketDataResponse) error {
	var response WSCandlestickResponse
	if len(intermediate.Data) == 0 {
		return errNoCandlestickDataFound
	}
	pair, er := ok.GetPairFromInstrumentID(intermediate.Argument.InstrumentID)
	if er != nil {
		return er
	}
	var a asset.Item
	a, er = ok.GetAssetTypeFromInstrumentType(intermediate.Argument.InstrumentType)
	if er != nil {

	}
	candleInterval := strings.TrimPrefix(intermediate.Argument.Channel, candle)
	for i := range response.Data {
		candles, okay := (intermediate.Data[i]).([7]string)
		if !okay {
			continue
		}
		timestamp, er := strconv.Atoi(candles[0])
		if er != nil {
			return er
		}
		candle := stream.KlineData{
			Pair:      pair,
			Exchange:  ok.Name,
			Timestamp: time.UnixMilli(int64(timestamp)),
			Interval:  candleInterval,
			AssetType: a,
		}
		candle.OpenPrice, er = strconv.ParseFloat(candles[1], 64)
		if er != nil {
			return er
		}
		candle.HighPrice, er = strconv.ParseFloat(candles[2], 64)
		if er != nil {
			return er
		}
		candle.LowPrice, er = strconv.ParseFloat(candles[3], 64)
		if er != nil {
			return er
		}
		candle.ClosePrice, er = strconv.ParseFloat(candles[4], 64)
		if er != nil {
			return er
		}
		candle.Volume, er = strconv.ParseFloat(candles[5], 64)
		if er != nil {
			return er
		}
		// tradingVolumeWithCurrencyUnit, er := strconv.ParseFloat(candles[6], 64)
		// if er != nil {
		// 	return er
		// }
		// candle.TradingVolumeWithCurrencyUnit = tradingVolumeWithCurrencyUnit
		ok.Websocket.DataHandler <- candle
	}
	return nil
}

// wsProcessTickers handles the trade ticker information.
func (ok *Okx) wsProcessTickers(data []byte) error {
	var response WSTickerResponse
	if er := json.Unmarshal(data, &response); er != nil {
		return er
	}
	for i := range response.Data {
		a := response.Data[i].InstrumentType
		if a == asset.Empty {
			a = ok.GuessAssetTypeFromInstrumentID(response.Data[i].InstrumentID)
		}
		if !(a == asset.Futures || a == asset.PerpetualSwap || a == asset.Margin || a == asset.Option || a == asset.Spot) {
			return errInvalidInstrumentType
		}
		var c currency.Pair
		var er error
		c, er = ok.GetPairFromInstrumentID(response.Data[i].InstrumentID)
		if er != nil {
			return er
		}
		var baseVolume float64
		var quoteVolume float64
		if a == asset.Spot || a == asset.Margin {
			baseVolume = response.Data[i].Vol24H
			quoteVolume = response.Data[i].VolCcy24H
		} else {
			baseVolume = response.Data[i].VolCcy24H
			quoteVolume = response.Data[i].Vol24H
		}
		ok.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: ok.Name,
			Open:         response.Data[i].Open24H,
			Volume:       baseVolume,
			QuoteVolume:  quoteVolume,
			High:         response.Data[i].High24H,
			Low:          response.Data[i].Low24H,
			Bid:          response.Data[i].BidPrice,
			Ask:          response.Data[i].BestAskPrice,
			BidSize:      response.Data[i].BidSize,
			AskSize:      response.Data[i].BestAskSize,
			Last:         response.Data[i].LastTradePrice,
			AssetType:    a,
			Pair:         c,
			LastUpdated:  response.Data[i].TickerDataGenerationTime,
		}
	}
	return nil
}

// GenerateDefaultSubscriptions returns a list of default subscription message.
func (ok *Okx) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	assets := ok.GetAssetTypes(true)
	if ok.Websocket.CanUseAuthenticatedEndpoints() {
		defaultSubscribedChannels = append(defaultSubscribedChannels,
			OkxChannelAccount,
			OkxChannelOrders,
		)
	}
	for x := range assets {
		pairs, err := ok.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}
		for y := range defaultSubscribedChannels {
			if defaultSubscribedChannels[y] == OkxChannelCandle5m ||
				defaultSubscribedChannels[y] == OkxChannelTickers ||
				defaultSubscribedChannels[y] == OkxChannelOrders ||
				defaultSubscribedChannels[y] == OkxChannelOrderBooks ||
				defaultSubscribedChannels[y] == OkxChannelOrderBooks5 ||
				defaultSubscribedChannels[y] == OkxChannelOrderBooks50TBT ||
				defaultSubscribedChannels[y] == OkxChannelOrderBooksTBT ||
				defaultSubscribedChannels[y] == OkxChannelTrades {
				for p := range pairs {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  defaultSubscribedChannels[y],
						Asset:    assets[x],
						Currency: pairs[p],
					})
				}
			} else {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel: defaultSubscribedChannels[y],
				})
			}
		}
	}
	return subscriptions, nil
}

// wsProcessPushData processes push data coming through the websocket channel
func (ok *Okx) wsProcessPushData(data []byte, resp interface{}) error {
	if er := json.Unmarshal(data, resp); er != nil {
		return nil
	}
	ok.Websocket.DataHandler <- resp
	return nil
}

// Websocket Trade methods

// WSPlaceOrder places an order throught the websocket connection stream, and returns a SubmitResponse and error message.
func (ok *Okx) WSPlaceOrder(arg PlaceOrderRequestParam) (*PlaceOrderResponse, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.TradeMode = strings.Trim(arg.TradeMode, " ")
	if !(strings.EqualFold("cross", arg.TradeMode) || strings.EqualFold("isolated", arg.TradeMode) || strings.EqualFold("cash", arg.TradeMode)) {
		return nil, errInvalidTradeModeValue
	}
	if !(strings.EqualFold(arg.Side, "buy") || strings.EqualFold(arg.Side, "sell")) {
		return nil, errMissingOrderSide
	}
	if !(strings.EqualFold(arg.OrderType, "market") || strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc") || strings.EqualFold(arg.OrderType, "optimal_limit_ioc")) {
		return nil, errInvalidOrderType
	}
	if arg.QuantityToBuyOrSell <= 0 {
		return nil, errInvalidQuantityToButOrSell
	}
	if arg.OrderPrice <= 0 && (strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc")) {
		return nil, fmt.Errorf("invalid order price for %s order types", arg.OrderType)
	}
	if !(strings.EqualFold(arg.QuantityType, "base_ccy") || strings.EqualFold(arg.QuantityType, "quote_ccy")) {
		arg.QuantityType = ""
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := WsPlaceOrderInput{
		ID:        randomID,
		Arguments: []PlaceOrderRequestParam{arg},
		Operation: "batch-orders",
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("order", input)
	if er != nil {
		return nil, er
	}
	var placeOrderResponse WSPlaceOrderResponse
	if er = json.Unmarshal(respData, &placeOrderResponse); er != nil {
		return nil, er
	}
	if !(placeOrderResponse.Code == "0" ||
		placeOrderResponse.Code == "2") {
		if placeOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(placeOrderResponse.Msg)
	} else if len(placeOrderResponse.Data) == 0 {
		if placeOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(placeOrderResponse.Msg)
	}
	return &(placeOrderResponse.Data[0]), nil
}

// WsPlaceMultipleOrder
func (ok *Okx) WsPlaceMultipleOrder(args []PlaceOrderRequestParam) ([]PlaceOrderResponse, error) {
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		arg.TradeMode = strings.Trim(arg.TradeMode, " ")
		if !(strings.EqualFold("cross", arg.TradeMode) || strings.EqualFold("isolated", arg.TradeMode) || strings.EqualFold("cash", arg.TradeMode)) {
			return nil, errInvalidTradeModeValue
		}
		if !(strings.EqualFold(arg.Side, "buy") || strings.EqualFold(arg.Side, "sell")) {
			return nil, errMissingOrderSide
		}
		if !(strings.EqualFold(arg.OrderType, "market") || strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
			strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc") || strings.EqualFold(arg.OrderType, "optimal_limit_ioc")) {
			return nil, errInvalidOrderType
		}
		if arg.QuantityToBuyOrSell <= 0 {
			return nil, errInvalidQuantityToButOrSell
		}
		if arg.OrderPrice <= 0 && (strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
			strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc")) {
			return nil, fmt.Errorf("invalid order price for %s order types", arg.OrderType)
		}
		if !(strings.EqualFold(arg.QuantityType, "base_ccy") || strings.EqualFold(arg.QuantityType, "quote_ccy")) {
			arg.QuantityType = ""
		}
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := WsPlaceOrderInput{
		ID:        randomID,
		Arguments: args,
		Operation: "batch-orders",
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("orders", input)
	if er != nil {
		return nil, er
	}
	var placeOrderResponse WSPlaceOrderResponse
	if er = json.Unmarshal(respData, &placeOrderResponse); er != nil {
		return nil, er
	}
	if !(placeOrderResponse.Code == "0" ||
		placeOrderResponse.Code == "2") {
		if placeOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(placeOrderResponse.Msg)
	}
	return placeOrderResponse.Data, nil
}

// WsCancelOrder websocket function to cancel a trade order
func (ok *Okx) WsCancelOrder(arg CancelOrderRequestParam) (*PlaceOrderResponse, error) {
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
		return nil, fmt.Errorf("either order id or client supplier id is required")
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := WsCancelOrderInput{
		ID:        randomID,
		Arguments: []CancelOrderRequestParam{arg},
		Operation: "cancel-order",
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("cancel-orders", input)
	if er != nil {
		return nil, er
	}
	var cancelOrderResponse WSPlaceOrderResponse
	if er = json.Unmarshal(respData, &cancelOrderResponse); er != nil {
		return nil, er
	}
	if !(cancelOrderResponse.Code == "1") || strings.EqualFold(cancelOrderResponse.Code, "60013") {
		if cancelOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(cancelOrderResponse.Msg)
	} else if len(cancelOrderResponse.Data) == 0 {
		if cancelOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(cancelOrderResponse.Msg)
	}
	return &(cancelOrderResponse.Data[0]), nil
}

// WsCancelMultipleOrder cancel multiple order through the websocket channel.
func (ok *Okx) WsCancleMultipleOrder(args []CancelOrderRequestParam) ([]PlaceOrderResponse, error) {
	for x := range args {
		arg := args[x]
		if strings.Trim(arg.InstrumentID, " ") == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
			return nil, fmt.Errorf("either order id or client supplier id is required")
		}
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := WsCancelOrderInput{
		ID:        randomID,
		Arguments: args,
		Operation: "batch-cancel-orders",
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("cancel-orders", input)
	if er != nil {
		return nil, er
	}
	var cancelOrderResponse WSPlaceOrderResponse
	if er = json.Unmarshal(respData, &cancelOrderResponse); er != nil {
		return nil, er
	}
	if !(cancelOrderResponse.Code == "1") || strings.EqualFold(cancelOrderResponse.Code, "60013") {
		if cancelOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(cancelOrderResponse.Msg)
	}
	return cancelOrderResponse.Data, nil
}

// WsAmendOrder method to amend trade order using a request throught the websocket channel.
func (ok *Okx) WsAmendOrder(arg AmendOrderRequestParams) (*AmendOrderResponse, error) {
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientSuppliedOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errMissingNewSizeOrPriceInformation
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := WsAmendOrderInput{
		ID:        randomID,
		Operation: "amend-order",
		Arguments: []AmendOrderRequestParams{arg},
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("amend-order", input)
	if er != nil {
		return nil, er
	}
	var amendOrderResponse WsAmendOrderResponse
	if er = json.Unmarshal(respData, &amendOrderResponse); er != nil {
		return nil, er
	}

	if !strings.EqualFold(amendOrderResponse.Code, "0") ||
		strings.EqualFold(amendOrderResponse.Code, "1") ||
		strings.EqualFold(amendOrderResponse.Code, "60013") {
		if amendOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(amendOrderResponse.Msg)
	} else if len(amendOrderResponse.Data) == 0 {
		if amendOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(amendOrderResponse.Msg)
	}
	return &amendOrderResponse.Data[0], nil
}

// WsAmendMultipleOrders a request through the websocket connection to amend multiple trade orders.
func (ok *Okx) WsAmendMultipleOrders(args []AmendOrderRequestParams) ([]AmendOrderResponse, error) {
	for x := range args {
		if strings.Trim(args[x].InstrumentID, " ") == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientSuppliedOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errMissingNewSizeOrPriceInformation
		}
	}
	randomID := common.GenerateRandomString(4, common.NumberCharacters)
	input := &WsAmendOrderInput{
		ID:        randomID,
		Operation: "batch-amend-orders",
		Arguments: args,
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("amend-orders", input)
	if er != nil {
		return nil, er
	}
	var amendOrderResponse WsAmendOrderResponse
	if er = json.Unmarshal(respData, &amendOrderResponse); er != nil {
		return nil, er
	}
	if !strings.EqualFold(amendOrderResponse.Code, "0") ||
		!strings.EqualFold(amendOrderResponse.Code, "2") ||
		strings.EqualFold(amendOrderResponse.Code, "1") ||
		strings.EqualFold(amendOrderResponse.Code, "60013") {
		if amendOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(amendOrderResponse.Msg)
	} else if len(amendOrderResponse.Data) == 0 {
		if amendOrderResponse.Msg == "" {
			return nil, errNoValidResponseFromServer
		}
		return nil, errors.New(amendOrderResponse.Msg)
	}
	return amendOrderResponse.Data, nil
}

// WsChannelSubscription send a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) WsChannelSubscription(operation, channel string, assetType asset.Item, pair currency.Pair, tooglers ...bool) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(operation, "subscribe") || strings.EqualFold(operation, "unsubscribe")) {
		return nil, errInvalidWebsocketEvent
	}
	var underlying string
	var instrumentID string
	var instrumentType string
	var er error
	if len(tooglers) > 0 && tooglers[0] {
		instrumentType = strings.ToUpper(assetType.String())
		if !(strings.EqualFold(instrumentType, "SPOT") ||
			strings.EqualFold(instrumentType, "MARGIN") ||
			strings.EqualFold(instrumentType, "SWAP") ||
			strings.EqualFold(instrumentType, "FUTURES") ||
			strings.EqualFold(instrumentType, "OPTION")) {
			instrumentType = "ANY"
		}
	}
	if len(tooglers) > 2 && tooglers[2] {
		if !pair.IsEmpty() {
			underlying, _ = ok.GetUnderlying(pair, assetType)
		}
	}
	if len(tooglers) > 1 && tooglers[1] {
		instrumentID, er = ok.GetInstrumentIDFromPair(pair, assetType)
		if er != nil {
			instrumentID = ""
		}
	}
	if channel == "" {
		return nil, errMissingValidChannelInformation
	}
	input := &SubscriptionOperationInput{
		Operation: operation,
		Arguments: []SubscriptionInfo{
			{
				Channel:        channel,
				InstrumentType: instrumentType,
				Underlying:     underlying,
				InstrumentID:   instrumentID,
			},
		},
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse(channel, input)
	if er != nil {
		return nil, er
	}
	var resp SubscriptionOperationResponse
	if er = json.Unmarshal(respData, &resp); er != nil {
		return nil, er
	}
	if strings.EqualFold(resp.Event, "error") || strings.EqualFold(resp.Code, "60012") {
		if len(resp.Msg) == 0 {
			return nil, fmt.Errorf("%s %s error %s", channel, operation, string(respData))
		}
		return nil, errors.New(resp.Msg)
	}
	return &resp, nil
}

// Private Channel Websocket methods

// WsAuthChannelSubscription send a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) WsAuthChannelSubscription(operation, channel string, assetType asset.Item, pair currency.Pair, uid, algoID string, tooglers ...bool) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(operation, "subscribe") || strings.EqualFold(operation, "unsubscribe")) {
		return nil, errInvalidWebsocketEvent
	}
	var underlying string
	var instrumentID string
	var instrumentType string
	var currency string
	var er error
	if len(tooglers) > 0 && tooglers[0] {
		instrumentType = strings.ToUpper(assetType.String())
		if !(strings.EqualFold(instrumentType, "SPOT") ||
			strings.EqualFold(instrumentType, "MARGIN") ||
			strings.EqualFold(instrumentType, "SWAP") ||
			strings.EqualFold(instrumentType, "FUTURES") ||
			strings.EqualFold(instrumentType, "OPTION")) {
			instrumentType = "ANY"
		}
	}
	if len(tooglers) > 2 && tooglers[2] {
		if !pair.IsEmpty() {
			underlying, _ = ok.GetUnderlying(pair, assetType)
		}
	}
	if len(tooglers) > 1 && tooglers[1] {
		instrumentID, er = ok.GetInstrumentIDFromPair(pair, assetType)
		if er != nil {
			instrumentID = ""
		}
	}
	if len(tooglers) > 3 && tooglers[3] {
		if !(pair.IsEmpty()) {
			if !(pair.Base.IsEmpty()) {
				currency = strings.ToUpper(pair.Base.String())
			} else {
				currency = strings.ToUpper(pair.Quote.String())
			}
		}
	}
	if channel == "" {
		return nil, errMissingValidChannelInformation
	}
	input := &SubscriptionOperationInput{
		Operation: operation,
		Arguments: []SubscriptionInfo{
			{
				Channel:        channel,
				InstrumentType: instrumentType,
				Underlying:     underlying,
				InstrumentID:   instrumentID,
				Currency:       currency,
				UID:            uid,
			},
		},
	}
	respData, er := ok.Websocket.AuthConn.SendMessageReturnResponse(channel, input)
	if er != nil {
		return nil, er
	}
	var resp SubscriptionOperationResponse
	if er = json.Unmarshal(respData, &resp); er != nil {
		return nil, er
	}
	if strings.EqualFold(resp.Event, "error") || strings.EqualFold(resp.Code, "60012") {
		if len(resp.Msg) == 0 {
			return nil, fmt.Errorf("%s %s error %s", channel, operation, string(respData))
		}
		return nil, errors.New(resp.Msg)
	}
	return &resp, nil
}

// WsAccountSubscription retrieve account information. Data will be pushed when triggered by
//  events such as placing order, canceling order, transaction execution, etc.
// It will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) WsAccountSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "account", assetType, pair, "", "", false, false, false, true)
}

// WsPositionChannel retrives the position data. The first snapshot will be sent in accordance with the granularity of the subscription. Data will be pushed when certain actions, such placing or canceling an order, trigger it. It will also be pushed periodically based on the granularity of the subscription.
func (ok *Okx) WsPositionChannel(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "positions", assetType, pair, "", "", true)
}

// BalanceAndPositionSubscription retrieve account balance and position information. Data will be pushed when triggered by events such as filled order, funding transfer.
func (ok *Okx) BalanceAndPositionSubscription(operation, uid string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "balance_and_position", asset.Empty, currency.EMPTYPAIR, "", "")
}

// WsOrderChannel for subscribing for orders.
func (ok *Okx) WsOrderChannel(operation string, assetType asset.Item, pair currency.Pair, instrumentID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "orders", assetType, pair, "", "", true, true, true)
}

// AlgoOrdersSubscription for subscribing to algo - order channels
func (ok *Okx) AlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "orders-algo", assetType, pair, "", "", true, true, true)
}

// AdvanceAlgoOrdersSubscription algo order subscription to retrieve advance algo orders (including Iceberg order, TWAP order, Trailing order). Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) AdvanceAlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair, algoID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "algo-advance", assetType, pair, "", algoID, true, true)
}

// PositionRiskWarningSubscription this push channel is only used as a risk warning, and is not recommended as a risk judgment for strategic trading
// In the case that the market is not moving violently, there may be the possibility that the position has been liquidated at the same time that this message is pushed.
func (ok *Okx) PositionRiskWarningSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "liquidation-warning", assetType, pair, "", "", true, true, true)
}

// AccountGreeksSubscription algo order subscription to retrieve account greeks information. Data will be pushed when triggered by events such as increase/decrease positions or cash balance in account, and will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) AccountGreeksSubscription(operation string, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "account-greeks", asset.Empty, pair, "", "", false, false, false, true)
}

// RfqSubscription subscription to retrive Rfq updates on RFQ orders.
func (ok *Okx) RfqSubscription(operation, uid string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "rfqs", asset.Empty, currency.EMPTYPAIR, uid, "")
}

// QuotesSubscription subscription to retrive Quote subscription
func (ok *Okx) QuotesSubscription(operation string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "quotes", asset.Empty, currency.EMPTYPAIR, "", "")
}

// StructureBlockTradesSubscription to retrive Structural block subscription
func (ok *Okx) StructureBlockTradesSubscription(operation string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "struc-block-trades", asset.Empty, currency.EMPTYPAIR, "", "")
}

// SpotGridAlgoOrdersSubscription to retrieve spot grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) SpotGridAlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair, algoID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "grid-orders-spot", assetType, pair, "", algoID, true, true, true)
}

// ContractGridAlgoOrders to retrieve contract grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) ContractGridAlgoOrders(operation string, assetType asset.Item, pair currency.Pair, algoID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "grid-orders-contract", assetType, pair, "", algoID, true, true, true)
}

// GridPositionsSubscription to retrive grid positions. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) GridPositionsSubscription(operation, algoID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "grid-positions", asset.Empty, currency.EMPTYPAIR, "", algoID)
}

// GridSubOrders to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
func (ok *Okx) GridSubOrders(operation, algoID string) (*SubscriptionOperationResponse, error) {
	return ok.WsAuthChannelSubscription(operation, "grid-sub-orders", asset.Empty, currency.EMPTYPAIR, "", algoID)
}

// Public Websocket stream subscription

// InstrumentsSubscription to subscribe for instruments. The full instrument list will be pushed
// for the first time after subscription. Subsequently, the instruments will be pushed if there is any change to the instrument’s state (such as delivery of FUTURES,
// exercise of OPTION, listing of new contracts / trading pairs, trading suspension, etc.).
func (ok *Okx) InstrumentsSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "instruments", assetType, pair, true)
}

// TickersSubscription subscribing to "ticker" channel to retrieve the last traded price, bid price, ask price and 24-hour trading volume of instruments. Data will be pushed every 100 ms.
func (ok *Okx) TickersSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "tickers", assetType, pair, false, true)
}

// OpenInterestSubscription to subscribe or unsubscribe to "open-interest" channel to retrieve the open interest. Data will by pushed every 3 seconds.
func (ok *Okx) OpenInterestSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "open-interest", assetType, pair, false, true)
}

// CandlesticksSubscription to subscribe or unsubscribe to "candle" channels to retrieve the candlesticks data of an instrument. the push frequency is the fastest interval 500ms push the data.
func (ok *Okx) CandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(channel, "candle1Y") || strings.EqualFold(channel, "candle6M") || strings.EqualFold(channel, "candle3M") || strings.EqualFold(channel, "candle1M") || strings.EqualFold(channel, "candle1W") || strings.EqualFold(channel, "candle1D") || strings.EqualFold(channel, "candle2D") || strings.EqualFold(channel, "candle3D") || strings.EqualFold(channel, "candle5D") || strings.EqualFold(channel, "candle12H") || strings.EqualFold(channel, "candle6H") || strings.EqualFold(channel, "candle4H") || strings.EqualFold(channel, "candle2H") || strings.EqualFold(channel, "candle1H") || strings.EqualFold(channel, "candle30m") || strings.EqualFold(channel, "candle15m") || strings.EqualFold(channel, "candle5m") || strings.EqualFold(channel, "candle3m") || strings.EqualFold(channel, "candle1m") || strings.EqualFold(channel, "candle1Yutc") || strings.EqualFold(channel, "candle3Mutc") || strings.EqualFold(channel, "candle1Mutc") || strings.EqualFold(channel, "candle1Wutc") || strings.EqualFold(channel, "candle1Dutc") || strings.EqualFold(channel, "candle2Dutc") || strings.EqualFold(channel, "candle3Dutc") || strings.EqualFold(channel, "candle5Dutc") || strings.EqualFold(channel, "candle12Hutc") || strings.EqualFold(channel, "candle6Hutc")) {
		return nil, errMissingValidChannelInformation
	}
	return ok.WsChannelSubscription(operation, channel, assetType, pair, false, true)
}

// TradesSubscription to subscribe or unsubscribe to "trades" channel to retrieve the recent trades data. Data will be pushed whenever there is a trade. Every update contain only one trade.
func (ok *Okx) TradesSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "trades", assetType, pair, false, true)
}

// EstimatedDeliveryExercisePriceChannel to subscribe or unsubscribe to "estimated-price" channel to retrieve the estimated delivery/exercise price of FUTURES contracts and OPTION.
func (ok *Okx) EstimatedDeliveryExercisePriceSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "estimated-price", assetType, pair, true, false, true)
}

// MarkPriceSubscription to subscribe or unsubscribe to to "mark-price" to retrieve the mark price. Data will be pushed every 200 ms when the mark price changes, and will be pushed every 10 seconds when the mark price does not change.
func (ok *Okx) MarkPriceSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "mark-price", assetType, pair, false, true)
}

// MarkPriceCandlesticksSubscription to subscribe or unsubscribe to "mark-price-candles" channels to retrieve the candlesticks data of the mark price. Data will be pushed every 500 ms.
func (ok *Okx) MarkPriceCandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(channel, "mark-price-candle1Y") || strings.EqualFold(channel, "mark-price-candle6M") || strings.EqualFold(channel, "mark-price-candle3M") || strings.EqualFold(channel, "mark-price-candle1M") || strings.EqualFold(channel, "mark-price-candle1W") || strings.EqualFold(channel, "mark-price-candle1D") || strings.EqualFold(channel, "mark-price-candle2D") || strings.EqualFold(channel, "mark-price-candle3D") || strings.EqualFold(channel, "mark-price-candle5D") || strings.EqualFold(channel, "mark-price-candle12H") || strings.EqualFold(channel, "mark-price-candle6H") || strings.EqualFold(channel, "mark-price-candle4H") || strings.EqualFold(channel, "mark-price-candle2H") || strings.EqualFold(channel, "mark-price-candle1H") || strings.EqualFold(channel, "mark-price-candle30m") || strings.EqualFold(channel, "mark-price-candle15m") || strings.EqualFold(channel, "mark-price-candle5m") || strings.EqualFold(channel, "mark-price-candle3m") || strings.EqualFold(channel, "mark-price-candle1m") || strings.EqualFold(channel, "mark-price-candle1Yutc") || strings.EqualFold(channel, "mark-price-candle3Mutc") || strings.EqualFold(channel, "mark-price-candle1Mutc") || strings.EqualFold(channel, "mark-price-candle1Wutc") || strings.EqualFold(channel, "mark-price-candle1Dutc") || strings.EqualFold(channel, "mark-price-candle2Dutc") || strings.EqualFold(channel, "mark-price-candle3Dutc") || strings.EqualFold(channel, "mark-price-candle5Dutc") || strings.EqualFold(channel, "mark-price-candle12Hutc") || strings.EqualFold(channel, "mark-price-candle6Hutc")) {
		return nil, errMissingValidChannelInformation
	}
	return ok.WsChannelSubscription(operation, channel, assetType, pair, false, true)
}

// PriceLimitSubscription subscribe or unsubscribe to "price-limit" channel to retrieve the maximum buy price and minimum sell price of the instrument. Data will be pushed every 5 seconds when there are changes in limits, and will not be pushed when there is no changes on limit.
func (ok *Okx) PriceLimitSubscription(operation, instrumentID string) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(operation, "subscribe") || strings.EqualFold(operation, "unsubscribe")) {
		return nil, errInvalidWebsocketEvent
	}
	var er error
	input := &SubscriptionOperationInput{
		Operation: operation,
		Arguments: []SubscriptionInfo{
			{
				Channel:      "price-limit",
				InstrumentID: instrumentID,
			},
		},
	}
	respData, er := ok.Websocket.Conn.SendMessageReturnResponse("price-limit", input)
	if er != nil {
		return nil, er
	}
	var resp SubscriptionOperationResponse
	if er = json.Unmarshal(respData, &resp); er != nil {
		return nil, er
	}
	if strings.EqualFold(resp.Event, "error") || strings.EqualFold(resp.Code, "60012") {
		if len(resp.Msg) == 0 {
			return nil, fmt.Errorf("%s %s error %s", "price-limit", operation, string(respData))
		}
		return nil, errors.New(resp.Msg)
	}
	return &resp, nil
}

// OrderBooksSubscription subscribe or unsubscribe to "books*" channel to retrieve order book data.
func (ok *Okx) OrderBooksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(channel, "books") || strings.EqualFold(channel, "books5") || strings.EqualFold(channel, "books50-l2-tbt") || strings.EqualFold(channel, "books-l2-tbt")) {
		return nil, errMissingValidChannelInformation
	}
	return ok.WsChannelSubscription(operation, channel, assetType, pair, false, true)
}

// OptionSummarySubscription a method to subscribe or unsubscribe to "opt-summary" channel
// to retrieve detailed pricing information of all OPTION contracts. Data will be pushed at once.
func (ok *Okx) OptionSummarySubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "opt-summary", assetType, pair, false, false, true)
}

// FundingRateSubscription a methos to subscribe and unsubscribe to "funding-rate" channel.
// retrieve funding rate. Data will be pushed in 30s to 90s.
func (ok *Okx) FundingRateSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "funding-rate", assetType, pair, false, true)
}

// IndexCandlesticksSubscription a method to subscribe and unsubscribe to "index-candle*" channel
// to retrieve the candlesticks data of the index. Data will be pushed every 500 ms.
func (ok *Okx) IndexCandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	if !(strings.EqualFold(channel, "index-candle1Y") || strings.EqualFold(channel, "index-candle6M") || strings.EqualFold(channel, "index-candle3M") || strings.EqualFold(channel, "index-candle1M") || strings.EqualFold(channel, "index-candle1W") || strings.EqualFold(channel, "index-candle1D") || strings.EqualFold(channel, "index-candle2D") || strings.EqualFold(channel, "index-candle3D") || strings.EqualFold(channel, "index-candle5D") || strings.EqualFold(channel, "index-candle12H") ||
		strings.EqualFold(channel, "index-candle6H") || strings.EqualFold(channel, "index-candle4H") || strings.EqualFold(channel, "index -candle2H") || strings.EqualFold(channel, "index-candle1H") || strings.EqualFold(channel, "index-candle30m") || strings.EqualFold(channel, "index-candle15m") || strings.EqualFold(channel, "index-candle5m") || strings.EqualFold(channel, "index-candle3m") || strings.EqualFold(channel, "index-candle1m") || strings.EqualFold(channel, "index-candle1Yutc") || strings.EqualFold(channel, "index-candle3Mutc") || strings.EqualFold(channel, "index-candle1Mutc") || strings.EqualFold(channel, "index-candle1Wutc") || strings.EqualFold(channel, "index-candle1Dutc") || strings.EqualFold(channel, "index-candle2Dutc") || strings.EqualFold(channel, "index-candle3Dutc") || strings.EqualFold(channel, "index-candle5Dutc") || strings.EqualFold(channel, "index-candle12Hutc") || strings.EqualFold(channel, "index-candle6Hutc")) {
		return nil, errMissingValidChannelInformation
	}
	return ok.WsChannelSubscription(operation, channel, assetType, pair, false, true)
}

// IndexTickerChannel a method to subscribe and unsubscribe to "index-tickers" channel
func (ok *Okx) IndexTickerChannel(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "index-tickers", assetType, pair, false, true)
}

// StatusSubscription get the status of system maintenance and push when the system maintenance status changes.
// First subscription: "Push the latest change data"; every time there is a state change, push the changed content
func (ok *Okx) StatusSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "status", assetType, pair)
}

// PublicStructureBlockTrades a method to subscribe or unsubscribe to "public-struc-block-trades" channel
func (ok *Okx) PublicStructureBlockTradesSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "public-struc-block-trades", assetType, pair)
}

// BlocTickerSubscription a method to subscribe and unsubscribe to a "block-tickers" channel to retrieve the latest block trading volume in the last 24 hours.
// The data will be pushed when triggered by transaction execution event. In addition, it will also be pushed in 5 minutes interval according to subscription granularity.
func (ok *Okx) BlockTickerSubscription(operation string, assetType asset.Item, pair currency.Pair) (*SubscriptionOperationResponse, error) {
	return ok.WsChannelSubscription(operation, "block-tickers", assetType, pair, false, true)
}
