package gate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUnifiedAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccount(t.Context(), currency.EMPTYCODE, "")
	require.NoError(t, err, "GetUnifiedAccount must not error")
	if mockTests {
		exp := &UnifiedUserAccountResponse{
			Mode:        "multi_currency",
			UserID:      10001,
			RefreshTime: types.Time(time.UnixMilli(1744190254944)),
			Balances: map[currency.Code]BalanceDetails{
				currency.BTC: {
					Available:         0.5,
					Freeze:            0.1,
					Borrowed:          0.02,
					Equity:            0.58,
					TotalFreeze:       0.1,
					TotalLiabilities:  0.02,
					Funding:           0.01,
					FundingVersion:    3,
					CrossBalance:      0.5,
					EnabledCollateral: true,
					BalanceVersion:    42,
				},
				currency.USDT: {Available: 1000, Equity: 1000, CrossBalance: 1000, BalanceVersion: 7},
			},
			Total:                          48050,
			Borrowed:                       1600,
			TotalInitialMargin:             1025.05,
			TotalMarginBalance:             48050,
			TotalMaintenanceMargin:         205.01,
			TotalInitialMarginRate:         46.87,
			TotalMaintenanceMarginRate:     234.37,
			TotalAvailableMargin:           47024.95,
			UnifiedAccountTotal:            48050,
			UnifiedAccountTotalLiabilities: 1600,
			UnifiedAccountTotalEquity:      46450,
			Leverage:                       2,
			SpotOrderLoss:                  12,
			OptionsOrderLoss:               3,
			UseFunding:                     true,
		}
		assert.Equal(t, exp, result, "GetUnifiedAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUnifiedAccount should return the account")
}

func TestGetMaximumBorrowableAmountUnifiedAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaximumBorrowableAmountUnifiedAccount must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.ETH)
	require.NoError(t, err, "GetMaximumBorrowableAmountUnifiedAccount must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.ETH, Amount: 10000}
		assert.Equal(t, exp, result, "GetMaximumBorrowableAmountUnifiedAccount should decode every field")
		return
	}
	assert.Equal(t, currency.ETH, result.Currency, "GetMaximumBorrowableAmountUnifiedAccount should return the requested currency")
}

func TestGetUnifiedAccountMaximumTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedAccountMaximumTransferableAmount must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.ETH)
	require.NoError(t, err, "GetUnifiedAccountMaximumTransferableAmount must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.ETH, Amount: 250.5}
		assert.Equal(t, exp, result, "GetUnifiedAccountMaximumTransferableAmount should decode every field")
		return
	}
	assert.Equal(t, currency.ETH, result.Currency, "GetUnifiedAccountMaximumTransferableAmount should return the requested currency")
}

func TestGetMultipleTransferableAmountForUnifiedAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetMultipleTransferableAmountForUnifiedAccounts must reject no currencies")

	_, err = e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultipleTransferableAmountForUnifiedAccounts must reject an empty currency")

	_, err = e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), make([]currency.Code, 101)...)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultipleTransferableAmountForUnifiedAccounts must reject more than 100 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err, "GetMultipleTransferableAmountForUnifiedAccounts must not error")
	if mockTests {
		exp := []*CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 1.25}, {Currency: currency.ETH, Amount: 250.5}}
		assert.Equal(t, exp, result, "GetMultipleTransferableAmountForUnifiedAccounts should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetMultipleTransferableAmountForUnifiedAccounts should return each requested currency")
}

func TestGetBatchUnifiedAccountMaximumBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject no currencies")

	_, err = e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject an empty currency")

	_, err = e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), make([]currency.Code, 11)...)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetBatchUnifiedAccountMaximumBorrowableAmount must reject more than 10 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err, "GetBatchUnifiedAccountMaximumBorrowableAmount must not error")
	if mockTests {
		exp := []*CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 0.5}, {Currency: currency.ETH, Amount: 12}}
		assert.Equal(t, exp, result, "GetBatchUnifiedAccountMaximumBorrowableAmount should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetBatchUnifiedAccountMaximumBorrowableAmount should return each requested currency")
}

func TestBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := e.BorrowOrRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BorrowOrRepay must reject a nil request")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "BorrowOrRepay must reject an empty currency")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "BorrowOrRepay must reject an empty type")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "lend", Amount: 1})
	require.ErrorIs(t, err, errInvalidLoanType, "BorrowOrRepay must reject a type other than borrow or repay")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "borrow"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "BorrowOrRepay must reject a missing amount")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "borrow", RepaidAll: true})
	require.ErrorIs(t, err, errInvalidRepaidAllOperation, "BorrowOrRepay must reject a full repayment of a borrow")

	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "repay", Amount: 1, RepaidAll: true})
	require.ErrorIs(t, err, errAmountOverriddenByRepaidAll, "BorrowOrRepay must reject an amount with a full repayment")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Amount: 2, Currency: currency.ETH, Type: "borrow"})
	require.NoError(t, err, "BorrowOrRepay must not error")
	if mockTests {
		assert.Equal(t, &BorrowOrRepayResponse{TransactionID: 9527}, result, "BorrowOrRepay should decode the transaction ID")
	}

	result, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayRequest{Currency: currency.ETH, Type: "repay", RepaidAll: true})
	require.NoError(t, err, "BorrowOrRepay must accept a full repayment without an amount")
	if mockTests {
		assert.Equal(t, &BorrowOrRepayResponse{TransactionID: 9528}, result, "BorrowOrRepay should decode the full repayment's transaction ID")
	}
}

func TestGetLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLoans must reject a nil request")

	_, err = e.GetLoans(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "repay"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetLoans must reject a loan record type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLoans(t.Context(), &UnifiedLoansRequest{Currency: currency.USDT, Type: "margin", Page: 1, Limit: 100})
	require.NoError(t, err, "GetLoans must not error")
	if mockTests {
		exp := []*UnifiedLoanResponse{{
			Currency:     currency.USDT,
			CurrencyPair: BTCUSDT,
			Amount:       100,
			Type:         "margin",
			CreateTime:   types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:   types.Time(time.UnixMilli(1744190254944)),
		}}
		assert.Equal(t, exp, result, "GetLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLoans should return a result")
}

func TestGetLoanRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLoanRecords must reject a nil request")

	_, err = e.GetLoanRecords(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "margin"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetLoanRecords must reject a loan type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLoanRecords(t.Context(), &UnifiedLoansRequest{Currency: currency.ETH, Type: "repay", Page: 1, Limit: 100})
	require.NoError(t, err, "GetLoanRecords must not error")
	if mockTests {
		exp := []*UnifiedLoanRecordResponse{{
			ID:            16442,
			Type:          "repay",
			RepaymentType: "auto_repay",
			BorrowType:    "auto_borrow",
			MarginMode:    "cross",
			CurrencyPair:  currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter),
			Currency:      currency.ETH,
			Amount:        0.5,
			CreateTime:    types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetLoanRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLoanRecords should return a result")
}

func TestGetLoansHelper(t *testing.T) {
	t.Parallel()
	var result []*UnifiedLoanResponse
	err := e.getLoans(t.Context(), unifiedLoansEPL, "unified/loans", &UnifiedLoansRequest{Limit: 101}, &result)
	require.ErrorIs(t, err, errInvalidLimit, "getLoans must reject a limit above 100")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	err = e.getLoans(t.Context(), unifiedLoansEPL, "unified/loans", &UnifiedLoansRequest{Currency: currency.USDT, Type: "margin", Page: 1, Limit: 100}, &result)
	require.NoError(t, err, "getLoans must not error")
	if mockTests {
		require.Len(t, result, 1, "getLoans must decode the recorded loan")
		assert.Equal(t, currency.USDT, result[0].Currency, "getLoans should decode into the supplied result")
		return
	}
	assert.NotNil(t, result, "getLoans should decode a result")
}

func TestGetInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetInterestDeductionRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInterestDeductionRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "platform", StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInterestDeductionRecords must reject a start after the end")

	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Limit: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetInterestDeductionRecords must reject a limit above 100")

	_, err = e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "invalid"})
	require.ErrorIs(t, err, errInvalidLoanType, "GetInterestDeductionRecords must reject an unknown loan type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInterestDeductionRecords(t.Context(), &UnifiedInterestRecordsRequest{Currency: currency.BTC, Type: "platform", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetInterestDeductionRecords must not error")
	if mockTests {
		exp := []*InterestDeductionRecordResponse{{
			Status:       1,
			CurrencyPair: BTCUSDT,
			Currency:     currency.BTC,
			ActualRate:   0.00000236,
			Interest:     0.00006136,
			LoanType:     "platform",
			CreateTime:   types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetInterestDeductionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInterestDeductionRecords should return a result")
}

func TestSetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	err := e.SetUnifiedAccountMode(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUnifiedAccountMode must reject a nil request")

	err = e.SetUnifiedAccountMode(t.Context(), &UnifiedAccountModeRequest{Settings: &UnifiedAccountModeSettingsRequest{}})
	require.ErrorIs(t, err, errMissingUnifiedAccountMode, "SetUnifiedAccountMode must reject an empty mode")

	arg := &UnifiedAccountModeRequest{Mode: "portfolio", Settings: &UnifiedAccountModeSettingsRequest{SpotHedge: new(true), USDTFutures: new(false)}}
	payload, err := json.Marshal(arg)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"mode":"portfolio","settings":{"spot_hedge":true,"usdt_futures":false}}`, string(payload), "SetUnifiedAccountMode should send a disabled switch and omit unset ones")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetUnifiedAccountMode(t.Context(), arg), "SetUnifiedAccountMode should not error")
}

func TestGetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountMode(t.Context())
	require.NoError(t, err, "GetUnifiedAccountMode must not error")
	if mockTests {
		exp := &UnifiedAccountModeResponse{
			Mode:     "portfolio",
			Settings: UnifiedAccountModeSettings{USDTFutures: true, SpotHedge: true, Options: true},
		}
		assert.Equal(t, exp, result, "GetUnifiedAccountMode should decode every field")
		return
	}
	assert.NotEmpty(t, result.Mode, "GetUnifiedAccountMode should return the mode")
}

func TestGetUnifiedAccountEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetUnifiedAccountEstimatedInterestRate must reject no currencies")

	_, err = e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []currency.Code{currency.BTC, currency.USDT, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedAccountEstimatedInterestRate must reject an empty currency")

	_, err = e.GetUnifiedAccountEstimatedInterestRate(t.Context(), make([]currency.Code, 11))
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetUnifiedAccountEstimatedInterestRate must reject more than 10 currencies")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []currency.Code{currency.BTC, currency.USDT, currency.ETH})
	require.NoError(t, err, "GetUnifiedAccountEstimatedInterestRate must not error")
	if mockTests {
		exp := map[currency.Code]types.Number{currency.BTC: 0.000002, currency.USDT: 0.000001, currency.ETH: 0}
		assert.Equal(t, exp, result, "GetUnifiedAccountEstimatedInterestRate should decode every rate, an unsupported currency's as zero")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountEstimatedInterestRate should return rates")
}

func TestTierUpperLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   TierUpperLimit
	}{
		{input: `"+"`, exp: TierUpperLimit(math.Inf(1))},
		{input: `""`, exp: TierUpperLimit(math.Inf(1))},
		{input: `"100000"`, exp: 100000},
		{input: `500000.5`, exp: 500000.5},
	} {
		var got TierUpperLimit
		require.NoErrorf(t, got.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, got, "UnmarshalJSON should decode %s", tc.input)
		assert.Equalf(t, float64(tc.exp), got.Float64(), "Float64 should return the decoded boundary for %s", tc.input)
	}

	var got TierUpperLimit
	assert.Error(t, got.UnmarshalJSON([]byte(`"unbounded"`)), "UnmarshalJSON should reject a non-numeric boundary")

	payload, err := TierUpperLimit(math.Inf(1)).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error for an unbounded tier")
	assert.Equal(t, `"+"`, string(payload), "MarshalJSON should encode an unbounded tier as Gate does")

	payload, err = json.Marshal([]TierUpperLimit{100000})
	require.NoError(t, err, "Marshal must not error for a bounded tier")
	assert.Equal(t, `["100000"]`, string(payload), "Marshal should encode a bounded tier as a quoted number")
}

func TestGetUnifiedAccountTiered(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedAccountTiered(t.Context())
	require.NoError(t, err, "GetUnifiedAccountTiered must not error")
	if mockTests {
		exp := []*UnifiedAccountTieredDetailResponse{
			{
				Currency:      currency.USDT,
				DiscountTiers: []*UnifiedDiscountTier{{Tier: "1", Discount: 1, UpperLimit: TierUpperLimit(math.Inf(1))}},
			},
			{
				Currency: currency.USDC,
				DiscountTiers: []*UnifiedDiscountTier{
					{Tier: "1", Discount: 1, UpperLimit: 10000000},
					{Tier: "2", Discount: 0.98, LowerLimit: 10000000, UpperLimit: 15000000},
					{Tier: "3", Discount: 0.95, LowerLimit: 15000000, UpperLimit: 20000000},
					{Tier: "4", Discount: 0.925, LowerLimit: 20000000, UpperLimit: 50000000},
					{Tier: "5", Discount: 0.9, LowerLimit: 50000000, UpperLimit: 100000000},
					{Tier: "6", LowerLimit: 100000000, UpperLimit: TierUpperLimit(math.Inf(1))},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUnifiedAccountTiered should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountTiered should return tiers")
}

func TestGetUnifiedAccountTieredLoanMargin(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedAccountTieredLoanMargin(t.Context())
	require.NoError(t, err, "GetUnifiedAccountTieredLoanMargin must not error")
	if mockTests {
		exp := []*UnifiedAccountLoanMarginResponse{{
			Currency: currency.USDT,
			MarginTiers: []*UnifiedMarginTier{
				{Tier: "1", MarginRate: 0.01, UpperLimit: 100000, Leverage: 20},
				{Tier: "2", MarginRate: 0.02, LowerLimit: 100000, UpperLimit: 500000, Leverage: 10},
				{Tier: "3", MarginRate: 0.04, LowerLimit: 500000, UpperLimit: 1000000, Leverage: 8},
				{Tier: "4", MarginRate: 0.06, LowerLimit: 1000000, UpperLimit: 2000000, Leverage: 5},
				{Tier: "5", MarginRate: 0.08, LowerLimit: 2000000, UpperLimit: 5000000, Leverage: 4},
				{Tier: "6", MarginRate: 0.1, LowerLimit: 5000000, UpperLimit: 10000000, Leverage: 2.5},
				{Tier: "7", MarginRate: 0.15, LowerLimit: 10000000, UpperLimit: 20000000, Leverage: 1.5},
				{Tier: "8", MarginRate: 0.3, LowerLimit: 20000000, UpperLimit: TierUpperLimit(math.Inf(1))},
			},
		}}
		assert.Equal(t, exp, result, "GetUnifiedAccountTieredLoanMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedAccountTieredLoanMargin should return tiers")
}

func TestCalculatePortfolioMargin(t *testing.T) {
	t.Parallel()
	spotPair, futuresPair, optionsPair := getPair(t, asset.Spot), getPair(t, asset.USDTMarginedFutures), getPair(t, asset.Options)
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginCalculatorRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "nil spot balance", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{nil}}, err: common.ErrNilPointer},
		{name: "spot balance without currency", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{{Equity: 1000}}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "spot balance without equity", arg: &PortfolioMarginCalculatorRequest{SpotBalances: []*SpotBalanceParams{{Currency: currency.BTC}}}, err: errInvalidEquity},
		{name: "nil spot order", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{nil}}, err: common.ErrNilPointer},
		{name: "spot order without pair", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{OrderPrice: 60000, Left: 0.1, Side: order.Buy}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "spot order without price", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, Left: 0.1, Side: order.Buy}}}, err: limits.ErrPriceBelowMin},
		{name: "spot order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Side: order.Buy}}}, err: errInvalidOrderSize},
		{name: "spot order without side", arg: &PortfolioMarginCalculatorRequest{SpotOrders: []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Left: 0.1}}}, err: order.ErrSideIsInvalid},
		{name: "nil futures position", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "futures position without contract", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{{Size: -1}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "futures position without size", arg: &PortfolioMarginCalculatorRequest{FuturesPositions: []*FuturesOrderInfo{{Contract: futuresPair}}}, err: errInvalidOrderSize},
		{name: "nil futures order", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "futures order without contract", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Size: 1, Left: 0.5}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "futures order without size", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Contract: futuresPair, Left: 0.5}}}, err: errInvalidOrderSize},
		{name: "futures order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{FuturesOrders: []*FuturesOrderInfo{{Contract: futuresPair, Size: 1}}}, err: errInvalidOrderSize},
		{name: "nil options position", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "options position without name", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{{Size: 1}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "options position without size", arg: &PortfolioMarginCalculatorRequest{OptionsPositions: []*OptionsOrderInfo{{OptionsName: optionsPair}}}, err: errInvalidOrderSize},
		{name: "nil options order", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{nil}}, err: common.ErrNilPointer},
		{name: "options order without name", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{Size: 1, Left: 0.5}}}, err: currency.ErrCurrencyPairEmpty},
		{name: "options order without size", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{OptionsName: optionsPair, Left: 0.5}}}, err: errInvalidOrderSize},
		{name: "options order without unfilled quantity", arg: &PortfolioMarginCalculatorRequest{OptionsOrders: []*OptionsOrderInfo{{OptionsName: optionsPair, Size: 1}}}, err: errInvalidOrderSize},
	} {
		_, err := e.CalculatePortfolioMargin(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "CalculatePortfolioMargin must reject a request with a %s", tc.name)
	}

	// Short positions and sell orders carry negative sizes
	arg := &PortfolioMarginCalculatorRequest{
		SpotBalances:     []*SpotBalanceParams{{Currency: currency.BTC, Equity: -1}},
		SpotOrders:       []*SpotOrderParams{{CurrencyPair: spotPair, OrderPrice: 60000, Count: 1, Left: 1, Side: order.Sell}},
		FuturesPositions: []*FuturesOrderInfo{{Contract: futuresPair, Size: -100}},
		FuturesOrders:    []*FuturesOrderInfo{{Contract: futuresPair, Size: -10, Left: -8}},
	}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CalculatePortfolioMargin(t.Context(), arg)
	require.NoError(t, err, "CalculatePortfolioMargin must not error")
	if mockTests {
		exp := &PortfolioMarginCalculationResponse{
			MaintainMarginTotal: 123.171795,
			InitialMarginTotal:  160.1233335,
			CalculateTime:       types.Time(time.Unix(1791854278, 0)),
			RiskUnits: []*PortfolioMarginRiskUnit{{
				Symbol:         currency.BTC,
				MaintainMargin: 123.171795,
				InitialMargin:  160.1233335,
				MarginResults: []*PortfolioMarginResult{{
					Type: "original_position",
					ProfitLossRanges: []*VolatilityInfo{
						{PricePercentage: -0.15, ImpliedVolatilityPercentage: -0.25, ProfitLoss: 123.171795},
						{PricePercentage: -0.12, ImpliedVolatilityPercentage: -0.25, ProfitLoss: 98.537436},
					},
					MaxLoss:    &VolatilityInfo{PricePercentage: 0.15, ImpliedVolatilityPercentage: -0.25, ProfitLoss: -123.171795},
					StressTest: 123.171795,
				}},
				Delta: -0.01,
			}},
		}
		assert.Equal(t, exp, result, "CalculatePortfolioMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result.RiskUnits, "CalculatePortfolioMargin should return risk units")
}

func TestSpotOrderParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := (&SpotOrderParams{CurrencyPair: BTCUSDT, OrderPrice: 60000, Count: 1, Left: 0.5, Side: order.Buy}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"currency_pairs":"BTC_USDT","order_price":"60000","count":"1","left":"0.5","type":"buy"}`, string(payload), "MarshalJSON should send the order side in lower case as the type")
}

func TestSetAllLoanCurrenciesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetAllLoanCurrenciesLeverage(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidLeverage, "SetAllLoanCurrenciesLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAllLoanCurrenciesLeverage(t.Context(), 10)
	require.NoError(t, err, "SetAllLoanCurrenciesLeverage must not error")
	if mockTests {
		exp := []*LeverageFailedCurrencyResponse{{Currency: currency.USDT, Reason: "leverage cannot be changed with an outstanding loan"}}
		assert.Equal(t, exp, result, "SetAllLoanCurrenciesLeverage should decode every failed currency")
		return
	}
	assert.NotNil(t, result, "SetAllLoanCurrenciesLeverage should return the failed currencies")
}

func TestGetMaxMinCurrencyLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxMinCurrencyLeverage(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxMinCurrencyLeverage must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxMinCurrencyLeverage(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxMinCurrencyLeverage must not error")
	if mockTests {
		exp := &UnifiedCurrencyLeverageConfigResponse{CurrentLeverage: 2, MaxLeverage: 8}
		assert.Equal(t, exp, result, "GetMaxMinCurrencyLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMaxMinCurrencyLeverage should return the leverage limits")
}

func TestGetUserCurrencyLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserCurrencyLeverage(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetUserCurrencyLeverage must not error")
	if mockTests {
		exp := []*UnifiedLeverageSettingResponse{{Currency: currency.BTC, Leverage: 3}, {Currency: currency.ETH, Leverage: 2}}
		assert.Equal(t, exp, result, "GetUserCurrencyLeverage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUserCurrencyLeverage should return currency leverages")
}

func TestSetLoanCurrencyLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetLoanCurrencyLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetLoanCurrencyLeverage must reject a nil request")

	err = e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetLoanCurrencyLeverage must reject an empty currency")

	err = e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidLeverage, "SetLoanCurrencyLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetLoanCurrencyLeverage(t.Context(), &UnifiedLeverageSettingResponse{Currency: currency.BTC, Leverage: 3}), "SetLoanCurrencyLeverage should not error")
}

func TestGetUnifiedLoanCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetUnifiedLoanCurrencies(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetUnifiedLoanCurrencies must not error")
	if mockTests {
		exp := []*UnifiedLoanCurrencyResponse{{
			Name:                     currency.BTC,
			Precision:                0.000001,
			MinBorrowAmount:          0.01,
			UserMaxBorrowAmountUSDT:  1000000,
			TotalMaxBorrowAmountUSDT: 1000000,
			LoanStatus:               "enable",
		}}
		assert.Equal(t, exp, result, "GetUnifiedLoanCurrencies should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUnifiedLoanCurrencies should return currencies")
}

func TestGetUnifiedHistoricalLendingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedHistoricalLendingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUnifiedHistoricalLendingRates must reject a nil request")

	_, err = e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUnifiedHistoricalLendingRates must reject an empty currency")

	_, err = e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{Currency: currency.USDT, Limit: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetUnifiedHistoricalLendingRates must reject a limit above 100")

	result, err := e.GetUnifiedHistoricalLendingRates(t.Context(), &UnifiedHistoricalLendingRatesRequest{Currency: currency.USDT})
	require.NoError(t, err, "GetUnifiedHistoricalLendingRates must not error")
	if mockTests {
		exp := &UnifiedHistoricalLendingRateResponse{
			Currency:   currency.USDT,
			Tier:       "1",
			TierUpRate: 1.18,
			Rates:      []*UnifiedLendingRate{{Time: types.Time(time.UnixMilli(1729476160000)), Rate: 0.00010287}},
		}
		assert.Equal(t, exp, result, "GetUnifiedHistoricalLendingRates should decode every field")
		return
	}
	assert.NotEmpty(t, result.Rates, "GetUnifiedHistoricalLendingRates should return rates")
}

func TestSetUnifiedCollateralCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.SetUnifiedCollateralCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUnifiedCollateralCurrency must reject a nil request")

	_, err = e.SetUnifiedCollateralCurrency(t.Context(), &UnifiedSetCollateralRequest{CollateralType: 2})
	require.ErrorIs(t, err, errInvalidCollateralType, "SetUnifiedCollateralCurrency must reject an unknown collateral type")

	payload, err := json.Marshal(&UnifiedSetCollateralRequest{})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"collateral_type":0}`, string(payload), "SetUnifiedCollateralCurrency should send the all currencies collateral type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUnifiedCollateralCurrency(t.Context(), &UnifiedSetCollateralRequest{CollateralType: 1, EnableList: []currency.Code{currency.BTC, currency.ETH}})
	require.NoError(t, err, "SetUnifiedCollateralCurrency must not error")
	if mockTests {
		assert.Equal(t, &UnifiedSetCollateralResponse{IsSuccess: true}, result, "SetUnifiedCollateralCurrency should decode the result")
		return
	}
	assert.True(t, result.IsSuccess, "SetUnifiedCollateralCurrency should succeed")
}

func TestEstimateQuickRepaymentDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.EstimateQuickRepaymentDetails(t.Context())
	require.NoError(t, err, "EstimateQuickRepaymentDetails must not error")
	if mockTests {
		exp := &UnifiedQuickEstimateRepaymentResponse{
			DebtCurrencies:      []*UnifiedDebtCurrency{{Currency: currency.ETH, DebtAmount: 1.5, EstimatedUSD: 4500, Borrowed: 1.5}},
			AvailableCurrencies: []*UnifiedAvailableCurrency{{Currency: currency.USDT, Available: 5000, EstimatedUSD: 5000}},
		}
		assert.Equal(t, exp, result, "EstimateQuickRepaymentDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "EstimateQuickRepaymentDetails should return an estimate")
}

func TestQuickRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.QuickRepayment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QuickRepayment must reject a nil request")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "QuickRepayment must reject missing debt currencies")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{DebtCurrencies: []currency.Code{currency.USDT}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "QuickRepayment must reject missing available currencies")

	_, err = e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{DebtCurrencies: []currency.Code{currency.USDT}, AvailableCurrencies: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "QuickRepayment must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.QuickRepayment(t.Context(), &UnifiedQuickRepaymentRequest{
		DebtCurrencies:      []currency.Code{currency.USDT, currency.ETH},
		AvailableCurrencies: []currency.Code{currency.USDT, currency.BTC},
	})
	require.NoError(t, err, "QuickRepayment must not error")
	if mockTests {
		exp := &UnifiedQuickRepaymentResponse{
			OrderID:     "qr_123456",
			RepaidInfos: []*UnifiedRepaidInfo{{Currency: currency.USDT, Repaid: 100}, {Currency: currency.ETH, Repaid: 0.5, Left: 1}},
			UsedInfos:   []*UnifiedUsedInfo{{Currency: currency.USDT, Used: 1600}},
		}
		assert.Equal(t, exp, result, "QuickRepayment should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "QuickRepayment should return an order ID")
}

func TestSetUnifiedDeltaNeutral(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUnifiedDeltaNeutral(t.Context(), true)
	require.NoError(t, err, "SetUnifiedDeltaNeutral must not error")
	assert.Equal(t, &DeltaNeutralResponse{Enabled: true}, result, "SetUnifiedDeltaNeutral should return the enabled state")
}

func TestGetUnifiedDeltaNeutral(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnifiedDeltaNeutral(t.Context())
	require.NoError(t, err, "GetUnifiedDeltaNeutral must not error")
	if mockTests {
		assert.Equal(t, &DeltaNeutralResponse{Enabled: true}, result, "GetUnifiedDeltaNeutral should decode the setting")
		return
	}
	assert.NotNil(t, result, "GetUnifiedDeltaNeutral should return the setting")
}
