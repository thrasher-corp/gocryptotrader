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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
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
	optionsWebsocketURL        = "wss://op-ws.gateio.live/v4/ws"
	optionsWebsocketTestnetURL = "wss://op-ws-testnet.gateio.live/v4/ws"

	// channels
	optionsPingChannel                   = "options.ping"
	optionsContractTickersChannel        = "options.contract_tickers"
	optionsUnderlyingTickersChannel      = "options.ul_tickers"
	optionsTradesChannel                 = "options.trades"
	optionsUnderlyingTradesChannel       = "options.ul_trades"
	optionsUnderlyingPriceChannel        = "options.ul_price"
	optionsMarkPriceChannel              = "options.mark_price"
	optionsSettlementChannel             = "options.settlements"
	optionsContractsChannel              = "options.contracts"
	optionsContractCandlesticksChannel   = "options.contract_candlesticks"
	optionsUnderlyingCandlesticksChannel = "options.ul_candlesticks"
	optionsOrderbookChannel              = "options.order_book"
	optionsOrderbookTickerChannel        = "options.book_ticker"
	optionsOrderbookUpdateChannel        = "options.order_book_update"
	optionsOrdersChannel                 = "options.orders"
	optionsUserTradesChannel             = "options.usertrades"
	optionsLiquidatesChannel             = "options.liquidates"
	optionsUserSettlementChannel         = "options.user_settlements"
	optionsPositionCloseChannel          = "options.position_closes"
	optionsBalancesChannel               = "options.balances"
	optionsPositionsChannel              = "options.positions"

	optionOrderbookUpdateLimit uint64 = 50
)

var defaultOptionsSubscriptions = []string{
	optionsContractTickersChannel,
	optionsUnderlyingTickersChannel,
	optionsTradesChannel,
	optionsUnderlyingTradesChannel,
	optionsContractCandlesticksChannel,
	optionsUnderlyingCandlesticksChannel,
	optionsOrderbookUpdateChannel,
}

// WsOptionsConnect initiates a websocket connection to options websocket endpoints.
func (e *Exchange) WsOptionsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := e.CurrencyPairs.IsAssetEnabled(asset.Options); err != nil {
		return err
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	pingHandler, err := getWSPingHandler(optionsPingChannel)
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, pingHandler)
	return nil
}

// GenerateOptionsDefaultSubscriptions generates list of channel subscriptions for options asset type.
// TODO: Update to use the new subscription template system
func (e *Exchange) GenerateOptionsDefaultSubscriptions() (subscription.List, error) {
	ctx := context.TODO()
	channelsToSubscribe := defaultOptionsSubscriptions
	var userID int64
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		var err error
		_, err = e.GetCredentials(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			goto getEnabledPairs
		}
		response, err := e.GetSubAccountBalances(ctx, "")
		if err != nil {
			return nil, err
		}
		if len(response) != 0 {
			channelsToSubscribe = append(channelsToSubscribe,
				optionsUserTradesChannel,
				optionsBalancesChannel,
			)
			userID = response[0].UserID
		} else if e.Verbose {
			log.Errorf(log.ExchangeSys, "no subaccount found for authenticated options channel subscriptions")
		}
	}

getEnabledPairs:

	pairs, err := e.GetEnabledPairs(asset.Options)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil // no enabled pairs, subscriptions require an associated pair.
		}
		return nil, err
	}

	var subscriptions subscription.List
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]any)
			switch channelsToSubscribe[i] {
			case optionsOrderbookChannel:
				params["accuracy"] = "0"
				params["level"] = "20"
			case optionsContractCandlesticksChannel, optionsUnderlyingCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case optionsOrderbookUpdateChannel:
				params["interval"] = kline.HundredMilliseconds
				params["level"] = strconv.FormatUint(optionOrderbookUpdateLimit, 10)
			case optionsOrdersChannel,
				optionsUserTradesChannel,
				optionsLiquidatesChannel,
				optionsUserSettlementChannel,
				optionsPositionCloseChannel,
				optionsBalancesChannel,
				optionsPositionsChannel:
				if userID == 0 {
					continue
				}
				params["user_id"] = userID
			}
			fPair, err := e.FormatExchangeCurrency(pairs[j], asset.Options)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fPair.Upper()},
				Params:  params,
				Asset:   asset.Options,
			})
		}
	}
	return subscriptions, nil
}

