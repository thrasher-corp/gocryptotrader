package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Ratelimit intervals.
const (
	oneMinuteInterval = time.Minute
	// Bitfinex rate limits - Public
	undocumentedFallback5ReqRate  = 5  // Fallback for undocumented legacy V1 endpoints that have historically used 5 req/min.
	undocumentedFallback10ReqRate = 10 // Fallback for undocumented legacy V1 endpoints that have historically used 10 req/min.
	undocumentedFallback15ReqRate = 15 // Fallback for undocumented legacy V1 endpoints that have historically used 15 req/min.
	undocumentedFallback45ReqRate = 45 // Fallback for endpoints with no explicit per-endpoint value in current Bitfinex docs.

	platformStatusReqRate     = 15
	tickerBatchReqRate        = 30
	tickerReqRate             = 90
	tradeReqRate              = 30
	orderbookReqRate          = 240
	statsReqRate              = 90
	candleReqRate             = 30
	configsReqRate            = 15
	statusReqRate             = undocumentedFallback15ReqRate
	liquidReqRate             = undocumentedFallback15ReqRate
	leaderBoardReqRate        = 90
	marketAveragePriceReqRate = 20
	fxReqRate                 = 90

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalanceReqRate = undocumentedFallback45ReqRate
	accountWalletHistoryReqRate = undocumentedFallback45ReqRate
	// Orders -
	retrieveOrderReqRate  = undocumentedFallback45ReqRate
	submitOrderReqRate    = undocumentedFallback45ReqRate
	updateOrderReqRate    = undocumentedFallback45ReqRate
	cancelOrderReqRate    = undocumentedFallback45ReqRate
	orderBatchReqRate     = undocumentedFallback45ReqRate
	cancelBatchReqRate    = undocumentedFallback45ReqRate
	orderHistoryReqRate   = undocumentedFallback45ReqRate
	getOrderTradesReqRate = undocumentedFallback45ReqRate
	getTradesReqRate      = undocumentedFallback45ReqRate
	getLedgersReqRate     = undocumentedFallback45ReqRate
	// Positions -
	getAccountMarginInfoReqRate       = undocumentedFallback45ReqRate
	getActivePositionsReqRate         = undocumentedFallback45ReqRate
	claimPositionReqRate              = undocumentedFallback45ReqRate
	getPositionHistoryReqRate         = undocumentedFallback45ReqRate
	getPositionAuditReqRate           = undocumentedFallback45ReqRate
	updateCollateralOnPositionReqRate = undocumentedFallback45ReqRate
	// Margin funding -
	getMarginInfoRate               = 90
	getActiveFundingOffersReqRate   = undocumentedFallback45ReqRate
	submitFundingOfferReqRate       = undocumentedFallback45ReqRate
	cancelFundingOfferReqRate       = undocumentedFallback45ReqRate
	cancelAllFundingOfferReqRate    = undocumentedFallback45ReqRate
	closeFundingReqRate             = undocumentedFallback45ReqRate
	fundingAutoRenewReqRate         = undocumentedFallback45ReqRate
	keepFundingReqRate              = undocumentedFallback45ReqRate
	getOffersHistoryReqRate         = undocumentedFallback45ReqRate
	getFundingLoansReqRate          = undocumentedFallback45ReqRate
	getFundingLoanHistoryReqRate    = undocumentedFallback45ReqRate
	getFundingCreditsReqRate        = undocumentedFallback45ReqRate
	getFundingCreditsHistoryReqRate = undocumentedFallback45ReqRate
	getFundingTradesReqRate         = undocumentedFallback45ReqRate
	getFundingInfoReqRate           = undocumentedFallback45ReqRate
	// Account actions
	getUserInfoReqRate               = undocumentedFallback45ReqRate
	transferBetweenWalletsReqRate    = undocumentedFallback45ReqRate
	getDepositAddressReqRate         = undocumentedFallback45ReqRate
	withdrawalReqRate                = undocumentedFallback45ReqRate
	getMovementsReqRate              = undocumentedFallback45ReqRate
	getAlertListReqRate              = undocumentedFallback45ReqRate
	setPriceAlertReqRate             = 90
	deletePriceAlertReqRate          = 90
	getBalanceForOrdersOffersReqRate = 90
	userSettingsWriteReqRate         = 90
	userSettingsReadReqRate          = 90
	userSettingsDeleteReqRate        = 90
	// Account V1 endpoints
	getAccountFeesReqRate    = 5
	getWithdrawalFeesReqRate = 5
	getAccountSummaryReqRate = undocumentedFallback5ReqRate
	newDepositAddressReqRate = undocumentedFallback5ReqRate
	getKeyPermissionsReqRate = undocumentedFallback5ReqRate
	getMarginInfoReqRate     = undocumentedFallback5ReqRate
	getAccountBalanceReqRate = 10
	walletTransferReqRate    = undocumentedFallback10ReqRate
	withdrawV1ReqRate        = undocumentedFallback10ReqRate
	orderV1ReqRate           = undocumentedFallback10ReqRate
	orderMultiReqRate        = undocumentedFallback10ReqRate
	statsV1ReqRate           = 10
	fundingBookReqRate       = 15
	lendsReqRate             = 30

	// Rate limit endpoint functionality declaration
	platformStatus request.EndpointLimit = iota
	tickerBatch
	tickerFunction
	tradeRateLimit
	orderbookFunction
	stats
	candle
	configs
	status
	liquid
	leaderBoard
	marketAveragePrice
	fx

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalance
	accountWalletHistory
	// Orders -
	retrieveOrder
	submitOrder
	updateOrder
	cancelOrder
	orderBatch
	cancelBatch
	orderHistory
	getOrderTrades
	getTrades
	getLedgers
	// Positions -
	getAccountMarginInfo
	getActivePositions
	claimPosition
	getPositionHistory
	getPositionAudit
	updateCollateralOnPosition
	// Margin funding -
	getActiveFundingOffers
	submitFundingOffer
	cancelFundingOffer
	cancelAllFundingOffer
	closeFunding
	fundingAutoRenew
	keepFunding
	getOffersHistory
	getFundingLoans
	getFundingLoanHistory
	getFundingCredits
	getFundingCreditsHistory
	getFundingTrades
	getFundingInfo
	// Account actions
	getUserInfo
	transferBetweenWallets
	getDepositAddress
	withdrawal
	getMovements
	getAlertList
	setPriceAlert
	deletePriceAlert
	getBalanceForOrdersOffers
	userSettingsWrite
	userSettingsRead
	userSettingsDelete
	// Account V1 endpoints
	getAccountFees
	getWithdrawalFees
	getAccountSummary
	newDepositAddress
	getKeyPermissions
	getMarginInfo
	getAccountBalance
	walletTransfer
	withdrawV1
	orderV1
	orderMulti
	statsV1
	fundingBook
	lends
)

