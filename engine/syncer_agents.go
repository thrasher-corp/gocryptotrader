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

// TickerAgent implements the synchroniser interface
type TickerAgent struct {
	Exchange    exchange.IBotExchange
	AssetType   asset.Item
	Pair        currency.Pair
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *TickerAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *TickerAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *TickerAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *TickerAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *TickerAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *TickerAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *TickerAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *TickerAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *TickerAgent) Execute() {
	start := time.Now()
	t, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).UpdateTicker(a.Pair, a.AssetType, a.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Ticker item took [%s] to update \n",
			end.Sub(start))
	}

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  t,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *TickerAgent) Cancel() {
	fmt.Println("cancelling ticker update")
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *TickerAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
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

// OrderbookAgent implements the synchroniser interface
type OrderbookAgent struct {
	Exchange    exchange.IBotExchange
	Function    func()
	AssetType   asset.Item
	Pair        currency.Pair
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *OrderbookAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *OrderbookAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *OrderbookAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *OrderbookAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *OrderbookAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *OrderbookAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *OrderbookAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *OrderbookAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *OrderbookAgent) Execute() {
	start := time.Now()
	o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).UpdateOrderbook(a.Pair, a.AssetType, a.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Orderbook item took [%s] to update \n",
			end.Sub(start))
	}

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  o,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *OrderbookAgent) Cancel() {
	fmt.Println("cancelling orderbook update: ", a.Protocol, a.Pair, a.AssetType)
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *OrderbookAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
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

// TradeAgent implements the synchroniser interface
type TradeAgent struct {
	Exchange    exchange.IBotExchange
	Function    func()
	AssetType   asset.Item
	Pair        currency.Pair
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *TradeAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *TradeAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *TradeAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *TradeAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *TradeAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *TradeAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *TradeAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *TradeAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *TradeAgent) Execute() {
	start := time.Now()
	o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).UpdateOrderbook(a.Pair, a.AssetType, a.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Orderbook item took [%s] to update \n",
			end.Sub(start))
	}

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  &o,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *TradeAgent) Cancel() {
	fmt.Println("cancelling trade update: ", a.Protocol, a.Pair, a.AssetType)
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *TradeAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
func (a *TradeAgent) Stream(payload interface{}) Synchroniser {
	t, ok := payload.(*exchange.TradeHistory) // TODO: Change to correct type
	if !ok {
		return nil
	}

	if strings.EqualFold(a.Exchange.GetName(), t.Exchange) &&
		a.AssetType == t.AssetType &&
		a.Pair == t.Pair {
		return a
	}

	return nil
}

// AccountAgent implements the synchroniser interface
type AccountAgent struct {
	Exchange    exchange.IBotExchange
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *AccountAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *AccountAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *AccountAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *AccountAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *AccountAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *AccountAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *AccountAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *AccountAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *AccountAgent) Execute() {
	start := time.Now()
	acc, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).GetAccountInfo(a.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Account item took [%s] to update \n",
			end.Sub(start))
	}

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  &acc,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *AccountAgent) Cancel() {
	fmt.Println("cancelling account update: ", a.Protocol)
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *AccountAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
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

// OrderAgent implements the synchroniser interface
type OrderAgent struct {
	Exchange    exchange.IBotExchange
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *OrderAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *OrderAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *OrderAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *OrderAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *OrderAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *OrderAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *OrderAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *OrderAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *OrderAgent) Execute() {
	start := time.Now()
	o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).GetActiveOrders(nil, a.CancelMe)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Get Active Orders item took [%s] to update \n",
			end.Sub(start))
	}

	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  o,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *OrderAgent) Cancel() {
	fmt.Println("cancelling order update: ", a.Protocol)
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *OrderAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
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

// FeeAgent implements the synchroniser interface
type FeeAgent struct {
	Exchange    exchange.IBotExchange
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *FeeAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *FeeAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *FeeAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *FeeAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *FeeAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *FeeAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *FeeAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *FeeAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *FeeAgent) Execute() {
	start := time.Now()
	// TODO: Fee structure type need to be reworked
	// o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).GetFeeByType(nil)
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Fee item took [%s] to update \n",
			end.Sub(start))
	}
	// a.Pipe <- SyncUpdate{
	// 	Agent:   a,
	// 	Payload: o,
	// 	Procotol: syncProtocolREST,
	// 	Err:     err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *FeeAgent) Cancel() {
	fmt.Println("cancelling fee update: ", a.Protocol)
	select {
	case a.CancelMe <- 1:
	default:
		fmt.Println("failed to cancel")
	}
}

