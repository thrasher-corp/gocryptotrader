package mexc

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	filterPercentPriceBySide = "PERCENT_PRICE_BY_SIDE"

	typeFillOrKill        = "FILL_OR_KILL"
	typeImmediateOrCancel = "IMMEDIATE_OR_CANCEL"
	typeLimitMaker        = "LIMIT_MAKER"
	typeLimit             = "LIMIT"
	typeMarket            = "MARKET"
	typeStopLimit         = "STOP_LIMIT"
	typeStopMarketOrder   = "STOP_MARKET_ORDER"
	typePostOnly          = "POST_ONLY"
)

// ExchangeConfig represents rules and symbols of an exchange
type ExchangeConfig struct {
	Timezone        string          `json:"timezone"`
	ServerTime      types.Time      `json:"serverTime"`
	RateLimits      json.RawMessage `json:"rateLimits"`
	ExchangeFilters json.RawMessage `json:"exchangeFilters"`
	Symbols         []*SymbolDetail `json:"symbols"`
}

// SymbolDetail represents a symbol detail.
type SymbolDetail struct {
	Symbol                     string         `json:"symbol"`
	Status                     types.Number   `json:"status"`
	BaseAsset                  string         `json:"baseAsset"`
	BaseAssetPrecision         float64        `json:"baseAssetPrecision"`
	QuoteAsset                 string         `json:"quoteAsset"`
	QuotePrecision             float64        `json:"quotePrecision"`
	QuoteAssetPrecision        float64        `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    float64        `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   float64        `json:"quoteCommissionPrecision"`
	OrderTypes                 []string       `json:"orderTypes"`
	IsSpotTradingAllowed       bool           `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool           `json:"isMarginTradingAllowed"`
	QuoteAmountPrecision       types.Number   `json:"quoteAmountPrecision"`
	BaseSizePrecision          types.Number   `json:"baseSizePrecision"`
	Permissions                []string       `json:"permissions"`
	Filters                    []SymbolFilter `json:"filters"`
	MaxQuoteAmount             types.Number   `json:"maxQuoteAmount"`
	MakerCommission            types.Number   `json:"makerCommission"`
	TakerCommission            types.Number   `json:"takerCommission"`
	QuoteAmountPrecisionMarket types.Number   `json:"quoteAmountPrecisionMarket"`
	MaxQuoteAmountMarket       types.Number   `json:"maxQuoteAmountMarket"`
	FullName                   string         `json:"fullName"`
	TradeSideType              int64          `json:"tradeSideType"`
}

// OfflineSymbol is a symbol taken off the market
type OfflineSymbol struct {
	Symbol string `json:"symbol"`
	// State is 2 for suspended and 3 for delisted
	State       int64      `json:"state"`
	OfflineTime types.Time `json:"offlineTime"`
}

// AnnouncementPage is one page of venue announcements
type AnnouncementPage struct {
	TotalPage types.Number   `json:"totalPage"`
	Details   []Announcement `json:"details"`
}

// Announcement is a venue announcement
type Announcement struct {
	Title    string     `json:"title"`
	URL      string     `json:"url"`
	Language string     `json:"language"`
	PostTime types.Time `json:"postTime"`
}

// SymbolFilter is a trading rule attached to a symbol. PERCENT_PRICE_BY_SIDE bounds the order price
// against the last trade price: a buy at most lastPrice*(1+BidMultiplierUp) and a sell at least
// lastPrice*(1-AskMultiplierDown).
type SymbolFilter struct {
	FilterType        string       `json:"filterType"`
	BidMultiplierUp   types.Number `json:"bidMultiplierUp"`
	AskMultiplierDown types.Number `json:"askMultiplierDown"`
}

// Orderbook represents a symbol orderbook detail
type Orderbook struct {
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
	Timestamp    types.Time                       `json:"timestamp"`
}

// TradeDetail represents a trade detail
type TradeDetail struct {
	ID           string       `json:"id"`
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
	AggregatedTradeID string       `json:"a"`
	FirstTradeID      string       `json:"f"`
	LastTradeID       string       `json:"l"`
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

// UnmarshalJSON deserialises byte data into a CandlestickData instance
func (c *CandlestickData) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[8]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume})
}

// SymbolAveragePrice represents a symbol average price detail
type SymbolAveragePrice struct {
	Mins  int64        `json:"mins"`
	Price types.Number `json:"price"`
}

// TickerData represents a ticker data for a symbol
type TickerData struct {
	Symbol             string          `json:"symbol"`
	PriceChange        types.Number    `json:"priceChange"`
	PriceChangePercent types.Number    `json:"priceChangePercent"`
	PrevClosePrice     types.Number    `json:"prevClosePrice"`
	LastPrice          types.Number    `json:"lastPrice"`
	BidPrice           types.Number    `json:"bidPrice"`
	BidQty             types.Number    `json:"bidQty"`
	AskPrice           types.Number    `json:"askPrice"`
	AskQty             types.Number    `json:"askQty"`
	OpenPrice          types.Number    `json:"openPrice"`
	HighPrice          types.Number    `json:"highPrice"`
	LowPrice           types.Number    `json:"lowPrice"`
	Volume             types.Number    `json:"volume"`
	QuoteVolume        types.Number    `json:"quoteVolume"`
	OpenTime           types.Time      `json:"openTime"`
	CloseTime          types.Time      `json:"closeTime"`
	Count              json.RawMessage `json:"count"`
}

// TickerList represents list of ticker data
type TickerList []TickerData

// UnmarshalJSON deserialises byte data into TickerList. The endpoint returns a single object when one
// symbol is requested and an array otherwise, so the first byte decides which is decoded.
func (t *TickerList) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '{' {
		var val TickerData
		if err := json.Unmarshal(trimmed, &val); err != nil {
			return err
		}
		*t = TickerList{val}
		return nil
	}
	var tickers []TickerData
	if err := json.Unmarshal(data, &tickers); err != nil {
		return err
	}
	*t = tickers
	return nil
}

// SymbolPriceTicker represents a symbol price ticker info
type SymbolPriceTicker struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// SymbolOrderbookTicker represents a symbol orderbook ticker detail
type SymbolOrderbookTicker struct {
	Symbol   string       `json:"symbol"`
	BidPrice types.Number `json:"bidPrice"`
	BidQty   types.Number `json:"bidQty"`
	AskPrice types.Number `json:"askPrice"`
	AskQty   types.Number `json:"askQty"`
}

// SubAccountCreationResponse represents a sub-account creation response.
type SubAccountCreationResponse struct {
	SubAccount string `json:"subAccount"`
	Note       string `json:"note"`
}

// SubAccounts represents list of sub-accounts and sub-account detail
type SubAccounts struct {
	SubAccounts []SubAccountDetail `json:"subAccounts"`
}

// SubAccountDetail holds a sub-account and whether it is frozen
type SubAccountDetail struct {
	SubAccount string     `json:"subAccount"`
	IsFreeze   bool       `json:"isFreeze"`
	CreateTime types.Time `json:"createTime"`
	UID        string     `json:"uid"`
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
	SubAccount []*SubAccountAPIDetail `json:"subAccount"`
}

// AssetTransferResponse represents an asset transfer response
type AssetTransferResponse struct {
	TransferID int64 `json:"tranId"`
}

// UniversalTransferHistoryResponse represents a universal transfer history response detail
type UniversalTransferHistoryResponse struct {
	Rows  []*UniversalTransferHistoryData `json:"rows"`
	Total int64                           `json:"total"`
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

// APIKeyInfo represents an API key's details
type APIKeyInfo struct {
	Note string `json:"note"`
	// AccessKey is the public key. The documentation names this field apikey; the venue sends accessKey.
	AccessKey string `json:"accessKey"`
	// Status is VALID, DELETE or FROZEN
	Status string `json:"status"`
	// Permissions is the comma-separated list of the key's permissions
	Permissions string `json:"permissions"`
	// IPWhiteList is the comma-separated list of IP addresses linked to the key
	IPWhiteList string `json:"ipWhiteList"`
	// CreateTime is sent as an RFC 3339 timestamp with milliseconds and an offset
	// (2026-09-14T18:53:36.000+00:00); the published example gives epoch milliseconds, so both are read.
	CreateTime time.Time `json:"createTime"`
	// RemainingValidity is the days left: -999 for a permanent key, 0 once expired
	RemainingValidity types.Number `json:"remainingValidity"`
}

// UnmarshalJSON decodes an APIKeyInfo, reading createTime as either an RFC 3339 timestamp or epoch
// milliseconds
func (a *APIKeyInfo) UnmarshalJSON(data []byte) error {
	type Alias APIKeyInfo
	chil := &struct {
		*Alias
		CreateTime json.RawMessage `json:"createTime"`
	}{Alias: (*Alias)(a)}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.CreateTime = time.Time{}
	if len(chil.CreateTime) == 0 || string(chil.CreateTime) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(chil.CreateTime, &s); err == nil && strings.Contains(s, "T") {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("createTime: %w", err)
		}
		a.CreateTime = t
		return nil
	}
	var ms types.Time
	if err := json.Unmarshal(chil.CreateTime, &ms); err != nil {
		return fmt.Errorf("createTime: %w", err)
	}
	a.CreateTime = ms.Time()
	return nil
}

// STPGroup represents a self-trade prevention group
type STPGroup struct {
	TradeGroupName string       `json:"tradeGroupName"`
	TradeGroupID   types.Number `json:"tradeGroupId"`
	// TradeGroupUID is the comma-separated list of uids in the group
	TradeGroupUID string     `json:"tradeGroupUid"`
	CreateTime    types.Time `json:"createTime"`
	UpdateTime    types.Time `json:"updateTime"`
}

// ListenKeys represents the account's valid listen keys
type ListenKeys struct {
	ListenKeys []string `json:"listenKey"`
	Total      int64    `json:"total"`
	Available  int64    `json:"available"`
}

// OrderDetail represents an order detail
type OrderDetail struct {
	Symbol              string       `json:"symbol"`
	OrderID             string       `json:"orderId"`
	OrderListID         int64        `json:"orderListId"`
	Price               types.Number `json:"price"`
	OrigQty             types.Number `json:"origQty"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	TransactTime        types.Time   `json:"transactTime"`
	ClientOrderID       string       `json:"clientOrderId"`
	ExecutedQty         types.Number `json:"executedQty"`
	CummulativeQuoteQty types.Number `json:"cummulativeQuoteQty"`
	TimeInForce         string       `json:"timeInForce"`
	Status              string       `json:"status"`
	OrigClientOrderID   string       `json:"origClientOrderId"`
	StopPrice           types.Number `json:"stopPrice"`
	IcebergQuantity     types.Number `json:"icebergQty"`
	Time                types.Time   `json:"time"`
	UpdateTime          types.Time   `json:"updateTime"`
	IsWorking           bool         `json:"isWorking"`
	OrigQuoteOrderQty   types.Number `json:"origQuoteOrderQty"`
	// StpMode is the self-trade prevention mode the order was placed with: empty (none), cancel_maker,
	// cancel_taker or cancel_both.
	StpMode string `json:"stpMode"`
	// CancelReason is stp_cancel when the venue cancelled the order under its self-trade prevention mode.
	CancelReason string `json:"cancelReason"`
}

