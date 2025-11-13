package bitmex

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitmex
type Exchange struct {
	exchange.Base
}

const (
	bitmexAPIVersion    = "v1"
	bitmexAPIURL        = "https://www.bitmex.com/api/v1"
	bitmexAPItestnetURL = "https://testnet.bitmex.com/api/v1"
	tradeBaseURL        = "https://www.bitmex.com/app/trade/"

	// Public endpoints
	bitmexEndpointAnnouncement              = "/announcement"
	bitmexEndpointAnnouncementUrgent        = "/announcement/urgent"
	bitmexEndpointOrderbookL2               = "/orderBook/L2"
	bitmexEndpointTrollbox                  = "/chat"
	bitmexEndpointTrollboxChannels          = "/chat/channels"
	bitmexEndpointTrollboxConnected         = "/chat/connected"
	bitmexEndpointFundingHistory            = "/funding?"
	bitmexEndpointInstruments               = "/instrument"
	bitmexEndpointActiveInstruments         = "/instrument/active"
	bitmexEndpointActiveAndIndexInstruments = "/instrument/activeAndIndices"
	bitmexEndpointActiveIntervals           = "/instrument/activeIntervals"
	bitmexEndpointCompositeIndex            = "/instrument/compositeIndex"
	bitmexEndpointIndices                   = "/instrument/indices"
	bitmexEndpointLiquidation               = "/liquidation"
	bitmexEndpointLeader                    = "/leaderboard"
	bitmexEndpointAlias                     = "/leaderboard/name"
	bitmexEndpointQuote                     = "/quote"
	bitmexEndpointQuoteBucketed             = "/quote/bucketed"
	bitmexEndpointSettlement                = "/settlement"
	bitmexEndpointStats                     = "/stats"
	bitmexEndpointStatsHistory              = "/stats/history"
	bitmexEndpointStatsSummary              = "/stats/historyUSD"
	bitmexEndpointTrade                     = "/trade"
	bitmexEndpointTradeBucketed             = "/trade/bucketed"
	bitmexEndpointUserCheckReferralCode     = "/user/checkReferralCode"
	bitmexEndpointUserMinWithdrawalFee      = "/user/minWithdrawalFee"

	// Authenticated endpoints
	bitmexEndpointAPIkeys               = "/apiKey"
	bitmexEndpointDisableAPIkey         = "/apiKey/disable" //nolint:gosec // false positive
	bitmexEndpointEnableAPIkey          = "/apiKey/enable"  //nolint:gosec // false positive
	bitmexEndpointTrollboxSend          = "/chat"
	bitmexEndpointExecution             = "/execution"
	bitmexEndpointExecutionTradeHistory = "/execution/tradeHistory"
	bitmexEndpointNotifications         = "/notification"
	bitmexEndpointOrder                 = "/order"
	bitmexEndpointCancelAllOrders       = "/order/all"
	bitmexEndpointBulk                  = "/order/bulk"
	bitmexEndpointCancelOrderAfter      = "/order/cancelAllAfter"
	bitmexEndpointPosition              = "/position"
	bitmexEndpointIsolatePosition       = "/position/isolate"
	bitmexEndpointLeveragePosition      = "/position/leverage"
	bitmexEndpointAdjustRiskLimit       = "/position/riskLimit"
	bitmexEndpointTransferMargin        = "/position/transferMargin"
	bitmexEndpointUser                  = "/user"
	bitmexEndpointUserAffiliate         = "/user/affiliateStatus"
	bitmexEndpointUserCancelWithdraw    = "/user/cancelWithdrawal"
	bitmexEndpointUserCommision         = "/user/commission"
	bitmexEndpointUserConfirmEmail      = "/user/confirmEmail"
	bitmexEndpointUserConfirmTFA        = "/user/confirmEnableTFA"
	bitmexEndpointUserDepositAddress    = "/user/depositAddress"
	bitmexEndpointUserDisableTFA        = "/user/disableTFA"
	bitmexEndpointUserLogout            = "/user/logout"
	bitmexEndpointUserLogoutAll         = "/user/logoutAll"
	bitmexEndpointUserMargin            = "/user/margin"
	bitmexEndpointUserPreferences       = "/user/preferences"
	bitmexEndpointUserRequestTFA        = "/user/requestEnableTFA"
	bitmexEndpointUserWallet            = "/user/wallet"
	bitmexEndpointUserWalletHistory     = "/user/walletHistory"
	bitmexEndpointUserWalletSummary     = "/user/walletSummary"
	bitmexEndpointUserRequestWithdraw   = "/user/requestWithdrawal"

	constSatoshiBTC = 1e-08
	countLimit      = uint32(1000)

	perpetualContractID         = "FFWCSX"
	spotID                      = "IFXXXP"
	futuresID                   = "FFCCSX"
	bitMEXBasketIndexID         = "MRBXXX"
	bitMEXPriceIndexID          = "MRCXXX"
	bitMEXLendingPremiumIndexID = "MRRXXX"
	bitMEXVolatilityIndexID     = "MRIXXX"
)

