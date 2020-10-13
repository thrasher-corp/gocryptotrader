package lakebtc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/toorop/go-pusher"
)

const (
	lakeBTCWSURL         = "wss://ws.lakebtc.com:8085"
	marketGlobalEndpoint = "market-global"
	marketSubstring      = "market-"
	globalSubstring      = "-global"
	wssSchem             = "wss"
)

// WsConnect initiates a new websocket connection
func (l *LakeBTC) WsConnect() error {
	if !l.Websocket.IsEnabled() || !l.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	url := strings.Split(lakeBTCWSURL, "://")
	var err error
	l.WebsocketConn.Client, err = pusher.NewCustomClient(strings.ToLower(l.Name),
		url[1],
		wssSchem)
	if err != nil {
		return err
	}
	err = l.WebsocketConn.Client.Subscribe(marketGlobalEndpoint)
	if err != nil {
		return err
	}

	err = l.listenToEndpoints()
	if err != nil {
		return err
	}
	go l.wsHandleIncomingData()
	subs, err := l.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}
	return l.Websocket.SubscribeToChannels(subs)
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
	// LakeBTC does not provide enough trade data to sync to the trade database table
	// please use REST until the API is updated
	// l.WebsocketConn.Trade, err = l.WebsocketConn.Client.Bind("trades")
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (l *LakeBTC) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	enabledCurrencies, err := l.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	for j := range enabledCurrencies {
		enabledCurrencies[j].Delimiter = ""
		channel := marketSubstring +
			enabledCurrencies[j].Lower().String() +
			globalSubstring

		subscriptions = append(subscriptions, stream.ChannelSubscription{
			Channel:  channel,
			Currency: enabledCurrencies[j],
			Asset:    asset.Spot,
		})
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (l *LakeBTC) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToSubscribe {
		err := l.WebsocketConn.Client.Subscribe(channelsToSubscribe[i].Channel)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		l.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to unsubscribe from the channel
func (l *LakeBTC) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToUnsubscribe {
		err := l.WebsocketConn.Client.Unsubscribe(channelsToUnsubscribe[i].Channel)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		l.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
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
			err := l.processTicker(data.Data)
			if err != nil {
				l.Websocket.DataHandler <- err
			}
		case data := <-l.WebsocketConn.Orderbook:
			if l.Verbose {
				log.Debugf(log.ExchangeSys,
					"%v Websocket message received: %v", l.Name, data)
			}
			err := l.processOrderbook(data.Data, data.Channel)
			if err != nil {
				l.Websocket.DataHandler <- err
			}
		}
		select {
		case l.Websocket.TrafficAlert <- struct{}{}:
		default:
		}
	}
}

func (l *LakeBTC) processOrderbook(obUpdate, channel string) error {
	var update WsOrderbookUpdate
	err := json.Unmarshal([]byte(obUpdate), &update)
	if err != nil {
		return err
	}

	p, err := l.getCurrencyFromChannel(channel)
	if err != nil {
		return err
	}

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
			return err
		}
		price, err = strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			return err
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
			return err
		}
		price, err = strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			return err
		}
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}

	return l.Websocket.Orderbook.LoadSnapshot(&book)
}

func (l *LakeBTC) getCurrencyFromChannel(channel string) (currency.Pair, error) {
	curr := strings.Replace(channel, marketSubstring, "", 1)
	curr = strings.Replace(curr, globalSubstring, "", 1)
	return currency.NewPairFromString(curr)
}

func (l *LakeBTC) processTicker(data string) error {
	var tUpdate map[string]wsTicker
	err := json.Unmarshal([]byte(data), &tUpdate)
	if err != nil {
		l.Websocket.DataHandler <- err
		return err
	}

	enabled, err := l.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}

	for k, v := range tUpdate {
		returnCurrency, err := currency.NewPairFromString(k)
		if err != nil {
			return err
		}

		if !enabled.Contains(returnCurrency, true) {
			continue
		}

		l.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: l.Name,
			Bid:          v.Buy,
			High:         v.High,
			Last:         v.Last,
			Low:          v.Low,
			Ask:          v.Sell,
			Volume:       v.Volume,
			AssetType:    asset.Spot,
			Pair:         returnCurrency,
		}
	}
	return nil
}
