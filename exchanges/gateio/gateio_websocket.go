package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	gateioWebsocketEndpoint = "wss://api.gateio.ws/ws/v4/"

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
	{Enabled: false, Channel: spotOrderbookTickerChannel, Asset: asset.Spot, Interval: kline.TenMilliseconds, Levels: 1},
	{Enabled: false, Channel: spotOrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds, Levels: 100},
	{Enabled: true, Channel: spotBalancesChannel, Asset: asset.Spot, Authenticated: true},
	{Enabled: true, Channel: crossMarginBalanceChannel, Asset: asset.CrossMargin, Authenticated: true},
	{Enabled: true, Channel: marginBalancesChannel, Asset: asset.Margin, Authenticated: true},
	{Enabled: false, Channel: subscription.AllTradesChannel, Asset: asset.Spot},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    spotTickerChannel,
	subscription.OrderbookChannel: spotOrderbookUpdateChannel,
	subscription.CandlesChannel:   spotCandlesticksChannel,
	subscription.AllTradesChannel: spotTradesChannel,
}

var standardMarginAssetTypes = []asset.Item{asset.Spot, asset.Margin, asset.CrossMargin}

// WsConnectSpot initiates a websocket connection
func (g *Gateio) WsConnectSpot(ctx context.Context, conn websocket.Connection) error {
	err := g.CurrencyPairs.IsAssetEnabled(asset.Spot)
	if err != nil {
		return err
	}
	err = conn.Dial(ctx, &gws.Dialer{}, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{Channel: spotPingChannel})
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, websocket.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 15,
		Message:     pingMessage,
		MessageType: gws.TextMessage,
	})
	return nil
}

// websocketLogin authenticates the websocket connection
func (g *Gateio) websocketLogin(ctx context.Context, conn websocket.Connection, channel string) error {
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

	resp, err := conn.SendMessageReturnResponse(ctx, websocketRateLimitNotNeededEPL, payload.RequestID, req)
	if err != nil {
		return err
	}

	var inbound WebsocketAPIResponse
	if err := json.Unmarshal(resp, &inbound); err != nil {
		return err
	}

	if inbound.Header.Status == http.StatusOK {
		return nil
	}

	var wsErr WebsocketErrors
	if err := json.Unmarshal(inbound.Data, &wsErr.Errors); err != nil {
		return err
	}

	return fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
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
func (g *Gateio) WsHandleSpotData(ctx context.Context, respRaw []byte) error {
	push, err := parseWSHeader(respRaw)
	if err != nil {
		return err
	}

	if push.RequestID != "" {
		return g.Websocket.Match.RequireMatchWithData(push.RequestID, respRaw)
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		return g.Websocket.Match.RequireMatchWithData(push.ID, respRaw)
	}

	switch push.Channel { // TODO: Convert function params below to only use push.Result
	case spotTickerChannel:
		return g.processTicker(push.Result, push.Time)
	case spotTradesChannel:
		return g.processTrades(push.Result)
	case spotCandlesticksChannel:
		return g.processCandlestick(push.Result)
	case spotOrderbookTickerChannel:
		return g.processOrderbookTicker(push.Result, push.Time)
	case spotOrderbookUpdateChannel:
		return g.processOrderbookUpdate(ctx, push.Result, push.Time)
	case spotOrderbookChannel:
		return g.processOrderbookSnapshot(push.Result, push.Time)
	case spotOrdersChannel:
		return g.processSpotOrders(respRaw)
	case spotUserTradesChannel:
		return g.processUserPersonalTrades(respRaw)
	case spotBalancesChannel:
		return g.processSpotBalances(ctx, respRaw)
	case marginBalancesChannel:
		return g.processMarginBalances(ctx, respRaw)
	case spotFundingBalanceChannel:
		return g.processFundingBalances(respRaw)
	case crossMarginBalanceChannel:
		return g.processCrossMarginBalance(ctx, respRaw)
	case crossMarginLoanChannel:
		return g.processCrossMarginLoans(respRaw)
	case spotPongChannel:
	default:
		g.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: g.Name + websocket.UnhandledMessage + string(respRaw),
		}
		return errors.New(websocket.UnhandledMessage)
	}
	return nil
}

