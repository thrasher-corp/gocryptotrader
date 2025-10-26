package bitmex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// RequestError allows for a general error capture from requests
type RequestError struct {
	Error struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	} `json:"error"`
}

// Announcement General Announcements
type Announcement struct {
	Content string `json:"content"`
	Date    string `json:"date"`
	ID      int32  `json:"id"`
	Link    string `json:"link"`
	Title   string `json:"title"`
}

// APIKey Persistent API Keys for Developers
type APIKey struct {
	Cidr        string `json:"cidr"`
	Created     string `json:"created"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Nonce       int64  `json:"nonce"`
	Permissions []any  `json:"permissions"`
	Secret      string `json:"secret"`
	UserID      int32  `json:"userId"`
}

// Chat Trollbox Data
type Chat struct {
	ChannelID float64 `json:"channelID"`
	Date      string  `json:"date"`
	FromBot   bool    `json:"fromBot"`
	HTML      string  `json:"html"`
	ID        int32   `json:"id"`
	Message   string  `json:"message"`
	User      string  `json:"user"`
}

// ChatChannel chat channel
type ChatChannel struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// ConnectedUsers connected users
type ConnectedUsers struct {
	Bots  int32 `json:"bots"`
	Users int32 `json:"users"`
}

// Execution Raw Order and Balance Data
type Execution struct {
	Account               int64     `json:"account"`
	AvgPx                 float64   `json:"avgPx"`
	ClOrdID               string    `json:"clOrdID"`
	ClOrdLinkID           string    `json:"clOrdLinkID"`
	Commission            float64   `json:"commission"`
	ContingencyType       string    `json:"contingencyType"`
	CumQty                int64     `json:"cumQty"`
	Currency              string    `json:"currency"`
	DisplayQuantity       int64     `json:"displayQty"`
	ExDestination         string    `json:"exDestination"`
	ExecComm              int64     `json:"execComm"`
	ExecCost              int64     `json:"execCost"`
	ExecID                string    `json:"execID"`
	ExecInst              string    `json:"execInst"`
	ExecType              string    `json:"execType"`
	ForeignNotional       float64   `json:"foreignNotional"`
	HomeNotional          float64   `json:"homeNotional"`
	LastLiquidityInd      string    `json:"lastLiquidityInd"`
	LastMkt               string    `json:"lastMkt"`
	LastPx                float64   `json:"lastPx"`
	LastQty               int64     `json:"lastQty"`
	LeavesQty             int64     `json:"leavesQty"`
	MultiLegReportingType string    `json:"multiLegReportingType"`
	OrdRejReason          string    `json:"ordRejReason"`
	OrdStatus             string    `json:"ordStatus"`
	OrdType               string    `json:"ordType"`
	OrderID               string    `json:"orderID"`
	OrderQty              int64     `json:"orderQty"`
	PegOffsetValue        float64   `json:"pegOffsetValue"`
	PegPriceType          string    `json:"pegPriceType"`
	Price                 float64   `json:"price"`
	SettlCurrency         string    `json:"settlCurrency"`
	Side                  string    `json:"side"`
	SimpleCumQty          float64   `json:"simpleCumQty"`
	SimpleLeavesQty       float64   `json:"simpleLeavesQty"`
	SimpleOrderQty        float64   `json:"simpleOrderQty"`
	StopPx                float64   `json:"stopPx"`
	Symbol                string    `json:"symbol"`
	Text                  string    `json:"text"`
	TimeInForce           string    `json:"timeInForce"`
	Timestamp             time.Time `json:"timestamp"`
	TradePublishIndicator string    `json:"tradePublishIndicator"`
	TransactTime          time.Time `json:"transactTime"`
	TrdMatchID            string    `json:"trdMatchID"`
	Triggered             string    `json:"triggered"`
	UnderlyingLastPx      float64   `json:"underlyingLastPx"`
	WorkingIndicator      bool      `json:"workingIndicator"`
}

// Funding Swap Funding History
type Funding struct {
	FundingInterval  string    `json:"fundingInterval"`
	FundingRate      float64   `json:"fundingRate"`
	FundingRateDaily float64   `json:"fundingRateDaily"`
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
}

// Instrument Tradeable Contracts, Indices, and History
type Instrument struct {
	AskPrice                       float64       `json:"askPrice"`
	BankruptLimitDownPrice         float64       `json:"bankruptLimitDownPrice"`
	BankruptLimitUpPrice           float64       `json:"bankruptLimitUpPrice"`
	BidPrice                       float64       `json:"bidPrice"`
	BuyLeg                         string        `json:"buyLeg"`
	CalcInterval                   string        `json:"calcInterval"`
	Capped                         bool          `json:"capped"`
	ClosingTimestamp               time.Time     `json:"closingTimestamp"`
	Deleverage                     bool          `json:"deleverage"`
	Expiry                         string        `json:"expiry"`
	FairBasis                      float64       `json:"fairBasis"`
	FairBasisRate                  float64       `json:"fairBasisRate"`
	FairMethod                     string        `json:"fairMethod"`
	FairPrice                      float64       `json:"fairPrice"`
	Front                          string        `json:"front"`
	FundingBaseSymbol              string        `json:"fundingBaseSymbol"`
	FundingInterval                time.Time     `json:"fundingInterval"`
	FundingPremiumSymbol           string        `json:"fundingPremiumSymbol"`
	FundingQuoteSymbol             string        `json:"fundingQuoteSymbol"`
	FundingRate                    float64       `json:"fundingRate"`
	FundingTimestamp               time.Time     `json:"fundingTimestamp"`
	HasLiquidity                   bool          `json:"hasLiquidity"`
	HighPrice                      float64       `json:"highPrice"`
	ImpactAskPrice                 float64       `json:"impactAskPrice"`
	ImpactBidPrice                 float64       `json:"impactBidPrice"`
	ImpactMidPrice                 float64       `json:"impactMidPrice"`
	IndicativeFundingRate          float64       `json:"indicativeFundingRate"`
	IndicativeSettlePrice          float64       `json:"indicativeSettlePrice"`
	IndicativeTaxRate              float64       `json:"indicativeTaxRate"`
	InitMargin                     float64       `json:"initMargin"`
	InsuranceFee                   float64       `json:"insuranceFee"`
	InverseLeg                     string        `json:"inverseLeg"`
	IsInverse                      bool          `json:"isInverse"`
	IsQuanto                       bool          `json:"isQuanto"`
	LastChangePcnt                 float64       `json:"lastChangePcnt"`
	LastPrice                      float64       `json:"lastPrice"`
	LastPriceProtected             float64       `json:"lastPriceProtected"`
	LastTickDirection              string        `json:"lastTickDirection"`
	Limit                          float64       `json:"limit"`
	LimitDownPrice                 float64       `json:"limitDownPrice"`
	LimitUpPrice                   float64       `json:"limitUpPrice"`
	Listing                        string        `json:"listing"`
	LotSize                        float64       `json:"lotSize"`
	LowPrice                       float64       `json:"lowPrice"`
	MaintMargin                    float64       `json:"maintMargin"`
	MakerFee                       float64       `json:"makerFee"`
	MarkMethod                     string        `json:"markMethod"`
	MarkPrice                      float64       `json:"markPrice"`
	MaxOrderQty                    float64       `json:"maxOrderQty"`
	MaxPrice                       float64       `json:"maxPrice"`
	MidPrice                       float64       `json:"midPrice"`
	Multiplier                     float64       `json:"multiplier"`
	OpenInterest                   float64       `json:"openInterest"`
	OpenValue                      float64       `json:"openValue"`
	OpeningTimestamp               time.Time     `json:"openingTimestamp"`
	OptionMultiplier               float64       `json:"optionMultiplier"`
	OptionStrikePcnt               float64       `json:"optionStrikePcnt"`
	OptionStrikePrice              float64       `json:"optionStrikePrice"`
	OptionStrikeRound              float64       `json:"optionStrikeRound"`
	OptionUnderlyingPrice          float64       `json:"optionUnderlyingPrice"`
	PositionCurrency               string        `json:"positionCurrency"`
	PrevClosePrice                 float64       `json:"prevClosePrice"`
	PrevPrice24h                   float64       `json:"prevPrice24h"`
	PrevTotalTurnover              float64       `json:"prevTotalTurnover"`
	PrevTotalVolume                float64       `json:"prevTotalVolume"`
	PublishInterval                string        `json:"publishInterval"`
	PublishTime                    time.Time     `json:"publishTime"`
	QuoteCurrency                  string        `json:"quoteCurrency"`
	QuoteToSettleMultiplier        int64         `json:"quoteToSettleMultiplier"`
	RebalanceInterval              string        `json:"rebalanceInterval"`
	RebalanceTimestamp             time.Time     `json:"rebalanceTimestamp"`
	Reference                      string        `json:"reference"`
	ReferenceSymbol                string        `json:"referenceSymbol"`
	RelistInterval                 string        `json:"relistInterval"`
	RiskLimit                      float64       `json:"riskLimit"`
	RiskStep                       float64       `json:"riskStep"`
	RootSymbol                     string        `json:"rootSymbol"`
	SellLeg                        string        `json:"sellLeg"`
	SessionInterval                string        `json:"sessionInterval"`
	SettlementCurrency             currency.Code `json:"settlCurrency"`
	Settle                         string        `json:"settle"`
	SettledPrice                   float64       `json:"settledPrice"`
	SettlementFee                  float64       `json:"settlementFee"`
	State                          string        `json:"state"`
	Symbol                         string        `json:"symbol"`
	TakerFee                       float64       `json:"takerFee"`
	Taxed                          bool          `json:"taxed"`
	TickSize                       float64       `json:"tickSize"`
	Timestamp                      time.Time     `json:"timestamp"`
	TotalTurnover                  float64       `json:"totalTurnover"`
	TotalVolume                    float64       `json:"totalVolume"`
	Turnover                       float64       `json:"turnover"`
	Turnover24h                    float64       `json:"turnover24h"`
	Typ                            string        `json:"typ"`
	Underlying                     string        `json:"underlying"`
	UnderlyingSymbol               string        `json:"underlyingSymbol"`
	UnderlyingToPositionMultiplier float64       `json:"underlyingToPositionMultiplier"`
	UnderlyingToSettleMultiplier   float64       `json:"underlyingToSettleMultiplier"`
	Volume                         float64       `json:"volume"`
	Volume24h                      float64       `json:"volume24h"`
	Vwap                           float64       `json:"vwap"`
}

// InstrumentInterval instrument interval
type InstrumentInterval struct {
	Intervals []string `json:"intervals"`
	Symbols   []string `json:"symbols"`
}

// IndexComposite index composite
type IndexComposite struct {
	IndexSymbol string    `json:"indexSymbol"`
	LastPrice   float64   `json:"lastPrice"`
	Logged      string    `json:"logged"`
	Reference   string    `json:"reference"`
	Symbol      string    `json:"symbol"`
	Timestamp   time.Time `json:"timestamp"`
	Weight      float64   `json:"weight"`
}

// Insurance stores insurance fund data
type Insurance struct {
	Currency      string    `json:"currency"`
	Timestamp     time.Time `json:"timestamp"`
	WalletBalance int64     `json:"walletBalance"`
}

// Leaderboard Information on Top Users
type Leaderboard struct {
	IsRealName bool    `json:"isRealName"`
	Name       string  `json:"name"`
	Profit     float64 `json:"profit"`
}

// Alias Name refers to Trollbox client name
type Alias struct {
	Name string `json:"name"`
}

// Liquidation Active Liquidations
type Liquidation struct {
	LeavesQty int64   `json:"leavesQty"`
	OrderID   string  `json:"orderID"`
	Price     float64 `json:"price"`
	Side      string  `json:"side"`
	Symbol    string  `json:"symbol"`
}

// Notification Account Notifications
type Notification struct {
	Body              string `json:"body"`
	Closable          bool   `json:"closable"`
	Date              string `json:"date"`
	ID                int32  `json:"id"`
	Persist           bool   `json:"persist"`
	Sound             string `json:"sound"`
	Title             string `json:"title"`
	TTL               int32  `json:"ttl"`
	Type              string `json:"type"`
	WaitForVisibility bool   `json:"waitForVisibility"`
}

// Order Placement, Cancellation, Amending, and History
type Order struct {
	Account               int64     `json:"account"`
	AvgPx                 float64   `json:"avgPx"`
	ClOrdID               string    `json:"clOrdID"`
	ClOrdLinkID           string    `json:"clOrdLinkID"`
	ContingencyType       string    `json:"contingencyType"`
	CumQty                float64   `json:"cumQty"`
	Currency              string    `json:"currency"`
	DisplayQuantity       int64     `json:"displayQty"`
	ExDestination         string    `json:"exDestination"`
	ExecInst              string    `json:"execInst"`
	LeavesQty             float64   `json:"leavesQty"`
	MultiLegReportingType string    `json:"multiLegReportingType"`
	OrdRejReason          string    `json:"ordRejReason"`
	OrdStatus             string    `json:"ordStatus"`
	OrdType               int64     `json:"ordType,string"`
	OrderID               string    `json:"orderID"`
	OrderQty              float64   `json:"orderQty"`
	PegOffsetValue        float64   `json:"pegOffsetValue"`
	PegPriceType          string    `json:"pegPriceType"`
	Price                 float64   `json:"price"`
	SettlCurrency         string    `json:"settlCurrency"`
	Side                  int64     `json:"side,string"`
	SimpleCumQty          float64   `json:"simpleCumQty"`
	SimpleLeavesQty       float64   `json:"simpleLeavesQty"`
	SimpleOrderQty        float64   `json:"simpleOrderQty"`
	StopPx                float64   `json:"stopPx"`
	Symbol                string    `json:"symbol"`
	Text                  string    `json:"text"`
	TimeInForce           string    `json:"timeInForce"`
	Timestamp             time.Time `json:"timestamp"`
	TransactTime          time.Time `json:"transactTime"`
	Triggered             string    `json:"triggered"`
	WorkingIndicator      bool      `json:"workingIndicator"`
}

// OrderBookL2 contains order book l2
type OrderBookL2 struct {
	ID        int64     `json:"id"`
	Price     float64   `json:"price"`
	Side      string    `json:"side"`
	Size      int64     `json:"size"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
}

