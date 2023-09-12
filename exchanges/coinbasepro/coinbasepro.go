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
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	coinbaseproAPIURL               = "https://api.exchange.coinbase.com/"
	coinbaseproSandboxAPIURL        = "https://api-public.sandbox.exchange.coinbase.com/"
	coinbaseproAPIVersion           = "0"
	coinbaseproAccounts             = "accounts"
	coinbaseproHolds                = "holds"
	coinbaseproLedger               = "ledger"
	coinbaseproTransfers            = "transfers"
	coinbaseproAddressBook          = "address-book"
	coinbaseproAddress              = "addresses"
	coinbaseproConversions          = "conversions"
	coinbaseproCurrencies           = "currencies"
	coinbaseproDepositCoinbase      = "deposits/coinbase-account"
	coinbaseproPaymentMethodDeposit = "deposits/payment-method"
	coinbaseproPaymentMethod        = "payment-methods"
	coinbaseproFeeEstimate          = "withdrawals/fee-estimate"
	coinbaseproFees                 = "fees"
	coinbaseproFills                = "fills"
	coinbaseproOrders               = "orders"
	coinbaseproOracle               = "oracle"

	coinbaseproProducts                = "products"
	coinbaseproOrderbook               = "book"
	coinbaseproTicker                  = "ticker"
	coinbaseproTrades                  = "trades"
	coinbaseproHistory                 = "candles"
	coinbaseproStats                   = "stats"
	coinbaseproReports                 = "reports"
	coinbaseproTime                    = "time"
	coinbaseproTravelRules             = "travel-rules"
	coinbaseproMarginTransfer          = "profiles/margin-transfer"
	coinbaseproPosition                = "position"
	coinbaseproPositionClose           = "position/close"
	coinbaseproWithdrawalPaymentMethod = "withdrawals/payment-method"
	coinbaseproWithdrawalCoinbase      = "withdrawals/coinbase-account"
	coinbaseproWithdrawalCrypto        = "withdrawals/crypto"
	coinbaseproCoinbaseAccounts        = "coinbase-accounts"
	coinbaseproTrailingVolume          = "users/self/trailing-volume"
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

// GetAccounts returns a list of trading accounts associated with the APIKEYS
func (c *CoinbasePro) GetAccounts(ctx context.Context) ([]AccountResponse, error) {
	var resp []AccountResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAccounts, nil, &resp)
}

// GetAccount returns information for a single account. Use this endpoint when
// account_id is known
func (c *CoinbasePro) GetAccount(ctx context.Context, accountID string) (AccountResponse, error) {
	resp := AccountResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproAccounts, accountID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetHolds returns the holds that are placed on an account for any active
// orders or pending withdraw requests. As an order is filled, the hold amount
// is updated. If an order is canceled, any remaining hold is removed. For a
// withdraw, once it is completed, the hold is removed.
func (c *CoinbasePro) GetHolds(ctx context.Context, accountID, direction, step string, limit int64) ([]AccountHolds, error) {
	var resp []AccountHolds
	var params Params
	params.urlVals = url.Values{}
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproHolds)

	params.PrepareDSL(direction, step, limit)

	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetAccountLedger returns a list of ledger activity. Anything that increases
// or decreases your account balance. Items are paginated and sorted latest
// first.
func (c *CoinbasePro) GetAccountLedger(ctx context.Context, accountID, direction, step, pID string, startDate, endDate time.Time, limit int64) ([]AccountLedgerResponse, error) {
	var resp []AccountLedgerResponse
	var params Params
	params.urlVals = url.Values{}

	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return resp, err
	}

	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproLedger)

	params.PrepareDSL(direction, step, limit)

	if pID != "" {
		params.urlVals.Set("profile_id", pID)
	}

	path = common.EncodeURLValues(path, params.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetAccountTransfers returns a history of withdrawal and or deposit
// transactions for a single account
func (c *CoinbasePro) GetAccountTransfers(ctx context.Context, accountID, direction, step, transferType string, limit int64) ([]TransferHistory, error) {
	var resp []TransferHistory
	var params Params
	params.urlVals = url.Values{}
	path := fmt.Sprintf("%s/%s/%s", coinbaseproAccounts, accountID, coinbaseproTransfers)

	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)

	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetAddressBook returns all addresses stored in the address book
func (c *CoinbasePro) GetAddressBook(ctx context.Context) ([]GetAddressResponse, error) {
	var resp []GetAddressResponse

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproAddressBook, nil, &resp)
}

