package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupWebsocketRoutineManager creates a new websocket routine manager
func setupWebsocketRoutineManager(exchangeManager iExchangeManager, orderManager iOrderManager, syncer iCurrencyPairSyncer, cfg *currency.Config, verbose bool) (*WebsocketRoutineManager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if syncer == nil {
		return nil, errNilCurrencyPairSyncer
	}
	if cfg == nil {
		return nil, errNilCurrencyConfig
	}
	if cfg.CurrencyPairFormat == nil {
		return nil, errNilCurrencyPairFormat
	}
	man := &WebsocketRoutineManager{
		verbose:         verbose,
		exchangeManager: exchangeManager,
		orderManager:    orderManager,
		syncer:          syncer,
		currencyConfig:  cfg,
		dataReceivers:   make(map[string]context.CancelFunc),
	}
	return man, man.registerWebsocketDataHandler(man.websocketDataHandler, false)
}

// Start runs the subsystem
func (m *WebsocketRoutineManager) Start(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	if m.currencyConfig == nil {
		return errNilCurrencyConfig
	}

	if m.currencyConfig.CurrencyPairFormat == nil {
		return errNilCurrencyPairFormat
	}

	connectionCtx, connectionCancel := context.WithCancel(ctx)

	m.mu.Lock()
	if !m.state.CompareAndSwap(stoppedState, startingState) {
		m.mu.Unlock()
		connectionCancel()
		return ErrSubSystemAlreadyStarted
	}
	m.connectionCancel = connectionCancel
	m.mu.Unlock()

	go func() {
		m.websocketRoutine(connectionCtx)
		// It's okay for this to fail, just means shutdown has started
		m.state.CompareAndSwap(startingState, readyState)
	}()
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (m *WebsocketRoutineManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return m.state.Load() == readyState
}

// Stop attempts to shutdown the subsystem
func (m *WebsocketRoutineManager) Stop() error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	m.mu.Lock()
	if m.state.Load() == stoppedState {
		m.mu.Unlock()
		return fmt.Errorf("websocket routine manager %w", ErrSubSystemNotStarted)
	}
	m.state.Store(stoppedState)
	if m.connectionCancel != nil {
		m.connectionCancel()
		m.connectionCancel = nil
	}
	m.mu.Unlock()

	m.receiversMu.Lock()
	receivers := m.dataReceivers
	m.dataReceivers = nil
	m.receiversMu.Unlock()

	for _, cancel := range receivers {
		cancel()
	}
	m.wg.Wait()

	return nil
}

// websocketRoutine Initial routine management system for websocket
func (m *WebsocketRoutineManager) websocketRoutine(ctx context.Context) {
	if m.verbose {
		log.Debugln(log.WebsocketMgr, "Connecting exchange websocket services...")
	}
	exchanges, err := m.exchangeManager.GetExchanges()
	if err != nil {
		log.Errorf(log.WebsocketMgr, "websocket routine manager cannot get exchanges: %v", err)
	}
	var wg sync.WaitGroup
	for _, exch := range exchanges {
		if !exch.SupportsWebsocket() {
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: No",
					exch.GetName())
			}
			continue
		}

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "Exchange %s websocket support: Yes Enabled: %v",
				exch.GetName(),
				common.IsEnabled(exch.IsWebsocketEnabled()))
		}

		ws, err := exch.GetWebsocket()
		if err != nil {
			log.Errorf(log.WebsocketMgr, "Exchange %s GetWebsocket error: %s",
				exch.GetName(),
				err)
			continue
		}

		if !ws.IsEnabled() {
			continue
		}

		wg.Go(func() {
			if err := m.websocketDataReceiver(ctx, ws); err != nil {
				if errors.Is(err, errRoutineManagerNotStarted) && ctx.Err() != nil {
					return
				}
				log.Errorf(log.WebsocketMgr, "%v", err)
			}

			if err := ws.Connect(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Errorf(log.WebsocketMgr, "%v", err)
			}
		})
	}
	wg.Wait()
}

// registerExchange attaches an exchange loaded after the routine manager started
// to websocket data handling and initiates its connection. Exchanges loaded
// during startup are connected by websocketRoutine instead.
func (m *WebsocketRoutineManager) registerExchange(ctx context.Context, exch exchange.IBotExchange) error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	if exch == nil {
		return errExchangeIsNil
	}

	if m.state.Load() == stoppedState {
		return nil
	}

	if !exch.SupportsWebsocket() {
		return nil
	}

	ws, err := exch.GetWebsocket()
	if err != nil {
		return fmt.Errorf("%s GetWebsocket error: %w", exch.GetName(), err)
	}

	if ws == nil {
		return errNilWebsocket
	}

	if !ws.IsEnabled() {
		return nil
	}

	if err := m.websocketDataReceiver(ctx, ws); err != nil {
		return err
	}

	if err := ws.Connect(ctx); err != nil {
		return fmt.Errorf("%s websocket connect error: %w", exch.GetName(), err)
	}
	return nil
}

