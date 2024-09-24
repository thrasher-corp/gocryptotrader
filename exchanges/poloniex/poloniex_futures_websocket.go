package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	fCnlTicker               = "/contractMarket/ticker"
	fCnlLevel2Orderbook      = "/contractMarket/level2"
	fCnlContractExecution    = "/contractMarket/execution"
	fCnlLvl3Orderbook        = "/contractMarket/level3v2"
	fCnlOrderbookLvl2Depth5  = "/contractMarket/level2Depth5"
	fCnlOrderbookLvl2Depth50 = "/contractMarket/level2Depth50"
	fCnlInstruments          = "/contract/instrument"
	fCnlAnnouncement         = "/contract/announcement"
	fCnlTickerSnapshot       = "/contractMarket/snapshot"

	// Private channels

	fCnlTradeOrders       = "/contractMarket/tradeOrders"
	fCnlAdvancedOrders    = "/contractMarket/advancedOrders"
	fCnlWallet            = "/contractAccount/wallet"
	fCnlContractPositions = "/contract/position"
	fCnlCrossPositionInfo = "/contract/positionCross"
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
			log.Warnf(log.ExchangeSys, "Unexpected authenticated futures websocket servers instance fetch error %v", err)
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
	p.Websocket.Conn.SetupPingHandler(request.UnAuth, stream.PingHandler{
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
		if result.ID == "1" {
			// Handling ping messages.
			return nil
		}
		if !p.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return fmt.Errorf("could not match trade response with ID: %s Event: %s ", result.ID, result.Topic)
		}
		return nil
	}
	topicSplit := strings.Split(result.Topic, ":")
	if len(topicSplit) != 1 && len(topicSplit) != 2 {
		p.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: p.Name + stream.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", p.Name, string(respRaw))
	}
	switch topicSplit[0] {
	case fCnlTicker:
		return p.processFuturesWsTicker(result)
	case fCnlLevel2Orderbook:
		return p.processFuturesWsOrdderbook(topicSplit[1], result)
	case fCnlContractExecution:
		return p.processOrderFills(topicSplit[1], result)
	case fCnlLvl3Orderbook:
		return p.processV3FuturesLevel3Orderbook(result)
	case fCnlOrderbookLvl2Depth5,
		fCnlOrderbookLvl2Depth50:
		return p.processOrderbookLvl2Depth5(topicSplit[1], result)
	case fCnlInstruments:
		switch result.Subject {
		case "mark.index.price":
			var resp *InstrumentMarkAndIndexPrice
			err := json.Unmarshal(result.Data, &resp)
			if err != nil {
				return err
			}
			p.Websocket.DataHandler <- resp
		case "funding.rate":
			var resp *WsFuturesInstrumentFundingRate
			err := json.Unmarshal(result.Data, &resp)
			if err != nil {
				return err
			}
			p.Websocket.DataHandler <- resp
		}
		return nil
	case fCnlAnnouncement:
		switch result.Subject {
		case "funding.end", "funding.begin":
			var resp *WsSystemAnnouncement
			err := json.Unmarshal(result.Data, &resp)
			if err != nil {
				return err
			}
			p.Websocket.DataHandler <- resp
			return nil
		}
	case fCnlTickerSnapshot:
		return p.processFuturesTickerSnapshot(topicSplit[1], result)

		// Private channels.
	case fCnlTradeOrders:
		return p.processFuturesUserTrades(result)
	case fCnlAdvancedOrders:
		return p.processFuturesStopOrderLifecycleEvent(result)
	case fCnlWallet:
		return p.processFuturesAccountBalance(result)
	case fCnlContractPositions:
		return p.processFuturesPositionChange(topicSplit[1], result)
	case fCnlCrossPositionInfo:
		return p.processFuturesPositionInfo(result)
	default:
		p.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: p.Name + stream.UnhandledMessage + string(respRaw)}
		return fmt.Errorf("%s unhandled message: %s", p.Name, string(respRaw))
	}
	return nil
}