// Position Summary of Open and Closed Positions
type Position struct {
	Account              int64     `json:"account"`
	AvgCostPrice         float64   `json:"avgCostPrice"`
	AvgEntryPrice        float64   `json:"avgEntryPrice"`
	BankruptPrice        float64   `json:"bankruptPrice"`
	BreakEvenPrice       float64   `json:"breakEvenPrice"`
	Commission           float64   `json:"commission"`
	CrossMargin          bool      `json:"crossMargin"`
	Currency             string    `json:"currency"`
	CurrentComm          int64     `json:"currentComm"`
	CurrentCost          int64     `json:"currentCost"`
	CurrentQty           int64     `json:"currentQty"`
	CurrentTimestamp     time.Time `json:"currentTimestamp"`
	DeleveragePercentile float64   `json:"deleveragePercentile"`
	ExecBuyCost          int64     `json:"execBuyCost"`
	ExecBuyQty           int64     `json:"execBuyQty"`
	ExecComm             int64     `json:"execComm"`
	ExecCost             int64     `json:"execCost"`
	ExecQty              int64     `json:"execQty"`
	ExecSellCost         int64     `json:"execSellCost"`
	ExecSellQty          int64     `json:"execSellQty"`
	ForeignNotional      float64   `json:"foreignNotional"`
	GrossExecCost        int64     `json:"grossExecCost"`
	GrossOpenCost        int64     `json:"grossOpenCost"`
	GrossOpenPremium     int64     `json:"grossOpenPremium"`
	HomeNotional         float64   `json:"homeNotional"`
	IndicativeTax        int64     `json:"indicativeTax"`
	IndicativeTaxRate    float64   `json:"indicativeTaxRate"`
	InitMargin           int64     `json:"initMargin"`
	InitMarginReq        float64   `json:"initMarginReq"`
	IsOpen               bool      `json:"isOpen"`
	LastPrice            float64   `json:"lastPrice"`
	LastValue            int64     `json:"lastValue"`
	Leverage             float64   `json:"leverage"`
	LiquidationPrice     float64   `json:"liquidationPrice"`
	LongBankrupt         int64     `json:"longBankrupt"`
	MaintMargin          int64     `json:"maintMargin"`
	MaintMarginReq       float64   `json:"maintMarginReq"`
	MarginCallPrice      float64   `json:"marginCallPrice"`
	MarkPrice            float64   `json:"markPrice"`
	MarkValue            int64     `json:"markValue"`
	OpenOrderBuyCost     int64     `json:"openOrderBuyCost"`
	OpenOrderBuyPremium  int64     `json:"openOrderBuyPremium"`
	OpenOrderBuyQty      int64     `json:"openOrderBuyQty"`
	OpenOrderSellCost    int64     `json:"openOrderSellCost"`
	OpenOrderSellPremium int64     `json:"openOrderSellPremium"`
	OpenOrderSellQty     int64     `json:"openOrderSellQty"`
	OpeningComm          int64     `json:"openingComm"`
	OpeningCost          int64     `json:"openingCost"`
	OpeningQty           int64     `json:"openingQty"`
	OpeningTimestamp     time.Time `json:"openingTimestamp"`
	PosAllowance         int64     `json:"posAllowance"`
	PosComm              int64     `json:"posComm"`
	PosCost              int64     `json:"posCost"`
	PosCost2             int64     `json:"posCost2"`
	PosCross             int64     `json:"posCross"`
	PosInit              int64     `json:"posInit"`
	PosLoss              int64     `json:"posLoss"`
	PosMaint             int64     `json:"posMaint"`
	PosMargin            int64     `json:"posMargin"`
	PosState             string    `json:"posState"`
	PrevClosePrice       float64   `json:"prevClosePrice"`
	PrevRealisedPnl      int64     `json:"prevRealisedPnl"`
	PrevUnrealisedPnl    int64     `json:"prevUnrealisedPnl"`
	QuoteCurrency        string    `json:"quoteCurrency"`
	RealisedCost         int64     `json:"realisedCost"`
	RealisedGrossPnl     int64     `json:"realisedGrossPnl"`
	RealisedPnl          int64     `json:"realisedPnl"`
	RealisedTax          int64     `json:"realisedTax"`
	RebalancedPnl        int64     `json:"rebalancedPnl"`
	RiskLimit            int64     `json:"riskLimit"`
	RiskValue            int64     `json:"riskValue"`
	SessionMargin        int64     `json:"sessionMargin"`
	ShortBankrupt        int64     `json:"shortBankrupt"`
	SimpleCost           float64   `json:"simpleCost"`
	SimplePnl            float64   `json:"simplePnl"`
	SimplePnlPcnt        float64   `json:"simplePnlPcnt"`
	SimpleQty            float64   `json:"simpleQty"`
	SimpleValue          float64   `json:"simpleValue"`
	Symbol               string    `json:"symbol"`
	TargetExcessMargin   int64     `json:"targetExcessMargin"`
	TaxBase              int64     `json:"taxBase"`
	TaxableMargin        int64     `json:"taxableMargin"`
	Timestamp            time.Time `json:"timestamp"`
	Underlying           string    `json:"underlying"`
	UnrealisedCost       int64     `json:"unrealisedCost"`
	UnrealisedGrossPnl   int64     `json:"unrealisedGrossPnl"`
	UnrealisedPnl        int64     `json:"unrealisedPnl"`
	UnrealisedPnlPcnt    float64   `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt    float64   `json:"unrealisedRoePcnt"`
	UnrealisedTax        int64     `json:"unrealisedTax"`
	VarMargin            int64     `json:"varMargin"`
}

// Quote Best Bid/Offer Snapshots & Historical Bins
type Quote struct {
	AskPrice  float64   `json:"askPrice"`
	AskSize   int64     `json:"askSize"`
	BidPrice  float64   `json:"bidPrice"`
	BidSize   int64     `json:"bidSize"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
}

