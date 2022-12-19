package kucoin

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
)

const (
	publicBullets  = "/v1/bullet-public"
	privateBullets = "/v1/bullet-private"

	// channels

	channelPing = "ping"
	channelPong = "pong"
	typeWelcome = "welcome"

	// spot channels
	marketTickerChannel                    = "/market/ticker:%s" // /market/ticker:{symbol},{symbol}...
	marketAllTickersChannel                = "/market/ticker:all"
	marketTickerSnapshotChannel            = "/market/snapshot:%s"          // /market/snapshot:{symbol}
	marketTickerSnapshotForCurrencyChannel = "/market/snapshot:"            // /market/snapshot:{market}
	marketOrderbookLevel2Channels          = "/market/level2:%s"            // /market/level2:{symbol},{symbol}...
	marketOrderbookLevel2to5Channel        = "/spotMarket/level2Depth5:%s"  // /spotMarket/level2Depth5:{symbol},{symbol}...
	marketOrderbokLevel2To50Channel        = "/spotMarket/level2Depth50:%s" // /spotMarket/level2Depth50:{symbol},{symbol}...
	marketCandlesChannel                   = "/market/candles:%s_%s"        // /market/candles:{symbol}_{type}
	marketMatchChannel                     = "/market/match:%s"             // /market/match:{symbol},{symbol}...
	indexPriceIndicatorChannel             = "/indicator/index:%s"          // /indicator/index:{symbol0},{symbol1}..
	markPriceIndicatorChannel              = "/indicator/markPrice:%s"      // /indicator/markPrice:{symbol0},{symbol1}...
	marginFundingbookChangeChannel         = "/margin/fundingBook:%s"       // /margin/fundingBook:{currency0},{currency1}...

	// Private channel

	privateChannel            = "/spotMarket/tradeOrders"
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
	marketTickerSnapshotForCurrencyChannel,
	marketOrderbokLevel2To50Channel,
	marginFundingbookChangeChannel,
	marketCandlesChannel,

	futuresTickerV2Channel,
	futuresOrderbookLevel2Depth50Channel,
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
	} else {
		instances, err = ku.GetInstanceServers(context.Background())
	}
	if err != nil {
		return err
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
	pingMessage, err := json.Marshal(&WSConnMessages{
		ID:   strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
		Type: channelPing,
	})
	if err != nil {
		return err
	}
	ku.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Delay:       time.Millisecond * time.Duration(instances.InstanceServers[0].PingTimeout),
		Message:     pingMessage,
		MessageType: websocket.TextMessage,
	})
	return nil
}

// GetInstanceServers retrives the server list and temporary public token
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

// GetAuthenticatedInstanceServers retrives server instances for authenticated users.
func (ku *Kucoin) GetAuthenticatedInstanceServers(ctx context.Context) (*WSInstanceServers, error) {
	response := struct {
		Data WSInstanceServers `json:"data"`
		Error
	}{}
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, privateBullets, nil, &response)
	if err != nil && strings.Contains(err.Error(), "400003") {
		return &response.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, privateBullets, nil, &response)
	}
	return &response.Data, err
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
	case strings.HasPrefix(marketOrderbookLevel2Channels, topicInfo[0]),
		strings.HasPrefix(marketOrderbookLevel2to5Channel, topicInfo[0]),
		strings.HasPrefix(marketOrderbokLevel2To50Channel, topicInfo[0]):
		return ku.processOrderbook(resp.Data, resp.Subject, topicInfo[1])
	case strings.HasPrefix(marketCandlesChannel, topicInfo[0]):
		symbolAndInterval := strings.Split(topicInfo[1], "_")
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
	case strings.HasPrefix(privateChannel, topicInfo[0]):
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

		// ------
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
		Date:         resp.CreatedAt,
		LastUpdated:  resp.Timestamp,
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
		LastUpdated:     resp.OrderTime,
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
		LastUpdated:     resp.Timestamp,
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
	detail, err := ku.GetFuturesOrderbook(context.Background(), instrument)
	if err != nil {
		return err
	}
	detail.Sequence = resp.Sequence
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
		LastUpdated:  resp.FilledTime,
		ExchangeName: ku.Name,
		Pair:         pair,
		Ask:          resp.BestAskPrice,
		Bid:          resp.BestBidPrice,
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
		Date:         resp.CreatedAt,
		LastUpdated:  resp.Timestamp,
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
		Date:            response.OrderTime,
		LastUpdated:     response.Timestamp,
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
		Timestamp:    time.UnixMilli(response.Time),
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
		LastUpdated:  time.Now(),
		ExchangeName: ku.Name,
		Pair:         pair,
		Ask:          response.BestAsk,
		Bid:          response.BestBid,
		AskSize:      response.BestAskSize,
		BidSize:      response.BestBidSize,
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
		Timestamp:  time.UnixMilli(response.Time),
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

