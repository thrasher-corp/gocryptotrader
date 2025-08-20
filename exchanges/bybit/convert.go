package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Consts for WalletAccountType
const (
	Funding  WalletAccountType = "eb_convert_funding"
	Uta      WalletAccountType = "eb_convert_uta"
	Spot     WalletAccountType = "eb_convert_spot"
	Contract WalletAccountType = "eb_convert_contract"
	Inverse  WalletAccountType = "eb_convert_inverse"
)

var supportedAccountTypes = []WalletAccountType{Funding, Uta, Spot, Contract, Inverse}

var (
	errUnsupportedAccountType = errors.New("unsupported account type")
	errCurrencyCodesEqual     = errors.New("from and to currency codes cannot be equal")
	errRequestCoinInvalid     = errors.New("request coin must match from coin if provided")
	errQuoteTxIDEmpty         = errors.New("quoteTxID cannot be empty")
)

// WalletAccountType represents the different types of wallet accounts
type WalletAccountType string

// Coin represents a coin that can be converted
type Coin struct {
	Coin               currency.Code `json:"coin"`
	FullName           string        `json:"fullName"`
	Icon               string        `json:"icon"`
	IconNight          string        `json:"iconNight"`
	AccuracyLength     int           `json:"accuracyLength"`
	CoinType           string        `json:"coinType"`
	Balance            types.Number  `json:"balance"`
	BalanceInUSDT      types.Number  `json:"uBalance"`
	SingleFromMinLimit types.Number  `json:"singleFromMinLimit"` // The minimum amount of fromCoin per transaction
	SingleFromMaxLimit types.Number  `json:"singleFromMaxLimit"` // The maximum amount of fromCoin per transaction
	DisableFrom        bool          `json:"disableFrom"`        // true: the coin is disabled to be fromCoin, false: the coin is allowed to be fromCoin
	DisableTo          bool          `json:"disableTo"`          // true: the coin is disabled to be toCoin, false: the coin is allowed to be toCoin

	// Reserved fields, ignored for now
	TimePeriod        int          `json:"timePeriod"`
	SingleToMinLimit  types.Number `json:"singleToMinLimit"`
	SingleToMaxLimit  types.Number `json:"singleToMaxLimit"`
	DailyFromMinLimit types.Number `json:"dailyFromMinLimit"`
	DailyFromMaxLimit types.Number `json:"dailyFromMaxLimit"`
	DailyToMinLimit   types.Number `json:"dailyToMinLimit"`
	DailyToMaxLimit   types.Number `json:"dailyToMaxLimit"`
}

// GetConvertCoinList returns a list of coins you can convert to/from
func (e *Exchange) GetConvertCoinList(ctx context.Context, accountType WalletAccountType, coin currency.Code, isCoinToBuy bool) ([]Coin, error) {
	if !slices.Contains(supportedAccountTypes, accountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
	}

	params := url.Values{}
	params.Set("accountType", string(accountType))

	if isCoinToBuy {
		if coin.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		params.Set("coin", coin.Upper().String())
		params.Set("side", "1")
	} else {
		params.Set("side", "0")
	}

	var resp struct {
		List []Coin `json:"coins"`
	}

	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-coin-list", params, nil, &resp, defaultEPL)
}

// RequestAQuoteParams holds the parameters for requesting a quote
type RequestAQuoteParams struct {
	// Required fields
	AccountType WalletAccountType
	From        currency.Code // Convert from coin (coin to sell)
	To          currency.Code // Convert to coin (coin to buy)
	Amount      float64       // Convert amount

	// Optional fields
	RequestCoin  currency.Code // This will default to FromCoin
	FromCoinType string        // "crypto"
	ToCoinType   string        // "crypto"
	ParamType    string        // "opFrom", mainly used for API broker user
	ParamValue   string        // Broker ID, mainly used for API broker user
	RequestID    string
}

// RequestAQuoteResponse represents a response for a request a quote
type RequestAQuoteResponse struct {
	QuoteTxID    string          `json:"quoteTxId"` // Quote transaction ID. It is system generated, and it is used to confirm quote and query the result of transaction
	ExchangeRate types.Number    `json:"exchangeRate"`
	FromCoin     currency.Code   `json:"fromCoin"`
	FromCoinType string          `json:"fromCoinType"`
	ToCoin       currency.Code   `json:"toCoin"`
	ToCoinType   string          `json:"toCoinType"`
	FromAmount   types.Number    `json:"fromAmount"`
	ToAmount     types.Number    `json:"toAmount"`
	ExpiredTime  types.Time      `json:"expiredTime"` // The expiry time for this quote (15 seconds)
	RequestID    string          `json:"requestId"`
	ExtTaxAndFee json.RawMessage `json:"extTaxAndFee"` // Compliance-related field. Currently returns an empty array, which may be used in the future
}

