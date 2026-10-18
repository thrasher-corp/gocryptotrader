package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// P2P responses carry many fields their parameter tables leave out; the documented response examples are recordings
// of real responses, so every field they show is modelled with the JSON type it was recorded with. Scalar fields whose
// recorded and documented JSON types disagree accept either form

// P2POnlineAdsRequest holds the parameters for Get Ads
type P2POnlineAdsRequest struct {
	TokenID    currency.Code
	CurrencyID currency.Code
	// Side is "0" for buy ads or "1" for sell ads
	Side string
	Page uint64
	// Size is the page size, up to 300; 0 uses the default of 10
	Size uint64
}

// P2POnlineAdsResponse is a page of the advertisements listed on the P2P market
type P2POnlineAdsResponse struct {
	Count uint64        `json:"count"`
	Items []P2POnlineAd `json:"items"`
}

// P2POnlineAd is an advertisement listed on the P2P market
type P2POnlineAd struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	// UserID is documented as an integer but recorded as a string
	UserID     types.PreciseNumber `json:"userId"`
	NickName   string              `json:"nickName"`
	TokenID    currency.Code       `json:"tokenId"`
	TokenName  string              `json:"tokenName"`
	CurrencyID currency.Code       `json:"currencyId"`
	// Side is 0 for a buy ad or 1 for a sell ad; documented as a string but recorded as a number, as in Get My Ads
	Side types.Number `json:"side"`
	// PriceType is 0 for a fixed price or 1 for a floating price
	PriceType uint64       `json:"priceType"`
	Price     types.Number `json:"price"`
	// Premium is the share of the reference price a floating price ad uses: 130 is 130%
	Premium          types.Number `json:"premium"`
	LastQuantity     types.Number `json:"lastQuantity"`
	Quantity         types.Number `json:"quantity"`
	FrozenQuantity   types.Number `json:"frozenQuantity"`
	ExecutedQuantity types.Number `json:"executedQuantity"`
	MinAmount        types.Number `json:"minAmount"`
	MaxAmount        types.Number `json:"maxAmount"`
	Remark           string       `json:"remark"`
	// Status is 10 online, 20 offline or 30 completed
	Status     uint64     `json:"status"`
	CreateDate types.Time `json:"createDate"`
	// Payments holds the ad's payment method type IDs
	Payments     []string `json:"payments"`
	OrderNumber  uint64   `json:"orderNum"`
	FinishNumber uint64   `json:"finishNum"`
	// RecentOrderNumber is documented as a string but recorded as a number, as in every other P2P response
	RecentOrderNumber types.Number `json:"recentOrderNum"`
	// RecentExecuteRate is documented as a string but recorded as a number
	RecentExecuteRate    types.Number            `json:"recentExecuteRate"`
	Fee                  types.Number            `json:"fee"`
	IsOnline             bool                    `json:"isOnline"`
	LastLogoutTime       types.Time              `json:"lastLogoutTime"`
	Blocked              string                  `json:"blocked"`
	MakerContact         bool                    `json:"makerContact"`
	SymbolInfo           P2PSymbolInfo           `json:"symbolInfo"`
	TradingPreferenceSet P2PTradingPreferenceSet `json:"tradingPreferenceSet"`
	Version              uint64                  `json:"version"`
	AuthStatus           uint64                  `json:"authStatus"`
	Recommend            bool                    `json:"recommend"`
	RecommendTag         string                  `json:"recommendTag"`
	// AuthTag holds the advertiser's tags: GA general advertiser, VA verified advertiser, BA block advertiser
	AuthTag  []string `json:"authTag"`
	UserType string   `json:"userType"`
	// ItemType is ORIGIN for a regular ad or BULK for a bulk ad
	ItemType string `json:"itemType"`
	// PaymentPeriod is in minutes
	PaymentPeriod uint64 `json:"paymentPeriod"`
}

// P2PSymbolInfo holds the trading rules of an advertisement's token and currency pair
type P2PSymbolInfo struct {
	ID                    string        `json:"id"`
	ExchangeID            string        `json:"exchangeId"`
	OrgID                 string        `json:"orgId"`
	TokenID               currency.Code `json:"tokenId"`
	CurrencyID            currency.Code `json:"currencyId"`
	Status                uint64        `json:"status"`
	LowerLimitAlarm       uint64        `json:"lowerLimitAlarm"`
	UpperLimitAlarm       uint64        `json:"upperLimitAlarm"`
	ItemDownRange         types.Number  `json:"itemDownRange"`
	ItemUpRange           types.Number  `json:"itemUpRange"`
	CurrencyMinQuote      types.Number  `json:"currencyMinQuote"`
	CurrencyMaxQuote      types.Number  `json:"currencyMaxQuote"`
	CurrencyLowerMaxQuote types.Number  `json:"currencyLowerMaxQuote"`
	TokenMinQuote         types.Number  `json:"tokenMinQuote"`
	TokenMaxQuote         types.Number  `json:"tokenMaxQuote"`
	KYCCurrencyLimit      types.Number  `json:"kycCurrencyLimit"`
	ItemSideLimit         uint64        `json:"itemSideLimit"`
	BuyFeeRate            types.Number  `json:"buyFeeRate"`
	SellFeeRate           types.Number  `json:"sellFeeRate"`
	// OrderAutoCancelMinute and OrderFinishMinute are in minutes
	OrderAutoCancelMinute uint64            `json:"orderAutoCancelMinute"`
	OrderFinishMinute     uint64            `json:"orderFinishMinute"`
	TradeSide             uint64            `json:"tradeSide"`
	Currency              P2PSymbolCurrency `json:"currency"`
	Token                 P2PSymbolToken    `json:"token"`
	// BuyAd and SellAd are null for other users' ads
	BuyAd  P2PSymbolAdConfig `json:"buyAd"`
	SellAd P2PSymbolAdConfig `json:"sellAd"`
}

