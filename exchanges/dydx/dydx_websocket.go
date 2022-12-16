package dydx

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (

	// channels

	accountsChannel  = "v3_accounts"
	orderbookChannel = "v3_orderbook"
	tradesChannel    = "v3_trades"
	marketsChannel   = "v3_markets"
)

var defaultSubscriptions = []string{
	orderbookChannel,
	tradesChannel,
	marketsChannel,
}

// WsConnect connect to dydx websocket server.
func (dy *DYDX) WsConnect() error {
	if !dy.Websocket.IsEnabled() || !dy.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := dy.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if dy.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", dy.Name)
	}
	dy.Websocket.Wg.Add(1)
	go dy.wsReadData()
	return nil
}

func (dy *DYDX) wsReadData() {
	defer dy.Websocket.Wg.Done()
	for {
		resp := dy.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := dy.wsHandleData(resp.Raw)
		if err != nil {
			dy.Websocket.DataHandler <- err
		}
	}
}

func (dy *DYDX) wsHandleData(respRaw []byte) error {
	println(string(respRaw))
	var resp WsResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	switch resp.Channel {
	case accountsChannel:

	case orderbookChannel:
		var market MarketOrderbook
		err := json.Unmarshal(respRaw, &market)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(resp.ID)
		if err != nil {
			return err
		}
		newOrderbook := orderbook.Base{
			Asset:       asset.Spot,
			Asks:        market.Asks.generateOrderbookItem(),
			Bids:        market.Bids.generateOrderbookItem(),
			Exchange:    dy.Name,
			Pair:        pair,
			LastUpdated: time.Now(),
		}
		err = dy.Websocket.Orderbook.LoadSnapshot(&newOrderbook)
		if err != nil {
			return err
		}
		return nil
	case tradesChannel:
		var myTrades MarketTrades
		err := json.Unmarshal(resp.Contents, &myTrades)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(resp.ID)
		if err != nil {
			return err
		}
		trades := make([]trade.Data, len(myTrades.Trades))
		for i := range myTrades.Trades {
			side, err := order.StringToOrderSide(myTrades.Trades[i].Side)
			if err != nil {
				return err
			}
			trades[i] = trade.Data{
				Amount:       myTrades.Trades[i].Size,
				AssetType:    asset.Spot,
				CurrencyPair: pair,
				Exchange:     dy.Name,
				Side:         side,
				Timestamp:    time.Now(),
				Price:        myTrades.Trades[i].Price,
			}
		}
		return trade.AddTradesToBuffer(dy.Name, trades...)
	case marketsChannel:
		var market InstrumentDatas
		err := json.Unmarshal(resp.Contents, &market)
		if err != nil {
			return err
		}
		tickers := make([]ticker.Price, len(market.Markets))
		count := 0
		for x, _ := range market.Markets {
			pair, err := currency.NewPairFromString(x)
			if err != nil {
				return err
			}
			tickers[count] = ticker.Price{
				ExchangeName: dy.Name,
				Ask:          market.Markets[x].IndexPrice,
				Pair:         pair,
				AssetType:    asset.Spot,
			}
			count++
		}
		dy.Websocket.DataHandler <- tickers
	default:
		dy.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: dy.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (dy *DYDX) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultSubscriptions
	if dy.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, accountsChannel)
	}
	subscriptions := []stream.ChannelSubscription{}
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for a := range enabledPairs {
		for x := range channels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[x],
				Currency: enabledPairs[a],
			})
		}
	}
	return subscriptions, nil
}

func (dy *DYDX) generateSubscriptionPayload(subscriptions []stream.ChannelSubscription, operation string) []WsInput {
	payloads := make([]WsInput, len(subscriptions))
	for x := range subscriptions {
		payloads[x] = WsInput{
			Type:    operation,
			Channel: subscriptions[x].Channel,
			ID:      subscriptions[x].Currency.String(),
		}
	}
	return payloads
}

func (dy *DYDX) handleSubscriptions(subscriptions []stream.ChannelSubscription, operation string) error {
	payloads := dy.generateSubscriptionPayload(subscriptions, operation)
	var errs common.Errors
	for x := range payloads {
		err := dy.Websocket.Conn.SendJSONMessage(payloads[x])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		dy.Websocket.AddSuccessfulSubscriptions(subscriptions[x])
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Subscribe sends a subscriptions requests through the websocket connection.
func (dy *DYDX) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return dy.handleSubscriptions(subscriptions, "subscribe")
}

// Unsubscribe sends unsubscription to channels through the websocket connection.
func (dy *DYDX) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return dy.handleSubscriptions(subscriptions, "unsubscribe")
}
