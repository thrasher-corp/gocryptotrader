package bitfinex

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Bitfinex's v2 API has no equivalent of v1's wire withdrawal, as its withdrawal takes no bank details, so a wire
// withdrawal goes through v1, which signs requests differently: the body, with the path and a nonce, is sent base64
// encoded in a header and signed

// legacyAPIVersionPath prefixes the v1 endpoint paths
const legacyAPIVersionPath = "/v1/"

var (
	errFiatWithdrawalFailed            = errors.New("fiat withdrawal failed")
	errIntermediaryBankAccountRequired = errors.New("intermediary bank account required")
	errIntermediaryBankAccountInexact  = errors.New("intermediary bank account number cannot be held exactly")
)

// FiatWithdrawalRequest is a wire withdrawal RequestFiatWithdrawal makes
type FiatWithdrawalRequest struct {
	// Wallet is WalletTypeExchange, WalletTypeMargin or WalletTypeFunding
	Wallet string
	// Withdrawal holds the amount, currency and bank details, with any intermediary bank's
	Withdrawal *withdraw.Request
}

// fiatWithdrawalBody is the body of a v1 wire withdrawal. It sends no account_address, the account holder's, as GCT's
// bank accounts hold none
type fiatWithdrawalBody struct {
	Request        string `json:"request"`
	Nonce          string `json:"nonce"`
	WithdrawType   string `json:"withdraw_type"`
	WalletSelected string `json:"walletselected"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	AccountName    string `json:"account_name"`
	AccountNumber  string `json:"account_number"`
	BankName       string `json:"bank_name"`
	BankAddress    string `json:"bank_address"`
	BankCity       string `json:"bank_city"`
	BankCountry    string `json:"bank_country"`
	SWIFT          string `json:"swift"`
	DetailPayment  string `json:"detail_payment,omitempty"`
	// ExpressWire is 1 for an express wire, as the docs ask, and omitted for a normal one
	ExpressWire             uint8  `json:"expressWire,omitempty"`
	IntermediaryBankName    string `json:"intermediary_bank_name,omitempty"`
	IntermediaryBankAddress string `json:"intermediary_bank_address,omitempty"`
	IntermediaryBankCity    string `json:"intermediary_bank_city,omitempty"`
	IntermediaryBankCountry string `json:"intermediary_bank_country,omitempty"`
	IntermediaryBankAccount string `json:"intermediary_bank_account,omitempty"`
	IntermediaryBankSWIFT   string `json:"intermediary_bank_swift,omitempty"`
}

// FiatWithdrawalResponse is the outcome of a wire withdrawal
type FiatWithdrawalResponse struct {
	// Status is success or error
	Status       string `json:"status"`
	Message      string `json:"message"`
	WithdrawalID uint64 `json:"withdrawal_id"`
}

// RequestFiatWithdrawal withdraws fiat by wire from a wallet to a bank account through Bitfinex's v1 API, as v2 has no
// equivalent
func (e *Exchange) RequestFiatWithdrawal(ctx context.Context, req *FiatWithdrawalRequest) (*FiatWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.NilGuard(req.Withdrawal); err != nil {
		return nil, err
	}
	if err := checkAccountWallet(req.Wallet); err != nil {
		return nil, err
	}
	w := req.Withdrawal
	if w.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, w.Amount)
	}
	body := &fiatWithdrawalBody{
		WithdrawType:   "wire",
		WalletSelected: req.Wallet,
		Amount:         strconv.FormatFloat(w.Amount, 'f', -1, 64),
		Currency:       w.Currency.Upper().String(),
		AccountName:    w.Fiat.Bank.AccountName,
		AccountNumber:  w.Fiat.Bank.AccountNumber,
		BankName:       w.Fiat.Bank.BankName,
		BankAddress:    w.Fiat.Bank.BankAddress,
		BankCity:       w.Fiat.Bank.BankPostalCity,
		BankCountry:    w.Fiat.Bank.BankCountry,
		SWIFT:          w.Fiat.Bank.SWIFTCode,
		DetailPayment:  w.Description,
	}
	if w.Fiat.IsExpressWire {
		body.ExpressWire = 1
	}
	if w.Fiat.RequiresIntermediaryBank {
		account, err := intermediaryBankAccount(&w.Fiat)
		if err != nil {
			return nil, err
		}
		body.IntermediaryBankName = w.Fiat.IntermediaryBankName
		body.IntermediaryBankAddress = w.Fiat.IntermediaryBankAddress
		body.IntermediaryBankCity = w.Fiat.IntermediaryBankCity
		body.IntermediaryBankCountry = w.Fiat.IntermediaryBankCountry
		body.IntermediaryBankAccount = account
		body.IntermediaryBankSWIFT = w.Fiat.IntermediarySwiftCode
	}
	var resp []FiatWithdrawalResponse
	if err := e.sendLegacyAuthenticatedRequest(ctx, fiatWithdrawalEPL, "withdraw", body, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}
	if resp[0].Status != "success" {
		return nil, fmt.Errorf("%w: %s: %s", errFiatWithdrawalFailed, resp[0].Status, resp[0].Message)
	}
	return &resp[0], nil
}

// intermediaryBankAccount returns the intermediary bank account a wire goes through: its IBAN, or else its account
// number, which GCT holds as a float64 and so is refused once it cannot be held exactly, rather than sent rounded
func intermediaryBankAccount(f *withdraw.FiatRequest) (string, error) {
	if f.IntermediaryIBAN != "" {
		return f.IntermediaryIBAN, nil
	}
	n := f.IntermediaryBankAccountNumber
	switch {
	case n == 0:
		return "", errIntermediaryBankAccountRequired
	case n < 0 || n != math.Trunc(n) || n > 1<<53:
		return "", fmt.Errorf("%w: %v", errIntermediaryBankAccountInexact, n)
	}
	return strconv.FormatFloat(n, 'f', 0, 64), nil
}

// sendLegacyAuthenticatedRequest sends a v1 request, signing its body with the path and a nonce. The nonce comes from
// the counter v2 requests use, as each key refuses a nonce smaller than its last whichever version it signs
func (e *Exchange) sendLegacyAuthenticatedRequest(ctx context.Context, epl request.EndpointLimit, path string, body *fiatWithdrawalBody, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	var response json.RawMessage
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		body.Request = legacyAPIVersionPath + path
		body.Nonce = e.Requester.GetNonce(nonce.UnixNano).String()
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(payload)
		signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(encoded), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method: http.MethodPost,
			Path:   endpoint + legacyAPIVersionPath + path,
			Headers: map[string]string{
				"X-BFX-APIKEY":    creds.Key,
				"X-BFX-PAYLOAD":   encoded,
				"X-BFX-SIGNATURE": hex.EncodeToString(signature),
			},
			Result:                 &response,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	return decodeResponse(response, err, result)
}
