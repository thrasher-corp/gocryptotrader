package mexc

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// ExchangeConfig represents rules and symbols of an exchange
type ExchangeConfig struct {
	Timezone        string         `json:"timezone"`
	ServerTime      types.Time     `json:"serverTime"`
	RateLimits      []any          `json:"rateLimits"`
	ExchangeFilters []any          `json:"exchangeFilters"`
	Symbols         []SymbolDetail `json:"symbols"`
}

// SymbolDetail represents a symbol detail.
type SymbolDetail struct {
	Symbol                     string       `json:"symbol"`
	Status                     string       `json:"status"`
	BaseAsset                  string       `json:"baseAsset"`
	BaseAssetPrecision         int          `json:"baseAssetPrecision"`
	QuoteAsset                 string       `json:"quoteAsset"`
	QuotePrecision             int          `json:"quotePrecision"`
	QuoteAssetPrecision        int          `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int          `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int          `json:"quoteCommissionPrecision"`
	OrderTypes                 []string     `json:"orderTypes"`
	IsSpotTradingAllowed       bool         `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool         `json:"isMarginTradingAllowed"`
	QuoteAmountPrecision       string       `json:"quoteAmountPrecision"`
	BaseSizePrecision          string       `json:"baseSizePrecision"`
	Permissions                []string     `json:"permissions"`
	Filters                    []any        `json:"filters"`
	MaxQuoteAmount             string       `json:"maxQuoteAmount"`
	MakerCommission            types.Number `json:"makerCommission"`
	TakerCommission            types.Number `json:"takerCommission"`
	QuoteAmountPrecisionMarket types.Number `json:"quoteAmountPrecisionMarket"`
	MaxQuoteAmountMarket       types.Number `json:"maxQuoteAmountMarket"`
	FullName                   string       `json:"fullName"`
	TradeSideType              int64        `json:"tradeSideType"`
}

// Orderbook represents a symbol orderbook detail
type Orderbook struct {
	LastUpdateID int64             `json:"lastUpdateId"`
	Bids         [][2]types.Number `json:"bids"`
	Asks         [][2]types.Number `json:"asks"`
	Timestamp    types.Time        `json:"timestamp"`
}

// TradeDetail represents a trade detail
type TradeDetail struct {
	ID           any          `json:"id"`
	Price        types.Number `json:"price"`
	Quantity     types.Number `json:"qty"`
	QuoteQty     types.Number `json:"quoteQty"`
	Time         types.Time   `json:"time"`
	IsBuyerMaker bool         `json:"isBuyerMaker"`
	IsBestMatch  bool         `json:"isBestMatch"`
	TradeType    string       `json:"tradeType"`
}

// AggregatedTradeDetail represents an aggregated trade detail
type AggregatedTradeDetail struct {
	AggregatedTradeID any          `json:"a"`
	FirstTradeID      any          `json:"f"`
	LastTradeID       any          `json:"l"`
	Price             types.Number `json:"p"`
	Quantity          types.Number `json:"q"`
	Timestamp         types.Time   `json:"T"`
	MakerBuyer        bool         `json:"m"` // Was the buyer the maker?
	MathBestPrice     bool         `json:"M"` // Was the trade the best price match?
}

// CandlestickData represents a candlestick data for a symbol
type CandlestickData struct {
	OpenTime         types.Time
	OpenPrice        types.Number
	HighPrice        types.Number
	LowPrice         types.Number
	ClosePrice       types.Number
	Volume           types.Number
	CloseTime        types.Time
	QuoteAssetVolume types.Number
}

// UnmarshalJSON deserializes byte data into a CandlestickData instance
func (c *CandlestickData) UnmarshalJSON(data []byte) error {
	target := [8]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume}
	return json.Unmarshal(data, &target)
}

// SymbolAveragePrice represents a symbol average price detail
type SymbolAveragePrice struct {
	Mins  int64        `json:"mins"`
	Price types.Number `json:"price"`
}

// TickerData represents a ticker data for a symbol
type TickerData struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	PrevClosePrice     types.Number `json:"prevClosePrice"`
	LastPrice          types.Number `json:"lastPrice"`
	BidPrice           types.Number `json:"bidPrice"`
	BidQty             types.Number `json:"bidQty"`
	AskPrice           types.Number `json:"askPrice"`
	AskQty             types.Number `json:"askQty"`
	OpenPrice          types.Number `json:"openPrice"`
	HighPrice          types.Number `json:"highPrice"`
	LowPrice           types.Number `json:"lowPrice"`
	Volume             types.Number `json:"volume"`
	QuoteVolume        types.Number `json:"quoteVolume"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	Count              any          `json:"count"`
}

