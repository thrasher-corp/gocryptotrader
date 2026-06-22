package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errBizUIDRequired       = errors.New("counterparty biz uid required")
	errP2PAdIDRequired      = errors.New("P2P advertisement ID required")
	errP2PFiatUnitRequired  = errors.New("P2P fiat unit required")
	errP2PTradeTypeRequired = errors.New("P2P trade type required")
	errP2PPriceTypeInvalid  = errors.New("P2P price type must be 1 (floating) or 2 (fixed)")
	errP2PMinAmountRequired = errors.New("P2P minimum trade amount required")
	errP2PAdStatusInvalid   = errors.New("P2P ad status must be 1 (listed), 3 (delisted), or 4 (closed)")
	errP2PMessageRequired   = errors.New("P2P chat message required")
	errP2PImageTypeRequired = errors.New("P2P image content type required")
	errP2PImageDataRequired = errors.New("P2P base64 image data required")
)

// GetP2PAccountInfo retrieves the current user's P2P merchant account information.
func (e *Exchange) GetP2PAccountInfo(ctx context.Context) (*P2PMerchantInfo, error) {
	var resp p2pAPIResponse[P2PMerchantInfo]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/account/get_user_info", nil, nil, &resp)
}

// GetP2PCounterpartyInfo retrieves P2P user information for a counterparty by their biz_uid.
func (e *Exchange) GetP2PCounterpartyInfo(ctx context.Context, arg *GetCounterpartyInfoRequest) (*P2PCounterpartyInfo, error) {
	if arg.BizUID == "" {
		return nil, errBizUIDRequired
	}
	var resp p2pAPIResponse[P2PCounterpartyInfo]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/account/get_counterparty_user_info", nil, arg, &resp)
}

// GetP2PPaymentMethods retrieves the current user's bound P2P payment methods.
func (e *Exchange) GetP2PPaymentMethods(ctx context.Context, arg *GetP2PPaymentMethodsRequest) ([]*P2PPaymentMethodGroup, error) {
	var resp p2pAPIResponse[[]*P2PPaymentMethodGroup]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/account/get_myself_payment", nil, arg, &resp)
}

// GetP2PPendingOrders retrieves the current user's active (pending) P2P orders.
func (e *Exchange) GetP2PPendingOrders(ctx context.Context, arg *GetP2POrdersRequest) (*P2POrdersData, error) {
	var resp p2pAPIResponse[P2POrdersData]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/my_list", nil, arg, &resp)
}

// GetP2PHistoricalOrders retrieves the current user's historical P2P orders.
func (e *Exchange) GetP2PHistoricalOrders(ctx context.Context, from, to time.Time, page, limit uint64, statusList []int64) (*P2POrdersData, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	arg := &GetP2PHistoricalOrdersRequest{
		StatusList: statusList,
		Page:       page,
		Limit:      limit,
	}
	if !from.IsZero() {
		arg.From = from.UnixMilli()
	}
	if !to.IsZero() {
		arg.To = to.UnixMilli()
	}
	var resp p2pAPIResponse[P2POrdersData]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/my_history_list", nil, arg, &resp)
}

// GetP2POrderDetails retrieves detailed information for a specific P2P order.
// Channel is optional; use "web3" for Web3 orders, omit for normal P2P orders.
func (e *Exchange) GetP2POrderDetails(ctx context.Context, arg *GetP2POrderDetailsRequest) (*P2POrderDetail, error) {
	if arg.TransactionID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp p2pAPIResponse[P2POrderDetail]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/get_transaction_details", nil, arg, &resp)
}

// ConfirmP2PPayment confirms that payment has been made for a P2P order.
func (e *Exchange) ConfirmP2PPayment(ctx context.Context, arg *ConfirmP2PPaymentRequest) error {
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp p2pAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/confirm-payment", nil, arg, &resp)
}

// ConfirmP2PReceipt confirms that payment has been received for a P2P order.
func (e *Exchange) ConfirmP2PReceipt(ctx context.Context, arg *ConfirmP2PReceiptRequest) error {
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp p2pAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/confirm-receipt", nil, arg, &resp)
}

// CancelP2POrder cancels a P2P order.
// ReasonID and ReasonMemo are optional; ReasonMemo is required when ReasonID is 0.
func (e *Exchange) CancelP2POrder(ctx context.Context, arg *CancelP2POrderRequest) error {
	if arg.TransactionID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp p2pAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/transaction/cancel", nil, arg, &resp)
}

