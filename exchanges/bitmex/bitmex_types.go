package bitmex

import "github.com/thrasher-/gocryptotrader/decimal"

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
	Cidr        string        `json:"cidr"`
	Created     string        `json:"created"`
	Enabled     bool          `json:"enabled"`
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Nonce       int64         `json:"nonce"`
	Permissions []interface{} `json:"permissions"`
	Secret      string        `json:"secret"`
	UserID      int32         `json:"userId"`
}

// Chat Trollbox Data
type Chat struct {
	ChannelID decimal.Decimal `json:"channelID"`
	Date      string          `json:"date"`
	FromBot   bool            `json:"fromBot"`
	HTML      string          `json:"html"`
	ID        int32           `json:"id"`
	Message   string          `json:"message"`
	User      string          `json:"user"`
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
	Account               int64           `json:"account"`
	AvgPx                 decimal.Decimal `json:"avgPx"`
	ClOrdID               string          `json:"clOrdID"`
	ClOrdLinkID           string          `json:"clOrdLinkID"`
	Commission            decimal.Decimal `json:"commission"`
	ContingencyType       string          `json:"contingencyType"`
	CumQty                int64           `json:"cumQty"`
	Currency              string          `json:"currency"`
	DisplayQty            int64           `json:"displayQty"`
	ExDestination         string          `json:"exDestination"`
	ExecComm              int64           `json:"execComm"`
	ExecCost              int64           `json:"execCost"`
	ExecID                string          `json:"execID"`
	ExecInst              string          `json:"execInst"`
	ExecType              string          `json:"execType"`
	ForeignNotional       decimal.Decimal `json:"foreignNotional"`
	HomeNotional          decimal.Decimal `json:"homeNotional"`
	LastLiquidityInd      string          `json:"lastLiquidityInd"`
	LastMkt               string          `json:"lastMkt"`
	LastPx                decimal.Decimal `json:"lastPx"`
	LastQty               int64           `json:"lastQty"`
	LeavesQty             int64           `json:"leavesQty"`
	MultiLegReportingType string          `json:"multiLegReportingType"`
	OrdRejReason          string          `json:"ordRejReason"`
	OrdStatus             string          `json:"ordStatus"`
	OrdType               string          `json:"ordType"`
	OrderID               string          `json:"orderID"`
	OrderQty              int64           `json:"orderQty"`
	PegOffsetValue        decimal.Decimal `json:"pegOffsetValue"`
	PegPriceType          string          `json:"pegPriceType"`
	Price                 decimal.Decimal `json:"price"`
	SettlCurrency         string          `json:"settlCurrency"`
	Side                  string          `json:"side"`
	SimpleCumQty          decimal.Decimal `json:"simpleCumQty"`
	SimpleLeavesQty       decimal.Decimal `json:"simpleLeavesQty"`
	SimpleOrderQty        decimal.Decimal `json:"simpleOrderQty"`
	StopPx                decimal.Decimal `json:"stopPx"`
	Symbol                string          `json:"symbol"`
	Text                  string          `json:"text"`
	TimeInForce           string          `json:"timeInForce"`
	Timestamp             string          `json:"timestamp"`
	TradePublishIndicator string          `json:"tradePublishIndicator"`
	TransactTime          string          `json:"transactTime"`
	TrdMatchID            string          `json:"trdMatchID"`
	Triggered             string          `json:"triggered"`
	UnderlyingLastPx      decimal.Decimal `json:"underlyingLastPx"`
	WorkingIndicator      bool            `json:"workingIndicator"`
}

// Funding Swap Funding History
type Funding struct {
	FundingInterval  string          `json:"fundingInterval"`
	FundingRate      decimal.Decimal `json:"fundingRate"`
	FundingRateDaily decimal.Decimal `json:"fundingRateDaily"`
	Symbol           string          `json:"symbol"`
	Timestamp        string          `json:"timestamp"`
}

