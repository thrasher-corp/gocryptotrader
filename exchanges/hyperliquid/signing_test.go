package hyperliquid

import (
	"encoding/hex"
	"math"
	"testing"

	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// vectorNonce is the nonce of the derived signing vectors
	vectorNonce = 1700000000000
	// testZeroAddress is the zero EVM address, which info requests can query but which must never receive funds or
	// authority
	testZeroAddress = "0x0000000000000000000000000000000000000000"
)

// The TypeScript and Rust SDKs' published test keys, which only sign their test vectors
const (
	typeScriptSDKTestKey = "0x822e9959e022b78423eb653a62ea0020cd283e71a2a8133a6ff2aeffaf373cff"
	rustSDKTestKey       = "0xe908f86dbb4d55ac876378565aafeabc187f6690f046459397b17d9b9a19688e"
)

// dummySigningAction is the Python SDK's dummy action for its L1 signing test vectors
type dummySigningAction struct {
	Type string `msgpack:"type"`
	Num  uint64 `msgpack:"num"`
}

func TestNormaliseAddress(t *testing.T) {
	t.Parallel()
	normalised, raw, err := normaliseAddress("  0x1719884EB866CB12B2287399B15F7DB5E7D775EA  ")
	require.NoError(t, err, "normaliseAddress must not error for an upper-case address")
	assert.Equal(t, "0x1719884eb866cb12b2287399b15f7db5e7d775ea", normalised, "normaliseAddress should lower-case the address")
	assert.Equal(t, "1719884eb866cb12b2287399b15f7db5e7d775ea", hex.EncodeToString(raw[:]), "normaliseAddress should decode the raw address")

	normalised, _, err = normaliseAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	require.NoError(t, err, "normaliseAddress must not error for a valid EIP-55 address")
	assert.Equal(t, "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", normalised, "normaliseAddress should lower-case a checksummed address")

	normalised, raw, err = normaliseAddress(testZeroAddress)
	require.NoError(t, err, "normaliseAddress must not error for the zero address, which info requests can query")
	assert.Equal(t, testZeroAddress, normalised, "normaliseAddress should return the zero address")
	assert.Zero(t, raw, "normaliseAddress should decode the zero address")

	for _, address := range []string{
		"1719884eb866cb12b2287399b15f7db5e7d775ea",
		"0x17",
		"0xzz19884eb866cb12b2287399b15f7db5e7d775ea",
		"0x5AAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
	} {
		_, _, err := normaliseAddress(address)
		assert.ErrorIsf(t, err, errInvalidAddress, "normaliseAddress should reject %q", address)
	}
}

func TestNormaliseRecipientAddress(t *testing.T) {
	t.Parallel()
	normalised, err := normaliseRecipientAddress("0x1719884EB866CB12B2287399B15F7DB5E7D775EA")
	require.NoError(t, err, "normaliseRecipientAddress must not error for a valid address")
	assert.Equal(t, "0x1719884eb866cb12b2287399b15f7db5e7d775ea", normalised, "normaliseRecipientAddress should lower-case the address")
	for _, address := range []string{"0x17", testZeroAddress} {
		_, err := normaliseRecipientAddress(address)
		assert.ErrorIsf(t, err, errInvalidAddress, "normaliseRecipientAddress should reject %q", address)
	}
}

func TestParsePrivateKey(t *testing.T) {
	t.Parallel()
	for _, secret := range []string{testPrivateKey, testPrivateKey[2:], "0X" + testPrivateKey[2:]} {
		key, err := parsePrivateKey(secret)
		require.NoErrorf(t, err, "parsePrivateKey must not error for %q", secret)
		assert.Equal(t, testPrivateKey[2:], hex.EncodeToString(key.Serialize()), "parsePrivateKey should round trip the key")
		key.Zero()
	}
	// The errors quote no part of a key, as a mistyped key is mostly the account's real one
	const notHexadecimal, outOfRange = "invalid EVM private key: expected 32-byte hexadecimal scalar", "invalid EVM private key: scalar is outside secp256k1 range"
	for secret, exp := range map[string]string{
		"0x01": notHexadecimal,
		"0xzz23456789012345678901234567890123456789012345678901234567890123": notHexadecimal,
		"0xQ" + testPrivateKey[3:]: notHexadecimal,
		"0x0000000000000000000000000000000000000000000000000000000000000000": outOfRange,
		"0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141": outOfRange,
	} {
		_, err := parsePrivateKey(secret)
		assert.ErrorIsf(t, err, errInvalidPrivateKey, "parsePrivateKey should reject %q", secret)
		assert.EqualErrorf(t, err, exp, "parsePrivateKey should reject %q with a fixed message", secret)
	}
}

func TestKeccak256(t *testing.T) {
	t.Parallel()
	empty := keccak256(nil)
	assert.Equal(t, "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", hex.EncodeToString(empty[:]), "keccak256 should match the empty input vector")
	chunked := keccak256([]byte("a"), []byte("bc"))
	assert.Equal(t, "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45", hex.EncodeToString(chunked[:]), "keccak256 should hash chunks as one abc input")
}

func TestPrivateKeyAddress(t *testing.T) {
	t.Parallel()
	key, err := parsePrivateKey(testPrivateKey)
	require.NoError(t, err, "parsePrivateKey must not error")
	t.Cleanup(key.Zero)
	assert.Equal(t, testAccountAddress, privateKeyAddress(key), "privateKeyAddress should derive the test key's address")
}

func TestMsgpackAction(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		action any
		exp    string
	}{
		{
			name:   "a fast cancel",
			action: CancelAction{Type: "cancel", Cancels: []CancelRequest{{Asset: 0, OrderID: 123456789}}, Fast: true},
			exp:    "83a474797065a663616e63656ca763616e63656c739182a16100a16fce075bcd15a166c3",
		},
		{
			name:   "an order with a priority grouping",
			action: OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}}}}, Grouping: PriorityGroupingWire{PriorityRate: 12345}},
			exp:    "83a474797065a56f72646572a66f72646572739186a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3496f63a867726f7570696e6781a170cd3039",
		},
		{
			name:   "a merge of the maximum",
			action: UserOutcomeAction{Type: "userOutcome", MergeOutcome: &MergeOutcomeWire{Outcome: 1}},
			exp:    "82a474797065ab757365724f7574636f6d65ac6d657267654f7574636f6d6582a76f7574636f6d6501a6616d6f756e74c0",
		},
	} {
		encoded, err := msgpackAction(tc.action)
		require.NoErrorf(t, err, "msgpackAction must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, hex.EncodeToString(encoded), "msgpackAction should encode %s in Hyperliquid's key order with compact integers", tc.name)
	}
	_, err := msgpackAction(make(chan int))
	assert.ErrorContains(t, err, "encoding action as msgpack", "msgpackAction should report a value msgpack cannot encode")
}

func TestActionHash(t *testing.T) {
	t.Parallel()
	action := OrderAction{
		Type: "order",
		Orders: []OrderWire{{
			Asset: 4,
			IsBuy: true,
			Price: "1670.1",
			Size:  "0.0147",
			Type:  OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}},
		}},
		Grouping: GroupingNone,
	}
	hash, err := actionHash(action, "", 1677777606040, nil)
	require.NoError(t, err, "actionHash must not error")
	assert.Equal(t, "0fcbeda5ae3c4950a548021552a4fea2226858c4453571bf3f24ba017eac2908", hex.EncodeToString(hash[:]), "actionHash should match the Python SDK's order hash")

	_, err = actionHash(make(chan int), "", 0, nil)
	assert.Error(t, err, "actionHash should error for a value msgpack cannot encode")
	_, err = actionHash(action, "invalid", 0, nil)
	assert.ErrorIs(t, err, errInvalidAddress, "actionHash should reject an invalid vault address")

	withVault, err := actionHash(action, "0x1719884eb866cb12b2287399b15f7db5e7d775ea", 1677777606040, nil)
	require.NoError(t, err, "actionHash must not error with a vault")
	assert.NotEqual(t, hash, withVault, "actionHash should include the vault address")
	withExpiry, err := actionHash(action, "", 1677777606040, new(uint64(1677777666040)))
	require.NoError(t, err, "actionHash must not error with an expiry")
	assert.NotEqual(t, hash, withExpiry, "actionHash should include the expiry")
}

func TestEIP712DomainHash(t *testing.T) {
	t.Parallel()
	exchangeDomain := eip712DomainHash("Exchange", eip712ChainID)
	assert.Equal(t, exchangeDomain, eip712DomainHash("Exchange", eip712ChainID), "eip712DomainHash should be deterministic")
	assert.NotEqual(t, exchangeDomain, eip712DomainHash("HyperliquidSignTransaction", eip712ChainID), "eip712DomainHash should include the domain name")
	assert.NotEqual(t, exchangeDomain, eip712DomainHash("Exchange", userSignedChainID), "eip712DomainHash should include the chain ID")
}

func TestEIP712AgentDigest(t *testing.T) {
	t.Parallel()
	var connectionID [32]byte
	connectionID[0] = 1
	mainnet := eip712AgentDigest(connectionID, true)
	assert.NotEqual(t, mainnet, eip712AgentDigest(connectionID, false), "eip712AgentDigest should differ between mainnet and testnet")
	assert.Equal(t, mainnet, eip712AgentDigest(connectionID, true), "eip712AgentDigest should be deterministic")
	assert.NotEqual(t, mainnet, eip712AgentDigest([32]byte{}, true), "eip712AgentDigest should include the connection ID")
}

func TestEIP712UserDigest(t *testing.T) {
	t.Parallel()
	fields := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
		{Name: "enabled", Type: "bool", Value: true},
		{Name: "nonce", Type: "uint64", Value: uint64(1)},
	}
	digest, err := eip712UserDigest("TestAction", fields)
	require.NoError(t, err, "eip712UserDigest must not error for supported fields")
	assert.NotEqual(t, [32]byte{}, digest, "eip712UserDigest should not return an empty digest")
	fields[1].Value = false
	falseDigest, err := eip712UserDigest("TestAction", fields)
	require.NoError(t, err, "eip712UserDigest must not error for a false boolean")
	assert.NotEqual(t, digest, falseDigest, "eip712UserDigest should include boolean values")

	for _, tc := range []struct {
		name   string
		fields []eip712Field
	}{
		{name: "blank field name", fields: []eip712Field{{Type: "string", Value: "value"}}},
		{name: "blank field type", fields: []eip712Field{{Name: "value", Value: "value"}}},
		{name: "wrong string value", fields: []eip712Field{{Name: "value", Type: "string", Value: true}}},
		{name: "wrong bool value", fields: []eip712Field{{Name: "value", Type: "bool", Value: "true"}}},
		{name: "wrong uint64 value", fields: []eip712Field{{Name: "value", Type: "uint64", Value: int64(1)}}},
		{name: "wrong uint32 value", fields: []eip712Field{{Name: "value", Type: "uint32", Value: uint64(1)}}},
		{name: "wrong address value", fields: []eip712Field{{Name: "value", Type: "address", Value: []byte{1}}}},
		{name: "invalid address", fields: []eip712Field{{Name: "value", Type: "address", Value: "0x01"}}},
		{name: "wrong bytes value", fields: []eip712Field{{Name: "value", Type: "bytes", Value: []byte{1}}}},
		{name: "unprefixed bytes", fields: []eip712Field{{Name: "value", Type: "bytes", Value: "deadbeef"}}},
		{name: "invalid bytes", fields: []eip712Field{{Name: "value", Type: "bytes", Value: "0xzz"}}},
		{name: "unsupported type", fields: []eip712Field{{Name: "value", Type: "int256", Value: int64(1)}}},
	} {
		_, err := eip712UserDigest("Test", tc.fields)
		assert.ErrorIsf(t, err, errEIP712Field, "eip712UserDigest should reject %s", tc.name)
	}
	_, err = eip712UserDigest(" ", nil)
	assert.ErrorIs(t, err, errEIP712PrimaryType, "eip712UserDigest should reject a blank primary type")
}

