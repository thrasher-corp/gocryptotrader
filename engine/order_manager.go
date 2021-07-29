package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupOrderManager will boot up the OrderManager
func SetupOrderManager(exchangeManager iExchangeManager, communicationsManager iCommsManager, wg *sync.WaitGroup, verbose bool) (*OrderManager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if communicationsManager == nil {
		return nil, errNilCommunicationsManager
	}
	if wg == nil {
		return nil, errNilWaitGroup
	}

	return &OrderManager{
		shutdown: make(chan struct{}),
		orderStore: store{
			Orders:          make(map[string][]*order.Detail),
			exchangeManager: exchangeManager,
			commsManager:    communicationsManager,
			wg:              wg,
		},
		verbose: verbose,
	}, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *OrderManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *OrderManager) Start() error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("order manager %w", ErrSubSystemAlreadyStarted)
	}
	log.Debugln(log.OrderMgr, "Order manager starting...")
	m.shutdown = make(chan struct{})
	go m.run()
	return nil
}

// Stop attempts to shutdown the subsystem
func (m *OrderManager) Stop() error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugln(log.OrderMgr, "Order manager shutting down...")
	close(m.shutdown)
	return nil
}

// gracefulShutdown cancels all orders (if enabled) before shutting down
func (m *OrderManager) gracefulShutdown() {
	if m.cfg.CancelOrdersOnShutdown {
		log.Debugln(log.OrderMgr, "Order manager: Cancelling any open orders...")
		m.CancelAllOrders(m.orderStore.exchangeManager.GetExchanges())
	}
}

// run will periodically process orders
func (m *OrderManager) run() {
	log.Debugln(log.OrderMgr, "Order manager started.")
	tick := time.NewTicker(orderManagerDelay)
	m.orderStore.wg.Add(1)
	defer func() {
		log.Debugln(log.OrderMgr, "Order manager shutdown.")
		tick.Stop()
		m.orderStore.wg.Done()
	}()

	for {
		select {
		case <-m.shutdown:
			m.gracefulShutdown()
			return
		case <-tick.C:
			go m.processOrders()
		}
	}
}

// CancelAllOrders iterates and cancels all orders for each exchange provided
func (m *OrderManager) CancelAllOrders(exchangeNames []exchange.IBotExchange) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}

	orders := m.orderStore.get()
	if orders == nil {
		return
	}

	for i := range exchangeNames {
		exchangeOrders, ok := orders[strings.ToLower(exchangeNames[i].GetName())]
		if !ok {
			continue
		}
		for j := range exchangeOrders {
			log.Debugf(log.OrderMgr, "Order manager: Cancelling order(s) for exchange %s.", exchangeNames[i].GetName())
			err := m.Cancel(&order.Cancel{
				Exchange:      exchangeOrders[j].Exchange,
				ID:            exchangeOrders[j].ID,
				AccountID:     exchangeOrders[j].AccountID,
				ClientID:      exchangeOrders[j].ClientID,
				WalletAddress: exchangeOrders[j].WalletAddress,
				Type:          exchangeOrders[j].Type,
				Side:          exchangeOrders[j].Side,
				Pair:          exchangeOrders[j].Pair,
				AssetType:     exchangeOrders[j].AssetType,
			})
			if err != nil {
				log.Error(log.OrderMgr, err)
			}
		}
	}
}

