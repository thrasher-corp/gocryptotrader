package binance

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// rateLimitPool is a Binance request budget. The endpoint limits drawing on a pool share its limiter, so their weights
// count against one budget the way Binance counts them, unless Binance budgets the pool's endpoints separately
type rateLimitPool uint8

// Rate limit pools: the shared budgets of each host, then dedicated budgets for endpoints Binance limits apart from
// their weight on a shared pool. A limiter queues requests in turn, so a request drawing on a host's IP pool waits
// behind every market data request reserved before it, such as an order book synchronisation's snapshots. A host's
// cancel pool is the share of its IP budget kept for order management that must not wait on market data:
// cancellations, the listing a cancellation of every symbol's orders starts with, and the position mode lookup of a
// Long or Short futures order. A heartbeat pool is the share kept for the countdown cancel-all heartbeats and settings,
// as a heartbeat arriving after its countdown has run out has Binance cancel every order it covers
const (
	spotIPPool rateLimitPool = iota
	spotCancelPool
	spotOrderPool
	sapiIPPool
	sapiUIDPool
	derivativesIPPool
	derivativesCancelPool
	derivativesHeartbeatPool
	derivativesOrderPool
	futuresDataPool
	usdtMarginedFuturesFundingPool
	optionsIPPool
	optionsCancelPool
	optionsHeartbeatPool
	optionsOrderPool
	portfolioMarginIPPool
	portfolioMarginOrderPool

	marginMaxLeveragePool
	withdrawalHistoryPool
	subAccountFuturesTransferPool
	portfolioMarginProBNBTransferPool
	autoInvestPlanCreationPool
	autoInvestPlanAdjustmentPool
	autoInvestPlanStatusPool
	autoInvestPlanListPool
	autoInvestOneTimeTransactionPool
	autoInvestRedemptionPool
	flexibleLoanLTVAdjustmentHistoryPool
	vipLoanApplicationStatusPool
	brokerCreateAPIKeyPool
	brokerDeleteAPIKeyPool
	brokerFuturesTransferPool
	portfolioMarginBNBTransferPool
)

// perEndpoint reports whether Binance gives every endpoint drawing on the pool a budget of its own
func (p rateLimitPool) perEndpoint() bool {
	return p == sapiIPPool || p == sapiUIDPool
}

// rateLimitBudget is the weight a pool allows in each interval
type rateLimitBudget struct {
	interval time.Duration
	limit    int
}

// poolBudgets holds the budget of every pool, from each host's exchangeInfo rateLimits and the documentation. The
// limiter spreads a budget evenly with a burst of one, so where a host has two windows the stricter sustained rate
// applies, and daily or monthly quotas are left to Binance: spread evenly they would hold a second call for hours. A
// host's IP, cancel and heartbeat pools together hold its IP limit. The cancellations' share is about the host's order
// rate. The heartbeats' share keeps up with a heartbeat every 30 seconds, the cadence Binance's documentation gives,
// for each of five USDⓈ-M or COIN-M symbols, and with an options heartbeat, which covers every underlying, every 15
// seconds; heartbeats sent faster than it refills queue on the IP pool as other requests do
var poolBudgets = map[rateLimitPool]rateLimitBudget{
	spotIPPool:     {time.Minute, 5400},     // REQUEST_WEIGHT per IP, 6000 less the cancellations' share
	spotCancelPool: {time.Minute, 600},      // WebSocket API requests count against REQUEST_WEIGHT too
	spotOrderPool:  {10 * time.Second, 100}, // ORDERS (unfilled order count) per account; 200000 a day also applies
	// Binance gives each /sapi endpoint a budget of its own, so every endpoint limit heavier than 1 gets a limiter of
	// its own and a heavy call never holds up another endpoint. The weight-1 defaults, which most /sapi endpoints
	// use, share one limiter per pool: stricter, but still 200 IP-limited requests a second
	sapiIPPool:  {time.Minute, 12000},
	sapiUIDPool: {time.Minute, 180000},
	// USDⓈ-M and COIN-M share their budgets since the 2026-06-30 CM-UM integration. Orders are also limited to 300 per
	// 10 seconds, a looser sustained rate
	derivativesIPPool:              {time.Minute, 1800}, // 2400 less the cancellations' and heartbeats' shares
	derivativesCancelPool:          {time.Minute, 450},
	derivativesHeartbeatPool:       {time.Minute, 150},
	derivativesOrderPool:           {time.Minute, 1200},
	futuresDataPool:                {5 * time.Minute, 1000}, // per IP, for the /futures/data endpoints of both hosts
	usdtMarginedFuturesFundingPool: {5 * time.Minute, 500},  // per IP, shared by /fapi/v1/fundingRate and /fapi/v1/fundingInfo
	optionsIPPool:                  {time.Minute, 240},      // 400 less the cancellations' and heartbeats' shares
	optionsCancelPool:              {time.Minute, 100},
	optionsHeartbeatPool:           {time.Minute, 60},
	optionsOrderPool:               {time.Minute, 100}, // 30 per 10 seconds also applies, a looser sustained rate
	portfolioMarginIPPool:          {time.Minute, 6000},
	portfolioMarginOrderPool:       {time.Minute, 1200},

	marginMaxLeveragePool: {time.Minute, 1}, // POST /sapi/v1/margin/max-leverage
	// GET /sapi/v1/capital/withdraw/history: its weight of 18000 counts against a UID budget of 180000 a second of its
	// own, which Binance documents as 10 requests a second
	withdrawalHistoryPool: {time.Second, 10},
	// POST /sapi/v1/sub-account/futures/internalTransfer, per master account
	subAccountFuturesTransferPool:     {time.Minute, 2000},
	portfolioMarginProBNBTransferPool: {10 * time.Minute, 2}, // POST /sapi/v1/portfolio/bnb-transfer
	// Auto-Invest allowed one request every 3 seconds on each of these endpoints when it was last documented
	autoInvestPlanCreationPool:           {3 * time.Second, 1},
	autoInvestPlanAdjustmentPool:         {3 * time.Second, 1},
	autoInvestPlanStatusPool:             {3 * time.Second, 1},
	autoInvestPlanListPool:               {3 * time.Second, 1},
	autoInvestOneTimeTransactionPool:     {3 * time.Second, 1},
	autoInvestRedemptionPool:             {3 * time.Second, 1},
	flexibleLoanLTVAdjustmentHistoryPool: {time.Second, 5}, // GET /sapi/v2/loan/flexible/ltv/adjustment/history
	vipLoanApplicationStatusPool:         {time.Second, 5}, // GET /sapi/v1/loan/vip/request/data
	// Binance allows one API key a second per sub-account; one a second across all sub-accounts is stricter
	brokerCreateAPIKeyPool:         {time.Second, 1},
	brokerDeleteAPIKeyPool:         {time.Second, 1},
	brokerFuturesTransferPool:      {time.Minute, 5000},    // POST /sapi/v1/broker/transfer/futures, per master account
	portfolioMarginBNBTransferPool: {10 * time.Minute, 10}, // POST /papi/v1/bnb-transfer
}

// poolBursts holds the weight of the pools whose requests may go out at once, which the rest of the pool's budget
// refills over its interval, so no interval exceeds the budget. A cancel pool's burst takes a cancellation of every
// symbol's orders, its listings included, on each of the host's assets, and a heartbeat pool's a round of heartbeats.
// Every other pool spreads its budget evenly
var poolBursts = map[rateLimitPool]int{
	spotCancelPool:           100,
	derivativesCancelPool:    200,
	derivativesHeartbeatPool: 50,
	optionsCancelPool:        50,
	optionsHeartbeatPool:     20,
}

// newPoolLimiter returns a limiter holding a pool to its budget
func newPoolLimiter(pool rateLimitPool) *rate.Limiter {
	budget := poolBudgets[pool]
	burst := poolBursts[pool]
	if burst <= 1 {
		return request.NewRateLimit(budget.interval, budget.limit)
	}
	return rate.NewLimiter(rate.Limit(float64(budget.limit-burst)/budget.interval.Seconds()), burst)
}