// InitialSyncComplete sets initial sync to complete
func (a *FeeAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
func (a *FeeAgent) Stream(payload interface{}) Synchroniser {
	fee, ok := payload.(float64) // TODO: Fix fee structure
	if !ok {
		return nil
	}

	fmt.Println("fee structure for account == something", fee)

	return nil
}

// SupportedPairsAgent implements the synchroniser interface
type SupportedPairsAgent struct {
	Exchange    exchange.IBotExchange
	Protocol    string
	Processing  bool
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *SupportedPairsAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *SupportedPairsAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *SupportedPairsAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *SupportedPairsAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *SupportedPairsAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *SupportedPairsAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *SupportedPairsAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *SupportedPairsAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *SupportedPairsAgent) Execute() {
	// TODO: Add in check supported pairs, update every hour
	start := time.Now()
	// o, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).
	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Supported pairs item took [%s] to update \n",
			end.Sub(start))
	}
	// a.Pipe <- SyncUpdate{
	// 	Agent:   a,
	// 	Payload: &o,
	//  Procotol: syncProtocolREST,
	// 	Err:     err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *SupportedPairsAgent) Cancel() {
}

// InitialSyncComplete sets initial sync to complete
func (a *SupportedPairsAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
func (a *SupportedPairsAgent) Stream(payload interface{}) Synchroniser {
	// Should not have a stream update
	return nil
}

// ExchangeTradeHistoryAgent implements the synchroniser interface
type ExchangeTradeHistoryAgent struct {
	Exchange    exchange.IBotExchange
	Protocol    string
	Processing  bool
	Pair        currency.Pair
	AssetType   asset.Item
	NextUpdate  time.Time
	LastUpdated time.Time
	Pipe        chan SyncUpdate
	Wg          *sync.WaitGroup
	CancelMe    chan int
}

// GetLastUpdated  ...
func (a *ExchangeTradeHistoryAgent) GetLastUpdated() time.Time {
	return a.LastUpdated
}

// GetNextUpdate ...
func (a *ExchangeTradeHistoryAgent) GetNextUpdate() time.Time {
	return a.NextUpdate
}

// SetLastUpdated ...
func (a *ExchangeTradeHistoryAgent) SetLastUpdated(t time.Time) {
	a.LastUpdated = t
}

// SetNextUpdate ...
func (a *ExchangeTradeHistoryAgent) SetNextUpdate(t time.Time) {
	a.NextUpdate = t
}

// IsUsingProtocol ...
func (a *ExchangeTradeHistoryAgent) IsUsingProtocol(protocol string) bool {
	return protocol == a.Protocol
}

// SetUsingProtocol ...
func (a *ExchangeTradeHistoryAgent) SetUsingProtocol(protocol string) {
	a.Protocol = protocol
}

// IsProcessing ...
func (a *ExchangeTradeHistoryAgent) IsProcessing() bool {
	return a.Processing
}

// SetProcessing ...
func (a *ExchangeTradeHistoryAgent) SetProcessing(b bool) {
	a.Processing = b
}

// Execute ...
func (a *ExchangeTradeHistoryAgent) Execute() {
	// TODO: Add in exchange history support with configuration params
	start := time.Now()
	h, err := Bot.WoRkMaNaGeR.Exchange(syncManagerUUID, a.Exchange).GetExchangeHistory(&exchange.TradeHistoryRequest{
		Pair:  a.Pair,
		Asset: a.AssetType,
	}, a.CancelMe)

	end := time.Now()
	if Bot.Settings.Verbose {
		log.Debugf(log.SyncMgr,
			"Exchange History item took [%s] to update \n",
			end.Sub(start))
	}
	a.Pipe <- SyncUpdate{
		Agent:    a,
		Payload:  h,
		Procotol: syncProtocolREST,
		Err:      err}
}

// Cancel cancels job so when sleep is done it cancels and resets
func (a *ExchangeTradeHistoryAgent) Cancel() {
}

// InitialSyncComplete sets initial sync to complete
func (a *ExchangeTradeHistoryAgent) InitialSyncComplete() {
	a.Wg.Done()
}

// Stream couples protocol updates
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
