package grpcserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockCert(t *testing.T, pemType string, notAfter time.Time) []byte {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"gocryptotrader"}},
		NotBefore:    time.Now().Add(-2 * time.Hour),
		NotAfter:     notAfter,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	require.NoError(t, err, "CreateCertificate must not error")
	return pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: derBytes})
}

func TestVerifyCert(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		pemData []byte
		wantErr error
	}{
		{name: "valid certificate", pemData: mockCert(t, "CERTIFICATE", time.Now().Add(time.Hour))},
		{name: "no PEM data", wantErr: errCertDataIsNil},
		{name: "wrong PEM type", pemData: mockCert(t, "MEOW", time.Now().Add(time.Hour)), wantErr: errCertTypeInvalid},
		{name: "expired certificate", pemData: mockCert(t, "CERTIFICATE", time.Now().Add(-time.Hour)), wantErr: errCertExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, verifyCert(tc.pemData), tc.wantErr, "verifyCert should return the expected error")
		})
	}

	t.Run("unparsable certificate", func(t *testing.T) {
		t.Parallel()
		pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not DER")})
		assert.ErrorContains(t, verifyCert(pemData), "unable to parse TLS certificate", "verifyCert should report a parse failure")
	})
}

func TestGenCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, genCert(dir), "genCert must not error")

	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err, "LoadX509KeyPair must load the generated pair")

	pemData, err := os.ReadFile(certFile)
	require.NoError(t, err, "ReadFile must not error")
	require.NoError(t, verifyCert(pemData), "verifyCert must accept the generated certificate")
	block, _ := pem.Decode(pemData)
	require.NotNil(t, block, "generated certificate must be PEM encoded")
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err, "ParseCertificate must not error")
	assert.Contains(t, cert.DNSNames, "localhost", "certificate should cover the default localhost listen address")
	for _, want := range []string{"127.0.0.1", "::1"} {
		assert.Truef(t, slices.ContainsFunc(cert.IPAddresses, net.ParseIP(want).Equal), "certificate should cover %s", want)
	}
}

func TestCheckCerts(t *testing.T) {
	t.Parallel()

	t.Run("generates a missing pair", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, CheckCerts(dir), "CheckCerts must not error")
		assert.FileExists(t, filepath.Join(dir, "cert.pem"), "CheckCerts should write cert.pem")
		assert.FileExists(t, filepath.Join(dir, "key.pem"), "CheckCerts should write key.pem")
	})

	t.Run("keeps a valid pair", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, genCert(dir), "genCert must not error")
		before, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
		require.NoError(t, err, "ReadFile must not error")
		require.NoError(t, CheckCerts(dir), "CheckCerts must not error")
		after, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
		require.NoError(t, err, "ReadFile must not error")
		assert.Equal(t, before, after, "CheckCerts should keep a valid certificate")
	})

	for _, missing := range []string{"cert.pem", "key.pem"} {
		t.Run("regenerates when "+missing+" is missing", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, genCert(dir), "genCert must not error")
			require.NoError(t, os.Remove(filepath.Join(dir, missing)), "Remove must not error")
			require.NoError(t, CheckCerts(dir), "CheckCerts must not error")
			_, err := tls.LoadX509KeyPair(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
			assert.NoError(t, err, "CheckCerts should leave a matching key pair")
		})
	}

	t.Run("regenerates an expired certificate", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, genCert(dir), "genCert must not error")
		certFile := filepath.Join(dir, "cert.pem")
		require.NoError(t, os.WriteFile(certFile, mockCert(t, "CERTIFICATE", time.Now().Add(-time.Hour)), 0o600), "WriteFile must not error")
		require.NoError(t, CheckCerts(dir), "CheckCerts must not error")
		pemData, err := os.ReadFile(certFile)
		require.NoError(t, err, "ReadFile must not error")
		assert.NoError(t, verifyCert(pemData), "CheckCerts should replace the expired certificate")
	})

	t.Run("rejects an invalid certificate", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, genCert(dir), "genCert must not error")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("not a certificate"), 0o600), "WriteFile must not error")
		assert.ErrorIs(t, CheckCerts(dir), errCertDataIsNil, "CheckCerts should return the verification error")
	})
}