// P2PSymbolCurrency holds the fiat currency of a P2P trading pair
type P2PSymbolCurrency struct {
	ID         string        `json:"id"`
	ExchangeID string        `json:"exchangeId"`
	OrgID      string        `json:"orgId"`
	CurrencyID currency.Code `json:"currencyId"`
	Scale      uint64        `json:"scale"`
}

// P2PSymbolToken holds the token of a P2P trading pair
type P2PSymbolToken struct {
	ID         string        `json:"id"`
	ExchangeID string        `json:"exchangeId"`
	OrgID      string        `json:"orgId"`
	TokenID    currency.Code `json:"tokenId"`
	Scale      uint64        `json:"scale"`
	Sequence   uint64        `json:"sequence"`
}

// P2PSymbolAdConfig holds the payment periods, in minutes, an advertisement on one side may choose from
type P2PSymbolAdConfig struct {
	PaymentPeriods []uint64 `json:"paymentPeriods"`
}

// P2PTradingPreferenceSet holds the requirements an advertisement sets for counterparties
type P2PTradingPreferenceSet struct {
	HasUnPostAd     types.Boolean `json:"hasUnPostAd"`
	IsKYC           types.Boolean `json:"isKyc"`
	IsEmail         types.Boolean `json:"isEmail"`
	IsMobile        types.Boolean `json:"isMobile"`
	HasRegisterTime types.Boolean `json:"hasRegisterTime"`
	// RegisterTimeThreshold is in days
	RegisterTimeThreshold     uint64        `json:"registerTimeThreshold"`
	OrderFinishNumberDay30    uint64        `json:"orderFinishNumberDay30"`
	CompleteRateDay30         types.Number  `json:"completeRateDay30"`
	NationalLimit             string        `json:"nationalLimit"`
	HasOrderFinishNumberDay30 types.Boolean `json:"hasOrderFinishNumberDay30"`
	HasCompleteRateDay30      types.Boolean `json:"hasCompleteRateDay30"`
	HasNationalLimit          types.Boolean `json:"hasNationalLimit"`
}

// P2PTradingPreferenceSetRequest holds the counterparty requirements Post Ad and Update / Relist Ad set; every field is
// sent, so a zero value removes the requirement
type P2PTradingPreferenceSetRequest struct {
	HasUnPostAd     bool
	IsKYC           bool
	IsEmail         bool
	IsMobile        bool
	HasRegisterTime bool
	// RegisterTimeThreshold is in days
	RegisterTimeThreshold     uint64
	OrderFinishNumberDay30    uint64
	CompleteRateDay30         float64
	NationalLimit             string
	HasOrderFinishNumberDay30 bool
	HasCompleteRateDay30      bool
	HasNationalLimit          bool
}

// P2PPostAdRequest holds the parameters for Post Ad
type P2PPostAdRequest struct {
	TokenID    currency.Code
	CurrencyID currency.Code
	// Side is "0" to buy the token or "1" to sell it
	Side string
	// PriceType is "0" for a fixed price or "1" for a floating price
	PriceType string
	// Premium is the share of the reference price a floating price ad uses: 130 is 130%. Zero sends an empty premium
	Premium float64
	// Price is the fixed price; zero sends an empty price
	Price     float64
	MinAmount float64
	MaxAmount float64
	Remark    string
	// TradingPreferenceSet sends an empty object when nil
	TradingPreferenceSet *P2PTradingPreferenceSetRequest
	// PaymentIDs holds up to 5 payment method IDs from Get User Payment
	PaymentIDs []string
	Quantity   float64
	// PaymentPeriod is in minutes
	PaymentPeriod uint64
	// ItemType is ORIGIN for a regular ad or BULK for a bulk ad
	ItemType string
}

// P2PPostAdResponse holds the posted advertisement's ID and the security check it may need
type P2PPostAdResponse struct {
	ItemID            string `json:"itemId"`
	SecurityRiskToken string `json:"securityRiskToken"`
	RiskTokenType     string `json:"riskTokenType"`
	RiskVersion       string `json:"riskVersion"`
	NeedSecurityRisk  bool   `json:"needSecurityRisk"`
}

// P2PUpdateAdRequest holds the parameters for Update / Relist Ad
type P2PUpdateAdRequest struct {
	ID string
	// PriceType is "0" for a fixed price or "1" for a floating price
	PriceType string
	// Premium is the share of the reference price a floating price ad uses: 130 is 130%. Zero sends an empty premium
	Premium float64
	// Price is the fixed price; zero sends an empty price
	Price     float64
	MinAmount float64
	MaxAmount float64
	Remark    string
	// TradingPreferenceSet sends an empty object when nil
	TradingPreferenceSet *P2PTradingPreferenceSetRequest
	// PaymentIDs holds up to 5 payment method IDs from Get User Payment
	PaymentIDs []string
	// ActionType is MODIFY to change the ad or ACTIVE to relist it
	ActionType string
	Quantity   float64
	// PaymentPeriod is in minutes
	PaymentPeriod uint64
}

// P2PUpdateAdResponse holds the security check an advertisement update may need. The docs list no response
// parameters, but their recorded response carries these fields
type P2PUpdateAdResponse struct {
	SecurityRiskToken string `json:"securityRiskToken"`
	RiskTokenType     string `json:"riskTokenType"`
	RiskVersion       string `json:"riskVersion"`
	NeedSecurityRisk  bool   `json:"needSecurityRisk"`
}

