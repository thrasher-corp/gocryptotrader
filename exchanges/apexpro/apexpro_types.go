package apexpro

import (
	"math/big"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var one = big.NewInt(1)

// BitMask250 (2 ** 250) - 1
var BitMask250 = big.NewInt(0).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(250), nil), one)

// AllSymbolsConfigs represents all symbols configurations.
type AllSymbolsConfigs struct {
	SpotConfig struct {
		Assets []struct {
			TokenID       string       `json:"tokenId"`
			Token         string       `json:"token"`
			DisplayName   string       `json:"displayName"`
			Decimals      int64        `json:"decimals"`
			ShowStep      string       `json:"showStep"`
			IconURL       string       `json:"iconUrl"`
			L2WithdrawFee types.Number `json:"l2WithdrawFee"`
		} `json:"assets"`
		Global struct {
			DefaultRegisterTransferToken      string `json:"defaultRegisterTransferToken"`
			DefaultRegisterTransferTokenID    string `json:"defaultRegisterTransferTokenId"`
			DefaultRegisterSubAccountID       string `json:"defaultRegisterSubAccountId"`
			DefaultChangePubKeyZklinkChainID  string `json:"defaultChangePubKeyZklinkChainId"`
			DefaultChangePubKeyFeeTokenID     string `json:"defaultChangePubKeyFeeTokenId"`
			DefaultChangePubKeyFeeToken       string `json:"defaultChangePubKeyFeeToken"`
			DefaultChangePubKeyFee            string `json:"defaultChangePubKeyFee"`
			RegisterTransferLpAccountID       string `json:"registerTransferLpAccountId"`
			RegisterTransferLpSubAccount      string `json:"registerTransferLpSubAccount"`
			RegisterTransferLpSubAccountL2Key string `json:"registerTransferLpSubAccountL2Key"`
			PerpLpAccountID                   string `json:"perpLpAccountId"`
			PerpLpSubAccount                  string `json:"perpLpSubAccount"`
			PerpLpSubAccountL2Key             string `json:"perpLpSubAccountL2Key"`
			ContractAssetPoolAccountID        string `json:"contractAssetPoolAccountId"`
			ContractAssetPoolZkAccountID      string `json:"contractAssetPoolZkAccountId"`
			ContractAssetPoolSubAccount       string `json:"contractAssetPoolSubAccount"`
			ContractAssetPoolL2Key            string `json:"contractAssetPoolL2Key"`
			ContractAssetPoolEthAddress       string `json:"contractAssetPoolEthAddress"`
		} `json:"global"`
		Spot       []any `json:"spot"`
		MultiChain struct {
			Chains []struct {
				Chain              string `json:"chain"`
				ChainID            string `json:"chainId"`
				ChainType          string `json:"chainType"`
				L1ChainID          string `json:"l1ChainId"`
				ChainIconURL       string `json:"chainIconUrl"`
				ContractAddress    string `json:"contractAddress"`
				StopDeposit        bool   `json:"stopDeposit"`
				FeeLess            bool   `json:"feeLess"`
				GasLess            bool   `json:"gasLess"`
				GasToken           string `json:"gasToken"`
				DynamicFee         bool   `json:"dynamicFee"`
				FeeGasLimit        int64  `json:"feeGasLimit"`
				BlockTimeSeconds   int64  `json:"blockTimeSeconds"`
				RPCURL             string `json:"rpcUrl"`
				WebRPCURL          string `json:"webRpcUrl"`
				WebTxURL           string `json:"webTxUrl"`
				TxConfirm          int64  `json:"txConfirm"`
				WithdrawGasFeeLess bool   `json:"withdrawGasFeeLess"`
				Tokens             []struct {
					Decimals          int64  `json:"decimals"`
					IconURL           string `json:"iconUrl"`
					Token             string `json:"token"`
					TokenAddress      string `json:"tokenAddress"`
					PullOff           bool   `json:"pullOff"`
					WithdrawEnable    bool   `json:"withdrawEnable"`
					Slippage          string `json:"slippage"`
					IsDefaultToken    bool   `json:"isDefaultToken"`
					DisplayToken      string `json:"displayToken"`
					NeedResetApproval bool   `json:"needResetApproval"`
					MinFee            string `json:"minFee"`
					MaxFee            string `json:"maxFee"`
					FeeRate           string `json:"feeRate"`
				} `json:"tokens"`
			} `json:"chains"`
			MaxWithdraw string       `json:"maxWithdraw"`
			MinDeposit  types.Number `json:"minDeposit"`
			MinWithdraw types.Number `json:"minWithdraw"`
		} `json:"multiChain"`
	} `json:"spotConfig"`
	ContractConfig struct {
		Assets []AssetInfo `json:"assets"`
		Tokens []struct {
			Token    string       `json:"token"`
			StepSize types.Number `json:"stepSize"`
			IconURL  string       `json:"iconUrl"`
		} `json:"tokens"`
		Global struct {
			FeeAccountID             string `json:"feeAccountId"`
			FeeAccountL2Key          string `json:"feeAccountL2Key"`
			ContractAssetLpAccountID string `json:"contractAssetLpAccountId"`
			ContractAssetLpL2Key     string `json:"contractAssetLpL2Key"`
			OperationAccountID       string `json:"operationAccountId"`
			OperationL2Key           string `json:"operationL2Key"`
			ExperienceMoneyAccountID string `json:"experienceMoneyAccountId"`
			ExperienceMoneyL2Key     string `json:"experienceMoneyL2Key"`
			AgentAccountID           string `json:"agentAccountId"`
			AgentL2Key               string `json:"agentL2Key"`
			FinxFeeAccountID         string `json:"finxFeeAccountId"`
			FinxFeeL2Key             string `json:"finxFeeL2Key"`
			NegativeRateAccountID    string `json:"negativeRateAccountId"`
			NegativeRateL2Key        string `json:"negativeRateL2Key"`
			BrokerAccountID          string `json:"brokerAccountId"`
			BrokerL2Key              string `json:"brokerL2Key"`
		} `json:"global"`
		PerpetualContract []struct {
			BaselinePositionValue            string       `json:"baselinePositionValue"`
			CrossID                          int64        `json:"crossId"`
			CrossSymbolID                    int64        `json:"crossSymbolId"`
			CrossSymbolName                  string       `json:"crossSymbolName"`
			DigitMerge                       string       `json:"digitMerge"`
			DisplayMaxLeverage               string       `json:"displayMaxLeverage"`
			DisplayMinLeverage               string       `json:"displayMinLeverage"`
			EnableDisplay                    bool         `json:"enableDisplay"`
			EnableOpenPosition               bool         `json:"enableOpenPosition"`
			EnableTrade                      bool         `json:"enableTrade"`
			FundingImpactMarginNotional      string       `json:"fundingImpactMarginNotional"`
			FundingInterestRate              types.Number `json:"fundingInterestRate"`
			IncrementalInitialMarginRate     types.Number `json:"incrementalInitialMarginRate"`
			IncrementalMaintenanceMarginRate types.Number `json:"incrementalMaintenanceMarginRate"`
			IncrementalPositionValue         types.Number `json:"incrementalPositionValue"`
			InitialMarginRate                types.Number `json:"initialMarginRate"`
			MaintenanceMarginRate            types.Number `json:"maintenanceMarginRate"`
			MaxOrderSize                     types.Number `json:"maxOrderSize"`
			MaxPositionSize                  types.Number `json:"maxPositionSize"`
			MinOrderSize                     types.Number `json:"minOrderSize"`
			MaxMarketPriceRange              types.Number `json:"maxMarketPriceRange"`
			SettleAssetID                    string       `json:"settleAssetId"` // Collateral asset ID.
			BaseTokenID                      string       `json:"baseTokenId"`
			StepSize                         types.Number `json:"stepSize"`
			Symbol                           string       `json:"symbol"`
			SymbolDisplayName                string       `json:"symbolDisplayName"`
			TickSize                         types.Number `json:"tickSize"`
			MaxMaintenanceMarginRate         types.Number `json:"maxMaintenanceMarginRate"`
			MaxPositionValue                 types.Number `json:"maxPositionValue"`
			TagIconURL                       string       `json:"tagIconUrl"`
			Tag                              string       `json:"tag"`
			RiskTip                          bool         `json:"riskTip"`
			DefaultInitialMarginRate         types.Number `json:"defaultInitialMarginRate"`
			KlineStartTime                   types.Time   `json:"klineStartTime"`
			MaxMarketSizeBuffer              string       `json:"maxMarketSizeBuffer"`
			EnableFundingSettlement          bool         `json:"enableFundingSettlement"`
			IndexPriceDecimals               int64        `json:"indexPriceDecimals"`
			IndexPriceVarRate                types.Number `json:"indexPriceVarRate"`
			OpenPositionOiLimitRate          types.Number `json:"openPositionOiLimitRate"`
			FundingMaxRate                   types.Number `json:"fundingMaxRate"`
			FundingMinRate                   types.Number `json:"fundingMinRate"`
			FundingMaxValue                  types.Number `json:"fundingMaxValue"`
			EnableFundingMxValue             bool         `json:"enableFundingMxValue"`
			L2PairID                         string       `json:"l2PairId"`
			SettleTimeStamp                  types.Time   `json:"settleTimeStamp"`
			IsPrelaunch                      bool         `json:"isPrelaunch"`
			RiskLimitConfig                  struct {
				PositionSteps []string `json:"positionSteps"`
				ImrSteps      []string `json:"imrSteps"`
				MmrSteps      []string `json:"mmrSteps"`
			} `json:"riskLimitConfig"`
		} `json:"perpetualContract"`
		PrelaunchContract []struct {
			BaselinePositionValue            string       `json:"baselinePositionValue"`
			CrossID                          int64        `json:"crossId"`
			CrossSymbolID                    int64        `json:"crossSymbolId"`
			CrossSymbolName                  string       `json:"crossSymbolName"`
			DigitMerge                       string       `json:"digitMerge"`
			DisplayMaxLeverage               types.Number `json:"displayMaxLeverage"`
			DisplayMinLeverage               types.Number `json:"displayMinLeverage"`
			EnableDisplay                    bool         `json:"enableDisplay"`
			EnableOpenPosition               bool         `json:"enableOpenPosition"`
			EnableTrade                      bool         `json:"enableTrade"`
			FundingImpactMarginNotional      types.Number `json:"fundingImpactMarginNotional"`
			FundingInterestRate              types.Number `json:"fundingInterestRate"`
			IncrementalInitialMarginRate     types.Number `json:"incrementalInitialMarginRate"`
			IncrementalMaintenanceMarginRate types.Number `json:"incrementalMaintenanceMarginRate"`
			IncrementalPositionValue         types.Number `json:"incrementalPositionValue"`
			InitialMarginRate                types.Number `json:"initialMarginRate"`
			MaintenanceMarginRate            types.Number `json:"maintenanceMarginRate"`
			MaxOrderSize                     types.Number `json:"maxOrderSize"`
			MaxPositionSize                  types.Number `json:"maxPositionSize"`
			MinOrderSize                     types.Number `json:"minOrderSize"`
			MaxMarketPriceRange              types.Number `json:"maxMarketPriceRange"`
			SettleAssetID                    string       `json:"settleAssetId"`
			BaseTokenID                      string       `json:"baseTokenId"`
			StepSize                         types.Number `json:"stepSize"`
			Symbol                           string       `json:"symbol"`
			SymbolDisplayName                string       `json:"symbolDisplayName"`
			TickSize                         types.Number `json:"tickSize"`
			MaxMaintenanceMarginRate         types.Number `json:"maxMaintenanceMarginRate"`
			MaxPositionValue                 types.Number `json:"maxPositionValue"`
			TagIconURL                       string       `json:"tagIconUrl"`
			Tag                              string       `json:"tag"`
			RiskTip                          bool         `json:"riskTip"`
			DefaultLeverage                  types.Number `json:"defaultLeverage"`
			KlineStartTime                   types.Time   `json:"klineStartTime"`
			MaxMarketSizeBuffer              string       `json:"maxMarketSizeBuffer"`
			EnableFundingSettlement          bool         `json:"enableFundingSettlement"`
			IndexPriceDecimals               float64      `json:"indexPriceDecimals"`
			IndexPriceVarRate                types.Number `json:"indexPriceVarRate"`
			OpenPositionOiLimitRate          types.Number `json:"openPositionOiLimitRate"`
			FundingMaxRate                   types.Number `json:"fundingMaxRate"`
			FundingMinRate                   types.Number `json:"fundingMinRate"`
			FundingMaxValue                  types.Number `json:"fundingMaxValue"`
			EnableFundingMxValue             bool         `json:"enableFundingMxValue"`
			L2PairID                         string       `json:"l2PairId"`
			SettleTimeStamp                  types.Time   `json:"settleTimeStamp"`
			IsPrelaunch                      bool         `json:"isPrelaunch"`
			RiskLimitConfig                  struct {
				PositionSteps any `json:"positionSteps"`
				ImrSteps      any `json:"imrSteps"`
				MmrSteps      any `json:"mmrSteps"`
			} `json:"riskLimitConfig"`
		} `json:"prelaunchContract"`
		MaxMarketBalanceBuffer string `json:"maxMarketBalanceBuffer"`
	} `json:"contractConfig"`
}

