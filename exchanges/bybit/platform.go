package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var errLocaleRequired = errors.New("locale is required")

// GetAnnouncements returns Bybit's announcements, newest first (Get Announcement)
func (e *Exchange) GetAnnouncements(ctx context.Context, req *AnnouncementsRequest) (*AnnouncementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Locale == "" {
		return nil, errLocaleRequired
	}
	params := url.Values{}
	params.Set("locale", req.Locale)
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.Tag != "" {
		params.Set("tag", req.Tag)
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *AnnouncementsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/announcements/index", params), defaultEPL, &resp)
}

// RequestDemoTradingFunds adds funds to, or removes them from, a demo trading account (Request Demo Trading Funds).
// Bybit serves this endpoint only on its demo trading host, the exchange.RestSandbox URL, and it must be signed with the
// demo account's API key
func (e *Exchange) RequestDemoTradingFunds(ctx context.Context, req *DemoTradingFundsRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	body := make(map[string]any)
	if req.AdjustType != 0 {
		body["adjustType"] = req.AdjustType
	}
	if len(req.UTADemoApplyMoney) != 0 {
		funds := make([]map[string]any, len(req.UTADemoApplyMoney))
		for i := range req.UTADemoApplyMoney {
			fund := make(map[string]any)
			if !req.UTADemoApplyMoney[i].Coin.IsEmpty() {
				fund["coin"] = req.UTADemoApplyMoney[i].Coin.Upper().String()
			}
			if req.UTADemoApplyMoney[i].Amount != 0 {
				fund["amountStr"] = strconv.FormatFloat(req.UTADemoApplyMoney[i].Amount, 'f', -1, 64)
			}
			funds[i] = fund
		}
		body["utaDemoApplyMoney"] = funds
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSandbox, http.MethodPost, "/v5/account/demo-apply-money", nil, body, nil, requestDemoTradingFundsEPL)
}

// CreateDemoAccount creates a demo trading account under the calling main or sub account, or returns the existing
// one (Create Demo Account)
func (e *Exchange) CreateDemoAccount(ctx context.Context) (*CreateDemoAccountResponse, error) {
	var resp *CreateDemoAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-demo-member", nil, nil, &resp, createDemoAccountEPL)
}

// GetSystemStatus returns platform maintenance and service incidents, optionally filtered by ID and state (Get
// System Status)
func (e *Exchange) GetSystemStatus(ctx context.Context, id, state string) (*SystemStatusResponse, error) {
	params := url.Values{}
	if id != "" {
		params.Set("id", id)
	}
	if state != "" {
		params.Set("state", state)
	}
	var resp *SystemStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/system/status", params), defaultEPL, &resp)
}
