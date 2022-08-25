package ftx

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// MarginFundingData stores borrowing/lending data for margin trading
type MarginFundingData struct {
	Coin     currency.Code `json:"coin"`
	Estimate float64       `json:"estimate"`
	Previous float64       `json:"previous"`
}

// MarginDailyBorrowStats stores the daily borrowed amounts
type MarginDailyBorrowStats struct {
	Coin string  `json:"coin"`
	Size float64 `json:"size"`
}

// MarginMarketInfo stores margin market info
type MarginMarketInfo struct {
	Coin          string  `json:"coin"`
	Borrowed      float64 `json:"borrowed"`
	Free          float64 `json:"free"`
	EstimatedRate float64 `json:"estimatedRate"`
	PreviousRate  float64 `json:"previousRate"`
}

// MarginTransactionHistoryData stores margin borrowing/lending history
type MarginTransactionHistoryData struct {
	Coin     currency.Code `json:"coin"`
	Cost     float64       `json:"cost"`
	Rate     float64       `json:"rate"`
	Size     float64       `json:"size"`
	Proceeds float64       `json:"proceeds"`
	Time     time.Time     `json:"time"`
}

// LendingOffersData stores data for lending offers
type LendingOffersData struct {
	Coin string  `json:"coin"`
	Rate float64 `json:"rate"`
	Size float64 `json:"size"`
}

// LendingInfoData stores margin lending info
type LendingInfoData struct {
	Coin     string  `json:"coin"`
	Lendable float64 `json:"lendable"`
	Locked   float64 `json:"locked"`
	MinRate  float64 `json:"minRate"`
	Offered  float64 `json:"offered"`
}

// MarketData stores market data
type MarketData struct {
	Name                  string  `json:"name"`
	BaseCurrency          string  `json:"baseCurrency"`
	QuoteCurrency         string  `json:"quoteCurrency"`
	MarketType            string  `json:"type"`
	Underlying            string  `json:"underlying"`
	Change1h              float64 `json:"change1h"`
	Change24h             float64 `json:"change24h"`
	ChangeBod             float64 `json:"changeBod"`
	QuoteVolume24h        float64 `json:"quoteVolume24h"`
	Enabled               bool    `json:"enabled"`
	Ask                   float64 `json:"ask"`
	Bid                   float64 `json:"bid"`
	Last                  float64 `json:"last"`
	USDVolume24h          float64 `json:"volumeUSD24h"`
	MinProvideSize        float64 `json:"minProvideSize"`
	PriceIncrement        float64 `json:"priceIncrement"`
	SizeIncrement         float64 `json:"sizeIncrement"`
	Restricted            bool    `json:"restricted"`
	PostOnly              bool    `json:"postOnly"`
	Price                 float64 `json:"price"`
	HighLeverageFeeExempt bool    `json:"highLeverageFeeExempt"`
}

// OData stores orderdata in orderbook
type OData struct {
	Price float64
	Size  float64
}

// OrderbookData stores orderbook data
type OrderbookData struct {
	MarketName string
	Asks       []OData
	Bids       []OData
}

// TempOBData stores orderbook data temporarily
type TempOBData struct {
	Asks [][2]float64 `json:"asks"`
	Bids [][2]float64 `json:"bids"`
}

// TradeData stores data from trades
type TradeData struct {
	ID          int64     `json:"id"`
	Liquidation bool      `json:"liquidation"`
	Price       float64   `json:"price"`
	Side        string    `json:"side"`
	Size        float64   `json:"size"`
	Time        time.Time `json:"time"`
}

