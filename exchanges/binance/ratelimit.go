package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// rateLimitPool is a Binance request budget. The endpoint limits drawing on a pool share its limiter, so their weights
// count against one budget the way Binance counts them, unless Binance budgets the pool's endpoints separately
type rateLimitPool uint8

// Rate limit pools: the shared budgets of each host, then dedicated budgets for endpoints whose own documented limit
// is stricter than their weight on a shared pool
const (
	spotIPPool rateLimitPool = iota
	spotOrderPool
	sapiIPPool
	sapiUIDPool
	derivativesIPPool
	derivativesOrderPool
	futuresDataPool
	uFuturesFundingPool
	optionsIPPool
	optionsOrderPool
	portfolioMarginIPPool
	portfolioMarginOrderPool

	marginMaxLeveragePool
	subAccountFuturesTransferPool
	pmProBNBTransferPool
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
	pmBNBTransferPool
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
// applies, and daily or monthly quotas are left to Binance: spread evenly they would hold a second call for hours
var poolBudgets = map[rateLimitPool]rateLimitBudget{
	spotIPPool:    {time.Minute, 6000},     // REQUEST_WEIGHT per IP; WebSocket API requests count against it too
	spotOrderPool: {10 * time.Second, 100}, // ORDERS (unfilled order count) per account; 200000 a day also applies
	// Binance gives each /sapi endpoint a budget of its own, so every endpoint limit heavier than 1 gets a limiter of
	// its own and a heavy call never holds up another endpoint. The weight-1 defaults, which most /sapi endpoints
	// use, share one limiter per pool: stricter, but still 200 IP-limited requests a second
	sapiIPPool:  {time.Minute, 12000},
	sapiUIDPool: {time.Minute, 180000},
	// USDⓈ-M and COIN-M share their budgets since the 2026-06-30 CM-UM integration. Orders are also limited to 300 per
	// 10 seconds, a looser sustained rate
	derivativesIPPool:        {time.Minute, 2400},
	derivativesOrderPool:     {time.Minute, 1200},
	futuresDataPool:          {5 * time.Minute, 1000}, // per IP, for the /futures/data endpoints of both hosts
	uFuturesFundingPool:      {5 * time.Minute, 500},  // per IP, shared by /fapi/v1/fundingRate and /fapi/v1/fundingInfo
	optionsIPPool:            {time.Minute, 400},
	optionsOrderPool:         {time.Minute, 100}, // 30 per 10 seconds also applies, a looser sustained rate
	portfolioMarginIPPool:    {time.Minute, 6000},
	portfolioMarginOrderPool: {time.Minute, 1200},

	marginMaxLeveragePool: {time.Minute, 1}, // POST /sapi/v1/margin/max-leverage
	// POST /sapi/v1/sub-account/futures/internalTransfer, per master account
	subAccountFuturesTransferPool: {time.Minute, 2000},
	pmProBNBTransferPool:          {10 * time.Minute, 2}, // POST /sapi/v1/portfolio/bnb-transfer
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
	brokerCreateAPIKeyPool:    {time.Second, 1},
	brokerDeleteAPIKeyPool:    {time.Second, 1},
	brokerFuturesTransferPool: {time.Minute, 5000},    // POST /sapi/v1/broker/transfer/futures, per master account
	pmBNBTransferPool:         {10 * time.Minute, 10}, // POST /papi/v1/bnb-transfer
}

// Endpoint limits, grouped by host and documentation section. They start after request.UnAuth, so a stray
// request.Unset, request.Auth or request.UnAuth finds no Binance limiter instead of silently using one. Weight-1
// endpoints share their pool's default limit, and an endpoint whose weight depends on a parameter has a limit per
// tier, picked by a helper below
const (
	// Spot /api, REST and WebSocket API
	spotDefaultRate request.EndpointLimit = request.UnAuth + 1 + iota
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
	spotOCOOrderRate
	testNewOrderWithCommissionRate
	getCommissionRate
	getAllOCOOrdersRate
	spotAllOrdersRate
	spotAccountInformationRate
	spotOpenOrdersSpecificRate
	spotOpenOrdersAllRate
	spotOrderQueryRate
	getOCOListRate
	getAllocationsRate
	preventedMatchesRate
	preventedMatchesByOrderIDRate
	accountTradeListRate
	accountTradeListByOrderIDRate
	getOpenOCOListRate
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
	getMarginAccountOCOOrderRate
	getMarginAccountOrderRate
	marginAccountNewOrderRate
	marginAccountNewOrderBorrowRate
	marginAccountCancelOrderRate
	marginManualLiquidationRate
	marginCurrentOrderCountUsageRate
	getMarginAccountAllOCORate
	marginAccountsAllOrdersRate
	marginAccountOpenOCOOrdersRate
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
	classicPMAccountInfoRate
	repayClassicPMBankruptcyLoanRate
	getClassicPMBankruptcyLoanAmountRate
	classicPMNegativeBalanceInterestHistoryRate
	repayFuturesNegativeBalanceRate
	pmAssetLeverageRate
	classicPMCollateralRate
	pmAssetIndexPriceRate

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
	uFuturesDefaultRate
	uFuturesAccountInformationRate
	uFuturesTradingStatusAllRate
	uFuturesMultiAssetMarginRate
	uFuturesPositionModeRate
	uFuturesDownloadIDRate
	uFuturesDownloadLinkRate
	uFuturesIncomeHistoryRate
	uFuturesUserCommissionRate
	uFuturesAggregateTradesRate
	uFuturesFundingRateHistoryRate
	uFuturesFundingInfoRate
	uFuturesKline100Rate
	uFuturesKline500Rate
	uFuturesKline1000Rate
	uFuturesKlineMaxRate
	uFuturesMarkPriceAllRate
	uFuturesAssetIndexAllRate
	uFuturesHistoricalTradesRate
	uFuturesOrderbook50Rate
	uFuturesOrderbook100Rate
	uFuturesOrderbook500Rate
	uFuturesOrderbook1000Rate
	uFuturesIndexConstituentsRate
	uFuturesRecentTradesRate
	uFuturesRPIOrderbookRate
	uFuturesBookTickerRate
	uFuturesOrderbookTickerAllRate
	uFuturesSymbolPriceAllRate
	uFuturesTicker24HourAllRate
	uFuturesAccountTradeListRate
	uFuturesGetAllOrdersRate
	uFuturesCountdownCancelRate
	uFuturesOrdersDefaultRate
	uFuturesBatchOrdersRate
	uFuturesOpenAlgoOrdersAllRate
	uFuturesGetAllOpenOrdersRate
	uFuturesADLQuantileRate
	uFuturesPositionRiskRate
	uFuturesAllAlgoOrdersRate
	uFuturesSymbolForceOrdersRate
	uFuturesAllForceOrdersRate
	uFuturesAPIReferralRate

	// /futures/data on the USDⓈ-M and COIN-M hosts
	futuresDataRate

	// COIN-M futures /dapi
	cFuturesDefaultRate
	cFuturesAccountInformationRate
	cFuturesIncomeHistoryRate
	cFuturesAggregateTradesRate
	cFuturesKline100Rate
	cFuturesKline500Rate
	cFuturesKline1000Rate
	cFuturesKlineMaxRate
	cFuturesIndexMarkPriceRate
	cFuturesHistoricalTradesRate
	cFuturesOrderbook50Rate
	cFuturesOrderbook100Rate
	cFuturesOrderbook500Rate
	cFuturesOrderbook1000Rate
	cFuturesRecentTradesRate
	cFuturesBookTickerRate
	cFuturesOrderbookTickerAllRate
	cFuturesSymbolPriceAllRate
	cFuturesTicker24HourAllRate
	cFuturesAccountTradeListRate
	cFuturesAllOrdersRate
	cFuturesCountdownCancelRate
	cFuturesBatchOrdersRate
	cFuturesOrdersDefaultRate
	cFuturesGetAllOpenOrdersRate
	cFuturesADLQuantileRate
	cFuturesSymbolForceOrdersRate
	cFuturesAllForceOrdersRate

	// Options /eapi
	optionsDefaultRate
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
	pmDefaultRate
	pmGetAccountBalancesRate
	pmGetAccountInformationRate
	pmBNBTransferRate
	pmGetAutoRepayFuturesStatusRate
	pmChangeAutoRepayFuturesStatusRate
	pmGetCMCurrentPositionModeRate
	pmGetUMCurrentPositionModeRate
	pmFundAutoCollectionRate
	pmFundCollectionByAssetRate
	pmGetCMAccountDetailRate
	pmGetCMIncomeHistoryRate
	pmGetUMAccountDetailRate
	pmGetUMIncomeHistoryRate
	pmGetCMUserCommissionRate
	pmGetUMUserCommissionRate
	pmMarginMaxBorrowRate
	pmUMTradingQuantitativeRulesIndicatorsRate
	pmGetMarginLoanRecordRate
	pmGetMarginMaxWithdrawalRate
	pmGetMarginRepayRecordRate
	pmGetPortfolioMarginNegativeBalanceInterestHistoryRate
	pmGetUMPositionInformationRate
	pmNegativeBalanceExchangeRecordRate
	pmRepayFuturesNegativeBalanceRate
	pmOrderRate
	pmCancelMarginAccountOpenOrdersOnSymbolRate
	pmGetMarginAccountOCORate
	pmCancelMarginAccountOCORate
	pmGetMarginAccountOrderRate
	pmCancelMarginAccountOrderRate
	pmGetCMAccountTradeListWithSymbolRate
	pmGetCMAccountTradeListWithPairRate
	pmGetCMPositionADLQuantileEstimationRate
	pmMarginAccountLoanAndRepayRate
	pmOCOOrderRate
	pmGetMarginAccountTradeListRate
	pmAllCMConditionalOrderWithoutSymbolRate
	pmAllCMOrderWithSymbolRate
	pmAllCMOrderWithPairRate
	pmAllCMOpenConditionalOrdersWithoutSymbolRate
	pmRetrieveAllCMOpenOrdersForAllSymbolRate
	pmRetrieveAllUMOpenOrdersForAllSymbolRate
	pmAllMarginAccountOrdersRate
	pmGetAllUMOrdersRate
	pmGetUMAlgoOrderHistoryRate
	pmCurrentMarginOpenOrderRate
	pmGetMarginAccountsAllOCOOrdersRate
	pmGetMarginAccountsOpenOCOOrdersRate
	pmGetUserCMForceOrdersWithSymbolRate
	pmGetUserCMForceOrdersWithoutSymbolRate
	pmGetUserUMForceOrdersWithSymbolRate
	pmGetUserUMForceOrdersWithoutSymbolRate
	pmGetUMAccountTradeListRate
	pmGetUMPositionADLQuantileEstimationRate
	pmAPIReferralRate

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
	spotDefaultRate:                {spotIPPool, 1},
	spotExchangeInfoRate:           {spotIPPool, 20},
	aggTradesRate:                  {spotIPPool, 4},
	getCurrentAveragePriceRate:     {spotIPPool, 2},
	spotOrderbookDepth100Rate:      {spotIPPool, 5},
	spotOrderbookDepth500Rate:      {spotIPPool, 25},
	spotOrderbookDepth1000Rate:     {spotIPPool, 50},
	spotOrderbookDepth5000Rate:     {spotIPPool, 250},
	getRecentTradesListRate:        {spotIPPool, 25},
	getOldTradeLookupRate:          {spotIPPool, 25},
	getKlineRate:                   {spotIPPool, 2},
	spotTickerSymbols1Rate:         {spotIPPool, 4},
	spotTickerSymbols5Rate:         {spotIPPool, 20},
	spotTickerSymbols10Rate:        {spotIPPool, 40},
	spotTickerSymbols25Rate:        {spotIPPool, 100},
	spotTickerSymbolsMaxRate:       {spotIPPool, 200},
	getTickers20Rate:               {spotIPPool, 2},
	getTickers100Rate:              {spotIPPool, 40},
	spotPriceChangeAllRate:         {spotIPPool, 80},
	spotBookTickerRate:             {spotIPPool, 2},
	spotOrderbookTickerAllRate:     {spotIPPool, 4},
	spotSymbolPriceRate:            {spotIPPool, 2},
	spotSymbolPriceAllRate:         {spotIPPool, 4},
	spotOrderRate:                  {spotOrderPool, 1},
	spotOCOOrderRate:               {spotOrderPool, 2},
	testNewOrderWithCommissionRate: {spotIPPool, 20},
	getCommissionRate:              {spotIPPool, 20},
	getAllOCOOrdersRate:            {spotIPPool, 20},
	spotAllOrdersRate:              {spotIPPool, 20},
	spotAccountInformationRate:     {spotIPPool, 20},
	spotOpenOrdersSpecificRate:     {spotIPPool, 6},
	spotOpenOrdersAllRate:          {spotIPPool, 80},
	spotOrderQueryRate:             {spotIPPool, 4},
	getOCOListRate:                 {spotIPPool, 4},
	getAllocationsRate:             {spotIPPool, 20},
	preventedMatchesRate:           {spotIPPool, 2},
	preventedMatchesByOrderIDRate:  {spotIPPool, 20},
	accountTradeListRate:           {spotIPPool, 20},
	accountTradeListByOrderIDRate:  {spotIPPool, 5},
	getOpenOCOListRate:             {spotIPPool, 6},
	currentOrderCountUsageRate:     {spotIPPool, 40},
	wsSessionRate:                  {spotIPPool, 2},
	userDataStreamSubscribeRate:    {spotIPPool, 2},

	// /sapi weight-1 defaults, used by every SAPI family
	sapiDefaultRate:    {sapiIPPool, 1},
	sapiUIDDefaultRate: {sapiUIDPool, 1},

	// Margin
	adjustCrossMarginMaxLeverageRate:         {marginMaxLeveragePool, 1},
	getIsolatedMarginAccountInfoRate:         {sapiIPPool, 10},
	enableIsolatedMarginAccountRate:          {sapiUIDPool, 300},
	disableIsolatedMarginAccountRate:         {sapiUIDPool, 300},
	marginAccountSummaryRate:                 {sapiIPPool, 10},
	marginCapitalFlowRate:                    {sapiIPPool, 100},
	getCrossMarginAccountDetailRate:          {sapiIPPool, 10},
	allCrossMarginFeeDataRate:                {sapiIPPool, 5},
	allIsolatedMarginFeeDataRate:             {sapiIPPool, 10},
	marginHourlyInterestRate:                 {sapiIPPool, 100},
	borrowRepayRecordsInMarginAccountRate:    {sapiIPPool, 10},
	marginAccountBorrowRepayRate:             {sapiUIDPool, 1500},
	marginMaxBorrowRate:                      {sapiUIDPool, 750},
	crossMarginCollateralRatioRate:           {sapiIPPool, 100},
	allIsolatedMarginSymbolsRate:             {sapiIPPool, 10},
	marginTokensAndSymbolsDelistScheduleRate: {sapiIPPool, 100},
	marginAvailableInventoryRate:             {sapiUIDPool, 50},
	getPriceMarginIndexRate:                  {sapiIPPool, 10},
	getSmallLiabilityExchangeCoinListRate:    {sapiIPPool, 100},
	marginSmallLiabilityExchangeRate:         {sapiUIDPool, 3000},
	smallLiabilityExchangeHistoryRate:        {sapiUIDPool, 100},
	getMarginAccountsOpenOrdersRate:          {sapiIPPool, 10},
	getMarginAccountOCOOrderRate:             {sapiIPPool, 10},
	getMarginAccountOrderRate:                {sapiIPPool, 10},
	marginAccountNewOrderRate:                {sapiUIDPool, 6},
	marginAccountNewOrderBorrowRate:          {sapiUIDPool, 1500},
	marginAccountCancelOrderRate:             {sapiIPPool, 10},
	marginManualLiquidationRate:              {sapiUIDPool, 3000},
	marginCurrentOrderCountUsageRate:         {sapiIPPool, 20},
	getMarginAccountAllOCORate:               {sapiIPPool, 200},
	marginAccountsAllOrdersRate:              {sapiIPPool, 200},
	marginAccountOpenOCOOrdersRate:           {sapiIPPool, 10},
	marginAccountTradeListRate:               {sapiIPPool, 10},
	maxTransferOutRate:                       {sapiIPPool, 50},

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
	withdrawalHistoryRate:                  {sapiUIDPool, 18000},
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
	transferBNBRate:                             {pmProBNBTransferPool, 1},
	getAutoRepayFuturesStatusRate:               {sapiIPPool, 30},
	changeAutoRepayFuturesStatusRate:            {sapiIPPool, 1500},
	fundAutoCollectionRate:                      {sapiIPPool, 1500},
	fundCollectionByAssetRate:                   {sapiIPPool, 60},
	classicPMAccountInfoRate:                    {sapiUIDPool, 5},
	repayClassicPMBankruptcyLoanRate:            {sapiUIDPool, 3000},
	getClassicPMBankruptcyLoanAmountRate:        {sapiUIDPool, 500},
	classicPMNegativeBalanceInterestHistoryRate: {sapiIPPool, 50},
	repayFuturesNegativeBalanceRate:             {sapiIPPool, 1500},
	pmAssetLeverageRate:                         {sapiIPPool, 50},
	classicPMCollateralRate:                     {sapiIPPool, 50},
	pmAssetIndexPriceRate:                       {sapiIPPool, 50},

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
	uFuturesDefaultRate:            {derivativesIPPool, 1},
	uFuturesAccountInformationRate: {derivativesIPPool, 5},
	uFuturesTradingStatusAllRate:   {derivativesIPPool, 10},
	uFuturesMultiAssetMarginRate:   {derivativesIPPool, 30},
	uFuturesPositionModeRate:       {derivativesIPPool, 30},
	uFuturesDownloadIDRate:         {derivativesIPPool, 1000},
	uFuturesDownloadLinkRate:       {derivativesIPPool, 10},
	uFuturesIncomeHistoryRate:      {derivativesIPPool, 30},
	uFuturesUserCommissionRate:     {derivativesIPPool, 20},
	uFuturesAggregateTradesRate:    {derivativesIPPool, 20},
	uFuturesFundingRateHistoryRate: {uFuturesFundingPool, 1},
	uFuturesFundingInfoRate:        {uFuturesFundingPool, 1},
	uFuturesKline100Rate:           {derivativesIPPool, 1},
	uFuturesKline500Rate:           {derivativesIPPool, 2},
	uFuturesKline1000Rate:          {derivativesIPPool, 5},
	uFuturesKlineMaxRate:           {derivativesIPPool, 10},
	uFuturesMarkPriceAllRate:       {derivativesIPPool, 10},
	uFuturesAssetIndexAllRate:      {derivativesIPPool, 10},
	uFuturesHistoricalTradesRate:   {derivativesIPPool, 200},
	uFuturesOrderbook50Rate:        {derivativesIPPool, 2},
	uFuturesOrderbook100Rate:       {derivativesIPPool, 5},
	uFuturesOrderbook500Rate:       {derivativesIPPool, 10},
	uFuturesOrderbook1000Rate:      {derivativesIPPool, 20},
	uFuturesIndexConstituentsRate:  {derivativesIPPool, 2},
	uFuturesRecentTradesRate:       {derivativesIPPool, 5},
	uFuturesRPIOrderbookRate:       {derivativesIPPool, 20},
	uFuturesBookTickerRate:         {derivativesIPPool, 2},
	uFuturesOrderbookTickerAllRate: {derivativesIPPool, 5},
	uFuturesSymbolPriceAllRate:     {derivativesIPPool, 2},
	uFuturesTicker24HourAllRate:    {derivativesIPPool, 40},
	uFuturesAccountTradeListRate:   {derivativesIPPool, 5},
	uFuturesGetAllOrdersRate:       {derivativesIPPool, 5},
	uFuturesCountdownCancelRate:    {derivativesIPPool, 10},
	uFuturesOrdersDefaultRate:      {derivativesOrderPool, 1},
	uFuturesBatchOrdersRate:        {derivativesOrderPool, 5},
	uFuturesOpenAlgoOrdersAllRate:  {derivativesIPPool, 40},
	uFuturesGetAllOpenOrdersRate:   {derivativesIPPool, 40},
	uFuturesADLQuantileRate:        {derivativesIPPool, 5},
	uFuturesPositionRiskRate:       {derivativesIPPool, 5},
	uFuturesAllAlgoOrdersRate:      {derivativesIPPool, 5},
	uFuturesSymbolForceOrdersRate:  {derivativesIPPool, 20},
	uFuturesAllForceOrdersRate:     {derivativesIPPool, 50},
	uFuturesAPIReferralRate:        {derivativesIPPool, 100},

	// /futures/data on the USDⓈ-M and COIN-M hosts
	futuresDataRate: {futuresDataPool, 1},

	// COIN-M futures /dapi
	cFuturesDefaultRate:            {derivativesIPPool, 1},
	cFuturesAccountInformationRate: {derivativesIPPool, 5},
	cFuturesIncomeHistoryRate:      {derivativesIPPool, 20},
	cFuturesAggregateTradesRate:    {derivativesIPPool, 20},
	cFuturesKline100Rate:           {derivativesIPPool, 1},
	cFuturesKline500Rate:           {derivativesIPPool, 2},
	cFuturesKline1000Rate:          {derivativesIPPool, 5},
	cFuturesKlineMaxRate:           {derivativesIPPool, 10},
	cFuturesIndexMarkPriceRate:     {derivativesIPPool, 10},
	cFuturesHistoricalTradesRate:   {derivativesIPPool, 200},
	cFuturesOrderbook50Rate:        {derivativesIPPool, 2},
	cFuturesOrderbook100Rate:       {derivativesIPPool, 5},
	cFuturesOrderbook500Rate:       {derivativesIPPool, 10},
	cFuturesOrderbook1000Rate:      {derivativesIPPool, 20},
	cFuturesRecentTradesRate:       {derivativesIPPool, 5},
	cFuturesBookTickerRate:         {derivativesIPPool, 2},
	cFuturesOrderbookTickerAllRate: {derivativesIPPool, 5},
	cFuturesSymbolPriceAllRate:     {derivativesIPPool, 2},
	cFuturesTicker24HourAllRate:    {derivativesIPPool, 40},
	cFuturesAccountTradeListRate:   {derivativesIPPool, 5},
	cFuturesAllOrdersRate:          {derivativesIPPool, 5},
	cFuturesCountdownCancelRate:    {derivativesIPPool, 10},
	cFuturesBatchOrdersRate:        {derivativesOrderPool, 5},
	cFuturesOrdersDefaultRate:      {derivativesOrderPool, 1},
	cFuturesGetAllOpenOrdersRate:   {derivativesIPPool, 40},
	cFuturesADLQuantileRate:        {derivativesIPPool, 5},
	cFuturesSymbolForceOrdersRate:  {derivativesIPPool, 20},
	cFuturesAllForceOrdersRate:     {derivativesIPPool, 50},

	// Options /eapi
	optionsDefaultRate:                               {optionsIPPool, 1},
	optionsMarginAccountInfoRate:                     {optionsIPPool, 3},
	optionsDownloadIDForOptionTransactionHistoryRate: {optionsIPPool, 5},
	optionsGetTransHistoryDownloadLinkByIDRate:       {optionsIPPool, 5},
	optionsHistoricalExerciseRecordsRate:             {optionsIPPool, 3},
	optionsMarkPriceRate:                             {optionsIPPool, 5},
	optionsOrderbook50Rate:                           {optionsIPPool, 1},
	optionsOrderbook100Rate:                          {optionsIPPool, 5},
	optionsOrderbook500Rate:                          {optionsIPPool, 10},
	optionsOrderbook1000Rate:                         {optionsIPPool, 20},
	optionsRecentTradesRate:                          {optionsIPPool, 5},
	optionsAllTickerPriceStatisticsRate:              {optionsIPPool, 40},
	optionsAutoCancelAllOpenOrdersHeartbeatRate:      {optionsIPPool, 10},
	optionsAccountTradeListRate:                      {optionsIPPool, 5},
	optionsCancelAllByUnderlyingRate:                 {optionsIPPool, 5},
	optionsBatchOrderRate:                            {optionsOrderPool, 5},
	optionsCancelBatchOrdersRate:                     {optionsIPPool, 5},
	optionsDefaultOrderRate:                          {optionsOrderPool, 1},
	optionsPositionInformationRate:                   {optionsIPPool, 5},
	optionsAllQueryOpenOrdersRate:                    {optionsIPPool, 40},
	optionsGetOrderHistoryRate:                       {optionsIPPool, 3},
	optionsUserExerciseRecordRate:                    {optionsIPPool, 5},

	// Portfolio margin /papi
	pmDefaultRate:                                          {portfolioMarginIPPool, 1},
	pmGetAccountBalancesRate:                               {portfolioMarginIPPool, 20},
	pmGetAccountInformationRate:                            {portfolioMarginIPPool, 20},
	pmBNBTransferRate:                                      {pmBNBTransferPool, 1},
	pmGetAutoRepayFuturesStatusRate:                        {portfolioMarginIPPool, 30},
	pmChangeAutoRepayFuturesStatusRate:                     {portfolioMarginIPPool, 750},
	pmGetCMCurrentPositionModeRate:                         {portfolioMarginIPPool, 30},
	pmGetUMCurrentPositionModeRate:                         {portfolioMarginIPPool, 30},
	pmFundAutoCollectionRate:                               {portfolioMarginIPPool, 750},
	pmFundCollectionByAssetRate:                            {portfolioMarginIPPool, 30},
	pmGetCMAccountDetailRate:                               {portfolioMarginIPPool, 5},
	pmGetCMIncomeHistoryRate:                               {portfolioMarginIPPool, 30},
	pmGetUMAccountDetailRate:                               {portfolioMarginIPPool, 5},
	pmGetUMIncomeHistoryRate:                               {portfolioMarginIPPool, 30},
	pmGetCMUserCommissionRate:                              {portfolioMarginIPPool, 20},
	pmGetUMUserCommissionRate:                              {portfolioMarginIPPool, 20},
	pmMarginMaxBorrowRate:                                  {portfolioMarginIPPool, 5},
	pmUMTradingQuantitativeRulesIndicatorsRate:             {portfolioMarginIPPool, 10},
	pmGetMarginLoanRecordRate:                              {portfolioMarginIPPool, 10},
	pmGetMarginMaxWithdrawalRate:                           {portfolioMarginIPPool, 5},
	pmGetMarginRepayRecordRate:                             {portfolioMarginIPPool, 10},
	pmGetPortfolioMarginNegativeBalanceInterestHistoryRate: {portfolioMarginIPPool, 50},
	pmGetUMPositionInformationRate:                         {portfolioMarginIPPool, 5},
	pmNegativeBalanceExchangeRecordRate:                    {portfolioMarginIPPool, 100},
	pmRepayFuturesNegativeBalanceRate:                      {portfolioMarginIPPool, 750},
	pmOrderRate:                                            {portfolioMarginOrderPool, 1},
	pmCancelMarginAccountOpenOrdersOnSymbolRate:            {portfolioMarginIPPool, 5},
	pmGetMarginAccountOCORate:                              {portfolioMarginIPPool, 5},
	pmCancelMarginAccountOCORate:                           {portfolioMarginIPPool, 2},
	pmGetMarginAccountOrderRate:                            {portfolioMarginIPPool, 10},
	pmCancelMarginAccountOrderRate:                         {portfolioMarginIPPool, 2},
	pmGetCMAccountTradeListWithSymbolRate:                  {portfolioMarginIPPool, 20},
	pmGetCMAccountTradeListWithPairRate:                    {portfolioMarginIPPool, 40},
	pmGetCMPositionADLQuantileEstimationRate:               {portfolioMarginIPPool, 5},
	pmMarginAccountLoanAndRepayRate:                        {portfolioMarginIPPool, 100},
	pmOCOOrderRate:                                         {portfolioMarginOrderPool, 2},
	pmGetMarginAccountTradeListRate:                        {portfolioMarginIPPool, 5},
	pmAllCMConditionalOrderWithoutSymbolRate:               {portfolioMarginIPPool, 40},
	pmAllCMOrderWithSymbolRate:                             {portfolioMarginIPPool, 20},
	pmAllCMOrderWithPairRate:                               {portfolioMarginIPPool, 40},
	pmAllCMOpenConditionalOrdersWithoutSymbolRate:          {portfolioMarginIPPool, 40},
	pmRetrieveAllCMOpenOrdersForAllSymbolRate:              {portfolioMarginIPPool, 40},
	pmRetrieveAllUMOpenOrdersForAllSymbolRate:              {portfolioMarginIPPool, 40},
	pmAllMarginAccountOrdersRate:                           {portfolioMarginIPPool, 100},
	pmGetAllUMOrdersRate:                                   {portfolioMarginIPPool, 5},
	pmGetUMAlgoOrderHistoryRate:                            {portfolioMarginIPPool, 5},
	pmCurrentMarginOpenOrderRate:                           {portfolioMarginIPPool, 5},
	pmGetMarginAccountsAllOCOOrdersRate:                    {portfolioMarginIPPool, 100},
	pmGetMarginAccountsOpenOCOOrdersRate:                   {portfolioMarginIPPool, 5},
	pmGetUserCMForceOrdersWithSymbolRate:                   {portfolioMarginIPPool, 20},
	pmGetUserCMForceOrdersWithoutSymbolRate:                {portfolioMarginIPPool, 50},
	pmGetUserUMForceOrdersWithSymbolRate:                   {portfolioMarginIPPool, 20},
	pmGetUserUMForceOrdersWithoutSymbolRate:                {portfolioMarginIPPool, 50},
	pmGetUMAccountTradeListRate:                            {portfolioMarginIPPool, 5},
	pmGetUMPositionADLQuantileEstimationRate:               {portfolioMarginIPPool, 5},
	pmAPIReferralRate:                                      {portfolioMarginIPPool, 100},
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
		budget := poolBudgets[cost.pool]
		if cost.pool.perEndpoint() && cost.weight > 1 {
			limiters[epl] = request.NewRateLimit(budget.interval, budget.limit)
			continue
		}
		if shared[cost.pool] == nil {
			shared[cost.pool] = request.NewRateLimit(budget.interval, budget.limit)
		}
		limiters[epl] = shared[cost.pool]
	}
	return limiters
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

// spotOpenOrdersLimit returns the limit of an open orders query, which costs far more without a symbol
func spotOpenOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOpenOrdersAllRate
	}
	return spotOpenOrdersSpecificRate
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

