package bybit

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Advanced Earn product categories, the category parameter of the shared Advanced Earn endpoints
const (
	AdvancedEarnCategoryDualAssets    = "DualAssets"
	AdvancedEarnCategorySmartLeverage = "SmartLeverage"
	AdvancedEarnCategoryDoubleWin     = "DoubleWin"
	AdvancedEarnCategoryDiscountBuy   = "DiscountBuy"
)

// Advanced Earn order types; Redeem is an early redemption, offered by SmartLeverage and DoubleWin
const (
	AdvancedEarnOrderTypeStake  = "Stake"
	AdvancedEarnOrderTypeRedeem = "Redeem"
)

// AdvancedEarnProductInfoRequest holds the parameters of Get Product Info for one Advanced Earn category
type AdvancedEarnProductInfoRequest struct {
	Category string
	// Coin filters DualAssets products by coin and the other categories by underlying asset
	Coin currency.Code
	// Duration filters by product duration, such as 8h or 1d; DiscountBuy does not document it
	Duration string
}

// AdvancedEarnProductInfoResponse holds the products of the requested category. Each category returns different
// fields, so its products decode into its own type and only the requested category's slice is populated
type AdvancedEarnProductInfoResponse struct {
	Category      string
	DualAssets    []DualAssetsProduct
	SmartLeverage []SmartLeverageProduct
	DoubleWin     []DoubleWinProduct
	DiscountBuy   []DiscountBuyProduct
}

// UnmarshalJSON decodes each listed product into the type of the category it names
func (r *AdvancedEarnProductInfoResponse) UnmarshalJSON(data []byte) error {
	var result advancedEarnList
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*r = AdvancedEarnProductInfoResponse{Category: result.Category}
	return decodeAdvancedEarnElements(result.Category, result.List, map[string]func(json.RawMessage) error{
		AdvancedEarnCategoryDualAssets:    advancedEarnElementAppender(&r.DualAssets),
		AdvancedEarnCategorySmartLeverage: advancedEarnElementAppender(&r.SmartLeverage),
		AdvancedEarnCategoryDoubleWin:     advancedEarnElementAppender(&r.DoubleWin),
		AdvancedEarnCategoryDiscountBuy:   advancedEarnElementAppender(&r.DiscountBuy),
	})
}

// advancedEarnList is the result of the shared Advanced Earn list endpoints before its elements are decoded per
// category
type advancedEarnList struct {
	Category       string            `json:"category"`
	List           []json.RawMessage `json:"list"`
	NextPageCursor string            `json:"nextPageCursor"`
}

// advancedEarnElementCategory reads the category every Advanced Earn list element names
type advancedEarnElementCategory struct {
	Category string `json:"category"`
}

// decodeAdvancedEarnElements decodes each element with the decoder registered for its category. Elements are
// dispatched on their own category because DiscountBuy replies leave the result's category empty; an element without
// one falls back to the result's
func decodeAdvancedEarnElements(resultCategory string, elements []json.RawMessage, decoders map[string]func(json.RawMessage) error) error {
	for _, element := range elements {
		var probe advancedEarnElementCategory
		if err := json.Unmarshal(element, &probe); err != nil {
			return err
		}
		category := probe.Category
		if category == "" {
			category = resultCategory
		}
		decode, ok := decoders[category]
		if !ok {
			return fmt.Errorf("%w: %q", errUnsupportedAdvancedEarnCategory, category)
		}
		if err := decode(element); err != nil {
			return err
		}
	}
	return nil
}

// advancedEarnElementAppender returns a decoder that appends each element it decodes to s
func advancedEarnElementAppender[T any](s *[]T) func(json.RawMessage) error {
	return func(element json.RawMessage) error {
		var v T
		if err := json.Unmarshal(element, &v); err != nil {
			return err
		}
		*s = append(*s, v)
		return nil
	}
}

// DualAssetsProduct is a Dual Assets product
type DualAssetsProduct struct {
	Category                   string        `json:"category"`
	ProductID                  string        `json:"productId"`
	BaseCoin                   currency.Code `json:"baseCoin"`
	QuoteCoin                  currency.Code `json:"quoteCoin"`
	ExpectReceiveAt            types.Time    `json:"expectReceiveAt"`
	Duration                   string        `json:"duration"`
	Status                     string        `json:"status"`
	IsVIPProduct               bool          `json:"isVipProduct"`
	SubscribeStartAt           types.Time    `json:"subscribeStartAt"`
	SubscribeEndAt             types.Time    `json:"subscribeEndAt"`
	ApplyStartAt               types.Time    `json:"applyStartAt"` // Interest accrual start time
	SettlementTime             types.Time    `json:"settlementTime"`
	MinPurchaseQuoteAmount     types.Number  `json:"minPurchaseQuoteAmount"`
	MinPurchaseBaseAmount      types.Number  `json:"minPurchaseBaseAmount"`
	RemainingAmountQuote       types.Number  `json:"remainingAmountQuote"` // Remaining quota of the Buy Low direction
	RemainingAmountBase        types.Number  `json:"remainingAmountBase"`  // Remaining quota of the Sell High direction
	OrderPrecisionDigitalQuote uint64        `json:"orderPrecisionDigitalQuote"`
	OrderPrecisionDigitalBase  uint64        `json:"orderPrecisionDigitalBase"`
}

