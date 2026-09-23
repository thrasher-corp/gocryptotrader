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
	return &response.Data, e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
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
		return e.processTicker(ctx, resp.Data, instruments, topicInfo[0])
	case marketSnapshotChannel:
		return e.processMarketSnapshot(ctx, resp.Data, topicInfo[0])
	case marketOrderbookChannel:
		return e.processSpotOrderbookWithDepth(ctx, respData, topicInfo[1])
	case marketOrderbookDepth1Channel, marketOrderbookDepth5Channel, marketOrderbookDepth50Channel:
		return e.processOrderbook(ctx, resp.Data, topicInfo[1], topicInfo[0])
	case marketCandlesChannel:
		symbolAndInterval := strings.Split(topicInfo[1], currency.UnderscoreDelimiter)
		if len(symbolAndInterval) != 2 {
			return common.ErrMalformedData
		}
		return e.processCandlesticks(ctx, resp.Data, symbolAndInterval[0], symbolAndInterval[1], topicInfo[0])
	case marketMatchChannel:
		return e.processTradeData(resp.Data, topicInfo[1], topicInfo[0])
	case indexPriceIndicatorChannel, markPriceIndicatorChannel:
		var response WsPriceIndicator
		return e.processData(ctx, resp.Data, &response)
	case privateSpotTradeOrders:
		return e.processOrderChangeEvent(ctx, resp.Data, topicInfo[0])
	case accountBalanceChannel:
		return e.processAccountBalanceChange(ctx, resp.Data)
	case marginPositionChannel:
		if resp.Subject == "debt.ratio" {
			var response WsDebtRatioChange
			return e.processData(ctx, resp.Data, &response)
		}
		var response WsPositionStatus
		return e.processData(ctx, resp.Data, &response)
	case marginLoanChannel:
		if resp.Subject == "order.done" {
			var response WsMarginTradeOrderDoneEvent
			return e.processData(ctx, resp.Data, &response)
		}
		return e.processMarginLendingTradeOrderEvent(ctx, resp.Data)
	case spotMarketAdvancedChannel:
		return e.processStopOrderEvent(ctx, resp.Data)
	case futuresTickerChannel:
		return e.processFuturesTickerV2(ctx, resp.Data)
	case futuresExecutionDataChannel:
		var response WsFuturesExecutionData
		return e.processData(ctx, resp.Data, &response)
	case futuresOrderbookChannel:
		return e.processFuturesOrderbookLevel2(ctx, resp.Data, topicInfo[1])
	case futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
		return e.processFuturesOrderbookSnapshot(ctx, resp.Data, topicInfo[1])
	case futuresContractMarketDataChannel:
		switch resp.Subject {
		case "mark.index.price":
			return e.processFuturesMarkPriceAndIndexPrice(ctx, resp.Data, topicInfo[1])
		case "funding.rate":
			return e.processFuturesFundingData(ctx, resp.Data, topicInfo[1])
		}
	case futuresSystemAnnouncementChannel:
		return e.processFuturesSystemAnnouncement(ctx, resp.Data, resp.Subject)
	case futuresTransactionStatisticsTimerEventChannel:
		return e.processFuturesTransactionStatistics(resp.Data, topicInfo[1])
	case futuresTradeOrderChannel:
		return e.processFuturesPrivateTradeOrders(ctx, resp.Data)
	case futuresStopOrdersLifecycleEventChannel:
		return e.processFuturesStopOrderLifecycleEvent(ctx, resp.Data)
	case futuresAccountBalanceEventChannel:
		switch resp.Subject {
		case "orderMargin.change":
			var response WsFuturesOrderMarginEvent
			return e.processData(ctx, resp.Data, &response)
		case "availableBalance.change":
			return e.processFuturesAccountBalanceEvent(ctx, resp.Data)
		case "withdrawHold.change":
			var response WsFuturesWithdrawalAmountAndTransferOutAmountEvent
			return e.processData(ctx, resp.Data, &response)
		}
	case futuresPositionChangeEventChannel:
		switch resp.Subject {
		case "position.change":
			if resp.ChannelType == "private" {
				var response WsFuturesPosition
				return e.processData(ctx, resp.Data, &response)
			}
			var response WsFuturesMarkPricePositionChanges
			return e.processData(ctx, resp.Data, &response)
		case "position.settlement":
			var response WsFuturesPositionFundingSettlement
			return e.processData(ctx, resp.Data, &response)
		}
	case futuresLimitCandles:
		instrumentInfos := strings.Split(topicInfo[1], "_")
		if len(instrumentInfos) != 2 {
			return errors.New("invalid instrument information")
		}
		return e.processFuturesKline(ctx, resp.Data, instrumentInfos[1])
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respData),
		})
	}
	return nil
}

