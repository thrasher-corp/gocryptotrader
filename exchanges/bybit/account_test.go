package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Fixed query windows keep the mock fixtures' query strings stable
var (
	accountTestStartTime = time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
	accountTestEndTime   = time.Date(2025, 10, 2, 0, 0, 0, 0, time.UTC)
)

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetWalletBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWalletBalance must reject a nil request")
	_, err = e.GetWalletBalance(t.Context(), &WalletBalanceRequest{})
	require.ErrorIs(t, err, errMissingAccountType, "GetWalletBalance must reject a missing account type")
	_, err = e.GetWalletBalance(t.Context(), &WalletBalanceRequest{AccountType: "UNIFIED", Coins: []currency.Code{currency.BTC, currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWalletBalance must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWalletBalance(t.Context(), &WalletBalanceRequest{AccountType: "UNIFIED", Coins: []currency.Code{currency.BTC, currency.USDT}})
	require.NoError(t, err, "GetWalletBalance must not error")
	if mockTests {
		exp := &WalletBalanceResponse{
			List: []WalletBalance{
				{
					AccountType:                             "UNIFIED",
					AccountInitialMarginRate:                0.0221,
					AccountMaintenanceMarginRate:            0.0049,
					TotalEquity:                             1102.30486234,
					TotalWalletBalance:                      1099.05014562,
					TotalMarginBalance:                      1100.66014562,
					TotalAvailableBalance:                   1076.33564562,
					TotalPerpetualUnrealisedPNL:             1.61,
					TotalInitialMargin:                      24.3245,
					TotalMaintenanceMargin:                  5.3929,
					AccountInitialMarginRateByMarkPrice:     0.0221,
					AccountMaintenanceMarginRateByMarkPrice: 0.0049,
					TotalInitialMarginByMarkPrice:           24.3245,
					TotalMaintenanceMarginByMarkPrice:       5.3929,
					AccountLTV:                              0.0012,
					Coin: []WalletBalanceCoin{
						{
							Coin:                           currency.BTC,
							Equity:                         0.01694413,
							USDValue:                       1101.5,
							WalletBalance:                  0.01792413,
							Locked:                         0.002,
							SpotHedgingQuantity:            0.01592413,
							BorrowAmount:                   0.001,
							AccruedInterest:                0.00000127,
							TotalOrderInitialMargin:        0.0001,
							TotalPositionInitialMargin:     0.00025,
							TotalPositionMaintenanceMargin: 0.00005,
							UnrealisedPNL:                  0.00002,
							CumulativeRealisedPNL:          -0.00008,
							Bonus:                          0.5,
							MarginCollateral:               true,
							CollateralSwitch:               true,
							CollateralRestriction:          "1",
							SpotBorrow:                     0.001,
							Free:                           0.01592413,
							AvailableToWithdraw:            0.0085,
							AvailableToBorrow:              3,
						},
						{
							Coin:                  currency.USDT,
							Equity:                0.80486234,
							USDValue:              0.80434,
							WalletBalance:         0.80486234,
							MarginCollateral:      true,
							CollateralRestriction: "0",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetWalletBalance should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetWalletBalance should return the unified account")
}

func TestGetTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferableAmount(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTransferableAmount must reject no coins")
	_, err = e.GetTransferableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTransferableAmount must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTransferableAmount(t.Context(), currency.BTC, currency.SOL)
	require.NoError(t, err, "GetTransferableAmount must not error")
	if mockTests {
		exp := &TransferableAmountResponse{
			AvailableWithdrawal: 4.5454905,
			AvailableWithdrawalMap: map[currency.Code]types.Number{
				currency.BTC: 4.5454905,
				currency.SOL: 33.16713007,
			},
		}
		assert.Equal(t, exp, result, "GetTransferableAmount should decode every field")
		return
	}
	assert.NotEmpty(t, result.AvailableWithdrawalMap, "GetTransferableAmount should return each requested coin")
}

func TestJoinCoins(t *testing.T) {
	t.Parallel()
	_, err := joinCoins([]currency.Code{currency.BTC, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "joinCoins must reject an empty coin")
	assert.ErrorContains(t, err, "coin 1", "joinCoins should name the empty coin's position")

	for _, tc := range []struct {
		name  string
		coins []currency.Code
		exp   string
	}{
		{name: "no coins"},
		{name: "one coin", coins: []currency.Code{currency.BTC}, exp: "BTC"},
		{name: "lower case coins", coins: []currency.Code{currency.NewCode("btc"), currency.NewCode("sol"), currency.USDT}, exp: "BTC,SOL,USDT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			coins, err := joinCoins(tc.coins)
			require.NoError(t, err, "joinCoins must not error")
			assert.Equal(t, tc.exp, coins, "joinCoins should return the coins upper case and comma separated")
		})
	}
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransactionLog must reject a nil request")
	_, err = e.GetTransactionLog(t.Context(), &TransactionLogRequest{StartTime: accountTestEndTime, EndTime: accountTestStartTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTransactionLog must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *TransactionLogRequest
		exp  *TransactionLogResponse
	}{
		{
			name: "linear window",
			req: &TransactionLogRequest{
				AccountType: "UNIFIED",
				Category:    cLinear,
				Currency:    currency.USDT,
				StartTime:   accountTestStartTime,
				EndTime:     accountTestEndTime,
				Limit:       2,
			},
			exp: &TransactionLogResponse{
				List: []TransactionLogItem{
					{
						ID:              "592324_XRPUSDT_161440249321",
						Symbol:          "XRPUSDT",
						Category:        "linear",
						Side:            "Buy",
						TransactionTime: types.Time(time.Unix(1759320000, 0)),
						Type:            "SETTLEMENT",
						Quantity:        100,
						Size:            100,
						Currency:        currency.USDT,
						TradePrice:      0.3676,
						Funding:         -0.003676,
						Change:          -0.003676,
						CashBalance:     5086.55825002,
						FeeRate:         0.0001,
						TradeID:         "534c0003-4bf7-486f-aa02-78cee36825e4",
						OrderID:         "1759320000-8-592324-1-2",
						DisplayType:     "Funding",
					},
					{
						ID:                 "592324_XRPUSDT_161407743011",
						Symbol:             "XRPUSDT",
						Category:           "linear",
						Side:               "Sell",
						TransactionTime:    types.Time(time.UnixMilli(1759312382224)),
						Type:               "TRADE",
						TransactionSubType: "movePosition",
						Quantity:           88,
						Size:               -12,
						Currency:           currency.USDT,
						TradePrice:         0.3615,
						Fee:                0.0190872,
						CashFlow:           0.83142,
						Change:             0.8123328,
						CashBalance:        5086.56192602,
						FeeRate:            0.0006,
						BonusChange:        -0.1430544,
						TradeID:            "5184f079-88ec-54c7-8774-5173cafd2b4e",
						OrderID:            "592b7e41-78fd-42e2-9aa3-91e1835ef3e1",
						OrderLinkID:        "linear-order",
						ExtraFees:          `[{"feeCoin":"USDT","feeType":"GST","subFeeType":"IND_GST","feeRate":"0.0000675","fee":"0.00214729"}]`,
						DisplayType:        "Trade",
					},
				},
				NextPageCursor: "21963%3A1%2C14954%3A1",
			},
		},
		{
			name: "option move position trades",
			req: &TransactionLogRequest{
				Category:           cOption,
				BaseCoin:           currency.BTC,
				Type:               "TRADE",
				TransactionSubType: "movePosition",
				Cursor:             "21963%3A1%2C14954%3A1",
			},
			exp: &TransactionLogResponse{
				List: []TransactionLogItem{
					{
						ID:                 "592324_BTC-26DEC25-100000-C_161407743012",
						Symbol:             "BTC-26DEC25-100000-C",
						Category:           "option",
						Side:               "Buy",
						TransactionTime:    types.Time(time.UnixMilli(1759312382225)),
						Type:               "TRADE",
						TransactionSubType: "movePosition",
						Quantity:           0.1,
						Size:               0.1,
						Currency:           currency.USDT,
						TradePrice:         1250,
						Fee:                0.375,
						CashFlow:           -125,
						Change:             -125.375,
						CashBalance:        4961.18692602,
						FeeRate:            0.0003,
						TradeID:            "a8f6b2c4-7d13-5e9a-8b04-1c6f3e2d9a57",
						OrderID:            "9c4d2a71-3b8e-4f60-9a15-2d7e5c0b8f13",
						DisplayType:        "Trade",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTransactionLog(t.Context(), tc.req)
			require.NoError(t, err, "GetTransactionLog must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTransactionLog should decode every field")
				return
			}
			assert.NotNil(t, result, "GetTransactionLog should return a result")
		})
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountInfo(t.Context())
	require.NoError(t, err, "GetAccountInfo must not error")
	if mockTests {
		exp := &AccountInfoResponse{
			UnifiedMarginStatus: 5,
			MarginMode:          "REGULAR_MARGIN",
			IsMasterTrader:      true,
			SpotHedgingStatus:   "ON",
			UpdatedTime:         types.Time(time.Unix(1697078946, 0)),
			DCPStatus:           "OFF",
			TimeWindow:          10,
			SMPGroup:            12,
		}
		assert.Equal(t, exp, result, "GetAccountInfo should decode every field")
		return
	}
	assert.NotZero(t, result.UnifiedMarginStatus, "GetAccountInfo should return the account status")
}

func TestGetAccountInstrumentsInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountInstrumentsInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountInstrumentsInfo must reject a nil request")
	_, err = e.GetAccountInstrumentsInfo(t.Context(), &AccountInstrumentsInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetAccountInstrumentsInfo must reject a missing category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *AccountInstrumentsInfoRequest
		exp *AccountInstrumentsInfoResponse
	}{
		{
			req: &AccountInstrumentsInfoRequest{Category: cLinear, Limit: 3},
			exp: &AccountInstrumentsInfoResponse{
				Category: "linear",
				List: []AccountInstrument{
					{
						Symbol:          "BTC-26DEC26",
						SymbolID:        3021,
						ContractType:    "LinearFutures",
						Status:          "Trading",
						BaseCoin:        currency.BTC,
						QuoteCoin:       currency.USDC,
						LaunchTime:      types.Time(time.Unix(1750406400, 0)),
						DeliveryTime:    types.Time(time.Unix(1798272000, 0)),
						DeliveryFeeRate: 0.0005,
						PriceScale:      1,
						LeverageFilter: AccountInstrumentLeverageFilter{
							MinLeverage:  1,
							MaxLeverage:  100,
							LeverageStep: 0.01,
						},
						PriceFilter: AccountInstrumentPriceFilter{
							MinPrice: 0.5,
							MaxPrice: 1999999,
							TickSize: 0.5,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:         0.001,
							MaxOrderQuantity:         1190,
							QuantityStep:             0.001,
							MinNotionalValue:         5,
							MaxMktOrderQuantity:      500,
							PostOnlyMaxOrderQuantity: 1190,
						},
						UnifiedMarginTrade: true,
						SettleCoin:         currency.USDC,
						CopyTrading:        "none",
						FullName:           "Bitcoin",
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.05,
							PriceLimitRatioY: 0.1,
						},
						IsPublicRPI:     true,
						MyRPIPermission: true,
					},
					{
						Symbol:       "TSLAUSDT",
						SymbolID:     3388,
						ContractType: "LinearPerpetual",
						Status:       "Trading",
						BaseCoin:     currency.NewCode("TSLA"),
						QuoteCoin:    currency.USDT,
						SymbolType:   "stock",
						LaunchTime:   types.Time(time.Unix(1772438400, 0)),
						PriceScale:   2,
						LeverageFilter: AccountInstrumentLeverageFilter{
							MinLeverage:  1,
							MaxLeverage:  10,
							LeverageStep: 0.01,
						},
						PriceFilter: AccountInstrumentPriceFilter{
							MinPrice: 0.01,
							MaxPrice: 19999.98,
							TickSize: 0.01,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:         0.01,
							MaxOrderQuantity:         20000,
							QuantityStep:             0.01,
							MinNotionalValue:         5,
							MaxMktOrderQuantity:      4000,
							PostOnlyMaxOrderQuantity: 20000,
						},
						UnifiedMarginTrade: true,
						FundingInterval:    480,
						SettleCoin:         currency.USDT,
						CopyTrading:        "both",
						UpperFundingRate:   0.005,
						LowerFundingRate:   -0.005,
						DisplayName:        "TSLA",
						FullName:           "Tesla, Inc.",
						MarketRegion:       "US",
						UnderlyingTicker:   "TSLA",
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.03,
							PriceLimitRatioY: 0.05,
						},
					},
					{
						Symbol:       "NEWUSDT",
						SymbolID:     3391,
						ContractType: "LinearPerpetual",
						Status:       "PreLaunch",
						BaseCoin:     currency.NewCode("NEW"),
						QuoteCoin:    currency.USDT,
						SymbolType:   "innovation",
						LaunchTime:   types.Time(time.Unix(1760000000, 0)),
						PriceScale:   4,
						LeverageFilter: AccountInstrumentLeverageFilter{
							MinLeverage:  1,
							MaxLeverage:  2,
							LeverageStep: 0.01,
						},
						PriceFilter: AccountInstrumentPriceFilter{
							MinPrice: 0.0001,
							MaxPrice: 199.9998,
							TickSize: 0.0001,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:         1,
							MaxOrderQuantity:         100000,
							QuantityStep:             1,
							MinNotionalValue:         5,
							MaxMktOrderQuantity:      20000,
							PostOnlyMaxOrderQuantity: 100000,
						},
						UnifiedMarginTrade: true,
						FundingInterval:    240,
						SettleCoin:         currency.USDT,
						CopyTrading:        "none",
						UpperFundingRate:   0.02,
						LowerFundingRate:   -0.02,
						FullName:           "New Coin",
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.15,
							PriceLimitRatioY: 0.3,
						},
						IsPreListing: true,
						PreListingInfo: &AccountInstrumentPreListingInfo{
							CurrentAuctionPhase: "CallAuction",
							Phases: []AccountInstrumentPreListingPhase{
								{
									Phase:     "CallAuction",
									StartTime: types.Time(time.Unix(1760000000, 0)),
									EndTime:   types.Time(time.Unix(1760003600, 0)),
								},
								{
									Phase:     "ContinuousTrading",
									StartTime: types.Time(time.Unix(1760003600, 0)),
								},
							},
							AuctionFeeInfo: AccountInstrumentAuctionFeeInfo{
								AuctionFeeRate: 0.0001,
								TakerFeeRate:   0.001,
								MakerFeeRate:   0.0004,
							},
							SkipCallAuction: true,
						},
					},
				},
				NextPageCursor: "first%3DBTC-26DEC26%26last%3DNEWUSDT",
			},
		},
		{
			req: &AccountInstrumentsInfoRequest{Category: cSpot},
			exp: &AccountInstrumentsInfoResponse{
				Category: "spot",
				List: []AccountInstrument{
					{
						Symbol:    "BTCUSDT",
						SymbolID:  9,
						Status:    "Trading",
						BaseCoin:  currency.BTC,
						QuoteCoin: currency.USDT,
						PriceFilter: AccountInstrumentPriceFilter{
							TickSize: 0.01,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:          0.000001,
							MaxOrderQuantity:          17000,
							BasePrecision:             0.000001,
							QuotePrecision:            0.00000001,
							MinOrderAmount:            5,
							MaxOrderAmount:            1999999999,
							MaxLimitOrderQuantity:     17000,
							MaxMarketOrderQuantity:    8500,
							PostOnlyMaxLimitOrderSize: 60000,
						},
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.05,
							PriceLimitRatioY: 0.05,
						},
						IsPublicRPI:      true,
						MyRPIPermission:  true,
						XStockMultiplier: 1,
						MarginTrading:    "utaOnly",
					},
					{
						Symbol:     "AAPLXUSDT",
						SymbolID:   1206,
						Status:     "Trading",
						BaseCoin:   currency.NewCode("AAPLX"),
						QuoteCoin:  currency.USDT,
						SymbolType: "xstocks",
						PriceFilter: AccountInstrumentPriceFilter{
							TickSize: 0.01,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:          0.001,
							MaxOrderQuantity:          2000,
							BasePrecision:             0.001,
							QuotePrecision:            0.0001,
							MinOrderAmount:            1,
							MaxOrderAmount:            200000,
							MaxLimitOrderQuantity:     2000,
							MaxMarketOrderQuantity:    1000,
							PostOnlyMaxLimitOrderSize: 2000,
						},
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.02,
							PriceLimitRatioY: 0.04,
						},
						Innovation:       true,
						XStockMultiplier: 1.0185,
						MarginTrading:    "none",
						STTag:            true,
					},
				},
			},
		},
		{
			req: &AccountInstrumentsInfoRequest{Category: cInverse, Symbol: "BTCUSD", Limit: 1, Cursor: "first%3DBTCUSD"},
			exp: &AccountInstrumentsInfoResponse{
				Category: "inverse",
				List: []AccountInstrument{
					{
						Symbol:       "BTCUSD",
						SymbolID:     1,
						ContractType: "InversePerpetual",
						Status:       "Trading",
						BaseCoin:     currency.BTC,
						QuoteCoin:    currency.USD,
						LaunchTime:   types.Time(time.Unix(1585526400, 0)),
						PriceScale:   2,
						LeverageFilter: AccountInstrumentLeverageFilter{
							MinLeverage:  1,
							MaxLeverage:  100,
							LeverageStep: 0.01,
						},
						PriceFilter: AccountInstrumentPriceFilter{
							MinPrice: 0.5,
							MaxPrice: 1999999,
							TickSize: 0.5,
						},
						LotSizeFilter: AccountInstrumentLotSizeFilter{
							MinOrderQuantity:         1,
							MaxOrderQuantity:         25000000,
							QuantityStep:             1,
							MinNotionalValue:         1,
							MaxMktOrderQuantity:      5000000,
							PostOnlyMaxOrderQuantity: 25000000,
						},
						UnifiedMarginTrade: true,
						FundingInterval:    480,
						SettleCoin:         currency.BTC,
						CopyTrading:        "none",
						UpperFundingRate:   0.00375,
						LowerFundingRate:   -0.00375,
						FullName:           "Bitcoin",
						RiskParameters: AccountInstrumentRiskParameters{
							PriceLimitRatioX: 0.01,
							PriceLimitRatioY: 0.02,
						},
						IsPublicRPI:     true,
						MyRPIPermission: true,
					},
				},
			},
		},
	} {
		t.Run(tc.req.Category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAccountInstrumentsInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetAccountInstrumentsInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAccountInstrumentsInfo should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetAccountInstrumentsInfo should return instruments")
		})
	}
}

func TestManualBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.ManualBorrow(t.Context(), currency.EMPTYCODE, 0.01)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ManualBorrow must reject an empty coin")
	_, err = e.ManualBorrow(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ManualBorrow must reject a missing amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ManualBorrow(t.Context(), currency.BTC, 0.01)
	require.NoError(t, err, "ManualBorrow must not error")
	if mockTests {
		exp := &ManualBorrowResponse{
			Coin:   currency.BTC,
			Amount: 0.01,
		}
		assert.Equal(t, exp, result, "ManualBorrow should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Coin, "ManualBorrow should return the borrowed coin")
}

func TestManualRepayWithoutAssetConversion(t *testing.T) {
	t.Parallel()
	_, err := e.ManualRepayWithoutAssetConversion(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManualRepayWithoutAssetConversion must reject a nil request")
	_, err = e.ManualRepayWithoutAssetConversion(t.Context(), &ManualRepayRequest{Amount: 0.01})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ManualRepayWithoutAssetConversion must reject a missing coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ManualRepayWithoutAssetConversion(t.Context(), &ManualRepayRequest{Coin: currency.BTC, Amount: 0.01, RepaymentType: "FLEXIBLE"})
	require.NoError(t, err, "ManualRepayWithoutAssetConversion must not error")
	if mockTests {
		exp := &ManualRepayResponse{
			ResultStatus: "SU",
		}
		assert.Equal(t, exp, result, "ManualRepayWithoutAssetConversion should decode every field")
		return
	}
	assert.NotEmpty(t, result.ResultStatus, "ManualRepayWithoutAssetConversion should return the repayment status")
}

func TestManualRepay(t *testing.T) {
	t.Parallel()
	_, err := e.ManualRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManualRepay must reject a nil request")
	_, err = e.ManualRepay(t.Context(), &ManualRepayRequest{Amount: 0.01})
	require.ErrorIs(t, err, errRepayAmountWithoutCoin, "ManualRepay must reject an amount without a coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ManualRepay(t.Context(), &ManualRepayRequest{Coin: currency.BTC, Amount: 0.01, RepaymentType: "ALL"})
	require.NoError(t, err, "ManualRepay must not error")
	if mockTests {
		exp := &ManualRepayResponse{
			ResultStatus: "P",
		}
		assert.Equal(t, exp, result, "ManualRepay should decode every field")
		return
	}
	assert.NotEmpty(t, result.ResultStatus, "ManualRepay should return the repayment status")
}

func TestManualRepayBody(t *testing.T) {
	t.Parallel()
	exp := map[string]any{"coin": "BTC", "amount": "0.01", "repaymentType": "ALL"}
	assert.Equal(t, exp, manualRepayBody(&ManualRepayRequest{Coin: currency.NewCode("btc"), Amount: 0.01, RepaymentType: "ALL"}), "manualRepayBody should send every set parameter")
	assert.Empty(t, manualRepayBody(&ManualRepayRequest{}), "manualRepayBody should omit unset parameters")
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeRate must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *FeeRateRequest
		exp *FeeRateResponse
	}{
		{
			req: &FeeRateRequest{Category: cSpot, Symbol: "BTCUSDT"},
			exp: &FeeRateResponse{
				Category: "spot",
				List: []FeeRate{
					{
						Symbol:          "BTCUSDT",
						TakerFeeRate:    0.001,
						MakerFeeRate:    0.0008,
						RPIMakerFeeRate: 0.0007,
					},
				},
			},
		},
		{
			req: &FeeRateRequest{Category: cOption, BaseCoin: currency.BTC},
			exp: &FeeRateResponse{
				Category: "option",
				List: []FeeRate{
					{
						BaseCoin:     currency.BTC,
						TakerFeeRate: 0.0003,
						MakerFeeRate: 0.0002,
					},
				},
			},
		},
		{
			req: &FeeRateRequest{Category: cLinear, Symbol: "BTCUSDT"},
			exp: &FeeRateResponse{
				List: []FeeRate{
					{
						Symbol:          "BTCUSDT",
						TakerFeeRate:    0.00055,
						MakerFeeRate:    0.0002,
						RPIMakerFeeRate: -0.00002,
					},
				},
			},
		},
	} {
		t.Run(tc.req.Category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFeeRate(t.Context(), tc.req)
			require.NoError(t, err, "GetFeeRate must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFeeRate should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetFeeRate should return fee rates")
		})
	}
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCollateralInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCollateralInfo must not error")
	if mockTests {
		exp := &CollateralInfoResponse{
			List: []CollateralInfo{
				{
					Currency:            currency.BTC,
					HourlyBorrowRate:    0.00000147,
					MaxBorrowingAmount:  3,
					FreeBorrowingLimit:  0.01,
					FreeBorrowAmount:    0.001,
					BorrowAmount:        0.001,
					OtherBorrowAmount:   0.0002,
					AvailableToBorrow:   3,
					Borrowable:          true,
					BorrowUsageRate:     0.0004,
					MarginCollateral:    true,
					CollateralSwitch:    true,
					CollateralRatio:     0.95,
					FreeBorrowingAmount: 0.5,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCollateralInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetCollateralInfo should return the coin")
}

func TestGetDCPInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDCPInfo(t.Context())
	require.NoError(t, err, "GetDCPInfo must not error")
	if mockTests {
		exp := &DCPInfoResponse{
			DCPInfos: []DCPInfo{
				{
					Product:    "SPOT",
					DCPStatus:  "ON",
					TimeWindow: 10,
				},
				{
					Product:    "DERIVATIVES",
					DCPStatus:  "ON",
					TimeWindow: 30,
				},
			},
		}
		assert.Equal(t, exp, result, "GetDCPInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDCPInfo should return a result")
}

func TestSetCollateralCoin(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralCoin(t.Context(), currency.EMPTYCODE, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetCollateralCoin must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.SetCollateralCoin(t.Context(), currency.BTC, true), "SetCollateralCoin must not error")
}

func TestOnOff(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ON", onOff(true), "onOff should return ON for true")
	assert.Equal(t, "OFF", onOff(false), "onOff should return OFF for false")
}

func TestSetMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.SetMarginMode(t.Context(), "")
	require.ErrorIs(t, err, errInvalidMode, "SetMarginMode must reject a missing margin mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetMarginMode(t.Context(), "REGULAR_MARGIN")
	require.NoError(t, err, "SetMarginMode must not error")
	if !mockTests {
		assert.Empty(t, result.Reasons, "SetMarginMode should return no failure reasons")
		return
	}
	exp := &SetMarginModeResponse{
		Reasons: []SetMarginModeReason{},
	}
	assert.Equal(t, exp, result, "SetMarginMode should decode every field")

	// Bybit reports the failure reasons in result alongside a non-zero retCode, which the transport returns as an
	// error
	_, err = e.SetMarginMode(t.Context(), "PORTFOLIO_MARGIN")
	require.ErrorIs(t, err, errAPIResponse, "SetMarginMode must return the API error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SetMarginMode must return an APIError")
	assert.Equal(t, int64(3400045), apiErr.Code, "SetMarginMode should return the failure code")
}

func TestSetMarginModeResponseUnmarshal(t *testing.T) {
	t.Parallel()
	const inp = `{"reasons":[{"reasonCode":"3400000","reasonMsg":"Equity needs to be equal to or greater than 1000 USDC"}]}`
	var result SetMarginModeResponse
	require.NoError(t, json.Unmarshal([]byte(inp), &result), "Unmarshal must not error")
	exp := SetMarginModeResponse{Reasons: []SetMarginModeReason{{ReasonCode: "3400000", ReasonMessage: "Equity needs to be equal to or greater than 1000 USDC"}}}
	assert.Equal(t, exp, result, "SetMarginModeResponse should decode the failure reasons")
}

func TestSetSpotHedging(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.SetSpotHedging(t.Context(), true), "SetSpotHedging must not error")
}

func TestGetBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBorrowHistory must reject a nil request")
	_, err = e.GetBorrowHistory(t.Context(), &BorrowHistoryRequest{StartTime: accountTestEndTime, EndTime: accountTestStartTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBorrowHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBorrowHistory(t.Context(), &BorrowHistoryRequest{Currency: currency.BTC, StartTime: accountTestStartTime, EndTime: accountTestEndTime, Limit: 1, Cursor: "2671152%3A1%2C2671152%3A1"})
	require.NoError(t, err, "GetBorrowHistory must not error")
	if mockTests {
		exp := &BorrowHistoryResponse{
			List: []BorrowHistoryRecord{
				{
					Currency:                  currency.BTC,
					CreatedTime:               types.Time(time.UnixMilli(1759359600204)),
					BorrowCost:                0.00000129,
					HourlyBorrowRate:          0.000001216904,
					InterestBearingBorrowSize: 1.0632326570284079,
					CostExemption:             0.00000001,
					BorrowAmount:              1.0633326570284078,
					UnrealisedLoss:            0.0001,
					FreeBorrowedAmount:        0.0001,
				},
			},
			NextPageCursor: "2671153%3A1%2C2671153%3A1",
		}
		assert.Equal(t, exp, result, "GetBorrowHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBorrowHistory should return a result")
}

func TestBatchSetCollateralCoin(t *testing.T) {
	t.Parallel()
	_, err := e.BatchSetCollateralCoin(t.Context(), nil)
	require.ErrorIs(t, err, errNoCollateralCoins, "BatchSetCollateralCoin must reject no coins")
	_, err = e.BatchSetCollateralCoin(t.Context(), []SetCollateralCoinRequest{{Coin: currency.BTC}, {}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "BatchSetCollateralCoin must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchSetCollateralCoin(t.Context(), []SetCollateralCoinRequest{{Coin: currency.BTC}, {Coin: currency.ETH, CollateralSwitch: true}})
	require.NoError(t, err, "BatchSetCollateralCoin must not error")
	if mockTests {
		exp := &BatchSetCollateralCoinResponse{
			List: []CollateralCoinSwitch{
				{
					Coin:             currency.BTC,
					CollateralSwitch: "OFF",
				},
				{
					Coin:             currency.ETH,
					CollateralSwitch: "ON",
				},
			},
		}
		assert.Equal(t, exp, result, "BatchSetCollateralCoin should decode every field")
		return
	}
	assert.Len(t, result.List, 2, "BatchSetCollateralCoin should return each coin")
}

func TestGetCoinGreeks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinGreeks(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCoinGreeks must not error")
	if mockTests {
		exp := &CoinGreeksResponse{
			List: []CoinGreeks{
				{
					BaseCoin:   currency.BTC,
					TotalDelta: 0.00004001,
					TotalGamma: -0.00000009,
					TotalVega:  -0.00039689,
					TotalTheta: 0.01243824,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCoinGreeks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinGreeks should return a result")
}

func TestGetMMPState(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMPState(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, errBaseNotSet, "GetMMPState must reject an empty base coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMMPState(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMMPState must not error")
	if mockTests {
		exp := &MMPStateResponse{
			Result: []MMPState{
				{
					BaseCoin:       currency.BTC,
					MMPEnabled:     true,
					Window:         5000,
					FrozenPeriod:   100000,
					QuantityLimit:  0.01,
					DeltaLimit:     0.01,
					VegaLimit:      500000,
					MMPFrozenUntil: types.Time(time.UnixMilli(1675760625519)),
					MMPFrozen:      true,
				},
			},
		}
		assert.Equal(t, exp, result, "GetMMPState should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMMPState should return a result")
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	err := e.ResetMMP(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, errBaseNotSet, "ResetMMP must reject an empty base coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.ResetMMP(t.Context(), currency.ETH), "ResetMMP must not error")
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	err := e.SetMMP(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetMMP must reject a nil request")
	req := &SetMMPRequest{}
	err = e.SetMMP(t.Context(), req)
	require.ErrorIs(t, err, errBaseNotSet, "SetMMP must reject an empty base coin")
	req.BaseCoin = currency.ETH
	err = e.SetMMP(t.Context(), req)
	require.ErrorIs(t, err, errTimeWindowRequired, "SetMMP must reject a missing window")
	req.Window = 5 * time.Second
	err = e.SetMMP(t.Context(), req)
	require.ErrorIs(t, err, errQuantityLimitRequired, "SetMMP must reject a missing quantity limit")
	req.QuantityLimit = 50
	err = e.SetMMP(t.Context(), req)
	require.ErrorIs(t, err, errDeltaLimitRequired, "SetMMP must reject a missing delta limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.FrozenPeriod = 100 * time.Second
	req.DeltaLimit = 20
	req.VegaLimit = 400000
	require.NoError(t, e.SetMMP(t.Context(), req), "SetMMP must not error")
}

func TestGetSMPGroupID(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSMPGroupID(t.Context())
	require.NoError(t, err, "GetSMPGroupID must not error")
	if mockTests {
		exp := &SMPGroupIDResponse{
			SMPGroup: 12,
		}
		assert.Equal(t, exp, result, "GetSMPGroupID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSMPGroupID should return a result")
}

func TestGetTradeBehaviourConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradeBehaviourConfig(t.Context())
	require.NoError(t, err, "GetTradeBehaviourConfig must not error")
	if mockTests {
		exp := &TradeBehaviourConfigResponse{
			LPASpot:      true,
			LPAPerpetual: true,
			DeltaEnable:  true,
			SMSEF:        true,
			FMSEF:        true,
			SMPType:      2,
		}
		assert.Equal(t, exp, result, "GetTradeBehaviourConfig should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradeBehaviourConfig should return a result")
}

func TestSetDeltaNeutralMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetDeltaNeutralMode(t.Context(), true)
	require.NoError(t, err, "SetDeltaNeutralMode must not error")
	if mockTests {
		exp := &SetDeltaNeutralModeResponse{
			ResultStatus: "success",
		}
		assert.Equal(t, exp, result, "SetDeltaNeutralMode should decode every field")
		return
	}
	assert.NotNil(t, result, "SetDeltaNeutralMode should return a result")
}

func TestSetPriceLimitBehaviour(t *testing.T) {
	t.Parallel()
	err := e.SetPriceLimitBehaviour(t.Context(), "", true)
	require.ErrorIs(t, err, errCategoryNotSet, "SetPriceLimitBehaviour must reject a missing category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.SetPriceLimitBehaviour(t.Context(), cSpot, true), "SetPriceLimitBehaviour must not error")
}

func TestRepayLiability(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayLiability(t.Context(), currency.USDT)
	require.NoError(t, err, "RepayLiability must not error")
	if mockTests {
		exp := &RepayLiabilityResponse{
			List: []LiabilityRepayment{
				{
					Coin:              currency.BTC,
					RepaymentQuantity: 0.1054967,
				},
				{
					Coin:              currency.ETH,
					RepaymentQuantity: 2.27768114,
				},
			},
		}
		assert.Equal(t, exp, result, "RepayLiability should decode every field")
		return
	}
	assert.NotNil(t, result, "RepayLiability should return a result")
}

func TestUpgradeToUnifiedAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpgradeToUnifiedAccount(t.Context())
	require.NoError(t, err, "UpgradeToUnifiedAccount must not error")
	if mockTests {
		exp := &UnifiedAccountUpgradeResponse{
			UnifiedUpdateStatus: "FAIL",
			UnifiedUpdateMessage: UnifiedUpdateMessage{
				Message: []string{
					"Upgrade to UTA Pro failed. Please cancel all open orders first.",
					"Upgrade to UTA Pro is not allowed between 07:55 and 08:05 UTC.",
				},
			},
		}
		assert.Equal(t, exp, result, "UpgradeToUnifiedAccount should decode every field")
		return
	}
	assert.NotEmpty(t, result.UnifiedUpdateStatus, "UpgradeToUnifiedAccount should return the upgrade status")
}

func TestGetOptionAssetInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOptionAssetInfo(t.Context())
	require.NoError(t, err, "GetOptionAssetInfo must not error")
	if mockTests {
		exp := &OptionAssetInfoResponse{
			Result: []OptionAssetInfo{
				{
					Coin:                   currency.BTC,
					TotalDelta:             0.0118,
					TotalRealisedPNL:       -0.279,
					TotalUnrealisedPNL:     -47.6318,
					AssetInitialMargin:     12.531,
					AssetMaintenanceMargin: 3.1327,
					SendTime:               types.Time(time.UnixMilli(1773230923530)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetOptionAssetInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOptionAssetInfo should return a result")
}

func TestGetPayInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPayInfo(t.Context(), currency.SOL)
	require.NoError(t, err, "GetPayInfo must not error")
	if mockTests {
		exp := &PayInfoResponse{
			CollateralInfo: PayCollateralInfo{
				CollateralList: []PayCollateral{
					{
						Coin:           currency.USDT,
						AvailableSize:  13041.904274867704,
						AvailableValue: 13038.369918809216,
						CoinScale:      4,
					},
					{
						Coin:            currency.BTC,
						CoinScale:       9,
						BorrowSize:      0.001000068,
						SpotHedgeAmount: -0.001000067286851801,
						AssetFrozen:     0.0005,
					},
				},
			},
			BorrowInfo: PayBorrowInfo{
				Coin:             currency.SOL,
				BorrowSize:       0.10006838492650354,
				BorrowValue:      9.17249761916995,
				AssetFrozen:      0.01,
				AvailableBalance: 0.05,
			},
		}
		assert.Equal(t, exp, result, "GetPayInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPayInfo should return a result")
}

func TestGetTradeInfoForAnalysis(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeInfoForAnalysis(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradeInfoForAnalysis must reject a nil request")
	_, err = e.GetTradeInfoForAnalysis(t.Context(), &TradeInfoForAnalysisRequest{})
	require.ErrorIs(t, err, errSymbolMissing, "GetTradeInfoForAnalysis must reject a missing symbol")
	_, err = e.GetTradeInfoForAnalysis(t.Context(), &TradeInfoForAnalysisRequest{Symbol: "ETHUSDT", StartTime: accountTestEndTime, EndTime: accountTestStartTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradeInfoForAnalysis must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradeInfoForAnalysis(t.Context(), &TradeInfoForAnalysisRequest{Symbol: "ETHUSDT", StartTime: accountTestStartTime, EndTime: accountTestEndTime})
	require.NoError(t, err, "GetTradeInfoForAnalysis must not error")
	if mockTests {
		exp := &TradeInfoForAnalysisResponse{
			SymbolRealisedPNL:        12.5,
			NetExecutedQuantity:      -0.1,
			SumExecutedValue:         9418.6,
			SumExecutedQuantity:      2.5,
			AverageBuyExecutedPrice:  3767.75,
			SumBuyExecutedValue:      4521.3,
			SumBuyExecutedQuantity:   1.2,
			SumBuyExecutedFee:        4.52,
			SumBuyOrderQuantity:      1.5,
			AverageSellExecutedPrice: 3767.15,
			SumSellExecutedValue:     4897.3,
			SumSellExecutedQuantity:  1.3,
			SumSellExecutedFee:       4.8,
			SumSellOrderQuantity:     1.3,
			MaxMarginVersion:         3,
			BaseCoin:                 currency.ETH,
			SettleCoin:               currency.USDT,
			SumPriceList: []TradeInfoDailySum{
				{
					Day:                  "2025-10-01",
					SumBuyExecutedValue:  4521.3,
					SumSellExecutedValue: 4897.3,
					SumExecutedValue:     9418.6,
				},
			},
		}
		assert.Equal(t, exp, result, "GetTradeInfoForAnalysis should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTradeInfoForAnalysis should return a result")
}
