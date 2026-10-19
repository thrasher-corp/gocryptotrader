package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetTransactionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransactionSummary must reject a nil request")
	_, err = e.GetTransactionSummary(t.Context(), &TransactionSummaryRequest{ProductType: "UNKNOWN_PRODUCT_TYPE"})
	require.ErrorIs(t, err, errInvalidTransactionSummaryProductType, "GetTransactionSummary must refuse an undocumented product type")
	_, err = e.GetTransactionSummary(t.Context(), &TransactionSummaryRequest{ContractExpiryType: "UNKNOWN_CONTRACT_EXPIRY_TYPE"})
	require.ErrorIs(t, err, errInvalidTransactionSummaryContractExpiryType, "GetTransactionSummary must refuse an undocumented contract expiry type")
	_, err = e.GetTransactionSummary(t.Context(), &TransactionSummaryRequest{ProductVenue: ProductVenueDeribit})
	require.ErrorIs(t, err, errInvalidTransactionSummaryProductVenue, "GetTransactionSummary must refuse an undocumented product venue")

	skipLiveWithoutCredentials(t)
	req := &TransactionSummaryRequest{}
	if mockTests {
		// The docs' example answers an INTX derivatives request, the only one Coinbase sends venue data for
		req = &TransactionSummaryRequest{ProductType: ProductTypeFuture, ContractExpiryType: ContractExpiryTypePerpetual, ProductVenue: ProductVenueINTX}
	}
	resp, err := e.GetTransactionSummary(t.Context(), req)
	require.NoError(t, err, "GetTransactionSummary must not error")
	if mockTests {
		tier := &FeeTier{
			PricingTier:        "<$10k",
			TakerFeeRate:       0.001,
			MakerFeeRate:       0.002,
			AssetsOnPlatformTo: 10000,
			VolumeTypesAndRanges: []VolumeTypesAndRange{
				{VolumeTypes: []string{VolumeTypeSpot, VolumeTypeUSDerivatives}, VolumeTo: 50000},
			},
		}
		nextTier := &FeeTier{
			PricingTier:          "$10k-$50k",
			TakerFeeRate:         0.0008,
			MakerFeeRate:         0.0016,
			AssetsOnPlatformFrom: 10000,
			AssetsOnPlatformTo:   50000,
			VolumeTypesAndRanges: []VolumeTypesAndRange{
				{VolumeTypes: []string{VolumeTypeSpot, VolumeTypeUSDerivatives}, VolumeFrom: 50000, VolumeTo: 100000},
			},
		}
		exp := &TransactionSummaryResponse{
			TotalVolume:             1000,
			TotalFees:               25,
			FeeTier:                 *tier,
			MarginRate:              0.5,
			GoodsAndServicesTax:     &GoodsAndServicesTax{Rate: 0.1, Type: GoodsAndServicesTaxExclusive},
			AdvancedTradeOnlyVolume: 1000,
			AdvancedTradeOnlyFees:   25,
			CoinbaseProVolume:       1000,
			CoinbaseProFees:         25,
			TotalBalance:            1000,
			VolumeBreakdown:         []FeeTierVolume{{VolumeType: VolumeTypeSpot, Volume: 1000}},
			FeeTierWithoutPromotion: &FeeTierWithoutPromotion{
				PricingTier:          "Advanced 3",
				QualificationType:    FeeTierQualificationTypeSpotVolume,
				CurrentValue:         25000,
				NextTierThreshold:    50000,
				CurrentTier:          tier,
				NextTier:             nextTier,
				CurrentTierThreshold: 10000,
			},
			DeribitVenueData: &DeribitVenueData{
				Perpetuals:   &ProductTypeTransactionSummary{TotalVolume: 1000, TotalFees: 25, FeeTier: tier},
				DatedFutures: &ProductTypeTransactionSummary{TotalVolume: 2000, TotalFees: 50, FeeTier: tier},
				Options:      &ProductTypeTransactionSummary{TotalVolume: 3000, TotalFees: 75, FeeTier: nextTier},
			},
		}
		assert.Equal(t, exp, resp, "GetTransactionSummary should decode every field")
	} else {
		assert.NotEmpty(t, resp.FeeTier.PricingTier, "GetTransactionSummary should return the user's fee tier")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetTransactionSummary(t.Context(), &TransactionSummaryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransactionSummary should error for a null response")
	_, err = newUnavailableServerExchange(t).GetTransactionSummary(t.Context(), &TransactionSummaryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransactionSummary should return the request failure")
}

func TestTransactionSummaryRequestParams(t *testing.T) {
	t.Parallel()
	params, err := (&TransactionSummaryRequest{ProductType: ProductTypeEquity, ContractExpiryType: ContractExpiryTypeExpiring, ProductVenue: ProductVenueCBE}).params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, "contract_expiry_type=EXPIRING&product_type=EQUITY&product_venue=CBE", params.Encode(), "params should send every field")
	params, err = (&TransactionSummaryRequest{}).params()
	require.NoError(t, err, "params must not error for an empty request")
	assert.Empty(t, params, "params should send nothing for an empty request")

	for _, productType := range []string{ProductTypeSpot, ProductTypeFuture, ProductTypeEquity, ProductTypeOptionGroup, ProductTypeFutureGroup} {
		_, err := (&TransactionSummaryRequest{ProductType: productType}).params()
		assert.NoErrorf(t, err, "params should accept product type %s", productType)
	}
	for _, expiryType := range []string{ContractExpiryTypeExpiring, ContractExpiryTypePerpetual} {
		_, err := (&TransactionSummaryRequest{ContractExpiryType: expiryType}).params()
		assert.NoErrorf(t, err, "params should accept contract expiry type %s", expiryType)
	}
	for _, venue := range []string{ProductVenueCBE, ProductVenueFCM, ProductVenueINTX} {
		_, err := (&TransactionSummaryRequest{ProductVenue: venue}).params()
		assert.NoErrorf(t, err, "params should accept product venue %s", venue)
	}
	// Coinbase's enums are upper case
	_, err = (&TransactionSummaryRequest{ProductType: "spot"}).params()
	assert.ErrorIs(t, err, errInvalidTransactionSummaryProductType, "params should refuse a lower case product type")
}

func TestDecimalValueUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]DecimalValue{
		`{"value":"0.5"}`: 0.5,
		`{"value":""}`:    0,
		`{}`:              0,
		`0.5`:             0.5,
		`"0.5"`:           0.5,
		`null`:            0,
	} {
		d := DecimalValue(1)
		require.NoErrorf(t, d.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for %s", data)
		assert.Equalf(t, exp, d, "UnmarshalJSON should decode %s", data)
	}
	var d DecimalValue
	assert.ErrorIs(t, d.UnmarshalJSON([]byte(`true`)), errInvalidDecimalValue, "UnmarshalJSON should refuse a boolean")
	assert.ErrorIs(t, d.UnmarshalJSON([]byte(`{"value":true}`)), errInvalidDecimalValue, "UnmarshalJSON should refuse an object holding no decimal")
	assert.ErrorIs(t, d.UnmarshalJSON([]byte(`{"value":`)), errInvalidDecimalValue, "UnmarshalJSON should refuse a malformed object")
	assert.Equal(t, 0.5, DecimalValue(0.5).Float64(), "Float64 should return the decimal")
}
