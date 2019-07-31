package ob

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const wsOrderbookBufferLimit = 5

// Update updates a local cache using bid targets and ask targets then updates
// main cache in ob.go
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *WebsocketOrderbookLocal) Update(orderbookUpdate *WebsocketOrderbookUpdate) error {
	if (orderbookUpdate.Bids == nil && orderbookUpdate.Asks == nil) ||
		(len(orderbookUpdate.Bids) == 0 && len(orderbookUpdate.Asks) == 0) {
		return errors.New("cannot have bids and ask targets both nil")
	}
	w.m.Lock()
	defer w.m.Unlock()
	if _, ok := w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]; !ok {
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			orderbookUpdate.ExchangeName,
			orderbookUpdate.CurrencyPair.String(),
			orderbookUpdate.AssetType)
	}
	if orderbookUpdate.BufferEnabled {
		err, underBufferLimit := w.ProcessBufferUpdate(orderbookUpdate)
		if err != nil {
			return err
		}
		if underBufferLimit {
			return nil
		}
	} else {
		w.ProcessObUpdate(orderbookUpdate)
	}
	err := w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Process()
	if err != nil {
		return err
	}
	if orderbookUpdate.BufferEnabled {
		// Reset the buffer
		w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = []WebsocketOrderbookUpdate{}
	}
	return nil
}

func (w *WebsocketOrderbookLocal) ProcessBufferUpdate(orderbookUpdate *WebsocketOrderbookUpdate) (error, bool) {
	if w.buffer == nil {
		w.buffer = make(map[currency.Pair]map[string][]WebsocketOrderbookUpdate)
	}
	if w.buffer[orderbookUpdate.CurrencyPair] == nil {
		w.buffer[orderbookUpdate.CurrencyPair] = make(map[string][]WebsocketOrderbookUpdate)
	}
	if len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) <= wsOrderbookBufferLimit {
		log.Debugf("%v adding to ob buffer %v %v %v/%v", orderbookUpdate.ExchangeName, orderbookUpdate.CurrencyPair, orderbookUpdate.AssetType, len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]), wsOrderbookBufferLimit)
		w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = append(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], *orderbookUpdate)
		if len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) < wsOrderbookBufferLimit {
			return nil, true
		}
	}
	// sort by last updated to ensure each update is in order
	if orderbookUpdate.OrderByUpdateIDs {
		sort.Slice(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
			return w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UpdateID < w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].UpdateID
		})
	} else {
		sort.Slice(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
			return w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UpdateTime.Before(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].UpdateTime)
		})
	}
	for i := 0; i < len(w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]); i++ {
		w.ProcessObUpdate(&w.buffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i])
	}
	return nil, false
}

func (w *WebsocketOrderbookLocal) ProcessObUpdate(orderbookUpdate *WebsocketOrderbookUpdate) {
	if orderbookUpdate.UpdateByIDs {
		w.DoTheThing(orderbookUpdate)
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
	wg.Done()
}

// LoadSnapshot loads initial snapshot of ob data, overite allows full
// ob to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook *orderbook.Base, exchName string, overwrite bool) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return errors.New("snapshot ask and bids are nil")
	}
	w.m.Lock()
	defer w.m.Unlock()
	if w.ob == nil {
		w.ob = make(map[currency.Pair]map[string]*orderbook.Base)
	}
	if w.ob[newOrderbook.Pair] == nil {
		w.ob[newOrderbook.Pair] = make(map[string]*orderbook.Base)
	}
	if w.ob[newOrderbook.Pair][newOrderbook.AssetType] != nil &&
		(len(w.ob[newOrderbook.Pair][newOrderbook.AssetType].Asks) > 0 ||
			len(w.ob[newOrderbook.Pair][newOrderbook.AssetType].Bids) > 0) {
		if overwrite {
			w.ob[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
			return newOrderbook.Process()
		}
		return errors.New("snapshot instance already found")
	}
	w.ob[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	return newOrderbook.Process()
}

// GetOrderbook retrieves the orderbook for validation
// TODO replace with dedicated ob validation
func (w *WebsocketOrderbookLocal) GetOrderbook(curr currency.Pair, assetType string) *orderbook.Base {
	return w.ob[curr][assetType]
}

// DoTheThing studies the thing,
// understands its true purpose,
// reflects on how it impacts the world around us.
//
// Then fucking does it
func (w *WebsocketOrderbookLocal) DoTheThing(orderbookUpdate *WebsocketOrderbookUpdate) {
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
			for i := range w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids {
				if w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].ID == target.ID {
					w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids = append(w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[:i],
						w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i+1:]...)
					i--
					break
				}
			}
		}

		for _, target := range orderbookUpdate.Asks {
			for i := range w.ob[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks {
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

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.ob = nil
	w.buffer = nil
	w.m.Unlock()
}
