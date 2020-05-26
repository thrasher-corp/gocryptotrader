package ftx

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// MarketData stores market data
type MarketData struct {
	Name           string  `json:"name"`
	BaseCurrency   string  `json:"baseCurrency"`
	QuoteCurrency  string  `json:"quoteCurrency"`
	MarketType     string  `json:"type"`
	Underlying     string  `json:"underlying"`
	Enabled        bool    `json:"enabled"`
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Last           float64 `json:"last"`
	PriceIncrement float64 `json:"priceIncrement"`
	SizeIncrement  float64 `json:"sizeIncrement"`
}

// Markets stores all markets data
type Markets struct {
	Success bool         `json:"success"`
	Result  []MarketData `json:"result"`
}

// Market stores data for a given market
type Market struct {
	Success bool       `json:"success"`
	Result  MarketData `json:"result"`
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

// TempOrderbook stores order book
type TempOrderbook struct {
	Success bool       `json:"success"`
	Result  TempOBData `json:"result"`
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

// Trades stores data for multiple trades
type Trades struct {
	Success bool        `json:"success"`
	Result  []TradeData `json:"result"`
}

// OHLCVData stores historical OHLCV data
type OHLCVData struct {
	Close     float64   `json:"close"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Open      float64   `json:"open"`
	StartTime time.Time `json:"startTime"`
	Volume    float64   `json:"volume"`
}

// HistoricalData stores historical OHLCVData
type HistoricalData struct {
	Success bool        `json:"success"`
	Result  []OHLCVData `json:"result"`
}

// FuturesData stores data for futures
type FuturesData struct {
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	Change1h       float64 `json:"change1h"`
	Change24h      float64 `json:"change24h"`
	ChangeBod      float64 `json:"changeBod"`
	VolumeUSD24h   float64 `json:"volumeUsd24h"`
	Volume         float64 `json:"volume"`
	Description    string  `json:"description"`
	Enabled        bool    `json:"enabled"`
	Expired        bool    `json:"expired"`
	Expiry         string  `json:"expiry"`
	Index          float64 `json:"index"`
	Last           float64 `json:"last"`
	LowerBound     float64 `json:"lowerBound"`
	Mark           float64 `json:"mark"`
	Name           string  `json:"name"`
	Perpetual      bool    `json:"perpetual"`
	PostOnly       bool    `json:"postOnly"`
	PriceIncrement float64 `json:"priceIncrement"`
	SizeIncrement  float64 `json:"sizeIncrement"`
	Underlying     string  `json:"underlying"`
	UpperBound     float64 `json:"upperBound"`
	FutureType     string  `json:"type"`
}

// Futures stores futures data
type Futures struct {
	Success bool          `json:"success"`
	Result  []FuturesData `json:"result"`
}

// Future stores data for a singular future
type Future struct {
	Success bool        `json:"success"`
	Result  FuturesData `json:"result"`
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

// FutureStats stores future stats
type FutureStats struct {
	Success bool            `json:"success"`
	Result  FutureStatsData `json:"result"`
}

// FundingRatesData stores data on funding rates
type FundingRatesData struct {
	Future string    `json:"future"`
	Rate   float64   `json:"rate"`
	Time   time.Time `json:"time"`
}

// FundingRates stores data on funding rates
type FundingRates struct {
	Success bool               `json:"success"`
	Result  []FundingRatesData `json:"result"`
}

// PositionData stores data of an open position
type PositionData struct {
	Cost                         float64 `json:"cost"`
	EntryPrice                   float64 `json:"entryPrice"`
	EstimatedLiquidationPrice    float64 `json:"estimatedLiquidationPrice"`
	Future                       string  `json:"future"`
	InitialMarginRequirement     float64 `json:"initialMarginRequirement"`
	LongOrderSize                float64 `json:"longOrderSize"`
	MaintenanceMarginRequirement float64 `json:"maintenanceMarginRequirement"`
	NetSize                      float64 `json:"netSize"`
	OpenSize                     float64 `json:"openSize"`
	RealisedPnL                  float64 `json:"realisedPnL"`
	ShortOrderSide               float64 `json:"shortOrderSide"`
	Side                         string  `json:"side"`
	Size                         string  `json:"size"`
	UnrealisedPnL                float64 `json:"unrealisedPnL"`
}

// AccountInfoData stores account data
type AccountInfoData struct {
	BackstopProvider             bool           `json:"backstopProvider"`
	Collateral                   float64        `json:"collateral"`
	FreeCollateral               float64        `json:"freeCollateral"`
	InitialMarginRequirement     float64        `json:"initialMarginRequirement"`
	Leverage                     float64        `json:"float64"`
	Liquidating                  bool           `json:"liquidating"`
	MaintenanceMarginRequirement float64        `json:"maintenanceMarginRequirement"`
	MakerFee                     float64        `json:"makerFee"`
	MarginFraction               float64        `json:"marginFraction"`
	OpenMarginFraction           float64        `json:"openMarginFraction"`
	TakerFee                     float64        `json:"takerFee"`
	TotalAccountValue            float64        `json:"totalAccountValue"`
	TotalPositionSize            float64        `json:"totalPositionSize"`
	Username                     string         `json:"username"`
	Positions                    []PositionData `json:"positions"`
}

// AccountData stores account data
type AccountData struct {
	Success bool            `json:"success"`
	Result  AccountInfoData `json:"result"`
}

// Positions stores data about positions
type Positions struct {
	Success bool           `json:"success"`
	Result  []PositionData `json:"result"`
}

// WalletCoinsData stores data about wallet coins
type WalletCoinsData struct {
	CanDeposit  bool   `json:"canDeposit"`
	CanWithdraw bool   `json:"canWithdraw"`
	HasTag      bool   `json:"hasTag"`
	ID          string `json:"id"`
	Name        string `json:"name"`
}

// WalletCoins stores data about wallet coins
type WalletCoins struct {
	Success bool              `json:"success"`
	Result  []WalletCoinsData `json:"result"`
}

// BalancesData stores balances data
type BalancesData struct {
	Coin  string  `json:"coin"`
	Free  float64 `json:"free"`
	Total float64 `json:"total"`
}

// WalletBalances stores data about wallet's balances
type WalletBalances struct {
	Success bool           `json:"success"`
	Result  []BalancesData `json:"result"`
}

// AllWalletAccountData stores account data on all WalletCoins
type AllWalletAccountData struct {
	Main         []BalancesData `json:"main"`
	BattleRoyale []BalancesData `json:"Battle Royale"`
}

// AllWalletBalances stores data about all account balances including sub acconuts
type AllWalletBalances struct {
	Success bool                 `json:"success"`
	Result  AllWalletAccountData `json:"result"`
}

// DepositData stores deposit address data
type DepositData struct {
	Address string `json:"address"`
	Tag     string `json:"tag"`
}

// DepositAddress stores deposit address data of a given coin
type DepositAddress struct {
	Success bool        `json:"success"`
	Result  DepositData `json:"result"`
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

// DepositHistory stores deposit history data
type DepositHistory struct {
	Success bool              `json:"success"`
	Result  []TransactionData `json:"result"`
}

// WithdrawalHistory stores withdrawal data
type WithdrawalHistory struct {
	Success bool              `json:"success"`
	Result  []TransactionData `json:"result"`
}

// WithdrawData stores withdraw request data
type WithdrawData struct {
	Success bool            `json:"success"`
	Result  TransactionData `json:"result"`
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

// OpenOrders stores data of open orders
type OpenOrders struct {
	Success bool        `json:"success"`
	Result  []OrderData `json:"result"`
}

// OrderHistory stores order history data
type OrderHistory struct {
	Success bool        `json:"success"`
	Result  []OrderData `json:"result"`
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

// OpenTriggerOrders stores trigger orders' data that are open
type OpenTriggerOrders struct {
	Success bool               `json:"success"`
	Result  []TriggerOrderData `json:"result"`
}

// TriggerData stores trigger orders' trigger data
type TriggerData struct {
	Error      string    `json:"error"`
	FilledSize float64   `json:"filledSize"`
	OrderSize  float64   `json:"orderSize"`
	OrderID    int64     `json:"orderId"`
	Time       time.Time `json:"time"`
}

// Triggers stores trigger orders' data
type Triggers struct {
	Success bool          `json:"success"`
	Result  []TriggerData `json:"result"`
}

// TriggerOrderHistory stores trigger orders from past
type TriggerOrderHistory struct {
	Success bool               `json:"success"`
	Result  []TriggerOrderData `json:"result"`
}

// PlaceOrder stores data of placed orders
type PlaceOrder struct {
	Success bool      `json:"success"`
	Result  OrderData `json:"result"`
}

// PlaceTriggerOrder stores data of a placed trigger order
type PlaceTriggerOrder struct {
	Success bool             `json:"success"`
	Result  TriggerOrderData `json:"result"`
}

// ModifyOrder stores modified order data
type ModifyOrder struct {
	Success bool      `json:"success"`
	Result  OrderData `json:"result"`
}

// ModifyTriggerOrder stores modified trigger order data
type ModifyTriggerOrder struct {
	Success bool             `json:"success"`
	Result  TriggerOrderData `json:"result"`
}

// OrderStatus stores order status data
type OrderStatus struct {
	Success bool      `json:"success"`
	Result  OrderData `json:"result"`
}

// CancelOrderResponse stores cancel order response
type CancelOrderResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
}

// FillsData stores fills' data
type FillsData struct {
	Fee           float64   `json:"fee"`
	FeeRate       float64   `json:"feeRate"`
	Future        string    `json:"future"`
	ID            string    `json:"id"`
	Liquidity     string    `json:"liquidity"`
	Market        string    `json:"market"`
	BaseCurrency  string    `json:"baseCurrency"`
	QuoteCurrency string    `json:"quoteCurrency"`
	OrderID       string    `json:"orderID"`
	TradeID       string    `json:"tradeID"`
	Price         float64   `json:"price"`
	Side          string    `json:"side"`
	Size          string    `json:"size"`
	Time          time.Time `json:"time"`
	OrderType     string    `json:"type"`
}

// Fills stores fills' data
type Fills struct {
	Success bool        `json:"success"`
	Result  []FillsData `json:"result"`
}

// FundingPaymentsData stores funding payments' data
type FundingPaymentsData struct {
	Future  string    `json:"future"`
	ID      string    `json:"id"`
	Payment float64   `json:"payment"`
	Time    time.Time `json:"time"`
	Rate    float64   `json:"rate"`
}

// FundingPayments stores funding payments data
type FundingPayments struct {
	Success bool                  `json:"success"`
	Result  []FundingPaymentsData `json:"result"`
}

// LeveragedTokensData stores data of leveraged tokens
type LeveragedTokensData struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Underlying       string  `json:"underlying"`
	Leverage         float64 `json:"leverage"`
	Outstanding      float64 `json:"outstanding"`
	PricePerShare    float64 `json:"pricePerShare"`
	PositionPerShare float64 `json:"positionPerShare"`
	UnderlyingMark   float64 `json:"underlyingMark"`
	ContactAddress   string  `json:"contactAddress"`
	Change1h         float64 `json:"change1h"`
	Change24h        float64 `json:"change24h"`
}

// LeveragedTokens stores data of leveraged tokens
type LeveragedTokens struct {
	Success bool                  `json:"success"`
	Result  []LeveragedTokensData `json:"result"`
}

// TokenInfo stores token's info
type TokenInfo struct {
	Success bool                  `json:"success"`
	Result  []LeveragedTokensData `json:"result"`
}

// LTBalanceData stores balances of leveraged tokens
type LTBalanceData struct {
	Token   string  `json:"token"`
	Balance float64 `json:"balance"`
}

// LTBalances stores balances of leveraged tokens
type LTBalances struct {
	Success bool            `json:"success"`
	Result  []LTBalanceData `json:"result"`
}

// LTCreationData stores token creation requests' data
type LTCreationData struct {
	ID            string  `json:"id"`
	Token         string  `json:"token"`
	RequestedSize float64 `json:"requestedSize"`
	Pending       bool    `json:"pending"`
	CreatedSize   float64 `json:"createdize"`
	Price         float64 `json:"price"`
	Cost          float64 `json:"cost"`
	Fee           float64 `json:"fee"`
	RequestedAt   string  `json:"requestedAt"`
	FulfilledAt   string  `json:"fulfilledAt"`
}

// LTCreationList stores token creations requests' data
type LTCreationList struct {
	Success bool             `json:"success"`
	Result  []LTCreationData `json:"result"`
}

// RequestTokenCreationData stores data of the token creation requested
type RequestTokenCreationData struct {
	ID            string  `json:"id"`
	Token         string  `json:"token"`
	RequestedSize float64 `json:"requestedSize"`
	Cost          float64 `json:"cost"`
	Pending       bool    `json:"pending"`
	RequestedAt   string  `json:"requestedAt"`
}

// RequestTokenCreation stores data of the token creation requested
type RequestTokenCreation struct {
	Success bool                     `json:"success"`
	Result  RequestTokenCreationData `json:"result"`
}

// LTRedemptionData stores data of the token redemption request
type LTRedemptionData struct {
	ID          int64   `json:"id"`
	Token       string  `json:"token"`
	Size        float64 `json:"size"`
	Pending     bool    `json:"pending"`
	Price       float64 `json:"price"`
	Proceeds    float64 `json:"proceeds"`
	Fee         float64 `json:"fee"`
	RequestedAt string  `json:"requestedAt"`
	FulfilledAt string  `json:"fulfilledAt"`
}

// LTRedemptionList stores data of token redemption list
type LTRedemptionList struct {
	Success bool               `json:"success"`
	Result  []LTRedemptionData `json:"result"`
}

// LTRedemptionRequestData stores redemption request data for a leveraged token
type LTRedemptionRequestData struct {
	ID                string  `json:"id"`
	Token             string  `json:"token"`
	Size              float64 `json:"size"`
	ProjectedProceeds float64 `json:"projectedProceeds"`
	Pending           bool    `json:"pending"`
	RequestedAt       string  `json:"requestedAt"`
}

// LTRedemptionRequest stores redemption request data of a leveraged token
type LTRedemptionRequest struct {
	Success bool                    `json:"success"`
	Result  LTRedemptionRequestData `json:"result"`
}

// OptionData stores options' data
type OptionData struct {
	Underlying string  `json:"underlying"`
	OptionType string  `json:"type"`
	Strike     float64 `json:"strike"`
	Expiry     string  `json:"expiry"`
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

// QuoteRequests stores data of quote requests
type QuoteRequests struct {
	Success bool               `json:"success"`
	Result  []QuoteRequestData `json:"result"`
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

// PersonalQuotes stores quote data of your quotes
type PersonalQuotes struct {
	Success bool                 `json:"success"`
	Result  []PersonalQuotesData `json:"result"`
}

// CreateQuoteRequestData stores quote data of the request sent
type CreateQuoteRequestData struct {
	ID            int64     `json:"id"`
	Expiry        string    `json:"expiry"`
	Strike        float64   `json:"strike"`
	OptionType    string    `json:"type"`
	Underlying    string    `json:"underlying"`
	RequestExpiry string    `json:"requestExpiry"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	Status        string    `json:"status"`
	Time          time.Time `json:"time"`
}

// CreateQuote stores create quote request data
type CreateQuote struct {
	Success bool                   `json:"success"`
	Result  CreateQuoteRequestData `json:"result"`
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

// CancelQuote stores cancel quote request data
type CancelQuote struct {
	Success bool                   `json:"success"`
	Result  CancelQuoteRequestData `json:"result"`
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

// QuoteForQuoteResponse stores quote data for another quote
type QuoteForQuoteResponse struct {
	Success bool                `json:"success"`
	Result  []QuoteForQuoteData `json:"result"`
}

// AccountOptionsInfoData stores account's options' info data
type AccountOptionsInfoData struct {
	USDBalance       float64 `json:"usdBalance"`
	LiquidationPrice float64 `json:"liquidationPrice"`
	Liquidating      bool    `json:"liquidating"`
}

// AccountOptionsInfo stores account's options' info data
type AccountOptionsInfo struct {
	Success bool                   `json:"success"`
	Result  AccountOptionsInfoData `json:"result"`
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

// OptionsPositions stores account's options' info data
type OptionsPositions struct {
	Success bool                   `json:"success"`
	Result  []OptionsPositionsData `json:"result"`
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

// OptionsFills gets options' fills data
type OptionsFills struct {
	Success bool              `json:"success"`
	Result  []OptionFillsData `json:"result"`
}

// PublicOptionsTrades stores options' trades from public
type PublicOptionsTrades struct {
	Success bool                `json:"success"`
	Result  []OptionsTradesData `json:"result"`
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
	Checksum int          `json:"checksum"`
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
	OrderID   int64     `json:"int64"`
	TradeID   int64     `json:"tradeID"`
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

var errInvalidInterval = errors.New("invalid interval")

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

// RequestQuote stores data on the requested quote
type RequestQuote struct {
	Success bool             `json:"success"`
	Result  RequestQuoteData `json:"result"`
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

// QuoteStatus stores data of quotes' status
type QuoteStatus struct {
	Success bool              `json:"success"`
	Result  []QuoteStatusData `json:"result"`
}

// AcceptQuote stores data of accepted quote
type AcceptQuote struct {
	Success bool `json:"success"`
}
