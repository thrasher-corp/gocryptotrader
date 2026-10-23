package hyperliquid

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PerpetualDEXResponse describes one builder-deployed perpetual DEX
type PerpetualDEXResponse struct {
	Name                            string        `json:"name"`
	FullName                        string        `json:"fullName"`
	Deployer                        string        `json:"deployer"`
	OracleUpdater                   string        `json:"oracleUpdater"`
	FeeRecipient                    string        `json:"feeRecipient"`
	AssetToStreamingOpenInterestCap []AssetValue  `json:"assetToStreamingOiCap"`
	SubDeployers                    []SubDeployer `json:"subDeployers"`
	AssetToFundingMultiplier        []AssetValue  `json:"assetToFundingMultiplier"`
	AssetToFundingInterestRate      []AssetValue  `json:"assetToFundingInterestRate"`
	AssetToFundingClamp             []AssetValue  `json:"assetToFundingClamp"`
}

// AssetValue contains a numeric setting for one coin
type AssetValue struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin  string
	Value types.Number
}

// UnmarshalJSON decodes a [coin, value] tuple
func (a *AssetValue) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &a.Coin, &a.Value)
}

// SubDeployer contains the addresses a DEX deployer has delegated one deployer action to
type SubDeployer struct {
	Action    string
	Addresses []string
}

// UnmarshalJSON decodes an [action, addresses] tuple
func (s *SubDeployer) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &s.Action, &s.Addresses)
}

// PerpetualMetadataResponse contains the markets, margin tables and collateral token of a perpetual DEX
type PerpetualMetadataResponse struct {
	Universe        []PerpetualAssetMetadata `json:"universe"`
	MarginTables    []MarginTableEntry       `json:"marginTables"`
	CollateralToken uint64                   `json:"collateralToken"`
}

// PerpetualAssetMetadata contains the metadata of one perpetual market
type PerpetualAssetMetadata struct {
	Name          string `json:"name"`
	SizeDecimals  uint64 `json:"szDecimals"`
	MaxLeverage   uint64 `json:"maxLeverage"`
	MarginTableID uint64 `json:"marginTableId"`
	IsDelisted    bool   `json:"isDelisted"`
	// MarginMode is strictIsolated when margin cannot be removed, or noCross when only isolated margin is allowed
	MarginMode               string       `json:"marginMode"`
	GrowthMode               string       `json:"growthMode"`
	LastGrowthModeChangeTime ZonelessTime `json:"lastGrowthModeChangeTime"`
	DeployerFeeScale         types.Number `json:"deployerFeeScale"`
	LastFeeScaleChangeTime   ZonelessTime `json:"lastFeeScaleChangeTime"`
}

// MarginTableEntry contains a margin table and the ID markets reference it by
type MarginTableEntry struct {
	ID    uint64
	Table MarginTable
}

// UnmarshalJSON decodes an [id, table] tuple
func (m *MarginTableEntry) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &m.ID, &m.Table)
}

// MarginTable contains the leverage tiers of a margin table
type MarginTable struct {
	Description string       `json:"description"`
	MarginTiers []MarginTier `json:"marginTiers"`
}

// MarginTier contains the maximum leverage from a position notional lower bound
type MarginTier struct {
	LowerBound  types.Number `json:"lowerBound"`
	MaxLeverage uint64       `json:"maxLeverage"`
}

// PerpetualAssetContext contains current market data for one perpetual market
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

// PerpetualMetadataAndAssetContextsResponse contains a perpetual DEX's metadata and a context for each market in its
// universe, in the same order
type PerpetualMetadataAndAssetContextsResponse struct {
	Metadata      PerpetualMetadataResponse
	AssetContexts []PerpetualAssetContext
}

// UnmarshalJSON decodes a [metadata, contexts] tuple
func (p *PerpetualMetadataAndAssetContextsResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Metadata, &p.AssetContexts)
}

// ClearinghouseStateResponse contains an account's perpetual margin summary and positions on one DEX
type ClearinghouseStateResponse struct {
	MarginSummary              MarginSummary   `json:"marginSummary"`
	CrossMarginSummary         MarginSummary   `json:"crossMarginSummary"`
	CrossMaintenanceMarginUsed types.Number    `json:"crossMaintenanceMarginUsed"`
	Withdrawable               types.Number    `json:"withdrawable"`
	AssetPositions             []AssetPosition `json:"assetPositions"`
	Time                       types.Time      `json:"time"`
}

// MarginSummary contains aggregate perpetual margin values
type MarginSummary struct {
	AccountValue          types.Number `json:"accountValue"`
	TotalNotionalPosition types.Number `json:"totalNtlPos"`
	TotalRawUSD           types.Number `json:"totalRawUsd"`
	TotalMarginUsed       types.Number `json:"totalMarginUsed"`
}

// AssetPosition contains one perpetual position and its position type, which is oneWay
type AssetPosition struct {
	Type     string   `json:"type"`
	Position Position `json:"position"`
}

