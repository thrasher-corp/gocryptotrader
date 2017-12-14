package localbitcoins

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
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
)

var (
	// Payment Methods
	paymentMethodOne string
)

// LocalBitcoins is the overarching type across the localbitcoins package
type LocalBitcoins struct {
	exchange.Base
}

// SetDefaults sets the package defaults for localbitcoins
func (l *LocalBitcoins) SetDefaults() {
	l.Name = "LocalBitcoins"
	l.Enabled = false
	l.Verbose = false
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.RequestCurrencyPairFormat.Delimiter = ""
	l.RequestCurrencyPairFormat.Uppercase = true
	l.ConfigCurrencyPairFormat.Delimiter = ""
	l.ConfigCurrencyPairFormat.Uppercase = true
}

// Setup sets exchange configuration parameters
func (l *LocalBitcoins) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the fee for maker or taker
func (l *LocalBitcoins) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	}
	return l.TakerFee
}

// GetAccountInfo lets you retrieve the public user information on a
// LocalBitcoins user. The response contains the same information that is found
// on an account's public profile page.
func (l *LocalBitcoins) GetAccountInfo(username string, self bool) (AccountInfo, error) {
	type response struct {
		Data AccountInfo `json:"data"`
	}
	resp := response{}

	if self {
		err := l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIMyself, nil, &resp)
		if err != nil {
			return resp.Data, err
		}
	} else {
		path := fmt.Sprintf("%s/%s/%s/", localbitcoinsAPIURL, localbitcoinsAPIAccountInfo, username)
		err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)
		if err != nil {
			return resp.Data, err
		}
	}

	return resp.Data, nil
}

// Getads returns information of single advertisement based on the ad ID, if
// adID ommited.
//
// adID - [optional] string if ommited returns all ads
func (l *LocalBitcoins) Getads(adID string) (AdData, error) {
	type response struct {
		Data AdData `json:"data"`
	}
	resp := response{}

	if len(adID) > 0 {
		return resp.Data,
			l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIAdGet+adID+"/", nil, &resp)
	}

	return resp.Data,
		l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIAds, nil, &resp)
}

// EditAd updates set advertisements
//
// params - see localbitcoins_types.go AdEdit for reference
// adID - string for the ad you already created
func (l *LocalBitcoins) EditAd(params AdEdit, adID string) error {
	type response struct {
		Data AdData `json:"data"`
	}

	resp := response{}
	//request := make(map[string]interface{})

	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIAdEdit+adID+"/", nil, &resp)
}

// CreateAd creates a new advertisement
//
// params - see localbitcoins_types.go AdCreate for reference
func (l *LocalBitcoins) CreateAd(params AdCreate) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIAdCreate, nil, nil)
}

// UpdatePriceEquation updates price equation of an advertisement. If there are
// problems with new equation, the price and equation are not updated and
// advertisement remains visible.
//
// equation - string of equation
// adID - string of specific ad identification
func (l *LocalBitcoins) UpdatePriceEquation(equation, adID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIUpdateEquation+adID, nil, nil)
}

// DeleteAd deletes the advertisement by adID.
//
// adID - string of specific ad identification
func (l *LocalBitcoins) DeleteAd(adID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIDeleteAd+adID, nil, nil)
}

// ReleaseFunds releases Bitcoin trades specified by ID {contact_id}. If the
// release was successful a message is returned on the data key.
func (l *LocalBitcoins) ReleaseFunds(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIRelease+contactID, nil, nil)
}

// ReleaseFundsByPin releases Bitcoin trades specified by ID {contact_id}. if
// the current pincode is provided. If the release was successful a message is
// returned on the data key.
func (l *LocalBitcoins) ReleaseFundsByPin(pin int, contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIReleaseByPin+contactID, nil, nil)
}

// MarkAsPaid marks a trade as paid.
func (l *LocalBitcoins) MarkAsPaid(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIMarkAsPaid+contactID, nil, nil)
}

