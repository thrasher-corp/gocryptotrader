package dydx

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	timeFormat = "2006-01-02T15:04:05.999Z"
)

var (
	eip712OnboardingActionStructString = "dYdX(string action,string onlySignOn)"

	eip712OnboardingActionsStructTestnet = []map[string]string{
		{"type": "string", "name": "action"},
	}

	onlySignOnDomainMainnet = "https://trade.dydx.exchange"
)
var (
	eip712OnboardingActionsStruct = []map[string]string{
		{"type": "string", "name": "action"},
		{"type": "string", "name": "onlySignOn"},
	}
)

const (
	eip712StructName       = "dYdX"
	web3ProviderURL        = "http://localhost:8545"
	defaultEthereumAddress = "0x22d491Bde2303f2f43325b2108D26f1eAbA1e32b"
)

const (
	domain                       = "dYdX"
	version                      = "1.0"
	eip712DomainStringNoContract = "EIP712Domain(string name,string version,uint256 chainId)"
	ethSignMethod                = "eth_sign"
)

const (
	offChainOnboardingAction    = "dYdX Onboarding"
	offChainKeyDerivationAction = "dYdX STARK Key"

	ethereumSigningTemplate = `{
		"types": {
			"EIP712Domain": [
			  {"name": "name", "type": "string"},
			  {"name": "version", "type": "string"},
			  {"name": "chainId", "type": "uint64"}
			],
			"Message": [
			  {"name": "method", "type": "string"},
			  {"name": "requestPath", "type": "string"},
			  {"name": "body", "type": "string"},
			  {"name": "timestamp", "type": "string"}
			]
		  },
		  "primaryType": "dYdX",
		  "domain": {
			"name": "dydx",
			"version": "1.0",
			"chainId": 1
		  },
		  "message": {
			"method": "%s",
			"requestPath": "%s",
			"body": "%s",
			"timestamp": %s
		  }
	}`
)

// APIKeyCredentials represents authentication credentials {API Credentials} information.
type APIKeyCredentials struct {
	Key        string `json:"key"`
	Secret     string `json:"passphrase"`
	Passphrase string `json:"secret"`
}

var (
	eip712OnboardingActionStruct = []map[string]string{
		{"type": "string", "name": "action"},
		{"type": "string", "name": "onlySignOn"},
	}
	eip712OnboardingActionStructTestnet = []map[string]string{
		{"type": "string", "name": "action"},
	}
)

// InstrumentDatas metadata about each retrieved market.
type InstrumentDatas struct {
	Markets map[string]MarketDataDetail `json:"markets"`
}

// MarketDataDetail represents a market data detail
type MarketDataDetail *struct {
	Market                           string       `json:"market"`
	Status                           string       `json:"status"`
	BaseAsset                        string       `json:"baseAsset"`
	QuoteAsset                       string       `json:"quoteAsset"`
	StepSize                         types.Number `json:"stepSize"`
	TickSize                         types.Number `json:"tickSize"`
	IndexPrice                       types.Number `json:"indexPrice"`
	OraclePrice                      types.Number `json:"oraclePrice"`
	PriceChange24H                   types.Number `json:"priceChange24H"`
	NextFundingRate                  types.Number `json:"nextFundingRate"`
	NextFundingAt                    time.Time    `json:"nextFundingAt"`
	MinOrderSize                     types.Number `json:"minOrderSize"`
	Type                             string       `json:"type"`
	InitialMarginFraction            types.Number `json:"initialMarginFraction"`
	MaintenanceMarginFraction        types.Number `json:"maintenanceMarginFraction"`
	BaselinePositionSize             types.Number `json:"baselinePositionSize"`
	IncrementalPositionSize          types.Number `json:"incrementalPositionSize"`
	IncrementalInitialMarginFraction types.Number `json:"incrementalInitialMarginFraction"`
	Volume24H                        types.Number `json:"volume24H"`
	Trades24H                        types.Number `json:"trades24H"`
	OpenInterest                     string       `json:"openInterest"`
	MaxPositionSize                  types.Number `json:"maxPositionSize"`
	AssetResolution                  string       `json:"assetResolution"`
	SyntheticAssetID                 string       `json:"syntheticAssetId"`
	TransferMarginFraction           string       `json:"transferMarginFraction"`
}

// MarketOrderbook represents  bids and asks that are fillable are returned.
type MarketOrderbook struct {
	Bids orderbookDatas `json:"bids"`
	Asks orderbookDatas `json:"asks"`
}

type wsOrderbookUpdate [][2]string

// MarketOrderbookUpdate represents  bids and asks updates
type MarketOrderbookUpdate struct {
	Bids   wsOrderbookUpdate `json:"bids"`
	Asks   wsOrderbookUpdate `json:"asks"`
	Offset string            `json:"offset"`
}

