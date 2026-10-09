package okx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TestDocsPinnedRequestParameters pins the wire parameter names and endpoint
// routes against the OKX v5 documentation. OKX silently ignores unknown query
// parameters, so a misnamed filter fails quietly: every case asserts the
// documented name is sent and the previously sent name stays absent. Cases
// with a verify hook additionally pin that the documented response fields
// decode.
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
		"/account/positions-history":                     `{"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","mgnMode":"cross","type":"2","cTime":"1619776200285","uTime":"1619776200285","lever":"3","margin":"100","optVal":"0.1","usdPx":"42000.5","bePx":"41000","pnl":"5","closeOrderAlgo":[{"algoId":"123","slTriggerPx":"38000","slTriggerPxType":"last","tpTriggerPx":"50000","tpTriggerPxType":"last","tpOrdPx":"-1","slOrdPx":"39000","closeFraction":"0.5"}]}]}`,
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
	var gridSubOrders []GridSubOrderData
	var indexComponents *IndexComponent
	var discountRates []DiscountRate
	var pmPositionLimitations []PMLimitationResponse
	var convertCurrencies []ConvertCurrency
	var quoteProducts []QuoteProduct
	var unrealisedProfitSharing []ProfitSharingItem
	var accountRiskState []AccountRiskState
	var signalBotEventHistory []SignalBotEventHistory
	var accountPositions []AccountPosition
	var singleFundingRate *FundingRateResponse
	var gridAlgoDetails *GridAlgoOrderResponse
	var leadingPositions []PositionInfo
	var withdrawalHistory []WithdrawalHistoryResponse
	var subaccountList []SubaccountInfo
	var mmpConfigs []MMPConfigDetail
	var depositWithdrawStatuses []DepositWithdrawStatus
	var positionBuilder *PositionBuilderDetail

	for _, tc := range []struct {
		name   string
		call   func() error
		path   string
		params map[string]string
		absent []string
		verify func(t *testing.T)
	}{
		{
			name: "RFQs send clRfqId",
			call: func() error {
				_, err := e.GetRFQs(t.Context(), &RFQsRequest{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/rfqs",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "Quotes send clRfqId",
			call: func() error {
				_, err := e.GetQuotes(t.Context(), &QuotesRequest{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/quotes",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId"},
		},
		{
			name: "RFQ trades send clRfqId without state",
			call: func() error {
				_, err := e.GetRequestForQuoteTrades(t.Context(), &RequestForQuoteTradesRequest{ClientRFQID: "rfq-client-1"})
				return err
			},
			path:   "/rfq/trades",
			params: map[string]string{"clRfqId": "rfq-client-1"},
			absent: []string{"clRFQId", "state"},
		},
		{
			name: "Spread order books size uses sz",
			call: func() error {
				_, err := e.GetPublicSpreadOrderBooks(t.Context(), "BTC-USDT_BTC-USDT-SWAP", 50)
				return err
			},
			path:   "/sprd/books",
			params: map[string]string{"sz": "50"},
			absent: []string{"size"},
		},
		{
			name: "Subaccount bills filter uses subAcct",
			call: func() error {
				_, err := e.HistoryOfSubaccountTransfer(t.Context(), currency.BTC, "", "sub-one", time.Time{}, time.Time{}, 7)
				return err
			},
			path:   "/asset/subaccount/bills",
			params: map[string]string{"subAcct": "sub-one", "limit": "7"},
			absent: []string{"subacct", "setAcct"},
		},
		{
			name: "Custody subaccount list filter uses subAcct",
			call: func() error {
				_, err := e.GetCustodyTradingSubaccountList(t.Context(), "sub-one")
				return err
			},
			path:   "/users/entrust-subaccount-list",
			params: map[string]string{"subAcct": "sub-one"},
			absent: []string{"setAcct"},
		},
		{
			name: "Position tiers filter uses singular tier",
			call: func() error {
				_, err := e.GetPositionTiers(t.Context(), instTypeMargin, "cross", "", "", "BTC-USDT", "1", currency.EMPTYCODE)
				return err
			},
			path:   "/public/position-tiers",
			params: map[string]string{"tier": "1"},
			absent: []string{"tiers"},
		},
		{
			name: "Daily lead trader ProfitAndLoss hits the daily endpoint",
			call: func() error {
				_, err := e.GetDailyLeadTraderProfitAndLoss(t.Context(), "SWAP", "trader-1", "2")
				return err
			},
			path:   "/copytrading/public-pnl",
			params: map[string]string{"uniqueCode": "trader-1", "lastDays": "2"},
		},
		{
			name: "Lead trader currency preferences drop lastDays",
			call: func() error {
				_, err := e.GetLeadTraderCurrencyPreferences(t.Context(), "SWAP", "trader-1")
				return err
			},
			path:   "/copytrading/public-preference-currency",
			params: map[string]string{"uniqueCode": "trader-1"},
			absent: []string{"lastDays"},
		},
		{
			name: "Trade fee drops the response-only ruleType",
			call: func() error {
				_, err := e.GetTradeFee(t.Context(), instTypeSpot, "BTC-USDT", "", "")
				return err
			},
			path:   "/account/trade-fee",
			params: map[string]string{"instType": "SPOT", "instId": "BTC-USDT"},
			absent: []string{"ruleType"},
		},
		{
			name: "Max buy or sell amount drops unSpotOffset",
			call: func() error {
				_, err := e.GetMaximumBuySellAmountOrOpenAmount(t.Context(), currency.BTC, "BTC-USDT", "cash", "", 5)
				return err
			},
			path:   "/account/max-size",
			params: map[string]string{"instId": "BTC-USDT"},
			absent: []string{"unSpotOffset"},
		},
		{
			name: "Max available amount drops quickMgnType and upSpotOffset",
			call: func() error {
				_, err := e.GetMaximumAvailableTradableAmount(t.Context(), currency.BTC, "BTC-USDT", "cash", false, 5)
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
				pendingAlgoOrders, err = e.GetAlgoOrderList(t.Context(), "conditional", "", "test-algo-client-id", "", "", time.Time{}, time.Time{}, 1)
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
				require.Len(t, pendingAlgoOrders[0].TriggerParams, 1, "the smart-iceberg trigger parameters must decode")
				assert.Equal(t, 90000.0, pendingAlgoOrders[0].TriggerParams[0].TriggerPrice.Float64(), "the trigger price should decode")
				assert.Equal(t, "680800019749904385", pendingAlgoOrders[0].LinkedOrder.OrderID, "the linked OCO take-profit order ID should decode")
				assert.Equal(t, 62916.5, pendingAlgoOrders[0].LastPrice.Float64(), "the last filled price should decode")
				assert.Equal(t, "true", pendingAlgoOrders[0].ReduceOnly, "the documented reduceOnly string should decode")
			},
		},
		{
			name: "Recurring buy order list drops state",
			call: func() error {
				_, err := e.GetRecurringBuyOrderList(t.Context(), "", time.Time{}, time.Time{}, 30)
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
				recurringOrderDetails, err = e.GetRecurringOrderDetails(t.Context(), "560473220642766848")
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
				_, err := e.GetTakerVolume(t.Context(), currency.BTC, instTypeSpot, time.Time{}, time.Time{}, kline.OneDay)
				return err
			},
			path:   "/rubik/stat/taker-volume",
			params: map[string]string{"instType": "SPOT"},
			absent: []string{"instFamily"},
		},
		{
			name: "Deposit order detail sends ordId",
			call: func() error {
				_, err := e.GetDepositOrderDetail(t.Context(), "12345")
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
				oneClickRepayCurrencyList, err = e.GetOneClickRepayCurrencyList(t.Context(), "")
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
				gridSubOrders, err = e.GetGridAlgoSubOrders(t.Context(), AlgoOrdTypeGrid, "12345", "live", "", "", "", 0)
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
				assert.Equal(t, 12.5, gridSubOrders[0].ProfitAndLoss.Float64(), "the sub order PnL should decode")
				assert.Equal(t, "USDT", gridSubOrders[0].Currency.String(), "the documented ccy should decode")
				assert.Equal(t, 0.01, gridSubOrders[0].Rebate.Float64(), "the documented rebate should decode")
			},
		},
		{
			name: "Filled grid sub orders request the documented state",
			call: func() error {
				var err error
				gridSubOrders, err = e.GetGridAlgoSubOrders(t.Context(), AlgoOrdTypeGrid, "12345", "filled", "", "", "", 20)
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
				discountRates, err = e.GetDiscountRateAndInterestFreeQuota(t.Context(), currency.BTC, 0)
				return err
			},
			path: "/public/discount-rate-interest-free-quota",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, discountRates, 1, "the discount rate row must decode")
				require.Len(t, discountRates[0].Details, 1, "the documented discount tier details must decode")
				assert.Equal(t, 0.98, discountRates[0].Details[0].DiscountRate.Float64(), "the tier discount rate should decode")
				assert.Equal(t, "1", discountRates[0].Details[0].Tiers, "the tier number should decode")
			},
		},
		{
			name: "Portfolio margin position tiers decode posType",
			call: func() error {
				var err error
				pmPositionLimitations, err = e.GetPMPositionLimitation(t.Context(), "SWAP", "BTC-USD", "")
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
				unrealisedProfitSharing, err = e.GetUnrealizedProfitSharingDetails(t.Context(), "SWAP")
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
				assert.True(t, accountRiskState[0].IsTheAccountAtRisk, "the documented atRisk boolean should decode")
			},
		},
		{
			name: "Signal bot event history decodes the alert message",
			call: func() error {
				var err error
				signalBotEventHistory, err = e.GetSignalBotEventHistory(t.Context(), "12345", time.Time{}, time.Time{}, 0)
				return err
			},
			path: "/tradingBot/signal/event-history",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, signalBotEventHistory, 1, "the event history row must decode")
				assert.Equal(t, "price alert", signalBotEventHistory[0].AlertMsg, "the documented alertMsg string should decode")
			},
		},
		{
			name: "Positions decode the realised PnL accumulations",
			call: func() error {
				var err error
				accountPositions, err = e.GetPositions(t.Context(), "SWAP", "", "")
				return err
			},
			path: "/account/positions",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, accountPositions, 1, "the position row must decode")
				assert.Equal(t, 12.5, accountPositions[0].RealizedProfitAndLoss.Float64(), "the documented realizedPnl should decode")
				assert.Equal(t, -0.1, accountPositions[0].FundingFee.Float64(), "the accumulated funding fee should decode")
				assert.Equal(t, 41000.0, accountPositions[0].BreakEvenPrice.Float64(), "the documented bePx should decode")
			},
		},
		{
			name: "Position history decodes the aligned types and close algos",
			call: func() error {
				_, err := e.GetPositionsHistory(t.Context(), "SWAP", "", "", "", 0, 100, time.Time{}, time.Time{})
				return err
			},
			path: "/account/positions-history",
			verify: func(t *testing.T) {
				t.Helper()
				history, err := e.GetPositionsHistory(t.Context(), "SWAP", "", "", "", 0, 100, time.Time{}, time.Time{})
				require.NoError(t, err, "positions history must decode")
				require.Len(t, history, 1, "the history row must decode")
				row := history[0]
				assert.Equal(t, asset.PerpetualSwap, row.InstrumentType, "the documented instType should decode")
				assert.Equal(t, 3.0, row.Leverage.Float64(), "the documented lever should decode")
				assert.Equal(t, 100.0, row.Margin.Float64(), "the documented margin should decode")
				assert.Equal(t, 0.1, row.OptionValue.Float64(), "the documented optVal should decode")
				assert.Equal(t, 42000.5, row.USDPrice.Float64(), "the documented usdPx should decode")
				assert.Equal(t, 41000.0, row.BreakEvenPrice.Float64(), "the documented bePx should decode")
				require.Len(t, row.CloseOrderAlgo, 1, "the close algo row must decode")
				assert.Equal(t, "123", row.CloseOrderAlgo[0].AlgoID, "the close algo ID should decode")
				assert.Equal(t, 38000.0, row.CloseOrderAlgo[0].StopLossTriggerPrice.Float64(), "the close algo stop-loss trigger should decode")
				assert.Equal(t, 0.5, row.CloseOrderAlgo[0].CloseFraction.Float64(), "the close algo fraction should decode")
			},
		},
		{
			name: "Single funding rate decodes formula and realised rate",
			call: func() error {
				var err error
				singleFundingRate, err = e.GetSingleFundingRate(t.Context(), "BTC-USD-SWAP")
				return err
			},
			path: "/public/funding-rate",
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, singleFundingRate, "the funding rate response must decode")
				assert.Equal(t, "withRate", singleFundingRate.FormulaType, "the documented formulaType should decode")
				assert.Equal(t, 0.00012, singleFundingRate.RealisedRate.Float64(), "the documented realizedRate should decode")
				assert.Equal(t, 1.2, singleFundingRate.ImpactValue.Float64(), "the documented impactValue should decode")
			},
		},
		{
			name: "Grid algo details decode trigger parameters and fees",
			call: func() error {
				var err error
				gridAlgoDetails, err = e.GetGridAlgoOrderDetails(t.Context(), "grid", "12345")
				return err
			},
			path: "/tradingBot/grid/orders-algo-details",
			verify: func(t *testing.T) {
				t.Helper()
				require.NotNil(t, gridAlgoDetails, "the grid algo detail must decode")
				assert.Equal(t, "grid-client-1", gridAlgoDetails.AlgoClientOrderID, "the documented algoClOrdId should decode")
				assert.Equal(t, "BTC-USDT", gridAlgoDetails.InstrumentFamily, "the documented instFamily should decode")
				require.Len(t, gridAlgoDetails.TriggerParams, 1, "the grid trigger parameters must decode")
				assert.Equal(t, "rsi", gridAlgoDetails.TriggerParams[0].TriggerStrategy, "the trigger strategy should decode")
				assert.Equal(t, "cross_up", gridAlgoDetails.TriggerParams[0].TriggerCondition, "the trigger condition should decode")
				assert.Equal(t, "30", gridAlgoDetails.TriggerParams[0].Threshold.String(), "the RSI threshold should decode")
				assert.Equal(t, -0.2, gridAlgoDetails.Fee.Float64(), "the accumulated fee should decode")
			},
		},
		{
			name: "Leading positions decode margin and unrealised PnL",
			call: func() error {
				var err error
				leadingPositions, err = e.GetExistingLeadingPositions(t.Context(), "SWAP", "", time.Time{}, time.Time{}, 0)
				return err
			},
			path: "/copytrading/current-subpositions",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, leadingPositions, 1, "the sub position row must decode")
				assert.Equal(t, 100.0, leadingPositions[0].Margin.Float64(), "the documented margin should decode")
				assert.Equal(t, "USDT", leadingPositions[0].MarginCurrency.String(), "the documented margin currency should decode")
				assert.Equal(t, 1.5, leadingPositions[0].UnrealizedProfitAndLoss.Float64(), "the unrealised PnL should decode")
				assert.Equal(t, 50000.0, leadingPositions[0].TakeProfitOrderPrice.Float64(), "the take-profit order price should decode")
			},
		},
		{
			name: "Withdrawal history decodes address type and note",
			call: func() error {
				var err error
				withdrawalHistory, err = e.GetWithdrawalHistory(t.Context(), currency.BTC, "", "", "", "", time.Time{}, time.Time{}, 0)
				return err
			},
			path: "/asset/withdrawal-history",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, withdrawalHistory, 2, "the withdrawal rows must decode")
				assert.Equal(t, "2", withdrawalHistory[0].ToAddressType, "the documented toAddrType should decode")
				assert.Equal(t, "w1", withdrawalHistory[0].WithdrawalNote, "the documented note should decode")
				assert.Equal(t, "123456", withdrawalHistory[0].AddrEx["comment"], "the documented addrEx attachment object should decode")
				assert.Nil(t, withdrawalHistory[1].AddrEx, "the documented null addrEx should decode into a nil map")
			},
		},
		{
			name: "Subaccount list decodes uid and level",
			call: func() error {
				var err error
				subaccountList, err = e.ViewSubAccountList(t.Context(), false, "", time.Time{}, time.Time{}, 0)
				return err
			},
			path: "/users/subaccount/list",
			verify: func(t *testing.T) {
				t.Helper()
				require.Len(t, subaccountList, 1, "the subaccount row must decode")
				assert.Equal(t, "123456", subaccountList[0].UID, "the documented uid should decode")
				assert.Equal(t, "1", subaccountList[0].SubaccountLevel, "the documented subAcctLv should decode")
				assert.Equal(t, "sub-one", subaccountList[0].FirstLevelSubaccount, "the documented firstLvSubAcct name should decode")
				assert.True(t, subaccountList[0].DirectMarketAccess, "the documented ifDma flag should decode")
			},
		},
		{
			name: "MMP config decodes the quoted time interval",
			call: func() error {
				var err error
				mmpConfigs, err = e.GetMMPConfig(t.Context(), "")
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
				depositWithdrawStatuses, err = e.GetDepositWithdrawalStatus(t.Context(), currency.EMPTYCODE, "1244", "", "", "")
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
				positionBuilder, err = e.NewPositionBuilder(t.Context(), &PositionBuilderParam{InclRealPosAndEq: true})
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

// TestNewPositionBuilderSendsTheModeParameters pins the documented request
// fields that select multi-currency margin, its leverage and the price
// volatility: without acctLv OKX applies portfolio margin, where positions stay
// empty, and without idxVol the before-volatility fields come back empty.
func TestNewPositionBuilderSendsTheModeParameters(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"positions":[],"ts":"1724751378980"}]}`))
	}))
	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	_, err := e.NewPositionBuilder(t.Context(), &PositionBuilderParam{
		AccountLevel:    3,
		Leverage:        10,
		IndexVolatility: -0.05,
		SimPos:          []SimulatedPosition{{InstrumentID: "BTC-USDT-SWAP", Position: "10", AveragePrice: 100000, Leverage: 5}},
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

	unset, err := json.Marshal(&PositionBuilderParam{InclRealPosAndEq: true})
	require.NoError(t, err, "Marshal must not error")
	var defaults map[string]any
	require.NoError(t, json.Unmarshal(unset, &defaults), "Unmarshal must not error")
	for _, key := range []string{"acctLv", "lever", "idxVol"} {
		assert.NotContainsf(t, defaults, key, "an unset %s should be left for OKX to default", key)
	}
}

// TestPositionBuilderRiskUnitDecodesQuotedNumbers pins that the risk unit and
// portfolio decode their quoted wire numbers into types.Number fields that
// callers can read as numbers.
func TestPositionBuilderRiskUnitDecodesQuotedNumbers(t *testing.T) {
	t.Parallel()
	var detail PositionBuilderDetail
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
}

// TestGetSpreadTickersOverlayBook pins the spread ticker merge:
// market/sprd-ticker serves a cached snapshot whose bid/ask and timestamp lag
// the live book, so the top of book comes from sprd/books?sz=1 while the last
// and the 24-hour figures stay on the ticker response.
func TestGetSpreadTickersOverlayBook(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	counts := map[string]int{}
	var booksQuery url.Values
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts[r.URL.Path]++
		if r.URL.Path == "/sprd/books" {
			booksQuery = r.URL.Query()
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/market/sprd-ticker":
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","lastSz":"0.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","open24h":"4","high24h":"14.5","low24h":"-2.2","vol24h":"6.67","ts":"1715331406485"}]}`))
		case "/sprd/books":
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"asks":[["15.2","3.3","1"]],"bids":[["14.8","4.4","2"]],"ts":"1715331407999"}]}`))
		default:
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
		}
	}))

	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	result, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
	require.NoError(t, err, "GetPublicSpreadTickers must not error")
	require.NotEmpty(t, result, "GetPublicSpreadTickers must return a ticker")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, counts["/sprd/books"], "the spread ticker should refresh the top of book from sprd/books")
	assert.Equal(t, "1", booksQuery.Get("sz"), "the top of book request should ask for a single level")
	assert.Equal(t, 1, counts["/market/sprd-ticker"], "the last and 24 hour figures should come from market/sprd-ticker")
	assert.Equal(t, 14.8, result[0].BidPrice.Float64(), "bidPx should come from the sprd/books top of book")
	assert.Equal(t, 4.4, result[0].BidSize.Float64(), "bidSz should come from the sprd/books top of book")
	assert.Equal(t, 15.2, result[0].AskPrice.Float64(), "askPx should come from the sprd/books top of book")
	assert.Equal(t, 3.3, result[0].AskSize.Float64(), "askSz should come from the sprd/books top of book")
	assert.Equal(t, 14.5, result[0].Last.Float64(), "last should come from market/sprd-ticker")
	assert.Equal(t, 4.0, result[0].OpenPrice24Hour.Float64(), "open24h should come from market/sprd-ticker")
	assert.Equal(t, 14.5, result[0].HighestPrice24Hour.Float64(), "high24h should come from market/sprd-ticker")
	assert.Equal(t, -2.2, result[0].LowestPrice24Hour.Float64(), "low24h should come from market/sprd-ticker")
	assert.Equal(t, 6.67, result[0].TradingVolume24Hour.Float64(), "vol24h should come from market/sprd-ticker")
	assert.Equal(t, int64(1715331407999), result[0].Timestamp.Time().UnixMilli(), "the timestamp should advance to the sprd/books generation time")
}

// TestEstCompleteTimeUnmarshalJSON pins the documented estCompleteTime forms:
// the wall-clock MM/dd/yyyy, h:mm:ss AM/PM string in UTC+8 decodes, and the
// null and empty placeholders stay the zero time.
func TestEstCompleteTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var estCompleteTime EstimatedCompleteTime
	require.NoError(t, json.Unmarshal([]byte(`"01/09/2023, 8:10:48 PM"`), &estCompleteTime), "the documented wall-clock form must decode")
	expected := time.Date(2023, time.January, 9, 20, 10, 48, 0, time.FixedZone("UTC+8", 8*60*60))
	assert.True(t, estCompleteTime.Time().Equal(expected), "the documented UTC+8 wall-clock time should decode")
	for _, form := range []string{`null`, `""`} {
		var empty EstimatedCompleteTime
		require.NoErrorf(t, json.Unmarshal([]byte(form), &empty), "the %s placeholder must decode", form)
		assert.True(t, empty.Time().IsZero(), "the placeholder should stay the zero time")
	}
	var invalid EstimatedCompleteTime
	err := json.Unmarshal([]byte(`"not-a-time"`), &invalid)
	require.ErrorIs(t, err, types.ErrInvalidTimestampFormat, "an unparsable timestamp must wrap the shared sentinel")
	err = json.Unmarshal([]byte(`1673266248000`), &invalid)
	require.ErrorIs(t, err, types.ErrInvalidTimestampFormat, "a non-string estCompleteTime must wrap the shared sentinel")
	var escaped EstimatedCompleteTime
	require.NoError(t, json.Unmarshal([]byte(`"01\/09\/2023, 8:10:48 PM"`), &escaped), "a JSON-escaped wall-clock string must decode")
	assert.True(t, escaped.Time().Equal(expected), "the escaped form should decode to the same time")
	encoded, err := json.Marshal(estCompleteTime)
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, `"2023-01-09T20:10:48+08:00"`, string(encoded), "the estimate should serialise as an RFC 3339 timestamp")
	assert.Equal(t, "2023-01-09 20:10:48 +0800 UTC+8", estCompleteTime.String(), "the estimate should print as its time")
}

// TestOrderBookSequenceIDDecodesTheBareInteger pins the REST book's seqId
// typing: the wire carries a bare, never-negative integer, so the field is
// an unsigned integer rather than a types.Number.
func TestOrderBookSequenceIDDecodesTheBareInteger(t *testing.T) {
	t.Parallel()
	var ob OrderBookResponseDetail
	require.NoError(t, json.Unmarshal([]byte(`{"asks":[["42000","0.1","0","3"]],"bids":[],"seqId":3235851742,"ts":"1724751378980"}`), &ob), "the books row must decode")
	exp := OrderBookResponseDetail{
		Asks:                []OrderbookItemDetail{{DepthPrice: 42000, Amount: 0.1, NumberOfOrders: 3}},
		Bids:                []OrderbookItemDetail{},
		SequenceID:          uint64(3235851742),
		GenerationTimestamp: types.Time(time.UnixMilli(1724751378980)),
	}
	assert.Equal(t, exp, ob, "the books row should decode with its bare seqId integer")
}

// TestGetSpreadTickersBookFailure pins that a failed sprd/books request fails
// the spread ticker rather than returning market/sprd-ticker's cached top of book.
func TestGetSpreadTickersBookFailure(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var paths []string
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/sprd/books" {
			_, _ = w.Write([]byte(`{"code":"50026","msg":"System error, please try again later.","data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","ts":"1715331406485"}]}`))
	}))
	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	_, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
	require.ErrorContains(t, err, "50026", "a sprd/books failure must fail the spread ticker rather than return its cached top of book")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"/market/sprd-ticker", "/sprd/books"}, paths, "the books refresh should fail only after the ticker request succeeds")
}

// TestGetSpreadTickersPartialBook pins that a side sprd/books doesn't have is
// cleared rather than left at market/sprd-ticker's cached quote.
func TestGetSpreadTickersPartialBook(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		books string
		exp   SpreadTicker
	}{
		{
			name:  "bids only",
			books: `[{"asks":[],"bids":[["14.8","4.4","2"]],"ts":"1715331407999"}]`,
			exp:   SpreadTicker{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, BidPrice: 14.8, BidSize: 4.4, Timestamp: types.Time(time.UnixMilli(1715331407999))},
		},
		{
			name:  "asks only",
			books: `[{"asks":[["15.2","3.3","1"]],"bids":[],"ts":"1715331407999"}]`,
			exp:   SpreadTicker{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, AskPrice: 15.2, AskSize: 3.3, Timestamp: types.Time(time.UnixMilli(1715331407999))},
		},
		{
			name:  "empty book",
			books: `[{"asks":[],"bids":[],"ts":"1715331407999"}]`,
			exp:   SpreadTicker{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, Timestamp: types.Time(time.UnixMilli(1715331407999))},
		},
		{
			name:  "older book",
			books: `[{"asks":[["15.2","3.3","1"]],"bids":[],"ts":"1715331400000"}]`,
			exp:   SpreadTicker{SpreadID: "BTC-USDT_BTC-USDT-SWAP", Last: 14.5, AskPrice: 15.2, AskSize: 3.3, Timestamp: types.Time(time.UnixMilli(1715331406485))},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/market/sprd-ticker":
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","last":"14.5","askPx":"8.5","askSz":"12.0","bidPx":"0.5","bidSz":"12.0","ts":"1715331406485"}]}`))
				case "/sprd/books":
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.books + `}`))
				default:
					assert.Failf(t, "unexpected request", "path %s should not be requested", r.URL.Path)
				}
			}))
			b := e.GetBase()
			require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
			for k := range b.API.Endpoints.GetURLMap() {
				require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
			}

			result, err := e.GetPublicSpreadTickers(t.Context(), "BTC-USDT_BTC-USDT-SWAP")
			require.NoError(t, err, "GetPublicSpreadTickers must not error")
			require.Len(t, result, 1, "GetPublicSpreadTickers must return the ticker")
			assert.Equal(t, tc.exp, result[0], "a side missing from sprd/books should be cleared, not left at the cached quote")
		})
	}
}

// TestMMPConfigTimeIntervalNumberForms pins the timeInterval typing: the
// documented quoted form decodes, and a types.Number also reads the bare
// integer and empty forms the old `,string` tag rejected outright.
func TestMMPConfigTimeIntervalNumberForms(t *testing.T) {
	t.Parallel()
	var quoted MMPConfigDetail
	require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":"5000"}`), &quoted), "the documented quoted timeInterval must decode")
	assert.Equal(t, int64(5000), quoted.TimeInterval.Int64(), "the quoted timeInterval should decode")
	var bare MMPConfigDetail
	require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":5000}`), &bare), "the bare integer form must decode")
	assert.Equal(t, int64(5000), bare.TimeInterval.Int64(), "the bare timeInterval should decode")
	var empty MMPConfigDetail
	require.NoError(t, json.Unmarshal([]byte(`{"timeInterval":""}`), &empty), "the empty form must decode")
	assert.Equal(t, 0.0, empty.TimeInterval.Float64(), "the empty timeInterval should stay the zero number")
}