// Settlement Historical Settlement Data
type Settlement struct {
	Bankrupt              int64     `json:"bankrupt"`
	OptionStrikePrice     float64   `json:"optionStrikePrice"`
	OptionUnderlyingPrice float64   `json:"optionUnderlyingPrice"`
	SettledPrice          float64   `json:"settledPrice"`
	SettlementType        string    `json:"settlementType"`
	Symbol                string    `json:"symbol"`
	TaxBase               int64     `json:"taxBase"`
	TaxRate               float64   `json:"taxRate"`
	Timestamp             time.Time `json:"timestamp"`
}

// Stats Exchange Statistics
type Stats struct {
	Currency     string `json:"currency"`
	OpenInterest int64  `json:"openInterest"`
	OpenValue    int64  `json:"openValue"`
	RootSymbol   string `json:"rootSymbol"`
	Turnover24h  int64  `json:"turnover24h"`
	Volume24h    int64  `json:"volume24h"`
}

// StatsHistory stats history
type StatsHistory struct {
	Currency   string    `json:"currency"`
	Date       time.Time `json:"date"`
	RootSymbol string    `json:"rootSymbol"`
	Turnover   int64     `json:"turnover"`
	Volume     int64     `json:"volume"`
}

// StatsUSD contains summary of exchange stats
type StatsUSD struct {
	Currency     string `json:"currency"`
	RootSymbol   string `json:"rootSymbol"`
	Turnover     int64  `json:"turnover"`
	Turnover24h  int64  `json:"turnover24h"`
	Turnover30d  int64  `json:"turnover30d"`
	Turnover365d int64  `json:"turnover365d"`
}

