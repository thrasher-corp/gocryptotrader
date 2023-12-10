package okx

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// oneSecondInterval
	oneSecondInterval = time.Second
	// twoSecondInterval
	twoSecondsInterval = 2 * time.Second
	// threeSecondInterval
	threeSecondsInterval = 3 * time.Second
	// fiveSecondsInterval
	fiveSecondsInterval = 5 * time.Second
	// tenSecondsInterval
	tenSecondsInterval = 10 * time.Second
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	// Trade Endpoints
	PlaceOrder                        *rate.Limiter
	PlaceMultipleOrders               *rate.Limiter
	CancelOrder                       *rate.Limiter
	CancelMultipleOrders              *rate.Limiter
	AmendOrder                        *rate.Limiter
	AmendMultipleOrders               *rate.Limiter
	CloseDeposit                      *rate.Limiter
	GetOrderDetails                   *rate.Limiter
	GetOrderList                      *rate.Limiter
	GetOrderHistory7Days              *rate.Limiter
	GetOrderHistory3Months            *rate.Limiter
	GetTransactionDetail3Days         *rate.Limiter
	GetTransactionDetail3Months       *rate.Limiter
	SetTransactionDetail2YearInterval *rate.Limiter
	GetTransactionDetailsLast2Year    *rate.Limiter
	CancelAllAfterCountdown           *rate.Limiter
	PlaceAlgoOrder                    *rate.Limiter
	CancelAlgoOrder                   *rate.Limiter
	AmendAlgoOrder                    *rate.Limiter
	CancelAdvanceAlgoOrder            *rate.Limiter
	GetAlgoOrderDetail                *rate.Limiter
	GetAlgoOrderList                  *rate.Limiter
	GetAlgoOrderHistory               *rate.Limiter
	GetEasyConvertCurrencyList        *rate.Limiter
	PlaceEasyConvert                  *rate.Limiter
	GetEasyConvertHistory             *rate.Limiter
	GetOneClickRepayHistory           *rate.Limiter
	OneClickRepayCurrencyList         *rate.Limiter
	TradeOneClickRepay                *rate.Limiter
	MassCancelMMPOrder                *rate.Limiter
	// Block Trading endpoints
	GetCounterparties           *rate.Limiter
	CreateRfq                   *rate.Limiter
	CancelRfq                   *rate.Limiter
	CancelMultipleRfqs          *rate.Limiter
	CancelAllRfqs               *rate.Limiter
	ExecuteQuote                *rate.Limiter
	GetQuoteProducts            *rate.Limiter
	SetQuoteProducts            *rate.Limiter
	ResetMMPStatus              *rate.Limiter
	ResetRFQMMPStatus           *rate.Limiter
	SetMMP                      *rate.Limiter
	GetMMPConfig                *rate.Limiter
	CreateQuote                 *rate.Limiter
	CancelQuote                 *rate.Limiter
	CancelMultipleQuotes        *rate.Limiter
	CancelAllQuotes             *rate.Limiter
	GetRfqs                     *rate.Limiter
	GetQuotes                   *rate.Limiter
	GetTrades                   *rate.Limiter
	GetTradesHistory            *rate.Limiter
	OptionInstrumentTradeFamily *rate.Limiter
	OptionTrades                *rate.Limiter
	GetPublicTrades             *rate.Limiter
	// Funding
	GetCurrencies              *rate.Limiter
	GetBalance                 *rate.Limiter
	GetNonTradableAssets       *rate.Limiter
	GetAccountAssetValuation   *rate.Limiter
	FundsTransfer              *rate.Limiter
	GetFundsTransferState      *rate.Limiter
	AssetBillsDetails          *rate.Limiter
	LightningDeposits          *rate.Limiter
	GetDepositAddress          *rate.Limiter
	GetDepositHistory          *rate.Limiter
	Withdrawal                 *rate.Limiter
	LightningWithdrawals       *rate.Limiter
	CancelWithdrawal           *rate.Limiter
	GetWithdrawalHistory       *rate.Limiter
	GetDepositWithdrawalStatus *rate.Limiter
	SmallAssetsConvert         *rate.Limiter
	GetPublicExchangeList      *rate.Limiter

	// Savings
	GetSavingBalance       *rate.Limiter
	SavingsPurchaseRedempt *rate.Limiter
	SetLendingRate         *rate.Limiter
	GetLendingHistory      *rate.Limiter
	GetPublicBorrowInfo    *rate.Limiter
	GetPublicBorrowHistory *rate.Limiter
	// Convert
	GetConvertCurrencies   *rate.Limiter
	GetConvertCurrencyPair *rate.Limiter
	EstimateQuote          *rate.Limiter
	ConvertTrade           *rate.Limiter
	GetConvertHistory      *rate.Limiter
	// Account
	GetAccountBalance                 *rate.Limiter
	GetPositions                      *rate.Limiter
	GetPositionsHistory               *rate.Limiter
	GetAccountAndPositionRisk         *rate.Limiter
	GetBillsDetails                   *rate.Limiter
	GetBillsDetailArchive             *rate.Limiter
	GetAccountConfiguration           *rate.Limiter
	SetPositionMode                   *rate.Limiter
	SetLeverage                       *rate.Limiter
	GetMaximumBuyOrSellAmount         *rate.Limiter
	GetMaximumAvailableTradableAmount *rate.Limiter
	IncreaseOrDecreaseMargin          *rate.Limiter
	GetLeverage                       *rate.Limiter
	GetLeverageEstimatedInfo          *rate.Limiter
	GetTheMaximumLoanOfInstrument     *rate.Limiter
	GetFeeRates                       *rate.Limiter
	GetInterestAccruedData            *rate.Limiter
	GetInterestRate                   *rate.Limiter
	SetGreeks                         *rate.Limiter
	IsolatedMarginTradingSettings     *rate.Limiter
	GetMaximumWithdrawals             *rate.Limiter
	GetAccountRiskState               *rate.Limiter
	ManualBorrowAndRepay              *rate.Limiter
	GetBorrowAndRepayHistory          *rate.Limiter
	VipLoansBorrowAnsRepay            *rate.Limiter
	GetBorrowAndRepayHistoryHistory   *rate.Limiter
	GetVIPInterestAccruedData         *rate.Limiter
	GetVIPInterestDeductedData        *rate.Limiter
	GetVIPLoanOrderList               *rate.Limiter
	GetVIPLoanOrderDetail             *rate.Limiter
	GetBorrowInterestAndLimit         *rate.Limiter
	PositionBuilder                   *rate.Limiter
	GetGreeks                         *rate.Limiter
	GetPMLimitation                   *rate.Limiter
	SetRiskOffsetType                 *rate.Limiter
	ActivateOption                    *rate.Limiter
	SetAutoLoan                       *rate.Limiter
	SetAccountLevel                   *rate.Limiter
	// Sub Account Endpoints
	ViewSubaccountList                             *rate.Limiter
	ResetSubAccountAPIKey                          *rate.Limiter
	GetSubaccountTradingBalance                    *rate.Limiter
	GetSubaccountFundingBalance                    *rate.Limiter
	SubAccountMaxWithdrawal                        *rate.Limiter
	HistoryOfSubaccountTransfer                    *rate.Limiter
	ManagedSubAccountTransfer                      *rate.Limiter
	MasterAccountsManageTransfersBetweenSubaccount *rate.Limiter
	SetPermissionOfTransferOut                     *rate.Limiter
	GetCustodyTradingSubaccountList                *rate.Limiter
	SetSubAccountVIPLoanAllocation                 *rate.Limiter
	GetSubAccountBorrowInterestAndLimit            *rate.Limiter
	GridTrading                                    *rate.Limiter
	AmendGridAlgoOrder                             *rate.Limiter
	StopGridAlgoOrder                              *rate.Limiter
	ClosePositionForForContractGrid                *rate.Limiter
	CancelClosePositionOrderForContractGrid        *rate.Limiter
	InstantTriggerGridAlgoOrder                    *rate.Limiter
	GetGridAlgoOrderList                           *rate.Limiter
	GetGridAlgoOrderHistory                        *rate.Limiter
	GetGridAlgoOrderDetails                        *rate.Limiter
	GetGridAlgoSubOrders                           *rate.Limiter
	GetGridAlgoOrderPositions                      *rate.Limiter
	SpotGridWithdrawIncome                         *rate.Limiter
	ComputeMarginBalance                           *rate.Limiter
	AdjustMarginBalance                            *rate.Limiter
	GetGridAIParameter                             *rate.Limiter
	ComputeMinInvestment                           *rate.Limiter
	RSIBackTesting                                 *rate.Limiter

	// Signal Bot Trading
	SignalBotOrderDetails *rate.Limiter
	SignalBotPosition     *rate.Limiter
	SignalBotSubOrders    *rate.Limiter
	SignalBotEventHistory *rate.Limiter

	// Recurring Buy Order
	PlaceRecurringBuyOrder           *rate.Limiter
	AmendRecurringBuyOrder           *rate.Limiter
	StopRecurringBuyOrder            *rate.Limiter
	GetRecurringBuyOrder             *rate.Limiter
	GetRecurringBuyOrderHistory      *rate.Limiter
	GetRecurringBuyOrderDetail       *rate.Limiter
	GetRecurringBuySubOrders         *rate.Limiter
	GetExistingLeadingPositions      *rate.Limiter
	GetLeadingPositionHistory        *rate.Limiter
	PlaceLeadingStopOrder            *rate.Limiter
	GetCloseLeadingPosition          *rate.Limiter
	GetLeadingInstruments            *rate.Limiter
	GetProfitSharingLimit            *rate.Limiter
	GetTotalProfitSharing            *rate.Limiter
	SetFirstCopySettings             *rate.Limiter
	AmendFirstCopySettings           *rate.Limiter
	StopCopying                      *rate.Limiter
	GetCopySettings                  *rate.Limiter
	GetMultipleLeverages             *rate.Limiter
	SetBatchLeverage                 *rate.Limiter
	GetMyLeadTraders                 *rate.Limiter
	GetLeadTraderRanks               *rate.Limiter
	GetLeadTraderWeeklyPNL           *rate.Limiter
	GetLeadTraderDailyPNL            *rate.Limiter
	GetLeadTraderStats               *rate.Limiter
	GetLeadTraderCurrencyPreferences *rate.Limiter
	GetTraderCurrentLeadPositions    *rate.Limiter
	GetLeadTraderLeadPositionHistory *rate.Limiter

	// Earn
	GetOffer                   *rate.Limiter
	Purchase                   *rate.Limiter
	Redeem                     *rate.Limiter
	CancelPurchaseOrRedemption *rate.Limiter
	GetEarnActiveOrders        *rate.Limiter
	GetFundingOrderHistory     *rate.Limiter

	// ETH Staking
	PurchaseETHStaking       *rate.Limiter
	RedeemETHStaking         *rate.Limiter
	GetBETHBalance           *rate.Limiter
	GetPurchaseRedeemHistory *rate.Limiter
	GetAPYHistory            *rate.Limiter

	// Market Data
	GetTickers                     *rate.Limiter
	GetTicker                      *rate.Limiter
	GetIndexTickers                *rate.Limiter
	GetOrderBook                   *rate.Limiter
	GetOrderBooksLite              *rate.Limiter
	GetCandlesticks                *rate.Limiter
	GetCandlesticksHistory         *rate.Limiter
	GetIndexCandlestickHistory     *rate.Limiter
	GetMarkPriceCandlestickHistory *rate.Limiter
	GetEconomicCalendar            *rate.Limiter
	GetIndexCandlesticks           *rate.Limiter
	GetMarkPriceCandlesticks       *rate.Limiter
	GetTradesRequest               *rate.Limiter
	Get24HTotalVolume              *rate.Limiter
	GetOracle                      *rate.Limiter
	GetExchangeRateRequest         *rate.Limiter
	GetIndexComponents             *rate.Limiter
	GetBlockTickers                *rate.Limiter
	GetBlockTrades                 *rate.Limiter

	// Spread Orders rate limiters
	PlaceSpreadOrder      *rate.Limiter
	CancelSpreadOrder     *rate.Limiter
	CancelAllSpreadOrder  *rate.Limiter
	AmendSpreadOrder      *rate.Limiter
	GetSpreadOrderDetails *rate.Limiter
	GetActiveSpreadOrders *rate.Limiter
	GetSpreadOrders7Days  *rate.Limiter
	GetSpreadOrderTrades  *rate.Limiter
	GetSpreads            *rate.Limiter
	GetSpreadOrderBook    *rate.Limiter
	GetSpreadTicker       *rate.Limiter
	GetSpreadPublicTrades *rate.Limiter

	// Public Data endpoints rate limiters
	GetInstruments                         *rate.Limiter
	GetDeliveryExerciseHistory             *rate.Limiter
	GetOpenInterest                        *rate.Limiter
	GetFundingRate                         *rate.Limiter
	GetFundingRateHistory                  *rate.Limiter
	GetLimitPrice                          *rate.Limiter
	GetOptionMarketDate                    *rate.Limiter
	GetEstimatedDeliveryExercisePrice      *rate.Limiter
	GetDiscountRateAndInterestFreeQuota    *rate.Limiter
	GetSystemTime                          *rate.Limiter
	GetLiquidationOrders                   *rate.Limiter
	GetMarkPrice                           *rate.Limiter
	GetPositionTiers                       *rate.Limiter
	GetInterestRateAndLoanQuota            *rate.Limiter
	GetInterestRateAndLoanQuoteForVIPLoans *rate.Limiter
	GetUnderlying                          *rate.Limiter
	GetInsuranceFund                       *rate.Limiter
	UnitConvert                            *rate.Limiter
	OptionTickBands                        *rate.Limiter
	GetIndexTicker                         *rate.Limiter
	// Trading Data Endpoints
	GetSupportCoin                    *rate.Limiter
	GetTakerVolume                    *rate.Limiter
	GetMarginLendingRatio             *rate.Limiter
	GetLongShortRatio                 *rate.Limiter
	GetContractsOpenInterestAndVolume *rate.Limiter
	GetOptionsOpenInterestAndVolume   *rate.Limiter
	GetPutCallRatio                   *rate.Limiter
	GetOpenInterestAndVolume          *rate.Limiter
	GetTakerFlow                      *rate.Limiter
	// Status Endpoints
	GetEventStatus                   *rate.Limiter
	GetAffilateInviteesDetail        *rate.Limiter
	GetUserAffilateRebateInformation *rate.Limiter
}

