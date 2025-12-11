package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	publicBullets  = "/v1/bullet-public"
	privateBullets = "/v1/bullet-private"

	// Spot channels
	marketTickerChannel           = "/market/ticker"            // /market/ticker:{symbol},...
	marketSnapshotChannel         = "/market/snapshot"          // /market/snapshot:{symbol},...
	marketOrderbookChannel        = "/market/level2"            // /market/level2:{symbol},...
	marketOrderbookDepth1Channel  = "/spotMarket/level1"        // /spotMarket/level1:{symbol},...
	marketOrderbookDepth5Channel  = "/spotMarket/level2Depth5"  // /spotMarket/level2Depth5:{symbol},...
	marketOrderbookDepth50Channel = "/spotMarket/level2Depth50" // /spotMarket/level2Depth50:{symbol},...
	marketCandlesChannel          = "/market/candles"           // /market/candles:{symbol}_{interval},...
	marketMatchChannel            = "/market/match"             // /market/match:{symbol},...
	indexPriceIndicatorChannel    = "/indicator/index"          // /indicator/index:{symbol},...
	markPriceIndicatorChannel     = "/indicator/markPrice"      // /indicator/markPrice:{symbol},...

	// Private channels
	privateSpotTradeOrders    = "/spotMarket/tradeOrders"
	accountBalanceChannel     = "/account/balance"
	marginPositionChannel     = "/margin/position"
	marginLoanChannel         = "/margin/loan" // /margin/loan:{currency}
	spotMarketAdvancedChannel = "/spotMarket/advancedOrders"

	// Futures channels
	futuresTransactionStatisticsTimerEventChannel = "/contractMarket/snapshot"      // /contractMarket/snapshot:{symbol}
	futuresTickerChannel                          = "/contractMarket/tickerV2"      // /contractMarket/tickerV2:{symbol},...
	futuresOrderbookChannel                       = "/contractMarket/level2"        // /contractMarket/level2:{symbol},...
	futuresOrderbookDepth5Channel                 = "/contractMarket/level2Depth5"  // /contractMarket/level2Depth5:{symbol},...
	futuresOrderbookDepth50Channel                = "/contractMarket/level2Depth50" // /contractMarket/level2Depth50:{symbol},...
	futuresExecutionDataChannel                   = "/contractMarket/execution"     // /contractMarket/execution:{symbol},...
	futuresContractMarketDataChannel              = "/contract/instrument"          // /contract/instrument:{symbol},...
	futuresSystemAnnouncementChannel              = "/contract/announcement"
	futuresTrasactionStatisticsTimerEventChannel  = "/contractMarket/snapshot" // /contractMarket/snapshot:{symbol},...

	// futures private channels
	futuresTradeOrderChannel               = "/contractMarket/tradeOrders" // /contractMarket/tradeOrders:{symbol},...
	futuresPositionChangeEventChannel      = "/contract/position"          // /contract/position:{symbol},...
	futuresStopOrdersLifecycleEventChannel = "/contractMarket/advancedOrders"
	futuresAccountBalanceEventChannel      = "/contractAccount/wallet"

	futuresLimitCandles = "/contractMarket/limitCandle"

	wsConnection = "websocket_connection"
)

var subscriptionNames = map[asset.Item]map[string]string{
	asset.Futures: {
		subscription.TickerChannel:    futuresTickerChannel,
		subscription.OrderbookChannel: futuresOrderbookDepth5Channel, // This does not require a REST request to get the orderbook.
	},
	asset.All: {
		subscription.TickerChannel:    marketTickerChannel,
		subscription.OrderbookChannel: marketOrderbookDepth5Channel, // This does not require a REST request to get the orderbook.
		subscription.CandlesChannel:   marketCandlesChannel,
		subscription.AllTradesChannel: marketMatchChannel,
	},
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	{Enabled: false, Asset: asset.Spot, Channel: marketOrderbookChannel, Authenticated: true}, // Full orderbook depth requires REST snapshot which is an authenticated request.
	{Enabled: false, Asset: asset.Futures, Channel: futuresOrderbookChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Margin, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Futures, Channel: futuresTradeOrderChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: futuresStopOrdersLifecycleEventChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: futuresAccountBalanceEventChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Margin, Channel: marginPositionChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Margin, Channel: marginLoanChannel, Authenticated: true},
	{Enabled: true, Channel: accountBalanceChannel, Authenticated: true},
}

