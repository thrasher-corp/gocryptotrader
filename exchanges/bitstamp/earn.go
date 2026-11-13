package bitstamp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// EarnSubscribe subscribes an amount to a staking or lending product
func (e *Exchange) EarnSubscribe(ctx context.Context, req *EarnSubscriptionRequest) error {
	body, err := earnSubscription(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/subscribe/", nil, body, nil)
}

// EarnUnsubscribe unsubscribes an amount from a staking or lending product
func (e *Exchange) EarnUnsubscribe(ctx context.Context, req *EarnSubscriptionRequest) error {
	body, err := earnSubscription(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/unsubscribe/", nil, body, nil)
}

func earnSubscription(req *EarnSubscriptionRequest) (*earnSubscriptionBody, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.EarnType == "" {
		return nil, errEarnTypeRequired
	}
	if req.EarnTerm == "" {
		return nil, errEarnTermRequired
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	return &earnSubscriptionBody{
		Currency: req.Currency.Upper().String(),
		EarnType: req.EarnType,
		EarnTerm: req.EarnTerm,
		Amount:   req.Amount,
	}, nil
}

// SetEarnSubscriptionSetting opts in to or out of an Earn product; only staking is currently supported
func (e *Exchange) SetEarnSubscriptionSetting(ctx context.Context, req *EarnSubscriptionSettingRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Setting == "" {
		return errEarnSettingRequired
	}
	if req.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if req.EarnType == "" {
		return errEarnTypeRequired
	}
	body := &earnSubscriptionSettingBody{
		Setting:  req.Setting,
		Currency: req.Currency.Upper().String(),
		EarnType: req.EarnType,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/subscriptions/setting/", nil, body, nil)
}

// GetEarnTransactions returns the account's Earn transaction history
func (e *Exchange) GetEarnTransactions(ctx context.Context, req *EarnTransactionsRequest) ([]EarnTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if !req.QuoteCurrency.IsEmpty() {
		params.Set("quote_currency", req.QuoteCurrency.Upper().String())
	}
	var resp []EarnTransactionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/earn/transactions/", params, nil, &resp)
}

// GetEarnSubscriptions returns the account's Earn subscriptions
func (e *Exchange) GetEarnSubscriptions(ctx context.Context) ([]EarnSubscriptionResponse, error) {
	var resp []EarnSubscriptionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/earn/subscriptions/", nil, nil, &resp)
}
