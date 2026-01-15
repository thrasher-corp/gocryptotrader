package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	spotInterval    = time.Minute
	spotRequestRate = 6000
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval        = 10 * time.Second
	spotOrderRequestRate     = 100
	cFuturesInterval         = time.Minute
	cFuturesRequestRate      = 2400
	cFuturesOrderInterval    = time.Minute
	cFuturesOrderRequestRate = 1200
	uFuturesInterval         = time.Minute
	uFuturesRequestRate      = 2400
	portfolioMarginRate      = 1200
	portfolioMarginInterval  = time.Minute
	uFuturesOrderInterval    = time.Second * 10
	uFuturesOrderRequestRate = 300
)

// Binance Spot rate limits
const (
	spotDefaultRate request.EndpointLimit = iota
	aggTradesRate
	listenKeyRate
	sapiDefaultRate
	getV3SubAccountAssetsRate
	allCoinInfoRate
	dailyAccountSnapshotRate
	fundWithdrawalRate
	withdrawalHistoryRate
	spotExchangeInfo
	spotHistoricalTradesRate
	spotOrderbookDepth500Rate
	spotOrderbookDepth100Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	getRecentTradesListRate
	getOldTradeLookupRate
	spotOrderbookTickerAllRate
	spotBookTickerRate
	spotSymbolPriceAllRate
	spotSymbolPriceRate
	getAggregateTradeListRate
	getKlineRate
	getCurrentAveragePriceRate
	get24HrTickerPriceChangeStatisticsRate
	getTickers20Rate
	getTickers100Rate
	getTickersMoreThan100Rate
	spotPriceChangeAllRate
	spotTicker1Rate
	spotTicker20Rate
	spotTicker100Rate
	spotTickerAllRate
	spotOpenOrdersAllRate
	allCrossMarginFeeDataRate
	allIsolatedMarginFeeDataRate
	marginCurrentOrderCountUsageRate
	depositAddressesRate
	dustTransferRate
	assetDividendRecordRate
	userUniversalTransferRate
	userAssetsRate
	busdConvertHistoryRate
	busdConvertRate
	cloudMiningPaymentAndRefundHistoryRate
	autoConvertingStableCoinsRate
	getMinersListRate
	getEarningsListRate
	extraBonusListRate
	getHashrateRescaleRate
	getHashrateRescaleDetailRate
	getHasrateRescaleRequestRate
	cancelHashrateResaleConfigurationRate
	statisticsListRate
	miningAccountListRate
	miningAccountEarningRate
	getDepositAddressListInNetworkRate
	getUserWalletBalanceRate
	getUserDelegationHistoryRate
	symbolDelistScheduleForSpotRate
	withdrawAddressListRate
	crossMarginCollateralRatioRate
	smallLiabilityExchangeCoinListRate
	getSmallLiabilityExchangeCoinListRate
	getSmallLiabilityExchangeRate
	marginHourlyInterestRate
	marginCapitalFlowRate
	marginTokensAndSymbolsDelistScheduleRate
	marginAvailableInventoryRate
	marginManualLiquidiationRate
	getSubAccountAssetRate
	getSubAccountStatusOnMarginOrFuturesRate
	marginAccountInformationRate
	subAccountMarginAccountDetailRate
	getSubAccountSummaryOfMarginAccountRate
	getDetailSubAccountFuturesAccountRate
	getFuturesPositionRiskOfSubAccountV1Rate
	getFuturesSubAccountSummaryV2Rate
	ipRestrictionForSubAccountAPIKeyRate
	deleteIPListForSubAccountAPIKeyRate
	addIPRestrictionSubAccountAPIKey
	getManagedSubAccountSnapshotRate
	managedSubAccountTransferLogRate
	managedSubAccountFuturesAssetDetailRate
	getManagedSubAccountListRate
	getSubAccountTransactionStatisticsRate
	marginAccountBorrowRepayRate
	getCrossMarginAccountDetailRate
	getCrossMarginAccountOrderRate
	getMarginAccountsOpenOrdersRate
	marginAccountsAllOrdersRate
	marginAccountOpenOCOOrdersRate
	marginAccountTradeListRate
	marginMaxBorrowRate
	maxTransferOutRate
	marginAccountSummaryRate
	getIsolatedMarginAccountInfoRate
	deleteIsolatedMarginAccountRate
	enableIsolatedMarginAccountRate
	allIsolatedMarginSymbol
	marginOCOOrderRate
	getMarginAccountOCOOrderRate
	getMarginAccountAllOCORate
	getOCOListRate
	getAllOCOOrdersRate
	getOpenOCOListRate

	simpleEarnProductsRate
	getSimpleEarnProductPositionRate
	simpleAccountRate
	getFlexibleSubscriptionRecordRate
	nftRate
	spotRebateHistoryRate
	convertTradeFlowHistoryRate
	getLimitOpenOrdersRate
	cancelLimitOrderRate
	placeLimitOrderRate
	orderStatusRate
	acceptQuoteRate
	sendQuoteRequestRate
	getLockedSubscriptionRecordsRate
	getRedemptionRecordRate
	getRewardHistoryRate
	setAutoSubscribeRate
	personalLeftQuotaRate
	subscriptionPreviewRate
	simpleEarnRateHistoryRate
	etherumStakingRedemptionRate
	ethStakingHistoryRate
	ethRedemptionHistoryRate
	bethRewardDistributionHistoryRate
	currentETHStakingQuotaRate
	getWBETHRateHistoryRate
	ethStakingAccountRate
	wrapBETHRate
	wbethWrapOrUnwrapHistoryRate
	wbethRewardsHistoryRate

	solStakingAccountRate
	solStakingQuotaDetailsRate
	subscribeSOLStakingRate
	redeemSOLRate
	claimbBoostRewardsRate
	solStakingHistoryRate
	solRedemptionHistoryRate
	bnsolRewardsHistoryRate
	bnsolRateHistory
	boostRewardsHistoryRate
	unclaimedRewardsRate

	createSubAccountRate
	getSubAccountRate
	enableFuturesForSubAccountRate
	createAPIKeyForSubAccountRate

	futuresFundTransfersFetchRate
	futureTickLevelOrderbookHistoricalDataDownloadLinkRate
	fundAutoCollectionRate
	getAutoRepayFuturesStatusRate
	pmAssetLeverageRate

	getVIPLoanOngoingOrdersRate
	vipLoanRepayRate
	vipLoanRenewRate
	getVIPLoanRepaymentHistoryRate
	getVIPLoanAccruedInterest
	checkLockedValueVIPCollateralAccountRate
	vipLoanBorrowRate
	getVIPLoanableAssetsRate
	getCollateralAssetDataRate
	getApplicationStatusRate
	getVIPBorrowInterestRate
	vipLoanInterestRateHistoryRate

	fiatDepositWithdrawHistRate

	getAllConvertPairsRate
	getOrderQuantityPrecisionPerAssetRate
	testNewOrderWithCommissionRate
	payTradeEndpointsRate

	// Classic Portfolio Rate
	classicPMAccountInfoRate
	classicPMCollateralRate
	getClassicPMBankruptacyLoanAmountRate
	repayClassicPMBankruptacyLoanRate
	classicPMNegativeBalanceInterestHistory
	pmAssetIndexPriceRate
	fundCollectionByAssetRate
	transferBNBRate
	changeAutoRepayFuturesStatusRate
	repayFuturesNegativeBalanceRate

	// Spot Algo Endpoints
	spotTwapNewOrderRate

	// Staking Endpoints
	subscribeETHStakingRate

	// Futures Algo
	placeVPOrderRate
	placeTWAveragePriceNewOrderRate

	spotOpenOrdersSpecificRate
	spotOrderRate
	spotOrderQueryRate
	spotAllOrdersRate
	spotAccountInformationRate
	accountTradeListRate
	currentOrderCountUsageRate
	queryPreventedMatchsWithRate
	getAllocationsRate
	preventedMatchesByOrderIDRate
	getCommissionRate
	borrowRepayRecordsInMarginAccountRate
	getPriceMarginIndexRate
	marginAccountNewOrderRate
	marginAccountCancelOrderRate
	adjustCrossMarginMaxLeverageRate
	uFuturesDefaultRate
	uFuturesHistoricalTradesRate
	uFuturesSymbolOrdersRate
	uFuturesPairOrdersRate
	uFuturesCurrencyForceOrdersRate
	uFuturesAllForceOrdersRate
	uFuturesIncomeHistoryRate
	uFuturesOrderbook50Rate
	uFuturesOrderbook100Rate
	uFuturesOrderbook500Rate
	uFuturesOrderbook1000Rate
	uFuturesKline100Rate
	uFuturesKline500Rate
	uFuturesKline1000Rate
	uFuturesKlineMaxRate
	uFuturesTickerPriceHistoryRate
	uFuturesOrdersDefaultRate
	uFuturesGetAllOrdersRate
	uFuturesAccountInformationRate
	uFuturesOrderbookTickerAllRate
	uFuturesCountdownCancelRate
	uFuturesBatchOrdersRate
	uFuturesGetAllOpenOrdersRate
	cFuturesDefaultRate
	cFuturesHistoricalTradesRate
	cFuturesTickerPriceHistoryRate
	cFuturesIncomeHistoryRate
	cFuturesOrderbook50Rate
	cFuturesOrderbook100Rate
	cFuturesOrderbook500Rate
	cFuturesOrderbook1000Rate
	cFuturesKline100Rate
	cFuturesKline500Rate
	cFuturesKline1000Rate
	cFuturesKlineMaxRate
	cFuturesIndexMarkPriceRate
	cFuturesBatchOrdersRate
	cFuturesCancelAllOrdersRate
	cFuturesGetAllOpenOrdersRate
	cFuturesAllForceOrdersRate
	cFuturesCurrencyForceOrdersRate
	cFuturesPairOrdersRate
	cFuturesSymbolOrdersRate
	cFuturesAccountInformationRate
	cFuturesOrderbookTickerAllRate
	cFuturesOrdersDefaultRate
	uFuturesMultiAssetMarginRate
	uFuturesSetMultiAssetMarginRate
	optionsDefaultRate
	optionsRecentTradesRate
	optionsHistoricalTradesRate
	optionsMarkPriceRate
	optionsAllTickerPriceStatistics
	optionsHistoricalExerciseRecordsRate
	optionsAccountInfoRate
	optionsDefaultOrderRate
	optionsBatchOrderRate
	optionsAllQueryOpenOrdersRate
	optionsGetOrderHistory
	optionsPositionInformationRate
	optionsAccountTradeListRate
	optionsUserExerciseRecordRate
	optionsDownloadIDForOptionTrasactionHistoryRate
	optionsGetTransHistoryDownloadLinkByIDRate
	optionsMarginAccountInfoRate
	optionsAutoCancelAllOpenOrdersHeartbeatRate

	// the following are portfolio margin endpoint rates
	pmDefaultRate
	pmMarginAccountLoanAndRepayRate
	pmCancelMarginAccountOpenOrdersOnSymbolRate
	pmCancelMarginAccountOCORate
	pmRetrieveAllUMOpenOrdersForAllSymbolRate
	pmGetAllUMOrdersRate
	pmRetrieveAllCMOpenOrdersForAllSymbolRate
	pmAllCMOrderWithSymbolRate
	pmAllCMOrderWithoutSymbolRate
	pmUMOpenConditionalOrdersRate
	pmAllUMConditionalOrdersWithoutSymbolRate
	pmAllCMOpenConditionalOrdersWithoutSymbolRate
	pmAllCMConditionalOrderWithoutSymbolRate
	pmGetMarginAccountOrderRate
	pmCurrentMarginOpenOrderRate
	pmAllMarginAccountOrdersRate
	pmGetMarginAccountOCORate
	pmGetMarginAccountsAllOCOOrdersRate
	pmGetMarginAccountsOpenOCOOrdersRate
	pmGetMarginAccountTradeListRate
	pmGetAccountBalancesRate
	pmGetAccountInformationRate
	pmMarginMaxBorrowRate
	pmGetMarginMaxWithdrawalRate
	pmGetUMPositionInformationRate
	pmGetUMCurrentPositionModeRate
	pmGetCMCurrentPositionModeRate
	pmGetUMAccountTradeListRate
	pmGetCMAccountTradeListWithSymbolRate
	pmGetCMAccountTradeListWithPairRate
	pmGetUserUMForceOrdersWithSymbolRate
	pmGetUserUMForceOrdersWithoutSymbolRate
	pmGetUserCMForceOrdersWithSymbolRate
	pmGetUserCMForceOrdersWithoutSymbolRate
	pmUMTradingQuantitativeRulesIndicatorsRate
	pmGetUMUserCommissionRate
	pmGetCMUserCommissionRate
	pmGetMarginLoanRecordRate
	pmGetMarginRepayRecordRate
	pmGetPortfolioMarginNegativeBalanceInterestHistoryRate
	pmFundAutoCollectionRate
	pmFundCollectionByAssetRate
	pmBNBTransferRate
	pmGetUMIncomeHistoryRate
	pmGetCMIncomeHistoryRate
	pmGetUMAccountDetailRate
	pmGetCMAccountDetailRate
	pmChangeAutoRepayFuturesStatusRate
	pmGetAutoRepayFuturesStatusRate
	pmRepayFuturesNegativeBalanceRate
	pmGetUMPositionADLQuantileEstimationRate
	pmGetCMPositionADLQuantileEstimationRate

	getFlexibleSimpleEarnProductPositionRate

	cryptoLoansIncomeHistory
	getLoanBorrowHistoryRate
	getBorrowOngoingOrdersRate
	cryptoRepayLoanRate
	repaymentHistoryRate
	adjustLTVRate
	getLoanLTVAdjustmentHistoryRate
	getLoanableAssetsDataRate
	collateralAssetsDataRate
	checkCollateralRepayRate
	cryptoLoanCustomizeMarginRate
	borrowFlexibleRate
	getFlexibleLoanOngoingOrdersRate
	flexibleLoanLiquidiationHistoryRate
	flexibleBorrowHistoryRate
	repayFlexibleLoanHistoryRate
	flexibleLoanRepaymentHistoryRate
	flexibleLoanCollateralRepaymentRate
	adjustFlexibleLoanRate
	flexibleLoanAdjustLTVRate
	flexibleLoanAssetDataRate
	flexibleLoanCollateralAssetRate
)

