package hyperliquid

import (
	"encoding/hex"
	"math"
	"sync"
	"testing"

	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const officialSigningTestKey = "0x0123456789012345678901234567890123456789012345678901234567890123"

type dummySigningAction struct {
	Type string `msgpack:"type"`
	Num  uint64 `msgpack:"num"`
}

func TestNormaliseAddress(t *testing.T) {
	normalised, raw, err := normaliseAddress("  0x1719884EB866CB12B2287399B15F7DB5E7D775EA  ")
	require.NoError(t, err, "Normalising a valid address must not error")
	assert.Equal(t, "0x1719884eb866cb12b2287399b15f7db5e7d775ea", normalised, "Address should be lower-case")
	assert.Equal(t, "1719884eb866cb12b2287399b15f7db5e7d775ea", hex.EncodeToString(raw[:]), "Raw address should be decoded")

	normalised, _, err = normaliseAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	require.NoError(t, err, "Normalising a valid mixed-case EIP-55 address must not error")
	assert.Equal(t, "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", normalised, "Checksummed address should be normalised")

	for _, tc := range []struct {
		name    string
		address string
	}{
		{name: "missing prefix", address: "1719884eb866cb12b2287399b15f7db5e7d775ea"},
		{name: "short", address: "0x17"},
		{name: "invalid hexadecimal", address: "0xzz19884eb866cb12b2287399b15f7db5e7d775ea"},
		{name: "zero", address: "0x0000000000000000000000000000000000000000"},
		{name: "invalid checksum", address: "0x5AAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := normaliseAddress(tc.address)
			require.ErrorIs(t, err, errInvalidAddress, "Invalid address must return the expected error")
		})
	}
}

func TestParsePrivateKey(t *testing.T) {
	for _, secret := range []string{officialSigningTestKey, officialSigningTestKey[2:]} {
		key, err := parsePrivateKey(secret)
		require.NoError(t, err, "Parsing a valid private key must not error")
		assert.Equal(t, officialSigningTestKey[2:], hex.EncodeToString(key.Serialize()), "Private key should round trip")
		key.Zero()
	}

	for _, tc := range []struct {
		name   string
		secret string
	}{
		{name: "short", secret: "0x01"},
		{name: "invalid hexadecimal", secret: "0xzz23456789012345678901234567890123456789012345678901234567890123"},
		{name: "zero", secret: "0x0000000000000000000000000000000000000000000000000000000000000000"},
		{name: "curve order", secret: "0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parsePrivateKey(tc.secret)
			require.ErrorIs(t, err, errInvalidPrivateKey, "Invalid private key must return the expected error")
		})
	}
}

func TestPrivateKeyAddress(t *testing.T) {
	key, err := parsePrivateKey(officialSigningTestKey)
	require.NoError(t, err, "Parsing the official test key must not error")
	t.Cleanup(key.Zero)
	assert.Equal(t, "0x14791697260e4c9a71f18484c9f997b308e59325", privateKeyAddress(key), "Derived Ethereum address should match the official key")
}

func TestActionHash(t *testing.T) {
	action := orderAction{
		Type: "order",
		Orders: []orderWire{{
			AssetID:    4,
			IsBuy:      true,
			Price:      "1670.1",
			Size:       "0.0147",
			ReduceOnly: false,
			Type:       orderTypeWire{Limit: &limitOrderTypeWire{TimeInForce: "Ioc"}},
		}},
		Grouping: "na",
	}
	hash, err := actionHash(action, "", 1677777606040, nil)
	require.NoError(t, err, "Hashing an official order vector must not error")
	assert.Equal(t, "0fcbeda5ae3c4950a548021552a4fea2226858c4453571bf3f24ba017eac2908", hex.EncodeToString(hash[:]), "Action hash should match the official SDK")

	_, err = actionHash(make(chan int), "", 0, nil)
	require.Error(t, err, "Hashing an unsupported MessagePack value must error")
	_, err = actionHash(action, "invalid", 0, nil)
	require.ErrorIs(t, err, errInvalidAddress, "Hashing with an invalid vault must return the expected error")

	expiry := uint64(123)
	withExpiry, err := actionHash(action, "0x1719884eb866cb12b2287399b15f7db5e7d775ea", 1, &expiry)
	require.NoError(t, err, "Hashing with a vault and expiry must not error")
	assert.NotEqual(t, hash, withExpiry, "Vault and expiry should affect the action hash")
}

