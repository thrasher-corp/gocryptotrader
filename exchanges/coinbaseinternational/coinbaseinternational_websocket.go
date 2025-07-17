package coinbaseinternational

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	coinbaseinternationalWSAPIURL = "wss://ws-md.international.coinbase.com"

	cnlInstruments     = "INSTRUMENTS"
	cnlMatch           = "MATCH"
	cnlFunding         = "FUNDING"
	cnlRisk            = "RISK"
	cnlOrderbookLevel1 = "LEVEL1"
	cnlOrderbookLevel2 = "LEVEL2"
)

var defaultSubscriptions = []string{
	cnlInstruments,
	cnlMatch,
	cnlFunding,
	cnlRisk,
	cnlOrderbookLevel2,
}

// WsConnect connects to websocket client.
// The WebSocket feed is publicly available and provides real-time
// market data updates for orders and trades.
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	err := e.Websocket.Conn.Dial(context.Background(), &gws.Dialer{Proxy: http.ProxyFromEnvironment}, http.Header{})
	if err != nil {
		return err
	}
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.PingMessage,
		Delay:       time.Second * 10,
	})
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.Conn)

	return e.handleSubscription([]SubscriptionInput{{
		Type:       "SUBSCRIBE",
		ProductIDs: []string{"BTC-PERP"},
		Channels:   []string{"LEVEL2"},
	}})
}

// wsReadData gets and passes on websocket messages for processing
func (e *Exchange) wsReadData(conn websocket.Connection) {
	defer e.Websocket.Wg.Done()
	for {
		select {
		case <-e.Websocket.ShutdownC:
			return
		default:
			resp := conn.ReadMessage()
			if resp.Raw == nil {
				log.Warnf(log.WebsocketMgr, "%s Received empty message\n", e.Name)
				return
			}
			err := e.wsHandleData(resp.Raw)
			if err != nil {
				e.Websocket.DataHandler <- err
			}
		}
	}
}

func (e *Exchange) wsHandleData(respRaw []byte) error {
	var resp SubscriptionResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	var pairs currency.Pairs
	switch resp.Type {
	case "SUBSCRIBE":
		var subsccefulySubscribedChannels subscription.List
		for x := range resp.Channels {
			pairs, err = currency.NewPairsFromStrings(resp.Channels[x].ProductIDs)
			if err != nil {
				return err
			}
			subsccefulySubscribedChannels = append(subsccefulySubscribedChannels,
				&subscription.Subscription{
					Channel: resp.Channels[x].Name,
					Pairs:   pairs,
				})
		}
		err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, subsccefulySubscribedChannels...)
		if err != nil {
			return err
		}
	case "UNSUBSCRIBE":
		var subsccefulySubscribedChannels subscription.List
		for x := range resp.Channels {
			pairs, err = currency.NewPairsFromStrings(resp.Channels[x].ProductIDs)
			if err != nil {
				return err
			}
			subsccefulySubscribedChannels = append(subsccefulySubscribedChannels,
				&subscription.Subscription{
					Channel: resp.Channels[x].Name,
					Pairs:   pairs,
				})
		}
		err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subsccefulySubscribedChannels...)
		if err != nil {
			return err
		}
	case "REJECT":
		return fmt.Errorf("%s %v message: %s, reason: %s  ", resp.Channel, resp.Type, resp.Message, resp.Reason)
	default: //  SNAPSHOT and UPDATE
	}
	switch resp.Channel {
	case cnlInstruments:
		return e.processInstruments(respRaw)
	case cnlMatch:
		return e.processMatch(respRaw)
	case cnlFunding:
		return e.processFunding(respRaw)
	case cnlRisk:
		return e.processRisk(respRaw)
	case cnlOrderbookLevel1:
		return e.processOrderbookLevel1(respRaw)
	case cnlOrderbookLevel2:
		return e.processOrderbookLevel2(respRaw)
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: string(respRaw),
		}
		return fmt.Errorf("unhandled message: %s", string(respRaw))
	}
}