// OHLCVData stores historical OHLCV data
type OHLCVData struct {
	Close     float64   `json:"close"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Open      float64   `json:"open"`
	StartTime time.Time `json:"startTime"`
	Time      float64   `json:"time"`
	Volume    float64   `json:"volume"`
}

// FuturesData stores data for futures
type FuturesData struct {
	Ask                         float64   `json:"ask"`
	Bid                         float64   `json:"bid"`
	Change1h                    float64   `json:"change1h"`
	Change24h                   float64   `json:"change24h"`
	ChangeBod                   float64   `json:"changeBod"`
	VolumeUSD24h                float64   `json:"volumeUsd24h"`
	Volume                      float64   `json:"volume"`
	Description                 string    `json:"description"`
	Enabled                     bool      `json:"enabled"`
	Expired                     bool      `json:"expired"`
	Expiry                      time.Time `json:"expiry"`
	ExpiryDescription           string    `json:"expiryDescription"`
	Group                       string    `json:"group"`
	Index                       float64   `json:"index"`
	InitialMarginFractionFactor float64   `json:"imfFactor"`
	Last                        float64   `json:"last"`
	LowerBound                  float64   `json:"lowerBound"`
	MarginPrice                 float64   `json:"marginPrice"`
	Mark                        float64   `json:"mark"`
	MoveStart                   time.Time `json:"moveStart"`
	Name                        string    `json:"name"`
	OpenInterest                float64   `json:"openInterest"`
	OpenInterestUSD             float64   `json:"openInterestUsd"`
	Perpetual                   bool      `json:"perpetual"`
	PositionLimitWeight         float64   `json:"positionLimitWeight"`
	PostOnly                    bool      `json:"postOnly"`
	PriceIncrement              float64   `json:"priceIncrement"`
	SizeIncrement               float64   `json:"sizeIncrement"`
	Underlying                  string    `json:"underlying"`
	UnderlyingDescription       string    `json:"underlyingDescription"`
	UpperBound                  float64   `json:"upperBound"`
	FutureType                  string    `json:"type"`
}

// FutureStatsData stores data on futures stats
type FutureStatsData struct {
	Volume                   float64   `json:"volume"`
	NextFundingRate          float64   `json:"nextFundingRate"`
	NextFundingTime          time.Time `json:"nextFundingTime"`
	ExpirationPrice          float64   `json:"expirationPrice"`
	PredictedExpirationPrice float64   `json:"predictedExpirationPrice"`
	OpenInterest             float64   `json:"openInterest"`
	StrikePrice              float64   `json:"strikePrice"`
	Greeks                   *struct {
		ImpliedVolatility float64 `json:"impliedVolatility"`
		Delta             float64 `json:"delta"`
		Gamma             float64 `json:"gamma"`
	} `json:"greeks"`
}

// FundingRatesData stores data on funding rates
type FundingRatesData struct {
	Future string    `json:"future"`
	Rate   float64   `json:"rate"`
	Time   time.Time `json:"time"`
}

// IndexWeights stores index weights' data
type IndexWeights struct {
	Result map[string]float64 `json:"result"`
}

// PositionData stores data of an open position
type PositionData struct {
	Future                       currency.Pair `json:"future"`
	Size                         float64       `json:"size"`
	Side                         string        `json:"side"`
	NetSize                      float64       `json:"netSize"`
	LongOrderSize                float64       `json:"longOrderSize"`
	ShortOrderSize               float64       `json:"shortOrderSize"`
	Cost                         float64       `json:"cost"`
	EntryPrice                   float64       `json:"entryPrice"`
	UnrealizedPNL                float64       `json:"unrealizedPnl"`
	RealizedPNL                  float64       `json:"realizedPnl"`
	InitialMarginRequirement     float64       `json:"initialMarginRequirement"`
	MaintenanceMarginRequirement float64       `json:"maintenanceMarginRequirement"`
	OpenSize                     float64       `json:"openSize"`
	CollateralUsed               float64       `json:"collateralUsed"`
	EstimatedLiquidationPrice    float64       `json:"estimatedLiquidationPrice"`
	RecentAverageOpenPrice       float64       `json:"recentAverageOpenPrice"`
	RecentPNL                    float64       `json:"recentPnl"`
	RecentBreakEvenPrice         float64       `json:"recentBreakEvenPrice"`
	CumulativeBuySize            float64       `json:"cumulativeBuySize"`
	CumulativeSellSize           float64       `json:"cumulativeSellSize"`
}

// AccountInfoData stores account data
type AccountInfoData struct {
	BackstopProvider             bool           `json:"backstopProvider"`
	ChargeInterestOnNegativeUSD  bool           `json:"chargeInterestOnNegativeUsd"`
	Collateral                   float64        `json:"collateral"`
	FreeCollateral               float64        `json:"freeCollateral"`
	InitialMarginRequirement     float64        `json:"initialMarginRequirement"`
	Leverage                     float64        `json:"leverage"`
	Liquidating                  bool           `json:"liquidating"`
	MaintenanceMarginRequirement float64        `json:"maintenanceMarginRequirement"`
	MakerFee                     float64        `json:"makerFee"`
	MarginFraction               float64        `json:"marginFraction"`
	OpenMarginFraction           float64        `json:"openMarginFraction"`
	PositionLimit                float64        `json:"positionLimit"`
	PositionLimitUsed            float64        `json:"positionLimitUsed"`
	SpotLendingEnabled           bool           `json:"spotLendingEnabled"`
	SpotMarginEnabled            bool           `json:"spotMarginEnabled"`
	TakerFee                     float64        `json:"takerFee"`
	TotalAccountValue            float64        `json:"totalAccountValue"`
	TotalPositionSize            float64        `json:"totalPositionSize"`
	UseFTTCollateral             bool           `json:"useFttCollateral"`
	Username                     string         `json:"username"`
	Positions                    []PositionData `json:"positions"`
}

// WalletCoinsData stores data about wallet coins
type WalletCoinsData struct {
	USDFungible      bool     `json:"usdFungible"`
	CanDeposit       bool     `json:"canDeposit"`
	CanWithdraw      bool     `json:"canWithdraw"`
	CanConvert       bool     `json:"canConvert"`
	Collateral       bool     `json:"collateral"`
	CollateralWeight float64  `json:"collateralWeight"`
	CreditTo         string   `json:"creditTo"`
	ERC20Contract    string   `json:"erc20Contract"`
	BEP2Asset        string   `json:"bep2Asset"`
	TRC20Contract    string   `json:"trc20Contract"`
	SpotMargin       bool     `json:"spotMargin"`
	IndexPrice       float64  `json:"indexPrice"`
	SPLMint          string   `json:"splMint"`
	Fiat             bool     `json:"fiat"`
	HasTag           bool     `json:"hasTag"`
	Hidden           bool     `json:"hidden"`
	IsETF            bool     `json:"isEtf"`
	IsToken          bool     `json:"isToken"`
	Methods          []string `json:"methods"`
	ID               string   `json:"id"`
	Name             string   `json:"name"`
}

// WalletBalance stores balances data
type WalletBalance struct {
	Coin                   currency.Code          `json:"coin"`
	Free                   float64                `json:"free"`
	Total                  float64                `json:"total"`
	AvailableWithoutBorrow float64                `json:"availableWithoutBorrow"`
	USDValue               float64                `json:"usdValue"`
	FreeIgnoringCollateral float64                `json:"freeIgnoringCollateral"`
	SpotBorrow             float64                `json:"spotBorrow"`
	LockedBreakdown        BalanceLockedBreakdown `json:"lockedBreakdown"`
}

// BalanceLockedBreakdown provides a breakdown of where funding is
// locked up in, helpful in tracking how much one bids on NFTs
type BalanceLockedBreakdown struct {
	LockedInStakes                  float64 `json:"lockedInStakes"`
	LockedInNFTBids                 float64 `json:"lockedInNftBids"`
	LockedInFeeVoucher              float64 `json:"lockedInFeeVoucher"`
	LockedInSpotMarginFundingOffers float64 `json:"lockedInSpotMarginFundingOffers"`
	LockedInSpotOrders              float64 `json:"lockedInSpotOrders"`
	LockedAsCollateral              float64 `json:"lockedAsCollateral"`
}

// AllWalletBalances stores all the user's account balances
type AllWalletBalances map[string][]WalletBalance

// DepositData stores deposit address data
type DepositData struct {
	Address string `json:"address"`
	Tag     string `json:"tag"`
	Method  string `json:"method"`
	Coin    string `json:"coin"`
}

// DepositItem stores data about deposit history
type DepositItem struct {
	Coin          string    `json:"coin"`
	Confirmations int64     `json:"conformations"`
	ConfirmedTime time.Time `json:"confirmedTime"`
	Fee           float64   `json:"fee"`
	ID            int64     `json:"id"`
	SentTime      time.Time `json:"sentTime"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	Time          time.Time `json:"time"`
	TxID          string    `json:"txid"`
	Address       struct {
		Address string `json:"address"`
		Tag     string `json:"tag"`
		Method  string `json:"method"`
	} `json:"address"`
}

