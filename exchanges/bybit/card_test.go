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

func TestGetCardAssetRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetCardAssetRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCardAssetRecords must reject a nil request")
	_, err = e.GetCardAssetRecords(t.Context(), &CardAssetRecordsRequest{})
	require.ErrorIs(t, err, errCardQueryTypeRequired, "GetCardAssetRecords must require a query type without a transaction ID or order number")
	for _, req := range []*CardAssetRecordsRequest{
		{Type: "SIDE_QUERY_FINANCIAL", TransactionID: "txn-0001"},
		{Type: "SIDE_QUERY_FINANCIAL", OrderNumber: "order-0001"},
		{Type: "SIDE_QUERY_REFUND", TransactionID: "txn-0001"},
		{Type: "SIDE_QUERY_REFUND", OrderNumber: "order-0001"},
	} {
		_, err = e.GetCardAssetRecords(t.Context(), req)
		require.ErrorIsf(t, err, errCardTransactionIDAndOrderNotSet, "GetCardAssetRecords must require a transaction ID and order number for %s", req.Type)
	}
	queried := restReplyTestInstance(t, map[string]string{"/v5/card/transaction/query-asset-records": `{"retCode":0,"retMsg":"success","result":{},"retExtInfo":{},"time":1760000000000}`})
	for _, queryType := range []string{"SIDE_QUERY_FINANCIAL", "SIDE_QUERY_REFUND"} {
		_, err = queried.GetCardAssetRecords(t.Context(), &CardAssetRecordsRequest{Type: queryType, TransactionID: "txn-0001", OrderNumber: "order-0001"})
		require.NoErrorf(t, err, "GetCardAssetRecords must accept %s with a transaction ID and order number", queryType)
	}
	_, err = e.GetCardAssetRecords(t.Context(), &CardAssetRecordsRequest{
		Type: "SIDE_QUERY_AUTH_ALL", CreateBeginTime: time.UnixMilli(1672329600000), CreateEndTime: time.UnixMilli(1672156800000),
	})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCardAssetRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *CardAssetRecordsRequest
		exp  *CardAssetRecordsResponse
	}{
		{
			name: "filters",
			req: &CardAssetRecordsRequest{
				StatusCode: "1", Limit: 10, Page: 1, PAN4: "1234", CreateBeginTime: time.UnixMilli(1672156800000),
				CreateEndTime: time.UnixMilli(1672329600000), MerchantName: "Amazon", Type: "SIDE_QUERY_FINANCIAL_ALL",
				CardToken: "card-token-0001",
			},
			exp: &CardAssetRecordsResponse{
				PageSize:   10,
				PageNumber: 1,
				TotalCount: 2,
				Data: []CardAssetRecord{
					{
						PAN4:                        "1234",
						PAN6:                        "456789",
						TradeStatus:                 "1",
						Side:                        "3",
						BasicAmount:                 101.5,
						BasicCurrency:               currency.USD,
						TransactionAmount:           100,
						TransactionCurrency:         currency.USD,
						TransactionCreate:           types.Time(time.UnixMilli(1672211918471)),
						MerchantCountry:             "US",
						MerchantCity:                "New York",
						MerchantName:                "Amazon",
						TransactionID:               "TXN20230101001",
						TotalFees:                   1.5,
						UID:                         100000001,
						TransactionCurrencyAmount:   100,
						FXPad:                       0.35,
						InterchangeFee:              0.5,
						BillAmount:                  101.5,
						PaidAmount:                  101.5,
						PaidCurrency:                currency.USDT,
						BonusAmount:                 1,
						ForeignTransactionFee:       0.5,
						TotalTax:                    1.5,
						Status:                      "1",
						OrderNumber:                 "ORD20230101001",
						MCCCode:                     "5411",
						MerchantCategoryDescription: "Grocery Stores",
					},
					{
						PAN4:                        "5678",
						PAN6:                        "456789",
						TradeStatus:                 "2",
						Side:                        "13",
						BasicAmount:                 52.5,
						BasicCurrency:               currency.EUR,
						TransactionAmount:           50,
						TransactionCurrency:         currency.EUR,
						TransactionCreate:           types.Time(time.UnixMilli(1672298318471)),
						MerchantCountry:             "DE",
						MerchantCity:                "Berlin",
						MerchantName:                "Sparkasse ATM",
						TransactionID:               "TXN20230102001",
						DeclinedReason:              "Insufficient balance",
						TotalFees:                   2.5,
						UID:                         100000001,
						TransactionCurrencyAmount:   50,
						BillAmount:                  52.5,
						PaidCurrency:                currency.USDT,
						PaidFiat:                    52.5,
						WithdrawalFee:               2.5,
						Status:                      "2",
						OrderNumber:                 "ORD20230102001",
						MCCCode:                     "6011",
						MerchantCategoryDescription: "Automated Cash Disbursements",
					},
				},
			},
		},
		{
			name: "transaction",
			req:  &CardAssetRecordsRequest{TransactionID: "TXN20230101001", OrderNumber: "ORD20230101001"},
			exp: &CardAssetRecordsResponse{
				PageSize:   100,
				PageNumber: 1,
				TotalCount: 1,
				Data: []CardAssetRecord{
					{
						PAN4:                        "1234",
						PAN6:                        "456789",
						TradeStatus:                 "1",
						Side:                        "3",
						BasicAmount:                 101.5,
						BasicCurrency:               currency.USD,
						TransactionAmount:           100,
						TransactionCurrency:         currency.USD,
						TransactionCreate:           types.Time(time.UnixMilli(1672211918471)),
						MerchantCountry:             "US",
						MerchantCity:                "New York",
						MerchantName:                "Amazon",
						TransactionID:               "TXN20230101001",
						TotalFees:                   1.5,
						UID:                         100000001,
						TransactionCurrencyAmount:   100,
						FXPad:                       0.35,
						InterchangeFee:              0.5,
						BillAmount:                  101.5,
						PaidAmount:                  101.5,
						PaidCurrency:                currency.USDT,
						BonusAmount:                 1,
						ForeignTransactionFee:       0.5,
						TotalTax:                    1.5,
						Status:                      "1",
						OrderNumber:                 "ORD20230101001",
						MCCCode:                     "5411",
						MerchantCategoryDescription: "Grocery Stores",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCardAssetRecords(t.Context(), tc.req)
			require.NoError(t, err, "GetCardAssetRecords must not error")
			require.NotNil(t, result, "GetCardAssetRecords must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCardAssetRecords should decode every field")
			}
		})
	}
}

func TestGetCardPointBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCardPointBalance(t.Context())
	require.NoError(t, err, "GetCardPointBalance must not error")
	require.NotNil(t, result, "GetCardPointBalance must return a result")
	if mockTests {
		exp := &CardPointBalanceResponse{
			AccountID:        "100001",
			AvailablePoint:   5000,
			PendingPoint:     200,
			Status:           "active",
			UpdateTime:       types.Time(time.UnixMilli(1672211918471)),
			SettlementPeriod: 30,
		}
		assert.Equal(t, exp, result, "GetCardPointBalance should decode every field")
	}
}

func TestGetCardPointRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetCardPointRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCardPointRecords must reject a nil request")
	_, err = e.GetCardPointRecords(t.Context(), &CardPointRecordsRequest{StartTime: time.UnixMilli(1672329600000), EndTime: time.UnixMilli(1672156800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCardPointRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCardPointRecords(t.Context(), &CardPointRecordsRequest{
		Type: "CASHBACK", PageSize: 10, PageNumber: 1, StartTime: time.UnixMilli(1672156800000), EndTime: time.UnixMilli(1672329600000),
		OutOrderID: "ORD20230101001", BusinessID: "BIZ20230101001", BusinessTransactionID: "TXN20230101001", Side: "1",
	})
	require.NoError(t, err, "GetCardPointRecords must not error")
	require.NotNil(t, result, "GetCardPointRecords must return a result")
	if mockTests {
		exp := &CardPointRecordsResponse{
			PageSize:   10,
			PageNumber: 1,
			TotalCount: 2,
			Data: []CardPointRecord{
				{
					OutOrderID:                  "ORD20230101001",
					Point:                       100,
					Side:                        "1",
					Type:                        "CASHBACK",
					SubType:                     "CARD_SPEND",
					CreateTime:                  types.Time(time.UnixMilli(1672211918471)),
					UpdateTime:                  types.Time(time.UnixMilli(1672211919471)),
					BusinessID:                  "BIZ20230101001",
					BusinessTransactionID:       "TXN20230101001",
					TransactionDate:             "2023-01-01 12:00:00",
					TransactionID:               "TXN20230101001",
					TransactionAmount:           100,
					BasicCurrency:               currency.USD,
					MerchantCategoryDescription: "Grocery Stores",
					MerchantName:                "Amazon",
					MerchantCountry:             "US",
					MerchantCity:                "New York",
					PAN4:                        "1234",
					TransactionCurrencyAmount:   100,
				},
				{
					OutOrderID:                  "ORD20230101002",
					Point:                       40,
					Side:                        "1",
					Type:                        "CASHBACK",
					SubType:                     "CARD_SPEND",
					CreateTime:                  types.Time(time.UnixMilli(1672215518471)),
					UpdateTime:                  types.Time(time.UnixMilli(1672215519471)),
					BusinessID:                  "BIZ20230101002",
					BusinessTransactionID:       "TXN20230101002",
					TransactionDate:             "2023-01-01 13:00:00",
					TransactionID:               "TXN20230101002",
					TransactionAmount:           40,
					BasicCurrency:               currency.USD,
					MerchantCategoryDescription: "Eating Places",
					MerchantName:                "Cafe",
					MerchantCountry:             "US",
					MerchantCity:                "New York",
					PAN4:                        "1234",
					PayFiatAmount:               40,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCardPointRecords should decode every field")
	}
}

func TestGetCardTierInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCardTierInfo(t.Context())
	require.NoError(t, err, "GetCardTierInfo must not error")
	require.NotNil(t, result, "GetCardTierInfo must return a result")
	if mockTests {
		exp := &CardTierInfoResponse{
			UsedLimit:    10,
			Limit:        500,
			Unit:         "1",
			Tier:         "GOLD",
			AutoCashback: true,
		}
		assert.Equal(t, exp, result, "GetCardTierInfo should decode every field")
	}
}

func TestGetCardMallItemList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCardMallItemList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCardMallItemList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ascending := true
	result, err := e.GetCardMallItemList(t.Context(), &CardMallItemListRequest{
		PageNumber: 1, PageSize: 10, ItemType: 1, ItemBusinessType: 1, OrderBy: 3, Ascending: &ascending, Source: 1,
	})
	require.NoError(t, err, "GetCardMallItemList must not error")
	require.NotNil(t, result, "GetCardMallItemList must return a result")
	if mockTests {
		exp := &CardMallItemListResponse{
			PageNumber: 1,
			PageSize:   10,
			TotalCount: 5,
			Data: []CardMallItem{
				{
					ItemID:           "ITEM001",
					ItemName:         "Amazon Gift Card $10",
					Priority:         1,
					OnTime:           types.Time(time.UnixMilli(1672211918471)),
					OffTime:          types.Time(time.Unix(1704000000, 0)),
					Price:            5000,
					DiscountPrice:    4500,
					TotalNumber:      1000,
					RedeemNumber:     200,
					PicturePath:      "/images/item001.png",
					Currency:         currency.NewCode("POINT"),
					CurrencyType:     3,
					ItemType:         1,
					ItemBusinessType: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCardMallItemList should decode every field")
	}
}

func TestGetCardCashbackDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCardCashbackDetail(t.Context(), "")
	require.ErrorIs(t, err, errCardBusinessTransactionIDNotSet, "GetCardCashbackDetail must require a business transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCardCashbackDetail(t.Context(), "TXN20230101001")
	require.NoError(t, err, "GetCardCashbackDetail must not error")
	require.NotNil(t, result, "GetCardCashbackDetail must return a result")
	if mockTests {
		exp := &CardCashbackDetailResponse{
			Points:                500,
			Amount:                1,
			Currency:              currency.USDT,
			CurrencyType:          "CRYPTO",
			CreateTime:            "2023-01-01 12:00:00",
			BusinessTransactionID: "TXN20230101001",
			SourceID:              12345,
			SourceCode:            "ORD20230101001",
			OrderStatus:           3,
			OrderSubStatus:        2,
			OrderShowStatus:       "ORDER_FAIL",
			FailedBusinessCode:    "POINT_NOT_ENOUGH",
		}
		assert.Equal(t, exp, result, "GetCardCashbackDetail should decode every field")
	}
}
