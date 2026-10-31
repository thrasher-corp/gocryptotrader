package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitfinex limits each REST endpoint separately, per IP, in requests a minute, and blocks an IP that exceeds a limit
// for 60 seconds; see https://docs.bitfinex.com/docs/requirements-and-limitations. Endpoints whose docs give no limit
// use undocumentedRate, the lowest of the 10 to 90 a minute the docs give as the general policy
const undocumentedRate = 10

// Endpoint rate limits, in the order the docs list the endpoints
const (
	platformStatusEPL request.EndpointLimit = iota
	tickerEPL
	tickersEPL
	tickersHistoryEPL
	publicTradesEPL
	bookEPL
	statsEPL
	candlesEPL
	derivativesStatusEPL
	derivativesStatusHistoryEPL
	liquidationsEPL
	leaderboardsEPL
	fundingStatisticsEPL
	configsEPL
	virtualAssetServiceProvidersEPL
	marketAveragePriceEPL
	foreignExchangeRateEPL
	walletsEPL
	retrieveOrdersEPL
	retrieveOrdersBySymbolEPL
	submitOrderEPL
	updateOrderEPL
	cancelOrderEPL
	cancelOrdersEPL
	orderMultiOpEPL
	ordersHistoryEPL
	ordersHistoryBySymbolEPL
	orderTradesEPL
	tradesEPL
	tradesBySymbolEPL
	otcOrdersHistoryEPL
	ledgersEPL
	marginInfoEPL
	positionsEPL
	claimPositionEPL
	increasePositionEPL
	increasePositionInfoEPL
	positionsHistoryEPL
	positionsSnapshotEPL
	positionsAuditEPL
	updatePositionFundingTypeEPL
	derivativePositionCollateralEPL
	derivativePositionCollateralLimitsEPL
	fundingOffersEPL
	submitFundingOfferEPL
	cancelFundingOfferEPL
	cancelAllFundingOffersEPL
	fundingCloseEPL
	fundingAutoRenewEPL
	keepFundingEPL
	fundingOffersHistoryEPL
	fundingLoansEPL
	fundingLoansHistoryEPL
	fundingCreditsEPL
	fundingCreditsHistoryEPL
	fundingTradesEPL
	fundingInfoEPL
	userInfoEPL
	summaryEPL
	loginHistoryEPL
	keyPermissionsEPL
	generateTokenEPL
	changelogEPL
	transferEPL
	depositAddressEPL
	depositAddressListEPL
	generateInvoiceEPL
	lnxInvoicePaymentsEPL
	withdrawalEPL
	movementsEPL
	movementInfoEPL
	alertListEPL
	alertSetEPL
	alertDeleteEPL
	balanceAvailableEPL
	userSettingsWriteEPL
	userSettingsReadEPL
	userSettingsDeleteEPL
	thalexDepositEPL
	thalexWithdrawalEPL
	thalexFreeTransferCountEPL
	// fiatWithdrawalEPL is the v1 withdrawal, which wire withdrawals still use
	fiatWithdrawalEPL
)

