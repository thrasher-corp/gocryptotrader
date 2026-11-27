package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SubscriptionArgument holds a channel subscription argument, as sent in a
// subscribe request and echoed in the arg object of every push
type SubscriptionArgument struct {
	Channel          string        `json:"channel,omitempty"`
	InstrumentType   string        `json:"instType,omitempty"`
	InstrumentFamily string        `json:"instFamily,omitempty"`
	InstrumentID     currency.Pair `json:"instId,omitzero"`
	SpreadID         currency.Pair `json:"sprdId,omitzero"`
	AlgoID           string        `json:"algoId,omitempty"`
	Currency         currency.Code `json:"ccy,omitzero"`
	ExtraParams      string        `json:"extraParams,omitempty"`
	UserID           string        `json:"uid,omitempty"`
}

// WebsocketSubscriptionRequest holds a subscribe or unsubscribe request; OKX
// echoes its ID in the reply to each argument
type WebsocketSubscriptionRequest struct {
	ID        string                 `json:"id,omitempty"`
	Operation string                 `json:"op"`
	Arguments []SubscriptionArgument `json:"args"`
	subs      subscription.List
	replies   chan []byte
}

// WebsocketLoginRequest holds a login request
type WebsocketLoginRequest struct {
	Operation string                   `json:"op"`
	Arguments []WebsocketLoginArgument `json:"args"`
}

// WebsocketLoginArgument holds the credentials and signature of a login request
type WebsocketLoginArgument struct {
	APIKey     string `json:"apiKey"`
	Passphrase string `json:"passphrase"`
	Timestamp  uint64 `json:"timestamp,string"` // Unix seconds
	Sign       string `json:"sign"`
}

// WebsocketEventResponse holds the routing fields of an incoming frame: the
// event, error and notice frames carry no data, and every push carries its
// subscription argument
type WebsocketEventResponse struct {
	ID              string               `json:"id"`
	Event           string               `json:"event"`
	Argument        SubscriptionArgument `json:"arg"`
	Code            string               `json:"code"`
	Message         string               `json:"msg"`
	ConnectionID    string               `json:"connId"`
	Channel         string               `json:"channel"`
	ConnectionCount types.Number         `json:"connCount"`
}

// WebsocketNotice holds a notice OKX sends on a connection, such as the warning
// that it closes soon for a service upgrade
type WebsocketNotice struct {
	Code         string `json:"code"`
	Message      string `json:"msg"`
	ConnectionID string `json:"connId"`
}

// WebsocketOperationRequest holds an order operation request
type WebsocketOperationRequest struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Arguments any    `json:"args"`
}

// WebsocketOperationResponse holds an order operation reply; Data decodes into
// the operation's result type
type WebsocketOperationResponse struct {
	ID        string     `json:"id"`
	Operation string     `json:"op"`
	Code      uint64     `json:"code,string"`
	Message   string     `json:"msg"`
	Data      any        `json:"data"`
	InTime    types.Time `json:"inTime"`
	OutTime   types.Time `json:"outTime"`
}

// MassCancelOrderRequest holds the parameters for cancelling all MMP pending
// orders of an instrument family
type MassCancelOrderRequest struct {
	InstrumentType   string `json:"instType"`
	InstrumentFamily string `json:"instFamily"`
	LockInterval     uint64 `json:"lockInterval,omitempty,string"` // Milliseconds, 0 to 10000
}

