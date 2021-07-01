package bitfinex

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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
	statusReqRate             = 15 // This is not specified just inputed WCS
	liquidReqRate             = 15 // This is not specified just inputed WCS
	leaderBoardReqRate        = 90
	marketAveragePriceReqRate = 20
	fxReqRate                 = 20

	// Bitfinex rate limits - Authenticated
	// Wallets -
	accountWalletBalanceReqRate = 45
	accountWalletHistoryReqRate = 45
	// Orders -
	retrieveOrderReqRate  = 45
	submitOrderReqRate    = 45 // This is not specified just inputed above
	updateOrderReqRate    = 45 // This is not specified just inputed above
	cancelOrderReqRate    = 45 // This is not specified just inputed above
	orderBatchReqRate     = 45 // This is not specified just inputed above
	cancelBatchReqRate    = 45 // This is not specified just inputed above
	orderHistoryReqRate   = 45
	getOrderTradesReqRate = 45
	getTradesReqRate      = 45
	getLedgersReqRate     = 45
	// Positions -
	getAccountMarginInfoReqRate       = 45
	getActivePositionsReqRate         = 45
	claimPositionReqRate              = 45 // This is not specified just inputed above
	getPositionHistoryReqRate         = 45
	getPositionAuditReqRate           = 45
	updateCollateralOnPositionReqRate = 45 // This is not specified just inputed above
	// Margin funding -
	getMarginInfoRate               = 90
	getActiveFundingOffersReqRate   = 45
	submitFundingOfferReqRate       = 45 // This is not specified just inputed above
	cancelFundingOfferReqRate       = 45
	cancelAllFundingOfferReqRate    = 45 // This is not specified just inputed above
	closeFundingReqRate             = 45 // This is not specified just inputed above
	fundingAutoRenewReqRate         = 45 // This is not specified just inputed above
	keepFundingReqRate              = 45 // This is not specified just inputed above
	getOffersHistoryReqRate         = 45
	getFundingLoansReqRate          = 45
	getFundingLoanHistoryReqRate    = 45
	getFundingCreditsReqRate        = 45
	getFundingCreditsHistoryReqRate = 45
	getFundingTradesReqRate         = 45
	getFundingInfoReqRate           = 45
	// Account actions
	getUserInfoReqRate               = 45
	transferBetweenWalletsReqRate    = 45 // This is not specified just inputed above
	getDepositAddressReqRate         = 45 // This is not specified just inputed above
	withdrawalReqRate                = 45 // This is not specified just inputed above
	getMovementsReqRate              = 45
	getAlertListReqRate              = 45
	setPriceAlertReqRate             = 45 // This is not specified just inputed above
	deletePriceAlertReqRate          = 45 // This is not specified just inputed above
	getBalanceForOrdersOffersReqRate = 30
	userSettingsWriteReqRate         = 45 // This is not specified just inputed general count
	userSettingsReadReqRate          = 45
	userSettingsDeleteReqRate        = 45 // This is not specified just inputed above
	// Account V1 endpoints
	getAccountFeesReqRate    = 5
	getWithdrawalFeesReqRate = 5
	getAccountSummaryReqRate = 5 // This is not specified just inputed above
	newDepositAddressReqRate = 5 // This is not specified just inputed above
	getKeyPermissionsReqRate = 5 // This is not specified just inputed above
	getMarginInfoReqRate     = 5 // This is not specified just inputed above
	getAccountBalanceReqRate = 10
	walletTransferReqRate    = 10 // This is not specified just inputed above
	withdrawV1ReqRate        = 10 // This is not specified just inputed above
	orderV1ReqRate           = 10 // This is not specified just inputed above
	orderMultiReqRate        = 10 // This is not specified just inputed above
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

