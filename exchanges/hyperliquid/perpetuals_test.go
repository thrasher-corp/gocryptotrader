package hyperliquid

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testDefaultMetadata is the default perpetual DEX metadata the mock responses return
func testDefaultMetadata() PerpetualMetadataResponse {
	return PerpetualMetadataResponse{
		Universe: []PerpetualAssetMetadata{
			{Name: "BTC", SizeDecimals: 5, MaxLeverage: 40, MarginTableID: 56},
			{Name: "ETH", SizeDecimals: 4, MaxLeverage: 25, MarginTableID: 55},
			{Name: "MATIC", SizeDecimals: 1, MaxLeverage: 20, MarginTableID: 20, IsDelisted: true},
			{Name: "HPOS", MaxLeverage: 3, MarginTableID: 3, MarginMode: "strictIsolated"},
		},
		MarginTables: []MarginTableEntry{
			{ID: 50, Table: MarginTable{MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 50}}}},
			{ID: 51, Table: MarginTable{Description: "tiered 10x", MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 10}, {LowerBound: 3000000, MaxLeverage: 5}}}},
		},
	}
}

// testXYZMetadata is the xyz builder DEX metadata the mock responses return
func testXYZMetadata() PerpetualMetadataResponse {
	return PerpetualMetadataResponse{
		Universe: []PerpetualAssetMetadata{
			{
				Name:                     "xyz:XYZ100",
				SizeDecimals:             4,
				MaxLeverage:              30,
				MarginTableID:            30,
				GrowthMode:               "enabled",
				LastGrowthModeChangeTime: ZonelessTime(time.Date(2025, 11, 23, 17, 21, 40, 390706535, time.UTC)),
				DeployerFeeScale:         1,
				LastFeeScaleChangeTime:   ZonelessTime(time.Date(2025, 11, 23, 17, 37, 10, 33211662, time.UTC)),
			},
			{Name: "xyz:TSLA", SizeDecimals: 3, MaxLeverage: 20, MarginTableID: 20, MarginMode: "noCross"},
		},
		MarginTables: []MarginTableEntry{{ID: 30, Table: MarginTable{MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 30}}}}},
	}
}