// WithdrawItem stores data about withdraw history
type WithdrawItem struct {
	ID              int64     `json:"id"`
	Coin            string    `json:"coin"`
	Address         string    `json:"address"`
	Tag             string    `json:"tag"`
	Method          string    `json:"method"`
	TXID            string    `json:"txid"`
	Size            float64   `json:"size"`
	Fee             float64   `json:"fee"`
	Status          string    `json:"status"`
	Complete        time.Time `json:"complete"`
	Time            time.Time `json:"time"`
	Notes           string    `json:"notes"`
	DestinationName string    `json:"destinationName"`
}

// OrderData stores open order data
type OrderData struct {
	AvgFillPrice  float64       `json:"avgFillPrice"`
	ClientID      string        `json:"clientId"`
	CreatedAt     time.Time     `json:"createdAt"`
	FilledSize    float64       `json:"filledSize"`
	Future        string        `json:"future"`
	ID            int64         `json:"id"`
	IOC           bool          `json:"ioc"`
	Market        currency.Pair `json:"market"`
	PostOnly      bool          `json:"postOnly"`
	Price         float64       `json:"price"`
	ReduceOnly    bool          `json:"reduceOnly"`
	RemainingSize float64       `json:"remainingSize"`
	Side          string        `json:"side"`
	Size          float64       `json:"size"`
	Status        string        `json:"status"`
	Type          string        `json:"type"`
}

