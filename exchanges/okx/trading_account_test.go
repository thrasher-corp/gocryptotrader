package okx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OKX ignores unknown query parameters, so a misnamed filter would fail silently
func TestDocsPinnedRequestParameters(t *testing.T) {
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values

	payloads := map[string]string{
		"/trade/orders-algo-pending":                     `{"code":"0","msg":"","data":[{"algoId":"12345","algoClOrdId":"test-algo-client-id","clOrdId":"ord-client-1","cTime":"1724751378980","uTime":"1724751378999","ordIdList":["680800019749904384"],"advanceOrdType":"chase","chaseType":"distance","chaseVal":"10","maxChaseType":"ratio","maxChaseVal":"0.1","lmtOrderNumber":"5","aggressiveness":"conservative","triggerParams":[{"triggerAction":"start","triggerStrategy":"price","triggerPx":"90000","triggerCond":"cross_down"}],"linkedOrd":{"ordId":"680800019749904385"},"last":"62916.5","reduceOnly":"true","attachAlgoOrds":[{"attachAlgoClOrdId":"attach-client-1","tpTriggerPx":"50000","tpOrdPx":"-1","slTriggerPx":"40000"}]}]}`,
		"/tradingBot/recurring/orders-algo-details":      `{"code":"0","msg":"","data":[{"algoId":"560473220642766848","state":"running","amt":"100","period":"hourly","source":["1"],"tradeQuoteCcy":"USDT","recurringList":[{"ccy":"BTC","px":"36683.2","avgPx":"36500.1","profit":"12.5","ratio":"0.5","totalAmt":"100","minPx":"30000","maxPx":"50000"}]}]}`,
		"/fiat/deposit":                                  `{"code":"0","msg":"","data":[{}]}`,
		"/trade/one-click-repay-currency-list":           `{"code":"0","msg":"","data":[{"debtType":"cross","debtData":[{"debtCcy":"BTC","debtAmt":"1.5"}],"repayData":[{"repayCcy":"USDT","repayAmt":"100"}]}]}`,
		"/tradingBot/grid/sub-orders":                    `{"code":"0","msg":"","data":[{"algoId":"12345","algoClOrdId":"grid-client-1","ordId":"grid-ord-1","instId":"BTC-USDT","algoOrdType":"grid","groupId":"grid-group-1","px":"42000","sz":"0.1","avgPx":"41900","accFillSz":"0.1","pnl":"12.5","ccy":"USDT","rebate":"0.01","rebateCcy":"USDT","lever":"5","ctVal":"0.01","posSide":"net","side":"buy","state":"filled","ordType":"market","tdMode":"cross","cTime":"1724751378980","uTime":"1724751378999"}]}`,
		"/market/index-components":                       `{"code":"0","msg":"","data":[{"index":"BTC-USDT","last":"42000","ts":"1724751378980","components":[{"exch":"binance","symbol":"BTC/USDT","symPx":"42000.5","wgt":"0.9","cnvPx":"42000"}]}]}`,
		"/public/discount-rate-interest-free-quota":      `{"code":"0","msg":"","data":[{"ccy":"BTC","colRes":"0","amt":"1","details":[{"discountRate":"0.98","maxAmt":"10","minAmt":"0","tier":"1","liqPenaltyRate":"0.01","disCcyEq":"0.5"}]}]}`,
		"/account/position-tiers":                        `{"code":"0","msg":"","data":[{"maxSz":"1000","posType":"1","uly":"BTC-USD","instFamily":"BTC-USD"}]}`,
		"/asset/convert/currencies":                      `{"code":"0","msg":"","data":[{"ccy":"BTC","min":"0.0001","max":"100"}]}`,
		"/rfq/maker-instrument-settings":                 `{"code":"0","msg":"","data":[{"instType":"OPTION","includeAll":true,"data":[{"instFamily":"BTC-USD","maxBlockSz":"10000","makerPxBand":"5"}]}]}`,
		"/copytrading/unrealized-profit-sharing-details": `{"code":"0","msg":"","data":[{"ccy":"USDT","nickName":"Potato","unrealizedProfitSharingAmt":"0.455472","instType":"SWAP","ts":"1669901824779"}]}`,
		"/account/risk-state":                            `{"code":"0","msg":"","data":[{"atRisk":true,"atRiskIdx":[],"atRiskMgn":[],"ts":"1635745078794"}]}`,
		"/tradingBot/signal/event-history":               `{"code":"0","msg":"","data":[{"algoId":"12345","alertMsg":"price alert","eventCtime":"1724751378980","eventProcessMsg":"done","state":"done","triggerTime":"1724751378999"}]}`,
		"/account/positions":                             `{"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","realizedPnl":"12.5","fundingFee":"-0.1","bePx":"41000","pnl":"5"}]}`,
		"/account/positions-history":                     `{"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","mgnMode":"cross","type":"2","cTime":"1619776200285","uTime":"1619776200285","lever":"3","openAvgPx":"41000","closeAvgPx":"42000.5","triggerPx":"40000","pnl":"5","fee":"-0.5","fundingFee":"-0.1","liqPenalty":"-0.2"}]}`,
		"/public/funding-rate":                           `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USD-SWAP","formulaType":"withRate","fundingRate":"0.0001","realizedRate":"0.00012","interestRate":"0.00003","impactValue":"1.2","method":"current_period"}]}`,
		"/tradingBot/grid/orders-algo-details":           `{"code":"0","msg":"","data":[{"algoId":"12345","algoClOrdId":"grid-client-1","instFamily":"BTC-USDT","activeOrdNum":"3","ordFrozen":"100","availEq":"500","tpRatio":"0.1","slRatio":"0.05","fee":"-0.2","feeCcy":"USDT","fundingFee":"-0.3","triggerParams":[{"triggerAction":"start","triggerStrategy":"rsi","timeframe":"15m","thold":"30","triggerCond":"cross_up","timePeriod":"14"}]}]}`,
		"/copytrading/current-subpositions":              `{"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","margin":"100","ccy":"USDT","uniqueCode":"u1","markPx":"42000","upl":"1.5","uplRatio":"0.015","tpOrdPx":"50000","slOrdPx":"38000","availSubPos":"0.5","pnl":"2","pnlRatio":"0.02"}]}`,
		"/asset/withdrawal-history":                      `{"code":"0","msg":"","data":[{"ccy":"BTC","toAddrType":"2","note":"w1","amt":"0.1","ts":"1724751378980","addrEx":{"comment":"123456"}},{"ccy":"ETH","toAddrType":"1","note":"w2","amt":"2","ts":"1724751378981","addrEx":null}]}`,
		"/users/subaccount/list":                         `{"code":"0","msg":"","data":[{"subAcct":"sub-one","uid":"123456","frozenFunc":[],"subAcctLv":"1","firstLvSubAcct":"sub-one","ifDma":true,"enable":true,"ts":"1724751378980"}]}`,
		"/account/mmp-config":                            `{"code":"0","msg":"","data":[{"instFamily":"BTC-USD","timeInterval":"5000","frozenInterval":"2000","qtyLimit":"100","mmpFrozen":false,"mmpFrozenUntil":""}]}`,
		"/asset/deposit-withdraw-status":                 `{"code":"0","msg":"","data":[{"wdId":"1244","txId":"16f3638329c8a5b1f6acde28b0f0b3c9ethc15a05b53b52b3049a099fea19a92","state":"Pending withdrawal: transaction is being confirmed on-chain.","estCompleteTime":"01/09/2023, 8:10:48 PM"}]}`,
		"/account/position-builder":                      `{"code":"0","msg":"","data":[{"acctLever":"0.5","assets":[{"availEq":"500","borrowImr":"0.1","borrowMmr":"1.2","ccy":"USDT","spotInUse":"5"}],"borrowMmr":"1.2","derivMmr":"0.8","eq":"100","marginRatio":"0.02","riskUnitData":[{"riskUnit":"BTC","portfolios":[]}],"positions":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","amt":"1","posSide":"net","imr":"100","lever":"3","isRealPos":true}],"totalImr":"2.5","totalMmr":"1.5","ts":"1724751378980"}]}`,
	}

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		data := `{"code":"0","msg":"","data":[]}`
		if payload, ok := payloads[r.URL.Path]; ok {
			data = payload
		}
		_, _ = w.Write([]byte(data))
	}))

	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	lastRequest := func() (string, url.Values) {
		mu.Lock()
		defer mu.Unlock()
		return gotPath, gotQuery
	}

	var pendingAlgoOrders []AlgoOrderResponse
	var recurringOrderDetails *RecurringOrderDetailResponse
	var oneClickRepayCurrencyList []OneClickRepayCurrencyListResponse
	var gridSubOrders []GridSubOrderResponse
	var indexComponents *IndexComponentsResponse
	var discountRates []DiscountRateResponse
	var pmPositionLimitations []PMLimitationResponse
	var convertCurrencies []ConvertCurrencyResponse
	var quoteProducts []QuoteProductResponse
	var unrealisedProfitSharing []UnrealisedProfitSharingDetailResponse
	var accountRiskState []AccountRiskStateResponse
	var signalBotEventHistory []SignalBotEventHistoryResponse
	var accountPositions []AccountPositionResponse
	var fundingRates []FundingRateResponse
	var gridAlgoDetails *GridAlgoOrderResponse
	var leadingPositions []LeadPositionResponse
	var withdrawalHistory []WithdrawalHistoryResponse
	var subaccountList []SubAccountResponse
	var mmpConfigs []MMPConfigResponse
	var depositWithdrawStatuses []DepositWithdrawalStatusResponse
	var positionBuilder *PositionBuilderResponse

	for _, tc := range []struct {
		name   string
		call   func() error
		path   string
		params map[string]string
		absent []string
		verify func(t *testing.T)
	}{
		{
			name: "RequestForQuotes send clRfqId",
			call: func() error {
				_, err := e.GetRequestForQuotes(t.Context(), &RequestForQuotesRequest{ClientRequestForQuoteID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/rfqs",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "Quotes send clRfqId",
			call: func() error {
				_, err := e.GetQuotes(t.Context(), &QuotesRequest{ClientRequestForQuoteID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/quotes",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "RFQ trades send clRfqId without state",
			call: func() error {
				_, err := e.GetRequestForQuoteTrades(t.Context(), &RequestForQuoteTradesRequest{ClientRequestForQuoteID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/trades",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId", "state"},
		},
		{
			name: "Spread order books size uses sz",
			call: func() error {
				_, err := e.GetPublicSpreadOrderBooks(t.Context(), &GetPublicSpreadOrderBooksRequest{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Depth: 50})
				return err
			},
			path:   "/sprd/books",
			params: map[string]string{"sz": "50"},
			absent: []string{"size"},
		},
		{
			name: "Subaccount bills filter uses subAcct",
			call: func() error {
				_, err := e.GetSubAccountTransferHistory(t.Context(), &SubAccountTransferHistoryRequest{Currency: currency.BTC, SubAccountName: "sub-one", Limit: 7})
				return err
			},
			path:   "/asset/subaccount/bills",
			params: map[string]string{"subAcct": "sub-one", "limit": "7"},
			absent: []string{"subacct", "setAcct"},
		},
		{
			name: "Custody subaccount list filter uses subAcct",
			call: func() error {
				_, err := e.GetCustodyTradingSubAccountList(t.Context(), &CustodyTradingSubAccountListRequest{SubAccountName: "sub-one"})
				return err
			},
			path:   "/users/entrust-subaccount-list",
			params: map[string]string{"subAcct": "sub-one"},
			absent: []string{"setAcct"},
		},
		{
			name: "Position tiers filter uses singular tier",
			call: func() error {
				_, err := e.GetPositionTiers(t.Context(), &PositionTiersRequest{InstrumentType: instTypeMargin, TradeMode: "cross", InstrumentIDs: []string{"BTC-USDT"}, Tier: 1})
				return err
			},
			path:   "/public/position-tiers",
			params: map[string]string{"tier": "1"},
			absent: []string{"tiers"},
		},
		{
			name: "Daily lead trader ProfitAndLoss hits the daily endpoint",
			call: func() error {
				_, err := e.GetLeadTraderDailyProfitAndLoss(t.Context(), &LeadTraderPerformanceRequest{InstrumentType: "SWAP", UniqueCode: "trader-1", LastDays: "2"})
				return err
			},
			path:   "/copytrading/public-pnl",
			params: map[string]string{"uniqueCode": "trader-1", "lastDays": "2"},
		},
		{
			name: "Lead trader currency preferences drop lastDays",
			call: func() error {
				_, err := e.GetLeadTraderCurrencyPreferences(t.Context(), &LeadTraderRequest{InstrumentType: "SWAP", UniqueCode: "trader-1"})
				return err
			},
			path:   "/copytrading/public-preference-currency",
			params: map[string]string{"uniqueCode": "trader-1"},
			absent: []string{"lastDays"},
		},
		{
			name: "Trade fee drops the response-only ruleType",
			call: func() error {
				_, err := e.GetTradeFee(t.Context(), &TradeFeeRequest{InstrumentType: instTypeSpot, InstrumentID: "BTC-USDT"})
				return err
			},
			path:   "/account/trade-fee",
			params: map[string]string{"instType": "SPOT", "instId": "BTC-USDT"},
			absent: []string{"ruleType"},
		},
		{
			name: "Max buy or sell amount drops unSpotOffset",
			call: func() error {
				_, err := e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), &MaximumOrderQuantityRequest{InstrumentIDs: []string{"BTC-USDT"}, TradeMode: "cash", Currency: currency.BTC, Price: 5})
				return err
			},
			path:   "/account/max-size",
			params: map[string]string{"instId": "BTC-USDT"},
			absent: []string{"unSpotOffset"},
		},
		{
			name: "Max available amount drops quickMgnType and upSpotOffset",
			call: func() error {
				_, err := e.GetMaximumAvailableTradableAmount(t.Context(), &MaximumAvailableBalanceRequest{InstrumentIDs: []string{"BTC-USDT"}, TradeMode: "cash", Currency: currency.BTC, Price: 5})
				return err
			},
			path:   "/account/max-avail-size",
			params: map[string]string{"instId": "BTC-USDT"},
			absent: []string{"quickMgnType", "upSpotOffset"},
		},
		{
			// algoClOrdId is the last name the request table documented for
			// this filter: it was delisted on 2025-04-24, but live the endpoint
			// still narrows on it, and the old clOrdId name no longer filters.
			name: "Pending algo order list sends algoClOrdId",
			call: func() error {
				var err error
				pendingAlgoOrders, err = e.GetAlgoOrderList(t.Context(), &AlgoOrderListRequest{OrderType: "conditional", AlgoClientOrderID: "test-algo-client-id", Limit: 1})
				return err
			},
			path:   "/trade/orders-algo-pending",
			params: map[string]string{"ordType": "conditional", "algoClOrdId": "test-algo-client-id"},
			absent: []string{"clOrdId"},
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, pendingAlgoOrders, 1, "the response item must decode")
				assert.Equal(t, "test-algo-client-id", pendingAlgoOrders[0].AlgoClientOrderID, "the response should echo the algoClOrdId this filter narrows on")
				assert.Equal(t, "ord-client-1", pendingAlgoOrders[0].ClientOrderID, "the documented clOrdId should decode")
				assert.True(t, pendingAlgoOrders[0].UpdateTime.Time().Equal(time.UnixMilli(1724751378999)), "the documented uTime should decode")
				require.Len(t, pendingAlgoOrders[0].AttachedAlgoOrders, 1, "the documented attached algo order objects must decode")
				assert.Equal(t, "attach-client-1", pendingAlgoOrders[0].AttachedAlgoOrders[0].AttachAlgoClientOrderID, "the attached algo client order ID should decode")
				assert.Equal(t, 50000.0, pendingAlgoOrders[0].AttachedAlgoOrders[0].TakeProfitTriggerPrice.Float64(), "the attached take-profit trigger price should decode")
				assert.Equal(t, -1.0, pendingAlgoOrders[0].AttachedAlgoOrders[0].TakeProfitOrderPrice.Float64(), "the attached take-profit market-price sentinel should decode")
				assert.Equal(t, 40000.0, pendingAlgoOrders[0].AttachedAlgoOrders[0].StopLossTriggerPrice.Float64(), "the attached stop-loss trigger price should decode")
				assert.Equal(t, "680800019749904384", pendingAlgoOrders[0].OrderIDList[0], "the split TP/SL order IDs should decode")
				assert.Equal(t, "chase", pendingAlgoOrders[0].AdvanceOrderType, "the documented advanceOrdType should decode")
				assert.Equal(t, "distance", pendingAlgoOrders[0].ChaseType, "the documented chaseType should decode")
				assert.Equal(t, 10.0, pendingAlgoOrders[0].ChaseValue.Float64(), "the documented chaseVal should decode")
				assert.Equal(t, 0.1, pendingAlgoOrders[0].MaxChaseValue.Float64(), "the documented maxChaseVal should decode")
				assert.Equal(t, 5.0, pendingAlgoOrders[0].LimitOrderNumber.Float64(), "the smart-iceberg split count should decode")
				assert.Equal(t, "conservative", pendingAlgoOrders[0].Aggressiveness, "the documented aggressiveness should decode")
				require.Len(t, pendingAlgoOrders[0].TriggerParameters, 1, "the smart-iceberg trigger parameters must decode")
				assert.Equal(t, 90000.0, pendingAlgoOrders[0].TriggerParameters[0].TriggerPrice.Float64(), "the trigger price should decode")
				assert.Equal(t, "680800019749904385", pendingAlgoOrders[0].LinkedOrder.OrderID, "the linked OCO take-profit order ID should decode")
				assert.Equal(t, 62916.5, pendingAlgoOrders[0].LastPrice.Float64(), "the last filled price should decode")
				assert.Equal(t, "true", pendingAlgoOrders[0].ReduceOnly, "the documented reduceOnly string should decode")
			},
		},
		{
			name: "Recurring buy order list drops state",
			call: func() error {
				_, err := e.GetRecurringBuyOrderList(t.Context(), &RecurringBuyOrderListRequest{Limit: 30})
				return err
			},
			path:   "/tradingBot/recurring/orders-algo-pending",
			params: map[string]string{"limit": "30"},
			absent: []string{"state"},
		},
		{
			name: "Recurring order details drop state",
			call: func() error {
				var err error
				recurringOrderDetails, err = e.GetRecurringBuyOrderDetails(t.Context(), "560473220642766848")
				return err
			},
			path:   "/tradingBot/recurring/orders-algo-details",
			params: map[string]string{"algoId": "560473220642766848"},
			absent: []string{"state"},
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, recurringOrderDetails, "the recurring order detail response must decode")
				assert.Equal(t, "560473220642766848", recurringOrderDetails.AlgoID, "the documented algoId should decode")
				assert.Equal(t, "running", recurringOrderDetails.State, "the documented state should decode")
				assert.Equal(t, "100", recurringOrderDetails.Amount.String(), "the documented amt should decode")
				assert.Equal(t, []string{"1"}, recurringOrderDetails.Source, "the documented funding source array should decode")
				assert.Equal(t, "USDT", recurringOrderDetails.TradeQuoteCurrency.String(), "the documented tradeQuoteCcy should decode")
				require.Len(t, recurringOrderDetails.RecurringList, 1, "the detailed recurring list must decode")
				assert.Equal(t, "36683.2", recurringOrderDetails.RecurringList[0].Price.String(), "the documented purchase price should decode")
				assert.Equal(t, 30000.0, recurringOrderDetails.RecurringList[0].MinimumPrice.Float64(), "the documented minPx should decode")
				assert.Equal(t, 50000.0, recurringOrderDetails.RecurringList[0].MaximumPrice.Float64(), "the documented maxPx should decode")
			},
		},
		{
			name: "Taker volume drops instFamily",
			call: func() error {
				_, err := e.GetTakerVolume(t.Context(), &TakerVolumeRequest{Currency: currency.BTC, InstrumentType: instTypeSpot, Period: kline.OneDay})
				return err
			},
			path:   "/rubik/stat/taker-volume",
			params: map[string]string{"instType": "SPOT"},
			absent: []string{"instFamily"},
		},
		{
			name: "Deposit order detail sends ordId",
			call: func() error {
				_, err := e.GetFiatDepositOrderDetail(t.Context(), "12345")
				return err
			},
			path:   "/fiat/deposit",
			params: map[string]string{"ordId": "12345"},
			absent: []string{"ordID"},
		},
		{
			// The currency list is a different response schema from the
			// one-click repay and history endpoints it previously shared a
			// struct with: it returns debt and repay currency lists.
			name: "One-click repay currency list decodes debts and repay balances",
			call: func() error {
				var err error
				oneClickRepayCurrencyList, err = e.GetOneClickRepayCurrencyList(t.Context(), &OneClickRepayCurrencyListRequest{})
				return err
			},
			path: "/trade/one-click-repay-currency-list",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, oneClickRepayCurrencyList, 1, "the response item must decode")
				assert.Equal(t, "cross", oneClickRepayCurrencyList[0].DebtType, "the documented debt type should decode")
				require.Len(t, oneClickRepayCurrencyList[0].Debts, 1, "the documented debt list must decode")
				assert.Equal(t, "BTC", oneClickRepayCurrencyList[0].Debts[0].Currency.String(), "the debt currency should decode")
				assert.Equal(t, 1.5, oneClickRepayCurrencyList[0].Debts[0].Amount.Float64(), "the debt amount should decode")
				require.Len(t, oneClickRepayCurrencyList[0].Repays, 1, "the documented repay list must decode")
				assert.Equal(t, "USDT", oneClickRepayCurrencyList[0].Repays[0].Currency.String(), "the repay currency should decode")
				assert.Equal(t, 100.0, oneClickRepayCurrencyList[0].Repays[0].Amount.Float64(), "the repay amount should decode")
			},
		},
		{
			// Sub orders are order-level rows, not grid-algo-level rows: the
			// previously shared struct decoded every sub order field to zero.
			name: "Grid sub orders decode the sub order rows",
			call: func() error {
				var err error
				gridSubOrders, err = e.GetGridAlgoSubOrders(t.Context(), &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "12345", SubOrderType: "live"})
				return err
			},
			path: "/tradingBot/grid/sub-orders",
			params: map[string]string{
				"algoOrdType": "grid",
				"algoId":      "12345",
				"type":        "live",
			},
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, gridSubOrders, 1, "the sub order row must decode")
				assert.Equal(t, "grid-ord-1", gridSubOrders[0].OrderID, "the sub order ID should decode")
				assert.Equal(t, "grid-client-1", gridSubOrders[0].AlgoClientOrderID, "the documented algoClOrdId should decode")
				assert.Equal(t, 42000.0, gridSubOrders[0].Price.Float64(), "the sub order price should decode")
				assert.Equal(t, 12.5, gridSubOrders[0].PNL.Float64(), "the sub order PnL should decode")
				assert.Equal(t, "USDT", gridSubOrders[0].Currency.String(), "the documented ccy should decode")
				assert.Equal(t, 0.01, gridSubOrders[0].Rebate.Float64(), "the documented rebate should decode")
			},
		},
		{
			name: "Filled grid sub orders request the documented state",
			call: func() error {
				var err error
				gridSubOrders, err = e.GetGridAlgoSubOrders(t.Context(), &GridAlgoSubOrdersRequest{AlgoOrderType: AlgoOrdTypeGrid, AlgoID: "12345", SubOrderType: "filled", Limit: 20})
				return err
			},
			path: "/tradingBot/grid/sub-orders",
			params: map[string]string{
				"algoOrdType": "grid",
				"algoId":      "12345",
				"type":        "filled",
				"limit":       "20",
			},
		},
		{
			name: "Index components decode the component price",
			call: func() error {
				var err error
				indexComponents, err = e.GetIndexComponents(t.Context(), "BTC-USDT")
				return err
			},
			path: "/market/index-components",
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, indexComponents, "the index component response must decode")
				require.Len(t, indexComponents.Components, 1, "the documented components must decode")
				assert.Equal(t, 42000.5, indexComponents.Components[0].SymbolPairPrice.Float64(), "the documented symPx component price should decode")
			},
		},
		{
			name: "Discount rate decodes the discount tiers",
			call: func() error {
				var err error
				discountRates, err = e.GetDiscountRateAndInterestFreeQuota(t.Context(), &DiscountRateRequest{Currency: currency.BTC})
				return err
			},
			path: "/public/discount-rate-interest-free-quota",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, discountRates, 1, "the discount rate row must decode")
				require.Len(t, discountRates[0].Details, 1, "the documented discount tier details must decode")
				assert.Equal(t, 0.98, discountRates[0].Details[0].DiscountRate.Float64(), "the tier discount rate should decode")
				assert.Equal(t, "1", discountRates[0].Details[0].Tier, "the tier number should decode")
			},
		},
		{
			name: "Portfolio margin position tiers decode posType",
			call: func() error {
				var err error
				pmPositionLimitations, err = e.GetPMPositionLimitation(t.Context(), &PMPositionLimitationRequest{InstrumentType: "SWAP", InstrumentFamilies: []string{"BTC-USD"}})
				return err
			},
			path: "/account/position-tiers",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, pmPositionLimitations, 1, "the position tier row must decode")
				assert.Equal(t, "1", pmPositionLimitations[0].PositionType, "the documented posType should decode")
			},
		},
		{
			name: "Convert currencies decode the currency ID",
			call: func() error {
				var err error
				convertCurrencies, err = e.GetConvertCurrencies(t.Context())
				return err
			},
			path: "/asset/convert/currencies",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, convertCurrencies, 1, "the convert currency row must decode")
				assert.Equal(t, "BTC", convertCurrencies[0].Currency.String(), "the documented ccy should decode")
			},
		},
		{
			name: "Maker instrument settings decode includeAll",
			call: func() error {
				var err error
				quoteProducts, err = e.GetQuoteProducts(t.Context())
				return err
			},
			path: "/rfq/maker-instrument-settings",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, quoteProducts, 1, "the quote product row must decode")
				assert.True(t, quoteProducts[0].IncludeAll, "the documented includeAll should decode")
			},
		},
		{
			name: "Unrealised profit sharing decodes the unrealised amount",
			call: func() error {
				var err error
				unrealisedProfitSharing, err = e.GetUnrealisedProfitSharingDetails(t.Context(), "SWAP")
				return err
			},
			path: "/copytrading/unrealized-profit-sharing-details",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, unrealisedProfitSharing, 1, "the profit sharing row must decode")
				assert.Equal(t, 0.455472, unrealisedProfitSharing[0].UnrealisedProfitSharingAmount.Float64(), "the documented unrealizedProfitSharingAmt should decode")
			},
		},
		{
			name: "Account risk state decodes the atRisk boolean",
			call: func() error {
				var err error
				accountRiskState, err = e.GetAccountRiskState(t.Context())
				return err
			},
			path: "/account/risk-state",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, accountRiskState, 1, "the risk state row must decode")
				assert.True(t, accountRiskState[0].AtRisk, "the documented atRisk boolean should decode")
			},
		},
		{
			name: "Signal bot event history decodes the alert message",
			call: func() error {
				var err error
				signalBotEventHistory, err = e.GetSignalBotEventHistory(t.Context(), &SignalBotEventHistoryRequest{AlgoID: "12345"})
				return err
			},
			path: "/tradingBot/signal/event-history",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, signalBotEventHistory, 1, "the event history row must decode")
				assert.Equal(t, "price alert", signalBotEventHistory[0].AlertMessage, "the documented alertMsg string should decode")
			},
		},
		{
			name: "Positions decode the realised PnL accumulations",
			call: func() error {
				var err error
				accountPositions, err = e.GetPositions(t.Context(), &PositionsRequest{InstrumentType: "SWAP"})
				return err
			},
			path: "/account/positions",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, accountPositions, 1, "the position row must decode")
				assert.Equal(t, 12.5, accountPositions[0].RealisedProfitAndLoss.Float64(), "the documented realizedPnl should decode")
				assert.Equal(t, -0.1, accountPositions[0].FundingFee.Float64(), "the accumulated funding fee should decode")
				assert.Equal(t, 41000.0, accountPositions[0].BreakEvenPrice.Float64(), "the documented bePx should decode")
			},
		},
		{
			name: "Position history decodes the documented types",
			call: func() error {
				_, err := e.GetPositionsHistory(t.Context(), &PositionsHistoryRequest{InstrumentType: "SWAP", Limit: 100})
				return err
			},
			path:   "/account/positions-history",
			params: map[string]string{"instType": "SWAP", "limit": "100"},
			verify: func(t *testing.T) {
				t.Helper()
				history, err := e.GetPositionsHistory(t.Context(), &PositionsHistoryRequest{InstrumentType: "SWAP", Limit: 100})
				require.NoError(t, err, "positions history must decode")
				require.Len(t, history, 1, "the history row must decode")
				row := history[0]
				assert.Equal(t, instTypeSwap, row.InstrumentType, "the documented instType should decode")
				assert.Equal(t, 3.0, row.Leverage.Float64(), "the documented lever should decode")
				assert.Equal(t, "2", row.CloseType, "the documented close type should decode")
				assert.Equal(t, 41000.0, row.OpenAveragePrice.Float64(), "the documented openAvgPx should decode")
				assert.Equal(t, 42000.5, row.CloseAveragePrice.Float64(), "the documented closeAvgPx should decode")
				assert.Equal(t, 40000.0, row.TriggerPrice.Float64(), "the documented triggerPx should decode")
				assert.Equal(t, -0.2, row.LiquidationPenalty.Float64(), "the documented liqPenalty should decode")
			},
		},
		{
			name: "Single funding rate decodes formula and realised rate",
			call: func() error {
				var err error
				fundingRates, err = e.GetFundingRate(t.Context(), "BTC-USD-SWAP")
				return err
			},
			path: "/public/funding-rate",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, fundingRates, 1, "the funding rate response must decode")
				assert.Equal(t, "withRate", fundingRates[0].FormulaType, "the documented formulaType should decode")
				assert.Equal(t, 0.00012, fundingRates[0].RealisedRate.Float64(), "the documented realizedRate should decode")
				assert.Equal(t, 1.2, fundingRates[0].ImpactValue.Float64(), "the documented impactValue should decode")
			},
		},
		{
			name: "Grid algo details decode trigger parameters and fees",
			call: func() error {
				var err error
				gridAlgoDetails, err = e.GetGridAlgoOrderDetails(t.Context(), &GridAlgoOrderRequest{AlgoOrderType: "grid", AlgoID: "12345"})
				return err
			},
			path: "/tradingBot/grid/orders-algo-details",
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, gridAlgoDetails, "the grid algo detail must decode")
				assert.Equal(t, "grid-client-1", gridAlgoDetails.AlgoClientOrderID, "the documented algoClOrdId should decode")
				assert.Equal(t, "BTC-USDT", gridAlgoDetails.InstrumentFamily, "the documented instFamily should decode")
				require.Len(t, gridAlgoDetails.TriggerParameters, 1, "the grid trigger parameters must decode")
				assert.Equal(t, "rsi", gridAlgoDetails.TriggerParameters[0].TriggerStrategy, "the trigger strategy should decode")
				assert.Equal(t, "cross_up", gridAlgoDetails.TriggerParameters[0].TriggerCondition, "the trigger condition should decode")
				assert.Equal(t, "30", gridAlgoDetails.TriggerParameters[0].Threshold.String(), "the RSI threshold should decode")
				assert.Equal(t, -0.2, gridAlgoDetails.Fee.Float64(), "the accumulated fee should decode")
			},
		},
		{
			name: "Leading positions decode margin and unrealised PnL",
			call: func() error {
				var err error
				leadingPositions, err = e.GetExistingLeadPositions(t.Context(), &LeadPositionsRequest{InstrumentType: "SWAP"})
				return err
			},
			path: "/copytrading/current-subpositions",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, leadingPositions, 1, "the sub position row must decode")
				assert.Equal(t, 100.0, leadingPositions[0].Margin.Float64(), "the documented margin should decode")
				assert.Equal(t, "USDT", leadingPositions[0].MarginCurrency.String(), "the documented margin currency should decode")
				assert.Equal(t, 1.5, leadingPositions[0].UnrealisedProfitAndLoss.Float64(), "the unrealised PnL should decode")
				assert.Equal(t, 50000.0, leadingPositions[0].TakeProfitOrderPrice.Float64(), "the take-profit order price should decode")
			},
		},
		{
			name: "Withdrawal history decodes address type and note",
			call: func() error {
				var err error
				withdrawalHistory, err = e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Currency: currency.BTC})
				return err
			},
			path: "/asset/withdrawal-history",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, withdrawalHistory, 2, "the withdrawal rows must decode")
				assert.Equal(t, "2", withdrawalHistory[0].ToAddressType, "the documented toAddrType should decode")
				assert.Equal(t, "w1", withdrawalHistory[0].Note, "the documented note should decode")
				assert.Equal(t, "123456", withdrawalHistory[0].AddressAttachment["comment"], "the documented addrEx attachment object should decode")
				assert.Nil(t, withdrawalHistory[1].AddressAttachment, "the documented null addrEx should decode into a nil map")
			},
		},
		{
			name: "Subaccount list decodes uid and level",
			call: func() error {
				var err error
				subaccountList, err = e.GetSubAccountList(t.Context(), &SubAccountListRequest{})
				return err
			},
			path: "/users/subaccount/list",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, subaccountList, 1, "the subaccount row must decode")
				assert.Equal(t, "123456", subaccountList[0].UID, "the documented uid should decode")
				assert.Equal(t, "1", subaccountList[0].SubAccountLevel, "the documented subAcctLv should decode")
				assert.Equal(t, "sub-one", subaccountList[0].FirstLevelSubAccount, "the documented firstLvSubAcct name should decode")
				assert.True(t, subaccountList[0].DirectMarketAccess, "the documented ifDma flag should decode")
			},
		},
		{
			name: "MMP config decodes the quoted time interval",
			call: func() error {
				var err error
				mmpConfigs, err = e.GetMMPConfig(t.Context(), &MMPConfigRequest{})
				return err
			},
			path: "/account/mmp-config",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, mmpConfigs, 1, "the MMP config row must decode")
				assert.Equal(t, int64(5000), mmpConfigs[0].TimeInterval.Int64(), "the documented quoted timeInterval should decode")
				assert.Equal(t, "BTC-USD", mmpConfigs[0].InstrumentFamily, "the documented instFamily should decode")
			},
		},
		{
			name: "Deposit withdraw status decodes the estimated completion time",
			call: func() error {
				var err error
				depositWithdrawStatuses, err = e.GetDepositWithdrawalStatus(t.Context(), &DepositWithdrawalStatusRequest{WithdrawalID: "1244"})
				return err
			},
			path:   "/asset/deposit-withdraw-status",
			params: map[string]string{"wdId": "1244"},
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, depositWithdrawStatuses, 1, "the deposit withdraw status row must decode")
				expected := time.Date(2023, time.January, 9, 20, 10, 48, 0, time.FixedZone("UTC+8", 8*60*60))
				assert.True(t, depositWithdrawStatuses[0].EstimatedCompleteTime.Time().Equal(expected), "the documented estCompleteTime wall-clock form should decode")
			},
		},
		{
			name: "Position builder decodes the top-level positions",
			call: func() error {
				var err error
				positionBuilder, err = e.NewPositionBuilder(t.Context(), &PositionBuilderRequest{IncludeRealPositionsAndEquity: true})
				return err
			},
			path: "/account/position-builder",
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, positionBuilder, "the position builder response must decode")
				require.Len(t, positionBuilder.Positions, 1, "the documented top-level positions must decode")
				assert.Equal(t, "BTC-USDT-SWAP", positionBuilder.Positions[0].InstrumentID, "the position instrument should decode")
				assert.Equal(t, 1.0, positionBuilder.Positions[0].Amount.Float64(), "the position amount should decode")
				assert.Equal(t, 0.5, positionBuilder.AccountLeverage.Float64(), "the documented acctLever should decode")
				assert.Equal(t, 1.2, positionBuilder.BorrowMaintenanceMarginRequirement.Float64(), "the documented borrowMmr should decode")
				assert.Equal(t, 0.8, positionBuilder.DerivativesMaintenanceMarginRequirement.Float64(), "the documented derivMmr should decode")
				assert.Equal(t, 100.0, positionBuilder.Equity.Float64(), "the documented eq should decode")
				assert.Equal(t, 0.02, positionBuilder.MarginRatio.Float64(), "the documented marginRatio should decode")
				assert.Equal(t, 2.5, positionBuilder.TotalInitialMarginRequirement.Float64(), "the documented totalImr should decode")
				assert.Equal(t, 1.5, positionBuilder.TotalMaintenanceMarginRequirement.Float64(), "the documented totalMmr should decode")
				require.Len(t, positionBuilder.Assets, 1, "the documented asset row must decode")
				assert.Equal(t, "USDT", positionBuilder.Assets[0].Currency.String(), "the asset currency should decode")
				assert.Equal(t, 5.0, positionBuilder.Assets[0].SpotInUse.Float64(), "the documented spotInUse should decode")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call(), "the pinned request must not error")
			path, query := lastRequest()
			assert.Equal(t, tc.path, path, "the documented endpoint should be requested")
			for name, want := range tc.params {
				assert.Equalf(t, want, query.Get(name), "parameter %s should carry the documented value", name)
			}
			for _, name := range tc.absent {
				assert.NotContainsf(t, query, name, "OKX ignores an undocumented %s parameter, so it should not be sent", name)
			}
			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

func TestGetAccountInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountInstruments(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAccountInstruments(t.Context(), &AccountInstrumentsRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetAccountInstruments(t.Context(), &AccountInstrumentsRequest{InstrumentType: instTypeOption, InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.GetAccountInstruments(t.Context(), &AccountInstrumentsRequest{InstrumentType: instTypeEvents})
	require.ErrorIs(t, err, errSeriesIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountInstruments(contextGenerate(), &AccountInstrumentsRequest{InstrumentType: instTypeSpot, InstrumentID: "BTC-EUR"})
	require.NoError(t, err, "GetAccountInstruments must not error")
	if mockTests {
		exp := []AccountInstrumentResponse{{
			InstrumentType:                   instTypeSpot,
			SeriesID:                         "BTC-ABOVE-DAILY",
			InstrumentID:                     "BTC-EUR",
			Underlying:                       "BTC-USD",
			TradingFeeGroupID:                "4",
			InstrumentFamily:                 "BTC-USD",
			BaseCurrency:                     currency.BTC,
			QuoteCurrency:                    currency.EUR,
			SettlementCurrency:               currency.BTC,
			ContractValue:                    0.01,
			ContractMultiplier:               1,
			ContractValueCurrency:            currency.BTC,
			OptionType:                       "C",
			StrikePrice:                      100000,
			ListTime:                         types.Time(time.UnixMilli(1704876947000)),
			AuctionEndTime:                   types.Time(time.UnixMilli(1704876900000)),
			ContinuousTradingSwitchTime:      types.Time(time.UnixMilli(1704876947000)),
			PreMarketSwitchTime:              types.Time(time.UnixMilli(1704877000000)),
			OpenType:                         "call_auction",
			RPIPermission:                    "2",
			ExpiryTime:                       types.Time(time.UnixMilli(1767225600000)),
			MaxLeverage:                      10,
			TickSize:                         1,
			LotSize:                          0.00000001,
			MinimumOrderSize:                 0.00001,
			ContractType:                     "linear",
			State:                            "live",
			RuleType:                         "normal",
			PositionLimitAmount:              2500000,
			PositionLimitPercent:             30,
			MaxPlatformOpenInterestLimit:     1000000000,
			MaxPlatformOpenInterestCoinLimit: 25000,
			LongPositionRemainingQuota:       2000000,
			ShortPositionRemainingQuota:      1500000,
			MaxLimitOrderSize:                9999999999,
			MaxMarketOrderSize:               1000000,
			MaxLimitOrderAmountUSD:           1000000,
			MaxMarketOrderAmountUSD:          1000000,
			MaxTWAPOrderSize:                 9999999999,
			MaxIcebergOrderSize:              9999999999,
			MaxTriggerOrderSize:              9999999999,
			MaxStopOrderSize:                 1000000,
			FutureSettlement:                 true,
			TradeQuoteCurrencies:             []currency.Code{currency.EUR},
			InstrumentIDCode:                 1000000000,
			InstrumentCategory:               "1",
			InitialPriceLimitPercent:         0.05,
			FloatingPriceLimitPercent:        0.03,
			MaxPriceLimitPercent:             0.15,
			UpcomingChanges:                  []InstrumentUpcomingChange{{Parameter: "tickSz", NewValue: "0.0001", EffectiveTime: types.Time(time.UnixMilli(1704876947000))}},
		}}
		assert.Equal(t, exp, result, "GetAccountInstruments should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountInstruments should return a result")

	result, err = e.GetAccountInstruments(contextGenerate(), &AccountInstrumentsRequest{InstrumentType: instTypeSpot, InstrumentID: mainPair.String()})
	require.NoError(t, err, "GetAccountInstruments must not error for a spot instrument")
	assert.NotNil(t, result, "GetAccountInstruments should return spot instruments")

	result, err = e.GetAccountInstruments(contextGenerate(), &AccountInstrumentsRequest{InstrumentType: instTypeSwap, InstrumentFamily: mainPair.String(), InstrumentID: perpetualSwapPair.String()})
	require.NoError(t, err, "GetAccountInstruments must not error for a perpetual swap")
	assert.NotNil(t, result, "GetAccountInstruments should return perpetual swap instruments")

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.NotEmpty(t, p, "GetEnabledPairs must not return empty pairs")
	instFamily, err := e.instrumentFamilyFromInstID(instTypeOption, p[0].String())
	require.NoError(t, err, "instrumentFamilyFromInstID must not error")
	result, err = e.GetAccountInstruments(contextGenerate(), &AccountInstrumentsRequest{InstrumentType: instTypeOption, InstrumentFamily: instFamily, InstrumentID: p[0].String()})
	require.NoError(t, err, "GetAccountInstruments must not error for an option")
	assert.NotNil(t, result, "GetAccountInstruments should return option instruments")
}

// TestInstrumentUpcomingChanges guards the upcChg shape OKX documents for the
// public and account instrument endpoints: param, newValue and effTime.
func TestInstrumentUpcomingChanges(t *testing.T) {
	t.Parallel()
	const upcomingChanges = `{"instId":"BTC-USDT","upcChg":[{"param":"tickSz","newValue":"0.0001","effTime":"1704876947000"}]}`
	exp := []InstrumentUpcomingChange{{Parameter: "tickSz", NewValue: "0.0001", EffectiveTime: types.Time(time.UnixMilli(1704876947000))}}
	var i InstrumentResponse
	require.NoError(t, json.Unmarshal([]byte(upcomingChanges), &i), "InstrumentResponse must decode")
	assert.Equal(t, exp, i.UpcomingChanges, "InstrumentResponse should decode each upcoming change")
	var a AccountInstrumentResponse
	require.NoError(t, json.Unmarshal([]byte(upcomingChanges), &a), "AccountInstrumentResponse must decode")
	assert.Equal(t, exp, a.UpcomingChanges, "AccountInstrumentResponse should decode each upcoming change")
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAccountBalance(t.Context(), &AccountBalanceRequest{Currencies: make([]currency.Code, 21)})
	require.ErrorIs(t, err, errExceedLimit)

	t.Run("refuses a null account row", func(t *testing.T) {
		t.Parallel()
		const account = `{"uTime":"1795400001000","totalEq":"100","details":[{"ccy":"USDT","eq":"100","uTime":"1795400001000"}]}`
		for _, tc := range []struct {
			name, data string
			expErr     error
			exp        []AccountBalanceResponse
		}{
			{
				name: "an account",
				data: `[` + account + `]`,
				exp: []AccountBalanceResponse{{
					UpdateTime:  types.Time(time.UnixMilli(1795400001000)),
					TotalEquity: 100,
					Details:     []AccountBalanceDetail{{Currency: currency.USDT, EquityOfCurrency: 100, UpdateTime: types.Time(time.UnixMilli(1795400001000))}},
				}},
			},
			{name: "a null row", data: `[null]`, expErr: common.ErrInvalidResponse},
			{name: "a null row after an account", data: `[` + account + `,null]`, expErr: common.ErrInvalidResponse},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/account/balance", r.URL.Path, "the balances should come from the trading account")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				got, err := ex.GetAccountBalance(t.Context(), &AccountBalanceRequest{})
				require.ErrorIs(t, err, tc.expErr, "GetAccountBalance must refuse a reply holding a null account row, and only such a reply")
				assert.Equal(t, tc.exp, got, "GetAccountBalance should return a reply's accounts only when it holds no null row")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBalance(contextGenerate(), &AccountBalanceRequest{Currencies: []currency.Code{currency.USDT}})
	require.NoError(t, err, "GetAccountBalance must not error")
	if mockTests {
		exp := []AccountBalanceResponse{{
			UpdateTime:                   types.Time(time.UnixMilli(1705474164160)),
			TotalEquity:                  55837.43556134779,
			IsolatedMarginEquity:         15.5,
			AdjustedEquity:               55415.624719833286,
			AvailableEquity:              55415.624719833286,
			OrderFrozen:                  6.5,
			InitialMarginRequirement:     1250.5,
			MaintenanceMarginRequirement: 520.25,
			BorrowFrozen:                 12.5,
			MarginRatio:                  48.75,
			NotionalUSD:                  25000.5,
			NotionalUSDForBorrow:         1500.5,
			NotionalUSDForSwap:           13500.5,
			NotionalUSDForFutures:        7500.25,
			NotionalUSDForOption:         2500.75,
			UnrealisedProfitAndLoss:      -7.5456,
			Delta:                        1520.75,
			DeltaLeverage:                0.0272,
			DeltaNeutralStatus:           "1",
			Details: []AccountBalanceDetail{{
				Currency:                           currency.USDT,
				EquityOfCurrency:                   4992.890093622894,
				CashBalance:                        4850.435693622894,
				UpdateTime:                         types.Time(time.UnixMilli(1705449605015)),
				IsolatedEquity:                     15.5,
				AvailableEquity:                    4834.3170936228935,
				DiscountEquity:                     4991.542013297616,
				FixedBalance:                       4.5,
				AvailableBalance:                   4834.317093622894,
				FrozenBalance:                      158.573,
				OrderFrozen:                        6.5,
				Liabilities:                        21625.64,
				UnrealisedProfitAndLoss:            -7.545600000000006,
				UnrealisedProfitAndLossLiabilities: 3.5,
				CrossLiabilities:                   21.5,
				RewardBalance:                      1.5,
				IsolatedLiabilities:                2.25,
				MarginRatio:                        12.5,
				InitialMarginRequirement:           105.25,
				MaintenanceMarginRequirement:       42.1,
				Interest:                           9.01,
				ForcedRepaymentRiskLevel:           1,
				ForcedRepaymentType:                "1",
				MaxLoan:                            35000,
				EquityUSD:                          4991.542013297616,
				BorrowFrozen:                       3.25,
				NotionalLeverage:                   3.5,
				StrategyEquity:                     150,
				IsolatedUnrealisedProfitAndLoss:    -1.75,
				SpotInUseAmount:                    0.6,
				ClientSpotInUseAmount:              0.5,
				MaxSpotInUse:                       0.75,
				SpotIsolatedBalance:                0.4,
				SmartSyncEquity:                    7.5,
				SpotCopyTradingEquity:              8.5,
				SpotBalance:                        0.25,
				OpenAveragePrice:                   42000.5,
				AccumulatedAveragePrice:            41000.5,
				SpotUnrealisedProfitAndLoss:        275.5,
				SpotUnrealisedProfitAndLossRatio:   0.0265,
				TotalProfitAndLoss:                 310.25,
				TotalProfitAndLossRatio:            0.0301,
				CollateralRestriction:              "1",
				CollateralBorrowAutoConversion:     "2",
				CollateralBorrowRestricted:         true,
				CollateralEnabled:                  true,
				AutoLendStatus:                     "active",
				AutoLendMatchedAmount:              120.5,
			}},
		}}
		assert.Equal(t, exp, result, "GetAccountBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetAccountBalance should return a result")
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetPositions(t.Context(), &PositionsRequest{InstrumentIDs: make([]string, 11)})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetPositions(t.Context(), &PositionsRequest{PositionIDs: make([]string, 21)})
	require.ErrorIs(t, err, errExceedLimit)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPositions(contextGenerate(), &PositionsRequest{InstrumentType: instTypeMargin, InstrumentIDs: []string{mainPair.String()}})
	require.NoError(t, err, "GetPositions must not error")
	if mockTests {
		exp := []AccountPositionResponse{{
			InstrumentType:                        instTypeMargin,
			MarginMode:                            TradeModeIsolated,
			PositionID:                            "1752810569801498626",
			PositionSide:                          positionSideNet,
			QuantityOfPosition:                    0.00190433573,
			HedgedPosition:                        0.001,
			BaseBalance:                           0.0019,
			QuoteBalance:                          20.5,
			BaseBorrowed:                          0.0005,
			BaseInterest:                          0.0000012,
			QuoteBorrowed:                         99.9998,
			QuoteInterest:                         0.0081,
			PositionCurrency:                      currency.BTC,
			AvailablePosition:                     0.00190433573,
			AveragePrice:                          62961.4,
			NonSettlementAveragePrice:             62950.5,
			MarkPrice:                             62891.9,
			UnrealisedProfitAndLoss:               -0.0000033452492717,
			UnrealisedProfitAndLossRatio:          -0.0105311101755551,
			UnrealisedProfitAndLossLastPrice:      -0.0000033199677697,
			UnrealisedProfitAndLossRatioLastPrice: -0.0104515220008934,
			InstrumentID:                          mainPair.String(),
			Leverage:                              5,
			LiquidationPrice:                      53615.448336593756,
			InitialMarginRequirement:              0.0003,
			Margin:                                0.000317654,
			MarginRatio:                           9.404143929947395,
			MaintenanceMarginRequirement:          0.0000318005395854,
			Liabilities:                           -99.9998177776581948,
			LiabilitiesCurrency:                   currency.USDT,
			Interest:                              0.0000081,
			TradeID:                               "785524470",
			OptionValue:                           0.0021,
			PendingCloseOrderLiability:            12.5,
			NotionalUSD:                           119.756628017499,
			AutoDeleveraging:                      "1",
			Currency:                              currency.BTC,
			LastTradePrice:                        62892.9,
			IndexPrice:                            62890.5,
			USDPrice:                              62890.1,
			BreakEvenPrice:                        63050.5,
			DeltaBS:                               0.45,
			DeltaPA:                               0.0000071,
			GammaBS:                               0.000012,
			GammaPA:                               0.00000019,
			ThetaBS:                               -12.5,
			ThetaPA:                               -0.0002,
			VegaBS:                                25.5,
			VegaPA:                                0.0004,
			SpotInUseAmount:                       0.6,
			SpotInUseCurrency:                     currency.BTC,
			ClientSpotInUseAmount:                 0.5,
			MaxSpotInUseAmount:                    0.75,
			BusinessReferenceID:                   "coupon-1",
			BusinessReferenceType:                 "experience_coupon",
			RealisedProfitAndLoss:                 -0.0000172,
			SettledProfitAndLoss:                  0.0000021,
			ProfitAndLoss:                         0.0000125,
			Fee:                                   -0.0000254,
			FundingFee:                            -0.0000031,
			LiquidationPenalty:                    -0.0000012,
			CloseOrderAlgo:                        []PositionCloseOrderAlgo{{AlgoID: "1753184812254216192", StopLossTriggerPrice: 58000, StopLossTriggerPriceType: "mark", TakeProfitTriggerPrice: 68000, TakeProfitTriggerPriceType: "last", CloseFraction: 1}},
			CreationTime:                          types.Time(time.UnixMilli(1724740225685)),
			UpdateTime:                            types.Time(time.UnixMilli(1724742632153)),
		}}
		assert.Equal(t, exp, result, "GetPositions should decode every field")
	} else {
		assert.NotNil(t, result, "GetPositions should return a result")
	}
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetPositionsHistory(t.Context(), &PositionsHistoryRequest{Before: time.UnixMilli(1654177200000), After: time.UnixMilli(1654177100000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetPositionsHistory(t.Context(), &PositionsHistoryRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPositionsHistory(contextGenerate(), &PositionsHistoryRequest{
		InstrumentType: instTypeSwap,
		InstrumentID:   "BTC-USD-SWAP",
		MarginMode:     TradeModeCross,
		CloseType:      "1",
		PositionID:     "452587086133239818",
		After:          time.UnixMilli(1654177200000),
		Before:         time.UnixMilli(1654177100000),
		Limit:          1,
	})
	require.NoError(t, err, "GetPositionsHistory must not error")
	if mockTests {
		exp := []PositionHistoryResponse{{
			InstrumentType:            instTypeSwap,
			InstrumentID:              "BTC-USD-SWAP",
			MarginMode:                TradeModeCross,
			CloseType:                 "1",
			CreationTime:              types.Time(time.UnixMilli(1654177169995)),
			UpdateTime:                types.Time(time.UnixMilli(1654177174419)),
			OpenAveragePrice:          29783.8999999995535393,
			NonSettlementAveragePrice: 29780.5,
			CloseAveragePrice:         29786.5999999789081085,
			PositionID:                "452587086133239818",
			OpenMaxPosition:           1,
			CloseTotalPosition:        1,
			RealisedProfitAndLoss:     0.001,
			SettledProfitAndLoss:      0.00003,
			ProfitAndLossRatio:        0.000906447858888,
			Fee:                       -0.0001,
			FundingFee:                -0.00002,
			LiquidationPenalty:        -0.00001,
			ProfitAndLoss:             0.0011,
			PositionSide:              positionSideLong,
			Leverage:                  10,
			Direction:                 "long",
			TriggerPrice:              29500.5,
			Underlying:                "BTC-USD",
			Currency:                  currency.BTC,
		}}
		assert.Equal(t, exp, result, "GetPositionsHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetPositionsHistory should return a result")
	}
}

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountAndPositionRisk(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	t.Run("decodes the notional amounts", func(t *testing.T) {
		t.Parallel()
		// notionalCcy is the documented notional value in coin, a quantity
		// beside notionalUsd rather than a currency.
		var positionData []PositionRiskPosition
		require.NoError(t, json.Unmarshal([]byte(`[{"instId":"BTC-USDT","notionalCcy":"0.4","notionalUsd":"25000"}]`), &positionData), "Unmarshal must not error")
		exp := []PositionRiskPosition{{InstrumentID: "BTC-USDT", NotionalCoin: types.Number(0.4), NotionalUSD: types.Number(25000)}}
		assert.Equal(t, exp, positionData, "the position risk row should decode its notional amounts")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountAndPositionRisk(contextGenerate(), &AccountPositionRiskRequest{InstrumentType: instTypeMargin})
	require.NoError(t, err, "GetAccountAndPositionRisk must not error")
	if mockTests {
		exp := []AccountPositionRiskResponse{{
			Timestamp:      types.Time(time.UnixMilli(1620282889345)),
			AdjustedEquity: 174238.6793649711331679,
			Balances: []PositionRiskBalance{
				{Currency: currency.BTC, Equity: 1.3863533369419636, DiscountEquity: 78846.7803721021362242},
				{Currency: currency.USDT, Equity: 73323.395564963177146, DiscountEquity: 73417.2495112863300127},
			},
			Positions: []PositionRiskPosition{{
				InstrumentType:   instTypeMargin,
				MarginMode:       TradeModeIsolated,
				PositionID:       "310388685292318723",
				InstrumentID:     mainPair.String(),
				Position:         0.4,
				BaseBalance:      0.4,
				QuoteBalance:     12.5,
				PositionSide:     positionSideNet,
				PositionCurrency: currency.BTC,
				Currency:         currency.USDT,
				NotionalCoin:     0.4,
				NotionalUSD:      25000.5,
			}},
		}}
		assert.Equal(t, exp, result, "GetAccountAndPositionRisk should decode every field")
	} else {
		assert.NotNil(t, result, "GetAccountAndPositionRisk should return a result")
	}
}

func TestGetBillsDetailLast7Days(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsDetailLast7Days(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetBillsDetailLast7Days(t.Context(), &BillsDetailRequest{BeginTime: time.UnixMilli(1695033477000), EndTime: time.UnixMilli(1695033476000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetBillsDetailLast7Days(t.Context(), &BillsDetailRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailRequest{
		InstrumentType: instTypeSpot,
		InstrumentID:   mainPair.String(),
		Currency:       currency.USDT,
		MarginMode:     TradeModeIsolated,
		BillType:       2,
		BillSubType:    1,
		After:          "623950854533513220",
		Before:         "623950854533513218",
		BeginTime:      time.UnixMilli(1695033476000),
		EndTime:        time.UnixMilli(1695033477000),
		Limit:          1,
	})
	require.NoError(t, err, "GetBillsDetailLast7Days must not error")
	if mockTests {
		exp := []BillsDetailResponse{{
			InstrumentType:             instTypeSpot,
			BillID:                     "623950854533513219",
			Type:                       "2",
			SubType:                    "1",
			Timestamp:                  types.Time(time.UnixMilli(1695033476167)),
			BalanceChange:              0.0219338232210000,
			PositionLevelBalanceChange: 0.25,
			Balance:                    8694.2179403378290202,
			PositionLevelBalance:       12.5,
			Size:                       0.021955779,
			Price:                      27105.9,
			Currency:                   currency.USDT,
			ProfitAndLoss:              0.0035,
			Fee:                        -0.000021955779,
			EarnAmount:                 0.0015,
			EarnAPR:                    0.0525,
			MarginMode:                 TradeModeIsolated,
			InstrumentID:               mainPair.String(),
			OrderID:                    "623950854525124608",
			ExecutionType:              "T",
			From:                       "6",
			To:                         "18",
			Notes:                      "transfer note",
			Interest:                   0.0012,
			Tag:                        "gct-tag",
			FillTime:                   types.Time(time.UnixMilli(1695033476166)),
			TradeID:                    "586760148",
			ClientOrderID:              "b15e8d5e1c4bBCDE",
			FillIndexPrice:             27104.1,
			FillMarkPrice:              27106.2,
			FillPriceVolatility:        0.42,
			FillPriceUSD:               27105.9,
			FillMarkVolatility:         0.45,
			FillForwardPrice:           27110.5,
		}}
		assert.Equal(t, exp, result, "GetBillsDetailLast7Days should decode every field")
	} else {
		assert.NotNil(t, result, "GetBillsDetailLast7Days should return a result")
	}
}

func TestGetBillsDetail3Months(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsDetail3Months(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetBillsDetail3Months(t.Context(), &BillsDetailRequest{BeginTime: time.UnixMilli(1695033477000), EndTime: time.UnixMilli(1695033476000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetBillsDetail3Months(t.Context(), &BillsDetailRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBillsDetail3Months(contextGenerate(), &BillsDetailRequest{InstrumentType: instTypeSpot, Currency: currency.USDT, Limit: 1})
	require.NoError(t, err, "GetBillsDetail3Months must not error")
	if mockTests {
		exp := []BillsDetailResponse{{
			InstrumentType: instTypeSpot,
			BillID:         "623950854533513219",
			Type:           "2",
			SubType:        "1",
			Timestamp:      types.Time(time.UnixMilli(1695033476167)),
			BalanceChange:  0.0219338232210000,
			Balance:        8694.2179403378290202,
			Size:           0.021955779,
			Price:          27105.9,
			Currency:       currency.USDT,
			Fee:            -0.000021955779,
			MarginMode:     TradeModeIsolated,
			InstrumentID:   mainPair.String(),
			OrderID:        "623950854525124608",
			ExecutionType:  "T",
			FillTime:       types.Time(time.UnixMilli(1695033476166)),
			TradeID:        "586760148",
			FillIndexPrice: 27104.1,
		}}
		assert.Equal(t, exp, result, "GetBillsDetail3Months should decode every field")
	} else {
		assert.NotNil(t, result, "GetBillsDetail3Months should return a result")
	}
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *BillsDetailRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "window begins after it ends", arg: &BillsDetailRequest{BeginTime: time.UnixMilli(1708587373362), EndTime: time.UnixMilli(1708587373361)}, err: common.ErrStartAfterEnd},
		{name: "limit above 100", arg: &BillsDetailRequest{Limit: 101}, err: errLimitValueExceedsMaxOf100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.getBillsDetail(t.Context(), tc.arg, "account/bills-archive", getBillsDetailArchiveEPL)
			assert.ErrorIs(t, err, tc.err, "getBillsDetail should reject the request before sending it")
		})
	}

	begin, end := time.UnixMilli(1708587373361), time.UnixMilli(1708587373362)
	reply := []map[string]string{{"instType": "MARGIN", "billId": "623950854533513219", "type": "2", "subType": "1", "ccy": "USDT", "balChg": "-5.5", "ts": "1708587373362"}}
	exp := []BillsDetailResponse{{
		InstrumentType: instTypeMargin, BillID: "623950854533513219", Type: "2", SubType: "1", Currency: currency.USDT, BalanceChange: -5.5,
		Timestamp: types.Time(time.UnixMilli(1708587373362)),
	}}
	for _, tc := range []struct {
		name  string
		arg   *BillsDetailRequest
		query url.Values
	}{
		{name: "no filter", arg: &BillsDetailRequest{}, query: url.Values{}},
		{name: "instrument type sent upper case", arg: &BillsDetailRequest{InstrumentType: "margin"}, query: url.Values{"instType": {"MARGIN"}}},
		{name: "instrument ID", arg: &BillsDetailRequest{InstrumentID: mainPair.String()}, query: url.Values{"instId": {"BTC-USDT"}}},
		{name: "currency sent upper case", arg: &BillsDetailRequest{Currency: currency.USDT.Lower()}, query: url.Values{"ccy": {"USDT"}}},
		{name: "margin mode", arg: &BillsDetailRequest{MarginMode: TradeModeIsolated}, query: url.Values{"mgnMode": {"isolated"}}},
		{name: "contract type", arg: &BillsDetailRequest{ContractType: "linear"}, query: url.Values{"ctType": {"linear"}}},
		{name: "bill type", arg: &BillsDetailRequest{BillType: 2}, query: url.Values{"type": {"2"}}},
		{name: "bill sub type", arg: &BillsDetailRequest{BillSubType: 1}, query: url.Values{"subType": {"1"}}},
		{name: "after", arg: &BillsDetailRequest{After: "623950854533513219"}, query: url.Values{"after": {"623950854533513219"}}},
		{name: "before", arg: &BillsDetailRequest{Before: "623950854533513218"}, query: url.Values{"before": {"623950854533513218"}}},
		{name: "begin", arg: &BillsDetailRequest{BeginTime: begin}, query: url.Values{"begin": {"1708587373361"}}},
		{name: "end", arg: &BillsDetailRequest{EndTime: end}, query: url.Values{"end": {"1708587373362"}}},
		{name: "equal begin and end", arg: &BillsDetailRequest{BeginTime: begin, EndTime: begin}, query: url.Values{"begin": {"1708587373361"}, "end": {"1708587373361"}}},
		{name: "limit at 100", arg: &BillsDetailRequest{Limit: 100}, query: url.Values{"limit": {"100"}}},
		{
			name: "every filter",
			arg: &BillsDetailRequest{
				InstrumentType: instTypeMargin, InstrumentID: mainPair.String(), Currency: currency.USDT, MarginMode: TradeModeIsolated, ContractType: "linear",
				BillType: 2, BillSubType: 1, After: "623950854533513219", Before: "623950854533513218", BeginTime: begin, EndTime: end, Limit: 50,
			},
			query: url.Values{
				"instType": {"MARGIN"}, "instId": {"BTC-USDT"}, "ccy": {"USDT"}, "mgnMode": {"isolated"}, "ctType": {"linear"}, "type": {"2"}, "subType": {"1"},
				"after": {"623950854533513219"}, "before": {"623950854533513218"}, "begin": {"1708587373361"}, "end": {"1708587373362"}, "limit": {"50"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newQueryPinExchange(t, "/account/bills-archive", tc.query, reply)
			result, err := ex.getBillsDetail(t.Context(), tc.arg, "account/bills-archive", getBillsDetailArchiveEPL)
			require.NoError(t, err, "getBillsDetail must not error")
			assert.Equal(t, exp, result, "getBillsDetail should decode the bills from the caller's route")
		})
	}
}

func TestApplyBillDetails(t *testing.T) {
	t.Parallel()
	_, err := e.ApplyBillDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ApplyBillDetails(t.Context(), &BillsHistoryArchiveRequest{Quarter: "Q2"})
	require.ErrorIs(t, err, errYearRequired)
	_, err = e.ApplyBillDetails(t.Context(), &BillsHistoryArchiveRequest{Year: 2023})
	require.ErrorIs(t, err, errQuarterValueRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *BillsHistoryArchiveRequest
			body string
		}{
			{"every parameter", &BillsHistoryArchiveRequest{Year: 2023, Quarter: "Q2", BillTypes: "1,2"}, `{"year":"2023","quarter":"Q2","type":"1,2"}`},
			{"required parameters only", &BillsHistoryArchiveRequest{Year: 2023, Quarter: "Q2"}, `{"year":"2023","quarter":"Q2"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/bills-history-archive", tc.body, []map[string]string{{"result": "false", "ts": "1646892328000"}})
				_, err := ex.ApplyBillDetails(t.Context(), tc.req)
				require.NoError(t, err, "ApplyBillDetails must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ApplyBillDetails(contextGenerate(), &BillsHistoryArchiveRequest{Year: 2023, Quarter: "Q2", BillTypes: "1,2"})
	require.NoError(t, err, "ApplyBillDetails must not error")
	if mockTests {
		exp := &ApplyBillDetailsResponse{Result: true, Timestamp: types.Time(time.UnixMilli(1646892328000))}
		assert.Equal(t, exp, result, "ApplyBillDetails should decode every field")
	} else {
		assert.NotNil(t, result, "ApplyBillDetails should return a result")
	}
}

func TestGetBillsHistoryArchive(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsHistoryArchive(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetBillsHistoryArchive(t.Context(), &BillsHistoryArchiveRequest{Quarter: "Q2"})
	require.ErrorIs(t, err, errYearRequired)
	_, err = e.GetBillsHistoryArchive(t.Context(), &BillsHistoryArchiveRequest{Year: 2023})
	require.ErrorIs(t, err, errQuarterValueRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBillsHistoryArchive(contextGenerate(), &BillsHistoryArchiveRequest{Year: 2023, Quarter: "Q2", BillTypes: "1,2"})
	require.NoError(t, err, "GetBillsHistoryArchive must not error")
	if mockTests {
		exp := []BillsHistoryArchiveResponse{{FileHref: "http://xxx", Timestamp: types.Time(time.UnixMilli(1646892328000)), State: "finished"}}
		assert.Equal(t, exp, result, "GetBillsHistoryArchive should decode every field")
	} else {
		assert.NotNil(t, result, "GetBillsHistoryArchive should return a result")
	}
}

func TestBillsHistoryArchiveRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BillsHistoryArchiveRequest
		err  error
	}{
		{"nil request", nil, common.ErrNilPointer},
		{"missing year", &BillsHistoryArchiveRequest{Quarter: "Q1"}, errYearRequired},
		{"missing quarter", &BillsHistoryArchiveRequest{Year: 2024}, errQuarterValueRequired},
		{"valid", &BillsHistoryArchiveRequest{Year: 2024, Quarter: "Q1"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.validate(), tc.err, "validate should return the expected error")
		})
	}
}

func TestGetBillTypes(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillTypes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBillTypes(contextGenerate(), &BillTypesRequest{BillTypes: "1"})
	require.NoError(t, err, "GetBillTypes must not error")
	if mockTests {
		exp := []BillTypeResponse{{
			BillType:            "1",
			BillTypeDescription: "Transfer",
			SubTypeDetails:      []BillSubTypeDetail{{SubType: "11", SubTypeDescription: "Transfer in"}, {SubType: "12", SubTypeDescription: "Transfer out"}},
		}}
		assert.Equal(t, exp, result, "GetBillTypes should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetBillTypes should return bill types")
	}
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountConfiguration(contextGenerate())
	require.NoError(t, err, "GetAccountConfiguration must not error")
	if mockTests {
		exp := &AccountConfigurationResponse{
			UserID:                         "44705892343619585",
			MainUserID:                     "44705892343619584",
			AccountLevel:                   2,
			AccountSelfTradePreventionMode: "cancel_maker",
			PositionMode:                   positionModeLongShort,
			AutoLoan:                       true,
			GreeksType:                     "PA",
			FeeType:                        "1",
			Level:                          "Lv1",
			LevelTemporary:                 "Lv2",
			ContractIsolatedMode:           "automatic",
			MarginIsolatedMode:             "automatic",
			SpotOffsetType:                 "1",
			StrategyType:                   "1",
			RoleType:                       1,
			TraderInstruments:              []string{"BTC-USDT-SWAP"},
			SpotRoleType:                   2,
			SpotTraderInstruments:          []string{mainPair.String()},
			OptionTradingAuthorisation:     1,
			KYCLevel:                       3,
			Label:                          "v5 test",
			IPAddresses:                    "117.37.203.58,117.37.203.57",
			Permissions:                    "read_only,withdraw,trade",
			LiquidationGear:                -1,
			SpotBorrowEnabled:              true,
			SpotBorrowAutoRepay:            true,
			AccountType:                    1,
			SettlementCurrency:             currency.USDC,
			SettlementCurrencies:           []currency.Code{currency.USD, currency.USDC, currency.USDG},
		}
		assert.Equal(t, exp, result, "GetAccountConfiguration should decode every field")
	} else {
		assert.NotNil(t, result, "GetAccountConfiguration should return a result")
	}
}

// accountSwitch is an account mode switch a test makes
type accountSwitch func(context.Context, *Exchange) error

// positionModeSwitch returns a switch to the position mode
func positionModeSwitch(mode string) accountSwitch {
	return func(ctx context.Context, e *Exchange) error {
		_, err := e.SetPositionMode(ctx, mode)
		return err
	}
}

// accountLevelSwitch returns a switch to the account level
func accountLevelSwitch(level string) accountSwitch {
	return func(ctx context.Context, e *Exchange) error {
		_, err := e.SetAccountMode(ctx, level)
		return err
	}
}

// testSwitchesSerialise makes the first switch, whose reply the mock holds, and then the second, checking that the
// second reaches the exchange only once the first has completed and that the cached modes follow the later switch
func testSwitchesSerialise(t *testing.T, first, second accountSwitch, expModes map[string]string, expLevels map[string]uint64) {
	t.Helper()
	arrived := make(chan string, 4)
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	var held sync.Once
	ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "reading the switch body should not error") {
			return
		}
		arrived <- r.URL.Path
		held.Do(func() { <-release })
		switch r.URL.Path {
		case "/account/set-position-mode":
			var req SetPositionModeRequest
			assert.NoError(t, json.Unmarshal(body, &req), "the position mode switch should decode")
			writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
		case "/account/set-account-level":
			var req SetAccountModeRequest
			assert.NoError(t, json.Unmarshal(body, &req), "the account level switch should decode")
			writeOKXData(t, w, []map[string]string{{"acctLv": req.AccountLevel}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// Registered after the mock server's cleanup, so it runs first and frees a handler still holding its reply
	t.Cleanup(releaseOnce)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "KEY-1", Secret: "secret", ClientID: "passphrase"})
	done := make(chan error, 2)
	go func() { done <- first(ctx, ex) }()
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the first switch must reach the exchange")
	}
	go func() { done <- second(ctx, ex) }()
	select {
	case path := <-arrived:
		assert.Failf(t, "a switch should wait for the one in flight", "the %s switch reached the exchange while the first was in flight", path)
	case <-time.After(100 * time.Millisecond):
	}
	releaseOnce()
	for range 2 {
		select {
		case err := <-done:
			require.NoError(t, err, "each switch must complete")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "both switches must complete")
		}
	}
	assert.Equal(t, expModes, ex.accountPositionModes, "the cached position modes should follow the later switch")
	assert.Equal(t, expLevels, ex.accountLevels, "the cached account levels should follow the later switch")
}

// testQueuedSwitchGivesUp makes the first switch, whose reply the mock holds, and then the second with a context
// whose deadline passes while it waits, checking that the second returns its context's error before the first
// completes, without reaching the exchange
func testQueuedSwitchGivesUp(t *testing.T, first, second accountSwitch) {
	t.Helper()
	arrived := make(chan string, 4)
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	var held sync.Once
	ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "reading the switch body should not error") {
			return
		}
		arrived <- r.URL.Path
		held.Do(func() { <-release })
		switch r.URL.Path {
		case "/account/set-position-mode":
			var req SetPositionModeRequest
			assert.NoError(t, json.Unmarshal(body, &req), "the position mode switch should decode")
			writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
		case "/account/set-account-level":
			var req SetAccountModeRequest
			assert.NoError(t, json.Unmarshal(body, &req), "the account level switch should decode")
			writeOKXData(t, w, []map[string]string{{"acctLv": req.AccountLevel}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// Registered after the mock server's cleanup, so it runs first and frees a handler still holding its reply
	t.Cleanup(releaseOnce)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "KEY-1", Secret: "secret", ClientID: "passphrase"})
	done := make(chan error, 1)
	go func() { done <- first(ctx, ex) }()
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the first switch must reach the exchange")
	}
	queued, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	gaveUp := make(chan error, 1)
	go func() { gaveUp <- second(queued, ex) }()
	select {
	case err := <-gaveUp:
		require.ErrorIs(t, err, context.DeadlineExceeded, "a queued switch must return its context's error")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a queued switch must give up when its context ends, before the switch in flight completes")
	}
	releaseOnce()
	select {
	case err := <-done:
		require.NoError(t, err, "the first switch must complete")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the first switch must complete")
	}
	select {
	case path := <-arrived:
		assert.Failf(t, "a switch which gave up should not reach the exchange", "the %s switch reached the exchange", path)
	default:
	}
	assert.Equal(t, uint64(1), ex.accountPositionModeSwitches, "the switch which gave up should leave the cached modes to the one in flight")
}

// accountModeCaches returns an exchange whose cached modes and levels are populated for two keys, with handler
// answering its requests
func accountModeCaches(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := newMockExchange(t, handler)
	ex.accountPositionModes = map[string]string{"KEY-1": positionModeNet, "KEY-2": positionModeLongShort}
	ex.accountLevels = map[string]uint64{"KEY-1": 2, "KEY-2": 3}
	return ex
}

func TestSwitchOutcomeUnknown(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		exp  bool
	}{
		{name: "an OKX refusal", err: fmt.Errorf("authenticated request failed: %w", &ResponseError{Code: 59000})},
		{name: "OKX's endpoint timeout", err: fmt.Errorf("authenticated request failed: %w", &ResponseError{Code: endpointRequestTimeoutCode}), exp: true},
		{name: "a failed transport", err: errors.New("connection reset"), exp: true},
		{name: "a dead context", err: context.Canceled, exp: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, switchOutcomeUnknown(tc.err), "switchOutcomeUnknown should report whether the switch may have been applied")
		})
	}
}

func TestForgetAccountModes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		levelSwitch bool
		expLevels   map[string]uint64
	}{
		{name: "a position mode switch", expLevels: map[string]uint64{"KEY-2": 3}},
		{name: "an account level switch", levelSwitch: true, expLevels: map[string]uint64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{
				accountPositionModes:        map[string]string{"KEY-1": positionModeNet, "KEY-2": positionModeLongShort},
				accountLevels:               map[string]uint64{"KEY-1": 2, "KEY-2": 3},
				accountPositionModeSwitches: 4,
			}
			ex.forgetAccountModes("KEY-1", tc.levelSwitch)
			assert.Empty(t, ex.accountPositionModes, "every key's position mode should be forgotten")
			assert.Equal(t, tc.expLevels, ex.accountLevels, "the levels the switch may have changed should be forgotten")
			assert.Equal(t, uint64(5), ex.accountPositionModeSwitches, "the switch should be counted")
		})
	}
}

// TestSetPositionMode is not parallel: the mocked switch refreshes the shared
// exchange's position mode cache, which a concurrent contract order lookup
// would see as a switch overlapping its fetch.
func TestSetPositionMode(t *testing.T) { //nolint:tparallel // The recorded switch mutates the shared exchange, while the order placement subtests use their own
	_, err := e.SetPositionMode(t.Context(), "")
	require.ErrorIs(t, err, errInvalidPositionMode)

	t.Run("recorded", func(t *testing.T) {
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.SetPositionMode(contextGenerate(), positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		assert.Equal(t, &PositionModeResponse{PositionMode: positionModeLongShort}, result, "SetPositionMode should decode every field")
	})

	t.Run("refreshes order placement", func(t *testing.T) {
		t.Parallel()
		t.Run("placement follows the confirmed mode", func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/account/set-position-mode":
					writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
					return
				case "/trade/order":
				default:
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := ex.SetPositionMode(t.Context(), positionModeLongShort)
			require.NoError(t, err, "SetPositionMode must not error")
			_, err = ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange:   ex.Name,
				Pair:       mainPair,
				AssetType:  asset.Futures,
				Side:       order.Buy,
				Type:       order.Limit,
				Amount:     1,
				Price:      1,
				MarginType: margin.Multi,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the order request body must decode")
			assert.Equal(t, positionSideLong, sent["posSide"], "the order should follow the confirmed position mode without refetching it")
		})
		for _, tc := range []struct {
			name string
			data any
			err  error
		}{
			{"empty reply", nil, common.ErrNoResponse},
			{"null row", []any{nil}, common.ErrNoResponse},
			{"unrecognised mode", []map[string]string{{"posMode": "reverse"}}, errInvalidPositionMode},
			{"another mode", []map[string]string{{"posMode": positionModeNet}}, errModeSwitchMismatch},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/account/set-position-mode" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					writeOKXData(t, w, tc.data)
				}))
				_, err := ex.SetPositionMode(t.Context(), positionModeLongShort)
				require.ErrorIsf(t, err, tc.err, "SetPositionMode must report a %s", tc.name)
				assert.Empty(t, ex.accountPositionModes, "an unconfirmed mode switch should not refresh the cache")
			})
		}
	})

	t.Run("a later switch waits for an earlier one to complete", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name          string
			first, second accountSwitch
			expModes      map[string]string
			expLevels     map[string]uint64
		}{
			{"after a position mode switch", positionModeSwitch(positionModeLongShort), positionModeSwitch(positionModeNet), map[string]string{"KEY-1": positionModeNet}, map[string]uint64{}},
			{"after an account level switch", accountLevelSwitch("3"), positionModeSwitch(positionModeNet), map[string]string{"KEY-1": positionModeNet}, map[string]uint64{"KEY-1": 3}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				testSwitchesSerialise(t, tc.first, tc.second, tc.expModes, tc.expLevels)
			})
		}
	})

	t.Run("a queued switch gives up when its context ends", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			first accountSwitch
		}{
			{"behind a position mode switch", positionModeSwitch(positionModeLongShort)},
			{"behind an account level switch", accountLevelSwitch("3")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				testQueuedSwitchGivesUp(t, tc.first, positionModeSwitch(positionModeNet))
			})
		}
	})

	t.Run("an unconfirmed switch forgets the modes it may have changed", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			reply     string
			timeout   bool
			err       error
			expModes  map[string]string
			expLevels map[string]uint64
			switches  uint64
		}{
			{name: "no reply", reply: `{"code":"0","msg":"","data":null}`, err: common.ErrNoResponse, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "a null row", reply: `{"code":"0","msg":"","data":[null]}`, err: common.ErrNoResponse, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "an empty row", reply: `{"code":"0","msg":"","data":[{}]}`, err: errInvalidPositionMode, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "an unrecognised mode", reply: `{"code":"0","msg":"","data":[{"posMode":"reverse"}]}`, err: errInvalidPositionMode, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "another mode", reply: `{"code":"0","msg":"","data":[{"posMode":"net_mode"}]}`, err: errModeSwitchMismatch, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "OKX's endpoint timeout", reply: `{"code":"50004","msg":"","data":[]}`, err: ErrorCodes["50004"], expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{name: "a timed out request", timeout: true, err: context.DeadlineExceeded, expModes: map[string]string{}, expLevels: map[string]uint64{"KEY-2": 3}, switches: 1},
			{
				name: "a switch OKX refused", reply: `{"code":"59000","msg":"","data":[]}`, err: ErrorCodes["59000"],
				expModes: map[string]string{"KEY-1": positionModeNet, "KEY-2": positionModeLongShort}, expLevels: map[string]uint64{"KEY-1": 2, "KEY-2": 3},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				release := make(chan struct{})
				releaseOnce := sync.OnceFunc(func() { close(release) })
				ex := accountModeCaches(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/account/set-position-mode" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if tc.timeout {
						<-release
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(tc.reply))
					assert.NoError(t, err, "writing the reply should not error")
				})
				ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "KEY-1", Secret: "secret", ClientID: "passphrase"})
				if tc.timeout {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
				}
				// Registered after the mock server's cleanup, so it runs first and frees a handler still holding its reply
				t.Cleanup(releaseOnce)
				got, err := ex.SetPositionMode(ctx, positionModeLongShort)
				releaseOnce()
				require.ErrorIs(t, err, tc.err, "SetPositionMode must report the unconfirmed switch")
				assert.Nil(t, got, "SetPositionMode should return no confirmation")
				assert.Equal(t, tc.expModes, ex.accountPositionModes, "the position modes the switch may have changed should be forgotten")
				assert.Equal(t, tc.expLevels, ex.accountLevels, "the switching key's account level should be forgotten with them")
				assert.Equal(t, tc.switches, ex.accountPositionModeSwitches, "an unconfirmed switch should be counted so an overlapping fetch caches nothing")
			})
		}
	})

	t.Run("acts for one account when the exchange's credentials change", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/set-position-mode" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			keys = append(keys, r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
		}))
		ex.SetCredentials(primaryAccount())
		_, err := ex.SetPositionMode(&credentialSwitchContext{Context: t.Context(), ex: ex, next: secondaryAccount()}, positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		assert.Equal(t, map[string]string{"primary-key": positionModeLongShort}, ex.accountPositionModes, "the confirmed mode should be cached for the account it was switched for")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"primary-key"}, keys, "the switch should be sent for the account whose mode it caches")
	})
}

func TestSetLeverageRate(t *testing.T) {
	t.Parallel()
	_, err := e.SetLeverageRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{Leverage: 5, MarginMode: "isolated"})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{MarginMode: "isolated", InstrumentID: perpetualSwapPair.String()})
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{
		Leverage:     5,
		MarginMode:   "cash",
		InstrumentID: perpetualSwapPair.String(),
	})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{
		Currency:   currency.USDT,
		Leverage:   5,
		MarginMode: "isolated",
	})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.SetLeverageRate(t.Context(), &SetLeverageRequest{
		Leverage:     5,
		MarginMode:   "isolated",
		InstrumentID: perpetualSwapPair.String(),
		PositionSide: "reverse",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("live only: sets the leverage of a real account")
		}
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		_, err := e.SetLeverageRate(contextGenerate(), &SetLeverageRequest{
			Currency:     currency.USDT,
			Leverage:     5,
			MarginMode:   "cross",
			InstrumentID: perpetualSwapPair.String(),
		})
		assert.Truef(t, err == nil || errors.Is(err, common.ErrNoResponse), "SetLeverageRate should not error: %s", err)
	})

	t.Run("normalises the position side", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/set-leverage" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the set leverage body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"lever": "5", "mgnMode": TradeModeIsolated, "posSide": positionSideLong, "instId": perpetualSwapPair.String()}})
		}))
		_, err := ex.SetLeverageRate(t.Context(), &SetLeverageRequest{Leverage: 5, MarginMode: TradeModeIsolated, InstrumentID: perpetualSwapPair.String(), PositionSide: "LONG"})
		require.NoError(t, err, "SetLeverageRate must accept an upper-case position side")
		mu.Lock()
		var sent map[string]any
		err = json.Unmarshal(body, &sent)
		mu.Unlock()
		require.NoError(t, err, "the set leverage body must decode")
		assert.Equal(t, positionSideLong, sent["posSide"], "the position side should be sent in lower case")

		_, err = ex.SetLeverageRate(t.Context(), &SetLeverageRequest{Leverage: 5, MarginMode: TradeModeIsolated, Currency: currency.USDT})
		require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "a currency-scoped isolated leverage must be rejected")
		assert.ErrorContains(t, err, `requires "cross" margin`, "the error should name the margin mode a currency-scoped leverage requires")
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SetLeverageRequest
			body string
		}{
			{"every parameter", &SetLeverageRequest{InstrumentID: perpetualSwapPair.String(), Currency: currency.USDT, Leverage: 30, MarginMode: TradeModeIsolated, PositionSide: positionSideLong}, `{"instId":"BTC-USDT-SWAP","ccy":"USDT","lever":"30","mgnMode":"isolated","posSide":"long"}`},
			{"required parameters only", &SetLeverageRequest{Currency: currency.USDT, Leverage: 3, MarginMode: TradeModeCross}, `{"ccy":"USDT","lever":"3","mgnMode":"cross"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/set-leverage", tc.body, []map[string]string{{"lever": "3"}})
				_, err := ex.SetLeverageRate(t.Context(), tc.req)
				require.NoError(t, err, "SetLeverageRate must not error")
			})
		}
	})

	if mockTests {
		result, err := e.SetLeverageRate(contextGenerate(), &SetLeverageRequest{InstrumentID: perpetualSwapPair.String(), Leverage: 30, MarginMode: TradeModeIsolated, PositionSide: positionSideLong})
		require.NoError(t, err, "SetLeverageRate must not error")
		exp := &SetLeverageResponse{Leverage: 30, MarginMode: TradeModeIsolated, InstrumentID: perpetualSwapPair.String(), PositionSide: positionSideLong}
		assert.Equal(t, exp, result, "SetLeverageRate should decode every field")
	}
}

func TestGetMaximumBuySellAmountOrOpenAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), &MaximumOrderQuantityRequest{TradeMode: "cross", Currency: currency.BTC, Price: 5})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), &MaximumOrderQuantityRequest{InstrumentIDs: make([]string, 6), TradeMode: "cross"})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), &MaximumOrderQuantityRequest{InstrumentIDs: []string{mainPair.String()}, Currency: currency.BTC, Price: 5})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumBuySellAmountOrOpenAmount(contextGenerate(), &MaximumOrderQuantityRequest{InstrumentIDs: []string{mainPair.String()}, TradeMode: TradeModeCross, Currency: currency.BTC, Price: 27000, Leverage: 5})
	require.NoError(t, err, "GetMaximumBuySellAmountOrOpenAmount must not error")
	if mockTests {
		exp := []MaximumOrderQuantityResponse{{InstrumentID: mainPair.String(), Currency: currency.BTC, MaximumBuy: 0.0500695098559788, MaximumSell: 64.4798671570072269}}
		assert.Equal(t, exp, result, "GetMaximumBuySellAmountOrOpenAmount should decode every field")
	} else {
		assert.NotNil(t, result, "GetMaximumBuySellAmountOrOpenAmount should return a result")
	}
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumAvailableTradableAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetMaximumAvailableTradableAmount(t.Context(), &MaximumAvailableBalanceRequest{TradeMode: "cross", Currency: currency.BTC, ReduceOnly: true, Price: 123})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetMaximumAvailableTradableAmount(t.Context(), &MaximumAvailableBalanceRequest{InstrumentIDs: make([]string, 6), TradeMode: "cross"})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetMaximumAvailableTradableAmount(t.Context(), &MaximumAvailableBalanceRequest{InstrumentIDs: []string{mainPair.String()}, Currency: currency.BTC, ReduceOnly: true, Price: 123})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumAvailableTradableAmount(contextGenerate(), &MaximumAvailableBalanceRequest{InstrumentIDs: []string{mainPair.String()}, TradeMode: TradeModeIsolated, Currency: currency.BTC, ReduceOnly: true, Price: 27000})
	require.NoError(t, err, "GetMaximumAvailableTradableAmount must not error")
	if mockTests {
		exp := []MaximumAvailableBalanceResponse{{InstrumentID: mainPair.String(), AvailableBuy: 100, AvailableSell: 1}}
		assert.Equal(t, exp, result, "GetMaximumAvailableTradableAmount should decode every field")
	} else {
		assert.NotNil(t, result, "GetMaximumAvailableTradableAmount should return a result")
	}
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	_, err := e.IncreaseDecreaseMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.IncreaseDecreaseMargin(t.Context(), &IncreaseDecreaseMarginRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &IncreaseDecreaseMarginRequest{Currency: currency.USD}
	_, err = e.IncreaseDecreaseMargin(t.Context(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.IncreaseDecreaseMargin(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.IncreaseDecreaseMargin(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.MarginBalanceType = "reduce"
	_, err = e.IncreaseDecreaseMargin(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *IncreaseDecreaseMarginRequest
			body string
		}{
			{"every parameter", &IncreaseDecreaseMarginRequest{InstrumentID: mainPair.String(), PositionSide: positionSideNet, MarginBalanceType: marginBalanceAdd, Amount: 0.3, Currency: currency.BTC}, `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"0.3","ccy":"BTC"}`},
			{"required parameters only", &IncreaseDecreaseMarginRequest{InstrumentID: perpetualSwapPair.String(), PositionSide: positionSideLong, MarginBalanceType: marginBalanceReduce, Amount: 10}, `{"instId":"BTC-USDT-SWAP","posSide":"long","type":"reduce","amt":"10"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/position/margin-balance", tc.body, []map[string]string{{"amt": "0.3"}})
				_, err := ex.IncreaseDecreaseMargin(t.Context(), tc.req)
				require.NoError(t, err, "IncreaseDecreaseMargin must not error")
			})
		}
	})

	if mockTests {
		result, err := e.IncreaseDecreaseMargin(contextGenerate(), &IncreaseDecreaseMarginRequest{InstrumentID: mainPair.String(), PositionSide: positionSideNet, MarginBalanceType: marginBalanceAdd, Amount: 0.3, Currency: currency.BTC})
		require.NoError(t, err, "IncreaseDecreaseMargin must not error")
		exp := &IncreaseDecreaseMarginResponse{InstrumentID: mainPair.String(), PositionSide: positionSideNet, Amount: 0.3, Type: marginBalanceAdd, Leverage: 4.5, Currency: currency.BTC}
		assert.Equal(t, exp, result, "IncreaseDecreaseMargin should decode every field")
		return
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.IncreaseDecreaseMargin(contextGenerate(), &IncreaseDecreaseMarginRequest{
		InstrumentID:      mainPair.String(),
		PositionSide:      "long",
		MarginBalanceType: "add",
		Amount:            1000,
		Currency:          currency.USD,
	})
	require.NoError(t, err, "IncreaseDecreaseMargin must not error")
	assert.NotNil(t, result, "IncreaseDecreaseMargin should return a result")
}

func TestGetLeverageRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverageRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeverageRate(t.Context(), &LeverageInfoRequest{MarginMode: "cross"})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)
	_, err = e.GetLeverageRate(t.Context(), &LeverageInfoRequest{InstrumentIDs: make([]string, 21), MarginMode: "cross"})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetLeverageRate(t.Context(), &LeverageInfoRequest{Currencies: make([]currency.Code, 21), MarginMode: "cross"})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetLeverageRate(t.Context(), &LeverageInfoRequest{InstrumentIDs: []string{mainPair.String()}})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("live only: the recorded call below covers mock mode")
		}
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetLeverageRate(contextGenerate(), &LeverageInfoRequest{InstrumentIDs: []string{mainPair.String()}, MarginMode: "cross"})
		require.NoError(t, err, "GetLeverageRate must not error")
		assert.NotNil(t, result, "GetLeverageRate should return a result")
	})

	t.Run("sends the currency", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var gotCcy string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/leverage-info" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			gotCcy = r.URL.Query().Get("ccy")
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{})
		}))
		_, err := ex.GetLeverageRate(t.Context(), &LeverageInfoRequest{InstrumentIDs: []string{mainPair.String()}, Currencies: []currency.Code{currency.BTC}, MarginMode: TradeModeCross})
		require.NoError(t, err, "GetLeverageRate must not error with a currency set")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, "BTC", gotCcy, "the currency should be sent as the ccy parameter")
	})

	if mockTests {
		result, err := e.GetLeverageRate(contextGenerate(), &LeverageInfoRequest{InstrumentIDs: []string{perpetualSwapPair.String()}, Currencies: []currency.Code{currency.BTC}, MarginMode: TradeModeCross})
		require.NoError(t, err, "GetLeverageRate must not error")
		exp := []LeverageResponse{
			{InstrumentID: perpetualSwapPair.String(), MarginMode: TradeModeCross, PositionSide: positionSideLong, Leverage: 10},
			{InstrumentID: perpetualSwapPair.String(), MarginMode: TradeModeCross, PositionSide: positionSideShort, Leverage: 10},
			{Currency: currency.BTC, MarginMode: TradeModeCross, Leverage: 5},
		}
		assert.Equal(t, exp, result, "GetLeverageRate should decode every field")
	}
}

func TestGetLeverageEstimatedInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverageEstimatedInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLeverageEstimatedInfo(t.Context(), &LeverageEstimatedInfoRequest{MarginMode: "cross", Leverage: 1, InstrumentID: mainPair.String(), Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetLeverageEstimatedInfo(t.Context(), &LeverageEstimatedInfoRequest{InstrumentType: "MARGIN", Leverage: 1, InstrumentID: mainPair.String(), Currency: currency.BTC})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.GetLeverageEstimatedInfo(t.Context(), &LeverageEstimatedInfoRequest{InstrumentType: "MARGIN", MarginMode: "cross", InstrumentID: mainPair.String(), Currency: currency.BTC})
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = e.GetLeverageEstimatedInfo(t.Context(), &LeverageEstimatedInfoRequest{InstrumentType: instTypeSwap, MarginMode: "cross", Leverage: 1})
	require.ErrorIs(t, err, errMissingInstrumentID)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLeverageEstimatedInfo(contextGenerate(), &LeverageEstimatedInfoRequest{InstrumentType: instTypeMargin, MarginMode: TradeModeIsolated, Leverage: 3, InstrumentID: mainPair.String(), Currency: currency.BTC, PositionSide: positionSideNet})
	require.NoError(t, err, "GetLeverageEstimatedInfo must not error")
	if mockTests {
		exp := []LeverageEstimatedInfoResponse{{
			EstimatedAvailableQuoteTransfer: 1520.5,
			EstimatedAvailableTransfer:      1.1398040558348279,
			EstimatedLiquidationPrice:       51000.5,
			EstimatedMargin:                 0.0701959441651721,
			EstimatedQuoteMargin:            4400.25,
			EstimatedMaxAmount:              10.6095865868904898,
			EstimatedQuoteMaxAmount:         176889.6871254563042714,
			ExistingOrders:                  true,
			MaxLeverage:                     10,
			MinLeverage:                     0.01,
		}}
		assert.Equal(t, exp, result, "GetLeverageEstimatedInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetLeverageEstimatedInfo should return a result")
	}
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumLoanOfInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetMaximumLoanOfInstrument(t.Context(), &MaximumLoanRequest{InstrumentIDs: []string{"ZRX-BTC"}, MarginCurrency: currency.ZRX})
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.GetMaximumLoanOfInstrument(t.Context(), &MaximumLoanRequest{MarginMode: "isolated", MarginCurrency: currency.ZRX})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)
	_, err = e.GetMaximumLoanOfInstrument(t.Context(), &MaximumLoanRequest{MarginMode: "isolated", InstrumentIDs: make([]string, 6)})
	require.ErrorIs(t, err, errExceedLimit)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumLoanOfInstrument(contextGenerate(), &MaximumLoanRequest{MarginMode: TradeModeIsolated, InstrumentIDs: []string{mainPair.String()}, MarginCurrency: currency.USDT})
	require.NoError(t, err, "GetMaximumLoanOfInstrument must not error")
	if mockTests {
		exp := []MaximumLoanResponse{
			{InstrumentID: mainPair.String(), MarginMode: TradeModeIsolated, MarginCurrency: currency.USDT, MaxLoan: 0.1, Currency: currency.BTC, Side: "sell"},
			{InstrumentID: mainPair.String(), MarginMode: TradeModeIsolated, MarginCurrency: currency.USDT, MaxLoan: 0.2, Currency: currency.USDT, Side: "buy"},
		}
		assert.Equal(t, exp, result, "GetMaximumLoanOfInstrument should decode every field")
	} else {
		assert.NotNil(t, result, "GetMaximumLoanOfInstrument should return a result")
	}
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetTradeFee(t.Context(), &TradeFeeRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetTradeFee(t.Context(), &TradeFeeRequest{InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), GroupID: "1"})
	require.ErrorIs(t, err, errTradeFeeGroupIDConflict)
	_, err = e.GetTradeFee(t.Context(), &TradeFeeRequest{InstrumentType: instTypeSwap, InstrumentFamily: mainPair.String(), GroupID: "1"})
	require.ErrorIs(t, err, errTradeFeeGroupIDConflict)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradeFee(contextGenerate(), &TradeFeeRequest{InstrumentType: instTypeSpot, InstrumentID: mainPair.String()})
	require.NoError(t, err, "GetTradeFee must not error")
	if mockTests {
		groupMaker, groupTaker := types.Number(-0.0008), types.Number(-0.001)
		maker, taker := types.Number(-0.0008), types.Number(-0.001)
		makerUSDT, takerUSDT := types.Number(-0.0002), types.Number(-0.0005)
		makerUSDC, takerUSDC := types.Number(-0.0006), types.Number(-0.0009)
		exp := []TradeFeeRateResponse{{
			FeeRateLevel:      "Lv1",
			FeeGroup:          []TradeFeeGroup{{GroupID: "1", MakerRate: &groupMaker, TakerRate: &groupTaker, RPIMakerRate: -0.0008}},
			DeliveryFeeRate:   0.0005,
			ExerciseFeeRate:   0.0003,
			InstrumentType:    instTypeSpot,
			Timestamp:         types.Time(time.UnixMilli(1763979985847)),
			FeeRateTaker:      &taker,
			FeeRateMaker:      &maker,
			FeeRateTakerUSDT:  &takerUSDT,
			FeeRateMakerUSDT:  &makerUSDT,
			FeeRateTakerUSDC:  &takerUSDC,
			FeeRateMakerUSDC:  &makerUSDC,
			RuleType:          "normal",
			Category:          "1",
			Fiat:              []FiatItemInfo{{Currency: currency.USD, Taker: -0.002, Maker: -0.0015}},
			SettlementFeeRate: 0.0001,
		}}
		assert.Equal(t, exp, result, "GetTradeFee should decode every field")
	} else {
		assert.NotNil(t, result, "GetTradeFee should return a result")
	}
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	_, err := e.GetInterestAccruedData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetInterestAccruedData(t.Context(), &InterestAccruedRequest{Before: time.UnixMilli(1637312400001), After: time.UnixMilli(1637049599999)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetInterestAccruedData(t.Context(), &InterestAccruedRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInterestAccruedData(contextGenerate(), &InterestAccruedRequest{
		LoanType:     "2",
		Currency:     currency.USDT,
		InstrumentID: mainPair.String(),
		MarginMode:   TradeModeCross,
		After:        time.UnixMilli(1637312400001),
		Before:       time.UnixMilli(1637049599999),
		Limit:        2,
	})
	require.NoError(t, err, "GetInterestAccruedData must not error")
	if mockTests {
		exp := []InterestAccruedResponse{
			{LoanType: "2", Currency: currency.USDT, InstrumentID: mainPair.String(), MarginMode: TradeModeCross, Interest: 0.0003960833333334, InterestRate: 0.0000040833333333, Liability: 97, TotalLiability: 197, InterestFreeLiability: 100, Timestamp: types.Time(time.UnixMilli(1637312400000))},
			{LoanType: "2", Currency: currency.USDT, InstrumentID: mainPair.String(), MarginMode: TradeModeCross, Interest: 0.0004083333333334, InterestRate: 0.0000040833333333, Liability: 100, Timestamp: types.Time(time.UnixMilli(1637049600000))},
		}
		assert.Equal(t, exp, result, "GetInterestAccruedData should decode every field")
	} else {
		assert.NotNil(t, result, "GetInterestAccruedData should return a result")
	}
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInterestRate(contextGenerate(), &InterestRateRequest{Currency: currency.BTC})
	require.NoError(t, err, "GetInterestRate must not error")
	if mockTests {
		assert.Equal(t, []InterestRateResponse{{InterestRate: 0.0001, Currency: currency.BTC}}, result, "GetInterestRate should decode every field")
	} else {
		assert.NotNil(t, result, "GetInterestRate should return a result")
	}
}

func TestSetFeeType(t *testing.T) {
	t.Parallel()
	_, err := e.SetFeeType(t.Context(), "")
	require.ErrorIs(t, err, errFeeTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFeeType(contextGenerate(), "1")
	require.NoError(t, err, "SetFeeType must not error")
	if mockTests {
		assert.Equal(t, &FeeTypeResponse{FeeType: "1"}, result, "SetFeeType should decode every field")
	} else {
		assert.NotNil(t, result, "SetFeeType should return a result")
	}
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	_, err := e.SetGreeks(t.Context(), "")
	require.ErrorIs(t, err, errMissingValidGreeksType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetGreeks(contextGenerate(), "PA")
	require.NoError(t, err, "SetGreeks must not error")
	if mockTests {
		assert.Equal(t, &GreeksTypeResponse{GreeksType: "PA"}, result, "SetGreeks should decode every field")
	} else {
		assert.NotNil(t, result, "SetGreeks should return a result")
	}
}

func TestSetIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	_, err := e.SetIsolatedMarginTradingSettings(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetIsolatedMarginTradingSettings(t.Context(), &IsolatedMarginTradingSettingsRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetIsolatedMarginTradingSettings(t.Context(), &IsolatedMarginTradingSettingsRequest{IsolatedMode: "", InstrumentType: "MARGIN"})
	require.ErrorIs(t, err, errMissingIsolatedMarginTradingSetting)
	_, err = e.SetIsolatedMarginTradingSettings(t.Context(), &IsolatedMarginTradingSettingsRequest{IsolatedMode: "autonomy", InstrumentType: ""})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/account/set-isolated-mode", `{"isoMode":"auto_transfers_ccy","type":"MARGIN"}`, []map[string]string{{"isoMode": "auto_transfers_ccy"}})
		_, err := ex.SetIsolatedMarginTradingSettings(t.Context(), &IsolatedMarginTradingSettingsRequest{IsolatedMode: "auto_transfers_ccy", InstrumentType: instTypeMargin})
		require.NoError(t, err, "SetIsolatedMarginTradingSettings must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetIsolatedMarginTradingSettings(contextGenerate(), &IsolatedMarginTradingSettingsRequest{IsolatedMode: "automatic", InstrumentType: instTypeMargin})
	require.NoError(t, err, "SetIsolatedMarginTradingSettings must not error")
	if mockTests {
		assert.Equal(t, &IsolatedMarginTradingSettingsResponse{IsolatedMode: "automatic"}, result, "SetIsolatedMarginTradingSettings should decode every field")
	} else {
		assert.NotNil(t, result, "SetIsolatedMarginTradingSettings should return a result")
	}
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumWithdrawals(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetMaximumWithdrawals(t.Context(), &MaximumWithdrawalsRequest{Currencies: make([]currency.Code, 21)})
	require.ErrorIs(t, err, errExceedLimit)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumWithdrawals(contextGenerate(), &MaximumWithdrawalsRequest{Currencies: []currency.Code{currency.BTC, currency.ETH}})
	require.NoError(t, err, "GetMaximumWithdrawals must not error")
	if mockTests {
		exp := []MaximumWithdrawalResponse{
			{Currency: currency.BTC, MaxWithdrawal: 124, MaxWithdrawalIncludingBorrowed: 125, SpotOffsetMaxWithdrawal: 126, SpotOffsetMaxWithdrawalIncludingBorrowed: 127},
			{Currency: currency.ETH, MaxWithdrawal: 10, MaxWithdrawalIncludingBorrowed: 12},
		}
		assert.Equal(t, exp, result, "GetMaximumWithdrawals should decode every field")
	} else {
		assert.NotNil(t, result, "GetMaximumWithdrawals should return a result")
	}
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountRiskState(contextGenerate())
	require.NoError(t, err, "GetAccountRiskState must not error")
	if mockTests {
		exp := []AccountRiskStateResponse{{AtRisk: true, AtRiskDerivativesUnits: []string{mainPair.String()}, AtRiskMarginUnits: []string{mainPair.String()}, Timestamp: types.Time(time.UnixMilli(1635745078794))}}
		assert.Equal(t, exp, result, "GetAccountRiskState should decode every field")
	} else {
		assert.NotNil(t, result, "GetAccountRiskState should return a result")
	}
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowInterestAndLimit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBorrowInterestAndLimit(contextGenerate(), &BorrowInterestAndLimitRequest{LoanType: "2", Currency: currency.BTC})
	require.NoError(t, err, "GetBorrowInterestAndLimit must not error")
	if mockTests {
		exp := []BorrowInterestAndLimitResponse{{
			Debt:             0.85893159114900247077000000000000,
			Interest:         1e-32,
			NextDiscountTime: types.Time(time.UnixMilli(1729490400000)),
			NextInterestTime: types.Time(time.UnixMilli(1729490400000)),
			LoanAllocation:   50,
			Records: []BorrowInterestAndLimitRecord{{
				Currency:                 currency.BTC,
				Rate:                     0.0000276,
				LoanQuota:                175,
				SurplusLimit:             175,
				UsedLimit:                0.5,
				Interest:                 0.0002,
				InterestFreeLiability:    0.1,
				PotentialBorrowingAmount: 174.5,
				SurplusLimitDetails:      SurplusLimitDetail{AllAccountsRemainingQuota: 175, CurrentAccountRemainingQuota: 170, PlatformRemainingQuota: 600},
				PositionLoan:             2,
				AvailableLoan:            10,
				UsedLoan:                 0.5,
				AverageRate:              0.0000025,
			}},
		}}
		assert.Equal(t, exp, result, "GetBorrowInterestAndLimit should decode every field")
	} else {
		assert.NotNil(t, result, "GetBorrowInterestAndLimit should return a result")
	}
}

func TestManualBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := e.ManualBorrowOrRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ManualBorrowOrRepay(t.Context(), &ManualBorrowRepayRequest{Side: "borrow", Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.ManualBorrowOrRepay(t.Context(), &ManualBorrowRepayRequest{Currency: currency.USDT, Amount: 1})
	require.ErrorIs(t, err, errLendingSideRequired)
	_, err = e.ManualBorrowOrRepay(t.Context(), &ManualBorrowRepayRequest{Currency: currency.USDT, Side: "borrow"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/account/spot-manual-borrow-repay", `{"ccy":"USDT","side":"repay","amt":"12.5"}`, []map[string]string{{"ccy": "USDT"}})
		_, err := ex.ManualBorrowOrRepay(t.Context(), &ManualBorrowRepayRequest{Currency: currency.USDT, Side: "repay", Amount: 12.5})
		require.NoError(t, err, "ManualBorrowOrRepay must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ManualBorrowOrRepay(contextGenerate(), &ManualBorrowRepayRequest{Currency: currency.USDT, Side: "borrow", Amount: 100})
	require.NoError(t, err, "ManualBorrowOrRepay must not error")
	if mockTests {
		assert.Equal(t, &ManualBorrowRepayResponse{Currency: currency.USDT, Side: "borrow", Amount: 100}, result, "ManualBorrowOrRepay should decode every field")
	} else {
		assert.NotNil(t, result, "ManualBorrowOrRepay should return a result")
	}
}

func TestSetAutoRepay(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAutoRepay(contextGenerate(), true)
	require.NoError(t, err, "SetAutoRepay must not error")
	if mockTests {
		assert.Equal(t, &AutoRepayResponse{AutoRepay: true}, result, "SetAutoRepay should decode every field")
	} else {
		assert.NotNil(t, result, "SetAutoRepay should return a result")
	}
}

func TestGetBorrowRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowRepayHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetBorrowRepayHistory(t.Context(), &BorrowRepayHistoryRequest{Before: time.UnixMilli(1725330976644), After: time.UnixMilli(1725330976000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetBorrowRepayHistory(t.Context(), &BorrowRepayHistoryRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBorrowRepayHistory(contextGenerate(), &BorrowRepayHistoryRequest{Currency: currency.USDT, EventType: "auto_repay", After: time.UnixMilli(1725330976644), Before: time.UnixMilli(1725330976000), Limit: 1})
	require.NoError(t, err, "GetBorrowRepayHistory must not error")
	if mockTests {
		exp := []BorrowRepayHistoryResponse{{Currency: currency.USDT, EventType: "auto_repay", Amount: 6764.802661157592, AccumulatedBorrowed: 1520.5, Timestamp: types.Time(time.UnixMilli(1725330976644))}}
		assert.Equal(t, exp, result, "GetBorrowRepayHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetBorrowRepayHistory should return a result")
	}
}

func TestNewPositionBuilder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPositionBuilder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.NewPositionBuilder(t.Context(), &PositionBuilderRequest{SimulatedPositions: make([]SimulatedPosition, 201)})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.NewPositionBuilder(t.Context(), &PositionBuilderRequest{SimulatedAssets: make([]SimulatedAsset, 201)})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.NewPositionBuilder(t.Context(), &PositionBuilderRequest{SimulatedPositions: []SimulatedPosition{{Position: 1, AveragePrice: 1}}})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.NewPositionBuilder(t.Context(), &PositionBuilderRequest{SimulatedAssets: []SimulatedAsset{{Amount: 1}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	// Pins the documented request fields that select multi-currency margin,
	// its leverage and the price volatility: without acctLv OKX applies
	// portfolio margin, where positions stay empty, and without idxVol the
	// before-volatility fields come back empty.
	t.Run("sends the mode parameters", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var bodies []map[string]any
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "the position builder should be a POST")
			assert.Equal(t, "/account/position-builder", r.URL.Path, "the position builder path should be requested")
			raw, err := io.ReadAll(r.Body)
			if !assert.NoError(t, err, "reading the request body should not error") {
				return
			}
			var body map[string]any
			if !assert.NoError(t, json.Unmarshal(raw, &body), "the request body should decode") {
				return
			}
			mu.Lock()
			bodies = append(bodies, body)
			mu.Unlock()
			writeOKXData(t, w, []map[string]any{{"positions": []any{}, "ts": "1724751378980"}})
		}))
		_, err := ex.NewPositionBuilder(t.Context(), &PositionBuilderRequest{
			AccountLevel:       "3",
			Leverage:           10,
			IndexVolatility:    -0.05,
			SimulatedPositions: []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: 10, AveragePrice: 100000, Leverage: 5}},
		})
		require.NoError(t, err, "NewPositionBuilder must not error")
		mu.Lock()
		got := bodies
		mu.Unlock()
		require.Len(t, got, 1, "NewPositionBuilder must send one request")
		body := got[0]
		assert.Equal(t, "3", body["acctLv"], "acctLv should select multi-currency margin")
		assert.Equal(t, "10", body["lever"], "the cross margin leverage should be sent")
		assert.Equal(t, "-0.05", body["idxVol"], "the price volatility should be sent")
		assert.Equal(t, []any{map[string]any{"instId": "BTC-USDT-SWAP", "pos": "10", "avgPx": "100000", "lever": "5"}}, body["simPos"], "the simulated position should carry its average price and leverage")

		unset, err := json.Marshal(&PositionBuilderRequest{IncludeRealPositionsAndEquity: true})
		require.NoError(t, err, "Marshal must not error")
		var defaults map[string]any
		require.NoError(t, json.Unmarshal(unset, &defaults), "Unmarshal must not error")
		for _, key := range []string{"acctLv", "lever", "idxVol"} {
			assert.NotContainsf(t, defaults, key, "an unset %s should be left for OKX to default", key)
		}
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *PositionBuilderRequest
			body string
		}{
			{
				"every parameter",
				&PositionBuilderRequest{
					AccountLevel:                  "3",
					IncludeRealPositionsAndEquity: true,
					Leverage:                      2,
					SimulatedPositions:            []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: -10, AveragePrice: 100000, Leverage: 5}},
					SimulatedAssets:               []SimulatedAsset{{Currency: currency.USDT, Amount: 100}},
					GreeksType:                    "CASH",
					IndexVolatility:               0.01,
				},
				`{"acctLv":"3","inclRealPosAndEq":true,"lever":"2","simPos":[{"instId":"BTC-USDT-SWAP","pos":"-10","avgPx":"100000","lever":"5"}],"simAsset":[{"ccy":"USDT","amt":"100"}],"greeksType":"CASH","idxVol":"0.01"}`,
			},
			{"no optional parameters", &PositionBuilderRequest{}, `{"inclRealPosAndEq":false}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/position-builder", tc.body, []map[string]string{{"ts": "1736936801642"}})
				_, err := ex.NewPositionBuilder(t.Context(), tc.req)
				require.NoError(t, err, "NewPositionBuilder must not error")
			})
		}
	})

	t.Run("risk unit decodes quoted numbers", func(t *testing.T) {
		t.Parallel()
		var detail PositionBuilderResponse
		require.NoError(t, json.Unmarshal([]byte(`{"riskUnitData":[{"delta":"0.1","gamma":"0.01","imr":"500","imrBf":"480","indexUsd":"1000","mmr":"250","mmrBf":"240","mr1":"1164.4109244719994","mr1FinalResult":{"pnl":"-1164.4109244719994","spotShock":"0.12","volShock":"up"},"mr1Scenarios":{"volSame":{"30000":"-0.1"},"volShockDown":{"30000":"-0.2"},"volShockUp":{"30000":"0.3"}},"mr2":"0.5","mr3":"0.6","mr4":"0.7","mr5":"0.8","mr6":"0.9","mr6FinalResult":{"pnl":"-20","spotShock":"0.05"},"mr7":"1.1","mr8":"1.2","mr9":"1.3","upl":"4.5","portfolios":[{"amt":"0.1","avgPx":"97000","delta":"0.5","floatPnl":"1.5","gamma":"0.02","instId":"BTC-USDT-SWAP","instType":"SWAP","isRealPos":true,"markPx":"97000","markPxBf":"96900","notionalUsd":"9703.22","posSide":"long","theta":"-1.5","vega":"2.5"}],"riskUnit":"BTC-USDT-SWAP","theta":"-3","vega":"5"}],"ts":"1724751378980"}`), &detail), "the documented risk unit payload must decode")

		require.Len(t, detail.RiskUnitData, 1, "the risk unit row must decode")
		ru := detail.RiskUnitData[0]
		assert.Equal(t, "1164.4109244719994", ru.MR1.String(), "the documented mr1 should decode as a number")
		assert.Equal(t, "-1164.4109244719994", ru.MR1FinalResult.ProfitAndLoss.String(), "the MR1 worst-case ProfitAndLoss should decode as a number")
		assert.Equal(t, "0.12", ru.MR1FinalResult.SpotShock.String(), "the MR1 spot shock should decode as a number")
		assert.Equal(t, "up", ru.MR1FinalResult.VolatilityShock, "the MR1 volatility shock should decode")
		assert.Equal(t, "-0.2", ru.MR1Scenarios.VolatilityShockDown["30000"].String(), "the MR1 volatility scenarios should decode as numbers")
		assert.Equal(t, "0.5", ru.MR2.String(), "the mr2 stress value should decode as a number")
		assert.Equal(t, "1.3", ru.MR9.String(), "the mr9 stress value should decode as a number")
		assert.Equal(t, "-3", ru.Theta.String(), "the risk unit theta should decode as a number")
		assert.Equal(t, "5", ru.Vega.String(), "the risk unit vega should decode as a number")
		require.Len(t, ru.Portfolios, 1, "the portfolio must decode")
		p := ru.Portfolios[0]
		assert.Equal(t, "9703.22", p.NotionalUSD.String(), "the documented portfolio notionalUsd should decode as a number")
		assert.True(t, p.IsRealPosition, "the documented isRealPos should decode as a bool")
		assert.Equal(t, "-1.5", p.Theta.String(), "the portfolio theta should decode as a number")
		assert.Equal(t, "2.5", p.Vega.String(), "the portfolio vega should decode as a number")
	})

	if mockTests {
		result, err := e.NewPositionBuilder(contextGenerate(), &PositionBuilderRequest{AccountLevel: "4", IncludeRealPositionsAndEquity: true, GreeksType: "BS", IndexVolatility: -0.1})
		require.NoError(t, err, "NewPositionBuilder must not error")
		scenario := map[string]types.Number{"0": 0, "0.12": -1164.4109244719994, "-0.12": 1164.4109244719994}
		exp := &PositionBuilderResponse{
			Equity:                                  -78553.21888979999,
			TotalMaintenanceMarginRequirement:       2946.5480841048,
			TotalInitialMarginRequirement:           9643.45070718144,
			BorrowMaintenanceMarginRequirement:      1571.064377796,
			DerivativesMaintenanceMarginRequirement: 1375.4837063088003,
			MarginRatio:                             -25.95365779811705,
			UnrealisedProfitAndLoss:                 -78653.1728898,
			AccountLeverage:                         -0.1364949794742562,
			Timestamp:                               types.Time(time.UnixMilli(1736936801642)),
			Assets: []PositionBuilderAsset{{
				Currency:                           currency.USDT,
				AvailableEquity:                    -78589.37,
				SpotInUse:                          5,
				BorrowMaintenanceMarginRequirement: 1571.064377796,
				BorrowInitialMarginRequirement:     7855.32188898,
			}},
			RiskUnitData: []PositionBuilderRiskUnit{{
				RiskUnit:                           "BTC",
				MaintenanceMarginRequirementBefore: 1150.5,
				MaintenanceMarginRequirement:       1183.8207318516002,
				InitialMarginRequirementBefore:     1500.5,
				InitialMarginRequirement:           1538.9669514070802,
				UnrealisedProfitAndLoss:            290.49631020000027,
				MR1:                                1164.4109244719994,
				MR2:                                0.5,
				MR3:                                0.6,
				MR4:                                19.4098073796,
				MR5:                                0.7,
				MR6:                                1164.4109244720003,
				MR7:                                43.67206660410001,
				MR8:                                1571.064377796,
				MR9:                                0.8,
				MR1Scenarios:                       PositionBuilderMR1Scenarios{VolatilityShockDown: scenario, VolatilitySame: scenario, VolatilityShockUp: scenario},
				MR1FinalResult:                     PositionBuilderMR1FinalResult{ProfitAndLoss: -1164.4109244719994, SpotShock: 0.12, VolatilityShock: "up"},
				MR6FinalResult:                     PositionBuilderMR6FinalResult{ProfitAndLoss: -2328.8218489440005, SpotShock: 0.24},
				Delta:                              -9704.903689800001,
				Gamma:                              0.0012,
				Theta:                              -3,
				Vega:                               5,
				Portfolios: []PositionBuilderPortfolio{{
					InstrumentID:       "BTC-USDT-SWAP",
					InstrumentType:     instTypeSwap,
					Amount:             -10,
					PositionSide:       positionSideNet,
					AveragePrice:       100000,
					MarkPriceBefore:    97000.5,
					MarkPrice:          97093.7,
					FloatProfitAndLoss: 290.6300000000003,
					NotionalUSD:        9703.22,
					Delta:              -9704.903689800001,
					Gamma:              0.0012,
					Theta:              -1.5,
					Vega:               2.5,
					IsRealPosition:     true,
				}},
			}},
			Positions: []PositionBuilderPosition{{
				InstrumentID:                   "BTC-USDT-SWAP",
				InstrumentType:                 instTypeSwap,
				Amount:                         -10,
				PositionSide:                   positionSideNet,
				AveragePrice:                   100000,
				MarkPriceBefore:                97000.5,
				MarkPrice:                      97093.7,
				FloatProfitAndLoss:             290.63,
				InitialMarginRequirementBefore: 98.5,
				InitialMarginRequirement:       97.09,
				MarginRatio:                    25.5,
				Leverage:                       10,
				NotionalUSD:                    9703.22,
				IsRealPosition:                 true,
			}},
		}
		assert.Equal(t, exp, result, "NewPositionBuilder should decode every field")
		return
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewPositionBuilder(contextGenerate(), &PositionBuilderRequest{
		SimulatedPositions: []SimulatedPosition{
			{Position: -10, InstrumentID: "BTC-USDT-SWAP", AveragePrice: 100000},
			{Position: 10, InstrumentID: "LTC-USDT-SWAP", AveragePrice: 100},
		},
		SimulatedAssets: []SimulatedAsset{{Currency: currency.USDT, Amount: 100}},
		GreeksType:      "CASH",
	})
	require.NoError(t, err, "NewPositionBuilder must not error")
	assert.NotNil(t, result, "NewPositionBuilder should return a result")
}

func TestGetPositionBuilderTrendGraph(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionBuilderTrendGraph(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetPositionBuilderTrendGraph(t.Context(), &PositionBuilderTrendGraphRequest{})
	require.ErrorIs(t, err, errGraphTypeRequired)
	_, err = e.GetPositionBuilderTrendGraph(t.Context(), &PositionBuilderTrendGraphRequest{GraphType: "mmr", SimulatedPositions: []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", AveragePrice: 1}}})
	require.ErrorIs(t, err, errPositionQuantityRequired)

	trendGraph := &PositionBuilderTrendGraphRequest{
		IncludeRealPositionsAndEquity: true,
		SimulatedPositions:            []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: -10, AveragePrice: 100000, Leverage: 5}},
		SimulatedAssets:               []SimulatedAsset{{Currency: currency.USDT, Amount: 100}},
		GraphType:                     "mmr",
		MMRConfig:                     PositionBuilderMMRConfig{AccountLevel: "3", Leverage: 2},
	}
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *PositionBuilderTrendGraphRequest
			body string
		}{
			{"every parameter", trendGraph, `{"inclRealPosAndEq":true,"simPos":[{"instId":"BTC-USDT-SWAP","pos":"-10","avgPx":"100000","lever":"5"}],"simAsset":[{"ccy":"USDT","amt":"100"}],"type":"mmr","mmrConfig":{"acctLv":"3","lever":"2"}}`},
			{"required parameters only", &PositionBuilderTrendGraphRequest{GraphType: "mmr"}, `{"inclRealPosAndEq":false,"type":"mmr","mmrConfig":{}}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/position-builder-graph", tc.body, []map[string]string{{"type": "mmr"}})
				_, err := ex.GetPositionBuilderTrendGraph(t.Context(), tc.req)
				require.NoError(t, err, "GetPositionBuilderTrendGraph must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPositionBuilderTrendGraph(contextGenerate(), trendGraph)
	require.NoError(t, err, "GetPositionBuilderTrendGraph must not error")
	if mockTests {
		exp := &PositionBuilderTrendGraphResponse{
			GraphType: "mmr",
			MMRData: []PositionBuilderMMRData{
				{ShockFactor: -0.94, MaintenanceMarginRequirement: 1415.0254039225917, MaintenanceMarginRatio: -47.45603627655477},
				{ShockFactor: -0.93, MaintenanceMarginRequirement: 1417.732491243024, MaintenanceMarginRatio: -47.436684685735386},
			},
		}
		assert.Equal(t, exp, result, "GetPositionBuilderTrendGraph should decode every field")
	} else {
		assert.NotNil(t, result, "GetPositionBuilderTrendGraph should return a result")
	}
}

func TestSetRiskOffsetAmount(t *testing.T) {
	t.Parallel()
	_, err := e.SetRiskOffsetAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetRiskOffsetAmount(t.Context(), &SetRiskOffsetAmountRequest{ClientSpotInUseAmount: 123})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SetRiskOffsetAmount(t.Context(), &SetRiskOffsetAmountRequest{Currency: currency.USDT})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/account/set-riskOffset-amt", `{"ccy":"USDT","clSpotInUseAmt":"123"}`, []map[string]string{{"ccy": "USDT", "clSpotInUseAmt": "123"}})
		_, err := ex.SetRiskOffsetAmount(t.Context(), &SetRiskOffsetAmountRequest{Currency: currency.USDT, ClientSpotInUseAmount: 123})
		require.NoError(t, err, "SetRiskOffsetAmount must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetRiskOffsetAmount(contextGenerate(), &SetRiskOffsetAmountRequest{Currency: currency.BTC, ClientSpotInUseAmount: 0.5})
	require.NoError(t, err, "SetRiskOffsetAmount must not error")
	if mockTests {
		assert.Equal(t, &RiskOffsetAmountResponse{Currency: currency.BTC, ClientSpotInUseAmount: 0.5}, result, "SetRiskOffsetAmount should decode every field")
	} else {
		assert.NotNil(t, result, "SetRiskOffsetAmount should return a result")
	}
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	_, err := e.GetGreeks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGreeks(contextGenerate(), &GreeksRequest{Currency: currency.BTC})
	require.NoError(t, err, "GetGreeks must not error")
	if mockTests {
		exp := []GreeksResponse{{
			DeltaBS:   1520.75,
			DeltaPA:   0.025,
			GammaBS:   0.00012,
			GammaPA:   0.0000019,
			ThetaBS:   -12.5,
			ThetaPA:   -0.0002,
			VegaBS:    25.5,
			VegaPA:    0.0004,
			Currency:  currency.BTC,
			Timestamp: types.Time(time.UnixMilli(1620282889345)),
		}}
		assert.Equal(t, exp, result, "GetGreeks should decode every field")
	} else {
		assert.NotNil(t, result, "GetGreeks should return a result")
	}
}

func TestGetPMPositionLimitation(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMPositionLimitation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetPMPositionLimitation(t.Context(), &PMPositionLimitationRequest{InstrumentFamilies: []string{mainPair.String()}})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetPMPositionLimitation(t.Context(), &PMPositionLimitationRequest{InstrumentType: "SWAP"})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.GetPMPositionLimitation(t.Context(), &PMPositionLimitationRequest{InstrumentType: "SWAP", InstrumentFamilies: make([]string, 6)})
	require.ErrorIs(t, err, errExceedLimit)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMPositionLimitation(contextGenerate(), &PMPositionLimitationRequest{InstrumentType: "SWAP", InstrumentFamilies: []string{mainPair.String()}})
	require.NoError(t, err, "GetPMPositionLimitation must not error")
	if mockTests {
		exp := []PMLimitationResponse{{Underlying: mainPair.String(), InstrumentFamily: mainPair.String(), MaxSize: 10000, PositionType: "1"}}
		assert.Equal(t, exp, result, "GetPMPositionLimitation should decode every field")
	} else {
		assert.NotNil(t, result, "GetPMPositionLimitation should return a result")
	}
}

func TestActivateOption(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ActivateOption(contextGenerate())
	require.NoError(t, err, "ActivateOption must not error")
	if mockTests {
		assert.Equal(t, &ActivateOptionResponse{Timestamp: types.Time(time.UnixMilli(1600000000000))}, result, "ActivateOption should decode every field")
	} else {
		assert.NotNil(t, result, "ActivateOption should return a result")
	}
}

func TestActivateFeature(t *testing.T) {
	t.Parallel()
	err := e.ActivateFeature(t.Context(), "")
	require.ErrorIs(t, err, errFeatureRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ActivateFeature(contextGenerate(), "1"), "ActivateFeature should not error")
}

func TestSetAutoLoan(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAutoLoan(contextGenerate(), true)
	require.NoError(t, err, "SetAutoLoan must not error")
	if mockTests {
		assert.Equal(t, &AutoLoanResponse{AutoLoan: true}, result, "SetAutoLoan should decode every field")
	} else {
		assert.NotNil(t, result, "SetAutoLoan should return a result")
	}
}

func TestPresetAccountModeSwitch(t *testing.T) {
	t.Parallel()
	_, err := e.PresetAccountModeSwitch(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PresetAccountModeSwitch(t.Context(), &PresetAccountModeSwitchRequest{Leverage: 10})
	require.ErrorIs(t, err, errAccountLevelRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *PresetAccountModeSwitchRequest
			body string
		}{
			{"every parameter", &PresetAccountModeSwitchRequest{AccountLevel: "4", Leverage: 10, RiskOffsetType: "1"}, `{"acctLv":"4","lever":"10","riskOffsetType":"1"}`},
			{"required parameters only", &PresetAccountModeSwitchRequest{AccountLevel: "3"}, `{"acctLv":"3"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/account-level-switch-preset", tc.body, []map[string]string{{"acctLv": tc.req.AccountLevel}})
				_, err := ex.PresetAccountModeSwitch(t.Context(), tc.req)
				require.NoError(t, err, "PresetAccountModeSwitch must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PresetAccountModeSwitch(contextGenerate(), &PresetAccountModeSwitchRequest{AccountLevel: "2", Leverage: 10})
	require.NoError(t, err, "PresetAccountModeSwitch must not error")
	if mockTests {
		exp := &PresetAccountModeSwitchResponse{CurrentAccountLevel: "4", AccountLevel: "2", Leverage: 10, RiskOffsetType: "1"}
		assert.Equal(t, exp, result, "PresetAccountModeSwitch should decode every field")
	} else {
		assert.NotNil(t, result, "PresetAccountModeSwitch should return a result")
	}
}

func TestPrecheckAccountModeSwitch(t *testing.T) {
	t.Parallel()
	_, err := e.PrecheckAccountModeSwitch(t.Context(), "")
	require.ErrorIs(t, err, errAccountLevelRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PrecheckAccountModeSwitch(contextGenerate(), "3")
	require.NoError(t, err, "PrecheckAccountModeSwitch must not error")
	if mockTests {
		exp := &PrecheckAccountModeSwitchResponse{
			StatusCode:          "0",
			CurrentAccountLevel: "4",
			AccountLevel:        "3",
			RiskOffsetType:      "1",
			UnmatchedInfoCheck:  []AccountModeSwitchUnmatchedInfo{{Type: "asset_validation", TotalAsset: 12000.5, PositionIDs: []string{"2005456500916518912"}}},
			Positions: []AccountModeSwitchPosition{
				{PositionID: "2005456500916518912", Leverage: 50},
				{PositionID: "2005456108363218944", Leverage: 50},
			},
			PositionTierCheck: []AccountModeSwitchPositionTierCheck{{InstrumentFamily: mainPair.String(), InstrumentType: instTypeSwap, Position: 100, Leverage: 50, MaxSize: 80}},
			MarginBefore: &AccountModeSwitchMargin{
				AccountAvailableEquity: 77308.89735228613,
				MarginRatio:            4.460069474634038,
				Details:                []AccountModeSwitchMarginDetail{{Currency: currency.BTC, AvailableEquity: 1.25, MarginRatio: 4.5}},
			},
			MarginAfter: &AccountModeSwitchMargin{
				AccountAvailableEquity: 106002.2061970689,
				MarginRatio:            148.1652396878421,
				Details:                []AccountModeSwitchMarginDetail{{Currency: currency.USDT, AvailableEquity: 105000.5, MarginRatio: 150.25}},
			},
		}
		assert.Equal(t, exp, result, "PrecheckAccountModeSwitch should decode every field")
	} else {
		assert.NotNil(t, result, "PrecheckAccountModeSwitch should return a result")
	}
}

func TestSetAccountMode(t *testing.T) { //nolint:tparallel // The recorded switch clears the shared exchange's cached account modes, while the cache subtests use their own
	_, err := e.SetAccountMode(t.Context(), "")
	require.ErrorIs(t, err, errAccountLevelRequired)

	t.Run("recorded", func(t *testing.T) {
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.SetAccountMode(contextGenerate(), "1")
		require.NoError(t, err, "SetAccountMode must not error")
		if mockTests {
			assert.Equal(t, &AccountModeResponse{AccountLevel: "1"}, result, "SetAccountMode should decode every field")
		} else {
			assert.NotNil(t, result, "SetAccountMode should return a result")
		}
	})

	t.Run("refreshes the cached account modes", func(t *testing.T) {
		t.Parallel()
		ctxFor := func(key string) context.Context {
			return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
		}
		marginBuy := &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100}
		swapBuy := &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100}
		for _, tc := range []struct {
			name      string
			setLevel  string // the level each switch requests, 2 when empty
			setReply  string // the set-account-level reply, an error reply when empty
			steps     []string
			expLevels []string // the acctLv each account configuration fetch answers, in turn
			expSent   []string // each step's order body or error
		}{
			{
				name:      "a confirmed switch steers the next order without a fetch",
				setReply:  `[{"acctLv":"2"}]`,
				steps:     []string{"set KEY-1", "margin KEY-1"},
				expSent:   []string{"set", `"ccy":"USDT"`},
				expLevels: nil,
			},
			{
				name:      "another key of the account fetches its level again",
				setReply:  `[{"acctLv":"2"}]`,
				steps:     []string{"margin KEY-1", "set KEY-2", "margin KEY-1"},
				expLevels: []string{"3", "2"},
				expSent:   []string{"no ccy", "set", `"ccy":"USDT"`},
			},
			{
				name:      "the position mode is fetched again",
				setLevel:  "4",
				setReply:  `[{"acctLv":"4"}]`,
				steps:     []string{"swap KEY-1", "set KEY-1", "swap KEY-1"},
				expLevels: []string{"3", "4"},
				expSent:   []string{`"posSide":"long"`, "set", `"posSide":"net"`},
			},
			{
				name:      "a reply without a level confirms none",
				setReply:  `[{"acctLv":""}]`,
				steps:     []string{"set KEY-1", "margin KEY-1"},
				expLevels: []string{"2"},
				expSent:   []string{"set failed", `"ccy":"USDT"`},
			},
			{
				name:      "a failed switch keeps the cache",
				steps:     []string{"margin KEY-1", "set KEY-1", "margin KEY-1"},
				expLevels: []string{"3"},
				expSent:   []string{"no ccy", "set failed", "no ccy"},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var fetches int
				var lastBody string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/account/config":
						mu.Lock()
						fetches++
						n := fetches
						mu.Unlock()
						if n > len(tc.expLevels) {
							t.Errorf("account configuration fetch %d should not be made", n)
							http.Error(w, "unexpected fetch", http.StatusBadRequest)
							return
						}
						mode := positionModeLongShort
						if tc.expLevels[n-1] == "4" {
							mode = positionModeNet
						}
						writeOKXData(t, w, []map[string]string{{"acctLv": tc.expLevels[n-1], "posMode": mode}})
					case "/account/set-account-level":
						w.Header().Set("Content-Type", "application/json")
						if tc.setReply == "" {
							_, _ = w.Write([]byte(`{"code":"59000","msg":"Settings failed","data":[]}`))
							return
						}
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.setReply + `}`))
					case "/trade/order":
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the order body should not error")
						mu.Lock()
						lastBody = string(b)
						mu.Unlock()
						writeOKXData(t, w, orderActionReply)
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				for i, step := range tc.steps {
					kind, key, _ := strings.Cut(step, " ")
					switch kind {
					case "set":
						_, err := ex.SetAccountMode(ctxFor(key), cmp.Or(tc.setLevel, "2"))
						if tc.expSent[i] == "set failed" {
							require.Error(t, err, "SetAccountMode must report the failed switch")
						} else {
							require.NoError(t, err, "SetAccountMode must not error")
						}
						continue
					case "margin", "swap":
						submit := *marginBuy
						if kind == "swap" {
							submit = *swapBuy
						}
						submit.Exchange = ex.Name
						_, err := ex.SubmitOrder(ctxFor(key), &submit)
						require.NoErrorf(t, err, "SubmitOrder must not error at step %d", i)
					}
					mu.Lock()
					body := lastBody
					mu.Unlock()
					if tc.expSent[i] == "no ccy" {
						assert.NotContainsf(t, body, `"ccy"`, "step %d should send no margin currency", i)
					} else {
						assert.Containsf(t, body, tc.expSent[i], "step %d should place for the account's current modes", i)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, len(tc.expLevels), fetches, "the account configuration should be fetched only when no confirmed mode answers")
			})
		}
	})

	t.Run("a later switch waits for an earlier one to complete", func(t *testing.T) {
		t.Parallel()
		testSwitchesSerialise(t, accountLevelSwitch("3"), accountLevelSwitch("2"), map[string]string{}, map[string]uint64{"KEY-1": 2})
	})

	t.Run("a queued switch gives up when its context ends", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			first accountSwitch
		}{
			{"behind an account level switch", accountLevelSwitch("3")},
			{"behind a position mode switch", positionModeSwitch(positionModeLongShort)},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				testQueuedSwitchGivesUp(t, tc.first, accountLevelSwitch("2"))
			})
		}
	})

	t.Run("an unconfirmed level fails and forgets the modes it may have changed", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			reply   string
			timeout bool
			err     error
			keep    bool
		}{
			{name: "no reply", reply: `{"code":"0","msg":"","data":null}`, err: common.ErrNoResponse},
			{name: "a null row", reply: `{"code":"0","msg":"","data":[null]}`, err: common.ErrNoResponse},
			{name: "an empty row", reply: `{"code":"0","msg":"","data":[{}]}`, err: errInvalidAccountLevel},
			{name: "an unparsable level", reply: `{"code":"0","msg":"","data":[{"acctLv":"garbage"}]}`, err: errInvalidAccountLevel},
			{name: "a zero level", reply: `{"code":"0","msg":"","data":[{"acctLv":"0"}]}`, err: errInvalidAccountLevel},
			{name: "an undocumented level", reply: `{"code":"0","msg":"","data":[{"acctLv":"9"}]}`, err: errInvalidAccountLevel},
			{name: "the level after the documented ones", reply: `{"code":"0","msg":"","data":[{"acctLv":"5"}]}`, err: errInvalidAccountLevel},
			{name: "another level", reply: `{"code":"0","msg":"","data":[{"acctLv":"2"}]}`, err: errModeSwitchMismatch},
			{name: "OKX's endpoint timeout", reply: `{"code":"50004","msg":"","data":[]}`, err: ErrorCodes["50004"]},
			{name: "a timed out request", timeout: true, err: context.DeadlineExceeded},
			{name: "a switch OKX refused", reply: `{"code":"59000","msg":"","data":[]}`, err: ErrorCodes["59000"], keep: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				release := make(chan struct{})
				releaseOnce := sync.OnceFunc(func() { close(release) })
				ex := accountModeCaches(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/account/set-account-level" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if tc.timeout {
						<-release
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(tc.reply))
					assert.NoError(t, err, "writing the reply should not error")
				})
				ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "KEY-1", Secret: "secret", ClientID: "passphrase"})
				if tc.timeout {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
				}
				// Registered after the mock server's cleanup, so it runs first and frees a handler still holding its reply
				t.Cleanup(releaseOnce)
				got, err := ex.SetAccountMode(ctx, "4")
				releaseOnce()
				require.ErrorIs(t, err, tc.err, "SetAccountMode must report the unconfirmed switch")
				assert.Nil(t, got, "SetAccountMode should return no confirmation")
				expModes, expLevels, expSwitches := map[string]string{}, map[string]uint64{}, uint64(1)
				if tc.keep {
					expModes, expLevels, expSwitches = map[string]string{"KEY-1": positionModeNet, "KEY-2": positionModeLongShort}, map[string]uint64{"KEY-1": 2, "KEY-2": 3}, 0
				}
				assert.Equal(t, expModes, ex.accountPositionModes, "the position modes the switch may have changed should be forgotten")
				assert.Equal(t, expLevels, ex.accountLevels, "the account levels the switch may have changed should be forgotten")
				assert.Equal(t, expSwitches, ex.accountPositionModeSwitches, "an unconfirmed switch should be counted so an overlapping fetch caches nothing")
			})
		}
	})

	t.Run("acts for one account when the exchange's credentials change", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/set-account-level" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			keys = append(keys, r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"acctLv": "3"}})
		}))
		ex.SetCredentials(primaryAccount())
		_, err := ex.SetAccountMode(&credentialSwitchContext{Context: t.Context(), ex: ex, next: secondaryAccount()}, "3")
		require.NoError(t, err, "SetAccountMode must not error")
		assert.Equal(t, map[string]uint64{"primary-key": 3}, ex.accountLevels, "the confirmed level should be cached for the account it was switched for")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"primary-key"}, keys, "the switch should be sent for the account whose level it caches")
	})
}

