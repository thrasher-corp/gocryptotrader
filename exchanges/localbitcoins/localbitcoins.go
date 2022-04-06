package localbitcoins

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	localbitcoinsAPIURL = "https://localbitcoins.com"

	// Autheticated Calls
	localbitcoinsAPIAccountInfo        = "api/account_info"
	localbitcoinsAPIMyself             = "myself/"
	localbitcoinsAPIAds                = "ads/"
	localbitcoinsAPIAdGet              = "ad-get/"
	localbitcoinsAPIAdEdit             = "ad/"
	localbitcoinsAPIAdCreate           = "ad-create/"
	localbitcoinsAPIUpdateEquation     = "ad-equation/"
	localbitcoinsAPIDeleteAd           = "ad-delete/"
	localbitcoinsAPIRelease            = "contact_release/"
	localbitcoinsAPIReleaseByPin       = "contact_release_pin/"
	localbitcoinsAPIMarkAsPaid         = "contact_mark_as_paid/"
	localbitcoinsAPIMessages           = "contact_messages/"
	localbitcoinsAPISendMessage        = "contact_message_post/"
	localbitcoinsAPIDispute            = "contact_dispute/"
	localbitcoinsAPICancelTrade        = "contact_cancel/"
	localbitcoinsAPIFundTrade          = "contact_fund/"
	localbitcoinsAPIConfirmRealName    = "contact_mark_realname/"
	localbitcoinsAPIVerifyIdentity     = "contact_mark_identified/"
	localbitcoinsAPIInitiateTrade      = "contact_create/"
	localbitcoinsAPITradeInfo          = "contact_info/"
	localbitcoinsAPIDashboard          = "dashboard/"
	localbitcoinsAPIDashboardReleased  = "dashboard/released/"
	localbitcoinsAPIDashboardCancelled = "dashboard/canceled/"
	localbitcoinsAPIDashboardClosed    = "dashboard/closed/"
	localbitcoinsAPIFeedback           = "feedback/"
	localbitcoinsAPILogout             = "logout/"
	localbitcoinsAPICreateInvoice      = "merchant/new_invoice/"
	localbitcoinsAPIGetNotification    = "notifications/"
	localbitcoinsAPIMarkNotification   = "notifications/mark_as_read/"
	localbitcoinsAPIPinCode            = "pincode/"
	localbitcoinsAPIVerifyUsername     = "real_name_verifiers/"
	localbitcoinsAPIWallet             = "wallet/"
	localbitcoinsAPIWalletBalance      = "wallet-balance/"
	localbitcoinsAPIWalletSend         = "wallet-send/"
	localbitcoinsAPIWalletSendPin      = "wallet-send-pin/"
	localbitcoinsAPIWalletAddress      = "wallet-addr/"

	// Un-Autheticated Calls
	localbitcoinsAPICountryCodes   = "/api/countrycodes/"
	localbitcoinsAPICurrencies     = "/api/currencies/"
	localbitcoinsAPIPaymentMethods = "/api/payment_methods/"
	localbitcoinsAPIPlaces         = "/api/places/"
	localbitcoinsAPITicker         = "/bitcoinaverage/ticker-all-currencies/"
	localbitcoinsAPIBitcoincharts  = "/bitcoincharts/"
	localbitcoinsAPICashBuy        = "/buy-bitcoins-with-cash/"
	localbitcoinsAPIOnlineBuy      = "/buy-bitcoins-online/"
	localbitcoinsAPIOrderbook      = "/orderbook.json"
	localbitcoinsAPITrades         = "/trades.json"

	// Trade Types
	tradeTypeLocalSell  = "LOCAL_SELL"
	tradeTypeLocalBuy   = "LOCAL_BUY"
	tradeTypeOnlineSell = "ONLINE_SELL"
	tradeTypeOnlineBuy  = "ONLINE_BUY"

	// Reference Types
	refTypeShort   = "SHORT"
	refTypeLong    = "LONG"
	refTypeNumbers = "NUMBERS"
	refTypeLetters = "LETTERS"

	// Feedback Values
	feedbackTrust                = "trust"
	feedbackPositive             = "positive"
	feedbackNeutral              = "neutral"
	feedbackBlock                = "block"
	feedbackBlockWithoutFeedback = "block_without_feedback"

	// State Values
	stateNotOpened           = "NOT_OPENED"
	stateWaitingForPayment   = "WAITING_FOR_PAYMENT"
	statePaid                = "PAID"
	stateNotPaid             = "DIDNT_PAID"
	statePaidLate            = "PAID_IN_LATE"
	statePartlyPaid          = "PAID_PARTLY"
	statePaidAndConfirmed    = "PAID_AND_CONFIRMED"
	statePaidLateConfirmed   = "PAID_IN_LATE_AND_CONFIRMED"
	statePaidPartlyConfirmed = "PAID_PARTLY_AND_CONFIRMED"

	// String response used with order status
	null = "null"
)

