package ftx

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	ftxWSURL          = "wss://ftx.com/ws/"
	ftxWebsocketTimer = 13
)

// WsConnect connects to a websocket feed
func (f *Ftx) WsConnect() error {
	if !f.Websocket.IsEnabled() || !f.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := f.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	f.WebsocketConn.SetupPingHandler(wshandler.WebsocketPingHandler{
		MessageType: websocket.PingMessage,
		Delay:       ftxWebsocketTimer,
	})
	if f.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", f.Name)
	}
	// go f.wsReadData()
	if f.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err := f.WsAuth()
		if err != nil {
			f.Websocket.DataHandler <- err
			f.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	fmt.Printf("generate default subscriptions here \n\n\n")
	return nil
}

// WsAuth sends an authentication message to receive auth data
func (f *Ftx) WsAuth() error {
	nonce := strconv.FormatInt(int64(time.Now().UnixNano()/1000000), 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte(nonce+"websocket_login"),
		[]byte(f.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{Operation: "login",
		Args: AuthenticationData{Key: f.API.Credentials.Key,
			Sign: sign,
			Time: nonce,
		},
	}
	return f.WebsocketConn.SendJSONMessage(req)
}

// Subscribe sends a websocket message to receive data from the channel
func (f *Ftx) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub WsSub
	sub.Operation = "subscribe"
	sub.Channel = channelToSubscribe.Channel
	sub.Market = f.FormatExchangeCurrency(channelToSubscribe.Currency, asset.Spot).String()
	return f.WebsocketConn.SendJSONMessage(sub)
}

// GenerateDefaultSubscription generates default subscription
func GenerateDefaultSubscription() {

}