// WebsocketCancelSpreadOrderArgument holds the order a spread cancel operation cancels
type WebsocketCancelSpreadOrderArgument struct {
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// WebsocketCancelAllSpreadOrdersArgument holds the spread whose orders a spread mass cancel operation cancels
type WebsocketCancelAllSpreadOrdersArgument struct {
	SpreadID string `json:"sprdId,omitempty"`
}

// WebsocketAccountPush holds an account channel push
type WebsocketAccountPush struct {
	Argument    SubscriptionArgument `json:"arg"`
	EventType   string               `json:"eventType"`
	CurrentPage uint64               `json:"curPage"`
	LastPage    bool                 `json:"lastPage"`
	Data        []WebsocketAccount   `json:"data"`
}

// WebsocketAccount holds the account information an account channel push carries
type WebsocketAccount struct {
	UpdateTime                   types.Time               `json:"uTime"`
	TotalEquity                  types.Number             `json:"totalEq"`
	IsolatedMarginEquity         types.Number             `json:"isoEq"`
	AdjustedEquity               types.Number             `json:"adjEq"`
	AvailableEquity              types.Number             `json:"availEq"`
	OrderFrozen                  types.Number             `json:"ordFroz"`
	InitialMarginRequirement     types.Number             `json:"imr"`
	MaintenanceMarginRequirement types.Number             `json:"mmr"`
	BorrowFrozen                 types.Number             `json:"borrowFroz"`
	MarginRatio                  types.Number             `json:"mgnRatio"`
	NotionalUSD                  types.Number             `json:"notionalUsd"`
	NotionalUSDForBorrow         types.Number             `json:"notionalUsdForBorrow"`
	NotionalUSDForSwap           types.Number             `json:"notionalUsdForSwap"`
	NotionalUSDForFutures        types.Number             `json:"notionalUsdForFutures"`
	NotionalUSDForOption         types.Number             `json:"notionalUsdForOption"`
	UnrealisedProfitAndLoss      types.Number             `json:"upl"`
	Delta                        types.Number             `json:"delta"`
	DeltaLeverage                types.Number             `json:"deltaLever"`
	DeltaNeutralStatus           string                   `json:"deltaNeutralStatus"`
	Details                      []WebsocketAccountDetail `json:"details"`
}

// WebsocketAccountDetail holds one currency's balance in an account channel push
type WebsocketAccountDetail struct {
	Currency                           currency.Code `json:"ccy"`
	Equity                             types.Number  `json:"eq"`
	CashBalance                        types.Number  `json:"cashBal"`
	UpdateTime                         types.Time    `json:"uTime"`
	IsolatedMarginEquity               types.Number  `json:"isoEq"`
	AvailableEquity                    types.Number  `json:"availEq"`
	DiscountEquity                     types.Number  `json:"disEq"`
	FixedBalance                       types.Number  `json:"fixedBal"`
	AvailableBalance                   types.Number  `json:"availBal"`
	FrozenBalance                      types.Number  `json:"frozenBal"`
	OrderFrozen                        types.Number  `json:"ordFrozen"`
	Liabilities                        types.Number  `json:"liab"`
	UnrealisedProfitAndLoss            types.Number  `json:"upl"`
	UnrealisedProfitAndLossLiabilities types.Number  `json:"uplLiab"`
	CrossLiabilities                   types.Number  `json:"crossLiab"`
	IsolatedLiabilities                types.Number  `json:"isoLiab"`
	RewardBalance                      types.Number  `json:"rewardBal"`
	MarginRatio                        types.Number  `json:"mgnRatio"`
	InitialMarginRequirement           types.Number  `json:"imr"`
	MaintenanceMarginRequirement       types.Number  `json:"mmr"`
	Interest                           types.Number  `json:"interest"`
	ForcedRepaymentIndicator           types.Number  `json:"twap"`
	ForcedRepaymentType                string        `json:"frpType"`
	MaximumLoan                        types.Number  `json:"maxLoan"`
	EquityUSD                          types.Number  `json:"eqUsd"`
	BorrowFrozen                       types.Number  `json:"borrowFroz"`
	NotionalLeverage                   types.Number  `json:"notionalLever"`
	CoinUSDPrice                       types.Number  `json:"coinUsdPrice"`
	StrategyEquity                     types.Number  `json:"stgyEq"`
	IsolatedUnrealisedProfitAndLoss    types.Number  `json:"isoUpl"`
	SpotInUseAmount                    types.Number  `json:"spotInUseAmt"`
	ClientSpotInUseAmount              types.Number  `json:"clSpotInUseAmt"`
	MaximumSpotInUseAmount             types.Number  `json:"maxSpotInUseAmt"`
	SpotIsolatedBalance                types.Number  `json:"spotIsoBal"`
	SmartSyncEquity                    types.Number  `json:"smtSyncEq"`
	SpotCopyTradingEquity              types.Number  `json:"spotCopyTradingEq"`
	SpotBalance                        types.Number  `json:"spotBal"`
	OpenAveragePrice                   types.Number  `json:"openAvgPx"`
	AccumulatedAveragePrice            types.Number  `json:"accAvgPx"`
	SpotUnrealisedProfitAndLoss        types.Number  `json:"spotUpl"`
	SpotUnrealisedProfitAndLossRatio   types.Number  `json:"spotUplRatio"`
	TotalProfitAndLoss                 types.Number  `json:"totalPnl"`
	TotalProfitAndLossRatio            types.Number  `json:"totalPnlRatio"`
	CollateralRestriction              string        `json:"colRes"`
	CollateralBorrowAutoConversion     string        `json:"colBorrAutoConversion"`
	CollateralRestrict                 bool          `json:"collateralRestrict"`
	CollateralEnabled                  bool          `json:"collateralEnabled"`
	AutoLendStatus                     string        `json:"autoLendStatus"`
	AutoLendMatchedAmount              types.Number  `json:"autoLendMtAmt"`
}

// WebsocketPositionsPush holds a positions channel push
type WebsocketPositionsPush struct {
	Argument    SubscriptionArgument `json:"arg"`
	EventType   string               `json:"eventType"`
	CurrentPage uint64               `json:"curPage"`
	LastPage    bool                 `json:"lastPage"`
	Data        []WebsocketPosition  `json:"data"`
}

// WebsocketPosition holds a position a positions channel push carries
type WebsocketPosition struct {
	InstrumentType                        string                            `json:"instType"`
	MarginMode                            string                            `json:"mgnMode"`
	PositionID                            string                            `json:"posId"`
	PositionSide                          string                            `json:"posSide"`
	Position                              types.Number                      `json:"pos"`
	HedgedPosition                        types.Number                      `json:"hedgedPos"`
	BaseBalance                           types.Number                      `json:"baseBal"`
	QuoteBalance                          types.Number                      `json:"quoteBal"`
	BaseBorrowed                          types.Number                      `json:"baseBorrowed"`
	BaseInterest                          types.Number                      `json:"baseInterest"`
	QuoteBorrowed                         types.Number                      `json:"quoteBorrowed"`
	QuoteInterest                         types.Number                      `json:"quoteInterest"`
	PositionCurrency                      currency.Code                     `json:"posCcy"`
	AvailablePosition                     types.Number                      `json:"availPos"`
	AveragePrice                          types.Number                      `json:"avgPx"`
	UnrealisedProfitAndLoss               types.Number                      `json:"upl"`
	UnrealisedProfitAndLossRatio          types.Number                      `json:"uplRatio"`
	UnrealisedProfitAndLossLastPrice      types.Number                      `json:"uplLastPx"`
	UnrealisedProfitAndLossRatioLastPrice types.Number                      `json:"uplRatioLastPx"`
	InstrumentID                          currency.Pair                     `json:"instId"`
	Leverage                              types.Number                      `json:"lever"`
	LiquidationPrice                      types.Number                      `json:"liqPx"`
	MarkPrice                             types.Number                      `json:"markPx"`
	InitialMarginRequirement              types.Number                      `json:"imr"`
	Margin                                types.Number                      `json:"margin"`
	MarginRatio                           types.Number                      `json:"mgnRatio"`
	MaintenanceMarginRequirement          types.Number                      `json:"mmr"`
	Liabilities                           types.Number                      `json:"liab"`
	LiabilitiesCurrency                   currency.Code                     `json:"liabCcy"`
	Interest                              types.Number                      `json:"interest"`
	TradeID                               string                            `json:"tradeId"`
	NotionalUSD                           types.Number                      `json:"notionalUsd"`
	OptionValue                           types.Number                      `json:"optVal"`
	PendingCloseOrderLiabilityValue       types.Number                      `json:"pendingCloseOrdLiabVal"`
	AutoDeleveragingIndicator             string                            `json:"adl"`
	BusinessReferenceID                   string                            `json:"bizRefId"`
	BusinessReferenceType                 string                            `json:"bizRefType"`
	Currency                              currency.Code                     `json:"ccy"`
	LastPrice                             types.Number                      `json:"last"`
	IndexPrice                            types.Number                      `json:"idxPx"`
	USDPrice                              types.Number                      `json:"usdPx"`
	BreakEvenPrice                        types.Number                      `json:"bePx"`
	DeltaBlackScholes                     types.Number                      `json:"deltaBS"`
	DeltaPriceAdjusted                    types.Number                      `json:"deltaPA"`
	GammaBlackScholes                     types.Number                      `json:"gammaBS"`
	GammaPriceAdjusted                    types.Number                      `json:"gammaPA"`
	ThetaBlackScholes                     types.Number                      `json:"thetaBS"`
	ThetaPriceAdjusted                    types.Number                      `json:"thetaPA"`
	VegaBlackScholes                      types.Number                      `json:"vegaBS"`
	VegaPriceAdjusted                     types.Number                      `json:"vegaPA"`
	SpotInUseAmount                       types.Number                      `json:"spotInUseAmt"`
	SpotInUseCurrency                     currency.Code                     `json:"spotInUseCcy"`
	ClientSpotInUseAmount                 types.Number                      `json:"clSpotInUseAmt"`
	MaximumSpotInUseAmount                types.Number                      `json:"maxSpotInUseAmt"`
	RealisedProfitAndLoss                 types.Number                      `json:"realizedPnl"`
	ProfitAndLoss                         types.Number                      `json:"pnl"`
	Fee                                   types.Number                      `json:"fee"`
	FundingFee                            types.Number                      `json:"fundingFee"`
	LiquidationPenalty                    types.Number                      `json:"liqPenalty"`
	CloseOrderAlgo                        []WebsocketPositionCloseOrderAlgo `json:"closeOrderAlgo"`
	CreationTime                          types.Time                        `json:"cTime"`
	UpdateTime                            types.Time                        `json:"uTime"`
	PushTime                              types.Time                        `json:"pTime"`
	NonSettlementAveragePrice             types.Number                      `json:"nonSettleAvgPx"`
	SettledProfitAndLoss                  types.Number                      `json:"settledPnl"`
}

// WebsocketPositionCloseOrderAlgo holds a close position algo order attached to a position
type WebsocketPositionCloseOrderAlgo struct {
	AlgoID                     string       `json:"algoId"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	CloseFraction              types.Number `json:"closeFraction"`
}

// WebsocketBalanceAndPositionPush holds a balance_and_position channel push
type WebsocketBalanceAndPositionPush struct {
	Argument SubscriptionArgument          `json:"arg"`
	Data     []WebsocketBalanceAndPosition `json:"data"`
}

// WebsocketBalanceAndPosition holds the balances and positions an event changed
type WebsocketBalanceAndPosition struct {
	PushTime  types.Time                         `json:"pTime"`
	EventType string                             `json:"eventType"`
	Balances  []WebsocketBalance                 `json:"balData"`
	Positions []WebsocketBalancePosition         `json:"posData"`
	Trades    []WebsocketBalanceAndPositionTrade `json:"trades"`
}

// WebsocketBalance holds a currency's cash balance in a balance_and_position push
type WebsocketBalance struct {
	Currency    currency.Code `json:"ccy"`
	CashBalance types.Number  `json:"cashBal"`
	UpdateTime  types.Time    `json:"uTime"`
}

// WebsocketBalancePosition holds a position in a balance_and_position push
type WebsocketBalancePosition struct {
	PositionID                string        `json:"posId"`
	TradeID                   string        `json:"tradeId"`
	InstrumentID              currency.Pair `json:"instId"`
	InstrumentType            string        `json:"instType"`
	MarginMode                string        `json:"mgnMode"`
	AveragePrice              types.Number  `json:"avgPx"`
	Currency                  currency.Code `json:"ccy"`
	PositionSide              string        `json:"posSide"`
	Position                  types.Number  `json:"pos"`
	BaseBalance               types.Number  `json:"baseBal"`
	QuoteBalance              types.Number  `json:"quoteBal"`
	PositionCurrency          currency.Code `json:"posCcy"`
	NonSettlementAveragePrice types.Number  `json:"nonSettleAvgPx"`
	SettledProfitAndLoss      types.Number  `json:"settledPnl"`
	UpdateTime                types.Time    `json:"uTime"`
}

// WebsocketBalanceAndPositionTrade holds a trade that changed a position in a balance_and_position push
type WebsocketBalanceAndPositionTrade struct {
	InstrumentID currency.Pair `json:"instId"`
	TradeID      string        `json:"tradeId"`
}

// WebsocketPositionRiskWarningPush holds a liquidation-warning channel push
type WebsocketPositionRiskWarningPush struct {
	Argument SubscriptionArgument           `json:"arg"`
	Data     []WebsocketPositionRiskWarning `json:"data"`
}

// WebsocketPositionRiskWarning holds a position close to liquidation
type WebsocketPositionRiskWarning struct {
	InstrumentType   string        `json:"instType"`
	MarginMode       string        `json:"mgnMode"`
	PositionID       string        `json:"posId"`
	PositionSide     string        `json:"posSide"`
	Position         types.Number  `json:"pos"`
	PositionCurrency currency.Code `json:"posCcy"`
	InstrumentID     currency.Pair `json:"instId"`
	Leverage         types.Number  `json:"lever"`
	MarkPrice        types.Number  `json:"markPx"`
	MarginRatio      types.Number  `json:"mgnRatio"`
	Currency         currency.Code `json:"ccy"`
	CreationTime     types.Time    `json:"cTime"`
	UpdateTime       types.Time    `json:"uTime"`
	PushTime         types.Time    `json:"pTime"`
}

// WebsocketAccountGreeksPush holds an account-greeks channel push
type WebsocketAccountGreeksPush struct {
	Argument SubscriptionArgument     `json:"arg"`
	Data     []WebsocketAccountGreeks `json:"data"`
}

// WebsocketAccountGreeks holds the greeks of a settlement currency
type WebsocketAccountGreeks struct {
	DeltaBlackScholes  types.Number  `json:"deltaBS"`
	DeltaPriceAdjusted types.Number  `json:"deltaPA"`
	GammaBlackScholes  types.Number  `json:"gammaBS"`
	GammaPriceAdjusted types.Number  `json:"gammaPA"`
	ThetaBlackScholes  types.Number  `json:"thetaBS"`
	ThetaPriceAdjusted types.Number  `json:"thetaPA"`
	VegaBlackScholes   types.Number  `json:"vegaBS"`
	VegaPriceAdjusted  types.Number  `json:"vegaPA"`
	Currency           currency.Code `json:"ccy"`
	Timestamp          types.Time    `json:"ts"`
}

// WebsocketOrdersPush holds an orders channel push
type WebsocketOrdersPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketOrder     `json:"data"`
}

// WebsocketOrder holds an order an orders channel push carries
type WebsocketOrder struct {
	InstrumentType          string                       `json:"instType"`
	InstrumentID            currency.Pair                `json:"instId"`
	SizeType                string                       `json:"tgtCcy"`
	Currency                currency.Code                `json:"ccy"`
	OrderID                 string                       `json:"ordId"`
	ClientOrderID           string                       `json:"clOrdId"`
	Tag                     string                       `json:"tag"`
	Price                   types.Number                 `json:"px"`
	PriceUSD                types.Number                 `json:"pxUsd"`
	PriceVolatility         types.Number                 `json:"pxVol"`
	PriceType               string                       `json:"pxType"`
	Size                    types.Number                 `json:"sz"`
	NotionalUSD             types.Number                 `json:"notionalUsd"`
	OrderType               string                       `json:"ordType"`
	Side                    order.Side                   `json:"side"`
	PositionSide            string                       `json:"posSide"`
	TradeMode               string                       `json:"tdMode"`
	FillPrice               types.Number                 `json:"fillPx"`
	TradeID                 string                       `json:"tradeId"`
	FillSize                types.Number                 `json:"fillSz"`
	FillProfitAndLoss       types.Number                 `json:"fillPnl"`
	FillTime                types.Time                   `json:"fillTime"`
	FillFee                 types.Number                 `json:"fillFee"`
	FillFeeCurrency         currency.Code                `json:"fillFeeCcy"`
	FillPriceVolatility     types.Number                 `json:"fillPxVol"`
	FillPriceUSD            types.Number                 `json:"fillPxUsd"`
	FillMarkVolatility      types.Number                 `json:"fillMarkVol"`
	FillForwardPrice        types.Number                 `json:"fillFwdPx"`
	FillMarkPrice           types.Number                 `json:"fillMarkPx"`
	FillIndexPrice          types.Number                 `json:"fillIdxPx"`
	ExecutionType           string                       `json:"execType"`
	AccumulatedFillSize     types.Number                 `json:"accFillSz"`
	FillNotionalUSD         types.Number                 `json:"fillNotionalUsd"`
	AveragePrice            types.Number                 `json:"avgPx"`
	State                   string                       `json:"state"`
	Leverage                types.Number                 `json:"lever"`
	AttachAlgoClientOrderID string                       `json:"attachAlgoClOrdId"`
	TakeProfitTriggerPrice  types.Number                 `json:"tpTriggerPx"`
	TakeProfitTriggerType   string                       `json:"tpTriggerPxType"`
	TakeProfitOrderPrice    types.Number                 `json:"tpOrdPx"`
	StopLossTriggerPrice    types.Number                 `json:"slTriggerPx"`
	StopLossTriggerType     string                       `json:"slTriggerPxType"`
	StopLossOrderPrice      types.Number                 `json:"slOrdPx"`
	AttachedAlgoOrders      []WebsocketAttachedAlgoOrder `json:"attachAlgoOrds"`
	LinkedAlgoOrder         WebsocketLinkedAlgoOrder     `json:"linkedAlgoOrd"`
	SelfTradePreventionID   string                       `json:"stpId"`
	SelfTradePreventionMode string                       `json:"stpMode"`
	FeeCurrency             currency.Code                `json:"feeCcy"`
	Fee                     types.Number                 `json:"fee"`
	RebateCurrency          currency.Code                `json:"rebateCcy"`
	Rebate                  types.Number                 `json:"rebate"`
	ProfitAndLoss           types.Number                 `json:"pnl"`
	Source                  string                       `json:"source"`
	CancelSource            string                       `json:"cancelSource"`
	AmendSource             string                       `json:"amendSource"`
	Category                string                       `json:"category"`
	IsTakeProfitLimit       types.Boolean                `json:"isTpLimit"`
	UpdateTime              types.Time                   `json:"uTime"`
	CreationTime            types.Time                   `json:"cTime"`
	RequestID               string                       `json:"reqId"`
	AmendResult             string                       `json:"amendResult"`
	ReduceOnly              bool                         `json:"reduceOnly,string"`
	QuickMarginType         string                       `json:"quickMgnType"`
	AlgoClientOrderID       string                       `json:"algoClOrdId"`
	AlgoID                  string                       `json:"algoId"`
	LastPrice               types.Number                 `json:"lastPx"`
	Code                    string                       `json:"code"`
	Message                 string                       `json:"msg"`
	TradeQuoteCurrency      currency.Code                `json:"tradeQuoteCcy"`
	Outcome                 string                       `json:"outcome"`
	RiskBypassResult        string                       `json:"riskBypassResult"`
}

// WebsocketAttachedAlgoOrder holds a take profit, stop loss or trailing stop order attached to an order
type WebsocketAttachedAlgoOrder struct {
	AttachAlgoID            string       `json:"attachAlgoId"`
	AttachAlgoClientOrderID string       `json:"attachAlgoClOrdId"`
	TakeProfitOrderKind     string       `json:"tpOrdKind"`
	TakeProfitTriggerPrice  types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerRatio  types.Number `json:"tpTriggerRatio"`
	TakeProfitTriggerType   string       `json:"tpTriggerPxType"`
	TakeProfitOrderPrice    types.Number `json:"tpOrdPx"`
	StopLossTriggerPrice    types.Number `json:"slTriggerPx"`
	StopLossTriggerRatio    types.Number `json:"slTriggerRatio"`
	StopLossTriggerType     string       `json:"slTriggerPxType"`
	StopLossOrderPrice      types.Number `json:"slOrdPx"`
	Size                    types.Number `json:"sz"`
	AmendPriceOnTriggerType string       `json:"amendPxOnTriggerType"`
	CallbackRatio           types.Number `json:"callbackRatio"`
	CallbackSpread          types.Number `json:"callbackSpread"`
	ActivationPrice         types.Number `json:"activePx"`
}

// WebsocketLinkedAlgoOrder holds the stop loss order linked to a take profit limit order of a one-cancels-the-other order
type WebsocketLinkedAlgoOrder struct {
	AlgoID string `json:"algoId"`
}

// WebsocketFillsPush holds a fills channel push
type WebsocketFillsPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketFill      `json:"data"`
}

// WebsocketFill holds a fill, aggregated over trades of the same order and price
type WebsocketFill struct {
	InstrumentID  currency.Pair `json:"instId"`
	FillSize      types.Number  `json:"fillSz"`
	FillPrice     types.Number  `json:"fillPx"`
	Side          order.Side    `json:"side"`
	Timestamp     types.Time    `json:"ts"`
	OrderID       string        `json:"ordId"`
	ClientOrderID string        `json:"clOrdId"`
	TradeID       string        `json:"tradeId"`
	ExecutionType string        `json:"execType"`
	Count         uint64        `json:"count,string"`
}

// WebsocketAlgoOrdersPush holds an orders-algo channel push
type WebsocketAlgoOrdersPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketAlgoOrder `json:"data"`
}

// WebsocketAlgoOrder holds an algo order an orders-algo channel push carries
type WebsocketAlgoOrder struct {
	InstrumentType          string                         `json:"instType"`
	InstrumentID            currency.Pair                  `json:"instId"`
	Currency                currency.Code                  `json:"ccy"`
	OrderID                 string                         `json:"ordId"`
	OrderIDList             []string                       `json:"ordIdList"`
	SubAlgoIDList           []string                       `json:"subAlgoIdList"`
	AlgoID                  string                         `json:"algoId"`
	ClientOrderID           string                         `json:"clOrdId"`
	Size                    types.Number                   `json:"sz"`
	OrderType               string                         `json:"ordType"`
	Side                    order.Side                     `json:"side"`
	PositionSide            string                         `json:"posSide"`
	TradeMode               string                         `json:"tdMode"`
	SizeType                string                         `json:"tgtCcy"`
	Leverage                types.Number                   `json:"lever"`
	State                   string                         `json:"state"`
	TakeProfitTriggerPrice  types.Number                   `json:"tpTriggerPx"`
	TakeProfitTriggerType   string                         `json:"tpTriggerPxType"`
	TakeProfitOrderPrice    types.Number                   `json:"tpOrdPx"`
	StopLossTriggerPrice    types.Number                   `json:"slTriggerPx"`
	StopLossTriggerType     string                         `json:"slTriggerPxType"`
	StopLossOrderPrice      types.Number                   `json:"slOrdPx"`
	TriggerPrice            types.Number                   `json:"triggerPx"`
	TriggerPriceType        string                         `json:"triggerPxType"`
	OrderPrice              types.Number                   `json:"ordPx"`
	AdvanceOrderType        string                         `json:"advanceOrdType"`
	ChaseParameters         []WebsocketAlgoChaseParameters `json:"advChaseParams"`
	LastPrice               types.Number                   `json:"last"`
	ActualSize              types.Number                   `json:"actualSz"`
	ActualPrice             types.Number                   `json:"actualPx"`
	NotionalUSD             types.Number                   `json:"notionalUsd"`
	Tag                     string                         `json:"tag"`
	ActualSide              string                         `json:"actualSide"`
	TriggerTime             types.Time                     `json:"triggerTime"`
	ReduceOnly              string                         `json:"reduceOnly"` // "true", "false" or "": no bool decoding accepts all three
	CloseFraction           types.Number                   `json:"closeFraction"`
	QuickMarginType         string                         `json:"quickMgnType"`
	FailCode                string                         `json:"failCode"`
	AlgoClientOrderID       string                         `json:"algoClOrdId"`
	RequestID               string                         `json:"reqId"`
	AmendResult             string                         `json:"amendResult"`
	AmendPriceOnTriggerType string                         `json:"amendPxOnTriggerType"`
	AttachedAlgoOrders      []WebsocketAlgoAttachedOrder   `json:"attachAlgoOrds"`
	LinkedOrder             WebsocketAlgoLinkedOrder       `json:"linkedOrd"`
	CreationTime            types.Time                     `json:"cTime"`
	UpdateTime              types.Time                     `json:"uTime"`
	IsTradeBorrowMode       string                         `json:"isTradeBorrowMode"`
	ChaseType               string                         `json:"chaseType"`
	ChaseValue              types.Number                   `json:"chaseVal"`
	MaximumChaseType        string                         `json:"maxChaseType"`
	MaximumChaseValue       types.Number                   `json:"maxChaseVal"`
	TradeQuoteCurrency      currency.Code                  `json:"tradeQuoteCcy"`
}

// WebsocketAlgoChaseParameters holds the chase parameters of a trigger order whose sub order chases
type WebsocketAlgoChaseParameters struct {
	ChaseType         string       `json:"chaseType"`
	ChaseValue        types.Number `json:"chaseVal"`
	MaximumChaseType  string       `json:"maxChaseType"`
	MaximumChaseValue types.Number `json:"maxChaseVal"`
}

// WebsocketAlgoAttachedOrder holds a take profit, stop loss or trailing stop order attached to an algo order
type WebsocketAlgoAttachedOrder struct {
	AttachAlgoClientOrderID string       `json:"attachAlgoClOrdId"`
	TakeProfitTriggerPrice  types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerRatio  types.Number `json:"tpTriggerRatio"`
	TakeProfitTriggerType   string       `json:"tpTriggerPxType"`
	TakeProfitOrderPrice    types.Number `json:"tpOrdPx"`
	StopLossTriggerPrice    types.Number `json:"slTriggerPx"`
	StopLossTriggerRatio    types.Number `json:"slTriggerRatio"`
	StopLossTriggerType     string       `json:"slTriggerPxType"`
	StopLossOrderPrice      types.Number `json:"slOrdPx"`
	CallbackRatio           types.Number `json:"callbackRatio"`
	CallbackSpread          types.Number `json:"callbackSpread"`
	ActivationPrice         types.Number `json:"activePx"`
}

// WebsocketAlgoLinkedOrder holds the take profit order linked to a stop loss order of a one-cancels-the-other order
type WebsocketAlgoLinkedOrder struct {
	OrderID string `json:"ordId"`
}

// WebsocketAdvanceAlgoOrdersPush holds an algo-advance channel push
type WebsocketAdvanceAlgoOrdersPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketAdvanceAlgoOrder `json:"data"`
}

// WebsocketAdvanceAlgoOrder holds an iceberg, TWAP or trailing stop order an algo-advance channel push carries
type WebsocketAdvanceAlgoOrder struct {
	InstrumentType         string        `json:"instType"`
	InstrumentID           currency.Pair `json:"instId"`
	Currency               currency.Code `json:"ccy"`
	OrderID                string        `json:"ordId"`
	AlgoID                 string        `json:"algoId"`
	ClientOrderID          string        `json:"clOrdId"`
	Size                   types.Number  `json:"sz"`
	OrderType              string        `json:"ordType"`
	Side                   order.Side    `json:"side"`
	PositionSide           string        `json:"posSide"`
	TradeMode              string        `json:"tdMode"`
	SizeType               string        `json:"tgtCcy"`
	Leverage               types.Number  `json:"lever"`
	State                  string        `json:"state"`
	TakeProfitTriggerPrice types.Number  `json:"tpTriggerPx"`
	TakeProfitOrderPrice   types.Number  `json:"tpOrdPx"`
	StopLossTriggerPrice   types.Number  `json:"slTriggerPx"`
	StopLossOrderPrice     types.Number  `json:"slOrdPx"`
	TriggerPrice           types.Number  `json:"triggerPx"`
	OrderPrice             types.Number  `json:"ordPx"`
	ActualSize             types.Number  `json:"actualSz"`
	ActualPrice            types.Number  `json:"actualPx"`
	NotionalUSD            types.Number  `json:"notionalUsd"`
	Tag                    string        `json:"tag"`
	ActualSide             string        `json:"actualSide"`
	TriggerTime            types.Time    `json:"triggerTime"`
	CreationTime           types.Time    `json:"cTime"`
	PriceVariation         types.Number  `json:"pxVar"`
	PriceSpread            types.Number  `json:"pxSpread"`
	SizeLimit              types.Number  `json:"szLimit"`
	PriceLimit             types.Number  `json:"pxLimit"`
	TimeInterval           string        `json:"timeInterval"`
	Count                  types.Number  `json:"count"`
	CallbackRatio          types.Number  `json:"callbackRatio"`
	CallbackSpread         types.Number  `json:"callbackSpread"`
	ActivationPrice        types.Number  `json:"activePx"`
	MoveTriggerPrice       types.Number  `json:"moveTriggerPx"`
	FailCode               string        `json:"failCode"`
	AlgoClientOrderID      string        `json:"algoClOrdId"`
	ReduceOnly             string        `json:"reduceOnly"` // "true", "false" or "": no bool decoding accepts all three
	PushTime               types.Time    `json:"pTime"`
	IsTradeBorrowMode      bool          `json:"isTradeBorrowMode"`
	TradeQuoteCurrency     currency.Code `json:"tradeQuoteCcy"`
}

// WebsocketSpotGridAlgoOrdersPush holds a grid-orders-spot channel push
type WebsocketSpotGridAlgoOrdersPush struct {
	Argument SubscriptionArgument         `json:"arg"`
	Data     []WebsocketSpotGridAlgoOrder `json:"data"`
}

// WebsocketSpotGridAlgoOrder holds a spot grid algo order
type WebsocketSpotGridAlgoOrder struct {
	AlgoID                 string                           `json:"algoId"`
	AlgoClientOrderID      string                           `json:"algoClOrdId"`
	InstrumentType         string                           `json:"instType"`
	InstrumentID           currency.Pair                    `json:"instId"`
	CreationTime           types.Time                       `json:"cTime"`
	UpdateTime             types.Time                       `json:"uTime"`
	AlgoOrderType          string                           `json:"algoOrdType"`
	State                  string                           `json:"state"`
	RebateTransfers        []WebsocketGridRebateTransfer    `json:"rebateTrans"`
	TriggerParameters      []WebsocketGridTriggerParameters `json:"triggerParams"`
	MaximumPrice           types.Number                     `json:"maxPx"`
	MinimumPrice           types.Number                     `json:"minPx"`
	GridNumber             uint64                           `json:"gridNum,string"`
	RunType                string                           `json:"runType"`
	TakeProfitTriggerPrice types.Number                     `json:"tpTriggerPx"`
	StopLossTriggerPrice   types.Number                     `json:"slTriggerPx"`
	TradeNumber            uint64                           `json:"tradeNum,string"`
	ArbitrageNumber        uint64                           `json:"arbitrageNum,string"`
	SingleAmount           types.Number                     `json:"singleAmt"`
	PerMinimumProfitRate   types.Number                     `json:"perMinProfitRate"`
	PerMaximumProfitRate   types.Number                     `json:"perMaxProfitRate"`
	RunPrice               types.Number                     `json:"runPx"`
	TotalProfitAndLoss     types.Number                     `json:"totalPnl"`
	ProfitAndLossRatio     types.Number                     `json:"pnlRatio"`
	Investment             types.Number                     `json:"investment"`
	GridProfit             types.Number                     `json:"gridProfit"`
	FloatProfit            types.Number                     `json:"floatProfit"`
	TotalAnnualisedRate    types.Number                     `json:"totalAnnualizedRate"`
	AnnualisedRate         types.Number                     `json:"annualizedRate"`
	CancelType             string                           `json:"cancelType"`
	StopType               string                           `json:"stopType"`
	QuoteSize              types.Number                     `json:"quoteSz"`
	BaseSize               types.Number                     `json:"baseSz"`
	CurrentQuoteSize       types.Number                     `json:"curQuoteSz"`
	CurrentBaseSize        types.Number                     `json:"curBaseSz"`
	Profit                 types.Number                     `json:"profit"`
	StopResult             string                           `json:"stopResult"`
	ActiveOrderNumber      uint64                           `json:"activeOrdNum,string"`
	Tag                    string                           `json:"tag"`
	ProfitSharingRatio     types.Number                     `json:"profitSharingRatio"`
	CopyType               string                           `json:"copyType"`
	PushTime               types.Time                       `json:"pTime"`
	TradeQuoteCurrency     currency.Code                    `json:"tradeQuoteCcy"`
}

// WebsocketGridRebateTransfer holds a rebate transfer of a grid algo order
type WebsocketGridRebateTransfer struct {
	Rebate         types.Number  `json:"rebate"`
	RebateCurrency currency.Code `json:"rebateCcy"`
}

// WebsocketGridTriggerParameters holds the trigger parameters of a grid algo order
type WebsocketGridTriggerParameters struct {
	TriggerAction    string       `json:"triggerAction"`
	TriggerStrategy  string       `json:"triggerStrategy"`
	DelaySeconds     uint64       `json:"delaySeconds,string"`
	TriggerTime      types.Time   `json:"triggerTime"`
	TriggerType      string       `json:"triggerType"`
	Timeframe        string       `json:"timeframe"`
	Threshold        types.Number `json:"thold"`
	TriggerCondition string       `json:"triggerCond"`
	TimePeriod       types.Number `json:"timePeriod"`
	TriggerPrice     types.Number `json:"triggerPx"`
	StopType         string       `json:"stopType"`
}

// WebsocketContractGridAlgoOrdersPush holds a grid-orders-contract channel push
type WebsocketContractGridAlgoOrdersPush struct {
	Argument SubscriptionArgument             `json:"arg"`
	Data     []WebsocketContractGridAlgoOrder `json:"data"`
}

// WebsocketContractGridAlgoOrder holds a contract grid algo order
type WebsocketContractGridAlgoOrder struct {
	AlgoID                 string                           `json:"algoId"`
	AlgoClientOrderID      string                           `json:"algoClOrdId"`
	InstrumentType         string                           `json:"instType"`
	InstrumentID           currency.Pair                    `json:"instId"`
	CreationTime           types.Time                       `json:"cTime"`
	UpdateTime             types.Time                       `json:"uTime"`
	AlgoOrderType          string                           `json:"algoOrdType"`
	State                  string                           `json:"state"`
	RebateTransfers        []WebsocketGridRebateTransfer    `json:"rebateTrans"`
	TriggerParameters      []WebsocketGridTriggerParameters `json:"triggerParams"`
	MaximumPrice           types.Number                     `json:"maxPx"`
	MinimumPrice           types.Number                     `json:"minPx"`
	GridNumber             uint64                           `json:"gridNum,string"`
	RunType                string                           `json:"runType"`
	TakeProfitTriggerPrice types.Number                     `json:"tpTriggerPx"`
	StopLossTriggerPrice   types.Number                     `json:"slTriggerPx"`
	TradeNumber            uint64                           `json:"tradeNum,string"`
	ArbitrageNumber        uint64                           `json:"arbitrageNum,string"`
	SingleAmount           types.Number                     `json:"singleAmt"`
	PerMinimumProfitRate   types.Number                     `json:"perMinProfitRate"`
	PerMaximumProfitRate   types.Number                     `json:"perMaxProfitRate"`
	RunPrice               types.Number                     `json:"runPx"`
	TotalProfitAndLoss     types.Number                     `json:"totalPnl"`
	ProfitAndLossRatio     types.Number                     `json:"pnlRatio"`
	Investment             types.Number                     `json:"investment"`
	GridProfit             types.Number                     `json:"gridProfit"`
	FloatProfit            types.Number                     `json:"floatProfit"`
	TotalAnnualisedRate    types.Number                     `json:"totalAnnualizedRate"`
	AnnualisedRate         types.Number                     `json:"annualizedRate"`
	CancelType             string                           `json:"cancelType"`
	StopType               string                           `json:"stopType"`
	Direction              string                           `json:"direction"`
	BasePosition           bool                             `json:"basePos"`
	Size                   types.Number                     `json:"sz"`
	Leverage               types.Number                     `json:"lever"`
	ActualLeverage         types.Number                     `json:"actualLever"`
	LiquidationPrice       types.Number                     `json:"liqPx"`
	OrderFrozen            types.Number                     `json:"ordFrozen"`
	AvailableEquity        types.Number                     `json:"availEq"`
	Equity                 types.Number                     `json:"eq"`
	ActiveOrderNumber      uint64                           `json:"activeOrdNum,string"`
	Tag                    string                           `json:"tag"`
	ProfitSharingRatio     types.Number                     `json:"profitSharingRatio"`
	CopyType               string                           `json:"copyType"`
	TakeProfitRatio        types.Number                     `json:"tpRatio"`
	StopLossRatio          types.Number                     `json:"slRatio"`
	Fee                    types.Number                     `json:"fee"`
	FundingFee             types.Number                     `json:"fundingFee"`
	PushTime               types.Time                       `json:"pTime"`
}

// WebsocketGridPositionsPush holds a grid-positions channel push
type WebsocketGridPositionsPush struct {
	Argument SubscriptionArgument    `json:"arg"`
	Data     []WebsocketGridPosition `json:"data"`
}

// WebsocketGridPosition holds the position of a contract grid algo order
type WebsocketGridPosition struct {
	AlgoID                       string        `json:"algoId"`
	AlgoClientOrderID            string        `json:"algoClOrdId"`
	InstrumentType               string        `json:"instType"`
	InstrumentID                 currency.Pair `json:"instId"`
	CreationTime                 types.Time    `json:"cTime"`
	UpdateTime                   types.Time    `json:"uTime"`
	AveragePrice                 types.Number  `json:"avgPx"`
	Currency                     currency.Code `json:"ccy"`
	Leverage                     types.Number  `json:"lever"`
	LiquidationPrice             types.Number  `json:"liqPx"`
	PositionSide                 string        `json:"posSide"`
	Position                     types.Number  `json:"pos"`
	MarginMode                   string        `json:"mgnMode"`
	MarginRatio                  types.Number  `json:"mgnRatio"`
	InitialMarginRequirement     types.Number  `json:"imr"`
	MaintenanceMarginRequirement types.Number  `json:"mmr"`
	UnrealisedProfitAndLoss      types.Number  `json:"upl"`
	UnrealisedProfitAndLossRatio types.Number  `json:"uplRatio"`
	LastPrice                    types.Number  `json:"last"`
	NotionalUSD                  types.Number  `json:"notionalUsd"`
	AutoDeleveragingIndicator    string        `json:"adl"`
	MarkPrice                    types.Number  `json:"markPx"`
	PushTime                     types.Time    `json:"pTime"`
}

// WebsocketGridSubOrdersPush holds a grid-sub-orders channel push
type WebsocketGridSubOrdersPush struct {
	Argument SubscriptionArgument    `json:"arg"`
	Data     []WebsocketGridSubOrder `json:"data"`
}

// WebsocketGridSubOrder holds a sub order of a grid algo order
type WebsocketGridSubOrder struct {
	AlgoID              string        `json:"algoId"`
	AlgoClientOrderID   string        `json:"algoClOrdId"`
	InstrumentType      string        `json:"instType"`
	InstrumentID        currency.Pair `json:"instId"`
	AlgoOrderType       string        `json:"algoOrdType"`
	GroupID             string        `json:"groupId"`
	OrderID             string        `json:"ordId"`
	CreationTime        types.Time    `json:"cTime"`
	UpdateTime          types.Time    `json:"uTime"`
	TradeMode           string        `json:"tdMode"`
	Tag                 string        `json:"tag"`
	OrderType           string        `json:"ordType"`
	Size                types.Number  `json:"sz"`
	State               string        `json:"state"`
	Side                order.Side    `json:"side"`
	Price               types.Number  `json:"px"`
	Fee                 types.Number  `json:"fee"`
	FeeCurrency         currency.Code `json:"feeCcy"`
	Rebate              types.Number  `json:"rebate"`
	RebateCurrency      currency.Code `json:"rebateCcy"`
	AveragePrice        types.Number  `json:"avgPx"`
	AccumulatedFillSize types.Number  `json:"accFillSz"`
	PositionSide        string        `json:"posSide"`
	ProfitAndLoss       types.Number  `json:"pnl"`
	ContractValue       types.Number  `json:"ctVal"`
	Leverage            types.Number  `json:"lever"`
	PushTime            types.Time    `json:"pTime"`
}

// WebsocketRecurringBuyOrdersPush holds an algo-recurring-buy channel push
type WebsocketRecurringBuyOrdersPush struct {
	Argument SubscriptionArgument         `json:"arg"`
	Data     []WebsocketRecurringBuyOrder `json:"data"`
}

// WebsocketRecurringBuyOrder holds a recurring buy algo order
type WebsocketRecurringBuyOrder struct {
	AlgoID               string                      `json:"algoId"`
	AlgoClientOrderID    string                      `json:"algoClOrdId"`
	InstrumentType       string                      `json:"instType"`
	CreationTime         types.Time                  `json:"cTime"`
	UpdateTime           types.Time                  `json:"uTime"`
	AlgoOrderType        string                      `json:"algoOrdType"`
	State                string                      `json:"state"`
	StrategyName         string                      `json:"stgyName"`
	RecurringList        []WebsocketRecurringBuyItem `json:"recurringList"`
	Period               string                      `json:"period"`
	RecurringDay         string                      `json:"recurringDay"`
	RecurringHour        string                      `json:"recurringHour"`
	RecurringTime        string                      `json:"recurringTime"` // Hour of the day, 0 to 23
	TimeZone             string                      `json:"timeZone"`
	Amount               types.Number                `json:"amt"`
	InvestmentAmount     types.Number                `json:"investmentAmt"`
	InvestmentCurrency   currency.Code               `json:"investmentCcy"`
	NextInvestTime       types.Time                  `json:"nextInvestTime"`
	TotalProfitAndLoss   types.Number                `json:"totalPnl"`
	TotalAnnualisedRate  types.Number                `json:"totalAnnRate"`
	ProfitAndLossRatio   types.Number                `json:"pnlRatio"`
	MarketCapitalisation types.Number                `json:"mktCap"`
	Cycles               uint64                      `json:"cycles,string"`
	Tag                  string                      `json:"tag"`
	PushTime             types.Time                  `json:"pTime"`
	TradeQuoteCurrency   currency.Code               `json:"tradeQuoteCcy"`
	RecurringTimeType    string                      `json:"recurringTimeType"`
	RecurringTimeMinutes string                      `json:"recurringTimeMinutes"`
	Source               []string                    `json:"source"`
}

// WebsocketRecurringBuyItem holds one currency of a recurring buy algo order
type WebsocketRecurringBuyItem struct {
	Currency     currency.Code `json:"ccy"`
	Ratio        types.Number  `json:"ratio"`
	MinimumPrice types.Number  `json:"minPx"`
	MaximumPrice types.Number  `json:"maxPx"`
	TotalAmount  types.Number  `json:"totalAmt"`
	Profit       types.Number  `json:"profit"`
	AveragePrice types.Number  `json:"avgPx"`
	Price        types.Number  `json:"px"`
}

// WebsocketLeadTradingNotificationPush holds a copytrading-lead-notification channel push
type WebsocketLeadTradingNotificationPush struct {
	Argument SubscriptionArgument               `json:"arg"`
	Data     []WebsocketLeadTradingNotification `json:"data"`
}

// WebsocketLeadTradingNotification holds a lead trading failure notification
type WebsocketLeadTradingNotification struct {
	InstrumentType      string        `json:"instType"`
	InfoType            string        `json:"infoType"`
	SubPositionID       string        `json:"subPosId"`
	UniqueCode          string        `json:"uniqueCode"`
	InstrumentID        currency.Pair `json:"instId"`
	Side                string        `json:"side"` // Empty when the notification concerns no order
	PositionSide        string        `json:"posSide"`
	MaxLeadTraderNumber uint64        `json:"maxLeadTraderNum,string"`
	MinimumLeadEquity   types.Number  `json:"minLeadEq"`
}

// WebsocketTickersPush holds a tickers channel push
type WebsocketTickersPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []TickerResponse     `json:"data"`
}

// WebsocketCandlesticksPush holds a candlestick channel push
type WebsocketCandlesticksPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketCandlestick `json:"data"`
}

// WebsocketCandlestick holds a candlestick of an instrument
type WebsocketCandlestick struct {
	Timestamp           types.Time
	OpenPrice           types.Number
	HighestPrice        types.Number
	LowestPrice         types.Number
	ClosePrice          types.Number
	Volume              types.Number // Base currency on spot and margin, contracts on derivatives
	VolumeCurrency      types.Number // Quote currency on spot and margin, base currency on derivatives
	VolumeQuoteCurrency types.Number
	Confirm             types.Boolean
}

// UnmarshalJSON decodes a candlestick from its array form
func (c *WebsocketCandlestick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[9]any{&c.Timestamp, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &c.Volume, &c.VolumeCurrency, &c.VolumeQuoteCurrency, &c.Confirm})
}

// WebsocketTradesPush holds a trades or trades-all channel push
type WebsocketTradesPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketTrade     `json:"data"`
}

// WebsocketTrade holds a public trade; the trades channel aggregates trades
// of one taker order at one price and sets Count and SequenceID, and the
// trades-all channel pushes each trade alone
type WebsocketTrade struct {
	InstrumentID currency.Pair `json:"instId"`
	TradeID      string        `json:"tradeId"`
	Price        types.Number  `json:"px"`
	Size         types.Number  `json:"sz"`
	Side         order.Side    `json:"side"`
	Timestamp    types.Time    `json:"ts"`
	Count        uint64        `json:"count,string"`
	Source       string        `json:"source"`
	SequenceID   uint64        `json:"seqId"`
}

// WebsocketOrderBookPush holds an order book channel push
type WebsocketOrderBookPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Action   string               `json:"action"`
	Data     []WebsocketOrderBook `json:"data"`
}

// WebsocketOrderBook holds an order book snapshot or update; each level is
// price, size, a deprecated zero (the size of RPI orders on books-rpi) and the
// order count
type WebsocketOrderBook struct {
	Asks               [][4]types.Number `json:"asks"`
	Bids               [][4]types.Number `json:"bids"`
	InstrumentID       currency.Pair     `json:"instId"`
	Timestamp          types.Time        `json:"ts"`
	Checksum           int64             `json:"checksum"`  // Signed CRC32
	PreviousSequenceID int64             `json:"prevSeqId"` // -1 on a snapshot
	SequenceID         int64             `json:"seqId"`     // Signed to compare with prevSeqId, which is -1 on a snapshot
}

// WebsocketOptionTradesPush holds an option-trades channel push
type WebsocketOptionTradesPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketOptionTrade `json:"data"`
}

// WebsocketOptionTrade holds a public option trade
type WebsocketOptionTrade struct {
	InstrumentID     currency.Pair `json:"instId"`
	InstrumentFamily string        `json:"instFamily"`
	TradeID          string        `json:"tradeId"`
	Price            types.Number  `json:"px"`
	Size             types.Number  `json:"sz"`
	Side             order.Side    `json:"side"`
	OptionType       string        `json:"optType"`
	FillVolatility   types.Number  `json:"fillVol"`
	ForwardPrice     types.Number  `json:"fwdPx"`
	IndexPrice       types.Number  `json:"idxPx"`
	MarkPrice        types.Number  `json:"markPx"`
	Timestamp        types.Time    `json:"ts"`
}

// WebsocketCallAuctionDetailsPush holds a call-auction-details channel push
type WebsocketCallAuctionDetailsPush struct {
	Argument SubscriptionArgument          `json:"arg"`
	Data     []WebsocketCallAuctionDetails `json:"data"`
}

// WebsocketCallAuctionDetails holds the state of an instrument's call auction
type WebsocketCallAuctionDetails struct {
	InstrumentID     currency.Pair `json:"instId"`
	EquilibriumPrice types.Number  `json:"eqPx"`
	MatchedSize      types.Number  `json:"matchedSz"`
	UnmatchedSize    types.Number  `json:"unmatchedSz"`
	AuctionEndTime   types.Time    `json:"auctionEndTime"`
	State            string        `json:"state"`
	Timestamp        types.Time    `json:"ts"`
}

// WebsocketRequestForQuotesPush holds an rfqs channel push
type WebsocketRequestForQuotesPush struct {
	Argument SubscriptionArgument       `json:"arg"`
	Data     []WebsocketRequestForQuote `json:"data"`
}

// WebsocketRequestForQuote holds a request for quote
type WebsocketRequestForQuote struct {
	CreationTime            types.Time                   `json:"cTime"`
	UpdateTime              types.Time                   `json:"uTime"`
	State                   string                       `json:"state"`
	Counterparties          []string                     `json:"counterparties"`
	ValidUntil              types.Time                   `json:"validUntil"`
	ClientRequestForQuoteID string                       `json:"clRfqId"`
	Tag                     string                       `json:"tag"`
	FlowType                string                       `json:"flowType"`
	TraderCode              string                       `json:"traderCode"`
	RequestForQuoteID       string                       `json:"rfqId"`
	AllowPartialExecution   bool                         `json:"allowPartialExecution"`
	Legs                    []WebsocketBlockTradeLeg     `json:"legs"`
	GroupID                 string                       `json:"groupId"`
	AccountAllocations      []WebsocketAccountAllocation `json:"acctAlloc"`
}

// WebsocketBlockTradeLeg holds a leg of a request for quote or a quote
type WebsocketBlockTradeLeg struct {
	InstrumentID       currency.Pair `json:"instId"`
	TradeMode          string        `json:"tdMode"`
	Currency           currency.Code `json:"ccy"`
	Size               types.Number  `json:"sz"`
	Price              types.Number  `json:"px"`
	Side               order.Side    `json:"side"`
	PositionSide       string        `json:"posSide"`
	SizeType           string        `json:"tgtCcy"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// WebsocketAccountAllocation holds the legs a request for quote allocates to an account
type WebsocketAccountAllocation struct {
	Account string                          `json:"acct"`
	Legs    []WebsocketAccountAllocationLeg `json:"legs"`
}

// WebsocketAccountAllocationLeg holds a leg a request for quote allocates to an account
type WebsocketAccountAllocationLeg struct {
	InstrumentID currency.Pair `json:"instId"`
	Size         types.Number  `json:"sz"`
	TradeMode    string        `json:"tdMode"`
	Currency     currency.Code `json:"ccy"`
	PositionSide string        `json:"posSide"`
}

// WebsocketQuotesPush holds a quotes channel push
type WebsocketQuotesPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketQuote     `json:"data"`
}

// WebsocketQuote holds a quote on a request for quote
type WebsocketQuote struct {
	CreationTime            types.Time               `json:"cTime"`
	UpdateTime              types.Time               `json:"uTime"`
	State                   string                   `json:"state"`
	Reason                  string                   `json:"reason"`
	ValidUntil              types.Time               `json:"validUntil"`
	RequestForQuoteID       string                   `json:"rfqId"`
	ClientRequestForQuoteID string                   `json:"clRfqId"`
	QuoteID                 string                   `json:"quoteId"`
	ClientQuoteID           string                   `json:"clQuoteId"`
	Tag                     string                   `json:"tag"`
	TraderCode              string                   `json:"traderCode"`
	QuoteSide               order.Side               `json:"quoteSide"`
	Legs                    []WebsocketBlockTradeLeg `json:"legs"`
}

// WebsocketStructureBlockTradesPush holds a struc-block-trades channel push
type WebsocketStructureBlockTradesPush struct {
	Argument SubscriptionArgument           `json:"arg"`
	Data     []WebsocketStructureBlockTrade `json:"data"`
}

// WebsocketStructureBlockTrade holds an executed block trade the account took part in
type WebsocketStructureBlockTrade struct {
	CreationTime            types.Time                        `json:"cTime"`
	RequestForQuoteID       string                            `json:"rfqId"`
	ClientRequestForQuoteID string                            `json:"clRfqId"`
	QuoteID                 string                            `json:"quoteId"`
	ClientQuoteID           string                            `json:"clQuoteId"`
	BlockTradeID            string                            `json:"blockTdId"`
	Tag                     string                            `json:"tag"`
	TakerTraderCode         string                            `json:"tTraderCode"`
	MakerTraderCode         string                            `json:"mTraderCode"`
	IsSuccessful            bool                              `json:"isSuccessful"`
	ErrorCode               string                            `json:"errorCode"`
	Legs                    []WebsocketStructureBlockTradeLeg `json:"legs"`
	AccountAllocations      []WebsocketBlockTradeAllocation   `json:"acctAlloc"`
}

// WebsocketStructureBlockTradeLeg holds an executed leg of a block trade the account took part in
type WebsocketStructureBlockTradeLeg struct {
	InstrumentID currency.Pair `json:"instId"`
	Price        types.Number  `json:"px"`
	Size         types.Number  `json:"sz"`
	Side         order.Side    `json:"side"`
	SizeType     string        `json:"tgtCcy"`
	Fee          types.Number  `json:"fee"`
	FeeCurrency  currency.Code `json:"feeCcy"`
	TradeID      string        `json:"tradeId"`
}

// WebsocketBlockTradeAllocation holds the legs of a block trade allocated to an account
type WebsocketBlockTradeAllocation struct {
	BlockTradeID string                             `json:"blockTdId"`
	ErrorCode    string                             `json:"errorCode"`
	Account      string                             `json:"acct"`
	Legs         []WebsocketBlockTradeAllocationLeg `json:"legs"`
}

// WebsocketBlockTradeAllocationLeg holds a leg of a block trade allocated to an account
type WebsocketBlockTradeAllocationLeg struct {
	InstrumentID currency.Pair `json:"instId"`
	Size         types.Number  `json:"sz"`
	TradeID      string        `json:"tradeId"`
	Fee          types.Number  `json:"fee"`
	FeeCurrency  currency.Code `json:"feeCcy"`
}

// WebsocketPublicStructureBlockTradesPush holds a public-struc-block-trades channel push
type WebsocketPublicStructureBlockTradesPush struct {
	Argument SubscriptionArgument                 `json:"arg"`
	Data     []WebsocketPublicStructureBlockTrade `json:"data"`
}

// WebsocketPublicStructureBlockTrade holds an executed block trade with all its legs
type WebsocketPublicStructureBlockTrade struct {
	CreationTime types.Time                     `json:"cTime"`
	BlockTradeID string                         `json:"blockTdId"`
	GroupID      string                         `json:"groupId"`
	Strategy     string                         `json:"strategy"`
	Inverse      bool                           `json:"inverse"`
	Legs         []WebsocketPublicBlockTradeLeg `json:"legs"`
}

// WebsocketPublicBlockTradeLeg holds an executed leg of a public block trade
type WebsocketPublicBlockTradeLeg struct {
	InstrumentID currency.Pair `json:"instId"`
	Price        types.Number  `json:"px"`
	Size         types.Number  `json:"sz"`
	Side         order.Side    `json:"side"`
	TradeID      string        `json:"tradeId"`
}

// WebsocketPublicBlockTradesPush holds a public-block-trades channel push
type WebsocketPublicBlockTradesPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketPublicBlockTrade `json:"data"`
}

