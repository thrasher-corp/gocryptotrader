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
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	gateioWebsocketEndpoint  = "wss://api.gateio.ws/ws/v4/"
	gateioWebsocketRateLimit = 120 * time.Millisecond

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

	subscribeEvent   = "subscribe"
	unsubscribeEvent = "unsubscribe"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.FiveMin},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds},
	{Enabled: true, Channel: spotBalancesChannel, Asset: asset.Spot, Authenticated: true},
	{Enabled: true, Channel: crossMarginBalanceChannel, Asset: asset.CrossMargin, Authenticated: true},
	{Enabled: true, Channel: marginBalancesChannel, Asset: asset.Margin, Authenticated: true},
	{Enabled: false, Channel: subscription.AllTradesChannel, Asset: asset.Spot},
}

var fetchedCurrencyPairSnapshotOrderbook = make(map[string]bool)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    spotTickerChannel,
	subscription.OrderbookChannel: spotOrderbookUpdateChannel,
	subscription.CandlesChannel:   spotCandlesticksChannel,
	subscription.AllTradesChannel: spotTradesChannel,
}

// WsConnectSpot initiates a websocket connection
func (g *Gateio) WsConnectSpot(ctx context.Context, conn stream.Connection) error {
	err := g.CurrencyPairs.IsAssetEnabled(asset.Spot)
	if err != nil {
		return err
	}
	err = conn.DialContext(ctx, &websocket.Dialer{}, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{Channel: spotPingChannel})
	if err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 15,
		Message:     pingMessage,
		MessageType: websocket.TextMessage,
	})
	return nil
}

// authenticateSpot sends an authentication message to the websocket connection
func (g *Gateio) authenticateSpot(ctx context.Context, conn stream.Connection) error {
	return g.websocketLogin(ctx, conn, "spot.login")
}

// websocketLogin authenticates the websocket connection
func (g *Gateio) websocketLogin(ctx context.Context, conn stream.Connection, channel string) error {
	if conn == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, conn)
	}

	if channel == "" {
		return errChannelEmpty
	}

	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}

	tn := time.Now().Unix()
	msg := "api\n" + channel + "\n" + "\n" + strconv.FormatInt(tn, 10)
	mac := hmac.New(sha512.New, []byte(creds.Secret))
	if _, err = mac.Write([]byte(msg)); err != nil {
		return err
	}
	signature := hex.EncodeToString(mac.Sum(nil))

	payload := WebsocketPayload{
		RequestID: strconv.FormatInt(conn.GenerateMessageID(false), 10),
		APIKey:    creds.Key,
		Signature: signature,
		Timestamp: strconv.FormatInt(tn, 10),
	}

	req := WebsocketRequest{Time: tn, Channel: channel, Event: "api", Payload: payload}

	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.Payload.RequestID, req)
	if err != nil {
		return err
	}

	var inbound WebsocketAPIResponse
	if err := json.Unmarshal(resp, &inbound); err != nil {
		return err
	}

	if inbound.Header.Status != "200" {
		var wsErr WebsocketErrors
		if err := json.Unmarshal(inbound.Data, &wsErr.Errors); err != nil {
			return err
		}
		return fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	return nil
}