// Instrument Tradeable Contracts, Indices, and History
type Instrument struct {
	AskPrice                       decimal.Decimal `json:"askPrice"`
	BankruptLimitDownPrice         decimal.Decimal `json:"bankruptLimitDownPrice"`
	BankruptLimitUpPrice           decimal.Decimal `json:"bankruptLimitUpPrice"`
	BidPrice                       decimal.Decimal `json:"bidPrice"`
	BuyLeg                         string          `json:"buyLeg"`
	CalcInterval                   string          `json:"calcInterval"`
	Capped                         bool            `json:"capped"`
	ClosingTimestamp               string          `json:"closingTimestamp"`
	Deleverage                     bool            `json:"deleverage"`
	Expiry                         string          `json:"expiry"`
	FairBasis                      decimal.Decimal `json:"fairBasis"`
	FairBasisRate                  decimal.Decimal `json:"fairBasisRate"`
	FairMethod                     string          `json:"fairMethod"`
	FairPrice                      decimal.Decimal `json:"fairPrice"`
	Front                          string          `json:"front"`
	FundingBaseSymbol              string          `json:"fundingBaseSymbol"`
	FundingInterval                string          `json:"fundingInterval"`
	FundingPremiumSymbol           string          `json:"fundingPremiumSymbol"`
	FundingQuoteSymbol             string          `json:"fundingQuoteSymbol"`
	FundingRate                    decimal.Decimal `json:"fundingRate"`
	FundingTimestamp               string          `json:"fundingTimestamp"`
	HasLiquidity                   bool            `json:"hasLiquidity"`
	HighPrice                      decimal.Decimal `json:"highPrice"`
	ImpactAskPrice                 decimal.Decimal `json:"impactAskPrice"`
	ImpactBidPrice                 decimal.Decimal `json:"impactBidPrice"`
	ImpactMidPrice                 decimal.Decimal `json:"impactMidPrice"`
	IndicativeFundingRate          decimal.Decimal `json:"indicativeFundingRate"`
	IndicativeSettlePrice          decimal.Decimal `json:"indicativeSettlePrice"`
	IndicativeTaxRate              decimal.Decimal `json:"indicativeTaxRate"`
	InitMargin                     decimal.Decimal `json:"initMargin"`
	InsuranceFee                   decimal.Decimal `json:"insuranceFee"`
	InverseLeg                     string          `json:"inverseLeg"`
	IsInverse                      bool            `json:"isInverse"`
	IsQuanto                       bool            `json:"isQuanto"`
	LastChangePcnt                 decimal.Decimal `json:"lastChangePcnt"`
	LastPrice                      decimal.Decimal `json:"lastPrice"`
	LastPriceProtected             decimal.Decimal `json:"lastPriceProtected"`
	LastTickDirection              string          `json:"lastTickDirection"`
	Limit                          decimal.Decimal `json:"limit"`
	LimitDownPrice                 decimal.Decimal `json:"limitDownPrice"`
	LimitUpPrice                   decimal.Decimal `json:"limitUpPrice"`
	Listing                        string          `json:"listing"`
	LotSize                        int64           `json:"lotSize"`
	LowPrice                       decimal.Decimal `json:"lowPrice"`
	MaintMargin                    decimal.Decimal `json:"maintMargin"`
	MakerFee                       decimal.Decimal `json:"makerFee"`
	MarkMethod                     string          `json:"markMethod"`
	MarkPrice                      decimal.Decimal `json:"markPrice"`
	MaxOrderQty                    int64           `json:"maxOrderQty"`
	MaxPrice                       decimal.Decimal `json:"maxPrice"`
	MidPrice                       decimal.Decimal `json:"midPrice"`
	Multiplier                     int64           `json:"multiplier"`
	OpenInterest                   int64           `json:"openInterest"`
	OpenValue                      int64           `json:"openValue"`
	OpeningTimestamp               string          `json:"openingTimestamp"`
	OptionMultiplier               decimal.Decimal `json:"optionMultiplier"`
	OptionStrikePcnt               decimal.Decimal `json:"optionStrikePcnt"`
	OptionStrikePrice              decimal.Decimal `json:"optionStrikePrice"`
	OptionStrikeRound              decimal.Decimal `json:"optionStrikeRound"`
	OptionUnderlyingPrice          decimal.Decimal `json:"optionUnderlyingPrice"`
	PositionCurrency               string          `json:"positionCurrency"`
	PrevClosePrice                 decimal.Decimal `json:"prevClosePrice"`
	PrevPrice24h                   decimal.Decimal `json:"prevPrice24h"`
	PrevTotalTurnover              int64           `json:"prevTotalTurnover"`
	PrevTotalVolume                int64           `json:"prevTotalVolume"`
	PublishInterval                string          `json:"publishInterval"`
	PublishTime                    string          `json:"publishTime"`
	QuoteCurrency                  string          `json:"quoteCurrency"`
	QuoteToSettleMultiplier        int64           `json:"quoteToSettleMultiplier"`
	RebalanceInterval              string          `json:"rebalanceInterval"`
	RebalanceTimestamp             string          `json:"rebalanceTimestamp"`
	Reference                      string          `json:"reference"`
	ReferenceSymbol                string          `json:"referenceSymbol"`
	RelistInterval                 string          `json:"relistInterval"`
	RiskLimit                      int64           `json:"riskLimit"`
	RiskStep                       int64           `json:"riskStep"`
	RootSymbol                     string          `json:"rootSymbol"`
	SellLeg                        string          `json:"sellLeg"`
	SessionInterval                string          `json:"sessionInterval"`
	SettlCurrency                  string          `json:"settlCurrency"`
	Settle                         string          `json:"settle"`
	SettledPrice                   decimal.Decimal `json:"settledPrice"`
	SettlementFee                  decimal.Decimal `json:"settlementFee"`
	State                          string          `json:"state"`
	Symbol                         string          `json:"symbol"`
	TakerFee                       decimal.Decimal `json:"takerFee"`
	Taxed                          bool            `json:"taxed"`
	TickSize                       decimal.Decimal `json:"tickSize"`
	Timestamp                      string          `json:"timestamp"`
	TotalTurnover                  int64           `json:"totalTurnover"`
	TotalVolume                    int64           `json:"totalVolume"`
	Turnover                       int64           `json:"turnover"`
	Turnover24h                    int64           `json:"turnover24h"`
	Typ                            string          `json:"typ"`
	Underlying                     string          `json:"underlying"`
	UnderlyingSymbol               string          `json:"underlyingSymbol"`
	UnderlyingToPositionMultiplier int64           `json:"underlyingToPositionMultiplier"`
	UnderlyingToSettleMultiplier   int64           `json:"underlyingToSettleMultiplier"`
	Volume                         int64           `json:"volume"`
	Volume24h                      int64           `json:"volume24h"`
	Vwap                           decimal.Decimal `json:"vwap"`
}

