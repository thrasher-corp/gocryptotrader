package lakebtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/toorop/go-pusher"
)

const (
	lakeBTCWSURL         = "ws.lakebtc.com:8085"
	marketGlobalEndpoint = "market-global"
	marketSubstring      = "market-"
	globalSubstring      = "-global"
	tickerHighString     = "high"
	tickerLastString     = "last"
	tickerLowString      = "low"
	tickerVolumeString   = "volume"
	wssSchem             = "wss"
)

// WsConnect initiates a new websocket connection
func (l *LakeBTC) WsConnect() error {
	if !l.Websocket.IsEnabled() || !l.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var err error
	l.WebsocketConn.Client, err = pusher.NewCustomClient(strings.ToLower(l.Name), lakeBTCWSURL, wssSchem)
	if err != nil {
		return err
	}
	err = l.WebsocketConn.Client.Subscribe(marketGlobalEndpoint)
	if err != nil {
		return err
	}
	l.GenerateDefaultSubscriptions()
	err = l.listenToEndpoints()
	if err != nil {
		return err
	}
	go l.wsHandleIncomingData()
	return nil
}

func (l *LakeBTC) listenToEndpoints() error {
	var err error
	l.WebsocketConn.Ticker, err = l.WebsocketConn.Client.Bind("tickers")
	if err != nil {
		return fmt.Errorf("%s Websocket Bind error: %s", l.Name, err)
	}
	l.WebsocketConn.Orderbook, err = l.WebsocketConn.Client.Bind("update")
	if err != nil {
		return fmt.Errorf("%s Websocket Bind error: %s", l.Name, err)
	}
	l.WebsocketConn.Trade, err = l.WebsocketConn.Client.Bind("trades")
	if err != nil {
		return fmt.Errorf("%s Websocket Bind error: %s", l.Name, err)
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (l *LakeBTC) GenerateDefaultSubscriptions() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	enabledCurrencies := l.GetEnabledPairs(asset.Spot)

	for j := range enabledCurrencies {
		enabledCurrencies[j].Delimiter = ""
		channel := marketSubstring +
			enabledCurrencies[j].Lower().String() +
			globalSubstring

		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel:  channel,
			Currency: enabledCurrencies[j],
		})
	}
	l.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (l *LakeBTC) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	return l.WebsocketConn.Client.Subscribe(channelToSubscribe.Channel)
}

// wsHandleIncomingData services incoming data from the websocket connection
func (l *LakeBTC) wsHandleIncomingData() {
	l.Websocket.Wg.Add(1)
	defer l.Websocket.Wg.Done()
	for {
		select {
		case <-l.Websocket.ShutdownC:
			return
		case data := <-l.WebsocketConn.Ticker:
			if l.Verbose {
				log.Debugf(log.ExchangeSys,
					"%v Websocket message received: %v", l.Name, data)
			}
			l.Websocket.TrafficAlert <- struct{}{}
			err := l.processTicker(data.Data)
			if err != nil {
				l.Websocket.DataHandler <- err
				return
			}
		case data := <-l.WebsocketConn.Trade:
			l.Websocket.TrafficAlert <- struct{}{}
			if l.Verbose {
				log.Debugf(log.ExchangeSys,
					"%v Websocket message received: %v", l.Name, data)
			}
			err := l.processTrades(data.Data, data.Channel)
			if err != nil {
				l.Websocket.DataHandler <- err
				return
			}
		case data := <-l.WebsocketConn.Orderbook:
			l.Websocket.TrafficAlert <- struct{}{}
			if l.Verbose {
				log.Debugf(log.ExchangeSys,
					"%v Websocket message received: %v", l.Name, data)
			}
			err := l.processOrderbook(data.Data, data.Channel)
			if err != nil {
				l.Websocket.DataHandler <- err
				return
			}
		}
	}
}