// Endpoint limits, grouped by host and documentation section. They start after request.UnAuth, so a stray
// request.Unset, request.Auth or request.UnAuth finds no Binance limiter instead of silently using one. Weight-1
// endpoints share their pool's default limit, and an endpoint whose weight depends on a parameter has a limit per
// tier, picked by a helper below
const (
	// Spot /api, REST and WebSocket API
	spotDefaultRate request.EndpointLimit = request.UnAuth + 1 + iota
	spotCancelRate
	spotExchangeInfoRate
	aggTradesRate
	getCurrentAveragePriceRate
	spotOrderbookDepth100Rate
	spotOrderbookDepth500Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	getRecentTradesListRate
	getOldTradeLookupRate
	getKlineRate
	spotTickerSymbols1Rate
	spotTickerSymbols5Rate
	spotTickerSymbols10Rate
	spotTickerSymbols25Rate
	spotTickerSymbolsMaxRate
	getTickers20Rate
	getTickers100Rate
	spotPriceChangeAllRate
	spotBookTickerRate
	spotOrderbookTickerAllRate
	spotSymbolPriceRate
	spotSymbolPriceAllRate
	spotOrderRate
	spotOneCancelsTheOtherOrderRate
	testNewOrderWithCommissionRate
	getCommissionRate
	getAllOneCancelsTheOtherOrdersRate
	spotAllOrdersRate
	spotAccountInformationRate
	spotOpenOrdersSpecificRate
	spotOpenOrdersAllRate
	spotCancelAllListingRate
	spotOrderQueryRate
	getOneCancelsTheOtherListRate
	getAllocationsRate
	preventedMatchesRate
	preventedMatchesByOrderIDRate
	accountTradeListRate
	accountTradeListByOrderIDRate
	getOpenOneCancelsTheOtherListRate
	currentOrderCountUsageRate
	wsSessionRate
	userDataStreamSubscribeRate

	// /sapi weight-1 defaults, used by every SAPI family
	sapiDefaultRate
	sapiUIDDefaultRate

	// Margin
	adjustCrossMarginMaxLeverageRate
	getIsolatedMarginAccountInfoRate
	enableIsolatedMarginAccountRate
	disableIsolatedMarginAccountRate
	marginAccountSummaryRate
	marginCapitalFlowRate
	getCrossMarginAccountDetailRate
	allCrossMarginFeeDataRate
	allIsolatedMarginFeeDataRate
	marginHourlyInterestRate
	borrowRepayRecordsInMarginAccountRate
	marginAccountBorrowRepayRate
	marginMaxBorrowRate
	crossMarginCollateralRatioRate
	allIsolatedMarginSymbolsRate
	marginTokensAndSymbolsDelistScheduleRate
	marginAvailableInventoryRate
	getPriceMarginIndexRate
	getSmallLiabilityExchangeCoinListRate
	marginSmallLiabilityExchangeRate
	smallLiabilityExchangeHistoryRate
	getMarginAccountsOpenOrdersRate
	getMarginAccountOneCancelsTheOtherOrderRate
	getMarginAccountOrderRate
	marginAccountNewOrderRate
	marginAccountNewOrderBorrowRate
	marginAccountCancelOrderRate
	marginManualLiquidationRate
	marginCurrentOrderCountUsageRate
	getMarginAccountAllOneCancelsTheOtherRate
	marginAccountsAllOrdersRate
	marginAccountOpenOneCancelsTheOtherOrdersRate
	marginAccountTradeListRate
	maxTransferOutRate

	// Wallet
	dailyAccountSnapshotRate
	assetDividendRecordRate
	dustTransferRate
	cloudMiningPaymentAndRefundHistoryRate
	getUserDelegationHistoryRate
	userUniversalTransferRate
	getUserWalletBalanceRate
	userAssetsRate
	allCoinInfoRate
	depositAddressesRate
	getDepositAddressListInNetworkRate
	withdrawAddressListRate
	fundWithdrawalRate
	withdrawalHistoryRate
	symbolDelistScheduleForSpotRate
	depositQuestionnaireRate

	// Sub-account
	getFuturesPositionRiskOfSubAccountV1Rate
	getSubAccountStatusOnMarginOrFuturesRate
	getSubAccountTransactionStatisticsRate
	addIPRestrictionSubAccountAPIKeyRate
	deleteIPListForSubAccountAPIKeyRate
	ipRestrictionForSubAccountAPIKeyRate
	getDetailSubAccountFuturesAccountRate
	subAccountMarginAccountDetailRate
	getFuturesSubAccountSummaryV2Rate
	getSubAccountSummaryOfMarginAccountRate
	getV3SubAccountAssetsRate
	getSubAccountAssetRate
	subAccountFuturesAssetTransferRate
	universalTransferForMasterAccountRate
	managedSubAccountFuturesAssetDetailRate
	getManagedSubAccountListRate
	getManagedSubAccountSnapshotRate
	managedSubAccountTransferLogRate

	// Portfolio Margin Pro, formerly classic portfolio margin
	transferBNBRate
	getAutoRepayFuturesStatusRate
	changeAutoRepayFuturesStatusRate
	fundAutoCollectionRate
	fundCollectionByAssetRate
	classicPortfolioMarginAccountInfoRate
	repayClassicPortfolioMarginBankruptcyLoanRate
	getClassicPortfolioMarginBankruptcyLoanAmountRate
	classicPortfolioMarginNegativeBalanceInterestHistoryRate
	repayFuturesNegativeBalanceRate
	portfolioMarginAssetLeverageRate
	classicPortfolioMarginCollateralRate
	portfolioMarginAssetIndexPriceRate

	// Simple Earn
	personalLeftQuotaRate
	getFlexibleSimpleEarnProductPositionRate
	getRedemptionRecordRate
	getRewardHistoryRate
	subscriptionPreviewRate
	getFlexibleSubscriptionRecordRate
	getSimpleEarnProductPositionRate
	getLockedSubscriptionRecordsRate
	simpleEarnRateHistoryRate
	simpleEarnProductsRate
	setAutoSubscribeRate
	setRedeemOptionRate
	simpleAccountRate

	// Staking
	ethStakingAccountRate
	bethRewardDistributionHistoryRate
	currentETHStakingQuotaRate
	ethRedemptionHistoryRate
	ethStakingHistoryRate
	getWBETHRateHistoryRate
	wbethRewardsHistoryRate
	wbethWrapOrUnwrapHistoryRate
	ethereumStakingRedemptionRate
	subscribeETHStakingRate
	wrapBETHRate
	claimBoostRewardsRate
	bnsolRateHistoryRate
	bnsolRewardsHistoryRate
	boostRewardsHistoryRate
	solRedemptionHistoryRate
	solStakingHistoryRate
	solStakingQuotaDetailsRate
	unclaimedRewardsRate
	redeemSOLRate
	solStakingAccountRate
	subscribeSOLStakingRate

	// Auto-Invest, no longer documented
	autoInvestPlanCreationRate
	autoInvestPlanAdjustmentRate
	autoInvestPlanStatusRate
	autoInvestPlanListRate
	autoInvestOneTimeTransactionRate
	autoInvestRedemptionRate

	// Crypto Loan
	checkCollateralRepayRate
	adjustFlexibleLoanRate
	borrowFlexibleRate
	repayFlexibleLoanRate
	flexibleLoanAssetDataRate
	flexibleBorrowHistoryRate
	flexibleLoanCollateralAssetRate
	flexibleLoanLiquidationHistoryRate
	flexibleLoanLTVAdjustmentHistoryRate
	getFlexibleLoanOngoingOrdersRate
	flexibleLoanRepaymentHistoryRate
	cryptoLoansIncomeHistoryRate
	getLoanBorrowHistoryRate
	getLoanLTVAdjustmentHistoryRate
	repaymentHistoryRate

	// VIP Loan
	getVIPBorrowInterestRate
	getCollateralAssetDataRate
	getVIPLoanableAssetsRate
	vipLoanInterestRateHistoryRate
	vipLoanBorrowRate
	vipLoanRenewRate
	vipLoanRepayRate
	checkLockedValueVIPCollateralAccountRate
	getVIPLoanAccruedInterestRate
	getVIPLoanOngoingOrdersRate
	getVIPLoanRepaymentHistoryRate
	getApplicationStatusRate

	// Convert
	getAllConvertPairsRate
	getOrderQuantityPrecisionPerAssetRate
	acceptQuoteRate
	cancelLimitOrderRate
	convertTradeFlowHistoryRate
	convertOrderStatusRate
	placeLimitOrderRate
	getLimitOpenOrdersRate
	sendQuoteRequestRate

	// Algo
	placeTWAveragePriceNewOrderRate
	placeVPOrderRate
	spotTWAPNewOrderRate

	// Fiat, Pay, Rebate and NFT
	fiatDepositWithdrawHistRate
	payTradeHistoryRate
	spotRebateHistoryRate
	nftRate

	// Binance Link (CAAS)
	createAPIKeyForSubAccountRate
	deleteAPIKeyForSubAccountRate
	updateIPRestrictionForSubAccountAPIKeyRate
	brokerSubAccountDepositHistoryRate
	brokerSubAccountFuturesAssetInfoRate
	brokerSubAccountSpotAssetInfoRate
	brokerFuturesTransferRate
	brokerSubAccountCommissionRate

	// USDⓈ-M futures /fapi
	usdtMarginedFuturesDefaultRate
	usdtMarginedFuturesCancelRate
	usdtMarginedFuturesCancelAllListingRate
	usdtMarginedFuturesAccountInformationRate
	usdtMarginedFuturesTradingStatusAllRate
	usdtMarginedFuturesMultiAssetMarginRate
	usdtMarginedFuturesPositionModeRate
	usdtMarginedFuturesPositionModeLookupRate
	usdtMarginedFuturesDownloadIDRate
	usdtMarginedFuturesDownloadLinkRate
	usdtMarginedFuturesIncomeHistoryRate
	usdtMarginedFuturesUserCommissionRate
	usdtMarginedFuturesAggregateTradesRate
	usdtMarginedFuturesFundingRateHistoryRate
	usdtMarginedFuturesFundingInfoRate
	usdtMarginedFuturesKline100Rate
	usdtMarginedFuturesKline500Rate
	usdtMarginedFuturesKline1000Rate
	usdtMarginedFuturesKlineMaxRate
	usdtMarginedFuturesMarkPriceAllRate
	usdtMarginedFuturesAssetIndexAllRate
	usdtMarginedFuturesHistoricalTradesRate
	usdtMarginedFuturesOrderbook50Rate
	usdtMarginedFuturesOrderbook100Rate
	usdtMarginedFuturesOrderbook500Rate
	usdtMarginedFuturesOrderbook1000Rate
	usdtMarginedFuturesIndexConstituentsRate
	usdtMarginedFuturesRecentTradesRate
	usdtMarginedFuturesRetailPriceImprovementOrderbookRate
	usdtMarginedFuturesBookTickerRate
	usdtMarginedFuturesOrderbookTickerAllRate
	usdtMarginedFuturesSymbolPriceAllRate
	usdtMarginedFuturesTicker24HourAllRate
	usdtMarginedFuturesAccountTradeListRate
	usdtMarginedFuturesGetAllOrdersRate
	usdtMarginedFuturesCountdownCancelRate
	usdtMarginedFuturesHeartbeatOverflowRate
	usdtMarginedFuturesOrdersDefaultRate
	usdtMarginedFuturesBatchOrdersRate
	usdtMarginedFuturesOpenAlgoOrdersAllRate
	usdtMarginedFuturesGetAllOpenOrdersRate
	usdtMarginedFuturesAutoDeleveragingQuantileRate
	usdtMarginedFuturesPositionRiskRate
	usdtMarginedFuturesAllAlgoOrdersRate
	usdtMarginedFuturesSymbolForceOrdersRate
	usdtMarginedFuturesAllForceOrdersRate
	usdtMarginedFuturesAPIReferralRate

	// /futures/data on the USDⓈ-M and COIN-M hosts
	futuresDataRate

	// COIN-M futures /dapi
	coinMarginedFuturesDefaultRate
	coinMarginedFuturesCancelRate
	coinMarginedFuturesCancelAllListingRate
	coinMarginedFuturesAccountInformationRate
	coinMarginedFuturesIncomeHistoryRate
	coinMarginedFuturesAggregateTradesRate
	coinMarginedFuturesKline100Rate
	coinMarginedFuturesKline500Rate
	coinMarginedFuturesKline1000Rate
	coinMarginedFuturesKlineMaxRate
	coinMarginedFuturesIndexMarkPriceRate
	coinMarginedFuturesHistoricalTradesRate
	coinMarginedFuturesOrderbook50Rate
	coinMarginedFuturesOrderbook100Rate
	coinMarginedFuturesOrderbook500Rate
	coinMarginedFuturesOrderbook1000Rate
	coinMarginedFuturesRecentTradesRate
	coinMarginedFuturesBookTickerRate
	coinMarginedFuturesOrderbookTickerAllRate
	coinMarginedFuturesSymbolPriceAllRate
	coinMarginedFuturesTicker24HourAllRate
	coinMarginedFuturesAccountTradeListRate
	coinMarginedFuturesAllOrdersRate
	coinMarginedFuturesCountdownCancelRate
	coinMarginedFuturesHeartbeatOverflowRate
	coinMarginedFuturesBatchOrdersRate
	coinMarginedFuturesOrdersDefaultRate
	coinMarginedFuturesGetAllOpenOrdersRate
	coinMarginedFuturesOpenAlgoOrdersAllRate
	coinMarginedFuturesAllAlgoOrdersRate
	coinMarginedFuturesAutoDeleveragingQuantileRate
	coinMarginedFuturesSymbolForceOrdersRate
	coinMarginedFuturesAllForceOrdersRate

	// Options /eapi
	optionsDefaultRate
	optionsCancelRate
	optionsCancelAllListingRate
	optionsMarginAccountInfoRate
	optionsDownloadIDForOptionTransactionHistoryRate
	optionsGetTransHistoryDownloadLinkByIDRate
	optionsHistoricalExerciseRecordsRate
	optionsMarkPriceRate
	optionsOrderbook50Rate
	optionsOrderbook100Rate
	optionsOrderbook500Rate
	optionsOrderbook1000Rate
	optionsRecentTradesRate
	optionsAllTickerPriceStatisticsRate
	optionsAutoCancelAllOpenOrdersHeartbeatRate
	optionsHeartbeatOverflowRate
	optionsAutoCancelAllOpenOrdersConfigRate
	optionsAccountTradeListRate
	optionsCancelAllByUnderlyingRate
	optionsBatchOrderRate
	optionsCancelBatchOrdersRate
	optionsDefaultOrderRate
	optionsPositionInformationRate
	optionsAllQueryOpenOrdersRate
	optionsGetOrderHistoryRate
	optionsUserExerciseRecordRate

	// Portfolio margin /papi
	portfolioMarginDefaultRate
	portfolioMarginGetAccountBalancesRate
	portfolioMarginGetAccountInformationRate
	portfolioMarginBNBTransferRate
	portfolioMarginGetAutoRepayFuturesStatusRate
	portfolioMarginChangeAutoRepayFuturesStatusRate
	portfolioMarginGetCoinMarginedCurrentPositionModeRate
	portfolioMarginGetUSDTMarginedCurrentPositionModeRate
	portfolioMarginFundAutoCollectionRate
	portfolioMarginFundCollectionByAssetRate
	portfolioMarginGetCoinMarginedAccountDetailRate
	portfolioMarginGetCoinMarginedIncomeHistoryRate
	portfolioMarginGetUSDTMarginedAccountDetailRate
	portfolioMarginGetUSDTMarginedIncomeHistoryRate
	portfolioMarginGetCoinMarginedUserCommissionRate
	portfolioMarginGetUSDTMarginedUserCommissionRate
	portfolioMarginMarginMaxBorrowRate
	portfolioMarginUSDTMarginedTradingQuantitativeRulesIndicatorsRate
	portfolioMarginGetMarginLoanRecordRate
	portfolioMarginGetMarginMaxWithdrawalRate
	portfolioMarginGetMarginRepayRecordRate
	portfolioMarginGetNegativeBalanceInterestHistoryRate
	portfolioMarginGetUSDTMarginedPositionInformationRate
	portfolioMarginNegativeBalanceExchangeRecordRate
	portfolioMarginRepayFuturesNegativeBalanceRate
	portfolioMarginOrderRate
	portfolioMarginCancelMarginAccountOpenOrdersOnSymbolRate
	portfolioMarginGetMarginAccountOneCancelsTheOtherRate
	portfolioMarginCancelMarginAccountOneCancelsTheOtherRate
	portfolioMarginGetMarginAccountOrderRate
	portfolioMarginCancelMarginAccountOrderRate
	portfolioMarginGetCoinMarginedAccountTradeListWithSymbolRate
	portfolioMarginGetCoinMarginedAccountTradeListWithPairRate
	portfolioMarginGetCoinMarginedPositionAutoDeleveragingQuantileEstimationRate
	portfolioMarginMarginAccountLoanAndRepayRate
	portfolioMarginOneCancelsTheOtherOrderRate
	portfolioMarginGetMarginAccountTradeListRate
	portfolioMarginAllCoinMarginedConditionalOrderWithoutSymbolRate
	portfolioMarginAllCoinMarginedOrderWithSymbolRate
	portfolioMarginAllCoinMarginedOrderWithPairRate
	portfolioMarginAllCoinMarginedOpenConditionalOrdersWithoutSymbolRate
	portfolioMarginRetrieveAllCoinMarginedOpenOrdersForAllSymbolRate
	portfolioMarginRetrieveAllUSDTMarginedOpenOrdersForAllSymbolRate
	portfolioMarginAllMarginAccountOrdersRate
	portfolioMarginGetAllUSDTMarginedOrdersRate
	portfolioMarginGetUSDTMarginedAlgoOrderHistoryRate
	portfolioMarginCurrentMarginOpenOrderRate
	portfolioMarginGetMarginAccountsAllOneCancelsTheOtherOrdersRate
	portfolioMarginGetMarginAccountsOpenOneCancelsTheOtherOrdersRate
	portfolioMarginGetUserCoinMarginedForceOrdersWithSymbolRate
	portfolioMarginGetUserCoinMarginedForceOrdersWithoutSymbolRate
	portfolioMarginGetUserUSDTMarginedForceOrdersWithSymbolRate
	portfolioMarginGetUserUSDTMarginedForceOrdersWithoutSymbolRate
	portfolioMarginGetUSDTMarginedAccountTradeListRate
	portfolioMarginGetUSDTMarginedPositionAutoDeleveragingQuantileEstimationRate
	portfolioMarginAPIReferralRate

	// Stream subscriptions and pings, which Binance limits per connection
	wsConnectionMessageRate
)