func (p *Poloniex) processFuturesPositionInfo(resp *FuturesSubscriptionResp) error {
	var result *WsFuturesPositionInfo
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- result
	return nil
}

func (p *Poloniex) processFuturesPositionChange(pairString string, resp *FuturesSubscriptionResp) error {
	cp, err := currency.NewPairFromString(pairString)
	if err != nil {
		return err
	}
	switch resp.Subject {
	case "position.change":
		var result *WsFuturesPositionChange
		err := json.Unmarshal(resp.Data, &result)
		if err != nil {
			return err
		}
		result.Symbol = cp
		p.Websocket.DataHandler <- result
	case "position.settlement":
		var result *WsFuturesFundingSettlement
		err := json.Unmarshal(resp.Data, &result)
		if err != nil {
			return err
		}
		result.Symbol = cp
		p.Websocket.DataHandler <- result
	}
	return nil
}

func (p *Poloniex) processFuturesAccountBalance(resp *FuturesSubscriptionResp) error {
	var result *WsFuturesAccountBalance
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- account.Change{
		Exchange: p.Name,
		Currency: currency.NewCode(result.Currency),
		Asset:    asset.Futures,
		Amount:   result.AvailableBalance,
	}
	return nil
}

func (p *Poloniex) processFuturesStopOrderLifecycleEvent(resp *FuturesSubscriptionResp) error {
	var result *WsFuturesStopOrderLifecycleEvent
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(result.Symbol)
	if err != nil {
		return err
	}
	oSide, err := order.StringToOrderSide(result.Side)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- &order.Detail{
		Price:       result.OrderPrice.Float64(),
		Amount:      result.Size.Float64(),
		Exchange:    p.Name,
		OrderID:     result.OrderID,
		Type:        order.Stop,
		Side:        oSide,
		AssetType:   asset.Futures,
		CloseTime:   result.CreatedAt.Time(),
		LastUpdated: result.Timestamp.Time(),
		Pair:        cp,
	}
	return nil
}

func (p *Poloniex) processFuturesUserTrades(resp *FuturesSubscriptionResp) error {
	var result *WsFuturesTradeOrders
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(result.Symbol)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(result.OrderType)
	if err != nil {
		return err
	}
	oStatus, err := order.StringToOrderStatus(result.Status)
	if err != nil {
		return err
	}
	oSide, err := order.StringToOrderSide(result.Side)
	if err != nil {
		return err
	}
	oMarginType := margin.Isolated
	if result.MarginType == 0 {
		oMarginType = margin.Multi
	}
	p.Websocket.DataHandler <- &order.Detail{
		Price:                result.Price.Float64(),
		Amount:               result.Size.Float64(),
		AverageExecutedPrice: result.MatchPrice.Float64(),
		ExecutedAmount:       result.FilledSize.Float64(),
		RemainingAmount:      result.RemainSize.Float64(),
		Exchange:             p.Name,
		OrderID:              result.OrderID,
		ClientOrderID:        result.ClientOid,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            asset.Futures,
		CloseTime:            result.Timestamp.Time(),
		Pair:                 cp,
		MarginType:           oMarginType,
	}
	return nil
}

func (p *Poloniex) processFuturesTickerSnapshot(pairString string, resp *FuturesSubscriptionResp) error {
	var result *WsFuturesTicker
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(pairString)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- ticker.Price{
		Last:         result.LastPrice,
		Volume:       result.Volume24Hr,
		Pair:         cp,
		ExchangeName: p.Name,
		AssetType:    asset.Futures,
		LastUpdated:  result.SnapshotTime.Time(),
	}
	return nil
}

// orderbookSnapshotLoadedPairsMap used to check pair which has pair snapshots added.
var orderbookSnapshotLoadedPairsMap = map[string]bool{}