// package level rate limits for REST API
var packageRateLimits = request.RateLimitDefinitions{
	// Public
	platformStatus:       rateLimitPerMinute(platformStatusReqRate),
	tickerBatch:          rateLimitPerMinute(tickerBatchReqRate),
	tickerFunction:       rateLimitPerMinute(tickerReqRate),
	tradeRateLimit:       rateLimitPerMinute(tradeReqRate),
	orderbookFunction:    rateLimitPerMinute(orderbookReqRate),
	stats:                rateLimitPerMinute(statsReqRate),
	candle:               rateLimitPerMinute(candleReqRate),
	configs:              rateLimitPerMinute(configsReqRate),
	status:               rateLimitPerMinute(statusReqRate),
	liquid:               rateLimitPerMinute(liquidReqRate),
	leaderBoard:          rateLimitPerMinute(leaderBoardReqRate),
	marketAveragePrice:   rateLimitPerMinute(marketAveragePriceReqRate),
	fx:                   rateLimitPerMinute(fxReqRate),
	accountWalletBalance: rateLimitPerMinute(accountWalletBalanceReqRate),
	accountWalletHistory: rateLimitPerMinute(accountWalletHistoryReqRate),
	// Orders
	retrieveOrder:  rateLimitPerMinute(retrieveOrderReqRate),
	submitOrder:    rateLimitPerMinute(submitOrderReqRate),
	updateOrder:    rateLimitPerMinute(updateOrderReqRate),
	cancelOrder:    rateLimitPerMinute(cancelOrderReqRate),
	orderBatch:     rateLimitPerMinute(orderBatchReqRate),
	cancelBatch:    rateLimitPerMinute(cancelBatchReqRate),
	orderHistory:   rateLimitPerMinute(orderHistoryReqRate),
	getOrderTrades: rateLimitPerMinute(getOrderTradesReqRate),
	getTrades:      rateLimitPerMinute(getTradesReqRate),
	getLedgers:     rateLimitPerMinute(getLedgersReqRate),
	// Positions
	getAccountMarginInfo:       rateLimitPerMinute(getAccountMarginInfoReqRate),
	getActivePositions:         rateLimitPerMinute(getActivePositionsReqRate),
	claimPosition:              rateLimitPerMinute(claimPositionReqRate),
	getPositionHistory:         rateLimitPerMinute(getPositionHistoryReqRate),
	getPositionAudit:           rateLimitPerMinute(getPositionAuditReqRate),
	updateCollateralOnPosition: rateLimitPerMinute(updateCollateralOnPositionReqRate),
	// Margin funding
	getActiveFundingOffers:   rateLimitPerMinute(getActiveFundingOffersReqRate),
	submitFundingOffer:       rateLimitPerMinute(submitFundingOfferReqRate),
	cancelFundingOffer:       rateLimitPerMinute(cancelFundingOfferReqRate),
	cancelAllFundingOffer:    rateLimitPerMinute(cancelAllFundingOfferReqRate),
	closeFunding:             rateLimitPerMinute(closeFundingReqRate),
	fundingAutoRenew:         rateLimitPerMinute(fundingAutoRenewReqRate),
	keepFunding:              rateLimitPerMinute(keepFundingReqRate),
	getOffersHistory:         rateLimitPerMinute(getOffersHistoryReqRate),
	getFundingLoans:          rateLimitPerMinute(getFundingLoansReqRate),
	getFundingLoanHistory:    rateLimitPerMinute(getFundingLoanHistoryReqRate),
	getFundingCredits:        rateLimitPerMinute(getFundingCreditsReqRate),
	getFundingCreditsHistory: rateLimitPerMinute(getFundingCreditsHistoryReqRate),
	getFundingTrades:         rateLimitPerMinute(getFundingTradesReqRate),
	getFundingInfo:           rateLimitPerMinute(getFundingInfoReqRate),
	// Account actions
	getUserInfo:               rateLimitPerMinute(getUserInfoReqRate),
	transferBetweenWallets:    rateLimitPerMinute(transferBetweenWalletsReqRate),
	getDepositAddress:         rateLimitPerMinute(getDepositAddressReqRate),
	withdrawal:                rateLimitPerMinute(withdrawalReqRate),
	getMovements:              rateLimitPerMinute(getMovementsReqRate),
	getAlertList:              rateLimitPerMinute(getAlertListReqRate),
	setPriceAlert:             rateLimitPerMinute(setPriceAlertReqRate),
	deletePriceAlert:          rateLimitPerMinute(deletePriceAlertReqRate),
	getBalanceForOrdersOffers: rateLimitPerMinute(getBalanceForOrdersOffersReqRate),
	userSettingsWrite:         rateLimitPerMinute(userSettingsWriteReqRate),
	userSettingsRead:          rateLimitPerMinute(userSettingsReadReqRate),
	userSettingsDelete:        rateLimitPerMinute(userSettingsDeleteReqRate),
	// Account V1 endpoints
	getAccountFees:    rateLimitPerMinute(getAccountFeesReqRate),
	getWithdrawalFees: rateLimitPerMinute(getWithdrawalFeesReqRate),
	getAccountSummary: rateLimitPerMinute(getAccountSummaryReqRate),
	newDepositAddress: rateLimitPerMinute(newDepositAddressReqRate),
	getKeyPermissions: rateLimitPerMinute(getKeyPermissionsReqRate),
	getMarginInfo:     rateLimitPerMinute(getMarginInfoReqRate),
	getAccountBalance: rateLimitPerMinute(getAccountBalanceReqRate),
	walletTransfer:    rateLimitPerMinute(walletTransferReqRate),
	withdrawV1:        rateLimitPerMinute(withdrawV1ReqRate),
	orderV1:           rateLimitPerMinute(orderV1ReqRate),
	orderMulti:        rateLimitPerMinute(orderMultiReqRate),
	statsV1:           rateLimitPerMinute(statsV1ReqRate),
	fundingBook:       rateLimitPerMinute(fundingBookReqRate),
	lends:             rateLimitPerMinute(lendsReqRate),
}

func rateLimitPerMinute(limit int) *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(oneMinuteInterval, limit, 1)
}

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return packageRateLimits
}