// GetAnnouncement returns the general announcements from Bitmex
func (e *Exchange) GetAnnouncement(ctx context.Context) ([]Announcement, error) {
	var announcement []Announcement

	return announcement, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointAnnouncement,
		nil,
		&announcement)
}

// GetUrgentAnnouncement returns an urgent announcement for your account
func (e *Exchange) GetUrgentAnnouncement(ctx context.Context) ([]Announcement, error) {
	var announcement []Announcement

	return announcement, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointAnnouncementUrgent,
		nil,
		&announcement)
}

// GetAPIKeys returns the APIkeys from bitmex
func (e *Exchange) GetAPIKeys(ctx context.Context) ([]APIKey, error) {
	var keys []APIKey

	return keys, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointAPIkeys,
		nil,
		&keys)
}

// RemoveAPIKey removes an Apikey from the bitmex trading engine
func (e *Exchange) RemoveAPIKey(ctx context.Context, params APIKeyParams) (bool, error) {
	var keyDeleted bool

	return keyDeleted, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointAPIkeys,
		&params,
		&keyDeleted)
}

// DisableAPIKey disables an Apikey from the bitmex trading engine
func (e *Exchange) DisableAPIKey(ctx context.Context, params APIKeyParams) (APIKey, error) {
	var keyInfo APIKey

	return keyInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointDisableAPIkey,
		&params,
		&keyInfo)
}

// EnableAPIKey enables an Apikey from the bitmex trading engine
func (e *Exchange) EnableAPIKey(ctx context.Context, params APIKeyParams) (APIKey, error) {
	var keyInfo APIKey

	return keyInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointEnableAPIkey,
		&params,
		&keyInfo)
}

// GetTrollboxMessages returns messages from the bitmex trollbox
func (e *Exchange) GetTrollboxMessages(ctx context.Context, params ChatGetParams) ([]Chat, error) {
	var messages []Chat

	return messages, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollbox, &params, &messages)
}

// SendTrollboxMessage sends a message to the bitmex trollbox
func (e *Exchange) SendTrollboxMessage(ctx context.Context, params ChatSendParams) ([]Chat, error) {
	var messages []Chat

	return messages, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointTrollboxSend,
		&params,
		&messages)
}

// GetTrollboxChannels the channels from the bitmex trollbox
func (e *Exchange) GetTrollboxChannels(ctx context.Context) ([]ChatChannel, error) {
	var channels []ChatChannel

	return channels, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollboxChannels,
		nil,
		&channels)
}

// GetTrollboxConnectedUsers the channels from the bitmex trollbox
func (e *Exchange) GetTrollboxConnectedUsers(ctx context.Context) (ConnectedUsers, error) {
	var users ConnectedUsers

	return users, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrollboxConnected, nil, &users)
}

// GetAccountExecutions returns all raw transactions, which includes order
// opening and cancellation, and order status changes. It can be quite noisy.
// More focused information is available at /execution/tradeHistory.
func (e *Exchange) GetAccountExecutions(ctx context.Context, params *GenericRequestParams) ([]Execution, error) {
	var executionList []Execution

	return executionList, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointExecution,
		params,
		&executionList)
}

