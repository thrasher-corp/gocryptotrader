package gateio

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// p2pAPIResponse is a generic wrapper for all P2P API responses.
type p2pAPIResponse[T any] struct {
	Timestamp types.Number `json:"timestamp"`
	Method    string       `json:"method"`
	Code      int64        `json:"code"`
	Message   string       `json:"message"`
	Data      T            `json:"data"`
	Version   string       `json:"version"`
}

// Error implements the error chack interface to be used by the SendAuthenticatedHTTPRequest method
func (p *p2pAPIResponse[T]) Error() error {
	if p.Code != 0 {
		return fmt.Errorf("error code: %d message: %s", p.Code, p.Message)
	}
	return nil
}

// P2pMerchantInfo holds P2P merchant account information.
type P2pMerchantInfo struct {
	IsSelf                    bool         `json:"is_self"`
	UserLined                 string       `json:"user_lined"`
	CounterpartiesNum         int64        `json:"counterparties_num"`
	EmailVerified             string       `json:"email_verified"`
	Verified                  string       `json:"verified"`
	NickName                  string       `json:"nick_name"`
	BizUID                    string       `json:"biz_uid"`
	HaveTraded                int64        `json:"have_traded"`
	CompleteTransactions      string       `json:"complete_transactions"`
	PaidTransactions          string       `json:"paid_transactions"`
	AcceptableTransactions    string       `json:"acceptable_transactions"`
	CompleteTransactionsMonth string       `json:"complete_transactions_month"`
	CompleteRateMonth         string       `json:"complete_rate_month"`
	OrdersBuySaleMonth        int64        `json:"orders_buy_sale_month"`
	IsFixed                   int64        `json:"is_fixed"`
	FirstTradeDays            int64        `json:"first_trade_days"`
	TrendRegression           int64        `json:"trend_regression"`
	PayType                   string       `json:"pay_type"`
	PayMethod                 string       `json:"pay_method"`
	AccountStatus             int64        `json:"account_status"`
	TransactionsMonth         types.Number `json:"transactions_month"`
	TransactionsAll           types.Number `json:"transactions_all"`
	TradeVersatile            bool         `json:"trade_versatile"`
}

// GetCounterpartyInfoRequest holds the request parameters for getting counterparty info.
type GetCounterpartyInfoRequest struct {
	BizUID string `json:"biz_uid"`
}

// P2pCounterpartyInfo holds P2P counterparty user information.
type P2pCounterpartyInfo struct {
	UserLined                 string `json:"user_lined"`
	EmailVerified             string `json:"email_verified"`
	Verified                  string `json:"verified"`
	HasPhone                  string `json:"has_phone"`
	UserName                  string `json:"user_name"`
	CompleteTransactions      string `json:"complete_transactions"`
	PaidTransactions          string `json:"paid_transactions"`
	AcceptableTransactions    string `json:"acceptable_transactions"`
	CompleteTransactionsMonth string `json:"complete_transactions_month"`
	CancelledUserTimeMonth    string `json:"cancelled_user_time_month"`
	CompleteRateMonth         string `json:"complete_rate_month"`
}

// GetP2PPaymentMethodsRequest holds the request parameters for getting payment methods.
type GetP2PPaymentMethodsRequest struct {
	Fiat string `json:"fiat,omitempty"`
}

// P2pPaymentMethodGroup holds a group of payment methods of the same type.
type P2pPaymentMethodGroup struct {
	PayType string              `json:"pay_type"`
	PayName string              `json:"pay_name"`
	IDs     []uint64            `json:"ids"`
	List    []*P2pPaymentMethod `json:"list"`
}

// P2pPaymentMethod holds a single bound payment method account.
type P2pPaymentMethod struct {
	UID         uint64 `json:"uid"`
	Bank        string `json:"bank"`
	BankName    string `json:"bankname"`
	BankBranch  string `json:"bankbranch"`
	BankAddress string `json:"bankaddress"`
	HoldNote    string `json:"hold_note"`
	HoldSign    uint64 `json:"hold_sign"`
	RealName    string `json:"real_name"`
	AccountNo   string `json:"account_no"`
	TitleKey    string `json:"title_key"`
	MemoAll     string `json:"memo_all"`
	NoteMask    string `json:"note_mask"`
	Memo        string `json:"memo"`
	TradeType   string `json:"trade_type"`
}

// GetP2POrdersRequest holds request parameters for getting pending P2P orders.
type GetP2POrdersRequest struct {
	StatusList []uint64 `json:"status_list,omitempty"`
	Page       uint64   `json:"page,omitempty"`
	Limit      uint64   `json:"limit,omitempty"`
}

// GetP2PHistoricalOrdersRequest holds request parameters for getting historical P2P orders.
type GetP2PHistoricalOrdersRequest struct {
	StatusList []int64 `json:"status_list,omitempty"`
	Page       uint64  `json:"page,omitempty"`
	Limit      uint64  `json:"limit,omitempty"`
	From       int64   `json:"from,omitempty"`
	To         int64   `json:"to,omitempty"`
}

