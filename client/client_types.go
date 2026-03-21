// Package client provides a Go client for the GoCryptoTrader gRPC service.
//
// Clients running on the same server should prefer the Unix Domain Socket
// transport (ConnectViaSocket) because it bypasses the kernel network stack
// and is ~2-5x faster than TCP loopback. Remote clients use ConnectViaTCP.
//
// Typical same-server usage:
//
//	c, err := client.ConnectViaSocket("/tmp/gocryptotrader.sock", "admin", "Password")
//	if err != nil { ... }
//	defer c.Close()
//	info, err := c.GetInfo(ctx)
//
// Typical remote usage:
//
//	c, err := client.ConnectViaTCP("localhost:9052", "/path/to/cert.pem", "admin", "Password")
//	if err != nil { ... }
//	defer c.Close()
package client

import (
	"time"
)

// Config holds the parameters needed to connect to the GoCryptoTrader gRPC
// server, regardless of transport.
type Config struct {
	// Username and Password are the Basic-auth credentials required by the
	// server's authentication interceptor.
	Username string
	Password string

	// For Unix Domain Socket connections.
	SocketPath string

	// For TCP connections.
	Host     string
	CertPath string

	// Timeout applied to individual RPC calls.  Defaults to 30 s when zero.
	CallTimeout time.Duration
}