// Cancel will find the order in the OrderManager, send a cancel request
// to the exchange and if successful, update the status of the order
func (m *OrderManager) Cancel(cancel *order.Cancel) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	var err error
	defer func() {
		if err != nil {
			m.orderStore.commsManager.PushEvent(base.Event{
				Type:    "order",
				Message: err.Error(),
			})
		}
	}()

	if cancel == nil {
		err = errors.New("order cancel param is nil")
		return err
	}
	if cancel.Exchange == "" {
		err = errors.New("order exchange name is empty")
		return err
	}
	if cancel.ID == "" {
		err = errors.New("order id is empty")
		return err
	}

	exch := m.orderStore.exchangeManager.GetExchangeByName(cancel.Exchange)
	if exch == nil {
		err = ErrExchangeNotFound
		return err
	}

	if cancel.AssetType.String() != "" && !exch.GetAssetTypes(false).Contains(cancel.AssetType) {
		err = errors.New("order asset type not supported by exchange")
		return err
	}

	log.Debugf(log.OrderMgr, "Order manager: Cancelling order ID %v [%+v]",
		cancel.ID, cancel)

	err = exch.CancelOrder(cancel)
	if err != nil {
		err = fmt.Errorf("%v - Failed to cancel order: %w", cancel.Exchange, err)
		return err
	}
	var od *order.Detail
	od, err = m.orderStore.getByExchangeAndID(cancel.Exchange, cancel.ID)
	if err != nil {
		err = fmt.Errorf("%v - Failed to retrieve order %v to update cancelled status: %w", cancel.Exchange, cancel.ID, err)
		return err
	}

	od.Status = order.Cancelled
	msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
		od.Exchange, od.ID)
	log.Debugln(log.OrderMgr, msg)
	m.orderStore.commsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return nil
}

// GetOrderInfo calls the exchange's wrapper GetOrderInfo function
// and stores the result in the order manager
func (m *OrderManager) GetOrderInfo(exchangeName, orderID string, cp currency.Pair, a asset.Item) (order.Detail, error) {
	if m == nil {
		return order.Detail{}, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return order.Detail{}, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	if orderID == "" {
		return order.Detail{}, ErrOrderIDCannotBeEmpty
	}

	exch := m.orderStore.exchangeManager.GetExchangeByName(exchangeName)
	if exch == nil {
		return order.Detail{}, ErrExchangeNotFound
	}
	result, err := exch.GetOrderInfo(orderID, cp, a)
	if err != nil {
		return order.Detail{}, err
	}

	err = m.orderStore.add(&result)
	if err != nil && err != ErrOrdersAlreadyExists {
		return order.Detail{}, err
	}

	return result, nil
}

// validate ensures a submitted order is valid before adding to the manager
func (m *OrderManager) validate(newOrder *order.Submit) error {
	if newOrder == nil {
		return errors.New("order cannot be nil")
	}

	if newOrder.Exchange == "" {
		return errors.New("order exchange name must be specified")
	}

	if err := newOrder.Validate(); err != nil {
		return fmt.Errorf("order manager: %w", err)
	}

	if m.cfg.EnforceLimitConfig {
		if !m.cfg.AllowMarketOrders && newOrder.Type == order.Market {
			return errors.New("order market type is not allowed")
		}

		if m.cfg.LimitAmount > 0 && newOrder.Amount > m.cfg.LimitAmount {
			return errors.New("order limit exceeds allowed limit")
		}

		if len(m.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(m.cfg.AllowedExchanges, newOrder.Exchange) {
			return errors.New("order exchange not found in allowed list")
		}

		if len(m.cfg.AllowedPairs) > 0 && !m.cfg.AllowedPairs.Contains(newOrder.Pair, true) {
			return errors.New("order pair not found in allowed list")
		}
	}
	return nil
}

// Submit will take in an order struct, send it to the exchange and
// populate it in the OrderManager if successful
func (m *OrderManager) Submit(newOrder *order.Submit) (*OrderSubmitResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := m.orderStore.exchangeManager.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	// Checks for exchange min max limits for order amounts before order
	// execution can occur
	err = exch.CheckOrderExecutionLimits(newOrder.AssetType,
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.Type)
	if err != nil {
		return nil, fmt.Errorf("order manager: exchange %s unable to place order: %w",
			newOrder.Exchange,
			err)
	}

	result, err := exch.SubmitOrder(newOrder)
	if err != nil {
		return nil, err
	}

	return m.processSubmittedOrder(newOrder, result)
}

// SubmitFakeOrder runs through the same process as order submission
// but does not touch live endpoints
func (m *OrderManager) SubmitFakeOrder(newOrder *order.Submit, resultingOrder order.SubmitResponse, checkExchangeLimits bool) (*OrderSubmitResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := m.orderStore.exchangeManager.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	if checkExchangeLimits {
		// Checks for exchange min max limits for order amounts before order
		// execution can occur
		err = exch.CheckOrderExecutionLimits(newOrder.AssetType,
			newOrder.Pair,
			newOrder.Price,
			newOrder.Amount,
			newOrder.Type)
		if err != nil {
			return nil, fmt.Errorf("order manager: exchange %s unable to place order: %w",
				newOrder.Exchange,
				err)
		}
	}
	return m.processSubmittedOrder(newOrder, resultingOrder)
}

// GetOrdersSnapshot returns a snapshot of all orders in the orderstore. It optionally filters any orders that do not match the status
// but a status of "" or ANY will include all
// the time adds contexts for the when the snapshot is relevant for
func (m *OrderManager) GetOrdersSnapshot(s order.Status) ([]order.Detail, time.Time) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return nil, time.Time{}
	}
	var os []order.Detail
	var latestUpdate time.Time
	for _, v := range m.orderStore.Orders {
		for i := range v {
			if s != v[i].Status &&
				s != order.AnyStatus &&
				s != "" {
				continue
			}
			if v[i].LastUpdated.After(latestUpdate) {
				latestUpdate = v[i].LastUpdated
			}
			os = append(os, *v[i])
		}
	}

	return os, latestUpdate
}

