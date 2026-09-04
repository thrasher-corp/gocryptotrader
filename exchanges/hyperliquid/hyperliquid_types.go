package hyperliquid

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Exchange implements exchange.IBotExchange and provides Hyperliquid API access.
type Exchange struct {
	exchange.Base
	pairMappingsMu         sync.RWMutex
	pairMappingsFetchMu    sync.Mutex
	pairMappings           map[asset.Item][]pairMapping
	pairMappingMisses      map[string]time.Time
	authorityValidationMu  sync.Mutex
	authorityValidationKey authorityValidationKey
	authorityValidated     bool
	websocketPendingMu     sync.Mutex
	websocketPending       map[websocketPendingKey]*websocketPendingOperation
	lastNonce              atomic.Uint64
}

type authorityValidationKey struct {
	accountAddress string
	vaultAddress   string
	signerAddress  string
	mainnet        bool
}

// AccountAbstraction identifies how Hyperliquid shares balances across spot
// and perpetual DEXes.
type AccountAbstraction string

// Hyperliquid account abstraction modes returned by the info API.
const (
	AccountAbstractionDefault   AccountAbstraction = "default"
	AccountAbstractionDisabled  AccountAbstraction = "disabled"
	AccountAbstractionDEX       AccountAbstraction = "dexAbstraction"
	AccountAbstractionUnified   AccountAbstraction = "unifiedAccount"
	AccountAbstractionPortfolio AccountAbstraction = "portfolioMargin"
)

// PerpetualMetadata contains metadata for Hyperliquid perpetual markets.
type PerpetualMetadata struct {
	Universe        []PerpetualAssetMetadata `json:"universe"`
	CollateralToken uint64                   `json:"collateralToken"`
	MarginTables    []json.RawMessage        `json:"marginTables"`
}

// PerpetualDEX describes one builder-deployed perpetual DEX.
type PerpetualDEX struct {
	Name         string `json:"name"`
	FullName     string `json:"fullName"`
	Deployer     string `json:"deployer"`
	FeeRecipient string `json:"feeRecipient"`
}

// PerpetualAssetMetadata contains metadata for one perpetual market.
type PerpetualAssetMetadata struct {
	Name          string `json:"name"`
	SizeDecimals  uint64 `json:"szDecimals"`
	MaxLeverage   uint64 `json:"maxLeverage"`
	MarginTableID uint64 `json:"marginTableId"`
	OnlyIsolated  bool   `json:"onlyIsolated"`
	IsDelisted    bool   `json:"isDelisted"`
}

// SpotMetadata contains metadata for Hyperliquid spot markets and tokens.
type SpotMetadata struct {
	Universe []SpotAssetMetadata `json:"universe"`
	Tokens   []SpotTokenMetadata `json:"tokens"`
}

// SpotAssetMetadata contains metadata for one spot market.
type SpotAssetMetadata struct {
	Tokens      []uint64 `json:"tokens"`
	Name        string   `json:"name"`
	Index       uint64   `json:"index"`
	IsCanonical bool     `json:"isCanonical"`
}

// SpotTokenMetadata contains metadata for one spot token.
type SpotTokenMetadata struct {
	Name                    string          `json:"name"`
	SizeDecimals            uint64          `json:"szDecimals"`
	WeiDecimals             uint64          `json:"weiDecimals"`
	Index                   uint64          `json:"index"`
	TokenID                 string          `json:"tokenId"`
	IsCanonical             bool            `json:"isCanonical"`
	EVMContract             json.RawMessage `json:"evmContract"`
	FullName                *string         `json:"fullName"`
	DeployerTradingFeeShare types.Number    `json:"deployerTradingFeeShare"`
}

// PerpetualAssetContext contains current market data for one perpetual market.
type PerpetualAssetContext struct {
	Funding           types.Number   `json:"funding"`
	OpenInterest      types.Number   `json:"openInterest"`
	PreviousDayPrice  types.Number   `json:"prevDayPx"`
	DayNotionalVolume types.Number   `json:"dayNtlVlm"`
	Premium           types.Number   `json:"premium"`
	OraclePrice       types.Number   `json:"oraclePx"`
	MarkPrice         types.Number   `json:"markPx"`
	MidPrice          types.Number   `json:"midPx"`
	ImpactPrices      []types.Number `json:"impactPxs"`
	DayBaseVolume     types.Number   `json:"dayBaseVlm"`
}