// Trade Individual & Bucketed Trades
type Trade struct {
	ForeignNotional float64   `json:"foreignNotional"`
	GrossValue      int64     `json:"grossValue"`
	HomeNotional    float64   `json:"homeNotional"`
	Price           float64   `json:"price"`
	Side            string    `json:"side"`
	Size            int64     `json:"size"`
	Symbol          string    `json:"symbol"`
	TickDirection   string    `json:"tickDirection"`
	Timestamp       time.Time `json:"timestamp"`
	TrdMatchID      string    `json:"trdMatchID"`
}

// User Account Operations
type User struct {
	TFAEnabled   string          `json:"TFAEnabled"`
	AffiliateID  string          `json:"affiliateID"`
	Country      string          `json:"country"`
	Created      string          `json:"created"`
	Email        string          `json:"email"`
	Firstname    string          `json:"firstname"`
	GeoipCountry string          `json:"geoipCountry"`
	GeoipRegion  string          `json:"geoipRegion"`
	ID           int32           `json:"id"`
	LastUpdated  string          `json:"lastUpdated"`
	Lastname     string          `json:"lastname"`
	OwnerID      int32           `json:"ownerId"`
	PgpPubKey    string          `json:"pgpPubKey"`
	Phone        string          `json:"phone"`
	Preferences  UserPreferences `json:"preferences"`
	Typ          string          `json:"typ"`
	Username     string          `json:"username"`
}

