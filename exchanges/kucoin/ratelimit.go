package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// rateLimitPool is one of KuCoin's REST rate limit pools. Each request draws its endpoint's weight from the
// quota of the pool the docs assign the endpoint to
type rateLimitPool struct {
	interval time.Duration
	quota    int
}

// KuCoin's REST rate limit pools at VIP 0. The Classic and UTA docs share the Management and Public pools, and
// UTA trading has a pool of its own
var (
	spotPool           = &rateLimitPool{30 * time.Second, 4000}
	futuresPool        = &rateLimitPool{30 * time.Second, 2000}
	managementPool     = &rateLimitPool{30 * time.Second, 2000}
	publicPool         = &rateLimitPool{30 * time.Second, 2000}
	earnPool           = &rateLimitPool{30 * time.Second, 2000}
	copyTradingPool    = &rateLimitPool{30 * time.Second, 2000}
	brokerPool         = &rateLimitPool{30 * time.Second, 2000}
	unifiedAccountPool = &rateLimitPool{time.Second, 300}
)

// endpointWeight is the rate limit pool an endpoint draws from and the weight of each request
type endpointWeight struct {
	pool   *rateLimitPool
	weight request.Weight
}

// Endpoint limits, one per REST endpoint in the docs' order. They start after request.UnAuth, so a stray
// request.Unset, request.Auth or request.UnAuth finds no KuCoin limiter instead of silently using one
const (
	// UTA REST V2 > Market Data
	getUTAAnnouncementsEPL request.EndpointLimit = request.UnAuth + 1 + iota
	getUTACurrencyEPL
	getUTACurrenciesEPL
	getUTATickerEPL
	getUTASymbolEPL
	getUTAKlinesEPL
	getUTAOrderbookEPL
	getUTAIndexPriceEPL
	getUTATradesEPL
	getUTAPositionTiersEPL
	getUTACollateralRatioEPL
	getUTABorrowableCurrenciesEPL
	getUTACurrentFundingRateEPL
	getUTAFundingRateHistoryEPL
	getUTAFuturesOpenInterestEPL
	getUTAInterestRateIndexEPL
	getUTAPlatform24HourMarketStatisticsEPL
	getUTACallAuctionInfoEPL
	getUTAFiatPriceEPL
	getUTAThirdPartyCustodyCurrenciesEPL
	getUTAServiceStatusEPL
	getUTAKYCRegionsEPL
	getUTAClientIPAddressEPL

	// UTA REST V2 > Account Info > Account & Funding
	getUTAAccountOverviewEPL
	getUTAAccountCurrencyAssetsEPL
	getUTAClassicAccountCurrencyAssetsEPL
	getUTAAPIKeyInfoEPL
	getUTAAccountModeEPL
	setUTAAccountModeEPL
	getUTALeverageEPL
	modifyUTAFuturesLeverageEPL
	modifyUTACrossMarginLeverageEPL
	getUTABorrowingRatesAndLimitsEPL
	getUTAAccountLedgerEPL
	getUTAInterestHistoryEPL
	getUTAAPIRateLimitEPL
	getUTAAllAPIRateLimitEPL
	getUTAAPIRateLimitCapEPL
	getUTAFeeRateEPL
	setUTAKCSFeeDeductionEPL
	getUTAThirdPartyCustodyAccountCurrencyLimitsEPL

	// UTA REST V2 > Account Info > Sub Account
	getUTASubAccountCurrencyAssetsEPL
	getUTASubAccountListEPL
	getUTASubAccountAPIListEPL
	setUTASubAccountTransferPermissionEPL
	addUTASubAccountEPL
	addUTASubAccountAPIEPL
	modifyUTASubAccountAPIEPL
	setUTASubAccountsAPIRateLimitEPL
	deleteUTASubAccountAPIEPL

	// UTA REST V2 > Account Info > Deposit
	getUTADepositAddressEPL
	getUTADepositHistoryEPL

	// UTA REST V2 > Account Info > Transfer
	getUTATransferQuotasEPL
	utaFlexTransferEPL

	// UTA REST V2 > Account Info > Withdraw
	getUTAWithdrawalQuotasEPL
	getUTAWithdrawalHistoryEPL
	utaWithdrawEPL
	cancelUTAWithdrawalEPL

	// UTA REST V2 > Orders
	placeUTAOrderEPL
	amendUTAOrderEPL
	cancelUTAOrderEPL
	batchCancelUTAOrdersByIDEPL
	batchCancelUTAOrdersBySymbolEPL
	getUTAOrderDetailsEPL
	getUTAOpenOrderListEPL
	getUTAOrderHistoryEPL
	getUTATradeHistoryEPL
	getUTAMaxOrderQuantityEPL

	// UTA REST V2 > Positions
	getUTAPositionListEPL
	getUTAPositionsHistoryEPL
	getUTAPrivateFundingFeeHistoryEPL
	modifyUTAMarginModeEPL
	modifyUTAPositionMarginEPL
	getUTAMarginModeEPL

	// UTA REST V2 > Affiliate
	getUTAAffiliateInviteesEPL
	getUTAAffiliateCommissionEPL
	getUTAAffiliateTradeHistoryEPL
	getUTAAffiliateTransactionsEPL
	getUTAAffiliateKuminingEPL

	// UTA REST V2 > VIP Lending
	getUTAVIPLendingCollateralRatioEPL
	getUTAVIPLendingLoanInfoEPL
	getUTAVIPLendingAccountsEPL

	// Classic REST > Account Info > Account & Funding
	getAccountSummaryInformationEPL
	getAPIKeyInfoEPL
	getSpotAccountTypeEPL
	getAllAccountsEPL
	getAccountDetailEPL
	getCrossMarginAccountsDetailEPL
	getIsolatedMarginAccountDetailEPL
	getFuturesAccountOverviewEPL
	getAccountLedgersEPL
	getAccountLedgersHFTradeEPL
	getAccountLedgerHFMarginEPL
	getFuturesAccountLedgersEPL

	// Classic REST > Account Info > Sub Account
	createSubUserEPL
	addSubAccountMarginPermissionEPL
	addSubAccountFuturesPermissionEPL
	getAllSubAccountsInfoV2EPL
	getSubAccountDetailBalanceEPL
	getAllSubAccountsBalanceV2EPL
	getAllFuturesSubAccountBalancesEPL

	// Classic REST > Account Info > Sub Account API
	getSubAccountAPIListEPL
	createSubAccountAPIKeyEPL
	modifySubAccountAPIKeyEPL
	deleteSubAccountAPIKeyEPL

	// Classic REST > Account Info > Deposit
	addDepositAddressV3EPL
	getDepositAddressV3EPL
	getDepositHistoryEPL

	// Classic REST > Account Info > Withdrawals
	getWithdrawalQuotasEPL
	withdrawV3EPL
	cancelWithdrawalEPL
	getWithdrawalHistoryEPL
	getWithdrawalHistoryByIDEPL

	// Classic REST > Account Info > Transfer
	getTransferQuotasEPL
	flexTransferEPL

	// Classic REST > Account Info > Trade Fee
	getBasicFeeEPL
	getTradingPairActualFeesEPL
	getFuturesTradingPairsActualFeesEPL

	// Classic REST > Spot Trading > Market Data
	getAnnouncementsEPL
	getCurrencyDetailV3EPL
	getCurrenciesV3EPL
	getSymbolEPL
	getSymbolsEPL
	getTickerEPL
	getTickersEPL
	getTradeHistoryEPL
	getKlinesEPL
	getPartOrderbookEPL
	getOrderbookEPL
	getCallAuctionPartOrderbookEPL
	getCallAuctionInfoEPL
	getFiatPriceEPL
	get24HourStatsEPL
	getMarketListEPL
	getClientIPAddressEPL
	getCurrentServerTimeEPL
	getServiceStatusEPL
	getKYCRegionsEPL

	// Classic REST > Spot Trading > Orders
	placeHFOrderEPL
	placeHFOrderSyncEPL
	placeHFOrderTestEPL
	batchPlaceHFOrdersEPL
	batchPlaceHFOrdersSyncEPL
	cancelHFOrderByOrderIDEPL
	cancelHFOrderByOrderIDSyncEPL
	cancelHFOrderByClientOrderIDEPL
	cancelHFOrderByClientOrderIDSyncEPL
	cancelPartialHFOrderEPL
	cancelAllHFOrdersBySymbolEPL
	cancelAllHFOrdersEPL
	modifyHFOrderEPL
	getHFOrderByOrderIDEPL
	getHFOrderByClientOrderIDEPL
	getSymbolsWithOpenHFOrderEPL
	getOpenHFOrdersEPL
	getOpenHFOrdersByPageEPL
	getClosedHFOrdersEPL
	getHFTradeHistoryEPL
	getDCPEPL
	setDCPEPL
	placeStopOrderEPL
	batchCancelStopOrdersEPL
	getStopOrderListEPL
	getStopOrderByOrderIDEPL
	getStopOrderByClientOrderIDEPL
	placeOCOOrderEPL
	cancelOCOOrderByOrderIDEPL
	cancelOCOOrderByClientOrderIDEPL
	cancelOCOMultipleOrdersEPL
	getOCOOrderListEPL

	// Classic REST > Margin Trading > Market Data
	getCrossMarginSymbolsEPL
	getIsolatedMarginSymbolsEPL
	getMarkPriceEPL
	getMarginConfigurationEPL
	getAllMarginTradingPairsMarkPricesEPL
	getMarginCollateralRatioEPL
	getMarketAvailableInventoryEPL

	// Classic REST > Margin Trading > Orders
	placeMarginHFOrderEPL
	placeMarginHFOrderTestEPL
	cancelMarginHFOrderByOrderIDEPL
	cancelMarginHFOrderByClientOrderIDEPL
	cancelAllMarginHFOrdersBySymbolEPL
	getActiveMarginHFOrderSymbolsEPL
	getActiveMarginHFOrdersEPL
	getClosedMarginHFOrdersEPL
	getMarginHFTradeFillsEPL
	getMarginHFOrderDetailByOrderIDEPL
	getMarginHFOrderDetailByClientOrderIDEPL
	placeMarginStopOrderEPL
	cancelMarginStopOrderByOrderIDEPL
	cancelMarginStopOrderByClientOrderIDEPL
	cancelMarginStopOrdersEPL
	getMarginStopOrdersEPL
	getMarginStopOrderByOrderIDEPL
	getMarginStopOrderByClientOrderIDEPL
	placeMarginOCOOrderEPL
	cancelMarginOCOOrderByOrderIDEPL
	cancelMarginOCOOrderByClientOrderIDEPL
	cancelMarginOCOOrdersEPL
	getMarginOCOOrderByOrderIDEPL
	getMarginOCOOrderByClientOrderIDEPL
	getMarginOCOOrderDetailByOrderIDEPL
	getMarginOCOOrdersEPL

	// Classic REST > Margin Trading > Debit
	getBorrowInterestRateEPL
	postMarginBorrowOrderEPL
	getMarginBorrowingHistoryEPL
	postRepaymentEPL
	getRepaymentHistoryEPL
	getCrossIsolatedMarginInterestRecordsEPL
	modifyLeverageMultiplierEPL

	// Classic REST > Margin Trading > Credit
	getLendingCurrencyInformationEPL
	getLoanMarketInterestRateEPL
	marginLendingSubscriptionEPL
	modifySubscriptionOrderEPL
	getSubscriptionOrdersEPL
	redemptionEPL
	getRedemptionOrdersEPL

	// Classic REST > Margin Trading > Risk Limit
	getMarginRiskLimitEPL

	// Classic REST > Futures Trading > Market Data
	getFuturesContractEPL
	getFuturesOpenContractsEPL
	getFuturesTickerEPL
	getLatestTickersForAllContractsEPL
	getFuturesOrderbookEPL
	getFuturesPartOrderbookEPL
	getFuturesTradeHistoryEPL
	getFuturesKlineEPL
	getFuturesCurrentMarkPriceEPL
	getFuturesIndexListEPL
	getFuturesInterestRateEPL
	getFuturesPremiumIndexEPL
	get24HourFuturesTransactionVolumeEPL
	getFuturesServerTimeEPL
	getFuturesServiceStatusEPL

	// Classic REST > Futures Trading > Orders
	postFuturesOrderEPL
	postFuturesOrderTestEPL
	placeMultipleFuturesOrdersEPL
	postFuturesTakeProfitStopLossOrderEPL
	cancelFuturesOrderByOrderIDEPL
	cancelFuturesOrderByClientOrderIDEPL
	batchCancelFuturesOrdersEPL
	cancelAllFuturesOrdersEPL
	cancelAllFuturesStopOrdersEPL
	getFuturesOrderDetailsByOrderIDEPL
	getFuturesOrderDetailsByClientOrderIDEPL
	getFuturesOrdersEPL
	getFuturesRecentCompletedOrdersEPL
	getUntriggeredFuturesStopOrdersEPL
	getFuturesOpenOrderStatsEPL
	getFuturesRecentFillsEPL
	getFuturesFillsEPL

	// Classic REST > Futures Trading > Positions
	getFuturesMarginModeEPL
	switchFuturesMarginModeEPL
	batchSwitchFuturesMarginModeEPL
	getFuturesPositionModeEPL
	switchFuturesPositionModeEPL
	getMaximumOpenPositionSizeEPL
	getFuturesPositionDetailsEPL
	getFuturesPositionListEPL
	getPositionHistoryEPL
	getMaxWithdrawMarginEPL
	getFuturesCrossMarginLeverageEPL
	modifyFuturesCrossMarginLeverageEPL
	addMarginEPL
	removeMarginManuallyEPL
	getFuturesCrossMarginRiskLimitEPL
	getFuturesCrossMarginRequirementEPL
	getFuturesRiskLimitLevelEPL
	updateFuturesRiskLimitLevelEPL

	// Classic REST > Futures Trading > Funding Fees
	getFuturesCurrentFundingRateEPL
	getPublicFundingRateEPL
	getFuturesFundingHistoryEPL

	// Classic REST > Earn > Simple Earn
	subscribeToEarnFixedIncomeProductEPL
	getEarnRedeemPreviewByHoldingIDEPL
	redeemByEarnHoldingIDEPL
	getEarnSavingsProductsEPL
	getLimitedTimePromotionProductsEPL
	getEarnStakingProductsEPL
	getEarnKCSStakingProductsEPL
	getEarnETHStakingProductsEPL
	getEarnFixedIncomeCurrentHoldingsEPL

	// Classic REST > Earn > Structured Earn - Dual
	purchaseStructuredProductEPL
	getDualInvestmentProductsEPL
	getStructuredProductOrdersEPL

	// Classic REST > VIP Lending
	getVIPLendingCollateralRatioEPL
	getInformationOnOffExchangeFundingAndLoansEPL
	getInformationOnAccountInvolvedInOffExchangeLoansEPL

	// Classic REST > Affiliate
	getAffiliateInviteesEPL
	getAffiliateCommissionEPL
	getAffiliateTradeHistoryEPL
	getAffiliateTransactionsEPL
	getAffiliateKuminingEPL

	// Classic REST > Broker > Broker Pro
	applyForFastWithdrawalEPL
	getBrokerProRebateEPL
	getBrokerProCommissionEPL
	getBrokerProUserListEPL
	getBrokerProUserTransactionsEPL

	// Classic REST > Broker > Exchange Broker
	getExchangeBrokerRebateEPL
	getExchangeBrokerRebateV3EPL
	submitExchangeBrokerKYCEPL
	getExchangeBrokerKYCStatusEPL
	getExchangeBrokerKYCStatusListEPL
	getExchangeBrokerInfoEPL
	addExchangeBrokerSubAccountEPL
	getExchangeBrokerSubAccountsEPL
	addExchangeBrokerSubAccountAPIEPL
	getExchangeBrokerSubAccountAPIsEPL
	modifyExchangeBrokerSubAccountAPIEPL
	deleteExchangeBrokerSubAccountAPIEPL
	exchangeBrokerTransferEPL
	getExchangeBrokerTransferHistoryEPL
	getExchangeBrokerDepositListEPL
	getExchangeBrokerDepositDetailEPL
	getExchangeBrokerWithdrawDetailEPL
	setExchangeBrokerMarkupFeeEPL
	getExchangeBrokerMarkupFeeEPL

	// Classic REST > Copy Trading
	addCopyTradingOrderEPL
	addCopyTradingOrderTestEPL
	addCopyTradingTPSLOrderEPL
	cancelCopyTradingOrderByIDEPL
	cancelCopyTradingOrderByClientOrderIDEPL
	getCopyTradingMaxOpenSizeEPL
	getCopyTradingMaxWithdrawMarginEPL
	addCopyTradingIsolatedMarginEPL
	removeCopyTradingIsolatedMarginEPL
	modifyCopyTradingIsolatedMarginRiskLimitEPL
	modifyCopyTradingAutoDepositStatusEPL
	switchCopyTradingMarginModeEPL
	modifyCopyTradingCrossMarginLeverageEPL
	getCopyTradingCrossMarginRequirementEPL
	switchCopyTradingPositionModeEPL
	getCopyTradingActiveOrdersEPL
	getCopyTradingFillsEPL
	getCopyTradingPositionsEPL
	getCopyTradingOrderHistoryEPL
	getCopyTradingAccountOverviewEPL
	batchSwitchCopyTradingMarginModeEPL

	// Classic REST > Convert
	getConvertSymbolEPL
	getConvertCurrenciesEPL
	getConvertQuoteEPL
	addConvertOrderEPL
	getConvertOrderDetailEPL
	getConvertOrderHistoryEPL
	addConvertLimitOrderEPL
	getConvertLimitQuoteEPL
	getConvertLimitOrderDetailEPL
	getConvertLimitOrdersEPL
	cancelConvertLimitOrderEPL

	// Classic WebSocket > Base Info
	getPublicTokenEPL
	getPrivateTokenEPL
	getFuturesPublicTokenEPL
	getFuturesPrivateTokenEPL
)

