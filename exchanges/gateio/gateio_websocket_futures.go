package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	btcFuturesWebsocketURL  = "wss://fx-ws.gateio.ws/v4/ws/btc"
	usdtFuturesWebsocketURL = "wss://fx-ws.gateio.ws/v4/ws/usdt"

	futuresPingChannel            = "futures.ping"
	futuresTickersChannel         = "futures.tickers"
	futuresTradesChannel          = "futures.trades"
	futuresOrderbookChannel       = "futures.order_book"
	futuresOrderbookTickerChannel = "futures.book_ticker"
	futuresOrderbookUpdateChannel = "futures.order_book_update"
	futuresOrderbookV2            = "futures.obu"
	futuresCandlesticksChannel    = "futures.candlesticks"
	futuresOrdersChannel          = "futures.orders"

	//  authenticated channels
	futuresUserTradesChannel        = "futures.usertrades"
	futuresLiquidatesChannel        = "futures.liquidates"
	futuresAutoDeleveragesChannel   = "futures.auto_deleverages"
	futuresAutoPositionCloseChannel = "futures.position_closes"
	futuresBalancesChannel          = "futures.balances"
	futuresReduceRiskLimitsChannel  = "futures.reduce_risk_limits"
	futuresPositionsChannel         = "futures.positions"
	futuresAutoOrdersChannel        = "futures.autoorders"

	futuresOrderbookUpdateLimit uint64 = 20

	invalidUserID = "invalidUserID"
)

var defaultFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookV2,
	futuresCandlesticksChannel,
}

var defaultCoinMarginedFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookUpdateChannel,
	futuresCandlesticksChannel,
}

var errNoChannelsSupplied = errors.New("no channels supplied")

const (
	allFuturesContracts          = "!all"
	contractPayloadOverrideParam = "contractPayloadOverride"
	omitContractParam            = "omitContract"
	requiresUserPlaceholderParam = "requiresUserPlaceholder"
)

// WsFuturesConnect initiates a websocket connection for futures account
func (e *Exchange) WsFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	a := asset.USDTMarginedFutures
	if conn.GetURL() == btcFuturesWebsocketURL {
		a = asset.CoinMarginedFutures
	}
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return err
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{"X-Gate-Size-Decimal": []string{"1"}}, nil); err != nil {
		return err
	}
	pingHandler, err := getWSPingHandler(futuresPingChannel)
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, pingHandler)
	return nil
}