// RateLimit implements the rate.Limiter interface
type RateLimit struct {
	PlatformStatus       *rate.Limiter
	TickerBatch          *rate.Limiter
	Ticker               *rate.Limiter
	Trade                *rate.Limiter
	Orderbook            *rate.Limiter
	Stats                *rate.Limiter
	Candle               *rate.Limiter
	Configs              *rate.Limiter
	Status               *rate.Limiter
	Liquid               *rate.Limiter
	LeaderBoard          *rate.Limiter
	MarketAveragePrice   *rate.Limiter
	Fx                   *rate.Limiter
	AccountWalletBalance *rate.Limiter
	AccountWalletHistory *rate.Limiter
	// Orders -
	RetrieveOrder  *rate.Limiter
	SubmitOrder    *rate.Limiter
	UpdateOrder    *rate.Limiter
	CancelOrder    *rate.Limiter
	OrderBatch     *rate.Limiter
	CancelBatch    *rate.Limiter
	OrderHistory   *rate.Limiter
	GetOrderTrades *rate.Limiter
	GetTrades      *rate.Limiter
	GetLedgers     *rate.Limiter
	// Positions -
	GetAccountMarginInfo       *rate.Limiter
	GetActivePositions         *rate.Limiter
	ClaimPosition              *rate.Limiter
	GetPositionHistory         *rate.Limiter
	GetPositionAudit           *rate.Limiter
	UpdateCollateralOnPosition *rate.Limiter
	// Margin funding -
	GetActiveFundingOffers   *rate.Limiter
	SubmitFundingOffer       *rate.Limiter
	CancelFundingOffer       *rate.Limiter
	CancelAllFundingOffer    *rate.Limiter
	CloseFunding             *rate.Limiter
	FundingAutoRenew         *rate.Limiter
	KeepFunding              *rate.Limiter
	GetOffersHistory         *rate.Limiter
	GetFundingLoans          *rate.Limiter
	GetFundingLoanHistory    *rate.Limiter
	GetFundingCredits        *rate.Limiter
	GetFundingCreditsHistory *rate.Limiter
	GetFundingTrades         *rate.Limiter
	GetFundingInfo           *rate.Limiter
	// Account actions
	GetUserInfo               *rate.Limiter
	TransferBetweenWallets    *rate.Limiter
	GetDepositAddress         *rate.Limiter
	Withdrawal                *rate.Limiter
	GetMovements              *rate.Limiter
	GetAlertList              *rate.Limiter
	SetPriceAlert             *rate.Limiter
	DeletePriceAlert          *rate.Limiter
	GetBalanceForOrdersOffers *rate.Limiter
	UserSettingsWrite         *rate.Limiter
	UserSettingsRead          *rate.Limiter
	UserSettingsDelete        *rate.Limiter
	// Account V1 endpoints
	GetAccountFees    *rate.Limiter
	GetWithdrawalFees *rate.Limiter
	GetAccountSummary *rate.Limiter
	NewDepositAddress *rate.Limiter
	GetKeyPermissions *rate.Limiter
	GetMarginInfo     *rate.Limiter
	GetAccountBalance *rate.Limiter
	WalletTransfer    *rate.Limiter
	WithdrawV1        *rate.Limiter
	OrderV1           *rate.Limiter
	OrderMulti        *rate.Limiter
	StatsV1           *rate.Limiter
	Fundingbook       *rate.Limiter
	Lends             *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case platformStatus:
		time.Sleep(r.PlatformStatus.Reserve().Delay())
	case tickerBatch:
		time.Sleep(r.TickerBatch.Reserve().Delay())
	case tickerFunction:
		time.Sleep(r.Ticker.Reserve().Delay())
	case tradeRateLimit:
		time.Sleep(r.Trade.Reserve().Delay())
	case orderbookFunction:
		time.Sleep(r.Orderbook.Reserve().Delay())
	case stats:
		time.Sleep(r.Stats.Reserve().Delay())
	case candle:
		time.Sleep(r.Candle.Reserve().Delay())
	case configs:
		time.Sleep(r.Configs.Reserve().Delay())
	case status:
		time.Sleep(r.Stats.Reserve().Delay())
	case liquid:
		time.Sleep(r.Liquid.Reserve().Delay())
	case leaderBoard:
		time.Sleep(r.LeaderBoard.Reserve().Delay())
	case marketAveragePrice:
		time.Sleep(r.MarketAveragePrice.Reserve().Delay())
	case fx:
		time.Sleep(r.Fx.Reserve().Delay())
	case accountWalletBalance:
		time.Sleep(r.AccountWalletBalance.Reserve().Delay())
	case accountWalletHistory:
		time.Sleep(r.AccountWalletHistory.Reserve().Delay())
	case retrieveOrder:
		time.Sleep(r.RetrieveOrder.Reserve().Delay())
	case submitOrder:
		time.Sleep(r.SubmitOrder.Reserve().Delay())
	case updateOrder:
		time.Sleep(r.UpdateOrder.Reserve().Delay())
	case cancelOrder:
		time.Sleep(r.CancelOrder.Reserve().Delay())
	case orderBatch:
		time.Sleep(r.OrderBatch.Reserve().Delay())
	case cancelBatch:
		time.Sleep(r.CancelBatch.Reserve().Delay())
	case orderHistory:
		time.Sleep(r.OrderHistory.Reserve().Delay())
	case getOrderTrades:
		time.Sleep(r.GetOrderTrades.Reserve().Delay())
	case getTrades:
		time.Sleep(r.GetTrades.Reserve().Delay())
	case getLedgers:
		time.Sleep(r.GetLedgers.Reserve().Delay())
	case getAccountMarginInfo:
		time.Sleep(r.GetAccountMarginInfo.Reserve().Delay())
	case getActivePositions:
		time.Sleep(r.GetActivePositions.Reserve().Delay())
	case claimPosition:
		time.Sleep(r.ClaimPosition.Reserve().Delay())
	case getPositionHistory:
		time.Sleep(r.GetPositionHistory.Reserve().Delay())
	case getPositionAudit:
		time.Sleep(r.GetPositionAudit.Reserve().Delay())
	case updateCollateralOnPosition:
		time.Sleep(r.UpdateCollateralOnPosition.Reserve().Delay())
	case getActiveFundingOffers:
		time.Sleep(r.GetActiveFundingOffers.Reserve().Delay())
	case submitFundingOffer:
		time.Sleep(r.SubmitFundingOffer.Reserve().Delay())
	case cancelFundingOffer:
		time.Sleep(r.CancelFundingOffer.Reserve().Delay())
	case cancelAllFundingOffer:
		time.Sleep(r.CancelAllFundingOffer.Reserve().Delay())
	case closeFunding:
		time.Sleep(r.CloseFunding.Reserve().Delay())
	case fundingAutoRenew:
		time.Sleep(r.FundingAutoRenew.Reserve().Delay())
	case keepFunding:
		time.Sleep(r.KeepFunding.Reserve().Delay())
	case getOffersHistory:
		time.Sleep(r.GetOffersHistory.Reserve().Delay())
	case getFundingLoans:
		time.Sleep(r.GetFundingLoans.Reserve().Delay())
	case getFundingLoanHistory:
		time.Sleep(r.GetFundingLoanHistory.Reserve().Delay())
	case getFundingCredits:
		time.Sleep(r.GetFundingCredits.Reserve().Delay())
	case getFundingCreditsHistory:
		time.Sleep(r.GetFundingCreditsHistory.Reserve().Delay())
	case getFundingTrades:
		time.Sleep(r.GetFundingTrades.Reserve().Delay())
	case getFundingInfo:
		time.Sleep(r.GetFundingInfo.Reserve().Delay())
	case getUserInfo:
		time.Sleep(r.GetUserInfo.Reserve().Delay())
	case transferBetweenWallets:
		time.Sleep(r.TransferBetweenWallets.Reserve().Delay())
	case getDepositAddress:
		time.Sleep(r.GetDepositAddress.Reserve().Delay())
	case withdrawal:
		time.Sleep(r.Withdrawal.Reserve().Delay())
	case getMovements:
		time.Sleep(r.GetMovements.Reserve().Delay())
	case getAlertList:
		time.Sleep(r.GetAlertList.Reserve().Delay())
	case setPriceAlert:
		time.Sleep(r.SetPriceAlert.Reserve().Delay())
	case deletePriceAlert:
		time.Sleep(r.DeletePriceAlert.Reserve().Delay())
	case getBalanceForOrdersOffers:
		time.Sleep(r.GetBalanceForOrdersOffers.Reserve().Delay())
	case userSettingsWrite:
		time.Sleep(r.UserSettingsWrite.Reserve().Delay())
	case userSettingsRead:
		time.Sleep(r.UserSettingsRead.Reserve().Delay())
	case userSettingsDelete:
		time.Sleep(r.UserSettingsDelete.Reserve().Delay())

		//  Bitfinex V1 API
	case getAccountFees:
		time.Sleep(r.GetAccountFees.Reserve().Delay())
	case getWithdrawalFees:
		time.Sleep(r.GetWithdrawalFees.Reserve().Delay())
	case getAccountSummary:
		time.Sleep(r.GetAccountSummary.Reserve().Delay())
	case newDepositAddress:
		time.Sleep(r.NewDepositAddress.Reserve().Delay())
	case getKeyPermissions:
		time.Sleep(r.GetKeyPermissions.Reserve().Delay())
	case getMarginInfo:
		time.Sleep(r.GetMarginInfo.Reserve().Delay())
	case getAccountBalance:
		time.Sleep(r.GetAccountBalance.Reserve().Delay())
	case walletTransfer:
		time.Sleep(r.WalletTransfer.Reserve().Delay())
	case withdrawV1:
		time.Sleep(r.WithdrawV1.Reserve().Delay())
	case orderV1:
		time.Sleep(r.OrderV1.Reserve().Delay())
	case orderMulti:
		time.Sleep(r.OrderMulti.Reserve().Delay())
	case statsV1:
		time.Sleep(r.Stats.Reserve().Delay())
	case fundingbook:
		time.Sleep(r.Fundingbook.Reserve().Delay())
	case lends:
		time.Sleep(r.Lends.Reserve().Delay())
	default:
		return errors.New("endpoint rate limit functionality not found")
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		PlatformStatus:       request.NewRateLimit(requestLimitInterval, platformStatusReqRate),
		TickerBatch:          request.NewRateLimit(requestLimitInterval, tickerBatchReqRate),
		Ticker:               request.NewRateLimit(requestLimitInterval, tickerReqRate),
		Trade:                request.NewRateLimit(requestLimitInterval, tradeReqRate),
		Orderbook:            request.NewRateLimit(requestLimitInterval, orderbookReqRate),
		Stats:                request.NewRateLimit(requestLimitInterval, statsReqRate),
		Candle:               request.NewRateLimit(requestLimitInterval, candleReqRate),
		Configs:              request.NewRateLimit(requestLimitInterval, configsReqRate),
		Status:               request.NewRateLimit(requestLimitInterval, statusReqRate),
		Liquid:               request.NewRateLimit(requestLimitInterval, liquidReqRate),
		LeaderBoard:          request.NewRateLimit(requestLimitInterval, leaderBoardReqRate),
		MarketAveragePrice:   request.NewRateLimit(requestLimitInterval, marketAveragePriceReqRate),
		Fx:                   request.NewRateLimit(requestLimitInterval, fxReqRate),
		AccountWalletBalance: request.NewRateLimit(requestLimitInterval, accountWalletBalanceReqRate),
		AccountWalletHistory: request.NewRateLimit(requestLimitInterval, accountWalletHistoryReqRate),
		// Orders -
		RetrieveOrder:  request.NewRateLimit(requestLimitInterval, retrieveOrderReqRate),
		SubmitOrder:    request.NewRateLimit(requestLimitInterval, submitOrderReqRate),
		UpdateOrder:    request.NewRateLimit(requestLimitInterval, updateOrderReqRate),
		CancelOrder:    request.NewRateLimit(requestLimitInterval, cancelOrderReqRate),
		OrderBatch:     request.NewRateLimit(requestLimitInterval, orderBatchReqRate),
		CancelBatch:    request.NewRateLimit(requestLimitInterval, cancelBatchReqRate),
		OrderHistory:   request.NewRateLimit(requestLimitInterval, orderHistoryReqRate),
		GetOrderTrades: request.NewRateLimit(requestLimitInterval, getOrderTradesReqRate),
		GetTrades:      request.NewRateLimit(requestLimitInterval, getTradesReqRate),
		GetLedgers:     request.NewRateLimit(requestLimitInterval, getLedgersReqRate),
		// Positions -
		GetAccountMarginInfo:       request.NewRateLimit(requestLimitInterval, getAccountMarginInfoReqRate),
		GetActivePositions:         request.NewRateLimit(requestLimitInterval, getActivePositionsReqRate),
		ClaimPosition:              request.NewRateLimit(requestLimitInterval, claimPositionReqRate),
		GetPositionHistory:         request.NewRateLimit(requestLimitInterval, getPositionAuditReqRate),
		GetPositionAudit:           request.NewRateLimit(requestLimitInterval, getPositionAuditReqRate),
		UpdateCollateralOnPosition: request.NewRateLimit(requestLimitInterval, updateCollateralOnPositionReqRate),
		// Margin funding -
		GetActiveFundingOffers:   request.NewRateLimit(requestLimitInterval, getActiveFundingOffersReqRate),
		SubmitFundingOffer:       request.NewRateLimit(requestLimitInterval, submitFundingOfferReqRate),
		CancelFundingOffer:       request.NewRateLimit(requestLimitInterval, cancelFundingOfferReqRate),
		CancelAllFundingOffer:    request.NewRateLimit(requestLimitInterval, cancelAllFundingOfferReqRate),
		CloseFunding:             request.NewRateLimit(requestLimitInterval, closeFundingReqRate),
		FundingAutoRenew:         request.NewRateLimit(requestLimitInterval, fundingAutoRenewReqRate),
		KeepFunding:              request.NewRateLimit(requestLimitInterval, keepFundingReqRate),
		GetOffersHistory:         request.NewRateLimit(requestLimitInterval, getOffersHistoryReqRate),
		GetFundingLoans:          request.NewRateLimit(requestLimitInterval, getOffersHistoryReqRate),
		GetFundingLoanHistory:    request.NewRateLimit(requestLimitInterval, getFundingLoanHistoryReqRate),
		GetFundingCredits:        request.NewRateLimit(requestLimitInterval, getFundingCreditsReqRate),
		GetFundingCreditsHistory: request.NewRateLimit(requestLimitInterval, getFundingCreditsHistoryReqRate),
		GetFundingTrades:         request.NewRateLimit(requestLimitInterval, getFundingTradesReqRate),
		GetFundingInfo:           request.NewRateLimit(requestLimitInterval, getFundingInfoReqRate),
		// Account actions
		GetUserInfo:               request.NewRateLimit(requestLimitInterval, getUserInfoReqRate),
		TransferBetweenWallets:    request.NewRateLimit(requestLimitInterval, transferBetweenWalletsReqRate),
		GetDepositAddress:         request.NewRateLimit(requestLimitInterval, getDepositAddressReqRate),
		Withdrawal:                request.NewRateLimit(requestLimitInterval, withdrawalReqRate),
		GetMovements:              request.NewRateLimit(requestLimitInterval, getMovementsReqRate),
		GetAlertList:              request.NewRateLimit(requestLimitInterval, getAlertListReqRate),
		SetPriceAlert:             request.NewRateLimit(requestLimitInterval, setPriceAlertReqRate),
		DeletePriceAlert:          request.NewRateLimit(requestLimitInterval, deletePriceAlertReqRate),
		GetBalanceForOrdersOffers: request.NewRateLimit(requestLimitInterval, getBalanceForOrdersOffersReqRate),
		UserSettingsWrite:         request.NewRateLimit(requestLimitInterval, userSettingsWriteReqRate),
		UserSettingsRead:          request.NewRateLimit(requestLimitInterval, userSettingsReadReqRate),
		UserSettingsDelete:        request.NewRateLimit(requestLimitInterval, userSettingsDeleteReqRate),
		// Account V1 endpoints
		GetAccountFees:    request.NewRateLimit(requestLimitInterval, getAccountFeesReqRate),
		GetWithdrawalFees: request.NewRateLimit(requestLimitInterval, getWithdrawalFeesReqRate),
		GetAccountSummary: request.NewRateLimit(requestLimitInterval, getAccountSummaryReqRate),
		NewDepositAddress: request.NewRateLimit(requestLimitInterval, newDepositAddressReqRate),
		GetKeyPermissions: request.NewRateLimit(requestLimitInterval, getKeyPermissionsReqRate),
		GetMarginInfo:     request.NewRateLimit(requestLimitInterval, getMarginInfoReqRate),
		GetAccountBalance: request.NewRateLimit(requestLimitInterval, getAccountBalanceReqRate),
		WalletTransfer:    request.NewRateLimit(requestLimitInterval, walletTransferReqRate),
		WithdrawV1:        request.NewRateLimit(requestLimitInterval, withdrawV1ReqRate),
		OrderV1:           request.NewRateLimit(requestLimitInterval, orderV1ReqRate),
		OrderMulti:        request.NewRateLimit(requestLimitInterval, orderMultiReqRate),
		StatsV1:           request.NewRateLimit(requestLimitInterval, statsV1ReqRate),
		Fundingbook:       request.NewRateLimit(requestLimitInterval, fundingbookReqRate),
		Lends:             request.NewRateLimit(requestLimitInterval, lendsReqRate),
	}
}
