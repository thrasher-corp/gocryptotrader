package dydx

import (
	"math/big"

	"github.com/shopspring/decimal"
)

const (
	ORDER_PREFIX                = 3
	CONDITIONAL_TRANSFER_PREFIX = 5
	WITHDRAWAL_PREFIX           = 6

	ORDER_PADDING_BITS                = 17
	WITHDRAWAL_PADDING_BITS           = 49
	CONDITIONAL_TRANSFER_PADDING_BITS = 81

	CONDITIONAL_TRANSFER_FEE_ASSET_ID   = 0
	CONDITIONAL_TRANSFER_MAX_AMOUNT_FEE = 0

	COLLATERAL_TOKEN_DECIMALS = 6

	NONCE_UPPER_BOUND_EXCLUSIVE = 1 << 32 // 1 << ORDER_FIELD_BIT_LENGTHS['nonce']

	NETWORK_ID_MAINNET = 1
	NETWORK_ID_ROPSTEN = 3

	ONE_HOUR_IN_SECONDS                     = 60 * 60
	ORDER_SIGNATURE_EXPIRATION_BUFFER_HOURS = 24 * 7 // Seven days.

	COLLATERAL_ASSET = "USDC"

	ASSET_ID_MAINNET = "0x02893294412a4c8f915f75892b395ebbf6859ec246ec365c3b1f56f47c3a0a5d"
	ASSET_ID_ROPSTEN = "0x02c04d8b650f44092278a7cb1e1028c82025dff622db96c934b611b84cc8de5a"
)

var (
	mainNet, _     = big.NewInt(0).SetString(ASSET_ID_MAINNET, 0) // with prefix: 0x
	ropstenNet, _  = big.NewInt(0).SetString(ASSET_ID_ROPSTEN, 0) // with prefix: 0x
	resolutionUsdc = decimal.NewFromInt(ASSET_RESOLUTION[COLLATERAL_ASSET])
)

var COLLATERAL_ASSET_ID_BY_NETWORK_ID = map[int]*big.Int{
	NETWORK_ID_MAINNET: mainNet,    // MAINNET
	NETWORK_ID_ROPSTEN: ropstenNet, // ROPSTEN
}

var FACT_REGISTRY_CONTRACT = map[int]string{
	NETWORK_ID_MAINNET: "0xBE9a129909EbCb954bC065536D2bfAfBd170d27A",
	NETWORK_ID_ROPSTEN: "0x8Fb814935f7E63DEB304B500180e19dF5167B50e",
}

var TOKEN_CONTRACTS = map[string]map[int]string{
	COLLATERAL_ASSET: {
		NETWORK_ID_MAINNET: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		NETWORK_ID_ROPSTEN: "0x8707A5bf4C2842d46B31A405Ba41b858C0F876c4",
	},
}

// N_ELEMENT_BITS_ECDSA math.floor(math.log(FIELD_PRIME, 2))
var N_ELEMENT_BITS_ECDSA = big.NewInt(251)

// BIT_MASK_250 (2 ** 250) - 1
var BIT_MASK_250 = big.NewInt(0).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(250), nil), big.NewInt(1))

var ASSET_RESOLUTION = map[string]int64{
	"USDC":    1e6,
	"BTC":     1e10,
	"ETH":     1e9,
	"LINK":    1e7,
	"AAVE":    1e8,
	"UNI":     1e7,
	"SUSHI":   1e7,
	"SOL":     1e7,
	"YFI":     1e10,
	"ONEINCH": 1e7,
	"AVAX":    1e7,
	"SNX":     1e7,
	"CRV":     1e6,
	"UMA":     1e7,
	"DOT":     1e7,
	"DOGE":    1e5,
	"MATIC":   1e6,
	"MKR":     1e9,
	"FIL":     1e7,
	"ADA":     1e6,
	"ATOM":    1e7,
	"COMP":    1e8,
	"BCH":     1e8,
	"LTC":     1e8,
	"EOS":     1e6,
	"ALGO":    1e6,
	"ZRX":     1e6,
	"XMR":     1e8,
	"ZEC":     1e8,
}
