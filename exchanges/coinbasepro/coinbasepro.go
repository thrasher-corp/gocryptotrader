package coinbasepro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	coinbaseproAPIURL                  = "https://api.coinbase.com/"
	coinbaseproSandboxAPIURL           = "https://api-public.sandbox.exchange.coinbase.com/"
	coinbaseproAPIVersion              = "0"
	coinbaseproAccounts                = "accounts"
	coinbaseproHolds                   = "holds"
	coinbaseproLedger                  = "ledger"
	coinbaseproTransfers               = "transfers"
	coinbaseproAddressBook             = "address-book"
	coinbaseproCoinbaseAccounts        = "coinbase-accounts"
	coinbaseproAddress                 = "addresses"
	coinbaseproConversions             = "conversions"
	coinbaseproCurrencies              = "currencies"
	coinbaseproDepositCoinbase         = "deposits/coinbase-account"
	coinbaseproPaymentMethodDeposit    = "deposits/payment-method"
	coinbaseproPaymentMethod           = "payment-methods"
	coinbaseproTravelRules             = "travel-rules"
	coinbaseproWithdrawalCoinbase      = "withdrawals/coinbase-account"
	coinbaseproWithdrawalCrypto        = "withdrawals/crypto"
	coinbaseproFeeEstimate             = "withdrawals/fee-estimate"
	coinbaseproWithdrawalPaymentMethod = "withdrawals/payment-method"
	coinbaseproFees                    = "fees"
	coinbaseproFills                   = "fills"
	coinbaseproOrders                  = "orders"
	coinbaseproOracle                  = "oracle"
	coinbaseproProducts                = "products"
	coinbaseproOrderbook               = "book"
	coinbaseproHistory                 = "candles"
	coinbaseproStats                   = "stats"
	coinbaseproTicker                  = "ticker"
	coinbaseproTrades                  = "trades"
	coinbaseproProfiles                = "profiles"
	coinbaseproTransfer                = "transfer"
	coinbaseproDeactivate              = "deactivate"
	coinbaseproReports                 = "reports"
	coinbaseproUsers                   = "users"
	coinbaseproExchangeLimits          = "exchange-limits"
	coinbaseproSettlementPreferences   = "settlement-preferences"
	coinbaseproWrappedAssets           = "wrapped-assets"
	coinbaseproStakeWraps              = "stake-wrap"
	coinbaseproConversionRate          = "conversion-rate"

	// coinbaseproTime           = "time"
	// coinbaseproMarginTransfer = "profiles/margin-transfer"
	// coinbaseproPosition       = "position"
	// coinbaseproPositionClose  = "position/close"
	// coinbaseproTrailingVolume = "users/self/trailing-volume"
)

const (
	pageNone   = ""
	pageBefore = "before"
	pageAfter  = "after"
)

// CoinbasePro is the overarching type across the coinbasepro package
type CoinbasePro struct {
	exchange.Base
}

// GetAllAccounts returns information on all trading accounts associated with the API key
func (c *CoinbasePro) GetAllAccounts(ctx context.Context) ([]AccountResponse, error) {
	var resp []AccountResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAccounts, nil, &resp, nil)
}

// GetAccountByID returns information for a single account
func (c *CoinbasePro) GetAccountByID(ctx context.Context, accountID string) (*AccountResponse, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproAccounts, accountID)
	resp := AccountResponse{}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// GetHolds returns information on the holds of an account
func (c *CoinbasePro) GetHolds(ctx context.Context, accountID, direction, step string, limit int64) ([]AccountHolds, ReturnedPaginationHeaders, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproHolds)

	var params Params
	params.urlVals = url.Values{}
	// Warning: This endpoint doesn't seem to properly support pagination, the headers
	// indicating the cursor position are never actually present. Still, it's handled
	// as if it works, in case it gets fixed.
	params.PrepareDSL(direction, step, limit)

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []AccountHolds
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAccountLedger returns a list of ledger activity
func (c *CoinbasePro) GetAccountLedger(ctx context.Context, accountID, direction, step, pID string, startDate, endDate time.Time, limit int64) ([]AccountLedgerResponse, ReturnedPaginationHeaders, error) {
	var params Params
	params.urlVals = url.Values{}
	var rph ReturnedPaginationHeaders

	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return nil, rph, err
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproLedger)

	params.PrepareDSL(direction, step, limit)

	if pID != "" {
		params.urlVals.Set("profile_id", pID)
	}

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []AccountLedgerResponse
	retH := http.Header{}

	err = c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph = ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAccountTransfers returns a history of withdrawal and or deposit
// transactions for a single account
func (c *CoinbasePro) GetAccountTransfers(ctx context.Context, accountID, direction, step, transferType string, limit int64) ([]TransferResponse, ReturnedPaginationHeaders, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproTransfers)

	var params Params
	params.urlVals = url.Values{}

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)

	path = common.EncodeURLValues(path, params.urlVals)

	var resp []TransferResponse
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAddressBook returns all addresses stored in the address book
func (c *CoinbasePro) GetAddressBook(ctx context.Context) ([]GetAddressResponse, error) {
	var resp []GetAddressResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAddressBook, nil, &resp, nil)
}

