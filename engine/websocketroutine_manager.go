package engine

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupWebsocketRoutineManager creates a new websocket routine manager
func setupWebsocketRoutineManager(exchangeManager iExchangeManager, orderManager iOrderManager, syncer iCurrencyPairSyncer, cfg *config.CurrencyConfig, verbose bool) (*websocketRoutineManager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if orderManager == nil {
		return nil, errNilOrderManager
	}
	if syncer == nil {
		return nil, errNilCurrencyPairSyncer
	}
	if cfg == nil {
		return nil, errNilCurrencyConfig
	}
	if cfg.CurrencyPairFormat == nil && verbose {
		return nil, errNilCurrencyPairFormat
	}
	return &websocketRoutineManager{
		verbose:         verbose,
		exchangeManager: exchangeManager,
		orderManager:    orderManager,
		syncer:          syncer,
		currencyConfig:  cfg,
		shutdown:        make(chan struct{}),
	}, nil
}

// Start runs the subsystem
func (m *websocketRoutineManager) Start() error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return ErrSubSystemAlreadyStarted
	}
	m.shutdown = make(chan struct{})
	go m.websocketRoutine()
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (m *websocketRoutineManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Stop attempts to shutdown the subsystem
func (m *websocketRoutineManager) Stop() error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("websocket routine manager %w", ErrSubSystemNotStarted)
	}
	close(m.shutdown)
	m.wg.Wait()
	return nil
}

// websocketRoutine Initial routine management system for websocket
func (m *websocketRoutineManager) websocketRoutine() {
	if m.verbose {
		log.Debugln(log.WebsocketMgr, "Connecting exchange websocket services...")
	}
	exchanges := m.exchangeManager.GetExchanges()
	for i := range exchanges {
		go func(i int) {
			if exchanges[i].SupportsWebsocket() {
				if m.verbose {
					log.Debugf(log.WebsocketMgr,
						"Exchange %s websocket support: Yes Enabled: %v\n",
						exchanges[i].GetName(),
						common.IsEnabled(exchanges[i].IsWebsocketEnabled()),
					)
				}

				ws, err := exchanges[i].GetWebsocket()
				if err != nil {
					log.Errorf(
						log.WebsocketMgr,
						"Exchange %s GetWebsocket error: %s\n",
						exchanges[i].GetName(),
						err,
					)
					return
				}

				// Exchange sync manager might have already started ws
				// service or is in the process of connecting, so check
				if ws.IsConnected() || ws.IsConnecting() {
					return
				}

				// Data handler routine
				go m.WebsocketDataReceiver(ws)

				if ws.IsEnabled() {
					err = ws.Connect()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "%v\n", err)
					}
					err = ws.FlushChannels()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "Failed to subscribe: %v\n", err)
					}
				}
			} else if m.verbose {
				log.Debugf(log.WebsocketMgr,
					"Exchange %s websocket support: No\n",
					exchanges[i].GetName(),
				)
			}
		}(i)
	}
}

// WebsocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func (m *websocketRoutineManager) WebsocketDataReceiver(ws *stream.Websocket) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.shutdown:
			return
		case data := <-ws.ToRoutine:
			err := m.WebsocketDataHandler(ws.GetName(), data)
			if err != nil {
				log.Error(log.WebsocketMgr, err)
			}
		}
	}
}

