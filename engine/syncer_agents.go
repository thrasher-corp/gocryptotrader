package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Agent defines our core fields to implement the sychroniser interface.
// To add additional agents requires the new struct to imbed an agent and
// define an execution method and stream method
type Agent struct {
	Exchange        exchange.IBotExchange
	Processing      bool
	NextUpdate      time.Time
	LastUpdated     time.Time
	RestUpdateDelay time.Duration
	Pipe            chan SyncUpdate
	Wg              *sync.WaitGroup
	CancelMe        chan int
}

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
		panic("RestUpdateDelay not set")
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

// Cancel cancels job on the job stack
func (a *Agent) Cancel() {
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// TickerAgent synchronises the exchange currency pair ticker
type TickerAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// Execute gets the ticker from the REST protocol
func (a *TickerAgent) Execute() {
	t, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).UpdateTicker(a.Pair, a.AssetType, a.CancelMe)

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

// OrderbookAgent synchronises the exchange currency pair orderbook
type OrderbookAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// Execute gets the orderbook from the REST protocol
func (a *OrderbookAgent) Execute() {
	o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).UpdateOrderbook(a.Pair, a.AssetType, a.CancelMe)

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

// TradeAgent synchronises the exchange currency pair trades
type TradeAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// Execute gets the trades from the REST protocol
func (a *TradeAgent) Execute() {
	t, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).UpdateTrades(a.Pair, a.AssetType, a.CancelMe)

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  t,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *TradeAgent) Stream(payload interface{}) Synchroniser {
	t, ok := payload.([]order.Trade)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), t[0].Exchange) {
		return a
	}

	return nil
}

// AccountAgent synchronises the exchange account balances
type AccountAgent struct {
	Agent
}

// Execute gets the account balances from the REST protocol
func (a *AccountAgent) Execute() {
	acc, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).GetAccountInfo(a.CancelMe)

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  &acc,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *AccountAgent) Stream(payload interface{}) Synchroniser {
	acc, ok := payload.(*exchange.AccountInfo)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), acc.Exchange) {
		return a
	}

	return nil
}

// OrderAgent synchronises the exchange account orders
type OrderAgent struct {
	Agent
}

// Execute gets the account orders from the REST protocol
func (a *OrderAgent) Execute() {
	o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).GetActiveOrders(&order.GetOrdersRequest{}, a.CancelMe)

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

// FeeAgent synchronises the exchange account fees
type FeeAgent struct {
	Agent
}

// Execute gets the account fees from the REST protocol
func (a *FeeAgent) Execute() {
	panic("Fees not completed")
	// TODO: Fee structure type need to be reworked
	// o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).GetFeeByType(nil)

	// a.Pipe <- SyncUpdate{
	// 	Agent:   a,
	// 	Payload: o,
	// 	Procotol: REST,
	// 	Err:     err}
}

// Stream couples agent with incoming stream data
func (a *FeeAgent) Stream(payload interface{}) Synchroniser {
	return nil
}

// SupportedPairsAgent synchronises the exchange supported currency pairs
type SupportedPairsAgent struct {
	Agent
}

// Execute gets the account fees from the REST protocol
func (a *SupportedPairsAgent) Execute() {
	panic("Supported pairs not completed")
	// TODO: Add in check supported pairs, update every hour
	// o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).

	// a.Pipe <- SyncUpdate{
	// 	Agent:   a,
	// 	Payload: &o,
	//  Procotol: REST,
	// 	Err:     err}
}

// Stream couples agent with incoming stream data
func (a *SupportedPairsAgent) Stream(payload interface{}) Synchroniser {
	// Should not have a stream update
	return nil
}

// ExchangeTradeHistoryAgent implements the synchroniser interface
type ExchangeTradeHistoryAgent struct {
	Agent
	Pair      currency.Pair
	AssetType asset.Item
}

// Execute gets the exchange trade history from the REST protocol
func (a *ExchangeTradeHistoryAgent) Execute() {
	panic("ExchangeTradeHistory not completed")
	// TODO: Add in exchange history support with configuration params
	h, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).GetExchangeHistory(&exchange.TradeHistoryRequest{
		Pair:  a.Pair,
		Asset: a.AssetType,
	}, a.CancelMe)

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  h,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *ExchangeTradeHistoryAgent) Stream(payload interface{}) Synchroniser {
	h, ok := payload.([]exchange.TradeHistory)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), h[0].Exchange) &&
		a.AssetType == h[0].AssetType &&
		a.Pair == h[0].Pair {
		return a
	}

	return nil
}

// KlineAgent meow
type KlineAgent struct {
	Agent
	Pair      currency.Pair
	AssetType asset.Item
}

// Execute ...
func (a *KlineAgent) Execute() {
	// TODO: Add in KlineAgent support with configuration params
	start := time.Now()
	// h, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).

	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Exchange Kline item took [%s] to update \n",
			end.Sub(start))
	}
	// a.Pipe <- SyncUpdate{
	// 	Agent:    a,
	// 	Payload:  h,
	// 	Protocol: REST,
	// 	Err:      err}
}

// Stream couples agent with incoming stream data
func (a *KlineAgent) Stream(payload interface{}) Synchroniser {
	h, ok := payload.([]exchange.TradeHistory)
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), h[0].Exchange) &&
		a.AssetType == h[0].AssetType &&
		a.Pair == h[0].Pair {
		return a
	}

	return nil
}

// DepositAddressAgent synchronises the deposit addresses available on the
// exchange
type DepositAddressAgent struct {
	Agent
	Currency  currency.Code
	AccountID string
}

// Execute gets the deposit address for the agent
func (a *DepositAddressAgent) Execute() {
	start := time.Now()
	address, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID,
		a.Exchange).GetDepositAddress(a.Currency, a.AccountID, a.Agent.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Exchange Kline item took [%s] to update \n",
			end.Sub(start))
	}
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  address,
		Protocol: REST,
		Err:      err}
}

// Stream couples agent with incoming stream data
func (a *DepositAddressAgent) Stream(payload interface{}) Synchroniser {
	// Should not match
	return nil
}