// GetAccountExecutionTradeHistory returns all balance-affecting executions.
// This includes each trade, insurance charge, and settlement.
func (e *Exchange) GetAccountExecutionTradeHistory(ctx context.Context, params *GenericRequestParams) ([]Execution, error) {
	var tradeHistory []Execution

	return tradeHistory, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointExecutionTradeHistory,
		params,
		&tradeHistory)
}

// GetFullFundingHistory returns funding history
func (e *Exchange) GetFullFundingHistory(ctx context.Context, symbol, count, filter, columns, start string, reverse bool, startTime, endTime time.Time) ([]Funding, error) {
	var fundingHistory []Funding
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if count != "" {
		params.Set("count", count)
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	if columns != "" {
		params.Set("columns", columns)
	}
	if !startTime.IsZero() {
		params.Set("start", start)
	}
	params.Set("reverse", "true")
	if !reverse {
		params.Set("reverse", "false")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", startTime.Format(time.RFC3339))
		params.Set("endTime", endTime.Format(time.RFC3339))
	}
	return fundingHistory, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointFundingHistory+params.Encode(),
		nil,
		&fundingHistory)
}

// GetInstrument returns instrument data
func (e *Exchange) GetInstrument(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var instruments []Instrument

	return instruments, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointInstruments,
		params,
		&instruments)
}

// GetInstruments returns instrument data
func (e *Exchange) GetInstruments(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var instruments []Instrument

	return instruments, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointInstruments,
		params,
		&instruments)
}

// GetActiveInstruments returns active instruments
func (e *Exchange) GetActiveInstruments(ctx context.Context, params *GenericRequestParams) ([]Instrument, error) {
	var activeInstruments []Instrument

	return activeInstruments, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveInstruments,
		params,
		&activeInstruments)
}

// GetActiveAndIndexInstruments returns all active instruments and all indices
func (e *Exchange) GetActiveAndIndexInstruments(ctx context.Context) ([]Instrument, error) {
	var activeAndIndices []Instrument

	return activeAndIndices,
		e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveAndIndexInstruments,
			nil,
			&activeAndIndices)
}

// GetActiveIntervals returns funding history
func (e *Exchange) GetActiveIntervals(ctx context.Context) (InstrumentInterval, error) {
	var interval InstrumentInterval

	return interval, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointActiveIntervals,
		nil,
		&interval)
}

// GetCompositeIndex returns composite index
func (e *Exchange) GetCompositeIndex(ctx context.Context, symbol, count, filter, columns, start, reverse string, startTime, endTime time.Time) ([]IndexComposite, error) {
	var compositeIndices []IndexComposite
	params := url.Values{}
	params.Set("symbol", symbol)
	if count != "" {
		params.Set("count", count)
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	if columns != "" {
		params.Set("columns", columns)
	}
	if start != "" {
		params.Set("start", start)
	}
	if reverse != "" {
		params.Set("reverse", "true")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", startTime.Format(time.RFC3339))
		params.Set("endTime", endTime.Format(time.RFC3339))
	}
	return compositeIndices, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointCompositeIndex+"?"+params.Encode(),
		nil,
		&compositeIndices)
}

// GetIndices returns all price indices
func (e *Exchange) GetIndices(ctx context.Context) ([]Instrument, error) {
	var indices []Instrument

	return indices, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointIndices, nil, &indices)
}

// GetInsuranceFundHistory returns insurance fund history
func (e *Exchange) GetInsuranceFundHistory(ctx context.Context, params *GenericRequestParams) ([]Insurance, error) {
	var history []Insurance

	return history, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointIndices, params, &history)
}

// GetLeaderboard returns leaderboard information
func (e *Exchange) GetLeaderboard(ctx context.Context, params LeaderboardGetParams) ([]Leaderboard, error) {
	var leader []Leaderboard

	return leader, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointLeader, params, &leader)
}

