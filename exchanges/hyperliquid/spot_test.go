package hyperliquid

import (
	"math"
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

// testSpotMetadata is the spot metadata the mock responses return
func testSpotMetadata() SpotMetadataResponse {
	return SpotMetadataResponse{
		Universe: []SpotAssetMetadata{
			{Tokens: []uint64{1, 0}, Name: "PURR/USDC", IsCanonical: true},
			{Tokens: []uint64{150, 0}, Name: "@107", Index: 107},
			{Tokens: []uint64{360, 0}, Name: "@230", Index: 230},
		},
		Tokens: []SpotTokenMetadata{
			{Name: currency.USDC, SizeDecimals: 8, WeiDecimals: 8, TokenID: "0x6d1e7cde53ba9467b783cb7c530ce054", IsCanonical: true, EVMContract: &EVMContract{Address: "0x6b9e773128f453f5c2c60935ee2de2cbc5390a24", EVMExtraWeiDecimals: -2}},
			{Name: currency.NewCode("PURR"), WeiDecimals: 5, Index: 1, TokenID: "0xc1fb593aeffbeb02f85e0308e9956a90", IsCanonical: true, EVMContract: &EVMContract{Address: "0x9b498c3c8a0b8cd8ba1d9851d40d186f1872b44e", EVMExtraWeiDecimals: 13}},
			{Name: currency.HYPE, SizeDecimals: 2, WeiDecimals: 8, Index: 150, TokenID: "0x0d01dc56dcaaca66ad901c959b4011ec", FullName: "Hyperliquid"},
			{Name: currency.NewCode("RAT"), SizeDecimals: 2, WeiDecimals: 8, Index: 253, TokenID: "0xd7f58d3c744fbc0100f7d86f3e283d11", EVMContract: &EVMContract{Address: "0xfdc42f844f3aa19f8fc0b2d67d3276cf1efdc4ca", EVMExtraWeiDecimals: 10}, FullName: "Rat Race Combat", DeployerTradingFeeShare: 1, DeployerLabel: "rrc"},
			{Name: currency.NewCode("USDH"), SizeDecimals: 2, WeiDecimals: 8, Index: 360, TokenID: "0x54e00a5988577cb0b0c9ab0cb6ef7f4b", EVMContract: &EVMContract{Address: "0x111111a1a0667d36bd57c0a9f569b98057111111", EVMExtraWeiDecimals: -2}, FullName: "USDH", DeployerTradingFeeShare: 1},
		},
	}
}

func TestGetSpotMetadata(t *testing.T) {
	t.Parallel()
	result, err := e.GetSpotMetadata(t.Context())
	require.NoError(t, err, "GetSpotMetadata must not error")
	if mockTests {
		exp := testSpotMetadata()
		assert.Equal(t, &exp, result, "GetSpotMetadata should decode every field")
	} else {
		assert.True(t, result.Tokens[0].Name.Equal(currency.USDC), "GetSpotMetadata should list USDC as token 0")
	}

	_, err = newUnavailableServerExchange(t).GetSpotMetadata(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotMetadata should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetSpotMetadata(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotMetadata should reject a null response")
}

func TestGetSpotMetadataAndAssetContexts(t *testing.T) {
	t.Parallel()
	result, err := e.GetSpotMetadataAndAssetContexts(t.Context())
	require.NoError(t, err, "GetSpotMetadataAndAssetContexts must not error")
	if mockTests {
		exp := &SpotMetadataAndAssetContextsResponse{
			Metadata: testSpotMetadata(),
			AssetContexts: []SpotAssetContext{
				{PreviousDayPrice: 0.16577, DayNotionalVolume: 2361343.4740200015, MarkPrice: 0.15049, MidPrice: 0.150495, CirculatingSupply: 594786002.8013399839, Coin: "PURR/USDC", TotalSupply: 594786014.1078599691, DayBaseVolume: 14727160},
				{PreviousDayPrice: 93.102, DayNotionalVolume: 63321924.1126599833, MarkPrice: 93.146, MidPrice: 93.1465, CirculatingSupply: 298462929.368832469, Coin: "@107", TotalSupply: 998883571.7642624378, DayBaseVolume: 677650.0300000001},
				{PreviousDayPrice: 1.0001, DayNotionalVolume: 1501, MarkPrice: 1, CirculatingSupply: 50000000, Coin: "@230", TotalSupply: 50000000, DayBaseVolume: 1501},
				{PreviousDayPrice: 0.48, DayNotionalVolume: 1029.6, MarkPrice: 0.52, MidPrice: 0.515, CirculatingSupply: 1000, Coin: "#10", TotalSupply: 1000, DayBaseVolume: 2024.5},
			},
		}
		assert.Equal(t, exp, result, "GetSpotMetadataAndAssetContexts should decode the metadata and every context, including coins outside the universe")
	} else {
		assert.NotEmpty(t, result.AssetContexts, "GetSpotMetadataAndAssetContexts should return contexts")
	}

	_, err = newUnavailableServerExchange(t).GetSpotMetadataAndAssetContexts(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotMetadataAndAssetContexts should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetSpotMetadataAndAssetContexts(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotMetadataAndAssetContexts should reject a null response")
}

func TestGetSpotClearinghouseState(t *testing.T) {
	t.Parallel()
	const (
		zeroAddress = "0x0000000000000000000000000000000000000000"
		// portfolioMarginAddress is a placeholder, whose mock spot state copies a portfolio margin account's
		portfolioMarginAddress = "0x000000000000000000000000000000000000001d"
	)
	_, err := e.GetSpotClearinghouseState(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetSpotClearinghouseState must reject an invalid address")

	result, err := e.GetSpotClearinghouseState(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetSpotClearinghouseState must not error")
	if mockTests {
		exp := &SpotClearinghouseStateResponse{Balances: []SpotBalance{
			{Coin: currency.USDC, Total: 14.625485},
			{Coin: currency.HYPE, Token: 150, Total: 816.67225318, Hold: 2.5, EntryNotional: 19838.88581631},
		}}
		assert.Equal(t, exp, result, "GetSpotClearinghouseState should decode the trader's balances")
	}

	result, err = e.GetSpotClearinghouseState(t.Context(), testVaultAddress)
	require.NoError(t, err, "GetSpotClearinghouseState must not error for the vault")
	if mockTests {
		exp := &SpotClearinghouseStateResponse{
			Balances: []SpotBalance{
				{Coin: currency.USDC, Total: 100508.11272667, Hold: 49999.89, EntryNotional: 1},
				{Coin: currency.NewCode("USDH"), Token: 360},
			},
			TokenToAvailableAfterMaintenance: []TokenAmount{{Amount: 100508.11272667}, {Token: 360}},
		}
		assert.Equal(t, exp, result, "GetSpotClearinghouseState should decode a unified account's available amounts after maintenance")
	}

	result, err = e.GetSpotClearinghouseState(t.Context(), portfolioMarginAddress)
	require.NoError(t, err, "GetSpotClearinghouseState must not error for the portfolio margin account")
	if mockTests {
		exp := &SpotClearinghouseStateResponse{
			Balances: []SpotBalance{
				{Coin: currency.USDC, Total: 1784747.4809165401, Hold: -3204581.7878122497, SpotHold: 585798.4557817401},
				{Coin: currency.HYPE, Token: 150, Total: 68072.40171969},
				{Coin: currency.NewCode("USDT0"), Token: 268, Hold: -659983.31632743},
			},
			TokenToAvailableAfterMaintenance: []TokenAmount{{Amount: 6303976.4137682496}, {Token: 150, Amount: 68072.40171969}, {Token: 268, Amount: 769085.77021597}},
		}
		assert.Equal(t, exp, result, "GetSpotClearinghouseState should decode a portfolio margin account's spot holds and negative holds")
	}

	result, err = e.GetSpotClearinghouseState(t.Context(), zeroAddress)
	require.NoError(t, err, "GetSpotClearinghouseState must not error for the zero address")
	if mockTests {
		exp := &SpotClearinghouseStateResponse{Balances: []SpotBalance{
			{Coin: currency.USDC, Total: 1019.97983005},
			{Coin: currency.NewCode("PURR"), Token: 1, Total: 6.52278, EntryNotional: 1.43548571},
			{Coin: currency.NewCode("HFUN"), Token: 2, Total: 0.01074413, EntryNotional: 1.17424847},
		}}
		assert.Equal(t, exp, result, "GetSpotClearinghouseState should decode the zero address's balances")
	} else {
		assert.NotEmpty(t, result.Balances, "GetSpotClearinghouseState should return the tokens burnt to the zero address")
	}

	_, err = newUnavailableServerExchange(t).GetSpotClearinghouseState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotClearinghouseState should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetSpotClearinghouseState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotClearinghouseState should reject a null response")
}

func TestGetSpotDeployState(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotDeployState(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetSpotDeployState must reject an invalid address")

	result, err := e.GetSpotDeployState(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetSpotDeployState must not error")
	if mockTests {
		exp := &SpotDeployStateResponse{
			States: []SpotDeployState{{
				Token:                        354,
				Specification:                SpotTokenSpecification{Name: currency.NewCode("RED"), SizeDecimals: 2, WeiDecimals: 8},
				FullName:                     "RedStone",
				DeployerTradingFeeShare:      1,
				Spots:                        []uint64{717},
				MaxSupply:                    184467440737.0955200195,
				HyperliquidityGenesisBalance: 120000,
				TotalGenesisBalanceInWei:     math.MaxUint64,
				UserGenesisBalances:          []UserBalance{{User: "0x2000000000000000000000000000000000000162", Balance: 184467440737.0955200195}},
				ExistingTokenGenesisBalances: []ExistingTokenBalance{{Token: 1, Balance: 300000}},
				BlacklistUsers:               []string{"0x0000000000000000000000000000000000000019"},
			}},
			GasAuction: GasAuctionResponse{StartTime: types.Time(time.Unix(1733929200, 0)), DurationSeconds: 111600, StartGas: 181305.90046, CurrentGas: 181300, EndGas: 181291.247358},
		}
		assert.Equal(t, exp, result, "GetSpotDeployState should decode every field, including a genesis balance of the maximum uint64 wei")
	} else {
		assert.NotZero(t, result.GasAuction.DurationSeconds, "GetSpotDeployState should return the spot deploy gas auction")
	}

	_, err = newUnavailableServerExchange(t).GetSpotDeployState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotDeployState should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetSpotDeployState(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotDeployState should reject a null response")
}

func TestGetSpotPairDeployAuctionStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSpotPairDeployAuctionStatus(t.Context())
	require.NoError(t, err, "GetSpotPairDeployAuctionStatus must not error")
	if mockTests {
		exp := &GasAuctionResponse{StartTime: types.Time(time.Unix(1791180000, 0)), DurationSeconds: 111600, StartGas: 500, CurrentGas: 500}
		assert.Equal(t, exp, result, "GetSpotPairDeployAuctionStatus should decode a running auction")
	} else {
		assert.NotZero(t, result.DurationSeconds, "GetSpotPairDeployAuctionStatus should return the auction's duration")
	}

	_, err = newUnavailableServerExchange(t).GetSpotPairDeployAuctionStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSpotPairDeployAuctionStatus should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetSpotPairDeployAuctionStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSpotPairDeployAuctionStatus should reject a null response")
}

func TestGetTokenDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetTokenDetails(t.Context(), "HFUN")
	require.ErrorIs(t, err, errInvalidTokenID, "GetTokenDetails must reject a token name")

	result, err := e.GetTokenDetails(t.Context(), "0xBAF265EF389DA684513D98D68EDF4EAE")
	require.NoError(t, err, "GetTokenDetails must not error for an upper case token ID")
	if mockTests {
		exp := &TokenDetailsResponse{
			Name:              currency.NewCode("HFUN"),
			MaxSupply:         1000000,
			TotalSupply:       995724.63958273,
			CirculatingSupply: 995724.58875657,
			SizeDecimals:      2,
			WeiDecimals:       8,
			MidPrice:          12.583,
			MarkPrice:         12.573,
			PreviousDayPrice:  12.516,
			Genesis: &TokenGenesis{
				UserBalances:          []UserBalance{{User: "0x0000000000000000000000000000000000000015", Balance: 300000}, {User: "0xffffffffffffffffffffffffffffffffffffffff", Balance: 400000}},
				ExistingTokenBalances: []ExistingTokenBalance{{Token: 1, Balance: 300000}},
				BlacklistUsers:        []string{"0x000000000000000000000000000000000000001a"},
			},
			Deployer:                   "0x0000000000000000000000000000000000000015",
			DeployGas:                  500,
			DeployTime:                 ZonelessTime(time.Date(2024, 5, 16, 11, 27, 33, 468000000, time.UTC)),
			SeededUSDC:                 1500,
			NonCirculatingUserBalances: []UserBalance{{User: "0x0000000000000000000000000000000000000000", Balance: 0.01074413}, {User: "0x000000000000000000000000000000000000dead", Balance: 0.04008203}},
			FutureEmissions:            410984598.9547457695,
		}
		assert.Equal(t, exp, result, "GetTokenDetails should decode every field")
	} else {
		assert.True(t, result.Name.Equal(currency.NewCode("HFUN")), "GetTokenDetails should return the requested token")
	}

	result, err = e.GetTokenDetails(t.Context(), "0x6d1e7cde53ba9467b783cb7c530ce054")
	require.NoError(t, err, "GetTokenDetails must not error for USDC")
	if mockTests {
		exp := &TokenDetailsResponse{
			Name:                       currency.USDC,
			MaxSupply:                  1000000000000,
			TotalSupply:                396751657.3657590151,
			CirculatingSupply:          396751657.3657590151,
			SizeDecimals:               8,
			MidPrice:                   1,
			MarkPrice:                  1,
			PreviousDayPrice:           1,
			NonCirculatingUserBalances: []UserBalance{},
		}
		assert.Equal(t, exp, result, "GetTokenDetails should decode a protocol token without genesis or deployment details")
	} else {
		assert.Nil(t, result.Genesis, "GetTokenDetails should return USDC without a genesis distribution")
	}

	_, err = newUnavailableServerExchange(t).GetTokenDetails(t.Context(), "0x6d1e7cde53ba9467b783cb7c530ce054")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTokenDetails should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetTokenDetails(t.Context(), "0x6d1e7cde53ba9467b783cb7c530ce054")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTokenDetails should reject a null response")
}

func TestValidateTokenID(t *testing.T) {
	t.Parallel()
	for _, tokenID := range []string{"0x6d1e7cde53ba9467b783cb7c530ce054", "0X6D1E7CDE53BA9467B783CB7C530CE054"} {
		assert.NoErrorf(t, validateTokenID(tokenID), "validateTokenID should accept %s", tokenID)
	}
	for _, tokenID := range []string{"", "USDC", "0x6d1e7cde53ba9467b783cb7c530ce05", "0x6d1e7cde53ba9467b783cb7c530ce0544", "1x6d1e7cde53ba9467b783cb7c530ce054", "0x6d1e7cde53ba9467b783cb7c530ce05g"} {
		assert.ErrorIsf(t, validateTokenID(tokenID), errInvalidTokenID, "validateTokenID should reject %q", tokenID)
	}
}

func TestGetOutcomeMetadata(t *testing.T) {
	t.Parallel()
	result, err := e.GetOutcomeMetadata(t.Context())
	require.NoError(t, err, "GetOutcomeMetadata must not error")
	if mockTests {
		sides := []OutcomeSideSpecification{{Name: "Yes"}, {Name: "No"}}
		exp := &OutcomeMetadataResponse{
			Outcomes: []OutcomeSpecification{
				{Outcome: 1472, Name: "template fallback", Description: "other", SideSpecifications: sides, QuoteToken: currency.USDC, Venue: "out", DeployerFeeScale: 5},
				{Outcome: 9015, Name: "Recurring", Description: "class:priceBinary|underlying:BTC|expiry:20261007-0600|targetPrice:85501|period:1d", SideSpecifications: sides, QuoteToken: currency.USDC},
			},
			Questions: []OutcomeQuestion{{
				Question:             198,
				Name:                 "template:sportsTournamentWinner",
				Description:          "competition:English Premier League|officialSource:English Premier League|resolutionDeadline:20270605-1200|season:2026/2027|sport:football/soccer",
				FallbackOutcome:      1472,
				NamedOutcomes:        []uint64{1473, 1474, 1475, 1476, 1477, 1478},
				SettledNamedOutcomes: []uint64{1477},
			}},
			Deployers: []OutcomeDeployer{{
				Deployer:     "0x0000000000000000000000000000000000000017",
				Venue:        "out",
				SubDeployers: []SubDeployer{{Action: "settleOutcome", Addresses: []string{"0x0000000000000000000000000000000000000018"}}},
			}},
			FeeScale: 1,
		}
		assert.Equal(t, exp, result, "GetOutcomeMetadata should decode every field")
	} else {
		assert.Positive(t, result.FeeScale.Float64(), "GetOutcomeMetadata should return the fee scale")
	}

	_, err = newUnavailableServerExchange(t).GetOutcomeMetadata(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOutcomeMetadata should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetOutcomeMetadata(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOutcomeMetadata should reject a null response")
}

func TestGetSettledOutcome(t *testing.T) {
	t.Parallel()
	result, err := e.GetSettledOutcome(t.Context(), 0)
	require.NoError(t, err, "GetSettledOutcome must not error for outcome 0")
	if mockTests {
		exp := &SettledOutcomeResponse{
			Specification: OutcomeSpecification{
				Name:               "Recurring",
				Description:        "class:priceBinary|underlying:BTC|expiry:20260503-0600|targetPrice:78213|period:1d",
				SideSpecifications: []OutcomeSideSpecification{{Name: "Yes"}, {Name: "No"}},
				QuoteToken:         currency.NewCode("USDH"),
			},
			Details: "price:78212.4",
		}
		assert.Equal(t, exp, result, "GetSettledOutcome should decode a settled protocol outcome")
	} else {
		assert.Zero(t, result.Specification.Outcome, "GetSettledOutcome should return outcome 0")
		assert.NotEmpty(t, result.Details, "GetSettledOutcome should return outcome 0's settlement details")
	}

	result, err = e.GetSettledOutcome(t.Context(), 9092)
	require.NoError(t, err, "GetSettledOutcome must not error for a deployer outcome")
	if mockTests {
		exp := &SettledOutcomeResponse{
			Specification: OutcomeSpecification{
				Outcome:            9092,
				Name:               "template:binaryPrice",
				Description:        "perp:BTC|priceDescription:the Hyperliquid BTC perp trade|seconds:90|threshold:85558|time:20261006-0900",
				SideSpecifications: []OutcomeSideSpecification{{Name: "template:Yes"}, {Name: "template:No"}},
				QuoteToken:         currency.USDC,
				Venue:              "skew",
				DeployerFeeScale:   1,
			},
			SettleFraction: 1,
			Details:        "template",
			Question: &SettledOutcomeQuestion{
				Question:    QuestionState{Question: 390, State: QuestionStateSettled},
				Name:        "Recurring",
				Description: "class:priceBucket|underlying:BTC|expiry:20261006-0600|priceThresholds:84034,87464|period:1d",
			},
		}
		assert.Equal(t, exp, result, "GetSettledOutcome should decode every field")
	} else {
		assert.Equal(t, uint64(9092), result.Specification.Outcome, "GetSettledOutcome should return the requested outcome")
	}

	_, err = newUnavailableServerExchange(t).GetSettledOutcome(t.Context(), 9092)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSettledOutcome should return the request failure")

	var body map[string]any
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		_, err := w.Write([]byte(`null`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetSettledOutcome(t.Context(), 0)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSettledOutcome should report an unsettled outcome's null response")
	assert.Equal(t, map[string]any{"type": "settledOutcome", "outcome": float64(0)}, body, "GetSettledOutcome should send outcome 0, which Hyperliquid requires")
}

func TestGetOutcomeDeployerLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetOutcomeDeployerLimits(t.Context(), " ")
	require.ErrorIs(t, err, errVenueRequired, "GetOutcomeDeployerLimits must require a venue")

	result, err := e.GetOutcomeDeployerLimits(t.Context(), "out")
	require.NoError(t, err, "GetOutcomeDeployerLimits must not error")
	if mockTests {
		assert.Equal(t, &OutcomeDeployerLimitsResponse{NumberOfDailyOutcomesRemaining: 851, NumberOfActiveOutcomesRemaining: 27}, result, "GetOutcomeDeployerLimits should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOutcomeDeployerLimits(t.Context(), "out")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOutcomeDeployerLimits should return the request failure")
	_, err = newResponseServerExchange(t, `null`).GetOutcomeDeployerLimits(t.Context(), "out")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOutcomeDeployerLimits should reject a null response")
}

func TestGetOutcomeTemplates(t *testing.T) {
	t.Parallel()
	result, err := e.GetOutcomeTemplates(t.Context())
	require.NoError(t, err, "GetOutcomeTemplates must not error")
	if mockTests {
		exp := []OutcomeTemplateResponse{
			{
				ID:          "aiModelHeadToHead",
				Role:        OutcomeTemplateRole{StandaloneOutcome: &StandaloneOutcomeRole{SideNames: []string{"{shortNameA}", "{shortNameB}"}}},
				Name:        "{shortNameA} v {shortNameB}: {leaderboard} on {dateTime}",
				Description: "This market compares {modelA} ({shortNameA}) and {modelB} ({shortNameB}) on {dateTime} using {leaderboard} at {leaderboardUrl}. metadata=category:technology|subCategory:N/A|semanticRestriction:This market must compare two specific AI language models on a public performance leaderboard.",
				Keywords: []OutcomeTemplateKeyword{
					{Keyword: "benchmarkSettings", Hint: "string"},
					{Keyword: "comparisonDirection", Hint: "string"},
					{Keyword: "dateTime", Hint: "dateTime"},
					{Keyword: "leaderboard", Hint: "string"},
					{Keyword: "leaderboardUrl", Hint: "string"},
					{Keyword: "metric", Hint: "string"},
					{Keyword: "modelA", Hint: "string"},
					{Keyword: "modelB", Hint: "string"},
					{Keyword: "shortNameA", Hint: "shortString"},
					{Keyword: "shortNameB", Hint: "shortString"},
					{Keyword: "tiebreakers", Hint: "string"},
				},
			},
			{
				ID:          "binaryPrice",
				Role:        OutcomeTemplateRole{StandaloneOutcome: &StandaloneOutcomeRole{SideNames: []string{"Yes", "No"}}},
				Name:        "{perp} above {threshold} at {time}?",
				Description: "The market resolves to Yes if the {perp} price is above {threshold} at {time}, and otherwise resolves to No. metadata=category:price|subCategory:N/A|semanticRestriction:This market must refer to the price of a Hyperliquid perp.",
				Keywords: []OutcomeTemplateKeyword{
					{Keyword: "perp", Hint: "hlPerp"},
					{Keyword: "priceDescription", Hint: "string"},
					{Keyword: "seconds", Hint: "uInt"},
					{Keyword: "threshold", Hint: "uDecimal"},
					{Keyword: "time", Hint: "dateTime"},
				},
			},
			{
				ID:          "sportsTournamentParticipant",
				Role:        OutcomeTemplateRole{QuestionOutcome: &QuestionOutcomeRole{Parent: "sportsTournamentWinner"}},
				Name:        "{participant}",
				Description: "This outcome resolves to Yes if {participant} is named the sole winner of the Tournament under the parent question's rules.",
				Keywords:    []OutcomeTemplateKeyword{{Keyword: "participant", Hint: "string"}},
			},
			{
				ID:          "sportsTournamentWinner",
				Role:        OutcomeTemplateRole{Question: true},
				Name:        "{season} {competition} winner",
				Description: `This {sport} market has one named outcome per listed participant in the {season} {competition} (the "Tournament") plus the fallback. metadata=category:sports|subCategory:{sport}|semanticRestriction:This market must refer to the overall winner of a sports tournament.`,
				Keywords: []OutcomeTemplateKeyword{
					{Keyword: "competition", Hint: "string"},
					{Keyword: "officialSource", Hint: "string"},
					{Keyword: "resolutionDeadline", Hint: "dateTime"},
					{Keyword: "season", Hint: "string"},
					{Keyword: "sport", Hint: "string"},
				},
			},
		}
		assert.Equal(t, exp, result, "GetOutcomeTemplates should decode every role and keyword")
	} else {
		assert.NotEmpty(t, result, "GetOutcomeTemplates should return templates")
	}

	_, err = newUnavailableServerExchange(t).GetOutcomeTemplates(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOutcomeTemplates should return the request failure")
	_, err = newResponseServerExchange(t, `[{"id":"template","role":"unknown"}]`).GetOutcomeTemplates(t.Context())
	assert.ErrorIs(t, err, errInvalidOutcomeTemplateRole, "GetOutcomeTemplates should reject a template with an unknown role")
}

func TestSpotMetadataAndAssetContextsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var response SpotMetadataAndAssetContextsResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"tokens":[{"name":"USDC"}]},[{"coin":"#10","markPx":"0.52"}]]`), &response), "Unmarshal must not error")
	exp := SpotMetadataAndAssetContextsResponse{
		Metadata:      SpotMetadataResponse{Tokens: []SpotTokenMetadata{{Name: currency.USDC}}},
		AssetContexts: []SpotAssetContext{{Coin: "#10", MarkPrice: 0.52}},
	}
	assert.Equal(t, exp, response, "SpotMetadataAndAssetContextsResponse should decode its metadata and contexts")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[{}]`), &response), errInvalidTupleLength, "SpotMetadataAndAssetContextsResponse should reject a short tuple")
}

func TestTokenAmountUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var amount TokenAmount
	require.NoError(t, json.Unmarshal([]byte(`[360,"100508.11272667"]`), &amount), "Unmarshal must not error")
	assert.Equal(t, TokenAmount{Token: 360, Amount: 100508.11272667}, amount, "TokenAmount should decode its token and amount")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[360]`), &amount), errInvalidTupleLength, "TokenAmount should reject a short tuple")
}

func TestUserBalanceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var balance UserBalance
	require.NoError(t, json.Unmarshal([]byte(`["0x000000000000000000000000000000000000dead","0.04008203"]`), &balance), "Unmarshal must not error")
	assert.Equal(t, UserBalance{User: "0x000000000000000000000000000000000000dead", Balance: 0.04008203}, balance, "UserBalance should decode its user and balance")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["0x000000000000000000000000000000000000dead"]`), &balance), errInvalidTupleLength, "UserBalance should reject a short tuple")
}

func TestExistingTokenBalanceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var balance ExistingTokenBalance
	require.NoError(t, json.Unmarshal([]byte(`[1,"300000.0"]`), &balance), "Unmarshal must not error")
	assert.Equal(t, ExistingTokenBalance{Token: 1, Balance: 300000}, balance, "ExistingTokenBalance should decode its token and balance")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1]`), &balance), errInvalidTupleLength, "ExistingTokenBalance should reject a short tuple")
}

func TestQuestionStateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]QuestionState{
		`{"active":823}`:  {Question: 823, State: QuestionStateActive},
		`{"settled":390}`: {Question: 390, State: QuestionStateSettled},
	} {
		var state QuestionState
		require.NoErrorf(t, json.Unmarshal([]byte(data), &state), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, state, "QuestionState should decode %s", data)
	}
	for _, data := range []string{`{}`, `{"active":1,"settled":2}`, `[]`, `{"active":"1"}`} {
		var state QuestionState
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &state), errInvalidQuestionState, "QuestionState should reject %s", data)
	}
}

func TestOutcomeTemplateRoleUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]OutcomeTemplateRole{
		`"question"`: {Question: true},
		`{"standaloneOutcome":{"sideNames":["{shortNameA}","{shortNameB}"]}}`: {StandaloneOutcome: &StandaloneOutcomeRole{SideNames: []string{"{shortNameA}", "{shortNameB}"}}},
		`{"questionOutcome":{"parent":"sportsTournamentWinner"}}`:             {QuestionOutcome: &QuestionOutcomeRole{Parent: "sportsTournamentWinner"}},
	} {
		var role OutcomeTemplateRole
		require.NoErrorf(t, json.Unmarshal([]byte(data), &role), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, role, "OutcomeTemplateRole should decode %s", data)
	}
	for _, data := range []string{
		`"standaloneOutcome"`,
		`null`,
		`1`,
		`[]`,
		`{}`,
		`{"unknown":{}}`,
		`{"standaloneOutcome":null}`,
		`{"questionOutcome":"sportsTournamentWinner"}`,
		`{"standaloneOutcome":{"sideNames":["Yes","No"]},"questionOutcome":{"parent":"sportsTournamentWinner"}}`,
		`{"standaloneOutcome":{"sideNames":["Yes","No"]},"unknown":{}}`,
	} {
		var role OutcomeTemplateRole
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &role), errInvalidOutcomeTemplateRole, "OutcomeTemplateRole should reject %s", data)
	}
}

func TestOutcomeTemplateKeywordUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var keyword OutcomeTemplateKeyword
	require.NoError(t, json.Unmarshal([]byte(`["perp","hlPerp"]`), &keyword), "Unmarshal must not error")
	assert.Equal(t, OutcomeTemplateKeyword{Keyword: "perp", Hint: "hlPerp"}, keyword, "OutcomeTemplateKeyword should decode its keyword and hint")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["perp"]`), &keyword), errInvalidTupleLength, "OutcomeTemplateKeyword should reject a short tuple")
}
