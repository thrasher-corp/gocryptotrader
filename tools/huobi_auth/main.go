package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
)

func encodePEM(privKey *ecdsa.PrivateKey, pubKey bool) ([]byte, error) {
	if !pubKey {
		x509Enc, err := x509.MarshalECPrivateKey(privKey)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(
			&pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: x509Enc,
			},
		), nil
	}

	publicKey := &privKey.PublicKey
	x509EncPub, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PUBLIC KEY",
			Bytes: x509EncPub,
		},
	), nil
}

func decodePEM(PEMPrivKey, PEMPubKey []byte) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(PEMPrivKey))
	if block == nil {
		return nil, nil, errors.New("priv block data is nil")
	}

	x509Enc := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Enc)
	if err != nil {
		return nil, nil, err
	}

	blockPub, _ := pem.Decode([]byte(PEMPubKey))
	if block == nil {
		return nil, nil, errors.New("pub block data is nil")
	}

	x509EncPub := blockPub.Bytes
	genPubkey, err := x509.ParsePKIXPublicKey(x509EncPub)
	if err != nil {
		return nil, nil, err
	}

	publicKey := genPubkey.(*ecdsa.PublicKey)

	return privateKey, publicKey, nil
}

func writeFile(file string, data []byte) error {
	return common.WriteFile(file, data)
}

func main() {
	genKeys := false
	privKeyData, err := common.ReadFile("privatekey.pem")
	if err != nil {
		genKeys = true
	}

	log.Printf("Generating keys: %v.", genKeys)
	var privKey, pubKey []byte

	PEMEncoder := func(p *ecdsa.PrivateKey) {
		privKey, err = encodePEM(p, false)
		if err != nil {
			log.Fatal(err)
		}

		pubKey, err = encodePEM(p, true)
		if err != nil {
			log.Fatal(err)
		}
	}

	if genKeys {
		pKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Fatal(err)
		}

		PEMEncoder(pKey)

		err = writeFile("privatekey.pem", privKey)
		if err != nil {
			log.Fatal(err)
		}

		err = writeFile("publickey.pem", pubKey)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		pubKeyData, err := common.ReadFile("publickey.pem")
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Successfully read PEM files.")

		priv, _, err := decodePEM(privKeyData, pubKeyData)
		if err != nil {
			log.Fatal(err)
		}

		PEMEncoder(priv)

		if !bytes.Equal(privKey, privKeyData) {
			log.Fatalf("PEM privkey mismatch")
		}

		if !bytes.Equal(pubKey, pubKeyData) {
			log.Fatalf("PEM pubkey mismatch")
		}

		log.Println("PEM data matches.")
	}

	fmt.Println()
	fmt.Println(string(privKey))
	fmt.Println(string(pubKey))

	type JSONGeneration struct {
		APIAuthPEMKey string
	}

	r := JSONGeneration{
		APIAuthPEMKey: string(privKey),
	}

	resultk, err := common.JSONEncode(r)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Please visit https://github.com/huobiapi/API_Docs_en/wiki/Signing_API_Requests and follow from step 2 onwards.")
	log.Printf("After completing the above instructions, please copy and paste the below key (including the following ',') into your Huobi exchange config file:\n\n")
	fmt.Println(string(resultk[1:len(resultk)-1]) + ",")
}