// InstrumentInterval instrument interval
type InstrumentInterval struct {
	Intervals []string `json:"intervals"`
	Symbols   []string `json:"symbols"`
}

// IndexComposite index composite
type IndexComposite struct {
	IndexSymbol string          `json:"indexSymbol"`
	LastPrice   decimal.Decimal `json:"lastPrice"`
	Logged      string          `json:"logged"`
	Reference   string          `json:"reference"`
	Symbol      string          `json:"symbol"`
	Timestamp   string          `json:"timestamp"`
	Weight      decimal.Decimal `json:"weight"`
}

// Insurance Insurance Fund Data
type Insurance struct {
	Currency      string `json:"currency"`
	Timestamp     string `json:"timestamp"`
	WalletBalance int64  `json:"walletBalance"`
}

// Leaderboard Information on Top Users
type Leaderboard struct {
	IsRealName bool            `json:"isRealName"`
	Name       string          `json:"name"`
	Profit     decimal.Decimal `json:"profit"`
}

// Alias Name refers to Trollbox client name
type Alias struct {
	Name string `json:"name"`
}

// Liquidation Active Liquidations
type Liquidation struct {
	LeavesQty int64           `json:"leavesQty"`
	OrderID   string          `json:"orderID"`
	Price     decimal.Decimal `json:"price"`
	Side      string          `json:"side"`
	Symbol    string          `json:"symbol"`
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
	Account               int64           `json:"account"`
	AvgPx                 decimal.Decimal `json:"avgPx"`
	ClOrdID               string          `json:"clOrdID"`
	ClOrdLinkID           string          `json:"clOrdLinkID"`
	ContingencyType       string          `json:"contingencyType"`
	CumQty                int64           `json:"cumQty"`
	Currency              string          `json:"currency"`
	DisplayQty            int64           `json:"displayQty"`
	ExDestination         string          `json:"exDestination"`
	ExecInst              string          `json:"execInst"`
	LeavesQty             int64           `json:"leavesQty"`
	MultiLegReportingType string          `json:"multiLegReportingType"`
	OrdRejReason          string          `json:"ordRejReason"`
	OrdStatus             string          `json:"ordStatus"`
	OrdType               string          `json:"ordType"`
	OrderID               string          `json:"orderID"`
	OrderQty              int64           `json:"orderQty"`
	PegOffsetValue        decimal.Decimal `json:"pegOffsetValue"`
	PegPriceType          string          `json:"pegPriceType"`
	Price                 decimal.Decimal `json:"price"`
	SettlCurrency         string          `json:"settlCurrency"`
	Side                  string          `json:"side"`
	SimpleCumQty          decimal.Decimal `json:"simpleCumQty"`
	SimpleLeavesQty       decimal.Decimal `json:"simpleLeavesQty"`
	SimpleOrderQty        decimal.Decimal `json:"simpleOrderQty"`
	StopPx                decimal.Decimal `json:"stopPx"`
	Symbol                string          `json:"symbol"`
	Text                  string          `json:"text"`
	TimeInForce           string          `json:"timeInForce"`
	Timestamp             string          `json:"timestamp"`
	TransactTime          string          `json:"transactTime"`
	Triggered             string          `json:"triggered"`
	WorkingIndicator      bool            `json:"workingIndicator"`
}

