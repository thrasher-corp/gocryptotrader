package deribit

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	tickerChannel                          = "ticker."
	tradesChannel                          = "trades."
	tradesWithKindChannel                  = "trades"

	// private websocket channels
	userAccessLogChannel                             = "user.access_log"
	userChangesInstrumentsChannel                    = "user.changes."
	userChangesCurrencyChannel                       = "user.changes"
	userLockChannel                                  = "user.lock"
	userMMPTriggerChannel                            = "user.mmp_trigger"
	rawUserOrdersChannel                             = "user.orders.%s.raw"
	userOrdersWithIntervalChannel                    = "user.orders."
	rawUsersOrdersKindCurrencyChannel                = "user.orders.%s.%s.raw"
	rawUsersOrdersWithKindCurrencyAndIntervalChannel = "user.orders"
	userPortfolioChannel                             = "user.portfolio"
	userTradesChannelByInstrument                    = "user.trades."
	userTradesByKindCurrencyAndIntervalChannel       = "user.trades"
)

var (
	defaultSubscriptions = []string{
		chartTradesChannel, // chart trades channel to fetch candlestick data.
		orderbookChannel,
		tickerChannel,
		tradesWithKindChannel,
	}

	indexENUMS = []string{"ada_usd", "algo_usd", "avax_usd", "bch_usd", "bnb_usd", "btc_usd", "doge_usd", "dot_usd", "eth_usd", "link_usd", "ltc_usd", "luna_usd", "matic_usd", "near_usd", "shib_usd", "sol_usd", "trx_usd", "uni_usd", "usdc_usd", "xrp_usd", "ada_usdc", "bch_usdc", "algo_usdc", "avax_usdc", "btc_usdc", "doge_usdc", "dot_usdc", "bch_usdc", "bnb_usdc", "eth_usdc", "link_usdc", "ltc_usdc", "luna_usdc", "matic_usdc", "near_usdc", "shib_usdc", "sol_usdc", "trx_usdc", "uni_usdc", "xrp_usdc", "btcdvol_usdc", "ethdvol_usdc"}

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
		Params: map[string]interface{}{
			"interval": 15,
		},
	}
)

