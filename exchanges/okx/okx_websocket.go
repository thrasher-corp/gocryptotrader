package okx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	candlestickChannelsMap    = map[string]bool{channelCandle1Y: true, channelCandle6M: true, channelCandle3M: true, channelCandle1M: true, channelCandle1W: true, channelCandle1D: true, channelCandle2D: true, channelCandle3D: true, channelCandle5D: true, channelCandle12H: true, channelCandle6H: true, channelCandle4H: true, channelCandle2H: true, channelCandle1H: true, channelCandle30m: true, channelCandle15m: true, channelCandle5m: true, channelCandle3m: true, channelCandle1m: true, channelCandle1Yutc: true, channelCandle3Mutc: true, channelCandle1Mutc: true, channelCandle1Wutc: true, channelCandle1Dutc: true, channelCandle2Dutc: true, channelCandle3Dutc: true, channelCandle5Dutc: true, channelCandle12Hutc: true, channelCandle6Hutc: true}
	candlesticksMarkPriceMap  = map[string]bool{channelMarkPriceCandle1Y: true, channelMarkPriceCandle6M: true, channelMarkPriceCandle3M: true, channelMarkPriceCandle1M: true, channelMarkPriceCandle1W: true, channelMarkPriceCandle1D: true, channelMarkPriceCandle2D: true, channelMarkPriceCandle3D: true, channelMarkPriceCandle5D: true, channelMarkPriceCandle12H: true, channelMarkPriceCandle6H: true, channelMarkPriceCandle4H: true, channelMarkPriceCandle2H: true, channelMarkPriceCandle1H: true, channelMarkPriceCandle30m: true, channelMarkPriceCandle15m: true, channelMarkPriceCandle5m: true, channelMarkPriceCandle3m: true, channelMarkPriceCandle1m: true, channelMarkPriceCandle1Yutc: true, channelMarkPriceCandle3Mutc: true, channelMarkPriceCandle1Mutc: true, channelMarkPriceCandle1Wutc: true, channelMarkPriceCandle1Dutc: true, channelMarkPriceCandle2Dutc: true, channelMarkPriceCandle3Dutc: true, channelMarkPriceCandle5Dutc: true, channelMarkPriceCandle12Hutc: true, channelMarkPriceCandle6Hutc: true}
	candlesticksIndexPriceMap = map[string]bool{channelIndexCandle1Y: true, channelIndexCandle6M: true, channelIndexCandle3M: true, channelIndexCandle1M: true, channelIndexCandle1W: true, channelIndexCandle1D: true, channelIndexCandle2D: true, channelIndexCandle3D: true, channelIndexCandle5D: true, channelIndexCandle12H: true, channelIndexCandle6H: true, channelIndexCandle4H: true, channelIndexCandle2H: true, channelIndexCandle1H: true, channelIndexCandle30m: true, channelIndexCandle15m: true, channelIndexCandle5m: true, channelIndexCandle3m: true, channelIndexCandle1m: true, channelIndexCandle1Yutc: true, channelIndexCandle3Mutc: true, channelIndexCandle1Mutc: true, channelIndexCandle1Wutc: true, channelIndexCandle1Dutc: true, channelIndexCandle2Dutc: true, channelIndexCandle3Dutc: true, channelIndexCandle5Dutc: true, channelIndexCandle12Hutc: true, channelIndexCandle6Hutc: true}
)

var (
	pingMsg = []byte("ping")
	pongMsg = []byte("pong")
)

const (
	// wsOrderbookChecksumDelimiter to be used in validating checksum
	wsOrderbookChecksumDelimiter = ":"
	// allowableIterations use the first 25 bids and asks in the full load to form a string
	allowableIterations = 25
	// wsOrderbookSnapshot orderbook push data type 'snapshot'
	wsOrderbookSnapshot = "snapshot"
	// wsOrderbookUpdate orderbook push data type 'update'
	wsOrderbookUpdate = "update"
	// maxConnByteLen total length of multiple channels cannot exceed 4096 bytes.
	maxConnByteLen = 4096

	// Candlestick channels
	markPrice        = "mark-price-"
	indexCandlestick = "index-"
	candle           = "candle"

	// Spread Order

	// Operations
	okxSpreadOrder           = "sprd-order"
	okxSpreadAmendOrder      = "sprd-amend-order"
	okxSpreadCancelOrder     = "sprd-cancel-order"
	okxSpreadCancelAllOrders = "sprd-mass-cancel"

	// Subscriptions
	okxSpreadOrders = "sprd-orders"
	okxSpreadTrades = "sprd-trades"

	// Public Spread Subscriptions
	okxSpreadOrderbookLevel1 = "sprd-bbo-tbt"
	okxSpreadOrderbook       = "sprd-books5"
	okxSpreadPublicTrades    = "sprd-public-trades"
	okxSpreadPublicTicker    = "sprd-tickers"

	// Withdrawal Info Channel subscriptions
	okxWithdrawalInfo = "withdrawal-info"
	okxDepositInfo    = "deposit-info"

	// Ticker channel
	channelTickers                = "tickers"
	channelIndexTickers           = "index-tickers"
	channelStatus                 = "status"
	channelPublicStrucBlockTrades = "public-struc-block-trades"
	channelPublicBlockTrades      = "public-block-trades"
	channelBlockTickers           = "block-tickers"

	// Private Channels
	channelAccount              = "account"
	channelPositions            = "positions"
	channelBalanceAndPosition   = "balance_and_position"
	channelOrders               = "orders"
	channelAlgoOrders           = "orders-algo"
	channelAlgoAdvance          = "algo-advance"
	channelLiquidationWarning   = "liquidation-warning"
	channelAccountGreeks        = "account-greeks"
	channelRFQs                 = "rfqs"
	channelQuotes               = "quotes"
	channelStructureBlockTrades = "struc-block-trades"
	channelSpotGridOrder        = "grid-orders-spot"
	channelGridOrdersContract   = "grid-orders-contract"
	channelGridPositions        = "grid-positions"
	channelGridSubOrders        = "grid-sub-orders"
	channelRecurringBuy         = "algo-recurring-buy"
	liquidationOrders           = "liquidation-orders"
	adlWarning                  = "adl-warning"
	economicCalendar            = "economic-calendar"

	// Public channels
	channelInstruments     = "instruments"
	channelOpenInterest    = "open-interest"
	channelTrades          = "trades"
	channelAllTrades       = "trades-all"
	channelEstimatedPrice  = "estimated-price"
	channelMarkPrice       = "mark-price"
	channelPriceLimit      = "price-limit"
	channelOrderBooks      = "books"
	channelOptionTrades    = "option-trades"
	channelOrderBooks5     = "books5"
	channelOrderBooks50TBT = "books50-l2-tbt"
	channelOrderBooksTBT   = "books-l2-tbt"
	channelBBOTBT          = "bbo-tbt"
	channelOptSummary      = "opt-summary"
	channelFundingRate     = "funding-rate"

	// Websocket trade endpoint operations
	okxOpOrder             = "order"
	okxOpBatchOrders       = "batch-orders"
	okxOpCancelOrder       = "cancel-order"
	okxOpBatchCancelOrders = "batch-cancel-orders"
	okxOpAmendOrder        = "amend-order"
	okxOpBatchAmendOrders  = "batch-amend-orders"
	okxOpMassCancelOrder   = "mass-cancel"

	// Candlestick lengths
	channelCandle1Y     = candle + "1Y"
	channelCandle6M     = candle + "6M"
	channelCandle3M     = candle + "3M"
	channelCandle1M     = candle + "1M"
	channelCandle1W     = candle + "1W"
	channelCandle1D     = candle + "1D"
	channelCandle2D     = candle + "2D"
	channelCandle3D     = candle + "3D"
	channelCandle5D     = candle + "5D"
	channelCandle12H    = candle + "12H"
	channelCandle6H     = candle + "6H"
	channelCandle4H     = candle + "4H"
	channelCandle2H     = candle + "2H"
	channelCandle1H     = candle + "1H"
	channelCandle30m    = candle + "30m"
	channelCandle15m    = candle + "15m"
	channelCandle5m     = candle + "5m"
	channelCandle3m     = candle + "3m"
	channelCandle1m     = candle + "1m"
	channelCandle1Yutc  = candle + "1Yutc"
	channelCandle3Mutc  = candle + "3Mutc"
	channelCandle1Mutc  = candle + "1Mutc"
	channelCandle1Wutc  = candle + "1Wutc"
	channelCandle1Dutc  = candle + "1Dutc"
	channelCandle2Dutc  = candle + "2Dutc"
	channelCandle3Dutc  = candle + "3Dutc"
	channelCandle5Dutc  = candle + "5Dutc"
	channelCandle12Hutc = candle + "12Hutc"
	channelCandle6Hutc  = candle + "6Hutc"

	// Index Candlesticks Channels
	channelIndexCandle1Y     = indexCandlestick + channelCandle1Y
	channelIndexCandle6M     = indexCandlestick + channelCandle6M
	channelIndexCandle3M     = indexCandlestick + channelCandle3M
	channelIndexCandle1M     = indexCandlestick + channelCandle1M
	channelIndexCandle1W     = indexCandlestick + channelCandle1W
	channelIndexCandle1D     = indexCandlestick + channelCandle1D
	channelIndexCandle2D     = indexCandlestick + channelCandle2D
	channelIndexCandle3D     = indexCandlestick + channelCandle3D
	channelIndexCandle5D     = indexCandlestick + channelCandle5D
	channelIndexCandle12H    = indexCandlestick + channelCandle12H
	channelIndexCandle6H     = indexCandlestick + channelCandle6H
	channelIndexCandle4H     = indexCandlestick + channelCandle4H
	channelIndexCandle2H     = indexCandlestick + channelCandle2H
	channelIndexCandle1H     = indexCandlestick + channelCandle1H
	channelIndexCandle30m    = indexCandlestick + channelCandle30m
	channelIndexCandle15m    = indexCandlestick + channelCandle15m
	channelIndexCandle5m     = indexCandlestick + channelCandle5m
	channelIndexCandle3m     = indexCandlestick + channelCandle3m
	channelIndexCandle1m     = indexCandlestick + channelCandle1m
	channelIndexCandle1Yutc  = indexCandlestick + channelCandle1Yutc
	channelIndexCandle3Mutc  = indexCandlestick + channelCandle3Mutc
	channelIndexCandle1Mutc  = indexCandlestick + channelCandle1Mutc
	channelIndexCandle1Wutc  = indexCandlestick + channelCandle1Wutc
	channelIndexCandle1Dutc  = indexCandlestick + channelCandle1Dutc
	channelIndexCandle2Dutc  = indexCandlestick + channelCandle2Dutc
	channelIndexCandle3Dutc  = indexCandlestick + channelCandle3Dutc
	channelIndexCandle5Dutc  = indexCandlestick + channelCandle5Dutc
	channelIndexCandle12Hutc = indexCandlestick + channelCandle12Hutc
	channelIndexCandle6Hutc  = indexCandlestick + channelCandle6Hutc

	// Mark price candlesticks channel
	channelMarkPriceCandle1Y     = markPrice + channelCandle1Y
	channelMarkPriceCandle6M     = markPrice + channelCandle6M
	channelMarkPriceCandle3M     = markPrice + channelCandle3M
	channelMarkPriceCandle1M     = markPrice + channelCandle1M
	channelMarkPriceCandle1W     = markPrice + channelCandle1W
	channelMarkPriceCandle1D     = markPrice + channelCandle1D
	channelMarkPriceCandle2D     = markPrice + channelCandle2D
	channelMarkPriceCandle3D     = markPrice + channelCandle3D
	channelMarkPriceCandle5D     = markPrice + channelCandle5D
	channelMarkPriceCandle12H    = markPrice + channelCandle12H
	channelMarkPriceCandle6H     = markPrice + channelCandle6H
	channelMarkPriceCandle4H     = markPrice + channelCandle4H
	channelMarkPriceCandle2H     = markPrice + channelCandle2H
	channelMarkPriceCandle1H     = markPrice + channelCandle1H
	channelMarkPriceCandle30m    = markPrice + channelCandle30m
	channelMarkPriceCandle15m    = markPrice + channelCandle15m
	channelMarkPriceCandle5m     = markPrice + channelCandle5m
	channelMarkPriceCandle3m     = markPrice + channelCandle3m
	channelMarkPriceCandle1m     = markPrice + channelCandle1m
	channelMarkPriceCandle1Yutc  = markPrice + channelCandle1Yutc
	channelMarkPriceCandle3Mutc  = markPrice + channelCandle3Mutc
	channelMarkPriceCandle1Mutc  = markPrice + channelCandle1Mutc
	channelMarkPriceCandle1Wutc  = markPrice + channelCandle1Wutc
	channelMarkPriceCandle1Dutc  = markPrice + channelCandle1Dutc
	channelMarkPriceCandle2Dutc  = markPrice + channelCandle2Dutc
	channelMarkPriceCandle3Dutc  = markPrice + channelCandle3Dutc
	channelMarkPriceCandle5Dutc  = markPrice + channelCandle5Dutc
	channelMarkPriceCandle12Hutc = markPrice + channelCandle12Hutc
	channelMarkPriceCandle6Hutc  = markPrice + channelCandle6Hutc

	// Copy trading websocket endpoints.
	copyTrading = "copytrading-notification"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.AllTradesChannel: channelTrades,
	subscription.OrderbookChannel: channelOrderBooks,
	subscription.TickerChannel:    channelTickers,
	subscription.MyAccountChannel: channelAccount,
	subscription.MyOrdersChannel:  channelOrders,
}

