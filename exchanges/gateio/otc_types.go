package gateio

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OTCActionResponse holds the response for OTC action operations such as cancel or mark paid.
type OTCActionResponse struct {
	Code      int64      `json:"code"`
	Message   string     `json:"message"`
	Timestamp types.Time `json:"timestamp"`
}

// AsError returns the response status as an error, or nil when the request succeeded
func (r *OTCActionResponse) AsError() error {
	if r.Code != 0 {
		return fmt.Errorf("error code: %d message: %s", r.Code, r.Message)
	}
	return nil
}

// OTCQuoteRequest holds request parameters for creating a fiat and stablecoin quote.
type OTCQuoteRequest struct {
	Side             string        `json:"side"` // PAY quotes PayAmount, GET quotes GetAmount
	PayCoin          currency.Code `json:"pay_coin"`
	GetCoin          currency.Code `json:"get_coin"`
	PayAmount        types.Number  `json:"pay_amount,omitempty"`
	GetAmount        types.Number  `json:"get_amount,omitempty"`
	CreateQuoteToken string        `json:"create_quote_token,omitempty"` // "1" returns a quote token for order placement, "0" previews only
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
	ReciprocalExchangeRate types.Number  `json:"rate_reci"`
	PromotionCode          string        `json:"promotion_code"`
	Side                   string        `json:"side"`
	OrderType              string        `json:"order_type"`
	QuoteToken             string        `json:"quote_token"`
	ValidityPeriodSeconds  types.Number  `json:"validity_period"`
	RefreshLimit           uint64        `json:"refresh_limit"`
	RefreshLimitMessage    string        `json:"refresh_limit_msg"`
}

// OTCFiatOrderRequest holds request parameters for creating a fiat order.
type OTCFiatOrderRequest struct {
	Type           string        `json:"type"`
	Side           string        `json:"side"` // The side returned by the quote
	FiatCurrency   currency.Code `json:"fiat_currency"`
	CryptoCurrency currency.Code `json:"crypto_currency"`
	CryptoAmount   types.Number  `json:"crypto_amount"`
	FiatAmount     types.Number  `json:"fiat_amount"`
	PromotionCode  string        `json:"promotion_code,omitempty"`
	QuoteToken     string        `json:"quote_token"`
	BankID         string        `json:"bank_id"`
	ReceiveType    string        `json:"receive_type,omitempty"` // YOU, GATE or RECIPIENT for corporate users; GATE or PERSON for individuals
}

// OTCStablecoinOrderRequest holds request parameters for creating a stablecoin order.
type OTCStablecoinOrderRequest struct {
	PayCoin       currency.Code `json:"pay_coin"`
	GetCoin       currency.Code `json:"get_coin"`
	PayAmount     types.Number  `json:"pay_amount"`
	GetAmount     types.Number  `json:"get_amount"`
	Side          string        `json:"side"` // The side returned by the quote
	PromotionCode string        `json:"promotion_code,omitempty"`
	QuoteToken    string        `json:"quote_token"`
}

// OTCBankCard holds a single bank card entry.
type OTCBankCard struct {
	ID                   string `json:"id"`
	BankAccountName      string `json:"bank_account_name"`
	BankName             string `json:"bank_name"`
	BankCountry          string `json:"bank_country"`
	BankAddress          string `json:"bank_address"`
	IBAN                 string `json:"iban"`
	Swift                string `json:"swift"`
	RemittanceLineNumber string `json:"remittance_line_number"`
	AgentBankName        string `json:"agent_bank_name"`
	AgentBankSwift       string `json:"agent_bank_swift"`
	// SubmitTime and UpdateTime are kept as sent, since Gate documents them as strings without a format
	SubmitTime string        `json:"submit_time"`
	UpdateTime string        `json:"update_time"`
	Status     string        `json:"status"`
	IsDefault  types.Boolean `json:"is_default"`
}

// OTCBankCardListData holds the bank card list response data.
type OTCBankCardListData struct {
	Lists []*OTCBankCard `json:"lists"`
}

// OTCBankCreateMultipartRequest represents an OTC create bank card request parameter. Account-opening proof is
// supplied either as DocumentationFile or as a pre-uploaded DocumentationFileKey with its FileType.
type OTCBankCreateMultipartRequest struct {
	BankAccountName      string `json:"bank_account_name"`
	BankName             string `json:"bank_name"`
	BankCountry          string `json:"bank_country"`
	BankAddress          string `json:"bank_address"`
	IBAN                 string `json:"iban"`
	Swift                string `json:"swift"`
	RemittanceLineNumber string `json:"remittance_line_number,omitempty"`
	AgentBankName        string `json:"agent_bank_name,omitempty"`
	AgentBankSwift       string `json:"agent_bank_swift,omitempty"`
	DocumentationFile    string `json:"documentation_file,omitempty"`
	DocumentationFileKey string `json:"documentation_file_key,omitempty"` // The file key returned by PreUploadOTCFile
	FileType             string `json:"file_type,omitempty"`              // MIME type of the pre-uploaded file
}