// P2PMyAdsRequest holds the parameters for Get My Ads
type P2PMyAdsRequest struct {
	ItemID string
	// Status is "1" for sold out ads or "2" for available ads
	Status string
	// Side is "0" for buy ads or "1" for sell ads
	Side    string
	TokenID currency.Code
	Page    uint64
	// Size is the page size, up to 30; 0 uses the default of 10
	Size       uint64
	CurrencyID currency.Code
}

// P2PMyAdsResponse is a page of the user's advertisements. The docs table leaves out hiddenFlag, which their response
// example shows as a boolean
type P2PMyAdsResponse struct {
	Count      uint64    `json:"count"`
	Items      []P2PMyAd `json:"items"`
	HiddenFlag bool      `json:"hiddenFlag"`
}

// P2PMyAd is one of the user's advertisements
type P2PMyAd struct {
	ID         string        `json:"id"`
	AccountID  string        `json:"accountId"`
	UserID     string        `json:"userId"`
	NickName   string        `json:"nickName"`
	TokenID    currency.Code `json:"tokenId"`
	TokenName  string        `json:"tokenName"`
	CurrencyID currency.Code `json:"currencyId"`
	// Side is 0 for a buy ad or 1 for a sell ad
	Side uint64 `json:"side"`
	// PriceType is 0 for a fixed price or 1 for a floating price
	PriceType        uint64       `json:"priceType"`
	Price            types.Number `json:"price"`
	Premium          types.Number `json:"premium"`
	LastQuantity     types.Number `json:"lastQuantity"`
	Quantity         types.Number `json:"quantity"`
	FrozenQuantity   types.Number `json:"frozenQuantity"`
	ExecutedQuantity types.Number `json:"executedQuantity"`
	MinAmount        types.Number `json:"minAmount"`
	MaxAmount        types.Number `json:"maxAmount"`
	Remark           string       `json:"remark"`
	// Status is 10 online, 20 offline or 30 completed
	Status               uint64                  `json:"status"`
	CreateDate           types.Time              `json:"createDate"`
	Payments             []string                `json:"payments"`
	OrderNumber          uint64                  `json:"orderNum"`
	FinishNumber         uint64                  `json:"finishNum"`
	RecentOrderNumber    uint64                  `json:"recentOrderNum"`
	RecentExecuteRate    types.Number            `json:"recentExecuteRate"`
	Fee                  types.Number            `json:"fee"`
	IsOnline             bool                    `json:"isOnline"`
	LastLogoutTime       types.Time              `json:"lastLogoutTime"`
	SymbolInfo           P2PSymbolInfo           `json:"symbolInfo"`
	TradingPreferenceSet P2PTradingPreferenceSet `json:"tradingPreferenceSet"`
	PaymentTerms         []P2PAdPaymentTerm      `json:"paymentTerms"`
	Version              uint64                  `json:"version"`
	UpdateDate           types.Time              `json:"updateDate"`
	FeeRate              types.Number            `json:"feeRate"`
	// PaymentPeriod is in minutes
	PaymentPeriod uint64 `json:"paymentPeriod"`
	ItemType      string `json:"itemType"`
	// HiddenReason is why the ad is hidden; empty when it is not
	HiddenReason string `json:"hiddenReason"`
}

// P2PAdPaymentTerm is a payment method an advertisement accepts
type P2PAdPaymentTerm struct {
	ID       string `json:"id"`
	RealName string `json:"realName"`
	// PaymentType is the payment method type ID; documented as a string but recorded as a number
	PaymentType            types.Number       `json:"paymentType"`
	BankName               string             `json:"bankName"`
	BranchName             string             `json:"branchName"`
	AccountNo              string             `json:"accountNo"`
	QRCode                 string             `json:"qrcode"`
	Visible                uint64             `json:"visible"`
	PayMessage             string             `json:"payMessage"`
	FirstName              string             `json:"firstName"`
	LastName               string             `json:"lastName"`
	SecondLastName         string             `json:"secondLastName"`
	Clabe                  string             `json:"clabe"`
	DebitCardNumber        string             `json:"debitCardNumber"`
	Mobile                 string             `json:"mobile"`
	BusinessName           string             `json:"businessName"`
	Concept                string             `json:"concept"`
	PaymentExt1            string             `json:"paymentExt1"`
	PaymentExt2            string             `json:"paymentExt2"`
	PaymentExt3            string             `json:"paymentExt3"`
	PaymentExt4            string             `json:"paymentExt4"`
	PaymentExt5            string             `json:"paymentExt5"`
	PaymentExt6            string             `json:"paymentExt6"`
	PaymentTemplateVersion uint64             `json:"paymentTemplateVersion"`
	PaymentConfig          P2PAdPaymentConfig `json:"paymentConfig"`
	RealNameVerified       bool               `json:"realNameVerified"`
}

// P2PAdPaymentConfig describes an advertisement payment method's type
type P2PAdPaymentConfig struct {
	PaymentType    types.Number `json:"paymentType"`
	PaymentName    string       `json:"paymentName"`
	PaymentDialect string       `json:"paymentDialect"`
	// PaymentTemplateItem is only ever documented empty, so its elements are left undecoded
	PaymentTemplateItem json.RawMessage `json:"paymentTemplateItem"`
}

