package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
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
)

var (
	// maxWSUpdateBuffer defines max websocket updates to apply when an
	// orderbook is initially fetched
	maxWSUpdateBuffer = 150
	// maxWSOrderbookJobs defines max websocket orderbook jobs in queue to fetch
	// an orderbook snapshot via REST
	maxWSOrderbookJobs = 2000
	// maxWSOrderbookWorkers defines a max amount of workers allowed to execute
	// jobs from the job channel
	maxWSOrderbookWorkers = 10
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
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	e.fetchedFuturesOrderbookMutex.Lock()
	e.fetchedFuturesOrderbook = map[string]bool{}
	e.fetchedFuturesOrderbookMutex.Unlock()
	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var instances *WSInstanceServers
	_, err := e.GetCredentials(ctx)
	if err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		instances, err = e.GetAuthenticatedInstanceServers(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			return err
		}
	}
	if instances == nil {
		instances, err = e.GetInstanceServers(ctx)
		if err != nil {
			return err
		}
	}
	if len(instances.InstanceServers) == 0 {
		return errors.New("no websocket instance server found")
	}
	e.Websocket.Conn.SetURL(instances.InstanceServers[0].Endpoint + "?token=" + instances.Token)
	err = e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       time.Millisecond * time.Duration(instances.InstanceServers[0].PingTimeout),
		Message:     []byte(`{"type":"ping"}`),
		MessageType: gws.TextMessage,
	})

	e.setupOrderbookManager(ctx)
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

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()
	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := e.wsHandleData(ctx, resp.Raw)
		if err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
	}
}

