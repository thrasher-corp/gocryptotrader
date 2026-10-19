package engine

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/beevik/ntp"
)

var (
	errNTPReplyVersion = errors.New("unsupported NTP reply version")
	errNTPReceiveTime  = errors.New("missing NTP receive timestamp")
)

// queryNTP sends one NTP request and returns when it completes or ctx is cancelled.
// It waits for any started cancellation callback before returning, so socket is never used afterwards.
// It gives beevik raw UDP socket so beevik can use kernel receive timestamps.
func queryNTP(ctx context.Context, address string) (*ntp.Response, error) {
	checks := &ntpReplyChecks{}
	var stop func() bool
	closed := make(chan struct{})
	defer func() {
		if stop != nil && !stop() {
			<-closed
		}
	}()
	response, err := ntp.QueryWithOptions(address, ntp.QueryOptions{
		Version:    4,
		Timeout:    5 * time.Second,
		Extensions: []ntp.Extension{checks},
		Dialer: func(_, remote string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, "udp", remote)
			if err != nil {
				return nil, err
			}
			// Close socket when ctx is cancelled, which unblocks beevik's read.
			// Initial cancellation deadline could be overwritten when beevik sets its deadline after dialer returns.
			// Based on Go's context.AfterFunc connection example: https://github.com/golang/go/blob/go1.27.0/src/context/example_test.go
			stop = context.AfterFunc(ctx, func() {
				_ = conn.Close()
				close(closed)
			})
			return conn, nil
		},
	})
	if ctx.Err() != nil {
		return nil, fmt.Errorf("query NTP server %s: %w", address, ctx.Err())
	}
	if err != nil {
		return checks.kiss, fmt.Errorf("query NTP server %s: %w", address, err)
	}
	if err := response.Validate(); err != nil {
		// Return response too, so caller can act on kiss-o'-death code.
		return response, fmt.Errorf("validate NTP server %s: %w", address, err)
	}
	return response, nil
}

// ntpReplyChecks adds reply checks that beevik v1.6.0 lacks, without replacing its parser, correlation checks or arithmetic.
// It rejects unsupported NTP versions, as systemd-timesyncd does, and missing receive timestamps (RFC 5905 section 6).
// It also hands kiss-o'-death replies answering our request to caller, even though their timestamps may be empty (section 7.4).
// https://github.com/systemd/systemd/blob/885fe07ee37cff7316680b5088d11081e01813b1/src/timesync/timesyncd-manager.c#L488
// https://github.com/beevik/ntp/blob/953b63646f5273d44de88b68e5862ad155ed4660/ntp4.go#L262
type ntpReplyChecks struct {
	origin [8]byte
	kiss   *ntp.Response
}

func (checks *ntpReplyChecks) ProcessQuery(packet *bytes.Buffer) error {
	copy(checks.origin[:], packet.Bytes()[40:48])
	return nil
}

func (checks *ntpReplyChecks) ProcessResponse(packet []byte) error {
	if len(packet) < 48 {
		return ntp.ErrInvalidTime
	}
	version := packet[0] >> 3 & 7
	if version != 3 && version != 4 {
		return errNTPReplyVersion
	}
	if packet[1] == 0 {
		// beevik would otherwise reject unspecified kiss-o'-death transmit timestamps before exposing kiss code.
		// Reject kiss-o'-death replies whose origin timestamp does not match our request before applying their restrictions.
		// This follows RFC 8633 section 5.4 and does not authenticate server.
		if packet[0]&7 != 4 {
			return ntp.ErrInvalidMode
		}
		if !bytes.Equal(packet[24:32], checks.origin[:]) {
			return ntp.ErrServerResponseMismatch
		}
		poll := time.Second
		if packet[2] < 128 {
			// RFC 8633 limits accepted interval in RATE replies to 8192 seconds.
			// Accepting larger values could delay checks against this server for years.
			poll <<= min(packet[2], 13)
		}
		checks.kiss = &ntp.Response{Version: int(version), KissCode: string(packet[12:16]), Poll: poll}
		return ntp.ErrKissOfDeath
	}
	if binary.BigEndian.Uint64(packet[32:40]) == 0 {
		return errNTPReceiveTime
	}
	return nil
}
