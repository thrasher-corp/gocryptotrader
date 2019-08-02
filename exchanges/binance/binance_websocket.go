package binance

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/connection"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/monitor"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/ob"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
)

// WSConnect intiates a websocket connection
func (b *Binance) WSConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(monitor.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error

	tick := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs.Strings(), "@ticker/"), "-", "", -1)) + "@ticker"
	trade := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs.Strings(), "@trade/"), "-", "", -1)) + "@trade"
	kline := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs.Strings(), "@kline_1m/"), "-", "", -1)) + "@kline_1m"
	depth := strings.ToLower(
		strings.Replace(
			strings.Join(b.EnabledPairs.Strings(), "@depth/"), "-", "", -1)) + "@depth"

	wsurl := b.Websocket.GetWebsocketURL() +
		"/stream?streams=" +
		tick +
		"/" +
		trade +
		"/" +
		kline +
		"/" +
		depth
	for _, ePair := range b.GetEnabledCurrencies() {
		err = b.SeedLocalCache(ePair)
		if err != nil {
			return err
		}
	}

	b.WebsocketConn.URL = wsurl
	err = b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	go b.WsHandleData()

	return nil
}

// WsHandleData handles websocket data from WsReadData
func (b *Binance) WsHandleData() {
	b.Websocket.Wg.Add(1)
	defer func() {
		b.Websocket.Wg.Done()
	}()
	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			read, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			var multiStreamData MultiStreamData
			err = common.JSONDecode(read.Raw, &multiStreamData)
			if err != nil {
				b.Websocket.DataHandler <- fmt.Errorf("%v - Could not load multi stream data: %s",
					b.Name,
					read.Raw)
				continue
			}
			streamType := strings.Split(multiStreamData.Stream, "@")
			switch streamType[1] {
			case "trade":
				trade := TradeStream{}
				err := common.JSONDecode(multiStreamData.Data, &trade)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not unmarshal trade data: %s",
						b.Name,
						err)
					continue
				}

				price, err := strconv.ParseFloat(trade.Price, 64)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - price conversion error: %s",
						b.Name,
						err)
					continue
				}

				amount, err := strconv.ParseFloat(trade.Quantity, 64)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - amount conversion error: %s",
						b.Name,
						err)
					continue
				}

				b.Websocket.DataHandler <- monitor.TradeData{
					CurrencyPair: currency.NewPairFromString(trade.Symbol),
					Timestamp:    time.Unix(0, trade.TimeStamp),
					Price:        price,
					Amount:       amount,
					Exchange:     b.GetName(),
					AssetType:    orderbook.Spot,
					Side:         trade.EventType,
				}
				continue
			case "ticker":
				t := TickerStream{}
				err := common.JSONDecode(multiStreamData.Data, &t)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
						b.Name,
						err.Error())
					continue
				}

				var wsTicker monitor.TickerData

				wsTicker.Timestamp = time.Unix(t.EventTime/1000, 0)
				wsTicker.Pair = currency.NewPairFromString(t.Symbol)
				wsTicker.AssetType = ticker.Spot
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
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
						b.Name,
						err)
					continue
				}

				var wsKline monitor.KlineData
				wsKline.Timestamp = time.Unix(0, kline.EventTime)
				wsKline.Pair = currency.NewPairFromString(kline.Symbol)
				wsKline.AssetType = ticker.Spot
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
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not convert to depthStream structure %s",
						b.Name,
						err)
					continue
				}

				err = b.UpdateLocalCache(&depth)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - UpdateLocalCache error: %s",
						b.Name,
						err)
					continue
				}

				currencyPair := currency.NewPairFromString(depth.Pair)
				b.Websocket.DataHandler <- monitor.WebsocketOrderbookUpdate{
					Pair:     currencyPair,
					Asset:    orderbook.Spot,
					Exchange: b.GetName(),
				}
				continue
			}
		}
	}
}

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p currency.Pair) error {
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

	for _, bids := range orderbookNew.Bids {
		newOrderBook.Bids = append(newOrderBook.Bids,
			orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}
	for _, Asks := range orderbookNew.Asks {
		newOrderBook.Asks = append(newOrderBook.Asks,
			orderbook.Item{Amount: Asks.Quantity, Price: Asks.Price})
	}

	newOrderBook.LastUpdated = time.Unix(orderbookNew.LastUpdateID, 0)
	newOrderBook.Pair = currency.NewPairFromString(formattedPair.String())
	newOrderBook.AssetType = ticker.Spot

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, b.GetName(), false)
}

// UpdateLocalCache updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalCache(wsdp *WebsocketDepthStream) error {
	var updateBid, updateAsk []orderbook.Item
	for _, bidsToUpdate := range wsdp.UpdateBids {
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

	for _, asksToUpdate := range wsdp.UpdateAsks {
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
	currencyPair := currency.NewPairFromString(wsdp.Pair)

	return b.Websocket.Orderbook.Update(&ob.WebsocketOrderbookUpdate{
		Bids:              updateBid,
		Asks:              updateAsk,
		CurrencyPair:      currencyPair,
		UpdateEntriesByID: true,
		UpdateID:          wsdp.LastUpdateID,
		ExchangeName:      b.GetName(),
		AssetType:         orderbook.Spot,
		SortBuffer:        true,
		BufferEnabled:     true,
	})
}