// OrderBookL2 contains order book l2
type OrderBookL2 struct {
	ID     int64           `json:"id"`
	Price  decimal.Decimal `json:"price"`
	Side   string          `json:"side"`
	Size   int64           `json:"size"`
	Symbol string          `json:"symbol"`
}

// Position Summary of Open and Closed Positions
type Position struct {
	Account              int64           `json:"account"`
	AvgCostPrice         decimal.Decimal `json:"avgCostPrice"`
	AvgEntryPrice        decimal.Decimal `json:"avgEntryPrice"`
	BankruptPrice        decimal.Decimal `json:"bankruptPrice"`
	BreakEvenPrice       decimal.Decimal `json:"breakEvenPrice"`
	Commission           decimal.Decimal `json:"commission"`
	CrossMargin          bool            `json:"crossMargin"`
	Currency             string          `json:"currency"`
	CurrentComm          int64           `json:"currentComm"`
	CurrentCost          int64           `json:"currentCost"`
	CurrentQty           int64           `json:"currentQty"`
	CurrentTimestamp     string          `json:"currentTimestamp"`
	DeleveragePercentile decimal.Decimal `json:"deleveragePercentile"`
	ExecBuyCost          int64           `json:"execBuyCost"`
	ExecBuyQty           int64           `json:"execBuyQty"`
	ExecComm             int64           `json:"execComm"`
	ExecCost             int64           `json:"execCost"`
	ExecQty              int64           `json:"execQty"`
	ExecSellCost         int64           `json:"execSellCost"`
	ExecSellQty          int64           `json:"execSellQty"`
	ForeignNotional      decimal.Decimal `json:"foreignNotional"`
	GrossExecCost        int64           `json:"grossExecCost"`
	GrossOpenCost        int64           `json:"grossOpenCost"`
	GrossOpenPremium     int64           `json:"grossOpenPremium"`
	HomeNotional         decimal.Decimal `json:"homeNotional"`
	IndicativeTax        int64           `json:"indicativeTax"`
	IndicativeTaxRate    decimal.Decimal `json:"indicativeTaxRate"`
	InitMargin           int64           `json:"initMargin"`
	InitMarginReq        decimal.Decimal `json:"initMarginReq"`
	IsOpen               bool            `json:"isOpen"`
	LastPrice            decimal.Decimal `json:"lastPrice"`
	LastValue            int64           `json:"lastValue"`
	Leverage             decimal.Decimal `json:"leverage"`
	LiquidationPrice     decimal.Decimal `json:"liquidationPrice"`
	LongBankrupt         int64           `json:"longBankrupt"`
	MaintMargin          int64           `json:"maintMargin"`
	MaintMarginReq       decimal.Decimal `json:"maintMarginReq"`
	MarginCallPrice      decimal.Decimal `json:"marginCallPrice"`
	MarkPrice            decimal.Decimal `json:"markPrice"`
	MarkValue            int64           `json:"markValue"`
	OpenOrderBuyCost     int64           `json:"openOrderBuyCost"`
	OpenOrderBuyPremium  int64           `json:"openOrderBuyPremium"`
	OpenOrderBuyQty      int64           `json:"openOrderBuyQty"`
	OpenOrderSellCost    int64           `json:"openOrderSellCost"`
	OpenOrderSellPremium int64           `json:"openOrderSellPremium"`
	OpenOrderSellQty     int64           `json:"openOrderSellQty"`
	OpeningComm          int64           `json:"openingComm"`
	OpeningCost          int64           `json:"openingCost"`
	OpeningQty           int64           `json:"openingQty"`
	OpeningTimestamp     string          `json:"openingTimestamp"`
	PosAllowance         int64           `json:"posAllowance"`
	PosComm              int64           `json:"posComm"`
	PosCost              int64           `json:"posCost"`
	PosCost2             int64           `json:"posCost2"`
	PosCross             int64           `json:"posCross"`
	PosInit              int64           `json:"posInit"`
	PosLoss              int64           `json:"posLoss"`
	PosMaint             int64           `json:"posMaint"`
	PosMargin            int64           `json:"posMargin"`
	PosState             string          `json:"posState"`
	PrevClosePrice       decimal.Decimal `json:"prevClosePrice"`
	PrevRealisedPnl      int64           `json:"prevRealisedPnl"`
	PrevUnrealisedPnl    int64           `json:"prevUnrealisedPnl"`
	QuoteCurrency        string          `json:"quoteCurrency"`
	RealisedCost         int64           `json:"realisedCost"`
	RealisedGrossPnl     int64           `json:"realisedGrossPnl"`
	RealisedPnl          int64           `json:"realisedPnl"`
	RealisedTax          int64           `json:"realisedTax"`
	RebalancedPnl        int64           `json:"rebalancedPnl"`
	RiskLimit            int64           `json:"riskLimit"`
	RiskValue            int64           `json:"riskValue"`
	SessionMargin        int64           `json:"sessionMargin"`
	ShortBankrupt        int64           `json:"shortBankrupt"`
	SimpleCost           decimal.Decimal `json:"simpleCost"`
	SimplePnl            decimal.Decimal `json:"simplePnl"`
	SimplePnlPcnt        decimal.Decimal `json:"simplePnlPcnt"`
	SimpleQty            decimal.Decimal `json:"simpleQty"`
	SimpleValue          decimal.Decimal `json:"simpleValue"`
	Symbol               string          `json:"symbol"`
	TargetExcessMargin   int64           `json:"targetExcessMargin"`
	TaxBase              int64           `json:"taxBase"`
	TaxableMargin        int64           `json:"taxableMargin"`
	Timestamp            string          `json:"timestamp"`
	Underlying           string          `json:"underlying"`
	UnrealisedCost       int64           `json:"unrealisedCost"`
	UnrealisedGrossPnl   int64           `json:"unrealisedGrossPnl"`
	UnrealisedPnl        int64           `json:"unrealisedPnl"`
	UnrealisedPnlPcnt    decimal.Decimal `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt    decimal.Decimal `json:"unrealisedRoePcnt"`
	UnrealisedTax        int64           `json:"unrealisedTax"`
	VarMargin            int64           `json:"varMargin"`
}

// Quote Best Bid/Offer Snapshots & Historical Bins
type Quote struct {
	AskPrice  decimal.Decimal `json:"askPrice"`
	AskSize   int64           `json:"askSize"`
	BidPrice  decimal.Decimal `json:"bidPrice"`
	BidSize   int64           `json:"bidSize"`
	Symbol    string          `json:"symbol"`
	Timestamp string          `json:"timestamp"`
}

// Settlement Historical Settlement Data
type Settlement struct {
	Bankrupt              int64           `json:"bankrupt"`
	OptionStrikePrice     decimal.Decimal `json:"optionStrikePrice"`
	OptionUnderlyingPrice decimal.Decimal `json:"optionUnderlyingPrice"`
	SettledPrice          decimal.Decimal `json:"settledPrice"`
	SettlementType        string          `json:"settlementType"`
	Symbol                string          `json:"symbol"`
	TaxBase               int64           `json:"taxBase"`
	TaxRate               decimal.Decimal `json:"taxRate"`
	Timestamp             string          `json:"timestamp"`
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
	Currency   string `json:"currency"`
	Date       string `json:"date"`
	RootSymbol string `json:"rootSymbol"`
	Turnover   int64  `json:"turnover"`
	Volume     int64  `json:"volume"`
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
	ForeignNotional decimal.Decimal `json:"foreignNotional"`
	GrossValue      int64           `json:"grossValue"`
	HomeNotional    decimal.Decimal `json:"homeNotional"`
	Price           decimal.Decimal `json:"price"`
	Side            string          `json:"side"`
	Size            int64           `json:"size"`
	Symbol          string          `json:"symbol"`
	TickDirection   string          `json:"tickDirection"`
	Timestamp       string          `json:"timestamp"`
	TrdMatchID      string          `json:"trdMatchID"`
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
	AlertOnLiquidations     bool            `json:"alertOnLiquidations"`
	AnimationsEnabled       bool            `json:"animationsEnabled"`
	AnnouncementsLastSeen   string          `json:"announcementsLastSeen"`
	ChatChannelID           decimal.Decimal `json:"chatChannelID"`
	ColorTheme              string          `json:"colorTheme"`
	Currency                string          `json:"currency"`
	Debug                   bool            `json:"debug"`
	DisableEmails           []string        `json:"disableEmails"`
	HideConfirmDialogs      []string        `json:"hideConfirmDialogs"`
	HideConnectionModal     bool            `json:"hideConnectionModal"`
	HideFromLeaderboard     bool            `json:"hideFromLeaderboard"`
	HideNameFromLeaderboard bool            `json:"hideNameFromLeaderboard"`
	HideNotifications       []string        `json:"hideNotifications"`
	Locale                  string          `json:"locale"`
	MsgsSeen                []string        `json:"msgsSeen"`
	OrderBookBinning        interface{}     `json:"orderBookBinning"`
	OrderBookType           string          `json:"orderBookType"`
	OrderClearImmediate     bool            `json:"orderClearImmediate"`
	OrderControlsPlusMinus  bool            `json:"orderControlsPlusMinus"`
	ShowLocaleNumbers       bool            `json:"showLocaleNumbers"`
	Sounds                  []string        `json:"sounds"`
	StrictIPCheck           bool            `json:"strictIPCheck"`
	StrictTimeout           bool            `json:"strictTimeout"`
	TickerGroup             string          `json:"tickerGroup"`
	TickerPinned            bool            `json:"tickerPinned"`
	TradeLayout             string          `json:"tradeLayout"`
}

// AffiliateStatus affiliate Status details
type AffiliateStatus struct {
	Account         int64           `json:"account"`
	Currency        string          `json:"currency"`
	ExecComm        int64           `json:"execComm"`
	ExecTurnover    int64           `json:"execTurnover"`
	PayoutPcnt      decimal.Decimal `json:"payoutPcnt"`
	PendingPayout   int64           `json:"pendingPayout"`
	PrevComm        int64           `json:"prevComm"`
	PrevPayout      int64           `json:"prevPayout"`
	PrevTimestamp   string          `json:"prevTimestamp"`
	PrevTurnover    int64           `json:"prevTurnover"`
	ReferrerAccount decimal.Decimal `json:"referrerAccount"`
	Timestamp       string          `json:"timestamp"`
	TotalComm       int64           `json:"totalComm"`
	TotalReferrals  int64           `json:"totalReferrals"`
	TotalTurnover   int64           `json:"totalTurnover"`
}

// TransactionInfo Information
type TransactionInfo struct {
	Account        int64  `json:"account"`
	Address        string `json:"address"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Fee            int64  `json:"fee"`
	Text           string `json:"text"`
	Timestamp      string `json:"timestamp"`
	TransactID     string `json:"transactID"`
	TransactStatus string `json:"transactStatus"`
	TransactTime   string `json:"transactTime"`
	TransactType   string `json:"transactType"`
	Tx             string `json:"tx"`
}

