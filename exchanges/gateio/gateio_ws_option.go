package gateio

import (
	"context"
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
)

var defaultOptionsSubscriptions = []string{
	optionsContractTickersChannel,
	optionsUnderlyingTickersChannel,
	optionsTradesChannel,
	optionsUnderlyingTradesChannel,
	optionsContractCandlesticksChannel,
	optionsUnderlyingCandlesticksChannel,
	optionsOrderbookChannel,
	optionsOrderbookUpdateChannel,
}

var fetchedOptionsCurrencyPairSnapshotOrderbook = make(map[string]bool)

// WsOptionsConnect initiates a websocket connection to options websocket endpoints.
func (g *Gateio) WsOptionsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() || !g.IsAssetWebsocketSupported(asset.Options) {
		return fmt.Errorf("%s for asset type %s", stream.WebsocketNotEnabled, asset.Options)
	}
	optionsWebsocket, err := g.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		return err
	}
	err = g.CurrencyPairs.IsAssetEnabled(asset.Options)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = optionsWebsocket.SetWebsocketURL(optionsWebsocketURL, false, true)
	if err != nil {
		return err
	}
	err = optionsWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      optionsWebsocket.Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: optionsPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(1)
	go g.wsReadOptionsConnData()
	optionsWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// GenerateOptionsDefaultSubscriptions generates list of channel subscriptions for options asset type.
func (g *Gateio) GenerateOptionsDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channelsToSubscribe := defaultOptionsSubscriptions
	var userID int64
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		var err error
		_, err = g.GetCredentials(context.TODO())
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
			goto getEnabledPairs
		}
		response, err := g.GetSubAccountBalances(context.Background(), "")
		if err != nil {
			return nil, err
		}
		if len(response) != 0 {
			channelsToSubscribe = append(channelsToSubscribe,
				optionsUserTradesChannel,
				optionsBalancesChannel,
			)
			userID = response[0].UserID
		} else if g.Verbose {
			log.Error(log.ExchangeSys, "no subaccount found for authenticated options channel subscriptions")
		}
	}
getEnabledPairs:
	var subscriptions []stream.ChannelSubscription
	pairs, err := g.GetEnabledPairs(asset.Options)
	if err != nil {
		return nil, err
	}
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]interface{})
			switch channelsToSubscribe[i] {
			case optionsOrderbookChannel:
				params["accuracy"] = "0"
				params["level"] = "20"
			case optionsContractCandlesticksChannel, optionsUnderlyingCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case optionsOrderbookUpdateChannel:
				params["interval"] = kline.ThousandMilliseconds
				params["level"] = "20"
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
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Options)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
				Asset:    asset.Options,
			})
		}
	}
	return subscriptions, nil
}

func (g *Gateio) generateOptionsPayload(event string, channelsToSubscribe []stream.ChannelSubscription) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	optionsWebsocket, err := g.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		return []WsInput{}, fmt.Errorf("%w asset type: %v", err, asset.Options)
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
			var uly currency.Pair
			uly, err = g.GetUnderlyingFromCurrencyPair(channelsToSubscribe[i].Currency)
			if err != nil {
				return nil, err
			}
			params = append(params, uly.String())
		case optionsBalancesChannel:
			// options.balance channel does not require underlying or contract
		default:
			channelsToSubscribe[i].Currency.Delimiter = currency.UnderscoreDelimiter
			params = append(params, channelsToSubscribe[i].Currency.String())
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
			var creds *account.Credentials
			creds, err = g.GetCredentials(context.Background())
			if err != nil {
				return nil, err
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
		case optionsContractCandlesticksChannel,
			optionsUnderlyingCandlesticksChannel:
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
		}
		payloads[i] = WsInput{
			ID:      optionsWebsocket.Conn.GenerateMessageID(false),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		}
	}
	return payloads, nil
}

// wsReadOptionsConnData receives and passes on websocket messages for processing
func (g *Gateio) wsReadOptionsConnData() {
	defer g.Websocket.Wg.Done()
	optionsWebsocket, err := g.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v asset type: %v", err, asset.Options)
		return
	}
	for {
		resp := optionsWebsocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := g.wsHandleOptionsData(resp.Raw)
		if err != nil {
			g.Websocket.DataHandler <- err
		}
	}
}

