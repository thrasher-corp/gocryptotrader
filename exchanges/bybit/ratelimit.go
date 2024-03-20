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

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		defaultEPL:                               request.NewRateLimitWithToken(time.Second*5 /*See: https://bybit-exchange.github.io/docs/v5/rate-limit*/, 120, 1),
		createOrderEPL:                           request.NewRateLimitWithToken(time.Second, 10, 10),
		createSpotOrderEPL:                       request.NewRateLimitWithToken(time.Second, 20, 20),
		amendOrderEPL:                            request.NewRateLimitWithToken(time.Second, 10, 10),
		cancelOrderEPL:                           request.NewRateLimitWithToken(time.Second, 10, 10),
		cancelSpotEPL:                            request.NewRateLimitWithToken(time.Second, 20, 20),
		cancelAllEPL:                             request.NewRateLimitWithToken(time.Second, 1, 1),
		cancelAllSpotEPL:                         request.NewRateLimitWithToken(time.Second, 20, 20),
		createBatchOrderEPL:                      request.NewRateLimitWithToken(time.Second, 10, 10),
		amendBatchOrderEPL:                       request.NewRateLimitWithToken(time.Second, 10, 10),
		cancelBatchOrderEPL:                      request.NewRateLimitWithToken(time.Second, 10, 10),
		getOrderEPL:                              request.NewRateLimitWithToken(time.Second, 10, 10),
		getOrderHistoryEPL:                       request.NewRateLimitWithToken(time.Second, 10, 10),
		getPositionListEPL:                       request.NewRateLimitWithToken(time.Second, 10, 10),
		getExecutionListEPL:                      request.NewRateLimitWithToken(time.Second, 10, 10),
		getPositionClosedPNLEPL:                  request.NewRateLimitWithToken(time.Second, 10, 10),
		postPositionSetLeverageEPL:               request.NewRateLimitWithToken(time.Second, 10, 10),
		setPositionTPLSModeEPL:                   request.NewRateLimitWithToken(time.Second, 10, 10),
		setPositionRiskLimitEPL:                  request.NewRateLimitWithToken(time.Second, 10, 10),
		stopTradingPositionEPL:                   request.NewRateLimitWithToken(time.Second, 10, 10),
		getAccountWalletBalanceEPL:               request.NewRateLimitWithToken(time.Second, 10, 10),
		getAccountFeeEPL:                         request.NewRateLimitWithToken(time.Second, 10, 10),
		getAssetTransferQueryInfoEPL:             request.NewRateLimitWithToken(time.Minute, 60, 1),
		getAssetTransferQueryTransferCoinListEPL: request.NewRateLimitWithToken(time.Minute, 60, 1),
		getAssetTransferCoinListEPL:              request.NewRateLimitWithToken(time.Minute, 60, 1),
		getAssetInterTransferListEPL:             request.NewRateLimitWithToken(time.Minute, 60, 1),
		getSubMemberListEPL:                      request.NewRateLimitWithToken(time.Minute, 60, 1),
		getAssetUniversalTransferListEPL:         request.NewRateLimitWithToken(time.Second, 2, 2),
		getAssetAccountCoinBalanceEPL:            request.NewRateLimitWithToken(time.Second, 2, 2),
		getAssetDepositRecordsEPL:                request.NewRateLimitWithToken(time.Minute, 30, 1),
		getAssetDepositSubMemberRecordsEPL:       request.NewRateLimitWithToken(time.Minute, 30, 1),
		getAssetDepositSubMemberAddressEPL:       request.NewRateLimitWithToken(time.Minute, 30, 1),
		getWithdrawRecordsEPL:                    request.NewRateLimitWithToken(time.Minute, 30, 1),
		getAssetCoinInfoEPL:                      request.NewRateLimitWithToken(time.Minute, 30, 1),
		getExchangeOrderRecordEPL:                request.NewRateLimitWithToken(time.Minute, 30, 1),
		interTransferEPL:                         request.NewRateLimitWithToken(time.Minute, 20, 1),
		saveTransferSubMemberEPL:                 request.NewRateLimitWithToken(time.Minute, 20, 1),
		universalTransferEPL:                     request.NewRateLimitWithToken(time.Second, 5, 5),
		createWithdrawalEPL:                      request.NewRateLimitWithToken(time.Second, 1, 1),
		cancelWithdrawalEPL:                      request.NewRateLimitWithToken(time.Minute, 60, 1),
		userCreateSubMemberEPL:                   request.NewRateLimitWithToken(time.Second, 5, 5),
		userCreateSubAPIKeyEPL:                   request.NewRateLimitWithToken(time.Second, 5, 5),
		userFrozenSubMemberEPL:                   request.NewRateLimitWithToken(time.Second, 5, 5),
		userUpdateAPIEPL:                         request.NewRateLimitWithToken(time.Second, 5, 5),
		userUpdateSubAPIEPL:                      request.NewRateLimitWithToken(time.Second, 5, 5),
		userDeleteAPIEPL:                         request.NewRateLimitWithToken(time.Second, 5, 5),
		userDeleteSubAPIEPL:                      request.NewRateLimitWithToken(time.Second, 5, 5),
		userQuerySubMembersEPL:                   request.NewRateLimitWithToken(time.Second, 10, 10),
		userQueryAPIEPL:                          request.NewRateLimitWithToken(time.Second, 10, 10),
		getSpotLeverageTokenOrderRecordsEPL:      request.NewRateLimitWithToken(time.Second, 50, 50),
		spotLeverageTokenPurchaseEPL:             request.NewRateLimitWithToken(time.Second, 20, 20),
		spotLeverTokenRedeemEPL:                  request.NewRateLimitWithToken(time.Second, 20, 20),
		getSpotCrossMarginTradeLoanInfoEPL:       request.NewRateLimitWithToken(time.Second, 50, 50),
		getSpotCrossMarginTradeAccountEPL:        request.NewRateLimitWithToken(time.Second, 50, 50),
		getSpotCrossMarginTradeOrdersEPL:         request.NewRateLimitWithToken(time.Second, 50, 50),
		getSpotCrossMarginTradeRepayHistoryEPL:   request.NewRateLimitWithToken(time.Second, 50, 50),
		spotCrossMarginTradeLoanEPL:              request.NewRateLimitWithToken(time.Second, 20, 50),
		spotCrossMarginTradeRepayEPL:             request.NewRateLimitWithToken(time.Second, 20, 50),
		spotCrossMarginTradeSwitchEPL:            request.NewRateLimitWithToken(time.Second, 20, 50),
	}
}
