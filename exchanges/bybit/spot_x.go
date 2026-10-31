package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetLaunchpoolProjectList returns Launchpool projects, newest first (Get Launchpool Project List). The status is always
// sent, since 0 selects upcoming projects
func (e *Exchange) GetLaunchpoolProjectList(ctx context.Context, req *SpotXProjectListRequest) (*LaunchpoolProjectListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(req.Status, 10))
	if !req.ActivityCoin.IsEmpty() {
		params.Set("activityCoin", req.ActivityCoin.Upper().String())
	}
	if req.ProjectID != "" {
		params.Set("projectId", req.ProjectID)
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *LaunchpoolProjectListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-x/launchpool/project/list", params), defaultEPL, &resp)
}

// GetLaunchpoolUserActivityLog returns the account's Launchpool stake, redeem and reward records (Get Launchpool User
// Activity Log)
func (e *Exchange) GetLaunchpoolUserActivityLog(ctx context.Context, req *LaunchpoolUserActivityLogRequest) (*LaunchpoolUserActivityLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if !req.StartTime.IsZero() || !req.EndTime.IsZero() {
		// The window needs both bounds, which StartEndTimeCheck enforces
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		body["startTime"] = strconv.FormatInt(req.StartTime.UnixMilli(), 10)
		body["endTime"] = strconv.FormatInt(req.EndTime.UnixMilli(), 10)
	}
	if !req.StakeCoin.IsEmpty() {
		body["stakeCoin"] = req.StakeCoin.Upper().String()
	}
	if req.Type != nil {
		body["type"] = *req.Type
	}
	if req.Status != nil {
		body["status"] = *req.Status
	}
	if req.PageSize != 0 {
		body["pageSize"] = req.PageSize
	}
	if req.Current != 0 {
		body["current"] = req.Current
	}
	var resp *LaunchpoolUserActivityLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-x/launchpool/user/activity-log", nil, body, &resp, defaultEPL)
}

// GetLaunchpoolCurrentStaking returns the account's active Launchpool staking positions, up to 30 (Get Launchpool
// Current Staking)
func (e *Exchange) GetLaunchpoolCurrentStaking(ctx context.Context) (*LaunchpoolCurrentStakingResponse, error) {
	var resp *LaunchpoolCurrentStakingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-x/launchpool/user/current-staking", nil, nil, &resp, defaultEPL)
}

// GetLaunchpoolUserHistory returns the account's ended Launchpool staking positions (Get Launchpool User History)
func (e *Exchange) GetLaunchpoolUserHistory(ctx context.Context, req *LaunchpoolUserHistoryRequest) (*LaunchpoolUserHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if !req.StartTime.IsZero() || !req.EndTime.IsZero() {
		// The window needs both bounds, which StartEndTimeCheck enforces
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		body["startTime"] = strconv.FormatInt(req.StartTime.UnixMilli(), 10)
		body["endTime"] = strconv.FormatInt(req.EndTime.UnixMilli(), 10)
	}
	if !req.StakeCoin.IsEmpty() {
		body["stakeCoin"] = req.StakeCoin.Upper().String()
	}
	if !req.RewardCoin.IsEmpty() {
		body["rewardCoin"] = req.RewardCoin.Upper().String()
	}
	if req.PageSize != 0 {
		body["pageSize"] = req.PageSize
	}
	if req.Current != 0 {
		body["current"] = req.Current
	}
	var resp *LaunchpoolUserHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-x/launchpool/user/history", nil, body, &resp, defaultEPL)
}

// GetPuzzleProjectList returns Puzzle projects, newest first (Get Puzzle Project List)
func (e *Exchange) GetPuzzleProjectList(ctx context.Context, req *SpotXProjectListRequest) (*PuzzleProjectListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(req.Status, 10))
	if !req.ActivityCoin.IsEmpty() {
		params.Set("activityCoin", req.ActivityCoin.Upper().String())
	}
	if req.ProjectID != "" {
		params.Set("projectId", req.ProjectID)
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *PuzzleProjectListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-x/puzzle/project/list", params), defaultEPL, &resp)
}

// GetTokenSplashProjectList returns Token Splash projects, newest first (Get Token Splash Project List)
func (e *Exchange) GetTokenSplashProjectList(ctx context.Context, req *SpotXProjectListRequest) (*TokenSplashProjectListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(req.Status, 10))
	if !req.ActivityCoin.IsEmpty() {
		params.Set("activityCoin", req.ActivityCoin.Upper().String())
	}
	if req.ProjectID != "" {
		params.Set("projectId", req.ProjectID)
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *TokenSplashProjectListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-x/token-splash/project/list", params), defaultEPL, &resp)
}

// GetTokenSplashUserActivityParams returns the account's trade task progress in the Token Splash projects it registered
// for whose rewards are not yet distributed (Get Token Splash User Activity Params)
func (e *Exchange) GetTokenSplashUserActivityParams(ctx context.Context, projectID string, activityCoin currency.Code) (*TokenSplashUserActivityParamsResponse, error) {
	params := url.Values{}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	if !activityCoin.IsEmpty() {
		params.Set("activityCoin", activityCoin.Upper().String())
	}
	var resp *TokenSplashUserActivityParamsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-x/token-splash/user/activity-params", params, nil, &resp, defaultEPL)
}