func (e *Exchange) generateOptionsPayload(ctx context.Context, event string, channelsToSubscribe subscription.List) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	var err error
	var intervalString string
	payloads := make([]WsInput, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return nil, subscription.ErrNotSinglePair
		}
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		switch channelsToSubscribe[i].Channel {
		case optionsUnderlyingTickersChannel,
			optionsUnderlyingTradesChannel,
			optionsUnderlyingPriceChannel,
			optionsUnderlyingCandlesticksChannel:
			var uly currency.Pair
			uly, err = e.GetUnderlyingFromCurrencyPair(channelsToSubscribe[i].Pairs[0])
			if err != nil {
				return nil, err
			}
			params = append(params, uly.String())
		case optionsBalancesChannel:
			// options.balance channel does not require underlying or contract
		default:
			channelsToSubscribe[i].Pairs[0].Delimiter = currency.UnderscoreDelimiter
			params = append(params, channelsToSubscribe[i].Pairs[0].String())
		}
		switch channelsToSubscribe[i].Channel {
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
		case optionsUserTradesChannel,
			optionsBalancesChannel,
			optionsOrdersChannel,
			optionsLiquidatesChannel,
			optionsUserSettlementChannel,
			optionsPositionCloseChannel,
			optionsPositionsChannel:
			userID, ok := channelsToSubscribe[i].Params["user_id"].(int64)
			if !ok {
				continue
			}
			params = append([]string{strconv.FormatInt(userID, 10)}, params...)
			var creds *accounts.Credentials
			creds, err = e.GetCredentials(ctx)
			if err != nil {
				return nil, err
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
		case optionsOrderbookUpdateChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, fmt.Errorf("%w, missing options orderbook interval", orderbook.ErrOrderbookInvalid)
			}
			intervalString, err = getIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(params,
				intervalString)
			if value, ok := channelsToSubscribe[i].Params["level"].(int); ok {
				params = append(params, strconv.Itoa(value))
			}
		case optionsContractCandlesticksChannel,
			optionsUnderlyingCandlesticksChannel:
			interval, ok := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if !ok {
				return nil, errors.New("missing options underlying candlesticks interval")
			}
			intervalString, err = getIntervalString(interval)
			if err != nil {
				return nil, err
			}
			params = append(
				[]string{intervalString},
				params...)
		}
		payloads[i] = WsInput{
			ID:      e.MessageSequence(),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		}
	}
	return payloads, nil
}

// OptionsSubscribe sends a websocket message to stop receiving data for asset type options
func (e *Exchange) OptionsSubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, subscribeEvent, channelsToUnsubscribe, e.generateOptionsPayload)
}

// OptionsUnsubscribe sends a websocket message to stop receiving data for asset type options
func (e *Exchange) OptionsUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, unsubscribeEvent, channelsToUnsubscribe, e.generateOptionsPayload)
}

