package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// The OMS Toolkit endpoints, formerly Binance Link "Link and Trade", manage the customer IDs and rebates of users a
// partner refers; spot endpoints identify the partner by API agent code and USDⓈ-M futures endpoints by broker ID

var (
	errCustomerIDRequired          = errors.New("customer ID is required")
	errAPIAgentCodeRequired        = errors.New("API agent code is required")
	errCustomerIDAndEmailExclusive = errors.New("customer ID and email cannot both be set")
)

// GetSpotUsersCustomisedID calls Get User’s Customize Id (Spot) (USER_DATA), returning the customer ID the account set
// with CustomiseSpotOwnClientID
func (e *Exchange) GetSpotUsersCustomisedID(ctx context.Context, apiAgentCode string) (*SpotReferralCustomerIDResponse, error) {
	if apiAgentCode == "" {
		return nil, errAPIAgentCodeRequired
	}
	params := url.Values{}
	params.Set("apiAgentCode", apiAgentCode)
	var resp *SpotReferralCustomerIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apiReferral/userCustomization", params, sapiDefaultRate, &resp)
}

// CustomiseSpotOwnClientID calls Customize Id For Spot Client (USER_DATA), letting a referred user set the unique
// customer ID their partner identifies them by
func (e *Exchange) CustomiseSpotOwnClientID(ctx context.Context, customerID, apiAgentCode string) (*SpotReferralCustomerIDResponse, error) {
	if customerID == "" {
		return nil, errCustomerIDRequired
	}
	if apiAgentCode == "" {
		return nil, errAPIAgentCodeRequired
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("apiAgentCode", apiAgentCode)
	var resp *SpotReferralCustomerIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/apiReferral/userCustomization", params, sapiDefaultRate, &resp)
}

// GetSpotClientEmailCustomisedID calls Get Client Email Customized Id (USER_DATA), filtering by customer ID or email,
// not both
func (e *Exchange) GetSpotClientEmailCustomisedID(ctx context.Context, customerID, email string) ([]ReferralCustomerEmail, error) {
	if customerID != "" && email != "" {
		return nil, errCustomerIDAndEmailExclusive
	}
	params := url.Values{}
	if customerID != "" {
		params.Set("customerId", customerID)
	}
	if email != "" {
		params.Set("email", email)
	}
	var resp []ReferralCustomerEmail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apiReferral/customization", params, sapiDefaultRate, &resp)
}

// CustomiseSpotPartnerClientID calls Partner Customize Id For Client (USER_DATA), setting the unique customer ID of a
// referred user's email; setting another ID for the same email replaces it
func (e *Exchange) CustomiseSpotPartnerClientID(ctx context.Context, customerID, email string) (*ReferralCustomerEmailResponse, error) {
	params, err := referralCustomerEmailParams(customerID, email)
	if err != nil {
		return nil, err
	}
	var resp *ReferralCustomerEmailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/apiReferral/customization", params, sapiDefaultRate, &resp)
}

// GetSpotInfoAboutIfUserIsNew calls Query Client If The New User (Spot) (USER_DATA)
func (e *Exchange) GetSpotInfoAboutIfUserIsNew(ctx context.Context, apiAgentCode string) (*SpotReferralNewUserResponse, error) {
	if apiAgentCode == "" {
		return nil, errAPIAgentCodeRequired
	}
	params := url.Values{}
	params.Set("apiAgentCode", apiAgentCode)
	var resp *SpotReferralNewUserResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apiReferral/ifNewUser", params, sapiDefaultRate, &resp)
}

// GetSpotOwnRebateRecentRecords calls Query Rebate Recent Record, covering the last 7 days; limit is at most 1000 and
// defaults to 500
func (e *Exchange) GetSpotOwnRebateRecentRecords(ctx context.Context, startTime, endTime time.Time, limit uint64) ([]ReferralRebateRecord, error) {
	params, err := brokerTimeWindowParams(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []ReferralRebateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apiReferral/kickback/recentRecord", params, sapiDefaultRate, &resp)
}

// GetSpotOthersRebateRecentRecord calls Query Partner Rebate Recent Record; the window, at most 7 days, needs both
// bounds or neither, which returns the last 7 days
func (e *Exchange) GetSpotOthersRebateRecentRecord(ctx context.Context, arg *PartnerRebateRecordRequest) ([]PartnerRebateRecord, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	withWindow := !arg.StartTime.IsZero() || !arg.EndTime.IsZero()
	if withWindow {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	if arg.Limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errLimitNumberRequired)
	}
	params := url.Values{}
	if arg.CustomerID != "" {
		params.Set("customerId", arg.CustomerID)
	}
	if withWindow {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	var resp []PartnerRebateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apiReferral/rebate/recentRecord", params, sapiDefaultRate, &resp)
}

// GetPAPIUsersCustomisedID returns the customer ID assigned to the user by a broker
func (e *Exchange) GetPAPIUsersCustomisedID(ctx context.Context, brokerID string) (*PAPICustomisedIDResponse, error) {
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	var resp *PAPICustomisedIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/apiReferral/userCustomization", params, pmAPIReferralRate, &resp)
}

// CustomisePAPIOwnClientID assigns a unique customer ID to a referred portfolio margin user
func (e *Exchange) CustomisePAPIOwnClientID(ctx context.Context, customerID, brokerID string) (*PAPICustomisedIDResponse, error) {
	if customerID == "" {
		return nil, errCustomerIDRequired
	}
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("brokerId", brokerID)
	var resp *PAPICustomisedIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/apiReferral/userCustomization", params, pmAPIReferralRate, &resp)
}

// GetFuturesUsersCustomisedID calls Get User's Customize Id (Futures) (USER_DATA), returning the customer ID the
// account set with CustomiseFuturesOwnClientID
func (e *Exchange) GetFuturesUsersCustomisedID(ctx context.Context, brokerID string) (*FuturesReferralCustomerIDResponse, error) {
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	var resp *FuturesReferralCustomerIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/userCustomization", params, uFuturesAPIReferralRate, &resp)
}

// CustomiseFuturesOwnClientID calls Customize Id For Futures Client (USER_DATA), letting a referred user set the
// unique customer ID their partner identifies them by; portfolio margin accounts use the PAPI endpoint instead
func (e *Exchange) CustomiseFuturesOwnClientID(ctx context.Context, customerID, brokerID string) (*FuturesReferralCustomerIDResponse, error) {
	if customerID == "" {
		return nil, errCustomerIDRequired
	}
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("brokerId", brokerID)
	var resp *FuturesReferralCustomerIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/apiReferral/userCustomization", params, uFuturesAPIReferralRate, &resp)
}

// GetFuturesClientEmailCustomisedID calls Get Futures Client Email Customized Id (USER_DATA), filtering by customer ID
// or email, not both
func (e *Exchange) GetFuturesClientEmailCustomisedID(ctx context.Context, arg *ReferralCustomerEmailRequest) ([]ReferralCustomerEmail, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CustomerID != "" && arg.Email != "" {
		return nil, errCustomerIDAndEmailExclusive
	}
	params := url.Values{}
	if arg.CustomerID != "" {
		params.Set("customerId", arg.CustomerID)
	}
	if arg.Email != "" {
		params.Set("email", arg.Email)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []ReferralCustomerEmail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/customization", params, uFuturesAPIReferralRate, &resp)
}

// CustomiseFuturesPartnerClientID calls Partner Customize Id For Futures Client (USER_DATA), setting the unique
// customer ID of a referred user's email
func (e *Exchange) CustomiseFuturesPartnerClientID(ctx context.Context, customerID, email string) (*ReferralCustomerEmailResponse, error) {
	params, err := referralCustomerEmailParams(customerID, email)
	if err != nil {
		return nil, err
	}
	var resp *ReferralCustomerEmailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/apiReferral/customization", params, uFuturesAPIReferralRate, &resp)
}

// GetFuturesRebateDataOverview calls Get Rebate Data Overview (USER_DATA) for USDⓈ-M futures, or COIN-M futures when
// coinMargined is set
func (e *Exchange) GetFuturesRebateDataOverview(ctx context.Context, coinMargined bool) (*ReferralRebateOverviewResponse, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("type", "2")
	}
	var resp *ReferralRebateOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/overview", params, uFuturesAPIReferralRate, &resp)
}

// GetRebateVolume calls Get Rebate Volume (USER_DATA)
func (e *Exchange) GetRebateVolume(ctx context.Context, arg *ReferralVolumeRequest) ([]ReferralRebateVolume, error) {
	params, err := referralVolumeParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []ReferralRebateVolume
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/rebateVol", params, uFuturesAPIReferralRate, &resp)
}

// GetTraderDetail calls Get Trader Detail (USER_DATA), returning the trade and rebate volume of referred traders, or
// of the one customerID identifies
func (e *Exchange) GetTraderDetail(ctx context.Context, customerID string, arg *ReferralVolumeRequest) ([]ReferralTraderDetail, error) {
	params, err := referralVolumeParams(arg)
	if err != nil {
		return nil, err
	}
	if customerID != "" {
		params.Set("customerId", customerID)
	}
	var resp []ReferralTraderDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/traderSummary", params, uFuturesAPIReferralRate, &resp)
}

// GetFuturesReferredTradersNumber calls Get Trader Number (USER_DATA), returning the number of new and existing
// referred traders
func (e *Exchange) GetFuturesReferredTradersNumber(ctx context.Context, arg *ReferralVolumeRequest) ([]ReferralTraderNumber, error) {
	params, err := referralVolumeParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []ReferralTraderNumber
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/traderNum", params, uFuturesAPIReferralRate, &resp)
}

// GetUserTradeVolume calls Get User Trade Volume (USER_DATA), returning referred users' trade volume
func (e *Exchange) GetUserTradeVolume(ctx context.Context, arg *ReferralVolumeRequest) ([]ReferralTradeVolume, error) {
	params, err := referralVolumeParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []ReferralTradeVolume
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/tradeVol", params, uFuturesAPIReferralRate, &resp)
}

// GetPAPIClientIfNewUser returns whether a referred portfolio margin user is new to the broker, for the USDⓈ-M
// futures account or, when coinMargined is set, the COIN-M futures account
func (e *Exchange) GetPAPIClientIfNewUser(ctx context.Context, brokerID string, coinMargined bool) (*PAPIClientNewUserStatusResponse, error) {
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	if coinMargined {
		params.Set("type", "2")
	}
	var resp *PAPIClientNewUserStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/apiReferral/ifNewUser", params, pmAPIReferralRate, &resp)
}

// GetFuturesClientIfNewUser calls Query Futures Client New User Status (USER_DATA) for USDⓈ-M futures, or COIN-M
// futures when coinMargined is set
func (e *Exchange) GetFuturesClientIfNewUser(ctx context.Context, brokerID string, coinMargined bool) (*FuturesReferralNewUserResponse, error) {
	if brokerID == "" {
		return nil, errInvalidBrokerID
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	if coinMargined {
		params.Set("type", "2")
	}
	var resp *FuturesReferralNewUserResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiReferral/ifNewUser", params, uFuturesAPIReferralRate, &resp)
}

// referralCustomerEmailParams builds the parameters both partner customer ID customisations require
func referralCustomerEmailParams(customerID, email string) (url.Values, error) {
	if customerID == "" {
		return nil, errCustomerIDRequired
	}
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("email", email)
	return params, nil
}

// referralVolumeParams builds the parameters the futures referral statistics share
func referralVolumeParams(arg *ReferralVolumeRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := brokerTimeWindowParams(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	if arg.CoinMargined {
		params.Set("type", "2")
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, nil
}
