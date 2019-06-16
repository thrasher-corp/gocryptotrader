package asset

import (
	"strings"
)

// Item stores the asset type
type Item string

// Items stores a list of assets types
type Items []Item

// Const vars for asset package
const (
	Spot                   = Item("spot")
	Margin                 = Item("margin")
	Index                  = Item("index")
	Binary                 = Item("binary")
	PerpetualContract      = Item("perpetualcontract")
	PerpetualSwap          = Item("perpetualswap")
	Futures                = Item("futures")
	UpsideProfitContract   = Item("upsideprofitcontract")
	DownsideProfitContract = Item("downsideprofitcontract")
)

// Supported returns a list of supported asset types
func Supported() Items {
	var a Items
	a = append(a,
		Spot,
		Margin,
		Index,
		Binary,
		PerpetualContract,
		PerpetualSwap,
		Futures,
		UpsideProfitContract,
		DownsideProfitContract,
	)
	return a
}

// returns an Item to string
func (a Item) String() string {
	return string(a)
}

// Strings converts an asset type array to a string array
func (a Items) Strings() []string {
	var assets []string
	for x := range a {
		assets = append(assets, string(a[x]))
	}
	return assets
}

// Contains returns whether or not the supplied asset exists
// in the list of Items
func (a Items) Contains(asset Item) bool {
	if !IsValid(asset) {
		return false
	}

	for x := range a {
		if a[x] == asset {
			return true
		}
	}

	return false
}

// JoinToString joins an asset type array and converts it to a string
// with the supplied separator
func (a Items) JoinToString(separator string) string {
	return strings.Join(a.Strings(), separator)
}

// IsValid returns whether or not the supplied asset type is valid or
// not
func IsValid(input Item) bool {
	a := Supported()
	for x := range a {
		if strings.EqualFold(a[x].String(), input.String()) {
			return true
		}
	}
	return false
}

// New takes an input of asset types as string and returns an Items
// array
func New(input string) Items {
	if !strings.Contains(input, ",") {
		if IsValid(Item(input)) {
			return Items{
				Item(input),
			}
		}
		return nil
	}

	assets := strings.Split(input, ",")
	var result Items
	for x := range assets {
		if !IsValid(Item(assets[x])) {
			return nil
		}
		result = append(result, Item(assets[x]))
	}
	return result
}
