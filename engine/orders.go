package engine

import (
	"sync"
	"time"

	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// OrderManager manages orders for all enabled exchanges
type OrderManager struct {
	m      sync.Mutex
	Orders map[string][]exchange.OrderDetail
}

func (o *OrderManager) add() {
	o.m.Lock()
	defer o.m.Unlock()
}

// StartOrderManagerRoutine starts the orderbook manage routine
func StartOrderManagerRoutine() {
	log.Debugln("Starting order manager routine")
	if Bot.OrderManager == nil {
		Bot.OrderManager = new(OrderManager)
	}

	for {
		for x := range Bot.Exchanges {
			if !Bot.Exchanges[x].IsEnabled() || !Bot.Exchanges[x].GetAuthenticatedAPISupport() {
				continue
			}
			exchName := Bot.Exchanges[x].GetName()
			log.Printf("Getting active orders for %s", exchName)

			orders, err := Bot.Exchanges[x].GetActiveOrders(&exchange.GetOrdersRequest{})
			if err != nil {
				log.Printf("Get active orders failed: %s", err)
				continue
			}

			log.Printf("Orders for exchange %s: %v", exchName, orders)
		}
		time.Sleep(time.Second * 1)
	}
}