func (a wsOrderbookUpdate) generateOrderbookItem() ([]orderbook.Tranche, error) {
	books := make([]orderbook.Tranche, len(a))
	for x := range a {
		price, err := strconv.ParseFloat(a[x][0], 64)
		if err != nil {
			return nil, err
		}
		size, err := strconv.ParseFloat(a[x][1], 64)
		if err != nil {
			return nil, err
		}
		books[x] = orderbook.Tranche{
			Amount: size,
			Price:  price,
		}
	}
	return books, nil
}

// OrderbookData represents asks and bids price and size data.
type OrderbookData struct {
	Offset string       `json:"offset"`
	Price  types.Number `json:"price"`
	Size   types.Number `json:"size"`
}

type orderbookDatas []OrderbookData

func (a orderbookDatas) generateOrderbookItem() []orderbook.Tranche {
	books := make([]orderbook.Tranche, len(a))
	for x := range a {
		books[x] = orderbook.Tranche{
			Price:  a[x].Price.Float64(),
			Amount: a[x].Size.Float64(),
		}
	}
	return books
}

// MarketTrades represents trade information for specific market(instrument).
type MarketTrades struct {
	Trades []MarketTrade `json:"trades"`
}

// MarketTrade represents a market trade item.
type MarketTrade struct {
	Side        string       `json:"side"`
	Size        types.Number `json:"size"`
	Price       types.Number `json:"price"`
	CreatedAt   time.Time    `json:"createdAt"`
	Liquidation bool         `json:"liquidation"`
}

// WithdrawalLiquidityResponse represents accounts that have available funds for fast withdrawals.
type WithdrawalLiquidityResponse struct {
	LiquidityProviders map[string]LiquidityProvider `json:"liquidityProviders"`
}

// LiquidityProvider represents a liquidation provider item data
type LiquidityProvider struct {
	AvailableFunds types.Number `json:"availableFunds"`
	StarkKey       string       `json:"starkKey"`
	Quote          string       `json:"quote"`
}

// TickerDatas represents market's statistics data.
type TickerDatas struct {
	Markets map[string]TickerData `json:"markets"`
}

// TickerData represents ticker data for a market.
type TickerData struct {
	Market      string       `json:"market"`
	Open        types.Number `json:"open"`
	Close       types.Number `json:"close"`
	High        types.Number `json:"high"`
	Low         types.Number `json:"low"`
	BaseVolume  types.Number `json:"baseVolume"`
	QuoteVolume types.Number `json:"quoteVolume"`
	Type        string       `json:"type"`
	Fees        types.Number `json:"fees"`
}

// HistoricFundingResponse represents a historic funding response data.
type HistoricFundingResponse struct {
	HistoricalFundings []HistoricalFunding `json:"historicalFunding"`
}

// HistoricalFunding represents historical funding rates for a market.
type HistoricalFunding struct {
	Market      string       `json:"market"`
	Rate        types.Number `json:"rate"`
	Price       types.Number `json:"price"`
	EffectiveAt time.Time    `json:"effectiveAt"`
}

// MarketCandlesResponse represents response data for market candlestick data.
type MarketCandlesResponse struct {
	Candles []MarketCandle `json:"candles"`
}

// MarketCandle represents candle statistics for a specific market.
type MarketCandle struct {
	StartedAt            time.Time    `json:"startedAt"`
	UpdatedAt            time.Time    `json:"updatedAt"`
	Market               string       `json:"market"`
	Resolution           string       `json:"resolution"`
	Low                  types.Number `json:"low"`
	High                 types.Number `json:"high"`
	Open                 types.Number `json:"open"`
	Close                types.Number `json:"close"`
	BaseTokenVolume      types.Number `json:"baseTokenVolume"`
	Trades               string       `json:"trades"`
	UsdVolume            types.Number `json:"usdVolume"`
	StartingOpenInterest string       `json:"startingOpenInterest"`
}

// ConfigurationVariableResponse represents any configuration variables for the exchange.
type ConfigurationVariableResponse struct {
	CollateralAssetID             string       `json:"collateralAssetId"`
	CollateralTokenAddress        string       `json:"collateralTokenAddress"`
	DefaultMakerFee               types.Number `json:"defaultMakerFee"`
	DefaultTakerFee               types.Number `json:"defaultTakerFee"`
	ExchangeAddress               string       `json:"exchangeAddress"`
	MaxExpectedBatchLengthMinutes types.Number `json:"maxExpectedBatchLengthMinutes"`
	MaxFastWithdrawalAmount       types.Number `json:"maxFastWithdrawalAmount"`
	CancelOrderRateLimiting       struct {
		MaxPointsMulti  int64 `json:"maxPointsMulti"`
		MaxPointsSingle int64 `json:"maxPointsSingle"`
		WindowSecMulti  int64 `json:"windowSecMulti"`
		WindowSecSingle int64 `json:"windowSecSingle"`
	} `json:"cancelOrderRateLimiting"`
	PlaceOrderRateLimiting struct {
		MaxPoints                 int64 `json:"maxPoints"`
		WindowSec                 int64 `json:"windowSec"`
		TargetNotional            int64 `json:"targetNotional"`
		MinLimitConsumption       int64 `json:"minLimitConsumption"`
		MinMarketConsumption      int64 `json:"minMarketConsumption"`
		MinTriggerableConsumption int64 `json:"minTriggerableConsumption"`
		MaxOrderConsumption       int64 `json:"maxOrderConsumption"`
	} `json:"placeOrderRateLimiting"`
}