func TestSetCollateralAssets(t *testing.T) {
	t.Parallel()
	_, err := e.SetCollateralAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetCollateralAssets(t.Context(), &SetCollateralAssetsRequest{Type: "custom", CollateralEnabled: true})
	require.ErrorIs(t, err, errCurrencyListRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SetCollateralAssetsRequest
			body string
		}{
			{"every parameter", &SetCollateralAssetsRequest{Type: "custom", CollateralEnabled: true, Currencies: []currency.Code{currency.BTC, currency.ETH}}, `{"type":"custom","collateralEnabled":true,"ccyList":["BTC","ETH"]}`},
			{"no optional parameters", &SetCollateralAssetsRequest{}, `{"collateralEnabled":false}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/set-collateral-assets", tc.body, []map[string]any{{"type": tc.req.Type}})
				_, err := ex.SetCollateralAssets(t.Context(), tc.req)
				require.NoError(t, err, "SetCollateralAssets must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetCollateralAssets(contextGenerate(), &SetCollateralAssetsRequest{Type: "all", CollateralEnabled: true})
	require.NoError(t, err, "SetCollateralAssets must not error")
	if mockTests {
		exp := &CollateralAssetsResponse{Type: "all", CollateralEnabled: true, Currencies: []currency.Code{currency.BTC, currency.ETH}}
		assert.Equal(t, exp, result, "SetCollateralAssets should decode every field")
	} else {
		assert.NotNil(t, result, "SetCollateralAssets should return a result")
	}
}

func TestGetCollateralAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetCollateralAssets(t.Context(), &CollateralAssetsRequest{Currencies: make([]currency.Code, 21)})
	require.ErrorIs(t, err, errExceedLimit)

	enabled := true
	for _, tc := range []struct {
		name string
		req  *CollateralAssetsRequest
		exp  []CollateralAssetResponse
	}{
		{"every collateral state", &CollateralAssetsRequest{Currencies: []currency.Code{currency.BTC, currency.ETH}}, []CollateralAssetResponse{{Currency: currency.BTC, CollateralEnabled: true}, {Currency: currency.ETH}}},
		{"collateral only", &CollateralAssetsRequest{Currencies: []currency.Code{currency.BTC}, CollateralEnabled: &enabled}, []CollateralAssetResponse{{Currency: currency.BTC, CollateralEnabled: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetCollateralAssets(contextGenerate(), tc.req)
			require.NoError(t, err, "GetCollateralAssets must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCollateralAssets should decode every field")
			} else {
				assert.NotNil(t, result, "GetCollateralAssets should return a result")
			}
		})
	}
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ResetMMPStatus(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ResetMMPStatus(t.Context(), &ResetMMPStatusRequest{InstrumentType: instTypeOption})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ResetMMPStatusRequest
			body string
		}{
			{"every parameter", &ResetMMPStatusRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"}, `{"instType":"OPTION","instFamily":"BTC-USD"}`},
			{"required parameters only", &ResetMMPStatusRequest{InstrumentFamily: "ETH-USD"}, `{"instFamily":"ETH-USD"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/mmp-reset", tc.body, []map[string]bool{{"result": true}})
				_, err := ex.ResetMMPStatus(t.Context(), tc.req)
				require.NoError(t, err, "ResetMMPStatus must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetMMPStatus(contextGenerate(), &ResetMMPStatusRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"})
	require.NoError(t, err, "ResetMMPStatus must not error")
	if mockTests {
		assert.Equal(t, &MMPStatusResponse{Result: true}, result, "ResetMMPStatus should decode every field")
	} else {
		assert.NotNil(t, result, "ResetMMPStatus should return a result")
	}
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	_, err := e.SetMMP(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetMMP(t.Context(), &SetMMPRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetMMP(t.Context(), &SetMMPRequest{TimeInterval: 5000})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.SetMMP(t.Context(), &SetMMPRequest{InstrumentFamily: "BTC-USD"})
	require.ErrorIs(t, err, errInvalidQuantityLimit)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/account/mmp-config", `{"instFamily":"BTC-USD","timeInterval":"0","frozenInterval":"0","qtyLimit":"5"}`, []map[string]string{{"instFamily": "BTC-USD"}})
		_, err := ex.SetMMP(t.Context(), &SetMMPRequest{InstrumentFamily: "BTC-USD", QuantityLimit: 5})
		require.NoError(t, err, "SetMMP must send zero intervals, which disable MMP and freeze until a reset")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetMMP(contextGenerate(), &SetMMPRequest{InstrumentFamily: "BTC-USD", TimeInterval: 5000, FrozenInterval: 2000, QuantityLimit: 100})
	require.NoError(t, err, "SetMMP must not error")
	if mockTests {
		exp := &SetMMPResponse{InstrumentFamily: "BTC-USD", TimeInterval: 5000, FrozenInterval: 2000, QuantityLimit: 100}
		assert.Equal(t, exp, result, "SetMMP should decode every field")
	} else {
		assert.NotNil(t, result, "SetMMP should return a result")
	}
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMPConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	t.Run("time interval number forms", func(t *testing.T) {
		t.Parallel()
		var quoted MMPConfigResponse
		require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":"5000"}`), &quoted), "the documented quoted timeInterval must decode")
		assert.Equal(t, int64(5000), quoted.TimeInterval.Int64(), "the quoted timeInterval should decode")
		var bare MMPConfigResponse
		require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":5000}`), &bare), "the bare integer form must decode")
		assert.Equal(t, int64(5000), bare.TimeInterval.Int64(), "the bare timeInterval should decode")
		var empty MMPConfigResponse
		require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":""}`), &empty), "the empty form must decode")
		assert.Equal(t, 0.0, empty.TimeInterval.Float64(), "the empty timeInterval should stay the zero number")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMMPConfig(contextGenerate(), &MMPConfigRequest{InstrumentFamily: "ETH-USD"})
	require.NoError(t, err, "GetMMPConfig must not error")
	if mockTests {
		exp := []MMPConfigResponse{{InstrumentFamily: "ETH-USD", MMPFrozen: true, MMPFrozenUntil: 1000, TimeInterval: 5000, FrozenInterval: 2000, QuantityLimit: 10}}
		assert.Equal(t, exp, result, "GetMMPConfig should decode every field")
	} else {
		assert.NotNil(t, result, "GetMMPConfig should return a result")
	}
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	_, err := e.MovePositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	leg := MovePositionLeg{From: MovePositionLegSource{PositionID: "2065471111340792832", Size: 1, Side: "sell"}}
	for _, tc := range []struct {
		name string
		req  *MovePositionsRequest
		err  error
	}{
		{"missing source account", &MovePositionsRequest{ToAccount: "test", Legs: []MovePositionLeg{leg}, ClientID: "test"}, errSourceAccountRequired},
		{"missing destination account", &MovePositionsRequest{FromAccount: "0", Legs: []MovePositionLeg{leg}, ClientID: "test"}, errDestinationAccountRequired},
		{"missing legs", &MovePositionsRequest{FromAccount: "0", ToAccount: "test", ClientID: "test"}, errMissingLegs},
		{"missing position ID", &MovePositionsRequest{FromAccount: "0", ToAccount: "test", Legs: []MovePositionLeg{{From: MovePositionLegSource{Size: 1, Side: "sell"}}}, ClientID: "test"}, errPositionIDRequired},
		{"missing size", &MovePositionsRequest{FromAccount: "0", ToAccount: "test", Legs: []MovePositionLeg{{From: MovePositionLegSource{PositionID: "1", Side: "sell"}}}, ClientID: "test"}, order.ErrAmountIsInvalid},
		{"missing side", &MovePositionsRequest{FromAccount: "0", ToAccount: "test", Legs: []MovePositionLeg{{From: MovePositionLegSource{PositionID: "1", Size: 1}}}, ClientID: "test"}, order.ErrSideIsInvalid},
		{"missing client ID", &MovePositionsRequest{FromAccount: "0", ToAccount: "test", Legs: []MovePositionLeg{leg}}, errClientIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.MovePositions(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "MovePositions should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *MovePositionsRequest
			body string
		}{
			{
				"every parameter",
				&MovePositionsRequest{FromAccount: "sub1", ToAccount: "0", Legs: []MovePositionLeg{{From: MovePositionLegSource{PositionID: "1", Size: 2, Side: "buy"}, To: MovePositionLegDestination{TradeMode: TradeModeIsolated, PositionSide: positionSideLong, Currency: currency.USDT}}}, ClientID: "client1"},
				`{"fromAcct":"sub1","toAcct":"0","legs":[{"from":{"posId":"1","sz":"2","side":"buy"},"to":{"tdMode":"isolated","posSide":"long","ccy":"USDT"}}],"clientId":"client1"}`,
			},
			{
				"required parameters only",
				&MovePositionsRequest{FromAccount: "0", ToAccount: "sub1", Legs: []MovePositionLeg{{From: MovePositionLegSource{PositionID: "1", Size: 2, Side: "sell"}}}, ClientID: "client2"},
				`{"fromAcct":"0","toAcct":"sub1","legs":[{"from":{"posId":"1","sz":"2","side":"sell"},"to":{}}],"clientId":"client2"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/move-positions", tc.body, []map[string]string{{"clientId": tc.req.ClientID}})
				_, err := ex.MovePositions(t.Context(), tc.req)
				require.NoError(t, err, "MovePositions must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.MovePositions(contextGenerate(), &MovePositionsRequest{
		FromAccount: "0",
		ToAccount:   "test",
		Legs: []MovePositionLeg{
			{From: MovePositionLegSource{PositionID: "2065471111340792832", Size: 1, Side: "sell"}, To: MovePositionLegDestination{TradeMode: TradeModeCross, PositionSide: positionSideNet, Currency: currency.BTC}},
			{From: MovePositionLegSource{PositionID: "2063111180412153856", Size: 1, Side: "sell"}},
		},
		ClientID: "test",
	})
	require.NoError(t, err, "MovePositions must not error")
	if mockTests {
		const failure = "Order failed. Insufficient BTC margin in account"
		exp := &MovePositionsResponse{
			ClientID:     "test",
			BlockTradeID: "2065832911119076864",
			State:        "failed",
			Timestamp:    types.Time(time.UnixMilli(1734069018526)),
			FromAccount:  "0",
			ToAccount:    "test",
			Legs: []MovePositionLegResult{
				{
					From: MovePositionLegSourceResult{PositionID: "2065471111340792832", InstrumentID: "BTC-USD-SWAP", Price: 100042.7, Side: "sell", Size: 1, StatusCode: "0"},
					To:   MovePositionLegDestinationResult{InstrumentID: "BTC-USD-SWAP", Price: 100042.7, Side: "buy", Size: 1, TradeMode: TradeModeCross, PositionSide: positionSideNet, Currency: currency.BTC, StatusCode: "0"},
				},
				{
					From: MovePositionLegSourceResult{PositionID: "2063111180412153856", InstrumentID: "BTC-USDT-SWAP", Price: 100008.1, Side: "sell", Size: 1, StatusCode: "51008", StatusMessage: failure},
					To:   MovePositionLegDestinationResult{InstrumentID: "BTC-USDT-SWAP", Price: 100008.1, Side: "buy", Size: 1, TradeMode: TradeModeCross, PositionSide: positionSideNet, StatusCode: "51008", StatusMessage: failure},
				},
			},
		}
		assert.Equal(t, exp, result, "MovePositions should decode every field")
	} else {
		assert.NotNil(t, result, "MovePositions should return a result")
	}
}

func TestGetMovePositionsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMovePositionsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetMovePositionsHistory(t.Context(), &MovePositionsHistoryRequest{BeginTime: time.UnixMilli(1734085800000), EndTime: time.UnixMilli(1734085700000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetMovePositionsHistory(t.Context(), &MovePositionsHistoryRequest{Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMovePositionsHistory(contextGenerate(), &MovePositionsHistoryRequest{
		BlockTradeID: "2066393411110139648",
		ClientID:     "test",
		BeginTime:    time.UnixMilli(1734085700000),
		EndTime:      time.UnixMilli(1734085800000),
		Limit:        1,
		State:        "filled",
	})
	require.NoError(t, err, "GetMovePositionsHistory must not error")
	if mockTests {
		exp := []MovePositionsHistoryResponse{{
			ClientID:     "test",
			BlockTradeID: "2066393411110139648",
			State:        "filled",
			Timestamp:    types.Time(time.UnixMilli(1734085725000)),
			FromAccount:  "0",
			ToAccount:    "test",
			Legs: []MovePositionHistoryLeg{
				{
					From: MovePositionHistorySource{PositionID: "2065477911110792832", InstrumentID: "BTC-USD-SWAP", Price: 100123.8, Side: "sell", Size: 1},
					To:   MovePositionHistoryDestination{InstrumentID: "BTC-USD-SWAP", Price: 100123.8, Side: "buy", Size: 1, TradeMode: TradeModeCross, PositionSide: positionSideNet, Currency: currency.BTC},
				},
				{
					From: MovePositionHistorySource{PositionID: "2063533111112153856", InstrumentID: "BTC-USDT-SWAP", Price: 100078.7, Side: "sell", Size: 1},
					To:   MovePositionHistoryDestination{InstrumentID: "BTC-USDT-SWAP", Price: 100078.7, Side: "buy", Size: 1, TradeMode: TradeModeCross, PositionSide: positionSideNet},
				},
			},
		}}
		assert.Equal(t, exp, result, "GetMovePositionsHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetMovePositionsHistory should return a result")
	}
}

func TestSetAutoEarn(t *testing.T) {
	t.Parallel()
	_, err := e.SetAutoEarn(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetAutoEarn(t.Context(), &SetAutoEarnRequest{Action: "turn_on"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SetAutoEarn(t.Context(), &SetAutoEarnRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errAutoEarnActionRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SetAutoEarnRequest
			body string
		}{
			{"every parameter", &SetAutoEarnRequest{EarnType: "0", Currency: currency.BTC, Action: "turn_on", APR: 0.01}, `{"earnType":"0","ccy":"BTC","action":"turn_on","apr":"0.01"}`},
			{"required parameters only", &SetAutoEarnRequest{Currency: currency.USDG, Action: "turn_off"}, `{"ccy":"USDG","action":"turn_off"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/set-auto-earn", tc.body, []map[string]string{{"action": tc.req.Action}})
				_, err := ex.SetAutoEarn(t.Context(), tc.req)
				require.NoError(t, err, "SetAutoEarn must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAutoEarn(contextGenerate(), &SetAutoEarnRequest{EarnType: "0", Currency: currency.BTC, Action: "turn_on", APR: 0.01})
	require.NoError(t, err, "SetAutoEarn must not error")
	if mockTests {
		assert.Equal(t, &AutoEarnResponse{EarnType: "0", Currency: currency.BTC, Action: "turn_on", APR: 0.01}, result, "SetAutoEarn should decode every field")
	} else {
		assert.NotNil(t, result, "SetAutoEarn should return a result")
	}
}

func TestSetSettleCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.SetSettleCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetSettleCurrency(contextGenerate(), currency.USDC)
	require.NoError(t, err, "SetSettleCurrency must not error")
	if mockTests {
		assert.Equal(t, &SettleCurrencyResponse{SettlementCurrency: currency.USDC}, result, "SetSettleCurrency should decode every field")
	} else {
		assert.NotNil(t, result, "SetSettleCurrency should return a result")
	}
}

func TestSetTradingConfig(t *testing.T) {
	t.Parallel()
	_, err := e.SetTradingConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.SetTradingConfig(t.Context(), &SetTradingConfigRequest{StrategyType: "1"})
	require.ErrorIs(t, err, errTradingConfigTypeRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SetTradingConfigRequest
			body string
		}{
			{"every parameter", &SetTradingConfigRequest{Type: "stgyType", StrategyType: "0"}, `{"type":"stgyType","stgyType":"0"}`},
			{"required parameters only", &SetTradingConfigRequest{Type: "stgyType"}, `{"type":"stgyType"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/account/set-trading-config", tc.body, []map[string]string{{"type": tc.req.Type}})
				_, err := ex.SetTradingConfig(t.Context(), tc.req)
				require.NoError(t, err, "SetTradingConfig must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetTradingConfig(contextGenerate(), &SetTradingConfigRequest{Type: "stgyType", StrategyType: "1"})
	require.NoError(t, err, "SetTradingConfig must not error")
	if mockTests {
		assert.Equal(t, &TradingConfigResponse{Type: "stgyType", StrategyType: "1"}, result, "SetTradingConfig should decode every field")
	} else {
		assert.NotNil(t, result, "SetTradingConfig should return a result")
	}
}

func TestPrecheckSetDeltaNeutral(t *testing.T) {
	t.Parallel()
	_, err := e.PrecheckSetDeltaNeutral(t.Context(), "")
	require.ErrorIs(t, err, errStrategyTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PrecheckSetDeltaNeutral(contextGenerate(), "1")
	require.NoError(t, err, "PrecheckSetDeltaNeutral must not error")
	if mockTests {
		exp := &PrecheckSetDeltaNeutralResponse{UnmatchedInfoCheck: []DeltaNeutralUnmatchedInfo{
			{Type: "delta_risk", DeltaLeverage: 2.5, OrderIDs: []string{}, PositionIDs: []string{}},
			{Type: "pending_orders_options", OrderIDs: []string{"680800019749904384"}, PositionIDs: []string{}},
			{Type: "isolated_margin", OrderIDs: []string{}, PositionIDs: []string{"123", "124", "125"}},
		}}
		assert.Equal(t, exp, result, "PrecheckSetDeltaNeutral should decode every field")
	} else {
		assert.NotNil(t, result, "PrecheckSetDeltaNeutral should return a result")
	}
}

func TestAdjustDemoAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustDemoAccountBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	for _, tc := range []struct {
		name string
		req  *AdjustDemoAccountBalanceRequest
		err  error
	}{
		{"missing type", &AdjustDemoAccountBalanceRequest{Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC, Amount: 1}}}, errAdjustmentTypeRequired},
		{"missing adjustments", &AdjustDemoAccountBalanceRequest{Type: "increase"}, errAdjustmentsRequired},
		{"missing currency", &AdjustDemoAccountBalanceRequest{Type: "increase", Adjustments: []DemoBalanceAdjustment{{Amount: 1}}}, currency.ErrCurrencyCodeEmpty},
		{"missing amount", &AdjustDemoAccountBalanceRequest{Type: "reduce", Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC}}}, limits.ErrAmountBelowMin},
		{"duplicate currency", &AdjustDemoAccountBalanceRequest{Type: "reduce", Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC, Amount: 1}, {Currency: currency.NewCode("btc"), Amount: 2}}}, errDuplicateCurrency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AdjustDemoAccountBalance(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "AdjustDemoAccountBalance should return the expected error")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/account/demo-adjust-balance", `{"type":"reduce","adjustments":[{"ccy":"OKB","amt":"25"}]}`, []map[string]string{{"remainCnt": "3"}})
		_, err := ex.AdjustDemoAccountBalance(t.Context(), &AdjustDemoAccountBalanceRequest{Type: "reduce", Adjustments: []DemoBalanceAdjustment{{Currency: currency.OKB, Amount: 25}}})
		require.NoError(t, err, "AdjustDemoAccountBalance must not error")
	})

	t.Run("leaves the day's quota of increases to OKX", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var received []string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Type string `json:"type"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "the adjustment should decode")
			mu.Lock()
			spent := slices.Contains(received, "increase")
			received = append(received, body.Type)
			mu.Unlock()
			if spent && body.Type == "increase" {
				_, err := w.Write([]byte(`{"code":"50011","msg":"Rate limit reached. Please refer to API documentation and throttle requests accordingly.","data":[]}`))
				assert.NoError(t, err, "writing the reply should not error")
				return
			}
			writeOKXData(t, w, []map[string]string{{"remainCnt": "0", "totalCnt": "3"}})
		}))
		pools := chargeToPools(t, ex)
		increase := &AdjustDemoAccountBalanceRequest{Type: "increase", Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC, Amount: 0.5}}}
		_, err := ex.AdjustDemoAccountBalance(chargeContext(t), increase)
		require.NoError(t, err, "AdjustDemoAccountBalance must send the day's last increase")
		_, err = ex.AdjustDemoAccountBalance(chargeContext(t), increase)
		assert.ErrorIs(t, err, ErrorCodes["50011"], "AdjustDemoAccountBalance should report OKX's refusal of an increase past the day's quota")
		_, err = ex.AdjustDemoAccountBalance(chargeContext(t), &AdjustDemoAccountBalanceRequest{Type: "reduce", Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC, Amount: 0.5}}})
		require.NoError(t, err, "AdjustDemoAccountBalance must send a reduction")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"increase", "increase", "reduce"}, received, "AdjustDemoAccountBalance should send every adjustment, leaving the day's quota of increases to OKX")
		assert.Equal(t, map[request.EndpointLimit]float64{increaseDemoAccountBalanceEPL: 2, reduceDemoAccountBalanceEPL: 1}, takenTokens(pools), "each adjustment should be charged to its own endpoint limit alone")
	})

	if !mockTests {
		if !useTestNet {
			t.Skip("demo trading only: set useTestNet to adjust a demo account's balances")
		}
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AdjustDemoAccountBalance(contextGenerate(), &AdjustDemoAccountBalanceRequest{Type: "increase", Adjustments: []DemoBalanceAdjustment{{Currency: currency.BTC, Amount: 0.5}, {Currency: currency.USDT, Amount: 3000}}})
	require.NoError(t, err, "AdjustDemoAccountBalance must not error")
	if mockTests {
		exp := &AdjustDemoAccountBalanceResponse{
			RemainingCount: 2,
			TotalCount:     3,
			Details: []DemoBalanceAdjustmentResult{
				{Currency: currency.BTC, Amount: 0.5, Balance: 1.5},
				{Currency: currency.USDT, Amount: 3000, Balance: 13000},
			},
		}
		assert.Equal(t, exp, result, "AdjustDemoAccountBalance should decode every field")
	} else {
		assert.NotNil(t, result, "AdjustDemoAccountBalance should return a result")
	}
}

func TestGetGLPTodayPerformance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGLPTodayPerformance(contextGenerate())
	require.NoError(t, err, "GetGLPTodayPerformance must not error")
	if mockTests {
		pair := func(maker, taker types.Number) GLPMakerTaker { return GLPMakerTaker{Maker: maker, Taker: taker} }
		exp := &GLPTodayPerformanceResponse{
			DataReady: true,
			DataDate:  "2026-07-13",
			Account:   GLPAccount{MasterAccountID: "832545488879789797", CombinedAccountIDs: []string{"832545488879789798"}},
			Programs: []GLPProgramPerformance{{
				Program:               "SPOT",
				MarketMakerBusinessID: "1",
				EnrollmentStatus:      "ENROLLED",
				MarketMakerLevelID:    "42",
				EnrolledTierDisplay:   "Tier 1 Class A",
				QualifyingPool:        "TYPE_A",
				QualifyingRows:        []string{"TOTAL"},
				Daily: GLPPerformance{
					Volume: GLPPoolBreakdown{TypeA: pair(1000000, 1000000), TypeBTotal: pair(1000000, 1000000), TradFiDoubled: pair(1000000, 1000000), Total: pair(2000000, 2000000)},
					Share:  GLPPoolBreakdown{TypeA: pair(0.0012, 0.001), TypeBAdjusted: pair(0.0008, 0.0007), Total: pair(0.001, 0.0009)},
				},
				MonthToDate: GLPMonthToDatePerformance{
					Volume:            GLPPoolBreakdown{TypeA: pair(30000000, 30000000), TypeBTotal: pair(30000000, 30000000), TradFiDoubled: pair(30000000, 30000000), Total: pair(60000000, 60000000)},
					Share:             GLPPoolBreakdown{TypeA: pair(0.0011, 0.0009), TypeBAdjusted: pair(0.0007, 0.0006), Total: pair(0.0009, 0.0008)},
					MonthToDateStatus: "QUALIFIED",
					QualifyingShare:   pair(0.0011, 0.0009),
				},
			}},
		}
		assert.Equal(t, exp, result, "GetGLPTodayPerformance should decode every field")
	} else {
		assert.NotNil(t, result, "GetGLPTodayPerformance should return a result")
	}
}

func TestGetGLPHistoricalPerformance(t *testing.T) {
	t.Parallel()
	_, err := e.GetGLPHistoricalPerformance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetGLPHistoricalPerformance(t.Context(), &GLPHistoricalPerformanceRequest{Limit: 2})
	require.ErrorIs(t, err, errProgramRequired)
	_, err = e.GetGLPHistoricalPerformance(t.Context(), &GLPHistoricalPerformanceRequest{Program: "SPOT", BeginTime: time.UnixMilli(1783872000000), EndTime: time.UnixMilli(1783785600000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetGLPHistoricalPerformance(t.Context(), &GLPHistoricalPerformanceRequest{Program: "SPOT", Limit: 101})
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGLPHistoricalPerformance(contextGenerate(), &GLPHistoricalPerformanceRequest{Program: "SPOT", BeginTime: time.UnixMilli(1783785600000), EndTime: time.UnixMilli(1783872000000), Limit: 2})
	require.NoError(t, err, "GetGLPHistoricalPerformance must not error")
	if mockTests {
		pair := func(maker, taker types.Number) GLPMakerTaker { return GLPMakerTaker{Maker: maker, Taker: taker} }
		exp := []GLPHistoricalPerformanceResponse{
			{
				Date:   "2026-07-13",
				Volume: GLPPoolBreakdown{TypeA: pair(1000000, 1000000), TypeBTotal: pair(1000000, 1000000), TradFiDoubled: pair(1000000, 1000000), Total: pair(2000000, 2000000)},
				Share:  GLPPoolBreakdown{TypeA: pair(0.0012, 0.001), TypeBAdjusted: pair(0.0008, 0.0007), Total: pair(0.001, 0.0009)},
			},
			{
				Date:   "2026-07-12",
				Volume: GLPPoolBreakdown{TypeA: pair(950000, 980000), TypeBTotal: pair(850000, 900000), TradFiDoubled: pair(800000, 820000), Total: pair(1800000, 1880000)},
				Share:  GLPPoolBreakdown{TypeA: pair(0.0011, 0.0009), TypeBAdjusted: pair(0.0007, 0.0006), Total: pair(0.0009, 0.0008)},
			},
		}
		assert.Equal(t, exp, result, "GetGLPHistoricalPerformance should decode every field")
	} else {
		assert.NotNil(t, result, "GetGLPHistoricalPerformance should return a result")
	}
}

func TestGetGLPV2Overview(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGLPV2Overview(contextGenerate())
	require.NoError(t, err, "GetGLPV2Overview must not error")
	if mockTests {
		exp := &GLPOverviewResponse{Programs: []GLPOverviewProgram{{
			BusinessID:          "16",
			Program:             "GLP_B",
			Side:                "MAKER",
			Instrument:          "PERP",
			CurrentTier:         "Tier 2",
			EstimatedNextTier:   "Tier 2",
			EstimatedNextStatus: "qualify",
			GracePeriodEndTime:  types.Time(time.UnixMilli(1727308800000)),
			Requirements:        []GLPTierRequirement{{Tier: "Tier 1", Share: 0.005}, {Tier: "Tier 2", Share: 0.003}},
			Performance: []GLPOverviewPerformance{
				{Date: types.Time(time.UnixMilli(1726531200000)), AssessmentPeriod: "PREVIOUS", Share: 0.0035, Volume: 140000},
				{Date: types.Time(time.UnixMilli(1726876800000)), AssessmentPeriod: "CURRENT", Share: 0.004, Volume: 200},
			},
		}}}
		assert.Equal(t, exp, result, "GetGLPV2Overview should decode every field")
	} else {
		assert.NotNil(t, result, "GetGLPV2Overview should return a result")
	}
}

func TestGetGLPV2AssessmentDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetGLPV2AssessmentDetails(t.Context(), "")
	require.ErrorIs(t, err, errBusinessIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGLPV2AssessmentDetails(contextGenerate(), "16")
	require.NoError(t, err, "GetGLPV2AssessmentDetails must not error")
	if mockTests {
		exp := &GLPAssessmentDetailsResponse{
			AssessmentStartTime: types.Time(time.UnixMilli(1726790400000)),
			AssessmentEndTime:   types.Time(time.UnixMilli(1727308800000)),
			Performance: []GLPAssessmentPerformance{
				{Date: types.Time(time.UnixMilli(1726790400000)), OriginalVolume: 100, OriginalShare: 0.004, AdjustedVolume: 200, AdjustedShare: 0.004, AccumulatedOriginalVolume: 100, AccumulatedOriginalShare: 0.004, AccumulatedAdjustedVolume: 200, AccumulatedAdjustedShare: 0.004},
				{Date: types.Time(time.UnixMilli(1726876800000)), OriginalVolume: 200, OriginalShare: 0.004, AdjustedVolume: 400, AdjustedShare: 0.004, AccumulatedOriginalVolume: 300, AccumulatedOriginalShare: 0.004, AccumulatedAdjustedVolume: 600, AccumulatedAdjustedShare: 0.004},
			},
		}
		assert.Equal(t, exp, result, "GetGLPV2AssessmentDetails should decode every field")
	} else {
		assert.NotNil(t, result, "GetGLPV2AssessmentDetails should return a result")
	}
}

func TestGetGLPV2PerformanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetGLPV2PerformanceHistory(t.Context(), "")
	require.ErrorIs(t, err, errBusinessIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetGLPV2PerformanceHistory(contextGenerate(), "16")
	require.NoError(t, err, "GetGLPV2PerformanceHistory must not error")
	if mockTests {
		exp := &GLPPerformanceHistoryResponse{Performance: []GLPAssessmentResult{{
			AssessmentStartTime:       types.Time(time.UnixMilli(1726185600000)),
			AssessmentEndTime:         types.Time(time.UnixMilli(1726704000000)),
			Tier:                      "Tier 3",
			NextTier:                  "Tier 2",
			AccumulatedOriginalVolume: 100000,
			AccumulatedOriginalShare:  0.004,
			AccumulatedAdjustedVolume: 200000,
			AccumulatedAdjustedShare:  0.004,
			AssessmentResult:          "upgrade",
		}}}
		assert.Equal(t, exp, result, "GetGLPV2PerformanceHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetGLPV2PerformanceHistory should return a result")
	}
}

func TestValidateEntryCount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		count int
		err   error
	}{
		{"empty", 0, nil},
		{"one", 1, nil},
		{"at the maximum", 5, nil},
		{"above the maximum", 6, errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateEntryCount("instrument IDs", tc.count, 5), tc.err, "validateEntryCount should return the expected error")
		})
	}
}