// endpointCost is the pool an endpoint limit draws on and the weight each request costs
type endpointCost struct {
	pool   rateLimitPool
	weight request.Weight
}

// endpointCosts holds the documented weight of every endpoint limit except wsConnectionMessageRate. An endpoint
// documented with a weight of 0 costs 1, the least the limiter takes. Order placements on a host with an order budget
// cost their order count on its order pool, which binds before their request weight does
var endpointCosts = map[request.EndpointLimit]endpointCost{
	// Spot /api, REST and WebSocket API
	spotDefaultRate:                    {spotIPPool, 1},
	spotCancelRate:                     {spotCancelPool, 1},
	spotExchangeInfoRate:               {spotIPPool, 20},
	aggTradesRate:                      {spotIPPool, 4},
	getCurrentAveragePriceRate:         {spotIPPool, 2},
	spotOrderbookDepth100Rate:          {spotIPPool, 5},
	spotOrderbookDepth500Rate:          {spotIPPool, 25},
	spotOrderbookDepth1000Rate:         {spotIPPool, 50},
	spotOrderbookDepth5000Rate:         {spotIPPool, 250},
	getRecentTradesListRate:            {spotIPPool, 25},
	getOldTradeLookupRate:              {spotIPPool, 25},
	getKlineRate:                       {spotIPPool, 2},
	spotTickerSymbols1Rate:             {spotIPPool, 4},
	spotTickerSymbols5Rate:             {spotIPPool, 20},
	spotTickerSymbols10Rate:            {spotIPPool, 40},
	spotTickerSymbols25Rate:            {spotIPPool, 100},
	spotTickerSymbolsMaxRate:           {spotIPPool, 200},
	getTickers20Rate:                   {spotIPPool, 2},
	getTickers100Rate:                  {spotIPPool, 40},
	spotPriceChangeAllRate:             {spotIPPool, 80},
	spotBookTickerRate:                 {spotIPPool, 2},
	spotOrderbookTickerAllRate:         {spotIPPool, 4},
	spotSymbolPriceRate:                {spotIPPool, 2},
	spotSymbolPriceAllRate:             {spotIPPool, 4},
	spotOrderRate:                      {spotOrderPool, 1},
	spotOneCancelsTheOtherOrderRate:    {spotOrderPool, 2},
	testNewOrderWithCommissionRate:     {spotIPPool, 20},
	getCommissionRate:                  {spotIPPool, 20},
	getAllOneCancelsTheOtherOrdersRate: {spotIPPool, 20},
	spotAllOrdersRate:                  {spotIPPool, 20},
	spotAccountInformationRate:         {spotIPPool, 20},
	spotOpenOrdersSpecificRate:         {spotIPPool, 6},
	spotOpenOrdersAllRate:              {spotIPPool, 80},
	spotCancelAllListingRate:           {spotCancelPool, 80},
	spotOrderQueryRate:                 {spotIPPool, 4},
	getOneCancelsTheOtherListRate:      {spotIPPool, 4},
	getAllocationsRate:                 {spotIPPool, 20},
	preventedMatchesRate:               {spotIPPool, 2},
	preventedMatchesByOrderIDRate:      {spotIPPool, 20},
	accountTradeListRate:               {spotIPPool, 20},
	accountTradeListByOrderIDRate:      {spotIPPool, 5},
	getOpenOneCancelsTheOtherListRate:  {spotIPPool, 6},
	currentOrderCountUsageRate:         {spotIPPool, 40},
	wsSessionRate:                      {spotIPPool, 2},
	userDataStreamSubscribeRate:        {spotIPPool, 2},

	// /sapi weight-1 defaults, used by every SAPI family
	sapiDefaultRate:    {sapiIPPool, 1},
	sapiUIDDefaultRate: {sapiUIDPool, 1},

	// Margin
	adjustCrossMarginMaxLeverageRate:              {marginMaxLeveragePool, 1},
	getIsolatedMarginAccountInfoRate:              {sapiIPPool, 10},
	enableIsolatedMarginAccountRate:               {sapiUIDPool, 300},
	disableIsolatedMarginAccountRate:              {sapiUIDPool, 300},
	marginAccountSummaryRate:                      {sapiIPPool, 10},
	marginCapitalFlowRate:                         {sapiIPPool, 100},
	getCrossMarginAccountDetailRate:               {sapiIPPool, 10},
	allCrossMarginFeeDataRate:                     {sapiIPPool, 5},
	allIsolatedMarginFeeDataRate:                  {sapiIPPool, 10},
	marginHourlyInterestRate:                      {sapiIPPool, 100},
	borrowRepayRecordsInMarginAccountRate:         {sapiIPPool, 10},
	marginAccountBorrowRepayRate:                  {sapiUIDPool, 1500},
	marginMaxBorrowRate:                           {sapiUIDPool, 750},
	crossMarginCollateralRatioRate:                {sapiIPPool, 100},
	allIsolatedMarginSymbolsRate:                  {sapiIPPool, 10},
	marginTokensAndSymbolsDelistScheduleRate:      {sapiIPPool, 100},
	marginAvailableInventoryRate:                  {sapiUIDPool, 50},
	getPriceMarginIndexRate:                       {sapiIPPool, 10},
	getSmallLiabilityExchangeCoinListRate:         {sapiIPPool, 100},
	marginSmallLiabilityExchangeRate:              {sapiUIDPool, 3000},
	smallLiabilityExchangeHistoryRate:             {sapiUIDPool, 100},
	getMarginAccountsOpenOrdersRate:               {sapiIPPool, 10},
	getMarginAccountOneCancelsTheOtherOrderRate:   {sapiIPPool, 10},
	getMarginAccountOrderRate:                     {sapiIPPool, 10},
	marginAccountNewOrderRate:                     {sapiUIDPool, 6},
	marginAccountNewOrderBorrowRate:               {sapiUIDPool, 1500},
	marginAccountCancelOrderRate:                  {sapiIPPool, 10},
	marginManualLiquidationRate:                   {sapiUIDPool, 3000},
	marginCurrentOrderCountUsageRate:              {sapiIPPool, 20},
	getMarginAccountAllOneCancelsTheOtherRate:     {sapiIPPool, 200},
	marginAccountsAllOrdersRate:                   {sapiIPPool, 200},
	marginAccountOpenOneCancelsTheOtherOrdersRate: {sapiIPPool, 10},
	marginAccountTradeListRate:                    {sapiIPPool, 10},
	maxTransferOutRate:                            {sapiIPPool, 50},

	// Wallet
	dailyAccountSnapshotRate:               {sapiIPPool, 2400},
	assetDividendRecordRate:                {sapiIPPool, 10},
	dustTransferRate:                       {sapiUIDPool, 10},
	cloudMiningPaymentAndRefundHistoryRate: {sapiUIDPool, 600},
	getUserDelegationHistoryRate:           {sapiIPPool, 60},
	userUniversalTransferRate:              {sapiUIDPool, 300},
	getUserWalletBalanceRate:               {sapiIPPool, 60},
	userAssetsRate:                         {sapiIPPool, 5},
	allCoinInfoRate:                        {sapiIPPool, 10},
	depositAddressesRate:                   {sapiIPPool, 10},
	getDepositAddressListInNetworkRate:     {sapiIPPool, 10},
	withdrawAddressListRate:                {sapiIPPool, 10},
	fundWithdrawalRate:                     {sapiUIDPool, 900},
	withdrawalHistoryRate:                  {withdrawalHistoryPool, 1},
	symbolDelistScheduleForSpotRate:        {sapiIPPool, 100},
	depositQuestionnaireRate:               {sapiUIDPool, 600},

	// Sub-account
	getFuturesPositionRiskOfSubAccountV1Rate: {sapiIPPool, 10},
	getSubAccountStatusOnMarginOrFuturesRate: {sapiIPPool, 10},
	getSubAccountTransactionStatisticsRate:   {sapiIPPool, 60},
	addIPRestrictionSubAccountAPIKeyRate:     {sapiUIDPool, 3000},
	deleteIPListForSubAccountAPIKeyRate:      {sapiUIDPool, 3000},
	ipRestrictionForSubAccountAPIKeyRate:     {sapiUIDPool, 3000},
	getDetailSubAccountFuturesAccountRate:    {sapiIPPool, 10},
	subAccountMarginAccountDetailRate:        {sapiIPPool, 10},
	getFuturesSubAccountSummaryV2Rate:        {sapiIPPool, 10},
	getSubAccountSummaryOfMarginAccountRate:  {sapiIPPool, 10},
	getV3SubAccountAssetsRate:                {sapiUIDPool, 60},
	getSubAccountAssetRate:                   {sapiUIDPool, 60},
	subAccountFuturesAssetTransferRate:       {subAccountFuturesTransferPool, 1},
	universalTransferForMasterAccountRate:    {sapiUIDPool, 360},
	managedSubAccountFuturesAssetDetailRate:  {sapiUIDPool, 60},
	getManagedSubAccountListRate:             {sapiUIDPool, 60},
	getManagedSubAccountSnapshotRate:         {sapiIPPool, 2400},
	managedSubAccountTransferLogRate:         {sapiUIDPool, 60},

	// Portfolio Margin Pro, formerly classic portfolio margin
	transferBNBRate:                                          {portfolioMarginProBNBTransferPool, 1},
	getAutoRepayFuturesStatusRate:                            {sapiIPPool, 30},
	changeAutoRepayFuturesStatusRate:                         {sapiIPPool, 1500},
	fundAutoCollectionRate:                                   {sapiIPPool, 1500},
	fundCollectionByAssetRate:                                {sapiIPPool, 60},
	classicPortfolioMarginAccountInfoRate:                    {sapiUIDPool, 5},
	repayClassicPortfolioMarginBankruptcyLoanRate:            {sapiUIDPool, 3000},
	getClassicPortfolioMarginBankruptcyLoanAmountRate:        {sapiUIDPool, 500},
	classicPortfolioMarginNegativeBalanceInterestHistoryRate: {sapiIPPool, 50},
	repayFuturesNegativeBalanceRate:                          {sapiIPPool, 1500},
	portfolioMarginAssetLeverageRate:                         {sapiIPPool, 50},
	classicPortfolioMarginCollateralRate:                     {sapiIPPool, 50},
	portfolioMarginAssetIndexPriceRate:                       {sapiIPPool, 50},

	// Simple Earn
	personalLeftQuotaRate:                    {sapiIPPool, 150},
	getFlexibleSimpleEarnProductPositionRate: {sapiIPPool, 150},
	getRedemptionRecordRate:                  {sapiIPPool, 150},
	getRewardHistoryRate:                     {sapiIPPool, 150},
	subscriptionPreviewRate:                  {sapiIPPool, 150},
	getFlexibleSubscriptionRecordRate:        {sapiIPPool, 150},
	getSimpleEarnProductPositionRate:         {sapiIPPool, 150},
	getLockedSubscriptionRecordsRate:         {sapiIPPool, 150},
	simpleEarnRateHistoryRate:                {sapiIPPool, 150},
	simpleEarnProductsRate:                   {sapiIPPool, 150},
	setAutoSubscribeRate:                     {sapiIPPool, 150},
	setRedeemOptionRate:                      {sapiIPPool, 50},
	simpleAccountRate:                        {sapiIPPool, 150},

	// Staking
	ethStakingAccountRate:             {sapiIPPool, 150},
	bethRewardDistributionHistoryRate: {sapiIPPool, 150},
	currentETHStakingQuotaRate:        {sapiIPPool, 150},
	ethRedemptionHistoryRate:          {sapiIPPool, 150},
	ethStakingHistoryRate:             {sapiIPPool, 150},
	getWBETHRateHistoryRate:           {sapiIPPool, 150},
	wbethRewardsHistoryRate:           {sapiIPPool, 150},
	wbethWrapOrUnwrapHistoryRate:      {sapiIPPool, 150},
	ethereumStakingRedemptionRate:     {sapiIPPool, 150},
	subscribeETHStakingRate:           {sapiIPPool, 150},
	wrapBETHRate:                      {sapiIPPool, 150},
	claimBoostRewardsRate:             {sapiIPPool, 150},
	bnsolRateHistoryRate:              {sapiIPPool, 150},
	bnsolRewardsHistoryRate:           {sapiIPPool, 150},
	boostRewardsHistoryRate:           {sapiIPPool, 150},
	solRedemptionHistoryRate:          {sapiIPPool, 150},
	solStakingHistoryRate:             {sapiIPPool, 150},
	solStakingQuotaDetailsRate:        {sapiIPPool, 150},
	unclaimedRewardsRate:              {sapiIPPool, 150},
	redeemSOLRate:                     {sapiIPPool, 150},
	solStakingAccountRate:             {sapiIPPool, 150},
	subscribeSOLStakingRate:           {sapiIPPool, 150},

	// Auto-Invest, no longer documented
	autoInvestPlanCreationRate:       {autoInvestPlanCreationPool, 1},
	autoInvestPlanAdjustmentRate:     {autoInvestPlanAdjustmentPool, 1},
	autoInvestPlanStatusRate:         {autoInvestPlanStatusPool, 1},
	autoInvestPlanListRate:           {autoInvestPlanListPool, 1},
	autoInvestOneTimeTransactionRate: {autoInvestOneTimeTransactionPool, 1},
	autoInvestRedemptionRate:         {autoInvestRedemptionPool, 1},

	// Crypto Loan
	checkCollateralRepayRate:             {sapiIPPool, 6000},
	adjustFlexibleLoanRate:               {sapiUIDPool, 6000},
	borrowFlexibleRate:                   {sapiIPPool, 6000},
	repayFlexibleLoanRate:                {sapiIPPool, 6000},
	flexibleLoanAssetDataRate:            {sapiIPPool, 400},
	flexibleBorrowHistoryRate:            {sapiIPPool, 400},
	flexibleLoanCollateralAssetRate:      {sapiIPPool, 400},
	flexibleLoanLiquidationHistoryRate:   {sapiIPPool, 400},
	flexibleLoanLTVAdjustmentHistoryRate: {flexibleLoanLTVAdjustmentHistoryPool, 1},
	getFlexibleLoanOngoingOrdersRate:     {sapiIPPool, 300},
	flexibleLoanRepaymentHistoryRate:     {sapiIPPool, 400},
	cryptoLoansIncomeHistoryRate:         {sapiUIDPool, 6000},
	getLoanBorrowHistoryRate:             {sapiIPPool, 400},
	getLoanLTVAdjustmentHistoryRate:      {sapiIPPool, 400},
	repaymentHistoryRate:                 {sapiIPPool, 400},

	// VIP Loan
	getVIPBorrowInterestRate:                 {sapiIPPool, 400},
	getCollateralAssetDataRate:               {sapiIPPool, 400},
	getVIPLoanableAssetsRate:                 {sapiIPPool, 400},
	vipLoanInterestRateHistoryRate:           {sapiIPPool, 400},
	vipLoanBorrowRate:                        {sapiUIDPool, 6000},
	vipLoanRenewRate:                         {sapiUIDPool, 6000},
	vipLoanRepayRate:                         {sapiUIDPool, 6000},
	checkLockedValueVIPCollateralAccountRate: {sapiIPPool, 6000},
	getVIPLoanAccruedInterestRate:            {sapiIPPool, 400},
	getVIPLoanOngoingOrdersRate:              {sapiIPPool, 400},
	getVIPLoanRepaymentHistoryRate:           {sapiIPPool, 400},
	getApplicationStatusRate:                 {vipLoanApplicationStatusPool, 1},

	// Convert
	getAllConvertPairsRate:                {sapiIPPool, 3000},
	getOrderQuantityPrecisionPerAssetRate: {sapiIPPool, 100},
	acceptQuoteRate:                       {sapiUIDPool, 500},
	cancelLimitOrderRate:                  {sapiUIDPool, 200},
	convertTradeFlowHistoryRate:           {sapiUIDPool, 3000},
	convertOrderStatusRate:                {sapiUIDPool, 100},
	placeLimitOrderRate:                   {sapiUIDPool, 500},
	getLimitOpenOrdersRate:                {sapiUIDPool, 3000},
	sendQuoteRequestRate:                  {sapiUIDPool, 200},

	// Algo
	placeTWAveragePriceNewOrderRate: {sapiUIDPool, 3000},
	placeVPOrderRate:                {sapiUIDPool, 300},
	spotTWAPNewOrderRate:            {sapiUIDPool, 3000},

	// Fiat, Pay, Rebate and NFT
	fiatDepositWithdrawHistRate: {sapiUIDPool, 45000},
	payTradeHistoryRate:         {sapiUIDPool, 3000},
	spotRebateHistoryRate:       {sapiUIDPool, 12000},
	nftRate:                     {sapiUIDPool, 3000},

	// Binance Link (CAAS)
	createAPIKeyForSubAccountRate:              {brokerCreateAPIKeyPool, 1},
	deleteAPIKeyForSubAccountRate:              {brokerDeleteAPIKeyPool, 1},
	updateIPRestrictionForSubAccountAPIKeyRate: {sapiUIDPool, 3000},
	brokerSubAccountDepositHistoryRate:         {sapiIPPool, 10},
	brokerSubAccountFuturesAssetInfoRate:       {sapiUIDPool, 60},
	brokerSubAccountSpotAssetInfoRate:          {sapiUIDPool, 3000},
	brokerFuturesTransferRate:                  {brokerFuturesTransferPool, 1},
	brokerSubAccountCommissionRate:             {sapiUIDPool, 4},

	// USDⓈ-M futures /fapi
	usdtMarginedFuturesDefaultRate:                         {derivativesIPPool, 1},
	usdtMarginedFuturesCancelRate:                          {derivativesCancelPool, 1},
	usdtMarginedFuturesCancelAllListingRate:                {derivativesCancelPool, 40},
	usdtMarginedFuturesAccountInformationRate:              {derivativesIPPool, 5},
	usdtMarginedFuturesTradingStatusAllRate:                {derivativesIPPool, 10},
	usdtMarginedFuturesMultiAssetMarginRate:                {derivativesIPPool, 30},
	usdtMarginedFuturesPositionModeRate:                    {derivativesIPPool, 30},
	usdtMarginedFuturesPositionModeLookupRate:              {derivativesCancelPool, 30},
	usdtMarginedFuturesDownloadIDRate:                      {derivativesIPPool, 1000},
	usdtMarginedFuturesDownloadLinkRate:                    {derivativesIPPool, 10},
	usdtMarginedFuturesIncomeHistoryRate:                   {derivativesIPPool, 30},
	usdtMarginedFuturesUserCommissionRate:                  {derivativesIPPool, 20},
	usdtMarginedFuturesAggregateTradesRate:                 {derivativesIPPool, 20},
	usdtMarginedFuturesFundingRateHistoryRate:              {usdtMarginedFuturesFundingPool, 1},
	usdtMarginedFuturesFundingInfoRate:                     {usdtMarginedFuturesFundingPool, 1},
	usdtMarginedFuturesKline100Rate:                        {derivativesIPPool, 1},
	usdtMarginedFuturesKline500Rate:                        {derivativesIPPool, 2},
	usdtMarginedFuturesKline1000Rate:                       {derivativesIPPool, 5},
	usdtMarginedFuturesKlineMaxRate:                        {derivativesIPPool, 10},
	usdtMarginedFuturesMarkPriceAllRate:                    {derivativesIPPool, 10},
	usdtMarginedFuturesAssetIndexAllRate:                   {derivativesIPPool, 10},
	usdtMarginedFuturesHistoricalTradesRate:                {derivativesIPPool, 200},
	usdtMarginedFuturesOrderbook50Rate:                     {derivativesIPPool, 2},
	usdtMarginedFuturesOrderbook100Rate:                    {derivativesIPPool, 5},
	usdtMarginedFuturesOrderbook500Rate:                    {derivativesIPPool, 10},
	usdtMarginedFuturesOrderbook1000Rate:                   {derivativesIPPool, 20},
	usdtMarginedFuturesIndexConstituentsRate:               {derivativesIPPool, 2},
	usdtMarginedFuturesRecentTradesRate:                    {derivativesIPPool, 5},
	usdtMarginedFuturesRetailPriceImprovementOrderbookRate: {derivativesIPPool, 20},
	usdtMarginedFuturesBookTickerRate:                      {derivativesIPPool, 2},
	usdtMarginedFuturesOrderbookTickerAllRate:              {derivativesIPPool, 5},
	usdtMarginedFuturesSymbolPriceAllRate:                  {derivativesIPPool, 2},
	usdtMarginedFuturesTicker24HourAllRate:                 {derivativesIPPool, 40},
	usdtMarginedFuturesAccountTradeListRate:                {derivativesIPPool, 5},
	usdtMarginedFuturesGetAllOrdersRate:                    {derivativesIPPool, 5},
	usdtMarginedFuturesCountdownCancelRate:                 {derivativesHeartbeatPool, 10},
	usdtMarginedFuturesHeartbeatOverflowRate:               {derivativesIPPool, 10},
	usdtMarginedFuturesOrdersDefaultRate:                   {derivativesOrderPool, 1},
	usdtMarginedFuturesBatchOrdersRate:                     {derivativesOrderPool, 5},
	usdtMarginedFuturesOpenAlgoOrdersAllRate:               {derivativesIPPool, 40},
	usdtMarginedFuturesGetAllOpenOrdersRate:                {derivativesIPPool, 40},
	usdtMarginedFuturesAutoDeleveragingQuantileRate:        {derivativesIPPool, 5},
	usdtMarginedFuturesPositionRiskRate:                    {derivativesIPPool, 5},
	usdtMarginedFuturesAllAlgoOrdersRate:                   {derivativesIPPool, 5},
	usdtMarginedFuturesSymbolForceOrdersRate:               {derivativesIPPool, 20},
	usdtMarginedFuturesAllForceOrdersRate:                  {derivativesIPPool, 50},
	usdtMarginedFuturesAPIReferralRate:                     {derivativesIPPool, 100},

	// /futures/data on the USDⓈ-M and COIN-M hosts
	futuresDataRate: {futuresDataPool, 1},

	// COIN-M futures /dapi
	coinMarginedFuturesDefaultRate:                  {derivativesIPPool, 1},
	coinMarginedFuturesCancelRate:                   {derivativesCancelPool, 1},
	coinMarginedFuturesCancelAllListingRate:         {derivativesCancelPool, 40},
	coinMarginedFuturesAccountInformationRate:       {derivativesIPPool, 5},
	coinMarginedFuturesIncomeHistoryRate:            {derivativesIPPool, 20},
	coinMarginedFuturesAggregateTradesRate:          {derivativesIPPool, 20},
	coinMarginedFuturesKline100Rate:                 {derivativesIPPool, 1},
	coinMarginedFuturesKline500Rate:                 {derivativesIPPool, 2},
	coinMarginedFuturesKline1000Rate:                {derivativesIPPool, 5},
	coinMarginedFuturesKlineMaxRate:                 {derivativesIPPool, 10},
	coinMarginedFuturesIndexMarkPriceRate:           {derivativesIPPool, 10},
	coinMarginedFuturesHistoricalTradesRate:         {derivativesIPPool, 200},
	coinMarginedFuturesOrderbook50Rate:              {derivativesIPPool, 2},
	coinMarginedFuturesOrderbook100Rate:             {derivativesIPPool, 5},
	coinMarginedFuturesOrderbook500Rate:             {derivativesIPPool, 10},
	coinMarginedFuturesOrderbook1000Rate:            {derivativesIPPool, 20},
	coinMarginedFuturesRecentTradesRate:             {derivativesIPPool, 5},
	coinMarginedFuturesBookTickerRate:               {derivativesIPPool, 2},
	coinMarginedFuturesOrderbookTickerAllRate:       {derivativesIPPool, 5},
	coinMarginedFuturesSymbolPriceAllRate:           {derivativesIPPool, 2},
	coinMarginedFuturesTicker24HourAllRate:          {derivativesIPPool, 40},
	coinMarginedFuturesAccountTradeListRate:         {derivativesIPPool, 5},
	coinMarginedFuturesAllOrdersRate:                {derivativesIPPool, 5},
	coinMarginedFuturesCountdownCancelRate:          {derivativesHeartbeatPool, 10},
	coinMarginedFuturesHeartbeatOverflowRate:        {derivativesIPPool, 10},
	coinMarginedFuturesBatchOrdersRate:              {derivativesOrderPool, 5},
	coinMarginedFuturesOrdersDefaultRate:            {derivativesOrderPool, 1},
	coinMarginedFuturesGetAllOpenOrdersRate:         {derivativesIPPool, 40},
	coinMarginedFuturesOpenAlgoOrdersAllRate:        {derivativesIPPool, 40},
	coinMarginedFuturesAllAlgoOrdersRate:            {derivativesIPPool, 5},
	coinMarginedFuturesAutoDeleveragingQuantileRate: {derivativesIPPool, 5},
	coinMarginedFuturesSymbolForceOrdersRate:        {derivativesIPPool, 20},
	coinMarginedFuturesAllForceOrdersRate:           {derivativesIPPool, 50},

	// Options /eapi
	optionsDefaultRate:                               {optionsIPPool, 1},
	optionsCancelRate:                                {optionsCancelPool, 1},
	optionsCancelAllListingRate:                      {optionsCancelPool, 40},
	optionsMarginAccountInfoRate:                     {optionsIPPool, 3},
	optionsDownloadIDForOptionTransactionHistoryRate: {optionsIPPool, 5},
	optionsGetTransHistoryDownloadLinkByIDRate:       {optionsIPPool, 5},
	optionsHistoricalExerciseRecordsRate:             {optionsIPPool, 3},
	optionsMarkPriceRate:                             {optionsIPPool, 5},
	optionsOrderbook50Rate:                           {optionsIPPool, 2}, // documented as 1, but charged 2
	optionsOrderbook100Rate:                          {optionsIPPool, 5},
	optionsOrderbook500Rate:                          {optionsIPPool, 10},
	optionsOrderbook1000Rate:                         {optionsIPPool, 20},
	optionsRecentTradesRate:                          {optionsIPPool, 5},
	optionsAllTickerPriceStatisticsRate:              {optionsIPPool, 40},
	optionsAutoCancelAllOpenOrdersHeartbeatRate:      {optionsHeartbeatPool, 10},
	optionsHeartbeatOverflowRate:                     {optionsIPPool, 10},
	optionsAutoCancelAllOpenOrdersConfigRate:         {optionsHeartbeatPool, 1},
	optionsAccountTradeListRate:                      {optionsIPPool, 5},
	optionsCancelAllByUnderlyingRate:                 {optionsCancelPool, 5},
	optionsBatchOrderRate:                            {optionsOrderPool, 5},
	optionsCancelBatchOrdersRate:                     {optionsCancelPool, 5},
	optionsDefaultOrderRate:                          {optionsOrderPool, 1},
	optionsPositionInformationRate:                   {optionsIPPool, 5},
	optionsAllQueryOpenOrdersRate:                    {optionsIPPool, 40},
	optionsGetOrderHistoryRate:                       {optionsIPPool, 3},
	optionsUserExerciseRecordRate:                    {optionsIPPool, 5},

	// Portfolio margin /papi
	portfolioMarginDefaultRate:                                                   {portfolioMarginIPPool, 1},
	portfolioMarginGetAccountBalancesRate:                                        {portfolioMarginIPPool, 20},
	portfolioMarginGetAccountInformationRate:                                     {portfolioMarginIPPool, 20},
	portfolioMarginBNBTransferRate:                                               {portfolioMarginBNBTransferPool, 1},
	portfolioMarginGetAutoRepayFuturesStatusRate:                                 {portfolioMarginIPPool, 30},
	portfolioMarginChangeAutoRepayFuturesStatusRate:                              {portfolioMarginIPPool, 750},
	portfolioMarginGetCoinMarginedCurrentPositionModeRate:                        {portfolioMarginIPPool, 30},
	portfolioMarginGetUSDTMarginedCurrentPositionModeRate:                        {portfolioMarginIPPool, 30},
	portfolioMarginFundAutoCollectionRate:                                        {portfolioMarginIPPool, 750},
	portfolioMarginFundCollectionByAssetRate:                                     {portfolioMarginIPPool, 30},
	portfolioMarginGetCoinMarginedAccountDetailRate:                              {portfolioMarginIPPool, 5},
	portfolioMarginGetCoinMarginedIncomeHistoryRate:                              {portfolioMarginIPPool, 30},
	portfolioMarginGetUSDTMarginedAccountDetailRate:                              {portfolioMarginIPPool, 5},
	portfolioMarginGetUSDTMarginedIncomeHistoryRate:                              {portfolioMarginIPPool, 30},
	portfolioMarginGetCoinMarginedUserCommissionRate:                             {portfolioMarginIPPool, 20},
	portfolioMarginGetUSDTMarginedUserCommissionRate:                             {portfolioMarginIPPool, 20},
	portfolioMarginMarginMaxBorrowRate:                                           {portfolioMarginIPPool, 5},
	portfolioMarginUSDTMarginedTradingQuantitativeRulesIndicatorsRate:            {portfolioMarginIPPool, 10},
	portfolioMarginGetMarginLoanRecordRate:                                       {portfolioMarginIPPool, 10},
	portfolioMarginGetMarginMaxWithdrawalRate:                                    {portfolioMarginIPPool, 5},
	portfolioMarginGetMarginRepayRecordRate:                                      {portfolioMarginIPPool, 10},
	portfolioMarginGetNegativeBalanceInterestHistoryRate:                         {portfolioMarginIPPool, 50},
	portfolioMarginGetUSDTMarginedPositionInformationRate:                        {portfolioMarginIPPool, 5},
	portfolioMarginNegativeBalanceExchangeRecordRate:                             {portfolioMarginIPPool, 100},
	portfolioMarginRepayFuturesNegativeBalanceRate:                               {portfolioMarginIPPool, 750},
	portfolioMarginOrderRate:                                                     {portfolioMarginOrderPool, 1},
	portfolioMarginCancelMarginAccountOpenOrdersOnSymbolRate:                     {portfolioMarginIPPool, 5},
	portfolioMarginGetMarginAccountOneCancelsTheOtherRate:                        {portfolioMarginIPPool, 5},
	portfolioMarginCancelMarginAccountOneCancelsTheOtherRate:                     {portfolioMarginIPPool, 2},
	portfolioMarginGetMarginAccountOrderRate:                                     {portfolioMarginIPPool, 10},
	portfolioMarginCancelMarginAccountOrderRate:                                  {portfolioMarginIPPool, 2},
	portfolioMarginGetCoinMarginedAccountTradeListWithSymbolRate:                 {portfolioMarginIPPool, 20},
	portfolioMarginGetCoinMarginedAccountTradeListWithPairRate:                   {portfolioMarginIPPool, 40},
	portfolioMarginGetCoinMarginedPositionAutoDeleveragingQuantileEstimationRate: {portfolioMarginIPPool, 5},
	portfolioMarginMarginAccountLoanAndRepayRate:                                 {portfolioMarginIPPool, 100},
	portfolioMarginOneCancelsTheOtherOrderRate:                                   {portfolioMarginOrderPool, 2},
	portfolioMarginGetMarginAccountTradeListRate:                                 {portfolioMarginIPPool, 5},
	portfolioMarginAllCoinMarginedConditionalOrderWithoutSymbolRate:              {portfolioMarginIPPool, 40},
	portfolioMarginAllCoinMarginedOrderWithSymbolRate:                            {portfolioMarginIPPool, 20},
	portfolioMarginAllCoinMarginedOrderWithPairRate:                              {portfolioMarginIPPool, 40},
	portfolioMarginAllCoinMarginedOpenConditionalOrdersWithoutSymbolRate:         {portfolioMarginIPPool, 40},
	portfolioMarginRetrieveAllCoinMarginedOpenOrdersForAllSymbolRate:             {portfolioMarginIPPool, 40},
	portfolioMarginRetrieveAllUSDTMarginedOpenOrdersForAllSymbolRate:             {portfolioMarginIPPool, 40},
	portfolioMarginAllMarginAccountOrdersRate:                                    {portfolioMarginIPPool, 100},
	portfolioMarginGetAllUSDTMarginedOrdersRate:                                  {portfolioMarginIPPool, 5},
	portfolioMarginGetUSDTMarginedAlgoOrderHistoryRate:                           {portfolioMarginIPPool, 5},
	portfolioMarginCurrentMarginOpenOrderRate:                                    {portfolioMarginIPPool, 5},
	portfolioMarginGetMarginAccountsAllOneCancelsTheOtherOrdersRate:              {portfolioMarginIPPool, 100},
	portfolioMarginGetMarginAccountsOpenOneCancelsTheOtherOrdersRate:             {portfolioMarginIPPool, 5},
	portfolioMarginGetUserCoinMarginedForceOrdersWithSymbolRate:                  {portfolioMarginIPPool, 20},
	portfolioMarginGetUserCoinMarginedForceOrdersWithoutSymbolRate:               {portfolioMarginIPPool, 50},
	portfolioMarginGetUserUSDTMarginedForceOrdersWithSymbolRate:                  {portfolioMarginIPPool, 20},
	portfolioMarginGetUserUSDTMarginedForceOrdersWithoutSymbolRate:               {portfolioMarginIPPool, 50},
	portfolioMarginGetUSDTMarginedAccountTradeListRate:                           {portfolioMarginIPPool, 5},
	portfolioMarginGetUSDTMarginedPositionAutoDeleveragingQuantileEstimationRate: {portfolioMarginIPPool, 5},
	portfolioMarginAPIReferralRate:                                               {portfolioMarginIPPool, 100},
}