// OptionsSubscribe sends a websocket message to stop receiving data for asset type options
func (g *Gateio) OptionsSubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleOptionsSubscription("subscribe", channelsToUnsubscribe)
}

// OptionsUnsubscribe sends a websocket message to stop receiving data for asset type options
func (g *Gateio) OptionsUnsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleOptionsSubscription("unsubscribe", channelsToUnsubscribe)
}

// handleOptionsSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleOptionsSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	optionsWebsocket, err := g.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Options)
	}
	payloads, err := g.generateOptionsPayload(event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs error
	for k := range payloads {
		var result []byte
		result, err = optionsWebsocket.Conn.SendMessageReturnResponse(payloads[k].ID, payloads[k])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		var resp WsEventResponse
		if err = json.Unmarshal(result, &resp); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			if resp.Error != nil && resp.Error.Code != 0 {
				errs = common.AppendError(errs, fmt.Errorf("error while %s to channel %s asset type: options error code: %d message: %s", payloads[k].Event, payloads[k].Channel, resp.Error.Code, resp.Error.Message))
				continue
			}
			if payloads[k].Event == "subscribe" {
				optionsWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
			} else {
				optionsWebsocket.RemoveSuccessfulUnsubscriptions(channelsToSubscribe[k])
			}
		}
	}
	if err != nil {
		return fmt.Errorf("%v %w", asset.Options, errs)
	}
	return errs
}

func (g *Gateio) wsHandleOptionsData(respRaw []byte) error {
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
		return g.processFuturesAndOptionsOrderbookUpdate(respRaw, asset.Options)
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
		return g.processBalancePushData(respRaw, asset.Options)
	case optionsPositionsChannel:
		return g.processOptionsPositionPushData(respRaw)
	default:
		g.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: g.Name + stream.UnhandledMessage + string(respRaw),
		}
		return errors.New(stream.UnhandledMessage)
	}
}

func (g *Gateio) processOptionsContractTickers(data []byte) error {
	var response WsResponse
	tickerData := OptionsTicker{}
	response.Result = &tickerData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	currencyPair, err := currency.NewPairFromString(tickerData.Name)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &ticker.Price{
		Pair:         currencyPair,
		Last:         tickerData.LastPrice,
		Bid:          tickerData.Bid1Price,
		Ask:          tickerData.Ask1Price,
		AskSize:      tickerData.Ask1Size,
		BidSize:      tickerData.Bid1Size,
		ExchangeName: g.Name,
		AssetType:    asset.Options,
	}
	return nil
}