// uFuturesKlineLimit returns the limit of a USDⓈ-M kline request for its limit; 0 requests the default of 500
func uFuturesKlineLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return uFuturesKline1000Rate
	case limit < 100:
		return uFuturesKline100Rate
	case limit < 500:
		return uFuturesKline500Rate
	case limit <= 1000:
		return uFuturesKline1000Rate
	default:
		return uFuturesKlineMaxRate
	}
}

// uFuturesOrderbookLimit returns the limit of a USDⓈ-M order book request for its limit; 0 requests the default of 500
func uFuturesOrderbookLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return uFuturesOrderbook500Rate
	case limit <= 50:
		return uFuturesOrderbook50Rate
	case limit <= 100:
		return uFuturesOrderbook100Rate
	case limit <= 500:
		return uFuturesOrderbook500Rate
	default:
		return uFuturesOrderbook1000Rate
	}
}

// cFuturesKlineLimit returns the limit of a COIN-M kline request for its limit; 0 requests the default of 500
func cFuturesKlineLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return cFuturesKline1000Rate
	case limit < 100:
		return cFuturesKline100Rate
	case limit < 500:
		return cFuturesKline500Rate
	case limit <= 1000:
		return cFuturesKline1000Rate
	default:
		return cFuturesKlineMaxRate
	}
}

// cFuturesOrderbookLimit returns the limit of a COIN-M order book request for its limit; 0 requests the default of 500
func cFuturesOrderbookLimit(limit uint64) request.EndpointLimit {
	switch {
	case limit == 0:
		return cFuturesOrderbook500Rate
	case limit <= 50:
		return cFuturesOrderbook50Rate
	case limit <= 100:
		return cFuturesOrderbook100Rate
	case limit <= 500:
		return cFuturesOrderbook500Rate
	default:
		return cFuturesOrderbook1000Rate
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