// GenerateFuturesDefaultSubscriptions returns default subscriptions information.
// TODO: Update to use the new subscription template system
func (e *Exchange) GenerateFuturesDefaultSubscriptions(ctx context.Context, a asset.Item) (subscription.List, error) {
	if a != asset.USDTMarginedFutures && a != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}

	channelsToSubscribe := defaultFuturesSubscriptions
	if a == asset.CoinMarginedFutures {
		channelsToSubscribe = defaultCoinMarginedFuturesSubscriptions
	}

	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil // no enabled pairs, subscriptions require an associated pair.
		}
		return nil, err
	}

	if len(pairs) == 0 {
		return nil, nil
	}

	var userID string
	var subscriptionErr error
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err := e.GetCredentials(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		} else {
			settlementCurrency, err := getSettlementCurrency(currency.EMPTYPAIR, a)
			if err != nil {
				return nil, err
			}
			account, err := e.QueryFuturesAccount(ctx, settlementCurrency)
			switch {
			case err == nil && account != nil && account.User != 0:
				userID = strconv.FormatInt(account.User, 10)
				e.setFuturesUserID(creds.Key, settlementCurrency, userID)
			default:
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s: error querying futures account: %v", e.Name, err)
				}
				// Reuse the last known ID so a transient failure does not drop authenticated
				// channels, which a later refresh would otherwise unsubscribe.
				if userID = e.getFuturesUserID(creds.Key, settlementCurrency); userID == "" {
					userID = invalidUserID
					if err != nil {
						subscriptionErr = fmt.Errorf("%w: unable to query futures account user ID: %w", websocket.ErrSubscriptionPartial, err)
					} else {
						subscriptionErr = fmt.Errorf("%w: futures account user ID missing", websocket.ErrSubscriptionPartial)
					}
				}
			}
		}
	}

	var subscriptions subscription.List
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]any)
			switch channelsToSubscribe[i] {
			case futuresOrderbookChannel:
				params["limit"] = 100
				params["interval"] = "0"
			case futuresCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case futuresOrderbookUpdateChannel:
				// This is the fastest frequency available for futures orderbook updates 20 levels every 20ms
				params["frequency"] = kline.TwentyMilliseconds
				params["level"] = strconv.FormatUint(futuresOrderbookUpdateLimit, 10)
			case futuresOrderbookV2:
				// Fastest frequency available. 50 levels which defaults to 20ms frequency
				params["level"] = uint64(50)
			}
			fPair, err := e.FormatExchangeCurrency(pairs[j], a)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fPair.Upper()},
				Params:  params,
				Asset:   a,
			})
		}
	}
	if e.Websocket.CanUseAuthenticatedEndpoints() {
	channels:
		for _, channel := range []string{
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresLiquidatesChannel,
			futuresAutoDeleveragesChannel,
			futuresAutoPositionCloseChannel,
			futuresBalancesChannel,
			futuresReduceRiskLimitsChannel,
			futuresPositionsChannel,
			futuresAutoOrdersChannel,
		} {
			params := map[string]any{contractPayloadOverrideParam: allFuturesContracts}
			switch channel {
			case futuresPositionsChannel:
				// Gate deprecated the user ID value but still requires its positional placeholder.
				params[requiresUserPlaceholderParam] = true
			case futuresBalancesChannel:
				if userID == invalidUserID {
					log.Errorf(log.WebsocketMgr, "%s: skipping authenticated channel subscription: invalid user ID for %s channel", e.Name, channel)
					continue channels
				}
				delete(params, contractPayloadOverrideParam)
				params[omitContractParam] = true
				params["user"] = userID
			case futuresAutoOrdersChannel:
			default:
				if userID == invalidUserID {
					log.Errorf(log.WebsocketMgr, "%s: skipping authenticated channel subscription: invalid user ID for %s channel", e.Name, channel)
					continue channels
				}
				params["user"] = userID
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channel,
				Pairs:   pairs[0:1],
				Params:  params,
				Asset:   a,
			})
		}
	}
	return subscriptions, subscriptionErr
}

// futuresUserIDKey scopes a cached account ID to one credential and settlement account;
// Gate treats each settlement currency as a separate futures account.
func futuresUserIDKey(credentialKey string, settlementCurrency currency.Code) string {
	return credentialKey + "/" + settlementCurrency.Upper().String()
}

func (e *Exchange) setFuturesUserID(credentialKey string, settlementCurrency currency.Code, userID string) {
	e.futuresUserIDMu.Lock()
	defer e.futuresUserIDMu.Unlock()
	if e.futuresUserIDs == nil {
		e.futuresUserIDs = make(map[string]string)
	}
	e.futuresUserIDs[futuresUserIDKey(credentialKey, settlementCurrency)] = userID
}

func (e *Exchange) getFuturesUserID(credentialKey string, settlementCurrency currency.Code) string {
	e.futuresUserIDMu.RLock()
	defer e.futuresUserIDMu.RUnlock()
	return e.futuresUserIDs[futuresUserIDKey(credentialKey, settlementCurrency)]
}

// FuturesSubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) FuturesSubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, subscribeEvent, channelsToUnsubscribe, e.generateFuturesPayload)
}

// FuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) FuturesUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, unsubscribeEvent, channelsToUnsubscribe, e.generateFuturesPayload)
}

