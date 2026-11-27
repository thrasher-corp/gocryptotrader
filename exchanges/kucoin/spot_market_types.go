package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AnnouncementsRequest holds the optional filters for GetAnnouncements
type AnnouncementsRequest struct {
	CurrentPage      uint64
	PageSize         uint64
	AnnouncementType string // For example latest-announcements, new-listings or delistings
	Language         string // For example en_US
	StartTime        time.Time
	EndTime          time.Time
}

// AnnouncementsResponse holds a page of announcements
type AnnouncementsResponse struct {
	TotalNumber uint64         `json:"totalNum"`
	Items       []Announcement `json:"items"`
	CurrentPage uint64         `json:"currentPage"`
	PageSize    uint64         `json:"pageSize"`
	TotalPage   uint64         `json:"totalPage"`
}

// Announcement holds an announcement
type Announcement struct {
	AnnouncementID          uint64     `json:"annId"`
	AnnouncementTitle       string     `json:"annTitle"`
	AnnouncementType        []string   `json:"annType"`
	AnnouncementDescription string     `json:"annDesc"`
	CreateTime              types.Time `json:"cTime"` // Release time
	Language                string     `json:"language"`
	AnnouncementURL         string     `json:"annUrl"`
}

// CurrencyResponse holds a currency and its chains
type CurrencyResponse struct {
	Currency        currency.Code   `json:"currency"`
	Name            string          `json:"name"` // Changes when the currency is renamed; Currency does not
	FullName        string          `json:"fullName"`
	Precision       uint64          `json:"precision"`
	Confirms        uint64          `json:"confirms"`
	ContractAddress string          `json:"contractAddress"`
	IsMarginEnabled bool            `json:"isMarginEnabled"`
	IsDebitEnabled  bool            `json:"isDebitEnabled"`
	Chains          []CurrencyChain `json:"chains"`
}

// CurrencyChain holds a currency's deposit and withdrawal rules on one chain
type CurrencyChain struct {
	ChainName         string       `json:"chainName"`
	WithdrawalMinSize types.Number `json:"withdrawalMinSize"` // Abandoned by KuCoin in favour of WithdrawMinSize
	WithdrawMinSize   types.Number `json:"withdrawMinSize"`
	DepositMinSize    types.Number `json:"depositMinSize"`
	WithdrawFeeRate   types.Number `json:"withdrawFeeRate"`
	WithdrawalMinFee  types.Number `json:"withdrawalMinFee"` // Abandoned by KuCoin in favour of WithdrawMinFee
	WithdrawMinFee    types.Number `json:"withdrawMinFee"`
	IsWithdrawEnabled bool         `json:"isWithdrawEnabled"`
	IsDepositEnabled  bool         `json:"isDepositEnabled"`
	Confirms          uint64       `json:"confirms"`
	PreConfirms       uint64       `json:"preConfirms"`
	ContractAddress   string       `json:"contractAddress"`
	WithdrawPrecision uint64       `json:"withdrawPrecision"`
	MaxWithdraw       types.Number `json:"maxWithdraw"`
	MaxDeposit        types.Number `json:"maxDeposit"` // Only set for the Lightning Network
	NeedTag           bool         `json:"needTag"`
	ChainID           string       `json:"chainId"`
	DepositFeeRate    types.Number `json:"depositFeeRate"`
	WithdrawMaxFee    types.Number `json:"withdrawMaxFee"`
	DepositTierFee    string       `json:"depositTierFee"` // Undescribed by KuCoin and never seen populated, so its format is unknown
	AddressRegex      string       `json:"addressRegex"`
	MemoRegex         string       `json:"memoRegex"`
}

// SymbolResponse holds a spot symbol's trading rules
type SymbolResponse struct {
	Symbol                          string        `json:"symbol"`
	Name                            string        `json:"name"` // Changes when the symbol is renamed; Symbol does not
	BaseCurrency                    currency.Code `json:"baseCurrency"`
	QuoteCurrency                   currency.Code `json:"quoteCurrency"`
	FeeCurrency                     currency.Code `json:"feeCurrency"`
	Market                          string        `json:"market"`
	BaseMinSize                     types.Number  `json:"baseMinSize"`
	QuoteMinSize                    types.Number  `json:"quoteMinSize"`
	BaseMaxSize                     types.Number  `json:"baseMaxSize"`
	QuoteMaxSize                    types.Number  `json:"quoteMaxSize"`
	BaseIncrement                   types.Number  `json:"baseIncrement"`
	QuoteIncrement                  types.Number  `json:"quoteIncrement"`
	PriceIncrement                  types.Number  `json:"priceIncrement"`
	PriceLimitRate                  types.Number  `json:"priceLimitRate"`
	MinFunds                        types.Number  `json:"minFunds"`
	IsMarginEnabled                 bool          `json:"isMarginEnabled"`
	EnableTrading                   bool          `json:"enableTrading"`
	FeeCategory                     uint64        `json:"feeCategory"`
	MakerFeeCoefficient             types.Number  `json:"makerFeeCoefficient"`
	TakerFeeCoefficient             types.Number  `json:"takerFeeCoefficient"`
	SpecialTreatment                bool          `json:"st"`
	CallAuctionIsEnabled            bool          `json:"callauctionIsEnabled"`
	CallAuctionPriceFloor           types.Number  `json:"callauctionPriceFloor"`
	CallAuctionPriceCeiling         types.Number  `json:"callauctionPriceCeiling"`
	CallAuctionFirstStageStartTime  types.Time    `json:"callauctionFirstStageStartTime"`
	CallAuctionSecondStageStartTime types.Time    `json:"callauctionSecondStageStartTime"`
	CallAuctionThirdStageStartTime  types.Time    `json:"callauctionThirdStageStartTime"`
	TradingStartTime                types.Time    `json:"tradingStartTime"`
}

