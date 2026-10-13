package bybit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Rate limit errors
var (
	errUnknownCategory       = errors.New("unknown category")
	errUnknownOrderOperation = errors.New("unknown order entry operation")
)

// rateLimitBudget is the number of requests Bybit allows in each interval
type rateLimitBudget struct {
	interval time.Duration
	requests int
}

// ipBudget is Bybit's HTTP IP limit. It counts every REST request whatever its endpoint, and exceeding it bans the IP
// for ten minutes
var ipBudget = rateLimitBudget{5 * time.Second, 600}

// Endpoint limits. They start after request.UnAuth, so a stray request.Unset, request.Auth or request.UnAuth finds no
// Bybit limiter instead of silently using one. REST limits follow the order of the docs' rate limit table
const (
	// defaultEPL is for endpoints Bybit publishes no limit for, which draw on the HTTP IP limit alone
	defaultEPL request.EndpointLimit = request.UnAuth + 1 + iota

	// Trade: Bybit budgets order entry for linear and inverse together, and for options and spot apart
	createOrderEPL
	createOptionOrderEPL
	createSpotOrderEPL
	amendOrderEPL
	amendOptionOrderEPL
	amendSpotOrderEPL
	cancelOrderEPL
	cancelOptionOrderEPL
	cancelSpotOrderEPL
	cancelAllOrdersEPL
	cancelAllOptionOrdersEPL
	cancelAllSpotOrdersEPL
	createBatchOrderEPL
	createOptionBatchOrderEPL
	createSpotBatchOrderEPL
	amendBatchOrderEPL
	amendOptionBatchOrderEPL
	amendSpotBatchOrderEPL
	cancelBatchOrderEPL
	cancelOptionBatchOrderEPL
	cancelSpotBatchOrderEPL
	setDisconnectCancelAllEPL
	preCheckOrderEPL
	preCheckOptionOrderEPL
	preCheckSpotOrderEPL
	getOpenOrdersEPL
	getTradeOrderHistoryEPL
	getExecutionEPL
	getBorrowQuotaEPL
	getFuturesComboBotLimitEPL
	createFuturesComboBotEPL
	closeFuturesComboBotEPL
	getFuturesComboBotDetailEPL
	validateFuturesGridInputEPL
	createFuturesGridBotEPL
	closeFuturesGridBotEPL
	getFuturesGridBotDetailEPL
	getFuturesMartingaleBotLimitEPL
	createFuturesMartingaleBotEPL
	closeFuturesMartingaleBotEPL
	getFuturesMartingaleBotDetailEPL
	validateSpotGridInputEPL
	createSpotGridBotEPL
	closeSpotGridBotEPL
	getSpotGridBotDetailEPL
	createDCABotEPL
	closeDCABotEPL
	createStrategyOrderEPL
	getStrategyListEPL
	getStrategyOrderListEPL
	stopStrategyEPL

	// Position
	getPositionInfoEPL
	getClosedPNLEPL
	getClosedOptionsPositionsEPL
	getMovePositionHistoryEPL
	setLeverageLevelEPL
	switchPositionModeEPL
	setTradingStopEPL
	setAutoAddMarginEPL
	addOrReduceMarginEPL
	confirmNewRiskLimitEPL
	movePositionsEPL

	// Account
	getWalletBalanceEPL
	getTransferableAmountEPL
	getBorrowHistoryEPL
	manualBorrowEPL
	manualRepayEPL
	manualRepayWithoutAssetConversionEPL
	getCollateralInfoEPL
	getCoinGreeksEPL
	getTransactionLogEPL
	getFeeRateEPL
	getAccountInfoEPL
	getAccountInstrumentsInfoEPL
	getPayInfoEPL
	getDCPInfoEPL
	getSMPGroupIDEPL
	getTradeInfoForAnalysisEPL
	getTradeBehaviourConfigEPL
	setMMPEPL
	resetMMPEPL
	repayLiabilityEPL
	batchSetCollateralCoinEPL
	setPriceLimitBehaviourEPL
	setMarginModeEPL
	upgradeToUnifiedAccountEPL

	// Asset
	getTransferableCoinEPL
	getInternalTransferRecordsEPL
	getSubUIDEPL
	getUniversalTransferRecordsEPL
	getAllCoinsBalanceEPL
	getSingleCoinBalanceEPL
	getAssetOverviewEPL
	getWithdrawableAmountEPL
	getDepositRecordsEPL
	getSubDepositRecordsEPL
	getMasterDepositAddressEPL
	getSubDepositAddressEPL
	getWithdrawalRecordsEPL
	getCoinInfoEPL
	getCoinExchangeRecordsEPL
	getConvertCoinListEPL
	getSmallBalanceExchangeHistoryEPL
	getSmallBalanceCoinsEPL
	getDeliveryRecordEPL
	getInternalDepositRecordsOffChainEPL
	getFiatBalanceEPL
	getFiatTradingPairListEPL
	getFiatConvertStatusEPL
	getFiatConvertHistoryEPL
	getFundingAccountTransactionHistoryEPL
	getPortfolioMarginInfoEPL
	getUSDCSessionSettlementEPL
	getTotalMembersAssetsEPL
	getAvailableVASPsEPL
	getWithdrawalAddressListEPL
	createInternalTransferEPL
	createUniversalTransferEPL
	withdrawCurrencyEPL
	cancelWithdrawalEPL
	requestQuoteEPL
	confirmQuoteEPL
	confirmSmallBalanceQuoteEPL
	requestSmallBalanceQuoteEPL
	setDepositAccountEPL
	confirmFiatQuoteEPL
	requestFiatQuoteEPL
	getConvertStatusEPL
	getConvertHistoryEPL
	getFiatReferencePriceEPL

	// User
	createNewSubUserIDEPL
	createSubUIDAPIKeyEPL
	freezeSubUIDEPL
	modifyMasterAPIKeyEPL
	modifySubAPIKeyEPL
	deleteMasterAPIKeyEPL
	deleteSubAccountAPIKeyEPL
	getSubUIDListEPL
	getAPIKeyInformationEPL
	getAffiliateUserInfoEPL
	signAgreementEPL
	getSubUIDListUnlimitedEPL
	getFundCustodialSubAccountsEPL
	getSubAccountAllAPIKeysEPL
	getUIDWalletTypeEPL
	deleteSubUIDEPL
	getFriendReferralsEPL

	// Spot margin trade
	toggleMarginTradeEPL
	setSpotMarginTradeLeverageEPL
	getSpotMarginStatusAndLeverageEPL
	getMaxBorrowableAmountEPL
	getSpotMarginCoinStateEPL
	getSpotMarginAvailableAmountToRepayEPL
	setSpotMarginAutoRepayModeEPL
	getSpotMarginAutoRepayModeEPL
	getSpotMarginFixedRateBorrowOrderQuoteEPL
	spotMarginFixedRateBorrowEPL
	renewSpotMarginFixedRateBorrowEPL
	getSpotMarginFixedRateBorrowOrderInfoEPL
	getSpotMarginFixedRateBorrowContractInfoEPL
	getSpotMarginLiabilityEPL

	// Spread trading
	createSpreadOrderEPL
	amendSpreadOrderEPL
	cancelSpreadOrderEPL
	cancelAllSpreadOrdersEPL
	getSpreadOpenOrdersEPL
	getSpreadOrderHistoryEPL
	getSpreadTradeHistoryEPL
	getSpreadMaxQuantityEPL

	// RFQ
	cancelAllRFQQuotesEPL
	cancelRFQQuoteEPL
	acceptNonLPQuoteEPL
	cancelAllRFQsEPL
	createRFQQuoteEPL
	createRFQEPL
	executeRFQQuoteEPL
	cancelRFQEPL
	getRFQPublicTradesEPL
	getRFQQuotesEPL
	getRFQQuotesRealtimeEPL
	getRFQConfigEPL
	getRFQsEPL
	getRFQsRealtimeEPL
	getRFQTradeHistoryEPL
	getRFQDetailsEPL

	// Institutional loan
	getInstitutionalLoanProductInfoEPL
	getInstitutionalLoanMarginCoinInfoEPL
	getInstitutionalLoanOrdersEPL
	getInstitutionalLoanRepaymentOrdersEPL
	getInstitutionalLoanLTVEPL
	bindOrUnbindUIDEPL
	repayInstitutionalLoanEPL
	getInstitutionalLoanDelayLiquidationStatusEPL

	// Endpoints the rate limit table leaves out, which state their limits on their own pages

	// Demo trading, on the demo trading host
	requestDemoTradingFundsEPL
	createDemoAccountEPL

	// Tax
	batchCreateTaxReportsEPL
	getBatchTaxReportsStatusEPL

	// Crypto loan
	getCryptoLoanMaxCollateralReductionAmountEPL
	adjustCryptoLoanCollateralAmountEPL
	getCryptoLoanCollateralAdjustmentHistoryEPL
	getCryptoLoanPositionEPL
	getCryptoLoanMaxLoanAmountEPL
	getFlexibleLoanAvailableInventoryEPL
	borrowFlexibleLoanEPL
	repayFlexibleLoanEPL
	repayFlexibleLoanWithCollateralEPL
	getFlexibleLoansEPL
	getFlexibleLoanBorrowingHistoryEPL
	getFlexibleLoanRepaymentHistoryEPL
	getFixedLoanAvailableInventoryEPL
	createFixedLoanBorrowOrderEPL
	renewFixedLoanBorrowOrderEPL
	createFixedLoanSupplyOrderEPL
	cancelFixedLoanBorrowOrderEPL
	cancelFixedLoanSupplyOrderEPL
	getFixedLoanBorrowContractInfoEPL
	getFixedLoanSupplyContractInfoEPL
	getFixedLoanBorrowOrderInfoEPL
	getFixedLoanRenewOrderInfoEPL
	getFixedLoanSupplyOrderInfoEPL
	repayFixedLoanEPL
	repayFixedLoanWithCollateralEPL
	getFixedLoanRepaymentHistoryEPL

	// Exchange broker
	getExchangeBrokerEarningEPL
	getExchangeBrokerAccountInfoEPL
	getBrokerSubAccountDepositRecordsEPL
	setBrokerRateLimitEPL
	getBrokerRateLimitCapEPL
	getBrokerAllRateLimitsEPL

	// Earn
	getEarnCouponListEPL
	getFixedSavingProductInfoEPL
	placeFixedSavingOrderEPL
	redeemFixedSavingEPL
	getFixedSavingPositionInfoEPL
	getFixedSavingOrderListEPL
	setFixedSavingAutoInvestEPL
	getFlexibleSavingAutoSavingsEPL
	setFlexibleSavingAutoSavingsEPL

	// Advanced earn
	getAdvancedEarnProductInfoEPL
	getAdvancedEarnProductQuoteEPL
	placeAdvancedEarnOrderEPL
	getAdvancedEarnPositionInfoEPL
	getAdvancedEarnOrderInfoEPL
	getAdvancedEarnRedeemEstimatedAmountEPL
	getLiquidityMiningProductInfoEPL
	addLiquidityEPL
	removeLiquidityEPL
	reinvestLiquidityMiningEPL
	addLiquidityMiningMarginEPL
	claimLiquidityMiningInterestEPL
	getLiquidityMiningPositionInfoEPL
	getLiquidityMiningOrderInfoEPL
	getLiquidityMiningYieldRecordsEPL
	getLiquidityMiningLiquidationRecordsEPL

	// Real world assets
	getRWAProductListEPL
	placeRWAOrderEPL
	getRWAPositionListEPL
	getRWAOrderListEPL
	getRWANAVChartEPL

	// Web3 liquidity pools, per user
	executeWeb3LPStakeEPL
	executeWeb3LPRedeemEPL
	getWeb3LPOrderListEPL
	getWeb3LPPayTokenListEPL
	getWeb3LPPayTokenPriceEPL
	getWeb3LPPoolInfoEPL
	getWeb3LPPoolListEPL
	getWeb3LPPositionListEPL

	// Rate limits for pro accounts
	setAPIRateLimitEPL
	getAPIRateLimitEPL
	getAPIRateLimitCapEPL
	getAllAPIRateLimitsEPL

	// P2P: Bybit's defaults for reads and writes, which an account's X-Bapi-Limit header may raise
	getP2POnlineAdsEPL
	postP2PAdEPL
	removeP2PAdEPL
	updateP2PAdEPL
	getP2PMyAdsEPL
	getP2PMyAdDetailsEPL
	getAllP2POrdersEPL
	getP2POrderDetailEPL
	getP2PPendingOrdersEPL
	markP2POrderAsPaidEPL
	releaseP2PAssetsEPL
	reviewP2PSellerCancelOrderApplyEPL
	getP2PAccountInformationEPL
	getP2PCounterpartyUserInfoEPL
	getP2PUserPaymentEPL
	getP2PChatSessionListEPL
	getP2PChatSessionIDEPL
	uploadP2PChatFileEPL
	sendP2PChatMessageEPL
	getP2PChatMessageListEPL

	// Websocket order entry, which draws on the budgets of the REST order entry limits but not on the HTTP IP limit
	wsCreateOrderEPL
	wsCreateOptionOrderEPL
	wsCreateSpotOrderEPL
	wsAmendOrderEPL
	wsAmendOptionOrderEPL
	wsAmendSpotOrderEPL
	wsCancelOrderEPL
	wsCancelOptionOrderEPL
	wsCancelSpotOrderEPL
	wsCreateBatchOrderEPL
	wsCreateOptionBatchOrderEPL
	wsCreateSpotBatchOrderEPL
	wsAmendBatchOrderEPL
	wsAmendOptionBatchOrderEPL
	wsAmendSpotBatchOrderEPL
	wsCancelBatchOrderEPL
	wsCancelOptionBatchOrderEPL
	wsCancelSpotBatchOrderEPL

	// wsSubscriptionEPL is for websocket subscriptions, authentication and pings, which Bybit sets no rate for
	wsSubscriptionEPL
)

