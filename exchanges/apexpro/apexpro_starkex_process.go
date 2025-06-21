package apexpro

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/hash/solsha3"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/starkex"
)

var (
	errExpirationTimeRequired         = errors.New("expiration time is required")
	errContractNotFound               = errors.New("contract not found")
	errSettlementCurrencyInfoNotFound = errors.New("settlement currency information not found")
	errInvalidAssetID                 = errors.New("invalid asset ID provided")
	errInvalidPositionIDMissing       = errors.New("invalid position or account ID")
	errL2CredentialsMismatch          = errors.New("l2 credentials mismatch")
	errTokenDetailIsMissing           = errors.New("token detail is missing")
)

const orderSignatureExpirationBuffersHours = 24 * 7

// ProcessOrderSignature processes order request parameter and generates a starkEx signature
func (ap *Apexpro) ProcessOrderSignature(ctx context.Context, arg *CreateOrderParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	if creds.L2Secret == "" {
		return "", starkex.ErrInvalidPrivateKey
	}
	price := decimal.NewFromFloat(arg.Price)
	size := decimal.NewFromFloat(arg.Size)

	// check if the all symbols config is loaded, if not load
	if ap.SymbolsConfig == nil {
		ap.SymbolsConfig, err = ap.GetAllSymbolsConfigDataV1(ctx)
		if err != nil {
			return "", err
		}
	}
	var contractDetail *PerpetualContractDetail
	for a := range ap.SymbolsConfig.Data.PerpetualContract {
		if ap.SymbolsConfig.Data.PerpetualContract[a].Symbol == arg.Symbol.String() {
			contractDetail = &ap.SymbolsConfig.Data.PerpetualContract[a]
			if !contractDetail.EnableTrade {
				return "", currency.ErrPairNotEnabled
			}
			break
		}
	}
	if contractDetail == nil {
		return "", fmt.Errorf("%w, contract: %s", errContractNotFound, arg.Symbol.String())
	}
	syntheticAssetID, ok := big.NewInt(0).SetString(contractDetail.StarkExSyntheticAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, syntheticAssetId: %s", errInvalidAssetID, contractDetail.StarkExSyntheticAssetID)
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	takerFeeRate := -1.
	for k := range ap.UserAccountDetail.Accounts {
		if ap.UserAccountDetail.Accounts[k].Token == contractDetail.SettleCurrencyID {
			takerFeeRate = ap.UserAccountDetail.Accounts[k].TakerFeeRate.Float64()
			break
		}
	}
	if takerFeeRate == -1. {
		return "", fmt.Errorf("%w, account with a settlement "+contractDetail.SettleCurrencyID+" is missing", errLimitFeeRequired)
	}
	arg.LimitFee = takerFeeRate * arg.Size * arg.Price
	var collateralAsset *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == contractDetail.SettleCurrencyID {
			collateralAsset = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if collateralAsset == nil {
		return "", errSettlementCurrencyInfoNotFound
	}

	collateralAssetID, ok := big.NewInt(0).SetString(collateralAsset.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, collateralAsset.StarkExAssetID)
	}

	positionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 10)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	syntheticResolution, err := decimal.NewFromString(contractDetail.StarkExResolution)
	if err != nil {
		return "", err
	}
	collateralResolution, err := decimal.NewFromString(collateralAsset.StarkExResolution)
	if err != nil {
		return "", err
	}
	arg.Side = strings.ToUpper(arg.Side)
	isBuy := arg.Side == "BUY"
	var quantumsAmountCollateral decimal.Decimal
	if isBuy {
		quantumsAmountCollateral = size.Mul(price).Mul(collateralResolution).RoundUp(0)
	} else {
		quantumsAmountCollateral = size.Mul(price).Mul(collateralResolution).RoundDown(0)
	}
	quantumsAmountSynthetic := size.Mul(syntheticResolution)
	limitFeeRounded := decimal.NewFromFloat(takerFeeRate)
	if arg.ClientOrderID == "" {
		arg.ClientOrderID = strings.TrimPrefix(randomClientID(), "0")
	}
	expEpoch := int64(float64(arg.ExpirationTime) / float64(3600*1000))
	if arg.ExpirationTime == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.ExpirationTime = expEpoch * 3600 * 1000
	}
	newArg := &starkex.CreateOrderWithFeeParams{
		OrderType:               "LIMIT_ORDER_WITH_FEES",
		AssetIDSynthetic:        syntheticAssetID,
		AssetIDCollateral:       collateralAssetID,
		AssetIDFee:              collateralAssetID,
		QuantumAmountSynthetic:  quantumsAmountSynthetic.BigInt(),
		QuantumAmountCollateral: quantumsAmountCollateral.BigInt(),
		QuantumAmountFee:        limitFeeRounded.Mul(quantumsAmountCollateral).RoundUp(0).BigInt(),
		IsBuyingSynthetic:       isBuy,
		PositionID:              positionID,
		Nonce:                   nonceFromClientID(arg.ClientOrderID),
		ExpirationEpochHours:    big.NewInt(expEpoch),
	}
	r, s, err := ap.StarkConfig.Sign(newArg, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

func appendSignatures(r, s *big.Int) string {
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	for i := len(rBytes); i < 32; i++ {
		rBytes = append([]byte{byte(0)}, rBytes...)
	}
	for i := len(sBytes); i < 32; i++ {
		sBytes = append([]byte{byte(0)}, sBytes...)
	}
	return hex.EncodeToString(append(rBytes, sBytes...))
}

// ProcessWithdrawalToAddressSignatureV3 processes withdrawal to specified ethereum address request parameter and generates a starkEx signature
func (ap *Apexpro) ProcessWithdrawalToAddressSignatureV3(ctx context.Context, arg *AssetWithdrawalParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	var currencyInfo *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == arg.L1TargetTokenID.String() {
			currencyInfo = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if currencyInfo == nil {
		return "", errSettlementCurrencyInfoNotFound
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	if arg.ZKAccountID == "" {
		arg.ZKAccountID = ap.UserAccountDetail.ID
	}
	collateralAssetID, ok := big.NewInt(0).SetString(currencyInfo.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, currencyInfo.StarkExAssetID)
	}
	if arg.EthereumAddress == "" {
		return "", errEthereumAddressMissing
	}
	ethereumAddress, ok := big.NewInt(0).SetString(arg.EthereumAddress, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidEthereumAddress, arg.EthereumAddress)
	}
	positionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 0)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	resolution, err := decimal.NewFromString(currencyInfo.StarkExResolution)
	if err != nil {
		return "", err
	}
	amount := decimal.NewFromFloat(arg.Amount)

	r, s, err := ap.StarkConfig.Sign(&starkex.WithdrawalToAddressParams{
		AssetIDCollateral:    collateralAssetID,
		EthAddress:           ethereumAddress,
		PositionID:           positionID,
		Amount:               amount.Mul(resolution).BigInt(),
		Nonce:                nonceFromClientID(arg.Nonce),
		ExpirationEpochHours: big.NewInt(int64(math.Ceil(float64(arg.Timestamp.Unix())/float64(3600))) + orderSignatureExpirationBuffersHours),
	}, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

// ProcessWithdrawalToAddressSignature processes withdrawal to specified ethereum address request parameter and generates a starkEx signature for V1 and V2 api endpoints
func (ap *Apexpro) ProcessWithdrawalToAddressSignature(ctx context.Context, arg *WithdrawalToAddressParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	var currencyInfo *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == arg.Asset.String() {
			currencyInfo = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if currencyInfo == nil {
		return "", errSettlementCurrencyInfoNotFound
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	collateralAssetID, ok := big.NewInt(0).SetString(currencyInfo.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, currencyInfo.StarkExAssetID)
	}
	if arg.EthereumAddress == "" {
		return "", errEthereumAddressMissing
	}
	ethereumAddress, ok := big.NewInt(0).SetString(arg.EthereumAddress, 0)
	if !ok {
		return "", fmt.Errorf("%w, ethereum address: %s", errInvalidEthereumAddress, arg.EthereumAddress)
	}
	positionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 0)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	resolution, err := decimal.NewFromString(currencyInfo.StarkExResolution)
	if err != nil {
		return "", err
	}
	expEpoch := int64(float64(arg.ExpEpoch) / float64(3600*1000))
	if arg.ExpEpoch == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.ExpEpoch = expEpoch * 3600 * 1000
	}
	if arg.ClientOrderID == "" {
		arg.ClientOrderID = strings.TrimPrefix(randomClientID(), "0")
	}
	amount := decimal.NewFromFloat(arg.Amount)
	r, s, err := ap.StarkConfig.Sign(&starkex.WithdrawalToAddressParams{
		AssetIDCollateral:    collateralAssetID,
		EthAddress:           ethereumAddress,
		PositionID:           positionID,
		Amount:               amount.Mul(resolution).BigInt(),
		Nonce:                nonceFromClientID(arg.ClientOrderID),
		ExpirationEpochHours: big.NewInt(expEpoch),
	}, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

// ProcessWithdrawalSignature processes withdrawal request parameter and generates a starkEx signature
func (ap *Apexpro) ProcessWithdrawalSignature(ctx context.Context, arg *WithdrawalParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	var collateralInfo *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == arg.Asset.String() {
			collateralInfo = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if collateralInfo == nil {
		return "", errSettlementCurrencyInfoNotFound
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	collateralAssetID, ok := big.NewInt(0).SetString(collateralInfo.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, collateralInfo.StarkExAssetID)
	}
	positionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 0)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	collateralResolution, err := decimal.NewFromString(collateralInfo.StarkExResolution)
	if err != nil {
		return "", err
	}
	if arg.ClientID == "" {
		arg.ClientID = randomClientID()
	}
	amount := decimal.NewFromFloat(arg.Amount)
	expEpoch := int64(float64(arg.ExpEpoch) / float64(3600*1000))
	if arg.ExpEpoch == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.ExpEpoch = expEpoch * 3600 * 1000
	}
	newArg := &starkex.WithdrawalParams{
		AssetIDCollateral:    collateralAssetID,
		PositionID:           positionID,
		Amount:               amount.Mul(collateralResolution).BigInt(),
		Nonce:                nonceFromClientID(arg.ClientID),
		ExpirationEpochHours: big.NewInt(expEpoch),
	}
	r, s, err := ap.StarkConfig.Sign(newArg, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

// ProcessTransferSignature processes withdrawal request parameter and generates a starkEx signature
func (ap *Apexpro) ProcessTransferSignature(ctx context.Context, arg *FastWithdrawalParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	var currencyInfo *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == arg.Asset.String() {
			currencyInfo = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if currencyInfo == nil {
		return "", errSettlementCurrencyInfoNotFound
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	collateralAssetID, ok := big.NewInt(0).SetString(currencyInfo.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, currencyInfo.StarkExAssetID)
	}
	positionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 0)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	resolution, err := decimal.NewFromString(currencyInfo.StarkExResolution)
	if err != nil {
		return "", err
	}
	if arg.ClientID == "" {
		arg.ClientID = randomClientID()
	}
	expEpoch := int64(float64(arg.Expiration) / float64(3600*1000))
	if arg.Expiration == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.Expiration = expEpoch * 3600 * 1000
	}
	amount := decimal.NewFromFloat(arg.Amount)
	r, s, err := ap.StarkConfig.Sign(&starkex.TransferParams{
		AssetID:              collateralAssetID,
		AssetIDFee:           big.NewInt(0),
		SenderPositionID:     positionID,
		QuantumsAmount:       amount.Mul(resolution).BigInt(),
		Nonce:                nonceFromClientID(arg.ClientID),
		ExpirationEpochHours: big.NewInt(expEpoch),
	}, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

// ProcessConditionalTransfer processes conditional transfer request parameter and generates a starkEx signature
func (ap *Apexpro) ProcessConditionalTransfer(ctx context.Context, arg *FastWithdrawalParams) (string, error) {
	creds, err := ap.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	// check if the all symbols config is loaded, if not load
	if ap.SymbolsConfig == nil {
		ap.SymbolsConfig, err = ap.GetAllSymbolsConfigDataV1(ctx)
		if err != nil {
			return "", err
		}
	}
	var currencyInfo *V1CurrencyConfig
	for c := range ap.SymbolsConfig.Data.Currency {
		if ap.SymbolsConfig.Data.Currency[c].ID == arg.Asset.String() {
			currencyInfo = &ap.SymbolsConfig.Data.Currency[c]
			break
		}
	}
	if currencyInfo == nil {
		return "", errSettlementCurrencyInfoNotFound
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}

	assetID, ok := big.NewInt(0).SetString(currencyInfo.StarkExAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", errInvalidAssetID, currencyInfo.StarkExAssetID)
	}
	senderPositionID, ok := big.NewInt(0).SetString(ap.UserAccountDetail.PositionID, 0)
	if !ok {
		return "", errInvalidPositionIDMissing
	}
	if ap.UserAccountDetail == nil {
		ap.UserAccountDetail, err = ap.GetUserAccountDataV2(ctx)
		if err != nil {
			return "", err
		}
	}
	if ap.UserAccountDetail.StarkKey != creds.L2Key {
		return "", errL2CredentialsMismatch
	}
	resolution, err := decimal.NewFromString(currencyInfo.StarkExResolution)
	if err != nil {
		return "", err
	}
	amount := decimal.NewFromFloat(arg.Amount)
	if arg.ClientID == "" {
		arg.ClientID = randomClientID()
	}
	receiverPositionID, ok := big.NewInt(0).SetString(ap.SymbolsConfig.Data.Global.FastWithdrawAccountID, 0)
	if !ok {
		return "", fmt.Errorf("%w, invalid fast withdrawal position ID", errInvalidPositionIDMissing)
	}
	receiverPublicKey, ok := big.NewInt(0).SetString(ap.SymbolsConfig.Data.Global.FeeAccountL2Key, 0)
	if !ok {
		return "", fmt.Errorf("%w, invalid fast withdrawal L2 key", errL2KeyMissing)
	}
	fastWithdrawFactRegisterAddress, ok := big.NewInt(0).SetString(ap.SymbolsConfig.Data.Global.FastWithdrawFactRegisterAddress, 0)
	if !ok {
		return "", fmt.Errorf("%w, invalid fast withdraw fact register address", errL2KeyMissing)
	}
	var token *TokenInfo
	for k := range ap.SymbolsConfig.Data.MultiChain.Chains {
		if ap.SymbolsConfig.Data.MultiChain.Chains[k].ChainID == int64(ap.NetworkID) {
			for t := range ap.SymbolsConfig.Data.MultiChain.Chains[k].Tokens {
				if ap.SymbolsConfig.Data.MultiChain.Chains[k].Tokens[t].Token == arg.Asset.Upper().String() {
					token = &ap.SymbolsConfig.Data.MultiChain.Chains[k].Tokens[t]
				}
			}
		}
	}
	if token == nil {
		return "", errTokenDetailIsMissing
	}
	fact, err := GetTransferErc20Fact(int(token.Decimals),
		arg.ERC20Address,
		strconv.FormatFloat(arg.Amount, 'f', -1, 64), token.TokenAddress,
		"0x"+nonceFromClientID(arg.ClientID).Text(16))
	if err != nil {
		return "", err
	}
	expEpoch := int64(float64(arg.Expiration) / float64(3600*1000))
	if arg.Expiration == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.Expiration = expEpoch * 3600 * 1000
	}
	senderPublicKey, ok := big.NewInt(0).SetString(ap.UserAccountDetail.StarkKey, 0)
	if !ok {
		return "", errL2KeyMissing
	}
	r, s, err := ap.StarkConfig.Sign(&starkex.ConditionalTransferParams{
		AssetID:            assetID,
		AssetIDFee:         big.NewInt(0),
		MaxAmountFee:       big.NewInt(0),
		SenderPositionID:   senderPositionID,
		SenderPublicKey:    senderPublicKey,
		ReceiverPositionID: receiverPositionID,
		ReceiverPublicKey:  receiverPublicKey,
		Condition:          FactToCondition(fastWithdrawFactRegisterAddress.Text(16), fact),
		QuantumsAmount:     amount.Mul(resolution).BigInt(),
		Nonce:              nonceFromClientID(arg.ClientID),
		ExpTimestampHrs:    big.NewInt(expEpoch),
	}, creds.L2Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return appendSignatures(r, s), nil
}

// FactToCondition Generate the condition, signed as part of a conditional transfer.
func FactToCondition(factRegistryAddress, fact string) *big.Int {
	data := strings.TrimPrefix(factRegistryAddress, "0x") + fact
	hexBytes, _ := hex.DecodeString(data)
	hash := crypto.Keccak256Hash(hexBytes)
	fst := hash.Big()
	fst.And(fst, BitMask250)
	return fst
}

// GetTransferErc20Fact get erc20 fact
// tokenDecimals is COLLATERAL_TOKEN_DECIMALS
func GetTransferErc20Fact(tokenDecimals int, recipient, humanAmount, tokenAddress, salt string) (string, error) {
	amount, err := decimal.NewFromString(humanAmount)
	if err != nil {
		return "", err
	}
	saltInt, ok := big.NewInt(0).SetString(salt, 0) // with prefix: 0x
	if !ok {
		return "", fmt.Errorf("invalid salt: %v,can not parse to big.Int", salt)
	}
	tokenAmount := amount.Mul(decimal.New(10, int32(tokenDecimals-1)))
	fact, err := solsha3.SoliditySHA3(
		// types
		[]string{"address", "uint256", "address", "uint256"},
		// values
		[]interface{}{recipient, tokenAmount.String(), tokenAddress, saltInt.String()},
	)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(fact), nil
}
