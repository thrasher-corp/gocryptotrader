package bitget

import (
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Params is used within functions to make the setting of parameters easier
type Params struct {
	url.Values
}

// RespWrapper wraps responses from the Bitget API for proper JSON decoding
type RespWrapper struct {
	Data any `json:"data"`
}

// AnnouncementResp holds information on announcements
type AnnouncementResp struct {
	AnnouncementID          int64      `json:"annId,string"`
	AnnouncementTitle       string     `json:"annTitle"`
	AnnouncementDescription string     `json:"annDesc"`
	CreationTime            types.Time `json:"cTime"`
	Language                string     `json:"language"`
	AnnouncementURL         string     `json:"annUrl"`
	AnnouncementType        string     `json:"annType"`
	AnnouncementSubType     string     `json:"annSubType"`
}

// TimeResp holds information on the current server time
type TimeResp struct {
	ServerTime types.Time `json:"serverTime"`
}

// TradeRateResp holds information on the current maker and taker fee rates
type TradeRateResp struct {
	MakerFeeRate types.Number `json:"makerFeeRate"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
}

// SpotTrResp holds information on spot transactions
type SpotTrResp struct {
	ID          int64         `json:"id,string"`
	Coin        currency.Code `json:"coin"`
	SpotTaxType string        `json:"spotTaxType"`
	Amount      types.Number  `json:"amount"`
	Fee         types.Number  `json:"fee"`
	Balance     types.Number  `json:"balance"`
	Timestamp   types.Time    `json:"ts"`
}

// FutureTrResp holds information on futures transactions
type FutureTrResp struct {
	ID            int64         `json:"id,string"`
	Symbol        string        `json:"symbol"`
	MarginCoin    currency.Code `json:"marginCoin"`
	FutureTaxType string        `json:"futureTaxType"`
	Amount        types.Number  `json:"amount"`
	Fee           types.Number  `json:"fee"`
	Timestamp     types.Time    `json:"ts"`
}

// MarginTrResp holds information on margin transactions
type MarginTrResp struct {
	ID            int64         `json:"id,string"`
	Coin          currency.Code `json:"coin"`
	Symbol        string        `json:"symbol"`
	MarginTaxType string        `json:"marginTaxType"`
	Amount        types.Number  `json:"amount"`
	Fee           types.Number  `json:"fee"`
	Total         types.Number  `json:"total"`
	Timestamp     types.Time    `json:"ts"`
}

// P2PTrResp holds information on P2P transactions
type P2PTrResp struct {
	ID         int64         `json:"id,string"`
	Coin       currency.Code `json:"coin"`
	P2PTaxType string        `json:"p2pTaxType"`
	Total      types.Number  `json:"total"`
	Timestamp  types.Time    `json:"ts"`
}

// MerchantList is a sub-struct holding information on P2P merchants
type MerchantList struct {
	RegisterTime        types.Time   `json:"registerTime"`
	NickName            string       `json:"nickName"`
	IsOnline            string       `json:"isOnline"`
	AvgPaymentTime      int64        `json:"avgPaymentTime,string"`
	AvgReleaseTime      int64        `json:"avgReleaseTime,string"`
	TotalTrades         int64        `json:"totalTrades,string"`
	TotalBuy            int64        `json:"totalBuy,string"`
	TotalSell           int64        `json:"totalSell,string"`
	TotalCompletionRate types.Number `json:"totalCompletionRate"`
	Trades30D           int64        `json:"trades30d,string"`
	Sell30D             types.Number `json:"sell30d"`
	Buy30D              types.Number `json:"buy30d"`
	CompletionRate30D   types.Number `json:"completionRate30d"`
}

// P2PMerListResp holds information on P2P merchant lists
type P2PMerListResp struct {
	MerchantList      []MerchantList `json:"merchantList"`
	MinimumMerchantID int64          `json:"minMerchantId,string"`
}

// YesNoBool is a type used to unmarshal strings that are either "yes" or "no" into bools
type YesNoBool bool

// P2PMerInfoResp holds information on P2P merchant information
type P2PMerInfoResp struct {
	RegisterTime        types.Time   `json:"registerTime"`
	NickName            string       `json:"nickName"`
	MerchantID          int64        `json:"merchantId,string"`
	AvgPaymentTime      int64        `json:"avgPaymentTime,string"`
	AvgReleaseTime      int64        `json:"avgReleaseTime,string"`
	TotalTrades         int64        `json:"totalTrades,string"`
	TotalBuy            int64        `json:"totalBuy,string"`
	TotalSell           int64        `json:"totalSell,string"`
	TotalCompletionRate types.Number `json:"totalCompletionRate"`
	Trades30D           int64        `json:"trades30d,string"`
	Sell30D             types.Number `json:"sell30d"`
	Buy30D              types.Number `json:"buy30d"`
	CompletionRate30D   types.Number `json:"completionRate30d"`
	KYCStatus           YesNoBool    `json:"kycStatus"`
	EmailBindStatus     YesNoBool    `json:"emailBindStatus"`
	MobileBindStatus    YesNoBool    `json:"mobileBindStatus"`
	Email               string       `json:"email"`
	Mobile              string       `json:"mobile"`
}

// PayMethodInfo is a sub-struct holding information on P2P payment methods
type PayMethodInfo struct {
	Name     string    `json:"name"`
	Required YesNoBool `json:"required"`
	Type     string    `json:"type"`
	Value    string    `json:"value"`
}

// PaymentInfo is a sub-struct holding information on P2P payment methods
type PaymentInfo struct {
	PayMethodName string          `json:"paymethodName"`
	PayMethodID   string          `json:"paymethodId"`
	PayMethodInfo []PayMethodInfo `json:"paymethodInfo"`
}

// P2POrderList is a sub-struct holding information on P2P orders
type P2POrderList struct {
	OrderID        int64         `json:"orderId,string"`
	OrderNum       int64         `json:"orderNo,string"`
	AdvNum         int64         `json:"advNo,string"`
	Side           string        `json:"side"`
	Count          types.Number  `json:"count"`
	FiatCurrency   currency.Code `json:"fiat"`
	CryptoCurrency currency.Code `json:"coin"`
	Price          types.Number  `json:"price"`
	WithdrawTime   types.Time    `json:"withdrawTime"`
	RepresentTime  types.Time    `json:"representTime"`
	ReleaseTime    types.Time    `json:"releaseTime"`
	PaymentTime    types.Time    `json:"paymentTime"`
	Amount         types.Number  `json:"amount"`
	Status         string        `json:"status"`
	BuyerRealName  string        `json:"buyerRealName"`
	SellerRealName string        `json:"sellerRealName"`
	CreationTime   types.Time    `json:"ctime"`
	UpdateTime     types.Time    `json:"utime"`
	PaymentInfo    PaymentInfo   `json:"paymentInfo"`
}

// P2POrdersResp holds information on P2P orders
type P2POrdersResp struct {
	OrderList      []P2POrderList `json:"orderList"`
	MinimumOrderID int64          `json:"minOrderId,string"`
}

// UserLimitList is a sub-struct holding information on P2P user limits
type UserLimitList struct {
	MinimumOrderQuantity int64        `json:"minCompleteNum,string"`
	MaximumOrderQuantity int64        `json:"maxCompleteNum,string"`
	PlaceOrderNumber     int64        `json:"placeOrderNum,string"`
	AllowMerchantPlace   YesNoBool    `json:"allowMerchantPlace"`
	CompleteRate30D      types.Number `json:"completeRate30d"`
	Country              string       `json:"country"`
}

// ReqNameType is a sub-struct holding information on P2P payment methods
type ReqNameType struct {
	Required bool   `json:"required"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// PaymentMethodList is a sub-struct holding information on P2P payment methods
type PaymentMethodList struct {
	PaymentMethod string        `json:"paymentMethod"`
	PaymentID     int64         `json:"paymentId,string"`
	PaymentInfo   []ReqNameType `json:"paymentInfo"`
}

// MerchantCertifiedList is a sub-struct holding information on P2P merchant certifications
type MerchantCertifiedList struct {
	ImageURL string `json:"imageUrl"`
	Desc     string `json:"desc"`
}

// AdvertisementList is a sub-struct holding information on P2P advertisements
type AdvertisementList struct {
	AdvertisementID       int64                   `json:"advId,string"`
	AdvertisementNumber   int64                   `json:"advNo,string"`
	Side                  string                  `json:"side"`
	AdvertisementSize     types.Number            `json:"adSize"`
	Size                  types.Number            `json:"size"`
	CryptoCurrency        currency.Code           `json:"coin"`
	Price                 types.Number            `json:"price"`
	CryptoPrecision       uint8                   `json:"coinPrecision,string"`
	FiatCurrency          currency.Code           `json:"fiat"`
	FiatPrecision         uint8                   `json:"fiatPrecision,string"`
	FiatSymbol            string                  `json:"fiatSymbol"`
	Status                string                  `json:"status"`
	Hide                  YesNoBool               `json:"hide"`
	MaximumOrderQuantity  types.Number            `json:"maxTradeAmount"`
	MinimumOrderQuantity  types.Number            `json:"minTradeAmount"`
	PayDuration           int64                   `json:"payDuration,string"`
	TurnoverNumber        int64                   `json:"turnoverNum,string"`
	TurnoverRate          types.Number            `json:"turnoverRate"`
	Label                 string                  `json:"label"`
	CreationTime          types.Time              `json:"ctime"`
	UpdateTime            types.Time              `json:"utime"`
	UserLimitList         UserLimitList           `json:"userLimitList"`
	PaymentMethodList     []PaymentMethodList     `json:"paymentMethodList"`
	MerchantCertifiedList []MerchantCertifiedList `json:"merchantCertifiedList"`
}

// P2PAdListResp holds information on P2P advertisements
type P2PAdListResp struct {
	AdvertisementList      []AdvertisementList `json:"advList"`
	MinimumAdvertisementID int64               `json:"minAdvId,string"`
}

// WhaleNetFlowResp holds information on whale trading volumes
type WhaleNetFlowResp struct {
	Volume types.Number `json:"volume"`
	Date   types.Time   `json:"date"`
}

// ActiveVolumeResp holds information on active trading volumes
type ActiveVolumeResp struct {
	BuyVolume  types.Number `json:"buyVolume"`
	SellVolume types.Number `json:"sellVolume"`
	Timestamp  types.Time   `json:"ts"`
}

// PosRatFutureResp holds information on position ratios
type PosRatFutureResp struct {
	LongPositionRatio      types.Number `json:"longPositionRatio"`
	ShortPositionRatio     types.Number `json:"shortPositionRatio"`
	LongShortPositionRatio types.Number `json:"longShortPositionRatio"`
	Timestamp              types.Time   `json:"ts"`
}

// PosRatMarginResp holds information on position ratios in margin trading
type PosRatMarginResp struct {
	Timestamp      types.Time   `json:"ts"`
	LongShortRatio types.Number `json:"longShortRatio"`
}

// LoanGrowthResp holds information on loan growth
type LoanGrowthResp struct {
	Timestamp  types.Time   `json:"ts"`
	GrowthRate types.Number `json:"growthRate"`
}

// BorrowRatioResp holds information on borrowing ratios
type BorrowRatioResp struct {
	Timestamp  types.Time   `json:"ts"`
	BorrowRate types.Number `json:"borrowRate"`
}

// RatioResp holds information on ratios
type RatioResp struct {
	LongRatio      types.Number `json:"longRatio"`
	ShortRatio     types.Number `json:"shortRatio"`
	LongShortRatio types.Number `json:"longShortRatio"`
	Timestamp      types.Time   `json:"ts"`
}

// FundFlowResp holds information on fund flows
type FundFlowResp struct {
	WhaleBuyVolume    types.Number `json:"whaleBuyVolume"`
	DolphinBuyVolume  types.Number `json:"dolphinBuyVolume"`
	FishBuyVolume     types.Number `json:"fishBuyVolume"`
	WhaleSellVolume   types.Number `json:"whaleSellVolume"`
	DolphinSellVolume types.Number `json:"dolphinSellVolume"`
	FishSellVolume    types.Number `json:"fishSellVolume"`
	WhaleBuyRatio     types.Number `json:"whaleBuyRatio"`
	DolphinBuyRatio   types.Number `json:"dolphinBuyRatio"`
	FishBuyRatio      types.Number `json:"fishBuyRatio"`
	WhaleSellRatio    types.Number `json:"whaleSellRatio"`
	DolphinSellRatio  types.Number `json:"dolphinSellRatio"`
	FishSellRatio     types.Number `json:"fishSellRatio"`
}

// SymbolsResp holds information on supported symbols
type SymbolsResp struct {
	SpotList   []string `json:"spotList"`
	FutureList []string `json:"futureList"`
}

// WhaleFundFlowResp holds information on whale fund flows
type WhaleFundFlowResp struct {
	NetFlow   types.Number `json:"netFlow"`
	Timestamp types.Time   `json:"ts"`
}

// AccountRatioResp holds information on ratios
type AccountRatioResp struct {
	LongAccountRatio      types.Number `json:"longAccountRatio"`
	ShortAccountRatio     types.Number `json:"shortAccountRatio"`
	LongShortAccountRatio types.Number `json:"longShortAccountRatio"`
	Timestamp             types.Time   `json:"ts"`
}

// FailureList is a sub-struct holding information on failures
type FailureList struct {
	SubaccountName string `json:"subaAccountName"`
}

// SuccessList is a sub-struct holding information on successes
type SuccessList struct {
	SubaccountUID  string     `json:"subAccountUid"`
	SubaccountName string     `json:"subaAccountName"`
	Status         string     `json:"status"`
	PermList       []string   `json:"permList"`
	Label          string     `json:"label"`
	CreationTime   types.Time `json:"cTime"`
	UpdateTime     types.Time `json:"uTime"`
}

// CrVirSubResp contains information returned when creating virtual sub-accounts
type CrVirSubResp struct {
	FailureList []FailureList `json:"failureList"`
	SuccessList []SuccessList `json:"successList"`
}

// ResultWrapper wraps certain responses from the Bitget API for proper JSON decoding
type ResultWrapper struct {
	Result any `json:"result"`
}

// SuccessBool is a type used to unmarshal strings that are either "success" or "failure" into bools
type SuccessBool bool

// CrSubAccAPIKeyResp contains information returned when simultaneously creating a sub-account and an API key
type CrSubAccAPIKeyResp struct {
	SubaccountUID    string   `json:"subAccountUid"`
	SubaccountName   string   `json:"subAccountName"`
	Label            string   `json:"label"`
	SubaccountAPIKey string   `json:"subAccountApiKey"`
	SecretKey        string   `json:"secretKey"`
	PermList         []string `json:"permList"`
	IPList           []string `json:"ipList"`
}

// SubaccountList is a sub-struct holding information on sub-accounts
type SubaccountList struct {
	SubaccountUID  string     `json:"subAccountUid"`
	SubaccountName string     `json:"subAccountName"`
	Label          string     `json:"label"`
	Status         string     `json:"status"`
	PermList       []string   `json:"permList"`
	CreationTime   types.Time `json:"cTime"`
	UpdateTime     types.Time `json:"uTime"`
	// Documentation mentions "accountType" and "bindingTime" fields, but they aren't present in the response
}

// GetVirSubResp contains information on the user's virtual sub-accounts
type GetVirSubResp struct {
	SubaccountList []SubaccountList `json:"subAccountList"`
	EndID          int64            `json:"endId,string"`
}

// AlterAPIKeyResp contains information returned when creating or modifying an API key
type AlterAPIKeyResp struct {
	SubaccountUID    string   `json:"subAccountUid"`
	SubaccountAPIKey string   `json:"subAccountApiKey"`
	SecretKey        string   `json:"secretKey"`
	PermList         []string `json:"permList"`
	Label            string   `json:"label"`
	IPList           []string `json:"ipList"`
}

// GetAPIKeyResp contains information on the user's API keys
type GetAPIKeyResp struct {
	SubaccountUID    string   `json:"subAccountUid"`
	SubaccountAPIKey string   `json:"subAccountApiKey"`
	IPList           []string `json:"ipList"`
	PermList         []string `json:"permList"`
	Label            string   `json:"label"`
}

// FundingAssetsResp contains information on the user's funding assets
type FundingAssetsResp struct {
	Coin      currency.Code `json:"coin"`
	Available types.Number  `json:"available"`
	Frozen    types.Number  `json:"frozen"`
	USDTValue types.Number  `json:"usdtValue"`
}

// BotAccAssetsResp contains information on the user's bot account assets
type BotAccAssetsResp struct {
	Coin      currency.Code `json:"coin"`
	Available types.Number  `json:"available"`
	Equity    types.Number  `json:"equity"`
	Bonus     types.Number  `json:"bonus"`
	Frozen    types.Number  `json:"frozen"`
	USDTValue types.Number  `json:"usdtValue"`
}

// AssetOverviewResp contains information on the user's assets
type AssetOverviewResp struct {
	AccountType string       `json:"accountType"`
	USDTBalance types.Number `json:"usdtBalance"`
}

// ConvertCoinsResp contains information on the user's available currencies
type ConvertCoinsResp struct {
	Coin          currency.Code `json:"coin"`
	Available     types.Number  `json:"available"`
	MaximumAmount types.Number  `json:"maxAmount"`
	MinimumAmount types.Number  `json:"minAmount"`
}

// QuotedPriceResp contains information on a queried conversion
type QuotedPriceResp struct {
	FromCoin     currency.Code `json:"fromCoin"`
	FromCoinSize types.Number  `json:"fromCoinSize"`
	ConvertPrice types.Number  `json:"cnvtPrice"`
	ToCoin       currency.Code `json:"toCoin"`
	ToCoinSize   types.Number  `json:"toCoinSize"`
	TraceID      string        `json:"traceId"`
	Fee          types.Number  `json:"fee"`
}

// CommitConvResp contains information on a committed conversion
type CommitConvResp struct {
	ToCoin       currency.Code `json:"toCoin"`
	ToCoinSize   types.Number  `json:"toCoinSize"`
	ConvertPrice types.Number  `json:"cnvtPrice"`
	Timestamp    types.Time    `json:"ts"`
}

// DataList is a sub-struct holding information on the user's conversion history
type DataList struct {
	ID           int64         `json:"id,string"`
	Timestamp    types.Time    `json:"ts"`
	ConvertPrice types.Number  `json:"cnvtPrice"`
	Fee          types.Number  `json:"fee"`
	FromCoinSize types.Number  `json:"fromCoinSize"`
	FromCoin     currency.Code `json:"fromCoin"`
	ToCoinSize   types.Number  `json:"toCoinSize"`
	ToCoin       currency.Code `json:"toCoin"`
}

// ConvHistResp contains information on the user's conversion history
type ConvHistResp struct {
	DataList []DataList `json:"dataList"`
	EndID    int64      `json:"endId,string"`
}

// FeeAndRate is a sub-struct holding information on fees
type FeeAndRate struct {
	FeeRate types.Number `json:"feeRate"`
	Fee     types.Number `json:"fee"`
}

// BGBConvertCoinsResp contains information on the user's available currencies and conversions between those
// and BGB
type BGBConvertCoinsResp struct {
	Coin         currency.Code `json:"coin"`
	Available    types.Number  `json:"available"`
	BGBEstAmount types.Number  `json:"bgbEstAmount"`
	Precision    uint8         `json:"precision"`
	FeeDetail    []FeeAndRate  `json:"feeDetail"`
	CurrentTime  types.Time    `json:"cTime"`
}

// ConvertBGBResp contains information on a series of conversions between BGB and other currencies
type ConvertBGBResp struct {
	Coin    currency.Code `json:"coin"`
	OrderID int64         `json:"orderId,string"`
}

// FeeAndCoin is a sub-struct holding information on fees
type FeeAndCoin struct {
	FeeCoin currency.Code `json:"feeCoin"`
	Fee     types.Number  `json:"fee"`
}

// BGBConvHistResp contains information on the user's conversion history between BGB and other currencies
type BGBConvHistResp struct {
	OrderID       int64         `json:"orderId,string"`
	FromCoin      currency.Code `json:"fromCoin"`
	FromAmount    types.Number  `json:"fromAmount"`
	FromCoinPrice types.Number  `json:"fromCoinPrice"`
	ToCoin        currency.Code `json:"toCoin"`
	ToAmount      types.Number  `json:"toAmount"`
	ToCoinPrice   types.Number  `json:"toCoinPrice"`
	FeeDetail     []FeeAndCoin  `json:"feeDetail"`
	Status        SuccessBool   `json:"status"`
	CreationTime  types.Time    `json:"cTime"`
}

// ChainInfo is a sub-struct containing information on supported chains for a currency
type ChainInfo struct {
	Chain                 string       `json:"chain"`
	NeedTag               bool         `json:"needTag,string"`
	Withdrawable          bool         `json:"withdrawable,string"`
	Rechargeable          bool         `json:"rechargeable,string"`
	WithdrawFee           types.Number `json:"withdrawFee"`
	ExtraWithdrawFee      types.Number `json:"extraWithdrawFee"`
	DepositConfirm        uint16       `json:"depositConfirm,string"`
	WithdrawConfirm       uint16       `json:"withdrawConfirm,string"`
	MinimumDepositAmount  types.Number `json:"minDepositAmount"`
	MinimumWithdrawAmount types.Number `json:"minWithdrawAmount"`
	BrowserURL            string       `json:"browserUrl"`
	ContractAddress       string       `json:"contractAddress"`
	WithdrawStep          uint8        `json:"withdrawStep,string"`
	WithdrawMinimumScale  uint8        `json:"withdrawMinimumScale,string"`
	Congestion            string       `json:"congestion"`
}

// CoinInfoResp contains information on supported spot currencies
type CoinInfoResp struct {
	CoinID   uint32        `json:"coinId,string"`
	Coin     currency.Code `json:"coin"`
	Transfer bool          `json:"transfer,string"`
	Chains   []ChainInfo   `json:"chains"`
}

// SymbolInfoResp contains information on supported spot trading pairs
type SymbolInfoResp struct {
	Symbol              string        `json:"symbol"`
	BaseCoin            currency.Code `json:"baseCoin"`
	QuoteCoin           currency.Code `json:"quoteCoin"`
	MinimumTradeAmount  types.Number  `json:"minTradeAmount"`
	MaximumTradeAmount  types.Number  `json:"maxTradeAmount"`
	TakerFeeRate        types.Number  `json:"takerFeeRate"`
	MakerFeeRate        types.Number  `json:"makerFeeRate"`
	PricePrecision      uint8         `json:"pricePrecision,string"`
	QuantityPrecision   uint8         `json:"quantityPrecision,string"`
	QuotePrecision      uint8         `json:"quotePrecision,string"`
	MinimumTradeUSDT    types.Number  `json:"minTradeUSDT"`
	Status              string        `json:"status"`
	BuyLimitPriceRatio  types.Number  `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio types.Number  `json:"sellLimitPriceRatio"`
	AreaSymbol          YesNoBool     `json:"areaSymbol"`
	OrderQuantity       uint16        `json:"orderQuantity,string"`
	OpenTime            types.Time    `json:"openTime"`
}

// VIPFeeRateResp contains information on the different levels of VIP fee rates
type VIPFeeRateResp struct {
	Level        uint8        `json:"level,string"`
	DealAmount   types.Number `json:"dealAmount"`
	AssetAmount  types.Number `json:"assetAmount"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
	// 24-hour withdrawal limits
	BTCWithdrawAmount  types.Number `json:"btcWithdrawAmount"`
	USDTWithdrawAmount types.Number `json:"usdtWithdrawAmount"`
}

// InterestRateHistory contains information on the interest rate history
type InterestRateHistory struct {
	AnnualInterestRate types.Number `json:"annualInterestRate"`
	DailyInterestRate  types.Number `json:"dailyInterestRate"`
	Timestamp          types.Time   `json:"ts"`
}

// InterestRateResp contains information on the interest rate history
type InterestRateResp struct {
	Coin    currency.Code         `json:"coin"`
	History []InterestRateHistory `json:"historyInterestRateList"`
}

// ExchangeRateList is a sub-struct holding information on exchange rates
type ExchangeRateList struct {
	Tier          uint8        `json:"tier,string"`
	MinimumAmount types.Number `json:"minAmount"`
	MaximumAmount types.Number `json:"maxAmount"`
	ExchangeRate  types.Number `json:"exchangeRate"`
}

// ExchangeRateResp contains information on exchange rates
type ExchangeRateResp struct {
	Coin     currency.Code      `json:"coin"`
	RateList []ExchangeRateList `json:"exchangeRateList"`
}

// DiscountRateList is a sub-struct holding information on discount rates
type DiscountRateList struct {
	Tier          uint8        `json:"tier,string"`
	MinimumAmount types.Number `json:"minAmount"`
	MaximumAmount types.Number `json:"maxAmount"`
	DiscountRate  types.Number `json:"discountRate"`
}

// DiscountRateResp contains information on discount rates
type DiscountRateResp struct {
	Coin       currency.Code      `json:"coin"`
	UserLimit  uint64             `json:"userLimit,string"`
	TotalLimit uint64             `json:"totalLimit,string"`
	RateList   []DiscountRateList `json:"discountRateList"`
}

// TickerResp contains information on tickers
type TickerResp struct {
	Symbol       string       `json:"symbol"`
	High24H      types.Number `json:"high24h"`
	Open         types.Number `json:"open"`
	LastPrice    types.Number `json:"lastPr"`
	Low24H       types.Number `json:"low24h"`
	QuoteVolume  types.Number `json:"quoteVolume"`
	BaseVolume   types.Number `json:"baseVolume"`
	USDTVolume   types.Number `json:"usdtVolume"`
	BidPrice     types.Number `json:"bidPr"`
	AskPrice     types.Number `json:"askPr"`
	BidSize      types.Number `json:"bidSz"`
	AskSize      types.Number `json:"askSz"`
	OpenUTC      types.Number `json:"openUTC"`
	Timestamp    types.Time   `json:"ts"`
	ChangeUTC24H types.Number `json:"changeUTC24h"`
	Change24H    types.Number `json:"change24h"`
}

// DepthResp contains information on orderbook bids and asks, and any merging of orders done to them
type DepthResp struct {
	Asks           [][2]float64 `json:"asks"`
	Bids           [][2]float64 `json:"bids"`
	Precision      string       `json:"precision"`
	Scale          types.Number `json:"scale"`
	IsMaxPrecision YesNoBool    `json:"isMaxPrecision"`
	Timestamp      types.Time   `json:"ts"`
}

// OrderbookResp contains information on orderbook bids and asks
type OrderbookResp struct {
	Asks      [][2]types.Number `json:"asks"`
	Bids      [][2]types.Number `json:"bids"`
	Timestamp types.Time        `json:"ts"`
}

// OneSpotCandle contains a single candle
type OneSpotCandle struct {
	Timestamp   types.Time
	Open        types.Number
	High        types.Number
	Low         types.Number
	Close       types.Number
	BaseVolume  types.Number
	QuoteVolume types.Number
	USDTVolume  types.Number
}

// UnmarshalJSON deserializes kline data from a JSON array into OneSpotCandle fields
func (c *OneSpotCandle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[8]any{&c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close, &c.BaseVolume, &c.QuoteVolume, &c.USDTVolume})
}

// OneFuturesCandle contains a single candle
type OneFuturesCandle struct {
	Timestamp   types.Time
	Entry       types.Number
	High        types.Number
	Low         types.Number
	Exit        types.Number
	BaseVolume  types.Number
	QuoteVolume types.Number
}

// UnmarshalJSON deserializes kline data from a JSON array into OneFuturesCandle fields
func (c *OneFuturesCandle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&c.Timestamp, &c.Entry, &c.High, &c.Low, &c.Exit, &c.BaseVolume, &c.QuoteVolume})
}

