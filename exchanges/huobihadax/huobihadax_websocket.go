package huobihadax

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	huobiGlobalWebsocketEndpoint         = "wss://api.huobi.pro/ws"
	huobiGlobalAssetWebsocketEndpoint    = "wss://api.huobi.pro/ws/v1"
	huobiGlobalContractWebsocketEndpoint = "wss://www.hbdm.com/ws"
	wsMarketKline                        = "market.%s.kline.1min"
	wsMarketDepth                        = "market.%s.depth.step0"
	wsMarketTrade                        = "market.%s.trade.detail"
)

// WsConnect initiates a new websocket connection
func (h *HUOBIHADAX) WsConnect() error {
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
	h.WebsocketConn, _, err = dialer.Dial(h.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return err
	}

	go h.WsHandleData()

	h.GenerateDefaultSubscriptions()
	return nil
}

// WsReadData reads data from the websocket connection
func (h *HUOBIHADAX) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := h.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	h.Websocket.TrafficAlert <- struct{}{}

	b := bytes.NewReader(resp)
	gReader, err := gzip.NewReader(b)
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	unzipped, err := ioutil.ReadAll(gReader)
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}
	gReader.Close()

	return exchange.WebsocketResponse{Raw: unzipped}, nil
}

// WsHandleData handles data read from the websocket connection
func (h *HUOBIHADAX) WsHandleData() {
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

			var init WsResponse
			err = common.JSONDecode(resp.Raw, &init)
			if err != nil {
				h.Websocket.DataHandler <- err
				continue
			}

			if init.Status == "error" {
				h.Websocket.DataHandler <- fmt.Errorf("huobi.go Websocker error %s %s",
					init.ErrorCode,
					init.ErrorMessage)
				continue
			}

			if init.Subscribed != "" {
				continue
			}

			if init.Ping != 0 {
				err = h.WebsocketConn.WriteJSON(`{"pong":1337}`)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}
				continue
			}

			switch {
			case common.StringContains(init.Channel, "depth"):
				var depth WsDepth
				err := common.JSONDecode(resp.Raw, &depth)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				data := common.SplitStrings(depth.Channel, ".")

				h.WsProcessOrderbook(&depth, data[1])

			case common.StringContains(init.Channel, "kline"):
				var kline WsKline
				err := common.JSONDecode(resp.Raw, &kline)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				data := common.SplitStrings(kline.Channel, ".")

				h.Websocket.DataHandler <- exchange.KlineData{
					Timestamp:  time.Unix(0, kline.Timestamp),
					Exchange:   h.GetName(),
					AssetType:  assets.AssetTypeSpot,
					Pair:       currency.NewPairFromString(data[1]),
					OpenPrice:  kline.Tick.Open,
					ClosePrice: kline.Tick.Close,
					HighPrice:  kline.Tick.High,
					LowPrice:   kline.Tick.Low,
					Volume:     kline.Tick.Volume,
				}

			case common.StringContains(init.Channel, "trade"):
				var trade WsTrade
				err := common.JSONDecode(resp.Raw, &trade)
				if err != nil {
					h.Websocket.DataHandler <- err
					continue
				}

				data := common.SplitStrings(trade.Channel, ".")

				h.Websocket.DataHandler <- exchange.TradeData{
					Exchange:     h.GetName(),
					AssetType:    assets.AssetTypeSpot,
					CurrencyPair: currency.NewPairFromString(data[1]),
					Timestamp:    time.Unix(0, trade.Tick.Timestamp),
				}
			}
		}
	}
}

// WsProcessOrderbook processes new orderbook data
func (h *HUOBIHADAX) WsProcessOrderbook(ob *WsDepth, symbol string) error {
	var bids []orderbook.Item
	for _, data := range ob.Tick.Bids {
		bidLevel := data.([]interface{})
		bids = append(bids, orderbook.Item{Price: bidLevel[0].(float64),
			Amount: bidLevel[0].(float64)})
	}

	var asks []orderbook.Item
	for _, data := range ob.Tick.Asks {
		askLevel := data.([]interface{})
		asks = append(asks, orderbook.Item{Price: askLevel[0].(float64),
			Amount: askLevel[0].(float64)})
	}

	p := currency.NewPairFromString(symbol)

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.Pair = p

	err := h.Websocket.Orderbook.LoadSnapshot(&newOrderBook, h.GetName(), false)
	if err != nil {
		return err
	}

	h.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Exchange: h.GetName(),
		Asset:    assets.AssetTypeSpot,
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (h *HUOBIHADAX) GenerateDefaultSubscriptions() {
	var channels = []string{wsMarketKline, wsMarketDepth, wsMarketTrade}
	enabledCurrencies := h.GetEnabledPairs(assets.AssetTypeSpot)
	subscriptions := []exchange.WebsocketChannelSubscription{}
	for i := range channels {
		for j := range enabledCurrencies {
			enabledCurrencies[j].Delimiter = ""
			channel := fmt.Sprintf(channels[i], enabledCurrencies[j].Lower().String())
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  channel,
				Currency: enabledCurrencies[j],
			})
		}
	}
	h.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (h *HUOBIHADAX) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscription, err := common.JSONEncode(WsRequest{Subscribe: channelToSubscribe.Channel})
	if err != nil {
		return err
	}
	return h.wsSend(subscription)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (h *HUOBIHADAX) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscription, err := common.JSONEncode(WsRequest{Unsubscribe: channelToSubscribe.Channel})
	if err != nil {
		return err
	}
	return h.wsSend(subscription)
}

// WsSend sends data to the websocket server
func (h *HUOBIHADAX) wsSend(data []byte) error {
	h.wsRequestMtx.Lock()
	defer h.wsRequestMtx.Unlock()
	if h.Verbose {
		log.Debugf("%v sending message to websocket %s", h.Name, string(data))
	}
	return h.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}