// P2PMyAdDetailsResponse holds one of the user's advertisements
type P2PMyAdDetailsResponse struct {
	ID         string        `json:"id"`
	AccountID  string        `json:"accountId"`
	UserID     string        `json:"userId"`
	NickName   string        `json:"nickName"`
	TokenID    currency.Code `json:"tokenId"`
	TokenName  string        `json:"tokenName"`
	CurrencyID currency.Code `json:"currencyId"`
	// Side is 0 for a buy ad or 1 for a sell ad
	Side uint64 `json:"side"`
	// PriceType is 0 for a fixed price or 1 for a floating price
	PriceType        uint64       `json:"priceType"`
	Price            types.Number `json:"price"`
	Premium          types.Number `json:"premium"`
	LastQuantity     types.Number `json:"lastQuantity"`
	Quantity         types.Number `json:"quantity"`
	FrozenQuantity   types.Number `json:"frozenQuantity"`
	ExecutedQuantity types.Number `json:"executedQuantity"`
	MinAmount        types.Number `json:"minAmount"`
	MaxAmount        types.Number `json:"maxAmount"`
	Remark           string       `json:"remark"`
	// Status is 10 online, 20 offline or 30 completed
	Status               uint64                  `json:"status"`
	CreateDate           types.Time              `json:"createDate"`
	Payments             []string                `json:"payments"`
	OrderNumber          uint64                  `json:"orderNum"`
	FinishNumber         uint64                  `json:"finishNum"`
	RecentOrderNumber    uint64                  `json:"recentOrderNum"`
	RecentExecuteRate    types.Number            `json:"recentExecuteRate"`
	Fee                  types.Number            `json:"fee"`
	IsOnline             bool                    `json:"isOnline"`
	LastLogoutTime       types.Time              `json:"lastLogoutTime"`
	SymbolInfo           P2PSymbolInfo           `json:"symbolInfo"`
	TradingPreferenceSet P2PTradingPreferenceSet `json:"tradingPreferenceSet"`
	PaymentTerms         []P2PAdPaymentTerm      `json:"paymentTerms"`
	// Version is 1 for ads that pre-freeze funds or 2 for ads that do not
	Version    uint64       `json:"version"`
	UpdateDate types.Time   `json:"updateDate"`
	FeeRate    types.Number `json:"feeRate"`
	// PaymentPeriod is in minutes
	PaymentPeriod uint64 `json:"paymentPeriod"`
	ItemType      string `json:"itemType"`
}

// P2POrderListRequest holds the parameters for Get All Orders and Get Pending Orders
type P2POrderListRequest struct {
	Page uint64
	// Size is the page size, up to 30
	Size uint64
	// Status filters by order status, such as 10 waiting for the buyer to pay or 20 waiting for the seller to release
	Status    uint64
	BeginTime time.Time
	EndTime   time.Time
	TokenID   currency.Code
	// Side is "0" for buy orders or "1" for sell orders; empty returns both
	Side string
}

// P2PAllOrdersResponse is a page of P2P orders
type P2PAllOrdersResponse struct {
	Count uint64     `json:"count"`
	Items []P2POrder `json:"items"`
}

// P2POrder is a P2P order summary
type P2POrder struct {
	ID string `json:"id"`
	// Side is 0 for a buy or 1 for a sell
	Side    uint64        `json:"side"`
	TokenID currency.Code `json:"tokenId"`
	// OrderType is ORIGIN for a regular order, SMALL_COIN for a HotSwap order or WEB3 for a Web3 order
	OrderType           string        `json:"orderType"`
	Amount              types.Number  `json:"amount"`
	CurrencyID          currency.Code `json:"currencyId"`
	Price               types.Number  `json:"price"`
	NotifyTokenQuantity types.Number  `json:"notifyTokenQuantity"`
	NotifyTokenID       currency.Code `json:"notifyTokenId"`
	Fee                 types.Number  `json:"fee"`
	TargetNickName      string        `json:"targetNickName"`
	TargetUserID        string        `json:"targetUserId"`
	// Status is 5 waiting for chain, 10 waiting for the buyer to pay, 20 waiting for the seller to release, 30
	// appealing, 40 cancelled, 50 finished, 60 paying, 70 payment failed, 80 exception cancelled, 90 waiting for the
	// buyer to select a token, 100 objectioning or 110 waiting for an objection
	Status                 uint64       `json:"status"`
	SelfUnreadMessageCount types.Number `json:"selfUnreadMsgCount"`
	CreateDate             types.Time   `json:"createDate"`
	// TransferLastSeconds and AppealLastSeconds are the seconds left to pay and to appeal
	TransferLastSeconds types.Number      `json:"transferLastSeconds"`
	AppealLastSeconds   types.Number      `json:"appealLastSeconds"`
	UserID              string            `json:"userId"`
	SellerRealName      string            `json:"sellerRealName"`
	BuyerRealName       string            `json:"buyerRealName"`
	JudgeInfo           P2PJudgeInfo      `json:"judgeInfo"`
	UnreadMessageCount  types.Number      `json:"unreadMsgCount"`
	Extension           P2POrderExtension `json:"extension"`
	BulkOrderFlag       bool              `json:"bulkOrderFlag"`
}

// P2PJudgeInfo holds the arbitration state of a P2P order appeal
type P2PJudgeInfo struct {
	AutoJudgeUnlockTime types.Time `json:"autoJudgeUnlockTime"`
	DissentResult       string     `json:"dissentResult"`
	PreDissent          string     `json:"preDissent"`
	PostDissent         string     `json:"postDissent"`
}

// P2POrderExtension holds a P2P order's delayed withdrawal details
type P2POrderExtension struct {
	IsDelayWithdraw bool `json:"isDelayWithdraw"`
	// DelayTime is the withdrawal delay; its unit is undocumented
	DelayTime types.Number `json:"delayTime"`
	StartTime types.Time   `json:"startTime"`
}