// AddAddresses adds new addresses to the address book
func (c *CoinbasePro) AddAddresses(ctx context.Context, req []AddAddressRequest) ([]AddAddressResponse, error) {
	var resp []AddAddressResponse

	params := make(map[string]interface{})
	params["addresses"] = req

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproAddressBook, params, &resp)
}

// DeleteAddress deletes an address from the address book
func (c *CoinbasePro) DeleteAddress(ctx context.Context, addressID string) error {
	path := fmt.Sprintf("%s/%s", coinbaseproAddressBook, addressID)

	return c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil)
}

// GetCoinbaseWallets returns all of the user's available Coinbase wallets
func (c *CoinbasePro) GetCoinbaseWallets(ctx context.Context) ([]CoinbaseAccounts, error) {
	var resp []CoinbaseAccounts

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproCoinbaseAccounts, nil, &resp)
}

// GenerateCryptoAddress generates a one-time address for deposting crypto
func (c *CoinbasePro) GenerateCryptoAddress(ctx context.Context, accountID, profileID, network string) (CryptoAddressResponse, error) {
	resp := CryptoAddressResponse{}
	path := fmt.Sprintf("%s/%s/%s", coinbaseproCoinbaseAccounts, accountID, coinbaseproAddress)

	// In this case, accountID has to come from GetCoinbaseWallets, not GetAccounts

	params := map[string]interface{}{"account_id": accountID, "profile_id": profileID,
		"network": network}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, &resp)
}

// ConvertCurrency converts between two currencies in the specified profile
func (c *CoinbasePro) ConvertCurrency(ctx context.Context, profileID, from, to, nonce string, amount float64) (ConvertResponse, error) {
	resp := ConvertResponse{}

	params := map[string]interface{}{"profile_id": profileID, "from": from, "to": to,
		"amount": strconv.FormatFloat(amount, 'f', -1, 64), "nonce": nonce}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproConversions, params, &resp)
}

// GetConversionByID identifies the details of a past conversion, given its ID
func (c *CoinbasePro) GetConversionByID(ctx context.Context, conversionID, profileID string) (ConvertResponse, error) {
	resp := ConvertResponse{}
	path := fmt.Sprintf("%s/%s", coinbaseproConversions, conversionID)
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("profile_id", profileID)
	path = common.EncodeURLValues(path, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetCurrencies returns a list of currencies known by the exchange
// Warning: Currencies won't necessarily be available for trading
func (c *CoinbasePro) GetCurrencies(ctx context.Context) ([]Currency, error) {
	var currencies []Currency

	return currencies,
		c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproCurrencies, &currencies)
}

// GetCurrenciesByID returns into on a single currency given its ID in ISO 4217, or
// in a custom code for currencies which lack an ISO 4217 code
func (c *CoinbasePro) GetCurrenciesByID(ctx context.Context, currencyID string) (Currency, error) {
	resp := Currency{}
	path := fmt.Sprintf("%s/%s", coinbaseproCurrencies, currencyID)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// DepositViaCoinbase deposits funds from a Coinbase account
func (c *CoinbasePro) DepositViaCoinbase(ctx context.Context, profileID, currency, coinbaseAccountID string, amount float64) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	params := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": coinbaseAccountID, "currency": currency}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproDepositCoinbase, params, &resp)
}

// DepositViaPaymentMethod deposits funds from a payment method. SEPA is not allowed
func (c *CoinbasePro) DepositViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	params := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPaymentMethodDeposit, params, &resp)
}