// WebsocketDataHandler is a central point for exchange websocket implementations to send
// processed data. WebsocketDataHandler will then pass that to an appropriate handler
func (m *websocketRoutineManager) WebsocketDataHandler(exchName string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("exchange %s nil data sent to websocket",
			exchName)
	}

	switch d := data.(type) {
	case string:
		log.Info(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("exchange %s websocket error - %s", exchName, data)
	case stream.FundingData:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v",
				exchName,
				m.FormatCurrency(d.CurrencyPair),
				d.AssetType,
				d)
		}
	case *ticker.Price:
		if m.syncer.IsRunning() {
			err := m.syncer.Update(exchName,
				d.Pair,
				d.AssetType,
				SyncItemTicker,
				nil)
			if err != nil {
				return err
			}
		}
		err := ticker.ProcessTicker(d)
		if err != nil {
			return err
		}
		m.syncer.PrintTickerSummary(d, "websocket", err)
	case stream.KlineData:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
				exchName,
				m.FormatCurrency(d.Pair),
				d.AssetType,
				d)
		}
	case *orderbook.Base:
		if m.syncer.IsRunning() {
			err := m.syncer.Update(exchName,
				d.Pair,
				d.Asset,
				SyncItemOrderbook,
				nil)
			if err != nil {
				return err
			}
		}
		m.syncer.PrintOrderbookSummary(d, "websocket", nil)
	case *order.Detail:
		m.printOrderSummary(d)
		if !m.orderManager.Exists(d) {
			err := m.orderManager.Add(d)
			if err != nil {
				return err
			}
		} else {
			od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.ID)
			if err != nil {
				return err
			}
			od.UpdateOrderFromDetail(d)

			err = m.orderManager.UpdateExistingOrder(od)
			if err != nil {
				return err
			}
		}
	case *order.Modify:
		m.printOrderChangeSummary(d)
		od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.ID)
		if err != nil {
			return err
		}
		od.UpdateOrderFromModify(d)
		err = m.orderManager.UpdateExistingOrder(od)
		if err != nil {
			return err
		}
	case order.ClassificationError:
		return fmt.Errorf("%w %s", d.Err, d.Error())
	case stream.UnhandledMessageWarning:
		log.Warn(log.WebsocketMgr, d.Message)
	case account.Change:
		if m.verbose {
			m.printAccountHoldingsChangeSummary(d)
		}
	default:
		if m.verbose {
			log.Warnf(log.WebsocketMgr,
				"%s websocket Unknown type: %+v",
				exchName,
				d)
		}
	}
	return nil
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *websocketRoutineManager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.currencyConfig.CurrencyPairFormat.Delimiter,
		m.currencyConfig.CurrencyPairFormat.Uppercase)
}

// printOrderChangeSummary this function will be deprecated when a order manager
// update is done.
func (m *websocketRoutineManager) printOrderChangeSummary(o *order.Modify) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 || o == nil {
		return
	}

	log.Debugf(log.WebsocketMgr,
		"Order Change: %s %s %s %s %s %s OrderID:%s ClientOrderID:%s Price:%f Amount:%f Executed Amount:%f Remaining Amount:%f",
		o.Exchange,
		o.AssetType,
		o.Pair,
		o.Status,
		o.Type,
		o.Side,
		o.ID,
		o.ClientOrderID,
		o.Price,
		o.Amount,
		o.ExecutedAmount,
		o.RemainingAmount)
}

// printOrderSummary this function will be deprecated when a order manager
// update is done.
func (m *websocketRoutineManager) printOrderSummary(o *order.Detail) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 || o == nil {
		return
	}
	log.Debugf(log.WebsocketMgr,
		"New Order: %s %s %s %s %s %s OrderID:%s ClientOrderID:%s Price:%f Amount:%f Executed Amount:%f Remaining Amount:%f",
		o.Exchange,
		o.AssetType,
		o.Pair,
		o.Status,
		o.Type,
		o.Side,
		o.ID,
		o.ClientOrderID,
		o.Price,
		o.Amount,
		o.ExecutedAmount,
		o.RemainingAmount)
}

// printAccountHoldingsChangeSummary this function will be deprecated when a
// account holdings update is done.
func (m *websocketRoutineManager) printAccountHoldingsChangeSummary(o account.Change) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	log.Debugf(log.WebsocketMgr,
		"Account Holdings Balance Changed: %s %s %s has changed balance by %f for account: %s",
		o.Exchange,
		o.Asset,
		o.Currency,
		o.Amount,
		o.Account)
}
