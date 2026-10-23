package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const (
	futuresTestMasterAccountUID = "ba598ca1-65c1-4f48-927d-0e2b647d627a"
	futuresTestSubaccountUID    = "7f5c528e-2285-45f0-95f5-83d53d4bfcd2"
)

// futuresTestAccountUIDs returns the master account and subaccount UIDs a test acts on: the recorded ones in mock
// tests, or the live account's and its first subaccount's, skipping when it has none
func futuresTestAccountUIDs(t *testing.T) (masterAccountUID, subaccountUID string) {
	t.Helper()
	if mockTests {
		return futuresTestMasterAccountUID, futuresTestSubaccountUID
	}
	result, err := e.GetFuturesSubaccounts(t.Context())
	require.NoError(t, err, "GetFuturesSubaccounts must not error")
	if len(result.Subaccounts) == 0 {
		t.Skip("the account has no subaccounts")
	}
	return result.MasterAccountUID, result.Subaccounts[0].AccountUID
}

func TestGetFuturesSubaccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesSubaccounts(t.Context())
	require.NoError(t, err, "GetFuturesSubaccounts must not error")
	if mockTests {
		exp := &FuturesSubaccountsResponse{
			MasterAccountUID: futuresTestMasterAccountUID,
			Subaccounts: []FuturesSubaccount{
				{
					AccountUID: futuresTestSubaccountUID,
					Email:      "john.doe@example.com",
					FullName:   "John Doe",
					HoldingAccounts: []FuturesSubaccountHoldingBalance{
						{Currency: currency.NewCode("xrp"), Amount: 13662.85078},
						{Currency: currency.NewCode("eth"), Amount: 3.0000485057},
					},
					FuturesAccounts: []FuturesSubaccountMarginBalance{
						{Name: "f-xrp:usd", AvailableMargin: 16187.33210488726},
						{Name: "f-xbt:usd", AvailableMargin: -0.0009056832839642471},
					},
					FlexAccount: FuturesSubaccountFlexAccount{
						Currencies: []FuturesSubaccountFlexCurrency{
							{Currency: currency.NewCode("eth"), Quantity: 0.5, Value: 1646.575, Collateral: 1543.91104875, Available: 0.49999966035931903},
							{Currency: currency.NewCode("usd"), Quantity: 250, Value: 250.01, Collateral: 249.99, Available: 212.5},
						},
						InitialMargin:           37.5,
						InitialMarginWithOrders: 52.75,
						MaintenanceMargin:       18.75,
						BalanceValue:            1896.58,
						PortfolioValue:          1904.08,
						CollateralValue:         1793.91,
						UnrealisedPNL:           6.25,
						UnrealisedFunding:       1.25,
						TotalUnrealised:         7.5,
						TotalUnrealisedAsMargin: 7.13,
						AvailableMargin:         1763.54,
						MarginEquity:            1801.04,
						PortfolioMarginBreakdown: &FuturesPortfolioMarginBreakdown{
							TotalCrossAssetNettedMarketRisk:          28.4,
							TotalMarketRisk:                          31.9,
							TotalScenarioPNLs:                        []float64{-31.9, -12.6, 18.3},
							TotalAbsoluteOptionPositionDeltaNotional: 1520.75,
							NetPortfolioDelta:                        0.0215,
							TotalPremium:                             42.5,
							IsBuyOnly:                                true,
							FuturesMaintenanceMargin:                 9.6,
						},
					},
				},
			},
			ServerTime: time.Date(2022, 3, 31, 20, 38, 53, 677000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesSubaccounts should decode every field")
		return
	}
	assert.NotEmpty(t, result.MasterAccountUID, "GetFuturesSubaccounts should return the master account's UID")
}

func TestGetFuturesSubaccountTradingStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesSubaccountTradingStatus(t.Context(), "")
	require.ErrorIs(t, err, errFuturesSubaccountUIDEmpty, "GetFuturesSubaccountTradingStatus must reject an empty subaccount UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, subaccountUID := futuresTestAccountUIDs(t)
	result, err := e.GetFuturesSubaccountTradingStatus(t.Context(), subaccountUID)
	require.NoError(t, err, "GetFuturesSubaccountTradingStatus must not error")
	if mockTests {
		assert.Equal(t, &FuturesSubaccountTradingStatusResponse{TradingEnabled: true}, result, "GetFuturesSubaccountTradingStatus should decode every field")
	}
}

func TestUpdateFuturesSubaccountTradingStatus(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesSubaccountTradingStatus(t.Context(), "", true)
	require.ErrorIs(t, err, errFuturesSubaccountUIDEmpty, "UpdateFuturesSubaccountTradingStatus must reject an empty subaccount UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, subaccountUID := futuresTestAccountUIDs(t)
	// Disabling first leaves a live subaccount able to trade
	for _, tradingEnabled := range []bool{false, true} {
		result, err := e.UpdateFuturesSubaccountTradingStatus(t.Context(), subaccountUID, tradingEnabled)
		require.NoErrorf(t, err, "UpdateFuturesSubaccountTradingStatus must not error setting %t", tradingEnabled)
		assert.Equalf(t, &FuturesSubaccountTradingStatusResponse{TradingEnabled: tradingEnabled}, result, "UpdateFuturesSubaccountTradingStatus should return the status set to %t", tradingEnabled)
	}
}