func (ku *Kucoin) processOrderbook(respData []byte, subject, instrument string) error {
	response := WsOrderbook{}
	var err error
	if subject == "level2" {
		result := WsLevel2Orderbook{}
		err = json.Unmarshal(respData, &result)
		response.Symbol = instrument
		response.Changes.Asks = result.Asks
		response.Changes.Bids = result.Bids
		response.TimeMS = result.TimeMS
	} else {
		err = json.Unmarshal(respData, &response)
	}
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
		LastUpdated:     time.UnixMilli(response.TimeMS),
		Pair:            pair,
		Asset:           asset.Spot,
	}
	for x := range response.Changes.Asks {
		item := orderbook.Item{}
		item.Price, err = strconv.ParseFloat(response.Changes.Asks[x][0], 64)
		if err != nil {
			return err
		}
		item.Amount, err = strconv.ParseFloat(response.Changes.Asks[x][1], 64)
		if err != nil {
			return err
		}
		if response.Changes.Asks[x][2] != "" {
			item.ID, err = strconv.ParseInt(response.Changes.Asks[x][2], 10, 64)
			if err != nil {
				return err
			}
		}
		base.Asks = append(base.Asks, item)
	}
	for x := range response.Changes.Bids {
		item := orderbook.Item{}
		item.Price, err = strconv.ParseFloat(response.Changes.Bids[x][0], 64)
		if err != nil {
			return err
		}
		item.Amount, err = strconv.ParseFloat(response.Changes.Bids[x][1], 64)
		if err != nil {
			return err
		}
		if response.Changes.Bids[x][2] != "" {
			item.ID, err = strconv.ParseInt(response.Changes.Bids[x][2], 10, 64)
			if err != nil {
				return err
			}
		}
		base.Bids = append(base.Bids, item)
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&base)
}