func TestSignL1Action(t *testing.T) {
	t.Parallel()
	dummy := dummySigningAction{Type: "dummy", Num: 100000000000}
	for _, tc := range []struct {
		name    string
		action  any
		vault   string
		mainnet bool
		exp     ActionSignature
	}{
		{
			name:    "dummy mainnet",
			action:  dummy,
			mainnet: true,
			exp: ActionSignature{
				R: "0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298",
				S: "0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8",
				V: 27,
			},
		},
		{
			name:   "dummy testnet",
			action: dummy,
			exp: ActionSignature{
				R: "0x542af61ef1f429707e3c76c5293c80d01f74ef853e34b76efffcb57e574f9510",
				S: "0x17b8b32f086e8cdede991f1e2c529f5dd5297cbe8128500e00cbaf766204a613",
				V: 28,
			},
		},
		{
			name:    "dummy mainnet with vault",
			action:  dummy,
			vault:   "0x1719884eb866cb12b2287399b15f7db5e7d775ea",
			mainnet: true,
			exp: ActionSignature{
				R: "0x3c548db75e479f8012acf3000ca3a6b05606bc2ec0c29c50c515066a326239",
				S: "0x4d402be7396ce74fbba3795769cda45aec00dc3125a984f2a9f23177b190da2c",
				V: 28,
			},
		},
		{
			name: "order mainnet",
			action: OrderAction{
				Type:     "order",
				Orders:   []OrderWire{{Asset: 1, IsBuy: true, Price: "100", Size: "100", Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}},
				Grouping: GroupingNone,
			},
			mainnet: true,
			exp: ActionSignature{
				R: "0xd65369825a9df5d80099e513cce430311d7d26ddf477f5b3a33d2806b100d78e",
				S: "0x2b54116ff64054968aa237c20ca9ff68000f977c93289157748a3162b6ea940e",
				V: 28,
			},
		},
		{
			name: "order mainnet with client order ID",
			action: OrderAction{
				Type: "order",
				Orders: []OrderWire{{
					Asset:         1,
					IsBuy:         true,
					Price:         "100",
					Size:          "100",
					Type:          OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}},
					ClientOrderID: "0x00000000000000000000000000000001",
				}},
				Grouping: GroupingNone,
			},
			mainnet: true,
			exp: ActionSignature{
				R: "0x41ae18e8239a56cacbc5dad94d45d0b747e5da11ad564077fcac71277a946e3",
				S: "0x3c61f667e747404fe7eea8f90ab0e76cc12ce60270438b2058324681a00116da",
				V: 27,
			},
		},
		{
			name: "trigger order mainnet",
			action: OrderAction{
				Type: "order",
				Orders: []OrderWire{{
					Asset: 1,
					IsBuy: true,
					Price: "100",
					Size:  "100",
					Type:  OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "103", TakeProfitStopLoss: TriggerStopLoss}},
				}},
				Grouping: GroupingNone,
			},
			mainnet: true,
			exp: ActionSignature{
				R: "0x98343f2b5ae8e26bb2587daad3863bc70d8792b09af1841b6fdd530a2065a3f9",
				S: "0x6b5bb6bb0633b710aa22b721dd9dee6d083646a5f8e581a20b545be6c1feb405",
				V: 27,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			signature, err := signL1Action(testPrivateKey, tc.action, tc.vault, 0, nil, tc.mainnet)
			require.NoError(t, err, "signL1Action must not error")
			assert.Equal(t, tc.exp, signature, "signL1Action should match the Python SDK's signature")
		})
	}

	_, err := signL1Action("invalid", dummy, "", 0, nil, true)
	assert.ErrorIs(t, err, errInvalidPrivateKey, "signL1Action should reject an invalid key")
	_, err = signL1Action(testPrivateKey, make(chan int), "", 0, nil, true)
	assert.Error(t, err, "signL1Action should error for a value msgpack cannot encode")
	_, err = signL1Action(testPrivateKey, dummy, "", 0, new(uint64(1)), true)
	assert.NoError(t, err, "signL1Action should sign with an expiry")
}

// Expected signatures come from hyperliquid-python-sdk commit 2fdb18f9517675ea03695a0962bd19eece9c83f0 and its signing
// helpers; the USDC send and withdrawal vectors also match its tests/signing_test.py
func TestSignUserSignedAction(t *testing.T) {
	t.Parallel()
	const (
		destination = "0x5e9ee1089755c3435139848e47e6635505d5a13a"
		nonce       = uint64(1687816341423)
	)
	for _, tc := range []struct {
		name        string
		primaryType string
		fields      []eip712Field
		exp         ActionSignature
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
			exp: ActionSignature{
				R: "0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073",
				S: "0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7",
				V: 27,
			},
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
			exp: ActionSignature{
				R: "0x8363524c799e90ce9bc41022f7c39b4e9bdba786e5f9c72b20e43e1462c37cf9",
				S: "0x58b1411a775938b83e29182e8ef74975f9054c8e97ebf5ec2dc8d51bfc893881",
				V: 28,
			},
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
			exp: ActionSignature{
				R: "0xa5ed072c26df4148b6a74763648424ebb6e4789bb6cc4660f7d13fb5f86e03d9",
				S: "0x212b2040f67b118a112c7ce23db6f3c249aac6047741331042fe9ce4c52e94c7",
				V: 27,
			},
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
			exp: ActionSignature{
				R: "0x421ad83297bc324d2913c7c8447b19f406dcaf1015cb76e1407b95f6f37999bf",
				S: "0x29631aafeb0d5a09801204c1284d6be70ce7484ae79250dee012f2d947169d41",
				V: 27,
			},
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
			exp: ActionSignature{
				R: "0x56e1c7e87d76aff976aa748f539887ce9b7a7b08c2ced8be795a039c50a2c1d9",
				S: "0x38b5d1800c2ee3caa24d7dacaea7745e0c4e6d42ac03d44fbfb38830ba5b0258",
				V: 27,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			signature, err := signUserSignedAction(testPrivateKey, tc.primaryType, tc.fields)
			require.NoError(t, err, "signUserSignedAction must not error")
			assert.Equal(t, tc.exp, signature, "signUserSignedAction should match the Python SDK's signature")
		})
	}

	_, err := signUserSignedAction("invalid", "Test", nil)
	assert.ErrorIs(t, err, errInvalidPrivateKey, "signUserSignedAction should reject an invalid key")
	_, err = signUserSignedAction(testPrivateKey, "", nil)
	assert.ErrorIs(t, err, errEIP712PrimaryType, "signUserSignedAction should reject an empty primary type")
}

func TestValidateCompactSignature(t *testing.T) {
	t.Parallel()
	key, err := parsePrivateKey(testPrivateKey)
	require.NoError(t, err, "parsePrivateKey must not error")
	t.Cleanup(key.Zero)
	digest := keccak256([]byte("compact-signature-test"))

	for _, compact := range [][]byte{
		nil,
		append([]byte{ethereumSignatureVOffset - 1}, make([]byte, signatureComponentLength*2)...),
		append([]byte{ethereumSignatureVOffset + 2}, make([]byte, signatureComponentLength*2)...),
	} {
		_, err := validateCompactSignature(key, digest, compact)
		assert.ErrorIs(t, err, errInvalidRecoveryID, "validateCompactSignature should reject a malformed recovery ID or length")
	}
	_, err = validateCompactSignature(key, digest, append([]byte{ethereumSignatureVOffset}, make([]byte, signatureComponentLength*2)...))
	assert.ErrorContains(t, err, "recovering signature key", "validateCompactSignature should report zero scalars that cannot recover a key")

	otherKey, err := parsePrivateKey("0x1123456789012345678901234567890123456789012345678901234567890123")
	require.NoError(t, err, "parsePrivateKey must not error for the other key")
	t.Cleanup(otherKey.Zero)
	_, err = validateCompactSignature(key, digest, secpECDSA.SignCompact(otherKey, digest[:], false))
	assert.ErrorIs(t, err, errSigningRecoveryMismatch, "validateCompactSignature should reject another key's signature")

	signature, err := validateCompactSignature(key, digest, secpECDSA.SignCompact(key, digest[:], false))
	require.NoError(t, err, "validateCompactSignature must not error for the key's own signature")
	assert.NotEmpty(t, signature.R, "validateCompactSignature should return R")
	assert.NotEmpty(t, signature.S, "validateCompactSignature should return S")
	assert.Contains(t, []uint8{27, 28}, signature.V, "validateCompactSignature should return an Ethereum recovery ID")
}

func TestFormatSignatureComponent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		component []byte
		exp       string
	}{
		{exp: "0x0"},
		{component: []byte{0}, exp: "0x0"},
		{component: []byte{0, 0x0a}, exp: "0xa"},
		{component: []byte{0x10}, exp: "0x10"},
	} {
		assert.Equalf(t, tc.exp, formatSignatureComponent(tc.component), "formatSignatureComponent should format %x", tc.component)
	}
}

func TestFloatToWire(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value float64
		exp   string
		err   error
	}{
		{value: 100, exp: "100"},
		{value: 0.0147, exp: "0.0147"},
		{value: 1670.1, exp: "1670.1"},
		{value: math.Copysign(0, -1), exp: "0"},
		{value: 0.000012312312, err: errWireNumberRounding},
		{value: 1e-13, err: errWireNumberRounding},
		{value: -1e-13, err: errWireNumberRounding},
		{value: math.NaN(), err: errWireNumberRounding},
		{value: math.Inf(1), err: errWireNumberRounding},
	} {
		wire, err := floatToWire(tc.value)
		require.ErrorIsf(t, err, tc.err, "floatToWire must return the expected error for %v", tc.value)
		assert.Equalf(t, tc.exp, wire, "floatToWire should format %v", tc.value)
	}
}

