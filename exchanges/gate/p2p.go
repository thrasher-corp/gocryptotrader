package gate

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errBizUIDRequired          = errors.New("counterparty biz uid required")
	errP2PAdIDRequired         = errors.New("P2P advertisement ID required")
	errP2PTradeTypeRequired    = errors.New("P2P trade type required")
	errP2PAdStatusInvalid      = errors.New("P2P ad status must be 1 (listed), 3 (delisted), or 4 (closed)")
	errP2PAdOperationInvalid   = errors.New("P2P ad operation type must be 0 to 3")
	errP2PLimitBasisInvalid    = errors.New("P2P limit basis must be 0 (crypto) or 1 (fiat)")
	errP2PPayTypeRequired      = errors.New("P2P payment types required")
	errP2PUnitPriceRequired    = errors.New("P2P ad unit price required")
	errP2PTradeLimitRequired   = errors.New("P2P per-order minimum and maximum required for the ad's limit basis")
	errP2PMessageRequired      = errors.New("P2P chat message required")
	errP2PImageTypeRequired    = errors.New("P2P image content type required")
	errP2PImageDataRequired    = errors.New("P2P base64 image data required")
	errP2PWorkStatusInvalid    = errors.New("P2P work status must be 0 (resting), 1 (working), or 2 (custom working hours)")
	errP2PCycleTypeRequired    = errors.New("P2P custom working cycle type required")
	errP2PDayOfWeekRequired    = errors.New("P2P working days required for a weekly working cycle")
	errP2PTimeZoneRequired     = errors.New("P2P working hours time zone required")
	errP2PWorkingHoursRequired = errors.New("P2P custom working start and end times required")
)

// GetP2PAccountInfo retrieves the current user's P2P merchant account information.
func (e *Exchange) GetP2PAccountInfo(ctx context.Context) (*P2PMerchantInfoResponse, error) {
	var resp gateAPIResponse[*P2PMerchantInfoResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pAccountInfoEPL, http.MethodPost, "p2p/merchant/account/get_user_info", nil, nil, &resp)
}

// GetP2PCounterpartyInfo retrieves P2P user information for a counterparty by their biz_uid.
func (e *Exchange) GetP2PCounterpartyInfo(ctx context.Context, arg *GetCounterpartyInfoRequest) (*P2PCounterpartyInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.BizUID == "" {
		return nil, errBizUIDRequired
	}
	var resp gateAPIResponse[*P2PCounterpartyInfoResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pCounterpartyInfoEPL, http.MethodPost, "p2p/merchant/account/get_counterparty_user_info", nil, arg, &resp)
}

// GetP2PPaymentMethods retrieves the current user's bound P2P payment methods.
func (e *Exchange) GetP2PPaymentMethods(ctx context.Context, arg *GetP2PPaymentMethodsRequest) ([]*P2PPaymentMethodGroupResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	body := make(map[string]string, 1)
	if !arg.Fiat.IsEmpty() {
		body["fiat"] = arg.Fiat.String()
	}
	var resp gateAPIResponse[[]*P2PPaymentMethodGroupResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pPaymentMethodsEPL, http.MethodPost, "p2p/merchant/account/get_myself_payment", nil, &body, &resp)
}

// SetMerchantWorkingStatusAndCustomWorking sets the merchant's working status, along with custom working hours when
// the status is P2PWorkStatusCustomHours
func (e *Exchange) SetMerchantWorkingStatusAndCustomWorking(ctx context.Context, arg *SetMerchantWorkHoursRequest) (*P2PMerchantWorkStatusResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.WorkStatus > P2PWorkStatusCustomHours {
		return nil, errP2PWorkStatusInvalid
	}
	if arg.WorkStatus == P2PWorkStatusCustomHours {
		if arg.CycleType == "" {
			return nil, errP2PCycleTypeRequired
		}
		if arg.CycleType == "Weekly" && arg.DayOfWeek == "" {
			return nil, errP2PDayOfWeekRequired
		}
		if arg.TimeZone == "" {
			return nil, errP2PTimeZoneRequired
		}
		if arg.StartTime == "" || arg.EndTime == "" {
			return nil, errP2PWorkingHoursRequired
		}
	}
	var resp gateAPIResponse[*P2PMerchantWorkStatusResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pSetWorkHoursEPL, http.MethodPost, "p2p/merchant/account/set_merchant_work_hours", nil, arg, &resp)
}

