package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFundingAccountTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingAccountTransactionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingAccountTransactionHistory must reject a nil request")
	_, err = e.GetFundingAccountTransactionHistory(t.Context(), &FundingAccountTransactionHistoryRequest{StartTime: time.Unix(1772600000, 0)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingAccountTransactionHistory must reject a window without an end")
	_, err = e.GetFundingAccountTransactionHistory(t.Context(), &FundingAccountTransactionHistoryRequest{EndTime: time.Unix(1773200000, 0)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingAccountTransactionHistory must reject a window without a start")
	_, err = e.GetFundingAccountTransactionHistory(t.Context(), &FundingAccountTransactionHistoryRequest{StartTime: time.Unix(1773200000, 0), EndTime: time.Unix(1772600000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingAccountTransactionHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundingAccountTransactionHistory(t.Context(), &FundingAccountTransactionHistoryRequest{StartTime: time.Unix(1772600000, 0), EndTime: time.Unix(1773200000, 0), Limit: 2})
	require.NoError(t, err, "GetFundingAccountTransactionHistory must not error")
	t.Run("next page", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetFundingAccountTransactionHistory(t.Context(), &FundingAccountTransactionHistoryRequest{StartTime: time.Unix(1772600000, 0), EndTime: time.Unix(1773200000, 0), Limit: 2, Cursor: result.NextPageCursor})
		require.NoError(t, err, "GetFundingAccountTransactionHistory must not error")
		if mockTests {
			exp := &FundingAccountTransactionHistoryResponse{
				List: []FundingAccountTransaction{
					{
						MemberID:                "100000001",
						Currency:                currency.USDT,
						IODirection:             "I",
						TransactionAmount:       1200.5,
						AfterAmount:             1200.5,
						CreateTime:              types.Time(time.Unix(1772640000, 0)),
						ShowBusinessType:        "fundingAccountRecordDeposit",
						ShowBusinessTypeEnglish: "Deposit",
						Description:             "fundingAccountRecordOnChainDeposit",
						DescriptionEnglish:      "Deposit | On-chain Deposit",
						CurrcCursor:             "NTg2MDQyODg4OA==",
					},
				},
			}
			assert.Equal(t, exp, next, "GetFundingAccountTransactionHistory should send the first page's cursor and decode the last page")
			return
		}
		assert.NotNil(t, next, "GetFundingAccountTransactionHistory should return a page")
	})
	if mockTests {
		exp := &FundingAccountTransactionHistoryResponse{
			NextPageCursor: "MTM3MTU3OTk=",
			List: []FundingAccountTransaction{
				{
					MemberID:                "100000001",
					Currency:                currency.BTC,
					IODirection:             "I",
					TransactionAmount:       0.00003561,
					AfterAmount:             7.5547230662687035,
					CreateTime:              types.Time(time.Unix(1772669763, 0)),
					ShowBusinessType:        "fundingAccountRecordEarn",
					ShowBusinessTypeEnglish: "Earn",
					Description:             "fundingAccountRecordFlexSavingInterestDistribution",
					DescriptionEnglish:      "Easy Earn | Flexible Interest Distribution",
					CurrcCursor:             "NTg2MDQzMTI0OA==",
				},
				{
					MemberID:                "100000001",
					Currency:                currency.USDT,
					IODirection:             "O",
					TransactionAmount:       25,
					AfterAmount:             1175.5,
					CreateTime:              types.Time(time.Unix(1772660000, 0)),
					ShowBusinessType:        "fundingAccountRecordTransfer",
					ShowBusinessTypeEnglish: "Transfer",
					Description:             "fundingAccountRecordTransferToUnified",
					DescriptionEnglish:      "Transfer | To Unified Trading Account",
					CurrcCursor:             "NTg2MDQyOTk5MQ==",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFundingAccountTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFundingAccountTransactionHistory should return a page")
}

func TestGetDeliveryRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeliveryRecord must reject a nil request")
	_, err = e.GetDeliveryRecord(t.Context(), &DeliveryRecordRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetDeliveryRecord must reject an empty category")
	_, err = e.GetDeliveryRecord(t.Context(), &DeliveryRecordRequest{Category: cOption, StartTime: time.UnixMilli(1672300800860), EndTime: time.UnixMilli(1672214400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDeliveryRecord must reject a reversed window")

	arg := &DeliveryRecordRequest{Category: cOption, Limit: 20}
	if mockTests {
		// A single-digit day proves the unpadded expDate, 1DEC23; live runs omit it, as the API rejects dates over 2 years old
		arg.ExpiryDate = time.Date(2023, time.December, 1, 8, 0, 0, 0, time.UTC)
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryRecord(t.Context(), arg)
	require.NoError(t, err, "GetDeliveryRecord must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetDeliveryRecord(t.Context(), &DeliveryRecordRequest{
			Category:   cOption,
			Symbol:     "BTC-26SEP26-80000-C",
			StartTime:  time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC),
			ExpiryDate: time.Date(2026, time.September, 26, 8, 0, 0, 0, time.UTC),
			Limit:      20,
			Cursor:     result.NextPageCursor,
		})
		require.NoError(t, err, "GetDeliveryRecord must not error")
		if mockTests {
			exp := &DeliveryRecordResponse{
				Category: "option",
				List: []DeliveryRecord{
					{
						DeliveryTime:        types.Time(time.UnixMilli(1790409600000)),
						Symbol:              "BTC-26SEP26-80000-C",
						Side:                "Sell",
						Position:            0.02,
						EntryPrice:          2580.1,
						DeliveryPrice:       81250.4,
						Strike:              80000,
						Fee:                 0.35,
						DeliveryRealisedPNL: 26.594,
					},
				},
			}
			assert.Equal(t, exp, next, "GetDeliveryRecord should send every filter and decode the matching page")
			return
		}
		assert.Equal(t, cOption, next.Category, "GetDeliveryRecord should return the requested category")
	})
	if mockTests {
		exp := &DeliveryRecordResponse{
			Category: "option",
			List: []DeliveryRecord{
				{
					DeliveryTime:        types.Time(time.UnixMilli(1701417600000)),
					Symbol:              "BTC-1DEC23-48000-C",
					Side:                "Buy",
					Position:            0.01,
					EntryPrice:          120.5,
					DeliveryPrice:       37828.12,
					Strike:              48000,
					Fee:                 0.0001,
					DeliveryRealisedPNL: -1.205,
				},
			},
			NextPageCursor: "132791%3A0%2C132791%3A0",
		}
		assert.Equal(t, exp, result, "GetDeliveryRecord should decode every field")
		return
	}
	assert.Equal(t, cOption, result.Category, "GetDeliveryRecord should return the requested category")
}

func TestGetUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDCSessionSettlement(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDCSessionSettlement must reject a nil request")
	_, err = e.GetUSDCSessionSettlement(t.Context(), &USDCSessionSettlementRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetUSDCSessionSettlement must reject an empty category")
	_, err = e.GetUSDCSessionSettlement(t.Context(), &USDCSessionSettlementRequest{Category: cLinear, StartTime: time.UnixMilli(1672214400000), EndTime: time.UnixMilli(1672128000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDCSessionSettlement must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDCSessionSettlement(t.Context(), &USDCSessionSettlementRequest{Category: cLinear, Limit: 10})
	require.NoError(t, err, "GetUSDCSessionSettlement must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		next, err := e.GetUSDCSessionSettlement(t.Context(), &USDCSessionSettlementRequest{
			Category:  cLinear,
			Symbol:    "BTCPERP",
			StartTime: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
			Limit:     10,
			Cursor:    result.NextPageCursor,
		})
		require.NoError(t, err, "GetUSDCSessionSettlement must not error")
		if mockTests {
			exp := &USDCSessionSettlementResponse{
				Category: "linear",
				List: []USDCSessionSettlement{
					{
						Symbol:              "BTCPERP",
						Side:                "Sell",
						Size:                0.25,
						SessionAveragePrice: 110250.5,
						MarkPrice:           110180,
						RealisedPNL:         17.625,
						CreatedTime:         types.Time(time.UnixMilli(1788249600000)),
					},
				},
			}
			assert.Equal(t, exp, next, "GetUSDCSessionSettlement should send every filter and decode the matching page")
			return
		}
		assert.Equal(t, cLinear, next.Category, "GetUSDCSessionSettlement should return the requested category")
	})
	if mockTests {
		exp := &USDCSessionSettlementResponse{
			Category: "linear",
			List: []USDCSessionSettlement{
				{
					Symbol:              "BTCPERP",
					Side:                "Buy",
					Size:                1.5,
					SessionAveragePrice: 16667.52,
					MarkPrice:           16620,
					RealisedPNL:         -71.28,
					CreatedTime:         types.Time(time.Unix(1672214400, 0)),
				},
			},
			NextPageCursor: "116952%3A1%2C116952%3A1",
		}
		assert.Equal(t, exp, result, "GetUSDCSessionSettlement should decode every field")
		return
	}
	assert.Equal(t, cLinear, result.Category, "GetUSDCSessionSettlement should return the requested category")
}

func TestGetCoinExchangeRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinExchangeRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinExchangeRecords must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinExchangeRecords(t.Context(), &CoinExchangeRecordsRequest{Limit: 20})
	require.NoError(t, err, "GetCoinExchangeRecords must not error")
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		// Bybit takes the coins upper case only
		next, err := e.GetCoinExchangeRecords(t.Context(), &CoinExchangeRecordsRequest{FromCoin: currency.NewCode("btc"), ToCoin: currency.NewCode("eth"), Limit: 20, Cursor: result.NextPageCursor})
		require.NoError(t, err, "GetCoinExchangeRecords must not error")
		if mockTests {
			exp := &CoinExchangeRecordsResponse{
				OrderBody: []CoinExchangeRecord{
					{
						FromCoin:              currency.BTC,
						FromAmount:            0.05,
						ToCoin:                currency.ETH,
						ToAmount:              0.70421355,
						ExchangeRate:          14.084271,
						CreatedTime:           types.Time(time.Unix(1672100000, 0)),
						ExchangeTransactionID: "145102533285208544812654440001",
					},
				},
			}
			assert.Equal(t, exp, next, "GetCoinExchangeRecords should send every filter and decode the matching page")
			return
		}
		assert.NotNil(t, next, "GetCoinExchangeRecords should return a page")
	})
	if mockTests {
		exp := &CoinExchangeRecordsResponse{
			NextPageCursor: "173341:1672197760",
			OrderBody: []CoinExchangeRecord{
				{
					FromCoin:              currency.BTC,
					FromAmount:            0.1,
					ToCoin:                currency.ETH,
					ToAmount:              1.38586623,
					ExchangeRate:          13.85866238,
					CreatedTime:           types.Time(time.Unix(1672197760, 0)),
					ExchangeTransactionID: "145102533285208544812654440448",
				},
			},
		}
		assert.Equal(t, exp, result, "GetCoinExchangeRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinExchangeRecords should return a page")
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCoinInfo must not error")
	if mockTests {
		exp := &CoinInfoResponse{
			Rows: []CoinInfo{
				{
					Name: "BTC",
					Coin: currency.BTC,
					Chains: []CoinChain{
						{
							Chain:             "BTC",
							ChainType:         "BTC",
							Confirmation:      1,
							WithdrawFee:       0.0001,
							DepositMin:        0.000006,
							WithdrawMin:       0.0001,
							MinAccuracy:       8,
							ChainDeposit:      "1",
							ChainWithdraw:     "1",
							SafeConfirmNumber: 2,
							WithdrawMax:       -1,
						},
						{
							Chain:                 "BSC",
							ChainType:             "BSC (BEP20)",
							Confirmation:          15,
							WithdrawFee:           0.00001,
							DepositMin:            0.00001,
							WithdrawMin:           0.0001,
							MinAccuracy:           8,
							ChainDeposit:          "0",
							ChainWithdraw:         "0",
							WithdrawPercentageFee: 0.001,
							ContractAddress:       "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
							SafeConfirmNumber:     30,
							WithdrawMax:           100,
						},
					},
					RemainAmount: 5370,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCoinInfo should decode every field")
		return
	}
	require.NotEmpty(t, result.Rows, "GetCoinInfo must return the coin")
	assert.True(t, result.Rows[0].Coin.Equal(currency.BTC), "GetCoinInfo should return the requested coin")
}

func TestGetSubUID(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubUID(t.Context())
	require.NoError(t, err, "GetSubUID must not error")
	if mockTests {
		exp := &SubUIDResponse{
			SubMemberIDs: []string{
				"100000002",
				"100000003",
				"100000004",
			},
			TransferableSubMemberIDs: []string{
				"100000002",
				"100000003",
			},
		}
		assert.Equal(t, exp, result, "GetSubUID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubUID should return the sub UIDs")
}

func TestGetPortfolioMarginInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "GetPortfolioMarginInfo must not error")
	if mockTests {
		exp := &PortfolioMarginInfoResponse{
			Wallet: PortfolioMarginWallet{
				Equity:                       52197.86892104,
				CashBalance:                  45391.74423708,
				MarginBalance:                52145.70917115,
				AvailableBalance:             50893.36062347,
				AccountInitialMargin:         1304.50829757,
				AccountMaintenanceMargin:     1069.93399838,
				AccountMaintenanceMarginRate: 0.0204,
				AccountInitialMarginRate:     0.0249,
			},
			AssetPNLRange: []PortfolioMarginAssetPNLRange{
				{
					BaseCoin: currency.BTC,
					TotalPNLRanges: PortfolioMarginTotalPNLRanges{
						All: PortfolioMarginPNLRanges{
							PNLRanges: []PortfolioMarginPNLRange{
								{
									PriceScale: -0.1,
									PNLs: []types.Number{
										-1004.7471,
									},
								},
								{
									PriceScale: 0.1,
									PNLs: []types.Number{
										1016.1672,
									},
								},
							},
						},
						Perpetual: PortfolioMarginPNLRanges{
							PNLRanges: []PortfolioMarginPNLRange{
								{
									PriceScale: -0.1,
									PNLs: []types.Number{
										-972.1448288,
									},
								},
								{
									PriceScale: 0.1,
									PNLs: []types.Number{
										972.1448288,
									},
								},
							},
						},
						Option: PortfolioMarginPNLRanges{
							PNLRanges: []PortfolioMarginPNLRange{
								{
									PriceScale: -0.1,
									PNLs: []types.Number{
										-39.65273684,
									},
								},
								{
									PriceScale: 0.1,
									PNLs: []types.Number{
										44.02244694,
									},
								},
							},
						},
					},
					PerpetualPositionPNLRanges: []PortfolioMarginPerpetualPositionPNLRange{
						{
							SymbolName: "BTCUSDT",
							Position:   0.038,
							PNLRanges: []PortfolioMarginPNLRange{
								{
									PriceScale: -0.1,
									PNLs: []types.Number{
										-972.1448288,
									},
								},
								{
									PriceScale: 0.1,
									PNLs: []types.Number{
										972.1448288,
									},
								},
							},
							SessionAveragePrice: 75995.4295,
							MarkPrice:           255902.31,
							OrderSize:           0.002,
							ContractType:        2,
							SettleCoin:          currency.USDT,
							SymbolAlias:         "BTCUSDT",
						},
					},
					OptionExpiryDatePNLRanges: []PortfolioMarginOptionExpiryDatePNLRange{
						{
							ExpiryDateRepresentation: "25SEP26",
							PNLRanges: []PortfolioMarginPNLRange{
								{
									PriceScale: -0.1,
									PNLs: []types.Number{
										-24.06747339,
										-33.15841013,
										-39.65273684,
									},
								},
								{
									PriceScale: 0.1,
									PNLs: []types.Number{
										75.52147477,
										58.34116026,
										44.02244694,
									},
								},
							},
							OptionPositionPNLRanges: []PortfolioMarginOptionPositionPNLRange{
								{
									SymbolName: "BTC-25SEP26-80000-C-USDT",
									Position:   0.02,
									PNLRanges: []PortfolioMarginPNLRange{
										{
											PriceScale: -0.1,
											PNLs: []types.Number{
												-24.06747339,
												-33.15841013,
												-39.65273684,
											},
										},
										{
											PriceScale: 0.1,
											PNLs: []types.Number{
												75.52147477,
												58.34116026,
												44.02244694,
											},
										},
									},
									SessionAveragePrice: 2580.1,
									MarkPrice:           2608.56395729,
									OrderSize:           0.01,
									ContractType:        5,
									SettleCoin:          currency.USDT,
								},
							},
						},
					},
					Contingency: PortfolioMarginContingency{
						OptionContingency:      0.8037046,
						FutureDeltaContingency: 56.89174177,
						OptionVegaContingency:  1.2,
						ContingencyComponents:  59.79915097,
						USDTUSDCContingency:    0.1,
						FutureContingency:      0.8037046,
					},
					Asset: PortfolioMarginAsset{
						Coin:                   currency.BTC,
						AssetInitialMargin:     1274.8190514137673,
						AssetMaintenanceMargin: 1062.3492095114727,
					},
					MaxLossPriceMove:      -0.1,
					MaxLossIVShock:        -0.2,
					TotalClosePositionFee: -0.35,
					SpotHedgeInfo: PortfolioMarginSpotHedgeInfo{
						HedgeSpotSize: -0.00100006,
						WalletBalance: -0.00000007,
						USDIndexPrice: 121000.5,
						PNLRanges: []PortfolioMarginPNLRange{
							{
								PriceScale: -0.1,
								PNLs: []types.Number{
									7.05047869,
								},
							},
							{
								PriceScale: 0.1,
								PNLs: []types.Number{
									0,
								},
							},
						},
					},
					MaxLossIVShockList: []types.Number{
						-0.2,
						-0.1,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginInfo should return the account")
}

func TestGetTotalMembersAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTotalMembersAssets(t.Context(), currency.BTC)
	require.NoError(t, err, "GetTotalMembersAssets must not error")
	if mockTests {
		exp := &TotalMembersAssetsResponse{
			Total:      7369376.913631028,
			QuoteTotal: 126.18796084984636,
			Stat:       1,
			List: []MemberAssets{
				{
					UID:             100000001,
					IsMaster:        true,
					OriginalBalance: 7252396.538574542,
					QuoteBalance:    124.18352734589561,
					Items: []MemberAccountAssets{
						{
							Type:            "ACCOUNT_TYPE_FUND",
							OriginalBalance: 7180203.788461284,
							QuoteBalance:    122.9486950078987,
						},
						{
							Type:            "ACCOUNT_TYPE_UNIFIED",
							OriginalBalance: 17.46702425666126,
							QuoteBalance:    0.000299092881107213,
						},
						{
							Type:            "ACCOUNT_TYPE_CONTRACT",
							OriginalBalance: 72175.28308900143,
							QuoteBalance:    1.2345332451158044,
						},
					},
				},
				{
					UID:             100000002,
					Type:            1,
					Stat:            1,
					OriginalBalance: 116980.37505648528,
					QuoteBalance:    2.004433503950749,
					Items: []MemberAccountAssets{
						{
							Type:            "ACCOUNT_TYPE_UNIFIED",
							OriginalBalance: 116980.37505648528,
							QuoteBalance:    2.004433503950749,
							Stat:            1,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetTotalMembersAssets should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetTotalMembersAssets should return the master account")
}

func TestGetSingleCoinBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleCoinBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSingleCoinBalance must reject a nil request")
	arg := &SingleCoinBalanceRequest{}
	_, err = e.GetSingleCoinBalance(t.Context(), arg)
	require.ErrorIs(t, err, errMissingAccountType, "GetSingleCoinBalance must reject an empty account type")
	arg.AccountType = "UNIFIED"
	_, err = e.GetSingleCoinBalance(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleCoinBalance must reject an empty coin")
	arg.Coin = currency.USDT
	arg.WithLTVTransferSafeAmount = true
	_, err = e.GetSingleCoinBalance(t.Context(), arg)
	require.ErrorIs(t, err, errToAccountTypeRequired, "GetSingleCoinBalance must reject an LTV transfer safe amount query without a to account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.ToAccountType = "FUND"
	arg.WithBonus = true
	arg.WithTransferSafeAmount = true
	result, err := e.GetSingleCoinBalance(t.Context(), arg)
	require.NoError(t, err, "GetSingleCoinBalance must not error")
	t.Run("sub UID to another UID", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the member IDs name placeholder UIDs, which only the fixture knows")
		}
		// The master API key queries a sub UID with memberId, and the amount it can transfer to another UID with
		// toMemberId
		balance, err := e.GetSingleCoinBalance(t.Context(), &SingleCoinBalanceRequest{MemberID: "100000002", ToMemberID: "100000001", AccountType: "FUND", Coin: currency.USDT})
		require.NoError(t, err, "GetSingleCoinBalance must not error")
		exp := &SingleCoinBalanceResponse{
			AccountType:  "FUND",
			BusinessType: 1,
			AccountID:    "100000012",
			MemberID:     "100000002",
			Balance: SingleCoinBalance{
				Coin:            currency.USDT,
				WalletBalance:   500,
				TransferBalance: 480.5,
			},
		}
		assert.Equal(t, exp, balance, "GetSingleCoinBalance should send both member IDs and decode the sub UID's balance")
	})
	if mockTests {
		exp := &SingleCoinBalanceResponse{
			AccountType:  "UNIFIED",
			BusinessType: 1,
			AccountID:    "100000011",
			MemberID:     "100000001",
			Balance: SingleCoinBalance{
				Coin:                  currency.USDT,
				WalletBalance:         11999,
				TransferBalance:       11999,
				Bonus:                 5,
				TransferSafeAmount:    11500.5,
				LTVTransferSafeAmount: 7602.4861,
			},
		}
		assert.Equal(t, exp, result, "GetSingleCoinBalance should decode every field")
		return
	}
	assert.True(t, result.Balance.Coin.Equal(currency.USDT), "GetSingleCoinBalance should return the requested coin")
}

func TestGetAllCoinsBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCoinsBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllCoinsBalance must reject a nil request")
	_, err = e.GetAllCoinsBalance(t.Context(), &AllCoinsBalanceRequest{})
	require.ErrorIs(t, err, errMissingAccountType, "GetAllCoinsBalance must reject an empty account type")
	_, err = e.GetAllCoinsBalance(t.Context(), &AllCoinsBalanceRequest{AccountType: "UNIFIED"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllCoinsBalance must require coins for the UNIFIED account")
	_, err = e.GetAllCoinsBalance(t.Context(), &AllCoinsBalanceRequest{AccountType: "FUND", Coins: []currency.Code{currency.USDT, currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllCoinsBalance must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllCoinsBalance(t.Context(), &AllCoinsBalanceRequest{AccountType: "FUND"})
	require.NoError(t, err, "GetAllCoinsBalance must not error")
	t.Run("sub-account unified coins", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the member ID names a placeholder UID, which only the fixture knows")
		}
		// The Unified account needs the coins listed, upper case and comma separated
		balance, err := e.GetAllCoinsBalance(t.Context(), &AllCoinsBalanceRequest{MemberID: "100000002", AccountType: "UNIFIED", Coins: []currency.Code{currency.USDT, currency.NewCode("usdc")}, WithBonus: true})
		require.NoError(t, err, "GetAllCoinsBalance must not error")
		exp := &AllCoinsBalanceResponse{
			AccountType: "UNIFIED",
			MemberID:    "100000002",
			Balance: []CoinBalance{
				{
					Coin:            currency.USDT,
					WalletBalance:   260.5,
					TransferBalance: 250,
					Bonus:           10.5,
				},
				{
					Coin:            currency.USDC,
					WalletBalance:   100,
					TransferBalance: 100,
				},
			},
		}
		assert.Equal(t, exp, balance, "GetAllCoinsBalance should send every filter and decode the sub-account's balances")
	})
	if mockTests {
		exp := &AllCoinsBalanceResponse{
			AccountType: "FUND",
			MemberID:    "100000001",
			Balance: []CoinBalance{
				{
					Coin:            currency.USDC,
					WalletBalance:   1520.25,
					TransferBalance: 1500,
				},
				{
					Coin:            currency.USDT,
					WalletBalance:   300,
					TransferBalance: 300,
					Bonus:           2.5,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllCoinsBalance should decode every field")
		return
	}
	assert.Equal(t, "FUND", result.AccountType, "GetAllCoinsBalance should return the requested account type")
}

func TestGetWithdrawableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawableAmount must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawableAmount(t.Context(), currency.LTC)
	require.NoError(t, err, "GetWithdrawableAmount must not error")
	if mockTests {
		exp := &WithdrawableAmountResponse{
			LimitAmountUSD: 595051.7,
			WithdrawableAmount: WithdrawableAmountByWallet{
				Spot: &WalletWithdrawableAmount{
					Coin:               currency.LTC,
					WithdrawableAmount: 1.5,
					AvailableBalance:   2,
				},
				Fund: &WalletWithdrawableAmount{
					Coin:               currency.LTC,
					WithdrawableAmount: 155.847,
					AvailableBalance:   155.847,
				},
				UTA: &WalletWithdrawableAmount{
					Coin:               currency.LTC,
					WithdrawableAmount: 498.0882,
					AvailableBalance:   498.0882,
				},
				Earn: &WalletWithdrawableAmount{
					Coin:               currency.LTC,
					WithdrawableAmount: 10,
					AvailableBalance:   12.5,
				},
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawableAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWithdrawableAmount should return the amounts")
}

func TestGetAssetOverview(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetOverview(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAssetOverview must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetOverview(t.Context(), &AssetOverviewRequest{ValuationCurrency: currency.USD})
	require.NoError(t, err, "GetAssetOverview must not error")
	t.Run("sub-account funding account", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the member ID names a placeholder UID, which only the fixture knows")
		}
		overview, err := e.GetAssetOverview(t.Context(), &AssetOverviewRequest{MemberID: "100000002", ValuationCurrency: currency.EUR, AccountType: "FundingAccount"})
		require.NoError(t, err, "GetAssetOverview must not error")
		exp := &AssetOverviewResponse{
			TotalEquity: 1035.2,
			List: []AssetOverviewAccount{
				{
					AccountType:       "FundingAccount",
					TotalEquity:       1035.2,
					ValuationCurrency: currency.EUR,
					SnapshotTime:      types.Time(time.UnixMilli(1790000000000)),
					CoinDetail: []AssetOverviewCoin{
						{
							Coin:   currency.BTC,
							Equity: 0.01,
						},
						{
							Coin:   currency.USDT,
							Equity: 103.5,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, overview, "GetAssetOverview should send every filter and decode the sub-account's funding account")
	})
	if mockTests {
		exp := &AssetOverviewResponse{
			TotalEquity: 41656.79,
			List: []AssetOverviewAccount{
				{
					AccountType:       "CryptoLoans",
					TotalEquity:       20429.51,
					ValuationCurrency: currency.USD,
					SnapshotTime:      types.Time(time.UnixMilli(1772449024908)),
					CoinDetail: []AssetOverviewCoin{
						{
							Coin:   currency.BTC,
							Equity: -0.00100377,
						},
						{
							Coin:   currency.USDT,
							Equity: 20383.9175,
						},
					},
				},
				{
					AccountType:       "Earn",
					TotalEquity:       20888.1,
					ValuationCurrency: currency.USD,
					SnapshotTime:      types.Time(time.UnixMilli(1772449024908)),
					Categories: []AssetOverviewCategory{
						{
							Category: "Easy Earn",
							Equity:   20888.1,
							CoinDetail: []AssetOverviewCategoryCoin{
								{
									Coin:   currency.BTC,
									Equity: 0.3,
								},
								{
									Coin:   currency.USDT,
									Equity: 200,
								},
							},
						},
					},
				},
				{
					AccountType:       "Alpha",
					TotalEquity:       339.18,
					ValuationCurrency: currency.USD,
					SnapshotTime:      types.Time(time.UnixMilli(1772449024908)),
					Categories: []AssetOverviewCategory{
						{
							Category: "farm",
							Equity:   339.18,
							CoinDetail: []AssetOverviewCategoryCoin{
								{
									Coin:   currency.USDT,
									Equity: 339.18,
									ExtendedMap: &AssetOverviewExtendedMap{
										PriceUpper: 0.25,
										PriceLower: 0.2,
										EquityUnit: currency.USD,
									},
								},
							},
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetAssetOverview should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetAssetOverview should return the accounts")
}
