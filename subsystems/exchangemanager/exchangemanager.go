package exchangemanager

import (
	"errors"
	"strings"
	"sync"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")
)

type ExchangeManager struct {
	m         sync.Mutex
	exchanges map[string]exchange.IBotExchange
}

func (e *ExchangeManager) Add(exch exchange.IBotExchange) {
	e.m.Lock()
	if e.exchanges == nil {
		e.exchanges = make(map[string]exchange.IBotExchange)
	}
	e.exchanges[strings.ToLower(exch.GetName())] = exch
	e.m.Unlock()
}

func (e *ExchangeManager) GetExchanges() []exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}

	e.m.Lock()
	defer e.m.Unlock()
	var exchs []exchange.IBotExchange
	for x := range e.exchanges {
		exchs = append(exchs, e.exchanges[x])
	}
	return exchs
}

func (e *ExchangeManager) RemoveExchange(exchName string) error {
	if e.Len() == 0 {
		return ErrNoExchangesLoaded
	}
	exch := e.GetExchangeByName(exchName)
	if exch == nil {
		return ErrExchangeNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()
	delete(e.exchanges, strings.ToLower(exchName))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", exchName)
	return nil
}

func (e *ExchangeManager) GetExchangeByName(exchangeName string) exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}
	e.m.Lock()
	defer e.m.Unlock()
	exch, ok := e.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return nil
	}
	return exch
}

func (e *ExchangeManager) Len() int {
	e.m.Lock()
	defer e.m.Unlock()
	return len(e.exchanges)
}