var (
	// Payment Methods
	paymentMethodOne string
)

// LocalBitcoins is the overarching type across the localbitcoins package
type LocalBitcoins struct {
	exchange.Base
}

// GetAccountInformation lets you retrieve the public user information on a
// LocalBitcoins user. The response contains the same information that is found
// on an account's public profile page.
func (l *LocalBitcoins) GetAccountInformation(ctx context.Context, username string, self bool) (AccountInfo, error) {
	type response struct {
		Data AccountInfo `json:"data"`
	}
	resp := response{}

	if self {
		err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIMyself, nil, &resp)
		if err != nil {
			return resp.Data, err
		}
	} else {
		path := fmt.Sprintf("/%s/%s/", localbitcoinsAPIAccountInfo, username)
		err := l.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp, request.Unset)
		if err != nil {
			return resp.Data, err
		}
	}

	return resp.Data, nil
}

// Getads returns information of single advertisement based on the ad ID, if
// adID omitted.
//
// adID - [optional] string if omitted returns all ads
func (l *LocalBitcoins) Getads(ctx context.Context, args ...string) (AdData, error) {
	var resp struct {
		Data  AdData `json:"data"`
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"error_code"`
		} `json:"error"`
	}

	var err error
	if len(args) == 0 {
		err = l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
			localbitcoinsAPIAds,
			nil,
			&resp)
	} else {
		params := url.Values{"ads": {strings.Join(args, ",")}}
		err = l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
			localbitcoinsAPIAdGet,
			params,
			&resp)
	}

	if err != nil {
		return resp.Data, err
	}

	if resp.Error.Message != "" {
		return resp.Data, errors.New(resp.Error.Message)
	}
	return resp.Data, nil
}

// EditAd updates set advertisements
//
// params - see localbitcoins_types.go AdEdit for reference
// adID - string for the ad you already created
// TODO
func (l *LocalBitcoins) EditAd(ctx context.Context, _ *AdEdit, adID string) error {
	resp := struct {
		Data  AdData `json:"data"`
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"error_code"`
		}
	}{}

	err := l.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		localbitcoinsAPIAdEdit+adID+"/",
		nil,
		&resp)
	if err != nil {
		return err
	}

	if resp.Error.Message != "" {
		return errors.New(resp.Error.Message)
	}

	return nil
}

// CreateAd creates a new advertisement
//
// params - see localbitcoins_types.go AdCreate for reference
// TODO
func (l *LocalBitcoins) CreateAd(ctx context.Context, _ *AdCreate) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIAdCreate, nil, nil)
}

// UpdatePriceEquation updates price equation of an advertisement. If there are
// problems with new equation, the price and equation are not updated and
// advertisement remains visible.
//
// equation - string of equation
// adID - string of specific ad identification
// TODO
func (l *LocalBitcoins) UpdatePriceEquation(ctx context.Context, adID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIUpdateEquation+adID, nil, nil)
}

