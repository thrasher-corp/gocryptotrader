package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Tax errors
var (
	errTaxReportItemsNotSet  = errors.New("tax report items not set")
	errTaxReportTypeNotSet   = errors.New("tax report type not set")
	errTaxBatchIDNotSet      = errors.New("tax report batch ID not set")
	errInvalidTaxReportFiles = errors.New("invalid tax report files")
)

// GetUserRegisterDate returns when the user registered on Bybit (Get User Register Date)
func (e *Exchange) GetUserRegisterDate(ctx context.Context) (*UserRegisterDateResponse, error) {
	var resp *UserRegisterDateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/fht/compliance/tax/v3/private/registertime", nil, nil, &resp, defaultEPL)
}

// BatchCreateTaxReports requests the export of several tax reports in one batch, which Bybit generates
// asynchronously; a new batch is refused while an unfinished one exists (Batch Request Export Reports)
func (e *Exchange) BatchCreateTaxReports(ctx context.Context, req *BatchCreateTaxReportsRequest) (*BatchCreateTaxReportsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, errTaxReportItemsNotSet
	}
	items := make([]map[string]string, len(req.Items))
	for i := range req.Items {
		if req.Items[i].Type == "" {
			return nil, fmt.Errorf("%w: item %d", errTaxReportTypeNotSet, i)
		}
		// Bybit requires number even where it ignores it, so it is sent when empty
		items[i] = map[string]string{"type": req.Items[i].Type, "number": req.Items[i].Number}
	}
	body := map[string]any{
		"startTime": req.StartTime.Unix(),
		"endTime":   req.EndTime.Unix(),
		"items":     items,
	}
	if req.SourceInstitution != "" {
		body["sourceInstitution"] = req.SourceInstitution
	}
	if req.ExportFileType != "" {
		body["exportFileType"] = req.ExportFileType
	}
	var resp *BatchCreateTaxReportsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fht/compliance/tax/private/batch_create", nil, body, &resp, batchCreateTaxReportsEPL)
}

// GetBatchTaxReportsStatus returns the status and download location of each report in an export batch; Bybit
// recommends polling every 30 seconds (Get Batch Export Report Status)
func (e *Exchange) GetBatchTaxReportsStatus(ctx context.Context, batchID string) (*BatchTaxReportsStatusResponse, error) {
	if batchID == "" {
		return nil, errTaxBatchIDNotSet
	}
	params := url.Values{}
	params.Set("batchId", batchID)
	var resp *BatchTaxReportsStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/fht/compliance/tax/private/batch_query", params, nil, &resp, getBatchTaxReportsStatusEPL)
}