// AssetInfo represents an asset detail information.
type AssetInfo struct {
	TokenID       string       `json:"tokenId"`
	Token         string       `json:"token"`
	DisplayName   string       `json:"displayName"`
	Decimals      types.Number `json:"decimals"`
	ShowStep      string       `json:"showStep"`
	IconURL       string       `json:"iconUrl"`
	L2WithdrawFee types.Number `json:"l2WithdrawFee"`
}

// NewTradingData represents a new trading data detail.
type NewTradingData struct {
	Side      string       `json:"S"`
	Volume    types.Number `json:"v"`
	Price     types.Number `json:"p"`
	Symbol    string       `json:"s"`
	TradeTime types.Time   `json:"T"`
}

// MarketDepthV3 represents a market depth information.
type MarketDepthV3 struct {
	Asks       [][2]types.Number `json:"a"` // Sell
	Bids       [][2]types.Number `json:"b"` // Buy
	Symbol     string            `json:"s"`
	UpdateTime types.Time        `json:"u"`
}

// CandlestickData represents a candlestick chart data.
type CandlestickData struct {
	Start    types.Time   `json:"start"`
	Symbol   string       `json:"symbol"`
	Interval string       `json:"interval"`
	Low      types.Number `json:"low"`
	High     types.Number `json:"high"`
	Open     types.Number `json:"open"`
	Close    types.Number `json:"close"`
	Volume   types.Number `json:"volume"`
	Turnover string       `json:"turnover"`
}

// TickerData represents a price ticker data.
type TickerData struct {
	Symbol               string       `json:"symbol"`
	Price24HPcnt         types.Number `json:"price24hPcnt"`
	LastPrice            types.Number `json:"lastPrice"`
	HighPrice24H         types.Number `json:"highPrice24h"`
	LowPrice24H          types.Number `json:"lowPrice24h"`
	MarkPrice            types.Number `json:"markPrice"`
	IndexPrice           types.Number `json:"indexPrice"`
	OpenInterest         types.Number `json:"openInterest"`
	Turnover24H          types.Number `json:"turnover24h"`
	Volume24H            types.Number `json:"volume24h"`
	FundingRate          types.Number `json:"fundingRate"`
	PredictedFundingRate types.Number `json:"predictedFundingRate"`
	NextFundingTime      types.Time   `json:"nextFundingTime"`
	TradeCount           types.Number `json:"tradeCount"`
}

// FundingRateHistory represents a funding rate history response.
type FundingRateHistory struct {
	HistoryFunds []struct {
		Symbol           string       `json:"symbol"`
		Rate             types.Number `json:"rate"`
		Price            types.Number `json:"price"`
		FundingTime      types.Time   `json:"fundingTime"`
		FundingTimestamp types.Time   `json:"fundingTimestamp"`
	} `json:"historyFunds"`
	TotalSize int64 `json:"totalSize"`
}

// CurrencyInfo represents a currency detail.
type CurrencyInfo struct {
	ID                string       `json:"id"` // Settlement Currency ID.
	StarkExAssetID    string       `json:"starkExAssetId"`
	StarkExResolution string       `json:"starkExResolution"`
	StepSize          types.Number `json:"stepSize"`
	ShowStep          string       `json:"showStep"`
	IconURL           string       `json:"iconUrl"`
}