// UserPreferences user preferences
type UserPreferences struct {
	AlertOnLiquidations     bool     `json:"alertOnLiquidations"`
	AnimationsEnabled       bool     `json:"animationsEnabled"`
	AnnouncementsLastSeen   string   `json:"announcementsLastSeen"`
	ChatChannelID           float64  `json:"chatChannelID"`
	ColorTheme              string   `json:"colorTheme"`
	Currency                string   `json:"currency"`
	Debug                   bool     `json:"debug"`
	DisableEmails           []string `json:"disableEmails"`
	HideConfirmDialogs      []string `json:"hideConfirmDialogs"`
	HideConnectionModal     bool     `json:"hideConnectionModal"`
	HideFromLeaderboard     bool     `json:"hideFromLeaderboard"`
	HideNameFromLeaderboard bool     `json:"hideNameFromLeaderboard"`
	HideNotifications       []string `json:"hideNotifications"`
	Locale                  string   `json:"locale"`
	MsgsSeen                []string `json:"msgsSeen"`
	OrderBookBinning        any      `json:"orderBookBinning"`
	OrderBookType           string   `json:"orderBookType"`
	OrderClearImmediate     bool     `json:"orderClearImmediate"`
	OrderControlsPlusMinus  bool     `json:"orderControlsPlusMinus"`
	ShowLocaleNumbers       bool     `json:"showLocaleNumbers"`
	Sounds                  []string `json:"sounds"`
	StrictIPCheck           bool     `json:"strictIPCheck"`
	StrictTimeout           bool     `json:"strictTimeout"`
	TickerGroup             string   `json:"tickerGroup"`
	TickerPinned            bool     `json:"tickerPinned"`
	TradeLayout             string   `json:"tradeLayout"`
}

