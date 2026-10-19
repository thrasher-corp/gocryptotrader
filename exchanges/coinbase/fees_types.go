package coinbase

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Volume types, which name the trading volume a fee tier counts
const (
	VolumeTypeSpot          = "VOLUME_TYPE_SPOT"
	VolumeTypeUSDerivatives = "VOLUME_TYPE_US_DERIVATIVES"
	// VolumeTypeINTXPerpetuals is International Exchange perpetuals volume
	VolumeTypeINTXPerpetuals = "VOLUME_TYPE_INTX_PERPS"
	// VolumeTypeSimpleSpot, VolumeTypeSimpleFCM and VolumeTypeSimpleINTXPerpetuals are simple (RETAIL_SIMPLE) trading
	// volume, of spot, US expiring futures and INTX perpetuals
	VolumeTypeSimpleSpot           = "VOLUME_TYPE_SIMPLE_SPOT"
	VolumeTypeSimpleFCM            = "VOLUME_TYPE_SIMPLE_FCM"
	VolumeTypeSimpleINTXPerpetuals = "VOLUME_TYPE_SIMPLE_INTX_PERPS"
	// VolumeTypeEquity is stock and ETF volume, which counts apart from crypto spot volume
	VolumeTypeEquity       = "VOLUME_TYPE_EQUITY"
	VolumeTypeOptions      = "VOLUME_TYPE_OPTIONS"
	VolumeTypeDatedFutures = "VOLUME_TYPE_DATED_FUTURES"
)

// Goods and services tax types
const (
	// GoodsAndServicesTaxInclusive means the fee tier's rates include the tax, as in Singapore
	GoodsAndServicesTaxInclusive = "INCLUSIVE"
	// GoodsAndServicesTaxExclusive means the tax is charged on top of the fee tier's rates, as in Australia
	GoodsAndServicesTaxExclusive = "EXCLUSIVE"
)

// Fee tier qualification types, which name what qualifies the user for their fee tier without promotions
const (
	FeeTierQualificationTypeSpotVolume           = "FEE_TIER_QUALIFICATION_TYPE_SPOT_VOLUME"
	FeeTierQualificationTypeUSDerivativesVolume  = "FEE_TIER_QUALIFICATION_TYPE_US_DERIVATIVES_VOLUME"
	FeeTierQualificationTypeINTXPerpetualsVolume = "FEE_TIER_QUALIFICATION_TYPE_INTX_PERPS_VOLUME"
	// FeeTierQualificationTypeAssetsOnPlatform qualifies by the USD value of the user's assets on Coinbase
	FeeTierQualificationTypeAssetsOnPlatform = "FEE_TIER_QUALIFICATION_TYPE_AOP"
	// FeeTierQualificationTypeCombinedVolume qualifies by the sum of the volume types the fee tier's ranges name
	FeeTierQualificationTypeCombinedVolume = "FEE_TIER_QUALIFICATION_TYPE_COMBINED_VOLUME"
)

// TransactionSummaryRequest selects the products GetTransactionSummary summarises; a field left empty selects every
// value
type TransactionSummaryRequest struct {
	// ProductType is SPOT, FUTURE, EQUITY, OPTION_GROUP or FUTURE_GROUP
	ProductType string
	// ContractExpiryType selects expiring or perpetual futures, when ProductType is FUTURE
	ContractExpiryType string
	// ProductVenue is CBE, FCM or INTX
	ProductVenue string
}

// TransactionSummaryResponse is the user's fee tier and the trading volume and fees it is based on; volumes, fees and
// balances are in USD
type TransactionSummaryResponse struct {
	// TotalVolume is the volume across assets; the docs require it without describing it
	TotalVolume float64 `json:"total_volume"`
	TotalFees   float64 `json:"total_fees"`
	FeeTier     FeeTier `json:"fee_tier"`
	// MarginRate applies to the FUTURE product type alone
	MarginRate          DecimalValue         `json:"margin_rate"`
	GoodsAndServicesTax *GoodsAndServicesTax `json:"goods_and_services_tax"`
	// AdvancedTradeOnlyVolume and AdvancedTradeOnlyFees leave out Coinbase Pro's
	AdvancedTradeOnlyVolume float64 `json:"advanced_trade_only_volume"`
	AdvancedTradeOnlyFees   float64 `json:"advanced_trade_only_fees"`
	CoinbaseProVolume       float64 `json:"coinbase_pro_volume"`
	CoinbaseProFees         float64 `json:"coinbase_pro_fees"`
	// TotalBalance is the user's balance across spot, International Exchange and US derivatives
	TotalBalance types.Number `json:"total_balance"`
	// VolumeBreakdown is the volume of each type the fee tier counts
	VolumeBreakdown []FeeTierVolume `json:"volume_breakdown"`
	// HasCostPlusCommission is set when the user is on cost plus commission pricing
	HasCostPlusCommission bool `json:"has_cost_plus_commission"`
	// FeeTierWithoutPromotion is the fee tier the user's volume or assets qualify for, promotions aside
	FeeTierWithoutPromotion *FeeTierWithoutPromotion `json:"fee_tier_without_promotion"`
	// DeribitVenueData summarises each INTX derivatives product type, only for INTX derivatives
	DeribitVenueData *DeribitVenueData `json:"deribit_venue_data"`
}

