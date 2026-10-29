package kucoin

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorGetError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		resp Error
		exp  string
	}{
		{name: "success", resp: Error{Code: "200000"}},
		{name: "success with a message", resp: Error{Code: "200000", Message: "success"}},
		{name: "legacy success code", resp: Error{Code: "200"}},
		{name: "failure", resp: Error{Code: "400100", Message: "Parameter Error"}, exp: "code: 400100 message: Parameter Error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.resp.GetError()
			if tc.exp == "" {
				assert.NoError(t, err, "GetError should report no error for a success code")
				return
			}
			assert.EqualError(t, err, tc.exp, "GetError should report the code and message")
			var respErr *responseError
			assert.ErrorAs(t, err, &respErr, "GetError should report a failure KuCoin answered as a response error")
		})
	}
	err := Error{Code: "unknown"}.GetError()
	require.ErrorIs(t, err, strconv.ErrSyntax, "GetError must reject a code that is not a number")
}