// V2ConfigData v2 assets and symbols configuration response.
type V2ConfigData struct {
	Data struct {
		USDCConfig struct {
			Currency []CurrencyInfo `json:"currency"`
			Global   struct {
				FeeAccountID                    string `json:"feeAccountId"`
				FeeAccountL2Key                 string `json:"feeAccountL2Key"`
				StarkExCollateralCurrencyID     string `json:"starkExCollateralCurrencyId"`
				StarkExFundingValidityPeriod    int    `json:"starkExFundingValidityPeriod"`
				StarkExMaxFundingRate           string `json:"starkExMaxFundingRate"`
				StarkExOrdersTreeHeight         int    `json:"starkExOrdersTreeHeight"`
				StarkExPositionsTreeHeight      int    `json:"starkExPositionsTreeHeight"`
				StarkExPriceValidityPeriod      int    `json:"starkExPriceValidityPeriod"`
				StarkExContractAddress          string `json:"starkExContractAddress"`
				RegisterEnvID                   int    `json:"registerEnvId"`
				CrossChainAccountID             string `json:"crossChainAccountId"`
				CrossChainL2Key                 string `json:"crossChainL2Key"`
				FastWithdrawAccountID           string `json:"fastWithdrawAccountId"`
				FastWithdrawFactRegisterAddress string `json:"fastWithdrawFactRegisterAddress"`
				FastWithdrawL2Key               string `json:"fastWithdrawL2Key"`
				FastWithdrawMaxAmount           string `json:"fastWithdrawMaxAmount"`
				BybitWithdrawAccountID          string `json:"bybitWithdrawAccountId"`
				BybitWithdrawL2Key              string `json:"bybitWithdrawL2Key"`
				ExperienceMonenyAccountID       string `json:"experienceMonenyAccountId"`
				ExperienceMonenyL2Key           string `json:"experienceMonenyL2Key"`
				ExperienceMoneyAccountID        string `json:"experienceMoneyAccountId"`
				ExperienceMoneyL2Key            string `json:"experienceMoneyL2Key"`
			} `json:"global"`
			PerpetualContract []struct {
				Symbol                           string       `json:"symbol"`
				BaselinePositionValue            string       `json:"baselinePositionValue"`
				CrossID                          int64        `json:"crossId"`
				CrossSymbolID                    int64        `json:"crossSymbolId"`
				CrossSymbolName                  string       `json:"crossSymbolName"`
				DigitMerge                       string       `json:"digitMerge"`
				DisplayMaxLeverage               types.Number `json:"displayMaxLeverage"`
				DisplayMinLeverage               types.Number `json:"displayMinLeverage"`
				EnableDisplay                    bool         `json:"enableDisplay"`
				EnableOpenPosition               bool         `json:"enableOpenPosition"`
				EnableTrade                      bool         `json:"enableTrade"`
				FundingImpactMarginNotional      string       `json:"fundingImpactMarginNotional"`
				FundingInterestRate              types.Number `json:"fundingInterestRate"`
				IncrementalInitialMarginRate     types.Number `json:"incrementalInitialMarginRate"`
				IncrementalMaintenanceMarginRate types.Number `json:"incrementalMaintenanceMarginRate"`
				IncrementalPositionValue         types.Number `json:"incrementalPositionValue"`
				InitialMarginRate                types.Number `json:"initialMarginRate"`
				MaintenanceMarginRate            types.Number `json:"maintenanceMarginRate"`
				MaxOrderSize                     types.Number `json:"maxOrderSize"`
				MaxPositionSize                  types.Number `json:"maxPositionSize"`
				MinOrderSize                     types.Number `json:"minOrderSize"`
				MaxMarketPriceRange              types.Number `json:"maxMarketPriceRange"`
				SettleCurrencyID                 string       `json:"settleCurrencyId"`
				StarkExOraclePriceQuorum         string       `json:"starkExOraclePriceQuorum"`
				StarkExResolution                string       `json:"starkExResolution"`
				StarkExRiskFactor                string       `json:"starkExRiskFactor"`
				StarkExSyntheticAssetID          string       `json:"starkExSyntheticAssetId"`
				StepSize                         types.Number `json:"stepSize"`
				SymbolDisplayName                string       `json:"symbolDisplayName"`
				SymbolDisplayName2               string       `json:"symbolDisplayName2"`
				TickSize                         types.Number `json:"tickSize"`
				UnderlyingCurrencyID             string       `json:"underlyingCurrencyId"`
				MaxMaintenanceMarginRate         types.Number `json:"maxMaintenanceMarginRate"`
				MaxPositionValue                 types.Number `json:"maxPositionValue"`
				TagIconURL                       string       `json:"tagIconUrl"`
				Tag                              string       `json:"tag"`
				RiskTip                          bool         `json:"riskTip"`
				DefaultLeverage                  string       `json:"defaultLeverage"`
				KlineStartTime                   types.Time   `json:"klineStartTime"`
			} `json:"perpetualContract"`
			MultiChain struct {
				Chains []struct {
					Chain             string       `json:"chain"`
					ChainID           int64        `json:"chainId"`
					ChainIconURL      string       `json:"chainIconUrl"`
					ContractAddress   string       `json:"contractAddress"`
					DepositGasFeeLess bool         `json:"depositGasFeeLess"`
					StopDeposit       bool         `json:"stopDeposit"`
					FeeLess           bool         `json:"feeLess"`
					FeeRate           string       `json:"feeRate"`
					GasLess           bool         `json:"gasLess"`
					GasToken          string       `json:"gasToken"`
					MinFee            types.Number `json:"minFee"`
					DynamicFee        bool         `json:"dynamicFee"`
					RPCURL            string       `json:"rpcUrl"`
					WebRPCURL         string       `json:"webRpcUrl"`
					WebTxURL          string       `json:"webTxUrl"`
					BlockTime         string       `json:"blockTime"`
					TxConfirm         int          `json:"txConfirm"`
					Tokens            []struct {
						Decimals       int64  `json:"decimals"`
						IconURL        string `json:"iconUrl"`
						Token          string `json:"token"`
						TokenAddress   string `json:"tokenAddress"`
						PullOff        bool   `json:"pullOff"`
						WithdrawEnable bool   `json:"withdrawEnable"`
						Slippage       string `json:"slippage"`
						IsDefaultToken bool   `json:"isDefaultToken"`
						DisplayToken   string `json:"displayToken"`
					} `json:"tokens"`
					WithdrawGasFeeLess bool `json:"withdrawGasFeeLess"`
					IsGray             bool `json:"isGray"`
				} `json:"chains"`
				Currency    string       `json:"currency"`
				MaxWithdraw types.Number `json:"maxWithdraw"`
				MinDeposit  types.Number `json:"minDeposit"`
				MinWithdraw types.Number `json:"minWithdraw"`
			} `json:"multiChain"`
			DepositFromBybit bool `json:"depositFromBybit"`
		} `json:"usdcConfig"`
		USDTConfig struct {
			Currency []CurrencyInfo `json:"currency"`
			Global   struct {
				FeeAccountID                    string `json:"feeAccountId"`
				FeeAccountL2Key                 string `json:"feeAccountL2Key"`
				StarkExCollateralCurrencyID     string `json:"starkExCollateralCurrencyId"`
				StarkExFundingValidityPeriod    int64  `json:"starkExFundingValidityPeriod"`
				StarkExMaxFundingRate           string `json:"starkExMaxFundingRate"`
				StarkExOrdersTreeHeight         int64  `json:"starkExOrdersTreeHeight"`
				StarkExPositionsTreeHeight      int64  `json:"starkExPositionsTreeHeight"`
				StarkExPriceValidityPeriod      int64  `json:"starkExPriceValidityPeriod"`
				StarkExContractAddress          string `json:"starkExContractAddress"`
				RegisterEnvID                   int64  `json:"registerEnvId"`
				CrossChainAccountID             string `json:"crossChainAccountId"`
				CrossChainL2Key                 string `json:"crossChainL2Key"`
				FastWithdrawAccountID           string `json:"fastWithdrawAccountId"`
				FastWithdrawFactRegisterAddress string `json:"fastWithdrawFactRegisterAddress"`
				FastWithdrawL2Key               string `json:"fastWithdrawL2Key"`
				FastWithdrawMaxAmount           string `json:"fastWithdrawMaxAmount"`
				BybitWithdrawAccountID          string `json:"bybitWithdrawAccountId"`
				BybitWithdrawL2Key              string `json:"bybitWithdrawL2Key"`
				ExperienceMonenyAccountID       string `json:"experienceMonenyAccountId"`
				ExperienceMonenyL2Key           string `json:"experienceMonenyL2Key"`
				ExperienceMoneyAccountID        string `json:"experienceMoneyAccountId"`
				ExperienceMoneyL2Key            string `json:"experienceMoneyL2Key"`
			} `json:"global"`
			PerpetualContract []struct {
				BaselinePositionValue            string       `json:"baselinePositionValue"`
				CrossID                          int          `json:"crossId"`
				CrossSymbolID                    int          `json:"crossSymbolId"`
				CrossSymbolName                  string       `json:"crossSymbolName"`
				DigitMerge                       string       `json:"digitMerge"`
				DisplayMaxLeverage               string       `json:"displayMaxLeverage"`
				DisplayMinLeverage               string       `json:"displayMinLeverage"`
				EnableDisplay                    bool         `json:"enableDisplay"`
				EnableOpenPosition               bool         `json:"enableOpenPosition"`
				EnableTrade                      bool         `json:"enableTrade"`
				FundingImpactMarginNotional      string       `json:"fundingImpactMarginNotional"`
				FundingInterestRate              string       `json:"fundingInterestRate"`
				IncrementalInitialMarginRate     string       `json:"incrementalInitialMarginRate"`
				IncrementalMaintenanceMarginRate string       `json:"incrementalMaintenanceMarginRate"`
				IncrementalPositionValue         string       `json:"incrementalPositionValue"`
				InitialMarginRate                types.Number `json:"initialMarginRate"`
				MaintenanceMarginRate            types.Number `json:"maintenanceMarginRate"`
				MaxOrderSize                     types.Number `json:"maxOrderSize"`
				MaxPositionSize                  types.Number `json:"maxPositionSize"`
				MinOrderSize                     types.Number `json:"minOrderSize"`
				MaxMarketPriceRange              types.Number `json:"maxMarketPriceRange"`
				SettleCurrencyID                 string       `json:"settleCurrencyId"`
				StarkExOraclePriceQuorum         string       `json:"starkExOraclePriceQuorum"`
				StarkExResolution                string       `json:"starkExResolution"`
				StarkExRiskFactor                string       `json:"starkExRiskFactor"`
				StarkExSyntheticAssetID          string       `json:"starkExSyntheticAssetId"`
				StepSize                         types.Number `json:"stepSize"`
				Symbol                           string       `json:"symbol"`
				SymbolDisplayName                string       `json:"symbolDisplayName"`
				SymbolDisplayName2               string       `json:"symbolDisplayName2"`
				TickSize                         types.Number `json:"tickSize"`
				UnderlyingCurrencyID             string       `json:"underlyingCurrencyId"`
				MaxMaintenanceMarginRate         string       `json:"maxMaintenanceMarginRate"`
				MaxPositionValue                 string       `json:"maxPositionValue"`
				TagIconURL                       string       `json:"tagIconUrl"`
				Tag                              string       `json:"tag"`
				RiskTip                          bool         `json:"riskTip"`
				DefaultLeverage                  string       `json:"defaultLeverage"`
				KlineStartTime                   types.Time   `json:"klineStartTime"`
			} `json:"perpetualContract"`
			MultiChain struct {
				Chains []struct {
					Chain             string `json:"chain"`
					ChainID           int64  `json:"chainId"`
					ChainIconURL      string `json:"chainIconUrl"`
					ContractAddress   string `json:"contractAddress"`
					DepositGasFeeLess bool   `json:"depositGasFeeLess"`
					StopDeposit       bool   `json:"stopDeposit"`
					FeeLess           bool   `json:"feeLess"`
					FeeRate           string `json:"feeRate"`
					GasLess           bool   `json:"gasLess"`
					GasToken          string `json:"gasToken"`
					MinFee            string `json:"minFee"`
					DynamicFee        bool   `json:"dynamicFee"`
					RPCURL            string `json:"rpcUrl"`
					WebRPCURL         string `json:"webRpcUrl"`
					WebTxURL          string `json:"webTxUrl"`
					BlockTime         string `json:"blockTime"`
					TxConfirm         int64  `json:"txConfirm"`
					Tokens            []struct {
						Decimals       int    `json:"decimals"`
						IconURL        string `json:"iconUrl"`
						Token          string `json:"token"`
						TokenAddress   string `json:"tokenAddress"`
						PullOff        bool   `json:"pullOff"`
						WithdrawEnable bool   `json:"withdrawEnable"`
						Slippage       string `json:"slippage"`
						IsDefaultToken bool   `json:"isDefaultToken"`
						DisplayToken   string `json:"displayToken"`
					} `json:"tokens"`
					WithdrawGasFeeLess bool `json:"withdrawGasFeeLess"`
					IsGray             bool `json:"isGray"`
				} `json:"chains"`
				Currency    string `json:"currency"`
				MaxWithdraw string `json:"maxWithdraw"`
				MinDeposit  string `json:"minDeposit"`
				MinWithdraw string `json:"minWithdraw"`
			} `json:"multiChain"`
		} `json:"usdtConfig"`
	} `json:"data"`
	TimeCost int64 `json:"timeCost"`
}