// MarketFillsResp contains information on a batch of trades
type MarketFillsResp struct {
	Symbol    string       `json:"symbol"`
	TradeID   int64        `json:"tradeId,string"`
	Side      string       `json:"side"`
	Price     types.Number `json:"price"`
	Size      types.Number `json:"size"`
	Timestamp types.Time   `json:"ts"`
}

// PlaceSpotOrderStruct contains information on an order to be placed
type PlaceSpotOrderStruct struct {
	// Symbol needs to be included, despite it being absent in the documentation, or the exchange will return an error
	Pair                   currency.Pair `json:"symbol"`
	Side                   string        `json:"side"`
	OrderType              string        `json:"orderType"`
	Strategy               string        `json:"force"`
	Price                  types.Number  `json:"price"`
	Size                   types.Number  `json:"size"`
	ClientOrderID          string        `json:"clientOid,omitempty"`
	STPMode                string        `json:"stpMode"`
	PresetTakeProfitPrice  types.Number  `json:"presetTakeProfitPrice,omitempty"`
	ExecuteTakeProfitPrice types.Number  `json:"executeTakeProfitPrice,omitempty"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice,omitempty"`
	ExecuteStopLossPrice   types.Number  `json:"executeStopLossPrice,omitempty"`
}

// CancelSpotOrderStruct contains information on an order to be cancelled
type CancelSpotOrderStruct struct {
	Pair          currency.Pair `json:"symbol"`
	OrderID       int64         `json:"orderId,string,omitempty"`
	ClientOrderID string        `json:"clientOid,omitempty"`
}

// EmptyInt is a type used to unmarshal empty string into 0, and numbers encoded as strings into int64
type EmptyInt int64

// OrderIDAndError is a sub-struct containing information on an order ID and any errors associated with it
type OrderIDAndError struct {
	OrderID       EmptyInt `json:"orderId"`
	ClientOrderID string   `json:"clientOid"`
	ErrorCode     int64    `json:"errorCode,string"`
	ErrorMessage  string   `json:"errorMsg"`
}

// BatchOrderResp contains information on the success or failure of a batch of orders to place or cancel
type BatchOrderResp struct {
	SuccessList []OrderIDStruct   `json:"successList"`
	FailureList []OrderIDAndError `json:"failureList"`
}

// OrderIDStruct contains order IDs
type OrderIDStruct struct {
	OrderID       EmptyInt `json:"orderId,omitempty"`
	ClientOrderID string   `json:"clientOid,omitempty"`
}

// OrderDetailTemp contains information on an order in a partially-unmarshalled state
type OrderDetailTemp struct {
	UserID           uint64          `json:"userId,string"`
	Symbol           string          `json:"symbol"`
	OrderID          EmptyInt        `json:"orderId"`
	ClientOrderID    string          `json:"clientOid"`
	Price            types.Number    `json:"price"`
	Size             types.Number    `json:"size"`
	OrderType        string          `json:"orderType"`
	Side             string          `json:"side"`
	Status           string          `json:"status"`
	PriceAverage     types.Number    `json:"priceAvg"`
	BaseVolume       types.Number    `json:"baseVolume"`
	QuoteVolume      types.Number    `json:"quoteVolume"`
	EnterPointSource string          `json:"enterPointSource"`
	CreationTime     types.Time      `json:"cTime"`
	UpdateTime       types.Time      `json:"uTime"`
	OrderSource      string          `json:"orderSource"`
	FeeDetailTemp    json.RawMessage `json:"feeDetail"`
	TriggerPrice     types.Number    `json:"triggerPrice"`
	TPSLType         string          `json:"tpslType"`
	CancelReason     string          `json:"cancelReason"`
}

// FeeDetail contains information on fees
type FeeDetail struct {
	AmountCoupons     float64       `json:"c"`
	AmountDeductedBGB float64       `json:"d"`
	AmountRemaining   float64       `json:"r"`
	AmountTotal       float64       `json:"t"`
	Deduction         bool          `json:"deduction"`
	FeeCoinCode       currency.Code `json:"feeCoinCode"`
	TotalFee          float64       `json:"totalFee"`
	TotalDeductionFee float64       `json:"totalDeductionFee"`
}