// GetPayMethods returns a full list of payment methods
func (c *CoinbasePro) GetPayMethods(ctx context.Context) ([]PaymentMethod, error) {
	var resp []PaymentMethod

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPaymentMethod, nil, &resp)
}

// GetAllTransfers returns all in-progress and completed transfers in and out of any
// of the user's accounts
func (c *CoinbasePro) GetAllTransfers(ctx context.Context, profileID, direction, step, transferType string, limit int64) ([]TransferHistory, error) {
	resp := []TransferHistory{}
	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("profile_id", profileID)
	params.PrepareDSL(direction, step, limit)
	params.urlVals.Set("type", transferType)
	path := common.EncodeURLValues(coinbaseproTransfers, params.urlVals)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetTransferByID returns information on a single transfer when provided with its ID
func (c *CoinbasePro) GetTransferByID(ctx context.Context, transferID string) (TransferHistory, error) {
	resp := TransferHistory{}
	path := fmt.Sprintf("%s/%s", coinbaseproTransfers, transferID)

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// SendTravelInfoForTransfer sends travel rule information for a transfer
func (c *CoinbasePro) SendTravelInfoForTransfer(ctx context.Context, transferID, originName, originCountry string) (string, error) {
	var resp string
	path := fmt.Sprintf("%s/%s/%s", coinbaseproTransfers, transferID,
		coinbaseproTravelRules)
	params := map[string]interface{}{"transfer_id": transferID,
		"originator_name": originName, "originator_country": originCountry}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, &resp)
}

// WithdrawViaCoinbase withdraws funds to a coinbase account.
func (c *CoinbasePro) WithdrawViaCoinbase(ctx context.Context, profileID, accountID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := map[string]interface{}{"profile_id": profileID,
		"amount":              strconv.FormatFloat(amount, 'f', -1, 64),
		"coinbase_account_id": accountID, "currency": currency}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCoinbase, req, &resp)
}

// WithdrawCrypto withdraws funds to a crypto address
func (c *CoinbasePro) WithdrawCrypto(ctx context.Context, profileID, currency, cryptoAddress, destinationTag, twoFactorCode, network string, amount float64, noDestinationTag, addNetworkFee bool, nonce int32) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := map[string]interface{}{"profile_id": profileID,
		"amount":   strconv.FormatFloat(amount, 'f', -1, 64),
		"currency": currency, "crypto_address": cryptoAddress,
		"destination_tag": destinationTag, "no_destination_tag": noDestinationTag,
		"two_factor_code": twoFactorCode, "nonce": nonce, "network": network,
		"add_network_fee": addNetworkFee}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalCrypto, req, &resp)
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
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// WithdrawViaPaymentMethod withdraws funds to a payment method
func (c *CoinbasePro) WithdrawViaPaymentMethod(ctx context.Context, profileID, paymentID, currency string, amount float64) (DepositWithdrawalInfo, error) {
	resp := DepositWithdrawalInfo{}
	req := map[string]interface{}{"profile_id": profileID,
		"amount":            strconv.FormatFloat(amount, 'f', -1, 64),
		"payment_method_id": paymentID, "currency": currency}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproWithdrawalPaymentMethod, req, &resp)
}

// GetFees returns your current maker & taker fee rates, as well as your 30-day
// trailing volume. Quoted rates are subject to change.
func (c *CoinbasePro) GetFees(ctx context.Context) (FeeResponse, error) {
	resp := FeeResponse{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproFees, nil, &resp)
}