// APIServerTime represents the server time in ISO(string) and Epoch milliseconds.
type APIServerTime struct {
	ISO   string               `json:"iso"`
	Epoch convert.ExchangeTime `json:"epoch"`
}

// LeaderboardPNLs represents top PNLs for a specified period and how they rank against
type LeaderboardPNLs struct {
	PrizePool         int64   `json:"prizePool"`
	NumHedgiesWinners int64   `json:"numHedgiesWinners"`
	NumPrizeWinners   int64   `json:"numPrizeWinners"`
	RatioPromoted     float64 `json:"ratioPromoted"`
	RatioDemoted      float64 `json:"ratioDemoted"`
	MinimumEquity     int64   `json:"minimumEquity"`
	MinimumDYDXTokens int64   `json:"minimumDYDXTokens"`
	SeasonNumber      int64   `json:"seasonNumber"`
	TopPnls           []struct {
		Username              string `json:"username"`
		EthereumAddress       string `json:"ethereumAddress"`
		PublicID              string `json:"publicId"`
		AbsolutePnl           string `json:"absolutePnl"`
		PercentPnl            string `json:"percentPnl"`
		AbsoluteRank          int64  `json:"absoluteRank"`
		PercentRank           int64  `json:"percentRank"`
		SeasonExpectedOutcome string `json:"seasonExpectedOutcome"`
		HedgieWon             int64  `json:"hedgieWon"`
		PrizeWon              int64  `json:"prizeWon"`
	} `json:"topPnls"`
	NumParticipants int       `json:"numParticipants"`
	UpdatedAt       time.Time `json:"updatedAt"`
	StartedAt       time.Time `json:"startedAt"`
	EndsAt          time.Time `json:"endsAt"`
}

// RetroactiveMiningReward represents retroactive mining rewards for an ethereum address.
type RetroactiveMiningReward struct {
	Allocation   string `json:"allocation"`
	TargetVolume string `json:"targetVolume"`
}

// CurrentRevealedHedgies represents hedgies for competition distribution
type CurrentRevealedHedgies struct {
	Daily struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		CompetitionPeriod int64    `json:"competitionPeriod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"daily"`
	Weekly struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		Competitionperiod int64    `json:"competitionperiod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"weekly"`
}

// HistoricalRevealedHedgies represents historically revealed Hedgies.
type HistoricalRevealedHedgies struct {
	HistoricalTokenIds []struct {
		BlockNumber       int64    `json:"blockNumber,string"`
		CompetitionPeriod int64    `json:"competitionperiod"`
		TokenIds          []string `json:"tokenIds"`
	} `json:"historicalTokenIds"`
}

// InsuranceFundBalance represents balance of the dYdX insurance fund.
type InsuranceFundBalance struct {
	Balance string `json:"balance"`
}

// PublicProfile represents the public profile of a user given their public ID.
type PublicProfile struct {
	Username           string `json:"username"`
	EthereumAddress    string `json:"ethereumAddress"`
	DYDXHoldings       string `json:"DYDXHoldings"`
	StakedDYDXHoldings string `json:"stakedDYDXHoldings"`
	HedgiesHeld        []int  `json:"hedgiesHeld"`
	TwitterHandle      string `json:"twitterHandle"`
	TradingLeagues     struct {
		CurrentLeague        string `json:"currentLeague"`
		CurrentLeagueRanking int    `json:"currentLeagueRanking"`
	} `json:"tradingLeagues"`
	TradingPnls struct {
		AbsolutePnl30D string `json:"absolutePnl30D"`
		PercentPnl30D  string `json:"percentPnl30D"`
		Volume30D      string `json:"volume30D"`
	} `json:"tradingPnls"`
	TradingRewards struct {
		CurEpoch                  string `json:"curEpoch"`
		CurEpochEstimatedRewards  int    `json:"curEpochEstimatedRewards"`
		PrevEpochEstimatedRewards int    `json:"prevEpochEstimatedRewards"`
	} `json:"tradingRewards"`
}

