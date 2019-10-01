package wsorderbook

import (
	"fmt"
	"sort"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Setup sets private variables
func (w *WebsocketOrderbookLocal) Setup(obBufferLimit int, bufferEnabled, sortBuffer, sortBufferByUpdateIDs, updateEntriesByID bool, exchangeName string) {
	w.obBufferLimit = obBufferLimit
	w.bufferEnabled = bufferEnabled
	w.sortBuffer = sortBuffer
	w.sortBufferByUpdateIDs = sortBufferByUpdateIDs
	w.updateEntriesByID = updateEntriesByID
	w.exchangeName = exchangeName
}

// Update updates a local cache using bid targets and ask targets then updates
// main orderbook
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *WebsocketOrderbookLocal) Update(orderbookUpdate *WebsocketOrderbookUpdate) error {
	if (orderbookUpdate.Bids == nil && orderbookUpdate.Asks == nil) ||
		(len(orderbookUpdate.Bids) == 0 && len(orderbookUpdate.Asks) == 0) {
		return fmt.Errorf("%v cannot have bids and ask targets both nil", w.exchangeName)
	}
	w.m.Lock()
	defer w.m.Unlock()
	if _, ok := w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]; !ok {
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			w.exchangeName,
			orderbookUpdate.CurrencyPair.String(),
			orderbookUpdate.AssetType)
	}
	if w.bufferEnabled {
		overBufferLimit := w.processBufferUpdate(orderbookUpdate)
		if !overBufferLimit {
			return nil
		}
	} else {
		w.processObUpdate(orderbookUpdate)
	}
	err := w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Process()
	if err != nil {
		return err
	}
	if w.bufferEnabled {
		// Reset the buffer
		w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = nil
	}
	return nil
}

func (w *WebsocketOrderbookLocal) processBufferUpdate(orderbookUpdate *WebsocketOrderbookUpdate) bool {
	if w.buffer == nil {
		w.buffer = make(map[currency.Pair]map[asset.Item][]WebsocketOrderbookUpdate)
	}
	if w.buffer[orderbookUpdate.CurrencyPair] == nil {
		w.buffer[orderbookUpdate.CurrencyPair] = make(map[asset.Item][]WebsocketOrderbookUpdate)
	}
	if len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) <= w.obBufferLimit {
		w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = append(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], *orderbookUpdate)
		if len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) < w.obBufferLimit {
			return false
		}
	}
	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
				return w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UpdateID < w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].UpdateID
			})
		} else {
			sort.Slice(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
				return w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UpdateTime.Before(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].UpdateTime)
			})
		}
	}
	for i := 0; i < len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]); i++ {
		w.processObUpdate(&w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i])
	}
	return true
}

func (w *WebsocketOrderbookLocal) processObUpdate(orderbookUpdate *WebsocketOrderbookUpdate) {
	if w.updateEntriesByID {
		w.updateByIDAndAction(orderbookUpdate)
	} else {
		var wg sync.WaitGroup
		wg.Add(2)
		go w.updateAsksByPrice(orderbookUpdate, &wg)
		go w.updateBidsByPrice(orderbookUpdate, &wg)
		wg.Wait()
	}
}

func (w *WebsocketOrderbookLocal) updateAsksByPrice(base *WebsocketOrderbookUpdate, wg *sync.WaitGroup) {
	for j := 0; j < len(base.Asks); j++ {
		found := false
		for k := 0; k < len(w.ob[base.CurrencyPair][base.AssetType].Asks); k++ {
			if w.ob[base.CurrencyPair][base.AssetType].Asks[k].Price == base.Asks[j].Price {
				found = true
				if base.Asks[j].Amount == 0 {
					w.ob[base.CurrencyPair][base.AssetType].Asks = append(w.ob[base.CurrencyPair][base.AssetType].Asks[:k],
						w.ob[base.CurrencyPair][base.AssetType].Asks[k+1:]...)
					break
				}
				w.ob[base.CurrencyPair][base.AssetType].Asks[k].Amount = base.Asks[j].Amount
				break
			}
		}
		if !found {
			w.ob[base.CurrencyPair][base.AssetType].Asks = append(w.ob[base.CurrencyPair][base.AssetType].Asks, base.Asks[j])
		}
	}
	sort.Slice(w.ob[base.CurrencyPair][base.AssetType].Asks, func(i, j int) bool {
		return w.ob[base.CurrencyPair][base.AssetType].Asks[i].Price < w.ob[base.CurrencyPair][base.AssetType].Asks[j].Price
	})
	wg.Done()
}

