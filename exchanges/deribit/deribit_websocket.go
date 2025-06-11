package deribit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var deribitWebsocketAddress = "wss://www.deribit.com/ws" + deribitAPIVersion

const (
	rpcVersion    = "2.0"
	rateLimit     = 20
	errAuthFailed = 1002

	// public websocket channels
	announcementsChannel                   = "announcements"
	orderbookChannel                       = "book"
	chartTradesChannel                     = "chart.trades"
	priceIndexChannel                      = "deribit_price_index"
	priceRankingChannel                    = "deribit_price_ranking"
	priceStatisticsChannel                 = "deribit_price_statistics"
	volatilityIndexChannel                 = "deribit_volatility_index"
	estimatedExpirationPriceChannel        = "estimated_expiration_price"
	incrementalTickerChannel               = "incremental_ticker"
	instrumentStateChannel                 = "instrument.state"
	markPriceOptionsChannel                = "markprice.options"
	perpetualChannel                       = "perpetual."
	platformStateChannel                   = "platform_state"
	platformStatePublicMethodsStateChannel = "platform_state.public_methods_state"
	quoteChannel                           = "quote"
	requestForQuoteChannel                 = "rfq"
	tickerChannel                          = "ticker"
	tradesChannel                          = "trades"

	// private websocket channels
	userAccessLogChannel          = "user.access_log"
	userChangesInstrumentsChannel = "user.changes."
	userChangesCurrencyChannel    = "user.changes"
	userLockChannel               = "user.lock"
	userMMPTriggerChannel         = "user.mmp_trigger"
	userOrdersChannel             = "user.orders"
	userTradesChannel             = "user.trades"
	userPortfolioChannel          = "user.portfolio"
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:             tickerChannel,
	subscription.OrderbookChannel:          orderbookChannel,
	subscription.CandlesChannel:            chartTradesChannel,
	subscription.AllTradesChannel:          tradesChannel,
	subscription.MyTradesChannel:           userTradesChannel,
	subscription.MyOrdersChannel:           userOrdersChannel,
	announcementsChannel:                   announcementsChannel,
	priceIndexChannel:                      priceIndexChannel,
	priceRankingChannel:                    priceRankingChannel,
	priceStatisticsChannel:                 priceStatisticsChannel,
	volatilityIndexChannel:                 volatilityIndexChannel,
	estimatedExpirationPriceChannel:        estimatedExpirationPriceChannel,
	incrementalTickerChannel:               incrementalTickerChannel,
	instrumentStateChannel:                 instrumentStateChannel,
	markPriceOptionsChannel:                markPriceOptionsChannel,
	perpetualChannel:                       perpetualChannel,
	platformStateChannel:                   platformStateChannel,
	platformStatePublicMethodsStateChannel: platformStatePublicMethodsStateChannel,
	quoteChannel:                           quoteChannel,
	requestForQuoteChannel:                 requestForQuoteChannel,
	userAccessLogChannel:                   userAccessLogChannel,
	userChangesInstrumentsChannel:          userChangesInstrumentsChannel,
	userChangesCurrencyChannel:             userChangesCurrencyChannel,
	userLockChannel:                        userLockChannel,
	userMMPTriggerChannel:                  userMMPTriggerChannel,
	userPortfolioChannel:                   userPortfolioChannel,
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneDay},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, // Raw is available for authenticated users
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
}

var (
	pingMessage = WsSubscriptionInput{
		ID:             2,
		JSONRPCVersion: rpcVersion,
		Method:         "public/test",
		Params:         map[string][]string{},
	}
	setHeartBeatMessage = wsInput{
		ID:             1,
		JSONRPCVersion: rpcVersion,
		Method:         "public/set_heartbeat",
		Params: map[string]any{
			"interval": 15,
		},
	}
)