func TestKeccak256(t *testing.T) {
	empty := keccak256(nil)
	assert.Equal(t, "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", hex.EncodeToString(empty[:]), "Empty Keccak-256 digest should match the standard vector")
	chunked := keccak256([]byte("a"), []byte("bc"))
	assert.Equal(t, "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45", hex.EncodeToString(chunked[:]), "Chunked Keccak-256 digest should match the abc vector")
}

func TestEIP712AgentDigest(t *testing.T) {
	var connectionID [32]byte
	connectionID[0] = 1
	mainnet := eip712AgentDigest(connectionID, true)
	testnet := eip712AgentDigest(connectionID, false)
	assert.NotEqual(t, mainnet, testnet, "Mainnet and testnet sources should produce different EIP-712 digests")
	assert.Equal(t, mainnet, eip712AgentDigest(connectionID, true), "EIP-712 digest generation should be deterministic")
	assert.NotEqual(t, [32]byte{}, mainnet, "EIP-712 digest should not be empty")
}

func TestEIP712UserDigest(t *testing.T) {
	fields := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
		{Name: "enabled", Type: "bool", Value: true},
		{Name: "nonce", Type: "uint64", Value: uint64(1)},
	}
	digest, err := eip712UserDigest("TestAction", fields)
	require.NoError(t, err, "Hashing supported EIP-712 fields must not error")
	assert.NotEqual(t, [32]byte{}, digest, "User-signed EIP-712 digest should not be empty")
	assert.Equal(t, digest, mustUserDigest(t, "TestAction", fields), "User-signed EIP-712 digest generation should be deterministic")

	falseDigest, err := eip712UserDigest("TestAction", []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
		{Name: "enabled", Type: "bool", Value: false},
		{Name: "nonce", Type: "uint64", Value: uint64(1)},
	})
	require.NoError(t, err, "Hashing a false EIP-712 boolean must not error")
	assert.NotEqual(t, digest, falseDigest, "Boolean field value should affect the EIP-712 digest")

	for _, tc := range []struct {
		name       string
		primary    string
		fields     []eip712Field
		expectedIs error
	}{
		{name: "blank primary type", primary: " ", expectedIs: errEIP712PrimaryType},
		{name: "blank field name", primary: "Test", fields: []eip712Field{{Type: "string", Value: "value"}}, expectedIs: errEIP712Field},
		{name: "blank field type", primary: "Test", fields: []eip712Field{{Name: "value", Value: "value"}}, expectedIs: errEIP712Field},
		{name: "wrong string value", primary: "Test", fields: []eip712Field{{Name: "value", Type: "string", Value: true}}, expectedIs: errEIP712Field},
		{name: "wrong bool value", primary: "Test", fields: []eip712Field{{Name: "value", Type: "bool", Value: "true"}}, expectedIs: errEIP712Field},
		{name: "wrong uint64 value", primary: "Test", fields: []eip712Field{{Name: "value", Type: "uint64", Value: int64(1)}}, expectedIs: errEIP712Field},
		{name: "unsupported type", primary: "Test", fields: []eip712Field{{Name: "value", Type: "address", Value: officialSigningAddress}}, expectedIs: errEIP712Field},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := eip712UserDigest(tc.primary, tc.fields)
			require.ErrorIs(t, err, tc.expectedIs, "Invalid EIP-712 input must return the expected error")
		})
	}
}

func mustUserDigest(t *testing.T, primaryType string, fields []eip712Field) [32]byte {
	t.Helper()
	digest, err := eip712UserDigest(primaryType, fields)
	require.NoError(t, err, "Generating the expected EIP-712 digest must not error")
	return digest
}

