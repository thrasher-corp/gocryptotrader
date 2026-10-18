package bybit

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const spreadPublicPath = "/v5/public/spread"

// Spread trading websocket channels. The public channels are subscribed on the spread connection, and a subscription
// names its spread symbol in Params["symbol"]; an orderbook subscription takes its depth from Levels, 25 by default.
// The private channels are Bybit's topic names and travel on the private connection
const (
	ChannelSpreadOrderbook   = "spread.orderbook"
	ChannelSpreadPublicTrade = "spread.publicTrade"
	ChannelSpreadTicker      = "spread.tickers"
	ChannelSpreadOrder       = "spread.order"
	ChannelSpreadExecution   = "spread.execution"
)

const defaultTopicOrderbookDepth = 25

// Websocket topic errors
var (
	errSubscriptionSymbolMissing = errors.New("subscription symbol param missing")
	errTopicsRejected            = errors.New("subscription topics rejected")
)

// SpreadPublicConnectionSetup returns the setup of the spread trading public connection to url. It generates
// subscriptions only for configured spread channels, so it connects only when a user subscribes to one
func (e *Exchange) SpreadPublicConnectionSetup(exch *config.Exchange, url string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   url,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WebsocketConnect,
		GenerateSubscriptions: e.generateSpreadPublicSubscriptions,
		Subscriber:            e.spreadPublicSubscribe,
		Unsubscriber:          e.spreadPublicUnsubscribe,
		Handler:               e.wsHandleSpreadPublicData,
	}
}

// generateSpreadPublicSubscriptions returns the enabled subscriptions to spread public channels, qualified by topic. A
// subscription without a usable symbol is left out with a warning, since Setup disables the configured ones
func (e *Exchange) generateSpreadPublicSubscriptions() (subscription.List, error) {
	var subs subscription.List
	for _, s := range e.Features.Subscriptions.Enabled() {
		var prefix string
		switch s.Channel {
		case ChannelSpreadOrderbook:
			prefix = "orderbook." + strconv.Itoa(topicOrderbookDepth(s)) + "."
		case ChannelSpreadPublicTrade:
			prefix = "publicTrade."
		case ChannelSpreadTicker:
			prefix = "tickers."
		default:
			continue
		}
		symbol, err := subscriptionSymbol(s)
		if err != nil {
			// Setup disables the configured subscriptions without a symbol, so this one was added since; it is left out
			// rather than allowed to stop every connection
			log.Warnf(log.WebsocketMgr, "%s skipped a subscription: %v", e.Name, err)
			continue
		}
		subs = append(subs, topicSubscription(s, prefix+symbol))
	}
	return subs, nil
}

// spreadPublicSubscribe subscribes to topics on the spread public connection
func (e *Exchange) spreadPublicSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageCommandSubscriptions(ctx, conn, "subscribe", subs)
}

// spreadPublicUnsubscribe unsubscribes from topics on the spread public connection
func (e *Exchange) spreadPublicUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageCommandSubscriptions(ctx, conn, "unsubscribe", subs)
}