// WsConnect starts a new connection with the websocket API
func (d *Deribit) WsConnect() error {
	ctx := context.TODO()
	if !d.Websocket.IsEnabled() || !d.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := d.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	d.Websocket.Wg.Add(1)
	go d.wsReadData(ctx)
	if d.Websocket.CanUseAuthenticatedEndpoints() {
		err = d.wsLogin(ctx)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", d.Name, err)
			d.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return d.Websocket.Conn.SendJSONMessage(ctx, request.Unset, setHeartBeatMessage)
}

func (d *Deribit) wsLogin(ctx context.Context) error {
	if !d.IsWebsocketAuthenticationSupported() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", d.Name)
	}
	creds, err := d.GetCredentials(ctx)
	if err != nil {
		return err
	}
	d.Websocket.SetCanUseAuthenticatedEndpoints(true)
	n := d.Requester.GetNonce(nonce.UnixNano).String()
	strTS := strconv.FormatInt(time.Now().UnixMilli(), 10)
	str2Sign := strTS + "\n" + n + "\n"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(str2Sign), []byte(creds.Secret))
	if err != nil {
		return err
	}

	req := wsInput{
		JSONRPCVersion: rpcVersion,
		Method:         "public/auth",
		ID:             d.Websocket.Conn.GenerateMessageID(false),
		Params: map[string]any{
			"grant_type": "client_signature",
			"client_id":  creds.Key,
			"timestamp":  strTS,
			"nonce":      n,
			"signature":  hex.EncodeToString(hmac),
		},
	}
	resp, err := d.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		d.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	var response wsLoginResponse
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return fmt.Errorf("%v %v", d.Name, err)
	}
	if response.Error != nil && (response.Error.Code > 0 || response.Error.Message != "") {
		return fmt.Errorf("%v Error:%v Message:%v", d.Name, response.Error.Code, response.Error.Message)
	}
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (d *Deribit) wsReadData(ctx context.Context) {
	defer d.Websocket.Wg.Done()

	for {
		resp := d.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		err := d.wsHandleData(ctx, resp.Raw)
		if err != nil {
			d.Websocket.DataHandler <- err
		}
	}
}

func (d *Deribit) wsHandleData(ctx context.Context, respRaw []byte) error {
	var response WsResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return fmt.Errorf("%s - err %s could not parse websocket data: %s", d.Name, err, respRaw)
	}
	if response.Method == "heartbeat" {
		return d.Websocket.Conn.SendJSONMessage(ctx, request.Unset, pingMessage)
	}
	if response.ID > 2 {
		if !d.Websocket.Match.IncomingWithData(response.ID, respRaw) {
			return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %d", response.ID)
		}
		return nil
	} else if response.ID > 0 {
		return nil
	}
	channels := strings.Split(response.Params.Channel, ".")
	switch channels[0] {
	case "announcements":
		announcement := &Announcement{}
		response.Params.Data = announcement
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		d.Websocket.DataHandler <- announcement
	case "book":
		return d.processOrderbook(respRaw, channels)
	case "chart":
		return d.processCandleChart(respRaw, channels)
	case "deribit_price_index":
		indexPrice := &wsIndexPrice{}
		return d.processData(respRaw, indexPrice)
	case "deribit_price_ranking":
		priceRankings := &wsRankingPrices{}
		return d.processData(respRaw, priceRankings)
	case "deribit_price_statistics":
		priceStatistics := &wsPriceStatistics{}
		return d.processData(respRaw, priceStatistics)
	case "deribit_volatility_index":
		volatilityIndex := &wsVolatilityIndex{}
		return d.processData(respRaw, volatilityIndex)
	case "estimated_expiration_price":
		estimatedExpirationPrice := &wsEstimatedExpirationPrice{}
		return d.processData(respRaw, estimatedExpirationPrice)
	case "incremental_ticker":
		return d.processIncrementalTicker(respRaw, channels)
	case "instrument":
		instrumentState := &wsInstrumentState{}
		return d.processData(respRaw, instrumentState)
	case "markprice":
		markPriceOptions := []wsMarkPriceOptions{}
		return d.processData(respRaw, markPriceOptions)
	case "perpetual":
		perpetualInterest := &wsPerpetualInterest{}
		return d.processData(respRaw, perpetualInterest)
	case platformStateChannel:
		platformState := &wsPlatformState{}
		return d.processData(respRaw, platformState)
	case "quote": // Quote ticker information.
		return d.processQuoteTicker(respRaw, channels)
	case "rfq":
		rfq := &wsRequestForQuote{}
		return d.processData(respRaw, rfq)
	case "ticker":
		return d.processInstrumentTicker(respRaw, channels)
	case "trades":
		return d.processTrades(respRaw, channels)
	case "user":
		switch channels[1] {
		case "access_log":
			accessLog := &wsAccessLog{}
			return d.processData(respRaw, accessLog)
		case "changes":
			return d.processUserOrderChanges(respRaw, channels)
		case "lock":
			userLock := &WsUserLock{}
			return d.processData(respRaw, userLock)
		case "mmp_trigger":
			data := &WsMMPTrigger{
				Currency: channels[2],
			}
			return d.processData(respRaw, data)
		case "orders":
			return d.processUserOrders(respRaw, channels)
		case "portfolio":
			portfolio := &wsUserPortfolio{}
			return d.processData(respRaw, portfolio)
		case "trades":
			return d.processTrades(respRaw, channels)
		default:
			d.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: d.Name + websocket.UnhandledMessage + string(respRaw),
			}
			return nil
		}
	case "public/test", "public/set_heartbeat":
	default:
		switch result := response.Result.(type) {
		case string:
			if result == "ok" {
				return nil
			}
		default:
			d.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: d.Name + websocket.UnhandledMessage + string(respRaw),
			}
			return nil
		}
	}
	return nil
}