// WsHandleOptionsData handles options websocket data
func (e *Exchange) WsHandleOptionsData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	push, err := parseWSHeader(respRaw)
	if err != nil {
		return err
	}

	if push.Event == subscribeEvent || push.Event == unsubscribeEvent {
		return conn.RequireMatchWithData(push.ID, respRaw)
	}

	switch push.Channel {
	case optionsContractTickersChannel:
		return e.processOptionsContractTickers(ctx, push.Result)
	case optionsUnderlyingTickersChannel:
		return e.processOptionsUnderlyingTicker(ctx, push.Result)
	case optionsTradesChannel,
		optionsUnderlyingTradesChannel:
		return e.processOptionsTradesPushData(respRaw)
	case optionsUnderlyingPriceChannel:
		return e.processOptionsUnderlyingPricePushData(ctx, push.Result)
	case optionsMarkPriceChannel:
		return e.processOptionsMarkPrice(ctx, push.Result)
	case optionsSettlementChannel:
		return e.processOptionsSettlementPushData(ctx, push.Result)
	case optionsContractsChannel:
		return e.processOptionsContractPushData(ctx, push.Result)
	case optionsContractCandlesticksChannel,
		optionsUnderlyingCandlesticksChannel:
		return e.processOptionsCandlestickPushData(ctx, respRaw)
	case optionsOrderbookChannel:
		return e.processOptionsOrderbookSnapshotPushData(push.Event, push.Result, push.Time)
	case optionsOrderbookTickerChannel:
		return e.processOrderbookTickerPushData(ctx, respRaw)
	case optionsOrderbookUpdateChannel:
		return e.processOptionsOrderbookUpdate(ctx, push.Result, asset.Options, push.Time)
	case optionsOrdersChannel:
		return e.processOptionsOrderPushData(ctx, respRaw)
	case optionsUserTradesChannel:
		return e.processOptionsUserTradesPushData(respRaw)
	case optionsLiquidatesChannel:
		return e.processOptionsLiquidatesPushData(ctx, respRaw)
	case optionsUserSettlementChannel:
		return e.processOptionsUsersPersonalSettlementsPushData(ctx, respRaw)
	case optionsPositionCloseChannel:
		return e.processPositionCloseData(ctx, respRaw)
	case optionsBalancesChannel:
		return e.processBalancePushData(ctx, push.Result, asset.Options)
	case optionsPositionsChannel:
		return e.processOptionsPositionPushData(ctx, respRaw)
	case "options.pong":
		return nil
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

func (e *Exchange) processOptionsContractTickers(ctx context.Context, incoming []byte) error {
	var data OptionsTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		Pair:         data.Name,
		Last:         data.LastPrice.Float64(),
		Bid:          data.Bid1Price.Float64(),
		Ask:          data.Ask1Price.Float64(),
		AskSize:      data.Ask1Size,
		BidSize:      data.Bid1Size,
		ExchangeName: e.Name,
		AssetType:    asset.Options,
	})
}

