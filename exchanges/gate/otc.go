package gate

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errPaymentReceiptFileKeyRequired = errors.New("payment receipt file key required")
	errOTCSideRequired               = errors.New("OTC side (PAY or GET) is required")
	errOTCOrderTypeRequired          = errors.New("OTC order type (BUY or SELL) is required")
	errOTCQuoteTokenRequired         = errors.New("OTC quote token is required")
	errOTCBankIDRequired             = errors.New("OTC bank ID is required")
	errBankAccountNameRequired       = errors.New("OTC bank account name is required")
	errBankNameRequired              = errors.New("OTC bank name is required")
	errBankCountryRequired           = errors.New("OTC bank country is required")
	errBankAddressRequired           = errors.New("OTC bank address is required")
	errIBANRequired                  = errors.New("OTC IBAN is required")
	errSwiftCodeRequired             = errors.New("OTC SWIFT code is required")
	errDocumentationFileRequired     = errors.New("OTC documentation file or pre-uploaded documentation file key is required")
	errDocumentationFileConflict     = errors.New("OTC documentation file and documentation file key are mutually exclusive")
	errFileTypeRequired              = errors.New("OTC file type is required with a pre-uploaded file key")
	errSupplementMaterialRequired    = errors.New("OTC supplementary material file or relationship proof is required")
	errContentTypeRequired           = errors.New("OTC file content type is required")
)

// GetFiatStablecoinQuote creates a fiat and stablecoin quote, supporting both PAY and GET directions.
func (e *Exchange) GetFiatStablecoinQuote(ctx context.Context, arg *OTCQuoteRequest) (*OTCQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, errOTCSideRequired
	}
	if arg.PayCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: pay coin required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.GetCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: get coin required", currency.ErrCurrencyCodeEmpty)
	}
	switch arg.Side {
	case "PAY":
		if arg.PayAmount <= 0 {
			return nil, fmt.Errorf("%w: pay amount required for a PAY quote", order.ErrAmountMustBeSet)
		}
	case "GET":
		if arg.GetAmount <= 0 {
			return nil, fmt.Errorf("%w: get amount required for a GET quote", order.ErrAmountMustBeSet)
		}
	}
	var resp gateAPIResponse[*OTCQuoteResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcQuoteEPL, http.MethodPost, "otc/quote", nil, arg, &resp)
}

// CreateFiatOrder creates a fiat order, supporting BUY for on-ramp and SELL for off-ramp.
func (e *Exchange) CreateFiatOrder(ctx context.Context, arg *OTCFiatOrderRequest) (*OTCActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == "" {
		return nil, errOTCOrderTypeRequired
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w: quote side is required", order.ErrSideIsInvalid)
	}
	if arg.QuoteToken == "" {
		return nil, errOTCQuoteTokenRequired
	}
	if arg.BankID == "" {
		return nil, errOTCBankIDRequired
	}
	if arg.CryptoCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: crypto currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: fiat currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.CryptoAmount <= 0 {
		return nil, fmt.Errorf("%w: crypto amount required", order.ErrAmountMustBeSet)
	}
	if arg.FiatAmount <= 0 {
		return nil, fmt.Errorf("%w: fiat amount required", order.ErrAmountMustBeSet)
	}
	var resp *OTCActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderCreateEPL, http.MethodPost, "otc/order/create", nil, arg, &resp)
}

// CreateStablecoinOrder creates a stablecoin order.
func (e *Exchange) CreateStablecoinOrder(ctx context.Context, arg *OTCStablecoinOrderRequest) (*OTCActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PayCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: pay coin required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.GetCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: get coin required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.PayAmount <= 0 {
		return nil, fmt.Errorf("%w: pay amount required", order.ErrAmountMustBeSet)
	}
	if arg.GetAmount <= 0 {
		return nil, fmt.Errorf("%w: get amount required", order.ErrAmountMustBeSet)
	}
	if arg.Side == "" {
		return nil, errOTCSideRequired
	}
	if arg.QuoteToken == "" {
		return nil, errOTCQuoteTokenRequired
	}
	var resp *OTCActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcStablecoinOrderCreateEPL, http.MethodPost, "otc/stable_coin/order/create", nil, arg, &resp)
}

