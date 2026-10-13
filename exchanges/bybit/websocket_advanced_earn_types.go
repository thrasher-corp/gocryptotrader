package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AdvancedEarnSubscriptionRequest subscribes to or unsubscribes from Advanced Earn offer topics
type AdvancedEarnSubscriptionRequest struct {
	RequestID string   `json:"req_id"`
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

// AdvancedEarnWebsocketMessage is a frame of the Advanced Earn offers stream: an offers push, or the reply to a
// subscribe, unsubscribe or ping request
type AdvancedEarnWebsocketMessage struct {
	Topic         string          `json:"topic"`
	Data          json.RawMessage `json:"data"`
	Operation     string          `json:"op"`
	RequestID     string          `json:"req_id"`
	Success       bool            `json:"success"`
	ReturnMessage string          `json:"ret_msg"`
	ConnectionID  string          `json:"conn_id"`
	// Arguments holds the server time on a pong
	Arguments []types.Time `json:"args"`
}

// DualAssetsOffer holds the prices on offer for a Dual Assets product, pushed on earn.dualassets.offers
type DualAssetsOffer struct {
	// ProductID is documented as an integer but sent as a string
	ProductID     string                 `json:"p"`
	CurrentPrice  types.Number           `json:"c"`
	BuyLowPrice   []DualAssetsOfferPrice `json:"b"`
	SellHighPrice []DualAssetsOfferPrice `json:"s"`
}

// DualAssetsOfferPrice is a target price on offer for a Dual Assets direction
type DualAssetsOfferPrice struct {
	SelectPrice types.Number `json:"s"`
	// APYE8 is the APY scaled by 1e8, documented as an integer but sent as a string
	APYE8               types.Number `json:"a"`
	MaxInvestmentAmount types.Number `json:"m"`
	ExpiredAt           types.Time   `json:"x"`
}

// SmartLeverageOffer is the quote for a Smart Leverage product, pushed on earn.smartleverage.offers when it changes
// and every 30 seconds
type SmartLeverageOffer struct {
	ProductID           string       `json:"p"`
	BreakevenPrice      types.Number `json:"b"`
	ExpireAt            types.Time   `json:"e"`
	MaxInvestmentAmount types.Number `json:"m"`
	CurrentPrice        types.Number `json:"c"` // Underlying asset price
}

// DiscountBuyOffer is a quote offered for a Discount Buy product, pushed on earn.discountbuy.offers
type DiscountBuyOffer struct {
	ProductID        string       `json:"p"`
	CurrentPrice     types.Number `json:"c"` // Underlying index price
	PurchasePrice    types.Number `json:"b"` // Anchor buy price
	KnockoutPrice    types.Number `json:"k"`
	KnockoutCouponE8 types.Number `json:"e"` // Annualised interest rate scaled by 1e8
	// MaxInvestmentAmount is the most this quote accepts
	MaxInvestmentAmount types.Number `json:"m"`
	// InstitutionUID identifies the quoting market maker, and must be sent unchanged with the order
	InstitutionUID string `json:"i"`
	// ExpiredAt is zero when the institution set no expiry
	ExpiredAt types.Time `json:"x"`
}

// DoubleWinOffer is the quote for a fixed price range Double Win product, pushed on earn.doublewin.offers
type DoubleWinOffer struct {
	ProductID           string       `json:"p"`
	CurrentPrice        types.Number `json:"c"` // Underlying index price
	Leverage            types.Number `json:"l"` // Current maximum leverage
	MaxInvestmentAmount types.Number `json:"m"`
	ExpireTime          types.Time   `json:"e"`
}