// WsConnect initiates a websocket connection
func (ok *Okx) WsConnect() error {
	if !ok.Websocket.IsEnabled() || !ok.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	err := ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	ok.Websocket.Wg.Add(1)
	go ok.wsReadData(ok.Websocket.Conn)
	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ok.Websocket.GetWebsocketURL())
	}
	ok.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	if ok.Websocket.CanUseAuthenticatedEndpoints() {
		err = ok.WsAuth(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ok *Okx) WsAuth(ctx context.Context) error {
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ok.Name)
	}
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = ok.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	ok.Websocket.Wg.Add(1)
	go ok.wsReadData(ok.Websocket.AuthConn)
	ok.Websocket.AuthConn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})

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
	req := WebsocketEventRequest{
		Operation: operationLogin,
		Arguments: []WebsocketLoginData{
			{
				APIKey:     creds.Key,
				Passphrase: creds.ClientID,
				Timestamp:  timeUnix.Unix(),
				Sign:       base64Sign,
			},
		},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, req)
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
			if data.Event == operationLogin && data.StatusCode == "0" {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return nil
			} else if data.Event == "error" &&
				(data.StatusCode == "60022" || data.StatusCode == "60009" || data.StatusCode == "60004") {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
				return fmt.Errorf("%w code: %s message: %s", errWebsocketStreamNotAuthenticated, data.StatusCode, data.Message)
			}
			continue
		case <-timer.C:
			timer.Stop()
			return fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				req.Operation)
		}
	}
}

// wsReadData sends msgs from public and auth websockets to data handler
func (ok *Okx) wsReadData(ws stream.Connection) {
	defer ok.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := ok.WsHandleData(resp.Raw); err != nil {
			ok.Websocket.DataHandler <- err
		}
	}
}

// Subscribe sends a websocket subscription request to several channels to receive data.
func (ok *Okx) Subscribe(channelsToSubscribe subscription.List) error {
	return ok.handleSubscription(operationSubscribe, channelsToSubscribe)
}

// Unsubscribe sends a websocket unsubscription request to several channels to receive data.
func (ok *Okx) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	return ok.handleSubscription(operationUnsubscribe, channelsToUnsubscribe)
}