// WsInput represents a websocket input
type WsInput struct {
	Type    string `json:"type"`
	Channel string `json:"channel,omitempty"`
	ID      string `json:"id,omitempty"`

	// for authenticated channel subscription
	AccountNumber string `json:"accountNumber,omitempty"`
	APIKey        string `json:"apiKey,omitempty"`
	Signature     string `json:"signature,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	Passphrase    string `json:"passphrase,omitempty"`
}

// WsResponse represents a websocket response.
type WsResponse struct {
	Type         string          `json:"type,omitempty"`
	ConnectionID string          `json:"connection_id,omitempty"`
	MessageID    int64           `json:"message_id,omitempty"`
	Channel      string          `json:"channel,omitempty"`
	ID           string          `json:"id,omitempty"`
	Contents     json.RawMessage `json:"contents,omitempty"`
	Transfers    json.RawMessage `json:"transfers,omitempty"`
}

// OnboardingResponse represents an onboarding detail.
type OnboardingResponse struct {
	APIKey APIKeyCredentials `json:"apiKey"`
	User   struct {
		EthereumAddress         string      `json:"ethereumAddress"`
		IsRegistered            bool        `json:"isRegistered"`
		Email                   string      `json:"email"`
		Username                string      `json:"username"`
		ReferredByAffiliateLink interface{} `json:"referredByAffiliateLink"`
		MakerFeeRate            string      `json:"makerFeeRate"`
		TakerFeeRate            string      `json:"takerFeeRate"`
		MakerVolume30D          string      `json:"makerVolume30D"`
		TakerVolume30D          string      `json:"takerVolume30D"`
		Fees30D                 string      `json:"fees30D"`
		UserData                struct {
		} `json:"userData"`
		DydxTokenBalance       string      `json:"dydxTokenBalance"`
		StakedDydxTokenBalance string      `json:"stakedDydxTokenBalance"`
		IsEmailVerified        bool        `json:"isEmailVerified"`
		IsSharingUsername      interface{} `json:"isSharingUsername"`
		IsSharingAddress       bool        `json:"isSharingAddress"`
		Country                string      `json:"country"`
	} `json:"user"`
	Account Account `json:"account"`
}

// PositionResponse represents a position list data.
type PositionResponse struct {
	Positions []Position `json:"positions"`
}

// Position represents a user position information.
type Position struct {
	Market        string      `json:"market"`
	Status        string      `json:"status"`
	Side          string      `json:"side"`
	Size          string      `json:"size"`
	MaxSize       string      `json:"maxSize"`
	EntryPrice    string      `json:"entryPrice"`
	ExitPrice     interface{} `json:"exitPrice"`
	UnrealizedPnl string      `json:"unrealizedPnl"`
	RealizedPnl   string      `json:"realizedPnl"`
	CreatedAt     time.Time   `json:"createdAt"`
	ClosedAt      interface{} `json:"closedAt"`
	NetFunding    string      `json:"netFunding"`
	SumOpen       string      `json:"sumOpen"`
	SumClose      string      `json:"sumClose"`
}

// UsersResponse represents a user response detail for authenticated user.
type UsersResponse struct {
	User User `json:"user"`
}

// User represents a user account information.
type User struct {
	PublicID                     string         `json:"publicId"`
	EthereumAddress              string         `json:"ethereumAddress"`
	IsRegistered                 bool           `json:"isRegistered"`
	Email                        string         `json:"email"`
	Username                     string         `json:"username"`
	UserData                     UserDataDetail `json:"userData"`
	MakerFeeRate                 types.Number   `json:"makerFeeRate"`
	TakerFeeRate                 types.Number   `json:"takerFeeRate"`
	MakerVolume30D               string         `json:"makerVolume30D"`
	TakerVolume30D               string         `json:"takerVolume30D"`
	Fees30D                      string         `json:"fees30D"`
	ReferredByAffiliateLink      string         `json:"referredByAffiliateLink"`
	IsSharingUsername            bool           `json:"isSharingUsername"`
	IsSharingAddress             bool           `json:"isSharingAddress"`
	DydxTokenBalance             string         `json:"dydxTokenBalance"`
	StakedDydxTokenBalance       string         `json:"stakedDydxTokenBalance"`
	ActiveStakedDydxTokenBalance string         `json:"activeStakedDydxTokenBalance"`
	IsEmailVerified              bool           `json:"isEmailVerified"`
	Country                      interface{}    `json:"country"`
	HedgiesHeld                  []interface{}  `json:"hedgiesHeld"`
}

// UpdateUserParams request parameters for updating user information.
type UpdateUserParams struct {
	UserData          map[string]string `json:"userData"`
	Email             string            `json:"email,omitempty"`
	Username          string            `json:"username,omitempty"`
	IsSharingUsername bool              `json:"isSharingUsername,omitempty"`
	IsSharingAddress  bool              `json:"isSharingAddress,omitempty"`
	Country           string            `json:"country,omitempty"`
	LanguageCode      string            `json:"languageCode,omitempty"`
}

// UserDataDetail represents user data detailed information.
type UserDataDetail struct {
	WalletType  string `json:"walletType"`
	Preferences struct {
		SaveOrderAmount  bool `json:"saveOrderAmount"`
		UserTradeOptions map[string]struct {
			PostOnlyChecked           bool   `json:"postOnlyChecked"`
			GoodTilTimeInput          string `json:"goodTilTimeInput"`
			GoodTilTimeTimescale      string `json:"goodTilTimeTimescale"`
			SelectedTimeInForceOption string `json:"selectedTimeInForceOption"`
		} `json:"userTradeOptions"`
		PopUpNotifications      bool      `json:"popUpNotifications"`
		OrderbookAnimations     bool      `json:"orderbookAnimations"`
		OneTimeNotifications    []string  `json:"oneTimeNotifications"`
		LeaguesCurrentStartDate time.Time `json:"leaguesCurrentStartDate"`
	} `json:"preferences"`
	Notifications map[string]struct {
		Email bool `json:"email"`
	} `json:"notifications"`
	StarredMarkets []interface{} `json:"starredMarkets"`
}

// UserActiveLink represents a user's active link to the specified user type.
type UserActiveLink struct {
	UserType           string `json:"userType"`
	PrimaryAddress     string `json:"primaryAddress"`
	SecondaryAddresses string `json:"secondaryAddresses"`
}

// UserLinkParams represents a user's link request parameters.
type UserLinkParams struct {
	Action  string `json:"action,omitempty"`
	Address string `json:"address,omitempty"`
}

// UserPendingLink represents a user's pending link request
type UserPendingLink struct {
	UserType         string   `json:"userType"`
	OutgoingRequests []string `json:"outgoingRequests"`
	IncomingRequests []struct {
		PrimaryAddress   string `json:"primaryAddress"`
		SecondaryAddress string `json:"secondaryAddress"`
	} `json:"incomingRequests"`
}

// AccountLeaderboardPNL represents a leaderboard
type AccountLeaderboardPNL struct {
	AbsolutePnl           string    `json:"absolutePnl"`
	PercentPnl            string    `json:"percentPnl"`
	AbsoluteRank          int       `json:"absoluteRank"`
	PercentRank           int       `json:"percentRank"`
	StartedAt             time.Time `json:"startedAt"`
	EndsAt                time.Time `json:"endsAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	AccountID             string    `json:"accountId"`
	Period                string    `json:"period"`
	SeasonExpectedOutcome string    `json:"seasonExpectedOutcome"`
	SeasonNumber          int       `json:"seasonNumber"`
	HedgieWon             string    `json:"hedgieWon"`
	PrizeWon              string    `json:"prizeWon"`
}

