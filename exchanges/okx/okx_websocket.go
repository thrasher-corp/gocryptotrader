package okx

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	pingMsg = []byte("ping")
	pongMsg = []byte("pong")

	// See: https://www.okx.com/docs-v5/en/#error-code-websocket-public
	authConnErrorCodes = []string{
		"60007", "60022", "60023", "60024", "60026", "63999", "60032", "60011", "60009",
		"60005", "60021", "60031", "50110",
	}
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
func (ex *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !ex.Websocket.IsEnabled() || !ex.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	err := ex.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	ex.Websocket.Wg.Add(1)
	go ex.wsReadData(ctx, ex.Websocket.Conn)
	if ex.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ex.Websocket.GetWebsocketURL())
	}
	ex.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	if ex.Websocket.CanUseAuthenticatedEndpoints() {
		err = ex.WsAuth(ctx)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ex *Exchange) WsAuth(ctx context.Context) error {
	if !ex.AreCredentialsValid(ctx) || !ex.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ex.Name)
	}
	creds, err := ex.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var dialer gws.Dialer
	err = ex.Websocket.AuthConn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	ex.Websocket.Wg.Add(1)
	go ex.wsReadData(ctx, ex.Websocket.AuthConn)
	ex.Websocket.AuthConn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ts := time.Now().Unix()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(ts, 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}

	args := []WebsocketLoginData{
		{
			APIKey:     creds.Key,
			Passphrase: creds.ClientID,
			Timestamp:  ts,
			Sign:       base64.StdEncoding.EncodeToString(hmac),
		},
	}

	return ex.SendAuthenticatedWebsocketRequest(ctx, request.Unset, "login-response", operationLogin, args, nil)
}

// wsReadData sends msgs from public and auth websockets to data handler
func (ex *Exchange) wsReadData(ctx context.Context, ws websocket.Connection) {
	defer ex.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := ex.WsHandleData(ctx, resp.Raw); err != nil {
			ex.Websocket.DataHandler <- err
		}
	}
}

// Subscribe sends a websocket subscription request to several channels to receive data.
func (ex *Exchange) Subscribe(channelsToSubscribe subscription.List) error {
	ctx := context.TODO()
	return ex.handleSubscription(ctx, operationSubscribe, channelsToSubscribe)
}

// Unsubscribe sends a websocket unsubscription request to several channels to receive data.
func (ex *Exchange) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	ctx := context.TODO()
	return ex.handleSubscription(ctx, operationUnsubscribe, channelsToUnsubscribe)
}

