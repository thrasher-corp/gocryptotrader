package stats

import (
	"errors"
	"sort"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errInvalidParams = errors.New("cannot add or update, invalid params")

// item holds various fields for storing currency pair stats
type item struct {
	Exchange  string
	Pair      currency.Pair
	AssetType asset.Item
	Price     float64
	Volume    float64
}

// items holds a match lookup and alignment
var items = struct {
	bucket []item
	match  map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*item
	mu     sync.Mutex
}{
	match: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*item),
}

// ByPrice allows sorting by price
type ByPrice []item

func (b ByPrice) Len() int {
	return len(b)
}

func (b ByPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b ByPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// ByVolume allows sorting by volume
type ByVolume []item

func (b ByVolume) Len() int {
	return len(b)
}

func (b ByVolume) Less(i, j int) bool {
	return b[i].Volume < b[j].Volume
}

func (b ByVolume) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Add adds or updates the item stats
func Add(exchName string, p currency.Pair, a asset.Item, price, volume float64) error {
	if exchName == "" || p.Base.IsEmpty() || p.Quote.IsEmpty() || !a.IsValid() || price <= 0 || volume <= 0 {
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
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*item)
		items.match[exchName] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*item)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*item)
		m2[p.Base.Item] = m3
	}

	data := m3[p.Quote.Item]
	if data != nil {
		data.Price = price
		data.Volume = volume
		return
	}
	items.bucket = append(items.bucket, item{exchName, p, a, price, volume})
	m3[p.Quote.Item] = &items.bucket[len(items.bucket)-1]
}

// SortExchangesByVolume sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByVolume(p currency.Pair, a asset.Item, reverse bool) []item {
	items.mu.Lock()
	defer items.mu.Unlock()

	result := make(ByVolume, 0, len(items.bucket))
	for x := range items.bucket {
		if items.bucket[x].Pair.EqualIncludeReciprocal(p) &&
			items.bucket[x].AssetType == a {
			result = append(result, items.bucket[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(result))
	} else {
		sort.Sort(result)
	}
	return result
}

// SortExchangesByPrice sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByPrice(p currency.Pair, a asset.Item, reverse bool) []item {
	items.mu.Lock()
	defer items.mu.Unlock()

	result := make(ByPrice, 0, len(items.bucket))
	for x := range items.bucket {
		if items.bucket[x].Pair.EqualIncludeReciprocal(p) &&
			items.bucket[x].AssetType == a {
			result = append(result, items.bucket[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(result))
	} else {
		sort.Sort(result)
	}
	return result
}
