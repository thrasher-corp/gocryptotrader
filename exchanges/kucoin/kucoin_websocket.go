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

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	publicBullets  = "/v1/bullet-public"
	privateBullets = "/v1/bullet-private"

	// spot channels
	marketTickerChannel                    = "/market/ticker:%s" // /market/ticker:{symbol},{symbol}...
	marketAllTickersChannel                = "/market/ticker:all"
	marketTickerSnapshotChannel            = "/market/snapshot:%s"          // /market/snapshot:{symbol}
	marketTickerSnapshotForCurrencyChannel = "/market/snapshot:"            // /market/snapshot:{market} <--- market represents a currency
	marketOrderbookLevel2Channels          = "/market/level2:%s"            // /market/level2:{symbol},{symbol}...
	marketOrderbookLevel2to5Channel        = "/spotMarket/level2Depth5:%s"  // /spotMarket/level2Depth5:{symbol},{symbol}...
	marketOrderbokLevel2To50Channel        = "/spotMarket/level2Depth50:%s" // /spotMarket/level2Depth50:{symbol},{symbol}...
	marketCandlesChannel                   = "/market/candles:%s_%s"        // /market/candles:{symbol}_{type}
	marketMatchChannel                     = "/market/match:%s"             // /market/match:{symbol},{symbol}...
	indexPriceIndicatorChannel             = "/indicator/index:%s"          // /indicator/index:{symbol0},{symbol1}..
	markPriceIndicatorChannel              = "/indicator/markPrice:%s"      // /indicator/markPrice:{symbol0},{symbol1}...
	marginFundingbookChangeChannel         = "/margin/fundingBook:%s"       // /margin/fundingBook:{currency0},{currency1}...

	// Private channel

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