// TriggerOrderData stores trigger order data
type TriggerOrderData struct {
	CreatedAt        time.Time `json:"createdAt"`
	Error            string    `json:"error"`
	Future           string    `json:"future"`
	ID               int64     `json:"id"`
	Market           string    `json:"market"`
	OrderID          int64     `json:"orderId"`
	OrderPrice       float64   `json:"orderPrice"`
	ReduceOnly       bool      `json:"reduceOnly"`
	Side             string    `json:"side"`
	Size             float64   `json:"size"`
	Status           string    `json:"status"`
	TrailStart       float64   `json:"trailStart"`
	TrailValue       float64   `json:"trailvalue"`
	TriggerPrice     float64   `json:"triggerPrice"`
	TriggeredAt      string    `json:"triggeredAt"`
	OrderType        string    `json:"type"`
	MarketOrLimit    string    `json:"orderType"`
	FilledSize       float64   `json:"filledSize"`
	AvgFillPrice     float64   `json:"avgFillPrice"`
	RetryUntilFilled bool      `json:"retryUntilFilled"`
}

// TriggerData stores trigger orders' trigger data
type TriggerData struct {
	Error      string    `json:"error"`
	FilledSize float64   `json:"filledSize"`
	OrderSize  float64   `json:"orderSize"`
	OrderID    int64     `json:"orderId"`
	Time       time.Time `json:"time"`
}

// FillsData stores fills' data
type FillsData struct {
	Fee           float64       `json:"fee"`
	FeeCurrency   currency.Code `json:"feeCurrency"`
	FeeRate       float64       `json:"feeRate"`
	Future        string        `json:"future"`
	ID            int64         `json:"id"`
	Liquidity     string        `json:"liquidity"`
	Market        string        `json:"market"`
	BaseCurrency  string        `json:"baseCurrency"`
	QuoteCurrency string        `json:"quoteCurrency"`
	OrderID       int64         `json:"orderId"`
	TradeID       int64         `json:"tradeId"`
	Price         float64       `json:"price"`
	Side          string        `json:"side"`
	Size          float64       `json:"size"`
	Time          time.Time     `json:"time"`
	OrderType     string        `json:"type"`
}

// FundingPaymentsData stores funding payments' data
type FundingPaymentsData struct {
	Future  string    `json:"future"`
	ID      int64     `json:"id"`
	Payment float64   `json:"payment"`
	Time    time.Time `json:"time"`
	Rate    float64   `json:"rate"`
}

