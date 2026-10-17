package bitfinex

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// liveFundingOffer is an offer to provide funding at a daily rate far above the market's, so it is not taken while a
// live test holds it
func liveFundingOffer() *SubmitFundingOfferRequest {
	return &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUSD", Amount: 150, Rate: 0.01, Period: 2}
}

// liveKeepFundingRequest returns a request toggling whether one of the account's taken credits or loans is kept,
// skipping a live test when it has neither
func liveKeepFundingRequest(t *testing.T) *KeepFundingRequest {
	t.Helper()
	credits, err := e.GetFundingCredits(t.Context(), "")
	require.NoError(t, err, "GetFundingCredits must not error")
	loans, err := e.GetFundingLoans(t.Context(), "")
	require.NoError(t, err, "GetFundingLoans must not error")
	for _, taken := range []struct {
		fundingType string
		records     []FundingCredit
	}{{KeepFundingTypeCredit, credits}, {KeepFundingTypeLoan, loans}} {
		for i := range taken.records {
			// Only funding the account borrows can be kept
			if taken.records[i].Side <= 0 {
				return &KeepFundingRequest{Type: taken.fundingType, IDs: []uint64{taken.records[i].ID}}
			}
		}
	}
	t.Skip("The account has taken no funding to keep")
	return nil
}

func TestOptionalFundingSymbolPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbol, suffix, exp string
	}{
		{exp: "auth/r/funding/offers"},
		{suffix: "/hist", exp: "auth/r/funding/offers/hist"},
		{symbol: "fUSD", exp: "auth/r/funding/offers/fUSD"},
		{symbol: "fUSD", suffix: "/hist", exp: "auth/r/funding/offers/fUSD/hist"},
	} {
		path, err := optionalFundingSymbolPath("auth/r/funding/offers", tc.symbol, tc.suffix)
		require.NoErrorf(t, err, "optionalFundingSymbolPath must not error for symbol %q", tc.symbol)
		assert.Equalf(t, tc.exp, path, "optionalFundingSymbolPath should place symbol %q before suffix %q", tc.symbol, tc.suffix)
	}
	for _, symbol := range []string{"f", "tBTCUSD", "USD"} {
		_, err := optionalFundingSymbolPath("auth/r/funding/offers", symbol, "")
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "optionalFundingSymbolPath should refuse symbol %q", symbol)
	}
}

