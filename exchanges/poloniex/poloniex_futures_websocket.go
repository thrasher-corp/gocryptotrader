package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	fCnlTicker               = "/contractMarket/ticker"
	fCnlLevel2Orderbook      = "/contractMarket/level2"
	fCnlContractExecution    = "/contractMarket/execution"
	fCnlMarketExecution      = "/contractMarket/level3v2"
	fCnlOrderbookLvl2Depth5  = "/contractMarket/level2Depth5"
	fCnlOrderbookLvl2Depth50 = "/contractMarket/level2Depth50"
	fCnlInstruments          = "/contract/instrument"
	fCnlAnnouncement         = "/contract/announcement"
	fCnlContractMarket       = "/contractMarket/snapshot"
)

var defaultFuturesChannels = []string{
	fCnlTicker,
	fCnlOrderbookLvl2Depth50,
	fCnlInstruments,
}

// WsFuturesConnect establishes a websocket connection to the futures websocket server.
func (p *Poloniex) WsFuturesConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var instanceServers *FuturesWebsocketServerInstances
	var err error
	switch {
	case p.Websocket.CanUseAuthenticatedEndpoints():
		instanceServers, err = p.GetPrivateFuturesWebsocketServerInstances(context.Background())
		if err != nil {
			log.Warnf(log.ExchangeSys, err.Error())
			p.Websocket.SetCanUseAuthenticatedEndpoints(false)
			break
		}
		fallthrough
	default:
		instanceServers, err = p.GetPublicFuturesWebsocketServerInstances(context.Background())
		if err != nil {
			return err
		}
	}
	var dialer websocket.Dialer
	err = p.Websocket.SetWebsocketURL(instanceServers.Data.InstanceServers[0].Endpoint+"?token="+instanceServers.Data.Token+"&acceptUserMessage=true", false, false)
	if err != nil {
		return err
	}
	err = p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage := &struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}{
		ID:   "1",
		Type: "ping",
	}
	var pingPayload []byte
	pingPayload, err = json.Marshal(pingMessage)
	if err != nil {
		return err
	}
	p.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.TextMessage,
		Message:           pingPayload,
		Delay:             30,
	})
	p.Websocket.Wg.Add(1)
	go p.wsFuturesReadData(p.Websocket.Conn)
	return nil
}

// wsFuturesReadData handles data from the websocket connection for futures instruments subscriptions.
func (p *Poloniex) wsFuturesReadData(conn stream.Connection) {
	defer p.Websocket.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := p.wsFuturesHandleData(resp.Raw)
		if err != nil {
			p.Websocket.DataHandler <- fmt.Errorf("%s: %w", p.Name, err)
		}
	}
}

func (p *Poloniex) wsFuturesHandleData(respRaw []byte) error {
	var result *FuturesSubscriptionResp
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.ID != "" {
		if !p.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return fmt.Errorf("could not match trade response with ID: %s Event: %s ", result.ID, result.Topic)
		}
		return nil
	}
	if result.Topic != "" {
		log.Debugf(log.ExchangeSys, string(respRaw))
		return nil
	}
	switch result.Topic {
	case fCnlTicker, fCnlLevel2Orderbook, fCnlContractExecution, fCnlMarketExecution,
		fCnlOrderbookLvl2Depth5, fCnlOrderbookLvl2Depth50, fCnlInstruments,
		fCnlAnnouncement, fCnlContractMarket:
		// TODO: ...
	default:
		p.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: p.Name + stream.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", p.Name, string(respRaw))
	}
	return nil
}

// ------------------------------------------------------------------------------------------------

// GenerateFuturesDefaultSubscriptions adds default subscriptions to futures websockets.
func (p *Poloniex) GenerateFuturesDefaultSubscriptions() (subscription.List, error) {
	enabledCurrencies, err := p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	channels := defaultFuturesChannels
	subscriptions := make(subscription.List, 0, len(enabledCurrencies))
	for i := range channels {
		// TODO: ...
		println(i)
	}
	return subscriptions, nil
}

func (p *Poloniex) handleFuturesSubscriptions(operation string, subscs subscription.List) ([]FuturesSubscriptionInput, error) {
	payloads := []FuturesSubscriptionInput{}
	for x := range subscs {
		payloads = append(payloads, FuturesSubscriptionInput{
			ID:    p.Websocket.Conn.GenerateMessageID(false),
			Type:  operation,
			Topic: subscs[x].Channel + ":" + subscs[x].Pairs[0].String(),
		})
	}
	return payloads, nil
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (p *Poloniex) SubscribeFutures(subs subscription.List) error {
	payloads, err := p.handleFuturesSubscriptions("subscribe", subs)
	if err != nil {
		return err
	}
	for i := range payloads {
		// TODO:
		println(i)
	}
	return p.Websocket.AddSuccessfulSubscriptions(subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (p *Poloniex) UnsubscribeFutures(unsub subscription.List) error {
	payloads, err := p.handleFuturesSubscriptions("unsubscribe", unsub)
	if err != nil {
		return err
	}
	for i := range payloads {
		// TODO:
		println(i)
	}
	return p.Websocket.RemoveSubscriptions(unsub...)
}
