package asset

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotSupported is an error for an unsupported asset type
	ErrNotSupported = errors.New("received unsupported asset type")
)

// Item stores the asset type
type Item string

// Items stores a list of assets types
type Items []Item

// Const vars for asset package
const (
	Spot                   = Item("spot")
	Margin                 = Item("margin")
	MarginFunding          = Item("marginfunding")
	Index                  = Item("index")
	Binary                 = Item("binary")
	PerpetualContract      = Item("perpetualcontract")
	PerpetualSwap          = Item("perpetualswap")
	Futures                = Item("futures")
	UpsideProfitContract   = Item("upsideprofitcontract")
	DownsideProfitContract = Item("downsideprofitcontract")
	CoinMarginedFutures    = Item("coinmarginedfutures")
	USDTMarginedFutures    = Item("usdtmarginedfutures")
)

var supported = Items{
	Spot,
	Margin,
	MarginFunding,
	Index,
	Binary,
	PerpetualContract,
	PerpetualSwap,
	Futures,
	UpsideProfitContract,
	DownsideProfitContract,
	CoinMarginedFutures,
	USDTMarginedFutures,
}

// Supported returns a list of supported asset types
func Supported() Items {
	return supported
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
func (a Items) Contains(i Item) bool {
	if !i.IsValid() {
		return false
	}

	for x := range a {
		if a[x].String() == i.String() {
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
func (a Item) IsValid() bool {
	for x := range supported {
		if supported[x].String() == a.String() {
			return true
		}
	}
	return false
}

// New takes an input matches to relevant package assets
func New(input string) (Item, error) {
	input = strings.ToLower(input)
	for i := range supported {
		if string(supported[i]) == input {
			return supported[i], nil
		}
	}
	return "", fmt.Errorf("%w %v, only supports %v",
		ErrNotSupported,
		input,
		supported)
}

// UseDefault returns default asset type
func UseDefault() Item {
	return Spot
}