// UserCommission user commission
type UserCommission struct {
	MakerFee      decimal.Decimal `json:"makerFee"`
	MaxFee        decimal.Decimal `json:"maxFee"`
	SettlementFee decimal.Decimal `json:"settlementFee"`
	TakerFee      decimal.Decimal `json:"takerFee"`
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
	Account            int64           `json:"account"`
	Action             string          `json:"action"`
	Amount             int64           `json:"amount"`
	AvailableMargin    int64           `json:"availableMargin"`
	Commission         decimal.Decimal `json:"commission"`
	ConfirmedDebit     int64           `json:"confirmedDebit"`
	Currency           string          `json:"currency"`
	ExcessMargin       int64           `json:"excessMargin"`
	ExcessMarginPcnt   decimal.Decimal `json:"excessMarginPcnt"`
	GrossComm          int64           `json:"grossComm"`
	GrossExecCost      int64           `json:"grossExecCost"`
	GrossLastValue     int64           `json:"grossLastValue"`
	GrossMarkValue     int64           `json:"grossMarkValue"`
	GrossOpenCost      int64           `json:"grossOpenCost"`
	GrossOpenPremium   int64           `json:"grossOpenPremium"`
	IndicativeTax      int64           `json:"indicativeTax"`
	InitMargin         int64           `json:"initMargin"`
	MaintMargin        int64           `json:"maintMargin"`
	MarginBalance      int64           `json:"marginBalance"`
	MarginBalancePcnt  decimal.Decimal `json:"marginBalancePcnt"`
	MarginLeverage     decimal.Decimal `json:"marginLeverage"`
	MarginUsedPcnt     decimal.Decimal `json:"marginUsedPcnt"`
	PendingCredit      int64           `json:"pendingCredit"`
	PendingDebit       int64           `json:"pendingDebit"`
	PrevRealisedPnl    int64           `json:"prevRealisedPnl"`
	PrevState          string          `json:"prevState"`
	PrevUnrealisedPnl  int64           `json:"prevUnrealisedPnl"`
	RealisedPnl        int64           `json:"realisedPnl"`
	RiskLimit          int64           `json:"riskLimit"`
	RiskValue          int64           `json:"riskValue"`
	SessionMargin      int64           `json:"sessionMargin"`
	State              string          `json:"state"`
	SyntheticMargin    int64           `json:"syntheticMargin"`
	TargetExcessMargin int64           `json:"targetExcessMargin"`
	TaxableMargin      int64           `json:"taxableMargin"`
	Timestamp          string          `json:"timestamp"`
	UnrealisedPnl      int64           `json:"unrealisedPnl"`
	UnrealisedProfit   int64           `json:"unrealisedProfit"`
	VarMargin          int64           `json:"varMargin"`
	WalletBalance      int64           `json:"walletBalance"`
	WithdrawableMargin int64           `json:"withdrawableMargin"`
}