// P2POrderDetailResponse holds a P2P order's details
type P2POrderDetailResponse struct {
	ID string `json:"id"`
	// Side is 0 for a buy or 1 for a sell
	Side                     uint64        `json:"side"`
	ItemID                   string        `json:"itemId"`
	AccountID                string        `json:"accountId"`
	UserID                   string        `json:"userId"`
	NickName                 string        `json:"nickName"`
	MakerUserID              string        `json:"makerUserId"`
	TargetAccountID          string        `json:"targetAccountId"`
	TargetUserID             string        `json:"targetUserId"`
	TargetNickName           string        `json:"targetNickName"`
	TargetUserMaskID         string        `json:"targetUserMaskId"`
	TargetFirstName          string        `json:"targetFirstName"`
	TargetSecondName         string        `json:"targetSecondName"`
	TargetUserAuthStatus     uint64        `json:"targetUserAuthStatus"`
	TargetConnectInformation string        `json:"targetConnectInformation"`
	PayerRealName            string        `json:"payerRealName"`
	SellerRealName           string        `json:"sellerRealName"`
	BuyerRealName            string        `json:"buyerRealName"`
	TokenID                  currency.Code `json:"tokenId"`
	TokenName                string        `json:"tokenName"`
	CurrencyID               currency.Code `json:"currencyId"`
	Price                    types.Number  `json:"price"`
	Quantity                 types.Number  `json:"quantity"`
	Amount                   types.Number  `json:"amount"`
	PayCode                  string        `json:"payCode"`
	// PaymentType is the payment method type ID the buyer used
	PaymentType  types.Number `json:"paymentType"`
	TransferDate types.Time   `json:"transferDate"`
	// Status is 5 waiting for chain, 10 waiting for the buyer to pay, 20 waiting for the seller to release, 30
	// appealing, 90 waiting for the buyer to select a token, 100 objectioning or 110 waiting for an objection
	Status     uint64     `json:"status"`
	CreateDate types.Time `json:"createDate"`
	// PaymentTermList holds the payment methods the seller offers
	PaymentTermList []P2POrderPaymentTerm `json:"paymentTermList"`
	Remark          string                `json:"remark"`
	// TransferLastSeconds and AppealLastSeconds are the seconds left to pay and to appeal
	TransferLastSeconds types.Number        `json:"transferLastSeconds"`
	RecentOrderNumber   uint64              `json:"recentOrderNum"`
	RecentExecuteRate   types.Number        `json:"recentExecuteRate"`
	AppealLastSeconds   types.Number        `json:"appealLastSeconds"`
	AppealContent       string              `json:"appealContent"`
	AppealType          uint64              `json:"appealType"`
	AppealNickName      string              `json:"appealNickName"`
	CanAppeal           string              `json:"canAppeal"`
	TotalAppealTimes    types.Number        `json:"totalAppealTimes"`
	AppealedTimes       types.Number        `json:"appealedTimes"`
	PaymentTermResult   P2POrderPaymentTerm `json:"paymentTermResult"`
	// OrderFinishMinute is the time allowed to finish the order; its unit is undocumented despite the name
	OrderFinishMinute uint64 `json:"orderFinishMinute"`
	// ConfirmedPayTerm is the payment method the buyer selected
	ConfirmedPayTerm P2POrderPaymentTerm `json:"confirmedPayTerm"`
	MakerFee         types.Number        `json:"makerFee"`
	TakerFee         types.Number        `json:"takerFee"`
	Fee              types.Number        `json:"fee"`
	ShowContact      bool                `json:"showContact"`
	// ContactInfo is only ever documented empty, so its elements are left undecoded
	ContactInfo            json.RawMessage   `json:"contactInfo"`
	TokenBalance           types.Number      `json:"tokenBalance"`
	FiatBalance            types.Number      `json:"fiatBalance"`
	UnreadMessageCount     types.Number      `json:"unreadMsgCount"`
	UpdateDate             types.Time        `json:"updateDate"`
	Extension              P2POrderExtension `json:"extension"`
	SelfUnreadMessageCount types.Number      `json:"selfUnreadMsgCount"`
	JudgeType              string            `json:"judgeType"`
	CanReport              bool              `json:"canReport"`
	CanReportDisagree      bool              `json:"canReportDisagree"`
	CanReportType          []string          `json:"canReportType"`
	CanReportDisagreeType  []string          `json:"canReportDisagreeType"`
	AppraiseStatus         string            `json:"appraiseStatus"`
	AppraiseInfo           P2PAppraiseInfo   `json:"appraiseInfo"`
	CanReportDisagreeTypes []string          `json:"canReportDisagreeTypes"`
	CanReportTypes         []string          `json:"canReportTypes"`
	// OrderType is ORIGIN for a regular order, SMALL_COIN for a HotSwap order or WEB3 for a Web3 order
	OrderType           string        `json:"orderType"`
	MiddleToken         currency.Code `json:"middleToken"`
	BeforePrice         types.Number  `json:"beforePrice"`
	BeforeQuantity      types.Number  `json:"beforeQuantity"`
	BeforeToken         currency.Code `json:"beforeToken"`
	Alternative         string        `json:"alternative"`
	AppealUserID        string        `json:"appealUserId"`
	NotifyTokenID       currency.Code `json:"notifyTokenId"`
	NotifyTokenQuantity types.Number  `json:"notifyTokenQuantity"`
	CancelResponsible   string        `json:"cancelResponsible"`
	ChainType           string        `json:"chainType"`
	ChainAddress        string        `json:"chainAddress"`
	TradeHashCode       string        `json:"tradeHashCode"`
	EstimatedGasFee     types.Number  `json:"estimatedGasFee"`
	GasFeeTokenID       currency.Code `json:"gasFeeTokenId"`
	TradingFeeTokenID   currency.Code `json:"tradingFeeTokenId"`
	OnChainInfo         string        `json:"onChainInfo"`
	TransactionID       string        `json:"transactionId"`
	// DisplayRefund is undocumented; only 0 has been recorded
	DisplayRefund types.Number `json:"displayRefund"`
	// ChainWithdrawLastSeconds and ChainTransferLastSeconds are the seconds left for on-chain withdrawal and transfer
	ChainWithdrawLastSeconds types.Number `json:"chainWithdrawLastSeconds"`
	ChainTransferLastSeconds types.Number `json:"chainTransferLastSeconds"`
	OrderSource              string       `json:"orderSource"`
	CancelReason             string       `json:"cancelReason"`
	// NeedBuyerExamineCancel is set while a seller's cancel request awaits the buyer's review
	NeedBuyerExamineCancel         bool          `json:"needBuyerExamineCancel"`
	SellerCancelExamineRemainTime  types.Number  `json:"sellerCancelExamineRemainTime"`
	NeedSellerExamineCancel        bool          `json:"needSellerExamineCancel"`
	CouponCurrencyAmount           types.Number  `json:"couponCurrencyAmount"`
	TotalCurrencyAmount            types.Number  `json:"totalCurrencyAmount"`
	UsedCoupon                     bool          `json:"usedCoupon"`
	CouponTokenID                  currency.Code `json:"couponTokenId"`
	CouponQuantity                 types.Number  `json:"couponQuantity"`
	CompletedOrderAppealCount      uint64        `json:"completedOrderAppealCount"`
	TotalCompletedOrderAppealCount uint64        `json:"totalCompletedOrderAppealCount"`
	RealOrderStatus                uint64        `json:"realOrderStatus"`
	AppealVersion                  uint64        `json:"appealVersion"`
	JudgeInfo                      P2PJudgeInfo  `json:"judgeInfo"`
	HelpType                       string        `json:"helpType"`
	AppealFlowStatus               string        `json:"appealFlowStatus"`
	AppealSubStatus                string        `json:"appealSubStatus"`
	BulkOrderFlag                  bool          `json:"bulkOrderFlag"`
	// TargetUserType is PERSONAL or ORG
	TargetUserType          string   `json:"targetUserType"`
	TargetUserDisplays      []string `json:"targetUserDisplays"`
	AppealProcessChangeFlag bool     `json:"appealProcessChangeFlag"`
	AppealNegotiationNode   uint64   `json:"appealNegotiationNode"`
}

