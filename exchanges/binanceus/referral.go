package binanceus

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var errRowsRequired = errors.New("rows required")

// GetReferralRewardHistory returns a page of the referral rewards the account earned as a referrer or as a referee
func (e *Exchange) GetReferralRewardHistory(ctx context.Context, req *ReferralRewardHistoryRequest) (*ReferralRewardHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Rows == 0 {
		return nil, errRowsRequired
	}
	params := url.Values{}
	params.Set("userBizType", strconv.FormatUint(req.UserBusinessType, 10))
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	params.Set("rows", strconv.FormatUint(req.Rows, 10))
	var resp *ReferralRewardHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/marketing/referral/reward/history", params, defaultRate, &resp)
}
