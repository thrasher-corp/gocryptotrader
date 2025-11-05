package poloniex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	oneSecond = time.Second
)

const (
	unauthEPL request.EndpointLimit = iota
	referenceDataEPL
	sCreateOrderEPL
	sBatchOrderEPL
	sCancelReplaceOrderEPL
	sGetOpenOrdersEPL
	sGetOpenOrderDetailEPL
	sCancelOrderByIDEPL
	sCancelBatchOrdersEPL
	sCancelAllOrdersEPL
	sKillSwitchEPL
	sGetKillSwitchStatusEPL
	sAccountInfoEPL
	sAccountBalancesEPL
	sAccountActivitiEPL
	sAccountsTransferEPL
	sAccountsTransferRecordsEPL
	sFeeInfoEPL
	sInterestHistoryEPL
	sGetSubAccountEPL
	sGetSubAccountBalancesEPL
	sGetSubAccountTransfersEPL
	sGetDepositAddressesEPL
	sGetWalletActivityRecordsEPL
	sGetWalletAddressesEPL
	sWithdrawCurrencyEPL
	sAccountMarginEPL
	sBorrowStatusEPL
	sMaxMarginSizeEPL
	sCreateSmartOrdersEPL
	sCreateReplaceSmartOrdersEPL
	sGetSmartOrdersEPL
	sSmartOrderDetailEPL
	sCancelSmartOrderByIDEPL
	sCancelSmartOrdersByIDEPL
	sCancelAllSmartOrdersEPL
	sGetOrderHistoryEPL
	sGetSmartOrderHistoryEPL
	sGetTradesEPL
	sGetTradeDetailEPL
	fOrderEPL
	fBatchOrdersEPL
	fCancelOrderEPL
	fCancelBatchOrdersEPL
	fCancelAllLimitOrdersEPL
	fCancelPositionAtMarketPriceEPL
	fCancelAllPositionsAtMarketPriceEPL
	fGetFillsV2EPL
	fGetOrdersEPL
	fGetOrderHistoryEPL
	fGetPositionOpenEPL
	fGetPositionHistoryEPL
	fGetPositionModeEPL
	fSwitchPositionModeEPL
	fAdjustMarginEPL
	fGetPositionLeverageEPL
	fSetPositionLeverageEPL
	fGetAccountBalanceEPL
	fGetBillsDetailsEPL
	fMarketEPL
	fCandlestickEPL
)

// rateLimits returns the rate limit for the exchange
// As per https://docs.poloniex.com/#http-api
var rateLimits = request.RateLimitDefinitions{
	referenceDataEPL: request.NewRateLimitWithWeight(oneSecond, 30, 1),
	unauthEPL:        request.NewRateLimitWithWeight(oneSecond, 200, 1),

	fOrderEPL:                           request.NewRateLimitWithWeight(oneSecond, 50, 1),
	fBatchOrdersEPL:                     request.NewRateLimitWithWeight(oneSecond, 5, 1),
	fCancelOrderEPL:                     request.NewRateLimitWithWeight(oneSecond, 100, 1),
	fCancelBatchOrdersEPL:               request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fCancelAllLimitOrdersEPL:            request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fCancelPositionAtMarketPriceEPL:     request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fCancelAllPositionsAtMarketPriceEPL: request.NewRateLimitWithWeight(oneSecond, 2, 1),
	fGetOrdersEPL:                       request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetFillsV2EPL:                      request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetOrderHistoryEPL:                 request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetPositionOpenEPL:                 request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetPositionHistoryEPL:              request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetPositionModeEPL:                 request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fSwitchPositionModeEPL:              request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fAdjustMarginEPL:                    request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetPositionLeverageEPL:             request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fSetPositionLeverageEPL:             request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fGetAccountBalanceEPL:               request.NewRateLimitWithWeight(oneSecond, 50, 1),
	fGetBillsDetailsEPL:                 request.NewRateLimitWithWeight(oneSecond, 10, 1),
	fMarketEPL:                          request.NewRateLimitWithWeight(oneSecond, 300, 1),
	fCandlestickEPL:                     request.NewRateLimitWithWeight(oneSecond, 20, 1),

	sCreateOrderEPL:              request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sBatchOrderEPL:               request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sCancelReplaceOrderEPL:       request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetOpenOrdersEPL:            request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetOpenOrderDetailEPL:       request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCancelOrderByIDEPL:          request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCancelBatchOrdersEPL:        request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCancelAllOrdersEPL:          request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sKillSwitchEPL:               request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetKillSwitchStatusEPL:      request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sAccountInfoEPL:              request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sAccountBalancesEPL:          request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sAccountActivitiEPL:          request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sAccountsTransferEPL:         request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sAccountsTransferRecordsEPL:  request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sFeeInfoEPL:                  request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sInterestHistoryEPL:          request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetSubAccountEPL:            request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetSubAccountBalancesEPL:    request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetSubAccountTransfersEPL:   request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetDepositAddressesEPL:      request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetWalletActivityRecordsEPL: request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetWalletAddressesEPL:       request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sWithdrawCurrencyEPL:         request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sAccountMarginEPL:            request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sBorrowStatusEPL:             request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sMaxMarginSizeEPL:            request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCreateSmartOrdersEPL:        request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCreateReplaceSmartOrdersEPL: request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sGetSmartOrdersEPL:           request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sSmartOrderDetailEPL:         request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCancelSmartOrderByIDEPL:     request.NewRateLimitWithWeight(oneSecond, 50, 1),
	sCancelSmartOrdersByIDEPL:    request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sCancelAllSmartOrdersEPL:     request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetOrderHistoryEPL:          request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetSmartOrderHistoryEPL:     request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetTradesEPL:                request.NewRateLimitWithWeight(oneSecond, 10, 1),
	sGetTradeDetailEPL:           request.NewRateLimitWithWeight(oneSecond, 50, 1),
}