// FeeTier is a fee tier's maker and taker rates and the bounds that qualify for it, of assets on platform (the USD
// value of the user's assets on Coinbase) or of 30 day volume; a bound Coinbase leaves empty decodes as zero
type FeeTier struct {
	PricingTier  string       `json:"pricing_tier"`
	TakerFeeRate types.Number `json:"taker_fee_rate"`
	MakerFeeRate types.Number `json:"maker_fee_rate"`
	// AssetsOnPlatformFrom is inclusive and AssetsOnPlatformTo exclusive
	AssetsOnPlatformFrom types.Number `json:"aop_from"`
	AssetsOnPlatformTo   types.Number `json:"aop_to"`
	// VolumeTypesAndRanges are the 30 day volume bounds of the tier, each for a sum of volume types
	VolumeTypesAndRanges []VolumeTypesAndRange `json:"volume_types_and_range"`
}

// VolumeTypesAndRange is a fee tier's bounds for the sum of the 30 day volume of some volume types
type VolumeTypesAndRange struct {
	VolumeTypes []string `json:"volume_types"`
	// VolumeFrom is inclusive and VolumeTo exclusive
	VolumeFrom types.Number `json:"vol_from"`
	VolumeTo   types.Number `json:"vol_to"`
}

// DecimalValue is a decimal that Coinbase's schema wraps in an object holding a decimal string, {"value":"0.5"}, and
// its example sends bare, 0.5; it decodes from either, and from null as zero
type DecimalValue float64

// decimalObject is the object Coinbase's schema wraps a decimal in
type decimalObject struct {
	Value types.Number `json:"value"`
}

// UnmarshalJSON decodes a decimal from the object holding it, or from a bare number or decimal string
func (d *DecimalValue) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		var obj decimalObject
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("%w: %s", errInvalidDecimalValue, data)
		}
		*d = DecimalValue(obj.Value)
		return nil
	}
	var n types.Number
	if err := n.UnmarshalJSON(data); err != nil {
		return fmt.Errorf("%w: %s", errInvalidDecimalValue, data)
	}
	*d = DecimalValue(n)
	return nil
}

// Float64 returns the decimal as a float64
func (d DecimalValue) Float64() float64 {
	return float64(d)
}

// GoodsAndServicesTax is the goods and services tax the user's fees bear
type GoodsAndServicesTax struct {
	Rate types.Number `json:"rate"`
	// Type is INCLUSIVE when the fee tier's rates include the tax and EXCLUSIVE when it is charged on top
	Type string `json:"type"`
}

// FeeTierVolume is the volume of a volume type that counts towards the user's fee tier
type FeeTierVolume struct {
	VolumeType string  `json:"volume_type"`
	Volume     float64 `json:"volume"`
}

// FeeTierWithoutPromotion is the fee tier the user's volume or assets on platform qualify for, promotional fees and
// incentives aside, and what reaching the next tier takes
type FeeTierWithoutPromotion struct {
	// PricingTier is deprecated in favour of CurrentTier's
	PricingTier string `json:"pricing_tier"`
	// QualificationType names what the user qualifies by, volume of some types or assets on platform
	QualificationType string `json:"qualification_type"`
	// CurrentValue is the user's volume or assets on platform, in USD
	CurrentValue types.Number `json:"current_value"`
	// NextTierThreshold is the value the next tier takes; Coinbase leaves it empty, so it decodes as zero, in the top
	// tier
	NextTierThreshold types.Number `json:"next_tier_threshold"`
	CurrentTier       *FeeTier     `json:"current_tier"`
	// NextTier is nil in the top tier
	NextTier *FeeTier `json:"next_tier"`
	// CurrentTierThreshold is the current tier's lower bound; zero when Coinbase cannot determine it
	CurrentTierThreshold types.Number `json:"current_tier_threshold"`
}

// DeribitVenueData summarises each INTX derivatives product type
type DeribitVenueData struct {
	Perpetuals   *ProductTypeTransactionSummary `json:"perps"`
	DatedFutures *ProductTypeTransactionSummary `json:"dated_futures"`
	Options      *ProductTypeTransactionSummary `json:"options"`
}

// ProductTypeTransactionSummary is the user's volume and fees in USD, and their fee tier, for an INTX derivatives
// product type
type ProductTypeTransactionSummary struct {
	TotalVolume float64  `json:"total_volume"`
	TotalFees   float64  `json:"total_fees"`
	FeeTier     *FeeTier `json:"fee_tier"`
}
