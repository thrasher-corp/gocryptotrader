package gateio

import (
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestGetWSPingHandler(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		channel string
		err     error
	}{
		{optionsPingChannel, nil},
		{futuresPingChannel, nil},
		{spotPingChannel, nil},
		{"dong", errInvalidPingChannel},
	} {
		got, err := getWSPingHandler(tc.channel)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, time.Second*10, got.Delay)
		require.Equal(t, gws.TextMessage, got.MessageType)
		require.Contains(t, string(got.Message), tc.channel)
	}
}