// CancelAllOrdersResponse is the acknowledgement of an account-wide cancel
type CancelAllOrdersResponse struct {
	Code      int64      `json:"code"`
	Message   string     `json:"msg"`
	Timestamp types.Time `json:"timestamp"`
}

// BatchOrderResult is one entry of a batch order creation response. MEXC returns a mixed array where
// a rejected order carries code and msg in place of the order fields; without them a rejected entry
// decodes to a zero-value OrderDetail a caller cannot tell from a placed order. It also carries
// newClientOrderId, the identifier the error record echoes (distinct from clientOrderId).
type BatchOrderResult struct {
	OrderDetail
	NewClientOrderID string `json:"newClientOrderId"`
	Code             int64  `json:"code"`
	Msg              string `json:"msg"`
}

// BatchOrderCreationParam represents a batch order creation parameter
type BatchOrderCreationParam struct {
	OrderType        string        `json:"type"`
	Price            types.Number  `json:"price,omitempty"`
	Quantity         types.Number  `json:"quantity,omitempty"`
	QuoteOrderQty    types.Number  `json:"quoteOrderQty,omitempty"`
	Symbol           currency.Pair `json:"symbol"`
	Side             string        `json:"side,omitempty"`
	NewClientOrderID string        `json:"newClientOrderId,omitempty"`
	// StpMode selects self-trade prevention for the order: cancel_maker, cancel_taker or cancel_both.
	// Left empty the venue applies no self-trade restriction.
	StpMode string `json:"stpMode,omitempty"`
}

