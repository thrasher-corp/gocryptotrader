package mexc

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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
	BaseAssetPrecision         float64      `json:"baseAssetPrecision"`
	QuoteAsset                 string       `json:"quoteAsset"`
	QuotePrecision             float64      `json:"quotePrecision"`
	QuoteAssetPrecision        float64      `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    float64      `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   float64      `json:"quoteCommissionPrecision"`
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

// AssetTransferResponse represents an asset transfer response
type AssetTransferResponse struct {
	TransferID int64 `json:"tranId"`
}

// UniversalTransferHistoryData represents a universal asset transfer history detail
type UniversalTransferHistoryData struct {
	TranID          string       `json:"tranId"`
	ClientTranID    string       `json:"clientTranId"`
	Asset           string       `json:"asset"`
	Amount          types.Number `json:"amount"`
	FromAccountType string       `json:"fromAccountType"`
	ToAccountType   string       `json:"toAccountType"`
	FromSymbol      string       `json:"fromSymbol"`
	ToSymbol        string       `json:"toSymbol"`
	Status          string       `json:"status"`
	Timestamp       types.Time   `json:"timestamp"`

	// Used with sub-account universal asset transfers
	FromAccount string `json:"fromAccount"`
	ToAccount   string `json:"toAccount"`
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

// AssetConvertableToMX represents assets that can be converted to MX token
type AssetConvertableToMX struct {
	CommissionFeeMX   string        `json:"convertMx"`
	CommissionFeeUSDT string        `json:"convertUsdt"`
	Balance           types.Number  `json:"balance"`
	Asset             currency.Code `json:"asset"`
	Code              string        `json:"code"`
	Message           string        `json:"message"`
}

// DustConvertResponse represents a dust asset convertion response
type DustConvertResponse struct {
	SuccessList  []currency.Code `json:"successList"`
	FailedList   []currency.Code `json:"failedList"`
	TotalConvert types.Number    `json:"totalConvert"`
	ConvertFee   types.Number    `json:"convertFee"`
}

// DustLogDetail represents a dust log detail
type DustLogDetail struct {
	Data []struct {
		TotalConvert   types.Number `json:"totalConvert"`
		TotalFee       types.Number `json:"totalFee"`
		ConvertTime    types.Time   `json:"convertTime"`
		ConvertDetails []struct {
			ID      string        `json:"id"`
			Convert types.Number  `json:"convert"`
			Fee     types.Number  `json:"fee"`
			Amount  string        `json:"amount"`
			Time    types.Time    `json:"time"`
			Asset   currency.Code `json:"asset"`
		} `json:"convertDetails"`
	} `json:"data"`
	Page         int64 `json:"page"`
	TotalRecords int64 `json:"totalRecords"`
	TotalPageNum int64 `json:"totalPageNum"`
}

// InternalTransferDetail represents an internal asset transfer list
type InternalTransferDetail struct {
	Page         int64 `json:"page"`
	TotalRecords int64 `json:"totalRecords"`
	TotalPageNum int64 `json:"totalPageNum"`
	Data         []struct {
		TransferID    string       `json:"tranId"`
		Asset         string       `json:"asset"`
		Amount        types.Number `json:"amount"`
		ToAccountType string       `json:"toAccountType"`
		ToAccount     string       `json:"toAccount"`
		FromAccount   string       `json:"fromAccount"`
		Status        string       `json:"status"`
		Timestamp     types.Time   `json:"timestamp"`
	} `json:"data"`
}

// RebateHistory holds rebate transactions related to a user's trading activity
type RebateHistory struct {
	Page         int64 `json:"page"`
	TotalRecords int64 `json:"totalRecords"`
	TotalPageNum int64 `json:"totalPageNum"`
	Data         []struct {
		Spot       string       `json:"spot"`
		Futures    string       `json:"futures"`
		Total      types.Number `json:"total"`
		UID        string       `json:"uid"`
		Account    string       `json:"account"`
		InviteTime types.Time   `json:"inviteTime"`
	} `json:"data"`
}

// RebateRecordDetail holds rebate records detail
type RebateRecordDetail struct {
	Page         int64 `json:"page"`
	TotalRecords int64 `json:"totalRecords"`
	TotalPageNum int64 `json:"totalPageNum"`
	Data         []struct {
		Asset      string       `json:"asset"`
		Type       string       `json:"type"`
		Rate       types.Number `json:"rate"`
		Amount     types.Number `json:"amount"`
		UID        string       `json:"uid"`
		Account    string       `json:"account"`
		TradeTime  types.Time   `json:"tradeTime"`
		UpdateTime types.Time   `json:"updateTime"`
	} `json:"data"`
}

// ReferCode holds a refer code
type ReferCode struct {
	ReferCode string `json:"referCode"`
}

// AffiliateCommissionRecord holds an affiliate commission records as a list
type AffiliateCommissionRecord struct {
	Success bool   `json:"success"`
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageSize                  int64        `json:"pageSize"`
		TotalCount                int64        `json:"totalCount"`
		TotalPage                 int64        `json:"totalPage"`
		CurrentPage               int64        `json:"currentPage"`
		USDTAmount                types.Number `json:"usdtAmount"`
		TotalCommissionUsdtAmount types.Number `json:"totalCommissionUsdtAmount"`
		TotalTradeUsdtAmount      types.Number `json:"totalTradeUsdtAmount"`
		Finished                  types.Number `json:"finished"`
		ResultList                []struct {
			UID              string       `json:"uid"`
			Account          string       `json:"account"`
			InviteCode       string       `json:"inviteCode"`
			InviteTime       types.Time   `json:"inviteTime"`
			Spot             string       `json:"spot"`
			ETF              string       `json:"etf"`
			Futures          string       `json:"futures"`
			Total            types.Number `json:"total"`
			Deposit          types.Number `json:"deposit"`
			FirstDepositTime types.Time   `json:"firstDepositTime"`
		} `json:"resultList"`
	} `json:"data"`
}