// V1CurrencyConfig represents a V1 currency configuration.
type V1CurrencyConfig struct {
	ID                string       `json:"id"`
	StarkExAssetID    string       `json:"starkExAssetId"`
	StarkExResolution string       `json:"starkExResolution"`
	StepSize          types.Number `json:"stepSize"`
	ShowStep          string       `json:"showStep"`
	IconURL           string       `json:"iconUrl"`
}

// AllSymbolsV1Config represents a configuration information
type AllSymbolsV1Config struct {
	Data struct {
		Currency          []V1CurrencyConfig        `json:"currency"`
		Global            GlobalConfig              `json:"global"`
		PerpetualContract []PerpetualContractDetail `json:"perpetualContract"`
		MultiChain        MultiChainDetails         `json:"multiChain"`
	} `json:"data"`
	TimeCost int64 `json:"timeCost"`
}

// MultiChainDetails holds details about chains and summary information
type MultiChainDetails struct {
	Chains      []ChainInfo  `json:"chains"`
	Currency    string       `json:"currency"`
	MaxWithdraw types.Number `json:"maxWithdraw"`
	MinDeposit  types.Number `json:"minDeposit"`
	MinWithdraw types.Number `json:"minWithdraw"`
}

// GlobalConfig represents a global configuration details.
type GlobalConfig struct {
	FeeAccountID                    string       `json:"feeAccountId"`
	FeeAccountL2Key                 string       `json:"feeAccountL2Key"`
	StarkExCollateralCurrencyID     string       `json:"starkExCollateralCurrencyId"`
	StarkExFundingValidityPeriod    int64        `json:"starkExFundingValidityPeriod"`
	StarkExMaxFundingRate           types.Number `json:"starkExMaxFundingRate"`
	StarkExOrdersTreeHeight         int64        `json:"starkExOrdersTreeHeight"`
	StarkExPositionsTreeHeight      int64        `json:"starkExPositionsTreeHeight"`
	StarkExPriceValidityPeriod      int64        `json:"starkExPriceValidityPeriod"`
	StarkExContractAddress          string       `json:"starkExContractAddress"`
	RegisterEnvID                   int64        `json:"registerEnvId"`
	CrossChainAccountID             string       `json:"crossChainAccountId"`
	CrossChainL2Key                 string       `json:"crossChainL2Key"`
	FastWithdrawAccountID           string       `json:"fastWithdrawAccountId"`
	FastWithdrawFactRegisterAddress string       `json:"fastWithdrawFactRegisterAddress"`
	FastWithdrawL2Key               string       `json:"fastWithdrawL2Key"`
	FastWithdrawMaxAmount           string       `json:"fastWithdrawMaxAmount"`
}

// ChainInfo represents a chain information
type ChainInfo struct {
	Chain              string       `json:"chain"`
	ChainID            int64        `json:"chainId"`
	ChainIconURL       string       `json:"chainIconUrl"`
	ContractAddress    string       `json:"contractAddress"`
	DepositGasFeeLess  bool         `json:"depositGasFeeLess"`
	FeeLess            bool         `json:"feeLess"`
	FeeRate            types.Number `json:"feeRate"`
	GasLess            bool         `json:"gasLess"`
	GasToken           string       `json:"gasToken"`
	MinFee             types.Number `json:"minFee"`
	RPCURL             string       `json:"rpcUrl"`
	WebTxURL           string       `json:"webTxUrl"`
	TransactionConfirm int64        `json:"txConfirm"`
	Tokens             []TokenInfo  `json:"tokens"`
	WithdrawGasFeeLess bool         `json:"withdrawGasFeeLess"`
}

// TokenInfo represents a token info detail
type TokenInfo struct {
	Decimals     float64 `json:"decimals"`
	IconURL      string  `json:"iconUrl"`
	Token        string  `json:"token"`
	TokenAddress string  `json:"tokenAddress"`
	PullOff      bool    `json:"pullOff"`
}

// PerpetualContractDetail represents a perpetual contract detail.
type PerpetualContractDetail struct {
	BaselinePositionValue            string       `json:"baselinePositionValue"`
	CrossID                          int64        `json:"crossId"`
	CrossSymbolID                    int64        `json:"crossSymbolId"`
	CrossSymbolName                  string       `json:"crossSymbolName"`
	DigitMerge                       string       `json:"digitMerge"`
	DisplayMaxLeverage               types.Number `json:"displayMaxLeverage"`
	DisplayMinLeverage               types.Number `json:"displayMinLeverage"`
	EnableDisplay                    bool         `json:"enableDisplay"`
	EnableOpenPosition               bool         `json:"enableOpenPosition"`
	EnableTrade                      bool         `json:"enableTrade"`
	FundingImpactMarginNotional      string       `json:"fundingImpactMarginNotional"`
	FundingInterestRate              types.Number `json:"fundingInterestRate"`
	IncrementalInitialMarginRate     types.Number `json:"incrementalInitialMarginRate"`
	IncrementalMaintenanceMarginRate string       `json:"incrementalMaintenanceMarginRate"`
	IncrementalPositionValue         string       `json:"incrementalPositionValue"`
	InitialMarginRate                string       `json:"initialMarginRate"`
	MaintenanceMarginRate            string       `json:"maintenanceMarginRate"`
	MaxOrderSize                     string       `json:"maxOrderSize"`
	MaxPositionSize                  string       `json:"maxPositionSize"`
	MinOrderSize                     string       `json:"minOrderSize"`
	MaxMarketPriceRange              string       `json:"maxMarketPriceRange"`
	SettleCurrencyID                 string       `json:"settleCurrencyId"`
	StarkExOraclePriceQuorum         string       `json:"starkExOraclePriceQuorum"`
	StarkExResolution                string       `json:"starkExResolution"`
	StarkExRiskFactor                string       `json:"starkExRiskFactor"`
	StarkExSyntheticAssetID          string       `json:"starkExSyntheticAssetId"`
	StepSize                         string       `json:"stepSize"`
	Symbol                           string       `json:"symbol"`
	SymbolDisplayName                string       `json:"symbolDisplayName"`
	TickSize                         types.Number `json:"tickSize"`
	UnderlyingCurrencyID             string       `json:"underlyingCurrencyId"`
	MaxMaintenanceMarginRate         types.Number `json:"maxMaintenanceMarginRate"`
	MaxPositionValue                 types.Number `json:"maxPositionValue"`
}

// NonceResponse represents a nonce response.
type NonceResponse struct {
	Nonce        string     `json:"nonce"`
	NonceExpired types.Time `json:"nonceExpired"`
}

// WsMessage represents a websocket input message.
type WsMessage struct {
	Operation string   `json:"op"`
	Args      []string `json:"args"`
}

// WsDepth represents a websocket orderbook data.
type WsDepth struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Symbol   string            `json:"s"`
		Bids     [][2]types.Number `json:"b"`
		Asks     [][2]types.Number `json:"a"`
		UpdateID int64             `json:"u"`
	} `json:"data"`
	Cs        int64      `json:"cs"`
	Timestamp types.Time `json:"ts"`
}

// WsTrade represents a trade data pushed through the websocket stream.
type WsTrade struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  []struct {
		Timestamp       types.Time   `json:"T"`
		Symbol          string       `json:"s"`
		Side            string       `json:"S"`
		Volume          types.Number `json:"v"`
		Price           types.Number `json:"p"`
		TickerDirection string       `json:"L"`
		OrderID         string       `json:"i"`
	} `json:"data"`
	Cs        int64      `json:"cs"`
	Timestamp types.Time `json:"ts"`
}

// WsTicker represents a ticker item data.
type WsTicker struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Symbol               string       `json:"symbol"`
		LastPrice            types.Number `json:"lastPrice"`
		Price24HPcnt         types.Number `json:"price24hPcnt"`
		HighPrice24H         types.Number `json:"highPrice24h"`
		LowPrice24H          types.Number `json:"lowPrice24h"`
		Turnover24H          types.Number `json:"turnover24h"`
		Volume24H            types.Number `json:"volume24h"`
		NextFundingTime      time.Time    `json:"nextFundingTime"`
		OraclePrice          types.Number `json:"oraclePrice"`
		IndexPrice           types.Number `json:"indexPrice"`
		OpenInterest         types.Number `json:"openInterest"`
		TradeCount           types.Number `json:"tradeCount"`
		FundingRate          types.Number `json:"fundingRate"`
		PredictedFundingRate types.Number `json:"predictedFundingRate"`
	} `json:"data"`
	Cs        int64      `json:"cs"`
	Timestamp types.Time `json:"ts"`
}