// WsHandleFuturesData handles futures websocket data
func (e *Exchange) WsHandleFuturesData(ctx context.Context, conn websocket.Connection, respRaw []byte, a asset.Item) error {
	push, err := parseWSHeader(respRaw)
	if err != nil {
		return err
	}

	if push.RequestID != "" {
		return conn.RequireMatchWithData(push.RequestID, respRaw)
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		return conn.RequireMatchWithData(push.ID, respRaw)
	}

	switch push.Channel {
	case futuresTickersChannel:
		return e.processFuturesTickers(ctx, respRaw, a)
	case futuresTradesChannel:
		return e.processFuturesTrades(respRaw, a)
	case futuresOrderbookChannel:
		return e.processFuturesOrderbookSnapshot(push.Event, push.Result, a, push.Time)
	case futuresOrderbookTickerChannel:
		return e.processFuturesOrderbookTicker(ctx, push.Result)
	case futuresOrderbookUpdateChannel:
		return e.processFuturesOrderbookUpdate(ctx, push.Result, a, push.Time)
	case futuresOrderbookV2:
		return e.processOrderbookUpdateWithSnapshot(ctx, conn, push.Result, push.Time, a)
	case futuresCandlesticksChannel:
		return e.processFuturesCandlesticks(ctx, respRaw, a)
	case futuresOrdersChannel:
		processed, err := e.processFuturesOrdersPushData(respRaw, a)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, processed)
	case futuresUserTradesChannel:
		return e.processFuturesUserTrades(respRaw, a)
	case futuresLiquidatesChannel:
		return e.processFuturesLiquidatesNotification(ctx, respRaw)
	case futuresAutoDeleveragesChannel:
		return e.processFuturesAutoDeleveragesNotification(ctx, respRaw)
	case futuresAutoPositionCloseChannel:
		return e.processPositionCloseData(ctx, respRaw, a)
	case futuresBalancesChannel:
		return e.processBalancePushData(ctx, push.Result, a)
	case futuresReduceRiskLimitsChannel:
		return e.processFuturesReduceRiskLimitNotification(ctx, respRaw)
	case futuresPositionsChannel:
		return e.processFuturesPositionsNotification(ctx, respRaw, a)
	case futuresAutoOrdersChannel:
		return e.processFuturesAutoOrderPushData(ctx, respRaw)
	case "futures.pong":
		return nil
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

func (e *Exchange) generateFuturesPayload(ctx context.Context, event string, channelsToSubscribe subscription.List) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errNoChannelsSupplied
	}

	var creds *accounts.Credentials
	var err error
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = e.GetCredentials(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	outbound := make([]WsInput, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return nil, subscription.ErrNotSinglePair
		}
		var auth *WsAuthInput
		timestamp := time.Now()
		params := []string{channelsToSubscribe[i].Pairs[0].String()}
		if omitContract, _ := channelsToSubscribe[i].Params[omitContractParam].(bool); omitContract {
			params = nil
		} else if contractOverride, ok := channelsToSubscribe[i].Params[contractPayloadOverrideParam].(string); ok {
			params[0] = contractOverride
		}
		if e.Websocket.CanUseAuthenticatedEndpoints() {
			switch channelsToSubscribe[i].Channel {
			case futuresOrdersChannel, futuresUserTradesChannel,
				futuresLiquidatesChannel, futuresAutoDeleveragesChannel,
				futuresAutoPositionCloseChannel, futuresBalancesChannel,
				futuresReduceRiskLimitsChannel, futuresPositionsChannel,
				futuresAutoOrdersChannel:
				userID, hasUserID := channelsToSubscribe[i].Params["user"].(string)
				if channelsToSubscribe[i].Channel == futuresAutoPositionCloseChannel && (!hasUserID || userID == "") {
					return nil, fmt.Errorf("%w: user ID for %s", common.ErrParameterRequired, futuresAutoPositionCloseChannel)
				}
				if requiresUserPlaceholder, _ := channelsToSubscribe[i].Params[requiresUserPlaceholderParam].(bool); requiresUserPlaceholder {
					params = append([]string{""}, params...)
				} else if hasUserID {
					params = append([]string{userID}, params...)
				}
				var sigTemp string
				sigTemp, err = e.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp.Unix())
				if err != nil {
					return nil, err
				}
				auth = &WsAuthInput{
					Method: "api_key",
					Key:    creds.Key,
					Sign:   sigTemp,
				}
			}
		}
		frequency, okay := channelsToSubscribe[i].Params["frequency"].(kline.Interval)
		if okay {
			var frequencyString string
			frequencyString, err = getIntervalString(frequency)
			if err != nil {
				return nil, err
			}
			params = append(params, frequencyString)
		}
		levelString, okay := channelsToSubscribe[i].Params["level"].(string)
		if okay {
			params = append(params, levelString)
		}
		limit, okay := channelsToSubscribe[i].Params["limit"].(int)
		if okay {
			params = append(params, strconv.Itoa(limit))
		}
		accuracy, okay := channelsToSubscribe[i].Params["accuracy"].(string)
		if okay {
			params = append(params, accuracy)
		}
		switch channelsToSubscribe[i].Channel {
		case futuresCandlesticksChannel:
			interval, okay := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if okay {
				var intervalString string
				intervalString, err = getIntervalString(interval)
				if err != nil {
					return nil, err
				}
				params = append([]string{intervalString}, params...)
			}
		case futuresOrderbookChannel:
			intervalString, okay := channelsToSubscribe[i].Params["interval"].(string)
			if okay {
				params = append(params, intervalString)
			}
		case futuresOrderbookV2:
			level, ok := channelsToSubscribe[i].Params["level"]
			if !ok {
				return nil, fmt.Errorf("%w: %q for %q", common.ErrParameterRequired, "level", futuresOrderbookV2)
			}
			uintLvl, ok := level.(uint64)
			if !ok {
				return nil, common.GetTypeAssertError("uint64", level, "level must be of type uint64")
			}
			if len(params) != 1 || params[0] == "" {
				return nil, fmt.Errorf("%w: currency pair for %q", common.ErrParameterRequired, futuresOrderbookV2)
			}
			params[0] = "ob." + params[0] + "." + strconv.FormatUint(uintLvl, 10)
		}
		outbound[i] = WsInput{
			ID:      e.MessageSequence(),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		}
	}
	return outbound, nil
}