// P2pOrdersData wraps the list of P2P orders returned in a response.
type P2pOrdersData struct {
	List []*P2pOrderItem `json:"list"`
}

// P2pOrderItem holds a single P2P order item.
type P2pOrderItem struct {
	TransactionID uint64        `json:"txid"`
	Type          string        `json:"type"`
	Currency      currency.Code `json:"currency"`
	Fiat          string        `json:"fiat"`
	Amount        types.Number  `json:"amount"`
	Total         types.Number  `json:"total"`
	Price         types.Number  `json:"price"`
	Status        int64         `json:"status"`
	PaymentMethod string        `json:"pay_method"`
	CreateTime    types.Time    `json:"create_time"`
}

// GetP2POrderDetailsRequest holds request parameters for querying P2P order details.
type GetP2POrderDetailsRequest struct {
	TransactionID uint64 `json:"txid"`
	Channel       string `json:"channel,omitempty"`
}

// P2pOrderDetail holds detailed P2P order information.
type P2pOrderDetail struct {
	IsSelf        uint64       `json:"is_self"`
	TransactionID uint64       `json:"txid"`
	OrderID       uint64       `json:"orderid"`
	BizID         uint64       `json:"bizid"`
	LastPayTime   types.Number `json:"last_pay_time"`
	Type          string       `json:"type"`
	Status        int64        `json:"status"`
	Amount        types.Number `json:"amount"`
	TotalFiat     string       `json:"totalfat"`
	ReasonDesc    string       `json:"reason_desc"`
	ReasonNote    int64        `json:"reason_note"`
	DisputeTime   types.Time   `json:"dispute_time"`
	TradeType     string       `json:"trade_type"`
	TradeNote     string       `json:"trade_note"`
	BankName      string       `json:"bankname"`
	BankBranch    string       `json:"bankbranch"`
}