func (p *Poloniex) processOrderbookLvl2Depth5(pairString string, resp *FuturesSubscriptionResp) error {
	var result *WsFuturesLevel2Depth5OB
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(pairString)
	if err != nil {
		return err
	}
	asks := make(orderbook.Tranches, len(result.Asks))
	for i := range result.Asks {
		asks[i] = orderbook.Tranche{Price: result.Asks[i][0].Float64(), Amount: result.Asks[i][1].Float64()}
	}
	bids := make(orderbook.Tranches, len(result.Bids))
	for i := range result.Bids {
		bids[i] = orderbook.Tranche{Price: result.Bids[i][0].Float64(), Amount: result.Bids[i][1].Float64()}
	}
	found, okay := orderbookSnapshotLoadedPairsMap[pairString]
	if !found || !okay {
		orderbookSnapshotLoadedPairsMap[pairString] = true
		base := &orderbook.Base{
			Exchange:    p.Name,
			Pair:        cp,
			Asset:       asset.Futures,
			LastUpdated: result.Timestamp.Time(),
			Asks:        asks,
			Bids:        bids,
		}
		return p.Websocket.Orderbook.LoadSnapshot(base)
	}
	return p.Websocket.Orderbook.Update(&orderbook.Update{
		UpdateTime: result.Timestamp.Time(),
		Asset:      asset.Futures,
		Bids:       bids,
		Asks:       asks,
		Pair:       cp,
	})
}

func (p *Poloniex) processV3FuturesLevel3Orderbook(resp *FuturesSubscriptionResp) error {
	var result interface{}
	switch resp.Subject {
	case "received":
		result = &WsOrderbookLevel3V2{}
	case "open":
		result = &WsOrderbookOpen{}
	case "update":
		result = &WsOrderbookUpdateOrder{}
	case "match":
		result = &WsOrderbookMatch{}
	case "done":
		result = &WsOrderbookMatchDone{}
	default:
		return fmt.Errorf("unhandled websocket data %s", string(resp.Data))
	}
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- result
	return nil
}

func (p *Poloniex) processOrderFills(pairString string, resp *FuturesSubscriptionResp) error {
	var result *WsOrderFill
	cp, err := currency.NewPairFromString(pairString)
	if err != nil {
		return err
	}
	err = json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	oSide, err := order.StringToOrderSide(result.Side)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- []fill.Data{{
		ID:            result.TradeID,
		Timestamp:     result.FilledTime.Time(),
		Exchange:      p.Name,
		AssetType:     asset.Futures,
		CurrencyPair:  cp,
		Side:          oSide,
		OrderID:       result.MakerOrderID,
		ClientOrderID: result.TakerOrderID,
		TradeID:       result.TradeID,
		Price:         result.Price,
		Amount:        result.MatchSize,
	}}
	return nil
}

