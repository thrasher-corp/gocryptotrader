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
	// userSignedChainID is the EIP-712 domain chain used by the Python SDK for user-signed actions; hyperliquidChain,
	// not this value, selects mainnet or testnet
	userSignedChainID    = 0x66eee
	userSignedChainIDHex = "0x66eee"
)

// EIP-712 types of user-signed action fields
const (
	eip712TypeString  = "string"
	eip712TypeBool    = "bool"
	eip712TypeUint32  = "uint32"
	eip712TypeUint64  = "uint64"
	eip712TypeAddress = "address"
	eip712TypeBytes   = "bytes"
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

// ActionSignature is an ECDSA signature in the r, s and v form the exchange endpoint expects
type ActionSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V uint8  `json:"v"`
}

// eip712Field is one field of a user-signed action's EIP-712 message
type eip712Field struct {
	Name  string
	Type  string
	Value any
}

// normaliseAddress validates a 0x-prefixed EVM address, including its EIP-55 checksum when mixed case, and returns it
// lower-cased with its raw bytes
func normaliseAddress(address string) (normalised string, raw [ethereumAddressByteLength]byte, err error) {
	address = strings.TrimSpace(address)
	if len(address) != 2+ethereumAddressByteLength*2 || !strings.EqualFold(address[:2], "0x") {
		return "", raw, fmt.Errorf("%w: expected 0x-prefixed 20-byte hexadecimal value", errInvalidAddress)
	}
	encoded := address[2:]
	if _, err := hex.Decode(raw[:], []byte(encoded)); err != nil {
		return "", raw, fmt.Errorf("%w: %w", errInvalidAddress, err)
	}
	hasLower := strings.ContainsFunc(encoded, func(r rune) bool { return r >= 'a' && r <= 'f' })
	hasUpper := strings.ContainsFunc(encoded, func(r rune) bool { return r >= 'A' && r <= 'F' })
	if hasLower && hasUpper {
		checksum := keccak256([]byte(strings.ToLower(encoded)))
		for i := range len(encoded) {
			if encoded[i] >= '0' && encoded[i] <= '9' {
				continue
			}
			nibble := checksum[i/2] & 0x0f
			if i%2 == 0 {
				nibble = checksum[i/2] >> 4
			}
			if isUpper := encoded[i] >= 'A' && encoded[i] <= 'F'; isUpper != (nibble >= 8) {
				return "", raw, fmt.Errorf("%w: invalid EIP-55 checksum", errInvalidAddress)
			}
		}
	}
	return "0x" + strings.ToLower(encoded), raw, nil
}

// normaliseRecipientAddress validates an address that receives funds or authority, which normaliseAddress allows to be
// the zero address because info requests can query it
func normaliseRecipientAddress(address string) (string, error) {
	normalised, raw, err := normaliseAddress(address)
	if err != nil {
		return "", err
	}
	if raw == [ethereumAddressByteLength]byte{} {
		return "", fmt.Errorf("%w: zero address", errInvalidAddress)
	}
	return normalised, nil
}

// parsePrivateKey decodes a hexadecimal secp256k1 private key, with or without a 0x prefix
func parsePrivateKey(secret string) (*secp256k1.PrivateKey, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) >= 2 && strings.EqualFold(secret[:2], "0x") {
		secret = secret[2:]
	}
	if len(secret) != privateKeyByteLength*2 {
		return nil, fmt.Errorf("%w: expected 32-byte hexadecimal scalar", errInvalidPrivateKey)
	}
	var raw [privateKeyByteLength]byte
	defer clear(raw[:])
	// The decoding error quotes the invalid character, which is part of the secret
	if _, err := hex.Decode(raw[:], []byte(secret)); err != nil {
		return nil, fmt.Errorf("%w: expected 32-byte hexadecimal scalar", errInvalidPrivateKey)
	}
	var scalar secp256k1.ModNScalar
	if scalar.SetBytes(&raw) != 0 || scalar.IsZero() {
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

// privateKeyAddress derives the lower-case EVM address of a private key
func privateKeyAddress(key *secp256k1.PrivateKey) string {
	publicKey := key.PubKey().SerializeUncompressed()
	digest := keccak256(publicKey[1:])
	return "0x" + hex.EncodeToString(digest[len(digest)-ethereumAddressByteLength:])
}

// msgpackAction encodes an L1 action as Hyperliquid hashes it: struct fields in declaration order, with the smallest
// integer encodings
func msgpackAction(action any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := msgpack.NewEncoder(&buffer)
	encoder.UseCompactInts(true)
	if err := encoder.Encode(action); err != nil {
		return nil, fmt.Errorf("encoding action as msgpack: %w", err)
	}
	return buffer.Bytes(), nil
}

// actionHash returns the connection ID an L1 action is signed over: the msgpack action, nonce, vault and expiry
func actionHash(action any, vaultAddress string, nonce uint64, expiresAfter *uint64) ([32]byte, error) {
	preimage, err := msgpackAction(action)
	if err != nil {
		return [32]byte{}, err
	}
	preimage = binary.BigEndian.AppendUint64(preimage, nonce)
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
		preimage = append(preimage, 0)
		preimage = binary.BigEndian.AppendUint64(preimage, *expiresAfter)
	}
	return keccak256(preimage), nil
}

