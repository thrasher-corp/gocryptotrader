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
	spotInterval = time.Second * 5
)

const (
	defaultEPL request.EndpointLimit = iota
	createOrderEPL
	createSpotOrderEPL
	amendOrderEPL
	cancelOrderEPL
	cancelSpotEPL
	cancelAllEPL
	cancelAllSpotEPL
	createBatchOrderEPL
	amendBatchOrderEPL
	cancelBatchOrderEPL
	getOrderEPL
	getOrderHistoryEPL
	getPositionListEPL
	getExecutionListEPL
	getPositionClosedPNLEPL
	postPositionSetLeverageEPL
	setPositionTPLSModeEPL
	setPositionRiskLimitEPL
	stopTradingPositionEPL
	getAccountWalletBalanceEPL
	getAccountFeeEPL
	getAssetTransferQueryInfoEPL
	getAssetTransferQueryTransferCoinListEPL
	getAssetTransferCoinListEPL
	getAssetInterTransferListEPL
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
	CancelAllRate                             *rate.Limiter
	CancelAllSpotRate                         *rate.Limiter
	CreateBatchOrderRate                      *rate.Limiter
	AmendBatchOrderRate                       *rate.Limiter
	CancelBatchOrderRate                      *rate.Limiter
	GetOrderRate                              *rate.Limiter
	GetOrderHistoryRate                       *rate.Limiter
	GetPositionListRate                       *rate.Limiter
	GetExecutionListRate                      *rate.Limiter
	GetPositionClosedPNLRate                  *rate.Limiter
	PostPositionSetLeverageRate               *rate.Limiter
	SetPositionTPLSModeRate                   *rate.Limiter
	SetPositionRiskLimitRate                  *rate.Limiter
	StopTradingPositionRate                   *rate.Limiter
	GetAccountWalletBalanceRate               *rate.Limiter
	GetAccountFeeRate                         *rate.Limiter
	GetAssetTransferQueryInfoRate             *rate.Limiter
	GetAssetTransferQueryTransferCoinListRate *rate.Limiter
	GetAssetTransferCoinListRate              *rate.Limiter
	GetAssetInterTransferListRate             *rate.Limiter
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
	case cancelAllEPL:
		limiter, tokens = r.CancelAllRate, 1
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
	case postPositionSetLeverageEPL:
		limiter, tokens = r.PostPositionSetLeverageRate, 10
	case setPositionTPLSModeEPL:
		limiter, tokens = r.SetPositionTPLSModeRate, 10
	case setPositionRiskLimitEPL:
		limiter, tokens = r.SetPositionRiskLimitRate, 10
	case stopTradingPositionEPL:
		limiter, tokens = r.StopTradingPositionRate, 10
	case getAccountWalletBalanceEPL:
		limiter, tokens = r.GetAccountWalletBalanceRate, 10
	case getAccountFeeEPL:
		limiter, tokens = r.GetAccountFeeRate, 10
	case getAssetTransferQueryInfoEPL:
		limiter, tokens = r.GetAssetTransferQueryInfoRate, 1
	case getAssetTransferQueryTransferCoinListEPL:
		limiter, tokens = r.GetAssetTransferQueryTransferCoinListRate, 1
	case getAssetTransferCoinListEPL:
		limiter, tokens = r.GetAssetTransferCoinListRate, 1
	case getAssetInterTransferListEPL:
		limiter, tokens = r.GetAssetInterTransferListRate, 1
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
		SpotRate:                                  request.NewRateLimit(spotInterval, 120),
		CreateOrderRate:                           request.NewRateLimit(time.Second, 10),
		CreateSpotOrderRate:                       request.NewRateLimit(time.Second, 20),
		AmendOrderRate:                            request.NewRateLimit(time.Second, 10),
		CancelOrderRate:                           request.NewRateLimit(time.Second, 10),
		CancelSpotRate:                            request.NewRateLimit(time.Second, 20),
		CancelAllRate:                             request.NewRateLimit(time.Second, 1),
		CancelAllSpotRate:                         request.NewRateLimit(time.Second, 20),
		CreateBatchOrderRate:                      request.NewRateLimit(time.Second, 10),
		AmendBatchOrderRate:                       request.NewRateLimit(time.Second, 10),
		CancelBatchOrderRate:                      request.NewRateLimit(time.Second, 10),
		GetOrderRate:                              request.NewRateLimit(time.Second, 10),
		GetOrderHistoryRate:                       request.NewRateLimit(time.Second, 10),
		GetPositionListRate:                       request.NewRateLimit(time.Second, 10),
		GetExecutionListRate:                      request.NewRateLimit(time.Second, 10),
		GetPositionClosedPNLRate:                  request.NewRateLimit(time.Second, 10),
		PostPositionSetLeverageRate:               request.NewRateLimit(time.Second, 10),
		SetPositionTPLSModeRate:                   request.NewRateLimit(time.Second, 10),
		SetPositionRiskLimitRate:                  request.NewRateLimit(time.Second, 10),
		StopTradingPositionRate:                   request.NewRateLimit(time.Second, 10),
		GetAccountWalletBalanceRate:               request.NewRateLimit(time.Second, 10),
		GetAccountFeeRate:                         request.NewRateLimit(time.Second, 10),
		GetAssetTransferQueryInfoRate:             request.NewRateLimit(time.Minute, 60),
		GetAssetTransferQueryTransferCoinListRate: request.NewRateLimit(time.Minute, 60),
		GetAssetTransferCoinListRate:              request.NewRateLimit(time.Minute, 60),
		GetAssetInterTransferListRate:             request.NewRateLimit(time.Minute, 60),
		GetSubMemberListRate:                      request.NewRateLimit(time.Minute, 60),
		GetAssetUniversalTransferListRate:         request.NewRateLimit(time.Second, 2),
		GetAssetAccountCoinBalanceRate:            request.NewRateLimit(time.Second, 2),
		GetAssetDepositRecordsRate:                request.NewRateLimit(time.Minute, 300),
		GetAssetDepositSubMemberRecordsRate:       request.NewRateLimit(time.Minute, 300),
		GetAssetDepositSubMemberAddressRate:       request.NewRateLimit(time.Minute, 300),
		GetWithdrawRecordsRate:                    request.NewRateLimit(time.Minute, 300),
		GetAssetCoinInfoRate:                      request.NewRateLimit(time.Minute, 300),
		GetExchangeOrderRecordRate:                request.NewRateLimit(time.Minute, 300),
		InterTransferRate:                         request.NewRateLimit(time.Minute, 20),
		SaveTransferSubMemberRate:                 request.NewRateLimit(time.Minute, 20),
		UniversalTransferRate:                     request.NewRateLimit(time.Second, 5),
		CreateWithdrawalRate:                      request.NewRateLimit(time.Second, 1),
		CancelWithdrawalRate:                      request.NewRateLimit(time.Minute, 60),
		UserCreateSubMemberRate:                   request.NewRateLimit(time.Second, 5),
		UserCreateSubAPIKeyRate:                   request.NewRateLimit(time.Second, 5),
		UserFrozenSubMemberRate:                   request.NewRateLimit(time.Second, 5),
		UserUpdateAPIRate:                         request.NewRateLimit(time.Second, 5),
		UserUpdateSubAPIRate:                      request.NewRateLimit(time.Second, 5),
		UserDeleteAPIRate:                         request.NewRateLimit(time.Second, 5),
		UserDeleteSubAPIRate:                      request.NewRateLimit(time.Second, 5),
		UserQuerySubMembersRate:                   request.NewRateLimit(time.Second, 10),
		UserQueryAPIRate:                          request.NewRateLimit(time.Second, 10),
		GetSpotLeverageTokenOrderRecordsRate:      request.NewRateLimit(time.Second, 50),
		SpotLeverageTokenPurchaseRate:             request.NewRateLimit(time.Second, 20),
		SpotLeverTokenRedeemRate:                  request.NewRateLimit(time.Second, 20),
		GetSpotCrossMarginTradeLoanInfoRate:       request.NewRateLimit(time.Second, 50),
		GetSpotCrossMarginTradeAccountRate:        request.NewRateLimit(time.Second, 50),
		GetSpotCrossMarginTradeOrdersRate:         request.NewRateLimit(time.Second, 50),
		GetSpotCrossMarginTradeRepayHistoryRate:   request.NewRateLimit(time.Second, 50),
		SpotCrossMarginTradeLoanRate:              request.NewRateLimit(time.Second, 20),
		SpotCrossMarginTradeRepayRate:             request.NewRateLimit(time.Second, 20),
		SpotCrossMarginTradeSwitchRate:            request.NewRateLimit(time.Second, 20),
	}
}