func (l *LakeBTC) processTrades(data, channel string) error {
	var tradeData WsTrades
	err := json.Unmarshal([]byte(data), &tradeData)
	if err != nil {
		return err
	}
	curr := l.getCurrencyFromChannel(channel)
	for i := range tradeData.Trades {
		tSide, err := order.StringToOrderSide(tradeData.Trades[i].Type)
		if err != nil {
			l.Websocket.DataHandler <- order.ClassificationError{
				Exchange: l.Name,
				Err:      err,
			}
		}
		l.Websocket.DataHandler <- wshandler.TradeData{
			Timestamp:    time.Unix(tradeData.Trades[i].Date, 0),
			CurrencyPair: curr,
			AssetType:    asset.Spot,
			Exchange:     l.Name,
			EventType:    order.UnknownType,
			Price:        tradeData.Trades[i].Price,
			Amount:       tradeData.Trades[i].Amount,
			Side:         tSide,
		}
	}
	return nil
}

func (l *LakeBTC) processOrderbook(obUpdate, channel string) error {
	var update WsOrderbookUpdate
	err := json.Unmarshal([]byte(obUpdate), &update)
	if err != nil {
		return err
	}

	p := l.getCurrencyFromChannel(channel)

	book := orderbook.Base{
		Pair:         p,
		LastUpdated:  time.Now(),
		AssetType:    asset.Spot,
		ExchangeName: l.Name,
	}

	for i := range update.Asks {
		var amount, price float64
		amount, err = strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			l.Websocket.DataHandler <- fmt.Errorf("%v error parsing ticker data 'low' %v", l.Name, update.Asks[i])
			continue
		}
		price, err = strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			l.Websocket.DataHandler <- fmt.Errorf("%v error parsing orderbook price %v", l.Name, update.Asks[i])
			continue
		}
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}

	for i := range update.Bids {
		var amount, price float64
		amount, err = strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			l.Websocket.DataHandler <- fmt.Errorf("%v error parsing ticker data 'low' %v", l.Name, update.Bids[i])
			continue
		}
		price, err = strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			l.Websocket.DataHandler <- fmt.Errorf("%v error parsing orderbook price %v", l.Name, update.Bids[i])
			continue
		}
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}

	err = l.Websocket.Orderbook.LoadSnapshot(&book)
	if err != nil {
		return err
	}

	l.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    asset.Spot,
		Exchange: l.Name,
	}

	return nil
}

func (l *LakeBTC) getCurrencyFromChannel(channel string) currency.Pair {
	curr := strings.Replace(channel, marketSubstring, "", 1)
	curr = strings.Replace(curr, globalSubstring, "", 1)
	return currency.NewPairFromString(curr)
}

func (l *LakeBTC) processTicker(wsTicker string) error {
	var tUpdate map[string]interface{}
	err := json.Unmarshal([]byte(wsTicker), &tUpdate)
	if err != nil {
		l.Websocket.DataHandler <- err
		return err
	}

	enabled := l.GetEnabledPairs(asset.Spot)
	for k, v := range tUpdate {
		returnCurrency := currency.NewPairFromString(k)
		if !enabled.Contains(returnCurrency, true) {
			continue
		}

		tickerData := v.(map[string]interface{})
		processTickerItem := func(tick map[string]interface{}, item string) float64 {
			if tick[item] == nil {
				return 0
			}

			p, err := strconv.ParseFloat(tick[item].(string), 64)
			if err != nil {
				l.Websocket.DataHandler <- fmt.Errorf("%s error parsing ticker data '%s' %v",
					l.Name,
					item,
					tickerData)
				return 0
			}

			return p
		}

		l.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: l.Name,
			Bid:          processTickerItem(tickerData, order.Buy.Lower()),
			High:         processTickerItem(tickerData, tickerHighString),
			Last:         processTickerItem(tickerData, tickerLastString),
			Low:          processTickerItem(tickerData, tickerLowString),
			Ask:          processTickerItem(tickerData, order.Sell.Lower()),
			Volume:       processTickerItem(tickerData, tickerVolumeString),
			AssetType:    asset.Spot,
			Pair:         returnCurrency,
		}
	}
	return nil
}