func (e *Exchange) processOrderbookLevel2(respRaw []byte) error {
	var resp []WsOrderbookLevel2
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for x := range resp {
		pair, err := currency.NewPairFromString(resp[x].ProductID)
		if err != nil {
			return err
		}
		asks := make([]orderbook.Level, len(resp[x].Asks))
		for a := range resp[x].Asks {
			asks[a].Price = resp[x].Asks[a][0].Float64()
			asks[a].Amount = resp[x].Asks[a][1].Float64()
		}
		bids := make([]orderbook.Level, len(resp[x].Bids))
		for b := range resp[x].Bids {
			bids[b].Price = resp[x].Bids[b][0].Float64()
			bids[b].Amount = resp[x].Bids[b][1].Float64()
		}
		if resp[x].Type == "UPDATE" {
			err = e.Websocket.Orderbook.Update(&orderbook.Update{
				UpdateID:   resp[x].Sequence,
				UpdateTime: resp[x].Time,
				Asset:      asset.Spot,
				Action:     orderbook.UpdateAction,
				Bids:       bids,
				Asks:       asks,
				Pair:       pair,
			})
			if err != nil {
				return err
			}
		}
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Bids:         bids,
			Asks:         asks,
			Pair:         pair,
			Exchange:     e.Name,
			Asset:        asset.Spot,
			LastUpdated:  resp[x].Time,
			LastUpdateID: resp[x].Sequence,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processOrderbookLevel1(respRaw []byte) error {
	var resp []WsOrderbookLevel1
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	for x := range resp {
		pair, err := currency.NewPairFromString(resp[x].ProductID)
		if err != nil {
			return err
		}
		if resp[x].Type == "UPDATE" {
			err = e.Websocket.Orderbook.Update(&orderbook.Update{
				Pair:       pair,
				Asset:      asset.Spot,
				UpdateTime: resp[x].Time,
				Action:     orderbook.UpdateAction,
				UpdateID:   resp[x].Sequence,
				Asks:       []orderbook.Level{{Price: resp[x].AskPrice.Float64(), Amount: resp[x].AskQty.Float64()}},
				Bids:       []orderbook.Level{{Price: resp[x].BidPrice.Float64(), Amount: resp[x].BidQty.Float64()}},
			})
			if err != nil {
				return err
			}
		}
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:         pair,
			Exchange:     e.Name,
			Asset:        asset.Spot,
			LastUpdated:  resp[x].Time,
			LastUpdateID: resp[x].Sequence,
			Asks:         []orderbook.Level{{Price: resp[x].AskPrice.Float64(), Amount: resp[x].AskQty.Float64()}},
			Bids:         []orderbook.Level{{Price: resp[x].BidPrice.Float64(), Amount: resp[x].BidQty.Float64()}},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) processRisk(respRaw []byte) error {
	var resp []WsRisk
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processFunding(respRaw []byte) error {
	var resp []WsFunding
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	fundingInfos := make([]fundingrate.Rate, len(resp))
	for x := range resp {
		fundingInfos[x] = fundingrate.Rate{
			Time: resp[x].Time,
			Rate: decimal.NewFromFloat(resp[x].FundingRate.Float64()),
		}
	}
	e.Websocket.DataHandler <- fundingInfos
	return nil
}

func (e *Exchange) processMatch(respRaw []byte) error {
	var resp []WsMatch
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processInstruments(respRaw []byte) error {
	var resp []WsInstrument
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// GenerateSubscriptionPayload generates a subscription payloads list.
func (e *Exchange) GenerateSubscriptionPayload(subscriptions subscription.List, operation string) ([]SubscriptionInput, error) {
	if len(subscriptions) == 0 {
		return nil, common.ErrEmptyParams
	}
	channelPairsMap := make(map[string]currency.Pairs)
	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	for x := range subscriptions {
		_, okay := channelPairsMap[subscriptions[x].Channel]
		if !okay {
			channelPairsMap[subscriptions[x].Channel] = currency.Pairs{}
		}
		for p := range subscriptions[x].Pairs {
			channelPairsMap[subscriptions[x].Channel] = channelPairsMap[subscriptions[x].Channel].Add(subscriptions[x].Pairs[p].Format(format))
		}
	}
	payloads := make([]SubscriptionInput, 0, len(channelPairsMap))
	var payload *SubscriptionInput
	first := true
	for key, mPairs := range channelPairsMap {
		if first {
			first = false
			payload = &SubscriptionInput{
				Channels: []string{
					key,
				},
				ProductIDs:     mPairs.Strings(),
				ProductIDPairs: mPairs,
				Type:           operation,
			}
		}
		diff, err := payload.ProductIDPairs.FindDifferences(mPairs, format)
		if err != nil {
			return nil, err
		}
		if len(diff.New) == 0 && len(diff.Remove) == 0 {
			payload.Channels = append(payload.Channels, key)
		} else {
			match := false
			for p := range payloads {
				diff, err = payloads[p].ProductIDPairs.FindDifferences(mPairs, format)
				if err != nil {
					return nil, err
				}
				if len(diff.New) == 0 && len(diff.Remove) == 0 {
					match = true
					payloads[p].Channels = append(payloads[p].Channels, key)
					break
				}
			}
			if match {
				continue
			}
			payloads = append(payloads, *payload)
			payload = &SubscriptionInput{
				Type: operation,
				Channels: []string{
					key,
				},
				ProductIDs:     mPairs.Strings(),
				ProductIDPairs: mPairs,
			}
		}
	}
	payloads = append(payloads, *payload)
	return payloads, nil
}

func (e *Exchange) handleSubscription(payload []SubscriptionInput) error {
	var (
		authenticate bool
		creds        *account.Credentials
	)
	if e.AreCredentialsValid(context.Background()) && e.Websocket.CanUseAuthenticatedEndpoints() {
		var err error
		creds, err = e.GetCredentials(context.Background())
		if err != nil {
			return err
		}
		authenticate = true
	}
	for x := range payload {
		payload[x].Time = strconv.FormatInt(time.Now().Unix(), 10)
		if authenticate {
			err := e.signSubscriptionPayload(creds, &payload[x])
			if err != nil {
				return err
			}
		}
		err := e.Websocket.Conn.SendJSONMessage(context.Background(), request.Unset, payload[x])
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) signSubscriptionPayload(creds *account.Credentials, body *SubscriptionInput) error {
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(body.Time+creds.Key+"CBINTLMD"+creds.ClientID),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}
	body.Key = creds.Key
	body.Passphrase = creds.ClientID
	body.Signature = hex.EncodeToString(hmac)
	return nil
}

// GenerateDefaultSubscriptions generates default subscription
func (e *Exchange) GenerateDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	subscriptions := make(subscription.List, 0, len(enabledPairs))
	for x := range defaultSubscriptions {
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: defaultSubscriptions[x],
			Pairs:   enabledPairs,
			Asset:   asset.Spot,
		})
	}
	return subscriptions, nil
}

// Subscribe subscribe to channels
func (e *Exchange) Subscribe(subscriptions subscription.List) error {
	subscriptionPayloads, err := e.GenerateSubscriptionPayload(subscriptions, "SUBSCRIBE")
	if err != nil {
		return err
	}
	return e.handleSubscription(subscriptionPayloads)
}

// Unsubscribe unsubscribe to channels
func (e *Exchange) Unsubscribe(subscriptions subscription.List) error {
	subscriptionPayloads, err := e.GenerateSubscriptionPayload(subscriptions, "UNSUBSCRIBE")
	if err != nil {
		return err
	}
	return e.handleSubscription(subscriptionPayloads)
}