// handleSubscription sends a subscription and unsubscription information thought the websocket endpoint.
// as of the okx, exchange this endpoint sends subscription and unsubscription messages but with a list of json objects.
func (ex *Exchange) handleSubscription(ctx context.Context, operation string, subs subscription.List) error {
	reqs := WSSubscriptionInformationList{Operation: operation}
	authRequests := WSSubscriptionInformationList{Operation: operation}
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
				err = ex.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, authRequests)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
					err = ex.Websocket.RemoveSubscriptions(ex.Websocket.AuthConn, channels...)
				} else {
					err = ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.AuthConn, channels...)
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
				err = ex.Websocket.Conn.SendJSONMessage(ctx, request.Unset, reqs)
				if err != nil {
					return err
				}
				if operation == operationUnsubscribe {
					err = ex.Websocket.RemoveSubscriptions(ex.Websocket.Conn, channels...)
				} else {
					err = ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, channels...)
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
		if err := ex.Websocket.Conn.SendJSONMessage(ctx, request.Unset, reqs); err != nil {
			return err
		}
	}

	if len(authRequests.Arguments) > 0 && ex.Websocket.CanUseAuthenticatedEndpoints() {
		if err := ex.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, authRequests); err != nil {
			return err
		}
	}

	channels = append(channels, authChannels...)
	if operation == operationUnsubscribe {
		return ex.Websocket.RemoveSubscriptions(ex.Websocket.Conn, channels...)
	}
	return ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, channels...)
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (ex *Exchange) WsHandleData(ctx context.Context, respRaw []byte) error {
	if id, _ := jsonparser.GetString(respRaw, "id"); id != "" {
		return ex.Websocket.Match.RequireMatchWithData(id, respRaw)
	}

	var resp wsIncomingData
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		if bytes.Equal(respRaw, pongMsg) {
			return nil
		}
		return fmt.Errorf("%w unmarshalling %v", err, respRaw)
	}
	if resp.Event == operationLogin || (resp.Event == "error" && slices.Contains(authConnErrorCodes, resp.StatusCode)) {
		return ex.Websocket.Match.RequireMatchWithData("login-response", respRaw)
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
		return ex.wsProcessCandles(respRaw)
	case channelIndexCandle1Y, channelIndexCandle6M, channelIndexCandle3M, channelIndexCandle1M,
		channelIndexCandle1W, channelIndexCandle1D, channelIndexCandle2D, channelIndexCandle3D,
		channelIndexCandle5D, channelIndexCandle12H, channelIndexCandle6H, channelIndexCandle4H,
		channelIndexCandle2H, channelIndexCandle1H, channelIndexCandle30m, channelIndexCandle15m,
		channelIndexCandle5m, channelIndexCandle3m, channelIndexCandle1m, channelIndexCandle1Yutc,
		channelIndexCandle3Mutc, channelIndexCandle1Mutc, channelIndexCandle1Wutc,
		channelIndexCandle1Dutc, channelIndexCandle2Dutc, channelIndexCandle3Dutc, channelIndexCandle5Dutc,
		channelIndexCandle12Hutc, channelIndexCandle6Hutc:
		return ex.wsProcessIndexCandles(respRaw)
	case channelTickers:
		return ex.wsProcessTickers(respRaw)
	case channelIndexTickers:
		var response WsIndexTicker
		return ex.wsProcessPushData(respRaw, &response)
	case channelStatus:
		var response WsSystemStatusResponse
		return ex.wsProcessPushData(respRaw, &response)
	case channelPublicStrucBlockTrades:
		var response WsPublicTradesResponse
		return ex.wsProcessPushData(respRaw, &response)
	case channelPublicBlockTrades:
		return ex.wsProcessBlockPublicTrades(respRaw)
	case channelBlockTickers:
		var response WsBlockTicker
		return ex.wsProcessPushData(respRaw, &response)
	case channelAccountGreeks:
		var response WsGreeks
		return ex.wsProcessPushData(respRaw, &response)
	case channelAccount:
		var response WsAccountChannelPushData
		return ex.wsProcessPushData(respRaw, &response)
	case channelPositions,
		channelLiquidationWarning:
		var response WsPositionResponse
		return ex.wsProcessPushData(respRaw, &response)
	case channelBalanceAndPosition:
		return ex.wsProcessBalanceAndPosition(ctx, respRaw)
	case channelOrders:
		return ex.wsProcessOrders(respRaw)
	case channelAlgoOrders:
		var response WsAlgoOrder
		return ex.wsProcessPushData(respRaw, &response)
	case channelAlgoAdvance:
		var response WsAdvancedAlgoOrder
		return ex.wsProcessPushData(respRaw, &response)
	case channelRFQs:
		var response WsRFQ
		return ex.wsProcessPushData(respRaw, &response)
	case channelQuotes:
		var response WsQuote
		return ex.wsProcessPushData(respRaw, &response)
	case channelStructureBlockTrades:
		var response WsStructureBlocTrade
		return ex.wsProcessPushData(respRaw, &response)
	case channelSpotGridOrder:
		var response WsSpotGridAlgoOrder
		return ex.wsProcessPushData(respRaw, &response)
	case channelGridOrdersContract:
		var response WsContractGridAlgoOrder
		return ex.wsProcessPushData(respRaw, &response)
	case channelGridPositions:
		var response WsContractGridAlgoOrder
		return ex.wsProcessPushData(respRaw, &response)
	case channelGridSubOrders:
		var response WsGridSubOrderData
		return ex.wsProcessPushData(respRaw, &response)
	case channelInstruments:
		var response WSInstrumentResponse
		return ex.wsProcessPushData(respRaw, &response)
	case channelOpenInterest:
		var response WSOpenInterestResponse
		return ex.wsProcessPushData(respRaw, &response)
	case channelTrades, channelAllTrades:
		return ex.wsProcessTrades(respRaw)
	case channelEstimatedPrice:
		var response WsDeliveryEstimatedPrice
		return ex.wsProcessPushData(respRaw, &response)
	case channelMarkPrice, channelPriceLimit:
		var response WsMarkPrice
		return ex.wsProcessPushData(respRaw, &response)
	case channelOrderBooks5:
		return ex.wsProcessOrderbook5(respRaw)
	case okxSpreadOrderbookLevel1,
		okxSpreadOrderbook:
		return ex.wsProcessSpreadOrderbook(respRaw)
	case okxSpreadPublicTrades:
		return ex.wsProcessPublicSpreadTrades(respRaw)
	case okxSpreadPublicTicker:
		return ex.wsProcessPublicSpreadTicker(respRaw)
	case channelOrderBooks,
		channelOrderBooks50TBT,
		channelBBOTBT,
		channelOrderBooksTBT:
		return ex.wsProcessOrderBooks(respRaw)
	case channelOptionTrades:
		return ex.wsProcessOptionTrades(respRaw)
	case channelOptSummary:
		var response WsOptionSummary
		return ex.wsProcessPushData(respRaw, &response)
	case channelFundingRate:
		var response WsFundingRate
		return ex.wsProcessPushData(respRaw, &response)
	case channelMarkPriceCandle1Y, channelMarkPriceCandle6M, channelMarkPriceCandle3M, channelMarkPriceCandle1M,
		channelMarkPriceCandle1W, channelMarkPriceCandle1D, channelMarkPriceCandle2D, channelMarkPriceCandle3D,
		channelMarkPriceCandle5D, channelMarkPriceCandle12H, channelMarkPriceCandle6H, channelMarkPriceCandle4H,
		channelMarkPriceCandle2H, channelMarkPriceCandle1H, channelMarkPriceCandle30m, channelMarkPriceCandle15m,
		channelMarkPriceCandle5m, channelMarkPriceCandle3m, channelMarkPriceCandle1m, channelMarkPriceCandle1Yutc,
		channelMarkPriceCandle3Mutc, channelMarkPriceCandle1Mutc, channelMarkPriceCandle1Wutc, channelMarkPriceCandle1Dutc,
		channelMarkPriceCandle2Dutc, channelMarkPriceCandle3Dutc, channelMarkPriceCandle5Dutc, channelMarkPriceCandle12Hutc,
		channelMarkPriceCandle6Hutc:
		return ex.wsHandleMarkPriceCandles(respRaw)
	case okxSpreadOrders:
		return ex.wsProcessSpreadOrders(respRaw)
	case okxSpreadTrades:
		return ex.wsProcessSpreadTrades(respRaw)
	case okxWithdrawalInfo:
		resp := &struct {
			Arguments SubscriptionInfo `json:"arg"`
			Data      []WsDepositInfo  `json:"data"`
		}{}
		return ex.wsProcessPushData(respRaw, resp)
	case okxDepositInfo:
		resp := &struct {
			Arguments SubscriptionInfo  `json:"arg"`
			Data      []WsWithdrawlInfo `json:"data"`
		}{}
		return ex.wsProcessPushData(respRaw, resp)
	case channelRecurringBuy:
		resp := &struct {
			Arguments SubscriptionInfo    `json:"arg"`
			Data      []RecurringBuyOrder `json:"data"`
		}{}
		return ex.wsProcessPushData(respRaw, resp)
	case liquidationOrders:
		var resp *LiquidationOrder
		return ex.wsProcessPushData(respRaw, &resp)
	case adlWarning:
		var resp ADLWarning
		return ex.wsProcessPushData(respRaw, &resp)
	case economicCalendar:
		var resp EconomicCalendarResponse
		return ex.wsProcessPushData(respRaw, &resp)
	case copyTrading:
		var resp CopyTradingNotification
		return ex.wsProcessPushData(respRaw, &resp)
	default:
		ex.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + string(respRaw)}
		return nil
	}
}

