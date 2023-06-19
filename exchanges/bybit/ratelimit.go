package bybit

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// See: https://bybit-exchange.github.io/docs/v5/rate-limit
	spotInterval    = time.Second * 5
	spotRequestRate = 120

	futuresPublicInterval = time.Second
	futuresRequestRate    = 50

	spotPrivateInterval       = time.Second
	spotPrivateRequestRate    = 20
	spotPrivateFeeRequestRate = 10
	futuresInterval           = time.Minute
	futuresDefaultRateCount   = 100
	futuresOrderRate          = 100
	futuresOrderListRate      = 600
	futuresExecutionRate      = 120
	futuresPositionRateCount  = 75
	futuresPositionListRate   = 120
	futuresFundingRate        = 120
	futuresWalletRate         = 120
	futuresAccountRate        = 600

	usdcPerpetualPublicRate    = 50
	usdcPerpetualCancelAllRate = 1
	usdcPerpetualPrivateRate   = 5
	usdcPerpetualInterval      = time.Second
)

const (
	publicSpotRate request.EndpointLimit = iota
	publicFuturesRate
	privateSpotRate
	privateFeeRate
	cFuturesDefaultRate

	cFuturesCancelActiveOrderRate
	cFuturesCancelAllActiveOrderRate
	cFuturesCreateConditionalOrderRate
	cFuturesCancelConditionalOrderRate
	cFuturesReplaceActiveOrderRate
	cFuturesReplaceConditionalOrderRate
	cFuturesCreateOrderRate
	cFuturesCancelAllConditionalOrderRate

	cFuturesGetActiveOrderRate
	cFuturesGetConditionalOrderRate
	cFuturesGetRealtimeOrderRate

	cFuturesTradeRate

	cFuturesSetLeverageRate
	cFuturesUpdateMarginRate
	cFuturesSetTradingRate
	cFuturesSwitchPositionRate
	cFuturesGetTradingFeeRate

	cFuturesPositionRate
	cFuturesWalletBalanceRate

	cFuturesLastFundingFeeRate
	cFuturesPredictFundingRate

	cFuturesWalletFundRecordRate
	cFuturesWalletWithdrawalRate

	cFuturesAPIKeyInfoRate

	uFuturesDefaultRate

	uFuturesCreateOrderRate
	uFuturesCancelOrderRate
	uFuturesCancelAllOrderRate
	uFuturesCreateConditionalOrderRate
	uFuturesCancelConditionalOrderRate
	uFuturesCancelAllConditionalOrderRate

	uFuturesSetLeverageRate
	uFuturesSwitchMargin
	uFuturesSwitchPosition
	uFuturesSetMarginRate
	uFuturesSetTradingStopRate
	uFuturesUpdateMarginRate

	uFuturesPositionRate
	uFuturesGetClosedTradesRate
	uFuturesGetTradesRate

	uFuturesGetActiveOrderRate
	uFuturesGetActiveRealtimeOrderRate
	uFuturesGetConditionalOrderRate
	uFuturesGetConditionalRealtimeOrderRate

	uFuturesGetMyLastFundingFeeRate
	uFuturesPredictFundingRate

	futuresDefaultRate

	futuresCancelOrderRate
	futuresCreateOrderRate
	futuresReplaceOrderRate
	futuresCancelAllOrderRate
	futuresCancelAllConditionalOrderRate
	futuresReplaceConditionalOrderRate
	futuresCancelConditionalOrderRate
	futuresCreateConditionalOrderRate

	futuresGetActiveOrderRate
	futuresGetConditionalOrderRate
	futuresGetActiveRealtimeOrderRate
	futuresGetConditionalRealtimeOrderRate

	futuresGetTradeRate

	futuresSetLeverageRate
	futuresUpdateMarginRate
	futuresSetTradingStopRate
	futuresSwitchPositionModeRate
	futuresSwitchMarginRate
	futuresSwitchPositionRate

	futuresPositionRate

	usdcPublicRate

	usdcCancelAllOrderRate

	usdcPlaceOrderRate
	usdcModifyOrderRate
	usdcCancelOrderRate
	usdcGetOrderRate
	usdcGetOrderHistoryRate
	usdcGetTradeHistoryRate
	usdcGetTransactionRate
	usdcGetWalletRate
	usdcGetAssetRate
	usdcGetMarginRate
	usdcGetPositionRate
	usdcSetLeverageRate
	usdcGetSettlementRate
	usdcSetRiskRate
	usdcGetPredictedFundingRate
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate                    *rate.Limiter
	FuturesRate                 *rate.Limiter
	PrivateSpotRate             *rate.Limiter
	PrivateFeeRate              *rate.Limiter
	CMFuturesDefaultRate        *rate.Limiter
	CMFuturesOrderRate          *rate.Limiter
	CMFuturesOrderListRate      *rate.Limiter
	CMFuturesExecutionRate      *rate.Limiter
	CMFuturesPositionRate       *rate.Limiter
	CMFuturesPositionListRate   *rate.Limiter
	CMFuturesFundingRate        *rate.Limiter
	CMFuturesWalletRate         *rate.Limiter
	CMFuturesAccountRate        *rate.Limiter
	UFuturesDefaultRate         *rate.Limiter
	UFuturesOrderRate           *rate.Limiter
	UFuturesPositionRate        *rate.Limiter
	UFuturesPositionListRate    *rate.Limiter
	UFuturesOrderListRate       *rate.Limiter
	UFuturesFundingRate         *rate.Limiter
	FuturesDefaultRate          *rate.Limiter
	FuturesOrderRate            *rate.Limiter
	FuturesOrderListRate        *rate.Limiter
	FuturesExecutionRate        *rate.Limiter
	FuturesPositionRate         *rate.Limiter
	FuturesPositionListRate     *rate.Limiter
	USDCPublic                  *rate.Limiter
	USDCPlaceOrderRate          *rate.Limiter
	USDCModifyOrderRate         *rate.Limiter
	USDCCancelOrderRate         *rate.Limiter
	USDCCancelAllOrderRate      *rate.Limiter
	USDCGetOrderRate            *rate.Limiter
	USDCGetOrderHistoryRate     *rate.Limiter
	USDCGetTradeHistoryRate     *rate.Limiter
	USDCGetTransactionRate      *rate.Limiter
	USDCGetWalletRate           *rate.Limiter
	USDCGetAssetRate            *rate.Limiter
	USDCGetMarginRate           *rate.Limiter
	USDCGetPositionRate         *rate.Limiter
	USDCSetLeverageRate         *rate.Limiter
	USDCGetSettlementRate       *rate.Limiter
	USDCSetRiskRate             *rate.Limiter
	USDCGetPredictedFundingRate *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case publicSpotRate:
		limiter, tokens = r.SpotRate, 1
	case privateSpotRate:
		limiter, tokens = r.PrivateSpotRate, 1
	case privateFeeRate:
		limiter, tokens = r.PrivateFeeRate, 1
	case cFuturesDefaultRate:
		limiter, tokens = r.CMFuturesDefaultRate, 1
	case cFuturesCancelActiveOrderRate, cFuturesCreateConditionalOrderRate, cFuturesCancelConditionalOrderRate, cFuturesReplaceActiveOrderRate,
		cFuturesReplaceConditionalOrderRate, cFuturesCreateOrderRate:
		limiter, tokens = r.CMFuturesOrderRate, 1
	case cFuturesCancelAllActiveOrderRate, cFuturesCancelAllConditionalOrderRate:
		limiter, tokens = r.CMFuturesOrderRate, 10
	case cFuturesGetActiveOrderRate, cFuturesGetConditionalOrderRate, cFuturesGetRealtimeOrderRate:
		limiter, tokens = r.CMFuturesOrderListRate, 1
	case cFuturesTradeRate:
		limiter, tokens = r.CMFuturesExecutionRate, 1
	case cFuturesSetLeverageRate, cFuturesUpdateMarginRate, cFuturesSetTradingRate, cFuturesSwitchPositionRate, cFuturesGetTradingFeeRate:
		limiter, tokens = r.CMFuturesPositionRate, 1
	case cFuturesPositionRate, cFuturesWalletBalanceRate:
		limiter, tokens = r.CMFuturesPositionListRate, 1
	case cFuturesLastFundingFeeRate, cFuturesPredictFundingRate:
		limiter, tokens = r.CMFuturesFundingRate, 1
	case cFuturesWalletFundRecordRate, cFuturesWalletWithdrawalRate:
		limiter, tokens = r.CMFuturesWalletRate, 1
	case cFuturesAPIKeyInfoRate:
		limiter, tokens = r.CMFuturesAccountRate, 1
	case uFuturesDefaultRate:
		limiter, tokens = r.UFuturesDefaultRate, 1
	case uFuturesCreateOrderRate, uFuturesCancelOrderRate, uFuturesCreateConditionalOrderRate, uFuturesCancelConditionalOrderRate:
		limiter, tokens = r.UFuturesOrderRate, 1
	case uFuturesCancelAllOrderRate, uFuturesCancelAllConditionalOrderRate:
		limiter, tokens = r.UFuturesOrderRate, 10
	case uFuturesSetLeverageRate, uFuturesSwitchMargin, uFuturesSwitchPosition, uFuturesSetMarginRate, uFuturesSetTradingStopRate, uFuturesUpdateMarginRate:
		limiter, tokens = r.UFuturesPositionRate, 1
	case uFuturesPositionRate, uFuturesGetClosedTradesRate, uFuturesGetTradesRate:
		limiter, tokens = r.UFuturesPositionListRate, 1
	case uFuturesGetActiveOrderRate, uFuturesGetActiveRealtimeOrderRate, uFuturesGetConditionalOrderRate, uFuturesGetConditionalRealtimeOrderRate:
		limiter, tokens = r.UFuturesOrderListRate, 1
	case uFuturesGetMyLastFundingFeeRate, uFuturesPredictFundingRate:
		limiter, tokens = r.UFuturesFundingRate, 1
	case futuresDefaultRate:
		limiter, tokens = r.FuturesDefaultRate, 1
	case futuresCancelOrderRate, futuresCreateOrderRate, futuresReplaceOrderRate, futuresReplaceConditionalOrderRate, futuresCancelConditionalOrderRate,
		futuresCreateConditionalOrderRate:
		limiter, tokens = r.FuturesOrderRate, 1
	case futuresCancelAllOrderRate, futuresCancelAllConditionalOrderRate:
		limiter, tokens = r.FuturesOrderRate, 10
	case futuresGetActiveOrderRate, futuresGetConditionalOrderRate, futuresGetActiveRealtimeOrderRate, futuresGetConditionalRealtimeOrderRate:
		limiter, tokens = r.FuturesOrderListRate, 1
	case futuresGetTradeRate:
		limiter, tokens = r.FuturesExecutionRate, 1
	case futuresSetLeverageRate, futuresUpdateMarginRate, futuresSetTradingStopRate, futuresSwitchPositionModeRate, futuresSwitchMarginRate, futuresSwitchPositionRate:
		limiter, tokens = r.FuturesPositionRate, 1
	case futuresPositionRate:
		limiter, tokens = r.FuturesPositionListRate, 1
	case usdcPublicRate:
		limiter, tokens = r.USDCPublic, 1
	case usdcCancelAllOrderRate:
		limiter, tokens = r.USDCCancelAllOrderRate, 1
	case usdcPlaceOrderRate:
		limiter, tokens = r.USDCPlaceOrderRate, 1
	case usdcModifyOrderRate:
		limiter, tokens = r.USDCModifyOrderRate, 1
	case usdcCancelOrderRate:
		limiter, tokens = r.USDCCancelOrderRate, 1
	case usdcGetOrderRate:
		limiter, tokens = r.USDCGetOrderRate, 1
	case usdcGetOrderHistoryRate:
		limiter, tokens = r.USDCGetOrderHistoryRate, 1
	case usdcGetTradeHistoryRate:
		limiter, tokens = r.USDCGetTradeHistoryRate, 1
	case usdcGetTransactionRate:
		limiter, tokens = r.USDCGetTransactionRate, 1
	case usdcGetWalletRate:
		limiter, tokens = r.USDCGetWalletRate, 1
	case usdcGetAssetRate:
		limiter, tokens = r.USDCGetAssetRate, 1
	case usdcGetMarginRate:
		limiter, tokens = r.USDCGetMarginRate, 1
	case usdcGetPositionRate:
		limiter, tokens = r.USDCGetPositionRate, 1
	case usdcSetLeverageRate:
		limiter, tokens = r.USDCSetLeverageRate, 1
	case usdcGetSettlementRate:
		limiter, tokens = r.USDCGetSettlementRate, 1
	case usdcSetRiskRate:
		limiter, tokens = r.USDCSetRiskRate, 1
	case usdcGetPredictedFundingRate:
		limiter, tokens = r.USDCGetPredictedFundingRate, 1
	default:
		limiter, tokens = r.SpotRate, 1
	}

	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reserves[i] = limiter.Reserve()
		finalDelay = limiter.Reserve().Delay()
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
		SpotRate:                    request.NewRateLimit(spotInterval, spotRequestRate),
		FuturesRate:                 request.NewRateLimit(futuresPublicInterval, futuresRequestRate),
		PrivateSpotRate:             request.NewRateLimit(spotPrivateInterval, spotPrivateRequestRate),
		PrivateFeeRate:              request.NewRateLimit(spotPrivateInterval, spotPrivateFeeRequestRate),
		CMFuturesDefaultRate:        request.NewRateLimit(futuresInterval, futuresDefaultRateCount),
		CMFuturesOrderRate:          request.NewRateLimit(futuresInterval, futuresOrderRate),
		CMFuturesOrderListRate:      request.NewRateLimit(futuresInterval, futuresOrderListRate),
		CMFuturesExecutionRate:      request.NewRateLimit(futuresInterval, futuresExecutionRate),
		CMFuturesPositionRate:       request.NewRateLimit(futuresInterval, futuresPositionRateCount),
		CMFuturesPositionListRate:   request.NewRateLimit(futuresInterval, futuresPositionListRate),
		CMFuturesFundingRate:        request.NewRateLimit(futuresInterval, futuresFundingRate),
		CMFuturesWalletRate:         request.NewRateLimit(futuresInterval, futuresWalletRate),
		CMFuturesAccountRate:        request.NewRateLimit(futuresInterval, futuresAccountRate),
		UFuturesDefaultRate:         request.NewRateLimit(futuresInterval, futuresDefaultRateCount),
		UFuturesOrderRate:           request.NewRateLimit(futuresInterval, futuresOrderRate),
		UFuturesPositionRate:        request.NewRateLimit(futuresInterval, futuresPositionRateCount),
		UFuturesPositionListRate:    request.NewRateLimit(futuresInterval, futuresPositionListRate),
		UFuturesOrderListRate:       request.NewRateLimit(futuresInterval, futuresOrderListRate),
		UFuturesFundingRate:         request.NewRateLimit(futuresInterval, futuresFundingRate),
		FuturesDefaultRate:          request.NewRateLimit(futuresInterval, futuresDefaultRateCount),
		FuturesOrderRate:            request.NewRateLimit(futuresInterval, futuresOrderRate),
		FuturesOrderListRate:        request.NewRateLimit(futuresInterval, futuresOrderListRate),
		FuturesExecutionRate:        request.NewRateLimit(futuresInterval, futuresExecutionRate),
		FuturesPositionRate:         request.NewRateLimit(futuresInterval, futuresPositionRateCount),
		FuturesPositionListRate:     request.NewRateLimit(futuresInterval, futuresPositionListRate),
		USDCPublic:                  request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPublicRate),
		USDCPlaceOrderRate:          request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCModifyOrderRate:         request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCCancelOrderRate:         request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCCancelAllOrderRate:      request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualCancelAllRate),
		USDCGetOrderRate:            request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetOrderHistoryRate:     request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetTradeHistoryRate:     request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetTransactionRate:      request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetWalletRate:           request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetAssetRate:            request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetMarginRate:           request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetPositionRate:         request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCSetLeverageRate:         request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetSettlementRate:       request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCSetRiskRate:             request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
		USDCGetPredictedFundingRate: request.NewRateLimit(usdcPerpetualInterval, usdcPerpetualPrivateRate),
	}
}