// WebsocketPublicBlockTrade holds an executed block trade leg of an instrument
type WebsocketPublicBlockTrade struct {
	InstrumentID   currency.Pair `json:"instId"`
	TradeID        string        `json:"tradeId"`
	Price          types.Number  `json:"px"`
	Size           types.Number  `json:"sz"`
	Side           order.Side    `json:"side"`
	FillVolatility types.Number  `json:"fillVol"`
	ForwardPrice   types.Number  `json:"fwdPx"`
	IndexPrice     types.Number  `json:"idxPx"`
	MarkPrice      types.Number  `json:"markPx"`
	GroupID        string        `json:"groupId"`
	Timestamp      types.Time    `json:"ts"`
}

// WebsocketBlockTickersPush holds a block-tickers channel push
type WebsocketBlockTickersPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketBlockTicker `json:"data"`
}

// WebsocketBlockTicker holds the block trading volume of an instrument over the last 24 hours
type WebsocketBlockTicker struct {
	InstrumentID         currency.Pair `json:"instId"`
	InstrumentType       string        `json:"instType"`
	VolumeCurrency24Hour types.Number  `json:"volCcy24h"`
	Volume24Hour         types.Number  `json:"vol24h"`
	Timestamp            types.Time    `json:"ts"`
}

// WebsocketSpreadOrdersPush holds a sprd-orders channel push
type WebsocketSpreadOrdersPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketSpreadOrder `json:"data"`
}