// wsHandleData processes a websocket incoming data.
func (e *Exchange) wsHandleData(ctx context.Context, respData []byte) error {
	resp := WsPushData{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	if resp.Type == "pong" || resp.Type == "welcome" {
		return nil
	}
	if resp.ID != "" {
		return e.Websocket.Match.RequireMatchWithData(resp.ID, respData)
	}
	topicInfo := strings.Split(resp.Topic, ":")
	switch topicInfo[0] {
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
		return e.processOrderbookWithDepth(respData, topicInfo[1], topicInfo[0])
	case marketOrderbookDepth5Channel, marketOrderbookDepth50Channel:
		return e.processOrderbook(resp.Data, topicInfo[1], topicInfo[0])
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
		if err := e.ensureFuturesOrderbookSnapshotLoaded(ctx, topicInfo[1]); err != nil {
			return err
		}
		return e.processFuturesOrderbookLevel2(ctx, resp.Data, topicInfo[1])
	case futuresOrderbookDepth5Channel,
		futuresOrderbookDepth50Channel:
		if err := e.ensureFuturesOrderbookSnapshotLoaded(ctx, topicInfo[1]); err != nil {
			return err
		}
		return e.processFuturesOrderbookSnapshot(resp.Data, topicInfo[1])
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

// processData used to deserialize and forward the data to DataHandler.
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

// ensureFuturesOrderbookSnapshotLoaded makes sure an initial futures orderbook snapshot is loaded
func (e *Exchange) ensureFuturesOrderbookSnapshotLoaded(ctx context.Context, symbol string) error {
	e.fetchedFuturesOrderbookMutex.Lock()
	defer e.fetchedFuturesOrderbookMutex.Unlock()
	if e.fetchedFuturesOrderbook[symbol] {
		return nil
	}
	e.fetchedFuturesOrderbook[symbol] = true
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	cp, err := enabledPairs.DeriveFrom(symbol)
	if err != nil {
		return err
	}
	orderbooks, err := e.UpdateOrderbook(ctx, cp, asset.Futures)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(orderbooks)
}

// processFuturesOrderbookSnapshot processes a futures account orderbook websocket update.
func (e *Exchange) processFuturesOrderbookSnapshot(respData []byte, instrument string) error {
	var resp WsOrderbookLevel5Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(instrument, asset.Futures, false)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:                   resp.Sequence,
		UpdateTime:                 resp.Timestamp.Time(),
		Asset:                      asset.Futures,
		Bids:                       resp.Bids.Levels(),
		Asks:                       resp.Asks.Levels(),
		Pair:                       pair,
		SkipOutOfOrderLastUpdateID: true,
	})
}

// processFuturesOrderbookLevel2 processes a V2 futures account orderbook data
func (e *Exchange) processFuturesOrderbookLevel2(ctx context.Context, respData []byte, instrument string) error {
	resp := WsFuturesOrderbookInfo{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	detail, err := e.GetFuturesPartOrderbook100(ctx, instrument)
	if err != nil {
		return err
	}
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	pair, err := enabledPairs.DeriveFrom(instrument)
	if err != nil {
		return err
	}
	base := orderbook.Update{
		UpdateTime: detail.Time,
		Pair:       pair,
		Asset:      asset.Futures,
		Asks:       detail.Asks,
		Bids:       detail.Bids,
	}
	return e.Websocket.Orderbook.Update(&base)
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
		Volume:       resp.FilledSize.Float64(),
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
	var pair currency.Pair
	pair, err = currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &websocket.KlineData{
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
			LastUpdated:  response.Timestamp.Time(),
			ExchangeName: e.Name,
			Pair:         pair,
			Ask:          response.BestAsk,
			Bid:          response.BestBid,
			AskSize:      response.BestAskSize,
			BidSize:      response.BestBidSize,
			Volume:       response.Size,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processCandlesticks processes a candlestick data for an instrument with a particular interval
func (e *Exchange) processCandlesticks(ctx context.Context, respData []byte, instrument, intervalString, topic string) error {
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
		if err := e.Websocket.DataHandler.Send(ctx, &websocket.KlineData{
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
		}); err != nil {
			return err
		}
	}
	return nil
}

// processOrderbookWithDepth processes order book data with a specified depth for a particular symbol.
func (e *Exchange) processOrderbookWithDepth(respData []byte, instrument, topic string) error {
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	result := struct {
		Result *WsOrderbook `json:"data"`
	}{}
	err = json.Unmarshal(respData, &result)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		var init bool
		init, err = e.updateLocalBuffer(result.Result, assets[x])
		if err != nil {
			if init {
				return nil
			}
			return fmt.Errorf("%v - UpdateLocalCache for asset type: %v error: %s", e.Name, assets[x], err)
		}
	}
	return nil
}

// updateLocalBuffer updates orderbook buffer and checks status if the book is Initial Sync being via the REST
// protocol.
func (e *Exchange) updateLocalBuffer(wsdp *WsOrderbook, assetType asset.Item) (bool, error) {
	enabledPairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return false, err
	}

	format, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(wsdp.Symbol, enabledPairs, format)
	if err != nil {
		return false, err
	}
	err = e.obm.StageWsUpdate(wsdp, currencyPair, assetType)
	if err != nil {
		init, err2 := e.obm.CheckIsInitialSync(currencyPair, assetType)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = e.applyBufferUpdate(currencyPair, assetType)
	if err != nil {
		e.invalidateAndCleanupOrderbook(currencyPair, assetType)
	}

	return false, err
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
			Volume:       response.Data.Vol,
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
func (e *Exchange) Subscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	return e.manageSubscriptions(ctx, subscriptions, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	return e.manageSubscriptions(ctx, subscriptions, "unsubscribe")
}

func (e *Exchange) manageSubscriptions(ctx context.Context, subs subscription.List, operation string) error {
	var errs error
	for _, s := range subs {
		req := WsSubscriptionInput{
			ID:             e.MessageID(),
			Type:           operation,
			Topic:          s.QualifiedChannel,
			PrivateChannel: s.Authenticated,
			Response:       true,
		}
		if respRaw, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req); err != nil {
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
					err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
				} else {
					err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s)
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

// orderbookManager defines a way of managing and maintaining synchronisation
// across connections and assets.
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update
	sync.Mutex

	jobs chan job
}

type update struct {
	buffer            chan *WsOrderbook
	fetchingBook      bool
	initialSync       bool
	needsFetchingBook bool
	lastUpdateID      int64
}

// job defines a synchronisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair      currency.Pair
	AssetType asset.Item
}

// setupOrderbookManager sets up the orderbook manager for websocket orderbook data handling.
func (e *Exchange) setupOrderbookManager(ctx context.Context) {
	e.obmMutex.Lock()
	defer e.obmMutex.Unlock()
	if e.obm == nil {
		e.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		e.obm.Mutex.Lock()
		for _, m1 := range e.obm.state {
			for _, m2 := range m1 {
				for _, idk := range m2 {
					idk.initialSync = true
					idk.needsFetchingBook = true
					idk.lastUpdateID = 0
				}
			}
		}
		e.obm.Mutex.Unlock()
	}
	for range maxWSOrderbookWorkers {
		// 10 workers for synchronising book
		e.SynchroniseWebsocketOrderbook(ctx)
	}
}

// processOrderbookUpdate processes the websocket orderbook update
func (e *Exchange) processOrderbookUpdate(cp currency.Pair, a asset.Item, ws *WsOrderbook) error {
	updateBid := make([]orderbook.Level, len(ws.Changes.Bids))
	for i := range ws.Changes.Bids {
		var sequence int64
		if len(ws.Changes.Bids[i]) > 2 {
			sequence = ws.Changes.Bids[i][2].Int64()
		}
		updateBid[i] = orderbook.Level{Price: ws.Changes.Bids[i][0].Float64(), Amount: ws.Changes.Bids[i][1].Float64(), ID: sequence}
	}
	updateAsk := make([]orderbook.Level, len(ws.Changes.Asks))
	for i := range ws.Changes.Asks {
		var sequence int64
		if len(ws.Changes.Asks[i]) > 2 {
			sequence = ws.Changes.Asks[i][2].Int64()
		}
		updateAsk[i] = orderbook.Level{Price: ws.Changes.Asks[i][0].Float64(), Amount: ws.Changes.Asks[i][1].Float64(), ID: sequence}
	}

	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:                       updateBid,
		Asks:                       updateAsk,
		Pair:                       cp,
		UpdateID:                   ws.SequenceEnd,
		UpdateTime:                 ws.TimeMS.Time(),
		Asset:                      a,
		SkipOutOfOrderLastUpdateID: true,
	})
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (e *Exchange) applyBufferUpdate(pair currency.Pair, assetType asset.Item) error {
	fetching, needsFetching, err := e.obm.HandleFetchingBook(pair, assetType)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}
	if needsFetching {
		if e.Verbose {
			log.Debugf(log.WebsocketMgr, "%s Orderbook: Fetching via REST\n", e.Name)
		}
		return e.obm.FetchBookViaREST(pair, assetType)
	}

	recent, err := e.Websocket.Orderbook.GetOrderbook(pair, assetType)
	if err != nil {
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			e.Name,
			err)
	}

	if recent != nil {
		err = e.obm.checkAndProcessOrderbookUpdate(e.processOrderbookUpdate, pair, assetType, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				e.Name,
				err)
			err = e.obm.setNeedsFetchingBook(pair, assetType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// setNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) setNeedsFetchingBook(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			assetType)
	}
	state.needsFetchingBook = true
	return nil
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// asset
func (e *Exchange) SynchroniseWebsocketOrderbook(ctx context.Context) {
	e.Websocket.Wg.Go(func() {
		for {
			select {
			case <-e.Websocket.ShutdownC:
				for {
					select {
					case <-e.obm.jobs:
					default:
						return
					}
				}
			case j := <-e.obm.jobs:
				if err := e.processJob(ctx, j.Pair, j.AssetType); err != nil {
					log.Errorf(log.WebsocketMgr, "%s processing websocket orderbook error: %v", e.Name, err)
				}
			}
		}
	})
}

// SeedLocalCache seeds depth data
func (e *Exchange) SeedLocalCache(ctx context.Context, p currency.Pair, assetType asset.Item) error {
	ob, err := e.GetPartOrderbook100(ctx, p.String())
	if err != nil {
		return err
	}
	if ob.Sequence <= 0 {
		return fmt.Errorf("%w p", errMissingOrderbookSequence)
	}
	return e.SeedLocalCacheWithBook(p, ob, assetType)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (e *Exchange) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *Orderbook, assetType asset.Item) error {
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Pair:              p,
		Asset:             assetType,
		Exchange:          e.Name,
		LastUpdated:       orderbookNew.Time,
		LastUpdateID:      orderbookNew.Sequence,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbookNew.Bids,
		Asks:              orderbookNew.Asks,
	})
}

