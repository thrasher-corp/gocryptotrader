package gateio

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OTCBankSupplementChecklistItem represents a bank supplement check list item
type OTCBankSupplementChecklistItem struct {
	UserType string `json:"user_type"`
	Items    []struct {
		Code     string `json:"code"`
		EN       string `json:"en"`
		ZH       string `json:"zh"`
		Required bool   `json:"required"`
	} `json:"items"`
}

// OTCBankPersonalSupplementMultipartRequest represents bank personal
type OTCBankPersonalSupplementMultipartRequest struct {
	BankID          string `json:"bank_id"`
	IDDocumentFront string `json:"id_document_front"`
	IDDocumentBack  string `json:"id_document_back"`
	AddressProof    string `json:"address_proof"`
}

// OTCBankEnterpriseSupplementMultipartRequest represents an enterprise suppelement request
type OTCBankEnterpriseSupplementMultipartRequest struct {
	UID                   string `json:"uid"`
	BankID                string `json:"bank_id"`
	Certificate           string `json:"certificate"`
	ShareHolders          string `json:"share_holders"`
	Passport              string `json:"passport"`
	ShareHoldingStructure string `json:"share_holding_structure"`
	FundsStatement        string `json:"funds_statement"`
	Additional            string `json:"additional"`
}

// OTCActionResponse holds the response for OTC action operations such as cancel or mark paid.
type OTCActionResponse struct {
	Code      int64      `json:"code"`
	Message   string     `json:"message"`
	Timestamp types.Time `json:"timestamp"`
}

// Error implements the error interface for OTC action responses.
func (r *OTCActionResponse) Error() error {
	if r.Code != 0 {
		return fmt.Errorf("error code: %d message: %s", r.Code, r.Message)
	}
	return nil
}

// OTCQuoteRequest holds request parameters for creating a fiat and stablecoin quote.
type OTCQuoteRequest struct {
	Side             string        `json:"side"`
	PayCoin          currency.Code `json:"pay_coin"`
	GetCoin          currency.Code `json:"get_coin"`
	PayAmount        types.Number  `json:"pay_amount,omitempty"`
	GetAmount        types.Number  `json:"get_amount,omitempty"`
	CreateQuoteToken string        `json:"create_quote_token,omitempty"`
	PromotionCode    string        `json:"promotion_code,omitempty"`
}

// OTCQuoteData holds the quote data returned by the OTC quote API.
type OTCQuoteData struct {
	Type                   string        `json:"type"`
	PayCoin                currency.Code `json:"pay_coin"`
	GetCoin                currency.Code `json:"get_coin"`
	PayAmount              types.Number  `json:"pay_amount"`
	GetAmount              types.Number  `json:"get_amount"`
	Rate                   types.Number  `json:"rate"`
	ValidityPeriod         string        `json:"validity_period"`
	ReciprocalExchangeRate types.Number  `json:"rate_reci"`
	URL                    string        `json:"url"`
	Memo                   string        `json:"memo"`
	PromotionCode          string        `json:"promotion_code"`
	Side                   string        `json:"side"`
	HasSignature           string        `json:"has_signature"`
	ExchangeRate           types.Number  `json:"ex_rate"`
	USDCRate               types.Number  `json:"usdc_rate"`
	IsNeedFile             string        `json:"is_need_file"`
	GateBankID             string        `json:"gate_bank_id"`
	GateBankName           string        `json:"gate_bank_name"`
	OrderType              string        `json:"order_type"`
	QuoteToken             string        `json:"quote_token"`
	RefreshLimit           types.Number  `json:"refresh_limit"`
	RefreshLimitMsg        string        `json:"refresh_limit_msg"`
}