var defaultSubscriptionChannels = []string{
	marketTickerChannel,
	marginFundingbookChangeChannel,
	marketCandlesChannel,
	marketOrderbookLevel2Channels,
	marketOrderbookLevel2to5Channel,
	marketTickerSnapshotForCurrencyChannel,

	futuresTickerV2Channel,
	futuresOrderbookLevel2Depth50Channel,
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

var requiredSubscriptionIDS map[string]bool
var requiredSubscriptionIDSLock sync.Mutex

// checkRequiredSubscriptionID check whether the id included in the required subscription ids list.
func (ku *Kucoin) checkRequiredSubscriptionID(id string) bool {
	if len(requiredSubscriptionIDS) > 0 {
		if requiredSubscriptionIDS[id] {
			requiredSubscriptionIDSLock.Lock()
			delete(requiredSubscriptionIDS, id)
			requiredSubscriptionIDSLock.Unlock()
			return true
		}
	}
	return false
}

// WsConnect creates a new websocket connection.
func (ku *Kucoin) WsConnect() error {
	if !ku.Websocket.IsEnabled() || !ku.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
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
	if err != nil {
		return err
	}
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
			AuthRequest:   true,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	})
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
	} else if resp.ID != "" {
		if ku.checkRequiredSubscriptionID(resp.ID) {
			if !ku.Websocket.Match.IncomingWithData(resp.ID, respData) {
				return fmt.Errorf("can not match subscription message with signature ID:%s", resp.ID)
			}
		}
		return nil
	}
	if resp.Type == "pong" || resp.Type == "welcome" {
		return nil
	}
	topicInfo := strings.Split(resp.Topic, ":")
	switch {
	case strings.HasPrefix(marketAllTickersChannel, topicInfo[0]) ||
		strings.HasPrefix(marketTickerChannel, topicInfo[0]):
		var instruments string
		if topicInfo[1] == "all" {
			instruments = resp.Subject
		} else {
			instruments = topicInfo[1]
		}
		return ku.processTicker(resp.Data, instruments)
	case strings.HasPrefix(marketTickerSnapshotChannel, topicInfo[0]) ||
		strings.HasPrefix(marketTickerSnapshotForCurrencyChannel, topicInfo[0]):
		return ku.processMarketSnapshot(resp.Data, topicInfo[1])
	case strings.HasPrefix(marketOrderbookLevel2Channels, topicInfo[0]):
		return ku.processOrderbook(resp.Data)
	case strings.HasPrefix(marketOrderbookLevel2to5Channel, topicInfo[0]),
		strings.HasPrefix(marketOrderbokLevel2To50Channel, topicInfo[0]):
		return ku.processOrderbookWithDepth(resp.Data, topicInfo[1])
	case strings.HasPrefix(marketCandlesChannel, topicInfo[0]):
		symbolAndInterval := strings.Split(topicInfo[1], currency.UnderscoreDelimiter)
		if len(symbolAndInterval) != 2 {
			return errMalformedData
		}
		return ku.processCandlesticks(resp.Data, symbolAndInterval[0], symbolAndInterval[1])
	case strings.HasPrefix(marketMatchChannel, topicInfo[0]):
		return ku.processTradeData(resp.Data, topicInfo[1])
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
		return ku.processOrderChangeEvent(resp.Data)
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
		return ku.processFuturesOrderbookLevel2(resp.Data, topicInfo[1])
	case strings.HasPrefix(futuresExecutionDataChannel, topicInfo[0]):
		var response WsFuturesExecutionData
		return ku.processData(resp.Data, &response)
	case strings.HasPrefix(futuresOrderbookLevel2Depth5Channel, topicInfo[0]),
		strings.HasPrefix(futuresOrderbookLevel2Depth50Channel, topicInfo[0]):
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
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
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
	pair, err := currency.NewPairFromString(resp.Symbol)
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

func (ku *Kucoin) processFuturesOrderbookLevel5(respData []byte, instruments string) error {
	resp := WsOrderbookLevel5{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instruments)
	if err != nil {
		return err
	}
	base := orderbook.Base{
		Exchange:        ku.Name,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		LastUpdated:     resp.Timestamp.Time(),
		Pair:            pair,
		Asset:           asset.Futures,
		Asks:            resp.Asks,
		Bids:            resp.Bids,
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&base)
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
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	base := orderbook.Base{
		Exchange:        ku.Name,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		LastUpdated:     detail.Time,
		Pair:            pair,
		Asset:           asset.Futures,
		Asks:            detail.Asks,
		Bids:            detail.Bids,
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&base)
}

func (ku *Kucoin) processFuturesTickerV2(respData []byte) error {
	resp := WsFuturesTicker{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- &ticker.Price{
		AssetType:    asset.Futures,
		Last:         resp.FilledSize,
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
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
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

func (ku *Kucoin) processOrderChangeEvent(respData []byte) error {
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
		AssetType:       asset.Spot,
		Date:            response.OrderTime.Time(),
		LastUpdated:     response.Timestamp.Time(),
		Pair:            pair,
	}
	return nil
}

func (ku *Kucoin) processTradeData(respData []byte, instrument string) error {
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
	return ku.Websocket.Trade.Update(saveTradeData, trade.Data{
		CurrencyPair: pair,
		Timestamp:    response.Time.Time(),
		Price:        response.Price,
		Amount:       response.Size,
		Side:         side,
		Exchange:     ku.Name,
		TID:          response.TradeID,
		AssetType:    asset.Spot,
	})
}

func (ku *Kucoin) processTicker(respData []byte, instrument string) error {
	response := WsTicker{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- &ticker.Price{
		AssetType:    asset.Spot,
		Last:         response.Size,
		LastUpdated:  response.Timestamp.Time(),
		ExchangeName: ku.Name,
		Pair:         pair,
		Ask:          response.BestAsk,
		Bid:          response.BestBid,
		AskSize:      response.BestAskSize,
		BidSize:      response.BestBidSize,
		Volume:       response.Size,
	}
	return nil
}

func (ku *Kucoin) processCandlesticks(respData []byte, instrument, intervalString string) error {
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
	ku.Websocket.DataHandler <- stream.KlineData{
		Timestamp:  response.Time.Time(),
		Pair:       pair,
		AssetType:  asset.Spot,
		Exchange:   ku.Name,
		StartTime:  resp.Candles.StartTime,
		Interval:   intervalString,
		OpenPrice:  resp.Candles.OpenPrice,
		ClosePrice: resp.Candles.ClosePrice,
		HighPrice:  resp.Candles.HighPrice,
		LowPrice:   resp.Candles.LowPrice,
		Volume:     resp.Candles.TransactionVolume,
	}
	return nil
}

func (ku *Kucoin) processOrderbookWithDepth(respData []byte, instrument string) error {
	response := WsLevel2Orderbook{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	update := orderbook.Update{
		UpdateTime: response.TimeMS.Time(),
		Pair:       pair,
		Asset:      asset.Spot,
	}
	for x := range response.Asks {
		item := orderbook.Item{}
		item.Price, err = strconv.ParseFloat(response.Asks[x][0], 64)
		if err != nil {
			return err
		}
		item.Amount, err = strconv.ParseFloat(response.Asks[x][1], 64)
		if err != nil {
			return err
		}
		update.Asks = append(update.Asks, item)
	}
	for x := range response.Bids {
		item := orderbook.Item{}
		item.Price, err = strconv.ParseFloat(response.Bids[x][0], 64)
		if err != nil {
			return err
		}
		item.Amount, err = strconv.ParseFloat(response.Bids[x][1], 64)
		if err != nil {
			return err
		}
		update.Bids = append(update.Bids, item)
	}
	return ku.Websocket.Orderbook.Update(&update)
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (ku *Kucoin) UpdateLocalBuffer(wsdp *WsOrderbook) (bool, error) {
	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		return false, err
	}

	format, err := ku.GetPairFormat(asset.Spot, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(wsdp.Symbol,
		enabledPairs,
		format)
	if err != nil {
		return false, err
	}

	err = ku.obm.stageWsUpdate(wsdp, currencyPair, asset.Spot)
	if err != nil {
		init, err2 := ku.obm.checkIsInitialSync(currencyPair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = ku.applyBufferUpdate(currencyPair)
	if err != nil {
		ku.flushAndCleanup(currencyPair)
	}

	return false, err
}

func (ku *Kucoin) processOrderbook(respData []byte) error {
	var response WsOrderbook
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}

	init, err := ku.UpdateLocalBuffer(&response)
	if err != nil {
		if init {
			return nil
		}
		return fmt.Errorf("%v - UpdateLocalCache error: %s",
			ku.Name,
			err)
	}
	return nil
}

func (ku *Kucoin) processMarketSnapshot(respData []byte, instrument string) error {
	response := WsSpotTicker{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(instrument)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: ku.Name,
		AssetType:    asset.Spot,
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
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (ku *Kucoin) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return ku.handleSubscriptions(subscriptions, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (ku *Kucoin) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return ku.handleSubscriptions(subscriptions, "unsubscribe")
}

func (ku *Kucoin) handleSubscriptions(subscriptions []stream.ChannelSubscription, operation string) error {
	if requiredSubscriptionIDS == nil {
		requiredSubscriptionIDS = map[string]bool{}
	}
	payloads, err := ku.generatePayloads(subscriptions, operation)
	if err != nil {
		return err
	}

	var errs common.Errors
	for x := range payloads {
		err = ku.Websocket.Conn.SendJSONMessage(payloads[x])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ku.Websocket.AddSuccessfulSubscriptions(subscriptions[x])
	}
	return errs.Unwrap()
}

// getChannelsAssetType returns the asset type to which the subscription channel belongs to
// or returns an error otherwise.
func (ku *Kucoin) getChannelsAssetType(channelName string) (asset.Item, error) {
	switch channelName {
	case futuresTickerV2Channel, futuresTickerChannel, futuresOrderbookLevel2Channel, futuresExecutionDataChannel, futuresOrderbookLevel2Depth5Channel, futuresOrderbookLevel2Depth50Channel, futuresContractMarketDataChannel, futuresSystemAnnouncementChannel, futuresTrasactionStatisticsTimerEventChannel, futuresTradeOrdersBySymbolChannel, futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel, futuresAccountBalanceEventChannel, futuresPositionChangeEventChannel:
		return asset.Futures, nil
	case marketTickerChannel, marketAllTickersChannel, marketTickerSnapshotChannel, marketTickerSnapshotForCurrencyChannel, marketOrderbookLevel2Channels, marketOrderbookLevel2to5Channel, marketOrderbokLevel2To50Channel, marketCandlesChannel, marketMatchChannel, indexPriceIndicatorChannel, markPriceIndicatorChannel, marginFundingbookChangeChannel, privateSpotTradeOrders, accountBalanceChannel, marginPositionChannel, marginLoanChannel,
		spotMarketAdvancedChannel:
		return asset.Spot, nil
	default:
		return asset.Empty, errors.New("channel not supported")
	}
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket.
func (ku *Kucoin) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultSubscriptionChannels
	subscriptions := []stream.ChannelSubscription{}
	if ku.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels,
			accountBalanceChannel,
			marginPositionChannel,
			marginLoanChannel,

			// futures authenticated channels
			futuresTradeOrdersBySymbolChannel,
			futuresTradeOrderChannel,
			futuresStopOrdersLifecycleEventChannel,
			futuresAccountBalanceEventChannel,
		)
	}
	spotPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	futuresPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	for x := range channels {
		switch channels[x] {
		case accountBalanceChannel, marginPositionChannel,
			futuresTradeOrderChannel, futuresStopOrdersLifecycleEventChannel,
			spotMarketAdvancedChannel, privateSpotTradeOrders,
			marketAllTickersChannel, futuresSystemAnnouncementChannel,
			futuresAccountBalanceEventChannel:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
			})
		case marketTickerSnapshotChannel,
			marketOrderbookLevel2Channels,
			marketTickerSnapshotForCurrencyChannel,
			marketOrderbookLevel2to5Channel:
			for b := range spotPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Asset:    asset.Spot,
					Currency: spotPairs[b],
				})
			}
		case marketOrderbokLevel2To50Channel, indexPriceIndicatorChannel,
			markPriceIndicatorChannel,
			marketMatchChannel, marketTickerChannel:
			pairStrings := spotPairs.Join()
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
				Asset:   asset.Spot,
				Params:  map[string]interface{}{"symbols": pairStrings},
			})
		case marketCandlesChannel:
			for b := range spotPairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Asset:    asset.Spot,
					Currency: spotPairs[b],
					Params:   map[string]interface{}{"interval": kline.FifteenMin},
				})
			}
		case marginLoanChannel:
			currencyExist := map[currency.Code]bool{}
			for b := range spotPairs {
				if !currencyExist[spotPairs[b].Base] {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: currency.Pair{Base: spotPairs[b].Base},
					})
					currencyExist[spotPairs[b].Base] = true
				}
				if !currencyExist[spotPairs[b].Quote] {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: currency.Pair{Base: spotPairs[b].Quote},
					})
					currencyExist[spotPairs[b].Quote] = true
				}
			}
		case marginFundingbookChangeChannel:
			currencyExist := map[currency.Code]bool{}
			for b := range spotPairs {
				okay := currencyExist[spotPairs[b].Base]
				if !okay {
					currencyExist[spotPairs[b].Base] = true
				}
				okay = currencyExist[spotPairs[b].Quote]
				if !okay {
					currencyExist[spotPairs[b].Quote] = true
				}
			}
			currencies := ""
			for b := range currencyExist {
				currencies += b.String() + ","
			}
			currencies = strings.TrimSuffix(currencies, ",")
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
				Params:  map[string]interface{}{"currencies": currencies},
			})
		case futuresTickerV2Channel,
			futuresTickerChannel,
			futuresExecutionDataChannel,
			futuresOrderbookLevel2Channel,
			futuresOrderbookLevel2Depth5Channel,
			futuresOrderbookLevel2Depth50Channel,
			futuresContractMarketDataChannel,
			futuresTradeOrdersBySymbolChannel,
			futuresPositionChangeEventChannel,
			futuresTrasactionStatisticsTimerEventChannel:
			for b := range futuresPairs {
				futuresPairs[b], err = ku.FormatExchangeCurrency(futuresPairs[b], asset.Futures)
				if err != nil {
					continue
				}
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Asset:    asset.Futures,
					Currency: futuresPairs[b],
				})
			}
		}
	}
	return subscriptions, nil
}