func (d *Deribit) processUserOrders(respRaw []byte, channels []string) error {
	if len(channels) != 4 && len(channels) != 5 {
		return fmt.Errorf("%w, expected format 'user.orders.{instrument_name}.raw, user.orders.{instrument_name}.{interval}, user.orders.{kind}.{currency}.raw, or user.orders.{kind}.{currency}.{interval}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	var response WsResponse
	orderData := []WsOrder{}
	response.Params.Data = orderData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(orderData))
	for x := range orderData {
		cp, a, err := d.getAssetPairByInstrument(orderData[x].InstrumentName)
		if err != nil {
			return err
		}
		oType, err := order.StringToOrderType(orderData[x].OrderType)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(orderData[x].Direction)
		if err != nil {
			return err
		}
		status, err := order.StringToOrderStatus(orderData[x].OrderState)
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			Price:           orderData[x].Price,
			Amount:          orderData[x].Amount,
			ExecutedAmount:  orderData[x].FilledAmount,
			RemainingAmount: orderData[x].Amount - orderData[x].FilledAmount,
			Exchange:        d.Name,
			OrderID:         orderData[x].OrderID,
			Type:            oType,
			Side:            side,
			Status:          status,
			AssetType:       a,
			Date:            orderData[x].CreationTimestamp.Time(),
			LastUpdated:     orderData[x].LastUpdateTimestamp.Time(),
			Pair:            cp,
		}
	}
	d.Websocket.DataHandler <- orderDetails
	return nil
}