// SmartLeverageProduct is a Smart Leverage product
type SmartLeverageProduct struct {
	Category              string        `json:"category"`
	ProductID             string        `json:"productId"`
	InvestCoin            currency.Code `json:"investCoin"`
	UnderlyingAsset       currency.Code `json:"underlyingAsset"`
	Direction             string        `json:"direction"`
	Leverage              types.Number  `json:"leverage"`
	Duration              string        `json:"duration"`
	SubscribeStartAt      types.Time    `json:"subscribeStartAt"`
	SubscribeEndAt        types.Time    `json:"subscribeEndAt"`
	SettlementTime        types.Time    `json:"settlementTime"`
	ExpectReceiveAt       types.Time    `json:"expectReceiveAt"`
	MinPurchaseAmount     types.Number  `json:"minPurchaseAmount"`
	RemainingAmount       types.Number  `json:"remainingAmount"`
	OrderPrecisionDigital uint64        `json:"orderPrecisionDigital"`
}

// DoubleWinProduct is a Double Win product. Fixed range products (IsRFQProduct false) fill the price buffers; RFQ
// products fill the deviation ratios and the price tick size
type DoubleWinProduct struct {
	Category              string        `json:"category"`
	ProductID             string        `json:"productId"`
	InvestCoin            currency.Code `json:"investCoin"`
	UnderlyingAsset       currency.Code `json:"underlyingAsset"`
	Duration              string        `json:"duration"`
	SubscribeStartAt      types.Time    `json:"subscribeStartAt"`
	SubscribeEndAt        types.Time    `json:"subscribeEndAt"`
	SettlementTime        types.Time    `json:"settlementTime"`
	ExpectReceiveAt       types.Time    `json:"expectReceiveAt"`
	MinPurchaseAmount     types.Number  `json:"minPurchaseAmount"`
	OrderPrecisionDigital uint64        `json:"orderPrecisionDigital"`
	IsRFQProduct          bool          `json:"isRfqProduct"`
	LowerPriceBuffer      types.Number  `json:"lowerPriceBuffer"`
	UpperPriceBuffer      types.Number  `json:"upperPriceBuffer"`
	MinDeviationRatio     types.Number  `json:"minDeviationRatio"`
	MaxDeviationRatio     types.Number  `json:"maxDeviationRatio"`
	PriceTickSize         types.Number  `json:"priceTickSize"`
}

// DiscountBuyProduct is a Discount Buy product
type DiscountBuyProduct struct {
	Category              string        `json:"category"`
	ProductID             string        `json:"productId"`
	Coin                  currency.Code `json:"coin"`
	UnderlyingAsset       currency.Code `json:"underlyingAsset"`
	SettlementTime        types.Time    `json:"settlementTime"`
	Duration              string        `json:"duration"`
	IsVIPProduct          bool          `json:"isVipProduct"`
	SubscribeStartAt      types.Time    `json:"subscribeStartAt"`
	SubscribeEndAt        types.Time    `json:"subscribeEndAt"`
	MinPurchaseAmount     types.Number  `json:"minPurchaseAmount"`
	RemainingAmount       types.Number  `json:"remainingAmount"`
	OrderPrecisionDigital uint64        `json:"orderPrecisionDigital"`
	ExpectReceiveAt       types.Time    `json:"expectReceiveAt"`
}

// AdvancedEarnProductQuoteResponse holds the quotes of the requested category. Each category answers with a
// different result shape (DualAssets a list, SmartLeverage and DoubleWin one quote, DiscountBuy a list of offers), so
// only the requested category's field is populated
type AdvancedEarnProductQuoteResponse struct {
	// Category is empty for DiscountBuy, whose result carries the category on each offer instead
	Category      string
	DualAssets    []DualAssetsProductQuote
	SmartLeverage *SmartLeverageProductQuote
	DoubleWin     *DoubleWinProductQuote
	DiscountBuy   []DiscountBuyProductQuote
}

// UnmarshalJSON decodes the result into the field of the category it belongs to
func (r *AdvancedEarnProductQuoteResponse) UnmarshalJSON(data []byte) error {
	var result advancedEarnQuoteResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*r = AdvancedEarnProductQuoteResponse{Category: result.Category}
	if result.Offers != nil {
		return json.Unmarshal(result.Offers, &r.DiscountBuy)
	}
	switch result.Category {
	case AdvancedEarnCategoryDualAssets:
		if result.List == nil {
			return nil
		}
		return json.Unmarshal(result.List, &r.DualAssets)
	case AdvancedEarnCategorySmartLeverage:
		return json.Unmarshal(data, &r.SmartLeverage)
	case AdvancedEarnCategoryDoubleWin:
		return json.Unmarshal(data, &r.DoubleWin)
	}
	return fmt.Errorf("%w: %q", errUnsupportedAdvancedEarnCategory, result.Category)
}

// advancedEarnQuoteResult holds the keys that tell the Get Product Quote result shapes apart
type advancedEarnQuoteResult struct {
	Category string          `json:"category"`
	List     json.RawMessage `json:"list"`
	Offers   json.RawMessage `json:"offers"`
}