// AffiliateWithdrawRecords holds a list of withdrawal records
type AffiliateWithdrawRecords struct {
	Success bool   `json:"success"`
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			WithdrawTime types.Time   `json:"withdrawTime"`
			Asset        string       `json:"asset"`
			Amount       types.Number `json:"amount"`
		} `json:"resultList"`
	} `json:"data"`
}

// RebateAffiliateCommissionDetail holds a rebate affiliate commission detail
type RebateAffiliateCommissionDetail struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Message any   `json:"message"`
	Data    struct {
		PageSize                  int64        `json:"pageSize"`
		TotalCount                int64        `json:"totalCount"`
		TotalPage                 int64        `json:"totalPage"`
		CurrentPage               int64        `json:"currentPage"`
		TotalCommissionUsdtAmount types.Number `json:"totalCommissionUsdtAmount"`
		TotalTradeUsdtAmount      types.Number `json:"totalTradeUsdtAmount"`
		ResultList                []struct {
			Type           int64        `json:"type"`
			SourceType     int64        `json:"sourceType"`
			State          int64        `json:"state"`
			Date           types.Time   `json:"date"`
			UID            string       `json:"uid"`
			Rate           float64      `json:"rate"`
			Symbol         string       `json:"symbol"`
			TakerAmount    types.Number `json:"takerAmount"`
			MakerAmount    types.Number `json:"makerAmount"`
			AmountCurrency string       `json:"amountCurrency"`
			UsdtAmount     types.Number `json:"usdtAmount"`
			Commission     string       `json:"commission"`
			Currency       string       `json:"currency"`
		} `json:"resultList"`
	} `json:"data"`
}