// WebsocketSpreadOrder holds a spread order
type WebsocketSpreadOrder struct {
	SpreadID            currency.Pair  `json:"sprdId"`
	OrderID             string         `json:"ordId"`
	ClientOrderID       string         `json:"clOrdId"`
	Tag                 string         `json:"tag"`
	Price               types.Number   `json:"px"`
	Size                types.Number   `json:"sz"`
	OrderType           string         `json:"ordType"`
	Side                order.Side     `json:"side"`
	FillSize            types.Number   `json:"fillSz"`
	FillPrice           types.Number   `json:"fillPx"`
	TradeID             string         `json:"tradeId"`
	AccumulatedFillSize types.Number   `json:"accFillSz"`
	PendingFillSize     OptionalNumber `json:"pendingFillSz"`
	PendingSettleSize   types.Number   `json:"pendingSettleSz"`
	CanceledSize        types.Number   `json:"canceledSz"`
	AveragePrice        types.Number   `json:"avgPx"`
	State               string         `json:"state"`
	CancelSource        string         `json:"cancelSource"`
	UpdateTime          types.Time     `json:"uTime"`
	CreationTime        types.Time     `json:"cTime"`
	Code                string         `json:"code"`
	Message             string         `json:"msg"`
	RequestID           string         `json:"reqId"`
	AmendResult         string         `json:"amendResult"`
}