// AffiliateStatus affiliate Status details
type AffiliateStatus struct {
	Account         int64     `json:"account"`
	Currency        string    `json:"currency"`
	ExecComm        int64     `json:"execComm"`
	ExecTurnover    int64     `json:"execTurnover"`
	PayoutPcnt      float64   `json:"payoutPcnt"`
	PendingPayout   int64     `json:"pendingPayout"`
	PrevComm        int64     `json:"prevComm"`
	PrevPayout      int64     `json:"prevPayout"`
	PrevTimestamp   time.Time `json:"prevTimestamp"`
	PrevTurnover    int64     `json:"prevTurnover"`
	ReferrerAccount float64   `json:"referrerAccount"`
	Timestamp       time.Time `json:"timestamp"`
	TotalComm       int64     `json:"totalComm"`
	TotalReferrals  int64     `json:"totalReferrals"`
	TotalTurnover   int64     `json:"totalTurnover"`
}

// TransactionInfo Information
type TransactionInfo struct {
	Account        int64     `json:"account"`
	Address        string    `json:"address"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Fee            float64   `json:"fee"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
	TransactID     string    `json:"transactID"`
	Network        string    `json:"network"`
	TransactStatus string    `json:"transactStatus"`
	TransactTime   time.Time `json:"transactTime"`
	TransactType   string    `json:"transactType"`
	Tx             string    `json:"tx"`
}