// GetAliasOnLeaderboard returns your alias on the leaderboard
func (e *Exchange) GetAliasOnLeaderboard(ctx context.Context) (Alias, error) {
	var alias Alias

	return alias, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointAlias, nil, &alias)
}

// GetLiquidationOrders returns liquidation orders
func (e *Exchange) GetLiquidationOrders(ctx context.Context, params *GenericRequestParams) ([]Liquidation, error) {
	var orders []Liquidation

	return orders, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointLiquidation,
		params,
		&orders)
}

// GetCurrentNotifications returns your current notifications
func (e *Exchange) GetCurrentNotifications(ctx context.Context) ([]Notification, error) {
	var notifications []Notification

	return notifications, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointNotifications,
		nil,
		&notifications)
}

// GetOrders returns all the orders, open and closed
func (e *Exchange) GetOrders(ctx context.Context, params *OrdersRequest) ([]Order, error) {
	var orders []Order
	return orders, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointOrder,
		params,
		&orders)
}

// AmendOrder amends the quantity or price of an open order
func (e *Exchange) AmendOrder(ctx context.Context, params *OrderAmendParams) (Order, error) {
	var order Order

	return order, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointOrder,
		params,
		&order)
}

// CreateOrder creates a new order
func (e *Exchange) CreateOrder(ctx context.Context, params *OrderNewParams) (Order, error) {
	var orderInfo Order

	return orderInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointOrder,
		params,
		&orderInfo)
}

// CancelOrders cancels one or a batch of orders on the exchange and returns
// a cancelled order list
func (e *Exchange) CancelOrders(ctx context.Context, params *OrderCancelParams) ([]Order, error) {
	var cancelledOrders []Order

	return cancelledOrders, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointOrder,
		params,
		&cancelledOrders)
}

// CancelAllExistingOrders cancels all open orders on the exchange
func (e *Exchange) CancelAllExistingOrders(ctx context.Context, params OrderCancelAllParams) ([]Order, error) {
	var cancelledOrders []Order

	return cancelledOrders, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		bitmexEndpointCancelAllOrders,
		params,
		&cancelledOrders)
}

// AmendBulkOrders amends multiple orders for the same symbol
func (e *Exchange) AmendBulkOrders(ctx context.Context, params OrderAmendBulkParams) ([]Order, error) {
	var amendedOrders []Order

	return amendedOrders, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointBulk,
		params,
		&amendedOrders)
}

// CreateBulkOrders creates multiple orders for the same symbol
func (e *Exchange) CreateBulkOrders(ctx context.Context, params OrderNewBulkParams) ([]Order, error) {
	var orders []Order

	return orders, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointBulk,
		params,
		&orders)
}

// CancelAllOrdersAfterTime closes all positions after a certain time period
func (e *Exchange) CancelAllOrdersAfterTime(ctx context.Context, params OrderCancelAllAfterParams) ([]Order, error) {
	var cancelledOrder []Order

	return cancelledOrder, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointCancelOrderAfter,
		params,
		&cancelledOrder)
}

// ClosePosition closes a position WARNING deprecated use /order endpoint
func (e *Exchange) ClosePosition(ctx context.Context, params OrderClosePositionParams) ([]Order, error) {
	var closedPositions []Order

	return closedPositions, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointOrder,
		params,
		&closedPositions)
}

// GetOrderbook returns layer two orderbook data
func (e *Exchange) GetOrderbook(ctx context.Context, params OrderBookGetL2Params) ([]OrderBookL2, error) {
	var orderBooks []OrderBookL2

	return orderBooks, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointOrderbookL2,
		params,
		&orderBooks)
}

// GetPositions returns positions
func (e *Exchange) GetPositions(ctx context.Context, params PositionGetParams) ([]Position, error) {
	var positions []Position

	return positions, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointPosition,
		params,
		&positions)
}