// GetMessages returns all chat messages from the trade. Messages are on the message_list key.
func (l *LocalBitcoins) GetMessages(contactID string) (Message, error) {
	type response struct {
		MessageList Message `json:"message_list"`
	}
	resp := response{}

	return resp.MessageList,
		l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIMessages+contactID, nil, &resp)
}

// SendMessage posts a message and/or uploads an image to the trade. Encode
// images with multipart/form-data encoding.
func (l *LocalBitcoins) SendMessage(msg, contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPISendMessage+contactID, nil, nil)
}

// Dispute starts a dispute on the specified trade ID if the requirements for
// starting the dispute has been fulfilled.
//
// topic - [optional] String	Short description of issue to LocalBitcoins customer support.
func (l *LocalBitcoins) Dispute(topic, contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIDispute+contactID, nil, nil)
}

// CancelTrade cancels the trade if the token owner is the Bitcoin buyer.
// Bitcoin sellers cannot cancel trades.
func (l *LocalBitcoins) CancelTrade(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPICancelTrade+contactID, nil, nil)
}

// FundTrade attempts to fund an unfunded local trade from the token owners
// wallet. Works only if the token owner is the Bitcoin seller in the trade.
func (l *LocalBitcoins) FundTrade(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIFundTrade+contactID, nil, nil)
}

// ConfirmRealName creates or updates real name confirmation.
func (l *LocalBitcoins) ConfirmRealName(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIConfirmRealName+contactID, nil, nil)
}

// VerifyIdentity marks the identity of trade partner as verified. You must be
// the advertiser in this trade.
func (l *LocalBitcoins) VerifyIdentity(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIVerifyIdentity+contactID, nil, nil)
}

// InitiateTrade sttempts to start a Bitcoin trade from the specified
// advertisement ID.
func (l *LocalBitcoins) InitiateTrade(amount int, message, adID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIInitiateTrade+adID, nil, nil)
}

// GetTradeInfo returns information about a single trade that the token owner is
// part in.
func (l *LocalBitcoins) GetTradeInfo(contactID string) error {
	return l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPITradeInfo+contactID, nil, nil)
}

// GetCountryCodes returns a list of valid and recognized countrycodes
func (l *LocalBitcoins) GetCountryCodes() error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPICountryCodes, true, l.Verbose, nil)
}

// GetCurrencies returns a list of valid and recognized fiat currencies. Also
// contains human readable name for every currency and boolean that tells if
// currency is an altcoin.
func (l *LocalBitcoins) GetCurrencies() error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPICurrencies, true, l.Verbose, nil)
}

// GetDashboardInfo returns a list of trades on the data key contact_list. This
// API end point mirrors the website's dashboard, allowing access to contacts in
// different states.
// In addition all of these listings have buyer/ and seller/ sub-listings to
// view contacts where the token owner is either buying or selling, respectively.
// E.g. /api/dashboard/buyer/. All contacts where the token owner is
// participating are returned.
func (l *LocalBitcoins) GetDashboardInfo() (DashBoardInfo, error) {
	resp := DashBoardInfo{}

	return resp,
		l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIDashboard, nil, &resp)
}

// GetDashboardReleasedTrades returns a list of all released trades where the
// token owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardReleasedTrades() (DashBoardInfo, error) {
	resp := DashBoardInfo{}

	return resp,
		l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIDashboardReleased, nil, &resp)
}

// GetDashboardCancelledTrades returns a list of all canceled trades where the
// token owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardCancelledTrades() (DashBoardInfo, error) {
	resp := DashBoardInfo{}

	return resp,
		l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIDashboardCancelled, nil, &resp)
}

// GetDashboardClosedTrades returns a list of all closed trades where the token
// owner is either a buyer or seller.
func (l *LocalBitcoins) GetDashboardClosedTrades() (DashBoardInfo, error) {
	resp := DashBoardInfo{}

	return resp,
		l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIDashboardClosed, nil, &resp)
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
func (l *LocalBitcoins) SetFeedback(msg, feedback, username string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIFeedback, nil, nil)
}

// Logout expires the current access token immediately. To get a new token
// afterwards, public apps will need to re-authenticate, confidential apps can
// turn in a refresh token.
func (l *LocalBitcoins) Logout() error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPILogout, nil, nil)
}

