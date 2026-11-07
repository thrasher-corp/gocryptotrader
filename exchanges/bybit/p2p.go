package bybit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// P2P endpoint validation errors
var (
	errAdvertisementIDRequired = errors.New("advertisement ID required")
	errPriceTypeRequired       = errors.New("price type required")
	errPaymentIDsRequired      = errors.New("payment method IDs required")
	errPaymentPeriodRequired   = errors.New("payment period required")
	errItemTypeRequired        = errors.New("advertisement item type required")
	errActionTypeRequired      = errors.New("action type required")
	errPaymentTypeRequired     = errors.New("payment type required")
	errPaymentIDRequired       = errors.New("payment method ID required")
	errExamineResultRequired   = errors.New("examine result required")
	errRejectReasonRequired    = errors.New("reject reason required")
	errRejectProofsRequired    = errors.New("reject proofs required")
	errOriginalUIDRequired     = errors.New("counterparty UID required")
	errUserMaskIDRequired      = errors.New("user mask ID required")
	errSessionIDRequired       = errors.New("session ID required")
	errChatMessageRequired     = errors.New("chat message required")
	errChatContentTypeRequired = errors.New("chat content type required")
	errUploadFileNameRequired  = errors.New("upload file name required")
	errUploadFileEmpty         = errors.New("upload file is empty")
)

// GetP2POnlineAds returns the advertisements listed on the P2P market for a token, currency and side (Get Ads)
func (e *Exchange) GetP2POnlineAds(ctx context.Context, r *P2POnlineAdsRequest) (*P2POnlineAdsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TokenID.IsEmpty() {
		return nil, fmt.Errorf("%w: token ID", currency.ErrCurrencyCodeEmpty)
	}
	if r.CurrencyID.IsEmpty() {
		return nil, fmt.Errorf("%w: currency ID", currency.ErrCurrencyCodeEmpty)
	}
	if r.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	body := make(map[string]any)
	body["tokenId"] = r.TokenID.Upper().String()
	body["currencyId"] = r.CurrencyID.Upper().String()
	body["side"] = r.Side
	if r.Page != 0 {
		body["page"] = strconv.FormatUint(r.Page, 10)
	}
	if r.Size != 0 {
		body["size"] = strconv.FormatUint(r.Size, 10)
	}
	var resp *P2POnlineAdsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/online", nil, body, &resp, getP2POnlineAdsEPL)
}

