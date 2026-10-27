package gateio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesCancellationMixedClocks(t *testing.T) {
	t.Parallel()
	for _, fills := range []bool{false, true} {
		t.Run(strconv.FormatBool(fills), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
			require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			pair := currency.NewBTCUSDT()
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{pair}, true), "futures pair must be enabled")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				state, reason, finish := "open", "", "0"
				if r.Method == http.MethodGet {
					state, reason, finish = "finished", "cancelled", "1735720640.77"
				}
				_, err := fmt.Fprintf(w, `{"id":1,"contract":"BTC_USDT","size":3,"left":3,"price":"60000","status":%q,"finish_as":%q,"finish_time":%s,"create_time":1735720640,"tif":"gtc"}`, state, reason, finish)
				assert.NoError(t, err, "mock futures response should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must be installed")
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			submitted, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Amount: 3, Price: 60000, TimeInForce: order.GoodTillCancel})
			require.NoError(t, err, "mock futures submission must succeed")
			stored, err := submitted.DeriveDetail(uuid.Nil())
			require.NoError(t, err, "submitted futures detail must derive")
			stored.LastUpdated = time.Unix(1735720640, 800000000)
			if !fills {
				info, err := ex.GetOrderInfo(t.Context(), "1", pair, asset.USDTMarginedFutures)
				require.NoError(t, err, "cancelled futures detail must decode")
				restStored := stored.Copy()
				require.NoError(t, restStored.UpdateOrderFromDetail(info), "REST cancellation before the local stamp must merge")
				assert.Equal(t, order.Cancelled, restStored.Status, "REST should finish the order despite clock skew")
				assert.True(t, info.LastUpdated.Before(stored.LastUpdated), "REST fixture should report a venue time preceding the local stamp")
			}
			remaining := 3
			if fills {
				for _, left := range []int{2, 1} {
					push := fmt.Appendf(nil, `{"result":[{"id":1,"contract":"BTC_USDT","size":3,"left":%d,"price":"60000","fill_price":"59900","status":"open","finish_time":0,"tif":"gtc"}]}`, left)
					got, err := ex.processFuturesOrdersPushData(push, asset.USDTMarginedFutures)
					require.NoError(t, err, "futures fill push must decode")
					require.Len(t, got, 1, "fill push must contain one order")
					require.NoError(t, stored.UpdateOrderFromDetail(&got[0]), "fill progress must merge")
					remaining = left
				}
			}
			push := fmt.Appendf(nil, `{"result":[{"id":1,"contract":"BTC_USDT","size":3,"left":%d,"price":"60000","status":"finished","finish_as":"cancelled","finish_time":1735720640,"tif":"gtc"}]}`, remaining)
			got, err := ex.processFuturesOrdersPushData(push, asset.USDTMarginedFutures)
			require.NoError(t, err, "futures cancellation push must decode")
			require.Len(t, got, 1, "cancellation push must contain one order")
			require.NoError(t, stored.UpdateOrderFromDetail(&got[0]), "coarse cancellation timestamp must not suppress the finish")
			assert.Equal(t, order.Cancelled, stored.Status, "cancellation should finish the order after submission or multiple fills")
			assert.Equal(t, float64(3-remaining), stored.ExecutedAmount, "cancellation should retain cumulative execution")
			assert.Equal(t, time.Unix(1735720640, 0), stored.CloseTime, "cancellation should preserve the venue close time")
		})
	}
}

func TestSpotAmendmentRounding(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	stored := order.Detail{}
	for i, step := range []struct{ amount, left string }{{"0.2", "0.1"}, {"0.3", "0.2"}} {
		push := fmt.Appendf(nil, `{"result":[{"id":"1","currency_pair":"BTC_USDT","account":"spot","side":"buy","type":"limit","status":"open","price":"60000","amount":%q,"left":%q,"update_time_ms":1735720640000}]}`, step.amount, step.left)
		require.NoError(t, ex.processSpotOrders(t.Context(), push), "spot order push must decode")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "spot order push must emit one batch")
		got, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
		require.True(t, ok, "spot order push must emit order details")
		require.Len(t, got, 1, "spot order batch must contain one order")
		require.NoError(t, stored.UpdateOrderFromDetail(&got[0]), "spot amendment must merge despite float subtraction")
		if i == 1 {
			assert.Equal(t, 0.3, stored.Amount, "amendment should replace total quantity")
			assert.Equal(t, 0.2, stored.RemainingAmount, "amendment should replace remaining quantity")
			assert.Equal(t, 0.1, stored.ExecutedAmount, "amendment should not decrease cumulative execution through rounding")
		}
	}
}