// SpotAssetContext contains current market data for one spot market.
type SpotAssetContext struct {
	PreviousDayPrice  types.Number `json:"prevDayPx"`
	DayNotionalVolume types.Number `json:"dayNtlVlm"`
	MarkPrice         types.Number `json:"markPx"`
	MidPrice          types.Number `json:"midPx"`
	CirculatingSupply types.Number `json:"circulatingSupply"`
	TotalSupply       types.Number `json:"totalSupply"`
	Coin              string       `json:"coin"`
	DayBaseVolume     types.Number `json:"dayBaseVlm"`
}

// PerpetualMetadataAndAssetContexts contains perpetual metadata and aligned market contexts.
type PerpetualMetadataAndAssetContexts struct {
	Metadata      PerpetualMetadata
	AssetContexts []PerpetualAssetContext
}

// FundingRateRecord contains one hourly perpetual funding rate.
type FundingRateRecord struct {
	Coin        string       `json:"coin"`
	FundingRate types.Number `json:"fundingRate"`
	Premium     types.Number `json:"premium"`
	Time        types.Time   `json:"time"`
}

// UserLedgerUpdate contains one non-funding account ledger update.
type UserLedgerUpdate struct {
	Delta UserLedgerDelta `json:"delta"`
	Hash  string          `json:"hash"`
	Time  types.Time      `json:"time"`
}

// UserLedgerDelta contains common fields across Hyperliquid's account ledger
// update variants. Type identifies which fields are populated.
type UserLedgerDelta struct {
	Type            string       `json:"type"`
	USDC            types.Number `json:"usdc"`
	Fee             types.Number `json:"fee"`
	Amount          types.Number `json:"amount"`
	USDCValue       types.Number `json:"usdcValue"`
	NativeTokenFee  types.Number `json:"nativeTokenFee"`
	RequestedUSD    types.Number `json:"requestedUsd"`
	NetWithdrawnUSD types.Number `json:"netWithdrawnUsd"`
	Nonce           uint64       `json:"nonce"`
	User            string       `json:"user"`
	Destination     string       `json:"destination"`
	SourceDEX       string       `json:"sourceDex"`
	DestinationDEX  string       `json:"destinationDex"`
	Token           string       `json:"token"`
	FeeToken        string       `json:"feeToken"`
	Vault           string       `json:"vault"`
	ToPerp          bool         `json:"toPerp"`
}

// UserFees contains effective account-specific trade fee rates.
type UserFees struct {
	UserCrossRate     types.Number `json:"userCrossRate"`
	UserAddRate       types.Number `json:"userAddRate"`
	UserSpotCrossRate types.Number `json:"userSpotCrossRate"`
	UserSpotAddRate   types.Number `json:"userSpotAddRate"`
}

// ActiveAssetData contains account-specific settings for one perpetual market.
type ActiveAssetData struct {
	User     string          `json:"user"`
	Coin     string          `json:"coin"`
	Leverage AccountLeverage `json:"leverage"`
}

// AccountLeverage contains one cross or isolated leverage setting.
type AccountLeverage struct {
	Type   string       `json:"type"`
	Value  types.Number `json:"value"`
	RawUSD types.Number `json:"rawUsd"`
}

// SpotMetadataAndAssetContexts contains spot metadata and current market contexts.
type SpotMetadataAndAssetContexts struct {
	Metadata      SpotMetadata
	AssetContexts []SpotAssetContext
}

// L2BookRequest contains optional aggregation controls for an L2 book request.
type L2BookRequest struct {
	Coin               string
	SignificantFigures *uint64
	Mantissa           *uint64
}

// L2Book contains a complete Hyperliquid L2 book snapshot.
type L2Book struct {
	Coin   string      `json:"coin"`
	Levels [][]L2Level `json:"levels"`
	Time   types.Time  `json:"time"`
}

// L2Level contains one aggregated orderbook price level.
type L2Level struct {
	Price      types.Number `json:"px"`
	Size       types.Number `json:"sz"`
	OrderCount uint64       `json:"n"`
}

// RecentTrade contains one public trade reported by Hyperliquid.
type RecentTrade struct {
	Coin    string       `json:"coin"`
	Side    string       `json:"side"`
	Price   types.Number `json:"px"`
	Size    types.Number `json:"sz"`
	Time    types.Time   `json:"time"`
	Hash    string       `json:"hash"`
	TradeID uint64       `json:"tid"`
	Users   []string     `json:"users"`
}

// CandleRequest contains parameters for a candle snapshot request.
type CandleRequest struct {
	Coin      string
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
}

// Candle contains one Hyperliquid OHLCV interval.
type Candle struct {
	OpenTime   types.Time   `json:"t"`
	CloseTime  types.Time   `json:"T"`
	Symbol     string       `json:"s"`
	Interval   string       `json:"i"`
	Open       types.Number `json:"o"`
	Close      types.Number `json:"c"`
	High       types.Number `json:"h"`
	Low        types.Number `json:"l"`
	Volume     types.Number `json:"v"`
	TradeCount uint64       `json:"n"`
}