func (e *Exchange) processFuturesTickers(ctx context.Context, data []byte, assetType asset.Item) error {
	resp := struct {
		Time    types.Time       `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsFutureTicker `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	tickerPriceDatas := make([]ticker.Price, len(resp.Result))
	for x := range resp.Result {
		tickerPriceDatas[x] = ticker.Price{
			ExchangeName: e.Name,
			Volume:       resp.Result[x].Volume24HBase.Float64(),
			QuoteVolume:  resp.Result[x].Volume24HQuote.Float64(),
			High:         resp.Result[x].High24H.Float64(),
			Low:          resp.Result[x].Low24H.Float64(),
			Last:         resp.Result[x].Last.Float64(),
			AssetType:    assetType,
			Pair:         resp.Result[x].Contract,
			LastUpdated:  resp.Time.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, tickerPriceDatas)
}

func (e *Exchange) processFuturesTrades(data []byte, assetType asset.Item) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}

	resp := struct {
		Time    types.Time        `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsFuturesTrades `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	trades := make([]trade.Data, len(resp.Result))
	for x := range resp.Result {
		trades[x] = trade.Data{
			Timestamp:    resp.Result[x].CreateTime.Time(),
			CurrencyPair: resp.Result[x].Contract,
			AssetType:    assetType,
			Exchange:     e.Name,
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Size.Float64(),
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

func (e *Exchange) processFuturesCandlesticks(ctx context.Context, data []byte, assetType asset.Item) error {
	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []FuturesCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	klineDatas := make([]kline.Item, len(resp.Result))
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].Name, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return fmt.Errorf("%w: futures candlestick websocket", common.ErrMalformedData)
		}
		interval, err := e.GetIntervalFromString(icp[0])
		if err != nil {
			return err
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		klineDatas[x] = kline.Item{
			Pair:     currencyPair,
			Asset:    assetType,
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:   resp.Result[x].Timestamp.Time(),
				Open:   resp.Result[x].OpenPrice.Float64(),
				Close:  resp.Result[x].ClosePrice.Float64(),
				High:   resp.Result[x].HighestPrice.Float64(),
				Low:    resp.Result[x].LowestPrice.Float64(),
				Volume: resp.Result[x].Volume.Float64(),
			}},
		}
	}
	return e.Websocket.DataHandler.Send(ctx, klineDatas)
}