// ConfirmP2PPaymentRequest holds request parameters for confirming P2P payment.
type ConfirmP2PPaymentRequest struct {
	TransactionID string `json:"txid"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

// ConfirmP2PReceiptRequest holds request parameters for confirming P2P receipt.
type ConfirmP2PReceiptRequest struct {
	TransactionID string `json:"txid"`
}

// CancelP2POrderRequest holds request parameters for cancelling a P2P order.
// ReasonID values: 1=no longer want to buy, 2=cannot reach seller, 3=will not pay,
// 4=seller account not real, 5=price mismatch, 6=mutually agreed cancel,
// 7=poor communication, 8=other, 9=seller cannot release with refund,
// 10=terms not met, 11=seller payout risk-controlled.
type CancelP2POrderRequest struct {
	TransactionID string `json:"txid"`
	ReasonID      int64  `json:"reason_id,omitempty"`
	ReasonMemo    string `json:"reason_memo,omitempty"`
}

// PublishP2PAdRequest holds request parameters for publishing a P2P advertisement.
// PriceType: 1=floating (premium-based), 2=fixed.
type PublishP2PAdRequest struct {
	Asset             currency.Code `json:"asset"`
	FiatUnit          string        `json:"fiat_unit"`
	TradeType         string        `json:"trade_type"`
	PayIDs            string        `json:"pay_ids"`
	PriceType         int64         `json:"price_type"`
	PremiumRatio      string        `json:"premium_ratio,omitempty"`
	FixedPrice        string        `json:"fixed_price,omitempty"`
	MaxAmount         float64       `json:"max_amount,string"`
	MinAmount         float64       `json:"min_amount,string"`
	Remarks           string        `json:"remarks,omitempty"`
	AutoReply         string        `json:"auto_reply,omitempty"`
	RegDaysLimit      int64         `json:"reg_days_limit,omitempty"`
	KycLimit          int64         `json:"kyc_limit,omitempty"`
	CounterpartyLimit int64         `json:"counterparty_limit,omitempty"`
	NewKyc            int64         `json:"new_kyc,omitempty"`
	HasUnfinished     int64         `json:"has_unfinished,omitempty"`
	AdvOrderNumLimit  int64         `json:"adv_ordernum_limit,omitempty"`
	IsTrusted         int64         `json:"is_trusted,omitempty"`
	TradeAmount       string        `json:"trade_amount,omitempty"`
	TradeDays         int64         `json:"trade_days,omitempty"`
	MaxCompletedLimit int64         `json:"max_completed_limit,omitempty"`
	CompleteRateLimit int64         `json:"complete_rate_limit,omitempty"`
	IsHedge           int64         `json:"is_hedge,omitempty"`
}

// UpdateP2PAdStatusRequest holds request parameters for updating P2P ad status.
// AdvStatus: 1=listed, 3=delisted, 4=closed.
type UpdateP2PAdStatusRequest struct {
	AdvNo     int64 `json:"adv_no"`
	AdvStatus int64 `json:"adv_status"`
}

// P2pUpdateAdStatusResult holds the result of updating a P2P ad status.
type P2pUpdateAdStatusResult struct {
	Status int64 `json:"status"`
}

// GetP2PAdDetailsRequest holds request parameters for querying P2P ad details.
type GetP2PAdDetailsRequest struct {
	AdvNo string `json:"adv_no"`
}

// P2pAdDetail holds detailed P2P advertisement information.
type P2pAdDetail struct {
	Rate              string       `json:"rate"`
	Type              string       `json:"type"`
	Amount            types.Number `json:"amount"`
	MinAmount         types.Number `json:"min_amount"`
	MaxAmount         types.Number `json:"max_amount"`
	PayBest           int64        `json:"pay_best"`
	PayWeight         int64        `json:"pay_weight"`
	TradeType         string       `json:"trade_type"`
	TradeNote         string       `json:"trade_note"`
	NodeReply         string       `json:"node_reply"`
	Status            string       `json:"status"`
	AdvNo             types.Number `json:"adv_no"`
	LockedAmount      types.Number `json:"locked_amount"`
	CurrencyType      string       `json:"currency_type"`
	CreatedAt         types.Time   `json:"created_at"`
	TradeAmount       string       `json:"trade_amount"`
	TradeTypeID       int64        `json:"trade_type_id"`
	NoteState         int64        `json:"note_state"`
	OriginRate        string       `json:"origin_rate"`
	MaxCompletedLimit int64        `json:"max_completed_limit"`
	RegLimit          int64        `json:"reg_limit"`
	RegionAsk         int64        `json:"region_ask"`
	MaxNumOrdersLimit int64        `json:"max_num_orders_limit"`
	IsHedge           int64        `json:"is_hedge"`
	HidePayment       int64        `json:"hide_payment"`
}

// GetMyP2PAdsRequest holds request parameters for getting the current user's P2P ads.
type GetMyP2PAdsRequest struct {
	Asset     currency.Code `json:"asset,omitempty"`
	FiatUnit  string        `json:"fiat_unit,omitempty"`
	TradeType string        `json:"trade_type,omitempty"`
}

// P2pMyAdsData wraps the list of the user's own P2P ads.
type P2pMyAdsData struct {
	Lists []*P2pMyAdItem `json:"lists"`
}

// P2pMyAdItem holds a single item from the user's P2P ad list.
type P2pMyAdItem struct {
	Type              string        `json:"type"`
	Price             types.Number  `json:"price"`
	Rate              string        `json:"rate"`
	Status            string        `json:"status"`
	AdvNo             string        `json:"adv_no"`
	BuyTypeNum        types.Number  `json:"buy_type_num"`
	FiatUnit          string        `json:"fiat_unit"`
	Asset             currency.Code `json:"asset"`
	RegTimeLimit      int64         `json:"reg_time_limit"`
	NewKYC            int64         `json:"new_kyc"`
	PriceLimit        int64         `json:"price_limit"`
	CompleteRateLimit int64         `json:"complete_rate_limit"`
	Timestamp         types.Time    `json:"timestamp"`
	IsBadge           int64         `json:"is_badge"`
}

// GetP2PAdsListRequest holds request parameters for getting the public P2P ads list.
type GetP2PAdsListRequest struct {
	Asset     currency.Code `json:"asset"`
	FiatUnit  string        `json:"fiat_unit"`
	TradeType string        `json:"trade_type"`
}

// P2pAdListItem holds a single item from the public P2P ads list.
type P2pAdListItem struct {
	Index                int64         `json:"index"`
	Asset                currency.Code `json:"asset"`
	FiatUnit             string        `json:"fiat_unit"`
	Price                types.Number  `json:"price"`
	MaxSingleTransAmount string        `json:"max_single_trans_amount"`
	LibName              string        `json:"lib_name"`
	AdvertizementNo      uint64        `json:"adv_no"`
}

// P2pChatMessage holds a single P2P chat message.
type P2pChatMessage struct {
	Type        int64              `json:"type"`
	Time        types.Time         `json:"time"`
	Message     string             `json:"message"`
	FileKey     string             `json:"file_key"`
	FileURL     string             `json:"file_url"`
	FromUID     uint64             `json:"from_uid"`
	FromUIDInfo *P2pChatSenderInfo `json:"from_uid_info"`
}

// P2pChatSenderInfo holds basic info about the sender of a P2P chat message.
type P2pChatSenderInfo struct {
	UserName string `json:"user_name"`
	BizUID   string `json:"biz_uid"`
}

// SendP2PChatMessageRequest holds request parameters for sending a P2P chat message.
// Type: 0=text (default), 1=file (image or video).
type SendP2PChatMessageRequest struct {
	TransactionID uint64 `json:"txid"`
	Type          int64  `json:"type,omitempty"`
	Message       string `json:"message"`
}

// P2pSendMessageResult holds the result of sending a P2P chat message.
type P2pSendMessageResult struct {
	SendTime types.Time `json:"SendTM"`
}

// UploadP2PChatFileRequest holds request parameters for uploading a P2P chat file.
type UploadP2PChatFileRequest struct {
	ImageContentType string `json:"image_content_type"` // ImageContentType supports: image/png, image/jpg, image/jpeg, video/mp4. Max 20 MB.
	Base64Img        string `json:"base64_img"`
}

// P2pUploadFileResult holds the result of uploading a P2P chat file.
type P2pUploadFileResult struct {
	FileKey string `json:"file_key"`
}
