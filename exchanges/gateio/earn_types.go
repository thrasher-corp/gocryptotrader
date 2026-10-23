package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DualInvestmentProductListRequest holds dual investment product list filters
type DualInvestmentProductListRequest struct {
	PlanID        uint64
	Coin          currency.Code
	Type          string // put to buy low or call to sell high
	QuoteCurrency currency.Code
	Sort          string // apy, short-period or multiple
	Page          uint64
	PageSize      uint64
}

// DualInvestmentPlanResponse holds an earn dual investment plan detail
type DualInvestmentPlanResponse struct {
	ID               uint64        `json:"id"`
	InstrumentName   string        `json:"instrument_name"`
	Type             string        `json:"type"`
	InvestCurrency   currency.Code `json:"invest_currency"`
	ExerciseCurrency currency.Code `json:"exercise_currency"`
	ExercisePrice    float64       `json:"exercise_price"`
	DeliveryTime     types.Time    `json:"delivery_time"`
	APYDisplay       types.Number  `json:"apy_display"`
	MinAmount        types.Number  `json:"min_amount"`
	StartTime        types.Time    `json:"start_time"`
	EndTime          types.Time    `json:"end_time"`
	Status           string        `json:"status"`
}

// DualInvestmentOrderListRequest holds dual investment order list filters
type DualInvestmentOrderListRequest struct {
	From   time.Time // Settlement time
	To     time.Time // Settlement time
	Type   string    // put to buy low or call to sell high
	Status string    // HOLD, REPAY, PROCESSING, SETTLEMENT_PROCESSING or ALL
	Coin   currency.Code
	Page   uint64
	Limit  uint64
}

// DualInvestmentOrderDetailResponse holds a dual investment order item detail
type DualInvestmentOrderDetailResponse struct {
	ID                 uint64        `json:"id"`
	PlanID             uint64        `json:"plan_id"`
	InvestAmount       types.Number  `json:"invest_amount"`
	SettlementAmount   types.Number  `json:"settlement_amount"`
	CreateTime         types.Time    `json:"create_time"`
	CompleteTime       types.Time    `json:"complete_time"`
	Status             string        `json:"status"`
	InvestCurrency     currency.Code `json:"invest_currency"`
	ExerciseCurrency   currency.Code `json:"exercise_currency"`
	SettlementCurrency currency.Code `json:"settlement_currency"`
	ExercisePrice      types.Number  `json:"exercise_price"`
	SettlementPrice    types.Number  `json:"settlement_price"`
	DeliveryTime       types.Time    `json:"delivery_time"`
	APYDisplay         types.Number  `json:"apy_display"`
	APYSettlement      types.Number  `json:"apy_settlement"`
	Text               string        `json:"text"`
}

// DualCurrencyEarningAssetResponse represents the user's dual investment assets and interest
type DualCurrencyEarningAssetResponse struct {
	UserAssetUSDT         types.Number `json:"user_asset_usdt"`
	UserAssetBTC          types.Number `json:"user_asset_btc"`
	UserTotalInterestUSDT types.Number `json:"user_total_interest_usdt"`
	UserTotalInterestBTC  types.Number `json:"user_total_interest_btc"`
}

// DualCurrencyEarlyRedemptionResponse represents a dual investment order's early redemption preview
type DualCurrencyEarlyRedemptionResponse struct {
	CreateTime          types.Time    `json:"create_timest"`
	DeliveryTime        types.Time    `json:"delivery_timest"`
	ExercisePrice       types.Number  `json:"exercise_price"`
	InvestAmount        types.Number  `json:"invest_amount"`
	InvestCurrency      currency.Code `json:"invest_currency"`
	Name                string        `json:"name"`
	OrderID             uint64        `json:"order_id"`
	RequestID           string        `json:"req_id"`
	RefundServiceCharge float64       `json:"refund_service_charge"`
	SettlePrice         types.Number  `json:"settle_price"`
	SettlementAmount    types.Number  `json:"settlement_amount"`
	SettlementCurrency  currency.Code `json:"settlement_currency"`
	SettlementInterest  types.Number  `json:"settlement_interest"`
	SettlementPrincipal types.Number  `json:"settlement_principle"`
	Type                string        `json:"type"`
	MoneyBackTime       types.Time    `json:"money_back_timest"`
}

