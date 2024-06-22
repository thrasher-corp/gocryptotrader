package kucoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var fetchedFuturesSnapshotOrderbook map[string]bool

const (
	publicBullets  = "/v1/bullet-public"
	privateBullets = "/v1/bullet-private"

	// spot channels
	marketAllTickersChannel         = "/market/ticker:all"
	marketTickerChannel             = "/market/ticker:%s"            // /market/ticker:{symbol},{symbol}...
	marketSymbolSnapshotChannel     = "/market/snapshot:%s"          // /market/snapshot:{symbol}
	marketSnapshotChannel           = "/market/snapshot:%v"          // /market/snapshot:{market} <--- market represents a currency
	marketOrderbookLevel2Channels   = "/market/level2:%s"            // /market/level2:{pair},{pair}...
	marketOrderbookLevel2to5Channel = "/spotMarket/level2Depth5:%s"  // /spotMarket/level2Depth5:{symbol},{symbol}...
	marketOrderbokLevel2To50Channel = "/spotMarket/level2Depth50:%s" // /spotMarket/level2Depth50:{symbol},{symbol}...
	marketCandlesChannel            = "/market/candles:%s_%s"        // /market/candles:{symbol}_{interval}
	marketMatchChannel              = "/market/match:%s"             // /market/match:{symbol},{symbol}...
	indexPriceIndicatorChannel      = "/indicator/index:%s"          // /indicator/index:{symbol0},{symbol1}..
	markPriceIndicatorChannel       = "/indicator/markPrice:%s"      // /indicator/markPrice:{symbol0},{symbol1}...
	marginFundingbookChangeChannel  = "/margin/fundingBook:%s"       // /margin/fundingBook:{currency0},{currency1}...

	// Private channels
	privateSpotTradeOrders    = "/spotMarket/tradeOrders"
	accountBalanceChannel     = "/account/balance"
	marginPositionChannel     = "/margin/position"
	marginLoanChannel         = "/margin/loan:%s" // /margin/loan:{currency}
	spotMarketAdvancedChannel = "/spotMarket/advancedOrders"

	// futures channels
	futuresTickerV2Channel                       = "/contractMarket/tickerV2:%s"      // /contractMarket/tickerV2:{symbol}
	futuresTickerChannel                         = "/contractMarket/ticker:%s"        // /contractMarket/ticker:{symbol}
	futuresOrderbookLevel2Channel                = "/contractMarket/level2:%s"        // /contractMarket/level2:{symbol}
	futuresExecutionDataChannel                  = "/contractMarket/execution:%s"     // /contractMarket/execution:{symbol}
	futuresOrderbookLevel2Depth5Channel          = "/contractMarket/level2Depth5:%s"  // /contractMarket/level2Depth5:{symbol}
	futuresOrderbookLevel2Depth50Channel         = "/contractMarket/level2Depth50:%s" // /contractMarket/level2Depth50:{symbol}
	futuresContractMarketDataChannel             = "/contract/instrument:%s"          // /contract/instrument:{symbol}
	futuresSystemAnnouncementChannel             = "/contract/announcement"
	futuresTrasactionStatisticsTimerEventChannel = "/contractMarket/snapshot:%s" // /contractMarket/snapshot:{symbol}

	// futures private channels
	futuresTradeOrdersBySymbolChannel      = "/contractMarket/tradeOrders:%s" // /contractMarket/tradeOrders:{symbol}
	futuresTradeOrderChannel               = "/contractMarket/tradeOrders"
	futuresStopOrdersLifecycleEventChannel = "/contractMarket/advancedOrders"
	futuresAccountBalanceEventChannel      = "/contractAccount/wallet"
	futuresPositionChangeEventChannel      = "/contract/position:%s" // /contract/position:{symbol}
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    marketAllTickersChannel,         // This allows more subscriptions on the orderbook channel for this specific connection.
	subscription.OrderbookChannel: marketOrderbookLevel2to5Channel, // This does not require a REST request to get the orderbook.
	subscription.CandlesChannel:   marketCandlesChannel,
	subscription.AllTradesChannel: marketMatchChannel,
	// No equivalents for: AllOrders, MyTrades, MyOrders
}

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

// WsConnect creates a new websocket connection.
func (ku *Kucoin) WsConnect() error {
	if !ku.Websocket.IsEnabled() || !ku.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	fetchedFuturesSnapshotOrderbook = map[string]bool{}
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = ku.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var instances *WSInstanceServers
	_, err := ku.GetCredentials(context.Background())
	if err != nil {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if ku.Websocket.CanUseAuthenticatedEndpoints() {
		instances, err = ku.GetAuthenticatedInstanceServers(context.Background())
		if err != nil {
			ku.Websocket.DataHandler <- err
			ku.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	if instances == nil {
		instances, err = ku.GetInstanceServers(context.Background())
		if err != nil {
			return err
		}
	}
	if len(instances.InstanceServers) == 0 {
		return errors.New("no websocket instance server found")
	}
	ku.Websocket.Conn.SetURL(instances.InstanceServers[0].Endpoint + "?token=" + instances.Token)
	err = ku.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", ku.Name, err)
	}
	ku.Websocket.Wg.Add(1)
	go ku.wsReadData()
	ku.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Delay:       time.Millisecond * time.Duration(instances.InstanceServers[0].PingTimeout),
		Message:     []byte(`{"type":"ping"}`),
		MessageType: websocket.TextMessage,
	})

	ku.setupOrderbookManager()
	return nil
}

