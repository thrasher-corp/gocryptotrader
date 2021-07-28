package ftx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// MarginFundingData stores borrowing/lending data for margin trading
type MarginFundingData struct {
	Coin     string  `json:"coin"`
	Estimate float64 `json:"estimate"`
	Previous float64 `json:"previous"`
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
	Coin string    `json:"coin"`
	Cost float64   `json:"cost"`
	Rate float64   `json:"rate"`
	Size float64   `json:"size"`
	Time time.Time `json:"time"`
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
	Ask                 float64     `json:"ask"`
	Bid                 float64     `json:"bid"`
	Change1h            float64     `json:"change1h"`
	Change24h           float64     `json:"change24h"`
	ChangeBod           float64     `json:"changeBod"`
	VolumeUSD24h        float64     `json:"volumeUsd24h"`
	Volume              float64     `json:"volume"`
	Description         string      `json:"description"`
	Enabled             bool        `json:"enabled"`
	Expired             bool        `json:"expired"`
	Expiry              time.Time   `json:"expiry"`
	ExpiryDescription   string      `json:"expiryDescription"`
	Group               string      `json:"group"`
	Index               float64     `json:"index"`
	IMFFactor           float64     `json:"imfFactor"`
	Last                float64     `json:"last"`
	LowerBound          float64     `json:"lowerBound"`
	MarginPrice         float64     `json:"marginPrice"`
	Mark                float64     `json:"mark"`
	MoveStart           interface{} `json:"moveStart"`
	Name                string      `json:"name"`
	Perpetual           bool        `json:"perpetual"`
	PositionLimitWeight float64     `json:"positionLimitWeight"`
	PostOnly            bool        `json:"postOnly"`
	PriceIncrement      float64     `json:"priceIncrement"`
	SizeIncrement       float64     `json:"sizeIncrement"`
	Underlying          string      `json:"underlying"`
	UpperBound          float64     `json:"upperBound"`
	FutureType          string      `json:"type"`
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
	Cost                         float64 `json:"cost"`
	EntryPrice                   float64 `json:"entryPrice"`
	Future                       string  `json:"future"`
	InitialMarginRequirement     float64 `json:"initialMarginRequirement"`
	LongOrderSize                float64 `json:"longOrderSize"`
	MaintenanceMarginRequirement float64 `json:"maintenanceMarginRequirement"`
	NetSize                      float64 `json:"netSize"`
	OpenSize                     float64 `json:"openSize"`
	RealizedPnL                  float64 `json:"realizedPnL"`
	ShortOrderSize               float64 `json:"shortOrderSize"`
	Side                         string  `json:"side"`
	Size                         float64 `json:"size"`
	UnrealizedPnL                float64 `json:"unrealizedPnL"`
	CollateralUsed               float64 `json:"collateralUsed"`
	EstimatedLiquidationPrice    float64 `json:"estimatedLiquidationPrice"`
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
	Bep2Asset        interface{} `json:"bep2Asset"`
	CanConvert       bool        `json:"canConvert"`
	CanDeposit       bool        `json:"canDeposit"`
	CanWithdraw      bool        `json:"canWithdraw"`
	Collateral       bool        `json:"collateral"`
	CollateralWeight float64     `json:"collateralWeight"`
	CreditTo         interface{} `json:"creditTo"`
	ERC20Contract    interface{} `json:"erc20Contract"`
	Fiat             bool        `json:"fiat"`
	HasTag           bool        `json:"hasTag"`
	Hidden           bool        `json:"hidden"`
	IsETF            bool        `json:"isEtf"`
	IsToken          bool        `json:"isToken"`
	Methods          []interface{}
	ID               string `json:"id"`
	Name             string `json:"name"`
}

// WalletBalance stores balances data
type WalletBalance struct {
	Coin                   string  `json:"coin"`
	Free                   float64 `json:"free"`
	Total                  float64 `json:"total"`
	AvailableWithoutBorrow float64 `json:"availableWithoutBorrow"`
	USDValue               float64 `json:"usdValue"`
	SpotBorrow             float64 `json:"spotBorrow"`
}

// AllWalletBalances stores all the user's account balances
type AllWalletBalances map[string][]WalletBalance

// DepositData stores deposit address data
type DepositData struct {
	Address string `json:"address"`
	Tag     string `json:"tag"`
}

// TransactionData stores data about deposit history
type TransactionData struct {
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
}

