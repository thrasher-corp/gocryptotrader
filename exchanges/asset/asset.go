package asset

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotSupported is an error for an unsupported asset type
	ErrNotSupported = errors.New("unsupported asset type")
	// ErrNotEnabled is an error for an asset not enabled
	ErrNotEnabled = errors.New("asset type not enabled")
	// ErrInvalidAsset is returned when the assist isn't valid
	ErrInvalidAsset = errors.New("asset is invalid")
)

// Item stores the asset type
type Item uint32

// Items stores a list of assets types
type Items []Item

// Const vars for asset package
const (
	Empty Item = 0
	Spot  Item = 1 << iota
	Margin
	CrossMargin
	MarginFunding
	Index
	Binary
	PerpetualContract
	PerpetualSwap
	Futures
	DeliveryFutures
	UpsideProfitContract
	DownsideProfitContract
	CoinMarginedFutures
	USDTMarginedFutures
	USDCMarginedFutures
	Options
	OptionCombo
	FutureCombo

	// Added to represent a USDT and USDC based linear derivatives(futures/perpetual) assets in Bybit V5.
	LinearContract

	optionsFlag   = OptionCombo | Options
	futuresFlag   = PerpetualContract | PerpetualSwap | Futures | DeliveryFutures | UpsideProfitContract | DownsideProfitContract | CoinMarginedFutures | USDTMarginedFutures | USDCMarginedFutures | LinearContract | FutureCombo
	supportedFlag = Spot | Margin | CrossMargin | MarginFunding | Index | Binary | PerpetualContract | PerpetualSwap | Futures | DeliveryFutures | UpsideProfitContract | DownsideProfitContract | CoinMarginedFutures | USDTMarginedFutures | USDCMarginedFutures | Options | LinearContract | OptionCombo | FutureCombo

	spot                   = "spot"
	margin                 = "margin"
	crossMargin            = "cross_margin" // for Gateio exchange
	marginFunding          = "marginfunding"
	index                  = "index"
	binary                 = "binary"
	perpetualContract      = "perpetualcontract"
	perpetualSwap          = "perpetualswap"
	swap                   = "swap"
	futures                = "futures"
	deliveryFutures        = "delivery"
	upsideProfitContract   = "upsideprofitcontract"
	downsideProfitContract = "downsideprofitcontract"
	coinMarginedFutures    = "coinmarginedfutures"
	usdtMarginedFutures    = "usdtmarginedfutures"
	usdcMarginedFutures    = "usdcmarginedfutures"
	options                = "options"
	optionCombo            = "option_combo"
	futureCombo            = "future_combo"
)

var (
	supportedList = Items{Spot, Margin, CrossMargin, MarginFunding, Index, Binary, PerpetualContract, PerpetualSwap, Futures, DeliveryFutures, UpsideProfitContract, DownsideProfitContract, CoinMarginedFutures, USDTMarginedFutures, USDCMarginedFutures, Options, LinearContract, OptionCombo, FutureCombo}
)

// Supported returns a list of supported asset types
func Supported() Items {
	return supportedList
}

// String converts an Item to its string representation
func (a Item) String() string {
	switch a {
	case Spot:
		return spot
	case Margin:
		return margin
	case CrossMargin:
		return crossMargin
	case MarginFunding:
		return marginFunding
	case Index:
		return index
	case Binary:
		return binary
	case PerpetualContract:
		return perpetualContract
	case PerpetualSwap:
		return perpetualSwap
	case Futures:
		return futures
	case DeliveryFutures:
		return deliveryFutures
	case UpsideProfitContract:
		return upsideProfitContract
	case DownsideProfitContract:
		return downsideProfitContract
	case CoinMarginedFutures:
		return coinMarginedFutures
	case USDTMarginedFutures:
		return usdtMarginedFutures
	case USDCMarginedFutures:
		return usdcMarginedFutures
	case Options:
		return options
	case OptionCombo:
		return optionCombo
	case FutureCombo:
		return futureCombo
	default:
		return ""
	}
}

// Strings converts an asset type array to a string array
func (a Items) Strings() []string {
	assets := make([]string, len(a))
	for x := range a {
		assets[x] = a[x].String()
	}
	return assets
}

// Contains returns whether or not the supplied asset exists
// in the list of Items
func (a Items) Contains(i Item) bool {
	if i.IsValid() {
		for x := range a {
			if a[x] == i {
				return true
			}
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
	return a != Empty && supportedFlag&a == a
}

// UnmarshalJSON conforms type to the umarshaler interface
func (a *Item) UnmarshalJSON(d []byte) error {
	var assetString string
	err := json.Unmarshal(d, &assetString)
	if err != nil {
		return err
	}

	if assetString == "" {
		return nil
	}

	ai, err := New(assetString)
	if err != nil {
		return err
	}

	*a = ai
	return nil
}

// MarshalJSON conforms type to the marshaller interface
func (a Item) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// New takes an input matches to relevant package assets
func New(input string) (Item, error) {
	input = strings.ToLower(input)
	switch input {
	case spot:
		return Spot, nil
	case margin:
		return Margin, nil
	case marginFunding:
		return MarginFunding, nil
	case crossMargin:
		return CrossMargin, nil
	case deliveryFutures:
		return DeliveryFutures, nil
	case index:
		return Index, nil
	case binary:
		return Binary, nil
	case perpetualContract:
		return PerpetualContract, nil
	case perpetualSwap, swap:
		return PerpetualSwap, nil
	case futures:
		return Futures, nil
	case upsideProfitContract:
		return UpsideProfitContract, nil
	case downsideProfitContract:
		return DownsideProfitContract, nil
	case coinMarginedFutures:
		return CoinMarginedFutures, nil
	case usdtMarginedFutures:
		return USDTMarginedFutures, nil
	case usdcMarginedFutures:
		return USDCMarginedFutures, nil
	case options, "option":
		return Options, nil
	case optionCombo:
		return OptionCombo, nil
	case futureCombo:
		return FutureCombo, nil
	default:
		return 0, fmt.Errorf("%w '%v', only supports %s",
			ErrNotSupported,
			input,
			supportedList)
	}
}

// UseDefault returns default asset type
func UseDefault() Item {
	return Spot
}

// IsFutures checks if the asset type is a futures contract based asset
func (a Item) IsFutures() bool {
	return a != Empty && futuresFlag&a == a
}

// IsOptions checks if the asset type is options contract based asset
func (a Item) IsOptions() bool {
	return a != Empty && optionsFlag&a == a
}