// GetUserBankCardList retrieves the user's bank card list. The default card is flagged by IsDefault.
func (e *Exchange) GetUserBankCardList(ctx context.Context) ([]*OTCBankCard, error) {
	var resp gateAPIResponse[*OTCBankCardListResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankListEPL, http.MethodGet, "otc/bank/list", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.Lists, nil
}

// CreateBankCard binds a bank card. Under the Global entity, an account with a non-matching name may enter manual
// review (status pending) and require subsequent supplementary materials.
// For more, see: https://www.gate.com/docs/developers/apiv4/en/otc/#create-bank-card
func (e *Exchange) CreateBankCard(ctx context.Context, arg *OTCBankCreateMultipartRequest) (*OTCBankCardRequestResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.BankAccountName == "" {
		return nil, errBankAccountNameRequired
	}
	if arg.BankName == "" {
		return nil, errBankNameRequired
	}
	if arg.BankCountry == "" {
		return nil, errBankCountryRequired
	}
	if arg.BankAddress == "" {
		return nil, errBankAddressRequired
	}
	if arg.IBAN == "" {
		return nil, errIBANRequired
	}
	if arg.Swift == "" {
		return nil, errSwiftCodeRequired
	}
	if arg.DocumentationFile == "" && arg.DocumentationFileKey == "" {
		return nil, errDocumentationFileRequired
	}
	if arg.DocumentationFile != "" && arg.DocumentationFileKey != "" {
		return nil, errDocumentationFileConflict
	}
	if arg.DocumentationFileKey != "" && arg.FileType == "" {
		return nil, errFileTypeRequired
	}
	var resp gateAPIResponse[*OTCBankCardRequestResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankCreateEPL, http.MethodPost, "otc/bank/create", nil, arg, &resp)
}

// DeleteBankCard deletes a bank card
func (e *Exchange) DeleteBankCard(ctx context.Context, bankID string) error {
	if bankID == "" {
		return errOTCBankIDRequired
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankDeleteEPL, http.MethodPost, "otc/bank/delete", nil, &map[string]string{"bank_id": bankID}, &resp)
}

// SetDefaultBankCard sets the specified bank card as the default
func (e *Exchange) SetDefaultBankCard(ctx context.Context, bankID string) error {
	if bankID == "" {
		return errOTCBankIDRequired
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankSetDefaultEPL, http.MethodPost, "otc/bank/set_default", nil, &map[string]string{"bank_id": bankID}, &resp)
}

// GetCheckListOfMaterialsToSupplementForBankCard retrieves the supplementary materials required for a bank card,
// based on the user's approved advanced verification type
func (e *Exchange) GetCheckListOfMaterialsToSupplementForBankCard(ctx context.Context, bankID string) (*OTCBankSupplementChecklistResponse, error) {
	if bankID == "" {
		return nil, errOTCBankIDRequired
	}
	params := url.Values{}
	params.Set("bank_id", bankID)
	var resp gateAPIResponse[*OTCBankSupplementChecklistResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankSupplementChecklistEPL, http.MethodGet, "otc/bank/bank_supplement_checklist", params, nil, &resp)
}

// SubmitBankCardSupplementMaterials submits bank card supplementary materials for personal verification
func (e *Exchange) SubmitBankCardSupplementMaterials(ctx context.Context, arg *OTCBankPersonalSupplementMultipartRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.BankID == "" {
		return errOTCBankIDRequired
	}
	if arg.IDDocumentFront == "" && arg.IDDocumentBack == "" && arg.AddressProof == "" && arg.RelationshipProof == "" {
		return errSupplementMaterialRequired
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankPersonalSupplementEPL, http.MethodPost, "otc/bank/personal/bank_supplement", nil, arg, &resp)
}

// SubmitEnterpriseBankCardSupplementMaterials submits bank card supplementary materials for enterprise verification
func (e *Exchange) SubmitEnterpriseBankCardSupplementMaterials(ctx context.Context, arg *OTCBankEnterpriseSupplementMultipartRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.BankID == "" {
		return errOTCBankIDRequired
	}
	if arg.Certificate == "" && arg.ShareHolders == "" && arg.Passport == "" && arg.ShareHoldingStructure == "" &&
		arg.FundsStatement == "" && arg.Additional == "" && arg.RelationshipProof == "" {
		return errSupplementMaterialRequired
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankEnterpriseSupplementEPL, http.MethodPost, "otc/bank/enterprise/bank_supplement", nil, arg, &resp)
}

// PreUploadOTCFile retrieves a temporary-bucket POST policy for uploading a file directly to S3. The returned
// FileKey is then passed unchanged to the endpoint the file supports, such as MarkFiatOrderAsPaid or CreateBankCard.
func (e *Exchange) PreUploadOTCFile(ctx context.Context, arg *OTCPreUploadRequest) (*OTCPreUploadResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ContentType == "" {
		return nil, errContentTypeRequired
	}
	body := map[string]string{"content_type": base64.StdEncoding.EncodeToString([]byte(arg.ContentType))}
	if arg.Scene != "" {
		body["scene"] = arg.Scene
	}
	var resp gateAPIResponse[*OTCPreUploadResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcUploadPreUploadEPL, http.MethodPost, "otc/upload/pre_upload", nil, &body, &resp)
}

// MarkFiatOrderAsPaid marks a fiat buy order as paid, which requires the payment receipt's pre-uploaded file key
func (e *Exchange) MarkFiatOrderAsPaid(ctx context.Context, arg *OTCMarkOrderPaidRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.PaymentReceiptFileKey == "" {
		return errPaymentReceiptFileKeyRequired
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderPaidEPL, http.MethodPost, "otc/order/paid", nil, arg, &resp)
}

// CancelFiatOrder cancels a fiat order.
func (e *Exchange) CancelFiatOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderCancelEPL, http.MethodPost, "otc/order/cancel", params, nil, &resp)
}

