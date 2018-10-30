package bitstamp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
	pusher "github.com/toorop/go-pusher"
)

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

	for _, enabledPair := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		if enabledPair.String() == tradingPair {
			return tradingPair, nil
		}
	}

	return "", errors.New("bistamp_websocket.go error - could not find trading pair")
}

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	tradingPairs = make(map[string]string)
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		log.Warn("bitstamp_websocket.go warning - set proxy address error: proxy not supported")
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

	go b.WsReadData()

	for _, p := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		orderbookSeed, err := b.GetOrderbook(p.String())
		if err != nil {
			return err
		}

		var newOrderBook orderbook.Base

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

		newOrderBook.Asks = asks
		newOrderBook.Bids = bids
		newOrderBook.Pair = p
		newOrderBook.LastUpdated = time.Unix(0, orderbookSeed.Timestamp)
		newOrderBook.AssetType = assets.AssetTypeSpot

		err = b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, b.GetName(), false)
		if err != nil {
			return err
		}

		b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
			Pair:     p,
			Asset:    assets.AssetTypeSpot,
			Exchange: b.GetName(),
		}

		err = b.WebsocketConn.Client.Subscribe(fmt.Sprintf("live_trades_%s",
			p.Lower().String()))

		if err != nil {
			return fmt.Errorf("%s Websocket Trade subscription error: %s",
				b.GetName(),
				err)
		}

		err = b.WebsocketConn.Client.Subscribe(fmt.Sprintf("diff_order_book_%s",
			p.Lower().String()))

		if err != nil {
			return fmt.Errorf("%s Websocket Trade subscription error: %s",
				b.GetName(),
				err)
		}

	}
	return nil
}

// WsReadData reads data coming from bitstamp websocket connection
func (b *Bitstamp) WsReadData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		err := b.WebsocketConn.Client.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("bitstamp_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case data := <-b.WebsocketConn.Data:
			b.Websocket.TrafficAlert <- struct{}{}

			result := PusherOrderbook{}
			err := common.JSONDecode([]byte(data.Data), &result)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			currencyPair := common.SplitStrings(data.Channel, "_")
			p := currency.NewPairFromString(common.StringToUpper(currencyPair[3]))

			err = b.WsUpdateOrderbook(result, p, assets.AssetTypeSpot)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

		case trade := <-b.WebsocketConn.Trade:
			b.Websocket.TrafficAlert <- struct{}{}

			result := PusherTrade{}
			err := common.JSONDecode([]byte(trade.Data), &result)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			currencyPair := common.SplitStrings(trade.Channel, "_")

			b.Websocket.DataHandler <- exchange.TradeData{
				Price:        result.Price,
				Amount:       result.Amount,
				CurrencyPair: currency.NewPairFromString(currencyPair[2]),
				Exchange:     b.GetName(),
				AssetType:    assets.AssetTypeSpot,
			}
		}
	}
}

// WsUpdateOrderbook updates local cache of orderbook information
func (b *Bitstamp) WsUpdateOrderbook(ob PusherOrderbook, p currency.Pair, assetType assets.AssetType) error {
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