// DualInvestmentOrderRequest holds a dual investment order request
type DualInvestmentOrderRequest struct {
	PlanID string       `json:"plan_id"`
	Amount types.Number `json:"amount"`
	Text   string       `json:"text,omitempty"` // Must start with t-
}

// DualModifyOrderReinvestRequest holds a dual investment order reinvestment change
type DualModifyOrderReinvestRequest struct {
	OrderID uint64 `json:"order_id"`
	// Status is 0 to switch reinvestment off and 1 to switch it on; it is always sent as 0 is a valid status
	Status            uint8                 `json:"status"`
	EffectiveDuration types.DurationSeconds `json:"effective_time_duration,omitempty"` // Whole seconds, defaulting to 24 hours
}

// DualRecommendedProjectResponse holds a recommended dual investment product
type DualRecommendedProjectResponse struct {
	ID               uint64        `json:"id"`
	Category         uint64        `json:"category"`
	Type             string        `json:"type"`
	InvestCurrency   currency.Code `json:"invest_currency"`
	ExerciseCurrency currency.Code `json:"exercise_currency"`
	APYDisplay       types.Number  `json:"apy_display"`
	ExercisePrice    types.Number  `json:"exercise_price"`
	DeliveryTime     types.Time    `json:"delivery_timest"`
	MinAmount        types.Number  `json:"min_amount"`
	MaxAmount        types.Number  `json:"max_amount"`
	MinCopies        uint64        `json:"min_copies"`
	MaxCopies        uint64        `json:"max_copies"`
	InvestDays       uint64        `json:"invest_days"`
	InvestHours      types.Number  `json:"invest_hours"`
}

// AutoInvestCoinItemResponse holds a currency supported by auto invest
type AutoInvestCoinItemResponse struct {
	Key          currency.Code `json:"key"`
	Value        string        `json:"value"`
	AssetIconURL string        `json:"asset_icon_url"`
	Sort         uint64        `json:"sort"`
}

// AutoInvestMinAmountRequest holds parameters for querying the minimum investment amount
type AutoInvestMinAmountRequest struct {
	Money currency.Code         `json:"money"`
	Items []*AutoInvestPlanItem `json:"items"`
}

// AutoInvestMinAmountResponse holds the minimum investment amount response
type AutoInvestMinAmountResponse struct {
	MinAmount types.Number `json:"min_amount"`
}

// AutoInvestPlanExecutionRecord holds a single plan execution record item
type AutoInvestPlanExecutionRecord struct {
	ID            uint64        `json:"id"`
	Type          string        `json:"type"`
	Money         currency.Code `json:"money"`
	Asset         currency.Code `json:"asset"`
	UserID        uint64        `json:"user_id"`
	PlanID        uint64        `json:"plan_id"`
	PlanVersion   uint64        `json:"plan_version"`
	Amount        types.Number  `json:"amount"`
	CreateTime    types.Time    `json:"create_time"`
	UpdateTime    types.Time    `json:"update_time"`
	Status        string        `json:"status"`
	StatusType    uint64        `json:"status_type"`
	Side          uint64        `json:"side"` // 2 buy, anything else sell
	StatusMessage string        `json:"status_message"`
	Detail        string        `json:"detail"`
}

// AutoInvestPlanExecutionRecordsResponse holds the paginated response for plan execution records
type AutoInvestPlanExecutionRecordsResponse struct {
	Page      uint64                           `json:"page"`
	PageSize  uint64                           `json:"page_size"`
	TotalPage uint64                           `json:"total_page"`
	Total     uint64                           `json:"total"`
	List      []*AutoInvestPlanExecutionRecord `json:"list"`
}

// AutoInvestOrderItemResponse holds an auto invest order detail item
type AutoInvestOrderItemResponse struct {
	ID           uint64        `json:"id"`
	Type         string        `json:"type"`
	Amount       types.Number  `json:"amount"`
	PlanID       uint64        `json:"plan_id"`
	Side         uint64        `json:"side"`
	Asset        currency.Code `json:"asset"`
	RecordID     uint64        `json:"record_id"`
	TotalMoney   types.Number  `json:"total_money"`
	Market       currency.Pair `json:"market"`
	Price        types.Number  `json:"price"`
	CreateTime   types.Time    `json:"create_time"`
	Total        types.Number  `json:"total"`
	FundFlow     string        `json:"fund_flow"`
	ErrorCode    uint64        `json:"error_code"`
	ErrorMessage string        `json:"error_msg"`
	Status       uint64        `json:"status"`
}

