package binanceus

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ServerTimeResponse holds the server time
type ServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// SystemStatusResponse holds the system status: 0 when normal and 1 during system maintenance
type SystemStatusResponse struct {
	Status uint64 `json:"status"`
}

// ExchangeInfoRequest holds the optional filters of an exchange information request. Symbols cannot be combined with
// Permissions
type ExchangeInfoRequest struct {
	Symbols currency.Pairs
	// Permissions defaults to SPOT
	Permissions []Permission
	// ShowPermissionSets false leaves each symbol's permission sets empty; the endpoint defaults it to true
	ShowPermissionSets *bool
}

// ExchangeInfoResponse holds the exchange trading rules and symbol information
type ExchangeInfoResponse struct {
	Timezone        string                  `json:"timezone"`
	ServerTime      types.Time              `json:"serverTime"`
	RateLimits      []RateLimitItemResponse `json:"rateLimits"`
	ExchangeFilters []ExchangeFilter        `json:"exchangeFilters"`
	Symbols         []SymbolInfo            `json:"symbols"`
}

// SymbolInfo holds a symbol's trading rules and filters. Binance.US's matching engine reports OTO, OPO, amend and peg
// capabilities that its API does not document endpoints or parameters for
type SymbolInfo struct {
	Symbol string `json:"symbol"`
	// Status is PRE_TRADING, TRADING, POST_TRADING, END_OF_DAY, HALT, AUCTION_MATCH or BREAK
	Status             string        `json:"status"`
	BaseAsset          currency.Code `json:"baseAsset"`
	BaseAssetPrecision uint64        `json:"baseAssetPrecision"`
	QuoteAsset         currency.Code `json:"quoteAsset"`
	// QuotePrecision is to be removed in a future API version in favour of QuoteAssetPrecision
	QuotePrecision                  uint64         `json:"quotePrecision"`
	QuoteAssetPrecision             uint64         `json:"quoteAssetPrecision"`
	BaseCommissionPrecision         uint64         `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision        uint64         `json:"quoteCommissionPrecision"`
	OrderTypes                      []OrderType    `json:"orderTypes"`
	IcebergAllowed                  bool           `json:"icebergAllowed"`
	OneCancelsTheOtherAllowed       bool           `json:"ocoAllowed"`
	OneTriggersTheOtherAllowed      bool           `json:"otoAllowed"`
	OPOAllowed                      bool           `json:"opoAllowed"`
	QuoteOrderQuantityMarketAllowed bool           `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop               bool           `json:"allowTrailingStop"`
	CancelReplaceAllowed            bool           `json:"cancelReplaceAllowed"`
	AmendAllowed                    bool           `json:"amendAllowed"`
	PegInstructionsAllowed          bool           `json:"pegInstructionsAllowed"`
	IsSpotTradingAllowed            bool           `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed          bool           `json:"isMarginTradingAllowed"`
	Filters                         []SymbolFilter `json:"filters"`
	// Permissions is empty and to be removed; PermissionSets holds a symbol's permissions instead
	Permissions []Permission `json:"permissions"`
	// PermissionSets lists the permissions an account needs to trade the symbol: one permission of each inner list
	PermissionSets                  [][]Permission            `json:"permissionSets"`
	DefaultSelfTradePreventionMode  SelfTradePreventionMode   `json:"defaultSelfTradePreventionMode"`
	AllowedSelfTradePreventionModes []SelfTradePreventionMode `json:"allowedSelfTradePreventionModes"`
}