// WsConnect creates a new websocket connection.
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	var instances *WSInstanceServers
	var err error
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		instances, err = e.GetAuthenticatedInstanceServers(ctx)
	} else {
		instances, err = e.GetInstanceServers(ctx)
	}
	if err != nil {
		return err
	}
	if len(instances.InstanceServers) == 0 {
		return errors.New("no websocket instance server found")
	}

	if conn.GetURL() != instances.InstanceServers[0].Endpoint {
		log.Warnf(log.WebsocketMgr, "%s websocket endpoint has changed, overriding old: %s with new: %s", e.Name, conn.GetURL(), instances.InstanceServers[0].Endpoint)
		conn.SetURL(instances.InstanceServers[0].Endpoint)
	}

	values := url.Values{}
	values.Set("token", instances.Token)

	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	if err := conn.Dial(ctx, &dialer, nil, values); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       time.Millisecond * time.Duration(instances.InstanceServers[0].PingInterval),
		Message:     []byte(`{"type":"ping"}`),
		MessageType: gws.TextMessage,
	})
	return nil
}

// GetInstanceServers retrieves the server list and temporary public token
func (e *Exchange) GetInstanceServers(ctx context.Context) (*WSInstanceServers, error) {
	response := struct {
		Data WSInstanceServers `json:"data"`
		Error
	}{}
	return &(response.Data), e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpot)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpointPath + publicBullets,
			Result:                 &response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
}

// GetAuthenticatedInstanceServers retrieves server instances for authenticated users.
func (e *Exchange) GetAuthenticatedInstanceServers(ctx context.Context) (*WSInstanceServers, error) {
	response := struct {
		Data *WSInstanceServers `json:"data"`
		Error
	}{}
	err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, spotAuthenticationEPL, http.MethodPost, privateBullets, nil, &response)
	if err != nil && strings.Contains(err.Error(), "400003") {
		return response.Data, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAuthenticationEPL, http.MethodPost, privateBullets, nil, &response)
	}
	return response.Data, err
}

// wsHandleData processes a websocket incoming data.
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respData []byte) error {
	var resp WsPushData
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	if resp.Type == "pong" || resp.Type == "welcome" {
		return nil
	}
	if resp.ID != "" {
		return conn.RequireMatchWithData(resp.ID, respData)
	}

	switch topicInfo := strings.Split(resp.Topic, ":"); topicInfo[0] {
	case marketTickerChannel:
		var instruments string
		if topicInfo[1] == "all" {
			instruments = resp.Subject
		} else {
			instruments = topicInfo[1]
		}
		return e.processTicker(resp.Data, instruments, topicInfo[0])
	case marketSnapshotChannel:
		return e.processMarketSnapshot(resp.Data, topicInfo[0])
	case marketOrderbookChannel:
		return e.processOrderbookWithDepth(ctx, respData, topicInfo[1], topicInfo[0])
	case marketOrderbookDepth1Channel, marketOrderbookDepth5Channel, marketOrderbookDepth50Channel:
		return e.processOrderbook(resp.Data, topicInfo[1], topicInfo[0])
	case marketCandlesChannel:
		symbolAndInterval := strings.Split(topicInfo[1], currency.UnderscoreDelimiter)
		if len(symbolAndInterval) != 2 {
			return errMalformedData
		}
		return e.processCandlesticks(resp.Data, symbolAndInterval[0], symbolAndInterval[1], topicInfo[0])
	case marketMatchChannel:
		return e.processTradeData(resp.Data, topicInfo[1], topicInfo[0])
	case indexPriceIndicatorChannel, markPriceIndicatorChannel:
		var response WsPriceIndicator
		return e.processData(resp.Data, &response)
	case privateSpotTradeOrders:
		return e.processOrderChangeEvent(resp.Data, topicInfo[0])
	case accountBalanceChannel:
		return e.processAccountBalanceChange(ctx, resp.Data)
	case marginPositionChannel:
		if resp.Subject == "debt.ratio" {
			var response WsDebtRatioChange
			return e.processData(resp.Data, &response)
		}
		var response WsPositionStatus
		return e.processData(resp.Data, &response)
	case marginLoanChannel:
		if resp.Subject == "order.done" {
			var response WsMarginTradeOrderDoneEvent
			return e.processData(resp.Data, &response)
		}
		return e.processMarginLendingTradeOrderEvent(resp.Data)
	case spotMarketAdvancedChannel:
		return e.processStopOrderEvent(resp.Data)
	case futuresTickerChannel:
		return e.processFuturesTickerV2(resp.Data)
	case futuresExecutionDataChannel:
		var response WsFuturesExecutionData
		return e.processData(resp.Data, &response)
	case futuresOrderbookChannel:
		return e.processFuturesOrderbookLevel2(ctx, resp.Data, topicInfo[1])
	case futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
		return e.processFuturesOrderbookSnapshot(resp.Data, topicInfo[1])
	case futuresContractMarketDataChannel:
		switch resp.Subject {
		case "mark.index.price":
			return e.processFuturesMarkPriceAndIndexPrice(resp.Data, topicInfo[1])
		case "funding.rate":
			return e.processFuturesFundingData(resp.Data, topicInfo[1])
		}
	case futuresSystemAnnouncementChannel:
		return e.processFuturesSystemAnnouncement(resp.Data, resp.Subject)
	case futuresTransactionStatisticsTimerEventChannel:
		return e.processFuturesTransactionStatistics(resp.Data, topicInfo[1])
	case futuresTradeOrderChannel:
		return e.processFuturesPrivateTradeOrders(resp.Data)
	case futuresStopOrdersLifecycleEventChannel:
		return e.processFuturesStopOrderLifecycleEvent(resp.Data)
	case futuresAccountBalanceEventChannel:
		switch resp.Subject {
		case "orderMargin.change":
			var response WsFuturesOrderMarginEvent
			return e.processData(resp.Data, &response)
		case "availableBalance.change":
			return e.processFuturesAccountBalanceEvent(ctx, resp.Data)
		case "withdrawHold.change":
			var response WsFuturesWithdrawalAmountAndTransferOutAmountEvent
			return e.processData(resp.Data, &response)
		}
	case futuresPositionChangeEventChannel:
		switch resp.Subject {
		case "position.change":
			if resp.ChannelType == "private" {
				var response WsFuturesPosition
				return e.processData(resp.Data, &response)
			}
			var response WsFuturesMarkPricePositionChanges
			return e.processData(resp.Data, &response)
		case "position.settlement":
			var response WsFuturesPositionFundingSettlement
			return e.processData(resp.Data, &response)
		}
	case futuresLimitCandles:
		instrumentInfos := strings.Split(topicInfo[1], "_")
		if len(instrumentInfos) != 2 {
			return errors.New("invalid instrument information")
		}
		return e.processFuturesKline(resp.Data, instrumentInfos[1])
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respData),
		}
		return errors.New("push data not handled")
	}
	return nil
}