// Position contains one perpetual market position
type Position struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin string `json:"coin"`
	// SignedSize is positive for a long position and negative for a short position
	SignedSize        types.Number      `json:"szi"`
	Leverage          Leverage          `json:"leverage"`
	EntryPrice        types.Number      `json:"entryPx"`
	PositionValue     types.Number      `json:"positionValue"`
	UnrealisedPNL     types.Number      `json:"unrealizedPnl"`
	ReturnOnEquity    types.Number      `json:"returnOnEquity"`
	LiquidationPrice  types.Number      `json:"liquidationPx"`
	MarginUsed        types.Number      `json:"marginUsed"`
	MaxLeverage       uint64            `json:"maxLeverage"`
	CumulativeFunding CumulativeFunding `json:"cumFunding"`
}

// Leverage contains a cross or isolated leverage setting; RawUSD is only returned for isolated leverage
type Leverage struct {
	Type   string       `json:"type"`
	Value  uint64       `json:"value"`
	RawUSD types.Number `json:"rawUsd"`
}

// CumulativeFunding contains the funding a position has paid or received
type CumulativeFunding struct {
	AllTime     types.Number `json:"allTime"`
	SinceOpen   types.Number `json:"sinceOpen"`
	SinceChange types.Number `json:"sinceChange"`
}

// UserFundingRequest contains the account and inclusive time range of a user funding request; a zero EndTime ends the
// range now
type UserFundingRequest struct {
	User      string
	StartTime time.Time
	EndTime   time.Time
}

// UserFundingUpdateResponse contains one funding payment, or one day of a market's payments once Hyperliquid has compacted older
// history into daily aggregates
type UserFundingUpdateResponse struct {
	Time  types.Time       `json:"time"`
	Hash  string           `json:"hash"`
	Delta UserFundingDelta `json:"delta"`
}

// UserFundingDelta contains a funding payment; USDC is negative when the account paid funding
type UserFundingDelta struct {
	Type string `json:"type"`
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin string       `json:"coin"`
	USDC types.Number `json:"usdc"`
	// SignedSize is positive for a long position and negative for a short position, and is the day's average size in a
	// daily aggregate
	SignedSize  types.Number `json:"szi"`
	FundingRate types.Number `json:"fundingRate"`
	// NumberOfSamples is the number of hourly payments in a daily aggregate, and zero for a single hourly payment
	NumberOfSamples uint64 `json:"nSamples"`
}

// UserNonFundingLedgerUpdatesRequest contains the account and inclusive time range of a non-funding ledger updates
// request; a zero EndTime ends the range now
type UserNonFundingLedgerUpdatesRequest struct {
	User      string
	StartTime time.Time
	EndTime   time.Time
}

// UserLedgerUpdateResponse contains one non-funding ledger update, such as a deposit, transfer, withdrawal or liquidation
type UserLedgerUpdateResponse struct {
	Delta UserLedgerDelta `json:"delta"`
	Hash  string          `json:"hash"`
	Time  types.Time      `json:"time"`
}

// UserLedgerDelta contains the fields of every ledger update variant; Type identifies which fields are populated
type UserLedgerDelta struct {
	Type                       string               `json:"type"`
	USDC                       types.Number         `json:"usdc"`
	ToPerpetual                bool                 `json:"toPerp"`
	User                       string               `json:"user"`
	Destination                string               `json:"destination"`
	SourceDEX                  string               `json:"sourceDex"`
	DestinationDEX             string               `json:"destinationDex"`
	Token                      currency.Code        `json:"token"`
	Amount                     types.Number         `json:"amount"`
	USDCValue                  types.Number         `json:"usdcValue"`
	Fee                        types.Number         `json:"fee"`
	NativeTokenFee             types.Number         `json:"nativeTokenFee"`
	FeeToken                   currency.Code        `json:"feeToken"`
	Nonce                      uint64               `json:"nonce"`
	Vault                      string               `json:"vault"`
	RequestedUSD               types.Number         `json:"requestedUsd"`
	Commission                 types.Number         `json:"commission"`
	ClosingCost                types.Number         `json:"closingCost"`
	Basis                      types.Number         `json:"basis"`
	NetWithdrawnUSD            types.Number         `json:"netWithdrawnUsd"`
	AccountValue               types.Number         `json:"accountValue"`
	LeverageType               string               `json:"leverageType"`
	LiquidatedNotionalPosition types.Number         `json:"liquidatedNtlPos"`
	LiquidatedPositions        []LiquidatedPosition `json:"liquidatedPositions"`
	// IsDeposit is true for a cStakingTransfer into staking and false for one out of it
	IsDeposit bool `json:"isDeposit"`
	// Operation is the borrowLend operation: supply, withdraw, repay or borrow
	Operation      string       `json:"operation"`
	InterestAmount types.Number `json:"interestAmount"`
	// DEX is the DEX an activateDexAbstraction update activated, empty for the first perpetual DEX
	DEX string `json:"dex"`
}

// LiquidatedPosition contains one position closed by a liquidation
type LiquidatedPosition struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin       string       `json:"coin"`
	SignedSize types.Number `json:"szi"`
}