// GetRateLimits returns Binance's endpoint limits, each with its limiter and weight
func GetRateLimits() request.RateLimitDefinitions {
	limiters := endpointLimiters(endpointCosts)
	definitions := make(request.RateLimitDefinitions, len(limiters)+1)
	for epl, limiter := range limiters {
		definitions[epl] = request.GetRateLimiterWithWeight(limiter, endpointCosts[epl].weight)
	}
	// A nil limiter makes a websocket connection fall back to its own
	definitions[wsConnectionMessageRate] = request.RateLimitNotRequired
	return definitions
}

// endpointLimiters returns the limiter of every endpoint limit in costs: its pool's shared limiter, or a limiter of its
// own when it weighs more than 1 on a pool whose endpoints Binance budgets separately
func endpointLimiters(costs map[request.EndpointLimit]endpointCost) map[request.EndpointLimit]*rate.Limiter {
	shared := make(map[rateLimitPool]*rate.Limiter, len(poolBudgets))
	limiters := make(map[request.EndpointLimit]*rate.Limiter, len(costs))
	for epl, cost := range costs {
		if cost.pool.perEndpoint() && cost.weight > 1 {
			limiters[epl] = newPoolLimiter(cost.pool)
			continue
		}
		if shared[cost.pool] == nil {
			shared[cost.pool] = newPoolLimiter(cost.pool)
		}
		limiters[epl] = shared[cost.pool]
	}
	return limiters
}