// unregisterExchange stops the data receiver associated with an unloaded
// exchange so a replacement instance can register without duplicating receivers.
func (m *WebsocketRoutineManager) unregisterExchange(name string) {
	if m == nil || name == "" {
		return
	}

	key := strings.ToLower(name)
	m.receiversMu.Lock()
	cancel := m.dataReceivers[key]
	delete(m.dataReceivers, key)
	m.receiversMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// websocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func (m *WebsocketRoutineManager) websocketDataReceiver(ctx context.Context, ws *websocket.Manager) error {
	if m == nil {
		return fmt.Errorf("websocket routine manager %w", ErrNilSubsystem)
	}

	if ws == nil {
		return errNilWebsocket
	}

	if m.state.Load() == stoppedState {
		return errRoutineManagerNotStarted
	}

	// The receiver ignores ctx being cancelled: Engine.RequestShutdown cancels the runtime context before
	// Engine.Stop stops the order manager, which still takes updates until then. Stop, unregisterExchange
	// and a replacement registration cancel it instead.
	receiverCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	m.receiversMu.Lock()
	if m.state.Load() == stoppedState {
		m.receiversMu.Unlock()
		cancel()
		return errRoutineManagerNotStarted
	}
	if m.dataReceivers == nil {
		m.dataReceivers = make(map[string]context.CancelFunc)
	}
	key := strings.ToLower(ws.GetName())
	if previous := m.dataReceivers[key]; previous != nil {
		previous()
	}
	m.dataReceivers[key] = cancel

	// The receiver is counted in m.wg before releasing receiversMu so Stop
	// cannot reach wg.Wait ahead of it.
	m.wg.Go(func() {
		for {
			select {
			case <-receiverCtx.Done():
				return
			case payload := <-ws.DataHandler.C:
				if receiverCtx.Err() != nil {
					return
				}
				if payload.Data == nil {
					log.Errorf(log.WebsocketMgr, "exchange %s nil data sent to websocket", ws.GetName())
				}
				m.mu.RLock()
				for x := range m.dataHandlers {
					if err := m.dataHandlers[x](ws.GetName(), payload.Data); err != nil {
						log.Errorln(log.WebsocketMgr, err)
					}
				}
				m.mu.RUnlock()
			}
		}
	})
	m.receiversMu.Unlock()
	return nil
}

// websocketDataHandler is the default central point for exchange websocket
// implementations to send processed data which will then pass that to an
// appropriate handler.
func (m *WebsocketRoutineManager) websocketDataHandler(exchName string, data any) error {
	switch d := data.(type) {
	case string:
		log.Infoln(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("exchange %s websocket error - %s", exchName, data)
	case websocket.FundingData:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s funding updated %+v",
				exchName,
				m.FormatCurrency(d.CurrencyPair),
				d.AssetType,
				d)
		}
	case *ticker.Price:
		if m.syncer.IsRunning() {
			if err := m.syncer.WebsocketUpdate(exchName,
				d.Pair,
				d.AssetType,
				SyncItemTicker,
				nil); err != nil {
				return err
			}
		}
		m.syncer.PrintTickerSummary(d, "websocket", nil)
	case []ticker.Price:
		var errs error
		for x := range d {
			if m.syncer.IsRunning() {
				if err := m.syncer.WebsocketUpdate(exchName,
					d[x].Pair,
					d[x].AssetType,
					SyncItemTicker,
					nil); err != nil {
					errs = common.AppendError(errs, err)
					continue
				}
			}
			m.syncer.PrintTickerSummary(&d[x], "websocket", nil)
		}
		return errs
	case order.Detail, ticker.Price, orderbook.Depth:
		return errUseAPointer
	case kline.Item:
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
				exchName,
				m.FormatCurrency(d.Pair),
				d.Asset,
				d)
		}
	case []kline.Item:
		for x := range d {
			if m.verbose {
				log.Infof(log.WebsocketMgr, "%s websocket %s %s kline updated %+v",
					exchName,
					m.FormatCurrency(d[x].Pair),
					d[x].Asset,
					d[x])
			}
		}
	case *orderbook.Depth:
		base, err := d.Retrieve()
		if err != nil {
			return err
		}
		if m.syncer.IsRunning() {
			err := m.syncer.WebsocketUpdate(exchName,
				base.Pair,
				base.Asset,
				SyncItemOrderbook,
				nil)
			if err != nil {
				return err
			}
		}
		m.syncer.PrintOrderbookSummary(base, "websocket", nil)
	case *order.Detail:
		if !m.orderManager.IsRunning() {
			return nil
		}
		if !m.orderManager.Exists(d) {
			err := m.orderManager.Add(d)
			if err != nil {
				return err
			}
			m.printOrderSummary(d, false)
		} else {
			od, err := m.orderManager.GetByExchangeAndID(d.Exchange, d.OrderID)
			if err != nil {
				return err
			}
			err = od.UpdateOrderFromDetail(d)
			if err != nil {
				return err
			}

			err = m.orderManager.UpdateExistingOrder(od)
			if err != nil {
				return err
			}
			m.printOrderSummary(od, true)
		}
	case []order.Detail:
		if !m.orderManager.IsRunning() {
			return nil
		}
		for x := range d {
			if !m.orderManager.Exists(&d[x]) {
				err := m.orderManager.Add(&d[x])
				if err != nil {
					return err
				}
				m.printOrderSummary(&d[x], false)
			} else {
				od, err := m.orderManager.GetByExchangeAndID(d[x].Exchange, d[x].OrderID)
				if err != nil {
					return err
				}
				err = od.UpdateOrderFromDetail(&d[x])
				if err != nil {
					return err
				}
				err = m.orderManager.UpdateExistingOrder(od)
				if err != nil {
					return err
				}
				m.printOrderSummary(od, true)
			}
		}
	case order.ClassificationError:
		return fmt.Errorf("%w %s", d.Err, d.Error())
	case websocket.UnhandledMessageWarning:
		log.Warnf(log.WebsocketMgr, "%s unhandled message - %s", exchName, d.Message)
	case []accounts.Change, accounts.Change:
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s %+v", exchName, d)
		}
	case accounts.SubAccounts:
		// TODO: Ingest websocket account snapshots once the portfolio manager supports event-driven updates; logging is an intentional stopgap.
		if m.verbose {
			for _, subAccount := range d {
				if subAccount == nil {
					continue
				}
				for c, balance := range subAccount.Balances {
					log.Debugf(log.PortfolioMgr, "Portfolio [Websocket]: Received %s %s balance update: %s, %f", exchName, subAccount.AssetType, c, balance.Total)
				}
			}
		}
	case []trade.Data, trade.Data:
		if m.verbose {
			log.Infof(log.Trade, "%+v", d)
		}
	case []fill.Data:
		if m.verbose {
			log.Infof(log.Fill, "%+v", d)
		}
	case []futures.Position:
		// TODO: Apply canonical websocket snapshots once PositionController supports snapshot ingestion.
		if m.verbose {
			log.Infof(log.WebsocketMgr, "%s websocket futures positions updated %+v", exchName, d)
		}
	default:
		if m.verbose {
			log.Warnf(log.WebsocketMgr, "%s websocket Unknown type: %+v", exchName, d)
		}
	}
	return nil
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *WebsocketRoutineManager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || m.state.Load() == stoppedState {
		return p
	}
	return p.Format(*m.currencyConfig.CurrencyPairFormat)
}

