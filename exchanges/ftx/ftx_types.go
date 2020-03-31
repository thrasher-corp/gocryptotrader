package ftx

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
	ID          int64   `json:"id"`
	Liquidation bool    `json:"liquidation"`
	Price       float64 `json:"price"`
	Side        string  `json:"side"`
	Size        float64 `json:"size"`
	Time        string  `json:"time"`
}

// Trades stores data for multiple trades
type Trades struct {
	Success bool        `json:"success"`
	Result  []TradeData `json:"result"`
}

// OHLCVData stores historical OHLCV data
type OHLCVData struct {
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	StartTime string  `json:"startTime"`
	Volume    float64 `json:"volume"`
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
	Volume                   float64 `json:"volume"`
	NextFundingRate          float64 `json:"nextFundingRate"`
	NextFundingTime          string  `json:"nextFundingTime"`
	ExpirationPrice          float64 `json:"expirationPrice"`
	PredictedExpirationPrice float64 `json:"predictedExpirationPrice"`
	OpenInterest             float64 `json:"openInterest"`
	StrikePrice              float64 `json:"strikePrice"`
}

// FutureStats stores future stats
type FutureStats struct {
	Success bool            `json:"success"`
	Result  FutureStatsData `json:"result"`
}

// FundingRatesData stores data on funding rates
type FundingRatesData struct {
	Future string  `json:"future"`
	Rate   float64 `json:"rate"`
	Time   string  `json:"time"`
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
	Coin          string  `json:"coin"`
	Confirmations int64   `json:"conformations"`
	ConfirmedTime string  `json:"confirmedTime"`
	Fee           float64 `json:"fee"`
	ID            int64   `json:"id"`
	SentTime      string  `json:"sentTime"`
	Size          float64 `json:"size"`
	Status        string  `json:"status"`
	Time          string  `json:"time"`
	TxID          string  `json:"txid"`
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
	CreatedAt     string  `json:"createdAt"`
	FilledSize    float64 `json:"filledSize"`
	Future        string  `json:"future"`
	ID            int64   `json:"id"`
	Market        string  `json:"market"`
	Price         float64 `json:"price"`
	AvgFillPrice  float64 `json:"avgFillPrice"`
	RemainingSize float64 `json:"remainingSize"`
	Side          string  `json:"side"`
	Size          float64 `json:"size"`
	Status        string  `json:"status"`
	OrderType     string  `json:"type"`
	ReduceOnly    bool    `json:"reduceOnly"`
	IOC           bool    `json:"ioc"`
	PostOnly      bool    `json:"postOnly"`
	ClientID      string  `json:"clientId"`
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
	CreatedAt        string  `json:"createdAt"`
	Error            string  `json:"error"`
	Future           string  `json:"future"`
	ID               int64   `json:"id"`
	Market           string  `json:"market"`
	OrderID          int64   `json:"orderId"`
	OrderPrice       float64 `json:"orderPrice"`
	ReduceOnly       bool    `json:"reduceOnly"`
	Side             string  `json:"side"`
	Size             float64 `json:"size"`
	Status           string  `json:"status"`
	TrailStart       float64 `json:"trailStart"`
	TrailValue       float64 `json:"trailvalue"`
	TriggerPrice     float64 `json:"triggerPrice"`
	TriggeredAt      string  `json:"triggeredAt"`
	OrderType        string  `json:"type"`
	MarketOrLimit    string  `json:"orderType"`
	FilledSize       float64 `json:"filledSize"`
	AvgFillPrice     float64 `json:"avgFillPrice"`
	RetryUntilFilled bool    `json:"retryUntilFilled"`
}

// OpenTriggerOrders stores trigger orders' data that are open
type OpenTriggerOrders struct {
	Success bool               `json:"success"`
	Result  []TriggerOrderData `json:"result"`
}

// TriggerData stores trigger orders' trigger data
type TriggerData struct {
	Error      string  `json:"error"`
	FilledSize float64 `json:"filledSize"`
	OrderSize  float64 `json:"orderSize"`
	OrderID    int64   `json:"orderId"`
	Time       string  `json:"time"`
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