// AccountHistorical represents an account's historical leaderboard pnls.
type AccountHistorical struct {
	LeaderboardPnls []struct {
		AbsolutePnl   string      `json:"absolutePnl"`
		PercentPnl    string      `json:"percentPnl"`
		AbsoluteRank  int         `json:"absoluteRank"`
		PercentRank   int         `json:"percentRank"`
		StartedAt     time.Time   `json:"startedAt"`
		EndsAt        time.Time   `json:"endsAt"`
		UpdatedAt     time.Time   `json:"updatedAt"`
		AccountID     string      `json:"accountId"`
		Period        string      `json:"period"`
		SeasonOutcome string      `json:"seasonOutcome"`
		SeasonNumber  int         `json:"seasonNumber"`
		HedgieWon     interface{} `json:"hedgieWon"`
		PrizeWon      string      `json:"prizeWon"`
	} `json:"leaderboardPnls"`
}

// AccountsResponse represents the accounts response.
type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

// AccountResponse represents the list of accounts instance.
type AccountResponse struct {
	Account *Account `json:"accounts"`
}

// Account represents a user account instance.
type Account struct {
	ID                 string              `json:"id"`
	PositionID         string              `json:"positionId"`
	UserID             string              `json:"userId"`
	AccountNumber      string              `json:"accountNumber"`
	StarkKey           string              `json:"starkKey"`
	QuoteBalance       types.Number        `json:"quoteBalance"`
	PendingDeposits    types.Number        `json:"pendingDeposits"`
	PendingWithdrawals types.Number        `json:"pendingWithdrawals"`
	LastTransactionID  string              `json:"lastTransactionId"`
	Equity             string              `json:"equity"`
	FreeCollateral     types.Number        `json:"freeCollateral"`
	OpenPositions      map[string]Position `json:"openPositions"`
	CreatedAt          time.Time           `json:"createdAt"`
}

// TransfersResponse transfers for a user
type TransfersResponse struct {
	Transfers []TransferResponse `json:"transfers"`
}

// WithdrawalResponse withdrawals for a user
type WithdrawalResponse struct {
	Withdrawal TransferResponse `json:"withdrawal"`
}

// TransferResponse represents a user's transfer request response.
type TransferResponse struct {
	ID              string       `json:"id"`
	Type            string       `json:"type"`
	DebitAsset      string       `json:"debitAsset"`
	CreditAsset     string       `json:"creditAsset"`
	DebitAmount     types.Number `json:"debitAmount"`
	CreditAmount    types.Number `json:"creditAmount"`
	TransactionHash string       `json:"transactionHash"`
	Status          string       `json:"status"`
	CreatedAt       time.Time    `json:"createdAt"`
	ConfirmedAt     time.Time    `json:"confirmedAt"`
	ClientID        string       `json:"clientId"`
	FromAddress     string       `json:"fromAddress"`
	ToAddress       string       `json:"toAddress"`
}