// PublishP2PAdOrder publishes a new P2P advertisement.
// PriceType: 1=floating (uses PremiumRatio), 2=fixed (uses FixedPrice).
func (e *Exchange) PublishP2PAdOrder(ctx context.Context, arg *PublishP2PAdRequest) error {
	if arg.Asset.IsEmpty() {
		return fmt.Errorf("%w P2P asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatUnit == "" {
		return errP2PFiatUnitRequired
	}
	if arg.TradeType == "" {
		return errP2PTradeTypeRequired
	}
	if arg.PayIDs == "" {
		return fmt.Errorf("%w P2P payment method IDs required", order.ErrOrderIDNotSet)
	}
	if arg.PriceType != 1 && arg.PriceType != 2 {
		return errP2PPriceTypeInvalid
	}
	if arg.MaxAmount <= 0 {
		return fmt.Errorf("%w P2P maximum trade amount required", limits.ErrAmountBelowMin)
	}
	if arg.MinAmount <= 0 {
		return errP2PMinAmountRequired
	}
	var resp p2pAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/books/publish_ad_order", nil, arg, &resp)
}

// UpdateP2PAdStatus updates the status of a P2P advertisement.
// AdvStatus: 1=listed, 3=delisted, 4=closed.
func (e *Exchange) UpdateP2PAdStatus(ctx context.Context, arg *UpdateP2PAdStatusRequest) (*P2PUpdateAdStatusResult, error) {
	if arg.AdvNo == 0 {
		return nil, fmt.Errorf("%w: adv_no is required", errP2PAdIDRequired)
	}
	if arg.AdvStatus != 1 && arg.AdvStatus != 3 && arg.AdvStatus != 4 {
		return nil, errP2PAdStatusInvalid
	}
	var resp p2pAPIResponse[P2PUpdateAdStatusResult]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/books/ads_update_status", nil, arg, &resp)
}

// GetP2PAdDetails retrieves detailed information for a specific P2P advertisement.
func (e *Exchange) GetP2PAdDetails(ctx context.Context, arg *GetP2PAdDetailsRequest) (*P2PAdDetail, error) {
	if arg.AdvNo == "" {
		return nil, errP2PAdIDRequired
	}
	var resp p2pAPIResponse[P2PAdDetail]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/books/ad_detail", nil, arg, &resp)
}

// GetMyP2PAds retrieves the current user's P2P advertisements.
func (e *Exchange) GetMyP2PAds(ctx context.Context, arg *GetMyP2PAdsRequest) (*P2PMyAdsData, error) {
	var resp p2pAPIResponse[P2PMyAdsData]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/books/my_ads_list", nil, arg, &resp)
}

// GetP2PAdList retrieves the public P2P advertisement list for a given asset/fiat pair and trade side.
func (e *Exchange) GetP2PAdList(ctx context.Context, arg *GetP2PAdsListRequest) ([]*P2PAdListItem, error) {
	if arg.Asset.IsEmpty() {
		return nil, fmt.Errorf("%w P2P asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FiatUnit == "" {
		return nil, errP2PFiatUnitRequired
	}
	if arg.TradeType == "" {
		return nil, errP2PTradeTypeRequired
	}
	var resp p2pAPIResponse[[]*P2PAdListItem]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/books/ads_list", nil, arg, &resp)
}

// GetP2PChatHistory retrieves the chat history for a P2P order.
func (e *Exchange) GetP2PChatHistory(ctx context.Context, transactionID, counterparty int64) ([]*P2PChatMessage, error) {
	if transactionID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("txid", strconv.FormatInt(transactionID, 10))
	if counterparty > 0 {
		params.Set("counterparty", strconv.FormatInt(counterparty, 10))
	}
	var resp p2pAPIResponse[[]*P2PChatMessage]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "p2p/merchant/chat/history", params, nil, &resp)
}

// SendP2PChatMessage sends a chat message for a P2P order.
// Type: 0=text (default), 1=file; for file type pass the file_key from UploadP2PChatFile as Message.
func (e *Exchange) SendP2PChatMessage(ctx context.Context, arg *SendP2PChatMessageRequest) (*P2PSendMessageResult, error) {
	if arg.TransactionID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Message == "" {
		return nil, errP2PMessageRequired
	}
	var resp p2pAPIResponse[P2PSendMessageResult]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/chat/send_chat_message", nil, arg, &resp)
}

// UploadP2PChatFile uploads a file for use in P2P chat.
func (e *Exchange) UploadP2PChatFile(ctx context.Context, arg *UploadP2PChatFileRequest) (*P2PUploadFileResult, error) {
	if arg.ImageContentType == "" {
		return nil, errP2PImageTypeRequired
	}
	if arg.Base64Img == "" {
		return nil, errP2PImageDataRequired
	}
	var resp p2pAPIResponse[P2PUploadFileResult]
	return &resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "p2p/merchant/chat/upload_chat_file", nil, arg, &resp)
}
