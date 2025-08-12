package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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

var globalRateLimit = request.NewRateLimit(time.Second*5, 600)

var rateLimits = request.RateLimitDefinitions{
	defaultEPL:                               request.GetRateLimiterWithWeight(globalRateLimit, 1),
	createOrderEPL:                           request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	createSpotOrderEPL:                       request.NewRateLimitWithWeight(time.Second, 20, 20, globalRateLimit),
	amendOrderEPL:                            request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	cancelOrderEPL:                           request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	cancelSpotEPL:                            request.NewRateLimitWithWeight(time.Second, 20, 20, globalRateLimit),
	cancelAllEPL:                             request.NewWeightedRateLimitByDuration(time.Second, globalRateLimit),
	cancelAllSpotEPL:                         request.NewRateLimitWithWeight(time.Second, 20, 20, globalRateLimit),
	createBatchOrderEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	amendBatchOrderEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	cancelBatchOrderEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getOrderEPL:                              request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getOrderHistoryEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getPositionListEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getExecutionListEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getPositionClosedPNLEPL:                  request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	postPositionSetLeverageEPL:               request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	setPositionTPLSModeEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	setPositionRiskLimitEPL:                  request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	stopTradingPositionEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getAccountWalletBalanceEPL:               request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getAccountFeeEPL:                         request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getAssetTransferQueryInfoEPL:             request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	getAssetTransferQueryTransferCoinListEPL: request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	getAssetTransferCoinListEPL:              request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	getAssetInterTransferListEPL:             request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	getSubMemberListEPL:                      request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	getAssetUniversalTransferListEPL:         request.NewRateLimitWithWeight(time.Second, 2, 2, globalRateLimit),
	getAssetAccountCoinBalanceEPL:            request.NewRateLimitWithWeight(time.Second, 2, 2, globalRateLimit),
	getAssetDepositRecordsEPL:                request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	getAssetDepositSubMemberRecordsEPL:       request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	getAssetDepositSubMemberAddressEPL:       request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	getWithdrawRecordsEPL:                    request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	getAssetCoinInfoEPL:                      request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	getExchangeOrderRecordEPL:                request.NewRateLimitWithWeight(time.Minute, 30, 1, globalRateLimit),
	interTransferEPL:                         request.NewRateLimitWithWeight(time.Minute, 20, 1, globalRateLimit),
	saveTransferSubMemberEPL:                 request.NewRateLimitWithWeight(time.Minute, 20, 1, globalRateLimit),
	universalTransferEPL:                     request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	createWithdrawalEPL:                      request.NewWeightedRateLimitByDuration(time.Second, globalRateLimit),
	cancelWithdrawalEPL:                      request.NewRateLimitWithWeight(time.Minute, 60, 1, globalRateLimit),
	userCreateSubMemberEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userCreateSubAPIKeyEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userFrozenSubMemberEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userUpdateAPIEPL:                         request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userUpdateSubAPIEPL:                      request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userDeleteAPIEPL:                         request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userDeleteSubAPIEPL:                      request.NewRateLimitWithWeight(time.Second, 5, 5, globalRateLimit),
	userQuerySubMembersEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	userQueryAPIEPL:                          request.NewRateLimitWithWeight(time.Second, 10, 10, globalRateLimit),
	getSpotLeverageTokenOrderRecordsEPL:      request.NewRateLimitWithWeight(time.Second, 50, 50, globalRateLimit),
	spotLeverageTokenPurchaseEPL:             request.NewRateLimitWithWeight(time.Second, 20, 20, globalRateLimit),
	spotLeverTokenRedeemEPL:                  request.NewRateLimitWithWeight(time.Second, 20, 20, globalRateLimit),
	getSpotCrossMarginTradeLoanInfoEPL:       request.NewRateLimitWithWeight(time.Second, 50, 50, globalRateLimit),
	getSpotCrossMarginTradeAccountEPL:        request.NewRateLimitWithWeight(time.Second, 50, 50, globalRateLimit),
	getSpotCrossMarginTradeOrdersEPL:         request.NewRateLimitWithWeight(time.Second, 50, 50, globalRateLimit),
	getSpotCrossMarginTradeRepayHistoryEPL:   request.NewRateLimitWithWeight(time.Second, 50, 50, globalRateLimit),
	spotCrossMarginTradeLoanEPL:              request.NewRateLimitWithWeight(time.Second, 20, 50, globalRateLimit),
	spotCrossMarginTradeRepayEPL:             request.NewRateLimitWithWeight(time.Second, 20, 50, globalRateLimit),
	spotCrossMarginTradeSwitchEPL:            request.NewRateLimitWithWeight(time.Second, 20, 50, globalRateLimit),
}