// OTCBankCardRequestResponse represents a bank-card request response
type OTCBankCardRequestResponse struct {
	BankID uint64 `json:"bank_id"`
	Status uint64 `json:"status"`
}

// OTCBankSupplementChecklist holds the supplementary materials Gate requires for a bank card
type OTCBankSupplementChecklist struct {
	UserType string                           `json:"user_type"`
	Items    []OTCBankSupplementChecklistItem `json:"items"`
}

// OTCBankSupplementChecklistItem describes a single supplementary material to submit
type OTCBankSupplementChecklistItem struct {
	Description string `json:"description"`
}

// OTCBankPersonalSupplementMultipartRequest represents bank card supplementary materials for personal verification.
// Materials are supplied as file content, as pre-uploaded file keys in RelationshipProof, or a mix of both.
type OTCBankPersonalSupplementMultipartRequest struct {
	BankID            string `json:"bank_id"`
	IDDocumentFront   string `json:"id_document_front,omitempty"`
	IDDocumentBack    string `json:"id_document_back,omitempty"`
	AddressProof      string `json:"address_proof,omitempty"`
	RelationshipProof string `json:"relationship_proof,omitempty"` // JSON text listing pre-uploaded files by category
}

// OTCBankEnterpriseSupplementMultipartRequest represents bank card supplementary materials for enterprise
// verification. Materials are supplied as file content, as pre-uploaded file keys in RelationshipProof, or a mix
// of both.
type OTCBankEnterpriseSupplementMultipartRequest struct {
	UID                   string `json:"uid,omitempty"`
	BankID                string `json:"bank_id"`
	Certificate           string `json:"certificate,omitempty"`
	ShareHolders          string `json:"share_holders,omitempty"`
	Passport              string `json:"passport,omitempty"`
	ShareHoldingStructure string `json:"share_holding_structure,omitempty"`
	FundsStatement        string `json:"funds_statement,omitempty"`
	Additional            string `json:"additional,omitempty"`
	RelationshipProof     string `json:"relationship_proof,omitempty"` // JSON text listing pre-uploaded files by category
}

// OTCPreUploadRequest holds the parameters for requesting a temporary-bucket file upload credential
type OTCPreUploadRequest struct {
	// ContentType is the file's plain MIME type: image/png, image/jpeg, image/jpg or application/pdf. It is base64
	// encoded when sent, as the gateway may block a plain value containing "/".
	ContentType string
	// Scene is general, bank, assessment or credit, and defaults to general when empty
	Scene string
}

// OTCPreUploadResponse holds the credential and S3 POST policy for uploading a file to the temporary bucket
type OTCPreUploadResponse struct {
	FileKey          string                   `json:"file_key"` // Base64 object path to pass unchanged to the submitting endpoint
	URL              string                   `json:"url"`
	Fields           OTCPreUploadPolicyFields `json:"fields"`
	ExpiresInSeconds uint64                   `json:"expires_in"`
}

// OTCPreUploadPolicyFields holds the S3 POST policy fields to send unchanged as form data when uploading the file
type OTCPreUploadPolicyFields struct {
	Key         string `json:"key"`
	ContentType string `json:"Content-Type"`
	Credential  string `json:"X-Amz-Credential"`
	Algorithm   string `json:"X-Amz-Algorithm"`
	Date        string `json:"X-Amz-Date"`
	Policy      string `json:"Policy"`
	Signature   string `json:"X-Amz-Signature"`
}

// OTCMarkOrderPaidRequest holds request parameters for marking a fiat order as paid
type OTCMarkOrderPaidRequest struct {
	OrderID               string `json:"order_id"`
	ClientOrderID         string `json:"client_order_id,omitempty"`
	PaymentReceiptFileKey string `json:"payment_receipt_file_key"` // The file key returned by PreUploadOTCFile
	PaymentReceipt        string `json:"payment_receipt,omitempty"`
}

// OTCFiatOrderListRequest holds the optional filters for the fiat order list
type OTCFiatOrderListRequest struct {
	Type           string // BUY or SELL
	FiatCurrency   currency.Code
	CryptoCurrency currency.Code
	StartTime      time.Time
	EndTime        time.Time
	Status         string // DONE, CANCEL, PROCESSING or DISBURSED
	PageNumber     uint64
	PageSize       uint64
}

