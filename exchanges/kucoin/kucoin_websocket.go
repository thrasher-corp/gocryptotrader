package kucoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

const (
	publicBullets  = "/v1/bullet-public"
	privateBullets = "/v1/bullet-private"

	// channels

	channelPing                            = "ping"
	channelPong                            = "pong"
	typeWelcome                            = "welcome"
	marketTickerChannel                    = "/market/ticker"
	marketAllTickersChannel                = "/market/ticker:all"
	marketTickerSnapshotChannel            = "/market/snapshot:%s"
	marketTickerSnapshotForCurrencyChannel = "/market/snapshot:%s"       // /market/snapshot:{market}
	marketOrderbookLevel2Channels          = "/market/level2"            // /market/level2:{symbol},{symbol}...
	marketLevel2to5OrderbookChannel        = "/spotMarket/level2Depth5"  // /spotMarket/level2Depth5:{symbol},{symbol}...
	marketLevel2oTo50OrderbokChannel       = "/spotMarket/level2Depth50" // /spotMarket/level2Depth50:{symbol},{symbol}...
	marketCandlesChannel                   = "/market/candles:%s"        // /market/candles:{symbol}_{type}
	indexPriceIndicatorChannel             = "/indicator/index"          // /indicator/index:{symbol0},{symbol1}..
	markPriceIndicatorChannel              = "/indicator/markPrice:%s"   // /indicator/markPrice:{symbol0},{symbol1}...
	orderbookChangeChannel                 = "/margin/fundingBook"       // /margin/fundingBook:{currency0},{currency1}...

	// Private channel

	privateChannel            = "/spotMarket/tradeOrders"
	accountBalanceChannel     = "/account/balance"
	marginPositionChannel     = "/margin/position"
	marginLoanChannel         = "/margin/loan:%s" // /margin/loan:{currency}
	spotMarketAdvancedChannel = "/spotMarket/advancedOrders"
)

var defaultSubscriptionChannels = []string{}

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
	return nil
}

func (ku *Kucoin) Subscribe([]stream.ChannelSubscription) error {
	return nil
}

func (ku *Kucoin) Unsubscribe([]stream.ChannelSubscription) error {
	return nil
}

func (ku *Kucoin) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	return nil, nil
}