// wsProcessSpreadTrades handle and process spread order trades
func (ex *Exchange) wsProcessSpreadTrades(respRaw []byte) error {
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
	pair, err := ex.GetPairFromInstrumentID(resp.Argument.SpreadID)
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
			Exchange:     ex.Name,
			Side:         oSide,
			Timestamp:    resp.Data[x].Timestamp.Time(),
			TID:          resp.Data[x].TradeID,
			Price:        resp.Data[x].FillPrice.Float64(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

// wsProcessSpreadOrders retrieve order information from the sprd-order Websocket channel.
// Data will not be pushed when first subscribed.
// Data will only be pushed when triggered by events such as placing/canceling order.
func (ex *Exchange) wsProcessSpreadOrders(respRaw []byte) error {
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
	pair, err := ex.GetPairFromInstrumentID(resp.Argument.SpreadID)
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
			Exchange:             ex.Name,
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
	ex.Websocket.DataHandler <- orderDetails
	return nil
}

// wsProcessIndexCandles processes index candlestick data
func (ex *Exchange) wsProcessIndexCandles(respRaw []byte) error {
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

	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ex.getAssetsFromInstrumentID(response.Argument.InstrumentID.String())
		if err != nil {
			return err
		}
	}
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		candlesData := response.Data[i]
		myCandle := websocket.KlineData{
			Pair:       response.Argument.InstrumentID,
			Exchange:   ex.Name,
			Timestamp:  time.UnixMilli(candlesData[0].Int64()),
			Interval:   candleInterval,
			OpenPrice:  candlesData[1].Float64(),
			HighPrice:  candlesData[2].Float64(),
			LowPrice:   candlesData[3].Float64(),
			ClosePrice: candlesData[4].Float64(),
		}
		for i := range assets {
			myCandle.AssetType = assets[i]
			ex.Websocket.DataHandler <- myCandle
		}
	}
	return nil
}