func parseWSHeader(msg []byte) (r *WSResponse, errs error) {
	r = &WSResponse{}
	paths := [][]string{{"time_ms"}, {"time"}, {"channel"}, {"event"}, {"request_id"}, {"id"}, {"result"}}
	jsonparser.EachKey(msg, func(idx int, v []byte, _ jsonparser.ValueType, _ error) {
		switch idx {
		case 0: // time_ms
			if ts, err := strconv.ParseInt(string(v), 10, 64); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w parsing `time_ms`", err))
			} else {
				r.Time = time.UnixMilli(ts)
			}
		case 1: // time
			if r.Time.IsZero() {
				if ts, err := strconv.ParseInt(string(v), 10, 64); err != nil {
					errs = common.AppendError(errs, fmt.Errorf("%w parsing `time`", err))
				} else {
					r.Time = time.Unix(ts, 0)
				}
			}
		case 2:
			r.Channel = string(v)
		case 3:
			r.Event = string(v)
		case 4:
			r.RequestID = string(v)
		case 5:
			if id, err := strconv.ParseInt(string(v), 10, 64); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w parsing `id`", err))
			} else {
				r.ID = id
			}
		case 6:
			r.Result = json.RawMessage(v)
		}
	}, paths...)

	return r, errs
}

func (g *Gateio) processTicker(incoming []byte, pushTime time.Time) error {
	var data WsTicker
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	out := make([]ticker.Price, 0, len(standardMarginAssetTypes))
	for _, a := range standardMarginAssetTypes {
		if enabled, _ := g.CurrencyPairs.IsPairEnabled(data.CurrencyPair, a); enabled {
			out = append(out, ticker.Price{
				ExchangeName: g.Name,
				Volume:       data.BaseVolume.Float64(),
				QuoteVolume:  data.QuoteVolume.Float64(),
				High:         data.High24H.Float64(),
				Low:          data.Low24H.Float64(),
				Last:         data.Last.Float64(),
				Bid:          data.HighestBid.Float64(),
				Ask:          data.LowestAsk.Float64(),
				AssetType:    a,
				Pair:         data.CurrencyPair,
				LastUpdated:  pushTime,
			})
		}
	}
	g.Websocket.DataHandler <- out
	return nil
}

