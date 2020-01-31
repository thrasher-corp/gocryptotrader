package engine

import (
	"os"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// GetLastUpdated returns when the agent was last update
func (a *Agent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate returns when the agent needs to be updated
func (a *Agent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetNewUpdate sets agents last updated time and the updates the next required
// update time for REST protocol
func (a *Agent) SetNewUpdate() {
	a.LastUpdated = time.Now()
	if a.RestUpdateDelay == 0 {
		// TODO: Address
		log.Errorf(log.SyncMgr, "%s item RestUpdateDelay not set", a.Name)
		os.Exit(1)
	}
	a.NextUpdate = a.LastUpdated.Add(a.RestUpdateDelay)
}

// IsProcessing checks if agent is being processed by the REST protocol
func (a *Agent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing sets if agent is being processed by the REST protocol
func (a *Agent) SetProcessing(b bool) {
	a.Processing = b
}

// InitialSyncComplete sets initial sync to complete
func (a *Agent) InitialSyncComplete() {
	a.Wg.Done()
}

// DisableREST disables REST
func (a *Agent) DisableREST() {
	a.Disabled = true
}

// EnableREST enables REST
func (a *Agent) EnableREST() {
	a.Disabled = false
}

// IsRESTDisabled returns if REST has been disabled
func (a *Agent) IsRESTDisabled() bool {
	return a.Disabled
}

// GetExchangeName returns the exchange name
func (a *Agent) GetExchangeName() string {
	return a.Exchange.GetName()
}

// GetAgentName returns the name of the agent
func (a *Agent) GetAgentName() string {
	return a.Name
}

// Execute gets the ticker from the REST protocol
func (a *TickerAgent) Execute() {
	t, err := a.Exchange.UpdateTicker(a.Pair, a.AssetType)
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  t,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *TickerAgent) Stream(payload interface{}) Synchroniser {
	t, ok := payload.(*ticker.Price)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), t.ExchangeName) &&
		a.AssetType == t.AssetType &&
		a.Pair == t.Pair {
		return a
	}

	return nil
}

// Execute gets the orderbook from the REST protocol
func (a *OrderbookAgent) Execute() {
	o, err := a.Exchange.UpdateOrderbook(a.Pair, a.AssetType)
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  o,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *OrderbookAgent) Stream(payload interface{}) Synchroniser {
	o, ok := payload.(*orderbook.Base)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), o.ExchangeName) &&
		a.AssetType == o.AssetType &&
		a.Pair == o.Pair {
		return a
	}

	return nil
}

// Execute gets the trades from the REST protocol
func (a *TradeAgent) Execute() {
	t, err := a.Exchange.UpdateTrades(a.Pair, a.AssetType)
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  t,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *TradeAgent) Stream(payload interface{}) Synchroniser {
	t, ok := payload.([]order.TradeHistory)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), t[0].Exchange) {
		return a
	}

	return nil
}

// Execute gets the account balances from the REST protocol
func (a *AccountBalanceAgent) Execute() {
	acc, err := a.Exchange.UpdateAccountInfo()
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  &acc,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *AccountBalanceAgent) Stream(payload interface{}) Synchroniser {
	acc, ok := payload.(account.Holdings)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), acc.Exchange) {
		return a
	}

	return nil
}

// Execute gets the account orders from the REST protocol
func (a *OrderAgent) Execute() {
	o, err := a.Exchange.GetActiveOrders(&order.GetOrdersRequest{
		Pairs: []currency.Pair{a.Pair},
	})
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  o,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *OrderAgent) Stream(payload interface{}) Synchroniser {
	o, ok := payload.([]order.Detail)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), o[0].Exchange) {
		return a
	}

	return nil
}

// Execute gets the account fees from the REST protocol
func (a *SupportedPairsAgent) Execute() {
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  nil,
		Protocol: REST,
		Err:      a.Exchange.UpdateSupportedPairs()}
}

// Stream couples agent with incoming stream data
func (a *SupportedPairsAgent) Stream(payload interface{}) Synchroniser {
	// Should not have a stream update
	return nil
}