// TickerList represents list of ticker data
type TickerList []TickerData

// UnmarshalJSON deserializes byte data into TickerList
func (t *TickerList) UnmarshalJSON(data []byte) error {
	tickers := []TickerData{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *TickerData
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []TickerData{*val}
	}
	*t = tickers
	return nil
}

// SymbolPriceTicker represents a symbol price ticker info
type SymbolPriceTicker struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// SymbolPriceTickers represent list of symbol price tickers
type SymbolPriceTickers []SymbolPriceTicker

func (t *SymbolPriceTickers) UnmarshalJSON(data []byte) error {
	tickers := []SymbolPriceTicker{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *SymbolPriceTicker
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []SymbolPriceTicker{*val}
	}
	*t = tickers
	return nil
}

// SymbolOrderbookTicker represents a symbol orderbook ticker detail
type SymbolOrderbookTicker struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	BidQty   string `json:"bidQty"`
	AskPrice string `json:"askPrice"`
	AskQty   string `json:"askQty"`
}

// SymbolOrderbookTickerList represents a list of symbols orderbook ticker detail
type SymbolOrderbookTickerList []SymbolOrderbookTicker

func (t *SymbolOrderbookTickerList) UnmarshalJSON(data []byte) error {
	tickers := []SymbolOrderbookTicker{}
	err := json.Unmarshal(data, &tickers)
	if err != nil {
		var val *SymbolOrderbookTicker
		err = json.Unmarshal(data, &val)
		if err != nil {
			return err
		}
		tickers = []SymbolOrderbookTicker{*val}
	}
	*t = tickers
	return nil
}

// SubAccountCreationResponse represents a sub-account creation response.
type SubAccountCreationResponse struct {
	SubAccount string `json:"subAccount"`
	Note       string `json:"note"`
}

// SubAccounts represents list of sub-accounts and sub-account detail
type SubAccounts struct {
	SubAccounts []struct {
		SubAccount string     `json:"subAccount"`
		IsFreeze   bool       `json:"isFreeze"`
		CreateTime types.Time `json:"createTime"`
		UID        string     `json:"uid"`
	} `json:"subAccounts"`
}

// SubAccountAPIDetail represents a sub-account API key detail
type SubAccountAPIDetail struct {
	SubAccount  string     `json:"subAccount"`
	Note        string     `json:"note"`
	APIKey      string     `json:"apiKey"`
	SecretKey   string     `json:"secretKey"`
	Permissions string     `json:"permissions"`
	IP          string     `json:"ip"`
	CreatTime   types.Time `json:"creatTime"`
}

// SubAccountsAPIs represents a sub-account API keys detail
type SubAccountsAPIs struct {
	SubAccount []SubAccountAPIDetail `json:"subAccount"`
}

// UniversalTransferResponse represents a universal asset transfer response
type UniversalTransferResponse struct {
	TransferID int64 `json:"tranId"`
}

// UniversalTransferHistoryData represents a universal asset transfer history detail
type UniversalTransferHistoryData struct {
	TranID          string       `json:"tranId"`
	FromAccount     string       `json:"fromAccount"`
	ToAccount       string       `json:"toAccount"`
	ClientTranID    string       `json:"clientTranId"`
	Asset           string       `json:"asset"`
	Amount          types.Number `json:"amount"`
	FromAccountType string       `json:"fromAccountType"`
	ToAccountType   string       `json:"toAccountType"`
	FromSymbol      string       `json:"fromSymbol"`
	ToSymbol        string       `json:"toSymbol"`
	Status          string       `json:"status"`
	Timestamp       types.Time   `json:"timestamp"`
}