// GetOrdersFiltered returns a snapshot of all orders in the order store.
// Filtering is applied based on the order.Filter unless entries are empty
func (m *OrderManager) GetOrdersFiltered(f *order.Filter) ([]order.Detail, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if f == nil {
		return nil, fmt.Errorf("order manager, GetOrdersFiltered: Filter is nil")
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.getFilteredOrders(f)
}

// processSubmittedOrder adds a new order to the manager
func (m *OrderManager) processSubmittedOrder(newOrder *order.Submit, result order.SubmitResponse) (*OrderSubmitResponse, error) {
	if !result.IsOrderPlaced {
		return nil, errors.New("order unable to be placed")
	}

	id, err := uuid.NewV4()
	if err != nil {
		log.Warnf(log.OrderMgr,
			"Order manager: Unable to generate UUID. Err: %s",
			err)
	}
	if newOrder.Date.IsZero() {
		newOrder.Date = time.Now()
	}
	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v side=%v type=%v for time %v.",
		newOrder.Exchange,
		result.OrderID,
		id.String(),
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.Side,
		newOrder.Type,
		newOrder.Date)

	log.Debugln(log.OrderMgr, msg)
	m.orderStore.commsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})
	status := order.New
	if result.FullyMatched {
		status = order.Filled
	}
	err = m.orderStore.add(&order.Detail{
		ImmediateOrCancel: newOrder.ImmediateOrCancel,
		HiddenOrder:       newOrder.HiddenOrder,
		FillOrKill:        newOrder.FillOrKill,
		PostOnly:          newOrder.PostOnly,
		Price:             newOrder.Price,
		Amount:            newOrder.Amount,
		LimitPriceUpper:   newOrder.LimitPriceUpper,
		LimitPriceLower:   newOrder.LimitPriceLower,
		TriggerPrice:      newOrder.TriggerPrice,
		TargetAmount:      newOrder.TargetAmount,
		ExecutedAmount:    newOrder.ExecutedAmount,
		RemainingAmount:   newOrder.RemainingAmount,
		Fee:               newOrder.Fee,
		Exchange:          newOrder.Exchange,
		InternalOrderID:   id.String(),
		ID:                result.OrderID,
		AccountID:         newOrder.AccountID,
		ClientID:          newOrder.ClientID,
		ClientOrderID:     newOrder.ClientOrderID,
		WalletAddress:     newOrder.WalletAddress,
		Type:              newOrder.Type,
		Side:              newOrder.Side,
		Status:            status,
		AssetType:         newOrder.AssetType,
		Date:              time.Now(),
		LastUpdated:       time.Now(),
		Pair:              newOrder.Pair,
		Leverage:          newOrder.Leverage,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to add %v order %v to orderStore: %s", newOrder.Exchange, result.OrderID, err)
	}

	return &OrderSubmitResponse{
		SubmitResponse: order.SubmitResponse{
			IsOrderPlaced: result.IsOrderPlaced,
			OrderID:       result.OrderID,
		},
		InternalOrderID: id.String(),
	}, nil
}