// handleSubscription sends a subscription and unsubscription information thought the websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (ok *Okx) handleSubscription(operation string, subs subscription.List) error {
	reqs := WSSubscriptionInformationList{Operation: operation}
	authRequests := WSSubscriptionInformationList{Operation: operation}
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	var channels subscription.List
	var authChannels subscription.List
	var errs error
	for i := 0; i < len(subs); i++ {
		s := subs[i]
		var arg SubscriptionInfo
		if err := json.Unmarshal([]byte(s.QualifiedChannel), &arg); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}

		if s.Authenticated {
			authChannels = append(authChannels, s)
			authRequests.Arguments = append(authRequests.Arguments, arg)
			authChunk, err := json.Marshal(authRequests)
			if err != nil {
				return err
			}
			if len(authChunk) > maxConnByteLen {
				authRequests.Arguments = authRequests.Arguments[:len(authRequests.Arguments)-1]
				i--
				err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, authRequests)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
					err = ok.Websocket.RemoveSubscriptions(ok.Websocket.AuthConn, channels...)
				} else {
					err = ok.Websocket.AddSuccessfulSubscriptions(ok.Websocket.AuthConn, channels...)
				}
				if err != nil {
					return err
				}
				authChannels = subscription.List{}
				authRequests.Arguments = []SubscriptionInfo{}
			}
		} else {
			channels = append(channels, s)
			reqs.Arguments = append(reqs.Arguments, arg)
			chunk, err := json.Marshal(reqs)
			if err != nil {
				return err
			}
			if len(chunk) > maxConnByteLen {
				i--
				err = ok.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, reqs)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
					err = ok.Websocket.RemoveSubscriptions(ok.Websocket.Conn, channels...)
				} else {
					err = ok.Websocket.AddSuccessfulSubscriptions(ok.Websocket.Conn, channels...)
				}
				if err != nil {
					return err
				}
				channels = subscription.List{}
				reqs.Arguments = []SubscriptionInfo{}
				continue
			}
		}
	}

	if len(reqs.Arguments) > 0 {
		if err := ok.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, reqs); err != nil {
			return err
		}
	}

	if len(authRequests.Arguments) > 0 && ok.Websocket.CanUseAuthenticatedEndpoints() {
		if err := ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, authRequests); err != nil {
			return err
		}
	}

	channels = append(channels, authChannels...)
	if operation == operationUnsubscribe {
		return ok.Websocket.RemoveSubscriptions(ok.Websocket.Conn, channels...)
	}
	return ok.Websocket.AddSuccessfulSubscriptions(ok.Websocket.Conn, channels...)
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (ok *Okx) WsHandleData(respRaw []byte) error {
	var resp wsIncomingData
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		if bytes.Equal(respRaw, pongMsg) {
			return nil
		}
		return fmt.Errorf("%w unmarshalling %v", err, respRaw)
	}
	if (resp.Event != "" && (resp.Event == "login" || resp.Event == "error")) || resp.Operation != "" {
		ok.WsResponseMultiplexer.Message <- &resp
		return nil
	}
	if len(resp.Data) == 0 {
		return nil
	}
	switch resp.Argument.Channel {
	case channelCandle1Y, channelCandle6M, channelCandle3M, channelCandle1M, channelCandle1W,
		channelCandle1D, channelCandle2D, channelCandle3D, channelCandle5D, channelCandle12H,
		channelCandle6H, channelCandle4H, channelCandle2H, channelCandle1H, channelCandle30m,
		channelCandle15m, channelCandle5m, channelCandle3m, channelCandle1m, channelCandle1Yutc,
		channelCandle3Mutc, channelCandle1Mutc, channelCandle1Wutc, channelCandle1Dutc,
		channelCandle2Dutc, channelCandle3Dutc, channelCandle5Dutc, channelCandle12Hutc,
		channelCandle6Hutc:
		return ok.wsProcessCandles(respRaw)
	case channelIndexCandle1Y, channelIndexCandle6M, channelIndexCandle3M, channelIndexCandle1M,
		channelIndexCandle1W, channelIndexCandle1D, channelIndexCandle2D, channelIndexCandle3D,
		channelIndexCandle5D, channelIndexCandle12H, channelIndexCandle6H, channelIndexCandle4H,
		channelIndexCandle2H, channelIndexCandle1H, channelIndexCandle30m, channelIndexCandle15m,
		channelIndexCandle5m, channelIndexCandle3m, channelIndexCandle1m, channelIndexCandle1Yutc,
		channelIndexCandle3Mutc, channelIndexCandle1Mutc, channelIndexCandle1Wutc,
		channelIndexCandle1Dutc, channelIndexCandle2Dutc, channelIndexCandle3Dutc, channelIndexCandle5Dutc,
		channelIndexCandle12Hutc, channelIndexCandle6Hutc:
		return ok.wsProcessIndexCandles(respRaw)
	case channelTickers:
		return ok.wsProcessTickers(respRaw)
	case channelIndexTickers:
		var response WsIndexTicker
		return ok.wsProcessPushData(respRaw, &response)
	case channelStatus:
		var response WsSystemStatusResponse
		return ok.wsProcessPushData(respRaw, &response)
	case channelPublicStrucBlockTrades:
		var response WsPublicTradesResponse
		return ok.wsProcessPushData(respRaw, &response)
	case channelPublicBlockTrades:
		return ok.wsProcessBlockPublicTrades(respRaw)
	case channelBlockTickers:
		var response WsBlockTicker
		return ok.wsProcessPushData(respRaw, &response)
	case channelAccountGreeks:
		var response WsGreeks
		return ok.wsProcessPushData(respRaw, &response)
	case channelAccount:
		var response WsAccountChannelPushData
		return ok.wsProcessPushData(respRaw, &response)
	case channelPositions,
		channelLiquidationWarning:
		var response WsPositionResponse
		return ok.wsProcessPushData(respRaw, &response)
	case channelBalanceAndPosition:
		var response WsBalanceAndPosition
		return ok.wsProcessPushData(respRaw, &response)
	case channelOrders:
		return ok.wsProcessOrders(respRaw)
	case channelAlgoOrders:
		var response WsAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case channelAlgoAdvance:
		var response WsAdvancedAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case channelRFQs:
		var response WsRFQ
		return ok.wsProcessPushData(respRaw, &response)
	case channelQuotes:
		var response WsQuote
		return ok.wsProcessPushData(respRaw, &response)
	case channelStructureBlockTrades:
		var response WsStructureBlocTrade
		return ok.wsProcessPushData(respRaw, &response)
	case channelSpotGridOrder:
		var response WsSpotGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case channelGridOrdersContract:
		var response WsContractGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case channelGridPositions:
		var response WsContractGridAlgoOrder
		return ok.wsProcessPushData(respRaw, &response)
	case channelGridSubOrders:
		var response WsGridSubOrderData
		return ok.wsProcessPushData(respRaw, &response)
	case channelInstruments:
		var response WSInstrumentResponse
		return ok.wsProcessPushData(respRaw, &response)
	case channelOpenInterest:
		var response WSOpenInterestResponse
		return ok.wsProcessPushData(respRaw, &response)
	case channelTrades,
		channelAllTrades:
		return ok.wsProcessTrades(respRaw)
	case channelEstimatedPrice:
		var response WsDeliveryEstimatedPrice
		return ok.wsProcessPushData(respRaw, &response)
	case channelMarkPrice,
		channelPriceLimit:
		var response WsMarkPrice
		return ok.wsProcessPushData(respRaw, &response)
	case channelOrderBooks5:
		return ok.wsProcessOrderbook5(respRaw)
	case okxSpreadOrderbookLevel1,
		okxSpreadOrderbook:
		return ok.wsProcessSpreadOrderbook(respRaw)
	case okxSpreadPublicTrades:
		return ok.wsProcessPublicSpreadTrades(respRaw)
	case okxSpreadPublicTicker:
		return ok.wsProcessPublicSpreadTicker(respRaw)
	case channelOrderBooks,
		channelOrderBooks50TBT,
		channelBBOTBT,
		channelOrderBooksTBT:
		return ok.wsProcessOrderBooks(respRaw)
	case channelOptionTrades:
		return ok.wsProcessOptionTrades(respRaw)
	case channelOptSummary:
		var response WsOptionSummary
		return ok.wsProcessPushData(respRaw, &response)
	case channelFundingRate:
		var response WsFundingRate
		return ok.wsProcessPushData(respRaw, &response)
	case channelMarkPriceCandle1Y, channelMarkPriceCandle6M, channelMarkPriceCandle3M, channelMarkPriceCandle1M,
		channelMarkPriceCandle1W, channelMarkPriceCandle1D, channelMarkPriceCandle2D, channelMarkPriceCandle3D,
		channelMarkPriceCandle5D, channelMarkPriceCandle12H, channelMarkPriceCandle6H, channelMarkPriceCandle4H,
		channelMarkPriceCandle2H, channelMarkPriceCandle1H, channelMarkPriceCandle30m, channelMarkPriceCandle15m,
		channelMarkPriceCandle5m, channelMarkPriceCandle3m, channelMarkPriceCandle1m, channelMarkPriceCandle1Yutc,
		channelMarkPriceCandle3Mutc, channelMarkPriceCandle1Mutc, channelMarkPriceCandle1Wutc, channelMarkPriceCandle1Dutc,
		channelMarkPriceCandle2Dutc, channelMarkPriceCandle3Dutc, channelMarkPriceCandle5Dutc, channelMarkPriceCandle12Hutc,
		channelMarkPriceCandle6Hutc:
		return ok.wsHandleMarkPriceCandles(respRaw)
	case okxSpreadOrders:
		return ok.wsProcessSpreadOrders(respRaw)
	case okxSpreadTrades:
		return ok.wsProcessSpreadTrades(respRaw)
	case okxWithdrawalInfo:
		resp := &struct {
			Arguments SubscriptionInfo `json:"arg"`
			Data      []WsDepositInfo  `json:"data"`
		}{}
		return ok.wsProcessPushData(respRaw, resp)
	case okxDepositInfo:
		resp := &struct {
			Arguments SubscriptionInfo  `json:"arg"`
			Data      []WsWithdrawlInfo `json:"data"`
		}{}
		return ok.wsProcessPushData(respRaw, resp)
	case channelRecurringBuy:
		resp := &struct {
			Arguments SubscriptionInfo    `json:"arg"`
			Data      []RecurringBuyOrder `json:"data"`
		}{}
		return ok.wsProcessPushData(respRaw, resp)
	case liquidationOrders:
		var resp *LiquidationOrder
		return ok.wsProcessPushData(respRaw, &resp)
	case adlWarning:
		var resp ADLWarning
		return ok.wsProcessPushData(respRaw, &resp)
	case economicCalendar:
		var resp EconomicCalendarResponse
		return ok.wsProcessPushData(respRaw, &resp)
	case copyTrading:
		var resp CopyTradingNotification
		return ok.wsProcessPushData(respRaw, &resp)
	default:
		ok.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: ok.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
}