// UserData represents an account user information.
type UserData struct {
	EthereumAddress          string `json:"ethereumAddress"`
	IsRegistered             bool   `json:"isRegistered"`
	Email                    string `json:"email"`
	Username                 string `json:"username"`
	UserData                 any    `json:"userData"`
	IsEmailVerified          bool   `json:"isEmailVerified"`
	EmailNotifyGeneralEnable bool   `json:"emailNotifyGeneralEnable"`
	EmailNotifyTradingEnable bool   `json:"emailNotifyTradingEnable"`
	EmailNotifyAccountEnable bool   `json:"emailNotifyAccountEnable"`
	PopupNotifyTradingEnable bool   `json:"popupNotifyTradingEnable"`
}

// UserResponse represents a user account detail response.
type UserResponse struct {
	Data    interface{} `json:"data"`
	Code    int64       `json:"code"`
	Message string      `json:"msg"`
}

// WsCandlesticks represents a list of candlestick data.
type WsCandlesticks struct {
	Topic     string            `json:"topic"`
	Data      []CandlestickData `json:"data"`
	Timestamp types.Time        `json:"ts"`
	Type      string            `json:"type"`
}

// WsSymbolsTickerInformaton represents a ticker information for assets.
type WsSymbolsTickerInformaton struct {
	Topic string `json:"topic"`
	Data  []struct {
		Symbol                    string       `json:"s"`
		LastPrice                 types.Number `json:"p"`
		Price24HrChangePercentage types.Number `json:"pr"`
		Highest24Hr               types.Number `json:"h"`
		Lowest24Hr                types.Number `json:"l"`
		OpeningPrice              types.Number `json:"op,omitempty"`
		IndexPrice                types.Number `json:"xp"`
		Turnover24Hr              types.Number `json:"to"`
		Volume24Hr                types.Number `json:"v"`
		FundingRate               types.Number `json:"fr"`
		OpenInterest              types.Number `json:"o"`
		TradeCount24Hr            types.Number `json:"tc"`
		MarkPrice                 types.Number `json:"mp,omitempty"`
	} `json:"data"`
	Type      string     `json:"type"`
	Timestamp types.Time `json:"ts"`
}

// RegistrationAndOnboardingResponse represents a registration and onboarding response.
type RegistrationAndOnboardingResponse struct {
	APIKey struct {
		APIKey string   `json:"apiKey"`
		Key    string   `json:"key"`
		Secret string   `json:"secret"`
		Remark string   `json:"remark"`
		Ips    []string `json:"ips"`
	} `json:"apiKey"`
	User struct {
		EthereumAddress          string       `json:"ethereumAddress"`
		IsRegistered             bool         `json:"isRegistered"`
		Email                    string       `json:"email"`
		Username                 string       `json:"username"`
		ReferredByAffiliateLink  string       `json:"referredByAffiliateLink"`
		AffiliateLink            string       `json:"affiliateLink"`
		ApexTokenBalance         types.Number `json:"apexTokenBalance"`
		StakedApexTokenBalance   types.Number `json:"stakedApexTokenBalance"`
		IsEmailVerified          bool         `json:"isEmailVerified"`
		IsSharingUsername        bool         `json:"isSharingUsername"`
		IsSharingAddress         bool         `json:"isSharingAddress"`
		Country                  string       `json:"country"`
		ID                       string       `json:"id"`
		AvatarURL                string       `json:"avatarUrl"`
		AvatarBorderURL          string       `json:"avatarBorderUrl"`
		EmailNotifyGeneralEnable bool         `json:"emailNotifyGeneralEnable"`
		EmailNotifyTradingEnable bool         `json:"emailNotifyTradingEnable"`
		EmailNotifyAccountEnable bool         `json:"emailNotifyAccountEnable"`
		PopupNotifyTradingEnable bool         `json:"popupNotifyTradingEnable"`
		AppNotifyTradingEnable   bool         `json:"appNotifyTradingEnable"`
	} `json:"user"`
	Account struct {
		EthereumAddress string `json:"ethereumAddress"`
		L2Key           string `json:"l2Key"`
		ID              string `json:"id"`
		Version         string `json:"version"`
		SpotAccount     struct {
			CreatedAt            types.Time `json:"createdAt"`
			UpdatedAt            types.Time `json:"updatedAt"`
			ZkAccountID          string     `json:"zkAccountId"`
			IsMultiSigEthAddress bool       `json:"isMultiSigEthAddress"`
			DefaultSubAccountID  string     `json:"defaultSubAccountId"`
			Nonce                int        `json:"nonce"`
			Status               string     `json:"status"`
			SubAccounts          []struct {
				SubAccountID       string `json:"subAccountId"`
				L2Key              string `json:"l2Key"`
				Nonce              int    `json:"nonce"`
				NonceVersion       int    `json:"nonceVersion"`
				ChangePubKeyStatus string `json:"changePubKeyStatus"`
			} `json:"subAccounts"`
		} `json:"spotAccount"`
		SpotWallets []struct {
			UserID                   string       `json:"userId"`
			AccountID                string       `json:"accountId"`
			SubAccountID             string       `json:"subAccountId"`
			Balance                  types.Number `json:"balance"`
			TokenID                  string       `json:"tokenId"`
			PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
			PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
			PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
			PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
			CreatedAt                types.Time   `json:"createdAt"`
			UpdatedAt                types.Time   `json:"updatedAt"`
		} `json:"spotWallets"`
		ExperienceMoney []struct {
			AvailableAmount types.Number `json:"availableAmount"`
			TotalNumber     types.Number `json:"totalNumber"`
			TotalAmount     types.Number `json:"totalAmount"`
			RecycledAmount  types.Number `json:"recycledAmount"`
			Token           string       `json:"token"`
		} `json:"experienceMoney"`
		ContractAccount AccountInfo `json:"contractAccount"`
		ContractWallets []struct {
			UserID                   string       `json:"userId"`
			AccountID                string       `json:"accountId"`
			Balance                  types.Number `json:"balance"`
			Asset                    string       `json:"asset"`
			PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
			PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
			PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
			PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
		} `json:"contractWallets"`
		Positions []struct {
			IsPrelaunch             bool         `json:"isPrelaunch"`
			Symbol                  string       `json:"symbol"`
			Status                  string       `json:"status"`
			Side                    string       `json:"side"`
			Size                    types.Number `json:"size"`
			EntryPrice              types.Number `json:"entryPrice"`
			ExitPrice               types.Number `json:"exitPrice"`
			CreatedAt               types.Time   `json:"createdAt"`
			UpdatedTime             types.Time   `json:"updatedTime"`
			Fee                     types.Number `json:"fee"`
			FundingFee              types.Number `json:"fundingFee"`
			LightNumbers            types.Number `json:"lightNumbers"`
			CustomInitialMarginRate string       `json:"customInitialMarginRate"`
		} `json:"positions"`
		IsNewUser bool `json:"isNewUser"`
	} `json:"account"`
}

// EditUserDataParams represents a request parameter to edit user data.
type EditUserDataParams struct {
	Email                    string `json:"email,omitempty"`
	UserData                 string `json:"userData,omitempty"`
	Username                 string `json:"username,omitempty"`
	IsSharingUsername        bool   `json:"isSharingUsername,omitempty"`
	IsSharingAddress         bool   `json:"isSharingAddress,omitempty"`
	Country                  string `json:"country,omitempty"`
	EmailNotifyGeneralEnable bool   `json:"emailNotifyGeneralEnable,omitempty"`
	EmailNotifyTradingEnable bool   `json:"emailNotifyTradingEnable,omitempty"`
	EmailNotifyAccountEnable bool   `json:"emailNotifyAccountEnable,omitempty"`
	PopupNotifyTradingEnable bool   `json:"popupNotifyTradingEnable,omitempty"`
}

// UserDataResponse represents a user data response.
type UserDataResponse struct {
	EthereumAddress          string   `json:"ethereumAddress"`
	IsRegistered             bool     `json:"isRegistered"`
	Email                    string   `json:"email"`
	Username                 string   `json:"username"`
	UserData                 struct{} `json:"userData"`
	IsEmailVerified          bool     `json:"isEmailVerified"`
	EmailNotifyGeneralEnable bool     `json:"emailNotifyGeneralEnable"`
	EmailNotifyTradingEnable bool     `json:"emailNotifyTradingEnable"`
	EmailNotifyAccountEnable bool     `json:"emailNotifyAccountEnable"`
	PopupNotifyTradingEnable bool     `json:"popupNotifyTradingEnable"`
}

// UserAccountV2 represents a V2 user account detail.
type UserAccountV2 struct {
	ID              string `json:"id"`
	StarkKey        string `json:"starkKey"`
	PositionID      string `json:"positionId"`
	EthereumAddress string `json:"ethereumAddress"`
	ExperienceMoney []struct {
		AvailableAmount types.Number `json:"availableAmount"`
		TotalNumber     types.Number `json:"totalNumber"`
		TotalAmount     types.Number `json:"totalAmount"`
		RecycledAmount  types.Number `json:"recycledAmount"`
		Token           string       `json:"token"`
	} `json:"experienceMoney"`
	Accounts  []AccountInfo `json:"accounts"`
	Wallets   any           `json:"wallets"`
	Positions []struct {
		Token                   string       `json:"token"`
		Symbol                  string       `json:"symbol"`
		Status                  string       `json:"status"`
		Side                    string       `json:"side"`
		Size                    types.Number `json:"size"`
		EntryPrice              types.Number `json:"entryPrice"`
		ExitPrice               types.Number `json:"exitPrice"`
		CreatedAt               types.Time   `json:"createdAt"`
		UpdatedTime             types.Time   `json:"updatedTime"`
		Fee                     types.Number `json:"fee"`
		FundingFee              types.Number `json:"fundingFee"`
		LightNumbers            string       `json:"lightNumbers"`
		CustomInitialMarginRate types.Number `json:"customInitialMarginRate"`
	} `json:"positions"`
}