// AccountDetail represents an account detail information
type AccountDetail struct {
	CanTrade    bool                 `json:"canTrade"`
	CanWithdraw bool                 `json:"canWithdraw"`
	CanDeposit  bool                 `json:"canDeposit"`
	UpdateTime  types.Time           `json:"updateTime"`
	AccountType string               `json:"accountType"`
	Permissions []string             `json:"permissions"`
	Balances    []AccountBalanceInfo `json:"balances"`
}

// AccountTrade represents an account trade detail
type AccountTrade struct {
	Symbol          string       `json:"symbol"`
	ID              string       `json:"id"`
	ClientOrderID   string       `json:"clientOrderId"`
	OrderID         string       `json:"orderId"`
	OrderListID     int64        `json:"orderListId"`
	Commission      types.Number `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsBestMatch     bool         `json:"isBestMatch"`
	IsSelfTrade     bool         `json:"isSelfTrade"`
	Price           types.Number `json:"price"`
	Quantity        types.Number `json:"qty"`
	QuoteQuantity   types.Number `json:"quoteQty"`
	Time            types.Time   `json:"time"`
}

// MXDeductResponse represents an MX deduct response from spot commissions.
type MXDeductResponse struct {
	Data      MXDeductStatus `json:"data"`
	Code      int64          `json:"code"`
	Message   string         `json:"msg"`
	Timestamp types.Time     `json:"timestamp"`
}

// MXDeductStatus holds whether MX deduction of spot commission is enabled
type MXDeductStatus struct {
	MxDeductEnable bool `json:"mxDeductEnable"`
}

// SymbolCommissionFee represents a symbol trading fee
type SymbolCommissionFee struct {
	Data      CommissionRate `json:"data"`
	Code      int64          `json:"code"`
	Message   string         `json:"msg"`
	Timestamp types.Time     `json:"timestamp"`
}

// CommissionRate holds a symbol's maker and taker commission rates
type CommissionRate struct {
	MakerCommission float64 `json:"makerCommission"`
	TakerCommission float64 `json:"takerCommission"`
}

// CurrencyInformation represents a exchange's currency item details
type CurrencyInformation struct {
	Coin        string            `json:"coin"`
	Name        string            `json:"Name"`
	NetworkList []CurrencyNetwork `json:"networkList"`
}

// CurrencyNetwork holds a currency's deposit and withdrawal details on one network
type CurrencyNetwork struct {
	Coin                    currency.Code `json:"coin"`
	DepositDesc             string        `json:"depositDesc"`
	DepositEnable           bool          `json:"depositEnable"`
	MinConfirm              int64         `json:"minConfirm"`
	Name                    string        `json:"Name"`
	Network                 string        `json:"network"`
	WithdrawEnable          bool          `json:"withdrawEnable"`
	WithdrawFee             types.Number  `json:"withdrawFee"`
	WithdrawIntegerMultiple types.Number  `json:"withdrawIntegerMultiple"`
	WithdrawMax             types.Number  `json:"withdrawMax"`
	WithdrawMin             types.Number  `json:"withdrawMin"`
	SameAddress             bool          `json:"sameAddress"`
	Contract                string        `json:"contract"`
	WithdrawTips            string        `json:"withdrawTips"`
	DepositTips             string        `json:"depositTips"`
	NetWork                 string        `json:"netWork,omitempty"`
}

// IDResponse represents response data which specify id of an order or related
type IDResponse struct {
	ID string `json:"id"`
}

// FundDepositInfo represents a fund deposit detailed information
type FundDepositInfo struct {
	Amount        types.Number  `json:"amount"`
	Coin          currency.Code `json:"coin"`
	Network       string        `json:"network"`
	Status        int64         `json:"status"`
	Address       string        `json:"address"`
	TransactionID string        `json:"txId"`
	UnlockConfirm string        `json:"unlockConfirm"`
	Memo          string        `json:"memo"`
	InsertTime    types.Time    `json:"insertTime"`
	// ConfirmTimes is a network-confirmation counter (e.g. "241" or "1/1"), NOT a timestamp. It was
	// previously decoded as types.Time, which made a plain count such as "241" fail with
	// "invalid timestamp" and, when it did decode, stamped the record at the zero time. The deposit's real
	// time is insertTime.
	ConfirmTimes string `json:"confirmTimes"`
}

// WithdrawalInfo represents an asset withdrawal detailed information
type WithdrawalInfo struct {
	ID             string       `json:"id"`
	TransactionID  string       `json:"txId"`
	Coin           string       `json:"coin"`
	Network        string       `json:"network"`
	Address        string       `json:"address"`
	TransferType   int64        `json:"transferType"`
	Status         int64        `json:"status"`
	ConfirmNo      any          `json:"confirmNo"`
	Remark         string       `json:"remark"`
	Memo           string       `json:"memo"`
	TransHash      string       `json:"transHash"`
	CoinID         string       `json:"coinId"`
	VcoinID        string       `json:"vcoinId"`
	TransactionFee types.Number `json:"transactionFee"`
	Amount         types.Number `json:"amount"`
	ApplyTime      types.Time   `json:"applyTime"`
	UpdateTime     types.Time   `json:"updateTime"`
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
	Coin       currency.Code `json:"coin"`
	Network    string        `json:"network"`
	Address    string        `json:"address"`
	AddressTag string        `json:"addressTag"`
	Memo       string        `json:"memo"`
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
	CommissionFeeMX   types.Number  `json:"convertMx"`
	CommissionFeeUSDT types.Number  `json:"convertUsdt"`
	Balance           types.Number  `json:"balance"`
	Asset             currency.Code `json:"asset"`
	Code              string        `json:"code"`
	Message           string        `json:"message"`
}

// DustConvertResponse holds a dust asset conversion response
type DustConvertResponse struct {
	SuccessList  []currency.Code                   `json:"successList"`
	FailedList   []DustAssetConversionFailResponse `json:"failedList"`
	TotalConvert types.Number                      `json:"totalConvert"`
	ConvertFee   types.Number                      `json:"convertFee"`
}

// DustAssetConversionFailResponse represents a dust asset conversion failure message for each asset.
type DustAssetConversionFailResponse struct {
	Asset   string `json:"asset"`
	Message string `json:"message"`
	Code    int64  `json:"code"`
}

// DustLogDetail represents a dust log detail
type DustLogDetail struct {
	Data         []DustConversion `json:"data"`
	Page         int64            `json:"page"`
	TotalRecords int64            `json:"totalRecords"`
	TotalPageNum int64            `json:"totalPageNum"`
}

// DustConversion holds one dust conversion and the assets it converted
type DustConversion struct {
	TotalConvert   types.Number        `json:"totalConvert"`
	TotalFee       types.Number        `json:"totalFee"`
	ConvertTime    types.Time          `json:"convertTime"`
	ConvertDetails []DustConvertDetail `json:"convertDetails"`
}

// DustConvertDetail holds the conversion of one asset within a dust conversion
type DustConvertDetail struct {
	ID      string        `json:"id"`
	Convert types.Number  `json:"convert"`
	Fee     types.Number  `json:"fee"`
	Amount  types.Number  `json:"amount"`
	Time    types.Time    `json:"time"`
	Asset   currency.Code `json:"asset"`
}

// InternalTransferDetail represents an internal asset transfer list
type InternalTransferDetail struct {
	Page         int64                    `json:"page"`
	TotalRecords int64                    `json:"totalRecords"`
	TotalPageNum int64                    `json:"totalPageNum"`
	Data         []InternalTransferRecord `json:"data"`
}

// InternalTransferRecord holds an internal asset transfer
type InternalTransferRecord struct {
	TransferID    string       `json:"tranId"`
	Asset         string       `json:"asset"`
	Amount        types.Number `json:"amount"`
	ToAccountType string       `json:"toAccountType"`
	ToAccount     string       `json:"toAccount"`
	FromAccount   string       `json:"fromAccount"`
	Status        string       `json:"status"`
	Timestamp     types.Time   `json:"timestamp"`
}

// RebateHistory holds rebate transactions related to a user's trading activity
type RebateHistory struct {
	Page         int64           `json:"page"`
	TotalRecords int64           `json:"totalRecords"`
	TotalPageNum int64           `json:"totalPageNum"`
	Data         []InviteeRebate `json:"data"`
}

// InviteeRebate holds the rebates earned from an invited user
type InviteeRebate struct {
	Spot       string       `json:"spot"`
	Futures    string       `json:"futures"`
	Total      types.Number `json:"total"`
	UID        string       `json:"uid"`
	Account    string       `json:"account"`
	InviteTime types.Time   `json:"inviteTime"`
}

// RebateRecordDetail holds rebate records detail
type RebateRecordDetail struct {
	Page         int64          `json:"page"`
	TotalRecords int64          `json:"totalRecords"`
	TotalPageNum int64          `json:"totalPageNum"`
	Data         []RebateRecord `json:"data"`
}

// RebateRecord holds a rebate earned on an invited user's trade
type RebateRecord struct {
	Asset      string       `json:"asset"`
	Type       string       `json:"type"`
	Rate       types.Number `json:"rate"`
	Amount     types.Number `json:"amount"`
	UID        string       `json:"uid"`
	Account    string       `json:"account"`
	TradeTime  types.Time   `json:"tradeTime"`
	UpdateTime types.Time   `json:"updateTime"`
}

// ReferCode holds a refer code
type ReferCode struct {
	ReferCode string `json:"referCode"`
}

// AffiliateCommissionRecord holds an affiliate commission records as a list
type AffiliateCommissionRecord struct {
	Success bool                    `json:"success"`
	Code    int64                   `json:"code"`
	Message string                  `json:"message"`
	Data    AffiliateCommissionPage `json:"data"`
}

// AffiliateCommissionPage holds a page of affiliate commission records and their totals
type AffiliateCommissionPage struct {
	PageSize                  int64                 `json:"pageSize"`
	TotalCount                int64                 `json:"totalCount"`
	TotalPage                 int64                 `json:"totalPage"`
	CurrentPage               int64                 `json:"currentPage"`
	USDTAmount                types.Number          `json:"usdtAmount"`
	TotalCommissionUsdtAmount types.Number          `json:"totalCommissionUsdtAmount"`
	TotalTradeUsdtAmount      types.Number          `json:"totalTradeUsdtAmount"`
	Finished                  types.Number          `json:"finished"`
	ResultList                []AffiliateCommission `json:"resultList"`
}

// AffiliateCommission holds the commission earned from an invited user
type AffiliateCommission struct {
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
}

// AffiliateWithdrawRecords holds a list of withdrawal records
type AffiliateWithdrawRecords struct {
	Success bool                  `json:"success"`
	Code    int64                 `json:"code"`
	Message string                `json:"message"`
	Data    AffiliateWithdrawPage `json:"data"`
}

// AffiliateWithdrawPage holds a page of affiliate withdrawal records
type AffiliateWithdrawPage struct {
	PageSize    int64                 `json:"pageSize"`
	TotalCount  int64                 `json:"totalCount"`
	TotalPage   int64                 `json:"totalPage"`
	CurrentPage int64                 `json:"currentPage"`
	ResultList  []AffiliateWithdrawal `json:"resultList"`
}

// AffiliateWithdrawal holds an affiliate commission withdrawal
type AffiliateWithdrawal struct {
	WithdrawTime types.Time   `json:"withdrawTime"`
	Asset        string       `json:"asset"`
	Amount       types.Number `json:"amount"`
}

// RebateAffiliateCommissionDetail holds a rebate affiliate commission detail
type RebateAffiliateCommissionDetail struct {
	Success bool                          `json:"success"`
	Code    int64                         `json:"code"`
	Message any                           `json:"message"`
	Data    AffiliateCommissionDetailPage `json:"data"`
}

// AffiliateCommissionDetailPage holds a page of affiliate commission details and their totals
type AffiliateCommissionDetailPage struct {
	PageSize                  int64                       `json:"pageSize"`
	TotalCount                int64                       `json:"totalCount"`
	TotalPage                 int64                       `json:"totalPage"`
	CurrentPage               int64                       `json:"currentPage"`
	TotalCommissionUsdtAmount types.Number                `json:"totalCommissionUsdtAmount"`
	TotalTradeUsdtAmount      types.Number                `json:"totalTradeUsdtAmount"`
	ResultList                []AffiliateCommissionDetail `json:"resultList"`
}

// AffiliateCommissionDetail holds the commission earned on one trade or deposit
type AffiliateCommissionDetail struct {
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
	Commission     types.Number `json:"commission"`
	Currency       string       `json:"currency"`
}

// AffiliateCampaignData holds an affiliate campaign data
type AffiliateCampaignData struct {
	Success bool                  `json:"success"`
	Code    int64                 `json:"code"`
	Message any                   `json:"message"`
	Data    AffiliateCampaignPage `json:"data"`
}

// AffiliateCampaignPage holds a page of affiliate campaigns
type AffiliateCampaignPage struct {
	PageSize    int64           `json:"pageSize"`
	TotalCount  int64           `json:"totalCount"`
	TotalPage   int64           `json:"totalPage"`
	CurrentPage int64           `json:"currentPage"`
	ResultList  []*CampaignData `json:"resultList"`
}

// CampaignData represents a campaign data
type CampaignData struct {
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
}

// AffiliateReferralData holds an affiliate referral data
type AffiliateReferralData struct {
	Success bool                  `json:"success"`
	Code    int64                 `json:"code"`
	Message any                   `json:"message"`
	Data    AffiliateReferralPage `json:"data"`
}

// AffiliateReferralPage holds a page of affiliate referrals
type AffiliateReferralPage struct {
	PageSize    int64           `json:"pageSize"`
	TotalCount  int64           `json:"totalCount"`
	TotalPage   int64           `json:"totalPage"`
	CurrentPage int64           `json:"currentPage"`
	ResultList  []*ReferralData `json:"resultList"`
}

// ReferralData holds a referral detail
type ReferralData struct {
	UID              string       `json:"uid"`
	NickName         string       `json:"nickName"`
	Email            string       `json:"email"`
	RegisterTime     types.Time   `json:"registerTime"`
	InviteCode       string       `json:"inviteCode"`
	DepositAmount    types.Number `json:"depositAmount"`
	TradingAmount    types.Number `json:"tradingAmount"`
	Commission       types.Number `json:"commission"`
	FirstDepositTime types.Time   `json:"firstDepositTime"`
	FirstTradeTime   types.Time   `json:"firstTradeTime"`
	LastDepositTime  types.Time   `json:"lastDepositTime"`
	LastTradeTime    types.Time   `json:"lastTradeTime"`
	WithdrawAmount   types.Number `json:"withdrawAmount"`
	Asset            string       `json:"asset"`
	Identification   int64        `json:"identification"`
}

// SubAffiliateData represents a sub-affiliate details
type SubAffiliateData struct {
	Success bool             `json:"success"`
	Code    int64            `json:"code"`
	Message any              `json:"message"`
	Data    SubAffiliatePage `json:"data"`
}

// SubAffiliatePage holds a page of sub-affiliates
type SubAffiliatePage struct {
	PageSize    int64          `json:"pageSize"`
	TotalCount  int64          `json:"totalCount"`
	TotalPage   int64          `json:"totalPage"`
	CurrentPage int64          `json:"currentPage"`
	ResultList  []SubAffiliate `json:"resultList"`
}

// SubAffiliate holds a sub-affiliate and its activity
type SubAffiliate struct {
	SubaffiliateName string       `json:"subaffiliateName"`
	SubaffiliateMail string       `json:"subaffiliateMail"`
	Campaign         string       `json:"campaign"`
	InviteCode       string       `json:"inviteCode"`
	ActivationTime   types.Time   `json:"activationTime"`
	Registered       int64        `json:"registered"`
	Deposited        int64        `json:"deposited"`
	DepositAmount    types.Number `json:"depositAmount"`
	Commission       types.Number `json:"commission"`
}

// WsSubscriptionPayload represents a websocket subscription/unsubscription payload
type WsSubscriptionPayload struct {
	ID     int64    `json:"id,omitempty"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// WsSubscriptionResponse represents a websocket subscription status message response detail
type WsSubscriptionResponse struct {
	ID      int64  `json:"id"`
	Code    int64  `json:"code"` // default: 0
	Message string `json:"msg"`
}
