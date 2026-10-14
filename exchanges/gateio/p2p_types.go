package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// P2P merchant working statuses accepted by SetMerchantWorkingStatusAndCustomWorking
const (
	P2PWorkStatusResting     uint8 = 0
	P2PWorkStatusWorking     uint8 = 1
	P2PWorkStatusCustomHours uint8 = 2
)

// P2P advertisement operations accepted by PublishP2PAdOrder
const (
	P2PAdOperationPublishSell uint8 = 0
	P2PAdOperationPublishBuy  uint8 = 1
	P2PAdOperationEditSell    uint8 = 2
	P2PAdOperationEditBuy     uint8 = 3
)

// P2P advertisement trading limit units
const (
	P2PLimitBasisCrypto uint8 = 0
	P2PLimitBasisFiat   uint8 = 1
)

// P2PMerchantInfo holds P2P merchant account information.
type P2PMerchantInfo struct {
	IsSelf                    bool                   `json:"is_self"`
	UserRegistrationTime      string                 `json:"user_timest"` // Formatted date, such as 2025/11/19
	CounterpartiesNumber      uint64                 `json:"counterparties_num"`
	EmailVerified             types.Boolean          `json:"email_verified"`
	Verified                  types.Boolean          `json:"verified"`
	HasPhone                  types.Boolean          `json:"has_phone"`
	UserName                  string                 `json:"user_name"`
	UserNote                  string                 `json:"user_note"`
	CompleteTransactions      types.Number           `json:"complete_transactions"`
	PaidTransactions          types.Number           `json:"paid_transactions"`
	AcceptedTransactions      types.Number           `json:"accepted_transactions"`
	TransactionsUsedTime      types.Number           `json:"transactions_used_time"`
	CancelledUsedTimeMonth    types.Number           `json:"cancelled_used_time_month"`
	CompleteTransactionsMonth types.Number           `json:"complete_transactions_month"`
	CompleteRateMonth         types.Number           `json:"complete_rate_month"`
	OrdersBuyRateMonth        types.Number           `json:"orders_buy_rate_month"`
	IsBlack                   types.Boolean          `json:"is_black"`
	IsFollow                  types.Boolean          `json:"is_follow"`
	HaveTraded                types.Boolean          `json:"have_traded"`
	BizUID                    string                 `json:"biz_uid"`
	BlueVIP                   uint8                  `json:"blue_vip"`
	WorkStatus                uint8                  `json:"work_status"`
	RegistrationDays          uint64                 `json:"registration_days"`
	FirstTradeDays            uint64                 `json:"first_trade_days"`
	NeedReplenish             types.Boolean          `json:"need_replenish"`
	MerchantInfo              *P2PMerchantMarketInfo `json:"merchant_info"`
	OnlineStatus              uint8                  `json:"online_status"`
	WorkHours                 json.RawMessage        `json:"work_hours"` // Gate documents no fields for the work hours object
	TransactionsMonth         types.Number           `json:"transactions_month"`
	TransactionsAll           types.Number           `json:"transactions_all"`
	TradeVersatile            bool                   `json:"trade_versatile"`
}

// P2PMerchantMarketInfo holds the markets a merchant can place orders in
type P2PMerchantMarketInfo struct {
	Type   string `json:"type"`
	Market string `json:"market"`
}

// GetCounterpartyInfoRequest holds the request parameters for getting counterparty info.
type GetCounterpartyInfoRequest struct {
	BizUID string `json:"biz_uid"`
}

// P2PCounterpartyInfo holds P2P counterparty user information.
type P2PCounterpartyInfo struct {
	UserRegistrationTime      string        `json:"user_timest"` // Formatted date, such as 2025/11/19
	EmailVerified             types.Boolean `json:"email_verified"`
	Verified                  types.Boolean `json:"verified"`
	HasPhone                  types.Boolean `json:"has_phone"`
	UserName                  string        `json:"user_name"`
	UserNote                  string        `json:"user_note"`
	CompleteTransactions      types.Number  `json:"complete_transactions"`
	PaidTransactions          types.Number  `json:"paid_transactions"`
	AcceptedTransactions      types.Number  `json:"accepted_transactions"`
	TransactionsUsedTime      types.Number  `json:"transactions_used_time"`
	CancelledUsedTimeMonth    types.Number  `json:"cancelled_used_time_month"`
	CompleteTransactionsMonth types.Number  `json:"complete_transactions_month"`
	CompleteRateMonth         types.Number  `json:"complete_rate_month"`
	IsFollow                  types.Boolean `json:"is_follow"`
	HaveTraded                types.Boolean `json:"have_traded"`
	BizUID                    string        `json:"biz_uid"`
	RegistrationDays          uint64        `json:"registration_days"`
	FirstTradeDays            uint64        `json:"first_trade_days"`
	TradeVersatile            bool          `json:"trade_versatile"`
}