func (g *Gateio) generateWsSignature(secret, event, channel string, t int64) (string, error) {
	msg := "channel=" + channel + "&event=" + event + "&time=" + strconv.FormatInt(t, 10)
	mac := hmac.New(sha512.New, []byte(secret))
	if _, err := mac.Write([]byte(msg)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// WsHandleSpotData handles spot data
func (g *Gateio) WsHandleSpotData(_ context.Context, respRaw []byte) error {
	var push WsResponse
	if err := json.Unmarshal(respRaw, &push); err != nil {
		return err
	}

	if push.RequestID != "" {
		return g.Websocket.Match.EnsureMatchWithData(push.RequestID, respRaw)
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		return g.Websocket.Match.EnsureMatchWithData(push.ID, respRaw)
	}

	switch push.Channel { // TODO: Convert function params below to only use push.Result
	case spotTickerChannel:
		return g.processTicker(push.Result, push.Time.Time())
	case spotTradesChannel:
		return g.processTrades(push.Result)
	case spotCandlesticksChannel:
		return g.processCandlestick(push.Result)
	case spotOrderbookTickerChannel:
		return g.processOrderbookTicker(push.Result, push.TimeMs.Time())
	case spotOrderbookUpdateChannel:
		return g.processOrderbookUpdate(push.Result, push.TimeMs.Time())
	case spotOrderbookChannel:
		return g.processOrderbookSnapshot(push.Result, push.TimeMs.Time())
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

func (g *Gateio) processTicker(incoming []byte, pushTime time.Time) error {
	var data WsTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	tickerPrice := ticker.Price{
		ExchangeName: g.Name,
		Volume:       data.BaseVolume.Float64(),
		QuoteVolume:  data.QuoteVolume.Float64(),
		High:         data.High24H.Float64(),
		Low:          data.Low24H.Float64(),
		Last:         data.Last.Float64(),
		Bid:          data.HighestBid.Float64(),
		Ask:          data.LowestAsk.Float64(),
		AssetType:    asset.Spot,
		Pair:         data.CurrencyPair,
		LastUpdated:  pushTime,
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
		Price:        data.Price.Float64(),
		Amount:       data.Amount.Float64(),
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
		StartTime:  data.Timestamp.Time(),
		Interval:   icp[0],
		OpenPrice:  data.OpenPrice.Float64(),
		ClosePrice: data.ClosePrice.Float64(),
		HighPrice:  data.HighestPrice.Float64(),
		LowPrice:   data.LowestPrice.Float64(),
		Volume:     data.TotalVolume.Float64(),
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

func (g *Gateio) processOrderbookTicker(incoming []byte, updatePushedAt time.Time) error {
	var data WsOrderbookTickerData
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}

	return g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Exchange:       g.Name,
		Pair:           data.CurrencyPair,
		Asset:          asset.Spot,
		LastUpdated:    data.UpdateTimeMS.Time(),
		UpdatePushedAt: updatePushedAt,
		Bids:           []orderbook.Tranche{{Price: data.BestBidPrice.Float64(), Amount: data.BestBidAmount.Float64()}},
		Asks:           []orderbook.Tranche{{Price: data.BestAskPrice.Float64(), Amount: data.BestAskAmount.Float64()}},
	})
}

func (g *Gateio) processOrderbookUpdate(incoming []byte, updatePushedAt time.Time) error {
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
		UpdateTime:     data.UpdateTimeMs.Time(),
		UpdatePushedAt: updatePushedAt,
		Pair:           data.CurrencyPair,
	}
	updates.Asks = make([]orderbook.Tranche, len(data.Asks))
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
	updates.Bids = make([]orderbook.Tranche, len(data.Bids))
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

func (g *Gateio) processOrderbookSnapshot(incoming []byte, updatePushedAt time.Time) error {
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
		UpdatePushedAt:  updatePushedAt,
		LastUpdateID:    data.LastUpdateID,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	bases.Asks = make([]orderbook.Tranche, len(data.Asks))
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
	bases.Bids = make([]orderbook.Tranche, len(data.Bids))
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
			Amount:         resp.Result[x].Amount.Float64(),
			Exchange:       g.Name,
			OrderID:        resp.Result[x].ID,
			Side:           side,
			Type:           orderType,
			Pair:           resp.Result[x].CurrencyPair,
			Cost:           resp.Result[x].Fee.Float64(),
			AssetType:      a,
			Price:          resp.Result[x].Price.Float64(),
			ExecutedAmount: resp.Result[x].Amount.Float64() - resp.Result[x].Left.Float64(),
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
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Amount.Float64(),
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
			Amount:   resp.Result[x].Available.Float64(),
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
			Amount:   resp.Result[x].Available.Float64(),
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
			Amount:   resp.Result[x].Available.Float64(),
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

// generateSubscriptionsSpot returns configured subscriptions
func (g *Gateio) generateSubscriptionsSpot() (subscription.List, error) {
	return g.Features.Subscriptions.ExpandTemplates(g)
}

// GetSubscriptionTemplate returns a subscription channel template
func (g *Gateio) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").
		Funcs(sprig.FuncMap()).
		Funcs(template.FuncMap{
			"channelName":         channelName,
			"singleSymbolChannel": singleSymbolChannel,
			"interval":            g.GetIntervalString,
		}).
		Parse(subTplText)
}

// manageSubs sends a websocket message to subscribe or unsubscribe from a list of channel
func (g *Gateio) manageSubs(ctx context.Context, event string, conn stream.Connection, subs subscription.List) error {
	var errs error
	subs, errs = subs.ExpandTemplates(g)
	if errs != nil {
		return errs
	}

	for _, s := range subs {
		if err := func() error {
			msg, err := g.manageSubReq(ctx, event, conn, s)
			if err != nil {
				return err
			}
			result, err := conn.SendMessageReturnResponse(ctx, request.Unset, msg.ID, msg)
			if err != nil {
				return err
			}
			var resp WsEventResponse
			if err := json.Unmarshal(result, &resp); err != nil {
				return err
			}
			if resp.Error != nil && resp.Error.Code != 0 {
				return fmt.Errorf("(%d) %s", resp.Error.Code, resp.Error.Message)
			}
			if event == "unsubscribe" {
				return g.Websocket.RemoveSubscriptions(conn, s)
			}
			return g.Websocket.AddSuccessfulSubscriptions(conn, s)
		}(); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s %s: %w", s.Channel, s.Asset, s.Pairs, err))
		}
	}
	return errs
}

// manageSubReq constructs the subscription management message for a subscription
func (g *Gateio) manageSubReq(ctx context.Context, event string, conn stream.Connection, s *subscription.Subscription) (*WsInput, error) {
	req := &WsInput{
		ID:      conn.GenerateMessageID(false),
		Event:   event,
		Channel: channelName(s),
		Time:    time.Now().Unix(),
		Payload: strings.Split(s.QualifiedChannel, ","),
	}
	if s.Authenticated {
		creds, err := g.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}
		sig, err := g.generateWsSignature(creds.Secret, event, req.Channel, req.Time)
		if err != nil {
			return nil, err
		}
		req.Auth = &WsAuthInput{
			Method: "api_key",
			Key:    creds.Key,
			Sign:   sig,
		}
	}
	return req, nil
}

// Subscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Subscribe(ctx context.Context, conn stream.Connection, subs subscription.List) error {
	return g.manageSubs(ctx, subscribeEvent, conn, subs)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(ctx context.Context, conn stream.Connection, subs subscription.List) error {
	return g.manageSubs(ctx, unsubscribeEvent, conn, subs)
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

// GenerateWebsocketMessageID generates a message ID for the individual connection
func (g *Gateio) GenerateWebsocketMessageID(bool) int64 {
	return g.Counter.IncrementAndGet()
}

// channelName converts global channel names to gateio specific channel names
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	return s.Channel
}

// singleSymbolChannel returns if the channel should be fanned out into single symbol requests
func singleSymbolChannel(name string) bool {
	switch name {
	case spotCandlesticksChannel, spotOrderbookUpdateChannel, spotOrderbookChannel:
		return true
	}
	return false
}

const subTplText = `
{{- with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if singleSymbolChannel $name }}
			{{- range $i, $p := $pairs -}}
				{{- if eq $name "spot.candlesticks" }}{{ interval $.S.Interval -}} , {{- end }}
				{{- $p }}
				{{- if eq "spot.order_book" $name -}} , {{- $.S.Levels }}{{ end }}
				{{- if hasPrefix "spot.order_book" $name -}} , {{- interval $.S.Interval }}{{ end }}
				{{- $.PairSeparator }}
			{{- end }}
			{{- $.AssetSeparator }}
		{{- else }}
			{{- $pairs.Join }}
		{{- end }}
	{{- end }}
{{- end }}
`

// GeneratePayload returns the payload for a websocket message
type GeneratePayload func(ctx context.Context, conn stream.Connection, event string, channelsToSubscribe subscription.List) ([]WsInput, error)

// handleSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleSubscription(ctx context.Context, conn stream.Connection, event string, channelsToSubscribe subscription.List, generatePayload GeneratePayload) error {
	payloads, err := generatePayload(ctx, conn, event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs error
	for k := range payloads {
		result, err := conn.SendMessageReturnResponse(ctx, request.Unset, payloads[k].ID, payloads[k])
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
			if event == subscribeEvent {
				errs = common.AppendError(errs, g.Websocket.AddSuccessfulSubscriptions(conn, channelsToSubscribe[k]))
			} else {
				errs = common.AppendError(errs, g.Websocket.RemoveSubscriptions(conn, channelsToSubscribe[k]))
			}
		}
	}
	return errs
}