const (
	placeOrderEPL request.EndpointLimit = iota
	placeMultipleOrdersEPL
	cancelOrderEPL
	cancelMultipleOrdersEPL
	amendOrderEPL
	amendMultipleOrdersEPL
	closePositionEPL
	getOrderDetEPL
	getOrderListEPL
	getOrderHistory7DaysEPL
	getOrderHistory3MonthsEPL
	getTransactionDetail3DaysEPL
	getTransactionDetail3MonthsEPL
	setTransactionDetail2YearIntervalEPL
	getTransactionDetailLast2YearsEPL
	cancelAllAfterCountdownEPL
	placeAlgoOrderEPL
	cancelAlgoOrderEPL
	amendAlgoOrderEPL
	cancelAdvanceAlgoOrderEPL
	getAlgoOrderDetailEPL
	getAlgoOrderListEPL
	getAlgoOrderHistoryEPL
	getEasyConvertCurrencyListEPL
	placeEasyConvertEPL
	getEasyConvertHistoryEPL
	getOneClickRepayHistoryEPL
	oneClickRepayCurrencyListEPL
	tradeOneClickRepayEPL
	massCancemMMPOrderEPL
	getCounterpartiesEPL
	createRfqEPL
	cancelRfqEPL
	cancelMultipleRfqEPL
	cancelAllRfqsEPL
	executeQuoteEPL
	getQuoteProductsEPL
	setQuoteProductsEPL
	resetRFQMMPEPL
	setMMPEPL
	getMMPConfigEPL
	createQuoteEPL
	cancelQuoteEPL
	cancelMultipleQuotesEPL
	cancelAllQuotesEPL
	getRfqsEPL
	getQuotesEPL
	getTradesEPL
	getTradesHistoryEPL
	optionInstrumentTradeFamilyEPL
	optionTradesEPL
	getPublicTradesEPL
	getCurrenciesEPL
	getBalanceEPL
	getNonTradableAssetsEPL
	getAccountAssetValuationEPL
	fundsTransferEPL
	getFundsTransferStateEPL
	assetBillsDetailsEPL
	lightningDepositsEPL
	getDepositAddressEPL
	getDepositHistoryEPL
	withdrawalEPL
	lightningWithdrawalsEPL
	cancelWithdrawalEPL
	getWithdrawalHistoryEPL
	getDepositWithdrawalStatusEPL
	smallAssetsConvertEPL
	getPublicExchangeListEPL
	getSavingBalanceEPL
	savingsPurchaseRedemptionEPL
	setLendingRateEPL
	getLendingHistoryEPL
	getPublicBorrowInfoEPL
	getPublicBorrowHistoryEPL
	getConvertCurrenciesEPL
	getConvertCurrencyPairEPL
	estimateQuoteEPL
	convertTradeEPL
	getConvertHistoryEPL
	getAccountBalanceEPL
	getPositionsEPL
	getPositionsHistoryEPL
	getAccountAndPositionRiskEPL
	getBillsDetailsEPL
	getBillsDetailArchiveEPL
	getAccountConfigurationEPL
	setPositionModeEPL
	setLeverageEPL
	getMaximumBuyOrSellAmountEPL
	getMaximumAvailableTradableAmountEPL
	increaseOrDecreaseMarginEPL
	getLeverageEPL
	getLeverateEstimatedInfoEPL
	getTheMaximumLoanOfInstrumentEPL
	getFeeRatesEPL
	getInterestAccruedDataEPL
	getInterestRateEPL
	setGreeksEPL
	isolatedMarginTradingSettingsEPL
	getMaximumWithdrawalsEPL
	getAccountRiskStateEPL
	manualBorrowAndRepayEPL
	getBorrowAndRepayHistoryEPL
	vipLoansBorrowAnsRepayEPL
	getBorrowAnsRepayHistoryHistoryEPL
	getVIPInterestAccruedDataEPL
	getVIPInterestDeductedDataEPL
	getVIPLoanOrderListEPL
	getVIPLoanOrderDetailEPL
	getBorrowInterestAndLimitEPL
	positionBuilderEPL
	getGreeksEPL
	getPMLimitationEPL
	setRiskOffsetLimiterEPL
	activateOptionEPL
	setAutoLoanEPL
	setAccountLevelEPL
	resetMMPStatusEPL
	viewSubaccountListEPL
	resetSubAccountAPIKeyEPL
	getSubaccountTradingBalanceEPL
	getSubaccountFundingBalanceEPL
	getSubAccountMaxWithdrawalEPL
	historyOfSubaccountTransferEPL
	managedSubAccountTransferEPL
	masterAccountsManageTransfersBetweenSubaccountEPL
	setPermissionOfTransferOutEPL
	getCustodyTradingSubaccountListEPL
	setSubAccountVIPLoanAllocationEPL
	getSubAccountBorrowInterestAndLimitEPL
	gridTradingEPL
	amendGridAlgoOrderEPL
	stopGridAlgoOrderEPL
	closePositionForForContractGridEPL
	cancelClosePositionOrderForContractGridEPL
	instantTriggerGridAlgoOrderEPL
	getGridAlgoOrderListEPL
	getGridAlgoOrderHistoryEPL
	getGridAlgoOrderDetailsEPL
	getGridAlgoSubOrdersEPL
	getGridAlgoOrderPositionsEPL
	spotGridWithdrawIncomeEPL
	computeMarginBalanceEPL
	adjustMarginBalanceEPL
	getGridAIParameterEPL
	computeMinInvestmentEPL
	rsiBackTestingEPL
	signalBotOrderDetailsEPL
	signalBotOrderPositionsEPL
	signalBotSubOrdersEPL
	signalBotEventHistoryEPL
	placeRecurringBuyOrderEPL
	amendRecurringBuyOrderEPL
	stopRecurringBuyOrderEPL
	getRecurringBuyOrderListEPL
	getRecurringBuyOrderHistoryEPL
	getRecurringBuyOrderDetailEPL
	getRecurringBuySubOrdersEPL
	getExistingLeadingPositionsEPL
	getLeadingPositionHistoryEPL
	placeLeadingStopOrderEPL
	closeLeadingPositionEPL
	getLeadingInstrumentsEPL
	getProfitSharingLimitEPL
	getTotalProfitSharingEPL
	setFirstCopySettingsEPL
	amendFirstCopySettingsEPL
	stopCopyingEPL
	getCopySettingsEPL
	getMultipleLeveragesEPL
	setBatchLeverageEPL
	getMyLeadTradersEPL
	getLeadTraderRanksEPL
	getLeadTraderWeeklyPNLEPL
	getLeadTraderDailyPNLEPL
	getLeadTraderStatsEPL
	getLeadTraderCurrencyPreferencesEPL
	getTraderCurrentLeadPositionsEPL
	getLeadTraderLeadPositionHistoryEPL
	getOfferEPL
	purchaseEPL
	redeemEPL
	cancelPurchaseOrRedemptionEPL
	getEarnActiveOrdersEPL
	getFundingOrderHistoryEPL

	purchaseETHStakingEPL
	redeemETHStakingEPL
	getBETHBalanceEPL
	getPurchaseRedeemHistoryEPL
	getAPYHistoryEPL

	getTickersEPL
	getTickerEPL
	getIndexTickersEPL
	getOrderBookEPL
	getOrderBookLiteEPL
	getCandlesticksEPL
	getTradesRequestEPL
	get24HTotalVolumeEPL
	getOracleEPL
	getExchangeRateRequestEPL
	getIndexComponentsEPL
	getBlockTickersEPL
	getBlockTradesEPL
	placeSpreadOrderEPL
	cancelSpreadOrderEPL
	cancelAllSpreadOrderEPL
	amendSpreadOrderEPL
	getSpreadOrderDetailsEPL
	getSpreadOrderTradesEPL
	getSpreadsEPL
	getSpreadOrderbookEPL
	getSpreadTickerEPL
	getSpreadPublicTradesEPL
	getActiveSpreadOrdersEPL
	getSpreadOrders7DaysEPL
	getInstrumentsEPL
	getDeliveryExerciseHistoryEPL
	getOpenInterestEPL
	getFundingEPL
	getFundingRateHistoryEPL
	getLimitPriceEPL
	getOptionMarketDateEPL
	getEstimatedDeliveryPriceEPL
	getDiscountRateAndInterestFreeQuotaEPL
	getSystemTimeEPL
	getLiquidationOrdersEPL
	getMarkPriceEPL
	getPositionTiersEPL
	getInterestRateAndLoanQuotaEPL
	getInterestRateAndLoanQuoteForVIPLoansEPL
	getUnderlyingEPL
	getInsuranceFundEPL
	unitConvertEPL
	optionTickBandsEPL
	getIndexTickerEPL
	getSupportCoinEPL
	getTakerVolumeEPL
	getMarginLendingRatioEPL
	getLongShortRatioEPL
	getContractsOpenInterestAndVolumeEPL
	getOptionsOpenInterestAndVolumeEPL
	getPutCallRatioEPL
	getOpenInterestAndVolumeEPL
	getTakerFlowEPL
	getEventStatusEPL
	getCandlestickHistoryEPL
	getIndexCandlesticksEPL
	getIndexCandlesticksHistoryEPL
	getMarkPriceCandlesticksHistoryEPL
	getEconomicCalendarEPL

	getAffilateInviteesDetailEPL
	getUserAffilateRebateInformationEPL
)