// FeeDetailStore is a map of fee details for better unmarshalling
type FeeDetailStore map[string]FeeDetail

// SpotOrderDetailData contains information on an order for better unmarshalling
type SpotOrderDetailData struct {
	UserID           uint64
	Symbol           string
	OrderID          EmptyInt
	ClientOrderID    string
	Price            float64
	Size             float64
	OrderType        string
	Side             string
	Status           string
	PriceAverage     float64
	BaseVolume       float64
	QuoteVolume      float64
	EnterPointSource string
	CreationTime     types.Time
	UpdateTime       types.Time
	OrderSource      string
	FeeDetail        FeeDetailStore
	// This struct is used by two endpoints, check both before deleting fields
	TriggerPrice float64
	TPSLType     string
	CancelReason string
}

// UnfilledOrdersResp contains information on the user's unfilled orders
type UnfilledOrdersResp struct {
	UserID                 uint64       `json:"userId,string"`
	Symbol                 string       `json:"symbol"`
	OrderID                EmptyInt     `json:"orderId"`
	ClientOrderID          string       `json:"clientOid"`
	PriceAverage           types.Number `json:"priceAvg"`
	Size                   types.Number `json:"size"`
	OrderType              string       `json:"orderType"`
	Side                   string       `json:"side"`
	Status                 string       `json:"status"`
	BasePrice              types.Number `json:"basePrice"`
	BaseVolume             types.Number `json:"baseVolume"`
	QuoteVolume            types.Number `json:"quoteVolume"`
	EnterPointSource       string       `json:"enterPointSource"`
	OrderSource            string       `json:"orderSource"`
	PresetTakeProfitPrice  types.Number `json:"presetTakeProfitPrice"`
	ExecuteTakeProfitPrice types.Number `json:"executeTakeProfitPrice"`
	PresetStopLossPrice    types.Number `json:"presetStopLossPrice"`
	ExecuteStopLossPrice   types.Number `json:"executeStopLossPrice"`
	CreationTime           types.Time   `json:"cTime"`
	UpdateTime             types.Time   `json:"uTime"`
	TriggerType            string       `json:"triggerType"`
	TPSLType               string       `json:"tpslType"`
}

// AbridgedFeeDetail contains some information on fees
type AbridgedFeeDetail struct {
	Deduction         YesNoBool     `json:"deduction"`
	FeeCoin           currency.Code `json:"feeCoin"`
	TotalDeductionFee types.Number  `json:"totalDeductionFee"`
	TotalFee          types.Number  `json:"totalFee"`
}

// SpotFillsResp contains information on the user's fulfilled orders
type SpotFillsResp struct {
	UserID       uint64            `json:"userId,string"`
	Symbol       string            `json:"symbol"`
	OrderID      EmptyInt          `json:"orderId"`
	TradeID      int64             `json:"tradeId,string"`
	OrderType    string            `json:"orderType"`
	Side         string            `json:"side"`
	PriceAverage types.Number      `json:"priceAvg"`
	Size         types.Number      `json:"size"`
	Amount       types.Number      `json:"amount"`
	FeeDetail    AbridgedFeeDetail `json:"feeDetail"`
	TradeScope   string            `json:"tradeScope"`
	CreationTime types.Time        `json:"cTime"`
	UpdateTime   types.Time        `json:"uTime"`
}

// CancelAndPlaceResp contains information on the success or failure of a replaced order
type CancelAndPlaceResp struct {
	OrderID       EmptyInt    `json:"orderId"`
	ClientOrderID string      `json:"clientOid"`
	Success       SuccessBool `json:"success"`
	Message       string      `json:"msg"`
}

// ReplaceSpotOrderStruct contains information on an order to be replaced
type ReplaceSpotOrderStruct struct {
	Pair                   currency.Pair `json:"symbol"`
	Price                  types.Number  `json:"price"`
	Amount                 types.Number  `json:"size"`
	OldClientOrderID       string        `json:"clientOid,omitempty"`
	OrderID                int64         `json:"orderId,string,omitempty"`
	NewClientOrderID       string        `json:"newClientOid,omitempty"`
	PresetTakeProfitPrice  types.Number  `json:"presetTakeProfitPrice,omitempty"`
	ExecuteTakeProfitPrice types.Number  `json:"executeTakeProfitPrice,omitempty"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice,omitempty"`
	ExecuteStopLossPrice   types.Number  `json:"executeStopLossPrice,omitempty"`
}

// PlanSpotOrder is a sub-struct that contains information on a planned order
type PlanSpotOrder struct {
	OrderID          int64        `json:"orderId,string"`
	ClientOrderID    string       `json:"clientOid"`
	Symbol           string       `json:"symbol"`
	TriggerPrice     types.Number `json:"triggerPrice"`
	OrderType        string       `json:"orderType"`
	ExecutePrice     types.Number `json:"executePrice"`
	PlanType         string       `json:"planType"`
	Size             types.Number `json:"size"`
	Status           string       `json:"status"`
	Side             string       `json:"side"`
	TriggerType      string       `json:"triggerType"`
	EnterPointSource string       `json:"enterPointSource"`
	CreationTime     types.Time   `json:"cTime"`
	UpdateTime       types.Time   `json:"uTime"`
}

// PlanSpotOrderResp contains information on plan orders
type PlanSpotOrderResp struct {
	NextFlag   bool            `json:"nextFlag"`
	IDLessThan EmptyInt        `json:"idLessThan"`
	OrderList  []PlanSpotOrder `json:"orderList"`
}

// SubOrderResp contains information on sub-orders
type SubOrderResp struct {
	OrderID int64        `json:"orderId,string"`
	Price   types.Number `json:"price"`
	Type    string       `json:"type"`
	Status  string       `json:"status"`
}

// AccountInfoResp contains information on the user's account
type AccountInfoResp struct {
	UserID       uint64     `json:"userId,string"`
	InviterID    int64      `json:"inviterId,string"`
	ChannelCode  string     `json:"channelCode"`
	Channel      string     `json:"channel"`
	IPs          string     `json:"ips"`
	Authorities  []string   `json:"authorities"`
	ParentID     int64      `json:"parentId"`
	TraderType   string     `json:"traderType"`
	RegisterTime types.Time `json:"regisTime"`
}

// AssetData contains information on the amount of an asset an account owns
type AssetData struct {
	Coin           currency.Code `json:"coin"`
	Available      types.Number  `json:"available"`
	Frozen         types.Number  `json:"frozen"`
	Locked         types.Number  `json:"locked"`
	LimitAvailable types.Number  `json:"limitAvailable"`
	UpdateTime     types.Time    `json:"uTime"`
}

// SubaccountAssetsResp contains information on assets in a user's sub-accounts
type SubaccountAssetsResp struct {
	UserID     uint64      `json:"userId"`
	AssetsList []AssetData `json:"assetsList"`
}

// SpotAccBillResp contains information on the user's billing history
type SpotAccBillResp struct {
	CreationTime types.Time    `json:"cTime"`
	Coin         currency.Code `json:"coin"`
	GroupType    string        `json:"groupType"`
	BusinessType string        `json:"businessType"`
	Size         types.Number  `json:"size"`
	Balance      types.Number  `json:"balance"`
	Fees         types.Number  `json:"fees"`
	BillID       int64         `json:"billId,string"`
}

// TransferResp contains information on an asset transfer
type TransferResp struct {
	TransferID    int64  `json:"transferId,string"`
	ClientOrderID string `json:"clientOid"`
}

// SubaccTfrRecResp contains detailed information on asset transfers between sub-accounts
type SubaccTfrRecResp struct {
	Coin          currency.Code `json:"coin"`
	Status        string        `json:"status"`
	ToType        string        `json:"toType"`
	FromType      string        `json:"fromType"`
	Size          types.Number  `json:"size"`
	Timestamp     types.Time    `json:"ts"`
	ClientOrderID string        `json:"clientOid"`
	TransferID    int64         `json:"transferId,string"`
	FromUserID    uint64        `json:"fromUserId,string"`
	ToUserID      uint64        `json:"toUserId,string"`
}

// TransferRecResp contains detailed information on asset transfers
type TransferRecResp struct {
	Coin          currency.Code `json:"coin"`
	Status        string        `json:"status"`
	ToType        string        `json:"toType"`
	ToSymbol      string        `json:"toSymbol"`
	FromType      string        `json:"fromType"`
	FromSymbol    string        `json:"fromSymbol"`
	Size          types.Number  `json:"size"`
	Timestamp     types.Time    `json:"ts"`
	ClientOrderID string        `json:"clientOid"`
	TransferID    int64         `json:"transferId,string"`
}

// DepositAddressResp contains information on a deposit address
type DepositAddressResp struct {
	Address string        `json:"address"`
	Chain   string        `json:"chain"`
	Coin    currency.Code `json:"coin"`
	Tag     string        `json:"tag"`
	URL     string        `json:"url"`
}

// OnOffBool is a type used to unmarshal strings that are either "on" or "off" into bools
type OnOffBool bool

// SubaccDepRecResp contains detailed information on deposits to sub-accounts
type SubaccDepRecResp struct {
	OrderID       int64         `json:"orderId,string"`
	TradeID       int64         `json:"tradeId,string"`
	Coin          currency.Code `json:"coin"`
	ClientOrderID string        `json:"clientOid"`
	Size          types.Number  `json:"size"`
	Status        string        `json:"status"`
	FromAddress   string        `json:"fromAddress"`
	ToAddress     string        `json:"toAddress"`
	Chain         string        `json:"chain"`
	Confirm       uint32        `json:"confirm,string"`
	Destination   string        `json:"dest"`
	Tag           string        `json:"tag"`
	CreationTime  types.Time    `json:"cTime"`
	UpdateTime    types.Time    `json:"uTime"`
}

// WithdrawRecordsResp contains detailed information on withdrawals
type WithdrawRecordsResp struct {
	OrderID       int64         `json:"orderId,string"`
	TradeID       int64         `json:"tradeId,string"`
	Coin          currency.Code `json:"coin"`
	ClientOrderID string        `json:"clientOid"`
	OrderType     string        `json:"type"`
	Destination   string        `json:"dest"`
	Size          types.Number  `json:"size"`
	Fee           types.Number  `json:"fee"`
	Status        string        `json:"status"`
	FromAddress   string        `json:"fromAddress"`
	ToAddress     string        `json:"toAddress"`
	Chain         string        `json:"chain"`
	Confirm       uint32        `json:"confirm,string"`
	Tag           string        `json:"tag"`
	CreationTime  types.Time    `json:"cTime"`
	UpdateTime    types.Time    `json:"uTime"`
}

// CryptoDepRecResp contains detailed information on cryptocurrency deposits
type CryptoDepRecResp struct {
	OrderID      int64         `json:"orderId,string"`
	TradeID      int64         `json:"tradeId,string"`
	Coin         currency.Code `json:"coin"`
	OrderType    string        `json:"type"`
	Size         types.Number  `json:"size"`
	Status       string        `json:"status"`
	FromAddress  string        `json:"fromAddress"`
	ToAddress    string        `json:"toAddress"`
	Chain        string        `json:"chain"`
	Destination  string        `json:"dest"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
}

// FutureTickerResp contains information on a futures ticker
type FutureTickerResp struct {
	Symbol            string       `json:"symbol"`
	LastPrice         types.Number `json:"lastPr"`
	AskPrice          types.Number `json:"askPr"`
	BidPrice          types.Number `json:"bidPr"`
	BidSize           types.Number `json:"bidSz"`
	AskSize           types.Number `json:"askSz"`
	High24H           types.Number `json:"high24h"`
	Low24H            types.Number `json:"low24h"`
	Timestamp         types.Time   `json:"ts"`
	Change24H         types.Number `json:"change24h"`
	BaseVolume        types.Number `json:"baseVolume"`
	QuoteVolume       types.Number `json:"quoteVolume"`
	USDTVolume        types.Number `json:"usdtVolume"`
	OpenUTC           types.Number `json:"openUtc"`
	ChangeUTC24H      types.Number `json:"changeUtc24h"`
	IndexPrice        types.Number `json:"indexPrice"`
	FundingRate       types.Number `json:"fundingRate"`
	HoldingAmount     types.Number `json:"holdingAmount"`
	DeliveryStartTime types.Time   `json:"deliveryStartTime"`
	DeliveryTime      types.Time   `json:"deliveryTime"`
	DeliveryStatus    string       `json:"deliveryStatus"`
	Open24H           types.Number `json:"open24h"`
	MarkPrice         types.Number `json:"markPrice"`
}

// CallMode represents the call mode for the futures candlestick endpoints
type CallMode uint8

const (
	// CallModeNormal represents the normal call mode
	CallModeNormal CallMode = iota
	// CallModeHistory represents the history call mode
	CallModeHistory
	// CallModeIndex represents the historical index call mode
	CallModeIndex
	// CallModeMark represents the historical mark call mode
	CallModeMark
)

// OpenInterestList is a sub-struct containing information on open interest
type OpenInterestList struct {
	Symbol string       `json:"symbol"`
	Size   types.Number `json:"size"`
}

// OpenPositionsResp contains information on open positions
type OpenPositionsResp struct {
	OpenInterestList []OpenInterestList `json:"openInterestList"`
	Timestamp        types.Time         `json:"ts"`
}

// FundingTimeResp contains information on funding times
type FundingTimeResp struct {
	Symbol          string     `json:"symbol"`
	NextFundingTime types.Time `json:"nextFundingTime"`
	RatePeriod      uint16     `json:"ratePeriod,string"`
}

// FuturesPriceResp contains information on futures prices
type FuturesPriceResp struct {
	Symbol     string       `json:"symbol"`
	Price      types.Number `json:"price"`
	IndexPrice types.Number `json:"indexPrice"`
	MarkPrice  types.Number `json:"markPrice"`
	Timestamp  types.Time   `json:"ts"`
}

// FundingHistoryResp contains information on funding history
type FundingHistoryResp struct {
	Symbol      string       `json:"symbol"`
	FundingRate types.Number `json:"fundingRate"`
	FundingTime types.Time   `json:"fundingTime"`
}

// FundingCurrentResp contains information on current funding rates
type FundingCurrentResp struct {
	Symbol      string       `json:"symbol"`
	FundingRate types.Number `json:"fundingRate"`
}

// ContractConfigResp contains information on contract details
type ContractConfigResp struct {
	Symbol                    string        `json:"symbol"`
	BaseCoin                  currency.Code `json:"baseCoin"`
	QuoteCoin                 currency.Code `json:"quoteCoin"`
	BuyLimitPriceRatio        types.Number  `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio       types.Number  `json:"sellLimitPriceRatio"`
	FeeRateUpRatio            types.Number  `json:"feeRateUpRatio"`
	MakerFeeRate              types.Number  `json:"makerFeeRate"`
	TakerFeeRate              types.Number  `json:"takerFeeRate"`
	OpenCostUpRatio           types.Number  `json:"openCostUpRatio"`
	SupportMarginCoins        []string      `json:"supportMarginCoins"`
	MinimumTradeNumber        types.Number  `json:"minTradeNum"`
	PriceEndStep              types.Number  `json:"priceEndStep"`
	VolumePlace               types.Number  `json:"volumePlace"`
	PricePlace                types.Number  `json:"pricePlace"`
	SizeMultiplier            types.Number  `json:"sizeMultiplier"`
	SymbolType                string        `json:"symbolType"`
	MinimumTradeUSDT          types.Number  `json:"minTradeUSDT"`
	MaximumSymbolOrderNumber  int64         `json:"maxSymbolOrderNum,string"`
	MaximumProductOrderNumber int64         `json:"maxProductOrderNum,string"`
	MaximumPositionNumber     int64         `json:"maxPositionNum,string"`
	SymbolStatus              string        `json:"symbolStatus"`
	OffTime                   int64         `json:"offTime,string"`
	LimitOpenTime             int64         `json:"limitOpenTime,string"`
	DeliveryTime              types.Time    `json:"deliveryTime"`
	DeliveryStartTime         types.Time    `json:"deliveryStartTime"`
	DeliveryPeriod            string        `json:"deliveryPeriod"`
	LaunchTime                types.Time    `json:"launchTime"`
	FundInterval              EmptyInt      `json:"fundInterval"`
	MinimumLeverage           types.Number  `json:"minLever"`
	MaximumLeverage           types.Number  `json:"maxLever"`
	PosLimit                  types.Number  `json:"posLimit"`
	MaintainTime              types.Time    `json:"maintainTime"`
	OpenTime                  types.Time    `json:"openTime"`
}

