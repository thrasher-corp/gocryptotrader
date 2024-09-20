package dydx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/starkex"
	"golang.org/x/exp/rand"
)

const ORDER_SIGNATURE_EXPIRATION_BUFFER_HOURS = 24 * 7 // Seven days.
const NONCE_UPPER_BOUND_EXCLUSIVE = 1 << 32            // 1 << ORDER_FIELD_BIT_LENGTHS['nonce']

// ProcessOrderSignature processes order request parameter and generates a starkEx signature
func (dy *DYDX) ProcessOrderSignature(ctx context.Context, arg *CreateOrderRequestParams) (string, error) {
	creds, err := dy.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	if creds.L2Secret == "" {
		return "", starkex.ErrInvalidPrivateKey
	}
	price := decimal.NewFromFloat(arg.Price)
	size := decimal.NewFromFloat(arg.Size)

	// check if the all symbols config is loaded, if not load
	if dy.SymbolsConfig == nil {
		dy.SymbolsConfig, err = dy.GetMarkets(ctx, "")
		if err != nil {
			return "", err
		}
	}
	var contractDetail MarketDataDetail
	for a := range dy.SymbolsConfig.Markets {
		if dy.SymbolsConfig.Markets[a].Market == arg.Market {
			var ok bool
			contractDetail, ok = dy.SymbolsConfig.Markets[a]
			if !ok || contractDetail.Status == "OFFLINE" {
				return "", currency.ErrPairNotEnabled
			}
			break
		}
	}
	if contractDetail == nil {
		return "", fmt.Errorf("%w, contract: %s", starkex.ErrContractNotFound, arg.Market)
	}
	syntheticAssetID, ok := big.NewInt(0).SetString(contractDetail.SyntheticAssetID, 0)
	if !ok {
		return "", fmt.Errorf("%w, syntheticAssetId: %s", starkex.ErrInvalidAssetID, contractDetail.StarkExSyntheticAssetID)
	}
	if dy.UserAccountDetail == nil {
		accountDetail, err := dy.GetAccount(ctx, "")
		if err != nil {
			return "", err
		}
		dy.UserAccountDetail = accountDetail.Account
	}
	takerFeeRate := 0.003
	// if takerFeeRate == -1. {
	// 	return "", fmt.Errorf("%w, account with a settlement "+contractDetail.SettleCurrencyID+" is missing", errLimitFeeRequired)
	// }
	arg.LimitFee = takerFeeRate * arg.Size * arg.Price
	// var collateralAsset *V1CurrencyConfig
	// for c := range dy.SymbolsConfig.Data.Currency {
	// 	if dy.SymbolsConfig.Data.Currency[c].ID == contractDetail.SettleCurrencyID {
	// 		collateralAsset = &dy.SymbolsConfig.Data.Currency[c]
	// 		break
	// 	}
	// }
	// if collateralAsset == nil {
	// 	return "", starkex.ErrSettlementCurrencyInfoNotFound
	// }

	collateralAssetID, ok := big.NewInt(0).SetString(dy.UserAccountDetail.StarkKey, 0)
	if !ok {
		return "", fmt.Errorf("%w, assetId: %s", starkex.ErrInvalidAssetID, dy.UserAccountDetail.StarkKey)
	}

	positionID, ok := big.NewInt(0).SetString(dy.UserAccountDetail.PositionID, 10)
	if !ok {
		return "", starkex.ErrInvalidPositionIDMissing
	}
	syntheticResolution, err := decimal.NewFromString(contractDetail.AssetResolution)
	if err != nil {
		return "", err
	}
	// collateralResolution, err := decimal.NewFromString(dy.UserAccountDetail..StarkExResolution)
	if err != nil {
		return "", err
	}
	arg.Side = strings.ToUpper(arg.Side)
	isBuy := arg.Side == "BUY"
	var quantumsAmountCollateral decimal.Decimal
	// if isBuy {
	// 	quantumsAmountCollateral = size.Mul(price).Mul(collateralResolution).RoundUp(0)
	// } else {
	// 	quantumsAmountCollateral = size.Mul(price).Mul(collateralResolution).RoundDown(0)
	// }
	quantumsAmountSynthetic := size.Mul(syntheticResolution)
	limitFeeRounded := decimal.NewFromFloat(takerFeeRate)
	if arg.ClientID == "" {
		arg.ClientID = strings.TrimPrefix(randomClientID(), "0")
	}
	// expEpoch := int64(float64(arg.ExpirationTime) / float64(3600*1000))
	// if arg.ExpirationTime == 0 {
	// 	expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
	// 	arg.ExpirationTime = expEpoch * 3600 * 1000
	// }
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
		// Nonce:                   nonceFromClientID(arg.ClientOrderID), //nonceVal,
		// ExpirationEpochHours:    big.NewInt(expEpoch),
	}
	r, s, err := dy.StarkConfig.Sign(newArg, creds.Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return starkex.AppendSignatures(r, s), nil
}

func randomClientID() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	return strconv.FormatFloat(rand.Float64(), 'f', -1, 64)[2:]
}

func nonceFromClientID(clientID string) *big.Int {
	hasher := sha256.New()
	hasher.Write([]byte(clientID))
	hashBytes := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)
	nonce, _ := strconv.ParseUint(hashHex[0:8], 16, 64)
	return big.NewInt(int64(nonce))
}