// GetFiatOrderList retrieves the fiat order list with optional filters.
func (e *Exchange) GetFiatOrderList(ctx context.Context, arg *OTCFiatOrderListRequest) (*OTCOrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if !arg.FiatCurrency.IsEmpty() {
		params.Set("fiat_currency", arg.FiatCurrency.String())
	}
	if !arg.CryptoCurrency.IsEmpty() {
		params.Set("crypto_currency", arg.CryptoCurrency.String())
	}
	if !arg.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(arg.StartTime.Unix(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if arg.PageNumber > 0 {
		params.Set("pn", strconv.FormatUint(arg.PageNumber, 10))
	}
	if arg.PageSize > 0 {
		params.Set("ps", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp gateAPIResponse[*OTCOrderListResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderListEPL, http.MethodGet, "otc/order/list", params, nil, &resp)
}

// GetStablecoinOrderList retrieves the stablecoin order list with optional filters.
func (e *Exchange) GetStablecoinOrderList(ctx context.Context, arg *OTCStablecoinOrderListRequest) (*OTCStablecoinOrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.PageNumber > 0 {
		params.Set("page_number", strconv.FormatUint(arg.PageNumber, 10))
	}
	if !arg.Currency.IsEmpty() {
		params.Set("coin_name", arg.Currency.String())
	}
	if !arg.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(arg.StartTime.Unix(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	var resp gateAPIResponse[*OTCStablecoinOrderListResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcStablecoinOrderListEPL, http.MethodGet, "otc/stable_coin/order/list", params, nil, &resp)
}

// GetFiatOrderDetail retrieves details for a specific fiat order.
func (e *Exchange) GetFiatOrderDetail(ctx context.Context, orderID string) (*OTCOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp gateAPIResponse[*OTCOrderDetailResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderDetailEPL, http.MethodGet, "otc/order/detail", params, nil, &resp)
}
