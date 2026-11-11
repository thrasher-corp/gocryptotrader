package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name         string
		path         string
		statusCode   int
		body         string
		authenticate bool
		nilResult    bool
		expected     []error
	}
	for _, tc := range []testCase{
		{name: "negative V5 code", path: "/v5/test", statusCode: http.StatusOK, authenticate: true, body: `{"code":-1,"message":"negative failure","data":[]}`, expected: []error{request.ErrAuthRequestFailed}},
		{name: "V3 failure", path: "/api/v3/contract_hisorders", statusCode: http.StatusOK, authenticate: true, body: `{"code":403,"msg":"incorrect key","data":""}`, expected: []error{request.ErrAuthRequestFailed}},
		{name: "V3 success", path: "/api/v3/contract_hisorders", statusCode: http.StatusOK, authenticate: true, body: `{"code":200,"msg":"","data":[]}`},

		{
			name:       "authentication required",
			statusCode: http.StatusOK,
			expected:   []error{exchange.ErrAuthenticationSupportNotEnabled},
		},
		{
			name:         "empty data",
			statusCode:   http.StatusOK,
			body:         `{"code":200,"msg":"","data":"","ts":1604312615051}`,
			authenticate: true,
		},
		{
			name:         "no content without result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			nilResult:    true,
		},
		{
			name:         "no content with result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			expected:     []error{errExpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "unexpected response without result",
			statusCode:   http.StatusOK,
			body:         `{"code":200,"msg":"","data":[]}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{errUnexpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "API error without result",
			statusCode:   http.StatusOK,
			body:         `{"status":"error","err_code":1001,"err_msg":"invalid request"}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{request.ErrAuthRequestFailed},
		},
		{
			name:         "API error without message",
			statusCode:   http.StatusOK,
			body:         `{"status":"error","err_code":1001}`,
			authenticate: true,
			expected:     []error{request.ErrAuthRequestFailed},
		},
		{
			name:         "HTTP forbidden",
			statusCode:   http.StatusForbidden,
			body:         `{"status":"error","err_msg":"forbidden"}`,
			authenticate: true,
			expected:     []error{request.ErrBadStatus, request.ErrAuthRequestFailed},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := tc.path
			if path == "" {
				path = "/private"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, path, r.URL.Path, "request path should match")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request content type should match")
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			if tc.authenticate {
				h.API.AuthenticatedSupport = true
				h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			}
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
			response := new(FuturesFinancialRecordsResponse)
			var result any = response
			if tc.nilResult {
				result = nil
			}
			err := h.FuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, path, nil, nil, result)
			for _, expected := range tc.expected {
				assert.ErrorIs(t, err, expected, "FuturesAuthenticatedHTTPRequest should return the expected error")
			}
			if len(tc.expected) != 0 {
				require.Error(t, err, "FuturesAuthenticatedHTTPRequest must return an error")
				return
			}
			require.NoError(t, err, "FuturesAuthenticatedHTTPRequest must not error")
			if result != nil {
				assert.Empty(t, response.FinancialRecords.Records, "empty data should produce an empty result")
			}
		})
	}
}

func TestAddV3HistoryTimeRange(t *testing.T) {
	t.Parallel()
	req := make(map[string]any)
	err := addV3HistoryTimeRange(req, 2)
	require.NoError(t, err, "addV3HistoryTimeRange must accept a two-day lookback")
	startTime, ok := req["start_time"].(int64)
	require.True(t, ok, "start time must be set")
	endTime, ok := req["end_time"].(int64)
	require.True(t, ok, "end time must be set")
	assert.Greater(t, endTime, startTime, "end time should be after start time")
	assert.InDelta(t, int64(48*time.Hour/time.Millisecond), endTime-startTime, float64(time.Minute/time.Millisecond), "lookback should cover 48 hours")

	emptyReq := make(map[string]any)
	require.NoError(t, addV3HistoryTimeRange(emptyReq, 0), "addV3HistoryTimeRange must accept a zero lookback")
	assert.Empty(t, emptyReq, "zero lookback should not set a time range")
	require.ErrorIs(t, addV3HistoryTimeRange(make(map[string]any), 3), errInvalidLookbackDays, "addV3HistoryTimeRange must reject lookbacks over two days")
}

func TestFormatFuturesOrderStatuses(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name        string
		statuses    []order.Status
		expected    string
		expectedErr error
	}
	for _, tc := range []testCase{
		{name: "empty", expected: "0"},
		{name: "single", statuses: []order.Status{order.Active}, expected: "3"},
		{name: "multiple", statuses: []order.Status{order.PartiallyCancelled, order.Active}, expected: "5,3"},
		{name: "invalid", statuses: []order.Status{order.Rejected}, expectedErr: errInvalidOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := formatFuturesOrderStatuses(tc.statuses)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "formatFuturesOrderStatuses must return the expected error")
				assert.Contains(t, err.Error(), tc.statuses[0].String(), "error should include the invalid status")
				return
			}
			require.NoError(t, err, "formatFuturesOrderStatuses must not error")
			assert.Equal(t, tc.expected, actual, "formatted statuses should match")
		})
	}
}

func TestFuturesGetContractInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetContractInfo(t.Context(), "", "", currency.EMPTYPAIR)
		require.NoError(t, err, "live FuturesGetContractInfo request must succeed")
		require.NotNil(t, response, "live FuturesGetContractInfo response must decode")
	})
}

func TestFuturesIndexPriceInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesIndexPriceInfo(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesIndexPriceInfo request must succeed")
		require.NotNil(t, response, "live FuturesIndexPriceInfo response must decode")
	})
}

func TestFuturesContractPriceLimitations(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesContractPriceLimitations(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
		require.NoError(t, err, "live FuturesContractPriceLimitations request must succeed")
		require.NotNil(t, response, "live FuturesContractPriceLimitations response must decode")
	})
}

func TestFuturesContractOpenInterest(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesContractOpenInterest(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
		require.NoError(t, err, "live FuturesContractOpenInterest request must succeed")
		require.NotNil(t, response, "live FuturesContractOpenInterest response must decode")
	})
}

func TestFuturesGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetEstimatedDeliveryPrice(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetEstimatedDeliveryPrice request must succeed")
		require.NotNil(t, response, "live FuturesGetEstimatedDeliveryPrice response must decode")
	})
}

func TestFuturesGetMarketDepth(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetMarketDepth(t.Context(), btccwPair, "step5")
		require.NoError(t, err, "live FuturesGetMarketDepth request must succeed")
		require.NotNil(t, response, "live FuturesGetMarketDepth response must decode")
	})
}

func TestFuturesGetKline(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetKline(t.Context(), btccwPair, "5min", 5, time.Now().UTC().Add(-time.Hour), time.Now().UTC())
		require.NoError(t, err, "live FuturesGetKline request must succeed")
		require.NotNil(t, response, "live FuturesGetKline response must decode")
	})
	_, err := e.FuturesGetKline(t.Context(), btccwPair, "invalid", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "FGetKlineData must reject an invalid period")
	_, err = e.FuturesGetKline(t.Context(), btccwPair, "5min", 5, time.Now(), time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "FGetKlineData must reject a half-open interval")
}

func TestFuturesGetMarketOverview(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetMarketOverview(t.Context(), btccwPair)
		require.NoError(t, err, "live FuturesGetMarketOverview request must succeed")
		require.NotNil(t, response, "live FuturesGetMarketOverview response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/market/detail/merged", `{"status":"ok","tick":{"id":123}}`, nil)
	resp, err := h.FuturesGetMarketOverview(t.Context(), btccwPair)
	require.NoError(t, err, "FGetMarketOverviewData must not error")
	assert.Equal(t, uint64(123), resp.Tick.ID, "market overview ID should decode")
}

func TestFuturesLastTrade(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesLastTrade(t.Context(), btccwPair)
		require.NoError(t, err, "live FuturesLastTrade request must succeed")
		require.NotNil(t, response, "live FuturesLastTrade response must decode")
	})
}

func TestFuturesRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesRequestPublicBatchTrades(t.Context(), btccwPair, 50)
		require.NoError(t, err, "live FuturesRequestPublicBatchTrades request must succeed")
		require.NotNil(t, response, "live FuturesRequestPublicBatchTrades response must decode")
	})
}

func TestFuturesQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesQueryTieredAdjustmentFactor(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesQueryTieredAdjustmentFactor request must succeed")
		require.NotNil(t, response, "live FuturesQueryTieredAdjustmentFactor response must decode")
	})
}

func TestFuturesQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", 3)
		require.NoError(t, err, "live FuturesQueryHisOpenInterest request must succeed")
		require.NotNil(t, response, "live FuturesQueryHisOpenInterest response must decode")
	})
	for _, size := range []uint64{0, 201} {
		t.Run(strconv.FormatUint(size, 10), func(t *testing.T) {
			t.Parallel()
			_, err := e.FuturesQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", size)
			require.ErrorIs(t, err, errInvalidSize, "FQueryHisOpenInterest must reject an invalid size")
		})
	}
}

func TestFuturesQuerySystemStatus(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesQuerySystemStatus(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesQuerySystemStatus request must succeed")
		require.NotNil(t, response, "live FuturesQuerySystemStatus response must decode")
	})
}

func TestFuturesQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesQueryTopAccountsRatio(t.Context(), "BTC", "5min")
		require.NoError(t, err, "live FuturesQueryTopAccountsRatio request must succeed")
		require.NotNil(t, response, "live FuturesQueryTopAccountsRatio response must decode")
	})
}

func TestFuturesQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesQueryTopPositionsRatio(t.Context(), "BTC", "5min")
		require.NoError(t, err, "live FuturesQueryTopPositionsRatio request must succeed")
		require.NotNil(t, response, "live FuturesQueryTopPositionsRatio response must decode")
	})
}

func TestFuturesLiquidationOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesLiquidationOrders(t.Context(), currency.BTC, "filled", 0, 0, "", 0)
		require.NoError(t, err, "live FuturesLiquidationOrders request must succeed")
		require.NotNil(t, response, "live FuturesLiquidationOrders response must decode")
	})
	type testCase struct {
		name       string
		start, end uint64
	}
	for _, tc := range []testCase{
		{name: "both timestamps", start: 1604367600000, end: 1604367700000},
		{name: "start only", start: 1604367600000},
		{name: "end only", end: 1604367700000},
		{name: "no timestamps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/api/v3/contract_liquidation_orders", emptySuccessResponse, func(r *http.Request) {
				for key, timestamp := range map[string]uint64{"start_time": tc.start, "end_time": tc.end} {
					if timestamp == 0 {
						assert.NotContains(t, r.URL.Query(), key, "unset timestamps should be omitted")
					} else {
						assert.Equal(t, strconv.FormatUint(timestamp, 10), r.URL.Query().Get(key), "timestamp should match its argument")
					}
				}
			})
			_, err := h.FuturesLiquidationOrders(t.Context(), currency.BTC, "filled", tc.start, tc.end, "", 0)
			require.NoError(t, err, "FLiquidationOrders must accept optional timestamps")
		})
	}
}

func TestFuturesIndexKline(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesIndexKline(t.Context(), btccwPair, "5min", 5)
		require.NoError(t, err, "live FuturesIndexKline request must succeed")
		require.NotNil(t, response, "live FuturesIndexKline response must decode")
	})
}

func TestFuturesGetBasis(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{public: true})
		response, err := h.FuturesGetBasis(t.Context(), btccwPair, "5min", "open", 3)
		require.NoError(t, err, "live FuturesGetBasis request must succeed")
		require.NotNil(t, response, "live FuturesGetBasis response must decode")
	})
}

func TestFuturesGetAccountInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetAccountInfo(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetAccountInfo request must succeed")
		require.NotNil(t, response, "live FuturesGetAccountInfo response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_account_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetAccountInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAccountInfo must not error")
}

func TestFuturesGetPositionsInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetPositionsInfo(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetPositionsInfo request must succeed")
		require.NotNil(t, response, "live FuturesGetPositionsInfo response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_position_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetPositionsInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionsInfo must not error")
}

func TestFuturesGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetAllSubAccountAssets(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetAllSubAccountAssets request must succeed")
		require.NotNil(t, response, "live FuturesGetAllSubAccountAssets response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_account_list", emptySuccessResponse, nil)
	_, err := h.FuturesGetAllSubAccountAssets(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAllSubAccountAssets must not error")
}

func TestFuturesGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		_, err := h.FuturesGetSingleSubAccountInfo(t.Context(), "BTC", "9223372036854775807")
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesGetSingleSubAccountInfo input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_account_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetSingleSubAccountInfo(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubAccountInfo must not error")
}

func TestFuturesGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		_, err := h.FuturesGetSingleSubPositions(t.Context(), "BTC", "9223372036854775807")
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesGetSingleSubPositions input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_position_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetSingleSubPositions(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubPositions must not error")
}

func TestFuturesGetFinancialRecords(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetFinancialRecords(t.Context(), "BTC", "closeLong", 2, 0, 20)
		require.NoError(t, err, "live FuturesGetFinancialRecords request must succeed")
		require.NotNil(t, response, "live FuturesGetFinancialRecords response must decode")
	})
	t.Run("invalid record type", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).FuturesGetFinancialRecords(t.Context(), "BTC", "bogus", 1, 0, 0)
		require.ErrorIs(t, err, errInvalidRecordType, "invalid record types must fail before transport")
	})

	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v3/contract_financial_record", emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.FuturesGetFinancialRecords(t.Context(), "BTC", "closeLong", 2, 1, 20)
	require.NoError(t, err, "FGetFinancialRecords must not error")
	_, err = h.FuturesGetFinancialRecords(t.Context(), "BTC", "closeLong", 3, 1, 20)
	require.ErrorIs(t, err, errInvalidLookbackDays, "FGetFinancialRecords must reject lookbacks over two days")
}

func TestFuturesGetSettlementRecords(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetSettlementRecords(t.Context(), currency.BTC, 1, 20, time.Now().UTC().Add(-24*time.Hour), time.Now().UTC())
		require.NoError(t, err, "live FuturesGetSettlementRecords request must succeed")
		require.NotNil(t, response, "live FuturesGetSettlementRecords response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_user_settlement_records", emptySuccessResponse, nil)
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	_, err := h.FuturesGetSettlementRecords(t.Context(), currency.BTC, 1, 20, start, start.Add(time.Hour))
	require.NoError(t, err, "FGetSettlementRecords must not error")
	_, err = h.FuturesGetSettlementRecords(t.Context(), currency.BTC, 1, 20, start, time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "FGetSettlementRecords must reject a half-open interval")
}

