package binance

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

func TestPoolBudgets(t *testing.T) {
	t.Parallel()
	rows := []struct {
		pool        rateLimitPool
		interval    time.Duration
		limit       int
		perEndpoint bool
	}{
		{spotIPPool, time.Minute, 6000, false},
		{spotOrderPool, 10 * time.Second, 100, false},
		{sapiIPPool, time.Minute, 12000, true},
		{sapiUIDPool, time.Minute, 180000, true},
		{derivativesIPPool, time.Minute, 2400, false},
		{derivativesOrderPool, time.Minute, 1200, false},
		{futuresDataPool, 5 * time.Minute, 1000, false},
		{uFuturesFundingPool, 5 * time.Minute, 500, false},
		{optionsIPPool, time.Minute, 400, false},
		{optionsOrderPool, time.Minute, 100, false},
		{portfolioMarginIPPool, time.Minute, 6000, false},
		{portfolioMarginOrderPool, time.Minute, 1200, false},
		{marginMaxLeveragePool, time.Minute, 1, false},
		{subAccountFuturesTransferPool, time.Minute, 2000, false},
		{pmProBNBTransferPool, 10 * time.Minute, 2, false},
		{autoInvestPlanCreationPool, 3 * time.Second, 1, false},
		{autoInvestPlanAdjustmentPool, 3 * time.Second, 1, false},
		{autoInvestPlanStatusPool, 3 * time.Second, 1, false},
		{autoInvestPlanListPool, 3 * time.Second, 1, false},
		{autoInvestOneTimeTransactionPool, 3 * time.Second, 1, false},
		{autoInvestRedemptionPool, 3 * time.Second, 1, false},
		{flexibleLoanLTVAdjustmentHistoryPool, time.Second, 5, false},
		{vipLoanApplicationStatusPool, time.Second, 5, false},
		{brokerCreateAPIKeyPool, time.Second, 1, false},
		{brokerDeleteAPIKeyPool, time.Second, 1, false},
		{brokerFuturesTransferPool, time.Minute, 5000, false},
		{pmBNBTransferPool, 10 * time.Minute, 10, false},
	}
	listed := make(map[rateLimitPool]bool, len(rows))
	for _, tc := range rows {
		listed[tc.pool] = true
		assert.Equalf(t, rateLimitBudget{tc.interval, tc.limit}, poolBudgets[tc.pool], "pool %d should have its documented budget", tc.pool)
		assert.Equalf(t, tc.perEndpoint, tc.pool.perEndpoint(), "pool %d should only budget endpoints separately where Binance does", tc.pool)
	}
	for pool := spotIPPool; pool <= pmBNBTransferPool; pool++ {
		assert.Truef(t, listed[pool], "pool %d should have a documented budget row", pool)
	}
	assert.Equal(t, len(rows), len(poolBudgets), "poolBudgets should hold exactly the documented pools")

	used := make(map[rateLimitPool]bool, len(poolBudgets))
	for _, cost := range endpointCosts {
		used[cost.pool] = true
	}
	for pool := range poolBudgets {
		assert.Truef(t, used[pool], "pool %d should be drawn on by an endpoint limit", pool)
	}
}

