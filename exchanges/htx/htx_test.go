package htx

import (
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own test keys here for due diligence testing.
const (
	apiKey                  = "" //nolint:gosec // live HTX tests use developer-supplied local credentials.
	apiSecret               = ""      //nolint:gosec // live HTX tests use developer-supplied local credentials.
	canManipulateRealOrders = false
)

var (
	_                  exchange.IBotExchange = (*Exchange)(nil)
	e                  *Exchange
	btcFutureDatedPair currency.Pair
	btccwPair          = currency.NewPair(currency.BTC, currency.NewCode("CW"))
	btcusdPair         = currency.NewPairWithDelimiter("BTC", "USD", "-")
	btcusdtPair        = currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethusdPair         = currency.NewPairWithDelimiter("ETH", "USD", "-")
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("HTX Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetSignatureHost(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{
			name: "spot",
			in:   "https://api.huobi.pro",
			want: "api.huobi.pro",
		},
		{
			name: "futures with path",
			in:   "https://api.hbdm.com/swap-api/v1",
			want: "api.hbdm.com",
		},
		{
			name: "custom host with port",
			in:   "https://localhost:8443",
			want: "localhost:8443",
		},
		{
			name:    "missing host",
			in:      "/v1/order/orders",
			wantErr: errInvalidEndpoint,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := getSignatureHost(tt.in)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "getSignatureHost must return expected error")
				return
			}
			require.NoError(t, err, "getSignatureHost must not error")
			assert.Equal(t, tt.want, got, "signature host should match")
		})
	}
}

func TestSpotMatchResultsEndpoint(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/order/matchresults", htxGetOrdersMatch, "spot match results endpoint should match HTX docs")
}

func TestUSDTFuturesEndpointPaths(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/linear-swap-api/v1/swap_contract_info", linearSwapMarkets, "linear swap contract info endpoint should match HTX docs")
	assert.Equal(t, "/linear-swap-api/v1/swap_funding_rate", linearSwapFunding, "linear swap funding rate endpoint should match HTX docs")
	assert.Equal(t, "/linear-swap-api/v1/swap_batch_funding_rate", linearSwapBatchFunding, "linear swap batch funding rate endpoint should match HTX docs")
	assert.Equal(t, "/v5/account/balance", v5AccountBalance, "V5 account balance endpoint should match HTX docs")
	assert.Equal(t, "/v5/trade/order", v5TradeOrder, "V5 order endpoint should match HTX docs")
	assert.Equal(t, "/v5/trade/cancel_order", v5TradeCancelOrder, "V5 cancel order endpoint should match HTX docs")
	assert.Equal(t, "/v5/trade/cancel_all_orders", v5TradeCancelAllOrders, "V5 cancel all orders endpoint should match HTX docs")
	assert.Equal(t, "/v5/trade/order/opens", v5TradeOrderOpens, "V5 open orders endpoint should match HTX docs")
	assert.Equal(t, "/v5/market/open_interest", v5MarketOpenInterest, "V5 open interest endpoint should match HTX docs")
}

func TestV5OrderQueryResponseUnmarshal(t *testing.T) {
	t.Parallel()
	var resp *V5OrderQueryResponse
	err := json.Unmarshal([]byte(`{"code":200,"message":"Success","data":{"order_id":"1","contract_code":"BTC-USDT","side":"buy","type":"limit","price":"5000","volume":"1","trade_volume":"0.25","trade_turnover":"1250","fee":"0.1","lever_rate":10,"reduce_only":false,"created_time":"1769076510922","updated_time":"1769076510922"}}`), &resp)
	require.NoError(t, err, "Unmarshal must decode HTX V5 order response")
	require.NotNil(t, resp, "response must not be nil")
	assert.Equal(t, 5000.0, resp.Data.Price.Float64(), "price should decode from quoted number")
	assert.Equal(t, 0.25, resp.Data.TradeVolume.Float64(), "trade volume should decode from quoted number")
	assert.Equal(t, 10.0, resp.Data.LeverageRate.Float64(), "leverage should decode from bare number")
}

func TestV5OpenInterestResponseUnmarshal(t *testing.T) {
	t.Parallel()
	var resp *V5OpenInterestResponse
	err := json.Unmarshal([]byte(`{"code":200,"data":{"amount":"244.004","volume":"244004","value":"29275599.92","contract_code":"BTC-USDT","trade_amount":"9.838","trade_volume":"9838","trade_turnover":"1091416.458752"},"message":null,"success":true}`), &resp)
	require.NoError(t, err, "Unmarshal must decode HTX V5 open interest response")
	require.NotNil(t, resp, "response must not be nil")
	assert.True(t, resp.Success, "success should decode")
	assert.Equal(t, 244.004, resp.Data.Amount.Float64(), "amount should decode from quoted number")
	assert.Equal(t, 1091416.458752, resp.Data.TradeTurnover.Float64(), "trade turnover should decode from quoted number")
}

func TestIsEmptyHTXData(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		data []byte
		want bool
	}{
		{name: "empty", want: true},
		{name: "empty string", data: []byte(`""`), want: true},
		{name: "null", data: []byte(` null `), want: true},
		{name: "array", data: []byte(`[]`)},
		{name: "object", data: []byte(`{}`)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isEmptyHTXData(tt.data), "isEmptyHTXData should identify empty HTX data values")
		})
	}
}

