package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (

	// okxBusinessWebsocket
	okxBusinessWebsocket = "wss://ws.okx.com:8443/ws/v5/business"
)

// WsConnectBusiness connects to a business wbesocket channel.
func (ok *Okx) WsConnectBusiness() error {
	if !ok.Websocket.IsEnabled() || !ok.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	ok.Websocket.Conn.SetURL(okxBusinessWebsocket)
	err := ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	ok.Websocket.Wg.Add(1)
	go ok.wsReadData(ok.Websocket.Conn)
	if ok.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			ok.Websocket.GetWebsocketURL())
	}
	ok.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	if ok.IsWebsocketAuthenticationSupported() {
		var authDialer websocket.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = ok.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsSpreadAuth will connect to Okx's Private websocket connection and Authenticate with a login payload.
func (ok *Okx) WsSpreadAuth(ctx context.Context, dialer *websocket.Dialer) error {
	if !ok.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", ok.Name)
	}
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	timeUnix := time.Now()
	signPath := "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(timeUnix.UTC().Unix(), 10)+http.MethodGet+signPath),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	base64Sign := crypto.Base64Encode(hmac)
	request := WebsocketEventRequest{
		Operation: operationLogin,
		Arguments: []WebsocketLoginData{
			{
				APIKey:     creds.Key,
				Passphrase: creds.ClientID,
				Timestamp:  timeUnix,
				Sign:       base64Sign,
			},
		},
	}
	err = ok.Websocket.AuthConn.SendJSONMessage(request)
	if err != nil {
		return err
	}
	timer := time.NewTimer(ok.WebsocketResponseCheckTimeout)
	randomID, err := common.GenerateRandomString(16)
	if err != nil {
		return fmt.Errorf("%w, generating random string for incoming websocket response failed", err)
	}
	wsResponse := make(chan *wsIncomingData)
	ok.WsResponseMultiplexer.Register <- &wsRequestInfo{
		ID:    randomID,
		Chan:  wsResponse,
		Event: operationLogin,
	}
	ok.WsRequestSemaphore <- 1
	defer func() {
		<-ok.WsRequestSemaphore
	}()
	defer func() { ok.WsResponseMultiplexer.Unregister <- randomID }()
	for {
		select {
		case data := <-wsResponse:
			if data.Event == operationLogin && data.Code == "0" {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
				return nil
			} else if data.Event == "error" &&
				(data.Code == "60022" || data.Code == "60009") {
				ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
				return fmt.Errorf("authentication failed with error: %v", ErrorCodes[data.Code])
			}
			continue
		case <-timer.C:
			timer.Stop()
			return fmt.Errorf("%s websocket connection: timeout waiting for response with an operation: %v",
				ok.Name,
				request.Operation)
		}
	}
}
