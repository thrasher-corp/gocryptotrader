package bybit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetInstitutionalLoanProductInfo(t *testing.T) {
	t.Parallel()
	exp := &InstitutionalLoanProductInfoResponse{
		MarginProductInfo: []InstitutionalLoanProduct{
			{
				ProductID:               "45",
				Leverage:                4,
				SupportSpot:             true,
				SupportContract:         true,
				SupportMarginTrading:    true,
				DeferredLiquidationLine: 0.88,
				DeferredLiquidationTime: "1800",
				WithdrawLine:            0.8,
				TransferLine:            0.8,
				SpotBuyLine:             0.85,
				SpotSellLine:            0.9,
				ContractOpenLine:        0.85,
				LiquidationLine:         0.9,
				StopLiquidationLine:     0.85,
				ContractLeverage:        3,
				TransferRatio:           1,
				SpotSymbols: []string{
					"BTCUSDT",
					"ETHUSDT",
				},
				ContractSymbols: []string{
					"BTCUSDT",
				},
				SupportUSDCContract:    true,
				SupportUSDCOptions:     true,
				USDTPerpetualOpenLine:  0.85,
				USDCContractOpenLine:   0.85,
				USDCOptionsOpenLine:    0.85,
				USDTPerpetualCloseLine: 0.9,
				USDCContractCloseLine:  0.9,
				USDCOptionsCloseLine:   0.9,
				USDCContractSymbols: []string{
					"BTCPERP",
				},
				USDCOptionsSymbols: []string{
					"BTC-26DEC26-100000-C",
				},
				MarginLeverage: 5,
				USDTPerpetualLeverage: []InstitutionalLoanSymbolLeverage{
					{
						Symbol:   "ALABUSDT",
						Leverage: 10,
					},
					{
						Symbol:   "SNTUSDT",
						Leverage: 3,
					},
				},
				USDCContractLeverage: []InstitutionalLoanSymbolLeverage{
					{
						Symbol:   "TAOPERP",
						Leverage: 3,
					},
					{
						Symbol:   "ICNTPERP",
						Leverage: 3,
					},
				},
				ProductType: "Default",
			},
		},
	}
	public := testInstance(t)
	public.SetCredentials(nil)
	for _, tc := range []struct {
		name string
		ex   *Exchange
	}{
		{name: "signed with credentials", ex: e},
		{name: "public without credentials", ex: public},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetInstitutionalLoanProductInfo(t.Context(), "")
			require.NoError(t, err, "GetInstitutionalLoanProductInfo must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetInstitutionalLoanProductInfo should decode every field")
				return
			}
			assert.NotEmpty(t, result.MarginProductInfo, "GetInstitutionalLoanProductInfo should return products")
		})
	}
}

func TestGetInstitutionalLoanMarginCoinInfo(t *testing.T) {
	t.Parallel()
	exp := &InstitutionalLoanMarginCoinInfoResponse{
		MarginToken: []InstitutionalLoanMarginToken{
			{
				ProductID: "45",
				TokenInfo: []InstitutionalLoanMarginTokenInfo{
					{
						Token: currency.USDT,
						ConvertRatioList: []InstitutionalLoanConvertRatio{
							{
								Ladder:       "0-99999999999",
								ConvertRatio: 1,
							},
						},
					},
					{
						Token: currency.BTC,
						ConvertRatioList: []InstitutionalLoanConvertRatio{
							{
								Ladder:       "0-30000000",
								ConvertRatio: 1,
							},
							{
								Ladder:       "30000000-50000000",
								ConvertRatio: 0.98,
							},
						},
					},
				},
			},
		},
	}
	public := testInstance(t)
	public.SetCredentials(nil)
	for _, tc := range []struct {
		name string
		ex   *Exchange
	}{
		{name: "signed with credentials", ex: e},
		{name: "public without credentials", ex: public},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetInstitutionalLoanMarginCoinInfo(t.Context(), "")
			require.NoError(t, err, "GetInstitutionalLoanMarginCoinInfo must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetInstitutionalLoanMarginCoinInfo should decode every field")
				return
			}
			assert.NotEmpty(t, result.MarginToken, "GetInstitutionalLoanMarginCoinInfo should return margin coins")
		})
	}
}