// DualAssetsProductQuote holds the prices on offer for a Dual Assets product
type DualAssetsProductQuote struct {
	ProductID     string                 `json:"productId"`
	CurrentPrice  types.Number           `json:"currentPrice"`
	BuyLowPrice   []DualAssetsQuotePrice `json:"buyLowPrice"`
	SellHighPrice []DualAssetsQuotePrice `json:"sellHighPrice"`
}

// DualAssetsQuotePrice is a target price on offer for a Dual Assets direction
type DualAssetsQuotePrice struct {
	SelectPrice         types.Number `json:"selectPrice"`
	APYE8               types.Number `json:"apyE8"` // APY scaled by 1e8
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
	ExpiredAt           types.Time   `json:"expiredAt"`
}

// SmartLeverageProductQuote is the quote for a Smart Leverage product. BreakevenPrice, ExpireAt and
// MaxInvestmentAmount are empty when no institutional quote is available
type SmartLeverageProductQuote struct {
	Category            string       `json:"category"`
	ProductID           string       `json:"productId"`
	BreakevenPrice      types.Number `json:"breakevenPrice"`
	CurrentPrice        types.Number `json:"currentPrice"`
	ExpireAt            types.Time   `json:"expireAt"`
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
}

// DoubleWinProductQuote is the quote for a fixed price range Double Win product. Leverage, ExpireTime and
// MaxInvestmentAmount are empty when no quote is active
type DoubleWinProductQuote struct {
	Category            string       `json:"category"`
	ProductID           string       `json:"productId"`
	Leverage            types.Number `json:"leverage"` // Current maximum leverage
	ExpireTime          types.Time   `json:"expireTime"`
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
	CurrentPrice        types.Number `json:"currentPrice"` // Index price, to send as the order's initial price
}

// DiscountBuyProductQuote is a quote offered for a Discount Buy product
type DiscountBuyProductQuote struct {
	ProductID           string       `json:"productId"`
	CurrentPrice        types.Number `json:"currentPrice"`
	PurchasePrice       types.Number `json:"purchasePrice"`
	KnockoutPrice       types.Number `json:"knockoutPrice"`
	KnockoutCouponE8    types.Number `json:"knockoutCouponE8"` // Annualised interest rate scaled by 1e8
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
	// InstitutionUID identifies the quoting market maker, and must be sent unchanged with the order
	InstitutionUID string `json:"instUid"`
	// ExpiredAt is zero when the institution set no expiry
	ExpiredAt types.Time `json:"expiredAt"`
	Category  string     `json:"category"`
}

// AdvancedEarnPlaceOrderRequest holds the parameters of Place Order for any Advanced Earn category. Set the extra
// parameters of the category and order type: DualAssetsExtra for DualAssets, SmartLeverageStakeExtra or
// SmartLeverageRedeemExtra for SmartLeverage, DoubleWinStakeExtra or DoubleWinRedeemExtra for DoubleWin, and
// DiscountBuyExtra for DiscountBuy
type AdvancedEarnPlaceOrderRequest struct {
	Category  string
	ProductID string
	// OrderType is Stake, or Redeem for an early SmartLeverage or DoubleWin redemption
	OrderType string
	// Amount is required for Stake orders
	Amount float64
	// AccountType is FUND or UNIFIED, required for Stake orders and SmartLeverage redemptions
	AccountType string
	// Coin is required for Stake orders
	Coin        currency.Code
	OrderLinkID string

	DualAssetsExtra          *DualAssetsOrderExtra
	SmartLeverageStakeExtra  *SmartLeverageStakeExtra
	SmartLeverageRedeemExtra *AdvancedEarnRedeemExtra
	DoubleWinStakeExtra      *DoubleWinStakeExtra
	DoubleWinRedeemExtra     *AdvancedEarnRedeemExtra
	DiscountBuyExtra         *DiscountBuyOrderExtra
	// InterestCard applies an interest bonus card to a DualAssets order
	InterestCard *AdvancedEarnInterestCard
}

// DualAssetsOrderExtra holds the quote a Dual Assets order accepts, from Get Product Quote or the offers stream
type DualAssetsOrderExtra struct {
	// OrderDirection is BuyLow or SellHigh
	OrderDirection string
	SelectPrice    float64
	// APYE8 is the lowest APY accepted, scaled by 1e8; the order fails if the confirmed APY is lower
	APYE8 uint64
}

// SmartLeverageStakeExtra holds the prices a Smart Leverage stake order is placed at
type SmartLeverageStakeExtra struct {
	// InitialPrice is the underlying price the order expects; the order fails if the actual price deviates by more
	// than 5%
	InitialPrice   float64
	BreakevenPrice float64
}

// AdvancedEarnRedeemExtra holds the parameters of an early redemption of a Smart Leverage or Double Win position
type AdvancedEarnRedeemExtra struct {
	PositionID string
	// EstimatedRedeemAmount comes from Get Redeem Estimated Amount, whose result Bybit caches for 10 minutes
	EstimatedRedeemAmount float64
	// IsSlippageProtected fails the redemption when the actual amount deviates significantly from the estimate
	IsSlippageProtected bool
}

