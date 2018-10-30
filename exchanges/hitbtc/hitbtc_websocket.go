package hitbtc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	hitbtcWebsocketAddress = "wss://api.hitbtc.com/api/2/ws"
	rpcVersion             = "2.0"
)

// WsConnect starts a new connection with the websocket API
func (h *HitBTC) WsConnect() error {
	if !h.Websocket.IsEnabled() || !h.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer

	if h.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(h.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	h.WebsocketConn, _, err = dialer.Dial(hitbtcWebsocketAddress, http.Header{})
	if err != nil {
		return err
	}

	go h.WsHandleData()
	h.GenerateDefaultSubscriptions()

	return nil
}

// WsReadData reads from the websocket connection
func (h *HitBTC) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := h.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	h.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

// WsHandleData handles websocket data
func (h *HitBTC) WsHandleData() {
	h.Websocket.Wg.Add(1)

	defer func() {
		h.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-h.Websocket.ShutdownC:
			return

		default:
			resp, err := h.WsReadData()
			if err != nil {
				h.Websocket.DataHandler <- err
				return
			}

			var init capture
			err = common.JSONDecode(resp.Raw, &init)
			if err != nil {
				h.Websocket.DataHandler <- err
				continue
			}

			if init.Error.Message != "" || init.Error.Code != 0 {
				h.Websocket.DataHandler <- fmt.Errorf("hitbtc.go error - Code: %d, Message: %s",
					init.Error.Code,
					init.Error.Message)
				continue
			}

			if init.Result {
				continue
			}

			switch init.Method {
			case "ticker":
				var ticker WsTicker
				err := common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				ts, err := time.Parse(time.RFC3339, ticker.Params.Timestamp)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				h.Websocket.DataHandler <- exchange.TickerData{
					Exchange:  h.GetName(),
					AssetType: assets.AssetTypeSpot,
					Pair:      currency.NewPairFromString(ticker.Params.Symbol),
					Quantity:  ticker.Params.Volume,
					Timestamp: ts,
					OpenPrice: ticker.Params.Open,
					HighPrice: ticker.Params.High,
					LowPrice:  ticker.Params.Low,
				}

			case "snapshotOrderbook":
				var obSnapshot WsOrderbook
				err := common.JSONDecode(resp.Raw, &obSnapshot)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				err = h.WsProcessOrderbookSnapshot(obSnapshot)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

			case "updateOrderbook":
				var obUpdate WsOrderbook
				err := common.JSONDecode(resp.Raw, &obUpdate)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				h.WsProcessOrderbookUpdate(obUpdate)

			case "snapshotTrades":
				var tradeSnapshot WsTrade
				err := common.JSONDecode(resp.Raw, &tradeSnapshot)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

			case "updateTrades":
				var tradeUpdates WsTrade
				err := common.JSONDecode(resp.Raw, &tradeUpdates)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}
			}
		}
	}
}

// WsProcessOrderbookSnapshot processes a full orderbook snapshot to a local cache
func (h *HitBTC) WsProcessOrderbookSnapshot(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 || len(ob.Params.Ask) == 0 {
		return errors.New("hitbtc.go error - no orderbooks to process")
	}

	var bids []orderbook.Item
	for _, bid := range ob.Params.Bid {
		bids = append(bids, orderbook.Item{Amount: bid.Size, Price: bid.Price})
	}

	var asks []orderbook.Item
	for _, ask := range ob.Params.Ask {
		asks = append(asks, orderbook.Item{Amount: ask.Size, Price: ask.Price})
	}

	p := currency.NewPairFromString(ob.Params.Symbol)

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.AssetType = assets.AssetTypeSpot
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.Pair = p

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, h.GetName(), false)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     p,
	}

	return nil
}

// WsProcessOrderbookUpdate updates a local cache
func (h *HitBTC) WsProcessOrderbookUpdate(ob WsOrderbook) error {
	if len(ob.Params.Bid) == 0 && len(ob.Params.Ask) == 0 {
		return errors.New("hitbtc_websocket.go error - no data")
	}

	var bids, asks []orderbook.Item
	for _, bid := range ob.Params.Bid {
		bids = append(bids, orderbook.Item{Price: bid.Price, Amount: bid.Size})
	}

	for _, ask := range ob.Params.Ask {
		asks = append(asks, orderbook.Item{Price: ask.Price, Amount: ask.Size})
	}

	p := currency.NewPairFromString(ob.Params.Symbol)

	err := h.Websocket.Orderbook.Update(bids, asks, p, time.Now(), h.GetName(), assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: h.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     p,
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HitBTC) GenerateDefaultSubscriptions() {
	var channels = []string{"subscribeTicker", "subscribeOrderbook", "subscribeTrades", "subscribeCandles"}
	subscriptions := []exchange.WebsocketChannelSubscription{}
	enabledCurrencies := h.GetEnabledPairs(assets.AssetTypeSpot)
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HitBTC) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscribe := WsNotification{
		JSONRPCVersion: rpcVersion,
		Method:         channelToSubscribe.Channel,
		Params: params{
			Symbol: channelToSubscribe.Currency.String(),
		},
	}
	if strings.EqualFold(channelToSubscribe.Channel, "subscribeTrades") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Limit:  100,
		}
	} else if strings.EqualFold(channelToSubscribe.Channel, "subscribeCandles") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Period: "M30",
			Limit:  100,
		}
	}

	return h.wsSend(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HitBTC) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	unsubscribeChannel := strings.Replace(channelToSubscribe.Channel, "subscribe", "unsubscribe", 1)
	subscribe := WsNotification{
		JSONRPCVersion: rpcVersion,
		Method:         unsubscribeChannel,
		Params: params{
			Symbol: channelToSubscribe.Currency.String(),
		},
	}
	if strings.EqualFold(unsubscribeChannel, "unsubscribeTrades") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Limit:  100,
		}
	} else if strings.EqualFold(unsubscribeChannel, "unsubscribeCandles") {
		subscribe.Params = params{
			Symbol: channelToSubscribe.Currency.String(),
			Period: "M30",
			Limit:  100,
		}
	}

	return h.wsSend(subscribe)
}

// WsSend sends data to the websocket server
func (h *HitBTC) wsSend(data interface{}) error {
	h.wsRequestMtx.Lock()
	defer h.wsRequestMtx.Unlock()
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	if h.Verbose {
		log.Debugf("%v sending message to websocket %v", h.Name, data)
	}
	return h.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}