// GetP2PPaymentMethodsRequest holds the request parameters for getting payment methods.
type GetP2PPaymentMethodsRequest struct {
	Fiat currency.Code // Optional; all payment methods are returned when empty
}

// P2PPaymentMethodGroup holds a group of payment methods of the same type.
type P2PPaymentMethodGroup struct {
	PayType string              `json:"pay_type"`
	PayName string              `json:"pay_name"`
	IDs     []uint64            `json:"ids"`
	List    []*P2PPaymentMethod `json:"list"`
}

// P2PPaymentMethod holds a single bound payment method account. Bank accounts populate the bank fields, while
// other payment types populate the account fields.
type P2PPaymentMethod struct {
	UID                   uint64 `json:"uid"`
	BankID                string `json:"bankid"`
	Nickname              uint64 `json:"nickname"` // Gate documents this as the cardholder UID
	BankName              string `json:"bankname"`
	BankBranch            string `json:"bankbranch"`
	BankCity              string `json:"bankcity"`
	BankProvince          string `json:"bankprov"`
	BankCardNumber        string `json:"bankaddr"` // Possibly masked
	BankDescription       string `json:"bankdesc"`
	HolderUID             uint64 `json:"hold_uid"`
	HolderUsername        string `json:"hold_username"`
	RealName              string `json:"real_name"`
	ID                    string `json:"id"`
	AccountDescription    string `json:"account_des"`
	PaymentMethodType     string `json:"pay_type"`
	PaymentMethodFileLink string `json:"file"`
	PaymentMethodFileKey  string `json:"file_key"`
	Account               string `json:"account"` // Possibly masked
	Memo                  string `json:"memo"`
	Code                  string `json:"code"`
	MemoExtended          string `json:"memo_ext"`
	TradeTips             string `json:"trade_tips"`
}

// SetMerchantWorkHoursRequest represents request parameters to set a merchant's working status and hours. The
// remaining fields are only used, and except DayOfWeek are then required, when WorkStatus is
// P2PWorkStatusCustomHours.
type SetMerchantWorkHoursRequest struct {
	WorkStatus uint8  `json:"work_status"`
	CycleType  string `json:"cycle_type,omitempty"`  // Weekly or Daily
	DayOfWeek  string `json:"day_of_week,omitempty"` // Comma separated 1 (Monday) to 7 (Sunday); required for a Weekly cycle
	TimeZone   string `json:"time_zone,omitempty"`   // UTC offset from -12 to +14
	StartTime  string `json:"start_time,omitempty"`  // HH:mm
	EndTime    string `json:"end_time,omitempty"`    // HH:mm
}

// P2PMerchantWorkStatusResponse holds the merchant's working status after it has been set: 0 normal resting,
// 1 normal working, 2 custom resting or 3 custom working
type P2PMerchantWorkStatusResponse struct {
	WorkStatus uint8 `json:"work_status"`
}

