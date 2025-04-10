package asset

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Public errors related to assets
var (
	ErrNotSupported = errors.New("unsupported asset type")
	ErrNotEnabled   = errors.New("asset type not enabled")
	ErrInvalidAsset = errors.New("asset is invalid")
)

// Item stores the asset type
type Item uint32

// Items stores a list of assets types
type Items []Item

// Supported Assets
const (
	Empty Item = iota
	Spot
	Margin
	CrossMargin
	MarginFunding
	Index
	Binary
	// Futures asset consts must come below this comment for method `IsFutures`
	Futures
	PerpetualContract
	PerpetualSwap
	DeliveryFutures
	UpsideProfitContract
	DownsideProfitContract
	CoinMarginedFutures
	USDTMarginedFutures
	USDCMarginedFutures
	FutureCombo
	LinearContract
	Spread
	// Options asset consts must come below this comment for method `IsOptions`
	Options
	OptionCombo
	// All asset const must come immediately after all valid assets for method `IsValid`
	All
)

const (
	spot                   = "spot"
	margin                 = "margin"
	crossMargin            = "cross_margin"
	marginFunding          = "marginfunding"
	index                  = "index"
	binary                 = "binary"
	perpetualContract      = "perpetualcontract"
	perpetualSwap          = "perpetualswap"
	swap                   = "swap"
	spread                 = "spread"
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
	linearContract         = "linearcontract"
	all                    = "all"
)

var supportedList = Items{Spot, Margin, CrossMargin, MarginFunding, Index, Binary, PerpetualContract, PerpetualSwap, Futures, DeliveryFutures, UpsideProfitContract, DownsideProfitContract, CoinMarginedFutures, USDTMarginedFutures, USDCMarginedFutures, Options, LinearContract, OptionCombo, FutureCombo, Spread}

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
	case Spread:
		return spread
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
	case LinearContract:
		return linearContract
	case All:
		return all
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
	return slices.Contains(a, i)
}

// JoinToString joins an asset type array and converts it to a string
// with the supplied separator
func (a Items) JoinToString(separator string) string {
	return strings.Join(a.Strings(), separator)
}

// IsValid returns whether or not the supplied asset type is valid or not
func (a Item) IsValid() bool {
	return a > Empty && a < All
}

// IsFutures checks if the asset type is a futures contract based asset
func (a Item) IsFutures() bool {
	return a >= Futures && a < Options
}

// IsOptions checks if the asset type is options contract based asset
func (a Item) IsOptions() bool {
	return a >= Options && a < All
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
	case spread:
		return Spread, nil
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
	case linearContract:
		return LinearContract, nil
	case all:
		return All, nil
	default:
		return 0, fmt.Errorf("%w '%v', only supports %s", ErrNotSupported, input, supportedList)
	}
}

// UseDefault returns default asset type
func UseDefault() Item {
	return Spot
}