// DeleteAd deletes the advertisement by adID.
//
// adID - string of specific ad identification
// TODO
func (l *LocalBitcoins) DeleteAd(ctx context.Context, adID string) error {
	resp := struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"error_code"`
		} `json:"error"`
	}{}

	err := l.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		localbitcoinsAPIDeleteAd+adID+"/",
		nil,
		&resp)
	if err != nil {
		return err
	}

	if resp.Error.Message != "" {
		return errors.New(resp.Error.Message)
	}

	return nil
}

// ReleaseFunds releases Bitcoin trades specified by ID {contact_id}. If the
// release was successful a message is returned on the data key.
func (l *LocalBitcoins) ReleaseFunds(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIRelease+contactID, nil, nil)
}

// ReleaseFundsByPin releases Bitcoin trades specified by ID {contact_id}. if
// the current pincode is provided. If the release was successful a message is
// returned on the data key.
// TODO
func (l *LocalBitcoins) ReleaseFundsByPin(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIReleaseByPin+contactID, nil, nil)
}

// MarkAsPaid marks a trade as paid.
func (l *LocalBitcoins) MarkAsPaid(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIMarkAsPaid+contactID, nil, nil)
}

// GetMessages returns all chat messages from the trade. Messages are on the message_list key.
func (l *LocalBitcoins) GetMessages(ctx context.Context, contactID string) (Message, error) {
	type response struct {
		MessageList Message `json:"message_list"`
	}
	resp := response{}

	return resp.MessageList,
		l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIMessages+contactID, nil, &resp)
}

// SendMessage posts a message and/or uploads an image to the trade. Encode
// images with multipart/form-data encoding.
// TODO
func (l *LocalBitcoins) SendMessage(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPISendMessage+contactID, nil, nil)
}

// Dispute starts a dispute on the specified trade ID if the requirements for
// starting the dispute has been fulfilled.
//
// topic - [optional] String	Short description of issue to LocalBitcoins customer support.
// TODO
func (l *LocalBitcoins) Dispute(ctx context.Context, _, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIDispute+contactID, nil, nil)
}

// CancelTrade cancels the trade if the token owner is the Bitcoin buyer.
// Bitcoin sellers cannot cancel trades.
func (l *LocalBitcoins) CancelTrade(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPICancelTrade+contactID, nil, nil)
}

// FundTrade attempts to fund an unfunded local trade from the token owners
// wallet. Works only if the token owner is the Bitcoin seller in the trade.
func (l *LocalBitcoins) FundTrade(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIFundTrade+contactID, nil, nil)
}

// ConfirmRealName creates or updates real name confirmation.
func (l *LocalBitcoins) ConfirmRealName(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIConfirmRealName+contactID, nil, nil)
}

// VerifyIdentity marks the identity of trade partner as verified. You must be
// the advertiser in this trade.
func (l *LocalBitcoins) VerifyIdentity(ctx context.Context, contactID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIVerifyIdentity+contactID, nil, nil)
}

// InitiateTrade sttempts to start a Bitcoin trade from the specified
// advertisement ID.
// TODO
func (l *LocalBitcoins) InitiateTrade(ctx context.Context, adID string) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIInitiateTrade+adID, nil, nil)
}

// GetTradeInfo returns information about a single trade that the token owner is
// part in.
func (l *LocalBitcoins) GetTradeInfo(ctx context.Context, contactID string) (dbi DashBoardInfo, err error) {
	err = l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPITradeInfo+contactID+"/", nil, &dbi)
	return
}

// GetCountryCodes returns a list of valid and recognized countrycodes
func (l *LocalBitcoins) GetCountryCodes(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPICountryCodes, nil, request.Unset)
}

// GetCurrencies returns a list of valid and recognized fiat currencies. Also
// contains human readable name for every currency and boolean that tells if
// currency is an altcoin.
func (l *LocalBitcoins) GetCurrencies(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPICurrencies, nil, request.Unset)
}

// GetDashboardInfo returns a list of trades on the data key contact_list. This
// API end point mirrors the website's dashboard, allowing access to contacts in
// different states.
// In addition all of these listings have buyer/ and seller/ sub-listings to
// view contacts where the token owner is either buying or selling, respectively.
// E.g. /api/dashboard/buyer/. All contacts where the token owner is
// participating are returned.
func (l *LocalBitcoins) GetDashboardInfo(ctx context.Context) ([]DashBoardInfo, error) {
	var resp struct {
		Data struct {
			ContactList  []DashBoardInfo `json:"contact_list"`
			ContactCount int             `json:"contact_count"`
		}
	}

	return resp.Data.ContactList,
		l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIDashboard, nil, &resp)
}

// GetDashboardReleasedTrades returns a list of all released trades where the
// token owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardReleasedTrades(ctx context.Context) ([]DashBoardInfo, error) {
	var resp struct {
		Data struct {
			ContactList  []DashBoardInfo `json:"contact_list"`
			ContactCount int             `json:"contact_count"`
		}
	}

	return resp.Data.ContactList,
		l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIDashboardReleased, nil, &resp)
}

// GetDashboardCancelledTrades returns a list of all canceled trades where the
// token owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardCancelledTrades(ctx context.Context) ([]DashBoardInfo, error) {
	var resp struct {
		Data struct {
			ContactList  []DashBoardInfo `json:"contact_list"`
			ContactCount int             `json:"contact_count"`
		}
	}

	return resp.Data.ContactList,
		l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIDashboardCancelled, nil, &resp)
}

// GetDashboardClosedTrades returns a list of all closed trades where the token
// owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardClosedTrades(ctx context.Context) ([]DashBoardInfo, error) {
	var resp struct {
		Data struct {
			ContactList  []DashBoardInfo `json:"contact_list"`
			ContactCount int             `json:"contact_count"`
		}
	}

	return resp.Data.ContactList,
		l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIDashboardClosed, nil, &resp)
}

// SetFeedback gives feedback to user. Possible feedback values are: trust,
// positive, neutral, block, block_without_feedback, (check const values)
// This is only possible to set if there is a trade between the token owner and
// the user specified in {username} that is canceled or released. You may also
// set feedback message using msg field with few exceptions. Feedback
// block_without_feedback clears the message and with block the message is
// mandatory.
//
// feedback - string (use const valuesfor feedback)
// msg - [optional] Feedback message displayed alongside feedback on receivers
// profile page.
// username - username of trade contact
// TODO
func (l *LocalBitcoins) SetFeedback(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIFeedback, nil, nil)
}

// Logout expires the current access token immediately. To get a new token
// afterwards, public apps will need to re-authenticate, confidential apps can
// turn in a refresh token.
func (l *LocalBitcoins) Logout(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPILogout, nil, nil)
}

// CreateNewInvoice creates a new invoice.
// TODO
func (l *LocalBitcoins) CreateNewInvoice(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPICreateInvoice, nil, nil)
}

// GetInvoice returns information about a specific invoice created by the token
// owner.
// TODO
func (l *LocalBitcoins) GetInvoice(ctx context.Context) (Invoice, error) {
	resp := Invoice{}
	return resp, l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPICreateInvoice, nil, &resp)
}

// DeleteInvoice deletes a specific invoice. Deleting invoices is possible when
// it is sure that receiver cannot accidentally pay the invoice at the same time
// as the merchant is deleting it. You can use the API request
// /api/merchant/invoice/{invoice_id}/ to check if deleting is possible.
// TODO
func (l *LocalBitcoins) DeleteInvoice(ctx context.Context) (Invoice, error) {
	resp := Invoice{}
	return resp, l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPICreateInvoice, nil, &resp)
}

// GetNotifications returns recent notifications.
func (l *LocalBitcoins) GetNotifications(ctx context.Context) ([]NotificationInfo, error) {
	var resp []NotificationInfo
	return resp, l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIGetNotification, nil, &resp)
}

// MarkNotifications marks a specific notification as read.
// TODO
func (l *LocalBitcoins) MarkNotifications(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIMarkNotification, nil, nil)
}

// GetPaymentMethods returns a list of valid payment methods. Also contains name
// and code for payment methods, and possible limitations in currencies and bank
// name choices.
func (l *LocalBitcoins) GetPaymentMethods(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPIPaymentMethods, nil, request.Unset)
}

// GetPaymentMethodsByCountry returns a list of valid payment methods filtered
// by countrycodes.
func (l *LocalBitcoins) GetPaymentMethodsByCountry(ctx context.Context, countryCode string) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPIPaymentMethods+countryCode, nil, request.Unset)
}

// CheckPincode checks the given PIN code against the token owners currently
// active PIN code. You can use this method to ensure the person using the
// session is the legitimate user.
// Due to only requiring the read scope, the user is not guaranteed to have set
// a PIN code. If you protect your application using this request, please make
// the user has set a PIN code for his account.
func (l *LocalBitcoins) CheckPincode(ctx context.Context, pin int) (bool, error) {
	type response struct {
		Data struct {
			PinOK bool `json:"pincode_ok"`
		} `json:"data"`
	}
	resp := response{}
	values := url.Values{}
	values.Set("pincode", strconv.Itoa(pin))
	err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIPinCode, values, &resp)

	if err != nil {
		return false, err
	}

	if !resp.Data.PinOK {
		return false, errors.New("pin invalid")
	}

	return true, nil
}

// GetPlaces Looks up places near lat, lon and provides full URLs to buy and
// sell listings for each.
// TODO
func (l *LocalBitcoins) GetPlaces(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPIPlaces, nil, request.Unset)
}

// VerifyUsername returns list of real name verifiers for the user. Returns a
// list only when you have a trade with the user where you are the seller.
func (l *LocalBitcoins) VerifyUsername(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIVerifyUsername, nil, nil)
}

// GetRecentMessages returns maximum of 25 newest trade messages. Does not
// return messages older than one month. Messages are ordered by sending time,
// and the newest one is first.
// TODO
func (l *LocalBitcoins) GetRecentMessages(ctx context.Context) error {
	return l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIVerifyUsername, nil, nil)
}

// GetWalletInfo gets information about the token owner's wallet balance.
func (l *LocalBitcoins) GetWalletInfo(ctx context.Context) (WalletInfo, error) {
	type response struct {
		Data WalletInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIWallet, nil, &resp)

	if err != nil {
		return WalletInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return WalletInfo{}, errors.New("unable to fetch wallet info")
	}

	return resp.Data, nil
}

// GetWalletBalance Same as GetWalletInfo(), but only returns the message,
// receiving_address and total fields.
// Use this instead if you don't care about transactions at the moment.
func (l *LocalBitcoins) GetWalletBalance(ctx context.Context) (WalletBalanceInfo, error) {
	type response struct {
		Data WalletBalanceInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, localbitcoinsAPIWalletBalance, nil, &resp)

	if err != nil {
		return WalletBalanceInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return WalletBalanceInfo{}, errors.New("unable to fetch wallet balance")
	}

	return resp.Data, nil
}

// WalletSend sends amount of bitcoins from the token owner's wallet to address.
// On success, the response returns a message indicating success. It is highly
// recommended to minimize the lifetime of access tokens with the money
// permission. Use Logout() to make the current token expire instantly.
func (l *LocalBitcoins) WalletSend(ctx context.Context, address string, amount float64, pin int64) error {
	values := url.Values{}
	values.Set("address", address)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := localbitcoinsAPIWalletSend

	if pin > 0 {
		values.Set("pincode", strconv.FormatInt(pin, 10))
		path = localbitcoinsAPIWalletSendPin
	}

	resp := struct {
		Error struct {
			Message string            `json:"message"`
			Errors  map[string]string `json:"errors"`
			Code    int               `json:"error_code"`
		} `json:"error"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}{}

	err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, values, &resp)
	if err != nil {
		return err
	}

	if resp.Data.Message != "Money is being sent" {
		if len(resp.Error.Errors) != 0 {
			var details strings.Builder
			for x := range resp.Error.Errors {
				details.WriteString(resp.Error.Errors[x])
			}
			return errors.New(details.String())
		}
		return errors.New(resp.Data.Message)
	}

	return nil
}