// UserRole contains the account role for an on-chain address.
type UserRole struct {
	Role string       `json:"role"`
	Data UserRoleData `json:"data"`
}

// UserRoleData contains a role's related master or user address.
type UserRoleData struct {
	User   string `json:"user"`
	Master string `json:"master"`
}

// VaultDetails contains the ownership fields needed to validate vault trading authority.
type VaultDetails struct {
	VaultAddress string `json:"vaultAddress"`
	Leader       string `json:"leader"`
}

// SpotClearinghouseState contains spot balances for one account.
type SpotClearinghouseState struct {
	Balances []SpotBalance `json:"balances"`
}

// SpotBalance contains one spot token balance.
type SpotBalance struct {
	Coin       string       `json:"coin"`
	TokenIndex uint64       `json:"token"`
	Total      types.Number `json:"total"`
	Hold       types.Number `json:"hold"`
	EntryValue types.Number `json:"entryNtl"`
}

// ClearinghouseState contains perpetual account balances and positions.
type ClearinghouseState struct {
	MarginSummary      MarginSummary   `json:"marginSummary"`
	CrossMarginSummary MarginSummary   `json:"crossMarginSummary"`
	Withdrawable       types.Number    `json:"withdrawable"`
	AssetPositions     []AssetPosition `json:"assetPositions"`
}

// MarginSummary contains aggregate perpetual margin values.
type MarginSummary struct {
	AccountValue    types.Number `json:"accountValue"`
	TotalMarginUsed types.Number `json:"totalMarginUsed"`
	TotalNotional   types.Number `json:"totalNtlPos"`
	TotalRawUSD     types.Number `json:"totalRawUsd"`
}

// AssetPosition contains one perpetual position.
type AssetPosition struct {
	Type     string   `json:"type"`
	Position Position `json:"position"`
}

// Position contains one perpetual market position.
type Position struct {
	Coin             string       `json:"coin"`
	EntryPrice       types.Number `json:"entryPx"`
	LiquidationPrice types.Number `json:"liquidationPx"`
	MarginUsed       types.Number `json:"marginUsed"`
	PositionValue    types.Number `json:"positionValue"`
	ReturnOnEquity   types.Number `json:"returnOnEquity"`
	Size             types.Number `json:"szi"`
	UnrealisedProfit types.Number `json:"unrealizedPnl"`
}

// OpenOrder contains the common order fields returned by account info and websocket endpoints.
type OpenOrder struct {
	Coin             string       `json:"coin"`
	Side             string       `json:"side"`
	LimitPrice       types.Number `json:"limitPx"`
	Size             types.Number `json:"sz"`
	OriginalSize     types.Number `json:"origSz"`
	OrderID          uint64       `json:"oid"`
	Timestamp        types.Time   `json:"timestamp"`
	TriggerCondition string       `json:"triggerCondition"`
	IsTrigger        bool         `json:"isTrigger"`
	TriggerPrice     types.Number `json:"triggerPx"`
	IsPositionTPSL   bool         `json:"isPositionTpsl"`
	ReduceOnly       bool         `json:"reduceOnly"`
	OrderType        string       `json:"orderType"`
	TimeInForce      string       `json:"tif"`
	ClientOrderID    *string      `json:"cloid"`
}

// HistoricalOrder contains an order and its latest status.
type HistoricalOrder struct {
	Order           OpenOrder  `json:"order"`
	Status          string     `json:"status"`
	StatusTimestamp types.Time `json:"statusTimestamp"`
}

// OrderStatusResponse contains either an order result or an unknown-order status.
type OrderStatusResponse struct {
	Status string           `json:"status"`
	Order  *HistoricalOrder `json:"order"`
}

type exchangeActionResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

type exchangeActionData struct {
	Type string `json:"type"`
	Data struct {
		Statuses []json.RawMessage `json:"statuses"`
	} `json:"data"`
}

type restingOrderStatus struct {
	OrderID uint64 `json:"oid"`
}

type filledOrderStatus struct {
	TotalSize    types.Number `json:"totalSz"`
	AveragePrice types.Number `json:"avgPx"`
	OrderID      uint64       `json:"oid"`
}

type orderActionStatus struct {
	Resting  *restingOrderStatus `json:"resting"`
	Filled   *filledOrderStatus  `json:"filled"`
	Error    string              `json:"error"`
	Deferred string              `json:"-"`
}
