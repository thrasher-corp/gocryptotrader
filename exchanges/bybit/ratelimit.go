package bybit

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	spotInterval          = time.Second
	spotRequestRate       = 70
	futuresPublicInterval = time.Second
	futuresRequestRate    = 50

	spotPrivateRequestRate   = 20
	futuresInterval          = time.Minute
	futuresDefaultRateCount  = 100
	futuresOrderRate         = 100
	futuresOrderListRate     = 600
	futuresExecutionRate     = 120
	futuresPositionRateCount = 75
	futuresPositionListRate  = 120
	futuresFundingRate       = 120
	futuresWalletRate        = 120
	futuresAccountRate       = 600

	usdcPerpetualPublicRate    = 50
	usdcPerpetualCancelAllRate = 1
	usdcPerpetualPrivateRate   = 5
	usdcPerpetualInterval      = time.Second
)

const (
	publicSpotRate request.EndpointLimit = iota
	publicFuturesRate
	privateSpotRate

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
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, request.Tokens, error) {
	switch f {
	case publicSpotRate:
		return r.SpotRate, 1, nil
	case privateSpotRate:
		return r.PrivateSpotRate, 1, nil
	case cFuturesDefaultRate:
		return r.CMFuturesDefaultRate, 1, nil
	case cFuturesCancelActiveOrderRate, cFuturesCreateConditionalOrderRate, cFuturesCancelConditionalOrderRate, cFuturesReplaceActiveOrderRate,
		cFuturesReplaceConditionalOrderRate, cFuturesCreateOrderRate:
		return r.CMFuturesOrderRate, 1, nil
	case cFuturesCancelAllActiveOrderRate, cFuturesCancelAllConditionalOrderRate:
		return r.CMFuturesOrderRate, 10, nil
	case cFuturesGetActiveOrderRate, cFuturesGetConditionalOrderRate, cFuturesGetRealtimeOrderRate:
		return r.CMFuturesOrderListRate, 1, nil
	case cFuturesTradeRate:
		return r.CMFuturesExecutionRate, 1, nil
	case cFuturesSetLeverageRate, cFuturesUpdateMarginRate, cFuturesSetTradingRate, cFuturesSwitchPositionRate, cFuturesGetTradingFeeRate:
		return r.CMFuturesPositionRate, 1, nil
	case cFuturesPositionRate, cFuturesWalletBalanceRate:
		return r.CMFuturesPositionListRate, 1, nil
	case cFuturesLastFundingFeeRate, cFuturesPredictFundingRate:
		return r.CMFuturesFundingRate, 1, nil
	case cFuturesWalletFundRecordRate, cFuturesWalletWithdrawalRate:
		return r.CMFuturesWalletRate, 1, nil
	case cFuturesAPIKeyInfoRate:
		return r.CMFuturesAccountRate, 1, nil
	case uFuturesDefaultRate:
		return r.UFuturesDefaultRate, 1, nil
	case uFuturesCreateOrderRate, uFuturesCancelOrderRate, uFuturesCreateConditionalOrderRate, uFuturesCancelConditionalOrderRate:
		return r.UFuturesOrderRate, 1, nil
	case uFuturesCancelAllOrderRate, uFuturesCancelAllConditionalOrderRate:
		return r.UFuturesOrderRate, 10, nil
	case uFuturesSetLeverageRate, uFuturesSwitchMargin, uFuturesSwitchPosition, uFuturesSetMarginRate, uFuturesSetTradingStopRate, uFuturesUpdateMarginRate:
		return r.UFuturesPositionRate, 1, nil
	case uFuturesPositionRate, uFuturesGetClosedTradesRate, uFuturesGetTradesRate:
		return r.UFuturesPositionListRate, 1, nil
	case uFuturesGetActiveOrderRate, uFuturesGetActiveRealtimeOrderRate, uFuturesGetConditionalOrderRate, uFuturesGetConditionalRealtimeOrderRate:
		return r.UFuturesOrderListRate, 1, nil
	case uFuturesGetMyLastFundingFeeRate, uFuturesPredictFundingRate:
		return r.UFuturesFundingRate, 1, nil
	case futuresDefaultRate:
		return r.FuturesDefaultRate, 1, nil
	case futuresCancelOrderRate, futuresCreateOrderRate, futuresReplaceOrderRate, futuresReplaceConditionalOrderRate, futuresCancelConditionalOrderRate,
		futuresCreateConditionalOrderRate:
		return r.FuturesOrderRate, 1, nil
	case futuresCancelAllOrderRate, futuresCancelAllConditionalOrderRate:
		return r.FuturesOrderRate, 10, nil
	case futuresGetActiveOrderRate, futuresGetConditionalOrderRate, futuresGetActiveRealtimeOrderRate, futuresGetConditionalRealtimeOrderRate:
		return r.FuturesOrderListRate, 1, nil
	case futuresGetTradeRate:
		return r.FuturesExecutionRate, 1, nil
	case futuresSetLeverageRate, futuresUpdateMarginRate, futuresSetTradingStopRate, futuresSwitchPositionModeRate, futuresSwitchMarginRate, futuresSwitchPositionRate:
		return r.FuturesPositionRate, 1, nil
	case futuresPositionRate:
		return r.FuturesPositionListRate, 1, nil
	case usdcPublicRate:
		return r.USDCPublic, 1, nil
	case usdcCancelAllOrderRate:
		return r.USDCCancelAllOrderRate, 1, nil
	case usdcPlaceOrderRate:
		return r.USDCPlaceOrderRate, 1, nil
	case usdcModifyOrderRate:
		return r.USDCModifyOrderRate, 1, nil
	case usdcCancelOrderRate:
		return r.USDCCancelOrderRate, 1, nil
	case usdcGetOrderRate:
		return r.USDCGetOrderRate, 1, nil
	case usdcGetOrderHistoryRate:
		return r.USDCGetOrderHistoryRate, 1, nil
	case usdcGetTradeHistoryRate:
		return r.USDCGetTradeHistoryRate, 1, nil
	case usdcGetTransactionRate:
		return r.USDCGetTransactionRate, 1, nil
	case usdcGetWalletRate:
		return r.USDCGetWalletRate, 1, nil
	case usdcGetAssetRate:
		return r.USDCGetAssetRate, 1, nil
	case usdcGetMarginRate:
		return r.USDCGetMarginRate, 1, nil
	case usdcGetPositionRate:
		return r.USDCGetPositionRate, 1, nil
	case usdcSetLeverageRate:
		return r.USDCSetLeverageRate, 1, nil
	case usdcGetSettlementRate:
		return r.USDCGetSettlementRate, 1, nil
	case usdcSetRiskRate:
		return r.USDCSetRiskRate, 1, nil
	case usdcGetPredictedFundingRate:
		return r.USDCGetPredictedFundingRate, 1, nil
	default:
		return r.SpotRate, 1, nil
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		SpotRate:                    request.NewRateLimit(spotInterval, spotRequestRate),
		FuturesRate:                 request.NewRateLimit(futuresPublicInterval, futuresRequestRate),
		PrivateSpotRate:             request.NewRateLimit(spotInterval, spotPrivateRequestRate),
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