// wsProcessSpreadTrades handle and process spread order trades
func (ok *Okx) wsProcessSpreadTrades(respRaw []byte) error {
	if respRaw == nil {
		return common.ErrNilPointer
	}
	var resp WsSpreadOrderTrade
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	pair, err := ok.GetPairFromInstrumentID(resp.Argument.SpreadID)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for x := range resp.Data {
		oSide, err := order.StringToOrderSide(resp.Data[x].Side)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			Amount:       resp.Data[x].FillSize.Float64(),
			AssetType:    asset.Spread,
			CurrencyPair: pair,
			Exchange:     ok.Name,
			Side:         oSide,
			Timestamp:    resp.Data[x].Timestamp.Time(),
			TID:          resp.Data[x].TradeID,
			Price:        resp.Data[x].FillPrice.Float64(),
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
}

// wsProcessSpreadOrders retrieve order information from the sprd-order Websocket channel.
// Data will not be pushed when first subscribed.
// Data will only be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) wsProcessSpreadOrders(respRaw []byte) error {
	if respRaw == nil {
		return common.ErrNilPointer
	}
	resp := &struct {
		Argument SubscriptionInfo `json:"arg"`
		Data     []WsSpreadOrder  `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	pair, err := ok.GetPairFromInstrumentID(resp.Argument.SpreadID)
	if err != nil {
		return err
	}

	orderDetails := make([]order.Detail, len(resp.Data))
	for x := range resp.Data {
		oSide, err := order.StringToOrderSide(resp.Data[x].Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(resp.Data[x].State)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(resp.Data[x].OrderType)
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			AssetType:            asset.Spread,
			Amount:               resp.Data[x].Size.Float64(),
			AverageExecutedPrice: resp.Data[x].AveragePrice.Float64(),
			ClientOrderID:        resp.Data[x].ClientOrderID,
			Date:                 resp.Data[x].CreationTime.Time(),
			Exchange:             ok.Name,
			ExecutedAmount:       resp.Data[x].FillSize.Float64(),
			OrderID:              resp.Data[x].OrderID,
			Pair:                 pair,
			Price:                resp.Data[x].Price.Float64(),
			QuoteAmount:          resp.Data[x].Size.Float64() * resp.Data[x].Price.Float64(),
			RemainingAmount:      resp.Data[x].Size.Float64() - resp.Data[x].FillSize.Float64(),
			Side:                 oSide,
			Status:               oStatus,
			Type:                 oType,
			LastUpdated:          resp.Data[x].UpdateTime.Time(),
		}
	}
	ok.Websocket.DataHandler <- orderDetails
	return nil
}

// wsProcessIndexCandles processes index candlestick data
func (ok *Okx) wsProcessIndexCandles(respRaw []byte) error {
	if respRaw == nil {
		return common.ErrNilPointer
	}
	response := struct {
		Argument SubscriptionInfo  `json:"arg"`
		Data     [][5]types.Number `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}

	pair, err := currency.NewPairFromString(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ok.getAssetsFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		candlesData := response.Data[i]
		myCandle := stream.KlineData{
			Pair:       pair,
			Exchange:   ok.Name,
			Timestamp:  time.UnixMilli(candlesData[0].Int64()),
			Interval:   candleInterval,
			OpenPrice:  candlesData[1].Float64(),
			HighPrice:  candlesData[2].Float64(),
			LowPrice:   candlesData[3].Float64(),
			ClosePrice: candlesData[4].Float64(),
		}
		for i := range assets {
			myCandle.AssetType = assets[i]
			ok.Websocket.DataHandler <- myCandle
		}
	}
	return nil
}

// wsProcessPublicSpreadTicker process spread order ticker push data.
func (ok *Okx) wsProcessPublicSpreadTicker(respRaw []byte) error {
	var resp WsSpreadPushData
	data := []WsSpreadPublicTicker{}
	resp.Data = &data
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Argument.SpreadID)
	if err != nil {
		return err
	}
	tickers := make([]ticker.Price, len(data))
	for x := range data {
		tickers[x] = ticker.Price{
			Last:         data[x].Last.Float64(),
			Bid:          data[x].BidPrice.Float64(),
			Ask:          data[x].AskPrice.Float64(),
			Pair:         pair,
			ExchangeName: ok.Name,
			AssetType:    asset.Spread,
			LastUpdated:  data[x].Timestamp.Time(),
		}
	}
	ok.Websocket.DataHandler <- tickers
	return nil
}

// wsProcessPublicSpreadTrades retrieve the recent trades data from sprd-public-trades.
// Data will be pushed whenever there is a trade.
// Every update contains only one trade.
func (ok *Okx) wsProcessPublicSpreadTrades(respRaw []byte) error {
	var resp WsSpreadPushData
	data := []WsSpreadPublicTrade{}
	resp.Data = data
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Argument.SpreadID)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(data))
	for x := range data {
		oSide, err := order.StringToOrderSide(data[x].Side)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			TID:          data[x].TradeID,
			Exchange:     ok.Name,
			CurrencyPair: pair,
			AssetType:    asset.Spread,
			Side:         oSide,
			Price:        data[x].Price.Float64(),
			Amount:       data[x].Size.Float64(),
			Timestamp:    data[x].Timestamp.Time(),
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
}

// wsProcessSpreadOrderbook process spread orderbook data.
func (ok *Okx) wsProcessSpreadOrderbook(respRaw []byte) error {
	var resp WsSpreadOrderbook
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	pair, err := ok.GetPairFromInstrumentID(resp.Arg.SpreadID)
	if err != nil {
		return err
	}
	extractedResponse, err := resp.ExtractSpreadOrder()
	if err != nil {
		return err
	}
	for x := range extractedResponse.Data {
		err = ok.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asset:           asset.Spread,
			Asks:            extractedResponse.Data[x].Asks,
			Bids:            extractedResponse.Data[x].Bids,
			LastUpdated:     resp.Data[x].Timestamp.Time(),
			Pair:            pair,
			Exchange:        ok.Name,
			VerifyOrderbook: ok.CanVerifyOrderbook})
		if err != nil {
			return err
		}
	}
	return nil
}

// wsProcessOrderbook5 processes orderbook data
func (ok *Okx) wsProcessOrderbook5(data []byte) error {
	var resp WsOrderbook5
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	if len(resp.Data) != 1 {
		return fmt.Errorf("%s - no data returned", ok.Name)
	}
	assets, err := ok.getAssetsFromInstrumentID(resp.Argument.InstrumentID)
	if err != nil {
		return err
	}

	pair, err := currency.NewPairFromString(resp.Argument.InstrumentID)
	if err != nil {
		return err
	}

	asks := make([]orderbook.Tranche, len(resp.Data[0].Asks))
	for x := range resp.Data[0].Asks {
		asks[x].Price = resp.Data[0].Asks[x][0].Float64()
		asks[x].Amount = resp.Data[0].Asks[x][1].Float64()
	}

	bids := make([]orderbook.Tranche, len(resp.Data[0].Bids))
	for x := range resp.Data[0].Bids {
		bids[x].Price = resp.Data[0].Bids[x][0].Float64()
		bids[x].Amount = resp.Data[0].Bids[x][1].Float64()
	}

	for x := range assets {
		err = ok.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asset:           assets[x],
			Asks:            asks,
			Bids:            bids,
			LastUpdated:     resp.Data[0].Timestamp.Time(),
			Pair:            pair,
			Exchange:        ok.Name,
			VerifyOrderbook: ok.CanVerifyOrderbook})
		if err != nil {
			return err
		}
	}
	return nil
}