// AddAddresses adds new addresses to the address book
func (c *CoinbasePro) AddAddresses(ctx context.Context, req []AddAddressRequest) ([]AddAddressResponse, error) {
	params := make(map[string]interface{})
	params["addresses"] = req
	// The documentation also prompts us to add in an arbitrary amount of strings
	// into the parameters, without specifying what they're for. Adding some seemed
	// to do nothing

	var resp []AddAddressResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproAddressBook, params, &resp, nil)
}

// DeleteAddress deletes an address from the address book
func (c *CoinbasePro) DeleteAddress(ctx context.Context, addressID string) error {
	path := fmt.Sprintf("%s/%s", coinbaseproAddressBook, addressID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, nil)
}

// GetCoinbaseWallets returns all of the user's available Coinbase wallets
func (c *CoinbasePro) GetCoinbaseWallets(ctx context.Context) ([]CoinbaseAccounts, error) {
	var resp []CoinbaseAccounts

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproCoinbaseAccounts, nil, &resp, nil)
}

// GenerateCryptoAddress generates a one-time address for deposting crypto
func (c *CoinbasePro) GenerateCryptoAddress(ctx context.Context, accountID, profileID, network string) (*CryptoAddressResponse, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseproCoinbaseAccounts, accountID, coinbaseproAddress)

	// In this case, accountID has to come from GetCoinbaseWallets, not GetAccounts

	params := map[string]interface{}{"account_id": accountID, "profile_id": profileID,
		"network": network}

	resp := CryptoAddressResponse{}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, &resp, nil)
}

// ConvertCurrency converts between two currencies in the specified profile
func (c *CoinbasePro) ConvertCurrency(ctx context.Context, profileID, from, to, nonce string, amount float64) (ConvertResponse, error) {
	params := map[string]interface{}{"profile_id": profileID, "from": from, "to": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64), "nonce": nonce}

	resp := ConvertResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproConversions, params, &resp, nil)
}

// GetConversionByID returns the details of a past conversion, given its ID
func (c *CoinbasePro) GetConversionByID(ctx context.Context, conversionID, profileID string) (ConvertResponse, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproConversions, conversionID)
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("profile_id", profileID)
	path = common.EncodeURLValues(path, params.urlVals)

	resp := ConvertResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// GetAllCurrencies returns a list of currencies known by the exchange
// Warning: Currencies won't necessarily be available for trading
func (c *CoinbasePro) GetAllCurrencies(ctx context.Context) ([]Currency, error) {
	var currencies []Currency

	return currencies,
		c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproCurrencies, &currencies)
}

// GetCurrencyByID returns info on a single currency given its ID in ISO 4217, or
// in a custom code for currencies which lack an ISO 4217 code
func (c *CoinbasePro) GetCurrencyByID(ctx context.Context, currencyID string) (*Currency, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproCurrencies, currencyID)

	resp := Currency{}

	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// DepositViaCoinbase deposits funds from a Coinbase account
func (c *CoinbasePro) DepositViaCoinbase(ctx context.Context, profileID, currency, coinbaseAccountID string, amount float64) (DepositWithdrawalInfo, error) {
	params := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": coinbaseAccountID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproDepositCoinbase, params, &resp, nil)
}

// DepositViaPaymentMethod deposits funds from a payment method. SEPA is not allowed
func (c *CoinbasePro) DepositViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	params := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPaymentMethodDeposit, params, &resp, nil)
}

// GetPayMethods returns a full list of payment methods
func (c *CoinbasePro) GetPayMethods(ctx context.Context) ([]PaymentMethod, error) {
	var resp []PaymentMethod

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPaymentMethod, nil, &resp, nil)
}

// GetAllTransfers returns all in-progress and completed transfers in and out of any
// of the user's accounts
func (c *CoinbasePro) GetAllTransfers(ctx context.Context, profileID, direction, step, transferType string, limit int64) ([]TransferResponse, ReturnedPaginationHeaders, error) {
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("profile_id", profileID)
	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)
	path := common.EncodeURLValues(coinbaseproTransfers, params.urlVals)

	resp := []TransferResponse{}
	retH := http.Header{}

	err := c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph := ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetTransferByID returns information on a single transfer when provided with its ID
func (c *CoinbasePro) GetTransferByID(ctx context.Context, transferID string) (*TransferResponse, error) {
	path := fmt.Sprintf("%s/%s", coinbaseproTransfers, transferID)
	resp := TransferResponse{}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// SendTravelInfoForTransfer sends travel rule information for a transfer
func (c *CoinbasePro) SendTravelInfoForTransfer(ctx context.Context, transferID, originName, originCountry string) (string, error) {
	path := fmt.Sprintf("%s/%s/%s", coinbaseproTransfers, transferID,
		coinbaseproTravelRules)
	params := map[string]interface{}{"transfer_id": transferID,
		"originator_name": originName, "originator_country": originCountry}

	var resp string

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, &resp, nil)
}

