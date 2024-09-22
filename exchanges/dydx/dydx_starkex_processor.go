package dydx

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/internal/utils/starkex"
	"golang.org/x/exp/rand"
)

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
		return "", fmt.Errorf("%w, syntheticAssetId: %s", starkex.ErrInvalidAssetID, contractDetail.SyntheticAssetID)
	}
	if dy.UserAccountDetail == nil {
		accountDetail, err := dy.GetAccount(ctx, "")
		if err != nil {
			return "", err
		}
		dy.UserAccountDetail = accountDetail.Account
	}

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
	arg.Side = strings.ToUpper(arg.Side)
	isBuy := arg.Side == "BUY"
	var quantumsAmountCollateral decimal.Decimal
	if isBuy {
		quantumsAmountCollateral = size.Mul(price).Mul(resolutionUsdc).RoundUp(0)
	} else {
		quantumsAmountCollateral = size.Mul(price).Mul(resolutionUsdc).RoundDown(0)
	}
	quantumsAmountSynthetic := size.Mul(syntheticResolution)
	limitFeeRounded := decimal.NewFromFloat(arg.LimitFee)
	if arg.ClientID == "" {
		arg.ClientID = strings.TrimPrefix(randomClientID(), "0")
	}
	expEpoch := int64(float64(arg.Expiration) / float64(3600*1000))
	if arg.Expiration == 0 {
		expEpoch = int64(math.Ceil(float64(time.Now().Add(time.Hour*24*28).UnixMilli()) / float64(3600*1000)))
		arg.Expiration = expEpoch * 3600 * 1000
	}
	r, s, err := dy.StarkConfig.Sign(&starkex.CreateOrderWithFeeParams{
		OrderType:               "LIMIT_ORDER_WITH_FEES",
		AssetIDSynthetic:        syntheticAssetID,
		AssetIDCollateral:       collateralAssetID,
		AssetIDFee:              collateralAssetID,
		QuantumAmountSynthetic:  quantumsAmountSynthetic.BigInt(),
		QuantumAmountCollateral: quantumsAmountCollateral.BigInt(),
		QuantumAmountFee:        limitFeeRounded.Mul(quantumsAmountCollateral).RoundUp(0).BigInt(),
		IsBuyingSynthetic:       isBuy,
		PositionID:              positionID,
		Nonce:                   NonceByClientId(arg.ClientID),
		ExpirationEpochHours:    big.NewInt(expEpoch),
	}, creds.Secret, creds.L2Key, creds.L2KeyYCoordinate)
	if err != nil {
		return "", err
	}
	return starkex.AppendSignatures(r, s), nil
}

func randomClientID() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	return strconv.FormatFloat(rand.Float64(), 'f', -1, 64)[2:]
}

// NonceByClientId generate nonce by clientId
func NonceByClientId(clientId string) *big.Int {
	h := sha256.New()
	h.Write([]byte(clientId))

	a := new(big.Int)
	a.SetBytes(h.Sum(nil))
	res := a.Mod(a, big.NewInt(NONCE_UPPER_BOUND_EXCLUSIVE))
	return res
}
