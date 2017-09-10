package stats

import (
	"sort"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

// Item holds various fields for storing currency pair stats
type Item struct {
	Exchange  string
	Pair      pair.CurrencyPair
	AssetType string
	Price     float64
	Volume    float64
}

// Items var array
var Items []Item

// ByPrice allows sorting by price
type ByPrice []Item

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
type ByVolume []Item

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
func Add(exchange string, p pair.CurrencyPair, assetType string, price, volume float64) {
	if exchange == "" || assetType == "" || price == 0 || volume == 0 || p.FirstCurrency == "" || p.SecondCurrency == "" {
		return
	}

	if p.FirstCurrency == "XBT" {
		newPair := pair.NewCurrencyPair("BTC", p.SecondCurrency.String())
		Append(exchange, newPair, assetType, price, volume)
	}

	if p.SecondCurrency == "USDT" {
		newPair := pair.NewCurrencyPair(p.FirstCurrency.String(), "USD")
		Append(exchange, newPair, assetType, price, volume)
	}

	Append(exchange, p, assetType, price, volume)
}

// Append adds or updates the item stats for a specific
// currency pair and asset type
func Append(exchange string, p pair.CurrencyPair, assetType string, price, volume float64) {
	if AlreadyExists(exchange, p, assetType, price, volume) {
		return
	}

	i := Item{
		Exchange:  exchange,
		Pair:      p,
		AssetType: assetType,
		Price:     price,
		Volume:    volume,
	}

	Items = append(Items, i)
}

// AlreadyExists checks to see if item info already exists
// for a specific currency pair and asset type
func AlreadyExists(exchange string, p pair.CurrencyPair, assetType string, price, volume float64) bool {
	for i := range Items {
		if Items[i].Exchange == exchange && Items[i].Pair.Equal(p) && Items[i].AssetType == assetType {
			Items[i].Price, Items[i].Volume = price, volume
			return true
		}
	}
	return false
}

// SortExchangesByVolume sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByVolume(p pair.CurrencyPair, assetType string, reverse bool) []Item {
	var result []Item
	for x := range Items {
		if Items[x].Pair.Equal(p) && Items[x].AssetType == assetType {
			result = append(result, Items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByVolume(result)))
	} else {
		sort.Sort(ByVolume(result))
	}
	return result
}

// SortExchangesByPrice sorts item info by volume for a specific
// currency pair and asset type. Reverse will reverse the order from lowest to
// highest
func SortExchangesByPrice(p pair.CurrencyPair, assetType string, reverse bool) []Item {
	var result []Item
	for x := range Items {
		if Items[x].Pair.Equal(p) && Items[x].AssetType == assetType {
			result = append(result, Items[x])
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByPrice(result)))
	} else {
		sort.Sort(ByPrice(result))
	}
	return result
}
