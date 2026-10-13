package gateio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// newCollateralTransferExchange returns an exchange whose wallet transfers are served locally, so the collateral
// transfer helpers can be checked against the request they send without depending on the wallet transfer fixtures
func newCollateralTransferExchange(t *testing.T, from, to asset.Item) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "transfer request method should be POST")
		assert.Equal(t, "/api/v4/wallet/transfers", r.URL.Path, "transfer request should reach the wallet transfers endpoint")
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "transfer request body should be read")
		var body map[string]any
		assert.NoError(t, json.Unmarshal(raw, &body), "transfer request body should decode")
		assert.Equal(t, from.String(), body["from"], "transfer should debit the expected account")
		assert.Equal(t, to.String(), body["to"], "transfer should credit the expected account")
		assert.Equal(t, "USDT", body["currency"], "transfer should move the requested currency")
		assert.Equal(t, "BTC_USDT", body["currency_pair"], "transfer should name the isolated margin market")
		assert.Equal(t, "10", body["amount"], "transfer should move the requested amount")
		_, err = w.Write([]byte(`{"tx_id":59636381286}`))
		assert.NoError(t, err, "transfer response should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	return ex
}

func TestTransferCollateralToIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralToIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferCollateralToIsolatedMargin must reject an empty currency")

	ex := e
	if mockTests {
		ex = newCollateralTransferExchange(t, asset.Spot, asset.Margin)
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.TransferCollateralToIsolatedMargin(t.Context(), getPair(t, asset.Margin), currency.USDT, 10)
	require.NoError(t, err, "TransferCollateralToIsolatedMargin must not error")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381286}, result, "TransferCollateralToIsolatedMargin should decode the transaction ID")
		return
	}
	assert.NotZero(t, result.TransactionID, "TransferCollateralToIsolatedMargin should return a transaction ID")
}

func TestTransferCollateralFromIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralFromIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferCollateralFromIsolatedMargin must reject an empty currency")

	ex := e
	if mockTests {
		ex = newCollateralTransferExchange(t, asset.Margin, asset.Spot)
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.TransferCollateralFromIsolatedMargin(t.Context(), getPair(t, asset.Margin), currency.USDT, 10)
	require.NoError(t, err, "TransferCollateralFromIsolatedMargin must not error")
	if mockTests {
		assert.Equal(t, &TransactionIDResponse{TransactionID: 59636381286}, result, "TransferCollateralFromIsolatedMargin should decode the transaction ID")
		return
	}
	assert.NotZero(t, result.TransactionID, "TransferCollateralFromIsolatedMargin should return a transaction ID")
}

func TestGetIsolatedMarginAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginAccounts(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetIsolatedMarginAccounts must not error")
	if mockTests {
		exp := []MarginAccountItem{{
			CurrencyPair:          BTCUSDT,
			AccountType:           "mmr",
			Leverage:              20,
			RiskRate:              1.3318,
			MaintenanceMarginRate: 16.5949188975473644,
			Base:                  AccountBalanceInformation{Currency: currency.BTC, Available: 0.047060413211, Borrowed: 0.047233},
			Quote:                 AccountBalanceInformation{Currency: currency.USDT, Available: 1234},
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginAccounts should return a result")
}

func TestGetIsolatedMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	tn := time.Now()
	_, err := e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, tn.Add(time.Hour), tn, 0, 0, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIsolatedMarginAccountBalanceChangeHistory must reject a start after the end")

	_, err = e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, BTCUSDT, time.Time{}, time.Time{}, 1, 100, "invalid")
	require.ErrorIs(t, err, errInvalidIsolatedMarginAccountType, "GetIsolatedMarginAccountBalanceChangeHistory must reject an unknown account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 0, "margin_out")
	require.NoError(t, err, "GetIsolatedMarginAccountBalanceChangeHistory must not error")
	if mockTests {
		exp := []IsolatedMarginAccountBalanceChangeInfo{{
			ID:            "123456",
			Time:          types.Time(time.UnixMilli(1744150000123)),
			Currency:      currency.USDT,
			CurrencyPair:  BTCUSDT,
			AmountChanged: -100,
			Balance:       900,
			Type:          "margin_out",
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginAccountBalanceChangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginAccountBalanceChangeHistory should return a result")
}

func TestGetIsolatedMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginFundingAccountList(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetIsolatedMarginFundingAccountList must not error")
	if mockTests {
		exp := []MarginFundingAccountItem{{Currency: currency.BTC, Available: 1.238, Lent: 3.32, TotalLent: 3.32}}
		assert.Equal(t, exp, result, "GetIsolatedMarginFundingAccountList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginFundingAccountList should return a result")
}

func TestUpdateIsolatedMarginUsersAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateIsolatedMarginUsersAutoRepaymentSetting(t.Context(), false)
	require.NoError(t, err, "UpdateIsolatedMarginUsersAutoRepaymentSetting must not error")
	assert.Equal(t, &OnOffStatus{Status: "off"}, result, "UpdateIsolatedMarginUsersAutoRepaymentSetting should return the updated setting")
}

func TestGetIsolatedMarginUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginUserAutoRepaymentSetting(t.Context())
	require.NoError(t, err, "GetIsolatedMarginUserAutoRepaymentSetting must not error")
	if mockTests {
		assert.Equal(t, &OnOffStatus{Status: "on"}, result, "GetIsolatedMarginUserAutoRepaymentSetting should decode the setting")
		return
	}
	assert.Contains(t, []string{"on", "off"}, result.Status, "GetIsolatedMarginUserAutoRepaymentSetting should return on or off")
}

func TestGetIsolatedMarginMaxTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.EMPTYCODE, BTCUSDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetIsolatedMarginMaxTransferableAmount must reject an empty currency")

	_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginMaxTransferableAmount must require a pair for USDT")

	for _, pair := range []currency.Pair{
		currency.NewPair(currency.BTC, currency.BTC),
		currency.NewPair(currency.BTC, currency.EMPTYCODE),
	} {
		_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.BTC, pair)
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginMaxTransferableAmount must reject malformed pair %s", pair)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.Margin)
	result, err := e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, pair)
	require.NoError(t, err, "GetIsolatedMarginMaxTransferableAmount must not error")
	if mockTests {
		exp := &MaxTransferAndLoanAmount{Currency: currency.USDT, CurrencyPair: BTCUSDT, Amount: 10000}
		assert.Equal(t, exp, result, "GetIsolatedMarginMaxTransferableAmount should decode every field")
		return
	}
	assert.Equal(t, currency.USDT, result.Currency, "GetIsolatedMarginMaxTransferableAmount should return the requested currency")
}

func TestIsolatedMarginLendingMarketIsTradable(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)

	for _, tc := range []struct {
		name         string
		status       string
		delistedTime time.Time
		expected     bool
	}{
		{name: "enabled without delisting", status: "enabled", expected: true},
		{name: "disabled", status: "disabled", expected: false},
		{name: "past delisting", status: "enabled", delistedTime: now.Add(-time.Second), expected: false},
		{name: "delisting now", status: "enabled", delistedTime: now, expected: true},
		{name: "future delisting", status: "enabled", delistedTime: now.Add(time.Second), expected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			market := IsolatedMarginLendingMarket{Status: tc.status, DelistedTime: types.Time(tc.delistedTime)}
			assert.Equal(t, tc.expected, market.IsTradable(now), "tradability should match the market status and delisting time")
		})
	}
}

func TestGetIsolatedMarginLendingMarkets(t *testing.T) {
	t.Parallel()
	markets, err := e.GetIsolatedMarginLendingMarkets(t.Context())
	require.NoError(t, err, "GetIsolatedMarginLendingMarkets must not error")
	if mockTests {
		require.Len(t, markets, 7, "GetIsolatedMarginLendingMarkets must return every recorded market")
		exp := IsolatedMarginLendingMarket{
			Pair:                     BTCUSDT,
			BaseMinimumBorrowAmount:  0.00001,
			QuoteMinimumBorrowAmount: 1,
			PositionLeverage:         10,
			Status:                   "enabled",
		}
		assert.Equal(t, exp, markets[0], "GetIsolatedMarginLendingMarkets should decode every field")
		return
	}
	assert.NotEmpty(t, markets, "GetIsolatedMarginLendingMarkets should return markets")
}

func TestGetIsolatedMarginLendingMarketDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLendingMarketDetails(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginLendingMarketDetails must reject an empty pair")

	pair := getPair(t, asset.Margin)
	market, err := e.GetIsolatedMarginLendingMarketDetails(t.Context(), pair)
	require.NoError(t, err, "GetIsolatedMarginLendingMarketDetails must not error")
	if mockTests {
		exp := &IsolatedMarginLendingMarket{
			Pair:                     BTCUSDT,
			BaseMinimumBorrowAmount:  0.00001,
			QuoteMinimumBorrowAmount: 1,
			PositionLeverage:         10,
			Status:                   "enabled",
		}
		assert.Equal(t, exp, market, "GetIsolatedMarginLendingMarketDetails should decode every field")
		return
	}
	assert.True(t, pair.Equal(market.Pair), "GetIsolatedMarginLendingMarketDetails should return the requested market")
}

func TestGetIsolatedMarginEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetIsolatedMarginEstimatedInterestRate must reject no currencies")

	_, err = e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetIsolatedMarginEstimatedInterestRate must reject an empty currency")

	_, err = e.GetIsolatedMarginEstimatedInterestRate(t.Context(), make([]currency.Code, 11))
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetIsolatedMarginEstimatedInterestRate must reject more than 10 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.BTC, currency.USDT})
	require.NoError(t, err, "GetIsolatedMarginEstimatedInterestRate must not error")
	if mockTests {
		exp := map[string]types.Number{"BTC": 0.000002, "USDT": 0.0000005}
		assert.Equal(t, exp, result, "GetIsolatedMarginEstimatedInterestRate should decode every rate")
		return
	}
	assert.Contains(t, result, "BTC", "GetIsolatedMarginEstimatedInterestRate should return the requested currencies")
}

func TestIsolatedMarginBorrowOrRepay(t *testing.T) {
	t.Parallel()
	pair := getPair(t, asset.Margin)
	for _, tc := range []struct {
		name string
		arg  *IsolatedBorrowRepayRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty pair", arg: &IsolatedBorrowRepayRequest{Currency: currency.BTC, Type: "borrow", Amount: 1}, err: currency.ErrCurrencyPairEmpty},
		{name: "empty currency", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Type: "borrow", Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "invalid type", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Currency: currency.BTC, Type: "invalid", Amount: 1}, err: errInvalidIsolatedMarginLoanType},
		{name: "zero amount", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Currency: currency.BTC, Type: "borrow"}, err: order.ErrAmountIsInvalid},
		{name: "amount with full repayment", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Currency: currency.BTC, Type: "repay", Amount: 1, RepaidAll: true}, err: errAmountOverriddenByRepaidAll},
		{name: "negative amount with full repayment", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Currency: currency.BTC, Type: "repay", Amount: -1, RepaidAll: true}, err: errAmountOverriddenByRepaidAll},
		{name: "full repayment on a borrow", arg: &IsolatedBorrowRepayRequest{CurrencyPair: pair, Currency: currency.BTC, Type: "borrow", RepaidAll: true}, err: errInvalidRepaidAllOperation},
	} {
		assert.ErrorIsf(t, e.IsolatedMarginBorrowOrRepay(t.Context(), tc.arg), tc.err, "IsolatedMarginBorrowOrRepay should reject a request with %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: pair,
		Currency:     currency.BTC,
		Type:         "repay",
		Amount:       0.00004,
	}), "IsolatedMarginBorrowOrRepay should not error")
}

func TestIsolatedBorrowRepayRequestFullRepayment(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(&IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "repay",
		RepaidAll:    true,
	})
	require.NoError(t, err, "marshalling a full repayment request must not error")
	assert.JSONEq(t, `{"currency_pair":"BTC_USDT","currency":"BTC","type":"repay","amount":"","repaid_all":true}`, string(payload), "full repayment request should include repaid_all")
}

func TestGetIsolatedMarginLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLoans(t.Context(), currency.BTC, BTCUSDT, 0, 101)
	require.ErrorIs(t, err, errInvalidLimit, "GetIsolatedMarginLoans must reject a limit above 100")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginLoans(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err, "GetIsolatedMarginLoans must not error")
	if mockTests {
		exp := []IsolatedMarginLoanResponse{{
			Type:         "margin",
			CurrencyPair: BTCUSDT,
			Currency:     currency.USDT,
			Amount:       100,
			CreateTime:   types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:   types.Time(time.UnixMilli(1744190254944)),
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginLoans should return a result")
}

func TestGetIsolatedMarginLoanRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLoanRecords(t.Context(), currency.BTC, BTCUSDT, 0, 101, "")
	require.ErrorIs(t, err, errInvalidLimit, "GetIsolatedMarginLoanRecords must reject a limit above 100")

	_, err = e.GetIsolatedMarginLoanRecords(t.Context(), currency.BTC, BTCUSDT, 0, 0, "margin")
	require.ErrorIs(t, err, errInvalidIsolatedMarginLoanType, "GetIsolatedMarginLoanRecords must reject a type other than borrow or repay")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginLoanRecords(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, "")
	require.NoError(t, err, "GetIsolatedMarginLoanRecords must not error")
	if mockTests {
		exp := []IsolatedMarginLoanResponse{{
			Type:         "borrow",
			CurrencyPair: BTCUSDT,
			Currency:     currency.USDT,
			Amount:       1000,
			CreateTime:   types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginLoanRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginLoanRecords should return a result")
}

func TestGetIsolatedMarginInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.BTC, BTCUSDT, 0, 1001, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidLimit, "GetIsolatedMarginInterestDeductionRecords must reject a limit above 1000")
	tn := time.Now()
	_, err = e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.BTC, BTCUSDT, 0, 0, tn.Add(time.Hour), tn)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIsolatedMarginInterestDeductionRecords must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetIsolatedMarginInterestDeductionRecords must not error")
	if mockTests {
		exp := []InterestDeductionRecord{{
			Status:       1,
			CurrencyPair: BTCUSDT,
			Currency:     currency.USDT,
			ActualRate:   0.00000236,
			Interest:     0.00006136,
			LoanType:     "margin",
			CreateTime:   types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginInterestDeductionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginInterestDeductionRecords should return a result")
}

func TestGetIsolatedMarginMaxBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.EMPTYCODE, BTCUSDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetIsolatedMarginMaxBorrowableAmount must reject an empty currency")

	_, err = e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginMaxBorrowableAmount must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.Margin)
	result, err := e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), pair.Base, pair)
	require.NoError(t, err, "GetIsolatedMarginMaxBorrowableAmount must not error")
	if mockTests {
		exp := &MaxBorrowableAmount{Currency: currency.BTC, Borrowable: 1123.344, Pair: BTCUSDT}
		assert.Equal(t, exp, result, "GetIsolatedMarginMaxBorrowableAmount should decode every field")
		return
	}
	assert.True(t, pair.Base.Equal(result.Currency), "GetIsolatedMarginMaxBorrowableAmount should return the requested currency")
}

func TestGetIsolatedMarginUserLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginUserLeverageTiers must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	tiers, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err, "GetIsolatedMarginUserLeverageTiers must not error")
	if mockTests {
		exp := []IsolatedMarginLendingTier{
			{MaximumBorrowingLimit: 100000, MaintenanceMarginRate: 0.03, MaximumPermissibleLeverage: 10},
			{MaximumBorrowingLimit: 500000, MaintenanceMarginRate: 0.05, MaximumPermissibleLeverage: 9},
		}
		assert.Equal(t, exp, tiers, "GetIsolatedMarginUserLeverageTiers should decode every field")
		return
	}
	assert.NotEmpty(t, tiers, "GetIsolatedMarginUserLeverageTiers should return tiers")
}

func TestGetIsolatedMarginMarketLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginMarketLeverageTiers must reject an empty pair")

	tiers, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err, "GetIsolatedMarginMarketLeverageTiers must not error")
	if mockTests {
		require.Len(t, tiers, 8, "GetIsolatedMarginMarketLeverageTiers must return every recorded tier")
		exp := IsolatedMarginLendingTier{MaximumBorrowingLimit: 100000, MaintenanceMarginRate: 0.03, MaximumPermissibleLeverage: 11}
		assert.Equal(t, exp, tiers[1], "GetIsolatedMarginMarketLeverageTiers should decode every field")
		return
	}
	assert.NotEmpty(t, tiers, "GetIsolatedMarginMarketLeverageTiers should return tiers")
}