// wsProcessPublicSpreadTicker process spread order ticker push data.
func (ex *Exchange) wsProcessPublicSpreadTicker(respRaw []byte) error {
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
			ExchangeName: ex.Name,
			AssetType:    asset.Spread,
			LastUpdated:  data[x].Timestamp.Time(),
		}
	}
	ex.Websocket.DataHandler <- tickers
	return nil
}

// wsProcessPublicSpreadTrades retrieve the recent trades data from sprd-public-trades.
// Data will be pushed whenever there is a trade.
// Every update contains only one trade.
func (ex *Exchange) wsProcessPublicSpreadTrades(respRaw []byte) error {
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
			Exchange:     ex.Name,
			CurrencyPair: pair,
			AssetType:    asset.Spread,
			Side:         oSide,
			Price:        data[x].Price.Float64(),
			Amount:       data[x].Size.Float64(),
			Timestamp:    data[x].Timestamp.Time(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

// wsProcessSpreadOrderbook process spread orderbook data.
func (ex *Exchange) wsProcessSpreadOrderbook(respRaw []byte) error {
	var resp WsSpreadOrderbook
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	pair, err := ex.GetPairFromInstrumentID(resp.Arg.SpreadID)
	if err != nil {
		return err
	}
	extractedResponse, err := resp.ExtractSpreadOrder()
	if err != nil {
		return err
	}
	for x := range extractedResponse.Data {
		err = ex.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Asset:             asset.Spread,
			Asks:              extractedResponse.Data[x].Asks,
			Bids:              extractedResponse.Data[x].Bids,
			LastUpdated:       resp.Data[x].Timestamp.Time(),
			Pair:              pair,
			Exchange:          ex.Name,
			ValidateOrderbook: ex.ValidateOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// wsProcessOrderbook5 processes orderbook data
func (ex *Exchange) wsProcessOrderbook5(data []byte) error {
	var resp WsOrderbook5
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	if len(resp.Data) != 1 {
		return fmt.Errorf("%s - no data returned", ex.Name)
	}

	assets, err := ex.getAssetsFromInstrumentID(resp.Argument.InstrumentID.String())
	if err != nil {
		return err
	}

	asks := make([]orderbook.Level, len(resp.Data[0].Asks))
	for x := range resp.Data[0].Asks {
		asks[x].Price = resp.Data[0].Asks[x][0].Float64()
		asks[x].Amount = resp.Data[0].Asks[x][1].Float64()
	}

	bids := make([]orderbook.Level, len(resp.Data[0].Bids))
	for x := range resp.Data[0].Bids {
		bids[x].Price = resp.Data[0].Bids[x][0].Float64()
		bids[x].Amount = resp.Data[0].Bids[x][1].Float64()
	}

	for x := range assets {
		err = ex.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Asset:             assets[x],
			Asks:              asks,
			Bids:              bids,
			LastUpdated:       resp.Data[0].Timestamp.Time(),
			Pair:              resp.Argument.InstrumentID,
			Exchange:          ex.Name,
			ValidateOrderbook: ex.ValidateOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// wsProcessOptionTrades handles options trade data
func (ex *Exchange) wsProcessOptionTrades(data []byte) error {
	var resp WsOptionTrades
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		var pair currency.Pair
		pair, err = ex.GetPairFromInstrumentID(resp.Data[i].InstrumentID)
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
			Exchange:     ex.Name,
			Side:         oSide,
			Timestamp:    resp.Data[i].Timestamp.Time(),
			TID:          resp.Data[i].TradeID,
			Price:        resp.Data[i].Price.Float64(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

// wsProcessOrderBooks processes "snapshot" and "update" order book
func (ex *Exchange) wsProcessOrderBooks(data []byte) error {
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
		assets, err = ex.getAssetsFromInstrumentID(response.Argument.InstrumentID.String())
		if err != nil {
			return err
		}
	}
	if !response.Argument.InstrumentID.IsPopulated() {
		return currency.ErrCurrencyPairsEmpty
	}
	response.Argument.InstrumentID.Delimiter = currency.DashDelimiter
	for i := range response.Data {
		if response.Action == wsOrderbookSnapshot {
			err = ex.WsProcessSnapshotOrderBook(&response.Data[i], response.Argument.InstrumentID, assets)
		} else {
			if len(response.Data[i].Asks) == 0 && len(response.Data[i].Bids) == 0 {
				return nil
			}
			err = ex.WsProcessUpdateOrderbook(&response.Data[i], response.Argument.InstrumentID, assets)
		}
		if err != nil {
			if errors.Is(err, errInvalidChecksum) {
				err = ex.Subscribe(subscription.List{
					{
						Channel: response.Argument.Channel,
						Asset:   assets[0],
						Pairs:   currency.Pairs{response.Argument.InstrumentID},
					},
				})
				if err != nil {
					ex.Websocket.DataHandler <- err
				}
			} else {
				return err
			}
		}
	}
	if ex.Verbose {
		log.Debugf(log.ExchangeSys, "%s passed checksum for pair %s", ex.Name, response.Argument.InstrumentID)
	}
	return nil
}

// WsProcessSnapshotOrderBook processes snapshot order books
func (ex *Exchange) WsProcessSnapshotOrderBook(data *WsOrderBookData, pair currency.Pair, assets []asset.Item) error {
	signedChecksum, err := ex.CalculateOrderbookChecksum(data)
	if err != nil {
		return fmt.Errorf("%w %v: unable to calculate orderbook checksum: %s",
			errInvalidChecksum,
			pair,
			err)
	}
	if signedChecksum != uint32(data.Checksum) { //nolint:gosec // Requires type casting
		return fmt.Errorf("%w %v",
			errInvalidChecksum,
			pair)
	}

	asks, err := ex.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	bids, err := ex.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	for i := range assets {
		newOrderBook := orderbook.Book{
			Asset:             assets[i],
			Asks:              asks,
			Bids:              bids,
			LastUpdated:       data.Timestamp.Time(),
			Pair:              pair,
			Exchange:          ex.Name,
			ValidateOrderbook: ex.ValidateOrderbook,
		}
		err = ex.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
		if err != nil {
			return err
		}
	}
	return nil
}

// WsProcessUpdateOrderbook updates an existing orderbook using websocket data
// After merging WS data, it will sort, validate and finally update the existing
// orderbook
func (ex *Exchange) WsProcessUpdateOrderbook(data *WsOrderBookData, pair currency.Pair, assets []asset.Item) error {
	asks, err := ex.AppendWsOrderbookItems(data.Asks)
	if err != nil {
		return err
	}
	bids, err := ex.AppendWsOrderbookItems(data.Bids)
	if err != nil {
		return err
	}
	for i := range assets {
		if err := ex.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:             pair,
			Asset:            assets[i],
			UpdateTime:       data.Timestamp.Time(),
			GenerateChecksum: generateOrderbookChecksum,
			ExpectedChecksum: uint32(data.Checksum), //nolint:gosec // Requires type casting
			Asks:             asks,
			Bids:             bids,
		}); err != nil {
			return err
		}
	}
	return nil
}

// AppendWsOrderbookItems adds websocket orderbook data bid/asks into an orderbook item array
func (ex *Exchange) AppendWsOrderbookItems(entries [][4]types.Number) (orderbook.Levels, error) {
	items := make(orderbook.Levels, len(entries))
	for j := range entries {
		items[j] = orderbook.Level{Amount: entries[j][1].Float64(), Price: entries[j][0].Float64()}
	}
	return items, nil
}

// generateOrderbookChecksum alternates over the first 25 bid and ask
// entries of a merged orderbook. The checksum is made up of the price and the
// quantity with a semicolon (:) deliminating them. This will also work when
// there are less than 25 entries (for whatever reason)
// eg Bid:Ask:Bid:Ask:Ask:Ask
func generateOrderbookChecksum(orderbookData *orderbook.Book) uint32 {
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
	return crc32.ChecksumIEEE([]byte(checksumStr))
}

// CalculateOrderbookChecksum alternates over the first 25 bid and ask entries from websocket data.
func (ex *Exchange) CalculateOrderbookChecksum(orderbookData *WsOrderBookData) (uint32, error) {
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
	return crc32.ChecksumIEEE([]byte(checksumStr)), nil
}

// wsHandleMarkPriceCandles processes candlestick mark price push data as a result of  subscription to "mark-price-candle*" channel.
func (ex *Exchange) wsHandleMarkPriceCandles(data []byte) error {
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
	ex.Websocket.DataHandler <- candles
	return nil
}

// wsProcessTrades handles a list of trade information.
func (ex *Exchange) wsProcessTrades(data []byte) error {
	var response WsTradeOrder
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	saveTradeData := ex.IsSaveTradeDataEnabled()
	tradeFeed := ex.IsTradeFeedEnabled()
	if !saveTradeData && !tradeFeed {
		return nil
	}

	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ex.getAssetsFromInstrumentID(response.Argument.InstrumentID.String())
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
				Exchange:     ex.Name,
				Side:         response.Data[i].Side,
				Timestamp:    response.Data[i].Timestamp.Time().UTC(),
				TID:          response.Data[i].TradeID,
				Price:        response.Data[i].Price.Float64(),
			})
		}
	}
	if tradeFeed {
		for i := range trades {
			ex.Websocket.DataHandler <- trades[i]
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(trades...)
	}
	return nil
}

// wsProcessOrders handles websocket order push data responses.
func (ex *Exchange) wsProcessOrders(respRaw []byte) error {
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
			ex.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ex.Name,
				OrderID:  response.Data[x].OrderID,
				Err:      err,
			}
		}
		orderStatus, err := order.StringToOrderStatus(response.Data[x].State)
		if err != nil {
			ex.Websocket.DataHandler <- order.ClassificationError{
				Exchange: ex.Name,
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
			Exchange:             ex.Name,
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
		ex.Websocket.DataHandler <- d
	}
	return nil
}

// wsProcessCandles handler to get a list of candlestick messages.
func (ex *Exchange) wsProcessCandles(respRaw []byte) error {
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
	var assets []asset.Item
	if response.Argument.InstrumentType != "" {
		assetType, err := assetTypeFromInstrumentType(response.Argument.InstrumentType)
		if err != nil {
			return err
		}
		assets = append(assets, assetType)
	} else {
		assets, err = ex.getAssetsFromInstrumentID(response.Argument.InstrumentID.String())
		if err != nil {
			return err
		}
	}
	candleInterval := strings.TrimPrefix(response.Argument.Channel, candle)
	for i := range response.Data {
		for j := range assets {
			ex.Websocket.DataHandler <- websocket.KlineData{
				Timestamp:  time.UnixMilli(response.Data[i][0].Int64()),
				Pair:       response.Argument.InstrumentID,
				AssetType:  assets[j],
				Exchange:   ex.Name,
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
func (ex *Exchange) wsProcessTickers(data []byte) error {
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
			assets, err = ex.getAssetsFromInstrumentID(response.Argument.InstrumentID.String())
			if err != nil {
				return err
			}
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
				ExchangeName: ex.Name,
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
				Pair:         response.Data[i].InstrumentID,
				LastUpdated:  response.Data[i].TickerDataGenerationTime.Time(),
			}
			ex.Websocket.DataHandler <- tickData
		}
	}
	return nil
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (ex *Exchange) generateSubscriptions() (subscription.List, error) {
	return ex.Features.Subscriptions.ExpandTemplates(ex)
}

// GetSubscriptionTemplate returns a subscription channel template
func (ex *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":     channelName,
		"isSymbolChannel": isSymbolChannel,
		"isAssetChannel":  isAssetChannel,
		"instType":        GetInstrumentTypeFromAssetItem,
	}).Parse(subTplText)
}

// wsProcessBlockPublicTrades handles the recent block trades data by individual legs.
func (ex *Exchange) wsProcessBlockPublicTrades(data []byte) error {
	var resp PublicBlockTrades
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		var pair currency.Pair
		pair, err = ex.GetPairFromInstrumentID(resp.Data[i].InstrumentID)
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
			Exchange:     ex.Name,
			Side:         oSide,
			Timestamp:    resp.Data[i].Timestamp.Time(),
			TID:          resp.Data[i].TradeID,
			Price:        resp.Data[i].Price.Float64(),
		}
	}
	return trade.AddTradesToBuffer(trades...)
}

func (ex *Exchange) wsProcessBalanceAndPosition(ctx context.Context, data []byte) error {
	var resp WsBalanceAndPosition
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	creds, err := ex.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var changes []account.Change
	for i := range resp.Data {
		for j := range resp.Data[i].BalanceData {
			changes = append(changes, account.Change{
				AssetType: asset.Spot,
				Account:   resp.Argument.UID,
				Balance: &account.Balance{
					Currency:  currency.NewCode(resp.Data[i].BalanceData[j].Currency),
					Total:     resp.Data[i].BalanceData[j].CashBalance.Float64(),
					Free:      resp.Data[i].BalanceData[j].CashBalance.Float64(),
					UpdatedAt: resp.Data[i].BalanceData[j].UpdateTime.Time(),
				},
			})
		}
		// TODO: Handle position data
	}
	ex.Websocket.DataHandler <- changes
	return account.ProcessChange(ex.Name, changes, creds)
}

// wsProcessPushData processes push data coming through the websocket channel
func (ex *Exchange) wsProcessPushData(data []byte, resp any) error {
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	ex.Websocket.DataHandler <- resp
	return nil
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
