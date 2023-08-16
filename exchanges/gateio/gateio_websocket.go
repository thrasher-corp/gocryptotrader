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
	spotOrderbookTickerChannel,
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
	err = g.Websocket.Conn.Dial(&websocket.Dialer{}, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{Channel: spotPingChannel})
	if err != nil {
		return err
	}
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 15,
		Message:     pingMessage,
		MessageType: websocket.TextMessage,
	})
	g.Websocket.Wg.Add(1)
	go g.wsReadConnData()
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

func (g *Gateio) wsHandleData(respRaw []byte) error {
	var push WsResponse
	err := json.Unmarshal(respRaw, &push)
	if err != nil {
		return err
	}

	if push.Event == "subscribe" || push.Event == "unsubscribe" {
		if !g.Websocket.Match.IncomingWithData(push.ID, respRaw) {
			return fmt.Errorf("couldn't match subscription message with ID: %d", push.ID)
		}
		return nil
	}

	switch push.Channel { // TODO: Convert function params below to only use push.Result
	case spotTickerChannel:
		return g.processTicker(push.Result, push.Time)
	case spotTradesChannel:
		return g.processTrades(push.Result)
	case spotCandlesticksChannel:
		return g.processCandlestick(push.Result)
	case spotOrderbookTickerChannel:
		return g.processOrderbookTicker(push.Result)
	case spotOrderbookUpdateChannel:
		return g.processOrderbookUpdate(push.Result)
	case spotOrderbookChannel:
		return g.processOrderbookSnapshot(push.Result)
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

func (g *Gateio) processTicker(incoming []byte, pushTime int64) error {
	var data WsTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	tickerPrice := ticker.Price{
		ExchangeName: g.Name,
		Volume:       data.BaseVolume,
		QuoteVolume:  data.QuoteVolume,
		High:         data.High24H,
		Low:          data.Low24H,
		Last:         data.Last,
		Bid:          data.HighestBid,
		Ask:          data.LowestAsk,
		AssetType:    asset.Spot,
		Pair:         data.CurrencyPair,
		LastUpdated:  time.Unix(pushTime, 0),
	}
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(data.CurrencyPair)
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

func (g *Gateio) processTrades(incoming []byte) error {
	saveTradeData := g.IsSaveTradeDataEnabled()
	if !saveTradeData && !g.IsTradeFeedEnabled() {
		return nil
	}

	var data WsTrade
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}

	side, err := order.StringToOrderSide(data.Side)
	if err != nil {
		return err
	}
	tData := trade.Data{
		Timestamp:    data.CreateTimeMs.Time(),
		CurrencyPair: data.CurrencyPair,
		AssetType:    asset.Spot,
		Exchange:     g.Name,
		Price:        data.Price,
		Amount:       data.Amount,
		Side:         side,
		TID:          strconv.FormatInt(data.ID, 10),
	}

	for _, assetType := range []asset.Item{asset.Spot, asset.Margin, asset.CrossMargin} {
		if g.listOfAssetsCurrencyPairEnabledFor(data.CurrencyPair)[assetType] {
			tData.AssetType = assetType
			if err := g.Websocket.Trade.Update(saveTradeData, tData); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Gateio) processCandlestick(incoming []byte) error {
	var data WsCandlesticks
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	icp := strings.Split(data.NameOfSubscription, currency.UnderscoreDelimiter)
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
		StartTime:  time.Unix(data.Timestamp, 0),
		Interval:   icp[0],
		OpenPrice:  data.OpenPrice,
		ClosePrice: data.ClosePrice,
		HighPrice:  data.HighestPrice,
		LowPrice:   data.LowestPrice,
		Volume:     data.TotalVolume,
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

func (g *Gateio) processOrderbookTicker(incoming []byte) error {
	var data WsOrderbookTickerData
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}

	return g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Exchange:    g.Name,
		Pair:        data.CurrencyPair,
		Asset:       asset.Spot,
		LastUpdated: time.UnixMilli(data.UpdateTimeMS),
		Bids:        []orderbook.Item{{Price: data.BestBidPrice, Amount: data.BestBidAmount}},
		Asks:        []orderbook.Item{{Price: data.BestAskPrice, Amount: data.BestAskAmount}},
	})
}

func (g *Gateio) processOrderbookUpdate(incoming []byte) error {
	var data WsOrderbookUpdate
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(data.CurrencyPair)
	if !fetchedCurrencyPairSnapshotOrderbook[data.CurrencyPair.String()] {
		var orderbooks *orderbook.Base
		orderbooks, err = g.FetchOrderbook(context.Background(), data.CurrencyPair, asset.Spot) // currency pair orderbook data for Spot, Margin, and Cross Margin is same
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
		fetchedCurrencyPairSnapshotOrderbook[data.CurrencyPair.String()] = true
	}
	updates := orderbook.Update{
		UpdateTime: data.UpdateTimeMs.Time(),
		Pair:       data.CurrencyPair,
	}
	updates.Asks = make([]orderbook.Item, len(data.Asks))
	for x := range data.Asks {
		updates.Asks[x].Price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return err
		}
		updates.Asks[x].Amount, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return err
		}
	}
	updates.Bids = make([]orderbook.Item, len(data.Bids))
	for x := range data.Bids {
		updates.Bids[x].Price, err = strconv.ParseFloat(data.Bids[x][0], 64)
		if err != nil {
			return err
		}
		updates.Bids[x].Amount, err = strconv.ParseFloat(data.Bids[x][1], 64)
		if err != nil {
			return err
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

func (g *Gateio) processOrderbookSnapshot(incoming []byte) error {
	var data WsOrderbookSnapshot
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	assetPairEnabled := g.listOfAssetsCurrencyPairEnabledFor(data.CurrencyPair)
	bases := orderbook.Base{
		Exchange:        g.Name,
		Pair:            data.CurrencyPair,
		Asset:           asset.Spot,
		LastUpdated:     data.UpdateTimeMs.Time(),
		LastUpdateID:    data.LastUpdateID,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	bases.Asks = make([]orderbook.Item, len(data.Asks))
	for x := range data.Asks {
		bases.Asks[x].Price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return err
		}
		bases.Asks[x].Amount, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return err
		}
	}
	bases.Bids = make([]orderbook.Item, len(data.Bids))
	for x := range data.Bids {
		bases.Bids[x].Price, err = strconv.ParseFloat(data.Bids[x][0], 64)
		if err != nil {
			return err
		}
		bases.Bids[x].Amount, err = strconv.ParseFloat(data.Bids[x][1], 64)
		if err != nil {
			return err
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
			Pair:           resp.Result[x].CurrencyPair,
			Cost:           resp.Result[x].Fee,
			AssetType:      a,
			Price:          resp.Result[x].Price,
			ExecutedAmount: resp.Result[x].Amount - resp.Result[x].Left.Float64(),
			Date:           resp.Result[x].CreateTimeMs.Time(),
			LastUpdated:    resp.Result[x].UpdateTimeMs.Time(),
		}
	}
	g.Websocket.DataHandler <- details
	return nil
}

func (g *Gateio) processUserPersonalTrades(data []byte) error {
	if !g.IsFillsFeedEnabled() {
		return nil
	}

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
		side, err := order.StringToOrderSide(resp.Result[x].Side)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			Exchange:     g.Name,
			CurrencyPair: resp.Result[x].CurrencyPair,
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

	if g.IsSaveTradeDataEnabled() || g.IsTradeFeedEnabled() {
		channelsToSubscribe = append(channelsToSubscribe, spotTradesChannel)
	}

	var subscriptions []stream.ChannelSubscription
	var err error
	for i := range channelsToSubscribe {
		var pairs []currency.Pair
		var assetType asset.Item
		switch channelsToSubscribe[i] {
		case marginBalancesChannel:
			assetType = asset.Margin
			pairs, err = g.GetEnabledPairs(asset.Margin)
		case crossMarginBalanceChannel:
			assetType = asset.CrossMargin
			pairs, err = g.GetEnabledPairs(asset.CrossMargin)
		default:
			assetType = asset.Spot
			pairs, err = g.GetEnabledPairs(asset.Spot)
		}
		if err != nil {
			if errors.Is(err, asset.ErrNotEnabled) {
				continue // Skip if asset is not enabled.
			}
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
				params["interval"] = kline.HundredMilliseconds
			}

			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Spot)
			if err != nil {
				return nil, err
			}

			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Asset:    assetType,
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
	var errs error
	for k := range payloads {
		result, err := g.Websocket.Conn.SendMessageReturnResponse(payloads[k].ID, payloads[k])
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
				g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
			} else {
				g.Websocket.RemoveSuccessfulUnsubscriptions(channelsToSubscribe[k])
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
	var batch *[]string
	var intervalString string
	payloads := make([]WsInput, 0, len(channelsToSubscribe))
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

		payload := WsInput{
			ID:      g.Websocket.Conn.GenerateMessageID(false),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		}

		if channelsToSubscribe[i].Channel == "spot.book_ticker" {
			// To get all orderbook assets subscribed it needs to be batched and
			// only spot.book_ticker can be batched, if not it will take about
			// half an hour for initital sync.
			if batch != nil {
				*batch = append(*batch, params...)
			} else {
				// Sets up pointer to the field for the outbound payload.
				payloads = append(payloads, payload)
				batch = &payloads[len(payloads)-1].Payload
			}
			continue
		}
		payloads = append(payloads, payload)
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
