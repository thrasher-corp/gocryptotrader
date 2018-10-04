package binance

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
)

// OrderbookLocalCache stores a seeded localcache for websocket orderbook feeds,
// allowing updates and deletions
type OrderbookLocalCache struct {
	orderbooks   []orderbook.Base
	LastUpdateID int64
	sync.Mutex
}

var localCache OrderbookLocalCache

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p pair.CurrencyPair) error {
	var newOrderBook orderbook.Base

	orderbookNew, err := b.GetOrderBook(
		OrderBookDataRequestParams{
			Symbol: exchange.FormatExchangeCurrency(b.Name, p).String(),
			Limit:  1000,
		})

	if err != nil {
		return err
	}

	for _, bids := range orderbookNew.Bids {
		newOrderBook.Bids = append(newOrderBook.Bids,
			orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}
	for _, Asks := range orderbookNew.Asks {
		newOrderBook.Asks = append(newOrderBook.Asks,
			orderbook.Item{Amount: Asks.Quantity, Price: Asks.Price})
	}

	newOrderBook.Pair = p
	newOrderBook.CurrencyPair = p.Pair().String()
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.AssetType = "SPOT"
	localCache.orderbooks = append(localCache.orderbooks, newOrderBook)

	return nil
}

// UpdatePriceLevels defines price levels that are needing updates
type UpdatePriceLevels []orderbook.Item

// UpdateLocalCache updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalCache(ob WebsocketDepthStream) (orderbook.Base, error) {
	localCache.Lock()
	defer localCache.Unlock()

	var found bool
	for x := range localCache.orderbooks {
		if localCache.orderbooks[x].Pair.Pair().String() == ob.Pair {
			found = true
		}
	}

	if !found {
		err := b.SeedLocalCache(pair.NewCurrencyPairFromString(ob.Pair))
		if err != nil {
			return orderbook.Base{}, err
		}
	}

	if ob.LastUpdateID <= localCache.LastUpdateID {
		return orderbook.Base{}, errors.New("binance_websocket.go - depth event dropped")
	}

	var updateBid, updateAsk UpdatePriceLevels

	for _, bidsToUpdate := range ob.UpdateBids {
		var priceToBeUpdated orderbook.Item
		for i, bids := range bidsToUpdate.([]interface{}) {
			switch i {
			case 0:
				priceToBeUpdated.Price, _ = strconv.ParseFloat(bids.(string), 64)
			case 1:
				priceToBeUpdated.Amount, _ = strconv.ParseFloat(bids.(string), 64)
			}
		}
		updateBid = append(updateBid, priceToBeUpdated)
	}

	for _, asksToUpdate := range ob.UpdateAsks {
		var priceToBeUpdated orderbook.Item
		for i, asks := range asksToUpdate.([]interface{}) {
			switch i {
			case 0:
				priceToBeUpdated.Price, _ = strconv.ParseFloat(asks.(string), 64)
			case 1:
				priceToBeUpdated.Amount, _ = strconv.ParseFloat(asks.(string), 64)
			}
		}
		updateAsk = append(updateBid, priceToBeUpdated)
	}

	for x := range localCache.orderbooks {
		if localCache.orderbooks[x].Pair.Pair().String() == ob.Pair {

			for _, bidsToBeUpdated := range updateBid {
				for y := 0; y < len(localCache.orderbooks[x].Bids); y++ {
					if localCache.orderbooks[x].Bids[y].Price == bidsToBeUpdated.Price {
						if bidsToBeUpdated.Amount == 0 {
							localCache.orderbooks[x].Bids = append(localCache.orderbooks[x].Bids[:y],
								localCache.orderbooks[x].Bids[y+1:]...)
							continue
						}
						localCache.orderbooks[x].Bids[y].Amount = bidsToBeUpdated.Amount
					}
				}
				localCache.orderbooks[x].Bids = append(localCache.orderbooks[x].Bids, bidsToBeUpdated)
			}

			for _, asksToBeUpdated := range updateAsk {
				for y := 0; y < len(localCache.orderbooks[x].Asks); y++ {
					if localCache.orderbooks[x].Asks[y].Price == asksToBeUpdated.Price {
						if asksToBeUpdated.Amount == 0 {
							localCache.orderbooks[x].Asks = append(localCache.orderbooks[x].Asks[:y],
								localCache.orderbooks[x].Asks[y+1:]...)
							continue
						}
						localCache.orderbooks[x].Asks[y].Amount = asksToBeUpdated.Amount
					}
				}
				localCache.orderbooks[x].Asks = append(localCache.orderbooks[x].Asks, asksToBeUpdated)
			}
			localCache.LastUpdateID = ob.LastUpdateID
			return localCache.orderbooks[x], nil
		}
	}
	return orderbook.Base{}, errors.New("binance_websocket.go - local depth cache not seeded correctly")
}

// WSResponse is general response type for channel communications
type WSResponse struct {
	MsgType int
	Resp    []byte
}

// WSConnect intiates a websocket connection
func (b *Binance) WSConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var Dialer websocket.Dialer
	var err error

	ticker := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs, "@ticker/"), "-", "", -1)) + "@ticker"
	trade := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs, "@trade/"), "-", "", -1)) + "@trade"
	kline := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs, "@kline_1m/"), "-", "", -1)) + "@kline_1m"
	depth := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs, "@depth/"), "-", "", -1)) + "@depth"

	wsurl := b.Websocket.GetWebsocketURL() +
		"/stream?streams=" +
		ticker +
		"/" +
		trade +
		"/" +
		kline +
		"/" +
		depth

	if b.Websocket.GetProxyAddress() != "" {
		url, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		Dialer.Proxy = http.ProxyURL(url)
	}

	b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})
	if err != nil {
		return fmt.Errorf("binance_websocket.go - Unable to connect to Websocket. Error: %s",
			err)
	}

	go b.WsHandleData()

	return nil
}