func (p *Poloniex) processFuturesWsOrdderbook(pairString string, resp *FuturesSubscriptionResp) error {
	var result *WsFuturesLvl2Orderbook
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(pairString)
	if err != nil {
		return err
	}

	// Check and load orderbook snapshot for an asset type and currency pair.
	base, err := p.Websocket.Orderbook.GetOrderbook(cp, asset.Futures)
	if err != nil {
		var ob *Orderbook
		ob, err = p.GetFullOrderbookLevel2(context.Background(), pairString)
		if err != nil {
			return err
		}
		base, err = ob.GetOBBase()
		if err != nil {
			return err
		}
		base.Exchange = p.Name
	}
	if result.LastSequence <= base.LastUpdateID {
		// Discard the feed data of a sequence that is below or equals to the last orderbook snapshot sequence
		return nil
	}
	for i := range result.Changes {
		split := strings.Split(result.Changes[i], ",")
		if len(split) != 3 {
			continue
		}
		var price, amount float64
		price, err = strconv.ParseFloat(split[0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(split[2], 64)
		if err != nil {
			return err
		}
		switch split[1] {
		case "sell":
			found := false
			for j := range base.Asks {
				if price == base.Asks[j].Price {
					if amount == 0 {
						base.Asks = append(base.Asks[:j], base.Asks[j+1:]...)
					} else {
						base.Asks[j].Amount = amount
					}
					found = true
				}
			}
			if !found {
				base.Asks = append(base.Asks, orderbook.Tranche{Price: price, Amount: amount})
			}
		case "buy":
			found := false
			for j := range base.Bids {
				if price == base.Bids[j].Price {
					if amount == 0 {
						base.Bids = append(base.Bids[:j], base.Bids[j+1:]...)
					} else {
						base.Bids[j].Amount = amount
					}
					found = true
				}
			}
			if !found {
				base.Bids = append(base.Bids, orderbook.Tranche{Price: price, Amount: amount})
			}
		default:
			continue
		}
	}
	err = base.Process()
	if err != nil {
		return err
	}
	return p.Websocket.Orderbook.LoadSnapshot(base)
}

func (p *Poloniex) processFuturesWsTicker(resp *FuturesSubscriptionResp) error {
	var result *WsFuturesTickerInfo
	err := json.Unmarshal(resp.Data, &result)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairFromString(result.Symbol)
	if err != nil {
		return err
	}
	p.Websocket.DataHandler <- ticker.Price{
		Bid:          result.BestBidPrice,
		BidSize:      result.BestBidSize,
		Ask:          result.BestAskPrice,
		Volume:       result.Size,
		IndexPrice:   result.Price,
		Pair:         cp,
		ExchangeName: p.Name,
		AssetType:    asset.Futures,
		LastUpdated:  result.FilledTime.Time(),
	}
	return nil
}

// ------------------------------------------------------------------------------------------------

// GenerateFuturesDefaultSubscriptions adds default subscriptions to futures websockets.
func (p *Poloniex) GenerateFuturesDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	channels := defaultFuturesChannels
	subscriptions := subscription.List{}
	for i := range channels {
		switch channels[i] {
		case fCnlCrossPositionInfo,
			fCnlWallet,
			fCnlAdvancedOrders,
			fCnlTradeOrders:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:       channels[i],
				Asset:         asset.Futures,
				Authenticated: true,
			})
		case fCnlTicker,
			fCnlLevel2Orderbook,
			fCnlContractExecution,
			fCnlLvl3Orderbook,
			fCnlOrderbookLvl2Depth5,
			fCnlOrderbookLvl2Depth50,
			fCnlInstruments,
			fCnlAnnouncement,
			fCnlTickerSnapshot,
			fCnlContractPositions:
			authenticated := false
			if channels[i] == fCnlContractPositions {
				authenticated = true
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:       channels[i],
				Asset:         asset.Futures,
				Pairs:         enabledPairs,
				Authenticated: authenticated,
			})
		}
	}
	return subscriptions, nil
}

func (p *Poloniex) handleFuturesSubscriptions(operation string, subscs subscription.List) []FuturesSubscriptionInput {
	payloads := []FuturesSubscriptionInput{}
	for x := range subscs {
		if len(subscs[x].Pairs) == 0 {
			input := FuturesSubscriptionInput{
				ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
				Type:  operation,
				Topic: subscs[x].Channel,
			}
			payloads = append(payloads, input)
		} else {
			for i := range subscs[x].Pairs {
				input := FuturesSubscriptionInput{
					ID:    strconv.FormatInt(p.Websocket.Conn.GenerateMessageID(false), 10),
					Type:  operation,
					Topic: subscs[x].Channel,
				}
				if !subscs[x].Pairs[x].IsEmpty() {
					input.Topic += ":" + subscs[x].Pairs[i].String()
				}
				payloads = append(payloads, input)
			}
		}
	}
	return payloads
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (p *Poloniex) SubscribeFutures(subs subscription.List) error {
	payloads := p.handleFuturesSubscriptions("subscribe", subs)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.AddSuccessfulSubscriptions(subs...)
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (p *Poloniex) UnsubscribeFutures(unsub subscription.List) error {
	payloads := p.handleFuturesSubscriptions("unsubscribe", unsub)
	var err error
	for i := range payloads {
		err = p.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[i])
		if err != nil {
			return err
		}
	}
	return p.Websocket.RemoveSubscriptions(unsub...)
}
