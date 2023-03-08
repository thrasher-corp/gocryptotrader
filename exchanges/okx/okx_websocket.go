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
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// responseStream a channel thought which the data coming from the two websocket connection will go through.
var responseStream = make(chan stream.Response)

// defaultSubscribedChannels list of channels which are subscribed by default
var defaultSubscribedChannels = []string{
	okxChannelTrades,
	okxChannelOrderBooks,
	okxChannelCandle5m,
	okxChannelTickers,
}
var candlestickChannelsMap = map[string]bool{okxChannelCandle1Y: true, okxChannelCandle6M: true, okxChannelCandle3M: true, okxChannelCandle1M: true, okxChannelCandle1W: true, okxChannelCandle1D: true, okxChannelCandle2D: true, okxChannelCandle3D: true, okxChannelCandle5D: true, okxChannelCandle12H: true, okxChannelCandle6H: true, okxChannelCandle4H: true, okxChannelCandle2H: true, okxChannelCandle1H: true, okxChannelCandle30m: true, okxChannelCandle15m: true, okxChannelCandle5m: true, okxChannelCandle3m: true, okxChannelCandle1m: true, okxChannelCandle1Yutc: true, okxChannelCandle3Mutc: true, okxChannelCandle1Mutc: true, okxChannelCandle1Wutc: true, okxChannelCandle1Dutc: true, okxChannelCandle2Dutc: true, okxChannelCandle3Dutc: true, okxChannelCandle5Dutc: true, okxChannelCandle12Hutc: true, okxChannelCandle6Hutc: true}
var candlesticksMarkPriceMap = map[string]bool{okxChannelMarkPriceCandle1Y: true, okxChannelMarkPriceCandle6M: true, okxChannelMarkPriceCandle3M: true, okxChannelMarkPriceCandle1M: true, okxChannelMarkPriceCandle1W: true, okxChannelMarkPriceCandle1D: true, okxChannelMarkPriceCandle2D: true, okxChannelMarkPriceCandle3D: true, okxChannelMarkPriceCandle5D: true, okxChannelMarkPriceCandle12H: true, okxChannelMarkPriceCandle6H: true, okxChannelMarkPriceCandle4H: true, okxChannelMarkPriceCandle2H: true, okxChannelMarkPriceCandle1H: true, okxChannelMarkPriceCandle30m: true, okxChannelMarkPriceCandle15m: true, okxChannelMarkPriceCandle5m: true, okxChannelMarkPriceCandle3m: true, okxChannelMarkPriceCandle1m: true, okxChannelMarkPriceCandle1Yutc: true, okxChannelMarkPriceCandle3Mutc: true, okxChannelMarkPriceCandle1Mutc: true, okxChannelMarkPriceCandle1Wutc: true, okxChannelMarkPriceCandle1Dutc: true, okxChannelMarkPriceCandle2Dutc: true, okxChannelMarkPriceCandle3Dutc: true, okxChannelMarkPriceCandle5Dutc: true, okxChannelMarkPriceCandle12Hutc: true, okxChannelMarkPriceCandle6Hutc: true}
var candlesticksIndexPriceMap = map[string]bool{okxChannelIndexCandle1Y: true, okxChannelIndexCandle6M: true, okxChannelIndexCandle3M: true, okxChannelIndexCandle1M: true, okxChannelIndexCandle1W: true, okxChannelIndexCandle1D: true, okxChannelIndexCandle2D: true, okxChannelIndexCandle3D: true, okxChannelIndexCandle5D: true, okxChannelIndexCandle12H: true, okxChannelIndexCandle6H: true, okxChannelIndexCandle4H: true, okxChannelIndexCandle2H: true, okxChannelIndexCandle1H: true, okxChannelIndexCandle30m: true, okxChannelIndexCandle15m: true, okxChannelIndexCandle5m: true, okxChannelIndexCandle3m: true, okxChannelIndexCandle1m: true, okxChannelIndexCandle1Yutc: true, okxChannelIndexCandle3Mutc: true, okxChannelIndexCandle1Mutc: true, okxChannelIndexCandle1Wutc: true, okxChannelIndexCandle1Dutc: true, okxChannelIndexCandle2Dutc: true, okxChannelIndexCandle3Dutc: true, okxChannelIndexCandle5Dutc: true, okxChannelIndexCandle12Hutc: true, okxChannelIndexCandle6Hutc: true}