// WithdrawViaCoinbase withdraws funds to a coinbase account.
func (c *CoinbasePro) WithdrawViaCoinbase(ctx context.Context, profileID, accountID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": accountID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCoinbase, req, &resp, nil)
}

// WithdrawCrypto withdraws funds to a crypto address
func (c *CoinbasePro) WithdrawCrypto(ctx context.Context, profileID, currency, cryptoAddress, destinationTag, twoFactorCode, network string, amount float64, noDestinationTag, addNetworkFee bool, nonce int32) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":   strconv.FormatFloat(amount, 'f', -1, 64),
		"currency": currency, "crypto_address": cryptoAddress,
		"destination_tag": destinationTag, "no_destination_tag": noDestinationTag,
		"two_factor_code": twoFactorCode, "nonce": nonce, "network": network,
		"add_network_fee": addNetworkFee}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCrypto, req, &resp, nil)
}

// GetWithdrawalFeeEstimate has Coinbase estimate the fee for withdrawing in a certain
// network to a certain address
func (c *CoinbasePro) GetWithdrawalFeeEstimate(ctx context.Context, currency, cryptoAddress, network string) (WithdrawalFeeEstimate, error) {
	resp := WithdrawalFeeEstimate{}
	if currency == "" {
		return resp, errors.New("currency cannot be empty")
	}
	if cryptoAddress == "" {
		return resp, errors.New("cryptoAddress cannot be empty")
	}
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("currency", currency)
	params.urlVals.Set("crypto_address", cryptoAddress)
	params.urlVals.Set("network", network)
	path := common.EncodeURLValues(coinbaseproFeeEstimate, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
func (c *CoinbasePro) WithdrawViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	req := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	resp := DepositWithdrawalInfo{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalPaymentMethod, req, &resp, nil)
}

// GetFees returns your current maker & taker fee rates, as well as your 30-day
// trailing volume. Quoted rates are subject to change.
func (c *CoinbasePro) GetFees(ctx context.Context) (FeeResponse, error) {
	resp := FeeResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproFees, nil, &resp, nil)
}

// GetFills returns information of recent fills on the specified profile
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, currencyPair, direction, step, marketType string, limit int64, startDate, endDate time.Time) ([]FillResponse, ReturnedPaginationHeaders, error) {
	var rph ReturnedPaginationHeaders

	if orderID == "" && currencyPair == "" {
		return nil, rph, errors.New("requires either order id or product id")
	}
	var params Params
	params.urlVals = url.Values{}
	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return nil, rph, err
	}

	if orderID != "" {
		params.urlVals.Set("order_id", orderID)
	}
	if currencyPair != "" {
		params.urlVals.Set("product_id", currencyPair)
	}

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("market_type", marketType)

	path := common.EncodeURLValues(coinbaseproFills, params.urlVals)

	var resp []FillResponse
	retH := http.Header{}

	err = c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph = ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// GetAllOrders lists orders, filtered by their status
func (c *CoinbasePro) GetAllOrders(ctx context.Context, profileID, currencyPair, sortedBy, sorting, direction, step, marketType string, startDate, endDate time.Time, limit int64, status []string) ([]GeneralizedOrderResponse, ReturnedPaginationHeaders, error) {
	var rph ReturnedPaginationHeaders
	if limit < 1 {
		return nil, rph, errors.New("limit must be greater than 0")
	}
	var params Params
	params.urlVals = make(url.Values)
	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return nil, rph, err
	}

	params.PrepareProfIDAndProdID(profileID, currencyPair)
	params.urlVals.Set("sorted_by", sortedBy)
	params.urlVals.Set("sorting", sorting)
	params.PrepareDSL(direction, step, limit)

	for _, individualStatus := range status {
		params.urlVals.Add("status", individualStatus)
	}

	params.urlVals.Set("market_type", marketType)

	path := common.EncodeURLValues(coinbaseproOrders, params.urlVals)

	var resp []GeneralizedOrderResponse
	retH := http.Header{}

	err = c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, &retH)

	rph = ReturnedPaginationHeaders{before: retH.Get("CB-BEFORE"), after: retH.Get("CB-AFTER")}

	return resp, rph, err
}

// CancelAllExistingOrders attempts to cancel all open orders. The exchange warns that
// this may need to be called multiple times to properly close all of them
func (c *CoinbasePro) CancelAllExistingOrders(ctx context.Context, profileID, currencyPair string) ([]string, error) {
	var params Params
	params.urlVals = url.Values{}

	params.PrepareProfIDAndProdID(profileID, currencyPair)

	path := common.EncodeURLValues(coinbaseproOrders, params.urlVals)

	var resp []string

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, &resp, nil)
}

