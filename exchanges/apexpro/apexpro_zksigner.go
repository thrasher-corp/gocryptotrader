package apexpro

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink"
)

var (
	errZKLinkSeedsMissing = errors.New("L2Secret (seeds hex) is required for zkKey signing")
	errZKLinkPairNotFound = errors.New("l2PairId not found for symbol in V3 config")

	zkSignerMu    sync.Mutex
	cachedSigner  *zklink.Signer
	cachedSeedHex string
)

// ZKKeyInfo carries public information derived from the L2 seeds.
type ZKKeyInfo struct {
	Seeds         []byte
	L2Key         string
	PublicKeyHash []byte
}

// getOrInitZKLinkerSigner returns (or lazily initialises) the ZKLinkSigner
// from the hex-encoded seeds stored in L2Secret.
func (e *Exchange) getOrInitZKLinkerSigner(seedsHex string) (*zklink.Signer, error) {
	zkSignerMu.Lock()
	defer zkSignerMu.Unlock()

	if cachedSigner != nil && cachedSeedHex == seedsHex {
		return cachedSigner, nil
	}

	seeds, err := hex.DecodeString(strings.TrimPrefix(seedsHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid L2Secret (seeds must be hex): %w", err)
	}
	if len(seeds) == 0 {
		return nil, errZKLinkSeedsMissing
	}

	signer, err := zklink.NewZKLinkSignerFromSeeds(seeds)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZKLinkSigner: %w", err)
	}

	cachedSigner = signer
	cachedSeedHex = seedsHex
	return signer, nil
}

// ProcessZKKeyOrderSignature builds a ContractBuilder from the order params,
// Rescue-hashes it, and returns a 64-byte Schnorr signature as a hex string.
func (e *Exchange) ProcessZKKeyOrderSignature(ctx context.Context, arg *CreateOrderParams) (string, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	if creds.L2Secret == "" {
		return "", errZKLinkSeedsMissing
	}

	signer, err := e.getOrInitZKLinkerSigner(creds.L2Secret)
	if err != nil {
		return "", err
	}

	// Fetch V3 config for l2PairId and size/price resolutions
	configData, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetAllConfigDataV3: %w", err)
	}

	var (
		pairID   int64
		stepSize decimal.Decimal
		tickSize decimal.Decimal
		found    bool
	)
	for _, c := range configData.ContractConfig.PerpetualContract {
		if !c.Symbol.Equal(arg.Symbol) {
			continue
		}
		pairID, err = strconv.ParseInt(c.L2PairID, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid l2PairId %q: %w", c.L2PairID, err)
		}
		stepSize = c.StepSize.Decimal()
		tickSize = c.TickSize.Decimal()
		found = true
		break
	}
	if !found {
		return "", fmt.Errorf("%w: %s", errZKLinkPairNotFound, arg.Symbol)
	}

	// Fetch V3 account data
	accountData, err := e.GetUserAccountDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetUserAccountDataV3: %w", err)
	}

	zkAccountID, err := strconv.ParseInt(accountData.SpotAccount.ZkAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse zkAccountId: %w", err)
	}
	subAccountID, err := strconv.ParseInt(accountData.SpotAccount.DefaultSubAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse defaultSubAccountId: %w", err)
	}
	nonce := accountData.SpotAccount.Nonce

	// Fee rates → ZKLink units (rate × 100000), fitting in 8 bits each
	takerFeeRate := accountData.ContractAccount.TakerFeeRate.Float64()
	makerFeeRate := accountData.ContractAccount.MakerFeeRate.Float64()
	takerFeeInt := big.NewInt(int64(takerFeeRate * 100000))
	makerFeeInt := big.NewInt(int64(makerFeeRate * 100000))

	// Convert size and price to ZKLink integer representation
	sizeD := decimal.NewFromFloat(arg.Size)
	priceD := decimal.NewFromFloat(arg.Price)

	var sizeInt, priceInt *big.Int
	if !stepSize.IsZero() {
		sizeInt = sizeD.Div(stepSize).BigInt()
	} else {
		sizeInt = sizeD.BigInt()
	}
	if !tickSize.IsZero() {
		priceInt = priceD.Div(tickSize).BigInt()
	} else {
		priceInt = priceD.BigInt()
	}

	isBuy := strings.EqualFold(arg.Side, order.Buy.String())

	builder := &zklink.ContractBuilder{
		AccountID:    big.NewInt(zkAccountID),
		SubAccountID: big.NewInt(subAccountID),
		SlotID:       big.NewInt(0),
		Nonce:        big.NewInt(nonce),
		PairID:       big.NewInt(pairID),
		Size:         sizeInt,
		Price:        priceInt,
		Direction:    isBuy,
		TakerFeeRate: takerFeeInt,
		MakerFeeRate: makerFeeInt,
		HasSubsidy:   false,
	}

	sig, err := signer.Sign(builder.GetBytes())
	if err != nil {
		return "", fmt.Errorf("zklink sign order: %w", err)
	}
	return hex.EncodeToString(sig[:]), nil
}