// processOrders iterates over all exchange orders via API
// and adds them to the internal order store
func (m *OrderManager) processOrders() {
	exchanges := m.orderStore.exchangeManager.GetExchanges()
	for i := range exchanges {
		if !exchanges[i].GetAuthenticatedAPISupport(exchange.RestAuthentication) {
			continue
		}
		log.Debugf(log.OrderMgr,
			"Order manager: Processing orders for exchange %v.",
			exchanges[i].GetName())

		supportedAssets := exchanges[i].GetAssetTypes(true)
		for y := range supportedAssets {
			pairs, err := exchanges[i].GetEnabledPairs(supportedAssets[y])
			if err != nil {
				log.Errorf(log.OrderMgr,
					"Order manager: Unable to get enabled pairs for %s and asset type %s: %s",
					exchanges[i].GetName(),
					supportedAssets[y],
					err)
				continue
			}

			if len(pairs) == 0 {
				if m.verbose {
					log.Debugf(log.OrderMgr,
						"Order manager: No pairs enabled for %s and asset type %s, skipping...",
						exchanges[i].GetName(),
						supportedAssets[y])
				}
				continue
			}

			req := order.GetOrdersRequest{
				Side:      order.AnySide,
				Type:      order.AnyType,
				Pairs:     pairs,
				AssetType: supportedAssets[y],
			}
			result, err := exchanges[i].GetActiveOrders(&req)
			if err != nil {
				log.Warnf(log.OrderMgr,
					"Order manager: Unable to get active orders for %s and asset type %s: %s",
					exchanges[i].GetName(),
					supportedAssets[y],
					err)
				continue
			}

			for z := range result {
				ord := &result[z]
				result := m.orderStore.add(ord)
				if result != ErrOrdersAlreadyExists {
					msg := fmt.Sprintf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
						ord.Exchange, ord.ID, ord.Pair, ord.Price, ord.Amount, ord.Side, ord.Type)
					log.Debugf(log.OrderMgr, "%v", msg)
					m.orderStore.commsManager.PushEvent(base.Event{
						Type:    "order",
						Message: msg,
					})
					continue
				}
			}
		}
	}
}

// Exists checks whether an order exists in the order store
func (m *OrderManager) Exists(o *order.Detail) bool {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return false
	}

	return m.orderStore.exists(o)
}

// Add adds an order to the orderstore
func (m *OrderManager) Add(o *order.Detail) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	return m.orderStore.add(o)
}

// GetByExchangeAndID returns a copy of an order from an exchange if it matches the ID
func (m *OrderManager) GetByExchangeAndID(exchangeName, id string) (*order.Detail, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	o, err := m.orderStore.getByExchangeAndID(exchangeName, id)
	if err != nil {
		return nil, err
	}
	var cpy order.Detail
	cpy.UpdateOrderFromDetail(o)
	return &cpy, nil
}

// UpdateExistingOrder will update an existing order in the orderstore
func (m *OrderManager) UpdateExistingOrder(od *order.Detail) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.updateExisting(od)
}

// UpsertOrder updates an existing order or adds a new one to the orderstore
func (m *OrderManager) UpsertOrder(od *order.Detail) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.upsert(od)
}