// WebsocketSpreadTradesPush holds a sprd-trades channel push
type WebsocketSpreadTradesPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketSpreadTrade `json:"data"`
}

// WebsocketSpreadTrade holds a fill of a spread order
type WebsocketSpreadTrade struct {
	SpreadID      currency.Pair             `json:"sprdId"`
	TradeID       string                    `json:"tradeId"`
	OrderID       string                    `json:"ordId"`
	ClientOrderID string                    `json:"clOrdId"`
	Tag           string                    `json:"tag"`
	FillPrice     types.Number              `json:"fillPx"`
	FillSize      types.Number              `json:"fillSz"`
	Side          order.Side                `json:"side"`
	State         string                    `json:"state"`
	ExecutionType string                    `json:"execType"`
	Timestamp     types.Time                `json:"ts"`
	Legs          []WebsocketSpreadTradeLeg `json:"legs"`
	Code          string                    `json:"code"`
	Message       string                    `json:"msg"`
}

// WebsocketSpreadTradeLeg holds the fill of one leg of a spread trade
type WebsocketSpreadTradeLeg struct {
	InstrumentID      currency.Pair `json:"instId"`
	Price             types.Number  `json:"px"`
	Size              types.Number  `json:"sz"`
	SizeInContracts   types.Number  `json:"szCont"`
	Side              order.Side    `json:"side"`
	FillProfitAndLoss types.Number  `json:"fillPnl"`
	Fee               types.Number  `json:"fee"`
	FeeCurrency       currency.Code `json:"feeCcy"`
	TradeID           string        `json:"tradeId"`
}