// UserAccountDetail represents a user account detail.
type UserAccountDetail struct {
	EthereumAddress string `json:"ethereumAddress"`
	L2Key           string `json:"l2Key"`
	ID              string `json:"id"` // position ID or account ID
	Version         string `json:"version"`
	SpotAccount     struct {
		CreatedAt            types.Time `json:"createdAt"`
		UpdatedAt            types.Time `json:"updatedAt"`
		ZkAccountID          string     `json:"zkAccountId"`
		IsMultiSigEthAddress bool       `json:"isMultiSigEthAddress"`
		DefaultSubAccountID  string     `json:"defaultSubAccountId"`
		Nonce                int64      `json:"nonce"`
		Status               string     `json:"status"`
		SubAccounts          []struct {
			SubAccountID       string `json:"subAccountId"`
			L2Key              string `json:"l2Key"`
			Nonce              int64  `json:"nonce"`
			NonceVersion       int64  `json:"nonceVersion"`
			ChangePubKeyStatus string `json:"changePubKeyStatus"`
		} `json:"subAccounts"`
	} `json:"spotAccount"`
	SpotWallets []struct {
		UserID                   string       `json:"userId"`
		AccountID                string       `json:"accountId"`
		SubAccountID             string       `json:"subAccountId"`
		Balance                  types.Number `json:"balance"`
		TokenID                  string       `json:"tokenId"`
		PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
		PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
		PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
		PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
		CreatedAt                types.Time   `json:"createdAt"`
		UpdatedAt                types.Time   `json:"updatedAt"`
	} `json:"spotWallets"`
	ExperienceMoney []struct {
		AvailableAmount types.Number `json:"availableAmount"`
		TotalNumber     types.Number `json:"totalNumber"`
		TotalAmount     types.Number `json:"totalAmount"`
		RecycledAmount  types.Number `json:"recycledAmount"`
		Token           string       `json:"token"`
	} `json:"experienceMoney"`
	ContractAccount AccountInfo `json:"contractAccount"`
	ContractWallets []struct {
		UserID                   string       `json:"userId"`
		AccountID                string       `json:"accountId"`
		Asset                    string       `json:"asset"`
		Balance                  types.Number `json:"balance"`
		PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
		PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
		PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
		PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
	} `json:"contractWallets"`
	Positions []struct {
		IsPrelaunch             bool         `json:"isPrelaunch"`
		Symbol                  string       `json:"symbol"`
		Status                  string       `json:"status"`
		Side                    string       `json:"side"`
		Size                    types.Number `json:"size"`
		EntryPrice              types.Number `json:"entryPrice"`
		ExitPrice               string       `json:"exitPrice"`
		CreatedAt               types.Time   `json:"createdAt"`
		UpdatedTime             types.Time   `json:"updatedTime"`
		Fee                     types.Number `json:"fee"`
		FundingFee              types.Number `json:"fundingFee"`
		LightNumbers            string       `json:"lightNumbers"`
		CustomInitialMarginRate string       `json:"customInitialMarginRate"`
	} `json:"positions"`
	IsNewUser bool `json:"isNewUser"`
}

