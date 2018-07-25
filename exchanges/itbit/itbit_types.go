package itbit

import "github.com/kempeng/gocryptotrader/decimal"

// GeneralReturn is a generalized return type to capture any errors
type GeneralReturn struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	RequestID   string `json:"requestId"`
}

// Ticker holds returned ticker information
type Ticker struct {
	Pair          string          `json:"pair"`
	Bid           decimal.Decimal `json:"bid,string"`
	BidAmt        decimal.Decimal `json:"bidAmt,string"`
	Ask           decimal.Decimal `json:"ask,string"`
	AskAmt        decimal.Decimal `json:"askAmt,string"`
	LastPrice     decimal.Decimal `json:"lastPrice,string"`
	LastAmt       decimal.Decimal `json:"lastAmt,string"`
	Volume24h     decimal.Decimal `json:"volume24h,string"`
	VolumeToday   decimal.Decimal `json:"volumeToday,string"`
	High24h       decimal.Decimal `json:"high24h,string"`
	Low24h        decimal.Decimal `json:"low24h,string"`
	HighToday     decimal.Decimal `json:"highToday,string"`
	LowToday      decimal.Decimal `json:"lowToday,string"`
	OpenToday     decimal.Decimal `json:"openToday,string"`
	VwapToday     decimal.Decimal `json:"vwapToday,string"`
	Vwap24h       decimal.Decimal `json:"vwap24h,string"`
	ServertimeUTC string          `json:"serverTimeUTC"`
}

// OrderbookResponse contains multi-arrayed strings of bid and ask side
// information
type OrderbookResponse struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

// Trades holds recent trades with associated information
type Trades struct {
	RecentTrades []struct {
		Timestamp   string          `json:"timestamp"`
		MatchNumber int64           `json:"matchNumber"`
		Price       decimal.Decimal `json:"price,string"`
		Amount      decimal.Decimal `json:"amount,string"`
	} `json:"recentTrades"`
}

// Wallet contains specific wallet information
type Wallet struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	Balances    []Balance `json:"balances"`
	Description string    `json:"description"`
}

// Balance is a sub type holding balance information
type Balance struct {
	Currency         string          `json:"currency"`
	AvailableBalance decimal.Decimal `json:"availableBalance,string"`
	TotalBalance     decimal.Decimal `json:"totalBalance,string"`
	Description      string          `json:"description"`
}

// Records embodies records of trade history information
type Records struct {
	TotalNumberOfRecords int            `json:"totalNumberOfRecords,string"`
	CurrentPageNumber    int            `json:"currentPageNumber,string"`
	LatestExecutedID     int64          `json:"latestExecutionId,string"`
	RecordsPerPage       int            `json:"recordsPerPage,string"`
	TradingHistory       []TradeHistory `json:"tradingHistory"`
	Description          string         `json:"description"`
}

// TradeHistory stores historic trade values
type TradeHistory struct {
	OrderID            string          `json:"orderId"`
	Timestamp          string          `json:"timestamp"`
	Instrument         string          `json:"instrument"`
	Direction          string          `json:"direction"`
	CurrencyOne        string          `json:"currency1"`
	CurrencyOneAmount  decimal.Decimal `json:"currency1Amount,string"`
	CurrencyTwo        string          `json:"currency2"`
	CurrencyTwoAmount  decimal.Decimal `json:"currency2Amount"`
	Rate               decimal.Decimal `json:"rate,string"`
	CommissionPaid     decimal.Decimal `json:"commissionPaid,string"`
	CommissionCurrency string          `json:"commissionCurrency"`
	RebatesApplied     decimal.Decimal `json:"rebatesApplied,string"`
	RebateCurrency     string          `json:"rebateCurrency"`
}

// FundingRecords embodies records of fund history information
type FundingRecords struct {
	TotalNumberOfRecords int           `json:"totalNumberOfRecords,string"`
	CurrentPageNumber    int           `json:"currentPageNumber,string"`
	LatestExecutedID     int64         `json:"latestExecutionId,string"`
	RecordsPerPage       int           `json:"recordsPerPage,string"`
	FundingHistory       []FundHistory `json:"fundingHistory"`
	Description          string        `json:"description"`
}

// FundHistory stores historic funding transactions
type FundHistory struct {
	BankName                    string          `json:"bankName"`
	WithdrawalID                int64           `json:"withdrawalId"`
	HoldingPeriodCompletionDate string          `json:"holdingPeriodCompletionDate"`
	DestinationAddress          string          `json:"destinationAddress"`
	TxnHash                     string          `json:"txnHash"`
	Time                        string          `json:"time"`
	Currency                    string          `json:"currency"`
	TransactionType             string          `json:"transactionType"`
	Amount                      decimal.Decimal `json:"amount,string"`
	WalletName                  string          `json:"walletName"`
	Status                      string          `json:"status"`
}

// Order holds order information
type Order struct {
	ID                         string          `json:"id"`
	WalletID                   string          `json:"walletId"`
	Side                       string          `json:"side"`
	Instrument                 string          `json:"instrument"`
	Type                       string          `json:"type"`
	Currency                   string          `json:"currency"`
	Amount                     decimal.Decimal `json:"amount,string"`
	Price                      decimal.Decimal `json:"price,string"`
	AmountFilled               decimal.Decimal `json:"amountFilled,string"`
	VolumeWeightedAveragePrice decimal.Decimal `json:"volumeWeightedAveragePrice,string"`
	CreatedTime                string          `json:"createdTime"`
	Status                     string          `json:"Status"`
	Metadata                   interface{}     `json:"metadata"`
	ClientOrderIdentifier      string          `json:"clientOrderIdentifier"`
	Description                string          `json:"description"`
}

// CryptoCurrencyDeposit holds information about a new wallet
type CryptoCurrencyDeposit struct {
	ID             int         `json:"id"`
	WalletID       string      `json:"walletID"`
	DepositAddress string      `json:"depositAddress"`
	Metadata       interface{} `json:"metadata"`
	Description    string      `json:"description"`
}

// WalletTransfer holds wallet transfer information
type WalletTransfer struct {
	SourceWalletID      string          `json:"sourceWalletId"`
	DestinationWalletID string          `json:"destinationWalletId"`
	Amount              decimal.Decimal `json:"amount,string"`
	CurrencyCode        string          `json:"currencyCode"`
	Description         string          `json:"description"`
}