// DoubleWinStakeExtra holds the quote a Double Win stake order accepts. LowerPrice and UpperPrice apply to RFQ
// products only and must match the Get Custom Product Quote request
type DoubleWinStakeExtra struct {
	Leverage     float64
	InitialPrice float64
	LowerPrice   float64
	UpperPrice   float64
}

// DiscountBuyOrderExtra holds the Get Product Quote values a Discount Buy order accepts
type DiscountBuyOrderExtra struct {
	InitialPrice     float64
	PurchasePrice    float64
	KnockoutPrice    float64
	KnockoutCouponE8 uint64
	// SettleType is Base to receive the underlying asset when exercised, or Quote to receive USDT
	SettleType     string
	InstitutionUID string
}

// AdvancedEarnInterestCard identifies an interest bonus card
type AdvancedEarnInterestCard struct {
	AwardID  uint64
	SpecCode string
}

// AdvancedEarnOrderResponse holds the identifiers of an accepted Advanced Earn order, which Bybit processes
// asynchronously
type AdvancedEarnOrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// AdvancedEarnPositionInfoRequest holds the parameters of Get Position Info for one Advanced Earn category
type AdvancedEarnPositionInfoRequest struct {
	Category  string
	ProductID string
	// Coin filters the positions by coin; SmartLeverage and DoubleWin filter on the underlying asset
	Coin   currency.Code
	Limit  uint64
	Cursor string
}

// AdvancedEarnPositionInfoResponse holds a page of the requested category's active positions; only that category's
// slice is populated
type AdvancedEarnPositionInfoResponse struct {
	Category       string
	NextPageCursor string
	DualAssets     []DualAssetsPosition
	SmartLeverage  []SmartLeveragePosition
	DoubleWin      []DoubleWinPosition
	DiscountBuy    []DiscountBuyPosition
}

// UnmarshalJSON decodes each listed position into the type of the category it names
func (r *AdvancedEarnPositionInfoResponse) UnmarshalJSON(data []byte) error {
	var result advancedEarnList
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*r = AdvancedEarnPositionInfoResponse{Category: result.Category, NextPageCursor: result.NextPageCursor}
	return decodeAdvancedEarnElements(result.Category, result.List, map[string]func(json.RawMessage) error{
		AdvancedEarnCategoryDualAssets:    advancedEarnElementAppender(&r.DualAssets),
		AdvancedEarnCategorySmartLeverage: advancedEarnElementAppender(&r.SmartLeverage),
		AdvancedEarnCategoryDoubleWin:     advancedEarnElementAppender(&r.DoubleWin),
		AdvancedEarnCategoryDiscountBuy:   advancedEarnElementAppender(&r.DiscountBuy),
	})
}

// DualAssetsPosition is an active Dual Assets position
type DualAssetsPosition struct {
	PositionID string        `json:"positionId"`
	ProductID  string        `json:"productId"`
	Category   string        `json:"category"`
	BaseCoin   currency.Code `json:"baseCoin"`
	QuoteCoin  currency.Code `json:"quoteCoin"`
	// InvestCoin is the quote coin for BuyLow and the base coin for SellHigh
	InvestCoin         currency.Code `json:"investCoin"`
	Amount             types.Number  `json:"amount"`
	APYE8              types.Number  `json:"apyE8"` // Confirmed APY scaled by 1e8
	Direction          string        `json:"direction"`
	TargetPrice        types.Number  `json:"targetPrice"`
	SettlementTime     types.Time    `json:"settlementTime"`
	Status             string        `json:"status"`
	OrderID            string        `json:"orderId"`
	Duration           string        `json:"duration"`
	ExpectReturnCoin   currency.Code `json:"expectReturnCoin"`
	ExpectReturnAmount types.Number  `json:"expectReturnAmount"`
	AccountType        string        `json:"accountType"`
	ToAccountType      string        `json:"toAccountType"`
	YieldStartAt       types.Time    `json:"yieldStartAt"`
	YieldEndAt         types.Time    `json:"yieldEndAt"`
}

// SmartLeveragePosition is an active Smart Leverage position
type SmartLeveragePosition struct {
	PositionID      string        `json:"positionId"`
	ProductID       string        `json:"productId"`
	Category        string        `json:"category"`
	InvestCoin      currency.Code `json:"investCoin"`
	Amount          types.Number  `json:"amount"`
	Duration        string        `json:"duration"`
	SettlementTime  types.Time    `json:"settlementTime"`
	AccountType     string        `json:"accountType"`
	OrderID         string        `json:"orderId"`
	UnderlyingAsset currency.Code `json:"underlyingAsset"`
	Direction       string        `json:"direction"`
	Leverage        types.Number  `json:"leverage"`
	BreakevenPrice  types.Number  `json:"breakevenPrice"`
	InitialPrice    types.Number  `json:"initialPrice"`
	Status          string        `json:"status"`
	CreatedTime     types.Time    `json:"createdTime"`
	// Redeemable is false while redemption is disabled, a redeem order is pending, or settlement is under 60 minutes
	// away
	Redeemable  bool   `json:"redeemable"`
	OrderLinkID string `json:"orderLinkId"`
}