// MinWithdrawalFee minimum withdrawal fee information
type MinWithdrawalFee struct {
	Currency string `json:"currency"`
	Fee      int64  `json:"fee"`
	MinFee   int64  `json:"minFee"`
}

// WalletInfo wallet information
type WalletInfo struct {
	Account          int64    `json:"account"`
	Addr             string   `json:"addr"`
	Amount           int64    `json:"amount"`
	ConfirmedDebit   int64    `json:"confirmedDebit"`
	Currency         string   `json:"currency"`
	DeltaAmount      int64    `json:"deltaAmount"`
	DeltaDeposited   int64    `json:"deltaDeposited"`
	DeltaTransferIn  int64    `json:"deltaTransferIn"`
	DeltaTransferOut int64    `json:"deltaTransferOut"`
	DeltaWithdrawn   int64    `json:"deltaWithdrawn"`
	Deposited        int64    `json:"deposited"`
	PendingCredit    int64    `json:"pendingCredit"`
	PendingDebit     int64    `json:"pendingDebit"`
	PrevAmount       int64    `json:"prevAmount"`
	PrevDeposited    int64    `json:"prevDeposited"`
	PrevTimestamp    string   `json:"prevTimestamp"`
	PrevTransferIn   int64    `json:"prevTransferIn"`
	PrevTransferOut  int64    `json:"prevTransferOut"`
	PrevWithdrawn    int64    `json:"prevWithdrawn"`
	Script           string   `json:"script"`
	Timestamp        string   `json:"timestamp"`
	TransferIn       int64    `json:"transferIn"`
	TransferOut      int64    `json:"transferOut"`
	WithdrawalLock   []string `json:"withdrawalLock"`
	Withdrawn        int64    `json:"withdrawn"`
}
