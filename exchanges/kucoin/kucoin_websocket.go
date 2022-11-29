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
	marketAllTickersChannel,
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
	return &(response.Data), ku.SendPayload(ctx, publicSpotRate, func() (*request.Item, error) {
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
	return &response.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, privateBullets, nil, publicSpotRate, &response)
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
	if resp.ID != "" && !ku.Websocket.Match.IncomingWithData(resp.ID, respData) {
		return fmt.Errorf("can't send ws incoming data to Matched channel with RequestID: %s", resp.ID)
	}

	topicInfo := strings.Split(resp.Topic, ":")
	switch {
	case strings.HasPrefix(marketAllTickersChannel, topicInfo[0]) ||
		strings.HasPrefix(marketTickerChannel, topicInfo[0]):
		instruments := ""
		if topicInfo[1] == "all" {
			instruments = resp.Subject
		} else {
			instruments = topicInfo[1]
		}
		return ku.processTicker(resp.Data, instruments)
	case strings.HasPrefix(marketTickerSnapshotChannel, topicInfo[0]) ||
		strings.HasPrefix(marketTickerSnapshotForCurrencyChannel, topicInfo[0]):
		return ku.processMarketSnapshot(resp.Data)
	case strings.HasPrefix(marketOrderbookLevel2Channels, topicInfo[0]),
		strings.HasPrefix(marketOrderbookLevel2to5Channel, topicInfo[0]),
		strings.HasPrefix(marketOrderbokLevel2To50Channel, topicInfo[0]):
		return ku.processOrderbook(resp.Data, topicInfo[1])
	case strings.HasPrefix(marketCandlesChannel, topicInfo[0]):
		symbolAndInterval := strings.Split(topicInfo[1], "_")
		if len(symbolAndInterval) != 2 {
			return errMalformedData
		}
		return ku.processCandlesticks(resp.Data, symbolAndInterval[0], symbolAndInterval[1])
	case strings.HasPrefix(marketMatchChannel, topicInfo[0]):
		return ku.processTradeData(resp.Data, topicInfo[1])
	case strings.HasPrefix(indexPriceIndicatorChannel, topicInfo[0]):
		return ku.pricessIndexPriceIndicator(resp.Data)
	case strings.HasPrefix(markPriceIndicatorChannel, topicInfo[0]):
		return ku.pricessMarkPriceIndicator(resp.Data)
	case strings.HasPrefix(marginFundingbookChangeChannel, topicInfo[0]):
		return ku.processMariginFundingBook(resp.Data)
	case strings.HasPrefix(privateChannel, topicInfo[0]):
		return ku.processOrderChangeEvent(resp.Data)
	case strings.HasPrefix(accountBalanceChannel, topicInfo[0]):
		return ku.processAccountBalanceChange(resp.Data)
	case strings.HasPrefix(marginPositionChannel, topicInfo[0]):
		if resp.Subject == "debt.ratio" {
			return ku.processDebtRatioChange(resp.Data)
		}
		return ku.processPositionStatus(resp.Data)
	case strings.HasPrefix(marginLoanChannel, topicInfo[0]) && resp.Subject == "order.done":
		return ku.processMarginLendingTradeOrderDoneEvent(resp.Data)
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
		return ku.processFuturesExecutionData(resp.Data)
	case strings.HasPrefix(futuresOrderbookLevel2Depth5Channel, topicInfo[0]),
		strings.HasPrefix(futuresOrderbookLevel2Depth50Channel, topicInfo[0]):
		return ku.processFuturesOrderbookLevel5(resp.Data, topicInfo[1])
	case strings.HasPrefix(futuresContractMarketDataChannel, topicInfo[0]):
	case strings.HasPrefix(futuresSystemAnnouncementChannel, topicInfo[0]):
	case strings.HasPrefix(futuresTrasactionStatisticsTimerEventChannel, topicInfo[0]):
	case strings.HasPrefix(futuresTradeOrdersBySymbolChannel, topicInfo[0]):
	case strings.HasPrefix(futuresTradeOrderChannel, topicInfo[0]):
	case strings.HasPrefix(futuresStopOrdersLifecycleEventChannel, topicInfo[0]):
	case strings.HasPrefix(futuresAccountBalanceEventChannel, topicInfo[0]):
	case strings.HasPrefix(futuresPositionChangeEventChannel, topicInfo[0]):
	default:
		ku.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: ku.Name + stream.UnhandledMessage + string(respData),
		}
		return errors.New("push data not handled")
	}
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
		Bids:            resp.Bids}
	return ku.Websocket.Orderbook.LoadSnapshot(&base)
}

func (ku *Kucoin) processFuturesExecutionData(respData []byte) error {
	resp := WsFuturesExecutionData{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
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
	if detail.Sequence != resp.Sequence {
		return errors.New("orderbook data sequence mismatch")
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
		Bids:            detail.Bids}
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
		// AverageExecutedPrice: response.,
		Exchange:    ku.Name,
		ID:          resp.OrderID,
		Type:        oType,
		Side:        side,
		AssetType:   asset.Spot,
		Date:        resp.CreatedAt,
		LastUpdated: resp.Timestamp,
		Pair:        pair,
	}
	return nil
}