// GetWalletAddress returns an unused receiving address from the token owner's
// wallet. The address is returned in the address key of the response. Note that
// this API may keep returning the same (unused) address if requested repeatedly.
func (l *LocalBitcoins) GetWalletAddress(ctx context.Context) (string, error) {
	type response struct {
		Data struct {
			Message string `json:"message"`
			Address string `json:"address"`
		}
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, localbitcoinsAPIWalletAddress, nil, &resp)
	if err != nil {
		return "", err
	}

	if resp.Data.Message != "OK!" {
		return "", errors.New("unable to fetch wallet address")
	}

	return resp.Data.Address, nil
}

// GetBitcoinsWithCashAd returns buy or sell as cash local advertisements.
// TODO
func (l *LocalBitcoins) GetBitcoinsWithCashAd(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPICashBuy, nil, request.Unset)
}

// GetBitcoinsOnlineAd this API returns buy or sell Bitcoin online ads.
// TODO
func (l *LocalBitcoins) GetBitcoinsOnlineAd(ctx context.Context) error {
	return l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPIOnlineBuy, nil, request.Unset)
}

// GetTicker returns list of all completed trades.
func (l *LocalBitcoins) GetTicker(ctx context.Context) (map[string]Ticker, error) {
	result := make(map[string]Ticker)
	return result, l.SendHTTPRequest(ctx, exchange.RestSpot, localbitcoinsAPITicker, &result, tickerLimiter)
}