// SubAccountAssetBalances represents a sub-account asset balances
type SubAccountAssetBalances struct {
	Balances []AccountBalanceInfo `json:"balances"`
}

// AccountBalanceInfo represents an account balance information
type AccountBalanceInfo struct {
	Asset  string       `json:"asset"`
	Free   types.Number `json:"free"`
	Locked types.Number `json:"locked"`
}

// KYCStatusInfo represents a KYC status information
type KYCStatusInfo struct {
	Status string `json:"status"`
}

// OrderDetail represents an order detail
type OrderDetail struct {
	Symbol              string       `json:"symbol"`
	OrderID             string       `json:"orderId"`
	OrderListID         int          `json:"orderListId"`
	Price               types.Number `json:"price"`
	OrigQty             types.Number `json:"origQty"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	TransactTime        types.Time   `json:"transactTime"`
	ClientOrderID       string       `json:"clientOrderId"`
	ExecutedQty         string       `json:"executedQty"`
	CummulativeQuoteQty string       `json:"cummulativeQuoteQty"`
	TimeInForce         string       `json:"timeInForce"`
	Status              string       `json:"status"`
	OrigClientOrderID   string       `json:"origClientOrderId"`
	StopPrice           string       `json:"stopPrice"`
	IcebergQty          string       `json:"icebergQty"`
	Time                int64        `json:"time"`
	UpdateTime          int64        `json:"updateTime"`
	IsWorking           bool         `json:"isWorking"`
	OrigQuoteOrderQty   string       `json:"origQuoteOrderQty"`
}

// BatchOrderCreationParam represents a batch order creation parameter
type BatchOrderCreationParam struct {
	OrderType        string  `json:"type"`
	Price            float64 `json:"price,omitempty,string"`
	Quantity         float64 `json:"quantity,omitempty,string"`
	QuoteOrderQty    float64 `json:"quoteOrderQty,omitempty,string"`
	Symbol           string  `json:"symbol,omitempty"`
	Side             string  `json:"side,omitempty"`
	NewClientOrderID int64   `json:"newClientOrderId,omitempty"`
}

// AccountDetail represents an account detail information
type AccountDetail struct {
	CanTrade    bool                 `json:"canTrade"`
	CanWithdraw bool                 `json:"canWithdraw"`
	CanDeposit  bool                 `json:"canDeposit"`
	UpdateTime  types.Time           `json:"updateTime"`
	AccountType string               `json:"accountType"`
	Balances    []AccountBalanceInfo `json:"balances"`
	Permissions []string             `json:"permissions"`
}

// AccountTrade represents an account trade detail
type AccountTrade struct {
	Symbol          string       `json:"symbol"`
	ID              string       `json:"id"`
	OrderID         string       `json:"orderId"`
	OrderListID     int64        `json:"orderListId"`
	Price           types.Number `json:"price"`
	Quantity        types.Number `json:"qty"`
	QuoteQuantity   types.Number `json:"quoteQty"`
	Commission      string       `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsBestMatch     bool         `json:"isBestMatch"`
	IsSelfTrade     bool         `json:"isSelfTrade"`
	ClientOrderID   int64        `json:"clientOrderId"`
}

// MXDeductResponse represents an MX deduct response from spot commissions.
type MXDeductResponse struct {
	Data struct {
		MxDeductEnable bool `json:"mxDeductEnable"`
	} `json:"data"`
	Code      int64      `json:"code"`
	Message   string     `json:"msg"`
	Timestamp types.Time `json:"timestamp"`
}