// get returns all orders for all exchanges
// should not be exported as it can have large impact if used improperly
func (s *store) get() map[string][]*order.Detail {
	s.m.Lock()
	orders := s.Orders
	s.m.Unlock()
	return orders
}

// getByExchangeAndID returns a specific order by exchange and id
func (s *store) getByExchangeAndID(exchange, id string) (*order.Detail, error) {
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, ErrExchangeNotFound
	}

	for x := range r {
		if r[x].ID == id {
			return r[x], nil
		}
	}
	return nil, ErrOrderNotFound
}

// updateExisting checks if an order exists in the orderstore
// and then updates it
func (s *store) updateExisting(od *order.Detail) error {
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[strings.ToLower(od.Exchange)]
	if !ok {
		return ErrExchangeNotFound
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x].UpdateOrderFromDetail(od)
			return nil
		}
	}

	return ErrOrderNotFound
}

func (s *store) upsert(od *order.Detail) error {
	lName := strings.ToLower(od.Exchange)
	exch := s.exchangeManager.GetExchangeByName(lName)
	if exch == nil {
		return ErrExchangeNotFound
	}
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[lName]
	if !ok {
		s.Orders[lName] = []*order.Detail{od}
		return nil
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x].UpdateOrderFromDetail(od)
			return nil
		}
	}
	s.Orders[lName] = append(s.Orders[lName], od)
	return nil
}

// getByExchange returns orders by exchange
func (s *store) getByExchange(exchange string) ([]*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, ErrExchangeNotFound
	}
	return r, nil
}

// getByInternalOrderID will search all orders for our internal orderID
// and return the order
func (s *store) getByInternalOrderID(internalOrderID string) (*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	for _, v := range s.Orders {
		for x := range v {
			if v[x].InternalOrderID == internalOrderID {
				return v[x], nil
			}
		}
	}
	return nil, ErrOrderNotFound
}

// exists verifies if the orderstore contains the provided order
func (s *store) exists(det *order.Detail) bool {
	if det == nil {
		return false
	}
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(det.Exchange)]
	if !ok {
		return false
	}

	for x := range r {
		if r[x].ID == det.ID {
			return true
		}
	}
	return false
}

// Add Adds an order to the orderStore for tracking the lifecycle
func (s *store) add(det *order.Detail) error {
	if det == nil {
		return errors.New("order store: Order is nil")
	}
	exch := s.exchangeManager.GetExchangeByName(det.Exchange)
	if exch == nil {
		return ErrExchangeNotFound
	}
	if s.exists(det) {
		return ErrOrdersAlreadyExists
	}
	// Untracked websocket orders will not have internalIDs yet
	if det.InternalOrderID == "" {
		id, err := uuid.NewV4()
		if err != nil {
			log.Warnf(log.OrderMgr,
				"Order manager: Unable to generate UUID. Err: %s",
				err)
		} else {
			det.InternalOrderID = id.String()
		}
	}
	s.m.Lock()
	defer s.m.Unlock()
	orders := s.Orders[strings.ToLower(det.Exchange)]
	orders = append(orders, det)
	s.Orders[strings.ToLower(det.Exchange)] = orders

	return nil
}

// getFilteredOrders returns a filtered copy of the orders
func (s *store) getFilteredOrders(f *order.Filter) ([]order.Detail, error) {
	if f == nil {
		return nil, errors.New("filter is nil")
	}
	s.m.RLock()
	defer s.m.RUnlock()

	var os []order.Detail
	// optimization if Exchange is filtered
	if f.Exchange != "" {
		if e, ok := s.Orders[strings.ToLower(f.Exchange)]; ok {
			for i := range e {
				if !e[i].MatchFilter(f) {
					continue
				}
				os = append(os, e[i].Copy())
			}
		}
	} else {
		for _, e := range s.Orders {
			for i := range e {
				if !e[i].MatchFilter(f) {
					continue
				}
				os = append(os, e[i].Copy())
			}
		}
	}
	return os, nil
}
