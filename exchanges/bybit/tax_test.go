package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUserRegisterDate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserRegisterDate(t.Context())
	require.NoError(t, err, "GetUserRegisterDate must not error")
	require.NotNil(t, result, "GetUserRegisterDate must return a result")
	if mockTests {
		exp := &UserRegisterDateResponse{
			RegisterTime: types.Time(time.Unix(1634515200, 0)),
		}
		assert.Equal(t, exp, result, "GetUserRegisterDate should decode every field")
		return
	}
	assert.NotZero(t, result.RegisterTime, "GetUserRegisterDate should return a registration time")
}

func TestBatchCreateTaxReports(t *testing.T) {
	t.Parallel()
	_, err := e.BatchCreateTaxReports(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BatchCreateTaxReports must reject a nil request")
	req := &BatchCreateTaxReportsRequest{}
	_, err = e.BatchCreateTaxReports(t.Context(), req)
	require.ErrorIs(t, err, common.ErrDateUnset, "BatchCreateTaxReports must require a time window")
	req.StartTime, req.EndTime = time.Unix(1788220800, 0), time.Unix(1756684800, 0)
	_, err = e.BatchCreateTaxReports(t.Context(), req)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "BatchCreateTaxReports must reject a start time after the end time")
	req.StartTime, req.EndTime = time.Unix(1756684800, 0), time.Unix(1788220800, 0)
	_, err = e.BatchCreateTaxReports(t.Context(), req)
	require.ErrorIs(t, err, errTaxReportItemsNotSet, "BatchCreateTaxReports must require a report")
	req.Items = []TaxReportItem{{Type: "TRADE", Number: "1"}, {Number: "1"}}
	_, err = e.BatchCreateTaxReports(t.Context(), req)
	require.ErrorIs(t, err, errTaxReportTypeNotSet, "BatchCreateTaxReports must require each report's type")
	req.Items = []TaxReportItem{{Type: "TRADE", Number: "1"}, {Type: "EARN", Number: "1"}, {Type: "DEPOSIT&WITHDRAWAL", Number: "1"}}
	req.SourceInstitution = "INST_A"
	req.ExportFileType = "csv"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchCreateTaxReports(t.Context(), req)
	require.NoError(t, err, "BatchCreateTaxReports must not error")
	if mockTests {
		assert.Equal(t, &BatchCreateTaxReportsResponse{BatchID: "b5f4a3e2-1c8d-4f96-9a03-2e7b8c6d4f10"}, result, "BatchCreateTaxReports should decode the batch ID")
		return
	}
	assert.NotEmpty(t, result.BatchID, "BatchCreateTaxReports should return a batch ID")
}

func TestGetBatchTaxReportsStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchTaxReportsStatus(t.Context(), "")
	require.ErrorIs(t, err, errTaxBatchIDNotSet, "GetBatchTaxReportsStatus must require a batch ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBatchTaxReportsStatus(t.Context(), "b5f4a3e2-1c8d-4f96-9a03-2e7b8c6d4f10")
	require.NoError(t, err, "GetBatchTaxReportsStatus must not error")
	require.NotNil(t, result, "GetBatchTaxReportsStatus must return a result")
	if mockTests {
		exp := &BatchTaxReportsStatusResponse{
			Items: []TaxReportTask{
				{
					Type:   "TRADE",
					Number: "1",
					Status: 2,
					URL: TaxReportFiles{
						BasePath: "https://prod-bybit-tax-api-1705-rz.s3.ap-southeast-1.amazonaws.com/",
						Files: []string{
							"20260929/6b1b3341-6940-47a7-b112-5949e496736f/_SUCCESS",
							"20260929/6b1b3341-6940-47a7-b112-5949e496736f/part-00000-92859919-c8bf-47ce-b349-9cdcf44e8045-c000.csv",
						},
					},
					QueryID: "2104839833489960960",
				},
				{
					Type:    "EARN",
					Number:  "1",
					Status:  -1,
					QueryID: "2104839833871642624",
				},
				{
					Type:    "DEPOSIT&WITHDRAWAL",
					Number:  "1",
					Status:  1,
					QueryID: "2104839834236547072",
				},
			},
		}
		assert.Equal(t, exp, result, "GetBatchTaxReportsStatus should decode every field")
	}
}

func TestTaxReportFilesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const inp = `"{\"Basepath\":\"https://example.com/reports/\",\"Files\":[\"20260929/_SUCCESS\",\"20260929/part-00000.csv\"]}"`
	var files TaxReportFiles
	require.NoError(t, json.Unmarshal([]byte(inp), &files), "Unmarshal must not error")
	exp := TaxReportFiles{BasePath: "https://example.com/reports/", Files: []string{"20260929/_SUCCESS", "20260929/part-00000.csv"}}
	assert.Equal(t, exp, files, "TaxReportFiles should decode the encoded files")

	require.NoError(t, json.Unmarshal([]byte(`""`), &files), "Unmarshal must not error on an empty string")
	assert.Equal(t, TaxReportFiles{}, files, "TaxReportFiles should be empty until the report is ready")

	err := json.Unmarshal([]byte(`"https://example.com/reports/part-00000.csv"`), &files)
	assert.ErrorIs(t, err, errInvalidTaxReportFiles, "Unmarshal should reject a string that does not encode the files")
	err = json.Unmarshal([]byte(`{"Basepath":"https://example.com/reports/"}`), &files)
	assert.ErrorIs(t, err, errInvalidTaxReportFiles, "Unmarshal should reject files that are not a JSON-encoded string")
}