// LeveragedTokensData stores data of leveraged tokens
type LeveragedTokensData struct {
	Basket            map[string]interface{} `json:"basket"`
	Bep2AssetName     string                 `json:"bep2AssetName"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Underlying        string                 `json:"underlying"`
	Leverage          float64                `json:"leverage"`
	Outstanding       float64                `json:"outstanding"`
	PricePerShare     float64                `json:"pricePerShare"`
	PositionPerShare  float64                `json:"positionPerShare"`
	PositionsPerShare interface{}            `json:"positionsPerShare"`
	TargetComponents  []string               `json:"targetComponents"`
	TotalCollateral   float64                `json:"totalCollateral"`
	TotalNav          float64                `json:"totalNav"`
	UnderlyingMark    float64                `json:"underlyingMark"`
	ContactAddress    string                 `json:"contactAddress"`
	Change1h          float64                `json:"change1h"`
	Change24h         float64                `json:"change24h"`
	ChangeBod         float64                `json:"changeBod"`
}

// LTBalanceData stores balances of leveraged tokens
type LTBalanceData struct {
	Token   string  `json:"token"`
	Balance float64 `json:"balance"`
}

// LTCreationData stores token creation requests' data
type LTCreationData struct {
	ID            int64     `json:"id"`
	Token         string    `json:"token"`
	RequestedSize float64   `json:"requestedSize"`
	Pending       bool      `json:"pending"`
	CreatedSize   float64   `json:"createdSize"`
	Price         float64   `json:"price"`
	Cost          float64   `json:"cost"`
	Fee           float64   `json:"fee"`
	RequestedAt   time.Time `json:"requestedAt"`
	FulfilledAt   time.Time `json:"fulfilledAt"`
}

// RequestTokenCreationData stores data of the token creation requested
type RequestTokenCreationData struct {
	ID            int64     `json:"id"`
	Token         string    `json:"token"`
	RequestedSize float64   `json:"requestedSize"`
	Cost          float64   `json:"cost"`
	Pending       bool      `json:"pending"`
	RequestedAt   time.Time `json:"requestedAt"`
}

// LTRedemptionData stores data of the token redemption request
type LTRedemptionData struct {
	ID          int64     `json:"id"`
	Token       string    `json:"token"`
	Size        float64   `json:"size"`
	Pending     bool      `json:"pending"`
	Price       float64   `json:"price"`
	Proceeds    float64   `json:"proceeds"`
	Fee         float64   `json:"fee"`
	RequestedAt time.Time `json:"requestedAt"`
	FulfilledAt time.Time `json:"fulfilledAt"`
}

// LTRedemptionRequestData stores redemption request data for a leveraged token
type LTRedemptionRequestData struct {
	ID                int64     `json:"id"`
	Token             string    `json:"token"`
	Size              float64   `json:"size"`
	ProjectedProceeds float64   `json:"projectedProceeds"`
	Pending           bool      `json:"pending"`
	RequestedAt       time.Time `json:"requestedAt"`
}

// OptionData stores options' data
type OptionData struct {
	Underlying string    `json:"underlying"`
	OptionType string    `json:"type"`
	Strike     float64   `json:"strike"`
	Expiry     time.Time `json:"expiry"`
}

// QuoteRequestData stores option's quote request data
type QuoteRequestData struct {
	ID            int64      `json:"id"`
	Option        OptionData `json:"option"`
	Side          string     `json:"side"`
	Size          float64    `json:"size"`
	Time          time.Time  `json:"time"`
	RequestExpiry string     `json:"requestExpiry"`
	Status        string     `json:"status"`
}

// QuoteData stores quote's data
type QuoteData struct {
	Collateral  float64   `json:"collateral"`
	ID          int64     `json:"id"`
	Price       float64   `json:"price"`
	QuoteExpiry string    `json:"quoteExpiry"`
	Status      string    `json:"status"`
	Time        time.Time `json:"time"`
}

// PersonalQuotesData stores data of your quotes
type PersonalQuotesData struct {
	ID             int64       `json:"id"`
	Option         OptionData  `json:"option"`
	Side           string      `json:"side"`
	Size           float64     `json:"size"`
	Time           time.Time   `json:"time"`
	RequestExpiry  string      `json:"requestExpiry"`
	Status         string      `json:"status"`
	HideLimitPrice bool        `json:"hideLimitPrice"`
	LimitPrice     float64     `json:"limitPrice"`
	Quotes         []QuoteData `json:"quotes"`
}

// CreateQuoteRequestData stores quote data of the request sent
type CreateQuoteRequestData struct {
	ID            int64     `json:"id"`
	Expiry        time.Time `json:"expiry"`
	Strike        float64   `json:"strike"`
	OptionType    string    `json:"type"`
	Underlying    string    `json:"underlying"`
	RequestExpiry string    `json:"requestExpiry"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	Time          time.Time `json:"time"`
}

// CancelQuoteRequestData stores cancel quote request data
type CancelQuoteRequestData struct {
	ID            int64      `json:"id"`
	Option        OptionData `json:"option"`
	RequestExpiry string     `json:"requestExpiry"`
	Side          string     `json:"side"`
	Size          float64    `json:"size"`
	Status        string     `json:"status"`
	Time          time.Time  `json:"time"`
}

// QuoteForQuoteData gets quote data for your quote
type QuoteForQuoteData struct {
	Collateral  float64    `json:"collateral"`
	ID          int64      `json:"id"`
	Option      OptionData `json:"option"`
	Price       float64    `json:"price"`
	QuoteExpiry string     `json:"quoteExpiry"`
	QuoterSide  string     `json:"quoterSide"`
	RequestID   int64      `json:"requestID"`
	RequestSide string     `json:"requestSide"`
	Size        float64    `json:"size"`
	Status      string     `json:"status"`
	Time        time.Time  `json:"time"`
}

