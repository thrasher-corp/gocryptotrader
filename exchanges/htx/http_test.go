package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestHTTPResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, payload                              string
		authenticated, futures, version2, version5 bool
		nilResult                                  bool
		wantErr                                    error
		invalid                                    bool
		message                                    string
	}
	for _, tc := range []testCase{
		{name: "public success", payload: `{"status":"ok","data":{"equity":"12.5"}}`},
		{name: "public hyphenated error", payload: `{"err-code":"bad-request","err-msg":"hyphenated failure"}`, wantErr: errAPIResponse, message: "hyphenated failure"},
		{name: "public underscored error", payload: `{"err_code":123,"err_msg":"underscored failure"}`, wantErr: errAPIResponse, message: "underscored failure"},
		{name: "public status error", payload: `{"status":"error"}`, wantErr: errAPIResponse},
		{name: "public V5 success", version5: true, payload: `{"code":200,"data":{"equity":"12.5"}}`},
		{name: "public V5 error", version5: true, payload: `{"code":-1,"message":"V5 failure"}`, wantErr: errAPIResponse, message: "V5 failure"},
		{name: "spot authenticated success", authenticated: true, payload: `{"status":"ok","data":{"equity":"12.5"}}`},
		{name: "spot authenticated error", authenticated: true, payload: `{"status":"error","err-code":"bad-request","err-msg":"spot failure"}`, wantErr: request.ErrAuthRequestFailed, message: "spot failure"},
		{name: "V2 success", authenticated: true, version2: true, payload: `{"code":200,"data":{"equity":"12.5"}}`},
		{name: "V2 error", authenticated: true, version2: true, payload: `{"code":-1,"message":"V2 failure"}`, wantErr: request.ErrAuthRequestFailed, message: "V2 failure"},
		{name: "futures success", authenticated: true, futures: true, payload: `{"status":"ok","data":{"equity":"12.5"}}`},
		{name: "futures hyphenated error", authenticated: true, futures: true, payload: `{"err-code":123,"err-msg":"hyphenated failure"}`, wantErr: request.ErrAuthRequestFailed, message: "hyphenated failure"},
		{name: "futures underscored error", authenticated: true, futures: true, payload: `{"err_code":123,"err_msg":"underscored failure"}`, wantErr: request.ErrAuthRequestFailed, message: "underscored failure"},
		{name: "futures status error", authenticated: true, futures: true, payload: `{"status":"error"}`, wantErr: request.ErrAuthRequestFailed},
		{name: "V5 success", authenticated: true, futures: true, version5: true, payload: `{"code":0,"data":{"equity":"12.5"}}`},
		{name: "V5 error", authenticated: true, futures: true, version5: true, payload: `{"code":-1,"message":"V5 failure"}`, wantErr: request.ErrAuthRequestFailed, message: "V5 failure"},
		{name: "V3 error", authenticated: true, futures: true, version5: true, payload: `{"code":-1,"msg":"V3 failure"}`, wantErr: request.ErrAuthRequestFailed, message: "V3 failure"},
		{name: "public V3 error", version5: true, payload: `{"code":-1,"msg":"V3 failure"}`, wantErr: errAPIResponse, message: "V3 failure"},
		{name: "no result", payload: `{}`, nilResult: true, wantErr: errUnexpectedResponseBody},
		{name: "authenticated no result", authenticated: true, payload: `{}`, nilResult: true, wantErr: request.ErrAuthRequestFailed},
		{name: "empty body", wantErr: errExpectedResponseBody},
		{name: "malformed document", payload: `{`, invalid: true},
		{name: "invalid balance", payload: `{"data":{"equity":true}}`, invalid: true},
		{name: "authenticated invalid balance", authenticated: true, payload: `{"data":{"equity":true}}`, wantErr: request.ErrAuthRequestFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var balance V5AccountBalanceResponse
			response := httpResponse{Result: &balance, Authenticated: tc.authenticated, Futures: tc.futures, Version2: tc.version2, Version5: tc.version5}
			if tc.nilResult {
				response.Result = nil
			}
			err := response.UnmarshalJSON([]byte(tc.payload))
			assert.True(t, response.Received, "decoder should record receiving a body")
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "response must preserve the expected error")
				if tc.message != "" {
					assert.ErrorContains(t, err, tc.message, "response should preserve the exchange error message")
				}
				return
			}
			if tc.invalid {
				require.Error(t, err, "invalid responses must fail decoding")
				return
			}
			require.NoError(t, err, "successful responses must decode directly into their concrete type")
			assert.Equal(t, types.Number(12.5), balance.Balance.Equity, "balance should retain its quoted decimal value")
		})
	}
}
