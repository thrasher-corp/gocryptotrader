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
	aggTradesRate
	listenKeyRate
	sapiDefaultRate
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

	futuresFundTransfersFetchRate
	futureTickLevelOrderbookHistoricalDataDownloadLinkRate
	fundAutoCollectionRate
	getAutoRepayFuturesStatusRate
	pmAssetLeverageRate

	getVIPLoanOngoingOrdersRate
	vipLoanRepayRate
	vipLoanRenewRate
	getVIPLoanRepaymentHistoryRate
	checkLockedValueVIPCollateralAccountRate
	vipLoanBorrowRate
	getVIPLoanableAssetsRate
	getCollateralAssetDataRate
	getApplicationStatusRate
	getVIPBorrowInterestRate

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
	flexibleBorrowHistoryRate
	repayFlexibleLoanHistoryRate
	flexibleLoanRepaymentHistoryRate
	adjustFlexibleLoanRate
	flexibleLoanAdjustLTVRate
	flexibleLoanAssetDataRate
	flexibleLoanCollateralAssetRate
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

	// SAPI default rate

	SapiDefaultRate        *rate.Limiter
	MarginAccountTradeRate *rate.Limiter
	CryptoLoanRate         *rate.Limiter

	WalletRate     *rate.Limiter
	SubAccountRate *rate.Limiter
	ConvertRate    *rate.Limiter

	SimpleEarnRate              *rate.Limiter
	NFTRate                     *rate.Limiter
	SpotRebateHistoryRate       *rate.Limiter
	PayTradeEndpointsRate       *rate.Limiter
	VIPLoanEndpointsRate        *rate.Limiter
	FiatDepositWithdrawHistRate *rate.Limiter

	ClassicPM                     *rate.Limiter
	SpotAlgoRate                  *rate.Limiter
	PlaceVPOrderRate              *rate.Limiter
	MiningRate                    *rate.Limiter
	FuturesFundTransfersFetchRate *rate.Limiter
	StakingRate                   *rate.Limiter
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
		queryPreventedMatchsWithRate,
		aggTradesRate,
		listenKeyRate:
		limiter, tokens = r.SpotRate, 2
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate,
		getOCOListRate:
		limiter, tokens = r.SpotRate, 4
	case spotHistoricalTradesRate,
		spotOrderbookDepth100Rate,
		getHashrateRescaleRate:
		limiter, tokens = r.SpotRate, 5

	case spotOrderbookDepth500Rate,
		getRecentTradesListRate,
		getOldTradeLookupRate:
		limiter, tokens = r.SpotRate, 25
	case accountTradeListRate,
		spotExchangeInfo,
		testNewOrderWithCommissionRate,
		getAllOCOOrdersRate:
		limiter, tokens = r.SpotRate, 20

	case getAutoRepayFuturesStatusRate:
		limiter, tokens = r.SpotRate, 30
	case spotOrderbookDepth1000Rate,
		pmAssetLeverageRate:
		limiter, tokens = r.SpotRate, 50
	case spotPriceChangeAllRate,
		getTickersMoreThan100Rate:
		limiter, tokens = r.SpotRate, 80

	case spotOrderbookDepth5000Rate:
		limiter, tokens = r.SpotRate, 250
	case spotOrderRate:
		limiter, tokens = r.SpotOrdersRate, 1
	case spotOpenOrdersSpecificRate:
		limiter, tokens = r.SpotOrdersRate, 6
	case getOpenOCOListRate:
		limiter, tokens = r.SpotRate, 6
	case spotOrderQueryRate:
		limiter, tokens = r.SpotOrdersRate, 4
	case spotAllOrdersRate:
		limiter, tokens = r.SpotOrdersRate, 20
	case spotOpenOrdersAllRate:
		limiter, tokens = r.SpotOrdersRate, 80

	case currentOrderCountUsageRate,
		getTickers100Rate:
		limiter, tokens = r.SpotRate, 40
	case getAllocationsRate,
		preventedMatchesByOrderIDRate,
		getCommissionRate:
		limiter, tokens = r.SpotRate, 20
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

		// /sapi/* endpoints

	case sapiDefaultRate:
		limiter, tokens = r.SapiDefaultRate, 1

		// Wallet Endpoints
	case userAssetsRate,
		busdConvertHistoryRate,
		busdConvertRate:
		limiter, tokens = r.WalletRate, 5
	case allCoinInfoRate,
		depositAddressesRate,
		dustTransferRate,
		assetDividendRecordRate,
		getDepositAddressListInNetworkRate,
		withdrawAddressListRate:
		limiter, tokens = r.WalletRate, 10
	case getUserWalletBalanceRate,
		getUserDelegationHistoryRate:
		limiter, tokens = r.WalletRate, 60
	case symbolDelistScheduleForSpotRate:
		limiter, tokens = r.WalletRate, 100
	case fundWithdrawalRate,
		cloudMiningPaymentAndRefundHistoryRate,
		autoConvertingStableCoinsRate:
		limiter, tokens = r.WalletRate, 600
	case userUniversalTransferRate:
		limiter, tokens = r.WalletRate, 900
	case dailyAccountSnapshotRate:
		limiter, tokens = r.WalletRate, 2400
	case withdrawalHistoryRate:
		limiter, tokens = r.WalletRate, 18000

		// Sub-Account Rate
	case getSubAccountStatusOnMarginOrFuturesRate,
		subAccountMarginAccountDetailRate,
		getSubAccountSummaryOfMarginAccountRate,
		getDetailSubAccountFuturesAccountRate,
		getFuturesPositionRiskOfSubAccountV1Rate,
		getFuturesSubAccountSummaryV2Rate: // 60
		limiter, tokens = r.SubAccountRate, 10
	case getSubAccountAssetRate,
		managedSubAccountTransferLogRate,
		managedSubAccountFuturesAssetDetailRate,
		getManagedSubAccountListRate,
		getSubAccountTransactionStatisticsRate: // 360
		limiter, tokens = r.SubAccountRate, 60
	case getManagedSubAccountSnapshotRate: // 2400
		limiter, tokens = r.SubAccountRate, 2400
	case ipRestrictionForSubAccountAPIKeyRate,
		deleteIPListForSubAccountAPIKeyRate,
		addIPRestrictionSubAccountAPIKey:
		limiter, tokens = r.SubAccountRate, 3000

		// NFT Endpoints
	case nftRate:
		limiter, tokens = r.NFTRate, 3000

		// Spot Rebate History
	case spotRebateHistoryRate:
		return r.SpotRebateHistoryRate.Wait(ctx)

		// Convert Rate
	case getAllConvertPairsRate:
		limiter, tokens = r.ConvertRate, 20
	case getOrderQuantityPrecisionPerAssetRate,
		orderStatusRate:
		limiter, tokens = r.ConvertRate, 100
	case sendQuoteRequestRate,
		cancelLimitOrderRate:
		limiter, tokens = r.ConvertRate, 200
	case acceptQuoteRate,
		placeLimitOrderRate:
		limiter, tokens = r.ConvertRate, 500
	case getLimitOpenOrdersRate,
		convertTradeFlowHistoryRate:
		limiter, tokens = r.ConvertRate, 3000

		// Pay Endpoints
	case payTradeEndpointsRate:
		return r.PayTradeEndpointsRate.Wait(ctx)

	case fiatDepositWithdrawHistRate:
		return r.FiatDepositWithdrawHistRate.Wait(ctx)

	// VIP Endpoints
	case getVIPLoanOngoingOrdersRate,
		getVIPLoanRepaymentHistoryRate,
		getVIPLoanableAssetsRate,
		getCollateralAssetDataRate,
		getApplicationStatusRate,
		getVIPBorrowInterestRate:
		limiter, tokens = r.VIPLoanEndpointsRate, 400
	case vipLoanRepayRate,
		vipLoanRenewRate,
		checkLockedValueVIPCollateralAccountRate,
		vipLoanBorrowRate:
		limiter, tokens = r.VIPLoanEndpointsRate, 6000

		// Classic Portfolio Margin
	case classicPMAccountInfoRate:
		limiter, tokens = r.ClassicPM, 5
	case changeAutoRepayFuturesStatusRate:
		limiter, tokens = r.ClassicPM, 30
	case classicPMCollateralRate,
		classicPMNegativeBalanceInterestHistory,
		pmAssetIndexPriceRate:
		limiter, tokens = r.ClassicPM, 50
	case fundCollectionByAssetRate:
		limiter, tokens = r.ClassicPM, 60
	case getClassicPMBankruptacyLoanAmountRate:
		limiter, tokens = r.ClassicPM, 500
	case fundAutoCollectionRate,
		transferBNBRate,
		repayFuturesNegativeBalanceRate:
		limiter, tokens = r.ClassicPM, 1500
	case repayClassicPMBankruptacyLoanRate:
		limiter, tokens = r.ClassicPM, 3000

		// Spot-Algo Endpoints
	case spotTwapNewOrderRate:
		return r.SpotAlgoRate.Wait(ctx)

		// Futures-Algo Endpoints
	case placeVPOrderRate,
		placeTWAveragePriceNewOrderRate:
		limiter, tokens = r.PlaceVPOrderRate, 3000

	// Mining Endpoints
	case getMinersListRate,
		getEarningsListRate,
		extraBonusListRate,
		getHashrateRescaleDetailRate,
		getHasrateRescaleRequestRate,
		cancelHashrateResaleConfigurationRate,
		statisticsListRate,
		miningAccountListRate,
		miningAccountEarningRate:
		limiter, tokens = r.MiningRate, 5

	// Staking Endpoints
	case subscribeETHStakingRate,
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
		limiter, tokens = r.StakingRate, 150

		// Futures
	case futuresFundTransfersFetchRate:
		limiter, tokens = r.FuturesFundTransfersFetchRate, 10
	case futureTickLevelOrderbookHistoricalDataDownloadLinkRate:
		limiter, tokens = r.FuturesFundTransfersFetchRate, 200

		// Simple Earn endpoints.
	case simpleEarnProductsRate,
		getFlexibleSimpleEarnProductPositionRate,
		getSimpleEarnProductPositionRate,
		simpleAccountRate,
		getFlexibleSubscriptionRecordRate,
		getLockedSubscriptionRecordsRate,
		getRedemptionRecordRate,
		getRewardHistoryRate,
		setAutoSubscribeRate,
		personalLeftQuotaRate,
		subscriptionPreviewRate,
		simpleEarnRateHistoryRate:
		limiter, tokens = r.SimpleEarnRate, 150

		// Margin Account/Trade endpoints
	case allCrossMarginFeeDataRate:
		limiter, tokens = r.MarginAccountTradeRate, 5
	case marginAccountNewOrderRate,
		marginOCOOrderRate:
		limiter, tokens = r.MarginAccountTradeRate, 6
	case borrowRepayRecordsInMarginAccountRate,
		marginAccountCancelOrderRate,
		getCrossMarginAccountDetailRate,
		getPriceMarginIndexRate,
		getCrossMarginAccountOrderRate,
		getMarginAccountsOpenOrdersRate,
		getMarginAccountOCOOrderRate,
		marginAccountOpenOCOOrdersRate,
		marginAccountTradeListRate,
		marginAccountSummaryRate,
		getIsolatedMarginAccountInfoRate,
		allIsolatedMarginSymbol,
		allIsolatedMarginFeeDataRate:
		limiter, tokens = r.MarginAccountTradeRate, 10
	case marginCurrentOrderCountUsageRate:
		limiter, tokens = r.MarginAccountTradeRate, 20
	case marginMaxBorrowRate,
		maxTransferOutRate,
		marginAvailableInventoryRate:
		limiter, tokens = r.MarginAccountTradeRate, 50
	case crossMarginCollateralRatioRate,
		getSmallLiabilityExchangeCoinListRate,
		getSmallLiabilityExchangeRate,
		marginHourlyInterestRate,
		marginCapitalFlowRate,
		marginTokensAndSymbolsDelistScheduleRate:
		limiter, tokens = r.MarginAccountTradeRate, 100
	case marginAccountsAllOrdersRate,
		getMarginAccountAllOCORate:
		limiter, tokens = r.MarginAccountTradeRate, 200
	case deleteIsolatedMarginAccountRate,
		enableIsolatedMarginAccountRate:
		limiter, tokens = r.MarginAccountTradeRate, 300
	case marginAccountBorrowRepayRate,
		adjustCrossMarginMaxLeverageRate,
		smallLiabilityExchangeCoinListRate,
		marginManualLiquidiationRate:
		limiter, tokens = r.MarginAccountTradeRate, 3000

		// Crypto-Loan Endpoints.

	case cryptoLoansIncomeHistory,
		cryptoRepayLoanRate,
		adjustLTVRate,
		checkCollateralRepayRate,
		cryptoLoanCustomizeMarginRate,
		borrowFlexibleRate,
		repayFlexibleLoanHistoryRate,
		adjustFlexibleLoanRate:
		limiter, tokens = r.CryptoLoanRate, 6000
	case getLoanBorrowHistoryRate,
		repaymentHistoryRate,
		getLoanLTVAdjustmentHistoryRate,
		getLoanableAssetsDataRate,
		collateralAssetsDataRate,
		flexibleBorrowHistoryRate,
		flexibleLoanRepaymentHistoryRate,
		flexibleLoanAdjustLTVRate,
		flexibleLoanAssetDataRate,
		flexibleLoanCollateralAssetRate:
		limiter, tokens = r.CryptoLoanRate, 400
	case getBorrowOngoingOrdersRate,
		getFlexibleLoanOngoingOrdersRate:
		limiter, tokens = r.CryptoLoanRate, 300

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

		// Sapi Endpoints
		//
		// Endpoints are marked according to IP or UID limit and their corresponding weight value.
		// Each endpoint with IP limits has an independent 12000 per minute limit, or per second limit if specified explicitly
		// Each endpoint with UID limits has an independent 180000 per minute limit, or per second limit if specified explicitly
		SapiDefaultRate:        request.NewRateLimit(time.Second, 110),
		MarginAccountTradeRate: request.NewRateLimit(time.Second, 16917),

		WalletRate:     request.NewRateLimit(time.Second, 23995),
		SubAccountRate: request.NewRateLimit(time.Second, 11820),
		ConvertRate:    request.NewRateLimit(time.Second, 7620),

		SimpleEarnRate:              request.NewRateLimit(time.Second, 2700),
		NFTRate:                     request.NewRateLimit(time.Second, 12000),
		SpotRebateHistoryRate:       request.NewRateLimit(time.Second, 12000),
		PayTradeEndpointsRate:       request.NewRateLimit(time.Second, 3000),
		VIPLoanEndpointsRate:        request.NewRateLimit(time.Second, 26600),
		FiatDepositWithdrawHistRate: request.NewRateLimit(time.Second, 90000),

		ClassicPM:                     request.NewRateLimit(time.Second, 5145),
		SpotAlgoRate:                  request.NewRateLimit(time.Second, 3000),
		PlaceVPOrderRate:              request.NewRateLimit(time.Second, 6000),
		FuturesFundTransfersFetchRate: request.NewRateLimit(time.Second, 210),
		MiningRate:                    request.NewRateLimit(time.Second, 55),
		StakingRate:                   request.NewRateLimit(time.Second, 1500),
		CryptoLoanRate:                request.NewRateLimit(time.Second, 52900),
	}
}

func bestPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOrderbookTickerAllRate
	}

	return spotDefaultRate
}

// GetRateLimits returns the rate limit for the exchange
func GetRateLimits() request.RateLimitDefinitions {
	spotDefaultLimiter := request.NewRateLimit(spotInterval, spotRequestRate)
	spotOrderLimiter := request.NewRateLimit(spotOrderInterval, spotOrderRequestRate)
	usdMarginedFuturesLimiter := request.NewRateLimit(uFuturesInterval, uFuturesRequestRate)
	usdMarginedFuturesOrdersLimiter := request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate)
	coinMarginedFuturesLimiter := request.NewRateLimit(cFuturesInterval, cFuturesRequestRate)
	coinMarginedFuturesOrdersLimiter := request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate)

	return request.RateLimitDefinitions{
		spotDefaultRate:                 request.GetRateLimiterWithWeight(spotDefaultLimiter, 1),
		spotOrderbookTickerAllRate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 2),
		spotSymbolPriceAllRate:          request.GetRateLimiterWithWeight(spotDefaultLimiter, 2),
		spotHistoricalTradesRate:        request.GetRateLimiterWithWeight(spotDefaultLimiter, 5),
		spotOrderbookDepth500Rate:       request.GetRateLimiterWithWeight(spotDefaultLimiter, 5),
		spotOrderbookDepth1000Rate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 10),
		spotAccountInformationRate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 10),
		spotExchangeInfo:                request.GetRateLimiterWithWeight(spotDefaultLimiter, 10),
		spotPriceChangeAllRate:          request.GetRateLimiterWithWeight(spotDefaultLimiter, 40),
		spotOrderbookDepth5000Rate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 50),
		spotOrderRate:                   request.GetRateLimiterWithWeight(spotOrderLimiter, 1),
		spotOrderQueryRate:              request.GetRateLimiterWithWeight(spotOrderLimiter, 2),
		spotOpenOrdersSpecificRate:      request.GetRateLimiterWithWeight(spotOrderLimiter, 3),
		spotAllOrdersRate:               request.GetRateLimiterWithWeight(spotOrderLimiter, 10),
		spotOpenOrdersAllRate:           request.GetRateLimiterWithWeight(spotOrderLimiter, 40),
		uFuturesDefaultRate:             request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
		uFuturesKline100Rate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
		uFuturesOrderbook50Rate:         request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesKline500Rate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesKline1000Rate:           request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 10),
		uFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 10),
		uFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 20),
		uFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 20),
		uFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 40),
		uFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 1),
		uFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesGetAllOrdersRate:        request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesCountdownCancelRate:     request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 10),
		uFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 30),
		uFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 50),
		cFuturesDefaultRate:             request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 1),
		cFuturesKline500Rate:            request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 2),
		cFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 2),
		cFuturesKline1000Rate:           request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 5),
		cFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 5),
		cFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 10),
		cFuturesIndexMarkPriceRate:      request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 10),
		cFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 20),
		cFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 20),
		cFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 40),
		cFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 50),
		cFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 1),
		cFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesCancelAllOrdersRate:     request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 40),
		cFuturesOrderbook50Rate:         request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 2),
		cFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		uFuturesMultiAssetMarginRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 30),
		uFuturesSetMultiAssetMarginRate: request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
	}
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
