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
	okGroupGetDepositAddress    = "deposit/address"
	okGroupGetDepositHistory    = "deposit/history"
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

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
	requestURL := fmt.Sprintf("%v?%v", okGroupGetBillDetails, urlValues.Encode())
	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// GetDepositAddressForCurrency retrieves the deposit addresses of different tokens, including previously used addresses.
func (o *OKGroup) GetDepositAddressForCurrency(currency string) ([]GetDepositAddressRespoonse, error) {
	var resp []GetDepositAddressRespoonse
	urlValues := url.Values{}
	urlValues.Set("currency", currency)
	requestURL := fmt.Sprintf("%v?%v", okGroupGetDepositAddress, urlValues.Encode())
	return resp, o.SendAuthenticatedHTTPRequest("GET", okGroupAccountSubsection, requestURL, nil, &resp)
}

// GetDepositHistory retrieves the deposit history of all tokens.100 recent records will be returned at maximum
func (o *OKGroup) GetDepositHistory(currency string) ([]GetDepositHistoryResponse, error) {
	var resp []GetDepositHistoryResponse
	var requestURL string
	if len(currency) > 0 {
		requestURL = fmt.Sprintf("%v/%v", okGroupGetDepositHistory, currency)
	} else {
		requestURL = okGroupGetWithdrawalHistory
	}
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
		"34001": errors.New("withdrawal suspended"),
		"34002": errors.New("please add a withdrawal address"),
		"34003": errors.New("incorrect address"),
		"34004": errors.New("withdrawal fee is smaller than minimum limit"),
		"34005": errors.New("withdrawal fee exceeds the maximum limit"),
		"34006": errors.New("withdrawal amount is lower than the minimum limit"),
		"34007": errors.New("withdrawal amount exceeds the maximum limit"),
		"34008": errors.New("insufficient balance"),
		"34009": errors.New("your withdrawal amount exceeds the daily limit"),
		"34010": errors.New("transfer amount must be larger than 0"),
		"34011": errors.New("conditions not met, e.g. KYC level"),
		"34012": errors.New("special requirements"),
		"34013": errors.New("Token margin trading instrument ID required"),
		"34014": errors.New("Transfer limited"),
		"34015": errors.New("subaccount does not exist"),
		"34016": errors.New("either end of the account does not authorize the transfer"),
		"34017": errors.New("either end of the account does not authorize the transfer"),
		"34018": errors.New("incorrect trades password"),
		"34019": errors.New("please bind your email before withdrawal"),
		"34020": errors.New("please bind your funds password before withdrawal"),
		"34021": errors.New("Not verified address"),
		"34022": errors.New("Withdrawals are not available for sub accounts"),
	}
}