// AutoInvestConfigItemResponse holds an investment currency configuration item
type AutoInvestConfigItemResponse struct {
	Coin     currency.Code `json:"coin"`
	MaxLimit types.Number  `json:"max_limit"`
}

// AutoInvestPlanDetailsResponse holds auto invest plan details
type AutoInvestPlanDetailsResponse struct {
	ID         uint64                     `json:"id"`
	Version    uint64                     `json:"version"`
	Name       string                     `json:"name"`
	CreateTime types.Time                 `json:"create_time"`
	UpdateTime types.Time                 `json:"update_time"`
	UserID     uint64                     `json:"user_id"`
	Money      currency.Code              `json:"money"`
	Amount     types.Number               `json:"amount"`
	PeriodType string                     `json:"period_type"`
	PeriodDay  uint64                     `json:"period_day"`
	PeriodHour uint64                     `json:"period_hour"`
	Portfolio  []*AutoInvestPortfolioItem `json:"portfolio"`
	NextTime   types.Time                 `json:"next_time"`
	Period     uint64                     `json:"period"`
	FundSource string                     `json:"fund_source"`
	FundFlow   string                     `json:"fund_flow"`
}

// AutoInvestPortfolioItem holds a currency of an auto invest plan's portfolio
type AutoInvestPortfolioItem struct {
	Asset                currency.Code `json:"asset"`
	Ratio                types.Number  `json:"ratio"`
	CumulativeInvestment types.Number  `json:"cum_invest"`
	CumulativeHolding    types.Number  `json:"cum_hold"`
	CumulativeRedemption types.Number  `json:"cum_redeem"`
	AveragePrice         types.Number  `json:"avg_price"`
	RedeemStatus         uint64        `json:"redeem_status"`
	LendAmount           types.Number  `json:"lend_amount"`
}

// AutoInvestPlanListResponse holds the paginated response for auto invest plan list
type AutoInvestPlanListResponse struct {
	Page       uint64                           `json:"page"`
	PageSize   uint64                           `json:"page_size"`
	PageCount  uint64                           `json:"page_count"`
	TotalCount uint64                           `json:"total_count"`
	List       []*AutoInvestPlanDetailsResponse `json:"list"`
}

// AutoInvestPlanItem holds a single asset allocation item for plan creation
type AutoInvestPlanItem struct {
	Asset currency.Code `json:"asset"`
	Ratio types.Number  `json:"ratio"` // Share of the portfolio; every item's ratio must total 100
}

// CreateAutoInvestPlanRequest holds parameters for creating an auto invest plan
type CreateAutoInvestPlanRequest struct {
	PlanName        string `json:"plan_name,omitempty"`
	PlanDescription string `json:"plan_des,omitempty"`
	PlanPeriodType  string `json:"plan_period_type"` // daily, weekly, biweekly, monthly, hourly or 4-hourly
	// PlanPeriodDay is the day of the month (1-30) for monthly plans or the weekday (1-7 from Monday) for weekly and
	// biweekly plans; it is always sent as the API requires it even when the period type ignores it
	PlanPeriodDay uint64 `json:"plan_period_day"`
	// PlanPeriodHour is the hour from 0 to 23; it is always sent as 0 is a valid hour
	PlanPeriodHour uint64                `json:"plan_period_hour"`
	PlanMoney      currency.Code         `json:"plan_money"` // USDT or BTC
	PlanAmount     types.Number          `json:"plan_amount"`
	Type           uint8                 `json:"type,omitempty"` // 0 normal creation, 1 quick investment
	Items          []*AutoInvestPlanItem `json:"items"`
	FundSource     string                `json:"fund_source,omitempty"` // spot or earn, defaulting to spot
	FundFlow       string                `json:"fund_flow,omitempty"`   // auto_invest or earn, defaulting to auto_invest
}