// eip712DomainHash returns the hash of an EIP-712 domain with a zero verifying contract
func eip712DomainHash(name string, chainID uint64) [32]byte {
	domainTypeHash := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := keccak256([]byte(name))
	versionHash := keccak256([]byte("1"))
	var encodedChainID [32]byte
	binary.BigEndian.PutUint64(encodedChainID[24:], chainID)
	var verifyingContract [32]byte
	return keccak256(domainTypeHash[:], nameHash[:], versionHash[:], encodedChainID[:], verifyingContract[:])
}

// eip712AgentDigest returns the digest of the phantom agent message that L1 actions are signed with
func eip712AgentDigest(connectionID [32]byte, isMainnet bool) [32]byte {
	agentTypeHash := keccak256([]byte("Agent(string source,bytes32 connectionId)"))
	source := "b"
	if isMainnet {
		source = "a"
	}
	sourceHash := keccak256([]byte(source))
	domainHash := eip712DomainHash("Exchange", eip712ChainID)
	agentHash := keccak256(agentTypeHash[:], sourceHash[:], connectionID[:])
	return keccak256([]byte{0x19, 0x01}, domainHash[:], agentHash[:])
}

// eip712UserDigest returns the digest of a user-signed action's EIP-712 message
func eip712UserDigest(primaryType string, fields []eip712Field) ([32]byte, error) {
	primaryType = strings.TrimSpace(primaryType)
	if primaryType == "" {
		return [32]byte{}, errEIP712PrimaryType
	}
	var typeDefinition strings.Builder
	typeDefinition.WriteString(primaryType)
	typeDefinition.WriteByte('(')
	encodedFields := make([]byte, 0, 32*len(fields))
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
		case eip712TypeString:
			value, ok := fields[i].Value.(string)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a string", errEIP712Field, fields[i].Name)
			}
			encoded = keccak256([]byte(value))
		case eip712TypeBool:
			value, ok := fields[i].Value.(bool)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a bool", errEIP712Field, fields[i].Name)
			}
			if value {
				encoded[31] = 1
			}
		case eip712TypeUint64:
			value, ok := fields[i].Value.(uint64)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a uint64", errEIP712Field, fields[i].Name)
			}
			binary.BigEndian.PutUint64(encoded[24:], value)
		case eip712TypeUint32:
			value, ok := fields[i].Value.(uint32)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be a uint32", errEIP712Field, fields[i].Name)
			}
			binary.BigEndian.PutUint32(encoded[28:], value)
		case eip712TypeAddress:
			value, ok := fields[i].Value.(string)
			if !ok {
				return [32]byte{}, fmt.Errorf("%w: %s must be an address string", errEIP712Field, fields[i].Name)
			}
			_, raw, err := normaliseAddress(value)
			if err != nil {
				return [32]byte{}, fmt.Errorf("%w: %s: %w", errEIP712Field, fields[i].Name, err)
			}
			copy(encoded[32-ethereumAddressByteLength:], raw[:])
		case eip712TypeBytes:
			// Byte fields are carried as 0x-prefixed hexadecimal, as the action's JSON body sends them
			value, ok := fields[i].Value.(string)
			if !ok || len(value) < 2 || !strings.EqualFold(value[:2], "0x") {
				return [32]byte{}, fmt.Errorf("%w: %s must be 0x-prefixed hexadecimal", errEIP712Field, fields[i].Name)
			}
			decoded, err := hex.DecodeString(value[2:])
			if err != nil {
				return [32]byte{}, fmt.Errorf("%w: %s: %w", errEIP712Field, fields[i].Name, err)
			}
			encoded = keccak256(decoded)
		default:
			return [32]byte{}, fmt.Errorf("%w: unsupported type %q", errEIP712Field, fields[i].Type)
		}
		encodedFields = append(encodedFields, encoded[:]...)
	}
	typeDefinition.WriteByte(')')
	typeHash := keccak256([]byte(typeDefinition.String()))
	structHash := keccak256(typeHash[:], encodedFields)
	domainHash := eip712DomainHash("HyperliquidSignTransaction", userSignedChainID)
	return keccak256([]byte{0x19, 0x01}, domainHash[:], structHash[:]), nil
}