// endpointBudgets holds each REST endpoint limit's own budget, per UID unless noted, from Bybit's rate limit table, or
// from the endpoint's page where the table leaves it out. Where the table merges categories into one cell, they share
// the budget. These are the default budgets, which Bybit raises for VIP and pro accounts on the order entry endpoints
var endpointBudgets = map[request.EndpointLimit]rateLimitBudget{
	// Trade: Bybit budgets order entry for linear and inverse together, and for options and spot apart
	createOrderEPL:                   {time.Second, 10},
	createOptionOrderEPL:             {time.Second, 10},
	createSpotOrderEPL:               {time.Second, 20},
	amendOrderEPL:                    {time.Second, 10},
	amendOptionOrderEPL:              {time.Second, 10},
	amendSpotOrderEPL:                {time.Second, 10},
	cancelOrderEPL:                   {time.Second, 10},
	cancelOptionOrderEPL:             {time.Second, 10},
	cancelSpotOrderEPL:               {time.Second, 20},
	cancelAllOrdersEPL:               {time.Second, 10},
	cancelAllOptionOrdersEPL:         {time.Second, 1},
	cancelAllSpotOrdersEPL:           {time.Second, 20},
	createBatchOrderEPL:              {time.Second, 10},
	createOptionBatchOrderEPL:        {time.Second, 10},
	createSpotBatchOrderEPL:          {time.Second, 20},
	amendBatchOrderEPL:               {time.Second, 10},
	amendOptionBatchOrderEPL:         {time.Second, 10},
	amendSpotBatchOrderEPL:           {time.Second, 20},
	cancelBatchOrderEPL:              {time.Second, 10},
	cancelOptionBatchOrderEPL:        {time.Second, 10},
	cancelSpotBatchOrderEPL:          {time.Second, 20},
	setDisconnectCancelAllEPL:        {time.Second, 5},
	preCheckOrderEPL:                 {time.Second, 10},
	preCheckOptionOrderEPL:           {time.Second, 10},
	preCheckSpotOrderEPL:             {time.Second, 20},
	getOpenOrdersEPL:                 {time.Second, 50},
	getTradeOrderHistoryEPL:          {time.Second, 50},
	getExecutionEPL:                  {time.Second, 50},
	getBorrowQuotaEPL:                {time.Second, 50},
	getFuturesComboBotLimitEPL:       {time.Second, 10},
	createFuturesComboBotEPL:         {time.Second, 10},
	closeFuturesComboBotEPL:          {time.Second, 10},
	getFuturesComboBotDetailEPL:      {time.Second, 10},
	validateFuturesGridInputEPL:      {time.Second, 10},
	createFuturesGridBotEPL:          {time.Second, 10},
	closeFuturesGridBotEPL:           {time.Second, 10},
	getFuturesGridBotDetailEPL:       {time.Second, 10},
	getFuturesMartingaleBotLimitEPL:  {time.Second, 10},  // the table's limit; its page states 100 a second per IP
	createFuturesMartingaleBotEPL:    {time.Second, 100}, // the table's limit; its page states 10 a second
	closeFuturesMartingaleBotEPL:     {time.Second, 10},
	getFuturesMartingaleBotDetailEPL: {time.Second, 10},
	validateSpotGridInputEPL:         {time.Second, 100}, // per IP, its page says
	createSpotGridBotEPL:             {time.Second, 3},
	closeSpotGridBotEPL:              {time.Second, 3},
	getSpotGridBotDetailEPL:          {time.Second, 10},
	createDCABotEPL:                  {time.Second, 3},
	closeDCABotEPL:                   {time.Second, 3},
	createStrategyOrderEPL:           {time.Second, 100},
	getStrategyListEPL:               {time.Second, 200},
	getStrategyOrderListEPL:          {time.Second, 200},
	stopStrategyEPL:                  {time.Second, 100},

	// Position
	getPositionInfoEPL:           {time.Second, 50},
	getClosedPNLEPL:              {time.Second, 50},
	getClosedOptionsPositionsEPL: {time.Second, 50},
	getMovePositionHistoryEPL:    {time.Second, 10},
	setLeverageLevelEPL:          {time.Second, 10},
	switchPositionModeEPL:        {time.Second, 10},
	setTradingStopEPL:            {time.Second, 10},
	setAutoAddMarginEPL:          {time.Second, 10},
	addOrReduceMarginEPL:         {time.Second, 10},
	confirmNewRiskLimitEPL:       {time.Second, 10},
	movePositionsEPL:             {time.Second, 10},

	// Account
	getWalletBalanceEPL:                  {time.Second, 50},
	getTransferableAmountEPL:             {time.Second, 50},
	getBorrowHistoryEPL:                  {time.Second, 50},
	manualBorrowEPL:                      {time.Second, 1},
	manualRepayEPL:                       {time.Second, 1},
	manualRepayWithoutAssetConversionEPL: {time.Second, 1},
	getCollateralInfoEPL:                 {time.Second, 50},
	getCoinGreeksEPL:                     {time.Second, 50},
	getTransactionLogEPL:                 {time.Second, 25}, // lowered from 50 a second on 2026-05-21
	getFeeRateEPL:                        {time.Second, 5},
	getAccountInfoEPL:                    {time.Second, 50},
	getAccountInstrumentsInfoEPL:         {time.Second, 10},
	getPayInfoEPL:                        {time.Second, 50},
	getDCPInfoEPL:                        {time.Second, 5},
	getSMPGroupIDEPL:                     {time.Second, 5},
	getTradeInfoForAnalysisEPL:           {time.Second, 50},
	getTradeBehaviourConfigEPL:           {time.Second, 50},
	setMMPEPL:                            {time.Second, 5},
	resetMMPEPL:                          {time.Second, 5},
	repayLiabilityEPL:                    {time.Second, 1},
	batchSetCollateralCoinEPL:            {time.Second, 5},
	setPriceLimitBehaviourEPL:            {time.Second, 10},
	setMarginModeEPL:                     {time.Second, 5},
	upgradeToUnifiedAccountEPL:           {time.Second, 1},

	// Asset
	getTransferableCoinEPL:                 {time.Minute, 60},
	getInternalTransferRecordsEPL:          {time.Minute, 60},
	getSubUIDEPL:                           {time.Minute, 60},
	getUniversalTransferRecordsEPL:         {time.Second, 5},
	getAllCoinsBalanceEPL:                  {time.Second, 5},
	getSingleCoinBalanceEPL:                {time.Second, 450},
	getAssetOverviewEPL:                    {time.Second, 50},
	getWithdrawableAmountEPL:               {time.Second, 300},
	getDepositRecordsEPL:                   {time.Minute, 100},
	getSubDepositRecordsEPL:                {time.Minute, 300},
	getMasterDepositAddressEPL:             {time.Minute, 300},
	getSubDepositAddressEPL:                {time.Minute, 300},
	getWithdrawalRecordsEPL:                {time.Minute, 300},
	getCoinInfoEPL:                         {time.Second, 5},
	getCoinExchangeRecordsEPL:              {time.Minute, 600},
	getConvertCoinListEPL:                  {time.Second, 30}, // the table's limit; the Convert guideline's older table allows 100 a second
	getSmallBalanceExchangeHistoryEPL:      {time.Second, 5},  // the table's limit; its page allows 10 a second
	getSmallBalanceCoinsEPL:                {time.Second, 10},
	getDeliveryRecordEPL:                   {time.Second, 50},
	getInternalDepositRecordsOffChainEPL:   {time.Second, 300},
	getFiatBalanceEPL:                      {time.Second, 1000},
	getFiatTradingPairListEPL:              {time.Second, 1000},
	getFiatConvertStatusEPL:                {time.Second, 1000},
	getFiatConvertHistoryEPL:               {time.Second, 1000},
	getFundingAccountTransactionHistoryEPL: {time.Second, 30},
	getPortfolioMarginInfoEPL:              {time.Second, 50},
	getUSDCSessionSettlementEPL:            {time.Second, 50},
	getTotalMembersAssetsEPL:               {time.Second, 50},
	getAvailableVASPsEPL:                   {time.Second, 1},
	getWithdrawalAddressListEPL:            {time.Second, 300},
	createInternalTransferEPL:              {time.Minute, 60},
	createUniversalTransferEPL:             {time.Second, 5},
	withdrawCurrencyEPL:                    {time.Second, 5}, // Bybit also allows one withdrawal every 10 seconds for each chain and coin
	cancelWithdrawalEPL:                    {time.Minute, 60},
	requestQuoteEPL:                        {time.Second, 20}, // the table's limit; the Convert guideline allows 50 a second
	confirmQuoteEPL:                        {time.Second, 20}, // the table's limit; the Convert guideline allows 50 a second
	confirmSmallBalanceQuoteEPL:            {time.Second, 5},
	requestSmallBalanceQuoteEPL:            {time.Second, 5},
	setDepositAccountEPL:                   {time.Second, 300},
	confirmFiatQuoteEPL:                    {time.Second, 100},
	requestFiatQuoteEPL:                    {time.Second, 1000},
	getConvertStatusEPL:                    {time.Second, 50}, // the table's limit; the Convert guideline allows 100 a second
	getConvertHistoryEPL:                   {time.Second, 50}, // the table's limit; the Convert guideline allows 100 a second
	getFiatReferencePriceEPL:               {time.Second, 1000},

	// User
	createNewSubUserIDEPL:          {time.Second, 1},
	createSubUIDAPIKeyEPL:          {time.Second, 1},
	freezeSubUIDEPL:                {time.Second, 5},
	modifyMasterAPIKeyEPL:          {time.Second, 5},
	modifySubAPIKeyEPL:             {time.Second, 5},
	deleteMasterAPIKeyEPL:          {time.Second, 5},
	deleteSubAccountAPIKeyEPL:      {time.Second, 5},
	getSubUIDListEPL:               {time.Second, 10},
	getAPIKeyInformationEPL:        {time.Second, 10},
	getAffiliateUserInfoEPL:        {time.Second, 10},
	signAgreementEPL:               {time.Second, 20},
	getSubUIDListUnlimitedEPL:      {time.Second, 5},
	getFundCustodialSubAccountsEPL: {time.Second, 5},
	getSubAccountAllAPIKeysEPL:     {time.Second, 10},
	getUIDWalletTypeEPL:            {time.Second, 10},
	deleteSubUIDEPL:                {time.Second, 5},
	getFriendReferralsEPL:          {time.Second, 10},

	// Spot margin trade
	toggleMarginTradeEPL:                        {time.Second, 5},
	setSpotMarginTradeLeverageEPL:               {time.Second, 5},
	getSpotMarginStatusAndLeverageEPL:           {time.Second, 50},
	getMaxBorrowableAmountEPL:                   {time.Second, 50},
	getSpotMarginCoinStateEPL:                   {time.Second, 50},
	getSpotMarginAvailableAmountToRepayEPL:      {time.Second, 50},
	setSpotMarginAutoRepayModeEPL:               {time.Second, 5},
	getSpotMarginAutoRepayModeEPL:               {time.Second, 50},
	getSpotMarginFixedRateBorrowOrderQuoteEPL:   {time.Second, 5},
	spotMarginFixedRateBorrowEPL:                {time.Second, 1},
	renewSpotMarginFixedRateBorrowEPL:           {time.Second, 1},
	getSpotMarginFixedRateBorrowOrderInfoEPL:    {time.Second, 5},
	getSpotMarginFixedRateBorrowContractInfoEPL: {time.Second, 5},
	getSpotMarginLiabilityEPL:                   {time.Second, 5},

	// Spread trading
	createSpreadOrderEPL:     {time.Second, 20},
	amendSpreadOrderEPL:      {time.Second, 20},
	cancelSpreadOrderEPL:     {time.Second, 20},
	cancelAllSpreadOrdersEPL: {time.Second, 5},
	getSpreadOpenOrdersEPL:   {time.Second, 50},
	getSpreadOrderHistoryEPL: {time.Second, 50},
	getSpreadTradeHistoryEPL: {time.Second, 50},
	getSpreadMaxQuantityEPL:  {time.Second, 50},

	// RFQ
	cancelAllRFQQuotesEPL:   {time.Second, 50},
	cancelRFQQuoteEPL:       {time.Second, 50},
	acceptNonLPQuoteEPL:     {time.Second, 50},
	cancelAllRFQsEPL:        {time.Second, 50},
	createRFQQuoteEPL:       {time.Second, 50},
	createRFQEPL:            {time.Second, 50},
	executeRFQQuoteEPL:      {time.Second, 50},
	cancelRFQEPL:            {time.Second, 50},
	getRFQPublicTradesEPL:   {time.Second, 50},
	getRFQQuotesEPL:         {time.Second, 50},
	getRFQQuotesRealtimeEPL: {time.Second, 50},
	getRFQConfigEPL:         {time.Second, 50},
	getRFQsEPL:              {time.Second, 50},
	getRFQsRealtimeEPL:      {time.Second, 50},
	getRFQTradeHistoryEPL:   {time.Second, 50},
	getRFQDetailsEPL:        {time.Second, 50}, // from its page

	// Institutional loan
	getInstitutionalLoanProductInfoEPL:            {time.Second, 100},
	getInstitutionalLoanMarginCoinInfoEPL:         {time.Second, 100},
	getInstitutionalLoanOrdersEPL:                 {time.Second, 20},
	getInstitutionalLoanRepaymentOrdersEPL:        {time.Second, 20},
	getInstitutionalLoanLTVEPL:                    {time.Second, 20},
	bindOrUnbindUIDEPL:                            {time.Second, 1},
	repayInstitutionalLoanEPL:                     {time.Second, 1},
	getInstitutionalLoanDelayLiquidationStatusEPL: {time.Second, 2}, // from its page

	// Endpoints the rate limit table leaves out, which state their limits on their own pages

	// Demo trading, on the demo trading host
	requestDemoTradingFundsEPL: {time.Minute, 1},
	createDemoAccountEPL:       {time.Second, 5},

	// Tax
	batchCreateTaxReportsEPL:    {time.Second, 5},
	getBatchTaxReportsStatusEPL: {time.Second, 20},

	// Crypto loan
	getCryptoLoanMaxCollateralReductionAmountEPL: {time.Second, 5},
	adjustCryptoLoanCollateralAmountEPL:          {time.Second, 1},
	getCryptoLoanCollateralAdjustmentHistoryEPL:  {time.Second, 5},
	getCryptoLoanPositionEPL:                     {time.Second, 5},
	getCryptoLoanMaxLoanAmountEPL:                {time.Second, 5},
	getFlexibleLoanAvailableInventoryEPL:         {time.Second, 5},
	borrowFlexibleLoanEPL:                        {time.Second, 1},
	repayFlexibleLoanEPL:                         {time.Second, 1},
	repayFlexibleLoanWithCollateralEPL:           {time.Second, 1},
	getFlexibleLoansEPL:                          {time.Second, 5},
	getFlexibleLoanBorrowingHistoryEPL:           {time.Second, 5},
	getFlexibleLoanRepaymentHistoryEPL:           {time.Second, 5},
	getFixedLoanAvailableInventoryEPL:            {time.Second, 5},
	createFixedLoanBorrowOrderEPL:                {time.Second, 1},
	renewFixedLoanBorrowOrderEPL:                 {time.Second, 1},
	createFixedLoanSupplyOrderEPL:                {time.Second, 1},
	cancelFixedLoanBorrowOrderEPL:                {time.Second, 1},
	cancelFixedLoanSupplyOrderEPL:                {time.Second, 1},
	getFixedLoanBorrowContractInfoEPL:            {time.Second, 5},
	getFixedLoanSupplyContractInfoEPL:            {time.Second, 5},
	getFixedLoanBorrowOrderInfoEPL:               {time.Second, 5},
	getFixedLoanRenewOrderInfoEPL:                {time.Second, 5},
	getFixedLoanSupplyOrderInfoEPL:               {time.Second, 5},
	repayFixedLoanEPL:                            {time.Second, 1},
	repayFixedLoanWithCollateralEPL:              {time.Second, 1},
	getFixedLoanRepaymentHistoryEPL:              {time.Second, 5},

	// Exchange broker
	getExchangeBrokerEarningEPL:          {time.Second, 10},
	getExchangeBrokerAccountInfoEPL:      {time.Second, 10},
	getBrokerSubAccountDepositRecordsEPL: {time.Minute, 300},
	setBrokerRateLimitEPL:                {time.Second, 1},
	getBrokerRateLimitCapEPL:             {time.Second, 5},
	getBrokerAllRateLimitsEPL:            {time.Second, 1},

	// Earn
	getEarnCouponListEPL:            {time.Second, 10},
	getFixedSavingProductInfoEPL:    {time.Second, 50}, // per IP
	placeFixedSavingOrderEPL:        {time.Second, 5},
	redeemFixedSavingEPL:            {time.Second, 5},
	getFixedSavingPositionInfoEPL:   {time.Second, 10},
	getFixedSavingOrderListEPL:      {time.Second, 10},
	setFixedSavingAutoInvestEPL:     {time.Second, 5},
	getFlexibleSavingAutoSavingsEPL: {time.Second, 10},
	setFlexibleSavingAutoSavingsEPL: {time.Second, 5},

	// Advanced earn
	getAdvancedEarnProductInfoEPL:           {time.Second, 50}, // per IP
	getAdvancedEarnProductQuoteEPL:          {time.Second, 50}, // per IP
	placeAdvancedEarnOrderEPL:               {time.Second, 5},
	getAdvancedEarnPositionInfoEPL:          {time.Second, 10},
	getAdvancedEarnOrderInfoEPL:             {time.Second, 10},
	getAdvancedEarnRedeemEstimatedAmountEPL: {time.Second, 10},
	getLiquidityMiningProductInfoEPL:        {time.Second, 50}, // per IP
	addLiquidityEPL:                         {time.Second, 5},
	removeLiquidityEPL:                      {time.Second, 5},
	reinvestLiquidityMiningEPL:              {time.Second, 5},
	addLiquidityMiningMarginEPL:             {time.Second, 5},
	claimLiquidityMiningInterestEPL:         {time.Second, 5},
	getLiquidityMiningPositionInfoEPL:       {time.Second, 10},
	getLiquidityMiningOrderInfoEPL:          {time.Second, 10},
	getLiquidityMiningYieldRecordsEPL:       {time.Second, 10},
	getLiquidityMiningLiquidationRecordsEPL: {time.Second, 10},

	// Real world assets
	getRWAProductListEPL:  {time.Second, 20}, // per IP
	placeRWAOrderEPL:      {time.Second, 5},
	getRWAPositionListEPL: {time.Second, 10},
	getRWAOrderListEPL:    {time.Second, 10},
	getRWANAVChartEPL:     {time.Second, 20}, // per IP

	// Web3 liquidity pools, per user
	executeWeb3LPStakeEPL:     {time.Second, 1},
	executeWeb3LPRedeemEPL:    {time.Second, 1},
	getWeb3LPOrderListEPL:     {time.Second, 3},
	getWeb3LPPayTokenListEPL:  {time.Second, 5},
	getWeb3LPPayTokenPriceEPL: {time.Second, 5},
	getWeb3LPPoolInfoEPL:      {time.Second, 5},
	getWeb3LPPoolListEPL:      {time.Second, 5},
	getWeb3LPPositionListEPL:  {time.Second, 3},

	// Rate limits for pro accounts
	setAPIRateLimitEPL:     {time.Second, 50},
	getAPIRateLimitEPL:     {time.Second, 50},
	getAPIRateLimitCapEPL:  {time.Second, 50},
	getAllAPIRateLimitsEPL: {time.Second, 50},

	// P2P: Bybit's defaults for reads and writes, which an account's X-Bapi-Limit header may raise
	getP2POnlineAdsEPL:                 {time.Second, 10},
	postP2PAdEPL:                       {time.Second, 5},
	removeP2PAdEPL:                     {time.Second, 5},
	updateP2PAdEPL:                     {time.Second, 5},
	getP2PMyAdsEPL:                     {time.Second, 10},
	getP2PMyAdDetailsEPL:               {time.Second, 10},
	getAllP2POrdersEPL:                 {time.Second, 10},
	getP2POrderDetailEPL:               {time.Second, 10},
	getP2PPendingOrdersEPL:             {time.Second, 10},
	markP2POrderAsPaidEPL:              {time.Second, 5},
	releaseP2PAssetsEPL:                {time.Second, 5},
	reviewP2PSellerCancelOrderApplyEPL: {time.Second, 5},
	getP2PAccountInformationEPL:        {time.Second, 10},
	getP2PCounterpartyUserInfoEPL:      {time.Second, 10},
	getP2PUserPaymentEPL:               {time.Second, 10},
	getP2PChatSessionListEPL:           {time.Second, 10},
	getP2PChatSessionIDEPL:             {time.Second, 10},
	uploadP2PChatFileEPL:               {time.Second, 5},
	sendP2PChatMessageEPL:              {time.Second, 5},
	getP2PChatMessageListEPL:           {time.Second, 10},
}

