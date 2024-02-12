package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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

// 	case createOrderEPL:
// 		limiter, tokens = r.CreateOrderRate, 10
// 	case createSpotOrderEPL:
// 		limiter, tokens = r.CreateSpotOrderRate, 20
// 	case amendOrderEPL:
// 		limiter, tokens = r.AmendOrderRate, 10
// 	case cancelOrderEPL:
// 		limiter, tokens = r.CancelOrderRate, 10
// 	case cancelSpotEPL:
// 		limiter, tokens = r.CancelSpotRate, 20
// 	case cancelAllEPL:
// 		limiter, tokens = r.CancelAllRate, 1
// 	case cancelAllSpotEPL:
// 		limiter, tokens = r.CancelAllSpotRate, 20
// 	case createBatchOrderEPL:
// 		limiter, tokens = r.CreateBatchOrderRate, 10
// 	case amendBatchOrderEPL:
// 		limiter, tokens = r.AmendBatchOrderRate, 10
// 	case cancelBatchOrderEPL:
// 		limiter, tokens = r.CancelBatchOrderRate, 10
// 	case getOrderEPL:
// 		limiter, tokens = r.GetOrderRate, 10
// 	case getOrderHistoryEPL:
// 		limiter, tokens = r.GetOrderHistoryRate, 10
// 	case getPositionListEPL:
// 		limiter, tokens = r.GetPositionListRate, 10
// 	case getExecutionListEPL:
// 		limiter, tokens = r.GetExecutionListRate, 10
// 	case getPositionClosedPNLEPL:
// 		limiter, tokens = r.GetPositionClosedPNLRate, 10
// 	case postPositionSetLeverageEPL:
// 		limiter, tokens = r.PostPositionSetLeverageRate, 10
// 	case setPositionTPLSModeEPL:
// 		limiter, tokens = r.SetPositionTPLSModeRate, 10
// 	case setPositionRiskLimitEPL:
// 		limiter, tokens = r.SetPositionRiskLimitRate, 10
// 	case stopTradingPositionEPL:
// 		limiter, tokens = r.StopTradingPositionRate, 10
// 	case getAccountWalletBalanceEPL:
// 		limiter, tokens = r.GetAccountWalletBalanceRate, 10
// 	case getAccountFeeEPL:
// 		limiter, tokens = r.GetAccountFeeRate, 10
// 	case getAssetTransferQueryInfoEPL:
// 		limiter, tokens = r.GetAssetTransferQueryInfoRate, 1
// 	case getAssetTransferQueryTransferCoinListEPL:
// 		limiter, tokens = r.GetAssetTransferQueryTransferCoinListRate, 1
// 	case getAssetTransferCoinListEPL:
// 		limiter, tokens = r.GetAssetTransferCoinListRate, 1
// 	case getAssetInterTransferListEPL:
// 		limiter, tokens = r.GetAssetInterTransferListRate, 1
// 	case getSubMemberListEPL:
// 		limiter, tokens = r.GetSubMemberListRate, 1
// 	case getAssetUniversalTransferListEPL:
// 		limiter, tokens = r.GetAssetUniversalTransferListRate, 2
// 	case getAssetAccountCoinBalanceEPL:
// 		limiter, tokens = r.GetAssetAccountCoinBalanceRate, 2
// 	case getAssetDepositRecordsEPL:
// 		limiter, tokens = r.GetAssetDepositRecordsRate, 1
// 	case getAssetDepositSubMemberRecordsEPL:
// 		limiter, tokens = r.GetAssetDepositSubMemberRecordsRate, 1
// 	case getAssetDepositSubMemberAddressEPL:
// 		limiter, tokens = r.GetAssetDepositSubMemberAddressRate, 1
// 	case getWithdrawRecordsEPL:
// 		limiter, tokens = r.GetWithdrawRecordsRate, 1
// 	case getAssetCoinInfoEPL:
// 		limiter, tokens = r.GetAssetCoinInfoRate, 1
// 	case getExchangeOrderRecordEPL:
// 		limiter, tokens = r.GetExchangeOrderRecordRate, 1
// 	case interTransferEPL:
// 		limiter, tokens = r.InterTransferRate, 1
// 	case saveTransferSubMemberEPL:
// 		limiter, tokens = r.SaveTransferSubMemberRate, 1
// 	case universalTransferEPL:
// 		limiter, tokens = r.UniversalTransferRate, 5
// 	case createWithdrawalEPL:
// 		limiter, tokens = r.CreateWithdrawalRate, 1
// 	case cancelWithdrawalEPL:
// 		limiter, tokens = r.CancelWithdrawalRate, 1

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		defaultEPL:                               request.NewRateLimit(spotInterval, 120, 1),
		createOrderEPL:                           request.NewRateLimit(time.Second, 10, 10),
		createSpotOrderEPL:                       request.NewRateLimit(time.Second, 20, 20),
		amendOrderEPL:                            request.NewRateLimit(time.Second, 10, 10),
		cancelOrderEPL:                           request.NewRateLimit(time.Second, 10, 10),
		cancelSpotEPL:                            request.NewRateLimit(time.Second, 20, 1),
		cancelAllEPL:                             request.NewRateLimit(time.Second, 1, 1),
		cancelAllSpotEPL:                         request.NewRateLimit(time.Second, 20, 1),
		createBatchOrderEPL:                      request.NewRateLimit(time.Second, 10, 1),
		amendBatchOrderEPL:                       request.NewRateLimit(time.Second, 10, 1),
		cancelBatchOrderEPL:                      request.NewRateLimit(time.Second, 10, 1),
		getOrderEPL:                              request.NewRateLimit(time.Second, 10, 1),
		getOrderHistoryEPL:                       request.NewRateLimit(time.Second, 10, 1),
		getPositionListEPL:                       request.NewRateLimit(time.Second, 10, 1),
		getExecutionListEPL:                      request.NewRateLimit(time.Second, 10, 1),
		getPositionClosedPNLEPL:                  request.NewRateLimit(time.Second, 10, 1),
		postPositionSetLeverageEPL:               request.NewRateLimit(time.Second, 10, 1),
		setPositionTPLSModeEPL:                   request.NewRateLimit(time.Second, 10, 1),
		setPositionRiskLimitEPL:                  request.NewRateLimit(time.Second, 10, 1),
		stopTradingPositionEPL:                   request.NewRateLimit(time.Second, 10, 1),
		getAccountWalletBalanceEPL:               request.NewRateLimit(time.Second, 10, 1),
		getAccountFeeEPL:                         request.NewRateLimit(time.Second, 10, 1),
		getAssetTransferQueryInfoEPL:             request.NewRateLimit(time.Minute, 60, 1),
		getAssetTransferQueryTransferCoinListEPL: request.NewRateLimit(time.Minute, 60, 1),
		getAssetTransferCoinListEPL:              request.NewRateLimit(time.Minute, 60, 1),
		getAssetInterTransferListEPL:             request.NewRateLimit(time.Minute, 60, 1),
		getSubMemberListEPL:                      request.NewRateLimit(time.Minute, 60, 5),
		getAssetUniversalTransferListEPL:         request.NewRateLimit(time.Second, 2, 1),
		getAssetAccountCoinBalanceEPL:            request.NewRateLimit(time.Second, 2, 1),
		getAssetDepositRecordsEPL:                request.NewRateLimit(time.Minute, 30, 10),
		getAssetDepositSubMemberRecordsEPL:       request.NewRateLimit(time.Minute, 30, 10),
		getAssetDepositSubMemberAddressEPL:       request.NewRateLimit(time.Minute, 30, 10),
		getWithdrawRecordsEPL:                    request.NewRateLimit(time.Minute, 30, 10),
		getAssetCoinInfoEPL:                      request.NewRateLimit(time.Minute, 30, 10),
		getExchangeOrderRecordEPL:                request.NewRateLimit(time.Minute, 30, 10),
		interTransferEPL:                         request.NewRateLimit(time.Minute, 20, 1),
		saveTransferSubMemberEPL:                 request.NewRateLimit(time.Minute, 20, 1),
		universalTransferEPL:                     request.NewRateLimit(time.Second, 5, 1),
		createWithdrawalEPL:                      request.NewRateLimit(time.Second, 1, 1),
		cancelWithdrawalEPL:                      request.NewRateLimit(time.Minute, 60, 1),
		userCreateSubMemberEPL:                   request.NewRateLimit(time.Second, 5, 5),
		userCreateSubAPIKeyEPL:                   request.NewRateLimit(time.Second, 5, 5),
		userFrozenSubMemberEPL:                   request.NewRateLimit(time.Second, 5, 5),
		userUpdateAPIEPL:                         request.NewRateLimit(time.Second, 5, 5),
		userUpdateSubAPIEPL:                      request.NewRateLimit(time.Second, 5, 5),
		userDeleteAPIEPL:                         request.NewRateLimit(time.Second, 5, 5),
		userDeleteSubAPIEPL:                      request.NewRateLimit(time.Second, 5, 5),
		userQuerySubMembersEPL:                   request.NewRateLimit(time.Second, 10, 10),
		userQueryAPIEPL:                          request.NewRateLimit(time.Second, 10, 10),
		getSpotLeverageTokenOrderRecordsEPL:      request.NewRateLimit(time.Second, 50, 50),
		spotLeverageTokenPurchaseEPL:             request.NewRateLimit(time.Second, 20, 20),
		spotLeverTokenRedeemEPL:                  request.NewRateLimit(time.Second, 20, 20),
		getSpotCrossMarginTradeLoanInfoEPL:       request.NewRateLimit(time.Second, 50, 50),
		getSpotCrossMarginTradeAccountEPL:        request.NewRateLimit(time.Second, 50, 50),
		getSpotCrossMarginTradeOrdersEPL:         request.NewRateLimit(time.Second, 50, 50),
		getSpotCrossMarginTradeRepayHistoryEPL:   request.NewRateLimit(time.Second, 50, 50),
		spotCrossMarginTradeLoanEPL:              request.NewRateLimit(time.Second, 20, 50),
		spotCrossMarginTradeRepayEPL:             request.NewRateLimit(time.Second, 20, 50),
		spotCrossMarginTradeSwitchEPL:            request.NewRateLimit(time.Second, 20, 50),
	}
}