// wsHandleSpreadPublicData relays the spread public connection's frames as decoded, storing tickers first. Orderbooks
// and trades are not stored: spread prices can be zero, which the orderbook store rejects, and negative, which the trade
// buffer turns into a positive sell
func (e *Exchange) wsHandleSpreadPublicData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return handleCommandFrame(conn, respRaw)
		}
		return err
	}
	switch {
	case strings.HasPrefix(topic, "orderbook."):
		return e.relayFrame(ctx, respRaw, new(WebsocketSpreadOrderbookResponse))
	case strings.HasPrefix(topic, "publicTrade."):
		return e.relayFrame(ctx, respRaw, new(WebsocketSpreadPublicTradeResponse))
	case strings.HasPrefix(topic, "tickers."):
		return e.wsProcessSpreadTicker(ctx, respRaw)
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// wsProcessSpreadTicker stores a spread ticker under asset.Spread, with the pair currency.NewPairFromString makes of its
// symbol as for OKX spreads, then relays the frame. The frame is relayed rather than the stored ticker because Bybit
// lists no spread asset, so the engine's sync manager would log each relayed ticker as an error
func (e *Exchange) wsProcessSpreadTicker(ctx context.Context, respRaw []byte) error {
	var frame WebsocketSpreadTickerResponse
	if err := json.Unmarshal(respRaw, &frame); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(frame.Data.Symbol)
	if err != nil {
		return err
	}
	// volume24h counts the base coin, in which spreads are sized
	if err := ticker.ProcessTicker(&ticker.Price{
		Last:         frame.Data.LastPrice.Float64(),
		High:         frame.Data.HighPrice24Hour.Float64(),
		Low:          frame.Data.LowPrice24Hour.Float64(),
		Bid:          frame.Data.BidPrice.Float64(),
		BidSize:      frame.Data.BidSize.Float64(),
		Ask:          frame.Data.AskPrice.Float64(),
		AskSize:      frame.Data.AskSize.Float64(),
		BaseVolume:   frame.Data.Volume24Hour.Float64(),
		Open24Hour:   frame.Data.PreviousPrice24Hour.Float64(),
		Pair:         pair,
		ExchangeName: e.Name,
		AssetType:    asset.Spread,
		LastUpdated:  frame.Timestamp.Time(),
	}); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &frame)
}

// wsHandleSpreadPrivateData relays the private connection's spread order and execution frames as decoded
func (e *Exchange) wsHandleSpreadPrivateData(ctx context.Context, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil {
		return err
	}
	switch topic {
	case ChannelSpreadOrder:
		return e.relayFrame(ctx, respRaw, new(WebsocketSpreadOrderResponse))
	case ChannelSpreadExecution:
		return e.relayFrame(ctx, respRaw, new(WebsocketSpreadExecutionResponse))
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// relayFrame decodes a frame into frame and relays it
func (e *Exchange) relayFrame(ctx context.Context, respRaw []byte, frame any) error {
	if err := json.Unmarshal(respRaw, frame); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, frame)
}

// topicKey keys a subscription by the Bybit topic it subscribes to, for connections whose subscriptions name a Bybit
// symbol rather than an asset and pairs
type topicKey struct {
	*subscription.Subscription
}

var _ subscription.MatchableKey = topicKey{}

// GetSubscription returns the underlying subscription
func (k topicKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

// String returns the subscription's channel and topic
func (k topicKey) String() string {
	if k.Subscription == nil {
		return "Uninitialised topicKey"
	}
	return k.Channel + " " + k.QualifiedChannel
}

// Match returns whether eachKey subscribes to the same channel and topic
func (k topicKey) Match(eachKey subscription.MatchableKey) bool {
	if k.Subscription == nil || eachKey == nil {
		return false
	}
	s := eachKey.GetSubscription()
	return s != nil && s.Channel == k.Channel && s.QualifiedChannel == k.QualifiedChannel
}

// topicSubscription returns a copy of s qualified and keyed by the Bybit topic it subscribes to
func topicSubscription(s *subscription.Subscription, topic string) *subscription.Subscription {
	c := s.Clone()
	c.QualifiedChannel = topic
	c.SetKey(topicKey{c})
	return c
}

// subscriptionSymbol returns the Bybit symbol a subscription names in Params["symbol"]
func subscriptionSymbol(s *subscription.Subscription) (string, error) {
	if symbol, ok := s.Params["symbol"].(string); ok && symbol != "" {
		return symbol, nil
	}
	return "", fmt.Errorf("%w: %s", errSubscriptionSymbolMissing, s.Channel)
}

// disableUnusableSubscriptions disables, with a warning, each configured authenticated subscription, since the private
// connection generates its own, and each configured spread or event subscription without a usable symbol, and then takes
// the enabled subscriptions from the configuration again. Setup runs it before any connection generates subscriptions:
// a generation error would stop every connection, and changing the configuration once the websocket runs would race
// with whatever reads it
func (e *Exchange) disableUnusableSubscriptions() {
	var disabled bool
	for _, s := range e.Config.Features.Subscriptions {
		if !s.Enabled {
			continue
		}
		if s.Authenticated {
			log.Warnf(log.WebsocketMgr, "%s has an authenticated subscription %q in config which is not supported. Please remove.", e.Name, s.Channel)
			s.Enabled = false
			disabled = true
			continue
		}
		if !needsSymbol(s.Channel) {
			continue
		}
		if _, err := subscriptionSymbol(s); err != nil {
			log.Warnf(log.WebsocketMgr, "%s disabled a subscription in config: %v. Please correct or remove it.", e.Name, err)
			s.Enabled = false
			disabled = true
		}
	}
	if disabled {
		e.SetSubscriptionsFromConfig()
	}
}

// needsSymbol reports whether subscriptions to a channel name their spread or event contract in Params["symbol"]
func needsSymbol(channel string) bool {
	switch channel {
	case ChannelSpreadOrderbook, ChannelSpreadPublicTrade, ChannelSpreadTicker, ChannelEventOrderbook, ChannelEventPublicTrade:
		return true
	}
	return false
}

// topicOrderbookDepth returns the orderbook depth a subscription asks for, which Bybit documents as 25 for spread and
// event contracts
func topicOrderbookDepth(s *subscription.Subscription) int {
	if s.Levels != 0 {
		return s.Levels
	}
	return defaultTopicOrderbookDepth
}

// manageCommandSubscriptions subscribes or unsubscribes topics in batches on a connection that answers with a
// COMMAND_RESP frame. A failed batch does not stop the batches after it; the error names every topic that failed
func (e *Exchange) manageCommandSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, batch := range common.Batch(subs, 10) {
		// Bybit lists a topic once in its reply however often a batch names it, and QualifiedChannels sorts the topics
		topics := slices.Compact(batch.QualifiedChannels())
		resp, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, commandSignature(topics), &SubscriptionRequest{
			RequestID: e.MessageID(),
			Operation: operation,
			Arguments: topics,
		})
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s: %w", operation, strings.Join(topics, ", "), err))
			continue
		}
		var reply CommandResponse
		if err := json.Unmarshal(resp, &reply); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		failTopics := reply.Data.FailTopics
		if !reply.Success {
			failTopics = topics
		}
		errs = common.AppendError(errs, e.applyTopicResults(conn, operation, batch, failTopics))
	}
	return errs
}

