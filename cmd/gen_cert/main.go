package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func main() {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %s", err)
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
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if certData == nil {
		log.Fatalf("cert data is nil")
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		log.Printf("failed to marshal ECDSA private key: %s", err)
	}

	keyData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	if keyData == nil {
		log.Fatalf("key pem data is nil")
	}

	err = file.Write("key.pem", keyData)
	if err != nil {
		log.Fatalf("failed to write key.pem file %s", err)
	}
	log.Printf("wrote key.pem file")

	err = file.Write("cert.pem", certData)
	if err != nil {
		log.Fatalf("failed to write cert.pem file %s", err)
	}
	log.Printf("wrote cert.pem file")

	log.Printf("testing tls.LoadX509Keypair..")
	_, err = tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("ok!")
}