// GetInstanceServers retrieves the server list and temporary public token
func (ku *Kucoin) GetInstanceServers(ctx context.Context) (*WSInstanceServers, error) {
	response := struct {
		Data WSInstanceServers `json:"data"`
		Error
	}{}
	return &(response.Data), ku.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		endpointPath, err := ku.API.Endpoints.GetURL(exchange.RestSpot)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpointPath + publicBullets,
			Result:        &response,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	}, request.UnauthenticatedRequest)
}

// GetAuthenticatedInstanceServers retrieves server instances for authenticated users.
func (ku *Kucoin) GetAuthenticatedInstanceServers(ctx context.Context) (*WSInstanceServers, error) {
	response := struct {
		Data *WSInstanceServers `json:"data"`
		Error
	}{}
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, privateBullets, nil, &response)
	if err != nil && strings.Contains(err.Error(), "400003") {
		return response.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, privateBullets, nil, &response)
	}
	return response.Data, err
}

// wsReadData receives and passes on websocket messages for processing
func (ku *Kucoin) wsReadData() {
	defer ku.Websocket.Wg.Done()
	for {
		resp := ku.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := ku.wsHandleData(resp.Raw)
		if err != nil {
			ku.Websocket.DataHandler <- err
		}
	}
}

func (ku *Kucoin) wsHandleData(respData []byte) error {
	resp := WsPushData{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	if resp.Type == "pong" || resp.Type == "welcome" {
		return nil
	}
	if resp.ID != "" {
		if !ku.Websocket.Match.IncomingWithData("msgID:"+resp.ID, respData) {
			return fmt.Errorf("message listener not found: %s", resp.ID)
		}
		return nil
	}
	topicInfo := strings.Split(resp.Topic, ":")
	switch {
	case strings.HasPrefix(marketAllTickersChannel, topicInfo[0]),
		strings.HasPrefix(marketTickerChannel, topicInfo[0]):
		var instruments string
		if topicInfo[1] == "all" {
			instruments = resp.Subject
		} else {
			instruments = topicInfo[1]
		}
		return ku.processTicker(resp.Data, instruments, topicInfo[0])
	case strings.HasPrefix(marketSymbolSnapshotChannel, topicInfo[0]):
		return ku.processMarketSnapshot(resp.Data, topicInfo[0])
	case strings.HasPrefix(marketOrderbookLevel2Channels, topicInfo[0]):
		return ku.processOrderbookWithDepth(respData, topicInfo[1], topicInfo[0])
	case strings.HasPrefix(marketOrderbookLevel2to5Channel, topicInfo[0]),
		strings.HasPrefix(marketOrderbokLevel2To50Channel, topicInfo[0]):
		return ku.processOrderbook(resp.Data, topicInfo[1], topicInfo[0])
	case strings.HasPrefix(marketCandlesChannel, topicInfo[0]):
		symbolAndInterval := strings.Split(topicInfo[1], currency.UnderscoreDelimiter)
		if len(symbolAndInterval) != 2 {
			return errMalformedData
		}
		return ku.processCandlesticks(resp.Data, symbolAndInterval[0], symbolAndInterval[1], topicInfo[0])
	case strings.HasPrefix(marketMatchChannel, topicInfo[0]):
		return ku.processTradeData(resp.Data, topicInfo[1], topicInfo[0])
	case strings.HasPrefix(indexPriceIndicatorChannel, topicInfo[0]):
		var response WsPriceIndicator
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(markPriceIndicatorChannel, topicInfo[0]):
		var response WsPriceIndicator
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(marginFundingbookChangeChannel, topicInfo[0]):
		var response WsMarginFundingBook
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(privateSpotTradeOrders, topicInfo[0]):
		return ku.processOrderChangeEvent(resp.Data, topicInfo[0])
	case strings.HasPrefix(accountBalanceChannel, topicInfo[0]):
		return ku.processAccountBalanceChange(resp.Data)
	case strings.HasPrefix(marginPositionChannel, topicInfo[0]):
		if resp.Subject == "debt.ratio" {
			var response WsDebtRatioChange
			return ku.processData(resp.Data, &response)
		}
		var response WsPositionStatus
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(marginLoanChannel, topicInfo[0]) && resp.Subject == "order.done":
		var response WsMarginTradeOrderDoneEvent
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(marginLoanChannel, topicInfo[0]):
		return ku.processMarginLendingTradeOrderEvent(resp.Data)
	case strings.HasPrefix(spotMarketAdvancedChannel, topicInfo[0]):
		return ku.processStopOrderEvent(resp.Data)
	case strings.HasPrefix(futuresTickerV2Channel, topicInfo[0]),
		strings.HasPrefix(futuresTickerChannel, topicInfo[0]):
		return ku.processFuturesTickerV2(resp.Data)
	case strings.HasPrefix(futuresOrderbookLevel2Channel, topicInfo[0]):
		if !fetchedFuturesSnapshotOrderbook[topicInfo[1]] {
			fetchedFuturesSnapshotOrderbook[topicInfo[1]] = true
			var enabledPairs currency.Pairs
			enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
			if err != nil {
				return err
			}
			var cp currency.Pair
			cp, err = enabledPairs.DeriveFrom(topicInfo[1])
			if err != nil {
				return err
			}
			var orderbooks *orderbook.Base
			orderbooks, err = ku.FetchOrderbook(context.Background(), cp, asset.Futures)
			if err != nil {
				return err
			}
			err = ku.Websocket.Orderbook.LoadSnapshot(orderbooks)
			if err != nil {
				return err
			}
		}
		return ku.processFuturesOrderbookLevel2(resp.Data, topicInfo[1])
	case strings.HasPrefix(futuresExecutionDataChannel, topicInfo[0]):
		var response WsFuturesExecutionData
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(futuresOrderbookLevel2Depth5Channel, topicInfo[0]),
		strings.HasPrefix(futuresOrderbookLevel2Depth50Channel, topicInfo[0]):
		if !fetchedFuturesSnapshotOrderbook[topicInfo[1]] {
			fetchedFuturesSnapshotOrderbook[topicInfo[1]] = true
			var enabledPairs currency.Pairs
			enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
			if err != nil {
				return err
			}
			cp, err := enabledPairs.DeriveFrom(topicInfo[1])
			if err != nil {
				return err
			}
			orderbooks, err := ku.FetchOrderbook(context.Background(), cp, asset.Futures)
			if err != nil {
				return err
			}
			err = ku.Websocket.Orderbook.LoadSnapshot(orderbooks)
			if err != nil {
				return err
			}
		}
		return ku.processFuturesOrderbookLevel5(resp.Data, topicInfo[1])
	case strings.HasPrefix(futuresContractMarketDataChannel, topicInfo[0]):
		if resp.Subject == "mark.index.price" {
			return ku.processFuturesMarkPriceAndIndexPrice(resp.Data, topicInfo[1])
		} else if resp.Subject == "funding.rate" {
			return ku.processFuturesFundingData(resp.Data, topicInfo[1])
		}
	case strings.HasPrefix(futuresSystemAnnouncementChannel, topicInfo[0]):
		return ku.processFuturesSystemAnnouncement(resp.Data, resp.Subject)
	case strings.HasPrefix(futuresTrasactionStatisticsTimerEventChannel, topicInfo[0]):
		return ku.processFuturesTransactionStatistics(resp.Data, topicInfo[1])
	case strings.HasPrefix(futuresTradeOrdersBySymbolChannel, topicInfo[0]),
		strings.HasPrefix(futuresTradeOrderChannel, topicInfo[0]):
		return ku.processFuturesPrivateTradeOrders(resp.Data)
	case strings.HasPrefix(futuresStopOrdersLifecycleEventChannel, topicInfo[0]):
		return ku.processFuturesStopOrderLifecycleEvent(resp.Data)
	case strings.HasPrefix(futuresAccountBalanceEventChannel, topicInfo[0]):
		switch resp.Subject {
		case "orderMargin.change":
			var response WsFuturesOrderMarginEvent
			return ku.processData(resp.Data, &response)
		case "availableBalance.change":
			return ku.processFuturesAccountBalanceEvent(resp.Data)
		case "withdrawHold.change":
			var response WsFuturesWithdrawalAmountAndTransferOutAmountEvent
			return ku.processData(resp.Data, &response)
		}
	case strings.HasPrefix(futuresPositionChangeEventChannel, topicInfo[0]):
		if resp.Subject == "position.change" {
			if resp.ChannelType == "private" {
				var response WsFuturesPosition
				return ku.processData(resp.Data, &response)
			}
			var response WsFuturesMarkPricePositionChanges
			return ku.processData(resp.Data, &response)
		} else if resp.Subject == "position.settlement" {
			var response WsFuturesPositionFundingSettlement
			return ku.processData(resp.Data, &response)
		}
	default:
		ku.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: ku.Name + stream.UnhandledMessage + string(respData),
		}
		return errors.New("push data not handled")
	}
	return nil
}

