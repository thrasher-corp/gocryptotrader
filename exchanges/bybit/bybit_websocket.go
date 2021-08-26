package bybit

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bybitWSURLPublicTopicV2  = "wss://stream.bybit.com/spot/quote/ws/v2"
	bybitWSURLPrivateTopicV1 = "wss://stream.bybit.com/spot/ws"
	bybitWebsocketTimer      = 30 * time.Second
	wsOrderbook              = "depth"
	wsTicker                 = "bookTicker"
	wsTrades                 = "trades"
	wsMarkets                = "kline"

	wsAccountInfo = "outboundAccountInfo"
	wsOrder       = "executionReport"
	wsOrderFilled = "ticketInfo"

	wsUpdate    = "update"
	wsPartial   = "partial"
	subscribe   = "sub"
	unsubscribe = "unsubscribe"
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
		var sub WsReq
		sub.Topic = channelsToSubscribe[i].Channel
		sub.Event = subscribe

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
		sub.Symbol = formattedPair.String()
		err = by.Websocket.Conn.SendJSONMessage(sub)
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

	for i := range channelsToUnsubscribe {
		var unSub WsReq
		unSub.Event = unsubscribe
		unSub.Topic = channelsToUnsubscribe[i].Channel

		a, err := by.GetPairAssetType(channelsToUnsubscribe[i].Currency)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		unSub.Symbol = formattedPair.String()
		err = by.Websocket.Conn.SendJSONMessage(unSub)
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

			err := by.wsHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// GenerateDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	subscriptions = append(subscriptions, stream.ChannelSubscription{
		Channel: wsMarkets,
	})
	var channels = []string{wsTicker, wsTrades, wsOrderbook}
	assets := by.GetAssetTypes(true)
	for a := range assets {
		pairs, err := by.GetEnabledPairs(assets[a])
		if err != nil {
			return nil, err
		}
		for z := range pairs {
			newPair := currency.NewPairWithDelimiter(pairs[z].Base.String(),
				pairs[z].Quote.String(),
				"-")
			for x := range channels {
				subscriptions = append(subscriptions,
					stream.ChannelSubscription{
						Channel:  channels[x],
						Currency: newPair,
						Asset:    assets[a],
					})
			}
		}
	}
	if by.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		var authchan = []string{}
		for x := range authchan {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: authchan[x],
			})
		}
	}
	return subscriptions, nil
}

func timestampFromFloat64(ts float64) time.Time {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return time.Unix(secs, nsecs).UTC()
}

func (by *Bybit) wsHandleData(respRaw []byte) error {
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch result["topic"] {
	case wsUpdate:
		var p currency.Pair
		var a asset.Item
		market, ok := result["market"]
		if ok {
			p, err = currency.NewPairFromString(market.(string))
			if err != nil {
				return err
			}
			a, err = by.GetPairAssetType(p)
			if err != nil {
				return err
			}
		}
		switch result["channel"] {
		case wsTicker:
			var resultData WsTickerDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			by.Websocket.DataHandler <- &ticker.Price{
				ExchangeName: by.Name,
				Bid:          resultData.Ticker.Bid,
				Ask:          resultData.Ticker.Ask,
				LastUpdated:  timestampFromFloat64(resultData.Ticker.Time),
				Pair:         p,
				AssetType:    a,
			}
		case wsOrderbook:
			var resultData WsOrderbookDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			if len(resultData.OBData.Asks) == 0 && len(resultData.OBData.Bids) == 0 {
				return nil
			}
			err = by.WsProcessUpdateOB(&resultData.OBData, p, a)
			if err != nil {
				err2 := by.wsResubToOB(p)
				if err2 != nil {
					by.Websocket.DataHandler <- err2
				}
				return err
			}
		case wsTrades:
			// TODO
		default:
			by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
		}
	case wsPartial:
		switch result["channel"] {
		case "orderbook":
			var p currency.Pair
			var a asset.Item
			market, ok := result["market"]
			if ok {
				p, err = currency.NewPairFromString(market.(string))
				if err != nil {
					return err
				}
				a, err = by.GetPairAssetType(p)
				if err != nil {
					return err
				}
			}
			var resultData WsOrderbookDataStore
			err = json.Unmarshal(respRaw, &resultData)
			if err != nil {
				return err
			}
			err = by.WsProcessPartialOB(&resultData.OBData, p, a)
			if err != nil {
				err2 := by.wsResubToOB(p)
				if err2 != nil {
					by.Websocket.DataHandler <- err2
				}
				return err
			}
			// reset obchecksum failure blockage for pair
			delete(obSuccess, p)
		case wsMarkets:
			// TODO
		}
	case "error":
		by.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: by.Name + stream.UnhandledMessage + string(respRaw),
		}
	}
	return nil
}

