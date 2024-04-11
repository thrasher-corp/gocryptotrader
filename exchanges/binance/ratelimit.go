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
	spotOrderbookDepth500Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	spotOrderbookTickerAllRate
	spotPriceChangeAllRate
	spotSymbolPriceAllRate
	spotOpenOrdersAllRate
	spotOpenOrdersSpecificRate
	spotOrderRate
	spotOrderQueryRate
	spotAllOrdersRate
	spotAccountInformationRate
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
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate:
		limiter, tokens = r.SpotRate, 2
	case spotHistoricalTradesRate,
		spotOrderbookDepth500Rate:
		limiter, tokens = r.SpotRate, 5
	case spotOrderbookDepth1000Rate,
		spotAccountInformationRate,
		spotExchangeInfo:
		limiter, tokens = r.SpotRate, 10
	case walletSystemStatus:
		limiter, tokens = r.SpotRate, 1
	case allCoinInfoRate:
		limiter, tokens = r.SpotRate, 10
	case dailyAccountSnapshotRate:
		limiter, tokens = r.SpotRate, 2400
	case fundWithdrawalRate:
		limiter, tokens = r.SpotRate, 600
	case withdrawalHistoryRate:
		limiter, tokens = r.SpotRate, 10
	case spotPriceChangeAllRate:
		limiter, tokens = r.SpotRate, 40
	case spotOrderbookDepth5000Rate:
		limiter, tokens = r.SpotRate, 50
	case spotOrderRate:
		limiter, tokens = r.SpotOrdersRate, 1
	case spotOrderQueryRate:
		limiter, tokens = r.SpotOrdersRate, 2
	case spotOpenOrdersSpecificRate:
		limiter, tokens = r.SpotOrdersRate, 3
	case spotAllOrdersRate:
		limiter, tokens = r.SpotOrdersRate, 10
	case spotOpenOrdersAllRate:
		limiter, tokens = r.SpotOrdersRate, 40
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
	case optionsRecentTradesRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsHistoricalTradesRate:
		limiter, tokens = r.EOptionsRate, 20
	case optionsMarkPriceRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsAllTickerPriceStatistics:
		limiter, tokens = r.EOptionsRate, 5
	case optionsHistoricalExerciseRecordsRate:
		limiter, tokens = r.EOptionsRate, 3
	case optionsAccountInfoRate:
		limiter, tokens = r.EOptionsOrderRate, 3
	case optionsDefaultOrderRate:
		limiter, tokens = r.EOptionsOrderRate, 1
	case optionsBatchOrderRate:
		limiter, tokens = r.EOptionsOrderRate, 5
	case optionsAllQueryOpenOrdersRate:
		limiter, tokens = r.EOptionsOrderRate, 40
	case optionsGetOrderHistory:
		limiter, tokens = r.EOptionsOrderRate, 3
	case optionsPositionInformationRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsAccountTradeListRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsUserExerciseRecordRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsDownloadIDForOptionTrasactionHistoryRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsGetTransHistoryDownloadLinkByIDRate:
		limiter, tokens = r.EOptionsRate, 5
	case optionsMarginAccountInfoRate:
		limiter, tokens = r.EOptionsRate, 3
	case optionsAutoCancelAllOpenOrdersHeartbeatRate:
		limiter, tokens = r.EOptionsRate, 10

	case pmDefaultRate:
		limiter, tokens = r.PortfolioMarginRate, 1
	case pmMarginAccountLoanAndRepayRate:
		limiter, tokens = r.PortfolioMarginRate, 100
	case pmCancelMarginAccountOpenOrdersOnSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmCancelMarginAccountOCORate:
		limiter, tokens = r.PortfolioMarginRate, 2
	case pmRetrieveAllUMOpenOrdersForAllSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmGetAllUMOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmRetrieveAllCMOpenOrdersForAllSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmAllCMOrderWithSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmAllCMOrderWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmUMOpenConditionalOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmAllUMConditionalOrdersWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmAllCMOpenConditionalOrdersWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmAllCMConditionalOrderWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmGetMarginAccountOrderRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmCurrentMarginOpenOrderRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmAllMarginAccountOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 100
	case pmGetMarginAccountOCORate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetMarginAccountsAllOCOOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 100
	case pmGetMarginAccountsOpenOCOOrdersRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetMarginAccountTradeListRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetAccountBalancesRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetAccountInformationRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmMarginMaxBorrowRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetMarginMaxWithdrawalRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetUMPositionInformationRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetUMCurrentPositionModeRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmGetCMCurrentPositionModeRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmGetUMAccountTradeListRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetCMAccountTradeListWithSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetCMAccountTradeListWithPairRate:
		limiter, tokens = r.PortfolioMarginRate, 40
	case pmGetUserUMForceOrdersWithSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetUserUMForceOrdersWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 50
	case pmGetUserCMForceOrdersWithSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetUserCMForceOrdersWithoutSymbolRate:
		limiter, tokens = r.PortfolioMarginRate, 50
	case pmUMTradingQuantitativeRulesIndicatorsRate:
		limiter, tokens = r.PortfolioMarginRate, 10
	case pmGetUMUserCommissionRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetCMUserCommissionRate:
		limiter, tokens = r.PortfolioMarginRate, 20
	case pmGetMarginLoanRecordRate:
		limiter, tokens = r.PortfolioMarginRate, 10
	case pmGetMarginRepayRecordRate:
		limiter, tokens = r.PortfolioMarginRate, 10
	case pmGetPortfolioMarginNegativeBalanceInterestHistoryRate:
		limiter, tokens = r.PortfolioMarginRate, 50
	case pmFundAutoCollectionRate:
		limiter, tokens = r.PortfolioMarginRate, 750
	case pmFundCollectionByAssetRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmBNBTransferRate:
		limiter, tokens = r.PortfolioMarginRate, 750
	case pmGetUMIncomeHistoryRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmGetCMIncomeHistoryRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmGetUMAccountDetailRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetCMAccountDetailRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmChangeAutoRepayFuturesStatusRate:
		limiter, tokens = r.PortfolioMarginRate, 750
	case pmGetAutoRepayFuturesStatusRate:
		limiter, tokens = r.PortfolioMarginRate, 30
	case pmRepayFuturesNegativeBalanceRate:
		limiter, tokens = r.PortfolioMarginRate, 750
	case pmGetUMPositionADLQuantileEstimationRate:
		limiter, tokens = r.PortfolioMarginRate, 5
	case pmGetCMPositionADLQuantileEstimationRate:
		limiter, tokens = r.PortfolioMarginRate, 5
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
		return spotDefaultRate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}

	return spotOrderbookDepth5000Rate
}