// processData used to deserialise and forward the data to DataHandler.
func (e *Exchange) processData(ctx context.Context, respData []byte, resp any) error {
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
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
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// processFuturesStopOrderLifecycleEvent processes futures stop orders lifecycle events.
func (e *Exchange) processFuturesStopOrderLifecycleEvent(ctx context.Context, respData []byte) error {
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
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
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
	})
}

// processFuturesPrivateTradeOrders processes futures private trade orders updates.
func (e *Exchange) processFuturesPrivateTradeOrders(ctx context.Context, respData []byte) error {
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
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
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
	})
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
func (e *Exchange) processFuturesSystemAnnouncement(ctx context.Context, respData []byte, subject string) error {
	resp := WsFuturesFundingBegin{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Subject = subject
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processFuturesFundingData processes a futures account funding data.
func (e *Exchange) processFuturesFundingData(ctx context.Context, respData []byte, instrument string) error {
	resp := WsFundingRate{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processFuturesMarkPriceAndIndexPrice processes a futures account mark price and index price changes.
func (e *Exchange) processFuturesMarkPriceAndIndexPrice(ctx context.Context, respData []byte, instrument string) error {
	resp := WsFuturesMarkPriceAndIndexPrice{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processFuturesOrderbookSnapshot processes a futures account orderbook websocket update.
func (e *Exchange) processFuturesOrderbookSnapshot(ctx context.Context, respData []byte, instrument string) error {
	var resp WsFuturesOrderbookLevelResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(instrument, asset.Futures, false)
	if err != nil {
		return err
	}
	bids := mergeRoundedOrderbookLevels(resp.Bids.Levels())
	asks := mergeRoundedOrderbookLevels(resp.Asks.Levels())
	// Note: KuCoin snapshot timestamps are all the same and each update is 100ms apart.
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:     e.Name,
		LastUpdateID: resp.Sequence,
		LastUpdated:  resp.Timestamp.Time(),
		LastPushed:   resp.PushTimestamp.Time(),
		Asset:        asset.Futures,
		Bids:         bids,
		Asks:         asks,
		Pair:         pair,
	})
}

// processFuturesOrderbookLevel2 processes a V2 futures account orderbook data
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
func (e *Exchange) processFuturesTickerV2(ctx context.Context, respData []byte) error {
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
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		AssetType:    asset.Futures,
		Last:         resp.FilledPrice.Float64(),
		LastSize:     resp.FilledSize.Float64(),
		LastUpdated:  resp.FilledTime.Time(),
		ExchangeName: e.Name,
		Pair:         pair,
		Ask:          resp.BestAskPrice.Float64(),
		Bid:          resp.BestBidPrice.Float64(),
		AskSize:      resp.BestAskSize.Float64(),
		BidSize:      resp.BestBidSize.Float64(),
	})
}

// processFuturesKline represents a futures instrument kline data update.
func (e *Exchange) processFuturesKline(ctx context.Context, respData []byte, intervalStr string) error {
	resp := WsFuturesKline{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	interval, err := IntervalFromString(intervalStr)
	if err != nil {
		return err
	}
	var pair currency.Pair
	pair, err = currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &kline.Item{
		Asset:    asset.Futures,
		Exchange: e.Name,
		Pair:     pair,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   time.Unix(resp.Candles[0].Int64(), 0),
			Open:   resp.Candles[1].Float64(),
			Close:  resp.Candles[2].Float64(),
			High:   resp.Candles[3].Float64(),
			Low:    resp.Candles[4].Float64(),
			Volume: resp.Candles[6].Float64(),
		}},
	})
}

// processStopOrderEvent represents a stop order update event.
func (e *Exchange) processStopOrderEvent(ctx context.Context, respData []byte) error {
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
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
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
	})
}