func (e *Exchange) processFuturesOrderbookTicker(ctx context.Context, incoming []byte) error {
	var data WsFuturesOrderbookTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

func (e *Exchange) processFuturesOrderbookUpdate(ctx context.Context, incoming []byte, a asset.Item, pushTime time.Time) error {
	var data WsFuturesAndOptionsOrderbookUpdate
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	asks := make([]orderbook.Level, len(data.Asks))
	for x := range data.Asks {
		asks[x].Price = data.Asks[x].Price.Float64()
		asks[x].Amount = data.Asks[x].Size.Float64()
	}
	bids := make([]orderbook.Level, len(data.Bids))
	for x := range data.Bids {
		bids[x].Price = data.Bids[x].Price.Float64()
		bids[x].Amount = data.Bids[x].Size.Float64()
	}

	return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, data.FirstUpdatedID, &orderbook.Update{
		UpdateID:   data.LastUpdatedID,
		UpdateTime: data.Timestamp.Time(),
		LastPushed: pushTime,
		Pair:       data.ContractName,
		Asset:      a,
		Asks:       asks,
		Bids:       bids,
		AllowEmpty: true,
	})
}

func (e *Exchange) processFuturesOrderbookSnapshot(event string, incoming []byte, assetType asset.Item, lastPushed time.Time) error {
	if event == "all" {
		var data WsFuturesOrderbookSnapshot
		err := json.Unmarshal(incoming, &data)
		if err != nil {
			return err
		}
		base := orderbook.Book{
			Asset:             assetType,
			Exchange:          e.Name,
			Pair:              data.Contract,
			LastUpdated:       data.Timestamp.Time(),
			LastPushed:        lastPushed,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		base.Asks = make([]orderbook.Level, len(data.Asks))
		for x := range data.Asks {
			base.Asks[x].Amount = data.Asks[x].Size.Float64()
			base.Asks[x].Price = data.Asks[x].Price.Float64()
		}
		base.Bids = make([]orderbook.Level, len(data.Bids))
		for x := range data.Bids {
			base.Bids[x].Amount = data.Bids[x].Size.Float64()
			base.Bids[x].Price = data.Bids[x].Price.Float64()
		}
		return e.Websocket.Orderbook.LoadSnapshot(&base)
	}
	var data []WsFuturesOrderbookUpdateEvent
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Level{}
	for x := range data {
		ab, ok := dataMap[data[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Level{}
		}
		if data[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Level{
				Price:  data[x].Price.Float64(),
				Amount: data[x].Amount.Float64(),
			})
		} else {
			ab[0] = append(ab[0], orderbook.Level{
				Price:  data[x].Price.Float64(),
				Amount: -data[x].Amount.Float64(),
			})
		}
		if !ok {
			dataMap[data[x].CurrencyPair] = ab
		}
	}
	if len(dataMap) == 0 {
		return errors.New("missing orderbook ask and bid data")
	}
	for key, ab := range dataMap {
		currencyPair, err := currency.NewPairFromString(key)
		if err != nil {
			return err
		}
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Asks:              ab[0],
			Bids:              ab[1],
			Asset:             assetType,
			Exchange:          e.Name,
			Pair:              currencyPair,
			LastUpdated:       lastPushed,
			LastPushed:        lastPushed,
			ValidateOrderbook: e.ValidateOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processFuturesOrdersPushData(data []byte, assetType asset.Item) ([]order.Detail, error) {
	fmt.Printf("processFuturesOrdersPushData: %s\n", string(data))
	resp := struct {
		Time    types.Time     `json:"time"`
		Channel string         `json:"channel"`
		Event   string         `json:"event"`
		Result  []FuturesOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	orderDetails := make([]order.Detail, len(resp.Result))
	for x := range resp.Result {
		var status order.Status
		if resp.Result[x].Status == "finished" {
			if resp.Result[x].FinishAs == "ioc" || resp.Result[x].FinishAs == "reduce_only" {
				status = order.Cancelled
			} else {
				status, err = order.StringToOrderStatus(resp.Result[x].FinishAs)
			}
		} else {
			status, err = order.StringToOrderStatus(resp.Result[x].Status)
		}
		if err != nil {
			return nil, err
		}

		orderDetails[x] = order.Detail{
			Amount:         resp.Result[x].Size.Float64(),
			Exchange:       e.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           resp.Result[x].Contract,
			LastUpdated:    resp.Result[x].FinishTime.Time(),
			Date:           resp.Result[x].CreateTime.Time(),
			ExecutedAmount: resp.Result[x].Size.Float64() - resp.Result[x].RemainingAmount.Float64(),
			Price:          resp.Result[x].Price.Float64(),
			AssetType:      assetType,
			AccountID:      resp.Result[x].User,
			CloseTime:      resp.Result[x].FinishTime.Time(),
		}
	}
	return orderDetails, nil
}

func (e *Exchange) processFuturesUserTrades(data []byte, assetType asset.Item) error {
	fmt.Printf("processFuturesUserTrades: %s\n", string(data))
	if !e.IsFillsFeedEnabled() {
		return nil
	}

	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesUserTrade `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(resp.Result))
	for x := range resp.Result {
		fills[x] = fill.Data{
			Timestamp:    resp.Result[x].CreateTime.Time(),
			Exchange:     e.Name,
			CurrencyPair: resp.Result[x].Contract,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price.Float64(),
			Amount:       resp.Result[x].Size.Float64(),
			AssetType:    assetType,
		}
	}
	return e.Websocket.Fills.Update(fills...)
}

func (e *Exchange) processFuturesLiquidatesNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                         `json:"time"`
		Channel string                             `json:"channel"`
		Event   string                             `json:"event"`
		Result  []WsFuturesLiquidationNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processFuturesAutoDeleveragesNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                             `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesAutoDeleveragesNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processPositionCloseData emits zero-size canonical futures positions while
// preserving the established raw payload for options consumers.
func (e *Exchange) processPositionCloseData(ctx context.Context, data []byte, a asset.Item) error {
	fmt.Printf("processPositionCloseData: %s\n", string(data))
	resp := struct {
		Time    types.Time        `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsPositionClose `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if a == asset.Options {
		return e.Websocket.DataHandler.Send(ctx, &resp)
	}
	positions := make([]futures.Position, len(resp.Result))
	for i := range resp.Result {
		pair, err := e.MatchSymbolWithAvailablePairs(resp.Result[i].Contract, a, true)
		if err != nil {
			return err
		}
		direction, err := order.StringToOrderSide(resp.Result[i].Side)
		if err != nil {
			return err
		}
		collateralCurrency, err := getSettlementCurrency(pair, a)
		if err != nil {
			return err
		}
		status := order.Closed
		switch {
		case resp.Result[i].Text == "auto_deleveraging":
			status = order.AutoDeleverage
		case resp.Result[i].Text == "liquidation",
			resp.Result[i].Text == "pm_liquidate",
			resp.Result[i].Text == "comb_margin_liquidate",
			resp.Result[i].Text == "scm_liquidate",
			resp.Result[i].Text == "insurance",
			strings.HasPrefix(resp.Result[i].Text, "liq-"),
			strings.HasPrefix(resp.Result[i].Text, "hedge-liq-"):
			status = order.Liquidated
		}
		positions[i] = futures.Position{
			Exchange:           e.Name,
			Asset:              a,
			Pair:               pair,
			Underlying:         pair.Base,
			CollateralCurrency: collateralCurrency,
			Status:             status,
			RealisedPNL:        resp.Result[i].ProfitAndLoss.Decimal(),
			OpeningDirection:   direction,
			LatestDirection:    direction,
			LastUpdated:        resp.Result[i].Time.Time(),
			CloseDate:          resp.Result[i].Time.Time(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

func (e *Exchange) processBalancePushData(ctx context.Context, data []byte, assetType asset.Item) error {
	fmt.Printf("processBalancePushData: %s\n", string(data))
	var resp []*WsBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	// Gate's websocket user value identifies the same primary account that REST stores with an empty ID.
	// Using it as a subaccount ID retains both snapshots and causes portfolio balances to be double counted.
	subAcct := accounts.NewSubAccount(assetType, "")
	for _, bal := range resp {
		c := bal.Currency
		if c.IsEmpty() {
			var err error
			c, err = getSettlementCurrency(currency.EMPTYPAIR, assetType)
			if err != nil {
				return err
			}
		}
		balance, err := e.Accounts.UpdateBalance(ctx, "", assetType, c, func(balance *accounts.Balance) {
			balance.Total = bal.Balance.Float64()
			balance.Free = max(balance.Total-balance.Hold, 0)
			balance.AvailableWithoutBorrow = balance.Free
			balance.UpdatedAt = time.Time{}
		})
		if err != nil {
			return err
		}
		subAcct.Balances.Set(c, balance)
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAcct})
}

func (e *Exchange) processFuturesReduceRiskLimitNotification(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                             `json:"time"`
		Channel string                                 `json:"channel"`
		Event   string                                 `json:"event"`
		Result  []WsFuturesReduceRiskLimitNotification `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processFuturesPositionsNotification emits canonical positions with Gate's
// signed contract size represented as absolute size plus direction.
func (e *Exchange) processFuturesPositionsNotification(ctx context.Context, data []byte, a asset.Item) error {
	fmt.Printf("processFuturesPositionsNotification: %s\n", string(data))
	resp := struct {
		Time    types.Time          `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsFuturesPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	positions := make([]futures.Position, len(resp.Result))
	for i := range resp.Result {
		pair, err := e.MatchSymbolWithAvailablePairs(resp.Result[i].Contract, a, true)
		if err != nil {
			return err
		}
		size := resp.Result[i].Size.Decimal()
		direction := order.UnknownSide
		switch {
		case size.IsNegative():
			direction = order.Short
			size = size.Abs()
		case size.IsPositive():
			direction = order.Long
		case resp.Result[i].Mode == "dual_long":
			direction = order.Long
		case resp.Result[i].Mode == "dual_short":
			direction = order.Short
		}
		collateralCurrency, err := getSettlementCurrency(pair, a)
		if err != nil {
			return err
		}
		leverage := resp.Result[i].Leverage.Decimal()
		switch {
		case resp.Result[i].PositionMarginMode != "":
			leverage = resp.Result[i].PositionLeverage.Decimal()
		case leverage.IsZero():
			// Gate reports classic cross margin as leverage zero and carries its multiplier separately.
			leverage = resp.Result[i].CrossLeverageLimit.Decimal()
		}
		status := order.Closed
		if !size.IsZero() {
			status = order.Open
		}
		var closeDate time.Time
		if size.IsZero() {
			closeDate = resp.Result[i].Time.Time()
		}
		positions[i] = futures.Position{
			Exchange:                  e.Name,
			Asset:                     a,
			Pair:                      pair,
			Underlying:                pair.Base,
			CollateralCurrency:        collateralCurrency,
			Status:                    status,
			Leverage:                  leverage,
			PositionMargin:            resp.Result[i].Margin.Decimal(),
			MaintenanceMarginFraction: resp.Result[i].MaintenanceRate.Decimal(),
			EstimatedLiquidationPrice: resp.Result[i].LiqPrice.Decimal(),
			UpdateID:                  resp.Result[i].UpdateID,
			RealisedPNL:               resp.Result[i].RealisedPnl.Decimal(),
			OpeningPrice:              resp.Result[i].EntryPrice.Decimal(),
			OpeningDirection:          direction,
			LatestSize:                size,
			LatestDirection:           direction,
			LastUpdated:               resp.Result[i].Time.Time(),
			CloseDate:                 closeDate,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

func (e *Exchange) processFuturesAutoOrderPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsFuturesAutoOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