// TestOptionalFundingSymbolEndpoints checks the endpoints taking an optional funding currency request every
// currency's path without one, which the recorded fixtures, made with a currency, do not cover
func TestOptionalFundingSymbolEndpoints(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var paths []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		_, err := w.Write([]byte(`[]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err := ex.GetActiveFundingOffers(t.Context(), "")
	require.NoError(t, err, "GetActiveFundingOffers must not error")
	_, err = ex.GetFundingOffersHistory(t.Context(), &FundingOffersHistoryRequest{})
	require.NoError(t, err, "GetFundingOffersHistory must not error")
	_, err = ex.GetFundingLoans(t.Context(), "")
	require.NoError(t, err, "GetFundingLoans must not error")
	_, err = ex.GetFundingLoansHistory(t.Context(), &FundingLoansHistoryRequest{})
	require.NoError(t, err, "GetFundingLoansHistory must not error")
	_, err = ex.GetFundingCredits(t.Context(), "")
	require.NoError(t, err, "GetFundingCredits must not error")
	_, err = ex.GetFundingCreditsHistory(t.Context(), &FundingCreditsHistoryRequest{})
	require.NoError(t, err, "GetFundingCreditsHistory must not error")
	_, err = ex.GetAccountFundingTrades(t.Context(), &AccountFundingTradesRequest{})
	require.NoError(t, err, "GetAccountFundingTrades must not error")
	mu.Lock()
	defer mu.Unlock()
	exp := []string{
		"/v2/auth/r/funding/offers",
		"/v2/auth/r/funding/offers/hist",
		"/v2/auth/r/funding/loans",
		"/v2/auth/r/funding/loans/hist",
		"/v2/auth/r/funding/credits",
		"/v2/auth/r/funding/credits/hist",
		"/v2/auth/r/funding/trades/hist",
	}
	assert.Equal(t, exp, paths, "An empty symbol should request every currency's path")
}

func TestGetActiveFundingOffers(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"f", "tBTCUSD", "USD"} {
		_, err := e.GetActiveFundingOffers(t.Context(), symbol)
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "GetActiveFundingOffers should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	symbol := "fETH"
	if !mockTests {
		symbol = ""
	}
	result, err := e.GetActiveFundingOffers(t.Context(), symbol)
	require.NoError(t, err, "GetActiveFundingOffers must not error")
	if mockTests {
		exp := []FundingOffer{{
			ID:             652606505,
			Symbol:         "fETH",
			CreatedAt:      types.Time(time.UnixMilli(1574000611000)),
			UpdatedAt:      types.Time(time.UnixMilli(1574000611000)),
			Amount:         0.29797676,
			OriginalAmount: 0.29797676,
			Type:           FundingOfferTypeLimit,
			Status:         "ACTIVE",
			Rate:           0.0002,
			Period:         2,
		}}
		assert.Equal(t, exp, result, "GetActiveFundingOffers should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetActiveFundingOffers should return offer %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetActiveFundingOffers(t.Context(), symbol)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetActiveFundingOffers should return the request failure")
}

func TestSubmitFundingOffer(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitFundingOffer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitFundingOffer must reject a nil request")
	for _, offerType := range []string{"", OrderTypeMarket, "limit"} {
		_, err = e.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: offerType, Symbol: "fUSD", Amount: 150, Rate: 0.0002, Period: 2})
		require.ErrorIsf(t, err, errInvalidFundingOfferType, "SubmitFundingOffer must refuse type %q", offerType)
	}
	_, err = e.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeFlashReturnRateDeltaVariable, Symbol: "fUSD", Amount: 150, Rate: -0.0001, Period: 2})
	require.ErrorIs(t, err, errInvalidFundingOfferRate, "SubmitFundingOffer must refuse a negative FRRDELTAVAR offset")
	for _, symbol := range []string{"", "tBTCUSD", "USD"} {
		_, err = e.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: symbol, Amount: 150, Rate: 0.0002, Period: 2})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "SubmitFundingOffer must refuse symbol %q", symbol)
	}
	_, err = e.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUSD", Rate: 0.0002, Period: 2})
	require.ErrorIs(t, err, errInvalidAmount, "SubmitFundingOffer must refuse a zero amount")
	for _, period := range []uint64{0, 1} {
		_, err = e.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUSD", Amount: 150, Rate: 0.0002, Period: period})
		require.ErrorIsf(t, err, errInvalidFundingOfferPeriod, "SubmitFundingOffer must refuse period %d", period)
	}

	// A zero rate offers at the flash return rate, so it must be sent rather than left out, and FRRDELTAFIX takes a
	// negative offset
	ex := newBodyCheckExchange(t, `[1568713496510,"fon-req",null,null,[604366339,"fUSD"],null,"SUCCESS",null]`,
		`{"type":"FRRDELTAVAR","symbol":"fUSD","amount":"150","rate":"0","period":30,"flags":64}`,
		`{"type":"FRRDELTAFIX","symbol":"fUSD","amount":"-150","rate":"-0.0001","period":2}`,
	)
	_, err = ex.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeFlashReturnRateDeltaVariable, Symbol: "fUSD", Amount: 150, Period: 30, Flags: FundingOfferFlagHidden})
	require.NoError(t, err, "SubmitFundingOffer must not error for a flash return rate offer")
	_, err = ex.SubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeFlashReturnRateDeltaFixed, Symbol: "fUSD", Amount: -150, Rate: -0.0001, Period: 2})
	require.NoError(t, err, "SubmitFundingOffer must not error for a negative FRRDELTAFIX offset")

	skipLiveUnlessCanManipulate(t)
	req := &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUSD", Amount: 50, Rate: 0.00002, Period: 2}
	if !mockTests {
		req = liveFundingOffer()
	}
	result, err := e.SubmitFundingOffer(t.Context(), req)
	require.NoError(t, err, "SubmitFundingOffer must not error")
	if mockTests {
		exp := &Notification[FundingOffer]{
			Time: types.Time(time.UnixMilli(1568713496510)),
			Type: "fon-req",
			Data: FundingOffer{
				ID:             604366339,
				Symbol:         "fUSD",
				CreatedAt:      types.Time(time.UnixMilli(1568713496502)),
				UpdatedAt:      types.Time(time.UnixMilli(1568713496502)),
				Amount:         50,
				OriginalAmount: 50,
				Type:           FundingOfferTypeLimit,
				Status:         "ACTIVE",
				Rate:           0.00002,
				Period:         2,
			},
			Status: NotificationStatusSuccess,
			Text:   "Submitting funding offer of 50.0 USD at 0.002000 for 2 days.",
		}
		assert.Equal(t, exp, result, "SubmitFundingOffer should decode every field")
	} else {
		assert.NotZero(t, result.Data.ID, "SubmitFundingOffer should return the offer's ID")
		_, err = e.CancelFundingOffer(t.Context(), result.Data.ID)
		assert.NoError(t, err, "CancelFundingOffer should cancel the offer submitted")
	}

	_, err = newResponseServerExchange(t, `null`).SubmitFundingOffer(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SubmitFundingOffer should error for a null response")
	_, err = newUnavailableServerExchange(t).SubmitFundingOffer(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitFundingOffer should return the request failure")
}

func TestCancelFundingOffer(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFundingOffer(t.Context(), 0)
	require.ErrorIs(t, err, errFundingOfferIDRequired, "CancelFundingOffer must require an offer ID")

	skipLiveUnlessCanManipulate(t)
	offerID := uint64(604393839)
	if !mockTests {
		submitted, err := e.SubmitFundingOffer(t.Context(), liveFundingOffer())
		require.NoError(t, err, "SubmitFundingOffer must not error")
		offerID = submitted.Data.ID
	}
	result, err := e.CancelFundingOffer(t.Context(), offerID)
	require.NoError(t, err, "CancelFundingOffer must not error")
	if mockTests {
		exp := &Notification[FundingOffer]{
			Time: types.Time(time.UnixMilli(1568716652310)),
			Type: "foc-req",
			Data: FundingOffer{
				ID:             604393839,
				Symbol:         "fUSD",
				CreatedAt:      types.Time(time.UnixMilli(1568716545000)),
				UpdatedAt:      types.Time(time.UnixMilli(1568716545000)),
				Amount:         50,
				OriginalAmount: 50,
				Type:           FundingOfferTypeLimit,
				Status:         "ACTIVE",
				Rate:           0.06,
				Period:         2,
			},
			Status: NotificationStatusSuccess,
		}
		assert.Equal(t, exp, result, "CancelFundingOffer should decode every field")
	} else {
		assert.Equal(t, offerID, result.Data.ID, "CancelFundingOffer should return the offer cancelled")
	}

	_, err = newResponseServerExchange(t, `null`).CancelFundingOffer(t.Context(), offerID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelFundingOffer should error for a null response")
	_, err = newUnavailableServerExchange(t).CancelFundingOffer(t.Context(), offerID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelFundingOffer should return the request failure")
}

func TestCancelAllFundingOffers(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("CancelAllFundingOffers would cancel offers the test did not submit")
	}
	exp := &Notification[json.RawMessage]{
		Time:   types.Time(time.UnixMilli(1573911972518)),
		Type:   "foc_all-req",
		Status: NotificationStatusSuccess,
		Text:   "All (8) submitted for cancellation; waiting for confirmation.",
	}
	// An empty currency is left out, cancelling the offers in every currency
	for _, c := range []currency.Code{currency.USD, currency.EMPTYCODE} {
		result, err := e.CancelAllFundingOffers(t.Context(), c)
		require.NoErrorf(t, err, "CancelAllFundingOffers must not error for currency %q", c)
		assert.Equalf(t, exp, result, "CancelAllFundingOffers should decode every field for currency %q", c)
	}

	_, err := newResponseServerExchange(t, `null`).CancelAllFundingOffers(t.Context(), currency.USD)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllFundingOffers should error for a null response")
	_, err = newUnavailableServerExchange(t).CancelAllFundingOffers(t.Context(), currency.USD)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllFundingOffers should return the request failure")
}

func TestFundingClose(t *testing.T) {
	t.Parallel()
	_, err := e.FundingClose(t.Context(), 0)
	require.ErrorIs(t, err, errFundingLoanOrCreditIDRequired, "FundingClose must require a loan or credit ID")

	if !mockTests {
		t.Skip("FundingClose returns funding taken, which a test cannot undo")
	}
	result, err := e.FundingClose(t.Context(), 2995368)
	require.NoError(t, err, "FundingClose must not error")
	exp := &Notification[json.RawMessage]{
		Time:   types.Time(time.UnixMilli(1574075056449)),
		Type:   "fr-req",
		Status: NotificationStatusSuccess,
	}
	assert.Equal(t, exp, result, "FundingClose should decode every field")

	_, err = newResponseServerExchange(t, `null`).FundingClose(t.Context(), 2995368)
	assert.ErrorIs(t, err, common.ErrNoResponse, "FundingClose should error for a null response")
	_, err = newUnavailableServerExchange(t).FundingClose(t.Context(), 2995368)
	assert.ErrorIs(t, err, request.ErrBadStatus, "FundingClose should return the request failure")
}

func TestFundingAutoRenew(t *testing.T) {
	t.Parallel()
	_, err := e.FundingAutoRenew(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FundingAutoRenew must reject a nil request")
	_, err = e.FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Activate: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FundingAutoRenew must require a currency")

	// Status 0 deactivates auto-renew, so it must be sent rather than left out
	ex := newBodyCheckExchange(t, `[1574075756185,"fa-req",null,null,["USD",30,0.0002,100],null,"SUCCESS",null]`,
		`{"status":0,"currency":"USD"}`,
		`{"status":1,"currency":"USD","amount":"100","rate":"0.0002","period":30}`,
	)
	_, err = ex.FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Currency: currency.USD})
	require.NoError(t, err, "FundingAutoRenew must not error when deactivating")
	_, err = ex.FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Activate: true, Currency: currency.USD.Lower(), Amount: 100, Rate: 0.0002, Period: 30})
	require.NoError(t, err, "FundingAutoRenew must not error when activating with every setting")

	if !mockTests {
		t.Skip("FundingAutoRenew changes a setting no endpoint reads back for a test to restore")
	}
	result, err := e.FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Activate: true, Currency: currency.UST, Amount: 350, Period: 2})
	require.NoError(t, err, "FundingAutoRenew must not error")
	exp := &Notification[FundingAutoRenewSettings]{
		Time:   types.Time(time.UnixMilli(1574075756185)),
		Type:   "fa-req",
		Data:   FundingAutoRenewSettings{Currency: currency.UST, Period: 2, Threshold: 350},
		Status: NotificationStatusSuccess,
		Text:   "Auto-renew of UST offers activated at FRR for 2 days periods",
	}
	assert.Equal(t, exp, result, "FundingAutoRenew should decode every field")

	_, err = newResponseServerExchange(t, `null`).FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Currency: currency.UST})
	assert.ErrorIs(t, err, common.ErrNoResponse, "FundingAutoRenew should error for a null response")
	_, err = newUnavailableServerExchange(t).FundingAutoRenew(t.Context(), &FundingAutoRenewRequest{Currency: currency.UST})
	assert.ErrorIs(t, err, request.ErrBadStatus, "FundingAutoRenew should return the request failure")
}

func TestKeepFunding(t *testing.T) {
	t.Parallel()
	_, err := e.KeepFunding(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "KeepFunding must reject a nil request")
	for _, fundingType := range []string{"", "offer", "Credit"} {
		_, err = e.KeepFunding(t.Context(), &KeepFundingRequest{Type: fundingType, IDs: []uint64{1}})
		require.ErrorIsf(t, err, errInvalidKeepFundingType, "KeepFunding must refuse type %q", fundingType)
	}

	// Changes sends 1 to keep and 0 to stop keeping
	_, err = newBodyCheckExchange(t, `[1573914639029,"fk-req",null,null,null,null,"SUCCESS","Loan updated"]`,
		`{"type":"loan","id":[1,2],"changes":{"1":1,"2":0}}`,
	).KeepFunding(t.Context(), &KeepFundingRequest{Type: KeepFundingTypeLoan, IDs: []uint64{1, 2}, Changes: map[uint64]bool{1: true, 2: false}})
	require.NoError(t, err, "KeepFunding must not error for loans")

	skipLiveUnlessCanManipulate(t)
	req := &KeepFundingRequest{Type: KeepFundingTypeCredit, IDs: []uint64{26222883}, Changes: map[uint64]bool{26222883: true}}
	if !mockTests {
		req = liveKeepFundingRequest(t)
	}
	result, err := e.KeepFunding(t.Context(), req)
	require.NoError(t, err, "KeepFunding must not error")
	if mockTests {
		exp := &Notification[json.RawMessage]{
			Time:   types.Time(time.UnixMilli(1573914639029)),
			Type:   "fk-req",
			Status: NotificationStatusSuccess,
			Text:   "Credit updated",
		}
		assert.Equal(t, exp, result, "KeepFunding should decode every field")
	} else {
		// Toggling the funding again restores it
		_, err = e.KeepFunding(t.Context(), req)
		assert.NoError(t, err, "KeepFunding should toggle the funding back")
	}

	_, err = newResponseServerExchange(t, `null`).KeepFunding(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "KeepFunding should error for a null response")
	_, err = newUnavailableServerExchange(t).KeepFunding(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "KeepFunding should return the request failure")
}

func TestGetFundingOffersHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingOffersHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingOffersHistory must reject a nil request")
	_, err = e.GetFundingOffersHistory(t.Context(), &FundingOffersHistoryRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errFundingSymbolRequired, "GetFundingOffersHistory must refuse a trading pair")
	_, err = e.GetFundingOffersHistory(t.Context(), &FundingOffersHistoryRequest{Start: time.UnixMilli(1574100000000), End: time.UnixMilli(1574000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingOffersHistory must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &FundingOffersHistoryRequest{Symbol: "fUSD", Start: time.UnixMilli(1574000000000), End: time.UnixMilli(1574100000000), Limit: 25}
	if !mockTests {
		req = &FundingOffersHistoryRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 25}
	}
	result, err := e.GetFundingOffersHistory(t.Context(), req)
	require.NoError(t, err, "GetFundingOffersHistory must not error")
	if mockTests {
		exp := []FundingOffer{{
			ID:             653170899,
			Symbol:         "fUSD",
			CreatedAt:      types.Time(time.UnixMilli(1574072620000)),
			UpdatedAt:      types.Time(time.UnixMilli(1574072620000)),
			OriginalAmount: -57.9,
			Status:         "EXECUTED at 0.0368% (57.9)",
			Rate:           0.000369,
			Period:         2,
		}}
		assert.Equal(t, exp, result, "GetFundingOffersHistory should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetFundingOffersHistory should return offer %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetFundingOffersHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingOffersHistory should return the request failure")
}

func TestGetFundingLoans(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"f", "tBTCUSD", "USD"} {
		_, err := e.GetFundingLoans(t.Context(), symbol)
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingLoans should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	symbol := "fUST"
	if !mockTests {
		symbol = ""
	}
	result, err := e.GetFundingLoans(t.Context(), symbol)
	require.NoError(t, err, "GetFundingLoans must not error")
	if mockTests {
		exp := []FundingCredit{{
			ID:           2995368,
			Symbol:       "fUST",
			CreatedAt:    types.Time(time.UnixMilli(1574077517000)),
			UpdatedAt:    types.Time(time.UnixMilli(1574077517000)),
			Amount:       100,
			Status:       "ACTIVE",
			RateType:     "FIXED",
			Rate:         0.0024,
			Period:       2,
			OpenedAt:     types.Time(time.UnixMilli(1574077517000)),
			LastPayoutAt: types.Time(time.UnixMilli(1574077517000)),
		}}
		assert.Equal(t, exp, result, "GetFundingLoans should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetFundingLoans should return loan %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetFundingLoans(t.Context(), symbol)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingLoans should return the request failure")
}

func TestGetFundingLoansHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingLoansHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingLoansHistory must reject a nil request")
	_, err = e.GetFundingLoansHistory(t.Context(), &FundingLoansHistoryRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errFundingSymbolRequired, "GetFundingLoansHistory must refuse a trading pair")
	_, err = e.GetFundingLoansHistory(t.Context(), &FundingLoansHistoryRequest{Start: time.UnixMilli(1702700000000), End: time.UnixMilli(1702600000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingLoansHistory must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &FundingLoansHistoryRequest{Symbol: "fUST", Start: time.UnixMilli(1702600000000), End: time.UnixMilli(1702700000000), Limit: 25}
	if !mockTests {
		// The history covers the last 3 days only
		req = &FundingLoansHistoryRequest{Start: time.Now().Add(-48 * time.Hour), End: time.Now(), Limit: 25}
	}
	result, err := e.GetFundingLoansHistory(t.Context(), req)
	require.NoError(t, err, "GetFundingLoansHistory must not error")
	if mockTests {
		exp := []FundingCredit{{
			ID:           3027653,
			Symbol:       "fUST",
			Side:         -1,
			CreatedAt:    types.Time(time.UnixMilli(1702649922000)),
			UpdatedAt:    types.Time(time.UnixMilli(1702649922000)),
			Amount:       402.10731456,
			Status:       "CLOSED",
			Period:       2,
			OpenedAt:     types.Time(time.UnixMilli(1702649922000)),
			LastPayoutAt: types.Time(time.UnixMilli(1702649958000)),
		}}
		assert.Equal(t, exp, result, "GetFundingLoansHistory should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetFundingLoansHistory should return loan %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetFundingLoansHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingLoansHistory should return the request failure")
}

func TestGetFundingCredits(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"f", "tBTCUSD", "USD"} {
		_, err := e.GetFundingCredits(t.Context(), symbol)
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingCredits should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	symbol := "fUST"
	if !mockTests {
		symbol = ""
	}
	result, err := e.GetFundingCredits(t.Context(), symbol)
	require.NoError(t, err, "GetFundingCredits must not error")
	if mockTests {
		exp := []FundingCredit{{
			ID:           26222883,
			Symbol:       "fUST",
			Side:         1,
			CreatedAt:    types.Time(time.UnixMilli(1574013661000)),
			UpdatedAt:    types.Time(time.UnixMilli(1574079687000)),
			Amount:       350,
			Status:       "ACTIVE",
			RateType:     "FIXED",
			Rate:         0.0024,
			Period:       2,
			OpenedAt:     types.Time(time.UnixMilli(1574013661000)),
			LastPayoutAt: types.Time(time.UnixMilli(1574078487000)),
			PositionPair: "tBTCUST",
		}}
		assert.Equal(t, exp, result, "GetFundingCredits should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetFundingCredits should return credit %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetFundingCredits(t.Context(), symbol)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingCredits should return the request failure")
}

func TestGetFundingCreditsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingCreditsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingCreditsHistory must reject a nil request")
	_, err = e.GetFundingCreditsHistory(t.Context(), &FundingCreditsHistoryRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errFundingSymbolRequired, "GetFundingCreditsHistory must refuse a trading pair")
	_, err = e.GetFundingCreditsHistory(t.Context(), &FundingCreditsHistoryRequest{Start: time.UnixMilli(1574500000000), End: time.UnixMilli(1574200000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingCreditsHistory must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &FundingCreditsHistoryRequest{Symbol: "fUSD", Start: time.UnixMilli(1574200000000), End: time.UnixMilli(1574500000000), Limit: 25}
	if !mockTests {
		// The history covers the last 3 days only
		req = &FundingCreditsHistoryRequest{Start: time.Now().Add(-48 * time.Hour), End: time.Now(), Limit: 25}
	}
	result, err := e.GetFundingCreditsHistory(t.Context(), req)
	require.NoError(t, err, "GetFundingCreditsHistory must not error")
	if mockTests {
		exp := []FundingCredit{{
			ID:           171988300,
			Symbol:       "fUSD",
			Side:         1,
			CreatedAt:    types.Time(time.UnixMilli(1574230085000)),
			UpdatedAt:    types.Time(time.UnixMilli(1574402835000)),
			Amount:       50.70511182,
			Status:       "CLOSED (expired)",
			RateType:     "FIXED",
			Rate:         0.00024799,
			Period:       2,
			OpenedAt:     types.Time(time.UnixMilli(1574230085000)),
			LastPayoutAt: types.Time(time.UnixMilli(1574403364000)),
			PositionPair: "tEOSUSD",
		}}
		assert.Equal(t, exp, result, "GetFundingCreditsHistory should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetFundingCreditsHistory should return credit %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetFundingCreditsHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingCreditsHistory should return the request failure")
}

func TestGetAccountFundingTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountFundingTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountFundingTrades must reject a nil request")
	_, err = e.GetAccountFundingTrades(t.Context(), &AccountFundingTradesRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errFundingSymbolRequired, "GetAccountFundingTrades must refuse a trading pair")
	_, err = e.GetAccountFundingTrades(t.Context(), &AccountFundingTradesRequest{Start: time.UnixMilli(1574100000000), End: time.UnixMilli(1574000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountFundingTrades must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &AccountFundingTradesRequest{Symbol: "fUST", Start: time.UnixMilli(1574000000000), End: time.UnixMilli(1574100000000), Limit: 25}
	if !mockTests {
		req = &AccountFundingTradesRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 25}
	}
	result, err := e.GetAccountFundingTrades(t.Context(), req)
	require.NoError(t, err, "GetAccountFundingTrades must not error")
	if mockTests {
		exp := []AccountFundingTrade{{
			ID:        636040,
			Symbol:    "fUST",
			CreatedAt: types.Time(time.UnixMilli(1574077528000)),
			OfferID:   41237922,
			Amount:    -100,
			Rate:      0.0024,
			Period:    2,
		}}
		assert.Equal(t, exp, result, "GetAccountFundingTrades should decode every field")
	} else {
		for i := range result {
			assert.Truef(t, isFundingSymbol(result[i].Symbol), "GetAccountFundingTrades should return trade %d's funding currency", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetAccountFundingTrades(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountFundingTrades should return the request failure")
}

func TestGetFundingInfo(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"", "f", "tBTCUSD", "USD"} {
		_, err := e.GetFundingInfo(t.Context(), symbol)
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingInfo should refuse symbol %q", symbol)
	}

	skipLiveWithoutCredentials(t)
	symbol := "fUST"
	if !mockTests {
		symbol = "fUSD"
	}
	result, err := e.GetFundingInfo(t.Context(), symbol)
	require.NoError(t, err, "GetFundingInfo must not error")
	if mockTests {
		exp := &FundingInfoResponse{
			Key:    "sym",
			Symbol: "fUST",
			Info: FundingInfo{
				YieldLoan:    0.0024,
				YieldLend:    0.0024,
				DurationLoan: 1.9522164351851852,
				DurationLend: 1.4818560606060607,
			},
		}
		assert.Equal(t, exp, result, "GetFundingInfo should decode every field")
	} else {
		assert.Equal(t, symbol, result.Symbol, "GetFundingInfo should return the currency's funding info")
	}

	_, err = newResponseServerExchange(t, `null`).GetFundingInfo(t.Context(), symbol)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingInfo should error for a null response")
	_, err = newUnavailableServerExchange(t).GetFundingInfo(t.Context(), symbol)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingInfo should return the request failure")
}

// TestMarginFundingRecordsEveryPosition decodes each Margin Funding record from an array with a distinct value at
// every tagged position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestMarginFundingRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `["UST",1001,2.5,3.5]`,
		FundingAutoRenewSettings{Currency: currency.UST, Period: 1001, Rate: 2.5, Threshold: 3.5})
}
