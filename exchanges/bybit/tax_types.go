package bybit

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UserRegisterDateResponse holds when the user registered on Bybit
type UserRegisterDateResponse struct {
	RegisterTime types.Time `json:"registerTime"`
}

// BatchCreateTaxReportsRequest holds the parameters of Batch Request Export Reports; the window spans at most 12 months
type BatchCreateTaxReportsRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Items     []TaxReportItem
	// SourceInstitution identifies the institution making the request
	SourceInstitution string
	// ExportFileType is orc, the default, or csv
	ExportFileType string
}

// TaxReportItem selects a report to export: Type is a report type such as TRADE or EARN, or ALL for every report the
// site supports, and Number the report's subcategory, which Bybit ignores for ALL
type TaxReportItem struct {
	Type   string
	Number string
}

// BatchCreateTaxReportsResponse holds the ID of the export batch, which Bybit generates asynchronously
type BatchCreateTaxReportsResponse struct {
	BatchID string `json:"batchId"`
}

// BatchTaxReportsStatusResponse holds the status of every report in an export batch
type BatchTaxReportsStatusResponse struct {
	Items []TaxReportTask `json:"items"`
}

// TaxReportTask is the export of one report in a batch
type TaxReportTask struct {
	Type   string `json:"type"`
	Number string `json:"number"`
	// Status is 0 queued, 1 processing, 2 succeeded or -1 failed
	Status int64 `json:"status"`
	// URL locates the report's files once the task has succeeded
	URL     TaxReportFiles `json:"url"`
	QueryID string         `json:"queryId"`
}

// TaxReportFiles locates the files of a generated tax report: each file is downloaded from BasePath joined with its
// name. Bybit splits large reports into several files
type TaxReportFiles struct {
	BasePath string   `json:"Basepath"`
	Files    []string `json:"Files"`
}

// UnmarshalJSON decodes the files from the JSON-encoded string Bybit sends, which is empty until the report is ready
func (f *TaxReportFiles) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return fmt.Errorf("%w: %w", errInvalidTaxReportFiles, err)
	}
	*f = TaxReportFiles{}
	if encoded == "" {
		return nil
	}
	type taxReportFiles TaxReportFiles // Drops this method, so the encoded object decodes with the default rules
	if err := json.Unmarshal([]byte(encoded), (*taxReportFiles)(f)); err != nil {
		return fmt.Errorf("%w: %w", errInvalidTaxReportFiles, err)
	}
	return nil
}
