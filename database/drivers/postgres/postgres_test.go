package postgres

import (
	"database/sql"
	"net"
	"net/netip"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// connectionOpeners counts the routines open sql.DB pools run until they are closed
func connectionOpeners() int {
	buf := make([]byte, 1<<20)
	return strings.Count(string(buf[:runtime.Stack(buf, true)]), "database/sql.(*DB).connectionOpener")
}

func TestConnectClosesConnectionWhenPingFails(t *testing.T) {
	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "Listen must not error")
	addrPort, err := netip.ParseAddrPort(listener.Addr().String())
	require.NoError(t, err, "ParseAddrPort must not error")
	require.NoError(t, listener.Close(), "listener Close must not error")

	control, err := sql.Open(database.DBPostgreSQL, "postgres://127.0.0.1/control")
	require.NoError(t, err, "sql.Open must not error")
	require.Eventually(t, func() bool { return connectionOpeners() == 1 }, 5*time.Second, 10*time.Millisecond, "an open pool must run one connection opener")
	require.NoError(t, control.Close(), "Close must not error")
	require.Eventually(t, func() bool { return connectionOpeners() == 0 }, 5*time.Second, 10*time.Millisecond, "a closed pool must stop its connection opener")

	_, err = Connect(&database.Config{
		Enabled:  true,
		Driver:   database.DBPostgreSQL,
		Host:     "127.0.0.1",
		Port:     uint32(addrPort.Port()),
		Database: "gct",
	})
	require.Error(t, err, "Connect must error when the server refuses connections")
	assert.Eventually(t, func() bool { return connectionOpeners() == 0 }, 5*time.Second, 10*time.Millisecond, "Connect should close the connection it could not ping")
}
