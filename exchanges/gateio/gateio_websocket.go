package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	gateioWebsocketEndpoint  = "wss://api.gateio.ws/ws/v4/"
	gateioWebsocketRateLimit = 120

	spotPingChannel            = "spot.ping"
	spotPongChannel            = "spot.pong"
	spotTickerChannel          = "spot.tickers"
	spotTradesChannel          = "spot.trades"
	spotCandlesticksChannel    = "spot.candlesticks"
	spotOrderbookTickerChannel = "spot.book_ticker"       // Best bid or ask price
	spotOrderbookUpdateChannel = "spot.order_book_update" // Changed order book levels
	spotOrderbookChannel       = "spot.order_book"        // Limited-Level Full Order Book Snapshot
	spotOrdersChannel          = "spot.orders"
	spotUserTradesChannel      = "spot.usertrades"
	spotBalancesChannel        = "spot.balances"
	marginBalancesChannel      = "spot.margin_balances"
	spotFundingBalanceChannel  = "spot.funding_balances"
	crossMarginBalanceChannel  = "spot.cross_balances"
	crossMarginLoanChannel     = "spot.cross_loan"
)

var defaultSubscriptions = []string{
	spotTickerChannel,
	spotCandlesticksChannel,
	spotTradesChannel,
	spotOrderbookChannel,
	spotOrderbookUpdateChannel,
}

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      g.Websocket.Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: spotPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 15,
		Message:     pingMessage,
		MessageType: websocket.PingMessage,
	})
	g.Websocket.Wg.Add(2)
	go g.wsReadConnData()
	go g.RunWsMultiplexer()
	return nil
}

