package orderbook

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const wsOrderbookBufferLimit = 5

// Update updates a local cache using bid targets and ask targets then updates
// main cache in orderbook.go
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *WebsocketOrderbookLocal) Update(bidTargets, askTargets []orderbook.Item,
	p currency.Pair,
	updated time.Time,
	exchName, assetType string) error {
	if bidTargets == nil && askTargets == nil {
		return errors.New("exchange.go websocket orderbook cache Update() error - cannot have bids and ask targets both nil")
	}
	if _, ok := w.orderbook[p][assetType]; !ok {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			p.String(),
			assetType)
	}

	if w.orderbookBuffer == nil {
		w.orderbookBuffer = make(map[currency.Pair]map[string][]orderbook.Base)
	}
	if w.orderbookBuffer[p] == nil {
		w.orderbookBuffer[p] = make(map[string][]orderbook.Base)
	}

	if len(w.orderbookBuffer[p][assetType]) < wsOrderbookBufferLimit {
		ob, err := w.NewBase(bidTargets, askTargets, p, updated, exchName, assetType)
		if err != nil {
			return err
		}
		w.orderbookBuffer[p][assetType] = append(w.orderbookBuffer[p][assetType], *ob)
		return nil
		// add entry
	}
	// sort by last updated to ensure each update is in order
	sort.Slice(w.orderbookBuffer[p][assetType], func(i, j int) bool {
		return w.orderbookBuffer[p][assetType][i].LastUpdated.Before(w.orderbookBuffer[p][assetType][j].LastUpdated)
	})
	for i := range w.orderbookBuffer[p][assetType] {
		var wg sync.WaitGroup
		wg.Add(2)
		go w.updateAsksByPrice(&w.orderbookBuffer[p][assetType][i], assetType, p, &wg)
		go w.updateBidsByPrice(&w.orderbookBuffer[p][assetType][i], assetType, p, &wg)
		wg.Wait()
	}
	return w.orderbook[p][assetType].Process()
}

func (w *WebsocketOrderbookLocal) updateAsksByPrice(base *orderbook.Base, assetType string, p currency.Pair, wg *sync.WaitGroup) {
	for j := range base.Asks {
		found := false
		for k := range w.orderbook[p][assetType].Asks {
			if w.orderbook[p][assetType].Asks[k].Price == base.Asks[j].Price {
				found = true
				if base.Asks[j].Amount == 0 {
					w.orderbook[p][assetType].Asks = append(w.orderbook[p][assetType].Asks[:j],
						w.orderbook[p][assetType].Asks[j+1:]...)
					j--
					break
				}
				w.orderbook[p][assetType].Asks[k].Amount = base.Asks[j].Amount
				break
			}
		}
		if !found {
			w.orderbook[p][assetType].Asks = append(w.orderbook[p][assetType].Asks, base.Asks[j])
		}
	}
	wg.Done()
}

func (w *WebsocketOrderbookLocal) updateBidsByPrice(base *orderbook.Base, assetType string, p currency.Pair, wg *sync.WaitGroup) {
	for j := range base.Bids {
		found := false
		for k := range w.orderbook[p][assetType].Bids {
			if w.orderbook[p][assetType].Bids[k].Price == base.Bids[j].Price {
				found = true
				if w.orderbook[p][assetType].Bids[j].Amount == 0 {
					w.orderbook[p][assetType].Bids = append(w.orderbook[p][assetType].Bids[:j],
						w.orderbook[p][assetType].Bids[j+1:]...)
					j--
					break
				}
				w.orderbook[p][assetType].Bids[k].Amount = base.Bids[j].Amount
				break
			}
		}
		if !found {
			w.orderbook[p][assetType].Bids = append(w.orderbook[p][assetType].Bids, base.Bids[j])
		}
	}
	wg.Done()
}