// AffiliateCampaignData holds an affiliate campaign data
type AffiliateCampaignData struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Message any   `json:"message"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			Campaign      string       `json:"campaign"`
			InviteCode    string       `json:"inviteCode"`
			ClickTime     types.Time   `json:"clickTime"`
			CreateTime    types.Time   `json:"createTime"`
			Signup        int64        `json:"signup"`
			Traded        int64        `json:"traded"`
			Deposited     int64        `json:"deposited"`
			DepositAmount types.Number `json:"depositAmount"`
			TradingAmount types.Number `json:"tradingAmount"`
			Commission    types.Number `json:"commission"`
		} `json:"resultList"`
	} `json:"data"`
}

// AffiliateReferralData holds an affiliate referral data
type AffiliateReferralData struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Message any   `json:"message"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			UID              string       `json:"uid"`
			NickName         string       `json:"nickName"`
			Email            string       `json:"email"`
			RegisterTime     types.Time   `json:"registerTime"`
			InviteCode       string       `json:"inviteCode"`
			DepositAmount    types.Number `json:"depositAmount"`
			TradingAmount    types.Number `json:"tradingAmount"`
			Commission       string       `json:"commission"`
			FirstDepositTime types.Time   `json:"firstDepositTime"`
			FirstTradeTime   types.Time   `json:"firstTradeTime"`
			LastDepositTime  types.Time   `json:"lastDepositTime"`
			LastTradeTime    types.Time   `json:"lastTradeTime"`
			WithdrawAmount   types.Number `json:"withdrawAmount"`
			Asset            string       `json:"asset"`
			Identification   int64        `json:"identification"`
		} `json:"resultList"`
	} `json:"data"`
}

