package bybit

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCryptoLoanBorrowableCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetCryptoLoanBorrowableCoins(t.Context(), "VIP5", currency.ETH)
	require.NoError(t, err, "GetCryptoLoanBorrowableCoins must not error")
	if mockTests {
		exp := &CryptoLoanBorrowableCoinsResponse{
			List: []CryptoLoanBorrowableCoin{
				{
					Currency:                       currency.ETH,
					FixedBorrowable:                true,
					FixedBorrowingAccuracy:         8,
					FlexibleBorrowable:             true,
					FlexibleBorrowingAccuracy:      4,
					MaxBorrowingAmount:             5000,
					MinFixedBorrowingAmount:        0.25,
					MinFlexibleBorrowingAmount:     0.016,
					VIPLevel:                       "VIP5",
					FlexibleAnnualisedInterestRate: 0.0194098569084,
					AnnualisedInterestRate7Day:     0.027,
					AnnualisedInterestRate14Day:    0.027,
					AnnualisedInterestRate30Day:    0.027,
					AnnualisedInterestRate60Day:    0.027,
					AnnualisedInterestRate90Day:    0.04,
					AnnualisedInterestRate180Day:   0.054,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCryptoLoanBorrowableCoins should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetCryptoLoanBorrowableCoins should return borrowable coins")
}

func TestGetCryptoLoanCollateralCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetCryptoLoanCollateralCoins(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCryptoLoanCollateralCoins must not error")
	if mockTests {
		exp := &CryptoLoanCollateralCoinsResponse{
			CollateralRatioConfigList: []CryptoLoanCollateralRatioConfig{
				{
					CollateralRatioList: []CryptoLoanCollateralRatio{
						{
							CollateralRatio: 0.8,
							MaxValue:        10000,
						},
						{
							CollateralRatio: 0.7,
							MaxValue:        20000,
							MinValue:        10000,
						},
					},
					Currencies: CryptoLoanCurrencies{currency.ATOM, currency.AAVE, currency.BTC, currency.BOB},
				},
			},
			CurrencyLiquidationList: []CryptoLoanCurrencyLiquidation{
				{
					Currency:         currency.BTC,
					LiquidationOrder: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCryptoLoanCollateralCoins should decode every field")
		return
	}
	assert.NotEmpty(t, result.CollateralRatioConfigList, "GetCryptoLoanCollateralCoins should return collateral ratio tiers")
}

func TestCryptoLoanCurrenciesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got CryptoLoanCurrencies
	require.NoError(t, json.Unmarshal([]byte(`"ATOM,AAVE,BTC,BOB"`), &got), "Unmarshal must not error")
	exp := CryptoLoanCurrencies{currency.ATOM, currency.AAVE, currency.BTC, currency.BOB}
	assert.Equal(t, exp, got, "Unmarshal should decode every coin")

	require.NoError(t, json.Unmarshal([]byte(`""`), &got), "Unmarshal must not error on an empty string")
	assert.Nil(t, got, "Unmarshal should decode an empty string as no coins")

	assert.Error(t, json.Unmarshal([]byte(`1`), &got), "Unmarshal should reject a number")
}

func TestGetCryptoLoanMaxCollateralReductionAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetCryptoLoanMaxCollateralReductionAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCryptoLoanMaxCollateralReductionAmount must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCryptoLoanMaxCollateralReductionAmount(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCryptoLoanMaxCollateralReductionAmount must not error")
	if mockTests {
		exp := &CryptoLoanMaxCollateralReductionAmountResponse{
			MaxCollateralAmount: 0.08585184,
		}
		assert.Equal(t, exp, result, "GetCryptoLoanMaxCollateralReductionAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCryptoLoanMaxCollateralReductionAmount should return a result")
}

func TestAdjustCryptoLoanCollateralAmount(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustCryptoLoanCollateralAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AdjustCryptoLoanCollateralAmount must reject a nil request")
	_, err = e.AdjustCryptoLoanCollateralAmount(t.Context(), &AdjustCryptoLoanCollateralAmountRequest{Amount: 0.08, Direction: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AdjustCryptoLoanCollateralAmount must reject an empty coin")
	_, err = e.AdjustCryptoLoanCollateralAmount(t.Context(), &AdjustCryptoLoanCollateralAmountRequest{Currency: currency.BTC, Direction: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AdjustCryptoLoanCollateralAmount must reject a zero amount")
	_, err = e.AdjustCryptoLoanCollateralAmount(t.Context(), &AdjustCryptoLoanCollateralAmountRequest{Currency: currency.BTC, Amount: 0.08})
	require.ErrorIs(t, err, errCollateralAdjustmentDirectionRequired, "AdjustCryptoLoanCollateralAmount must reject an empty direction")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AdjustCryptoLoanCollateralAmount(t.Context(), &AdjustCryptoLoanCollateralAmountRequest{Currency: currency.BTC, Amount: 0.08, Direction: "1"})
	require.NoError(t, err, "AdjustCryptoLoanCollateralAmount must not error")
	if mockTests {
		exp := &AdjustCryptoLoanCollateralAmountResponse{
			AdjustID: 27511,
		}
		assert.Equal(t, exp, result, "AdjustCryptoLoanCollateralAmount should decode every field")
		return
	}
	assert.NotZero(t, result.AdjustID, "AdjustCryptoLoanCollateralAmount should return an adjustment ID")
}

func TestGetCryptoLoanCollateralAdjustmentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCryptoLoanCollateralAdjustmentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCryptoLoanCollateralAdjustmentHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCryptoLoanCollateralAdjustmentHistory(t.Context(), &CryptoLoanCollateralAdjustmentHistoryRequest{
		AdjustID:           "27511",
		CollateralCurrency: currency.BTC,
		Limit:              2,
		Cursor:             "27512",
	})
	require.NoError(t, err, "GetCryptoLoanCollateralAdjustmentHistory must not error")
	if mockTests {
		exp := &CryptoLoanCollateralAdjustmentHistoryResponse{
			List: []CryptoLoanCollateralAdjustment{
				{
					CollateralCurrency: currency.BTC,
					Amount:             0.08,
					AdjustID:           27511,
					AdjustTime:         types.Time(time.UnixMilli(1752627997907)),
					PreLTV:             0.524602,
					AfterLTV:           0.813743,
					Direction:          1,
					Status:             1,
				},
			},
			NextPageCursor: "27510",
		}
		assert.Equal(t, exp, result, "GetCryptoLoanCollateralAdjustmentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCryptoLoanCollateralAdjustmentHistory should return a result")
}

func TestGetCryptoLoanPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCryptoLoanPosition(t.Context())
	require.NoError(t, err, "GetCryptoLoanPosition must not error")
	if mockTests {
		exp := &CryptoLoanPositionResponse{
			BorrowList: []CryptoLoanBorrowPosition{
				{
					FlexibleHourlyInterestRate: 0.0000001361462,
					FlexibleTotalDebt:          0.08800022,
					FlexibleTotalDebtUSD:       9355.37,
					LoanCurrency:               currency.BTC,
				},
				{
					FixedTotalDebt:             0.1,
					FixedTotalDebtUSD:          282.8,
					FlexibleHourlyInterestRate: 0.00000188498892,
					LoanCurrency:               currency.ETH,
				},
			},
			CollateralList: []CryptoLoanCoinAmount{
				{
					Amount:    0.12,
					AmountUSD: 9930.11,
					Currency:  currency.BTC,
				},
				{
					Amount:    2,
					AmountUSD: 4524.81,
					Currency:  currency.ETH,
				},
				{
					Amount:    4002.12,
					AmountUSD: 3201.69,
					Currency:  currency.USDT,
				},
				{
					Amount:    1000,
					AmountUSD: 724.8,
					Currency:  currency.USDC,
				},
			},
			LTV: 0.524344,
			SupplyList: []CryptoLoanCoinAmount{
				{
					Amount:    800.1304109589041,
					AmountUSD: 800.13,
					Currency:  currency.USDT,
				},
			},
			TotalCollateral:       18381.41,
			TotalDebt:             9638.17,
			TotalSupply:           800.13,
			CollateralRestriction: "0",
		}
		assert.Equal(t, exp, result, "GetCryptoLoanPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCryptoLoanPosition should return a result")
}

func TestGetCryptoLoanMaxLoanAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetCryptoLoanMaxLoanAmount(t.Context(), currency.EMPTYCODE, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCryptoLoanMaxLoanAmount must reject an empty coin")
	_, err = e.GetCryptoLoanMaxLoanAmount(t.Context(), currency.BTC, []CryptoLoanCollateral{{Amount: 10000}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCryptoLoanMaxLoanAmount must reject an empty collateral coin")
	_, err = e.GetCryptoLoanMaxLoanAmount(t.Context(), currency.BTC, []CryptoLoanCollateral{{Currency: currency.USDT}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "GetCryptoLoanMaxLoanAmount must reject a zero collateral amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCryptoLoanMaxLoanAmount(t.Context(), currency.BTC, []CryptoLoanCollateral{{Currency: currency.USDT, Amount: 10000}})
	require.NoError(t, err, "GetCryptoLoanMaxLoanAmount must not error")
	if mockTests {
		exp := &CryptoLoanMaxLoanAmountResponse{
			Currency:       currency.BTC,
			MaxLoan:        0.1722,
			NotionalUSD:    16456.06,
			RemainingQuota: 9999999.9421,
		}
		assert.Equal(t, exp, result, "GetCryptoLoanMaxLoanAmount should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetCryptoLoanMaxLoanAmount should return the loan coin")
}

func TestGetFlexibleLoanAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanAvailableInventory(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFlexibleLoanAvailableInventory must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanAvailableInventory(t.Context(), currency.USDT)
	require.NoError(t, err, "GetFlexibleLoanAvailableInventory must not error")
	if mockTests {
		exp := &FlexibleLoanAvailableInventoryResponse{
			Currency:           currency.USDT,
			AvailableInventory: 100000000,
			UpdateTime:         types.Time(time.Unix(1699272487, 0)),
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanAvailableInventory should decode every field")
		return
	}
	assert.Equal(t, currency.USDT, result.Currency, "GetFlexibleLoanAvailableInventory should return the coin")
}

func TestBorrowFlexibleLoan(t *testing.T) {
	t.Parallel()
	_, err := e.BorrowFlexibleLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BorrowFlexibleLoan must reject a nil request")
	_, err = e.BorrowFlexibleLoan(t.Context(), &BorrowFlexibleLoanRequest{LoanAmount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "BorrowFlexibleLoan must reject an empty loan coin")
	_, err = e.BorrowFlexibleLoan(t.Context(), &BorrowFlexibleLoanRequest{LoanCurrency: currency.BTC})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "BorrowFlexibleLoan must reject a zero loan amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BorrowFlexibleLoan(t.Context(), &BorrowFlexibleLoanRequest{
		LoanCurrency:   currency.BTC,
		LoanAmount:     0.1,
		CollateralList: []CryptoLoanCollateral{{Currency: currency.USDT, Amount: 20000}},
	})
	require.NoError(t, err, "BorrowFlexibleLoan must not error")
	if mockTests {
		exp := &CryptoLoanOrderResponse{
			OrderID: "1363",
		}
		assert.Equal(t, exp, result, "BorrowFlexibleLoan should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "BorrowFlexibleLoan should return an order ID")
}

func TestRepayFlexibleLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayFlexibleLoan(t.Context(), currency.EMPTYCODE, 0.005)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayFlexibleLoan must reject an empty loan coin")
	_, err = e.RepayFlexibleLoan(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RepayFlexibleLoan must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayFlexibleLoan(t.Context(), currency.BTC, 0.005)
	require.NoError(t, err, "RepayFlexibleLoan must not error")
	if mockTests {
		exp := &CryptoLoanRepayResponse{
			RepayID: "1771",
		}
		assert.Equal(t, exp, result, "RepayFlexibleLoan should decode every field")
		return
	}
	assert.NotEmpty(t, result.RepayID, "RepayFlexibleLoan should return a repayment ID")
}

func TestRepayFlexibleLoanWithCollateral(t *testing.T) {
	t.Parallel()
	err := e.RepayFlexibleLoanWithCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RepayFlexibleLoanWithCollateral must reject a nil request")
	err = e.RepayFlexibleLoanWithCollateral(t.Context(), &RepayFlexibleLoanWithCollateralRequest{CollateralCoins: []currency.Code{currency.USDT}, Amount: 0.005})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayFlexibleLoanWithCollateral must reject an empty loan coin")
	err = e.RepayFlexibleLoanWithCollateral(t.Context(), &RepayFlexibleLoanWithCollateralRequest{LoanCurrency: currency.BTC, Amount: 0.005})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayFlexibleLoanWithCollateral must reject no collateral coins")
	err = e.RepayFlexibleLoanWithCollateral(t.Context(), &RepayFlexibleLoanWithCollateralRequest{LoanCurrency: currency.BTC, CollateralCoins: []currency.Code{currency.USDT}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RepayFlexibleLoanWithCollateral must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RepayFlexibleLoanWithCollateral(t.Context(), &RepayFlexibleLoanWithCollateralRequest{
		LoanCurrency:    currency.BTC,
		CollateralCoins: []currency.Code{currency.USDT, currency.ETH},
		Amount:          0.005,
	})
	assert.NoError(t, err, "RepayFlexibleLoanWithCollateral should not error")
}

func TestGetFlexibleLoans(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoans(t.Context(), currency.ETH)
	require.NoError(t, err, "GetFlexibleLoans must not error")
	if mockTests {
		exp := &FlexibleLoansResponse{
			List: []FlexibleLoan{
				{
					HourlyInterestRate: 0.0000018847396,
					LoanCurrency:       currency.ETH,
					TotalDebt:          0.10000019,
					UnpaidAmount:       0.1,
					UnpaidInterest:     0.00000019,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleLoans should return a result")
}

func TestGetFlexibleLoanBorrowingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanBorrowingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanBorrowingHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanBorrowingHistory(t.Context(), &FlexibleLoanBorrowingHistoryRequest{
		OrderID:      "1364",
		LoanCurrency: currency.BTC,
		Limit:        2,
		Cursor:       "1365",
	})
	require.NoError(t, err, "GetFlexibleLoanBorrowingHistory must not error")
	if mockTests {
		exp := &FlexibleLoanBorrowingHistoryResponse{
			List: []FlexibleLoanBorrow{
				{
					BorrowTime:        types.Time(time.UnixMilli(1752569950643)),
					InitialLoanAmount: 0.006,
					LoanCurrency:      currency.BTC,
					OrderID:           "1364",
					Status:            1,
				},
			},
			NextPageCursor: "1363",
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanBorrowingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleLoanBorrowingHistory should return a result")
}

func TestGetFlexibleLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanRepaymentHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanRepaymentHistory(t.Context(), &FlexibleLoanRepaymentHistoryRequest{
		RepayID:      "1773",
		LoanCurrency: currency.BTC,
		Limit:        3,
		Cursor:       "1774",
	})
	require.NoError(t, err, "GetFlexibleLoanRepaymentHistory must not error")
	if mockTests {
		exp := &FlexibleLoanRepaymentHistoryResponse{
			List: []FlexibleLoanRepayment{
				{
					LoanCurrency: currency.BTC,
					RepayAmount:  0.007,
					RepayID:      "1773",
					RepayStatus:  1,
					RepayTime:    types.Time(time.UnixMilli(1752570731274)),
					RepayType:    1,
				},
			},
			NextPageCursor: "1772",
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanRepaymentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleLoanRepaymentHistory should return a result")
}

func TestGetFixedLoanLendingMarket(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanLendingMarket(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanLendingMarket must reject a nil request")
	_, err = e.GetFixedLoanLendingMarket(t.Context(), &FixedLoanMarketRequest{OrderBy: "apy"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFixedLoanLendingMarket must reject an empty coin")
	_, err = e.GetFixedLoanLendingMarket(t.Context(), &FixedLoanMarketRequest{OrderCurrency: currency.USDT})
	require.ErrorIs(t, err, errOrderByRequired, "GetFixedLoanLendingMarket must reject an empty order by")

	result, err := e.GetFixedLoanLendingMarket(t.Context(), &FixedLoanMarketRequest{
		OrderCurrency: currency.USDT,
		Term:          7,
		OrderBy:       "apy",
		Sort:          1,
		Limit:         2,
	})
	require.NoError(t, err, "GetFixedLoanLendingMarket must not error")
	if mockTests {
		exp := &FixedLoanMarketResponse{
			List: []FixedLoanMarketQuote{
				{
					OrderCurrency: currency.USDT,
					Term:          7,
					AnnualRate:    0.0336,
					Quantity:      2064.43,
				},
				{
					OrderCurrency: currency.USDT,
					Term:          7,
					AnnualRate:    0.0335,
					Quantity:      23337.0173,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFixedLoanLendingMarket should decode every field")
		return
	}
	for i := range result.List {
		assert.Equalf(t, uint64(7), result.List[i].Term, "GetFixedLoanLendingMarket should return 7 day orders only, got %d days", result.List[i].Term)
	}
}

func TestGetFixedLoanBorrowingMarket(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanBorrowingMarket(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanBorrowingMarket must reject a nil request")
	_, err = e.GetFixedLoanBorrowingMarket(t.Context(), &FixedLoanMarketRequest{OrderBy: "apy"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFixedLoanBorrowingMarket must reject an empty coin")
	_, err = e.GetFixedLoanBorrowingMarket(t.Context(), &FixedLoanMarketRequest{OrderCurrency: currency.USDT})
	require.ErrorIs(t, err, errOrderByRequired, "GetFixedLoanBorrowingMarket must reject an empty order by")

	result, err := e.GetFixedLoanBorrowingMarket(t.Context(), &FixedLoanMarketRequest{OrderCurrency: currency.USDT, OrderBy: "apy"})
	require.NoError(t, err, "GetFixedLoanBorrowingMarket must not error")
	if mockTests {
		exp := &FixedLoanMarketResponse{
			List: []FixedLoanMarketQuote{
				{
					OrderCurrency: currency.USDT,
					Term:          7,
					AnnualRate:    0.038,
					Quantity:      1694,
				},
				{
					OrderCurrency: currency.USDT,
					Term:          30,
					AnnualRate:    0.0385,
					Quantity:      18334.0494,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFixedLoanBorrowingMarket should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetFixedLoanBorrowingMarket should return supply orders")
}

func TestFixedLoanMarketPath(t *testing.T) {
	t.Parallel()
	_, err := fixedLoanMarketPath("/v5/crypto-loan-fixed/supply-order-quote", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "fixedLoanMarketPath must reject a nil request")
	_, err = fixedLoanMarketPath("/v5/crypto-loan-fixed/supply-order-quote", &FixedLoanMarketRequest{OrderBy: "apy"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "fixedLoanMarketPath must reject an empty coin")
	_, err = fixedLoanMarketPath("/v5/crypto-loan-fixed/supply-order-quote", &FixedLoanMarketRequest{OrderCurrency: currency.USDT})
	require.ErrorIs(t, err, errOrderByRequired, "fixedLoanMarketPath must reject an empty order by")

	for _, tc := range []struct {
		name string
		req  *FixedLoanMarketRequest
		exp  string
	}{
		{
			name: "required parameters",
			req:  &FixedLoanMarketRequest{OrderCurrency: currency.NewCode("usdt"), OrderBy: "apy"},
			exp:  "/v5/crypto-loan-fixed/supply-order-quote?orderBy=apy&orderCurrency=USDT",
		},
		{
			name: "every parameter",
			req:  &FixedLoanMarketRequest{OrderCurrency: currency.ETH, Term: 30, OrderBy: "quantity", Sort: 1, Limit: 100},
			exp:  "/v5/crypto-loan-fixed/supply-order-quote?limit=100&orderBy=quantity&orderCurrency=ETH&sort=1&term=30",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := fixedLoanMarketPath("/v5/crypto-loan-fixed/supply-order-quote", tc.req)
			require.NoError(t, err, "fixedLoanMarketPath must not error")
			assert.Equal(t, tc.exp, path, "fixedLoanMarketPath should append the set parameters to the path")
		})
	}
}

func TestGetFixedLoanAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanAvailableInventory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanAvailableInventory must reject a nil request")
	_, err = e.GetFixedLoanAvailableInventory(t.Context(), &FixedLoanAvailableInventoryRequest{Term: 7, AnnualRate: 0.02})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFixedLoanAvailableInventory must reject an empty coin")
	_, err = e.GetFixedLoanAvailableInventory(t.Context(), &FixedLoanAvailableInventoryRequest{Currency: currency.USDT, AnnualRate: 0.02})
	require.ErrorIs(t, err, errLoanTermRequired, "GetFixedLoanAvailableInventory must reject an empty term")
	_, err = e.GetFixedLoanAvailableInventory(t.Context(), &FixedLoanAvailableInventoryRequest{Currency: currency.USDT, Term: 7})
	require.ErrorIs(t, err, errAnnualRateRequired, "GetFixedLoanAvailableInventory must reject an empty annual rate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanAvailableInventory(t.Context(), &FixedLoanAvailableInventoryRequest{Currency: currency.USDT, Term: 7, AnnualRate: 0.02})
	require.NoError(t, err, "GetFixedLoanAvailableInventory must not error")
	if mockTests {
		exp := &FixedLoanAvailableInventoryResponse{
			Currency:           currency.USDT,
			Term:               7,
			AnnualRate:         0.02,
			AvailableInventory: 50000000,
			UpdateTime:         types.Time(time.Unix(1699272487, 0)),
		}
		assert.Equal(t, exp, result, "GetFixedLoanAvailableInventory should decode every field")
		return
	}
	assert.Equal(t, currency.USDT, result.Currency, "GetFixedLoanAvailableInventory should return the coin")
}

func TestCreateFixedLoanBorrowOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFixedLoanBorrowOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFixedLoanBorrowOrder must reject a nil request")
	_, err = e.CreateFixedLoanBorrowOrder(t.Context(), &CreateFixedLoanBorrowOrderRequest{OrderAmount: 1.5, AnnualRate: 0.022, Term: 30})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFixedLoanBorrowOrder must reject an empty coin")
	_, err = e.CreateFixedLoanBorrowOrder(t.Context(), &CreateFixedLoanBorrowOrderRequest{OrderCurrency: currency.ETH, AnnualRate: 0.022, Term: 30})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateFixedLoanBorrowOrder must reject a zero amount")
	_, err = e.CreateFixedLoanBorrowOrder(t.Context(), &CreateFixedLoanBorrowOrderRequest{OrderCurrency: currency.ETH, OrderAmount: 1.5, Term: 30})
	require.ErrorIs(t, err, errAnnualRateRequired, "CreateFixedLoanBorrowOrder must reject an empty annual rate")
	_, err = e.CreateFixedLoanBorrowOrder(t.Context(), &CreateFixedLoanBorrowOrderRequest{OrderCurrency: currency.ETH, OrderAmount: 1.5, AnnualRate: 0.022})
	require.ErrorIs(t, err, errLoanTermRequired, "CreateFixedLoanBorrowOrder must reject an empty term")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFixedLoanBorrowOrder(t.Context(), &CreateFixedLoanBorrowOrderRequest{
		OrderCurrency:  currency.ETH,
		OrderAmount:    1.5,
		AnnualRate:     0.022,
		Term:           30,
		RepayType:      "1",
		StrategyType:   "PARTIAL",
		CollateralList: []CryptoLoanCollateral{{Currency: currency.BTC, Amount: 0.1}},
	})
	require.NoError(t, err, "CreateFixedLoanBorrowOrder must not error")
	if mockTests {
		exp := &CryptoLoanOrderResponse{
			OrderID: "13007",
		}
		assert.Equal(t, exp, result, "CreateFixedLoanBorrowOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CreateFixedLoanBorrowOrder should return an order ID")
}

func TestRenewFixedLoanBorrowOrder(t *testing.T) {
	t.Parallel()
	_, err := e.RenewFixedLoanBorrowOrder(t.Context(), "", nil)
	require.ErrorIs(t, err, errLoanIDRequired, "RenewFixedLoanBorrowOrder must reject an empty loan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RenewFixedLoanBorrowOrder(t.Context(), "2364", []CryptoLoanCollateral{{Currency: currency.BTC, Amount: 0.01}})
	require.NoError(t, err, "RenewFixedLoanBorrowOrder must not error")
	if mockTests {
		exp := &RenewFixedLoanBorrowOrderResponse{
			OrderID: preciseNumber(t, "49"),
		}
		assert.Equal(t, exp, result, "RenewFixedLoanBorrowOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "RenewFixedLoanBorrowOrder should return an order ID")
}

func TestRenewFixedLoanBorrowOrderResponseUnmarshal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, exp string
	}{
		{name: "quoted order ID", input: `{"orderId":"49"}`, exp: "49"},
		{name: "bare order ID", input: `{"orderId":49}`, exp: "49"},
		// An ID beyond float64's precision keeps every digit
		{name: "bare 18 digit order ID", input: `{"orderId":612349292702654771}`, exp: "612349292702654771"},
		{name: "quoted 18 digit order ID", input: `{"orderId":"612349292702654771"}`, exp: "612349292702654771"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result RenewFixedLoanBorrowOrderResponse
			require.NoError(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error")
			assert.Equal(t, RenewFixedLoanBorrowOrderResponse{OrderID: preciseNumber(t, tc.exp)}, result, "RenewFixedLoanBorrowOrderResponse should decode the order ID")
		})
	}
}

func TestCreateFixedLoanSupplyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFixedLoanSupplyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFixedLoanSupplyOrder must reject a nil request")
	_, err = e.CreateFixedLoanSupplyOrder(t.Context(), &CreateFixedLoanSupplyOrderRequest{OrderAmount: 1020, AnnualRate: 0.01, Term: 7})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateFixedLoanSupplyOrder must reject an empty coin")
	_, err = e.CreateFixedLoanSupplyOrder(t.Context(), &CreateFixedLoanSupplyOrderRequest{OrderCurrency: currency.USDT, AnnualRate: 0.01, Term: 7})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateFixedLoanSupplyOrder must reject a zero amount")
	_, err = e.CreateFixedLoanSupplyOrder(t.Context(), &CreateFixedLoanSupplyOrderRequest{OrderCurrency: currency.USDT, OrderAmount: 1020, Term: 7})
	require.ErrorIs(t, err, errAnnualRateRequired, "CreateFixedLoanSupplyOrder must reject an empty annual rate")
	_, err = e.CreateFixedLoanSupplyOrder(t.Context(), &CreateFixedLoanSupplyOrderRequest{OrderCurrency: currency.USDT, OrderAmount: 1020, AnnualRate: 0.01})
	require.ErrorIs(t, err, errLoanTermRequired, "CreateFixedLoanSupplyOrder must reject an empty term")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFixedLoanSupplyOrder(t.Context(), &CreateFixedLoanSupplyOrderRequest{
		OrderCurrency:   currency.USDT,
		OrderAmount:     1020,
		AnnualRate:      0.01,
		Term:            7,
		AvailableSource: "0",
	})
	require.NoError(t, err, "CreateFixedLoanSupplyOrder must not error")
	if mockTests {
		exp := &CryptoLoanOrderResponse{
			OrderID: "13564",
		}
		assert.Equal(t, exp, result, "CreateFixedLoanSupplyOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CreateFixedLoanSupplyOrder should return an order ID")
}

func TestCancelFixedLoanBorrowOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelFixedLoanBorrowOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFixedLoanBorrowOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CancelFixedLoanBorrowOrder(t.Context(), "13010"), "CancelFixedLoanBorrowOrder should not error")
}

func TestCancelFixedLoanSupplyOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelFixedLoanSupplyOrder(t.Context(), "", "0")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFixedLoanSupplyOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CancelFixedLoanSupplyOrder(t.Context(), "13564", "0"), "CancelFixedLoanSupplyOrder should not error")
}

func TestGetFixedLoanBorrowContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanBorrowContractInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanBorrowContractInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanBorrowContractInfo(t.Context(), &FixedLoanBorrowContractInfoRequest{
		OrderID:       "13007",
		LoanID:        "571",
		OrderCurrency: currency.ETH,
		Term:          30,
		Limit:         10,
		Cursor:        "572",
	})
	require.NoError(t, err, "GetFixedLoanBorrowContractInfo must not error")
	if mockTests {
		exp := &FixedLoanBorrowContractInfoResponse{
			List: []FixedLoanBorrowContract{
				{
					AnnualRate:              0.022,
					AutoRepay:               true,
					BorrowCurrency:          currency.ETH,
					BorrowTime:              types.Time(time.UnixMilli(1752633756068)),
					InterestPaid:            0.002531506849315069,
					LoanID:                  "571",
					OrderID:                 "13007",
					RepaymentTime:           types.Time(time.UnixMilli(1755225756068)),
					ResidualPenaltyInterest: 0.0001,
					ResidualPrincipal:       1.4,
					Status:                  1,
					Term:                    30,
					RepayType:               "1",
				},
			},
			NextPageCursor: "570",
		}
		assert.Equal(t, exp, result, "GetFixedLoanBorrowContractInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanBorrowContractInfo should return a result")
}

func TestGetFixedLoanSupplyContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanSupplyContractInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanSupplyContractInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanSupplyContractInfo(t.Context(), &FixedLoanSupplyContractInfoRequest{
		OrderID:        "13564",
		SupplyID:       "567",
		SupplyCurrency: currency.USDT,
		Term:           7,
		Limit:          10,
		Cursor:         "568",
	})
	require.NoError(t, err, "GetFixedLoanSupplyContractInfo must not error")
	if mockTests {
		exp := &FixedLoanSupplyContractInfoResponse{
			List: []FixedLoanSupplyContract{
				{
					AnnualRate:           0.01,
					SupplyCurrency:       currency.USDT,
					SupplyTime:           types.Time(time.UnixMilli(1752482796082)),
					SupplyAmount:         800,
					InterestPaid:         0.1304109589041096,
					SupplyID:             "567",
					OrderID:              "13564",
					RedemptionTime:       types.Time(time.UnixMilli(1753087596082)),
					PenaltyInterest:      0.0001,
					ActualRedemptionTime: types.Time(time.UnixMilli(1753087596082)),
					Status:               2,
					Term:                 7,
				},
			},
			NextPageCursor: "566",
		}
		assert.Equal(t, exp, result, "GetFixedLoanSupplyContractInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanSupplyContractInfo should return a result")
}

func TestFixedLoanSupplyContractUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FixedLoanSupplyContract{SupplyCurrency: currency.USDT, InterestPaid: 0.13041095890410959}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"documented interestPaid", `{"supplyCurrency":"USDT","interestPaid":"0.13041095890410959"}`},
		{"sampled interest", `{"supplyCurrency":"USDT","interest":"0.13041095890410959"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FixedLoanSupplyContract
			require.NoError(t, json.Unmarshal([]byte(tc.data), &got), "Unmarshal must not error")
			assert.Equal(t, exp, got, "Unmarshal should decode the paid interest")
		})
	}
	var got FixedLoanSupplyContract
	assert.Error(t, json.Unmarshal([]byte(`{"interest":true}`), &got), "Unmarshal should reject a non-numeric interest")

	var both FixedLoanSupplyContract
	require.NoError(t, json.Unmarshal([]byte(`{"supplyId":"3","interestPaid":"0","interest":"0.13041095890410959"}`), &both), "Unmarshal must not error")
	assert.Equal(t, FixedLoanSupplyContract{SupplyID: "3"}, both, "Unmarshal should take the documented interestPaid when both keys are sent")

	var page FixedLoanSupplyContractInfoResponse
	require.NoError(t, json.Unmarshal([]byte(`{"list":[{"supplyId":"1","interestPaid":"1.5"}]}`), &page), "Unmarshal must not error")
	require.NoError(t, json.Unmarshal([]byte(`{"list":[{"supplyId":"2","interest":"0.13041095890410959"}]}`), &page), "Unmarshal must not error")
	assert.Equal(t, []FixedLoanSupplyContract{{SupplyID: "2", InterestPaid: 0.13041095890410959}}, page.List, "Unmarshal should take the paid interest from the page decoded, not the one before")
}

func TestGetFixedLoanBorrowOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanBorrowOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanBorrowOrderInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanBorrowOrderInfo(t.Context(), &FixedLoanOrderInfoRequest{
		OrderID:       "13010",
		OrderCurrency: currency.USDT,
		State:         "2",
		Term:          30,
		Limit:         10,
		Cursor:        "13011",
	})
	require.NoError(t, err, "GetFixedLoanBorrowOrderInfo must not error")
	if mockTests {
		exp := &FixedLoanBorrowOrderInfoResponse{
			List: []FixedLoanBorrowOrder{
				{
					AnnualRate:     0.01,
					OrderID:        13010,
					OrderTime:      types.Time(time.UnixMilli(1752654035179)),
					FilledQuantity: 1500,
					OrderQuantity:  2000,
					OrderCurrency:  currency.USDT,
					State:          2,
					Term:           30,
					RepayType:      "2",
					StrategyType:   "PARTIAL",
				},
			},
			NextPageCursor: "13009",
		}
		assert.Equal(t, exp, result, "GetFixedLoanBorrowOrderInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanBorrowOrderInfo should return a result")
}

func TestGetFixedLoanRenewOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanRenewOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanRenewOrderInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanRenewOrderInfo(t.Context(), &FixedLoanRenewOrderInfoRequest{
		OrderID:       "49",
		OrderCurrency: currency.USDT,
		Limit:         10,
		Cursor:        "50",
	})
	require.NoError(t, err, "GetFixedLoanRenewOrderInfo must not error")
	if mockTests {
		exp := &FixedLoanRenewOrderInfoResponse{
			List: []FixedLoanRenewOrder{
				{
					BorrowCurrency:  currency.USDT,
					Amount:          11,
					AutoRepay:       2,
					ContractNumber:  "2092164378648656896",
					DueTime:         types.Time(time.Unix(1766750400, 0)),
					OrderID:         49,
					LoanID:          "2364",
					RenewLoanNumber: "2092170365690461952",
					Time:            types.Time(time.UnixMilli(1764142142913)),
				},
			},
			NextPageCursor: "48",
		}
		assert.Equal(t, exp, result, "GetFixedLoanRenewOrderInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanRenewOrderInfo should return a result")
}

func TestGetFixedLoanSupplyOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanSupplyOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanSupplyOrderInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanSupplyOrderInfo(t.Context(), &FixedLoanOrderInfoRequest{
		OrderID:       "13564",
		OrderCurrency: currency.USDT,
		State:         "2",
		Term:          7,
		Limit:         10,
		Cursor:        "13565",
	})
	require.NoError(t, err, "GetFixedLoanSupplyOrderInfo must not error")
	if mockTests {
		exp := &FixedLoanSupplyOrderInfoResponse{
			List: []FixedLoanSupplyOrder{
				{
					AnnualRate:     0.01,
					OrderID:        13564,
					OrderTime:      types.Time(time.UnixMilli(1752482751043)),
					FilledQuantity: 800,
					OrderQuantity:  1020,
					OrderCurrency:  currency.USDT,
					State:          2,
					Term:           7,
				},
			},
			NextPageCursor: "13563",
		}
		assert.Equal(t, exp, result, "GetFixedLoanSupplyOrderInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanSupplyOrderInfo should return a result")
}

func TestFixedLoanOrderInfoParams(t *testing.T) {
	t.Parallel()
	_, err := fixedLoanOrderInfoParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "fixedLoanOrderInfoParams must reject a nil request")

	for _, tc := range []struct {
		name string
		req  *FixedLoanOrderInfoRequest
		exp  url.Values
	}{
		{name: "no filters", req: &FixedLoanOrderInfoRequest{}, exp: url.Values{}},
		{
			name: "every filter",
			req:  &FixedLoanOrderInfoRequest{OrderID: "13010", OrderCurrency: currency.NewCode("usdt"), State: "2", Term: 30, Limit: 10, Cursor: "13011"},
			exp: url.Values{
				"orderId":       {"13010"},
				"orderCurrency": {"USDT"},
				"state":         {"2"},
				"term":          {"30"},
				"limit":         {"10"},
				"cursor":        {"13011"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := fixedLoanOrderInfoParams(tc.req)
			require.NoError(t, err, "fixedLoanOrderInfoParams must not error")
			assert.Equal(t, tc.exp, params, "fixedLoanOrderInfoParams should send only the set filters")
		})
	}
}

func TestRepayFixedLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayFixedLoan(t.Context(), "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errLoanIDOrLoanCurrencyRequired, "RepayFixedLoan must reject neither a loan ID nor a loan coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name         string
		loanID       string
		loanCurrency currency.Code
		exp          *CryptoLoanRepayResponse
	}{
		{name: "loan ID", loanID: "571", exp: &CryptoLoanRepayResponse{RepayID: "1782"}},
		{name: "loan coin", loanCurrency: currency.ETH, exp: &CryptoLoanRepayResponse{RepayID: "1783"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.RepayFixedLoan(t.Context(), tc.loanID, tc.loanCurrency)
			require.NoError(t, err, "RepayFixedLoan must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "RepayFixedLoan should decode every field")
				return
			}
			assert.NotEmpty(t, result.RepayID, "RepayFixedLoan should return a repayment ID")
		})
	}
}

func TestRepayFixedLoanWithCollateral(t *testing.T) {
	t.Parallel()
	err := e.RepayFixedLoanWithCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RepayFixedLoanWithCollateral must reject a nil request")
	err = e.RepayFixedLoanWithCollateral(t.Context(), &RepayFixedLoanWithCollateralRequest{CollateralCoins: []currency.Code{currency.BTC}, Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayFixedLoanWithCollateral must reject an empty loan coin")
	err = e.RepayFixedLoanWithCollateral(t.Context(), &RepayFixedLoanWithCollateralRequest{LoanCurrency: currency.ETH, Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayFixedLoanWithCollateral must reject no collateral coins")
	err = e.RepayFixedLoanWithCollateral(t.Context(), &RepayFixedLoanWithCollateralRequest{LoanCurrency: currency.ETH, CollateralCoins: []currency.Code{currency.BTC}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RepayFixedLoanWithCollateral must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RepayFixedLoanWithCollateral(t.Context(), &RepayFixedLoanWithCollateralRequest{
		LoanID:          "571",
		LoanCurrency:    currency.ETH,
		CollateralCoins: []currency.Code{currency.BTC},
		Amount:          0.1,
	})
	assert.NoError(t, err, "RepayFixedLoanWithCollateral should not error")
}

func TestGetFixedLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedLoanRepaymentHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedLoanRepaymentHistory(t.Context(), &FixedLoanRepaymentHistoryRequest{
		RepayID:      "1782",
		LoanCurrency: currency.ETH,
		Limit:        10,
		Cursor:       "1783",
	})
	require.NoError(t, err, "GetFixedLoanRepaymentHistory must not error")
	if mockTests {
		exp := &FixedLoanRepaymentHistoryResponse{
			List: []FixedLoanRepayment{
				{
					Details: []FixedLoanRepaymentDetail{
						{
							LoanCurrency: currency.ETH,
							RepayAmount:  0.1,
							LoanID:       "568",
						},
						{
							LoanCurrency: currency.ETH,
							RepayAmount:  1.4,
							LoanID:       "571",
						},
					},
					LoanCurrency: currency.ETH,
					RepayAmount:  1.5,
					RepayID:      "1782",
					RepayStatus:  1,
					RepayTime:    types.Time(time.UnixMilli(1752717174353)),
					RepayType:    1,
				},
			},
			NextPageCursor: "1674",
		}
		assert.Equal(t, exp, result, "GetFixedLoanRepaymentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedLoanRepaymentHistory should return a result")
}

func TestCollateralListPayload(t *testing.T) {
	t.Parallel()
	got := collateralListPayload([]CryptoLoanCollateral{{Currency: currency.NewCode("usdt"), Amount: 10000}, {}}, "ccy")
	exp := []map[string]any{{"ccy": "USDT", "amount": "10000"}, {}}
	assert.Equal(t, exp, got, "collateralListPayload should name the coin key and leave out unset fields")
}

func TestCollateralRepaymentBody(t *testing.T) {
	t.Parallel()
	got, err := collateralRepaymentBody(currency.NewCode("btc"), []currency.Code{currency.NewCode("usdt"), currency.ETH}, 0.005)
	require.NoError(t, err, "collateralRepaymentBody must not error")
	exp := map[string]any{"loanCurrency": "BTC", "collateralCoin": "USDT,ETH", "amount": "0.005"}
	assert.Equal(t, exp, got, "collateralRepaymentBody should upper case the coins and join the collateral coins with commas")

	_, err = collateralRepaymentBody(currency.EMPTYCODE, []currency.Code{currency.USDT}, 0.005)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "collateralRepaymentBody should reject an empty loan coin")
	_, err = collateralRepaymentBody(currency.BTC, nil, 0.005)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "collateralRepaymentBody should reject no collateral coins")
	_, err = collateralRepaymentBody(currency.BTC, []currency.Code{currency.USDT, currency.EMPTYCODE}, 0.005)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "collateralRepaymentBody should reject an empty collateral coin")
	_, err = collateralRepaymentBody(currency.BTC, []currency.Code{currency.USDT}, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin, "collateralRepaymentBody should reject a zero amount")
}