// CreateNewInvoice creates a new invoice.
func (l *LocalBitcoins) CreateNewInvoice(currency, description, returnURL string, amount float64, internal bool) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPICreateInvoice, nil, nil)
}

// GetInvoice returns information about a specific invoice created by the token
// owner.
func (l *LocalBitcoins) GetInvoice(invoiceID string) (Invoice, error) {
	resp := Invoice{}
	return resp, l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPICreateInvoice, nil, &resp)
}

// DeleteInvoice deletes a specific invoice. Deleting invoices is possible when
// it is sure that receiver cannot accidentally pay the invoice at the same time
// as the merchant is deleting it. You can use the API request
// /api/merchant/invoice/{invoice_id}/ to check if deleting is possible.
func (l *LocalBitcoins) DeleteInvoice(invoiceID string) (Invoice, error) {
	resp := Invoice{}
	return resp, l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPICreateInvoice, nil, &resp)
}

// GetNotifications returns recent notifications.
func (l *LocalBitcoins) GetNotifications() ([]NotificationInfo, error) {
	resp := []NotificationInfo{}
	return resp, l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIGetNotification, nil, &resp)
}

// MarkNotifications marks a specific notification as read.
func (l *LocalBitcoins) MarkNotifications(notificationID string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIMarkNotification, nil, nil)
}

// GetPaymentMethods returns a list of valid payment methods. Also contains name
// and code for payment methods, and possible limitations in currencies and bank
// name choices.
func (l *LocalBitcoins) GetPaymentMethods() error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPIPaymentMethods, true, l.Verbose, nil)
}

// GetPaymentMethodsByCountry returns a list of valid payment methods filtered
// by countrycodes.
func (l *LocalBitcoins) GetPaymentMethodsByCountry(countryCode string) error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPIPaymentMethods+countryCode, true, l.Verbose, nil)
}

// CheckPincode checks the given PIN code against the token owners currently
// active PIN code. You can use this method to ensure the person using the
// session is the legitimate user.
// Due to only requiring the read scope, the user is not guaranteed to have set
// a PIN code. If you protect your application using this request, please make
// the user has set a PIN code for his account.
func (l *LocalBitcoins) CheckPincode(pin int) (bool, error) {
	type response struct {
		Data struct {
			PinOK bool `json:"pincode_ok"`
		} `json:"data"`
	}
	resp := response{}
	values := url.Values{}
	values.Set("pincode", strconv.Itoa(pin))
	err := l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIPinCode, values, &resp)

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
func (l *LocalBitcoins) GetPlaces(lat, lon int, location, countryCode string) error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPIPlaces, true, l.Verbose, nil)
}

// VerifyUsername returns list of real name verifiers for the user. Returns a
// list only when you have a trade with the user where you are the seller.
func (l *LocalBitcoins) VerifyUsername() error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIVerifyUsername, nil, nil)
}

// GetRecentMessages returns maximum of 25 newest trade messages. Does not
// return messages older than one month. Messages are ordered by sending time,
// and the newest one is first.
func (l *LocalBitcoins) GetRecentMessages(after string) error {
	return l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIVerifyUsername, nil, nil)
}

