package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetFiatStablecoinQuote creates a fiat and stablecoin quote, supporting both PAY and GET directions.
func (e *Exchange) GetFiatStablecoinQuote(ctx context.Context, arg *OTCQuoteRequest) (*OTCQuoteData, error) {
	if arg.Side == "" {
		return nil, errOTCSideRequired
	}
	if arg.PayCoin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.GetCoin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp otcAPIResponse[*OTCQuoteData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcQuoteEPL, http.MethodPost, "otc/quote", nil, arg, &resp)
}

// CreateFiatOrder creates a fiat order, supporting BUY for on-ramp and SELL for off-ramp.
func (e *Exchange) CreateFiatOrder(ctx context.Context, arg *OTCFiatOrderRequest) (*OTCActionResponse, error) {
	if arg.Type == "" {
		return nil, errOTCOrderTypeRequired
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w, quote direction is required", order.ErrSideIsInvalid)
	}
	if arg.QuoteToken == "" {
		return nil, errOTCQuoteTokenRequired
	}
	if arg.BankID == "" {
		return nil, errOTCBankIDRequired
	}
	if arg.CryptoCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w; crypty currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w; fiat currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.CryptoAmount <= 0 {
		return nil, fmt.Errorf("%w crypto amount must be set", order.ErrAmountMustBeSet)
	}
	if arg.FiatAmount <= 0 {
		return nil, fmt.Errorf("%w fiat amount must be set", order.ErrAmountMustBeSet)
	}
	var resp *OTCActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderCreateEPL, http.MethodPost, "otc/order/create", nil, arg, &resp)
}

// CreateStablecoinOrder creates a stablecoin order.
func (e *Exchange) CreateStablecoinOrder(ctx context.Context, arg *OTCStablecoinOrderRequest) (*OTCActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *OTCActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcStablecoinOrderCreateEPL, http.MethodPost, "otc/stable_coin/order/create", nil, arg, &resp)
}

// GetUserBankCardList retrieves the user's bank card list.
func (e *Exchange) GetUserBankCardList(ctx context.Context) ([]*OTCBankCard, error) {
	var resp otcAPIResponse[*OTCBankCardListData]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankListEPL, http.MethodGet, "otc/bank/list", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.Lists, nil
}

// CreateBankCard binds a bank card. Under the Global entity, an account with a non-matching name may enter manual review (status pending) and require subsequent supplementary materials.
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
		return nil, errIBANAddresRequired
	}
	if arg.Swift == "" {
		return nil, errSwiftAddressRequired
	}
	if arg.DocumentationFile == "" {
		return nil, errDocumentationFileRequired
	}
	var resp otcAPIResponse[*OTCBankCardRequestResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankCreateEPL, http.MethodPost, "otc/bank/create", nil, arg, &resp)
}

// DeleteBankCard deletes a bank-card information
func (e *Exchange) DeleteBankCard(ctx context.Context, bankID string) error {
	if bankID == "" {
		return errOTCBankIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankDeleteEPL, http.MethodPost, "otc/bank/delete", nil, &map[string]string{"bank_id": bankID}, nil)
}

// SetDefaultBankCard set the specified bank card as default.
func (e *Exchange) SetDefaultBankCard(ctx context.Context, bankID string) error {
	if bankID == "" {
		return errOTCBankIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankSetDefaultEPL, http.MethodPost, "otc/bank/delete", nil, &map[string]string{"bank_id": bankID}, nil)
}

// GetCheckListOfMaterialsToSupplementForBankCard query the checklist of materials to supplement for a bank card
func (e *Exchange) GetCheckListOfMaterialsToSupplementForBankCard(ctx context.Context, bankID string) (*OTCBankSupplementChecklistItem, error) {
	if bankID == "" {
		return nil, errOTCBankIDRequired
	}
	params := url.Values{}
	params.Set("bank_id", bankID)

	var resp otcAPIResponse[*OTCBankSupplementChecklistItem]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankSupplementChecklistEPL, http.MethodGet, "otc/bank/bank_supplement_checklist", params, nil, &resp)
}