// DoubleWinPosition is an active Double Win position
type DoubleWinPosition struct {
	PositionID      string        `json:"positionId"`
	ProductID       string        `json:"productId"`
	Category        string        `json:"category"`
	InvestCoin      currency.Code `json:"investCoin"`
	UnderlyingAsset currency.Code `json:"underlyingAsset"`
	Amount          types.Number  `json:"amount"`
	Leverage        types.Number  `json:"leverage"`
	InitialPrice    types.Number  `json:"initialPrice"`
	LowerPrice      types.Number  `json:"lowerPrice"`
	UpperPrice      types.Number  `json:"upperPrice"`
	Duration        string        `json:"duration"`
	SettlementTime  types.Time    `json:"settlementTime"`
	CreatedTime     types.Time    `json:"createdTime"`
	Status          string        `json:"status"`
	// Redeemable is false while a redeem order is pending or settlement is under 30 minutes away
	Redeemable  bool   `json:"redeemable"`
	AccountType string `json:"accountType"`
	OrderID     string `json:"orderId"`
}

// DiscountBuyPosition is an active Discount Buy position
type DiscountBuyPosition struct {
	PositionID       string        `json:"positionId"`
	ProductID        string        `json:"productId"`
	Category         string        `json:"category"`
	Coin             currency.Code `json:"coin"`
	UnderlyingAsset  currency.Code `json:"underlyingAsset"`
	Amount           types.Number  `json:"amount"`
	PurchasePrice    types.Number  `json:"purchasePrice"`
	KnockoutPrice    types.Number  `json:"knockoutPrice"`
	KnockoutCouponE8 types.Number  `json:"knockoutCouponE8"` // Annualised interest rate scaled by 1e8
	Status           string        `json:"status"`
	OrderID          string        `json:"orderId"`
	Duration         string        `json:"duration"`
	SettlementTime   types.Time    `json:"settlementTime"`
	AccountType      string        `json:"accountType"`
	ToAccountType    string        `json:"toAccountType"`
	SettleType       string        `json:"settleType"`
	ExpectReceiveAt  types.Time    `json:"expectReceiveAt"`
}