const (
	// allowableIterations use the first 25 bids and asks in the full load to form a string
	allowableIterations = 25

	// OkxOrderBookSnapshot orderbook push data type 'snapshot'
	OkxOrderBookSnapshot = "snapshot"
	// OkxOrderBookUpdate orderbook push data type 'update'
	OkxOrderBookUpdate = "update"

	// ColonDelimiter to be used in validating checksum
	ColonDelimiter = ":"

	// maxConnByteLen otal length of multiple channels cannot exceed 4096 bytes.
	maxConnByteLen = 4096

	// Candlestick channels

	markPrice        = "mark-price-"
	indexCandlestick = "index-"
	candle           = "candle"

	// Candlesticks

	okxChannelCandle1Y     = candle + "1Y"
	okxChannelCandle6M     = candle + "6M"
	okxChannelCandle3M     = candle + "3M"
	okxChannelCandle1M     = candle + "1M"
	okxChannelCandle1W     = candle + "1W"
	okxChannelCandle1D     = candle + "1D"
	okxChannelCandle2D     = candle + "2D"
	okxChannelCandle3D     = candle + "3D"
	okxChannelCandle5D     = candle + "5D"
	okxChannelCandle12H    = candle + "12H"
	okxChannelCandle6H     = candle + "6H"
	okxChannelCandle4H     = candle + "4H"
	okxChannelCandle2H     = candle + "2H"
	okxChannelCandle1H     = candle + "1H"
	okxChannelCandle30m    = candle + "30m"
	okxChannelCandle15m    = candle + "15m"
	okxChannelCandle5m     = candle + "5m"
	okxChannelCandle3m     = candle + "3m"
	okxChannelCandle1m     = candle + "1m"
	okxChannelCandle1Yutc  = candle + "1Yutc"
	okxChannelCandle3Mutc  = candle + "3Mutc"
	okxChannelCandle1Mutc  = candle + "1Mutc"
	okxChannelCandle1Wutc  = candle + "1Wutc"
	okxChannelCandle1Dutc  = candle + "1Dutc"
	okxChannelCandle2Dutc  = candle + "2Dutc"
	okxChannelCandle3Dutc  = candle + "3Dutc"
	okxChannelCandle5Dutc  = candle + "5Dutc"
	okxChannelCandle12Hutc = candle + "12Hutc"
	okxChannelCandle6Hutc  = candle + "6Hutc"

	// Ticker channel
	okxChannelTickers                = "tickers"
	okxChannelIndexTickers           = "index-tickers"
	okxChannelStatus                 = "status"
	okxChannelPublicStrucBlockTrades = "public-struc-block-trades"
	okxChannelBlockTickers           = "block-tickers"

	// Private Channels
	okxChannelAccount              = "account"
	okxChannelPositions            = "positions"
	okxChannelBalanceAndPosition   = "balance_and_position"
	okxChannelOrders               = "orders"
	okxChannelAlgoOrders           = "orders-algo"
	okxChannelAlgoAdvance          = "algo-advance"
	okxChannelLiquidationWarning   = "liquidation-warning"
	okxChannelAccountGreeks        = "account-greeks"
	okxChannelRFQs                 = "rfqs"
	okxChannelQuotes               = "quotes"
	okxChannelStructureBlockTrades = "struc-block-trades"
	okxChannelSpotGridOrder        = "grid-orders-spot"
	okxChannelGridOrdersContract   = "grid-orders-contract"
	okxChannelGridPositions        = "grid-positions"
	okcChannelGridSubOrders        = "grid-sub-orders"
	okxChannelInstruments          = "instruments"
	okxChannelOpenInterest         = "open-interest"
	okxChannelTrades               = "trades"

	okxChannelEstimatedPrice  = "estimated-price"
	okxChannelMarkPrice       = "mark-price"
	okxChannelPriceLimit      = "price-limit"
	okxChannelOrderBooks      = "books"
	okxChannelOrderBooks5     = "books5"
	okxChannelOrderBooks50TBT = "books50-l2-tbt"
	okxChannelOrderBooksTBT   = "books-l2-tbt"
	okxChannelBBOTBT          = "bbo-tbt"
	okxChannelOptSummary      = "opt-summary"
	okxChannelFundingRate     = "funding-rate"

	// Index Candlesticks Channels
	okxChannelIndexCandle1Y     = indexCandlestick + okxChannelCandle1Y
	okxChannelIndexCandle6M     = indexCandlestick + okxChannelCandle6M
	okxChannelIndexCandle3M     = indexCandlestick + okxChannelCandle3M
	okxChannelIndexCandle1M     = indexCandlestick + okxChannelCandle1M
	okxChannelIndexCandle1W     = indexCandlestick + okxChannelCandle1W
	okxChannelIndexCandle1D     = indexCandlestick + okxChannelCandle1D
	okxChannelIndexCandle2D     = indexCandlestick + okxChannelCandle2D
	okxChannelIndexCandle3D     = indexCandlestick + okxChannelCandle3D
	okxChannelIndexCandle5D     = indexCandlestick + okxChannelCandle5D
	okxChannelIndexCandle12H    = indexCandlestick + okxChannelCandle12H
	okxChannelIndexCandle6H     = indexCandlestick + okxChannelCandle6H
	okxChannelIndexCandle4H     = indexCandlestick + okxChannelCandle4H
	okxChannelIndexCandle2H     = indexCandlestick + okxChannelCandle2H
	okxChannelIndexCandle1H     = indexCandlestick + okxChannelCandle1H
	okxChannelIndexCandle30m    = indexCandlestick + okxChannelCandle30m
	okxChannelIndexCandle15m    = indexCandlestick + okxChannelCandle15m
	okxChannelIndexCandle5m     = indexCandlestick + okxChannelCandle5m
	okxChannelIndexCandle3m     = indexCandlestick + okxChannelCandle3m
	okxChannelIndexCandle1m     = indexCandlestick + okxChannelCandle1m
	okxChannelIndexCandle1Yutc  = indexCandlestick + okxChannelCandle1Yutc
	okxChannelIndexCandle3Mutc  = indexCandlestick + okxChannelCandle3Mutc
	okxChannelIndexCandle1Mutc  = indexCandlestick + okxChannelCandle1Mutc
	okxChannelIndexCandle1Wutc  = indexCandlestick + okxChannelCandle1Wutc
	okxChannelIndexCandle1Dutc  = indexCandlestick + okxChannelCandle1Dutc
	okxChannelIndexCandle2Dutc  = indexCandlestick + okxChannelCandle2Dutc
	okxChannelIndexCandle3Dutc  = indexCandlestick + okxChannelCandle3Dutc
	okxChannelIndexCandle5Dutc  = indexCandlestick + okxChannelCandle5Dutc
	okxChannelIndexCandle12Hutc = indexCandlestick + okxChannelCandle12Hutc
	okxChannelIndexCandle6Hutc  = indexCandlestick + okxChannelCandle6Hutc

	// Mark price candlesticks channel
	okxChannelMarkPriceCandle1Y     = markPrice + okxChannelCandle1Y
	okxChannelMarkPriceCandle6M     = markPrice + okxChannelCandle6M
	okxChannelMarkPriceCandle3M     = markPrice + okxChannelCandle3M
	okxChannelMarkPriceCandle1M     = markPrice + okxChannelCandle1M
	okxChannelMarkPriceCandle1W     = markPrice + okxChannelCandle1W
	okxChannelMarkPriceCandle1D     = markPrice + okxChannelCandle1D
	okxChannelMarkPriceCandle2D     = markPrice + okxChannelCandle2D
	okxChannelMarkPriceCandle3D     = markPrice + okxChannelCandle3D
	okxChannelMarkPriceCandle5D     = markPrice + okxChannelCandle5D
	okxChannelMarkPriceCandle12H    = markPrice + okxChannelCandle12H
	okxChannelMarkPriceCandle6H     = markPrice + okxChannelCandle6H
	okxChannelMarkPriceCandle4H     = markPrice + okxChannelCandle4H
	okxChannelMarkPriceCandle2H     = markPrice + okxChannelCandle2H
	okxChannelMarkPriceCandle1H     = markPrice + okxChannelCandle1H
	okxChannelMarkPriceCandle30m    = markPrice + okxChannelCandle30m
	okxChannelMarkPriceCandle15m    = markPrice + okxChannelCandle15m
	okxChannelMarkPriceCandle5m     = markPrice + okxChannelCandle5m
	okxChannelMarkPriceCandle3m     = markPrice + okxChannelCandle3m
	okxChannelMarkPriceCandle1m     = markPrice + okxChannelCandle1m
	okxChannelMarkPriceCandle1Yutc  = markPrice + okxChannelCandle1Yutc
	okxChannelMarkPriceCandle3Mutc  = markPrice + okxChannelCandle3Mutc
	okxChannelMarkPriceCandle1Mutc  = markPrice + okxChannelCandle1Mutc
	okxChannelMarkPriceCandle1Wutc  = markPrice + okxChannelCandle1Wutc
	okxChannelMarkPriceCandle1Dutc  = markPrice + okxChannelCandle1Dutc
	okxChannelMarkPriceCandle2Dutc  = markPrice + okxChannelCandle2Dutc
	okxChannelMarkPriceCandle3Dutc  = markPrice + okxChannelCandle3Dutc
	okxChannelMarkPriceCandle5Dutc  = markPrice + okxChannelCandle5Dutc
	okxChannelMarkPriceCandle12Hutc = markPrice + okxChannelCandle12Hutc
	okxChannelMarkPriceCandle6Hutc  = markPrice + okxChannelCandle6Hutc

	// Websocket trade endpoint operations
	okxOpOrder             = "order"
	okxOpBatchOrders       = "batch-orders"
	okxOpCancelOrder       = "cancel-order"
	okxOpBatchCancelOrders = "batch-cancel-orders"
	okxOpAmendOrder        = "amend-order"
	okxOpBatchAmendOrders  = "batch-amend-orders"
)

