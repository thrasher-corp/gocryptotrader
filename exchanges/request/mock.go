package request

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

var mockResponseFlag = struct{ name string }{name: "mockResponse"}

// IsMockResponse returns true if the request has a mock response set
func IsMockResponse(ctx context.Context) bool {
	return ctx.Value(mockResponseFlag) != nil
}

// WithMockResponse sets the mock response for a request. This is used for testing purposes.
// REST response is single. Websocket response can be multiple. This allows expected responses to be set for a request if required.
func WithMockResponse(ctx context.Context, mockResponse ...[]byte) context.Context {
	return context.WithValue(ctx, mockResponseFlag, mockResponse)
}

// GetMockResponse returns the mock response for a request
func GetMockResponse(ctx context.Context) [][]byte {
	mockResponse, _ := ctx.Value(mockResponseFlag).([][]byte)
	return mockResponse
}

func getRESTResponseFromMock(ctx context.Context) *http.Response {
	mockResp := GetMockResponse(ctx)
	if len(mockResp) != 1 {
		panic("mock REST response invalid, requires exactly one response")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(io.Reader(io.LimitReader(bytes.NewBuffer(mockResp[0]), drainBodyLimit))),
	}
}
