package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	eoptionsWebsocketURL = "wss://nbstream.binance.com/eoptions/"
)

// WsOptionsConnect initiates a websocket connection to coin margined futures websocket
func (b *Binance) WsOptionsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var err error
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	wsURL := eoptionsWebsocketURL
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Errorf(log.ExchangeSys,
			"%v unable to connect to authenticated Websocket. Error: %s", b.Name, err)
	}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetEOptionsWsAuthStreamKey(context.TODO())
		switch {
		case err != nil:
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s", b.Name, err)
		default:
			wsURL = wsURL + "ws/" + listenKey
			err = b.Websocket.SetWebsocketURL(wsURL, false, false)
			if err != nil {
				return err
			}
		}
	}
	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", b.Name, err)
	}
	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})
	b.Websocket.Wg.Add(1)
	go b.wsEOptionsFuturesReadData()
	return nil
}

// GetEOptionsWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetEOptionsWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestOptions)
	if err != nil {
		return "", err
	}

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return "", err
	}

	var resp UserAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:        http.MethodPost,
		Path:          endpointPath + "/eapi/v1/listenKey",
		Headers:       headers,
		Result:        &resp,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}

	return resp.ListenKey, b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// wsEOptionsFuturesReadData receives and passes on websocket messages for processing
// for Coin margined instruments.
func (b *Binance) wsEOptionsFuturesReadData() {
	defer b.Websocket.Wg.Done()
	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleEOptionsData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Binance) wsHandleEOptionsData(respRaw []byte) error {
	result := struct {
		Result json.RawMessage `json:"result"`
		ID     int64           `json:"id"`
		Stream string          `json:"stream"`
		Data   json.RawMessage `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.Stream == "" || (result.ID != 0 && result.Result != nil) {
		if !b.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	var stream string
	switch result.Stream {
	case assetIndexAllChan, forceOrderAllChan, bookTickerAllChan, tickerAllChan, miniTickerAllChan:
		stream = result.Stream
	default:
		stream = extractStreamInfo(result.Stream)
	}
	switch stream {
	// case contractInfoAllChan:
	// 	return b.processContractInfoStream(result.Data)
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}