// WsProcessUpdateOB processes an update on the orderbook
func (by *Bybit) WsProcessUpdateOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	update := buffer.Update{
		Asset:      a,
		Pair:       p,
		UpdateTime: timestampFromFloat64(data.Time),
	}

	var err error
	for x := range data.Bids {
		update.Bids = append(update.Bids, orderbook.Item{
			Price:  data.Bids[x][0],
			Amount: data.Bids[x][1],
		})
	}
	for x := range data.Asks {
		update.Asks = append(update.Asks, orderbook.Item{
			Price:  data.Asks[x][0],
			Amount: data.Asks[x][1],
		})
	}

	err = by.Websocket.Orderbook.Update(&update)
	if err != nil {
		return err
	}

	updatedOb, err := by.Websocket.Orderbook.GetOrderbook(p, a)
	if err != nil {
		return err
	}
	checksum := by.CalcUpdateOBChecksum(updatedOb)

	if checksum != data.Checksum {
		log.Warnf(log.ExchangeSys, "%s checksum failure for item %s",
			by.Name,
			p)
		return errors.New("checksum failed")
	}
	return nil
}

func (by *Bybit) wsResubToOB(p currency.Pair) error {
	if ok := obSuccess[p]; ok {
		return nil
	}

	obSuccess[p] = true

	channelToResubscribe := &stream.ChannelSubscription{
		Channel:  wsOrderbook,
		Currency: p,
	}
	err := by.Websocket.ResubscribeToChannel(channelToResubscribe)
	if err != nil {
		return fmt.Errorf("%s resubscribe to orderbook failure %s", by.Name, err)
	}
	return nil
}

// WsProcessPartialOB creates an OB from websocket data
func (by *Bybit) WsProcessPartialOB(data *WsOrderbookData, p currency.Pair, a asset.Item) error {
	signedChecksum := by.CalcPartialOBChecksum(data)
	if signedChecksum != data.Checksum {
		return fmt.Errorf("%s channel: %s. Orderbook partial for %v checksum invalid",
			by.Name,
			a,
			p)
	}
	var bids, asks []orderbook.Item
	for x := range data.Bids {
		bids = append(bids, orderbook.Item{
			Price:  data.Bids[x][0],
			Amount: data.Bids[x][1],
		})
	}
	for x := range data.Asks {
		asks = append(asks, orderbook.Item{
			Price:  data.Asks[x][0],
			Amount: data.Asks[x][1],
		})
	}

	newOrderBook := orderbook.Base{
		Asks:            asks,
		Bids:            bids,
		Asset:           a,
		LastUpdated:     timestampFromFloat64(data.Time),
		Pair:            p,
		Exchange:        by.Name,
		VerifyOrderbook: by.CanVerifyOrderbook,
	}
	return by.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// CalcPartialOBChecksum calculates checksum of partial OB data received from WS
func (by *Bybit) CalcPartialOBChecksum(data *WsOrderbookData) int64 {
	var checksum strings.Builder
	var price, amount string
	for i := 0; i < 100; i++ {
		if len(data.Bids)-1 >= i {
			price = checksumParseNumber(data.Bids[i][0])
			amount = checksumParseNumber(data.Bids[i][1])
			checksum.WriteString(price + ":" + amount + ":")
		}
		if len(data.Asks)-1 >= i {
			price = checksumParseNumber(data.Asks[i][0])
			amount = checksumParseNumber(data.Asks[i][1])
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int64(crc32.ChecksumIEEE([]byte(checksumStr)))
}

// CalcUpdateOBChecksum calculates checksum of update OB data received from WS
func (by *Bybit) CalcUpdateOBChecksum(data *orderbook.Base) int64 {
	var checksum strings.Builder
	var price, amount string
	for i := 0; i < 100; i++ {
		if len(data.Bids)-1 >= i {
			price = checksumParseNumber(data.Bids[i].Price)
			amount = checksumParseNumber(data.Bids[i].Amount)
			checksum.WriteString(price + ":" + amount + ":")
		}
		if len(data.Asks)-1 >= i {
			price = checksumParseNumber(data.Asks[i].Price)
			amount = checksumParseNumber(data.Asks[i].Amount)
			checksum.WriteString(price + ":" + amount + ":")
		}
	}
	checksumStr := strings.TrimSuffix(checksum.String(), ":")
	return int64(crc32.ChecksumIEEE([]byte(checksumStr)))
}

func checksumParseNumber(num float64) string {
	modifier := byte('f')
	if num < 0.0001 {
		modifier = 'e'
	}
	r := strconv.FormatFloat(num, modifier, -1, 64)
	if strings.IndexByte(r, '.') == -1 && modifier != 'e' {
		r += ".0"
	}
	return r
}