// OneAccResp contains information on a single account
type OneAccResp struct {
	MarginCoin                   currency.Code `json:"marginCoin"`
	Locked                       types.Number  `json:"locked"`
	Available                    types.Number  `json:"available"`
	CrossedMaximumAvailable      types.Number  `json:"crossedMaxAvailable"`
	IsolatedMaximumAvailable     types.Number  `json:"isolatedMaxAvailable"`
	MaximumTransferOut           types.Number  `json:"maxTransferOut"`
	AccountEquity                types.Number  `json:"accountEquity"`
	USDTEquity                   types.Number  `json:"usdtEquity"`
	BTCEquity                    types.Number  `json:"btcEquity"`
	CrossedRiskRate              types.Number  `json:"crossedRiskRate"`
	CrossedMarginleverage        float64       `json:"crossedMarginleverage"`
	IsolatedLongLeverage         float64       `json:"isolatedLongLever"`
	IsolatedShortLeverage        float64       `json:"isolatedShortLever"`
	MarginMode                   string        `json:"marginMode"`
	PositionMode                 string        `json:"posMode"`
	UnrealizedProfitLoss         types.Number  `json:"unrealizedPL"`
	Coupon                       types.Number  `json:"coupon"`
	CrossedUnrealizedProfitLoss  types.Number  `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedProfitLoss types.Number  `json:"isolatedUnrealizedPL"`
	AssetMode                    string        `json:"assetMode"`
	// The following fields are not in the documentation, but are still returned
	Grant          types.Number `json:"grant"`
	IsolatedMargin types.Number `json:"isolatedMargin"`
	CrossedMargin  types.Number `json:"crossedMargin"`
}

// FutureAccDetails contains information on a user's futures account
type FutureAccDetails struct {
	MarginCoin                   currency.Code `json:"marginCoin"`
	Locked                       types.Number  `json:"locked"`
	Available                    types.Number  `json:"available"`
	CrossedMaxAvailable          types.Number  `json:"crossedMaxAvailable"`
	IsolatedMaxAvailable         types.Number  `json:"isolatedMaxAvailable"`
	MaximumTransferOut           types.Number  `json:"maxTransferOut"`
	AccountEquity                types.Number  `json:"accountEquity"`
	USDTEquity                   types.Number  `json:"usdtEquity"`
	BTCEquity                    types.Number  `json:"btcEquity"`
	CrossedRiskRate              types.Number  `json:"crossedRiskRate"`
	UnrealizedProfitLoss         types.Number  `json:"unrealizedPL"`
	Coupon                       types.Number  `json:"coupon"`
	CrossedUnrealizedProfitLoss  types.Number  `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedProfitLoss types.Number  `json:"isolatedUnrealizedPL"`
	AssetMode                    string        `json:"assetMode"`
	IsolatedMargin               types.Number  `json:"isolatedMargin"`
	CrossedMargin                types.Number  `json:"crossedMargin"`
	// The following field is not in the documentation, but is still returned
	Grant types.Number `json:"grant"`
}

// FutureSubaccDetails contains information on a futures-related sub-account
type FutureSubaccDetails struct {
	MarginCoin           currency.Code `json:"marginCoin"`
	Locked               types.Number  `json:"locked"`
	Available            types.Number  `json:"available"`
	CrossedMaxAvailable  types.Number  `json:"crossedMaxAvailable"`
	IsolatedMaxAvailable types.Number  `json:"isolatedMaxAvailable"`
	MaximumTransferOut   types.Number  `json:"maxTransferOut"`
	AccountEquity        types.Number  `json:"accountEquity"`
	USDTEquity           types.Number  `json:"usdtEquity"`
	BTCEquity            types.Number  `json:"btcEquity"`
	UnrealizedProfitLoss types.Number  `json:"unrealizedPL"`
	Coupon               types.Number  `json:"coupon"`
	// The following fields are not in the documentation, but are still returned
	CrossedUnrealizedProfitLoss  types.Number `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedProfitLoss types.Number `json:"isolatedUnrealizedPL"`
	Grant                        types.Number `json:"grant"`
	AssetMode                    string       `json:"assetMode"`
	IsolatedMargin               types.Number `json:"isolatedMargin"`
	CrossedMargin                types.Number `json:"crossedMargin"`
}

// SubaccountFuturesResp contains information on futures details of a user's sub-accounts
type SubaccountFuturesResp struct {
	UserID    uint64                `json:"userId"`
	AssetList []FutureSubaccDetails `json:"assetList"`
}

// InterestList is a sub-struct containing information on interest
type InterestList struct {
	Coin              currency.Code `json:"coin"`
	Liability         types.Number  `json:"liability"`
	InterestFreeLimit types.Number  `json:"interestFreeLimit"`
	InterestLimit     types.Number  `json:"interestLimit"`
	HourInterestRate  types.Number  `json:"hourInterestRate"`
	Interest          types.Number  `json:"interest"`
	CreationTime      types.Time    `json:"cTime"`
}

// USDTInterestHistory contains information on USDT interest history
type USDTInterestHistory struct {
	NextSettleTime types.Time     `json:"nextSettleTime"`
	BorrowAmount   types.Number   `json:"borrowAmount"`
	BorrowLimit    types.Number   `json:"borrowLimit"`
	InterestList   []InterestList `json:"interestList"`
	EndID          int64          `json:"endId,string"`
}

// ChangeLeverageResp contains information on the leverage of a position
type ChangeLeverageResp struct {
	Symbol              string        `json:"symbol"`
	MarginCoin          currency.Code `json:"marginCoin"`
	LongLeverage        types.Number  `json:"longLeverage"`
	ShortLeverage       types.Number  `json:"shortLeverage"`
	CrossMarginLeverage types.Number  `json:"crossMarginLeverage"`
	MarginMode          string        `json:"marginMode"`
}

// ChangeMarginModeResp contains information on the leverage of a position
type ChangeMarginModeResp struct {
	Symbol        string        `json:"symbol"`
	MarginCoin    currency.Code `json:"marginCoin"`
	LongLeverage  types.Number  `json:"longLeverage"`
	ShortLeverage types.Number  `json:"shortLeverage"`
	MarginMode    string        `json:"marginMode"`
}

// FutureBills contains information on futures billing history
type FutureBills struct {
	BillID       int64         `json:"billId,string"`
	Symbol       string        `json:"symbol"`
	Amount       types.Number  `json:"amount"`
	Fee          types.Number  `json:"fee"`
	FeeByCoupon  types.Number  `json:"feeByCoupon"`
	BusinessType string        `json:"businessType"`
	Coin         currency.Code `json:"coin"`
	Balance      types.Number  `json:"balance"`
	CreationTime types.Time    `json:"cTime"`
}

// FutureAccBillResp contains information on futures billing history
type FutureAccBillResp struct {
	Bills []FutureBills `json:"bills"`
	EndID int64         `json:"endId,string"`
}

// PositionTierResp contains information on position configurations
type PositionTierResp struct {
	Symbol         string       `json:"symbol"`
	Level          uint8        `json:"level,string"`
	StartUnit      types.Number `json:"startUnit"`
	EndUnit        types.Number `json:"endUnit"`
	Leverage       types.Number `json:"leverage"`
	KeepMarginRate types.Number `json:"keepMarginRate"`
}

// SinglePositionResp contains information on positions
type SinglePositionResp struct {
	MarginCoin           currency.Code `json:"marginCoin"`
	Symbol               string        `json:"symbol"`
	HoldSide             string        `json:"holdSide"`
	OpenDelegateSize     types.Number  `json:"openDelegateSize"`
	MarginSize           types.Number  `json:"marginSize"`
	Available            types.Number  `json:"available"`
	Locked               types.Number  `json:"locked"`
	Total                types.Number  `json:"total"`
	Leverage             types.Number  `json:"leverage"`
	AchievedProfits      types.Number  `json:"achievedProfits"`
	OpenPriceAverage     types.Number  `json:"openPriceAvg"`
	MarginMode           string        `json:"marginMode"`
	PositionMode         string        `json:"posMode"`
	UnrealizedProfitLoss types.Number  `json:"unrealizedPL"`
	LiquidationPrice     types.Number  `json:"liquidationPrice"`
	KeepMarginRate       types.Number  `json:"keepMarginRate"`
	MarkPrice            types.Number  `json:"markPrice"`
	MarginRatio          types.Number  `json:"marginRatio"`
	BreakEvenPrice       types.Number  `json:"breakEvenPrice"`
	TotalFee             types.Number  `json:"totalFee"`
	DeductedFee          types.Number  `json:"deductedFee"`
	CreationTime         types.Time    `json:"cTime"`
	AssetMode            string        `json:"assetMode"`
	UpdateTime           types.Time    `json:"uTime"`
	AutoMargin           OnOffBool     `json:"autoMargin"`
}

// AllPositionResp contains information on positions
type AllPositionResp struct {
	MarginCoin           currency.Code `json:"marginCoin"`
	Symbol               string        `json:"symbol"`
	HoldSide             string        `json:"holdSide"`
	OpenDelegateSize     types.Number  `json:"openDelegateSize"`
	MarginSize           types.Number  `json:"marginSize"`
	Available            types.Number  `json:"available"`
	Locked               types.Number  `json:"locked"`
	Total                types.Number  `json:"total"`
	Leverage             types.Number  `json:"leverage"`
	AchievedProfits      types.Number  `json:"achievedProfits"`
	OpenPriceAverage     types.Number  `json:"openPriceAvg"`
	MarginMode           string        `json:"marginMode"`
	PositionMode         string        `json:"posMode"`
	UnrealizedProfitLoss types.Number  `json:"unrealizedPL"`
	LiquidationPrice     types.Number  `json:"liquidationPrice"`
	KeepMarginRate       types.Number  `json:"keepMarginRate"`
	MarkPrice            types.Number  `json:"markPrice"`
	MarginRatio          types.Number  `json:"marginRatio"`
	BreakEvenPrice       types.Number  `json:"breakEvenPrice"`
	TotalFee             types.Number  `json:"totalFee"`
	TakeProfit           types.Number  `json:"takeProfit"`
	StopLoss             types.Number  `json:"stopLoss"`
	TakeProfitID         int64         `json:"takeProfitId,string"`
	StopLossID           int64         `json:"stopLossId,string"`
	DeductedFee          types.Number  `json:"deductedFee"`
	CreationTime         types.Time    `json:"cTime"`
	AssetMode            string        `json:"assetMode"`
	UpdateTime           types.Time    `json:"uTime"`
}

// HistPositions is a sub-struct containing information on historical positions
type HistPositions struct {
	PositionID         int64         `json:"positionId,string"`
	MarginCoin         currency.Code `json:"marginCoin"`
	Symbol             string        `json:"symbol"`
	HoldSide           string        `json:"holdSide"`
	OpenAveragePrice   types.Number  `json:"openAvgPrice"`
	CloseAveragePrice  types.Number  `json:"closeAvgPrice"`
	MarginMode         string        `json:"marginMode"`
	OpenTotalPosition  types.Number  `json:"openTotalPos"`
	CloseTotalPosition types.Number  `json:"closeTotalPos"`
	ProfitAndLoss      types.Number  `json:"pnl"`
	NetProfit          types.Number  `json:"netProfit"`
	TotalFunding       types.Number  `json:"totalFunding"`
	OpenFee            types.Number  `json:"openFee"`
	CloseFee           types.Number  `json:"closeFee"`
	UpdateTime         types.Time    `json:"uTime"`
	CreationTime       types.Time    `json:"cTime"`
}

// HistPositionResp contains information on historical positions
type HistPositionResp struct {
	List  []HistPositions `json:"list"`
	EndID int64           `json:"endId,string"`
}

// PlaceFuturesOrderStruct contains information on an order to be placed
type PlaceFuturesOrderStruct struct {
	Size            types.Number `json:"size"`
	Price           types.Number `json:"price"`
	Side            string       `json:"side"`
	TradeSide       string       `json:"tradeSide"`
	OrderType       string       `json:"orderType"`
	Strategy        string       `json:"force"`
	ClientOID       string       `json:"clientOId,omitempty"`
	ReduceOnly      YesNoBool    `json:"reduceOnly"`
	TakeProfitValue types.Number `json:"presetStopSurplusPrice,omitempty"`
	StopLossValue   types.Number `json:"presetStopLossPrice,omitempty"`
	STPMode         string       `json:"stpMode,omitempty"`
}

// FuturesOrderDetailResp contains information on a futures order
type FuturesOrderDetailResp struct {
	Symbol                 string        `json:"symbol"`
	Size                   types.Number  `json:"size"`
	OrderID                EmptyInt      `json:"orderId"`
	ClientOrderID          string        `json:"clientOid"`
	BaseVolume             types.Number  `json:"baseVolume"`
	PriceAverage           types.Number  `json:"priceAvg"`
	Fee                    types.Number  `json:"fee"`
	Price                  types.Number  `json:"price"`
	State                  string        `json:"state"`
	Side                   string        `json:"side"`
	Force                  string        `json:"force"`
	TotalProfits           types.Number  `json:"totalProfits"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             currency.Code `json:"marginCoin"`
	PresetStopSurplusPrice types.Number  `json:"presetStopSurplusPrice"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice"`
	QuoteVolume            types.Number  `json:"quoteVolume"`
	OrderType              string        `json:"orderType"`
	Leverage               types.Number  `json:"leverage"`
	MarginMode             string        `json:"marginMode"`
	ReduceOnly             YesNoBool     `json:"reduceOnly"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderSource            string        `json:"orderSource"`
	CancelReason           string        `json:"cancelReason"`
	CreationTime           types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
	// The following fields are not in the documentation, but are still returned
	PresetStopSurplusExecutePrice types.Number `json:"presetStopSurplusExecutePrice"`
	PresetStopSurplusType         string       `json:"presetStopSurplusType"`
	PresetStopLossExecutePrice    types.Number `json:"presetStopLossExecutePrice"`
	PresetStopLossType            string       `json:"presetStopLossType"`
	NewTradeSide                  string       `json:"newTradeSide"`
}

// FuturesFill is a sub-struct containing information on fulfilled futures orders
type FuturesFill struct {
	TradeID          int64               `json:"tradeId,string"`
	Symbol           string              `json:"symbol"`
	OrderID          int64               `json:"orderId,string"`
	Price            types.Number        `json:"price"`
	BaseVolume       types.Number        `json:"baseVolume"`
	FeeDetail        []AbridgedFeeDetail `json:"feeDetail"`
	Side             string              `json:"side"`
	QuoteVolume      types.Number        `json:"quoteVolume"`
	Profit           types.Number        `json:"profit"`
	EnterPointSource string              `json:"enterPointSource"`
	TradeSide        string              `json:"tradeSide"`
	PositionMode     string              `json:"posMode"`
	TradeScope       string              `json:"tradeScope"`
	CreationTime     types.Time          `json:"cTime"`
}

// FuturesFillsResp contains information on fulfilled futures orders
type FuturesFillsResp struct {
	FillList []FuturesFill `json:"fillList"`
	EndID    EmptyInt      `json:"endId"`
}

// FuturesOrder is a sub-struct containing information on futures orders
type FuturesOrder struct {
	Symbol                        string        `json:"symbol"`
	Size                          types.Number  `json:"size"`
	OrderID                       int64         `json:"orderId,string"`
	ClientOrderID                 string        `json:"clientOid"`
	BaseVolume                    types.Number  `json:"baseVolume"`
	Fee                           types.Number  `json:"fee"`
	Price                         types.Number  `json:"price"`
	PriceAverage                  types.Number  `json:"priceAvg"`
	Status                        string        `json:"status"`
	Side                          string        `json:"side"`
	Force                         string        `json:"force"`
	TotalProfits                  types.Number  `json:"totalProfits"`
	PositionSide                  string        `json:"posSide"`
	MarginCoin                    currency.Code `json:"marginCoin"`
	QuoteVolume                   types.Number  `json:"quoteVolume"`
	Leverage                      types.Number  `json:"leverage"`
	MarginMode                    string        `json:"marginMode"`
	ReduceOnly                    YesNoBool     `json:"reduceOnly"`
	EnterPointSource              string        `json:"enterPointSource"`
	TradeSide                     string        `json:"tradeSide"`
	PositionMode                  string        `json:"posMode"`
	OrderType                     string        `json:"orderType"`
	OrderSource                   string        `json:"orderSource"`
	CreationTime                  types.Time    `json:"cTime"`
	UpdateTime                    types.Time    `json:"uTime"`
	PresetStopSurplusPrice        types.Number  `json:"presetStopSurplusPrice"`
	PresetStopSurplusType         string        `json:"presetStopSurplusType"`
	PresetStopSurplusExecutePrice types.Number  `json:"presetStopSurplusExecutePrice"`
	PresetStopLossPrice           types.Number  `json:"presetStopLossPrice"`
	PresetStopLossType            string        `json:"presetStopLossType"`
	PresetStopLossExecutePrice    types.Number  `json:"presetStopLossExecutePrice"`
}

// FuturesOrdResp contains information on futures orders
type FuturesOrdResp struct {
	EntrustedList []FuturesOrder `json:"entrustedList"`
	EndID         EmptyInt       `json:"endId"`
}

// HistFuturesOrder is a sub-struct containing information on historical futures orders
type HistFuturesOrder struct {
	Symbol                 string        `json:"symbol"`
	Size                   types.Number  `json:"size"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	BaseVolume             types.Number  `json:"baseVolume"`
	Fee                    types.Number  `json:"fee"`
	Price                  types.Number  `json:"price"`
	PriceAverage           types.Number  `json:"priceAvg"`
	Status                 string        `json:"status"`
	Side                   string        `json:"side"`
	Force                  string        `json:"force"`
	TotalProfits           types.Number  `json:"totalProfits"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             currency.Code `json:"marginCoin"`
	QuoteVolume            types.Number  `json:"quoteVolume"`
	Leverage               types.Number  `json:"leverage"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	OrderSource            string        `json:"orderSource"`
	ReduceOnly             YesNoBool     `json:"reduceOnly"`
	CreationTime           types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
	PresetStopSurplusPrice types.Number  `json:"presetStopSurplusPrice"`
	PresetStopSurplusType  string        `json:"presetStopSurplusType"`
}