// AccountOptionsInfoData stores account's options' info data
type AccountOptionsInfoData struct {
	USDBalance       float64 `json:"usdBalance"`
	LiquidationPrice float64 `json:"liquidationPrice"`
	Liquidating      bool    `json:"liquidating"`
}

// OptionsPositionsData stores options positions' data
type OptionsPositionsData struct {
	EntryPrice            float64    `json:"entryPrice"`
	NetSize               float64    `json:"netSize"`
	Option                OptionData `json:"option"`
	Side                  string     `json:"side"`
	Size                  float64    `json:"size"`
	PessimisticValuation  float64    `json:"pessimisticValuation,omitempty"`
	PessimisticIndexPrice float64    `json:"pessimisticIndexPrice,omitempty"`
}

// OptionsTradesData stores options' trades' data
type OptionsTradesData struct {
	ID     int64      `json:"id"`
	Option OptionData `json:"option"`
	Price  float64    `json:"price"`
	Size   float64    `json:"size"`
	Time   time.Time  `json:"time"`
}

// OptionFillsData stores option's fills data
type OptionFillsData struct {
	Fee       float64    `json:"fee"`
	FeeRate   float64    `json:"feeRate"`
	ID        int64      `json:"id"`
	Liquidity string     `json:"liquidity"`
	Option    OptionData `json:"option"`
	Price     float64    `json:"price"`
	QuoteID   int64      `json:"quoteId"`
	Side      string     `json:"side"`
	Size      float64    `json:"size"`
	Time      string     `json:"time"`
}

// AuthenticationData stores authentication variables required
type AuthenticationData struct {
	Key        string `json:"key"`
	Sign       string `json:"sign"`
	Time       int64  `json:"time"`
	SubAccount string `json:"subaccount,omitempty"`
}

// Authenticate stores authentication variables required
type Authenticate struct {
	Args      AuthenticationData `json:"args"`
	Operation string             `json:"op"`
}

// WsResponseData stores basic ws response data on being subscribed to a channel successfully
type WsResponseData struct {
	ResponseType string      `json:"type"`
	Channel      string      `json:"channel"`
	Market       string      `json:"market"`
	Data         interface{} `json:"data"`
}

// WsTickerData stores ws ticker data
type WsTickerData struct {
	Bid     float64 `json:"bid"`
	Ask     float64 `json:"ask"`
	BidSize float64 `json:"bidSize"`
	AskSize float64 `json:"askSize"`
	Last    float64 `json:"last"`
	Time    float64 `json:"time"`
}

// WsTradeData stores ws trade data
type WsTradeData struct {
	ID          int64     `json:"id"`
	Price       float64   `json:"price"`
	Size        float64   `json:"size"`
	Side        string    `json:"side"`
	Liquidation bool      `json:"liquidation"`
	Time        time.Time `json:"time"`
}

// WsOrderbookData stores ws orderbook data
type WsOrderbookData struct {
	Action   string       `json:"action"`
	Bids     [][2]float64 `json:"bids"`
	Asks     [][2]float64 `json:"asks"`
	Time     float64      `json:"time"`
	Checksum int64        `json:"checksum"`
}

