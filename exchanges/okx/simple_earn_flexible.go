package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Savings purchase rate bounds, 1% to 365% a year
const (
	minSavingsPurchaseRate = 0.01
	maxSavingsPurchaseRate = 3.65
)

var errSavingsRateOutOfRange = errors.New("savings purchase rate out of range")

// GetSavingBalance returns the account's savings balances, every currency when ccy is empty
func (e *Exchange) GetSavingBalance(ctx context.Context, ccy currency.Code) ([]SavingBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SavingBalanceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSavingBalanceEPL, http.MethodGet, common.EncodeURLValues("finance/savings/balance", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// SavingsPurchaseOrRedemption purchases savings shares with funding account assets or redeems them
func (e *Exchange) SavingsPurchaseOrRedemption(ctx context.Context, arg *SavingsPurchaseRedemptionRequest) (*SavingsPurchaseRedemptionResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w: side must be purchase or redempt", order.ErrSideIsInvalid)
	}
	if arg.Rate != 0 && (arg.Rate < minSavingsPurchaseRate || arg.Rate > maxSavingsPurchaseRate) {
		return nil, fmt.Errorf("%w: %v must be between %v and %v", errSavingsRateOutOfRange, arg.Rate, minSavingsPurchaseRate, maxSavingsPurchaseRate)
	}
	var resp *SavingsPurchaseRedemptionResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, savingsPurchaseRedemptionEPL, http.MethodPost, "finance/savings/purchase-redempt", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetLendingHistory returns the account's savings lending over the past month
func (e *Exchange) GetLendingHistory(ctx context.Context, arg *LendingHistoryRequest) ([]LendingHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := savingsHistoryValues(arg.Currency, arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []LendingHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getLendingHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPublicBorrowInfo returns the market's borrowing rates, every currency when ccy is empty
func (e *Exchange) GetPublicBorrowInfo(ctx context.Context, ccy currency.Code) ([]PublicBorrowInfoResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []PublicBorrowInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowInfoEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-summary", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPublicBorrowHistory returns the market's borrowing and lending rate history
func (e *Exchange) GetPublicBorrowHistory(ctx context.Context, arg *PublicBorrowHistoryRequest) ([]PublicBorrowHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := savingsHistoryValues(arg.Currency, arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []PublicBorrowHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-history", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// savingsHistoryValues validates a savings history time window and encodes it with the currency and page size as query
// parameters; before bounds the window's start and after its end
func savingsHistoryValues(ccy currency.Code, after, before time.Time, limit uint64) (url.Values, error) {
	if !after.IsZero() && !before.IsZero() && before.After(after) {
		return nil, fmt.Errorf("%w: before %s must not be later than after %s", common.ErrStartAfterEnd, before, after)
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}