// WsConnect initiates a websocket connection
func (ok *Okx) WsConnect() error {
	if !ok.Websocket.IsEnabled() || !ok.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	err := ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	ok.Websocket.Wg.Add(2)
	go ok.wsFunnelConnectionData(ok.Websocket.Conn)
	go ok.WsReadData()
	go ok.WsResponseMultiplexer.Run()
	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ok.Websocket.GetWebsocketURL())
	}
	ok.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 10,
	})
	if ok.IsWebsocketAuthenticationSupported() {
		var authDialer websocket.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = ok.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ok *Okx) WsAuth(ctx context.Context, dialer *websocket.Dialer) error {
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ok.Name)
	}
	err := ok.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", ok.Name, okxAPIWebsocketPrivateURL, err)
	}
	ok.Websocket.Wg.Add(1)
	go ok.wsFunnelConnectionData(ok.Websocket.AuthConn)
	ok.Websocket.AuthConn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 5,
	})
	creds, err := ok.GetCredentials(ctx)
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
		Operation: operationLogin,
		Arguments: []WebsocketLoginData{
			{
				APIKey:     creds.Key,
				Passphrase: creds.ClientID,
				Timestamp:  timeUnix,
				Sign:       base64Sign,
			},
		},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(request)
	if err != nil {
		return err
	}
	timer := time.NewTimer(ok.WebsocketResponseCheckTimeout)
	randomID, err := common.GenerateRandomString(16)
	if err != nil {
		return fmt.Errorf("%w, generating random string for incoming websocket response failed", err)
	}
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:    randomID,
		Chan:  wsResponse,
		Event: operationLogin,
	}
	ok.WsRequestSemaphore <- 1
	defer func() {
		<-ok.WsRequestSemaphore
	}()
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Event == operationLogin && data.Code == "0" {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return nil
			} else if data.Event == "error" &&
				(data.Code == "60022" || data.Code == "60009") {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
				return fmt.Errorf("authentication failed with error: %v", ErrorCodes[data.Code])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				request.Operation)
		}
	}
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
	return ok.handleSubscription(operationSubscribe, channelsToSubscribe)
}

// Unsubscribe sends a websocket unsubscription request to several channels to receive data.
func (ok *Okx) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return ok.handleSubscription(operationUnsubscribe, channelsToUnsubscribe)
}

