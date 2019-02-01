package okgroup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	// just your average return type from okex
	returnTypeOne = "map[string]interface {}"

	okexAuthRate   = 0
	okexUnauthRate = 0

	// OkGroupAPIPath const to help with api url formatting
	OkGroupAPIPath = "api/"

	// API subsections
	okGroupAccountSubsection = "account"
	// Account based endpoints
	okGroupGetCurrencies        = "currencies"
	okGroupGetWalletInformation = "wallet"
	okGroupFundsTransfer        = "transfer"
	okGroupWithdraw             = "withdrawal"
	okGroupGetWithdrawalFees    = "withdrawal/fee"
	okGroupGetWithdrawalHistory = "withdrawal/history"
	okGroupGetBillDetails       = "ledger"
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

// Okcoin is okgroup implementation for OKEX
var Okcoin = OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okcoin.com/", OkGroupAPIPath),
	WebsocketURL: "",
	APIVersion:   "/v3/",
	ExchangeName: "OKCoin",
}

// OKGroup is the overaching type across the all of OKEx's exchange methods
type OKGroup struct {
	exchange.Base
	ExchangeName  string
	WebsocketConn *websocket.Conn
	mu            sync.Mutex

	// Spot and contract market error codes as per https://www.okex.com/rest_request.html
	ErrorCodes map[string]error

	// Stores for corresponding variable checks
	ContractTypes    []string
	CurrencyPairs    []string
	ContractPosition []string
	Types            []string

	// URLs to be overridden by implementations of OKGroup
	APIURL       string
	APIVersion   string
	WebsocketURL string
}

// SetDefaults method assignes the default values for Bittrex
func (o *OKGroup) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = o.ExchangeName
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okexAuthRate),
		request.NewRateLimit(time.Second, okexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = o.APIURL
	o.APIUrl = o.APIUrlDefault
	o.AssetTypes = []string{ticker.Spot}
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup method sets current configuration details if enabled
func (o *OKGroup) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Name = exch.Name
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetEnabled(exch.Websocket)
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = o.WebsocketSetup(o.WsConnect,
			exch.Name,
			exch.Websocket,
			okexDefaultWebsocketURL,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// -------------------------------------------------------------------------------------
// Public endpoints
// -------------------------------------------------------------------------------------

// GetSpotInstruments returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetSpotInstruments() ([]SpotInstrument, error) {
	var resp []SpotInstrument
	path := fmt.Sprintf("%vspot%v%v", o.APIUrl, o.APIVersion, "instruments")
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (o *OKGroup) SendHTTPRequest(path string, result interface{}) error {
	return o.SendPayload("GET", path, nil, nil, result, false, o.Verbose)
}

// -------------------------------------------------------------------------------------
// Private endpoints
// -------------------------------------------------------------------------------------

// GetCurrencies returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetCurrencies() ([]CurrencyResponse, error) {
	var resp []CurrencyResponse
	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, okGroupGetCurrencies, nil, &resp)
}

// GetWalletInformation returns a list of wallets and their properties
func (o *OKGroup) GetWalletInformation(currency string) ([]WalletInformationResponse, error) {
	var resp []WalletInformationResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetWalletInformation, currency)
	} else {
		requestURL = okGroupGetWalletInformation
	}

	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// TransferFunds  the transfer of funds between wallet, trading accounts, main account and sub accounts.
func (o *OKGroup) TransferFunds(request FundTransferRequest) (FundTransferResponse, error) {
	var resp FundTransferResponse
	return resp, o.SendAuthenticatedHTTPRequest("POST", okGroupAccountSubsection, okGroupFundsTransfer, request, &resp)
}

// Withdraw withdrawal of tokens to OKCoin International, other OKEx accounts or other addresses.
func (o *OKGroup) Withdraw(request WithdrawRequest) (WithdrawResponse, error) {
	var resp WithdrawResponse
	return resp, o.SendAuthenticatedHTTPRequest("POST", okGroupAccountSubsection, okGroupWithdraw, request, &resp)
}

// GetWithdrawalFee retrieves the information about the recommended network transaction fee for withdrawals to digital asset addresses. The higher the fees are, the sooner the confirmations you will get.
func (o *OKGroup) GetWithdrawalFee(currency string) ([]WithdrawalFeeResponse, error) {
	var resp []WithdrawalFeeResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v?currency=%v", okGroupGetWithdrawalFees, currency)
	} else {
		requestURL = okGroupGetWalletInformation
	}

	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// GetWithdrawalHistory retrieves all recent withdrawal records.