// CreateOrderRequestParams represents parameters for creating a new order.
type CreateOrderRequestParams struct {
	Market          string      `json:"market"`
	Side            string      `json:"side"`
	Type            string      `json:"type"`
	PostOnly        bool        `json:"postOnly"`
	Size            float64     `json:"size,string"`
	Price           float64     `json:"price,string"`
	LimitFee        float64     `json:"limitFee"`
	Expiration      dYdXTimeUTC `json:"expiration,omitempty"`
	TimeInForce     string      `json:"timeInForce,omitempty"`
	Cancelled       bool        `json:"cancelId,string"`
	TriggerPrice    float64     `json:"triggerPrice,omitempty,string"`
	TrailingPercent float64     `json:"trailingPercent,omitempty,string"`
	ReduceOnly      bool        `json:"reduceOnly,omitempty"`
	ClientID        string      `json:"clientId"`
	Signature       string      `json:"signature"`
}

// OrderResponse represents an order response data.
type OrderResponse struct {
	Order Order `json:"order"`
}

// Order represents a single order instance.
type Order struct {
	ID               string       `json:"id"`
	ClientAssignedID string       `json:"clientId"`
	AccountID        string       `json:"accountId"`
	Market           string       `json:"market"`
	Side             string       `json:"side"`
	Price            types.Number `json:"price"`
	TriggerPrice     types.Number `json:"triggerPrice"`
	TrailingPercent  types.Number `json:"trailingPercent"`
	Size             types.Number `json:"size"`
	RemainingSize    types.Number `json:"remainingSize"`
	Type             string       `json:"type"`
	CreatedAt        time.Time    `json:"createdAt"`
	UnfillableAt     interface{}  `json:"unfillableAt"`
	ExpiresAt        time.Time    `json:"expiresAt"`
	Status           string       `json:"status"`
	TimeInForce      string       `json:"timeInForce"`
	PostOnly         bool         `json:"postOnly"`
	ReduceOnly       bool         `json:"reduceOnly"`
	CancelReason     string       `json:"cancelReason"`
	LimitFee         types.Number `json:"limitFee"`
	Signature        string       `json:"signature"`
}

// OrderFills  represents list of order fills.
type OrderFills struct {
	Fills []OrderFill `json:"fills"`
}

// OrderFill represents order fill.
type OrderFill struct {
	ID        string    `json:"id"`
	Side      string    `json:"side"`
	Liquidity string    `json:"liquidity"`
	Type      string    `json:"type"`
	Market    string    `json:"market"`
	OrderID   string    `json:"orderId"`
	Price     string    `json:"price"`
	Size      string    `json:"size"`
	Fee       string    `json:"fee"`
	CreatedAt time.Time `json:"createdAt"`
}

// FundingPayments represents a list of funding payments
type FundingPayments struct {
	FundingPayments []FundingPayment `json:"fundingPayments"`
}

// FundingPayment represents a funding payment instance.
type FundingPayment struct {
	Market       string    `json:"market"`
	Payment      string    `json:"payment"`
	Rate         string    `json:"rate"`
	PositionSize string    `json:"positionSize"`
	Price        string    `json:"price"`
	EffectiveAt  time.Time `json:"effectiveAt"`
}

// HistoricPNLResponse represents a historic PNL response.
type HistoricPNLResponse struct {
	HistoricalPNL []HistoricPNL `json:"historicalPnl"`
}

// HistoricPNL represents a historical PNL instance.
type HistoricPNL struct {
	Equity       string    `json:"equity"`
	TotalPnl     string    `json:"totalPnl"`
	CreatedAt    time.Time `json:"createdAt"`
	NetTransfers string    `json:"netTransfers"`
	AccountID    string    `json:"accountId"`
}

// TradingRewards represents trading rewards detail.
type TradingRewards struct {
	Epoch      int64     `json:"epoch"`
	EpochStart time.Time `json:"epochStart"`
	EpochEnd   time.Time `json:"epochEnd"`
	Fees       struct {
		FeesPaid      string `json:"feesPaid"`
		TotalFeesPaid string `json:"totalFeesPaid"`
	} `json:"fees"`
	OpenInterest struct {
		AverageOpenInterest      string `json:"averageOpenInterest"`
		TotalAverageOpenInterest string `json:"totalAverageOpenInterest"`
	} `json:"openInterest"`
	StakedDYDX struct {
		PrimaryStakedDYDX          string `json:"primaryStakedDYDX"`
		AverageStakedDYDX          string `json:"averageStakedDYDX"`
		AverageStakedDYDXWithFloor string `json:"averageStakedDYDXWithFloor"`
		TotalAverageStakedDYDX     string `json:"totalAverageStakedDYDX"`
	} `json:"stakedDYDX"`
	Weight struct {
		Weight      string `json:"weight"`
		TotalWeight string `json:"totalWeight"`
	} `json:"weight"`
	TotalRewards     string `json:"totalRewards"`
	EstimatedRewards string `json:"estimatedRewards"`
}