// processJob fetches and processes orderbook updates
func (e *Exchange) processJob(ctx context.Context, p currency.Pair, assetType asset.Item) error {
	err := e.SeedLocalCache(ctx, p, assetType)
	if err != nil {
		err = e.obm.StopFetchingBook(p, assetType)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, assetType, err)
	}

	err = e.obm.StopFetchingBook(p, assetType)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = e.applyBufferUpdate(p, assetType)
	if err != nil {
		e.invalidateAndCleanupOrderbook(p, assetType)
		return err
	}
	return nil
}

// invalidateAndCleanupOrderbook invalidates orderbook and cleans local cache
func (e *Exchange) invalidateAndCleanupOrderbook(p currency.Pair, assetType asset.Item) {
	if err := e.Websocket.Orderbook.InvalidateOrderbook(p, assetType); err != nil {
		log.Errorf(log.WebsocketMgr, "%s invalidate websocket error: %v", e.Name, err)
	}
	if err := e.obm.Cleanup(p, assetType); err != nil {
		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v", e.Name, err)
	}
}

// StageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) StageWsUpdate(u *WsOrderbook, pair currency.Pair, a asset.Item) error {
	o.Lock()
	defer o.Unlock()
	m1, ok := o.state[pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*update)
		o.state[pair.Base] = m1
	}

	m2, ok := m1[pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*update)
		m1[pair.Quote] = m2
	}

	state, ok := m2[a]
	if !ok {
		state = &update{
			// 100ms update assuming we might have up to a 10 second delay.
			// There could be a potential 100 updates for the currency.
			buffer:            make(chan *WsOrderbook, maxWSUpdateBuffer),
			fetchingBook:      false,
			initialSync:       true,
			needsFetchingBook: true,
		}
		m2[a] = state
	}

	if state.lastUpdateID != 0 && u.SequenceStart > state.lastUpdateID+1 {
		// Apply the new Level 2 data flow to the local snapshot to ensure that sequenceStart(new)<=sequenceEnd+1(old) and sequenceEnd(new) > sequenceEnd(old)
		return fmt.Errorf("websocket orderbook synchronisation failure for pair %s and asset %s", pair, a)
	}
	state.lastUpdateID = u.SequenceEnd
	select {
	// Put update in the channel buffer to be processed
	case state.buffer <- u:
		return nil
	default:
		<-state.buffer    // pop one element
		state.buffer <- u // to shift buffer on fail
		return fmt.Errorf("channel blockage for %s, asset %s and connection",
			pair, a)
	}
}