// orderBudget is one of Bybit's order entry budgets, which REST and websocket requests share
type orderBudget struct {
	// rest also draws on the HTTP IP limit
	rest request.EndpointLimit
	// websocket draws on the budget alone; it is 0 where Bybit offers no websocket operation
	websocket request.EndpointLimit
}

// orderEntryBudgets holds the budgets of an order entry operation: one for linear and inverse together, one for options
// and one for spot
type orderEntryBudgets struct {
	derivatives, option, spot orderBudget
}

// forCategory returns the budget of a category. Categories other than option and spot draw on the linear and inverse
// budget, so a category Bybit rejects spends no other budget
func (b orderEntryBudgets) forCategory(category string) orderBudget {
	switch category {
	case cOption:
		return b.option
	case cSpot:
		return b.spot
	default:
		return b.derivatives
	}
}

// Order entry budgets
var (
	createOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{createOrderEPL, wsCreateOrderEPL},
		option:      orderBudget{createOptionOrderEPL, wsCreateOptionOrderEPL},
		spot:        orderBudget{createSpotOrderEPL, wsCreateSpotOrderEPL},
	}
	amendOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{amendOrderEPL, wsAmendOrderEPL},
		option:      orderBudget{amendOptionOrderEPL, wsAmendOptionOrderEPL},
		spot:        orderBudget{amendSpotOrderEPL, wsAmendSpotOrderEPL},
	}
	cancelOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{cancelOrderEPL, wsCancelOrderEPL},
		option:      orderBudget{cancelOptionOrderEPL, wsCancelOptionOrderEPL},
		spot:        orderBudget{cancelSpotOrderEPL, wsCancelSpotOrderEPL},
	}
	cancelAllOrdersBudgets = orderEntryBudgets{
		derivatives: orderBudget{rest: cancelAllOrdersEPL},
		option:      orderBudget{rest: cancelAllOptionOrdersEPL},
		spot:        orderBudget{rest: cancelAllSpotOrdersEPL},
	}
	createBatchOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{createBatchOrderEPL, wsCreateBatchOrderEPL},
		option:      orderBudget{createOptionBatchOrderEPL, wsCreateOptionBatchOrderEPL},
		spot:        orderBudget{createSpotBatchOrderEPL, wsCreateSpotBatchOrderEPL},
	}
	amendBatchOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{amendBatchOrderEPL, wsAmendBatchOrderEPL},
		option:      orderBudget{amendOptionBatchOrderEPL, wsAmendOptionBatchOrderEPL},
		spot:        orderBudget{amendSpotBatchOrderEPL, wsAmendSpotBatchOrderEPL},
	}
	cancelBatchOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{cancelBatchOrderEPL, wsCancelBatchOrderEPL},
		option:      orderBudget{cancelOptionBatchOrderEPL, wsCancelOptionBatchOrderEPL},
		spot:        orderBudget{cancelSpotBatchOrderEPL, wsCancelSpotBatchOrderEPL},
	}
	preCheckOrderBudgets = orderEntryBudgets{
		derivatives: orderBudget{rest: preCheckOrderEPL},
		option:      orderBudget{rest: preCheckOptionOrderEPL},
		spot:        orderBudget{rest: preCheckSpotOrderEPL},
	}
)