func TestSetUserMarketLeverageMultiplier(t *testing.T) {
	t.Parallel()
	err := e.SetUserMarketLeverageMultiplier(t.Context(), currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetUserMarketLeverageMultiplier must reject an empty pair")
	err = e.SetUserMarketLeverageMultiplier(t.Context(), BTCUSDT, 0)
	require.ErrorIs(t, err, errInvalidLeverage, "SetUserMarketLeverageMultiplier must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetUserMarketLeverageMultiplier(t.Context(), getPair(t, asset.Margin), 1), "SetUserMarketLeverageMultiplier should not error")
}

func TestGetIsolatedMarginAccountList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginAccountList(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetIsolatedMarginAccountList must not error")
	if mockTests {
		exp := []MarginAccountItem{{
			CurrencyPair:          BTCUSDT,
			AccountType:           "margin",
			Leverage:              3,
			RiskRate:              0.9,
			MaintenanceMarginRate: 0.03,
			Base:                  AccountBalanceInformation{Currency: currency.BTC, Available: 0.05},
			Quote:                 AccountBalanceInformation{Currency: currency.USDT, Available: 1000},
		}}
		assert.Equal(t, exp, result, "GetIsolatedMarginAccountList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginAccountList should return a result")
}

func TestGetIsolatedMarginPoolLoans(t *testing.T) {
	t.Parallel()
	// The web front-end endpoint is bot protected and cannot be recorded, so its responses are served locally
	for _, tc := range []struct {
		name string
		body string
		exp  *IsolatedMarginPoolLoanResponse
		err  error
	}{
		{name: "null", body: "null", err: common.ErrNoResponse},
		{
			name: "pools",
			body: `{"timestamp":1744150000,"method":"/apiw/v2/spot_loan/margin/margin_loan_info","code":200,"message":"Success","data":{"total":1,"list":[{"market":"BTC_USDT","deal":"0","leverage":"10","money_total_lend_available":"1000000","stock_total_lend_available":"12.5"}],"vip_settings":[{"vip_level":1,"margin_multiple":"1.2"}]}}`,
			exp: &IsolatedMarginPoolLoanResponse{
				Timestamp: types.Time(time.Unix(1744150000, 0)),
				Method:    "/apiw/v2/spot_loan/margin/margin_loan_info",
				Code:      200,
				Message:   "Success",
				Data: IsolatedMarginPoolLoanData{
					Total:       1,
					List:        []Loan{{Market: BTCUSDT, Leverage: 10, MoneyTotalLendAvailable: 1000000, StockTotalLendAvailable: 12.5}},
					VIPSettings: []VIPSetting{{VIPLevel: 1, MarginMultiple: 1.2}},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/spot_loan/margin/margin_loan_info", r.URL.Path, "pool loans request should reach the front-end endpoint")
				assert.Equal(t, "BTC", r.URL.Query().Get("search_coin"), "pool loans request should filter by coin")
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "pool loans response should be written")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.EdgeCase1.String(), server.URL+"/"), "SetRunningURL must not error")

			got, err := ex.GetIsolatedMarginPoolLoans(t.Context(), currency.BTC, 0, 100)
			require.ErrorIs(t, err, tc.err, "GetIsolatedMarginPoolLoans must return the expected error")
			assert.Equal(t, tc.exp, got, "GetIsolatedMarginPoolLoans should decode the response")
		})
	}

	if mockTests {
		return
	}
	ctx := request.WithHeaders(t.Context(), http.Header{
		"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Sec-Ch-Ua":                 {`"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {`"Windows"`},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"none"},
		"Sec-Fetch-User":            {"?1"},
		"Upgrade-Insecure-Requests": {"1"},
	})
	_, err := e.GetIsolatedMarginPoolLoans(ctx, currency.BTC, 18446744073709551615, 100)
	require.ErrorContains(t, err, "Type mismatch", "page value above the API range must return a type mismatch")

	got, err := e.GetIsolatedMarginPoolLoans(ctx, currency.BTC, 0, 100)
	if err != nil {
		require.ErrorContains(t, err, "504", "pool loan request failure must be a gateway timeout")
		return
	}
	require.NoError(t, err, "GetIsolatedMarginPoolLoans must not error")
	require.NotEmpty(t, got, "GetIsolatedMarginPoolLoans must return some loans")
}
