package binance

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
)

var lastUpdateID map[string]int64
var m sync.Mutex

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p currency.Pair) error {
	var newOrderBook orderbook.Base

	formattedPair := b.FormatExchangeCurrency(p, assets.AssetTypeSpot)

	orderbookNew, err := b.GetOrderBook(
		OrderBookDataRequestParams{
			Symbol: formattedPair.String(),
			Limit:  1000,
		})

	if err != nil {
		return err
	}

	m.Lock()
	if lastUpdateID == nil {
		lastUpdateID = make(map[string]int64)
	}

	lastUpdateID[formattedPair.String()] = orderbookNew.LastUpdateID
	m.Unlock()

	for _, bids := range orderbookNew.Bids {
		newOrderBook.Bids = append(newOrderBook.Bids,
			orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}
	for _, Asks := range orderbookNew.Asks {
		newOrderBook.Asks = append(newOrderBook.Asks,
			orderbook.Item{Amount: Asks.Quantity, Price: Asks.Price})
	}

	newOrderBook.Pair = currency.NewPairFromString(formattedPair.String())
	newOrderBook.AssetType = assets.AssetTypeSpot
	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, b.GetName(), false)
}

// UpdateLocalCache updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalCache(ob *WebsocketDepthStream) error {
	m.Lock()
	ID, ok := lastUpdateID[ob.Pair]
	if !ok {
		m.Unlock()
		return errors.New("binance_websocket.go - Unable to find lastUpdateID")
	}

	if ob.LastUpdateID+1 <= ID || ID >= ob.LastUpdateID+1 {
		// Drop update, out of order
		m.Unlock()
		return nil
	}

	lastUpdateID[ob.Pair] = ob.LastUpdateID
	m.Unlock()

	var updateBid, updateAsk []orderbook.Item

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
		updateAsk = append(updateAsk, priceToBeUpdated)
	}

	updatedTime := time.Unix(ob.Timestamp, 0)
	currencyPair := currency.NewPairFromString(ob.Pair)

	return b.Websocket.Orderbook.Update(updateBid,
		updateAsk,
		currencyPair,
		updatedTime,
		b.GetName(),
		assets.AssetTypeSpot)
}

// WSConnect intiates a websocket connection
func (b *Binance) WSConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var Dialer websocket.Dialer
	var err error

	pairs := b.GetEnabledPairs(assets.AssetTypeSpot).Strings()
	tick := strings.ToLower(
		strings.Replace(
			strings.Join(pairs, "@ticker/"), "-", "", -1)) + "@ticker"
	trade := strings.ToLower(
		strings.Replace(
			strings.Join(pairs, "@trade/"), "-", "", -1)) + "@trade"
	kline := strings.ToLower(
		strings.Replace(
			strings.Join(pairs, "@kline_1m/"), "-", "", -1)) + "@kline_1m"
	depth := strings.ToLower(
		strings.Replace(
			strings.Join(pairs, "@depth/"), "-", "", -1)) + "@depth"

	wsurl := b.Websocket.GetWebsocketURL() +
		"/stream?streams=" +
		tick +
		"/" +
		trade +
		"/" +
		kline +
		"/" +
		depth

	if b.Websocket.GetProxyAddress() != "" {
		var u *url.URL
		u, err = url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return fmt.Errorf("binance_websocket.go - Unable to connect to parse proxy address. Error: %s",
				err)
		}

		Dialer.Proxy = http.ProxyURL(u)
	}

	for _, ePair := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		err = b.SeedLocalCache(ePair)
		if err != nil {
			return err
		}
	}

	b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})
	if err != nil {
		return fmt.Errorf("binance_websocket.go - Unable to connect to Websocket. Error: %s",
			err)
	}

	go b.WsHandleData()

	return nil
}

// WSReadData reads from the websocket connection and returns the response
func (b *Binance) WSReadData() (exchange.WebsocketResponse, error) {
	msgType, resp, err := b.WebsocketConn.ReadMessage()

	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	b.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Type: msgType, Raw: resp}, nil
}

// WsHandleData handles websocket data from WsReadData
func (b *Binance) WsHandleData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		err := b.WebsocketConn.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			read, err := b.WSReadData()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}

			if read.Type == websocket.TextMessage {
				multiStreamData := MultiStreamData{}
				err = common.JSONDecode(read.Raw, &multiStreamData)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not load multi stream data: %s",
						string(read.Raw))
					continue
				}
				streamType := strings.Split(multiStreamData.Stream, "@")
				switch streamType[1] {
				case "trade":
					trade := TradeStream{}

					err := common.JSONDecode(multiStreamData.Data, &trade)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not unmarshal trade data: %s",
							err)
						continue
					}

					price, err := strconv.ParseFloat(trade.Price, 64)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - price conversion error: %s",
							err)
						continue
					}

					amount, err := strconv.ParseFloat(trade.Quantity, 64)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - amount conversion error: %s",
							err)
						continue
					}

					b.Websocket.DataHandler <- exchange.TradeData{
						CurrencyPair: currency.NewPairFromString(trade.Symbol),
						Timestamp:    time.Unix(0, trade.TimeStamp),
						Price:        price,
						Amount:       amount,
						Exchange:     b.GetName(),
						AssetType:    assets.AssetTypeSpot,
						Side:         trade.EventType,
					}
					continue
				case "ticker":
					t := TickerStream{}

					err := common.JSONDecode(multiStreamData.Data, &t)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to a TickerStream structure %s",
							err.Error())
						continue
					}

					var wsTicker exchange.TickerData

					wsTicker.Timestamp = time.Unix(t.EventTime/1000, 0)
					wsTicker.Pair = currency.NewPairFromString(t.Symbol)
					wsTicker.AssetType = assets.AssetTypeSpot
					wsTicker.Exchange = b.GetName()
					wsTicker.ClosePrice, _ = strconv.ParseFloat(t.CurrDayClose, 64)
					wsTicker.Quantity, _ = strconv.ParseFloat(t.TotalTradedVolume, 64)
					wsTicker.OpenPrice, _ = strconv.ParseFloat(t.OpenPrice, 64)
					wsTicker.HighPrice, _ = strconv.ParseFloat(t.HighPrice, 64)
					wsTicker.LowPrice, _ = strconv.ParseFloat(t.LowPrice, 64)

					b.Websocket.DataHandler <- wsTicker

					continue
				case "kline":
					kline := KlineStream{}

					err := common.JSONDecode(multiStreamData.Data, &kline)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to a KlineStream structure %s",
							err)
						continue
					}

					var wsKline exchange.KlineData

					wsKline.Timestamp = time.Unix(0, kline.EventTime)
					wsKline.Pair = currency.NewPairFromString(kline.Symbol)
					wsKline.AssetType = assets.AssetTypeSpot
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
				case "depth":
					depth := WebsocketDepthStream{}

					err := common.JSONDecode(multiStreamData.Data, &depth)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not convert to depthStream structure %s",
							err)
						continue
					}

					err = b.UpdateLocalCache(&depth)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - UpdateLocalCache error: %s",
							err)
						continue
					}

					currencyPair := currency.NewPairFromString(depth.Pair)

					b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
						Pair:     currencyPair,
						Asset:    assets.AssetTypeSpot,
						Exchange: b.GetName(),
					}
					continue
				}
			}
		}
	}
}