func (d *Deribit) processUserOrderChanges(respRaw []byte, channels []string) error {
	if len(channels) < 4 || len(channels) > 5 {
		return fmt.Errorf("%w, expected format 'trades.{instrument_name}.{interval} or trades.{kind}.{currency}.{interval}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	var response WsResponse
	changeData := &wsChanges{}
	response.Params.Data = changeData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	td := make([]trade.Data, len(changeData.Trades))
	for x := range changeData.Trades {
		var side order.Side
		side, err = order.StringToOrderSide(changeData.Trades[x].Direction)
		if err != nil {
			return err
		}
		var cp currency.Pair
		var a asset.Item
		cp, a, err = d.getAssetPairByInstrument(changeData.Trades[x].InstrumentName)
		if err != nil {
			return err
		}

		td[x] = trade.Data{
			CurrencyPair: cp,
			Exchange:     d.Name,
			Timestamp:    changeData.Trades[x].Timestamp.Time(),
			Price:        changeData.Trades[x].Price,
			Amount:       changeData.Trades[x].Amount,
			Side:         side,
			TID:          changeData.Trades[x].TradeID,
			AssetType:    a,
		}
	}
	err = trade.AddTradesToBuffer(td...)
	if err != nil {
		return err
	}
	orders := make([]order.Detail, len(changeData.Orders))
	for x := range orders {
		oType, err := order.StringToOrderType(changeData.Orders[x].OrderType)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(changeData.Orders[x].Direction)
		if err != nil {
			return err
		}
		status, err := order.StringToOrderStatus(changeData.Orders[x].OrderState)
		if err != nil {
			return err
		}
		cp, a, err := d.getAssetPairByInstrument(changeData.Orders[x].InstrumentName)
		if err != nil {
			return err
		}
		orders[x] = order.Detail{
			Price:           changeData.Orders[x].Price,
			Amount:          changeData.Orders[x].Amount,
			ExecutedAmount:  changeData.Orders[x].FilledAmount,
			RemainingAmount: changeData.Orders[x].Amount - changeData.Orders[x].FilledAmount,
			Exchange:        d.Name,
			OrderID:         changeData.Orders[x].OrderID,
			Type:            oType,
			Side:            side,
			Status:          status,
			AssetType:       a,
			Date:            changeData.Orders[x].CreationTimestamp.Time(),
			LastUpdated:     changeData.Orders[x].LastUpdateTimestamp.Time(),
			Pair:            cp,
		}
	}
	d.Websocket.DataHandler <- orders
	d.Websocket.DataHandler <- changeData.Positions
	return nil
}

func (d *Deribit) processQuoteTicker(respRaw []byte, channels []string) error {
	cp, a, err := d.getAssetPairByInstrument(channels[1])
	if err != nil {
		return err
	}
	var response WsResponse
	quoteTicker := &wsQuoteTickerInformation{}
	response.Params.Data = quoteTicker
	err = json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	d.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: d.Name,
		Pair:         cp,
		AssetType:    a,
		LastUpdated:  quoteTicker.Timestamp.Time(),
		Bid:          quoteTicker.BestBidPrice,
		Ask:          quoteTicker.BestAskPrice,
		BidSize:      quoteTicker.BestBidAmount,
		AskSize:      quoteTicker.BestAskAmount,
	}
	return nil
}

func (d *Deribit) processTrades(respRaw []byte, channels []string) error {
	tradeFeed := d.IsTradeFeedEnabled()
	saveTradeData := d.IsSaveTradeDataEnabled()
	if !tradeFeed && !saveTradeData {
		return nil
	}

	if len(channels) < 3 || len(channels) > 5 {
		return fmt.Errorf("%w, expected format 'trades.{instrument_name}.{interval} or trades.{kind}.{currency}.{interval}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	var response WsResponse
	var tradeList []wsTrade
	response.Params.Data = &tradeList
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(tradeList) == 0 {
		return fmt.Errorf("%v, empty list of trades found", common.ErrNoResponse)
	}
	tradesData := make([]trade.Data, len(tradeList))
	for x := range tradesData {
		var cp currency.Pair
		var a asset.Item
		cp, a, err = d.getAssetPairByInstrument(tradeList[x].InstrumentName)
		if err != nil {
			return err
		}
		tradesData[x] = trade.Data{
			CurrencyPair: cp,
			Exchange:     d.Name,
			Timestamp:    tradeList[x].Timestamp.Time().UTC(),
			Price:        tradeList[x].Price,
			Amount:       tradeList[x].Amount,
			Side:         tradeList[x].Direction,
			TID:          tradeList[x].TradeID,
			AssetType:    a,
		}
	}
	if tradeFeed {
		for i := range tradesData {
			d.Websocket.DataHandler <- tradesData[i]
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(tradesData...)
	}
	return nil
}

func (d *Deribit) processIncrementalTicker(respRaw []byte, channels []string) error {
	if len(channels) != 2 {
		return fmt.Errorf("%w, expected format 'incremental_ticker.{instrument_name}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	cp, a, err := d.getAssetPairByInstrument(channels[1])
	if err != nil {
		return err
	}
	var response WsResponse
	incrementalTicker := &WsIncrementalTicker{}
	response.Params.Data = incrementalTicker
	err = json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	d.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: d.Name,
		Pair:         cp,
		AssetType:    a,
		LastUpdated:  incrementalTicker.Timestamp.Time(),
		BidSize:      incrementalTicker.BestBidAmount,
		AskSize:      incrementalTicker.BestAskAmount,
		High:         incrementalTicker.MaxPrice,
		Low:          incrementalTicker.MinPrice,
		Volume:       incrementalTicker.Stats.Volume,
		QuoteVolume:  incrementalTicker.Stats.VolumeUsd,
		Ask:          incrementalTicker.ImpliedAsk,
		Bid:          incrementalTicker.ImpliedBid,
	}
	return nil
}

func (d *Deribit) processInstrumentTicker(respRaw []byte, channels []string) error {
	if len(channels) != 3 {
		return fmt.Errorf("%w, expected format 'ticker.{instrument_name}.{interval}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	return d.processTicker(respRaw, channels)
}

func (d *Deribit) processTicker(respRaw []byte, channels []string) error {
	cp, a, err := d.getAssetPairByInstrument(channels[1])
	if err != nil {
		return err
	}
	var response WsResponse
	tickerPriceResponse := &wsTicker{}
	response.Params.Data = tickerPriceResponse
	err = json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	tickerPrice := &ticker.Price{
		ExchangeName: d.Name,
		Pair:         cp,
		AssetType:    a,
		LastUpdated:  tickerPriceResponse.Timestamp.Time(),
		Bid:          tickerPriceResponse.BestBidPrice,
		Ask:          tickerPriceResponse.BestAskPrice,
		BidSize:      tickerPriceResponse.BestBidAmount,
		AskSize:      tickerPriceResponse.BestAskAmount,
		Last:         tickerPriceResponse.LastPrice,
		High:         tickerPriceResponse.Stats.High,
		Low:          tickerPriceResponse.Stats.Low,
		Volume:       tickerPriceResponse.Stats.Volume,
	}
	if a != asset.Futures {
		tickerPrice.Low = tickerPriceResponse.MinPrice
		tickerPrice.High = tickerPriceResponse.MaxPrice
		tickerPrice.Last = tickerPriceResponse.MarkPrice
		tickerPrice.Ask = tickerPriceResponse.ImpliedAsk
		tickerPrice.Bid = tickerPriceResponse.ImpliedBid
	}
	d.Websocket.DataHandler <- tickerPrice
	return nil
}

func (d *Deribit) processData(respRaw []byte, result any) error {
	var response WsResponse
	response.Params.Data = result
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	d.Websocket.DataHandler <- result
	return nil
}

func (d *Deribit) processCandleChart(respRaw []byte, channels []string) error {
	if len(channels) != 4 {
		return fmt.Errorf("%w, expected format 'chart.trades.{instrument_name}.{resolution}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	cp, a, err := d.getAssetPairByInstrument(channels[2])
	if err != nil {
		return err
	}
	var response WsResponse
	candleData := &wsCandlestickData{}
	response.Params.Data = candleData
	err = json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	d.Websocket.DataHandler <- websocket.KlineData{
		Timestamp:  candleData.Tick.Time(),
		Pair:       cp,
		AssetType:  a,
		Exchange:   d.Name,
		OpenPrice:  candleData.Open,
		HighPrice:  candleData.High,
		LowPrice:   candleData.Low,
		ClosePrice: candleData.Close,
		Volume:     candleData.Volume,
	}
	return nil
}

func (d *Deribit) processOrderbook(respRaw []byte, channels []string) error {
	var response WsResponse
	orderbookData := &wsOrderbook{}
	response.Params.Data = orderbookData
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return err
	}
	if len(channels) == 3 {
		cp, a, err := d.getAssetPairByInstrument(orderbookData.InstrumentName)
		if err != nil {
			return err
		}
		asks := make(orderbook.Levels, 0, len(orderbookData.Asks))
		for x := range orderbookData.Asks {
			if len(orderbookData.Asks[x]) != 3 {
				return errMalformedData
			}
			price, okay := orderbookData.Asks[x][1].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid orderbook price", errMalformedData)
			}
			amount, okay := orderbookData.Asks[x][2].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid amount", errMalformedData)
			}
			asks = append(asks, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		bids := make(orderbook.Levels, 0, len(orderbookData.Bids))
		for x := range orderbookData.Bids {
			if len(orderbookData.Bids[x]) != 3 {
				return errMalformedData
			}
			price, okay := orderbookData.Bids[x][1].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid orderbook price", errMalformedData)
			} else if price == 0.0 {
				continue
			}
			amount, okay := orderbookData.Bids[x][2].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid amount", errMalformedData)
			}
			bids = append(bids, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}

		switch orderbookData.Type {
		case "snapshot":
			return d.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Exchange:          d.Name,
				ValidateOrderbook: d.ValidateOrderbook,
				LastUpdated:       orderbookData.Timestamp.Time(),
				Pair:              cp,
				Asks:              asks,
				Bids:              bids,
				Asset:             a,
				LastUpdateID:      orderbookData.ChangeID,
			})
		case "change":
			return d.Websocket.Orderbook.Update(&orderbook.Update{
				Asks:       asks,
				Bids:       bids,
				Pair:       cp,
				Asset:      a,
				UpdateID:   orderbookData.ChangeID,
				UpdateTime: orderbookData.Timestamp.Time(),
			})
		}
	} else if len(channels) == 5 {
		cp, a, err := d.getAssetPairByInstrument(orderbookData.InstrumentName)
		if err != nil {
			return err
		}
		asks := make(orderbook.Levels, 0, len(orderbookData.Asks))
		for x := range orderbookData.Asks {
			if len(orderbookData.Asks[x]) != 2 {
				return errMalformedData
			}
			price, okay := orderbookData.Asks[x][0].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid orderbook price", errMalformedData)
			} else if price == 0 {
				continue
			}
			amount, okay := orderbookData.Asks[x][1].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid amount", errMalformedData)
			}
			asks = append(asks, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		bids := make([]orderbook.Level, 0, len(orderbookData.Bids))
		for x := range orderbookData.Bids {
			if len(orderbookData.Bids[x]) != 2 {
				return errMalformedData
			}
			price, okay := orderbookData.Bids[x][0].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid orderbook price", errMalformedData)
			} else if price == 0 {
				continue
			}
			amount, okay := orderbookData.Bids[x][1].(float64)
			if !okay {
				return fmt.Errorf("%w, invalid amount", errMalformedData)
			}
			bids = append(bids, orderbook.Level{
				Price:  price,
				Amount: amount,
			})
		}
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}
		return d.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Asks:         asks,
			Bids:         bids,
			Pair:         cp,
			Asset:        a,
			Exchange:     d.Name,
			LastUpdateID: orderbookData.ChangeID,
			LastUpdated:  orderbookData.Timestamp.Time(),
		})
	}
	return nil
}

// generateSubscriptions returns a list of configured subscriptions
func (d *Deribit) generateSubscriptions() (subscription.List, error) {
	return d.Features.Subscriptions.ExpandTemplates(d)
}

// GetSubscriptionTemplate returns a subscription channel template
func (d *Deribit) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":     channelName,
		"interval":        channelInterval,
		"isSymbolChannel": isSymbolChannel,
		"fmt":             formatChannelPair,
	}).
		Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (d *Deribit) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	errs := d.handleSubscription(ctx, "public/subscribe", subs.Public())
	return common.AppendError(errs, d.handleSubscription(ctx, "private/subscribe", subs.Private()))
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (d *Deribit) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	errs := d.handleSubscription(ctx, "public/unsubscribe", subs.Public())
	return common.AppendError(errs, d.handleSubscription(ctx, "private/unsubscribe", subs.Private()))
}