func (o *OKGroup) GetWithdrawalHistory(currency string) ([]WithdrawalHistoryResponse, error) {
	var resp []WithdrawalHistoryResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetWithdrawalHistory, currency)
	} else {
		requestURL = okGroupGetWithdrawalHistory
	}
	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// GetBillDetails retrieves the bill details of the wallet. All the information will be paged and sorted in reverse chronological order,
// which means the latest will be at the top. Please refer to the pagination section for additional records after the first page.
// 3 months recent records will be returned at maximum
func (o *OKGroup) GetBillDetails(request GetBillDetailsRequest) ([]GetBillDetailsResponse, error) {
	var resp []GetBillDetailsResponse
	urlValues := url.Values{}
	if request.Type > 0 {
		urlValues.Set("type", strconv.FormatInt(request.Type, 10))
	}
	if len(request.Currency) > 0 {
		urlValues.Set("currency", request.Currency)
	}
	if request.From > 0 {
		urlValues.Set("from", strconv.FormatInt(request.From, 10))
	}
	if request.To > 0 {
		urlValues.Set("to", strconv.FormatInt(request.To, 10))
	}
	if request.Limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(request.Limit, 10))
	}
	requestURL := fmt.Sprintf("%v%v", okGroupGetBillDetails, urlValues.Encode())
	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// GetErrorCode returns an error code