// TestL1ActionVectors checks each L1 action's msgpack encoding, connection ID and signatures against vectors derived
// from an independent implementation of the SDK signing, which reproduces every published SDK vector, and against the
// published Python, TypeScript and Rust SDK vectors themselves
func TestL1ActionVectors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		key          string
		nonce        uint64
		action       any
		vault        string
		expiresAfter *uint64
		msgpack      string
		connectionID string
		mainnet      ActionSignature
		testnet      ActionSignature
	}{
		{
			name:         "order_builder",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone, Builder: &BuilderFeeWire{Builder: "0x1234567890123456789012345678901234567890", Fee: 10}},
			msgpack:      "84a474797065a56f72646572a66f72646572739186a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3477463a867726f7570696e67a26e61a76275696c64657282a162d92a307831323334353637383930313233343536373839303132333435363738393031323334353637383930a1660a",
			connectionID: "0x6a4bf8fb4b5da3b97a5a53c254f5ce2bbedce99b68b81524128e037954587cf8",
			mainnet:      ActionSignature{R: "0xbbb56c7e95d8eacd5ab2f7478f8e84f233c3499eb05fab6c82b83a49f251c934", S: "0x12bf30013b8f2e0de0810cfd5bb75b45d5d91a683e23c7ed8ee41fd2fa5ffb5c", V: 27},
			testnet:      ActionSignature{R: "0x9ea53271686294a01fa2e300737d88e7b6bfefa431ad2e4e528e50dbc3abde25", S: "0x4ddd6d01304362ef14ae69753d9ddf1f0ee94d66db2ce96679cbf9800c07c88", V: 28},
		},
		{
			name:         "order_priority",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}}}}, Grouping: PriorityGroupingWire{PriorityRate: 12345}},
			msgpack:      "83a474797065a56f72646572a66f72646572739186a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3496f63a867726f7570696e6781a170cd3039",
			connectionID: "0x3adaa78e27a561cbbdf2c6fabc6d19a37b301c2996d3f95cceb8a899ad72301f",
			mainnet:      ActionSignature{R: "0x54a8dda05ba2fcc1a0e31ef1e03f273f0b4b47de959be460e4fa1c8bdf5e3677", S: "0x2df0cfbed50a79d94a63a2f839448d309e9ef1f68a5473ce31a5561ad97d8154", V: 27},
			testnet:      ActionSignature{R: "0xe49a102825ef48265d54ac0e0e5ee9bd80d340979f6d7a4297d6975a70ba73a3", S: "0x164867551f59627354af97cf80b8b7f4015b47da0ef105649685619ec889f6b6", V: 28},
		},
		{
			name:         "order_normalTpsl",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}, {Asset: 0, IsBuy: false, Price: "29000", Size: "0.1", ReduceOnly: true, Type: OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "29000", TakeProfitStopLoss: TriggerStopLoss}}}}, Grouping: GroupingNormalTPSL},
			msgpack:      "83a474797065a56f72646572a66f72646572739286a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a347746386a16100a162c2a170a53239303030a173a3302e31a172c3a17481a77472696767657283a869734d61726b6574c3a9747269676765725078a53239303030a47470736ca2736ca867726f7570696e67aa6e6f726d616c5470736c",
			connectionID: "0xfae54f6852d5079e079de1e37fc44e772d8dcf3ece231fed8169bfac670981ff",
			mainnet:      ActionSignature{R: "0xb4969077dd0772163d80e6657e268643992060693fb7115711a2dcfef697ac7b", S: "0x4f450e95277a8d15e05c86db017b60f92304426ed5a3dc924cf1361a5482a061", V: 27},
			testnet:      ActionSignature{R: "0xf09672765a49fccfe71c09cd7487ec4d8f32c40cccc57a6bdbf517ff6110ebe6", S: "0x73bbdd942defc7f411a966fe9a1da0d852f545c16db134ead17cf5a7f1b776e4", V: 28},
		},
		{
			name:         "order_vault_expiresAfter",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			vault:        "0x1719884eb866cb12b2287399b15f7db5e7d775ea",
			expiresAfter: new(uint64(1700000060000)),
			msgpack:      "83a474797065a56f72646572a66f72646572739186a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3477463a867726f7570696e67a26e61",
			connectionID: "0x152ec37828070d51be50fc2b353af34e7c47a197dac7ed3ebe49ddf44863f3bd",
			mainnet:      ActionSignature{R: "0xac3e0b73697e205ab0b5a7d8585b45750f7c8a02a55c62acdde3c40982aed1f0", S: "0x3af2a62562948fef9fd9a973fcef2fbbbb8a08323b22c7112805d294f94e1ec9", V: 27},
			testnet:      ActionSignature{R: "0x84e3bb7a927b9156ea6529c05d51ed07e548de087f27547847160f41a8b77a51", S: "0x19995e6da41fe67be4cddf49c078eb48a633fc1eddbad35c5c45fa23d6bccc4f", V: 28},
		},
		{
			name:         "cancel",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       CancelAction{Type: "cancel", Cancels: []CancelRequest{{Asset: 0, OrderID: 123456789}}},
			msgpack:      "82a474797065a663616e63656ca763616e63656c739182a16100a16fce075bcd15",
			connectionID: "0xdb71705050659e5985b6f2be3929a51c19b8b8216089fb39e31a8c7e67c3cdb3",
			mainnet:      ActionSignature{R: "0xb310cce7be049df255902a1a096f2f79332c1611adcf3983151728e43c2db8a1", S: "0x313fd44c6ae32b8173269dc8597ce68e411d4ccfac3c82b73f032ad1bc52f940", V: 28},
			testnet:      ActionSignature{R: "0x7ed2cd4d56239218cefb882fee7b274f0213db230075a3869896d53170a473dc", S: "0x6f940ccac6e378821877045d863e19d785c4025f51ce3189735ef9d3dc9c7c40", V: 28},
		},
		{
			name:         "cancel_fast",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       CancelAction{Type: "cancel", Cancels: []CancelRequest{{Asset: 0, OrderID: 123456789}}, Fast: true},
			msgpack:      "83a474797065a663616e63656ca763616e63656c739182a16100a16fce075bcd15a166c3",
			connectionID: "0xb93daf4c2d4db6edd9d500100b0014daed91142e1f28d26e332a80d00b219e28",
			mainnet:      ActionSignature{R: "0xc5e834aeb89c97bdd402dd17397714cd736b1368d2261b5d67bf8bf97c7746fe", S: "0x22aaeb49baf1a1de11c606a55bb6e3c4116b769f3ab8122bdc346817af61a087", V: 28},
			testnet:      ActionSignature{R: "0xdc43951d6b22ccdc41653dde3c4a728fc7ff255e167cdf18b9ffea6e028e15a1", S: "0x4c8531c78edacfed4a9af1abf266b04e940f18b120e5989a36b3b1813b2e5938", V: 28},
		},
		{
			name:         "cancelByCloid",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       CancelByClientOrderIDAction{Type: "cancelByCloid", Cancels: []CancelByClientOrderIDRequest{{Asset: 0, ClientOrderID: "0x00000000000000000000000000000001"}}},
			msgpack:      "82a474797065ad63616e63656c4279436c6f6964a763616e63656c739182a5617373657400a5636c6f6964d92230783030303030303030303030303030303030303030303030303030303030303031",
			connectionID: "0x1d854324d277d0ec7ab94e7d490f930b8cabb29049f2563b612afb8d0394cd50",
			mainnet:      ActionSignature{R: "0x80bc61ec5a6bd1931885c88ccb664ac5543ba861475262636445d21263b97318", S: "0x6bbfac7cec115672a533ce377a822433f48a320f1e3c3ce09d2f38184f82979c", V: 27},
			testnet:      ActionSignature{R: "0x2cd955fece7370ae4966770d9cc84dd210fdb0be55c76990dac1052e9cd76545", S: "0x184f430d6943231a58efaf738b7f8fe0f67bf33bf953f14292549e5fffe91759", V: 27},
		},
		{
			name:         "cancelByCloid_fast",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       CancelByClientOrderIDAction{Type: "cancelByCloid", Cancels: []CancelByClientOrderIDRequest{{Asset: 0, ClientOrderID: "0x00000000000000000000000000000001"}}, Fast: true},
			msgpack:      "83a474797065ad63616e63656c4279436c6f6964a763616e63656c739182a5617373657400a5636c6f6964d92230783030303030303030303030303030303030303030303030303030303030303031a166c3",
			connectionID: "0x1f3aa67ee508060b823ea4f51cdaff96051c8d60dc4d25609bcdf8c875502ba5",
			mainnet:      ActionSignature{R: "0x2ce4bd96cb7f170352646f63b557e413203914f9e2710258ae709ef284e4d72e", S: "0x75afc49b505452b8cd5a43422e3eb2835c470a0273bcf6108e3075a9d370a0a4", V: 27},
			testnet:      ActionSignature{R: "0x66fff04e93ee80dde4b9deefd343154d63e5b0f0018a82bc173077fec9553dc6", S: "0x42bbc97813b56a45301444c72136eeb850e16404ec289ec15701b7826a30ecf7", V: 28},
		},
		{
			name:         "scheduleCancel_time",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       ScheduleCancelAction{Type: "scheduleCancel", Time: new(uint64(1700000060000))},
			msgpack:      "82a474797065ae7363686564756c6543616e63656ca474696d65cf0000018bcfe65260",
			connectionID: "0x543d84c42876132614aca907b0dbd95399b942229d79e0f1c5eb81f1592e2ddf",
			mainnet:      ActionSignature{R: "0x63f82e27ce8ba37e7374cd5991215dd814a032644ed03d0480b7a6902bd0194c", S: "0x17e5777edf89919c48c2999d1c591ca7fb228cf4b8341705a4a585d1c269a990", V: 27},
			testnet:      ActionSignature{R: "0xd3ef1229f0436a45171f21af4147aa9bbe1cac764e4c27f2a41e1b22a0550ef5", S: "0x57280ba9f9786a518be0b1b009bb0ff7b8fe0ef88ddff2ac158791b2a9038f16", V: 28},
		},
		{
			name:         "modify_oid",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       ModifyAction{Type: "modify", OrderID: uint64(123456789), Order: OrderWire{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceALO}}}},
			msgpack:      "83a474797065a66d6f64696679a36f6964ce075bcd15a56f7264657286a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3416c6f",
			connectionID: "0x6354c239bb1cb32ee15f85c23aa9298f9f9c8d2c70257bdb6950ff78398b6dee",
			mainnet:      ActionSignature{R: "0xc87854743c08a4ed0716cccdae1d981f9d68dda60f9ef9225623b37061fcb871", S: "0x1302ff7454556371f39f0a7b41a2b6e25daa1a26a9d74a5b955faf19d9126134", V: 28},
			testnet:      ActionSignature{R: "0xfa6f6867e03b9946a9d39620fcdd0b216f2ad22281b511e50c2a7adcd0a1181f", S: "0x6ba364aee142f0e29ea5d83373a765e80887399d083b8f3049791b67c42553b4", V: 27},
		},
		{
			name:         "modify_cloid_alwaysPlace",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       ModifyAction{Type: "modify", OrderID: "0x00000000000000000000000000000001", Order: OrderWire{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}}, ClientOrderID: "0x00000000000000000000000000000001"}, AlwaysPlace: true},
			msgpack:      "84a474797065a66d6f64696679a36f6964d92230783030303030303030303030303030303030303030303030303030303030303031a56f7264657287a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3496f63a163d92230783030303030303030303030303030303030303030303030303030303030303031a161c3",
			connectionID: "0x8b5b599ff52b881041f2aeba365725ad0b901c23cb145075dada46a47f7c1773",
			mainnet:      ActionSignature{R: "0x8ba23c3ea72e8234623c6a7035cfe64db055425395b6fc14249d9027f7a22604", S: "0x62e011ff4d57d5b7125dc47426bebdff5aa4eb0168cbaae1af65c22125de7d77", V: 27},
			testnet:      ActionSignature{R: "0xa46f866ed0937fbae5671599157b17b47b4fac97f1f03e9b2d924dc06979245b", S: "0x67ceab895df85fdc34860a599d45b81abeaa218bb10876dd25f8c141ce6c98b2", V: 28},
		},
		{
			name:         "batchModify",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       BatchModifyAction{Type: "batchModify", Modifies: []ModifyWire{{OrderID: uint64(123456789), Order: OrderWire{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}}},
			msgpack:      "82a474797065ab62617463684d6f64696679a86d6f6469666965739182a36f6964ce075bcd15a56f7264657286a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3477463",
			connectionID: "0xfbc7796eaa5a047c9d79929b59d6ef2b6d5a8004c9c77b3806af43066f826b34",
			mainnet:      ActionSignature{R: "0xe94710aed6ee01c5c815a9a16fcca04b2675689f1ca5e139176dfad77ed8de92", S: "0x4e1c82dac52bbc90b2c20ad5926e5606a4f9757295255a42ed4abb7ee74983c7", V: 28},
			testnet:      ActionSignature{R: "0x477c2372e7f6a68c41bd9cd45f58c37af40180b8c39b846aa7dfcd3c9e6f4c61", S: "0x53c18d7fa5b613c5ef5ef1ef4bface898f906e512680971f78192623f982ed", V: 28},
		},
		{
			name:         "batchModify_alwaysPlace",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       BatchModifyAction{Type: "batchModify", Modifies: []ModifyWire{{OrderID: uint64(123456789), Order: OrderWire{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}}, AlwaysPlace: true},
			msgpack:      "83a474797065ab62617463684d6f64696679a86d6f6469666965739182a36f6964ce075bcd15a56f7264657286a16100a162c3a170a53330303030a173a3302e31a172c2a17481a56c696d697481a3746966a3477463a161c3",
			connectionID: "0xa2fda6154ccaf838d31ebb01b72f414574c6fe4d3c93d485839acb5adb02e56b",
			mainnet:      ActionSignature{R: "0x6fa3149ade9d215d96825d60ec0ff0a47bdcdbd52c6477c159a99d30596ab7b0", S: "0x566eebca57e005d1c37175d8b75ab74bebfeb9f8fa7c1136cf1afe01036864f2", V: 28},
			testnet:      ActionSignature{R: "0x2df2591abe0483f2b64fb72e19e078e2315911503d8fdb9935dc0e7412a1b508", S: "0x7bd2d06ae21e7efd3e75fa18c6403bc2f1db85a7c8f767a8d32246c66f30aac8", V: 28},
		},
		{
			name:         "updateLeverage",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UpdateLeverageAction{Type: "updateLeverage", Asset: 0, IsCross: true, Leverage: 10},
			msgpack:      "84a474797065ae7570646174654c65766572616765a5617373657400a7697343726f7373c3a86c657665726167650a",
			connectionID: "0xe69c035a0297dd24d9b288c3e8509ae251baf7bf43a43b895327a2cab098389e",
			mainnet:      ActionSignature{R: "0x4fa0ed1f37b56019934737616f720140dc2d415064df1f02ef44e0a6431dc02d", S: "0x541f7b41434b08725ae4d5908e3dc426e8a9ae09833f907d077fc4e32df6c2e", V: 27},
			testnet:      ActionSignature{R: "0xd4498ef57bf0c1361161ef3cd6da84a912a93c162257f2d45f7211711115f238", S: "0x1f351995d1950021ddcc17c1f9b1ad60870c56ab7d87bc3f32210970229b6bbf", V: 27},
		},
		{
			name:         "updateIsolatedMargin_remove",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UpdateIsolatedMarginAction{Type: "updateIsolatedMargin", Asset: 0, IsBuy: true, SignedNotional: -1000000},
			msgpack:      "84a474797065b475706461746549736f6c617465644d617267696ea5617373657400a56973427579c3a46e746c69d2fff0bdc0",
			connectionID: "0xe074d8082296de8d9d091128b0c96323c3356e4e2cc6c54c005e15d9a2d1ecc0",
			mainnet:      ActionSignature{R: "0x13b5aa2260092bea46935789ac4078e689d091e04948cf04575058e5a2ed4bfd", S: "0x664114841fd3e4f9735e129e8c8acc9d7b9fc9fd67c0a6a7c555491f64411703", V: 27},
			testnet:      ActionSignature{R: "0x156e45528987429b58fef2446fd1bee6e3c89a4c31fb26fadb25f8c4b28a9e74", S: "0x6cfa27b0e9cc1b1ffd457ee50e0d68d7a0435aa57a458d6259c5f3d2722ae4ff", V: 28},
		},
		{
			name:         "updateIsolatedMargin_add",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UpdateIsolatedMarginAction{Type: "updateIsolatedMargin", Asset: 0, IsBuy: true, SignedNotional: 2000000},
			msgpack:      "84a474797065b475706461746549736f6c617465644d617267696ea5617373657400a56973427579c3a46e746c69ce001e8480",
			connectionID: "0x902662eb53531f9bbc799b947ce8a7e57e333375147e8c639834a31573a60828",
			mainnet:      ActionSignature{R: "0x62e1ff9431cf29429c6b832ed904654ed1fd0958fe9c9c8c9531187b4bca1422", S: "0x34cb65673371ff55d6c4ade72b3c9397a2131ec1542f6dad2d2e1ed4a9bec8af", V: 28},
			testnet:      ActionSignature{R: "0xa1bc810551670bdc0ea694c9d1a7f22f25a822f085e894d8ebf1fe6204d0af9a", S: "0x4e5e58a8edfb6a79af2afe23072a555c4ebe92fad8a3f2346f55d1d85915ed0c", V: 28},
		},
		{
			name:         "topUpIsolatedOnlyMargin",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       TopUpIsolatedOnlyMarginAction{Type: "topUpIsolatedOnlyMargin", Asset: 0, Leverage: "0.5"},
			msgpack:      "83a474797065b7746f70557049736f6c617465644f6e6c794d617267696ea5617373657400a86c65766572616765a3302e35",
			connectionID: "0x038e92fa7cc5fb9c2626674b762ec9126176489246cb18580fc5aea3ac43605d",
			mainnet:      ActionSignature{R: "0xd3dbbe7808db4f93967e76db922751942410266be4eb5d0b48ae3b13baccbe7c", S: "0x48bfa7edd7005dcb21e61ca482e9056247abd4604ab87c7235b9d5dff258f7ee", V: 28},
			testnet:      ActionSignature{R: "0x658bb852670d59313c405a057d0d7107d3e7c96ba429cc156fd0c35e9e552335", S: "0xf41c0795932ef45a719a5ac944b24f026f7165a0dff09b5efeb40517d9b69c3", V: 27},
		},
		{
			name:         "agentSendAsset",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       AgentSendAssetAction{Type: "agentSendAsset", Destination: "0x5e9ee1089755c3435139848e47e6635505d5a13a", DestinationDEX: "spot", Token: "USDC", Amount: "1", Nonce: vectorNonce},
			msgpack:      "88a474797065ae6167656e7453656e644173736574ab64657374696e6174696f6ed92a307835653965653130383937353563333433353133393834386534376536363335353035643561313361a9736f75726365446578a0ae64657374696e6174696f6e446578a473706f74a5746f6b656ea455534443a6616d6f756e74a131ae66726f6d5375624163636f756e74a0a56e6f6e6365cf0000018bcfe56800",
			connectionID: "0xec8024cc42a8628a0bf3a68921fef30c645941d0adb0c536714a3e310a64bd24",
			mainnet:      ActionSignature{R: "0x1920b347b87d3d9cf60be8025911b1bcebb95948fc8ce4ca0f54b67ab3e5d692", S: "0x4b07dad155d173e0baf87f4500fdb31836ef07ab13ec0c09c3bf4d8437c20bf6", V: 27},
			testnet:      ActionSignature{R: "0x64109af8036b78573271b8bbb4a24fe1ed0e6a91984ef62d2e5918ccd107d555", S: "0x5bbe0a293138674f4b98eebd7172fe15bc3ef7a8c0d072157518d00a5bacddf9", V: 28},
		},
		{
			name:         "vaultTransfer",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       VaultTransferAction{Type: "vaultTransfer", VaultAddress: "0x1719884eb866cb12b2287399b15f7db5e7d775ea", IsDeposit: true, USD: 5000000},
			msgpack:      "84a474797065ad7661756c745472616e73666572ac7661756c7441646472657373d92a307831373139383834656238363663623132623232383733393962313566376462356537643737356561a969734465706f736974c3a3757364ce004c4b40",
			connectionID: "0xb974966aa6554835291cb85bdd65990bfc4f8c08f27e15877634693bfa381b8a",
			mainnet:      ActionSignature{R: "0xc9edc37d4c557fd445dee76391b1f387a9210571e13da79fee395e8434c58b22", S: "0x44a381185add06b1f71b06ce839860edc1b7b83382882c55ca89daf3c2f754cb", V: 28},
			testnet:      ActionSignature{R: "0x9559cf8fa41eca906b034d0bbf914e44d521dd1cffa4146f874aa70471458a5", S: "0x54e146d703b5cc50ba87c9c339b2644fad40b8c41cf74060dd4ba0f4f5ba81fb", V: 27},
		},
		{
			name:         "hip3LiquidatorTransfer",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       HIP3LiquidatorTransferAction{Type: "hip3LiquidatorTransfer", DEX: "xyz", Notional: 1000000000, IsDeposit: true},
			msgpack:      "84a474797065b6686970334c697175696461746f725472616e73666572a3646578a378797aa36e746cce3b9aca00a969734465706f736974c3",
			connectionID: "0xb45ea4ee980a8208c40f3f59d26757b3b955caa60acdc39199639d9e5198aae2",
			mainnet:      ActionSignature{R: "0xbf8ae983d982802404c70f4feec4f28aea9f75d904d143cb1bfcd7e8902884fe", S: "0x4b2605a574a3c0859eea9aaa9f0173a41602476edff17845567d33747741968f", V: 27},
			testnet:      ActionSignature{R: "0x6f17fdaf3264d244b2fffc0e702168c8b7ab67eab9b399bae6357b267a4bf444", S: "0x6cc916dd2f203f1d78a1076b4d375ee61959315abb9a06fd0b3a1e83c72bc0ba", V: 28},
		},
		{
			name:         "twapOrder",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       TWAPOrderAction{Type: "twapOrder", TWAP: TWAPWire{Asset: 0, IsBuy: true, Size: "0.1", Minutes: 30}},
			msgpack:      "82a474797065a9747761704f72646572a47477617086a16100a162c3a173a3302e31a172c2a16d1ea174c2",
			connectionID: "0x1220289874c2dcf7607e5f62232c14f5bd0d62328082e53cc87aab921865d638",
			mainnet:      ActionSignature{R: "0xc49cc9f8a0cee9658ba10a18f3152e89fd2a5a91aac23d89d1f8f974407a457f", S: "0x7b85a2c7bafc2aa751e4e8933f5792d169ffbf2ff3c75f1878e6edcfa2420108", V: 28},
			testnet:      ActionSignature{R: "0xf97c65f8869fd3f3003747d199ae44ba8f247c669367b1975c24136f3dcbd792", S: "0x420a60f2e904793b91dad08b55bcfadd37b5471e9e2eb01a9d2ccd1d7b6cef1c", V: 28},
		},
		{
			name:         "twapCancel",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       TWAPCancelAction{Type: "twapCancel", Asset: 0, TWAPID: 4321},
			msgpack:      "83a474797065aa7477617043616e63656ca16100a174cd10e1",
			connectionID: "0xd7c465d345bc1c116fccca5c25154ca4f8cf7f700b83684adf608611fc821f81",
			mainnet:      ActionSignature{R: "0x145aed2ebb8b8a974031e58eeb2b2a50aa1bc622063bcfd8815a598587f1e52d", S: "0x4c2d014402761b06e4f5a78badae6e3b585bc7bd9255ff3a404c8a8900c2879c", V: 28},
			testnet:      ActionSignature{R: "0xf758006148852da02bd1fdcf27c9d344933526c6793107eec36fc689f680c53f", S: "0x23ab2f7563ff18838298701870781e50c845e3e78c786fc49b48c6fe66f1931f", V: 28},
		},
		{
			name:         "trailingStop_pct_noActivation",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       TrailingStopAction{Type: "trailingStop", Size: "0.1", ReduceOnly: true, Retracement: RetracementWire{Percent: "1%"}},
			msgpack:      "87a474797065ac747261696c696e6753746f70a5617373657400a56973427579c2a2737aa3302e31aa7265647563654f6e6c79c3ab726574726163656d656e7481a3706374a23125ac61637469766174696f6e5078c0",
			connectionID: "0xd73470fd53478e89f59a13f25527b01e3a94c5a118e9d5c61bd1733424c0a82b",
			mainnet:      ActionSignature{R: "0xcff12005693c1ea4295997353c441620a16b3d8000e8ea8658f80c9ab936acf4", S: "0x6289a1bbd18a8bc3864ad291eddb5ff2ded3eeba52fd24913641c6e71feafd86", V: 28},
			testnet:      ActionSignature{R: "0x4a007ff61c74e333e0db813023b775db44b34ce0f10f1f249a4f96b02e038966", S: "0x17223c135cecdcc780fd5cf99cada380875bedc48a5d95d7542d863928ab4c1d", V: 28},
		},
		{
			name:         "trailingStop_px_activation",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       TrailingStopAction{Type: "trailingStop", Size: "0.1", ReduceOnly: true, Retracement: RetracementWire{Price: "500"}, ActivationPrice: new("65000")},
			msgpack:      "87a474797065ac747261696c696e6753746f70a5617373657400a56973427579c2a2737aa3302e31aa7265647563654f6e6c79c3ab726574726163656d656e7481a27078a3353030ac61637469766174696f6e5078a53635303030",
			connectionID: "0xe0e177957839d0c2b9497d51bc05303504494d6584377c7cada975bbab3f7c36",
			mainnet:      ActionSignature{R: "0x93a95a3c9f7dffab2998f455d8e9f24e8d8b8aa1028eeb3402e59f9865d7ad91", S: "0x6ff728771eb30b34ad37ecac212d791ade7e6db23181953b98d55522f9e02abf", V: 28},
			testnet:      ActionSignature{R: "0xabf7367f0e58adcc69949edb99c7a9a0b0b263a1e9fecd1c4c329d0301cd0869", S: "0x41cce9af9526a5aadf44c322526273fbc4e3ca459069527b15165fcc86a8ac83", V: 27},
		},
		{
			name:         "reserveRequestWeight",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       ReserveRequestWeightAction{Type: "reserveRequestWeight", Weight: 10},
			msgpack:      "82a474797065b47265736572766552657175657374576569676874a67765696768740a",
			connectionID: "0xc52c2d4aaf97e879278e9fba20ea7903fd6b949c3b3463e81c4873aca3a2841e",
			mainnet:      ActionSignature{R: "0x253d20ee6d467b6ba1046abb0545624697aef893a065d601aa0c2c4c0b3319b8", S: "0x41ed5806f0840a17aac102a4f796ae4cd60953a7f6aaf517e899e645ded8ac6f", V: 27},
			testnet:      ActionSignature{R: "0x7083d0643b1d4c4c02513829610d88646c95fbe72072bb699e0c17f40b54e046", S: "0x71b4a38f08c8ae936a8e2e5c5861075dec7c4a9da6875bb23129f822e9329888", V: 27},
		},
		{
			name:         "reserveRequestWeight_destination",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       ReserveRequestWeightAction{Type: "reserveRequestWeight", Weight: 10, Destination: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
			msgpack:      "83a474797065b47265736572766552657175657374576569676874a67765696768740aab64657374696e6174696f6ed92a307835653965653130383937353563333433353133393834386534376536363335353035643561313361",
			connectionID: "0x1fd4308f6b609239394b4ffde4cbc960578242cb5c8eb4e13d75d46f6db8c342",
			mainnet:      ActionSignature{R: "0x2d949d2b8d8eff20354eb01049469822187a3032d480c315f94dc45af94d09f8", S: "0x7953aad4f94a7d2d14519320db6b434c3a31a2a6955ab7c8e540639b8e7d4f56", V: 28},
			testnet:      ActionSignature{R: "0xff995388d24c428994e54eb7a283d03af714b9f5b438e11d6e61f6e4413dbdcd", S: "0x5fe70981bb6a2d8c0d4553a9b1f4d0862bdb73042dc7979aad1d0d62464e4d68", V: 28},
		},
		{
			name:         "noop",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       EmptyAction{Type: "noop"},
			msgpack:      "81a474797065a46e6f6f70",
			connectionID: "0xef5dcef9775ebb2c5a6553314e66a6a57bd7e9b2319a869a8b17f08fa48bdcaf",
			mainnet:      ActionSignature{R: "0xa094d7afdcaffc2e2643f31df05a7594de12880e6558e17021d863c868a06972", S: "0x2ec74c7efddc03c04b04a660effe31f52ce3cddd2e0b80c68486ec772ca42ce2", V: 28},
			testnet:      ActionSignature{R: "0x4b9af944e3380469044c5ec5d33a9da8cfb406c8e06d5b758f7b9d5bc4d095f", S: "0x862f995ef927f761c5dadc2c7a91d39dab55e6991c9b01e10b5f6ab1d3cfaeb", V: 27},
		},
		{
			name:         "agentEnableDexAbstraction",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       EmptyAction{Type: "agentEnableDexAbstraction"},
			msgpack:      "81a474797065b96167656e74456e61626c654465784162737472616374696f6e",
			connectionID: "0x22fcc93b98d265d62f3ffe5ec83f14d24a85bc3ed0c003615bf57aa0243101f1",
			mainnet:      ActionSignature{R: "0x6f4c06bd5aacfbfb98a877bf9058c8483ded16c8051e9e71ecce38e83963b551", S: "0x454486824bc1e47e295b895f8e46f8a359149a0a71bd76e507faab59f86b6cc2", V: 28},
			testnet:      ActionSignature{R: "0x4e1636932a93e5a5b1dc07944e7f087204ffd0a41a975eb1cd8729cf7bb4710c", S: "0x2e49aa19622c0546592b800f74e0d07d34e7f3cb37084ca6cfd0a9beaf0ae01c", V: 27},
		},
		{
			name:         "agentSetAbstraction",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       AgentSetAbstractionAction{Type: "agentSetAbstraction", Abstraction: "u"},
			msgpack:      "82a474797065b36167656e745365744162737472616374696f6eab6162737472616374696f6ea175",
			connectionID: "0x7632c5247830574d780c36b0537a66f0dbb1e03ec04202ea7ad230786d8571ed",
			mainnet:      ActionSignature{R: "0x800a72cf01f77e9d6995c8dc12506e6f6a4a2b5d2e892b8170002c731903b302", S: "0x6b789570b562607700c1633697cafd5b2b046549b83bc5ff33d810f786d3ec9e", V: 27},
			testnet:      ActionSignature{R: "0x93874db7738dc23602081943a5ea8b37a394b06893aa21c8c44dd1dad59bdfc", S: "0x1c9120ce0f7db5789aff0ab3d2fba1089bbe429b90d1a2167aa91d380d730b12", V: 28},
		},
		{
			name:         "userOutcome_splitOutcome",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UserOutcomeAction{Type: "userOutcome", SplitOutcome: &SplitOutcomeWire{Outcome: 1, Amount: "10"}},
			msgpack:      "82a474797065ab757365724f7574636f6d65ac73706c69744f7574636f6d6582a76f7574636f6d6501a6616d6f756e74a23130",
			connectionID: "0x678b2b09bedc58338af70dfbf80646b4d6ea8374dc941d96f1a5f83331f1edcd",
			mainnet:      ActionSignature{R: "0x9754c876cfb9fa50c37c09649f078f93b681a694dcb35715c44c2837994f4864", S: "0x6462d803970f82da447fab4edbf4a9d3544d1332fac7fb7324d06b6822e66d43", V: 27},
			testnet:      ActionSignature{R: "0x3a17cca9f4856cbea62f457c6f0a583e774fa679c99e0a538a3a1003716b0f70", S: "0x7cc1880bd506a04051f2621b672be5e433beb4b2287c26e38b8f3519ff0ea309", V: 27},
		},
		{
			name:         "userOutcome_mergeOutcome_max",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UserOutcomeAction{Type: "userOutcome", MergeOutcome: &MergeOutcomeWire{Outcome: 1}},
			msgpack:      "82a474797065ab757365724f7574636f6d65ac6d657267654f7574636f6d6582a76f7574636f6d6501a6616d6f756e74c0",
			connectionID: "0x0ac1ceffebdb06e7c2cee99897d9182528804e29c532284d59a3291af2a1e05e",
			mainnet:      ActionSignature{R: "0x26cd392d4bc87fee7264f000a569ec438ef9bba8f754c3cfbe187989027bd67b", S: "0x788b62c28e746d2055b61c08c47ad4fde3a414c1ca06c810ec32ba35008cb6c0", V: 28},
			testnet:      ActionSignature{R: "0x69311eb166279e738f33ac26507e1619462a0602511e66fec4fdfa873a154ed4", S: "0x1b9990ea47afb4fe7b5aa2718033bbbeb421d4c83fde7f31f45b8482f1b1635f", V: 27},
		},
		{
			name:         "userOutcome_mergeQuestion",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UserOutcomeAction{Type: "userOutcome", MergeQuestion: &MergeQuestionWire{Question: 2, Amount: new("10")}},
			msgpack:      "82a474797065ab757365724f7574636f6d65ad6d657267655175657374696f6e82a87175657374696f6e02a6616d6f756e74a23130",
			connectionID: "0x3a68a4e06d6471e7399d41c69005494074af406d49aaa0d3742cf1aff3c4567b",
			mainnet:      ActionSignature{R: "0x9a73df61b40e17da227d329381bc99d5cf793aa349e3910ffe9b4562616588", S: "0x43079ef75c865018b33ad254b0849fe14c949216a736730fc27884dd978be4ea", V: 28},
			testnet:      ActionSignature{R: "0x12b15c674a040cc3dbbfcdd39a5312e933a367a584b6b67bae01cc522e06179b", S: "0x736b31c7f82a9e9d79ac592f0a161d59a10c7fdbd6400b173f9ef67c90cea857", V: 28},
		},
		{
			name:         "userOutcome_negateOutcome",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       UserOutcomeAction{Type: "userOutcome", NegateOutcome: &NegateOutcomeWire{Question: 2, Outcome: 1, Amount: "10"}},
			msgpack:      "82a474797065ab757365724f7574636f6d65ad6e65676174654f7574636f6d6583a87175657374696f6e02a76f7574636f6d6501a6616d6f756e74a23130",
			connectionID: "0x82e1b2fbecceb4ef3690830880bde548906bc0862b9344e4da120097fe42e511",
			mainnet:      ActionSignature{R: "0x74f772335f5505d810aade51938d3f01c3894ce9bf36bdf82c91bc4c5053c7d8", S: "0x7145550e8c1ac8c77e9b6b5b55b3ea193b83c39ab7255cc4a62ddb63f4c46b79", V: 28},
			testnet:      ActionSignature{R: "0x6b70cdb2d26fee447d1ecf1b9db961c0cb0d053963eb4608134a2dc1ebf75370", S: "0x4d1394860506e04b765fe8d510499122a7ef04b3f893f29c01e01647071493ce", V: 27},
		},
		{
			name:         "claimRewards",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       EmptyAction{Type: "claimRewards"},
			msgpack:      "81a474797065ac636c61696d52657761726473",
			connectionID: "0x51aaf1b5200df7fd6fe4d7b9695b49a19d5fedb54caccfd6005949da3fbdd50a",
			mainnet:      ActionSignature{R: "0xf7a4d8dcf32317c538466c0f698fe57afdd22aa459e6df17a2c649d54f7e6058", S: "0x454fcc7fdcf5f5765e7b3038e2ec62b6465107f8567af0ec75be1206ac756a79", V: 28},
			testnet:      ActionSignature{R: "0x500d8d170f6ae9f87ce8b72ee3fa43f7d0400a48ae009bad176ea5ab5ddf70ad", S: "0x3f713f8392a36b277e08b5d1cd8f11f33106ff49b90a65ff815d4752bc47f3c3", V: 27},
		},
		{
			name:         "gossipPriorityBid",
			key:          testPrivateKey,
			nonce:        vectorNonce,
			action:       GossipPriorityBidAction{Type: "gossipPriorityBid", IP: "1.2.3.4", MaxGas: 100000000},
			msgpack:      "84a474797065b1676f737369705072696f72697479426964a6736c6f74496400a26970a7312e322e332e34a66d6178476173ce05f5e100",
			connectionID: "0xfab9f7418fd0387a860180530444318881ded2c5a2f866c9bbedc87bf6177701",
			mainnet:      ActionSignature{R: "0x69088e501705d1b1f76e0c8f3d568ea03f4d62606343cebe5b43499507cc41aa", S: "0x19390bccebeb79d44a1fd7344123e15fd1f68042ea2d0c791710bdaee47e6d1", V: 27},
			testnet:      ActionSignature{R: "0x283b2ed64c44987fda04ee3b318551cb151f16d4baf93213739dc199467c6e67", S: "0x3862bf94f58c9b37d1b42f1365dcc9c8cfff1def78cb08577ffb9e576ccd6f76", V: 27},
		},
		{
			name:         "python order testnet",
			key:          testPrivateKey,
			nonce:        0,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "100", Size: "100", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			connectionID: "0x884f2c32bb6dbdd65f6033e32fb28c0cb6f5b345db0f6471fd3366d85c9252c1",
			testnet:      ActionSignature{R: "0x82b2ba28e76b3d761093aaded1b1cdad4960b3af30212b343fb2e6cdfa4e3d54", S: "0x6b53878fc99d26047f4d7e8c90eb98955a109f44209163f52d8dc4278cbbd9f5", V: 27},
		},
		{
			name:         "python order with client order ID testnet",
			key:          testPrivateKey,
			nonce:        0,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "100", Size: "100", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}, ClientOrderID: "0x00000000000000000000000000000001"}}, Grouping: GroupingNone},
			connectionID: "0x0ba500cedd8f4ba6ded620a0b1cd04f124d9ba745e2e2893fcc763bcc1444af5",
			testnet:      ActionSignature{R: "0xeba0664bed2676fc4e5a743bf89e5c7501aa6d870bdb9446e122c9466c5cd16d", S: "0x7f3e74825c9114bc59086f1eebea2928c190fdfbfde144827cb02b85bbe90988", V: 28},
		},
		{
			name:         "python trigger order testnet",
			key:          testPrivateKey,
			nonce:        0,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "100", Size: "100", ReduceOnly: false, Type: OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "103", TakeProfitStopLoss: TriggerStopLoss}}}}, Grouping: GroupingNone},
			connectionID: "0x430a86fb9876e901920d931f5bb20c9d011f6389bd179f39a73c09e6219adcad",
			testnet:      ActionSignature{R: "0x971c554d917c44e0e1b6cc45d8f9404f32172a9d3b3566262347d0302896a2e4", S: "0x206257b104788f80450f8e786c329daa589aa0b32ba96948201ae556d5637eac", V: 28},
		},
		{
			name:         "python schedule cancel clear",
			key:          testPrivateKey,
			nonce:        0,
			action:       ScheduleCancelAction{Type: "scheduleCancel"},
			connectionID: "0xa2887a3147b6542306b61d311a056fd1753913d63cc904f30cba61712a98f4ae",
			mainnet:      ActionSignature{R: "0x6cdfb286702f5917e76cd9b3b8bf678fcc49aec194c02a73e6d4f16891195df9", S: "0x6557ac307fa05d25b8d61f21fb8a938e703b3d9bf575f6717ba21ec61261b2a0", V: 27},
			testnet:      ActionSignature{R: "0xc75bb195c3f6a4e06b7d395acc20bbb224f6d23ccff7c6a26d327304e6efaeed", S: "0x342f8ede109a29f2c0723bd5efb9e9100e3bbb493f8fb5164ee3d385908233df", V: 28},
		},
		{
			name:         "python schedule cancel time",
			key:          testPrivateKey,
			nonce:        0,
			action:       ScheduleCancelAction{Type: "scheduleCancel", Time: new(uint64(123456789))},
			connectionID: "0x4be18e445114437c5d1d9dd35a09f5601a3cc34ed4ac94a0281251b9bd8f6832",
			mainnet:      ActionSignature{R: "0x609cb20c737945d070716dcc696ba030e9976fcf5edad87afa7d877493109d55", S: "0x16c685d63b5c7a04512d73f183b3d7a00da5406ff1f8aad33f8ae2163bab758b", V: 28},
			testnet:      ActionSignature{R: "0x4e4f2dbd4107c69783e251b7e1057d9f2b9d11cee213441ccfa2be63516dc5bc", S: "0x706c656b23428c8ba356d68db207e11139ede1670481a9e01ae2dfcdb0e1a678", V: 27},
		},
		{
			name:         "typescript order",
			key:          typeScriptSDKTestKey,
			nonce:        1234567890,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			connectionID: "0x25367e0dba84351148288c2233cd6130ed6cec5967ded0c0b7334f36f957cc90",
			mainnet:      ActionSignature{R: "0x61078d8ffa3cb591de045438a1ae2ed299b271891d1943a33901e7cfb3a31ed8", S: "0xe91df4f9841641d3322dad8d932874b74d7e082cdb5b533f804964a6963aef9", V: 28},
			testnet:      ActionSignature{R: "0x6b0283a894d87b996ad0182b86251cc80d27d61ef307449a2ed249a508ded1f7", S: "0x6f884e79f4a0a10af62db831af6f8e03b3f11d899eb49b352f836746ee9226da", V: 27},
		},
		{
			name:         "typescript order with vault",
			key:          typeScriptSDKTestKey,
			nonce:        1234567890,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			vault:        "0x1234567890123456789012345678901234567890",
			connectionID: "0x214e2ea3270981b6fd18174216691e69f56872663139d396b10ded319cb4bb1e",
		},
		{
			name:         "typescript order with expiry",
			key:          typeScriptSDKTestKey,
			nonce:        1234567890,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			expiresAfter: new(uint64(1234567890)),
			connectionID: "0xc30b002ba3775e4c31c43c1dfd3291dfc85c6ae06c6b9f393991de86cad5fac7",
		},
		{
			name:         "typescript order with vault and expiry",
			key:          typeScriptSDKTestKey,
			nonce:        1234567890,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 0, IsBuy: true, Price: "30000", Size: "0.1", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}}}, Grouping: GroupingNone},
			vault:        "0x1234567890123456789012345678901234567890",
			expiresAfter: new(uint64(1234567890)),
			connectionID: "0x2d62412aa0fc57441b5189841d81554a6a9680bf07204e1454983a9ca44f0744",
		},
		{
			name:         "rust order",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "2000.0", Size: "3.5", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}}}}, Grouping: GroupingNone},
			connectionID: "0x5983a9453b8d32668daefa9310e1a81bc1f4d7da50a9ad8869a4011d12068ea0",
			mainnet:      ActionSignature{R: "0x77957e58e70f43b6b68581f2dc42011fc384538a2e5b7bf42d5b936f19fbb673", S: "0x60721a8598727230f67080efee48c812a6a4442013fd3b0eed509171bef9f23f", V: 28},
			testnet:      ActionSignature{R: "0xcd0925372ff1ed499e54883e9a6205ecfadec748f80ec463fe2f84f120964877", S: "0x6377961965cb7b12414186b1ea291e95fd512722427efcbcfb3b0b2bcd4d79d0", V: 28},
		},
		{
			name:         "rust order with client order ID",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "2000.0", Size: "3.5", ReduceOnly: false, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceIOC}}, ClientOrderID: "0x1e60610f0b3d420597c88c1fed2ad5ee"}}, Grouping: GroupingNone},
			connectionID: "0xc5fe546c778ef44d4804681e65bdbc5cd678cbbbcdc13d1b80eabf0d5a188704",
			mainnet:      ActionSignature{R: "0xd3e894092eb27098077145714630a77bbe3836120ee29df7d935d8510b03a08f", S: "0x456de5ec1be82aa65fc6ecda9ef928b0445e212517a98858cfaa251c4cd7552b", V: 28},
			testnet:      ActionSignature{R: "0x3768349dbb22a7fd770fc9fc50c7b5124a7da342ea579b309f58002ceae49b43", S: "0x57badc7909770919c45d850aabb08474ff2b7b3204ae5b66d9f7375582981f11", V: 28},
		},
		{
			name:         "rust tp trigger order",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "2000.0", Size: "3.5", ReduceOnly: false, Type: OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "2000.0", TakeProfitStopLoss: TriggerTakeProfit}}}}, Grouping: GroupingNone},
			connectionID: "0xce28b892ad1b09ae7a2a50faa6802f3fa1b0e76ab9dd60b98f423673cba01fc4",
			mainnet:      ActionSignature{R: "0xb91e5011dff15e4b4a40753730bda44972132e7b75641f3cac58b66159534a17", S: "0xd422ee1ac3c7a7a2e11e298108a2d6b8da8612caceaeeb3e571de3b2dfda9e4", V: 27},
			testnet:      ActionSignature{R: "0x6df38b609904d0d4439884756b8f366f22b3a081801dbdd23f279094a2299fac", S: "0x6424cb0cdc48c3706aeaa368f81959e91059205403d3afd23a55983f710aee87", V: 27},
		},
		{
			name:         "rust sl trigger order",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       OrderAction{Type: "order", Orders: []OrderWire{{Asset: 1, IsBuy: true, Price: "2000.0", Size: "3.5", ReduceOnly: false, Type: OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "2000.0", TakeProfitStopLoss: TriggerStopLoss}}}}, Grouping: GroupingNone},
			connectionID: "0x93c6fc867458280be8fa653623dab767b96d63c2b049d7d1800cdaddb6759d6f",
			mainnet:      ActionSignature{R: "0x8456d2ace666fce1bee1084b00e9620fb20e810368841e9d4dd80eb29014611a", S: "0x843416e51b1529c22dd2fc28f7ff8f6443875635c72011f60b62cbb8ce90e2d", V: 28},
			testnet:      ActionSignature{R: "0xeb5bdb52297c1d19da45458758bd569dcb24c07e5c7bd52cf76600fd92fdd821", S: "0x3e661e21899c985421ec018a9ee7f3790e7b7d723a9932b7b5adcd7def535460", V: 28},
		},
		{
			name:         "rust cancel",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       CancelAction{Type: "cancel", Cancels: []CancelRequest{{Asset: 1, OrderID: 82382}}},
			connectionID: "0xa53262329d1e221a88ff73e5ce330e0e47fb646091cae7020c34c346e592b77e",
			mainnet:      ActionSignature{R: "0x2f76cc5b16e0810152fa0e14e7b219f49c361e3325f771544c6f54e157bf9fa", S: "0x17ed0afc11a98596be85d5cd9f86600aad515337318f7ab346e5ccc1b03425d5", V: 27},
			testnet:      ActionSignature{R: "0x6ffebadfd48067663390962539fbde76cfa36f53be65abe2ab72c9db6d0db444", S: "0x57720db9d7c4860f142a484f070c84eb4b9694c3a617c83f0d698a27e55fd5e0", V: 28},
		},
		{
			name:         "rust claim rewards",
			key:          rustSDKTestKey,
			nonce:        1583838,
			action:       EmptyAction{Type: "claimRewards"},
			connectionID: "0xe7116fe27a8f1fc959de19f565fd16658b4d1fa6ea874b0eb64375f833ef8389",
			mainnet:      ActionSignature{R: "0xe13542800ba5ec821153401e1cafac484d1f861adbbb86c00b580ec2560c1532", S: "0x48b8d9f0e004ecc86959c07d44b591861ebab2167b54651a81367e2c3d472d4e", V: 28},
			testnet:      ActionSignature{R: "0x16de9b346ddd8e200492a2db45ec9104dcdfc7fbfdbcd85890a6063bdd56df2c", S: "0x44846714c261a431de7095ad52e07143346eb26d9e66c6aed4674f120a104813", V: 28},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.msgpack != "" {
				encoded, err := msgpackAction(tc.action)
				require.NoError(t, err, "msgpackAction must not error")
				assert.Equal(t, tc.msgpack, hex.EncodeToString(encoded), "msgpackAction should encode the action in Hyperliquid's key order")
			}
			connectionID, err := actionHash(tc.action, tc.vault, tc.nonce, tc.expiresAfter)
			require.NoError(t, err, "actionHash must not error")
			assert.Equal(t, tc.connectionID, "0x"+hex.EncodeToString(connectionID[:]), "actionHash should match the vector's connection ID")
			for _, network := range []struct {
				mainnet bool
				exp     ActionSignature
			}{{mainnet: true, exp: tc.mainnet}, {exp: tc.testnet}} {
				if network.exp == (ActionSignature{}) {
					continue
				}
				signature, err := signL1Action(tc.key, tc.action, tc.vault, tc.nonce, tc.expiresAfter, network.mainnet)
				require.NoErrorf(t, err, "signL1Action must not error for mainnet %t", network.mainnet)
				assert.Equalf(t, network.exp, signature, "signL1Action should match the vector for mainnet %t", network.mainnet)
			}
		})
	}
}