// WsOrders stores ws orders' data
type WsOrders struct {
	ID            int64     `json:"id"`
	ClientID      string    `json:"clientId"`
	Market        string    `json:"market"`
	OrderType     string    `json:"type"`
	Side          string    `json:"side"`
	Price         float64   `json:"price"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	FilledSize    float64   `json:"filledSize"`
	RemainingSize float64   `json:"remainingSize"`
	ReduceOnly    bool      `json:"reduceOnly"`
	Liquidation   bool      `json:"liquidation"`
	AvgFillPrice  float64   `json:"avgFillPrice"`
	PostOnly      bool      `json:"postOnly"`
	IOC           bool      `json:"ioc"`
	CreatedAt     time.Time `json:"createdAt"`
}

// WsFills stores websocket fills' data
type WsFills struct {
	ID            int64     `json:"id"`
	Market        string    `json:"market"`
	Future        string    `json:"future"`
	BaseCurrency  string    `json:"baseCurrency"`
	QuoteCurrency string    `json:"quoteCurrency"`
	Type          string    `json:"type"`
	Side          string    `json:"side"`
	Price         float64   `json:"price"`
	Size          float64   `json:"size"`
	OrderID       int64     `json:"orderId"`
	Time          time.Time `json:"time"`
	TradeID       int64     `json:"tradeId"`
	FeeRate       float64   `json:"feeRate"`
	Fee           float64   `json:"fee"`
	FeeCurrency   string    `json:"feeCurrency"`
	Liquidity     string    `json:"liquidity"`
}

// WsSub has the data used to subscribe to a channel
type WsSub struct {
	Channel   string `json:"channel,omitempty"`
	Market    string `json:"market,omitempty"`
	Operation string `json:"op,omitempty"`
}

// WsTickerDataStore stores ws ticker data
type WsTickerDataStore struct {
	Channel     string       `json:"channel"`
	Market      string       `json:"market"`
	MessageType string       `json:"type"`
	Ticker      WsTickerData `json:"data"`
}

// WsOrderbookDataStore stores ws orderbook data
type WsOrderbookDataStore struct {
	Channel     string          `json:"channel"`
	Market      string          `json:"market"`
	MessageType string          `json:"type"`
	OBData      WsOrderbookData `json:"data"`
}

// WsTradeDataStore stores ws trades' data
type WsTradeDataStore struct {
	Channel     string        `json:"channel"`
	Market      string        `json:"market"`
	MessageType string        `json:"type"`
	TradeData   []WsTradeData `json:"data"`
}

// WsOrderDataStore stores ws orders' data
type WsOrderDataStore struct {
	Channel     string   `json:"channel"`
	MessageType string   `json:"type"`
	OrderData   WsOrders `json:"data"`
}

// WsFillsDataStore stores ws fills' data
type WsFillsDataStore struct {
	Channel     string  `json:"channel"`
	MessageType string  `json:"type"`
	FillsData   WsFills `json:"data"`
}

// TimeInterval represents interval enum.
type TimeInterval string

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side      order.Side
	Status    order.Status
	OrderType order.Type
	Fee       float64
}

// WsMarketsData stores websocket markets data
type WsMarketsData struct {
	Data map[string]WsMarketsDataStorage `json:"data"`
}

// WsMarketsDataStorage stores websocket markets data
type WsMarketsDataStorage struct {
	Name           string              `json:"name"`
	Enabled        bool                `json:"enabled"`
	PriceIncrement float64             `json:"priceIncrement"`
	SizeIncrement  float64             `json:"sizeIncrement"`
	MarketType     string              `json:"marketType"`
	BaseCurrency   string              `json:"baseCurrency"`
	QuoteCurrency  string              `json:"quoteCurrency"`
	Underlying     string              `json:"underlying"`
	Restricted     bool                `json:"restricted"`
	Future         WsMarketsFutureData `json:"future"`
}

// WsMarketsFutureData stores websocket markets' future data
type WsMarketsFutureData struct {
	Name                        string    `json:"name"`
	Underlying                  string    `json:"underlying"`
	Description                 string    `json:"description"`
	MarketType                  string    `json:"type"`
	Expiry                      time.Time `json:"expiry"`
	Perpetual                   bool      `json:"perpetual"`
	Expired                     bool      `json:"expired"`
	Enabled                     bool      `json:"enabled"`
	PostOnly                    bool      `json:"postOnly"`
	InitialMarginFractionFactor float64   `json:"imfFactor"`
	UnderlyingDescription       string    `json:"underlyingDescription"`
	ExpiryDescription           string    `json:"expiryDescription"`
	MoveStart                   string    `json:"moveStart"`
	PositionLimitWeight         float64   `json:"positionLimitWeight"`
	Group                       string    `json:"group"`
}

// WSMarkets stores websocket markets data
type WSMarkets struct {
	Channel     string        `json:"channel"`
	MessageType string        `json:"type"`
	Data        WsMarketsData `json:"data"`
	Action      string        `json:"action"`
}

// RequestQuoteData stores data on the requested quote
type RequestQuoteData struct {
	QuoteID int64 `json:"quoteId"`
}

// QuoteStatusData stores data of quotes' status
type QuoteStatusData struct {
	BaseCoin  string  `json:"baseCoin"`
	Cost      float64 `json:"cost"`
	Expired   bool    `json:"expired"`
	Filled    bool    `json:"filled"`
	FromCoin  string  `json:"fromCoin"`
	ID        int64   `json:"id"`
	Price     float64 `json:"price"`
	Proceeds  float64 `json:"proceeds"`
	QuoteCoin string  `json:"quoteCoin"`
	Side      string  `json:"side"`
	ToCoin    string  `json:"toCoin"`
}

// AcceptQuote stores data of accepted quote
type AcceptQuote struct {
	Success bool `json:"success"`
}

// Subaccount stores subaccount data
type Subaccount struct {
	Nickname    string `json:"nickname"`
	Special     bool   `json:"special"`
	Deletable   bool   `json:"deletable"`
	Editable    bool   `json:"editable"`
	Competition bool   `json:"competition"`
}

// SubaccountTransferStatus stores the subaccount transfer details
type SubaccountTransferStatus struct {
	ID     int64     `json:"id"`
	Coin   string    `json:"coin"`
	Size   float64   `json:"size"`
	Time   time.Time `json:"time"`
	Notes  string    `json:"notes"`
	Status string    `json:"status"`
}

// SubaccountBalance stores the user's subaccount balance
type SubaccountBalance struct {
	Coin                   string  `json:"coin"`
	Free                   float64 `json:"free"`
	Total                  float64 `json:"total"`
	SpotBorrow             float64 `json:"spotBorrow"`
	AvailableWithoutBorrow float64 `json:"availableWithoutBorrow"`
}

// Stake stores an individual coin stake
type Stake struct {
	Coin      string    `json:"coin"`
	CreatedAt time.Time `json:"createdAt"`
	ID        int64     `json:"id"`
	Size      float64   `json:"size"`
}

// UnstakeRequest stores data for an unstake request
type UnstakeRequest struct {
	Stake
	Status       string    `json:"status"`
	UnlockAt     time.Time `json:"unlockAt"`
	FractionToGo float64   `json:"fractionToGo"`
	Fee          float64   `json:"fee"`
}

// StakeBalance stores an individual coin stake balance
type StakeBalance struct {
	Coin               string  `json:"coin"`
	LifetimeRewards    float64 `json:"lifetimeRewards"`
	ScheduledToUnstake float64 `json:"scheduledToUnstake"`
	Staked             float64 `json:"staked"`
}

// StakeReward stores an individual staking reward
type StakeReward struct {
	Coin   string    `json:"coin"`
	ID     int64     `json:"id"`
	Size   float64   `json:"size"`
	Notes  string    `json:"notes"`
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// CollateralWeightHolder stores collateral weights over the lifecycle of the application
type CollateralWeightHolder map[*currency.Item]CollateralWeight

// CollateralWeight holds collateral information provided by FTX
// it is used to scale collateral when the currency is not in USD
type CollateralWeight struct {
	Initial                     float64
	Total                       float64
	InitialMarginFractionFactor float64
}

// CollateralResponse returned from the collateral endpoint
type CollateralResponse struct {
	PositiveBalances                   []CollateralBalance  `json:"positiveBalances"`
	NegativeBalances                   []CollateralBalance  `json:"negativeBalances"`
	Positions                          []CollateralPosition `json:"positions"`
	PositiveSpotBalanceTotal           decimal.Decimal      `json:"positiveSpotBalanceTotal"`
	CollateralFromPositiveSpotBalances decimal.Decimal      `json:"collateralFromPositiveSpotBalances"`
	UsedBySpotMargin                   decimal.Decimal      `json:"usedBySpotMargin"`
	UsedByFutures                      decimal.Decimal      `json:"usedByFutures"`
	CollateralAvailable                decimal.Decimal      `json:"collateralAvailable"`
}

// CollateralBalance holds collateral information for a coin's balance
type CollateralBalance struct {
	Coin                        currency.Code   `json:"coin"`
	PositionSize                decimal.Decimal `json:"positionSize"`
	OpenOrderSize               decimal.Decimal `json:"openOrderSize"`
	Total                       decimal.Decimal `json:"total"`
	AvailableIgnoringCollateral decimal.Decimal `json:"availableIgnoringCollateral"`
	ApproximateFairMarketValue  decimal.Decimal `json:"approxFair"`
	CollateralContribution      decimal.Decimal `json:"collateralContribution"`
	CollateralUsed              decimal.Decimal `json:"collateralUsed"`
	CollateralWeight            decimal.Decimal `json:"collateralWeight"`
}

// CollateralPosition holds collateral information for a market position
type CollateralPosition struct {
	Future         currency.Pair   `json:"future"`
	Size           decimal.Decimal `json:"size"`
	OpenOrderSize  decimal.Decimal `json:"openOrderSize"`
	PositionSize   decimal.Decimal `json:"positionSize"`
	MarkPrice      decimal.Decimal `json:"markPrice"`
	RequiredMargin decimal.Decimal `json:"requiredMargin"`
	CollateralUsed decimal.Decimal `json:"totalCollateralUsed"`
}