// SymbolCommissionFee represents a symbol trading fee
type SymbolCommissionFee struct {
	Data struct {
		MakerCommission float64 `json:"makerCommission"`
		TakerCommission float64 `json:"takerCommission"`
	} `json:"data"`
	Code      int64      `json:"code"`
	Message   string     `json:"msg"`
	Timestamp types.Time `json:"timestamp"`
}

// CurrencyInformation represents a exchange's currency item details
type CurrencyInformation struct {
	Coin        string `json:"coin"`
	Name        string `json:"Name"`
	NetworkList []struct {
		Coin                    string       `json:"coin"`
		DepositDesc             string       `json:"depositDesc"`
		DepositEnable           bool         `json:"depositEnable"`
		MinConfirm              int64        `json:"minConfirm"`
		Name                    string       `json:"Name"`
		Network                 string       `json:"network"`
		WithdrawEnable          bool         `json:"withdrawEnable"`
		WithdrawFee             types.Number `json:"withdrawFee"`
		WithdrawIntegerMultiple string       `json:"withdrawIntegerMultiple"`
		WithdrawMax             types.Number `json:"withdrawMax"`
		WithdrawMin             types.Number `json:"withdrawMin"`
		SameAddress             bool         `json:"sameAddress"`
		Contract                string       `json:"contract"`
		WithdrawTips            string       `json:"withdrawTips"`
		DepositTips             string       `json:"depositTips"`
		NetWork                 string       `json:"netWork,omitempty"`
	} `json:"networkList"`
}

// IDResponse represents response data which specify id of an order or related
type IDResponse struct {
	ID string `json:"id"`
}

// FundDepositInfo represents a fund deposit detailed information
type FundDepositInfo struct {
	Amount        types.Number `json:"amount"`
	Coin          string       `json:"coin"`
	Network       string       `json:"network"`
	Status        int          `json:"status"`
	Address       string       `json:"address"`
	TransactionID string       `json:"txId"`
	InsertTime    types.Time   `json:"insertTime"`
	UnlockConfirm string       `json:"unlockConfirm"`
	ConfirmTimes  types.Time   `json:"confirmTimes"`
	Memo          string       `json:"memo"`
}

// WithdrawalInfo represents an asset withdrawal detailed information
type WithdrawalInfo struct {
	ID             string       `json:"id"`
	TransactionID  any          `json:"txId"`
	Coin           string       `json:"coin"`
	Network        string       `json:"network"`
	Address        string       `json:"address"`
	Amount         types.Number `json:"amount"`
	TransferType   int64        `json:"transferType"`
	Status         int64        `json:"status"`
	TransactionFee types.Number `json:"transactionFee"`
	ConfirmNo      any          `json:"confirmNo"`
	ApplyTime      types.Time   `json:"applyTime"`
	Remark         string       `json:"remark"`
	Memo           string       `json:"memo"`
	TransHash      string       `json:"transHash"`
	UpdateTime     types.Time   `json:"updateTime"`
	CoinID         string       `json:"coinId"`
	VcoinID        string       `json:"vcoinId"`
}

// DepositAddressInfo represents a deposit address information
type DepositAddressInfo struct {
	Coin    string `json:"coin"`
	Network string `json:"network"`
	Address string `json:"address"`
	Tag     string `json:"tag,omitempty"`
	Memo    string `json:"memo,omitempty"`
}

// WithdrawalAddressTag represents an asset withdrawal address detail
type WithdrawalAddressTag struct {
	Coin       string `json:"coin"`
	Network    string `json:"network"`
	Address    string `json:"address"`
	AddressTag string `json:"addressTag"`
	Memo       string `json:"memo"`
}

// WithdrawalAddressesDetail represents a detailed list of previously used withdrawal addresses
type WithdrawalAddressesDetail struct {
	Data         []WithdrawalAddressTag `json:"data"`
	TotalRecords int64                  `json:"totalRecords"`
	Page         int64                  `json:"page"`
	TotalPageNum int64                  `json:"totalPageNum"`
}

// UserUniversalTransferResponse represents a user account asset transfer response
type UserUniversalTransferResponse struct {
	TranID string `json:"tranId"`
}
