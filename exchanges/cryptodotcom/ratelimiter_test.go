package cryptodotcom

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"Auth":                         publicAuthRate,
		"Instruments":                  publicInstrumentsRate,
		"Orderbook":                    publicOrderbookRate,
		"Candlestick":                  publicCandlestickRate,
		"Ticker":                       publicTickerRate,
		"Valuation":                    publicValuationRate,
		"Trades":                       publicTradesRate,
		"Get Valuations":               publicGetValuationsRate,
		"Get Expired Settlement Price": publicGetExpiredSettlementPriceRate,
		"Get Insurance":                publicGetInsuranceRate,
		"User Balance":                 privateUserBalanceRate,
		"User Balance History":         privateUserBalanceHistoryRate,
		"Create Sub Account Transfer":  privateCreateSubAccountTransferRate,
		"Get Sub Account Balances":     privateGetSubAccountBalancesRate,
		"Get Positions":                privateGetPositionsRate,
		"Create Order":                 privateCreateOrderRate,
		"Cancel Order":                 privateCancelOrderRate,
		"Create Order List":            privateCreateOrderListRate,
		"Cancel Order List":            privateCancelOrderListRate,
		"Get Order List":               privateGetOrderListRate,
		"Cancel All Orders":            privateCancelAllOrdersRate,
		"Close Position":               privateClosePositionRate,
		"Get Order History":            privateGetOrderHistoryRate,
		"Get Open Orders":              privateGetOpenOrdersRate,
		"Get Order Detail":             privateGetOrderDetailRate,
		"Get Trades":                   privateGetTradesRate,
		"Change Account Leverage":      privateChangeAccountLeverageRate,
		"Get Transactions":             privateGetTransactionsRate,
		"Post Withdrawal":              postWithdrawalRate,
		"Get Currency Networks":        privateGetCurrencyNetworksRate,
		"get Deposit Address":          privategetDepositAddressRate,
		"Get Accounts":                 privateGetAccountsRate,
		"Create Sub Account":           privateCreateSubAccountRate,
		"Get OTC User":                 privateGetOTCUserRate,
		"Get OTC Instruments":          privateGetOTCInstrumentsRate,
		"OTC Request Quote":            privateOTCRequestQuoteRate,
		"OTC Accept Quote":             privateOTCAcceptQuoteRate,
		"Get OTC Quote History":        privateGetOTCQuoteHistoryRate,
		"Get OTC Trade History":        privateGetOTCTradeHistoryRate,
		"Get Withdrawal History":       privateGetWithdrawalHistoryRate,
		"Get Deposit History":          privateGetDepositHistoryRate,
		"Get Account Summary":          privateGetAccountSummaryRate,
		"Create Export Request":        createExportRequestRate,
		"Get Export Request":           getExportRequestRate,
		"Create OTC Order":             privateCreateOTCOrderRate,
	}

	rl, err := request.New("RateLimitTest", http.DefaultClient, request.WithLimiter(GetRateLimit()))
	require.NoError(t, err)

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := rl.InitiateRateLimit(context.Background(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