// AutoInvestPlanResponse represents an auto-invest plan response.
type AutoInvestPlanResponse struct {
	ID         uint64        `json:"id"`
	Amount     types.Number  `json:"amount"`
	Money      currency.Code `json:"money"`
	NextTime   types.Time    `json:"next_time"`
	PeriodType string        `json:"period_type"`
	PeriodDay  uint64        `json:"period_day"`
	PeriodHour uint64        `json:"period_hour"`
	FundFlow   string        `json:"fund_flow"`
	FundSource string        `json:"fund_source"`
}

// AutoInvestPlanUpdateRequest holds parameters for updating an auto invest plan
type AutoInvestPlanUpdateRequest struct {
	PlanID     uint64 `json:"plan_id"`
	FundSource string `json:"fund_source,omitempty"` // spot or earn, defaulting to spot
	FundFlow   string `json:"fund_flow,omitempty"`   // auto_invest or earn, defaulting to auto_invest
}

// AutoInvestPlanStopRequest holds parameters for stopping an auto invest plan
type AutoInvestPlanStopRequest struct {
	PlanID uint64 `json:"plan_id"`
}

// AutoInvestPlanAddPositionRequest holds parameters for adding a position to an auto invest plan
type AutoInvestPlanAddPositionRequest struct {
	PlanID uint64       `json:"plan_id"`
	Amount types.Number `json:"amount"`
}

// FixedTermProduct holds a fixed-term earn product
type FixedTermProduct struct {
	ID                uint64        `json:"id"`
	Name              string        `json:"name"`
	Asset             currency.Code `json:"asset"`
	LockUpPeriodDays  uint64        `json:"lock_up_period"`
	MinLendAmount     types.Number  `json:"min_lend_amount"`
	UserMaxLendAmount types.Number  `json:"user_max_lend_amount"`
	TotalLendAmount   types.Number  `json:"total_lend_amount"`
	YearRate          types.Number  `json:"year_rate"`
	Type              uint8         `json:"type"` // 1 regular, 2 VIP
	PreRedeem         types.Boolean `json:"pre_redeem"`
	Reinvest          types.Boolean `json:"reinvest"`
	RedeemAccount     uint8         `json:"redeem_account"` // Whether fixed-to-flexible conversion is supported
	MinVIP            uint8         `json:"min_vip"`
	MaxVIP            uint8         `json:"max_vip"`
	Status            uint8         `json:"status"` // 1 unlisted, 2 listed, 3 delisted
	CreateTime        time.Time     `json:"create_time"`
	UserMaxLendVolume types.Number  `json:"user_max_lend_volume"`
	UserTotalAmount   types.Number  `json:"user_total_amount"`
	SaleStatus        uint8         `json:"sale_status"` // 1 on sale, 2 sold out
}

// FixedTermProductsResponse holds a page of fixed-term earn products
type FixedTermProductsResponse struct {
	List  []*FixedTermProduct `json:"list"`
	Total uint64              `json:"total"`
}

// FixedTermProductSimple holds a fixed-term earn product from a single currency's product list
type FixedTermProductSimple struct {
	ID               uint64        `json:"id"`
	Asset            currency.Code `json:"asset"`
	LockUpPeriodDays uint64        `json:"lock_up_period"`
	YearRate         types.Number  `json:"year_rate"`
	Type             uint8         `json:"type"` // 1 regular, 2 VIP
	PreRedeem        types.Boolean `json:"pre_redeem"`
	Reinvest         types.Boolean `json:"reinvest"`
	SimpleEarn       types.Boolean `json:"simple_earn"` // Whether fixed-to-flexible conversion is supported
	// RedeemAccount is returned by the live API in place of the documented simple_earn
	RedeemAccount uint8 `json:"redeem_account"`
	MinVIP        uint8 `json:"min_vip"`
	MaxVIP        uint8 `json:"max_vip"`
	SaleStatus    uint8 `json:"sale_status"` // 1 on sale, 2 sold out
}

// FixedTermProductsByAssetResponse holds a single currency's fixed-term earn products
type FixedTermProductsByAssetResponse struct {
	List []*FixedTermProductSimple `json:"list"`
}