func (w *WebsocketOrderbookLocal) updateBidsByPrice(base *WebsocketOrderbookUpdate, wg *sync.WaitGroup) {
	for j := 0; j < len(base.Bids); j++ {
		found := false
		for k := 0; k < len(w.ob[base.CurrencyPair][base.AssetType].Bids); k++ {
			if w.ob[base.CurrencyPair][base.AssetType].Bids[k].Price == base.Bids[j].Price {
				found = true
				if base.Bids[j].Amount == 0 {
					w.ob[base.CurrencyPair][base.AssetType].Bids = append(w.ob[base.CurrencyPair][base.AssetType].Bids[:k],
						w.ob[base.CurrencyPair][base.AssetType].Bids[k+1:]...)
					break
				}
				w.ob[base.CurrencyPair][base.AssetType].Bids[k].Amount = base.Bids[j].Amount
				break
			}
		}
		if !found {
			w.ob[base.CurrencyPair][base.AssetType].Bids = append(w.ob[base.CurrencyPair][base.AssetType].Bids, base.Bids[j])
		}
	}
	sort.Slice(w.ob[base.CurrencyPair][base.AssetType].Bids, func(i, j int) bool {
		return w.ob[base.CurrencyPair][base.AssetType].Bids[i].Price > w.ob[base.CurrencyPair][base.AssetType].Bids[j].Price
	})
	wg.Done()
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (w *WebsocketOrderbookLocal) updateByIDAndAction(orderbookUpdate *WebsocketOrderbookUpdate) {
	switch orderbookUpdate.Action {
	case "update":
		for _, target := range orderbookUpdate.Bids {
			for i := range w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids {
				if w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].ID == target.ID {
					w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].Amount = target.Amount
					break
				}
			}
		}
		for _, target := range orderbookUpdate.Asks {
			for i := range w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks {
				if w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].ID == target.ID {
					w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].Amount = target.Amount
					break
				}
			}
		}
	case "delete":
		for _, target := range orderbookUpdate.Bids {
			for i := 0; i < len(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids); i++ {
				if w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].ID == target.ID {
					w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids = append(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[:i],
						w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i+1:]...)
					i--
					break
				}
			}
		}
		for _, target := range orderbookUpdate.Asks {
			for i := 0; i < len(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks); i++ {
				if w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].ID == target.ID {
					w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks = append(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[:i],
						w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i+1:]...)
					i--
					break
				}
			}
		}
	case "insert":
		w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids = append(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids, orderbookUpdate.Bids...)
		w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks = append(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks, orderbookUpdate.Asks...)
	}
}

// LoadSnapshot loads initial snapshot of ob data, overwrite allows full
// ob to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook *orderbook.Base) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return fmt.Errorf("%v snapshot ask and bids are nil", w.exchangeName)
	}
	w.m.Lock()
	defer w.m.Unlock()
	if w.ob == nil {
		w.ob = make(map[currency.Pair]map[asset.Item]*orderbook.Base)
	}
	if w.ob[newOrderbook.Pair] == nil {
		w.ob[newOrderbook.Pair] = make(map[asset.Item]*orderbook.Base)
	}
	if w.ob[newOrderbook.Pair][newOrderbook.AssetType] != nil &&
		(len(w.ob[newOrderbook.Pair][newOrderbook.AssetType].Asks) > 0 ||
			len(w.ob[newOrderbook.Pair][newOrderbook.AssetType].Bids) > 0) {
		w.ob[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
		return newOrderbook.Process()
	}
	w.ob[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	return newOrderbook.Process()
}

// GetOrderbook use sparingly. Modifying anything here will ruin hash calculation and cause problems
func (w *WebsocketOrderbookLocal) GetOrderbook(p currency.Pair, assetType asset.Item) *orderbook.Base {
	w.m.Lock()
	defer w.m.Unlock()
	return w.ob[p][assetType]
}

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.ob = nil
	w.buffer = nil
	w.m.Unlock()
}