// TickerResponse holds a symbol's level 1 market data
type TickerResponse struct {
	Time        types.Time   `json:"time"`
	Sequence    types.Number `json:"sequence"`
	Price       types.Number `json:"price"`
	Size        types.Number `json:"size"`
	BestBid     types.Number `json:"bestBid"`
	BestBidSize types.Number `json:"bestBidSize"`
	BestAsk     types.Number `json:"bestAsk"`
	BestAskSize types.Number `json:"bestAskSize"`
}

// TickersResponse holds every symbol's 24 hour ticker
type TickersResponse struct {
	Time   types.Time   `json:"time"`
	Ticker []TickerInfo `json:"ticker"`
}

// TickerInfo holds a symbol's 24 hour ticker
type TickerInfo struct {
	Symbol             string       `json:"symbol"`
	SymbolName         string       `json:"symbolName"`
	Buy                types.Number `json:"buy"`
	BestBidSize        types.Number `json:"bestBidSize"`
	Sell               types.Number `json:"sell"`
	BestAskSize        types.Number `json:"bestAskSize"`
	ChangeRate         types.Number `json:"changeRate"`
	ChangePrice        types.Number `json:"changePrice"`
	Open               types.Number `json:"open"`
	High               types.Number `json:"high"`
	Low                types.Number `json:"low"`
	Volume             types.Number `json:"vol"`
	VolumeValue        types.Number `json:"volValue"`
	Last               types.Number `json:"last"`
	LastSize           types.Number `json:"lastSize"`
	AveragePrice       types.Number `json:"averagePrice"`
	TakerFeeRate       types.Number `json:"takerFeeRate"`
	MakerFeeRate       types.Number `json:"makerFeeRate"`
	TakerCoefficient   types.Number `json:"takerCoefficient"`
	MakerCoefficient   types.Number `json:"makerCoefficient"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
}

// TradeResponse holds a public trade
type TradeResponse struct {
	Sequence string       `json:"sequence"` // Kept as a string, since it exceeds the integers a float64 holds exactly
	TradeID  string       `json:"tradeId"`
	Price    types.Number `json:"price"`
	Size     types.Number `json:"size"`
	Side     string       `json:"side"` // The taker's side
	Time     types.Time   `json:"time"`
}

// KlinesRequest holds the parameters for GetKlines
type KlinesRequest struct {
	Symbol  string
	Type    string // 1min, 3min, 5min, 15min, 30min, 1hour, 2hour, 4hour, 6hour, 8hour, 12hour, 1day, 1week or 1month
	StartAt time.Time
	EndAt   time.Time
}

// KlineResponse holds a candle
type KlineResponse struct {
	StartTime types.Time
	Open      types.Number
	Close     types.Number
	High      types.Number
	Low       types.Number
	Volume    types.Number // Base currency volume, counting one side of each trade
	Amount    types.Number // Quote currency amount, counting one side of each trade
}

// UnmarshalJSON decodes a candle from its array form
func (k *KlineResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.Open, &k.Close, &k.High, &k.Low, &k.Volume, &k.Amount})
}

// OrderbookResponse holds an orderbook snapshot aggregated by price
type OrderbookResponse struct {
	Time     types.Time                       `json:"time"`
	Sequence types.Number                     `json:"sequence"` // -1 from the call auction orderbook outside an auction
	Bids     orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks     orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// CallAuctionInfoResponse holds a symbol's call auction estimates and price ranges
type CallAuctionInfoResponse struct {
	Symbol                  string       `json:"symbol"`
	EstimatedPrice          types.Number `json:"estimatedPrice"`
	EstimatedSize           types.Number `json:"estimatedSize"`
	SellOrderRangeLowPrice  types.Number `json:"sellOrderRangeLowPrice"`
	SellOrderRangeHighPrice types.Number `json:"sellOrderRangeHighPrice"`
	BuyOrderRangeLowPrice   types.Number `json:"buyOrderRangeLowPrice"`
	BuyOrderRangeHighPrice  types.Number `json:"buyOrderRangeHighPrice"`
	Time                    types.Time   `json:"time"`
}

// Stats24HourResponse holds a symbol's statistics over the last 24 hours
type Stats24HourResponse struct {
	Time             types.Time   `json:"time"`
	Symbol           string       `json:"symbol"`
	Buy              types.Number `json:"buy"`
	Sell             types.Number `json:"sell"`
	ChangeRate       types.Number `json:"changeRate"`
	ChangePrice      types.Number `json:"changePrice"`
	High             types.Number `json:"high"`
	Low              types.Number `json:"low"`
	Volume           types.Number `json:"vol"`
	VolumeValue      types.Number `json:"volValue"`
	Last             types.Number `json:"last"`
	AveragePrice     types.Number `json:"averagePrice"`
	TakerFeeRate     types.Number `json:"takerFeeRate"`
	MakerFeeRate     types.Number `json:"makerFeeRate"`
	TakerCoefficient types.Number `json:"takerCoefficient"`
	MakerCoefficient types.Number `json:"makerCoefficient"`
}

// ServiceStatusResponse holds the spot service status
type ServiceStatusResponse struct {
	Status  string `json:"status"` // open, close or cancelonly
	Message string `json:"msg"`
}

// KYCRegionResponse holds a region users can verify their identity in
type KYCRegionResponse struct {
	Code        string `json:"code"`
	EnglishName string `json:"enName"`
}