// GetRateLimits returns the rate limit for the exchange
func GetRateLimits() request.RateLimitDefinitions {
	spotLimiter := request.NewRateLimit(spotInterval, spotRequestRate)
	spotOrdersLimiter := request.NewRateLimit(spotOrderInterval, spotOrderRequestRate)
	uFuturesLimiter := request.NewRateLimit(uFuturesInterval, uFuturesRequestRate)
	uFuturesOrdersLimiter := request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate)
	cFuturesLimiter := request.NewRateLimit(cFuturesInterval, cFuturesRequestRate)
	cFuturesOrdersLimiter := request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate)
	eOptionsLimiter := request.NewRateLimit(time.Minute, 400)
	eOptionsOrderLimiter := request.NewRateLimit(time.Minute, 100)
	portfolioMarginLimiter := request.NewRateLimit(portfolioMarginInterval, portfolioMarginRate)

	// Sapi Endpoints
	//
	// Endpoints are marked according to IP or UID limit and their corresponding weight value.
	// Each endpoint with IP limits has an independent 12000 per minute limit, or per second limit if specified explicitly
	// Each endpoint with UID limits has an independent 180000 per minute limit, or per second limit if specified explicitly
	sapiDefaultLimiter := request.NewRateLimit(time.Second, 110)
	marginAccountTradeLimiter := request.NewRateLimit(time.Second, 16917)
	walletLimiter := request.NewRateLimit(time.Second, 23995)
	subAccountLimiter := request.NewRateLimit(time.Second, 11820)
	convertLimiter := request.NewRateLimit(time.Second, 7620)
	simpleEarnLimiter := request.NewRateLimit(time.Second, 2700)
	nftLimiter := request.NewRateLimit(time.Second, 12000)
	spotRebateHistoryLimiter := request.NewRateLimit(time.Second, 12000)
	payTradeEndpointsLimiter := request.NewRateLimit(time.Second, 3000)
	vipLoanEndpointsLimiter := request.NewRateLimit(time.Second, 26600)
	fiatDepositWithdrawHistLimiter := request.NewRateLimit(time.Second, 90000)
	classicPMLimiter := request.NewRateLimit(time.Second, 5145)
	spotAlgoLimiter := request.NewRateLimit(time.Second, 3000)
	placeVPOrderLimiter := request.NewRateLimit(time.Second, 6000)
	futuresFundTransfersFetchLimiter := request.NewRateLimit(time.Second, 210)
	miningLimiter := request.NewRateLimit(time.Second, 55)
	stakingLimiter := request.NewRateLimit(time.Second, 1500)
	cryptoLoanLimiter := request.NewRateLimit(time.Second, 52900)

	return request.RateLimitDefinitions{
		spotDefaultRate:                        request.GetRateLimiterWithWeight(spotLimiter, 1),
		spotBookTickerRate:                     request.GetRateLimiterWithWeight(spotLimiter, 2),
		spotSymbolPriceRate:                    request.GetRateLimiterWithWeight(spotLimiter, 2),
		getAggregateTradeListRate:              request.GetRateLimiterWithWeight(spotLimiter, 2),
		getKlineRate:                           request.GetRateLimiterWithWeight(spotLimiter, 2),
		getCurrentAveragePriceRate:             request.GetRateLimiterWithWeight(spotLimiter, 2),
		get24HrTickerPriceChangeStatisticsRate: request.GetRateLimiterWithWeight(spotLimiter, 2),
		getTickers20Rate:                       request.GetRateLimiterWithWeight(spotLimiter, 2),
		queryPreventedMatchsWithRate:           request.GetRateLimiterWithWeight(spotLimiter, 2),
		aggTradesRate:                          request.GetRateLimiterWithWeight(spotLimiter, 2),
		listenKeyRate:                          request.GetRateLimiterWithWeight(spotLimiter, 2),
		spotOrderbookTickerAllRate:             request.GetRateLimiterWithWeight(spotLimiter, 4),
		spotSymbolPriceAllRate:                 request.GetRateLimiterWithWeight(spotLimiter, 4),
		getOCOListRate:                         request.GetRateLimiterWithWeight(spotLimiter, 4),
		spotHistoricalTradesRate:               request.GetRateLimiterWithWeight(spotLimiter, 5),
		spotOrderbookDepth100Rate:              request.GetRateLimiterWithWeight(spotLimiter, 5),
		getHashrateRescaleRate:                 request.GetRateLimiterWithWeight(spotLimiter, 5),
		spotOrderbookDepth500Rate:              request.GetRateLimiterWithWeight(spotLimiter, 25),
		getRecentTradesListRate:                request.GetRateLimiterWithWeight(spotLimiter, 25),
		getOldTradeLookupRate:                  request.GetRateLimiterWithWeight(spotLimiter, 25),
		accountTradeListRate:                   request.GetRateLimiterWithWeight(spotLimiter, 20),
		spotExchangeInfo:                       request.GetRateLimiterWithWeight(spotLimiter, 20),
		testNewOrderWithCommissionRate:         request.GetRateLimiterWithWeight(spotLimiter, 20),
		getAllOCOOrdersRate:                    request.GetRateLimiterWithWeight(spotLimiter, 20),
		getAutoRepayFuturesStatusRate:          request.GetRateLimiterWithWeight(spotLimiter, 30),
		spotOrderbookDepth1000Rate:             request.GetRateLimiterWithWeight(spotLimiter, 50),
		pmAssetLeverageRate:                    request.GetRateLimiterWithWeight(spotLimiter, 50),
		spotPriceChangeAllRate:                 request.GetRateLimiterWithWeight(spotLimiter, 80),
		getTickersMoreThan100Rate:              request.GetRateLimiterWithWeight(spotLimiter, 80),

		spotOrderRate:              request.GetRateLimiterWithWeight(spotOrdersLimiter, 1),
		spotOpenOrdersSpecificRate: request.GetRateLimiterWithWeight(spotOrdersLimiter, 6),
		spotOrderQueryRate:         request.GetRateLimiterWithWeight(spotOrdersLimiter, 4),
		spotAllOrdersRate:          request.GetRateLimiterWithWeight(spotOrdersLimiter, 20),
		spotOpenOrdersAllRate:      request.GetRateLimiterWithWeight(spotOrdersLimiter, 80),

		spotOrderbookDepth5000Rate:    request.GetRateLimiterWithWeight(spotLimiter, 250),
		spotAccountInformationRate:    request.GetRateLimiterWithWeight(spotLimiter, 20),
		getOpenOCOListRate:            request.GetRateLimiterWithWeight(spotLimiter, 6),
		currentOrderCountUsageRate:    request.GetRateLimiterWithWeight(spotLimiter, 40),
		getTickers100Rate:             request.GetRateLimiterWithWeight(spotLimiter, 40),
		getAllocationsRate:            request.GetRateLimiterWithWeight(spotLimiter, 20),
		preventedMatchesByOrderIDRate: request.GetRateLimiterWithWeight(spotLimiter, 20),
		getCommissionRate:             request.GetRateLimiterWithWeight(spotLimiter, 20),

		uFuturesDefaultRate:            request.GetRateLimiterWithWeight(uFuturesLimiter, 1),
		uFuturesKline100Rate:           request.GetRateLimiterWithWeight(uFuturesLimiter, 1),
		uFuturesOrderbook50Rate:        request.GetRateLimiterWithWeight(uFuturesLimiter, 2),
		uFuturesKline500Rate:           request.GetRateLimiterWithWeight(uFuturesLimiter, 2),
		uFuturesOrderbookTickerAllRate: request.GetRateLimiterWithWeight(uFuturesLimiter, 2),

		uFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(uFuturesLimiter, 5),
		uFuturesKline1000Rate:           request.GetRateLimiterWithWeight(uFuturesLimiter, 5),
		uFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(uFuturesLimiter, 5),
		uFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(uFuturesLimiter, 10),
		uFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(uFuturesLimiter, 10),
		uFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(uFuturesLimiter, 20),
		uFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(uFuturesLimiter, 20),
		uFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(uFuturesLimiter, 40),
		uFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 1),
		uFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 5),
		uFuturesGetAllOrdersRate:        request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 5),
		uFuturesCountdownCancelRate:     request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 10),
		uFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 20),
		uFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 20),
		uFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 30),
		uFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 40),
		uFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 40),
		uFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(uFuturesOrdersLimiter, 50),
		cFuturesKline100Rate:            request.GetRateLimiterWithWeight(cFuturesLimiter, 1),
		cFuturesKline500Rate:            request.GetRateLimiterWithWeight(cFuturesLimiter, 2),
		cFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithWeight(cFuturesLimiter, 2),
		cFuturesKline1000Rate:           request.GetRateLimiterWithWeight(cFuturesLimiter, 5),
		cFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(cFuturesLimiter, 5),
		cFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(cFuturesLimiter, 10),
		cFuturesIndexMarkPriceRate:      request.GetRateLimiterWithWeight(cFuturesLimiter, 10),
		cFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(cFuturesLimiter, 20),
		cFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(cFuturesLimiter, 20),
		cFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(cFuturesLimiter, 40),
		cFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(cFuturesLimiter, 50),
		cFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 1),
		cFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 5),
		cFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 5),
		cFuturesCancelAllOrdersRate:     request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 10),
		cFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 20),
		cFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 20),
		cFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(cFuturesOrdersLimiter, 40),
		cFuturesOrderbook50Rate:         request.GetRateLimiterWithWeight(cFuturesLimiter, 2),
		cFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(cFuturesLimiter, 5),
		cFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(cFuturesLimiter, 10),
		cFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(cFuturesLimiter, 20),
		cFuturesDefaultRate:             request.GetRateLimiterWithWeight(cFuturesLimiter, 1),
		uFuturesMultiAssetMarginRate:    request.GetRateLimiterWithWeight(uFuturesLimiter, 30),
		uFuturesSetMultiAssetMarginRate: request.GetRateLimiterWithWeight(uFuturesLimiter, 1),

		// Options Rate Limits
		optionsDefaultRate:                                     request.GetRateLimiterWithWeight(eOptionsLimiter, 1),
		optionsRecentTradesRate:                                request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsMarkPriceRate:                                   request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsAllTickerPriceStatistics:                        request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsPositionInformationRate:                         request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsAccountTradeListRate:                            request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsUserExerciseRecordRate:                          request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsDownloadIDForOptionTrasactionHistoryRate:        request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsGetTransHistoryDownloadLinkByIDRate:             request.GetRateLimiterWithWeight(eOptionsLimiter, 5),
		optionsHistoricalTradesRate:                            request.GetRateLimiterWithWeight(eOptionsLimiter, 20),
		optionsHistoricalExerciseRecordsRate:                   request.GetRateLimiterWithWeight(eOptionsLimiter, 3),
		optionsMarginAccountInfoRate:                           request.GetRateLimiterWithWeight(eOptionsLimiter, 3),
		optionsAccountInfoRate:                                 request.GetRateLimiterWithWeight(eOptionsOrderLimiter, 5),
		optionsBatchOrderRate:                                  request.GetRateLimiterWithWeight(eOptionsOrderLimiter, 5),
		optionsDefaultOrderRate:                                request.GetRateLimiterWithWeight(eOptionsOrderLimiter, 1),
		optionsAllQueryOpenOrdersRate:                          request.GetRateLimiterWithWeight(eOptionsOrderLimiter, 40),
		optionsGetOrderHistory:                                 request.GetRateLimiterWithWeight(eOptionsOrderLimiter, 3),
		optionsAutoCancelAllOpenOrdersHeartbeatRate:            request.GetRateLimiterWithWeight(eOptionsLimiter, 10),
		pmDefaultRate:                                          request.GetRateLimiterWithWeight(portfolioMarginLimiter, 1),
		pmMarginAccountLoanAndRepayRate:                        request.GetRateLimiterWithWeight(portfolioMarginLimiter, 100),
		pmAllMarginAccountOrdersRate:                           request.GetRateLimiterWithWeight(portfolioMarginLimiter, 100),
		pmGetMarginAccountsAllOCOOrdersRate:                    request.GetRateLimiterWithWeight(portfolioMarginLimiter, 100),
		pmCancelMarginAccountOpenOrdersOnSymbolRate:            request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetAllUMOrdersRate:                                   request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetMarginAccountOrderRate:                            request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmCurrentMarginOpenOrderRate:                           request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetMarginAccountOCORate:                              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetMarginAccountsOpenOCOOrdersRate:                   request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetMarginAccountTradeListRate:                        request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmMarginMaxBorrowRate:                                  request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetMarginMaxWithdrawalRate:                           request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetUMPositionInformationRate:                         request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetUMAccountTradeListRate:                            request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetUMAccountDetailRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetCMAccountDetailRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetUMPositionADLQuantileEstimationRate:               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmGetCMPositionADLQuantileEstimationRate:               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 5),
		pmCancelMarginAccountOCORate:                           request.GetRateLimiterWithWeight(portfolioMarginLimiter, 2),
		pmRetrieveAllUMOpenOrdersForAllSymbolRate:              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmRetrieveAllCMOpenOrdersForAllSymbolRate:              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmAllCMOrderWithoutSymbolRate:                          request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmUMOpenConditionalOrdersRate:                          request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmAllUMConditionalOrdersWithoutSymbolRate:              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmAllCMOpenConditionalOrdersWithoutSymbolRate:          request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmAllCMConditionalOrderWithoutSymbolRate:               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmGetCMAccountTradeListWithPairRate:                    request.GetRateLimiterWithWeight(portfolioMarginLimiter, 40),
		pmAllCMOrderWithSymbolRate:                             request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetAccountBalancesRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetAccountInformationRate:                            request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetCMAccountTradeListWithSymbolRate:                  request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetUserUMForceOrdersWithSymbolRate:                   request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetUserCMForceOrdersWithSymbolRate:                   request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetUMUserCommissionRate:                              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetCMUserCommissionRate:                              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 20),
		pmGetUMCurrentPositionModeRate:                         request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmGetCMCurrentPositionModeRate:                         request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmFundCollectionByAssetRate:                            request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmGetUMIncomeHistoryRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmGetCMIncomeHistoryRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmGetAutoRepayFuturesStatusRate:                        request.GetRateLimiterWithWeight(portfolioMarginLimiter, 30),
		pmGetUserUMForceOrdersWithoutSymbolRate:                request.GetRateLimiterWithWeight(portfolioMarginLimiter, 50),
		pmGetUserCMForceOrdersWithoutSymbolRate:                request.GetRateLimiterWithWeight(portfolioMarginLimiter, 50),
		pmGetPortfolioMarginNegativeBalanceInterestHistoryRate: request.GetRateLimiterWithWeight(portfolioMarginLimiter, 50),
		pmUMTradingQuantitativeRulesIndicatorsRate:             request.GetRateLimiterWithWeight(portfolioMarginLimiter, 10),
		pmGetMarginLoanRecordRate:                              request.GetRateLimiterWithWeight(portfolioMarginLimiter, 10),
		pmGetMarginRepayRecordRate:                             request.GetRateLimiterWithWeight(portfolioMarginLimiter, 10),
		pmFundAutoCollectionRate:                               request.GetRateLimiterWithWeight(portfolioMarginLimiter, 750),
		pmBNBTransferRate:                                      request.GetRateLimiterWithWeight(portfolioMarginLimiter, 750),
		pmChangeAutoRepayFuturesStatusRate:                     request.GetRateLimiterWithWeight(portfolioMarginLimiter, 750),
		pmRepayFuturesNegativeBalanceRate:                      request.GetRateLimiterWithWeight(portfolioMarginLimiter, 750),

		// /sapi/* endpoints
		sapiDefaultRate:           request.GetRateLimiterWithWeight(sapiDefaultLimiter, 1),
		getV3SubAccountAssetsRate: request.GetRateLimiterWithWeight(sapiDefaultLimiter, 60),

		// Wallet Endpoints
		userAssetsRate:                         request.GetRateLimiterWithWeight(walletLimiter, 5),
		busdConvertHistoryRate:                 request.GetRateLimiterWithWeight(walletLimiter, 5),
		busdConvertRate:                        request.GetRateLimiterWithWeight(walletLimiter, 5),
		allCoinInfoRate:                        request.GetRateLimiterWithWeight(walletLimiter, 10),
		depositAddressesRate:                   request.GetRateLimiterWithWeight(walletLimiter, 10),
		dustTransferRate:                       request.GetRateLimiterWithWeight(walletLimiter, 10),
		assetDividendRecordRate:                request.GetRateLimiterWithWeight(walletLimiter, 10),
		getDepositAddressListInNetworkRate:     request.GetRateLimiterWithWeight(walletLimiter, 10),
		withdrawAddressListRate:                request.GetRateLimiterWithWeight(walletLimiter, 10),
		getUserWalletBalanceRate:               request.GetRateLimiterWithWeight(walletLimiter, 60),
		getUserDelegationHistoryRate:           request.GetRateLimiterWithWeight(walletLimiter, 60),
		symbolDelistScheduleForSpotRate:        request.GetRateLimiterWithWeight(walletLimiter, 100),
		fundWithdrawalRate:                     request.GetRateLimiterWithWeight(walletLimiter, 600),
		cloudMiningPaymentAndRefundHistoryRate: request.GetRateLimiterWithWeight(walletLimiter, 600),
		autoConvertingStableCoinsRate:          request.GetRateLimiterWithWeight(walletLimiter, 600),
		userUniversalTransferRate:              request.GetRateLimiterWithWeight(walletLimiter, 900),
		dailyAccountSnapshotRate:               request.GetRateLimiterWithWeight(walletLimiter, 2400),
		withdrawalHistoryRate:                  request.GetRateLimiterWithWeight(walletLimiter, 10),

		// Sub-Account Rate
		getSubAccountStatusOnMarginOrFuturesRate: request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		marginAccountInformationRate:             request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		subAccountMarginAccountDetailRate:        request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		getSubAccountSummaryOfMarginAccountRate:  request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		getDetailSubAccountFuturesAccountRate:    request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		getFuturesPositionRiskOfSubAccountV1Rate: request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		getFuturesSubAccountSummaryV2Rate:        request.GetRateLimiterWithWeight(subAccountLimiter, 10),
		getSubAccountAssetRate:                   request.GetRateLimiterWithWeight(subAccountLimiter, 60),
		managedSubAccountTransferLogRate:         request.GetRateLimiterWithWeight(subAccountLimiter, 60),
		managedSubAccountFuturesAssetDetailRate:  request.GetRateLimiterWithWeight(subAccountLimiter, 60),
		getManagedSubAccountListRate:             request.GetRateLimiterWithWeight(subAccountLimiter, 60),
		getSubAccountTransactionStatisticsRate:   request.GetRateLimiterWithWeight(subAccountLimiter, 60),
		getManagedSubAccountSnapshotRate:         request.GetRateLimiterWithWeight(subAccountLimiter, 2400),
		ipRestrictionForSubAccountAPIKeyRate:     request.GetRateLimiterWithWeight(subAccountLimiter, 3000),
		deleteIPListForSubAccountAPIKeyRate:      request.GetRateLimiterWithWeight(subAccountLimiter, 3000),
		addIPRestrictionSubAccountAPIKey:         request.GetRateLimiterWithWeight(subAccountLimiter, 3000),

		// NFT Endpoints
		nftRate: request.GetRateLimiterWithWeight(nftLimiter, 3000),

		// Spot Rebate History
		spotRebateHistoryRate: request.GetRateLimiterWithWeight(spotRebateHistoryLimiter, 12000),

		// Convert Rate
		getAllConvertPairsRate:                request.GetRateLimiterWithWeight(convertLimiter, 20),
		getOrderQuantityPrecisionPerAssetRate: request.GetRateLimiterWithWeight(convertLimiter, 100),
		orderStatusRate:                       request.GetRateLimiterWithWeight(convertLimiter, 100),
		sendQuoteRequestRate:                  request.GetRateLimiterWithWeight(convertLimiter, 200),
		cancelLimitOrderRate:                  request.GetRateLimiterWithWeight(convertLimiter, 200),
		acceptQuoteRate:                       request.GetRateLimiterWithWeight(convertLimiter, 500),
		placeLimitOrderRate:                   request.GetRateLimiterWithWeight(convertLimiter, 500),
		getLimitOpenOrdersRate:                request.GetRateLimiterWithWeight(convertLimiter, 3000),
		convertTradeFlowHistoryRate:           request.GetRateLimiterWithWeight(convertLimiter, 3000),

		// Pay Endpoints
		payTradeEndpointsRate: request.GetRateLimiterWithWeight(payTradeEndpointsLimiter, 3000),

		fiatDepositWithdrawHistRate: request.GetRateLimiterWithWeight(fiatDepositWithdrawHistLimiter, 90000),

		// VIP Endpoints
		getVIPLoanOngoingOrdersRate:              request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getVIPLoanRepaymentHistoryRate:           request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getVIPLoanAccruedInterest:                request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getVIPLoanableAssetsRate:                 request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getCollateralAssetDataRate:               request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getApplicationStatusRate:                 request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		getVIPBorrowInterestRate:                 request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		vipLoanInterestRateHistoryRate:           request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 400),
		vipLoanRepayRate:                         request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 6000),
		vipLoanRenewRate:                         request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 6000),
		checkLockedValueVIPCollateralAccountRate: request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 6000),
		vipLoanBorrowRate:                        request.GetRateLimiterWithWeight(vipLoanEndpointsLimiter, 6000),

		// Classic Portfolio Margin
		classicPMAccountInfoRate:                request.GetRateLimiterWithWeight(classicPMLimiter, 5),
		changeAutoRepayFuturesStatusRate:        request.GetRateLimiterWithWeight(classicPMLimiter, 30),
		classicPMCollateralRate:                 request.GetRateLimiterWithWeight(classicPMLimiter, 50),
		classicPMNegativeBalanceInterestHistory: request.GetRateLimiterWithWeight(classicPMLimiter, 50),
		pmAssetIndexPriceRate:                   request.GetRateLimiterWithWeight(classicPMLimiter, 50),
		fundCollectionByAssetRate:               request.GetRateLimiterWithWeight(classicPMLimiter, 60),
		getClassicPMBankruptacyLoanAmountRate:   request.GetRateLimiterWithWeight(classicPMLimiter, 500),
		fundAutoCollectionRate:                  request.GetRateLimiterWithWeight(classicPMLimiter, 1500),
		transferBNBRate:                         request.GetRateLimiterWithWeight(classicPMLimiter, 1500),
		repayFuturesNegativeBalanceRate:         request.GetRateLimiterWithWeight(classicPMLimiter, 1500),
		repayClassicPMBankruptacyLoanRate:       request.GetRateLimiterWithWeight(classicPMLimiter, 3000),

		// Spot-Algo Endpoints
		spotTwapNewOrderRate: request.GetRateLimiterWithWeight(spotAlgoLimiter, 3000),

		// Futures-Algo Endpoints
		placeVPOrderRate:                request.GetRateLimiterWithWeight(placeVPOrderLimiter, 3000),
		placeTWAveragePriceNewOrderRate: request.GetRateLimiterWithWeight(placeVPOrderLimiter, 3000),

		// Mining Endpoints
		getMinersListRate:                     request.GetRateLimiterWithWeight(miningLimiter, 5),
		getEarningsListRate:                   request.GetRateLimiterWithWeight(miningLimiter, 5),
		extraBonusListRate:                    request.GetRateLimiterWithWeight(miningLimiter, 5),
		getHashrateRescaleDetailRate:          request.GetRateLimiterWithWeight(miningLimiter, 5),
		getHasrateRescaleRequestRate:          request.GetRateLimiterWithWeight(miningLimiter, 5),
		cancelHashrateResaleConfigurationRate: request.GetRateLimiterWithWeight(miningLimiter, 5),
		statisticsListRate:                    request.GetRateLimiterWithWeight(miningLimiter, 5),
		miningAccountListRate:                 request.GetRateLimiterWithWeight(miningLimiter, 5),
		miningAccountEarningRate:              request.GetRateLimiterWithWeight(miningLimiter, 5),

		// Staking Endpoints
		subscribeETHStakingRate:           request.GetRateLimiterWithWeight(stakingLimiter, 150),
		etherumStakingRedemptionRate:      request.GetRateLimiterWithWeight(stakingLimiter, 150),
		ethStakingHistoryRate:             request.GetRateLimiterWithWeight(stakingLimiter, 150),
		ethRedemptionHistoryRate:          request.GetRateLimiterWithWeight(stakingLimiter, 150),
		bethRewardDistributionHistoryRate: request.GetRateLimiterWithWeight(stakingLimiter, 150),
		currentETHStakingQuotaRate:        request.GetRateLimiterWithWeight(stakingLimiter, 150),
		getWBETHRateHistoryRate:           request.GetRateLimiterWithWeight(stakingLimiter, 150),
		ethStakingAccountRate:             request.GetRateLimiterWithWeight(stakingLimiter, 150),
		wrapBETHRate:                      request.GetRateLimiterWithWeight(stakingLimiter, 150),
		wbethWrapOrUnwrapHistoryRate:      request.GetRateLimiterWithWeight(stakingLimiter, 150),
		wbethRewardsHistoryRate:           request.GetRateLimiterWithWeight(stakingLimiter, 150),

		solStakingAccountRate:      request.GetRateLimiterWithWeight(stakingLimiter, 150),
		solStakingQuotaDetailsRate: request.GetRateLimiterWithWeight(stakingLimiter, 150),
		subscribeSOLStakingRate:    request.GetRateLimiterWithWeight(stakingLimiter, 150),
		redeemSOLRate:              request.GetRateLimiterWithWeight(stakingLimiter, 150),
		claimbBoostRewardsRate:     request.GetRateLimiterWithWeight(stakingLimiter, 150),
		solStakingHistoryRate:      request.GetRateLimiterWithWeight(stakingLimiter, 150),
		solRedemptionHistoryRate:   request.GetRateLimiterWithWeight(stakingLimiter, 150),
		bnsolRewardsHistoryRate:    request.GetRateLimiterWithWeight(stakingLimiter, 150),
		bnsolRateHistory:           request.GetRateLimiterWithWeight(stakingLimiter, 150),
		boostRewardsHistoryRate:    request.GetRateLimiterWithWeight(stakingLimiter, 150),
		unclaimedRewardsRate:       request.GetRateLimiterWithWeight(stakingLimiter, 150),

		createSubAccountRate:           request.NewRateLimitWithWeight(time.Second, 1, 1),
		getSubAccountRate:              request.NewRateLimitWithWeight(time.Second, 1, 1),
		enableFuturesForSubAccountRate: request.NewRateLimitWithWeight(time.Second, 1, 1),
		createAPIKeyForSubAccountRate:  request.NewRateLimitWithWeight(time.Second, 8, 1),

		// Futures
		futuresFundTransfersFetchRate:                          request.GetRateLimiterWithWeight(futuresFundTransfersFetchLimiter, 10),
		futureTickLevelOrderbookHistoricalDataDownloadLinkRate: request.GetRateLimiterWithWeight(futuresFundTransfersFetchLimiter, 200),

		// Simple Earn endpoints.
		simpleEarnProductsRate:                   request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getFlexibleSimpleEarnProductPositionRate: request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getSimpleEarnProductPositionRate:         request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		simpleAccountRate:                        request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getFlexibleSubscriptionRecordRate:        request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getLockedSubscriptionRecordsRate:         request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getRedemptionRecordRate:                  request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		getRewardHistoryRate:                     request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		setAutoSubscribeRate:                     request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		personalLeftQuotaRate:                    request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		subscriptionPreviewRate:                  request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),
		simpleEarnRateHistoryRate:                request.GetRateLimiterWithWeight(simpleEarnLimiter, 150),

		// Margin Account/Trade endpoints
		allCrossMarginFeeDataRate:                request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 5),
		marginAccountNewOrderRate:                request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 6),
		marginOCOOrderRate:                       request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 6),
		borrowRepayRecordsInMarginAccountRate:    request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		marginAccountCancelOrderRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getCrossMarginAccountDetailRate:          request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getPriceMarginIndexRate:                  request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getCrossMarginAccountOrderRate:           request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getMarginAccountsOpenOrdersRate:          request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getMarginAccountOCOOrderRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		marginAccountOpenOCOOrdersRate:           request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		marginAccountTradeListRate:               request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		marginAccountSummaryRate:                 request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		getIsolatedMarginAccountInfoRate:         request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		allIsolatedMarginSymbol:                  request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		allIsolatedMarginFeeDataRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 10),
		marginCurrentOrderCountUsageRate:         request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 20),
		marginMaxBorrowRate:                      request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 50),
		maxTransferOutRate:                       request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 50),
		marginAvailableInventoryRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 50),
		crossMarginCollateralRatioRate:           request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		getSmallLiabilityExchangeCoinListRate:    request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		getSmallLiabilityExchangeRate:            request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		marginHourlyInterestRate:                 request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		marginCapitalFlowRate:                    request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		marginTokensAndSymbolsDelistScheduleRate: request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 100),
		marginAccountsAllOrdersRate:              request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 200),
		getMarginAccountAllOCORate:               request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 200),
		deleteIsolatedMarginAccountRate:          request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 3000),
		enableIsolatedMarginAccountRate:          request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 300),
		marginAccountBorrowRepayRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 3000),
		adjustCrossMarginMaxLeverageRate:         request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 3000),
		smallLiabilityExchangeCoinListRate:       request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 3000),
		marginManualLiquidiationRate:             request.GetRateLimiterWithWeight(marginAccountTradeLimiter, 3000),

		// Crypto-Loan Endpoints.
		cryptoLoansIncomeHistory:            request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		cryptoRepayLoanRate:                 request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		adjustLTVRate:                       request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		checkCollateralRepayRate:            request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		cryptoLoanCustomizeMarginRate:       request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		borrowFlexibleRate:                  request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		repayFlexibleLoanHistoryRate:        request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		adjustFlexibleLoanRate:              request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		getLoanBorrowHistoryRate:            request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		repaymentHistoryRate:                request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		getLoanLTVAdjustmentHistoryRate:     request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		getLoanableAssetsDataRate:           request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		collateralAssetsDataRate:            request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		flexibleBorrowHistoryRate:           request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		flexibleLoanRepaymentHistoryRate:    request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		flexibleLoanCollateralRepaymentRate: request.GetRateLimiterWithWeight(cryptoLoanLimiter, 6000),
		flexibleLoanAdjustLTVRate:           request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		flexibleLoanAssetDataRate:           request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		flexibleLoanCollateralAssetRate:     request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
		getBorrowOngoingOrdersRate:          request.GetRateLimiterWithWeight(cryptoLoanLimiter, 300),
		getFlexibleLoanOngoingOrdersRate:    request.GetRateLimiterWithWeight(cryptoLoanLimiter, 300),
		flexibleLoanLiquidiationHistoryRate: request.GetRateLimiterWithWeight(cryptoLoanLimiter, 400),
	}
}

func openOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOpenOrdersAllRate
	}

	return spotOpenOrdersSpecificRate
}

func orderbookLimit(depth uint64) request.EndpointLimit {
	switch {
	case depth <= 100:
		return spotOrderbookDepth100Rate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}

	return spotOrderbookDepth5000Rate
}