// websocketOrderBudgets holds the budgets of each websocket order entry operation
var websocketOrderBudgets = map[string]orderEntryBudgets{
	"order.create":       createOrderBudgets,
	"order.amend":        amendOrderBudgets,
	"order.cancel":       cancelOrderBudgets,
	"order.create-batch": createBatchOrderBudgets,
	"order.amend-batch":  amendBatchOrderBudgets,
	"order.cancel-batch": cancelBatchOrderBudgets,
}

// rateLimits holds the limits the REST requester and the websocket connections draw on. Exchange instances share
// them, since they send from one IP
var rateLimits = rateLimitDefinitions()

// rateLimitDefinitions returns Bybit's endpoint limits. Every REST limit draws on the HTTP IP limit as well as its own
// budget, and defaultEPL on the IP limit alone. Websocket limits leave the IP limit out, since it counts HTTP requests
func rateLimitDefinitions() request.RateLimitDefinitions {
	ip := request.NewRateLimit(ipBudget.interval, ipBudget.requests)
	limiters := endpointLimiters()
	definitions := make(request.RateLimitDefinitions, len(limiters)+2)
	definitions[defaultEPL] = request.GetRateLimiterWithWeight(ip, 1)
	for epl, limiter := range limiters {
		if _, isREST := endpointBudgets[epl]; isREST {
			definitions[epl] = request.GetRateLimiterWithShared(limiter, 1, ip)
			continue
		}
		definitions[epl] = request.GetRateLimiterWithWeight(limiter, 1)
	}
	// A nil limiter leaves these messages to the connection's own limiter, and Bybit's connections have none
	definitions[wsSubscriptionEPL] = request.RateLimitNotRequired
	return definitions
}

