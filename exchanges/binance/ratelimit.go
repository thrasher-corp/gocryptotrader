package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	spotInterval    = time.Minute
	spotRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval        = 10 * time.Second
	spotOrderRequestRate     = 100
	cFuturesInterval         = time.Minute
	cFuturesRequestRate      = 6000
	cFuturesOrderInterval    = time.Minute
	cFuturesOrderRequestRate = 1200
	uFuturesInterval         = time.Minute
	uFuturesRequestRate      = 2400
	uFuturesOrderInterval    = time.Minute
	uFuturesOrderRequestRate = 1200
	portfolioMarginRate      = 1200
	portfolioMarginInterval  = time.Minute
)

// Binance Spot rate limits
const (
	spotDefaultRate request.EndpointLimit = iota
	walletSystemStatus
	allCoinInfoRate
	dailyAccountSnapshotRate
	fundWithdrawalRate
	withdrawalHistoryRate
	spotExchangeInfo
	spotHistoricalTradesRate
	spotOrderbookDepth100Rate
	spotOrderbookDepth500Rate
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
	spotOpenOrdersAllRate
	allCrossMarginFeeDataRate
	allIsolatedMarginFeeDataRate
	marginCurrentOrderCountUsageRate
	depositAddressesRate
	assetDividendRecordRate
	userAssetsRate
	getMinersListRate
	getEarningsListRate
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
	crossMarginCollateralRatioRate
	smallLiabilityExchCoinListRate
	marginHourlyInterestRate
	marginCapitalFlowRate
	marginTokensAndSymbolsDelistScheduleRate
	getSubAccountAssetRate
	getSubAccountStatusOnMarginOrFuturesRate
	subAccountMarginAccountDetailRate
	getSubAccountSummaryOfMarginAccountRate
	getDetailSubAccountFuturesAccountRate
	getFuturesPositionRiskOfSubAccountV1Rate
	getFuturesSubAccountSummaryV2Rate
	getManagedSubAccountSnapshotRate
	getCrossMarginAccountDetailRate
	getCrossMarginAccountOrderRate
	getMarginAccountsOpenOrdersRate
	marginAccountsAllOrdersRate
	marginAccountOpenOCOOrdersRate
	marginAccountTradeListRate
	marginMaxBorrowRate
	maxTransferOutRate
	marginAccountSummaryRate
	isolatedMarginAccountInfoRate
	allIsolatedMarginSymbol
	ocoOrderRate
	getMarginAccountAllOCORate

	simpleEarnProductsRate
	getSimpleEarnProductPositionRate
	simpleAccountRate
	getFlexibleSubscriptionRecordRate
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

	futuresFundTransfersFetchRate
	futureTickLevelOrderbookHistoricalDataDownloadLinkRate
	pmAssetIndexPriceRate
	fundAutoCollectionRate
	transferBNBRate
	changeAutoRepayFuturesStatusRate
	getAutoRepayFuturesStatusRate
	repayFuturesNegativeBalanceRate
	pmAssetLeverageRate

	vipLoanOngoingOrdersRate
	getVIPLoanRepaymentHistoryRate
	checkLockedValueVIPCollateralAccountRate

	getAllConvertPairsRate
	getOrderQuantityPrecisionPerAssetRate

	// planceVOOrderRate
	classicPMAccountInfoRate
	classicPMCollateralRate
	classicPMNegativeBalanceInterestHistory
	fundCollectionByAssetRate

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
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate            *rate.Limiter
	SpotOrdersRate      *rate.Limiter
	UFuturesRate        *rate.Limiter
	UFuturesOrdersRate  *rate.Limiter
	CFuturesRate        *rate.Limiter
	CFuturesOrdersRate  *rate.Limiter
	EOptionsRate        *rate.Limiter
	EOptionsOrderRate   *rate.Limiter
	PortfolioMarginRate *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case spotDefaultRate:
		limiter, tokens = r.SpotRate, 1
	case spotBookTickerRate,
		spotSymbolPriceRate,
		getAggregateTradeListRate,
		getKlineRate,
		getCurrentAveragePriceRate,
		get24HrTickerPriceChangeStatisticsRate,
		getTickers20Rate,
		queryPreventedMatchsWithRate:
		limiter, tokens = r.SpotRate, 2
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate:
		limiter, tokens = r.SpotRate, 4
	case spotHistoricalTradesRate,
		spotOrderbookDepth100Rate,
		marginMaxBorrowRate,
		userAssetsRate,
		getMinersListRate,
		getEarningsListRate,
		getHashrateRescaleRate,
		getHashrateRescaleDetailRate,
		getHasrateRescaleRequestRate,
		cancelHashrateResaleConfigurationRate,
		statisticsListRate,
		miningAccountListRate,
		miningAccountEarningRate,
		classicPMAccountInfoRate:
		limiter, tokens = r.SpotRate, 5

	case spotOrderbookDepth500Rate,
		getRecentTradesListRate,
		getOldTradeLookupRate:
		limiter, tokens = r.SpotRate, 25
	case spotAccountInformationRate,
		accountTradeListRate,
		spotExchangeInfo,
		getAllConvertPairsRate:
		limiter, tokens = r.SpotRate, 20

	case getAutoRepayFuturesStatusRate:
		limiter, tokens = r.SpotRate, 30
	case spotOrderbookDepth1000Rate,
		maxTransferOutRate,
		marginAccountSummaryRate,
		classicPMCollateralRate,
		classicPMNegativeBalanceInterestHistory,
		pmAssetIndexPriceRate,
		pmAssetLeverageRate:
		limiter, tokens = r.SpotRate, 50
	case walletSystemStatus:
		limiter, tokens = r.SpotRate, 1
	case dailyAccountSnapshotRate:
		limiter, tokens = r.SpotRate, 2400
	case fundWithdrawalRate:
		limiter, tokens = r.SpotRate, 600
	case spotPriceChangeAllRate,
		getTickersMoreThan100Rate:
		limiter, tokens = r.SpotRate, 80

	case spotOrderbookDepth5000Rate:
		limiter, tokens = r.SpotRate, 250
	case spotOrderRate:
		limiter, tokens = r.SpotOrdersRate, 1
	case spotOrderQueryRate:
		limiter, tokens = r.SpotOrdersRate, 4
	case spotOpenOrdersSpecificRate:
		limiter, tokens = r.SpotOrdersRate, 3
	case spotAllOrdersRate:
		limiter, tokens = r.SpotOrdersRate, 10
	case spotOpenOrdersAllRate:
		limiter, tokens = r.SpotOrdersRate, 40
	case allCrossMarginFeeDataRate:
		limiter, tokens = r.SpotRate, 5

	case depositAddressesRate,
		assetDividendRecordRate,
		withdrawalHistoryRate,
		allCoinInfoRate,
		isolatedMarginAccountInfoRate,
		getDepositAddressListInNetworkRate,
		allIsolatedMarginSymbol,
		allIsolatedMarginFeeDataRate,
		getSubAccountStatusOnMarginOrFuturesRate,
		subAccountMarginAccountDetailRate,
		getSubAccountSummaryOfMarginAccountRate,
		getDetailSubAccountFuturesAccountRate,
		getFuturesPositionRiskOfSubAccountV1Rate,
		getFuturesSubAccountSummaryV2Rate,
		getCrossMarginAccountDetailRate,
		getCrossMarginAccountOrderRate,
		getMarginAccountsOpenOrdersRate,
		marginAccountOpenOCOOrdersRate,
		marginAccountTradeListRate,
		borrowRepayRecordsInMarginAccountRate,
		getPriceMarginIndexRate,
		futuresFundTransfersFetchRate:
		limiter, tokens = r.SpotRate, 10

	case marginCurrentOrderCountUsageRate:
		limiter, tokens = r.SpotRate, 20
	case getUserWalletBalanceRate,
		getUserDelegationHistoryRate,
		getSubAccountAssetRate,
		fundCollectionByAssetRate:
		limiter, tokens = r.SpotRate, 60
	case symbolDelistScheduleForSpotRate,
		crossMarginCollateralRatioRate,
		smallLiabilityExchCoinListRate,
		marginHourlyInterestRate,
		marginCapitalFlowRate,
		marginTokensAndSymbolsDelistScheduleRate,
		getOrderQuantityPrecisionPerAssetRate:
		limiter, tokens = r.SpotRate, 100
	case simpleEarnProductsRate,
		getSimpleEarnProductPositionRate,
		simpleAccountRate,
		getFlexibleSubscriptionRecordRate,
		getLockedSubscriptionRecordsRate,
		getRedemptionRecordRate,
		getRewardHistoryRate,
		setAutoSubscribeRate,
		personalLeftQuotaRate,
		subscriptionPreviewRate,
		simpleEarnRateHistoryRate,
		etherumStakingRedemptionRate,
		ethStakingHistoryRate,
		ethRedemptionHistoryRate,
		bethRewardDistributionHistoryRate,
		currentETHStakingQuotaRate,
		getWBETHRateHistoryRate,
		ethStakingAccountRate,
		wrapBETHRate,
		wbethWrapOrUnwrapHistoryRate,
		wbethRewardsHistoryRate:
		limiter, tokens = r.SpotRate, 150
	case fundAutoCollectionRate,
		transferBNBRate,
		changeAutoRepayFuturesStatusRate,
		repayFuturesNegativeBalanceRate:
		limiter, tokens = r.SpotRate, 1500
	case marginAccountsAllOrdersRate,
		futureTickLevelOrderbookHistoricalDataDownloadLinkRate,
		getMarginAccountAllOCORate:
		limiter, tokens = r.SpotRate, 200
	case ocoOrderRate:
		limiter, tokens = r.SpotOrdersRate, 2
	case getManagedSubAccountSnapshotRate:
		limiter, tokens = r.SpotRate, 2400
	case currentOrderCountUsageRate,
		getTickers100Rate:
		limiter, tokens = r.SpotRate, 40
	case vipLoanOngoingOrdersRate,
		getVIPLoanRepaymentHistoryRate:
		limiter, tokens = r.SpotRate, 400
	case checkLockedValueVIPCollateralAccountRate:
		limiter, tokens = r.SpotRate, 6000
	case getAllocationsRate,
		preventedMatchesByOrderIDRate,
		getCommissionRate:
		limiter, tokens = r.SpotRate, 20
	case marginAccountNewOrderRate:
		limiter, tokens = r.SpotOrdersRate, 6
	case marginAccountCancelOrderRate:
		limiter, tokens = r.SpotOrdersRate, 10
	case uFuturesDefaultRate,
		uFuturesKline100Rate:
		limiter, tokens = r.UFuturesRate, 1
	case uFuturesOrderbook50Rate,
		uFuturesKline500Rate,
		uFuturesOrderbookTickerAllRate:
		limiter, tokens = r.UFuturesRate, 2
	case uFuturesOrderbook100Rate,
		uFuturesKline1000Rate,
		uFuturesAccountInformationRate:
		limiter, tokens = r.UFuturesRate, 5
	case uFuturesOrderbook500Rate,
		uFuturesKlineMaxRate:
		limiter, tokens = r.UFuturesRate, 10
	case uFuturesOrderbook1000Rate,
		uFuturesHistoricalTradesRate:
		limiter, tokens = r.UFuturesRate, 20
	case uFuturesTickerPriceHistoryRate:
		limiter, tokens = r.UFuturesRate, 40
	case uFuturesOrdersDefaultRate:
		limiter, tokens = r.UFuturesOrdersRate, 1
	case uFuturesBatchOrdersRate,
		uFuturesGetAllOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 5
	case uFuturesCountdownCancelRate:
		limiter, tokens = r.UFuturesOrdersRate, 10
	case uFuturesCurrencyForceOrdersRate,
		uFuturesSymbolOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 20
	case uFuturesIncomeHistoryRate:
		limiter, tokens = r.UFuturesOrdersRate, 30
	case uFuturesPairOrdersRate,
		uFuturesGetAllOpenOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 40
	case uFuturesAllForceOrdersRate:
		limiter, tokens = r.UFuturesOrdersRate, 50
	case cFuturesKline100Rate:
		limiter, tokens = r.CFuturesRate, 1
	case cFuturesKline500Rate,
		cFuturesOrderbookTickerAllRate:
		limiter, tokens = r.CFuturesRate, 2
	case cFuturesKline1000Rate,
		cFuturesAccountInformationRate:
		limiter, tokens = r.CFuturesRate, 5
	case cFuturesKlineMaxRate,
		cFuturesIndexMarkPriceRate:
		limiter, tokens = r.CFuturesRate, 10
	case cFuturesHistoricalTradesRate,
		cFuturesCurrencyForceOrdersRate:
		limiter, tokens = r.CFuturesRate, 20
	case cFuturesTickerPriceHistoryRate:
		limiter, tokens = r.CFuturesRate, 40
	case cFuturesAllForceOrdersRate:
		limiter, tokens = r.CFuturesRate, 50
	case cFuturesOrdersDefaultRate:
		limiter, tokens = r.CFuturesOrdersRate, 1
	case cFuturesBatchOrdersRate,
		cFuturesGetAllOpenOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 5
	case cFuturesCancelAllOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 10
	case cFuturesIncomeHistoryRate,
		cFuturesSymbolOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 20
	case cFuturesPairOrdersRate:
		limiter, tokens = r.CFuturesOrdersRate, 40
	case cFuturesOrderbook50Rate:
		limiter, tokens = r.CFuturesRate, 2
	case cFuturesOrderbook100Rate:
		limiter, tokens = r.CFuturesRate, 5
	case cFuturesOrderbook500Rate:
		limiter, tokens = r.CFuturesRate, 10
	case cFuturesOrderbook1000Rate:
		limiter, tokens = r.CFuturesRate, 20
	case cFuturesDefaultRate:
		limiter, tokens = r.CFuturesRate, 1
	case uFuturesMultiAssetMarginRate:
		limiter, tokens = r.UFuturesRate, 30
	case uFuturesSetMultiAssetMarginRate:
		limiter, tokens = r.UFuturesRate, 1

		// Options Rate Limits
	case optionsDefaultRate:
		limiter, tokens = r.EOptionsRate, 1
	case optionsRecentTradesRate,
		optionsMarkPriceRate,
		optionsAllTickerPriceStatistics,
		optionsPositionInformationRate,
		optionsAccountTradeListRate,
		optionsUserExerciseRecordRate,
		optionsDownloadIDForOptionTrasactionHistoryRate,
		optionsGetTransHistoryDownloadLinkByIDRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsHistoricalTradesRate:
		limiter, tokens = r.EOptionsRate, 20
	case optionsHistoricalExerciseRecordsRate,
		optionsMarginAccountInfoRate:
		limiter, tokens = r.EOptionsRate, 3
	case optionsAccountInfoRate,
		optionsBatchOrderRate:
		limiter, tokens = r.EOptionsOrderRate, 5
	case optionsDefaultOrderRate:
		limiter, tokens = r.EOptionsOrderRate, 1
	case optionsAllQueryOpenOrdersRate:
		limiter, tokens = r.EOptionsOrderRate, 40
	case optionsGetOrderHistory:
		limiter, tokens = r.EOptionsOrderRate, 3
	case optionsAutoCancelAllOpenOrdersHeartbeatRate:
		limiter, tokens = r.EOptionsRate, 10

	case pmDefaultRate:
		limiter, tokens = r.PortfolioMarginRate, 1
	case pmMarginAccountLoanAndRepayRate,
		pmAllMarginAccountOrdersRate,
		pmGetMarginAccountsAllOCOOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 100
	case pmCancelMarginAccountOpenOrdersOnSymbolRate,
		pmGetAllUMOrdersRate,
		pmGetMarginAccountOrderRate,
		pmCurrentMarginOpenOrderRate,
		pmGetMarginAccountOCORate,
		pmGetMarginAccountsOpenOCOOrdersRate,
		pmGetMarginAccountTradeListRate,
		pmMarginMaxBorrowRate,
		pmGetMarginMaxWithdrawalRate,
		pmGetUMPositionInformationRate,
		pmGetUMAccountTradeListRate,
		pmGetUMAccountDetailRate,
		pmGetCMAccountDetailRate,
		pmGetUMPositionADLQuantileEstimationRate,
		pmGetCMPositionADLQuantileEstimationRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmCancelMarginAccountOCORate:
		limiter, tokens = r.PortfolioMarginRate, 2
	case pmRetrieveAllUMOpenOrdersForAllSymbolRate,
		pmRetrieveAllCMOpenOrdersForAllSymbolRate,
		pmAllCMOrderWithoutSymbolRate,
		pmUMOpenConditionalOrdersRate,
		pmAllUMConditionalOrdersWithoutSymbolRate,
		pmAllCMOpenConditionalOrdersWithoutSymbolRate,
		pmAllCMConditionalOrderWithoutSymbolRate,
		pmGetCMAccountTradeListWithPairRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmAllCMOrderWithSymbolRate,
		pmGetAccountBalancesRate,
		pmGetAccountInformationRate,
		pmGetCMAccountTradeListWithSymbolRate,
		pmGetUserUMForceOrdersWithSymbolRate,
		pmGetUserCMForceOrdersWithSymbolRate,
		pmGetUMUserCommissionRate,
		pmGetCMUserCommissionRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetUMCurrentPositionModeRate,
		pmGetCMCurrentPositionModeRate,
		pmFundCollectionByAssetRate,
		pmGetUMIncomeHistoryRate,
		pmGetCMIncomeHistoryRate,
		pmGetAutoRepayFuturesStatusRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmGetUserUMForceOrdersWithoutSymbolRate,
		pmGetUserCMForceOrdersWithoutSymbolRate,
		pmGetPortfolioMarginNegativeBalanceInterestHistoryRate:
		limiter, tokens = r.PortfolioMarginRate, 50
	case pmUMTradingQuantitativeRulesIndicatorsRate,
		pmGetMarginLoanRecordRate,
		pmGetMarginRepayRecordRate:
		limiter, tokens = r.PortfolioMarginRate, 10
	case pmFundAutoCollectionRate,
		pmBNBTransferRate,
		pmChangeAutoRepayFuturesStatusRate,
		pmRepayFuturesNegativeBalanceRate:
		limiter, tokens = r.PortfolioMarginRate, 750
	default:
		limiter, tokens = r.SpotRate, 1
	}

	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		// Cancel all potential reservations to free up rate limiter if deadline
		// is exceeded.
		for x := range reserves {
			reserves[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	time.Sleep(finalDelay)
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		SpotRate:            request.NewRateLimit(spotInterval, spotRequestRate),
		SpotOrdersRate:      request.NewRateLimit(spotOrderInterval, spotOrderRequestRate),
		UFuturesRate:        request.NewRateLimit(uFuturesInterval, uFuturesRequestRate),
		UFuturesOrdersRate:  request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate),
		CFuturesRate:        request.NewRateLimit(cFuturesInterval, cFuturesRequestRate),
		CFuturesOrdersRate:  request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate),
		EOptionsRate:        request.NewRateLimit(time.Minute, 400),
		EOptionsOrderRate:   request.NewRateLimit(time.Minute, 100),
		PortfolioMarginRate: request.NewRateLimit(portfolioMarginInterval, portfolioMarginRate),
	}
}

func bestPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOrderbookTickerAllRate
	}

	return spotDefaultRate
}

func openOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOpenOrdersAllRate
	}

	return spotOpenOrdersSpecificRate
}

func orderbookLimit(depth int64) request.EndpointLimit {
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