// P2POrderPaymentTerm is a payment method attached to a P2P order
type P2POrderPaymentTerm struct {
	ID       string `json:"id"`
	RealName string `json:"realName"`
	// PaymentType is the payment method type ID; documented as a string but recorded as a number
	PaymentType     types.Number `json:"paymentType"`
	BankName        string       `json:"bankName"`
	BranchName      string       `json:"branchName"`
	AccountNo       string       `json:"accountNo"`
	QRCode          string       `json:"qrcode"`
	Visible         uint64       `json:"visible"`
	PayMessage      string       `json:"payMessage"`
	FirstName       string       `json:"firstName"`
	LastName        string       `json:"lastName"`
	SecondLastName  string       `json:"secondLastName"`
	Clabe           string       `json:"clabe"`
	DebitCardNumber string       `json:"debitCardNumber"`
	Mobile          string       `json:"mobile"`
	BusinessName    string       `json:"businessName"`
	Concept         string       `json:"concept"`
	// Online is "0" for a non-balance payment method or "1" for a balance payment method, and empty when unset
	Online                 string             `json:"online"`
	PaymentExt1            string             `json:"paymentExt1"`
	PaymentExt2            string             `json:"paymentExt2"`
	PaymentExt3            string             `json:"paymentExt3"`
	PaymentExt4            string             `json:"paymentExt4"`
	PaymentExt5            string             `json:"paymentExt5"`
	PaymentExt6            string             `json:"paymentExt6"`
	PaymentTemplateVersion uint64             `json:"paymentTemplateVersion"`
	PaymentConfigVO        P2PPaymentConfigVO `json:"paymentConfigVo"`
	RUPaymentPrompt        bool               `json:"ruPaymentPrompt"`
}

// P2PPaymentConfigVO describes a payment method type and the fields it collects
type P2PPaymentConfigVO struct {
	// PaymentType is the payment method type ID; sent as a string here and as a number in payment terms
	PaymentType types.Number `json:"paymentType"`
	CheckType   uint64       `json:"checkType"`
	Sort        uint64       `json:"sort"`
	PaymentName string       `json:"paymentName"`
	AddTips     string       `json:"addTips"`
	ItemTips    string       `json:"itemTips"`
	// Online is 0 for a non-balance payment method or 1 for a balance payment method
	Online uint64                 `json:"online"`
	Items  []P2PPaymentConfigItem `json:"items"`
}

// P2PPaymentConfigItem is a field a payment method type collects
type P2PPaymentConfigItem struct {
	View        bool         `json:"view"`
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Placeholder string       `json:"placeholder"`
	Type        string       `json:"type"`
	MaxLength   types.Number `json:"maxLength"`
	Required    bool         `json:"required"`
}

// P2PAppraiseInfo holds a P2P order's review
type P2PAppraiseInfo struct {
	Anonymous       string     `json:"anonymous"`
	AppraiseContent string     `json:"appraiseContent"`
	AppraiseID      string     `json:"appraiseId"`
	AppraiseType    string     `json:"appraiseType"`
	ModifyFlag      string     `json:"modifyFlag"`
	UpdateDate      types.Time `json:"updateDate"`
}

// P2PPendingOrdersResponse is a page of pending P2P orders
type P2PPendingOrdersResponse struct {
	Count uint64            `json:"count"`
	Items []P2PPendingOrder `json:"items"`
}

