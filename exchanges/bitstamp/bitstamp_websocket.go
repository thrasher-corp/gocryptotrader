package bitstamp

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/toorop/go-pusher"
)

var orderbookCache []orderbook.Base
var mtx sync.Mutex

// WebsocketConn defins a pusher websocket connection
type WebsocketConn struct {
	Client *pusher.Client
	Data   chan *pusher.Event
	Trade  chan *pusher.Event
}

// PusherOrderbook holds order book information to be pushed
type PusherOrderbook struct {
	Asks      [][]string `json:"asks"`
	Bids      [][]string `json:"bids"`
	Timestamp int64      `json:"timestamp,string"`
}

// PusherTrade holds trade information to be pushed
type PusherTrade struct {
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
	ID          int64   `json:"id"`
	Type        int64   `json:"type"`
	Timestamp   int64   `json:"timestamp,string"`
	BuyOrderID  int64   `json:"buy_order_id"`
	SellOrderID int64   `json:"sell_order_id"`
}

// PusherOrders defines order information
type PusherOrders struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount"`
	Price  float64 `json:""`
}

const (
	// BitstampPusherKey holds the current pusher key
	BitstampPusherKey = "de504dc5763aeef9ff52"
)

var tradingPairs map[string]string

// findPairFromChannel extracts the capitalized trading pair from the channel and returns it only if enabled in the config
func (b *Bitstamp) findPairFromChannel(channelName string) (string, error) {
	split := strings.Split(channelName, "_")
	tradingPair := strings.ToUpper(split[len(split)-1])

	for _, enabledPair := range b.EnabledPairs {
		if enabledPair == tradingPair {
			return tradingPair, nil
		}
	}

	return "", errors.New("Could not find trading pair")
}

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	tradingPairs = make(map[string]string)
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		log.Fatal("Bitstamp - set proxy address error: proxy not supported")
	}

	b.WebsocketConn.Client, err = pusher.NewClient(BitstampPusherKey)
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			b.GetName(),
			err)
	}

	b.WebsocketConn.Data, err = b.WebsocketConn.Client.Bind("data")
	if err != nil {
		return fmt.Errorf("%s Websocket Bind error: %s", b.GetName(), err)

	}

	b.WebsocketConn.Trade, err = b.WebsocketConn.Client.Bind("trade")
	if err != nil {
		return fmt.Errorf("%s Websocket Bind error: %s", b.GetName(), err)
	}

	for _, p := range b.EnabledPairs {
		orderbookSeed, err := b.GetOrderbook(p)
		if err != nil {
			return err
		}

		// re-use memory on disconnect reconnect and flush bid and asks
		var flushed bool
		for i := range orderbookCache {
			if orderbookCache[i].CurrencyPair == p {
				var localOrderbook orderbook.Base

				var asks []orderbook.Item
				for _, ask := range orderbookSeed.Asks {
					var item orderbook.Item
					item.Amount = ask.Amount
					item.Price = ask.Price
					asks = append(asks, item)
				}

				var bids []orderbook.Item
				for _, bid := range orderbookSeed.Bids {
					var item orderbook.Item
					item.Amount = bid.Amount
					item.Price = bid.Price
					bids = append(bids, item)
				}

				localOrderbook.Asks = asks
				localOrderbook.Bids = bids
				localOrderbook.CurrencyPair = p
				localOrderbook.Pair = pair.NewCurrencyPairFromString(p)
				localOrderbook.LastUpdated = time.Unix(0, orderbookSeed.Timestamp)
				localOrderbook.AssetType = "SPOT"

				orderbookCache = append(orderbookCache, localOrderbook)
				flushed = true
			}
		}

		if !flushed {
			var localOrderbook orderbook.Base

			var asks []orderbook.Item
			for _, ask := range orderbookSeed.Asks {
				var item orderbook.Item
				item.Amount = ask.Amount
				item.Price = ask.Price
				asks = append(asks, item)
			}

			var bids []orderbook.Item
			for _, bid := range orderbookSeed.Bids {
				var item orderbook.Item
				item.Amount = bid.Amount
				item.Price = bid.Price
				bids = append(bids, item)
			}

			localOrderbook.Asks = asks
			localOrderbook.Bids = bids
			localOrderbook.CurrencyPair = p
			localOrderbook.Pair = pair.NewCurrencyPairFromString(p)
			localOrderbook.LastUpdated = time.Unix(0, orderbookSeed.Timestamp)
			localOrderbook.AssetType = "SPOT"

			orderbookCache = append(orderbookCache, localOrderbook)
		}

		err = b.WebsocketConn.Client.Subscribe(fmt.Sprintf("live_trades_%s",
			strings.ToLower(p)))

		if err != nil {
			return fmt.Errorf("%s Websocket Trade subscription error: %s",
				b.GetName(),
				err)
		}

		err = b.WebsocketConn.Client.Subscribe(fmt.Sprintf("diff_order_book_%s",
			strings.ToLower(p)))
		if err != nil {
			return fmt.Errorf("%s Websocket Trade subscription error: %s",
				b.GetName(),
				err)
		}
	}

	go b.WsReadData()

	return nil
}