// Limit executes rate limiting for Okx exchange given the context and EndpointLimit
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case placeOrderEPL:
		return r.PlaceOrder.Wait(ctx)
	case placeMultipleOrdersEPL:
		return r.PlaceMultipleOrders.Wait(ctx)
	case cancelOrderEPL:
		return r.CancelOrder.Wait(ctx)
	case cancelMultipleOrdersEPL:
		return r.CancelMultipleOrders.Wait(ctx)
	case amendOrderEPL:
		return r.AmendOrder.Wait(ctx)
	case amendMultipleOrdersEPL:
		return r.AmendMultipleOrders.Wait(ctx)
	case closePositionEPL:
		return r.CloseDeposit.Wait(ctx)
	case getOrderDetEPL:
		return r.GetOrderDetails.Wait(ctx)
	case getOrderListEPL:
		return r.GetOrderList.Wait(ctx)
	case getOrderHistory7DaysEPL:
		return r.GetOrderHistory7Days.Wait(ctx)
	case getOrderHistory3MonthsEPL:
		return r.GetOrderHistory3Months.Wait(ctx)
	case getTransactionDetail3DaysEPL:
		return r.GetTransactionDetail3Days.Wait(ctx)
	case getTransactionDetail3MonthsEPL:
		return r.GetTransactionDetail3Months.Wait(ctx)
	case setTransactionDetail2YearIntervalEPL:
		return r.SetTransactionDetail2YearInterval.Wait(ctx)
	case getTransactionDetailLast2YearsEPL:
		return r.GetTransactionDetailsLast2Year.Wait(ctx)
	case cancelAllAfterCountdownEPL:
		return r.CancelAllAfterCountdown.Wait(ctx)
	case placeAlgoOrderEPL:
		return r.PlaceAlgoOrder.Wait(ctx)
	case cancelAlgoOrderEPL:
		return r.CancelAlgoOrder.Wait(ctx)
	case amendAlgoOrderEPL:
		return r.AmendAlgoOrder.Wait(ctx)
	case cancelAdvanceAlgoOrderEPL:
		return r.CancelAdvanceAlgoOrder.Wait(ctx)
	case getAlgoOrderDetailEPL:
		return r.GetAlgoOrderDetail.Wait(ctx)
	case getAlgoOrderListEPL:
		return r.GetAlgoOrderList.Wait(ctx)
	case getAlgoOrderHistoryEPL:
		return r.GetAlgoOrderHistory.Wait(ctx)
	case getEasyConvertCurrencyListEPL:
		return r.GetEasyConvertCurrencyList.Wait(ctx)
	case placeEasyConvertEPL:
		return r.PlaceEasyConvert.Wait(ctx)
	case getEasyConvertHistoryEPL:
		return r.GetEasyConvertHistory.Wait(ctx)
	case getOneClickRepayHistoryEPL:
		return r.GetOneClickRepayHistory.Wait(ctx)
	case oneClickRepayCurrencyListEPL:
		return r.OneClickRepayCurrencyList.Wait(ctx)
	case tradeOneClickRepayEPL:
		return r.TradeOneClickRepay.Wait(ctx)
	case massCancemMMPOrderEPL:
		return r.MassCancelMMPOrder.Wait(ctx)
	case getCounterpartiesEPL:
		return r.GetCounterparties.Wait(ctx)
	case createRfqEPL:
		return r.CreateRfq.Wait(ctx)
	case cancelRfqEPL:
		return r.CancelRfq.Wait(ctx)
	case cancelMultipleRfqEPL:
		return r.CancelMultipleRfqs.Wait(ctx)
	case cancelAllRfqsEPL:
		return r.CancelAllRfqs.Wait(ctx)
	case executeQuoteEPL:
		return r.ExecuteQuote.Wait(ctx)
	case getQuoteProductsEPL:
		return r.GetQuoteProducts.Wait(ctx)
	case setQuoteProductsEPL:
		return r.SetQuoteProducts.Wait(ctx)
	case resetMMPStatusEPL:
		return r.ResetMMPStatus.Wait(ctx)
	case resetRFQMMPEPL:
		return r.ResetRFQMMPStatus.Wait(ctx)
	case setMMPEPL:
		return r.SetMMP.Wait(ctx)
	case getMMPConfigEPL:
		return r.GetMMPConfig.Wait(ctx)
	case createQuoteEPL:
		return r.CreateQuote.Wait(ctx)
	case cancelQuoteEPL:
		return r.CancelQuote.Wait(ctx)
	case cancelMultipleQuotesEPL:
		return r.CancelMultipleQuotes.Wait(ctx)
	case cancelAllQuotesEPL:
		return r.CancelAllQuotes.Wait(ctx)
	case getRfqsEPL:
		return r.GetRfqs.Wait(ctx)
	case getQuotesEPL:
		return r.GetQuotes.Wait(ctx)
	case getTradesEPL:
		return r.GetTrades.Wait(ctx)
	case getTradesHistoryEPL:
		return r.GetTradesHistory.Wait(ctx)
	case optionInstrumentTradeFamilyEPL:
		return r.OptionInstrumentTradeFamily.Wait(ctx)
	case optionTradesEPL:
		return r.OptionTrades.Wait(ctx)
	case getPublicTradesEPL:
		return r.GetPublicTrades.Wait(ctx)
	case getCurrenciesEPL:
		return r.GetCurrencies.Wait(ctx)
	case getBalanceEPL:
		return r.GetBalance.Wait(ctx)
	case getNonTradableAssetsEPL:
		return r.GetNonTradableAssets.Wait(ctx)
	case getAccountAssetValuationEPL:
		return r.GetAccountAssetValuation.Wait(ctx)
	case fundsTransferEPL:
		return r.FundsTransfer.Wait(ctx)
	case getFundsTransferStateEPL:
		return r.GetFundsTransferState.Wait(ctx)
	case assetBillsDetailsEPL:
		return r.AssetBillsDetails.Wait(ctx)
	case lightningDepositsEPL:
		return r.LightningDeposits.Wait(ctx)
	case getDepositAddressEPL:
		return r.GetDepositAddress.Wait(ctx)
	case getDepositHistoryEPL:
		return r.GetDepositHistory.Wait(ctx)
	case withdrawalEPL:
		return r.Withdrawal.Wait(ctx)
	case lightningWithdrawalsEPL:
		return r.LightningWithdrawals.Wait(ctx)
	case cancelWithdrawalEPL:
		return r.CancelWithdrawal.Wait(ctx)
	case getWithdrawalHistoryEPL:
		return r.GetWithdrawalHistory.Wait(ctx)
	case getDepositWithdrawalStatusEPL:
		return r.GetDepositWithdrawalStatus.Wait(ctx)
	case smallAssetsConvertEPL:
		return r.SmallAssetsConvert.Wait(ctx)
	case getPublicExchangeListEPL:
		return r.GetPublicExchangeList.Wait(ctx)
	case getSavingBalanceEPL:
		return r.GetSavingBalance.Wait(ctx)
	case savingsPurchaseRedemptionEPL:
		return r.SavingsPurchaseRedempt.Wait(ctx)
	case setLendingRateEPL:
		return r.SetLendingRate.Wait(ctx)
	case getLendingHistoryEPL:
		return r.GetLendingHistory.Wait(ctx)
	case getPublicBorrowInfoEPL:
		return r.GetPublicBorrowInfo.Wait(ctx)
	case getPublicBorrowHistoryEPL:
		return r.GetPublicBorrowHistory.Wait(ctx)
	case getConvertCurrenciesEPL:
		return r.GetConvertCurrencies.Wait(ctx)
	case getConvertCurrencyPairEPL:
		return r.GetConvertCurrencyPair.Wait(ctx)
	case estimateQuoteEPL:
		return r.EstimateQuote.Wait(ctx)
	case convertTradeEPL:
		return r.ConvertTrade.Wait(ctx)
	case getConvertHistoryEPL:
		return r.GetConvertHistory.Wait(ctx)
	case getAccountBalanceEPL:
		return r.GetAccountBalance.Wait(ctx)
	case getPositionsEPL:
		return r.GetPositions.Wait(ctx)
	case getPositionsHistoryEPL:
		return r.GetPositionsHistory.Wait(ctx)
	case getAccountAndPositionRiskEPL:
		return r.GetAccountAndPositionRisk.Wait(ctx)
	case getBillsDetailsEPL:
		return r.GetBillsDetails.Wait(ctx)
	case getBillsDetailArchiveEPL:
		return r.GetBillsDetailArchive.Wait(ctx)
	case getAccountConfigurationEPL:
		return r.GetAccountConfiguration.Wait(ctx)
	case setPositionModeEPL:
		return r.SetPositionMode.Wait(ctx)
	case setLeverageEPL:
		return r.SetLeverage.Wait(ctx)
	case getMaximumBuyOrSellAmountEPL:
		return r.GetMaximumBuyOrSellAmount.Wait(ctx)
	case getMaximumAvailableTradableAmountEPL:
		return r.GetMaximumAvailableTradableAmount.Wait(ctx)
	case increaseOrDecreaseMarginEPL:
		return r.IncreaseOrDecreaseMargin.Wait(ctx)
	case getLeverageEPL:
		return r.GetLeverage.Wait(ctx)
	case getLeverateEstimatedInfoEPL:
		return r.GetLeverageEstimatedInfo.Wait(ctx)
	case getTheMaximumLoanOfInstrumentEPL:
		return r.GetTheMaximumLoanOfInstrument.Wait(ctx)
	case getFeeRatesEPL:
		return r.GetFeeRates.Wait(ctx)
	case getInterestAccruedDataEPL:
		return r.GetInterestAccruedData.Wait(ctx)
	case getInterestRateEPL:
		return r.GetInterestRate.Wait(ctx)
	case setGreeksEPL:
		return r.SetGreeks.Wait(ctx)
	case isolatedMarginTradingSettingsEPL:
		return r.IsolatedMarginTradingSettings.Wait(ctx)
	case getMaximumWithdrawalsEPL:
		return r.GetMaximumWithdrawals.Wait(ctx)
	case getAccountRiskStateEPL:
		return r.GetAccountRiskState.Wait(ctx)
	case manualBorrowAndRepayEPL:
		return r.ManualBorrowAndRepay.Wait(ctx)
	case getBorrowAndRepayHistoryEPL:
		return r.GetBorrowAndRepayHistory.Wait(ctx)
	case vipLoansBorrowAnsRepayEPL:
		return r.VipLoansBorrowAnsRepay.Wait(ctx)
	case getBorrowAnsRepayHistoryHistoryEPL:
		return r.GetBorrowAndRepayHistoryHistory.Wait(ctx)
	case getVIPInterestAccruedDataEPL:
		return r.GetVIPInterestAccruedData.Wait(ctx)
	case getVIPInterestDeductedDataEPL:
		return r.GetVIPInterestDeductedData.Wait(ctx)
	case getVIPLoanOrderListEPL:
		return r.GetVIPLoanOrderList.Wait(ctx)
	case getVIPLoanOrderDetailEPL:
		return r.GetVIPLoanOrderDetail.Wait(ctx)
	case getBorrowInterestAndLimitEPL:
		return r.GetBorrowInterestAndLimit.Wait(ctx)
	case positionBuilderEPL:
		return r.PositionBuilder.Wait(ctx)
	case getGreeksEPL:
		return r.GetGreeks.Wait(ctx)
	case getPMLimitationEPL:
		return r.GetPMLimitation.Wait(ctx)
	case setRiskOffsetLimiterEPL:
		return r.SetRiskOffsetType.Wait(ctx)
	case activateOptionEPL:
		return r.ActivateOption.Wait(ctx)
	case setAutoLoanEPL:
		return r.SetAutoLoan.Wait(ctx)
	case setAccountLevelEPL:
		return r.SetAccountLevel.Wait(ctx)
	case viewSubaccountListEPL:
		return r.ViewSubaccountList.Wait(ctx)
	case resetSubAccountAPIKeyEPL:
		return r.ResetSubAccountAPIKey.Wait(ctx)
	case getSubaccountTradingBalanceEPL:
		return r.GetSubaccountTradingBalance.Wait(ctx)
	case getSubaccountFundingBalanceEPL:
		return r.GetSubaccountFundingBalance.Wait(ctx)
	case getSubAccountMaxWithdrawalEPL:
		return r.SubAccountMaxWithdrawal.Wait(ctx)
	case historyOfSubaccountTransferEPL:
		return r.HistoryOfSubaccountTransfer.Wait(ctx)
	case managedSubAccountTransferEPL:
		return r.ManagedSubAccountTransfer.Wait(ctx)
	case masterAccountsManageTransfersBetweenSubaccountEPL:
		return r.MasterAccountsManageTransfersBetweenSubaccount.Wait(ctx)
	case setPermissionOfTransferOutEPL:
		return r.SetPermissionOfTransferOut.Wait(ctx)
	case getCustodyTradingSubaccountListEPL:
		return r.GetCustodyTradingSubaccountList.Wait(ctx)
	case setSubAccountVIPLoanAllocationEPL:
		return r.SetSubAccountVIPLoanAllocation.Wait(ctx)
	case getSubAccountBorrowInterestAndLimitEPL:
		return r.GetSubAccountBorrowInterestAndLimit.Wait(ctx)
	case gridTradingEPL:
		return r.GridTrading.Wait(ctx)
	case amendGridAlgoOrderEPL:
		return r.AmendGridAlgoOrder.Wait(ctx)
	case stopGridAlgoOrderEPL:
		return r.StopGridAlgoOrder.Wait(ctx)
	case closePositionForForContractGridEPL:
		return r.ClosePositionForForContractGrid.Wait(ctx)
	case cancelClosePositionOrderForContractGridEPL:
		return r.CancelClosePositionOrderForContractGrid.Wait(ctx)
	case instantTriggerGridAlgoOrderEPL:
		return r.InstantTriggerGridAlgoOrder.Wait(ctx)
	case getGridAlgoOrderListEPL:
		return r.GetGridAlgoOrderList.Wait(ctx)
	case getGridAlgoOrderHistoryEPL:
		return r.GetGridAlgoOrderHistory.Wait(ctx)
	case getGridAlgoOrderDetailsEPL:
		return r.GetGridAlgoOrderDetails.Wait(ctx)
	case getGridAlgoSubOrdersEPL:
		return r.GetGridAlgoSubOrders.Wait(ctx)
	case getGridAlgoOrderPositionsEPL:
		return r.GetGridAlgoOrderPositions.Wait(ctx)
	case spotGridWithdrawIncomeEPL:
		return r.SpotGridWithdrawIncome.Wait(ctx)
	case computeMarginBalanceEPL:
		return r.ComputeMarginBalance.Wait(ctx)
	case adjustMarginBalanceEPL:
		return r.AdjustMarginBalance.Wait(ctx)
	case getGridAIParameterEPL:
		return r.GetGridAIParameter.Wait(ctx)
	case computeMinInvestmentEPL:
		return r.ComputeMinInvestment.Wait(ctx)
	case rsiBackTestingEPL:
		return r.RSIBackTesting.Wait(ctx)

	case signalBotOrderDetailsEPL:
		return r.SignalBotOrderDetails.Wait(ctx)
	case signalBotOrderPositionsEPL:
		return r.SignalBotPosition.Wait(ctx)
	case signalBotSubOrdersEPL:
		return r.SignalBotSubOrders.Wait(ctx)
	case signalBotEventHistoryEPL:
		return r.SignalBotEventHistory.Wait(ctx)
	case placeRecurringBuyOrderEPL:
		return r.PlaceRecurringBuyOrder.Wait(ctx)
	case amendRecurringBuyOrderEPL:
		return r.AmendRecurringBuyOrder.Wait(ctx)
	case stopRecurringBuyOrderEPL:
		return r.StopRecurringBuyOrder.Wait(ctx)
	case getRecurringBuyOrderListEPL:
		return r.GetRecurringBuyOrder.Wait(ctx)
	case getRecurringBuyOrderHistoryEPL:
		return r.GetRecurringBuyOrderHistory.Wait(ctx)
	case getRecurringBuyOrderDetailEPL:
		return r.GetRecurringBuyOrderDetail.Wait(ctx)
	case getRecurringBuySubOrdersEPL:
		return r.GetRecurringBuySubOrders.Wait(ctx)
	case getExistingLeadingPositionsEPL:
		return r.GetExistingLeadingPositions.Wait(ctx)
	case getLeadingPositionHistoryEPL:
		return r.GetLeadingPositionHistory.Wait(ctx)
	case placeLeadingStopOrderEPL:
		return r.PlaceLeadingStopOrder.Wait(ctx)
	case closeLeadingPositionEPL:
		return r.GetCloseLeadingPosition.Wait(ctx)
	case getLeadingInstrumentsEPL:
		return r.GetLeadingInstruments.Wait(ctx)
	case getProfitSharingLimitEPL:
		return r.GetProfitSharingLimit.Wait(ctx)
	case getTotalProfitSharingEPL:
		return r.GetTotalProfitSharing.Wait(ctx)
	case setFirstCopySettingsEPL:
		return r.SetFirstCopySettings.Wait(ctx)
	case amendFirstCopySettingsEPL:
		return r.AmendFirstCopySettings.Wait(ctx)
	case stopCopyingEPL:
		return r.StopCopying.Wait(ctx)
	case getCopySettingsEPL:
		return r.GetCopySettings.Wait(ctx)
	case getMultipleLeveragesEPL:
		return r.GetMultipleLeverages.Wait(ctx)
	case setBatchLeverageEPL:
		return r.SetBatchLeverage.Wait(ctx)
	case getMyLeadTradersEPL:
		return r.GetMyLeadTraders.Wait(ctx)
	case getLeadTraderRanksEPL:
		return r.GetLeadTraderRanks.Wait(ctx)
	case getLeadTraderWeeklyPNLEPL:
		return r.GetLeadTraderWeeklyPNL.Wait(ctx)
	case getLeadTraderDailyPNLEPL:
		return r.GetLeadTraderDailyPNL.Wait(ctx)
	case getLeadTraderStatsEPL:
		return r.GetLeadTraderStats.Wait(ctx)
	case getLeadTraderCurrencyPreferencesEPL:
		return r.GetLeadTraderCurrencyPreferences.Wait(ctx)
	case getTraderCurrentLeadPositionsEPL:
		return r.GetTraderCurrentLeadPositions.Wait(ctx)
	case getLeadTraderLeadPositionHistoryEPL:
		return r.GetLeadTraderLeadPositionHistory.Wait(ctx)
	case getOfferEPL:
		return r.GetOffer.Wait(ctx)
	case purchaseEPL:
		return r.Purchase.Wait(ctx)
	case redeemEPL:
		return r.Redeem.Wait(ctx)
	case cancelPurchaseOrRedemptionEPL:
		return r.CancelPurchaseOrRedemption.Wait(ctx)
	case getEarnActiveOrdersEPL:
		return r.GetEarnActiveOrders.Wait(ctx)
	case getFundingOrderHistoryEPL:
		return r.GetFundingOrderHistory.Wait(ctx)

	case purchaseETHStakingEPL:
		return r.PurchaseETHStaking.Wait(ctx)
	case redeemETHStakingEPL:
		return r.RedeemETHStaking.Wait(ctx)
	case getBETHBalanceEPL:
		return r.GetBETHBalance.Wait(ctx)
	case getPurchaseRedeemHistoryEPL:
		return r.GetPurchaseRedeemHistory.Wait(ctx)
	case getAPYHistoryEPL:
		return r.GetAPYHistory.Wait(ctx)

	case getTickersEPL:
		return r.GetTickers.Wait(ctx)
	case getTickerEPL:
		return r.GetTicker.Wait(ctx)
	case getIndexTickersEPL:
		return r.GetIndexTickers.Wait(ctx)
	case getOrderBookEPL:
		return r.GetOrderBook.Wait(ctx)
	case getOrderBookLiteEPL:
		return r.GetOrderBooksLite.Wait(ctx)
	case getCandlesticksEPL:
		return r.GetCandlesticks.Wait(ctx)
	case getCandlestickHistoryEPL:
		return r.GetCandlesticksHistory.Wait(ctx)
	case getIndexCandlesticksHistoryEPL:
		return r.GetIndexCandlestickHistory.Wait(ctx)
	case getMarkPriceCandlesticksHistoryEPL:
		return r.GetMarkPriceCandlestickHistory.Wait(ctx)
	case getEconomicCalendarEPL:
		return r.GetEconomicCalendar.Wait(ctx)
	case getIndexCandlesticksEPL:
		return r.GetIndexCandlesticks.Wait(ctx)
	case getTradesRequestEPL:
		return r.GetTradesRequest.Wait(ctx)
	case get24HTotalVolumeEPL:
		return r.Get24HTotalVolume.Wait(ctx)
	case getOracleEPL:
		return r.GetOracle.Wait(ctx)
	case getExchangeRateRequestEPL:
		return r.GetExchangeRateRequest.Wait(ctx)
	case getIndexComponentsEPL:
		return r.GetIndexComponents.Wait(ctx)
	case getBlockTickersEPL:
		return r.GetBlockTickers.Wait(ctx)
	case getBlockTradesEPL:
		return r.GetBlockTrades.Wait(ctx)
	case placeSpreadOrderEPL:
		return r.PlaceSpreadOrder.Wait(ctx)
	case cancelSpreadOrderEPL:
		return r.CancelSpreadOrder.Wait(ctx)
	case cancelAllSpreadOrderEPL:
		return r.CancelAllSpreadOrder.Wait(ctx)
	case amendSpreadOrderEPL:
		return r.AmendSpreadOrder.Wait(ctx)
	case getSpreadOrderDetailsEPL:
		return r.GetSpreadOrderDetails.Wait(ctx)
	case getSpreadOrderTradesEPL:
		return r.GetSpreadOrderTrades.Wait(ctx)
	case getSpreadsEPL:
		return r.GetSpreads.Wait(ctx)
	case getSpreadOrderbookEPL:
		return r.GetSpreadOrderBook.Wait(ctx)
	case getSpreadTickerEPL:
		return r.GetSpreadTicker.Wait(ctx)
	case getSpreadPublicTradesEPL:
		return r.GetSpreadPublicTrades.Wait(ctx)
	case getActiveSpreadOrdersEPL:
		return r.GetActiveSpreadOrders.Wait(ctx)
	case getSpreadOrders7DaysEPL:
		return r.GetSpreadOrders7Days.Wait(ctx)
	case getInstrumentsEPL:
		return r.GetInstruments.Wait(ctx)
	case getDeliveryExerciseHistoryEPL:
		return r.GetDeliveryExerciseHistory.Wait(ctx)
	case getOpenInterestEPL:
		return r.GetOpenInterest.Wait(ctx)
	case getFundingEPL:
		return r.GetFundingRate.Wait(ctx)
	case getFundingRateHistoryEPL:
		return r.GetFundingRateHistory.Wait(ctx)
	case getLimitPriceEPL:
		return r.GetLimitPrice.Wait(ctx)
	case getOptionMarketDateEPL:
		return r.GetOptionMarketDate.Wait(ctx)
	case getEstimatedDeliveryPriceEPL:
		return r.GetEstimatedDeliveryExercisePrice.Wait(ctx)
	case getDiscountRateAndInterestFreeQuotaEPL:
		return r.GetDiscountRateAndInterestFreeQuota.Wait(ctx)
	case getSystemTimeEPL:
		return r.GetSystemTime.Wait(ctx)
	case getLiquidationOrdersEPL:
		return r.GetLiquidationOrders.Wait(ctx)
	case getMarkPriceEPL:
		return r.GetMarkPrice.Wait(ctx)
	case getPositionTiersEPL:
		return r.GetPositionTiers.Wait(ctx)
	case getInterestRateAndLoanQuotaEPL:
		return r.GetInterestRateAndLoanQuota.Wait(ctx)
	case getInterestRateAndLoanQuoteForVIPLoansEPL:
		return r.GetInterestRateAndLoanQuoteForVIPLoans.Wait(ctx)
	case getUnderlyingEPL:
		return r.GetUnderlying.Wait(ctx)
	case getInsuranceFundEPL:
		return r.GetInsuranceFund.Wait(ctx)
	case unitConvertEPL:
		return r.UnitConvert.Wait(ctx)
	case optionTickBandsEPL:
		return r.OptionTickBands.Wait(ctx)
	case getIndexTickerEPL:
		return r.GetIndexTicker.Wait(ctx)
	case getSupportCoinEPL:
		return r.GetSupportCoin.Wait(ctx)
	case getTakerVolumeEPL:
		return r.GetTakerVolume.Wait(ctx)
	case getMarginLendingRatioEPL:
		return r.GetMarginLendingRatio.Wait(ctx)
	case getLongShortRatioEPL:
		return r.GetLongShortRatio.Wait(ctx)
	case getContractsOpenInterestAndVolumeEPL:
		return r.GetContractsOpenInterestAndVolume.Wait(ctx)
	case getOptionsOpenInterestAndVolumeEPL:
		return r.GetOptionsOpenInterestAndVolume.Wait(ctx)
	case getPutCallRatioEPL:
		return r.GetPutCallRatio.Wait(ctx)
	case getOpenInterestAndVolumeEPL:
		return r.GetOpenInterestAndVolume.Wait(ctx)
	case getTakerFlowEPL:
		return r.GetTakerFlow.Wait(ctx)
	case getEventStatusEPL:
		return r.GetEventStatus.Wait(ctx)
	case getAffilateInviteesDetailEPL:
		return r.GetAffilateInviteesDetail.Wait(ctx)
	case getUserAffilateRebateInformationEPL:
		return r.GetUserAffilateRebateInformation.Wait(ctx)
	default:
		return errors.New("endpoint rate limit functionality not found")
	}
}