// processMarginLendingTradeOrderEvent represents a margin lending trade order event.
func (e *Exchange) processMarginLendingTradeOrderEvent(ctx context.Context, respData []byte) error {
	resp := WsMarginTradeOrderEntersEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
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
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// processOrderChangeEvent processes order update events.
func (e *Exchange) processOrderChangeEvent(ctx context.Context, respData []byte, topic string) error {
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
		if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
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
		}); err != nil {
			return err
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
func (e *Exchange) processTicker(ctx context.Context, respData []byte, instrument, topic string) error {
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
		if err := e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			AssetType:    assets[x],
			Last:         response.Price,
			LastSize:     response.Size,
			LastUpdated:  response.Timestamp.Time(),
			ExchangeName: e.Name,
			Pair:         pair,
			Ask:          response.BestAsk,
			Bid:          response.BestBid,
			AskSize:      response.BestAskSize,
			BidSize:      response.BestBidSize,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processCandlesticks processes a candlestick data for an instrument with a particular interval
func (e *Exchange) processCandlesticks(ctx context.Context, respData []byte, instrument, intervalStr, topic string) error {
	interval, err := IntervalFromString(intervalStr)
	if err != nil {
		return err
	}
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
		if err := e.Websocket.DataHandler.Send(ctx, &kline.Item{
			Pair:     pair,
			Asset:    assets[x],
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:   resp.Candles.StartTime.Time(),
				Open:   resp.Candles.OpenPrice.Float64(),
				Close:  resp.Candles.ClosePrice.Float64(),
				High:   resp.Candles.HighPrice.Float64(),
				Low:    resp.Candles.LowPrice.Float64(),
				Volume: resp.Candles.TransactionVolume.Float64(),
			}},
		}); err != nil {
			return err
		}
	}
	return nil
}