// GetFills returns a list of recent fills on this profile
func (c *CoinbasePro) GetFills(ctx context.Context, orderID, currencyPair, direction, step, marketType string, limit int64, startDate, endDate time.Time) ([]FillResponse, error) {
	var resp []FillResponse
	if orderID == "" && currencyPair == "" {
		return resp, errors.New("requires either order id or product id")
	}
	var params Params
	params.urlVals = url.Values{}
	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return resp, err
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

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetOrders lists all open or unsettled orders
func (c *CoinbasePro) GetOrders(ctx context.Context, profileID, currencyPair, sortedBy, sorting, direction, step, marketType string, startDate, endDate time.Time, limit int64, status []string) ([]GeneralizedOrderResponse, error) {
	var resp []GeneralizedOrderResponse
	if limit < 1 {
		return resp, errors.New("limit must be greater than 0")
	}
	var params Params
	params.urlVals = make(url.Values)
	err := params.PrepareDateString(startDate, endDate)
	if err != nil {
		return resp, err
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

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// CancelAllExistingOrders attempts to cancel all open orders. The exchange warns that
// this may need to be called multiple times to properly close all of them
func (c *CoinbasePro) CancelAllExistingOrders(ctx context.Context, profileID, currencyPair string) ([]string, error) {
	var resp []string
	var params Params
	params.urlVals = url.Values{}

	params.PrepareProfIDAndProdID(profileID, currencyPair)

	path := common.EncodeURLValues(coinbaseproOrders, params.urlVals)
	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, &resp)
}

// PlaceOrder places either a limit, market, or stop order
func (c *CoinbasePro) PlaceOrder(ctx context.Context, profileID, orderType, side, currencyPair, stp, stop, timeInForce, cancelAfter, clientOID string, stopPrice, price, size, funds float64, postOnly bool) (GeneralizedOrderResponse, error) {
	var resp GeneralizedOrderResponse

	if (orderType == order.Market.Lower() || orderType == "" || orderType == "stop") &&
		(price == 0 || size == 0) {
		return resp, errors.New("price and size must be greater than 0 for limit or stop orders")
	}
	if orderType == order.Market.Lower() && (size == 0 && funds == 0) {
		return resp, errors.New("size or funds must be greater than 0 for market orders")
	}

	req := map[string]interface{}{"profile_id": profileID, "type": orderType, "side": side,
		"product_id": currencyPair, "stp": stp, "stop": stop,
		"stop_price":    strconv.FormatFloat(stopPrice, 'f', -1, 64),
		"price":         strconv.FormatFloat(price, 'f', -1, 64),
		"size":          strconv.FormatFloat(size, 'f', -1, 64),
		"funds":         strconv.FormatFloat(funds, 'f', -1, 64),
		"time_in_force": timeInForce, "cancel_after": cancelAfter, "post_only": postOnly,
		"client_oid": clientOID}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproOrders, req, &resp)
}

// GetOrder returns a single order by order id.
func (c *CoinbasePro) GetOrder(ctx context.Context, orderID, marketType string, clientID bool) (GeneralizedOrderResponse, error) {
	resp := GeneralizedOrderResponse{}
	if orderID == "" {
		return resp, errors.New("order id cannot be empty")
	}
	if clientID {
		orderID = fmt.Sprintf("client:%s", orderID)
	}
	path := fmt.Sprintf("%s/%s", coinbaseproOrders, orderID)
	var param Params
	param.urlVals = url.Values{}

	param.urlVals.Set("market_type", marketType)

	path = common.EncodeURLValues(path, param.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
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

	param.urlVals.Set("profile_id", profileID)
	param.urlVals.Set("product_id", productID)

	path = common.EncodeURLValues(path, param.urlVals)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, nil, nil)
}

// GetSignedPrices returns some cryptographically signed prices ready to be
// posted on-chain using Compound's Open Oracle smart contract
func (c *CoinbasePro) GetSignedPrices(ctx context.Context) (SignedPrices, error) {
	resp := SignedPrices{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproOracle, nil, &resp)
}

// GetProducts returns information on all currency pairs that are available for trading
func (c *CoinbasePro) GetProducts(ctx context.Context, productType string) ([]Product, error) {
	var products []Product

	var params Params
	params.urlVals = url.Values{}
	params.urlVals.Set("type", productType)

	path := common.EncodeURLValues(coinbaseproProducts, params.urlVals)

	return products, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &products)
}