// processData used to deserialize and forward the data to DataHandler.
func (e *Exchange) processData(respData []byte, resp any) error {
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// processFuturesAccountBalanceEvent used to process futures account balance change incoming data.
func (e *Exchange) processFuturesAccountBalanceEvent(ctx context.Context, respData []byte) error {
	resp := WsFuturesAvailableBalance{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
		Total:     resp.AvailableBalance + resp.HoldBalance,
		Hold:      resp.HoldBalance,
		Free:      resp.AvailableBalance,
		UpdatedAt: resp.Timestamp.Time(),
	})
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

// processFuturesStopOrderLifecycleEvent processes futures stop orders lifecycle events.
func (e *Exchange) processFuturesStopOrderLifecycleEvent(respData []byte) error {
	resp := WsStopOrderLifecycleEvent{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	var enabledPairs currency.Pairs
	enabledPairs, err = e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	pair, err := enabledPairs.DeriveFrom(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &order.Detail{
		Price:        resp.OrderPrice,
		TriggerPrice: resp.StopPrice,
		Amount:       resp.Size,
		Exchange:     e.Name,
		OrderID:      resp.OrderID,
		Type:         oType,
		Side:         side,
		AssetType:    asset.Futures,
		Date:         resp.CreatedAt.Time(),
		LastUpdated:  resp.Timestamp.Time(),
		Pair:         pair,
	}
	return nil
}

// processFuturesPrivateTradeOrders processes futures private trade orders updates.
func (e *Exchange) processFuturesPrivateTradeOrders(respData []byte) error {
	resp := WsFuturesTradeOrder{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	oStatus, err := e.StringToOrderStatus(resp.Status)
	if err != nil {
		return err
	}
	var enabledPairs currency.Pairs
	enabledPairs, err = e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	pair, err := enabledPairs.DeriveFrom(resp.Symbol)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &order.Detail{
		Type:            oType,
		Status:          oStatus,
		Pair:            pair,
		Side:            side,
		Amount:          resp.OrderSize,
		Price:           resp.OrderPrice,
		Exchange:        e.Name,
		ExecutedAmount:  resp.FilledSize,
		RemainingAmount: resp.RemainSize,
		ClientOrderID:   resp.ClientOid,
		OrderID:         resp.TradeID,
		AssetType:       asset.Futures,
		LastUpdated:     resp.OrderTime.Time(),
	}
	return nil
}

// processFuturesTransactionStatistics processes a futures transaction statistics
func (e *Exchange) processFuturesTransactionStatistics(respData []byte, instrument string) error {
	resp := WsFuturesTransactionStatisticsTimeEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	return nil
}

// processFuturesSystemAnnouncement processes a system announcement.
func (e *Exchange) processFuturesSystemAnnouncement(respData []byte, subject string) error {
	resp := WsFuturesFundingBegin{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Subject = subject
	e.Websocket.DataHandler <- &resp
	return nil
}

// processFuturesFundingData processes a futures account funding data.
func (e *Exchange) processFuturesFundingData(respData []byte, instrument string) error {
	resp := WsFundingRate{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	e.Websocket.DataHandler <- &resp
	return nil
}

// processFuturesMarkPriceAndIndexPrice processes a futures account mark price and index price changes.
func (e *Exchange) processFuturesMarkPriceAndIndexPrice(respData []byte, instrument string) error {
	resp := WsFuturesMarkPriceAndIndexPrice{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	e.Websocket.DataHandler <- &resp
	return nil
}

// processFuturesOrderbookSnapshot processes a futures account orderbook websocket update.
func (e *Exchange) processFuturesOrderbookSnapshot(respData []byte, instrument string) error {
	var resp WsFuturesOrderbookLevelResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(instrument, asset.Futures, false)
	if err != nil {
		return err
	}
	// Note: KuCoin snapshot timestamps are all the same and each update is 100ms apart.
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:     e.Name,
		LastUpdateID: resp.Sequence,
		LastUpdated:  resp.Timestamp.Time(),
		LastPushed:   resp.PushTimestamp.Time(),
		Asset:        asset.Futures,
		Bids:         resp.Bids.Levels(),
		Asks:         resp.Asks.Levels(),
		Pair:         pair,
	})
}

// ProcessFuturesOrderbookLevel2 processes a V2 futures account orderbook data.
func (e *Exchange) processFuturesOrderbookLevel2(ctx context.Context, respData []byte, instrument string) error {
	pair, err := e.MatchSymbolWithAvailablePairs(instrument, asset.Futures, false)
	if err != nil {
		return err
	}

	var resp WsFuturesOrderbookInfo
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}

	parts := strings.Split(resp.Change, ",")
	if len(parts) != 3 {
		return fmt.Errorf("unexpected orderbook change format: %s", resp.Change)
	}

	price, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return err
	}

	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return err
	}

	var bids, asks []orderbook.Level
	switch parts[1] {
	case "buy":
		bids = []orderbook.Level{{Price: price, Amount: amount, ID: resp.Sequence}}
	case "sell":
		asks = []orderbook.Level{{Price: price, Amount: amount, ID: resp.Sequence}}
	default:
		return fmt.Errorf("unexpected orderbook side: %q", parts[1])
	}

	return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, resp.Sequence, &orderbook.Update{
		UpdateTime: resp.Timestamp.Time(),
		LastPushed: resp.Timestamp.Time(),
		UpdateID:   resp.Sequence,
		Pair:       pair,
		Asset:      asset.Futures,
		Asks:       asks,
		Bids:       bids,
	})
}

// processFuturesTickerV2 processes a futures account ticker data.
func (e *Exchange) processFuturesTickerV2(respData []byte) error {
	resp := WsFuturesTicker{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	pair, err := enabledPairs.DeriveFrom(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &ticker.Price{
		AssetType:    asset.Futures,
		Last:         resp.FilledPrice.Float64(),
		Volume:       resp.FilledSize.Float64(),
		LastUpdated:  resp.FilledTime.Time(),
		ExchangeName: e.Name,
		Pair:         pair,
		Ask:          resp.BestAskPrice.Float64(),
		Bid:          resp.BestBidPrice.Float64(),
		AskSize:      resp.BestAskSize.Float64(),
		BidSize:      resp.BestBidSize.Float64(),
	}
	return nil
}

// processFuturesKline represents a futures instrument kline data update.
func (e *Exchange) processFuturesKline(respData []byte, intervalStr string) error {
	resp := WsFuturesKline{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	var pair currency.Pair
	pair, err = currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &websocket.KlineData{
		Timestamp:  resp.Time.Time(),
		AssetType:  asset.Futures,
		Exchange:   e.Name,
		StartTime:  time.Unix(resp.Candles[0].Int64(), 0),
		Interval:   intervalStr,
		OpenPrice:  resp.Candles[1].Float64(),
		ClosePrice: resp.Candles[2].Float64(),
		HighPrice:  resp.Candles[3].Float64(),
		LowPrice:   resp.Candles[4].Float64(),
		Volume:     resp.Candles[6].Float64(),
		Pair:       pair,
	}
	return nil
}

// processStopOrderEvent represents a stop order update event.
func (e *Exchange) processStopOrderEvent(respData []byte) error {
	resp := WsStopOrder{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	var pair currency.Pair
	pair, err = currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- &order.Detail{
		Price:        resp.OrderPrice,
		TriggerPrice: resp.StopPrice,
		Amount:       resp.Size,
		Exchange:     e.Name,
		OrderID:      resp.OrderID,
		Type:         oType,
		Side:         side,
		AssetType:    asset.Spot,
		Date:         resp.CreatedAt.Time(),
		LastUpdated:  resp.Timestamp.Time(),
		Pair:         pair,
	}
	return nil
}

// processMarginLendingTradeOrderEvent represents a margin lending trade order event.
func (e *Exchange) processMarginLendingTradeOrderEvent(respData []byte) error {
	resp := WsMarginTradeOrderEntersEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// processAccountBalanceChange processes an account balance change
func (e *Exchange) processAccountBalanceChange(ctx context.Context, respData []byte) error {
	resp := WsAccountBalance{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
		Total:     resp.Total,
		Hold:      resp.Hold,
		Free:      resp.Available,
		UpdatedAt: resp.Time.Time(),
	})
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

// processOrderChangeEvent processes order update events.
func (e *Exchange) processOrderChangeEvent(respData []byte, topic string) error {
	response := WsTradeOrder{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(response.OrderType)
	if err != nil {
		return err
	}
	oStatus, err := e.StringToOrderStatus(response.Status)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(response.Symbol)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(response.Side)
	if err != nil {
		return err
	}
	// TODO: should amend this function as we need to know the order asset type when we call it
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		e.Websocket.DataHandler <- &order.Detail{
			Price:           response.Price,
			Amount:          response.Size,
			ExecutedAmount:  response.FilledSize,
			RemainingAmount: response.RemainSize,
			Exchange:        e.Name,
			OrderID:         response.OrderID,
			ClientOrderID:   response.ClientOid,
			Type:            oType,
			Side:            side,
			Status:          oStatus,
			AssetType:       assets[x],
			Date:            response.OrderTime.Time(),
			LastUpdated:     response.Timestamp.Time(),
			Pair:            pair,
		}
	}
	return nil
}

// processTradeData processes a websocket trade data and instruments.
func (e *Exchange) processTradeData(respData []byte, instrument, topic string) error {
	response := WsTrade{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!e.IsTradeFeedEnabled() {
		return nil
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(response.Side)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		err = e.Websocket.Trade.Update(saveTradeData, trade.Data{
			CurrencyPair: pair,
			Timestamp:    response.Time.Time(),
			Price:        response.Price,
			Amount:       response.Size,
			Side:         side,
			Exchange:     e.Name,
			TID:          response.TradeID,
			AssetType:    assets[x],
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// processTicker processes a ticker data for an instrument.
func (e *Exchange) processTicker(respData []byte, instrument, topic string) error {
	response := WsTicker{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		e.Websocket.DataHandler <- &ticker.Price{
			AssetType:    assets[x],
			Last:         response.Price,
			LastUpdated:  response.Timestamp.Time(),
			ExchangeName: e.Name,
			Pair:         pair,
			Ask:          response.BestAsk,
			Bid:          response.BestBid,
			AskSize:      response.BestAskSize,
			BidSize:      response.BestBidSize,
			Volume:       response.Size,
		}
	}
	return nil
}

// processCandlesticks processes a candlestick data for an instrument with a particular interval
func (e *Exchange) processCandlesticks(respData []byte, instrument, intervalString, topic string) error {
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	var resp WsCandlestick
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		e.Websocket.DataHandler <- &websocket.KlineData{
			Timestamp:  resp.Time.Time(),
			Pair:       pair,
			AssetType:  assets[x],
			Exchange:   e.Name,
			StartTime:  resp.Candles.StartTime.Time(),
			Interval:   intervalString,
			OpenPrice:  resp.Candles.OpenPrice.Float64(),
			ClosePrice: resp.Candles.ClosePrice.Float64(),
			HighPrice:  resp.Candles.HighPrice.Float64(),
			LowPrice:   resp.Candles.LowPrice.Float64(),
			Volume:     resp.Candles.TransactionVolume.Float64(),
		}
	}
	return nil
}

// processOrderbookWithDepth processes order book data with a specified depth for a particular symbol.
func (e *Exchange) processOrderbookWithDepth(ctx context.Context, respData []byte, instrument, topic string) error {
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	var resp struct {
		Result *WsOrderbook `json:"data"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}

	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}

	bids := make([]orderbook.Level, len(resp.Result.Changes.Bids))
	for i := range resp.Result.Changes.Bids {
		bids[i] = orderbook.Level{
			Price:  resp.Result.Changes.Bids[i][0].Float64(),
			Amount: resp.Result.Changes.Bids[i][1].Float64(),
			ID:     resp.Result.Changes.Bids[i][2].Int64(),
		}
	}

	asks := make([]orderbook.Level, len(resp.Result.Changes.Asks))
	for i := range resp.Result.Changes.Asks {
		asks[i] = orderbook.Level{
			Price:  resp.Result.Changes.Asks[i][0].Float64(),
			Amount: resp.Result.Changes.Asks[i][1].Float64(),
			ID:     resp.Result.Changes.Asks[i][2].Int64(),
		}
	}

	for _, a := range assets {
		if err := e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, resp.Result.SequenceStart, &orderbook.Update{
			UpdateID:   resp.Result.SequenceEnd,
			UpdateTime: resp.Result.TimeMS.Time(),
			LastPushed: resp.Result.TimeMS.Time(), // Realtime so this is pushed when a change occurs
			Asset:      a,
			Bids:       bids,
			Asks:       asks,
			Pair:       pair,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processOrderbook processes orderbook data for a specific symbol.
func (e *Exchange) processOrderbook(respData []byte, symbol, topic string) error {
	var resp Level2Depth5Or20
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}

	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}

	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}

	lastUpdatedTime := resp.Timestamp.Time()
	if lastUpdatedTime.IsZero() {
		lastUpdatedTime = time.Now()
	}
	for x := range assets {
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Exchange:    e.Name,
			Asks:        resp.Asks.Levels(),
			Bids:        resp.Bids.Levels(),
			Pair:        pair,
			Asset:       assets[x],
			LastUpdated: lastUpdatedTime,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// processMarketSnapshot processes a price ticker information for a symbol.
func (e *Exchange) processMarketSnapshot(respData []byte, topic string) error {
	response := WsSnapshot{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(response.Data.Symbol)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		e.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: e.Name,
			AssetType:    assets[x],
			Last:         response.Data.LastTradedPrice,
			Pair:         pair,
			Low:          response.Data.Low,
			High:         response.Data.High,
			QuoteVolume:  response.Data.VolValue,
			Volume:       response.Data.Vol,
			Open:         response.Data.Open,
			Close:        response.Data.Close,
			LastUpdated:  response.Data.Datetime.Time(),
		}
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.manageSubscriptions(ctx, conn, subscriptions, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subscriptions subscription.List) error {
	return e.manageSubscriptions(ctx, conn, subscriptions, "unsubscribe")
}

func (e *Exchange) manageSubscriptions(ctx context.Context, conn websocket.Connection, subs subscription.List, operation string) error {
	var errs error
	for _, s := range subs {
		req := WsSubscriptionInput{
			ID:             e.MessageID(),
			Type:           operation,
			Topic:          s.QualifiedChannel,
			PrivateChannel: s.Authenticated,
			Response:       true,
		}
		if respRaw, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			rType, err := jsonparser.GetUnsafeString(respRaw, "type")
			switch {
			case err != nil:
				errs = common.AppendError(errs, err)
			case rType == "error":
				code, _ := jsonparser.GetUnsafeString(respRaw, "code")
				msg, msgErr := jsonparser.GetUnsafeString(respRaw, "data")
				if msgErr != nil {
					msg = "unknown error"
				}
				errs = common.AppendError(errs, fmt.Errorf("%s (%s)", msg, code))
			case rType != "ack":
				errs = common.AppendError(errs, fmt.Errorf("%w: %s from %s", errInvalidMsgType, rType, respRaw))
			default:
				if operation == "unsubscribe" {
					err = e.Websocket.RemoveSubscriptions(conn, s)
				} else {
					err = e.Websocket.AddSuccessfulSubscriptions(conn, s)
					if e.Verbose {
						log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s", e.Name, s.Channel)
					}
				}
				if err != nil {
					errs = common.AppendError(errs, err)
				}
			}
		}
	}
	return errs
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").
		Funcs(template.FuncMap{
			"channelName":           channelName,
			"mergeMarginPairs":      e.mergeMarginPairs,
			"isCurrencyChannel":     isCurrencyChannel,
			"isSymbolChannel":       isSymbolChannel,
			"channelInterval":       channelInterval,
			"assetCurrencies":       assetCurrencies,
			"joinPairsWithInterval": joinPairsWithInterval,
			"batch":                 common.Batch[currency.Pairs],
		}).
		Parse(subTplText)
}

// CalculateAssets returns the available asset types for a currency pair
func (e *Exchange) CalculateAssets(topic string, cp currency.Pair) ([]asset.Item, error) {
	switch {
	case cp.Quote.Equal(currency.USDTM), strings.HasPrefix(topic, "/contract"):
		if err := e.CurrencyPairs.IsAssetEnabled(asset.Futures); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Futures}, nil
	case strings.HasPrefix(topic, "/margin"), strings.HasPrefix(topic, "/index"):
		if err := e.CurrencyPairs.IsAssetEnabled(asset.Margin); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Margin}, nil
	default:
		resp := make([]asset.Item, 0, 2)
		spotEnabled, err := e.IsPairEnabled(cp, asset.Spot)
		if err != nil && !errors.Is(err, currency.ErrCurrencyNotFound) {
			return nil, err
		}
		if spotEnabled {
			resp = append(resp, asset.Spot)
		}
		marginEnabled, err := e.IsPairEnabled(cp, asset.Margin)
		if err != nil && !errors.Is(err, currency.ErrCurrencyNotFound) {
			return nil, err
		}
		if marginEnabled {
			resp = append(resp, asset.Margin)
		}
		return resp, nil
	}
}

// checkSubscriptions looks for any backwards incompatibilities with missing assets
// This should be unnecessary and removable by 2025
func (e *Exchange) checkSubscriptions() {
	upgraded := false
	for _, s := range e.Config.Features.Subscriptions {
		if s.Asset != asset.Empty {
			continue
		}
		upgraded = true
		s.Channel = strings.TrimSuffix(s.Channel, ":%s")
		switch s.Channel {
		case subscription.TickerChannel, subscription.OrderbookChannel:
			s.Asset = asset.All
		case subscription.AllTradesChannel:
			for _, d := range defaultSubscriptions {
				if d.Channel == s.Channel {
					e.Config.Features.Subscriptions = append(e.Config.Features.Subscriptions, d)
				}
			}
		case futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel, futuresAccountBalanceEventChannel:
			s.Asset = asset.Futures
		case marginPositionChannel, marginLoanChannel:
			s.Asset = asset.Margin
		}
	}
	e.Config.Features.Subscriptions = slices.DeleteFunc(e.Config.Features.Subscriptions, func(s *subscription.Subscription) bool {
		switch s.Channel {
		case "/contractMarket/level2Depth50", // Replaced by subsctiption.Orderbook for asset.All
			"/contractMarket/tickerV2", // Replaced by subscription.Ticker for asset.All
			"/margin/fundingBook":      // Deprecated and removed
			return true
		case subscription.AllTradesChannel:
			return s.Asset == asset.Empty
		}
		return false
	})
	if upgraded {
		e.Features.Subscriptions = e.Config.Features.Subscriptions.Enabled()
	}
}

// channelName returns the correct channel name for the asset
func channelName(s *subscription.Subscription, a asset.Item) string {
	if byAsset, hasAsset := subscriptionNames[a]; hasAsset {
		if name, ok := byAsset[s.Channel]; ok {
			return name
		}
	}
	if allAssets, hasAll := subscriptionNames[asset.All]; hasAll {
		if name, ok := allAssets[s.Channel]; ok {
			return name
		}
	}
	return s.Channel
}

// mergeMarginPairs merges margin pairs into spot pairs for shared subs (ticker, orderbook, etc) if Spot asset and sub are enabled,
// because Kucoin errors on duplicate pairs in separate subs, and doesn't have separate subs for spot and margin
func (e *Exchange) mergeMarginPairs(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) string {
	if strings.HasPrefix(s.Channel, "/margin") {
		return ""
	}
	wantKey := &subscription.IgnoringAssetKey{Subscription: s}
	switch s.Asset {
	case asset.All:
		_, marginEnabled := ap[asset.Margin]
		_, spotEnabled := ap[asset.Spot]
		if marginEnabled && spotEnabled {
			marginPairs, _ := e.GetEnabledPairs(asset.Margin)
			ap[asset.Spot] = common.SortStrings(ap[asset.Spot].Add(marginPairs...))
			ap[asset.Margin] = currency.Pairs{}
		}
	case asset.Spot:
		// If there's a margin sub then we should merge the pairs into spot
		hasMarginSub := slices.ContainsFunc(e.Features.Subscriptions, func(sB *subscription.Subscription) bool {
			if sB.Asset != asset.Margin && sB.Asset != asset.All {
				return false
			}
			return wantKey.Match(&subscription.IgnoringAssetKey{Subscription: sB})
		})
		if hasMarginSub {
			marginPairs, _ := e.GetEnabledPairs(asset.Margin)
			ap[asset.Spot] = common.SortStrings(ap[asset.Spot].Add(marginPairs...))
		}
	case asset.Margin:
		// If there's a spot sub, all margin pairs are already merged, so empty the margin pairs
		hasSpotSub := slices.ContainsFunc(e.Features.Subscriptions, func(sB *subscription.Subscription) bool {
			if sB.Asset != asset.Spot && sB.Asset != asset.All {
				return false
			}
			return wantKey.Match(&subscription.IgnoringAssetKey{Subscription: sB})
		})
		if hasSpotSub {
			ap[asset.Margin] = currency.Pairs{}
		}
	}
	return ""
}

// isSymbolChannel returns if the channel expects receive a symbol
func isSymbolChannel(s *subscription.Subscription) bool {
	switch channelName(s, s.Asset) {
	case privateSpotTradeOrders, accountBalanceChannel, marginPositionChannel, spotMarketAdvancedChannel, futuresSystemAnnouncementChannel,
		futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel, futuresAccountBalanceEventChannel:
		return false
	}
	return true
}

// isCurrencyChannel returns if the channel expects receive a currency
func isCurrencyChannel(s *subscription.Subscription) bool {
	return s.Channel == marginLoanChannel
}

// channelInterval returns the channel interval if it has one
func channelInterval(s *subscription.Subscription) string {
	if channelName(s, s.Asset) == marketCandlesChannel {
		if i, err := IntervalToString(s.Interval); err == nil {
			return i
		}
	}
	return ""
}

// assetCurrencies returns the currencies from all pairs in an asset
// Updates the AssetPairs map parameter to contain only those currencies as Base items for expandTemplates to see
func assetCurrencies(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) currency.Currencies {
	cs := common.SortStrings(ap[s.Asset].GetCurrencies())
	p := currency.Pairs{}
	for _, c := range cs {
		p = append(p, currency.Pair{Base: c})
	}
	ap[s.Asset] = p
	return cs
}

// joinPairsWithInterval returns a list of currency pair symbols joined by comma
// If the subscription has a viable interval it's appended after each symbol
func joinPairsWithInterval(b currency.Pairs, s *subscription.Subscription) string {
	out := make([]string, len(b))
	suffix, err := IntervalToString(s.Interval)
	if err == nil {
		suffix = "_" + suffix
	}
	for i, p := range b {
		out[i] = p.String() + suffix
	}
	return strings.Join(out, ",")
}

const subTplText = `
{{- mergeMarginPairs $.S $.AssetPairs }}
{{- if isCurrencyChannel $.S }}
	{{- channelName $.S $.S.Asset -}} : {{- (assetCurrencies $.S $.AssetPairs).Join }}
{{- else if isSymbolChannel $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- with $name := channelName $.S $asset }}
			{{- if and (eq $name "/market/ticker") (gt (len $pairs) 10) }}
				{{- $name -}} :all
				{{- with $i := channelInterval $.S }}_{{ $i }}{{ end }}
				{{- $.BatchSize }} {{- len $pairs }}
			{{- else }}
				{{- range $b := batch $pairs 100 }}
					{{- $name -}} : {{- joinPairsWithInterval $b $.S }}
					{{- $.PairSeparator }}
				{{- end }}
				{{- $.BatchSize -}} 100
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- channelName $.S $.S.Asset }}
{{- end }}
`