// endpointLimiters returns the limiter of every endpoint limit with a budget: one for each REST limit, which the
// websocket order entry limit drawing on the same budget shares
func endpointLimiters() map[request.EndpointLimit]*rate.Limiter {
	limiters := make(map[request.EndpointLimit]*rate.Limiter, len(endpointBudgets)+3*len(websocketOrderBudgets))
	for epl, budget := range endpointBudgets {
		limiters[epl] = request.NewRateLimit(budget.interval, budget.requests)
	}
	for _, budgets := range websocketOrderBudgets {
		for _, budget := range []orderBudget{budgets.derivatives, budgets.option, budgets.spot} {
			limiters[budget.websocket] = limiters[budget.rest]
		}
	}
	return limiters
}

// getWSRateLimitEPLByCategory returns the endpoint limit of a websocket order entry operation in a category
func getWSRateLimitEPLByCategory(operation, category string) (request.EndpointLimit, error) {
	budgets, ok := websocketOrderBudgets[operation]
	if !ok {
		return 0, fmt.Errorf("%w: %q", errUnknownOrderOperation, operation)
	}
	switch category {
	case cSpot, cLinear, cInverse, cOption:
		return budgets.forCategory(category).websocket, nil
	default:
		return 0, fmt.Errorf("%w: %q", errUnknownCategory, category)
	}
}

// batchOrderLimit returns the endpoint limit a batch order request is sent with, after taking one request from the
// batch's budget for each order beyond the first, which the request itself takes: Bybit counts every order of a linear,
// inverse or spot batch, but an options batch once. The orders draw on the budget through its websocket limit, which
// leaves out the HTTP IP limit, since the batch is one HTTP request
func (e *Exchange) batchOrderLimit(ctx context.Context, budgets orderEntryBudgets, category string, orders int) (request.EndpointLimit, error) {
	budget := budgets.forCategory(category)
	if category != cOption {
		for range orders - 1 {
			if err := e.Requester.InitiateRateLimit(ctx, budget.websocket); err != nil {
				return 0, fmt.Errorf("rate limiting batch orders: %w", err)
			}
		}
	}
	return budget.rest, nil
}