func (e *Exchange) processOptionsUnderlyingTicker(ctx context.Context, incoming []byte) error {
	var data WsOptionUnderlyingTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsTradesPushData(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!e.IsTradeFeedEnabled() {
		return nil
	}
	resp := struct {
		Time    types.Time        `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsOptionsTrades `json:"result"`
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
			AssetType:    asset.Options,
			Exchange:     e.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

func (e *Exchange) processOptionsUnderlyingPricePushData(ctx context.Context, incoming []byte) error {
	var data WsOptionsUnderlyingPrice
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsMarkPrice(ctx context.Context, incoming []byte) error {
	var data WsOptionsMarkPrice
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsSettlementPushData(ctx context.Context, incoming []byte) error {
	var data WsOptionsSettlement
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsContractPushData(ctx context.Context, incoming []byte) error {
	var data WsOptionsContract
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsCandlestickPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                     `json:"time"`
		Channel string                         `json:"channel"`
		Event   string                         `json:"event"`
		Result  []WsOptionsContractCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	klineDatas := make([]websocket.KlineData, len(resp.Result))
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].NameOfSubscription, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return errors.New("malformed options candlestick websocket push data")
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		klineDatas[x] = websocket.KlineData{
			Pair:       currencyPair,
			AssetType:  asset.Options,
			Exchange:   e.Name,
			StartTime:  resp.Result[x].Timestamp.Time(),
			Interval:   icp[0],
			OpenPrice:  resp.Result[x].OpenPrice.Float64(),
			ClosePrice: resp.Result[x].ClosePrice.Float64(),
			HighPrice:  resp.Result[x].HighestPrice.Float64(),
			LowPrice:   resp.Result[x].LowestPrice.Float64(),
			Volume:     resp.Result[x].Amount.Float64(),
		}
	}
	return e.Websocket.DataHandler.Send(ctx, klineDatas)
}

func (e *Exchange) processOrderbookTickerPushData(ctx context.Context, incoming []byte) error {
	var data WsOptionsOrderbookTicker
	err := json.Unmarshal(incoming, &data)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &data)
}

func (e *Exchange) processOptionsOrderbookUpdate(ctx context.Context, incoming []byte, a asset.Item, pushTime time.Time) error {
	var data WsFuturesAndOptionsOrderbookUpdate
	if err := json.Unmarshal(incoming, &data); err != nil {
		return err
	}
	asks := make([]orderbook.Level, len(data.Asks))
	for x := range data.Asks {
		asks[x].Price = data.Asks[x].Price.Float64()
		asks[x].Amount = data.Asks[x].Size
	}
	bids := make([]orderbook.Level, len(data.Bids))
	for x := range data.Bids {
		bids[x].Price = data.Bids[x].Price.Float64()
		bids[x].Amount = data.Bids[x].Size
	}
	return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, e, data.FirstUpdatedID, &orderbook.Update{
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

func (e *Exchange) processOptionsOrderbookSnapshotPushData(event string, incoming []byte, lastPushed time.Time) error {
	if event == "all" {
		var data WsOptionsOrderbookSnapshot
		err := json.Unmarshal(incoming, &data)
		if err != nil {
			return err
		}
		base := orderbook.Book{
			Asset:             asset.Options,
			Exchange:          e.Name,
			Pair:              data.Contract,
			LastUpdated:       data.Timestamp.Time(),
			LastPushed:        lastPushed,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		base.Asks = make([]orderbook.Level, len(data.Asks))
		for x := range data.Asks {
			base.Asks[x].Amount = data.Asks[x].Size
			base.Asks[x].Price = data.Asks[x].Price.Float64()
		}
		base.Bids = make([]orderbook.Level, len(data.Bids))
		for x := range data.Bids {
			base.Bids[x].Amount = data.Bids[x].Size
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
				Price: data[x].Price.Float64(), Amount: data[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Level{
				Price: data[x].Price.Float64(), Amount: -data[x].Amount,
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
			Asset:             asset.Options,
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

func (e *Exchange) processOptionsOrderPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time       `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsOptionsOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	orderDetails := make([]order.Detail, len(resp.Result))
	for x := range resp.Result {
		status, err := order.StringToOrderStatus(func() string {
			if resp.Result[x].Status == "finished" {
				return "cancelled"
			}
			return resp.Result[x].Status
		}())
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       e.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           resp.Result[x].Contract,
			Date:           resp.Result[x].CreationTime.Time(),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      asset.Options,
			AccountID:      resp.Result[x].User,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, orderDetails)
}

func (e *Exchange) processOptionsUserTradesPushData(data []byte) error {
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	resp := struct {
		Time    types.Time           `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsOptionsUserTrade `json:"result"`
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
			Amount:       resp.Result[x].Size,
		}
	}
	return e.Websocket.Fills.Update(fills...)
}

func (e *Exchange) processOptionsLiquidatesPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time            `json:"time"`
		Channel string                `json:"channel"`
		Event   string                `json:"event"`
		Result  []WsOptionsLiquidates `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processOptionsUsersPersonalSettlementsPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time                `json:"time"`
		Channel string                    `json:"channel"`
		Event   string                    `json:"event"`
		Result  []WsOptionsUserSettlement `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

func (e *Exchange) processOptionsPositionPushData(ctx context.Context, data []byte) error {
	resp := struct {
		Time    types.Time          `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsOptionsPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