func (g *Gateio) processTrades(incoming []byte) error {
	saveTradeData := g.IsSaveTradeDataEnabled()
	if !saveTradeData && !g.IsTradeFeedEnabled() {
		return nil
	}

	var data WsTrade
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}

	side, err := order.StringToOrderSide(data.Side)
	if err != nil {
		return err
	}

	for _, a := range standardMarginAssetTypes {
		if enabled, _ := g.CurrencyPairs.IsPairEnabled(data.CurrencyPair, a); enabled {
			if err := g.Websocket.Trade.Update(saveTradeData, trade.Data{
				Timestamp:    data.CreateTime.Time(),
				CurrencyPair: data.CurrencyPair,
				AssetType:    a,
				Exchange:     g.Name,
				Price:        data.Price.Float64(),
				Amount:       data.Amount.Float64(),
				Side:         side,
				TID:          strconv.FormatInt(data.ID, 10),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Gateio) processCandlestick(incoming []byte) error {
	var data WsCandlesticks
	if err := json.Unmarshal(incoming, &data); err != nil {
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

	out := make([]websocket.KlineData, 0, len(standardMarginAssetTypes))
	for _, a := range standardMarginAssetTypes {
		if enabled, _ := g.CurrencyPairs.IsPairEnabled(currencyPair, a); enabled {
			out = append(out, websocket.KlineData{
				Pair:       currencyPair,
				AssetType:  a,
				Exchange:   g.Name,
				StartTime:  data.Timestamp.Time(),
				Interval:   icp[0],
				OpenPrice:  data.OpenPrice.Float64(),
				ClosePrice: data.ClosePrice.Float64(),
				HighPrice:  data.HighestPrice.Float64(),
				LowPrice:   data.LowestPrice.Float64(),
				Volume:     data.TotalVolume.Float64(),
			})
		}
	}
	g.Websocket.DataHandler <- out
	return nil
}

func (g *Gateio) processOrderbookTicker(incoming []byte, lastPushed time.Time) error {
	var data WsOrderbookTickerData
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	return g.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:    g.Name,
		Pair:        data.Pair,
		Asset:       asset.Spot,
		LastUpdated: data.UpdateTime.Time(),
		LastPushed:  lastPushed,
		Bids:        []orderbook.Level{{Price: data.BestBidPrice.Float64(), Amount: data.BestBidAmount.Float64()}},
		Asks:        []orderbook.Level{{Price: data.BestAskPrice.Float64(), Amount: data.BestAskAmount.Float64()}},
	})
}

func (g *Gateio) processOrderbookUpdate(ctx context.Context, incoming []byte, lastPushed time.Time) error {
	var data WsOrderbookUpdate
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	asks := make([]orderbook.Level, len(data.Asks))
	for x := range data.Asks {
		asks[x].Price = data.Asks[x][0].Float64()
		asks[x].Amount = data.Asks[x][1].Float64()
	}
	bids := make([]orderbook.Level, len(data.Bids))
	for x := range data.Bids {
		bids[x].Price = data.Bids[x][0].Float64()
		bids[x].Amount = data.Bids[x][1].Float64()
	}
	return g.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, g, data.FirstUpdateID, &orderbook.Update{
		UpdateID:   data.LastUpdateID,
		UpdateTime: data.UpdateTime.Time(),
		LastPushed: lastPushed,
		Pair:       data.Pair,
		Asset:      asset.Spot,
		Asks:       asks,
		Bids:       bids,
		AllowEmpty: true,
	})
}

func (g *Gateio) processOrderbookSnapshot(incoming []byte, lastPushed time.Time) error {
	var data WsOrderbookSnapshot
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}

	asks := make([]orderbook.Level, len(data.Asks))
	for x := range data.Asks {
		asks[x].Price = data.Asks[x][0].Float64()
		asks[x].Amount = data.Asks[x][1].Float64()
	}
	bids := make([]orderbook.Level, len(data.Bids))
	for x := range data.Bids {
		bids[x].Price = data.Bids[x][0].Float64()
		bids[x].Amount = data.Bids[x][1].Float64()
	}

	for _, a := range standardMarginAssetTypes {
		if enabled, _ := g.CurrencyPairs.IsPairEnabled(data.CurrencyPair, a); enabled {
			if err := g.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Exchange:    g.Name,
				Pair:        data.CurrencyPair,
				Asset:       a,
				LastUpdated: data.UpdateTime.Time(),
				LastPushed:  lastPushed,
				Bids:        bids,
				Asks:        asks,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Gateio) processSpotOrders(data []byte) error {
	resp := struct {
		Time    types.Time    `json:"time"`
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
			Date:           resp.Result[x].CreateTime.Time(),
			LastUpdated:    resp.Result[x].UpdateTime.Time(),
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
		Time    types.Time            `json:"time"`
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
			Timestamp:    resp.Result[x].CreateTime.Time(),
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

func (g *Gateio) processSpotBalances(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time      `json:"time"`
		Channel string          `json:"channel"`
		Event   string          `json:"event"`
		Result  []WsSpotBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	changes := make([]account.Change, len(resp.Result))
	for i := range resp.Result {
		changes[i] = account.Change{
			Account:   resp.Result[i].User,
			AssetType: asset.Spot,
			Balance: &account.Balance{
				Currency:  currency.NewCode(resp.Result[i].Currency),
				Total:     resp.Result[i].Total.Float64(),
				Free:      resp.Result[i].Available.Float64(),
				Hold:      resp.Result[i].Total.Float64() - resp.Result[i].Available.Float64(),
				UpdatedAt: resp.Result[i].Timestamp.Time(),
			},
		}
	}
	g.Websocket.DataHandler <- changes
	return account.ProcessChange(g.Name, changes, creds)
}

func (g *Gateio) processMarginBalances(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time        `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsMarginBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	changes := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		changes[x] = account.Change{
			AssetType: asset.Margin,
			Balance: &account.Balance{
				Currency:  currency.NewCode(resp.Result[x].Currency),
				Total:     resp.Result[x].Available.Float64() + resp.Result[x].Freeze.Float64(),
				Free:      resp.Result[x].Available.Float64(),
				Hold:      resp.Result[x].Freeze.Float64(),
				UpdatedAt: resp.Result[x].Timestamp.Time(),
			},
		}
	}
	g.Websocket.DataHandler <- changes
	return account.ProcessChange(g.Name, changes, creds)
}

func (g *Gateio) processFundingBalances(data []byte) error {
	resp := struct {
		Time    types.Time         `json:"time"`
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

func (g *Gateio) processCrossMarginBalance(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time             `json:"time"`
		Channel string                 `json:"channel"`
		Event   string                 `json:"event"`
		Result  []WsCrossMarginBalance `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	changes := make([]account.Change, len(resp.Result))
	for x := range resp.Result {
		changes[x] = account.Change{
			Account:   resp.Result[x].User,
			AssetType: asset.Margin,
			Balance: &account.Balance{
				Currency:  currency.NewCode(resp.Result[x].Currency),
				Total:     resp.Result[x].Total.Float64(),
				Free:      resp.Result[x].Available.Float64(),
				UpdatedAt: resp.Result[x].Timestamp.Time(),
			},
		}
	}
	g.Websocket.DataHandler <- changes
	return account.ProcessChange(g.Name, changes, creds)
}

func (g *Gateio) processCrossMarginLoans(data []byte) error {
	resp := struct {
		Time    types.Time        `json:"time"`
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
			"orderbookInterval":   orderbookChannelInterval,
			"candlesInterval":     candlesChannelInterval,
			"levels":              channelLevels,
		}).Parse(subTplText)
}

// manageSubs sends a websocket message to subscribe or unsubscribe from a list of channel
func (g *Gateio) manageSubs(ctx context.Context, event string, conn websocket.Connection, subs subscription.List) error {
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
			result, err := conn.SendMessageReturnResponse(ctx, websocketRateLimitNotNeededEPL, msg.ID, msg)
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
func (g *Gateio) manageSubReq(ctx context.Context, event string, conn websocket.Connection, s *subscription.Subscription) (*WsInput, error) {
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
func (g *Gateio) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return g.manageSubs(ctx, subscribeEvent, conn, subs)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return g.manageSubs(ctx, unsubscribeEvent, conn, subs)
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

// ValidateSubscriptions implements the subscription.ListValidator interface.
// It ensures that, for each orderbook pair asset, only one type of subscription (e.g., best bid/ask, orderbook update, or orderbook snapshot)
// is active at a time. Multiple concurrent subscriptions for the same asset are disallowed to prevent orderbook data corruption.
func (g *Gateio) ValidateSubscriptions(l subscription.List) error {
	orderbookGuard := map[key.PairAsset]string{}
	for _, s := range l {
		n := channelName(s)
		if !isSingleOrderbookChannel(n) {
			continue
		}
		for _, p := range s.Pairs {
			k := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: s.Asset}
			existingChanName, ok := orderbookGuard[k]
			if !ok {
				orderbookGuard[k] = n
				continue
			}
			if existingChanName != n {
				return fmt.Errorf("%w for %q %q between %q and %q, please enable only one type", subscription.ErrExclusiveSubscription, k.Pair(), k.Asset, existingChanName, n)
			}
		}
	}
	return nil
}

// isSingleOrderbookChannel checks if the specified channel represents a single orderbook subscription.
// It returns true for channels like orderbook updates, snapshots, or tickers, as multiple subscriptions
// for the same pair asset could corrupt the stored orderbook data.
func isSingleOrderbookChannel(name string) bool {
	switch name {
	case spotOrderbookUpdateChannel,
		spotOrderbookChannel,
		spotOrderbookTickerChannel,
		futuresOrderbookChannel,
		futuresOrderbookTickerChannel,
		futuresOrderbookUpdateChannel,
		optionsOrderbookChannel,
		optionsOrderbookTickerChannel,
		optionsOrderbookUpdateChannel:
		return true
	}
	return false
}

var channelIntervalsMap = map[asset.Item]map[string][]kline.Interval{
	asset.Spot: {
		spotOrderbookTickerChannel: {},
		spotOrderbookChannel:       {kline.HundredMilliseconds, kline.ThousandMilliseconds},
		spotOrderbookUpdateChannel: {kline.TwentyMilliseconds, kline.HundredMilliseconds},
	},
	asset.Futures: {
		futuresOrderbookTickerChannel: {},
		futuresOrderbookChannel:       {0},
		futuresOrderbookUpdateChannel: {kline.TwentyMilliseconds, kline.HundredMilliseconds},
	},
	asset.DeliveryFutures: {
		futuresOrderbookTickerChannel: {},
		futuresOrderbookChannel:       {0},
		futuresOrderbookUpdateChannel: {kline.HundredMilliseconds, kline.ThousandMilliseconds},
	},
	asset.Options: {
		optionsOrderbookTickerChannel: {},
		optionsOrderbookChannel:       {0},
		optionsOrderbookUpdateChannel: {kline.HundredMilliseconds, kline.ThousandMilliseconds},
	},
}

func candlesChannelInterval(s *subscription.Subscription) (string, error) {
	if s.Channel == subscription.CandlesChannel {
		return getIntervalString(s.Interval)
	}
	return "", nil
}

func orderbookChannelInterval(s *subscription.Subscription, a asset.Item) (string, error) {
	cName := channelName(s)

	assetChannels, ok := channelIntervalsMap[a]
	if !ok {
		return "", nil
	}

	switch intervals, ok := assetChannels[cName]; {
	case !ok:
		return "", nil
	case len(intervals) == 0:
		if s.Interval != 0 {
			return "", fmt.Errorf("%w for %s: %q; interval not supported for channel", subscription.ErrInvalidInterval, cName, s.Interval)
		}
		return "", nil
	case !slices.Contains(intervals, s.Interval):
		return "", fmt.Errorf("%w for %s: %q; supported: %q", subscription.ErrInvalidInterval, cName, s.Interval, intervals)
	case cName == futuresOrderbookUpdateChannel && s.Interval == kline.TwentyMilliseconds && s.Levels != 20:
		return "", fmt.Errorf("%w for %q: 20ms only valid with Levels 20", subscription.ErrInvalidInterval, cName)
	case s.Interval == 0:
		return "0", nil // Do not move this into getIntervalString, it's only valid for ws subs
	}

	return getIntervalString(s.Interval)
}

var channelLevelsMap = map[asset.Item]map[string][]int{
	asset.Spot: {
		spotOrderbookTickerChannel: {},
		spotOrderbookUpdateChannel: {},
		spotOrderbookChannel:       {1, 5, 10, 20, 50, 100},
	},
	asset.Futures: {
		futuresOrderbookChannel:       {1, 5, 10, 20, 50, 100},
		futuresOrderbookTickerChannel: {},
		futuresOrderbookUpdateChannel: {20, 50, 100},
	},
	asset.DeliveryFutures: {
		futuresOrderbookChannel:       {1, 5, 10, 20, 50, 100},
		futuresOrderbookTickerChannel: {},
		futuresOrderbookUpdateChannel: {5, 10, 20, 50, 100},
	},
	asset.Options: {
		optionsOrderbookTickerChannel: {},
		optionsOrderbookUpdateChannel: {5, 10, 20, 50},
		optionsOrderbookChannel:       {5, 10, 20, 50},
	},
}

func channelLevels(s *subscription.Subscription, a asset.Item) (string, error) {
	cName := channelName(s)
	assetChannels, ok := channelLevelsMap[a]
	if !ok {
		return "", nil
	}

	switch levels, ok := assetChannels[cName]; {
	case !ok:
		return "", nil
	case len(levels) == 0:
		if s.Levels != 0 {
			return "", fmt.Errorf("%w for %s: `%d`; levels not supported for channel", subscription.ErrInvalidLevel, cName, s.Levels)
		}
		return "", nil
	case !slices.Contains(levels, s.Levels):
		return "", fmt.Errorf("%w for %s: %d; supported: %v", subscription.ErrInvalidLevel, cName, s.Levels, levels)
	}

	return strconv.Itoa(s.Levels), nil
}

const subTplText = `
{{- with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if singleSymbolChannel $name }}
			{{- range $i, $p := $pairs -}}
				{{- with $i := candlesInterval $.S }}{{ $i -}} , {{- end }}
				{{- $p }}
				{{- with $l := levels $.S $asset -}} , {{- $l }}{{ end }}
				{{- with $i := orderbookInterval $.S $asset -}} , {{- $i }}{{- end }}
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
type GeneratePayload func(ctx context.Context, conn websocket.Connection, event string, channelsToSubscribe subscription.List) ([]WsInput, error)

// handleSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleSubscription(ctx context.Context, conn websocket.Connection, event string, channelsToSubscribe subscription.List, generatePayload GeneratePayload) error {
	payloads, err := generatePayload(ctx, conn, event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs error
	for k := range payloads {
		result, err := conn.SendMessageReturnResponse(ctx, websocketRateLimitNotNeededEPL, payloads[k].ID, payloads[k])
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

type resultHolder struct {
	Result any `json:"result"`
}

// SendWebsocketRequest sends a websocket request to the exchange
func (g *Gateio) SendWebsocketRequest(ctx context.Context, epl request.EndpointLimit, channel string, connSignature, params, result any, expectedResponses int) error {
	paramPayload, err := json.Marshal(params)
	if err != nil {
		return err
	}

	conn, err := g.Websocket.GetConnection(connSignature)
	if err != nil {
		return err
	}

	tn := time.Now().Unix()
	req := &WebsocketRequest{
		Time:    tn,
		Channel: channel,
		Event:   "api",
		Payload: WebsocketPayload{
			RequestID:    strconv.FormatInt(conn.GenerateMessageID(false), 10),
			RequestParam: paramPayload,
			Timestamp:    strconv.FormatInt(tn, 10),
		},
	}

	responses, err := conn.SendMessageReturnResponsesWithInspector(ctx, epl, req.Payload.RequestID, req, expectedResponses, wsRespAckInspector{})
	if err != nil {
		return err
	}

	if len(responses) == 0 {
		return common.ErrNoResponse
	}

	// responses may include an ack resp, which we skip
	endResponse := responses[len(responses)-1]

	var inbound WebsocketAPIResponse
	if err := json.Unmarshal(endResponse, &inbound); err != nil {
		return err
	}

	if inbound.Header.Status != http.StatusOK {
		var wsErr WebsocketErrors
		if err := json.Unmarshal(inbound.Data, &wsErr); err != nil {
			return err
		}
		return fmt.Errorf("%s: %s", wsErr.Errors.Label, wsErr.Errors.Message)
	}

	return json.Unmarshal(inbound.Data, &resultHolder{Result: result})
}

type wsRespAckInspector struct{}

// IsFinal checks the payload for an ack, it returns true if the payload does not contain an ack.
// This will force the cancellation of further waiting for responses.
func (wsRespAckInspector) IsFinal(data []byte) bool {
	return !strings.Contains(string(data), "ack")
}
