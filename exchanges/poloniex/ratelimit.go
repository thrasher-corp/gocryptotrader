package poloniex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	publicEPL request.EndpointLimit = iota
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
	sAccountActivityEPL
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
	referenceDataEPL: request.NewRateLimitWithWeight(time.Second, 30, 1),
	publicEPL:        request.NewRateLimitWithWeight(time.Second, 200, 1),

	fOrderEPL:                           request.NewRateLimitWithWeight(time.Second, 50, 1),
	fBatchOrdersEPL:                     request.NewRateLimitWithWeight(time.Second, 5, 1),
	fCancelOrderEPL:                     request.NewRateLimitWithWeight(time.Second, 100, 1),
	fCancelBatchOrdersEPL:               request.NewRateLimitWithWeight(time.Second, 10, 1),
	fCancelAllLimitOrdersEPL:            request.NewRateLimitWithWeight(time.Second, 10, 1),
	fCancelPositionAtMarketPriceEPL:     request.NewRateLimitWithWeight(time.Second, 10, 1),
	fCancelAllPositionsAtMarketPriceEPL: request.NewRateLimitWithWeight(time.Second, 2, 1),
	fGetOrdersEPL:                       request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetFillsV2EPL:                      request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetOrderHistoryEPL:                 request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetPositionOpenEPL:                 request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetPositionHistoryEPL:              request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetPositionModeEPL:                 request.NewRateLimitWithWeight(time.Second, 10, 1),
	fSwitchPositionModeEPL:              request.NewRateLimitWithWeight(time.Second, 10, 1),
	fAdjustMarginEPL:                    request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetPositionLeverageEPL:             request.NewRateLimitWithWeight(time.Second, 10, 1),
	fSetPositionLeverageEPL:             request.NewRateLimitWithWeight(time.Second, 10, 1),
	fGetAccountBalanceEPL:               request.NewRateLimitWithWeight(time.Second, 50, 1),
	fGetBillsDetailsEPL:                 request.NewRateLimitWithWeight(time.Second, 10, 1),
	fMarketEPL:                          request.NewRateLimitWithWeight(time.Second, 300, 1),
	fCandlestickEPL:                     request.NewRateLimitWithWeight(time.Second, 20, 1),

	sCreateOrderEPL:              request.NewRateLimitWithWeight(time.Second, 50, 1),
	sBatchOrderEPL:               request.NewRateLimitWithWeight(time.Second, 10, 1),
	sCancelReplaceOrderEPL:       request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetOpenOrdersEPL:            request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetOpenOrderDetailEPL:       request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCancelOrderByIDEPL:          request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCancelBatchOrdersEPL:        request.NewRateLimitWithWeight(time.Second, 10, 1),
	sCancelAllOrdersEPL:          request.NewRateLimitWithWeight(time.Second, 10, 1),
	sKillSwitchEPL:               request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetKillSwitchStatusEPL:      request.NewRateLimitWithWeight(time.Second, 50, 1),
	sAccountInfoEPL:              request.NewRateLimitWithWeight(time.Second, 50, 1),
	sAccountBalancesEPL:          request.NewRateLimitWithWeight(time.Second, 50, 1),
	sAccountActivityEPL:          request.NewRateLimitWithWeight(time.Second, 10, 1),
	sAccountsTransferEPL:         request.NewRateLimitWithWeight(time.Second, 50, 1),
	sAccountsTransferRecordsEPL:  request.NewRateLimitWithWeight(time.Second, 10, 1),
	sFeeInfoEPL:                  request.NewRateLimitWithWeight(time.Second, 50, 1),
	sInterestHistoryEPL:          request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetSubAccountEPL:            request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetSubAccountBalancesEPL:    request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetSubAccountTransfersEPL:   request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetDepositAddressesEPL:      request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetWalletActivityRecordsEPL: request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetWalletAddressesEPL:       request.NewRateLimitWithWeight(time.Second, 50, 1),
	sWithdrawCurrencyEPL:         request.NewRateLimitWithWeight(time.Second, 10, 1),
	sAccountMarginEPL:            request.NewRateLimitWithWeight(time.Second, 50, 1),
	sBorrowStatusEPL:             request.NewRateLimitWithWeight(time.Second, 50, 1),
	sMaxMarginSizeEPL:            request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCreateSmartOrdersEPL:        request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCreateReplaceSmartOrdersEPL: request.NewRateLimitWithWeight(time.Second, 50, 1),
	sGetSmartOrdersEPL:           request.NewRateLimitWithWeight(time.Second, 10, 1),
	sSmartOrderDetailEPL:         request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCancelSmartOrderByIDEPL:     request.NewRateLimitWithWeight(time.Second, 50, 1),
	sCancelSmartOrdersByIDEPL:    request.NewRateLimitWithWeight(time.Second, 10, 1),
	sCancelAllSmartOrdersEPL:     request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetOrderHistoryEPL:          request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetSmartOrderHistoryEPL:     request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetTradesEPL:                request.NewRateLimitWithWeight(time.Second, 10, 1),
	sGetTradeDetailEPL:           request.NewRateLimitWithWeight(time.Second, 50, 1),
}