// applyTopicResults stores or removes the subscriptions whose topics did not fail, and returns an error naming the
// topics that did
func (e *Exchange) applyTopicResults(conn websocket.Connection, operation string, batch subscription.List, failTopics []string) error {
	var accepted subscription.List
	var failed []string
	for _, s := range batch {
		if slices.Contains(failTopics, s.QualifiedChannel) {
			failed = append(failed, s.QualifiedChannel)
			continue
		}
		accepted = append(accepted, s)
	}
	var err error
	if len(accepted) != 0 {
		if operation == "unsubscribe" {
			err = e.Websocket.RemoveSubscriptions(conn, accepted...)
		} else {
			err = e.Websocket.AddSuccessfulSubscriptions(conn, accepted...)
		}
	}
	if len(failed) != 0 {
		err = common.AppendError(err, fmt.Errorf("%w: %s %s", errTopicsRejected, operation, strings.Join(failed, ", ")))
	}
	return err
}

// commandSignature identifies a COMMAND_RESP frame by the topics of the request it answers, which it lists in any
// order
func commandSignature(topics []string) string {
	return "COMMAND_RESP " + strings.Join(slices.Sorted(slices.Values(topics)), ",")
}

// handleCommandFrame passes a spread or event connection's COMMAND_RESP frame to the request awaiting it; pongs need
// no handling
func handleCommandFrame(conn websocket.Connection, respRaw []byte) error {
	if op, err := jsonparser.GetString(respRaw, "op"); err == nil && op == "pong" {
		return nil
	}
	var reply CommandResponse
	if err := json.Unmarshal(respRaw, &reply); err != nil {
		return err
	}
	if reply.Type != "COMMAND_RESP" {
		return fmt.Errorf("%w: %s", errUnhandledStreamData, respRaw)
	}
	return conn.RequireMatchWithData(commandSignature(slices.Concat(reply.Data.SuccessTopics, reply.Data.FailTopics)), respRaw)
}
