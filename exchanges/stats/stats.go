package stats

import (
	"errors"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Add adds or updates the Item stats
func Add(exchange string, p currency.Pair, a asset.Item, price, volume float64) error {
	if exchange == "" ||
		a == asset.Empty ||
		price == 0 ||
		volume == 0 ||
		p.Base.IsEmpty() ||
		p.Quote.IsEmpty() {
		return errors.New("cannot add or update, invalid params")
	}

	if p.Base.Equal(currency.XBT) {
		newPair, err := currency.NewPairFromStrings(currency.BTC.String(),
			p.Quote.String())
		if err != nil {
			return err
		}
		Append(exchange, newPair, a, price, volume)
	}

	if p.Quote.Equal(currency.USDT) {
		newPair, err := currency.NewPairFromStrings(p.Base.String(), currency.USD.String())
		if err != nil {
			return err
		}
		Append(exchange, newPair, a, price, volume)
	}

	Append(exchange, p, a, price, volume)
	return nil
}

// Append adds the Item stats for a specific currency pair and asset type
// if it doesn't exist
func Append(exchange string, p currency.Pair, a asset.Item, price, volume float64) {
	statMutex.Lock()
	defer statMutex.Unlock()
	if alreadyExistsRequiresLock(exchange, p, a, price, volume) {
		return
	}
	i := Item{
		Exchange:  exchange,
		Pair:      p,
		AssetType: a,
		Price:     price,
		Volume:    volume,
	}

	items = append(items, i)
}

// alreadyExistsRequiresLock checks to see if Item info already exists
// requires a locking beforehand because of globals
func alreadyExistsRequiresLock(exchange string, p currency.Pair, assetType asset.Item, price, volume float64) bool {
	for i := range items {
		if items[i].Exchange == exchange &&
			items[i].Pair.EqualIncludeReciprocal(p) &&
			items[i].AssetType == assetType {
			items[i].Price, items[i].Volume = price, volume
			return true
		}
	}
	return false
}

// AlreadyExists checks to see if Item info already exists
// for a specific currency pair and asset type
func AlreadyExists(exchange string, p currency.Pair, assetType asset.Item, price, volume float64) bool {
	statMutex.Lock()
	defer statMutex.Unlock()
	return alreadyExistsRequiresLock(exchange, p, assetType, price, volume)
}

// SortExchangesByVolume sorts Item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByVolume(p currency.Pair, assetType asset.Item, reverse bool) []Item {
	var result []Item
	statMutex.Lock()
	defer statMutex.Unlock()
	for x := range items {
		if items[x].Pair.EqualIncludeReciprocal(p) &&
			items[x].AssetType == assetType {
			result = append(result, items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(byVolume(result)))
	} else {
		sort.Sort(byVolume(result))
	}
	return result
}

// SortExchangesByPrice sorts Item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByPrice(p currency.Pair, assetType asset.Item, reverse bool) []Item {
	var result []Item
	statMutex.Lock()
	defer statMutex.Unlock()
	for x := range items {
		if items[x].Pair.EqualIncludeReciprocal(p) &&
			items[x].AssetType == assetType {
			result = append(result, items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(byPrice(result)))
	} else {
		sort.Sort(byPrice(result))
	}
	return result
}

func (b byPrice) Len() int {
	return len(b)
}

func (b byPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b byPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byVolume) Len() int {
	return len(b)
}

func (b byVolume) Less(i, j int) bool {
	return b[i].Volume < b[j].Volume
}

func (b byVolume) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