// handleSubscription sends a subscription and unsubscription information thought the websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (ok *Okx) handleSubscription(operation string, subscriptions []stream.ChannelSubscription) error {
	request := WSSubscriptionInformationList{
		Operation: operation,
		Arguments: []SubscriptionInfo{},
	}

	authRequests := WSSubscriptionInformationList{
		Operation: operation,
		Arguments: []SubscriptionInfo{},
	}
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	var channels []stream.ChannelSubscription
	var authChannels []stream.ChannelSubscription
	var err error
	var format currency.PairFormat
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

		if arg.Channel == okxChannelAccount ||
			arg.Channel == okxChannelOrders {
			authSubscription = true
		}
		if arg.Channel == okxChannelGridPositions {
			algoID, _ = subscriptions[i].Params["algoId"].(string)
		}

		if arg.Channel == okcChannelGridSubOrders ||
			arg.Channel == okxChannelGridPositions {
			uid, _ = subscriptions[i].Params["uid"].(string)
		}

		if strings.HasPrefix(arg.Channel, "candle") ||
			arg.Channel == okxChannelTickers ||
			arg.Channel == okxChannelOrderBooks ||
			arg.Channel == okxChannelOrderBooks5 ||
			arg.Channel == okxChannelOrderBooks50TBT ||
			arg.Channel == okxChannelOrderBooksTBT ||
			arg.Channel == okxChannelTrades {
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
				format, err = ok.GetPairFormat(subscriptions[i].Asset, false)
				if err != nil {
					return err
				}
				if subscriptions[i].Currency.Base.String() == "" || subscriptions[i].Currency.Quote.String() == "" {
					return errIncompleteCurrencyPair
				}
				instrumentID = format.Format(subscriptions[i].Currency)
			}
		}
		if arg.Channel == okxChannelInstruments ||
			arg.Channel == okxChannelPositions ||
			arg.Channel == okxChannelOrders ||
			arg.Channel == okxChannelAlgoOrders ||
			arg.Channel == okxChannelAlgoAdvance ||
			arg.Channel == okxChannelLiquidationWarning ||
			arg.Channel == okxChannelSpotGridOrder ||
			arg.Channel == okxChannelGridOrdersContract ||
			arg.Channel == okxChannelEstimatedPrice {
			instrumentType = ok.GetInstrumentTypeFromAssetItem(subscriptions[i].Asset)
		}

		if arg.Channel == okxChannelPositions ||
			arg.Channel == okxChannelOrders ||
			arg.Channel == okxChannelAlgoOrders ||
			arg.Channel == okxChannelEstimatedPrice ||
			arg.Channel == okxChannelOptSummary {
			underlying, _ = ok.GetUnderlying(subscriptions[i].Currency, subscriptions[i].Asset)
		}
		arg.InstrumentID = instrumentID
		arg.Underlying = underlying
		arg.InstrumentType = instrumentType
		arg.UID = uid
		arg.AlgoID = algoID

		if authSubscription {
			var authChunk []byte
			authChannels = append(authChannels, subscriptions[i])
			authRequests.Arguments = append(authRequests.Arguments, arg)
			authChunk, err = json.Marshal(authRequests)
			if err != nil {
				return err
			}
			if len(authChunk) > maxConnByteLen {
				authRequests.Arguments = authRequests.Arguments[:len(authRequests.Arguments)-1]
				i--
				err = ok.Websocket.AuthConn.SendJSONMessage(authRequests)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
					ok.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
				} else {
					ok.Websocket.AddSuccessfulSubscriptions(channels...)
				}
				authChannels = []stream.ChannelSubscription{}
				authRequests.Arguments = []SubscriptionInfo{}
			}
		} else {
			var chunk []byte
			channels = append(channels, subscriptions[i])
			request.Arguments = append(request.Arguments, arg)
			chunk, err = json.Marshal(request)
			if err != nil {
				return err
			}
			if len(chunk) > maxConnByteLen {
				i--
				err = ok.Websocket.Conn.SendJSONMessage(request)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
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
		err = ok.Websocket.Conn.SendJSONMessage(request)
		if err != nil {
			return err
		}
	}

	if len(authRequests.Arguments) > 0 && ok.Websocket.CanUseAuthenticatedEndpoints() {
		err = ok.Websocket.AuthConn.SendJSONMessage(authRequests)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	if operation == operationUnsubscribe {
		channels = append(channels, authChannels...)
		ok.Websocket.RemoveSuccessfulUnsubscriptions(channels...)
	} else {
		channels = append(channels, authChannels...)
		ok.Websocket.AddSuccessfulSubscriptions(channels...)
	}
	return nil
}

// WsReadData read coming messages thought the websocket connection and process the data.
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
	var resp wsIncomingData
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if (resp.Event != "" && (resp.Event == "login" || resp.Event == "error")) || resp.Operation != "" {
		ok.WsResponseMultiplexer.Message <- &resp
		return nil
	}
	if len(resp.Data) == 0 {
		return nil
	}
	switch resp.Argument.Channel {
	case okxChannelCandle1Y, okxChannelCandle6M, okxChannelCandle3M, okxChannelCandle1M, okxChannelCandle1W,
		okxChannelCandle1D, okxChannelCandle2D, okxChannelCandle3D, okxChannelCandle5D, okxChannelCandle12H,
		okxChannelCandle6H, okxChannelCandle4H, okxChannelCandle2H, okxChannelCandle1H, okxChannelCandle30m,
		okxChannelCandle15m, okxChannelCandle5m, okxChannelCandle3m, okxChannelCandle1m, okxChannelCandle1Yutc,
		okxChannelCandle3Mutc, okxChannelCandle1Mutc, okxChannelCandle1Wutc, okxChannelCandle1Dutc,
		okxChannelCandle2Dutc, okxChannelCandle3Dutc, okxChannelCandle5Dutc, okxChannelCandle12Hutc,
		okxChannelCandle6Hutc:
		return ok.wsProcessCandles(respRaw)
	case okxChannelIndexCandle1Y, okxChannelIndexCandle6M, okxChannelIndexCandle3M, okxChannelIndexCandle1M,
		okxChannelIndexCandle1W, okxChannelIndexCandle1D, okxChannelIndexCandle2D, okxChannelIndexCandle3D,
		okxChannelIndexCandle5D, okxChannelIndexCandle12H, okxChannelIndexCandle6H, okxChannelIndexCandle4H,
		okxChannelIndexCandle2H, okxChannelIndexCandle1H, okxChannelIndexCandle30m, okxChannelIndexCandle15m,
		okxChannelIndexCandle5m, okxChannelIndexCandle3m, okxChannelIndexCandle1m, okxChannelIndexCandle1Yutc,
		okxChannelIndexCandle3Mutc, okxChannelIndexCandle1Mutc, okxChannelIndexCandle1Wutc,
		okxChannelIndexCandle1Dutc, okxChannelIndexCandle2Dutc, okxChannelIndexCandle3Dutc, okxChannelIndexCandle5Dutc,
		okxChannelIndexCandle12Hutc, okxChannelIndexCandle6Hutc:
		return ok.wsProcessIndexCandles(respRaw)
	case okxChannelTickers:
		return ok.wsProcessTickers(respRaw)
	case okxChannelIndexTickers:
		var response WsIndexTicker
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelStatus:
		var response WsSystemStatusResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelPublicStrucBlockTrades:
		var response WsPublicTradesResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelBlockTickers:
		var response WsBlockTicker
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelAccountGreeks:
		var response WsGreeks
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelAccount:
		var response WsAccountChannelPushData
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelPositions,
		okxChannelLiquidationWarning:
		var response WsPositionResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelBalanceAndPosition:
		var response WsBalanceAndPosition
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelOrders:
		return ok.wsProcessOrders(respRaw)
	case okxChannelAlgoOrders:
		var response WsAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelAlgoAdvance:
		var response WsAdvancedAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelRFQs:
		var response WsRFQ
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelQuotes:
		var response WsQuote
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelStructureBlockTrades:
		var response WsStructureBlocTrade
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelSpotGridOrder:
		var response WsSpotGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelGridOrdersContract:
		var response WsContractGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelGridPositions:
		var response WsContractGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case okcChannelGridSubOrders:
		var response WsGridSubOrderData
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelInstruments:
		var response WSInstrumentResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelOpenInterest:
		var response WSOpenInterestResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelTrades:
		return ok.wsProcessTrades(respRaw)
	case okxChannelEstimatedPrice:
		var response WsDeliveryEstimatedPrice
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelMarkPrice,
		okxChannelPriceLimit:
		var response WsMarkPrice
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelOrderBooks,
		okxChannelOrderBooks5,
		okxChannelOrderBooks50TBT,
		okxChannelBBOTBT,
		okxChannelOrderBooksTBT:
		return ok.wsProcessOrderBooks(respRaw)
	case okxChannelOptSummary:
		var response WsOptionSummary
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelFundingRate:
		var response WsFundingRate
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelMarkPriceCandle1Y, okxChannelMarkPriceCandle6M, okxChannelMarkPriceCandle3M, okxChannelMarkPriceCandle1M,
		okxChannelMarkPriceCandle1W, okxChannelMarkPriceCandle1D, okxChannelMarkPriceCandle2D, okxChannelMarkPriceCandle3D,
		okxChannelMarkPriceCandle5D, okxChannelMarkPriceCandle12H, okxChannelMarkPriceCandle6H, okxChannelMarkPriceCandle4H,
		okxChannelMarkPriceCandle2H, okxChannelMarkPriceCandle1H, okxChannelMarkPriceCandle30m, okxChannelMarkPriceCandle15m,
		okxChannelMarkPriceCandle5m, okxChannelMarkPriceCandle3m, okxChannelMarkPriceCandle1m, okxChannelMarkPriceCandle1Yutc,
		okxChannelMarkPriceCandle3Mutc, okxChannelMarkPriceCandle1Mutc, okxChannelMarkPriceCandle1Wutc, okxChannelMarkPriceCandle1Dutc,
		okxChannelMarkPriceCandle2Dutc, okxChannelMarkPriceCandle3Dutc, okxChannelMarkPriceCandle5Dutc, okxChannelMarkPriceCandle12Hutc,
		okxChannelMarkPriceCandle6Hutc:
		return ok.wsHandleMarkPriceCandles(respRaw)
	default:
		ok.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: ok.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
}

// wsProcessIndexCandles processes index candlestick data
func (ok *Okx) wsProcessIndexCandles(respRaw []byte) error {
	if respRaw == nil {
		return errNilArgument
	}
	response := struct {
		Argument SubscriptionInfo `json:"arg"`
		Data     [][5]string      `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return errNoCandlestickDataFound
	}
	pair, err := ok.GetPairFromInstrumentID(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	var a asset.Item
	a, _ = GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		candlesData := response.Data[i]
		timestamp, err := strconv.ParseInt(candlesData[0], 10, 64)
		if err != nil {
			return err
		}
		myCandle := stream.KlineData{
			Pair:      pair,
			Exchange:  ok.Name,
			Timestamp: time.UnixMilli(timestamp),
			Interval:  candleInterval,
			AssetType: a,
		}
		myCandle.OpenPrice, err = strconv.ParseFloat(candlesData[1], 64)
		if err != nil {
			return err
		}
		myCandle.HighPrice, err = strconv.ParseFloat(candlesData[2], 64)
		if err != nil {
			return err
		}
		myCandle.LowPrice, err = strconv.ParseFloat(candlesData[3], 64)
		if err != nil {
			return err
		}
		myCandle.ClosePrice, err = strconv.ParseFloat(candlesData[4], 64)
		if err != nil {
			return err
		}
		ok.Websocket.DataHandler <- myCandle
	}
	return nil
}

// wsProcessOrderBooks processes "snapshot" and "update" order book
func (ok *Okx) wsProcessOrderBooks(data []byte) error {
	var response WsOrderBook
	var err error
	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	if response.Argument.Channel == okxChannelOrderBooks &&
		response.Action != OkxOrderBookUpdate &&
		response.Action != OkxOrderBookSnapshot {
		return errors.New("invalid order book action")
	}
	var pair currency.Pair
	var a asset.Item
	a, _ = GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if a == asset.Empty {
		a, err = ok.GuessAssetTypeFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	pair, err = ok.GetPairFromInstrumentID(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	if !pair.IsPopulated() {
		return errIncompleteCurrencyPair
	}
	pair.Delimiter = currency.DashDelimiter
	for i := range response.Data {
		if response.Action == OkxOrderBookSnapshot {
			err = ok.WsProcessSnapshotOrderBook(response.Data[i], pair, a)
			if err != nil {
				if err2 := ok.Subscribe([]stream.ChannelSubscription{
					{
						Channel:  response.Argument.Channel,
						Asset:    a,
						Currency: pair,
					},
				}); err2 != nil {
					ok.Websocket.DataHandler <- err2
				}
				return err
			}
			if ok.Verbose {
				log.Debugf(log.ExchangeSys,
					"%s passed checksum for pair %v",
					ok.Name, pair,
				)
			}
		} else {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			err := ok.WsProcessUpdateOrderbook(response.Data[i], pair, a)
			if err != nil {
				if err2 := ok.Subscribe([]stream.ChannelSubscription{
					{
						Channel:  response.Argument.Channel,
						Asset:    a,
						Currency: pair,
					},
				}); err2 != nil {
					ok.Websocket.DataHandler <- err2
				}
				return err
			}
		}
	}
	return nil
}

// WsProcessSnapshotOrderBook processes snapshot order books
func (ok *Okx) WsProcessSnapshotOrderBook(data WsOrderBookData, pair currency.Pair, a asset.Item) error {
	var err error
	asks, err := ok.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	bids, err := ok.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	newOrderBook := orderbook.Base{
		Asset:           a,
		Asks:            asks,
		Bids:            bids,
		LastUpdated:     data.Timestamp.Time(),
		Pair:            pair,
		Exchange:        ok.Name,
		VerifyOrderbook: ok.CanVerifyOrderbook,
	}
	err = ok.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
	if err != nil {
		return err
	}
	var signedChecksum int32
	signedChecksum, err = ok.CalculateOrderbookChecksum(data)
	if err != nil {
		return fmt.Errorf("%s channel: Orderbook unable to calculate orderbook checksum: %s", ok.Name, err)
	}
	if signedChecksum != data.Checksum {
		return fmt.Errorf("%s channel: Orderbook for %v checksum invalid",
			ok.Name,
			pair)
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing
// orderbook
func (ok *Okx) WsProcessUpdateOrderbook(data WsOrderBookData, pair currency.Pair, a asset.Item) error {
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
	update.Checksum = uint32(data.Checksum)
	return ok.Websocket.Orderbook.Update(update)
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (ok *Okx) AppendWsOrderbookItems(entries [][4]string) ([]orderbook.Item, error) {
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

// CalculateUpdateOrderbookChecksum alternates over the first 25 bid and ask
// entries of a merged orderbook. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func (ok *Okx) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base, checksumVal uint32) error {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + ColonDelimiter + amount + ColonDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + ColonDelimiter + amount + ColonDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ColonDelimiter)
	if crc32.ChecksumIEEE([]byte(checksumStr)) != checksumVal {
		return fmt.Errorf("%s order book update checksum failed for pair %v", ok.Name, orderbookData.Pair)
	}
	return nil
}

// CalculateOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data.
func (ok *Okx) CalculateOrderbookChecksum(orderbookData WsOrderBookData) (int32, error) {
	var checksum strings.Builder
	for i := 0; i < allowableIterations; i++ {
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
	var err error
	err = json.Unmarshal(data, tempo)
	if err != nil {
		return err
	}
	var tsInt int64
	var ts time.Time
	var op float64
	var hp float64
	var lp float64
	var cp float64
	candles := make([]CandlestickMarkPrice, len(tempo.Data))
	for x := range tempo.Data {
		tsInt, err = strconv.ParseInt(tempo.Data[x][0], 10, 64)
		if err != nil {
			return err
		}
		ts = time.UnixMilli(tsInt)
		op, err = strconv.ParseFloat(tempo.Data[x][1], 64)
		if err != nil {
			return err
		}
		hp, err = strconv.ParseFloat(tempo.Data[x][2], 64)
		if err != nil {
			return err
		}
		lp, err = strconv.ParseFloat(tempo.Data[x][3], 64)
		if err != nil {
			return err
		}
		cp, err = strconv.ParseFloat(tempo.Data[x][4], 64)
		if err != nil {
			return err
		}
		candles[x] = CandlestickMarkPrice{
			Timestamp:    ts,
			OpenPrice:    op,
			HighestPrice: hp,
			LowestPrice:  lp,
			ClosePrice:   cp,
		}
	}
	ok.Websocket.DataHandler <- candles
	return nil
}

// wsProcessTrades handles a list of trade information.
func (ok *Okx) wsProcessTrades(data []byte) error {
	var response WsTradeOrder
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	assetType, _ := GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if assetType == asset.Empty {
		assetType, err = ok.GuessAssetTypeFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	trades := make([]trade.Data, len(response.Data))
	for i := range response.Data {
		pair, err := ok.GetPairFromInstrumentID(response.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(response.Data[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			Amount:       response.Data[i].Quantity,
			AssetType:    assetType,
			CurrencyPair: pair,
			Exchange:     ok.Name,
			Side:         side,
			Timestamp:    response.Data[i].Timestamp.Time(),
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
	var err error
	err = json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	assetType, err = GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if err != nil {
		return err
	}
	for x := range response.Data {
		orderType, err := order.StringToOrderType(response.Data[x].OrderType)
		if err != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      err,
			}
		}
		orderStatus, err := order.StringToOrderStatus(response.Data[x].State)
		if err != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      err,
			}
		}
		var a asset.Item
		a, err = GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			ok.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ok.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      err,
			}
			a = assetType
		}
		pair, err = ok.GetPairFromInstrumentID(response.Data[x].InstrumentID)
		if err != nil {
			return err
		}
		ok.Websocket.DataHandler <- &order.Detail{
			Price:           response.Data[x].Price,
			Amount:          response.Data[x].Size,
			ExecutedAmount:  response.Data[x].LastFilledSize.Float64(),
			RemainingAmount: response.Data[x].AccumulatedFillSize.Float64() - response.Data[x].LastFilledSize.Float64(),
			Exchange:        ok.Name,
			OrderID:         response.Data[x].OrderID,
			Type:            orderType,
			Side:            response.Data[x].Side,
			Status:          orderStatus,
			AssetType:       a,
			Date:            response.Data[x].CreationTime,
			Pair:            pair,
		}
	}
	return nil
}

// wsProcessCandles handler to get a list of candlestick messages.
func (ok *Okx) wsProcessCandles(respRaw []byte) error {
	if respRaw == nil {
		return errNilArgument
	}
	response := struct {
		Argument SubscriptionInfo `json:"arg"`
		Data     [][7]string      `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return errNoCandlestickDataFound
	}
	pair, err := ok.GetPairFromInstrumentID(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	var a asset.Item
	a, err = GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
	if err != nil {
		a, err = ok.GuessAssetTypeFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		candlesItem := (response.Data[i])
		timestamp, err := strconv.ParseInt(candlesItem[0], 10, 64)
		if err != nil {
			return err
		}
		myCandle := stream.KlineData{
			Pair:      pair,
			Exchange:  ok.Name,
			Timestamp: time.UnixMilli(timestamp),
			Interval:  candleInterval,
			AssetType: a,
		}
		myCandle.OpenPrice, err = strconv.ParseFloat(candlesItem[1], 64)
		if err != nil {
			return err
		}
		myCandle.HighPrice, err = strconv.ParseFloat(candlesItem[2], 64)
		if err != nil {
			return err
		}
		myCandle.LowPrice, err = strconv.ParseFloat(candlesItem[3], 64)
		if err != nil {
			return err
		}
		myCandle.ClosePrice, err = strconv.ParseFloat(candlesItem[4], 64)
		if err != nil {
			return err
		}
		myCandle.Volume, err = strconv.ParseFloat(candlesItem[5], 64)
		if err != nil {
			return err
		}
		ok.Websocket.DataHandler <- myCandle
	}
	return nil
}

// wsProcessTickers handles the trade ticker information.
func (ok *Okx) wsProcessTickers(data []byte) error {
	var response WSTickerResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	for i := range response.Data {
		a := response.Data[i].InstrumentType
		assetEnabledInMargin := false
		if a == asset.Empty || a == asset.Spot {
			// Here, if the asset type is empty, then we do a guess for the asset type of the instrument depending on the instrument id.
			// But, if the asset type is Spot and since the ticker data for an instrument of asset.Spot and asset.Margin is same, I will do a check if the instrument ID is enabled for asset.Margins.
			// If so, I will send the ticker data to the data handler with Spot and Margin asset types.
			a, err = ok.GuessAssetTypeFromInstrumentID(response.Data[i].InstrumentID)
			if err == nil && a == asset.Margin {
				assetEnabledInMargin = true
			}
		}
		if !ok.SupportsAsset(a) {
			return errInvalidInstrumentType
		}
		var c currency.Pair
		var err error
		c, err = ok.GetPairFromInstrumentID(response.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		var baseVolume float64
		var quoteVolume float64
		switch a {
		case asset.Spot, asset.Margin:
			baseVolume = response.Data[i].Vol24H.Float64()
			quoteVolume = response.Data[i].VolCcy24H.Float64()
		case asset.PerpetualSwap, asset.Futures, asset.Options:
			baseVolume = response.Data[i].VolCcy24H.Float64()
			quoteVolume = response.Data[i].Vol24H.Float64()
		default:
			return fmt.Errorf("%w, asset type %s is not supported", errInvalidInstrumentType, a.String())
		}
		tickData := &ticker.Price{
			ExchangeName: ok.Name,
			Open:         response.Data[i].Open24H.Float64(),
			Volume:       baseVolume,
			QuoteVolume:  quoteVolume,
			High:         response.Data[i].High24H.Float64(),
			Low:          response.Data[i].Low24H.Float64(),
			Bid:          response.Data[i].BidPrice.Float64(),
			Ask:          response.Data[i].BestAskPrice.Float64(),
			BidSize:      response.Data[i].BidSize.Float64(),
			AskSize:      response.Data[i].BestAskSize.Float64(),
			Last:         response.Data[i].LastTradePrice.Float64(),
			AssetType:    a,
			Pair:         c,
			LastUpdated:  response.Data[i].TickerDataGenerationTime.Time(),
		}
		ok.Websocket.DataHandler <- tickData
		if assetEnabledInMargin {
			tickData.AssetType = asset.Margin
			ok.Websocket.DataHandler <- tickData
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
			okxChannelAccount,
			okxChannelOrders,
		)
	}
	for x := range assets {
		pairs, err := ok.GetEnabledPairs(assets[x])
		if err != nil {
			return nil, err
		}
		for y := range defaultSubscribedChannels {
			if defaultSubscribedChannels[y] == okxChannelCandle5m ||
				defaultSubscribedChannels[y] == okxChannelTickers ||
				defaultSubscribedChannels[y] == okxChannelOrders ||
				defaultSubscribedChannels[y] == okxChannelOrderBooks ||
				defaultSubscribedChannels[y] == okxChannelOrderBooks5 ||
				defaultSubscribedChannels[y] == okxChannelOrderBooks50TBT ||
				defaultSubscribedChannels[y] == okxChannelOrderBooksTBT ||
				defaultSubscribedChannels[y] == okxChannelTrades {
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
	if len(subscriptions) >= 240 {
		log.Warnf(log.WebsocketMgr, "OKx has 240 subscription limit, only subscribing within limit. Requested %v", len(subscriptions))
		subscriptions = subscriptions[:239]
	}
	return subscriptions, nil
}

// wsProcessPushData processes push data coming through the websocket channel
func (ok *Okx) wsProcessPushData(data []byte, resp interface{}) error {
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	ok.Websocket.DataHandler <- resp
	return nil
}

// Websocket Trade methods

// WsPlaceOrder places an order thought the websocket connection stream, and returns a SubmitResponse and error message.
func (ok *Okx) WsPlaceOrder(arg *PlaceOrderRequestParam) (*OrderData, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := ok.validatePlaceOrderParams(arg)
	if err != nil {
		return nil, err
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsPlaceOrderInput{
		ID:        randomID,
		Arguments: []PlaceOrderRequestParam{*arg},
		Operation: okxOpOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpOrder && data.ID == input.ID {
				if data.Code == "0" || data.Code == "1" {
					resp, err := data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					if len(resp.Data) != 1 {
						return nil, errNoValidResponseFromServer
					}
					if data.Code == "1" {
						return nil, fmt.Errorf("error code:%s message: %s", resp.Data[0].SCode, resp.Data[0].SMessage)
					}
					return &resp.Data[0], nil
				}
				return nil, fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// WsPlaceMultipleOrder creates an order through the websocket stream.
func (ok *Okx) WsPlaceMultipleOrder(args []PlaceOrderRequestParam) ([]OrderData, error) {
	var err error
	for x := range args {
		arg := args[x]
		err = ok.validatePlaceOrderParams(&arg)
		if err != nil {
			return nil, err
		}
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsPlaceOrderInput{
		ID:        randomID,
		Arguments: args,
		Operation: okxOpBatchOrders,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpBatchOrders && data.ID == input.ID {
				if data.Code == "0" || data.Code == "2" {
					var resp *WSOrderResponse
					resp, err = data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					return resp.Data, nil
				}
				var resp WsOrderActionResponse
				err = resp.populateFromIncomingData(data)
				if err != nil {
					return nil, err
				}
				err = json.Unmarshal(data.Data, &(resp.Data))
				if err != nil {
					return nil, err
				}
				if len(data.Data) == 0 {
					return nil, fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
				}
				var errs error
				for x := range resp.Data {
					if resp.Data[x].SCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s message: %s", resp.Data[x].SCode, resp.Data[x].SMessage))
					}
				}
				return nil, errs
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// WsCancelOrder websocket function to cancel a trade order
func (ok *Okx) WsCancelOrder(arg CancelOrderRequestParam) (*OrderData, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
		return nil, fmt.Errorf("either order id or client supplier id is required")
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsCancelOrderInput{
		ID:        randomID,
		Arguments: []CancelOrderRequestParam{arg},
		Operation: okxOpCancelOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpCancelOrder && data.ID == input.ID {
				if data.Code == "0" || data.Code == "1" {
					resp, err := data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					if len(resp.Data) != 1 {
						return nil, errNoValidResponseFromServer
					}
					if data.Code == "1" {
						return nil, fmt.Errorf("error code: %s message: %s", resp.Data[0].SCode, resp.Data[0].SMessage)
					}
					return &resp.Data[0], nil
				}
				return nil, fmt.Errorf("error code: %s message: %v", data.Code, ErrorCodes[data.Code])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// WsCancelMultipleOrder cancel multiple order through the websocket channel.
func (ok *Okx) WsCancelMultipleOrder(args []CancelOrderRequestParam) ([]OrderData, error) {
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
			return nil, fmt.Errorf("either order id or client supplier id is required")
		}
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsCancelOrderInput{
		ID:        randomID,
		Arguments: args,
		Operation: okxOpBatchCancelOrders,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpBatchCancelOrders && data.ID == input.ID {
				if data.Code == "0" || data.Code == "2" {
					var resp *WSOrderResponse
					resp, err = data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					return resp.Data, nil
				}
				if len(data.Data) == 0 {
					return nil, fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
				}
				var resp WsOrderActionResponse
				err = resp.populateFromIncomingData(data)
				if err != nil {
					return nil, err
				}
				err = json.Unmarshal(data.Data, &(resp.Data))
				if err != nil {
					return nil, err
				}
				var errs error
				for x := range resp.Data {
					if resp.Data[x].SCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s message: %v", resp.Data[x].SCode, resp.Data[x].SMessage))
					}
				}
				return nil, errs
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// WsAmendOrder method to amend trade order using a request thought the websocket channel.
func (ok *Okx) WsAmendOrder(arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientSuppliedOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsAmendOrderInput{
		ID:        randomID,
		Operation: okxOpAmendOrder,
		Arguments: []AmendOrderRequestParams{*arg},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpAmendOrder && data.ID == input.ID {
				if data.Code == "0" || data.Code == "1" {
					resp, err := data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					if len(resp.Data) != 1 {
						return nil, errNoValidResponseFromServer
					}
					if data.Code == "1" {
						return nil, fmt.Errorf("error code: %s message: %s", resp.Data[0].SCode, resp.Data[0].SMessage)
					}
					return &resp.Data[0], nil
				}
				return nil, fmt.Errorf("error code: %s message: %v", data.Code, ErrorCodes[data.Code])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// WsAmendMultipleOrders a request through the websocket connection to amend multiple trade orders.
func (ok *Okx) WsAmendMultipleOrders(args []AmendOrderRequestParams) ([]OrderData, error) {
	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientSuppliedOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := &WsAmendOrderInput{
		ID:        randomID,
		Operation: okxOpBatchAmendOrders,
		Arguments: args,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(input)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(ok.WebsocketResponseMaxLimit)
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:   randomID,
		Chan: wsResponse,
	}
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Operation == okxOpBatchAmendOrders && data.ID == input.ID {
				if data.Code == "0" || data.Code == "2" {
					var resp *WSOrderResponse
					resp, err = data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					return resp.Data, nil
				}
				if len(data.Data) == 0 {
					return nil, fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
				}
				var resp WsOrderActionResponse
				err = resp.populateFromIncomingData(data)
				if err != nil {
					return nil, err
				}
				err = json.Unmarshal(data.Data, &(resp.Data))
				if err != nil {
					return nil, err
				}
				var errs error
				for x := range resp.Data {
					if resp.Data[x].SCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s message: %v", resp.Data[x].SCode, resp.Data[x].SMessage))
					}
				}
				return nil, errs
			}
			continue
		case <-timer.C:
			timer.Stop()
			return nil, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// Run this functions distributes websocket request responses to
func (m *wsRequestDataChannelsMultiplexer) Run() {
	tickerData := time.NewTicker(time.Second)
	for {
		select {
		case <-tickerData.C:
			for x, myChan := range m.WsResponseChannelsMap {
				if myChan == nil {
					delete(m.WsResponseChannelsMap, x)
				}
			}
		case id := <-m.Unregister:
			delete(m.WsResponseChannelsMap, id)
		case reg := <-m.Register:
			m.WsResponseChannelsMap[reg.ID] = reg
		case msg := <-m.Message:
			if msg.ID != "" && m.WsResponseChannelsMap[msg.ID] != nil {
				m.WsResponseChannelsMap[msg.ID].Chan <- msg
				continue
			}
			for _, myChan := range m.WsResponseChannelsMap {
				if (msg.Event == "error" || myChan.Event == operationLogin) &&
					(msg.Code == "60009" || msg.Code == "60022" || msg.Code == "0") &&
					strings.Contains(msg.Msg, myChan.Channel) {
					myChan.Chan <- msg
					continue
				} else if msg.Event != myChan.Event ||
					msg.Argument.Channel != myChan.Channel ||
					msg.Argument.InstrumentType != myChan.InstrumentType ||
					msg.Argument.InstrumentID != myChan.InstrumentID {
					continue
				}
				myChan.Chan <- msg
				break
			}
		}
	}
}

// wsChannelSubscription sends a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) wsChannelSubscription(operation, channel string, assetType asset.Item, pair currency.Pair, tInstrumentType, tInstrumentID, tUnderlying bool) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	if channel == "" {
		return errMissingValidChannelInformation
	}
	var underlying string
	var instrumentID string
	var instrumentType string
	var format currency.PairFormat
	var err error
	if tInstrumentType {
		instrumentType = strings.ToLower(ok.GetInstrumentTypeFromAssetItem(assetType))
		if instrumentType != okxInstTypeSpot &&
			instrumentType != okxInstTypeMargin &&
			instrumentType != okxInstTypeSwap &&
			instrumentType != okxInstTypeFutures &&
			instrumentType != okxInstTypeOption {
			instrumentType = okxInstTypeANY
		}
	}
	if tUnderlying {
		if !pair.IsEmpty() {
			underlying, _ = ok.GetUnderlying(pair, assetType)
		}
	}
	if tInstrumentID {
		format, err = ok.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		if !pair.IsPopulated() {
			return errIncompleteCurrencyPair
		}
		instrumentID = format.Format(pair)
		if err != nil {
			instrumentID = ""
		}
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
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	return ok.Websocket.Conn.SendJSONMessage(input)
}

// Private Channel Websocket methods

// wsAuthChannelSubscription send a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) wsAuthChannelSubscription(operation, channel string, assetType asset.Item, pair currency.Pair, uid, algoID string, params wsSubscriptionParameters) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	var underlying string
	var instrumentID string
	var instrumentType string
	var ccy string
	var err error
	var format currency.PairFormat
	if params.InstrumentType {
		instrumentType = strings.ToUpper(ok.GetInstrumentTypeFromAssetItem(assetType))
		if instrumentType != okxInstTypeMargin &&
			instrumentType != okxInstTypeSwap &&
			instrumentType != okxInstTypeFutures &&
			instrumentType != okxInstTypeOption {
			instrumentType = okxInstTypeANY
		}
	}
	if params.Underlying {
		if !pair.IsEmpty() {
			underlying, _ = ok.GetUnderlying(pair, assetType)
		}
	}
	if params.InstrumentID {
		format, err = ok.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		if !pair.IsPopulated() {
			return errIncompleteCurrencyPair
		}
		instrumentID = format.Format(pair)
		if err != nil {
			instrumentID = ""
		}
	}
	if params.Currency {
		if !pair.IsEmpty() {
			if !pair.Base.IsEmpty() {
				ccy = strings.ToUpper(pair.Base.String())
			} else {
				ccy = strings.ToUpper(pair.Quote.String())
			}
		}
	}
	if channel == "" {
		return errMissingValidChannelInformation
	}
	input := &SubscriptionOperationInput{
		Operation: operation,
		Arguments: []SubscriptionInfo{
			{
				Channel:        channel,
				InstrumentType: instrumentType,
				Underlying:     underlying,
				InstrumentID:   instrumentID,
				AlgoID:         algoID,
				Currency:       ccy,
				UID:            uid,
			},
		},
	}
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	return ok.Websocket.AuthConn.SendJSONMessage(input)
}

// WsAccountSubscription retrieve account information. Data will be pushed when triggered by
// events such as placing order, canceling order, transaction execution, etc.
// It will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) WsAccountSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelAccount, assetType, pair, "", "", wsSubscriptionParameters{Currency: true})
}

// WsPositionChannel retrieve the position data. The first snapshot will be sent in accordance with the granularity of the subscription. Data will be pushed when certain actions, such placing or canceling an order, trigger it. It will also be pushed periodically based on the granularity of the subscription.
func (ok *Okx) WsPositionChannel(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelPositions, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true})
}

// BalanceAndPositionSubscription retrieve account balance and position information. Data will be pushed when triggered by events such as filled order, funding transfer.
func (ok *Okx) BalanceAndPositionSubscription(operation, uid string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelBalanceAndPosition, asset.Empty, currency.EMPTYPAIR, uid, "", wsSubscriptionParameters{})
}

// WsOrderChannel for subscribing for orders.
func (ok *Okx) WsOrderChannel(operation string, assetType asset.Item, pair currency.Pair, instrumentID string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelOrders, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AlgoOrdersSubscription for subscribing to algo - order channels
func (ok *Okx) AlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelAlgoOrders, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AdvanceAlgoOrdersSubscription algo order subscription to retrieve advance algo orders (including Iceberg order, TWAP order, Trailing order). Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) AdvanceAlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelAlgoAdvance, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, InstrumentID: true})
}

// PositionRiskWarningSubscription this push channel is only used as a risk warning, and is not recommended as a risk judgment for strategic trading
// In the case that the market is not moving violently, there may be the possibility that the position has been liquidated at the same time that this message is pushed.
func (ok *Okx) PositionRiskWarningSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelLiquidationWarning, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AccountGreeksSubscription algo order subscription to retrieve account greeks information. Data will be pushed when triggered by events such as increase/decrease positions or cash balance in account, and will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) AccountGreeksSubscription(operation string, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelAccountGreeks, asset.Empty, pair, "", "", wsSubscriptionParameters{Currency: true})
}

// RfqSubscription subscription to retrieve Rfq updates on RFQ orders.
func (ok *Okx) RfqSubscription(operation, uid string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelRFQs, asset.Empty, currency.EMPTYPAIR, uid, "", wsSubscriptionParameters{})
}

// QuotesSubscription subscription to retrieve Quote subscription
func (ok *Okx) QuotesSubscription(operation string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelQuotes, asset.Empty, currency.EMPTYPAIR, "", "", wsSubscriptionParameters{})
}

// StructureBlockTradesSubscription to retrieve Structural block subscription
func (ok *Okx) StructureBlockTradesSubscription(operation string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelStructureBlockTrades, asset.Empty, currency.EMPTYPAIR, "", "", wsSubscriptionParameters{})
}

// SpotGridAlgoOrdersSubscription to retrieve spot grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) SpotGridAlgoOrdersSubscription(operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelSpotGridOrder, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, Underlying: true})
}

// ContractGridAlgoOrders to retrieve contract grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) ContractGridAlgoOrders(operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelGridOrdersContract, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, Underlying: true})
}

// GridPositionsSubscription to retrieve grid positions. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) GridPositionsSubscription(operation, algoID string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelGridPositions, asset.Empty, currency.EMPTYPAIR, "", algoID, wsSubscriptionParameters{})
}

// GridSubOrders to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
func (ok *Okx) GridSubOrders(operation, algoID string) error {
	return ok.wsAuthChannelSubscription(operation, okcChannelGridSubOrders, asset.Empty, currency.EMPTYPAIR, "", algoID, wsSubscriptionParameters{})
}

// Public Websocket stream subscription

// InstrumentsSubscription to subscribe for instruments. The full instrument list will be pushed
// for the first time after subscription. Subsequently, the instruments will be pushed if there is any change to the instrument’s state (such as delivery of FUTURES,
// exercise of OPTION, listing of new contracts / trading pairs, trading suspension, etc.).
func (ok *Okx) InstrumentsSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelInstruments, assetType, pair, true, false, false)
}

// TickersSubscription subscribing to "ticker" channel to retrieve the last traded price, bid price, ask price and 24-hour trading volume of instruments. Data will be pushed every 100 ms.
func (ok *Okx) TickersSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelTickers, assetType, pair, false, true, false)
}