func TestEndpointCosts(t *testing.T) {
	t.Parallel()
	rows := []struct {
		epl       request.EndpointLimit
		pool      rateLimitPool
		weight    request.Weight
		endpoints string
	}{
		// Spot /api, REST and WebSocket API
		{spotDefaultRate, spotIPPool, 1, "weight-1 /api endpoints: GET /api/v3/time, order cancels, order tests without commission rates"},
		{spotExchangeInfoRate, spotIPPool, 20, "GET /api/v3/exchangeInfo"},
		{aggTradesRate, spotIPPool, 4, "GET /api/v3/aggTrades, WS trades.aggregate"},
		{getCurrentAveragePriceRate, spotIPPool, 2, "GET /api/v3/avgPrice, WS avgPrice"},
		{spotOrderbookDepth100Rate, spotIPPool, 5, "GET /api/v3/depth limit 1-100"},
		{spotOrderbookDepth500Rate, spotIPPool, 25, "GET /api/v3/depth limit 101-500"},
		{spotOrderbookDepth1000Rate, spotIPPool, 50, "GET /api/v3/depth limit 501-1000"},
		{spotOrderbookDepth5000Rate, spotIPPool, 250, "GET /api/v3/depth limit 1001-5000"},
		{getRecentTradesListRate, spotIPPool, 25, "GET /api/v3/trades, WS trades.recent"},
		{getOldTradeLookupRate, spotIPPool, 25, "GET /api/v3/historicalTrades"},
		{getKlineRate, spotIPPool, 2, "GET /api/v3/klines, GET /api/v3/uiKlines"},
		{spotTickerSymbols1Rate, spotIPPool, 4, "GET /api/v3/ticker and /ticker/tradingDay, 1 symbol at 4 each"},
		{spotTickerSymbols5Rate, spotIPPool, 20, "GET /api/v3/ticker and /ticker/tradingDay, up to 5 symbols at 4 each"},
		{spotTickerSymbols10Rate, spotIPPool, 40, "GET /api/v3/ticker and /ticker/tradingDay, up to 10 symbols at 4 each"},
		{spotTickerSymbols25Rate, spotIPPool, 100, "GET /api/v3/ticker and /ticker/tradingDay, up to 25 symbols at 4 each"},
		{spotTickerSymbolsMaxRate, spotIPPool, 200, "GET /api/v3/ticker and /ticker/tradingDay, the cap of 200"},
		{getTickers20Rate, spotIPPool, 2, "GET /api/v3/ticker/24hr, 1-20 symbols"},
		{getTickers100Rate, spotIPPool, 40, "GET /api/v3/ticker/24hr, 21-100 symbols"},
		{spotPriceChangeAllRate, spotIPPool, 80, "GET /api/v3/ticker/24hr, more than 100 symbols or all"},
		{spotBookTickerRate, spotIPPool, 2, "GET /api/v3/ticker/bookTicker, one symbol"},
		{spotOrderbookTickerAllRate, spotIPPool, 4, "GET /api/v3/ticker/bookTicker, symbols or all"},
		{spotSymbolPriceRate, spotIPPool, 2, "GET /api/v3/ticker/price, one symbol"},
		{spotSymbolPriceAllRate, spotIPPool, 4, "GET /api/v3/ticker/price, symbols or all"},
		{spotOrderRate, spotOrderPool, 1, "POST /api/v3/order, /order/cancelReplace, /sor/order: 1 order"},
		{spotOCOOrderRate, spotOrderPool, 2, "POST /api/v3/orderList/oco: 2 orders"},
		{testNewOrderWithCommissionRate, spotIPPool, 20, "POST /api/v3/order/test and /sor/order/test with computeCommissionRates"},
		{getCommissionRate, spotIPPool, 20, "GET /api/v3/account/commission, WS account.commission"},
		{getAllOCOOrdersRate, spotIPPool, 20, "GET /api/v3/allOrderList, WS allOrderLists"},
		{spotAllOrdersRate, spotIPPool, 20, "GET /api/v3/allOrders, WS allOrders"},
		{spotAccountInformationRate, spotIPPool, 20, "GET /api/v3/account, WS account.status"},
		{spotOpenOrdersSpecificRate, spotIPPool, 6, "GET /api/v3/openOrders, one symbol"},
		{spotOpenOrdersAllRate, spotIPPool, 80, "GET /api/v3/openOrders, all symbols"},
		{spotOrderQueryRate, spotIPPool, 4, "GET /api/v3/order, WS order.status"},
		{getOCOListRate, spotIPPool, 4, "GET /api/v3/orderList, WS orderList.status"},
		{getAllocationsRate, spotIPPool, 20, "GET /api/v3/myAllocations, WS myAllocations"},
		{preventedMatchesRate, spotIPPool, 2, "GET /api/v3/myPreventedMatches by preventedMatchId"},
		{preventedMatchesByOrderIDRate, spotIPPool, 20, "GET /api/v3/myPreventedMatches by orderId"},
		{accountTradeListRate, spotIPPool, 20, "GET /api/v3/myTrades without orderId"},
		{accountTradeListByOrderIDRate, spotIPPool, 5, "GET /api/v3/myTrades with orderId"},
		{getOpenOCOListRate, spotIPPool, 6, "GET /api/v3/openOrderList, WS openOrderLists.status"},
		{currentOrderCountUsageRate, spotIPPool, 40, "GET /api/v3/rateLimit/order, WS account.rateLimits.orders"},
		{wsSessionRate, spotIPPool, 2, "WS session.status and session.logout"},
		{userDataStreamSubscribeRate, spotIPPool, 2, "WS userDataStream.subscribe.signature and userDataStream.subscribe.listenToken"},
		// /sapi weight-1 defaults, used by every SAPI family
		{sapiDefaultRate, sapiIPPool, 1, "weight-1 /sapi IP endpoints, and undocumented ones, which default to 1"},
		{sapiUIDDefaultRate, sapiUIDPool, 1, "DELETE /sapi/v1/margin/orderList, GET /sapi/v1/managed-subaccount/deposit/address, POST /sapi/v1/userListenToken"},
		// Margin
		{adjustCrossMarginMaxLeverageRate, marginMaxLeveragePool, 1, "POST /sapi/v1/margin/max-leverage: UID 3000 and 1 request per minute"},
		{getIsolatedMarginAccountInfoRate, sapiIPPool, 10, "GET /sapi/v1/margin/isolated/account"},
		{enableIsolatedMarginAccountRate, sapiUIDPool, 300, "POST /sapi/v1/margin/isolated/account"},
		{disableIsolatedMarginAccountRate, sapiUIDPool, 300, "DELETE /sapi/v1/margin/isolated/account"},
		{marginAccountSummaryRate, sapiIPPool, 10, "GET /sapi/v1/margin/tradeCoeff"},
		{marginCapitalFlowRate, sapiIPPool, 100, "GET /sapi/v1/margin/capital-flow"},
		{getCrossMarginAccountDetailRate, sapiIPPool, 10, "GET /sapi/v1/margin/account"},
		{allCrossMarginFeeDataRate, sapiIPPool, 5, "GET /sapi/v1/margin/crossMarginData without coin"},
		{allIsolatedMarginFeeDataRate, sapiIPPool, 10, "GET /sapi/v1/margin/isolatedMarginData without symbol"},
		{marginHourlyInterestRate, sapiIPPool, 100, "GET /sapi/v1/margin/next-hourly-interest-rate"},
		{borrowRepayRecordsInMarginAccountRate, sapiIPPool, 10, "GET /sapi/v1/margin/borrow-repay"},
		{marginAccountBorrowRepayRate, sapiUIDPool, 1500, "POST /sapi/v1/margin/borrow-repay"},
		{marginMaxBorrowRate, sapiUIDPool, 750, "GET /sapi/v1/margin/maxBorrowable: UID 750 since the 2026-04-16 changelog"},
		{crossMarginCollateralRatioRate, sapiIPPool, 100, "GET /sapi/v1/margin/crossMarginCollateralRatio"},
		{allIsolatedMarginSymbolsRate, sapiIPPool, 10, "GET /sapi/v1/margin/isolated/allPairs"},
		{marginTokensAndSymbolsDelistScheduleRate, sapiIPPool, 100, "GET /sapi/v1/margin/delist-schedule"},
		{marginAvailableInventoryRate, sapiUIDPool, 50, "GET /sapi/v1/margin/available-inventory"},
		{getPriceMarginIndexRate, sapiIPPool, 10, "GET /sapi/v1/margin/priceIndex"},
		{getSmallLiabilityExchangeCoinListRate, sapiIPPool, 100, "GET /sapi/v1/margin/exchange-small-liability"},
		{marginSmallLiabilityExchangeRate, sapiUIDPool, 3000, "POST /sapi/v1/margin/exchange-small-liability"},
		{smallLiabilityExchangeHistoryRate, sapiUIDPool, 100, "GET /sapi/v1/margin/exchange-small-liability-history"},
		{getMarginAccountsOpenOrdersRate, sapiIPPool, 10, "GET /sapi/v1/margin/openOrders"},
		{getMarginAccountOCOOrderRate, sapiIPPool, 10, "GET /sapi/v1/margin/orderList"},
		{getMarginAccountOrderRate, sapiIPPool, 10, "GET /sapi/v1/margin/order"},
		{marginAccountNewOrderRate, sapiUIDPool, 6, "POST /sapi/v1/margin/order and /order/oco without borrowing"},
		{marginAccountNewOrderBorrowRate, sapiUIDPool, 1500, "POST /sapi/v1/margin/order and /order/oco with MARGIN_BUY or AUTO_BORROW_REPAY"},
		{marginAccountCancelOrderRate, sapiIPPool, 10, "DELETE /sapi/v1/margin/order"},
		{marginManualLiquidationRate, sapiUIDPool, 3000, "POST /sapi/v1/margin/manual-liquidation"},
		{marginCurrentOrderCountUsageRate, sapiIPPool, 20, "GET /sapi/v1/margin/rateLimit/order"},
		{getMarginAccountAllOCORate, sapiIPPool, 200, "GET /sapi/v1/margin/allOrderList"},
		{marginAccountsAllOrdersRate, sapiIPPool, 200, "GET /sapi/v1/margin/allOrders"},
		{marginAccountOpenOCOOrdersRate, sapiIPPool, 10, "GET /sapi/v1/margin/openOrderList"},
		{marginAccountTradeListRate, sapiIPPool, 10, "GET /sapi/v1/margin/myTrades"},
		{maxTransferOutRate, sapiIPPool, 50, "GET /sapi/v1/margin/maxTransferable"},
		// Wallet
		{dailyAccountSnapshotRate, sapiIPPool, 2400, "GET /sapi/v1/accountSnapshot"},
		{assetDividendRecordRate, sapiIPPool, 10, "GET /sapi/v1/asset/assetDividend"},
		{dustTransferRate, sapiUIDPool, 10, "POST /sapi/v1/asset/dust"},
		{cloudMiningPaymentAndRefundHistoryRate, sapiUIDPool, 600, "GET /sapi/v1/asset/ledger-transfer/cloud-mining/queryByPage"},
		{getUserDelegationHistoryRate, sapiIPPool, 60, "GET /sapi/v1/asset/custody/transfer-history"},
		{userUniversalTransferRate, sapiUIDPool, 300, "POST /sapi/v1/asset/transfer"},
		{getUserWalletBalanceRate, sapiIPPool, 60, "GET /sapi/v1/asset/wallet/balance"},
		{userAssetsRate, sapiIPPool, 5, "POST /sapi/v3/asset/getUserAsset"},
		{allCoinInfoRate, sapiIPPool, 10, "GET /sapi/v1/capital/config/getall"},
		{depositAddressesRate, sapiIPPool, 10, "GET /sapi/v1/capital/deposit/address"},
		{getDepositAddressListInNetworkRate, sapiIPPool, 10, "GET /sapi/v1/capital/deposit/address/list"},
		{withdrawAddressListRate, sapiIPPool, 10, "GET /sapi/v1/capital/withdraw/address/list"},
		{fundWithdrawalRate, sapiUIDPool, 900, "POST /sapi/v1/capital/withdraw/apply"},
		{withdrawalHistoryRate, sapiUIDPool, 18000, "GET /sapi/v1/capital/withdraw/history"},
		{symbolDelistScheduleForSpotRate, sapiIPPool, 100, "GET /sapi/v1/spot/delist-schedule"},
		{depositQuestionnaireRate, sapiUIDPool, 600, "PUT /sapi/v1/localentity/deposit/provide-info"},
		// Sub-account
		{getFuturesPositionRiskOfSubAccountV1Rate, sapiIPPool, 10, "GET /sapi/v1/sub-account/futures/positionRisk"},
		{getSubAccountStatusOnMarginOrFuturesRate, sapiIPPool, 10, "GET /sapi/v1/sub-account/status"},
		{getSubAccountTransactionStatisticsRate, sapiIPPool, 60, "GET /sapi/v1/sub-account/transaction-statistics"},
		{addIPRestrictionSubAccountAPIKeyRate, sapiUIDPool, 3000, "POST /sapi/v2/sub-account/subAccountApi/ipRestriction"},
		{deleteIPListForSubAccountAPIKeyRate, sapiUIDPool, 3000, "DELETE /sapi/v1/sub-account/subAccountApi/ipRestriction/ipList"},
		{ipRestrictionForSubAccountAPIKeyRate, sapiUIDPool, 3000, "GET /sapi/v1/sub-account/subAccountApi/ipRestriction"},
		{getDetailSubAccountFuturesAccountRate, sapiIPPool, 10, "GET /sapi/v1/sub-account/futures/account"},
		{subAccountMarginAccountDetailRate, sapiIPPool, 10, "GET /sapi/v1/sub-account/margin/account"},
		{getFuturesSubAccountSummaryV2Rate, sapiIPPool, 10, "GET /sapi/v2/sub-account/futures/accountSummary"},
		{getSubAccountSummaryOfMarginAccountRate, sapiIPPool, 10, "GET /sapi/v1/sub-account/margin/accountSummary"},
		{getV3SubAccountAssetsRate, sapiUIDPool, 60, "GET /sapi/v3/sub-account/assets"},
		{getSubAccountAssetRate, sapiUIDPool, 60, "GET /sapi/v4/sub-account/assets"},
		{subAccountFuturesAssetTransferRate, subAccountFuturesTransferPool, 1, "POST /sapi/v1/sub-account/futures/internalTransfer: 2000 a minute"},
		{universalTransferForMasterAccountRate, sapiUIDPool, 360, "POST /sapi/v1/sub-account/universalTransfer: IP 1 and UID 360"},
		{managedSubAccountFuturesAssetDetailRate, sapiUIDPool, 60, "GET /sapi/v1/managed-subaccount/fetch-future-asset"},
		{getManagedSubAccountListRate, sapiUIDPool, 60, "GET /sapi/v1/managed-subaccount/info"},
		{getManagedSubAccountSnapshotRate, sapiIPPool, 2400, "GET /sapi/v1/managed-subaccount/accountSnapshot"},
		{managedSubAccountTransferLogRate, sapiUIDPool, 60, "GET /sapi/v1/managed-subaccount/query-trans-log, GET /sapi/v1/managed-subaccount/queryTransLogForTradeParent"},
		// Portfolio Margin Pro, formerly classic portfolio margin
		{transferBNBRate, pmProBNBTransferPool, 1, "POST /sapi/v1/portfolio/bnb-transfer: IP 1500, 2 calls per 10 minutes"},
		{getAutoRepayFuturesStatusRate, sapiIPPool, 30, "GET /sapi/v1/portfolio/repay-futures-switch"},
		{changeAutoRepayFuturesStatusRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/repay-futures-switch"},
		{fundAutoCollectionRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/auto-collection"},
		{fundCollectionByAssetRate, sapiIPPool, 60, "POST /sapi/v1/portfolio/asset-collection"},
		{classicPMAccountInfoRate, sapiUIDPool, 5, "GET /sapi/v1/portfolio/account"},
		{repayClassicPMBankruptcyLoanRate, sapiUIDPool, 3000, "POST /sapi/v1/portfolio/repay"},
		{getClassicPMBankruptcyLoanAmountRate, sapiUIDPool, 500, "GET /sapi/v1/portfolio/pmLoan"},
		{classicPMNegativeBalanceInterestHistoryRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/interest-history"},
		{repayFuturesNegativeBalanceRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/repay-futures-negative-balance"},
		{pmAssetLeverageRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/margin-asset-leverage"},
		{classicPMCollateralRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/collateralRate"},
		{pmAssetIndexPriceRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/asset-index-price without asset"},
		// Simple Earn
		{personalLeftQuotaRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/personalLeftQuota, GET /sapi/v1/simple-earn/locked/personalLeftQuota"},
		{getFlexibleSimpleEarnProductPositionRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/position"},
		{getRedemptionRecordRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/history/redemptionRecord, GET /sapi/v1/simple-earn/locked/history/redemptionRecord"},
		{getRewardHistoryRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/history/rewardsRecord, GET /sapi/v1/simple-earn/locked/history/rewardsRecord"},
		{subscriptionPreviewRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/subscriptionPreview, GET /sapi/v1/simple-earn/locked/subscriptionPreview"},
		{getFlexibleSubscriptionRecordRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/history/subscriptionRecord"},
		{getSimpleEarnProductPositionRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/locked/position"},
		{getLockedSubscriptionRecordsRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/locked/history/subscriptionRecord"},
		{simpleEarnRateHistoryRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/history/rateHistory"},
		{simpleEarnProductsRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/flexible/list, GET /sapi/v1/simple-earn/locked/list"},
		{setAutoSubscribeRate, sapiIPPool, 150, "POST /sapi/v1/simple-earn/flexible/setAutoSubscribe, POST /sapi/v1/simple-earn/locked/setAutoSubscribe"},
		{setRedeemOptionRate, sapiIPPool, 50, "POST /sapi/v1/simple-earn/locked/setRedeemOption"},
		{simpleAccountRate, sapiIPPool, 150, "GET /sapi/v1/simple-earn/account"},
		// Staking
		{ethStakingAccountRate, sapiIPPool, 150, "GET /sapi/v2/eth-staking/account"},
		{bethRewardDistributionHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/history/rewardsHistory, no longer documented"},
		{currentETHStakingQuotaRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/quota"},
		{ethRedemptionHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/history/redemptionHistory"},
		{ethStakingHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/history/stakingHistory"},
		{getWBETHRateHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/history/rateHistory"},
		{wbethRewardsHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/eth/history/wbethRewardsHistory"},
		{wbethWrapOrUnwrapHistoryRate, sapiIPPool, 150, "GET /sapi/v1/eth-staking/wbeth/history/unwrapHistory, GET /sapi/v1/eth-staking/wbeth/history/wrapHistory"},
		{ethereumStakingRedemptionRate, sapiIPPool, 150, "POST /sapi/v1/eth-staking/eth/redeem"},
		{subscribeETHStakingRate, sapiIPPool, 150, "POST /sapi/v2/eth-staking/eth/stake"},
		{wrapBETHRate, sapiIPPool, 150, "POST /sapi/v1/eth-staking/wbeth/wrap"},
		{claimBoostRewardsRate, sapiIPPool, 150, "POST /sapi/v1/sol-staking/sol/claim"},
		{bnsolRateHistoryRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/rateHistory"},
		{bnsolRewardsHistoryRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/bnsolRewardsHistory"},
		{boostRewardsHistoryRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/boostRewardsHistory"},
		{solRedemptionHistoryRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/redemptionHistory"},
		{solStakingHistoryRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/stakingHistory"},
		{solStakingQuotaDetailsRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/quota"},
		{unclaimedRewardsRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/sol/history/unclaimedRewards"},
		{redeemSOLRate, sapiIPPool, 150, "POST /sapi/v1/sol-staking/sol/redeem"},
		{solStakingAccountRate, sapiIPPool, 150, "GET /sapi/v1/sol-staking/account"},
		{subscribeSOLStakingRate, sapiIPPool, 150, "POST /sapi/v1/sol-staking/sol/stake"},
		// Auto-Invest, no longer documented
		{autoInvestPlanCreationRate, autoInvestPlanCreationPool, 1, "POST /sapi/v1/lending/auto-invest/plan/add: one request every 3 seconds when last documented"},
		{autoInvestPlanAdjustmentRate, autoInvestPlanAdjustmentPool, 1, "POST /sapi/v1/lending/auto-invest/plan/edit: one request every 3 seconds when last documented"},
		{autoInvestPlanStatusRate, autoInvestPlanStatusPool, 1, "POST /sapi/v1/lending/auto-invest/plan/edit-status: one request every 3 seconds when last documented"},
		{autoInvestPlanListRate, autoInvestPlanListPool, 1, "GET /sapi/v1/lending/auto-invest/plan/list: one request every 3 seconds when last documented"},
		{autoInvestOneTimeTransactionRate, autoInvestOneTimeTransactionPool, 1, "POST /sapi/v1/lending/auto-invest/one-off: one request every 3 seconds when last documented"},
		{autoInvestRedemptionRate, autoInvestRedemptionPool, 1, "POST /sapi/v1/lending/auto-invest/redeem: one request every 3 seconds when last documented"},
		// Crypto Loan
		{checkCollateralRepayRate, sapiIPPool, 6000, "GET /sapi/v2/loan/flexible/repay/rate"},
		{adjustFlexibleLoanRate, sapiUIDPool, 6000, "POST /sapi/v2/loan/flexible/adjust/ltv"},
		{borrowFlexibleRate, sapiIPPool, 6000, "POST /sapi/v2/loan/flexible/borrow"},
		{repayFlexibleLoanRate, sapiIPPool, 6000, "POST /sapi/v2/loan/flexible/repay"},
		{flexibleLoanAssetDataRate, sapiIPPool, 400, "GET /sapi/v2/loan/flexible/loanable/data"},
		{flexibleBorrowHistoryRate, sapiIPPool, 400, "GET /sapi/v2/loan/flexible/borrow/history"},
		{flexibleLoanCollateralAssetRate, sapiIPPool, 400, "GET /sapi/v2/loan/flexible/collateral/data"},
		{flexibleLoanLiquidationHistoryRate, sapiIPPool, 400, "GET /sapi/v2/loan/flexible/liquidation/history"},
		{flexibleLoanLTVAdjustmentHistoryRate, flexibleLoanLTVAdjustmentHistoryPool, 1, "GET /sapi/v2/loan/flexible/ltv/adjustment/history: UID 400, 5 a second"},
		{getFlexibleLoanOngoingOrdersRate, sapiIPPool, 300, "GET /sapi/v2/loan/flexible/ongoing/orders"},
		{flexibleLoanRepaymentHistoryRate, sapiIPPool, 400, "GET /sapi/v2/loan/flexible/repay/history"},
		{cryptoLoansIncomeHistoryRate, sapiUIDPool, 6000, "GET /sapi/v1/loan/income"},
		{getLoanBorrowHistoryRate, sapiIPPool, 400, "GET /sapi/v1/loan/borrow/history"},
		{getLoanLTVAdjustmentHistoryRate, sapiIPPool, 400, "GET /sapi/v1/loan/ltv/adjustment/history"},
		{repaymentHistoryRate, sapiIPPool, 400, "GET /sapi/v1/loan/repay/history"},
		// VIP Loan
		{getVIPBorrowInterestRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/request/interestRate"},
		{getCollateralAssetDataRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/collateral/data"},
		{getVIPLoanableAssetsRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/loanable/data"},
		{vipLoanInterestRateHistoryRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/interestRateHistory"},
		{vipLoanBorrowRate, sapiUIDPool, 6000, "POST /sapi/v1/loan/vip/borrow"},
		{vipLoanRenewRate, sapiUIDPool, 6000, "POST /sapi/v1/loan/vip/renew"},
		{vipLoanRepayRate, sapiUIDPool, 6000, "POST /sapi/v1/loan/vip/repay"},
		{checkLockedValueVIPCollateralAccountRate, sapiIPPool, 6000, "GET /sapi/v1/loan/vip/collateral/account"},
		{getVIPLoanAccruedInterestRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/accruedInterest"},
		{getVIPLoanOngoingOrdersRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/ongoing/orders"},
		{getVIPLoanRepaymentHistoryRate, sapiIPPool, 400, "GET /sapi/v1/loan/vip/repay/history"},
		{getApplicationStatusRate, vipLoanApplicationStatusPool, 1, "GET /sapi/v1/loan/vip/request/data: UID 400, 5 a second"},
		// Convert
		{getAllConvertPairsRate, sapiIPPool, 3000, "GET /sapi/v1/convert/exchangeInfo"},
		{getOrderQuantityPrecisionPerAssetRate, sapiIPPool, 100, "GET /sapi/v1/convert/assetInfo"},
		{acceptQuoteRate, sapiUIDPool, 500, "POST /sapi/v1/convert/acceptQuote"},
		{cancelLimitOrderRate, sapiUIDPool, 200, "POST /sapi/v1/convert/limit/cancelOrder"},
		{convertTradeFlowHistoryRate, sapiUIDPool, 3000, "GET /sapi/v1/convert/tradeFlow"},
		{convertOrderStatusRate, sapiUIDPool, 100, "GET /sapi/v1/convert/orderStatus"},
		{placeLimitOrderRate, sapiUIDPool, 500, "POST /sapi/v1/convert/limit/placeOrder"},
		{getLimitOpenOrdersRate, sapiUIDPool, 3000, "GET /sapi/v1/convert/limit/queryOpenOrders"},
		{sendQuoteRequestRate, sapiUIDPool, 200, "POST /sapi/v1/convert/getQuote"},
		// Algo
		{placeTWAveragePriceNewOrderRate, sapiUIDPool, 3000, "POST /sapi/v1/algo/futures/newOrderTwap"},
		{placeVPOrderRate, sapiUIDPool, 300, "POST /sapi/v1/algo/futures/newOrderVp"},
		{spotTWAPNewOrderRate, sapiUIDPool, 3000, "POST /sapi/v1/algo/spot/newOrderTwap"},
		// Fiat, Pay, Rebate and NFT
		{fiatDepositWithdrawHistRate, sapiUIDPool, 45000, "GET /sapi/v1/fiat/orders"},
		{payTradeHistoryRate, sapiUIDPool, 3000, "GET /sapi/v1/pay/transactions"},
		{spotRebateHistoryRate, sapiUIDPool, 12000, "GET /sapi/v1/rebate/taxQuery"},
		{nftRate, sapiUIDPool, 3000, "GET /sapi/v1/nft endpoints, no longer documented"},
		// Binance Link (CAAS)
		{createAPIKeyForSubAccountRate, brokerCreateAPIKeyPool, 1, "POST /sapi/v1/broker/subAccountApi: IP 8, 1 key per sub-account a second"},
		{deleteAPIKeyForSubAccountRate, brokerDeleteAPIKeyPool, 1, "DELETE /sapi/v1/broker/subAccountApi: 1 key per sub-account a second"},
		{updateIPRestrictionForSubAccountAPIKeyRate, sapiUIDPool, 3000, "POST /sapi/v2/broker/subAccountApi/ipRestriction"},
		{brokerSubAccountDepositHistoryRate, sapiIPPool, 10, "GET /sapi/v1/broker/subAccount/depositHist"},
		{brokerSubAccountFuturesAssetInfoRate, sapiUIDPool, 60, "GET /sapi/v3/broker/subAccount/futuresSummary"},
		{brokerSubAccountSpotAssetInfoRate, sapiUIDPool, 3000, "GET /sapi/v1/broker/subAccount/spotSummary"},
		{brokerFuturesTransferRate, brokerFuturesTransferPool, 1, "POST /sapi/v1/broker/transfer/futures: 5000 a minute"},
		{brokerSubAccountCommissionRate, sapiUIDPool, 4, "POST /sapi/v1/broker/subAccountApi/commission"},
		// USDⓈ-M futures /fapi
		{uFuturesDefaultRate, derivativesIPPool, 1, "weight-1 /fapi endpoints, and the single-symbol forms of those weighted by symbol"},
		{uFuturesAccountInformationRate, derivativesIPPool, 5, "GET /fapi/v3/balance, GET /fapi/v2/account, GET /fapi/v3/account"},
		{uFuturesTradingStatusAllRate, derivativesIPPool, 10, "GET /fapi/v1/apiTradingStatus without symbol"},
		{uFuturesMultiAssetMarginRate, derivativesIPPool, 30, "GET /fapi/v1/multiAssetsMargin"},
		{uFuturesPositionModeRate, derivativesIPPool, 30, "GET /fapi/v1/positionSide/dual"},
		{uFuturesDownloadIDRate, derivativesIPPool, 1000, "GET /fapi/v1/order/asyn, /trade/asyn and /income/asyn"},
		{uFuturesDownloadLinkRate, derivativesIPPool, 10, "GET /fapi/v1/order/asyn/id, /trade/asyn/id and /income/asyn/id"},
		{uFuturesIncomeHistoryRate, derivativesIPPool, 30, "GET /fapi/v1/income"},
		{uFuturesUserCommissionRate, derivativesIPPool, 20, "GET /fapi/v1/commissionRate"},
		{uFuturesAggregateTradesRate, derivativesIPPool, 20, "GET /fapi/v1/aggTrades"},
		{uFuturesFundingRateHistoryRate, uFuturesFundingPool, 1, "GET /fapi/v1/fundingRate: shares 500 per 5 minutes with fundingInfo"},
		{uFuturesFundingInfoRate, uFuturesFundingPool, 1, "GET /fapi/v1/fundingInfo: weight 0, shares 500 per 5 minutes with fundingRate"},
		{uFuturesKline100Rate, derivativesIPPool, 1, "USDⓈ-M klines, limit [1,100)"},
		{uFuturesKline500Rate, derivativesIPPool, 2, "USDⓈ-M klines, limit [100,500)"},
		{uFuturesKline1000Rate, derivativesIPPool, 5, "USDⓈ-M klines, limit [500,1000] and the default"},
		{uFuturesKlineMaxRate, derivativesIPPool, 10, "USDⓈ-M klines, limit above 1000"},
		{uFuturesMarkPriceAllRate, derivativesIPPool, 10, "GET /fapi/v1/premiumIndex without symbol"},
		{uFuturesAssetIndexAllRate, derivativesIPPool, 10, "GET /fapi/v1/assetIndex without symbol"},
		{uFuturesHistoricalTradesRate, derivativesIPPool, 200, "GET /fapi/v1/historicalTrades"},
		{uFuturesOrderbook50Rate, derivativesIPPool, 2, "GET /fapi/v1/depth limit 5-50"},
		{uFuturesOrderbook100Rate, derivativesIPPool, 5, "GET /fapi/v1/depth limit 100"},
		{uFuturesOrderbook500Rate, derivativesIPPool, 10, "GET /fapi/v1/depth limit 500 and the default"},
		{uFuturesOrderbook1000Rate, derivativesIPPool, 20, "GET /fapi/v1/depth limit 1000"},
		{uFuturesIndexConstituentsRate, derivativesIPPool, 2, "GET /fapi/v1/constituents"},
		{uFuturesRecentTradesRate, derivativesIPPool, 5, "GET /fapi/v1/trades"},
		{uFuturesRPIOrderbookRate, derivativesIPPool, 20, "GET /fapi/v1/rpiDepth"},
		{uFuturesBookTickerRate, derivativesIPPool, 2, "GET /fapi/v1/ticker/bookTicker, one symbol"},
		{uFuturesOrderbookTickerAllRate, derivativesIPPool, 5, "GET /fapi/v1/ticker/bookTicker, all symbols"},
		{uFuturesSymbolPriceAllRate, derivativesIPPool, 2, "GET /fapi/v2/ticker/price, all symbols"},
		{uFuturesTicker24HourAllRate, derivativesIPPool, 40, "GET /fapi/v1/ticker/24hr, all symbols"},
		{uFuturesAccountTradeListRate, derivativesIPPool, 5, "GET /fapi/v1/userTrades"},
		{uFuturesGetAllOrdersRate, derivativesIPPool, 5, "GET /fapi/v1/allOrders"},
		{uFuturesCountdownCancelRate, derivativesIPPool, 10, "POST /fapi/v1/countdownCancelAll"},
		{uFuturesOrdersDefaultRate, derivativesOrderPool, 1, "POST and PUT /fapi/v1/order, POST /fapi/v1/algoOrder: 1 order"},
		{uFuturesBatchOrdersRate, derivativesOrderPool, 5, "POST and PUT /fapi/v1/batchOrders: 5 on the 10 second order window"},
		{uFuturesOpenAlgoOrdersAllRate, derivativesIPPool, 40, "GET /fapi/v1/openAlgoOrders, all symbols"},
		{uFuturesGetAllOpenOrdersRate, derivativesIPPool, 40, "GET /fapi/v1/openOrders, all symbols"},
		{uFuturesADLQuantileRate, derivativesIPPool, 5, "GET /fapi/v1/adlQuantile"},
		{uFuturesPositionRiskRate, derivativesIPPool, 5, "GET /fapi/v2/positionRisk, GET /fapi/v3/positionRisk"},
		{uFuturesAllAlgoOrdersRate, derivativesIPPool, 5, "GET /fapi/v1/allAlgoOrders"},
		{uFuturesSymbolForceOrdersRate, derivativesIPPool, 20, "GET /fapi/v1/forceOrders, one symbol"},
		{uFuturesAllForceOrdersRate, derivativesIPPool, 50, "GET /fapi/v1/forceOrders, all symbols"},
		{uFuturesAPIReferralRate, derivativesIPPool, 100, "the OMS Toolkit /fapi/v1/apiReferral endpoints"},
		// /futures/data on the USDⓈ-M and COIN-M hosts
		{futuresDataRate, futuresDataPool, 1, "GET /futures/data/* on both hosts: 1000 requests per 5 minutes"},
		// COIN-M futures /dapi
		{cFuturesDefaultRate, derivativesIPPool, 1, "weight-1 /dapi endpoints, and the single-symbol forms of those weighted by symbol"},
		{cFuturesAccountInformationRate, derivativesIPPool, 5, "GET /dapi/v1/account"},
		{cFuturesIncomeHistoryRate, derivativesIPPool, 20, "GET /dapi/v1/income"},
		{cFuturesAggregateTradesRate, derivativesIPPool, 20, "GET /dapi/v1/aggTrades"},
		{cFuturesKline100Rate, derivativesIPPool, 1, "COIN-M klines, limit [1,100)"},
		{cFuturesKline500Rate, derivativesIPPool, 2, "COIN-M klines, limit [100,500)"},
		{cFuturesKline1000Rate, derivativesIPPool, 5, "COIN-M klines, limit [500,1000] and the default"},
		{cFuturesKlineMaxRate, derivativesIPPool, 10, "COIN-M klines, limit above 1000"},
		{cFuturesIndexMarkPriceRate, derivativesIPPool, 10, "GET /dapi/v1/premiumIndex"},
		{cFuturesHistoricalTradesRate, derivativesIPPool, 200, "GET /dapi/v1/historicalTrades"},
		{cFuturesOrderbook50Rate, derivativesIPPool, 2, "GET /dapi/v1/depth limit 5-50"},
		{cFuturesOrderbook100Rate, derivativesIPPool, 5, "GET /dapi/v1/depth limit 100"},
		{cFuturesOrderbook500Rate, derivativesIPPool, 10, "GET /dapi/v1/depth limit 500 and the default"},
		{cFuturesOrderbook1000Rate, derivativesIPPool, 20, "GET /dapi/v1/depth limit 1000"},
		{cFuturesRecentTradesRate, derivativesIPPool, 5, "GET /dapi/v1/trades"},
		{cFuturesBookTickerRate, derivativesIPPool, 2, "GET /dapi/v1/ticker/bookTicker, one symbol"},
		{cFuturesOrderbookTickerAllRate, derivativesIPPool, 5, "GET /dapi/v1/ticker/bookTicker, all symbols"},
		{cFuturesSymbolPriceAllRate, derivativesIPPool, 2, "GET /dapi/v1/ticker/price, all symbols"},
		{cFuturesTicker24HourAllRate, derivativesIPPool, 40, "GET /dapi/v1/ticker/24hr, all symbols"},
		{cFuturesAccountTradeListRate, derivativesIPPool, 5, "GET /dapi/v1/userTrades"},
		{cFuturesAllOrdersRate, derivativesIPPool, 5, "GET /dapi/v1/allOrders"},
		{cFuturesCountdownCancelRate, derivativesIPPool, 10, "POST /dapi/v1/countdownCancelAll"},
		{cFuturesBatchOrdersRate, derivativesOrderPool, 5, "POST /dapi/v1/batchOrders: up to 5 orders"},
		{cFuturesOrdersDefaultRate, derivativesOrderPool, 1, "POST /dapi/v1/order: 1 order"},
		{cFuturesGetAllOpenOrdersRate, derivativesIPPool, 40, "GET /dapi/v1/openOrders, all symbols"},
		{cFuturesADLQuantileRate, derivativesIPPool, 5, "GET /dapi/v1/adlQuantile"},
		{cFuturesSymbolForceOrdersRate, derivativesIPPool, 20, "GET /dapi/v1/forceOrders, one symbol"},
		{cFuturesAllForceOrdersRate, derivativesIPPool, 50, "GET /dapi/v1/forceOrders, all symbols"},
		// Options /eapi
		{optionsDefaultRate, optionsIPPool, 1, "weight-1 /eapi endpoints; GET /eapi/v1/openInterest is documented at 0"},
		{optionsMarginAccountInfoRate, optionsIPPool, 3, "GET /eapi/v1/marginAccount"},
		{optionsDownloadIDForOptionTransactionHistoryRate, optionsIPPool, 5, "GET /eapi/v1/income/asyn, no longer documented"},
		{optionsGetTransHistoryDownloadLinkByIDRate, optionsIPPool, 5, "GET /eapi/v1/income/asyn/id, no longer documented"},
		{optionsHistoricalExerciseRecordsRate, optionsIPPool, 3, "GET /eapi/v1/exerciseHistory"},
		{optionsMarkPriceRate, optionsIPPool, 5, "GET /eapi/v1/mark"},
		{optionsOrderbook50Rate, optionsIPPool, 1, "GET /eapi/v1/depth limit 5-50"},
		{optionsOrderbook100Rate, optionsIPPool, 5, "GET /eapi/v1/depth limit 100 and the default"},
		{optionsOrderbook500Rate, optionsIPPool, 10, "GET /eapi/v1/depth limit 500"},
		{optionsOrderbook1000Rate, optionsIPPool, 20, "GET /eapi/v1/depth limit 1000"},
		{optionsRecentTradesRate, optionsIPPool, 5, "GET /eapi/v1/trades"},
		{optionsAllTickerPriceStatisticsRate, optionsIPPool, 40, "GET /eapi/v1/ticker, all symbols"},
		{optionsAutoCancelAllOpenOrdersHeartbeatRate, optionsIPPool, 10, "POST /eapi/v1/countdownCancelAllHeartBeat"},
		{optionsAccountTradeListRate, optionsIPPool, 5, "GET /eapi/v1/userTrades"},
		{optionsCancelAllByUnderlyingRate, optionsIPPool, 5, "DELETE /eapi/v1/allOpenOrdersByUnderlying"},
		{optionsBatchOrderRate, optionsOrderPool, 5, "POST /eapi/v1/batchOrders: weight 5"},
		{optionsCancelBatchOrdersRate, optionsIPPool, 5, "DELETE /eapi/v1/batchOrders"},
		{optionsDefaultOrderRate, optionsOrderPool, 1, "POST /eapi/v1/order: 1 order"},
		{optionsPositionInformationRate, optionsIPPool, 5, "GET /eapi/v1/position"},
		{optionsAllQueryOpenOrdersRate, optionsIPPool, 40, "GET /eapi/v1/openOrders, all symbols"},
		{optionsGetOrderHistoryRate, optionsIPPool, 3, "GET /eapi/v1/historyOrders"},
		{optionsUserExerciseRecordRate, optionsIPPool, 5, "GET /eapi/v1/exerciseRecord"},
		// Portfolio margin /papi
		{pmDefaultRate, portfolioMarginIPPool, 1, "weight-1 /papi endpoints, and the single-symbol forms of those weighted by symbol"},
		{pmGetAccountBalancesRate, portfolioMarginIPPool, 20, "GET /papi/v1/balance"},
		{pmGetAccountInformationRate, portfolioMarginIPPool, 20, "GET /papi/v1/account"},
		{pmBNBTransferRate, pmBNBTransferPool, 1, "POST /papi/v1/bnb-transfer: IP 750, 10 calls per 10 minutes"},
		{pmGetAutoRepayFuturesStatusRate, portfolioMarginIPPool, 30, "GET /papi/v1/repay-futures-switch"},
		{pmChangeAutoRepayFuturesStatusRate, portfolioMarginIPPool, 750, "POST /papi/v1/repay-futures-switch"},
		{pmGetCMCurrentPositionModeRate, portfolioMarginIPPool, 30, "GET /papi/v1/cm/positionSide/dual"},
		{pmGetUMCurrentPositionModeRate, portfolioMarginIPPool, 30, "GET /papi/v1/um/positionSide/dual"},
		{pmFundAutoCollectionRate, portfolioMarginIPPool, 750, "POST /papi/v1/auto-collection"},
		{pmFundCollectionByAssetRate, portfolioMarginIPPool, 30, "POST /papi/v1/asset-collection"},
		{pmGetCMAccountDetailRate, portfolioMarginIPPool, 5, "GET /papi/v1/cm/account"},
		{pmGetCMIncomeHistoryRate, portfolioMarginIPPool, 30, "GET /papi/v1/cm/income"},
		{pmGetUMAccountDetailRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/account"},
		{pmGetUMIncomeHistoryRate, portfolioMarginIPPool, 30, "GET /papi/v1/um/income"},
		{pmGetCMUserCommissionRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/commissionRate"},
		{pmGetUMUserCommissionRate, portfolioMarginIPPool, 20, "GET /papi/v1/um/commissionRate"},
		{pmMarginMaxBorrowRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/maxBorrowable"},
		{pmUMTradingQuantitativeRulesIndicatorsRate, portfolioMarginIPPool, 10, "GET /papi/v1/um/apiTradingStatus without symbol"},
		{pmGetMarginLoanRecordRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/marginLoan"},
		{pmGetMarginMaxWithdrawalRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/maxWithdraw"},
		{pmGetMarginRepayRecordRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/repayLoan"},
		{pmGetPortfolioMarginNegativeBalanceInterestHistoryRate, portfolioMarginIPPool, 50, "GET /papi/v1/portfolio/interest-history"},
		{pmGetUMPositionInformationRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/positionRisk"},
		{pmNegativeBalanceExchangeRecordRate, portfolioMarginIPPool, 100, "GET /papi/v1/portfolio/negative-balance-exchange-record"},
		{pmRepayFuturesNegativeBalanceRate, portfolioMarginIPPool, 750, "POST /papi/v1/repay-futures-negative-balance"},
		{pmOrderRate, portfolioMarginOrderPool, 1, "POST /papi/v1 um, cm and margin orders, cm conditional and um algo orders: 1 order"},
		{pmCancelMarginAccountOpenOrdersOnSymbolRate, portfolioMarginIPPool, 5, "DELETE /papi/v1/margin/allOpenOrders"},
		{pmGetMarginAccountOCORate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/orderList"},
		{pmCancelMarginAccountOCORate, portfolioMarginIPPool, 2, "DELETE /papi/v1/margin/orderList"},
		{pmGetMarginAccountOrderRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/order"},
		{pmCancelMarginAccountOrderRate, portfolioMarginIPPool, 2, "DELETE /papi/v1/margin/order"},
		{pmGetCMAccountTradeListWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/userTrades with symbol"},
		{pmGetCMAccountTradeListWithPairRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/userTrades with pair"},
		{pmGetCMPositionADLQuantileEstimationRate, portfolioMarginIPPool, 5, "GET /papi/v1/cm/adlQuantile"},
		{pmMarginAccountLoanAndRepayRate, portfolioMarginIPPool, 100, "POST /papi/v1/marginLoan, POST /papi/v1/repayLoan"},
		{pmOCOOrderRate, portfolioMarginOrderPool, 2, "POST /papi/v1/margin/order/oco: 2 orders"},
		{pmGetMarginAccountTradeListRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/myTrades"},
		{pmAllCMConditionalOrderWithoutSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/conditional/allOrders without symbol"},
		{pmAllCMOrderWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/allOrders with symbol"},
		{pmAllCMOrderWithPairRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/allOrders with pair"},
		{pmAllCMOpenConditionalOrdersWithoutSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/conditional/openOrders without symbol"},
		{pmRetrieveAllCMOpenOrdersForAllSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/openOrders without symbol"},
		{pmRetrieveAllUMOpenOrdersForAllSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/um/openOrders without symbol"},
		{pmAllMarginAccountOrdersRate, portfolioMarginIPPool, 100, "GET /papi/v1/margin/allOrders"},
		{pmGetAllUMOrdersRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/allOrders"},
		{pmGetUMAlgoOrderHistoryRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/algo/allAlgoOrders"},
		{pmCurrentMarginOpenOrderRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/openOrders"},
		{pmGetMarginAccountsAllOCOOrdersRate, portfolioMarginIPPool, 100, "GET /papi/v1/margin/allOrderList"},
		{pmGetMarginAccountsOpenOCOOrdersRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/openOrderList"},
		{pmGetUserCMForceOrdersWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/forceOrders with symbol"},
		{pmGetUserCMForceOrdersWithoutSymbolRate, portfolioMarginIPPool, 50, "GET /papi/v1/cm/forceOrders without symbol"},
		{pmGetUserUMForceOrdersWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/um/forceOrders with symbol"},
		{pmGetUserUMForceOrdersWithoutSymbolRate, portfolioMarginIPPool, 50, "GET /papi/v1/um/forceOrders without symbol"},
		{pmGetUMAccountTradeListRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/userTrades"},
		{pmGetUMPositionADLQuantileEstimationRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/adlQuantile"},
		{pmAPIReferralRate, portfolioMarginIPPool, 100, "the OMS Toolkit /papi/v1/apiReferral endpoints"},
	}
	seen := make(map[request.EndpointLimit]bool, len(rows))
	for _, tc := range rows {
		require.Falsef(t, seen[tc.epl], "%s must have one row", tc.endpoints)
		seen[tc.epl] = true
		assert.Equalf(t, endpointCost{tc.pool, tc.weight}, endpointCosts[tc.epl], "%s should cost its documented weight on its pool", tc.endpoints)
	}
	assert.Equal(t, len(rows), len(endpointCosts), "endpointCosts should hold exactly the documented endpoint limits")
}

// endpointLimitNames returns the names of the endpoint limit constants in declaration order
func endpointLimitNames(t *testing.T) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "ratelimit.go", nil, 0)
	require.NoError(t, err, "ratelimit.go must parse")
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST || len(gen.Specs) == 0 {
			continue
		}
		first, ok := gen.Specs[0].(*ast.ValueSpec)
		if !ok {
			continue
		}
		if sel, ok := first.Type.(*ast.SelectorExpr); !ok || sel.Sel.Name != "EndpointLimit" {
			continue
		}
		names := make([]string, 0, len(gen.Specs))
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			require.True(t, ok, "every endpoint limit declaration must be a value spec")
			for _, name := range valueSpec.Names {
				names = append(names, name.Name)
			}
		}
		return names
	}
	require.FailNow(t, "ratelimit.go must declare the endpoint limits")
	return nil
}

func TestEndpointLimitsComplete(t *testing.T) {
	t.Parallel()
	names := endpointLimitNames(t)
	require.Equal(t, int(wsConnectionMessageRate-spotDefaultRate)+1, len(names), "the endpoint limits must run from spotDefaultRate to wsConnectionMessageRate")
	require.Equal(t, "wsConnectionMessageRate", names[len(names)-1], "wsConnectionMessageRate must be the last endpoint limit")
	assert.Greater(t, spotDefaultRate, request.UnAuth, "endpoint limits should not alias the request package's generic limits")

	definitions := GetRateLimits()
	for i, name := range names[:len(names)-1] {
		epl := spotDefaultRate + request.EndpointLimit(i)
		cost, ok := endpointCosts[epl]
		if !assert.Truef(t, ok, "%s should have a cost", name) {
			continue
		}
		if assert.NotNilf(t, definitions[epl], "%s should have a limiter", name) {
			assert.Equalf(t, cost.weight, definitions[epl].Weight(), "%s should carry its weight", name)
		}
	}
	limiter, ok := definitions[wsConnectionMessageRate]
	assert.True(t, ok, "wsConnectionMessageRate should be defined")
	assert.Nil(t, limiter, "wsConnectionMessageRate should leave messages to the connection's limiter")
	assert.Equal(t, len(names), len(definitions), "GetRateLimits should define every endpoint limit and nothing else")
}

func TestEndpointLimiters(t *testing.T) {
	t.Parallel()
	const (
		ipDefault request.EndpointLimit = iota + 1
		ipDefaultAlias
		ipHeavy
		ipHeavySameWeight
		uidDefault
		uidHeavy
		spotLight
		spotHeavy
	)
	limiters := endpointLimiters(map[request.EndpointLimit]endpointCost{
		ipDefault:         {sapiIPPool, 1},
		ipDefaultAlias:    {sapiIPPool, 1},
		ipHeavy:           {sapiIPPool, 10},
		ipHeavySameWeight: {sapiIPPool, 10},
		uidDefault:        {sapiUIDPool, 1},
		uidHeavy:          {sapiUIDPool, 60},
		spotLight:         {spotIPPool, 1},
		spotHeavy:         {spotIPPool, 20},
	})
	assert.Same(t, limiters[ipDefault], limiters[ipDefaultAlias], "weight-1 /sapi IP limits should share their pool's limiter")
	assert.NotSame(t, limiters[ipHeavy], limiters[ipHeavySameWeight], "heavy /sapi limits should not share a limiter, even at the same weight")
	assert.NotSame(t, limiters[ipDefault], limiters[ipHeavy], "a heavy /sapi limit should not share the weight-1 limiter")
	assert.NotSame(t, limiters[ipDefault], limiters[uidDefault], "the IP and UID pools should not share a limiter")
	assert.NotSame(t, limiters[uidDefault], limiters[uidHeavy], "a heavy /sapi UID limit should not share the weight-1 limiter")
	assert.Same(t, limiters[spotLight], limiters[spotHeavy], "limits of a shared pool should share its limiter whatever their weight")
}

func TestSAPIEndpointLimiters(t *testing.T) {
	t.Parallel()
	limiters := endpointLimiters(endpointCosts)
	owners := make(map[*rate.Limiter][]request.EndpointLimit, len(limiters))
	for epl, limiter := range limiters {
		owners[limiter] = append(owners[limiter], epl)
	}
	poolLimiters := make(map[rateLimitPool]*rate.Limiter)
	for epl, cost := range endpointCosts {
		limiter := limiters[epl]
		budget := poolBudgets[cost.pool]
		assert.InDeltaf(t, float64(budget.limit)/budget.interval.Seconds(), float64(limiter.Limit()), 1e-9, "endpoint limit %d should have its pool's budget", epl)
		if cost.pool.perEndpoint() && cost.weight > 1 {
			assert.Lenf(t, owners[limiter], 1, "/sapi endpoint limit %d should have a limiter of its own", epl)
			continue
		}
		if poolLimiters[cost.pool] == nil {
			poolLimiters[cost.pool] = limiter
		}
		assert.Samef(t, poolLimiters[cost.pool], limiter, "endpoint limit %d should share its pool's limiter", epl)
	}
	assert.NotSame(t, limiters[getAllConvertPairsRate], limiters[getOrderQuantityPrecisionPerAssetRate], "two heavy /sapi IP limits should have separate limiters")
	assert.NotSame(t, limiters[acceptQuoteRate], limiters[placeLimitOrderRate], "two heavy /sapi UID limits of the same weight should have separate limiters")
	assert.Same(t, poolLimiters[sapiIPPool], limiters[sapiDefaultRate], "sapiDefaultRate should keep its pool's shared limiter")
	assert.Same(t, poolLimiters[sapiUIDPool], limiters[sapiUIDDefaultRate], "sapiUIDDefaultRate should keep its pool's shared limiter")
	assert.NotSame(t, limiters[sapiDefaultRate], limiters[sapiUIDDefaultRate], "the /sapi IP and UID defaults should not share a limiter")
	assert.Same(t, limiters[spotDefaultRate], limiters[spotExchangeInfoRate], "spot limits should share the spot IP limiter")
}

func TestEndpointLimitHelpers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		limit  request.EndpointLimit
		pool   rateLimitPool
		weight request.Weight
	}{
		{"spot depth by default", spotOrderbookLimit(0), spotIPPool, 5},
		{"spot depth 1", spotOrderbookLimit(1), spotIPPool, 5},
		{"spot depth 100", spotOrderbookLimit(100), spotIPPool, 5},
		{"spot depth 101", spotOrderbookLimit(101), spotIPPool, 25},
		{"spot depth 500", spotOrderbookLimit(500), spotIPPool, 25},
		{"spot depth 501", spotOrderbookLimit(501), spotIPPool, 50},
		{"spot depth 1000", spotOrderbookLimit(1000), spotIPPool, 50},
		{"spot depth 1001", spotOrderbookLimit(1001), spotIPPool, 250},
		{"spot depth 5000", spotOrderbookLimit(5000), spotIPPool, 250},
		{"spot open orders of a symbol", spotOpenOrdersLimit("BTCUSDT"), spotIPPool, 6},
		{"spot open orders of every symbol", spotOpenOrdersLimit(""), spotIPPool, 80},
		{"spot ticker for 1 symbol", spotTickerSymbolsLimit(1), spotIPPool, 4},
		{"spot ticker for 5 symbols", spotTickerSymbolsLimit(5), spotIPPool, 20},
		{"spot ticker for 50 symbols", spotTickerSymbolsLimit(50), spotIPPool, 200},
		{"spot ticker for 100 symbols", spotTickerSymbolsLimit(100), spotIPPool, 200},
		{"spot 24hr ticker for every symbol", spotTicker24HourLimit(0), spotIPPool, 80},
		{"spot 24hr ticker for 1 symbol", spotTicker24HourLimit(1), spotIPPool, 2},
		{"spot 24hr ticker for 20 symbols", spotTicker24HourLimit(20), spotIPPool, 2},
		{"spot 24hr ticker for 21 symbols", spotTicker24HourLimit(21), spotIPPool, 40},
		{"spot 24hr ticker for 100 symbols", spotTicker24HourLimit(100), spotIPPool, 40},
		{"spot 24hr ticker for 101 symbols", spotTicker24HourLimit(101), spotIPPool, 80},
		{"spot trades without an order ID", spotAccountTradeListLimit(0), spotIPPool, 20},
		{"spot trades of an order", spotAccountTradeListLimit(12345), spotIPPool, 5},
		{"margin order without a side effect", marginNewOrderLimit(""), sapiUIDPool, 6},
		{"margin order NO_SIDE_EFFECT", marginNewOrderLimit("NO_SIDE_EFFECT"), sapiUIDPool, 6},
		{"margin order AUTO_REPAY", marginNewOrderLimit("AUTO_REPAY"), sapiUIDPool, 6},
		{"margin order MARGIN_BUY", marginNewOrderLimit("MARGIN_BUY"), sapiUIDPool, 1500},
		{"margin order AUTO_BORROW_REPAY", marginNewOrderLimit("AUTO_BORROW_REPAY"), sapiUIDPool, 1500},
		{"USDⓈ-M klines by default", uFuturesKlineLimit(0), derivativesIPPool, 5},
		{"USDⓈ-M klines 1", uFuturesKlineLimit(1), derivativesIPPool, 1},
		{"USDⓈ-M klines 99", uFuturesKlineLimit(99), derivativesIPPool, 1},
		{"USDⓈ-M klines 100", uFuturesKlineLimit(100), derivativesIPPool, 2},
		{"USDⓈ-M klines 499", uFuturesKlineLimit(499), derivativesIPPool, 2},
		{"USDⓈ-M klines 500", uFuturesKlineLimit(500), derivativesIPPool, 5},
		{"USDⓈ-M klines 1000", uFuturesKlineLimit(1000), derivativesIPPool, 5},
		{"USDⓈ-M klines 1001", uFuturesKlineLimit(1001), derivativesIPPool, 10},
		{"USDⓈ-M klines 1500", uFuturesKlineLimit(1500), derivativesIPPool, 10},
		{"USDⓈ-M depth by default", uFuturesOrderbookLimit(0), derivativesIPPool, 10},
		{"USDⓈ-M depth 5", uFuturesOrderbookLimit(5), derivativesIPPool, 2},
		{"USDⓈ-M depth 50", uFuturesOrderbookLimit(50), derivativesIPPool, 2},
		{"USDⓈ-M depth 100", uFuturesOrderbookLimit(100), derivativesIPPool, 5},
		{"USDⓈ-M depth 500", uFuturesOrderbookLimit(500), derivativesIPPool, 10},
		{"USDⓈ-M depth 1000", uFuturesOrderbookLimit(1000), derivativesIPPool, 20},
		{"COIN-M klines by default", cFuturesKlineLimit(0), derivativesIPPool, 5},
		{"COIN-M klines 1", cFuturesKlineLimit(1), derivativesIPPool, 1},
		{"COIN-M klines 99", cFuturesKlineLimit(99), derivativesIPPool, 1},
		{"COIN-M klines 100", cFuturesKlineLimit(100), derivativesIPPool, 2},
		{"COIN-M klines 499", cFuturesKlineLimit(499), derivativesIPPool, 2},
		{"COIN-M klines 500", cFuturesKlineLimit(500), derivativesIPPool, 5},
		{"COIN-M klines 1000", cFuturesKlineLimit(1000), derivativesIPPool, 5},
		{"COIN-M klines 1001", cFuturesKlineLimit(1001), derivativesIPPool, 10},
		{"COIN-M klines 1500", cFuturesKlineLimit(1500), derivativesIPPool, 10},
		{"COIN-M depth by default", cFuturesOrderbookLimit(0), derivativesIPPool, 10},
		{"COIN-M depth 5", cFuturesOrderbookLimit(5), derivativesIPPool, 2},
		{"COIN-M depth 50", cFuturesOrderbookLimit(50), derivativesIPPool, 2},
		{"COIN-M depth 100", cFuturesOrderbookLimit(100), derivativesIPPool, 5},
		{"COIN-M depth 500", cFuturesOrderbookLimit(500), derivativesIPPool, 10},
		{"COIN-M depth 1000", cFuturesOrderbookLimit(1000), derivativesIPPool, 20},
		{"options depth by default", optionsOrderbookLimit(0), optionsIPPool, 5},
		{"options depth 5", optionsOrderbookLimit(5), optionsIPPool, 1},
		{"options depth 50", optionsOrderbookLimit(50), optionsIPPool, 1},
		{"options depth 100", optionsOrderbookLimit(100), optionsIPPool, 5},
		{"options depth 500", optionsOrderbookLimit(500), optionsIPPool, 10},
		{"options depth 1000", optionsOrderbookLimit(1000), optionsIPPool, 20},
	} {
		assert.Equalf(t, endpointCost{tc.pool, tc.weight}, endpointCosts[tc.limit], "%s should cost its documented weight", tc.name)
	}
	// The rolling window and trading day tickers cost 4 a symbol up to 200; a tier may round up but never under-charge
	for symbols := 1; symbols <= 100; symbols++ {
		assert.GreaterOrEqualf(t, endpointCosts[spotTickerSymbolsLimit(symbols)].weight, request.Weight(min(4*symbols, 200)), "a ticker request for %d symbols should cost at least its documented weight", symbols)
	}
}

func TestEndpointLimitArguments(t *testing.T) {
	t.Parallel()
	// The argument index of the endpoint limit of each sender whose limit must be a weighted Binance limit
	senders := map[string]int{
		"SendHTTPRequest":        3,
		"SendAPIKeyHTTPRequest":  4,
		"SendAuthHTTPRequest":    5,
		"SendWsAPIRequest":       1,
		"SendSignedWsAPIRequest": 1,
	}
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "ratelimit.go" {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if pkg, ok := n.X.(*ast.Ident); ok && pkg.Name == "request" {
					switch n.Sel.Name {
					case "Unset", "Auth", "UnAuth":
						assert.Failf(t, "generic endpoint limit", "%s should use a Binance endpoint limit instead of request.%s", fset.Position(n.Pos()), n.Sel.Name)
					}
				}
			case *ast.CallExpr:
				sel, ok := n.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if i, ok := senders[sel.Sel.Name]; ok && i < len(n.Args) {
					if arg, ok := n.Args[i].(*ast.Ident); ok && arg.Name == "wsConnectionMessageRate" {
						assert.Failf(t, "unweighted endpoint limit", "%s should pass a weighted endpoint limit to %s", fset.Position(n.Pos()), sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
}

func TestEndpointLimitsUsed(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	used := make(map[string]bool)
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		for _, decl := range f.Decls {
			if _, isFunc := decl.(*ast.FuncDecl); !isFunc && file == "ratelimit.go" {
				// ratelimit.go declares and prices every endpoint limit, so only its helpers count as uses
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok {
					used[ident.Name] = true
				}
				return true
			})
		}
	}
	for _, name := range endpointLimitNames(t) {
		assert.Truef(t, used[name], "%s should be used by an endpoint", name)
	}
}

func TestRateLimitsWithRequester(t *testing.T) {
	t.Parallel()
	r, err := request.New("binanceRateLimitTest", &http.Client{}, request.WithLimiter(GetRateLimits()))
	require.NoError(t, err, "request.New must not error")
	for _, epl := range []request.EndpointLimit{request.Unset, request.Auth, request.UnAuth} {
		assert.ErrorIsf(t, r.InitiateRateLimit(t.Context(), epl), common.ErrNilPointer, "generic endpoint limit %d should find no Binance limiter", epl)
	}
	require.NoError(t, r.InitiateRateLimit(t.Context(), spotDefaultRate), "a weight-1 request must not wait on a fresh pool")
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, r.InitiateRateLimit(ctx, spotOrderbookDepth5000Rate), context.DeadlineExceeded, "a weight-250 request should not fit within a millisecond of a 6000 a minute pool")
}