// GetPendingP2POrders retrieves a list of pending P2P orders.
func (e *Exchange) GetPendingP2POrders(ctx context.Context, arg *PendingP2POrderRequest) (*P2POrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CryptoCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: crypto currency is missing", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: fiat currency is missing", currency.ErrCurrencyCodeEmpty)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	var resp gateAPIResponse[*P2POrderListResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pPendingTransactionsEPL, http.MethodPost, "p2p/merchant/transaction/get_pending_transaction_list", nil, arg, &resp)
}

// GetHistoricalP2POrders retrieves a list of all or historical P2P orders
func (e *Exchange) GetHistoricalP2POrders(ctx context.Context, arg *P2PCompletedOrderRequest) (*P2POrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CryptoCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: crypto currency is missing", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: fiat currency is missing", currency.ErrCurrencyCodeEmpty)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	var resp gateAPIResponse[*P2POrderListResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pCompletedTransactionsEPL, http.MethodPost, "p2p/merchant/transaction/get_completed_transaction_list", nil, arg, &resp)
}

// GetP2POrderDetails retrieves detailed information for a specific P2P order.
func (e *Exchange) GetP2POrderDetails(ctx context.Context, arg *GetP2POrderDetailsRequest) (*P2POrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TransactionID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp gateAPIResponse[*P2POrderDetailResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pTransactionDetailsEPL, http.MethodPost, "p2p/merchant/transaction/get_transaction_details", nil, arg, &resp)
}

// ConfirmP2PPayment confirms that payment has been made for a P2P order.
func (e *Exchange) ConfirmP2PPayment(ctx context.Context, arg *ConfirmP2PPaymentRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp gateAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pConfirmPaymentEPL, http.MethodPost, "p2p/merchant/transaction/confirm-payment", nil, arg, &resp)
}

// ConfirmP2PReceipt confirms that payment has been received for a P2P order.
func (e *Exchange) ConfirmP2PReceipt(ctx context.Context, arg *ConfirmP2PReceiptRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp gateAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pConfirmReceiptEPL, http.MethodPost, "p2p/merchant/transaction/confirm-receipt", nil, arg, &resp)
}

// CancelP2POrder cancels a P2P order.
func (e *Exchange) CancelP2POrder(ctx context.Context, arg *CancelP2POrderRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp gateAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pCancelTransactionEPL, http.MethodPost, "p2p/merchant/transaction/cancel", nil, arg, &resp)
}

