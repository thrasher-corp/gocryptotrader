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
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

	channelPing                            = "ping"
	channelPong                            = "pong"
	typeWelcome                            = "welcome"
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
)

var defaultSubscriptionChannels = []string{
	marketAllTickersChannel,
	marketTickerSnapshotForCurrencyChannel,
	marketOrderbokLevel2To50Channel,
	marginFundingbookChangeChannel,
	marketOrderbokLevel2To50Channel,
	marketCandlesChannel,
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
	ku.Websocket.Wg.Add(1)
	ku.Websocket.Wg = &sync.WaitGroup{}
	println(string(pingMessage))
	ku.Websocket.Wg.Add(1)
	// ku.Websocket.Conn.SetupPingHandler(stream.PingHandler{
	// 	Delay:       time.Millisecond * time.Duration(instances.InstanceServers[0].PingTimeout),
	// 	Message:     pingMessage,
	// 	MessageType: websocket.TextMessage,
	// })
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
	if resp.Type == "message" {
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
		case true: //privateChannel:
		case true: //accountBalanceChannel:
		case true: //marginPositionChannel:
		case true: //marginLoanChannel:
		case true: //spotMarketAdvancedChannel:
		}
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
		// AssetType: asset.Futures,
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
		Timestamp: time.UnixMilli(response.Time),
		Pair:      pair,
		// AssetType: asset.Futures,
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
	pair, err := currency.NewPairFromString(response.Data.Symbol)
	if err != nil {
		return err
	}
	ku.Websocket.DataHandler <- &ticker.Price{
		ExchangeName: ku.Name,
		// AssetType:    asset.Futures,
		Last: response.Data.LastTradedPrice,
		Pair: pair,
		// Open: response.Data.,
		// Close: response.Data.Close,
		Low:         response.Data.Low,
		High:        response.Data.High,
		QuoteVolume: response.Data.VolValue,
		Volume:      response.Data.Vol,
		LastUpdated: time.UnixMilli(response.Data.Datetime),
	}
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
			marginLoanChannel)
	}
	subscriptions = append(subscriptions, stream.ChannelSubscription{
		Channel: marketAllTickersChannel,
	})
	assets := ku.GetAssetTypes(false)
	for x := range channels {
		switch channels[x] {
		case accountBalanceChannel, marginPositionChannel:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
			})
		case marketTickerSnapshotChannel:
			for a := range assets {
				pairs, err := ku.GetEnabledPairs(assets[a])
				if err != nil {
					continue
				}
				for b := range pairs {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  marketTickerSnapshotChannel,
						Asset:    assets[a],
						Currency: pairs[b],
					})
				}
			}
		case marketOrderbokLevel2To50Channel,
			marketMatchChannel:
			for a := range assets {
				pairs, err := ku.GetEnabledPairs(assets[a])
				if err != nil {
					continue
				}
				pairStrings := pairs.Join()
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel: marketOrderbokLevel2To50Channel,
					Asset:   assets[a],
					Params:  map[string]interface{}{"symbols": pairStrings},
				})
			}
		case marketCandlesChannel:
			for a := range assets {
				pairs, err := ku.GetEnabledPairs(assets[a])
				if err != nil {
					continue
				}
				for b := range pairs {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  marketCandlesChannel,
						Asset:    assets[a],
						Currency: pairs[b],
						Params:   map[string]interface{}{"interval": kline.FifteenMin},
					})
				}
			}
		case marginLoanChannel:
			currencyExist := map[currency.Code]bool{}
			for a := range assets {
				pairs, err := ku.GetEnabledPairs(assets[a])
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
			}
		case marginFundingbookChangeChannel:
			currencyExist := map[currency.Code]bool{}
			for a := range assets {
				pairs, err := ku.GetEnabledPairs(assets[a])
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
			spotMarketAdvancedChannel:
			input := WsSubscriptionInput{
				ID:       ku.Websocket.AuthConn.GenerateMessageID(false),
				Type:     operation,
				Topic:    fmt.Sprintf(subscriptions[x].Channel, subscriptions[x].Currency.String()),
				Response: true,
			}
			if subscriptions[x].Channel != marketAllTickersChannel {
				input.PrivateChannel = true
			}
			payloads[x] = input
		case marketTickerSnapshotChannel: // Symbols
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
