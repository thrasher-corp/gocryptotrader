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
	spotTradesChannel,
	spotOrderbookChannel,
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
		ID: g.Websocket.Conn.GenerateMessageID(false),
		Time: func() int64 {
			return time.Now().Unix()
		}(),
		Channel: spotPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket: true,
		Delay:     time.Second * 5,
		Message:   pingMessage,
	})
	g.Websocket.Wg.Add(1)
	go g.wsReadData()
	if g.IsWebsocketAuthenticationSupported() {
		if err != nil {
			g.Websocket.DataHandler <- err
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		} else {
			subscriptions, err := g.GenerateDefaultSubscriptions()
			if err != nil {
				println(err.Error())
				return err
			}
			go g.Subscribe(
				subscriptions,
			)
		}
	}
	go g.WsChannelsMultiplexer.Run()
	return nil
}

func (g *Gateio) generateSpotWsSignature(secret, event, channel string, dtime time.Time) (string, error) {
	msg := fmt.Sprintf("channel=%s&event=%s&time=%d", channel, event, dtime.Unix())
	mac := hmac.New(sha512.New, []byte(secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (g *Gateio) wsServerSignIn(ctx context.Context) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := time.Now()
	sigTemp, err := g.generateSpotWsSignature(creds.Secret, "subscribe", "spot.balances", timestamp)
	if err != nil {
		return err
	}
	signinWsRequest := WsInput{
		Time:    timestamp.Unix(),
		ID:      g.Websocket.Conn.GenerateMessageID(false),
		Channel: "spot.balances",
		Event:   "subscribe",
		Auth: &WsAuthInput{
			Method: "api_key",
			Key:    creds.Key,
			Sign:   sigTemp,
		},
	}
	err = g.Websocket.Conn.SendJSONMessage( /*signinWsRequest.ID, */ signinWsRequest)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (g *Gateio) wsReadData() {
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
		g.Websocket.DataHandler <- err
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
		Timestamp:    time.UnixMilli(int64(tradeData.CreateTimeMs)),
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
	bids := make([]orderbook.Item, len(update.Bids))
	asks := make([]orderbook.Item, len(update.Asks))
	for x := range asks {
		price, err := strconv.ParseFloat(update.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[x][1], 64)
		if err != nil {
			return err
		}
		asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range bids {
		price, err := strconv.ParseFloat(update.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[x][1], 64)
		if err != nil {
			return err
		}
		bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	if len(asks) == 0 && len(bids) == 0 {
		return nil
	}
	return g.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateTime: time.UnixMilli(update.UpdateTimeMs),
		Asks:       asks,
		Bids:       bids,
		Pair:       pair,
		Asset:      asset.Spot,
		MaxDepth:   int(update.LastOrderbookUpdatedID - update.FirstOrderbookUpdatedID),
	})
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
	bids := make([]orderbook.Item, len(snapshot.Bids))
	asks := make([]orderbook.Item, len(snapshot.Asks))
	for x := range asks {
		price, err := strconv.ParseFloat(snapshot.Asks[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(snapshot.Asks[x][1], 64)
		if err != nil {
			return err
		}
		asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	for x := range bids {
		price, err := strconv.ParseFloat(snapshot.Bids[x][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(snapshot.Bids[x][1], 64)
		if err != nil {
			return err
		}
		bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	return g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Asks:            asks,
		Bids:            bids,
		Asset:           asset.Spot,
		Exchange:        g.Name,
		Pair:            pair,
		LastUpdated:     time.UnixMilli(snapshot.UpdateTimeMs),
		VerifyOrderbook: g.CanVerifyOrderbook,
	})
}

func (g *Gateio) processSpotOrders(data []byte) error {
	type response struct {
		Time    int64         `json:"time"`
		Channel string        `json:"channel"`
		Event   string        `json:"event"`
		Result  []WsSpotOrder `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
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
		g.Websocket.DataHandler <- &order.Detail{
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
	return nil
}

func (g *Gateio) processUserPersonalTrades(data []byte) error {
	type response struct {
		Time    int64                 `json:"time"`
		Channel string                `json:"channel"`
		Event   string                `json:"event"`
		Result  []WsUserPersonalTrade `json:"result"`
	}
	var resp response
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
			Timestamp:    time.UnixMilli(int64(resp.Result[x].CreateTimeMs)),
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
	type response struct {
		Time    int64           `json:"time"`
		Channel string          `json:"channel"`
		Event   string          `json:"event"`
		Result  []WsSpotBalance `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		g.Websocket.DataHandler <- account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Spot,
			Amount:   resp.Result[x].Available,
		}
	}
	return nil
}

func (g *Gateio) processMarginBalances(data []byte) error {
	type response struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsMarginBalance `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		g.Websocket.DataHandler <- account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Margin,
			Amount:   resp.Result[x].Available,
		}
	}
	return nil
}

func (g *Gateio) processFundingBalances(data []byte) error {
	type response struct {
		Time    int64              `json:"time"`
		Channel string             `json:"channel"`
		Event   string             `json:"event"`
		Result  []WsFundingBalance `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- resp
	return nil
}

func (g *Gateio) processCrossMarginBalance(data []byte) error {
	type response struct {
		Time    int64                  `json:"time"`
		Channel string                 `json:"channel"`
		Event   string                 `json:"event"`
		Result  []WsCrossMarginBalance `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		code := currency.NewCode(resp.Result[x].Currency)
		g.Websocket.DataHandler <- account.Change{
			Exchange: g.Name,
			Currency: code,
			Asset:    asset.Margin,
			Amount:   resp.Result[x].Available,
		}
	}
	return nil
}

func (g *Gateio) processCrossMarginLoans(data []byte) error {
	type response struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  WsCrossMarginLoan `json:"result"`
	}
	var resp response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- resp
	return nil
}

// GenerateDefaultSubscriptions returns default subscriptions
func (g *Gateio) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		defaultSubscriptions = append(defaultSubscriptions, []string{
			crossMarginBalanceChannel,
			marginBalancesChannel,
			spotBalancesChannel}...)
	}
	var subscriptions []stream.ChannelSubscription
	var pairs []currency.Pair
	var err error
	for i := range defaultSubscriptions {
		if defaultSubscriptions[i] == marginBalancesChannel {
			pairs, err = g.GetEnabledPairs(asset.Margin)
		} else if defaultSubscriptions[i] == crossMarginBalanceChannel {
			pairs, err = g.GetEnabledPairs(asset.CrossMargin)
		} else {
			pairs, err = g.GetEnabledPairs(asset.Spot)
		}
		if err != nil {
			return nil, err
		}
		for j := range pairs {
			params := make(map[string]interface{})
			if strings.EqualFold(defaultSubscriptions[i], spotOrderbookChannel) {
				params["level"] = 5
				params["interval"] = kline.OneMin
			} else if strings.EqualFold(defaultSubscriptions[i], spotCandlesticksChannel) {
				params["interval"] = kline.FiveMin
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Spot)
			if err != nil {
				return nil, err
			}
			sub := stream.ChannelSubscription{
				Channel:  defaultSubscriptions[i],
				Currency: fpair.Upper(),
				Params:   params,
			}
			if defaultSubscriptions[i] == spotCandlesticksChannel {
				value, _ := json.Marshal(sub)
				println(string(value))
			}

			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultSubscriptions[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (g *Gateio) handleSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generatePayload(channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs common.Errors
	for k := range payloads {
		payloads[k].Event = event
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
		ticker := time.NewTicker(time.Second * 3)
	receive:
		for {
			select {
			case resp := <-channel:
				if resp.Result.Status != "success" {
					errs = append(errs, fmt.Errorf("%s websocket connection: timeout waiting for response with and subscription: %v",
						g.Name,
						payloads[k].Channel))
					break receive
				}
				g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
				g.WsChannelsMultiplexer.Unregister <- strconv.FormatInt(payloads[k].ID, 10)
				break receive
			case <-ticker.C:
				ticker.Stop()
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

func (g *Gateio) generatePayload(channelsToSubscribe []stream.ChannelSubscription) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	payloads := make([]WsInput, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		params := []string{channelsToSubscribe[i].Currency.String()}
		if strings.EqualFold(channelsToSubscribe[i].Channel, spotOrderbookChannel) {
			params = append(params,
				strconv.Itoa(channelsToSubscribe[i].Params["level"].(int)),
				g.GetIntervalString(channelsToSubscribe[i].Params["interval"].(kline.Interval)))
		} else if strings.EqualFold(channelsToSubscribe[i].Channel, spotCandlesticksChannel) {
			params = append([]string{g.GetIntervalString(channelsToSubscribe[i].Params["interval"].(kline.Interval))}, params...)
		}
		payloads[i] = WsInput{
			ID:      g.Websocket.Conn.GenerateMessageID(false),
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
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
