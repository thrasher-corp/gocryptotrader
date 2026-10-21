package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestGetLinearSwapMarkets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, date, deliveryTime string
		expected                 time.Time
		expectedDeliveryTime     time.Time
		wantErr                  bool
	}{
		{name: "dated contract", date: "20211203", deliveryTime: "1638518400000", expectedDeliveryTime: time.UnixMilli(1638518400000), expected: time.Date(2021, time.December, 3, 0, 0, 0, 0, time.UTC)},
		{name: "perpetual", date: ""},
		{name: "malformed date", date: "20211303", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-api/v1/swap_contract_info", `{"status":"ok","data":[{"contract_code":"BTC-USDT","delivery_date":"`+tc.date+`","delivery_time":"`+tc.deliveryTime+`","settlement_date":"1638518400000"}]}`, nil)
			resp, err := h.GetLinearSwapMarkets(t.Context(), btcusdtPair, "cross", "swap", "futures")
			if tc.wantErr {
				require.Error(t, err, "GetLinearSwapMarkets must reject a malformed date")
				return
			}
			require.NoError(t, err, "GetLinearSwapMarkets must decode contract metadata")
			require.Len(t, resp, 1, "decoded markets must be returned")
			assert.Equal(t, "BTC-USDT", resp[0].ContractCode, "contract code should decode")
			assert.Equal(t, tc.expected, resp[0].DeliveryDate.Time(), "delivery date should follow the documented date or empty perpetual representation")
			assert.Equal(t, tc.expectedDeliveryTime, resp[0].DeliveryTime.Time(), "delivery time should accept milliseconds or an empty perpetual value")
			assert.Equal(t, time.UnixMilli(1638518400000), resp[0].SettlementDate.Time(), "settlement date should retain milliseconds")
		})
	}
}

func TestGetLinearSwapMarketDepth(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-ex/market/depth", `{"status":"ok","tick":{"id":123}}`, nil)
	resp, err := h.GetLinearSwapMarketDepth(t.Context(), btcusdtPair, "step0")
	require.NoError(t, err, "GetLinearSwapMarketDepth must not error")
	assert.Equal(t, uint64(123), resp.Tick.ID, "depth ID should decode")
}

func TestGetLinearSwapMarketOverview(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-ex/market/detail/merged", `{"status":"ok","ch":"market.BTC-USDT.detail","tick":{"id":123}}`, nil)
	resp, err := h.GetLinearSwapMarketOverview(t.Context(), btcusdtPair)
	require.NoError(t, err, "GetLinearSwapMarketOverview must not error")
	assert.Equal(t, uint64(123), resp.Tick.ID, "overview ID should decode")
}

func TestGetLinearSwapKlineData(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-ex/market/history/kline", `{"status":"ok","data":[{"id":1604312615,"close":1}]}`, nil)
	resp, err := h.GetLinearSwapKlineData(t.Context(), btcusdtPair, "1min", 10, time.Time{}, time.Time{})
	require.NoError(t, err, "GetLinearSwapKlineData must not error")
	require.Len(t, resp.Data, 1, "decoded candles must be returned")
	assert.Equal(t, float64(1), resp.Data[0].Close, "close price should decode")
	_, err = h.GetLinearSwapKlineData(t.Context(), btcusdtPair, "invalid", 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetLinearSwapKlineData must reject an invalid period")
	_, err = h.GetLinearSwapKlineData(t.Context(), btcusdtPair, "1min", 10, time.Now(), time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetLinearSwapKlineData must reject a half-open interval")
}

func TestGetLinearSwapBatchTrades(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-ex/market/history/trade", `{"status":"ok","id":123,"data":[{"id":1,"data":[{"id":2,"price":10}]}]}`, nil)
	resp, err := h.GetLinearSwapBatchTrades(t.Context(), btcusdtPair, 10)
	require.NoError(t, err, "GetLinearSwapBatchTrades must not error")
	assert.Equal(t, uint64(123), resp.ID, "batch ID should decode")
}