// PostP2PAd posts a P2P advertisement (Post Ad)
func (e *Exchange) PostP2PAd(ctx context.Context, r *P2PPostAdRequest) (*P2PPostAdResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TokenID.IsEmpty() {
		return nil, fmt.Errorf("%w: token ID", currency.ErrCurrencyCodeEmpty)
	}
	if r.CurrencyID.IsEmpty() {
		return nil, fmt.Errorf("%w: currency ID", currency.ErrCurrencyCodeEmpty)
	}
	if r.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if r.PriceType == "" {
		return nil, errPriceTypeRequired
	}
	if r.MinAmount <= 0 {
		return nil, fmt.Errorf("%w: min amount", order.ErrAmountIsInvalid)
	}
	if r.MaxAmount <= 0 {
		return nil, fmt.Errorf("%w: max amount", order.ErrAmountIsInvalid)
	}
	if len(r.PaymentIDs) == 0 {
		return nil, errPaymentIDsRequired
	}
	if r.Quantity <= 0 {
		return nil, fmt.Errorf("%w: quantity", order.ErrAmountIsInvalid)
	}
	if r.PaymentPeriod == 0 {
		return nil, errPaymentPeriodRequired
	}
	if r.ItemType == "" {
		return nil, errItemTypeRequired
	}
	body := make(map[string]any)
	body["tokenId"] = r.TokenID.Upper().String()
	body["currencyId"] = r.CurrencyID.Upper().String()
	body["side"] = r.Side
	body["priceType"] = r.PriceType
	body["premium"] = p2pOptionalDecimal(r.Premium)
	body["price"] = p2pOptionalDecimal(r.Price)
	body["minAmount"] = strconv.FormatFloat(r.MinAmount, 'f', -1, 64)
	body["maxAmount"] = strconv.FormatFloat(r.MaxAmount, 'f', -1, 64)
	body["remark"] = r.Remark
	body["tradingPreferenceSet"] = r.TradingPreferenceSet.payload()
	body["paymentIds"] = r.PaymentIDs
	body["quantity"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	body["paymentPeriod"] = strconv.FormatUint(r.PaymentPeriod, 10)
	body["itemType"] = r.ItemType
	var resp *P2PPostAdResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/create", nil, body, &resp, postP2PAdEPL)
}

// p2pOptionalDecimal formats a decimal Post Ad and Update / Relist Ad must send even when it does not apply, such as a
// fixed price ad's premium, as the docs do: empty when zero
func p2pOptionalDecimal(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// payload returns the trading preference set as Post Ad and Update / Relist Ad send it, with every value a string; a
// nil set sends an empty object
func (p *P2PTradingPreferenceSetRequest) payload() map[string]string {
	if p == nil {
		return map[string]string{}
	}
	prefs := make(map[string]string)
	prefs["hasUnPostAd"] = p2pFlag(p.HasUnPostAd)
	prefs["isKyc"] = p2pFlag(p.IsKYC)
	prefs["isEmail"] = p2pFlag(p.IsEmail)
	prefs["isMobile"] = p2pFlag(p.IsMobile)
	prefs["hasRegisterTime"] = p2pFlag(p.HasRegisterTime)
	prefs["registerTimeThreshold"] = strconv.FormatUint(p.RegisterTimeThreshold, 10)
	prefs["orderFinishNumberDay30"] = strconv.FormatUint(p.OrderFinishNumberDay30, 10)
	prefs["completeRateDay30"] = strconv.FormatFloat(p.CompleteRateDay30, 'f', -1, 64)
	prefs["nationalLimit"] = p.NationalLimit
	prefs["hasOrderFinishNumberDay30"] = p2pFlag(p.HasOrderFinishNumberDay30)
	prefs["hasCompleteRateDay30"] = p2pFlag(p.HasCompleteRateDay30)
	prefs["hasNationalLimit"] = p2pFlag(p.HasNationalLimit)
	return prefs
}

func p2pFlag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// RemoveP2PAd removes one of the user's P2P advertisements (Remove Ad)
func (e *Exchange) RemoveP2PAd(ctx context.Context, itemID string) error {
	if itemID == "" {
		return errAdvertisementIDRequired
	}
	body := make(map[string]any)
	body["itemId"] = itemID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/cancel", nil, body, nil, removeP2PAdEPL)
}

// UpdateP2PAd modifies or relists one of the user's P2P advertisements; an advertisement may be modified at most 10
// times in 5 minutes (Update / Relist Ad)
func (e *Exchange) UpdateP2PAd(ctx context.Context, r *P2PUpdateAdRequest) (*P2PUpdateAdResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ID == "" {
		return nil, errAdvertisementIDRequired
	}
	if r.PriceType == "" {
		return nil, errPriceTypeRequired
	}
	if r.MinAmount <= 0 {
		return nil, fmt.Errorf("%w: min amount", order.ErrAmountIsInvalid)
	}
	if r.MaxAmount <= 0 {
		return nil, fmt.Errorf("%w: max amount", order.ErrAmountIsInvalid)
	}
	if len(r.PaymentIDs) == 0 {
		return nil, errPaymentIDsRequired
	}
	if r.ActionType == "" {
		return nil, errActionTypeRequired
	}
	if r.Quantity <= 0 {
		return nil, fmt.Errorf("%w: quantity", order.ErrAmountIsInvalid)
	}
	if r.PaymentPeriod == 0 {
		return nil, errPaymentPeriodRequired
	}
	body := make(map[string]any)
	body["id"] = r.ID
	body["priceType"] = r.PriceType
	body["premium"] = p2pOptionalDecimal(r.Premium)
	body["price"] = p2pOptionalDecimal(r.Price)
	body["minAmount"] = strconv.FormatFloat(r.MinAmount, 'f', -1, 64)
	body["maxAmount"] = strconv.FormatFloat(r.MaxAmount, 'f', -1, 64)
	body["remark"] = r.Remark
	body["tradingPreferenceSet"] = r.TradingPreferenceSet.payload()
	body["paymentIds"] = r.PaymentIDs
	body["actionType"] = r.ActionType
	body["quantity"] = strconv.FormatFloat(r.Quantity, 'f', -1, 64)
	body["paymentPeriod"] = strconv.FormatUint(r.PaymentPeriod, 10)
	var resp *P2PUpdateAdResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/update", nil, body, &resp, updateP2PAdEPL)
}

// GetP2PMyAds returns the user's P2P advertisements (Get My Ads)
func (e *Exchange) GetP2PMyAds(ctx context.Context, r *P2PMyAdsRequest) (*P2PMyAdsResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.ItemID != "" {
		body["itemId"] = r.ItemID
	}
	if r.Status != "" {
		body["status"] = r.Status
	}
	if r.Side != "" {
		body["side"] = r.Side
	}
	if !r.TokenID.IsEmpty() {
		body["tokenId"] = r.TokenID.Upper().String()
	}
	if r.Page != 0 {
		body["page"] = strconv.FormatUint(r.Page, 10)
	}
	if r.Size != 0 {
		body["size"] = strconv.FormatUint(r.Size, 10)
	}
	if !r.CurrencyID.IsEmpty() {
		body["currencyId"] = r.CurrencyID.Upper().String()
	}
	var resp *P2PMyAdsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/personal/list", nil, body, &resp, getP2PMyAdsEPL)
}

// GetP2PMyAdDetails returns one of the user's P2P advertisements (Get My Ad Details)
func (e *Exchange) GetP2PMyAdDetails(ctx context.Context, itemID string) (*P2PMyAdDetailsResponse, error) {
	if itemID == "" {
		return nil, errAdvertisementIDRequired
	}
	body := make(map[string]any)
	body["itemId"] = itemID
	var resp *P2PMyAdDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/item/info", nil, body, &resp, getP2PMyAdDetailsEPL)
}

// GetAllP2POrders returns the user's P2P orders; the last 90 days by default, and up to 180 days back (Get All Orders)
func (e *Exchange) GetAllP2POrders(ctx context.Context, r *P2POrderListRequest) (*P2PAllOrdersResponse, error) {
	body, err := p2pOrderListBody(r)
	if err != nil {
		return nil, err
	}
	var resp *P2PAllOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/simplifyList", nil, body, &resp, getAllP2POrdersEPL)
}

// p2pOrderListBody builds the body Get All Orders and Get Pending Orders share. The time filters are documented only as
// strings, so they are sent as millisecond timestamps like the order times P2P returns
func p2pOrderListBody(r *P2POrderListRequest) (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Page == 0 {
		return nil, errPageIndexRequired
	}
	if r.Size == 0 {
		return nil, errPageLimitRequired
	}
	if !r.BeginTime.IsZero() && !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.BeginTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	body := make(map[string]any)
	body["page"] = r.Page
	body["size"] = r.Size
	if r.Status != 0 {
		body["status"] = r.Status
	}
	if !r.BeginTime.IsZero() {
		body["beginTime"] = strconv.FormatInt(r.BeginTime.UnixMilli(), 10)
	}
	if !r.EndTime.IsZero() {
		body["endTime"] = strconv.FormatInt(r.EndTime.UnixMilli(), 10)
	}
	if !r.TokenID.IsEmpty() {
		body["tokenId"] = r.TokenID.Upper().String()
	}
	if r.Side != "" {
		// The order lists take side as an integer, unlike the advertisement endpoints
		side, err := strconv.ParseUint(r.Side, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, r.Side)
		}
		body["side"] = side
	}
	return body, nil
}

// GetP2POrderDetail returns a P2P order's details (Get Order Detail)
func (e *Exchange) GetP2POrderDetail(ctx context.Context, orderID string) (*P2POrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	body := make(map[string]any)
	body["orderId"] = orderID
	var resp *P2POrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/info", nil, body, &resp, getP2POrderDetailEPL)
}

// GetP2PPendingOrders returns the user's pending P2P orders; the last 90 days by default, and up to 180 days back (Get
// Pending Orders)
func (e *Exchange) GetP2PPendingOrders(ctx context.Context, r *P2POrderListRequest) (*P2PPendingOrdersResponse, error) {
	body, err := p2pOrderListBody(r)
	if err != nil {
		return nil, err
	}
	var resp *P2PPendingOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/pending/simplifyList", nil, body, &resp, getP2PPendingOrdersEPL)
}

// MarkP2POrderAsPaid marks a P2P order as paid with one of the user's payment methods; the Balance payment method is
// not supported (Mark Order as Paid)
func (e *Exchange) MarkP2POrderAsPaid(ctx context.Context, r *P2PMarkOrderAsPaidRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if r.PaymentType == "" {
		return errPaymentTypeRequired
	}
	if r.PaymentID == "" {
		return errPaymentIDRequired
	}
	body := make(map[string]any)
	body["orderId"] = r.OrderID
	body["paymentType"] = r.PaymentType
	body["paymentId"] = r.PaymentID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/pay", nil, body, nil, markP2POrderAsPaidEPL)
}

// ReleaseP2PAssets releases a P2P order's digital assets to the buyer (Release Assets)
func (e *Exchange) ReleaseP2PAssets(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	body := make(map[string]any)
	body["orderId"] = orderID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/finish", nil, body, nil, releaseP2PAssetsEPL)
}

// ReviewP2PSellerCancelOrderApply accepts or rejects, as the buyer, a seller's pending request to cancel a P2P order
// (Review Seller Cancel Order Apply)
func (e *Exchange) ReviewP2PSellerCancelOrderApply(ctx context.Context, r *P2PReviewSellerCancelOrderApplyRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if r.ExamineResult == "" {
		return errExamineResultRequired
	}
	if r.ExamineResult == "REJECT" {
		if r.RejectReason == "" {
			return errRejectReasonRequired
		}
		if len(r.RejectProofs) == 0 {
			return errRejectProofsRequired
		}
	}
	body := make(map[string]any)
	body["orderId"] = r.OrderID
	body["examineResult"] = r.ExamineResult
	if r.RejectReason != "" {
		body["rejectReason"] = r.RejectReason
	}
	if len(r.RejectProofs) != 0 {
		body["rejectProofs"] = strings.Join(r.RejectProofs, ",")
	}
	if r.RejectRemark != "" {
		body["rejectRemark"] = r.RejectRemark
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/order/buyer/examine/sellerCancelOrderApply", nil, body, nil, reviewP2PSellerCancelOrderApplyEPL)
}

// GetP2PAccountInformation returns the user's P2P profile (Get Account Information)
func (e *Exchange) GetP2PAccountInformation(ctx context.Context) (*P2PUserInfoResponse, error) {
	var resp *P2PUserInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/user/personal/info", nil, nil, &resp, getP2PAccountInformationEPL)
}

// GetP2PCounterpartyUserInfo returns the P2P profile of an order's counterparty (Get Counterparty User Info)
func (e *Exchange) GetP2PCounterpartyUserInfo(ctx context.Context, originalUID, orderID string) (*P2PUserInfoResponse, error) {
	if originalUID == "" {
		return nil, errOriginalUIDRequired
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	body := make(map[string]any)
	body["originalUid"] = originalUID
	body["orderId"] = orderID
	var resp *P2PUserInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/user/order/personal/info", nil, body, &resp, getP2PCounterpartyUserInfoEPL)
}

// GetP2PUserPayment returns the payment methods added to the user's account (Get User Payment)
func (e *Exchange) GetP2PUserPayment(ctx context.Context) ([]P2PUserPaymentResponse, error) {
	var resp []P2PUserPaymentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/user/payment/list", nil, nil, &resp, getP2PUserPaymentEPL)
}

// GetP2PChatSessionList returns the user's P2P chat sessions (Get Chat Session List)
func (e *Exchange) GetP2PChatSessionList(ctx context.Context, r *P2PChatSessionListRequest) (*P2PChatSessionListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Size == 0 {
		return nil, errPageLimitRequired
	}
	// lastId and readStatus are required and zero is meaningful for both: the first page, and unread sessions
	body := make(map[string]any)
	body["lastId"] = r.LastID
	body["size"] = r.Size
	body["readStatus"] = r.ReadStatus
	if r.Type != "" {
		body["type"] = r.Type
	}
	var resp *P2PChatSessionListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/chat/session/list_v1", nil, body, &resp, getP2PChatSessionListEPL)
}

// GetP2PChatSessionID returns the session ID of the chat with the counterparty whose user mask ID an order detail's
// TargetUserMaskID holds (Get Session ID)
func (e *Exchange) GetP2PChatSessionID(ctx context.Context, userMaskID string) (*P2PChatSessionIDResponse, error) {
	if userMaskID == "" {
		return nil, errUserMaskIDRequired
	}
	body := make(map[string]any)
	body["userMaskId"] = userMaskID
	var resp *P2PChatSessionIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/chat/session/getSessionId", nil, body, &resp, getP2PChatSessionIDEPL)
}

// UploadP2PChatFile uploads a jpg, jpeg, png, pdf or mp4 file for P2P chat; Send Chat Message then sends the returned
// URL (Upload Chat File)
func (e *Exchange) UploadP2PChatFile(ctx context.Context, fileName string, content []byte) (*P2PChatFileResponse, error) {
	body, contentType, err := p2pChatFileBody(fileName, content)
	if err != nil {
		return nil, err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return nil, err
	}
	var resp *P2PChatFileResponse
	return resp, e.sendRequest(ctx, uploadP2PChatFileEPL, request.AuthenticatedRequest, &resp, func() (*request.Item, error) {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		// The multipart body is signed as a JSON body would be, since SendAuthHTTPRequest only sends JSON
		sign := getSign(timestamp+creds.Key+defaultRecvWindow+string(body), creds.Secret)
		// Not recorded: the mock server cannot match a multipart body, so a recording could never be replayed
		return &request.Item{
			Method: http.MethodPost,
			Path:   endpoint + "/v5/p2p/oss/upload_file",
			Headers: map[string]string{
				"X-BAPI-API-KEY":     creds.Key,
				"X-BAPI-TIMESTAMP":   timestamp,
				"X-BAPI-RECV-WINDOW": defaultRecvWindow,
				"X-BAPI-SIGN":        sign,
				"Content-Type":       contentType,
			},
			Body:                   bytes.NewReader(body),
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// p2pChatFileBody builds the multipart body Upload Chat File sends, with the file in its upload_file field, and
// returns it with its content type
func p2pChatFileBody(fileName string, content []byte) (body []byte, contentType string, err error) {
	if fileName == "" {
		return nil, "", errUploadFileNameRequired
	}
	if len(content) == 0 {
		return nil, "", errUploadFileEmpty
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", multipart.FileContentDisposition("upload_file", fileName))
	header.Set("Content-Type", http.DetectContentType(content))
	// The writers below write into buf, which never fails, so the errors are only carried through
	part, err := w.CreatePart(header)
	if err == nil {
		_, err = part.Write(content)
	}
	if err == nil {
		err = w.Close()
	}
	return buf.Bytes(), w.FormDataContentType(), err
}

// SendP2PChatMessage sends a text message, or the URL of a file from UploadP2PChatFile, in a P2P order's chat (Send
// Chat Message)
func (e *Exchange) SendP2PChatMessage(ctx context.Context, r *P2PSendChatMessageRequest) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}
	if r.Message == "" {
		return errChatMessageRequired
	}
	if r.ContentType == "" {
		return errChatContentTypeRequired
	}
	if r.SessionID == "" {
		return errSessionIDRequired
	}
	if r.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	body := make(map[string]any)
	body["message"] = r.Message
	body["contentType"] = r.ContentType
	body["sessionId"] = r.SessionID
	body["orderId"] = r.OrderID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/chat/message/send_v1", nil, body, nil, sendP2PChatMessageEPL)
}

// GetP2PChatMessageList returns a P2P chat session's messages, newest first (Get Chat Message List)
func (e *Exchange) GetP2PChatMessageList(ctx context.Context, r *P2PChatMessageListRequest) (*P2PChatMessageListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Limit == 0 {
		return nil, errPageLimitRequired
	}
	if r.SessionID == "" {
		return nil, errSessionIDRequired
	}
	// lastId is required and zero is meaningful: the latest messages
	body := make(map[string]any)
	body["lastId"] = r.LastID
	body["limit"] = r.Limit
	body["sessionId"] = r.SessionID
	var resp *P2PChatMessageListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/p2p/chat/message/listpage_v1", nil, body, &resp, getP2PChatMessageListEPL)
}

// MessageContent decodes the JSON document a chat message carries
func (m *P2PChatMessage) MessageContent() (*P2PChatMessageContent, error) {
	var content *P2PChatMessageContent
	if err := json.Unmarshal([]byte(m.Message), &content); err != nil {
		return nil, fmt.Errorf("error decoding P2P chat message %s content: %w", m.ID, err)
	}
	if content == nil {
		return nil, fmt.Errorf("%w: P2P chat message %s has no content", common.ErrNoResponse, m.ID)
	}
	return content, nil
}
