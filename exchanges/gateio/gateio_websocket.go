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
	spotOrderbookChannel,
}

var fetchedCurrencyPairSnapshotOrderbook = make(map[string]bool)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	err := g.CurrencyPairs.IsAssetEnabled(asset.Spot)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = g.Websocket.AssetTypeWebsockets[asset.Spot].Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		Channel: spotPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.AssetTypeWebsockets[asset.Spot].Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 15,
		Message:     pingMessage,
		MessageType: websocket.TextMessage,
	})
	g.Websocket.Wg.Add(1)
	go g.wsReadConnData()
	subscriptions, _ := g.GenerateDefaultSubscriptions()
	return g.Subscribe(subscriptions)
	// return nil
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
		resp := g.Websocket.AssetTypeWebsockets[asset.Spot].Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := g.wsHandleData(resp.Raw)
		if err != nil {
			g.Websocket.DataHandler <- err
		}
	}
}

func (g *Gateio) wsHandleData(respRaw []byte) error {
	var result WsResponse
	var eventResponse WsEventResponse
	err := json.Unmarshal(respRaw, &eventResponse)
	if err == nil &&
		(eventResponse.Result != nil || eventResponse.Error != nil) &&
		(eventResponse.Event == "subscribe" || eventResponse.Event == "unsubscribe") {
		if !g.Websocket.Match.IncomingWithData(eventResponse.ID, respRaw) {
			return fmt.Errorf("couldn't match subscription message with ID: %d", eventResponse.ID)
		}
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
	case spotPongChannel:
	default:
		g.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: g.Name + stream.UnhandledMessage + string(respRaw),
		}
		return errors.New(stream.UnhandledMessage)
	}
	return nil
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
	tickerPrice := ticker.Price{
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
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(currencyPair)
	if assetPairEnabled[asset.Spot] {
		g.Websocket.DataHandler <- &tickerPrice
	}
	if assetPairEnabled[asset.Margin] {
		marginTicker := tickerPrice
		marginTicker.AssetType = asset.Margin
		g.Websocket.DataHandler <- &marginTicker
	}
	if assetPairEnabled[asset.CrossMargin] {
		crossMarginTicker := tickerPrice
		crossMarginTicker.AssetType = asset.CrossMargin
		g.Websocket.DataHandler <- &crossMarginTicker
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
	spotTradeData := trade.Data{
		Timestamp:    time.UnixMicro(int64(tradeData.CreateTimeMs * 1e3)), // the timestamp data is coming as a floating number.
		CurrencyPair: currencyPair,
		AssetType:    asset.Spot,
		Exchange:     g.Name,
		Price:        tradeData.Price,
		Amount:       tradeData.Amount,
		Side:         side,
		TID:          strconv.FormatInt(tradeData.ID, 10),
	}
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(currencyPair)
	if assetPairEnabled[asset.Spot] {
		err = trade.AddTradesToBuffer(g.Name, spotTradeData)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.Margin] {
		marginTradeData := spotTradeData
		marginTradeData.AssetType = asset.Margin
		err = trade.AddTradesToBuffer(g.Name, marginTradeData)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.CrossMargin] {
		crossMarginTradeData := spotTradeData
		crossMarginTradeData.AssetType = asset.CrossMargin
		err = trade.AddTradesToBuffer(g.Name, crossMarginTradeData)
		if err != nil {
			return err
		}
	}
	return nil
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
	spotCandlestick := stream.KlineData{
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
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(currencyPair)
	if assetPairEnabled[asset.Spot] {
		g.Websocket.DataHandler <- spotCandlestick
	}
	if assetPairEnabled[asset.Margin] {
		marginCandlestick := spotCandlestick
		marginCandlestick.AssetType = asset.Margin
		g.Websocket.DataHandler <- marginCandlestick
	}
	if assetPairEnabled[asset.CrossMargin] {
		crossMarginCandlestick := spotCandlestick
		crossMarginCandlestick.AssetType = asset.CrossMargin
		g.Websocket.DataHandler <- crossMarginCandlestick
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
	update := new(WsOrderbookUpdate)
	response.Result = update
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(update.CurrencyPair)
	if err != nil {
		return err
	}
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(pair)
	if !fetchedCurrencyPairSnapshotOrderbook[update.CurrencyPair] {
		var orderbooks *orderbook.Base
		orderbooks, err = g.FetchOrderbook(context.Background(), pair, asset.Spot) // currency pair orderbook data for Spot, Margin, and Cross Margin is same
		if err != nil {
			return err
		}
		// TODO: handle orderbook update synchronisation
		for _, assetType := range []asset.Item{asset.Spot, asset.Margin, asset.CrossMargin} {
			if !assetPairEnabled[assetType] {
				continue
			}
			assetOrderbook := *orderbooks
			assetOrderbook.Asset = assetType
			err = g.Websocket.Orderbook.LoadSnapshot(&assetOrderbook)
			if err != nil {
				return err
			}
		}
		fetchedCurrencyPairSnapshotOrderbook[update.CurrencyPair] = true
	}
	updates := orderbook.Update{
		UpdateTime: update.UpdateTimeMs.Time(),
		Pair:       pair,
	}
	updates.Bids = make([]orderbook.Item, len(update.Bids))
	updates.Asks = make([]orderbook.Item, len(update.Asks))
	var price float64
	var amount float64
	for x := range updates.Asks {
		price, err = strconv.ParseFloat(update.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(update.Asks[x][1], 64)
		if err != nil {
			return err
		}
		updates.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range updates.Bids {
		price, err = strconv.ParseFloat(update.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(update.Bids[x][1], 64)
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
	if assetPairEnabled[asset.Spot] {
		updates.Asset = asset.Spot
		err = g.Websocket.Orderbook.Update(&updates)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.Margin] {
		marginUpdates := updates
		marginUpdates.Asset = asset.Margin
		err = g.Websocket.Orderbook.Update(&marginUpdates)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.CrossMargin] {
		crossMarginUpdate := updates
		crossMarginUpdate.Asset = asset.CrossMargin
		err = g.Websocket.Orderbook.Update(&crossMarginUpdate)
		if err != nil {
			return err
		}
	}
	return nil
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
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(pair)
	bases := orderbook.Base{
		Exchange:        g.Name,
		Pair:            pair,
		Asset:           asset.Spot,
		LastUpdated:     snapshot.UpdateTimeMs.Time(),
		LastUpdateID:    snapshot.LastUpdateID,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	bases.Bids = make([]orderbook.Item, len(snapshot.Bids))
	bases.Asks = make([]orderbook.Item, len(snapshot.Asks))
	var price float64
	var amount float64
	for x := range bases.Asks {
		price, err = strconv.ParseFloat(snapshot.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(snapshot.Asks[x][1], 64)
		if err != nil {
			return err
		}
		bases.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range bases.Bids {
		price, err = strconv.ParseFloat(snapshot.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(snapshot.Bids[x][1], 64)
		if err != nil {
			return err
		}
		bases.Bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	if assetPairEnabled[asset.Spot] {
		err = g.Websocket.Orderbook.LoadSnapshot(&bases)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.Margin] {
		marginBases := bases
		marginBases.Asset = asset.Margin
		err = g.Websocket.Orderbook.LoadSnapshot(&marginBases)
		if err != nil {
			return err
		}
	}
	if assetPairEnabled[asset.CrossMargin] {
		crossMarginBases := bases
		crossMarginBases.Asset = asset.CrossMargin
		err = g.Websocket.Orderbook.LoadSnapshot(&crossMarginBases)
		if err != nil {
			return err
		}
	}
	return nil
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
			Date:           resp.Result[x].CreateTimeMs.Time(),
			LastUpdated:    resp.Result[x].UpdateTimeMs.Time(),
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
			Timestamp:    resp.Result[x].CreateTimeMicroS,
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
	var crossMarginPairs, spotPairs currency.Pairs
	marginPairs, err := g.GetEnabledPairs(asset.Margin)
	if err != nil {
		return nil, err
	}
	crossMarginPairs, err = g.GetEnabledPairs(asset.CrossMargin)
	if err != nil {
		return nil, err
	}
	spotPairs, err = g.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for i := range channelsToSubscribe {
		switch channelsToSubscribe[i] {
		case marginBalancesChannel:
			pairs = marginPairs
		case crossMarginBalanceChannel:
			pairs = crossMarginPairs
		default:
			pairs = spotPairs
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
			if spotTradesChannel == channelsToSubscribe[i] {
				if !g.IsSaveTradeDataEnabled() {
					continue
				}
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Spot)
			if err != nil {
				return nil, err
			}

			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
				Asset:    asset.Spot,
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
	var errs error
	for k := range payloads {
		result, err := g.Websocket.AssetTypeWebsockets[asset.Spot].Conn.SendMessageReturnResponse(payloads[k].ID, payloads[k])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		var resp WsEventResponse
		if err = json.Unmarshal(result, &resp); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			if resp.Error != nil && resp.Error.Code != 0 {
				errs = common.AppendError(errs, fmt.Errorf("error while %s to channel %s error code: %d message: %s", payloads[k].Event, payloads[k].Channel, resp.Error.Code, resp.Error.Message))
				continue
			}
			if payloads[k].Event == "subscribe" {
				g.Websocket.AssetTypeWebsockets[asset.Spot].AddSuccessfulSubscriptions(channelsToSubscribe[k])
			} else {
				g.Websocket.AssetTypeWebsockets[asset.Spot].RemoveSuccessfulUnsubscriptions(channelsToSubscribe[k])
			}
		}
	}
	return errs
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
			return nil, err
		}
	}
	var intervalString string
	payloads := make([]WsInput, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		var auth *WsAuthInput
		timestamp := time.Now()
		channelsToSubscribe[i].Currency.Delimiter = currency.UnderscoreDelimiter
		params := []string{channelsToSubscribe[i].Currency.String()}
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
		switch channelsToSubscribe[i].Channel {
		case spotUserTradesChannel,
			spotBalancesChannel,
			marginBalancesChannel,
			spotFundingBalanceChannel,
			crossMarginBalanceChannel,
			crossMarginLoanChannel:
			if !g.Websocket.CanUseAuthenticatedEndpoints() {
				continue
			}
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
		case spotOrderbookUpdateChannel:
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
			ID:      g.Websocket.AssetTypeWebsockets[asset.Spot].Conn.GenerateMessageID(false),
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

func (g *Gateio) listOfAssetsCurrencyPairEnabledFor(cp currency.Pair) map[asset.Item]bool {
	assetTypes := g.CurrencyPairs.GetAssetTypes(true)
	// we need this all asset types on the map even if their value is false
	assetPairEnabled := map[asset.Item]bool{asset.Spot: false, asset.Options: false, asset.Futures: false, asset.CrossMargin: false, asset.Margin: false, asset.DeliveryFutures: false}
	for i := range assetTypes {
		pairs, err := g.GetEnabledPairs(assetTypes[i])
		if err != nil {
			continue
		}
		assetPairEnabled[assetTypes[i]] = pairs.Contains(cp, true)
	}
	return assetPairEnabled
}