func (d *Deribit) handleSubscription(ctx context.Context, method string, subs subscription.List) error {
	var err error
	subs, err = subs.ExpandTemplates(d)
	if err != nil || len(subs) == 0 {
		return err
	}

	r := WsSubscriptionInput{
		JSONRPCVersion: rpcVersion,
		ID:             d.Websocket.Conn.GenerateMessageID(false),
		Method:         method,
		Params:         map[string][]string{"channels": subs.QualifiedChannels()},
	}

	data, err := d.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, r.ID, r)
	if err != nil {
		return err
	}

	var response wsSubscriptionResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return fmt.Errorf("%v %v", d.Name, err)
	}
	subAck := map[string]bool{}
	for _, c := range response.Result {
		subAck[c] = true
	}
	if len(subAck) != len(subs) {
		err = websocket.ErrSubscriptionFailure
	}
	for _, s := range subs {
		if _, ok := subAck[s.QualifiedChannel]; ok {
			delete(subAck, s.QualifiedChannel)
			if !strings.Contains(method, "unsubscribe") {
				err = common.AppendError(err, d.Websocket.AddSuccessfulSubscriptions(d.Websocket.Conn, s))
			} else {
				err = common.AppendError(err, d.Websocket.RemoveSubscriptions(d.Websocket.Conn, s))
			}
		} else {
			err = common.AppendError(err, errors.New(s.String()+" failed to "+method))
		}
	}

	for key := range subAck {
		err = common.AppendError(err, fmt.Errorf("unexpected channel %q in result", key))
	}

	return err
}