// GetTradableCurrencies returns a list of tradable fiat currencies
func (l *LocalBitcoins) GetTradableCurrencies(ctx context.Context) ([]string, error) {
	resp, err := l.GetTicker(ctx)
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range resp {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// GetTrades returns all closed trades in online buy and online sell categories,
// updated every 15 minutes.
func (l *LocalBitcoins) GetTrades(ctx context.Context, currency string, values url.Values) ([]Trade, error) {
	endpoint := localbitcoinsAPIBitcoincharts + currency + localbitcoinsAPITrades
	path := common.EncodeURLValues(endpoint, values)
	var result []Trade
	return result, l.SendHTTPRequest(ctx, exchange.RestSpot, path, &result, request.Unset)
}

// GetOrderbook returns buy and sell bitcoin online advertisements. Amount is
// the maximum amount available for the trade request. Price is the hourly
// updated price. The price is based on the price equation and commission %
// entered by the ad author.
func (l *LocalBitcoins) GetOrderbook(ctx context.Context, currency string) (Orderbook, error) {
	type response struct {
		Bids [][2]string `json:"bids"`
		Asks [][2]string `json:"asks"`
	}

	path := localbitcoinsAPIBitcoincharts + currency + localbitcoinsAPIOrderbook
	resp := response{}
	var ob Orderbook
	if err := l.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp, orderBookLimiter); err != nil {
		return ob, err
	}

	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return ob, err
		}
		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return ob, err
		}
		ob.Bids = append(ob.Bids, Price{price, amount})
	}

	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return ob, err
		}
		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return ob, err
		}
		ob.Asks = append(ob.Asks, Price{price, amount})
	}

	return ob, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (l *LocalBitcoins) SendHTTPRequest(ctx context.Context, endpoint exchange.URL, path string, result interface{}, ep request.EndpointLimit) error {
	ePoint, err := l.API.Endpoints.GetURL(endpoint)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          ePoint + path,
		Result:        result,
		Verbose:       l.Verbose,
		HTTPDebugging: l.HTTPDebugging,
		HTTPRecording: l.HTTPRecording,
	}

	return l.SendPayload(ctx, ep, func() (*request.Item, error) {
		return item, nil
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to
// localbitcoins
func (l *LocalBitcoins) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, result interface{}) (err error) {
	creds, err := l.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := l.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	return l.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := l.Requester.GetNonce(true).String()

		fullPath := "/api/" + path
		encoded := params.Encode()
		message := n + creds.Key + fullPath + encoded
		hmac, err := crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["Apiauth-Key"] = creds.Key
		headers["Apiauth-Nonce"] = n
		headers["Apiauth-Signature"] = strings.ToUpper(crypto.HexEncodeToString(hmac))
		headers["Content-Type"] = "application/x-www-form-urlencoded"

		if method == http.MethodGet && len(encoded) > 0 {
			fullPath += "?" + encoded
		}

		return &request.Item{
			Method:        method,
			Path:          endpoint + fullPath,
			Headers:       headers,
			Body:          bytes.NewBufferString(encoded),
			Result:        result,
			AuthRequest:   true,
			NonceEnabled:  true,
			Verbose:       l.Verbose,
			HTTPDebugging: l.HTTPDebugging,
			HTTPRecording: l.HTTPRecording,
		}, nil
	})
}

// GetFee returns an estimate of fee based on type of transaction
func (l *LocalBitcoins) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	// No fees will be used
	return 0, nil
}
