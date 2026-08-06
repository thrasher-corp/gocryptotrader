package hyperliquid

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/crypto/sha3"
)

const (
	ethereumAddressByteLength = 20
	privateKeyByteLength      = 32
	signatureComponentLength  = 32
	ethereumSignatureVOffset  = 27
	eip712ChainID             = 1337
	userSignedChainID         = 0x66eee
	userSignedChainIDHex      = "0x66eee"
)

var (
	errEIP712Field             = errors.New("invalid EIP-712 field")
	errEIP712PrimaryType       = errors.New("invalid EIP-712 primary type")
	errInvalidAddress          = errors.New("invalid EVM address")
	errInvalidPrivateKey       = errors.New("invalid EVM private key")
	errInvalidRecoveryID       = errors.New("invalid ECDSA recovery ID")
	errPrivateKeyRequired      = errors.New("private key is required for signed exchange actions")
	errSigningRecoveryMismatch = errors.New("signature did not recover the signing key")
	errWireNumberRounding      = errors.New("number cannot be represented with 8 decimal places")
)

type l1Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V uint8  `json:"v"`
}

type eip712Field struct {
	Name  string
	Type  string
	Value any
}

type limitOrderTypeWire struct {
	TimeInForce string `json:"tif" msgpack:"tif"`
}

type triggerOrderTypeWire struct {
	IsMarket           bool   `json:"isMarket"  msgpack:"isMarket"`
	TriggerPrice       string `json:"triggerPx" msgpack:"triggerPx"`
	TakeProfitStopLoss string `json:"tpsl"      msgpack:"tpsl"`
}