func TestSignL1ActionOfficialVectors(t *testing.T) {
	dummy := dummySigningAction{Type: "dummy", Num: 100000000000}
	for _, tc := range []struct {
		name      string
		action    any
		vault     string
		mainnet   bool
		expectedR string
		expectedS string
		expectedV uint8
	}{
		{
			name:      "dummy mainnet",
			action:    dummy,
			mainnet:   true,
			expectedR: "0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298",
			expectedS: "0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8",
			expectedV: 27,
		},
		{
			name:      "dummy testnet",
			action:    dummy,
			expectedR: "0x542af61ef1f429707e3c76c5293c80d01f74ef853e34b76efffcb57e574f9510",
			expectedS: "0x17b8b32f086e8cdede991f1e2c529f5dd5297cbe8128500e00cbaf766204a613",
			expectedV: 28,
		},
		{
			name:      "dummy mainnet with vault",
			action:    dummy,
			vault:     "0x1719884eb866cb12b2287399b15f7db5e7d775ea",
			mainnet:   true,
			expectedR: "0x3c548db75e479f8012acf3000ca3a6b05606bc2ec0c29c50c515066a326239",
			expectedS: "0x4d402be7396ce74fbba3795769cda45aec00dc3125a984f2a9f23177b190da2c",
			expectedV: 28,
		},
		{
			name: "order mainnet",
			action: orderAction{
				Type: "order",
				Orders: []orderWire{{
					AssetID: 1, IsBuy: true, Price: "100", Size: "100",
					Type: orderTypeWire{Limit: &limitOrderTypeWire{TimeInForce: "Gtc"}},
				}},
				Grouping: "na",
			},
			mainnet:   true,
			expectedR: "0xd65369825a9df5d80099e513cce430311d7d26ddf477f5b3a33d2806b100d78e",
			expectedS: "0x2b54116ff64054968aa237c20ca9ff68000f977c93289157748a3162b6ea940e",
			expectedV: 28,
		},
		{
			name: "order mainnet with client ID",
			action: orderAction{
				Type: "order",
				Orders: []orderWire{{
					AssetID: 1, IsBuy: true, Price: "100", Size: "100",
					Type:          orderTypeWire{Limit: &limitOrderTypeWire{TimeInForce: "Gtc"}},
					ClientOrderID: "0x00000000000000000000000000000001",
				}},
				Grouping: "na",
			},
			mainnet:   true,
			expectedR: "0x41ae18e8239a56cacbc5dad94d45d0b747e5da11ad564077fcac71277a946e3",
			expectedS: "0x3c61f667e747404fe7eea8f90ab0e76cc12ce60270438b2058324681a00116da",
			expectedV: 27,
		},
		{
			name: "trigger order mainnet",
			action: orderAction{
				Type: "order",
				Orders: []orderWire{{
					AssetID: 1, IsBuy: true, Price: "100", Size: "100",
					Type: orderTypeWire{Trigger: &triggerOrderTypeWire{
						IsMarket: true, TriggerPrice: "103", TakeProfitStopLoss: "sl",
					}},
				}},
				Grouping: "na",
			},
			mainnet:   true,
			expectedR: "0x98343f2b5ae8e26bb2587daad3863bc70d8792b09af1841b6fdd530a2065a3f9",
			expectedS: "0x6b5bb6bb0633b710aa22b721dd9dee6d083646a5f8e581a20b545be6c1feb405",
			expectedV: 27,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sig, err := signL1Action(officialSigningTestKey, tc.action, tc.vault, 0, nil, tc.mainnet)
			require.NoError(t, err, "Signing an official vector must not error")
			assert.Equal(t, tc.expectedR, sig.R, "Signature R should match the official SDK")
			assert.Equal(t, tc.expectedS, sig.S, "Signature S should match the official SDK")
			assert.Equal(t, tc.expectedV, sig.V, "Signature recovery ID should match the official SDK")
		})
	}

	_, err := signL1Action("invalid", dummy, "", 0, nil, true)
	require.ErrorIs(t, err, errInvalidPrivateKey, "Signing with an invalid key must return the expected error")
	_, err = signL1Action(officialSigningTestKey, make(chan int), "", 0, nil, true)
	require.Error(t, err, "Signing an unsupported MessagePack value must error")
	expiry := uint64(1)
	_, err = signL1Action(officialSigningTestKey, dummy, "", 0, &expiry, true)
	require.NoError(t, err, "Signing with an expiry must not error")
}