func TestValidateTimeWindow(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1734085700000), time.UnixMilli(1734085800000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{"unset", time.Time{}, time.Time{}, nil},
		{"start only", start, time.Time{}, nil},
		{"end only", time.Time{}, end, nil},
		{"ordered", start, end, nil},
		{"equal", start, start, nil},
		{"start after end", end, start, common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateTimeWindow(tc.start, tc.end), tc.err, "validateTimeWindow should return the expected error")
		})
	}
}

func TestValidateSimulations(t *testing.T) {
	t.Parallel()
	position := SimulatedPosition{InstrumentID: "BTC-USDT-SWAP", Position: -10, AveragePrice: 100000}
	simulatedAsset := SimulatedAsset{Currency: currency.USDT, Amount: -100}
	for _, tc := range []struct {
		name      string
		positions []SimulatedPosition
		assets    []SimulatedAsset
		err       error
	}{
		{"none", nil, nil, nil},
		{"valid", []SimulatedPosition{position}, []SimulatedAsset{simulatedAsset}, nil},
		{"missing instrument ID", []SimulatedPosition{position, {Position: 1, AveragePrice: 1}}, nil, errMissingInstrumentID},
		{"missing position quantity", []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", AveragePrice: 1}}, nil, errPositionQuantityRequired},
		{"missing average price", []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: 1}}, nil, errAveragePriceRequired},
		{"negative average price", []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: 1, AveragePrice: -1}}, nil, errAveragePriceRequired},
		{"missing asset currency", nil, []SimulatedAsset{simulatedAsset, {Amount: 1}}, currency.ErrCurrencyCodeEmpty},
		{"missing asset amount", nil, []SimulatedAsset{{Currency: currency.USDT}}, limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateSimulations(tc.positions, tc.assets), tc.err, "validateSimulations should return the expected error")
		})
	}
}

func TestJoinCurrencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		codes []currency.Code
		exp   string
	}{
		{"empty", nil, ""},
		{"single", []currency.Code{currency.BTC}, "BTC"},
		{"lower case codes are sent upper case", []currency.Code{currency.BTC.Lower(), currency.NewCode("eth"), currency.USDT}, "BTC,ETH,USDT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, joinCurrencies(tc.codes), "joinCurrencies should join the upper case codes")
		})
	}
}
