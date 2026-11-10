package grpcserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errCertExpired     = errors.New("gRPC TLS certificate has expired")
	errCertDataIsNil   = errors.New("gRPC TLS certificate PEM data is nil")
	errCertTypeInvalid = errors.New("gRPC TLS certificate type is invalid")
)

// CheckCerts verifies the gRPC TLS certificate and key in certDir, generating a new pair when either is missing or
// the certificate has expired
func CheckCerts(certDir string) error {
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	if !file.Exists(certFile) || !file.Exists(keyFile) {
		log.Warnln(log.Global, "gRPC certificate/key file missing, recreating...")
		return genCert(certDir)
	}

	pemData, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("unable to open TLS cert file: %w", err)
	}

	if err := verifyCert(pemData); err != nil {
		if !errors.Is(err, errCertExpired) {
			return err
		}
		log.Warnln(log.Global, "gRPC certificate has expired, regenerating...")
		return genCert(certDir)
	}

	log.Infoln(log.Global, "gRPC TLS certificate and key files exist, will use them.")
	return nil
}

func verifyCert(pemData []byte) error {
	pemBlock, _ := pem.Decode(pemData)
	if pemBlock == nil {
		return errCertDataIsNil
	}

	if pemBlock.Type != "CERTIFICATE" {
		return errCertTypeInvalid
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return fmt.Errorf("unable to parse TLS certificate: %w", err)
	}

	if time.Now().After(cert.NotAfter) {
		return errCertExpired
	}
	return nil
}

func genCert(targetDir string) error {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ecdsa private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	dnsNames := []string{host}
	if host != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gocryptotrader"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if certData == nil {
		return errors.New("cert data is nil")
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key: %w", err)
	}

	keyData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	if keyData == nil {
		return errors.New("key pem data is nil")
	}

	if err := file.Write(filepath.Join(targetDir, "key.pem"), keyData); err != nil {
		return fmt.Errorf("failed to write key.pem file: %w", err)
	}

	if err := file.Write(filepath.Join(targetDir, "cert.pem"), certData); err != nil {
		return fmt.Errorf("failed to write cert.pem file: %w", err)
	}

	log.Infof(log.Global, "gRPC TLS key.pem and cert.pem files written to %s\n", targetDir)
	return nil
}