// ProcessZKKeyWithdrawalSignature builds a WithdrawBuilder from the withdrawal params,
// Rescue-hashes it, and returns a 64-byte Schnorr signature as a hex string.
func (e *Exchange) ProcessZKKeyWithdrawalSignature(ctx context.Context, arg *AssetWithdrawalParams) (string, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	if creds.L2Secret == "" {
		return "", errZKLinkSeedsMissing
	}

	signer, err := e.getOrInitZKLinkerSigner(creds.L2Secret)
	if err != nil {
		return "", err
	}

	// Get V3 account data for ZkAccountID / DefaultSubAccountID / Nonce
	accountData, err := e.GetUserAccountDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetUserAccountDataV3: %w", err)
	}

	zkAccountID, err := strconv.ParseInt(accountData.SpotAccount.ZkAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse zkAccountId: %w", err)
	}
	subAccountID, err := strconv.ParseInt(accountData.SpotAccount.DefaultSubAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse defaultSubAccountId: %w", err)
	}
	nonce := accountData.SpotAccount.Nonce

	// Fetch V3 config to look up token IDs for L2SourceToken / L1TargetToken
	configData, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetAllConfigDataV3: %w", err)
	}

	var l2TokenID, l1TokenID int64
	for _, asset := range configData.SpotConfig.Assets {
		if strings.EqualFold(asset.Token, arg.L2SourceTokenID.String()) {
			l2TokenID, err = strconv.ParseInt(asset.TokenID, 10, 64)
			if err != nil {
				return "", fmt.Errorf("parse l2 tokenId: %w", err)
			}
		}
		if strings.EqualFold(asset.Token, arg.L1TargetTokenID.String()) {
			l1TokenID, err = strconv.ParseInt(asset.TokenID, 10, 64)
			if err != nil {
				return "", fmt.Errorf("parse l1 tokenId: %w", err)
			}
		}
	}

	// Parse toChainId
	toChainID, err := strconv.ParseInt(arg.ToChainID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse toChainId: %w", err)
	}

	// ToAddress: ethereum address as big.Int
	toAddress, ok := new(big.Int).SetString(strings.TrimPrefix(arg.EthereumAddress, "0x"), 16)
	if !ok {
		return "", fmt.Errorf("invalid ethereum address: %s", arg.EthereumAddress)
	}

	// Amount and fee as integers (using 1e8 resolution; adjust per token decimals)
	const resolution = 1e8
	amountInt := big.NewInt(int64(arg.Amount * resolution))
	feeInt := big.NewInt(int64(arg.Fee * resolution))

	// Timestamp as Unix seconds
	ts := big.NewInt(arg.Timestamp.Unix())
	if arg.Timestamp.IsZero() {
		ts = big.NewInt(time.Now().Unix())
	}

	builder := &zklink.WithdrawBuilder{
		AccountID:        big.NewInt(zkAccountID),
		SubAccountID:     big.NewInt(subAccountID),
		ToChainID:        big.NewInt(toChainID),
		ToAddress:        toAddress,
		L2SourceToken:    big.NewInt(l2TokenID),
		L1TargetToken:    big.NewInt(l1TokenID),
		Amount:           amountInt,
		Fee:              feeInt,
		Nonce:            big.NewInt(nonce),
		WithdrawFeeRatio: big.NewInt(0),
		WithdrawToL1:     true,
		Timestamp:        ts,
	}

	sig, err := signer.Sign(builder.GetBytes())
	if err != nil {
		return "", fmt.Errorf("zklink sign withdrawal: %w", err)
	}
	return hex.EncodeToString(sig[:]), nil
}