func (g *Gateio) processOptionsUnderlyingTicker(data []byte) error {
	var response WsResponse
	response.Result = &WsOptionUnderlyingTicker{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsTradesPushData(data []byte) error {
	saveTradeData := g.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!g.IsTradeFeedEnabled() {
		return nil
	}
	resp := struct {
		Time    int64             `json:"time"`
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
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			CurrencyPair: currencyPair,
			AssetType:    asset.Options,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return g.Websocket.Trade.Update(saveTradeData, trades...)
}

func (g *Gateio) processOptionsUnderlyingPricePushData(data []byte) error {
	var response WsResponse
	priceD := WsOptionsUnderlyingPrice{}
	response.Result = &priceD
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsMarkPrice(data []byte) error {
	var response WsResponse
	markPrice := WsOptionsMarkPrice{}
	response.Result = &markPrice
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsSettlementPushData(data []byte) error {
	var response WsResponse
	settlementData := WsOptionsSettlement{}
	response.Result = &settlementData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsContractPushData(data []byte) error {
	var response WsResponse
	contractData := WsOptionsContract{}
	response.Result = &contractData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsCandlestickPushData(data []byte) error {
	resp := struct {
		Time    int64                          `json:"time"`
		Channel string                         `json:"channel"`
		Event   string                         `json:"event"`
		Result  []WsOptionsContractCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	klineDatas := make([]stream.KlineData, len(resp.Result))
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].NameOfSubscription, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return errors.New("malformed options candlestick websocket push data")
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		klineDatas[x] = stream.KlineData{
			Pair:       currencyPair,
			AssetType:  asset.Options,
			Exchange:   g.Name,
			StartTime:  time.Unix(resp.Result[x].Timestamp, 0),
			Interval:   icp[0],
			OpenPrice:  resp.Result[x].OpenPrice,
			ClosePrice: resp.Result[x].ClosePrice,
			HighPrice:  resp.Result[x].HighestPrice,
			LowPrice:   resp.Result[x].LowestPrice,
			Volume:     resp.Result[x].Amount,
		}
	}
	g.Websocket.DataHandler <- klineDatas
	return nil
}

func (g *Gateio) processOrderbookTickerPushData(data []byte) error {
	var response WsResponse
	orderbookTicker := WsOptionsOrderbookTicker{}
	response.Result = &orderbookTicker
	err := json.Unmarshal(data, &orderbookTicker)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsOrderbookSnapshotPushData(event string, data []byte) error {
	if event == "all" {
		var response WsResponse
		snapshot := WsOptionsOrderbookSnapshot{}
		response.Result = &snapshot
		err := json.Unmarshal(data, &response)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(snapshot.Contract)
		if err != nil {
			return err
		}
		base := orderbook.Base{
			Asset:           asset.Options,
			Exchange:        g.Name,
			Pair:            pair,
			LastUpdated:     snapshot.Timestamp.Time(),
			VerifyOrderbook: g.CanVerifyOrderbook,
		}
		base.Asks = make([]orderbook.Item, len(snapshot.Asks))
		base.Bids = make([]orderbook.Item, len(snapshot.Bids))
		for x := range base.Asks {
			base.Asks[x] = orderbook.Item{
				Amount: snapshot.Asks[x].Size,
				Price:  snapshot.Asks[x].Price,
			}
		}
		for x := range base.Bids {
			base.Bids[x] = orderbook.Item{
				Amount: snapshot.Bids[x].Size,
				Price:  snapshot.Bids[x].Price,
			}
		}
		return g.Websocket.Orderbook.LoadSnapshot(&base)
	}
	resp := struct {
		Time    int64                           `json:"time"`
		Channel string                          `json:"channel"`
		Event   string                          `json:"event"`
		Result  []WsFuturesOrderbookUpdateEvent `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Item{}
	for x := range resp.Result {
		ab, ok := dataMap[resp.Result[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Item{}
		}
		if resp.Result[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: resp.Result[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: -resp.Result[x].Amount,
			})
		}
		if !ok {
			dataMap[resp.Result[x].CurrencyPair] = ab
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
		err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asks:            ab[0],
			Bids:            ab[1],
			Asset:           asset.Options,
			Exchange:        g.Name,
			Pair:            currencyPair,
			LastUpdated:     time.Unix(resp.Time, 0),
			VerifyOrderbook: g.CanVerifyOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateio) processOptionsOrderPushData(data []byte) error {
	resp := struct {
		Time    int64            `json:"time"`
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
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
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
			Exchange:       g.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           currencyPair,
			Date:           resp.Result[x].CreationTimeMs.Time(),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      asset.Options,
			AccountID:      resp.Result[x].User,
		}
	}
	g.Websocket.DataHandler <- orderDetails
	return nil
}

func (g *Gateio) processOptionsUserTradesPushData(data []byte) error {
	resp := struct {
		Time    int64                `json:"time"`
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
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			Timestamp:    resp.Result[x].CreateTimeMs.Time(),
			Exchange:     g.Name,
			CurrencyPair: currencyPair,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
		}
	}
	return g.Websocket.Fills.Update(fills...)
}

func (g *Gateio) processOptionsLiquidatesPushData(data []byte) error {
	resp := struct {
		Time    int64                 `json:"time"`
		Channel string                `json:"channel"`
		Event   string                `json:"event"`
		Result  []WsOptionsLiquidates `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processOptionsUsersPersonalSettlementsPushData(data []byte) error {
	resp := struct {
		Time    int64                     `json:"time"`
		Channel string                    `json:"channel"`
		Event   string                    `json:"event"`
		Result  []WsOptionsUserSettlement `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processOptionsPositionPushData(data []byte) error {
	resp := struct {
		Time    int64               `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsOptionsPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}