func (ku *Kucoin) processData(respData []byte, resp interface{}) error {
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
}

func (ku *Kucoin) processFuturesAccountBalanceEvent(respData []byte) error {
	resp := WsFuturesAvailableBalance{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- account.Change{
		Exchange: ku.Name,
		Currency: currency.NewCode(resp.Currency),
		Asset:    asset.Futures,
		Amount:   resp.AvailableBalance,
	}
	return nil
}

func (ku *Kucoin) processFuturesStopOrderLifecycleEvent(respData []byte) error {
	resp := WsStopOrderLifecycleEvent{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	var enabledPairs currency.Pairs
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
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
	ku.Websocket.DataHandler <- &order.Detail{
		Price:        resp.OrderPrice,
		TriggerPrice: resp.StopPrice,
		Amount:       resp.Size,
		Exchange:     ku.Name,
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

func (ku *Kucoin) processFuturesPrivateTradeOrders(respData []byte) error {
	resp := WsFuturesTradeOrder{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	oStatus, err := ku.stringToOrderStatus(resp.Status)
	if err != nil {
		return err
	}
	var enabledPairs currency.Pairs
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
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
	ku.Websocket.DataHandler <- &order.Detail{
		Type:            oType,
		Status:          oStatus,
		Pair:            pair,
		Side:            side,
		Amount:          resp.OrderSize,
		Price:           resp.OrderPrice,
		Exchange:        ku.Name,
		ExecutedAmount:  resp.FilledSize,
		RemainingAmount: resp.RemainSize,
		ClientOrderID:   resp.ClientOid,
		OrderID:         resp.TradeID,
		AssetType:       asset.Futures,
		LastUpdated:     resp.OrderTime.Time(),
	}
	return nil
}

func (ku *Kucoin) processFuturesTransactionStatistics(respData []byte, instrument string) error {
	resp := WsFuturesTransactionStatisticsTimeEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	return nil
}

func (ku *Kucoin) processFuturesSystemAnnouncement(respData []byte, subject string) error {
	resp := WsFuturesFundingBegin{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Subject = subject
	ku.Websocket.DataHandler <- &resp
	return nil
}

func (ku *Kucoin) processFuturesFundingData(respData []byte, instrument string) error {
	resp := WsFundingRate{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	ku.Websocket.DataHandler <- &resp
	return nil
}

func (ku *Kucoin) processFuturesMarkPriceAndIndexPrice(respData []byte, instrument string) error {
	resp := WsFuturesMarkPriceAndIndexPrice{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	resp.Symbol = instrument
	ku.Websocket.DataHandler <- &resp
	return nil
}

func (ku *Kucoin) processFuturesOrderbookLevel5(respData []byte, instrument string) error {
	response := WsOrderbookLevel5Response{}
	if err := json.Unmarshal(respData, &response); err != nil {
		return err
	}
	resp := response.ExtractOrderbookItems()
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	cp, err := enabledPairs.DeriveFrom(instrument)
	if err != nil {
		return err
	}
	return ku.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateID:   resp.Sequence,
		UpdateTime: resp.Timestamp.Time(),
		Asset:      asset.Futures,
		Bids:       resp.Bids,
		Asks:       resp.Asks,
		Pair:       cp,
	})
}

func (ku *Kucoin) processFuturesOrderbookLevel2(respData []byte, instrument string) error {
	resp := WsFuturesOrderbokInfo{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	detail, err := ku.GetFuturesPartOrderbook100(context.Background(), instrument)
	if err != nil {
		return err
	}
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
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
	return ku.Websocket.Orderbook.Update(&base)
}

func (ku *Kucoin) processFuturesTickerV2(respData []byte) error {
	resp := WsFuturesTicker{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	pair, err := enabledPairs.DeriveFrom(resp.Symbol)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- &ticker.Price{
		AssetType:    asset.Futures,
		Last:         resp.FilledPrice,
		Volume:       resp.FilledSize,
		LastUpdated:  resp.FilledTime.Time(),
		ExchangeName: ku.Name,
		Pair:         pair,
		Ask:          resp.BestAskPrice.Float64(),
		Bid:          resp.BestBidPrice.Float64(),
		AskSize:      resp.BestAskSize,
		BidSize:      resp.BestBidSize,
	}
	return nil
}

func (ku *Kucoin) processStopOrderEvent(respData []byte) error {
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
	ku.Websocket.DataHandler <- &order.Detail{
		Price:        resp.OrderPrice,
		TriggerPrice: resp.StopPrice,
		Amount:       resp.Size,
		Exchange:     ku.Name,
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

func (ku *Kucoin) processMarginLendingTradeOrderEvent(respData []byte) error {
	resp := WsMarginTradeOrderEntersEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
}

func (ku *Kucoin) processAccountBalanceChange(respData []byte) error {
	response := WsAccountBalance{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- account.Change{
		Exchange: ku.Name,
		Currency: currency.NewCode(response.Currency),
		Asset:    asset.Futures,
		Amount:   response.Available,
	}
	return nil
}

func (ku *Kucoin) processOrderChangeEvent(respData []byte, topic string) error {
	response := WsTradeOrder{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(response.OrderType)
	if err != nil {
		return err
	}
	oStatus, err := ku.stringToOrderStatus(response.Status)
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
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		ku.Websocket.DataHandler <- &order.Detail{
			Price:           response.Price,
			Amount:          response.Size,
			ExecutedAmount:  response.FilledSize,
			RemainingAmount: response.RemainSize,
			Exchange:        ku.Name,
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

func (ku *Kucoin) processTradeData(respData []byte, instrument, topic string) error {
	response := WsTrade{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	saveTradeData := ku.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!ku.IsTradeFeedEnabled() {
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
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		err = ku.Websocket.Trade.Update(saveTradeData, trade.Data{
			CurrencyPair: pair,
			Timestamp:    response.Time.Time(),
			Price:        response.Price,
			Amount:       response.Size,
			Side:         side,
			Exchange:     ku.Name,
			TID:          response.TradeID,
			AssetType:    assets[x],
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ku *Kucoin) processTicker(respData []byte, instrument, topic string) error {
	response := WsTicker{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !ku.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		ku.Websocket.DataHandler <- &ticker.Price{
			AssetType:    assets[x],
			Last:         response.Price,
			LastUpdated:  response.Timestamp.Time(),
			ExchangeName: ku.Name,
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

func (ku *Kucoin) processCandlesticks(respData []byte, instrument, intervalString, topic string) error {
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	response := WsCandlestickData{}
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	resp, err := response.getCandlestickData()
	if err != nil {
		return err
	}
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !ku.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		ku.Websocket.DataHandler <- &stream.KlineData{
			Timestamp:  response.Time.Time(),
			Pair:       pair,
			AssetType:  assets[x],
			Exchange:   ku.Name,
			StartTime:  resp.Candles.StartTime,
			Interval:   intervalString,
			OpenPrice:  resp.Candles.OpenPrice,
			ClosePrice: resp.Candles.ClosePrice,
			HighPrice:  resp.Candles.HighPrice,
			LowPrice:   resp.Candles.LowPrice,
			Volume:     resp.Candles.TransactionVolume,
		}
	}
	return nil
}

func (ku *Kucoin) processOrderbookWithDepth(respData []byte, instrument, topic string) error {
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
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		var init bool
		init, err = ku.UpdateLocalBuffer(result.Result, assets[x])
		if err != nil {
			if init {
				return nil
			}
			return fmt.Errorf("%v - UpdateLocalCache for asset type: %v error: %s", ku.Name, assets[x], err)
		}
	}
	return nil
}

// UpdateLocalBuffer updates orderbook buffer and checks status if the book is Initial Sync being via the REST
// protocol.
func (ku *Kucoin) UpdateLocalBuffer(wsdp *WsOrderbook, assetType asset.Item) (bool, error) {
	enabledPairs, err := ku.GetEnabledPairs(assetType)
	if err != nil {
		return false, err
	}

	format, err := ku.GetPairFormat(assetType, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(wsdp.Symbol,
		enabledPairs,
		format)
	if err != nil {
		return false, err
	}
	err = ku.obm.stageWsUpdate(wsdp, currencyPair, assetType)
	if err != nil {
		init, err2 := ku.obm.checkIsInitialSync(currencyPair, assetType)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = ku.applyBufferUpdate(currencyPair, assetType)
	if err != nil {
		ku.flushAndCleanup(currencyPair, assetType)
	}

	return false, err
}

func (ku *Kucoin) processOrderbook(respData []byte, symbol, topic string) error {
	var response Level2Depth5Or20
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}

	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}

	asks := make([]orderbook.Tranche, len(response.Asks))
	for x := range response.Asks {
		asks[x].Price = response.Asks[x][0].Float64()
		asks[x].Amount = response.Asks[x][1].Float64()
	}

	bids := make([]orderbook.Tranche, len(response.Bids))
	for x := range response.Bids {
		bids[x].Price = response.Bids[x][0].Float64()
		bids[x].Amount = response.Bids[x][1].Float64()
	}

	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}

	lastUpdated := time.UnixMilli(response.Timestamp)

	for x := range assets {
		err = ku.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Exchange:    ku.Name,
			Asks:        asks,
			Bids:        bids,
			Pair:        pair,
			Asset:       assets[x],
			LastUpdated: lastUpdated,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ku *Kucoin) processMarketSnapshot(respData []byte, topic string) error {
	response := WsSnapshot{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(response.Data.Symbol)
	if err != nil {
		return err
	}
	assets, err := ku.CalculateAssets(topic, pair)
	if err != nil {
		return err
	}
	for x := range assets {
		if !ku.AssetWebsocketSupport.IsAssetWebsocketSupported(assets[x]) {
			continue
		}
		ku.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: ku.Name,
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
func (ku *Kucoin) Subscribe(subscriptions subscription.List) error {
	return ku.manageSubscriptions(subscriptions, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (ku *Kucoin) Unsubscribe(subscriptions subscription.List) error {
	return ku.manageSubscriptions(subscriptions, "unsubscribe")
}

// expandManualSubscription takes a subscription list and expand all the subscriptions across the relevant assets and pairs
func (ku *Kucoin) expandManualSubscriptions(in subscription.List) (subscription.List, error) {
	subs := make(subscription.List, 0, len(in))
	for _, s := range in {
		if isSymbolChannel(s.Channel) {
			if len(s.Pairs) == 0 {
				return nil, errSubscriptionPairRequired
			}
			a := s.Asset
			if !a.IsValid() {
				a = getChannelsAssetType(s.Channel)
			}
			assetPairs := map[asset.Item]currency.Pairs{a: s.Pairs}
			n, err := ku.expandSubscription(s, assetPairs)
			if err != nil {
				return nil, err
			}
			subs = append(subs, n...)
		} else {
			subs = append(subs, s)
		}
	}
	return subs, nil
}

func (ku *Kucoin) manageSubscriptions(subs subscription.List, operation string) error {
	var errs error
	subs, errs = ku.expandManualSubscriptions(subs)
	for _, s := range subs {
		msgID := strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10)
		req := WsSubscriptionInput{
			ID:             msgID,
			Type:           operation,
			Topic:          s.Channel,
			PrivateChannel: s.Authenticated,
			Response:       true,
		}
		if respRaw, err := ku.Websocket.Conn.SendMessageReturnResponse("msgID:"+msgID, req); err != nil {
			errs = common.AppendError(errs, err)
		} else {
			rType, err := jsonparser.GetUnsafeString(respRaw, "type")
			switch {
			case err != nil:
				errs = common.AppendError(errs, err)
			case rType != "ack":
				errs = common.AppendError(errs, fmt.Errorf("%w: %s from %s", errInvalidMsgType, rType, respRaw))
			default:
				if operation == "unsubscribe" {
					err = ku.Websocket.RemoveSubscriptions(s)
				} else {
					err = ku.Websocket.AddSuccessfulSubscriptions(s)
					if ku.Verbose {
						log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s", ku.Name, s.Channel)
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

// getChannelsAssetType returns the asset type to which the subscription channel belongs to or asset.Empty
func getChannelsAssetType(channelName string) asset.Item {
	switch channelName {
	case futuresTickerV2Channel, futuresTickerChannel, futuresOrderbookLevel2Channel, futuresExecutionDataChannel, futuresOrderbookLevel2Depth5Channel, futuresOrderbookLevel2Depth50Channel, futuresContractMarketDataChannel, futuresSystemAnnouncementChannel, futuresTrasactionStatisticsTimerEventChannel, futuresTradeOrdersBySymbolChannel, futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel, futuresAccountBalanceEventChannel, futuresPositionChangeEventChannel:
		return asset.Futures
	case marketTickerChannel, marketAllTickersChannel,
		marketSnapshotChannel, marketSymbolSnapshotChannel,
		marketOrderbookLevel2Channels, marketOrderbookLevel2to5Channel,
		marketOrderbokLevel2To50Channel, marketCandlesChannel,
		marketMatchChannel, indexPriceIndicatorChannel, markPriceIndicatorChannel,
		privateSpotTradeOrders, accountBalanceChannel, spotMarketAdvancedChannel:
		return asset.Spot
	case marginFundingbookChangeChannel, marginPositionChannel, marginLoanChannel:
		return asset.Margin
	default:
		return asset.Empty
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (ku *Kucoin) generateSubscriptions() (subscription.List, error) {
	assetPairs := map[asset.Item]currency.Pairs{}
	for _, a := range ku.GetAssetTypes(false) {
		if p, err := ku.GetEnabledPairs(a); err == nil {
			assetPairs[a] = p
		} else {
			assetPairs[a] = currency.Pairs{} // err is probably that Asset isn't enabled, but we don't care about errors of any type
		}
	}
	authed := ku.Websocket.CanUseAuthenticatedEndpoints()
	subscriptions := subscription.List{}
	for _, s := range ku.Features.Subscriptions {
		if !authed && s.Authenticated {
			continue
		}
		subs, err := ku.expandSubscription(s, assetPairs)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subs...)
	}
	return subscriptions, nil
}

// expandSubscription takes a subscription and expands it across the relevant assets and pairs passed in
func (ku *Kucoin) expandSubscription(baseSub *subscription.Subscription, assetPairs map[asset.Item]currency.Pairs) (subscription.List, error) {
	var subscriptions = subscription.List{}
	if baseSub == nil {
		return nil, common.ErrNilPointer
	}
	s := baseSub.Clone()
	s.Channel = channelName(s.Channel)
	if !s.Asset.IsValid() {
		s.Asset = getChannelsAssetType(s.Channel)
	}

	if len(assetPairs[s.Asset]) == 0 {
		return nil, nil
	}

	switch {
	case s.Channel == marginLoanChannel:
		for _, c := range assetPairs[asset.Margin].GetCurrencies() {
			i := s.Clone()
			i.Channel = fmt.Sprintf(s.Channel, c)
			subscriptions = append(subscriptions, i)
		}
	case s.Channel == marketCandlesChannel:
		interval, err := ku.intervalToString(s.Interval)
		if err != nil {
			return nil, err
		}
		subs := spotOrMarginPairSubs(assetPairs, s, false, interval)
		subscriptions = append(subscriptions, subs...)
	case s.Channel == marginFundingbookChangeChannel:
		s.Channel = fmt.Sprintf(s.Channel, assetPairs[asset.Margin].GetCurrencies().Join())
		subscriptions = append(subscriptions, s)
	case s.Channel == marketSnapshotChannel:
		subs, err := spotOrMarginCurrencySubs(assetPairs, s)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subs...)
	case getChannelsAssetType(s.Channel) == asset.Futures && isSymbolChannel(s.Channel):
		for _, p := range assetPairs[asset.Futures] {
			c, err := ku.FormatExchangeCurrency(p, asset.Futures)
			if err != nil {
				continue
			}
			i := s.Clone()
			i.Channel = fmt.Sprintf(s.Channel, c)
			subscriptions = append(subscriptions, i)
		}
	case isSymbolChannel(s.Channel):
		// Subscriptions which can use a single comma-separated sub per asset
		subs := spotOrMarginPairSubs(assetPairs, s, true)
		subscriptions = append(subscriptions, subs...)
	default:
		subscriptions = append(subscriptions, s)
	}
	return subscriptions, nil
}

// isSymbolChannel returns true it this channel path ends in a formatting %s to accept a Symbol
func isSymbolChannel(c string) bool {
	return strings.HasSuffix(c, "%s") || strings.HasSuffix(c, "%v")
}

// channelName converts global channel Names used in config of channel input into kucoin channel names
// returns the name unchanged if no match is found
func channelName(name string) string {
	if s, ok := subscriptionNames[name]; ok {
		return s
	}
	return name
}

// spotOrMarginPairSubs accepts a map of pairs and a template subscription and returns a list of subscriptions for Spot and Margin pairs
// If there's a Spot subscription, it won't be added again as a Margin subscription
// If joined param is true then one subscription per asset type with the currencies comma delimited
func spotOrMarginPairSubs(assetPairs map[asset.Item]currency.Pairs, b *subscription.Subscription, join bool, fmtArgs ...any) subscription.List {
	subs := subscription.List{}
	add := func(a asset.Item, pairs currency.Pairs) {
		if len(pairs) == 0 {
			return
		}
		if join {
			f := append([]any{pairs.Join()}, fmtArgs...)
			s := b.Clone()
			s.Asset = a
			s.Channel = fmt.Sprintf(b.Channel, f...)
			subs = append(subs, s)
		} else {
			for i := range pairs {
				f := append([]any{pairs[i].String()}, fmtArgs...)
				s := b.Clone()
				s.Asset = a
				s.Channel = fmt.Sprintf(b.Channel, f...)
				subs = append(subs, s)
			}
		}
	}

	add(asset.Spot, assetPairs[asset.Spot])

	marginPairs := currency.Pairs{}
	for _, p := range assetPairs[asset.Margin] {
		if !assetPairs[asset.Spot].Contains(p, false) {
			marginPairs = marginPairs.Add(p)
		}
	}
	add(asset.Margin, marginPairs)

	return subs
}

// spotOrMarginCurrencySubs accepts a map of pairs and a template subscription and returns a list of subscriptions for every currency in Spot and Margin pairs
// If there's a Spot subscription, it won't be added again as a Margin subscription
func spotOrMarginCurrencySubs(assetPairs map[asset.Item]currency.Pairs, b *subscription.Subscription) (subscription.List, error) {
	if b == nil {
		return nil, common.ErrNilPointer
	}
	subs := subscription.List{}
	add := func(a asset.Item, currs currency.Currencies) {
		if len(currs) == 0 {
			return
		}
		for _, c := range currs {
			s := b.Clone()
			s.Asset = a
			s.Channel = fmt.Sprintf(b.Channel, c)
			subs = append(subs, s)
		}
	}

	add(asset.Spot, assetPairs[asset.Spot].GetCurrencies())

	marginCurrencies := currency.Currencies{}
	for _, c := range assetPairs[asset.Margin].GetCurrencies() {
		if !assetPairs[asset.Spot].ContainsCurrency(c) {
			marginCurrencies = marginCurrencies.Add(c)
		}
	}
	add(asset.Margin, marginCurrencies)

	return subs, nil
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

func (ku *Kucoin) setupOrderbookManager() {
	locker.Lock()
	defer locker.Unlock()
	if ku.obm == nil {
		ku.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		ku.obm.Mutex.Lock()
		for _, m1 := range ku.obm.state {
			for _, m2 := range m1 {
				for _, idk := range m2 {
					idk.initialSync = true
					idk.needsFetchingBook = true
					idk.lastUpdateID = 0
				}
			}
		}
		ku.obm.Mutex.Unlock()
	}
	for i := 0; i < maxWSOrderbookWorkers; i++ {
		// 10 workers for synchronising book
		ku.SynchroniseWebsocketOrderbook()
	}
}

// ProcessUpdate processes the websocket orderbook update
func (ku *Kucoin) ProcessUpdate(cp currency.Pair, a asset.Item, ws *WsOrderbook) error {
	updateBid := make([]orderbook.Tranche, len(ws.Changes.Bids))
	for i := range ws.Changes.Bids {
		p, err := strconv.ParseFloat(ws.Changes.Bids[i][0], 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.Changes.Bids[i][1], 64)
		if err != nil {
			return err
		}
		var sequence int64
		if len(ws.Changes.Bids[i]) > 2 && ws.Changes.Bids[i][2] != "" {
			sequence, err = strconv.ParseInt(ws.Changes.Bids[i][2], 10, 64)
			if err != nil {
				return err
			}
		}
		updateBid[i] = orderbook.Tranche{Price: p, Amount: a, ID: sequence}
	}

	updateAsk := make([]orderbook.Tranche, len(ws.Changes.Asks))
	for i := range ws.Changes.Asks {
		p, err := strconv.ParseFloat(ws.Changes.Asks[i][0], 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.Changes.Asks[i][1], 64)
		if err != nil {
			return err
		}
		var sequence int64
		if len(ws.Changes.Asks[i]) > 2 && ws.Changes.Asks[i][2] != "" {
			sequence, err = strconv.ParseInt(ws.Changes.Asks[i][2], 10, 64)
			if err != nil {
				return err
			}
		}
		updateAsk[i] = orderbook.Tranche{Price: p, Amount: a, ID: sequence}
	}

	return ku.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       updateBid,
		Asks:       updateAsk,
		Pair:       cp,
		UpdateID:   ws.SequenceEnd,
		UpdateTime: ws.TimeMS.Time(),
		Asset:      a,
	})
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (ku *Kucoin) applyBufferUpdate(pair currency.Pair, assetType asset.Item) error {
	fetching, needsFetching, err := ku.obm.handleFetchingBook(pair, assetType)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}
	if needsFetching {
		if ku.Verbose {
			log.Debugf(log.WebsocketMgr, "%s Orderbook: Fetching via REST\n", ku.Name)
		}
		return ku.obm.fetchBookViaREST(pair, assetType)
	}

	recent, err := ku.Websocket.Orderbook.GetOrderbook(pair, assetType)
	if err != nil {
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			ku.Name,
			err)
	}

	if recent != nil {
		err = ku.obm.checkAndProcessUpdate(ku.ProcessUpdate, pair, assetType, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				ku.Name,
				err)
			err = ku.obm.setNeedsFetchingBook(pair, assetType)
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
func (ku *Kucoin) SynchroniseWebsocketOrderbook() {
	ku.Websocket.Wg.Add(1)
	go func() {
		defer ku.Websocket.Wg.Done()
		for {
			select {
			case <-ku.Websocket.ShutdownC:
				for {
					select {
					case <-ku.obm.jobs:
					default:
						return
					}
				}
			case j := <-ku.obm.jobs:
				err := ku.processJob(j.Pair, j.AssetType)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%s processing websocket orderbook error %v",
						ku.Name, err)
				}
			}
		}
	}()
}

// SeedLocalCache seeds depth data
func (ku *Kucoin) SeedLocalCache(ctx context.Context, p currency.Pair, assetType asset.Item) error {
	var ob *Orderbook
	var err error
	ob, err = ku.GetPartOrderbook100(ctx, p.String())
	if err != nil {
		return err
	}
	if ob.Sequence <= 0 {
		return fmt.Errorf("%w p", errMissingOrderbookSequence)
	}
	return ku.SeedLocalCacheWithBook(p, ob, assetType)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (ku *Kucoin) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *Orderbook, assetType asset.Item) error {
	newOrderBook := orderbook.Base{
		Pair:            p,
		Asset:           assetType,
		Exchange:        ku.Name,
		LastUpdated:     time.Now(),
		LastUpdateID:    orderbookNew.Sequence,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		Bids:            make(orderbook.Tranches, len(orderbookNew.Bids)),
		Asks:            make(orderbook.Tranches, len(orderbookNew.Asks)),
	}
	for i := range orderbookNew.Bids {
		newOrderBook.Bids[i] = orderbook.Tranche{
			Amount: orderbookNew.Bids[i].Amount,
			Price:  orderbookNew.Bids[i].Price,
		}
	}
	for i := range orderbookNew.Asks {
		newOrderBook.Asks[i] = orderbook.Tranche{
			Amount: orderbookNew.Asks[i].Amount,
			Price:  orderbookNew.Asks[i].Price,
		}
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// processJob fetches and processes orderbook updates
func (ku *Kucoin) processJob(p currency.Pair, assetType asset.Item) error {
	err := ku.SeedLocalCache(context.TODO(), p, assetType)
	if err != nil {
		err = ku.obm.stopFetchingBook(p, assetType)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, assetType, err)
	}

	err = ku.obm.stopFetchingBook(p, assetType)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = ku.applyBufferUpdate(p, assetType)
	if err != nil {
		ku.flushAndCleanup(p, assetType)
		return err
	}
	return nil
}

// flushAndCleanup flushes orderbook and clean local cache
func (ku *Kucoin) flushAndCleanup(p currency.Pair, assetType asset.Item) {
	errClean := ku.Websocket.Orderbook.FlushOrderbook(p, assetType)
	if errClean != nil {
		log.Errorf(log.WebsocketMgr,
			"%s flushing websocket error: %v",
			ku.Name,
			errClean)
	}
	errClean = ku.obm.cleanup(p, assetType)
	if errClean != nil {
		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v",
			ku.Name,
			errClean)
	}
}

// stageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) stageWsUpdate(u *WsOrderbook, pair currency.Pair, a asset.Item) error {
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

// handleFetchingBook checks if a full book is being fetched or needs to be
// fetched
func (o *orderbookManager) handleFetchingBook(pair currency.Pair, assetType asset.Item) (fetching, needsFetching bool, err error) {
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

// stopFetchingBook completes the book fetching.
func (o *orderbookManager) stopFetchingBook(pair currency.Pair, assetType asset.Item) error {
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

// completeInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) completeInitialSync(pair currency.Pair, assetType asset.Item) error {
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

// checkIsInitialSync checks status if the book is Initial Sync being via the REST
// protocol.
func (o *orderbookManager) checkIsInitialSync(pair currency.Pair, assetType asset.Item) (bool, error) {
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

// fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) fetchBookViaREST(pair currency.Pair, assetType asset.Item) error {
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

func (o *orderbookManager) checkAndProcessUpdate(processor func(currency.Pair, asset.Item, *WsOrderbook) error, pair currency.Pair, assetType asset.Item, recent *orderbook.Base) error {
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
			process, err := state.validate(d, recent)
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

// validate checks for correct update alignment
func (u *update) validate(updt *WsOrderbook, recent *orderbook.Base) (bool, error) {
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

// cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) cleanup(pair currency.Pair, assetType asset.Item) error {
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
	_ = o.stopFetchingBook(pair, assetType)
	_ = o.completeInitialSync(pair, assetType)
	_ = o.stopNeedsFetchingBook(pair, assetType)
	return nil
}

// stopNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) stopNeedsFetchingBook(pair currency.Pair, assetType asset.Item) error {
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
func (ku *Kucoin) CalculateAssets(topic string, cp currency.Pair) ([]asset.Item, error) {
	switch {
	case cp.Quote.Equal(currency.USDTM), strings.HasPrefix(topic, "/contract"):
		if err := ku.CurrencyPairs.IsAssetEnabled(asset.Futures); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Futures}, nil
	case strings.HasPrefix(topic, "/margin"), strings.HasPrefix(topic, "/index"):
		if err := ku.CurrencyPairs.IsAssetEnabled(asset.Margin); err != nil {
			if !errors.Is(err, asset.ErrNotSupported) {
				return nil, err
			}
			return nil, nil
		}
		return []asset.Item{asset.Margin}, nil
	default:
		resp := make([]asset.Item, 0, 2)
		spotEnabled, err := ku.IsPairEnabled(cp, asset.Spot)
		if err != nil && !errors.Is(currency.ErrCurrencyNotFound, err) {
			return nil, err
		}
		if spotEnabled {
			resp = append(resp, asset.Spot)
		}
		marginEnabled, err := ku.IsPairEnabled(cp, asset.Margin)
		if err != nil && !errors.Is(currency.ErrCurrencyNotFound, err) {
			return nil, err
		}
		if marginEnabled {
			resp = append(resp, asset.Margin)
		}
		return resp, nil
	}
}
