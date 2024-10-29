package bitget

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// Params is used within functions to make the setting of parameters easier
type Params struct {
	url.Values
}

// UnixTimestamp is a type used to unmarshal unix millisecond timestamps returned from the exchange
type UnixTimestamp time.Time

// UnixTimestampNumber is a type used to unmarshal unix millisecond timestamps returned from the exchange, when they
// aren't provided as strings
type UnixTimestampNumber time.Time

// AnnResp holds information on announcements
type AnnResp struct {
	AnnID    string        `json:"annId"`
	AnnTitle string        `json:"annTitle"`
	AnnDesc  string        `json:"annDesc"`
	CTime    UnixTimestamp `json:"cTime"`
	Language string        `json:"language"`
	AnnURL   string        `json:"annUrl"`
}

// TimeResp holds information on the current server time
type TimeResp struct {
	ServerTime UnixTimestamp `json:"serverTime"`
}

// TradeRateResp holds information on the current maker and taker fee rates
type TradeRateResp struct {
	MakerFeeRate float64 `json:"makerFeeRate,string"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
}

// SpotTrResp holds information on spot transactions
type SpotTrResp struct {
	ID          int64         `json:"id,string"`
	Coin        string        `json:"coin"`
	SpotTaxType string        `json:"spotTaxType"`
	Amount      float64       `json:"amount,string"`
	Fee         float64       `json:"fee,string"`
	Balance     float64       `json:"balance,string"`
	Timestamp   UnixTimestamp `json:"ts"`
}

// FutureTrResp holds information on futures transactions
type FutureTrResp struct {
	ID            int64         `json:"id,string"`
	Symbol        string        `json:"symbol"`
	MarginCoin    string        `json:"marginCoin"`
	FutureTaxType string        `json:"futureTaxType"`
	Amount        float64       `json:"amount,string"`
	Fee           float64       `json:"fee,string"`
	Timestamp     UnixTimestamp `json:"ts"`
}

// MarginTrResp holds information on margin transactions
type MarginTrResp struct {
	ID            int64         `json:"id,string"`
	Coin          string        `json:"coin"`
	Symbol        string        `json:"symbol"`
	MarginTaxType string        `json:"marginTaxType"`
	Amount        float64       `json:"amount,string"`
	Fee           float64       `json:"fee,string"`
	Total         float64       `json:"total,string"`
	Timestamp     UnixTimestamp `json:"ts"`
}

// P2PTrResp holds information on P2P transactions
type P2PTrResp struct {
	ID         int64         `json:"id,string"`
	Coin       string        `json:"coin"`
	P2PTaxType string        `json:"p2pTaxType"`
	Total      float64       `json:"total,string"`
	Timestamp  UnixTimestamp `json:"ts"`
}

// MerchantList is a sub-struct holding information on P2P merchants
type MerchantList struct {
	RegisterTime        UnixTimestamp `json:"registerTime"`
	NickName            string        `json:"nickName"`
	IsOnline            string        `json:"isOnline"`
	MerchantID          int64         `json:"merchantId,string"`
	AvgPaymentTime      int64         `json:"avgPaymentTime,string"`
	AvgReleaseTime      int64         `json:"avgReleaseTime,string"`
	TotalTrades         int64         `json:"totalTrades,string"`
	TotalBuy            int64         `json:"totalBuy,string"`
	TotalSell           int64         `json:"totalSell,string"`
	TotalCompletionRate float64       `json:"totalCompletionRate,string"`
	Trades30D           int64         `json:"trades30d,string"`
	Sell30D             float64       `json:"sell30d,string"`
	Buy30D              float64       `json:"buy30d,string"`
	CompletionRate30D   float64       `json:"completionRate30d,string"`
}

// P2PMerResp holds information on P2P merchant lists
type P2PMerListResp struct {
	MerchantList  []MerchantList `json:"merchantList"`
	MinMerchantID int64          `json:"minMerchantId,string"`
}

// YesNoBool is a type used to unmarshal strings that are either "yes" or "no" into bools
type YesNoBool bool

// P2PMerInfoResp holds information on P2P merchant information
type P2PMerInfoResp struct {
	RegisterTime        UnixTimestamp `json:"registerTime"`
	NickName            string        `json:"nickName"`
	MerchantID          int64         `json:"merchantId,string"`
	AvgPaymentTime      int64         `json:"avgPaymentTime,string"`
	AvgReleaseTime      int64         `json:"avgReleaseTime,string"`
	TotalTrades         int64         `json:"totalTrades,string"`
	TotalBuy            int64         `json:"totalBuy,string"`
	TotalSell           int64         `json:"totalSell,string"`
	TotalCompletionRate float64       `json:"totalCompletionRate,string"`
	Trades30D           int64         `json:"trades30d,string"`
	Sell30D             float64       `json:"sell30d,string"`
	Buy30D              float64       `json:"buy30d,string"`
	CompletionRate30D   float64       `json:"completionRate30d,string"`
	KYCStatus           YesNoBool     `json:"kycStatus"`
	EmailBindStatus     YesNoBool     `json:"emailBindStatus"`
	MobileBindStatus    YesNoBool     `json:"mobileBindStatus"`
	Email               string        `json:"email"`
	Mobile              string        `json:"mobile"`
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
	Count          float64       `json:"count,string"`
	FiatCurrency   string        `json:"fiat"`
	CryptoCurrency string        `json:"coin"`
	Price          float64       `json:"price,string"`
	WithdrawTime   UnixTimestamp `json:"withdrawTime"`
	RepresentTime  UnixTimestamp `json:"representTime"`
	ReleaseTime    UnixTimestamp `json:"releaseTime"`
	PaymentTime    UnixTimestamp `json:"paymentTime"`
	Amount         float64       `json:"amount,string"`
	Status         string        `json:"status"`
	BuyerRealName  string        `json:"buyerRealName"`
	SellerRealName string        `json:"sellerRealName"`
	CreationTime   UnixTimestamp `json:"ctime"`
	UpdateTime     UnixTimestamp `json:"utime"`
	PaymentInfo    PaymentInfo   `json:"paymentInfo"`
}

// P2POrdersResp holds information on P2P orders
type P2POrdersResp struct {
	OrderList  []P2POrderList `json:"orderList"`
	MinOrderID int64          `json:"minOrderId,string"`
}

// UserLimitList is a sub-struct holding information on P2P user limits
type UserLimitList struct {
	MinCompleteNum     int64     `json:"minCompleteNum,string"`
	MaxCompleteNum     int64     `json:"maxCompleteNum,string"`
	PlaceOrderNum      int64     `json:"placeOrderNum,string"`
	AllowMerchantPlace YesNoBool `json:"allowMerchantPlace"`
	CompleteRate30D    float64   `json:"completeRate30d,string"`
	Country            string    `json:"country"`
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
	AdID                  int64                   `json:"adId,string"`
	AdvNum                int64                   `json:"advNo,string"`
	Side                  string                  `json:"side"`
	AdSize                float64                 `json:"adSize,string"`
	Size                  float64                 `json:"size,string"`
	CryptoCurrency        string                  `json:"coin"`
	Price                 float64                 `json:"price,string"`
	CryptoPrecision       uint8                   `json:"coinPrecision,string"`
	FiatCurrency          string                  `json:"fiat"`
	FiatPrecision         uint8                   `json:"fiatPrecision,string"`
	FiatSymbol            string                  `json:"fiatSymbol"`
	Status                string                  `json:"status"`
	Hide                  YesNoBool               `json:"hide"`
	MaxTradeAmount        float64                 `json:"maxTradeAmount,string"`
	MinTradeAmount        float64                 `json:"minTradeAmount,string"`
	PayDuration           int64                   `json:"payDuration,string"`
	TurnoverNum           int64                   `json:"turnoverNum,string"`
	TurnoverRate          float64                 `json:"turnoverRate,string"`
	Label                 string                  `json:"label"`
	CreationTime          UnixTimestamp           `json:"ctime"`
	UpdateTime            UnixTimestamp           `json:"utime"`
	UserLimitList         UserLimitList           `json:"userLimitList"`
	PaymentMethodList     []PaymentMethodList     `json:"paymentMethodList"`
	MerchantCertifiedList []MerchantCertifiedList `json:"merchantCertifiedList"`
}

// P2PAdListResp holds information on P2P advertisements
type P2PAdListResp struct {
	AdvertisementList  []AdvertisementList `json:"advList"`
	MinAdvertisementID int64               `json:"minAdvId,string"`
}

// WhaleNetFlowResp holds information on whale trading volumes
type WhaleNetFlowResp struct {
	Volume float64       `json:"volume,string"`
	Date   UnixTimestamp `json:"date"`
}

// ActiveVolumeResp holds information on active trading volumes
type ActiveVolumeResp struct {
	BuyVolume  float64       `json:"buyVolume,string"`
	SellVolume float64       `json:"sellVolume,string"`
	Timestamp  UnixTimestamp `json:"ts"`
}

// PosRatFutureResp holds information on position ratios
type PosRatFutureResp struct {
	LongPositionRatio      float64       `json:"longPositionRatio,string"`
	ShortPositionRatio     float64       `json:"shortPositionRatio,string"`
	LongShortPositionRatio float64       `json:"longShortPositionRatio,string"`
	Timestamp              UnixTimestamp `json:"ts"`
}

// PosRatMarginResp holds information on position ratios in margin trading
type PosRatMarginResp struct {
	Timestamp      UnixTimestamp `json:"ts"`
	LongShortRatio float64       `json:"longShortRatio,string"`
}

// LoanGrowthResp holds information on loan growth
type LoanGrowthResp struct {
	Timestamp  UnixTimestamp `json:"ts"`
	GrowthRate float64       `json:"growthRate,string"`
}

// BorrowRatioResp holds information on borrowing ratios
type BorrowRatioResp struct {
	Timestamp  UnixTimestamp `json:"ts"`
	BorrowRate float64       `json:"borrowRate,string"`
}

// RatioResp holds information on ratios
type RatioResp struct {
	LongRatio      float64       `json:"longRatio,string"`
	ShortRatio     float64       `json:"shortRatio,string"`
	LongShortRatio float64       `json:"longShortRatio,string"`
	Timestamp      UnixTimestamp `json:"ts"`
}

// FundFlowResp holds information on fund flows
type FundFlowResp struct {
	WhaleBuyVolume    float64 `json:"whaleBuyVolume,string"`
	DolphinBuyVolume  float64 `json:"dolphinBuyVolume,string"`
	FishBuyVolume     float64 `json:"fishBuyVolume,string"`
	WhaleSellVolume   float64 `json:"whaleSellVolume,string"`
	DolphinSellVolume float64 `json:"dolphinSellVolume,string"`
	FishSellVolume    float64 `json:"fishSellVolume,string"`
	WhaleBuyRatio     float64 `json:"whaleBuyRatio,string"`
	DolphinBuyRatio   float64 `json:"dolphinBuyRatio,string"`
	FishBuyRatio      float64 `json:"fishBuyRatio,string"`
	WhaleSellRatio    float64 `json:"whaleSellRatio,string"`
	DolphinSellRatio  float64 `json:"dolphinSellRatio,string"`
	FishSellRatio     float64 `json:"fishSellRatio,string"`
}

// SymbolsResp holds information on supported symbols
type SymbolsResp struct {
	SpotList   []string `json:"spotList"`
	FutureList []string `json:"futureList"`
}

// WhaleFundFlowResp holds information on whale fund flows
type WhaleFundFlowResp struct {
	NetFlow   float64       `json:"netFlow,string"`
	Timestamp UnixTimestamp `json:"ts"`
}

// AccountRatioResp holds information on ratios
type AccountRatioResp struct {
	LongAccountRatio      float64       `json:"longAccountRatio,string"`
	ShortAccountRatio     float64       `json:"shortAccountRatio,string"`
	LongShortAccountRatio float64       `json:"longShortAccountRatio,string"`
	Timestamp             UnixTimestamp `json:"ts"`
}

// FailureList is a sub-struct holding information on failures
type FailureList struct {
	SubaccountName string `json:"subaAccountName"`
}

// SuccessList is a sub-struct holding information on successes
type SuccessList struct {
	SubaccountUID  string        `json:"subAccountUid"`
	SubaccountName string        `json:"subaAccountName"`
	Status         string        `json:"status"`
	PermList       []string      `json:"permList"`
	Label          string        `json:"label"`
	CreationTime   UnixTimestamp `json:"cTime"`
	UpdateTime     UnixTimestamp `json:"uTime"`
}

// CrVirSubResp contains information returned when creating virtual sub-accounts
type CrVirSubResp struct {
	FailureList []FailureList `json:"failureList"`
	SuccessList []SuccessList `json:"successList"`
}

// SuccessBool is a type used to unmarshal strings that are either "success" or "failure" into bools
type SuccessBool bool

// CrSubAccAPIKeyResp contains information returned when simultaneously creating a sub-account and
// an API key
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
	SubaccountUID  string        `json:"subAccountUid"`
	SubaccountName string        `json:"subAccountName"`
	Label          string        `json:"label"`
	Status         string        `json:"status"`
	PermList       []string      `json:"permList"`
	CreationTime   UnixTimestamp `json:"cTime"`
	UpdateTime     UnixTimestamp `json:"uTime"`
}

// GetVirSubResp contains information on the user's virtual sub-accounts
type GetVirSubResp struct {
	SubaccountList []SubaccountList `json:"subAccountList"`
	EndID          int64            `json:"endId,string"`
}

// AlterAPIKeyResp contains information returned when creating or modifying an API key
type AlterAPIKeyResp struct {
	SubaccountUID    string   `json:"subAccountUid"`
	SubaccountApiKey string   `json:"subAccountApiKey"`
	SecretKey        string   `json:"secretKey"`
	PermList         []string `json:"permList"`
	Label            string   `json:"label"`
	IPList           []string `json:"ipList"`
}

// GetAPIKeyResp contains information on the user's API keys
type GetAPIKeyResp struct {
	SubaccountUID    string   `json:"subAccountUid"`
	SubaccountApiKey string   `json:"subAccountApiKey"`
	IPList           []string `json:"ipList"`
	PermList         []string `json:"permList"`
	Label            string   `json:"label"`
}

// FundingAssetsResp contains information on the user's funding assets
type FundingAssetsResp struct {
	Coin      string  `json:"coin"`
	Available float64 `json:"available,string"`
	Frozen    float64 `json:"frozen,string"`
	USDTValue float64 `json:"usdtValue,string"`
}

// BotAccAssetsResp contains information on the user's bot account assets
type BotAccAssetsResp struct {
	Coin      string  `json:"coin"`
	Available float64 `json:"available,string"`
	Equity    float64 `json:"equity,string"`
	Bonus     float64 `json:"bonus,string"`
	Frozen    float64 `json:"frozen,string"`
	USDTValue float64 `json:"usdtValue,string"`
}

// AssetOverviewResp contains information on the user's assets
type AssetOverviewResp struct {
	AccountType string  `json:"accountType"`
	USDTBalance float64 `json:"usdtBalance,string"`
}

// ConvertCoinsResp contains information on the user's available currencies
type ConvertCoinsResp struct {
	Coin      string  `json:"coin"`
	Available float64 `json:"available,string"`
	MaxAmount float64 `json:"maxAmount,string"`
	MinAmount float64 `json:"minAmount,string"`
}

// QuotedPriceResp contains information on a queried conversion
type QuotedPriceResp struct {
	FromCoin     string  `json:"fromCoin"`
	FromCoinSize float64 `json:"fromCoinSize,string"`
	ConvertPrice float64 `json:"cnvtPrice,string"`
	ToCoin       string  `json:"toCoin"`
	ToCoinSize   float64 `json:"toCoinSize,string"`
	TraceID      string  `json:"traceId"`
	Fee          float64 `json:"fee,string"`
}

// CommitConvResp contains information on a committed conversion
type CommitConvResp struct {
	ToCoin       string        `json:"toCoin"`
	ToCoinSize   float64       `json:"toCoinSize,string"`
	ConvertPrice float64       `json:"cnvtPrice,string"`
	Timestamp    UnixTimestamp `json:"ts"`
}

// DataList is a sub-struct holding information on the user's conversion history
type DataList struct {
	ID           int64         `json:"id,string"`
	Timestamp    UnixTimestamp `json:"ts"`
	ConvertPrice float64       `json:"cnvtPrice,string"`
	Fee          float64       `json:"fee,string"`
	FromCoinSize float64       `json:"fromCoinSize,string"`
	FromCoin     string        `json:"fromCoin"`
	ToCoinSize   float64       `json:"toCoinSize,string"`
	ToCoin       string        `json:"toCoin"`
}

// ConvHistResp contains information on the user's conversion history
type ConvHistResp struct {
	DataList []DataList `json:"dataList"`
	EndID    int64      `json:"endId,string"`
}

// FeeAndRate is a sub-struct holding information on fees
type FeeAndRate struct {
	FeeRate float64 `json:"feeRate,string"`
	Fee     float64 `json:"fee,string"`
}

// BGBConvertCoinsResp contains information on the user's available currencies and conversions between those
// and BGB
type BGBConvertCoinsResp struct {
	Coin         string        `json:"coin"`
	Available    float64       `json:"available,string"`
	BGBEstAmount float64       `json:"bgbEstAmount,string"`
	Precision    uint8         `json:"precision"`
	FeeDetail    []FeeAndRate  `json:"feeDetail"`
	CurrentTime  UnixTimestamp `json:"cTime"`
}

// ConvertBGBResp contains information on a series of conversions between BGB and other currencies
type ConvertBGBResp struct {
	Coin    string `json:"coin"`
	OrderID int64  `json:"orderId,string"`
}

// FeeAndCoin is a sub-struct holding information on fees
type FeeAndCoin struct {
	FeeCoin string  `json:"feeCoin"`
	Fee     float64 `json:"fee,string"`
}

// BGBConvHistResp contains information on the user's conversion history between BGB and other currencies
type BGBConvHistResp struct {
	OrderID       int64         `json:"orderId,string"`
	FromCoin      string        `json:"fromCoin"`
	FromAmount    float64       `json:"fromAmount,string"`
	FromCoinPrice float64       `json:"fromCoinPrice,string"`
	ToCoin        string        `json:"toCoin"`
	ToAmount      float64       `json:"toAmount,string"`
	ToCoinPrice   float64       `json:"toCoinPrice,string"`
	FeeDetail     []FeeAndCoin  `json:"feeDetail"`
	Status        SuccessBool   `json:"status"`
	CreationTime  UnixTimestamp `json:"cTime"`
}

// ChainInfo is a sub-struct containing information on supported chains for a currency
type ChainInfo struct {
	Chain                string  `json:"chain"`
	NeedTag              bool    `json:"needTag,string"`
	Withdrawable         bool    `json:"withdrawable,string"`
	Rechargeable         bool    `json:"rechargeable,string"`
	WithdrawFee          float64 `json:"withdrawFee,string"`
	ExtraWithdrawFee     float64 `json:"extraWithdrawFee,string"`
	DepositConfirm       uint16  `json:"depositConfirm,string"`
	WithdrawConfirm      uint16  `json:"withdrawConfirm,string"`
	MinDepositAmount     float64 `json:"minDepositAmount,string"`
	MinWithdrawAmount    float64 `json:"minWithdrawAmount,string"`
	BrowserURL           string  `json:"browserUrl"`
	ContractAddress      string  `json:"contractAddress"`
	WithdrawStep         uint8   `json:"withdrawStep,string"`
	WithdrawMinimumScale uint8   `json:"withdrawMinimumScale,string"`
	Congestion           string  `json:"congestion"`
}

// CoinInfoResp contains information on supported spot currencies
type CoinInfoResp struct {
	CoinID   uint32      `json:"coinId,string"`
	Coin     string      `json:"coin"`
	Transfer bool        `json:"transfer,string"`
	Chains   []ChainInfo `json:"chains"`
}

// SymbolInfoResp contains information on supported spot trading pairs
type SymbolInfoResp struct {
	Symbol              string       `json:"symbol"`
	BaseCoin            string       `json:"baseCoin"`
	QuoteCoin           string       `json:"quoteCoin"`
	MinTradeAmount      float64      `json:"minTradeAmount,string"`
	MaxTradeAmount      float64      `json:"maxTradeAmount,string"`
	TakerFeeRate        float64      `json:"takerFeeRate,string"`
	MakerFeeRate        float64      `json:"makerFeeRate,string"`
	PricePrecision      uint8        `json:"pricePrecision,string"`
	QuantityPrecision   uint8        `json:"quantityPrecision,string"`
	QuotePrecision      uint8        `json:"quotePrecision,string"`
	MinTradeUSDT        float64      `json:"minTradeUSDT,string"`
	Status              string       `json:"status"`
	BuyLimitPriceRatio  types.Number `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio types.Number `json:"sellLimitPriceRatio"`
}