// ProcessZKKeyTransferSignature builds a TransferBuilder from the fast-withdrawal params,
// Rescue-hashes it, and returns a 64-byte Schnorr signature as a hex string.
func (e *Exchange) ProcessZKKeyTransferSignature(ctx context.Context, arg *FastWithdrawalParams) (string, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	if creds.L2Secret == "" {
		return "", errZKLinkSeedsMissing
	}

	signer, err := e.getOrInitZKLinkerSigner(creds.L2Secret)
	if err != nil {
		return "", err
	}

	// Get V3 account data
	accountData, err := e.GetUserAccountDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetUserAccountDataV3: %w", err)
	}

	fromAccountID, err := strconv.ParseInt(accountData.SpotAccount.ZkAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse zkAccountId: %w", err)
	}
	fromSubAccountID, err := strconv.ParseInt(accountData.SpotAccount.DefaultSubAccountID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse defaultSubAccountId: %w", err)
	}
	nonce := accountData.SpotAccount.Nonce

	// Fetch V3 config for token ID and LP account
	configData, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return "", fmt.Errorf("GetAllConfigDataV3: %w", err)
	}

	var tokenID int64
	for _, asset := range configData.SpotConfig.Assets {
		if strings.EqualFold(asset.Token, arg.Asset.String()) {
			tokenID, err = strconv.ParseInt(asset.TokenID, 10, 64)
			if err != nil {
				return "", fmt.Errorf("parse tokenId: %w", err)
			}
			break
		}
	}

	// LP account ID from V3 global config
	toAccountID, ok := new(big.Int).SetString(configData.SpotConfig.Global.PerpLpAccountID, 10)
	if !ok {
		toAccountID = big.NewInt(0)
	}
	toSubAccountID, ok := new(big.Int).SetString(configData.SpotConfig.Global.PerpLpSubAccount, 10)
	if !ok {
		toSubAccountID = big.NewInt(0)
	}

	// LP L2 address as big.Int
	toAddress, _ := new(big.Int).SetString(
		strings.TrimPrefix(configData.SpotConfig.Global.PerpLpSubAccountL2Key, "0x"), 16)
	if toAddress == nil {
		toAddress = big.NewInt(0)
	}

	const resolution = 1e8
	amountInt := big.NewInt(int64(arg.Amount * resolution))

	ts := big.NewInt(time.Now().Unix())
	if arg.Expiration > 0 {
		ts = big.NewInt(arg.Expiration / 1000) // convert ms to seconds
	}

	builder := &zklink.TransferBuilder{
		AccountID:        big.NewInt(fromAccountID),
		ToAddress:        toAddress,
		FromSubAccountID: big.NewInt(fromSubAccountID),
		ToSubAccountID:   toSubAccountID,
		Token:            big.NewInt(tokenID),
		Amount:           amountInt,
		Fee:              big.NewInt(0),
		Nonce:            big.NewInt(nonce),
		Timestamp:        ts,
	}
	_ = toAccountID // LP account used for routing, not part of the signed builder fields

	sig, err := signer.Sign(builder.GetBytes())
	if err != nil {
		return "", fmt.Errorf("zklink sign transfer: %w", err)
	}
	return hex.EncodeToString(sig[:]), nil
}