// WsConnect starts a new connection with the websocket API
func (d *Deribit) WsConnect() error {
	if !d.Websocket.IsEnabled() || !d.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := d.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	d.Websocket.Wg.Add(1)
	go d.wsReadData()
	if d.Websocket.CanUseAuthenticatedEndpoints() {
		err = d.wsLogin(context.TODO())
		if err != nil {
			log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", d.Name, err)
			d.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return d.Websocket.Conn.SendJSONMessage(setHeartBeatMessage)
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
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(str2Sign),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	request := wsInput{
		JSONRPCVersion: rpcVersion,
		Method:         "public/auth",
		ID:             d.Websocket.Conn.GenerateMessageID(false),
		Params: map[string]interface{}{
			"grant_type": "client_signature",
			"client_id":  creds.Key,
			"timestamp":  strTS,
			"nonce":      n,
			"signature":  crypto.HexEncodeToString(hmac),
		},
	}
	resp, err := d.Websocket.Conn.SendMessageReturnResponse(request.ID, request)
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
func (d *Deribit) wsReadData() {
	defer d.Websocket.Wg.Done()

	for {
		resp := d.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}

		err := d.wsHandleData(resp.Raw)
		if err != nil {
			d.Websocket.DataHandler <- err
		}
	}
}

func (d *Deribit) wsHandleData(respRaw []byte) error {
	var response WsResponse
	err := json.Unmarshal(respRaw, &response)
	if err != nil {
		return fmt.Errorf("%s - err %s could not parse websocket data: %s", d.Name, err, respRaw)
	}
	if response.Method == "heartbeat" {
		return d.Websocket.Conn.SendJSONMessage(pingMessage)
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
			d.Websocket.DataHandler <- stream.UnhandledMessageWarning{
				Message: d.Name + stream.UnhandledMessage + string(respRaw),
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
			d.Websocket.DataHandler <- stream.UnhandledMessageWarning{
				Message: d.Name + stream.UnhandledMessage + string(respRaw),
			}
			return nil
		}
	}
	return nil
}

func (d *Deribit) processUserOrders(respRaw []byte, channels []string) error {
	var response WsResponse
	if len(channels) != 4 && len(channels) != 5 {
		return fmt.Errorf("%w, expected format 'user.orders.{instrument_name}.raw, user.orders.{instrument_name}.{interval}, user.orders.{kind}.{currency}.raw, or user.orders.{kind}.{currency}.{interval}', but found %s", errMalformedData, strings.Join(channels, "."))
	}
	var orderData []WsOrder
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
	err = trade.AddTradesToBuffer(d.Name, td...)
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
	tradeDatas := make([]trade.Data, len(tradeList))
	for x := range tradeDatas {
		var cp currency.Pair
		var a asset.Item
		cp, a, err = d.getAssetPairByInstrument(tradeList[x].InstrumentName)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(tradeList[x].Direction)
		if err != nil {
			return err
		}
		tradeDatas[x] = trade.Data{
			CurrencyPair: cp,
			Exchange:     d.Name,
			Timestamp:    tradeList[x].Timestamp.Time(),
			Price:        tradeList[x].Price,
			Amount:       tradeList[x].Amount,
			Side:         side,
			TID:          tradeList[x].TradeID,
			AssetType:    a,
		}
	}
	return trade.AddTradesToBuffer(d.Name, tradeDatas...)
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

func (d *Deribit) processData(respRaw []byte, result interface{}) error {
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
	d.Websocket.DataHandler <- stream.KlineData{
		Timestamp:  time.UnixMilli(candleData.Tick),
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
		asks := make(orderbook.Tranches, 0, len(orderbookData.Asks))
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
			asks = append(asks, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		bids := make(orderbook.Tranches, 0, len(orderbookData.Bids))
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
			bids = append(bids, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}
		if orderbookData.Type == "snapshot" {
			return d.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Exchange:        d.Name,
				VerifyOrderbook: d.CanVerifyOrderbook,
				LastUpdated:     orderbookData.Timestamp.Time(),
				Pair:            cp,
				Asks:            asks,
				Bids:            bids,
				Asset:           a,
				LastUpdateID:    orderbookData.ChangeID,
			})
		} else if orderbookData.Type == "change" {
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
		asks := make(orderbook.Tranches, 0, len(orderbookData.Asks))
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
			asks = append(asks, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		bids := make([]orderbook.Tranche, 0, len(orderbookData.Bids))
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
			bids = append(bids, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		if len(asks) == 0 && len(bids) == 0 {
			return nil
		}
		return d.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
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

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (d *Deribit) GenerateDefaultSubscriptions() ([]subscription.Subscription, error) {
	var subscriptions []subscription.Subscription
	assets := d.GetAssetTypes(true)
	subscriptionChannels := defaultSubscriptions
	if d.Websocket.CanUseAuthenticatedEndpoints() {
		subscriptionChannels = append(
			subscriptionChannels,

			// authenticated subscriptions
			rawUsersOrdersKindCurrencyChannel,
			rawUsersOrdersWithKindCurrencyAndIntervalChannel,
			userTradesByKindCurrencyAndIntervalChannel,
		)
	}
	var err error
	assetPairs := make(map[asset.Item][]currency.Pair, len(assets))
	for _, a := range assets {
		assetPairs[a], err = d.GetEnabledPairs(a)
		if err != nil {
			return nil, err
		}
		if len(assetPairs[a]) > 5 {
			assetPairs[a] = assetPairs[a][:5]
		}
	}
	for x := range subscriptionChannels {
		switch subscriptionChannels[x] {
		case chartTradesChannel:
			for _, a := range assets {
				for z := range assetPairs[a] {
					if ((assetPairs[a][z].Quote.Upper().String() == perpString ||
						!strings.Contains(assetPairs[a][z].Quote.Upper().String(), perpString)) &&
						a == asset.Futures) || (a != asset.Spot && a != asset.Futures) {
						continue
					}
					subscriptions = append(subscriptions,
						subscription.Subscription{
							Channel: subscriptionChannels[x],
							Pair:    assetPairs[a][z],
							Params: map[string]interface{}{
								"resolution": "1D",
							},
							Asset: a,
						})
				}
			}
		case incrementalTickerChannel,
			quoteChannel,
			rawUserOrdersChannel:
			for _, a := range assets {
				for z := range assetPairs[a] {
					if ((assetPairs[a][z].Quote.Upper().String() == perpString ||
						!strings.Contains(assetPairs[a][z].Quote.Upper().String(), perpString)) &&
						a == asset.Futures) || (a != asset.Spot && a != asset.Futures) {
						continue
					}
					subscriptions = append(subscriptions, subscription.Subscription{
						Channel: subscriptionChannels[x],
						Pair:    assetPairs[a][z],
						Asset:   a,
					})
				}
			}
		case orderbookChannel:
			for _, a := range assets {
				for z := range assetPairs[a] {
					if ((assetPairs[a][z].Quote.Upper().String() == perpString ||
						!strings.Contains(assetPairs[a][z].Quote.Upper().String(), perpString)) &&
						a == asset.Futures) || (a != asset.Spot && a != asset.Futures) {
						continue
					}
					subscriptions = append(subscriptions,
						subscription.Subscription{
							Channel: subscriptionChannels[x],
							Pair:    assetPairs[a][z],
							// if needed, group and depth of orderbook can be passed as follow "group":    "250", "depth":    "20",
							Interval: kline.HundredMilliseconds,
							Asset:    a,
							Params: map[string]interface{}{
								"group": "none",
								"depth": "10",
							},
						},
					)
					if d.Websocket.CanUseAuthenticatedEndpoints() {
						subscriptions = append(subscriptions, subscription.Subscription{
							Channel:  orderbookChannel,
							Pair:     assetPairs[a][z],
							Asset:    a,
							Interval: kline.Interval(0),
							Params: map[string]interface{}{
								"group": "none",
								"depth": "10",
							},
						})
					}
				}
			}
		case tickerChannel,
			tradesChannel:
			for _, a := range assets {
				for z := range assetPairs[a] {
					if ((assetPairs[a][z].Quote.Upper().String() != perpString &&
						!strings.Contains(assetPairs[a][z].Quote.Upper().String(), perpString)) &&
						a == asset.Futures) || (a != asset.Spot && a != asset.Futures) {
						continue
					}
					subscriptions = append(subscriptions,
						subscription.Subscription{
							Channel:  subscriptionChannels[x],
							Pair:     assetPairs[a][z],
							Interval: kline.HundredMilliseconds,
							Asset:    a,
						})
				}
			}
		case perpetualChannel,
			userChangesInstrumentsChannel,
			userTradesChannelByInstrument:
			for _, a := range assets {
				for z := range assetPairs[a] {
					if subscriptionChannels[x] == perpetualChannel && !strings.Contains(assetPairs[a][z].Quote.Upper().String(), perpString) {
						continue
					}
					subscriptions = append(subscriptions,
						subscription.Subscription{
							Channel:  subscriptionChannels[x],
							Pair:     assetPairs[a][z],
							Interval: kline.HundredMilliseconds,
							Asset:    a,
						})
				}
			}
		case instrumentStateChannel,
			rawUsersOrdersKindCurrencyChannel:
			var okay bool
			for _, a := range assets {
				currencyPairsName := make(map[currency.Code]bool, 2*len(assetPairs[a]))
				for z := range assetPairs[a] {
					if okay = currencyPairsName[assetPairs[a][z].Base]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Asset:   a,
							Channel: subscriptionChannels[x],
							Pair:    currency.Pair{Base: assetPairs[a][z].Base},
						})
						currencyPairsName[assetPairs[a][z].Base] = true
					}
					if okay = currencyPairsName[assetPairs[a][z].Quote]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Asset:   a,
							Channel: subscriptionChannels[x],
							Pair:    currency.Pair{Base: assetPairs[a][z].Quote},
						})
						currencyPairsName[assetPairs[a][z].Quote] = true
					}
				}
			}
		case userChangesCurrencyChannel,
			userOrdersWithIntervalChannel,
			rawUsersOrdersWithKindCurrencyAndIntervalChannel,
			userTradesByKindCurrencyAndIntervalChannel,
			tradesWithKindChannel:
			for _, a := range assets {
				currencyPairsName := make(map[currency.Code]bool, 2*len(assetPairs[a]))
				var okay bool
				for z := range assetPairs[a] {
					if okay = currencyPairsName[assetPairs[a][z].Base]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Asset:    a,
							Channel:  subscriptionChannels[x],
							Pair:     currency.Pair{Base: assetPairs[a][z].Base},
							Interval: kline.HundredMilliseconds,
						})
						currencyPairsName[assetPairs[a][z].Base] = true
					}
					if okay = currencyPairsName[assetPairs[a][z].Quote]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Asset:    a,
							Channel:  subscriptionChannels[x],
							Pair:     currency.Pair{Base: assetPairs[a][z].Quote},
							Interval: kline.HundredMilliseconds,
						})
						currencyPairsName[assetPairs[a][z].Quote] = true
					}
				}
			}
		case requestForQuoteChannel,
			userMMPTriggerChannel,
			userPortfolioChannel:
			for _, a := range assets {
				currencyPairsName := make(map[currency.Code]bool, 2*len(assetPairs[a]))
				var okay bool
				for z := range assetPairs[a] {
					if okay = currencyPairsName[assetPairs[a][z].Base]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Channel: subscriptionChannels[x],
							Pair:    currency.Pair{Base: assetPairs[a][z].Base},
							Asset:   a,
						})
						currencyPairsName[assetPairs[a][z].Base] = true
					}
					if okay = currencyPairsName[assetPairs[a][z].Quote]; !okay {
						subscriptions = append(subscriptions, subscription.Subscription{
							Channel: subscriptionChannels[x],
							Pair:    currency.Pair{Base: assetPairs[a][z].Quote},
							Asset:   a,
						})
						currencyPairsName[assetPairs[a][z].Quote] = true
					}
				}
			}
		case announcementsChannel,
			userAccessLogChannel,
			platformStateChannel,
			userLockChannel,
			platformStatePublicMethodsStateChannel:
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: subscriptionChannels[x],
			})
		case priceIndexChannel,
			priceRankingChannel,
			priceStatisticsChannel,
			volatilityIndexChannel,
			markPriceOptionsChannel,
			estimatedExpirationPriceChannel:
			for i := range indexENUMS {
				subscriptions = append(subscriptions, subscription.Subscription{
					Channel: subscriptionChannels[x],
					Params: map[string]interface{}{
						"index_name": indexENUMS[i],
					},
				})
			}
		}
	}
	return subscriptions, nil
}