// VIPFeeRateResp contains information on the different levels of VIP fee rates
type VIPFeeRateResp struct {
	Level        uint8   `json:"level,string"`
	DealAmount   float64 `json:"dealAmount,string"`
	AssetAmount  float64 `json:"assetAmount,string"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
	MakerFeeRate float64 `json:"makerFeeRate,string"`
	// 24-hour withdrawal limits
	BTCWithdrawAmount float64 `json:"btcWithdrawAmount,string"`
	USDWithdrawAmount float64 `json:"usdWithdrawAmount,string"`
}

// TickerResp contains information on tickers
type TickerResp struct {
	Symbol       string        `json:"symbol"`
	High24H      float64       `json:"high24h,string"`
	Open         float64       `json:"open,string"`
	LastPrice    float64       `json:"lastPr,string"`
	Low24H       float64       `json:"low24h,string"`
	QuoteVolume  float64       `json:"quoteVolume,string"`
	BaseVolume   float64       `json:"baseVolume,string"`
	USDTVolume   float64       `json:"usdtVolume,string"`
	BidPrice     float64       `json:"bidPr,string"`
	AskPrice     float64       `json:"askPr,string"`
	BidSize      float64       `json:"bidSz,string"`
	AskSize      float64       `json:"askSz,string"`
	OpenUTC      float64       `json:"openUTC,string"`
	Timestamp    UnixTimestamp `json:"ts"`
	ChangeUTC24H float64       `json:"changeUTC24h,string"`
	Change24H    float64       `json:"change24h,string"`
}

// DepthResp contains information on orderbook bids and asks, and any merging of orders done to them
type DepthResp struct {
	Asks           [][2]float64  `json:"asks"`
	Bids           [][2]float64  `json:"bids"`
	Precision      string        `json:"precision"`
	Scale          float64       `json:"scale,string"`
	IsMaxPrecision YesNoBool     `json:"isMaxPrecision"`
	Timestamp      UnixTimestamp `json:"ts"`
}

// OrderbookResp contains information on orderbook bids and asks
type OrderbookResp struct {
	Asks      [][2]types.Number `json:"asks"`
	Bids      [][2]types.Number `json:"bids"`
	Timestamp UnixTimestamp     `json:"ts"`
}

// CandleResponse contains unsorted candle data
type CandleResponse struct {
	Data [][8]any `json:"data"`
}

// OneSpotCandle contains a single candle
type OneSpotCandle struct {
	Timestamp   time.Time
	Open        float64
	High        float64
	Low         float64
	Close       float64
	BaseVolume  float64
	QuoteVolume float64
	USDTVolume  float64
}

// OneFuturesCandle contains a single candle
type OneFuturesCandle struct {
	Timestamp   time.Time
	Entry       float64
	High        float64
	Low         float64
	Exit        float64
	BaseVolume  float64
	QuoteVolume float64
}

// CandleData contains sorted candle data
type CandleData struct {
	SpotCandles    []OneSpotCandle
	FuturesCandles []OneFuturesCandle
}

// MarketFillsResp contains information on a batch of trades
type MarketFillsResp struct {
	Symbol    string        `json:"symbol"`
	TradeID   int64         `json:"tradeId,string"`
	Side      string        `json:"side"`
	Price     float64       `json:"price,string"`
	Size      float64       `json:"size,string"`
	Timestamp UnixTimestamp `json:"ts"`
}

// PlaceOrderStruct contains information on an order to be placed
type PlaceSpotOrderStruct struct {
	Pair                   string  `json:"symbol"`
	Side                   string  `json:"side"`
	OrderType              string  `json:"orderType"`
	Strategy               string  `json:"force"`
	Price                  float64 `json:"price,string"`
	Size                   float64 `json:"size,string"`
	ClientOrderID          string  `json:"clientOId,omitempty"`
	STPMode                string  `json:"stpMode"`
	PresetTakeProfitPrice  float64 `json:"presetTakeProfitPrice,string,omitempty"`
	ExecuteTakeProfitPrice float64 `json:"executeTakeProfitPrice,string,omitempty"`
	PresetStopLossPrice    float64 `json:"presetStopLossPrice,string,omitempty"`
	ExecuteStopLossPrice   float64 `json:"executeStopLossPrice,string,omitempty"`
}

// CancelSpotOrderStruct contains information on an order to be cancelled
type CancelSpotOrderStruct struct {
	Pair          string `json:"symbol"`
	OrderID       int64  `json:"orderId,string,omitempty"`
	ClientOrderID string `json:"clientOId,omitempty"`
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
	UserID           string          `json:"userId"` // Check whether this should be a different type
	Symbol           string          `json:"symbol"`
	OrderID          EmptyInt        `json:"orderId"`
	ClientOrderID    string          `json:"clientOid"`
	Price            float64         `json:"price,string"`
	Size             float64         `json:"size,string"`
	OrderType        string          `json:"orderType"`
	Side             string          `json:"side"`
	Status           string          `json:"status"`
	PriceAverage     float64         `json:"priceAvg,string"`
	BaseVolume       float64         `json:"baseVolume,string"`
	QuoteVolume      float64         `json:"quoteVolume,string"`
	EnterPointSource string          `json:"enterPointSource"`
	CreationTime     UnixTimestamp   `json:"cTime"`
	UpdateTime       UnixTimestamp   `json:"uTime"`
	OrderSource      string          `json:"orderSource"`
	FeeDetailTemp    json.RawMessage `json:"feeDetail"`
}

// FeeDetail contains information on fees
type FeeDetail struct {
	AmountCoupons     float64 `json:"c"`
	AmountDeductedBGB float64 `json:"d"`
	AmountRemaining   float64 `json:"r"`
	AmountTotal       float64 `json:"t"`
	Deduction         bool    `json:"deduction"`
	FeeCoinCode       string  `json:"feeCoinCode"`
	TotalFee          float64 `json:"totalFee"`
	TotalDeductionFee float64 `json:"totalDeductionFee"`
}

// FeeDetailStore is a map of fee details for better unmarshalling
type FeeDetailStore map[string]FeeDetail

// SpotOrderDetailData contains information on an order for better unmarshalling
type SpotOrderDetailData struct {
	UserID           string // Check whether this should be a different type
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
	CreationTime     UnixTimestamp
	UpdateTime       UnixTimestamp
	OrderSource      string
	FeeDetail        FeeDetailStore
}

// UnfilledOrdersResp contains information on the user's unfilled orders
type UnfilledOrdersResp struct {
	UserID           string        `json:"userId"` // Check whether this should be a different type
	Symbol           string        `json:"symbol"`
	OrderID          EmptyInt      `json:"orderId"`
	ClientOrderID    string        `json:"clientOid"`
	PriceAverage     float64       `json:"priceAvg,string"`
	Size             float64       `json:"size,string"`
	OrderType        string        `json:"orderType"`
	Side             string        `json:"side"`
	Status           string        `json:"status"`
	BasePrice        float64       `json:"basePrice,string"`
	BaseVolume       float64       `json:"baseVolume,string"`
	QuoteVolume      float64       `json:"quoteVolume,string"`
	EnterPointSource string        `json:"enterPointSource"`
	OrderSource      string        `json:"orderSource"`
	CreationTime     UnixTimestamp `json:"cTime"`
	UpdateTime       UnixTimestamp `json:"uTime"`
}

// AbridgedFeeDetail contains some information on fees
type AbridgedFeeDetail struct {
	Deduction         YesNoBool    `json:"deduction"`
	FeeCoin           string       `json:"feeCoin"`
	TotalDeductionFee types.Number `json:"totalDeductionFee"`
	TotalFee          float64      `json:"totalFee,string"`
}

// SpotFillsResp contains information on the user's fulfilled orders
type SpotFillsResp struct {
	UserID       string            `json:"userId"` // Check whether this should be a different type
	Symbol       string            `json:"symbol"`
	OrderID      EmptyInt          `json:"orderId"`
	TradeID      int64             `json:"tradeId,string"`
	OrderType    string            `json:"orderType"`
	Side         string            `json:"side"`
	PriceAverage float64           `json:"priceAvg,string"`
	Size         float64           `json:"size,string"`
	Amount       float64           `json:"amount,string"`
	FeeDetail    AbridgedFeeDetail `json:"feeDetail"`
	TradeScope   string            `json:"tradeScope"`
	CreationTime UnixTimestamp     `json:"cTime"`
	UpdateTime   UnixTimestamp     `json:"uTime"`
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
	Pair                   string  `json:"symbol"`
	Price                  float64 `json:"price,string"`
	Amount                 float64 `json:"size,string"`
	OldClientOrderID       string  `json:"clientOid,omitempty"`
	OrderID                int64   `json:"orderId,string,omitempty"`
	NewClientOrderID       string  `json:"newClientOid,omitempty"`
	PresetTakeProfitPrice  float64 `json:"presetTakeProfitPrice,string,omitempty"`
	ExecuteTakeProfitPrice float64 `json:"executeTakeProfitPrice,string,omitempty"`
	PresetStopLossPrice    float64 `json:"presetStopLossPrice,string,omitempty"`
	ExecuteStopLossPrice   float64 `json:"executeStopLossPrice,string,omitempty"`
}

// PlanSpotOrder is a sub-struct that contains information on a planned order
type PlanSpotOrder struct {
	OrderID          int64         `json:"orderId,string"`
	ClientOrderID    string        `json:"clientOid"`
	Symbol           string        `json:"symbol"`
	TriggerPrice     float64       `json:"triggerPrice,string"`
	OrderType        string        `json:"orderType"`
	ExecutePrice     types.Number  `json:"executePrice"`
	PlanType         string        `json:"planType"`
	Size             float64       `json:"size,string"`
	Status           string        `json:"status"`
	Side             string        `json:"side"`
	TriggerType      string        `json:"triggerType"`
	EnterPointSource string        `json:"enterPointSource"`
	CreationTime     UnixTimestamp `json:"cTime"`
	UpdateTime       UnixTimestamp `json:"uTime"`
}

// PlanSpotOrderResp contains information on plan orders
type PlanSpotOrderResp struct {
	NextFlag   bool            `json:"nextFlag"`
	IDLessThan EmptyInt        `json:"idLessThan"`
	OrderList  []PlanSpotOrder `json:"orderList"`
}

// SubOrderResp contains information on sub-orders
type SubOrderResp struct {
	OrderID int64   `json:"orderId,string"`
	Price   float64 `json:"price,string"`
	Type    string  `json:"type"`
	Status  string  `json:"status"`
}

// AccountInfoResp contains information on the user's account
type AccountInfoResp struct {
	UserID       int64         `json:"userId,string"`
	InviterID    int64         `json:"inviterId,string"`
	ChannelCode  string        `json:"channelCode"`
	Channel      string        `json:"channel"`
	IPs          string        `json:"ips"`
	Authorities  []string      `json:"authorities"`
	ParentID     int64         `json:"parentId"`
	TraderType   string        `json:"traderType"`
	RegisterTime UnixTimestamp `json:"regisTime"`
}

// AssetData contains information on the amount of an assset an account owns
type AssetData struct {
	Coin           string        `json:"coin"`
	Available      float64       `json:"available,string"`
	Frozen         float64       `json:"frozen,string"`
	Locked         float64       `json:"locked,string"`
	LimitAvailable float64       `json:"limitAvailable,string"`
	UpdateTime     UnixTimestamp `json:"uTime"`
}

// SubaccountAssetsResp contains information on assets in a user's sub-accounts
type SubaccountAssetsResp struct {
	UserID     int64       `json:"userId,string"`
	AssetsList []AssetData `json:"assetsList"`
}

// SuccessBoolResp2 contains a success bool in a secondary format returned by the exchange
type SuccessBoolResp2 struct {
	Success SuccessBool `json:"data"`
}

// SpotAccBillResp contains information on the user's billing history
type SpotAccBillResp struct {
	CreationTime UnixTimestamp `json:"cTime"`
	Coin         string        `json:"coin"`
	GroupType    string        `json:"groupType"`
	BusinessType string        `json:"businessType"`
	Size         float64       `json:"size,string"`
	Balance      float64       `json:"balance,string"`
	Fees         float64       `json:"fees,string"`
	BillID       int64         `json:"billId,string"`
}

// TransferResp contains information on an asset transfer
type TransferResp struct {
	TransferID    int64  `json:"transferId,string"`
	ClientOrderID string `json:"clientOid"`
}

// SubaccTfrRecResp contains detailed information on asset transfers between sub-accounts
type SubaccTfrRecResp struct {
	Coin          string        `json:"coin"`
	Status        string        `json:"status"`
	ToType        string        `json:"toType"`
	FromType      string        `json:"fromType"`
	Size          float64       `json:"size,string"`
	Timestamp     UnixTimestamp `json:"ts"`
	ClientOrderID string        `json:"clientOid"`
	TransferID    int64         `json:"transferId,string"`
	FromUserID    int64         `json:"fromUserId,string"`
	ToUserID      int64         `json:"toUserId,string"`
}

// TransferRecResp contains detailed information on asset transfers
type TransferRecResp struct {
	Coin          string        `json:"coin"`
	Status        string        `json:"status"`
	ToType        string        `json:"toType"`
	ToSymbol      string        `json:"toSymbol"`
	FromType      string        `json:"fromType"`
	FromSymbol    string        `json:"fromSymbol"`
	Size          float64       `json:"size,string"`
	Timestamp     UnixTimestamp `json:"ts"`
	ClientOrderID string        `json:"clientOid"`
	TransferID    int64         `json:"transferId,string"`
}

// DepositAddressResp contains information on a deposit address
type DepositAddressResp struct {
	Address string `json:"address"`
	Chain   string `json:"chain"`
	Coin    string `json:"coin"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// SubaccDepRecResp contains detailed information on deposits to sub-accounts
type SubaccDepRecResp struct {
	OrderID      int64         `json:"orderId,string"`
	TradeID      int64         `json:"tradeId,string"`
	Coin         string        `json:"coin"`
	Size         float64       `json:"size,string"`
	Status       string        `json:"status"`
	FromAddress  string        `json:"fromAddress"`
	ToAddress    string        `json:"toAddress"`
	Chain        string        `json:"chain"`
	Destination  string        `json:"dest"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
}

// WithdrawRecordsResp contains detailed information on withdrawals
type WithdrawRecordsResp struct {
	OrderID       int64         `json:"orderId,string"`
	TradeID       int64         `json:"tradeId,string"`
	Coin          string        `json:"coin"`
	ClientOrderID string        `json:"clientOid"`
	OrderType     string        `json:"type"`
	Destination   string        `json:"dest"`
	Size          float64       `json:"size,string"`
	Fee           float64       `json:"fee,string"`
	Status        string        `json:"status"`
	FromAddress   string        `json:"fromAddress"`
	ToAddress     string        `json:"toAddress"`
	Chain         string        `json:"chain"`
	Confirm       uint32        `json:"confirm,string"`
	Tag           string        `json:"tag"`
	CreationTime  UnixTimestamp `json:"cTime"`
	UpdateTime    UnixTimestamp `json:"uTime"`
}

// CryptoDepRecResp contains detailed information on cryptocurrency deposits
type CryptoDepRecResp struct {
	OrderID      int64         `json:"orderId,string"`
	TradeID      int64         `json:"tradeId,string"`
	Coin         string        `json:"coin"`
	OrderType    string        `json:"type"`
	Size         float64       `json:"size,string"`
	Status       string        `json:"status"`
	FromAddress  string        `json:"fromAddress"`
	ToAddress    string        `json:"toAddress"`
	Chain        string        `json:"chain"`
	Destination  string        `json:"dest"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
}

// FutureTickerResp contains information on a futures ticker
type FutureTickerResp struct {
	Symbol            string        `json:"symbol"`
	LastPrice         float64       `json:"lastPr,string"`
	AskPrice          float64       `json:"askPr,string"`
	BidPrice          float64       `json:"bidPr,string"`
	BidSize           float64       `json:"bidSz,string"`
	AskSize           float64       `json:"askSz,string"`
	High24H           float64       `json:"high24h,string"`
	Low24H            float64       `json:"low24h,string"`
	Timestamp         UnixTimestamp `json:"ts"`
	Change24H         float64       `json:"change24h,string"`
	BaseVolume        float64       `json:"baseVolume,string"`
	QuoteVolume       float64       `json:"quoteVolume,string"`
	USDTVolume        float64       `json:"usdtVolume,string"`
	OpenUTC           float64       `json:"openUtc,string"`
	ChangeUTC24H      float64       `json:"changeUtc24h,string"`
	IndexPrice        float64       `json:"indexPrice,string"`
	FundingRate       float64       `json:"fundingRate,string"`
	HoldingAmount     float64       `json:"holdingAmount,string"`
	DeliveryStartTime UnixTimestamp `json:"deliveryStartTime"`
	DeliveryTime      UnixTimestamp `json:"deliveryTime"`
	DeliveryStatus    string        `json:"deliveryStatus"`
	Open24H           float64       `json:"open24h,string"`
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
	Symbol string  `json:"symbol"`
	Size   float64 `json:"size,string"`
}

// OpenPositionResp contains information on open positions
type OpenPositionsResp struct {
	OpenInterestList []OpenInterestList `json:"openInterestList"`
	Timestamp        UnixTimestamp      `json:"ts"`
}

// FundingTimeResp contains information on funding times
type FundingTimeResp struct {
	Symbol          string        `json:"symbol"`
	NextFundingTime UnixTimestamp `json:"nextFundingTime"`
	RatePeriod      uint16        `json:"ratePeriod,string"`
}

// FuturesPriceResp contains information on futures prices
type FuturesPriceResp struct {
	Symbol     string        `json:"symbol"`
	Price      float64       `json:"price,string"`
	IndexPrice float64       `json:"indexPrice,string"`
	MarkPrice  float64       `json:"markPrice,string"`
	Timestamp  UnixTimestamp `json:"ts"`
}

// FundingHistoryResp contains information on funding history
type FundingHistoryResp struct {
	Symbol      string        `json:"symbol"`
	FundingRate float64       `json:"fundingRate,string"`
	FundingTime UnixTimestamp `json:"fundingTime"`
}

// FundingCurrentResp contains information on current funding rates
type FundingCurrentResp struct {
	Symbol      string  `json:"symbol"`
	FundingRate float64 `json:"fundingRate,string"`
}

// ContractConfigResp contains information on contract details
type ContractConfigResp struct {
	Symbol                string        `json:"symbol"`
	BaseCoin              string        `json:"baseCoin"`
	QuoteCoin             string        `json:"quoteCoin"`
	BuyLimitPriceRatio    float64       `json:"buyLimitPriceRatio,string"`
	SellLimitPriceRatio   float64       `json:"sellLimitPriceRatio,string"`
	FeeRateUpRatio        float64       `json:"feeRateUpRatio,string"`
	MakerFeeRate          float64       `json:"makerFeeRate,string"`
	TakerFeeRate          float64       `json:"takerFeeRate,string"`
	OpenCostUpRatio       float64       `json:"openCostUpRatio,string"`
	SupportMarginCoins    []string      `json:"supportMarginCoins"`
	MinTradeNum           float64       `json:"minTradeNum,string"`
	PriceEndStep          float64       `json:"priceEndStep,string"`
	VolumePlace           float64       `json:"volumePlace,string"`
	PricePlace            float64       `json:"pricePlace,string"`
	SizeMultiplier        float64       `json:"sizeMultiplier,string"`
	SymbolType            string        `json:"symbolType"`
	MinTradeUSDT          float64       `json:"minTradeUSDT,string"`
	MaxSymbolOrderNum     int64         `json:"maxSymbolOrderNum,string"`
	MaxSymbolOpenOrderNum int64         `json:"maxSymbolOpenOrderNum,string"`
	MaxPositionNum        int64         `json:"maxPositionNum,string"`
	SymbolStatus          string        `json:"symbolStatus"`
	OffTime               int64         `json:"offTime,string"`
	LimitOpenTime         int64         `json:"limitOpenTime,string"`
	DeliveryTime          UnixTimestamp `json:"deliveryTime"`
	DeliveryStartTime     UnixTimestamp `json:"deliveryStartTime"`
	DeliveryPeriod        string        `json:"deliveryPeriod"`
	LaunchTime            UnixTimestamp `json:"launchTime"`
	FundInterval          EmptyInt      `json:"fundInterval"`
	MinLever              float64       `json:"minLever,string"`
	MaxLever              float64       `json:"maxLever,string"`
	PosLimit              float64       `json:"posLimit,string"`
	MaintainTime          UnixTimestamp `json:"maintainTime"`
}

// OneAccResp contains information on a single account
type OneAccResp struct {
	MarginCoin            string       `json:"marginCoin"`
	Locked                float64      `json:"locked,string"`
	Available             float64      `json:"available,string"`
	CrossedMaxAvailable   float64      `json:"crossedMaxAvailable,string"`
	IsolatedMaxAvailable  float64      `json:"isolatedMaxAvailable,string"`
	MaxTransferOut        float64      `json:"maxTransferOut,string"`
	AccountEquity         float64      `json:"accountEquity,string"`
	USDTEquity            float64      `json:"usdtEquity,string"`
	BTCEquity             float64      `json:"btcEquity,string"`
	CrossedRiskRate       float64      `json:"crossedRiskRate,string"`
	CrossedMarginleverage float64      `json:"crossedMarginleverage"`
	IsolatedLongLever     float64      `json:"isolatedLongLever"`
	IsolatedShortLever    float64      `json:"isolatedShortLever"`
	MarginMode            string       `json:"marginMode"`
	PositionMode          string       `json:"posMode"`
	UnrealizedPL          types.Number `json:"unrealizedPL"`
	Coupon                types.Number `json:"coupon,string"`
	CrossedUnrealizedPL   types.Number `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedPL  types.Number `json:"isolatedUnrealizedPL"`
}

// FutureAccDetails contains information on a user's futures account
type FutureAccDetails struct {
	MarginCoin           string       `json:"marginCoin"`
	Locked               float64      `json:"locked,string"`
	Available            float64      `json:"available,string"`
	CrossedMaxAvailable  float64      `json:"crossedMaxAvailable,string"`
	IsolatedMaxAvailable float64      `json:"isolatedMaxAvailable,string"`
	MaxTransferOut       float64      `json:"maxTransferOut,string"`
	AccountEquity        float64      `json:"accountEquity,string"`
	USDTEquity           float64      `json:"usdtEquity,string"`
	BTCEquity            float64      `json:"btcEquity,string"`
	CrossedRiskRate      float64      `json:"crossedRiskRate,string"`
	UnrealizedPL         types.Number `json:"unrealizedPL"`
	Coupon               types.Number `json:"coupon"`
	CrossedUnrealizedPL  types.Number `json:"crossedUnrealizedPL"`
	IsolatedUnrealizedPL types.Number `json:"isolatedUnrealizedPL"`
}

// SubaccountFuturesResp contains information on futures details of a user's sub-accounts
type SubaccountFuturesResp struct {
	UserID    int64              `json:"userId"`
	AssetList []FutureAccDetails `json:"assetList"`
}

// LeverageResp contains information on the leverage of a position
type LeverageResp struct {
	Symbol              string       `json:"symbol"`
	MarginCoin          string       `json:"marginCoin"`
	LongLeverage        float64      `json:"longLeverage,string"`
	ShortLeverage       float64      `json:"shortLeverage,string"`
	CrossMarginLeverage types.Number `json:"crossMarginLeverage"`
	MarginMode          string       `json:"marginMode"`
}

// FutureAccBillResp contains information on futures billing history
type FutureAccBillResp struct {
	OrderID      int64         `json:"orderId,string"`
	Symbol       string        `json:"symbol"`
	Amount       float64       `json:"amount,string"`
	Fee          float64       `json:"fee,string"`
	FeeByCoupon  types.Number  `json:"feeByCoupon"`
	FeeCoin      string        `json:"feeCoin"`
	BusinessType string        `json:"businessType"`
	Coin         string        `json:"coin"`
	CreationTime UnixTimestamp `json:"cTime"`
}

// PositionTierResp contains information on position configurations
type PositionTierResp struct {
	Symbol         string  `json:"symbol"`
	Level          uint8   `json:"level,string"`
	StartUnit      float64 `json:"startUnit,string"`
	EndUnit        float64 `json:"endUnit,string"`
	Leverage       float64 `json:"leverage,string"`
	KeepMarginRate float64 `json:"keepMarginRate,string"`
}

// PositionResp contains information on positions
type PositionResp struct {
	MarginCoin       string        `json:"marginCoin"`
	Symbol           string        `json:"symbol"`
	HoldSide         string        `json:"holdSide"`
	OpenDelegateSize float64       `json:"openDelegateSize,string"`
	MarginSize       float64       `json:"marginSize,string"`
	Available        float64       `json:"available,string"`
	Locked           float64       `json:"locked,string"`
	Total            float64       `json:"total,string"`
	Leverage         float64       `json:"leverage,string"`
	AchievedProfits  float64       `json:"achievedProfits,string"`
	OpenPriceAverage float64       `json:"openPriceAvg,string"`
	MarginMode       string        `json:"marginMode"`
	PositionMode     string        `json:"posMode"`
	UnrealizedPL     float64       `json:"unrealizedPL,string"`
	LiquidationPrice float64       `json:"liquidationPrice,string"`
	KeepMarginRate   float64       `json:"keepMarginRate,string"`
	MarkPrice        float64       `json:"markPrice,string"`
	MarginRatio      float64       `json:"marginRatio,string"`
	CreationTime     UnixTimestamp `json:"cTime"`
}

// HistPositions is a sub-struct containing information on historical positions
type HistPositions struct {
	MarginCoin         string        `json:"marginCoin"`
	Symbol             string        `json:"symbol"`
	HoldSide           string        `json:"holdSide"`
	OpenAveragePrice   float64       `json:"openAvgPrice,string"`
	CloseAveragePrice  float64       `json:"closeAvgPrice,string"`
	MarginMode         string        `json:"marginMode"`
	OpenTotalPosition  float64       `json:"openTotalPos,string"`
	CloseTotalPosition float64       `json:"closeTotalPos,string"`
	PNL                float64       `json:"pnl,string"`
	NetProfit          float64       `json:"netProfit,string"`
	TotalFunding       float64       `json:"totalFunding,string"`
	OpenFee            float64       `json:"openFee,string"`
	CloseFee           float64       `json:"closeFee,string"`
	UpdateTime         UnixTimestamp `json:"uTime"`
	CreationTime       UnixTimestamp `json:"cTime"`
}

// HistPositionResp contains information on historical positions
type HistPositionResp struct {
	List  []HistPositions `json:"list"`
	EndID int64           `json:"endId,string"`
}

// PlaceFuturesOrderStruct contains information on an order to be placed
type PlaceFuturesOrderStruct struct {
	Size            float64   `json:"size,string"`
	Price           float64   `json:"price,string"`
	Side            string    `json:"side"`
	TradeSide       string    `json:"tradeSide"`
	OrderType       string    `json:"orderType"`
	Strategy        string    `json:"force"`
	ClientOID       string    `json:"clientOId"`
	ReduceOnly      YesNoBool `json:"reduceOnly"`
	TakeProfitValue float64   `json:"presetStopSurplusPrice,string,omitempty"`
	StopLossValue   float64   `json:"presetStopLossPrice,string,omitempty"`
}

// FuturesOrderDetailResp contains information on a futures order
type FuturesOrderDetailResp struct {
	Symbol                 string        `json:"symbol"`
	Size                   float64       `json:"size,string"`
	OrderID                EmptyInt      `json:"orderId"`
	ClientOrderID          string        `json:"clientOid"`
	BaseVolume             float64       `json:"baseVolume,string"`
	PriceAverage           float64       `json:"priceAvg,string"`
	Fee                    types.Number  `json:"fee"`
	Price                  float64       `json:"price,string"`
	State                  string        `json:"state"`
	Side                   string        `json:"side"`
	Force                  string        `json:"force"`
	TotalProfits           float64       `json:"totalProfits,string"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             string        `json:"marginCoin"`
	PresetStopSurplusPrice float64       `json:"presetStopSurplusPrice,string"`
	PresetStopLossPrice    float64       `json:"presetStopLossPrice,string"`
	QuoteVolume            float64       `json:"quoteVolume,string"`
	OrderType              string        `json:"orderType"`
	Leverage               float64       `json:"leverage,string"`
	MarginMode             string        `json:"marginMode"`
	ReduceOnly             YesNoBool     `json:"reduceOnly"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderSource            string        `json:"orderSource"`
	CreationTime           UnixTimestamp `json:"cTime"`
	UpdateTime             UnixTimestamp `json:"uTime"`
}

// FuturesFill is a sub-struct containing information on fulfilled futures orders
type FuturesFill struct {
	TradeID          int64               `json:"tradeId,string"`
	Symbol           string              `json:"symbol"`
	OrderID          int64               `json:"orderId,string"`
	Price            float64             `json:"price,string"`
	BaseVolume       float64             `json:"baseVolume,string"`
	FeeDetail        []AbridgedFeeDetail `json:"feeDetail"`
	Side             string              `json:"side"`
	QuoteVolume      float64             `json:"quoteVolume,string"`
	Profit           float64             `json:"profit,string"`
	EnterPointSource string              `json:"enterPointSource"`
	TradeSide        string              `json:"tradeSide"`
	PositionMode     string              `json:"posMode"`
	TradeScope       string              `json:"tradeScope"`
	CreationTime     UnixTimestamp       `json:"cTime"`
}

// FuturesFillsResp contains information on fulfilled futures orders
type FuturesFillsResp struct {
	FillList []FuturesFill `json:"fillList"`
	EndID    EmptyInt      `json:"endId"`
}

// FuturesOrder is a sub-struct containing information on futures orders
type FuturesOrder struct {
	Symbol                 string        `json:"symbol"`
	Size                   float64       `json:"size,string"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	BaseVolume             float64       `json:"baseVolume,string"`
	Fee                    types.Number  `json:"fee"`
	Price                  types.Number  `json:"price"`
	PriceAverage           types.Number  `json:"priceAvg"`
	Status                 string        `json:"status"`
	Side                   string        `json:"side"`
	Force                  string        `json:"force"`
	TotalProfits           float64       `json:"totalProfits,string"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             string        `json:"marginCoin"`
	QuoteVolume            float64       `json:"quoteVolume,string"`
	Leverage               float64       `json:"leverage,string"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	OrderSource            string        `json:"orderSource"`
	CreationTime           UnixTimestamp `json:"cTime"`
	UpdateTime             UnixTimestamp `json:"uTime"`
	PresetStopSurplusPrice types.Number  `json:"presetStopSurplusPrice"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice"`
}

// FuturesOrdResp contains information on futures orders
type FuturesOrdResp struct {
	EntrustedList []FuturesOrder `json:"entrustedList"`
	EndID         EmptyInt       `json:"endId"`
}

// PlanFuturesOrder is a sub-struct containing information on planned futures orders
type PlanFuturesOrder struct {
	PlanType               string        `json:"planType"`
	Symbol                 string        `json:"symbol"`
	Size                   float64       `json:"size,string"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	Price                  types.Number  `json:"price"`
	CallbackRatio          types.Number  `json:"callbackRatio"`
	TriggerPrice           float64       `json:"triggerPrice,string"`
	TriggerType            string        `json:"triggerType"`
	PlanStatus             string        `json:"planStatus"`
	Side                   string        `json:"side"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             string        `json:"marginCoin"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	OrderSource            string        `json:"orderSource"`
	CreationTime           UnixTimestamp `json:"cTime"`
	UpdateTime             UnixTimestamp `json:"uTime"`
	PresetTakeProfitPrice  types.Number  `json:"presetStopSurplusPrice"`
	TakeprofitTriggerPrice types.Number  `json:"stopSurplusTriggerPrice"`
	TakeProfitTriggerType  string        `json:"stopSurplusTriggerType"`
	PresetStopLossPrice    types.Number  `json:"presetStopLossPrice"`
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
	Size                   float64       `json:"size,string"`
	OrderID                int64         `json:"orderId,string"`
	ExecuteOrderID         int64         `json:"executeOrderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	PlanStatus             string        `json:"planStatus"`
	Price                  float64       `json:"price,string"`
	PriceAverage           float64       `json:"priceAvg,string"`
	BaseVolume             float64       `json:"baseVolume,string"`
	CallbackRatio          types.Number  `json:"callbackRatio"`
	TriggerPrice           float64       `json:"triggerPrice,string"`
	TriggerType            string        `json:"triggerType"`
	Side                   string        `json:"side"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             string        `json:"marginCoin"`
	MarginMode             string        `json:"marginMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	TradeSide              string        `json:"tradeSide"`
	PositionMode           string        `json:"posMode"`
	OrderType              string        `json:"orderType"`
	CreationTime           UnixTimestamp `json:"cTime"`
	UpdateTime             UnixTimestamp `json:"uTime"`
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
	Symbol                    string  `json:"symbol"`
	BaseCoin                  string  `json:"baseCoin"`
	QuoteCoin                 string  `json:"quoteCoin"`
	MaxCrossedLeverage        float64 `json:"maxCrossedLeverage,string"`
	MaxIsolatedLeverage       float64 `json:"maxIsolatedLeverage,string"`
	WarningRiskRatio          float64 `json:"warningRiskRatio,string"`
	LiquidationRiskRatio      float64 `json:"liquidationRiskRatio,string"`
	MinTradeAmount            float64 `json:"minTradeAmount,string"`
	MaxTradeAmount            float64 `json:"maxTradeAmount,string"`
	TakerFeeRate              float64 `json:"takerFeeRate,string"`
	MakerFeeRate              float64 `json:"makerFeeRate,string"`
	PricePrecision            uint8   `json:"pricePrecision,string"`
	QuantityPrecision         uint8   `json:"quantityPrecision,string"`
	MinTradeUSDT              float64 `json:"minTradeUSDT,string"`
	IsBorrowable              bool    `json:"isBorrowable"`
	UserMinBorrow             float64 `json:"userMinBorrow,string"`
	Status                    string  `json:"status"`
	IsIsolatedBaseBorrowable  bool    `json:"isIsolatedBaseBorrowable"`
	IsIsolatedQuoteBorrowable bool    `json:"isIsolatedQuoteBorrowable"`
	IsCrossBorrowable         bool    `json:"isCrossBorrowable"`
}

// CrossBorrow is a sub-struct containing information on borrowing for cross margin
type CrossBorrow struct {
	LoanID       int64         `json:"loanId,string"`
	Coin         string        `json:"coin"`
	BorrowAmount float64       `json:"borrowAmount,string"`
	BorrowType   string        `json:"borrowType"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
}

// BorrowHistCross contains information on borrowing history for cross margin
type BorrowHistCross struct {
	ResultList []CrossBorrow `json:"resultList"`
	MaxID      EmptyInt      `json:"maxId"`
	MinID      EmptyInt      `json:"minId"`
}

// Repayment is a sub-struct containing information on repayment
type Repayment struct {
	RepayID        int64         `json:"repayId,string"`
	Coin           string        `json:"coin"`
	RepayAmount    float64       `json:"repayAmount,string"`
	RepayType      string        `json:"repayType"`
	RepayInterest  float64       `json:"repayInterest,string"`
	RepayPrincipal float64       `json:"repayPrincipal,string"`
	CreationTime   UnixTimestamp `json:"cTime"`
	UpdateTime     UnixTimestamp `json:"uTime"`
}

// RepayHistResp contains information on repayment history
type RepayHistResp struct {
	ResultList []Repayment `json:"resultList"`
	MaxID      EmptyInt    `json:"maxId"`
	MinID      EmptyInt    `json:"minId"`
}

// CrossInterest is a sub-struct containing information on interest for cross margin
type CrossInterest struct {
	InterestID        int64         `json:"interestId,string"`
	LoanCoin          string        `json:"loanCoin"`
	InterestCoin      string        `json:"interestCoin"`
	DailyInterestRate float64       `json:"dailyInterestRate,string"`
	InterestAmount    float64       `json:"interestAmount,string"`
	InterestType      string        `json:"interstType"` // sic
	CreationTime      UnixTimestamp `json:"cTime"`
	UpdateTime        UnixTimestamp `json:"uTime"`
}

// InterHistCross contains information on interest history for cross margin
type InterHistCross struct {
	MinID      EmptyInt        `json:"minId"`
	MaxID      EmptyInt        `json:"maxId"`
	ResultList []CrossInterest `json:"resultList"`
}

// CrossLiquidation is a sub-struct containing information on liquidation for cross margin
type CrossLiquidation struct {
	LiquidationID        int64         `json:"liqId,string"`
	LiquidationStartTime UnixTimestamp `json:"liqStartTime"`
	LiquidationEndTime   UnixTimestamp `json:"liqEndTime"`
	LiquidationRiskRatio float64       `json:"liqRiskRatio,string"`
	TotalAssets          float64       `json:"totalAssets,string"`
	TotalDebt            float64       `json:"totalDebt,string"`
	LiquidationFee       float64       `json:"liqFee,string"`
	UpdateTime           UnixTimestamp `json:"uTime"`
	CreationTime         UnixTimestamp `json:"cTime"`
}

// LiquidHistCross contains information on liquidation history for cross margin
type LiquidHistCross struct {
	MinID      EmptyInt           `json:"minId"`
	MaxID      EmptyInt           `json:"maxId"`
	ResultList []CrossLiquidation `json:"resultList"`
}

// CrossFinHist is a sub-struct containing information on financial history for cross margin
type CrossFinHist struct {
	MarginID     int64         `json:"marginId,string"`
	Amount       float64       `json:"amount,string"`
	Coin         string        `json:"coin"`
	Balance      float64       `json:"balance,string"`
	Fee          float64       `json:"fee,string"`
	MarginType   string        `json:"marginType"`
	UpdateTime   UnixTimestamp `json:"uTime"`
	CreationTime UnixTimestamp `json:"cTime"`
}

// FinHistCrossResp contains information on financial history for cross margin
type FinHistCrossResp struct {
	MinID      EmptyInt       `json:"minId"`
	MaxID      EmptyInt       `json:"maxId"`
	ResultList []CrossFinHist `json:"resultList"`
}

// CrossAssetResp contains information on assets being utilised in cross margin
type CrossAssetResp struct {
	Coin         string        `json:"coin"`
	TotalAmount  float64       `json:"totalAmount,string"`
	Available    float64       `json:"available,string"`
	Frozen       float64       `json:"frozen,string"`
	Borrow       float64       `json:"borrow,string"`
	Interest     float64       `json:"interest,string"`
	Net          float64       `json:"net,string"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
	Coupon       float64       `json:"coupon,string"`
}

// BorrowCross contains information on borrowing for cross margin
type BorrowCross struct {
	LoanID       int64   `json:"loanId,string"`
	Coin         string  `json:"coin"`
	BorrowAmount float64 `json:"borrowAmount,string"`
}

// RepayCross contains information on repayment for cross margin
type RepayCross struct {
	Coin                string  `json:"coin"`
	RepayID             int64   `json:"repayId,string"`
	RemainingDebtAmount float64 `json:"remainDebtAmount,string"`
	RepayAmount         float64 `json:"repayAmount,string"`
}

// MaxBorrowCross contains information on the maximum amount that can be borrowed for cross margin
type MaxBorrowCross struct {
	Coin                string  `json:"coin"`
	MaxBorrowableAmount float64 `json:"maxBorrowableAmount,string"`
}

// MaxTransferCross contains information on the maximum amount that can be transferred out of cross margin
type MaxTransferCross struct {
	Coin                 string  `json:"coin"`
	MaxTransferOutAmount float64 `json:"maxTransferOutAmount,string"`
}

// VIPInfo is a sub-struct containing information on VIP levels
type VIPInfo struct {
	Level              int64   `json:"level,string"`
	Limit              float64 `json:"limit,string"`
	DailyInterestRate  float64 `json:"dailyInterestRate,string"`
	AnnualInterestRate float64 `json:"annualInterestRate,string"`
	DiscountRate       float64 `json:"discountRate,string"`
}

// IntRateMaxBorrowCross contains information on the interest rate and the maximum amount that can be borrowed for
// cross margin
type IntRateMaxBorrowCross struct {
	Transferable        bool      `json:"transferable"`
	Leverage            float64   `json:"leverage,string"`
	Coin                string    `json:"coin"`
	Borrowable          bool      `json:"borrowable"`
	DailyInterestRate   float64   `json:"dailyInterestRate,string"`
	AnnualInterestRate  float64   `json:"annualInterestRate,string"`
	MaxBorrowableAmount float64   `json:"maxBorrowableAmount,string"`
	VIPList             []VIPInfo `json:"vipList"`
}

// TierConfigCross contains information on tier configurations for cross margin
type TierConfigCross struct {
	Tier                int64   `json:"tier,string"`
	Leverage            float64 `json:"leverage,string"`
	Coin                string  `json:"coin"`
	MaxBorrowableAmount float64 `json:"maxBorrowableAmount,string"`
	MaintainMarginRate  float64 `json:"maintainMarginRate,string"`
}

// FlashRepayCross contains information on a flash repayment for cross margin
type FlashRepayCross struct {
	RepayID int64  `json:"repayId,string"`
	Coin    string `json:"coin"`
}

// FlashRepayResult contains information on the result of a flash repayment
type FlashRepayResult struct {
	RepayID int64  `json:"repayId,string"`
	Status  string `json:"status"`
}

// MarginOrderData contains information on a margin order
type MarginOrderData struct {
	Side          string  `json:"side"`
	OrderType     string  `json:"orderType"`
	Price         float64 `json:"price,string"`
	Strategy      string  `json:"force"`
	BaseAmount    float64 `json:"baseSize,string"`
	QuoteAmount   float64 `json:"quoteSize,string"`
	LoanType      string  `json:"loanType"`
	ClientOrderID string  `json:"clientOid"`
}

// MarginOrder is a sub-struct containing information on a margin order
type MarginOrder struct {
	OrderID          int64         `json:"orderId,string"`
	Symbol           string        `json:"symbol"`
	OrderType        string        `json:"orderType"`
	EnterPointSource string        `json:"enterPointSource"`
	ClientOrderID    string        `json:"clientOid"`
	LoanType         string        `json:"loanType"`
	Price            float64       `json:"price,string"`
	Side             string        `json:"side"`
	Status           string        `json:"status"`
	BaseSize         float64       `json:"baseSize,string"`
	QuoteSize        float64       `json:"quoteSize,string"`
	Size             float64       `json:"size,string"`
	Amount           float64       `json:"amount,string"`
	Force            string        `json:"force"`
	CreationTime     UnixTimestamp `json:"cTime"`
	UpdateTime       UnixTimestamp `json:"uTime"`
}

// MarginOpenOrds contains information on open margin orders
type MarginOpenOrds struct {
	OrderList []MarginOrder `json:"orderList"`
	MaxID     EmptyInt      `json:"maxId"`
	MinID     EmptyInt      `json:"minId"`
}

// MarginOrdWithAveragePrice is a sub-struct containing information on a margin order with an average price
type MarginOrdWithAveragePrice struct {
	OrderID          int64         `json:"orderId,string"`
	Symbol           string        `json:"symbol"`
	OrderType        string        `json:"orderType"`
	EnterPointSource string        `json:"enterPointSource"`
	ClientOrderID    string        `json:"clientOid"`
	LoanType         string        `json:"loanType"`
	Price            float64       `json:"price,string"`
	Side             string        `json:"side"`
	Status           string        `json:"status"`
	BaseSize         float64       `json:"baseSize,string"`
	QuoteSize        float64       `json:"quoteSize,string"`
	PriceAverage     float64       `json:"priceAvg,string"`
	Size             float64       `json:"size,string"`
	Amount           float64       `json:"amount,string"`
	Force            string        `json:"force"`
	CreationTime     UnixTimestamp `json:"cTime"`
	UpdateTime       UnixTimestamp `json:"uTime"`
}

// MarginHistOrds contains information on historical margin orders
type MarginHistOrds struct {
	OrderList []MarginOrdWithAveragePrice `json:"orderList"`
	MaxID     EmptyInt                    `json:"maxId"`
	MinID     EmptyInt                    `json:"minId"`
}

// MarginFill is a sub-struct containing information on fulfilled margin orders
type MarginFill struct {
	OrderID      int64             `json:"orderId,string"`
	TradeID      int64             `json:"tradeId,string"`
	OrderType    string            `json:"orderType"`
	Side         string            `json:"side"`
	PriceAverage float64           `json:"priceAvg,string"`
	Size         float64           `json:"size,string"`
	Amount       float64           `json:"amount,string"`
	TradeScope   string            `json:"tradeScope"`
	CreationTime UnixTimestamp     `json:"cTime"`
	UpdateTime   UnixTimestamp     `json:"uTime"`
	FeeDetail    AbridgedFeeDetail `json:"feeDetail"`
}

// MarginOrderFills contains information on fulfilled margin orders
type MarginOrderFills struct {
	Fills []MarginFill `json:"fills"`
	MaxID EmptyInt     `json:"maxId"`
	MinID EmptyInt     `json:"minId"`
}

// LiquidationOrder is a sub-struct containing information on liquidation orders
type LiquidationOrder struct {
	Symbol       string        `json:"symbol"`
	OrderType    string        `json:"orderType"`
	Side         string        `json:"side"`
	PriceAverage float64       `json:"priceAvg,string"`
	Price        float64       `json:"price,string"`
	FillSize     float64       `json:"fillSize,string"`
	Size         float64       `json:"size,string"`
	Amount       float64       `json:"amount,string"`
	OrderID      int64         `json:"orderId,string"`
	FromCoin     string        `json:"fromCoin"`
	ToCoin       string        `json:"toCoin"`
	FromSize     types.Number  `json:"fromSize"`
	ToSize       types.Number  `json:"toSize"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
}

// LiquidationResp contains information on liquidation orders
type LiquidationResp struct {
	ResultList []LiquidationOrder `json:"resultList"`
	IDLessThan EmptyInt           `json:"idLessThan"`
}

// IsoBorrow is a sub-struct containing information on borrowing for isolated margin
type IsoBorrow struct {
	LoanID       int64         `json:"loanId,string"`
	Coin         string        `json:"coin"`
	BorrowAmount float64       `json:"borrowAmount,string"`
	BorrowType   string        `json:"borrowType"`
	Symbol       string        `json:"symbol"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
}

// BorrowHistIso contains information on borrowing history for isolated margin
type BorrowHistIso struct {
	ResultList []IsoBorrow `json:"resultList"`
	MaxID      EmptyInt    `json:"maxId"`
	MinID      EmptyInt    `json:"minId"`
}

// IsoInterest is a sub-struct containing information on interest for isolated margin
type IsoInterest struct {
	InterestID        int64         `json:"interestId,string"`
	LoanCoin          string        `json:"loanCoin"`
	InterestCoin      string        `json:"interestCoin"`
	DailyInterestRate float64       `json:"dailyInterestRate,string"`
	InterestAmount    float64       `json:"interestAmount,string"`
	InterestType      string        `json:"interstType"` // sic
	Symbol            string        `json:"symbol"`
	CreationTime      UnixTimestamp `json:"cTime"`
	UpdateTime        UnixTimestamp `json:"uTime"`
}

// InterHistIso contains information on interest history for isolated margin
type InterHistIso struct {
	MinID      EmptyInt      `json:"minId"`
	MaxID      EmptyInt      `json:"maxId"`
	ResultList []IsoInterest `json:"resultList"`
}

// IsoLiquidation is a sub-struct containing information on liquidation for isolated margin
type IsoLiquidation struct {
	LiquidationID        int64         `json:"liqId,string"`
	Symbol               string        `json:"symbol"`
	LiquidationStartTime UnixTimestamp `json:"liqStartTime"`
	LiquidationEndTime   UnixTimestamp `json:"liqEndTime"`
	LiquidationRiskRatio float64       `json:"liqRiskRatio,string"`
	TotalAssets          float64       `json:"totalAssets,string"`
	TotalDebt            float64       `json:"totalDebt,string"`
	LiquidationFee       float64       `json:"liqFee,string"`
	UpdateTime           UnixTimestamp `json:"uTime"`
	CreationTime         UnixTimestamp `json:"cTime"`
}

// LiquidHistIso contains information on liquidation history for isolated margin
type LiquidHistIso struct {
	MinID      EmptyInt         `json:"minId"`
	MaxID      EmptyInt         `json:"maxId"`
	ResultList []IsoLiquidation `json:"resultList"`
}

// IsoFinHist is a sub-struct containing information on financial history for isolated margin
type IsoFinHist struct {
	MarginID     int64         `json:"marginId,string"`
	Amount       float64       `json:"amount,string"`
	Coin         string        `json:"coin"`
	Symbol       string        `json:"symbol"`
	Balance      float64       `json:"balance,string"`
	Fee          float64       `json:"fee,string"`
	MarginType   string        `json:"marginType"`
	UpdateTime   UnixTimestamp `json:"uTime"`
	CreationTime UnixTimestamp `json:"cTime"`
}

// FinHistIsoResp contains information on financial history for isolated margin
type FinHistIsoResp struct {
	MinID      EmptyInt     `json:"minId"`
	MaxID      EmptyInt     `json:"maxId"`
	ResultList []IsoFinHist `json:"resultList"`
}

// IsoAssetResp contains information on assets being utilised in isolated margin
type IsoAssetResp struct {
	Symbol       string        `json:"symbol"`
	Coin         string        `json:"coin"`
	TotalAmount  float64       `json:"totalAmount,string"`
	Available    float64       `json:"available,string"`
	Frozen       float64       `json:"frozen,string"`
	Borrow       float64       `json:"borrow,string"`
	Interest     float64       `json:"interest,string"`
	Net          float64       `json:"net,string"`
	CreationTime UnixTimestamp `json:"cTime"`
	UpdateTime   UnixTimestamp `json:"uTime"`
	Coupon       float64       `json:"coupon,string"`
}

// BorrowIso contains information on borrowing for isolated margin
type BorrowIso struct {
	LoanID       int64   `json:"loanId,string"`
	Symbol       string  `json:"symbol"`
	Coin         string  `json:"coin"`
	BorrowAmount float64 `json:"borrowAmount,string"`
}

// RepayIso contains information on repayment for isolated margin
type RepayIso struct {
	Coin                string  `json:"coin"`
	Symbol              string  `json:"symbol"`
	RepayID             int64   `json:"repayId,string"`
	RemainingDebtAmount float64 `json:"remainDebtAmount,string"`
	RepayAmount         float64 `json:"repayAmount,string"`
}

// RiskRateIso contains information on the risk rate for isolated margin
type RiskRateIso struct {
	Symbol        string  `json:"symbol"`
	RiskRateRatio float64 `json:"riskRateRatio,string"`
}

// IsoVIPList contains information on VIP lists for isolated margin
type IsoVIPList struct {
	Level              int64   `json:"level,string"`
	Limit              float64 `json:"limit,string"`
	DailyInterestRate  float64 `json:"dailyInterestRate,string"`
	AnnualInterestRate float64 `json:"annuallyInterestRate,string"`
	DiscountRate       float64 `json:"discountRate,string"`
}

// IntRateMaxBorrowIso contains information on the interest rate and the maximum amount that can be borrowed for
// isolated margin
type IntRateMaxBorrowIso struct {
	Symbol                   string       `json:"symbol"`
	Leverage                 float64      `json:"leverage,string"`
	BaseCoin                 string       `json:"baseCoin"`
	BaseTransferable         bool         `json:"baseTransferable"`
	BaseBorrowable           bool         `json:"baseBorrowable"`
	BaseDailyInterestRate    float64      `json:"baseDailyInterestRate,string"`
	BaseAnnualInterestRate   float64      `json:"baseAnnuallyInterestRate,string"` // sic
	BaseMaxBorrowableAmount  float64      `json:"baseMaxBorrowableAmount,string"`
	BaseVIPList              []IsoVIPList `json:"baseVipList"`
	QuoteCoin                string       `json:"quoteCoin"`
	QuoteTransferable        bool         `json:"quoteTransferable"`
	QuoteBorrowable          bool         `json:"quoteBorrowable"`
	QuoteDailyInterestRate   float64      `json:"quoteDailyInterestRate,string"`
	QuoteAnnualInterestRate  float64      `json:"quoteAnnuallyInterestRate,string"` // sic
	QuoteMaxBorrowableAmount float64      `json:"quoteMaxBorrowableAmount,string"`
	QuoteVIPList             []IsoVIPList `json:"quoteList"`
}

// TierConfigIso contains information on tier configurations for isolated margin
type TierConfigIso struct {
	Tier                     int64   `json:"tier,string"`
	Symbol                   string  `json:"symbol"`
	Leverage                 float64 `json:"leverage,string"`
	BaseCoin                 string  `json:"baseCoin"`
	QuoteCoin                string  `json:"quoteCoin"`
	BaseMaxBorrowableAmount  float64 `json:"baseMaxBorrowableAmount,string"`
	QuoteMaxBorrowableAmount float64 `json:"quoteMaxBorrowableAmount,string"`
	MaintainMarginRate       float64 `json:"maintainMarginRate,string"`
	InitRate                 float64 `json:"initRate,string"`
}

// MaxBorrowIso contains information on the maximum amount that can be borrowed for isolated margin
type MaxBorrowIso struct {
	Symbol                       string  `json:"symbol"`
	BaseCoin                     string  `json:"baseCoin"`
	BaseCoinMaxBorrowableAmount  float64 `json:"baseCoinmaxBorrowAmount,string"`
	QuoteCoin                    string  `json:"quoteCoin"`
	QuoteCoinMaxBorrowableAmount float64 `json:"quoteCoinmaxBorrowAmount,string"`
}

// MaxTransferIso contains information on the maximum amount that can be transferred out of isolated margin
type MaxTransferIso struct {
	BaseCoin                      string       `json:"baseCoin"`
	Symbol                        string       `json:"symbol"`
	BaseCoinMaxTransferOutAmount  types.Number `json:"baseCoinMaxTransferOutAmount"`
	QuoteCoin                     string       `json:"quoteCoin"`
	QuoteCoinMaxTransferOutAmount types.Number `json:"quoteCoinMaxTransferOutAmount"`
}

// FlashRepayIso contains information on a flash repayment for isolated margin
type FlashRepayIso struct {
	RepayID int64       `json:"repayId,string"`
	Symbol  string      `json:"symbol"`
	Result  SuccessBool `json:"result"`
}

type APY struct {
	RateLevel    int64   `json:"rateLevel,string"`
	MinStepValue float64 `json:"minStepVal,string"`
	MaxStepValue float64 `json:"maxStepVal,string"`
	CurrentAPY   float64 `json:"currentAPY,string"`
}

// SavingsProductList contains information on savings products
type SavingsProductList struct {
	ProductID     int64     `json:"productId,string"`
	Coin          string    `json:"coin"`
	PeriodType    string    `json:"periodType"`
	Period        EmptyInt  `json:"period"`
	APYType       string    `json:"apyType"`
	AdvanceRedeem YesNoBool `json:"advanceRedeem"`
	SettleMethod  string    `json:"settleMethod"`
	APYList       []APY     `json:"apyList"`
	Status        string    `json:"status"`
	ProductLevel  string    `json:"productLevel"`
}

// SavingsBalance contains information on savings balances
type SavingsBalance struct {
	BTCAmount          float64 `json:"btcAmount,string"`
	USDTAmount         float64 `json:"usdtAmount,string"`
	BTC24HourEarnings  float64 `json:"btc24HourEarning,string"`
	USDT24HourEarnings float64 `json:"usdt24HourEarning,string"`
	BTCTotalEarnings   float64 `json:"btcTotalEarning,string"`
	USDTTotalEarnings  float64 `json:"usdtTotalEarning,string"`
}

// SavingsAsset is a sub-struct containing information on savings assets
type SavingsAsset struct {
	ProductID       int64     `json:"productId,string"`
	OrderID         int64     `json:"orderId,string"` // Docs are inconsistent, check whether this exists
	ProductCoin     string    `json:"productCoin"`
	InterestCoin    string    `json:"interestCoin"`
	PeriodType      string    `json:"periodType"`
	Period          EmptyInt  `json:"period"`
	HoldAmount      float64   `json:"holdAmount,string"`
	LastProfit      float64   `json:"lastProfit,string"`
	TotalProfit     float64   `json:"totalProfit,string"`
	HoldDays        EmptyInt  `json:"holdDays"`
	Status          string    `json:"status"`
	AllowRedemption YesNoBool `json:"allowRedemption"` // Docs are inconsistent, check whether this exists
	ProductLevel    string    `json:"productLevel"`
	APY             []APY     `json:"apy"`
}

// SavingsAssetsResp contains information on savings assets
type SavingsAssetsResp struct {
	ResultList []SavingsAsset `json:"resultList"`
	EndID      EmptyInt       `json:"endId"`
}

// SavingsTransaction is a sub-struct containing information on a savings transaction
type SavingsTransaction struct {
	OrderID        int64         `json:"orderId,string"`
	CoinName       string        `json:"coinName"`
	SettleCoinName string        `json:"settleCoinName"`
	ProductType    string        `json:"productType"`
	Period         EmptyInt      `json:"period"`
	ProductLevel   string        `json:"productLevel"`
	Amount         float64       `json:"amount,string"`
	Timestamp      UnixTimestamp `json:"ts"`
	OrderType      string        `json:"orderType"`
}

// SavingsRecords contains information on previous transactions
type SavingsRecords struct {
	ResultList []SavingsTransaction `json:"resultList"`
	EndID      EmptyInt             `json:"endId"`
}

// SavingsSubDetail contains information about a potential subscription
type SavingsSubDetail struct {
	SingleMinAmount    float64       `json:"singleMinAmount,string"`
	SingleMaxAmount    float64       `json:"singleMaxAmount,string"`
	RemainingAmount    float64       `json:"remainingAmount,string"`
	SubscribePrecision uint8         `json:"subscribePrecision,string"`
	ProfitPrecision    uint8         `json:"profitPrecision,string"`
	SubscribeTime      UnixTimestamp `json:"subscribeTime"`
	InterestTime       UnixTimestamp `json:"interestTime"`
	SettleTime         UnixTimestamp `json:"settleTime"`
	ExpireTime         UnixTimestamp `json:"expireTime"`
	RedeemTime         UnixTimestamp `json:"redeemTime"`
	SettleMethod       string        `json:"settleMethod"`
	APYList            []APY         `json:"apyList"`
	RedeemDelay        string        `json:"redeemDelay"`
}

// SubResp contains information on a transaction involving a savings product
type SaveResp struct {
	OrderID int64  `json:"orderId,string"`
	Status  string `json:"status"` // Double-check, might be a float64
}

// SubResult contains information on the result of a transaction involving a savings product
type SaveResult struct {
	Result  SuccessBool `json:"result"`
	Message string      `json:"msg"`
}

// EarnAssets contains information on assets in the earn account
type EarnAssets struct {
	Coin   string  `json:"coin"`
	Amount float64 `json:"amount,string"`
}

// SharkFinProduct is a sub-struct containing information on a shark fin product
type SharKFinProduct struct {
	ProductID         int64         `json:"productId,string"`
	ProductName       string        `json:"productName"`
	ProductCoin       string        `json:"productCoin"`
	SubscribeCoin     string        `json:"subscribeCoin"`
	FarmingStartTime  UnixTimestamp `json:"farmingStartTime"`
	FarmingEndTime    UnixTimestamp `json:"farmingEndTime"`
	LowerRate         float64       `json:"lowerRate,string"`
	DefaultRate       float64       `json:"defaultRate,string"`
	UpperRate         float64       `json:"upperRate,string"`
	Period            EmptyInt      `json:"period"`
	InterestStartTime UnixTimestamp `json:"interestStartTime"`
	Status            string        `json:"status"`
	MinAmount         float64       `json:"minAmount,string"`
	LimitAmount       float64       `json:"limitAmount,string"`
	SoldAmount        float64       `json:"soldAmount,string"`
	EndTime           UnixTimestamp `json:"endTime"`
	StartTime         UnixTimestamp `json:"startTime"`
}

// SharkFinProductResp contains information on shark fin products
type SharkFinProductResp struct {
	ResultList []SharKFinProduct `json:"resultList"`
	EndID      EmptyInt          `json:"endId"`
}

// SharkFinBalance contains information on one's shark fin balance and amount earned
type SharkFinBalance struct {
	BTCSubscribeAmount   float64 `json:"btcSubscribeAmount,string"`
	USDTSubscribeAmount  float64 `json:"usdtSubscribeAmount,string"`
	BTCHistoricalAmount  float64 `json:"btcHistoricalAmount,string"`
	USDTHistoricalAmount float64 `json:"usdtHistoricalAmount,string"`
	BTCTotalEarning      float64 `json:"btcTotalEarning,string"`
	USDTTotalEarning     float64 `json:"usdtTotalEarning,string"`
}

// SharkFinAsset is a sub-struct containing information on a shark fin asset
type SharkFinAsset struct {
	ProductID         int64         `json:"productId,string"`
	InterestStartTime UnixTimestamp `json:"interestStartTime"`
	InterestEndTime   UnixTimestamp `json:"interestEndTime"`
	ProductCoin       string        `json:"productCoin"`
	SubscribeCoin     string        `json:"subscribeCoin"`
	Trend             string        `json:"trend"`
	SettleTime        UnixTimestamp `json:"settleTime"`
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
	OrderID   int64         `json:"orderId,string"`
	Product   string        `json:"product"`
	Period    EmptyInt      `json:"period"`
	Amount    float64       `json:"amount,string"`
	Timestamp UnixTimestamp `json:"ts"`
	Type      string        `json:"type"`
}

// SharkFinSubDetail contains information useful when subscribing to a shark fin product
type SharkFinSubDetail struct {
	ProductCoin        string        `json:"productCoin"`
	SubscribeCoin      string        `json:"subscribeCoin"`
	InterestTime       UnixTimestamp `json:"interestTime"`
	ExpirationTime     UnixTimestamp `json:"expirationTime"`
	MinPrice           float64       `json:"minPrice,string"`
	CurrentPrice       float64       `json:"currentPrice,string"`
	MaxPrice           float64       `json:"maxPrice,string"`
	MinRate            float64       `json:"minRate,string"`
	DefaultRate        float64       `json:"defaultRate,string"`
	MaxRate            float64       `json:"maxRate,string"`
	Period             EmptyInt      `json:"period"`
	ProductMinAmount   float64       `json:"productMinAmount,string"`
	AvailableBalance   float64       `json:"availableBalance,string"`
	UserAmount         float64       `json:"userAmount,string"`
	RemainingAmount    float64       `json:"remainingAmount,string"`
	ProfitPrecision    uint8         `json:"profitPrecision,string"`
	SubscribePrecision uint8         `json:"subscribePrecision,string"`
}

// LoanInfos is a sub-struct containing information on loans
type LoanInfos struct {
	Coin            string  `json:"coin"`
	HourlyRate7Day  float64 `json:"hourRate7D,string"`
	Rate7Day        float64 `json:"rate7D,string"`
	HourlyRate30Day float64 `json:"hourRate30D,string"`
	Rate30Day       float64 `json:"rate30D,string"`
	MinUSDT         float64 `json:"minUsdt,string"`
	MaxUSDT         float64 `json:"maxUsdt,string"`
	Min             float64 `json:"min,string"`
	Max             float64 `json:"max,string"`
}

// PledgeInfos is a sub-struct containing information on pledges
type PledgeInfos struct {
	Coin              string  `json:"coin"`
	InitialRate       float64 `json:"initRate,string"`
	SupplementaryRate float64 `json:"supRate,string"`
	ForceRate         float64 `json:"forceRate,string"`
	MinUSDT           float64 `json:"minUsdt,string"`
	MaxUSDT           float64 `json:"maxUsdt,string"`
}

// LoanCurList contains information on currencies which can be loaned
type LoanCurList struct {
	LoanInfos   []LoanInfos   `json:"loanInfos"`
	PledgeInfos []PledgeInfos `json:"pledgeInfos"`
}

// EstimateInterest contains information on estimated interest payments and borrowable amounts
type EstimateInterest struct {
	HourInterest float64 `json:"hourInterest,string"`
	LoanAmount   float64 `json:"loanAmount,string"`
}

// BorrowResp contains information on a loan
type BorrowResp struct {
	OrderID int64 `json:"orderId,string"`
}

// OngoingLoans contains information on ongoing loans
type OngoingLoans struct {
	OrderID           int64         `json:"orderId,string"`
	LoanCoin          string        `json:"loanCoin"`
	LoanAmount        float64       `json:"loanAmount,string"`
	InterestAmount    float64       `json:"interestAmount,string"`
	HourInterestRate  float64       `json:"hourInterestRate,string"`
	PledgeCoin        string        `json:"pledgeCoin"`
	PledgeAmount      float64       `json:"pledgeAmount,string"`
	SupplementaryRate float64       `json:"supRate,string"`
	ForceRate         float64       `json:"forceRate,string"`
	BorrowTime        UnixTimestamp `json:"borrowTime"`
	ExpireTime        UnixTimestamp `json:"expireTime"`
}

// RepayResp contains information on a repayment
type RepayResp struct {
	LoanCoin          string  `json:"loanCoin"`
	PledgeCoin        string  `json:"pledgeCoin"`
	RepayAmount       float64 `json:"repayAmount,string"`
	PayInterest       float64 `json:"payInterest,string"`
	RepayLoanAmount   float64 `json:"repayLoanAmount,string"`
	RepayUnlockAmount float64 `json:"repayUnlockAmount,string"`
}

// RepayRecords contains information on repayment records
type RepayRecords struct {
	OrderID           int64         `json:"orderId,string"`
	LoanCoin          string        `json:"loanCoin"`
	PledgeCoin        string        `json:"pledgeCoin"`
	RepayAmount       float64       `json:"repayAmount,string"`
	PayInterest       float64       `json:"payInterest,string"`
	RepayLoanAmount   float64       `json:"repayLoanAmount,string"`
	RepayUnlockAmount float64       `json:"repayUnlockAmount,string"`
	RepayTime         UnixTimestamp `json:"repayTime"`
}

// ModPledgeResp contains information on a pledge modification
type ModPledgeResp struct {
	LoanCoin        string  `json:"loanCoin"`
	PledgeCoin      string  `json:"pledgeCoin"`
	AfterPledgeRate float64 `json:"afterPledgeRate,string"`
}

// PledgeRateHist contains information on historical pledge rates
type PledgeRateHist struct {
	LoanCoin         string        `json:"loanCoin"`
	PledgeCoin       string        `json:"pledgeCoin"`
	OrderID          int64         `json:"orderId,string"`
	ReviseTime       UnixTimestamp `json:"reviseTime"`
	ReviseSide       string        `json:"reviseSide"`
	ReviseAmount     float64       `json:"reviseAmount,string"`
	AfterPledgeRate  float64       `json:"afterPledgeRate,string"`
	BeforePledgeRate float64       `json:"beforePledgeRate,string"`
}

// LoanHistory contains information on loans
type LoanHistory struct {
	OrderID             int64         `json:"orderId,string"`
	LoanCoin            string        `json:"loanCoin"`
	PledgeCoin          string        `json:"pledgeCoin"`
	InitialPledgeAmount float64       `json:"initPledgeAmount,string"`
	InitialLoanAmount   float64       `json:"initLoanAmount,string"`
	HourlyRate          float64       `json:"hourRate,string"`
	Daily               float64       `json:"daily,string"`
	BorrowTime          UnixTimestamp `json:"borrowTime"`
	Status              string        `json:"status"`
}

// CoinAm includes fields for coins, amounts, and amount-equivalents in USDT
type CoinAm struct {
	Coin       string  `json:"coin"`
	Amount     float64 `json:"amount,string"`
	AmountUSDT float64 `json:"amountUsdt,string"`
}

// DebtsResp contains information on debts
type DebtsResp struct {
	PledgeInfos []CoinAm `json:"pledgeInfos"`
	LoanInfos   []CoinAm `json:"loanInfos"`
}

// LiquidRecs contains information on liquidation records
type LiquidRecs struct {
	OrderID         int64         `json:"orderId,string"`
	LoanCoin        string        `json:"loanCoin"`
	PledgeCoin      string        `json:"pledgeCoin"`
	ReduceTime      UnixTimestamp `json:"reduceTime"`
	PledgeRate      float64       `json:"pledgeRate,string"`
	PledgePrice     float64       `json:"pledgePrice,string"`
	Status          string        `json:"status"`
	PledgeAmount    float64       `json:"pledgeAmount,string"`
	ReduceFee       string        `json:"reduceFee"`
	ResidueAmount   float64       `json:"residueAmount,string"`
	RunlockAmount   float64       `json:"runlockAmount,string"`
	RepayLoanAmount float64       `json:"repayLoanAmount,string"`
}

// WsResponse contains information on a websocket response
type WsResponse struct {
	Event     string              `json:"event"`
	Code      int                 `json:"code"`
	Message   string              `json:"msg"`
	Arg       WsArgument          `json:"arg"`
	Action    string              `json:"action"`
	Data      json.RawMessage     `json:"data"`
	Timestamp UnixTimestampNumber `json:"ts"`
}

// WsArgument contains information used in a websocket request and response
type WsArgument struct {
	InstrumentType string `json:"instType"`
	Channel        string `json:"channel"`
	InstrumentID   string `json:"instId,omitempty"`
	Coin           string `json:"coin,omitempty"`
}

// WsRequest contains information on a websocket request
type WsRequest struct {
	Operation string       `json:"op"`
	Arguments []WsArgument `json:"args"`
}

// WsLoginArgument contains information usied in a websocket login request
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

// WsTickerSnapshot contains information on a ticker snapshot
type WsTickerSnapshot struct {
	InstrumentID string        `json:"instId"`
	LastPrice    float64       `json:"lastPr,string"`
	Open24H      float64       `json:"open24h,string"`
	High24H      float64       `json:"high24h,string"`
	Low24H       float64       `json:"low24h,string"`
	Change24H    float64       `json:"change24h,string"`
	BidPrice     float64       `json:"bidPr,string"`
	AskPrice     float64       `json:"askPr,string"`
	BidSize      float64       `json:"bidSz,string"`
	AskSize      float64       `json:"askSz,string"`
	BaseVolume   float64       `json:"baseVolume,string"`
	QuoteVolume  float64       `json:"quoteVolume,string"`
	OpenUTC      float64       `json:"openUtc,string"`
	ChangeUTC24H float64       `json:"changeUtc24h,string"`
	Timestamp    UnixTimestamp `json:"ts"`
}

// WsAccountSpotResponse contains information on an account response for spot trading
type WsAccountSpotResponse struct {
	Coin           string        `json:"coin"`
	Available      float64       `json:"available,string"`
	Frozen         float64       `json:"frozen,string"`
	Locked         float64       `json:"locked,string"`
	LimitAvailable float64       `json:"limitAvailable,string"`
	UpdateTime     UnixTimestamp `json:"uTime"`
}

// WsTradeResponse contains information on a trade response
type WsTradeResponse struct {
	Timestamp UnixTimestamp `json:"ts"`
	Price     float64       `json:"price,string"`
	Size      float64       `json:"size,string"`
	Side      string        `json:"side"`
	TradeID   int64         `json:"tradeId,string"`
}

// WsOrderBookResponse contains information on an order book response
type WsOrderBookResponse struct {
	Asks      [][2]string   `json:"asks"`
	Bids      [][2]string   `json:"bids"`
	Timestamp UnixTimestamp `json:"ts"`
	Checksum  int32         `json:"checksum"`
}

// WsFillSpotResponse contains information on a fill response for spot trading
type WsFillSpotResponse struct {
	OrderID      int64               `json:"orderId,string"`
	TradeID      int64               `json:"tradeId,string"`
	Symbol       string              `json:"symbol"`
	OrderType    string              `json:"orderType"`
	Side         string              `json:"side"`
	PriceAverage float64             `json:"priceAvg,string"`
	Size         float64             `json:"size,string"`
	Amount       float64             `json:"amount,string"`
	TradeScope   string              `json:"tradeScope"`
	FeeDetail    []AbridgedFeeDetail `json:"feeDetail"`
	CreationTime UnixTimestamp       `json:"cTime"`
	UpdateTime   UnixTimestamp       `json:"uTime"`
}

// WsOrderSpotResponse contains information on an order response for spot trading
type WsOrderSpotResponse struct {
	InstrumentID      string              `json:"instId"`
	OrderID           int64               `json:"orderId,string"`
	ClientOrderID     string              `json:"clientOid"`
	Price             float64             `json:"price,string"`
	Size              float64             `json:"size,string"`
	NewSize           float64             `json:"newSize,string"`
	Notional          float64             `json:"notional,string"`
	OrderType         string              `json:"orderType"`
	Force             string              `json:"force"`
	Side              string              `json:"side"`
	FillPrice         float64             `json:"fillPrice,string"`
	TradeID           int64               `json:"tradeId,string"`
	BaseVolume        float64             `json:"baseVolume,string"`
	FillTime          UnixTimestamp       `json:"fillTime"`
	FillFee           float64             `json:"fillFee,string"`
	FillFeeCoin       string              `json:"fillFeeCoin"`
	TradeScope        string              `json:"tradeScope"`
	AccountBaseVolume float64             `json:"accBaseVolume,string"`
	PriceAverage      float64             `json:"priceAvg,string"`
	Status            string              `json:"status"`
	CreationTime      UnixTimestamp       `json:"cTime"`
	UpdateTime        UnixTimestamp       `json:"uTime"`
	STPMode           string              `json:"stpMode"`
	FeeDetail         []AbridgedFeeDetail `json:"feeDetail"`
	EnterPointSource  string              `json:"enterPointSource"`
}

// WsTriggerOrderSpotResponse contains information on a trigger order response for spot trading
type WsTriggerOrderSpotResponse struct {
	InstrumentID     string        `json:"instId"`
	OrderID          int64         `json:"orderId,string"`
	ClientOrderID    string        `json:"clientOid"`
	TriggerPrice     float64       `json:"triggerPrice,string"`
	TriggerType      string        `json:"triggerType"`
	PlanType         string        `json:"planType"`
	Price            float64       `json:"price,string"`
	Size             float64       `json:"size,string"`
	ActualSize       float64       `json:"actualSize,string"`
	OrderType        string        `json:"orderType"`
	Side             string        `json:"side"`
	Status           string        `json:"status"`
	ExecutePrice     float64       `json:"execPrice,string"`
	EnterPointSource string        `json:"enterPointSource"`
	CreationTime     UnixTimestamp `json:"cTime"`
	UpdateTime       UnixTimestamp `json:"uTime"`
	STPMode          string        `json:"stpMode"`
}

// WsAccountFuturesResponse contains information on an account response for futures trading
type WsAccountFuturesResponse struct {
	MarginCoin               string  `json:"marginCoin"`
	Frozen                   float64 `json:"frozen,string"`
	Available                float64 `json:"available,string"`
	MaxOpenPositionAvailable float64 `json:"maxOpenPositionAvailable,string"`
	MaxTransferOut           float64 `json:"maxTransferOut,string"`
	Equity                   float64 `json:"equity,string"`
	USDTEquity               float64 `json:"usdtEquity,string"`
}

// WsPositionResponse contains information on a position response
type WsPositionResponse struct {
	PositionID               int64         `json:"posId,string"`
	InstrumentID             string        `json:"instId"`
	MarginCoin               string        `json:"marginCoin"`
	MarginSize               float64       `json:"marginSize,string"`
	MarginMode               string        `json:"marginMode"`
	HoldSide                 string        `json:"holdSide"`
	PositionMode             string        `json:"posMode"`
	Total                    float64       `json:"total,string"`
	Available                float64       `json:"available,string"`
	Frozen                   float64       `json:"frozen,string"`
	OpenPriceAverage         float64       `json:"openPriceAvg,string"`
	Leverage                 float64       `json:"leverage,string"`
	AchievedProfits          float64       `json:"achievedProfits,string"`
	UnrealizedProfitLoss     float64       `json:"unrealizedPL,string"`
	UnrealizedProfitLossRate float64       `json:"unrealizedPLR,string"`
	LiquidationPrice         float64       `json:"liquidationPrice,string"`
	KeepMarginRate           float64       `json:"keepMarginRate,string"`
	MarginRate               float64       `json:"marginRate,string"`
	CreationTime             UnixTimestamp `json:"cTime"`
	BreakEvenPrice           float64       `json:"breakEvenPrice,string"`
	TotalFee                 float64       `json:"totalFee,string"`
	DeductedFee              float64       `json:"deductedFee,string"`
	UpdateTime               UnixTimestamp `json:"uTime"`
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
	Price        float64             `json:"price,string"`
	BaseVolume   float64             `json:"baseVolume,string"`
	QuoteVolume  float64             `json:"quoteVolume,string"`
	Profit       float64             `json:"profit,string"`
	TradeSide    string              `json:"tradeSide"`
	TradeScope   string              `json:"tradeScope"`
	FeeDetail    []AbridgedFeeDetail `json:"feeDetail"`
	CreationTime UnixTimestamp       `json:"cTime"`
	UpdateTime   UnixTimestamp       `json:"uTime"`
}

// WsOrderFuturesResponse contains information on an order response for futures trading
type WsOrderFuturesResponse struct {
	FilledQuantity   float64       `json:"accBaseVolume,string"`
	CreationTime     UnixTimestamp `json:"cTime"`
	ClientOrderID    string        `json:"clientOid"`
	FeeDetail        []FeeAndCoin  `json:"feeDetail"`
	FillFee          float64       `json:"fillFee,string"`
	FillFeeCoin      string        `json:"fillFeeCoin"`
	FillNotionalUSD  float64       `json:"fillNotionalUsd,string"`
	FillPrice        float64       `json:"fillPrice,string"`
	BaseVolume       float64       `json:"baseVolume,string"`
	FillTime         UnixTimestamp `json:"fillTime"`
	Force            string        `json:"force"`
	InstrumentID     string        `json:"instId"`
	Leverage         float64       `json:"leverage,string"`
	MarginCoin       string        `json:"marginCoin"`
	MarginMode       string        `json:"marginMode"`
	NotionalUSD      float64       `json:"notionalUsd,string"`
	OrderID          int64         `json:"orderId,string"`
	OrderType        string        `json:"orderType"`
	ProfitAndLoss    float64       `json:"pnl,string"`
	PositionMode     string        `json:"posMode"`
	PositionSide     string        `json:"posSide"`
	Price            float64       `json:"price,string"`
	PriceAverage     float64       `json:"priceAvg,string"`
	ReduceOnly       YesNoBool
	STPMode          string        `json:"stpMode"`
	Side             string        `json:"side"`
	Size             float64       `json:"size,string"`
	EnterPointSource string        `json:"enterPointSource"`
	Status           string        `json:"status"`
	TradeScope       string        `json:"tradeScope"`
	TradeID          int64         `json:"tradeId,string"`
	TradeSide        string        `json:"tradeSide"`
	UpdateTime       UnixTimestamp `json:"uTime"`
}

// WsTriggerOrderFuturesResponse contains information on a trigger order response for futures trading
type WsTriggerOrderFuturesResponse struct {
	InstrumentID           string        `json:"instId"`
	OrderID                int64         `json:"orderId,string"`
	ClientOrderID          string        `json:"clientOid"`
	TriggerPrice           float64       `json:"triggerPrice,string"`
	TriggerType            string        `json:"triggerType"`
	TriggerTime            UnixTimestamp `json:"triggerTime"`
	PlanType               string        `json:"planType"`
	Price                  float64       `json:"price,string"`
	Size                   float64       `json:"size,string"`
	ActualSize             float64       `json:"actualSize,string"`
	OrderType              string        `json:"orderType"`
	Side                   string        `json:"side"`
	TradeSide              string        `json:"tradeSide"`
	PositionSide           string        `json:"posSide"`
	MarginCoin             string        `json:"marginCoin"`
	Status                 string        `json:"status"`
	PositionMode           string        `json:"posMode"`
	EnterPointSource       string        `json:"enterPointSource"`
	StopSurplusTriggerType string        `json:"stopSurplusTriggerType"`
	StopLossTriggerType    string        `json:"stopLossTriggerType"`
	STPMode                string        `json:"stpMode"`
	CreationTime           UnixTimestamp `json:"cTime"`
	UpdateTime             UnixTimestamp `json:"uTime"`
}

// WsPositionHistoryResponse contains information on a position history response
type WsPositionHistoryResponse struct {
	PositionID        int64         `json:"posId,string"`
	InstrumentID      string        `json:"instId"`
	MarginCoin        string        `json:"marginCoin"`
	MarginMode        string        `json:"marginMode"`
	HoldSide          string        `json:"holdSide"`
	PositionMode      string        `json:"posMode"`
	OpenPriceAverage  float64       `json:"openPriceAvg,string"`
	ClosePriceAverage float64       `json:"closePriceAvg,string"`
	OpenSize          float64       `json:"openSize,string"`
	CloseSize         float64       `json:"closeSize,string"`
	AchievedProfits   float64       `json:"achievedProfits,string"`
	SettleFee         float64       `json:"settleFee,string"`
	OpenFee           float64       `json:"openFee,string"`
	CloseFee          float64       `json:"closeFee,string"`
	CreationTime      UnixTimestamp `json:"cTime"`
	UpdateTime        UnixTimestamp `json:"uTime"`
}

// WsIndexPriceResponse contains information on an index price response
type WsIndexPriceResponse struct {
	Symbol     string        `json:"symbol"`
	BaseCoin   string        `json:"baseCoin"`
	QuoteCoin  string        `json:"quoteCoin"`
	IndexPrice float64       `json:"indexPrice,string"`
	Timestamp  UnixTimestamp `json:"ts"`
}

// WsAccountCrossMarginResponse contains information on an account response for cross margin trading
type WsAccountCrossMarginResponse struct {
	UpdateTime UnixTimestamp `json:"uTime"`
	ID         int64         `json:"id,string"`
	Coin       string        `json:"coin"`
	Available  float64       `json:"available,string"`
	Borrow     float64       `json:"borrow,string"`
	Frozen     float64       `json:"frozen,string"`
	Interest   float64       `json:"interest,string"`
	Coupon     float64       `json:"coupon,string"`
}

// WsOrderCrossMarginResponse contains information on an order response for margin trading
type WsOrderMarginResponse struct {
	Force            string              `json:"force"`
	OrderType        string              `json:"orderType"`
	Price            float64             `json:"price,string"`
	QuoteSize        float64             `json:"quoteSize,string"`
	Side             string              `json:"side"`
	FeeDetail        []AbridgedFeeDetail `json:"feeDetail"`
	EnterPointSource string              `json:"enterPointSource"`
	Status           string              `json:"status"`
	BaseSize         float64             `json:"baseSize,string"`
	CreationTime     UnixTimestamp       `json:"cTime"`
	ClientOrderID    string              `json:"clientOid"`
	FillPrice        float64             `json:"fillPrice,string"`
	BaseVolume       float64             `json:"baseVolume,string"`
	FillTotalAmount  float64             `json:"fillTotalAmount,string"`
	LoanType         string              `json:"loanType"`
	OrderID          int64               `json:"orderId,string"`
	STPMode          string              `json:"stpMode"`
}

// WsAccountisolatedMarginResponse contains information on an account response for isolated margin trading
type WsAccountIsolatedMarginResponse struct {
	UpdateTime UnixTimestamp `json:"uTime"`
	ID         int64         `json:"id,string"`
	Coin       string        `json:"coin"`
	Symbol     string        `json:"symbol"`
	Available  float64       `json:"available,string"`
	Borrow     float64       `json:"borrow,string"`
	Frozen     float64       `json:"frozen,string"`
	Interest   float64       `json:"interest,string"`
	Coupon     float64       `json:"coupon,string"`
}