// printOrderSummary this function will be deprecated when a order manager
// update is done.
func (m *WebsocketRoutineManager) printOrderSummary(o *order.Detail, isUpdate bool) {
	if m == nil || m.state.Load() == stoppedState || o == nil {
		return
	}

	orderNotif := "New Order:"
	if isUpdate {
		orderNotif = "Order Change:"
	}

	log.Debugf(log.WebsocketMgr,
		"%s %s %s %s %s %s %s OrderID:%s ClientOrderID:%s Price:%f Average Executed Price:%f Amount:%f Executed Amount:%f Remaining Amount:%f TimeInForce:%s ReduceOnly:%t",
		orderNotif,
		o.Exchange,
		o.AssetType,
		o.Pair,
		o.Status,
		o.Type,
		o.Side,
		o.OrderID,
		o.ClientOrderID,
		o.Price,
		o.AverageExecutedPrice,
		o.Amount,
		o.ExecutedAmount,
		o.RemainingAmount,
		o.TimeInForce,
		o.ReduceOnly)
}

// registerWebsocketDataHandler registers an externally (GCT Library) defined
// dedicated filter specific data types for internal & external strategy use.
// InterceptorOnly as true will purge all other registered handlers
// (including default) bypassing all other handling.
func (m *WebsocketRoutineManager) registerWebsocketDataHandler(fn WebsocketDataHandler, interceptorOnly bool) error {
	if m == nil {
		return fmt.Errorf("%T %w", m, ErrNilSubsystem)
	}

	if fn == nil {
		return errNilWebsocketDataHandlerFunction
	}

	if interceptorOnly {
		return m.setWebsocketDataHandler(fn)
	}

	m.mu.Lock()
	// Push front so that any registered data handler has first preference
	// over the GCT default handler.
	m.dataHandlers = append([]WebsocketDataHandler{fn}, m.dataHandlers...)
	m.mu.Unlock()
	return nil
}

// setWebsocketDataHandler sets a single websocket data handler, removing all
// pre-existing handlers.
func (m *WebsocketRoutineManager) setWebsocketDataHandler(fn WebsocketDataHandler) error {
	if m == nil {
		return fmt.Errorf("%T %w", m, ErrNilSubsystem)
	}
	if fn == nil {
		return errNilWebsocketDataHandlerFunction
	}
	m.mu.Lock()
	m.dataHandlers = []WebsocketDataHandler{fn}
	m.mu.Unlock()
	return nil
}
