package bybit

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errUnknownCategory = errors.New("unknown category")

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

	wsOrderSpotEPL
	wsOrderInverseEPL
	wsOrderLinearEPL
	wsOrderOptionsEPL
	wsSubscriptionEPL
)

var rateLimits = request.RateLimitDefinitions{
	defaultEPL:                               request.NewRateLimitWithWeight(time.Second*5, 600, 1), // See: https://bybit-exchange.github.io/docs/v5/rate-limit
	createOrderEPL:                           request.NewRateLimitWithWeight(time.Second, 10, 10),
	createSpotOrderEPL:                       request.NewRateLimitWithWeight(time.Second, 20, 20),
	amendOrderEPL:                            request.NewRateLimitWithWeight(time.Second, 10, 10),
	cancelOrderEPL:                           request.NewRateLimitWithWeight(time.Second, 10, 10),
	cancelSpotEPL:                            request.NewRateLimitWithWeight(time.Second, 20, 20),
	cancelAllEPL:                             request.NewWeightedRateLimitByDuration(time.Second),
	cancelAllSpotEPL:                         request.NewRateLimitWithWeight(time.Second, 20, 20),
	createBatchOrderEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10),
	amendBatchOrderEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10),
	cancelBatchOrderEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10),
	getOrderEPL:                              request.NewRateLimitWithWeight(time.Second, 10, 10),
	getOrderHistoryEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10),
	getPositionListEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 10),
	getExecutionListEPL:                      request.NewRateLimitWithWeight(time.Second, 10, 10),
	getPositionClosedPNLEPL:                  request.NewRateLimitWithWeight(time.Second, 10, 10),
	postPositionSetLeverageEPL:               request.NewRateLimitWithWeight(time.Second, 10, 10),
	setPositionTPLSModeEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10),
	setPositionRiskLimitEPL:                  request.NewRateLimitWithWeight(time.Second, 10, 10),
	stopTradingPositionEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10),
	getAccountWalletBalanceEPL:               request.NewRateLimitWithWeight(time.Second, 10, 10),
	getAccountFeeEPL:                         request.NewRateLimitWithWeight(time.Second, 10, 10),
	getAssetTransferQueryInfoEPL:             request.NewRateLimitWithWeight(time.Minute, 60, 1),
	getAssetTransferQueryTransferCoinListEPL: request.NewRateLimitWithWeight(time.Minute, 60, 1),
	getAssetTransferCoinListEPL:              request.NewRateLimitWithWeight(time.Minute, 60, 1),
	getAssetInterTransferListEPL:             request.NewRateLimitWithWeight(time.Minute, 60, 1),
	getSubMemberListEPL:                      request.NewRateLimitWithWeight(time.Minute, 60, 1),
	getAssetUniversalTransferListEPL:         request.NewRateLimitWithWeight(time.Second, 2, 2),
	getAssetAccountCoinBalanceEPL:            request.NewRateLimitWithWeight(time.Second, 2, 2),
	getAssetDepositRecordsEPL:                request.NewRateLimitWithWeight(time.Minute, 30, 1),
	getAssetDepositSubMemberRecordsEPL:       request.NewRateLimitWithWeight(time.Minute, 30, 1),
	getAssetDepositSubMemberAddressEPL:       request.NewRateLimitWithWeight(time.Minute, 30, 1),
	getWithdrawRecordsEPL:                    request.NewRateLimitWithWeight(time.Minute, 30, 1),
	getAssetCoinInfoEPL:                      request.NewRateLimitWithWeight(time.Minute, 30, 1),
	getExchangeOrderRecordEPL:                request.NewRateLimitWithWeight(time.Minute, 30, 1),
	interTransferEPL:                         request.NewRateLimitWithWeight(time.Minute, 20, 1),
	saveTransferSubMemberEPL:                 request.NewRateLimitWithWeight(time.Minute, 20, 1),
	universalTransferEPL:                     request.NewRateLimitWithWeight(time.Second, 5, 5),
	createWithdrawalEPL:                      request.NewWeightedRateLimitByDuration(time.Second),
	cancelWithdrawalEPL:                      request.NewRateLimitWithWeight(time.Minute, 60, 1),
	userCreateSubMemberEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5),
	userCreateSubAPIKeyEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5),
	userFrozenSubMemberEPL:                   request.NewRateLimitWithWeight(time.Second, 5, 5),
	userUpdateAPIEPL:                         request.NewRateLimitWithWeight(time.Second, 5, 5),
	userUpdateSubAPIEPL:                      request.NewRateLimitWithWeight(time.Second, 5, 5),
	userDeleteAPIEPL:                         request.NewRateLimitWithWeight(time.Second, 5, 5),
	userDeleteSubAPIEPL:                      request.NewRateLimitWithWeight(time.Second, 5, 5),
	userQuerySubMembersEPL:                   request.NewRateLimitWithWeight(time.Second, 10, 10),
	userQueryAPIEPL:                          request.NewRateLimitWithWeight(time.Second, 10, 10),
	getSpotLeverageTokenOrderRecordsEPL:      request.NewRateLimitWithWeight(time.Second, 50, 50),
	spotLeverageTokenPurchaseEPL:             request.NewRateLimitWithWeight(time.Second, 20, 20),
	spotLeverTokenRedeemEPL:                  request.NewRateLimitWithWeight(time.Second, 20, 20),
	getSpotCrossMarginTradeLoanInfoEPL:       request.NewRateLimitWithWeight(time.Second, 50, 50),
	getSpotCrossMarginTradeAccountEPL:        request.NewRateLimitWithWeight(time.Second, 50, 50),
	getSpotCrossMarginTradeOrdersEPL:         request.NewRateLimitWithWeight(time.Second, 50, 50),
	getSpotCrossMarginTradeRepayHistoryEPL:   request.NewRateLimitWithWeight(time.Second, 50, 50),
	spotCrossMarginTradeLoanEPL:              request.NewRateLimitWithWeight(time.Second, 20, 50),
	spotCrossMarginTradeRepayEPL:             request.NewRateLimitWithWeight(time.Second, 20, 50),
	spotCrossMarginTradeSwitchEPL:            request.NewRateLimitWithWeight(time.Second, 20, 50),

	wsOrderSpotEPL:    request.NewRateLimitWithWeight(time.Second, 20, 1),
	wsOrderInverseEPL: request.NewRateLimitWithWeight(time.Second, 20, 1),
	wsOrderLinearEPL:  request.NewRateLimitWithWeight(time.Second, 20, 1),
	wsOrderOptionsEPL: request.NewRateLimitWithWeight(time.Second, 20, 1),
	wsSubscriptionEPL: request.RateLimitNotRequired,
}

func getWSRateLimitEPLByCategory(category string) (request.EndpointLimit, error) {
	switch category {
	case cSpot:
		return wsOrderSpotEPL, nil
	case cInverse:
		return wsOrderInverseEPL, nil
	case cLinear:
		return wsOrderLinearEPL, nil
	case cOption:
		return wsOrderOptionsEPL, nil
	default:
		return 0, fmt.Errorf("%w: %q", errUnknownCategory, category)
	}
}