// HistFuturesOrdResp contains information on historical futures orders
type HistFuturesOrdResp struct {
	EntrustedList []FuturesOrder `json:"entrustedList"`
	EndID         EmptyInt       `json:"endId"`
}

// PlanFuturesOrder is a sub-struct containing information on planned futures orders
type PlanFuturesOrder struct {
	PlanType               string        `json:"planType"`
	Symbol                 string        `json:"symbol"`
	Size                   types.Number  `json:"size"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	Price                  types.Number  `json:"price"`
	ExecutePrice           types.Number  `json:"executePrice"`
	CallbackRatio          types.Number  `json:"callbackRatio"`
	TriggerPrice           types.Number  `json:"triggerPrice"`
	TriggerType            string        `json:"triggerType"`
	PlanStatus             string        `json:"planStatus"`
	Side                   string        `json:"side"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             currency.Code `json:"marginCoin"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	OrderSource            string        `json:"orderSource"`
	CreationTime           types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
	TakeProfitExecutePrice types.Number  `json:"stopSurplusExecutePrice"`
	TakeProfitTriggerPrice types.Number  `json:"stopSurplusTriggerPrice"`
	TakeProfitTriggerType  string        `json:"stopSurplusTriggerType"`
	StopLossExecutePrice   types.Number  `json:"stopLossExecutePrice"`
	StopLossTriggerPrice   types.Number  `json:"stopLossTriggerPrice"`
	StopLossTriggerType    string        `json:"stopLossTriggerType"`
}

// PlanFuturesOrdResp contains information on planned futures orders
type PlanFuturesOrdResp struct {
	EntrustedList []PlanFuturesOrder `json:"entrustedList"`
	EndID         EmptyInt           `json:"endId"`
}

// HistTriggerFuturesOrd is a sub-struct containing information on historical trigger futures orders
type HistTriggerFuturesOrd struct {
	PlanType               string        `json:"planType"`
	Symbol                 string        `json:"symbol"`
	Size                   types.Number  `json:"size"`
	OrderID                int64         `json:"orderId,string"`
	ExecuteOrderID         int64         `json:"executeOrderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	PlanStatus             string        `json:"planStatus"`
	Price                  types.Number  `json:"price"`
	ExecutePrice           types.Number  `json:"executePrice"`
	PriceAverage           types.Number  `json:"priceAvg"`
	BaseVolume             types.Number  `json:"baseVolume"`
	CallbackRatio          types.Number  `json:"callbackRatio"`
	TriggerPrice           types.Number  `json:"triggerPrice"`
	TriggerType            string        `json:"triggerType"`
	Side                   string        `json:"side"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             currency.Code `json:"marginCoin"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	CreationTime           types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
	PresetTakeProfitPrice  types.Number  `json:"presetStopSurplusPrice"`
	TakeprofitTriggerPrice types.Number  `json:"stopSurplusTriggerPrice"`
	TakeProfitTriggerType  string        `json:"stopSurplusTriggerType"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice"`
	StopLossTriggerPrice   types.Number  `json:"stopLossTriggerPrice"`
	StopLossTriggerType    string        `json:"stopLossTriggerType"`
}

// HistTriggerFuturesOrdResp contains information on historical trigger futures orders
type HistTriggerFuturesOrdResp struct {
	EntrustedList []HistTriggerFuturesOrd `json:"entrustedList"`
	EndID         EmptyInt                `json:"endId"`
}

// SupCurrencyResp contains information on supported currencies
type SupCurrencyResp struct {
	Symbol                    string        `json:"symbol"`
	BaseCoin                  currency.Code `json:"baseCoin"`
	QuoteCoin                 currency.Code `json:"quoteCoin"`
	MaximumCrossedLeverage    types.Number  `json:"maxCrossedLeverage"`
	MaximumIsolatedLeverage   types.Number  `json:"maxIsolatedLeverage"`
	WarningRiskRatio          types.Number  `json:"warningRiskRatio"`
	LiquidationRiskRatio      types.Number  `json:"liquidationRiskRatio"`
	MinimumTradeAmount        types.Number  `json:"minTradeAmount"`
	MaximumTradeAmount        types.Number  `json:"maxTradeAmount"`
	TakerFeeRate              types.Number  `json:"takerFeeRate"`
	MakerFeeRate              types.Number  `json:"makerFeeRate"`
	PricePrecision            uint8         `json:"pricePrecision,string"`
	QuantityPrecision         uint8         `json:"quantityPrecision,string"`
	MinimumTradeUSDT          types.Number  `json:"minTradeUSDT"`
	IsBorrowable              bool          `json:"isBorrowable"`
	UserMinBorrow             types.Number  `json:"userMinBorrow"`
	Status                    string        `json:"status"`
	IsIsolatedBaseBorrowable  bool          `json:"isIsolatedBaseBorrowable"`
	IsIsolatedQuoteBorrowable bool          `json:"isIsolatedQuoteBorrowable"`
	IsCrossBorrowable         bool          `json:"isCrossBorrowable"`
}

// CrossBorrow is a sub-struct containing information on borrowing for cross margin
type CrossBorrow struct {
	LoanID       int64         `json:"loanId,string"`
	Coin         currency.Code `json:"coin"`
	BorrowAmount types.Number  `json:"borrowAmount"`
	BorrowType   string        `json:"borrowType"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
}

// BorrowHistCross contains information on borrowing history for cross margin
type BorrowHistCross struct {
	ResultList []CrossBorrow `json:"resultList"`
	MaximumID  EmptyInt      `json:"maxId"`
	MinimumID  EmptyInt      `json:"minId"`
}

// CrossRepayment is a sub-struct containing information on a repayment for cross margin
type CrossRepayment struct {
	RepayID        int64         `json:"repayId,string"`
	Coin           currency.Code `json:"coin"`
	RepayAmount    types.Number  `json:"repayAmount"`
	RepayType      string        `json:"repayType"`
	RepayInterest  types.Number  `json:"repayInterest"`
	RepayPrincipal types.Number  `json:"repayPrincipal"`
	CreationTime   types.Time    `json:"cTime"`
	UpdateTime     types.Time    `json:"uTime"`
}

// CrossRepayHistResp contains information on repayment history for cross margin
type CrossRepayHistResp struct {
	ResultList []CrossRepayment `json:"resultList"`
	MaximumID  EmptyInt         `json:"maxId"`
	MinimumID  EmptyInt         `json:"minId"`
}

// CrossInterest is a sub-struct containing information on interest for cross margin
type CrossInterest struct {
	InterestID        int64         `json:"interestId,string"`
	LoanCoin          currency.Code `json:"loanCoin"`
	InterestCoin      currency.Code `json:"interestCoin"`
	DailyInterestRate types.Number  `json:"dailyInterestRate"`
	InterestAmount    types.Number  `json:"interestAmount"`
	InterestType      string        `json:"interstType"` // Misspelling of interestType
	CreationTime      types.Time    `json:"cTime"`
	UpdateTime        types.Time    `json:"uTime"`
}

// InterHistCross contains information on interest history for cross margin
type InterHistCross struct {
	MinimumID  EmptyInt        `json:"minId"`
	MaximumID  EmptyInt        `json:"maxId"`
	ResultList []CrossInterest `json:"resultList"`
}

// CrossLiquidation is a sub-struct containing information on liquidation for cross margin
type CrossLiquidation struct {
	LiquidationID        int64        `json:"liqId,string"`
	LiquidationStartTime types.Time   `json:"liqStartTime"`
	LiquidationEndTime   types.Time   `json:"liqEndTime"`
	LiquidationRiskRatio types.Number `json:"liqRiskRatio"`
	TotalAssets          types.Number `json:"totalAssets"`
	TotalDebt            types.Number `json:"totalDebt"`
	LiquidationFee       types.Number `json:"liqFee"`
	UpdateTime           types.Time   `json:"uTime"`
	CreationTime         types.Time   `json:"cTime"`
}

// LiquidHistCross contains information on liquidation history for cross margin
type LiquidHistCross struct {
	MinimumID  EmptyInt           `json:"minId"`
	MaximumID  EmptyInt           `json:"maxId"`
	ResultList []CrossLiquidation `json:"resultList"`
}

// CrossFinHist is a sub-struct containing information on financial history for cross margin
type CrossFinHist struct {
	MarginID     int64         `json:"marginId,string"`
	Amount       types.Number  `json:"amount"`
	Coin         currency.Code `json:"coin"`
	Balance      types.Number  `json:"balance"`
	Fee          types.Number  `json:"fee"`
	MarginType   string        `json:"marginType"`
	UpdateTime   types.Time    `json:"uTime"`
	CreationTime types.Time    `json:"cTime"`
}

// FinHistCrossResp contains information on financial history for cross margin
type FinHistCrossResp struct {
	MinimumID  EmptyInt       `json:"minId"`
	MaximumID  EmptyInt       `json:"maxId"`
	ResultList []CrossFinHist `json:"resultList"`
}

// CrossAssetResp contains information on assets being utilised in cross margin
type CrossAssetResp struct {
	Coin         currency.Code `json:"coin"`
	TotalAmount  types.Number  `json:"totalAmount"`
	Available    types.Number  `json:"available"`
	Frozen       types.Number  `json:"frozen"`
	Borrow       types.Number  `json:"borrow"`
	Interest     types.Number  `json:"interest"`
	Net          types.Number  `json:"net"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
	Coupon       types.Number  `json:"coupon"`
}

// BorrowCross contains information on borrowing for cross margin
type BorrowCross struct {
	LoanID       int64         `json:"loanId,string"`
	Coin         currency.Code `json:"coin"`
	BorrowAmount types.Number  `json:"borrowAmount"`
}

// RepayCross contains information on repayment for cross margin
type RepayCross struct {
	Coin                currency.Code `json:"coin"`
	RepayID             int64         `json:"repayId,string"`
	RemainingDebtAmount types.Number  `json:"remainDebtAmount"`
	RepayAmount         types.Number  `json:"repayAmount"`
}

// MaxBorrowCross contains information on the maximum amount that can be borrowed for cross margin
type MaxBorrowCross struct {
	Coin                    currency.Code `json:"coin"`
	MaximumBorrowableAmount types.Number  `json:"maxBorrowableAmount"`
}

// MaxTransferCross contains information on the maximum amount that can be transferred out of cross margin
type MaxTransferCross struct {
	Coin                     currency.Code `json:"coin"`
	MaximumTransferOutAmount types.Number  `json:"maxTransferOutAmount"`
}

// VIPInfo is a sub-struct containing information on VIP levels
type VIPInfo struct {
	Level              int64        `json:"level,string"`
	Limit              types.Number `json:"limit"`
	DailyInterestRate  types.Number `json:"dailyInterestRate"`
	AnnualInterestRate types.Number `json:"annualInterestRate"`
	DiscountRate       types.Number `json:"discountRate"`
}

// IntRateMaxBorrowCross contains information on the interest rate and the maximum amount that can be borrowed for
// cross margin
type IntRateMaxBorrowCross struct {
	Transferable            bool          `json:"transferable"`
	Leverage                types.Number  `json:"leverage"`
	Coin                    currency.Code `json:"coin"`
	Borrowable              bool          `json:"borrowable"`
	DailyInterestRate       types.Number  `json:"dailyInterestRate"`
	AnnualInterestRate      types.Number  `json:"annualInterestRate"`
	MaximumBorrowableAmount types.Number  `json:"maxBorrowableAmount"`
	VIPList                 []VIPInfo     `json:"vipList"`
}

// TierConfigCross contains information on tier configurations for cross margin
type TierConfigCross struct {
	Tier                    int64         `json:"tier,string"`
	Leverage                types.Number  `json:"leverage"`
	Coin                    currency.Code `json:"coin"`
	MaximumBorrowableAmount types.Number  `json:"maxBorrowableAmount"`
	MaintainMarginRate      types.Number  `json:"maintainMarginRate"`
}

// FlashRepayCross contains information on a flash repayment for cross margin
type FlashRepayCross struct {
	RepayID int64         `json:"repayId,string"`
	Coin    currency.Code `json:"coin"`
}

// FlashRepayResult contains information on the result of a flash repayment
type FlashRepayResult struct {
	RepayID int64  `json:"repayId,string"`
	Status  string `json:"status"`
}

// MarginOrderData contains information on a margin order
type MarginOrderData struct {
	Side          string       `json:"side"`
	OrderType     string       `json:"orderType"`
	Price         types.Number `json:"price"`
	Strategy      string       `json:"force"`
	BaseAmount    types.Number `json:"baseSize"`
	QuoteAmount   types.Number `json:"quoteSize"`
	LoanType      string       `json:"loanType"`
	ClientOrderID string       `json:"clientOid"`
	STPMode       string       `json:"stpMode"`
}