// GetProduct returns information on a single specified currency pair
func (c *CoinbasePro) GetProduct(ctx context.Context, productID string) (Product, error) {
	resp := Product{}
	path := fmt.Sprintf("%s/%s", coinbaseproProducts, productID)

	return resp, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

func (c *CoinbasePro) OrderbookHelper(obI OrderbookIntermediaryResponse, level int32) (*OrderbookFinalResponse, error) {
	// obF := OrderbookFinalResponse{
	// 	Bids:     make([]GenOrderDetail, len(obI.Bids)),
	// 	Asks:     make([]GenOrderDetail, len(obI.Asks)),
	// 	Sequence: obI.Sequence,
	// }
	return nil, nil
}

// GetOrderbook returns orderbook by currency pair and level
func (c *CoinbasePro) GetOrderbook(ctx context.Context, symbol string, level int32) (*OrderbookFinalResponse, error) {
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

	resp, err := c.OrderbookHelper(data, level)
	if err != nil {
		return nil, err
	}

	// if level == 3 {
	// 	ob := OrderbookL3{
	// 		Sequence: orderbook.Sequence,
	// 		Bids:     make([]OrderL3, len(orderbook.Bids)),
	// 		Asks:     make([]OrderL3, len(orderbook.Asks)),
	// 	}
	// 	ob.Sequence = orderbook.Sequence
	// 	for x := range orderbook.Asks {
	// 		priceConv, ok := orderbook.Asks[x][0].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert price")
	// 		}
	// 		price, err := strconv.ParseFloat(priceConv, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		amountConv, ok := orderbook.Asks[x][1].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert amount")
	// 		}
	// 		amount, err := strconv.ParseFloat(amountConv, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		ordID, ok := orderbook.Asks[x][2].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert order ID")
	// 		}
	// 		ob.Asks[x] = OrderL3{Price: price, Amount: amount, OrderID: ordID}
	// 	}
	// 	for x := range orderbook.Bids {
	// 		priceConv, ok := orderbook.Bids[x][0].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert price")
	// 		}
	// 		price, err := strconv.ParseFloat(priceConv, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		amountConv, ok := orderbook.Bids[x][1].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert amount")
	// 		}
	// 		amount, err := strconv.ParseFloat(amountConv, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		ordID, ok := orderbook.Bids[x][2].(string)
	// 		if !ok {
	// 			return nil, errors.New("unable to type assert order ID")
	// 		}
	// 		ob.Bids[x] = OrderL3{Price: price, Amount: amount, OrderID: ordID}
	// 	}
	// 	return ob, nil
	// }
	// ob := OrderbookL1L2{
	// 	Sequence: orderbook.Sequence,
	// 	Bids:     make([]OrderL1L2, len(orderbook.Bids)),
	// 	Asks:     make([]OrderL1L2, len(orderbook.Asks)),
	// }
	// for x := range orderbook.Asks {
	// 	priceConv, ok := orderbook.Asks[x][0].(string)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert price")
	// 	}
	// 	price, err := strconv.ParseFloat(priceConv, 64)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	amountConv, ok := orderbook.Asks[x][1].(string)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert amount")
	// 	}
	// 	amount, err := strconv.ParseFloat(amountConv, 64)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	numOrders, ok := orderbook.Asks[x][2].(float64)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert number of orders")
	// 	}
	// 	ob.Asks[x] = OrderL1L2{Price: price, Amount: amount, NumOrders: numOrders}
	// }
	// for x := range orderbook.Bids {
	// 	priceConv, ok := orderbook.Bids[x][0].(string)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert price")
	// 	}
	// 	price, err := strconv.ParseFloat(priceConv, 64)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	amountConv, ok := orderbook.Bids[x][1].(string)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert amount")
	// 	}
	// 	amount, err := strconv.ParseFloat(amountConv, 64)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	numOrders, ok := orderbook.Bids[x][2].(float64)
	// 	if !ok {
	// 		return nil, errors.New("unable to type assert number of orders")
	// 	}
	// 	ob.Bids[x] = OrderL1L2{Price: price, Amount: amount, NumOrders: numOrders}
	// }
	// return ob, nil
	return resp, nil
}

// GetTicker returns ticker by currency pair
// currencyPair - example "BTC-USD"
func (c *CoinbasePro) GetTicker(ctx context.Context, currencyPair string) (Ticker, error) {
	tick := Ticker{}
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTicker)
	return tick, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &tick)
}