// OpenInterestSubscription to subscribe or unsubscribe to "open-interest" channel to retrieve the open interest. Data will by pushed every 3 seconds.
func (ok *Okx) OpenInterestSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	if assetType != asset.Futures && assetType != asset.Options && assetType != asset.PerpetualSwap {
		return fmt.Errorf("%w, only FUTURES, SWAP and OPTION asset types are supported", errInvalidInstrumentType)
	}
	return ok.wsChannelSubscription(operation, okxChannelOpenInterest, assetType, pair, false, true, false)
}

// CandlesticksSubscription to subscribe or unsubscribe to "candle" channels to retrieve the candlesticks data of an instrument. the push frequency is the fastest interval 500ms push the data.
func (ok *Okx) CandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlestickChannelsMap[channel]; !okay {
		return errMissingValidChannelInformation
	}
	return ok.wsChannelSubscription(operation, channel, assetType, pair, false, true, false)
}

// TradesSubscription to subscribe or unsubscribe to "trades" channel to retrieve the recent trades data. Data will be pushed whenever there is a trade. Every update contain only one trade.
func (ok *Okx) TradesSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelTrades, assetType, pair, false, true, false)
}

// EstimatedDeliveryExercisePriceSubscription to subscribe or unsubscribe to "estimated-price" channel to retrieve the estimated delivery/exercise price of FUTURES contracts and OPTION.
func (ok *Okx) EstimatedDeliveryExercisePriceSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	if assetType != asset.Futures && assetType != asset.Options {
		return fmt.Errorf("%w, only FUTURES and OPTION asset types are supported", errInvalidInstrumentType)
	}
	return ok.wsChannelSubscription(operation, okxChannelEstimatedPrice, assetType, pair, true, true, false)
}