// wsProcessOptionTrades handles options trade data
func (ok *Okx) wsProcessOptionTrades(data []byte) error {
	var resp WsOptionTrades
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		var pair currency.Pair
		pair, err = ok.GetPairFromInstrumentID(resp.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp.Data[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			Amount:       resp.Data[i].Size.Float64(),
			AssetType:    asset.Options,
			CurrencyPair: pair,
			Exchange:     ok.Name,
			Side:         oSide,
			Timestamp:    resp.Data[i].Timestamp.Time(),
			TID:          resp.Data[i].TradeID,
			Price:        resp.Data[i].Price.Float64(),
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
}

// wsProcessOrderBooks processes "snapshot" and "update" order book
func (ok *Okx) wsProcessOrderBooks(data []byte) error {
	var response WsOrderBook
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	if response.Argument.Channel == channelOrderBooks &&
		response.Action != wsOrderbookUpdate &&
		response.Action != wsOrderbookSnapshot {
		return fmt.Errorf("%w, %s", orderbook.ErrInvalidAction, response.Action)
	}
	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ok.getAssetsFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	pair, err := currency.NewPairFromString(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	if !pair.IsPopulated() {
		return currency.ErrCurrencyPairsEmpty
	}
	pair.Delimiter = currency.DashDelimiter
	for i := range response.Data {
		if response.Action == wsOrderbookSnapshot {
			err = ok.WsProcessSnapshotOrderBook(&response.Data[i], pair, assets)
		} else {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			err = ok.WsProcessUpdateOrderbook(&response.Data[i], pair, assets)
		}
		if err != nil {
			if errors.Is(err, errInvalidChecksum) {
				err = ok.Subscribe(subscription.List{
					{
						Channel: response.Argument.Channel,
						Asset:   assets[0],
						Pairs:   currency.Pairs{pair},
					},
				})
				if err != nil {
					ok.Websocket.DataHandler <- err
				}
			} else {
				return err
			}
		}
	}
	if ok.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s passed checksum for pair %v",
			ok.Name, pair,
		)
	}
	return nil
}

// WsProcessSnapshotOrderBook processes snapshot order books
func (ok *Okx) WsProcessSnapshotOrderBook(data *WsOrderBookData, pair currency.Pair, assets []asset.Item) error {
	signedChecksum, err := ok.CalculateOrderbookChecksum(data)
	if err != nil {
		return fmt.Errorf("%w %v: unable to calculate orderbook checksum: %s",
			errInvalidChecksum,
			pair,
			err)
	}
	if signedChecksum != data.Checksum {
		return fmt.Errorf("%w %v",
			errInvalidChecksum,
			pair)
	}

	asks, err := ok.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	bids, err := ok.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	for i := range assets {
		newOrderBook := orderbook.Base{
			Asset:           assets[i],
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
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing
// orderbook
func (ok *Okx) WsProcessUpdateOrderbook(data *WsOrderBookData, pair currency.Pair, assets []asset.Item) error {
	update := orderbook.Update{
		Pair:       pair,
		UpdateTime: data.Timestamp.Time(),
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
	for i := range assets {
		ob := update
		ob.Asset = assets[i]
		err = ok.Websocket.Orderbook.Update(&ob)
		if err != nil {
			return err
		}
	}
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (ok *Okx) AppendWsOrderbookItems(entries [][4]types.Number) (orderbook.Tranches, error) {
	items := make(orderbook.Tranches, len(entries))
	for j := range entries {
		items[j] = orderbook.Tranche{Amount: entries[j][1].Float64(), Price: entries[j][0].Float64()}
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
	for i := range allowableIterations {
		if len(orderbookData.Bids)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Bids[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Bids[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + wsOrderbookChecksumDelimiter + amount + wsOrderbookChecksumDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			price := strconv.FormatFloat(orderbookData.Asks[i].Price, 'f', -1, 64)
			amount := strconv.FormatFloat(orderbookData.Asks[i].Amount, 'f', -1, 64)
			checksum.WriteString(price + wsOrderbookChecksumDelimiter + amount + wsOrderbookChecksumDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), wsOrderbookChecksumDelimiter)
	if crc32.ChecksumIEEE([]byte(checksumStr)) != checksumVal {
		return fmt.Errorf("%s order book update checksum failed for pair %v", ok.Name, orderbookData.Pair)
	}
	return nil
}

// CalculateOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data.
func (ok *Okx) CalculateOrderbookChecksum(orderbookData *WsOrderBookData) (int32, error) {
	var checksum strings.Builder
	for i := range allowableIterations {
		if len(orderbookData.Bids)-1 >= i {
			bidPrice := orderbookData.Bids[i][0].String()
			bidAmount := orderbookData.Bids[i][1].String()
			checksum.WriteString(
				bidPrice +
					wsOrderbookChecksumDelimiter +
					bidAmount +
					wsOrderbookChecksumDelimiter)
		}
		if len(orderbookData.Asks)-1 >= i {
			askPrice := orderbookData.Asks[i][0].String()
			askAmount := orderbookData.Asks[i][1].String()
			checksum.WriteString(askPrice +
				wsOrderbookChecksumDelimiter +
				askAmount +
				wsOrderbookChecksumDelimiter)
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), wsOrderbookChecksumDelimiter)
	return int32(crc32.ChecksumIEEE([]byte(checksumStr))), nil
}

// wsHandleMarkPriceCandles processes candlestick mark price push data as a result of  subscription to "mark-price-candle*" channel.
func (ok *Okx) wsHandleMarkPriceCandles(data []byte) error {
	tempo := &struct {
		Argument SubscriptionInfo  `json:"arg"`
		Data     [][5]types.Number `json:"data"`
	}{}
	err := json.Unmarshal(data, tempo)
	if err != nil {
		return err
	}
	candles := make([]CandlestickMarkPrice, len(tempo.Data))
	for x := range tempo.Data {
		candles[x] = CandlestickMarkPrice{
			Timestamp:    time.UnixMilli(tempo.Data[x][0].Int64()),
			OpenPrice:    tempo.Data[x][1].Float64(),
			HighestPrice: tempo.Data[x][2].Float64(),
			LowestPrice:  tempo.Data[x][3].Float64(),
			ClosePrice:   tempo.Data[x][4].Float64(),
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
	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ok.getAssetsFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	trades := make([]trade.Data, 0, len(response.Data)*len(assets))
	for i := range response.Data {
		pair, err := currency.NewPairFromString(response.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		for j := range assets {
			trades = append(trades, trade.Data{
				Amount:       response.Data[i].Quantity.Float64(),
				AssetType:    assets[j],
				CurrencyPair: pair,
				Exchange:     ok.Name,
				Side:         response.Data[i].Side,
				Timestamp:    response.Data[i].Timestamp.Time(),
				TID:          response.Data[i].TradeID,
				Price:        response.Data[i].Price.Float64(),
			})
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
}

// wsProcessOrders handles websocket order push data responses.
func (ok *Okx) wsProcessOrders(respRaw []byte) error {
	var response WsOrderResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	a, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
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
		pair, err := currency.NewPairFromString(response.Data[x].InstrumentID)
		if err != nil {
			return err
		}

		avgPrice := response.Data[x].AveragePrice.Float64()
		orderAmount := response.Data[x].Size.Float64()
		execAmount := response.Data[x].AccumulatedFillSize.Float64()

		var quoteAmount float64
		if response.Data[x].SizeType == "quote_ccy" {
			// Size is quote amount.
			quoteAmount = orderAmount
			if orderStatus == order.Filled {
				// We prefer to take execAmount over calculating from quoteAmount / avgPrice
				// because it avoids rounding issues
				orderAmount = execAmount
			} else {
				if avgPrice > 0 {
					orderAmount /= avgPrice
				} else {
					// Size not in Base, and we can't derive a sane value for it
					orderAmount = 0
				}
			}
		}

		var remainingAmount float64
		// Float64 rounding may lead to execAmount > orderAmount by a tiny fraction
		// noting that the order can be fully executed before it's marked as status Filled
		if orderStatus != order.Filled && orderAmount > execAmount {
			remainingAmount = orderAmount - execAmount
		}

		d := &order.Detail{
			Amount:               orderAmount,
			AssetType:            a,
			AverageExecutedPrice: avgPrice,
			ClientOrderID:        response.Data[x].ClientOrderID,
			Date:                 response.Data[x].CreationTime.Time(),
			Exchange:             ok.Name,
			ExecutedAmount:       execAmount,
			Fee:                  0.0 - response.Data[x].Fee.Float64(),
			FeeAsset:             response.Data[x].FeeCurrency,
			OrderID:              response.Data[x].OrderID,
			Pair:                 pair,
			Price:                response.Data[x].Price.Float64(),
			QuoteAmount:          quoteAmount,
			RemainingAmount:      remainingAmount,
			Side:                 response.Data[x].Side,
			Status:               orderStatus,
			Type:                 orderType,
		}
		if orderStatus == order.Filled {
			d.CloseTime = response.Data[x].FillTime.Time()
			if d.Amount == 0 {
				d.Amount = d.ExecutedAmount
			}
		}
		ok.Websocket.DataHandler <- d
	}
	return nil
}

// wsProcessCandles handler to get a list of candlestick messages.
func (ok *Okx) wsProcessCandles(respRaw []byte) error {
	if respRaw == nil {
		return common.ErrNilPointer
	}
	response := struct {
		Argument SubscriptionInfo  `json:"arg"`
		Data     [][7]types.Number `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	pair, err := currency.NewPairFromString(response.Argument.InstrumentID)
	if err != nil {
		return err
	}
	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ok.getAssetsFromInstrumentID(response.Argument.InstrumentID)
		if err != nil {
			return err
		}
	}
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		for j := range assets {
			ok.Websocket.DataHandler <- stream.KlineData{
				Timestamp:  time.UnixMilli(response.Data[i][0].Int64()),
				Pair:       pair,
				AssetType:  assets[j],
				Exchange:   ok.Name,
				Interval:   candleInterval,
				OpenPrice:  response.Data[i][1].Float64(),
				ClosePrice: response.Data[i][4].Float64(),
				HighPrice:  response.Data[i][2].Float64(),
				LowPrice:   response.Data[i][3].Float64(),
				Volume:     response.Data[i][5].Float64(),
			}
		}
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
		var assets []asset.Item
		if response.Argument.InstrumentType != "" {
			assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
			if err != nil {
				return err
			}
			assets = append(assets, assetType)
		} else {
			assets, err = ok.getAssetsFromInstrumentID(response.Argument.InstrumentID)
			if err != nil {
				return err
			}
		}
		c, err := currency.NewPairFromString(response.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		var baseVolume float64
		var quoteVolume float64
		if cap(assets) == 2 {
			baseVolume = response.Data[i].Vol24H.Float64()
			quoteVolume = response.Data[i].VolCcy24H.Float64()
		} else {
			baseVolume = response.Data[i].VolCcy24H.Float64()
			quoteVolume = response.Data[i].Vol24H.Float64()
		}
		for j := range assets {
			tickData := &ticker.Price{
				ExchangeName: ok.Name,
				Open:         response.Data[i].Open24H.Float64(),
				Volume:       baseVolume,
				QuoteVolume:  quoteVolume,
				High:         response.Data[i].High24H.Float64(),
				Low:          response.Data[i].Low24H.Float64(),
				Bid:          response.Data[i].BestBidPrice.Float64(),
				Ask:          response.Data[i].BestAskPrice.Float64(),
				BidSize:      response.Data[i].BestBidSize.Float64(),
				AskSize:      response.Data[i].BestAskSize.Float64(),
				Last:         response.Data[i].LastTradePrice.Float64(),
				AssetType:    assets[j],
				Pair:         c,
				LastUpdated:  response.Data[i].TickerDataGenerationTime.Time(),
			}
			ok.Websocket.DataHandler <- tickData
		}
	}
	return nil
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (ok *Okx) generateSubscriptions() (subscription.List, error) {
	return ok.Features.Subscriptions.ExpandTemplates(ok)
}

// GetSubscriptionTemplate returns a subscription channel template
func (ok *Okx) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":     channelName,
		"isSymbolChannel": isSymbolChannel,
		"isAssetChannel":  isAssetChannel,
		"instType":        GetInstrumentTypeFromAssetItem,
	}).Parse(subTplText)
}

// wsProcessBlockPublicTrades handles the recent block trades data by individual legs.
func (ok *Okx) wsProcessBlockPublicTrades(data []byte) error {
	var resp PublicBlockTrades
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		var pair currency.Pair
		pair, err = ok.GetPairFromInstrumentID(resp.Data[i].InstrumentID)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(resp.Data[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			Amount:       resp.Data[i].Size.Float64(),
			AssetType:    asset.Options,
			CurrencyPair: pair,
			Exchange:     ok.Name,
			Side:         oSide,
			Timestamp:    resp.Data[i].Timestamp.Time(),
			TID:          resp.Data[i].TradeID,
			Price:        resp.Data[i].Price.Float64(),
		}
	}
	return trade.AddTradesToBuffer(ok.Name, trades...)
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
func (ok *Okx) WsPlaceOrder(ctx context.Context, arg *PlaceOrderRequestParam) (*OrderData, error) {
	if arg == nil || *arg == (PlaceOrderRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	err := ok.validatePlaceOrderParams(arg)
	if err != nil {
		return nil, err
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []PlaceOrderRequestParam{*arg},
		Operation: okxOpOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), placeOrderEPL, input)
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
				var dataHolder *OrderData
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s error message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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

// WsPlaceMultipleOrders creates an order through the websocket stream.
func (ok *Okx) WsPlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, order.ErrSubmissionIsNil
	}
	var err error
	for x := range args {
		arg := args[x]
		err = ok.validatePlaceOrderParams(&arg)
		if err != nil {
			return nil, err
		}
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: args,
		Operation: okxOpBatchOrders,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), placeMultipleOrdersEPL, input)
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
				if data.StatusCode == "0" || data.StatusCode == "2" {
					var resp *WsPlaceOrderResponse
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
					return nil, fmt.Errorf("error code:%s error message: %v", data.StatusCode, ErrorCodes[data.StatusCode])
				}
				var errs error
				for x := range resp.Data {
					if resp.Data[x].StatusCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s error message: %s", resp.Data[x].StatusCode, resp.Data[x].StatusMessage))
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
func (ok *Okx) WsCancelOrder(ctx context.Context, arg *CancelOrderRequestParam) (*OrderData, error) {
	if arg == nil || *arg == (CancelOrderRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []CancelOrderRequestParam{*arg},
		Operation: okxOpCancelOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, cancelOrderEPL, input)
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
				var dataHolder *OrderData
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s error message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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
func (ok *Okx) WsCancelMultipleOrder(ctx context.Context, args []CancelOrderRequestParam) ([]OrderData, error) {
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: args,
		Operation: okxOpBatchCancelOrders,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), cancelMultipleOrdersEPL, input)
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
				if data.StatusCode == "0" || data.StatusCode == "2" {
					var resp *WsPlaceOrderResponse
					resp, err = data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					return resp.Data, nil
				}
				if len(data.Data) == 0 {
					return nil, fmt.Errorf("error code:%s error message: %v", data.StatusCode, ErrorCodes[data.StatusCode])
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
					if resp.Data[x].StatusCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s error message: %v", resp.Data[x].StatusCode, resp.Data[x].StatusMessage))
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
func (ok *Okx) WsAmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Operation: okxOpAmendOrder,
		Arguments: []AmendOrderRequestParams{*arg},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, amendOrderEPL, input)
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
				var dataHolder *OrderData
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s error message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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
func (ok *Okx) WsAmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]OrderData, error) {
	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := &WsOperationInput{
		ID:        randomID,
		Operation: okxOpBatchAmendOrders,
		Arguments: args,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, amendMultipleOrdersEPL, input)
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
				if data.StatusCode == "0" || data.StatusCode == "2" {
					var resp *WsPlaceOrderResponse
					resp, err = data.copyToPlaceOrderResponse()
					if err != nil {
						return nil, err
					}
					return resp.Data, nil
				}
				if len(data.Data) == 0 {
					return nil, fmt.Errorf("error code:%s error message: %v", data.StatusCode, ErrorCodes[data.StatusCode])
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
					if resp.Data[x].StatusCode != "0" {
						errs = common.AppendError(errs, fmt.Errorf("error code:%s error message: %v", resp.Data[x].StatusCode, resp.Data[x].StatusMessage))
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

// WsMassCancelOrders cancel all the MMP pending orders of an instrument family.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) WsMassCancelOrders(ctx context.Context, args []CancelMassReqParam) (bool, error) {
	for x := range args {
		if args[x] == (CancelMassReqParam{}) {
			return false, common.ErrEmptyParams
		}
		if args[x].InstrumentType == "" {
			return false, fmt.Errorf("%w, instrument type can not be empty", errInvalidInstrumentType)
		}
		if args[x].InstrumentFamily == "" {
			return false, errInstrumentFamilyRequired
		}
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return false, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return false, err
	}
	input := &WsOperationInput{
		ID:        randomID,
		Operation: okxOpMassCancelOrder,
		Arguments: args,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, input)
	if err != nil {
		return false, err
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
			if data.Operation == okxOpMassCancelOrder && data.ID == input.ID {
				if data.StatusCode == "0" || data.StatusCode == "2" {
					resp := []struct {
						Result bool `json:"result"`
					}{}
					err := json.Unmarshal(data.Data, &resp)
					if err != nil {
						return false, err
					}
					if len(data.Data) == 0 {
						return false, fmt.Errorf("error code:%s message: %v", data.StatusCode, ErrorCodes[data.StatusCode])
					}
					return resp[0].Result, nil
				}
				return false, fmt.Errorf("error code:%s message: %v", data.StatusCode, data.Message)
			}
			continue
		case <-timer.C:
			timer.Stop()
			return false, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
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
		case <-m.shutdown:
			// We've consumed the shutdown, so create a new chan for subsequent runs
			m.shutdown = make(chan bool)
			return
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
					(msg.StatusCode == "60009" || msg.StatusCode == "60004" || msg.StatusCode == "60022" || msg.StatusCode == "0") &&
					strings.Contains(msg.Message, myChan.Channel) {
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

// Shutdown causes the multiplexer to exit its Run loop
// All channels are left open, but websocket shutdown first will ensure no more messages block on multiplexer reading
func (m *wsRequestDataChannelsMultiplexer) Shutdown() {
	close(m.shutdown)
}

// wsChannelSubscription sends a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) wsChannelSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair, tInstrumentType, tInstrumentID, tUnderlying bool) error {
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
		instrumentType = GetInstrumentTypeFromAssetItem(assetType)
		if instrumentType != instTypeSpot &&
			instrumentType != instTypeMargin &&
			instrumentType != instTypeSwap &&
			instrumentType != instTypeFutures &&
			instrumentType != instTypeOption {
			instrumentType = instTypeANY
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
			return currency.ErrCurrencyPairsEmpty
		}
		instrumentID = format.Format(pair)
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
	return ok.Websocket.Conn.SendJSONMessage(ctx, request.Unset, input)
}

// Private Channel Websocket methods

// wsAuthChannelSubscription send a subscription or unsubscription request for different channels through the websocket stream.
func (ok *Okx) wsAuthChannelSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair, uid, algoID string, params wsSubscriptionParameters) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	var underlying string
	var instrumentID string
	var instrumentType string
	var ccy string
	if params.InstrumentType {
		instrumentType = GetInstrumentTypeFromAssetItem(assetType)
		if instrumentType != instTypeMargin &&
			instrumentType != instTypeSwap &&
			instrumentType != instTypeFutures &&
			instrumentType != instTypeOption {
			instrumentType = instTypeANY
		}
	}
	if params.Underlying {
		if !pair.IsEmpty() {
			underlying, _ = ok.GetUnderlying(pair, assetType)
		}
	}
	if params.InstrumentID {
		if !pair.IsPopulated() {
			return currency.ErrCurrencyPairsEmpty
		}
		format, err := ok.GetPairFormat(assetType, false)
		if err != nil {
			return err
		}
		instrumentID = format.Format(pair)
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
	return ok.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, input)
}

// WsAccountSubscription retrieve account information. Data will be pushed when triggered by
// events such as placing order, canceling order, transaction execution, etc.
// It will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) WsAccountSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelAccount, assetType, pair, "", "", wsSubscriptionParameters{Currency: true})
}

// WsPositionChannel retrieve the position data. The first snapshot will be sent in accordance with the granularity of the subscription. Data will be pushed when certain actions, such placing or canceling an order, trigger it. It will also be pushed periodically based on the granularity of the subscription.
func (ok *Okx) WsPositionChannel(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelPositions, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true})
}

// BalanceAndPositionSubscription retrieve account balance and position information. Data will be pushed when triggered by events such as filled order, funding transfer.
func (ok *Okx) BalanceAndPositionSubscription(ctx context.Context, operation, uid string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelBalanceAndPosition, asset.Empty, currency.EMPTYPAIR, uid, "", wsSubscriptionParameters{})
}

// WsOrderChannel for subscribing for orders.
func (ok *Okx) WsOrderChannel(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair, _ string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelOrders, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AlgoOrdersSubscription for subscribing to algo - order channels
func (ok *Okx) AlgoOrdersSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelAlgoOrders, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AdvanceAlgoOrdersSubscription algo order subscription to retrieve advance algo orders (including Iceberg order, TWAP order, Trailing order). Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) AdvanceAlgoOrdersSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelAlgoAdvance, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, InstrumentID: true})
}

// PositionRiskWarningSubscription this push channel is only used as a risk warning, and is not recommended as a risk judgment for strategic trading
// In the case that the market is not moving violently, there may be the possibility that the position has been liquidated at the same time that this message is pushed.
func (ok *Okx) PositionRiskWarningSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelLiquidationWarning, assetType, pair, "", "", wsSubscriptionParameters{InstrumentType: true, InstrumentID: true, Underlying: true})
}

// AccountGreeksSubscription algo order subscription to retrieve account greeks information. Data will be pushed when triggered by events such as increase/decrease positions or cash balance in account, and will also be pushed in regular interval according to subscription granularity.
func (ok *Okx) AccountGreeksSubscription(ctx context.Context, operation string, pair currency.Pair) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelAccountGreeks, asset.Empty, pair, "", "", wsSubscriptionParameters{Currency: true})
}

// RFQSubscription subscription to retrieve RFQ updates on RFQ orders.
func (ok *Okx) RFQSubscription(ctx context.Context, operation, uid string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelRFQs, asset.Empty, currency.EMPTYPAIR, uid, "", wsSubscriptionParameters{})
}

// QuotesSubscription subscription to retrieve Quote subscription
func (ok *Okx) QuotesSubscription(ctx context.Context, operation string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelQuotes, asset.Empty, currency.EMPTYPAIR, "", "", wsSubscriptionParameters{})
}

// StructureBlockTradesSubscription to retrieve Structural block subscription
func (ok *Okx) StructureBlockTradesSubscription(ctx context.Context, operation string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelStructureBlockTrades, asset.Empty, currency.EMPTYPAIR, "", "", wsSubscriptionParameters{})
}

// SpotGridAlgoOrdersSubscription to retrieve spot grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) SpotGridAlgoOrdersSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelSpotGridOrder, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, Underlying: true})
}

// ContractGridAlgoOrders to retrieve contract grid algo orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) ContractGridAlgoOrders(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair, algoID string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelGridOrdersContract, assetType, pair, "", algoID, wsSubscriptionParameters{InstrumentType: true, Underlying: true})
}

// GridPositionsSubscription to retrieve grid positions. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing/canceling order.
func (ok *Okx) GridPositionsSubscription(ctx context.Context, operation, algoID string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelGridPositions, asset.Empty, currency.EMPTYPAIR, "", algoID, wsSubscriptionParameters{})
}

// GridSubOrders to retrieve grid sub orders. Data will be pushed when first subscribed. Data will be pushed when triggered by events such as placing order.
func (ok *Okx) GridSubOrders(ctx context.Context, operation, algoID string) error {
	return ok.wsAuthChannelSubscription(ctx, operation, channelGridSubOrders, asset.Empty, currency.EMPTYPAIR, "", algoID, wsSubscriptionParameters{})
}

// Public Websocket stream subscription

// InstrumentsSubscription to subscribe for instruments. The full instrument list will be pushed
// for the first time after subscription. Subsequently, the instruments will be pushed if there is any change to the instrument’s state (such as delivery of FUTURES,
// exercise of OPTION, listing of new contracts / trading pairs, trading suspension, etc.).
func (ok *Okx) InstrumentsSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelInstruments, assetType, pair, true, false, false)
}

// TickersSubscription subscribing to "ticker" channel to retrieve the last traded price, bid price, ask price and 24-hour trading volume of instruments. Data will be pushed every 100 ms.
func (ok *Okx) TickersSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelTickers, assetType, pair, false, true, false)
}

// OpenInterestSubscription to subscribe or unsubscribe to "open-interest" channel to retrieve the open interest. Data will be pushed every 3 seconds.
func (ok *Okx) OpenInterestSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	if assetType != asset.Futures && assetType != asset.Options && assetType != asset.PerpetualSwap {
		return fmt.Errorf("%w, received '%v' only FUTURES, SWAP and OPTION asset types are supported", errInvalidInstrumentType, assetType)
	}
	return ok.wsChannelSubscription(ctx, operation, channelOpenInterest, assetType, pair, false, true, false)
}

// CandlesticksSubscription to subscribe or unsubscribe to "candle" channels to retrieve the candlesticks data of an instrument. the push frequency is the fastest interval 500ms push the data.
func (ok *Okx) CandlesticksSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlestickChannelsMap[channel]; !okay {
		return errMissingValidChannelInformation
	}
	return ok.wsChannelSubscription(ctx, operation, channel, assetType, pair, false, true, false)
}

// TradesSubscription to subscribe or unsubscribe to "trades" channel to retrieve the recent trades data. Data will be pushed whenever there is a trade. Every update contain only one trade.
func (ok *Okx) TradesSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelTrades, assetType, pair, false, true, false)
}

// EstimatedDeliveryExercisePriceSubscription to subscribe or unsubscribe to "estimated-price" channel to retrieve the estimated delivery/exercise price of FUTURES contracts and OPTION.
func (ok *Okx) EstimatedDeliveryExercisePriceSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	if assetType != asset.Futures && assetType != asset.Options {
		return fmt.Errorf("%w, received '%v' only FUTURES and OPTION asset types are supported", errInvalidInstrumentType, assetType)
	}
	return ok.wsChannelSubscription(ctx, operation, channelEstimatedPrice, assetType, pair, true, true, false)
}

// MarkPriceSubscription to subscribe or unsubscribe to the "mark-price" to retrieve the mark price. Data will be pushed every 200 ms when the mark price changes, and will be pushed every 10 seconds when the mark price does not change.
func (ok *Okx) MarkPriceSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelMarkPrice, assetType, pair, false, true, false)
}

// MarkPriceCandlesticksSubscription to subscribe or unsubscribe to "mark-price-candles" channels to retrieve the candlesticks data of the mark price. Data will be pushed every 500 ms.
func (ok *Okx) MarkPriceCandlesticksSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlesticksMarkPriceMap[channel]; !okay {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(ctx, operation, channel, assetType, pair, false, true, false)
}

// PriceLimitSubscription subscribe or unsubscribe to "price-limit" channel to retrieve the maximum buy price and minimum sell price of the instrument. Data will be pushed every 5 seconds when there are changes in limits, and will not be pushed when there is no changes on limit.
func (ok *Okx) PriceLimitSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	return ok.wsChannelSubscription(ctx, operation, channelPriceLimit, assetType, pair, false, true, false)
}

// OrderBooksSubscription subscribe or unsubscribe to "books*" channel to retrieve order book data.
func (ok *Okx) OrderBooksSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if channel != channelOrderBooks && channel != channelOrderBooks5 && channel != channelOrderBooks50TBT && channel != channelOrderBooksTBT && channel != channelBBOTBT {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(ctx, operation, channel, assetType, pair, false, true, false)
}

// OptionSummarySubscription a method to subscribe or unsubscribe to "opt-summary" channel
// to retrieve detailed pricing information of all OPTION contracts. Data will be pushed at once.
func (ok *Okx) OptionSummarySubscription(ctx context.Context, operation string, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelOptSummary, asset.Options, pair, false, false, true)
}

// FundingRateSubscription a method to subscribe and unsubscribe to "funding-rate" channel.
// retrieve funding rate. Data will be pushed in 30s to 90s.
func (ok *Okx) FundingRateSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelFundingRate, assetType, pair, false, true, false)
}

// IndexCandlesticksSubscription a method to subscribe and unsubscribe to "index-candle*" channel
// to retrieve the candlesticks data of the index. Data will be pushed every 500 ms.
func (ok *Okx) IndexCandlesticksSubscription(ctx context.Context, operation, channel string, assetType asset.Item, pair currency.Pair) error {
	if _, okay := candlesticksIndexPriceMap[channel]; !okay {
		return fmt.Errorf("%w channel: %v", errMissingValidChannelInformation, channel)
	}
	return ok.wsChannelSubscription(ctx, operation, channel, assetType, pair, false, true, false)
}

// IndexTickerChannel a method to subscribe and unsubscribe to "index-tickers" channel
func (ok *Okx) IndexTickerChannel(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelIndexTickers, assetType, pair, false, true, false)
}

// StatusSubscription get the status of system maintenance and push when the system maintenance status changes.
// First subscription: "Push the latest change data"; every time there is a state change, push the changed content
func (ok *Okx) StatusSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelStatus, assetType, pair, false, false, false)
}

// PublicStructureBlockTradesSubscription a method to subscribe or unsubscribe to "public-struc-block-trades" channel
func (ok *Okx) PublicStructureBlockTradesSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelPublicStrucBlockTrades, assetType, pair, false, false, false)
}

// BlockTickerSubscription a method to subscribe and unsubscribe to a "block-tickers" channel to retrieve the latest block trading volume in the last 24 hours.
// The data will be pushed when triggered by transaction execution event. In addition, it will also be pushed in 5 minutes interval according to subscription granularity.
func (ok *Okx) BlockTickerSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelBlockTickers, assetType, pair, false, true, false)
}

// PublicBlockTradesSubscription a method to subscribe and unsubscribe to a "public-block-trades" channel to retrieve the recent block trades data by individual legs.
// Each leg in a block trade is pushed in a separate update. Data will be pushed whenever there is a block trade.
func (ok *Okx) PublicBlockTradesSubscription(ctx context.Context, operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(ctx, operation, channelPublicBlockTrades, assetType, pair, false, true, false)
}

// Websocket Spread Trade methods

// handleIncomingData extracts the incoming data to the dataHolder interface after few checks and return nil or return error message otherwise
func (ok *Okx) handleIncomingData(data *wsIncomingData, dataHolder any) error {
	if data.StatusCode == "0" || data.StatusCode == "1" {
		err := data.copyResponseToInterface(dataHolder)
		if err != nil {
			return err
		}
		if dataHolder == nil {
			return fmt.Errorf("%w, invalid incoming data", common.ErrNoResponse)
		}
		return nil
	}
	return fmt.Errorf("error code:%s error message: %v", data.StatusCode, ErrorCodes[data.StatusCode])
}

// WsPlaceSpreadOrder places a spread order thought the websocket connection stream, and returns a SubmitResponse and error message.
func (ok *Okx) WsPlaceSpreadOrder(ctx context.Context, arg *SpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil || *arg == (SpreadOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	err := ok.validatePlaceSpreadOrderParam(arg)
	if err != nil {
		return nil, err
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []SpreadOrderParam{*arg},
		Operation: okxSpreadOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.UnAuth, input)
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
			if data.Operation == okxSpreadOrder && data.ID == input.ID {
				var dataHolder *SpreadOrderResponse
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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

// WsAmandSpreadOrder amends incomplete spread order through the websocket channel.
func (ok *Okx) WsAmandSpreadOrder(ctx context.Context, arg *AmendSpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil || *arg == (AmendSpreadOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewPrice == 0 && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []AmendSpreadOrderParam{*arg},
		Operation: okxSpreadAmendOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.UnAuth, input)
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
			if data.Operation == okxSpreadAmendOrder && data.ID == input.ID {
				var dataHolder *SpreadOrderResponse
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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

// WsCancelSpreadOrder cancels an incomplete spread order through the websocket connection.
func (ok *Okx) WsCancelSpreadOrder(ctx context.Context, orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	arg := make(map[string]string)
	if orderID != "" {
		arg["ordId"] = orderID
	}
	if clientOrderID != "" {
		arg["clOrdId"] = clientOrderID
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []map[string]string{arg},
		Operation: okxSpreadCancelOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.UnAuth, input)
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
			if data.Operation == okxSpreadCancelOrder && data.ID == input.ID {
				var dataHolder *SpreadOrderResponse
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return nil, err
				}
				if data.StatusCode == "1" {
					return nil, fmt.Errorf("error code:%s message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder, nil
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

// WsCancelAllSpreadOrders cancels all spread orders and return success message through the websocket channel.
func (ok *Okx) WsCancelAllSpreadOrders(ctx context.Context, spreadID string) (bool, error) {
	if !ok.AreCredentialsValid(ctx) || !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return false, errWebsocketStreamNotAuthenticated
	}
	arg := make(map[string]string, 1)
	if spreadID != "" {
		arg["sprdId"] = spreadID
	}
	randomID, err := common.GenerateRandomString(32, common.SmallLetters, common.CapitalLetters, common.NumberCharacters)
	if err != nil {
		return false, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []map[string]string{arg},
		Operation: okxSpreadCancelAllOrders,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(ctx, request.UnAuth, input)
	if err != nil {
		return false, err
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
			if data.Operation == okxSpreadCancelAllOrders && data.ID == input.ID {
				dataHolder := &ResponseSuccess{}
				err = ok.handleIncomingData(data, &dataHolder)
				if err != nil {
					return false, err
				}
				if data.StatusCode == "1" {
					return false, fmt.Errorf("error code:%s message: %s", dataHolder.StatusCode, dataHolder.StatusMessage)
				}
				return dataHolder.Result, nil
			}
			continue
		case <-timer.C:
			timer.Stop()
			return false, fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				input.Operation)
		}
	}
}

// channelName converts global subscription channel names to exchange specific names
func channelName(s *subscription.Subscription) string {
	if s, ok := subscriptionNames[s.Channel]; ok {
		return s
	}
	return s.Channel
}

// isAssetChannel returns if the channel expects one Asset per subscription
func isAssetChannel(s *subscription.Subscription) bool {
	return s.Channel == subscription.MyOrdersChannel
}

// isSymbolChannel returns if the channel expects one Symbol per subscription
func isSymbolChannel(s *subscription.Subscription) bool {
	switch s.Channel {
	case subscription.CandlesChannel, subscription.TickerChannel, subscription.OrderbookChannel, subscription.AllTradesChannel, channelFundingRate:
		return true
	}
	return false
}

const subTplText = `
{{- with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if isAssetChannel $.S -}}
			{"channel":"{{ $name }}","instType":"{{ instType $asset }}"}
		{{- else if isSymbolChannel $.S }}
			{{- range $p := $pairs -}}
				{"channel":"{{ $name }}","instID":"{{ $p }}"}
				{{ $.PairSeparator }}
			{{- end -}}
		{{- else }}
			{"channel":"{{ $name }}"
			{{- with $algoId := index $.S.Params "algoId" -}} ,"algoId":"{{ $algoId }}" {{- end -}}
			}
		{{- end }}
	{{- $.AssetSeparator }}
	{{- end }}
{{- end }}
`