// FixedTermSubscriptionRequest represents a fixed-term earn product subscription
type FixedTermSubscriptionRequest struct {
	ProductID uint64       `json:"product_id"`
	Amount    types.Number `json:"amount"`
	YearRate  types.Number `json:"year_rate,omitempty"`
	// ReinvestStatus is 1 to renew automatically at maturity; it is always sent so 0 explicitly disables renewal
	ReinvestStatus    uint8  `json:"reinvest_status"`
	RedeemAccountType uint8  `json:"redeem_account_type,omitempty"` // 1 spot account
	FinancialRateID   uint64 `json:"financial_rate_id,omitempty"`   // Interest rate boost coupon to apply
	SubBusiness       uint64 `json:"sub_business,omitempty"`
}

// FixedTermBusinessFilterRequest narrows fixed-term earn queries to a business line
type FixedTermBusinessFilterRequest struct {
	Business    uint8  `json:"business"` // 1 regular, 2 VIP
	SubBusiness uint64 `json:"sub_business"`
}

// FixedTermSubscriptionOrdersRequest holds fixed-term earn subscription order filters
type FixedTermSubscriptionOrdersRequest struct {
	ProductID      uint64
	OrderID        uint64
	Asset          currency.Code
	OrderType      uint8 // 1 current orders, 2 historical orders
	Page           uint64
	Limit          uint64
	SubBusiness    uint64
	BusinessFilter []FixedTermBusinessFilterRequest
}

// FixedTermLendOrder holds a fixed-term earn subscription order
type FixedTermLendOrder struct {
	ID                   uint64                `json:"id"`
	Business             uint8                 `json:"business"` // 1 regular, 2 VIP
	OrderID              uint64                `json:"order_id"`
	UserID               uint64                `json:"user_id"`
	Asset                currency.Code         `json:"asset"`
	ProductID            uint64                `json:"product_id"`
	LockUpPeriodDays     uint64                `json:"lock_up_period"`
	Principal            types.Number          `json:"principal"`
	YearRate             types.Number          `json:"year_rate"`
	ProductType          uint8                 `json:"product_type"` // 1 regular, 2 VIP
	Interest             types.Number          `json:"interest"`
	Status               uint8                 `json:"status"` // 1 holding, 2 redeemed, 3 matured, 4 settled
	ReinvestStatus       uint8                 `json:"reinvest_status"`
	RedeemAccountType    uint8                 `json:"redeem_account_type"`
	OriginOrder          string                `json:"origin_order"` // Previous order of an automatically renewed order
	RedeemType           uint8                 `json:"redeem_type"`  // 1 early redemption, 2 redemption at maturity
	RedeemTime           time.Time             `json:"redeem_time"`
	FinishTime           time.Time             `json:"finish_time"`
	CreateTime           time.Time             `json:"create_time"`
	YearRatePercent      types.Number          `json:"year_rate_perent"`
	TotalYearRatePercent types.Number          `json:"total_year_rate_percent"`
	TotalInterest        types.Number          `json:"total_interest"`
	ProductInfo          *FixedTermProductInfo `json:"product_info"`
	BonusInfo            *FixedTermBonusInfo   `json:"bonus_info"`
	CouponInfo           *FixedTermCouponInfo  `json:"coupon_info"`
	RedeemAt             types.Time            `json:"redeem_at"`
	FinishAt             types.Time            `json:"finish_at"`
	CreateAt             types.Time            `json:"create_at"`
	Icon                 string                `json:"icon"`
}

// FixedTermProductInfo holds a fixed-term earn subscription order's product configuration
type FixedTermProductInfo struct {
	PreRedeem     types.Boolean `json:"pre_redeem"`
	Reinvest      types.Boolean `json:"reinvest"`
	RedeemAccount uint8         `json:"redeem_account"`
	MinVIP        uint8         `json:"min_vip"`
	MaxVIP        uint8         `json:"max_vip"`
}