// MarkPriceSubscription to subscribe or unsubscribe to the "mark-price" to retrieve the mark price. Data will be pushed every 200 ms when the mark price changes, and will be pushed every 10 seconds when the mark price does not change.
func (ok *Okx) MarkPriceSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelMarkPrice, assetType, pair, false, true, false)
}

// MarkPriceCandlesticksSubscription to subscribe or unsubscribe to "mark-price-candles" channels to retrieve the candlesticks data of the mark price. Data will be pushed every 500 ms.
func (ok *Okx) MarkPriceCandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlesticksMarkPriceMap[channel]; !okay {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(operation, channel, assetType, pair, false, true, false)
}

// PriceLimitSubscription subscribe or unsubscribe to "price-limit" channel to retrieve the maximum buy price and minimum sell price of the instrument. Data will be pushed every 5 seconds when there are changes in limits, and will not be pushed when there is no changes on limit.
func (ok *Okx) PriceLimitSubscription(operation string, pair currency.Pair) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	return ok.wsChannelSubscription(operation, okxChannelPriceLimit, asset.Empty, pair, false, true, false)
}

// OrderBooksSubscription subscribe or unsubscribe to "books*" channel to retrieve order book data.
func (ok *Okx) OrderBooksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if channel != okxChannelOrderBooks && channel != okxChannelOrderBooks5 && channel != okxChannelOrderBooks50TBT && channel != okxChannelOrderBooksTBT && channel != okxChannelBBOTBT {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(operation, channel, assetType, pair, false, true, false)
}