// AdvancedEarnOrderInfoRequest holds the parameters of Get Order Info for one Advanced Earn category
type AdvancedEarnOrderInfoRequest struct {
	Category    string
	ProductID   string
	OrderID     string
	OrderLinkID string
	// StartTime and EndTime bound the order creation time
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// AdvancedEarnOrderInfoResponse holds a page of the requested category's orders; only that category's slice is
// populated
type AdvancedEarnOrderInfoResponse struct {
	Category       string
	NextPageCursor string
	DualAssets     []DualAssetsOrder
	SmartLeverage  []SmartLeverageOrder
	DoubleWin      []DoubleWinOrder
	DiscountBuy    []DiscountBuyOrder
}

// UnmarshalJSON decodes each listed order into the type of the category it names
func (r *AdvancedEarnOrderInfoResponse) UnmarshalJSON(data []byte) error {
	var result advancedEarnList
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*r = AdvancedEarnOrderInfoResponse{Category: result.Category, NextPageCursor: result.NextPageCursor}
	return decodeAdvancedEarnElements(result.Category, result.List, map[string]func(json.RawMessage) error{
		AdvancedEarnCategoryDualAssets:    advancedEarnElementAppender(&r.DualAssets),
		AdvancedEarnCategorySmartLeverage: advancedEarnElementAppender(&r.SmartLeverage),
		AdvancedEarnCategoryDoubleWin:     advancedEarnElementAppender(&r.DoubleWin),
		AdvancedEarnCategoryDiscountBuy:   advancedEarnElementAppender(&r.DiscountBuy),
	})
}

// DualAssetsOrder is a Dual Assets order. The settlement fields are only set once the order is Settled
type DualAssetsOrder struct {
	OrderID          string        `json:"orderId"`
	OrderLinkID      string        `json:"orderLinkId"`
	ProductID        string        `json:"productId"`
	Category         string        `json:"category"`
	OrderType        string        `json:"orderType"`
	Amount           types.Number  `json:"amount"`
	Coin             currency.Code `json:"coin"`
	BaseCoin         currency.Code `json:"baseCoin"`
	QuoteCoin        currency.Code `json:"quoteCoin"`
	Status           string        `json:"status"`
	CreatedTime      types.Time    `json:"createdTime"`
	UpdatedTime      types.Time    `json:"updatedTime"`
	Direction        string        `json:"direction"`
	TargetPrice      types.Number  `json:"targetPrice"`
	SettlementTime   types.Time    `json:"settlementTime"`
	EstimateAPYE8    types.Number  `json:"estimateApyE8"` // Estimated APY scaled by 1e8
	Duration         string        `json:"duration"`
	AccountType      string        `json:"accountType"`
	ToAccountType    string        `json:"toAccountType"`
	SelectAPYE8      types.Number  `json:"selectApyE8"` // APY selected at order time, scaled by 1e8
	IsVIP            bool          `json:"isVip"`
	SettlementCoin   currency.Code `json:"settlementCoin"`
	SettlementAmount types.Number  `json:"settlementAmount"`
	OrderMode        string        `json:"orderMode"`
	SettlementPrice  types.Number  `json:"settlementPrice"`
	// RefundStatus is only set on failed orders
	RefundStatus     string       `json:"refundStatus"`
	TrialBonusAmount types.Number `json:"trialBonusAmount"`
	TrialBonusPNL    types.Number `json:"trialBonusPnl"`
}

// SmartLeverageOrder is a Smart Leverage stake or redeem order
type SmartLeverageOrder struct {
	OrderID         string        `json:"orderId"`
	OrderLinkID     string        `json:"orderLinkId"`
	ProductID       string        `json:"productId"`
	Category        string        `json:"category"`
	OrderType       string        `json:"orderType"`
	Amount          types.Number  `json:"amount"`
	CreatedTime     types.Time    `json:"createdTime"`
	Status          string        `json:"status"`
	Duration        string        `json:"duration"`
	AccountType     string        `json:"accountType"`
	SettlementTime  types.Time    `json:"settlementTime"`
	RefundStatus    string        `json:"refundStatus"`
	InvestCoin      currency.Code `json:"investCoin"`
	UnderlyingAsset currency.Code `json:"underlyingAsset"`
	Direction       string        `json:"direction"`
	Leverage        types.Number  `json:"leverage"`
	BreakevenPrice  types.Number  `json:"breakevenPrice"`
	InitialPrice    types.Number  `json:"initialPrice"`
	SettlementPrice types.Number  `json:"settlementPrice"`
	PNL             types.Number  `json:"pnl"`
	ToAccountType   string        `json:"toAccountType"`
}

// DoubleWinOrder is a Double Win stake or redeem order
type DoubleWinOrder struct {
	OrderID         string        `json:"orderId"`
	OrderLinkID     string        `json:"orderLinkId"`
	ProductID       string        `json:"productId"`
	Category        string        `json:"category"`
	OrderType       string        `json:"orderType"`
	InvestCoin      currency.Code `json:"investCoin"`
	Amount          types.Number  `json:"amount"`
	UnderlyingAsset currency.Code `json:"underlyingAsset"`
	Leverage        types.Number  `json:"leverage"`
	InitialPrice    types.Number  `json:"initialPrice"`
	LowerPrice      types.Number  `json:"lowerPrice"`
	UpperPrice      types.Number  `json:"upperPrice"`
	Duration        string        `json:"duration"`
	SettlementTime  types.Time    `json:"settlementTime"`
	CreatedTime     types.Time    `json:"createdTime"`
	UpdatedTime     types.Time    `json:"updatedTime"`
	Status          string        `json:"status"`
	SettlementPrice types.Number  `json:"settlementPrice"`
	PNL             types.Number  `json:"pnl"`
	RefundStatus    string        `json:"refundStatus"`
	AccountType     string        `json:"accountType"`
	ToAccountType   string        `json:"toAccountType"`
}

// DiscountBuyOrder is a Discount Buy order
type DiscountBuyOrder struct {
	OrderID          string        `json:"orderId"`
	OrderLinkID      string        `json:"orderLinkId"`
	ProductID        string        `json:"productId"`
	Category         string        `json:"category"`
	OrderType        string        `json:"orderType"`
	Amount           types.Number  `json:"amount"`
	Coin             currency.Code `json:"coin"`
	UnderlyingAsset  currency.Code `json:"underlyingAsset"`
	Status           string        `json:"status"`
	CreatedTime      types.Time    `json:"createdTime"`
	PurchasePrice    types.Number  `json:"purchasePrice"`
	KnockoutPrice    types.Number  `json:"knockoutPrice"`
	KnockoutCouponE8 types.Number  `json:"knockoutCouponE8"` // Annualised interest rate scaled by 1e8
	Duration         string        `json:"duration"`
	SettlementTime   types.Time    `json:"settlementTime"`
	AccountType      string        `json:"accountType"`
	ToAccountType    string        `json:"toAccountType"`
	SettleType       string        `json:"settleType"`
	// SettlementPrice is the 30 minute TWAP before the settlement time
	SettlementPrice  types.Number  `json:"settlementPrice"`
	SettlementCoin   currency.Code `json:"settlementCoin"`
	SettlementAmount types.Number  `json:"settlementAmount"`
	IsVIP            bool          `json:"isVip"`
	RefundStatus     string        `json:"refundStatus"`
}

// AdvancedEarnRedeemEstimateResponse holds the estimated early redemption of each requested position
type AdvancedEarnRedeemEstimateResponse struct {
	List []AdvancedEarnRedeemEstimate `json:"list"`
}

// AdvancedEarnRedeemEstimate is the estimated early redemption of a Smart Leverage or Double Win position
type AdvancedEarnRedeemEstimate struct {
	Success               bool         `json:"success"`
	PositionID            string       `json:"positionId"`
	EstimatedRedeemAmount types.Number `json:"estRedeemAmount"`
	EstimatedRedeemTime   types.Time   `json:"estRedeemTime"`
	SlippageRate          types.Number `json:"slippageRate"`
}

// DoubleWinCustomQuoteRequest holds the custom price range an RFQ Double Win product is quoted for. LowerPrice and
// UpperPrice must be multiples of the product's price tick size, below and above InitialPrice
type DoubleWinCustomQuoteRequest struct {
	ProductID    string
	InitialPrice float64
	LowerPrice   float64
	UpperPrice   float64
}

// DoubleWinCustomQuoteResponse is the quote for a custom Double Win price range; the order must be placed before
// ExpireTime
type DoubleWinCustomQuoteResponse struct {
	ProductID           string       `json:"productId"`
	InitialPrice        types.Number `json:"initialPrice"`
	LowerPrice          types.Number `json:"lowerPrice"`
	UpperPrice          types.Number `json:"upperPrice"`
	Leverage            types.Number `json:"leverage"`
	ExpireTime          types.Time   `json:"expireTime"`
	MaxInvestmentAmount types.Number `json:"maxInvestmentAmount"`
}

// LiquidityMiningProductInfoResponse holds the Liquidity Mining products
type LiquidityMiningProductInfoResponse struct {
	Products []LiquidityMiningProduct `json:"products"`
}

// LiquidityMiningProduct is a Liquidity Mining pool
type LiquidityMiningProduct struct {
	ProductID          string        `json:"productId"`
	BaseCoin           currency.Code `json:"baseCoin"`
	QuoteCoin          currency.Code `json:"quoteCoin"`
	Status             string        `json:"status"`
	MaxLeverage        uint64        `json:"maxLeverage"`
	MinInvestmentQuote types.Number  `json:"minInvestmentQuote"`
	MinInvestmentBase  types.Number  `json:"minInvestmentBase"`
	MaxInvestmentQuote types.Number  `json:"maxInvestmentQuote"`
	MaxInvestmentBase  types.Number  `json:"maxInvestmentBase"`
	// MinWithdrawalAmount and MinReinvestAmount are in the quote coin
	MinWithdrawalAmount types.Number    `json:"minWithdrawalAmount"`
	MinReinvestAmount   types.Number    `json:"minReinvestAmount"`
	BaseCoinPrecision   uint64          `json:"baseCoinPrecision"`
	QuoteCoinPrecision  uint64          `json:"quoteCoinPrecision"`
	YieldCoins          []currency.Code `json:"yieldCoins"`
	// APYE8 and APY7DayE8 are APY scaled by 1e8; the docs type them as integers but Bybit sends strings
	APYE8              types.Number   `json:"apyE8"`
	APY7DayE8          types.Number   `json:"apy7dE8"`
	PoolLiquidityValue types.Number   `json:"poolLiquidityValue"` // In the quote coin
	DailyYield         types.Number   `json:"dailyYield"`
	SlippageLevels     []types.Number `json:"slippageLevels"`     // Investment amount at which each slippage tier starts
	SlippageRateE8List []types.Number `json:"slippageRateE8List"` // Slippage rate of each tier, scaled by 1e8
	// APYBreakdown and APY7DayBreakdown split the APY and the 7 day APY by yield coin
	APYBreakdown     []LiquidityMiningAPYBreakdown `json:"apyBreakdown"`
	APY7DayBreakdown []LiquidityMiningAPYBreakdown `json:"apy7dBreakdown"`
}

// LiquidityMiningAPYBreakdown is the APY one yield coin contributes to a Liquidity Mining pool
type LiquidityMiningAPYBreakdown struct {
	Coin currency.Code `json:"coin"`
	APY  types.Number  `json:"apy"`
}

// AddLiquidityRequest holds the parameters of Add Liquidity; at least one of QuoteAmount and BaseAmount is required,
// each with the account type it is taken from
type AddLiquidityRequest struct {
	ProductID        string
	OrderLinkID      string
	QuoteAmount      float64
	BaseAmount       float64
	QuoteAccountType string
	BaseAccountType  string
	// Leverage defaults to 1, no leverage
	Leverage float64
}

// RemoveLiquidityRequest holds the parameters of Remove Liquidity
type RemoveLiquidityRequest struct {
	ProductID   string
	OrderLinkID string
	PositionID  string
	// RemoveRate is the percentage withdrawn, 1 to 100; Bybit withdraws everything when it is zero
	RemoveRate uint64
	// RemoveType is Normal (both coins, the default), SingleQuoteCoin or SingleBaseCoin
	RemoveType string
}

// ReinvestLiquidityMiningRequest holds the parameters of Reinvest, which returns a position's claimable yield to the
// pool
type ReinvestLiquidityMiningRequest struct {
	ProductID   string
	OrderLinkID string
	PositionID  string
	// Leverage is a whole multiplier, defaulting to 1
	Leverage uint64
}

// AddLiquidityMiningMarginRequest holds the parameters of Add Margin, which adds quote coin margin to a leveraged
// position
type AddLiquidityMiningMarginRequest struct {
	ProductID        string
	OrderLinkID      string
	PositionID       string
	Amount           float64
	QuoteAccountType string
}

// LiquidityMiningPositionInfoResponse holds the active Liquidity Mining positions
type LiquidityMiningPositionInfoResponse struct {
	Positions []LiquidityMiningPosition `json:"positions"`
}

// LiquidityMiningPosition is an active Liquidity Mining position; its amounts and values are recalculated at the
// current market price, in the quote coin unless named otherwise
type LiquidityMiningPosition struct {
	PositionID              string        `json:"positionId"`
	ProductID               string        `json:"productId"`
	BaseCoin                currency.Code `json:"baseCoin"`
	QuoteCoin               currency.Code `json:"quoteCoin"`
	QuoteAmount             types.Number  `json:"quoteAmount"`
	BaseAmount              types.Number  `json:"baseAmount"`
	PrincipalQuoteAmount    types.Number  `json:"principalQuoteAmount"`
	PrincipalBaseAmount     types.Number  `json:"principalBaseAmount"`
	PrincipalLiquidityValue types.Number  `json:"principalLiquidityValue"`
	LeveragedValue          types.Number  `json:"leveragedValue"`
	Loan                    types.Number  `json:"loan"`
	ClaimableYield          types.Number  `json:"claimableYield"`
	CurrentAPR              types.Number  `json:"currentApr"` // APR scaled by 1e8
	Leverage                types.Number  `json:"leverage"`
	Margin                  types.Number  `json:"margin"`
	LiquidationPrice        types.Number  `json:"liquidationPrice"`
	CurrentPriceY           types.Number  `json:"currentPriceY"` // Base coin price
	Status                  string        `json:"status"`
	CreatedTime             types.Time    `json:"createdTime"`
}

// LiquidityMiningOrderInfoRequest holds the parameters of Get Order Info for Liquidity Mining. OrderID or
// OrderLinkID alone fetches one order; without either the other fields filter the order list
type LiquidityMiningOrderInfoRequest struct {
	OrderID     string
	OrderLinkID string
	ProductID   string
	// OrderType is AddLiquidity, RemoveLiquidity, Reinvest or AddMargin
	OrderType string
	// StartTime and EndTime bound the order creation time
	StartTime time.Time
	EndTime   time.Time
	// Status is Success or Processing; Bybit returns every terminal status when it is empty
	Status string
	Limit  uint64
	Cursor string
}

// LiquidityMiningOrderInfoResponse holds a page of Liquidity Mining orders
type LiquidityMiningOrderInfoResponse struct {
	Orders         []LiquidityMiningOrder `json:"orders"`
	NextPageCursor string                 `json:"nextPageCursor"`
}

// LiquidityMiningOrder is a Liquidity Mining order. RemoveType and RemoveRate apply to RemoveLiquidity orders;
// Leverage and SlippageValue to AddLiquidity orders
type LiquidityMiningOrder struct {
	OrderID       string        `json:"orderId"`
	OrderLinkID   string        `json:"orderLinkId"`
	ProductID     string        `json:"productId"`
	OrderType     string        `json:"orderType"`
	BaseCoin      currency.Code `json:"baseCoin"`
	QuoteCoin     currency.Code `json:"quoteCoin"`
	QuoteAmount   types.Number  `json:"quoteAmount"`
	BaseAmount    types.Number  `json:"baseAmount"`
	Status        string        `json:"status"`
	RemoveType    string        `json:"removeType"`
	RemoveRate    uint64        `json:"removeRate"`
	Leverage      types.Number  `json:"leverage"`
	SlippageValue types.Number  `json:"slippageValue"`
	CreatedTime   types.Time    `json:"createdTime"`
}

// LiquidityMiningRecordsRequest holds the parameters of Get Yield Records and Get Liquidation Records
type LiquidityMiningRecordsRequest struct {
	BaseCoin  currency.Code
	QuoteCoin currency.Code
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// LiquidityMiningYieldRecordsResponse holds a page of Liquidity Mining yield claims
type LiquidityMiningYieldRecordsResponse struct {
	Records        []LiquidityMiningYieldRecord `json:"records"`
	NextPageCursor string                       `json:"nextPageCursor"`
}

// LiquidityMiningYieldRecord is a yield claim, made manually or settled on removing liquidity
type LiquidityMiningYieldRecord struct {
	Coin        currency.Code `json:"coin"`
	Amount      types.Number  `json:"amount"`
	BaseCoin    currency.Code `json:"baseCoin"`
	QuoteCoin   currency.Code `json:"quoteCoin"`
	Type        string        `json:"type"`
	Status      string        `json:"status"`
	CreatedTime types.Time    `json:"createdTime"`
}

// LiquidityMiningLiquidationRecordsResponse holds a page of Liquidity Mining liquidations
type LiquidityMiningLiquidationRecordsResponse struct {
	Records        []LiquidityMiningLiquidationRecord `json:"records"`
	NextPageCursor string                             `json:"nextPageCursor"`
}

// LiquidityMiningLiquidationRecord is the liquidation of a leveraged Liquidity Mining position; the amounts are what
// was returned after liquidation
type LiquidityMiningLiquidationRecord struct {
	BaseCoin         currency.Code `json:"baseCoin"`
	QuoteCoin        currency.Code `json:"quoteCoin"`
	BaseAmount       types.Number  `json:"baseAmount"`
	QuoteAmount      types.Number  `json:"quoteAmount"`
	LiquidationPrice types.Number  `json:"liquidationPrice"`
	LiquidationTime  types.Time    `json:"liquidationTime"`
}