// endpointWeights holds the documented pool and weight of every endpoint limit
var endpointWeights = map[request.EndpointLimit]endpointWeight{
	getUTAAnnouncementsEPL:                               {publicPool, 20},
	getUTACurrencyEPL:                                    {publicPool, 3},
	getUTACurrenciesEPL:                                  {publicPool, 3},
	getUTATickerEPL:                                      {publicPool, 15},
	getUTASymbolEPL:                                      {publicPool, 4},
	getUTAKlinesEPL:                                      {publicPool, 3},
	getUTAOrderbookEPL:                                   {managementPool, 3},
	getUTAIndexPriceEPL:                                  {publicPool, 2},
	getUTATradesEPL:                                      {publicPool, 3},
	getUTAPositionTiersEPL:                               {publicPool, 15},
	getUTACollateralRatioEPL:                             {publicPool, 10},
	getUTABorrowableCurrenciesEPL:                        {publicPool, 15},
	getUTACurrentFundingRateEPL:                          {publicPool, 2},
	getUTAFundingRateHistoryEPL:                          {publicPool, 5},
	getUTAFuturesOpenInterestEPL:                         {publicPool, 10},
	getUTAInterestRateIndexEPL:                           {publicPool, 3},
	getUTAPlatform24HourMarketStatisticsEPL:              {publicPool, 3},
	getUTACallAuctionInfoEPL:                             {publicPool, 2},
	getUTAFiatPriceEPL:                                   {publicPool, 3},
	getUTAThirdPartyCustodyCurrenciesEPL:                 {publicPool, 15},
	getUTAServiceStatusEPL:                               {publicPool, 3},
	getUTAKYCRegionsEPL:                                  {publicPool, 20},
	getUTAClientIPAddressEPL:                             {publicPool, 10},
	getUTAAccountOverviewEPL:                             {managementPool, 5},
	getUTAAccountCurrencyAssetsEPL:                       {managementPool, 5},
	getUTAClassicAccountCurrencyAssetsEPL:                {managementPool, 5},
	getUTAAPIKeyInfoEPL:                                  {managementPool, 30},
	getUTAAccountModeEPL:                                 {managementPool, 30},
	setUTAAccountModeEPL:                                 {managementPool, 30},
	getUTALeverageEPL:                                    {managementPool, 10},
	modifyUTAFuturesLeverageEPL:                          {managementPool, 20},
	modifyUTACrossMarginLeverageEPL:                      {managementPool, 20},
	getUTABorrowingRatesAndLimitsEPL:                     {managementPool, 10},
	getUTAAccountLedgerEPL:                               {managementPool, 2},
	getUTAInterestHistoryEPL:                             {managementPool, 15},
	getUTAAPIRateLimitEPL:                                {managementPool, 3},
	getUTAAllAPIRateLimitEPL:                             {managementPool, 20},
	getUTAAPIRateLimitCapEPL:                             {managementPool, 5},
	getUTAFeeRateEPL:                                     {managementPool, 3},
	setUTAKCSFeeDeductionEPL:                             {managementPool, 5},
	getUTAThirdPartyCustodyAccountCurrencyLimitsEPL:      {managementPool, 20},
	getUTASubAccountCurrencyAssetsEPL:                    {managementPool, 5},
	getUTASubAccountListEPL:                              {managementPool, 2},
	getUTASubAccountAPIListEPL:                           {managementPool, 20},
	setUTASubAccountTransferPermissionEPL:                {managementPool, 15},
	addUTASubAccountEPL:                                  {managementPool, 15},
	addUTASubAccountAPIEPL:                               {managementPool, 20},
	modifyUTASubAccountAPIEPL:                            {managementPool, 30},
	setUTASubAccountsAPIRateLimitEPL:                     {managementPool, 15},
	deleteUTASubAccountAPIEPL:                            {managementPool, 30},
	getUTADepositAddressEPL:                              {managementPool, 5},
	getUTADepositHistoryEPL:                              {managementPool, 2},
	getUTATransferQuotasEPL:                              {managementPool, 20},
	utaFlexTransferEPL:                                   {managementPool, 4},
	getUTAWithdrawalQuotasEPL:                            {managementPool, 1},
	getUTAWithdrawalHistoryEPL:                           {managementPool, 2},
	utaWithdrawEPL:                                       {managementPool, 5},
	cancelUTAWithdrawalEPL:                               {unifiedAccountPool, 5},
	placeUTAOrderEPL:                                     {unifiedAccountPool, 1},
	amendUTAOrderEPL:                                     {unifiedAccountPool, 1},
	cancelUTAOrderEPL:                                    {unifiedAccountPool, 1},
	batchCancelUTAOrdersByIDEPL:                          {unifiedAccountPool, 4},
	batchCancelUTAOrdersBySymbolEPL:                      {unifiedAccountPool, 20},
	getUTAOrderDetailsEPL:                                {managementPool, 4},
	getUTAOpenOrderListEPL:                               {managementPool, 4},
	getUTAOrderHistoryEPL:                                {managementPool, 4},
	getUTATradeHistoryEPL:                                {managementPool, 4},
	getUTAMaxOrderQuantityEPL:                            {managementPool, 10},
	getUTAPositionListEPL:                                {managementPool, 3},
	getUTAPositionsHistoryEPL:                            {managementPool, 2},
	getUTAPrivateFundingFeeHistoryEPL:                    {managementPool, 15},
	modifyUTAMarginModeEPL:                               {managementPool, 10},
	modifyUTAPositionMarginEPL:                           {unifiedAccountPool, 5},
	getUTAMarginModeEPL:                                  {managementPool, 10},
	getUTAAffiliateInviteesEPL:                           {managementPool, 30},
	getUTAAffiliateCommissionEPL:                         {managementPool, 30},
	getUTAAffiliateTradeHistoryEPL:                       {managementPool, 30},
	getUTAAffiliateTransactionsEPL:                       {managementPool, 30},
	getUTAAffiliateKuminingEPL:                           {managementPool, 30},
	getUTAVIPLendingCollateralRatioEPL:                   {managementPool, 10},
	getUTAVIPLendingLoanInfoEPL:                          {managementPool, 5},
	getUTAVIPLendingAccountsEPL:                          {managementPool, 20},
	getAccountSummaryInformationEPL:                      {managementPool, 20},
	getAPIKeyInfoEPL:                                     {managementPool, 20},
	getSpotAccountTypeEPL:                                {spotPool, 30},
	getAllAccountsEPL:                                    {managementPool, 5},
	getAccountDetailEPL:                                  {managementPool, 5},
	getCrossMarginAccountsDetailEPL:                      {spotPool, 15},
	getIsolatedMarginAccountDetailEPL:                    {spotPool, 15},
	getFuturesAccountOverviewEPL:                         {futuresPool, 5},
	getAccountLedgersEPL:                                 {managementPool, 2},
	getAccountLedgersHFTradeEPL:                          {spotPool, 2},
	getAccountLedgerHFMarginEPL:                          {spotPool, 2},
	getFuturesAccountLedgersEPL:                          {futuresPool, 2},
	createSubUserEPL:                                     {managementPool, 15},
	addSubAccountMarginPermissionEPL:                     {managementPool, 15},
	addSubAccountFuturesPermissionEPL:                    {managementPool, 15},
	getAllSubAccountsInfoV2EPL:                           {managementPool, 20},
	getSubAccountDetailBalanceEPL:                        {managementPool, 15},
	getAllSubAccountsBalanceV2EPL:                        {managementPool, 20},
	getAllFuturesSubAccountBalancesEPL:                   {futuresPool, 6},
	getSubAccountAPIListEPL:                              {managementPool, 20},
	createSubAccountAPIKeyEPL:                            {managementPool, 20},
	modifySubAccountAPIKeyEPL:                            {managementPool, 30},
	deleteSubAccountAPIKeyEPL:                            {managementPool, 30},
	addDepositAddressV3EPL:                               {managementPool, 20},
	getDepositAddressV3EPL:                               {managementPool, 5},
	getDepositHistoryEPL:                                 {managementPool, 5},
	getWithdrawalQuotasEPL:                               {managementPool, 20},
	withdrawV3EPL:                                        {managementPool, 5},
	cancelWithdrawalEPL:                                  {managementPool, 20},
	getWithdrawalHistoryEPL:                              {managementPool, 20},
	getWithdrawalHistoryByIDEPL:                          {managementPool, 20},
	getTransferQuotasEPL:                                 {managementPool, 20},
	flexTransferEPL:                                      {managementPool, 4},
	getBasicFeeEPL:                                       {spotPool, 3},
	getTradingPairActualFeesEPL:                          {spotPool, 3},
	getFuturesTradingPairsActualFeesEPL:                  {futuresPool, 3},
	getAnnouncementsEPL:                                  {publicPool, 20},
	getCurrencyDetailV3EPL:                               {publicPool, 3},
	getCurrenciesV3EPL:                                   {publicPool, 3},
	getSymbolEPL:                                         {publicPool, 4},
	getSymbolsEPL:                                        {publicPool, 4},
	getTickerEPL:                                         {publicPool, 2},
	getTickersEPL:                                        {publicPool, 15},
	getTradeHistoryEPL:                                   {publicPool, 3},
	getKlinesEPL:                                         {publicPool, 3},
	getPartOrderbookEPL:                                  {publicPool, 2},
	getOrderbookEPL:                                      {spotPool, 3},
	getCallAuctionPartOrderbookEPL:                       {publicPool, 2},
	getCallAuctionInfoEPL:                                {publicPool, 2},
	getFiatPriceEPL:                                      {publicPool, 3},
	get24HourStatsEPL:                                    {publicPool, 15},
	getMarketListEPL:                                     {publicPool, 3},
	getClientIPAddressEPL:                                {publicPool, 1}, // KuCoin documents weight 0, which the limiter cannot take
	getCurrentServerTimeEPL:                              {publicPool, 3},
	getServiceStatusEPL:                                  {publicPool, 3},
	getKYCRegionsEPL:                                     {publicPool, 20},
	placeHFOrderEPL:                                      {spotPool, 1},
	placeHFOrderSyncEPL:                                  {spotPool, 1},
	placeHFOrderTestEPL:                                  {spotPool, 1},
	batchPlaceHFOrdersEPL:                                {spotPool, 1},
	batchPlaceHFOrdersSyncEPL:                            {spotPool, 1},
	cancelHFOrderByOrderIDEPL:                            {spotPool, 1},
	cancelHFOrderByOrderIDSyncEPL:                        {spotPool, 1},
	cancelHFOrderByClientOrderIDEPL:                      {spotPool, 1},
	cancelHFOrderByClientOrderIDSyncEPL:                  {spotPool, 1},
	cancelPartialHFOrderEPL:                              {spotPool, 2},
	cancelAllHFOrdersBySymbolEPL:                         {spotPool, 2},
	cancelAllHFOrdersEPL:                                 {spotPool, 30},
	modifyHFOrderEPL:                                     {spotPool, 1},
	getHFOrderByOrderIDEPL:                               {spotPool, 2},
	getHFOrderByClientOrderIDEPL:                         {spotPool, 2},
	getSymbolsWithOpenHFOrderEPL:                         {spotPool, 2},
	getOpenHFOrdersEPL:                                   {spotPool, 2},
	getOpenHFOrdersByPageEPL:                             {spotPool, 2},
	getClosedHFOrdersEPL:                                 {spotPool, 2},
	getHFTradeHistoryEPL:                                 {spotPool, 2},
	getDCPEPL:                                            {spotPool, 2},
	setDCPEPL:                                            {spotPool, 2},
	placeStopOrderEPL:                                    {spotPool, 1},
	batchCancelStopOrdersEPL:                             {spotPool, 3},
	getStopOrderListEPL:                                  {spotPool, 8},
	getStopOrderByOrderIDEPL:                             {spotPool, 3},
	getStopOrderByClientOrderIDEPL:                       {spotPool, 3},
	placeOCOOrderEPL:                                     {spotPool, 2},
	cancelOCOOrderByOrderIDEPL:                           {spotPool, 3},
	cancelOCOOrderByClientOrderIDEPL:                     {spotPool, 3},
	cancelOCOMultipleOrdersEPL:                           {spotPool, 3},
	getOCOOrderListEPL:                                   {spotPool, 2},
	getCrossMarginSymbolsEPL:                             {publicPool, 3},
	getIsolatedMarginSymbolsEPL:                          {publicPool, 3},
	getMarkPriceEPL:                                      {publicPool, 2},
	getMarginConfigurationEPL:                            {spotPool, 25},
	getAllMarginTradingPairsMarkPricesEPL:                {publicPool, 10},
	getMarginCollateralRatioEPL:                          {spotPool, 10},
	getMarketAvailableInventoryEPL:                       {publicPool, 10},
	placeMarginHFOrderEPL:                                {spotPool, 2},
	placeMarginHFOrderTestEPL:                            {spotPool, 2},
	cancelMarginHFOrderByOrderIDEPL:                      {spotPool, 2},
	cancelMarginHFOrderByClientOrderIDEPL:                {spotPool, 2},
	cancelAllMarginHFOrdersBySymbolEPL:                   {spotPool, 5},
	getActiveMarginHFOrderSymbolsEPL:                     {spotPool, 2},
	getActiveMarginHFOrdersEPL:                           {spotPool, 4},
	getClosedMarginHFOrdersEPL:                           {spotPool, 10},
	getMarginHFTradeFillsEPL:                             {spotPool, 5},
	getMarginHFOrderDetailByOrderIDEPL:                   {spotPool, 5},
	getMarginHFOrderDetailByClientOrderIDEPL:             {spotPool, 5},
	placeMarginStopOrderEPL:                              {spotPool, 3},
	cancelMarginStopOrderByOrderIDEPL:                    {spotPool, 3},
	cancelMarginStopOrderByClientOrderIDEPL:              {spotPool, 5},
	cancelMarginStopOrdersEPL:                            {spotPool, 3},
	getMarginStopOrdersEPL:                               {spotPool, 8},
	getMarginStopOrderByOrderIDEPL:                       {spotPool, 3},
	getMarginStopOrderByClientOrderIDEPL:                 {spotPool, 3},
	placeMarginOCOOrderEPL:                               {spotPool, 2},
	cancelMarginOCOOrderByOrderIDEPL:                     {spotPool, 3},
	cancelMarginOCOOrderByClientOrderIDEPL:               {spotPool, 3},
	cancelMarginOCOOrdersEPL:                             {spotPool, 3},
	getMarginOCOOrderByOrderIDEPL:                        {spotPool, 2},
	getMarginOCOOrderByClientOrderIDEPL:                  {spotPool, 2},
	getMarginOCOOrderDetailByOrderIDEPL:                  {spotPool, 2},
	getMarginOCOOrdersEPL:                                {spotPool, 2},
	getBorrowInterestRateEPL:                             {spotPool, 5},
	postMarginBorrowOrderEPL:                             {spotPool, 15},
	getMarginBorrowingHistoryEPL:                         {spotPool, 15},
	postRepaymentEPL:                                     {spotPool, 10},
	getRepaymentHistoryEPL:                               {spotPool, 15},
	getCrossIsolatedMarginInterestRecordsEPL:             {spotPool, 20},
	modifyLeverageMultiplierEPL:                          {spotPool, 8},
	getLendingCurrencyInformationEPL:                     {spotPool, 10},
	getLoanMarketInterestRateEPL:                         {publicPool, 5},
	marginLendingSubscriptionEPL:                         {spotPool, 15},
	modifySubscriptionOrderEPL:                           {spotPool, 10},
	getSubscriptionOrdersEPL:                             {spotPool, 10},
	redemptionEPL:                                        {spotPool, 15},
	getRedemptionOrdersEPL:                               {spotPool, 10},
	getMarginRiskLimitEPL:                                {spotPool, 20},
	getFuturesContractEPL:                                {publicPool, 3},
	getFuturesOpenContractsEPL:                           {publicPool, 3},
	getFuturesTickerEPL:                                  {publicPool, 2},
	getLatestTickersForAllContractsEPL:                   {publicPool, 5},
	getFuturesOrderbookEPL:                               {publicPool, 3},
	getFuturesPartOrderbookEPL:                           {publicPool, 5},
	getFuturesTradeHistoryEPL:                            {publicPool, 5},
	getFuturesKlineEPL:                                   {publicPool, 3},
	getFuturesCurrentMarkPriceEPL:                        {publicPool, 3},
	getFuturesIndexListEPL:                               {publicPool, 2},
	getFuturesInterestRateEPL:                            {publicPool, 5},
	getFuturesPremiumIndexEPL:                            {publicPool, 3},
	get24HourFuturesTransactionVolumeEPL:                 {futuresPool, 3},
	getFuturesServerTimeEPL:                              {publicPool, 2},
	getFuturesServiceStatusEPL:                           {publicPool, 4},
	postFuturesOrderEPL:                                  {futuresPool, 2},
	postFuturesOrderTestEPL:                              {futuresPool, 2},
	placeMultipleFuturesOrdersEPL:                        {futuresPool, 20},
	postFuturesTakeProfitStopLossOrderEPL:                {futuresPool, 2},
	cancelFuturesOrderByOrderIDEPL:                       {futuresPool, 1},
	cancelFuturesOrderByClientOrderIDEPL:                 {futuresPool, 1},
	batchCancelFuturesOrdersEPL:                          {futuresPool, 20},
	cancelAllFuturesOrdersEPL:                            {futuresPool, 10},
	cancelAllFuturesStopOrdersEPL:                        {futuresPool, 15},
	getFuturesOrderDetailsByOrderIDEPL:                   {futuresPool, 5},
	getFuturesOrderDetailsByClientOrderIDEPL:             {futuresPool, 5},
	getFuturesOrdersEPL:                                  {futuresPool, 2},
	getFuturesRecentCompletedOrdersEPL:                   {futuresPool, 5},
	getUntriggeredFuturesStopOrdersEPL:                   {futuresPool, 6},
	getFuturesOpenOrderStatsEPL:                          {futuresPool, 10},
	getFuturesRecentFillsEPL:                             {futuresPool, 3},
	getFuturesFillsEPL:                                   {futuresPool, 5},
	getFuturesMarginModeEPL:                              {futuresPool, 2},
	switchFuturesMarginModeEPL:                           {futuresPool, 2},
	batchSwitchFuturesMarginModeEPL:                      {futuresPool, 2},
	getFuturesPositionModeEPL:                            {futuresPool, 2},
	switchFuturesPositionModeEPL:                         {futuresPool, 2},
	getMaximumOpenPositionSizeEPL:                        {futuresPool, 2},
	getFuturesPositionDetailsEPL:                         {futuresPool, 2},
	getFuturesPositionListEPL:                            {futuresPool, 2},
	getPositionHistoryEPL:                                {futuresPool, 2},
	getMaxWithdrawMarginEPL:                              {futuresPool, 10},
	getFuturesCrossMarginLeverageEPL:                     {futuresPool, 2},
	modifyFuturesCrossMarginLeverageEPL:                  {futuresPool, 2},
	addMarginEPL:                                         {futuresPool, 4},
	removeMarginManuallyEPL:                              {futuresPool, 10},
	getFuturesCrossMarginRiskLimitEPL:                    {futuresPool, 2},
	getFuturesCrossMarginRequirementEPL:                  {futuresPool, 3},
	getFuturesRiskLimitLevelEPL:                          {publicPool, 5},
	updateFuturesRiskLimitLevelEPL:                       {futuresPool, 5},
	getFuturesCurrentFundingRateEPL:                      {publicPool, 2},
	getPublicFundingRateEPL:                              {publicPool, 5},
	getFuturesFundingHistoryEPL:                          {futuresPool, 5},
	subscribeToEarnFixedIncomeProductEPL:                 {earnPool, 5},
	getEarnRedeemPreviewByHoldingIDEPL:                   {earnPool, 5},
	redeemByEarnHoldingIDEPL:                             {earnPool, 5},
	getEarnSavingsProductsEPL:                            {earnPool, 5},
	getLimitedTimePromotionProductsEPL:                   {earnPool, 1}, // KuCoin documents no weight
	getEarnStakingProductsEPL:                            {earnPool, 5},
	getEarnKCSStakingProductsEPL:                         {earnPool, 5},
	getEarnETHStakingProductsEPL:                         {earnPool, 5},
	getEarnFixedIncomeCurrentHoldingsEPL:                 {earnPool, 5},
	purchaseStructuredProductEPL:                         {earnPool, 5},
	getDualInvestmentProductsEPL:                         {earnPool, 3},
	getStructuredProductOrdersEPL:                        {earnPool, 5},
	getVIPLendingCollateralRatioEPL:                      {managementPool, 10},
	getInformationOnOffExchangeFundingAndLoansEPL:        {managementPool, 5},
	getInformationOnAccountInvolvedInOffExchangeLoansEPL: {managementPool, 20},
	getAffiliateInviteesEPL:                              {managementPool, 30},
	getAffiliateCommissionEPL:                            {managementPool, 30},
	getAffiliateTradeHistoryEPL:                          {managementPool, 30},
	getAffiliateTransactionsEPL:                          {managementPool, 30},
	getAffiliateKuminingEPL:                              {managementPool, 30},
	applyForFastWithdrawalEPL:                            {managementPool, 5},
	getBrokerProRebateEPL:                                {managementPool, 3},
	getBrokerProCommissionEPL:                            {managementPool, 3},
	getBrokerProUserListEPL:                              {managementPool, 3},
	getBrokerProUserTransactionsEPL:                      {managementPool, 3},
	getExchangeBrokerRebateEPL:                           {brokerPool, 3},
	getExchangeBrokerRebateV3EPL:                         {brokerPool, 20},
	submitExchangeBrokerKYCEPL:                           {brokerPool, 1}, // KuCoin documents no weight
	getExchangeBrokerKYCStatusEPL:                        {brokerPool, 1}, // KuCoin documents no weight
	getExchangeBrokerKYCStatusListEPL:                    {brokerPool, 1}, // KuCoin documents no weight
	getExchangeBrokerInfoEPL:                             {brokerPool, 2},
	addExchangeBrokerSubAccountEPL:                       {brokerPool, 3},
	getExchangeBrokerSubAccountsEPL:                      {brokerPool, 2},
	addExchangeBrokerSubAccountAPIEPL:                    {brokerPool, 3},
	getExchangeBrokerSubAccountAPIsEPL:                   {brokerPool, 2},
	modifyExchangeBrokerSubAccountAPIEPL:                 {brokerPool, 3},
	deleteExchangeBrokerSubAccountAPIEPL:                 {brokerPool, 3},
	exchangeBrokerTransferEPL:                            {brokerPool, 1},
	getExchangeBrokerTransferHistoryEPL:                  {brokerPool, 1},
	getExchangeBrokerDepositListEPL:                      {brokerPool, 10},
	getExchangeBrokerDepositDetailEPL:                    {brokerPool, 1},
	getExchangeBrokerWithdrawDetailEPL:                   {brokerPool, 1},
	setExchangeBrokerMarkupFeeEPL:                        {brokerPool, 3},
	getExchangeBrokerMarkupFeeEPL:                        {brokerPool, 3},
	addCopyTradingOrderEPL:                               {copyTradingPool, 2},
	addCopyTradingOrderTestEPL:                           {copyTradingPool, 2},
	addCopyTradingTPSLOrderEPL:                           {copyTradingPool, 2},
	cancelCopyTradingOrderByIDEPL:                        {copyTradingPool, 1},
	cancelCopyTradingOrderByClientOrderIDEPL:             {copyTradingPool, 1},
	getCopyTradingMaxOpenSizeEPL:                         {copyTradingPool, 4},
	getCopyTradingMaxWithdrawMarginEPL:                   {copyTradingPool, 10},
	addCopyTradingIsolatedMarginEPL:                      {copyTradingPool, 4},
	removeCopyTradingIsolatedMarginEPL:                   {copyTradingPool, 10},
	modifyCopyTradingIsolatedMarginRiskLimitEPL:          {copyTradingPool, 2},
	modifyCopyTradingAutoDepositStatusEPL:                {copyTradingPool, 4},
	switchCopyTradingMarginModeEPL:                       {copyTradingPool, 2},
	modifyCopyTradingCrossMarginLeverageEPL:              {copyTradingPool, 2},
	getCopyTradingCrossMarginRequirementEPL:              {copyTradingPool, 3},
	switchCopyTradingPositionModeEPL:                     {copyTradingPool, 2},
	getCopyTradingActiveOrdersEPL:                        {copyTradingPool, 1},
	getCopyTradingFillsEPL:                               {copyTradingPool, 1},
	getCopyTradingPositionsEPL:                           {copyTradingPool, 1},
	getCopyTradingOrderHistoryEPL:                        {copyTradingPool, 1},
	getCopyTradingAccountOverviewEPL:                     {copyTradingPool, 1},
	batchSwitchCopyTradingMarginModeEPL:                  {copyTradingPool, 1},
	getConvertSymbolEPL:                                  {spotPool, 5},
	getConvertCurrenciesEPL:                              {spotPool, 5},
	getConvertQuoteEPL:                                   {spotPool, 20},
	addConvertOrderEPL:                                   {spotPool, 20},
	getConvertOrderDetailEPL:                             {spotPool, 5},
	getConvertOrderHistoryEPL:                            {spotPool, 5},
	addConvertLimitOrderEPL:                              {spotPool, 20},
	getConvertLimitQuoteEPL:                              {spotPool, 20},
	getConvertLimitOrderDetailEPL:                        {spotPool, 5},
	getConvertLimitOrdersEPL:                             {spotPool, 5},
	cancelConvertLimitOrderEPL:                           {spotPool, 5},
	getPublicTokenEPL:                                    {publicPool, 10},
	getPrivateTokenEPL:                                   {spotPool, 10},
	getFuturesPublicTokenEPL:                             {publicPool, 10},
	getFuturesPrivateTokenEPL:                            {futuresPool, 10},
}

// GetRateLimit returns the rate limit definitions of KuCoin's REST endpoints. Each endpoint draws its weight from
// one limiter its pool's endpoints share
func GetRateLimit() request.RateLimitDefinitions {
	limiters := make(map[*rateLimitPool]*rate.Limiter)
	definitions := make(request.RateLimitDefinitions, len(endpointWeights))
	for epl, w := range endpointWeights {
		if limiters[w.pool] == nil {
			limiters[w.pool] = request.NewRateLimit(w.pool.interval, w.pool.quota)
		}
		definitions[epl] = request.GetRateLimiterWithWeight(limiters[w.pool], w.weight)
	}
	return definitions
}