// TestUserSignedActionVectors checks each user-signed action's EIP-712 digest and signatures against vectors derived from
// an independent implementation of the SDK signing, and against the published TypeScript and Rust SDK vectors
func TestUserSignedActionVectors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		key           string
		primaryType   string
		fields        []eip712Field
		mainnetDigest string
		mainnet       ActionSignature
		testnetDigest string
		testnet       ActionSignature
	}{
		{
			name:        "usdSend",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:UsdSend",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x66fdd5ad8d92e9585a18ec3ed6c5476e0447bd7aa4fcd442c071ae2ab83d4bca",
			mainnet:       ActionSignature{R: "0x115e9b16a9d32fa3e8feeeaa92e152e5694c43389b1db3b77707665d1a2bed0e", S: "0x1fc96a61f717f21c2ba121cb71e0b3831c43763fd2e620b92de537f85f07c8f0", V: 27},
			testnetDigest: "0x5198d8ce9813c4c35be4938ac3fdb4797da4d08ba3c73e396d649038d102aca7",
			testnet:       ActionSignature{R: "0x1ca9bb1fbf6bb9cab124c3492a18136662ac18fb3b23b0a44adde4070ba6d573", S: "0x10329679a0151bb92ea5827f99c75199a88cb6c052add6fb0068d982538e654e", V: 27},
		},
		{
			name:        "spotSend",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:SpotSend",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "token", Type: "string", Value: "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xd62b4ef10592194df95e5430acfe91bec8ad3cc3b471516475b6bbbd2316d09c",
			mainnet:       ActionSignature{R: "0x1ce67539362cddee3e74883723b09db334b543de0a1c6ad7d32223e1aa348ecb", S: "0x4dfdff980c2b647540d30ee10b225cb74d9d3cde913331177919b5def2bf3544", V: 28},
			testnetDigest: "0x871ab49eca5bf74ff6295a6ea7eb3657d1f4dd25c0efd10aba97b614ee522fca",
			testnet:       ActionSignature{R: "0xbb7d822808bafc88641f71cd512b877c0e965b41eceb0468606ca9bb4b5318d", S: "0x7be936da04792a86ce9fa89e44c46104417986036c3ca1b34fdcd1f97e7747e8", V: 27},
		},
		{
			name:        "withdraw3",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:Withdraw",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x123ba704d6ca647fc28675a40f816ccb3cb0fcd2b20527149e3f527135732f80",
			mainnet:       ActionSignature{R: "0x81daeb81bf08c959181cde0443e56881567ceb0754e44965e9abeb2e2d7ab4c5", S: "0x7d7005b32717eb54dcf4ae47b19d09359c4a281996673bae85f798f5f8cd2613", V: 27},
			testnetDigest: "0xc6a95fe49e5f8fe279693408f0bdd68474bbca12bdf7687a2bdbc4f262998180",
			testnet:       ActionSignature{R: "0x7c752ee3b15229aaaa538df4d296113220c67dd5884f7590ba7c9113c536a28d", S: "0x3d0060f7e638534eb372af14cc761224ab1251419dab19b6e6d9793864ef3a48", V: 27},
		},
		{
			name:        "usdClassTransfer",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:UsdClassTransfer",
			fields: []eip712Field{
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "toPerp", Type: "bool", Value: true},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x43a05c10368d24e881be889649f4cf124f3f43c1c38a4bd9c61c7c94d90ba73c",
			mainnet:       ActionSignature{R: "0xe7f8a6d0eced93e25cf2c8687397b4f0dceaad182d1295c7ab655dd6ae1c6314", S: "0x6ad6d95296dfc1f221745f94e5a1f36a1dca04401858fa6cc1437e0b5274a03b", V: 27},
			testnetDigest: "0x57589e90e204e926dd70c04dbb3aa6bc5c5fa9c6baca22422769ba7941c67503",
			testnet:       ActionSignature{R: "0xc2faa52ae4e59ce98a6e51e94798f01970f45fd177d4a61f55d88d04e1e84ebd", S: "0x2cc583a3f4144282c6c35f1a4c294270dd5df1b7270bd479b4dfbe95d7992e70", V: 27},
		},
		{
			name:        "usdClassTransfer_subaccount",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:UsdClassTransfer",
			fields: []eip712Field{
				{Name: "amount", Type: "string", Value: "1 subaccount:0x1719884eb866cb12b2287399b15f7db5e7d775ea"},
				{Name: "toPerp", Type: "bool", Value: false},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xba17211614b46ed9f3e779afff30fc75162e500bf120600f415e36497adfbcbb",
			mainnet:       ActionSignature{R: "0xd5e5ad8976366014632eaf46e4de253864252b8d738b6779e926202e13fe30cd", S: "0x484c62d816ec97d659a6e1dade2466c5f6b560ddbc44f3b2abd279060d7c8c9", V: 28},
			testnetDigest: "0x72c9649b91e1e92858408fad4ade3cae4814bcbaffb9c8b6f52252b2582b8bf7",
			testnet:       ActionSignature{R: "0xbcc62ef5356b668555ec5379c0d337beae60b012b63b515b9846e42681fb06cc", S: "0x19e1a74e2b557931a0a255768b6f9642dc9155e06706d5fa121837fa6538b082", V: 28},
		},
		{
			name:        "sendAsset",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:SendAsset",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "sourceDex", Type: "string", Value: ""},
				{Name: "destinationDex", Type: "string", Value: "spot"},
				{Name: "token", Type: "string", Value: "USDC"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "fromSubAccount", Type: "string", Value: ""},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xb25adab2ea9be716124b7a4dc2d460fc7b597f66e652a2c5bd02be50dfa2d6ff",
			mainnet:       ActionSignature{R: "0x7c1cb1468b27a81fe68accda88215862befdc32187d637071da71b665eb09ce6", S: "0x29d5ad41f0b53e0b3ce8d7791a1b18a70661a4801264fff1631c2977c1357039", V: 27},
			testnetDigest: "0x210465d584ba94b347ad70f3e151278700ca756c68a7b202078077f53469c98d",
			testnet:       ActionSignature{R: "0x9ae62174e16f21cec03fae0b49e20ded482b88885e0801ff26f2a4309061d1a0", S: "0x6e1f7b2e2bb3f906cee0b1f4e5f8d39bee7d15939e2d3f5605b8c4a0e2a8eac9", V: 27},
		},
		{
			name:        "sendToEvmWithData_emptyData",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:SendToEvmWithData",
			fields: []eip712Field{
				{Name: "token", Type: "string", Value: "USDC"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "sourceDex", Type: "string", Value: "spot"},
				{Name: "destinationRecipient", Type: "string", Value: "0x0000000000000000000000000000000000000001"},
				{Name: "addressEncoding", Type: "string", Value: "hex"},
				{Name: "destinationChainId", Type: "uint32", Value: uint32(998)},
				{Name: "gasLimit", Type: "uint64", Value: uint64(200000)},
				{Name: "data", Type: "bytes", Value: "0x"},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x3ee698ce96a3b5a2f0707a287a0520e224e26647f557f814010aca4f46fdd556",
			mainnet:       ActionSignature{R: "0x3997d05f1d896efd4894e1c01848c04288da745e4299e1e95aeb52d398747e6e", S: "0x17ffa4b76015077b234a76e0629cab7a1fc642dfee31b9abb591d479703c1214", V: 27},
			testnetDigest: "0xf7a3f38469732f5ecd5277062adfe5adc23b5c730000724639dfb3f15da97012",
			testnet:       ActionSignature{R: "0xaa28162965d261cd24fb7d8955c43e550864d7e50c72912c220bfbaec66f7848", S: "0x39afdbd0eace38220a7d83a1bb321a46837a0e19ed79d423f08340b6b5a783fb", V: 27},
		},
		{
			name:        "sendToEvmWithData_data",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:SendToEvmWithData",
			fields: []eip712Field{
				{Name: "token", Type: "string", Value: "USDC"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "sourceDex", Type: "string", Value: "spot"},
				{Name: "destinationRecipient", Type: "string", Value: "0x0000000000000000000000000000000000000001"},
				{Name: "addressEncoding", Type: "string", Value: "hex"},
				{Name: "destinationChainId", Type: "uint32", Value: uint32(998)},
				{Name: "gasLimit", Type: "uint64", Value: uint64(200000)},
				{Name: "data", Type: "bytes", Value: "0xdeadbeef"},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x8e7c13d915ca77522646f7456a7586c40c034a16530552048a28763064bf55db",
			mainnet:       ActionSignature{R: "0x2d63d42112d4db6572735ce6e1e2e05840e459944d5238fd1d1124ecc529a545", S: "0x6e1ff8938c35a2fe96ba38c43d6735fcca36aa3f65b47edd7cc014e4c02ff844", V: 27},
			testnetDigest: "0x2e3daa6d020cae2126deca32cb659ab1e25f03c714285583982dd0eeddedca7d",
			testnet:       ActionSignature{R: "0x5f72f5d85ea3c2dbfe475a11b4f6fcca4a2556159d09a5d319f92588c21330ac", S: "0x6d5ba3bd2f3c0031f8a70dc9f7c0224d3d3d2c3ee414c7413cb5296d2189aeb7", V: 27},
		},
		{
			name:        "cDeposit",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:CDeposit",
			fields: []eip712Field{
				{Name: "wei", Type: "uint64", Value: uint64(100000000)},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xae33421c367d1a2aa2cd0c1d0a88675b0d85ebc5133df4038fc3b242e5763253",
			mainnet:       ActionSignature{R: "0x9e05d03871ee070fba04df5287eee8fd0de5ed259809bc179565b3bbd949a4", S: "0x5b961f85ff278a5e473269f91a2020b0c129545feb80528438c06631186dccba", V: 27},
			testnetDigest: "0xe95793fc4f24520c9decf14e407c7029ef14c2fd2e13c5cf5b35dd622c8692e4",
			testnet:       ActionSignature{R: "0xce565b18b7825a83ddb5cd6950b78429b76bf85458ec830954b0a0e58ce5fdb7", S: "0x26cca9c05d08d9f72d772394ed9fe33d7b19106f29f7a1f3cdffae7aacb512ea", V: 28},
		},
		{
			name:        "cWithdraw",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:CWithdraw",
			fields: []eip712Field{
				{Name: "wei", Type: "uint64", Value: uint64(100000000)},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x0f3731764f8fe4519a0588c6fcc81fe0c2c3daa02b9da087904a6493967fcac4",
			mainnet:       ActionSignature{R: "0xe75751e34d0a6f080b76cb43f96ea9ea85b33a346ff773672e98f2925f27a4c4", S: "0x1ca097754433e19821f250c86d76bf486884edc70b7198797e9b6fd506b62827", V: 28},
			testnetDigest: "0xf1e1dcb696181fec18019f395de0f9bd2274fa4a9b5e0569efa66de195b7967e",
			testnet:       ActionSignature{R: "0x19b1735d7056ef0db533a6bac2acae4753595aba31a5c4713a0bff8c27e928d5", S: "0x171e0684da1d9419dda11e4e1773b948702fe05b3a90e665b37dd126b70035cc", V: 27},
		},
		{
			name:        "tokenDelegate",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:TokenDelegate",
			fields: []eip712Field{
				{Name: "validator", Type: "address", Value: "0xa012b9040d83c5cbad9e6ea73c525027b755f596"},
				{Name: "wei", Type: "uint64", Value: uint64(100000000)},
				{Name: "isUndelegate", Type: "bool", Value: false},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xe83edb9afdc2dafc26442f0a9f85ee2bcfcfb00b0611c557de2c6fc8ea302c05",
			mainnet:       ActionSignature{R: "0x1bee07c23cd158a91f5a0735a21fba9e27349c569ab494829194bbfc212b1755", S: "0x45feeef013f721bb1cf39812ece79d12bbfd3ed1f174df5bb1d6dea52bb7c9c5", V: 27},
			testnetDigest: "0xa65b9a153b6ab1b5ab9fb585c65f8137438c99247c516b94ccb056818b615992",
			testnet:       ActionSignature{R: "0x3530c88a21f96806184484b1c656466f3804677f4db5f78148624035f3417309", S: "0x1c38218121f609a31444166600a8d19bcfb59adee6bc4bfeb15e670364f6e09b", V: 28},
		},
		{
			name:        "approveAgent_named",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:ApproveAgent",
			fields: []eip712Field{
				{Name: "agentAddress", Type: "address", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "agentName", Type: "string", Value: "gct"},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0x4e64a5bdcf6d9cdeabeaab7342875418cdff6112e92e9c9ea882a8ef12455050",
			mainnet:       ActionSignature{R: "0xc4ac39a116d356fe3385db68e6b6c5b85d8a69f402605c00737fe87c96c52497", S: "0x32dc0619351988c404854de7c6489868dad99d8b7acdeb69984c0f4114463b91", V: 27},
			testnetDigest: "0x6d45e3554c11d4d9125e5e8669b1387356f165d485a0d951efe29feeabda69a0",
			testnet:       ActionSignature{R: "0x87a7b0717b990841662f0e0c2ff94aa37dce853a775148067d78ff777265ae98", S: "0xc310e0b21a5704f038a96f8ec28343c59be24971a567e5f273f3c6b6df82dea", V: 28},
		},
		{
			name:        "approveAgent_unnamed",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:ApproveAgent",
			fields: []eip712Field{
				{Name: "agentAddress", Type: "address", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "agentName", Type: "string", Value: ""},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xcdcca6f57e457cf30b0ec9eb74a6f1a7cbe7bea5fcb227e5a3d7783d8ee6cbea",
			mainnet:       ActionSignature{R: "0x39b38d050a63b34e54a8ec5a82a8312729dc8e367f11271054e69017af9e1dd1", S: "0x5c4a5676db9655d570d6ad57fad9bb410534790522092f070fc4a56a2786eba7", V: 28},
			testnetDigest: "0xfa4b547dba5c04e447f9b78bc77c361a8f28f7d76fe0ca7ae3d567a364ec86dd",
			testnet:       ActionSignature{R: "0x524862e2b34223c4eb528fc24097be7cfc62eae3341856ee0eb2146948511ad6", S: "0x5fe72cee4d895e2632e457f9b42ee7688b3737950e188cd6a936713b992cf109", V: 28},
		},
		{
			name:        "approveBuilderFee",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:ApproveBuilderFee",
			fields: []eip712Field{
				{Name: "maxFeeRate", Type: "string", Value: "0.001%"},
				{Name: "builder", Type: "address", Value: "0x1234567890123456789012345678901234567890"},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xa95c234627c41c6f5553011bad5046430f3a44d36cc36ecb190cefba3d3a1f0d",
			mainnet:       ActionSignature{R: "0x301dbe4e59f964826818b1d04d2cabc3e96790fabf5eb0b62215765b41dcc251", S: "0x23dcd50dda91a33bf163efc355b98a30dcd05075bbc2db83d59469178fd6f422", V: 27},
			testnetDigest: "0xdc903b0c9a4f96015a8866fcfe590efa91fd7e0cb91dfa80f72d967dbb47515d",
			testnet:       ActionSignature{R: "0xab62ac03bcd8e075cfb9c429d9e1f42e8d057c7bca345d794efd60adbac621e2", S: "0x26aac9bef7df7e13f7254fefa1432e39164ea445bdd73d098fd519958c3b224d", V: 27},
		},
		{
			name:        "userDexAbstraction",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:UserDexAbstraction",
			fields: []eip712Field{
				{Name: "user", Type: "address", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "enabled", Type: "bool", Value: true},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xa19081574c062f40a5b6a687488b31e7553d6e41c0c55c3201378fb9f34b8745",
			mainnet:       ActionSignature{R: "0x2ced5c5561f3aa87fdd39655082a30de86133458a9dc456bdf0a1e125cfeadb1", S: "0x2147d908677a03ece096fce643501bb0c001e9574959ee6992cb04710faa6d1d", V: 28},
			testnetDigest: "0x25b34250cbdaf9cdaacb7d65b0140ead678f3f9bcfd9b4855c6aa408aa62e392",
			testnet:       ActionSignature{R: "0x575896e1ae7307faef4bee049402ba3c8f6ea2fb78549a776724c0baf93700bb", S: "0x348de7f369713ce21c35133a30f8d929156e620a7573c5da456418c21b5e595f", V: 27},
		},
		{
			name:        "userSetAbstraction",
			key:         testPrivateKey,
			primaryType: "HyperliquidTransaction:UserSetAbstraction",
			fields: []eip712Field{
				{Name: "user", Type: "address", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
				{Name: "abstraction", Type: "string", Value: "unifiedAccount"},
				{Name: "nonce", Type: "uint64", Value: uint64(1700000000000)},
			},
			mainnetDigest: "0xba1028175b6b7ab00e1e5b303949511b9e8afa66869a280b1835b4cea077a266",
			mainnet:       ActionSignature{R: "0x984629fe91fd3b4104e8579eacd7ace258f70a79742598409e3685c3793919ec", S: "0x56fb8404cc7468a826a1e5ec53661f8651581f796e214adb8b348863cbb0d97", V: 28},
			testnetDigest: "0x1a45a25355dc6a882818ee085c823ad23403fdb4511a20a4230466dbba0a4711",
			testnet:       ActionSignature{R: "0xee7be37e47a1ad42268aa9f107115d1546e61b4103207442bd55c5dadc3ae42d", S: "0x684a557acd67d9bfcaa0250c6b9adc84c2c899ee46a8976eaf37a18eaeee926", V: 28},
		},
		{
			name:        "typescript usdSend",
			key:         typeScriptSDKTestKey,
			primaryType: "HyperliquidTransaction:UsdSend",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x1234567890123456789012345678901234567890"},
				{Name: "amount", Type: "string", Value: "1000"},
				{Name: "time", Type: "uint64", Value: uint64(1234567890)},
			},
			mainnet: ActionSignature{R: "0xf777c38efe7c24cc71209526ae608f4e384d0586edf578f0e97b4b9f7c7adcc6", S: "0x104a4a97c48ae77bf5bd777bdd45fe72d8f5ff29116b5ff64fd8cfe4ea610786", V: 28},
		},
		{
			name:        "rust usdSend",
			key:         rustSDKTestKey,
			primaryType: "HyperliquidTransaction:UsdSend",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x0D1d9635D0640821d15e323ac8AdADfA9c111414"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: uint64(1690393044548)},
			},
			testnet: ActionSignature{R: "0x214d507bbdaebba52fa60928f904a8b2df73673e3baba6133d66fe846c7ef704", S: "0x51e82453a6d8db124e7ed6e60fa00d4b7c46e4d96cb2bd61fd81b6e8953cc9d2", V: 27},
		},
		{
			name:        "rust withdraw3",
			key:         rustSDKTestKey,
			primaryType: "HyperliquidTransaction:Withdraw",
			fields: []eip712Field{
				{Name: "destination", Type: "string", Value: "0x0D1d9635D0640821d15e323ac8AdADfA9c111414"},
				{Name: "amount", Type: "string", Value: "1"},
				{Name: "time", Type: "uint64", Value: uint64(1690393044548)},
			},
			testnet: ActionSignature{R: "0xb3172e33d2262dac2b4cb135ce3c167fda55dafa6c62213564ab728b9f9ba76b", S: "0x769a938e9f6d603dae7154c83bf5a4c3ebab81779dc2db25463a3ed663c82ae4", V: 28},
		},
		{
			name:        "rust approveBuilderFee",
			key:         rustSDKTestKey,
			primaryType: "HyperliquidTransaction:ApproveBuilderFee",
			fields: []eip712Field{
				{Name: "maxFeeRate", Type: "string", Value: "0.001%"},
				{Name: "builder", Type: "address", Value: "0x1234567890123456789012345678901234567890"},
				{Name: "nonce", Type: "uint64", Value: uint64(1583838)},
			},
			mainnet: ActionSignature{R: "0x343c9078af7c3d6683abefd0ca3b2960de5b669b716863e6dc49090853a4a3cd", S: "0x6c016301239461091a8ca3ea5ac783362526c4d9e9e624ffc563aea93d6ac239", V: 27},
			testnet: ActionSignature{R: "0x2ada43eeebeba9cfe13faf95aa84e5b8c4885c3a07cbf4536f2df5edd340d4eb", S: "0x1ed0e24f60a80d199a842258d5fa737a18d486f7d4e656268b434d226f2811d7", V: 28},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, network := range []struct {
				chain  string
				digest string
				exp    ActionSignature
			}{{chain: "Mainnet", digest: tc.mainnetDigest, exp: tc.mainnet}, {chain: "Testnet", digest: tc.testnetDigest, exp: tc.testnet}} {
				if network.exp == (ActionSignature{}) {
					continue
				}
				fields := append([]eip712Field{{Name: "hyperliquidChain", Type: "string", Value: network.chain}}, tc.fields...)
				if network.digest != "" {
					digest, err := eip712UserDigest(tc.primaryType, fields)
					require.NoErrorf(t, err, "eip712UserDigest must not error on %s", network.chain)
					assert.Equalf(t, network.digest, "0x"+hex.EncodeToString(digest[:]), "eip712UserDigest should match the vector's %s digest", network.chain)
				}
				signature, err := signUserSignedAction(tc.key, tc.primaryType, fields)
				require.NoErrorf(t, err, "signUserSignedAction must not error on %s", network.chain)
				assert.Equalf(t, network.exp, signature, "signUserSignedAction should match the vector on %s", network.chain)
			}
		})
	}
}