func TestInstitutionalLoanPublicDataSignsWithCredentials(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		call func(*Exchange, context.Context) error
	}{
		{name: "GetInstitutionalLoanProductInfo", call: func(ex *Exchange, ctx context.Context) error {
			_, err := ex.GetInstitutionalLoanProductInfo(ctx, "45")
			return err
		}},
		{name: "GetInstitutionalLoanMarginCoinInfo", call: func(ex *Exchange, ctx context.Context) error {
			_, err := ex.GetInstitutionalLoanMarginCoinInfo(ctx, "45")
			return err
		}},
	} {
		for _, signed := range []bool{true, false} {
			t.Run(tc.name+"/signed="+strconv.FormatBool(signed), func(t *testing.T) {
				t.Parallel()
				var apiKey, query string
				ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					apiKey, query = r.Header.Get("X-BAPI-API-KEY"), r.URL.RawQuery
					_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"","result":{},"retExtInfo":{},"time":1789012718462}`)
					assert.NoError(t, err, "writing the response should not error")
				})
				if !signed {
					ex.SetCredentials(nil)
				}
				require.NoErrorf(t, tc.call(ex, t.Context()), "%s must not error", tc.name)
				assert.Equalf(t, signed, apiKey != "", "%s should sign only when credentials are set", tc.name)
				assert.Equalf(t, "productId=45", query, "%s should send the product ID either way", tc.name)
			})
		}
	}
}

func TestGetInstitutionalLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstitutionalLoanOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstitutionalLoanOrders must reject a nil request")
	_, err = e.GetInstitutionalLoanOrders(t.Context(), &InstitutionalLoanOrdersRequest{StartTime: time.Date(2023, 7, 20, 0, 0, 0, 0, time.UTC), EndTime: time.Date(2023, 7, 19, 0, 0, 0, 0, time.UTC)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInstitutionalLoanOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionalLoanOrders(t.Context(), &InstitutionalLoanOrdersRequest{
		OrderID:   "1468005106166530304",
		StartTime: time.Date(2023, 7, 19, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 7, 20, 0, 0, 0, 0, time.UTC),
		Limit:     10,
	})
	require.NoError(t, err, "GetInstitutionalLoanOrders must not error")
	if mockTests {
		exp := &InstitutionalLoanOrdersResponse{
			LoanInfo: []InstitutionalLoanOrder{
				{
					OrderID:                 "1468005106166530304",
					OrderProductID:          "96",
					ParentUID:               "100000001",
					LoanTime:                types.Time(time.Unix(1689735916, 0)),
					LoanCoin:                currency.USDT,
					LoanAmount:              204,
					UnpaidAmount:            52.07924201,
					UnpaidInterest:          0.0125,
					RepaidAmount:            151.92075799,
					RepaidInterest:          0.0391,
					InterestRate:            0.00019178,
					Status:                  "1",
					Leverage:                4,
					SupportSpot:             true,
					SupportContract:         true,
					WithdrawLine:            0.6,
					TransferLine:            0.65,
					SpotBuyLine:             0.71,
					SpotSellLine:            0.71,
					ContractOpenLine:        0.71,
					DeferredLiquidationLine: 0.88,
					DeferredLiquidationTime: "1800",
					ReserveToken:            currency.USDT,
					ReserveQuantity:         10,
					LiquidationLine:         0.75,
					StopLiquidationLine:     0.35,
					ContractLeverage:        7,
					TransferRatio:           1,
					SpotSymbols: []string{
						"BTCUSDT",
					},
					ContractSymbols: []string{
						"BTCUSDT",
					},
					SupportUSDCContract:    true,
					SupportUSDCOptions:     true,
					SupportMarginTrading:   true,
					USDTPerpetualOpenLine:  0.71,
					USDCContractOpenLine:   0.71,
					USDCOptionsOpenLine:    0.71,
					USDTPerpetualCloseLine: 0.71,
					USDCContractCloseLine:  0.71,
					USDCOptionsCloseLine:   0.71,
					USDCContractSymbols: []string{
						"BTCPERP",
					},
					USDCOptionsSymbols: []string{
						"BTC-26DEC26-100000-C",
					},
					MarginLeverage: 4,
					USDTPerpetualLeverage: []InstitutionalLoanSymbolLeverage{
						{
							Symbol:   "SUSHIUSDT",
							Leverage: 7,
						},
						{
							Symbol:   "INJUSDT",
							Leverage: 7,
						},
					},
					USDCContractLeverage: []InstitutionalLoanSymbolLeverage{
						{
							Symbol:   "BTCPERP",
							Leverage: 8,
						},
						{
							Symbol:   "ETHPERP",
							Leverage: 8,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetInstitutionalLoanOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionalLoanOrders should return a result")
}

func TestGetInstitutionalLoanRepaymentOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstitutionalLoanRepaymentOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstitutionalLoanRepaymentOrders must reject a nil request")
	_, err = e.GetInstitutionalLoanRepaymentOrders(t.Context(), &InstitutionalLoanRepaymentOrdersRequest{StartTime: time.Date(2022, 9, 15, 0, 0, 0, 0, time.UTC), EndTime: time.Date(2022, 9, 14, 0, 0, 0, 0, time.UTC)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInstitutionalLoanRepaymentOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionalLoanRepaymentOrders(t.Context(), &InstitutionalLoanRepaymentOrdersRequest{
		StartTime: time.Date(2022, 9, 14, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2022, 9, 15, 0, 0, 0, 0, time.UTC),
		Limit:     100,
	})
	require.NoError(t, err, "GetInstitutionalLoanRepaymentOrders must not error")
	if mockTests {
		exp := &InstitutionalLoanRepaymentOrdersResponse{
			RepayInfo: []InstitutionalLoanRepaymentOrder{
				{
					RepayOrderID: "8189",
					RepaidTime:   types.Time(time.Unix(1663126393, 0)),
					Token:        currency.USDT,
					Quantity:     30000,
					Interest:     12.5,
					BusinessType: "1",
					Status:       "1",
				},
			},
		}
		assert.Equal(t, exp, result, "GetInstitutionalLoanRepaymentOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionalLoanRepaymentOrders should return a result")
}

func TestGetInstitutionalLoanLTV(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionalLoanLTV(t.Context())
	require.NoError(t, err, "GetInstitutionalLoanLTV must not error")
	if mockTests {
		exp := &InstitutionalLoanLTVResponse{
			LTVInfo: []InstitutionalLoanLTV{
				{
					LTV:                      0.86,
					RemainingLiquidationTime: 1766465620,
					ParentUID:                "100000001",
					SubAccountUIDs: []string{
						"100000002",
					},
					UnpaidAmount: 30,
					UnpaidInfo: []InstitutionalLoanUnpaid{
						{
							Token:          currency.USDT,
							UnpaidQuantity: 30,
							UnpaidInterest: 0.01,
						},
					},
					Balance: 40,
					BalanceInfo: []InstitutionalLoanBalance{
						{
							Token:           currency.USDT,
							Price:           1,
							Quantity:        40,
							ConvertedAmount: 40,
						},
					},
				},
				{
					ParentUID: "100000003",
					SubAccountUIDs: []string{
						"100000004",
						"100000005",
					},
					UnpaidInfo:        []InstitutionalLoanUnpaid{},
					BalanceInfo:       []InstitutionalLoanBalance{},
					LiquidationStatus: 3,
				},
			},
			LiquidationStatus: 3,
		}
		assert.Equal(t, exp, result, "GetInstitutionalLoanLTV should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionalLoanLTV should return a result")
}

func TestBindOrUnbindUID(t *testing.T) {
	t.Parallel()
	_, err := e.BindOrUnbindUID(t.Context(), "", "0")
	require.ErrorIs(t, err, errUIDRequired, "BindOrUnbindUID must reject an empty UID")
	_, err = e.BindOrUnbindUID(t.Context(), "100000002", "")
	require.ErrorIs(t, err, errOperateRequired, "BindOrUnbindUID must reject an empty operation")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BindOrUnbindUID(t.Context(), "100000002", "0")
	require.NoError(t, err, "BindOrUnbindUID must not error")
	if mockTests {
		exp := &BindOrUnbindUIDResponse{
			UID:     "100000002",
			Operate: "0",
		}
		assert.Equal(t, exp, result, "BindOrUnbindUID should decode every field")
		return
	}
	assert.Equal(t, "100000002", result.UID, "BindOrUnbindUID should return the bound UID")
}

func TestRepayInstitutionalLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayInstitutionalLoan(t.Context(), currency.EMPTYCODE, 500000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayInstitutionalLoan must reject an empty coin")
	_, err = e.RepayInstitutionalLoan(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RepayInstitutionalLoan must reject a zero quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayInstitutionalLoan(t.Context(), currency.USDT, 500000)
	require.NoError(t, err, "RepayInstitutionalLoan must not error")
	if mockTests {
		exp := &RepayInstitutionalLoanResponse{
			RepayOrderStatus: "P",
		}
		assert.Equal(t, exp, result, "RepayInstitutionalLoan should decode every field")
		return
	}
	assert.NotEmpty(t, result.RepayOrderStatus, "RepayInstitutionalLoan should return a repayment status")
}

func TestGetInstitutionalLoanCoinDeltaAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionalLoanCoinDeltaAmount(t.Context(), currency.BTC)
	require.NoError(t, err, "GetInstitutionalLoanCoinDeltaAmount must not error")
	if mockTests {
		exp := &InstitutionalLoanCoinDeltaAmountResponse{
			RiskUnitDeltaAmount:          500000,
			RiskUnitDeltaAvailableAmount: 350000,
			RiskUnitDelta:                0.8,
			List: []InstitutionalLoanCoinDeltaInfo{
				{
					Coin:                     currency.BTC,
					CoinDeltaSize:            10,
					CoinDeltaAvailableAmount: 200000,
					CoinDeltaAmount:          300000,
				},
			},
		}
		assert.Equal(t, exp, result, "GetInstitutionalLoanCoinDeltaAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionalLoanCoinDeltaAmount should return a result")
}

func TestGetInstitutionalLoanDelayLiquidationStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionalLoanDelayLiquidationStatus(t.Context())
	require.NoError(t, err, "GetInstitutionalLoanDelayLiquidationStatus must not error")
	if mockTests {
		exp := &InstitutionalLoanDelayLiquidationStatusResponse{
			LTV:                              0.9123,
			LiquidationStatus:                3,
			DelayLiquidationStartTime:        types.Time(time.UnixMilli(1789012118462)),
			DelayLiquidationRemainingSeconds: 1200,
		}
		assert.Equal(t, exp, result, "GetInstitutionalLoanDelayLiquidationStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionalLoanDelayLiquidationStatus should return a result")
}