// GetTrades listd the latest trades for a product
// currencyPair - example "BTC-USD"
func (c *CoinbasePro) GetTrades(ctx context.Context, currencyPair string) ([]Trade, error) {
	var trades []Trade
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproTrades)
	return trades, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &trades)
}

// GetHistoricRates returns historic rates for a product. Rates are returned in
// grouped buckets based on requested granularity.
func (c *CoinbasePro) GetHistoricRates(ctx context.Context, currencyPair, start, end string, granularity int64) ([]History, error) {
	values := url.Values{}

	if len(start) > 0 {
		values.Set("start", start)
	} else {
		values.Set("start", "")
	}

	if len(end) > 0 {
		values.Set("end", end)
	} else {
		values.Set("end", "")
	}

	allowedGranularities := [6]int64{60, 300, 900, 3600, 21600, 86400}
	validGran, _ := common.InArray(granularity, allowedGranularities)
	if !validGran {
		return nil, errors.New("Invalid granularity value: " + strconv.FormatInt(granularity, 10) + ". Allowed values are {60, 300, 900, 3600, 21600, 86400}")
	}
	if granularity > 0 {
		values.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	var resp [][6]float64
	path := common.EncodeURLValues(
		fmt.Sprintf("%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproHistory),
		values)
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

// GetStats returns a 24 hr stat for the product. Volume is in base currency
// units. open, high, low are in quote currency units.
func (c *CoinbasePro) GetStats(ctx context.Context, currencyPair string) (Stats, error) {
	stats := Stats{}
	path := fmt.Sprintf(
		"%s/%s/%s", coinbaseproProducts, currencyPair, coinbaseproStats)

	return stats, c.SendHTTPRequest(ctx, exchange.RestSpot, path, &stats)
}

// GetCurrentServerTime returns the API server time
func (c *CoinbasePro) GetCurrentServerTime(ctx context.Context) (ServerTime, error) {
	serverTime := ServerTime{}
	return serverTime, c.SendHTTPRequest(ctx, exchange.RestSpot, coinbaseproTime, &serverTime)
}

// MarginTransfer sends funds between a standard/default profile and a margin
// profile.
// A deposit will transfer funds from the default profile into the margin
// profile. A withdraw will transfer funds from the margin profile to the
// default profile. Withdraws will fail if they would set your margin ratio
// below the initial margin ratio requirement.
//
// amount - the amount to transfer between the default and margin profile
// transferType - either "deposit" or "withdraw"
// profileID - The id of the margin profile to deposit or withdraw from
// currency - currency to transfer, currently on "BTC" or "USD"
func (c *CoinbasePro) MarginTransfer(ctx context.Context, amount float64, transferType, profileID, currency string) (MarginTransfer, error) {
	resp := MarginTransfer{}
	req := make(map[string]interface{})
	req["type"] = transferType
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["currency"] = currency
	req["margin_profile_id"] = profileID

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproMarginTransfer, req, &resp)
}

// GetPosition returns an overview of account profile.
func (c *CoinbasePro) GetPosition(ctx context.Context) (AccountOverview, error) {
	resp := AccountOverview{}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproPosition, nil, &resp)
}

// ClosePosition closes a position and allowing you to repay position as well
// repayOnly -  allows the position to be repaid
func (c *CoinbasePro) ClosePosition(ctx context.Context, repayOnly bool) (AccountOverview, error) {
	resp := AccountOverview{}
	req := make(map[string]interface{})
	req["repay_only"] = repayOnly

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproPositionClose, req, &resp)
}

// GetCoinbaseAccounts returns a list of coinbase accounts
func (c *CoinbasePro) GetCoinbaseAccounts(ctx context.Context) ([]CoinbaseAccounts, error) {
	var resp []CoinbaseAccounts

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproCoinbaseAccounts, nil, &resp)
}

