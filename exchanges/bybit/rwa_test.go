package bybit

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetRealWorldAssetProductList(t *testing.T) {
	t.Parallel()
	// Signed requests return account specific fields, so the method signs whenever credentials are set
	unsigned := testInstance(t)
	unsigned.SetCredentials(nil)
	for _, tc := range []struct {
		name string
		ex   *Exchange
	}{
		{name: "credentials", ex: e},
		{name: "no credentials", ex: unsigned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetRealWorldAssetProductList(t.Context(), currency.USDC)
			require.NoError(t, err, "GetRealWorldAssetProductList must not error")
			if mockTests {
				exp := &RealWorldAssetProductListResponse{
					List: []RealWorldAssetProduct{
						{
							ProductID:             3,
							Coin:                  currency.USDC,
							AssetSymbol:           "OPAL",
							Manager:               "blackOpal",
							BaseAPR:               0.1105,
							BonusAPR:              0.02,
							SavingType:            "Flexible",
							NAV:                   1.016426,
							MinStakeAmount:        500,
							MaxStakeAmount:        20000000,
							UserMaxAmount:         20000000,
							UserQuota:             19999500,
							MinRedeemShare:        1,
							RedeemFeeRate:         0.001,
							ExternalLink:          "https://example.com/opal",
							AmountPrecision:       4,
							SharePrecision:        6,
							RedeemDurationMinutes: 7200,
						},
						{
							ProductID:             2,
							Coin:                  currency.USDC,
							AssetSymbol:           "CMIGB",
							Manager:               "cmbi",
							BaseAPR:               -0.1227,
							SavingType:            "Fixed",
							DurationInDays:        90,
							NAV:                   0.984478,
							MinStakeAmount:        2000,
							MinRedeemShare:        1,
							SubscriptionFee:       0.001,
							AmountPrecision:       4,
							SharePrecision:        6,
							RedeemDurationMinutes: 5760,
						},
					},
				}
				assert.Equal(t, exp, result, "GetRealWorldAssetProductList should decode every field")
				return
			}
			require.NotEmpty(t, result.List, "GetRealWorldAssetProductList must return products")
			assert.True(t, result.List[0].Coin.Equal(currency.USDC), "GetRealWorldAssetProductList should return products settled in the coin")
		})
	}
}

func TestPlaceRealWorldAssetOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceRealWorldAssetOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceRealWorldAssetOrder must reject a nil request")
	req := &PlaceRealWorldAssetOrderRequest{}
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "PlaceRealWorldAssetOrder must reject an empty product ID")
	req.ProductID = 3
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceRealWorldAssetOrder must reject an empty order type")
	req.OrderType = "Stake"
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceRealWorldAssetOrder must reject an empty coin")
	req.Coin = currency.USDC
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "PlaceRealWorldAssetOrder must reject an empty order link ID")
	req.OrderLinkID = "my-stake-001"
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRealWorldAssetOrder must reject an order without a stake amount or redeem shares")
	// Bybit ignores the other order type's amount, so it cannot stand in for the order's own
	req.RedeemShares = 97.560975
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRealWorldAssetOrder must reject a stake with only redeem shares")
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), &PlaceRealWorldAssetOrderRequest{ProductID: 3, OrderType: "Redeem", Coin: currency.USDC, OrderLinkID: "my-redeem-001", StakeAmount: 500})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRealWorldAssetOrder must reject a redemption with only a stake amount")
	_, err = e.PlaceRealWorldAssetOrder(t.Context(), &PlaceRealWorldAssetOrderRequest{ProductID: 3, OrderType: "Buy", Coin: currency.USDC, OrderLinkID: "my-buy-001", StakeAmount: 500})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceRealWorldAssetOrder must reject an order type other than Stake and Redeem")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceRealWorldAssetOrderRequest
		exp  *PlaceRealWorldAssetOrderResponse
	}{
		{
			name: "stake",
			req:  &PlaceRealWorldAssetOrderRequest{ProductID: 3, OrderType: "Stake", Coin: currency.USDC, OrderLinkID: "my-stake-001", StakeAmount: 500, AccountType: "FUND"},
			exp: &PlaceRealWorldAssetOrderResponse{
				OrderID:     "550e8400-e29b-41d4-a716-446655440000",
				OrderLinkID: "my-stake-001",
			},
		},
		{
			name: "redeem",
			req:  &PlaceRealWorldAssetOrderRequest{ProductID: 3, OrderType: "Redeem", Coin: currency.USDC, OrderLinkID: "my-redeem-001", RedeemShares: 97.560975, AccountType: "UNIFIED"},
			exp: &PlaceRealWorldAssetOrderResponse{
				OrderID:     "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
				OrderLinkID: "my-redeem-001",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceRealWorldAssetOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceRealWorldAssetOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceRealWorldAssetOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceRealWorldAssetOrder should return an order ID")
		})
	}
}