// SubmitBankCardSupplementMaterials submit Bank Card Supplement Materials
func (e *Exchange) SubmitBankCardSupplementMaterials(ctx context.Context, arg *OTCBankPersonalSupplementMultipartRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.BankID == "" {
		return errOTCBankIDRequired
	}
	if arg.IDDocumentFront == "" {
		return fmt.Errorf("%w ID document front-side file content required", errDocumentationFileRequired)
	}
	if arg.IDDocumentBack == "" {
		return fmt.Errorf("%w ID document back-side file content required", errDocumentationFileRequired)
	}
	if arg.AddressProof == "" {
		return fmt.Errorf("%w address proof is required", errBankAddressRequired)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankPersonalSupplementEPL, http.MethodPost, "otc/bank/personal/bank_supplement", nil, arg, nil)
}

// SubmitEnterpriseBankCardSupplementMaterials users submit supplementary materials.
func (e *Exchange) SubmitEnterpriseBankCardSupplementMaterials(ctx context.Context, arg *OTCBankEnterpriseSupplementMultipartRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.BankID == "" {
		return errOTCBankIDRequired
	}
	if arg.Certificate == "" {
		return errBusinessLicenseCertificateRequired
	}
	if arg.ShareHolders == "" {
		return errShareholdersRequired
	}
	if arg.Passport == "" {
		return errPassportRequired
	}
	if arg.ShareHolders == "" {
		return fmt.Errorf("%w ownership structure chart file content", errShareholdersRequired)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcBankEnterpriseSupplementEPL, http.MethodPost, "otc/bank/personal/bank_supplement", nil, arg, nil)
}

// MarkFiatOrderAsPaid marks a fiat order as paid.
func (e *Exchange) MarkFiatOrderAsPaid(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp *OTCActionResponse
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderPaidEPL, http.MethodPost, "otc/order/paid", nil, &OTCMarkOrderPaidRequest{OrderID: orderID}, &resp)
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
func (e *Exchange) GetFiatOrderList(ctx context.Context, orderType, status string, fiatCurrency, cryptoCurrency currency.Code, fromTime, endTime time.Time, pageNumber, pageSize uint64) (*OTCOrderListData, error) {
	if !fromTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(fromTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !fiatCurrency.IsEmpty() {
		params.Set("fiat_currency", fiatCurrency.String())
	}
	if !cryptoCurrency.IsEmpty() {
		params.Set("crypto_currency", cryptoCurrency.String())
	}
	if status != "" {
		params.Set("status", status)
	}
	if !fromTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(fromTime.UTC().Unix(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(endTime.UTC().Unix(), 10))
	}
	if pageNumber > 0 {
		params.Set("pn", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("ps", strconv.FormatUint(pageSize, 10))
	}
	var resp otcAPIResponse[*OTCOrderListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderListEPL, http.MethodGet, "otc/order/list", params, nil, &resp)
}

// GetStablecoinOrderList retrieves the stablecoin order list with optional filters.
func (e *Exchange) GetStablecoinOrderList(ctx context.Context, coinName currency.Code, status string, startTime, endTime time.Time, pageNumber, pageSize uint64) (*OTCStablecoinOrderListData, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !coinName.IsEmpty() {
		params.Set("coin_name", coinName.String())
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.UTC().Unix(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(endTime.UTC().Unix(), 10))
	}
	if pageNumber > 0 {
		params.Set("page_number", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp otcAPIResponse[*OTCStablecoinOrderListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcStablecoinOrderListEPL, http.MethodGet, "otc/stable_coin/order/list", params, nil, &resp)
}

// GetFiatOrderDetail retrieves details for a specific fiat order.
func (e *Exchange) GetFiatOrderDetail(ctx context.Context, orderID string) (*OTCOrderDetailData, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp otcAPIResponse[*OTCOrderDetailData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, otcOrderDetailEPL, http.MethodGet, "otc/order/detail", params, nil, &resp)
}