func (d *Deribit) generatePayloadFromSubscriptionInfos(operation string, subscs []subscription.Subscription) ([]WsSubscriptionInput, error) {
	subscriptionPayloads := make([]WsSubscriptionInput, len(subscs))
	for x := range subscs {
		sub := WsSubscriptionInput{
			JSONRPCVersion: rpcVersion,
			ID:             d.Websocket.Conn.GenerateMessageID(false),
			Method:         "public/" + operation,
			Params:         map[string][]string{},
		}
		switch subscs[x].Channel {
		case userAccessLogChannel, userChangesInstrumentsChannel, userChangesCurrencyChannel, userLockChannel, userMMPTriggerChannel, rawUserOrdersChannel,
			userOrdersWithIntervalChannel, rawUsersOrdersKindCurrencyChannel, userPortfolioChannel, userTradesChannelByInstrument, userTradesByKindCurrencyAndIntervalChannel:
			if !d.Websocket.CanUseAuthenticatedEndpoints() {
				continue
			}
			sub.Method = "private/" + operation
		}
		var instrumentID string
		if !subscs[x].Pair.IsEmpty() {
			pairFormat, err := d.GetPairFormat(subscs[x].Asset, true)
			if err != nil {
				return nil, err
			}
			subscs[x].Pair = subscs[x].Pair.Format(pairFormat)
			if subscs[x].Asset == asset.Futures {
				instrumentID = d.formatFuturesTradablePair(subscs[x].Pair.Format(pairFormat))
			} else {
				instrumentID = subscs[x].Pair.String()
			}
		}
		switch subscs[x].Channel {
		case announcementsChannel,
			userAccessLogChannel,
			platformStateChannel,
			platformStatePublicMethodsStateChannel,
			userLockChannel:
			sub.Params["channels"] = []string{subscs[x].Channel}
		case orderbookChannel:
			if subscs[x].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			intervalString, err := d.GetResolutionFromInterval(subscs[x].Interval)
			if err != nil {
				return nil, err
			}
			group, okay := subscs[x].Params["group"].(string)
			if !okay {
				sub.Params["channels"] = []string{orderbookChannel + "." + instrumentID + "." + intervalString}
				break
			}
			depth, okay := subscs[x].Params["depth"].(string)
			if !okay {
				sub.Params["channels"] = []string{orderbookChannel + "." + instrumentID + "." + intervalString}
				break
			}
			sub.Params["channels"] = []string{orderbookChannel + "." + instrumentID + "." + group + "." + depth + "." + intervalString}
		case chartTradesChannel:
			if subscs[x].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			resolution, okay := subscs[x].Params["resolution"].(string)
			if !okay {
				resolution = "1D"
			}
			sub.Params["channels"] = []string{chartTradesChannel + "." + d.formatFuturesTradablePair(subscs[x].Pair) + "." + resolution}
		case priceIndexChannel,
			priceRankingChannel,
			priceStatisticsChannel,
			volatilityIndexChannel,
			markPriceOptionsChannel,
			estimatedExpirationPriceChannel:
			indexName, okay := subscs[x].Params["index_name"].(string)
			if !okay {
				return nil, errUnsupportedIndexName
			}
			sub.Params["channels"] = []string{subscs[x].Channel + "." + indexName}
		case instrumentStateChannel:
			kind := d.GetAssetKind(subscs[x].Asset)
			currencyCode := getValidatedCurrencyCode(subscs[x].Pair)
			sub.Params["channels"] = []string{"instrument.state." + kind + "." + currencyCode}
		case rawUsersOrdersKindCurrencyChannel:
			kind := d.GetAssetKind(subscs[x].Asset)
			currencyCode := getValidatedCurrencyCode(subscs[x].Pair)
			sub.Params["channels"] = []string{"user.orders." + kind + "." + currencyCode + ".raw"}
		case quoteChannel,
			incrementalTickerChannel:
			if subscs[x].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			sub.Params["channels"] = []string{subscs[x].Channel + "." + instrumentID}
		case rawUserOrdersChannel:
			if subscs[x].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			sub.Params["channels"] = []string{"user.orders." + instrumentID + ".raw"}
		case requestForQuoteChannel,
			userMMPTriggerChannel,
			userPortfolioChannel:
			currencyCode := getValidatedCurrencyCode(subscs[x].Pair)
			sub.Params["channels"] = []string{subscs[x].Channel + "." + currencyCode}
		case tradesChannel,
			userChangesInstrumentsChannel,
			userOrdersWithIntervalChannel,
			tickerChannel,
			perpetualChannel,
			userTradesChannelByInstrument:
			if subscs[x].Pair.IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
			if subscs[x].Interval.Duration() == 0 {
				sub.Params["channels"] = []string{subscs[x].Channel + instrumentID}
				continue
			}
			intervalString, err := d.GetResolutionFromInterval(subscs[x].Interval)
			if err != nil {
				return nil, err
			}
			sub.Params["channels"] = []string{subscs[x].Channel + instrumentID + "." + intervalString}
		case userChangesCurrencyChannel,
			tradesWithKindChannel,
			rawUsersOrdersWithKindCurrencyAndIntervalChannel,
			userTradesByKindCurrencyAndIntervalChannel:
			kind := d.GetAssetKind(subscs[x].Asset)
			currencyCode := getValidatedCurrencyCode(subscs[x].Pair)
			if subscs[x].Interval.Duration() == 0 {
				sub.Params["channels"] = []string{subscs[x].Channel + "." + kind + "." + currencyCode}
				continue
			}
			intervalString, err := d.GetResolutionFromInterval(subscs[x].Interval)
			if err != nil {
				return nil, err
			}
			sub.Params["channels"] = []string{subscs[x].Channel + "." + kind + "." + currencyCode + "." + intervalString}
		default:
			return nil, errUnsupportedChannel
		}
		subscriptionPayloads[x] = sub
	}
	return filterSubscriptionPayloads(subscriptionPayloads), nil
}

