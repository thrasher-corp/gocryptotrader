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

	createOrderRate      = 10 // 1s
	createSpotOrderRate  = 20 // 1s
	amendOrderRate       = 10 // 1s
	cancelOrderRate      = 10 // 1s
	cancelSpotRate       = 20 // 1s
	calcelAllRate        = 1  // 1s
	cancelAllSpotRate    = 20 // 1s
	createBatchOrderRate = 10 // 1s
	amendBatchOrderRate  = 10 // 1s
	cancelBatchOrderRate = 10 // 1s

	getOrderRate        = 10 // 1s
	getOrderHistoryRate = 10 // 1s

	getPositionListRate      = 10 // 1s
	getExecutionListRate     = 10 // 1s
	getPositionClosedPNLRate = 10 // 1s

	postPOsitionSetLeverageRate = 10 // 1s
	setPositionTPLSModeRate     = 10 // 1s
	setPositionRiskLimitRate    = 10 // 1s
	stopTradingPositionRate     = 10 // 1s

	getAccountWalletBalaceRate = 10 // 1s
	getAccountFeeRate          = 10 // 1s

	getAssetTransferQueryInfoRate             = 60 // 1 min
	getAssetTransferQueryTransferCoinListRate = 60 // 1 min
	getAssetTransferCOinListRate              = 60 // 1min
	getAssetinterTransferListRate             = 60 // 1min
	getSubMemberListRate                      = 60 // 1min
	getAssetUniversalTransferListRate         = 2
	getAssetAccountCoinBalanceRate            = 2
	getAssetDepositRecordsRate                = 300 // 1min
	getAssetDepositSubMemberRecordsRate       = 300 // 1min
	getAssetDepositSubMemberAddressRate       = 300 // 1min

	getWithdrawRecordsRate     = 300 // 1min
	getAssetCoinInfoRate       = 300 // 1min
	getExchangeOrderRecordRate = 300 // 1min

	interTransferRate         = 20 // 1min
	saveTransferSubMemberRate = 20 // 1min
	universalTransferRate     = 5
	createWithdrawalRate      = 10 // 1min
	cancelWithdrawalRate      = 60 // 1min

	userCreateSubMemberRate = 5  // 1s
	userCreateSubAPIKeyRate = 5  // 1s
	userFrozenSubMemberRate = 5  // 1s
	userUpdateAPIRate       = 5  // 1s
	userUpdateSubAPIRate    = 5  // 1s
	userDeleteAPIRate       = 5  // 1s
	userDeleteSubAPIRate    = 5  // 1s
	userQuerySubMembersRate = 10 // 1s
	userQueryAPIRate        = 10 // 1s

	getSpotLeverageTokenOrderRecordsRate = 50 // 1s
	spotLeverageTokenPurchaseRate        = 20 // 1s
	spotLeverTokenRedeemRate             = 20 // 1s

	getSpotCrossMarginTradeLoanInfoRate     = 50 // 1s
	getSpotCrossMarginTradeAccountRate      = 50 // 1s
	getSpotCrossMarginTradeOrdersRate       = 50 // 1s
	getSpotCrossMarginTradeRepayHistoryRate = 50 // 1s
	spotCrossMarginTradeLoanRate            = 20 // 1s
	spotCrossMarginTradeRepayRate           = 20 // 1s
	spotCrossMarginTradeSwitchRate          = 20 // 1s

	// intervals 1s, 1min, 5sec

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
	defaultEPL request.EndpointLimit = iota
	createOrderEPL
	createSpotOrderEPL
	amendOrderEPL
	cancelOrderEPL
	cancelSpotEPL
	calcelAllEPL
	cancelAllSpotEPL
	createBatchOrderEPL
	amendBatchOrderEPL
	cancelBatchOrderEPL
	getOrderEPL
	getOrderHistoryEPL
	getPositionListEPL
	getExecutionListEPL
	getPositionClosedPNLEPL
	postPOsitionSetLeverageEPL
	setPositionTPLSModeEPL
	setPositionRiskLimitEPL
	stopTradingPositionEPL
	getAccountWalletBalaceEPL
	getAccountFeeEPL
	getAssetTransferQueryInfoEPL
	getAssetTransferQueryTransferCoinListEPL
	getAssetTransferCOinListEPL
	getAssetinterTransferListEPL
	getSubMemberListEPL
	getAssetUniversalTransferListEPL
	getAssetAccountCoinBalanceEPL
	getAssetDepositRecordsEPL
	getAssetDepositSubMemberRecordsEPL
	getAssetDepositSubMemberAddressEPL
	getWithdrawRecordsEPL
	getAssetCoinInfoEPL
	getExchangeOrderRecordEPL
	interTransferEPL
	saveTransferSubMemberEPL
	universalTransferEPL
	createWithdrawalEPL
	cancelWithdrawalEPL
	userCreateSubMemberEPL
	userCreateSubAPIKeyEPL
	userFrozenSubMemberEPL
	userUpdateAPIEPL
	userUpdateSubAPIEPL
	userDeleteAPIEPL
	userDeleteSubAPIEPL
	userQuerySubMembersEPL
	userQueryAPIEPL
	getSpotLeverageTokenOrderRecordsEPL
	spotLeverageTokenPurchaseEPL
	spotLeverTokenRedeemEPL
	getSpotCrossMarginTradeLoanInfoEPL
	getSpotCrossMarginTradeAccountEPL
	getSpotCrossMarginTradeOrdersEPL
	getSpotCrossMarginTradeRepayHistoryEPL
	spotCrossMarginTradeLoanEPL
	spotCrossMarginTradeRepayEPL
	spotCrossMarginTradeSwitchEPL
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate                                  *rate.Limiter
	CreateOrderRate                           *rate.Limiter
	CreateSpotOrderRate                       *rate.Limiter
	AmendOrderRate                            *rate.Limiter
	CancelOrderRate                           *rate.Limiter
	CancelSpotRate                            *rate.Limiter
	CalcelAllRate                             *rate.Limiter
	CancelAllSpotRate                         *rate.Limiter
	CreateBatchOrderRate                      *rate.Limiter
	AmendBatchOrderRate                       *rate.Limiter
	CancelBatchOrderRate                      *rate.Limiter
	GetOrderRate                              *rate.Limiter
	GetOrderHistoryRate                       *rate.Limiter
	GetPositionListRate                       *rate.Limiter
	GetExecutionListRate                      *rate.Limiter
	GetPositionClosedPNLRate                  *rate.Limiter
	PostPOsitionSetLeverageRate               *rate.Limiter
	SetPositionTPLSModeRate                   *rate.Limiter
	SetPositionRiskLimitRate                  *rate.Limiter
	StopTradingPositionRate                   *rate.Limiter
	GetAccountWalletBalaceRate                *rate.Limiter
	GetAccountFeeRate                         *rate.Limiter
	GetAssetTransferQueryInfoRate             *rate.Limiter
	GetAssetTransferQueryTransferCoinListRate *rate.Limiter
	GetAssetTransferCOinListRate              *rate.Limiter
	GetAssetinterTransferListRate             *rate.Limiter
	GetSubMemberListRate                      *rate.Limiter
	GetAssetUniversalTransferListRate         *rate.Limiter
	GetAssetAccountCoinBalanceRate            *rate.Limiter
	GetAssetDepositRecordsRate                *rate.Limiter
	GetAssetDepositSubMemberRecordsRate       *rate.Limiter
	GetAssetDepositSubMemberAddressRate       *rate.Limiter
	GetWithdrawRecordsRate                    *rate.Limiter
	GetAssetCoinInfoRate                      *rate.Limiter
	GetExchangeOrderRecordRate                *rate.Limiter
	InterTransferRate                         *rate.Limiter
	SaveTransferSubMemberRate                 *rate.Limiter
	UniversalTransferRate                     *rate.Limiter
	CreateWithdrawalRate                      *rate.Limiter
	CancelWithdrawalRate                      *rate.Limiter
	UserCreateSubMemberRate                   *rate.Limiter
	UserCreateSubAPIKeyRate                   *rate.Limiter
	UserFrozenSubMemberRate                   *rate.Limiter
	UserUpdateAPIRate                         *rate.Limiter
	UserUpdateSubAPIRate                      *rate.Limiter
	UserDeleteAPIRate                         *rate.Limiter
	UserDeleteSubAPIRate                      *rate.Limiter
	UserQuerySubMembersRate                   *rate.Limiter
	UserQueryAPIRate                          *rate.Limiter
	GetSpotLeverageTokenOrderRecordsRate      *rate.Limiter
	SpotLeverageTokenPurchaseRate             *rate.Limiter
	SpotLeverTokenRedeemRate                  *rate.Limiter
	GetSpotCrossMarginTradeLoanInfoRate       *rate.Limiter
	GetSpotCrossMarginTradeAccountRate        *rate.Limiter
	GetSpotCrossMarginTradeOrdersRate         *rate.Limiter
	GetSpotCrossMarginTradeRepayHistoryRate   *rate.Limiter
	SpotCrossMarginTradeLoanRate              *rate.Limiter
	SpotCrossMarginTradeRepayRate             *rate.Limiter
	SpotCrossMarginTradeSwitchRate            *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case defaultEPL:
		limiter, tokens = r.SpotRate, 1
	case createOrderEPL:
		limiter, tokens = r.CreateOrderRate, 10
	case createSpotOrderEPL:
		limiter, tokens = r.CreateSpotOrderRate, 20
	case amendOrderEPL:
		limiter, tokens = r.AmendOrderRate, 10
	case cancelOrderEPL:
		limiter, tokens = r.CancelOrderRate, 10
	case cancelSpotEPL:
		limiter, tokens = r.CancelSpotRate, 20
	case calcelAllEPL:
		limiter, tokens = r.CalcelAllRate, 1
	case cancelAllSpotEPL:
		limiter, tokens = r.CancelAllSpotRate, 20
	case createBatchOrderEPL:
		limiter, tokens = r.CreateBatchOrderRate, 10
	case amendBatchOrderEPL:
		limiter, tokens = r.AmendBatchOrderRate, 10
	case cancelBatchOrderEPL:
		limiter, tokens = r.CancelBatchOrderRate, 10
	case getOrderEPL:
		limiter, tokens = r.GetOrderRate, 10
	case getOrderHistoryEPL:
		limiter, tokens = r.GetOrderHistoryRate, 10
	case getPositionListEPL:
		limiter, tokens = r.GetPositionListRate, 10
	case getExecutionListEPL:
		limiter, tokens = r.GetExecutionListRate, 10
	case getPositionClosedPNLEPL:
		limiter, tokens = r.GetPositionClosedPNLRate, 10
	case postPOsitionSetLeverageEPL:
		limiter, tokens = r.PostPOsitionSetLeverageRate, 10
	case setPositionTPLSModeEPL:
		limiter, tokens = r.SetPositionTPLSModeRate, 10
	case setPositionRiskLimitEPL:
		limiter, tokens = r.SetPositionRiskLimitRate, 10
	case stopTradingPositionEPL:
		limiter, tokens = r.StopTradingPositionRate, 10
	case getAccountWalletBalaceEPL:
		limiter, tokens = r.GetAccountWalletBalaceRate, 10
	case getAccountFeeEPL:
		limiter, tokens = r.GetAccountFeeRate, 10
	case getAssetTransferQueryInfoEPL:
		limiter, tokens = r.GetAssetTransferQueryInfoRate, 1
	case getAssetTransferQueryTransferCoinListEPL:
		limiter, tokens = r.GetAssetTransferQueryTransferCoinListRate, 1
	case getAssetTransferCOinListEPL:
		limiter, tokens = r.GetAssetTransferCOinListRate, 1
	case getAssetinterTransferListEPL:
		limiter, tokens = r.GetAssetinterTransferListRate, 1
	case getSubMemberListEPL:
		limiter, tokens = r.GetSubMemberListRate, 1
	case getAssetUniversalTransferListEPL:
		limiter, tokens = r.GetAssetUniversalTransferListRate, 2
	case getAssetAccountCoinBalanceEPL:
		limiter, tokens = r.GetAssetAccountCoinBalanceRate, 2
	case getAssetDepositRecordsEPL:
		limiter, tokens = r.GetAssetDepositRecordsRate, 1
	case getAssetDepositSubMemberRecordsEPL:
		limiter, tokens = r.GetAssetDepositSubMemberRecordsRate, 1
	case getAssetDepositSubMemberAddressEPL:
		limiter, tokens = r.GetAssetDepositSubMemberAddressRate, 1
	case getWithdrawRecordsEPL:
		limiter, tokens = r.GetWithdrawRecordsRate, 1
	case getAssetCoinInfoEPL:
		limiter, tokens = r.GetAssetCoinInfoRate, 1
	case getExchangeOrderRecordEPL:
		limiter, tokens = r.GetExchangeOrderRecordRate, 1
	case interTransferEPL:
		limiter, tokens = r.InterTransferRate, 1
	case saveTransferSubMemberEPL:
		limiter, tokens = r.SaveTransferSubMemberRate, 1
	case universalTransferEPL:
		limiter, tokens = r.UniversalTransferRate, 5
	case createWithdrawalEPL:
		limiter, tokens = r.CreateWithdrawalRate, 1
	case cancelWithdrawalEPL:
		limiter, tokens = r.CancelWithdrawalRate, 1
	case userCreateSubMemberEPL:
		limiter, tokens = r.UserCreateSubMemberRate, 5
	case userCreateSubAPIKeyEPL:
		limiter, tokens = r.UserCreateSubAPIKeyRate, 5
	case userFrozenSubMemberEPL:
		limiter, tokens = r.UserFrozenSubMemberRate, 5
	case userUpdateAPIEPL:
		limiter, tokens = r.UserUpdateAPIRate, 5
	case userUpdateSubAPIEPL:
		limiter, tokens = r.UserUpdateSubAPIRate, 5
	case userDeleteAPIEPL:
		limiter, tokens = r.UserDeleteAPIRate, 5
	case userDeleteSubAPIEPL:
		limiter, tokens = r.UserDeleteSubAPIRate, 5
	case userQuerySubMembersEPL:
		limiter, tokens = r.UserQuerySubMembersRate, 10
	case userQueryAPIEPL:
		limiter, tokens = r.UserQueryAPIRate, 10
	case getSpotLeverageTokenOrderRecordsEPL:
		limiter, tokens = r.GetSpotLeverageTokenOrderRecordsRate, 50
	case spotLeverageTokenPurchaseEPL:
		limiter, tokens = r.SpotLeverageTokenPurchaseRate, 20
	case spotLeverTokenRedeemEPL:
		limiter, tokens = r.SpotLeverTokenRedeemRate, 20
	case getSpotCrossMarginTradeLoanInfoEPL:
		limiter, tokens = r.GetSpotCrossMarginTradeLoanInfoRate, 50
	case getSpotCrossMarginTradeAccountEPL:
		limiter, tokens = r.GetSpotCrossMarginTradeAccountRate, 50
	case getSpotCrossMarginTradeOrdersEPL:
		limiter, tokens = r.GetSpotCrossMarginTradeOrdersRate, 50
	case getSpotCrossMarginTradeRepayHistoryEPL:
		limiter, tokens = r.GetSpotCrossMarginTradeRepayHistoryRate, 50
	case spotCrossMarginTradeLoanEPL:
		limiter, tokens = r.SpotCrossMarginTradeLoanRate, 50
	case spotCrossMarginTradeRepayEPL:
		limiter, tokens = r.SpotCrossMarginTradeRepayRate, 50
	case spotCrossMarginTradeSwitchEPL:
		limiter, tokens = r.SpotCrossMarginTradeSwitchRate, 50
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
		SpotRate:                                  request.NewRateLimit(spotInterval, spotRequestRate),
		CreateOrderRate:                           request.NewRateLimit(time.Second, createOrderRate),
		CreateSpotOrderRate:                       request.NewRateLimit(time.Second, createSpotOrderRate),
		AmendOrderRate:                            request.NewRateLimit(time.Second, amendOrderRate),
		CancelOrderRate:                           request.NewRateLimit(time.Second, cancelOrderRate),
		CancelSpotRate:                            request.NewRateLimit(time.Second, cancelSpotRate),
		CalcelAllRate:                             request.NewRateLimit(time.Second, calcelAllRate),
		CancelAllSpotRate:                         request.NewRateLimit(time.Second, cancelAllSpotRate),
		CreateBatchOrderRate:                      request.NewRateLimit(time.Second, createBatchOrderRate),
		AmendBatchOrderRate:                       request.NewRateLimit(time.Second, amendBatchOrderRate),
		CancelBatchOrderRate:                      request.NewRateLimit(time.Second, cancelBatchOrderRate),
		GetOrderRate:                              request.NewRateLimit(time.Second, getOrderRate),
		GetOrderHistoryRate:                       request.NewRateLimit(time.Second, getOrderHistoryRate),
		GetPositionListRate:                       request.NewRateLimit(time.Second, getPositionListRate),
		GetExecutionListRate:                      request.NewRateLimit(time.Second, getExecutionListRate),
		GetPositionClosedPNLRate:                  request.NewRateLimit(time.Second, getPositionClosedPNLRate),
		PostPOsitionSetLeverageRate:               request.NewRateLimit(time.Second, postPOsitionSetLeverageRate),
		SetPositionTPLSModeRate:                   request.NewRateLimit(time.Second, setPositionTPLSModeRate),
		SetPositionRiskLimitRate:                  request.NewRateLimit(time.Second, setPositionRiskLimitRate),
		StopTradingPositionRate:                   request.NewRateLimit(time.Second, stopTradingPositionRate),
		GetAccountWalletBalaceRate:                request.NewRateLimit(time.Second, getAccountWalletBalaceRate),
		GetAccountFeeRate:                         request.NewRateLimit(time.Second, getAccountFeeRate),
		GetAssetTransferQueryInfoRate:             request.NewRateLimit(time.Minute, getAssetTransferQueryInfoRate),
		GetAssetTransferQueryTransferCoinListRate: request.NewRateLimit(time.Minute, getAssetTransferQueryTransferCoinListRate),
		GetAssetTransferCOinListRate:              request.NewRateLimit(time.Minute, getAssetTransferCOinListRate),
		GetAssetinterTransferListRate:             request.NewRateLimit(time.Minute, getAssetinterTransferListRate),
		GetSubMemberListRate:                      request.NewRateLimit(time.Minute, getSubMemberListRate),
		GetAssetUniversalTransferListRate:         request.NewRateLimit(time.Second, getAssetUniversalTransferListRate),
		GetAssetAccountCoinBalanceRate:            request.NewRateLimit(time.Second, getAssetAccountCoinBalanceRate),
		GetAssetDepositRecordsRate:                request.NewRateLimit(time.Minute, getAssetDepositRecordsRate),
		GetAssetDepositSubMemberRecordsRate:       request.NewRateLimit(time.Minute, getAssetDepositSubMemberRecordsRate),
		GetAssetDepositSubMemberAddressRate:       request.NewRateLimit(time.Minute, getAssetDepositSubMemberAddressRate),
		GetWithdrawRecordsRate:                    request.NewRateLimit(time.Minute, getWithdrawRecordsRate),
		GetAssetCoinInfoRate:                      request.NewRateLimit(time.Minute, getAssetCoinInfoRate),
		GetExchangeOrderRecordRate:                request.NewRateLimit(time.Minute, getExchangeOrderRecordRate),
		InterTransferRate:                         request.NewRateLimit(time.Minute, interTransferRate),
		SaveTransferSubMemberRate:                 request.NewRateLimit(time.Minute, saveTransferSubMemberRate),
		UniversalTransferRate:                     request.NewRateLimit(time.Second, universalTransferRate),
		CreateWithdrawalRate:                      request.NewRateLimit(time.Minute, createWithdrawalRate),
		CancelWithdrawalRate:                      request.NewRateLimit(time.Minute, cancelWithdrawalRate),
		UserCreateSubMemberRate:                   request.NewRateLimit(time.Second, userCreateSubMemberRate),
		UserCreateSubAPIKeyRate:                   request.NewRateLimit(time.Second, userCreateSubAPIKeyRate),
		UserFrozenSubMemberRate:                   request.NewRateLimit(time.Second, userFrozenSubMemberRate),
		UserUpdateAPIRate:                         request.NewRateLimit(time.Second, userUpdateAPIRate),
		UserUpdateSubAPIRate:                      request.NewRateLimit(time.Second, userUpdateSubAPIRate),
		UserDeleteAPIRate:                         request.NewRateLimit(time.Second, userDeleteAPIRate),
		UserDeleteSubAPIRate:                      request.NewRateLimit(time.Second, userDeleteSubAPIRate),
		UserQuerySubMembersRate:                   request.NewRateLimit(time.Second, userQuerySubMembersRate),
		UserQueryAPIRate:                          request.NewRateLimit(time.Second, userQueryAPIRate),
		GetSpotLeverageTokenOrderRecordsRate:      request.NewRateLimit(time.Second, getSpotLeverageTokenOrderRecordsRate),
		SpotLeverageTokenPurchaseRate:             request.NewRateLimit(time.Second, spotLeverageTokenPurchaseRate),
		SpotLeverTokenRedeemRate:                  request.NewRateLimit(time.Second, spotLeverTokenRedeemRate),
		GetSpotCrossMarginTradeLoanInfoRate:       request.NewRateLimit(time.Second, getSpotCrossMarginTradeLoanInfoRate),
		GetSpotCrossMarginTradeAccountRate:        request.NewRateLimit(time.Second, getSpotCrossMarginTradeAccountRate),
		GetSpotCrossMarginTradeOrdersRate:         request.NewRateLimit(time.Second, getSpotCrossMarginTradeOrdersRate),
		GetSpotCrossMarginTradeRepayHistoryRate:   request.NewRateLimit(time.Second, getSpotCrossMarginTradeRepayHistoryRate),
		SpotCrossMarginTradeLoanRate:              request.NewRateLimit(time.Second, spotCrossMarginTradeLoanRate),
		SpotCrossMarginTradeRepayRate:             request.NewRateLimit(time.Second, spotCrossMarginTradeRepayRate),
		SpotCrossMarginTradeSwitchRate:            request.NewRateLimit(time.Second, spotCrossMarginTradeSwitchRate),
	}
}