// FundingHistoryRequest contains the coin and inclusive time range of a funding history request; a zero EndTime ends
// the range now
type FundingHistoryRequest struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin      string
	StartTime time.Time
	EndTime   time.Time
}

// FundingRateHistoryResponse contains one hourly perpetual funding rate
type FundingRateHistoryResponse struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin        string       `json:"coin"`
	FundingRate types.Number `json:"fundingRate"`
	Premium     types.Number `json:"premium"`
	Time        types.Time   `json:"time"`
}

// PredictedFundingResponse contains a coin's predicted funding on each venue
type PredictedFundingResponse struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin   string
	Venues []PredictedFundingVenue
}

// UnmarshalJSON decodes a [coin, venues] tuple
func (p *PredictedFundingResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Coin, &p.Venues)
}

// PredictedFundingVenue contains one venue's predicted funding; Funding is nil when the venue does not list the coin
type PredictedFundingVenue struct {
	Venue   string
	Funding *PredictedFundingRate
}

// UnmarshalJSON decodes a [venue, funding] tuple
func (p *PredictedFundingVenue) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Venue, &p.Funding)
}

// PredictedFundingRate contains a venue's predicted rate for its next funding interval
type PredictedFundingRate struct {
	FundingRate     types.Number `json:"fundingRate"`
	NextFundingTime types.Time   `json:"nextFundingTime"`
	// FundingIntervalHours is omitted for some Binance markets, which then report a zero rate
	FundingIntervalHours uint64 `json:"fundingIntervalHours"`
}

// ActiveAssetDataResponse contains an account's leverage and trading capacity for one perpetual market
type ActiveAssetDataResponse struct {
	User string `json:"user"`
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin     string   `json:"coin"`
	Leverage Leverage `json:"leverage"`
	// MaxTradeSizes and AvailableToTrade contain the buy then the sell values
	MaxTradeSizes    [2]types.Number `json:"maxTradeSzs"`
	AvailableToTrade [2]types.Number `json:"availableToTrade"`
	MarkPrice        types.Number    `json:"markPx"`
}

// PerpetualDEXLimitsResponse contains a builder-deployed perpetual DEX's open interest and transfer limits
type PerpetualDEXLimitsResponse struct {
	// TotalOpenInterestCap is the notional open interest cap summed over every market of the DEX
	TotalOpenInterestCap types.Number `json:"totalOiCap"`
	// OpenInterestSizeCapPerPerpetual is each market's size-denominated open interest cap
	OpenInterestSizeCapPerPerpetual types.Number `json:"oiSzCapPerPerp"`
	MaxTransferNotional             types.Number `json:"maxTransferNtl"`
	// CoinToOpenInterestCap contains each market's notional open interest cap
	CoinToOpenInterestCap []AssetValue `json:"coinToOiCap"`
}

// PerpetualDEXStatusResponse contains a perpetual DEX's total net deposits
type PerpetualDEXStatusResponse struct {
	TotalNetDeposit types.Number `json:"totalNetDeposit"`
}

// PerpetualAnnotationResponse contains a builder-deployed perpetual market's display annotation
type PerpetualAnnotationResponse struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	DisplayName string   `json:"displayName"`
	Keywords    []string `json:"keywords"`
}

// PerpetualCategoryResponse contains a builder-deployed perpetual market's category
type PerpetualCategoryResponse struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin     string
	Category string
}

// UnmarshalJSON decodes a [coin, category] tuple
func (p *PerpetualCategoryResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Coin, &p.Category)
}

// PerpetualConciseAnnotationResponse contains a builder-deployed perpetual market's annotation, whose Description is never
// populated
type PerpetualConciseAnnotationResponse struct {
	// Coin is the perpetual's case-sensitive market name, such as BTC, kPEPE or xyz:XYZ100, rather than a currency code
	Coin       string
	Annotation PerpetualAnnotationResponse
}

// UnmarshalJSON decodes a [coin, annotation] tuple
func (p *PerpetualConciseAnnotationResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &p.Coin, &p.Annotation)
}

// UserStarStateResponse contains an account's approval state on every HIP-3* perpetual DEX that has approved it
type UserStarStateResponse struct {
	// DEXToState is sent as a list of [dex, state] tuples, not the object keyed by DEX that the documentation shows
	DEXToState []DEXStarState `json:"dexToState"`
}

// DEXStarState contains an account's approval state on one HIP-3* perpetual DEX; State is nil once the DEX has removed
// the account's approval
type DEXStarState struct {
	DEX   string
	State *StarState
}

// UnmarshalJSON decodes a [dex, state] tuple
func (d *DEXStarState) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &d.DEX, &d.State)
}

// StarState contains the flags a HIP-3* perpetual DEX deployer sets for an approved account
type StarState struct {
	// IsReduceOnly restricts the account to reducing its positions on the DEX
	IsReduceOnly bool `json:"isReduceOnly"`
	// IsBackstopLiquidatorDepositAllowed allows the account to deposit into and withdraw from the DEX's backstop
	// liquidator
	IsBackstopLiquidatorDepositAllowed bool `json:"isBackstopLiquidatorDepositAllowed"`
}