// WsReadData reads data coming from bitstamp websocket connection
func (b *Bitstamp) WsReadData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case data := <-b.WebsocketConn.Data:
			b.Websocket.TrafficTimer.Reset(exchange.WebsocketTrafficLimitTime)

			result := PusherOrderbook{}
			err := common.JSONDecode([]byte(data.Data), &result)
			if err != nil {
				log.Fatal(err)
			}

			currencyPair := common.SplitStrings(data.Channel, "_")

			go b.WsUpdateOrderbook(result, currencyPair[3])

		case trade := <-b.WebsocketConn.Trade:
			b.Websocket.TrafficTimer.Reset(exchange.WebsocketTrafficLimitTime)

			result := PusherTrade{}
			err := common.JSONDecode([]byte(trade.Data), &result)
			if err != nil {
				log.Fatal(err)
			}

			currencyPair := common.SplitStrings(trade.Channel, "_")

			b.Websocket.DataHandler <- exchange.TradeData{
				Price:        result.Price,
				Amount:       result.Amount,
				CurrencyPair: pair.NewCurrencyPairFromString(currencyPair[2]),
				Exchange:     b.GetName(),
				AssetType:    "SPOT",
			}
		}
	}
}

// WsShutdown shuts down websocket connection
func (b *Bitstamp) WsShutdown() error {
	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(b.Websocket.ShutdownC)
		b.Websocket.Wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-timer.C:
		return errors.New("bitstamp.go error - failed to shut down routines")
	case <-c:
		close(b.WebsocketConn.Data)
		close(b.WebsocketConn.Trade)
		return b.WebsocketConn.Client.Close()
	}
}

// WsUpdateOrderbook updates local cache of orderbook information
func (b *Bitstamp) WsUpdateOrderbook(ob PusherOrderbook, pair string) {
	mtx.Lock()
	defer mtx.Unlock()

	for i := range orderbookCache {
		if orderbookCache[i].CurrencyPair == pair {
			if len(ob.Asks) > 0 {
				for _, ask := range ob.Asks {
					target, err := strconv.ParseFloat(ask[0], 64)
					if err != nil {
						log.Fatal(err)
					}

					VolumeAdjust, err := strconv.ParseFloat(ask[1], 64)
					if err != nil {
						log.Fatal(err)
					}

					var found bool
					for x := range orderbookCache[i].Asks {
						if orderbookCache[i].Asks[x].Price == target {
							found = true
							if VolumeAdjust == 0 {
								orderbookCache[i].Asks = append(orderbookCache[i].Asks[:x],
									orderbookCache[i].Asks[x+1:]...)
								break
							}
							orderbookCache[i].Asks[x].Amount = VolumeAdjust
							break
						}
					}

					if !found {
						orderbookCache[i].Asks = append(orderbookCache[i].Asks,
							orderbook.Item{
								Price:  target,
								Amount: VolumeAdjust,
							},
						)
					}
				}
			}

			if len(ob.Bids) > 0 {
				for _, bid := range ob.Bids {
					target, err := strconv.ParseFloat(bid[0], 64)
					if err != nil {
						log.Fatal(err)
					}

					VolumeAdjust, err := strconv.ParseFloat(bid[1], 64)
					if err != nil {
						log.Fatal(err)
					}

					var found bool
					for x := range orderbookCache[i].Bids {
						if orderbookCache[i].Asks[x].Price == target {
							found = true
							if VolumeAdjust == 0 {
								orderbookCache[i].Bids = append(orderbookCache[i].Bids[:x],
									orderbookCache[i].Bids[x+1:]...)
								break
							}
							orderbookCache[i].Bids[x].Amount = VolumeAdjust
							break
						}
					}

					if !found {
						orderbookCache[i].Asks = append(orderbookCache[i].Bids,
							orderbook.Item{
								Price:  target,
								Amount: VolumeAdjust,
							},
						)
					}
				}
			}
		}
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{}
}