// LiquidityProviderRewards represents liquidity provider rewards of a given epoch
type LiquidityProviderRewards struct {
	Epoch      int       `json:"epoch"`
	EpochStart string    `json:"epochStart"`
	EpochEnd   time.Time `json:"epochEnd"`
	Markets    map[string]struct {
		Market              string `json:"market"`
		DepthSpreadScore    string `json:"depthSpreadScore"`
		Uptime              string `json:"uptime"`
		LinkedUptime        string `json:"linkedUptime"`
		MaxUptime           string `json:"maxUptime"`
		Score               string `json:"score"`
		TotalScore          string `json:"totalScore"`
		MakerVolume         string `json:"makerVolume"`
		TotalMakerVolume    string `json:"totalMakerVolume"`
		TotalRewards        string `json:"totalRewards"`
		EstimatedRewards    string `json:"estimatedRewards"`
		SecondaryAllocation string `json:"secondaryAllocation"`
	} `json:"markets"`
	StakedDYDX struct {
		AverageStakedDYDX      string `json:"averageStakedDYDX"`
		TotalAverageStakedDYDX string `json:"totalAverageStakedDYDX"`
	} `json:"stakedDYDX"`
	LinkedAddressRewards map[string]struct {
		Markets map[string]struct {
			Market              string `json:"market"`
			DepthSpreadScore    string `json:"depthSpreadScore"`
			Uptime              string `json:"uptime"`
			LinkedUptime        string `json:"linkedUptime"`
			MaxUptime           string `json:"maxUptime"`
			Score               string `json:"score"`
			TotalScore          string `json:"totalScore"`
			MakerVolume         string `json:"makerVolume"`
			TotalMakerVolume    string `json:"totalMakerVolume"`
			TotalRewards        string `json:"totalRewards"`
			EstimatedRewards    string `json:"estimatedRewards"`
			SecondaryAllocation string `json:"secondaryAllocation"`
		} `json:"markets"`
		AverageStakedDYDX string `json:"averageStakedDYDX"`
	} `json:"linkedAddressRewards"`
}

// RetroactiveMining represents the retroactive mining rewards of a given epoch.
type RetroactiveMining struct {
	Epoch             int64     `json:"epoch"`
	EpochStart        time.Time `json:"epochStart"`
	EpochEnd          time.Time `json:"epochEnd"`
	RetroactiveMining struct {
		Allocation   string `json:"allocation"`
		TargetVolume string `json:"targetVolume"`
		Volume       string `json:"volume"`
	} `json:"retroactiveMining"`
	EstimatedRewards string `json:"estimatedRewards"`
}

// TestnetToken represents a tokens on dYdX's staging server.
type TestnetToken struct {
	Transfer struct {
		ID              string      `json:"id"`
		Type            string      `json:"type"`
		DebitAsset      string      `json:"debitAsset"`
		CreditAsset     string      `json:"creditAsset"`
		DebitAmount     string      `json:"debitAmount"`
		CreditAmount    string      `json:"creditAmount"`
		TransactionHash interface{} `json:"transactionHash"`
		Status          string      `json:"status"`
		CreatedAt       time.Time   `json:"createdAt"`
		ConfirmedAt     interface{} `json:"confirmedAt"`
		ClientID        string      `json:"clientId"`
		FromAddress     interface{} `json:"fromAddress"`
		ToAddress       interface{} `json:"toAddress"`
	} `json:"transfer"`
}

// PrivateProfile represents a profile data of the user.
type PrivateProfile struct {
	Username           string        `json:"username"`
	PublicID           string        `json:"publicId"`
	EthereumAddress    string        `json:"ethereumAddress"`
	DYDXHoldings       string        `json:"DYDXHoldings"`
	StakedDYDXHoldings string        `json:"stakedDYDXHoldings"`
	HedgiesHeld        []interface{} `json:"hedgiesHeld"`
	TwitterHandle      string        `json:"twitterHandle"`
	AffiliateLinks     []struct {
		Link         string `json:"link"`
		DiscountRate string `json:"discountRate"`
	} `json:"affiliateLinks"`
	AffiliateApplicationStatus interface{} `json:"affiliateApplicationStatus"`
	TradingLeagues             struct {
		CurrentLeague        interface{} `json:"currentLeague"`
		CurrentLeagueRanking interface{} `json:"currentLeagueRanking"`
	} `json:"tradingLeagues"`
	TradingPnls struct {
		AbsolutePnl30D string `json:"absolutePnl30D"`
		PercentPnl30D  string `json:"percentPnl30D"`
		Volume30D      string `json:"volume30D"`
	} `json:"tradingPnls"`
	TradingRewards struct {
		CurEpoch                  int    `json:"curEpoch"`
		CurEpochEstimatedRewards  string `json:"curEpochEstimatedRewards"`
		PrevEpochEstimatedRewards string `json:"prevEpochEstimatedRewards"`
	} `json:"tradingRewards"`
	AffiliateStatistics struct {
		CurrentEpoch struct {
			UsersReferred    string `json:"usersReferred"`
			Revenue          string `json:"revenue"`
			RevenueShareRate string `json:"revenueShareRate"`
		} `json:"currentEpoch"`
		PreviousEpochs struct {
			UsersReferred string `json:"usersReferred"`
			Revenue       string `json:"revenue"`
			RevenuePaid   string `json:"revenuePaid"`
		} `json:"previousEpochs"`
		LastEpochPaid string `json:"lastEpochPaid"`
	} `json:"affiliateStatistics"`
}