// UserCommission user commission
type UserCommission struct {
	MakerFee      float64 `json:"makerFee"`
	MaxFee        float64 `json:"maxFee"`
	SettlementFee float64 `json:"settlementFee"`
	TakerFee      float64 `json:"takerFee"`
}

// ConfirmEmail confirmatin email endpoint data
type ConfirmEmail struct {
	ID      string `json:"id"`
	TTL     int64  `json:"ttl"`
	Created string `json:"created"`
	UserID  int64  `json:"userId"`
}

// UserMargin margin information
type UserMargin struct {
	Account            int64     `json:"account"`
	Action             string    `json:"action"`
	Amount             int64     `json:"amount"`
	AvailableMargin    int64     `json:"availableMargin"`
	Commission         float64   `json:"commission"`
	ConfirmedDebit     int64     `json:"confirmedDebit"`
	Currency           string    `json:"currency"`
	ExcessMargin       int64     `json:"excessMargin"`
	ExcessMarginPcnt   float64   `json:"excessMarginPcnt"`
	GrossComm          int64     `json:"grossComm"`
	GrossExecCost      int64     `json:"grossExecCost"`
	GrossLastValue     int64     `json:"grossLastValue"`
	GrossMarkValue     int64     `json:"grossMarkValue"`
	GrossOpenCost      int64     `json:"grossOpenCost"`
	GrossOpenPremium   int64     `json:"grossOpenPremium"`
	IndicativeTax      int64     `json:"indicativeTax"`
	InitMargin         int64     `json:"initMargin"`
	MaintMargin        int64     `json:"maintMargin"`
	MarginBalance      int64     `json:"marginBalance"`
	MarginBalancePcnt  float64   `json:"marginBalancePcnt"`
	MarginLeverage     float64   `json:"marginLeverage"`
	MarginUsedPcnt     float64   `json:"marginUsedPcnt"`
	PendingCredit      int64     `json:"pendingCredit"`
	PendingDebit       int64     `json:"pendingDebit"`
	PrevRealisedPnl    int64     `json:"prevRealisedPnl"`
	PrevState          string    `json:"prevState"`
	PrevUnrealisedPnl  int64     `json:"prevUnrealisedPnl"`
	RealisedPnl        int64     `json:"realisedPnl"`
	RiskLimit          int64     `json:"riskLimit"`
	RiskValue          int64     `json:"riskValue"`
	SessionMargin      int64     `json:"sessionMargin"`
	State              string    `json:"state"`
	SyntheticMargin    int64     `json:"syntheticMargin"`
	TargetExcessMargin int64     `json:"targetExcessMargin"`
	TaxableMargin      int64     `json:"taxableMargin"`
	Timestamp          time.Time `json:"timestamp"`
	UnrealisedPnl      int64     `json:"unrealisedPnl"`
	UnrealisedProfit   int64     `json:"unrealisedProfit"`
	VarMargin          int64     `json:"varMargin"`
	WalletBalance      int64     `json:"walletBalance"`
	WithdrawableMargin int64     `json:"withdrawableMargin"`
}