// OTCOrderListItem holds a single fiat order list item.
type OTCOrderListItem struct {
	Time               string          `json:"time"` // Display time, as Timestamp carries the same moment
	Timestamp          types.Time      `json:"timestamp"`
	OrderID            string          `json:"order_id"`
	TradeNumber        string          `json:"trade_no"`
	Type               string          `json:"type"`
	Status             string          `json:"status"`
	FiatCurrency       currency.Code   `json:"fiat_currency"`
	FiatCurrencyInfo   *NameAndIconURL `json:"fiat_currency_info"`
	FiatAmount         types.Number    `json:"fiat_amount"`
	CryptoCurrency     currency.Code   `json:"crypto_currency"`
	CryptoCurrencyInfo *NameAndIconURL `json:"crypto_currency_info"`
	CryptoAmount       types.Number    `json:"crypto_amount"`
	Rate               types.Number    `json:"rate"`
	PromotionCode      string          `json:"promotion_code"`
}

// NameAndIconURL represents a name and asset icon URL.
type NameAndIconURL struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// OTCOrderListData holds the fiat order list response data.
type OTCOrderListData struct {
	PageNumber      uint64              `json:"pn"`
	PageSize        uint64              `json:"ps"`
	TotalPageNumber uint64              `json:"total_pn"`
	Count           uint64              `json:"count"`
	List            []*OTCOrderListItem `json:"list"`
}

// UnmarshalJSON decodes the order list from either "list", as documented, or "lists", as the live API has returned it
func (o *OTCOrderListData) UnmarshalJSON(data []byte) error {
	type Alias OTCOrderListData
	aux := &struct {
		Lists []*OTCOrderListItem `json:"lists"`
		*Alias
	}{Alias: (*Alias)(o)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if o.List == nil {
		o.List = aux.Lists
	}
	return nil
}

// OTCStablecoinOrderListRequest holds the optional filters for the stablecoin order list
type OTCStablecoinOrderListRequest struct {
	Currency   currency.Code
	Status     string // PROCESSING, DONE or FAILED
	StartTime  time.Time
	EndTime    time.Time
	PageNumber uint64
	PageSize   uint64
}

// OTCStableCoinOrder holds a single stablecoin order item.
type OTCStableCoinOrder struct {
	ID                     uint64        `json:"id"`
	TradeNumber            string        `json:"trade_no"`
	PayCoin                currency.Code `json:"pay_coin"`
	PayIcon                string        `json:"pay_icon"`
	PayAmount              types.Number  `json:"pay_amount"`
	GetCoin                currency.Code `json:"get_coin"`
	GetIcon                string        `json:"get_icon"`
	GetAmount              types.Number  `json:"get_amount"`
	Rate                   types.Number  `json:"rate"`
	ReciprocalExchangeRate types.Number  `json:"rate_reci"`
	Status                 string        `json:"status"`
	CreateTimestamp        types.Time    `json:"create_timest"`
	CreateTime             string        `json:"create_time"` // Display time, as CreateTimestamp carries the same moment
}

// OTCStablecoinOrderListData holds the stablecoin order list response data.
type OTCStablecoinOrderListData struct {
	Total      uint64                `json:"total"`
	PageSize   uint64                `json:"page_size"`
	PageNumber uint64                `json:"page_number"`
	TotalPage  uint64                `json:"total_page"`
	List       []*OTCStableCoinOrder `json:"list"`
}

// OTCOrderDetailData holds the fiat order detail data. The Gate beneficiary fields are only set for BUY orders.
type OTCOrderDetailData struct {
	OrderID                       string         `json:"order_id"`
	UID                           string         `json:"uid"`
	Type                          string         `json:"type"`
	FiatCurrency                  currency.Code  `json:"fiat_currency"`
	FiatAmount                    types.Number   `json:"fiat_amount"`
	CryptoCurrency                currency.Code  `json:"crypto_currency"`
	CryptoAmount                  types.Number   `json:"crypto_amount"`
	Rate                          types.Number   `json:"rate"`
	BankAccountName               string         `json:"bank_account_name"`
	BankName                      string         `json:"bank_name"`
	BankCountry                   string         `json:"bank_country"`
	BankAddress                   string         `json:"bank_address"`
	BankAccountNumberIBAN         string         `json:"bank_account_number_iban"`
	SwiftCode                     string         `json:"swift_code"`
	IntermediaryBankName          string         `json:"intermediate_bank_name"`
	IntermediaryBankSwiftCode     string         `json:"intermediary_bank_swift_code"`
	GateBankAccountName           string         `json:"gate_bank_account_name"`
	GateBankName                  string         `json:"gate_bank_name"`
	GateBankCountry               string         `json:"gate_bank_country"`
	GateBankAddress               string         `json:"gate_bank_address"`
	GateBankAccountNumberIBAN     string         `json:"gate_bank_account_number_iban"`
	GateSwiftCode                 string         `json:"gate_swift_code"`
	GateIntermediaryBankName      string         `json:"gate_intermediary_bank_name"`
	GateIntermediaryBankSwiftCode string         `json:"gate_intermediary_bank_swift_code"`
	GateTransferRemark            string         `json:"gate_transfer_remark"`
	GateReferenceCode             string         `json:"gate_reference_code"`
	Status                        string         `json:"status"`
	CreateTime                    types.DateTime `json:"create_time"`
}
