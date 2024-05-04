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
	cloudMiningPaymentAndRefundHistoryRate
	autoConvertingStableCoinsRate
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

	SapiDefaultRate                          *rate.Limiter
	AllCoinInfoRate                          *rate.Limiter
	DailyAccountSnapshotRate                 *rate.Limiter
	FundWithdrawalRate                       *rate.Limiter
	WithdrawalHistoryRate                    *rate.Limiter
	DepositAddressesRate                     *rate.Limiter
	DustTransferRate                         *rate.Limiter
	AssetDividendRecordRate                  *rate.Limiter
	UserUniversalTransferRate                *rate.Limiter
	UserAssetsRate                           *rate.Limiter
	BUSDConvertHistoryRate                   *rate.Limiter
	CloudMiningPaymentAndRefundHistoryRate   *rate.Limiter
	AutoConvertingStableCoinsRate            *rate.Limiter
	GetDepositAddressListInNetworkRate       *rate.Limiter
	GetUserWalletBalanceRate                 *rate.Limiter
	GetUserDelegationHistoryRate             *rate.Limiter
	SymbolDelistScheduleForSpotRate          *rate.Limiter
	WithdrawAddressListRate                  *rate.Limiter
	GetSubAccountAssetRate                   *rate.Limiter
	GetSubAccountStatusOnMarginOrFuturesRate *rate.Limiter
	SubAccountMarginAccountDetailRate        *rate.Limiter
	SubAccountSummaryOfMarginAccountRate     *rate.Limiter
	DetailSubAccountFuturesAccountRate       *rate.Limiter
	FuturesPositionRiskOfSubAccountV1Rate    *rate.Limiter
	FuturesSubAccountSummaryV2Rate           *rate.Limiter
	IPRestrictionForSubAccountAPIKeyRate     *rate.Limiter
	DeleteIPListForSubAccountAPIKeyRate      *rate.Limiter
	AddIPRestrictionSubAccountAPIKeyRate     *rate.Limiter
	ManagedSubAccountSnapshotRate            *rate.Limiter
	ManagedSubAccountTransferLogRate         *rate.Limiter
	ManagedSubAccountFuturesAssetDetailRate  *rate.Limiter
	ManagedSubAccountListRate                *rate.Limiter
	SubAccountTransactionStatisticsRate      *rate.Limiter
	MarginAccountBorrowRepayRate             *rate.Limiter
	BorrowRepayRecordsInMarginAccountRate    *rate.Limiter
	PriceMarginIndexRate                     *rate.Limiter
	MarginAccountNewOrderRate                *rate.Limiter
	MarginAccountCancelOrderRate             *rate.Limiter
	AdjustCrossMarginMaxLeverageRate         *rate.Limiter
	CrossMarginAccountDetailRate             *rate.Limiter
	CrossMarginAccountOrderRate              *rate.Limiter
	MarginAccountsOpenOrdersRate             *rate.Limiter
	MarginAccountsAllOrdersRate              *rate.Limiter
	MarginOCOOrderRate                       *rate.Limiter
	MarginAccountOCOOrderRate                *rate.Limiter
	MarginAccountAllOCORate                  *rate.Limiter
	MarginAccountOpenOCOOrdersRate           *rate.Limiter
	MarginAccountTradeListRate               *rate.Limiter
	MarginMaxBorrowRate                      *rate.Limiter
	MaxTransferOutRate                       *rate.Limiter
	MarginAccountSummaryRate                 *rate.Limiter
	IsolatedMarginAccountInfoRate            *rate.Limiter
	DeleteIsolatedMarginAccountRate          *rate.Limiter
	EnableIsolatedMarginAccountRate          *rate.Limiter
	AllIsolatedMarginSymbol                  *rate.Limiter
	AllCrossMarginFeeDataRate                *rate.Limiter
	AllIsolatedMarginFeeDataRate             *rate.Limiter
	MarginCurrentOrderCountUsageRate         *rate.Limiter
	CrossMarginCollateralRatioRate           *rate.Limiter
	SmallLiabilityExchangeCoinListRate       *rate.Limiter
	GetSmallLiabilityExchangeCoinListRate    *rate.Limiter
	GetSmallLiabilityExchangeRate            *rate.Limiter
	MarginHourlyInterestRate                 *rate.Limiter
	MarginCapitalFlowRate                    *rate.Limiter
	MarginTokensAndSymbolsDelistScheduleRate *rate.Limiter
	MarginAvailableInventoryRate             *rate.Limiter
	MarginManualLiquidiationRate             *rate.Limiter
	SimpleEarnProductsRate                   *rate.Limiter
	FlexibleSimpleEarnProductPositionRate    *rate.Limiter
	GetSimpleEarnProductPositionRate         *rate.Limiter
	SimpleAccountRate                        *rate.Limiter
	GetFlexibleSubscriptionRecordRate        *rate.Limiter
	NFTRate                                  *rate.Limiter
	SpotRebateHistoryRate                    *rate.Limiter
	ConvertTradeFlowHistoryRate              *rate.Limiter
	LimitOpenOrdersRate                      *rate.Limiter
	CancelLimitOrderRate                     *rate.Limiter
	PlaceLimitOrderRate                      *rate.Limiter
	OrderStatusRate                          *rate.Limiter
	AcceptQuoteRate                          *rate.Limiter
	SendQuoteRequestRate                     *rate.Limiter
	GetOrderQuantityPrecisionPerAssetRate    *rate.Limiter
	GetAllConvertPairsRate                   *rate.Limiter
	PayTradeEndpointsRate                    *rate.Limiter
	VIPLoanEndpointsRate                     *rate.Limiter
	FiatDepositWithdrawHistRate              *rate.Limiter
	ClassicPM                                *rate.Limiter
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
		aggTradesRate:
		limiter, tokens = r.SpotRate, 2
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate,
		getOCOListRate:
		limiter, tokens = r.SpotRate, 4
	case spotHistoricalTradesRate,
		spotOrderbookDepth100Rate,
		getMinersListRate,
		getEarningsListRate,
		getHashrateRescaleRate,
		getHashrateRescaleDetailRate,
		getHasrateRescaleRequestRate,
		cancelHashrateResaleConfigurationRate,
		statisticsListRate,
		miningAccountListRate,
		miningAccountEarningRate:
		limiter, tokens = r.SpotRate, 5

	case spotOrderbookDepth500Rate,
		getRecentTradesListRate,
		getOldTradeLookupRate:
		limiter, tokens = r.SpotRate, 25
	case spotAccountInformationRate,
		accountTradeListRate,
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

	case futuresFundTransfersFetchRate:
		limiter, tokens = r.SpotRate, 10

	case getLockedSubscriptionRecordsRate,
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
	case futureTickLevelOrderbookHistoricalDataDownloadLinkRate:
		limiter, tokens = r.SpotRate, 200
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
	case allCoinInfoRate:
		return r.AllCoinInfoRate.Wait(ctx)
	case dailyAccountSnapshotRate:
		return r.DailyAccountSnapshotRate.Wait(ctx)
	case fundWithdrawalRate: // uid
		return r.FundWithdrawalRate.Wait(ctx)
	case withdrawalHistoryRate:
		return r.WithdrawalHistoryRate.Wait(ctx)
	case depositAddressesRate:
		return r.DepositAddressesRate.Wait(ctx)
	case dustTransferRate:
		return r.DustTransferRate.Wait(ctx)
	case assetDividendRecordRate:
		return r.AssetDividendRecordRate.Wait(ctx)
	case userUniversalTransferRate:
		return r.UserUniversalTransferRate.Wait(ctx)
	case userAssetsRate:
		return r.UserAssetsRate.Wait(ctx)
	case busdConvertHistoryRate:
		return r.BUSDConvertHistoryRate.Wait(ctx)
	case cloudMiningPaymentAndRefundHistoryRate:
		return r.CloudMiningPaymentAndRefundHistoryRate.Wait(ctx)
	case autoConvertingStableCoinsRate:
		limiter, tokens = r.AutoConvertingStableCoinsRate, 600
	case getDepositAddressListInNetworkRate:
		return r.GetDepositAddressListInNetworkRate.Wait(ctx)
	case getUserWalletBalanceRate:
		return r.GetUserWalletBalanceRate.Wait(ctx)
	case getUserDelegationHistoryRate:
		return r.GetUserDelegationHistoryRate.Wait(ctx)
	case symbolDelistScheduleForSpotRate:
		return r.SymbolDelistScheduleForSpotRate.Wait(ctx)
	case withdrawAddressListRate:
		return r.WithdrawAddressListRate.Wait(ctx)
	case getSubAccountAssetRate:
		return r.GetSubAccountAssetRate.Wait(ctx)
	case getSubAccountStatusOnMarginOrFuturesRate:
		return r.GetSubAccountStatusOnMarginOrFuturesRate.Wait(ctx)
	case subAccountMarginAccountDetailRate:
		return r.SubAccountMarginAccountDetailRate.Wait(ctx)
	case getSubAccountSummaryOfMarginAccountRate:
		return r.SubAccountSummaryOfMarginAccountRate.Wait(ctx)
	case getDetailSubAccountFuturesAccountRate:
		return r.DetailSubAccountFuturesAccountRate.Wait(ctx)
	case getFuturesPositionRiskOfSubAccountV1Rate:
		return r.FuturesPositionRiskOfSubAccountV1Rate.Wait(ctx)
	case getFuturesSubAccountSummaryV2Rate:
		return r.FuturesSubAccountSummaryV2Rate.Wait(ctx)
	case ipRestrictionForSubAccountAPIKeyRate:
		return r.IPRestrictionForSubAccountAPIKeyRate.Wait(ctx)
	case deleteIPListForSubAccountAPIKeyRate:
		return r.DeleteIPListForSubAccountAPIKeyRate.Wait(ctx)
	case addIPRestrictionSubAccountAPIKey:
		return r.AddIPRestrictionSubAccountAPIKeyRate.Wait(ctx)
	case getManagedSubAccountSnapshotRate:
		return r.ManagedSubAccountSnapshotRate.Wait(ctx)
	case managedSubAccountTransferLogRate:
		return r.ManagedSubAccountTransferLogRate.Wait(ctx)
	case managedSubAccountFuturesAssetDetailRate:
		return r.ManagedSubAccountFuturesAssetDetailRate.Wait(ctx)
	case getManagedSubAccountListRate:
		return r.ManagedSubAccountListRate.Wait(ctx)
	case getSubAccountTransactionStatisticsRate:
		return r.SubAccountTransactionStatisticsRate.Wait(ctx)
	case marginAccountBorrowRepayRate:
		return r.MarginAccountBorrowRepayRate.Wait(ctx)
	case borrowRepayRecordsInMarginAccountRate:
		return r.BorrowRepayRecordsInMarginAccountRate.Wait(ctx)
	case getPriceMarginIndexRate:
		return r.PriceMarginIndexRate.Wait(ctx)
	case marginAccountNewOrderRate:
		return r.MarginAccountNewOrderRate.Wait(ctx)
	case marginAccountCancelOrderRate:
		return r.MarginAccountCancelOrderRate.Wait(ctx)
	case adjustCrossMarginMaxLeverageRate:
		return r.AdjustCrossMarginMaxLeverageRate.Wait(ctx)
	case getCrossMarginAccountDetailRate:
		return r.CrossMarginAccountDetailRate.Wait(ctx)
	case getCrossMarginAccountOrderRate:
		return r.CrossMarginAccountOrderRate.Wait(ctx)
	case getMarginAccountsOpenOrdersRate:
		return r.MarginAccountsOpenOrdersRate.Wait(ctx)
	case marginAccountsAllOrdersRate:
		return r.MarginAccountsAllOrdersRate.Wait(ctx)
	case marginOCOOrderRate:
		return r.MarginOCOOrderRate.Wait(ctx)
	case getMarginAccountOCOOrderRate:
		return r.MarginAccountOCOOrderRate.Wait(ctx)
	case getMarginAccountAllOCORate:
		return r.MarginAccountAllOCORate.Wait(ctx)
	case marginAccountOpenOCOOrdersRate:
		return r.MarginAccountOpenOCOOrdersRate.Wait(ctx)
	case marginAccountTradeListRate:
		return r.MarginAccountTradeListRate.Wait(ctx)
	case marginMaxBorrowRate:
		return r.MarginMaxBorrowRate.Wait(ctx)
	case maxTransferOutRate:
		return r.MaxTransferOutRate.Wait(ctx)
	case marginAccountSummaryRate:
		return r.MarginAccountSummaryRate.Wait(ctx)
	case getIsolatedMarginAccountInfoRate:
		return r.IsolatedMarginAccountInfoRate.Wait(ctx)
	case deleteIsolatedMarginAccountRate:
		return r.DeleteIsolatedMarginAccountRate.Wait(ctx)
	case enableIsolatedMarginAccountRate:
		return r.EnableIsolatedMarginAccountRate.Wait(ctx)
	case allIsolatedMarginSymbol:
		return r.AllIsolatedMarginSymbol.Wait(ctx)
	case allCrossMarginFeeDataRate:
		return r.AllCrossMarginFeeDataRate.Wait(ctx)
	case allIsolatedMarginFeeDataRate:
		return r.AllIsolatedMarginFeeDataRate.Wait(ctx)
	case marginCurrentOrderCountUsageRate:
		return r.MarginCurrentOrderCountUsageRate.Wait(ctx)
	case crossMarginCollateralRatioRate:
		return r.CrossMarginCollateralRatioRate.Wait(ctx)
	case smallLiabilityExchangeCoinListRate:
		return r.SmallLiabilityExchangeCoinListRate.Wait(ctx)
	case getSmallLiabilityExchangeCoinListRate:
		return r.GetSmallLiabilityExchangeCoinListRate.Wait(ctx)
	case getSmallLiabilityExchangeRate:
		return r.GetSmallLiabilityExchangeRate.Wait(ctx)
	case marginHourlyInterestRate:
		return r.MarginHourlyInterestRate.Wait(ctx)
	case marginCapitalFlowRate:
		return r.MarginCapitalFlowRate.Wait(ctx)
	case marginTokensAndSymbolsDelistScheduleRate:
		return r.MarginTokensAndSymbolsDelistScheduleRate.Wait(ctx)
	case marginAvailableInventoryRate:
		return r.MarginAvailableInventoryRate.Wait(ctx)
	case marginManualLiquidiationRate:
		return r.MarginManualLiquidiationRate.Wait(ctx)
	case simpleEarnProductsRate:
		return r.SimpleEarnProductsRate.Wait(ctx)
	case getFlexibleSimpleEarnProductPositionRate:
		return r.FlexibleSimpleEarnProductPositionRate.Wait(ctx)
	case getSimpleEarnProductPositionRate:
		return r.GetSimpleEarnProductPositionRate.Wait(ctx)
	case simpleAccountRate:
		return r.SimpleAccountRate.Wait(ctx)
	case getFlexibleSubscriptionRecordRate:
		return r.GetFlexibleSubscriptionRecordRate.Wait(ctx)
	case nftRate:
		limiter, tokens = r.NFTRate, 3000
	case spotRebateHistoryRate:
		return r.SpotRebateHistoryRate.Wait(ctx)
	case convertTradeFlowHistoryRate:
		return r.ConvertTradeFlowHistoryRate.Wait(ctx)
	case getLimitOpenOrdersRate:
		return r.LimitOpenOrdersRate.Wait(ctx)
	case cancelLimitOrderRate:
		return r.CancelLimitOrderRate.Wait(ctx)
	case placeLimitOrderRate:
		return r.PlaceLimitOrderRate.Wait(ctx)
	case orderStatusRate:
		return r.OrderStatusRate.Wait(ctx)
	case acceptQuoteRate:
		return r.AcceptQuoteRate.Wait(ctx)
	case sendQuoteRequestRate:
		return r.SendQuoteRequestRate.Wait(ctx)
	case getOrderQuantityPrecisionPerAssetRate:
		return r.GetOrderQuantityPrecisionPerAssetRate.Wait(ctx)
	case getAllConvertPairsRate:
		return r.GetAllConvertPairsRate.Wait(ctx)
	case payTradeEndpointsRate:
		return r.PayTradeEndpointsRate.Wait(ctx)

	case fiatDepositWithdrawHistRate:
		return r.FiatDepositWithdrawHistRate.Wait(ctx)

		// VIP
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
		SapiDefaultRate:                          request.NewRateLimit(time.Second, 1),
		DailyAccountSnapshotRate:                 request.NewRateLimit(time.Second, 2400),
		FundWithdrawalRate:                       request.NewRateLimit(time.Second, 600),
		WithdrawalHistoryRate:                    request.NewRateLimit(time.Second, 18000),
		UserUniversalTransferRate:                request.NewRateLimit(time.Second, 900),
		UserAssetsRate:                           request.NewRateLimit(time.Second, 5),
		BUSDConvertHistoryRate:                   request.NewRateLimit(time.Second, 5),
		CloudMiningPaymentAndRefundHistoryRate:   request.NewRateLimit(time.Second, 600),
		AutoConvertingStableCoinsRate:            request.NewRateLimit(time.Second, 1200),
		SymbolDelistScheduleForSpotRate:          request.NewRateLimit(time.Second, 100),
		AllCoinInfoRate:                          request.NewRateLimit(time.Second, 10),
		DepositAddressesRate:                     request.NewRateLimit(time.Second, 10),
		DustTransferRate:                         request.NewRateLimit(time.Second, 10),
		AssetDividendRecordRate:                  request.NewRateLimit(time.Second, 10),
		GetDepositAddressListInNetworkRate:       request.NewRateLimit(time.Second, 10),
		WithdrawAddressListRate:                  request.NewRateLimit(time.Second, 10),
		GetSubAccountStatusOnMarginOrFuturesRate: request.NewRateLimit(time.Second, 10),
		SubAccountMarginAccountDetailRate:        request.NewRateLimit(time.Second, 10),
		SubAccountSummaryOfMarginAccountRate:     request.NewRateLimit(time.Second, 10),
		DetailSubAccountFuturesAccountRate:       request.NewRateLimit(time.Second, 10),
		FuturesPositionRiskOfSubAccountV1Rate:    request.NewRateLimit(time.Second, 10),
		FuturesSubAccountSummaryV2Rate:           request.NewRateLimit(time.Second, 10),
		IPRestrictionForSubAccountAPIKeyRate:     request.NewRateLimit(time.Second, 3000),
		DeleteIPListForSubAccountAPIKeyRate:      request.NewRateLimit(time.Second, 3000),
		AddIPRestrictionSubAccountAPIKeyRate:     request.NewRateLimit(time.Second, 3000),
		ManagedSubAccountSnapshotRate:            request.NewRateLimit(time.Second, 2400),
		GetUserWalletBalanceRate:                 request.NewRateLimit(time.Second, 60),
		GetUserDelegationHistoryRate:             request.NewRateLimit(time.Second, 60),
		GetSubAccountAssetRate:                   request.NewRateLimit(time.Second, 60),
		ManagedSubAccountTransferLogRate:         request.NewRateLimit(time.Second, 60),
		ManagedSubAccountFuturesAssetDetailRate:  request.NewRateLimit(time.Second, 60),
		ManagedSubAccountListRate:                request.NewRateLimit(time.Second, 60),
		SubAccountTransactionStatisticsRate:      request.NewRateLimit(time.Second, 60),
		MarginAccountBorrowRepayRate:             request.NewRateLimit(time.Second, 3000),
		BorrowRepayRecordsInMarginAccountRate:    request.NewRateLimit(time.Second, 10),
		PriceMarginIndexRate:                     request.NewRateLimit(time.Second, 10),
		MarginAccountNewOrderRate:                request.NewRateLimit(time.Second, 6),
		MarginAccountCancelOrderRate:             request.NewRateLimit(time.Second, 10),
		AdjustCrossMarginMaxLeverageRate:         request.NewRateLimit(time.Second, 3000),
		CrossMarginAccountDetailRate:             request.NewRateLimit(time.Second, 10),
		CrossMarginAccountOrderRate:              request.NewRateLimit(time.Second, 10),
		MarginAccountsOpenOrdersRate:             request.NewRateLimit(time.Second, 10),
		MarginAccountsAllOrdersRate:              request.NewRateLimit(time.Second, 200),
		MarginOCOOrderRate:                       request.NewRateLimit(time.Second, 6),
		MarginAccountOCOOrderRate:                request.NewRateLimit(time.Second, 10),
		MarginAccountAllOCORate:                  request.NewRateLimit(time.Second, 200),
		MarginAccountOpenOCOOrdersRate:           request.NewRateLimit(time.Second, 10),
		MarginAccountTradeListRate:               request.NewRateLimit(time.Second, 10),
		MarginMaxBorrowRate:                      request.NewRateLimit(time.Second, 50),
		MaxTransferOutRate:                       request.NewRateLimit(time.Second, 50),
		MarginAccountSummaryRate:                 request.NewRateLimit(time.Second, 10),
		IsolatedMarginAccountInfoRate:            request.NewRateLimit(time.Second, 10),
		DeleteIsolatedMarginAccountRate:          request.NewRateLimit(time.Second, 300),
		EnableIsolatedMarginAccountRate:          request.NewRateLimit(time.Second, 300),
		AllIsolatedMarginSymbol:                  request.NewRateLimit(time.Second, 10),
		AllCrossMarginFeeDataRate:                request.NewRateLimit(time.Second, 5),
		AllIsolatedMarginFeeDataRate:             request.NewRateLimit(time.Second, 10),
		MarginCurrentOrderCountUsageRate:         request.NewRateLimit(time.Second, 20),
		CrossMarginCollateralRatioRate:           request.NewRateLimit(time.Second, 100),
		SmallLiabilityExchangeCoinListRate:       request.NewRateLimit(time.Second, 3000),
		GetSmallLiabilityExchangeCoinListRate:    request.NewRateLimit(time.Second, 100),
		GetSmallLiabilityExchangeRate:            request.NewRateLimit(time.Second, 100),
		MarginHourlyInterestRate:                 request.NewRateLimit(time.Second, 100),
		MarginCapitalFlowRate:                    request.NewRateLimit(time.Second, 100),
		MarginTokensAndSymbolsDelistScheduleRate: request.NewRateLimit(time.Second, 100),
		MarginAvailableInventoryRate:             request.NewRateLimit(time.Second, 50),
		MarginManualLiquidiationRate:             request.NewRateLimit(time.Second, 3000),
		SimpleEarnProductsRate:                   request.NewRateLimit(time.Second, 150),
		FlexibleSimpleEarnProductPositionRate:    request.NewRateLimit(time.Second, 150),
		GetSimpleEarnProductPositionRate:         request.NewRateLimit(time.Second, 150),
		SimpleAccountRate:                        request.NewRateLimit(time.Second, 150),
		GetFlexibleSubscriptionRecordRate:        request.NewRateLimit(time.Second, 150),
		NFTRate:                                  request.NewRateLimit(time.Second, 12000),
		SpotRebateHistoryRate:                    request.NewRateLimit(time.Second, 12000),
		ConvertTradeFlowHistoryRate:              request.NewRateLimit(time.Second, 3000),
		LimitOpenOrdersRate:                      request.NewRateLimit(time.Second, 3000),
		CancelLimitOrderRate:                     request.NewRateLimit(time.Second, 200),
		PlaceLimitOrderRate:                      request.NewRateLimit(time.Second, 500),
		OrderStatusRate:                          request.NewRateLimit(time.Second, 100),
		AcceptQuoteRate:                          request.NewRateLimit(time.Second, 500),
		SendQuoteRequestRate:                     request.NewRateLimit(time.Second, 200),
		GetOrderQuantityPrecisionPerAssetRate:    request.NewRateLimit(time.Second, 100),
		GetAllConvertPairsRate:                   request.NewRateLimit(time.Second, 20),
		PayTradeEndpointsRate:                    request.NewRateLimit(time.Second, 3000),
		VIPLoanEndpointsRate:                     request.NewRateLimit(time.Second, 26600),
		FiatDepositWithdrawHistRate:              request.NewRateLimit(time.Second, 90000),
		ClassicPM:                                request.NewRateLimit(time.Second, 500), // TODO:
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