func (o *OKGroup) GetErrorCode(code interface{}) error {
	var assertedCode string

	switch reflect.TypeOf(code).String() {
	case "float64":
		assertedCode = strconv.FormatFloat(code.(float64), 'f', -1, 64)
	case "string":
		assertedCode = code.(string)
	default:
		return errors.New("unusual type returned")
	}

	if i, ok := o.ErrorCodes[assertedCode]; ok {
		return i
	}
	return errors.New("unable to find SPOT error code")
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (o *OKGroup) SendAuthenticatedHTTPRequest(httpMethod, requestType, requestPath string, data interface{}, result interface{}) (err error) {
	if !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	utcTime := time.Now().UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:23]) + "Z"

	payload := []byte("")

	if data != nil {
		payload, err = common.JSONEncode(data)
		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if o.Verbose {
			log.Debugf("Request JSON: %s\n", payload)
		}
	}

	path := o.APIUrl + requestType + o.APIVersion + requestPath
	signPath := fmt.Sprintf("/%v%v%v%v", OkGroupAPIPath, requestType, o.APIVersion, requestPath)
	hmac := common.GetHMAC(common.HashSHA256, []byte(iso+httpMethod+signPath+string(payload)), []byte(o.APISecret))
	base64 := common.Base64Encode(hmac)

	if o.Verbose {
		log.Debugf("Sending %v request to %s with params \n", requestType, path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["OK-ACCESS-KEY"] = o.APIKey
	headers["OK-ACCESS-SIGN"] = base64
	headers["OK-ACCESS-TIMESTAMP"] = iso
	headers["OK-ACCESS-PASSPHRASE"] = o.ClientID

	var intermediary json.RawMessage

	errCap := struct {
		Result bool  `json:"result"`
		Error  int64 `json:"error_code"`
	}{}

	err = o.SendPayload(strings.ToUpper(httpMethod), path, headers, bytes.NewBuffer(payload), &intermediary, true, o.Verbose)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCap)
	if err == nil {
		if !errCap.Result {
			return fmt.Errorf("SendAuthenticatedHTTPRequest error - %s",
				o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
	}

	return common.JSONDecode(intermediary, result)
}

// SetCheckVarDefaults sets main variables that will be used in requests because
// api does not return an error if there are misspellings in strings. So better
// to check on this, this end.
func (o *OKGroup) SetCheckVarDefaults() {
	o.ContractTypes = []string{"this_week", "next_week", "quarter"}
	o.CurrencyPairs = []string{"btc_usd", "ltc_usd", "eth_usd", "etc_usd", "bch_usd"}
	o.Types = []string{"1min", "3min", "5min", "15min", "30min", "1day", "3day",
		"1week", "1hour", "2hour", "4hour", "6hour", "12hour"}
	o.ContractPosition = []string{"1", "2", "3", "4"}
}

// CheckContractPosition checks to see if the string is a valid position for OKGroup
func (o *OKGroup) CheckContractPosition(position string) error {
	if !common.StringDataCompare(o.ContractPosition, position) {
		return errors.New("invalid position string - e.g. 1 = open long position, 2 = open short position, 3 = liquidate long position, 4 = liquidate short position")
	}
	return nil
}

// CheckSymbol checks to see if the string is a valid symbol for OKGroup
func (o *OKGroup) CheckSymbol(symbol string) error {
	if !common.StringDataCompare(o.CurrencyPairs, symbol) {
		return errors.New("invalid symbol string")
	}
	return nil
}

// CheckContractType checks to see if the string is a correct asset
func (o *OKGroup) CheckContractType(contractType string) error {
	if !common.StringDataCompare(o.ContractTypes, contractType) {
		return errors.New("invalid contract type string")
	}
	return nil
}

// CheckType checks to see if the string is a correct type
func (o *OKGroup) CheckType(typeInput string) error {
	if !common.StringDataCompare(o.Types, typeInput) {
		return errors.New("invalid type string")
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.001
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

// SetErrorDefaults sets the full error default list
func (o *OKGroup) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		//Spot Errors
		"10000": errors.New("Required field, can not be null"),
		"10001": errors.New("Request frequency too high to exceed the limit allowed"),
		"10002": errors.New("System error"),
		"10004": errors.New("Request failed - Your API key might need to be recreated"),
		"10005": errors.New("'SecretKey' does not exist"),
		"10006": errors.New("'Api_key' does not exist"),
		"10007": errors.New("Signature does not match"),
		"10008": errors.New("Illegal parameter"),
		"10009": errors.New("Order does not exist"),
		"10010": errors.New("Insufficient funds"),
		"10011": errors.New("Amount too low"),
		"10012": errors.New("Only btc_usd ltc_usd supported"),
		"10013": errors.New("Only support https request"),
		"10014": errors.New("Order price must be between 0 and 1,000,000"),
		"10015": errors.New("Order price differs from current market price too much"),
		"10016": errors.New("Insufficient coins balance"),
		"10017": errors.New("API authorization error"),
		"10018": errors.New("borrow amount less than lower limit [usd:100,btc:0.1,ltc:1]"),
		"10019": errors.New("loan agreement not checked"),
		"10020": errors.New("rate cannot exceed 1%"),
		"10021": errors.New("rate cannot less than 0.01%"),
		"10023": errors.New("fail to get latest ticker"),
		"10024": errors.New("balance not sufficient"),
		"10025": errors.New("quota is full, cannot borrow temporarily"),
		"10026": errors.New("Loan (including reserved loan) and margin cannot be withdrawn"),
		"10027": errors.New("Cannot withdraw within 24 hrs of authentication information modification"),
		"10028": errors.New("Withdrawal amount exceeds daily limit"),
		"10029": errors.New("Account has unpaid loan, please cancel/pay off the loan before withdraw"),
		"10031": errors.New("Deposits can only be withdrawn after 6 confirmations"),
		"10032": errors.New("Please enabled phone/google authenticator"),
		"10033": errors.New("Fee higher than maximum network transaction fee"),
		"10034": errors.New("Fee lower than minimum network transaction fee"),
		"10035": errors.New("Insufficient BTC/LTC"),
		"10036": errors.New("Withdrawal amount too low"),
		"10037": errors.New("Trade password not set"),
		"10040": errors.New("Withdrawal cancellation fails"),
		"10041": errors.New("Withdrawal address not exsit or approved"),
		"10042": errors.New("Admin password error"),
		"10043": errors.New("Account equity error, withdrawal failure"),
		"10044": errors.New("fail to cancel borrowing order"),
		"10047": errors.New("this function is disabled for sub-account"),
		"10048": errors.New("withdrawal information does not exist"),
		"10049": errors.New("User can not have more than 50 unfilled small orders (amount<0.15BTC)"),
		"10050": errors.New("can't cancel more than once"),
		"10051": errors.New("order completed transaction"),
		"10052": errors.New("not allowed to withdraw"),
		"10064": errors.New("after a USD deposit, that portion of assets will not be withdrawable for the next 48 hours"),
		"10100": errors.New("User account frozen"),
		"10101": errors.New("order type is wrong"),
		"10102": errors.New("incorrect ID"),
		"10103": errors.New("the private otc order's key incorrect"),
		"10216": errors.New("Non-available API"),
		"1002":  errors.New("The transaction amount exceed the balance"),
		"1003":  errors.New("The transaction amount is less than the minimum requirement"),
		"1004":  errors.New("The transaction amount is less than 0"),
		"1007":  errors.New("No trading market information"),
		"1008":  errors.New("No latest market information"),
		"1009":  errors.New("No order"),
		"1010":  errors.New("Different user of the cancelled order and the original order"),
		"1011":  errors.New("No documented user"),
		"1013":  errors.New("No order type"),
		"1014":  errors.New("No login"),
		"1015":  errors.New("No market depth information"),
		"1017":  errors.New("Date error"),
		"1018":  errors.New("Order failed"),
		"1019":  errors.New("Undo order failed"),
		"1024":  errors.New("Currency does not exist"),
		"1025":  errors.New("No chart type"),
		"1026":  errors.New("No base currency quantity"),
		"1027":  errors.New("Incorrect parameter may exceeded limits"),
		"1028":  errors.New("Reserved decimal failed"),
		"1029":  errors.New("Preparing"),
		"1030":  errors.New("Account has margin and futures, transactions can not be processed"),
		"1031":  errors.New("Insufficient Transferring Balance"),
		"1032":  errors.New("Transferring Not Allowed"),
		"1035":  errors.New("Password incorrect"),
		"1036":  errors.New("Google Verification code Invalid"),
		"1037":  errors.New("Google Verification code incorrect"),
		"1038":  errors.New("Google Verification replicated"),
		"1039":  errors.New("Message Verification Input exceed the limit"),
		"1040":  errors.New("Message Verification invalid"),
		"1041":  errors.New("Message Verification incorrect"),
		"1042":  errors.New("Wrong Google Verification Input exceed the limit"),
		"1043":  errors.New("Login password cannot be same as the trading password"),
		"1044":  errors.New("Old password incorrect"),
		"1045":  errors.New("2nd Verification Needed"),
		"1046":  errors.New("Please input old password"),
		"1048":  errors.New("Account Blocked"),
		"1201":  errors.New("Account Deleted at 00: 00"),
		"1202":  errors.New("Account Not Exist"),
		"1203":  errors.New("Insufficient Balance"),
		"1204":  errors.New("Invalid currency"),
		"1205":  errors.New("Invalid Account"),
		"1206":  errors.New("Cash Withdrawal Blocked"),
		"1207":  errors.New("Transfer Not Support"),
		"1208":  errors.New("No designated account"),
		"1209":  errors.New("Invalid api"),
		"1216":  errors.New("Market order temporarily suspended. Please send limit order"),
		"1217":  errors.New("Order was sent at ±5% of the current market price. Please resend"),
		"1218":  errors.New("Place order failed. Please try again later"),
		// Errors for both
		"HTTP ERROR CODE 403": errors.New("Too many requests, IP is shielded"),
		"Request Timed Out":   errors.New("Too many requests, IP is shielded"),
		// contract errors
		"405":   errors.New("method not allowed"),
		"20001": errors.New("User does not exist"),
		"20002": errors.New("Account frozen"),
		"20003": errors.New("Account frozen due to liquidation"),
		"20004": errors.New("Contract account frozen"),
		"20005": errors.New("User contract account does not exist"),
		"20006": errors.New("Required field missing"),
		"20007": errors.New("Illegal parameter"),
		"20008": errors.New("Contract account balance is too low"),
		"20009": errors.New("Contract status error"),
		"20010": errors.New("Risk rate ratio does not exist"),
		"20011": errors.New("Risk rate lower than 90%/80% before opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% before opening LTC position with 10x/20x leverage"),
		"20012": errors.New("Risk rate lower than 90%/80% after opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% after opening LTC position with 10x/20x leverage"),
		"20013": errors.New("Temporally no counter party price"),
		"20014": errors.New("System error"),
		"20015": errors.New("Order does not exist"),
		"20016": errors.New("Close amount bigger than your open positions"),
		"20017": errors.New("Not authorized/illegal operation"),
		"20018": errors.New("Order price cannot be more than 103% or less than 97% of the previous minute price"),
		"20019": errors.New("IP restricted from accessing the resource"),
		"20020": errors.New("secretKey does not exist"),
		"20021": errors.New("Index information does not exist"),
		"20022": errors.New("Wrong API interface (Cross margin mode shall call cross margin API, fixed margin mode shall call fixed margin API)"),
		"20023": errors.New("Account in fixed-margin mode"),
		"20024": errors.New("Signature does not match"),
		"20025": errors.New("Leverage rate error"),
		"20026": errors.New("API Permission Error"),
		"20027": errors.New("no transaction record"),
		"20028": errors.New("no such contract"),
		"20029": errors.New("Amount is large than available funds"),
		"20030": errors.New("Account still has debts"),
		"20038": errors.New("Due to regulation, this function is not available in the country/region your currently reside in"),
		"20049": errors.New("Request frequency too high"),
	}
}
