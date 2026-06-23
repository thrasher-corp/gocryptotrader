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

// Error implements the error check interface to be used by the SendAuthenticatedHTTPRequest method
func (p *p2pAPIResponse[T]) Error() error {
	if p.Code != 0 {
		return fmt.Errorf("error code: %d message: %s", p.Code, p.Message)
	}
	return nil
}

// P2PMerchantInfo holds P2P merchant account information.
type P2PMerchantInfo struct {
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

// P2PCounterpartyInfo holds P2P counterparty user information.
type P2PCounterpartyInfo struct {
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
	UserTimest                string `json:"user_timest"`
	AcceptedTransactions      string `json:"accepted_transactions"`
	TransactionsUsedTime      string `json:"transactions_used_time"`
	CancelledUsedTimeMonth    string `json:"cancelled_used_time_month"`
	IsFollow                  uint64 `json:"is_follow"`
	HaveTraded                uint64 `json:"have_traded"`
	BizUID                    string `json:"biz_uid"`
	RegistrationDays          uint64 `json:"registration_days"`
	FirstTradeDays            uint64 `json:"first_trade_days"`
	TradeVersatile            bool   `json:"trade_versatile"`
}

// GetP2PPaymentMethodsRequest holds the request parameters for getting payment methods.
type GetP2PPaymentMethodsRequest struct {
	Fiat string `json:"fiat,omitempty"`
}

// P2PPaymentMethodGroup holds a group of payment methods of the same type.
type P2PPaymentMethodGroup struct {
	PayType string              `json:"pay_type"`
	PayName string              `json:"pay_name"`
	IDs     []uint64            `json:"ids"`
	List    []*P2PPaymentMethod `json:"list"`
}

// P2PPaymentMethod holds a single bound payment method account.
type P2PPaymentMethod struct {
	UID                   uint64 `json:"uid"`
	ID                    string `json:"id"`
	BankID                string `json:"bankid"`
	BankName              string `json:"bankname"`
	BankBranch            string `json:"bankbranch"`
	BankAddress           string `json:"bankaddr"`
	BankCity              string `json:"bankcity"`
	BankProvince          string `json:"bankprov"`
	BankDescription       string `json:"bankdesc"`
	RealName              string `json:"real_name"`
	AccountDescription    string `json:"account_des"`
	BankHolderUID         string `json:"hold_uid"`
	BankHoderUsername     string `json:"hold_username"`
	PaymentMethodType     string `json:"pay_type"`
	PaymentMethodFileLink string `json:"file"`
	PaymentMethodFileKey  string `json:"file_key"`
	Account               string `json:"account"`
	Memo                  string `json:"memo"`
	Code                  string `json:"code"`
	MemoExtended          string `json:"memo_ext"`
	TradeTips             string `json:"trade_tips"`
	Version               string `json:"version"`
	Nickname              int    `json:"nickname"`
}

// SetMerchantWorkHoursRequest represents request paramters to sent merchant working hour
type SetMerchantWorkHoursRequest struct {
	WorkStatus int64  `json:"work_status"`
	CycleType  string `json:"cycle_type"`
	DayOfWeek  string `json:"day_of_week"`
	TimeZone   string `json:"time_zone"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

// WorkStatusResponse represents a response payload after setting working time
type WorkStatusResponse struct {
	WorkStatus int64 `json:"work_status"`
}

// PendingP2POrderRequest represents a p2p order request parameter
type PendingP2POrderRequest struct {
	CryptoCurrency currency.Code `json:"crypto_currency"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	OrderTab       string        `json:"order_tab"`
	SelectType     string        `json:"select_type"`
	Status         string        `json:"status"`
	TransatinoID   uint64        `json:"txid"`
	StartTime      uint64        `json:"start_time"`
	EndTime        uint64        `json:"end_time"`
}

// P2POrderList represents a P2P transactions detail list
type P2POrderList struct {
	List      []*P2POrderInfo `json:"list"`
	TransTime []struct {
		OdTime int `json:"od_time"`
	} `json:"trans_time"`
	Count       int `json:"count"`
	ExportedNum int `json:"exported_num"`
}

// P2POrderInfo represents a P2P order detail instance
type P2POrderInfo struct {
	TypeBuy        int64                `json:"type_buy"`
	Timest         string               `json:"timest"`
	TimestExpire   string               `json:"timest_expire"`
	Timestamp      types.Time           `json:"timestamp"`
	Rate           types.Number         `json:"rate"`
	Amount         types.Number         `json:"amount"`
	Total          types.Number         `json:"total"`
	TransactionID  uint64               `json:"txid"`
	Status         string               `json:"status"`
	ItsRealname    string               `json:"its_realname"`
	ItsUID         string               `json:"its_uid"`
	ItsNick        string               `json:"its_nick"`
	SellerRealname string               `json:"seller_realname"`
	BuyerRealname  string               `json:"buyer_realname"`
	Cancelable     int64                `json:"cancelable"`
	CurrencyType   string               `json:"currency_type"`
	WantType       string               `json:"want_type"`
	HidePayment    int64                `json:"hide_payment"`
	SelPaytype     string               `json:"sel_paytype"`
	CountdownTime  int64                `json:"cd_time"`
	OrderType      int64                `json:"order_type"`
	OrderTag       []string             `json:"order_tag"`
	ConvertInfo    *P2PConvertInfo      `json:"convert_info"`
	PayOthers      []OtherPaymentMethod `json:"pay_others"`
	IsSelf         uint64               `json:"is_self"`
	BizID          uint64               `json:"bizid"`
	LastPayTime    types.Number         `json:"last_pay_time"`
	Type           string               `json:"type"`
	TotalFiat      string               `json:"totalfat"`
	DisputeTime    types.Time           `json:"dispute_time"`
	TradeType      string               `json:"trade_type"`
	TradeNote      string               `json:"trade_note"`
	BankName       string               `json:"bankname"`
	BankBranch     string               `json:"bankbranch"`
}

// P2POrderDetail represents a P2P order detail
type P2POrderDetail struct {
	IsSell                int64                `json:"is_sell"`
	Txid                  int64                `json:"txid"`
	Orderid               int64                `json:"orderid"`
	Timest                int64                `json:"timest"`
	LastPayTime           int64                `json:"last_pay_time"`
	RemainPayTime         int64                `json:"remain_pay_time"`
	CurrencyType          string               `json:"currency_type"`
	WantType              string               `json:"want_type"`
	Symbol                currency.Pair        `json:"symbol"`
	Rate                  types.Number         `json:"rate"`
	Amount                types.Number         `json:"amount"`
	Total                 types.Number         `json:"total"`
	Status                string               `json:"status"`
	ReasonID              string               `json:"reason_id"`
	ReasonDesc            string               `json:"reason_desc"`
	CancelTime            types.Time           `json:"cancel_time"`
	InAppeal              int64                `json:"in_appeal"`
	DisputeTime           types.Number         `json:"dispute_time"`
	Cancelable            int64                `json:"cancelable"`
	HidePayment           int64                `json:"hide_payment"`
	TradeTips             string               `json:"trade_tips"`
	ShowBank              string               `json:"show_bank"`
	BankName              string               `json:"bankname"`
	BankBranch            string               `json:"bankbranch"`
	BankID                string               `json:"bankid"`
	BankHolderRealname    string               `json:"bank_holder_realname"`
	ShowAlipayDetail      string               `json:"show_ali"`
	Aliname               string               `json:"aliname"`
	IsAlicode             int64                `json:"is_alicode"`
	ShowWechat            string               `json:"show_wechat"`
	Wename                string               `json:"wename"`
	ShowOthers            string               `json:"show_others"`
	PayOthers             []OtherPaymentMethod `json:"pay_others"`
	SelPaytype            string               `json:"sel_paytype"`
	ItsUID                string               `json:"its_uid"`
	ItsNickname           string               `json:"its_nickname"`
	ItsRealname           string               `json:"its_realname"`
	HaveTraded            int64                `json:"have_traded"`
	AppealAllowCancel     int64                `json:"appeal_allow_cancel"`
	AppealVerdictHasOpen  string               `json:"appeal_verdict_has_open"`
	ImUnread              int64                `json:"im_unread"`
	PaymentVoucherURL     []string             `json:"payment_voucher_url"`
	TimestPaid            int64                `json:"timest_paid"`
	OwnRealname           string               `json:"own_realname"`
	OrderType             int64                `json:"order_type"`
	IsShowReceive         int64                `json:"is_show_receive"`
	ShowSellerContactInfo bool                 `json:"show_seller_contact_info"`
	SupportedPayTypes     []string             `json:"supported_pay_types"`
}

// OtherPaymentMethod represents other payment methods detail
type OtherPaymentMethod struct {
	ID                         string `json:"id"`
	PaymentAccountDescriptions string `json:"account_des"`
	PayType                    string `json:"pay_type"`
	PayName                    string `json:"pay_name"`
	Account                    string `json:"account"`
	Memo                       string `json:"memo"`
	TradeTips                  string `json:"trade_tips"`
}

// P2PConvertInfo represents a P2P order transction convert info
type P2PConvertInfo struct {
	ConvertType         string       `json:"convert_type"`
	ConvertStatus       string       `json:"convert_status"`
	ExpectedPriceRate   types.Number `json:"pre_rate"`
	ExecutionRate       types.Number `json:"rate"`
	ExpectedFiatPrice   types.Number `json:"pre_fiat_rate"`
	FiatRate            types.Number `json:"fiat_rate"`
	Amount              types.Number `json:"amount"`
	SwapAmount          types.Number `json:"convert_amount"`
	SlippageCalculation types.Number `json:"slippage"`
	Status              string       `json:"status"`
}

// P2PCompletedOrderRequest holds request parameters to retrieve completed p2p orders
type P2PCompletedOrderRequest struct {
	CryptoCurrency currency.Code `json:"crypto_currency"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	SelectType     string        `json:"select_type"`
	Status         string        `json:"status"`
	TransactionID  int64         `json:"txid"`
	StartTime      int64         `json:"start_time"`
	EndTime        int64         `json:"end_time"`
	QueryDispute   int64         `json:"query_dispute"`
	Page           int64         `json:"page"`
	PerPage        int64         `json:"per_page"`
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

// P2POrdersData wraps the list of P2P orders returned in a response.
type P2POrdersData struct {
	List []*P2POrderItem `json:"list"`
}

// P2POrderItem holds a single P2P order item.
type P2POrderItem struct {
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

// P2PUpdateAdStatusResult holds the result of updating a P2P ad status.
type P2PUpdateAdStatusResult struct {
	Status int64 `json:"status"`
}

// GetP2PAdDetailsRequest holds request parameters for querying P2P ad details.
type GetP2PAdDetailsRequest struct {
	AdvNo string `json:"adv_no"`
}

// P2PAdDetail holds detailed P2P advertisement information.
type P2PAdDetail struct {
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
	Asset     currency.Code `json:"asset"`
	FiatUnit  string        `json:"fiat_unit,omitempty"`
	TradeType string        `json:"trade_type,omitempty"`
}

// P2PMyAdsData wraps the list of the user's own P2P ads.
type P2PMyAdsData struct {
	Lists []*P2PMyAdItem `json:"lists"`
}

// P2PMyAdItem holds a single item from the user's P2P ad list.
type P2PMyAdItem struct {
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

// P2PAdListItem holds a single item from the public P2P ads list.
type P2PAdListItem struct {
	Index                int64         `json:"index"`
	Asset                currency.Code `json:"asset"`
	FiatUnit             string        `json:"fiat_unit"`
	Price                types.Number  `json:"price"`
	MaxSingleTransAmount string        `json:"max_single_trans_amount"`
	LibName              string        `json:"lib_name"`
	AdvertizementNo      uint64        `json:"adv_no"`
}

// P2PChatMessagesResponse holds a single P2P chat message.
type P2PChatMessagesResponse struct {
	Messages      []*P2PMessageDetail `json:"messages"`
	Memo          string              `json:"memo"`
	HasHistory    bool                `json:"has_history"`
	TransactionID uint64              `json:"txid"`
	ServerTime    uint64              `json:"SRVTM"`
	OrderStatus   string              `json:"order_status"`
}

// P2PMessageDetail represents a P2P conversation message detail
type P2PMessageDetail struct {
	IsSell        int64          `json:"is_sell,omitempty"`
	MessageType   int64          `json:"msg_type,omitempty"`
	Msg           string         `json:"msg"`
	Username      string         `json:"username"`
	Timest        types.Time     `json:"timest"`
	RiskType      int64          `json:"risk_type,omitempty"`
	ToastMsg      string         `json:"toast_msg,omitempty"`
	UID           string         `json:"uid,omitempty"`
	Type          int64          `json:"type,omitempty"`
	MessageObject *MessageObject `json:"msg_obj,omitempty"`
	Pic           string         `json:"pic,omitempty"`
	FileKey       string         `json:"file_key,omitempty"`
	FileType      string         `json:"file_type,omitempty"`
}

// MessageObject represents
type MessageObject struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Text               string `json:"text"`
	ReasonID           int    `json:"reason_id"`
	ToastID            int    `json:"toast_id"`
	ReasonMemo         string `json:"reason_memo"`
	CancelTime         int64  `json:"cancel_time"`
	SellerConfirm      int64  `json:"seller_confirm"`
	PaymentVoucher     []any  `json:"payment_voucher"`
	AccountDescription string `json:"account_des"`
	PaymentType        string `json:"pay_type"`
	File               string `json:"file"`
	FileKey            string `json:"file_key"`
	Account            string `json:"account"`
	Memo               string `json:"memo"`
	Code               string `json:"code"`
	MemoExtended       string `json:"memo_ext"`
	TradeTips          string `json:"trade_tips"`
	RealName           string `json:"real_name"`
	IsDelete           int    `json:"is_delete"`
	PayName            string `json:"pay_name"`
}

// P2PChatSenderInfo holds basic info about the sender of a P2P chat message.
type P2PChatSenderInfo struct {
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

// P2PSendMessageResult holds the result of sending a P2P chat message.
type P2PSendMessageResult struct {
	Srvtm          int64  `json:"SRVTM"`
	TransactionID  int64  `json:"txid"`
	ConversationID string `json:"conversation_id"`
	MessageType    int64  `json:"msg_type"`
	RiskType       int64  `json:"risk_type"`
	ToastMessge    string `json:"toast_msg"`
}

// UploadP2PChatFileRequest holds request parameters for uploading a P2P chat file.
type UploadP2PChatFileRequest struct {
	ImageContentType string `json:"image_content_type"`
	Base64Img        string `json:"base64_img"`
}

// P2PUploadFileResult holds the result of uploading a P2P chat file.
type P2PUploadFileResult struct {
	FileKey string `json:"file_key"`
}