// sendOnShare sends a request drawing on share when the share can take it at once, and otherwise on whichever of share
// and overflow, the host's IP pool, admits it sooner, so a share keeps the request clear of queued market data without
// holding it longer than the IP pool would. A request the request package retries after it went out on the share can
// be sent again on overflow, so it suits requests Binance applies alike when repeated
func (e *Exchange) sendOnShare(ctx context.Context, share, overflow request.EndpointLimit, send func(context.Context, request.EndpointLimit) error) error {
	if err := send(request.WithDelayNotAllowed(ctx), share); !errors.Is(err, request.ErrDelayNotAllowed) {
		return err
	}
	definitions := e.Requester.GetRateLimiterDefinitions()
	if definitions[share].Delay() <= definitions[overflow].Delay() {
		return send(ctx, share)
	}
	return send(ctx, overflow)
}

// spotOrderbookLimit returns the limit of an order book request for its depth; 0 requests the default of 100
func spotOrderbookLimit(depth uint64) request.EndpointLimit {
	switch {
	case depth <= 100:
		return spotOrderbookDepth100Rate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	default:
		return spotOrderbookDepth5000Rate
	}
}

// spotTickerSymbolsLimit returns the limit of a rolling window or trading day ticker request, which costs 4 for each
// symbol and 200 from 50 symbols on. A limit has a fixed weight, so each tier charges for its largest symbol count
func spotTickerSymbolsLimit(symbols int) request.EndpointLimit {
	switch {
	case symbols <= 1:
		return spotTickerSymbols1Rate
	case symbols <= 5:
		return spotTickerSymbols5Rate
	case symbols <= 10:
		return spotTickerSymbols10Rate
	case symbols <= 25:
		return spotTickerSymbols25Rate
	default:
		return spotTickerSymbolsMaxRate
	}
}

