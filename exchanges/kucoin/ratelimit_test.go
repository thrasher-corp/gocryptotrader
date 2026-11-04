package kucoin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimitPools(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		pool     *rateLimitPool
		interval time.Duration
		quota    int
	}{
		{"Spot", spotPool, 30 * time.Second, 4000},
		{"Futures", futuresPool, 30 * time.Second, 2000},
		{"Management", managementPool, 30 * time.Second, 2000},
		{"Public", publicPool, 30 * time.Second, 2000},
		{"Earn", earnPool, 30 * time.Second, 2000},
		{"CopyTrading", copyTradingPool, 30 * time.Second, 2000},
		{"Broker", brokerPool, 30 * time.Second, 2000},
		{"Unified Account", unifiedAccountPool, time.Second, 300},
	} {
		assert.Equalf(t, rateLimitPool{tc.interval, tc.quota}, *tc.pool, "the %s pool should have its VIP 0 quota", tc.name)
	}
}

func TestEndpointWeights(t *testing.T) {
	t.Parallel()
	rows := []struct {
		epl      request.EndpointLimit
		pool     *rateLimitPool
		weight   request.Weight
		endpoint string
	}{
		// UTA REST V2 > Market Data
		{getUTAAnnouncementsEPL, publicPool, 20, "GET api.kucoin.com/api/ua/v2/market/announcement"},
		{getUTACurrencyEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/market/currency"},
		{getUTACurrenciesEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/asset/currencies"},
		{getUTATickerEPL, publicPool, 15, "GET api.kucoin.com/api/ua/v2/market/ticker"},
		{getUTASymbolEPL, publicPool, 4, "GET api.kucoin.com/api/ua/v2/market/instrument"},
		{getUTAKlinesEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/market/kline"},
		{getUTAOrderbookEPL, managementPool, 3, "GET api.kucoin.com/api/ua/v2/market/orderbook"},
		{getUTAIndexPriceEPL, publicPool, 2, "GET api.kucoin.com/api/ua/v2/market/index-price"},
		{getUTATradesEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/market/trade"},
		{getUTAPositionTiersEPL, publicPool, 15, "GET api.kucoin.com/api/ua/v2/market/position-tiers"},
		{getUTACollateralRatioEPL, publicPool, 10, "GET api.kucoin.com/api/ua/v2/market/collateral-discount-ratio"},
		{getUTABorrowableCurrenciesEPL, publicPool, 15, "GET api.kucoin.com/api/ua/v2/market/borrowable-currency"},
		{getUTACurrentFundingRateEPL, publicPool, 2, "GET api.kucoin.com/api/ua/v2/market/funding-rate"},
		{getUTAFundingRateHistoryEPL, publicPool, 5, "GET api.kucoin.com/api/ua/v2/market/funding-rate-history"},
		{getUTAFuturesOpenInterestEPL, publicPool, 10, "GET api.kucoin.com/api/ua/v2/market/open-interest"},
		{getUTAInterestRateIndexEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/market/interest-rate-index"},
		{getUTAPlatform24HourMarketStatisticsEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/trade-statistics"},
		{getUTACallAuctionInfoEPL, publicPool, 2, "GET api.kucoin.com/api/ua/v2/market/call-auction-info"},
		{getUTAFiatPriceEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/market/fiat-price"},
		{getUTAThirdPartyCustodyCurrenciesEPL, publicPool, 15, "GET api.kucoin.com/api/ua/v2/oes/currency"},
		{getUTAServiceStatusEPL, publicPool, 3, "GET api.kucoin.com/api/ua/v2/server/status"},
		{getUTAKYCRegionsEPL, publicPool, 20, "GET api.kucoin.com/api/ua/v2/user/kyc-region"},
		{getUTAClientIPAddressEPL, publicPool, 10, "GET api.kucoin.com/api/ua/v2/user/my-ip"},

		// UTA REST V2 > Account Info > Account & Funding
		{getUTAAccountOverviewEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/unified/account/overview"},
		{getUTAAccountCurrencyAssetsEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/unified/account/balance"},
		{getUTAClassicAccountCurrencyAssetsEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/account/balance"},
		{getUTAAPIKeyInfoEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/user/api-key"},
		{getUTAAccountModeEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/account/mode"},
		{setUTAAccountModeEPL, managementPool, 30, "POST api.kucoin.com/api/ua/v2/account/mode"},
		{getUTALeverageEPL, managementPool, 10, "GET api.kucoin.com/api/ua/v2/unified/account/leverage"},
		{modifyUTAFuturesLeverageEPL, managementPool, 20, "POST api.kucoin.com/api/ua/v2/unified/account/modify-leverage"},
		{modifyUTACrossMarginLeverageEPL, managementPool, 20, "POST api.kucoin.com/api/ua/v2/unified/account/modify-leverage-margin-cross"},
		{getUTABorrowingRatesAndLimitsEPL, managementPool, 10, "GET api.kucoin.com/api/ua/v2/account/interest-limits"},
		{getUTAAccountLedgerEPL, managementPool, 2, "GET api.kucoin.com/api/ua/v2/account/ledger"},
		{getUTAInterestHistoryEPL, managementPool, 15, "GET api.kucoin.com/api/ua/v2/account/interest-history"},
		{getUTAAPIRateLimitEPL, managementPool, 3, "GET api.kucoin.com/api/ua/v2/rate-limit/query"},
		{getUTAAllAPIRateLimitEPL, managementPool, 20, "GET api.kucoin.com/api/ua/v2/rate-limit/query-all"},
		{getUTAAPIRateLimitCapEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/rate-limit/query-cap"},
		{getUTAFeeRateEPL, managementPool, 3, "GET api.kucoin.com/api/ua/v2/user/fee-rate"},
		{setUTAKCSFeeDeductionEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/account/fee/kcs-deduct"},
		{getUTAThirdPartyCustodyAccountCurrencyLimitsEPL, managementPool, 20, "GET api.kucoin.com/api/ua/v2/oes/custody-quota"},

		// UTA REST V2 > Account Info > Sub Account
		{getUTASubAccountCurrencyAssetsEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/sub-account/balance"},
		{getUTASubAccountListEPL, managementPool, 2, "GET api.kucoin.com/api/ua/v2/user/sub-account-list"},
		{getUTASubAccountAPIListEPL, managementPool, 20, "GET api.kucoin.com/api/ua/v2/user/sub-api-key"},
		{setUTASubAccountTransferPermissionEPL, managementPool, 15, "POST api.kucoin.com/api/ua/v2/sub-account/canTransferOut"},
		{addUTASubAccountEPL, managementPool, 15, "POST api.kucoin.com/api/ua/v2/user/sub/create-sub-account"},
		{addUTASubAccountAPIEPL, managementPool, 20, "POST api.kucoin.com/api/ua/v2/user/create-sub-api-key"},
		{modifyUTASubAccountAPIEPL, managementPool, 30, "POST api.kucoin.com/api/ua/v2/user/modify-sub-api-key"},
		{setUTASubAccountsAPIRateLimitEPL, managementPool, 15, "POST api.kucoin.com/api/ua/v2/rate-limit/set"},
		{deleteUTASubAccountAPIEPL, managementPool, 30, "DELETE api.kucoin.com/api/ua/v2/user/sub-api-key"},

		// UTA REST V2 > Account Info > Deposit
		{getUTADepositAddressEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/asset/deposit/address"},
		{getUTADepositHistoryEPL, managementPool, 2, "GET api.kucoin.com/api/ua/v2/asset/deposit/history"},

		// UTA REST V2 > Account Info > Transfer
		{getUTATransferQuotasEPL, managementPool, 20, "GET api.kucoin.com/api/ua/v2/account/transfer-quota"},
		{utaFlexTransferEPL, managementPool, 4, "POST api.kucoin.com/api/ua/v2/account/transfer"},

		// UTA REST V2 > Account Info > Withdraw
		{getUTAWithdrawalQuotasEPL, managementPool, 1, "GET api.kucoin.com/api/ua/v2/withdrawals/quotas"},
		{getUTAWithdrawalHistoryEPL, managementPool, 2, "GET api.kucoin.com/api/ua/v2/asset/withdrawal/history"},
		{utaWithdrawEPL, managementPool, 5, "POST api.kucoin.com/api/ua/v2/asset/withdrawal"},
		{cancelUTAWithdrawalEPL, unifiedAccountPool, 5, "POST api.kucoin.com/api/ua/v2/asset/withdraw/cancel"},

		// UTA REST V2 > Orders
		{placeUTAOrderEPL, unifiedAccountPool, 1, "POST api.kucoin.com/api/ua/v2/unified/order/place"},
		{amendUTAOrderEPL, unifiedAccountPool, 1, "POST api.kucoin.com/api/ua/v2/unified/order/amend"},
		{cancelUTAOrderEPL, unifiedAccountPool, 1, "POST api.kucoin.com/api/ua/v2/unified/order/cancel"},
		{batchCancelUTAOrdersByIDEPL, unifiedAccountPool, 4, "POST api.kucoin.com/api/ua/v2/unified/order/cancel-batch"},
		{batchCancelUTAOrdersBySymbolEPL, unifiedAccountPool, 20, "POST api.kucoin.com/api/ua/v2/unified/order/cancel-all"},
		{getUTAOrderDetailsEPL, managementPool, 4, "GET api.kucoin.com/api/ua/v2/unified/order/detail"},
		{getUTAOpenOrderListEPL, managementPool, 4, "GET api.kucoin.com/api/ua/v2/unified/order/open-list"},
		{getUTAOrderHistoryEPL, managementPool, 4, "GET api.kucoin.com/api/ua/v2/unified/order/history"},
		{getUTATradeHistoryEPL, managementPool, 4, "GET api.kucoin.com/api/ua/v2/unified/order/execution"},
		{getUTAMaxOrderQuantityEPL, managementPool, 10, "GET api.kucoin.com/api/ua/v2/order/max-order-quantity"},

		// UTA REST V2 > Positions
		{getUTAPositionListEPL, managementPool, 3, "GET api.kucoin.com/api/ua/v2/unified/position/open-list"},
		{getUTAPositionsHistoryEPL, managementPool, 2, "GET api.kucoin.com/api/ua/v2/position/history"},
		{getUTAPrivateFundingFeeHistoryEPL, managementPool, 15, "GET api.kucoin.com/api/ua/v2/position/funding-history"},
		{modifyUTAMarginModeEPL, managementPool, 10, "POST api.kucoin.com/api/ua/v2/unified/position/margin-mode"},
		{modifyUTAPositionMarginEPL, unifiedAccountPool, 5, "POST api.kucoin.com/api/ua/v2/unified/position/modify-margin"},
		{getUTAMarginModeEPL, managementPool, 10, "GET api.kucoin.com/api/ua/v2/unified/position/margin-mode"},

		// UTA REST V2 > Affiliate
		{getUTAAffiliateInviteesEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/affiliate/queryInvitees"},
		{getUTAAffiliateCommissionEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/affiliate/queryMyCommission"},
		{getUTAAffiliateTradeHistoryEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/affiliate/queryTransactionByUid"},
		{getUTAAffiliateTransactionsEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/affiliate/queryTransactionByTime"},
		{getUTAAffiliateKuminingEPL, managementPool, 30, "GET api.kucoin.com/api/ua/v2/affiliate/queryKumining"},

		// UTA REST V2 > VIP Lending
		{getUTAVIPLendingCollateralRatioEPL, managementPool, 10, "GET api.kucoin.com/api/ua/v2/otc-loan/discount-rate"},
		{getUTAVIPLendingLoanInfoEPL, managementPool, 5, "GET api.kucoin.com/api/ua/v2/otc-loan/loan"},
		{getUTAVIPLendingAccountsEPL, managementPool, 20, "GET api.kucoin.com/api/ua/v2/otc-loan/account"},

		// Classic REST > Account Info > Account & Funding
		{getAccountSummaryInformationEPL, managementPool, 20, "GET api.kucoin.com/api/v2/user-info"},
		{getAPIKeyInfoEPL, managementPool, 20, "GET api.kucoin.com/api/v1/user/api-key"},
		{getSpotAccountTypeEPL, spotPool, 30, "GET api.kucoin.com/api/v1/hf/accounts/opened"},
		{getAllAccountsEPL, managementPool, 5, "GET api.kucoin.com/api/v1/accounts"},
		{getAccountDetailEPL, managementPool, 5, "GET api.kucoin.com/api/v1/accounts/{accountId}"},
		{getCrossMarginAccountsDetailEPL, spotPool, 15, "GET api.kucoin.com/api/v3/margin/accounts"},
		{getIsolatedMarginAccountDetailEPL, spotPool, 15, "GET api.kucoin.com/api/v3/isolated/accounts"},
		{getFuturesAccountOverviewEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/account-overview"},
		{getAccountLedgersEPL, managementPool, 2, "GET api.kucoin.com/api/v1/accounts/ledgers"},
		{getAccountLedgersHFTradeEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/accounts/ledgers"},
		{getAccountLedgerHFMarginEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/account/ledgers"},
		{getFuturesAccountLedgersEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v1/transaction-history"},

		// Classic REST > Account Info > Sub Account
		{createSubUserEPL, managementPool, 15, "POST api.kucoin.com/api/v2/sub/user/created"},
		{addSubAccountMarginPermissionEPL, managementPool, 15, "POST api.kucoin.com/api/v3/sub/user/margin/enable"},
		{addSubAccountFuturesPermissionEPL, managementPool, 15, "POST api.kucoin.com/api/v3/sub/user/futures/enable"},
		{getAllSubAccountsInfoV2EPL, managementPool, 20, "GET api.kucoin.com/api/v2/sub/user"},
		{getSubAccountDetailBalanceEPL, managementPool, 15, "GET api.kucoin.com/api/v1/sub-accounts/{subUserId}"},
		{getAllSubAccountsBalanceV2EPL, managementPool, 20, "GET api.kucoin.com/api/v2/sub-accounts"},
		{getAllFuturesSubAccountBalancesEPL, futuresPool, 6, "GET api-futures.kucoin.com/api/v1/account-overview-all"},

		// Classic REST > Account Info > Sub Account API
		{getSubAccountAPIListEPL, managementPool, 20, "GET api.kucoin.com/api/v1/sub/api-key"},
		{createSubAccountAPIKeyEPL, managementPool, 20, "POST api.kucoin.com/api/v1/sub/api-key"},
		{modifySubAccountAPIKeyEPL, managementPool, 30, "POST api.kucoin.com/api/v1/sub/api-key/update"},
		{deleteSubAccountAPIKeyEPL, managementPool, 30, "DELETE api.kucoin.com/api/v1/sub/api-key"},

		// Classic REST > Account Info > Deposit
		{addDepositAddressV3EPL, managementPool, 20, "POST api.kucoin.com/api/v3/deposit-address/create"},
		{getDepositAddressV3EPL, managementPool, 5, "GET api.kucoin.com/api/v3/deposit-addresses"},
		{getDepositHistoryEPL, managementPool, 5, "GET api.kucoin.com/api/v1/deposits"},

		// Classic REST > Account Info > Withdrawals
		{getWithdrawalQuotasEPL, managementPool, 20, "GET api.kucoin.com/api/v1/withdrawals/quotas"},
		{withdrawV3EPL, managementPool, 5, "POST api.kucoin.com/api/v3/withdrawals"},
		{cancelWithdrawalEPL, managementPool, 20, "DELETE api.kucoin.com/api/v1/withdrawals/{withdrawalId}"},
		{getWithdrawalHistoryEPL, managementPool, 20, "GET api.kucoin.com/api/v1/withdrawals"},
		{getWithdrawalHistoryByIDEPL, managementPool, 20, "GET api.kucoin.com/api/v1/withdrawals/{withdrawalId}"},

		// Classic REST > Account Info > Transfer
		{getTransferQuotasEPL, managementPool, 20, "GET api.kucoin.com/api/v1/accounts/transferable"},
		{flexTransferEPL, managementPool, 4, "POST api.kucoin.com/api/v3/accounts/universal-transfer"},

		// Classic REST > Account Info > Trade Fee
		{getBasicFeeEPL, spotPool, 3, "GET api.kucoin.com/api/v1/base-fee"},
		{getTradingPairActualFeesEPL, spotPool, 3, "GET api.kucoin.com/api/v1/trade-fees"},
		{getFuturesTradingPairsActualFeesEPL, futuresPool, 3, "GET api-futures.kucoin.com/api/v1/trade-fees"},

		// Classic REST > Spot Trading > Market Data
		{getAnnouncementsEPL, publicPool, 20, "GET api.kucoin.com/api/v3/announcements"},
		{getCurrencyDetailV3EPL, publicPool, 3, "GET api.kucoin.com/api/v3/currencies/{currency}"},
		{getCurrenciesV3EPL, publicPool, 3, "GET api.kucoin.com/api/v3/currencies"},
		{getSymbolEPL, publicPool, 4, "GET api.kucoin.com/api/v2/symbols/{symbol}"},
		{getSymbolsEPL, publicPool, 4, "GET api.kucoin.com/api/v2/symbols"},
		{getTickerEPL, publicPool, 2, "GET api.kucoin.com/api/v1/market/orderbook/level1"},
		{getTickersEPL, publicPool, 15, "GET api.kucoin.com/api/v1/market/allTickers"},
		{getTradeHistoryEPL, publicPool, 3, "GET api.kucoin.com/api/v1/market/histories"},
		{getKlinesEPL, publicPool, 3, "GET api.kucoin.com/api/v1/market/candles"},
		{getPartOrderbookEPL, publicPool, 2, "GET api.kucoin.com/api/v1/market/orderbook/level2_{size}"},
		{getOrderbookEPL, spotPool, 3, "GET api.kucoin.com/api/v3/market/orderbook/level2"},
		{getCallAuctionPartOrderbookEPL, publicPool, 2, "GET api.kucoin.com/api/v1/market/orderbook/callauction/level2_{size}"},
		{getCallAuctionInfoEPL, publicPool, 2, "GET api.kucoin.com/api/v1/market/callauctionData"},
		{getFiatPriceEPL, publicPool, 3, "GET api.kucoin.com/api/v1/prices"},
		{get24HourStatsEPL, publicPool, 15, "GET api.kucoin.com/api/v1/market/stats"},
		{getMarketListEPL, publicPool, 3, "GET api.kucoin.com/api/v1/markets"},
		{getClientIPAddressEPL, publicPool, 1, "GET api.kucoin.com/api/v1/my-ip"},
		{getCurrentServerTimeEPL, publicPool, 3, "GET api.kucoin.com/api/v1/timestamp"},
		{getServiceStatusEPL, publicPool, 3, "GET api.kucoin.com/api/v1/status"},
		{getKYCRegionsEPL, publicPool, 20, "GET api.kucoin.com/api/kyc/regions/v4"},

		// Classic REST > Spot Trading > Orders
		{placeHFOrderEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders"},
		{placeHFOrderSyncEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders/sync"},
		{placeHFOrderTestEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders/test"},
		{batchPlaceHFOrdersEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders/multi"},
		{batchPlaceHFOrdersSyncEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders/multi/sync"},
		{cancelHFOrderByOrderIDEPL, spotPool, 1, "DELETE api.kucoin.com/api/v1/hf/orders/{orderId}"},
		{cancelHFOrderByOrderIDSyncEPL, spotPool, 1, "DELETE api.kucoin.com/api/v1/hf/orders/sync/{orderId}"},
		{cancelHFOrderByClientOrderIDEPL, spotPool, 1, "DELETE api.kucoin.com/api/v1/hf/orders/client-order/{clientOid}"},
		{cancelHFOrderByClientOrderIDSyncEPL, spotPool, 1, "DELETE api.kucoin.com/api/v1/hf/orders/sync/client-order/{clientOid}"},
		{cancelPartialHFOrderEPL, spotPool, 2, "DELETE api.kucoin.com/api/v1/hf/orders/cancel/{orderId}"},
		{cancelAllHFOrdersBySymbolEPL, spotPool, 2, "DELETE api.kucoin.com/api/v1/hf/orders"},
		{cancelAllHFOrdersEPL, spotPool, 30, "DELETE api.kucoin.com/api/v1/hf/orders/cancelAll"},
		{modifyHFOrderEPL, spotPool, 1, "POST api.kucoin.com/api/v1/hf/orders/alter"},
		{getHFOrderByOrderIDEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/{orderId}"},
		{getHFOrderByClientOrderIDEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/client-order/{clientOid}"},
		{getSymbolsWithOpenHFOrderEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/active/symbols"},
		{getOpenHFOrdersEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/active"},
		{getOpenHFOrdersByPageEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/active/page"},
		{getClosedHFOrdersEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/done"},
		{getHFTradeHistoryEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/fills"},
		{getDCPEPL, spotPool, 2, "GET api.kucoin.com/api/v1/hf/orders/dead-cancel-all/query"},
		{setDCPEPL, spotPool, 2, "POST api.kucoin.com/api/v1/hf/orders/dead-cancel-all"},
		{placeStopOrderEPL, spotPool, 1, "POST api.kucoin.com/api/v1/stop-order"},
		{batchCancelStopOrdersEPL, spotPool, 3, "DELETE api.kucoin.com/api/v1/stop-order/cancel"},
		{getStopOrderListEPL, spotPool, 8, "GET api.kucoin.com/api/v1/stop-order"},
		{getStopOrderByOrderIDEPL, spotPool, 3, "GET api.kucoin.com/api/v1/stop-order/{orderId}"},
		{getStopOrderByClientOrderIDEPL, spotPool, 3, "GET api.kucoin.com/api/v1/stop-order/queryOrderByClientOid"},
		{placeOCOOrderEPL, spotPool, 2, "POST api.kucoin.com/api/v3/oco/order"},
		{cancelOCOOrderByOrderIDEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/oco/order/{orderId}"},
		{cancelOCOOrderByClientOrderIDEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/oco/client-order/{clientOid}"},
		{cancelOCOMultipleOrdersEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/oco/orders"},
		{getOCOOrderListEPL, spotPool, 2, "GET api.kucoin.com/api/v3/oco/orders"},

		// Classic REST > Margin Trading > Market Data
		{getCrossMarginSymbolsEPL, publicPool, 3, "GET api.kucoin.com/api/v3/margin/symbols"},
		{getIsolatedMarginSymbolsEPL, publicPool, 3, "GET api.kucoin.com/api/v1/isolated/symbols"},
		{getMarkPriceEPL, publicPool, 2, "GET api.kucoin.com/api/v1/mark-price/{symbol}/current"},
		{getMarginConfigurationEPL, spotPool, 25, "GET api.kucoin.com/api/v1/margin/config"},
		{getAllMarginTradingPairsMarkPricesEPL, publicPool, 10, "GET api.kucoin.com/api/v3/mark-price/all-symbols"},
		{getMarginCollateralRatioEPL, spotPool, 10, "GET api.kucoin.com/api/v3/margin/collateralRatio"},
		{getMarketAvailableInventoryEPL, publicPool, 10, "GET api.kucoin.com/api/v3/margin/available-inventory"},

		// Classic REST > Margin Trading > Orders
		{placeMarginHFOrderEPL, spotPool, 2, "POST api.kucoin.com/api/v3/hf/margin/order"},
		{placeMarginHFOrderTestEPL, spotPool, 2, "POST api.kucoin.com/api/v3/hf/margin/order/test"},
		{cancelMarginHFOrderByOrderIDEPL, spotPool, 2, "DELETE api.kucoin.com/api/v3/hf/margin/orders/{orderId}"},
		{cancelMarginHFOrderByClientOrderIDEPL, spotPool, 2, "DELETE api.kucoin.com/api/v3/hf/margin/orders/client-order/{clientOid}"},
		{cancelAllMarginHFOrdersBySymbolEPL, spotPool, 5, "DELETE api.kucoin.com/api/v3/hf/margin/orders"},
		{getActiveMarginHFOrderSymbolsEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/order/active/symbols"},
		{getActiveMarginHFOrdersEPL, spotPool, 4, "GET api.kucoin.com/api/v3/hf/margin/orders/active"},
		{getClosedMarginHFOrdersEPL, spotPool, 10, "GET api.kucoin.com/api/v3/hf/margin/orders/done"},
		{getMarginHFTradeFillsEPL, spotPool, 5, "GET api.kucoin.com/api/v3/hf/margin/fills"},
		{getMarginHFOrderDetailByOrderIDEPL, spotPool, 5, "GET api.kucoin.com/api/v3/hf/margin/orders/{orderId}"},
		{getMarginHFOrderDetailByClientOrderIDEPL, spotPool, 5, "GET api.kucoin.com/api/v3/hf/margin/orders/client-order/{clientOid}"},
		{placeMarginStopOrderEPL, spotPool, 3, "POST api.kucoin.com/api/v3/hf/margin/stop-order"},
		{cancelMarginStopOrderByOrderIDEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/hf/margin/stop-order/cancel-by-id"},
		{cancelMarginStopOrderByClientOrderIDEPL, spotPool, 5, "DELETE api.kucoin.com/api/v3/hf/margin/stop-order/cancel-by-clientOid"},
		{cancelMarginStopOrdersEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/hf/margin/stop-order/cancel"},
		{getMarginStopOrdersEPL, spotPool, 8, "GET api.kucoin.com/api/v3/hf/margin/stop-orders"},
		{getMarginStopOrderByOrderIDEPL, spotPool, 3, "GET api.kucoin.com/api/v3/hf/margin/stop-order/orderId"},
		{getMarginStopOrderByClientOrderIDEPL, spotPool, 3, "GET api.kucoin.com/api/v3/hf/margin/stop-order/clientOid"},
		{placeMarginOCOOrderEPL, spotPool, 2, "POST api.kucoin.com/api/v3/hf/margin/oco-order"},
		{cancelMarginOCOOrderByOrderIDEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/hf/margin/oco-order/cancel-by-id"},
		{cancelMarginOCOOrderByClientOrderIDEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/hf/margin/oco-order/cancel-by-clientOid"},
		{cancelMarginOCOOrdersEPL, spotPool, 3, "DELETE api.kucoin.com/api/v3/hf/margin/oco-order/cancel"},
		{getMarginOCOOrderByOrderIDEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/oco-order/orderId"},
		{getMarginOCOOrderByClientOrderIDEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/oco-order/clientOid"},
		{getMarginOCOOrderDetailByOrderIDEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/oco-order/detail/orderId"},
		{getMarginOCOOrdersEPL, spotPool, 2, "GET api.kucoin.com/api/v3/hf/margin/oco-orders"},

		// Classic REST > Margin Trading > Debit
		{getBorrowInterestRateEPL, spotPool, 5, "GET api.kucoin.com/api/v3/margin/borrowRate"},
		{postMarginBorrowOrderEPL, spotPool, 15, "POST api.kucoin.com/api/v3/margin/borrow"},
		{getMarginBorrowingHistoryEPL, spotPool, 15, "GET api.kucoin.com/api/v3/margin/borrow"},
		{postRepaymentEPL, spotPool, 10, "POST api.kucoin.com/api/v3/margin/repay"},
		{getRepaymentHistoryEPL, spotPool, 15, "GET api.kucoin.com/api/v3/margin/repay"},
		{getCrossIsolatedMarginInterestRecordsEPL, spotPool, 20, "GET api.kucoin.com/api/v3/margin/interest"},
		{modifyLeverageMultiplierEPL, spotPool, 8, "POST api.kucoin.com/api/v3/position/update-user-leverage"},

		// Classic REST > Margin Trading > Credit
		{getLendingCurrencyInformationEPL, spotPool, 10, "GET api.kucoin.com/api/v3/project/list"},
		{getLoanMarketInterestRateEPL, publicPool, 5, "GET api.kucoin.com/api/v3/project/marketInterestRate"},
		{marginLendingSubscriptionEPL, spotPool, 15, "POST api.kucoin.com/api/v3/purchase"},
		{modifySubscriptionOrderEPL, spotPool, 10, "POST api.kucoin.com/api/v3/lend/purchase/update"},
		{getSubscriptionOrdersEPL, spotPool, 10, "GET api.kucoin.com/api/v3/purchase/orders"},
		{redemptionEPL, spotPool, 15, "POST api.kucoin.com/api/v3/redeem"},
		{getRedemptionOrdersEPL, spotPool, 10, "GET api.kucoin.com/api/v3/redeem/orders"},

		// Classic REST > Margin Trading > Risk Limit
		{getMarginRiskLimitEPL, spotPool, 20, "GET api.kucoin.com/api/v3/margin/currencies"},

		// Classic REST > Futures Trading > Market Data
		{getFuturesContractEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/contracts/{symbol}"},
		{getFuturesOpenContractsEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/contracts/active"},
		{getFuturesTickerEPL, publicPool, 2, "GET api-futures.kucoin.com/api/v1/ticker"},
		{getLatestTickersForAllContractsEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/allTickers"},
		{getFuturesOrderbookEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/level2/snapshot"},
		{getFuturesPartOrderbookEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/level2/depth{size}"},
		{getFuturesTradeHistoryEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/trade/history"},
		{getFuturesKlineEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/kline/query"},
		{getFuturesCurrentMarkPriceEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/mark-price/{symbol}/current"},
		{getFuturesIndexListEPL, publicPool, 2, "GET api-futures.kucoin.com/api/v1/index/query"},
		{getFuturesInterestRateEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/interest/query"},
		{getFuturesPremiumIndexEPL, publicPool, 3, "GET api-futures.kucoin.com/api/v1/premium/query"},
		{get24HourFuturesTransactionVolumeEPL, futuresPool, 3, "GET api-futures.kucoin.com/api/v1/trade-statistics"},
		{getFuturesServerTimeEPL, publicPool, 2, "GET api-futures.kucoin.com/api/v1/timestamp"},
		{getFuturesServiceStatusEPL, publicPool, 4, "GET api-futures.kucoin.com/api/v1/status"},

		// Classic REST > Futures Trading > Orders
		{postFuturesOrderEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v1/orders"},
		{postFuturesOrderTestEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v1/orders/test"},
		{placeMultipleFuturesOrdersEPL, futuresPool, 20, "POST api-futures.kucoin.com/api/v1/orders/multi"},
		{postFuturesTakeProfitStopLossOrderEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v1/st-orders"},
		{cancelFuturesOrderByOrderIDEPL, futuresPool, 1, "DELETE api-futures.kucoin.com/api/v1/orders/{orderId}"},
		{cancelFuturesOrderByClientOrderIDEPL, futuresPool, 1, "DELETE api-futures.kucoin.com/api/v1/orders/client-order/{clientOid}"},
		{batchCancelFuturesOrdersEPL, futuresPool, 20, "DELETE api-futures.kucoin.com/api/v1/orders/multi-cancel"},
		{cancelAllFuturesOrdersEPL, futuresPool, 10, "DELETE api-futures.kucoin.com/api/v3/orders"},
		{cancelAllFuturesStopOrdersEPL, futuresPool, 15, "DELETE api-futures.kucoin.com/api/v1/stopOrders"},
		{getFuturesOrderDetailsByOrderIDEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/orders/{order-id}"},
		{getFuturesOrderDetailsByClientOrderIDEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/orders/byClientOid"},
		{getFuturesOrdersEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v1/orders"},
		{getFuturesRecentCompletedOrdersEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/recentDoneOrders"},
		{getUntriggeredFuturesStopOrdersEPL, futuresPool, 6, "GET api-futures.kucoin.com/api/v1/stopOrders"},
		{getFuturesOpenOrderStatsEPL, futuresPool, 10, "GET api-futures.kucoin.com/api/v1/openOrderStatistics"},
		{getFuturesRecentFillsEPL, futuresPool, 3, "GET api-futures.kucoin.com/api/v1/recentFills"},
		{getFuturesFillsEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/fills"},

		// Classic REST > Futures Trading > Positions
		{getFuturesMarginModeEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/position/getMarginMode"},
		{switchFuturesMarginModeEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v2/position/changeMarginMode"},
		{batchSwitchFuturesMarginModeEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v2/position/batchChangeMarginMode"},
		{getFuturesPositionModeEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/position/getPositionMode"},
		{switchFuturesPositionModeEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v2/position/switchPositionMode"},
		{getMaximumOpenPositionSizeEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/getMaxOpenSize"},
		{getFuturesPositionDetailsEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/position"},
		{getFuturesPositionListEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v1/positions"},
		{getPositionHistoryEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v1/history-positions"},
		{getMaxWithdrawMarginEPL, futuresPool, 10, "GET api-futures.kucoin.com/api/v1/margin/maxWithdrawMargin"},
		{getFuturesCrossMarginLeverageEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/getCrossUserLeverage"},
		{modifyFuturesCrossMarginLeverageEPL, futuresPool, 2, "POST api-futures.kucoin.com/api/v2/changeCrossUserLeverage"},
		{addMarginEPL, futuresPool, 4, "POST api-futures.kucoin.com/api/v1/position/margin/deposit-margin"},
		{removeMarginManuallyEPL, futuresPool, 10, "POST api-futures.kucoin.com/api/v1/margin/withdrawMargin"},
		{getFuturesCrossMarginRiskLimitEPL, futuresPool, 2, "GET api-futures.kucoin.com/api/v2/batchGetCrossOrderLimit"},
		{getFuturesCrossMarginRequirementEPL, futuresPool, 3, "POST api-futures.kucoin.com/api/v2/getCrossModeMarginRequirement"},
		{getFuturesRiskLimitLevelEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/contracts/risk-limit/{symbol}"},
		{updateFuturesRiskLimitLevelEPL, futuresPool, 5, "POST api-futures.kucoin.com/api/v1/position/risk-limit-level/change"},

		// Classic REST > Futures Trading > Funding Fees
		{getFuturesCurrentFundingRateEPL, publicPool, 2, "GET api-futures.kucoin.com/api/v1/funding-rate/{symbol}/current"},
		{getPublicFundingRateEPL, publicPool, 5, "GET api-futures.kucoin.com/api/v1/contract/funding-rates"},
		{getFuturesFundingHistoryEPL, futuresPool, 5, "GET api-futures.kucoin.com/api/v1/funding-history"},

		// Classic REST > Earn > Simple Earn
		{subscribeToEarnFixedIncomeProductEPL, earnPool, 5, "POST api.kucoin.com/api/v1/earn/orders"},
		{getEarnRedeemPreviewByHoldingIDEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/redeem-preview"},
		{redeemByEarnHoldingIDEPL, earnPool, 5, "DELETE api.kucoin.com/api/v1/earn/orders"},
		{getEarnSavingsProductsEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/saving/products"},
		{getLimitedTimePromotionProductsEPL, earnPool, 1, "GET api.kucoin.com/api/v1/earn/promotion/products"},
		{getEarnStakingProductsEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/staking/products"},
		{getEarnKCSStakingProductsEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/kcs-staking/products"},
		{getEarnETHStakingProductsEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/eth-staking/products"},
		{getEarnFixedIncomeCurrentHoldingsEPL, earnPool, 5, "GET api.kucoin.com/api/v1/earn/hold-assets"},

		// Classic REST > Earn > Structured Earn - Dual
		{purchaseStructuredProductEPL, earnPool, 5, "POST api.kucoin.com/api/v1/struct-earn/orders"},
		{getDualInvestmentProductsEPL, earnPool, 3, "GET api.kucoin.com/api/v1/struct-earn/dual/products"},
		{getStructuredProductOrdersEPL, earnPool, 5, "GET api.kucoin.com/api/v1/struct-earn/orders"},

		// Classic REST > VIP Lending
		{getVIPLendingCollateralRatioEPL, managementPool, 10, "GET api.kucoin.com/api/v1/otc-loan/discount-rate-configs"},
		{getInformationOnOffExchangeFundingAndLoansEPL, managementPool, 5, "GET api.kucoin.com/api/v1/otc-loan/loan"},
		{getInformationOnAccountInvolvedInOffExchangeLoansEPL, managementPool, 20, "GET api.kucoin.com/api/v1/otc-loan/accounts"},

		// Classic REST > Affiliate
		{getAffiliateInviteesEPL, managementPool, 30, "GET api.kucoin.com/api/v2/affiliate/queryInvitees"},
		{getAffiliateCommissionEPL, managementPool, 30, "GET api.kucoin.com/api/v2/affiliate/queryMyCommission"},
		{getAffiliateTradeHistoryEPL, managementPool, 30, "GET api.kucoin.com/api/v2/affiliate/queryTransactionByUid"},
		{getAffiliateTransactionsEPL, managementPool, 30, "GET api.kucoin.com/api/v2/affiliate/queryTransactionByTime"},
		{getAffiliateKuminingEPL, managementPool, 30, "GET api.kucoin.com/api/v2/affiliate/queryKumining"},

		// Classic REST > Broker > Broker Pro
		{applyForFastWithdrawalEPL, managementPool, 5, "POST api.kucoin.com/api/v2/broker/withdrawal"},
		{getBrokerProRebateEPL, managementPool, 3, "GET api.kucoin.com/api/v2/broker/api/rebate/download"},
		{getBrokerProCommissionEPL, managementPool, 3, "GET api.kucoin.com/api/v2/broker/queryMyCommission"},
		{getBrokerProUserListEPL, managementPool, 3, "GET api.kucoin.com/api/v2/broker/queryUser"},
		{getBrokerProUserTransactionsEPL, managementPool, 3, "GET api.kucoin.com/api/v2/broker/queryDetailByUid"},

		// Classic REST > Broker > Exchange Broker
		{getExchangeBrokerRebateEPL, brokerPool, 3, "GET api-broker.kucoin.com/api/v1/broker/nd/rebate/download"},
		{getExchangeBrokerRebateV3EPL, brokerPool, 20, "GET api-broker.kucoin.com/api/v3/broker/nd/rebate/download"},
		{submitExchangeBrokerKYCEPL, brokerPool, 1, "POST api-broker.kucoin.com/api/kyc/ndBroker/proxyClient/submit"},
		{getExchangeBrokerKYCStatusEPL, brokerPool, 1, "GET api-broker.kucoin.com/api/kyc/ndBroker/proxyClient/status/list"},
		{getExchangeBrokerKYCStatusListEPL, brokerPool, 1, "GET api-broker.kucoin.com/api/kyc/ndBroker/proxyClient/status/page"},
		{getExchangeBrokerInfoEPL, brokerPool, 2, "GET api-broker.kucoin.com/api/v1/broker/nd/info"},
		{addExchangeBrokerSubAccountEPL, brokerPool, 3, "POST api-broker.kucoin.com/api/v1/broker/nd/account"},
		{getExchangeBrokerSubAccountsEPL, brokerPool, 2, "GET api-broker.kucoin.com/api/v1/broker/nd/account"},
		{addExchangeBrokerSubAccountAPIEPL, brokerPool, 3, "POST api-broker.kucoin.com/api/v1/broker/nd/account/apikey"},
		{getExchangeBrokerSubAccountAPIsEPL, brokerPool, 2, "GET api-broker.kucoin.com/api/v1/broker/nd/account/apikey"},
		{modifyExchangeBrokerSubAccountAPIEPL, brokerPool, 3, "POST api-broker.kucoin.com/api/v1/broker/nd/account/update-apikey"},
		{deleteExchangeBrokerSubAccountAPIEPL, brokerPool, 3, "DELETE api-broker.kucoin.com/api/v1/broker/nd/account/apikey"},
		{exchangeBrokerTransferEPL, brokerPool, 1, "POST api-broker.kucoin.com/api/v1/broker/nd/transfer"},
		{getExchangeBrokerTransferHistoryEPL, brokerPool, 1, "GET api-broker.kucoin.com/api/v3/broker/nd/transfer/detail"},
		{getExchangeBrokerDepositListEPL, brokerPool, 10, "GET api-broker.kucoin.com/api/v1/asset/ndbroker/deposit/list"},
		{getExchangeBrokerDepositDetailEPL, brokerPool, 1, "GET api-broker.kucoin.com/api/v3/broker/nd/deposit/detail"},
		{getExchangeBrokerWithdrawDetailEPL, brokerPool, 1, "GET api-broker.kucoin.com/api/v3/broker/nd/withdraw/detail"},
		{setExchangeBrokerMarkupFeeEPL, brokerPool, 3, "POST api-broker.kucoin.com/api/v1/broker/nd/mark-up"},
		{getExchangeBrokerMarkupFeeEPL, brokerPool, 3, "GET api-broker.kucoin.com/api/v1/broker/nd/mark-up"},

		// Classic REST > Copy Trading
		{addCopyTradingOrderEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/orders"},
		{addCopyTradingOrderTestEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/orders/test"},
		{addCopyTradingTPSLOrderEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/st-orders"},
		{cancelCopyTradingOrderByIDEPL, copyTradingPool, 1, "DELETE api-futures.kucoin.com/api/v1/copy-trade/futures/orders"},
		{cancelCopyTradingOrderByClientOrderIDEPL, copyTradingPool, 1, "DELETE api-futures.kucoin.com/api/v1/copy-trade/futures/orders/client-order"},
		{getCopyTradingMaxOpenSizeEPL, copyTradingPool, 4, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/get-max-open-size"},
		{getCopyTradingMaxWithdrawMarginEPL, copyTradingPool, 10, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/position/margin/max-withdraw-margin"},
		{addCopyTradingIsolatedMarginEPL, copyTradingPool, 4, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/margin/deposit-margin"},
		{removeCopyTradingIsolatedMarginEPL, copyTradingPool, 10, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/margin/withdraw-margin"},
		{modifyCopyTradingIsolatedMarginRiskLimitEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/risk-limit-level/change"},
		{modifyCopyTradingAutoDepositStatusEPL, copyTradingPool, 4, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/margin/auto-deposit-status"},
		{switchCopyTradingMarginModeEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/changeMarginMode"},
		{modifyCopyTradingCrossMarginLeverageEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v2/copy-trade/futures/changeCrossUserLeverage"},
		{getCopyTradingCrossMarginRequirementEPL, copyTradingPool, 3, "POST api-futures.kucoin.com/api/v2/copy-trade/getCrossModeMarginRequirement"},
		{switchCopyTradingPositionModeEPL, copyTradingPool, 2, "POST api-futures.kucoin.com/api/v2/copy-trade/position/switchPositionMode"},
		{getCopyTradingActiveOrdersEPL, copyTradingPool, 1, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/orders"},
		{getCopyTradingFillsEPL, copyTradingPool, 1, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/fills"},
		{getCopyTradingPositionsEPL, copyTradingPool, 1, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/positions"},
		{getCopyTradingOrderHistoryEPL, copyTradingPool, 1, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/orders/history"},
		{getCopyTradingAccountOverviewEPL, copyTradingPool, 1, "GET api-futures.kucoin.com/api/v1/copy-trade/futures/account-overview"},
		{batchSwitchCopyTradingMarginModeEPL, copyTradingPool, 1, "POST api-futures.kucoin.com/api/v1/copy-trade/futures/position/changeMarginModeBatch"},

		// Classic REST > Convert
		{getConvertSymbolEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/symbol"},
		{getConvertCurrenciesEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/currencies"},
		{getConvertQuoteEPL, spotPool, 20, "GET api.kucoin.com/api/v1/convert/quote"},
		{addConvertOrderEPL, spotPool, 20, "POST api.kucoin.com/api/v1/convert/order"},
		{getConvertOrderDetailEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/order/detail"},
		{getConvertOrderHistoryEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/order/history"},
		{addConvertLimitOrderEPL, spotPool, 20, "POST api.kucoin.com/api/v1/convert/limit/order"},
		{getConvertLimitQuoteEPL, spotPool, 20, "GET api.kucoin.com/api/v1/convert/limit/quote"},
		{getConvertLimitOrderDetailEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/limit/order/detail"},
		{getConvertLimitOrdersEPL, spotPool, 5, "GET api.kucoin.com/api/v1/convert/limit/orders"},
		{cancelConvertLimitOrderEPL, spotPool, 5, "DELETE api.kucoin.com/api/v1/convert/limit/order/cancel"},

		// Classic WebSocket > Base Info
		{getPublicTokenEPL, publicPool, 10, "POST api.kucoin.com/api/v1/bullet-public"},
		{getPrivateTokenEPL, spotPool, 10, "POST api.kucoin.com/api/v1/bullet-private"},
		{getFuturesPublicTokenEPL, publicPool, 10, "POST api-futures.kucoin.com/api/v1/bullet-public"},
		{getFuturesPrivateTokenEPL, futuresPool, 10, "POST api-futures.kucoin.com/api/v1/bullet-private"},
	}
	for _, tc := range rows {
		w, ok := endpointWeights[tc.epl]
		if assert.Truef(t, ok, "%s should have an endpoint weight", tc.endpoint) {
			assert.Samef(t, tc.pool, w.pool, "%s should draw from its documented pool", tc.endpoint)
			assert.Equalf(t, tc.weight, w.weight, "%s should have its documented weight", tc.endpoint)
		}
	}
	assert.Len(t, endpointWeights, len(rows), "endpointWeights should hold exactly the documented weights")
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
	require.Equal(t, "getUTAAnnouncementsEPL", names[0], "getUTAAnnouncementsEPL must be the first endpoint limit")
	require.Equal(t, "getFuturesPrivateTokenEPL", names[len(names)-1], "getFuturesPrivateTokenEPL must be the last endpoint limit")
	require.Equal(t, int(getFuturesPrivateTokenEPL-getUTAAnnouncementsEPL)+1, len(names), "the endpoint limits must run from getUTAAnnouncementsEPL to getFuturesPrivateTokenEPL")
	assert.Greater(t, getUTAAnnouncementsEPL, request.UnAuth, "endpoint limits should not alias the request package's generic limits")
	definitions := GetRateLimit()
	for i, name := range names {
		epl := getUTAAnnouncementsEPL + request.EndpointLimit(i)
		w, ok := endpointWeights[epl]
		if !assert.Truef(t, ok, "%s should have an endpoint weight", name) {
			continue
		}
		if limiter := definitions[epl]; assert.NotNilf(t, limiter, "%s should be defined", name) {
			assert.Equalf(t, w.weight, limiter.Weight(), "%s should draw its documented weight", name)
		}
	}
	assert.Len(t, definitions, len(names), "GetRateLimit should define every endpoint limit and nothing else")
}

func TestGetRateLimit(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		// The first endpoint limit of each pool drains it, so the next request waits one token longer only when it
		// draws from the same pool
		drainers := make(map[*rateLimitPool]request.EndpointLimit)
		for epl, w := range endpointWeights {
			if drainer, ok := drainers[w.pool]; !ok || epl < drainer {
				drainers[w.pool] = epl
			}
		}
		for pool, drainer := range drainers {
			for epl, w := range endpointWeights {
				definitions := GetRateLimit()
				require.NoErrorf(t, definitions[drainer].RateLimit(t.Context()), "endpoint limit %d must drain its pool", drainer)
				start := time.Now()
				require.NoErrorf(t, definitions[epl].RateLimit(t.Context()), "endpoint limit %d must take its weight", epl)
				tokens := float64(w.weight - 1)
				if w.pool == pool {
					tokens++
				}
				exp := time.Duration(tokens * float64(w.pool.interval) / float64(w.pool.quota))
				assert.InDeltaf(t, exp, time.Since(start), float64(time.Microsecond), "endpoint limit %d should wait for its weight after endpoint limit %d", epl, drainer)
			}
		}
	})
}

func TestGetRateLimitGenericLimits(t *testing.T) {
	t.Parallel()
	r, err := request.New(t.Name(), &http.Client{}, request.WithLimiter(GetRateLimit()))
	require.NoError(t, err, "request.New must not error")
	for _, epl := range []request.EndpointLimit{request.Unset, request.Auth, request.UnAuth} {
		assert.ErrorIsf(t, r.InitiateRateLimit(t.Context(), epl), common.ErrNilPointer, "generic endpoint limit %d should find no KuCoin limiter", epl)
	}
}

// restSenders holds the argument index of the endpoint limit of every function that sends REST requests
var restSenders = map[string]int{"SendHTTPRequest": 2, "SendAuthHTTPRequest": 2, "SendPayload": 1}

func TestEndpointLimitArguments(t *testing.T) {
	t.Parallel()
	limits := make(map[string]bool)
	for _, name := range endpointLimitNames(t) {
		limits[name] = true
	}
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	fset := token.NewFileSet()
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
			i, ok := restSenders[sel.Sel.Name]
			if !ok || i >= len(call.Args) {
				return true
			}
			// A sender passing its caller's limit through is checked at its callers
			if ident, ok := call.Args[i].(*ast.Ident); !ok || !limits[ident.Name] && ident.Name != "epl" {
				assert.Failf(t, "endpoint limit", "%s should pass %s one of the package's endpoint limits", fset.Position(call.Pos()), sel.Sel.Name)
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
		if strings.HasSuffix(file, "_test.go") || file == "ratelimit.go" {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		ast.Inspect(f, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				used[ident.Name] = true
			}
			return true
		})
	}
	for _, name := range endpointLimitNames(t) {
		assert.Truef(t, used[name], "%s should be used by an endpoint", name)
	}
}
