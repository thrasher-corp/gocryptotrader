package okx

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
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	// defaultSubscribedChannels list of channels which are subscribed by default
	defaultSubscribedChannels = []string{
		okxChannelTrades,
		okxChannelOrderBooks,
		okxChannelTickers,
	}
	// defaultAuthChannels list of channels which are subscribed when authenticated
	defaultAuthChannels = []string{
		okxChannelAccount,
		okxChannelOrders,
	}
)

var (
	candlestickChannelsMap    = map[string]bool{okxChannelCandle1Y: true, okxChannelCandle6M: true, okxChannelCandle3M: true, okxChannelCandle1M: true, okxChannelCandle1W: true, okxChannelCandle1D: true, okxChannelCandle2D: true, okxChannelCandle3D: true, okxChannelCandle5D: true, okxChannelCandle12H: true, okxChannelCandle6H: true, okxChannelCandle4H: true, okxChannelCandle2H: true, okxChannelCandle1H: true, okxChannelCandle30m: true, okxChannelCandle15m: true, okxChannelCandle5m: true, okxChannelCandle3m: true, okxChannelCandle1m: true, okxChannelCandle1Yutc: true, okxChannelCandle3Mutc: true, okxChannelCandle1Mutc: true, okxChannelCandle1Wutc: true, okxChannelCandle1Dutc: true, okxChannelCandle2Dutc: true, okxChannelCandle3Dutc: true, okxChannelCandle5Dutc: true, okxChannelCandle12Hutc: true, okxChannelCandle6Hutc: true}
	candlesticksMarkPriceMap  = map[string]bool{okxChannelMarkPriceCandle1Y: true, okxChannelMarkPriceCandle6M: true, okxChannelMarkPriceCandle3M: true, okxChannelMarkPriceCandle1M: true, okxChannelMarkPriceCandle1W: true, okxChannelMarkPriceCandle1D: true, okxChannelMarkPriceCandle2D: true, okxChannelMarkPriceCandle3D: true, okxChannelMarkPriceCandle5D: true, okxChannelMarkPriceCandle12H: true, okxChannelMarkPriceCandle6H: true, okxChannelMarkPriceCandle4H: true, okxChannelMarkPriceCandle2H: true, okxChannelMarkPriceCandle1H: true, okxChannelMarkPriceCandle30m: true, okxChannelMarkPriceCandle15m: true, okxChannelMarkPriceCandle5m: true, okxChannelMarkPriceCandle3m: true, okxChannelMarkPriceCandle1m: true, okxChannelMarkPriceCandle1Yutc: true, okxChannelMarkPriceCandle3Mutc: true, okxChannelMarkPriceCandle1Mutc: true, okxChannelMarkPriceCandle1Wutc: true, okxChannelMarkPriceCandle1Dutc: true, okxChannelMarkPriceCandle2Dutc: true, okxChannelMarkPriceCandle3Dutc: true, okxChannelMarkPriceCandle5Dutc: true, okxChannelMarkPriceCandle12Hutc: true, okxChannelMarkPriceCandle6Hutc: true}
	candlesticksIndexPriceMap = map[string]bool{okxChannelIndexCandle1Y: true, okxChannelIndexCandle6M: true, okxChannelIndexCandle3M: true, okxChannelIndexCandle1M: true, okxChannelIndexCandle1W: true, okxChannelIndexCandle1D: true, okxChannelIndexCandle2D: true, okxChannelIndexCandle3D: true, okxChannelIndexCandle5D: true, okxChannelIndexCandle12H: true, okxChannelIndexCandle6H: true, okxChannelIndexCandle4H: true, okxChannelIndexCandle2H: true, okxChannelIndexCandle1H: true, okxChannelIndexCandle30m: true, okxChannelIndexCandle15m: true, okxChannelIndexCandle5m: true, okxChannelIndexCandle3m: true, okxChannelIndexCandle1m: true, okxChannelIndexCandle1Yutc: true, okxChannelIndexCandle3Mutc: true, okxChannelIndexCandle1Mutc: true, okxChannelIndexCandle1Wutc: true, okxChannelIndexCandle1Dutc: true, okxChannelIndexCandle2Dutc: true, okxChannelIndexCandle3Dutc: true, okxChannelIndexCandle5Dutc: true, okxChannelIndexCandle12Hutc: true, okxChannelIndexCandle6Hutc: true}
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
	okxChannelTickers                = "tickers"
	okxChannelIndexTickers           = "index-tickers"
	okxChannelStatus                 = "status"
	okxChannelPublicStrucBlockTrades = "public-struc-block-trades"
	okxChannelPublicBlockTrades      = "public-block-trades"
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
	okxChannelRfqs                 = "rfqs"
	okxChannelQuotes               = "quotes"
	okxChannelStructureBlockTrades = "struc-block-trades"
	okxChannelSpotGridOrder        = "grid-orders-spot"
	okxChannelGridOrdersContract   = "grid-orders-contract"
	okxChannelMoonGridAlgoOrders   = "grid-orders-moon"
	okxChannelGridPositions        = "grid-positions"
	okxChannelGridSubOrders        = "grid-sub-orders"
	okxRecurringBuyChannel         = "algo-recurring-buy"
	okxLiquidationOrders           = "liquidation-orders"
	okxADLWarning                  = "adl-warning"
	okxEconomicCalendar            = "economic-calendar"

	// Public channels
	okxChannelInstruments     = "instruments"
	okxChannelOpenInterest    = "open-interest"
	okxChannelTrades          = "trades"
	okxChannelAllTrades       = "trades-all"
	okxChannelEstimatedPrice  = "estimated-price"
	okxChannelMarkPrice       = "mark-price"
	okxChannelPriceLimit      = "price-limit"
	okxChannelOrderBooks      = "books"
	okxChannelOptionTrades    = "option-trades"
	okxChannelOrderBooks5     = "books5"
	okxChannelOrderBooks50TBT = "books50-l2-tbt"
	okxChannelOrderBooksTBT   = "books-l2-tbt"
	okxChannelBBOTBT          = "bbo-tbt"
	okxChannelOptSummary      = "opt-summary"
	okxChannelFundingRate     = "funding-rate"

	// Websocket trade endpoint operations
	okxOpOrder             = "order"
	okxOpBatchOrders       = "batch-orders"
	okxOpCancelOrder       = "cancel-order"
	okxOpBatchCancelOrders = "batch-cancel-orders"
	okxOpAmendOrder        = "amend-order"
	okxOpBatchAmendOrders  = "batch-amend-orders"
	okxOpMassCancelOrder   = "mass-cancel"

	// Candlestick lengths
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

	// Copy trading websocket endpoints.
	okxCopyTrading = "copytrading-notification"
)

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
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
			if data.Event == operationLogin && data.Code == "0" {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return nil
			} else if data.Event == "error" &&
				(data.Code == "60022" || data.Code == "60009" || data.Code == "60004") {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
				return fmt.Errorf("%w code: %s message: %s", errWebsocketStreamNotAuthenticated, data.Code, data.Message)
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
func (ok *Okx) handleSubscription(operation string, subscriptions subscription.List) error {
	reqs := WSSubscriptionInformationList{Operation: operation}
	authRequests := WSSubscriptionInformationList{Operation: operation}
	ok.WsRequestSemaphore <- 1
	defer func() { <-ok.WsRequestSemaphore }()
	var channels subscription.List
	var authChannels subscription.List
	for i := 0; i < len(subscriptions); i++ {
		s := subscriptions[i]
		var (
			instrumentIDs, underlyings []string
			instrumentType,
			instrumentFamily, algoID, uid string
			authSubscription bool
		)

		switch s.Channel {
		case okxChannelAccount,
			okxChannelPositions,
			okxChannelBalanceAndPosition,
			okxChannelOrders,
			okxChannelAlgoOrders,
			okxChannelAlgoAdvance,
			okxChannelLiquidationWarning,
			okxChannelAccountGreeks,
			okxChannelRfqs,
			okxChannelQuotes,
			okxChannelStructureBlockTrades,
			okxChannelSpotGridOrder,
			okxChannelGridOrdersContract,
			okxChannelMoonGridAlgoOrders,
			okxChannelGridPositions,
			okxChannelGridSubOrders,
			okxRecurringBuyChannel,
			okxDepositInfo,
			okxLiquidationOrders,
			okxADLWarning,
			okxWithdrawalInfo,
			okxEconomicCalendar,
			okxCopyTrading:
			authSubscription = true
		}

		switch s.Channel {
		case okxChannelGridPositions,
			okxRecurringBuyChannel:
			algoID, _ = subscriptions[i].Params["algoId"].(string)
		}

		switch s.Channel {
		case okxChannelGridSubOrders,
			okxChannelGridPositions:
			uid, _ = subscriptions[i].Params["uid"].(string)
		}

		if strings.HasPrefix(s.Channel, "candle") ||
			s.Channel == okxChannelTickers ||
			s.Channel == okxChannelOrderBooks ||
			s.Channel == okxChannelOrderBooks5 ||
			s.Channel == okxChannelOrderBooks50TBT ||
			s.Channel == okxChannelOrderBooksTBT ||
			s.Channel == okxChannelFundingRate ||
			s.Channel == okxChannelAllTrades ||
			s.Channel == okxChannelTrades ||
			s.Channel == okxChannelOptionTrades ||
			s.Channel == okxCopyTrading {
			if len(s.Pairs) == 0 {
				return fmt.Errorf("%w, for channel %q for asset: %s", currency.ErrCurrencyPairsEmpty, s.Channel, s.Asset.String())
			}
			format, err := ok.GetPairFormat(s.Asset, false)
			if err != nil {
				return err
			}
			instrumentIDs = make([]string, len(s.Pairs))
			for p := range s.Pairs {
				if s.Pairs[p].Base.String() == "" || s.Pairs[p].Quote.String() == "" {
					return fmt.Errorf("%w, for channel %q for asset: %s", currency.ErrCurrencyPairsEmpty, s.Channel, s.Asset.String())
				}
				instrumentIDs[p] = format.Format(s.Pairs[p])
			}
		}
		switch s.Channel {
		case okxChannelInstruments, okxChannelPositions, okxChannelOrders, okxChannelAlgoOrders,
			okxChannelAlgoAdvance, okxChannelLiquidationWarning, okxChannelSpotGridOrder,
			okxChannelGridOrdersContract, okxChannelMoonGridAlgoOrders, okxChannelEstimatedPrice,
			okxADLWarning, okxLiquidationOrders, okxRecurringBuyChannel, okxCopyTrading:
			instrumentType = ok.GetInstrumentTypeFromAssetItem(subscriptions[i].Asset)
		}
		switch s.Channel {
		case okxChannelOptionTrades, okxADLWarning:
			instrumentFamily, _ = subscriptions[i].Params["instFamily"].(string)
		}

		switch s.Channel {
		case okxChannelPositions, okxChannelOrders, okxChannelAlgoOrders,
			okxChannelEstimatedPrice, okxChannelOptSummary:
			var underlying string
			for p := range s.Pairs {
				underlying, _ = ok.GetUnderlying(s.Pairs[p], subscriptions[i].Asset)
				underlyings = append(underlyings, underlying)
			}
		}
		args := []SubscriptionInfo{}
		if len(instrumentIDs) > 0 {
			for iid := range instrumentIDs {
				args = append(args, SubscriptionInfo{
					Channel:          s.Channel,
					InstrumentType:   instrumentType,
					UID:              uid,
					AlgoID:           algoID,
					InstrumentID:     instrumentIDs[iid],
					InstrumentFamily: instrumentFamily,
				})
			}
			if len(underlyings) > 0 {
				if len(underlyings) != len(instrumentIDs) {
					return fmt.Errorf("%w, instrument IDs and underlyings length is not equal", errLengthMismatch)
				}
				for uliID := range underlyings {
					args[uliID].Underlying = underlyings[uliID]
				}
			}
		} else {
			args = append(args, SubscriptionInfo{
				Channel:          s.Channel,
				InstrumentType:   instrumentType,
				UID:              uid,
				AlgoID:           algoID,
				InstrumentFamily: instrumentFamily,
			})
		}

		if authSubscription {
			authChannels = append(authChannels, s)
			authRequests.Arguments = append(authRequests.Arguments, args...)
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
			reqs.Arguments = append(reqs.Arguments, args...)
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
	case okxChannelPublicBlockTrades:
		return ok.wsProcessBlockPublicTrades(respRaw)
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
	case okxChannelRfqs:
		var response WsRfq
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
	case okxChannelGridSubOrders:
		var response WsGridSubOrderData
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelInstruments:
		var response WSInstrumentResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelOpenInterest:
		var response WSOpenInterestResponse
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelTrades,
		okxChannelAllTrades:
		return ok.wsProcessTrades(respRaw)
	case okxChannelEstimatedPrice:
		var response WsDeliveryEstimatedPrice
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelMarkPrice,
		okxChannelPriceLimit:
		var response WsMarkPrice
		return ok.wsProcessPushData(respRaw, &response)
	case okxChannelOrderBooks5:
		return ok.wsProcessOrderbook5(respRaw)
	case okxSpreadOrderbookLevel1,
		okxSpreadOrderbook:
		return ok.wsProcessSpreadOrderbook(respRaw)
	case okxSpreadPublicTrades:
		return ok.wsProcessPublicSpreadTrades(respRaw)
	case okxSpreadPublicTicker:
		return ok.wsProcessPublicSpreadTicker(respRaw)
	case okxChannelOrderBooks,
		okxChannelOrderBooks50TBT,
		okxChannelBBOTBT,
		okxChannelOrderBooksTBT:
		return ok.wsProcessOrderBooks(respRaw)
	case okxChannelOptionTrades:
		return ok.wsProcessOptionTrades(respRaw)
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
	case okxRecurringBuyChannel:
		resp := &struct {
			Arguments SubscriptionInfo    `json:"arg"`
			Data      []RecurringBuyOrder `json:"data"`
		}{}
		return ok.wsProcessPushData(respRaw, resp)
	case okxLiquidationOrders:
		var resp LiquidiationOrder
		return ok.wsProcessPushData(respRaw, &resp)
	case okxADLWarning:
		var resp ADLWarning
		return ok.wsProcessPushData(respRaw, &resp)
	case okxEconomicCalendar:
		var resp EconomicCalendarResponse
		return ok.wsProcessPushData(respRaw, &resp)
	case okxCopyTrading:
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
	pair, err := ok.GetPairFromInstrumentID(resp.Argument.InstrumentID)
	if err != nil {
		return err
	}

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
		d := &order.Detail{
			Amount:               resp.Data[x].Size.Float64(),
			AverageExecutedPrice: resp.Data[x].AvgPrice.Float64(),
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
		ok.Websocket.DataHandler <- d
	}
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
	assets, err := ok.GetAssetsFromInstrumentTypeOrID(response.Argument.InstrumentType, response.Argument.InstrumentID)
	if err != nil {
		return err
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
	resp.Data = data
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

	assets, err := ok.GetAssetsFromInstrumentTypeOrID("", resp.Argument.InstrumentID)
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
	if response.Argument.Channel == okxChannelOrderBooks &&
		response.Action != wsOrderbookUpdate &&
		response.Action != wsOrderbookSnapshot {
		return fmt.Errorf("%w, %s", orderbook.ErrInvalidAction, response.Action)
	}
	assets, err := ok.GetAssetsFromInstrumentTypeOrID(response.Argument.InstrumentType, response.Argument.InstrumentID)
	if err != nil {
		return err
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
	assets, err := ok.GetAssetsFromInstrumentTypeOrID(response.Argument.InstrumentType, response.Argument.InstrumentID)
	if err != nil {
		return err
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
	a := GetAssetTypeFromInstrumentType(response.Argument.InstrumentType)
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
	assets, err = ok.GetAssetsFromInstrumentTypeOrID(response.Argument.InstrumentType, response.Argument.InstrumentID)
	if err != nil {
		return err
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
		assets, err = ok.GetAssetsFromInstrumentTypeOrID(response.Argument.InstrumentType, response.Data[i].InstrumentID)
		if err != nil {
			return err
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

// GenerateDefaultSubscriptions returns a list of default subscription message.
func (ok *Okx) GenerateDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	assets := ok.GetAssetTypes(true)
	if assets.Contains(asset.Spread) {
		for a := range assets {
			if assets[a] == asset.Spread {
				if a == len(assets)-1 {
					assets = assets[:len(assets)-1]
				} else {
					assets = append(assets[:a], assets[a+1:]...)
				}
				break
			}
		}
	}
	subs := make([]string, 0, len(defaultSubscribedChannels)+len(defaultAuthChannels))
	subs = append(subs, defaultSubscribedChannels...)
	if ok.Websocket.CanUseAuthenticatedEndpoints() {
		subs = append(subs, defaultAuthChannels...)
	}
	for c := range subs {
		switch subs[c] {
		case okxChannelOrders:
			for x := range assets {
				enabledPairs, err := ok.GetEnabledPairs(assets[x])
				if err != nil {
					return nil, err
				}
				subscriptions = append(subscriptions, &subscription.Subscription{
					Channel: subs[c],
					Asset:   assets[x],
					Pairs:   enabledPairs,
				})
			}
		case okxChannelCandle5m, okxChannelTickers, okxChannelOrderBooks,
			okxChannelFundingRate, okxChannelOrderBooks5, okxChannelOrderBooks50TBT,
			okxChannelOrderBooksTBT, okxChannelTrades:
			for x := range assets {
				enabledPairs, err := ok.GetEnabledPairs(assets[x])
				if err != nil {
					return nil, err
				}
				subscriptions = append(subscriptions, &subscription.Subscription{
					Channel: subs[c],
					Asset:   assets[x],
					Pairs:   enabledPairs,
				})
			}
		case okxChannelOptionTrades:
			enabledPairs, err := ok.GetEnabledPairs(asset.Options)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: subs[c],
				Asset:   asset.Options,
				Pairs:   enabledPairs,
			})
		case okxCopyTrading:
			enabledPairs, err := ok.GetEnabledPairs(asset.PerpetualSwap)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: subs[c],
				Asset:   asset.PerpetualSwap,
				Pairs:   enabledPairs,
			})
		default:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: subs[c],
			})
		}
	}
	return subscriptions, nil
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
func (ok *Okx) WsPlaceOrder(arg *PlaceOrderRequestParam) (*OrderData, error) {
	if arg == nil || *arg == (PlaceOrderRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	err := ok.validatePlaceOrderParams(arg)
	if err != nil {
		return nil, err
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
				if data.Code == "0" || data.Code == "2" {
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
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketStreamNotAuthenticated
	}
	randomID, err := common.GenerateRandomString(4, common.NumberCharacters)
	if err != nil {
		return nil, err
	}
	input := WsOperationInput{
		ID:        randomID,
		Arguments: []CancelOrderRequestParam{arg},
		Operation: okxOpCancelOrder,
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), cancelOrderEPL, input)
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
		if arg.OrderID == "" && arg.ClientOrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
				if data.Code == "0" || data.Code == "2" {
					var resp *WsPlaceOrderResponse
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
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), amendOrderEPL, input)
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), amendMultipleOrdersEPL, input)
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
					var resp *WsPlaceOrderResponse
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

// WsMassCancelOrders cancel all the MMP pending orders of an instrument family.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) WsMassCancelOrders(args []CancelMassReqParam) (bool, error) {
	for x := range args {
		if args[x].InstrumentType == "" {
			return false, errInstrumentTypeRequired
		}
		if args[x].InstrumentFamily == "" {
			return false, errInstrumentFamilyRequired
		}
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.Background(), request.Unset, input)
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
				if data.Code == "0" || data.Code == "2" {
					resp := []struct {
						Result bool `json:"result"`
					}{}
					err := json.Unmarshal(data.Data, &resp)
					if err != nil {
						return false, err
					}
					if len(data.Data) == 0 {
						return false, fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
					}
					return resp[0].Result, nil
				}
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
					(msg.Code == "60009" || msg.Code == "60004" || msg.Code == "60022" || msg.Code == "0") &&
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
		instrumentType = ok.GetInstrumentTypeFromAssetItem(assetType)
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
	return ok.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, input)
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
	if params.InstrumentType {
		instrumentType = ok.GetInstrumentTypeFromAssetItem(assetType)
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
	return ok.Websocket.AuthConn.SendJSONMessage(context.TODO(), request.Unset, input)
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
func (ok *Okx) WsOrderChannel(operation string, assetType asset.Item, pair currency.Pair, _ string) error {
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

// RfqSubscription subscription to retrieve Rfq updates on Rfq orders.
func (ok *Okx) RfqSubscription(operation, uid string) error {
	return ok.wsAuthChannelSubscription(operation, okxChannelRfqs, asset.Empty, currency.EMPTYPAIR, uid, "", wsSubscriptionParameters{})
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
	return ok.wsAuthChannelSubscription(operation, okxChannelGridSubOrders, asset.Empty, currency.EMPTYPAIR, "", algoID, wsSubscriptionParameters{})
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
		return fmt.Errorf("%w, received '%v' only FUTURES, SWAP and OPTION asset types are supported", errInvalidInstrumentType, assetType)
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
		return fmt.Errorf("%w, received '%v' only FUTURES and OPTION asset types are supported", errInvalidInstrumentType, assetType)
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
func (ok *Okx) PriceLimitSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	if operation != operationSubscribe && operation != operationUnsubscribe {
		return errInvalidWebsocketEvent
	}
	return ok.wsChannelSubscription(operation, okxChannelPriceLimit, assetType, pair, false, true, false)
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

// PublicBlockTradesSubscription a method to subscribe and unsubscribe to a "public-block-trades" channel to retrieve the recent block trades data by individual legs.
// Each leg in a block trade is pushed in a separate update. Data will be pushed whenever there is a block trade.
func (ok *Okx) PublicBlockTradesSubscription(operation string, assetType asset.Item, pair currency.Pair) error {
	return ok.wsChannelSubscription(operation, okxChannelPublicBlockTrades, assetType, pair, false, true, false)
}

// Websocket Spread Trade methods

// handleIncomingData extracts the incoming data to the dataHolder interface after few checks and return nil or return error message otherwise
func (ok *Okx) handleIncomingData(data *wsIncomingData, dataHolder StatusCodeHolder) error {
	sliceDataHolder := []StatusCodeHolder{dataHolder}
	if data.Code == "0" || data.Code == "1" {
		err := data.copyResponseToInterface(&sliceDataHolder)
		if err != nil {
			return err
		}
		if dataHolder == nil {
			return fmt.Errorf("%w, invalid incoming data", common.ErrNoResponse)
		}
		if data.Code == "1" {
			return fmt.Errorf("error code:%s message: %s", dataHolder.GetSCode(), dataHolder.GetSMsg())
		}
		return nil
	}
	return fmt.Errorf("error code:%s message: %v", data.Code, ErrorCodes[data.Code])
}

// WsPlaceSpreadOrder places a spread order thought the websocket connection stream, and returns a SubmitResponse and error message.
func (ok *Okx) WsPlaceSpreadOrder(arg *SpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil || *arg == (SpreadOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	err := ok.validatePlaceSpreadOrderParam(arg)
	if err != nil {
		return nil, err
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, input)
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
func (ok *Okx) WsAmandSpreadOrder(arg *AmendSpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil || *arg == (AmendSpreadOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewPrice == 0 && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, input)
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
func (ok *Okx) WsCancelSpreadOrder(orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, input)
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
				return dataHolder, ok.handleIncomingData(data, dataHolder)
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
func (ok *Okx) WsCancelAllSpreadOrders(spreadID string) (bool, error) {
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
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
	err = ok.Websocket.AuthConn.SendJSONMessage(context.Background(), request.UnAuth, input)
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
				return dataHolder.Result, ok.handleIncomingData(data, dataHolder)
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