// spotTicker24HourLimit returns the limit of a 24hr ticker request for its number of symbols; none requests every
// symbol
func spotTicker24HourLimit(symbols int) request.EndpointLimit {
	switch {
	case symbols == 0, symbols > 100:
		return spotPriceChangeAllRate
	case symbols > 20:
		return getTickers100Rate
	default:
		return getTickers20Rate
	}
}

// spotAccountTradeListLimit returns the limit of an account trade list query, which costs less for one order
func spotAccountTradeListLimit(orderID uint64) request.EndpointLimit {
	if orderID != 0 {
		return accountTradeListByOrderIDRate
	}
	return accountTradeListRate
}

// marginNewOrderLimit returns the limit of a new margin order or OCO, which costs 1500 instead of 6 when it borrows
func marginNewOrderLimit(sideEffectType string) request.EndpointLimit {
	if sideEffectType == "MARGIN_BUY" || sideEffectType == "AUTO_BORROW_REPAY" {
		return marginAccountNewOrderBorrowRate
	}
	return marginAccountNewOrderRate
}

// usdtMarginedFuturesKlineLimit returns the limit of a USDⓈ-M kline request for its limit; 0 requests the default of
// 500
func usdtMarginedFuturesKlineLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return usdtMarginedFuturesKline1000Rate
	case limit < 100:
		return usdtMarginedFuturesKline100Rate
	case limit < 500:
		return usdtMarginedFuturesKline500Rate
	case limit <= 1000:
		return usdtMarginedFuturesKline1000Rate
	default:
		return usdtMarginedFuturesKlineMaxRate
	}
}