// WSReadData reads from the websocket connection
func (b *Binance) WSReadData(c chan WSResponse) {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				if common.StringContains(err.Error(), "websocket: close 1008") {
					b.Websocket.DataHandler <- exchange.WebsocketDisconnected
					return
				}

				b.Websocket.DataHandler <- err
				continue
			}
			b.Websocket.TrafficTimer.Reset(exchange.WebsocketTrafficLimitTime)
			c <- WSResponse{MsgType: msgType, Resp: resp}
		}
	}
}

// WsHandleData handles websocket data from WsReadData
func (b *Binance) WsHandleData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	var c = make(chan WSResponse, 1)
	go b.WSReadData(c)

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case read := <-c:
			switch read.MsgType {
			case websocket.TextMessage:
				multiStreamData := MultiStreamData{}

				err := common.JSONDecode(read.Resp, &multiStreamData)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not load multi stream data: %s",
						string(read.Resp))
					continue
				}

				if strings.Contains(multiStreamData.Stream, "trade") {
					trade := TradeStream{}

					err := common.JSONDecode(multiStreamData.Data, &trade)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

					price, err := strconv.ParseFloat(trade.Price, 64)
					if err != nil {
						log.Fatal(err)
					}

					amount, err := strconv.ParseFloat(trade.Quantity, 64)
					if err != nil {
						log.Fatal(err)
					}

					b.Websocket.DataHandler <- exchange.TradeData{
						CurrencyPair: pair.NewCurrencyPairFromString(trade.Symbol),
						Timestamp:    time.Unix(0, trade.TimeStamp),
						Price:        price,
						Amount:       amount,
						Exchange:     b.GetName(),
						AssetType:    "SPOT",
						Side:         trade.EventType,
					}
					continue

				} else if strings.Contains(multiStreamData.Stream, "ticker") {
					ticker := TickerStream{}

					err := common.JSONDecode(multiStreamData.Data, &ticker)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to a TickerStream structure %s",
							err.Error())
						continue
					}

					var wsTicker exchange.TickerData

					wsTicker.Timestamp = time.Unix(0, ticker.EventTime)
					wsTicker.Pair = pair.NewCurrencyPairFromString(ticker.Symbol)
					wsTicker.AssetType = "SPOT"
					wsTicker.Exchange = b.GetName()
					wsTicker.ClosePrice, _ = strconv.ParseFloat(ticker.CurrDayClose, 64)
					wsTicker.Quantity, _ = strconv.ParseFloat(ticker.TotalTradedVolume, 64)
					wsTicker.OpenPrice, _ = strconv.ParseFloat(ticker.OpenPrice, 64)
					wsTicker.HighPrice, _ = strconv.ParseFloat(ticker.HighPrice, 64)
					wsTicker.LowPrice, _ = strconv.ParseFloat(ticker.LowPrice, 64)

					b.Websocket.DataHandler <- wsTicker
					continue

				} else if strings.Contains(multiStreamData.Stream, "kline") {
					kline := KlineStream{}

					err := common.JSONDecode(multiStreamData.Data, &kline)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to a KlineStream structure %s",
							err.Error())
						continue
					}

					var wsKline exchange.KlineData

					wsKline.Timestamp = time.Unix(0, kline.EventTime)
					wsKline.Pair = pair.NewCurrencyPairFromString(kline.Symbol)
					wsKline.AssetType = "SPOT"
					wsKline.Exchange = b.GetName()
					wsKline.StartTime = time.Unix(0, kline.Kline.StartTime)
					wsKline.CloseTime = time.Unix(0, kline.Kline.CloseTime)
					wsKline.Interval = kline.Kline.Interval
					wsKline.OpenPrice, _ = strconv.ParseFloat(kline.Kline.OpenPrice, 64)
					wsKline.ClosePrice, _ = strconv.ParseFloat(kline.Kline.ClosePrice, 64)
					wsKline.HighPrice, _ = strconv.ParseFloat(kline.Kline.HighPrice, 64)
					wsKline.LowPrice, _ = strconv.ParseFloat(kline.Kline.LowPrice, 64)
					wsKline.Volume, _ = strconv.ParseFloat(kline.Kline.Volume, 64)

					b.Websocket.DataHandler <- wsKline
					continue

				} else if common.StringContains(multiStreamData.Stream, "depth") {
					depth := WebsocketDepthStream{}

					err := common.JSONDecode(multiStreamData.Data, &depth)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to depthStream structure %s",
							err.Error())
						continue
					}

					newOrderbook, err := b.UpdateLocalCache(depth)
					if err != nil {
						if common.StringContains(err.Error(), "depth event dropped") {
							continue
						}
						b.Websocket.DataHandler <- err
						continue
					}

					orderbook.ProcessOrderbook(b.GetName(), newOrderbook.Pair, newOrderbook, newOrderbook.AssetType)

					b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
						Pair:     newOrderbook.Pair,
						Asset:    newOrderbook.AssetType,
						Exchange: b.GetName(),
					}
					continue
				}
				log.Fatal("binance_websocket.go - websocket edge case, please open issue @github.com",
					multiStreamData)
			}
		}
	}
}

// WSShutdown shuts down websocket connection
func (b *Binance) WSShutdown() error {
	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)
	go func(c chan struct{}) {
		close(b.Websocket.ShutdownC)
		b.Websocket.Wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-c:
		return b.WebsocketConn.Close()
	case <-timer.C:
		return errors.New("binance_websocket.go - websocket routines failed to shutdown")
	}
}