// NewBase creates an orderbook base for websocket use
func (w *WebsocketOrderbookLocal) NewBase(bidTargets, askTargets []orderbook.Item,
	p currency.Pair,
	updated time.Time,
	exchName, assetType string) (*orderbook.Base, error) {
	orderbookAddress := orderbook.Base{
		AssetType:    assetType,
		ExchangeName: exchName,
		Pair:         p,
		LastUpdated:  updated,
	}
	for x := range bidTargets {
		orderbookAddress.Bids = append(orderbookAddress.Bids, orderbook.Item{
			Price:  bidTargets[x].Price,
			Amount: bidTargets[x].Amount,
		})
	}
	for x := range askTargets {
		orderbookAddress.Asks = append(orderbookAddress.Asks, orderbook.Item{
			Price:  askTargets[x].Price,
			Amount: askTargets[x].Amount,
		})
	}
	return &orderbookAddress, nil
}

// LoadSnapshot loads initial snapshot of orderbook data, overite allows full
// orderbook to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook *orderbook.Base, exchName string, overwrite bool) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - snapshot ask and bids are nil")
	}
	w.m.Lock()
	defer w.m.Unlock()
	if w.orderbook == nil {
		w.orderbook = make(map[currency.Pair]map[string]*orderbook.Base)
	}
	if w.orderbook[newOrderbook.Pair] == nil {
		w.orderbook[newOrderbook.Pair] = make(map[string]*orderbook.Base)
	}
	if w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] == nil {
		w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	}
	if len(w.orderbook[newOrderbook.Pair][newOrderbook.AssetType].Asks) > 0 ||
		len(w.orderbook[newOrderbook.Pair][newOrderbook.AssetType].Bids) > 0 {
		if overwrite {
			w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
			return newOrderbook.Process()
		}
		return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - Snapshot instance already found")
	}
	w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	return newOrderbook.Process()
}

// UpdateUsingID updates orderbooks using specified ID
func (w *WebsocketOrderbookLocal) UpdateUsingID(bidTargets, askTargets []orderbook.Item,
	p currency.Pair,
	exchName, assetType, action string) error {
	w.m.Lock()
	defer w.m.Unlock()

	if _, ok := w.orderbook[p][assetType]; !ok {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			assetType,
			p.String())
	}
	switch action {
	case "update":
		for _, target := range bidTargets {
			for i := range w.orderbook[p][assetType].Bids {
				if w.orderbook[p][assetType].Bids[i].ID == target.ID {
					w.orderbook[p][assetType].Bids[i].Amount = target.Amount
					break
				}
			}
		}

		for _, target := range askTargets {
			for i := range w.orderbook[p][assetType].Asks {
				if w.orderbook[p][assetType].Asks[i].ID == target.ID {
					w.orderbook[p][assetType].Asks[i].Amount = target.Amount
					break
				}
			}
		}

	case "delete":
		for _, target := range bidTargets {
			for i := range w.orderbook[p][assetType].Bids {
				if w.orderbook[p][assetType].Bids[i].ID == target.ID {
					w.orderbook[p][assetType].Bids = append(w.orderbook[p][assetType].Bids[:i],
						w.orderbook[p][assetType].Bids[i+1:]...)
					break
				}
			}
		}

		for _, target := range askTargets {
			for i := range w.orderbook[p][assetType].Asks {
				if w.orderbook[p][assetType].Asks[i].ID == target.ID {
					w.orderbook[p][assetType].Asks = append(w.orderbook[p][assetType].Asks[:i],
						w.orderbook[p][assetType].Asks[i+1:]...)
					break
				}
			}
		}

	case "insert":
		w.orderbook[p][assetType].Bids = append(w.orderbook[p][assetType].Bids, bidTargets...)
		w.orderbook[p][assetType].Asks = append(w.orderbook[p][assetType].Asks, askTargets...)
	}

	return w.orderbook[p][assetType].Process()
}

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.orderbook = nil
	w.orderbookBuffer = nil
	w.m.Unlock()
}