// SubAffiliateData represents a sub-affiliate details
type SubAffiliateData struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Message any   `json:"message"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			SubaffiliateName string `json:"subaffiliateName"`
			SubaffiliateMail string `json:"subaffiliateMail"`
			Campaign         string `json:"campaign"`
			InviteCode       string `json:"inviteCode"`
			ActivationTime   int64  `json:"activationTime"`
			Registered       int64  `json:"registered"`
			Deposited        int64  `json:"deposited"`
			DepositAmount    string `json:"depositAmount"`
			Commission       string `json:"commission"`
		} `json:"resultList"`
	} `json:"data"`
}

// FuturesContractsDetail holds a list of futures contracts detail
type FuturesContractsDetail struct {
	Success bool                 `json:"success"`
	Code    int64                `json:"code"`
	Data    FuturesContractsList `json:"data"`
}

// FuturesContractsList holds a list of futures contracts
type FuturesContractsList []FuturesContractDetail

// UnmarshalJSON deserializes a futures contract list byte data into FuturesContractsList instance
func (fcl *FuturesContractsList) UnmarshalJSON(data []byte) error {
	var target []FuturesContractDetail
	err := json.Unmarshal(data, &target)
	if err != nil {
		var newTarget *FuturesContractDetail
		err = json.Unmarshal(data, &newTarget)
		if err != nil {
			return err
		}
		target = append(target, *newTarget)
	}
	*fcl = target
	return nil
}

// FuturesContractDetail holds a single futures contract detail
type FuturesContractDetail struct {
	Symbol                     string   `json:"symbol"`
	DisplayName                string   `json:"displayName"`
	DisplayNameEn              string   `json:"displayNameEn"`
	PositionOpenType           int64    `json:"positionOpenType"`
	BaseCoin                   string   `json:"baseCoin"`
	QuoteCoin                  string   `json:"quoteCoin"`
	SettleCoin                 string   `json:"settleCoin"`
	ContractSize               float64  `json:"contractSize"`
	MinLeverage                int64    `json:"minLeverage"`
	MaxLeverage                int64    `json:"maxLeverage"`
	PriceScale                 int64    `json:"priceScale"`
	VolumeScale                int64    `json:"volScale"`
	AmountScale                int64    `json:"amountScale"`
	PriceUnit                  float64  `json:"priceUnit"`
	VolUnit                    int64    `json:"volUnit"`
	MinVol                     int64    `json:"minVol"`
	MaxVol                     int64    `json:"maxVol"`
	BidLimitPriceRate          float64  `json:"bidLimitPriceRate"`
	AskLimitPriceRate          float64  `json:"askLimitPriceRate"`
	TakerFeeRate               float64  `json:"takerFeeRate"`
	MakerFeeRate               float64  `json:"makerFeeRate"`
	MaintenanceMarginRate      float64  `json:"maintenanceMarginRate"`
	InitialMarginRate          float64  `json:"initialMarginRate"`
	RiskBaseVol                int64    `json:"riskBaseVol"`
	RiskIncrVol                int64    `json:"riskIncrVol"`
	RiskIncrMmr                float64  `json:"riskIncrMmr"`
	RiskIncrImr                float64  `json:"riskIncrImr"`
	RiskLevelLimit             int64    `json:"riskLevelLimit"`
	PriceCoefficientVariation  float64  `json:"priceCoefficientVariation"`
	IndexOrigin                []string `json:"indexOrigin"`
	State                      int64    `json:"state"`
	IsNew                      bool     `json:"isNew"`
	IsHot                      bool     `json:"isHot"`
	IsHidden                   bool     `json:"isHidden"`
	ConceptPlate               []string `json:"conceptPlate"`
	RiskLimitType              string   `json:"riskLimitType"`
	MaxNumOrders               []int64  `json:"maxNumOrders"`
	MarketOrderMaxLevel        int64    `json:"marketOrderMaxLevel"`
	MarketOrderPriceLimitRate1 float64  `json:"marketOrderPriceLimitRate1"`
	MarketOrderPriceLimitRate2 float64  `json:"marketOrderPriceLimitRate2"`
	TriggerProtect             float64  `json:"triggerProtect"`
	Appraisal                  int64    `json:"appraisal"`
	ShowAppraisalCountdown     int64    `json:"showAppraisalCountdown"`
	AutomaticDelivery          int64    `json:"automaticDelivery"`
	APIAllowed                 bool     `json:"apiAllowed"`
}

// TransferableCurrencies holds a list of transferable currencies
type TransferableCurrencies struct {
	Success    bool     `json:"success"`
	Code       int64    `json:"code"`
	Currencies []string `json:"data"`
}

// ContractOrderbook holds futures contracts orderbook details
type ContractOrderbook struct {
	Asks      []OrderbookData `json:"asks"`
	Bids      []OrderbookData `json:"bids"`
	Version   int64           `json:"version"`
	Timestamp types.Time      `json:"timestamp"`
}

// OrderbookData holds orderbook depth detail
type OrderbookData orderbook.Tranche

// UnmarshalJSON deserializes slice of byte data into OrderbookData
func (od *OrderbookData) UnmarshalJSON(data []byte) error {
	target := [2]any{&od.Price, &od.Amount}
	return json.Unmarshal(data, &target)
}

// ContractOrderbookWithDepth holds orderbook depth details
type ContractOrderbookWithDepth struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    []struct {
		Asks    []OrderbookDataWithDepth `json:"asks"`
		Bids    []OrderbookDataWithDepth `json:"bids"`
		Version int64                    `json:"version"`
	} `json:"data"`
}

// OrderbookDataWithDepth holds orderbook data with the depth
type OrderbookDataWithDepth orderbook.Tranche

// UnmarshalJSON deserializes slice of byte data into OrderbookDataWithDepth
func (od *OrderbookDataWithDepth) UnmarshalJSON(data []byte) error {
	target := [3]any{&od.Price, &od.Amount, &od.OrderCount}
	return json.Unmarshal(data, &target)
}

// ContractIndexPriceDetail holds index price details
type ContractIndexPriceDetail struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		Symbol     string     `json:"symbol"`
		IndexPrice float64    `json:"indexPrice"`
		Timestamp  types.Time `json:"timestamp"`
	} `json:"data"`
}

// ContractFairPrice holds contracts fair price details
type ContractFairPrice struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		Symbol    string     `json:"symbol"`
		FairPrice float64    `json:"fairPrice"`
		Timestamp types.Time `json:"timestamp"`
	} `json:"data"`
}

// ContractFundingRate holds contract's funding rate details
type ContractFundingRate struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		Symbol         string     `json:"symbol"`
		FundingRate    float64    `json:"fundingRate"`
		MaxFundingRate float64    `json:"maxFundingRate"`
		MinFundingRate float64    `json:"minFundingRate"`
		CollectCycle   int64      `json:"collectCycle"`
		NextSettleTime types.Time `json:"nextSettleTime"`
		Timestamp      types.Time `json:"timestamp"`
	} `json:"data"`
}

// ContractCandlestickData holds futures contracts candlestick data
type ContractCandlestickData struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		Time       []types.Time `json:"time"`
		OpenPrice  []float64    `json:"open"`
		ClosePrice []float64    `json:"close"`
		HighPrice  []float64    `json:"high"`
		LowPrice   []float64    `json:"low"`
		Volume     []float64    `json:"vol"`
		Amount     []float64    `json:"amount"`
		RealOpen   []float64    `json:"realOpen"`
		RealClose  []float64    `json:"realClose"`
		RealHigh   []float64    `json:"realHigh"`
		RealLow    []float64    `json:"realLow"`
	} `json:"data"`
}

// ContractTransactions holds list of contract transactions
type ContractTransactions struct {
	Success bool                `json:"success"`
	Code    int64               `json:"code"`
	Data    []TransactionDetail `json:"data"`
}

// TransactionDetail holds a list of transaction details
type TransactionDetail struct {
	TransactionPrice float64    `json:"p"`
	Quantity         float64    `json:"v"`
	DealType         int64      `json:"T"` // Deal type,1:purchase,2:sell
	OpenPosition     int64      `json:"O"` // Open position, 1: Yes,2: No, when O is 1, vol is additional position
	SelfTransact     int64      `json:"M"` // Self-transact,1:yes,2:no
	TransactionTime  types.Time `json:"t"`
}

// ContractTickers holds list of contracts ticker data and its details.
type ContractTickers struct {
	Success bool                `json:"success"`
	Code    int64               `json:"code"`
	Data    ContractTickersList `json:"data"`
}

// ContractTickerDetail holds a contract ticker detail
type ContractTickerDetail struct {
	Symbol        string     `json:"symbol"`
	LastPrice     float64    `json:"lastPrice"`
	Bid1          float64    `json:"bid1"`
	Ask1          float64    `json:"ask1"`
	Volume24      float64    `json:"volume24"`
	Amount24      float64    `json:"amount24"`
	HoldVol       float64    `json:"holdVol"`
	Lower24Price  float64    `json:"lower24Price"`
	High24Price   float64    `json:"high24Price"`
	RiseFallRate  float64    `json:"riseFallRate"`
	RiseFallValue float64    `json:"riseFallValue"`
	IndexPrice    float64    `json:"indexPrice"`
	FairPrice     float64    `json:"fairPrice"`
	FundingRate   float64    `json:"fundingRate"`
	MaxBidPrice   float64    `json:"maxBidPrice"`
	MinAskPrice   float64    `json:"minAskPrice"`
	Timestamp     types.Time `json:"timestamp"`
}

// ContractTickersList holds a list of contract ticker details.
type ContractTickersList []ContractTickerDetail

// UnmarshalJSON deserializes a contract ticker byte data into ContractTickersList
func (cts *ContractTickersList) UnmarshalJSON(data []byte) error {
	var targets []ContractTickerDetail
	err := json.Unmarshal(data, &targets)
	if err != nil {
		var target *ContractTickerDetail
		err := json.Unmarshal(data, &target)
		if err != nil {
			return err
		}
		targets = append(targets, *target)
	}
	*cts = targets
	return nil
}

// ContractRiskFundBalances holds a list of contracts risk fund balance
type ContractRiskFundBalances struct {
	Success bool                      `json:"success"`
	Code    int64                     `json:"code"`
	Data    []ContractRiskFundBalance `json:"data"`
}

// ContractRiskFundBalance holds a contract fund balance
type ContractRiskFundBalance struct {
	Symbol    string     `json:"symbol"`
	Currency  string     `json:"currency"`
	Available float64    `json:"available"`
	Timestamp types.Time `json:"timestamp"`
}

// ContractRiskFundBalanceHistory holds list of contract risk fund balance history
type ContractRiskFundBalanceHistory struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			Symbol       string     `json:"symbol"`
			Currency     string     `json:"currency"`
			Available    float64    `json:"available"`
			SnapshotTime types.Time `json:"snapshotTime"`
		} `json:"resultList"`
	} `json:"data"`
}

// ContractFundingRateHistory holds contract funding rate history
type ContractFundingRateHistory struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			Symbol      string     `json:"symbol"`
			FundingRate float64    `json:"fundingRate"`
			SettleTime  types.Time `json:"settleTime"`
		} `json:"resultList"`
	} `json:"data"`
}

// UserAssetsBalance holds user asset balances
type UserAssetsBalance struct {
	Success bool               `json:"success"`
	Code    int64              `json:"code"`
	Data    []UserAssetBalance `json:"data"`
}

// UserAssetBalance holds a user's single currency balance details
type UserAssetBalance struct {
	Currency         string  `json:"currency"`
	PositionMargin   float64 `json:"positionMargin"`
	AvailableBalance float64 `json:"availableBalance"`
	CashBalance      float64 `json:"cashBalance"`
	FrozenBalance    float64 `json:"frozenBalance"`
	Equity           float64 `json:"equity"`
	Unrealized       float64 `json:"unrealized"`
	Bonus            float64 `json:"bonus"`
}

// AssetTransfers holds user's asset transfer records
type AssetTransfers struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		PageSize    int64 `json:"pageSize"`
		TotalCount  int64 `json:"totalCount"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"currentPage"`
		ResultList  []struct {
			ID            int64      `json:"id"`
			TransactionID string     `json:"txid"`
			Currency      string     `json:"currency"`
			Amount        float64    `json:"amount"`
			TransferType  string     `json:"type"`
			State         string     `json:"state"`
			CreateTime    types.Time `json:"createTime"`
			UpdateTime    types.Time `json:"updateTime"`
		} `json:"resultList"`
	} `json:"data"`
}

// Positions holds list position and their details
type Positions struct {
	Success bool                 `json:"success"`
	Code    int64                `json:"code"`
	Message string               `json:"message"`
	Data    []UserPositionDetail `json:"data"`
}

// UserPositionDetail holds user's position detailed information
type UserPositionDetail struct {
	PositionID            int64      `json:"positionId"`
	Symbol                string     `json:"symbol"`
	PositionType          int64      `json:"positionType"`
	OpenType              int64      `json:"openType"`
	State                 int64      `json:"state"`
	HoldVolume            float64    `json:"holdVol"`
	FrozenVolume          float64    `json:"frozenVol"`
	CloseVolume           float64    `json:"closeVol"`
	HoldAvgPrice          float64    `json:"holdAvgPrice"`
	OpenAvgPrice          float64    `json:"openAvgPrice"`
	CloseAvgPrice         float64    `json:"closeAvgPrice"`
	LiquidatePrice        float64    `json:"liquidatePrice"`
	OriginalInitialMargin float64    `json:"oim"`
	InitialMargin         float64    `json:"im"`
	HoldFee               float64    `json:"holdFee"`
	Realised              float64    `json:"realised"`
	AdlLevel              int64      `json:"adlLevel"`
	Leverage              int64      `json:"leverage"`
	CreateTime            types.Time `json:"createTime"`
	UpdateTime            types.Time `json:"updateTime"`
	AutoAddIm             bool       `json:"autoAddIm"`
}

// FundingRateHistory holds list of funding rate details and their details
type FundingRateHistory struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		PageSize    int64               `json:"pageSize"`
		TotalCount  int64               `json:"totalCount"`
		TotalPage   int64               `json:"totalPage"`
		CurrentPage int64               `json:"currentPage"`
		ResultList  []FundingRateDetail `json:"resultList"`
	} `json:"data"`
}

// FundingRateDetail holds funding rate details
type FundingRateDetail struct {
	ID            int64      `json:"id"`
	Symbol        string     `json:"symbol"`
	PositionType  int64      `json:"positionType"`
	PositionValue float64    `json:"positionValue"`
	Funding       float64    `json:"funding"`
	Rate          float64    `json:"rate"`
	SettleTime    types.Time `json:"settleTime"`
}

// FuturesOrders holds a futures orders history
type FuturesOrders struct {
	Success bool                 `json:"success"`
	Code    int64                `json:"code"`
	Message string               `json:"message"`
	Data    []FuturesOrderDetail `json:"data"`
}

// FuturesOrderDetail holds futures order details
type FuturesOrderDetail struct {
	OrderID         int64      `json:"orderId"`
	Symbol          string     `json:"symbol"`
	PositionID      int64      `json:"positionId"`
	Price           float64    `json:"price"`
	Volume          float64    `json:"vol"`
	Leverage        int64      `json:"leverage"`
	Side            int64      `json:"side"`
	Category        int64      `json:"category"`
	OrderType       int64      `json:"orderType"`
	DealAvgPrice    float64    `json:"dealAvgPrice"`
	DealVol         float64    `json:"dealVol"`
	OrderMargin     float64    `json:"orderMargin"`
	TakerFee        float64    `json:"takerFee"`
	MakerFee        float64    `json:"makerFee"`
	Profit          float64    `json:"profit"`
	FeeCurrency     string     `json:"feeCurrency"`
	OpenType        int64      `json:"openType"`
	State           int64      `json:"state"`
	ExternalOid     string     `json:"externalOid"`
	ErrorCode       int64      `json:"errorCode"`
	UsedMargin      float64    `json:"usedMargin"`
	CreateTime      types.Time `json:"createTime"`
	UpdateTime      types.Time `json:"updateTime"`
	StopLossPrice   float64    `json:"stopLossPrice"`
	TakeProfitPrice float64    `json:"takeProfitPrice"`
}

// OrderTransactions holds list of transactions for an order.
type OrderTransactions struct {
	Success bool                      `json:"success"`
	Code    int64                     `json:"code"`
	Data    []FuturesOrderTransaction `json:"data"`
}

// FuturesOrderTransaction holds an order's transactions
type FuturesOrderTransaction struct {
	ID              string     `json:"id"`
	Symbol          string     `json:"symbol"`
	Side            int64      `json:"side"`
	Volume          float64    `json:"vol"`
	Price           float64    `json:"price"`
	FeeCurrency     string     `json:"feeCurrency"`
	Fee             float64    `json:"fee"`
	Profit          float64    `json:"profit"`
	Category        int64      `json:"category"`
	OrderID         string     `json:"orderId"`
	Taker           bool       `json:"taker"`
	IsTaker         bool       `json:"isTaker"`
	OpponentOrderID int64      `json:"opponentOrderId"`
	Timestamp       types.Time `json:"timestamp"`
}

// FuturesTriggerOrders holds futures trigger orders
type FuturesTriggerOrders struct {
	Success bool   `json:"success"`
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		ID           int64      `json:"id"`
		Symbol       string     `json:"symbol"`
		Leverage     int64      `json:"leverage"`
		Side         int64      `json:"side"`
		TriggerPrice float64    `json:"triggerPrice"`
		Price        float64    `json:"price"`
		Volume       float64    `json:"vol"`
		OpenType     int64      `json:"openType"`
		TriggerType  int64      `json:"triggerType"`
		State        int64      `json:"state"`
		ExecuteCycle int64      `json:"executeCycle"`
		Trend        int64      `json:"trend"`
		OrderType    int64      `json:"orderType"`
		OrderID      int64      `json:"orderId"`
		ErrorCode    int64      `json:"errorCode"`
		CreateTime   types.Time `json:"createTime"`
		UpdateTime   types.Time `json:"updateTime"`
	} `json:"data"`
}

// FuturesStopLimitOrders holds list of futures stop limit orders details
type FuturesStopLimitOrders struct {
	Success bool                    `json:"success"`
	Code    int64                   `json:"code"`
	Message string                  `json:"message"`
	Data    []FuturesStopLimitOrder `json:"data"`
}

// FuturesStopLimitOrder holds a stop-limit order detail
type FuturesStopLimitOrder struct {
	ID              int64      `json:"id"`
	OrderID         int64      `json:"orderId"`
	Symbol          string     `json:"symbol"`
	PositionID      int64      `json:"positionId"`
	StopLossPrice   float64    `json:"stopLossPrice"`
	TakeProfitPrice float64    `json:"takeProfitPrice"`
	State           int64      `json:"state"`
	TriggerSide     int64      `json:"triggerSide"`
	PositionType    int64      `json:"positionType"`
	Volume          float64    `json:"vol"`
	RealityVol      float64    `json:"realityVol"`
	PlaceOrderID    int64      `json:"placeOrderId"`
	ErrorCode       int64      `json:"errorCode"`
	Version         int64      `json:"version"`
	IsFinished      int64      `json:"isFinished"`
	CreateTime      types.Time `json:"createTime"`
	UpdateTime      types.Time `json:"updateTime"`
}

// FutureRiskLimit holds futures symbols risk limit
type FutureRiskLimit struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    map[string][]struct {
		Level                 int64   `json:"level"`
		MaxVolume             int64   `json:"maxVol"`
		MaxLeverage           int64   `json:"maxLeverage"`
		MaintenanceMarginRate float64 `json:"mmr"`
		InitialMarginRate     float64 `json:"imr"`
		Symbol                string  `json:"symbol"`
		PositionType          int64   `json:"positionType"`
	} `json:"data"`
}

// FuturesTradingFeeRates holds trading fee details of a symbol
type FuturesTradingFeeRates struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		Level            int64   `json:"level"`
		DealAmount       float64 `json:"dealAmount"`
		WalletBalance    float64 `json:"walletBalance"`
		MakerFee         float64 `json:"makerFee"`
		TakerFee         float64 `json:"takerFee"`
		MakerFeeDiscount int64   `json:"makerFeeDiscount"`
		TakerFeeDiscount int64   `json:"takerFeeDiscount"`
	} `json:"data"`
}

// ContractLeverageInfo holds leverage information for an instrument
type ContractLeverageInfo struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    struct {
		PositionType          string  `json:"positionType"`
		Level                 int64   `json:"level"`
		MaintenanceMarginRate float64 `json:"mmr"`
		InitialMarginRate     float64 `json:"imr"`
		Leverage              int64   `json:"leverage"`
	} `json:"data"`
}

// PositionLeverageResponse holds position leverage switching response
type PositionLeverageResponse struct {
	PositionID   int64  `json:"positionId"`
	Leverage     int64  `json:"leverage"`
	PositionType int64  `json:"positionType"`
	Symbol       string `json:"symbol"`
}

// PositionMode represents a position mode response
type PositionMode struct {
	Success bool  `json:"success"`
	Code    int64 `json:"code"`
	Data    int64 `json:"data"` // position mode,1:hedge，2:one-way
}

// StatusResponse holds a status code and status message response
type StatusResponse struct {
	Success bool        `json:"success"`
	Code    int64       `json:"code"`
	Data    interface{} `json:"data"`
}

// PlaceFuturesOrderParams holds futures order creation parameters
type PlaceFuturesOrderParams struct {
	Symbol          string
	Price           float64
	Volume          float64
	Leverage        int64
	Side            order.Side
	OrderType       order.Type
	MarginType      margin.Type
	ExternalOrderID string
	StopLossPrice   float64
	TakeProfitPrice float64
	PositionID      int64
	PositionMode    int64
	ReduceOnly      bool
}

// PlaceFuturesTriggerOrderParams holds a futures trigger price parameters
type PlaceFuturesTriggerOrderParams struct {
	Symbol           string `json:"symbol"`
	Price            float64
	Volume           float64
	Leverage         int64
	Side             order.Side
	MarginType       margin.Type
	TriggerPrice     float64
	TriggerPriceType order.PriceType
	ExecutionCycle   kline.Interval
	OrderType        order.Type
	PriceType        order.PriceType
}

// FuturesOrderInfo represents a futures order info
type FuturesOrderInfo struct {
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	Vol             float64 `json:"vol"`
	Leverage        int64   `json:"leverage"`
	Side            int64   `json:"side"`
	Type            int64   `json:"type"`
	OpenType        int64   `json:"openType"`
	ExternalOrderID string  `json:"externalOid"`
}

// OrderCancellationResponse holds order cancellation response by ExternalOrderID
type OrderCancellationResponse struct {
	Symbol          string `json:"symbol"`
	ExternalOrderID string `json:"externalOid"`
}

// OrderIDDetail holds an order ID and symbol info
type OrderIDDetail struct {
	Symbol  string `json:"symbol"`
	OrderID int64  `json:"orderId"`
}
