package assets

import (
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
)

// AssetType stores the asset type
type AssetType string

// AssetTypes stores a list of assets
type AssetTypes []AssetType

// Const vars for asset package
const (
	AssetTypeSpot                   = AssetType("spot")
	AssetTypeMargin                 = AssetType("margin")
	AssetTypeIndex                  = AssetType("index")
	AssetTypeBinary                 = AssetType("binary")
	AssetTypePerpetualContract      = AssetType("perpetualcontract")
	AssetTypePerpetualSwap          = AssetType("perpetualswap")
	AssetTypeFutures                = AssetType("futures")
	AssetTypeUpsideProfitContract   = AssetType("upsideprofitcontract")
	AssetTypeDownsideProfitContract = AssetType("downsideprofitcontract")
)

// Supported returns a list of supported asset types
func Supported() AssetTypes {
	var a AssetTypes
	a = append(a,
		AssetTypeSpot,
		AssetTypeMargin,
		AssetTypeIndex,
		AssetTypeBinary,
		AssetTypePerpetualContract,
		AssetTypePerpetualSwap,
		AssetTypeFutures,
		AssetTypeUpsideProfitContract,
		AssetTypeDownsideProfitContract,
	)
	return a
}

// returns an AssetType to string
func (a AssetType) String() string {
	return string(a)
}

// Strings converts an asset type array to a string array
func (a AssetTypes) Strings() []string {
	var assets []string
	for x := range a {
		assets = append(assets, string(a[x]))
	}
	return assets
}

// Contains returns whether or not the supplied asset exists
// in the list of AssetTypes
func (a AssetTypes) Contains(asset AssetType) bool {
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
func (a AssetTypes) JoinToString(separator string) string {
	return strings.Join(a.Strings(), separator)
}

// IsValid returns whether or not the supplied asset type is valid or
// not
func IsValid(input AssetType) bool {
	a := Supported()
	for x := range a {
		if strings.EqualFold(a[x].String(), input.String()) {
			return true
		}
	}
	return false
}

// New takes an input of asset types as string and returns an AssetTypes
// array
func New(input string) AssetTypes {
	if !common.StringContains(input, ",") {
		if IsValid(AssetType(input)) {
			return AssetTypes{
				AssetType(input),
			}
		}
		return nil
	}

	assets := strings.Split(input, ",")
	var result AssetTypes
	for x := range assets {
		if !IsValid(AssetType(assets[x])) {
			return nil
		}
		result = append(result, AssetType(assets[x]))
	}
	return result
}