// signL1Action signs a trading action, such as an order or cancel, with the phantom agent scheme
func signL1Action(secret string, action any, vaultAddress string, nonce uint64, expiresAfter *uint64, isMainnet bool) (ActionSignature, error) {
	key, err := parsePrivateKey(secret)
	if err != nil {
		return ActionSignature{}, err
	}
	defer key.Zero()
	connectionID, err := actionHash(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return ActionSignature{}, err
	}
	digest := eip712AgentDigest(connectionID, isMainnet)
	return validateCompactSignature(key, digest, secpECDSA.SignCompact(key, digest[:], false))
}

// signUserSignedAction signs a human-readable action, such as a transfer, with the EIP-712 user-signed scheme
func signUserSignedAction(secret, primaryType string, fields []eip712Field) (ActionSignature, error) {
	key, err := parsePrivateKey(secret)
	if err != nil {
		return ActionSignature{}, err
	}
	defer key.Zero()
	digest, err := eip712UserDigest(primaryType, fields)
	if err != nil {
		return ActionSignature{}, err
	}
	return validateCompactSignature(key, digest, secpECDSA.SignCompact(key, digest[:], false))
}

// validateCompactSignature checks a compact signature recovers the signing key before splitting it into r, s and v
func validateCompactSignature(key *secp256k1.PrivateKey, digest [32]byte, compact []byte) (ActionSignature, error) {
	if len(compact) != 1+signatureComponentLength*2 {
		return ActionSignature{}, fmt.Errorf("%w: invalid compact signature length %d", errInvalidRecoveryID, len(compact))
	}
	if compact[0] < ethereumSignatureVOffset || compact[0] > ethereumSignatureVOffset+1 {
		return ActionSignature{}, fmt.Errorf("%w: %d", errInvalidRecoveryID, compact[0])
	}
	recovered, _, err := secpECDSA.RecoverCompact(compact, digest[:])
	if err != nil {
		return ActionSignature{}, fmt.Errorf("recovering signature key: %w", err)
	}
	if subtle.ConstantTimeCompare(recovered.SerializeUncompressed(), key.PubKey().SerializeUncompressed()) != 1 {
		return ActionSignature{}, errSigningRecoveryMismatch
	}
	return ActionSignature{
		R: formatSignatureComponent(compact[1 : 1+signatureComponentLength]),
		S: formatSignatureComponent(compact[1+signatureComponentLength:]),
		V: compact[0],
	}, nil
}

// formatSignatureComponent encodes a signature component as 0x-prefixed hexadecimal without leading zeros
func formatSignatureComponent(component []byte) string {
	encoded := strings.TrimLeft(hex.EncodeToString(component), "0")
	if encoded == "" {
		encoded = "0"
	}
	return "0x" + encoded
}

// floatToWire formats a number with at most eight decimals and no trailing zeros, as signed actions require
func floatToWire(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("%w: %v", errWireNumberRounding, value)
	}
	roundedText := strconv.FormatFloat(value, 'f', 8, 64)
	rounded, _ := strconv.ParseFloat(roundedText, 64) // FormatFloat of a finite value always parses
	// A value below the tolerance would otherwise round to zero, signing a non-zero size, price or amount as zero
	if math.Abs(rounded-value) >= 1e-12 || (rounded == 0 && value != 0) {
		return "", fmt.Errorf("%w: %v", errWireNumberRounding, value)
	}
	roundedText = strings.TrimRight(strings.TrimRight(roundedText, "0"), ".")
	if roundedText == "-0" {
		return "0", nil
	}
	return roundedText, nil
}