// usdtMarginedFuturesOrderbookLimit returns the limit of a USDⓈ-M order book request for its limit; 0 requests the
// default of 500
func usdtMarginedFuturesOrderbookLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return usdtMarginedFuturesOrderbook500Rate
	case limit <= 50:
		return usdtMarginedFuturesOrderbook50Rate
	case limit <= 100:
		return usdtMarginedFuturesOrderbook100Rate
	case limit <= 500:
		return usdtMarginedFuturesOrderbook500Rate
	default:
		return usdtMarginedFuturesOrderbook1000Rate
	}
}

// coinMarginedFuturesKlineLimit returns the limit of a COIN-M kline request for its limit; 0 requests the default of
// 500
func coinMarginedFuturesKlineLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return coinMarginedFuturesKline1000Rate
	case limit < 100:
		return coinMarginedFuturesKline100Rate
	case limit < 500:
		return coinMarginedFuturesKline500Rate
	case limit <= 1000:
		return coinMarginedFuturesKline1000Rate
	default:
		return coinMarginedFuturesKlineMaxRate
	}
}

// coinMarginedFuturesOrderbookLimit returns the limit of a COIN-M order book request for its limit; 0 requests the
// default of 500
func coinMarginedFuturesOrderbookLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return coinMarginedFuturesOrderbook500Rate
	case limit <= 50:
		return coinMarginedFuturesOrderbook50Rate
	case limit <= 100:
		return coinMarginedFuturesOrderbook100Rate
	case limit <= 500:
		return coinMarginedFuturesOrderbook500Rate
	default:
		return coinMarginedFuturesOrderbook1000Rate
	}
}

// optionsOrderbookLimit returns the limit of an options order book request for its limit; 0 requests the default of
// 100
func optionsOrderbookLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return optionsOrderbook100Rate
	case limit <= 50:
		return optionsOrderbook50Rate
	case limit <= 100:
		return optionsOrderbook100Rate
	case limit <= 500:
		return optionsOrderbook500Rate
	default:
		return optionsOrderbook1000Rate
	}
}