// RequestAQuote requests a conversion quote between two coins with the specified parameters.
func (e *Exchange) RequestAQuote(ctx context.Context, params *RequestAQuoteParams) (*RequestAQuoteResponse, error) {
	if !slices.Contains(supportedAccountTypes, params.AccountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, params.AccountType)
	}

	if params.From.IsEmpty() {
		return nil, fmt.Errorf("from %w", currency.ErrCurrencyCodeEmpty)
	}
	if params.To.IsEmpty() {
		return nil, fmt.Errorf("to %w", currency.ErrCurrencyCodeEmpty)
	}

	if params.From.Equal(params.To) {
		return nil, errCurrencyCodesEqual
	}

	if !params.RequestCoin.IsEmpty() {
		if !params.RequestCoin.Equal(params.From) {
			return nil, errRequestCoinInvalid
		}
	} else {
		params.RequestCoin = params.From
	}

	if params.Amount <= 0 {
		return nil, fmt.Errorf("amount %w", order.ErrAmountIsInvalid)
	}

	payload := &struct {
		AccountType  string `json:"accountType"`
		From         string `json:"fromCoin"`
		To           string `json:"toCoin"`
		Amount       string `json:"requestAmount"`
		RequestCoin  string `json:"requestCoin"`
		FromCoinType string `json:"fromCoinType,omitempty"`
		ToCoinType   string `json:"toCoinType,omitempty"`
		ParamType    string `json:"paramType,omitempty"`
		ParamValue   string `json:"paramValue,omitempty"`
		RequestID    string `json:"requestId,omitempty"`
	}{
		AccountType:  string(params.AccountType),
		From:         params.From.Upper().String(),
		To:           params.To.Upper().String(),
		Amount:       strconv.FormatFloat(params.Amount, 'f', -1, 64),
		RequestCoin:  params.RequestCoin.Upper().String(),
		FromCoinType: params.FromCoinType,
		ToCoinType:   params.ToCoinType,
		ParamType:    params.ParamType,
		ParamValue:   params.ParamValue,
		RequestID:    params.RequestID,
	}

	var resp *RequestAQuoteResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/quote-apply", nil, payload, &resp, defaultEPL)
}

// ConfirmAQuoteResponse represents a response for confirming a quote
type ConfirmAQuoteResponse struct {
	ExchangeStatus string `json:"exchangeStatus"`
	QuoteTxID      string `json:"quoteTxId"`
}

// ConfirmAQuote confirms a quote transaction and executes the conversion
func (e *Exchange) ConfirmAQuote(ctx context.Context, quoteTxID string) (*ConfirmAQuoteResponse, error) {
	if quoteTxID == "" {
		return nil, errQuoteTxIDEmpty
	}

	payload := struct {
		QuoteTxID string `json:"quoteTxId"`
	}{
		QuoteTxID: quoteTxID,
	}

	var resp *ConfirmAQuoteResponse
	err := e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/convert-execute", nil, payload, &resp, defaultEPL)
	return resp, err
}

// ConvertStatusResponse represents the response for a conversion status query
type ConvertStatusResponse struct {
	AccountType    WalletAccountType `json:"accountType"`
	ExchangeTxID   string            `json:"exchangeTxId"`
	UserID         string            `json:"userId"`
	FromCoin       currency.Code     `json:"fromCoin"`
	FromCoinType   string            `json:"fromCoinType"`
	FromAmount     types.Number      `json:"fromAmount"`
	ToCoin         currency.Code     `json:"toCoin"`
	ToCoinType     string            `json:"toCoinType"`
	ToAmount       types.Number      `json:"toAmount"`
	ExchangeStatus string            `json:"exchangeStatus"`
	ExtInfo        json.RawMessage   `json:"extInfo"` // Reserved field, ignored for now
	ConvertRate    types.Number      `json:"convertRate"`
	CreatedAt      types.Time        `json:"createdAt"`
}

// GetConvertStatus retrieves the status of a conversion transaction
func (e *Exchange) GetConvertStatus(ctx context.Context, accountType WalletAccountType, quoteTxID string) (*ConvertStatusResponse, error) {
	if !slices.Contains(supportedAccountTypes, accountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
	}

	if quoteTxID == "" {
		return nil, errQuoteTxIDEmpty
	}

	params := url.Values{}
	params.Set("quoteTxId", quoteTxID)
	params.Set("accountType", string(accountType))

	var resp struct {
		ConvertStatus ConvertStatusResponse `json:"result"`
	}
	return &resp.ConvertStatus, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/convert-result-query", params, nil, &resp, defaultEPL)
}

// ConvertHistoryResponse represents a response for conversion history
type ConvertHistoryResponse struct {
	AccountType    WalletAccountType `json:"accountType"`
	ExchangeTxID   string            `json:"exchangeTxId"`
	UserID         string            `json:"userId"`
	FromCoin       currency.Code     `json:"fromCoin"`
	FromCoinType   string            `json:"fromCoinType"`
	FromAmount     types.Number      `json:"fromAmount"`
	ToCoin         currency.Code     `json:"toCoin"`
	ToCoinType     string            `json:"toCoinType"`
	ToAmount       types.Number      `json:"toAmount"`
	ExchangeStatus string            `json:"exchangeStatus"`
	ExtInfo        json.RawMessage   `json:"extInfo"`
	ConvertRate    types.Number      `json:"convertRate"`
	CreatedAt      types.Time        `json:"createdAt"`
}

// GetConvertHistory retrieves the conversion history for the specified account types.
// All params are optional
func (e *Exchange) GetConvertHistory(ctx context.Context, accountTypes []WalletAccountType, index, limit uint64) ([]ConvertHistoryResponse, error) {
	params := url.Values{}
	for _, accountType := range accountTypes {
		if !slices.Contains(supportedAccountTypes, accountType) {
			return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
		}
		params.Add("accountType", string(accountType))
	}

	if index != 0 {
		params.Set("index", strconv.FormatUint(index, 10))
	}

	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}

	var resp struct {
		List []ConvertHistoryResponse `json:"list"`
	}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-convert-history", params, nil, &resp, defaultEPL)
}