// MarginOrder is a sub-struct containing information on a margin order
type MarginOrder struct {
	OrderID          int64        `json:"orderId,string"`
	Symbol           string       `json:"symbol"`
	OrderType        string       `json:"orderType"`
	EnterPointSource string       `json:"enterPointSource"`
	ClientOrderID    string       `json:"clientOid"`
	LoanType         string       `json:"loanType"`
	Price            types.Number `json:"price"`
	Side             string       `json:"side"`
	Status           string       `json:"status"`
	BaseSize         types.Number `json:"baseSize"`
	QuoteSize        types.Number `json:"quoteSize"`
	PriceAverage     types.Number `json:"priceAvg"`
	Size             types.Number `json:"size"`
	Amount           types.Number `json:"amount"`
	Force            string       `json:"force"`
	CreationTime     types.Time   `json:"cTime"`
	UpdateTime       types.Time   `json:"uTime"`
}

// MarginOrders contains information on margin orders
type MarginOrders struct {
	OrderList []MarginOrder `json:"orderList"`
	MaximumID EmptyInt      `json:"maxId"`
	MinimumID EmptyInt      `json:"minId"`
}

// MarginFill is a sub-struct containing information on fulfilled margin orders
type MarginFill struct {
	OrderID      int64             `json:"orderId,string"`
	TradeID      int64             `json:"tradeId,string"`
	OrderType    string            `json:"orderType"`
	Side         string            `json:"side"`
	PriceAverage types.Number      `json:"priceAvg"`
	Size         types.Number      `json:"size"`
	Amount       types.Number      `json:"amount"`
	TradeScope   string            `json:"tradeScope"`
	CreationTime types.Time        `json:"cTime"`
	UpdateTime   types.Time        `json:"uTime"`
	FeeDetail    AbridgedFeeDetail `json:"feeDetail"`
}

// MarginOrderFills contains information on fulfilled margin orders
type MarginOrderFills struct {
	Fills     []MarginFill `json:"fills"`
	MaximumID EmptyInt     `json:"maxId"`
	MinimumID EmptyInt     `json:"minId"`
}

// LiquidationOrder is a sub-struct containing information on liquidation orders
type LiquidationOrder struct {
	Symbol       string        `json:"symbol"`
	OrderType    string        `json:"orderType"`
	Side         string        `json:"side"`
	PriceAverage types.Number  `json:"priceAvg"`
	Price        types.Number  `json:"price"`
	FillSize     types.Number  `json:"fillSize"`
	Size         types.Number  `json:"size"`
	Amount       types.Number  `json:"amount"`
	OrderID      int64         `json:"orderId,string"`
	FromCoin     currency.Code `json:"fromCoin"`
	ToCoin       currency.Code `json:"toCoin"`
	FromSize     types.Number  `json:"fromSize"`
	ToSize       types.Number  `json:"toSize"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
}

// LiquidationResp contains information on liquidation orders
type LiquidationResp struct {
	ResultList []LiquidationOrder `json:"resultList"`
	IDLessThan EmptyInt           `json:"idLessThan"`
}

// IsoRepayment is a sub-struct containing information on a repayment for isolated margin
type IsoRepayment struct {
	RepayID        int64         `json:"repayId,string"`
	Coin           currency.Code `json:"coin"`
	RepayAmount    types.Number  `json:"repayAmount"`
	RepayType      string        `json:"repayType"`
	RepayInterest  types.Number  `json:"repayInterest"`
	RepayPrincipal types.Number  `json:"repayPrincipal"`
	Symbol         string        `json:"symbol"`
	CreationTime   types.Time    `json:"cTime"`
	UpdateTime     types.Time    `json:"uTime"`
}

// IsoRepayHistResp contains information on repayment history for isolated margin
type IsoRepayHistResp struct {
	ResultList []IsoRepayment `json:"resultList"`
	MaximumID  EmptyInt       `json:"maxId"`
	MinimumID  EmptyInt       `json:"minId"`
}

// IsoBorrow is a sub-struct containing information on borrowing for isolated margin
type IsoBorrow struct {
	LoanID       int64         `json:"loanId,string"`
	Coin         currency.Code `json:"coin"`
	BorrowAmount types.Number  `json:"borrowAmount"`
	BorrowType   string        `json:"borrowType"`
	Symbol       string        `json:"symbol"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
}

// BorrowHistIso contains information on borrowing history for isolated margin
type BorrowHistIso struct {
	ResultList []IsoBorrow `json:"resultList"`
	MaximumID  EmptyInt    `json:"maxId"`
	MinimumID  EmptyInt    `json:"minId"`
}

// IsoInterest is a sub-struct containing information on interest for isolated margin
type IsoInterest struct {
	InterestID        int64         `json:"interestId,string"`
	LoanCoin          currency.Code `json:"loanCoin"`
	InterestCoin      currency.Code `json:"interestCoin"`
	DailyInterestRate types.Number  `json:"dailyInterestRate"`
	InterestAmount    types.Number  `json:"interestAmount"`
	InterestType      string        `json:"interstType"` // Misspelling of interestType
	Symbol            string        `json:"symbol"`
	CreationTime      types.Time    `json:"cTime"`
	UpdateTime        types.Time    `json:"uTime"`
}

// InterHistIso contains information on interest history for isolated margin
type InterHistIso struct {
	MinimumID  EmptyInt      `json:"minId"`
	MaximumID  EmptyInt      `json:"maxId"`
	ResultList []IsoInterest `json:"resultList"`
}

// IsoLiquidation is a sub-struct containing information on liquidation for isolated margin
type IsoLiquidation struct {
	LiquidationID        int64        `json:"liqId,string"`
	Symbol               string       `json:"symbol"`
	LiquidationStartTime types.Time   `json:"liqStartTime"`
	LiquidationEndTime   types.Time   `json:"liqEndTime"`
	LiquidationRiskRatio types.Number `json:"liqRiskRatio"`
	TotalAssets          types.Number `json:"totalAssets"`
	TotalDebt            types.Number `json:"totalDebt"`
	LiquidationFee       types.Number `json:"liqFee"`
	UpdateTime           types.Time   `json:"uTime"`
	CreationTime         types.Time   `json:"cTime"`
}

// LiquidHistIso contains information on liquidation history for isolated margin
type LiquidHistIso struct {
	MinimumID  EmptyInt         `json:"minId"`
	MaximumID  EmptyInt         `json:"maxId"`
	ResultList []IsoLiquidation `json:"resultList"`
}

// IsoFinHist is a sub-struct containing information on financial history for isolated margin
type IsoFinHist struct {
	MarginID     int64         `json:"marginId,string"`
	Amount       types.Number  `json:"amount"`
	Coin         currency.Code `json:"coin"`
	Symbol       string        `json:"symbol"`
	Balance      types.Number  `json:"balance"`
	Fee          types.Number  `json:"fee"`
	MarginType   string        `json:"marginType"`
	UpdateTime   types.Time    `json:"uTime"`
	CreationTime types.Time    `json:"cTime"`
}

// FinHistIsoResp contains information on financial history for isolated margin
type FinHistIsoResp struct {
	MinimumID  EmptyInt     `json:"minId"`
	MaximumID  EmptyInt     `json:"maxId"`
	ResultList []IsoFinHist `json:"resultList"`
}

// IsoAssetResp contains information on assets being utilised in isolated margin
type IsoAssetResp struct {
	Symbol       string        `json:"symbol"`
	Coin         currency.Code `json:"coin"`
	TotalAmount  types.Number  `json:"totalAmount"`
	Available    types.Number  `json:"available"`
	Frozen       types.Number  `json:"frozen"`
	Borrow       types.Number  `json:"borrow"`
	Interest     types.Number  `json:"interest"`
	Net          types.Number  `json:"net"`
	CreationTime types.Time    `json:"cTime"`
	UpdateTime   types.Time    `json:"uTime"`
	Coupon       types.Number  `json:"coupon"`
}

// BorrowIso contains information on borrowing for isolated margin
type BorrowIso struct {
	LoanID       int64         `json:"loanId,string"`
	Symbol       string        `json:"symbol"`
	Coin         currency.Code `json:"coin"`
	BorrowAmount types.Number  `json:"borrowAmount"`
}

// RepayIso contains information on repayment for isolated margin
type RepayIso struct {
	Coin                currency.Code `json:"coin"`
	Symbol              string        `json:"symbol"`
	RepayID             int64         `json:"repayId,string"`
	RemainingDebtAmount types.Number  `json:"remainDebtAmount"`
	RepayAmount         types.Number  `json:"repayAmount"`
}

// RiskRateIso contains information on the risk rate for isolated margin
type RiskRateIso struct {
	Symbol        string       `json:"symbol"`
	RiskRateRatio types.Number `json:"riskRateRatio"`
}

// IsoVIPList contains information on VIP lists for isolated margin
type IsoVIPList struct {
	Level              int64        `json:"level,string"`
	Limit              types.Number `json:"limit"`
	DailyInterestRate  types.Number `json:"dailyInterestRate"`
	AnnualInterestRate types.Number `json:"annuallyInterestRate"` // Misspelling of annualInterestRate
	DiscountRate       types.Number `json:"discountRate"`
}

// IntRateMaxBorrowIso contains information on the interest rate and the maximum amount that can be borrowed for isolated margin
type IntRateMaxBorrowIso struct {
	Symbol                   string        `json:"symbol"`
	Leverage                 types.Number  `json:"leverage"`
	BaseCoin                 currency.Code `json:"baseCoin"`
	BaseTransferable         bool          `json:"baseTransferable"`
	BaseBorrowable           bool          `json:"baseBorrowable"`
	BaseDailyInterestRate    types.Number  `json:"baseDailyInterestRate"`
	BaseAnnualInterestRate   types.Number  `json:"baseAnnuallyInterestRate"` // Misspelling of baseAnnualInterestRate
	BaseMaxBorrowableAmount  types.Number  `json:"baseMaxBorrowableAmount"`
	BaseVIPList              []IsoVIPList  `json:"baseVipList"`
	QuoteCoin                currency.Code `json:"quoteCoin"`
	QuoteTransferable        bool          `json:"quoteTransferable"`
	QuoteBorrowable          bool          `json:"quoteBorrowable"`
	QuoteDailyInterestRate   types.Number  `json:"quoteDailyInterestRate"`
	QuoteAnnualInterestRate  types.Number  `json:"quoteAnnuallyInterestRate"` // Misspelling of quoteAnnualInterestRate
	QuoteMaxBorrowableAmount types.Number  `json:"quoteMaxBorrowableAmount"`
	QuoteVIPList             []IsoVIPList  `json:"quoteList"`
}

// TierConfigIso contains information on tier configurations for isolated margin
type TierConfigIso struct {
	Tier                     int64         `json:"tier,string"`
	Symbol                   string        `json:"symbol"`
	Leverage                 types.Number  `json:"leverage"`
	BaseCoin                 currency.Code `json:"baseCoin"`
	QuoteCoin                currency.Code `json:"quoteCoin"`
	BaseMaxBorrowableAmount  types.Number  `json:"baseMaxBorrowableAmount"`
	QuoteMaxBorrowableAmount types.Number  `json:"quoteMaxBorrowableAmount"`
	MaintainMarginRate       types.Number  `json:"maintainMarginRate"`
	InitRate                 types.Number  `json:"initRate"`
}

// MaxBorrowIso contains information on the maximum amount that can be borrowed for isolated margin
type MaxBorrowIso struct {
	Symbol                       string        `json:"symbol"`
	BaseCoin                     currency.Code `json:"baseCoin"`
	BaseCoinMaxBorrowableAmount  types.Number  `json:"baseCoinmaxBorrowAmount"`
	QuoteCoin                    currency.Code `json:"quoteCoin"`
	QuoteCoinMaxBorrowableAmount types.Number  `json:"quoteCoinmaxBorrowAmount"`
}

// MaxTransferIso contains information on the maximum amount that can be transferred out of isolated margin
type MaxTransferIso struct {
	BaseCoin                      currency.Code `json:"baseCoin"`
	Symbol                        string        `json:"symbol"`
	BaseCoinMaxTransferOutAmount  types.Number  `json:"baseCoinMaxTransferOutAmount"`
	QuoteCoin                     currency.Code `json:"quoteCoin"`
	QuoteCoinMaxTransferOutAmount types.Number  `json:"quoteCoinMaxTransferOutAmount"`
}

// FlashRepayIso contains information on a flash repayment for isolated margin
type FlashRepayIso struct {
	RepayID int64       `json:"repayId,string"`
	Symbol  string      `json:"symbol"`
	Result  SuccessBool `json:"result"`
}

// APY contains information on the APY of a savings product
type APY struct {
	RateLevel        int64        `json:"rateLevel,string"`
	MinimumStepValue types.Number `json:"minStepVal"`
	MaximumStepValue types.Number `json:"maxStepVal"`
	CurrentAPY       types.Number `json:"currentAPY"`
}

// SavingsProductList contains information on savings products
type SavingsProductList struct {
	ProductID     int64         `json:"productId,string"`
	Coin          currency.Code `json:"coin"`
	PeriodType    string        `json:"periodType"`
	Period        EmptyInt      `json:"period"`
	APYType       string        `json:"apyType"`
	AdvanceRedeem YesNoBool     `json:"advanceRedeem"`
	SettleMethod  string        `json:"settleMethod"`
	APYList       []APY         `json:"apyList"`
	Status        string        `json:"status"`
	ProductLevel  string        `json:"productLevel"`
}

// SavingsBalance contains information on savings balances
type SavingsBalance struct {
	BTCAmount          types.Number `json:"btcAmount"`
	USDTAmount         types.Number `json:"usdtAmount"`
	BTC24HourEarnings  types.Number `json:"btc24HourEarning"`
	USDT24HourEarnings types.Number `json:"usdt24HourEarning"`
	BTCTotalEarnings   types.Number `json:"btcTotalEarning"`
	USDTTotalEarnings  types.Number `json:"usdtTotalEarning"`
}

// SavingsAsset is a sub-struct containing information on savings assets
type SavingsAsset struct {
	ProductID       int64         `json:"productId,string"`
	OrderID         int64         `json:"orderId,string"`
	ProductCoin     currency.Code `json:"productCoin"`
	InterestCoin    currency.Code `json:"interestCoin"`
	PeriodType      string        `json:"periodType"`
	Period          EmptyInt      `json:"period"`
	HoldAmount      types.Number  `json:"holdAmount"`
	LastProfit      types.Number  `json:"lastProfit"`
	TotalProfit     types.Number  `json:"totalProfit"`
	HoldDays        EmptyInt      `json:"holdDays"`
	Status          string        `json:"status"`
	AllowRedemption YesNoBool     `json:"allowRedemption"`
	ProductLevel    string        `json:"productLevel"`
	APY             []APY         `json:"apy"`
}

// SavingsAssetsResp contains information on savings assets
type SavingsAssetsResp struct {
	ResultList []SavingsAsset `json:"resultList"`
	EndID      EmptyInt       `json:"endId"`
}

// SavingsTransaction is a sub-struct containing information on a savings transaction
type SavingsTransaction struct {
	OrderID        int64         `json:"orderId,string"`
	CoinName       currency.Code `json:"coinName"`
	SettleCoinName currency.Code `json:"settleCoinName"`
	ProductType    string        `json:"productType"`
	Period         EmptyInt      `json:"period"`
	ProductLevel   string        `json:"productLevel"`
	Amount         types.Number  `json:"amount"`
	Timestamp      types.Time    `json:"ts"`
	OrderType      string        `json:"orderType"`
}

// SavingsRecords contains information on previous transactions
type SavingsRecords struct {
	ResultList []SavingsTransaction `json:"resultList"`
	EndID      EmptyInt             `json:"endId"`
}

