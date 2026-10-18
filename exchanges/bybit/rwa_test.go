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

func TestGetRWAProductList(t *testing.T) {
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
			result, err := tc.ex.GetRWAProductList(t.Context(), currency.USDC)
			require.NoError(t, err, "GetRWAProductList must not error")
			if mockTests {
				exp := &RWAProductListResponse{
					List: []RWAProduct{
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
							Duration:              90,
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
				assert.Equal(t, exp, result, "GetRWAProductList should decode every field")
				return
			}
			require.NotEmpty(t, result.List, "GetRWAProductList must return products")
			assert.True(t, result.List[0].Coin.Equal(currency.USDC), "GetRWAProductList should return products settled in the coin")
		})
	}
}

func TestPlaceRWAOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceRWAOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceRWAOrder must reject a nil request")
	req := &PlaceRWAOrderRequest{}
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "PlaceRWAOrder must reject an empty product ID")
	req.ProductID = 3
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceRWAOrder must reject an empty order type")
	req.OrderType = "Stake"
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceRWAOrder must reject an empty coin")
	req.Coin = currency.USDC
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "PlaceRWAOrder must reject an empty order link ID")
	req.OrderLinkID = "my-stake-001"
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRWAOrder must reject an order without a stake amount or redeem shares")
	// Bybit ignores the other order type's amount, so it cannot stand in for the order's own
	req.RedeemShares = 97.560975
	_, err = e.PlaceRWAOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRWAOrder must reject a stake with only redeem shares")
	_, err = e.PlaceRWAOrder(t.Context(), &PlaceRWAOrderRequest{ProductID: 3, OrderType: "Redeem", Coin: currency.USDC, OrderLinkID: "my-redeem-001", StakeAmount: 500})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceRWAOrder must reject a redemption with only a stake amount")
	_, err = e.PlaceRWAOrder(t.Context(), &PlaceRWAOrderRequest{ProductID: 3, OrderType: "Buy", Coin: currency.USDC, OrderLinkID: "my-buy-001", StakeAmount: 500})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceRWAOrder must reject an order type other than Stake and Redeem")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceRWAOrderRequest
		exp  *PlaceRWAOrderResponse
	}{
		{
			name: "stake",
			req:  &PlaceRWAOrderRequest{ProductID: 3, OrderType: "Stake", Coin: currency.USDC, OrderLinkID: "my-stake-001", StakeAmount: 500, AccountType: "FUND"},
			exp: &PlaceRWAOrderResponse{
				OrderID:     "550e8400-e29b-41d4-a716-446655440000",
				OrderLinkID: "my-stake-001",
			},
		},
		{
			name: "redeem",
			req:  &PlaceRWAOrderRequest{ProductID: 3, OrderType: "Redeem", Coin: currency.USDC, OrderLinkID: "my-redeem-001", RedeemShares: 97.560975, AccountType: "UNIFIED"},
			exp: &PlaceRWAOrderResponse{
				OrderID:     "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
				OrderLinkID: "my-redeem-001",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceRWAOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceRWAOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceRWAOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceRWAOrder should return an order ID")
		})
	}
}

func TestGetRWAPositionList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRWAPositionList(t.Context())
	require.NoError(t, err, "GetRWAPositionList must not error")
	if mockTests {
		exp := &RWAPositionListResponse{
			List: []RWAPosition{
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
					Duration:       90,
				},
			},
		}
		assert.Equal(t, exp, result, "GetRWAPositionList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetRWAPositionList should return a result")
}

func TestGetRWAOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRWAOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRWAOrderList must reject a nil request")
	_, err = e.GetRWAOrderList(t.Context(), &RWAOrderListRequest{StartTime: time.Unix(1791331200, 0), EndTime: time.Unix(1791072000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRWAOrderList must reject a start time after the end time")

	start, end := time.Unix(1791072000, 0), time.Unix(1791331200, 0)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// Orders older than 180 days cannot be queried
		end = time.Now()
		start = end.AddDate(0, 0, -3)
	}
	for _, tc := range []struct {
		name string
		req  *RWAOrderListRequest
		exp  *RWAOrderListResponse
	}{
		{
			name: "list",
			req:  &RWAOrderListRequest{OrderType: "Stake", ProductID: 3, StartTime: start, EndTime: end, Limit: 20},
			exp: &RWAOrderListResponse{
				List: []RWAOrder{
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
			req:  &RWAOrderListRequest{OrderID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8", OrderLinkID: "my-redeem-001"},
			exp: &RWAOrderListResponse{
				List: []RWAOrder{
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
			result, err := e.GetRWAOrderList(t.Context(), tc.req)
			require.NoError(t, err, "GetRWAOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRWAOrderList should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetRWAOrderList(t.Context(), tc.req)
			assert.NoError(t, err, "GetRWAOrderList should not error for the next page")
		})
	}
}

func TestGetRWANAVChart(t *testing.T) {
	t.Parallel()
	_, err := e.GetRWANAVChart(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRWANAVChart must reject a nil request")
	_, err = e.GetRWANAVChart(t.Context(), &RWANAVChartRequest{})
	require.ErrorIs(t, err, errProductIDRequired, "GetRWANAVChart must reject an empty product ID")
	_, err = e.GetRWANAVChart(t.Context(), &RWANAVChartRequest{ProductID: 3, StartTime: time.Unix(1791331200, 0), EndTime: time.Unix(1791072000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRWANAVChart must reject a start time after the end time")

	// Bybit answers a window that starts and ends on one day with that day's NAV
	var gotQuery url.Values
	navEx := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"","result":{"productId":"3","list":[{"date":"2026-10-04","nav":"1.015201"}]},"retExtInfo":{},"time":1791100000000}`)
		assert.NoError(t, err, "writing the response should not error")
	})
	day := time.Unix(1791072000, 0)
	nav, err := navEx.GetRWANAVChart(t.Context(), &RWANAVChartRequest{ProductID: 3, StartTime: day, EndTime: day})
	require.NoError(t, err, "GetRWANAVChart must send a window that starts and ends on one day")
	assert.Equal(t, url.Values{"productId": {"3"}, "startTime": {"1791072000"}, "endTime": {"1791072000"}}, gotQuery, "GetRWANAVChart should send both bounds")
	assert.Len(t, nav.List, 1, "GetRWANAVChart should return the day's NAV")

	req := &RWANAVChartRequest{ProductID: 3, StartTime: time.Unix(1791072000, 0), EndTime: time.Unix(1791331200, 0)}
	if !mockTests {
		products, err := e.GetRWAProductList(t.Context(), currency.EMPTYCODE)
		require.NoError(t, err, "GetRWAProductList must not error")
		require.NotEmpty(t, products.List, "GetRWAProductList must return products")
		req.ProductID, err = strconv.ParseUint(products.List[0].ProductID.String(), 10, 64)
		require.NoError(t, err, "GetRWAProductList must return an unsigned integer product ID")
		req.EndTime = time.Now()
		req.StartTime = req.EndTime.AddDate(0, 0, -3)
	}
	result, err := e.GetRWANAVChart(t.Context(), req)
	require.NoError(t, err, "GetRWANAVChart must not error")
	if mockTests {
		exp := &RWANAVChartResponse{
			ProductID: 3,
			List: []RWANAV{
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
		assert.Equal(t, exp, result, "GetRWANAVChart should decode every field")
		return
	}
	assert.Equal(t, strconv.FormatUint(req.ProductID, 10), result.ProductID.String(), "GetRWANAVChart should return the product's NAV")
	assert.NotEmpty(t, result.List, "GetRWANAVChart should return NAVs")
}