// SetRateLimit returns a RateLimit instance, which implements the request.Limiter interface.
func SetRateLimit() *RateLimit {
	return &RateLimit{
		// Trade Endpoints
		PlaceOrder:                        request.NewRateLimit(twoSecondsInterval, 60),
		PlaceMultipleOrders:               request.NewRateLimit(twoSecondsInterval, 4),
		CancelOrder:                       request.NewRateLimit(twoSecondsInterval, 60),
		CancelMultipleOrders:              request.NewRateLimit(twoSecondsInterval, 300),
		AmendOrder:                        request.NewRateLimit(twoSecondsInterval, 60),
		AmendMultipleOrders:               request.NewRateLimit(twoSecondsInterval, 4),
		CloseDeposit:                      request.NewRateLimit(twoSecondsInterval, 20),
		GetOrderDetails:                   request.NewRateLimit(twoSecondsInterval, 60),
		GetOrderList:                      request.NewRateLimit(twoSecondsInterval, 60),
		GetOrderHistory7Days:              request.NewRateLimit(twoSecondsInterval, 40),
		GetOrderHistory3Months:            request.NewRateLimit(twoSecondsInterval, 20),
		GetTransactionDetail3Days:         request.NewRateLimit(twoSecondsInterval, 60),
		GetTransactionDetail3Months:       request.NewRateLimit(twoSecondsInterval, 10),
		SetTransactionDetail2YearInterval: request.NewRateLimit(time.Hour*24, 5),
		GetTransactionDetailsLast2Year:    request.NewRateLimit(twoSecondsInterval, 10),
		CancelAllAfterCountdown:           request.NewRateLimit(oneSecondInterval, 1),
		PlaceAlgoOrder:                    request.NewRateLimit(twoSecondsInterval, 20),
		CancelAlgoOrder:                   request.NewRateLimit(twoSecondsInterval, 20),
		AmendAlgoOrder:                    request.NewRateLimit(twoSecondsInterval, 20),
		CancelAdvanceAlgoOrder:            request.NewRateLimit(twoSecondsInterval, 20),
		GetAlgoOrderDetail:                request.NewRateLimit(twoSecondsInterval, 20),
		GetAlgoOrderList:                  request.NewRateLimit(twoSecondsInterval, 20),
		GetAlgoOrderHistory:               request.NewRateLimit(twoSecondsInterval, 20),
		GetEasyConvertCurrencyList:        request.NewRateLimit(twoSecondsInterval, 1),
		PlaceEasyConvert:                  request.NewRateLimit(twoSecondsInterval, 1),
		GetEasyConvertHistory:             request.NewRateLimit(twoSecondsInterval, 1),
		GetOneClickRepayHistory:           request.NewRateLimit(twoSecondsInterval, 1),
		OneClickRepayCurrencyList:         request.NewRateLimit(twoSecondsInterval, 1),
		TradeOneClickRepay:                request.NewRateLimit(twoSecondsInterval, 1),
		MassCancelMMPOrder:                request.NewRateLimit(twoSecondsInterval, 5),
		// Block Trading endpoints
		GetCounterparties:           request.NewRateLimit(twoSecondsInterval, 5),
		CreateRfq:                   request.NewRateLimit(twoSecondsInterval, 5),
		CancelRfq:                   request.NewRateLimit(twoSecondsInterval, 5),
		CancelMultipleRfqs:          request.NewRateLimit(twoSecondsInterval, 2),
		CancelAllRfqs:               request.NewRateLimit(twoSecondsInterval, 2),
		ExecuteQuote:                request.NewRateLimit(threeSecondsInterval, 2),
		GetQuoteProducts:            request.NewRateLimit(twoSecondsInterval, 5),
		SetQuoteProducts:            request.NewRateLimit(twoSecondsInterval, 5),
		ResetMMPStatus:              request.NewRateLimit(twoSecondsInterval, 5),
		ResetRFQMMPStatus:           request.NewRateLimit(twoSecondsInterval, 5),
		SetMMP:                      request.NewRateLimit(tenSecondsInterval, 2),
		GetMMPConfig:                request.NewRateLimit(twoSecondsInterval, 5),
		CreateQuote:                 request.NewRateLimit(twoSecondsInterval, 50),
		CancelQuote:                 request.NewRateLimit(twoSecondsInterval, 50),
		CancelMultipleQuotes:        request.NewRateLimit(twoSecondsInterval, 2),
		CancelAllQuotes:             request.NewRateLimit(twoSecondsInterval, 2),
		GetRfqs:                     request.NewRateLimit(twoSecondsInterval, 2),
		GetQuotes:                   request.NewRateLimit(twoSecondsInterval, 2),
		GetTrades:                   request.NewRateLimit(twoSecondsInterval, 5),
		GetTradesHistory:            request.NewRateLimit(twoSecondsInterval, 10),
		OptionInstrumentTradeFamily: request.NewRateLimit(twoSecondsInterval, 20),
		OptionTrades:                request.NewRateLimit(twoSecondsInterval, 20),
		GetPublicTrades:             request.NewRateLimit(twoSecondsInterval, 5),
		// Funding
		GetCurrencies:              request.NewRateLimit(oneSecondInterval, 6),
		GetBalance:                 request.NewRateLimit(oneSecondInterval, 6),
		GetNonTradableAssets:       request.NewRateLimit(oneSecondInterval, 6),
		GetAccountAssetValuation:   request.NewRateLimit(twoSecondsInterval, 1),
		FundsTransfer:              request.NewRateLimit(oneSecondInterval, 1),
		GetFundsTransferState:      request.NewRateLimit(oneSecondInterval, 1),
		AssetBillsDetails:          request.NewRateLimit(oneSecondInterval, 6),
		LightningDeposits:          request.NewRateLimit(oneSecondInterval, 2),
		GetDepositAddress:          request.NewRateLimit(oneSecondInterval, 6),
		GetDepositHistory:          request.NewRateLimit(oneSecondInterval, 6),
		Withdrawal:                 request.NewRateLimit(oneSecondInterval, 6),
		LightningWithdrawals:       request.NewRateLimit(oneSecondInterval, 2),
		CancelWithdrawal:           request.NewRateLimit(oneSecondInterval, 6),
		GetWithdrawalHistory:       request.NewRateLimit(oneSecondInterval, 6),
		GetDepositWithdrawalStatus: request.NewRateLimit(twoSecondsInterval, 1),
		SmallAssetsConvert:         request.NewRateLimit(oneSecondInterval, 1),
		GetPublicExchangeList:      request.NewRateLimit(oneSecondInterval, 6),
		GetSavingBalance:           request.NewRateLimit(oneSecondInterval, 6),
		SavingsPurchaseRedempt:     request.NewRateLimit(oneSecondInterval, 6),
		SetLendingRate:             request.NewRateLimit(oneSecondInterval, 6),
		GetLendingHistory:          request.NewRateLimit(oneSecondInterval, 6),
		GetPublicBorrowInfo:        request.NewRateLimit(oneSecondInterval, 6),
		GetPublicBorrowHistory:     request.NewRateLimit(oneSecondInterval, 6),
		// Convert
		GetConvertCurrencies:   request.NewRateLimit(oneSecondInterval, 6),
		GetConvertCurrencyPair: request.NewRateLimit(oneSecondInterval, 6),
		EstimateQuote:          request.NewRateLimit(oneSecondInterval, 10),
		ConvertTrade:           request.NewRateLimit(oneSecondInterval, 10),
		GetConvertHistory:      request.NewRateLimit(oneSecondInterval, 6),
		// Account
		GetAccountBalance:                 request.NewRateLimit(twoSecondsInterval, 10),
		GetPositions:                      request.NewRateLimit(twoSecondsInterval, 10),
		GetPositionsHistory:               request.NewRateLimit(tenSecondsInterval, 1),
		GetAccountAndPositionRisk:         request.NewRateLimit(twoSecondsInterval, 10),
		GetBillsDetails:                   request.NewRateLimit(oneSecondInterval, 5),
		GetBillsDetailArchive:             request.NewRateLimit(twoSecondsInterval, 5),
		GetAccountConfiguration:           request.NewRateLimit(twoSecondsInterval, 5),
		SetPositionMode:                   request.NewRateLimit(twoSecondsInterval, 5),
		SetLeverage:                       request.NewRateLimit(twoSecondsInterval, 20),
		GetMaximumBuyOrSellAmount:         request.NewRateLimit(twoSecondsInterval, 20),
		GetMaximumAvailableTradableAmount: request.NewRateLimit(twoSecondsInterval, 20),
		IncreaseOrDecreaseMargin:          request.NewRateLimit(twoSecondsInterval, 20),
		GetLeverage:                       request.NewRateLimit(twoSecondsInterval, 20),
		GetLeverageEstimatedInfo:          request.NewRateLimit(twoSecondsInterval, 5),
		GetTheMaximumLoanOfInstrument:     request.NewRateLimit(twoSecondsInterval, 20),
		GetFeeRates:                       request.NewRateLimit(twoSecondsInterval, 5),
		GetInterestAccruedData:            request.NewRateLimit(twoSecondsInterval, 5),
		GetInterestRate:                   request.NewRateLimit(twoSecondsInterval, 5),
		SetGreeks:                         request.NewRateLimit(twoSecondsInterval, 5),
		IsolatedMarginTradingSettings:     request.NewRateLimit(twoSecondsInterval, 5),
		GetMaximumWithdrawals:             request.NewRateLimit(twoSecondsInterval, 20),
		GetAccountRiskState:               request.NewRateLimit(twoSecondsInterval, 10),
		ManualBorrowAndRepay:              request.NewRateLimit(twoSecondsInterval, 5),
		GetBorrowAndRepayHistory:          request.NewRateLimit(twoSecondsInterval, 5),
		VipLoansBorrowAnsRepay:            request.NewRateLimit(oneSecondInterval, 6),
		GetBorrowAndRepayHistoryHistory:   request.NewRateLimit(twoSecondsInterval, 5),
		GetVIPInterestAccruedData:         request.NewRateLimit(twoSecondsInterval, 5),
		GetVIPInterestDeductedData:        request.NewRateLimit(twoSecondsInterval, 5),
		GetVIPLoanOrderList:               request.NewRateLimit(twoSecondsInterval, 5),
		GetVIPLoanOrderDetail:             request.NewRateLimit(twoSecondsInterval, 5),
		GetBorrowInterestAndLimit:         request.NewRateLimit(twoSecondsInterval, 5),
		PositionBuilder:                   request.NewRateLimit(twoSecondsInterval, 2),
		GetGreeks:                         request.NewRateLimit(twoSecondsInterval, 10),
		GetPMLimitation:                   request.NewRateLimit(twoSecondsInterval, 10),
		SetRiskOffsetType:                 request.NewRateLimit(twoSecondsInterval, 10),
		ActivateOption:                    request.NewRateLimit(twoSecondsInterval, 5),
		SetAutoLoan:                       request.NewRateLimit(twoSecondsInterval, 5),
		SetAccountLevel:                   request.NewRateLimit(twoSecondsInterval, 5),

		// Sub Account Endpoints

		ViewSubaccountList:                             request.NewRateLimit(twoSecondsInterval, 2),
		ResetSubAccountAPIKey:                          request.NewRateLimit(oneSecondInterval, 1),
		GetSubaccountTradingBalance:                    request.NewRateLimit(twoSecondsInterval, 2),
		GetSubaccountFundingBalance:                    request.NewRateLimit(twoSecondsInterval, 2),
		HistoryOfSubaccountTransfer:                    request.NewRateLimit(oneSecondInterval, 6),
		ManagedSubAccountTransfer:                      request.NewRateLimit(oneSecondInterval, 6),
		MasterAccountsManageTransfersBetweenSubaccount: request.NewRateLimit(oneSecondInterval, 1),
		SetPermissionOfTransferOut:                     request.NewRateLimit(oneSecondInterval, 1),
		GetCustodyTradingSubaccountList:                request.NewRateLimit(oneSecondInterval, 1),
		SetSubAccountVIPLoanAllocation:                 request.NewRateLimit(oneSecondInterval, 5),
		GetSubAccountBorrowInterestAndLimit:            request.NewRateLimit(twoSecondsInterval, 5),
		// Grid Trading Endpoints

		GridTrading:                             request.NewRateLimit(twoSecondsInterval, 20),
		AmendGridAlgoOrder:                      request.NewRateLimit(twoSecondsInterval, 20),
		StopGridAlgoOrder:                       request.NewRateLimit(twoSecondsInterval, 20),
		ClosePositionForForContractGrid:         request.NewRateLimit(twoSecondsInterval, 20),
		CancelClosePositionOrderForContractGrid: request.NewRateLimit(twoSecondsInterval, 20),
		InstantTriggerGridAlgoOrder:             request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAlgoOrderList:                    request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAlgoOrderHistory:                 request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAlgoOrderDetails:                 request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAlgoSubOrders:                    request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAlgoOrderPositions:               request.NewRateLimit(twoSecondsInterval, 20),
		SpotGridWithdrawIncome:                  request.NewRateLimit(twoSecondsInterval, 20),
		ComputeMarginBalance:                    request.NewRateLimit(twoSecondsInterval, 20),
		AdjustMarginBalance:                     request.NewRateLimit(twoSecondsInterval, 20),
		GetGridAIParameter:                      request.NewRateLimit(twoSecondsInterval, 20),
		ComputeMinInvestment:                    request.NewRateLimit(twoSecondsInterval, 20),
		RSIBackTesting:                          request.NewRateLimit(twoSecondsInterval, 20),

		// Signal Bot Trading
		SignalBotOrderDetails: request.NewRateLimit(twoSecondsInterval, 20),
		SignalBotPosition:     request.NewRateLimit(twoSecondsInterval, 20),
		SignalBotSubOrders:    request.NewRateLimit(twoSecondsInterval, 20),
		SignalBotEventHistory: request.NewRateLimit(twoSecondsInterval, 20),

		// Recurring Buy Order
		PlaceRecurringBuyOrder:           request.NewRateLimit(twoSecondsInterval, 20),
		AmendRecurringBuyOrder:           request.NewRateLimit(twoSecondsInterval, 20),
		StopRecurringBuyOrder:            request.NewRateLimit(twoSecondsInterval, 20),
		GetRecurringBuyOrder:             request.NewRateLimit(twoSecondsInterval, 20),
		GetRecurringBuyOrderHistory:      request.NewRateLimit(twoSecondsInterval, 20),
		GetRecurringBuyOrderDetail:       request.NewRateLimit(twoSecondsInterval, 20),
		GetRecurringBuySubOrders:         request.NewRateLimit(twoSecondsInterval, 20),
		GetExistingLeadingPositions:      request.NewRateLimit(twoSecondsInterval, 20),
		GetLeadingPositionHistory:        request.NewRateLimit(twoSecondsInterval, 20),
		PlaceLeadingStopOrder:            request.NewRateLimit(twoSecondsInterval, 20),
		GetCloseLeadingPosition:          request.NewRateLimit(twoSecondsInterval, 20),
		GetLeadingInstruments:            request.NewRateLimit(twoSecondsInterval, 5),
		GetProfitSharingLimit:            request.NewRateLimit(twoSecondsInterval, 5),
		GetTotalProfitSharing:            request.NewRateLimit(twoSecondsInterval, 5),
		SetFirstCopySettings:             request.NewRateLimit(twoSecondsInterval, 5),
		AmendFirstCopySettings:           request.NewRateLimit(twoSecondsInterval, 5),
		StopCopying:                      request.NewRateLimit(twoSecondsInterval, 5),
		GetCopySettings:                  request.NewRateLimit(twoSecondsInterval, 5),
		GetMultipleLeverages:             request.NewRateLimit(twoSecondsInterval, 5),
		SetBatchLeverage:                 request.NewRateLimit(twoSecondsInterval, 5),
		GetMyLeadTraders:                 request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderRanks:               request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderWeeklyPNL:           request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderDailyPNL:            request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderStats:               request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderCurrencyPreferences: request.NewRateLimit(twoSecondsInterval, 5),
		GetTraderCurrentLeadPositions:    request.NewRateLimit(twoSecondsInterval, 5),
		GetLeadTraderLeadPositionHistory: request.NewRateLimit(twoSecondsInterval, 5),

		// Earn
		GetOffer:                   request.NewRateLimit(oneSecondInterval, 3),
		Purchase:                   request.NewRateLimit(oneSecondInterval, 2),
		Redeem:                     request.NewRateLimit(oneSecondInterval, 2),
		CancelPurchaseOrRedemption: request.NewRateLimit(oneSecondInterval, 2),
		GetEarnActiveOrders:        request.NewRateLimit(oneSecondInterval, 3),
		GetFundingOrderHistory:     request.NewRateLimit(oneSecondInterval, 3),

		// ETH Staking
		PurchaseETHStaking:       request.NewRateLimit(oneSecondInterval, 2),
		RedeemETHStaking:         request.NewRateLimit(oneSecondInterval, 2),
		GetBETHBalance:           request.NewRateLimit(oneSecondInterval, 6),
		GetPurchaseRedeemHistory: request.NewRateLimit(oneSecondInterval, 6),
		GetAPYHistory:            request.NewRateLimit(oneSecondInterval, 6),

		// Market Data
		GetTickers:                     request.NewRateLimit(twoSecondsInterval, 20),
		GetTicker:                      request.NewRateLimit(twoSecondsInterval, 20),
		GetIndexTickers:                request.NewRateLimit(twoSecondsInterval, 20),
		GetOrderBook:                   request.NewRateLimit(twoSecondsInterval, 40),
		GetOrderBooksLite:              request.NewRateLimit(twoSecondsInterval, 6),
		GetCandlesticks:                request.NewRateLimit(twoSecondsInterval, 40),
		GetCandlesticksHistory:         request.NewRateLimit(twoSecondsInterval, 20),
		GetIndexCandlestickHistory:     request.NewRateLimit(twoSecondsInterval, 10),
		GetMarkPriceCandlestickHistory: request.NewRateLimit(twoSecondsInterval, 10),
		GetEconomicCalendar:            request.NewRateLimit(oneSecondInterval, 5),
		GetIndexCandlesticks:           request.NewRateLimit(twoSecondsInterval, 20),
		GetMarkPriceCandlesticks:       request.NewRateLimit(twoSecondsInterval, 20),
		GetTradesRequest:               request.NewRateLimit(twoSecondsInterval, 100),
		Get24HTotalVolume:              request.NewRateLimit(twoSecondsInterval, 2),
		GetOracle:                      request.NewRateLimit(fiveSecondsInterval, 1),
		GetExchangeRateRequest:         request.NewRateLimit(twoSecondsInterval, 1),
		GetIndexComponents:             request.NewRateLimit(twoSecondsInterval, 20),
		GetBlockTickers:                request.NewRateLimit(twoSecondsInterval, 20),
		GetBlockTrades:                 request.NewRateLimit(twoSecondsInterval, 20),

		// Spread Orders rate limiters
		PlaceSpreadOrder:      request.NewRateLimit(twoSecondsInterval, 20),
		CancelSpreadOrder:     request.NewRateLimit(twoSecondsInterval, 20),
		CancelAllSpreadOrder:  request.NewRateLimit(twoSecondsInterval, 10),
		AmendSpreadOrder:      request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreadOrderDetails: request.NewRateLimit(twoSecondsInterval, 20),
		GetActiveSpreadOrders: request.NewRateLimit(twoSecondsInterval, 10),
		GetSpreadOrders7Days:  request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreadOrderTrades:  request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreads:            request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreadOrderBook:    request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreadTicker:       request.NewRateLimit(twoSecondsInterval, 20),
		GetSpreadPublicTrades: request.NewRateLimit(twoSecondsInterval, 20),

		// Public Data Endpoints

		GetInstruments:                         request.NewRateLimit(twoSecondsInterval, 20),
		GetDeliveryExerciseHistory:             request.NewRateLimit(twoSecondsInterval, 40),
		GetOpenInterest:                        request.NewRateLimit(twoSecondsInterval, 20),
		GetFundingRate:                         request.NewRateLimit(twoSecondsInterval, 20),
		GetFundingRateHistory:                  request.NewRateLimit(twoSecondsInterval, 10),
		GetLimitPrice:                          request.NewRateLimit(twoSecondsInterval, 20),
		GetOptionMarketDate:                    request.NewRateLimit(twoSecondsInterval, 20),
		GetEstimatedDeliveryExercisePrice:      request.NewRateLimit(twoSecondsInterval, 10),
		GetDiscountRateAndInterestFreeQuota:    request.NewRateLimit(twoSecondsInterval, 2),
		GetSystemTime:                          request.NewRateLimit(twoSecondsInterval, 10),
		GetLiquidationOrders:                   request.NewRateLimit(twoSecondsInterval, 40), // Missing from documentation
		GetMarkPrice:                           request.NewRateLimit(twoSecondsInterval, 10),
		GetPositionTiers:                       request.NewRateLimit(twoSecondsInterval, 10),
		GetInterestRateAndLoanQuota:            request.NewRateLimit(twoSecondsInterval, 2),
		GetInterestRateAndLoanQuoteForVIPLoans: request.NewRateLimit(twoSecondsInterval, 2),
		GetUnderlying:                          request.NewRateLimit(twoSecondsInterval, 20),
		GetInsuranceFund:                       request.NewRateLimit(twoSecondsInterval, 10),
		UnitConvert:                            request.NewRateLimit(twoSecondsInterval, 10),
		OptionTickBands:                        request.NewRateLimit(twoSecondsInterval, 5),
		GetIndexTicker:                         request.NewRateLimit(twoSecondsInterval, 20),

		// Trading Data Endpoints

		GetSupportCoin:                    request.NewRateLimit(twoSecondsInterval, 5),
		GetTakerVolume:                    request.NewRateLimit(twoSecondsInterval, 5),
		GetMarginLendingRatio:             request.NewRateLimit(twoSecondsInterval, 5),
		GetLongShortRatio:                 request.NewRateLimit(twoSecondsInterval, 5),
		GetContractsOpenInterestAndVolume: request.NewRateLimit(twoSecondsInterval, 5),
		GetOptionsOpenInterestAndVolume:   request.NewRateLimit(twoSecondsInterval, 5),
		GetPutCallRatio:                   request.NewRateLimit(twoSecondsInterval, 5),
		GetOpenInterestAndVolume:          request.NewRateLimit(twoSecondsInterval, 5),
		GetTakerFlow:                      request.NewRateLimit(twoSecondsInterval, 5),

		// Status Endpoints

		GetEventStatus:                   request.NewRateLimit(fiveSecondsInterval, 1),
		GetAffilateInviteesDetail:        request.NewRateLimit(twoSecondsInterval, 20),
		GetUserAffilateRebateInformation: request.NewRateLimit(twoSecondsInterval, 20),
	}
}