// PlaceOrder places either a limit, market, or stop order
func (c *CoinbasePro) PlaceOrder(ctx context.Context, profileID, orderType, side, currencyPair, stp, stop, timeInForce, cancelAfter, clientOID string, stopPrice, price, size, funds float64, postOnly bool) (*GeneralizedOrderResponse, error) {
	var resp GeneralizedOrderResponse

	if (orderType == order.Limit.Lower() || orderType == "" || orderType == "stop") &&
		(price == 0 || size == 0) {
		return &resp, errors.New("price and size must be greater than 0 for limit or stop orders")
	}
	if orderType == order.Market.Lower() && (size == 0 && funds == 0) {
		return &resp, errors.New("size or funds must be greater than 0 for market orders")
	}
	if side != order.Buy.Lower() && side != order.Sell.Lower() {
		return &resp, errors.New("side must be buy or sell")
	}

	req := map[string]interface{}{"profile_id": profileID, "type": orderType, "side": side,
		"product_id": currencyPair, "stp": stp, "stop": stop,
		"size":          strconv.FormatFloat(size, 'f', -1, 64),
		"time_in_force": timeInForce, "cancel_after": cancelAfter, "post_only": postOnly,
		"client_oid": clientOID}

	if stopPrice != 0 {
		req["stop_price"] = strconv.FormatFloat(stopPrice, 'f', -1, 64)
	}
	if funds != 0 {
		req["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
	}
	if price != 0 {
		req["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	}

	return &resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproOrders, req, &resp, nil)
}

// GetOrderByID returns a single order by order id.
func (c *CoinbasePro) GetOrderByID(ctx context.Context, orderID, marketType string, clientID bool) (*GeneralizedOrderResponse, error) {
	resp := GeneralizedOrderResponse{}
	if orderID == "" {
		return &resp, errors.New("order id cannot be empty")
	}
	if clientID {
		orderID = fmt.Sprintf("client:%s", orderID)
	}
	var param Params
	param.urlVals = url.Values{}

	// Spot's seemingly the only supported market type; passing in anything else here will
	// cause the request to time out after a minute
	if marketType == "spot" {
		param.urlVals.Set("market_type", marketType)
	}

	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)
	path = common.EncodeURLValues(path, param.urlVals)

	return &resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// CancelExistingOrder cancels order by orderID
func (c *CoinbasePro) CancelExistingOrder(ctx context.Context, orderID, profileID, productID string, clientID bool) (string, error) {
	var resp string
	if orderID == "" {
		return resp, errors.New("order id cannot be empty")
	}
	if clientID {
		orderID = fmt.Sprintf("client:%s", orderID)
	}
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)
	var param Params
	param.urlVals = url.Values{}

	param.PrepareProfIDAndProdID(profileID, productID)

	path = common.EncodeURLValues(path, param.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, nil)
}

// GetSignedPrices returns some cryptographically signed prices ready to be
// posted on-chain using Compound's Open Oracle smart contract
func (c *CoinbasePro) GetSignedPrices(ctx context.Context) (SignedPrices, error) {
	resp := SignedPrices{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproOracle, nil, &resp, nil)
}

// GetAllProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetAllProducts(ctx context.Context, productType string) ([]Product, error) {
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("type", productType)

	path := common.EncodeURLValues(coinbaseproProducts, params.urlVals)

	var products []Product

	return products, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &products)
}

