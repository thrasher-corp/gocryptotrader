package coinbaseinternational

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
func (co *CoinbaseInternational) WsConnect() error {
	if !co.Websocket.IsEnabled() || !co.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := co.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	co.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.PingMessage,
		Delay:       time.Second * 10,
	})
	co.Websocket.Wg.Add(1)
	go co.wsReadData(co.Websocket.Conn)
	subscription := &ChannelSubscription{
		Type:       "SUBSCRIBE",
		ProductIds: []string{"BTC-PERP"},
		Channels:   []string{"LEVEL2"},
		Time:       strconv.FormatInt(time.Now().Unix(), 10),
	}
	err = co.signSubscriptionPayload(subscription)
	if err != nil {
		return err
	}
	return co.Websocket.Conn.SendJSONMessage(subscription)
}

// wsReadData gets and passes on websocket messages for processing
func (co *CoinbaseInternational) wsReadData(conn stream.Connection) {
	defer co.Websocket.Wg.Done()

	for {
		select {
		case <-co.Websocket.ShutdownC:
			return
		default:
			resp := conn.ReadMessage()
			if resp.Raw == nil {
				log.Warnf(log.WebsocketMgr, "%s Received empty message\n", co.Name)
				return
			}

			err := co.wsHandleData(resp.Raw)
			if err != nil {
				co.Websocket.DataHandler <- err
			}
		}
	}
}

func (co *CoinbaseInternational) wsHandleData(respRaw []byte) error {
	println(string(respRaw))
	switch "" {
	case cnlInstruments:
	case cnlMatch:
	case cnlFunding:
	case cnlRisk:
	case cnlOrderbookLevel1:
	case cnlOrderbookLevel2:
	}
	return nil
}

func (co *CoinbaseInternational) handleSubscription(payload []ChannelSubscription) error {
	return nil
}

func (co *CoinbaseInternational) signSubscriptionPayload(body *ChannelSubscription) error {
	creds, err := co.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	var hmac []byte
	secretBytes, err := crypto.Base64Decode(creds.Secret)
	if err != nil {
		return err
	}
	hmac, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(body.Time+", "+creds.Key+", "+"CBINTLMD, "+creds.ClientID),
		secretBytes)
	if err != nil {
		return err
	}
	body.Key = creds.Key
	body.Passphrase = creds.ClientID
	body.Signature = crypto.Base64Encode(hmac)
	return nil
}

func (co *CoinbaseInternational) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	enabledPairs, err := co.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	subscriptions := make([]stream.ChannelSubscription, 0, len(enabledPairs))
	for p := range enabledPairs {
		for x := range defaultSubscriptions {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultSubscriptions[x],
				Currency: enabledPairs[p],
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

func (co *CoinbaseInternational) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}

func (co *CoinbaseInternational) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}