func TestGetPerpetualDEXs(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualDEXs(t.Context())
	require.NoError(t, err, "GetPerpetualDEXs must not error")
	if mockTests {
		exp := []*PerpetualDEXResponse{
			nil,
			{
				Name:                            "xyz",
				FullName:                        "XYZ",
				Deployer:                        "0x0000000000000000000000000000000000000001",
				OracleUpdater:                   "0x0000000000000000000000000000000000000002",
				FeeRecipient:                    "0x0000000000000000000000000000000000000003",
				AssetToStreamingOpenInterestCap: []AssetValue{{Coin: "xyz:XYZ100", Value: 25000000}},
				SubDeployers:                    []SubDeployer{{Action: "setOracle", Addresses: []string{"0x0000000000000000000000000000000000000004"}}},
				AssetToFundingMultiplier:        []AssetValue{{Coin: "xyz:XYZ100", Value: 0.5}},
				AssetToFundingInterestRate:      []AssetValue{{Coin: "xyz:XYZ100", Value: 0.0000456621}},
				AssetToFundingClamp:             []AssetValue{{Coin: "xyz:XYZ100", Value: 0.0003}},
			},
			{
				Name:                            "flx",
				FullName:                        "Felix Exchange",
				Deployer:                        "0x0000000000000000000000000000000000000005",
				OracleUpdater:                   "0x0000000000000000000000000000000000000006",
				FeeRecipient:                    "0x0000000000000000000000000000000000000007",
				AssetToStreamingOpenInterestCap: []AssetValue{{Coin: "flx:TSLA", Value: 8000000}},
				SubDeployers:                    []SubDeployer{{Action: "registerAsset", Addresses: []string{"0x0000000000000000000000000000000000000008", "0x0000000000000000000000000000000000000009"}}},
				AssetToFundingMultiplier:        []AssetValue{{Coin: "flx:TSLA", Value: 1}},
				AssetToFundingInterestRate:      []AssetValue{{Coin: "flx:TSLA", Value: 0}},
				AssetToFundingClamp:             []AssetValue{{Coin: "flx:TSLA", Value: 0.0005}},
			},
		}
		assert.Equal(t, exp, result, "GetPerpetualDEXs should decode every field")
	} else {
		assert.Greater(t, len(result), 1, "GetPerpetualDEXs should list builder DEXs")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualDEXs(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualDEXs should return the request failure")
	for _, body := range []string{`[]`, `[{"name":"xyz"}]`} {
		_, err := newResponseServerExchange(t, body).GetPerpetualDEXs(t.Context())
		assert.ErrorIsf(t, err, errUnexpectedResponseLength, "GetPerpetualDEXs should reject a registry %s without the default DEX first", body)
	}
}

func TestGetPerpetualMetadata(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualMetadata(t.Context(), "")
	require.NoError(t, err, "GetPerpetualMetadata must not error")
	if mockTests {
		exp := testDefaultMetadata()
		assert.Equal(t, &exp, result, "GetPerpetualMetadata should decode every field")
	} else {
		assert.Equal(t, "BTC", result.Universe[0].Name, "GetPerpetualMetadata should list BTC first")
	}

	result, err = e.GetPerpetualMetadata(t.Context(), "xyz")
	require.NoError(t, err, "GetPerpetualMetadata must not error for a builder DEX")
	if mockTests {
		exp := testXYZMetadata()
		assert.Equal(t, &exp, result, "GetPerpetualMetadata should decode the builder DEX's fields")
	} else {
		assert.NotEmpty(t, result.Universe, "GetPerpetualMetadata should return the builder DEX's markets")
	}

	result, err = e.GetPerpetualMetadata(t.Context(), "flx")
	require.NoError(t, err, "GetPerpetualMetadata must not error for a builder DEX with non-USDC collateral")
	if mockTests {
		exp := &PerpetualMetadataResponse{
			Universe: []PerpetualAssetMetadata{{
				Name:                   "flx:TSLA",
				SizeDecimals:           2,
				MaxLeverage:            10,
				MarginTableID:          10,
				GrowthMode:             "enabled",
				DeployerFeeScale:       1,
				LastFeeScaleChangeTime: ZonelessTime(time.Date(2025, 11, 24, 21, 19, 25, 980742012, time.UTC)),
			}},
			MarginTables:    []MarginTableEntry{{ID: 10, Table: MarginTable{MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 10}}}}},
			CollateralToken: 360,
		}
		assert.Equal(t, exp, result, "GetPerpetualMetadata should decode the builder DEX's collateral token")
	} else {
		assert.NotZero(t, result.CollateralToken, "GetPerpetualMetadata should return flx's USDH collateral token")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualMetadata(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualMetadata should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetPerpetualMetadata(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualMetadata should reject a null response")
}

func TestGetPerpetualMetadataAndAssetContexts(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualMetadataAndAssetContexts(t.Context(), "")
	require.NoError(t, err, "GetPerpetualMetadataAndAssetContexts must not error")
	if mockTests {
		exp := &PerpetualMetadataAndAssetContextsResponse{
			Metadata: testDefaultMetadata(),
			AssetContexts: []PerpetualAssetContext{
				{Funding: 0.0000125, OpenInterest: 39310.43958, PreviousDayPrice: 86248, DayNotionalVolume: 1820651593.0427696705, Premium: -0.0002691175, OraclePrice: 85836.1, MarkPrice: 85810.5, MidPrice: 85812.5, ImpactPrices: []types.Number{85812, 85813}, DayBaseVolume: 21219.11844},
				{Funding: 0.0000125, OpenInterest: 1184127.1915999996, PreviousDayPrice: 2722, DayNotionalVolume: 700331577.8523004055, Premium: -0.0004310234, OraclePrice: 2714.47, MarkPrice: 2713.2, MidPrice: 2713.25, ImpactPrices: []types.Number{2713.2, 2713.3}, DayBaseVolume: 258520.2890000001},
				{PreviousDayPrice: 0.37, OraclePrice: 0.37, MarkPrice: 0.37},
				{Funding: 0.0000125, OpenInterest: 1250, PreviousDayPrice: 0.0451, DayNotionalVolume: 1213.92, Premium: 0.0003, OraclePrice: 0.0452, MarkPrice: 0.0452, MidPrice: 0.04515, ImpactPrices: []types.Number{0.0451, 0.0452}, DayBaseVolume: 26900},
			},
		}
		assert.Equal(t, exp, result, "GetPerpetualMetadataAndAssetContexts should decode the metadata and every context")
	} else {
		assert.Len(t, result.AssetContexts, len(result.Metadata.Universe), "GetPerpetualMetadataAndAssetContexts should return a context per market")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualMetadataAndAssetContexts(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualMetadataAndAssetContexts should return the request failure")
	_, err = newResponseServerExchange(t, `[{"universe":[{"name":"BTC"}]},[]]`).GetPerpetualMetadataAndAssetContexts(t.Context(), "")
	assert.ErrorIs(t, err, errUnexpectedResponseLength, "GetPerpetualMetadataAndAssetContexts should reject contexts that do not match the universe")
	_, err = newResponseServerExchange(t, `[{"universe":[]}]`).GetPerpetualMetadataAndAssetContexts(t.Context(), "")
	assert.ErrorIs(t, err, errInvalidTupleLength, "GetPerpetualMetadataAndAssetContexts should reject a response without contexts")
	_, err = newResponseServerExchange(t, `null`).GetPerpetualMetadataAndAssetContexts(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualMetadataAndAssetContexts should reject a null response")
}

func TestGetClearinghouseState(t *testing.T) {
	t.Parallel()
	_, err := e.GetClearinghouseState(t.Context(), "invalid", "")
	require.ErrorIs(t, err, errInvalidAddress, "GetClearinghouseState must reject an invalid address")

	result, err := e.GetClearinghouseState(t.Context(), testTraderAddress, "")
	require.NoError(t, err, "GetClearinghouseState must not error")
	if mockTests {
		exp := &ClearinghouseStateResponse{
			MarginSummary:              MarginSummary{AccountValue: 13109.482328, TotalNotionalPosition: 100.02765, TotalRawUSD: 13009.454678, TotalMarginUsed: 4.967826},
			CrossMarginSummary:         MarginSummary{AccountValue: 13104.514502, TotalNotionalPosition: 50, TotalRawUSD: 13054.514502, TotalMarginUsed: 2.5},
			CrossMaintenanceMarginUsed: 1.25,
			Withdrawable:               13104.514502,
			AssetPositions: []AssetPosition{{
				Type: "oneWay",
				Position: Position{
					Coin:              "BTC",
					SignedSize:        0.57588,
					Leverage:          Leverage{Type: "isolated", Value: 20, RawUSD: -95.059824},
					EntryPrice:        85650.4,
					PositionValue:     49492.27896,
					UnrealisedPNL:     167.918393,
					ReturnOnEquity:    0.068087408,
					LiquidationPrice:  81234.5,
					MarginUsed:        2474.613948,
					MaxLeverage:       40,
					CumulativeFunding: CumulativeFunding{AllTime: -612543.888709, SinceOpen: 453.922467, SinceChange: 12.5},
				},
			}},
			Time: milli(1791275833327),
		}
		assert.Equal(t, exp, result, "GetClearinghouseState should decode every field")
	} else {
		assert.NotZero(t, result.Time, "GetClearinghouseState should return the state time")
	}

	_, err = newUnavailableServerExchange(t).GetClearinghouseState(t.Context(), testTraderAddress, "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetClearinghouseState should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetClearinghouseState(t.Context(), testTraderAddress, "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetClearinghouseState should reject a null response")
}

func TestGetUserFunding(t *testing.T) {
	t.Parallel()
	const zeroHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
	start, end := getTime()
	_, err := e.GetUserFunding(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserFunding must reject a nil request")
	_, err = e.GetUserFunding(t.Context(), &UserFundingRequest{User: "invalid", StartTime: start, EndTime: end})
	require.ErrorIs(t, err, errInvalidAddress, "GetUserFunding must reject an invalid address")
	_, err = e.GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: time.Unix(0, 0)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserFunding must reject a start at the epoch, which would drop the required start time")
	_, err = e.GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserFunding must reject a reversed range")

	result, err := e.GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: start, EndTime: end})
	require.NoError(t, err, "GetUserFunding must not error")
	if mockTests {
		exp := []UserFundingUpdateResponse{
			{Time: milli(1791255600066), Hash: zeroHash, Delta: UserFundingDelta{Type: "funding", Coin: "BTC", USDC: -0.583273, SignedSize: 0.54241, FundingRate: 0.0000125}},
			{Time: milli(1791259200066), Hash: zeroHash, Delta: UserFundingDelta{Type: "funding", Coin: "xyz:XYZ100", USDC: -0.044393, SignedSize: 0.2284, FundingRate: 0.00000625}},
		}
		assert.Equal(t, exp, result, "GetUserFunding should decode every hourly payment")
	}
	require.NotEmpty(t, result, "GetUserFunding must return the vault's hourly payments")

	// Both bounds are inclusive, so a range of one instant returns every payment made at that instant
	payment := result[0]
	result, err = e.GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: payment.Time.Time(), EndTime: payment.Time.Time()})
	require.NoError(t, err, "GetUserFunding must not error for a range of one instant")
	assert.Contains(t, result, payment, "GetUserFunding should return the payments made at the instant")

	// Hyperliquid compacts history older than one to two weeks into one aggregate per market and day
	dailyStart := start.Add(-30 * 24 * time.Hour)
	result, err = e.GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: dailyStart, EndTime: dailyStart.Add(24 * time.Hour)})
	require.NoError(t, err, "GetUserFunding must not error for compacted history")
	if mockTests {
		exp := []UserFundingUpdateResponse{{
			Time:  milli(1788739200000),
			Hash:  "0xa166e3fa63c25663024b03f2e0da011a00307e4017465df020210d3d432e7cb8",
			Delta: UserFundingDelta{Type: "funding", Coin: "0G", USDC: -0.081144, SignedSize: 1395.2083333333, FundingRate: 0.0000125, NumberOfSamples: 24},
		}}
		assert.Equal(t, exp, result, "GetUserFunding should decode a daily aggregate's sample count")
	} else if len(result) > 0 {
		assert.NotZero(t, result[0].Delta.NumberOfSamples, "GetUserFunding should return daily aggregates for compacted history")
	}

	_, err = newUnavailableServerExchange(t).GetUserFunding(t.Context(), &UserFundingRequest{User: testTraderAddress, StartTime: start, EndTime: end})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserFunding should return the request failure")
}

func TestGetUserNonFundingLedgerUpdates(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := e.GetUserNonFundingLedgerUpdates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserNonFundingLedgerUpdates must reject a nil request")
	_, err = e.GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: "invalid", StartTime: start, EndTime: end})
	require.ErrorIs(t, err, errInvalidAddress, "GetUserNonFundingLedgerUpdates must reject an invalid address")
	_, err = e.GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: testVaultAddress, StartTime: time.Unix(0, 0)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserNonFundingLedgerUpdates must reject a start at the epoch, which would drop the required start time")
	_, err = e.GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: testVaultAddress, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserNonFundingLedgerUpdates must reject a reversed range")

	result, err := e.GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: testVaultAddress, StartTime: start, EndTime: end})
	require.NoError(t, err, "GetUserNonFundingLedgerUpdates must not error")
	if mockTests {
		exp := []UserLedgerUpdateResponse{
			{Time: milli(1791252100000), Hash: "0x6e5030418dccaafb6fc90445f45dbe0201de002728cfc9cd1218db944cc084e6", Delta: UserLedgerDelta{Type: "accountClassTransfer", USDC: 50099.8, ToPerpetual: true}},
			{Time: milli(1791253000000), Hash: "0x31d548837f8c7791334f0445f45eaa0201be00691a8f9663d59df3d63e80517b", Delta: UserLedgerDelta{
				Type:           "send",
				User:           "0x0000000000000000000000000000000000000012",
				Destination:    "0x0000000000000000000000000000000000000013",
				SourceDEX:      "spot",
				DestinationDEX: "xyz",
				Token:          currency.USDC,
				Amount:         99959.73,
				USDCValue:      99959.73,
				Fee:            0.1,
				NativeTokenFee: 0.0001,
				Nonce:          1761521253203,
				FeeToken:       currency.USDC,
			}},
			{Time: milli(1791254000000), Hash: "0xa0237867410998a5a19d0445f45f610201c3004cdc0cb77743ec23ba000d7290", Delta: UserLedgerDelta{
				Type:            "vaultWithdraw",
				Vault:           testVaultAddress,
				User:            "0x0000000000000000000000000000000000000012",
				RequestedUSD:    72509.49,
				Commission:      1.5,
				ClosingCost:     0.25,
				Basis:           72590.327117,
				NetWithdrawnUSD: 72507.74,
			}},
			{Time: milli(1791255000000), Hash: "0xc660efebf2d06244c7da0445f45f6e0202e700d18dd381166a299b3eb1d43c2f", Delta: UserLedgerDelta{
				Type:                       "liquidation",
				LiquidatedNotionalPosition: 356.80015,
				AccountValue:               2.956494,
				LeverageType:               "Cross",
				LiquidatedPositions:        []LiquidatedPosition{{Coin: "ATOM", SignedSize: 29.21}},
			}},
			{Time: milli(1791256000000), Hash: "0xfcd4d536f70782ba8be46a1337181a4588374ad928e5b89295866003abe9af45", Delta: UserLedgerDelta{Type: "cStakingTransfer", Token: currency.HYPE, Amount: 1, IsDeposit: true}},
			{Time: milli(1791257000000), Hash: "0x1a7d1a8328117777839597254792990d8f4baa2765123f45494f0435788716bf", Delta: UserLedgerDelta{Type: "borrowLend", Token: currency.USDC, Amount: 250.5, Operation: "repay", InterestAmount: 0.0821}},
			{Time: milli(1791258000000), Hash: "0xd11f5d7637eb7bc58484871adf443302d39841ab7a6696b4d324328b0c3b525f", Delta: UserLedgerDelta{Type: "activateDexAbstraction", Token: currency.USDC, Amount: 1, DEX: "xyz"}},
			{Time: milli(1791259000000), Hash: "0x1d01c444ebadbd182e1f765c9031114ce6fe709f58bd421eb0ea757cbaf91059", Delta: UserLedgerDelta{Type: "deployGasAuction", Token: currency.HYPE, Amount: 500}},
		}
		assert.Equal(t, exp, result, "GetUserNonFundingLedgerUpdates should decode every delta field")
	}

	// Both bounds are inclusive, so a range of one instant returns every update made at that instant; the vault may have
	// had no updates in the live window
	if len(result) > 0 {
		update := result[0]
		result, err = e.GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: testVaultAddress, StartTime: update.Time.Time(), EndTime: update.Time.Time()})
		require.NoError(t, err, "GetUserNonFundingLedgerUpdates must not error for a range of one instant")
		assert.Contains(t, result, update, "GetUserNonFundingLedgerUpdates should return the updates made at the instant")
	}

	_, err = newUnavailableServerExchange(t).GetUserNonFundingLedgerUpdates(t.Context(), &UserNonFundingLedgerUpdatesRequest{User: testVaultAddress, StartTime: start, EndTime: end})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserNonFundingLedgerUpdates should return the request failure")
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := e.GetFundingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingHistory must reject a nil request")
	_, err = e.GetFundingHistory(t.Context(), &FundingHistoryRequest{StartTime: start, EndTime: end})
	require.ErrorIs(t, err, errCoinRequired, "GetFundingHistory must require a coin")
	_, err = e.GetFundingHistory(t.Context(), &FundingHistoryRequest{Coin: "BTC", StartTime: time.Unix(0, 0)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingHistory must reject a start at the epoch, which would drop the required start time")
	_, err = e.GetFundingHistory(t.Context(), &FundingHistoryRequest{Coin: "BTC", StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingHistory must reject a reversed range")

	result, err := e.GetFundingHistory(t.Context(), &FundingHistoryRequest{Coin: "BTC", StartTime: start, EndTime: end})
	require.NoError(t, err, "GetFundingHistory must not error")
	if mockTests {
		exp := []FundingRateHistoryResponse{
			{Coin: "BTC", FundingRate: 0.0000125, Premium: -0.000226673, Time: milli(1791255600057)},
			{Coin: "BTC", FundingRate: -0.0000032, Premium: -0.000525601, Time: milli(1791259200041)},
		}
		assert.Equal(t, exp, result, "GetFundingHistory should decode every field")
	}
	require.NotEmpty(t, result, "GetFundingHistory must return hourly rates")

	// Both bounds are inclusive, so a range of one instant returns the rate at that instant
	rate := result[0]
	result, err = e.GetFundingHistory(t.Context(), &FundingHistoryRequest{Coin: "BTC", StartTime: rate.Time.Time(), EndTime: rate.Time.Time()})
	require.NoError(t, err, "GetFundingHistory must not error for a range of one instant")
	assert.Equal(t, []FundingRateHistoryResponse{rate}, result, "GetFundingHistory should return the one rate at the instant")

	_, err = newUnavailableServerExchange(t).GetFundingHistory(t.Context(), &FundingHistoryRequest{Coin: "BTC", StartTime: start, EndTime: end})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingHistory should return the request failure")
}

func TestGetPredictedFundings(t *testing.T) {
	t.Parallel()
	result, err := e.GetPredictedFundings(t.Context())
	require.NoError(t, err, "GetPredictedFundings must not error")
	if mockTests {
		exp := []PredictedFundingResponse{
			{Coin: "BTC", Venues: []PredictedFundingVenue{
				{Venue: "BinPerp", Funding: &PredictedFundingRate{FundingRate: -0.00004049, NextFundingTime: milli(1791302400000), FundingIntervalHours: 8}},
				{Venue: "HlPerp", Funding: &PredictedFundingRate{FundingRate: 0.0000125, NextFundingTime: milli(1791277200000), FundingIntervalHours: 1}},
				{Venue: "BybitPerp", Funding: &PredictedFundingRate{FundingRate: 0.00005069, NextFundingTime: milli(1791302400000), FundingIntervalHours: 8}},
			}},
			{Coin: "AI", Venues: []PredictedFundingVenue{
				{Venue: "BinPerp", Funding: &PredictedFundingRate{NextFundingTime: milli(1791288000000)}},
				{Venue: "HlPerp", Funding: &PredictedFundingRate{NextFundingTime: milli(1791277200000), FundingIntervalHours: 1}},
				{Venue: "BybitPerp"},
			}},
		}
		assert.Equal(t, exp, result, "GetPredictedFundings should decode every venue, including one that does not list the coin")
	} else {
		assert.NotEmpty(t, result, "GetPredictedFundings should return predictions")
	}

	_, err = newUnavailableServerExchange(t).GetPredictedFundings(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPredictedFundings should return the request failure")
}

func TestGetPerpetualsAtOpenInterestCap(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualsAtOpenInterestCap(t.Context(), "")
	require.NoError(t, err, "GetPerpetualsAtOpenInterestCap must not error")
	if mockTests {
		assert.Equal(t, []string{"CANTO", "FTM", "JELLY", "LOOM", "RLB", "VINE", "ZEREBRO"}, result, "GetPerpetualsAtOpenInterestCap should return every capped coin")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualsAtOpenInterestCap(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualsAtOpenInterestCap should return the request failure")
}

func TestGetPerpetualDeployAuctionStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualDeployAuctionStatus(t.Context())
	require.NoError(t, err, "GetPerpetualDeployAuctionStatus must not error")
	if mockTests {
		exp := &GasAuctionResponse{StartTime: types.Time(time.Unix(1791180000, 0)), DurationSeconds: 111600, StartGas: 1098.12115292, EndGas: 571.8561851}
		assert.Equal(t, exp, result, "GetPerpetualDeployAuctionStatus should decode a won auction")
	} else {
		assert.NotZero(t, result.DurationSeconds, "GetPerpetualDeployAuctionStatus should return the auction's duration")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualDeployAuctionStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualDeployAuctionStatus should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetPerpetualDeployAuctionStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualDeployAuctionStatus should reject a null response")
}

func TestGetActiveAssetData(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveAssetData(t.Context(), "invalid", "BTC")
	require.ErrorIs(t, err, errInvalidAddress, "GetActiveAssetData must reject an invalid address")
	_, err = e.GetActiveAssetData(t.Context(), testVaultAddress, "")
	require.ErrorIs(t, err, errCoinRequired, "GetActiveAssetData must require a coin")

	result, err := e.GetActiveAssetData(t.Context(), testVaultAddress, "BTC")
	require.NoError(t, err, "GetActiveAssetData must not error")
	if mockTests {
		exp := &ActiveAssetDataResponse{
			User:             testVaultAddress,
			Coin:             "BTC",
			Leverage:         Leverage{Type: "isolated", Value: 20, RawUSD: -95.059824},
			MaxTradeSizes:    [2]types.Number{9754.65405, 9754.65405},
			AvailableToTrade: [2]types.Number{41851855.4974759966, 41851855.4974759966},
			MarkPrice:        85809,
		}
		assert.Equal(t, exp, result, "GetActiveAssetData should decode every field")
	} else {
		assert.Equal(t, "BTC", result.Coin, "GetActiveAssetData should return the requested coin")
	}

	_, err = newUnavailableServerExchange(t).GetActiveAssetData(t.Context(), testVaultAddress, "BTC")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetActiveAssetData should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetActiveAssetData(t.Context(), testVaultAddress, "BTC")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetActiveAssetData should reject a null response")
}

func TestGetPerpetualDEXLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetPerpetualDEXLimits(t.Context(), " ")
	require.ErrorIs(t, err, errDEXRequired, "GetPerpetualDEXLimits must require a builder DEX")

	result, err := e.GetPerpetualDEXLimits(t.Context(), "xyz")
	require.NoError(t, err, "GetPerpetualDEXLimits must not error")
	if mockTests {
		exp := &PerpetualDEXLimitsResponse{
			TotalOpenInterestCap:            10000000000,
			OpenInterestSizeCapPerPerpetual: 20000000000,
			MaxTransferNotional:             3000000000,
			CoinToOpenInterestCap:           []AssetValue{{Coin: "xyz:AAOI", Value: 25000000}, {Coin: "xyz:AAPL", Value: 200000000}},
		}
		assert.Equal(t, exp, result, "GetPerpetualDEXLimits should decode every limit")
	} else {
		assert.NotZero(t, result.TotalOpenInterestCap, "GetPerpetualDEXLimits should return the DEX's open interest cap")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualDEXLimits(t.Context(), "xyz")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualDEXLimits should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetPerpetualDEXLimits(t.Context(), "xyz")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualDEXLimits should reject a null response")
}

func TestGetPerpetualDEXStatus(t *testing.T) {
	t.Parallel()
	for dex, exp := range map[string]types.Number{"": 3126723996.6663417816, "xyz": 1057417474.976467967} {
		result, err := e.GetPerpetualDEXStatus(t.Context(), dex)
		require.NoErrorf(t, err, "GetPerpetualDEXStatus must not error for DEX %q", dex)
		if mockTests {
			assert.Equalf(t, &PerpetualDEXStatusResponse{TotalNetDeposit: exp}, result, "GetPerpetualDEXStatus should decode DEX %q's net deposits", dex)
		} else {
			assert.Positivef(t, result.TotalNetDeposit.Float64(), "GetPerpetualDEXStatus should return DEX %q's net deposits", dex)
		}
	}

	_, err := newUnavailableServerExchange(t).GetPerpetualDEXStatus(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualDEXStatus should return the request failure")

	var body map[string]any
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"totalNetDeposit":"1.5"}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetPerpetualDEXStatus(t.Context(), "")
	require.NoError(t, err, "GetPerpetualDEXStatus must not error for the default DEX")
	assert.Equal(t, map[string]any{"type": "perpDexStatus", "dex": ""}, body, "GetPerpetualDEXStatus should send an empty DEX, which Hyperliquid requires")

	_, err = newResponseServerExchange(t, `null`).GetPerpetualDEXStatus(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualDEXStatus should reject a null response")
}

func TestGetAllPerpetualMetadata(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllPerpetualMetadata(t.Context())
	require.NoError(t, err, "GetAllPerpetualMetadata must not error")
	if mockTests {
		exp := []PerpetualMetadataResponse{
			{
				Universe:     []PerpetualAssetMetadata{{Name: "BTC", SizeDecimals: 5, MaxLeverage: 40, MarginTableID: 56}},
				MarginTables: []MarginTableEntry{{ID: 51, Table: MarginTable{Description: "tiered 10x", MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 10}, {LowerBound: 3000000, MaxLeverage: 5}}}}},
			},
			{
				Universe: []PerpetualAssetMetadata{{
					Name:                     "xyz:URANIUM",
					SizeDecimals:             3,
					MaxLeverage:              10,
					MarginTableID:            10,
					IsDelisted:               true,
					MarginMode:               "strictIsolated",
					GrowthMode:               "enabled",
					LastGrowthModeChangeTime: ZonelessTime(time.Date(2025, 11, 23, 17, 21, 40, 390706535, time.UTC)),
					DeployerFeeScale:         1,
					LastFeeScaleChangeTime:   ZonelessTime(time.Date(2026, 1, 26, 16, 47, 8, 614356922, time.UTC)),
				}},
				MarginTables: []MarginTableEntry{{ID: 10, Table: MarginTable{MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 10}}}}},
			},
			{
				Universe: []PerpetualAssetMetadata{{
					Name:                   "flx:TSLA",
					SizeDecimals:           2,
					MaxLeverage:            10,
					MarginTableID:          10,
					GrowthMode:             "enabled",
					DeployerFeeScale:       1,
					LastFeeScaleChangeTime: ZonelessTime(time.Date(2025, 11, 24, 21, 19, 25, 980742012, time.UTC)),
				}},
				MarginTables:    []MarginTableEntry{{ID: 10, Table: MarginTable{MarginTiers: []MarginTier{{LowerBound: 0, MaxLeverage: 10}}}}},
				CollateralToken: 360,
			},
		}
		assert.Equal(t, exp, result, "GetAllPerpetualMetadata should decode every DEX's metadata")
	} else {
		require.NotEmpty(t, result, "GetAllPerpetualMetadata must return the default DEX")
		assert.Equal(t, "BTC", result[0].Universe[0].Name, "GetAllPerpetualMetadata should list the default DEX first")
	}

	_, err = newUnavailableServerExchange(t).GetAllPerpetualMetadata(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAllPerpetualMetadata should return the request failure")
	_, err = newResponseServerExchange(t, `[]`).GetAllPerpetualMetadata(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAllPerpetualMetadata should reject a response without the default DEX")
}

func TestGetPerpetualAnnotation(t *testing.T) {
	t.Parallel()
	_, err := e.GetPerpetualAnnotation(t.Context(), " ")
	require.ErrorIs(t, err, errCoinRequired, "GetPerpetualAnnotation must require a coin")

	result, err := e.GetPerpetualAnnotation(t.Context(), "io:OAI")
	require.NoError(t, err, "GetPerpetualAnnotation must not error")
	if mockTests {
		exp := &PerpetualAnnotationResponse{Category: "preipo", Description: "A perpetual future contract tracking the implied valuation of OpenAI.", DisplayName: "OPENAI", Keywords: []string{"openai", "oai"}}
		assert.Equal(t, exp, result, "GetPerpetualAnnotation should decode every field")
	} else {
		assert.NotEmpty(t, result.Category, "GetPerpetualAnnotation should return the market's category")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualAnnotation(t.Context(), "io:OAI")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualAnnotation should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetPerpetualAnnotation(t.Context(), "BTC")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPerpetualAnnotation should report a default DEX market's null annotation")
}

func TestGetPerpetualCategories(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualCategories(t.Context())
	require.NoError(t, err, "GetPerpetualCategories must not error")
	if mockTests {
		assert.Equal(t, []PerpetualCategoryResponse{{Coin: "flx:BTC", Category: "crypto"}, {Coin: "io:OAI", Category: "preipo"}}, result, "GetPerpetualCategories should decode every category")
	} else {
		assert.NotEmpty(t, result, "GetPerpetualCategories should return categories")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualCategories(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualCategories should return the request failure")
}

func TestGetPerpetualConciseAnnotations(t *testing.T) {
	t.Parallel()
	result, err := e.GetPerpetualConciseAnnotations(t.Context())
	require.NoError(t, err, "GetPerpetualConciseAnnotations must not error")
	if mockTests {
		exp := []PerpetualConciseAnnotationResponse{
			{Coin: "io:OAI", Annotation: PerpetualAnnotationResponse{Category: "preipo", DisplayName: "OPENAI", Keywords: []string{"openai", "oai"}}},
			{Coin: "mkts:USTECH", Annotation: PerpetualAnnotationResponse{Category: "indices", DisplayName: "QQQ", Keywords: []string{"USTECH", "NQ"}}},
		}
		assert.Equal(t, exp, result, "GetPerpetualConciseAnnotations should decode every annotation")
	} else {
		assert.NotEmpty(t, result, "GetPerpetualConciseAnnotations should return annotations")
	}

	_, err = newUnavailableServerExchange(t).GetPerpetualConciseAnnotations(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPerpetualConciseAnnotations should return the request failure")
}

func TestGetUserStarState(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserStarState(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserStarState must reject an invalid address")

	result, err := e.GetUserStarState(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetUserStarState must not error")
	if mockTests {
		exp := &UserStarStateResponse{DEXToState: []DEXStarState{
			{DEX: "ignp", State: &StarState{IsReduceOnly: true, IsBackstopLiquidatorDepositAllowed: true}},
			{DEX: "nimb"},
		}}
		assert.Equal(t, exp, result, "GetUserStarState should decode every DEX's state, including a removed approval")
	}

	_, err = newUnavailableServerExchange(t).GetUserStarState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserStarState should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetUserStarState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserStarState should reject a null response")
}

func TestTimeRangeMilli(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(mockStartTime)
	end := time.UnixMilli(mockEndTime)
	for _, tc := range []struct {
		start, end       time.Time
		expStart, expEnd uint64
		err              error
	}{
		{start: start, end: end, expStart: mockStartTime, expEnd: mockEndTime},
		{start: start, end: start, expStart: mockStartTime, expEnd: mockStartTime},
		{start: start, expStart: mockStartTime},
		{err: common.ErrDateUnset},
		{start: time.Unix(0, 0), err: common.ErrDateUnset},
		{start: time.Unix(-5, 0), end: end, err: common.ErrDateUnset},
		{start: time.Now().Add(time.Hour), err: common.ErrStartAfterTimeNow},
		{start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		startTime, endTime, err := timeRangeMilli(tc.start, tc.end)
		require.ErrorIsf(t, err, tc.err, "timeRangeMilli must return the expected error for %s to %s", tc.start, tc.end)
		assert.Equalf(t, tc.expStart, startTime, "timeRangeMilli should convert the start of %s to %s", tc.start, tc.end)
		assert.Equalf(t, tc.expEnd, endTime, "timeRangeMilli should convert or omit the end of %s to %s", tc.start, tc.end)
	}
}

func TestAssetValueUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var value AssetValue
	require.NoError(t, json.Unmarshal([]byte(`["xyz:XYZ100","25000000.0"]`), &value), "Unmarshal must not error")
	assert.Equal(t, AssetValue{Coin: "xyz:XYZ100", Value: 25000000}, value, "AssetValue should decode its coin and value")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["xyz:XYZ100"]`), &value), errInvalidTupleLength, "AssetValue should reject a short tuple")
}

func TestSubDeployerUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var deployer SubDeployer
	require.NoError(t, json.Unmarshal([]byte(`["setOracle",["0x0000000000000000000000000000000000000004"]]`), &deployer), "Unmarshal must not error")
	assert.Equal(t, SubDeployer{Action: "setOracle", Addresses: []string{"0x0000000000000000000000000000000000000004"}}, deployer, "SubDeployer should decode its action and addresses")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["setOracle"]`), &deployer), errInvalidTupleLength, "SubDeployer should reject a short tuple")

	// Testnet's registry sends HIP-3* delegations as one-key objects
	require.NoError(t, json.Unmarshal([]byte(`[{"hip3Star":"modifyApproval"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]]`), &deployer), "Unmarshal must not error for an action sent as an object")
	assert.Equal(t, SubDeployer{Action: "modifyApproval", ActionSet: "hip3Star", Addresses: []string{"0x73b69db46d41c6b721fb5cb9ce9f29193140950e"}}, deployer, "SubDeployer should decode an action sent as an object with its key")
	require.NoError(t, json.Unmarshal([]byte(`["setOracle",["0x0000000000000000000000000000000000000004"]]`), &deployer), "Unmarshal must not error")
	assert.Equal(t, SubDeployer{Action: "setOracle", Addresses: []string{"0x0000000000000000000000000000000000000004"}}, deployer, "SubDeployer should clear the key of an earlier action sent as an object")
	for _, action := range []string{`{}`, `{"hip3Star":"setOracle","other":"cancel"}`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(`[`+action+`,["0x0000000000000000000000000000000000000004"]]`), &deployer), errMalformedResponse, "SubDeployer should reject the action %s", action)
	}
	for _, action := range []string{`{"hip3Star":1}`, `["setOracle"]`} {
		assert.Errorf(t, json.Unmarshal([]byte(`[`+action+`,["0x0000000000000000000000000000000000000004"]]`), &deployer), "SubDeployer should reject the action %s", action)
	}

	// An entry of testnet's live perpDexs reply, 26 October 2026, which failed to decode while actions were only names
	var registry []*PerpetualDEXResponse
	require.NoError(t, json.Unmarshal([]byte(`[null,{"name":"fcmt","fullName":"FCM control test","deployer":"0x19e0af27ea5db8f8a5d060dadbf7d8b4c2348a87","oracleUpdater":null,"feeRecipient":null,"assetToStreamingOiCap":[],"subDeployers":[[{"hip3Star":"modifyApproval"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"modifyBackstopLiquidatorApproval"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"setReduceOnly"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"cancel"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"cancelAll"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"order"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]],[{"hip3Star":"sendAsset"},["0x73b69db46d41c6b721fb5cb9ce9f29193140950e"]]],"assetToFundingMultiplier":[],"assetToFundingInterestRate":[],"assetToFundingClamp":[]}]`), &registry), "Unmarshal must decode a registry holding actions sent as objects")
	require.Len(t, registry, 2, "The registry must hold the default DEX and fcmt")
	require.NotNil(t, registry[1], "fcmt's entry must decode")
	assert.Equal(t, "fcmt", registry[1].Name, "PerpetualDEXResponse should decode the DEX's name")
	require.Len(t, registry[1].SubDeployers, 7, "fcmt must list seven delegated actions")
	assert.Equal(t, SubDeployer{Action: "cancelAll", ActionSet: "hip3Star", Addresses: []string{"0x73b69db46d41c6b721fb5cb9ce9f29193140950e"}}, registry[1].SubDeployers[4], "PerpetualDEXResponse should decode each delegated action")
}

func TestMarginTableEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry MarginTableEntry
	require.NoError(t, json.Unmarshal([]byte(`[51,{"description":"tiered 10x","marginTiers":[{"lowerBound":"3000000.0","maxLeverage":5}]}]`), &entry), "Unmarshal must not error")
	exp := MarginTableEntry{ID: 51, Table: MarginTable{Description: "tiered 10x", MarginTiers: []MarginTier{{LowerBound: 3000000, MaxLeverage: 5}}}}
	assert.Equal(t, exp, entry, "MarginTableEntry should decode its ID and table")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[51]`), &entry), errInvalidTupleLength, "MarginTableEntry should reject a short tuple")
}

func TestPerpetualMetadataAndAssetContextsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var response PerpetualMetadataAndAssetContextsResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"universe":[{"name":"BTC","szDecimals":5}],"collateralToken":360},[{"markPx":"85810.5"}]]`), &response), "Unmarshal must not error")
	exp := PerpetualMetadataAndAssetContextsResponse{
		Metadata:      PerpetualMetadataResponse{Universe: []PerpetualAssetMetadata{{Name: "BTC", SizeDecimals: 5}}, CollateralToken: 360},
		AssetContexts: []PerpetualAssetContext{{MarkPrice: 85810.5}},
	}
	assert.Equal(t, exp, response, "PerpetualMetadataAndAssetContextsResponse should decode its metadata and contexts")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[{},[],[]]`), &response), errInvalidTupleLength, "PerpetualMetadataAndAssetContextsResponse should reject a long tuple")
}

func TestPredictedFundingUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var funding PredictedFundingResponse
	require.NoError(t, json.Unmarshal([]byte(`["BTC",[["HlPerp",{"fundingRate":"0.0000125","nextFundingTime":1791277200000,"fundingIntervalHours":1}],["BybitPerp",null]]]`), &funding), "Unmarshal must not error")
	exp := PredictedFundingResponse{Coin: "BTC", Venues: []PredictedFundingVenue{
		{Venue: "HlPerp", Funding: &PredictedFundingRate{FundingRate: 0.0000125, NextFundingTime: milli(1791277200000), FundingIntervalHours: 1}},
		{Venue: "BybitPerp"},
	}}
	assert.Equal(t, exp, funding, "PredictedFunding should decode its coin and venues")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["BTC"]`), &funding), errInvalidTupleLength, "PredictedFunding should reject a short tuple")
}

func TestPredictedFundingVenueUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var venue PredictedFundingVenue
	require.NoError(t, json.Unmarshal([]byte(`["BybitPerp",null]`), &venue), "Unmarshal must not error for a venue that does not list the coin")
	assert.Equal(t, PredictedFundingVenue{Venue: "BybitPerp"}, venue, "PredictedFundingVenue should decode a null funding as nil")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["BybitPerp",null,null]`), &venue), errInvalidTupleLength, "PredictedFundingVenue should reject a long tuple")
}

func TestPerpetualCategoryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var category PerpetualCategoryResponse
	require.NoError(t, json.Unmarshal([]byte(`["flx:BTC","crypto"]`), &category), "Unmarshal must not error")
	assert.Equal(t, PerpetualCategoryResponse{Coin: "flx:BTC", Category: "crypto"}, category, "PerpetualCategory should decode its coin and category")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["flx:BTC"]`), &category), errInvalidTupleLength, "PerpetualCategory should reject a short tuple")
}

func TestPerpetualConciseAnnotationUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var annotation PerpetualConciseAnnotationResponse
	require.NoError(t, json.Unmarshal([]byte(`["io:OAI",{"category":"preipo","keywords":["oai"]}]`), &annotation), "Unmarshal must not error")
	exp := PerpetualConciseAnnotationResponse{Coin: "io:OAI", Annotation: PerpetualAnnotationResponse{Category: "preipo", Keywords: []string{"oai"}}}
	assert.Equal(t, exp, annotation, "PerpetualConciseAnnotation should decode its coin and annotation")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["io:OAI"]`), &annotation), errInvalidTupleLength, "PerpetualConciseAnnotation should reject a short tuple")
}

func TestDEXStarStateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var approved DEXStarState
	require.NoError(t, json.Unmarshal([]byte(`["ignp",{"isReduceOnly":false,"isBackstopLiquidatorDepositAllowed":true}]`), &approved), "Unmarshal must not error")
	assert.Equal(t, DEXStarState{DEX: "ignp", State: &StarState{IsBackstopLiquidatorDepositAllowed: true}}, approved, "DEXStarState should decode its DEX and state")
	var removed DEXStarState
	require.NoError(t, json.Unmarshal([]byte(`["nimb",null]`), &removed), "Unmarshal must not error for a removed approval")
	assert.Equal(t, DEXStarState{DEX: "nimb"}, removed, "DEXStarState should decode a removed approval as a nil state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["nimb"]`), &removed), errInvalidTupleLength, "DEXStarState should reject a short tuple")
}