// Expected signatures were generated with hyperliquid-python-sdk commit
// 2fdb18f9517675ea03695a0962bd19eece9c83f0 using its corresponding signing
// helpers; the USDC send and withdrawal cases also match tests/signing_test.py.
func TestSignUserSignedActionOfficialVectors(t *testing.T) {
	const (
		destination = "0x5e9ee1089755c3435139848e47e6635505d5a13a"
		nonce       = uint64(1687816341423)
	)
	for _, tc := range []struct {
		name        string
		primaryType string
		fields      []eip712Field
		expectedR   string
		expectedS   string
		expectedV   uint8
	}{
		{
			name:        "USDC send testnet",
			primaryType: "HyperliquidTransaction:UsdSend",
			fields: []eip712Field{
				{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
				{Name: "destination", Type: "string", Value: destination},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: nonce},
			},
			expectedR: "0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073",
			expectedS: "0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7",
			expectedV: 27,
		},
		{
			name:        "bridge withdrawal testnet",
			primaryType: "HyperliquidTransaction:Withdraw",
			fields: []eip712Field{
				{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
				{Name: "destination", Type: "string", Value: destination},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: nonce},
			},
			expectedR: "0x8363524c799e90ce9bc41022f7c39b4e9bdba786e5f9c72b20e43e1462c37cf9",
			expectedS: "0x58b1411a775938b83e29182e8ef74975f9054c8e97ebf5ec2dc8d51bfc893881",
			expectedV: 28,
		},
		{
			name:        "spot send mainnet",
			primaryType: "HyperliquidTransaction:SpotSend",
			fields: []eip712Field{
				{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
				{Name: "destination", Type: "string", Value: destination},
				{Name: "token", Type: "string", Value: "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: nonce},
			},
			expectedR: "0xa5ed072c26df4148b6a74763648424ebb6e4789bb6cc4660f7d13fb5f86e03d9",
			expectedS: "0x212b2040f67b118a112c7ce23db6f3c249aac6047741331042fe9ce4c52e94c7",
			expectedV: 27,
		},
		{
			name:        "class transfer testnet",
			primaryType: "HyperliquidTransaction:UsdClassTransfer",
			fields: []eip712Field{
				{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
				{Name: "amount", Type: "string", Value: "1.23"},
				{Name: "toPerp", Type: "bool", Value: true},
				{Name: "nonce", Type: "uint64", Value: nonce},
			},
			expectedR: "0x421ad83297bc324d2913c7c8447b19f406dcaf1015cb76e1407b95f6f37999bf",
			expectedS: "0x29631aafeb0d5a09801204c1284d6be70ce7484ae79250dee012f2d947169d41",
			expectedV: 27,
		},
		{
			name:        "asset transfer mainnet",
			primaryType: "HyperliquidTransaction:SendAsset",
			fields: []eip712Field{
				{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
				{Name: "destination", Type: "string", Value: destination},
				{Name: "sourceDex", Type: "string", Value: ""},
				{Name: "destinationDex", Type: "string", Value: "xyz"},
				{Name: "token", Type: "string", Value: "USDC"},
				{Name: "amount", Type: "string", Value: "1.23"},
				{Name: "fromSubAccount", Type: "string", Value: ""},
				{Name: "nonce", Type: "uint64", Value: nonce},
			},
			expectedR: "0x56e1c7e87d76aff976aa748f539887ce9b7a7b08c2ced8be795a039c50a2c1d9",
			expectedS: "0x38b5d1800c2ee3caa24d7dacaea7745e0c4e6d42ac03d44fbfb38830ba5b0258",
			expectedV: 27,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			signature, err := signUserSignedAction(officialSigningTestKey, tc.primaryType, tc.fields)
			require.NoError(t, err, "Signing an official user-action vector must not error")
			assert.Equal(t, tc.expectedR, signature.R, "Signature R should match the official SDK")
			assert.Equal(t, tc.expectedS, signature.S, "Signature S should match the official SDK")
			assert.Equal(t, tc.expectedV, signature.V, "Signature recovery ID should match the official SDK")
		})
	}

	_, err := signUserSignedAction("invalid", "Test", nil)
	require.ErrorIs(t, err, errInvalidPrivateKey, "User signing with an invalid key must return the expected error")
	_, err = signUserSignedAction(officialSigningTestKey, "", nil)
	require.ErrorIs(t, err, errEIP712PrimaryType, "User signing with an invalid primary type must return the expected error")
}

func TestValidateCompactSignature(t *testing.T) {
	key, err := parsePrivateKey(officialSigningTestKey)
	require.NoError(t, err, "Parsing the expected signing key must not error")
	t.Cleanup(key.Zero)
	digest := keccak256([]byte("compact-signature-test"))

	for _, compact := range [][]byte{
		nil,
		append([]byte{ethereumSignatureVOffset - 1}, make([]byte, signatureComponentLength*2)...),
		append([]byte{ethereumSignatureVOffset + 2}, make([]byte, signatureComponentLength*2)...),
	} {
		_, err := validateCompactSignature(key, digest, compact)
		require.ErrorIs(t, err, errInvalidRecoveryID, "Malformed compact signature must return the expected recovery-ID error")
	}

	invalidSignature := append([]byte{ethereumSignatureVOffset}, make([]byte, signatureComponentLength*2)...)
	_, err = validateCompactSignature(key, digest, invalidSignature)
	require.Error(t, err, "Invalid signature scalars must fail recovery")

	otherKey, err := parsePrivateKey("0x1123456789012345678901234567890123456789012345678901234567890123")
	require.NoError(t, err, "Parsing the alternate signing key must not error")
	t.Cleanup(otherKey.Zero)
	_, err = validateCompactSignature(key, digest, secpECDSA.SignCompact(otherKey, digest[:], false))
	require.ErrorIs(t, err, errSigningRecoveryMismatch, "Signature from another key must return the recovery mismatch")

	result, err := validateCompactSignature(key, digest, secpECDSA.SignCompact(key, digest[:], false))
	require.NoError(t, err, "Valid compact signature must not error")
	assert.NotEmpty(t, result.R, "Valid compact signature should include R")
	assert.NotEmpty(t, result.S, "Valid compact signature should include S")
	assert.Contains(t, []uint8{27, 28}, result.V, "Valid compact signature should include an Ethereum recovery ID")
}

func TestFormatSignatureComponent(t *testing.T) {
	assert.Equal(t, "0x0", formatSignatureComponent(nil), "Empty signature component should encode as zero")
	assert.Equal(t, "0x0", formatSignatureComponent([]byte{0}), "Zero signature component should encode as zero")
	assert.Equal(t, "0xa", formatSignatureComponent([]byte{0, 0x0a}), "Leading zeroes should be removed from a signature component")
	assert.Equal(t, "0x10", formatSignatureComponent([]byte{0x10}), "Non-zero signature component should retain its hexadecimal value")
}

func TestFloatToWire(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    float64
		expected string
		errorIs  error
	}{
		{name: "integer", value: 100, expected: "100"},
		{name: "fraction", value: 0.0147, expected: "0.0147"},
		{name: "trailing zeros", value: 1670.1, expected: "1670.1"},
		{name: "negative zero", value: math.Copysign(0, -1), expected: "0"},
		{name: "too precise", value: 0.000012312312, errorIs: errWireNumberRounding},
		{name: "nan", value: math.NaN(), errorIs: errWireNumberRounding},
		{name: "infinity", value: math.Inf(1), errorIs: errWireNumberRounding},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := floatToWire(tc.value)
			require.ErrorIs(t, err, tc.errorIs, "Formatting a wire number must return the expected error")
			assert.Equal(t, tc.expected, wire, "Wire number should match")
		})
	}
}

func TestNextNonce(t *testing.T) {
	ex := new(Exchange)
	first := ex.nextNonce()
	second := ex.nextNonce()
	assert.Greater(t, second, first, "Sequential nonces should be strictly increasing")

	const goroutines = 32
	nonces := make(chan uint64, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			nonces <- ex.nextNonce()
		}()
	}
	wg.Wait()
	close(nonces)
	seen := make(map[uint64]struct{}, goroutines)
	for nonce := range nonces {
		_, exists := seen[nonce]
		assert.False(t, exists, "Concurrent nonces should be unique")
		seen[nonce] = struct{}{}
	}
	assert.Len(t, seen, goroutines, "Every concurrent call should return one nonce")
}