func (g *Gateio) generateWsSignature(secret, event, channel string, dtime time.Time) (string, error) {
	msg := "channel=" + channel + "&event=" + event + "&time=" + strconv.FormatInt(dtime.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// wsReadConnData receives and passes on websocket messages for processing
func (g *Gateio) wsReadConnData() {
	defer g.Websocket.Wg.Done()
	for {
		resp := g.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := g.wsHandleData(resp.Raw)
		if err != nil {
			g.Websocket.DataHandler <- err
		}
	}
}

// wsReadData read coming messages thought the websocket connection and pass the data to wsHandleData for further process.
func (g *Gateio) wsReadData() {
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.ShutdownC:
			select {
			case resp := <-responseStream:
				err := g.wsHandleData(resp.Raw)
				if err != nil {
					select {
					case g.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", g.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseStream:
			err := g.wsHandleData(resp.Raw)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (g *Gateio) wsFunnelConnectionData(ws stream.Connection) {
	defer g.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- stream.Response{Raw: resp.Raw}
	}
}

func (g *Gateio) wsHandleData(respRaw []byte) error {
	var result WsResponse
	var eventResponse WsEventResponse
	err := json.Unmarshal(respRaw, &eventResponse)
	if err == nil &&
		(eventResponse.Result != nil || eventResponse.Error != nil) &&
		(eventResponse.Event == "subscribe" || eventResponse.Event == "unsubscribe") {
		g.WsChannelsMultiplexer.Message <- &eventResponse
		return nil
	}
	err = json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result.Channel {
	case spotTickerChannel:
		return g.processTicker(respRaw)
	case spotTradesChannel:
		return g.processTrades(respRaw)
	case spotCandlesticksChannel:
		return g.processCandlestick(respRaw)
	case spotOrderbookTickerChannel:
		return g.processOrderbookTicker(respRaw)
	case spotOrderbookUpdateChannel:
		return g.processOrderbookUpdate(respRaw)
	case spotOrderbookChannel:
		return g.processOrderbookSnapshot(respRaw)
	case spotOrdersChannel:
		return g.processSpotOrders(respRaw)
	case spotUserTradesChannel:
		return g.processUserPersonalTrades(respRaw)
	case spotBalancesChannel:
		return g.processSpotBalances(respRaw)
	case marginBalancesChannel:
		return g.processMarginBalances(respRaw)
	case spotFundingBalanceChannel:
		return g.processFundingBalances(respRaw)
	case crossMarginBalanceChannel:
		return g.processCrossMarginBalance(respRaw)
	case crossMarginLoanChannel:
		return g.processCrossMarginLoans(respRaw)
	case futuresTickersChannel:
		return g.processFuturesTickers(respRaw)
	case futuresTradesChannel:
		return g.processFuturesTrades(respRaw)
	case futuresOrderbookChannel:
		return g.processFuturesOrderbookSnapshot(result.Event, respRaw)
	case futuresOrderbookTickerChannel:
		return g.processFuturesOrderbookTicker(respRaw)
	case futuresOrderbookUpdateChannel:
		return g.processFuturesAndOptionsOrderbookUpdate(respRaw)
	case futuresCandlesticksChannel:
		return g.processFuturesCandlesticks(respRaw)
	case futuresOrdersChannel:
		return g.processFuturesOrdersPushData(respRaw)
	case futuresUserTradesChannel:
		return g.procesFuturesUserTrades(respRaw)
	case futuresLiquidatesChannel:
		return g.processFuturesLiquidatesNotification(respRaw)
	case futuresAutoDeleveragesChannel:
		return g.processFuturesAutoDeleveragesNotification(respRaw)
	case futuresAutoPositionCloseChannel:
		return g.processPositionCloseData(respRaw)
	case futuresBalancesChannel:
		return g.processBalancePushData(respRaw)
	case futuresReduceRiskLimitsChannel:
		return g.processFuturesReduceRiskLimitNotification(respRaw)
	case futuresPositionsChannel:
		return g.processFuturesPositionsNotification(respRaw)
	case futuresAutoOrdersChannel:
		return g.processFuturesAutoOrderPushData(respRaw)

		//  Options push data handlers
	case optionsContractTickersChannel:
		return g.processOptionsContractTickers(respRaw)
	case optionsUnderlyingTickersChannel:
		return g.processOptionsUnderlyingTicker(respRaw)
	case optionsTradesChannel,
		optionsUnderlyingTradesChannel:
		return g.processOptionsTradesPushData(respRaw)
	case optionsUnderlyingPriceChannel:
		return g.processOptionsUnderlyingPricePushData(respRaw)
	case optionsMarkPriceChannel:
		return g.processOptionsMarkPrice(respRaw)
	case optionsSettlementChannel:
		return g.processOptionsSettlementPushData(respRaw)
	case optionsContractsChannel:
		return g.processOptionsContractPushData(respRaw)
	case optionsContractCandlesticksChannel,
		optionsUnderlyingCandlesticksChannel:
		return g.processOptionsCandlestickPushData(respRaw)
	case optionsOrderbookChannel:
		return g.processOptionsOrderbookSnapshotPushData(result.Event, respRaw)
	case optionsOrderbookTickerChannel:
		return g.processOrderbookTickerPushData(respRaw)
	case optionsOrderbookUpdateChannel:
		return g.processFuturesAndOptionsOrderbookUpdate(respRaw)
	case optionsOrdersChannel:
		return g.processOptionsOrderPushData(respRaw)
	case optionsUserTradesChannel:
		return g.processOptionsUserTradesPushData(respRaw)
	case optionsLiquidatesChannel:
		return g.processOptionsLiquidatesPushData(respRaw)
	case optionsUserSettlementChannel:
		return g.processOptionsUsersPersonalSettlementsPushData(respRaw)
	case optionsPositionCloseChannel:
		return g.processPositionCloseData(respRaw)
	case optionsBalancesChannel:
		return g.processBalancePushData(respRaw)
	case optionsPositionsChannel:
		return g.processOptionsPositionPushData(respRaw)
	default:
		g.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: g.Name + stream.UnhandledMessage + string(respRaw),
		}
		return errors.New(stream.UnhandledMessage)
	}
}

func (g *Gateio) processTicker(data []byte) error {
	var response WsResponse
	tickerData := &WsTicker{}
	response.Result = tickerData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	currencyPair, err := currency.NewPairFromString(tickerData.CurrencyPair)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: g.Name,
		Volume:       tickerData.BaseVolume,
		QuoteVolume:  tickerData.QuoteVolume,
		High:         tickerData.High24H,
		Low:          tickerData.Low24H,
		Last:         tickerData.Last,
		Bid:          tickerData.HighestBid,
		Ask:          tickerData.LowestAsk,
		AssetType:    asset.Spot,
		Pair:         currencyPair,
		LastUpdated:  time.Unix(response.Time, 0),
	}
	return nil
}

func (g *Gateio) processTrades(data []byte) error {
	var response WsResponse
	tradeData := &WsTrade{}
	response.Result = tradeData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	currencyPair, err := currency.NewPairFromString(tradeData.CurrencyPair)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(tradeData.Side)
	if err != nil {
		return err
	}
	return trade.AddTradesToBuffer(g.Name, trade.Data{
		Timestamp:    time.UnixMicro(int64(tradeData.CreateTimeMs * 1e3)), // the timestamp data is coming as a floating number.
		CurrencyPair: currencyPair,
		AssetType:    asset.Spot,
		Exchange:     g.Name,
		Price:        tradeData.Price,
		Amount:       tradeData.Amount,
		Side:         side,
		TID:          strconv.FormatInt(tradeData.ID, 10),
	})
}

func (g *Gateio) processCandlestick(data []byte) error {
	var response WsResponse
	candleData := &WsCandlesticks{}
	response.Result = candleData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	icp := strings.Split(candleData.NameOfSubscription, currency.UnderscoreDelimiter)
	if len(icp) < 3 {
		return errors.New("malformed candlestick websocket push data")
	}
	currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- stream.KlineData{
		Pair:       currencyPair,
		AssetType:  asset.Spot,
		Exchange:   g.Name,
		StartTime:  time.Unix(candleData.Timestamp, 0),
		Interval:   icp[0],
		OpenPrice:  candleData.OpenPrice,
		ClosePrice: candleData.ClosePrice,
		HighPrice:  candleData.HighestPrice,
		LowPrice:   candleData.LowestPrice,
		Volume:     candleData.TotalVolume,
	}
	return nil
}

func (g *Gateio) processOrderbookTicker(data []byte) error {
	var response WsResponse
	tickerData := &WsOrderbookTickerData{}
	response.Result = tickerData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- tickerData
	return nil
}

func (g *Gateio) processOrderbookUpdate(data []byte) error {
	var response WsResponse
	update := &WsOrderbookUpdate{}
	response.Result = update
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(update.CurrencyPair)
	if err != nil {
		return err
	}
	updates := orderbook.Update{
		UpdateTime: time.UnixMilli(update.UpdateTimeMs),
		Pair:       pair,
		Asset:      asset.Spot,
	}
	updates.Bids = make([]orderbook.Item, len(update.Bids))
	updates.Asks = make([]orderbook.Item, len(update.Asks))
	for x := range updates.Asks {
		price, err := strconv.ParseFloat(update.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[x][1], 64)
		if err != nil {
			return err
		}
		updates.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range updates.Bids {
		price, err := strconv.ParseFloat(update.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[x][1], 64)
		if err != nil {
			return err
		}
		updates.Bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	if len(updates.Asks) == 0 && len(updates.Bids) == 0 {
		return nil
	}
	return g.Websocket.Orderbook.Update(&updates)
}

func (g *Gateio) processOrderbookSnapshot(data []byte) error {
	var response WsResponse
	snapshot := &WsOrderbookSnapshot{}
	response.Result = snapshot
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(snapshot.CurrencyPair)
	if err != nil {
		return err
	}
	bases := orderbook.Base{
		Asset:           asset.Spot,
		Exchange:        g.Name,
		Pair:            pair,
		LastUpdated:     time.UnixMilli(snapshot.UpdateTimeMs),
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	bases.Bids = make([]orderbook.Item, len(snapshot.Bids))
	bases.Asks = make([]orderbook.Item, len(snapshot.Asks))
	for x := range bases.Asks {
		price, err := strconv.ParseFloat(snapshot.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(snapshot.Asks[x][1], 64)
		if err != nil {
			return err
		}
		bases.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range bases.Bids {
		price, err := strconv.ParseFloat(snapshot.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(snapshot.Bids[x][1], 64)
		if err != nil {
			return err
		}
		bases.Bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	return g.Websocket.Orderbook.LoadSnapshot(&bases)
}

func (g *Gateio) processSpotOrders(data []byte) error {
	resp := struct {
		Time    int64         `json:"time"`
		Channel string        `json:"channel"`
		Event   string        `json:"event"`
		Result  []WsSpotOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	details := make([]order.Detail, len(resp.Result))
	for x := range resp.Result {
		pair, err := currency.NewPairFromString(resp.Result[x].CurrencyPair)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(resp.Result[x].Side)
		if err != nil {
			return err
		}
		orderType, err := order.StringToOrderType(resp.Result[x].Type)
		if err != nil {
			return err
		}
		a, err := asset.New(resp.Result[x].Account)
		if err != nil {
			return err
		}
		details[x] = order.Detail{
			Amount:         resp.Result[x].Amount,
			Exchange:       g.Name,
			OrderID:        resp.Result[x].ID,
			Side:           side,
			Type:           orderType,
			Pair:           pair,
			Cost:           resp.Result[x].Fee,
			AssetType:      a,
			Price:          resp.Result[x].Price,
			ExecutedAmount: resp.Result[x].Amount - resp.Result[x].Left,
			Date:           resp.Result[x].CreateTimeMs,
			LastUpdated:    resp.Result[x].UpdateTimeMs,
		}
	}
	g.Websocket.DataHandler <- details
	return nil
}

func (g *Gateio) processUserPersonalTrades(data []byte) error {
	resp := struct {
		Time    int64                 `json:"time"`
		Channel string                `json:"channel"`
		Event   string                `json:"event"`
		Result  []WsUserPersonalTrade `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(resp.Result))
	for x := range fills {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].CurrencyPair)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(resp.Result[x].Side)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			Timestamp:    time.UnixMicro(resp.Result[x].CreateTimeMicroS),
			Exchange:     g.Name,
			CurrencyPair: currencyPair,
			Side:         side,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      strconv.FormatInt(resp.Result[x].ID, 10),
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Amount,
		}
	}
	return g.Websocket.Fills.Update(fills...)
}

func (g *Gateio) processSpotBalances(data []byte) error {
	resp := struct {
		Time    int64           `json:"time"`
		Channel string          `json:"channel"`
		Event   string          `json:"event"`
		Result  []WsSpotBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	accountChanges := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		accountChanges[x] = account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Spot,
			Amount:   resp.Result[x].Available,
		}
	}
	g.Websocket.DataHandler <- accountChanges
	return nil
}

func (g *Gateio) processMarginBalances(data []byte) error {
	resp := struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsMarginBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	accountChange := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		accountChange[x] = account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Margin,
			Amount:   resp.Result[x].Available,
		}
	}
	g.Websocket.DataHandler <- accountChange
	return nil
}

func (g *Gateio) processFundingBalances(data []byte) error {
	resp := struct {
		Time    int64              `json:"time"`
		Channel string             `json:"channel"`
		Event   string             `json:"event"`
		Result  []WsFundingBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- resp
	return nil
}

func (g *Gateio) processCrossMarginBalance(data []byte) error {
	resp := struct {
		Time    int64                  `json:"time"`
		Channel string                 `json:"channel"`
		Event   string                 `json:"event"`
		Result  []WsCrossMarginBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	accountChanges := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		accountChanges[x] = account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Margin,
			Amount:   resp.Result[x].Available,
			Account:  resp.Result[x].User,
		}
	}
	g.Websocket.DataHandler <- accountChanges
	return nil
}

func (g *Gateio) processCrossMarginLoans(data []byte) error {
	resp := struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  WsCrossMarginLoan `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- resp
	return nil
}

// GenerateDefaultSubscriptions returns default subscriptions
func (g *Gateio) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channelsToSubscribe := defaultSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe, []string{
			crossMarginBalanceChannel,
			marginBalancesChannel,
			spotBalancesChannel}...)
	}
	var subscriptions []stream.ChannelSubscription
	var pairs []currency.Pair
	var err error
	for i := range channelsToSubscribe {
		switch channelsToSubscribe[i] {
		case marginBalancesChannel:
			pairs, err = g.GetEnabledPairs(asset.Margin)
		case crossMarginBalanceChannel:
			pairs, err = g.GetEnabledPairs(asset.CrossMargin)
		default:
			pairs, err = g.GetEnabledPairs(asset.Spot)
		}
		if err != nil {
			return nil, err
		}
		for j := range pairs {
			params := make(map[string]interface{})
			switch channelsToSubscribe[i] {
			case spotOrderbookChannel:
				params["level"] = 100
				params["interval"] = kline.HundredMilliseconds
			case spotCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case spotOrderbookUpdateChannel:
				params["interval"] = kline.ThousandMilliseconds
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Spot)
			if err != nil {
				return nil, err
			}

			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}

// handleSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generatePayload(event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs common.Errors
	for k := range payloads {
		err = g.Websocket.Conn.SendJSONMessage(payloads[k])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		channel := make(chan *WsEventResponse)
		g.WsChannelsMultiplexer.Register <- &wsChanReg{
			ID:   strconv.FormatInt(payloads[k].ID, 10),
			Chan: channel,
		}
		t := time.NewTicker(time.Second * 3)
	receive:
		for {
			select {
			case resp := <-channel:
				if resp.Result != nil && resp.Result.Status != "success" {
					errs = append(errs, fmt.Errorf("%s websocket connection: timeout waiting for response with and subscription: %v", g.Name, payloads[k].Channel))
					break receive
				} else if resp.Error != nil && resp.Error.Code != 0 {
					errs = append(errs, fmt.Errorf("error while %s to channel %s error code: %d message: %s", payloads[k].Event, payloads[k].Channel, resp.Error.Code, resp.Error.Message))
					break receive
				}
				g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
				g.WsChannelsMultiplexer.Unregister <- strconv.FormatInt(payloads[k].ID, 10)
				break receive
			case <-t.C:
				t.Stop()
				errs = append(errs, fmt.Errorf("%s websocket connection: timeout waiting for response with and subscription: %v",
					g.Name, payloads[k].Channel))
				g.WsChannelsMultiplexer.Unregister <- strconv.FormatInt(payloads[k].ID, 10)
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (g *Gateio) generatePayload(event string, channelsToSubscribe []stream.ChannelSubscription) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *account.Credentials
	var err error
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(context.TODO())
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	var intervalString string
	payloads := make([]WsInput, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		switch channelsToSubscribe[i].Channel {
		case optionsUnderlyingTickersChannel,
			optionsUnderlyingTradesChannel,
			optionsUnderlyingPriceChannel,
			optionsUnderlyingCandlesticksChannel:
			var uly string
			uly, err = g.GetUnderlyingFromCurrencyPair(channelsToSubscribe[i].Currency)
			if err != nil {
				return nil, err
			}
			params = append(params, uly)
		case optionsBalancesChannel:
			// options.balance channel does not require underlying or contract
		default:
			channelsToSubscribe[i].Currency.Delimiter = currency.UnderscoreDelimiter
			params = append(params, channelsToSubscribe[i].Currency.String())
		}
		switch channelsToSubscribe[i].Channel {
		case spotOrderbookChannel:
			interval, okay := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !okay {
				return nil, errors.New("invalid interval parameter")
			}
			level, okay := channelsToSubscribe[i].Params["level"].(int)
			if !okay {
				return nil, errors.New("invalid spot order level")
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(params,
				strconv.Itoa(level),
				intervalString,
			)
		case futuresOrderbookChannel:
			limit, ok := channelsToSubscribe[i].Params["limit"].(int)
			if !ok {
				return nil, fmt.Errorf("%w, invalid futures orderbook limit", orderbook.ErrOrderbookInvalid)
			}
			interval, ok := channelsToSubscribe[i].Params["interval"].(string)
			if !ok {
				return nil, fmt.Errorf("%w, missing futures orderbook interval", orderbook.ErrOrderbookInvalid)
			}
			params = append(params,
				strconv.Itoa(limit), interval)
		case optionsOrderbookChannel:
			accuracy, ok := channelsToSubscribe[i].Params["accuracy"].(string)
			if !ok {
				return nil, fmt.Errorf("%w, invalid options orderbook accuracy", orderbook.ErrOrderbookInvalid)
			}
			level, ok := channelsToSubscribe[i].Params["level"].(string)
			if !ok {
				return nil, fmt.Errorf("%w, invalid options orderbook level", orderbook.ErrOrderbookInvalid)
			}
			params = append(
				params,
				level,
				accuracy,
			)
		case futuresOrderbookUpdateChannel:
			interval, ok := channelsToSubscribe[i].Params["frequency"].(kline.Interval)
			if !ok {
				return nil, fmt.Errorf("%w, missing frequency for futures orderbook update", orderbook.ErrOrderbookInvalid)
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(
				params,
				intervalString)
			if value, ok := channelsToSubscribe[i].Params["level"].(string); ok {
				params = append(params, value)
			}
		case optionsOrderbookUpdateChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, fmt.Errorf("%w, missing options orderbook interval", orderbook.ErrOrderbookInvalid)
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(params,
				intervalString)
			if value, ok := channelsToSubscribe[i].Params["level"].(int); ok {
				params = append(params, strconv.Itoa(value))
			}
		case futuresCandlesticksChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, errors.New("missing futures candlesticks interval")
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(
				[]string{intervalString},
				params...)
		case optionsContractCandlesticksChannel, optionsUnderlyingCandlesticksChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, errors.New("missing options underlying candlesticks interval")
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(
				[]string{intervalString},
				params...)
		case spotCandlesticksChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, errors.New("missing spot candlesticks interval")
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(
				[]string{intervalString},
				params...)
		}
		if g.Websocket.CanUseAuthenticatedEndpoints() && (channelsToSubscribe[i].Channel == spotUserTradesChannel ||
			channelsToSubscribe[i].Channel == spotBalancesChannel ||
			channelsToSubscribe[i].Channel == marginBalancesChannel ||
			channelsToSubscribe[i].Channel == spotFundingBalanceChannel ||
			channelsToSubscribe[i].Channel == crossMarginBalanceChannel ||
			channelsToSubscribe[i].Channel == crossMarginLoanChannel ||
			channelsToSubscribe[i].Channel == optionsOrdersChannel ||
			channelsToSubscribe[i].Channel == optionsUserTradesChannel ||
			channelsToSubscribe[i].Channel == optionsLiquidatesChannel ||
			channelsToSubscribe[i].Channel == optionsUserSettlementChannel ||
			channelsToSubscribe[i].Channel == optionsPositionCloseChannel ||
			channelsToSubscribe[i].Channel == optionsBalancesChannel ||
			channelsToSubscribe[i].Channel == optionsPositionsChannel) {
			value, ok := channelsToSubscribe[i].Params["user"].(string)
			if ok {
				params = append(
					[]string{value},
					params...)
			}
			var sigTemp string
			sigTemp, err = g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp)
			if err != nil {
				return nil, err
			}
			auth = &WsAuthInput{
				Method: "api_key",
				Key:    creds.Key,
				Sign:   sigTemp,
			}
		} else if channelsToSubscribe[i].Channel == spotOrderbookUpdateChannel {
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, errors.New("missing spot orderbook interval")
			}
			intervalString, err = g.GetIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(params, intervalString)
		}
		payloads[i] = WsInput{
			ID:      g.Websocket.Conn.GenerateMessageID(false),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		}
	}
	return payloads, nil
}

// Subscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Subscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleSubscription("subscribe", channelsToUnsubscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleSubscription("unsubscribe", channelsToUnsubscribe)
}