func (ku *Kucoin) processMarginLendingTradeOrderDoneEvent(respData []byte) error {
	resp := WsMarginTradeOrderDoneEvent{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
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

func (ku *Kucoin) processPositionStatus(data []byte) error {
	resp := WsPositionStatus{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
}

func (ku *Kucoin) processDebtRatioChange(data []byte) error {
	resp := WsDebtRatioChange{}
	if err := json.Unmarshal(data, &resp); err != nil {
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
		Price:  response.Price,
		Amount: response.Size,
		// AverageExecutedPrice: response.,
		ExecutedAmount:  response.FilledSize,
		RemainingAmount: response.RemainSize,
		Exchange:        ku.Name,
		ID:              response.OrderID,
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

func (ku *Kucoin) processMariginFundingBook(respData []byte) error {
	resp := WsMarginFundingBook{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
}

func (ku *Kucoin) pricessMarkPriceIndicator(respData []byte) error {
	resp := WsPriceIndicator{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
	return nil
}
func (ku *Kucoin) pricessIndexPriceIndicator(respData []byte) error {
	resp := WsPriceIndicator{}
	err := json.Unmarshal(respData, &resp)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- resp
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

func (ku *Kucoin) processOrderbook(respData []byte, instrument string) error {
	response := WsOrderbook{}
	err := json.Unmarshal(respData, &response)
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
		price, err := strconv.ParseFloat(response.Changes.Asks[x][0], 64)
		if err != nil {
			return err
		}
		size, err := strconv.ParseFloat(response.Changes.Asks[x][1], 64)
		if err != nil {
			return err
		}
		sequence, err := strconv.ParseInt(response.Changes.Asks[x][2], 10, 64)
		base.Asks = append(base.Asks, orderbook.Item{
			Price:  price,
			Amount: size,
			ID:     sequence,
		})
	}
	for x := range response.Changes.Bids {
		price, err := strconv.ParseFloat(response.Changes.Bids[x][0], 64)
		if err != nil {
			return err
		}
		size, err := strconv.ParseFloat(response.Changes.Bids[x][1], 64)
		if err != nil {
			return err
		}
		sequence, _ := strconv.ParseInt(response.Changes.Bids[x][2], 10, 64)
		base.Bids = append(base.Bids, orderbook.Item{
			Price:  price,
			Amount: size,
			ID:     sequence})
	}
	return ku.Websocket.Orderbook.LoadSnapshot(&base)
}

func (ku *Kucoin) processMarketSnapshot(respData []byte) error {
	response := WsTickerDetail{}
	err := json.Unmarshal(respData, &response)
	if err != nil {
		return err
	}
	tickers := make([]ticker.Price, len(response.Data))
	for x := range response.Data {
		pair, err := currency.NewPairFromString(response.Data[x].Symbol)
		if err != nil {
			return err
		}
		tickers[x] = ticker.Price{
			ExchangeName: ku.Name,
			AssetType:    asset.Spot,
			Last:         response.Data[x].LastTradedPrice,
			Pair:         pair,
			// Open: response.Data.,
			// Close: response.Data.Close,
			Low:         response.Data[x].Low,
			High:        response.Data[x].High,
			QuoteVolume: response.Data[x].VolValue,
			Volume:      response.Data[x].Vol,
			LastUpdated: time.UnixMilli(response.Data[x].Datetime),
		}
	}
	ku.Websocket.DataHandler <- tickers
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (ku *Kucoin) Subscribe(subscriptions []stream.ChannelSubscription) error {
	payloads, err := ku.generatePayloads(subscriptions, "subscribe")
	if err != nil {
		return err
	}
	return ku.handleSubscriptions(payloads)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (ku *Kucoin) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	payloads, err := ku.generatePayloads(subscriptions, "unsubscribe")
	if err != nil {
		return err
	}
	return ku.handleSubscriptions(payloads)
}

func (ku *Kucoin) handleSubscriptions(payloads []WsSubscriptionInput) error {
	for x := range payloads {
		response, err := ku.Websocket.Conn.SendMessageReturnResponse(payloads[x].ID, payloads[x])
		if err != nil {
			return err
		}
		resp := WSSubscriptionResponse{}
		return json.Unmarshal(response, &resp)
	}
	return nil
}

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
	subscriptions = append(subscriptions, stream.ChannelSubscription{
		Channel: marketAllTickersChannel,
	})
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
			marketMatchChannel:
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
				Type:     operation,
				Topic:    subscriptions[x].Channel,
				Response: true,
			}
			switch subscriptions[x].Channel {
			case marketAllTickersChannel,
				futuresTradeOrderChannel,
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
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
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, currencies),
				Response: true,
			}
		}
	}
	return payloads, nil
}