// SavingsSubDetail contains information about a potential subscription
type SavingsSubDetail struct {
	SingleMinAmount    types.Number `json:"singleMinAmount"`
	SingleMaxAmount    types.Number `json:"singleMaxAmount"`
	RemainingAmount    types.Number `json:"remainingAmount"`
	SubscribePrecision uint8        `json:"subscribePrecision,string"`
	ProfitPrecision    uint8        `json:"profitPrecision,string"`
	SubscribeTime      types.Time   `json:"subscribeTime"`
	InterestTime       types.Time   `json:"interestTime"`
	SettleTime         types.Time   `json:"settleTime"`
	ExpireTime         types.Time   `json:"expireTime"`
	RedeemTime         types.Time   `json:"redeemTime"`
	SettleMethod       string       `json:"settleMethod"`
	APYList            []APY        `json:"apyList"`
	RedeemDelay        string       `json:"redeemDelay"`
}

// SaveResp contains information on a transaction involving a savings product
type SaveResp struct {
	OrderID int64  `json:"orderId,string"`
	Status  string `json:"status"` // Double-check, might be a float64
}

// SaveResult contains information on the result of a transaction involving a savings product
type SaveResult struct {
	Result  SuccessBool `json:"result"`
	Message string      `json:"msg"`
}

// EarnAssets contains information on assets in the earn account
type EarnAssets struct {
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
}

// SharkFinProduct is a sub-struct containing information on a shark fin product
type SharkFinProduct struct {
	ProductID         int64         `json:"productId,string"`
	ProductName       string        `json:"productName"`
	ProductCoin       currency.Code `json:"productCoin"`
	SubscribeCoin     currency.Code `json:"subscribeCoin"`
	FarmingStartTime  types.Time    `json:"farmingStartTime"`
	FarmingEndTime    types.Time    `json:"farmingEndTime"`
	LowerRate         types.Number  `json:"lowerRate"`
	DefaultRate       types.Number  `json:"defaultRate"`
	UpperRate         types.Number  `json:"upperRate"`
	Period            EmptyInt      `json:"period"`
	InterestStartTime types.Time    `json:"interestStartTime"`
	Status            string        `json:"status"`
	MinimumAmount     types.Number  `json:"minAmount"`
	LimitAmount       types.Number  `json:"limitAmount"`
	SoldAmount        types.Number  `json:"soldAmount"`
	EndTime           types.Time    `json:"endTime"`
	StartTime         types.Time    `json:"startTime"`
}

// SharkFinProductResp contains information on shark fin products
type SharkFinProductResp struct {
	ResultList []SharkFinProduct `json:"resultList"`
	EndID      EmptyInt          `json:"endId"`
}

// SharkFinBalance contains information on one's shark fin balance and amount earned
type SharkFinBalance struct {
	BTCSubscribeAmount   types.Number `json:"btcSubscribeAmount"`
	USDTSubscribeAmount  types.Number `json:"usdtSubscribeAmount"`
	BTCHistoricalAmount  types.Number `json:"btcHistoricalAmount"`
	USDTHistoricalAmount types.Number `json:"usdtHistoricalAmount"`
	BTCTotalEarning      types.Number `json:"btcTotalEarning"`
	USDTTotalEarning     types.Number `json:"usdtTotalEarning"`
}

// SharkFinAsset is a sub-struct containing information on a shark fin asset
type SharkFinAsset struct {
	ProductID         int64         `json:"productId,string"`
	InterestStartTime types.Time    `json:"interestStartTime"`
	InterestEndTime   types.Time    `json:"interestEndTime"`
	ProductCoin       currency.Code `json:"productCoin"`
	SubscribeCoin     currency.Code `json:"subscribeCoin"`
	Trend             string        `json:"trend"`
	SettleTime        types.Time    `json:"settleTime"`
	InterestAmount    types.Number  `json:"interestAmount"`
	ProductStatus     string        `json:"productStatus"`
}

// SharkFinAssetsResp contains information on one's shark fin assets
type SharkFinAssetsResp struct {
	ResultList []SharkFinAsset `json:"resultList"`
	EndID      EmptyInt        `json:"endId"`
}

// SharkFinRecords contains information on one's shark fin records
type SharkFinRecords struct {
	OrderID   int64        `json:"orderId,string"`
	Product   string       `json:"product"`
	Period    EmptyInt     `json:"period"`
	Amount    types.Number `json:"amount"`
	Timestamp types.Time   `json:"ts"`
	Type      string       `json:"type"`
}

// SharkFinSubDetail contains information useful when subscribing to a shark fin product
type SharkFinSubDetail struct {
	ProductCoin        currency.Code `json:"productCoin"`
	SubscribeCoin      currency.Code `json:"subscribeCoin"`
	InterestTime       types.Time    `json:"interestTime"`
	ExpirationTime     types.Time    `json:"expirationTime"`
	MinimumPrice       types.Number  `json:"minPrice"`
	CurrentPrice       types.Number  `json:"currentPrice"`
	MaximumPrice       types.Number  `json:"maxPrice"`
	MinimumRate        types.Number  `json:"minRate"`
	DefaultRate        types.Number  `json:"defaultRate"`
	MaximumRate        types.Number  `json:"maxRate"`
	Period             EmptyInt      `json:"period"`
	ProductMinAmount   types.Number  `json:"productMinAmount"`
	AvailableBalance   types.Number  `json:"availableBalance"`
	UserAmount         types.Number  `json:"userAmount"`
	RemainingAmount    types.Number  `json:"remainingAmount"`
	ProfitPrecision    uint8         `json:"profitPrecision,string"`
	SubscribePrecision uint8         `json:"subscribePrecision,string"`
}

// LoanInfos is a sub-struct containing information on loans
type LoanInfos struct {
	Coin            currency.Code `json:"coin"`
	HourlyRate7Day  types.Number  `json:"hourRate7D"`
	Rate7Day        types.Number  `json:"rate7D"`
	HourlyRate30Day types.Number  `json:"hourRate30D"`
	Rate30Day       types.Number  `json:"rate30D"`
	MinimumUSDT     types.Number  `json:"minUsdt"`
	MaximumUSDT     types.Number  `json:"maxUsdt"`
	Minimum         types.Number  `json:"min"`
	Maximum         types.Number  `json:"max"`
}

// PledgeInfos is a sub-struct containing information on pledges
type PledgeInfos struct {
	Coin              currency.Code `json:"coin"`
	InitialRate       types.Number  `json:"initRate"`
	SupplementaryRate types.Number  `json:"supRate"`
	ForceRate         types.Number  `json:"forceRate"`
	MinimumUSDT       types.Number  `json:"minUsdt"`
	MaximumUSDT       types.Number  `json:"maxUsdt"`
}

// LoanCurList contains information on currencies which can be loaned
type LoanCurList struct {
	LoanInfos   []LoanInfos   `json:"loanInfos"`
	PledgeInfos []PledgeInfos `json:"pledgeInfos"`
}

// EstimateInterest contains information on estimated interest payments and borrowable amounts
type EstimateInterest struct {
	HourInterest types.Number `json:"hourInterest"`
	LoanAmount   types.Number `json:"loanAmount"`
}

// OngoingLoans contains information on ongoing loans
type OngoingLoans struct {
	OrderID           int64         `json:"orderId,string"`
	LoanCoin          currency.Code `json:"loanCoin"`
	LoanAmount        types.Number  `json:"loanAmount"`
	InterestAmount    types.Number  `json:"interestAmount"`
	HourInterestRate  types.Number  `json:"hourInterestRate"`
	PledgeCoin        currency.Code `json:"pledgeCoin"`
	PledgeAmount      types.Number  `json:"pledgeAmount"`
	PledgeRate        types.Number  `json:"pledgeRate"`
	SupplementaryRate types.Number  `json:"supRate"`
	ForceRate         types.Number  `json:"forceRate"`
	BorrowTime        types.Time    `json:"borrowTime"`
	ExpireTime        types.Time    `json:"expireTime"`
}

// RepayResp contains information on a repayment
type RepayResp struct {
	LoanCoin          currency.Code `json:"loanCoin"`
	PledgeCoin        currency.Code `json:"pledgeCoin"`
	RepayAmount       types.Number  `json:"repayAmount"`
	PayInterest       types.Number  `json:"payInterest"`
	RepayLoanAmount   types.Number  `json:"repayLoanAmount"`
	RepayUnlockAmount types.Number  `json:"repayUnlockAmount"`
}

// RepayRecords contains information on repayment records
type RepayRecords struct {
	OrderID           int64         `json:"orderId,string"`
	LoanCoin          currency.Code `json:"loanCoin"`
	PledgeCoin        currency.Code `json:"pledgeCoin"`
	RepayAmount       types.Number  `json:"repayAmount"`
	PayInterest       types.Number  `json:"payInterest"`
	RepayLoanAmount   types.Number  `json:"repayLoanAmount"`
	RepayUnlockAmount types.Number  `json:"repayUnlockAmount"`
	RepayTime         types.Time    `json:"repayTime"`
}

// ModPledgeResp contains information on a pledge modification
type ModPledgeResp struct {
	LoanCoin        currency.Code `json:"loanCoin"`
	PledgeCoin      currency.Code `json:"pledgeCoin"`
	AfterPledgeRate types.Number  `json:"afterPledgeRate"`
}

// PledgeRateHist contains information on historical pledge rates
type PledgeRateHist struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	PledgeCoin       currency.Code `json:"pledgeCoin"`
	OrderID          int64         `json:"orderId,string"`
	ReviseTime       types.Time    `json:"reviseTime"`
	ReviseSide       string        `json:"reviseSide"`
	ReviseAmount     types.Number  `json:"reviseAmount"`
	AfterPledgeRate  types.Number  `json:"afterPledgeRate"`
	BeforePledgeRate types.Number  `json:"beforePledgeRate"`
}

// LoanHistory contains information on loans
type LoanHistory struct {
	OrderID             int64         `json:"orderId,string"`
	LoanCoin            currency.Code `json:"loanCoin"`
	PledgeCoin          currency.Code `json:"pledgeCoin"`
	InitialPledgeAmount types.Number  `json:"initPledgeAmount"`
	InitialLoanAmount   types.Number  `json:"initLoanAmount"`
	HourlyRate          types.Number  `json:"hourRate"`
	Daily               types.Number  `json:"daily"`
	BorrowTime          types.Time    `json:"borrowTime"`
	Status              string        `json:"status"`
}

// CoinAm includes fields for coins, amounts, and amount-equivalents in USDT
type CoinAm struct {
	Coin       currency.Code `json:"coin"`
	Amount     types.Number  `json:"amount"`
	AmountUSDT types.Number  `json:"amountUsdt"`
}

// DebtsResp contains information on debts
type DebtsResp struct {
	PledgeInfos []CoinAm `json:"pledgeInfos"`
	LoanInfos   []CoinAm `json:"loanInfos"`
}

// LiquidRecs contains information on liquidation records
type LiquidRecs struct {
	OrderID         int64         `json:"orderId,string"`
	LoanCoin        currency.Code `json:"loanCoin"`
	PledgeCoin      currency.Code `json:"pledgeCoin"`
	ReduceTime      types.Time    `json:"reduceTime"`
	PledgeRate      types.Number  `json:"pledgeRate"`
	PledgePrice     types.Number  `json:"pledgePrice"`
	Status          string        `json:"status"`
	PledgeAmount    types.Number  `json:"pledgeAmount"`
	ReduceFee       string        `json:"reduceFee"`
	ResidueAmount   types.Number  `json:"residueAmount"`
	RunlockAmount   types.Number  `json:"runlockAmount"`
	RepayLoanAmount types.Number  `json:"repayLoanAmount"`
}

// LoanInfo contains information on a loan
type LoanInfo struct {
	ProductID           string       `json:"productId"`
	Leverage            types.Number `json:"leverage"`
	TransferLine        types.Number `json:"transferLine"`
	SpotBuyLine         types.Number `json:"spotBuyLine"`
	LiquidationLine     types.Number `json:"liquidationLine"`
	StopLiquidationLine types.Number `json:"stopLiquidationLine"`
}

// CoinConverts contains information on coin conversion ratios
type CoinConverts struct {
	Coin                currency.Code `json:"coin"`
	ConvertRatio        types.Number  `json:"convertRatio"`
	MaximumConvertValue types.Number  `json:"maxConvertValue"`
}

// MarginCoinRatio contains information on margin coin conversion ratios
type MarginCoinRatio struct {
	ProductID string         `json:"productId"`
	CoinInfo  []CoinConverts `json:"coinInfo"`
}

// SpotSymbols contains information on spot symbols
type SpotSymbols struct {
	ProductID   string   `json:"productId"`
	SpotSymbols []string `json:"spotSymbols"`
}

// UnpaidLoanInfo contains information on unpaid loans
type UnpaidLoanInfo struct {
	Coin           currency.Code `json:"coin"`
	UnpaidQuantity types.Number  `json:"unpaidQty"`
	UnpaidInterest types.Number  `json:"unpaidInterest"`
}

// BalanceInfo contains information on balances
type BalanceInfo struct {
	Coin                currency.Code `json:"coin"`
	Price               types.Number  `json:"price"`
	Amount              types.Number  `json:"amount"`
	ConvertedUSDTAmount types.Number  `json:"convertedUsdtAmount"`
}

// LoanToValue contains information on loan-to-value ratios
type LoanToValue struct {
	LTV              types.Number     `json:"ltv"`
	SubAccountUIDs   []string         `json:"subAccountUids"`
	UnpaidUSDTAmount types.Number     `json:"unpaidUsdtAmount"`
	USDTBalance      types.Number     `json:"usdtBalance"`
	UnpaidInfo       []UnpaidLoanInfo `json:"unpaidInfo"`
	BalanceInfo      []BalanceInfo    `json:"balanceInfo"`
}

// TransferableAmount contains information on transferable amounts
type TransferableAmount struct {
	Coin        currency.Code `json:"coin"`
	Transferred types.Number  `json:"transfered"` //nolint:misspell // Bitget spelling mistake
	UserID      uint64        `json:"userId,string"`
}

// LoanOrders contains information on loan orders
type LoanOrders struct {
	OrderID        int64         `json:"orderId,string"`
	OrderProductID string        `json:"orderProductId"`
	UID            string        `json:"uid"`
	LoanTime       types.Time    `json:"loanTime"`
	LoanCoin       currency.Code `json:"loanCoin"`
	UnpaidAmount   types.Number  `json:"unpaidAmount"`
	UnpaidInterest types.Number  `json:"unpaidInterest"`
	LoanAmount     types.Number  `json:"loanAmount"`
	Status         string        `json:"status"`
	RepaidAmount   types.Number  `json:"repaidAmount"`
	RepaidInterest types.Number  `json:"repaidInterest"`
}

// RepaymentOrders contains information on repayment orders
type RepaymentOrders struct {
	RepayOrderID   int64         `json:"repayOrderId,string"`
	BusinessType   string        `json:"businessType"`
	RepayType      string        `json:"repayType"`
	RepaidTime     types.Time    `json:"repaidTime"`
	Coin           currency.Code `json:"coin"`
	RepaidAmount   types.Number  `json:"repaidAmount"`
	RepaidInterest types.Number  `json:"repaidInterest"`
}

// WsResponse contains information on a websocket response
type WsResponse struct {
	Event     string          `json:"event"`
	Code      int             `json:"code"`
	Message   string          `json:"msg"`
	Arg       WsArgument      `json:"arg"`
	Action    string          `json:"action"`
	Data      json.RawMessage `json:"data"`
	Timestamp types.Time      `json:"ts"`
}

// WsArgument contains information used in a websocket request and response
type WsArgument struct {
	InstrumentType string        `json:"instType"`
	Channel        string        `json:"channel"`
	InstrumentID   string        `json:"instId,omitempty"`
	Coin           currency.Code `json:"coin,omitempty"`
}

// WsRequest contains information on a websocket request
type WsRequest struct {
	Operation string       `json:"op"`
	Arguments []WsArgument `json:"args"`
}

// WsLoginArgument contains information used in a websocket login request
type WsLoginArgument struct {
	APIKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  string `json:"timestamp"`
	Signature  string `json:"sign"`
}

// WsLogin contains information on a websocket login request
type WsLogin struct {
	Operation string            `json:"op"`
	Arguments []WsLoginArgument `json:"args"`
}

