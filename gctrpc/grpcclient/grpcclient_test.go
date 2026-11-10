package grpcclient

import (
	"encoding/pem"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("missing certificate file", func(t *testing.T) {
		t.Parallel()
		_, err := New("localhost:9052", filepath.Join(t.TempDir(), "cert.pem"), "user", "pass")
		assert.ErrorIs(t, err, fs.ErrNotExist, "New should return the certificate read error")
	})

	t.Run("certificate file without a certificate", func(t *testing.T) {
		t.Parallel()
		certFile := filepath.Join(t.TempDir(), "cert.pem")
		require.NoError(t, os.WriteFile(certFile, []byte("not a certificate"), 0o600), "WriteFile must not error")
		_, err := New("localhost:9052", certFile, "user", "pass")
		assert.Error(t, err, "New should reject a file without a certificate")
	})

	t.Run("valid certificate", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewTLSServer(http.NotFoundHandler())
		t.Cleanup(srv.Close)
		certFile := filepath.Join(t.TempDir(), "cert.pem")
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
		require.NoError(t, os.WriteFile(certFile, certPEM, 0o600), "WriteFile must not error")
		conn, err := New("localhost:9052", certFile, "user", "pass")
		require.NoError(t, err, "New must not error")
		assert.Equal(t, "localhost:9052", conn.Target(), "New should connect to the given target")
		assert.NoError(t, conn.Close(), "Close should not error")
	})
}