func TestGetCurrenciesIncludingChains(t *testing.T) {
	t.Parallel()
	r, err := e.GetCurrenciesIncludingChains(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.Greater(t, len(r), 1, "should get more than one currency back")
	r, err = e.GetCurrenciesIncludingChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.Equal(t, 1, len(r), "Should only get one currency back")
}

func TestGetMarginRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginRates(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKline(t.Context(), KlinesRequestParams{Symbol: btcusdtPair, Period: "1min"})
	require.NoError(t, err)
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetailMerged(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepth(t.Context(),
		&OrderBookDataRequestParams{
			Symbol: btcusdtPair,
			Type:   OrderBookDataRequestParamsTypeStep1,
		})
	require.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestSpotPrice(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), btcusdtPair, 50)
	require.NoError(t, err)
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetail(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context())
	require.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencies(t.Context())
	require.NoError(t, err)
}

func TestGet24HrMarketSummary(t *testing.T) {
	t.Parallel()
	_, err := e.Get24HrMarketSummary(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetAccounts(t.Context())
	require.NoError(t, err)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = e.GetAccountBalance(t.Context(), userID)
	require.NoError(t, err, "GetAccountBalance must not error")
}

func TestGetAccountBalanceCredentialsError(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetAccountBalance(t.Context(), "1")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetAccountBalance must return credentials error")
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAggregatedBalance(t.Context())
	require.NoError(t, err)
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg := SpotNewOrderRequestParams{
		Symbol:    btcusdtPair,
		AccountID: 1997024,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := e.SpotNewOrder(t.Context(), &arg)
	require.NoError(t, err)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelExistingOrder(t.Context(), 1337)
	assert.Error(t, err)
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetOrder(t.Context(), 1337)
	require.NoError(t, err)
}

func TestGetOrderMatchResults(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrderMatchResults(t.Context(), 1337)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrderMatchResults must return credentials error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrders(t.Context(), btcusdtPair, "buy-limit", "2019-03-10", "2019-03-19", "submitted", "5", "prev", "100")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrders must return credentials error")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOpenOrders(t.Context(), btcusdtPair, "100009", "buy", 10)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOpenOrders must return credentials error")
}

func TestGetOrdersMatch(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetOrdersMatch(t.Context(), btcusdtPair, "buy-limit", "2019-03-10", "2019-03-19", "5", "prev", "100")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrdersMatch must return credentials error")
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginLoanOrders(t.Context(), btcusdtPair, "", "", "", "", "", "", "")
	require.NoError(t, err)
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginAccountBalance(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestMarginTransfer(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   bool
	}{
		{name: "in", in: true},
		{name: "out"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			_, err := h.MarginTransfer(t.Context(), btcusdtPair, "usdt", 1.25, tt.in)
			require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginTransfer must return credentials error")
		})
	}
}

func TestMarginOrder(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.MarginOrder(t.Context(), btcusdtPair, "usdt", 1.25)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginOrder must return credentials error")
}

func TestMarginRepayment(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.MarginRepayment(t.Context(), 1, 1.25)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "MarginRepayment must return credentials error")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		code    currency.Code
		address string
		amount  float64
		wantErr error
	}{
		{name: "invalid", wantErr: errWithdrawDetailsUnset},
		{name: "credentials", code: currency.USDT, address: "address", amount: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			_, err := h.Withdraw(t.Context(), tt.code, tt.address, "", "trc20usdt", tt.amount, 0.1)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "Withdraw must return expected error")
				return
			}
			require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "Withdraw must return credentials error")
		})
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	t.Run("credentials", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		_, err := h.CancelWithdraw(t.Context(), 1337)
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelWithdraw must return credentials error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		_, err := e.CancelWithdraw(t.Context(), 1337)
		require.Error(t, err)
	})
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestQueryDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.QueryDepositAddress(t.Context(), currency.USDT)
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestQueryWithdrawQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.QueryWithdrawQuotas(t.Context(), currency.BTC.Lower().String())
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.SearchForExistedWithdrawsAndDeposits(t.Context(), currency.BTC, "deposit", "", 0, 100)
	require.NoError(t, err)
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelOrderBatch(t.Context(), []string{"1234"}, nil)
	require.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSDT(),
		},
	})
	require.NoError(t, err)
}

func TestGetBatchLinearSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchLinearSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchFuturesContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchFuturesContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func updatePairsOnce(tb testing.TB, h *Exchange) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	testexch.UpdatePairsOnce(tb, h)

	h.futureContractCodesMutex.Lock()
	if len(h.futureContractCodes) == 0 {
		// Restored pairs from cache, so haven't populated futureContract Codes
		require.NotEmpty(tb, futureContractCodesCache, "futureContractCodesCache must not be empty")
		h.futureContractCodes = futureContractCodesCache
	} else {
		futureContractCodesCache = h.futureContractCodes
	}
	h.futureContractCodesMutex.Unlock()

	if btcFutureDatedPair.Equal(currency.EMPTYPAIR) {
		p, err := h.pairFromContractExpiryCode(btccwPair)
		require.NoError(tb, err, "pairFromContractCode must not error")
		btcFutureDatedPair = p
	}

	err := h.CurrencyPairs.EnablePair(asset.Futures, btcFutureDatedPair) // Must enable every time we refresh the CurrencyPairs from cache
	require.NoError(tb, common.ExcludeError(err, currency.ErrPairAlreadyEnabled))
}