// GetWalletInfo gets information about the token owner's wallet balance.
func (l *LocalBitcoins) GetWalletInfo() (WalletInfo, error) {
	type response struct {
		Data WalletInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIWallet, nil, &resp)

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
func (l *LocalBitcoins) GetWalletBalance() (WalletBalanceInfo, error) {
	type response struct {
		Data WalletBalanceInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", localbitcoinsAPIWalletBalance, nil, &resp)

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
func (l *LocalBitcoins) WalletSend(address string, amount float64, pin int) (bool, error) {
	values := url.Values{}
	values.Set("address", address)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := localbitcoinsAPIWalletSend

	if pin > 0 {
		values.Set("pincode", strconv.Itoa(pin))
		path = localbitcoinsAPIWalletSendPin
	}

	type response struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", path, values, &resp)
	if err != nil {
		return false, err
	}

	if resp.Data.Message != "Money is being sent" {
		return false, errors.New("unable to send Bitcoins")
	}

	return true, nil
}

// GetWalletAddress returns an unused receiving address from the token owner's
// wallet. The address is returned in the address key of the response. Note that
// this API may keep returning the same (unused) address if requested repeatedly.
func (l *LocalBitcoins) GetWalletAddress() (string, error) {
	type response struct {
		Data struct {
			Message string `json:"message"`
			Address string `json:"address"`
		}
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", localbitcoinsAPIWalletAddress, nil, &resp)
	if err != nil {
		return "", err
	}

	if resp.Data.Message != "OK!" {
		return "", errors.New("unable to fetch wallet address")
	}

	return resp.Data.Address, nil
}

// GetBitcoinsWithCashAd returns buy or sell as cash local advertisements.
func (l *LocalBitcoins) GetBitcoinsWithCashAd(locationID, locationSlug string, BuySide bool) error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPICashBuy, true, l.Verbose, nil)
}

// GetBitcoinsOnlineAd this API returns buy or sell Bitcoin online ads.
func (l *LocalBitcoins) GetBitcoinsOnlineAd(countryCode, countryName, paymentMethod string, BuySide bool) error {
	return common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPIOnlineBuy, true, l.Verbose, nil)
}

// GetTicker returns list of all completed trades.
func (l *LocalBitcoins) GetTicker() (map[string]Ticker, error) {
	result := make(map[string]Ticker)

	return result,
		common.SendHTTPGetRequest(localbitcoinsAPIURL+localbitcoinsAPITicker, true, l.Verbose, &result)
}

// GetTrades returns all closed trades in online buy and online sell categories,
// updated every 15 minutes.
func (l *LocalBitcoins) GetTrades(currency string, values url.Values) ([]Trade, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/%s/trades.json", localbitcoinsAPIURL+localbitcoinsAPIBitcoincharts, currency), values)
	result := []Trade{}

	return result, common.SendHTTPGetRequest(path, true, l.Verbose, &result)
}

// GetOrderbook returns buy and sell bitcoin online advertisements. Amount is
// the maximum amount available for the trade request. Price is the hourly
// updated price. The price is based on the price equation and commission %
// entered by the ad author.
func (l *LocalBitcoins) GetOrderbook(currency string) (Orderbook, error) {
	type response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}

	path := fmt.Sprintf("%s/%s/orderbook.json", localbitcoinsAPIURL+localbitcoinsAPIBitcoincharts, currency)
	resp := response{}
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)

	if err != nil {
		return Orderbook{}, err
	}

	orderbook := Orderbook{}

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, Price{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, Price{price, amount})
	}

	return orderbook, nil
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to
// localbitcoins
func (l *LocalBitcoins) SendAuthenticatedHTTPRequest(method, path string, values url.Values, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	if l.Nonce.Get() == 0 {
		l.Nonce.Set(time.Now().UnixNano())
	} else {
		l.Nonce.Inc()
	}

	payload := ""
	path = "/api/" + path

	if len(values) > 0 {
		payload = values.Encode()
	}

	message := l.Nonce.String() + l.APIKey + path + payload
	hmac := common.GetHMAC(common.HashSHA256, []byte(message), []byte(l.APISecret))
	headers := make(map[string]string)
	headers["Apiauth-Key"] = l.APIKey
	headers["Apiauth-Nonce"] = l.Nonce.String()
	headers["Apiauth-Signature"] = common.StringToUpper(common.HexEncodeToString(hmac))
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	if l.Verbose {
		log.Printf("Raw Path: \n%s\n", path)
	}

	resp, err := common.SendHTTPRequest(method, localbitcoinsAPIURL+path, headers, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return err
	}

	if l.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	errCapture := GeneralError{}
	if err = common.JSONDecode([]byte(resp), &errCapture); err == nil {
		if len(errCapture.Error.Message) != 0 {
			return errors.New(errCapture.Error.Message)
		}
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return err
	}

	return nil
}