// GetRateLimit returns the rate limit of every REST endpoint
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		platformStatusEPL:                     request.NewRateLimitWithWeight(time.Minute, 30, 1),
		tickerEPL:                             request.NewRateLimitWithWeight(time.Minute, 90, 1),
		tickersEPL:                            request.NewRateLimitWithWeight(time.Minute, 30, 1),
		tickersHistoryEPL:                     request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		publicTradesEPL:                       request.NewRateLimitWithWeight(time.Minute, 15, 1),
		bookEPL:                               request.NewRateLimitWithWeight(time.Minute, 240, 1),
		statsEPL:                              request.NewRateLimitWithWeight(time.Minute, 15, 1),
		candlesEPL:                            request.NewRateLimitWithWeight(time.Minute, 30, 1),
		derivativesStatusEPL:                  request.NewRateLimitWithWeight(time.Minute, 90, 1),
		derivativesStatusHistoryEPL:           request.NewRateLimitWithWeight(time.Minute, 90, 1),
		liquidationsEPL:                       request.NewRateLimitWithWeight(time.Minute, 3, 1),
		leaderboardsEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingStatisticsEPL:                  request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		configsEPL:                            request.NewRateLimitWithWeight(time.Minute, 90, 1),
		virtualAssetServiceProvidersEPL:       request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		marketAveragePriceEPL:                 request.NewRateLimitWithWeight(time.Minute, 90, 1),
		foreignExchangeRateEPL:                request.NewRateLimitWithWeight(time.Minute, 90, 1),
		walletsEPL:                            request.NewRateLimitWithWeight(time.Minute, 90, 1),
		retrieveOrdersEPL:                     request.NewRateLimitWithWeight(time.Minute, 90, 1),
		retrieveOrdersBySymbolEPL:             request.NewRateLimitWithWeight(time.Minute, 90, 1),
		submitOrderEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		updateOrderEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		cancelOrderEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		cancelOrdersEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		orderMultiOpEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		ordersHistoryEPL:                      request.NewRateLimitWithWeight(time.Minute, 90, 1),
		ordersHistoryBySymbolEPL:              request.NewRateLimitWithWeight(time.Minute, 90, 1),
		orderTradesEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		tradesEPL:                             request.NewRateLimitWithWeight(time.Minute, 90, 1),
		tradesBySymbolEPL:                     request.NewRateLimitWithWeight(time.Minute, 90, 1),
		otcOrdersHistoryEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		ledgersEPL:                            request.NewRateLimitWithWeight(time.Minute, 90, 1),
		marginInfoEPL:                         request.NewRateLimitWithWeight(time.Minute, 90, 1),
		positionsEPL:                          request.NewRateLimitWithWeight(time.Minute, 90, 1),
		claimPositionEPL:                      request.NewRateLimitWithWeight(time.Minute, 90, 1),
		increasePositionEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		increasePositionInfoEPL:               request.NewRateLimitWithWeight(time.Minute, 90, 1),
		positionsHistoryEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		positionsSnapshotEPL:                  request.NewRateLimitWithWeight(time.Minute, 90, 1),
		positionsAuditEPL:                     request.NewRateLimitWithWeight(time.Minute, 90, 1),
		updatePositionFundingTypeEPL:          request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		derivativePositionCollateralEPL:       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		derivativePositionCollateralLimitsEPL: request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingOffersEPL:                      request.NewRateLimitWithWeight(time.Minute, 90, 1),
		submitFundingOfferEPL:                 request.NewRateLimitWithWeight(time.Minute, 90, 1),
		cancelFundingOfferEPL:                 request.NewRateLimitWithWeight(time.Minute, 90, 1),
		cancelAllFundingOffersEPL:             request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingCloseEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingAutoRenewEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		keepFundingEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingOffersHistoryEPL:               request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingLoansEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingLoansHistoryEPL:                request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingCreditsEPL:                     request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingCreditsHistoryEPL:              request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingTradesEPL:                      request.NewRateLimitWithWeight(time.Minute, 90, 1),
		fundingInfoEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		userInfoEPL:                           request.NewRateLimitWithWeight(time.Minute, 90, 1),
		summaryEPL:                            request.NewRateLimitWithWeight(time.Minute, 90, 1),
		loginHistoryEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		keyPermissionsEPL:                     request.NewRateLimitWithWeight(time.Minute, 90, 1),
		generateTokenEPL:                      request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		changelogEPL:                          request.NewRateLimitWithWeight(time.Minute, 90, 1),
		transferEPL:                           request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		depositAddressEPL:                     request.NewRateLimitWithWeight(time.Minute, 10, 1),
		depositAddressListEPL:                 request.NewRateLimitWithWeight(time.Minute, 10, 1),
		generateInvoiceEPL:                    request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		lnxInvoicePaymentsEPL:                 request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		withdrawalEPL:                         request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		movementsEPL:                          request.NewRateLimitWithWeight(time.Minute, 90, 1),
		movementInfoEPL:                       request.NewRateLimitWithWeight(time.Minute, 90, 1),
		alertListEPL:                          request.NewRateLimitWithWeight(time.Minute, 45, 1),
		alertSetEPL:                           request.NewRateLimitWithWeight(time.Minute, 90, 1),
		alertDeleteEPL:                        request.NewRateLimitWithWeight(time.Minute, 90, 1),
		balanceAvailableEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		userSettingsWriteEPL:                  request.NewRateLimitWithWeight(time.Minute, 90, 1),
		userSettingsReadEPL:                   request.NewRateLimitWithWeight(time.Minute, 90, 1),
		userSettingsDeleteEPL:                 request.NewRateLimitWithWeight(time.Minute, 90, 1),
		thalexDepositEPL:                      request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		thalexWithdrawalEPL:                   request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		thalexFreeTransferCountEPL:            request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
		fiatWithdrawalEPL:                     request.NewRateLimitWithWeight(time.Minute, undocumentedRate, 1),
	}
}