func (ku *Kucoin) processMarketSnapshot(respData []byte, instrument string) error {
	response := WsSpotTicker{}
	err := json.Unmarshal(respData, &(response))
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
		Last:         response.LastTradedPrice,
		Pair:         pair,
		Low:          response.Low,
		High:         response.High,
		QuoteVolume:  response.VolValue,
		Volume:       response.Vol,
		LastUpdated:  time.UnixMilli(response.Datetime),
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
	return errs
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
			futuresAccountBalanceEventChannel,
		)
	}
	for x := range channels {
		switch channels[x] {
		case accountBalanceChannel, marginPositionChannel, futuresTradeOrderChannel, futuresAccountBalanceEventChannel:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
			})
		case marketTickerSnapshotChannel:
			pairs, err := ku.GetEnabledPairs(asset.Spot)
			if err != nil {
				continue
			}
			for b := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  marketTickerSnapshotChannel,
					Asset:    asset.Spot,
					Currency: pairs[b],
				})
			}
		case marketOrderbokLevel2To50Channel,
			marketMatchChannel, marketTickerChannel:
			pairs, err := ku.GetEnabledPairs(asset.Spot)
			if err != nil {
				continue
			}
			pairStrings := pairs.Join()
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: marketOrderbokLevel2To50Channel,
				Asset:   asset.Spot,
				Params:  map[string]interface{}{"symbols": pairStrings},
			})
		case marketCandlesChannel:
			pairs, err := ku.GetEnabledPairs(asset.Spot)
			if err != nil {
				continue
			}
			for b := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  marketCandlesChannel,
					Asset:    asset.Spot,
					Currency: pairs[b],
					Params:   map[string]interface{}{"interval": kline.FifteenMin},
				})
			}
		case marginLoanChannel:
			currencyExist := map[currency.Code]bool{}
			pairs, err := ku.GetEnabledPairs(asset.Spot)
			if err != nil {
				continue
			}
			for b := range pairs {
				okay := currencyExist[pairs[b].Base]
				if !okay {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: currency.Pair{Base: pairs[b].Base},
					})
					currencyExist[pairs[b].Base] = true
				}
				okay = currencyExist[pairs[b].Quote]
				if !okay {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: currency.Pair{Base: pairs[b].Quote},
					})
					currencyExist[pairs[b].Quote] = true
				}
			}
		case marginFundingbookChangeChannel:
			currencyExist := map[currency.Code]bool{}
			pairs, err := ku.GetEnabledPairs(asset.Spot)
			if err != nil {
				continue
			}
			for b := range pairs {
				okay := currencyExist[pairs[b].Base]
				if !okay {
					currencyExist[pairs[b].Base] = true
				}
				okay = currencyExist[pairs[b].Quote]
				if !okay {
					currencyExist[pairs[b].Quote] = true
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
		case futuresTickerV2Channel:
			pairs, err := ku.GetEnabledPairs(asset.Futures)
			if err != nil {
				continue
			}
			for b := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  futuresTickerV2Channel,
					Asset:    asset.Futures,
					Currency: pairs[b],
				})
			}
		case futuresOrderbookLevel2Depth50Channel:
			pairs, err := ku.GetEnabledPairs(asset.Futures)
			if err != nil {
				continue
			}
			for b := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  futuresOrderbookLevel2Depth50Channel,
					Asset:    asset.Futures,
					Currency: pairs[b],
				})
			}

			// For authenticated subscriptions
		case futuresTradeOrdersBySymbolChannel:
			pairs, err := ku.GetEnabledPairs(asset.Futures)
			if err != nil {
				continue
			}
			for b := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  futuresTradeOrdersBySymbolChannel,
					Asset:    asset.Futures,
					Currency: pairs[b],
				})
			}
		}
	}
	return subscriptions, nil
}

func (ku *Kucoin) generatePayloads(subscriptions []stream.ChannelSubscription, operation string) ([]WsSubscriptionInput, error) {
	payloads := make([]WsSubscriptionInput, len(subscriptions))
	for x := range subscriptions {
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
				return nil, errors.New("symbols not passed")
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, symbols),
				Response: true,
			}
		case marketAllTickersChannel,
			privateChannel,
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
				futuresAccountBalanceEventChannel:
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
			symbol, err := ku.FormatSymbol(subscriptions[x].Currency, subscriptions[x].Asset)
			if err != nil {
				return nil, err
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, symbol),
				Response: true,
			}
			switch subscriptions[x].Channel {
			case futuresPositionChangeEventChannel,
				futuresTradeOrdersBySymbolChannel:
				payloads[x].PrivateChannel = true
			}
		case marketTickerSnapshotForCurrencyChannel,
			marginLoanChannel:
			if subscriptions[x].Channel == marketTickerSnapshotForCurrencyChannel {
				subscriptions[x].Channel += "%s"
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.Base.Upper().String()),
				Response: true,
			}
		case marketCandlesChannel:
			interval, err := ku.intervalToString(subscriptions[x].Params["interval"].(kline.Interval))
			if err != nil {
				return nil, err
			}
			payloads[x] = WsSubscriptionInput{
				ID:       strconv.FormatInt(ku.Websocket.Conn.GenerateMessageID(false), 10),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.Base.Upper().String(), interval),
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
