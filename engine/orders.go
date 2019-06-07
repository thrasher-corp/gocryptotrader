package engine

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// vars for the fund manager package
var (
	OrderManagerDelay      = time.Second * 10
	ErrOrdersAlreadyExists = errors.New("order already exists")
)

type orderStore struct {
	m      sync.Mutex
	Orders map[string][]exchange.OrderDetail
}

func (o *orderStore) exists(order *exchange.OrderDetail) bool {
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

func (o *orderStore) Add(order *exchange.OrderDetail) error {
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

type orderManager struct {
	started    int32
	stopped    int32
	shutdown   chan struct{}
	orderStore orderStore
}

func (o *orderManager) Started() bool {
	return atomic.LoadInt32(&o.started) == 1
}

func (o *orderManager) Start() error {
	if atomic.AddInt32(&o.started, 1) != 1 {
		return errors.New("order manager already started")
	}

	log.Debugln("Order manager starting...")
	o.shutdown = make(chan struct{})
	o.orderStore.Orders = make(map[string][]exchange.OrderDetail)
	go o.run()
	return nil
}
func (o *orderManager) Stop() error {
	if atomic.AddInt32(&o.stopped, 1) != 1 {
		return errors.New("order manager is already stopped")
	}

	log.Debugln("Order manager shutting down...")
	close(o.shutdown)
	return nil
}

func (o *orderManager) run() {
	log.Debugln("Order manager started.")
	tick := time.NewTicker(OrderManagerDelay)
	defer func() {
		log.Debugf("Order manager shutdown.")
		tick.Stop()
	}()

	for {
		select {
		case <-o.shutdown:
			return
		case <-tick.C:
			o.processOrders()
		}
	}
}

func (o *orderManager) Cancel() {}

func (o *orderManager) Place() {}

func (o *orderManager) processOrders() {
	authExchanges := GetAuthAPISupportedExchanges()
	for x := range authExchanges {
		log.Debugf("Order manager: Procesing orders for exchange %v.", authExchanges[x])
		exch := GetExchangeByName(authExchanges[x])
		req := exchange.GetOrdersRequest{
			OrderSide: exchange.AnyOrderSide,
			OrderType: exchange.AnyOrderType,
		}
		result, err := exch.GetActiveOrders(&req)
		if err != nil {
			log.Debugf("Order manager: Unable to get active orders: %s", err)
			continue
		}

		for x := range result {
			order := &result[x]
			result := o.orderStore.Add(order)
			if result != ErrOrdersAlreadyExists {
				log.Debugf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
					order.Exchange, order.ID, order.CurrencyPair, order.Price, order.Amount, order.OrderSide, order.OrderType)
			}
		}
	}
}