func TestGetRealWorldAssetPositionList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRealWorldAssetPositionList(t.Context())
	require.NoError(t, err, "GetRealWorldAssetPositionList must not error")
	if mockTests {
		exp := &RealWorldAssetPositionListResponse{
			List: []RealWorldAssetPosition{
				{
					ProductID:             3,
					Coin:                  currency.USDC,
					AssetSymbol:           "OPAL",
					EffectiveShare:        100,
					ProcessingStakeAmount: 50,
					ProcessingRedeemShare: 10,
					BonusEarned:           1.25,
					NAV:                   1.016426,
					HoldAmount:            101.6426,
				},
				{
					ProductID:      2,
					Coin:           currency.USDC,
					AssetSymbol:    "CMIGB",
					EffectiveShare: 2000,
					NAV:            0.984478,
					HoldAmount:     1968.956,
					DurationInDays: 90,
				},
			},
		}
		assert.Equal(t, exp, result, "GetRealWorldAssetPositionList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetRealWorldAssetPositionList should return a result")
}

func TestGetRealWorldAssetOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRealWorldAssetOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRealWorldAssetOrderList must reject a nil request")
	_, err = e.GetRealWorldAssetOrderList(t.Context(), &RealWorldAssetOrderListRequest{StartTime: time.Unix(1791331200, 0), EndTime: time.Unix(1791072000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRealWorldAssetOrderList must reject a start time after the end time")

	start, end := time.Unix(1791072000, 0), time.Unix(1791331200, 0)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// Orders older than 180 days cannot be queried
		end = time.Now()
		start = end.AddDate(0, 0, -3)
	}
	for _, tc := range []struct {
		name string
		req  *RealWorldAssetOrderListRequest
		exp  *RealWorldAssetOrderListResponse
	}{
		{
			name: "list",
			req:  &RealWorldAssetOrderListRequest{OrderType: "Stake", ProductID: 3, StartTime: start, EndTime: end, Limit: 20},
			exp: &RealWorldAssetOrderListResponse{
				List: []RealWorldAssetOrder{
					{
						OrderID:       "550e8400-e29b-41d4-a716-446655440000",
						OrderLinkID:   "my-stake-001",
						OrderType:     "Stake",
						ProductID:     3,
						Coin:          currency.USDC,
						StakeAmount:   500,
						Status:        "Success",
						AccountType:   "FUND",
						CreatedTime:   types.Time(time.Unix(1791072600, 0)),
						UpdatedTime:   types.Time(time.Unix(1791076200, 0)),
						SettledShares: 491.918,
					},
				},
				NextPageCursor: "eyJpZCI6MTIzNDU2fQ==",
			},
		},
		{
			name: "single order",
			req:  &RealWorldAssetOrderListRequest{OrderID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8", OrderLinkID: "my-redeem-001"},
			exp: &RealWorldAssetOrderListResponse{
				List: []RealWorldAssetOrder{
					{
						OrderID:       "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
						OrderLinkID:   "my-redeem-001",
						OrderType:     "Redeem",
						ProductID:     3,
						Coin:          currency.USDC,
						RedeemShares:  97.560975,
						Status:        "Success",
						AccountType:   "UNIFIED",
						CreatedTime:   types.Time(time.Unix(1791159000, 0)),
						UpdatedTime:   types.Time(time.Unix(1791591000, 0)),
						SettledAmount: 99.1638,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRealWorldAssetOrderList(t.Context(), tc.req)
			require.NoError(t, err, "GetRealWorldAssetOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRealWorldAssetOrderList should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetRealWorldAssetOrderList(t.Context(), tc.req)
			assert.NoError(t, err, "GetRealWorldAssetOrderList should not error for the next page")
		})
	}
}

func TestGetRealWorldAssetNAVChart(t *testing.T) {
	t.Parallel()
	_, err := e.GetRealWorldAssetNAVChart(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRealWorldAssetNAVChart must reject a nil request")
	_, err = e.GetRealWorldAssetNAVChart(t.Context(), &RealWorldAssetNAVChartRequest{})
	require.ErrorIs(t, err, errProductIDRequired, "GetRealWorldAssetNAVChart must reject an empty product ID")
	_, err = e.GetRealWorldAssetNAVChart(t.Context(), &RealWorldAssetNAVChartRequest{ProductID: 3, StartTime: time.Unix(1791331200, 0), EndTime: time.Unix(1791072000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRealWorldAssetNAVChart must reject a start time after the end time")

	// Bybit answers a window that starts and ends on one day with that day's NAV
	var gotQuery url.Values
	navEx := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"","result":{"productId":"3","list":[{"date":"2026-10-04","nav":"1.015201"}]},"retExtInfo":{},"time":1791100000000}`)
		assert.NoError(t, err, "writing the response should not error")
	})
	day := time.Unix(1791072000, 0)
	nav, err := navEx.GetRealWorldAssetNAVChart(t.Context(), &RealWorldAssetNAVChartRequest{ProductID: 3, StartTime: day, EndTime: day})
	require.NoError(t, err, "GetRealWorldAssetNAVChart must send a window that starts and ends on one day")
	assert.Equal(t, url.Values{"productId": {"3"}, "startTime": {"1791072000"}, "endTime": {"1791072000"}}, gotQuery, "GetRealWorldAssetNAVChart should send both bounds")
	assert.Len(t, nav.List, 1, "GetRealWorldAssetNAVChart should return the day's NAV")

	req := &RealWorldAssetNAVChartRequest{ProductID: 3, StartTime: time.Unix(1791072000, 0), EndTime: time.Unix(1791331200, 0)}
	if !mockTests {
		products, err := e.GetRealWorldAssetProductList(t.Context(), currency.EMPTYCODE)
		require.NoError(t, err, "GetRealWorldAssetProductList must not error")
		require.NotEmpty(t, products.List, "GetRealWorldAssetProductList must return products")
		req.ProductID, err = strconv.ParseUint(products.List[0].ProductID.String(), 10, 64)
		require.NoError(t, err, "GetRealWorldAssetProductList must return an unsigned integer product ID")
		req.EndTime = time.Now()
		req.StartTime = req.EndTime.AddDate(0, 0, -3)
	}
	result, err := e.GetRealWorldAssetNAVChart(t.Context(), req)
	require.NoError(t, err, "GetRealWorldAssetNAVChart must not error")
	if mockTests {
		exp := &RealWorldAssetNAVChartResponse{
			ProductID: 3,
			List: []RealWorldAssetNAV{
				{
					Date: "2026-10-04",
					NAV:  1.015201,
				},
				{
					Date: "2026-10-05",
					NAV:  1.015575,
				},
			},
		}
		assert.Equal(t, exp, result, "GetRealWorldAssetNAVChart should decode every field")
		return
	}
	assert.Equal(t, strconv.FormatUint(req.ProductID, 10), result.ProductID.String(), "GetRealWorldAssetNAVChart should return the product's NAV")
	assert.NotEmpty(t, result.List, "GetRealWorldAssetNAVChart should return NAVs")
}