// OrderData stores open order data
type OrderData struct {
	CreatedAt     time.Time `json:"createdAt"`
	FilledSize    float64   `json:"filledSize"`
	Future        string    `json:"future"`
	ID            int64     `json:"id"`
	Market        string    `json:"market"`
	Price         float64   `json:"price"`
	AvgFillPrice  float64   `json:"avgFillPrice"`
	RemainingSize float64   `json:"remainingSize"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	OrderType     string    `json:"type"`
	ReduceOnly    bool      `json:"reduceOnly"`
	IOC           bool      `json:"ioc"`
	PostOnly      bool      `json:"postOnly"`
	ClientID      string    `json:"clientId"`
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
	Fee           float64   `json:"fee"`
	FeeRate       float64   `json:"feeRate"`
	Future        string    `json:"future"`
	ID            int64     `json:"id"`
	Liquidity     string    `json:"liquidity"`
	Market        string    `json:"market"`
	BaseCurrency  string    `json:"baseCurrency"`
	QuoteCurrency string    `json:"quoteCurrency"`
	OrderID       int64     `json:"orderId"`
	TradeID       int64     `json:"tradeId"`
	Price         float64   `json:"price"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Time          time.Time `json:"time"`
	OrderType     string    `json:"type"`
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
	Key  string `json:"key"`
	Sign string `json:"sign"`
	Time int64  `json:"time"`
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
	ID            int64   `json:"id"`
	ClientID      string  `json:"clientId"`
	Market        string  `json:"market"`
	OrderType     string  `json:"type"`
	Side          string  `json:"side"`
	Size          float64 `json:"size"`
	Price         float64 `json:"price"`
	ReduceOnly    bool    `json:"reduceOnly"`
	IOC           bool    `json:"ioc"`
	PostOnly      bool    `json:"postOnly"`
	Status        string  `json:"status"`
	FilledSize    float64 `json:"filedSize"`
	RemainingSize float64 `json:"remainingSize"`
	AvgFillPrice  float64 `json:"avgFillPrice"`
}

// WsFills stores websocket fills' data
type WsFills struct {
	Fee       float64   `json:"fee"`
	FeeRate   float64   `json:"feeRate"`
	Future    string    `json:"future"`
	ID        int64     `json:"id"`
	Liquidity string    `json:"liquidity"`
	Market    string    `json:"market"`
	OrderID   int64     `json:"orderId"`
	TradeID   int64     `json:"tradeId"`
	Price     float64   `json:"price"`
	Side      string    `json:"side"`
	Size      float64   `json:"size"`
	Time      time.Time `json:"time"`
	OrderType string    `json:"orderType"`
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
	FillsData   WsFills `json:"fills"`
}

// TimeInterval represents interval enum.
type TimeInterval string

// Vars related to time intervals
var (
	TimeIntervalFifteenSeconds = TimeInterval("15")
	TimeIntervalMinute         = TimeInterval("60")
	TimeIntervalFiveMinutes    = TimeInterval("300")
	TimeIntervalFifteenMinutes = TimeInterval("900")
	TimeIntervalHour           = TimeInterval("3600")
	TimeIntervalFourHours      = TimeInterval("14400")
	TimeIntervalDay            = TimeInterval("86400")
)

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
	Name           string              `json:"name,omitempty"`
	Enabled        bool                `json:"enabled,omitempty"`
	PriceIncrement float64             `json:"priceIncrement,omitempty"`
	SizeIncrement  float64             `json:"sizeIncrement,omitempty"`
	MarketType     string              `json:"marketType,omitempty"`
	BaseCurrency   string              `json:"baseCurrency,omitempty"`
	QuoteCurrency  string              `json:"quoteCurrency,omitempty"`
	Underlying     string              `json:"underlying,omitempty"`
	Restricted     bool                `json:"restricted,omitempty"`
	Future         WsMarketsFutureData `json:"future,omitempty"`
}

// WsMarketsFutureData stores websocket markets' future data
type WsMarketsFutureData struct {
	Name                  string    `json:"name,omitempty"`
	Underlying            string    `json:"underlying,omitempty"`
	Description           string    `json:"description,omitempty"`
	MarketType            string    `json:"type,omitempty"`
	Expiry                time.Time `json:"expiry,omitempty"`
	Perpetual             bool      `json:"perpetual,omitempty"`
	Expired               bool      `json:"expired,omitempty"`
	Enabled               bool      `json:"enabled,omitempty"`
	PostOnly              bool      `json:"postOnly,omitempty"`
	IMFFactor             float64   `json:"imfFactor,omitempty"`
	UnderlyingDescription string    `json:"underlyingDescription,omitempty"`
	ExpiryDescription     string    `json:"expiryDescription,omitempty"`
	MoveStart             string    `json:"moveStart,omitempty"`
	PositionLimitWeight   float64   `json:"positionLimitWeight,omitempty"`
	Group                 string    `json:"group,omitempty"`
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