// OTCFiatOrderRequest holds request parameters for creating a fiat order.
type OTCFiatOrderRequest struct {
	Type           string        `json:"type"`
	Side           string        `json:"side"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	CryptoCurrency currency.Code `json:"crypto_currency"`
	CryptoAmount   types.Number  `json:"crypto_amount"`
	FiatAmount     types.Number  `json:"fiat_amount"`
	PromotionCode  string        `json:"promotion_code,omitempty"`
	QuoteToken     string        `json:"quote_token"`
	BankID         string        `json:"bank_id"`
}

// OTCStablecoinOrderRequest holds request parameters for creating a stablecoin order.
type OTCStablecoinOrderRequest struct {
	PayCoin       currency.Code `json:"pay_coin"`
	GetCoin       currency.Code `json:"get_coin"`
	PayAmount     types.Number  `json:"pay_amount,omitempty"`
	GetAmount     types.Number  `json:"get_amount,omitempty"`
	Side          string        `json:"side,omitempty"`
	PromotionCode string        `json:"promotion_code,omitempty"`
	QuoteToken    string        `json:"quote_token,omitempty"`
}

// OTCBankCard holds a single bank card entry.
type OTCBankCard struct {
	ID                    string     `json:"id"`
	BankAccountName       string     `json:"bank_account_name"`
	BankName              string     `json:"bank_name"`
	BankCountry           string     `json:"bank_country"`
	BankAddress           string     `json:"bank_address"`
	BranchCode            string     `json:"branch_code"`
	BankCode              string     `json:"bank_code"`
	IBAN                  string     `json:"iban"`
	Swift                 string     `json:"swift"`
	RemittanceLineNumber  string     `json:"remittance_line_number"`
	AgentBankName         string     `json:"agent_bank_name"`
	AgentBankSwift        string     `json:"agent_bank_swift"`
	SubmitTime            types.Time `json:"submit_time"`
	UpdateTime            types.Time `json:"update_time"`
	Status                string     `json:"status"`
	DocumentationFileType string     `json:"documentation_file_type"`
	Memo                  string     `json:"memo"`
	IsDefault             string     `json:"is_default"`
	DocumentationFileKey  string     `json:"documentation_file_key_url"`
}

// OTCBankCardRequestResponse represents a bank-card request response
type OTCBankCardRequestResponse struct {
	BankID uint64 `json:"bank_id"`
	Status uint64 `json:"status"`
}

// OTCBankCardListData holds the bank card list response data.
type OTCBankCardListData struct {
	Lists []*OTCBankCard `json:"lists"`
}

// OTCBankCreateMultipartRequest represents an OTC create multipart request parameter
type OTCBankCreateMultipartRequest struct {
	BankAccountName      string `json:"bank_account_name"`
	BankName             string `json:"bank_name"`
	BankCountry          string `json:"bank_country"`
	BankAddress          string `json:"bank_address"`
	IBAN                 string `json:"iban"`
	Swift                string `json:"swift"`
	RemittanceLineNumber string `json:"remittance_line_number"`
	AgentBankName        string `json:"agent_bank_name"`
	AgentBankSwift       string `json:"agent_bank_swift"`
	DocumentationFile    string `json:"documentation_file"`
}

// OTCOrderListItem holds a single fiat order list item.
type OTCOrderListItem struct {
	CreateAt            types.Time      `json:"create_at"`
	TradeNumber         string          `json:"trade_no"`
	DBStatus            string          `json:"db_status"`
	Type                string          `json:"type"`
	CryptoCurrency      currency.Code   `json:"crypto_currency"`
	CreateAt2           types.Time      `json:"create_at2"`
	BankAccountIBM      string          `json:"bank_account_ibm"`
	Rate                types.Number    `json:"rate"`
	CreateBankIBAM      string          `json:"create_bank_ibam"`
	PromotionCode       string          `json:"promotion_code"`
	Time                types.Time      `json:"time"`
	Timestamp           types.Time      `json:"timestamp"`
	OrderID             string          `json:"order_id"`
	Status              string          `json:"status"`
	FiatCurrency        currency.Code   `json:"fiat_currency"`
	FiatCurrencyInfo    *NameAndIconURL `json:"fiat_currency_info"`
	FiatAmount          types.Number    `json:"fiat_amount"`
	CryptoCurrencyInfo  *NameAndIconURL `json:"crypto_currency_info"`
	CryptoAmount        types.Number    `json:"crypto_amount"`
	GateBankAccountIBAN string          `json:"gate_bank_account_iban"`
}

// NameAndIconURL represents a name and asset icon URL.
type NameAndIconURL struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// OTCOrderListData holds the fiat order list response data.
type OTCOrderListData struct {
	PageNumber      int64               `json:"pn"`
	PageSize        int64               `json:"ps"`
	TotalPageNumber int64               `json:"total_pn"`
	Count           int64               `json:"count"`
	List            []*OTCOrderListItem `json:"list"`
}

// OTCStableCoinOrder holds a single stablecoin order item.
type OTCStableCoinOrder struct {
	ID           uint64        `json:"id"`
	TradeNo      string        `json:"trade_no"`
	PayCoin      currency.Code `json:"pay_coin"`
	PayAmount    types.Number  `json:"pay_amount"`
	GetCoin      currency.Code `json:"get_coin"`
	GetAmount    types.Number  `json:"get_amount"`
	Rate         types.Number  `json:"rate"`
	Status       string        `json:"status"`
	CreateTime   types.Time    `json:"create_time"`
	CreateTimest int64         `json:"create_timest"`
}

// OTCStablecoinOrderListData holds the stablecoin order list response data.
type OTCStablecoinOrderListData struct {
	Total      int64                 `json:"total"`
	PageSize   int64                 `json:"page_size"`
	PageNumber int64                 `json:"page_number"`
	TotalPage  int64                 `json:"total_page"`
	List       []*OTCStableCoinOrder `json:"list"`
}

// OTCOrderDetailData holds the fiat order detail data.
type OTCOrderDetailData struct {
	ID             string        `json:"id"`
	UID            string        `json:"uid"`
	Type           string        `json:"type"`
	CryptoCurrency currency.Code `json:"crypto_currency"`
	CryptoAmount   types.Number  `json:"crypto_amount"`
	CreateTime     types.Time    `json:"create_time"`
	Rate           types.Number  `json:"rate"`
	PromotionCode  string        `json:"promotion_code"`
	TradeNo        string        `json:"trade_no"`
	Status         string        `json:"status"`
	TransferRemark string        `json:"transfer_remark"`
	OrderID        string        `json:"order_id"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	FiatAmount     types.Number  `json:"fiat_amount"`
	ReferenceCode  string        `json:"reference_code"`
	DBStatus       string        `json:"db_status"`
	Memo           string        `json:"memo"`
	Side           string        `json:"side"`
}