// processSpotOrderbookWithDepth processes order book data with a specified depth for a particular symbol.
func (e *Exchange) processSpotOrderbookWithDepth(ctx context.Context, respData []byte, instrument string) error {
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	var resp struct {
		Result WsOrderbook `json:"data"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
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

	assets, err := e.CalculateAssets(marketOrderbookChannel, pair)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		// A subscribed pair outside the enabled lists has always been booked as spot.
		assets = []asset.Item{asset.Spot}
	}
	// Each asset gets its own level slices to prevent sharing mutable state.
	for _, a := range assets {
		if err := e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, resp.Result.SequenceStart, &orderbook.Update{
			UpdateID:   resp.Result.SequenceEnd,
			UpdateTime: resp.Result.TimeMS.Time(),
			LastPushed: resp.Result.TimeMS.Time(), // Realtime so this is pushed when a change occurs
			Asset:      a,
			Bids:       slices.Clone(bids),
			Asks:       slices.Clone(asks),
			Pair:       pair,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processOrderbook processes orderbook data for a specific symbol.
func (e *Exchange) processOrderbook(ctx context.Context, respData []byte, symbol, topic string) error {
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
	asks := mergeRoundedOrderbookLevels(resp.Asks.Levels())
	bids := mergeRoundedOrderbookLevels(resp.Bids.Levels())
	for x := range assets {
		err = e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:    e.Name,
			Asks:        asks,
			Bids:        bids,
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
func (e *Exchange) processMarketSnapshot(ctx context.Context, respData []byte, topic string) error {
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
		if err := e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			AssetType:    assets[x],
			Last:         response.Data.LastTradedPrice,
			Pair:         pair,
			Low:          response.Data.Low,
			High:         response.Data.High,
			QuoteVolume:  response.Data.VolValue,
			BaseVolume:   response.Data.Vol,
			Open:         response.Data.Open,
			Close:        response.Data.Close,
			LastUpdated:  response.Data.Datetime.Time(),
		}); err != nil {
			return err
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
	outbound := collapseSubscriptionList(subs)

	var errs error
	for assoc, s := range outbound {
		req := WsSubscriptionInput{
			ID:             e.MessageID(),
			Type:           operation,
			Topic:          s.QualifiedChannel,
			PrivateChannel: s.Authenticated,
			Response:       true,
		}
		respRaw, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
		if err != nil {
			return err
		}

		intermediary := struct {
			Type string `json:"type"`
			Code uint64 `json:"code"`
			Data any    `json:"data"`
		}{}
		if err := json.Unmarshal(respRaw, &intermediary); err != nil {
			return err
		}

		switch intermediary.Type {
		case "error":
			msg, ok := intermediary.Data.(string)
			if !ok {
				msg = "unknown error"
			}
			errs = common.AppendError(errs, fmt.Errorf("%s (%d)", msg, intermediary.Code))
		case "ack":
			if operation == "unsubscribe" {
				err = e.Websocket.RemoveSubscriptions(conn, *assoc...)
			} else {
				err = e.Websocket.AddSuccessfulSubscriptions(conn, *assoc...)
				if e.Verbose {
					log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s", e.Name, s.Channel)
				}
			}
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		default:
			errs = common.AppendError(errs, fmt.Errorf("%w: %s from %s", errInvalidMsgType, intermediary.Type, respRaw))
		}
	}
	return errs
}

// collapseSubscriptionList merges per-pair subscriptions into KuCoin-compatible
// channel batches, capped at 100 entries per request. The returned map preserves
// the original subscriptions in each batch so subscribe and unsubscribe handling
// can still update connection state for the individual inputs.
func collapseSubscriptionList(subs subscription.List) map[*subscription.List]*subscription.Subscription {
	m := make(map[string][]*subscription.Subscription)
	for _, s := range subs {
		parts := strings.Split(s.QualifiedChannel, ":")
		m[parts[0]] = append(m[parts[0]], s)
	}

	result := make(map[*subscription.List]*subscription.Subscription)
	for qChan, group := range m {
		for _, batch := range common.Batch(group, 100) {
			s := batch[0].Clone()
			s.Pairs = nil
			suffixes := make([]string, 0, len(batch))
			for _, sub := range batch {
				s.Pairs = append(s.Pairs, sub.Pairs...)
				parts := strings.SplitN(sub.QualifiedChannel, ":", 2)
				if len(parts) == 2 && parts[1] != "" {
					suffixes = append(suffixes, parts[1])
				}
			}
			s.QualifiedChannel = qChan
			if len(suffixes) != 0 {
				s.QualifiedChannel += ":" + strings.Join(suffixes, ",")
			}
			key := subscription.List(batch)
			result[&key] = s
		}
	}
	return result
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	if err != nil {
		return subs, err
	}
	// Depth channels reload the whole book on every push, which would clobber a realtime book for the same pair, so
	// the realtime upgrade leaves pairs an explicit depth channel already feeds to that channel. Symbols are matched as
	// the depth topic spells them, so a topic KuCoin will not serve, such as a config-format pair or a candle suffix,
	// keeps its realtime feed. Level 1 is left out: its best bid and ask never decode as a book.
	depthFed := make(map[bool]map[string]struct{}, 2)
	depthTopics := make(map[string]struct{})
	for _, s := range subs {
		switch s.Channel {
		case marketOrderbookDepth5Channel, marketOrderbookDepth50Channel, futuresOrderbookDepth5Channel, futuresOrderbookDepth50Channel:
			isFutures := s.Channel == futuresOrderbookDepth5Channel || s.Channel == futuresOrderbookDepth50Channel
			if depthFed[isFutures] == nil {
				depthFed[isFutures] = make(map[string]struct{})
			}
			channel, symbols, _ := strings.Cut(s.QualifiedChannel, ":")
			for symbol := range strings.SplitSeq(symbols, ",") {
				depthFed[isFutures][symbol] = struct{}{}
				depthTopics[channel+":"+symbol] = struct{}{}
			}
		}
	}
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		// KuCoin keeps one subscription per topic, so a generic orderbook on an explicit depth channel's topic would
		// silence it when the realtime upgrade unsubscribes the generic; the explicit channel carries that topic alone.
		return slices.DeleteFunc(subs, func(s *subscription.Subscription) bool {
			_, shared := depthTopics[s.QualifiedChannel]
			return shared && s.Channel == subscription.OrderbookChannel
		}), nil
	}
	// Resolve authenticated orderbooks after expansion so the realtime feed is reflected in
	// subscription reconciliation keys and does not retain the public depth feed interval.
	configured := subscription.NewStore()
	seen := subscription.NewStore()
	resolved := make(subscription.List, 0, len(subs))
	for _, s := range subs {
		if s.Channel != subscription.OrderbookChannel {
			resolved = append(resolved, s)
			continue
		}
		if err := configured.Add(s.Clone()); err != nil {
			return nil, err
		}
		s = s.Clone()
		channel := marketOrderbookChannel
		formatAsset := s.Asset
		if s.Asset == asset.Futures {
			channel = futuresOrderbookChannel
		} else if formatAsset == asset.All || formatAsset == asset.Empty {
			formatAsset = asset.Spot
		}
		format, err := e.GetPairFormat(formatAsset, true)
		if err != nil {
			return nil, err
		}
		if fed := depthFed[s.Asset == asset.Futures]; len(fed) != 0 && len(s.Pairs) != 0 {
			s.Pairs = slices.DeleteFunc(s.Pairs, func(pair currency.Pair) bool {
				_, ok := fed[pair.Format(format).String()]
				return ok
			})
			if len(s.Pairs) == 0 {
				continue
			}
		}
		s.Channel = channel
		s.QualifiedChannel = channel + ":" + s.Pairs.Format(format).Join()
		s.Interval = 0
		s.Levels = 0
		if err := seen.Add(s); err != nil {
			if errors.Is(err, subscription.ErrDuplicate) {
				continue
			}
			return nil, err
		}
		resolved = append(resolved, s)
	}
	return resolved, nil
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").
		Funcs(template.FuncMap{
			"channelName":           channelName,
			"formatOrderbookPairs":  e.formatOrderbookPairs,
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
func (e *Exchange) checkSubscriptions() error {
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
	before := len(e.Config.Features.Subscriptions)
	var replacements subscription.List
	e.Config.Features.Subscriptions = slices.DeleteFunc(e.Config.Features.Subscriptions, func(s *subscription.Subscription) bool {
		switch s.Channel {
		case marketOrderbookChannel, futuresOrderbookChannel:
			if s.Enabled {
				assets := asset.Items{s.Asset}
				if s.Asset == asset.All || s.Asset == asset.Empty {
					assets = e.GetAssetTypes(true)
				}
				for _, assetType := range assets {
					replacements = append(replacements, &subscription.Subscription{
						Enabled: true, Channel: subscription.OrderbookChannel, Asset: assetType,
						Pairs: slices.Clone(s.Pairs), Interval: kline.HundredMilliseconds,
					})
				}
			}
			return true
		case "/contractMarket/level2Depth50", // Replaced by subsctiption.Orderbook for asset.All
			"/contractMarket/tickerV2", // Replaced by subscription.Ticker for asset.All
			"/margin/fundingBook":      // Deprecated and removed
			return true
		case subscription.AllTradesChannel:
			return s.Asset == asset.Empty
		}
		return false
	})
	upgraded = upgraded || before != len(e.Config.Features.Subscriptions)
	for _, replacement := range replacements {
		var coveredPairs currency.Pairs
		covered := false
		for i, sub := range e.Config.Features.Subscriptions {
			if !sub.Enabled || sub.Channel != subscription.OrderbookChannel || (sub.Asset != asset.All && sub.Asset != replacement.Asset) {
				continue
			}
			// A generic with a candle interval keeps the suffix on its public topics, where KuCoin sends nothing. It only
			// covers the legacy feed under websocket authentication, which can change after this migration is saved.
			if _, err := IntervalToString(sub.Interval); err == nil {
				continue
			}
			if sub.Authenticated && !e.API.AuthenticatedWebsocketSupport {
				if len(sub.Pairs) == 0 && sub.Asset == replacement.Asset {
					sub = sub.Clone()
					sub.Authenticated = false
					e.Config.Features.Subscriptions[i] = sub
					covered = true
					break
				}
				continue
			}
			if len(sub.Pairs) == 0 {
				covered = true
				break
			}
			coveredPairs = coveredPairs.Add(sub.Pairs...)
		}
		if covered {
			continue
		}
		if len(replacement.Pairs) != 0 && len(coveredPairs) != 0 {
			format, err := e.GetPairFormat(replacement.Asset, true)
			if err != nil {
				return fmt.Errorf("cannot migrate %s orderbook pair coverage: %w", replacement.Asset, err)
			}
			coveredSymbols := make(map[string]struct{}, len(coveredPairs))
			for _, pair := range coveredPairs.Format(format) {
				coveredSymbols[pair.String()] = struct{}{}
			}
			replacement.Pairs = slices.DeleteFunc(replacement.Pairs, func(pair currency.Pair) bool {
				_, ok := coveredSymbols[pair.Format(format).String()]
				return ok
			})
			if len(replacement.Pairs) == 0 {
				continue
			}
		}
		e.Config.Features.Subscriptions = append(e.Config.Features.Subscriptions, replacement)
	}
	if upgraded {
		e.Features.Subscriptions = e.Config.Features.Subscriptions.Enabled()
	}
	return nil
}

func (e *Exchange) formatOrderbookPairs(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) (string, error) {
	// Other channels retain their existing request spelling; formatting assetless private channels would fail pair lookup.
	if s.Channel != subscription.OrderbookChannel {
		return "", nil
	}
	if pairs, ok := ap[asset.Futures]; ok && (s.Asset == asset.All || s.Asset == asset.Futures) {
		ap[asset.Futures] = e.filterAssetFeedPairs(s, asset.Futures, pairs)
	}
	for assetType, pairs := range ap {
		if assetType == asset.Empty {
			continue
		}
		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return "", err
		}
		ap[assetType] = pairs.Format(format)
	}
	return "", nil
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

// mergeMarginPairs assigns each pair to one subscription on KuCoin's shared spot and margin feeds.
func (e *Exchange) mergeMarginPairs(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) string {
	if strings.HasPrefix(s.Channel, "/margin") {
		return ""
	}
	switch s.Asset {
	case asset.All:
		spotPairs, spotEnabled := ap[asset.Spot]
		marginPairs, marginEnabled := ap[asset.Margin]
		if spotEnabled {
			if marginEnabled {
				spotPairs = spotPairs.Add(marginPairs...)
				ap[asset.Margin] = currency.Pairs{}
			}
			ap[asset.Spot] = e.filterSharedFeedPairs(s, spotPairs)
		} else if marginEnabled {
			ap[asset.Margin] = e.filterSharedFeedPairs(s, marginPairs)
		}
	case asset.Spot:
		pairs := ap[asset.Spot]
		if len(s.Pairs) == 0 {
			for _, other := range e.Features.Subscriptions {
				if other.Asset == asset.Margin && e.sharedFeedCandidate(other, s) {
					pairs = pairs.Add(e.sharedFeedPairs(other)...)
				}
			}
		}
		ap[asset.Spot] = e.filterSharedFeedPairs(s, pairs)
	case asset.Margin:
		ap[asset.Margin] = e.filterSharedFeedPairs(s, ap[asset.Margin])
	}
	return ""
}

func (e *Exchange) filterAssetFeedPairs(s *subscription.Subscription, assetType asset.Item, pairs currency.Pairs) currency.Pairs {
	score := e.sharedFeedPriority(s)
	claimed := make(map[[2]*currency.Item]struct{})
	for _, other := range e.Features.Subscriptions {
		if other == s || other.QualifiedChannel != "" || other.Authenticated && !e.Websocket.CanUseAuthenticatedEndpoints() ||
			(other.Asset != asset.All && other.Asset != assetType) || !sameSharedFeed(other, s) {
			continue
		}
		if otherScore := e.sharedFeedPriority(other); otherScore > score || otherScore == score && other.Asset != s.Asset && other.Asset == assetType {
			for _, pair := range e.subscriptionPairsForAsset(other, assetType) {
				claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}] = struct{}{}
			}
		}
	}
	return slices.DeleteFunc(common.SortStrings(slices.Clone(pairs)), func(pair currency.Pair) bool {
		_, ok := claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}]
		return ok
	})
}

func (e *Exchange) subscriptionPairsForAsset(s *subscription.Subscription, assetType asset.Item) currency.Pairs {
	if len(s.Pairs) != 0 {
		return s.Pairs
	}
	pairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return nil
	}
	return pairs
}

func (e *Exchange) filterSharedFeedPairs(s *subscription.Subscription, pairs currency.Pairs) currency.Pairs {
	score := e.sharedFeedPriority(s)
	claimed := make(map[[2]*currency.Item]struct{})
	for _, other := range e.Features.Subscriptions {
		if other == s || !e.sharedFeedCandidate(other, s) {
			continue
		}
		if otherScore := e.sharedFeedPriority(other); otherScore > score || otherScore == score && other.Asset != s.Asset && sharedAssetPriority(other.Asset) > sharedAssetPriority(s.Asset) {
			for _, pair := range e.sharedFeedPairs(other) {
				claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}] = struct{}{}
			}
		}
	}
	return slices.DeleteFunc(common.SortStrings(slices.Clone(pairs)), func(pair currency.Pair) bool {
		_, ok := claimed[[2]*currency.Item{pair.Base.Item, pair.Quote.Item}]
		return ok
	})
}

func (e *Exchange) sharedFeedCandidate(candidate, reference *subscription.Subscription) bool {
	if candidate.QualifiedChannel != "" || candidate.Authenticated && !e.Websocket.CanUseAuthenticatedEndpoints() ||
		(candidate.Asset != asset.All && candidate.Asset != asset.Spot && candidate.Asset != asset.Margin) || !sameSharedFeed(candidate, reference) {
		return false
	}
	if candidate.Asset == asset.All {
		return e.CurrencyPairs.IsAssetEnabled(asset.Spot) == nil || e.CurrencyPairs.IsAssetEnabled(asset.Margin) == nil
	}
	return e.CurrencyPairs.IsAssetEnabled(candidate.Asset) == nil
}

func (e *Exchange) sharedFeedPairs(s *subscription.Subscription) currency.Pairs {
	if len(s.Pairs) != 0 {
		return s.Pairs
	}
	var pairs currency.Pairs
	for _, assetType := range []asset.Item{asset.Spot, asset.Margin} {
		if s.Asset != asset.All && s.Asset != assetType {
			continue
		}
		enabled, err := e.GetEnabledPairs(assetType)
		if err == nil {
			pairs = pairs.Add(enabled...)
		}
	}
	if s.Asset == asset.Spot {
		for _, other := range e.Features.Subscriptions {
			if other.Asset == asset.Margin && e.sharedFeedCandidate(other, s) {
				pairs = pairs.Add(e.sharedFeedPairs(other)...)
			}
		}
	}
	return pairs
}

func (e *Exchange) sharedFeedPriority(s *subscription.Subscription) int {
	priority := 0
	if len(s.Pairs) != 0 {
		priority += 2
	}
	if s.Asset != asset.All {
		priority++
	}
	// A public orderbook topic keeps any candle suffix, where KuCoin sends nothing, so that entry ranks below every entry
	// with a served topic and yields the pairs they share; the realtime rewrite drops the suffix, so auth leaves ranks alone.
	if s.Channel == subscription.OrderbookChannel && !e.Websocket.CanUseAuthenticatedEndpoints() {
		if _, err := IntervalToString(s.Interval); err == nil {
			priority -= 4
		}
	}
	return priority
}

func sharedAssetPriority(a asset.Item) int {
	switch a {
	case asset.Spot:
		return 2
	case asset.Margin:
		return 1
	default:
		return 0
	}
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

func sameSharedFeed(a, b *subscription.Subscription) bool {
	if a.Channel != b.Channel {
		return false
	}
	if a.Channel == subscription.OrderbookChannel {
		return true
	}
	if a.Levels != b.Levels {
		return false
	}
	aInterval, _ := IntervalToString(a.Interval)
	bInterval, _ := IntervalToString(b.Interval)
	return aInterval == bInterval
}

// assetCurrencies returns the currencies from all pairs in an asset
// Updates the AssetPairs map parameter to contain only those currencies as Base items for expandTemplates to see
func assetCurrencies(s *subscription.Subscription, ap map[asset.Item]currency.Pairs) currency.Currencies {
	cs := common.SortStrings(ap[s.Asset].GetCurrencies())
	p := make(currency.Pairs, 0, len(cs))
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
{{- formatOrderbookPairs $.S $.AssetPairs }}
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
				{{- range $b := batch $pairs 1 }}
					{{- $name -}} : {{- joinPairsWithInterval $b $.S }}
					{{- $.PairSeparator }}
				{{- end }}
				{{- $.BatchSize -}} 1
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- channelName $.S $.S.Asset }}
{{- end }}
`
