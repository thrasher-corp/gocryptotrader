package bybit

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bybitWSURLPublicTopicV1  = "wss://stream.bybit.com/spot/quote/ws/v1"
	bybitWSURLPublicTopicV2  = "wss://stream.bybit.com/spot/quote/ws/v2"
	bybitWSURLPrivateTopicV1 = "wss://stream.bybit.com/spot/ws"
	bybitWebsocketTimer      = 30 * time.Second
	wsTicker                 = "ticker"
	wsTrades                 = "trades"
	wsOrderbook              = "orderbook"
	wsMarkets                = "markets"
	wsFills                  = "fills"
	wsOrders                 = "orders"
	wsUpdate                 = "update"
	wsPartial                = "partial"
	subscribe                = "subscribe"
	unsubscribe              = "unsubscribe"
)

var obSuccess = make(map[currency.Pair]bool)

// WsConnect connects to a websocket feed
func (by *Bybit) WsConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := by.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", by.Name)
	}

	go by.wsReadData()
	if by.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = by.WsAuth()
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	return nil
}

// WsAuth sends an authentication message to receive auth data
func (by *Bybit) WsAuth() error {
	intNonce := (time.Now().Unix() + 1) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(by.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []string{by.API.Credentials.Key, strNonce, sign},
	}
	return by.Websocket.Conn.SendJSONMessage(req)
}

// Subscribe sends a websocket message to receive data from the channel
func (by *Bybit) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
channels:
	for i := range channelsToSubscribe {
		var sub WsSub
		sub.Channel = channelsToSubscribe[i].Channel
		sub.Operation = subscribe

		switch channelsToSubscribe[i].Channel {
		case wsFills, wsOrders, wsMarkets:
		default:
			a, err := by.GetPairAssetType(channelsToSubscribe[i].Currency)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}

			formattedPair, err := by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, a)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			sub.Market = formattedPair.String()
		}
		err := by.Websocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
channels:
	for i := range channelsToUnsubscribe {
		var unSub WsSub
		unSub.Operation = unsubscribe
		unSub.Channel = channelsToUnsubscribe[i].Channel
		switch channelsToUnsubscribe[i].Channel {
		case wsFills, wsOrders, wsMarkets:
		default:
			a, err := by.GetPairAssetType(channelsToUnsubscribe[i].Currency)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}

			formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
			if err != nil {
				errs = append(errs, err)
				continue channels
			}
			unSub.Market = formattedPair.String()
		}
		err := by.Websocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

// wsReadData gets and passes on websocket messages for processing
func (by *Bybit) wsReadData() {
	by.Websocket.Wg.Add(1)
	defer by.Websocket.Wg.Done()

	for {
		select {
		case <-by.Websocket.ShutdownC:
			return
		default:
			resp := by.Websocket.Conn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			/*
				err := by.wsHandleData(resp.Raw)
				if err != nil {
					by.Websocket.DataHandler <- err
				}
			*/
		}
	}
}