// GetProductByID returns information on a single specified currency pair
func (c *CoinbasePro) GetProductByID(ctx context.Context, productID string) (*Product, error) {
	if productID == "" {
		return nil, errors.New("product id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproProducts, productID)

	resp := Product{}

	return &resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetOrderbook returns orderbook by currency pair and level
func (c *CoinbasePro) GetOrderbook(ctx context.Context, symbol string, level int32) (*OrderbookFinalResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproProducts, symbol, coinbaseproOrderbook)
	if level > 0 {
		var params Params
		params.urlVals = url.Values{}
		params.urlVals.Set("level", strconv.Itoa(int(level)))

		path = common.EncodeURLValues(path, params.urlVals)
	}

	data := OrderbookIntermediaryResponse{}
	err := c.SendHTTPRequest(ctx, exchange.RestSpot, path, &data)
	if err != nil {
		return nil, err
	}

	obF := OrderbookFinalResponse{
		Sequence:    data.Sequence,
		AuctionMode: data.AuctionMode,
		Auction:     data.Auction,
		Time:        data.Time,
	}

	obF.Bids, err = OrderbookHelper(data.Bids, level)
	if err != nil {
		return nil, err
	}
	obF.Asks, err = OrderbookHelper(data.Asks, level)
	if err != nil {
		return nil, err
	}

	return &obF, nil
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity. Contrary to the documentation,
// requests that return more than 300 data points aren't rejected; just truncated
// to the 300 most recent data points.
func (c *CoinbasePro) GetHistoricRates(ctx context.Context, currencyPair string, granularity int64, startDate, endDate time.Time) ([]History, error) {
	if currencyPair == "" {
		return nil, errors.New("currency pair cannot be empty")
	}

	var params Params
	params.urlVals = url.Values{}

	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return nil, err
	}

	allowedGranularities := [7]int64{0, 60, 300, 900, 3600, 21600, 86400}
	validGran, _ := common.InArray(granularity, allowedGranularities)
	if !validGran {
		return nil, fmt.Errorf("invalid granularity %v, allowed granularities are: %+v",
			granularity, allowedGranularities)
	}
	if granularity > 0 {
		params.urlVals.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	var resp [][6]float64

	path := common.EncodeURLValues(
		fmt.Sprintf("%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproHistory),
		params.urlVals)
	if err := c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp); err != nil {
		return nil, err
	}

	history := make([]History, len(resp))
	for x := range resp {
		history[x] = History{
			Time:   time.Unix(int64(resp[x][0]), 0),
			Low:    resp[x][1],
			High:   resp[x][2],
			Open:   resp[x][3],
			Close:  resp[x][4],
			Volume: resp[x][5],
		}
	}

	return history, nil
}

// GetStats returns 30 day and 24 hour stats for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (c *CoinbasePro) GetStats(ctx context.Context, currencyPair string) (Stats, error) {
	stats := Stats{}
	if currencyPair == "" {
		return stats, errors.New("currency pair cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproStats)

	return stats, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetTicker returns snapshot information about the last trade (tick), best bid/ask and
// 24h volume.
func (c *CoinbasePro) GetTicker(ctx context.Context, currencyPair string) (*Ticker, error) {
	if currencyPair == "" {
		return nil, errors.New("currency pair cannot be empty")
	}
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTicker)
	tick := Ticker{}

	return &tick, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &tick)
}

// GetTrades lists information on the latest trades for a product
func (c *CoinbasePro) GetTrades(ctx context.Context, currencyPair, direction, step string, limit int64) ([]Trade, error) {
	if currencyPair == "" {
		return nil, errors.New("currency pair cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTrades)

	var params Params
	params.urlVals = url.Values{}
	params.PrepareDSL(direction, step, limit)

	path = common.EncodeURLValues(path, params.urlVals)

	var trades []Trade

	return trades, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// GetAllProfiles returns information on all of the current user's profiles
func (c *CoinbasePro) GetAllProfiles(ctx context.Context, active *bool) ([]Profile, error) {
	var params Params
	params.urlVals = url.Values{}

	if active != nil {
		params.urlVals.Set("active", strconv.FormatBool(*active))
	}

	var resp []Profile

	path := common.EncodeURLValues(coinbaseproProfiles, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// CreateAProfile creates a new profile, failing if no name is provided,
// or if the user already has the max number of profiles
func (c *CoinbasePro) CreateAProfile(ctx context.Context, name string) (Profile, error) {
	var resp Profile
	if name == "" {
		return resp, errors.New("name cannot be empty")
	}

	req := map[string]interface{}{"name": name}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproProfiles, req, &resp, nil)
}

// TransferBetweenProfiles transfers an amount of currency from one profile to another
func (c *CoinbasePro) TransferBetweenProfiles(ctx context.Context, from, to, currency string, amount float64) (string, error) {
	var resp string
	if from == "" || to == "" || currency == "" {
		return resp, errors.New("from, to, and currency must all not be empty")
	}

	req := map[string]interface{}{"from": from, "to": to, "currency": currency,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64)}

	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, coinbaseproTransfer)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, req, &resp, nil)
}

// GetProfileByID returns information on a single profile, provided its ID
func (c *CoinbasePro) GetProfileByID(ctx context.Context, profileID string, active *bool) (Profile, error) {
	var params Params
	params.urlVals = url.Values{}
	if active != nil {
		params.urlVals.Set("active", strconv.FormatBool(*active))
	}

	var resp Profile
	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, profileID)
	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// RenameProfile renames a profile, provided its ID
func (c *CoinbasePro) RenameProfile(ctx context.Context, profileID, newName string) (Profile, error) {
	var resp Profile
	if newName == "" {
		return resp, errors.New("new name cannot be empty")
	}

	req := map[string]interface{}{"profile_id": profileID, "name": newName}

	path := fmt.Sprintf("%s/%s", coinbaseproProfiles, profileID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, req, &resp, nil)
}

// DeleteProfile deletes a profile and transfers its funds to a specified
// profile. Fails if there are any open orders on the profile facing deletion
func (c *CoinbasePro) DeleteProfile(ctx context.Context, profileID, transferTo string) (string, error) {
	var resp string
	if profileID == "" || transferTo == "" {
		return resp, errors.New("neither profileID nor transferTo can be empty")
	}

	req := map[string]interface{}{"profile_id": profileID, "to": transferTo}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproProfiles, profileID, coinbaseproDeactivate)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, req, &resp, nil)
}

// GetAllReports returns a list of all user-generated reports
func (c *CoinbasePro) GetAllReports(ctx context.Context, profileID string, reportType string, after time.Time, limit int64, ignoreExpired bool) ([]Report, error) {
	var resp []Report

	var params Params
	params.urlVals = url.Values{}

	params.urlVals.Set("profile_id", profileID)
	params.urlVals.Set("after", after.Format(time.RFC3339))
	if limit != 0 {
		params.urlVals.Set("limit", strconv.FormatInt(limit, 10))
	}

	params.urlVals.Set("type", reportType)
	params.urlVals.Set("ignore_expired", strconv.FormatBool(ignoreExpired))

	path := common.EncodeURLValues(coinbaseproReports, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// CreateReport creates a new report
func (c *CoinbasePro) CreateReport(ctx context.Context, reportType, year, format, email, profileID, productID, accountID string, balanceDate, startDate, endDate time.Time) (CreateReportResponse, error) {
	var resp CreateReportResponse

	if reportType == "" {
		return resp, errors.New("report type cannot be empty")
	}
	if reportType == "1099k-transaction-history" && year == "" {
		return resp, errors.New("year cannot be empty for 1099k-transaction-history reports")
	}
	if reportType != "balance" {
		err := common.StartEndTimeCheck(startDate, endDate)
		if err != nil {
			return resp, err
		}
	}

	req := map[string]interface{}{"type": reportType, "year": year, "format": format,
		"email": email, "profile_id": profileID}

	if reportType == "account" {
		req["account"] = ReportAccountStruct{StartDate: startDate.Format(time.RFC3339),
			EndDate: endDate.Format(time.RFC3339), AccountID: accountID}
	}
	if reportType == "balance" {
		req["balance"] = ReportBalanceStruct{DateTime: balanceDate.Format(time.RFC3339)}
	}
	if reportType == "fills" || reportType == "otc-fills" || reportType == "rfq-fills" ||
		reportType == "tax-invoice" {
		req[reportType] = ReportFillsTaxStruct{StartDate: startDate.Format(time.RFC3339),
			EndDate: endDate.Format(time.RFC3339), ProductID: productID}
	}
	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproReports, req, &resp, nil)
}

// GetReportByID returns a single report, provided its ID
func (c *CoinbasePro) GetReportByID(ctx context.Context, reportID string) (Report, error) {
	var resp Report
	if reportID == "" {
		return resp, errors.New("report id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// GetTravelRules returns a list of all travel rule information
func (c *CoinbasePro) GetTravelRules(ctx context.Context, direction, step, address string, limit int64) ([]TravelRule, error) {
	var resp []TravelRule
	var params Params
	params.urlVals = url.Values{}

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("address", address)

	path := common.EncodeURLValues(coinbaseproTravelRules, params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// CreateTravelRule creates a travel rule entry
func (c *CoinbasePro) CreateTravelRule(ctx context.Context, address, originName, originCountry string) (TravelRule, error) {
	var resp TravelRule

	req := map[string]interface{}{"address": address, "originator_name": originName,
		"originator_country": originCountry}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproTravelRules, req, &resp, nil)
}

// DeleteTravelRule deletes a travel rule entry
func (c *CoinbasePro) DeleteTravelRule(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproTravelRules, id)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil, nil)
}

// GetExchangeLimits returns information on payment method transfer limits,
// as well as buy/sell limits per currency
func (c *CoinbasePro) GetExchangeLimits(ctx context.Context, userID string) (ExchangeLimits, error) {
	var resp ExchangeLimits

	path := fmt.Sprintf("%s/%s/%s", coinbaseproUsers, userID, coinbaseproExchangeLimits)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// UpdateSettlementPreference updates whether one wants their funds to
// automatically convert to USD, USDC, or to remain in the currency received
func (c *CoinbasePro) UpdateSettlementPreference(ctx context.Context, userID, preference string) (string, error) {
	if userID == "" || preference == "" {
		return "", errors.New("neither userID nor preference can be empty")
	}

	req := map[string]interface{}{"settlement_preference": preference}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproUsers, userID, coinbaseproSettlementPreferences)

	var resp string

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, path, req, &resp, nil)
}

// GetAllWrappedAssets returns information on all supported wrapped assets
func (c *CoinbasePro) GetAllWrappedAssets(ctx context.Context) (AllWrappedAssetResponse, error) {
	var resp AllWrappedAssetResponse

	return resp,
		c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproWrappedAssets, &resp)
}

// GetAllStakeWraps returns details of all stake-wraps under the profile associated
// with the API key
func (c *CoinbasePro) GetAllStakeWraps(ctx context.Context, direction, from, to, status string, timestamp time.Time, limit int64) ([]StakeWrap, error) {
	var resp []StakeWrap

	var params Params
	params.urlVals = url.Values{}

	if !timestamp.IsZero() && !timestamp.Equal(time.Unix(0, 0)) {
		params.PrepareDSL(direction, timestamp.Format(time.RFC3339), limit)
	} else {
		params.urlVals.Set("limit", strconv.FormatInt(limit, 10))
	}

	params.urlVals.Set("from", from)
	params.urlVals.Set("to", to)
	params.urlVals.Set("status", status)

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps)

	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// CreateStakeWrap stakes and wraps from one currency to another, under the profile
// associated with the API key
func (c *CoinbasePro) CreateStakeWrap(ctx context.Context, from, to string, amount float64) (StakeWrap, error) {
	if from == "" || to == "" || amount == 0 {
		return StakeWrap{}, errors.New("none of from, to, or amount can be empty or zero")
	}
	var resp StakeWrap

	req := map[string]interface{}{"from": from, "to": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64)}

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, req, &resp, nil)
}

// GetStakeWrapByID returns details of a single stake-wrap
func (c *CoinbasePro) GetStakeWrapByID(ctx context.Context, stakeWrapID string) (StakeWrap, error) {
	var resp StakeWrap

	if stakeWrapID == "" {
		return resp, errors.New("stake wrap id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproWrappedAssets, coinbaseproStakeWraps, stakeWrapID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, nil)
}

// GetWrappedAssetByID returns details of a single wrapped asset
func (c *CoinbasePro) GetWrappedAssetByID(ctx context.Context, wrappedAssetID string) (WrappedAssetResponse, error) {
	var resp WrappedAssetResponse

	if wrappedAssetID == "" {
		return resp, errors.New("wrapped asset id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", coinbaseproWrappedAssets, wrappedAssetID)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetWrappedAssetConversionRate returns the conversion rate for a wrapped asset
func (c *CoinbasePro) GetWrappedAssetConversionRate(ctx context.Context, wrappedAssetID string) (WrappedAssetConversionRate, error) {
	var resp WrappedAssetConversionRate

	if wrappedAssetID == "" {
		return resp, errors.New("wrapped asset id cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproWrappedAssets, wrappedAssetID, coinbaseproConversionRate)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// // GetCurrentServerTime returns the API server time
// func (c *CoinbasePro) GetCurrentServerTime(ctx context.Context) (ServerTime, error) {
// 	serverTime := ServerTime{}
// 	return serverTime, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproTime, &serverTime)
// }

// // MarginTransfer sends funds between a standard/default profile and a margin
// // profile.
// // A deposit will transfer funds from the default profile into the margin
// // profile. A withdraw will transfer funds from the margin profile to the
// // default profile. Withdraws will fail if they would set your margin ratio
// // below the initial margin ratio requirement.
// //
// // amount - the amount to transfer between the default and margin profile
// // transferType - either "deposit" or "withdraw"
// // profileID - The id of the margin profile to deposit or withdraw from
// // currency - currency to transfer, currently on "BTC" or "USD"
// func (c *CoinbasePro) MarginTransfer(ctx context.Context, amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
// 	resp := MarginTransfer{}
// 	req := make(map[string]interface{})
// 	req["type"] = transferType
// 	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
// 	req["currency"] = currency
// 	req["margin_profile_id"] = profileID

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproMarginTransfer, req, &resp)
// }

// // GetPosition returns an overview of account profile.
// func (c *CoinbasePro) GetPosition(ctx context.Context) (AccountOverview, error) {
// 	resp := AccountOverview{}

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPosition, nil, &resp)
// }

// // ClosePosition closes a position and allowing you to repay position as well
// // repayOnly -  allows the position to be repaid
// func (c *CoinbasePro) ClosePosition(ctx context.Context, repayOnly bool) (AccountOverview, error) {
// 	resp := AccountOverview{}
// 	req := make(map[string]interface{})
// 	req["repay_only"] = repayOnly

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPositionClose, req, &resp)
// }

// // GetReportStatus once a report request has been accepted for processing, the
// // status is available by polling the report resource endpoint.
// func (c *CoinbasePro) GetReportStatus(ctx context.Context, reportID string) (Report, error) {
// 	resp := Report{}
// 	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

// 	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
// }

// // GetTrailingVolume this request will return your 30-day trailing volume for
// // all products.
// func (c *CoinbasePro) GetTrailingVolume(ctx context.Context) ([]Volume, error) {
// 	var resp []Volume

// 	return resp,
// 		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproTrailingVolume, nil, &resp)
// }

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *CoinbasePro) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
	}

	return c.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, bodyParams map[string]interface{}, result interface{}, returnHead *http.Header) (err error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	newRequest := func() (*request.Item, error) {
		payload := []byte("")
		if bodyParams != nil {
			payload, err = json.Marshal(bodyParams)
			if err != nil {
				return nil, err
			}
		}

		n := strconv.FormatInt(time.Now().Unix(), 10)
		message := n + method + "/" + path + string(payload)

		fmt.Println(message)

		hmac, err := crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["CB-ACCESS-KEY"] = creds.Key
		headers["CB-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		headers["CB-ACCESS-TIMESTAMP"] = n
		headers["CB-ACCESS-PASSPHRASE"] = creds.ClientID
		headers["Content-Type"] = "application/json"

		return &request.Item{
			Method:         method,
			Path:           endpoint + path,
			Headers:        headers,
			Body:           bytes.NewBuffer(payload),
			Result:         result,
			Verbose:        c.Verbose,
			HTTPDebugging:  c.HTTPDebugging,
			HTTPRecording:  c.HTTPRecording,
			HeaderResponse: returnHead,
		}, nil
	}
	err = c.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
	return err
}

// // GetFee returns an estimate of fee based on type of transaction
// func (c *CoinbasePro) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
// 	var fee float64
// 	switch feeBuilder.FeeType {
// 	case exchange.CryptocurrencyTradeFee:
// 		fees, err := c.GetFees(ctx)
// 		if err != nil {
// 			fee = fees.TakerFeeRate
// 		} else {
// 			fee = 0.006
// 		}
// 	case exchange.InternationalBankWithdrawalFee:
// 		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency)
// 	case exchange.InternationalBankDepositFee:
// 		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency)
// 	case exchange.OfflineTradeFee:
// 		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
// 	}

// 	if fee < 0 {
// 		fee = 0
// 	}

// 	return fee, nil
// }

// // getOfflineTradeFee calculates the worst case-scenario trading fee
// func getOfflineTradeFee(price, amount float64) float64 {
// 	return 0.0025 * price * amount
// }

// func (c *CoinbasePro) calculateTradingFee(trailingVolume []Volume, base, quote currency.Code, delimiter string, purchasePrice, amount float64, isMaker bool) float64 {
// 	var fee float64
// 	for _, i := range trailingVolume {
// 		if strings.EqualFold(i.ProductID, base.String()+delimiter+quote.String()) {
// 			switch {
// 			case isMaker:
// 				fee = 0
// 			case i.Volume <= 10000000:
// 				fee = 0.003
// 			case i.Volume > 10000000 && i.Volume <= 100000000:
// 				fee = 0.002
// 			case i.Volume > 100000000:
// 				fee = 0.001
// 			}
// 			break
// 		}
// 	}
// 	return fee * amount * purchasePrice
// }

// func getInternationalBankWithdrawalFee(c currency.Code) float64 {
// 	var fee float64

// 	if c.Equal(currency.USD) {
// 		fee = 25
// 	} else if c.Equal(currency.EUR) {
// 		fee = 0.15
// 	}

// 	return fee
// }

// func getInternationalBankDepositFee(c currency.Code) float64 {
// 	var fee float64

// 	if c.Equal(currency.USD) {
// 		fee = 10
// 	} else if c.Equal(currency.EUR) {
// 		fee = 0.15
// 	}

// 	return fee
// }

// PrepareDSL adds the direction, step, and limit queries for pagination
func (p *Params) PrepareDSL(direction, step string, limit int64) {
	p.urlVals.Set(direction, step)
	if limit >= 0 {
		p.urlVals.Set("limit", strconv.FormatInt(limit, 10))
	}
}

// PrepareDateString encodes a set of parameters indicating start & end dates
func (p *Params) PrepareDateString(startDate, endDate time.Time) error {
	err := common.StartEndTimeCheck(startDate, endDate)

	if err == nil {
		p.urlVals.Set("start_date", startDate.Format(time.RFC3339))
		p.urlVals.Set("end_date", endDate.Format(time.RFC3339))
	}

	if err != nil {
		if err.Error() == "start date unset" || err.Error() == "end date unset" {
			return nil
		}
	}

	return err
}

// PrepareProfIDAndProdID encodes a set of parameters indicating profile and product IDs
func (p *Params) PrepareProfIDAndProdID(profileID, currencyPair string) {
	p.urlVals.Set("profile_id", profileID)
	p.urlVals.Set("product_id", currencyPair)
}

// PrepareAddAddress constructs an element of a slice to be passed to the
// AddAddresses function
func PrepareAddAddress(currency, address, destination_tag, label, vaspID string,
	verifiedSelfHosted bool) (AddAddressRequest, error) {
	if address == "" {
		return AddAddressRequest{}, errors.New("address cannot be empty")
	}
	vIDCheck := []string{"Coinbase", "Anchorage", "Balance", "bitFlyer", "BitGo",
		"Bittrex", "BlockFi", "Circle", "Coinhako", "Fidelity", "Gemini", "Huobi",
		"Kraken", "Paxos", "PayPal", "Robinhood", "Shakepay", "StandardCustody",
		"Tradestation", "Zero Hash", "Bitstamp"}
	if vaspID != "" && !common.StringDataCompare(vIDCheck, vaspID) {
		return AddAddressRequest{},
			errors.New("vaspID must be one of the following or empty: " +
				strings.Join(vIDCheck, ", "))
	}

	req := AddAddressRequest{currency, To{address, destination_tag}, label,
		verifiedSelfHosted, vaspID}

	// TODO: It also lets us add an arbitrary amount of strings under this object,
	// but doesn't explain what they do. Investigate more later.

	return req, nil
}

// OrderbookHelper handles the transfer of bids and asks of unclear levels, to a
// generalised format
func OrderbookHelper(iOD InterOrderDetail, level int32) ([]GenOrderDetail, error) {
	gOD := make([]GenOrderDetail, len(iOD))

	for i := range iOD {
		priceConv, ok := iOD[i][0].(string)
		if !ok {
			return nil, errors.New("unable to type assert price")
		}
		price, err := strconv.ParseFloat(priceConv, 64)
		if err != nil {
			return nil, err
		}
		gOD[i].Price = price

		amountConv, ok := iOD[i][1].(string)
		if !ok {
			return nil, errors.New("unable to type assert amount")
		}
		amount, err := strconv.ParseFloat(amountConv, 64)
		if err != nil {
			return nil, err
		}
		gOD[i].Amount = amount

		if level == 3 {
			orderID, ok := iOD[i][2].(string)
			if !ok {
				return nil, errors.New("unable to type assert order ID")
			}
			gOD[i].OrderID = orderID
		} else {
			numOrders, ok := iOD[i][2].(float64)
			if !ok {
				return nil, errors.New("unable to type assert number of orders")
			}
			gOD[i].NumOrders = numOrders
		}

	}
	return gOD, nil

}