// IsolatePosition enables isolated margin or cross margin per-position
func (e *Exchange) IsolatePosition(ctx context.Context, params PositionIsolateMarginParams) (Position, error) {
	var position Position

	return position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointIsolatePosition,
		params,
		&position)
}

// LeveragePosition chooses leverage for a position
func (e *Exchange) LeveragePosition(ctx context.Context, params PositionUpdateLeverageParams) (Position, error) {
	var position Position

	return position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointLeveragePosition,
		params,
		&position)
}

// UpdateRiskLimit updates risk limit on a position
func (e *Exchange) UpdateRiskLimit(ctx context.Context, params PositionUpdateRiskLimitParams) (Position, error) {
	var position Position

	return position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointAdjustRiskLimit,
		params,
		&position)
}

// TransferMargin transfers equity in or out of a position
func (e *Exchange) TransferMargin(ctx context.Context, params PositionTransferIsolatedMarginParams) (Position, error) {
	var position Position

	return position, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointTransferMargin,
		params,
		&position)
}

// GetQuotes returns quotations
func (e *Exchange) GetQuotes(ctx context.Context, params *GenericRequestParams) ([]Quote, error) {
	var quotations []Quote

	return quotations, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointQuote,
		params,
		&quotations)
}

// GetQuotesByBuckets returns previous quotes in time buckets
func (e *Exchange) GetQuotesByBuckets(ctx context.Context, params *QuoteGetBucketedParams) ([]Quote, error) {
	var quotations []Quote

	return quotations, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointQuoteBucketed,
		params,
		&quotations)
}

// GetSettlementHistory returns settlement history
func (e *Exchange) GetSettlementHistory(ctx context.Context, params *GenericRequestParams) ([]Settlement, error) {
	var history []Settlement

	return history, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointSettlement,
		params,
		&history)
}

// GetStats returns exchange wide per series turnover and volume statistics
func (e *Exchange) GetStats(ctx context.Context) ([]Stats, error) {
	var stats []Stats

	return stats, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStats, nil, &stats)
}

// GetStatsHistorical historic stats
func (e *Exchange) GetStatsHistorical(ctx context.Context) ([]StatsHistory, error) {
	var history []StatsHistory

	return history, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStatsHistory, nil, &history)
}

// GetStatSummary returns the stats summary in USD terms
func (e *Exchange) GetStatSummary(ctx context.Context) ([]StatsUSD, error) {
	var summary []StatsUSD

	return summary, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointStatsSummary, nil, &summary)
}

// GetTrade returns executed trades on the desk
func (e *Exchange) GetTrade(ctx context.Context, params *GenericRequestParams) ([]Trade, error) {
	var trade []Trade

	return trade, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTrade, params, &trade)
}

// GetPreviousTrades previous trade history in time buckets
func (e *Exchange) GetPreviousTrades(ctx context.Context, params *TradeGetBucketedParams) ([]Trade, error) {
	var trade []Trade

	return trade, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointTradeBucketed,
		params,
		&trade)
}

// GetUserInfo returns your user information
func (e *Exchange) GetUserInfo(ctx context.Context) (User, error) {
	var userInfo User

	return userInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUser,
		nil,
		&userInfo)
}

// UpdateUserInfo updates user information
func (e *Exchange) UpdateUserInfo(ctx context.Context, params *UserUpdateParams) (User, error) {
	var userInfo User

	return userInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPut,
		bitmexEndpointUser,
		params,
		&userInfo)
}

// GetAffiliateStatus returns your affiliate status
func (e *Exchange) GetAffiliateStatus(ctx context.Context) (AffiliateStatus, error) {
	var status AffiliateStatus

	return status, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserAffiliate,
		nil,
		&status)
}

// CancelWithdraw cancels a current withdrawal
func (e *Exchange) CancelWithdraw(ctx context.Context, token string) (TransactionInfo, error) {
	var info TransactionInfo

	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserCancelWithdraw,
		UserTokenParams{Token: token},
		&info)
}