type orderTypeWire struct {
	Limit   *limitOrderTypeWire   `json:"limit,omitempty"   msgpack:"limit,omitempty"`
	Trigger *triggerOrderTypeWire `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

type orderWire struct {
	AssetID       uint64        `json:"a"           msgpack:"a"`
	IsBuy         bool          `json:"b"           msgpack:"b"`
	Price         string        `json:"p"           msgpack:"p"`
	Size          string        `json:"s"           msgpack:"s"`
	ReduceOnly    bool          `json:"r"           msgpack:"r"`
	Type          orderTypeWire `json:"t"           msgpack:"t"`
	ClientOrderID string        `json:"c,omitempty" msgpack:"c,omitempty"`
}

type orderAction struct {
	Type     string      `json:"type"     msgpack:"type"`
	Orders   []orderWire `json:"orders"   msgpack:"orders"`
	Grouping string      `json:"grouping" msgpack:"grouping"`
}

type modifyWire struct {
	OrderID any       `json:"oid"   msgpack:"oid"`
	Order   orderWire `json:"order" msgpack:"order"`
}

type batchModifyAction struct {
	Type     string       `json:"type"     msgpack:"type"`
	Modifies []modifyWire `json:"modifies" msgpack:"modifies"`
}

type cancelWire struct {
	AssetID uint64 `json:"a" msgpack:"a"`
	OrderID uint64 `json:"o" msgpack:"o"`
}

type cancelAction struct {
	Type    string       `json:"type"    msgpack:"type"`
	Cancels []cancelWire `json:"cancels" msgpack:"cancels"`
}

type cancelByClientOrderIDWire struct {
	AssetID       uint64 `json:"asset" msgpack:"asset"`
	ClientOrderID string `json:"cloid" msgpack:"cloid"`
}

type cancelByClientOrderIDAction struct {
	Type    string                      `json:"type"    msgpack:"type"`
	Cancels []cancelByClientOrderIDWire `json:"cancels" msgpack:"cancels"`
}

type updateLeverageAction struct {
	Type     string `json:"type"     msgpack:"type"`
	AssetID  uint64 `json:"asset"    msgpack:"asset"`
	IsCross  bool   `json:"isCross"  msgpack:"isCross"`
	Leverage uint64 `json:"leverage" msgpack:"leverage"`
}

type signedActionRequest struct {
	Action       any         `json:"action"`
	Nonce        uint64      `json:"nonce"`
	Signature    l1Signature `json:"signature"`
	VaultAddress string      `json:"vaultAddress,omitempty"`
	ExpiresAfter *uint64     `json:"expiresAfter,omitempty"`
}

func normaliseAddress(address string) (normalised string, raw [ethereumAddressByteLength]byte, err error) {
	address = strings.TrimSpace(address)
	if len(address) != 2+ethereumAddressByteLength*2 ||
		!strings.EqualFold(address[:2], "0x") {
		return "", raw, fmt.Errorf("%w: expected 0x-prefixed 20-byte hexadecimal value", errInvalidAddress)
	}
	encoded := address[2:]
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return "", raw, fmt.Errorf("%w: %w", errInvalidAddress, err)
	}
	copy(raw[:], decoded)
	if bytes.Equal(raw[:], make([]byte, ethereumAddressByteLength)) {
		return "", raw, fmt.Errorf("%w: zero address", errInvalidAddress)
	}
	hasLower := strings.IndexFunc(encoded, func(r rune) bool { return r >= 'a' && r <= 'f' }) != -1
	hasUpper := strings.IndexFunc(encoded, func(r rune) bool { return r >= 'A' && r <= 'F' }) != -1
	if hasLower && hasUpper {
		lower := strings.ToLower(encoded)
		checksum := keccak256([]byte(lower))
		for i := range encoded {
			if encoded[i] >= '0' && encoded[i] <= '9' {
				continue
			}
			nibble := checksum[i/2] & 0x0f
			if i%2 == 0 {
				nibble = checksum[i/2] >> 4
			}
			isUpper := encoded[i] >= 'A' && encoded[i] <= 'F'
			if isUpper != (nibble >= 8) {
				return "", raw, fmt.Errorf("%w: invalid EIP-55 checksum", errInvalidAddress)
			}
		}
	}
	return "0x" + strings.ToLower(encoded), raw, nil
}

func parsePrivateKey(secret string) (*secp256k1.PrivateKey, error) {
	secret = strings.TrimSpace(secret)
	if strings.HasPrefix(secret, "0x") || strings.HasPrefix(secret, "0X") {
		secret = secret[2:]
	}
	if len(secret) != privateKeyByteLength*2 {
		return nil, fmt.Errorf("%w: expected 32-byte hexadecimal scalar", errInvalidPrivateKey)
	}
	raw, err := hex.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidPrivateKey, err)
	}
	defer clear(raw)
	var scalar secp256k1.ModNScalar
	if scalar.SetByteSlice(raw) || scalar.IsZero() {
		scalar.Zero()
		return nil, fmt.Errorf("%w: scalar is outside secp256k1 range", errInvalidPrivateKey)
	}
	key := secp256k1.NewPrivateKey(&scalar)
	scalar.Zero()
	return key, nil
}

func keccak256(parts ...[]byte) [32]byte {
	hasher := sha3.NewLegacyKeccak256()
	for i := range parts {
		_, _ = hasher.Write(parts[i])
	}
	var digest [32]byte
	hasher.Sum(digest[:0])
	return digest
}

func privateKeyAddress(key *secp256k1.PrivateKey) string {
	publicKey := key.PubKey().SerializeUncompressed()
	digest := keccak256(publicKey[1:])
	return "0x" + hex.EncodeToString(digest[len(digest)-ethereumAddressByteLength:])
}

func actionHash(action any, vaultAddress string, nonce uint64, expiresAfter *uint64) ([32]byte, error) {
	var actionBuffer bytes.Buffer
	encoder := msgpack.NewEncoder(&actionBuffer)
	encoder.UseCompactInts(true)
	if err := encoder.Encode(action); err != nil {
		return [32]byte{}, err
	}
	encodedAction := actionBuffer.Bytes()
	var nonceBytes [8]byte
	binary.BigEndian.PutUint64(nonceBytes[:], nonce)
	preimage := make([]byte, 0, len(encodedAction)+8+1+ethereumAddressByteLength+1+8)
	preimage = append(preimage, encodedAction...)
	preimage = append(preimage, nonceBytes[:]...)
	if vaultAddress == "" {
		preimage = append(preimage, 0)
	} else {
		_, raw, err := normaliseAddress(vaultAddress)
		if err != nil {
			return [32]byte{}, err
		}
		preimage = append(preimage, 1)
		preimage = append(preimage, raw[:]...)
	}
	if expiresAfter != nil {
		var expiryBytes [8]byte
		binary.BigEndian.PutUint64(expiryBytes[:], *expiresAfter)
		preimage = append(preimage, 0)
		preimage = append(preimage, expiryBytes[:]...)
	}
	return keccak256(preimage), nil
}

func eip712AgentDigest(connectionID [32]byte, isMainnet bool) [32]byte {
	domainTypeHash := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	agentTypeHash := keccak256([]byte("Agent(string source,bytes32 connectionId)"))
	nameHash := keccak256([]byte("Exchange"))
	versionHash := keccak256([]byte("1"))
	source := "b"
	if isMainnet {
		source = "a"
	}
	sourceHash := keccak256([]byte(source))
	var chainID [32]byte
	binary.BigEndian.PutUint64(chainID[24:], eip712ChainID)
	domainHash := keccak256(
		domainTypeHash[:],
		nameHash[:],
		versionHash[:],
		chainID[:],
		make([]byte, 32),
	)
	agentHash := keccak256(agentTypeHash[:], sourceHash[:], connectionID[:])
	return keccak256([]byte{0x19, 0x01}, domainHash[:], agentHash[:])
}

func eip712UserDigest(primaryType string, fields []eip712Field) ([32]byte, error) {
	primaryType = strings.TrimSpace(primaryType)
	if primaryType == "" {
		return [32]byte{}, errEIP712PrimaryType
	}
	var typeDefinition strings.Builder
	typeDefinition.WriteString(primaryType)
	typeDefinition.WriteByte('(')
	encodedFields := make([]byte, 0, 32*(len(fields)+1))
	for i := range fields {
		if i != 0 {
			typeDefinition.WriteByte(',')
		}
		if strings.TrimSpace(fields[i].Name) == "" || strings.TrimSpace(fields[i].Type) == "" {
			return [32]byte{}, fmt.Errorf("%w: field %d requires a name and type", errEIP712Field, i)
		}
		typeDefinition.WriteString(fields[i].Type)
		typeDefinition.WriteByte(' ')
		typeDefinition.WriteString(fields[i].Name)
		var encoded [32]byte
		switch fields[i].Type {
		case "string":
			value, ok := fields[i].Value.(string)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a string", errEIP712Field, fields[i].Name)
			}
			encoded = keccak256([]byte(value))
		case "bool":
			value, ok := fields[i].Value.(bool)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a bool", errEIP712Field, fields[i].Name)
			}
			if value {
				encoded[31] = 1
			}
		case "uint64":
			value, ok := fields[i].Value.(uint64)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a uint64", errEIP712Field, fields[i].Name)
			}
			binary.BigEndian.PutUint64(encoded[24:], value)
		default:
			return [32]byte{}, fmt.Errorf("%w: unsupported type %q", errEIP712Field, fields[i].Type)
		}
		encodedFields = append(encodedFields, encoded[:]...)
	}
	typeDefinition.WriteByte(')')
	typeHash := keccak256([]byte(typeDefinition.String()))
	structHash := keccak256(typeHash[:], encodedFields)

	domainTypeHash := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := keccak256([]byte("HyperliquidSignTransaction"))
	versionHash := keccak256([]byte("1"))
	var chainID [32]byte
	binary.BigEndian.PutUint64(chainID[24:], userSignedChainID)
	domainHash := keccak256(
		domainTypeHash[:],
		nameHash[:],
		versionHash[:],
		chainID[:],
		make([]byte, 32),
	)
	return keccak256([]byte{0x19, 0x01}, domainHash[:], structHash[:]), nil
}

func signL1Action(secret string, action any, vaultAddress string, nonce uint64, expiresAfter *uint64, isMainnet bool) (l1Signature, error) {
	key, err := parsePrivateKey(secret)
	if err != nil {
		return l1Signature{}, err
	}
	defer key.Zero()
	connectionID, err := actionHash(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return l1Signature{}, err
	}
	digest := eip712AgentDigest(connectionID, isMainnet)
	compact := secpECDSA.SignCompact(key, digest[:], false)
	return validateCompactSignature(key, digest, compact)
}

func signUserSignedAction(secret, primaryType string, fields []eip712Field) (l1Signature, error) {
	key, err := parsePrivateKey(secret)
	if err != nil {
		return l1Signature{}, err
	}
	defer key.Zero()
	digest, err := eip712UserDigest(primaryType, fields)
	if err != nil {
		return l1Signature{}, err
	}
	return validateCompactSignature(key, digest, secpECDSA.SignCompact(key, digest[:], false))
}

func validateCompactSignature(key *secp256k1.PrivateKey, digest [32]byte, compact []byte) (l1Signature, error) {
	if len(compact) != 1+signatureComponentLength*2 {
		return l1Signature{}, fmt.Errorf("%w: invalid compact signature length %d", errInvalidRecoveryID, len(compact))
	}
	if compact[0] < ethereumSignatureVOffset ||
		compact[0] > ethereumSignatureVOffset+1 {
		return l1Signature{}, fmt.Errorf("%w: %d", errInvalidRecoveryID, compact[0])
	}
	recovered, _, err := secpECDSA.RecoverCompact(compact, digest[:])
	if err != nil {
		return l1Signature{}, err
	}
	if subtle.ConstantTimeCompare(recovered.SerializeUncompressed(), key.PubKey().SerializeUncompressed()) != 1 {
		return l1Signature{}, errSigningRecoveryMismatch
	}
	return l1Signature{
		R: formatSignatureComponent(compact[1 : 1+signatureComponentLength]),
		S: formatSignatureComponent(compact[1+signatureComponentLength:]),
		V: compact[0],
	}, nil
}

func formatSignatureComponent(component []byte) string {
	encoded := strings.TrimLeft(hex.EncodeToString(component), "0")
	if encoded == "" {
		encoded = "0"
	}
	return "0x" + encoded
}

func floatToWire(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("%w: %v", errWireNumberRounding, value)
	}
	roundedText := strconv.FormatFloat(value, 'f', 8, 64)
	rounded, _ := strconv.ParseFloat(roundedText, 64) // FormatFloat always returns a valid floating-point number.
	if math.Abs(rounded-value) >= 1e-12 {
		return "", fmt.Errorf("%w: %v", errWireNumberRounding, value)
	}
	roundedText = strings.TrimRight(strings.TrimRight(roundedText, "0"), ".")
	if roundedText == "" || roundedText == "-0" {
		return "0", nil
	}
	return roundedText, nil
}

func (e *Exchange) nextNonce() uint64 {
	now := uint64(time.Now().UnixMilli()) //nolint:gosec // Unix milliseconds are positive for the supported runtime epoch.
	for {
		previous := e.lastNonce.Load()
		next := now
		if next <= previous {
			next = previous + 1
		}
		if e.lastNonce.CompareAndSwap(previous, next) {
			return next
		}
	}
}
