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
