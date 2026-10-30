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
		{spotIPPool, time.Minute, 5400, false},
		{spotCancelPool, time.Minute, 600, false},
		{spotOrderPool, 10 * time.Second, 100, false},
		{sapiIPPool, time.Minute, 12000, true},
		{sapiUIDPool, time.Minute, 180000, true},
		{derivativesIPPool, time.Minute, 1800, false},
		{derivativesCancelPool, time.Minute, 450, false},
		{derivativesHeartbeatPool, time.Minute, 150, false},
		{derivativesOrderPool, time.Minute, 1200, false},
		{futuresDataPool, 5 * time.Minute, 1000, false},
		{usdtMarginedFuturesFundingPool, 5 * time.Minute, 500, false},
		{optionsIPPool, time.Minute, 240, false},
		{optionsCancelPool, time.Minute, 100, false},
		{optionsHeartbeatPool, time.Minute, 60, false},
		{optionsOrderPool, time.Minute, 100, false},
		{portfolioMarginIPPool, time.Minute, 6000, false},
		{portfolioMarginOrderPool, time.Minute, 1200, false},
		{marginMaxLeveragePool, time.Minute, 1, false},
		{withdrawalHistoryPool, time.Second, 10, false},
		{subAccountFuturesTransferPool, time.Minute, 2000, false},
		{portfolioMarginProBNBTransferPool, 10 * time.Minute, 2, false},
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
		{portfolioMarginBNBTransferPool, 10 * time.Minute, 10, false},
	}
	listed := make(map[rateLimitPool]bool, len(rows))
	for _, tc := range rows {
		listed[tc.pool] = true
		assert.Equalf(t, rateLimitBudget{tc.interval, tc.limit}, poolBudgets[tc.pool], "pool %d should have its documented budget", tc.pool)
		assert.Equalf(t, tc.perEndpoint, tc.pool.perEndpoint(), "pool %d should only budget endpoints separately where Binance does", tc.pool)
	}
	for pool := spotIPPool; pool <= portfolioMarginBNBTransferPool; pool++ {
		assert.Truef(t, listed[pool], "pool %d should have a documented budget row", pool)
	}
	assert.Equal(t, len(rows), len(poolBudgets), "poolBudgets should hold exactly the documented pools")

	// A host's IP, cancel and heartbeat pools share its documented IP limit, their bursts included
	for _, tc := range []struct {
		pools []rateLimitPool
		limit int
	}{
		{[]rateLimitPool{spotIPPool, spotCancelPool}, 6000},
		{[]rateLimitPool{derivativesIPPool, derivativesCancelPool, derivativesHeartbeatPool}, 2400},
		{[]rateLimitPool{optionsIPPool, optionsCancelPool, optionsHeartbeatPool}, 400},
	} {
		var shared int
		for _, pool := range tc.pools {
			shared += poolBudgets[pool].limit
		}
		assert.Equalf(t, tc.limit, shared, "pools %v should share their host's IP limit", tc.pools)
	}
	exp := map[rateLimitPool]int{
		spotCancelPool:           100,
		derivativesCancelPool:    200,
		derivativesHeartbeatPool: 50,
		optionsCancelPool:        50,
		optionsHeartbeatPool:     20,
	}
	assert.Equal(t, exp, poolBursts, "only the cancel and heartbeat pools should burst")
	for pool, burst := range poolBursts {
		assert.Lessf(t, burst, poolBudgets[pool].limit, "pool %d should refill part of its budget", pool)
	}

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
		{spotDefaultRate, spotIPPool, 1, "weight-1 /api endpoints: GET /api/v3/time, order tests without commission rates"},
		{spotCancelRate, spotCancelPool, 1, "DELETE /api/v3/order, /api/v3/openOrders and /api/v3/orderList, WS order.cancel, openOrders.cancelAll and orderList.cancel"},
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
		{spotOneCancelsTheOtherOrderRate, spotOrderPool, 2, "POST /api/v3/orderList/oco: 2 orders"},
		{testNewOrderWithCommissionRate, spotIPPool, 20, "POST /api/v3/order/test and /sor/order/test with computeCommissionRates"},
		{getCommissionRate, spotIPPool, 20, "GET /api/v3/account/commission, WS account.commission"},
		{getAllOneCancelsTheOtherOrdersRate, spotIPPool, 20, "GET /api/v3/allOrderList, WS allOrderLists"},
		{spotAllOrdersRate, spotIPPool, 20, "GET /api/v3/allOrders, WS allOrders"},
		{spotAccountInformationRate, spotIPPool, 20, "GET /api/v3/account, WS account.status"},
		{spotOpenOrdersSpecificRate, spotIPPool, 6, "GET /api/v3/openOrders, one symbol"},
		{spotOpenOrdersAllRate, spotIPPool, 80, "GET /api/v3/openOrders, all symbols"},
		{spotCancelAllListingRate, spotCancelPool, 80, "GET /api/v3/openOrders and WS openOrders.status, all symbols, listed to cancel every symbol's orders"},
		{spotOrderQueryRate, spotIPPool, 4, "GET /api/v3/order, WS order.status"},
		{getOneCancelsTheOtherListRate, spotIPPool, 4, "GET /api/v3/orderList, WS orderList.status"},
		{getAllocationsRate, spotIPPool, 20, "GET /api/v3/myAllocations, WS myAllocations"},
		{preventedMatchesRate, spotIPPool, 2, "GET /api/v3/myPreventedMatches by preventedMatchId"},
		{preventedMatchesByOrderIDRate, spotIPPool, 20, "GET /api/v3/myPreventedMatches by orderId"},
		{accountTradeListRate, spotIPPool, 20, "GET /api/v3/myTrades without orderId"},
		{accountTradeListByOrderIDRate, spotIPPool, 5, "GET /api/v3/myTrades with orderId"},
		{getOpenOneCancelsTheOtherListRate, spotIPPool, 6, "GET /api/v3/openOrderList, WS openOrderLists.status"},
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
		{getMarginAccountOneCancelsTheOtherOrderRate, sapiIPPool, 10, "GET /sapi/v1/margin/orderList"},
		{getMarginAccountOrderRate, sapiIPPool, 10, "GET /sapi/v1/margin/order"},
		{marginAccountNewOrderRate, sapiUIDPool, 6, "POST /sapi/v1/margin/order and /order/oco without borrowing"},
		{marginAccountNewOrderBorrowRate, sapiUIDPool, 1500, "POST /sapi/v1/margin/order and /order/oco with MARGIN_BUY or AUTO_BORROW_REPAY"},
		{marginAccountCancelOrderRate, sapiIPPool, 10, "DELETE /sapi/v1/margin/order"},
		{marginManualLiquidationRate, sapiUIDPool, 3000, "POST /sapi/v1/margin/manual-liquidation"},
		{marginCurrentOrderCountUsageRate, sapiIPPool, 20, "GET /sapi/v1/margin/rateLimit/order"},
		{getMarginAccountAllOneCancelsTheOtherRate, sapiIPPool, 200, "GET /sapi/v1/margin/allOrderList"},
		{marginAccountsAllOrdersRate, sapiIPPool, 200, "GET /sapi/v1/margin/allOrders"},
		{marginAccountOpenOneCancelsTheOtherOrdersRate, sapiIPPool, 10, "GET /sapi/v1/margin/openOrderList"},
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
		{withdrawalHistoryRate, withdrawalHistoryPool, 1, "GET /sapi/v1/capital/withdraw/history: 10 requests a second"},
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
		{transferBNBRate, portfolioMarginProBNBTransferPool, 1, "POST /sapi/v1/portfolio/bnb-transfer: IP 1500, 2 calls per 10 minutes"},
		{getAutoRepayFuturesStatusRate, sapiIPPool, 30, "GET /sapi/v1/portfolio/repay-futures-switch"},
		{changeAutoRepayFuturesStatusRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/repay-futures-switch"},
		{fundAutoCollectionRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/auto-collection"},
		{fundCollectionByAssetRate, sapiIPPool, 60, "POST /sapi/v1/portfolio/asset-collection"},
		{classicPortfolioMarginAccountInfoRate, sapiUIDPool, 5, "GET /sapi/v1/portfolio/account"},
		{repayClassicPortfolioMarginBankruptcyLoanRate, sapiUIDPool, 3000, "POST /sapi/v1/portfolio/repay"},
		{getClassicPortfolioMarginBankruptcyLoanAmountRate, sapiUIDPool, 500, "GET /sapi/v1/portfolio/pmLoan"},
		{classicPortfolioMarginNegativeBalanceInterestHistoryRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/interest-history"},
		{repayFuturesNegativeBalanceRate, sapiIPPool, 1500, "POST /sapi/v1/portfolio/repay-futures-negative-balance"},
		{portfolioMarginAssetLeverageRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/margin-asset-leverage"},
		{classicPortfolioMarginCollateralRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/collateralRate"},
		{portfolioMarginAssetIndexPriceRate, sapiIPPool, 50, "GET /sapi/v1/portfolio/asset-index-price without asset"},
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
		{usdtMarginedFuturesDefaultRate, derivativesIPPool, 1, "weight-1 /fapi endpoints, and the single-symbol forms of those weighted by symbol"},
		{usdtMarginedFuturesCancelRate, derivativesCancelPool, 1, "DELETE /fapi/v1/order, /fapi/v1/batchOrders, /fapi/v1/allOpenOrders, /fapi/v1/algoOrder and /fapi/v1/algoOpenOrders"},
		{usdtMarginedFuturesCancelAllListingRate, derivativesCancelPool, 40, "GET /fapi/v1/openOrders and /fapi/v1/openAlgoOrders, all symbols, listed to cancel every symbol's orders"},
		{usdtMarginedFuturesAccountInformationRate, derivativesIPPool, 5, "GET /fapi/v3/balance, GET /fapi/v2/account, GET /fapi/v3/account"},
		{usdtMarginedFuturesTradingStatusAllRate, derivativesIPPool, 10, "GET /fapi/v1/apiTradingStatus without symbol"},
		{usdtMarginedFuturesMultiAssetMarginRate, derivativesIPPool, 30, "GET /fapi/v1/multiAssetsMargin"},
		{usdtMarginedFuturesPositionModeRate, derivativesIPPool, 30, "GET /fapi/v1/positionSide/dual"},
		{usdtMarginedFuturesPositionModeLookupRate, derivativesCancelPool, 30, "GET /fapi/v1/positionSide/dual for a Long or Short order"},
		{usdtMarginedFuturesDownloadIDRate, derivativesIPPool, 1000, "GET /fapi/v1/order/asyn, /trade/asyn and /income/asyn"},
		{usdtMarginedFuturesDownloadLinkRate, derivativesIPPool, 10, "GET /fapi/v1/order/asyn/id, /trade/asyn/id and /income/asyn/id"},
		{usdtMarginedFuturesIncomeHistoryRate, derivativesIPPool, 30, "GET /fapi/v1/income"},
		{usdtMarginedFuturesUserCommissionRate, derivativesIPPool, 20, "GET /fapi/v1/commissionRate"},
		{usdtMarginedFuturesAggregateTradesRate, derivativesIPPool, 20, "GET /fapi/v1/aggTrades"},
		{usdtMarginedFuturesFundingRateHistoryRate, usdtMarginedFuturesFundingPool, 1, "GET /fapi/v1/fundingRate: shares 500 per 5 minutes with fundingInfo"},
		{usdtMarginedFuturesFundingInfoRate, usdtMarginedFuturesFundingPool, 1, "GET /fapi/v1/fundingInfo: weight 0, shares 500 per 5 minutes with fundingRate"},
		{usdtMarginedFuturesKline100Rate, derivativesIPPool, 1, "USDⓈ-M klines, limit [1,100)"},
		{usdtMarginedFuturesKline500Rate, derivativesIPPool, 2, "USDⓈ-M klines, limit [100,500)"},
		{usdtMarginedFuturesKline1000Rate, derivativesIPPool, 5, "USDⓈ-M klines, limit [500,1000] and the default"},
		{usdtMarginedFuturesKlineMaxRate, derivativesIPPool, 10, "USDⓈ-M klines, limit above 1000"},
		{usdtMarginedFuturesMarkPriceAllRate, derivativesIPPool, 10, "GET /fapi/v1/premiumIndex without symbol"},
		{usdtMarginedFuturesAssetIndexAllRate, derivativesIPPool, 10, "GET /fapi/v1/assetIndex without symbol"},
		{usdtMarginedFuturesHistoricalTradesRate, derivativesIPPool, 200, "GET /fapi/v1/historicalTrades"},
		{usdtMarginedFuturesOrderbook50Rate, derivativesIPPool, 2, "GET /fapi/v1/depth limit 5-50"},
		{usdtMarginedFuturesOrderbook100Rate, derivativesIPPool, 5, "GET /fapi/v1/depth limit 100"},
		{usdtMarginedFuturesOrderbook500Rate, derivativesIPPool, 10, "GET /fapi/v1/depth limit 500 and the default"},
		{usdtMarginedFuturesOrderbook1000Rate, derivativesIPPool, 20, "GET /fapi/v1/depth limit 1000"},
		{usdtMarginedFuturesIndexConstituentsRate, derivativesIPPool, 2, "GET /fapi/v1/constituents"},
		{usdtMarginedFuturesRecentTradesRate, derivativesIPPool, 5, "GET /fapi/v1/trades"},
		{usdtMarginedFuturesRetailPriceImprovementOrderbookRate, derivativesIPPool, 20, "GET /fapi/v1/rpiDepth"},
		{usdtMarginedFuturesBookTickerRate, derivativesIPPool, 2, "GET /fapi/v1/ticker/bookTicker, one symbol"},
		{usdtMarginedFuturesOrderbookTickerAllRate, derivativesIPPool, 5, "GET /fapi/v1/ticker/bookTicker, all symbols"},
		{usdtMarginedFuturesSymbolPriceAllRate, derivativesIPPool, 2, "GET /fapi/v2/ticker/price, all symbols"},
		{usdtMarginedFuturesTicker24HourAllRate, derivativesIPPool, 40, "GET /fapi/v1/ticker/24hr, all symbols"},
		{usdtMarginedFuturesAccountTradeListRate, derivativesIPPool, 5, "GET /fapi/v1/userTrades"},
		{usdtMarginedFuturesGetAllOrdersRate, derivativesIPPool, 5, "GET /fapi/v1/allOrders"},
		{usdtMarginedFuturesCountdownCancelRate, derivativesHeartbeatPool, 10, "POST /fapi/v1/countdownCancelAll"},
		{usdtMarginedFuturesHeartbeatOverflowRate, derivativesIPPool, 10, "POST /fapi/v1/countdownCancelAll beyond the heartbeat pool"},
		{usdtMarginedFuturesOrdersDefaultRate, derivativesOrderPool, 1, "POST and PUT /fapi/v1/order, POST /fapi/v1/algoOrder: 1 order"},
		{usdtMarginedFuturesBatchOrdersRate, derivativesOrderPool, 5, "POST and PUT /fapi/v1/batchOrders: 5 on the 10 second order window"},
		{usdtMarginedFuturesOpenAlgoOrdersAllRate, derivativesIPPool, 40, "GET /fapi/v1/openAlgoOrders, all symbols"},
		{usdtMarginedFuturesGetAllOpenOrdersRate, derivativesIPPool, 40, "GET /fapi/v1/openOrders, all symbols"},
		{usdtMarginedFuturesAutoDeleveragingQuantileRate, derivativesIPPool, 5, "GET /fapi/v1/adlQuantile"},
		{usdtMarginedFuturesPositionRiskRate, derivativesIPPool, 5, "GET /fapi/v2/positionRisk, GET /fapi/v3/positionRisk"},
		{usdtMarginedFuturesAllAlgoOrdersRate, derivativesIPPool, 5, "GET /fapi/v1/allAlgoOrders"},
		{usdtMarginedFuturesSymbolForceOrdersRate, derivativesIPPool, 20, "GET /fapi/v1/forceOrders, one symbol"},
		{usdtMarginedFuturesAllForceOrdersRate, derivativesIPPool, 50, "GET /fapi/v1/forceOrders, all symbols"},
		{usdtMarginedFuturesAPIReferralRate, derivativesIPPool, 100, "the OMS Toolkit /fapi/v1/apiReferral endpoints"},
		// /futures/data on the USDⓈ-M and COIN-M hosts
		{futuresDataRate, futuresDataPool, 1, "GET /futures/data/* on both hosts: 1000 requests per 5 minutes"},
		// COIN-M futures /dapi
		{coinMarginedFuturesDefaultRate, derivativesIPPool, 1, "weight-1 /dapi endpoints, and the single-symbol forms of those weighted by symbol"},
		{coinMarginedFuturesCancelRate, derivativesCancelPool, 1, "DELETE /dapi/v1/order, /dapi/v1/batchOrders, /dapi/v1/allOpenOrders, /dapi/v1/algoOrder and /dapi/v1/algoOpenOrders"},
		{coinMarginedFuturesCancelAllListingRate, derivativesCancelPool, 40, "GET /dapi/v1/openOrders and /dapi/v1/openAlgoOrders, all symbols, listed to cancel every symbol's orders"},
		{coinMarginedFuturesAccountInformationRate, derivativesIPPool, 5, "GET /dapi/v1/account"},
		{coinMarginedFuturesIncomeHistoryRate, derivativesIPPool, 20, "GET /dapi/v1/income"},
		{coinMarginedFuturesAggregateTradesRate, derivativesIPPool, 20, "GET /dapi/v1/aggTrades"},
		{coinMarginedFuturesKline100Rate, derivativesIPPool, 1, "COIN-M klines, limit [1,100)"},
		{coinMarginedFuturesKline500Rate, derivativesIPPool, 2, "COIN-M klines, limit [100,500)"},
		{coinMarginedFuturesKline1000Rate, derivativesIPPool, 5, "COIN-M klines, limit [500,1000] and the default"},
		{coinMarginedFuturesKlineMaxRate, derivativesIPPool, 10, "COIN-M klines, limit above 1000"},
		{coinMarginedFuturesIndexMarkPriceRate, derivativesIPPool, 10, "GET /dapi/v1/premiumIndex"},
		{coinMarginedFuturesHistoricalTradesRate, derivativesIPPool, 200, "GET /dapi/v1/historicalTrades"},
		{coinMarginedFuturesOrderbook50Rate, derivativesIPPool, 2, "GET /dapi/v1/depth limit 5-50"},
		{coinMarginedFuturesOrderbook100Rate, derivativesIPPool, 5, "GET /dapi/v1/depth limit 100"},
		{coinMarginedFuturesOrderbook500Rate, derivativesIPPool, 10, "GET /dapi/v1/depth limit 500 and the default"},
		{coinMarginedFuturesOrderbook1000Rate, derivativesIPPool, 20, "GET /dapi/v1/depth limit 1000"},
		{coinMarginedFuturesRecentTradesRate, derivativesIPPool, 5, "GET /dapi/v1/trades"},
		{coinMarginedFuturesBookTickerRate, derivativesIPPool, 2, "GET /dapi/v1/ticker/bookTicker, one symbol"},
		{coinMarginedFuturesOrderbookTickerAllRate, derivativesIPPool, 5, "GET /dapi/v1/ticker/bookTicker, all symbols"},
		{coinMarginedFuturesSymbolPriceAllRate, derivativesIPPool, 2, "GET /dapi/v1/ticker/price, all symbols"},
		{coinMarginedFuturesTicker24HourAllRate, derivativesIPPool, 40, "GET /dapi/v1/ticker/24hr, all symbols"},
		{coinMarginedFuturesAccountTradeListRate, derivativesIPPool, 5, "GET /dapi/v1/userTrades"},
		{coinMarginedFuturesAllOrdersRate, derivativesIPPool, 5, "GET /dapi/v1/allOrders"},
		{coinMarginedFuturesCountdownCancelRate, derivativesHeartbeatPool, 10, "POST /dapi/v1/countdownCancelAll"},
		{coinMarginedFuturesHeartbeatOverflowRate, derivativesIPPool, 10, "POST /dapi/v1/countdownCancelAll beyond the heartbeat pool"},
		{coinMarginedFuturesBatchOrdersRate, derivativesOrderPool, 5, "POST /dapi/v1/batchOrders: up to 5 orders"},
		{coinMarginedFuturesOrdersDefaultRate, derivativesOrderPool, 1, "POST /dapi/v1/order and /dapi/v1/algoOrder: 1 order"},
		{coinMarginedFuturesGetAllOpenOrdersRate, derivativesIPPool, 40, "GET /dapi/v1/openOrders, all symbols"},
		{coinMarginedFuturesOpenAlgoOrdersAllRate, derivativesIPPool, 40, "GET /dapi/v1/openAlgoOrders, all symbols"},
		{coinMarginedFuturesAllAlgoOrdersRate, derivativesIPPool, 5, "GET /dapi/v1/allAlgoOrders"},
		{coinMarginedFuturesAutoDeleveragingQuantileRate, derivativesIPPool, 5, "GET /dapi/v1/adlQuantile"},
		{coinMarginedFuturesSymbolForceOrdersRate, derivativesIPPool, 20, "GET /dapi/v1/forceOrders, one symbol"},
		{coinMarginedFuturesAllForceOrdersRate, derivativesIPPool, 50, "GET /dapi/v1/forceOrders, all symbols"},
		// Options /eapi
		{optionsDefaultRate, optionsIPPool, 1, "weight-1 /eapi endpoints; GET /eapi/v1/openInterest is documented at 0"},
		{optionsCancelRate, optionsCancelPool, 1, "DELETE /eapi/v1/order and /eapi/v1/allOpenOrders"},
		{optionsCancelAllListingRate, optionsCancelPool, 40, "GET /eapi/v1/openOrders, all symbols, listed to cancel every symbol's orders"},
		{optionsMarginAccountInfoRate, optionsIPPool, 3, "GET /eapi/v1/marginAccount"},
		{optionsDownloadIDForOptionTransactionHistoryRate, optionsIPPool, 5, "GET /eapi/v1/income/asyn, no longer documented"},
		{optionsGetTransHistoryDownloadLinkByIDRate, optionsIPPool, 5, "GET /eapi/v1/income/asyn/id, no longer documented"},
		{optionsHistoricalExerciseRecordsRate, optionsIPPool, 3, "GET /eapi/v1/exerciseHistory"},
		{optionsMarkPriceRate, optionsIPPool, 5, "GET /eapi/v1/mark"},
		{optionsOrderbook50Rate, optionsIPPool, 2, "GET /eapi/v1/depth limit 5-50"},
		{optionsOrderbook100Rate, optionsIPPool, 5, "GET /eapi/v1/depth limit 100 and the default"},
		{optionsOrderbook500Rate, optionsIPPool, 10, "GET /eapi/v1/depth limit 500"},
		{optionsOrderbook1000Rate, optionsIPPool, 20, "GET /eapi/v1/depth limit 1000"},
		{optionsRecentTradesRate, optionsIPPool, 5, "GET /eapi/v1/trades"},
		{optionsAllTickerPriceStatisticsRate, optionsIPPool, 40, "GET /eapi/v1/ticker, all symbols"},
		{optionsAutoCancelAllOpenOrdersHeartbeatRate, optionsHeartbeatPool, 10, "POST /eapi/v1/countdownCancelAllHeartBeat"},
		{optionsHeartbeatOverflowRate, optionsIPPool, 10, "POST /eapi/v1/countdownCancelAllHeartBeat beyond the heartbeat pool"},
		{optionsAutoCancelAllOpenOrdersConfigRate, optionsHeartbeatPool, 1, "POST /eapi/v1/countdownCancelAll"},
		{optionsAccountTradeListRate, optionsIPPool, 5, "GET /eapi/v1/userTrades"},
		{optionsCancelAllByUnderlyingRate, optionsCancelPool, 5, "DELETE /eapi/v1/allOpenOrdersByUnderlying"},
		{optionsBatchOrderRate, optionsOrderPool, 5, "POST /eapi/v1/batchOrders: weight 5"},
		{optionsCancelBatchOrdersRate, optionsCancelPool, 5, "DELETE /eapi/v1/batchOrders"},
		{optionsDefaultOrderRate, optionsOrderPool, 1, "POST /eapi/v1/order: 1 order"},
		{optionsPositionInformationRate, optionsIPPool, 5, "GET /eapi/v1/position"},
		{optionsAllQueryOpenOrdersRate, optionsIPPool, 40, "GET /eapi/v1/openOrders, all symbols"},
		{optionsGetOrderHistoryRate, optionsIPPool, 3, "GET /eapi/v1/historyOrders"},
		{optionsUserExerciseRecordRate, optionsIPPool, 5, "GET /eapi/v1/exerciseRecord"},
		// Portfolio margin /papi
		{portfolioMarginDefaultRate, portfolioMarginIPPool, 1, "weight-1 /papi endpoints, and the single-symbol forms of those weighted by symbol"},
		{portfolioMarginGetAccountBalancesRate, portfolioMarginIPPool, 20, "GET /papi/v1/balance"},
		{portfolioMarginGetAccountInformationRate, portfolioMarginIPPool, 20, "GET /papi/v1/account"},
		{portfolioMarginBNBTransferRate, portfolioMarginBNBTransferPool, 1, "POST /papi/v1/bnb-transfer: IP 750, 10 calls per 10 minutes"},
		{portfolioMarginGetAutoRepayFuturesStatusRate, portfolioMarginIPPool, 30, "GET /papi/v1/repay-futures-switch"},
		{portfolioMarginChangeAutoRepayFuturesStatusRate, portfolioMarginIPPool, 750, "POST /papi/v1/repay-futures-switch"},
		{portfolioMarginGetCoinMarginedCurrentPositionModeRate, portfolioMarginIPPool, 30, "GET /papi/v1/cm/positionSide/dual"},
		{portfolioMarginGetUSDTMarginedCurrentPositionModeRate, portfolioMarginIPPool, 30, "GET /papi/v1/um/positionSide/dual"},
		{portfolioMarginFundAutoCollectionRate, portfolioMarginIPPool, 750, "POST /papi/v1/auto-collection"},
		{portfolioMarginFundCollectionByAssetRate, portfolioMarginIPPool, 30, "POST /papi/v1/asset-collection"},
		{portfolioMarginGetCoinMarginedAccountDetailRate, portfolioMarginIPPool, 5, "GET /papi/v1/cm/account"},
		{portfolioMarginGetCoinMarginedIncomeHistoryRate, portfolioMarginIPPool, 30, "GET /papi/v1/cm/income"},
		{portfolioMarginGetUSDTMarginedAccountDetailRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/account"},
		{portfolioMarginGetUSDTMarginedIncomeHistoryRate, portfolioMarginIPPool, 30, "GET /papi/v1/um/income"},
		{portfolioMarginGetCoinMarginedUserCommissionRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/commissionRate"},
		{portfolioMarginGetUSDTMarginedUserCommissionRate, portfolioMarginIPPool, 20, "GET /papi/v1/um/commissionRate"},
		{portfolioMarginMarginMaxBorrowRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/maxBorrowable"},
		{portfolioMarginUSDTMarginedTradingQuantitativeRulesIndicatorsRate, portfolioMarginIPPool, 10, "GET /papi/v1/um/apiTradingStatus without symbol"},
		{portfolioMarginGetMarginLoanRecordRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/marginLoan"},
		{portfolioMarginGetMarginMaxWithdrawalRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/maxWithdraw"},
		{portfolioMarginGetMarginRepayRecordRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/repayLoan"},
		{portfolioMarginGetNegativeBalanceInterestHistoryRate, portfolioMarginIPPool, 50, "GET /papi/v1/portfolio/interest-history"},
		{portfolioMarginGetUSDTMarginedPositionInformationRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/positionRisk"},
		{portfolioMarginNegativeBalanceExchangeRecordRate, portfolioMarginIPPool, 100, "GET /papi/v1/portfolio/negative-balance-exchange-record"},
		{portfolioMarginRepayFuturesNegativeBalanceRate, portfolioMarginIPPool, 750, "POST /papi/v1/repay-futures-negative-balance"},
		{portfolioMarginOrderRate, portfolioMarginOrderPool, 1, "POST /papi/v1 um, cm and margin orders, cm conditional and um algo orders: 1 order"},
		{portfolioMarginCancelMarginAccountOpenOrdersOnSymbolRate, portfolioMarginIPPool, 5, "DELETE /papi/v1/margin/allOpenOrders"},
		{portfolioMarginGetMarginAccountOneCancelsTheOtherRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/orderList"},
		{portfolioMarginCancelMarginAccountOneCancelsTheOtherRate, portfolioMarginIPPool, 2, "DELETE /papi/v1/margin/orderList"},
		{portfolioMarginGetMarginAccountOrderRate, portfolioMarginIPPool, 10, "GET /papi/v1/margin/order"},
		{portfolioMarginCancelMarginAccountOrderRate, portfolioMarginIPPool, 2, "DELETE /papi/v1/margin/order"},
		{portfolioMarginGetCoinMarginedAccountTradeListWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/userTrades with symbol"},
		{portfolioMarginGetCoinMarginedAccountTradeListWithPairRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/userTrades with pair"},
		{portfolioMarginGetCoinMarginedPositionAutoDeleveragingQuantileEstimationRate, portfolioMarginIPPool, 5, "GET /papi/v1/cm/adlQuantile"},
		{portfolioMarginMarginAccountLoanAndRepayRate, portfolioMarginIPPool, 100, "POST /papi/v1/marginLoan, POST /papi/v1/repayLoan"},
		{portfolioMarginOneCancelsTheOtherOrderRate, portfolioMarginOrderPool, 2, "POST /papi/v1/margin/order/oco: 2 orders"},
		{portfolioMarginGetMarginAccountTradeListRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/myTrades"},
		{portfolioMarginAllCoinMarginedConditionalOrderWithoutSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/conditional/allOrders without symbol"},
		{portfolioMarginAllCoinMarginedOrderWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/allOrders with symbol"},
		{portfolioMarginAllCoinMarginedOrderWithPairRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/allOrders with pair"},
		{portfolioMarginAllCoinMarginedOpenConditionalOrdersWithoutSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/conditional/openOrders without symbol"},
		{portfolioMarginRetrieveAllCoinMarginedOpenOrdersForAllSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/cm/openOrders without symbol"},
		{portfolioMarginRetrieveAllUSDTMarginedOpenOrdersForAllSymbolRate, portfolioMarginIPPool, 40, "GET /papi/v1/um/openOrders without symbol"},
		{portfolioMarginAllMarginAccountOrdersRate, portfolioMarginIPPool, 100, "GET /papi/v1/margin/allOrders"},
		{portfolioMarginGetAllUSDTMarginedOrdersRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/allOrders"},
		{portfolioMarginGetUSDTMarginedAlgoOrderHistoryRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/algo/allAlgoOrders"},
		{portfolioMarginCurrentMarginOpenOrderRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/openOrders"},
		{portfolioMarginGetMarginAccountsAllOneCancelsTheOtherOrdersRate, portfolioMarginIPPool, 100, "GET /papi/v1/margin/allOrderList"},
		{portfolioMarginGetMarginAccountsOpenOneCancelsTheOtherOrdersRate, portfolioMarginIPPool, 5, "GET /papi/v1/margin/openOrderList"},
		{portfolioMarginGetUserCoinMarginedForceOrdersWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/cm/forceOrders with symbol"},
		{portfolioMarginGetUserCoinMarginedForceOrdersWithoutSymbolRate, portfolioMarginIPPool, 50, "GET /papi/v1/cm/forceOrders without symbol"},
		{portfolioMarginGetUserUSDTMarginedForceOrdersWithSymbolRate, portfolioMarginIPPool, 20, "GET /papi/v1/um/forceOrders with symbol"},
		{portfolioMarginGetUserUSDTMarginedForceOrdersWithoutSymbolRate, portfolioMarginIPPool, 50, "GET /papi/v1/um/forceOrders without symbol"},
		{portfolioMarginGetUSDTMarginedAccountTradeListRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/userTrades"},
		{portfolioMarginGetUSDTMarginedPositionAutoDeleveragingQuantileEstimationRate, portfolioMarginIPPool, 5, "GET /papi/v1/um/adlQuantile"},
		{portfolioMarginAPIReferralRate, portfolioMarginIPPool, 100, "the OMS Toolkit /papi/v1/apiReferral endpoints"},
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
		burst := poolBursts[cost.pool]
		refill := budget.limit
		if burst > 1 {
			refill -= burst
		}
		assert.InDeltaf(t, float64(refill)/budget.interval.Seconds(), float64(limiter.Limit()), 1e-9, "endpoint limit %d should have its pool's budget", epl)
		assert.Equalf(t, max(burst, 1), limiter.Burst(), "endpoint limit %d should have its pool's burst", epl)
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

func TestCancelsDoNotQueueBehindMarketData(t *testing.T) {
	t.Parallel()
	// A limiter queues requests in turn, so order book snapshots reserving a host's IP pool seconds ahead would hold up a
	// cancellation drawing on the same pool
	limiters := endpointLimiters(endpointCosts)
	now := time.Now()
	for _, tc := range []struct {
		snapshot, marketData request.EndpointLimit
		snapshots            int
		cancels              []request.EndpointLimit
		burst                int
	}{
		{spotOrderbookDepth1000Rate, spotExchangeInfoRate, 10, []request.EndpointLimit{spotCancelRate, spotCancelAllListingRate}, 100},
		{usdtMarginedFuturesOrderbook1000Rate, coinMarginedFuturesOrderbook1000Rate, 10, []request.EndpointLimit{usdtMarginedFuturesCancelRate, coinMarginedFuturesCancelRate, usdtMarginedFuturesCancelAllListingRate, coinMarginedFuturesCancelAllListingRate, usdtMarginedFuturesPositionModeLookupRate}, 200},
		{optionsOrderbook1000Rate, optionsDefaultRate, 10, []request.EndpointLimit{optionsCancelRate, optionsCancelBatchOrdersRate, optionsCancelAllByUnderlyingRate, optionsCancelAllListingRate}, 50},
	} {
		for range tc.snapshots * int(endpointCosts[tc.snapshot].weight) {
			limiters[tc.snapshot].ReserveN(now, 1)
		}
		assert.Positivef(t, limiters[tc.marketData].ReserveN(now, 1).DelayFrom(now), "market data limit %d should queue behind the snapshots", tc.marketData)
		for _, epl := range tc.cancels[1:] {
			assert.Samef(t, limiters[tc.cancels[0]], limiters[epl], "cancellation limit %d should share its host's cancel pool", epl)
		}
		// The burst lets a cancellation of every order go out at once, after which cancellations are spread out
		for range tc.burst {
			assert.Zerof(t, limiters[tc.cancels[0]].ReserveN(now, 1).DelayFrom(now), "cancellation limit %d should not queue behind market data", tc.cancels[0])
		}
		assert.Positivef(t, limiters[tc.cancels[0]].ReserveN(now, 1).DelayFrom(now), "cancellation limit %d should be spread out beyond its burst", tc.cancels[0])
	}
}

func TestHeartbeatsDoNotQueueBehindMarketData(t *testing.T) {
	t.Parallel()
	// A countdown heartbeat reaching Binance after its countdown has run out has every order it covers cancelled, so
	// heartbeats and countdown settings draw on a share of their own instead of queueing behind order book snapshots,
	// and leave the cancel pools to cancellations
	limiters := endpointLimiters(endpointCosts)
	now := time.Now()
	for _, tc := range []struct {
		snapshot   request.EndpointLimit
		heartbeats []request.EndpointLimit
		cancel     request.EndpointLimit
		burst      int
	}{
		{usdtMarginedFuturesOrderbook1000Rate, []request.EndpointLimit{usdtMarginedFuturesCountdownCancelRate, coinMarginedFuturesCountdownCancelRate}, usdtMarginedFuturesCancelRate, 50},
		{optionsOrderbook1000Rate, []request.EndpointLimit{optionsAutoCancelAllOpenOrdersHeartbeatRate, optionsAutoCancelAllOpenOrdersConfigRate}, optionsCancelRate, 20},
	} {
		for range 10 * int(endpointCosts[tc.snapshot].weight) {
			limiters[tc.snapshot].ReserveN(now, 1)
		}
		for _, epl := range tc.heartbeats[1:] {
			assert.Samef(t, limiters[tc.heartbeats[0]], limiters[epl], "heartbeat limit %d should share its host's heartbeat pool", epl)
		}
		assert.NotSamef(t, limiters[tc.cancel], limiters[tc.heartbeats[0]], "heartbeat limit %d should not draw on the cancel pool", tc.heartbeats[0])
		// A round of heartbeats goes out at once, after which heartbeats are spread out while cancellations still go out
		for range tc.burst {
			assert.Zerof(t, limiters[tc.heartbeats[0]].ReserveN(now, 1).DelayFrom(now), "heartbeat limit %d should not queue behind market data", tc.heartbeats[0])
		}
		assert.Positivef(t, limiters[tc.heartbeats[0]].ReserveN(now, 1).DelayFrom(now), "heartbeat limit %d should be spread out beyond its burst", tc.heartbeats[0])
		assert.Zerof(t, limiters[tc.cancel].ReserveN(now, 1).DelayFrom(now), "cancellation limit %d should not queue behind heartbeats", tc.cancel)
	}
}

// requestDelay reserves an endpoint's weight at now, one unit at a time as the request package does, and returns how
// long the request waits
func requestDelay(limiters map[request.EndpointLimit]*rate.Limiter, epl request.EndpointLimit, now time.Time) time.Duration {
	var delay time.Duration
	for range endpointCosts[epl].weight {
		delay = limiters[epl].ReserveN(now, 1).DelayFrom(now)
	}
	return delay
}

func TestWithdrawalHistoryLimit(t *testing.T) {
	t.Parallel()
	// Binance counts withdrawal history's weight against a UID budget of its own for each second, ten requests' worth,
	// so a lone request goes out at once
	limiters := endpointLimiters(endpointCosts)
	now := time.Now()
	for i := range 10 {
		assert.InDeltaf(t, (time.Duration(i) * 100 * time.Millisecond).Seconds(), requestDelay(limiters, withdrawalHistoryRate, now).Seconds(), 1e-6, "withdrawal history request %d should go out at ten a second", i)
	}
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
		{"USDⓈ-M klines by default", usdtMarginedFuturesKlineLimit(0), derivativesIPPool, 5},
		{"USDⓈ-M klines 1", usdtMarginedFuturesKlineLimit(1), derivativesIPPool, 1},
		{"USDⓈ-M klines 99", usdtMarginedFuturesKlineLimit(99), derivativesIPPool, 1},
		{"USDⓈ-M klines 100", usdtMarginedFuturesKlineLimit(100), derivativesIPPool, 2},
		{"USDⓈ-M klines 499", usdtMarginedFuturesKlineLimit(499), derivativesIPPool, 2},
		{"USDⓈ-M klines 500", usdtMarginedFuturesKlineLimit(500), derivativesIPPool, 5},
		{"USDⓈ-M klines 1000", usdtMarginedFuturesKlineLimit(1000), derivativesIPPool, 5},
		{"USDⓈ-M klines 1001", usdtMarginedFuturesKlineLimit(1001), derivativesIPPool, 10},
		{"USDⓈ-M klines 1500", usdtMarginedFuturesKlineLimit(1500), derivativesIPPool, 10},
		{"USDⓈ-M depth by default", usdtMarginedFuturesOrderbookLimit(0), derivativesIPPool, 10},
		{"USDⓈ-M depth 5", usdtMarginedFuturesOrderbookLimit(5), derivativesIPPool, 2},
		{"USDⓈ-M depth 50", usdtMarginedFuturesOrderbookLimit(50), derivativesIPPool, 2},
		{"USDⓈ-M depth 100", usdtMarginedFuturesOrderbookLimit(100), derivativesIPPool, 5},
		{"USDⓈ-M depth 500", usdtMarginedFuturesOrderbookLimit(500), derivativesIPPool, 10},
		{"USDⓈ-M depth 1000", usdtMarginedFuturesOrderbookLimit(1000), derivativesIPPool, 20},
		{"COIN-M klines by default", coinMarginedFuturesKlineLimit(0), derivativesIPPool, 5},
		{"COIN-M klines 1", coinMarginedFuturesKlineLimit(1), derivativesIPPool, 1},
		{"COIN-M klines 99", coinMarginedFuturesKlineLimit(99), derivativesIPPool, 1},
		{"COIN-M klines 100", coinMarginedFuturesKlineLimit(100), derivativesIPPool, 2},
		{"COIN-M klines 499", coinMarginedFuturesKlineLimit(499), derivativesIPPool, 2},
		{"COIN-M klines 500", coinMarginedFuturesKlineLimit(500), derivativesIPPool, 5},
		{"COIN-M klines 1000", coinMarginedFuturesKlineLimit(1000), derivativesIPPool, 5},
		{"COIN-M klines 1001", coinMarginedFuturesKlineLimit(1001), derivativesIPPool, 10},
		{"COIN-M klines 1500", coinMarginedFuturesKlineLimit(1500), derivativesIPPool, 10},
		{"COIN-M depth by default", coinMarginedFuturesOrderbookLimit(0), derivativesIPPool, 10},
		{"COIN-M depth 5", coinMarginedFuturesOrderbookLimit(5), derivativesIPPool, 2},
		{"COIN-M depth 50", coinMarginedFuturesOrderbookLimit(50), derivativesIPPool, 2},
		{"COIN-M depth 100", coinMarginedFuturesOrderbookLimit(100), derivativesIPPool, 5},
		{"COIN-M depth 500", coinMarginedFuturesOrderbookLimit(500), derivativesIPPool, 10},
		{"COIN-M depth 1000", coinMarginedFuturesOrderbookLimit(1000), derivativesIPPool, 20},
		{"options depth by default", optionsOrderbookLimit(0), optionsIPPool, 5},
		{"options depth 5", optionsOrderbookLimit(5), optionsIPPool, 2},
		{"options depth 50", optionsOrderbookLimit(50), optionsIPPool, 2},
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
		"SendHTTPRequest":               3,
		"SendAPIKeyHTTPRequest":         4,
		"SendAuthHTTPRequest":           5,
		"SendWebsocketAPIRequest":       1,
		"SendSignedWebsocketAPIRequest": 1,
	}
	// A sender renamed without this list would otherwise go unchecked
	called := make(map[string]bool, len(senders))
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
					called[sel.Sel.Name] = true
					if arg, ok := n.Args[i].(*ast.Ident); ok && arg.Name == "wsConnectionMessageRate" {
						assert.Failf(t, "unweighted endpoint limit", "%s should pass a weighted endpoint limit to %s", fset.Position(n.Pos()), sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
	for sender := range senders {
		assert.Truef(t, called[sender], "%s should be called by the package", sender)
	}
}

func TestCancelEndpointLimits(t *testing.T) {
	t.Parallel()
	// Every cancellation draws on its host's cancel pool, so none queues behind market data
	want := map[string]string{
		"/api/v3/order":                      "spotCancelRate",
		"/api/v3/openOrders":                 "spotCancelRate",
		"/api/v3/orderList":                  "spotCancelRate",
		"order.cancel":                       "spotCancelRate",
		"openOrders.cancelAll":               "spotCancelRate",
		"orderList.cancel":                   "spotCancelRate",
		"/fapi/v1/order":                     "usdtMarginedFuturesCancelRate",
		"/fapi/v1/batchOrders":               "usdtMarginedFuturesCancelRate",
		"/fapi/v1/allOpenOrders":             "usdtMarginedFuturesCancelRate",
		"/fapi/v1/algoOrder":                 "usdtMarginedFuturesCancelRate",
		"/fapi/v1/algoOpenOrders":            "usdtMarginedFuturesCancelRate",
		"/dapi/v1/order":                     "coinMarginedFuturesCancelRate",
		"/dapi/v1/batchOrders":               "coinMarginedFuturesCancelRate",
		"/dapi/v1/allOpenOrders":             "coinMarginedFuturesCancelRate",
		"/dapi/v1/algoOrder":                 "coinMarginedFuturesCancelRate",
		"/dapi/v1/algoOpenOrders":            "coinMarginedFuturesCancelRate",
		"/eapi/v1/order":                     "optionsCancelRate",
		"/eapi/v1/batchOrders":               "optionsCancelBatchOrdersRate",
		"/eapi/v1/allOpenOrders":             "optionsCancelRate",
		"/eapi/v1/allOpenOrdersByUnderlying": "optionsCancelAllByUnderlyingRate",
	}
	found := make(map[string]bool, len(want))
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	fset := token.NewFileSet()
	literal := func(e ast.Expr) string {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return strings.Trim(lit.Value, `"`)
		}
		return ""
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var endpoint string
			var limit ast.Expr
			switch {
			case sel.Sel.Name == "SendAuthHTTPRequest" && len(call.Args) > 5:
				if method, isSelector := call.Args[2].(*ast.SelectorExpr); !isSelector || method.Sel.Name != "MethodDelete" {
					return true
				}
				endpoint, limit = literal(call.Args[3]), call.Args[5]
			case sel.Sel.Name == "SendSignedWebsocketAPIRequest" && len(call.Args) > 2:
				endpoint, limit = literal(call.Args[2]), call.Args[1]
			default:
				return true
			}
			exp, ok := want[endpoint]
			if !ok {
				return true
			}
			found[endpoint] = true
			got, _ := limit.(*ast.Ident)
			if assert.NotNilf(t, got, "%s should pass a named endpoint limit for %s", fset.Position(call.Pos()), endpoint) {
				assert.Equalf(t, exp, got.Name, "%s should cancel %s on its host's cancel pool", fset.Position(call.Pos()), endpoint)
			}
			return true
		})
	}
	for endpoint := range want {
		assert.Truef(t, found[endpoint], "a cancellation should be sent to %s", endpoint)
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
