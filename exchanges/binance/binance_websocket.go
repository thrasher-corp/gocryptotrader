package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
)

// WsConnect intiates a websocket connection
func (b *Binance) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error

	pairs := b.GetEnabledPairs(asset.Spot).Strings()
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
	enabledPairs := b.GetEnabledPairs(asset.Spot)
	for i := range enabledPairs {
		err = b.SeedLocalCache(enabledPairs[i])
		if err != nil {
			return err
		}
	}

	b.WebsocketConn.URL = wsurl
	b.WebsocketConn.Verbose = b.Verbose

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
				b.Websocket.ReadMessageErrors <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			var multiStreamData MultiStreamData
			err = json.Unmarshal(read.Raw, &multiStreamData)
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
				err := json.Unmarshal(multiStreamData.Data, &trade)
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

				b.Websocket.DataHandler <- wshandler.TradeData{
					CurrencyPair: currency.NewPairFromFormattedPairs(trade.Symbol, b.GetEnabledPairs(asset.Spot),
						b.GetPairFormat(asset.Spot, true)),
					Timestamp: time.Unix(0, trade.TimeStamp*int64(time.Millisecond)),
					Price:     price,
					Amount:    amount,
					Exchange:  b.Name,
					AssetType: asset.Spot,
					Side:      trade.EventType,
				}
				continue
			case "ticker":
				t := TickerStream{}
				err := json.Unmarshal(multiStreamData.Data, &t)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not convert to a TickerStream structure %s",
						b.Name,
						err.Error())
					continue
				}

				b.Websocket.DataHandler <- wshandler.TickerData{
					Exchange:    b.Name,
					Open:        t.OpenPrice,
					Close:       t.ClosePrice,
					Volume:      t.TotalTradedVolume,
					QuoteVolume: t.TotalTradedQuoteVolume,
					High:        t.HighPrice,
					Low:         t.LowPrice,
					Bid:         t.BestBidPrice,
					Ask:         t.BestAskPrice,
					Last:        t.LastPrice,
					Timestamp:   time.Unix(0, t.EventTime*int64(time.Millisecond)),
					AssetType:   asset.Spot,
					Pair: currency.NewPairFromFormattedPairs(t.Symbol, b.GetEnabledPairs(asset.Spot),
						b.GetPairFormat(asset.Spot, true)),
				}

				continue
			case "kline_1m":
				kline := KlineStream{}
				err := json.Unmarshal(multiStreamData.Data, &kline)
				if err != nil {
					b.Websocket.DataHandler <- fmt.Errorf("%v - Could not convert to a KlineStream structure %s",
						b.Name,
						err)
					continue
				}

				var wsKline wshandler.KlineData
				wsKline.Timestamp = time.Unix(0, kline.EventTime*int64(time.Millisecond))
				wsKline.Pair = currency.NewPairFromFormattedPairs(kline.Symbol, b.GetEnabledPairs(asset.Spot),
					b.GetPairFormat(asset.Spot, true))
				wsKline.AssetType = asset.Spot
				wsKline.Exchange = b.Name
				wsKline.StartTime = time.Unix(0, kline.Kline.StartTime*int64(time.Millisecond))
				wsKline.CloseTime = time.Unix(0, kline.Kline.CloseTime*int64(time.Millisecond))
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
				err := json.Unmarshal(multiStreamData.Data, &depth)
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

				currencyPair := currency.NewPairFromFormattedPairs(depth.Pair, b.GetEnabledPairs(asset.Spot),
					b.GetPairFormat(asset.Spot, true))
				b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
					Pair:     currencyPair,
					Asset:    asset.Spot,
					Exchange: b.Name,
				}
				continue
			}
		}
	}
}

// SeedLocalCache seeds depth data
func (b *Binance) SeedLocalCache(p currency.Pair) error {
	var newOrderBook orderbook.Base
	orderbookNew, err := b.GetOrderBook(
		OrderBookDataRequestParams{
			Symbol: b.FormatExchangeCurrency(p, asset.Spot).String(),
			Limit:  1000,
		})
	if err != nil {
		return err
	}

	for i := range orderbookNew.Bids {
		newOrderBook.Bids = append(newOrderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[i].Quantity,
			Price:  orderbookNew.Bids[i].Price,
		})
	}
	for i := range orderbookNew.Asks {
		newOrderBook.Asks = append(newOrderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[i].Quantity,
			Price:  orderbookNew.Asks[i].Price,
		})
	}

	newOrderBook.Pair = p
	newOrderBook.AssetType = asset.Spot
	newOrderBook.ExchangeName = b.Name

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// UpdateLocalCache updates and returns the most recent iteration of the orderbook
func (b *Binance) UpdateLocalCache(wsdp *WebsocketDepthStream) error {
	var updateBid, updateAsk []orderbook.Item
	for i := range wsdp.UpdateBids {
		p, err := strconv.ParseFloat(wsdp.UpdateBids[i][0].(string), 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(wsdp.UpdateBids[i][1].(string), 64)
		if err != nil {
			return err
		}

		updateBid = append(updateBid, orderbook.Item{Price: p, Amount: a})
	}

	for i := range wsdp.UpdateAsks {
		p, err := strconv.ParseFloat(wsdp.UpdateAsks[i][0].(string), 64)
		if err != nil {
			return err
		}
		a, err := strconv.ParseFloat(wsdp.UpdateAsks[i][1].(string), 64)
		if err != nil {
			return err
		}

		updateAsk = append(updateAsk, orderbook.Item{Price: p, Amount: a})
	}
	currencyPair := currency.NewPairFromFormattedPairs(wsdp.Pair, b.GetEnabledPairs(asset.Spot),
		b.GetPairFormat(asset.Spot, true))

	return b.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
		Bids:     updateBid,
		Asks:     updateAsk,
		Pair:     currencyPair,
		UpdateID: wsdp.LastUpdateID,
		Asset:    asset.Spot,
	})
}
