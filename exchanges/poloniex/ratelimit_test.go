package poloniex

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"unauth":                           publicEPL,
		"futuresWebsocketPublicEPL":        fWebsocketPublicEPL,
		"futuresWebsocketPrivateEPL":       fWebsocketPrivateEPL,
		"spotWebsocketPublicEPL":           sWebsocketPublicEPL,
		"spotWebsocketPrivateEPL":          sWebsocketPrivateEPL,
		"referenceData":                    referenceDataEPL,
		"sCreateOrder":                     sCreateOrderEPL,
		"sBatchOrder":                      sBatchOrderEPL,
		"sCancelReplaceOrder":              sCancelReplaceOrderEPL,
		"sGetOpenOrders":                   sGetOpenOrdersEPL,
		"sGetOpenOrderDetail":              sGetOpenOrderDetailEPL,
		"sCancelOrderByID":                 sCancelOrderByIDEPL,
		"sCancelBatchOrders":               sCancelBatchOrdersEPL,
		"sCancelAllOrders":                 sCancelAllOrdersEPL,
		"sKillSwitch":                      sKillSwitchEPL,
		"sGetKillSwitchStatus":             sGetKillSwitchStatusEPL,
		"sAccountInfo":                     sAccountInfoEPL,
		"sAccountBalances":                 sAccountBalancesEPL,
		"sAccountActiviti":                 sAccountActivityEPL,
		"sAccountsTransfer":                sAccountsTransferEPL,
		"sAccountsTransferRecords":         sAccountsTransferRecordsEPL,
		"sFeeInfo":                         sFeeInfoEPL,
		"sInterestHistory":                 sInterestHistoryEPL,
		"sGetSubAccount":                   sGetSubAccountEPL,
		"sGetSubAccountBalances":           sGetSubAccountBalancesEPL,
		"sGetSubAccountTransfers":          sGetSubAccountTransfersEPL,
		"sGetDepositAddresses":             sGetDepositAddressesEPL,
		"sGetWalletActivityRecords":        sGetWalletActivityRecordsEPL,
		"sGetWalletAddresses":              sGetWalletAddressesEPL,
		"sWithdrawCurrency":                sWithdrawCurrencyEPL,
		"sAccountMargin":                   sAccountMarginEPL,
		"sBorrowStatus":                    sBorrowStatusEPL,
		"sMaxMarginSize":                   sMaxMarginSizeEPL,
		"sCreateSmartOrders":               sCreateSmartOrdersEPL,
		"sCreateReplaceSmartOrders":        sCreateReplaceSmartOrdersEPL,
		"sGetSmartOrders":                  sGetSmartOrdersEPL,
		"sSmartOrderDetail":                sSmartOrderDetailEPL,
		"sCancelSmartOrderByID":            sCancelSmartOrderByIDEPL,
		"sCancelSmartOrdersByID":           sCancelSmartOrdersByIDEPL,
		"sCancelAllSmartOrders":            sCancelAllSmartOrdersEPL,
		"sGetOrderHistory":                 sGetOrderHistoryEPL,
		"sGetSmartOrderHistory":            sGetSmartOrderHistoryEPL,
		"sGetTrades":                       sGetTradesEPL,
		"sGetTradeDetail":                  sGetTradeDetailEPL,
		"fOrder":                           fOrderEPL,
		"fBatchOrders":                     fBatchOrdersEPL,
		"fCancelOrder":                     fCancelOrderEPL,
		"fCancelBatchOrders":               fCancelBatchOrdersEPL,
		"fCancelAllLimitOrders":            fCancelAllLimitOrdersEPL,
		"fCancelPositionAtMarketPrice":     fCancelPositionAtMarketPriceEPL,
		"fCancelAllPositionsAtMarketPrice": fCancelAllPositionsAtMarketPriceEPL,
		"fGetFillsV2":                      fGetFillsV2EPL,
		"fGetOrders":                       fGetOrdersEPL,
		"fGetOrderHistory":                 fGetOrderHistoryEPL,
		"fGetPositionOpen":                 fGetPositionOpenEPL,
		"fGetPositionHistory":              fGetPositionHistoryEPL,
		"fGetPositionMode":                 fGetPositionModeEPL,
		"fGetPositionRiskLimit":            fGetPositionRiskLimitEPL,
		"fSwitchPositionMode":              fSwitchPositionModeEPL,
		"fAdjustMargin":                    fAdjustMarginEPL,
		"fGetPositionLeverage":             fGetPositionLeverageEPL,
		"fSetPositionLeverage":             fSetPositionLeverageEPL,
		"fGetAccountBalance":               fGetAccountBalanceEPL,
		"fGetBillsDetails":                 fGetBillsDetailsEPL,
		"fMarket":                          fMarketEPL,
		"fCandlestick":                     fCandlestickEPL,
	}
	rl, err := request.New("rateLimitTest2", http.DefaultClient, request.WithLimiter(rateLimits))
	require.NoError(t, err, "request.New must not error")

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, rl.InitiateRateLimit(t.Context(), tt), "InitiateRateLimit must not error")
		})
	}
}