// UserAccountDetailV1 represents a user account detail through the v1 API endpoint.
type UserAccountDetailV1 struct {
	StarkKey     string       `json:"starkKey"`
	PositionID   string       `json:"positionId"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
	CreatedAt    types.Time   `json:"createdAt"`
	Wallets      []struct {
		UserID                   string       `json:"userId"`
		AccountID                string       `json:"accountId"`
		Asset                    string       `json:"asset"`
		Balance                  types.Number `json:"balance"`
		PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
		PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
		PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
		PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
	} `json:"wallets"`
	OpenPositions []struct {
		Symbol       string       `json:"symbol"`
		Side         string       `json:"side"`
		Size         types.Number `json:"size"`
		EntryPrice   types.Number `json:"entryPrice"`
		Fee          types.Number `json:"fee"`
		FundingFee   types.Number `json:"fundingFee"`
		CreatedAt    types.Time   `json:"createdAt"`
		UpdatedTime  types.Time   `json:"updatedTime"`
		LightNumbers string       `json:"lightNumbers"`
	} `json:"openPositions"`
	ID string `json:"id"`
}

// UserAccountBalanceResponse represents a user account balance.
type UserAccountBalanceResponse struct {
	TotalEquityValue    types.Number `json:"totalEquityValue"`
	AvailableBalance    types.Number `json:"availableBalance"`
	InitialMargin       types.Number `json:"initialMargin"`
	MaintenanceMargin   types.Number `json:"maintenanceMargin"`
	SymbolToOraclePrice map[string]struct {
		OraclePrice types.Number `json:"oraclePrice"`
		CreatedTime types.Time   `json:"createdTime"`
	} `json:"symbolToOraclePrice"`
}

// UserAccountBalanceV2Response represents a V2 user account balance information.
type UserAccountBalanceV2Response struct {
	USDTBalance *UserAccountBalanceResponse `json:"usdtBalance"`
	USDCBalance *UserAccountBalanceResponse `json:"usdcBalance"`
}

// UserWithdrawals represents users withdrawals list.
type UserWithdrawals struct {
	Transfers []UserWithdrawal `json:"transfers"`
}

// UserWithdrawalsV2 represents users withdrawals list.
type UserWithdrawalsV2 struct {
	Transfers []UserWithdrawalV2 `json:"transfers"`
}

// UserWithdrawalV2 represents a user asset withdrawal info
type UserWithdrawalV2 struct {
	ID              string       `json:"id"`
	Type            string       `json:"type"`
	CurrencyID      string       `json:"currencyId"`
	Amount          types.Number `json:"amount"`
	TransactionHash string       `json:"transactionHash"`
	Status          string       `json:"status"`
	CreatedAt       types.Time   `json:"createdAt"`
	UpdatedTime     types.Time   `json:"updatedTime"`
	ConfirmedAt     types.Time   `json:"confirmedAt"`
	ClientID        string       `json:"clientId"`
	ConfirmedCount  int64        `json:"confirmedCount"`
	RequiredCount   int64        `json:"requiredCount"`
	OrderID         string       `json:"orderId"`
	ChainID         string       `json:"chainId"`
	Fee             types.Number `json:"fee"`
}

// UserWithdrawal represents a user withdrawal information.
type UserWithdrawal struct {
	ID              string       `json:"id"`
	Type            string       `json:"type"`
	Amount          types.Number `json:"amount"`
	TransactionHash string       `json:"transactionHash"`
	Status          string       `json:"status"`
	CreatedAt       types.Time   `json:"createdAt"`
	UpdatedAt       types.Time   `json:"updatedAt"`
	ConfirmedAt     types.Time   `json:"confirmedAt"`
	FromTokenID     string       `json:"fromTokenId"`
	ToTokenID       string       `json:"toTokenId"`
	ChainID         string       `json:"chainId"`
	OrderID         string       `json:"orderId"`
	EthAddress      string       `json:"ethAddress"`
	FromEthAddress  string       `json:"fromEthAddress"`
	ToEthAddress    string       `json:"toEthAddress"`
	Fee             types.Number `json:"fee"`
	ClientID        string       `json:"clientId"`
}

// WithdrawalToAddressParams represents a withdrawal parameter to an address through the V2 API
type WithdrawalToAddressParams struct {
	Amount          float64       `json:"amount"`
	ClientOrderID   string        `json:"clientId"`
	ExpEpoch        int64         `json:"expiration"`
	Asset           currency.Code `json:"asset"`
	EthereumAddress string        `json:"ethAddress"`
}

// AssetWithdrawalParams represents a user asset withdrawal parameter.
type AssetWithdrawalParams struct {
	Amount           float64
	ClientWithdrawID string
	Timestamp        time.Time
	EthereumAddress  string
	Signature        string
	ZKAccountID      string
	SubAccountID     string
	L2Key            string
	ToChainID        string
	L2SourceTokenID  currency.Code // L2 currency(Token ID). Eg. 'USDT' or 'USDC'
	L1TargetTokenID  currency.Code // L1 currency(Token ID). Eg. 'USDT' or 'USDC'
	Fee              float64
	Nonce            string
	IsFastWithdraw   bool
}

// WithdrawalResponse represents a withdrawal placing response.
type WithdrawalResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// WithdrawalFeeInfos represents an asset withdrawal fee information
type WithdrawalFeeInfos struct {
	WithdrawFeeAndPoolBalances []struct {
		ChainID                 string       `json:"chainId"`
		TokenID                 string       `json:"tokenId"`
		Fee                     types.Number `json:"fee"`
		ZkAvailableAmount       types.Number `json:"zkAvailableAmount"`
		FastpoolAvailableAmount types.Number `json:"fastpoolAvailableAmount"`
	} `json:"withdrawFeeAndPoolBalances"`
}

// ContractTransferLimit represents a contract transfer limit detail.
type ContractTransferLimit struct {
	WithdrawAvailableAmount          types.Number `json:"withdrawAvailableAmount"`
	TransferAvailableAmount          types.Number `json:"transferAvailableAmount"`
	ExperienceMoneyAvailableAmount   types.Number `json:"experienceMoneyAvailableAmount"`
	ExperienceMoneyRecycledAmount    types.Number `json:"experienceMoneyRecycledAmount"`
	WithdrawAvailableOriginAmount    types.Number `json:"withdrawAvailableOriginAmount"`
	ExperienceMoneyNeedRecycleAmount types.Number `json:"experienceMoneyNeedRecycleAmount"`
}

// TradeHistory represents a trade history
type TradeHistory struct {
	Orders    []TradeFill `json:"orders"`
	TotalSize int64       `json:"totalSize"`
}

// TradeFill  represents a trade fill information.
type TradeFill struct {
	ID                   string       `json:"id"`
	ClientID             string       `json:"clientId"`
	AccountID            string       `json:"accountId"`
	Symbol               string       `json:"symbol"`
	Side                 string       `json:"side"`
	Price                types.Number `json:"price"`
	LimitFee             types.Number `json:"limitFee"`
	Fee                  types.Number `json:"fee"`
	TriggerPrice         types.Number `json:"triggerPrice"`
	TrailingPercent      types.Number `json:"trailingPercent"`
	Size                 types.Number `json:"size"`
	Type                 string       `json:"type"`
	CreatedAt            types.Time   `json:"createdAt"`
	UpdatedTime          types.Time   `json:"updatedTime"`
	ExpiresAt            types.Time   `json:"expiresAt"`
	Status               string       `json:"status"`
	TimeInForce          string       `json:"timeInForce"`
	PostOnly             bool         `json:"postOnly"`
	ReduceOnly           bool         `json:"reduceOnly"`
	LatestMatchFillPrice string       `json:"latestMatchFillPrice"`
	CumMatchFillSize     types.Number `json:"cumMatchFillSize"`
	CumMatchFillValue    types.Number `json:"cumMatchFillValue"`
	CumMatchFillFee      types.Number `json:"cumMatchFillFee"`
	CumSuccessFillSize   types.Number `json:"cumSuccessFillSize"`
	CumSuccessFillValue  types.Number `json:"cumSuccessFillValue"`
	CumSuccessFillFee    types.Number `json:"cumSuccessFillFee"`
}

// SymbolWorstPrice represents a worst price of a contract.
type SymbolWorstPrice struct {
	WorstPrice  types.Number `json:"worstPrice"`
	BidOnePrice types.Number `json:"bidOnePrice"`
	AskOnePrice types.Number `json:"askOnePrice"`
}

// OrderDetail represents an order detail
type OrderDetail struct {
	ID              string       `json:"id"`
	ClientOrderID   string       `json:"clientOrderId"`
	AccountID       string       `json:"accountId"`
	Symbol          string       `json:"symbol"`
	Side            string       `json:"side"`
	Price           types.Number `json:"price"`
	TriggerPrice    types.Number `json:"triggerPrice"`
	TrailingPercent string       `json:"trailingPercent"`
	Size            types.Number `json:"size"`
	OrderType       string       `json:"type"`
	CreatedAt       types.Time   `json:"createdAt"`
	ExpiresAt       types.Time   `json:"expiresAt"`
	Status          string       `json:"status"`
	TimeInForce     string       `json:"timeInForce"`
	PostOnly        bool         `json:"postOnly"`

	// Included in the V3 API response.
	LimitFee             types.Number `json:"limitFee"`
	Fee                  types.Number `json:"fee"`
	UpdatedTime          types.Time   `json:"updatedTime"`
	ReduceOnly           bool         `json:"reduceOnly"`
	LatestMatchFillPrice types.Number `json:"latestMatchFillPrice"`
	CumMatchFillSize     types.Number `json:"cumMatchFillSize"`
	CumMatchFillValue    types.Number `json:"cumMatchFillValue"`
	CumMatchFillFee      types.Number `json:"cumMatchFillFee"`
	CumSuccessFillSize   types.Number `json:"cumSuccessFillSize"`
	CumSuccessFillValue  types.Number `json:"cumSuccessFillValue"`
	CumSuccessFillFee    types.Number `json:"cumSuccessFillFee"`

	// used by the V1 API endpoint response.
	UnfillableAt types.Time `json:"unfillableAt"`
	CancelReason string     `json:"cancelReason"`

	IsDeleverage  bool         `json:"isDeleverage"`
	UpdatedAt     int64        `json:"updatedAt"`
	IsLiquidate   bool         `json:"isLiquidate"`
	RemainingSize types.Number `json:"remainingSize"`
}

// OrderHistoryResponse represents list of order.
type OrderHistoryResponse struct {
	Orders    []OrderDetail `json:"orders"`
	TotalSize int64         `json:"totalSize"`
}

// FundingRateResponse represents a list of funding rates.
type FundingRateResponse struct {
	FundingValues []struct {
		ID            string       `json:"id"`
		Symbol        string       `json:"symbol"`
		FundingValue  string       `json:"fundingValue"`
		Rate          types.Number `json:"rate"`
		PositionSize  types.Number `json:"positionSize"`
		Price         types.Number `json:"price"`
		Side          string       `json:"side"`
		Status        string       `json:"status"`
		FundingTime   types.Time   `json:"fundingTime"`
		TransactionID string       `json:"transactionId"`
	} `json:"fundingValues"`
	TotalSize int64 `json:"totalSize"`
}

// PNLHistory represents positions profit and loss(PNL) history
type PNLHistory struct {
	HistoricalPnl []PNLDetail `json:"historicalPnl"`
	TotalSize     int64       `json:"totalSize"`
}

// PNLDetail represents a profit and loss information of a symbol
type PNLDetail struct {
	Symbol       string       `json:"symbol"`
	Size         types.Number `json:"size"`
	TotalPnl     types.Number `json:"totalPnl"`
	Price        types.Number `json:"price"`
	CreatedAt    types.Time   `json:"createdAt"`
	OrderType    string       `json:"type"`
	IsLiquidate  bool         `json:"isLiquidate"`
	IsDeleverage bool         `json:"isDeleverage"`
}

// AssetValueHistory represents a historical value of an asset.
type AssetValueHistory struct {
	HistoryValues []struct {
		AccountTotalValue types.Number `json:"accountTotalValue"`
		DateTime          types.Time   `json:"dateTime"`
	} `json:"historyValues"`
}

// WithdrawalsV2 represents an asset withdrawal details
type WithdrawalsV2 struct {
	Transfers []struct {
		ID              string       `json:"id"`
		Type            string       `json:"type"`
		CurrencyID      string       `json:"currencyId"`
		Amount          types.Number `json:"amount"`
		TransactionHash string       `json:"transactionHash"`
		Status          string       `json:"status"`
		CreatedAt       types.Time   `json:"createdAt"`
		UpdatedTime     types.Time   `json:"updatedTime"`
		ConfirmedAt     types.Time   `json:"confirmedAt"`
		ClientID        string       `json:"clientId"`
		OrderID         string       `json:"orderId"`
		ChainID         string       `json:"chainId"`
		Fee             types.Number `json:"fee"`
	} `json:"transfers"`
	TotalSize int64 `json:"totalSize"`
}

// FastAndCrossChainWithdrawalFees represents a fast and cross-chain uncommon withdrawal fees
type FastAndCrossChainWithdrawalFees struct {
	Fee                 types.Number `json:"fee"`
	PoolAvailableAmount types.Number `json:"poolAvailableAmount"`
}

// TransferAndWithdrawalLimit represents an asset transfer and withdrawal limit detail.
type TransferAndWithdrawalLimit struct {
	WithdrawAvailableAmount types.Number `json:"withdrawAvailableAmount"`
	TransferAvailableAmount types.Number `json:"transferAvailableAmount"`
}

// CreateOrderParams represents a request parameter for creating order.
type CreateOrderParams struct {
	Symbol          currency.Pair `json:"symbol,omitempty"`
	Side            string        `json:"side,omitempty"`
	OrderType       string        `json:"type,omitempty"`
	Size            float64       `json:"size,omitempty,string"`
	Price           float64       `json:"price,omitempty,string"`
	LimitFee        float64       `json:"limitFee,omitempty,string"`
	ExpirationTime  int64         `json:"expiration,omitempty,string"`
	TimeInForce     string        `json:"timeInForce,omitempty"`
	TriggerPrice    float64       `json:"triggerPrice,omitempty,string"`
	TrailingPercent float64       `json:"trailingPercent,omitempty,string"`
	ClientOrderID   string        `json:"clientOrderId,omitempty"`
	ReduceOnly      bool          `json:"reduceOnly,omitempty,string"`
	Signature       string        `json:"signature,omitempty"`

	TriggerPriceType string `json:"triggerPriceType"`

	ClientID        string `json:"clientId,omitempty"`
	IsPositionTPSL  string `json:"isPositionTpsl,omitempty"`
	IsOpenTPSLOrder string `json:"isOpenTpslOrder,omitempty"`

	IsSetOpenSL string `json:"isSetOpenSl,omitempty"`
	IsSetOpenTP string `json:"isSetOpenTp,omitempty"`

	SlClientOrderID    string `json:"slClientOrderId,omitempty"`
	SlPrice            string `json:"slPrice,omitempty"`
	SlSide             string `json:"slSide,omitempty"`
	SlSize             string `json:"slSize,omitempty"`
	SlTriggerPrice     string `json:"slTriggerPrice,omitempty"`
	SlTriggerPriceType string `json:"slTriggerPriceType,omitempty"`
	SlExpiration       string `json:"slExpiration,omitempty"`
	SlLimitFee         string `json:"slLimitFee,omitempty"`
	SlSignature        string `json:"slSignature,omitempty"`
	TpClientOrderID    string `json:"tpClientOrderId,omitempty"`
	TpPrice            string `json:"tpPrice,omitempty"`
	TpSide             string `json:"tpSide,omitempty"`
	TpSize             string `json:"tpSize,omitempty"`
	TpTriggerPrice     string `json:"tpTriggerPrice,omitempty"`
	TpTriggerPriceType string `json:"tpTriggerPriceType,omitempty"`
	TpExpiration       string `json:"tpExpiration,omitempty"`
	TpLimitFee         string `json:"tpLimitFee,omitempty"`
	TpSignature        string `json:"tpSignature,omitempty"`
	SourceFlag         string `json:"sourceFlag,omitempty"`
	BrokerID           string `json:"brokerId,omitempty"`
}

// SignatureInfo holds the r and s signature string of ECDSA signature
type SignatureInfo struct {
	R string `json:"r,omitempty"`
	S string `json:"s,omitempty"`
}

// WithdrawalParams represents an asset withdrawal parameters
type WithdrawalParams struct {
	Amount   float64
	ClientID string
	Asset    currency.Code

	EthereumAddress string
	ExpEpoch        int64
}

// FastWithdrawalParams represents a cross-chain withdrawal parameters
type FastWithdrawalParams struct {
	Amount       float64       `json:"amount"`
	ClientID     string        `json:"clientId"`
	Expiration   int64         `json:"expiration"`
	Asset        currency.Code `json:"asset"`
	ERC20Address string        `json:"erc20Address"`
	ChainID      string        `json:"fee"`
	Fees         float64       `json:"chainId"`
	IPAccountID  string        `json:"lpAccountId,omitempty"`
	Signature    string        `json:"signature"`
}

// WsInput represents a websocket input data
type WsInput struct {
	Type        string   `json:"type"`
	Topics      []string `json:"topics,omitempty"`
	HTTPMethod  string   `json:"httpMethod,omitempty"`
	RequestPath string   `json:"requestPath,omitempty"`
	APIKey      string   `json:"apiKey,omitempty"`
	Passphrase  string   `json:"passphrase,omitempty"`
	Timestamp   int64    `json:"timestamp,omitempty"`
	Signature   string   `json:"signature,omitempty"`
}

// WsAuthResponse represents a response through the websocket channel
type WsAuthResponse struct {
	Type      string          `json:"type"`
	Timestamp types.Time      `json:"timestamp"`
	Topic     string          `json:"topic"`
	Contents  json.RawMessage `json:"contents,omitempty"`
}

// AccountDeleverage represents an account deleverage details
type AccountDeleverage struct {
	Symbol      string       `json:"symbol"`
	LightNumber types.Number `json:"lightNumber"`
	Side        string       `json:"side"`
}

// ContractWalletInfo represents a contract account wallet information
type ContractWalletInfo struct {
	PendingDepositAmount     types.Number `json:"pendingDepositAmount"`
	Balance                  types.Number `json:"balance"`
	PendingWithdrawAmount    types.Number `json:"pendingWithdrawAmount"`
	PendingTransferInAmount  types.Number `json:"pendingTransferInAmount"`
	PendingTransferOutAmount types.Number `json:"pendingTransferOutAmount"`
	Token                    string       `json:"token"`
}

// ExperiencedMoney represents an experienced money detail
type ExperiencedMoney struct {
	TotalAmount     string `json:"totalAmount"`
	TotalNumber     string `json:"totalNumber"`
	RecycledAmount  string `json:"recycledAmount"`
	AvailableAmount string `json:"availableAmount"`
	Token           string `json:"token"`
}

// WsAccountOrderFill represents a websocket account order fill detail
type WsAccountOrderFill struct {
	Symbol      string       `json:"symbol"`
	Side        string       `json:"side"`
	OrderID     string       `json:"orderId"`
	Fee         types.Number `json:"fee"`
	Liquidity   string       `json:"liquidity"`
	AccountID   string       `json:"accountId"`
	CreatedAt   types.Time   `json:"createdAt"`
	IsOpen      bool         `json:"isOpen"`
	Size        types.Number `json:"size"`
	Price       types.Number `json:"price"`
	QuoteAmount types.Number `json:"quoteAmount"`
	ID          string       `json:"id"`
	UpdatedAt   types.Time   `json:"updatedAt"`
}

// AuthWebsocketAccountResponse represents a detailed response of websocket account detail
type AuthWebsocketAccountResponse struct {
	Deleverages     []AccountDeleverage    `json:"deleverages"`
	ContractWallets []ContractWalletInfo   `json:"contractWallets"`
	ExperienceMoney []ExperiencedMoney     `json:"experienceMoney"`
	Orders          []OrderDetail          `json:"orders"`
	Fills           []WsAccountOrderFill   `json:"fills"`
	Positions       []AccountPositionInfo  `json:"positions"`
	Accounts        []AccountInfo          `json:"accounts"`
	Transfers       []AccountAssetTransfer `json:"transfers"`
	Wallets         []struct {
		Balance types.Number `json:"balance"`
		Asset   string       `json:"asset"`
	} `json:"wallets"`
}

// AccountPositionInfo represents an account's position details
type AccountPositionInfo struct {
	AccountID   string       `json:"accountId"`
	Symbol      string       `json:"symbol"`
	Side        string       `json:"side"`
	SumOpen     string       `json:"sumOpen"`
	RealizedPNL string       `json:"realizedPnl"`
	ExitPrice   types.Number `json:"exitPrice"`
	MaxSize     types.Number `json:"maxSize"`
	SumClose    types.Number `json:"sumClose"`
	NetFunding  types.Number `json:"netFunding"`
	EntryPrice  types.Number `json:"entryPrice"`
	CreatedAt   types.Time   `json:"createdAt"`
	Size        types.Number `json:"size"`
	ClosedAt    types.Time   `json:"closedAt"`
	UpdatedAt   types.Time   `json:"updatedAt"`
	OpenValue   types.Number `json:"openValue"`
	FundingFee  types.Number `json:"fundingFee"`
	CustomImr   string       `json:"customImr"`
}

// AccountInfo represents an account's basic information
type AccountInfo struct {
	CreatedAt             types.Time   `json:"createdAt"`
	TakerFeeRate          types.Number `json:"takerFeeRate"`
	MakerFeeRate          types.Number `json:"makerFeeRate"`
	MinInitialMarginRate  types.Number `json:"minInitialMarginRate"`
	Status                string       `json:"status"`
	Token                 string       `json:"token"`
	UnrealizePnlPriceType string       `json:"unrealizePnlPriceType"`
}

// AccountAssetTransfer represents an account's asset transfer details
type AccountAssetTransfer struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	Status          string     `json:"status"`
	TransactionID   string     `json:"transactionId"`
	CreditAsset     string     `json:"creditAsset"`
	CreditAmount    types.Time `json:"creditAmount"`
	TransactionHash types.Time `json:"transactionHash"`
	ConfirmedAt     types.Time `json:"confirmedAt"`
	CreatedAt       types.Time `json:"createdAt"`
	ExpiresAt       types.Time `json:"expiresAt"`
}

// WsAccountNotificationsResponse represents an account's notification responses
type WsAccountNotificationsResponse struct {
	UnreadNum     int                       `json:"unreadNum"`
	NotifyMsgList []AccountNotificationInfo `json:"notifyMsgList"`
	NotifyList    []AccountNotificationInfo `json:"notify_list"`
}

// AccountNotificationInfo represents an account notification detail
type AccountNotificationInfo struct {
	ID          string     `json:"id"`
	Category    int64      `json:"category"`
	Lang        string     `json:"lang"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	AndroidLink string     `json:"androidLink"`
	IosLink     string     `json:"iosLink"`
	WebLink     string     `json:"webLink"`
	Read        bool       `json:"read"`
	CreatedTime types.Time `json:"createdTime"`
}

