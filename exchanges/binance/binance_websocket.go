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
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
)

var lastUpdateID map[string]int64
var m sync.Mutex

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p pair.CurrencyPair) error {
	var newOrderBook orderbook.Base

	formattedPair := exchange.FormatExchangeCurrency(b.Name, p)

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

	newOrderBook.Pair = pair.NewCurrencyPairFromString(formattedPair.String())
	newOrderBook.CurrencyPair = formattedPair.String()
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.AssetType = "SPOT"

	return b.Websocket.Orderbook.LoadSnapshot(newOrderBook, b.GetName())
}

// UpdateLocalCache updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalCache(ob WebsocketDepthStream) error {
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
		updateAsk = append(updateBid, priceToBeUpdated)
	}

	updatedTime := time.Unix(ob.Timestamp, 0)
	currencyPair := pair.NewCurrencyPairFromString(ob.Pair)

	return b.Websocket.Orderbook.Update(updateBid,
		updateAsk,
		currencyPair,
		updatedTime,
		b.GetName(),
		"SPOT")
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
			return fmt.Errorf("binance_websocket.go - Unable to connect to parse proxy address. Error: %s",
				err)
		}

		Dialer.Proxy = http.ProxyURL(url)
	}

	for _, ePair := range b.GetEnabledCurrencies() {
		err := b.SeedLocalCache(ePair)
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

// WSReadData reads from the websocket connection
func (b *Binance) WSReadData() {
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
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Websocket Read Data. Error: %s",
					err)
				return
			}

			b.Websocket.TrafficAlert <- struct{}{}
			b.Websocket.Intercomm <- exchange.WebsocketResponse{Type: msgType, Raw: resp}
		}
	}
}

// WsHandleData handles websocket data from WsReadData
func (b *Binance) WsHandleData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	go b.WSReadData()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case read := <-b.Websocket.Intercomm:
			switch read.Type {
			case websocket.TextMessage:
				multiStreamData := MultiStreamData{}

				err := common.JSONDecode(read.Raw, &multiStreamData)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - Could not load multi stream data: %s",
						string(read.Raw))
					continue
				}

				if strings.Contains(multiStreamData.Stream, "trade") {
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
							err)
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
							err)
						continue
					}

					err = b.UpdateLocalCache(depth)
					if err != nil {
						b.Websocket.DataHandler <- fmt.Errorf("binance_websocket.go - UpdateLocalCache error: %s",
							err)
						continue
					}

					currencyPair := pair.NewCurrencyPairFromString(depth.Pair)

					b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
						Pair:     currencyPair,
						Asset:    "SPOT",
						Exchange: b.GetName(),
					}
					continue
				}
			}
		}
	}
}