// PublishP2PAdOrder publishes a new P2P advertisement or edits an existing one. When Gate's risk control rejects the
// ad's trade tips or auto reply, the risk details are returned alongside the error.
func (e *Exchange) PublishP2PAdOrder(ctx context.Context, arg *PublishP2PAdRequest) (*P2PPublishAdResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyType.IsEmpty() {
		return nil, fmt.Errorf("%w: crypto currency type required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ExchangeType.IsEmpty() {
		return nil, fmt.Errorf("%w: fiat exchange type required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Type > P2PAdOperationEditBuy {
		return nil, errP2PAdOperationInvalid
	}
	if (arg.Type == P2PAdOperationEditSell || arg.Type == P2PAdOperationEditBuy) && arg.AdvertisementID == "" {
		return nil, fmt.Errorf("%w: editing an ad requires its ID", errP2PAdIDRequired)
	}
	if arg.UnitPrice <= 0 {
		return nil, errP2PUnitPriceRequired
	}
	if arg.Number <= 0 {
		return nil, fmt.Errorf("%w: ad amount required", order.ErrAmountMustBeSet)
	}
	if arg.PayType == "" {
		return nil, errP2PPayTypeRequired
	}
	switch arg.LimitBasis {
	case P2PLimitBasisCrypto:
		if arg.MinAmount <= 0 || arg.MaxAmount <= 0 {
			return nil, errP2PTradeLimitRequired
		}
	case P2PLimitBasisFiat:
		if arg.FiatMinAmount <= 0 || arg.FiatMaxAmount <= 0 {
			return nil, errP2PTradeLimitRequired
		}
	default:
		return nil, errP2PLimitBasisInvalid
	}
	var resp gateAPIResponse[*P2PPublishAdResponse]
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pPublishAdEPL, http.MethodPost, "p2p/merchant/books/place_biz_push_order", nil, arg, &resp)
	return resp.Data, err
}

// UpdateP2PAdStatus updates the status of a P2P advertisement to 1 (listed), 3 (delisted) or 4 (closed).
func (e *Exchange) UpdateP2PAdStatus(ctx context.Context, arg *UpdateP2PAdStatusRequest) (*P2PUpdateAdStatusResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AdvertisementID == 0 {
		return nil, errP2PAdIDRequired
	}
	if arg.Status != 1 && arg.Status != 3 && arg.Status != 4 {
		return nil, errP2PAdStatusInvalid
	}
	var resp gateAPIResponse[*P2PUpdateAdStatusResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pUpdateAdStatusEPL, http.MethodPost, "p2p/merchant/books/ads_update_status", nil, arg, &resp)
}

// GetP2PAdDetails retrieves detailed information for a specific P2P advertisement.
func (e *Exchange) GetP2PAdDetails(ctx context.Context, arg *GetP2PAdDetailsRequest) (*P2PAdDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AdvertisementID == "" {
		return nil, errP2PAdIDRequired
	}
	var resp gateAPIResponse[*P2PAdDetailResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pAdDetailEPL, http.MethodPost, "p2p/merchant/books/ads_detail", nil, arg, &resp)
}

// GetMyP2PAds retrieves the current user's P2P advertisements.
func (e *Exchange) GetMyP2PAds(ctx context.Context, arg *GetMyP2PAdsRequest) (*P2PMyAdsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	body := make(map[string]string, 3)
	if !arg.Asset.IsEmpty() {
		body["asset"] = arg.Asset.String()
	}
	if !arg.FiatUnit.IsEmpty() {
		body["fiat_unit"] = arg.FiatUnit.String()
	}
	if arg.TradeType != "" {
		body["trade_type"] = arg.TradeType
	}
	var resp gateAPIResponse[*P2PMyAdsResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pMyAdsListEPL, http.MethodPost, "p2p/merchant/books/my_ads_list", nil, &body, &resp)
}

// GetP2PAdList retrieves the P2P advertisement list for a given asset/fiat pair and trade side.
func (e *Exchange) GetP2PAdList(ctx context.Context, arg *GetP2PAdsListRequest) ([]*P2PAdListItemResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Asset.IsEmpty() {
		return nil, fmt.Errorf("%w: P2P asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatUnit.IsEmpty() {
		return nil, fmt.Errorf("%w: P2P fiat unit required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.TradeType == "" {
		return nil, errP2PTradeTypeRequired
	}
	var resp gateAPIResponse[[]*P2PAdListItemResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pAdsListEPL, http.MethodPost, "p2p/merchant/books/ads_list", nil, arg, &resp)
}

// GetP2PChatHistory retrieves the chat history for a P2P order.
func (e *Exchange) GetP2PChatHistory(ctx context.Context, arg *GetP2PChatHistoryRequest) (*P2PChatMessagesResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp gateAPIResponse[*P2PChatMessagesResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pChatHistoryEPL, http.MethodPost, "p2p/merchant/chat/get_chats_list", nil, arg, &resp)
}

// SendP2PChatMessage sends a chat message for a P2P order. A message hitting off-platform traffic diversion risk
// control is still accepted, with the result's RiskType and ToastMessage set.
func (e *Exchange) SendP2PChatMessage(ctx context.Context, arg *SendP2PChatMessageRequest) (*P2PSendMessageResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TransactionID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Message == "" {
		return nil, errP2PMessageRequired
	}
	var resp gateAPIResponse[*P2PSendMessageResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pSendChatMessageEPL, http.MethodPost, "p2p/merchant/chat/send_chat_message", nil, arg, &resp)
}

// UploadP2PChatFile uploads a file for use in P2P chat.
func (e *Exchange) UploadP2PChatFile(ctx context.Context, arg *UploadP2PChatFileRequest) (*P2PUploadFileResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ImageContentType == "" {
		return nil, errP2PImageTypeRequired
	}
	if arg.Base64Img == "" {
		return nil, errP2PImageDataRequired
	}
	var resp gateAPIResponse[*P2PUploadFileResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, p2pUploadChatFileEPL, http.MethodPost, "p2p/merchant/chat/upload_chat_file", nil, arg, &resp)
}