// WebsocketSpreadOrderBookPush holds a spread order book channel push
type WebsocketSpreadOrderBookPush struct {
	Argument SubscriptionArgument       `json:"arg"`
	Action   string                     `json:"action"`
	Data     []WebsocketSpreadOrderBook `json:"data"`
}

// WebsocketSpreadOrderBook holds a spread order book snapshot or update; each
// level is price, size and order count, as sent, for the checksum
type WebsocketSpreadOrderBook struct {
	Asks               [][3]types.PreciseNumber `json:"asks"`
	Bids               [][3]types.PreciseNumber `json:"bids"`
	Timestamp          types.Time               `json:"ts"`
	Checksum           int64                    `json:"checksum"`  // Signed CRC32
	PreviousSequenceID int64                    `json:"prevSeqId"` // -1 on a snapshot
	SequenceID         int64                    `json:"seqId"`     // Signed to compare with prevSeqId, which is -1 on a snapshot
}

// WebsocketSpreadPublicTradesPush holds a sprd-public-trades channel push
type WebsocketSpreadPublicTradesPush struct {
	Argument SubscriptionArgument         `json:"arg"`
	Data     []WebsocketSpreadPublicTrade `json:"data"`
}

// WebsocketSpreadPublicTrade holds a public spread trade
type WebsocketSpreadPublicTrade struct {
	SpreadID  currency.Pair `json:"sprdId"`
	TradeID   string        `json:"tradeId"`
	Price     types.Number  `json:"px"`
	Size      types.Number  `json:"sz"`
	Side      order.Side    `json:"side"`
	Timestamp types.Time    `json:"ts"`
}