// CreateOrderParam represents a stark order creation parameters
type CreateOrderParam struct {
	AmountCollateral    string        `json:"amount_collateral"`
	AmountFee           string        `json:"amount_fee"`
	AmountSynthetic     string        `json:"amount_synthetic"`
	AssetIDCollateral   string        `json:"asset_id_collateral"`
	AssetIDSynthetic    string        `json:"asset_id_synthetic"`
	ExpirationTimestamp string        `json:"expiration_timestamp"`
	IsBuyingSynthetic   bool          `json:"is_buying_synthetic"`
	Nonce               string        `json:"nonce"`
	OrderType           string        `json:"order_type"`
	PositionID          string        `json:"position_id"`
	PublicKey           string        `json:"public_key"`
	Signature           SignatureInfo `json:"signature"`
}

// LoanRepaymentRates represents a loan repayment rates
type LoanRepaymentRates struct {
	RepaymentTokens []struct {
		Token string       `json:"token"`
		Price types.Number `json:"price"`
		Size  types.Number `json:"size"`
	} `json:"repaymentTokens"`
}

// RepaymentTokenAndAmount holds loan repayment tokens and amount
type RepaymentTokenAndAmount struct {
	Token  currency.Code
	Amount float64
}

// UserLoanRepaymentParams holds user manual loans repayment parameter
type UserLoanRepaymentParams struct {
	RepaymentTokens     []RepaymentTokenAndAmount `json:"repaymentTokens"`
	ClientID            string                    `json:"clientId"`
	PoolRepaymentTokens []RepaymentTokenAndAmount `json:"poolRepaymentTokens"`
}

// LoanRepaymentTokenAndAmountList holds list of tokens and amount details
type LoanRepaymentTokenAndAmountList []RepaymentTokenAndAmount

// MarshalJSON serializes the LoanRepaymentTokenAndAmount into byte data
func (l LoanRepaymentTokenAndAmountList) MarshalJSON() ([]byte, error) {
	var marshaledString string
	for a := range l {
		marshaledString += l[a].Token.String() + "|" + strconv.FormatFloat(l[a].Amount, 'f', -1, 64)
	}
	byteData := append([]byte{'"'}, append([]byte(marshaledString), []byte{'"'}...)...)
	return byteData, nil
}

// UserManualRepaymentParams holds request parameters for user manual repayments
type UserManualRepaymentParams struct {
	LoanRepaymentTokenAndAmount LoanRepaymentTokenAndAmountList
	ClientID                    string
	ExpiryTime                  time.Time
	PoolRepaymentTokensDetail   LoanRepaymentTokenAndAmountList
}

// IDResponse holds id data as a response
type IDResponse struct {
	ID string `json:"id"`
}