// GetReport returns batches of historic information about your account in
// various human and machine readable forms.
//
// reportType - "fills" or "account"
// startDate - Starting date for the report (inclusive)
// endDate - Ending date for the report (inclusive)
// currencyPair - ID of the product to generate a fills report for.
// E.c. BTC-USD. *Required* if type is fills
// accountID - ID of the account to generate an account report for. *Required*
// if type is account
// format - 	pdf or csv (default is pdf)
// email - [optional] Email address to send the report to
func (c *CoinbasePro) GetReport(ctx context.Context, reportType, startDate, endDate, currencyPair, accountID, format, email string) (Report, error) {
	resp := Report{}
	req := make(map[string]interface{})
	req["type"] = reportType
	req["start_date"] = startDate
	req["end_date"] = endDate
	req["format"] = "pdf"

	if currencyPair != "" {
		req["product_id"] = currencyPair
	}
	if accountID != "" {
		req["account_id"] = accountID
	}
	if format == "csv" {
		req["format"] = format
	}
	if email != "" {
		req["email"] = email
	}

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, coinbaseproReports, req, &resp)
}

// GetReportStatus once a report request has been accepted for processing, the
// status is available by polling the report resource endpoint.
func (c *CoinbasePro) GetReportStatus(ctx context.Context, reportID string) (Report, error) {
	resp := Report{}
	path := fmt.Sprintf("%s/%s", coinbaseproReports, reportID)

	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
}

// GetTrailingVolume this request will return your 30-day trailing volume for
// all products.
func (c *CoinbasePro) GetTrailingVolume(ctx context.Context) ([]Volume, error) {
	var resp []Volume

	return resp,
		c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, coinbaseproTrailingVolume, nil, &resp)
}

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
func (c *CoinbasePro) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params map[string]interface{}, result interface{}) (err error) {
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
		if params != nil {
			payload, err = json.Marshal(params)
			if err != nil {
				return nil, err
			}
		}

		n := strconv.FormatInt(time.Now().Unix(), 10)
		message := n + method + "/" + path + string(payload)

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
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        result,
			Verbose:       c.Verbose,
			HTTPDebugging: c.HTTPDebugging,
			HTTPRecording: c.HTTPRecording,
		}, nil
	}
	return c.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		trailingVolume, err := c.GetTrailingVolume(ctx)
		if err != nil {
			return 0, err
		}
		fee = c.calculateTradingFee(trailingVolume,
			feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote,
			feeBuilder.Pair.Delimiter,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}

	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}

func (c *CoinbasePro) calculateTradingFee(trailingVolume []Volume, base, quote currency.Code, delimiter string, purchasePrice, amount float64, isMaker bool) float64 {
	var fee float64
	for _, i := range trailingVolume {
		if strings.EqualFold(i.ProductID, base.String()+delimiter+quote.String()) {
			switch {
			case isMaker:
				fee = 0
			case i.Volume <= 10000000:
				fee = 0.003
			case i.Volume > 10000000 && i.Volume <= 100000000:
				fee = 0.002
			case i.Volume > 100000000:
				fee = 0.001
			}
			break
		}
	}
	return fee * amount * purchasePrice
}

func getInternationalBankWithdrawalFee(c currency.Code) float64 {
	var fee float64

	if c.Equal(currency.USD) {
		fee = 25
	} else if c == currency.EUR {
		fee = 0.15
	}

	return fee
}

func getInternationalBankDepositFee(c currency.Code) float64 {
	var fee float64

	if c.Equal(currency.USD) {
		fee = 10
	} else if c == currency.EUR {
		fee = 0.15
	}

	return fee
}

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

// func (c *CoinbasePro) GetTravelRules(ctx context.Context) ([]TravelRule, error) {
// 	var resp []TravelRule
// 	accounts, err := c.GetAccounts(ctx)
// 	path := fmt.Sprintf("/%s/", accounts[0].ID)
// 	return resp, c.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp)
// }