// FastWithdrawalParam represents a parameter for asset withdrawal
type FastWithdrawalParam struct {
	ToAddress         string      `json:"toAddress"`
	CreditAsset       string      `json:"creditAsset"`
	CreditAmount      float64     `json:"creditAmount,string"`
	DebitAmount       float64     `json:"debitAmount,string"`
	SlippageTolerance float64     `json:"slippageTolerance,omitempty,string"`
	LPPositionID      int64       `json:"lpPositionId,omitempty,string"`
	ClientID          string      `json:"clientId"`
	Expiration        dYdXTimeUTC `json:"expiration,omitempty"`
	Hey               string      `json:"Hey,omitempty"`
	Signature         string      `json:"signature"`
}

// FastWithdrawalRequestParam represents a parameter for fast withdrawal
type FastWithdrawalRequestParam struct {
	CreditAsset  currency.Code `json:"creditAsset"`
	CreditAmount float64       `json:"creditAmount,omitempty,string"`
	DebitAmount  float64       `json:"debitAmount,omitempty,string"`
}

// TransferParam represents a parameter for transfer
type TransferParam struct {
	Amount             float64     `json:"amount,string"`
	ClientID           string      `json:"clientId"`
	Expiration         dYdXTimeUTC `json:"expiration,omitempty"`
	ReceiverAccountID  string      `json:"receiverAccountId"`
	Signature          string      `json:"signature,omitempty"`
	ReceiverPublicKey  string      `json:"receiverPublicKey"`
	ReceiverPositionID string      `json:"receiverPositionID"`
}

// WithdrawalParam argument struct representing withdrawal request input.
type WithdrawalParam struct {
	Amount            float64     `json:"amount,string"`
	Asset             string      `json:"asset"`
	Expiration        dYdXTimeUTC `json:"expiration"`
	ClientGeneratedID string      `json:"clientId"`
	Signature         string      `json:"signature"`
}

// AccountSubscriptionResponse represents a subscriptions to v3_accounts subscription.
type AccountSubscriptionResponse struct {
	Type         string `json:"type"`
	Channel      string `json:"channel"`
	ConnectionID string `json:"connection_id"`
	ID           string `json:"id"`
	MessageID    int    `json:"message_id"`
	Contents     struct {
		Orders  []Order `json:"orders"`
		Account Account `json:"account"`
	} `json:"contents"`
	Transfers       []TransferResponse `json:"transfers"`
	FundingPayments []FundingPayment   `json:"fundingPayments"`
}

// AccountChannelData represents a push data message to v3_account subscription.
type AccountChannelData struct {
	Type         string `json:"type"`
	Channel      string `json:"channel"`
	ConnectionID string `json:"connection_id"`
	ID           string `json:"id"`
	MessageID    int64  `json:"message_id"`
	Contents     struct {
		Fills     []OrderFill `json:"fills"`
		Orders    []Order     `json:"orders"`
		Positions []Position  `json:"positions"`
		Accounts  []Account   `json:"accounts"`
	} `json:"contents,omitempty"`
}

// OnboardingParam represents onboarding request parameters.
type OnboardingParam struct {
	StarkXCoordinate        string `json:"starkKey"`
	StarkYCoordinate        string `json:"starkKeyYCoordinate"`
	EthereumAddress         string `json:"ethereumAddress"`
	ReferredByAffiliateLink string `json:"referredByAffiliateLink"` // Optional
	Country                 string `json:"country"`
}

// RecoverAPIKeysResponse represents parameters for recovering stark-key,quote balance, and open positions.
type RecoverAPIKeysResponse struct {
	StarkKey       string     `json:"starkKey"`
	PositionID     string     `json:"positionId"`
	Equity         string     `json:"equity"`
	FreeCollateral string     `json:"freeCollateral"`
	QuoteBalance   string     `json:"quoteBalance"`
	Positions      []Position `json:"positions"`
}

// SignatureResponse represents a signature response containing a string
// representing ethereum signature authorizing the user's Ethereum address to register for the corresponding position id.
type SignatureResponse struct {
	Signature string `json:"signature"`
}