// WebsocketSpreadTickersPush holds a sprd-tickers channel push
type WebsocketSpreadTickersPush struct {
	Argument SubscriptionArgument    `json:"arg"`
	Data     []WebsocketSpreadTicker `json:"data"`
}

// WebsocketSpreadTicker holds a spread ticker
type WebsocketSpreadTicker struct {
	SpreadID           currency.Pair `json:"sprdId"`
	LastPrice          types.Number  `json:"last"`
	LastSize           types.Number  `json:"lastSz"`
	BestAskPrice       types.Number  `json:"askPx"`
	BestAskSize        types.Number  `json:"askSz"`
	BestBidPrice       types.Number  `json:"bidPx"`
	BestBidSize        types.Number  `json:"bidSz"`
	OpenPrice24Hour    types.Number  `json:"open24h"`
	HighestPrice24Hour types.Number  `json:"high24h"`
	LowestPrice24Hour  types.Number  `json:"low24h"`
	Volume24Hour       types.Number  `json:"vol24h"` // Base currency, or USD on an inverse spread
	Timestamp          types.Time    `json:"ts"`
}

// WebsocketSpreadCandlesticksPush holds a spread candlestick channel push
type WebsocketSpreadCandlesticksPush struct {
	Argument SubscriptionArgument         `json:"arg"`
	Data     []WebsocketSpreadCandlestick `json:"data"`
}

// WebsocketSpreadCandlestick holds a candlestick of a spread
type WebsocketSpreadCandlestick struct {
	Timestamp    types.Time
	OpenPrice    types.Number
	HighestPrice types.Number
	LowestPrice  types.Number
	ClosePrice   types.Number
	Volume       types.Number
	Confirm      types.Boolean
}

// UnmarshalJSON decodes a spread candlestick from its array form
func (c *WebsocketSpreadCandlestick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&c.Timestamp, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &c.Volume, &c.Confirm})
}

// WebsocketInstrumentsPush holds an instruments channel push
type WebsocketInstrumentsPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []InstrumentResponse `json:"data"`
}

// WebsocketEventContractMarketsPush holds an event-contract-markets channel push
type WebsocketEventContractMarketsPush struct {
	Argument SubscriptionArgument           `json:"arg"`
	Data     []WebsocketEventContractMarket `json:"data"`
}

// WebsocketEventContractMarket holds the state of an event contract market
type WebsocketEventContractMarket struct {
	SeriesID     string        `json:"seriesId"`
	EventID      string        `json:"eventId"`
	InstrumentID currency.Pair `json:"instId"`
	ListTime     types.Time    `json:"listTime"`
	FixTime      types.Time    `json:"fixTime"`
	ExpiryTime   types.Time    `json:"expTime"`
	State        string        `json:"state"`
	Outcome      string        `json:"outcome"`
	FloorStrike  types.Number  `json:"floorStrike"`
	CapStrike    string        `json:"capStrike"` // A number, or "INF" when there is no cap
	SettleValue  types.Number  `json:"settleValue"`
	Disputed     bool          `json:"disputed"`
	HitDirection string        `json:"hitDir"`
}

// WebsocketOpenInterestPush holds an open-interest channel push
type WebsocketOpenInterestPush struct {
	Argument SubscriptionArgument    `json:"arg"`
	Data     []WebsocketOpenInterest `json:"data"`
}

// WebsocketOpenInterest holds the open interest of an instrument
type WebsocketOpenInterest struct {
	InstrumentType       string        `json:"instType"`
	InstrumentID         currency.Pair `json:"instId"`
	OpenInterest         types.Number  `json:"oi"`
	OpenInterestCurrency types.Number  `json:"oiCcy"`
	OpenInterestUSD      types.Number  `json:"oiUsd"`
	Timestamp            types.Time    `json:"ts"`
}

// WebsocketFundingRatePush holds a funding-rate channel push
type WebsocketFundingRatePush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketFundingRate `json:"data"`
}

// WebsocketFundingRate holds the funding rate of a perpetual or X-Perps contract
type WebsocketFundingRate struct {
	InstrumentType        string        `json:"instType"`
	InstrumentID          currency.Pair `json:"instId"`
	Method                string        `json:"method"`
	FormulaType           string        `json:"formulaType"`
	FundingRate           types.Number  `json:"fundingRate"`
	NextFundingRate       types.Number  `json:"nextFundingRate"`
	FundingTime           types.Time    `json:"fundingTime"`
	NextFundingTime       types.Time    `json:"nextFundingTime"`
	PreviousFundingTime   types.Time    `json:"prevFundingTime"`
	MinimumFundingRate    types.Number  `json:"minFundingRate"`
	MaximumFundingRate    types.Number  `json:"maxFundingRate"`
	InterestRate          types.Number  `json:"interestRate"`
	ImpactValue           types.Number  `json:"impactValue"`
	SettlementState       string        `json:"settState"`
	SettlementFundingRate types.Number  `json:"settFundingRate"`
	Premium               types.Number  `json:"premium"`
	Timestamp             types.Time    `json:"ts"`
}

// WebsocketPriceLimitPush holds a price-limit channel push
type WebsocketPriceLimitPush struct {
	Argument SubscriptionArgument  `json:"arg"`
	Data     []WebsocketPriceLimit `json:"data"`
}

// WebsocketPriceLimit holds the highest buy and lowest sell prices of an instrument
type WebsocketPriceLimit struct {
	InstrumentType string        `json:"instType"`
	InstrumentID   currency.Pair `json:"instId"`
	BuyLimit       types.Number  `json:"buyLmt"`
	SellLimit      types.Number  `json:"sellLmt"`
	Timestamp      types.Time    `json:"ts"`
	Enabled        bool          `json:"enabled"`
}

// WebsocketOptionSummaryPush holds an opt-summary channel push
type WebsocketOptionSummaryPush struct {
	Argument SubscriptionArgument     `json:"arg"`
	Data     []WebsocketOptionSummary `json:"data"`
}

