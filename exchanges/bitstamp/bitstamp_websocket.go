package bitstamp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net"
)

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if b.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	b.WebsocketConn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	if b.Verbose {
		log.Debugf("%s Connected to Websocket.\n", b.GetName())
	}

	err = b.seedOrderBook()
	if err != nil {
		b.Websocket.DataHandler <- err
	}

	b.generateDefaultSubscriptions()
	go b.WsHandleData()

	return nil
}

// WsReadData reads data coming from bitstamp websocket connection
func (b *Bitstamp) WsReadData() (exchange.WebsocketResponse, error) {
	msgType, resp, err := b.WebsocketConn.ReadMessage()

	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	if b.Verbose {
		log.Debugf("%s websocket raw response: %s", b.GetName(), resp)
	}

	b.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Type: msgType, Raw: resp}, nil
}

// WsHandleData handles websocket data from WsReadData
func (b *Bitstamp) WsHandleData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			resp, err := b.WsReadData()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}

			wsResponse := websocketResponse{}
			err = common.JSONDecode(resp.Raw, &wsResponse)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			switch wsResponse.Event {
			case "bts:request_reconnect":
				if b.Verbose {
					log.Debugf("%v - Websocket reconnection request received", b.GetName())
				}
				go b.Websocket.WebsocketReset()

			case "data":
				wsOrderBookTemp := websocketOrderBookResponse{}
				err := common.JSONDecode(resp.Raw, &wsOrderBookTemp)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				currencyPair := common.SplitStrings(wsResponse.Channel, "_")
				p := currency.NewPairFromString(common.StringToUpper(currencyPair[3]))

				err = b.wsUpdateOrderbook(wsOrderBookTemp.Data, p, ticker.Spot)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

			case "trade":
				wsTradeTemp := websocketTradeResponse{}

				err := common.JSONDecode(resp.Raw, &wsTradeTemp)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				currencyPair := common.SplitStrings(wsResponse.Channel, "_")
				p := currency.NewPairFromString(common.StringToUpper(currencyPair[2]))

				b.Websocket.DataHandler <- exchange.TradeData{
					Price:        wsTradeTemp.Data.Price,
					Amount:       wsTradeTemp.Data.Amount,
					CurrencyPair: p,
					Exchange:     b.GetName(),
					AssetType:    ticker.Spot,
				}
			}
		}
	}
}

func (b *Bitstamp) generateDefaultSubscriptions() {
	var channels = []string{"live_trades_", "diff_order_book_"}
	enabledCurrencies := b.GetEnabledCurrencies()
	subscriptions := []exchange.WebsocketChannelSubscription{}
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel: fmt.Sprintf("%v%v", channels[i], enabledCurrencies[j].Lower().String()),
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *Bitstamp) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()

	req := websocketEventRequest{
		Event: "bts:subscribe",
		Data: websocketData{
			Channel: channelToSubscribe.Channel,
		},
	}
	return b.WebsocketConn.WriteJSON(req)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *Bitstamp) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()

	req := websocketEventRequest{
		Event: "bts:unsubscribe",
		Data: websocketData{
			Channel: channelToSubscribe.Channel,
		},
	}
	return b.WebsocketConn.WriteJSON(req)
}

func (b *Bitstamp) wsUpdateOrderbook(ob websocketOrderBook, p currency.Pair, assetType string) error {
	if len(ob.Asks) == 0 && len(ob.Bids) == 0 {
		return errors.New("bitstamp_websocket.go error - no orderbook data")
	}

	var asks, bids []orderbook.Item

	if len(ob.Asks) > 0 {
		for _, ask := range ob.Asks {
			target, err := strconv.ParseFloat(ask[0], 64)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			amount, err := strconv.ParseFloat(ask[1], 64)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			asks = append(asks, orderbook.Item{Price: target, Amount: amount})
		}
	}

	if len(ob.Bids) > 0 {
		for _, bid := range ob.Bids {
			target, err := strconv.ParseFloat(bid[0], 64)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			amount, err := strconv.ParseFloat(bid[1], 64)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			bids = append(bids, orderbook.Item{Price: target, Amount: amount})
		}
	}

	err := b.Websocket.Orderbook.Update(bids, asks, p, time.Now(), b.GetName(), assetType)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    assetType,
		Exchange: b.GetName(),
	}

	return nil
}

func (b *Bitstamp) seedOrderBook() error {
	p := b.GetEnabledCurrencies()
	for x := range p {
		orderbookSeed, err := b.GetOrderbook(p[x].String())
		if err != nil {
			return err
		}

		var newOrderBook orderbook.Base
		var asks, bids []orderbook.Item

		for _, ask := range orderbookSeed.Asks {
			var item orderbook.Item
			item.Amount = ask.Amount
			item.Price = ask.Price
			asks = append(asks, item)
		}

		for _, bid := range orderbookSeed.Bids {
			var item orderbook.Item
			item.Amount = bid.Amount
			item.Price = bid.Price
			bids = append(bids, item)
		}

		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.Pair = p[x]
		newOrderBook.AssetType = ticker.Spot

		err = b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, b.GetName(), false)
		if err != nil {
			return err
		}

		b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Pair:     p[x],
			Asset:    ticker.Spot,
			Exchange: b.GetName(),
		}
	}
	return nil
}