// P2PPendingOrder is a pending P2P order summary
type P2PPendingOrder struct {
	ID string `json:"id"`
	// Side is 0 for a buy or 1 for a sell
	Side    uint64        `json:"side"`
	TokenID currency.Code `json:"tokenId"`
	// OrderType is ORIGIN for a regular order, SMALL_COIN for a HotSwap order or WEB3 for a Web3 order
	OrderType           string        `json:"orderType"`
	Amount              types.Number  `json:"amount"`
	CurrencyID          currency.Code `json:"currencyId"`
	Price               types.Number  `json:"price"`
	NotifyTokenQuantity types.Number  `json:"notifyTokenQuantity"`
	NotifyTokenID       currency.Code `json:"notifyTokenId"`
	Fee                 types.Number  `json:"fee"`
	TargetNickName      string        `json:"targetNickName"`
	TargetUserID        string        `json:"targetUserId"`
	TargetUserMaskID    string        `json:"targetUserMaskId"`
	// Status is 5 waiting for chain, 10 waiting for the buyer to pay, 20 waiting for the seller to release, 30
	// appealing, 90 waiting for the buyer to select a token, 100 objectioning or 110 waiting for an objection
	Status                 uint64       `json:"status"`
	SelfUnreadMessageCount types.Number `json:"selfUnreadMsgCount"`
	CreateDate             types.Time   `json:"createDate"`
	// TransferLastSeconds and AppealLastSeconds are the seconds left to pay and to appeal
	TransferLastSeconds types.Number      `json:"transferLastSeconds"`
	AppealLastSeconds   types.Number      `json:"appealLastSeconds"`
	UserID              string            `json:"userId"`
	SellerRealName      string            `json:"sellerRealName"`
	BuyerRealName       string            `json:"buyerRealName"`
	JudgeInfo           P2PJudgeInfo      `json:"judgeInfo"`
	UnreadMessageCount  types.Number      `json:"unreadMsgCount"`
	Extension           P2POrderExtension `json:"extension"`
	BulkOrderFlag       bool              `json:"bulkOrderFlag"`
}

// P2PMarkOrderAsPaidRequest holds the parameters for Mark Order as Paid
type P2PMarkOrderAsPaidRequest struct {
	OrderID string
	// PaymentType is the payment method type ID, such as "14" for a bank transfer
	PaymentType string
	PaymentID   string
}

// P2PReviewSellerCancelOrderApplyRequest holds the parameters for Review Seller Cancel Order Apply
type P2PReviewSellerCancelOrderApplyRequest struct {
	OrderID string
	// ExamineResult is PASS or REJECT
	ExamineResult string
	// RejectReason and RejectProofs are required to reject, such as buyerRefuseOrderCancelReason_haveMadePayment
	RejectReason string
	// RejectProofs holds proof image URLs or identifiers
	RejectProofs []string
	RejectRemark string
}

// P2PUserInfoResponse holds a P2P user's profile, from Get Account Information or Get Counterparty User Info
type P2PUserInfoResponse struct {
	NickName string `json:"nickName"`
	// DefaultNickName is set when the nickname was generated by the system
	DefaultNickName bool   `json:"defaultNickName"`
	WhiteFlag       uint64 `json:"whiteFlag"`
	ContactConfig   bool   `json:"contactConfig"`
	IsOnline        bool   `json:"isOnline"`
	Email           string `json:"email"`
	Mobile          string `json:"mobile"`
	// KYCLevel is documented as a string but recorded as a number
	KYCLevel       types.Number `json:"kycLevel"`
	LastLogoutTime types.Time   `json:"lastLogoutTime"`
	// RecentRate is the completion rate over the past 30 days; documented as both a string and an integer
	RecentRate           types.Number `json:"recentRate"`
	TotalFinishCount     uint64       `json:"totalFinishCount"`
	TotalFinishSellCount uint64       `json:"totalFinishSellCount"`
	TotalFinishBuyCount  uint64       `json:"totalFinishBuyCount"`
	RecentFinishCount    uint64       `json:"recentFinishCount"`
	// AverageReleaseTime and AverageTransferTime are in minutes
	AverageReleaseTime  types.Number `json:"averageReleaseTime"`
	AverageTransferTime types.Number `json:"averageTransferTime"`
	AccountCreateDays   uint64       `json:"accountCreateDays"`
	FirstTradeDays      uint64       `json:"firstTradeDays"`
	RealName            string       `json:"realName"`
	// RecentTradeAmount and TotalTradeAmount are in USDT
	RecentTradeAmount types.Number  `json:"recentTradeAmount"`
	TotalTradeAmount  types.Number  `json:"totalTradeAmount"`
	ExecuteNumber     uint64        `json:"executeNum"`
	OrderNumber       uint64        `json:"orderNum"`
	HasUnPostAd       types.Boolean `json:"hasUnPostAd"`
	RegisterTime      types.Time    `json:"registerTime"`
	// AuthStatus is 1 for a verified advertiser or 2 otherwise
	AuthStatus        uint64       `json:"authStatus"`
	KYCCountryCode    string       `json:"kycCountryCode"`
	Blocked           string       `json:"blocked"`
	GoodAppraiseRate  types.Number `json:"goodAppraiseRate"`
	GoodAppraiseCount uint64       `json:"goodAppraiseCount"`
	BadAppraiseCount  uint64       `json:"badAppraiseCount"`
	// AccountID is documented as an integer but recorded as a string
	AccountID                 types.PreciseNumber `json:"accountId"`
	PaymentCount              uint64              `json:"paymentCount"`
	ContactCount              uint64              `json:"contactCount"`
	RealNameMask              string              `json:"realNameMask"`
	VIPLevel                  uint64              `json:"vipLevel"`
	VIPProfit                 []string            `json:"vipProfit"`
	UserTag                   []string            `json:"userTag"`
	UserCancelCountLimit      uint64              `json:"userCancelCountLimit"`
	PaymentRealNameUneditable bool                `json:"paymentRealNameUneditable"`
	LostRoleAffected          bool                `json:"lostRoleAffected"`
	UserCurrentPrivilege      []string            `json:"userCurPrivilege"`
	UserType                  string              `json:"userType"`
	UserID                    string              `json:"userId"`
	RealNameEnglish           string              `json:"realNameEn"`
	CanSubOnline              bool                `json:"canSubOnline"`
	CurrentPrivilegeInfo      []P2PPrivilegeInfo  `json:"curPrivilegeInfo"`
	OpenAPISwitch             uint64              `json:"openApiSwitch"`
}

