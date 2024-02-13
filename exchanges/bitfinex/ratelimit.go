package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Bitfinex rate limits - Public
	requestLimitInterval      = time.Minute
	platformStatusReqRate     = 15
	tickerBatchReqRate        = 30
	tickerReqRate             = 30
	tradeReqRate              = 30
	orderbookReqRate          = 30
	statsReqRate              = 90
	candleReqRate             = 60
	configsReqRate            = 15
	statusReqRate             = 15 // This is not specified just inputted WCS
	liquidReqRate             = 15 // This is not specified just inputted WCS
	leaderBoardReqRate        = 90
	marketAveragePriceReqRate = 20
	fxReqRate                 = 20

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalanceReqRate = 45
	accountWalletHistoryReqRate = 45
	// Orders -
	retrieveOrderReqRate  = 45
	submitOrderReqRate    = 45 // This is not specified just inputted above
	updateOrderReqRate    = 45 // This is not specified just inputted above
	cancelOrderReqRate    = 45 // This is not specified just inputted above
	orderBatchReqRate     = 45 // This is not specified just inputted above
	cancelBatchReqRate    = 45 // This is not specified just inputted above
	orderHistoryReqRate   = 45
	getOrderTradesReqRate = 45
	getTradesReqRate      = 45
	getLedgersReqRate     = 45
	// Positions -
	getAccountMarginInfoReqRate       = 45
	getActivePositionsReqRate         = 45
	claimPositionReqRate              = 45 // This is not specified just inputted above
	getPositionHistoryReqRate         = 45
	getPositionAuditReqRate           = 45
	updateCollateralOnPositionReqRate = 45 // This is not specified just inputted above
	// Margin funding -
	getMarginInfoRate               = 90
	getActiveFundingOffersReqRate   = 45
	submitFundingOfferReqRate       = 45 // This is not specified just inputted above
	cancelFundingOfferReqRate       = 45
	cancelAllFundingOfferReqRate    = 45 // This is not specified just inputted above
	closeFundingReqRate             = 45 // This is not specified just inputted above
	fundingAutoRenewReqRate         = 45 // This is not specified just inputted above
	keepFundingReqRate              = 45 // This is not specified just inputted above
	getOffersHistoryReqRate         = 45
	getFundingLoansReqRate          = 45
	getFundingLoanHistoryReqRate    = 45
	getFundingCreditsReqRate        = 45
	getFundingCreditsHistoryReqRate = 45
	getFundingTradesReqRate         = 45
	getFundingInfoReqRate           = 45
	// Account actions
	getUserInfoReqRate               = 45
	transferBetweenWalletsReqRate    = 45 // This is not specified just inputted above
	getDepositAddressReqRate         = 45 // This is not specified just inputted above
	withdrawalReqRate                = 45 // This is not specified just inputted above
	getMovementsReqRate              = 45
	getAlertListReqRate              = 45
	setPriceAlertReqRate             = 45 // This is not specified just inputted above
	deletePriceAlertReqRate          = 45 // This is not specified just inputted above
	getBalanceForOrdersOffersReqRate = 30
	userSettingsWriteReqRate         = 45 // This is not specified just inputted general count
	userSettingsReadReqRate          = 45
	userSettingsDeleteReqRate        = 45 // This is not specified just inputted above
	// Account V1 endpoints
	getAccountFeesReqRate    = 5
	getWithdrawalFeesReqRate = 5
	getAccountSummaryReqRate = 5 // This is not specified just inputted above
	newDepositAddressReqRate = 5 // This is not specified just inputted above
	getKeyPermissionsReqRate = 5 // This is not specified just inputted above
	getMarginInfoReqRate     = 5 // This is not specified just inputted above
	getAccountBalanceReqRate = 10
	walletTransferReqRate    = 10 // This is not specified just inputted above
	withdrawV1ReqRate        = 10 // This is not specified just inputted above
	orderV1ReqRate           = 10 // This is not specified just inputted above
	orderMultiReqRate        = 10 // This is not specified just inputted above
	statsV1ReqRate           = 10
	fundingbookReqRate       = 15
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
	fundingbook
	lends
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		platformStatus:       request.NewRateLimitWithToken(requestLimitInterval, platformStatusReqRate, 1),
		tickerBatch:          request.NewRateLimitWithToken(requestLimitInterval, tickerBatchReqRate, 1),
		tickerFunction:       request.NewRateLimitWithToken(requestLimitInterval, tickerReqRate, 1),
		tradeRateLimit:       request.NewRateLimitWithToken(requestLimitInterval, tradeReqRate, 1),
		orderbookFunction:    request.NewRateLimitWithToken(requestLimitInterval, orderbookReqRate, 1),
		stats:                request.NewRateLimitWithToken(requestLimitInterval, statsReqRate, 1),
		candle:               request.NewRateLimitWithToken(requestLimitInterval, candleReqRate, 1),
		configs:              request.NewRateLimitWithToken(requestLimitInterval, configsReqRate, 1),
		status:               request.NewRateLimitWithToken(requestLimitInterval, statusReqRate, 1),
		liquid:               request.NewRateLimitWithToken(requestLimitInterval, liquidReqRate, 1),
		leaderBoard:          request.NewRateLimitWithToken(requestLimitInterval, leaderBoardReqRate, 1),
		marketAveragePrice:   request.NewRateLimitWithToken(requestLimitInterval, marketAveragePriceReqRate, 1),
		fx:                   request.NewRateLimitWithToken(requestLimitInterval, fxReqRate, 1),
		accountWalletBalance: request.NewRateLimitWithToken(requestLimitInterval, accountWalletBalanceReqRate, 1),
		accountWalletHistory: request.NewRateLimitWithToken(requestLimitInterval, accountWalletHistoryReqRate, 1),
		// Orders -
		retrieveOrder:  request.NewRateLimitWithToken(requestLimitInterval, retrieveOrderReqRate, 1),
		submitOrder:    request.NewRateLimitWithToken(requestLimitInterval, submitOrderReqRate, 1),
		updateOrder:    request.NewRateLimitWithToken(requestLimitInterval, updateOrderReqRate, 1),
		cancelOrder:    request.NewRateLimitWithToken(requestLimitInterval, cancelOrderReqRate, 1),
		orderBatch:     request.NewRateLimitWithToken(requestLimitInterval, orderBatchReqRate, 1),
		cancelBatch:    request.NewRateLimitWithToken(requestLimitInterval, cancelBatchReqRate, 1),
		orderHistory:   request.NewRateLimitWithToken(requestLimitInterval, orderHistoryReqRate, 1),
		getOrderTrades: request.NewRateLimitWithToken(requestLimitInterval, getOrderTradesReqRate, 1),
		getTrades:      request.NewRateLimitWithToken(requestLimitInterval, getTradesReqRate, 1),
		getLedgers:     request.NewRateLimitWithToken(requestLimitInterval, getLedgersReqRate, 1),
		// Positions -
		getAccountMarginInfo:       request.NewRateLimitWithToken(requestLimitInterval, getAccountMarginInfoReqRate, 1),
		getActivePositions:         request.NewRateLimitWithToken(requestLimitInterval, getActivePositionsReqRate, 1),
		claimPosition:              request.NewRateLimitWithToken(requestLimitInterval, claimPositionReqRate, 1),
		getPositionHistory:         request.NewRateLimitWithToken(requestLimitInterval, getPositionAuditReqRate, 1),
		getPositionAudit:           request.NewRateLimitWithToken(requestLimitInterval, getPositionAuditReqRate, 1),
		updateCollateralOnPosition: request.NewRateLimitWithToken(requestLimitInterval, updateCollateralOnPositionReqRate, 1),
		// Margin funding -
		getActiveFundingOffers:   request.NewRateLimitWithToken(requestLimitInterval, getActiveFundingOffersReqRate, 1),
		submitFundingOffer:       request.NewRateLimitWithToken(requestLimitInterval, submitFundingOfferReqRate, 1),
		cancelFundingOffer:       request.NewRateLimitWithToken(requestLimitInterval, cancelFundingOfferReqRate, 1),
		cancelAllFundingOffer:    request.NewRateLimitWithToken(requestLimitInterval, cancelAllFundingOfferReqRate, 1),
		closeFunding:             request.NewRateLimitWithToken(requestLimitInterval, closeFundingReqRate, 1),
		fundingAutoRenew:         request.NewRateLimitWithToken(requestLimitInterval, fundingAutoRenewReqRate, 1),
		keepFunding:              request.NewRateLimitWithToken(requestLimitInterval, keepFundingReqRate, 1),
		getOffersHistory:         request.NewRateLimitWithToken(requestLimitInterval, getOffersHistoryReqRate, 1),
		getFundingLoans:          request.NewRateLimitWithToken(requestLimitInterval, getOffersHistoryReqRate, 1),
		getFundingLoanHistory:    request.NewRateLimitWithToken(requestLimitInterval, getFundingLoanHistoryReqRate, 1),
		getFundingCredits:        request.NewRateLimitWithToken(requestLimitInterval, getFundingCreditsReqRate, 1),
		getFundingCreditsHistory: request.NewRateLimitWithToken(requestLimitInterval, getFundingCreditsHistoryReqRate, 1),
		getFundingTrades:         request.NewRateLimitWithToken(requestLimitInterval, getFundingTradesReqRate, 1),
		getFundingInfo:           request.NewRateLimitWithToken(requestLimitInterval, getFundingInfoReqRate, 1),
		// Account actions
		getUserInfo:               request.NewRateLimitWithToken(requestLimitInterval, getUserInfoReqRate, 1),
		transferBetweenWallets:    request.NewRateLimitWithToken(requestLimitInterval, transferBetweenWalletsReqRate, 1),
		getDepositAddress:         request.NewRateLimitWithToken(requestLimitInterval, getDepositAddressReqRate, 1),
		withdrawal:                request.NewRateLimitWithToken(requestLimitInterval, withdrawalReqRate, 1),
		getMovements:              request.NewRateLimitWithToken(requestLimitInterval, getMovementsReqRate, 1),
		getAlertList:              request.NewRateLimitWithToken(requestLimitInterval, getAlertListReqRate, 1),
		setPriceAlert:             request.NewRateLimitWithToken(requestLimitInterval, setPriceAlertReqRate, 1),
		deletePriceAlert:          request.NewRateLimitWithToken(requestLimitInterval, deletePriceAlertReqRate, 1),
		getBalanceForOrdersOffers: request.NewRateLimitWithToken(requestLimitInterval, getBalanceForOrdersOffersReqRate, 1),
		userSettingsWrite:         request.NewRateLimitWithToken(requestLimitInterval, userSettingsWriteReqRate, 1),
		userSettingsRead:          request.NewRateLimitWithToken(requestLimitInterval, userSettingsReadReqRate, 1),
		userSettingsDelete:        request.NewRateLimitWithToken(requestLimitInterval, userSettingsDeleteReqRate, 1),
		// Account V1 endpoints
		getAccountFees:    request.NewRateLimitWithToken(requestLimitInterval, getAccountFeesReqRate, 1),
		getWithdrawalFees: request.NewRateLimitWithToken(requestLimitInterval, getWithdrawalFeesReqRate, 1),
		getAccountSummary: request.NewRateLimitWithToken(requestLimitInterval, getAccountSummaryReqRate, 1),
		newDepositAddress: request.NewRateLimitWithToken(requestLimitInterval, newDepositAddressReqRate, 1),
		getKeyPermissions: request.NewRateLimitWithToken(requestLimitInterval, getKeyPermissionsReqRate, 1),
		getMarginInfo:     request.NewRateLimitWithToken(requestLimitInterval, getMarginInfoReqRate, 1),
		getAccountBalance: request.NewRateLimitWithToken(requestLimitInterval, getAccountBalanceReqRate, 1),
		walletTransfer:    request.NewRateLimitWithToken(requestLimitInterval, walletTransferReqRate, 1),
		withdrawV1:        request.NewRateLimitWithToken(requestLimitInterval, withdrawV1ReqRate, 1),
		orderV1:           request.NewRateLimitWithToken(requestLimitInterval, orderV1ReqRate, 1),
		orderMulti:        request.NewRateLimitWithToken(requestLimitInterval, orderMultiReqRate, 1),
		statsV1:           request.NewRateLimitWithToken(requestLimitInterval, statsV1ReqRate, 1),
		fundingbook:       request.NewRateLimitWithToken(requestLimitInterval, fundingbookReqRate, 1),
		lends:             request.NewRateLimitWithToken(requestLimitInterval, lendsReqRate, 1),
	}
}