// CheckReferalCode checks a code, will return a percentage eg 0.1 for 10% or
// if err a 404
func (e *Exchange) CheckReferalCode(ctx context.Context, referralCode string) (float64, error) {
	var percentage float64

	return percentage, e.SendHTTPRequest(ctx, exchange.RestSpot, bitmexEndpointUserCheckReferralCode,
		UserCheckReferralCodeParams{ReferralCode: referralCode},
		&percentage)
}

// GetUserCommision returns your account's commission status.
func (e *Exchange) GetUserCommision(ctx context.Context) (UserCommission, error) {
	var commissionInfo UserCommission

	return commissionInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserCommision,
		nil,
		&commissionInfo)
}

// ConfirmEmail confirms email address with a token
func (e *Exchange) ConfirmEmail(ctx context.Context, token string) (ConfirmEmail, error) {
	var confirmation ConfirmEmail

	return confirmation, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserConfirmEmail,
		UserTokenParams{Token: token},
		&confirmation)
}

// ConfirmTwoFactorAuth confirms 2FA for this account
func (e *Exchange) ConfirmTwoFactorAuth(ctx context.Context, token, typ string) (bool, error) {
	var working bool

	return working, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserConfirmTFA,
		UserConfirmTFAParams{Token: token, Type: typ},
		&working)
}

// ConfirmWithdrawal confirms a withdrawal
func (e *Exchange) ConfirmWithdrawal(ctx context.Context, token string) (TransactionInfo, error) {
	var info TransactionInfo

	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserCancelWithdraw,
		UserTokenParams{Token: token},
		&info)
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (e *Exchange) GetCryptoDepositAddress(ctx context.Context, cryptoCurrency string) (string, error) {
	var address string
	if !strings.EqualFold(cryptoCurrency, currency.XBT.String()) {
		return "", fmt.Errorf("%v %w only bitcoin", cryptoCurrency, currency.ErrCurrencyNotSupported)
	}

	return address, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserDepositAddress,
		UserCurrencyParams{Currency: "XBt"},
		&address)
}

// DisableTFA dsiables 2 factor authentication for your account
func (e *Exchange) DisableTFA(ctx context.Context, token, typ string) (bool, error) {
	var disabled bool

	return disabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserDisableTFA,
		UserConfirmTFAParams{Token: token, Type: typ},
		&disabled)
}

// UserLogOut logs you out of BitMEX
func (e *Exchange) UserLogOut(ctx context.Context) error {
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserLogout,
		nil,
		nil)
}

// UserLogOutAll logs you out of all systems for BitMEX
func (e *Exchange) UserLogOutAll(ctx context.Context) (int64, error) {
	var status int64

	return status, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserLogoutAll,
		nil,
		&status)
}

// GetUserMargin returns user margin information
func (e *Exchange) GetUserMargin(ctx context.Context, ccy string) (UserMargin, error) {
	var info UserMargin
	params := UserCurrencyParams{Currency: ccy}
	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bitmexEndpointUserMargin, params, &info)
}

// GetAllUserMargin returns user margin information
func (e *Exchange) GetAllUserMargin(ctx context.Context) ([]UserMargin, error) {
	var info []UserMargin

	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		bitmexEndpointUserMargin,
		UserCurrencyParams{Currency: "all"},
		&info)
}

// GetMinimumWithdrawalFee returns minimum withdrawal fee information
func (e *Exchange) GetMinimumWithdrawalFee(ctx context.Context, ccy string) (MinWithdrawalFee, error) {
	var fee MinWithdrawalFee
	params := UserCurrencyParams{Currency: ccy}
	return fee, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bitmexEndpointUserMinWithdrawalFee, params, &fee)
}

// GetUserPreferences returns user preferences
func (e *Exchange) GetUserPreferences(ctx context.Context, params UserPreferencesParams) (User, error) {
	var userInfo User

	return userInfo, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserPreferences,
		params,
		&userInfo)
}

// EnableTFA enables 2 factor authentication
func (e *Exchange) EnableTFA(ctx context.Context, typ string) (bool, error) {
	var enabled bool

	return enabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserRequestTFA,
		UserConfirmTFAParams{Type: typ},
		&enabled)
}

