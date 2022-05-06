package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceusDefaultWebsocketURL = "wss://stream.binance.us:9443/stream"
	pingDelay                    = time.Minute * 9
)

var listenKey string

var (
	// maxWSUpdateBuffer defines max websocket updates to apply when an
	// orderbook is initially fetched
	maxWSUpdateBuffer = 150
	// maxWSOrderbookJobs defines max websocket orderbook jobs in queue to fetch
	// an orderbook snapshot via REST
	maxWSOrderbookJobs = 2000
	// maxWSOrderbookWorkers defines a max amount of workers allowed to execute
	// jobs from the job channel
	maxWSOrderbookWorkers = 10
)

// WsConnect initiates a websocket connection
func (b *Binanceus) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetWsAuthStreamKey(context.TODO())
		if err != nil {
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
		} else {
			// cleans on failed connection
			clean := strings.Split(b.Websocket.GetWebsocketURL(), "?streams=")
			authPayload := clean[0] + "?streams=" + listenKey
			err = b.Websocket.SetWebsocketURL(authPayload, false, false)
			if err != nil {
				return err
			}
		}
	}
	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	if b.Websocket.CanUseAuthenticatedEndpoints() {
		go b.KeepAuthKeyAlive()
	}

	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	b.Websocket.Wg.Add(1)
	// go b.wsReadData()

	// b.setupOrderbookManager()
	return nil
}

// KeepAuthKeyAlive will continuously send messages to
// keep the WS auth key active
func (b *Binanceus) KeepAuthKeyAlive() {
	b.Websocket.Wg.Add(1)
	defer b.CloseUserDataStream(context.Background())
	defer b.Websocket.Wg.Done()
	// Looping in 30 seconds and updating the listenKey
	ticks := time.NewTicker(time.Minute * 30)
	for {
		select {
		case <-b.Websocket.ShutdownC:
			ticks.Stop()
			return
		case <-ticks.C:
			err := b.MaintainWsAuthStreamKey(context.TODO())
			if err != nil {
				b.Websocket.DataHandler <- err
				log.Warnf(log.ExchangeSys,
					b.Name+" - Unable to renew auth websocket token, may experience shutdown")
			}
		}
	}
}
