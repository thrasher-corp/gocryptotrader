package kucoin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	thirtySecondsInterval = time.Second * 30
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate       *rate.Limiter
	FuturesRate    *rate.Limiter
	ManagementRate *rate.Limiter
	PublicRate     *rate.Limiter
}

const (
	accountSummaryInfoEPL request.EndpointLimit = iota
	allAccountEPL
	accountDetailEPL
	accountLedgersEPL
	hfAccountLedgersEPL
	hfAccountLedgersMarginEPL
	futuresAccountLedgersEPL
	subAccountInfoV1EPL
	allSubAccountsInfoV2EPL
	createSubUserEPL
	subAccountsEPL
	subAccountBalancesEPL
	allSubAccountBalancesV2EPL
	subAccountSpotAPIListEPL
	createSpotAPIForSubAccountEPL
	modifySubAccountSpotAPIEPL
	deleteSubAccountSpotAPIEPL
	marginAccountDetailEPL
	crossMarginAccountsDetailEPL
	isolatedMarginAccountDetailEPL
	futuresAccountsDetailEPL
	allFuturesSubAccountBalancesEPL
	createDepositAddressEPL
	depositAddressesV2EPL
	depositAddressesV1EPL
	depositListEPL
	historicDepositListEPL
	withdrawalListEPL
	retrieveV1HistoricalWithdrawalListEPL
	withdrawalQuotaEPL
	applyWithdrawalEPL
	cancelWithdrawalsEPL
	getTransferablesEPL
	flexiTransferEPL
	masterSubUserTransferEPL
	innerTransferEPL
	toMainOrTradeAccountEPL
	toFuturesAccountEPL
	futuresTransferOutRequestRecordsEPL
	basicFeesEPL
	tradeFeesEPL
	spotCurrenciesV3EPL
	spotCurrencyDetailEPL
	symbolsEPL
	tickersEPL
	allTickersEPL
	statistics24HrEPL
	marketListEPL
	partOrderbook20EPL
	partOrderbook100EPL
	fullOrderbookEPL
	tradeHistoryEPL
	klinesEPL
	fiatPriceEPL
	currentServerTimeEPL
	serviceStatusEPL
	hfPlaceOrderEPL
	hfSyncPlaceOrderEPL
	hfMultipleOrdersEPL
	hfSyncPlaceMultipleHFOrdersEPL
	hfModifyOrderEPL
	cancelHFOrderEPL
	hfSyncCancelOrderEPL
	hfCancelOrderByClientOrderIDEPL
	cancelSpecifiedNumberHFOrdersByOrderIDEPL
	hfCancelAllOrdersBySymbolEPL
	hfCancelAllOrdersEPL
	hfGetAllActiveOrdersEPL
	hfSymbolsWithActiveOrdersEPL
	hfCompletedOrderListEPL
	hfOrderDetailByOrderIDEPL
	autoCancelHFOrderSettingEPL
	autoCancelHFOrderSettingQueryEPL
	hfFilledListEPL
	placeOrderEPL
	placeBulkOrdersEPL
	cancelOrderEPL
	cancelOrderByClientOrderIDEPL
	cancelAllOrdersEPL
	listOrdersEPL
	recentOrdersEPL
	orderDetailByIDEPL
	getOrderByClientSuppliedOrderIDEPL
	listFillsEPL
	getRecentFillsEPL
	placeStopOrderEPL
	cancelStopOrderEPL
	cancelStopOrderByClientIDEPL
	cancelStopOrdersEPL
	listStopOrdersEPL
	getStopOrderDetailEPL
	getStopOrderByClientIDEPL
	placeOCOOrderEPL
	cancelOCOOrderByIDEPL
	cancelMultipleOCOOrdersEPL
	getOCOOrderByIDEPL
	getOCOOrderDetailsByOrderIDEPL
	getOCOOrdersEPL
	placeMarginOrderEPL
	cancelMarginHFOrderByIDEPL
	getMarginHFOrderDetailByID
	cancelAllMarginHFOrdersBySymbolEPL
	getActiveMarginHFOrdersEPL
	getFilledHFMarginOrdersEPL
	getMarginHFOrderDetailByOrderIDEPL
	getMarginHFTradeFillsEPL
	placeMarginOrdersEPL
	leveragedTokenInfoEPL
	getMarkPriceEPL
	getMarginConfigurationEPL
	crossIsolatedMarginRiskLimitCurrencyConfigEPL
	isolatedMarginPairConfigEPL
	isolatedMarginAccountInfoEPL
	singleIsolatedMarginAccountInfoEPL
	postMarginBorrowOrderEPL
	postMarginRepaymentEPL
	marginBorrowingHistoryEPL
	marginRepaymentHistoryEPL
	lendingCurrencyInfoEPL
	interestRateEPL
	marginLendingSubscriptionEPL
	redemptionEPL
	modifySubscriptionEPL
	getRedemptionOrdersEPL
	getSubscriptionOrdersEPL
	futuresOpenContractsEPL
	futuresContractEPL
	futuresTickerEPL
	futuresOrderbookEPL
	futuresPartOrderbookDepth20EPL
	futuresPartOrderbookDepth100EPL
	futuresTransactionHistoryEPL
	futuresKlineEPL
	futuresInterestRateEPL
	futuresIndexListEPL
	futuresCurrentMarkPriceEPL
	futuresPremiumIndexEPL
	futuresTransactionVolumeEPL
	futuresServerTimeEPL
	futuresServiceStatusEPL
	multipleFuturesOrdersEPL
	futuresCancelAnOrderEPL
	futuresPlaceOrderEPL
	futuresLimitOrderMassCancelationEPL
	cancelUntriggeredFuturesStopOrdersEPL
	futuresCancelMultipleLimitOrdersEPL
	futuresRetrieveOrderListEPL
	futuresRecentCompletedOrdersEPL
	futuresOrdersByIDEPL
	futuresRetrieveFillsEPL
	futuresRecentFillsEPL
	futuresOpenOrderStatsEPL
	futuresPositionEPL
	futuresPositionListEPL
	setAutoDepositMarginEPL
	maxWithdrawMarginEPL
	removeMarginManuallyEPL
	futuresAddMarginManuallyEPL
	futuresRiskLimitLevelEPL
	futuresUpdateRiskLmitLevelEPL
	futuresCurrentFundingRateEPL
	futuresPublicFundingRateEPL
	futuresFundingHistoryEPL
	spotAuthenticationEPL
	futuresAuthenticationEPL
	futuresOrderDetailsByClientOrderIDEPL
	modifySubAccountAPIEPL
	allSubAccountsBalanceEPL
	allUserSubAccountsV2EPL
	futuresRetrieveTransactionHistoryEPL
	futuresAccountOverviewEPL
	createSubAccountAPIKeyEPL
	transferOutToMainEPL
	transferFundToFuturesAccountEPL
	futuresTransferOutListEPL
)

