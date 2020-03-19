package engine

import (
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
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

// Lock mtx locks the agent
func (a *Agent) Lock() {
	a.mtx.Lock()
}

// Unlock mtx unlocks the agent
func (a *Agent) Unlock() {
	a.mtx.Unlock()
}

// Cancel cancels REST component
func (a *Agent) Cancel() {
	a.Cancelled = true
}

// Clear resets cancelled REST component
func (a *Agent) Clear() {
	a.Cancelled = false
}

// IsCancelled returns if the agent item has been cancelled thus reducing REST
// calls
func (a *Agent) IsCancelled() bool {
	return a.Cancelled
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
	t, ok := payload.(order.TradeHistory)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), t.Exchange) &&
		a.Pair.Equal(t.Pair) && a.AssetType == t.AssetType {
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