// P2PPrivilegeInfo is a P2P privilege a user holds and its setting
type P2PPrivilegeInfo struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// P2PUserPayment is a payment method added to the user's account
type P2PUserPayment struct {
	ID       string `json:"id"`
	RealName string `json:"realName"`
	// PaymentType is the payment method type ID; documented as a string but recorded as a number
	PaymentType     types.Number `json:"paymentType"`
	BankName        string       `json:"bankName"`
	BranchName      string       `json:"branchName"`
	AccountNo       string       `json:"accountNo"`
	QRCode          string       `json:"qrcode"`
	Visible         uint64       `json:"visible"`
	PayMessage      string       `json:"payMessage"`
	FirstName       string       `json:"firstName"`
	LastName        string       `json:"lastName"`
	SecondLastName  string       `json:"secondLastName"`
	Clabe           string       `json:"clabe"`
	DebitCardNumber string       `json:"debitCardNumber"`
	Mobile          string       `json:"mobile"`
	BusinessName    string       `json:"businessName"`
	Concept         string       `json:"concept"`
	// Online is "0" for a non-balance payment method or "1" for a balance payment method
	Online                    string             `json:"online"`
	CountNo                   string             `json:"countNo"`
	PaymentExt1               string             `json:"paymentExt1"`
	PaymentExt2               string             `json:"paymentExt2"`
	PaymentExt3               string             `json:"paymentExt3"`
	PaymentExt4               string             `json:"paymentExt4"`
	PaymentExt5               string             `json:"paymentExt5"`
	PaymentExt6               string             `json:"paymentExt6"`
	PaymentTemplateVersion    uint64             `json:"paymentTemplateVersion"`
	HasPaymentTemplateChanged bool               `json:"hasPaymentTemplateChanged"`
	PaymentConfigVO           P2PPaymentConfigVO `json:"paymentConfigVo"`
	RealNameVerified          bool               `json:"realNameVerified"`
	Channel                   string             `json:"channel"`
	CurrencyBalance           []currency.Code    `json:"currencyBalance"`
}

// P2PChatSessionListRequest holds the parameters for Get Chat Session List
type P2PChatSessionListRequest struct {
	// LastID is the ID of the last session already read; 0 returns the first page
	LastID uint64
	Size   uint64
	// ReadStatus is 0 for unread sessions, 1 for read sessions or 2 for both
	ReadStatus uint64
	// Type is GROUP or SINGLE
	Type string
}

// P2PChatSessionListResponse holds P2P chat sessions
type P2PChatSessionListResponse struct {
	ChatSession []P2PChatSession `json:"chatSession"`
}

// P2PChatSession is a P2P chat session
type P2PChatSession struct {
	// SessionName is the counterparty's nickname
	SessionName string `json:"sessionName"`
	SessionID   string `json:"sessionId"`
	Type        string `json:"type"`
	// ID is the cursor for the next page; documented as an integer but recorded as a string
	ID          types.PreciseNumber `json:"id"`
	UnreadCount types.Number        `json:"unreadCount"`
}

// P2PChatSessionIDResponse holds the encrypted session ID of a chat with a counterparty
type P2PChatSessionIDResponse struct {
	SessionID string `json:"sessionId"`
}

// P2PChatFileResponse holds an uploaded chat file's URL, which Send Chat Message sends as its message
type P2PChatFileResponse struct {
	UploadID string `json:"uploadId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
}

// P2PSendChatMessageRequest holds the parameters for Send Chat Message
type P2PSendChatMessageRequest struct {
	// Message is the text, or an uploaded file's URL, up to 1000 characters
	Message string
	// ContentType is str, pic, pdf or video
	ContentType string
	SessionID   string
	OrderID     string
}

// P2PChatMessageListRequest holds the parameters for Get Chat Message List
type P2PChatMessageListRequest struct {
	// LastID is the ID of the last message already read; 0 returns the latest messages
	LastID uint64
	// Limit is the page size, up to 30
	Limit     uint64
	SessionID string
}

// P2PChatMessageListResponse holds P2P chat messages, newest first
type P2PChatMessageListResponse struct {
	Messages []P2PChatMessage `json:"messages"`
}

// P2PChatMessage is a P2P chat message
type P2PChatMessage struct {
	// ID is the cursor for the next page; documented as an integer but recorded as a string
	ID               types.PreciseNumber `json:"id"`
	SendUserNickName string              `json:"sendUserNickName"`
	// ContentType is str, pic, pdf or video
	ContentType string `json:"contentType"`
	// Message is a JSON document; MessageContent decodes it
	Message    string     `json:"message"`
	CreateDate types.Time `json:"createDate"`
}

// P2PChatMessageContent is the decoded content of a P2P chat message
type P2PChatMessageContent struct {
	// Content is the text, or the file URL for files
	Content     string `json:"content"`
	MessageCode string `json:"msgCode"`
	MessageType string `json:"msgType"`
	FileName    string `json:"fileName"`
	// Size is the file size in bytes
	Size types.Number `json:"size"`
}