// Limit executes rate limiting functionality for Kucoin
func (r *RateLimit) Limit(ctx context.Context, epl request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch epl {
	case accountSummaryInfoEPL:
		limiter, tokens = r.ManagementRate, 20
	case allAccountEPL:
		limiter, tokens = r.ManagementRate, 5
	case accountDetailEPL:
		limiter, tokens = r.ManagementRate, 5
	case accountLedgersEPL:
		limiter, tokens = r.ManagementRate, 2
	case hfAccountLedgersEPL:
		limiter, tokens = r.SpotRate, 2
	case hfAccountLedgersMarginEPL:
		limiter, tokens = r.SpotRate, 2
	case futuresAccountLedgersEPL:
		limiter, tokens = r.SpotRate, 2
	case subAccountInfoV1EPL:
		limiter, tokens = r.ManagementRate, 20
	case allSubAccountsInfoV2EPL:
		limiter, tokens = r.ManagementRate, 20
	case createSubUserEPL:
		limiter, tokens = r.ManagementRate, 20
	case subAccountsEPL:
		limiter, tokens = r.ManagementRate, 15
	case subAccountBalancesEPL:
		limiter, tokens = r.ManagementRate, 20
	case allSubAccountBalancesV2EPL:
		limiter, tokens = r.ManagementRate, 20
	case subAccountSpotAPIListEPL:
		limiter, tokens = r.ManagementRate, 20
	case createSpotAPIForSubAccountEPL:
		limiter, tokens = r.ManagementRate, 20
	case modifySubAccountSpotAPIEPL:
		limiter, tokens = r.ManagementRate, 30
	case deleteSubAccountSpotAPIEPL:
		limiter, tokens = r.ManagementRate, 30
	case marginAccountDetailEPL:
		limiter, tokens = r.SpotRate, 40
	case crossMarginAccountsDetailEPL:
		limiter, tokens = r.SpotRate, 15
	case isolatedMarginAccountDetailEPL:
		limiter, tokens = r.SpotRate, 15
	case futuresAccountsDetailEPL:
		limiter, tokens = r.FuturesRate, 5
	case allFuturesSubAccountBalancesEPL:
		limiter, tokens = r.FuturesRate, 6
	case createDepositAddressEPL:
		limiter, tokens = r.ManagementRate, 20
	case depositAddressesV2EPL:
		limiter, tokens = r.ManagementRate, 5
	case depositAddressesV1EPL:
		limiter, tokens = r.ManagementRate, 5
	case depositListEPL:
		limiter, tokens = r.ManagementRate, 5
	case historicDepositListEPL:
		limiter, tokens = r.ManagementRate, 5
	case withdrawalListEPL:
		limiter, tokens = r.ManagementRate, 20
	case retrieveV1HistoricalWithdrawalListEPL:
		limiter, tokens = r.ManagementRate, 20
	case withdrawalQuotaEPL:
		limiter, tokens = r.ManagementRate, 20
	case applyWithdrawalEPL:
		limiter, tokens = r.ManagementRate, 5
	case cancelWithdrawalsEPL:
		limiter, tokens = r.ManagementRate, 20
	case getTransferablesEPL:
		limiter, tokens = r.ManagementRate, 20
	case flexiTransferEPL:
		limiter, tokens = r.ManagementRate, 4
	case masterSubUserTransferEPL:
		limiter, tokens = r.ManagementRate, 30
	case innerTransferEPL:
		limiter, tokens = r.ManagementRate, 10
	case toMainOrTradeAccountEPL:
		limiter, tokens = r.ManagementRate, 20
	case toFuturesAccountEPL:
		limiter, tokens = r.ManagementRate, 20
	case futuresTransferOutRequestRecordsEPL:
		limiter, tokens = r.ManagementRate, 20
	case basicFeesEPL:
		limiter, tokens = r.SpotRate, 3
	case tradeFeesEPL:
		limiter, tokens = r.SpotRate, 3
	case spotCurrenciesV3EPL:
		limiter, tokens = r.PublicRate, 3
	case spotCurrencyDetailEPL:
		limiter, tokens = r.PublicRate, 3
	case symbolsEPL:
		limiter, tokens = r.PublicRate, 4
	case tickersEPL:
		limiter, tokens = r.PublicRate, 2
	case allTickersEPL:
		limiter, tokens = r.PublicRate, 15
	case statistics24HrEPL:
		limiter, tokens = r.PublicRate, 15
	case marketListEPL:
		limiter, tokens = r.PublicRate, 3
	case partOrderbook20EPL:
		limiter, tokens = r.PublicRate, 2
	case partOrderbook100EPL:
		limiter, tokens = r.PublicRate, 4
	case fullOrderbookEPL:
		limiter, tokens = r.SpotRate, 3
	case tradeHistoryEPL:
		limiter, tokens = r.PublicRate, 3
	case klinesEPL:
		limiter, tokens = r.PublicRate, 3
	case fiatPriceEPL:
		limiter, tokens = r.PublicRate, 3
	case currentServerTimeEPL:
		limiter, tokens = r.PublicRate, 3
	case serviceStatusEPL:
		limiter, tokens = r.PublicRate, 3
	case hfPlaceOrderEPL:
		limiter, tokens = r.SpotRate, 1
	case hfSyncPlaceOrderEPL:
		limiter, tokens = r.SpotRate, 1
	case hfMultipleOrdersEPL:
		limiter, tokens = r.SpotRate, 1
	case hfSyncPlaceMultipleHFOrdersEPL:
		limiter, tokens = r.SpotRate, 1
	case hfModifyOrderEPL:
		limiter, tokens = r.SpotRate, 3
	case cancelHFOrderEPL:
		limiter, tokens = r.SpotRate, 1
	case hfSyncCancelOrderEPL:
		limiter, tokens = r.SpotRate, 1
	case hfCancelOrderByClientOrderIDEPL:
		limiter, tokens = r.SpotRate, 1
	case cancelSpecifiedNumberHFOrdersByOrderIDEPL:
		limiter, tokens = r.SpotRate, 2
	case hfCancelAllOrdersBySymbolEPL:
		limiter, tokens = r.SpotRate, 2
	case hfCancelAllOrdersEPL:
		limiter, tokens = r.SpotRate, 30
	case hfGetAllActiveOrdersEPL:
		limiter, tokens = r.SpotRate, 2
	case hfSymbolsWithActiveOrdersEPL:
		limiter, tokens = r.SpotRate, 2
	case hfCompletedOrderListEPL:
		limiter, tokens = r.SpotRate, 2
	case hfOrderDetailByOrderIDEPL:
		limiter, tokens = r.SpotRate, 2
	case autoCancelHFOrderSettingEPL:
		limiter, tokens = r.SpotRate, 2
	case autoCancelHFOrderSettingQueryEPL:
		limiter, tokens = r.SpotRate, 2
	case hfFilledListEPL:
		limiter, tokens = r.SpotRate, 2
	case placeOrderEPL:
		limiter, tokens = r.SpotRate, 2
	case placeBulkOrdersEPL:
		limiter, tokens = r.SpotRate, 3
	case cancelOrderEPL:
		limiter, tokens = r.SpotRate, 3
	case cancelOrderByClientOrderIDEPL:
		limiter, tokens = r.SpotRate, 5
	case cancelAllOrdersEPL:
		limiter, tokens = r.SpotRate, 20
	case listOrdersEPL:
		limiter, tokens = r.SpotRate, 2
	case recentOrdersEPL:
		limiter, tokens = r.SpotRate, 3
	case orderDetailByIDEPL:
		limiter, tokens = r.SpotRate, 2
	case getOrderByClientSuppliedOrderIDEPL:
		limiter, tokens = r.SpotRate, 3
	case listFillsEPL:
		limiter, tokens = r.SpotRate, 10
	case getRecentFillsEPL:
		limiter, tokens = r.SpotRate, 20
	case placeStopOrderEPL:
		limiter, tokens = r.SpotRate, 2
	case cancelStopOrderEPL:
		limiter, tokens = r.SpotRate, 3
	case cancelStopOrderByClientIDEPL:
		limiter, tokens = r.SpotRate, 5
	case cancelStopOrdersEPL:
		limiter, tokens = r.SpotRate, 3
	case listStopOrdersEPL:
		limiter, tokens = r.SpotRate, 8
	case getStopOrderDetailEPL:
		limiter, tokens = r.SpotRate, 3
	case getStopOrderByClientIDEPL:
		limiter, tokens = r.SpotRate, 3
	case placeOCOOrderEPL:
		limiter, tokens = r.SpotRate, 2
	case cancelOCOOrderByIDEPL:
		limiter, tokens = r.SpotRate, 3
	case cancelMultipleOCOOrdersEPL:
		limiter, tokens = r.SpotRate, 3
	case getOCOOrderByIDEPL:
		limiter, tokens = r.SpotRate, 2
	case getOCOOrderDetailsByOrderIDEPL:
		limiter, tokens = r.SpotRate, 2
	case getOCOOrdersEPL:
		limiter, tokens = r.SpotRate, 2
	case placeMarginOrderEPL:
		limiter, tokens = r.SpotRate, 5
	case cancelMarginHFOrderByIDEPL:
		limiter, tokens = r.SpotRate, 5
	case getMarginHFOrderDetailByID:
		limiter, tokens = r.SpotRate, 5
	case cancelAllMarginHFOrdersBySymbolEPL:
		limiter, tokens = r.SpotRate, 10
	case getActiveMarginHFOrdersEPL:
		limiter, tokens = r.SpotRate, 4
	case getFilledHFMarginOrdersEPL:
		limiter, tokens = r.SpotRate, 10
	case getMarginHFOrderDetailByOrderIDEPL:
		limiter, tokens = r.SpotRate, 4
	case getMarginHFTradeFillsEPL:
		limiter, tokens = r.SpotRate, 5
	case placeMarginOrdersEPL:
		limiter, tokens = r.SpotRate, 5
	case leveragedTokenInfoEPL:
		limiter, tokens = r.SpotRate, 25
	case getMarkPriceEPL:
		limiter, tokens = r.PublicRate, 2
	case getMarginConfigurationEPL:
		limiter, tokens = r.SpotRate, 25
	case crossIsolatedMarginRiskLimitCurrencyConfigEPL:
		limiter, tokens = r.SpotRate, 20
	case isolatedMarginPairConfigEPL:
		limiter, tokens = r.SpotRate, 20
	case isolatedMarginAccountInfoEPL:
		limiter, tokens = r.SpotRate, 50
	case singleIsolatedMarginAccountInfoEPL:
		limiter, tokens = r.SpotRate, 50
	case postMarginBorrowOrderEPL:
		limiter, tokens = r.SpotRate, 15
	case postMarginRepaymentEPL:
		limiter, tokens = r.SpotRate, 10
	case marginBorrowingHistoryEPL:
		limiter, tokens = r.SpotRate, 15
	case marginRepaymentHistoryEPL:
		limiter, tokens = r.SpotRate, 15
	case lendingCurrencyInfoEPL:
		limiter, tokens = r.SpotRate, 10
	case interestRateEPL:
		limiter, tokens = r.PublicRate, 5
	case marginLendingSubscriptionEPL:
		limiter, tokens = r.SpotRate, 15
	case redemptionEPL:
		limiter, tokens = r.SpotRate, 15
	case modifySubscriptionEPL:
		limiter, tokens = r.SpotRate, 10
	case getRedemptionOrdersEPL:
		limiter, tokens = r.SpotRate, 10
	case getSubscriptionOrdersEPL:
		limiter, tokens = r.SpotRate, 10
	case futuresOpenContractsEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresContractEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresTickerEPL:
		limiter, tokens = r.PublicRate, 2
	case futuresOrderbookEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresPartOrderbookDepth20EPL:
		limiter, tokens = r.PublicRate, 5
	case futuresPartOrderbookDepth100EPL:
		limiter, tokens = r.PublicRate, 10
	case futuresTransactionHistoryEPL:
		limiter, tokens = r.PublicRate, 5
	case futuresKlineEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresInterestRateEPL:
		limiter, tokens = r.PublicRate, 5
	case futuresIndexListEPL:
		limiter, tokens = r.PublicRate, 2
	case futuresCurrentMarkPriceEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresPremiumIndexEPL:
		limiter, tokens = r.PublicRate, 3
	case futuresTransactionVolumeEPL:
		limiter, tokens = r.FuturesRate, 3
	case futuresServerTimeEPL:
		limiter, tokens = r.PublicRate, 2
	case futuresServiceStatusEPL:
		limiter, tokens = r.PublicRate, 4
	case multipleFuturesOrdersEPL:
		limiter, tokens = r.FuturesRate, 20
	case futuresCancelAnOrderEPL:
		limiter, tokens = r.FuturesRate, 1
	case futuresPlaceOrderEPL:
		limiter, tokens = r.FuturesRate, 2
	case futuresLimitOrderMassCancelationEPL:
		limiter, tokens = r.FuturesRate, 30
	case cancelUntriggeredFuturesStopOrdersEPL:
		limiter, tokens = r.FuturesRate, 15
	case futuresCancelMultipleLimitOrdersEPL:
		limiter, tokens = r.FuturesRate, 30
	case futuresRetrieveOrderListEPL:
		limiter, tokens = r.FuturesRate, 2
	case futuresRecentCompletedOrdersEPL:
		limiter, tokens = r.FuturesRate, 5
	case futuresOrdersByIDEPL:
		limiter, tokens = r.FuturesRate, 5
	case futuresRetrieveFillsEPL:
		limiter, tokens = r.FuturesRate, 5
	case futuresRecentFillsEPL:
		limiter, tokens = r.FuturesRate, 3
	case futuresOpenOrderStatsEPL:
		limiter, tokens = r.FuturesRate, 10
	case futuresPositionEPL:
		limiter, tokens = r.FuturesRate, 2
	case futuresPositionListEPL:
		limiter, tokens = r.FuturesRate, 2
	case setAutoDepositMarginEPL:
		limiter, tokens = r.FuturesRate, 4
	case maxWithdrawMarginEPL:
		limiter, tokens = r.FuturesRate, 10
	case removeMarginManuallyEPL:
		limiter, tokens = r.FuturesRate, 10
	case futuresAddMarginManuallyEPL:
		limiter, tokens = r.FuturesRate, 4
	case futuresRiskLimitLevelEPL:
		limiter, tokens = r.FuturesRate, 5
	case futuresUpdateRiskLmitLevelEPL:
		limiter, tokens = r.FuturesRate, 4
	case futuresCurrentFundingRateEPL:
		limiter, tokens = r.PublicRate, 2
	case futuresPublicFundingRateEPL:
		limiter, tokens = r.PublicRate, 5
	case futuresFundingHistoryEPL:
		limiter, tokens = r.FuturesRate, 5
	case spotAuthenticationEPL:
		limiter, tokens = r.SpotRate, 10
	case futuresAuthenticationEPL:
		limiter, tokens = r.FuturesRate, 10
	case futuresOrderDetailsByClientOrderIDEPL:
		limiter, tokens = r.FuturesRate, 5

	case modifySubAccountAPIEPL:
		limiter, tokens = r.ManagementRate, 30
	case allSubAccountsBalanceEPL:
		limiter, tokens = r.ManagementRate, 20
	case allUserSubAccountsV2EPL:
		limiter, tokens = r.ManagementRate, 20
	case futuresRetrieveTransactionHistoryEPL:
		limiter, tokens = r.ManagementRate, 2
	case futuresAccountOverviewEPL:
		limiter, tokens = r.FuturesRate, 5
	case createSubAccountAPIKeyEPL:
		limiter, tokens = r.ManagementRate, 20
	case transferOutToMainEPL:
		limiter, tokens = r.ManagementRate, 20
	case transferFundToFuturesAccountEPL:
		limiter, tokens = r.ManagementRate, 20
	case futuresTransferOutListEPL:
		limiter, tokens = r.ManagementRate, 20

	default:
		return errors.New("endpoint rate limit functionality not found")
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

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() *RateLimit {
	return &RateLimit{
		// default spot and futures rates
		SpotRate:       request.NewRateLimit(thirtySecondsInterval, 3000),
		FuturesRate:    request.NewRateLimit(thirtySecondsInterval, 2000),
		ManagementRate: request.NewRateLimit(thirtySecondsInterval, 2000),
		PublicRate:     request.NewRateLimit(thirtySecondsInterval, 2000),
	}
}