// OptionSummarySubscription a method to subscribe or unsubscribe to "opt-summary" channel
// to retrieve detailed pricing information of all OPTION contracts. Data will be pushed at once.
func (ok *Okx) OptionSummarySubscription(operation string, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelOptSummary, asset.Options, pair, false, false, true)
}

// FundingRateSubscription a method to subscribe and unsubscribe to "funding-rate" channel.
// retrieve funding rate. Data will be pushed in 30s to 90s.
func (ok *Okx) FundingRateSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelFundingRate, assetType, pair, false, true, false)
}

// IndexCandlesticksSubscription a method to subscribe and unsubscribe to "index-candle*" channel
// to retrieve the candlesticks data of the index. Data will be pushed every 500 ms.
func (ok *Okx) IndexCandlesticksSubscription(operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlesticksIndexPriceMap[channel]; !okay {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(operation, channel, assetType, pair, false, true, false)
}

// IndexTickerChannel a method to subscribe and unsubscribe to "index-tickers" channel
func (ok *Okx) IndexTickerChannel(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelIndexTickers, assetType, pair, false, true, false)
}

// StatusSubscription get the status of system maintenance and push when the system maintenance status changes.
// First subscription: "Push the latest change data"; every time there is a state change, push the changed content
func (ok *Okx) StatusSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelStatus, assetType, pair, false, false, false)
}

// PublicStructureBlockTradesSubscription a method to subscribe or unsubscribe to "public-struc-block-trades" channel
func (ok *Okx) PublicStructureBlockTradesSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelPublicStrucBlockTrades, assetType, pair, false, false, false)
}

// BlockTickerSubscription a method to subscribe and unsubscribe to a "block-tickers" channel to retrieve the latest block trading volume in the last 24 hours.
// The data will be pushed when triggered by transaction execution event. In addition, it will also be pushed in 5 minutes interval according to subscription granularity.
func (ok *Okx) BlockTickerSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelBlockTickers, assetType, pair, false, true, false)
}
