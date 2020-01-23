package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// vars for the fund manager package
var (
	OrderManagerDelay      = time.Second * 10
	ErrOrdersAlreadyExists = errors.New("order already exists")
)

func (o *orderStore) Get() map[string][]order.Detail {
	o.m.Lock()
	defer o.m.Unlock()
	return o.Orders
}

func (o *orderStore) exists(order *order.Detail) bool {
	r, ok := o.Orders[order.Exchange]
	if !ok {
		return false
	}

	for x := range r {
		if r[x].ID == order.ID {
			return true
		}
	}

	return false
}

func (o *orderStore) Add(order *order.Detail) error {
	o.m.Lock()
	defer o.m.Unlock()

	if o.exists(order) {
		return ErrOrdersAlreadyExists
	}

	orders := o.Orders[order.Exchange]
	orders = append(orders, *order)
	o.Orders[order.Exchange] = orders
	return nil
}

func (o *orderManager) Started() bool {
	return atomic.LoadInt32(&o.started) == 1
}

func (o *orderManager) Start() error {
	if atomic.AddInt32(&o.started, 1) != 1 {
		return errors.New("order manager already started")
	}

	log.Debugln(log.OrderBook, "Order manager starting...")

	o.shutdown = make(chan struct{})
	o.orderStore.Orders = make(map[string][]order.Detail)
	go o.run()
	return nil
}

func (o *orderManager) Stop() error {
	if atomic.LoadInt32(&o.started) == 0 {
		return errors.New("order manager not started")
	}

	if atomic.AddInt32(&o.stopped, 1) != 1 {
		return errors.New("order manager is already stopped")
	}
	defer func() {
		atomic.CompareAndSwapInt32(&o.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&o.started, 1, 0)
	}()

	log.Debugln(log.OrderBook, "Order manager shutting down...")
	close(o.shutdown)
	return nil
}

func (o *orderManager) gracefulShutdown() {
	if o.cfg.CancelOrdersOnShutdown {
		log.Debugln(log.OrderMgr, "Order manager: Cancelling any open orders...")
		orders := o.orderStore.Get()
		if orders == nil {
			return
		}

		for k, v := range orders {
			log.Debugf(log.OrderMgr, "Order manager: Cancelling order(s) for exchange %s.\n", k)
			for y := range v {
				log.Debugf(log.OrderMgr, "order manager: Cancelling order ID %v [%v]",
					v[y].ID, v[y])
				err := o.Cancel(k, &order.Cancel{
					OrderID: v[y].ID,
				})
				if err != nil {
					msg := fmt.Sprintf("Order manager: Exchange %s unable to cancel order ID=%v. Err: %s",
						k, v[y].ID, err)
					log.Debugln(log.OrderBook, msg)
					Bot.CommsManager.PushEvent(base.Event{
						Type:    "order",
						Message: msg,
					})
					continue
				}

				msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
					k, v[y].ID)
				log.Debugln(log.OrderBook, msg)
				Bot.CommsManager.PushEvent(base.Event{
					Type:    "order",
					Message: msg,
				})
			}
		}
	}
}

func (o *orderManager) run() {
	log.Debugln(log.OrderBook, "Order manager started.")
	tick := time.NewTicker(OrderManagerDelay)
	Bot.ServicesWG.Add(1)
	defer func() {
		log.Debugln(log.OrderMgr, "Order manager shutdown.")
		tick.Stop()
		Bot.ServicesWG.Done()
	}()

	for {
		select {
		case <-o.shutdown:
			o.gracefulShutdown()
			return
		case <-tick.C:
			o.processOrders()
		}
	}
}

func (o *orderManager) CancelAllOrders() {}

func (o *orderManager) Cancel(exchName string, cancel *order.Cancel) error {
	if exchName == "" {
		return errors.New("order exchange name is empty")
	}

	if cancel == nil {
		return errors.New("order cancel param is nil")
	}

	if cancel.OrderID == "" {
		return errors.New("order id is empty")
	}

	exch := GetExchangeByName(exchName)
	if exch == nil {
		return errors.New("unable to get exchange by name")
	}

	if cancel.AssetType.String() != "" && !exch.GetAssetTypes().Contains(cancel.AssetType) {
		return errors.New("order asset type not supported by exchange")
	}

	return exch.CancelOrder(cancel)
}

func (o *orderManager) Submit(exchName string, newOrder *order.Submit) (*orderSubmitResponse, error) {
	if exchName == "" {
		return nil, errors.New("order exchange name must be specified")
	}

	if err := newOrder.Validate(); err != nil {
		return nil, err
	}

	if o.cfg.EnforceLimitConfig {
		if !o.cfg.AllowMarketOrders && newOrder.OrderType == order.Market {
			return nil, errors.New("order market type is not allowed")
		}

		if o.cfg.LimitAmount > 0 && newOrder.Amount > o.cfg.LimitAmount {
			return nil, errors.New("order limit exceeds allowed limit")
		}

		if len(o.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(o.cfg.AllowedExchanges, exchName) {
			return nil, errors.New("order exchange not found in allowed list")
		}

		if len(o.cfg.AllowedPairs) > 0 && !o.cfg.AllowedPairs.Contains(newOrder.Pair, true) {
			return nil, errors.New("order pair not found in allowed list")
		}
	}

	exch := GetExchangeByName(exchName)
	if exch == nil {
		return nil, errors.New("unable to get exchange by name")
	}

	id, err := uuid.NewV4()
	if err != nil {
		log.Warnf(log.OrderMgr,
			"Order manager: Unable to generate UUID. Err: %s\n",
			err)
	}

	result, err := exch.SubmitOrder(newOrder)
	if err != nil {
		return nil, err
	}

	if !result.IsOrderPlaced {
		return nil, errors.New("order unable to be placed")
	}

	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v side=%v type=%v.",
		exchName,
		result.OrderID,
		id.String(),
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.OrderSide,
		newOrder.OrderType)

	log.Debugln(log.OrderMgr, msg)
	Bot.CommsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return &orderSubmitResponse{
		SubmitResponse: order.SubmitResponse{
			OrderID: result.OrderID,
		},
		OurOrderID: id.String(),
	}, nil
}

func (o *orderManager) processOrders() {
	authExchanges := GetAuthAPISupportedExchanges()
	for x := range authExchanges {
		log.Debugf(log.OrderMgr, "Order manager: Procesing orders for exchange %v.\n", authExchanges[x])
		exch := GetExchangeByName(authExchanges[x])
		req := order.GetOrdersRequest{
			OrderSide: order.AnySide,
			OrderType: order.AnyType,
		}
		result, err := exch.GetActiveOrders(&req)
		if err != nil {
			log.Warnf(log.OrderMgr, "Order manager: Unable to get active orders: %s\n", err)
			continue
		}

		for x := range result {
			ord := &result[x]
			result := o.orderStore.Add(ord)
			if result != ErrOrdersAlreadyExists {
				msg := fmt.Sprintf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
					ord.Exchange, ord.ID, ord.CurrencyPair, ord.Price, ord.Amount, ord.OrderSide, ord.OrderType)
				log.Debugf(log.OrderMgr, "%v\n", msg)
				Bot.CommsManager.PushEvent(base.Event{
					Type:    "order",
					Message: msg,
				})
				continue
			}
		}
	}
}