func TestFuturesGetOrderLimits(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetOrderLimits(t.Context(), "BTC", "limit")
		require.NoError(t, err, "live FuturesGetOrderLimits request must succeed")
		require.NotNil(t, response, "live FuturesGetOrderLimits response must decode")
	})
	t.Run("invalid order price type", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		_, err := h.FuturesGetOrderLimits(t.Context(), "BTC", "not-real")
		require.Error(t, err, "FGetOrderLimits must reject invalid order price type")
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_limit", emptySuccessResponse, nil)
		_, err := h.FuturesGetOrderLimits(t.Context(), "BTC", "limit")
		require.NoError(t, err, "FGetOrderLimits must not error")
	})
}

func TestFuturesContractTradingFee(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesContractTradingFee(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesContractTradingFee request must succeed")
		require.NotNil(t, response, "live FuturesContractTradingFee response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_fee", emptySuccessResponse, nil)
	_, err := h.FuturesContractTradingFee(t.Context(), currency.BTC)
	require.NoError(t, err, "FuturesContractTradingFee must not error")
}

func TestFuturesGetTransferLimits(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetTransferLimits(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetTransferLimits request must succeed")
		require.NotNil(t, response, "live FuturesGetTransferLimits response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_transfer_limit", emptySuccessResponse, nil)
	_, err := h.FuturesGetTransferLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetTransferLimits must not error")
}

func TestFuturesGetPositionLimits(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetPositionLimits(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetPositionLimits request must succeed")
		require.NotNil(t, response, "live FuturesGetPositionLimits response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_position_limit", emptySuccessResponse, nil)
	_, err := h.FuturesGetPositionLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionLimits must not error")
}

func TestFuturesGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetAssetsAndPositions(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetAssetsAndPositions request must succeed")
		require.NotNil(t, response, "live FuturesGetAssetsAndPositions response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_account_position_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetAssetsAndPositions(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAssetsAndPositions must not error")
}

func TestFuturesTransfer(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesTransfer(t.Context(), "9223372036854775807", "BTC", "sub_to_master", 1)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesTransfer input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_master_sub_transfer", emptySuccessResponse, nil)
	_, err := h.FuturesTransfer(t.Context(), "154263566", "BTC", "sub_to_master", 5)
	require.NoError(t, err, "FTransfer must not error")
}

func TestFuturesGetTransferRecords(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetTransferRecords(t.Context(), "BTC", "master_to_sub", 2, 1, 20)
		require.NoError(t, err, "live FuturesGetTransferRecords request must succeed")
		require.NotNil(t, response, "live FuturesGetTransferRecords response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_master_sub_transfer_record", emptySuccessResponse, nil)
	_, err := h.FuturesGetTransferRecords(t.Context(), "BTC", "master_to_sub", 2, 1, 20)
	require.NoError(t, err, "FGetTransferRecords must not error")
}

func TestFuturesGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetAvailableLeverage(t.Context(), currency.BTC)
		require.NoError(t, err, "live FuturesGetAvailableLeverage request must succeed")
		require.NotNil(t, response, "live FuturesGetAvailableLeverage response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_available_level_rate", emptySuccessResponse, nil)
	_, err := h.FuturesGetAvailableLeverage(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAvailableLeverage must not error")
}

func TestFuturesOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesOrder(t.Context(), currency.EMPTYPAIR, "INVALID", "quarter", "", "buy", "open", "limit", 1, 1, 1)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesOrder input")
	})
	for _, offset := range []string{"bogus", ""} {
		t.Run("invalid offset "+offset, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "setup must succeed")
			_, err := h.FuturesOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "", "buy", offset, orderPriceTypeLimit, 1, 1, 1)
			require.ErrorIs(t, err, errInvalidOffsetAmounts, "invalid offsets must fail before transport")
		})
	}

	type testCase struct {
		name             string
		priceType        string
		clientOrderID    string
		expectedClientID uint64
		expectedErr      error
	}
	for _, tc := range []testCase{
		{name: "limit", priceType: orderPriceTypeLimit, clientOrderID: "123", expectedClientID: 123},
		{name: "post only", priceType: orderPriceTypePostOnly, clientOrderID: "123", expectedClientID: 123},
		{name: "no client ID", priceType: orderPriceTypeLimit},
		{name: "minimum client ID", priceType: orderPriceTypeLimit, clientOrderID: "1", expectedClientID: 1},
		{name: "maximum client ID", priceType: orderPriceTypeLimit, clientOrderID: "9223372036854775807", expectedClientID: 9223372036854775807},
		{name: "zero client ID", priceType: orderPriceTypeLimit, clientOrderID: "0", expectedErr: errInvalidClientOrderID},
		{name: "negative client ID", priceType: orderPriceTypeLimit, clientOrderID: "-1", expectedErr: errInvalidClientOrderID},
		{name: "overflow client ID", priceType: orderPriceTypeLimit, clientOrderID: "9223372036854775808", expectedErr: errInvalidClientOrderID},
		{name: "invalid client ID", priceType: orderPriceTypeLimit, clientOrderID: "invalid", expectedErr: errInvalidClientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order", emptySuccessResponse, func(r *http.Request) {
				assert.NoError(t, tc.expectedErr, "invalid client IDs should not reach the endpoint")
				type orderRequest struct {
					OrderPriceType string  `json:"order_price_type"`
					LeverageRate   uint64  `json:"lever_rate"`
					ClientOrderID  *uint64 `json:"client_order_id"`
				}
				var body orderRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body), "order request must decode")
				assert.Equal(t, tc.priceType, body.OrderPriceType, "order price type should match")
				assert.Equal(t, uint64(1), body.LeverageRate, "leverage should match the requested integer")
				if tc.clientOrderID == "" {
					assert.Nil(t, body.ClientOrderID, "unset client IDs should be omitted")
				} else {
					require.NotNil(t, body.ClientOrderID, "client order ID must be present")
					assert.Equal(t, tc.expectedClientID, *body.ClientOrderID, "client ID should retain precision as a JSON integer")
				}
			})
			_, err := h.FuturesOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", tc.clientOrderID, "buy", "open", tc.priceType, 1, 1, 1)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "FuturesOrder must reject client IDs outside the documented range")
				return
			}
			require.NoError(t, err, "FuturesOrder must accept valid client order IDs")
		})
	}
}

func TestFuturesPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.FuturesPlaceBatchOrder(t.Context(), []*FuturesBatchOrderRequest{{Symbol: "INVALID", ContractType: "quarter", Price: 1, Volume: 1, Direction: "buy", Offset: "open", LeverageRate: 1, OrderPriceType: "limit"}})
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection response must decode")
			require.NotEmpty(t, response.Results.Errors, "fabricated batch order must have a venue rejection")
			assert.Empty(t, response.Results.Success, "fabricated batch order should never succeed")
		}
	})
	for _, offset := range []string{"bogus", ""} {
		t.Run("invalid offset "+offset, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "setup must succeed")
			_, err := h.FuturesPlaceBatchOrder(t.Context(), []*FuturesBatchOrderRequest{{Symbol: "BTC", ContractType: "quarter", Price: 5, Volume: 1, Direction: "buy", Offset: offset, LeverageRate: 1, OrderPriceType: "limit"}})
			require.ErrorIs(t, err, errInvalidOffsetAmounts, "invalid offsets must fail before transport")
		})
	}

	type testCase struct {
		name     string
		count    int
		nilBatch bool
		nilOrder bool
		wantErr  error
	}
	for _, tc := range []testCase{
		{name: "nil batch", nilBatch: true, wantErr: errEmptyOrderBatch},
		{name: "empty batch", wantErr: errEmptyOrderBatch},
		{name: "too many orders", count: 11, wantErr: errBatchOrderLimitExceeded},
		{name: "nil order", count: 1, nilOrder: true, wantErr: common.ErrNilPointer},
		{name: "single order", count: 1},
		{name: "maximum batch", count: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_batchorder", emptySuccessResponse, func(r *http.Request) {
				calls.Add(1)
				type batchPayload struct {
					Orders []map[string]any `json:"orders_data"`
				}
				var payload batchPayload
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "batch payload should decode") {
					return
				}
				assert.Len(t, payload.Orders, tc.count, "batch payload should retain every order")
				for _, row := range payload.Orders {
					assert.Equal(t, "BTC", row["symbol"], "underlying currency should be retained")
					assert.Equal(t, "quarter", row["contract_type"], "contract type should be retained")
					assert.Equal(t, float64(5), row["price"], "order price should be retained")
					assert.Equal(t, float64(1), row["volume"], "order volume should be retained")
					assert.Equal(t, "buy", row["direction"], "order direction should be retained")
					assert.Equal(t, "open", row["offset"], "order offset should be retained")
					assert.Equal(t, float64(1), row["lever_rate"], "leverage should use the documented field")
					assert.Equal(t, "limit", row["order_price_type"], "order price type should use the documented field")
					assert.NotContains(t, row, "leverRate", "batch request should omit the undocumented leverage field")
					assert.NotContains(t, row, "orderPriceType", "batch request should omit the undocumented order price type field")
				}
			})
			var orders []*FuturesBatchOrderRequest
			if !tc.nilBatch {
				orders = make([]*FuturesBatchOrderRequest, tc.count)
				if !tc.nilOrder {
					for i := range orders {
						orders[i] = &FuturesBatchOrderRequest{
							Symbol:         "BTC",
							ContractType:   "quarter",
							Price:          5,
							Volume:         1,
							Direction:      "buy",
							Offset:         "open",
							LeverageRate:   1,
							OrderPriceType: "limit",
						}
					}
				}
			}
			_, err := h.FuturesPlaceBatchOrder(t.Context(), orders)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid batch must return the expected validation error")
				assert.Zero(t, calls.Load(), "invalid batch should not send a request")
				return
			}
			require.NoError(t, err, "valid batch must be accepted")
			assert.Equal(t, int32(1), calls.Load(), "valid batch should send one request")
		})
	}
	t.Run("mixed batch results", func(t *testing.T) {
		t.Parallel()
		const results = `{"errors":[{"index":1,"err_code":1037,"err_msg":"The leverage is invalid."}],"success":[{"index":2,"order_id":773120304138219520,"order_id_str":"773120304138219520","client_order_id":9223372036854775807}]}`
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_batchorder", `{"status":"ok","data":`+results+`,"ts":1604367844388}`, nil)
		response, err := h.FuturesPlaceBatchOrder(t.Context(), []*FuturesBatchOrderRequest{
			{Symbol: "BTC", ContractType: "quarter", Price: 5, Volume: 1, Direction: "buy", Offset: "open", LeverageRate: 1, OrderPriceType: "limit"},
			{Symbol: "BTC", ContractType: "quarter", Price: 5, Volume: 1, Direction: "buy", Offset: "open", LeverageRate: 1, OrderPriceType: "limit"},
		})
		require.NoError(t, err, "mixed batch response must decode")
		assert.Equal(t, []BatchOrderError{{Index: 1, ErrorCode: 1037, ErrorMessage: "The leverage is invalid."}}, response.Results.Errors, "batch response should retain each venue rejection")
		assert.Equal(t, []BatchOrderSuccess{{Index: 2, OrderID: 773120304138219520, OrderIDString: "773120304138219520", ClientOrderID: 9223372036854775807}}, response.Results.Success, "batch response should retain successful order identifiers without losing precision")
		assert.Equal(t, int64(1604367844388), response.Timestamp.Time().UnixMilli(), "batch response should retain its timestamp")
	})
}

func TestFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.FuturesCancelOrder(t.Context(), currency.BTC, "9223372036854775807", "")
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection response must decode")
			require.NotEmpty(t, response.Cancellations.Errors, "fabricated cancellation must have a venue rejection")
			assert.Empty(t, response.Cancellations.Successes, "fabricated cancellation should never succeed")
		}
	})
	type testCase struct {
		name           string
		orderIDs       string
		clientOrderIDs string
		expectedErr    error
	}
	for _, tc := range []testCase{
		{name: "one order", orderIDs: "123"},
		{name: "one client order", clientOrderIDs: "456"},
		{name: "ten orders", orderIDs: "1,2,3,4,5,6,7,8,9,10"},
		{name: "ten client orders", clientOrderIDs: "1,2,3,4,5,6,7,8,9,10"},
		{name: "missing identifiers", expectedErr: order.ErrOrderIDNotSet},
		{name: "mixed identifiers", orderIDs: "123", clientOrderIDs: "456", expectedErr: errMixedOrderIdentifiers},
		{name: "too many orders", orderIDs: "1,2,3,4,5,6,7,8,9,10,11", expectedErr: errCancelOrderLimitExceeded},
		{name: "too many client orders", clientOrderIDs: "1,2,3,4,5,6,7,8,9,10,11", expectedErr: errCancelOrderLimitExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_cancel", emptySuccessResponse, func(r *http.Request) {
				require.NoError(t, tc.expectedErr, "invalid cancellation must not send a request")
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request), "cancellation request must decode")
				assert.Equal(t, "BTC", request["symbol"], "symbol should match")
				if tc.orderIDs != "" {
					assert.Equal(t, tc.orderIDs, request["order_id"], "order IDs should be sent in one request")
					assert.NotContains(t, request, "client_order_id", "order cancellation should omit client IDs")
				} else {
					assert.Equal(t, tc.clientOrderIDs, request["client_order_id"], "client IDs should be sent in one request")
					assert.NotContains(t, request, "order_id", "client cancellation should omit order IDs")
				}
			})
			_, err := h.FuturesCancelOrder(t.Context(), currency.BTC, tc.orderIDs, tc.clientOrderIDs)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "FuturesCancelOrder must reject invalid identifiers")
				return
			}
			require.NoError(t, err, "FuturesCancelOrder must send one cancellation request")
		})
	}
}

func TestFuturesCancelAllOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesCancelAllOrders(t.Context(), currency.EMPTYPAIR, "INVALID", "quarter")
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesCancelAllOrders input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_cancelall", emptySuccessResponse, nil)
	_, err := h.FuturesCancelAllOrders(t.Context(), currency.EMPTYPAIR, "BTC", "quarter")
	require.NoError(t, err, "FCancelAllOrders must not error")
}

func TestFuturesFlashCloseOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesFlashCloseOrder(t.Context(), currency.EMPTYPAIR, "INVALID", "quarter", "buy", "lightning", "", 1)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesFlashCloseOrder input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/lightning_close_position", emptySuccessResponse, nil)
	_, err := h.FuturesFlashCloseOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "buy", "lightning", "", 1)
	require.NoError(t, err, "FFlashCloseOrder must not error")
}

func TestFuturesGetOrderInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetOrderInfo(t.Context(), "BTC", "", "9223372036854775807")
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection response must decode")
			assert.Empty(t, response.Orders, "fabricated lookup should return no orders")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_info", emptySuccessResponse, nil)
	_, err := h.FuturesGetOrderInfo(t.Context(), "BTC", "", "123")
	require.NoError(t, err, "FGetOrderInfo must not error")
}

func TestFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesOrderDetails(t.Context(), "BTC", "9223372036854775807", "quotation", time.Now().UTC().Add(-time.Hour), 1, 20)
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection response must decode")
			assert.Zero(t, response.Order.OrderID, "fabricated lookup should return no order")
		}
	})
	for _, tc := range []struct {
		name      string
		createdAt time.Time
	}{
		{name: "omitted creation time"},
		{name: "millisecond creation time", createdAt: time.Date(2026, time.November, 11, 12, 34, 56, 789000000, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := make(chan []byte, 1)
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_detail", emptySuccessResponse, func(r *http.Request) {
				payload, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "request body should be readable")
				body <- payload
			})
			_, err := h.FuturesOrderDetails(t.Context(), "BTC", "123", "quotation", tc.createdAt, 1, 20)
			require.NoError(t, err, "FuturesOrderDetails must not error")
			var requestBody map[string]any
			require.NoError(t, json.Unmarshal(<-body, &requestBody), "request body must decode")
			if tc.createdAt.IsZero() {
				assert.NotContains(t, requestBody, "created_at", "omitted creation time should not be sent")
			} else {
				assert.Equal(t, strconv.FormatInt(tc.createdAt.UnixMilli(), 10), requestBody["created_at"], "creation time should retain milliseconds")
			}
		})
	}
}

func TestFuturesGetOpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetOpenOrders(t.Context(), currency.BTC, 1, 20)
		require.NoError(t, err, "live FuturesGetOpenOrders request must succeed")
		require.NotNil(t, response, "live FuturesGetOpenOrders response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_openorders", emptySuccessResponse, nil)
	_, err := h.FuturesGetOpenOrders(t.Context(), currency.BTC, 1, 20)
	require.NoError(t, err, "FGetOpenOrders must not error")
}

func TestFuturesGetOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, 2, 0, 20)
		require.NoError(t, err, "live FuturesGetOrderHistory request must succeed")
		require.NotNil(t, response, "live FuturesGetOrderHistory response must decode")
	})
	_, err := e.FuturesGetOrderHistory(t.Context(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		3, 0, 0)
	require.ErrorIs(t, err, errInvalidLookbackDays, "FGetOrderHistory must reject lookbacks over two days")
}

func TestFuturesGetOrderHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, time.Now().UTC().Add(-24*time.Hour), time.Now().UTC(), 0, 20)
		require.NoError(t, err, "live FuturesGetOrderHistoryByTimeRange request must succeed")
		require.NotNil(t, response, "live FuturesGetOrderHistoryByTimeRange response must decode")
	})
	body := make(chan []byte, 1)
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v3/contract_hisorders", `{"code":200,"data":[{"query_id":12,"order_id":34,"order_id_str":"34","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6}]}`, func(r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		body <- payload
	})
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(48 * time.Hour)
	resp, err := h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime, 11, 50)
	require.NoError(t, err, "FGetOrderHistoryByTimeRange must not error")
	require.Len(t, resp.OrderHistory.Orders, 1, "decoded order history must be returned")
	assert.Equal(t, uint64(12), resp.OrderHistory.Orders[0].QueryID, "query ID should decode")

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(<-body, &requestBody), "request body must decode")
	assert.Equal(t, float64(startTime.UnixMilli()), requestBody["start_time"], "start time should be preserved")
	assert.Equal(t, float64(endTime.UnixMilli()), requestBody["end_time"], "end time should be preserved")
	assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "pagination direction should move forwards")
	assert.Equal(t, float64(11), requestBody["from_id"], "cursor should be preserved")
	assert.Equal(t, float64(50), requestBody["limit"], "limit should be preserved")

	_, err = h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime.Add(time.Millisecond), 0, 50)
	require.ErrorIs(t, err, errHistoryTimeRangeExceeded, "FGetOrderHistoryByTimeRange must reject intervals over 48 hours")
	_, err = h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, endTime, startTime, 0, 50)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FGetOrderHistoryByTimeRange must reject reversed intervals")
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 2, 0, 20)
		require.NoError(t, err, "live FuturesTradeHistory request must succeed")
		require.NotNil(t, response, "live FuturesTradeHistory response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v3/contract_matchresults", emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.FuturesTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 2, 1, 20)
	require.NoError(t, err, "FTradeHistory must not error")
}

func TestFuturesPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "INVALID", "quarter", "greaterOrEqual", "limit", "buy", "open", 1, 1, 1, 1)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesPlaceTriggerOrder input")
	})
	for _, offset := range []string{"bogus", ""} {
		t.Run("invalid offset "+offset, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "setup must succeed")
			_, err := h.FuturesPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "greaterOrEqual", "limit", "buy", offset, 1.1, 1.05, 5, 2)
			require.ErrorIs(t, err, errInvalidOffset, "invalid offsets must fail before transport")
		})
	}

	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_order", emptySuccessResponse, nil)
	_, err := h.FuturesPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err, "FPlaceTriggerOrder must not error")
}

func TestFuturesCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		response, err := h.FuturesCancelTriggerOrder(t.Context(), "BTC", "9223372036854775807")
		if err != nil {
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue rejection must report an authenticated request failure")
		} else {
			require.NotNil(t, response, "live rejection response must decode")
			require.NotEmpty(t, response.Cancellations.Errors, "fabricated cancellation must have a venue rejection")
			assert.Empty(t, response.Cancellations.Successes, "fabricated cancellation should never succeed")
		}
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_cancel", emptySuccessResponse, nil)
	_, err := h.FuturesCancelTriggerOrder(t.Context(), "BTC", "123")
	require.NoError(t, err, "FCancelTriggerOrder must not error")
}

func TestFuturesCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		_, err := h.FuturesCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "INVALID", "this_week")
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesCancelAllTriggerOrders input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_cancelall", emptySuccessResponse, nil)
	_, err := h.FuturesCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err, "FCancelAllTriggerOrders must not error")
}

func TestFuturesQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 1, 20)
		require.NoError(t, err, "live FuturesQueryTriggerOpenOrders request must succeed")
		require.NotNil(t, response, "live FuturesQueryTriggerOpenOrders response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_openorders", emptySuccessResponse, nil)
	_, err := h.FuturesQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 1, 20)
	require.NoError(t, err, "FQueryTriggerOpenOrders must not error")
}

func TestFuturesQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{})
		response, err := h.FuturesQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", 2, 1, 20)
		require.NoError(t, err, "live FuturesQueryTriggerOrderHistory request must succeed")
		require.NotNil(t, response, "live FuturesQueryTriggerOrderHistory response must decode")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_hisorders", emptySuccessResponse, nil)
	_, err := h.FuturesQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", 10, 1, 20)
	require.NoError(t, err, "FQueryTriggerOrderHistory must not error")
}

func TestFormatFuturesPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, want   string
		pair         currency.Pair
		convertQuote bool
		wantErr      error
	}{
		{name: "delivery alias", pair: btccwPair, want: "BTC_CW"},
		{name: "lowercase dated pair", pair: currency.NewPair(currency.BTC, currency.NewCode("271231")).Lower(), want: "BTC271231"},
		{name: "resolve delivery alias", pair: btccwPair, convertQuote: true, want: "BTC271231"},
		{name: "stable quote", pair: currency.NewBTCUSDT(), want: "BTC-USDT"},
		{name: "missing expiry", pair: currency.NewPair(currency.BTC, currency.NewCode("NW")), convertQuote: true, wantErr: errInvalidContractType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "formatter test exchange must initialise")
			h.futureContractCodes = map[string]currency.Code{"CW": currency.NewCode("271231")}
			got, err := h.formatFuturesPair(tc.pair, tc.convertQuote)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "missing expiry must return the expected error")
			} else {
				require.NoError(t, err, "supported futures pair must format")
				assert.Equal(t, tc.want, got, "futures symbol should use the requested alias or concrete expiry")
			}
		})
	}
}

func TestFormatFuturesCode(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	formatted, err := h.formatFuturesCode(currency.NewCode("btc"))
	require.NoError(t, err, "formatFuturesCode must not error")
	assert.Equal(t, "BTC", formatted, "futures code should use the configured case")
}

func TestPairFromContractExpiryCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, alias, date string
		base              currency.Code
		wantErr           error
	}{
		{name: "current week", alias: "CW", date: "271231", base: currency.BTC},
		{name: "next week", alias: "NW", date: "280107", base: currency.BTC},
		{name: "current quarter", alias: "CQ", date: "280331", base: currency.BTC},
		{name: "next quarter", alias: "NQ", date: "280630", base: currency.BTC},
		{name: "preserve underlying", alias: "CW", date: "271231", base: currency.ETH},
		{name: "missing alias", alias: "missing", base: currency.BTC, wantErr: errInvalidContractType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := &Exchange{futureContractCodes: make(map[string]currency.Code)}
			if tc.date != "" {
				h.futureContractCodes[tc.alias] = currency.NewCode(tc.date)
			}
			pair := currency.NewPair(tc.base, currency.NewCode(tc.alias))
			got, err := h.pairFromContractExpiryCode(pair)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "unknown delivery alias must return the expected error")
				assert.Equal(t, pair, got, "failed resolution should retain the supplied pair")
			} else {
				require.NoError(t, err, "cached delivery alias must resolve")
				assert.Equal(t, currency.NewPair(tc.base, currency.NewCode(tc.date)), got, "resolved pair should preserve the underlying and use the cached expiry")
			}
		})
	}
}

func TestFuturesSwitchLeverage(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("Live testing requires the mock_test_off build tag")
		}
		h := newHTXLiveExchange(t, apiCredentials, htxLiveOptions{changesAccount: true})
		err := h.FuturesSwitchLeverage(t.Context(), currency.NewCode("INVALID"), 5)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "venue must reject the fabricated FuturesSwitchLeverage input")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_switch_lever_rate", `{"status":"ok","data":{"symbol":"BTC","lever_rate":5}}`, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"symbol":"BTC","lever_rate":5}`, string(body), "delivery leverage body should match")
	})
	require.NoError(t, h.FuturesSwitchLeverage(t.Context(), currency.BTC, 5), "FSwitchLeverage must not error")
}