// PendingP2POrderRequest represents a pending P2P order list request parameter
type PendingP2POrderRequest struct {
	CryptoCurrency currency.Code `json:"crypto_currency"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	OrderTab       string        `json:"order_tab,omitempty"`   // pending or dispute
	SelectType     string        `json:"select_type,omitempty"` // buy or sell
	Status         string        `json:"status,omitempty"`      // open, paid, locked or dispute
	TransactionID  uint64        `json:"txid,omitempty"`
	StartTime      time.Time     `json:"-"`
	EndTime        time.Time     `json:"-"`
}

// MarshalJSON sends the time range as Unix seconds
func (r *PendingP2POrderRequest) MarshalJSON() ([]byte, error) {
	type Alias PendingP2POrderRequest
	return json.Marshal(&struct {
		*Alias
		StartTime int64 `json:"start_time,omitempty"`
		EndTime   int64 `json:"end_time,omitempty"`
	}{
		Alias:     (*Alias)(r),
		StartTime: unixSecondsOrZero(r.StartTime),
		EndTime:   unixSecondsOrZero(r.EndTime),
	})
}

// P2PCompletedOrderRequest holds request parameters to retrieve completed P2P orders
type P2PCompletedOrderRequest struct {
	CryptoCurrency currency.Code `json:"crypto_currency"`
	FiatCurrency   currency.Code `json:"fiat_currency"`
	SelectType     string        `json:"select_type,omitempty"` // buy or sell
	Status         string        `json:"status,omitempty"`      // closed, cancel, locked, open, paid or completed
	TransactionID  uint64        `json:"txid,omitempty"`
	StartTime      time.Time     `json:"-"`
	EndTime        time.Time     `json:"-"`
	QueryDispute   bool          `json:"-"`
	Page           uint64        `json:"page,omitempty"`
	PerPage        uint64        `json:"per_page,omitempty"`
}

// MarshalJSON sends the time range as Unix seconds and the dispute flag as 1 or 0
func (r *P2PCompletedOrderRequest) MarshalJSON() ([]byte, error) {
	type Alias P2PCompletedOrderRequest
	aux := &struct {
		*Alias
		StartTime    int64 `json:"start_time,omitempty"`
		EndTime      int64 `json:"end_time,omitempty"`
		QueryDispute uint8 `json:"query_dispute,omitempty"`
	}{
		Alias:     (*Alias)(r),
		StartTime: unixSecondsOrZero(r.StartTime),
		EndTime:   unixSecondsOrZero(r.EndTime),
	}
	if r.QueryDispute {
		aux.QueryDispute = 1
	}
	return json.Marshal(aux)
}

// unixSecondsOrZero returns t as Unix seconds, or 0 for an unset time so it can be omitted
func unixSecondsOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// P2POrderList represents a P2P transactions detail list
type P2POrderList struct {
	List            []*P2POrderInfo             `json:"list"`
	TransactionTime []*P2PTransactionTimeMarker `json:"trans_time"`
	Count           uint64                      `json:"count"`
	ExportedNumber  uint64                      `json:"exported_num"`
}

// P2PTransactionTimeMarker represents an order countdown
type P2PTransactionTimeMarker struct {
	OrderTime int64 `json:"od_time"`
}

// P2POrderInfo represents a P2P order list item
type P2POrderInfo struct {
	IsBuy                types.Boolean            `json:"type_buy"` // From the current user's view
	CreationTime         string                   `json:"timest"`   // Display time, as Timestamp carries the same moment
	ExpiryTime           string                   `json:"timest_expire"`
	Timestamp            types.Time               `json:"timestamp"`
	Rate                 types.Number             `json:"rate"`
	Amount               types.Number             `json:"amount"`
	Total                types.Number             `json:"total"`
	TransactionID        uint64                   `json:"txid"`
	Status               string                   `json:"status"`
	CounterpartyRealName string                   `json:"its_realname"`
	CounterpartyUID      string                   `json:"its_uid"`
	CounterpartyNickname string                   `json:"its_nick"`
	SellerRealName       string                   `json:"seller_realname"`
	BuyerRealName        string                   `json:"buyer_realname"`
	Cancelable           types.Boolean            `json:"cancelable"`
	CurrencyType         currency.Code            `json:"currency_type"`
	WantType             currency.Code            `json:"want_type"`
	HidePayment          types.Boolean            `json:"hide_payment"`
	SelectedPayType      string                   `json:"sel_paytype"`
	PayOthers            []*P2POtherPaymentMethod `json:"pay_others"`
	CountdownSeconds     int64                    `json:"cd_time"`
	OrderType            uint8                    `json:"order_type"` // 1 standard, 2 partner, 3 flash swap or 4 Web3
	OrderTags            []string                 `json:"order_tag"`
	ConvertInfo          *P2PConvertInfo          `json:"convert_info"`
}

// P2POrderDetail represents a P2P order detail
type P2POrderDetail struct {
	IsSell                  types.Boolean            `json:"is_sell"`
	TransactionID           uint64                   `json:"txid"`
	OrderID                 uint64                   `json:"orderid"`
	CreationTime            types.Time               `json:"timest"`
	PaymentDeadline         types.Time               `json:"last_pay_time"`
	RemainingPaymentSeconds int64                    `json:"remain_pay_time"`
	CurrencyType            currency.Code            `json:"currency_type"`
	WantType                currency.Code            `json:"want_type"`
	FiatSymbol              string                   `json:"symbol"`
	Rate                    types.Number             `json:"rate"`
	Amount                  types.Number             `json:"amount"`
	Total                   types.Number             `json:"total"`
	Status                  string                   `json:"status"`
	ReasonID                string                   `json:"reason_id"`
	ReasonDescription       string                   `json:"reason_desc"`
	CancelTime              string                   `json:"cancel_time"` // Kept as sent, since Gate documents no format
	InAppeal                types.Boolean            `json:"in_appeal"`
	DisputeTime             types.Time               `json:"dispute_time"`
	Cancelable              types.Boolean            `json:"cancelable"`
	HidePayment             types.Boolean            `json:"hide_payment"`
	TradeTips               string                   `json:"trade_tips"`
	ShowBank                types.Boolean            `json:"show_bank"`
	BankName                string                   `json:"bankname"`
	BankBranch              string                   `json:"bankbranch"`
	BankAccount             string                   `json:"bankid"` // Possibly masked
	BankHolderRealName      string                   `json:"bank_holder_realname"`
	ShowAlipay              types.Boolean            `json:"show_ali"`
	AlipayName              string                   `json:"aliname"`
	HasAlipayCode           types.Boolean            `json:"is_alicode"`
	ShowWechat              types.Boolean            `json:"show_wechat"`
	WechatName              string                   `json:"wename"`
	ShowOthers              types.Boolean            `json:"show_others"`
	PayOthers               []*P2POtherPaymentMethod `json:"pay_others"`
	SelectedPayType         string                   `json:"sel_paytype"`
	CounterpartyUID         string                   `json:"its_uid"`
	CounterpartyNickname    string                   `json:"its_nickname"`
	CounterpartyRealName    string                   `json:"its_realname"`
	HaveTraded              types.Boolean            `json:"have_traded"`
	AppealAllowCancel       types.Boolean            `json:"appeal_allow_cancel"`
	AppealVerdictHasOpen    string                   `json:"appeal_verdict_has_open"`
	UnreadMessages          uint64                   `json:"im_unread"`
	PaymentVoucherURLs      []string                 `json:"payment_voucher_url"`
	PaidTime                types.Time               `json:"timest_paid"`
	OwnRealName             string                   `json:"own_realname"`
	OrderType               uint8                    `json:"order_type"` // 1 standard, 2 partner, 3 flash swap or 4 Web3
	IsShowReceive           types.Boolean            `json:"is_show_receive"`
	ShowSellerContactInfo   bool                     `json:"show_seller_contact_info"`
	SupportedPayTypes       []string                 `json:"supported_pay_types"`
}

// P2POtherPaymentMethod represents a payment method other than bank, Alipay or WeChat. Order lists only populate
// the pay type and name.
type P2POtherPaymentMethod struct {
	ID                 string `json:"id"`
	AccountDescription string `json:"account_des"`
	PayType            string `json:"pay_type"`
	PayName            string `json:"pay_name"`
	Account            string `json:"account"` // Possibly masked
	Memo               string `json:"memo"`
	TradeTips          string `json:"trade_tips"`
}

// P2PConvertInfo represents a P2P flash swap order's conversion details
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

// GetP2POrderDetailsRequest holds request parameters for querying P2P order details.
type GetP2POrderDetailsRequest struct {
	TransactionID uint64 `json:"txid"`
	Channel       string `json:"channel,omitempty"` // web3 for Web3 orders, empty for normal P2P orders
}

// ConfirmP2PPaymentRequest holds request parameters for confirming P2P payment.
type ConfirmP2PPaymentRequest struct {
	TransactionID string `json:"txid"`
	PaymentMethod string `json:"payment_method,omitempty"` // One of the order's supported pay types
}

// ConfirmP2PReceiptRequest holds request parameters for confirming P2P receipt.
type ConfirmP2PReceiptRequest struct {
	TransactionID string `json:"txid"`
}

// CancelP2POrderRequest holds request parameters for cancelling a P2P order.
// ReasonID values: 1=no longer want to buy, 2=cannot reach seller, 3=will not pay, 4=seller account not real,
// 5=payout account issue, 6=price mismatch, 7=mutually agreed cancel, 8=poor communication, 9=other,
// 10=seller cannot release with refund, 11=terms not met, 12=seller payout risk-controlled.
type CancelP2POrderRequest struct {
	TransactionID string `json:"txid"`
	ReasonID      uint8  `json:"reason_id,omitempty,string"`
	ReasonMemo    string `json:"reason_memo,omitempty"` // Explains the cancellation, such as for reason 9
}

// PublishP2PAdRequest holds request parameters for publishing or editing a P2P advertisement. The per-order limits
// are set with MinAmount and MaxAmount when LimitBasis is P2PLimitBasisCrypto, or with FiatMinAmount and
// FiatMaxAmount when it is P2PLimitBasisFiat.
type PublishP2PAdRequest struct {
	CurrencyType          currency.Code `json:"currencyType"`
	ExchangeType          currency.Code `json:"exchangeType"` // Fiat currency
	Type                  uint8         `json:"type,string"`  // One of the P2PAdOperation values
	UnitPrice             types.Number  `json:"unitPrice"`
	Number                types.Number  `json:"number"`
	PayType               string        `json:"payType"`                 // Comma separated pay types enabled for the ad, such as bank,swift
	PayTypeJSON           string        `json:"pay_type_json,omitempty"` // JSON text mapping each pay type to a payment method ID
	RateFixed             string        `json:"rateFixed,omitempty"`     // 0 floating or 1 fixed
	AdvertisementID       string        `json:"oid,omitempty"`           // Required when editing an ad
	MinAmount             types.Number  `json:"minAmount,omitempty"`
	MaxAmount             types.Number  `json:"maxAmount,omitempty"`
	LimitBasis            uint8         `json:"limitBasis,omitempty"`
	FiatMinAmount         types.Number  `json:"fiatMinAmount,omitempty"`
	FiatMaxAmount         types.Number  `json:"fiatMaxAmount,omitempty"`
	TierLimit             types.Number  `json:"tierLimit,omitempty"`
	VerifiedLimit         types.Number  `json:"verifiedLimit,omitempty"`
	RegistrationTimeLimit types.Number  `json:"regTimeLimit,omitempty"`
	AdvertisersLimit      string        `json:"advertisersLimit,omitempty"` // 0 unrestricted or 1 restricted
	PolymarketLimit       uint8         `json:"polymarket_limit,omitempty"` // 0 unrestricted or 1 restricted
	ExpireMinutes         types.Number  `json:"expire_min,omitempty"`
	TradeTips             string        `json:"trade_tips,omitempty"`
	AutoReply             string        `json:"auto_reply,omitempty"`
	MinCompletedLimit     types.Number  `json:"min_completed_limit,omitempty"` // -1 unlimited
	MaxCompletedLimit     types.Number  `json:"max_completed_limit,omitempty"` // -1 unlimited
	CompletedRateLimit    types.Number  `json:"completed_rate_limit,omitempty"`
	UserCountryLimit      string        `json:"user_country_limit,omitempty"` // -1 unrestricted
	UserOrderLimit        types.Number  `json:"user_order_limit,omitempty"`   // -1 unlimited
	RateReferenceID       string        `json:"rateReferenceId,omitempty"`    // 1 platform, 2 Gate or 3 spot reference
	RateOffset            types.Number  `json:"rateOffset,omitempty"`         // Absolute percentage, 0.5 being 0.5%
	FloatTrend            string        `json:"float_trend,omitempty"`        // 0 markup or 1 markdown
	TeamPaymentUID        string        `json:"team_payment_uid,omitempty"`
}

// P2PPublishAdResponse holds the risk control details returned when an advertisement's trade tips or auto reply
// are rejected; it is empty when the advertisement is accepted
type P2PPublishAdResponse struct {
	RiskCode  int64         `json:"risk_code"`
	RiskEvent *P2PRiskEvent `json:"risk_event"`
}

// P2PRiskEvent holds a risk control prompt
type P2PRiskEvent struct {
	Type            string                `json:"type"`
	Title           string                `json:"title"`
	Message         string                `json:"msg"`
	Actions         []*P2PRiskEventAction `json:"action"`
	ContentRiskType string                `json:"content_risk_type"` // trade_tips, auto_reply or trade_tips_auto_reply
	TradeTips       string                `json:"trade_tips"`
	AutoReply       string                `json:"auto_reply"`
}

// P2PRiskEventAction holds an action offered by a risk control prompt
type P2PRiskEventAction struct {
	ActionType string          `json:"action_type"`
	Title      string          `json:"title"`
	IsPrimary  types.Boolean   `json:"mainly"`
	ActionData json.RawMessage `json:"action_data"`
}

// UpdateP2PAdStatusRequest holds request parameters for updating P2P ad status.
type UpdateP2PAdStatusRequest struct {
	AdvertisementID uint64 `json:"adv_no"`
	Status          uint8  `json:"adv_status"` // 1 listed, 3 delisted or 4 closed
}

// P2PUpdateAdStatusResult holds the result of updating a P2P ad status.
type P2PUpdateAdStatusResult struct {
	Status uint8 `json:"status"`
}

// GetP2PAdDetailsRequest holds request parameters for querying P2P ad details.
type GetP2PAdDetailsRequest struct {
	AdvertisementID string `json:"adv_no"`
}

// P2PAdDetail holds detailed P2P advertisement information.
type P2PAdDetail struct {
	Rate                  types.Number  `json:"rate"`
	Type                  string        `json:"type"`
	Amount                types.Number  `json:"amount"`
	MinAmount             types.Number  `json:"min_amount"`
	MaxAmount             types.Number  `json:"max_amount"`
	FiatMinAmount         types.Number  `json:"fiat_min_amount"`
	FiatMaxAmount         types.Number  `json:"fiat_max_amount"`
	LimitBasis            uint8         `json:"limit_basis"`
	LimitBasisText        string        `json:"limit_basis_text"`
	Total                 types.Number  `json:"total"`
	PayAlipay             types.Boolean `json:"pay_ali"`
	PayBank               types.Boolean `json:"pay_bank"`
	PayPaypal             types.Boolean `json:"pay_paypal"`
	PayWechat             types.Boolean `json:"pay_wechat"`
	PayTypeNumbers        string        `json:"pay_type_num"`  // Comma separated payment method IDs
	PayTypeJSON           string        `json:"pay_type_json"` // JSON text mapping each pay type to a payment method ID
	LockedAmount          types.Number  `json:"locked_amount"`
	OrderID               uint64        `json:"orderid"`
	Timestamp             types.Time    `json:"timestamp"`
	CurrencyType          currency.Code `json:"currency_type"`
	WantType              currency.Code `json:"want_type"`
	HiddenRate            types.Number  `json:"hide_rate"`
	TradeTips             string        `json:"trade_tips"`
	AutoReply             string        `json:"auto_reply"`
	RateReferenceID       int64         `json:"rate_ref_id"` // 1 platform, 2 Gate, 3 spot, or 0 or below for a fixed price
	RateOffset            types.Number  `json:"rate_offset"`
	Status                string        `json:"status"`
	RateFixed             types.Boolean `json:"rate_fixed"`
	FloatTrend            uint8         `json:"float_trend"` // 0 markup or 1 markdown
	ExpireMinutes         uint64        `json:"expire_min"`
	TierLimit             uint64        `json:"tier_limit"`
	RegistrationTimeLimit uint64        `json:"reg_time_limit"`
	AdvertisersLimit      types.Boolean `json:"advertisers_limit"`
	PolymarketLimit       types.Boolean `json:"polymarket_limit"`
	MinCompletedLimit     int64         `json:"min_completed_limit"`
	MaxCompletedLimit     int64         `json:"max_completed_limit"`
	UserOrdersLimit       int64         `json:"user_orders_limit"`
	CompletedRateLimit    types.Number  `json:"completed_rate_limit"`
	LimitCountryCN        string        `json:"limit_country_cn"`
	LimitCountryEN        string        `json:"limit_country_en"`
	IsHedge               types.Boolean `json:"is_hedge"`
	HidePayment           types.Boolean `json:"hide_payment"`
}

// GetMyP2PAdsRequest holds the optional filters for getting the current user's P2P ads.
type GetMyP2PAdsRequest struct {
	Asset     currency.Code
	FiatUnit  currency.Code
	TradeType string // buy or sell
}

// P2PMyAdsData wraps the list of the user's own P2P ads.
type P2PMyAdsData struct {
	Lists []*P2PMyAdItem `json:"lists"`
}

// P2PMyAdItem holds a single item from the user's P2P ad list.
type P2PMyAdItem struct {
	Type                  string        `json:"type"`
	Rate                  types.Number  `json:"rate"`
	OriginalRate          types.Number  `json:"original_rate"`
	Amount                types.Number  `json:"amount"`
	Total                 types.Number  `json:"total"`
	LimitTotal            string        `json:"limit_total"` // Per-order crypto range, such as 10~500
	LimitFiat             string        `json:"limit_fiat"`  // Per-order fiat range, such as 12.7~635
	MinAmount             types.Number  `json:"min_amount"`
	MaxAmount             types.Number  `json:"max_amount"`
	PayTypeNumbers        string        `json:"pay_type_num"`  // Comma separated payment method IDs
	PayTypeJSON           string        `json:"pay_type_json"` // JSON text mapping each pay type to a payment method ID
	ExpireMinutes         types.Number  `json:"expire_min"`
	TierLimit             types.Number  `json:"tier_limit"`
	AdvertisersLimit      types.Boolean `json:"advertisers_limit"`
	RegistrationTimeLimit uint64        `json:"reg_time_limit"`
	VerifiedLimit         uint64        `json:"verified_limit"`
	MinCompletedLimit     int64         `json:"min_completed_limit"`
	MaxCompletedLimit     int64         `json:"max_completed_limit"`
	UserCountryLimit      int64         `json:"user_country_limit"`
	CompletedRateLimit    types.Number  `json:"completed_rate_limit"`
	UserOrdersLimit       int64         `json:"user_orders_limit"`
	HidePayment           types.Boolean `json:"hide_payment"`
	CurrencyType          currency.Code `json:"currencyType"`
	WantType              currency.Code `json:"want_type"`
	TradeTips             string        `json:"trade_tips"`
	NewHand               uint8         `json:"new_hand"` // Special ad type, 0 being a normal ad
	ID                    string        `json:"id"`
	Status                string        `json:"status"`
	LockedAmount          types.Number  `json:"locked_amount"`
	HiddenRate            types.Number  `json:"hide_rate"`
	IsOutTime             types.Boolean `json:"is_out_time"`
	RateReferenceID       int64         `json:"rate_ref_id"` // 1 platform, 2 Gate, 3 spot, or 0 or below for a fixed price
	RateOffset            types.Number  `json:"rate_offset"`
	RateFixed             types.Boolean `json:"rate_fixed"`
	FloatTrend            uint8         `json:"float_trend"` // 0 markup or 1 markdown
	InDispute             types.Boolean `json:"in_dispute"`
	AutoReply             string        `json:"auto_reply"`
	Timestamp             types.Time    `json:"timestamp"`
	IsHedge               types.Boolean `json:"is_hedge"`
}

// GetP2PAdsListRequest holds request parameters for getting the public P2P ads list.
type GetP2PAdsListRequest struct {
	Asset     currency.Code `json:"asset"`
	FiatUnit  currency.Code `json:"fiat_unit"`
	TradeType string        `json:"trade_type"` // buy or sell
}

// P2PAdListItem holds a single item from the public P2P ads list.
type P2PAdListItem struct {
	Index                   uint64               `json:"index"`
	Asset                   currency.Code        `json:"asset"`
	FiatUnit                currency.Code        `json:"fiat_unit"`
	AdvertisementID         uint64               `json:"adv_no"`
	Price                   types.Number         `json:"price"`
	SurplusAmount           types.Number         `json:"surplus_amount"`
	MaxSingleTransferAmount types.Number         `json:"max_single_trans_amount"`
	MinSingleTransferAmount types.Number         `json:"min_single_trans_amount"`
	FiatMinAmount           types.Number         `json:"fiat_min_amount"`
	FiatMaxAmount           types.Number         `json:"fiat_max_amount"`
	LimitBasis              uint8                `json:"limit_basis"`
	LimitBasisText          string               `json:"limit_basis_text"`
	TradeMethods            []*P2PAdsTradeMethod `json:"trade_methods"`
	AdvertiserNickname      string               `json:"nick_name"`
}

// P2PAdsTradeMethod holds a payment method supported by a P2P advertisement
type P2PAdsTradeMethod struct {
	IconURL         string `json:"icon_url_color"`
	Identifier      string `json:"identifier"`
	PayID           string `json:"pay_id"`
	PayType         string `json:"pay_type"`
	TradeMethodName string `json:"trade_method_name"`
}

// GetP2PChatHistoryRequest holds the request parameters for retrieving P2P chat history
type GetP2PChatHistoryRequest struct {
	TransactionID uint64    `json:"txid,omitempty"` // The latest order with chat history is used when unset
	LastReceived  time.Time `json:"-"`              // Fetches messages after this one, for incremental updates
	FirstReceived time.Time `json:"-"`              // Fetches messages before this one, for paging backwards
}

// MarshalJSON sends the message times as Unix seconds
func (r *GetP2PChatHistoryRequest) MarshalJSON() ([]byte, error) {
	type Alias GetP2PChatHistoryRequest
	return json.Marshal(&struct {
		*Alias
		LastReceived  int64 `json:"lastreceived,omitempty"`
		FirstReceived int64 `json:"firstreceived,omitempty"`
	}{
		Alias:         (*Alias)(r),
		LastReceived:  unixSecondsOrZero(r.LastReceived),
		FirstReceived: unixSecondsOrZero(r.FirstReceived),
	})
}

// P2PChatMessagesResponse holds a P2P order's chat history.
type P2PChatMessagesResponse struct {
	Messages          []*P2PMessageDetail `json:"messages"`
	Memo              string              `json:"memo"`
	HasHistory        bool                `json:"has_history"`
	TransactionID     uint64              `json:"txid"`
	LatestMessageTime types.Time          `json:"SRVTM"`
	OrderStatus       string              `json:"order_status"`
}

// P2PMessageDetail represents a P2P conversation message
type P2PMessageDetail struct {
	IsSell        types.Boolean     `json:"is_sell"`
	MessageType   uint8             `json:"msg_type"` // 0 text, 1 file, 2 template, 3 order share, 4 payment share or 5 status update
	Message       string            `json:"msg"`
	Username      string            `json:"username"`
	Timestamp     types.Time        `json:"timest"`
	MessageObject *P2PMessageObject `json:"msg_obj"`
	UID           string            `json:"uid"`
	Type          uint8             `json:"type"` // 1 file or 2 system message
	Picture       string            `json:"pic"`
	FileKey       string            `json:"file_key"`
	FileType      string            `json:"file_type"`
	RiskType      uint8             `json:"risk_type"`
	ToastMessage  string            `json:"toast_msg"`
}

// P2PMessageObject holds the order status update or shared payment method carried by a P2P chat message
type P2PMessageObject struct {
	Status             string        `json:"status"`
	Text               string        `json:"text"`
	PaymentVoucher     []string      `json:"payment_voucher"`
	ReasonID           uint8         `json:"reason_id"`
	ToastID            uint64        `json:"toast_id"`
	ReasonMemo         string        `json:"reason_memo"`
	CancelTime         types.Time    `json:"cancel_time"`
	SellerConfirm      uint8         `json:"seller_confirm"` // 0 pending, 1 confirmed or 2 rejected
	ID                 string        `json:"id"`
	AccountDescription string        `json:"account_des"`
	PaymentType        string        `json:"pay_type"`
	File               string        `json:"file"`
	FileKey            string        `json:"file_key"`
	Account            string        `json:"account"`
	Memo               string        `json:"memo"`
	Code               string        `json:"code"`
	MemoExtended       string        `json:"memo_ext"`
	TradeTips          string        `json:"trade_tips"`
	RealName           string        `json:"real_name"`
	IsDeleted          types.Boolean `json:"is_delete"`
	PayName            string        `json:"pay_name"`
}

// SendP2PChatMessageRequest holds request parameters for sending a P2P chat message.
type SendP2PChatMessageRequest struct {
	TransactionID uint64 `json:"txid"`
	Type          uint8  `json:"type,omitempty"` // 0 text (default) or 1 file, whose Message is the file key from UploadP2PChatFile
	Message       string `json:"message"`
}

// P2PSendMessageResult holds the result of sending a P2P chat message.
type P2PSendMessageResult struct {
	SentTime       types.Time `json:"SRVTM"`
	TransactionID  uint64     `json:"txid"`
	ConversationID string     `json:"conversation_id"`
	MessageType    uint8      `json:"msg_type"`
	RiskType       uint8      `json:"risk_type"` // 1 when the message hit off-platform traffic diversion risk control
	ToastMessage   string     `json:"toast_msg"`
}

// UploadP2PChatFileRequest holds request parameters for uploading a P2P chat file.
type UploadP2PChatFileRequest struct {
	ImageContentType string `json:"image_content_type"` // image/jpeg, image/jpg, image/png or video/mp4
	Base64Img        string `json:"base64_img"`
}

// P2PUploadFileResult holds the result of uploading a P2P chat file.
type P2PUploadFileResult struct {
	FileKey string `json:"file_key"`
}