// MinWithdrawalFee minimum withdrawal fee information
type MinWithdrawalFee struct {
	Currency string `json:"currency"`
	Fee      int64  `json:"fee"`
	MinFee   int64  `json:"minFee"`
}

// WalletInfo wallet information
type WalletInfo struct {
	Account          int64         `json:"account"`
	Addr             string        `json:"addr"`
	Amount           float64       `json:"amount"`
	ConfirmedDebit   int64         `json:"confirmedDebit"`
	Currency         currency.Code `json:"currency"`
	DeltaAmount      int64         `json:"deltaAmount"`
	DeltaDeposited   int64         `json:"deltaDeposited"`
	DeltaTransferIn  int64         `json:"deltaTransferIn"`
	DeltaTransferOut int64         `json:"deltaTransferOut"`
	DeltaWithdrawn   int64         `json:"deltaWithdrawn"`
	Deposited        int64         `json:"deposited"`
	PendingCredit    int64         `json:"pendingCredit"`
	PendingDebit     int64         `json:"pendingDebit"`
	PrevAmount       int64         `json:"prevAmount"`
	PrevDeposited    int64         `json:"prevDeposited"`
	PrevTimestamp    time.Time     `json:"prevTimestamp"`
	PrevTransferIn   int64         `json:"prevTransferIn"`
	PrevTransferOut  int64         `json:"prevTransferOut"`
	PrevWithdrawn    int64         `json:"prevWithdrawn"`
	Script           string        `json:"script"`
	Timestamp        time.Time     `json:"timestamp"`
	TransferIn       int64         `json:"transferIn"`
	TransferOut      int64         `json:"transferOut"`
	WithdrawalLock   []string      `json:"withdrawalLock"`
	Withdrawn        int64         `json:"withdrawn"`
}

// orderTypeMap holds order type info based on Bitmex data
var orderTypeMap = map[int64]order.Type{
	1: order.Market,
	2: order.Limit,
	3: order.Stop,
	7: order.TrailingStop,
}

// orderSideMap holds order type info based on Bitmex data
var orderSideMap = map[int64]order.Side{
	1: order.Buy,
	2: order.Sell,
}