// Subscribe sends a websocket message to receive data from the channel
func (d *Deribit) Subscribe(channelsToSubscribe []subscription.Subscription) error {
	return d.handleSubscription("subscribe", channelsToSubscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (d *Deribit) Unsubscribe(channelsToUnsubscribe []subscription.Subscription) error {
	return d.handleSubscription("unsubscribe", channelsToUnsubscribe)
}

func filterSubscriptionPayloads(subscription []WsSubscriptionInput) []WsSubscriptionInput {
	newSubscriptionsMap := map[string]bool{}
	newSubscs := make([]WsSubscriptionInput, 0, len(subscription))
	for x := range subscription {
		if len(subscription[x].Params["channels"]) == 0 {
			continue
		}
		if !newSubscriptionsMap[subscription[x].Params["channels"][0]] {
			newSubscriptionsMap[subscription[x].Params["channels"][0]] = true
			newSubscs = append(newSubscs, subscription[x])
		}
	}
	return newSubscs
}

func (d *Deribit) handleSubscription(operation string, channels []subscription.Subscription) error {
	payloads, err := d.generatePayloadFromSubscriptionInfos(operation, channels)
	if err != nil {
		return err
	}
	for x := range payloads {
		data, err := d.Websocket.Conn.SendMessageReturnResponse(payloads[x].ID, payloads[x])
		if err != nil {
			return err
		}
		var response wsSubscriptionResponse
		err = json.Unmarshal(data, &response)
		if err != nil {
			return fmt.Errorf("%v %v", d.Name, err)
		}
		if payloads[x].ID == response.ID && len(response.Result) == 0 {
			log.Errorf(log.ExchangeSys, "subscription to channel %s was not successful", payloads[x].Params["channels"][0])
		}
	}
	return nil
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
