package stats

import (
	"errors"
	"sort"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errInvalidParams = errors.New("cannot add or update, invalid params")

// Item holds various fields for storing currency pair stats
type Item struct {
	Exchange  string
	Pair      currency.Pair
	AssetType asset.Item
	Price     float64
	Volume    float64
}

// items holds a match lookup and alignment
var items = struct {
	// The bucket field is a slice containing all ticker items that have been updated.
	bucket []Item
	// The match field is a map that allows for fast lookup of ticker items by address.
	match map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Item
	// The mu field is a mutex used to synchronize access to the bucket and match fields.
	mu sync.Mutex
}{
	match: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Item),
}

// Add adds or updates the item stats
func Add(exchName string, p currency.Pair, a asset.Item, price, volume float64) error {
	if exchName == "" || !p.IsPopulated() || !a.IsValid() || price <= 0 || volume <= 0 {
		return errInvalidParams
	}

	if p.Base.Equal(currency.XBT) {
		similarMatch := currency.NewPair(currency.BTC, p.Quote)
		update(exchName, similarMatch, a, price, volume)
	}

	if p.Quote.Equal(currency.USDT) {
		similarMatch := currency.NewPair(p.Base, currency.USD)
		update(exchName, similarMatch, a, price, volume)
	}
	update(exchName, p, a, price, volume)
	return nil
}

// update adds or updates the item stats for a specific currency pair and asset
func update(exchName string, p currency.Pair, a asset.Item, price, volume float64) {
	items.mu.Lock()
	defer items.mu.Unlock()
	m1, ok := items.match[exchName]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Item)
		items.match[exchName] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Item)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Item)
		m2[p.Base.Item] = m3
	}

	data := m3[p.Quote.Item]
	if data != nil {
		data.Price = price
		data.Volume = volume
		return
	}
	// If not found append item to the bucket list
	items.bucket = append(items.bucket, Item{exchName, p, a, price, volume})
	// Take last address entered and use for lookup table item for faster
	// matching.
	m3[p.Quote.Item] = &items.bucket[len(items.bucket)-1]
}

// SortExchangesByVolume sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByVolume(p currency.Pair, a asset.Item, reverse bool) []Item {
	// NOTE: Opted to not pre-alloc here because its only going to be the number
	// of enabled exchanges and the underlying bucket can be in excess of
	// thousands.
	var result []Item
	items.mu.Lock()
	for x := range items.bucket {
		if items.bucket[x].Pair.EqualIncludeReciprocal(p) &&
			items.bucket[x].AssetType == a {
			result = append(result, items.bucket[x])
		}
	}
	items.mu.Unlock()

	if reverse {
		sort.Slice(result, func(i, j int) bool { return result[i].Volume > result[j].Volume })
	} else {
		sort.Slice(result, func(i, j int) bool { return result[i].Volume < result[j].Volume })
	}
	return result
}

// SortExchangesByPrice sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByPrice(p currency.Pair, a asset.Item, reverse bool) []Item {
	// NOTE: Opted to not pre-alloc here because its only going to be the number
	// of enabled exchanges and the underlying bucket can be in excess of
	// thousands.
	var result []Item
	items.mu.Lock()
	for x := range items.bucket {
		if items.bucket[x].Pair.EqualIncludeReciprocal(p) &&
			items.bucket[x].AssetType == a {
			result = append(result, items.bucket[x])
		}
	}
	items.mu.Unlock()

	if reverse {
		sort.Slice(result, func(i, j int) bool { return result[i].Price > result[j].Price })
	} else {
		sort.Slice(result, func(i, j int) bool { return result[i].Price < result[j].Price })
	}
	return result
}
