package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryNTP(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		modify func([]byte)
		want   error
	}{
		{name: "valid v4", modify: func([]byte) {}},
		{name: "valid v3", modify: func(p []byte) { p[0] = 3<<3 | 4 }},
		{name: "unexpected version", modify: func(p []byte) { p[0] = 1<<3 | 4 }, want: errNTPReplyVersion},
		{name: "future version", modify: func(p []byte) { p[0] = 7<<3 | 4 }, want: errNTPReplyVersion},
		{name: "zero receive", modify: func(p []byte) { clear(p[32:40]) }, want: errNTPReceiveTime},
		{name: "zero transmit", modify: func(p []byte) { clear(p[40:48]) }, want: ntp.ErrInvalidTransmitTime},
		{name: "wrong origin", modify: func(p []byte) { p[24] ^= 1 }, want: ntp.ErrServerResponseMismatch},
		{name: "wrong mode", modify: func(p []byte) { p[0] = 4<<3 | 3 }, want: ntp.ErrInvalidMode},
		{name: "unsynchronised", modify: func(p []byte) { p[0] |= 3 << 6 }, want: ntp.ErrInvalidLeapSecond},
		{name: "uncorrelated KoD", modify: func(p []byte) { p[1] = 0; copy(p[12:16], "RATE"); p[24] ^= 1; clear(p[32:48]) }, want: ntp.ErrServerResponseMismatch},
		{name: "KoD without server timestamps", modify: func(p []byte) { p[1] = 0; copy(p[12:16], "RATE"); clear(p[32:48]) }, want: ntp.ErrKissOfDeath},
		{name: "KoD", modify: func(p []byte) { p[1] = 0; copy(p[12:16], "RATE"); clear(p[32:40]) }, want: ntp.ErrKissOfDeath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			address, _ := ntpTestServer(t, tc.modify)
			response, err := queryNTP(t.Context(), address)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want, "invalid response must be rejected")
				if errors.Is(err, ntp.ErrKissOfDeath) {
					require.NotNil(t, response, "KoD response must remain available")
					assert.Equal(t, "RATE", response.KissCode, "KoD reason should be preserved")
				}
				return
			}
			require.NoError(t, err, "valid response must be accepted")
			assert.InDelta(t, float64(time.Second), float64(response.ClockOffset), float64(100*time.Millisecond), "server one second ahead should yield a positive correction")
		})
	}
}

// TestQueryNTPControlCorrelation checks RATE, DENY and RSTR replies are returned to caller only when they match request.
// GCT keeps its raw UDP adapter for these checks.
// Matching and non-matching kiss-o'-death cases follow ntpd-rs test_handle_kod: https://github.com/pendulum-project/ntpd-rs/blob/46ec9bb4d5b6cb24f814f5543d85b9138afb4cba/ntp-proto/src/source.rs#L1339
func TestQueryNTPControlCorrelation(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"RATE", "DENY", "RSTR"} {
		for _, correlated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/correlated=%t", code, correlated), func(t *testing.T) {
				t.Parallel()
				address, _ := ntpTestServer(t, func(packet []byte) {
					packet[1] = 0
					copy(packet[12:16], code)
					clear(packet[32:48])
					if !correlated {
						packet[24] ^= 1
					}
				})
				response, err := queryNTP(t.Context(), address)
				if !correlated {
					assert.ErrorIs(t, err, ntp.ErrServerResponseMismatch, "uncorrelated control reply should be rejected")
					assert.Nil(t, response, "uncorrelated reply should not expose an actionable control response")
					return
				}
				require.ErrorIs(t, err, ntp.ErrKissOfDeath, "correlated control reply must remain distinguishable")
				require.NotNil(t, response, "correlated control reply must preserve its code")
				assert.Equal(t, code, response.KissCode, "control reply should retain its exact meaning")
			})
		}
	}
}

func TestQueryNTPCancellation(t *testing.T) {
	t.Parallel()
	address, received := ntpTestServer(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := queryNTP(ctx, address)
		done <- err
	}()
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("query must reach the server")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "cancellation should interrupt the connected read")
	case <-time.After(time.Second):
		t.Fatal("cancelled query must return before the five-second socket timeout")
	}
}

func TestQueryNTPDeadline(t *testing.T) {
	t.Parallel()
	address, _ := ntpTestServer(t, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := queryNTP(ctx, address)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "deadline should interrupt the connected read")
}

// ntpTestServer returns a valid matching response one second ahead of local clock.
// A nil modifier withholds the response.
func ntpTestServer(t *testing.T, modify func([]byte)) (address string, requestReceived <-chan struct{}) {
	t.Helper()
	listener, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	require.NoError(t, err, "loopback server must listen")
	received := make(chan struct{})
	done := make(chan struct{})
	t.Cleanup(func() {
		assert.NoError(t, listener.Close(), "loopback server should close")
		<-done
	})
	go func() {
		defer close(done)
		request := make([]byte, 512)
		_, peer, err := listener.ReadFrom(request)
		if err != nil {
			return
		}
		close(received)
		if modify == nil {
			return
		}
		reply := make([]byte, 48)
		reply[0], reply[1] = 4<<3|4, 1
		reply[2], reply[3] = 10, 236
		now := time.Now().Add(time.Second)
		stamp := uint64((now.Unix()+2208988800)&0xffffffff)<<32 | uint64(now.Nanosecond()&0x7fffffff)<<32/1e9
		binary.BigEndian.PutUint64(reply[16:24], stamp)
		copy(reply[24:32], request[40:48])
		binary.BigEndian.PutUint64(reply[32:40], stamp)
		binary.BigEndian.PutUint64(reply[40:48], stamp)
		modify(reply)
		_, _ = listener.WriteTo(reply, peer)
	}()
	return listener.LocalAddr().String(), received
}