// HandleFetchingBook checks if a full book is being fetched or needs to be
// fetched
func (o *orderbookManager) HandleFetchingBook(pair currency.Pair, assetType asset.Item) (fetching, needsFetching bool, err error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return false,
			false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				assetType)
	}

	if state.fetchingBook {
		return true, false, nil
	}

	if state.needsFetchingBook {
		state.needsFetchingBook = false
		state.fetchingBook = true
		return false, true, nil
	}
	return false, false, nil
}

// StopFetchingBook completes the book fetching.
func (o *orderbookManager) StopFetchingBook(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			assetType)
	}
	if !state.fetchingBook {
		return fmt.Errorf("fetching book already set to false for %s %s",
			pair,
			assetType)
	}
	state.fetchingBook = false
	return nil
}

// CompleteInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) CompleteInitialSync(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
			pair,
			assetType)
	}
	if !state.initialSync {
		return fmt.Errorf("initial sync already set to false for %s %s",
			pair,
			assetType)
	}
	state.initialSync = false
	return nil
}

// CheckIsInitialSync checks status if the book is Initial Sync being via the REST
// protocol.
func (o *orderbookManager) CheckIsInitialSync(pair currency.Pair, assetType asset.Item) (bool, error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return false,
			fmt.Errorf("checkIsInitialSync of orderbook cannot match currency pair %s asset type %s",
				pair,
				assetType)
	}
	return state.initialSync, nil
}

// FetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) FetchBookViaREST(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	defer o.Unlock()

	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			assetType)
	}

	state.initialSync = true
	state.fetchingBook = true

	select {
	case o.jobs <- job{pair, assetType}:
		return nil
	default:
		return fmt.Errorf("%s %s book synchronisation channel blocked up",
			pair,
			assetType)
	}
}

func (o *orderbookManager) checkAndProcessOrderbookUpdate(processor func(currency.Pair, asset.Item, *WsOrderbook) error, pair currency.Pair, assetType asset.Item, recent *orderbook.Book) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
			pair, assetType)
	}

	// This will continuously remove updates from the buffered channel and
	// apply them to the current orderbook.
buffer:
	for {
		select {
		case d := <-state.buffer:
			process, err := state.Validate(d, recent)
			if err != nil {
				return err
			}
			if process {
				err := processor(pair, assetType, d)
				if err != nil {
					return fmt.Errorf("%s %s processing update error: %w",
						pair, assetType, err)
				}
			}
		default:
			break buffer
		}
	}
	return nil
}

// Validate checks for correct update alignment
func (u *update) Validate(updt *WsOrderbook, recent *orderbook.Book) (bool, error) {
	if updt.SequenceEnd <= recent.LastUpdateID {
		// Drop any event where u is <= lastUpdateId in the snapshot.
		return false, nil
	}

	id := recent.LastUpdateID + 1
	if u.initialSync {
		// The first processed event should have U <= lastUpdateId+1 AND
		// u >= lastUpdateId+1.
		if updt.SequenceStart > id || updt.SequenceEnd < id {
			return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
				recent.Pair,
				recent.Asset)
		}
		u.initialSync = false
	}
	return true, nil
}

// Cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) Cleanup(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		o.Unlock()
		return fmt.Errorf("cleanup cannot match %s %s to hash table",
			pair,
			assetType)
	}

bufferEmpty:
	for {
		select {
		case <-state.buffer:
			// bleed and discard buffer
		default:
			break bufferEmpty
		}
	}
	o.Unlock()
	// disable rest orderbook synchronisation
	_ = o.StopFetchingBook(pair, assetType)
	_ = o.CompleteInitialSync(pair, assetType)
	_ = o.StopNeedsFetchingBook(pair, assetType)
	return nil
}

// StopNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) StopNeedsFetchingBook(pair currency.Pair, assetType asset.Item) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][assetType]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			assetType)
	}
	if !state.needsFetchingBook {
		return fmt.Errorf("needs fetching book already set to false for %s %s",
			pair,
			assetType)
	}
	state.needsFetchingBook = false
	return nil
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