// WsTickerSnapshotSpot contains information on a ticker snapshot
type WsTickerSnapshotSpot struct {
	InstrumentID string       `json:"instId"`
	LastPrice    types.Number `json:"lastPr"`
	Open24H      types.Number `json:"open24h"`
	High24H      types.Number `json:"high24h"`
	Low24H       types.Number `json:"low24h"`
	Change24H    types.Number `json:"change24h"`
	BidPrice     types.Number `json:"bidPr"`
	AskPrice     types.Number `json:"askPr"`
	BidSize      types.Number `json:"bidSz"`
	AskSize      types.Number `json:"askSz"`
	BaseVolume   types.Number `json:"baseVolume"`
	QuoteVolume  types.Number `json:"quoteVolume"`
	OpenUTC      types.Number `json:"openUtc"`
	ChangeUTC24H types.Number `json:"changeUtc24h"`
	Timestamp    types.Time   `json:"ts"`
}

// WsAccountSpotResponse contains information on an account response for spot trading
type WsAccountSpotResponse struct {
	Coin           currency.Code `json:"coin"`
	Available      types.Number  `json:"available"`
	Frozen         types.Number  `json:"frozen"`
	Locked         types.Number  `json:"locked"`
	LimitAvailable types.Number  `json:"limitAvailable"`
	UpdateTime     types.Time    `json:"uTime"`
}

// WsTradeResponse contains information on a trade response
type WsTradeResponse struct {
	Timestamp types.Time   `json:"ts"`
	Price     types.Number `json:"price"`
	Size      types.Number `json:"size"`
	Side      string       `json:"side"`
	TradeID   int64        `json:"tradeId,string"`
}

// WsOrderBookResponse contains information on an order book response
type WsOrderBookResponse struct {
	Asks      [][2]string `json:"asks"`
	Bids      [][2]string `json:"bids"`
	Timestamp types.Time  `json:"ts"`
	Checksum  int32       `json:"checksum"`
}

// WsFillSpotResponse contains information on a fill response for spot trading
type WsFillSpotResponse struct {
	OrderID      int64               `json:"orderId,string"`
	TradeID      int64               `json:"tradeId,string"`
	Symbol       string              `json:"symbol"`
	OrderType    string              `json:"orderType"`
	Side         string              `json:"side"`
	PriceAverage types.Number        `json:"priceAvg"`
	Size         types.Number        `json:"size"`
	Amount       types.Number        `json:"amount"`
	TradeScope   string              `json:"tradeScope"`
	FeeDetail    []AbridgedFeeDetail `json:"feeDetail"`
	CreationTime types.Time          `json:"cTime"`
	UpdateTime   types.Time          `json:"uTime"`
}

// WsOrderSpotResponse contains information on an order response for spot trading
type WsOrderSpotResponse struct {
	InstrumentID      string              `json:"instId"`
	OrderID           int64               `json:"orderId,string"`
	ClientOrderID     string              `json:"clientOid"`
	Size              types.Number        `json:"size"`
	NewSize           types.Number        `json:"newSize"`
	Notional          types.Number        `json:"notional"`
	OrderType         string              `json:"orderType"`
	Force             string              `json:"force"`
	Side              string              `json:"side"`
	FillPrice         types.Number        `json:"fillPrice"`
	TradeID           int64               `json:"tradeId,string"`
	BaseVolume        types.Number        `json:"baseVolume"`
	FillTime          types.Time          `json:"fillTime"`
	FillFee           types.Number        `json:"fillFee"`
	FillFeeCoin       currency.Code       `json:"fillFeeCoin"`
	TradeScope        string              `json:"tradeScope"`
	AccountBaseVolume types.Number        `json:"accBaseVolume"`
	PriceAverage      types.Number        `json:"priceAvg"`
	Status            string              `json:"status"`
	CreationTime      types.Time          `json:"cTime"`
	UpdateTime        types.Time          `json:"uTime"`
	STPMode           string              `json:"stpMode"`
	FeeDetail         []AbridgedFeeDetail `json:"feeDetail"`
	EnterPointSource  string              `json:"enterPointSource"`
}

// WsTriggerOrderSpotResponse contains information on a trigger order response for spot trading
type WsTriggerOrderSpotResponse struct {
	InstrumentID     string       `json:"instId"`
	OrderID          int64        `json:"orderId,string"`
	ClientOrderID    string       `json:"clientOid"`
	TriggerPrice     types.Number `json:"triggerPrice"`
	TriggerType      string       `json:"triggerType"`
	PlanType         string       `json:"planType"`
	Price            types.Number `json:"price"`
	Size             types.Number `json:"size"`
	ActualSize       types.Number `json:"actualSize"`
	OrderType        string       `json:"orderType"`
	Side             string       `json:"side"`
	Status           string       `json:"status"`
	ExecutePrice     types.Number `json:"execPrice"`
	EnterPointSource string       `json:"enterPointSource"`
	CreationTime     types.Time   `json:"cTime"`
	UpdateTime       types.Time   `json:"uTime"`
	STPMode          string       `json:"stpMode"`
}

// WsTickerSnapshotFutures contains information on a ticker snapshot
type WsTickerSnapshotFutures struct {
	InstrumentID  string       `json:"instId"`
	LastPrice     types.Number `json:"lastPr"`
	Open24H       types.Number `json:"open24h"`
	High24H       types.Number `json:"high24h"`
	Low24H        types.Number `json:"low24h"`
	Change24H     types.Number `json:"change24h"`
	BidPrice      types.Number `json:"bidPr"`
	AskPrice      types.Number `json:"askPr"`
	BidSize       types.Number `json:"bidSz"`
	AskSize       types.Number `json:"askSz"`
	FundingRate   types.Number `json:"fundingRate"`
	NextFunding   types.Time   `json:"nextFundingTime"`
	MarkPrice     types.Number `json:"markPrice"`
	IndexPrice    types.Number `json:"indexPrice"`
	HoldingAmount types.Number `json:"holdingAmount"`
	BaseVolume    types.Number `json:"baseVolume"`
	QuoteVolume   types.Number `json:"quoteVolume"`
	OpenUTC       types.Number `json:"openUtc"`
	SymbolType    uint8        `json:"symbolType,string"`
	Symbol        string       `json:"symbol"`
	DeliveryPrice types.Number `json:"deliveryPrice"`
	Timestamp     types.Time   `json:"ts"`
}

// WsAccountFuturesResponse contains information on an account response for futures trading
type WsAccountFuturesResponse struct {
	MarginCoin                   currency.Code `json:"marginCoin"`
	Frozen                       types.Number  `json:"frozen"`
	Available                    types.Number  `json:"available"`
	MaximumOpenPositionAvailable types.Number  `json:"maxOpenPositionAvailable"`
	MaximumTransferOut           types.Number  `json:"maxTransferOut"`
	Equity                       types.Number  `json:"equity"`
	USDTEquity                   types.Number  `json:"usdtEquity"`
}

// WsPositionResponse contains information on a position response
type WsPositionResponse struct {
	PositionID               int64         `json:"posId,string"`
	InstrumentID             string        `json:"instId"`
	MarginCoin               currency.Code `json:"marginCoin"`
	MarginSize               types.Number  `json:"marginSize"`
	MarginMode               string        `json:"marginMode"`
	HoldSide                 string        `json:"holdSide"`
	PositionMode             string        `json:"posMode"`
	Total                    types.Number  `json:"total"`
	Available                types.Number  `json:"available"`
	Frozen                   types.Number  `json:"frozen"`
	OpenPriceAverage         types.Number  `json:"openPriceAvg"`
	Leverage                 types.Number  `json:"leverage"`
	AchievedProfits          types.Number  `json:"achievedProfits"`
	UnrealizedProfitLoss     types.Number  `json:"unrealizedPL"`
	UnrealizedProfitLossRate types.Number  `json:"unrealizedPLR"`
	LiquidationPrice         types.Number  `json:"liquidationPrice"`
	KeepMarginRate           types.Number  `json:"keepMarginRate"`
	MarginRate               types.Number  `json:"marginRate"`
	CreationTime             types.Time    `json:"cTime"`
	BreakEvenPrice           types.Number  `json:"breakEvenPrice"`
	TotalFee                 types.Number  `json:"totalFee"`
	DeductedFee              types.Number  `json:"deductedFee"`
	UpdateTime               types.Time    `json:"uTime"`
	AutoMargin               string        `json:"autoMargin"`
}

// WsFillFuturesResponse contains information on a fill response for futures trading
type WsFillFuturesResponse struct {
	OrderID      int64               `json:"orderId,string"`
	TradeID      int64               `json:"tradeId,string"`
	Symbol       string              `json:"symbol"`
	Side         string              `json:"side"`
	OrderType    string              `json:"orderType"`
	PosMode      string              `json:"posMode"`
	Price        types.Number        `json:"price"`
	BaseVolume   types.Number        `json:"baseVolume"`
	QuoteVolume  types.Number        `json:"quoteVolume"`
	Profit       types.Number        `json:"profit"`
	TradeSide    string              `json:"tradeSide"`
	TradeScope   string              `json:"tradeScope"`
	FeeDetail    []AbridgedFeeDetail `json:"feeDetail"`
	CreationTime types.Time          `json:"cTime"`
	UpdateTime   types.Time          `json:"uTime"`
}

// WsOrderFuturesResponse contains information on an order response for futures trading
type WsOrderFuturesResponse struct {
	FilledQuantity         types.Number  `json:"accBaseVolume"`
	CreationTime           types.Time    `json:"cTime"`
	ClientOrderID          string        `json:"clientOid"`
	FeeDetail              []FeeAndCoin  `json:"feeDetail"`
	FillFee                types.Number  `json:"fillFee"`
	FillFeeCoin            currency.Code `json:"fillFeeCoin"`
	FillNotionalUSD        types.Number  `json:"fillNotionalUsd"`
	FillPrice              types.Number  `json:"fillPrice"`
	BaseVolume             types.Number  `json:"baseVolume"`
	FillTime               types.Time    `json:"fillTime"`
	Force                  string        `json:"force"`
	InstrumentID           string        `json:"instId"`
	Leverage               types.Number  `json:"leverage"`
	MarginCoin             currency.Code `json:"marginCoin"`
	MarginMode             string        `json:"marginMode"`
	NotionalUSD            types.Number  `json:"notionalUsd"`
	OrderID                int64         `json:"orderId,string"`
	OrderType              string        `json:"orderType"`
	ProfitAndLoss          types.Number  `json:"pnl"`
	PositionMode           string        `json:"posMode"`
	PositionSide           string        `json:"posSide"`
	Price                  types.Number  `json:"price"`
	PriceAverage           types.Number  `json:"priceAvg"`
	ReduceOnly             YesNoBool     `json:"reduceOnly"`
	STPMode                string        `json:"stpMode"`
	Side                   string        `json:"side"`
	Size                   types.Number  `json:"size"`
	EnterPointSource       string        `json:"enterPointSource"`
	Status                 string        `json:"status"`
	TradeScope             string        `json:"tradeScope"`
	TradeID                int64         `json:"tradeId,string"`
	TradeSide              string        `json:"tradeSide"`
	PresetStopSurplusPrice types.Number  `json:"presetStopSurplusPrice"`
	TotalProfits           types.Number  `json:"totalProfits"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice"`
	UpdateTime             types.Time    `json:"uTime"`
}

// WsTriggerOrderFuturesResponse contains information on a trigger order response for futures trading
type WsTriggerOrderFuturesResponse struct {
	InstrumentID           string        `json:"instId"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	TriggerPrice           types.Number  `json:"triggerPrice"`
	TriggerType            string        `json:"triggerType"`
	TriggerTime            types.Time    `json:"triggerTime"`
	PlanType               string        `json:"planType"`
	Price                  types.Number  `json:"price"`
	ExecutePrice           types.Number  `json:"executePrice"`
	Size                   types.Number  `json:"size"`
	ActualSize             types.Number  `json:"actualSize"`
	OrderType              string        `json:"orderType"`
	Side                   string        `json:"side"`
	TradeSide              string        `json:"tradeSide"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             currency.Code `json:"marginCoin"`
	Status                 string        `json:"status"`
	PositionMode           string        `json:"posMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	StopSurplusTriggerType string        `json:"stopSurplusTriggerType"`
	StopLossTriggerType    string        `json:"stopLossTriggerType"`
	STPMode                string        `json:"stpMode"`
	CreationTime           types.Time    `json:"cTime"`
	UpdateTime             types.Time    `json:"uTime"`
}

// WsPositionHistoryResponse contains information on a position history response
type WsPositionHistoryResponse struct {
	PositionID        int64         `json:"posId,string"`
	InstrumentID      string        `json:"instId"`
	MarginCoin        currency.Code `json:"marginCoin"`
	MarginMode        string        `json:"marginMode"`
	HoldSide          string        `json:"holdSide"`
	PositionMode      string        `json:"posMode"`
	OpenPriceAverage  types.Number  `json:"openPriceAvg"`
	ClosePriceAverage types.Number  `json:"closePriceAvg"`
	OpenSize          types.Number  `json:"openSize"`
	CloseSize         types.Number  `json:"closeSize"`
	AchievedProfits   types.Number  `json:"achievedProfits"`
	SettleFee         types.Number  `json:"settleFee"`
	OpenFee           types.Number  `json:"openFee"`
	CloseFee          types.Number  `json:"closeFee"`
	CreationTime      types.Time    `json:"cTime"`
	UpdateTime        types.Time    `json:"uTime"`
}

// WsIndexPriceResponse contains information on an index price response
type WsIndexPriceResponse struct {
	Symbol     string        `json:"symbol"`
	BaseCoin   currency.Code `json:"baseCoin"`
	QuoteCoin  currency.Code `json:"quoteCoin"`
	IndexPrice types.Number  `json:"indexPrice"`
	Timestamp  types.Time    `json:"ts"`
}

// WsAccountCrossMarginResponse contains information on an account response for cross margin trading
type WsAccountCrossMarginResponse struct {
	UpdateTime types.Time    `json:"uTime"`
	ID         int64         `json:"id,string"`
	Coin       currency.Code `json:"coin"`
	Available  types.Number  `json:"available"`
	Borrow     types.Number  `json:"borrow"`
	Frozen     types.Number  `json:"frozen"`
	Interest   types.Number  `json:"interest"`
	Coupon     types.Number  `json:"coupon"`
}

// WsOrderMarginResponse contains information on an order response for margin trading
type WsOrderMarginResponse struct {
	Force            string              `json:"force"`
	OrderType        string              `json:"orderType"`
	Price            types.Number        `json:"price"`
	QuoteSize        types.Number        `json:"quoteSize"`
	Side             string              `json:"side"`
	FeeDetail        []AbridgedFeeDetail `json:"feeDetail"`
	EnterPointSource string              `json:"enterPointSource"`
	Status           string              `json:"status"`
	BaseSize         types.Number        `json:"baseSize"`
	CreationTime     types.Time          `json:"cTime"`
	ClientOrderID    string              `json:"clientOid"`
	FillPrice        types.Number        `json:"fillPrice"`
	BaseVolume       types.Number        `json:"baseVolume"`
	FillTotalAmount  types.Number        `json:"fillTotalAmount"`
	LoanType         string              `json:"loanType"`
	OrderID          int64               `json:"orderId,string"`
	STPMode          string              `json:"stpMode"`
}

// WsAccountIsolatedMarginResponse contains information on an account response for isolated margin trading
type WsAccountIsolatedMarginResponse struct {
	UpdateTime types.Time    `json:"uTime"`
	ID         int64         `json:"id,string"`
	Coin       currency.Code `json:"coin"`
	Symbol     string        `json:"symbol"`
	Available  types.Number  `json:"available"`
	Borrow     types.Number  `json:"borrow"`
	Frozen     types.Number  `json:"frozen"`
	Interest   types.Number  `json:"interest"`
	Coupon     types.Number  `json:"coupon"`
}