// UserRequestWithdrawal This will send a confirmation email to the email
// address on record, unless requested via an API Key with the withdraw
// permission.
func (e *Exchange) UserRequestWithdrawal(ctx context.Context, params UserRequestWithdrawalParams) (TransactionInfo, error) {
	var info TransactionInfo

	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		bitmexEndpointUserRequestWithdraw,
		params,
		&info)
}

// GetWalletInfo returns user wallet information
func (e *Exchange) GetWalletInfo(ctx context.Context, ccy string) (WalletInfo, error) {
	var info WalletInfo
	params := UserCurrencyParams{Currency: ccy}
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bitmexEndpointUserWallet, params, &info); err != nil {
		return info, err
	}

	// Bitmex has an "interesting" of dealing with currencies,
	// for instance XBt is actually BTC but in Satoshi units,
	// for sanity purposes apply here a conversion to normalize
	// this
	// avoid a copy here since this is a big struct
	normalizeWalletInfo(&info)

	return info, nil
}

// GetWalletHistory returns user wallet history transaction data
func (e *Exchange) GetWalletHistory(ctx context.Context, ccy string) ([]TransactionInfo, error) {
	var info []TransactionInfo
	params := UserCurrencyParams{Currency: ccy}
	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bitmexEndpointUserWalletHistory, params, &info)
}

// GetWalletSummary returns user wallet summary
func (e *Exchange) GetWalletSummary(ctx context.Context, ccy string) ([]TransactionInfo, error) {
	var info []TransactionInfo
	params := UserCurrencyParams{Currency: ccy}
	return info, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bitmexEndpointUserWalletSummary, params, &info)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, params Parameter, result any) error {
	var respCheck any
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = endpoint + path
	if params != nil && !params.IsNil() {
		path, err = params.ToURLVals(path)
		if err != nil {
			return err
		}
	}

	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   path,
		Result:                 &respCheck,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, request.UnAuth, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}

	return e.CaptureError(respCheck, result)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to bitmex
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, verb, path string, params Parameter, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	var respCheck any
	newRequest := func() (*request.Item, error) {
		ts := strconv.FormatInt(time.Now().Add(time.Second*10).UnixMilli(), 10)

		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["api-expires"] = ts
		headers["api-key"] = creds.Key

		var payload string
		if params != nil {
			if err := params.VerifyData(); err != nil {
				return nil, err
			}
			data, err := json.Marshal(params)
			if err != nil {
				return nil, err
			}
			payload = string(data)
		}

		hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(verb+"/api/v1"+path+ts+payload), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers["api-signature"] = hex.EncodeToString(hmac)

		return &request.Item{
			Method:                 verb,
			Path:                   endpoint + path,
			Headers:                headers,
			Body:                   strings.NewReader(payload),
			Result:                 &respCheck,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}
	err = e.SendPayload(ctx, request.Auth, newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	return e.CaptureError(respCheck, result)
}

// CaptureError little hack that captures an error
func (e *Exchange) CaptureError(resp, reType any) error {
	var Error RequestError

	marshalled, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(marshalled, &Error)
	if err == nil && Error.Error.Name != "" {
		return fmt.Errorf("bitmex error %s: %s",
			Error.Error.Name,
			Error.Error.Message)
	}

	return json.Unmarshal(marshalled, reType)
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	var err error
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, err
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.000750 * price * amount
}

// calculateTradingFee returns the fee for trading any currency on Bitmex
func calculateTradingFee(purchasePrice, amount float64, isMaker bool) float64 {
	fee := 0.000750
	if isMaker {
		fee -= 0.000250
	}

	return fee * purchasePrice * amount
}

var xbtCurr = currency.NewCode("XBt")

// normalizeWalletInfo converts any non-standard currencies (eg. XBt -> BTC)
func normalizeWalletInfo(w *WalletInfo) {
	if !w.Currency.Equal(xbtCurr) {
		return
	}

	w.Currency = currency.BTC
	w.Amount *= constSatoshiBTC
}