// FixedTermBonusInfo holds a fixed-term earn bonus reward campaign
type FixedTermBonusInfo struct {
	ID                    uint64                `json:"id"`
	ProductID             uint64                `json:"product_id"`
	Asset                 currency.Code         `json:"asset"`
	BonusAsset            currency.Code         `json:"bonus_asset"`
	KYCLimit              string                `json:"kyc_limit"` // Comma separated KYC levels
	LadderAPR             []*FixedTermLadderAPR `json:"ladder_apr"`
	TotalBonusAmount      types.Number          `json:"total_bonus_amount"`
	UserTotalBonusAmount  types.Number          `json:"user_total_bonus_amount"`
	Status                uint8                 `json:"status"` // 1 unlisted, 2 listed, 3 delisted
	StartTime             time.Time             `json:"start_time"`
	EndTime               time.Time             `json:"end_time"`
	CreateTime            time.Time             `json:"create_time"`
	StartAt               types.Time            `json:"start_at"`
	EndAt                 types.Time            `json:"end_at"`
	TotalIssuedAmount     types.Number          `json:"total_issued_amount"`
	UserTotalIssuedAmount types.Number          `json:"user_total_issued_amount"`
	BonusAssetPrice       types.Number          `json:"bonus_asset_price"`
	ProductAssetPrice     types.Number          `json:"product_asset_price"`
	ProductYearRate       types.Number          `json:"product_year_rate"`
}

// FixedTermLadderAPR holds a tier of a fixed-term earn bonus reward campaign
type FixedTermLadderAPR struct {
	APR   types.Number `json:"apr"`
	Left  types.Number `json:"left"`
	Right types.Number `json:"right"`
}

// FixedTermCouponInfo holds a fixed-term earn interest rate boost coupon
type FixedTermCouponInfo struct {
	ID              uint64        `json:"id"`
	Business        uint8         `json:"business"`
	UserID          uint64        `json:"user_id"`
	Asset           currency.Code `json:"asset"`
	OrderID         uint64        `json:"order_id"`
	FinancialRateID uint64        `json:"financial_rate_id"`
	BuyLimitLow     types.Number  `json:"buy_limit_low"`
	BuyLimitHigh    types.Number  `json:"buy_limit_high"`
	RateDays        uint64        `json:"rate_day"`
	RateRatio       types.Number  `json:"rate_ratio"`
	CouponDays      uint64        `json:"coupon_days"`
	CouponPrincipal types.Number  `json:"coupon_principal"`
	CouponYearRate  types.Number  `json:"coupon_year_rate"`
	CouponInterest  types.Number  `json:"coupon_interest"`
	Status          uint8         `json:"status"` // 1 active, 2 settled
	FinishTime      time.Time     `json:"finish_time"`
	CreateTime      time.Time     `json:"create_time"`
}

// FixedTermSubscriptionOrdersResponse holds a page of fixed-term earn subscription orders
type FixedTermSubscriptionOrdersResponse struct {
	List  []*FixedTermLendOrder `json:"list"`
	Total uint64                `json:"total"`
}

// FixedTermHistoryRequest holds fixed-term earn history filters
type FixedTermHistoryRequest struct {
	ProductID      uint64
	OrderID        string
	Asset          currency.Code
	Type           uint8 // 1 subscription, 2 redemption, 3 interest, 4 bonus reward
	Page           uint64
	Limit          uint64
	StartTime      time.Time
	EndTime        time.Time
	SubBusiness    uint64
	BusinessFilter []FixedTermBusinessFilterRequest
}

// FixedTermHistoryRecord holds a fixed-term earn history record
type FixedTermHistoryRecord struct {
	ID               uint64        `json:"id"`
	OrderID          uint64        `json:"order_id"`
	UserID           uint64        `json:"user_id"`
	Asset            currency.Code `json:"asset"`
	UniqueTime       string        `json:"uniq_time"` // Date the record belongs to
	BonusID          uint64        `json:"bonus_id"`
	ProductID        uint64        `json:"product_id"`
	BonusAsset       currency.Code `json:"bonus_asset"`
	TotalPrincipal   types.Number  `json:"total_principal"`
	Amount           types.Number  `json:"amount"`
	AssetPrice       types.Number  `json:"asset_price"`
	Status           uint64        `json:"status"`
	Detail           string        `json:"detail"`
	CreateTime       time.Time     `json:"create_time"`
	CreateAt         types.Time    `json:"create_at"`
	LockUpPeriodDays uint64        `json:"lock_up_period"`
}

// FixedTermHistoryResponse holds a page of fixed-term earn history records
type FixedTermHistoryResponse struct {
	List  []*FixedTermHistoryRecord `json:"list"`
	Total uint64                    `json:"total"`
}