func (ku *Kucoin) generatePayloads(subscriptions []stream.ChannelSubscription, operation string) ([]WsSubscriptionInput, error) {
	payloads := make([]WsSubscriptionInput, len(subscriptions))
	marketTickerSnapshotForCurrencyChannelCurrencyFilter := map[currency.Code]int{}
	for x := range subscriptions {
		var err error
		var a asset.Item
		a, err = ku.getChannelsAssetType(subscriptions[x].Channel)
		if err != nil {
			return nil, err
		}
		subscriptions[x].Currency, err = ku.FormatExchangeCurrency(subscriptions[x].Currency, a)
		if err != nil {
			return nil, err
		}
		if subscriptions[x].Asset == asset.Futures {
			subscriptions[x].Currency, err = ku.FormatExchangeCurrency(subscriptions[x].Currency, asset.Futures)
			if err != nil {
				continue
			}
		}
		switch subscriptions[x].Channel {
		case marketTickerChannel,
			marketOrderbookLevel2Channels,
			marketOrderbookLevel2to5Channel,
			marketOrderbokLevel2To50Channel,
			indexPriceIndicatorChannel,
			marketMatchChannel,
			markPriceIndicatorChannel:
			symbols, okay := subscriptions[x].Params["symbols"].(string)
			if !okay {
				if subscriptions[x].Currency.IsEmpty() {
					return nil, errors.New("symbols not passed")
				}
				symbols = subscriptions[x].Currency.String()
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, symbols),
				Response: true,
			}
		case marketAllTickersChannel,
			privateSpotTradeOrders,
			accountBalanceChannel,
			marginPositionChannel,
			spotMarketAdvancedChannel,
			futuresTradeOrderChannel,
			futuresStopOrdersLifecycleEventChannel,
			futuresAccountBalanceEventChannel, futuresSystemAnnouncementChannel:
			input := WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    subscriptions[x].Channel,
				Response: true,
			}
			switch subscriptions[x].Channel {
			case futuresTradeOrderChannel,
				futuresStopOrdersLifecycleEventChannel,
				futuresAccountBalanceEventChannel,
				privateSpotTradeOrders,
				accountBalanceChannel,
				marginPositionChannel,
				spotMarketAdvancedChannel:
				input.PrivateChannel = true
			}
			payloads[x] = input
		case marketTickerSnapshotChannel,
			futuresPositionChangeEventChannel,
			futuresTradeOrdersBySymbolChannel,
			futuresTrasactionStatisticsTimerEventChannel,
			futuresContractMarketDataChannel,
			futuresOrderbookLevel2Depth50Channel,
			futuresOrderbookLevel2Depth5Channel,
			futuresExecutionDataChannel,
			futuresOrderbookLevel2Channel,
			futuresTickerChannel,
			futuresTickerV2Channel: // Symbols
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.String()),
				Response: true,
			}
			switch subscriptions[x].Channel {
			case futuresPositionChangeEventChannel,
				futuresTradeOrdersBySymbolChannel:
				payloads[x].PrivateChannel = true
			}
		case marketTickerSnapshotForCurrencyChannel,
			marginLoanChannel:
			// 3 means the Currency is used by both switch cases
			// 2 means the currency is used by channel = marginLoanChannel
			// 1 if used by marketTickerSnapshotForCurrencyChannel
			if stat := marketTickerSnapshotForCurrencyChannelCurrencyFilter[subscriptions[x].Currency.Base]; stat == 3 || (stat == 2 && subscriptions[x].Channel == marginLoanChannel) || stat == 1 {
				continue
			}
			input := WsSubscriptionInput{}
			if subscriptions[x].Channel == marginLoanChannel {
				input.PrivateChannel = true
				marketTickerSnapshotForCurrencyChannelCurrencyFilter[subscriptions[x].Currency.Base] += 2
			} else {
				marketTickerSnapshotForCurrencyChannelCurrencyFilter[subscriptions[x].Currency.Base]++
				subscriptions[x].Channel += "%s"
			}
			input.ID = strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10)
			input.Type = operation
			input.Topic = fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.Base.Upper().String())
			input.Response = true
			payloads[x] = input
		case marketCandlesChannel:
			interval, err := ku.intervalToString(subscriptions[x].Params["interval"].(kline.Interval))
			if err != nil {
				return nil, err
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.Upper().String(), interval),
				Response: true,
			}
		case marginFundingbookChangeChannel:
			currencies, okay := subscriptions[x].Params["currencies"].(string)
			if !okay {
				return nil, errors.New("currencies not passed")
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, currencies),
				Response: true,
			}
		}
	}
	return payloads, nil
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
	Pair currency.Pair
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
	updateBid := make([]orderbook.Item, len(ws.Changes.Bids))
	for i := range ws.Changes.Bids {
		p, err := strconv.ParseFloat(ws.Changes.Bids[i][0], 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.Changes.Bids[i][1], 64)
		if err != nil {
			return err
		}
		sequence, err := strconv.ParseInt(ws.Changes.Bids[i][2], 10, 64)
		if err != nil {
			return err
		}
		updateBid[i] = orderbook.Item{Price: p, Amount: a, ID: sequence}
	}

	updateAsk := make([]orderbook.Item, len(ws.Changes.Asks))
	for i := range ws.Changes.Asks {
		p, err := strconv.ParseFloat(ws.Changes.Asks[i][0], 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(ws.Changes.Asks[i][1], 64)
		if err != nil {
			return err
		}
		sequence, err := strconv.ParseInt(ws.Changes.Asks[i][2], 10, 64)
		if err != nil {
			return err
		}
		updateAsk[i] = orderbook.Item{Price: p, Amount: a, ID: sequence}
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
func (ku *Kucoin) applyBufferUpdate(pair currency.Pair) error {
	fetching, needsFetching, err := ku.obm.handleFetchingBook(pair)
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
		return ku.obm.fetchBookViaREST(pair)
	}

	recent, err := ku.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			ku.Name,
			err)
	}

	if recent != nil {
		err = ku.obm.checkAndProcessUpdate(ku.ProcessUpdate, pair, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				ku.Name,
				err)
			err = ku.obm.setNeedsFetchingBook(pair)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// setNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) setNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
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
				err := ku.processJob(j.Pair)
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
func (ku *Kucoin) SeedLocalCache(ctx context.Context, p currency.Pair) error {
	ob, err := ku.GetPartOrderbook100(ctx, p.String())
	if err != nil {
		return err
	}
	if ob.Sequence <= 0 {
		return fmt.Errorf("%w p", errMissingOrderbookSequence)
	}
	return ku.SeedLocalCacheWithBook(p, ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (ku *Kucoin) SeedLocalCacheWithBook(p currency.Pair, orderbookNew *Orderbook) error {
	newOrderBook := orderbook.Base{
		Pair:            p,
		Asset:           asset.Spot,
		Exchange:        ku.Name,
		LastUpdateID:    orderbookNew.Sequence,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		Bids:            make(orderbook.Items, len(orderbookNew.Bids)),
		Asks:            make(orderbook.Items, len(orderbookNew.Asks)),
	}
	for i := range orderbookNew.Bids {
		newOrderBook.Bids[i] = orderbook.Item{
			Amount: orderbookNew.Bids[i].Amount,
			Price:  orderbookNew.Bids[i].Price,
		}
	}
	for i := range orderbookNew.Asks {
		newOrderBook.Asks[i] = orderbook.Item{
			Amount: orderbookNew.Asks[i].Amount,
			Price:  orderbookNew.Asks[i].Price,
		}
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// processJob fetches and processes orderbook updates
func (ku *Kucoin) processJob(p currency.Pair) error {
	err := ku.SeedLocalCache(context.TODO(), p)
	if err != nil {
		err = ku.obm.stopFetchingBook(p)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, asset.Spot, err)
	}

	err = ku.obm.stopFetchingBook(p)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = ku.applyBufferUpdate(p)
	if err != nil {
		ku.flushAndCleanup(p)
		return err
	}
	return nil
}

// flushAndCleanup flushes orderbook and clean local cache
func (ku *Kucoin) flushAndCleanup(p currency.Pair) {
	errClean := ku.Websocket.Orderbook.FlushOrderbook(p, asset.Spot)
	if errClean != nil {
		log.Errorf(log.WebsocketMgr,
			"%s flushing websocket error: %v",
			ku.Name,
			errClean)
	}
	errClean = ku.obm.cleanup(p)
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
func (o *orderbookManager) handleFetchingBook(pair currency.Pair) (fetching, needsFetching bool, err error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
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
func (o *orderbookManager) stopFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.fetchingBook {
		return fmt.Errorf("fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.fetchingBook = false
	return nil
}

// completeInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) completeInitialSync(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}
	if !state.initialSync {
		return fmt.Errorf("initital sync already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.initialSync = false
	return nil
}

// checkIsInitialSync checks status if the book is Initial Sync being via the REST
// protocol.
func (o *orderbookManager) checkIsInitialSync(pair currency.Pair) (bool, error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			fmt.Errorf("checkIsInitialSync of orderbook cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}
	return state.initialSync, nil
}

// fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) fetchBookViaREST(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()

	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}

	state.initialSync = true
	state.fetchingBook = true

	select {
	case o.jobs <- job{pair}:
		return nil
	default:
		return fmt.Errorf("%s %s book synchronisation channel blocked up",
			pair,
			asset.Spot)
	}
}

func (o *orderbookManager) checkAndProcessUpdate(processor func(currency.Pair, asset.Item, *WsOrderbook) error, pair currency.Pair, recent *orderbook.Base) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
			pair, asset.Spot)
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
				err := processor(pair, asset.Spot, d)
				if err != nil {
					return fmt.Errorf("%s %s processing update error: %w",
						pair, asset.Spot, err)
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
				asset.Spot)
		}
		u.initialSync = false
	}
	return true, nil
}

// cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) cleanup(pair currency.Pair) error {
	o.Lock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		o.Unlock()
		return fmt.Errorf("cleanup cannot match %s %s to hash table",
			pair,
			asset.Spot)
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
	_ = o.stopFetchingBook(pair)
	_ = o.completeInitialSync(pair)
	_ = o.stopNeedsFetchingBook(pair)
	return nil
}

// stopNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) stopNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.needsFetchingBook {
		return fmt.Errorf("needs fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.needsFetchingBook = false
	return nil
}