// WebsocketOptionSummary holds the pricing information of an option
type WebsocketOptionSummary struct {
	InstrumentType       string        `json:"instType"`
	InstrumentID         currency.Pair `json:"instId"`
	Underlying           string        `json:"uly"`
	Delta                types.Number  `json:"delta"`
	Gamma                types.Number  `json:"gamma"`
	Vega                 types.Number  `json:"vega"`
	Theta                types.Number  `json:"theta"`
	DeltaBlackScholes    types.Number  `json:"deltaBS"`
	GammaBlackScholes    types.Number  `json:"gammaBS"`
	VegaBlackScholes     types.Number  `json:"vegaBS"`
	ThetaBlackScholes    types.Number  `json:"thetaBS"`
	Leverage             types.Number  `json:"lever"`
	MarkVolatility       types.Number  `json:"markVol"`
	BidVolatility        types.Number  `json:"bidVol"`
	AskVolatility        types.Number  `json:"askVol"`
	RealisedVolatility   types.Number  `json:"realVol"`
	AtTheMoneyVolatility types.Number  `json:"volLv"`
	ForwardPrice         types.Number  `json:"fwdPx"`
	BuyAnnualisedReturn  types.Number  `json:"buyApr"`
	SellAnnualisedReturn types.Number  `json:"sellApr"`
	Distance             types.Number  `json:"distance"`
	Timestamp            types.Time    `json:"ts"`
}

// WebsocketEstimatedPricePush holds an estimated-price channel push
type WebsocketEstimatedPricePush struct {
	Argument SubscriptionArgument      `json:"arg"`
	Data     []WebsocketEstimatedPrice `json:"data"`
}

// WebsocketEstimatedPrice holds the estimated delivery, exercise or settlement price of a contract
type WebsocketEstimatedPrice struct {
	InstrumentType  string        `json:"instType"`
	InstrumentID    currency.Pair `json:"instId"`
	SettlementType  string        `json:"settleType"`
	SettlementPrice types.Number  `json:"settlePx"`
	Timestamp       types.Time    `json:"ts"`
}

// WebsocketMarkPricePush holds a mark-price channel push
type WebsocketMarkPricePush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketMarkPrice `json:"data"`
}

// WebsocketMarkPrice holds the mark price of an instrument
type WebsocketMarkPrice struct {
	InstrumentType string        `json:"instType"`
	InstrumentID   currency.Pair `json:"instId"`
	MarkPrice      types.Number  `json:"markPx"`
	Timestamp      types.Time    `json:"ts"`
}

// WebsocketIndexTickersPush holds an index-tickers channel push
type WebsocketIndexTickersPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketIndexTicker `json:"data"`
}

// WebsocketIndexTicker holds an index ticker
type WebsocketIndexTicker struct {
	InstrumentID       currency.Pair `json:"instId"`
	IndexPrice         types.Number  `json:"idxPx"`
	OpenPrice24Hour    types.Number  `json:"open24h"`
	HighestPrice24Hour types.Number  `json:"high24h"`
	LowestPrice24Hour  types.Number  `json:"low24h"`
	OpenPriceUTC0      types.Number  `json:"sodUtc0"`
	OpenPriceUTC8      types.Number  `json:"sodUtc8"`
	Timestamp          types.Time    `json:"ts"`
}

// WebsocketMarkPriceCandlesticksPush holds a mark price candlestick channel push
type WebsocketMarkPriceCandlesticksPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketPriceCandlestick `json:"data"`
}

// WebsocketIndexCandlesticksPush holds an index candlestick channel push
type WebsocketIndexCandlesticksPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketPriceCandlestick `json:"data"`
}

// WebsocketPriceCandlestick holds a mark price or index candlestick, which has no volume
type WebsocketPriceCandlestick struct {
	Timestamp    types.Time
	OpenPrice    types.Number
	HighestPrice types.Number
	LowestPrice  types.Number
	ClosePrice   types.Number
	Confirm      types.Boolean
}

// UnmarshalJSON decodes a mark price or index candlestick from its array form
func (c *WebsocketPriceCandlestick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&c.Timestamp, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &c.Confirm})
}

// WebsocketLiquidationOrdersPush holds a liquidation-orders channel push
type WebsocketLiquidationOrdersPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketLiquidationOrder `json:"data"`
}

// WebsocketLiquidationOrder holds the liquidations of an instrument
type WebsocketLiquidationOrder struct {
	InstrumentType   string                            `json:"instType"`
	InstrumentID     currency.Pair                     `json:"instId"`
	InstrumentFamily string                            `json:"instFamily"`
	Underlying       string                            `json:"uly"`
	Details          []WebsocketLiquidationOrderDetail `json:"details"`
}

// WebsocketLiquidationOrderDetail holds a liquidation
type WebsocketLiquidationOrderDetail struct {
	Side            string        `json:"side"` // Empty on margin and option liquidations
	PositionSide    string        `json:"posSide"`
	BankruptcyPrice types.Number  `json:"bkPx"`
	Size            types.Number  `json:"sz"`
	BankruptcyLoss  types.Number  `json:"bkLoss"`
	Currency        currency.Code `json:"ccy"`
	Timestamp       types.Time    `json:"ts"`
}

// WebsocketADLWarningPush holds an adl-warning channel push
type WebsocketADLWarningPush struct {
	Argument SubscriptionArgument  `json:"arg"`
	Data     []WebsocketADLWarning `json:"data"`
}

// WebsocketADLWarning holds the insurance fund state of an instrument family under an auto-deleveraging warning
type WebsocketADLWarning struct {
	InstrumentType          string        `json:"instType"`
	InstrumentFamily        string        `json:"instFamily"`
	State                   string        `json:"state"`
	Balance                 types.Number  `json:"bal"`
	Currency                currency.Code `json:"ccy"`
	MaximumBalance          types.Number  `json:"maxBal"`
	MaximumBalanceTimestamp types.Time    `json:"maxBalTs"`
	ADLType                 string        `json:"adlType"`
	ADLBalance              types.Number  `json:"adlBal"`
	ADLRecoveryBalance      types.Number  `json:"adlRecBal"`
	Timestamp               types.Time    `json:"ts"`
	DeclineRate             types.Number  `json:"decRate"`
	ADLRate                 types.Number  `json:"adlRate"`
	ADLRecoveryRate         types.Number  `json:"adlRecRate"`
}

// WebsocketEconomicCalendarPush holds an economic-calendar channel push
type WebsocketEconomicCalendarPush struct {
	Argument SubscriptionArgument        `json:"arg"`
	Data     []WebsocketEconomicCalendar `json:"data"`
}

// WebsocketEconomicCalendar holds a macro-economic calendar event
type WebsocketEconomicCalendar struct {
	Event           string        `json:"event"`
	Region          string        `json:"region"`
	Category        string        `json:"category"`
	Actual          types.Number  `json:"actual"`
	Previous        types.Number  `json:"previous"`
	Forecast        types.Number  `json:"forecast"`
	PreviousInitial types.Number  `json:"prevInitial"`
	Date            types.Time    `json:"date"`
	ReferenceDate   types.Time    `json:"refDate"`
	CalendarID      string        `json:"calendarId"`
	Unit            string        `json:"unit"`
	Currency        currency.Code `json:"ccy"`
	Importance      string        `json:"importance"`
	Timestamp       types.Time    `json:"ts"`
}

// WebsocketDepositInfoPush holds a deposit-info channel push
type WebsocketDepositInfoPush struct {
	Argument SubscriptionArgument   `json:"arg"`
	Data     []WebsocketDepositInfo `json:"data"`
}

// WebsocketDepositInfo holds a deposit
type WebsocketDepositInfo struct {
	UserID                     string        `json:"uid"`
	SubAccount                 string        `json:"subAcct"`
	PushTime                   types.Time    `json:"pTime"`
	Currency                   currency.Code `json:"ccy"`
	Chain                      string        `json:"chain"`
	Amount                     types.Number  `json:"amt"`
	From                       string        `json:"from"`
	AreaCodeFrom               string        `json:"areaCodeFrom"`
	To                         string        `json:"to"`
	TransactionID              string        `json:"txId"`
	Timestamp                  types.Time    `json:"ts"`
	State                      string        `json:"state"`
	DepositID                  string        `json:"depId"`
	FromWithdrawalID           string        `json:"fromWdId"`
	ActualDepositBlockConfirms uint64        `json:"actualDepBlkConfirm,string"`
}

// WebsocketWithdrawalInfoPush holds a withdrawal-info channel push
type WebsocketWithdrawalInfoPush struct {
	Argument SubscriptionArgument      `json:"arg"`
	Data     []WebsocketWithdrawalInfo `json:"data"`
}

// WebsocketWithdrawalInfo holds a withdrawal
type WebsocketWithdrawalInfo struct {
	UserID           string                          `json:"uid"`
	SubAccount       string                          `json:"subAcct"`
	PushTime         types.Time                      `json:"pTime"`
	Currency         currency.Code                   `json:"ccy"`
	Chain            string                          `json:"chain"`
	NonTradableAsset bool                            `json:"nonTradableAsset"`
	Amount           types.Number                    `json:"amt"`
	Timestamp        types.Time                      `json:"ts"`
	From             string                          `json:"from"`
	AreaCodeFrom     string                          `json:"areaCodeFrom"`
	To               string                          `json:"to"`
	AreaCodeTo       string                          `json:"areaCodeTo"`
	ToAddressType    string                          `json:"toAddrType"`
	Tag              string                          `json:"tag"`
	PaymentID        string                          `json:"pmtId"`
	Memo             string                          `json:"memo"`
	AddressExtension WebsocketWithdrawalAddressExtra `json:"addrEx"`
	TransactionID    string                          `json:"txId"`
	Fee              types.Number                    `json:"fee"`
	FeeCurrency      currency.Code                   `json:"feeCcy"`
	State            string                          `json:"state"`
	WithdrawalID     string                          `json:"wdId"`
	ClientID         string                          `json:"clientId"`
	Note             string                          `json:"note"`
}

// WebsocketWithdrawalAddressExtra holds the attachment a withdrawal address needs, such as a TON comment
type WebsocketWithdrawalAddressExtra map[string]string

// WebsocketStatusPush holds a status channel push
type WebsocketStatusPush struct {
	Argument SubscriptionArgument `json:"arg"`
	Data     []WebsocketStatus    `json:"data"`
}

// WebsocketStatus holds a system maintenance notice
type WebsocketStatus struct {
	Title               string     `json:"title"`
	State               string     `json:"state"`
	Begin               types.Time `json:"begin"`
	End                 types.Time `json:"end"`
	PreOpenBegin        types.Time `json:"preOpenBegin"`
	Href                string     `json:"href"`
	ServiceType         string     `json:"serviceType"`
	System              string     `json:"system"`
	ScheduleDescription string     `json:"scheDesc"`
	MaintenanceType     string     `json:"maintType"`
	Environment         string     `json:"env"`
	Timestamp           types.Time `json:"ts"`
}