func getValidatedCurrencyCode(pair currency.Pair) string {
	currencyCode := pair.Base.Upper().String()
	switch currencyCode {
	case currencyBTC, currencyETH,
		currencySOL, currencyUSDT,
		currencyUSDC, currencyEURR:
		return currencyCode
	default:
		switch {
		case strings.Contains(pair.String(), currencyUSDC):
			return currencyUSDC
		case strings.Contains(pair.String(), currencyUSDT):
			return currencyUSDT
		}
		return "any"
	}
}

func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

// channelInterval converts an interval to an exchange specific interval
// We convert 1s to agg2; Docs do not explain agg2 but support explained that it may vary under load but is currently 1 second
func channelInterval(s *subscription.Subscription) string {
	if s.Interval != 0 {
		if channelName(s) == chartTradesChannel {
			if s.Interval == kline.OneDay {
				return "1D"
			}
			m := s.Interval.Duration().Minutes()
			switch m {
			case 1, 3, 5, 10, 15, 30, 60, 120, 180, 360, 720: // Valid Minute intervals
				return strconv.Itoa(int(m))
			}
			panic(fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, s.Interval))
		}
		switch s.Interval {
		case kline.ThousandMilliseconds:
			return "agg2"
		case kline.HundredMilliseconds, kline.Raw:
			return s.Interval.Short()
		}
		panic(fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, s.Interval))
	}
	return ""
}

func isSymbolChannel(s *subscription.Subscription) bool {
	switch channelName(s) {
	case orderbookChannel, chartTradesChannel, tickerChannel, tradesChannel, perpetualChannel, quoteChannel,
		userChangesInstrumentsChannel, incrementalTickerChannel, userOrdersChannel, userTradesChannel:
		return true
	}
	return false
}

func formatChannelPair(pair currency.Pair) string {
	if str := pair.Quote.String(); strings.Contains(str, "PERPETUAL") && strings.Contains(str, "-") {
		pair.Delimiter = "_"
	}
	return pair.String()
}

const subTplText = `
{{- if isSymbolChannel $.S -}}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- range $p := $pairs }}
			{{- channelName $.S -}} . {{- fmt $p }}
			{{- with $i := interval $.S -}} . {{- $i }}{{ end }}
			{{- $.PairSeparator }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- channelName $.S -}}
	{{- with $i := interval $.S -}} . {{- $i }}{{ end }}
{{- end }}
`
