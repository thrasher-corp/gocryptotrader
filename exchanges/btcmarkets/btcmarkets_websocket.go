package btcmarkets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	btcMarketsWSURL = "wss://socket.btcmarkets.net/v2"
)

// WsConnect connects to a websocket feed
func (b *BTCMarkets) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.Name)
	}
	go b.WsHandleData()
	if b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		b.createChannels()
		if err != nil {
			b.Websocket.DataHandler <- err
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	b.generateDefaultSubscriptions()
	return nil
}

// WsHandleData handles websocket data from WsReadData
func (b *BTCMarkets) WsHandleData() {
	b.Websocket.Wg.Add(1)
	defer func() {
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		default:
			resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.ReadMessageErrors <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			var wsResponse WsMessageType
			err = json.Unmarshal(resp.Raw, &wsResponse)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}
			switch wsResponse.MessageType {
			case heartbeat:
				if b.Verbose {
					log.Debugf(log.ExchangeSys, "%v - Websocket heartbeat received %s", b.Name, resp.Raw)
				}
			case wsOB:
				var ob WsOrderbook
				err := json.Unmarshal(resp.Raw, &ob)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				p, err := currency.NewPairFromString(ob.Currency)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				var bids, asks []orderbook.Item
				for x := range ob.Bids {
					var price, amount float64
					price, err = strconv.ParseFloat(ob.Bids[x][0], 64)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					amount, err = strconv.ParseFloat(ob.Bids[x][1], 64)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					bids = append(bids, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}
				for x := range ob.Asks {
					var price, amount float64
					price, err = strconv.ParseFloat(ob.Asks[x][0], 64)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					amount, err = strconv.ParseFloat(ob.Asks[x][1], 64)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					asks = append(asks, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}
				err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
					Pair:         p,
					Bids:         bids,
					Asks:         asks,
					LastUpdated:  ob.Timestamp,
					AssetType:    asset.Spot,
					ExchangeName: b.Name,
				})
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
					Pair:     p,
					Asset:    asset.Spot,
					Exchange: b.Name,
				}
			case trade:
				var trade WsTrade
				err := json.Unmarshal(resp.Raw, &trade)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				p, err := currency.NewPairFromString(trade.Currency)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				b.Websocket.DataHandler <- wshandler.TradeData{
					Timestamp:    trade.Timestamp,
					CurrencyPair: p,
					AssetType:    asset.Spot,
					Exchange:     b.Name,
					Price:        trade.Price,
					Amount:       trade.Volume,
					Side:         order.SideUnknown.String(),
					EventType:    order.Unknown.String(),
				}
			case tick:
				var tick WsTick
				err := json.Unmarshal(resp.Raw, &tick)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				p, err := currency.NewPairFromString(tick.Currency)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				b.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: b.Name,
					Volume:       tick.Volume,
					High:         tick.High24,
					Low:          tick.Low24h,
					Bid:          tick.Bid,
					Ask:          tick.Ask,
					Last:         tick.Last,
					LastUpdated:  tick.Timestamp,
					AssetType:    asset.Spot,
					Pair:         p,
				}
			case fundChange:
				var transferData WsFundTransfer
				err := json.Unmarshal(resp.Raw, &transferData)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				b.Websocket.DataHandler <- transferData
			case orderChange:
				var orderData WsOrderChange
				err := json.Unmarshal(resp.Raw, &orderData)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				b.Websocket.DataHandler <- orderData
			case "error":
				var wsErr WsError
				err := json.Unmarshal(resp.Raw, &wsErr)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				b.Websocket.DataHandler <- fmt.Errorf("%v websocket error. Code: %v Message: %v", b.Name, wsErr.Code, wsErr.Message)
			default:
				b.Websocket.DataHandler <- fmt.Errorf("%v Unhandled websocket message %s", b.Name, resp.Raw)
			}
		}
	}
}

var channels = []string{tick, trade, wsOB}

func (b *BTCMarkets) generateDefaultSubscriptions() {
	enabledCurrencies, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.WebsocketMgr,
			"%s could not generate default subscriptions Err: %s",
			b.Name,
			err)
		return
	}
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

var unauthChannels = []string{tick, trade, wsOB}
var authChannels = []string{fundChange, heartbeat, orderChange}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTCMarkets) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	switch {
	case common.StringDataCompare(unauthChannels, channelToSubscribe.Channel):
		req := WsSubscribe{
			MarketIDs:   []string{b.FormatExchangeCurrency(channelToSubscribe.Currency, asset.Spot).String()},
			Channels:    []string{channelToSubscribe.Channel},
			MessageType: subscribe,
		}
		err := b.WebsocketConn.SendJSONMessage(req)
		if err != nil {
			return err
		}
	case common.StringDataCompare(authChannels, channelToSubscribe.Channel):
		message, ok := channelToSubscribe.Params["AuthSub"].(WsAuthSubscribe)
		if !ok {
			return errors.New("invalid params data")
		}
		tempAuthData := b.generateAuthSubscriptions()
		message.Channels = append(message.Channels, channelToSubscribe.Channel, heartbeat)
		message.Key = tempAuthData.Key
		message.Signature = tempAuthData.Signature
		message.Timestamp = tempAuthData.Timestamp
		err := b.WebsocketConn.SendJSONMessage(message)
		if err != nil {
			return err
		}
	}
	return nil
}

// Login logs in allowing private ws events
func (b *BTCMarkets) generateAuthSubscriptions() WsAuthSubscribe {
	var authSubInfo WsAuthSubscribe
	signTime := strconv.FormatInt(time.Now().UTC().UnixNano()/1000000, 10)
	strToSign := "/users/self/subscribe" + "\n" + signTime
	tempSign := crypto.GetHMAC(crypto.HashSHA512,
		[]byte(strToSign),
		[]byte(b.API.Credentials.Secret))
	sign := crypto.Base64Encode(tempSign)
	authSubInfo.Key = b.API.Credentials.Key
	authSubInfo.Signature = sign
	authSubInfo.Timestamp = signTime
	return authSubInfo
}

var tempChannels = []string{orderChange, fundChange}

// createChannels creates channels that need to be
func (b *BTCMarkets) createChannels() {
	var wsChannels []wshandler.WebsocketChannelSubscription
	pairArray, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.WebsocketMgr,
			"%s could not create channels Err: %s",
			b.Name,
			err)
		return
	}
	for y := range tempChannels {
		for x := range pairArray {
			var authSub WsAuthSubscribe
			var channel wshandler.WebsocketChannelSubscription
			channel.Params = make(map[string]interface{})
			channel.Channel = tempChannels[y]
			authSub.MarketIDs = append(authSub.MarketIDs, b.FormatExchangeCurrency(pairArray[x], asset.Spot).String())
			authSub.MessageType = subscribe
			channel.Params["AuthSub"] = authSub
			wsChannels = append(wsChannels, channel)
		}
	}
	b.Websocket.SubscribeToChannels(wsChannels)
}
