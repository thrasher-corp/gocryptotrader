package hyperliquid

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SpotMetadataResponse contains the spot markets and the tokens they trade
type SpotMetadataResponse struct {
	Universe []SpotAssetMetadata `json:"universe"`
	Tokens   []SpotTokenMetadata `json:"tokens"`
}

// SpotAssetMetadata contains the metadata of one spot market
type SpotAssetMetadata struct {
	// Tokens contains the base then the quote token index
	Tokens      []uint64 `json:"tokens"`
	Name        string   `json:"name"`
	Index       uint64   `json:"index"`
	IsCanonical bool     `json:"isCanonical"`
}

// SpotTokenMetadata contains the metadata of one spot token
type SpotTokenMetadata struct {
	Name                    currency.Code `json:"name"`
	SizeDecimals            uint64        `json:"szDecimals"`
	WeiDecimals             uint64        `json:"weiDecimals"`
	Index                   uint64        `json:"index"`
	TokenID                 string        `json:"tokenId"`
	IsCanonical             bool          `json:"isCanonical"`
	EVMContract             *EVMContract  `json:"evmContract"`
	FullName                string        `json:"fullName"`
	DeployerTradingFeeShare types.Number  `json:"deployerTradingFeeShare"`
	DeployerLabel           string        `json:"deployerLabel"`
}

// EVMContract contains the HyperEVM contract linked to a spot token
type EVMContract struct {
	Address string `json:"address"`
	// EVMExtraWeiDecimals is the EVM contract's additional decimals over the token's wei decimals, and may be negative
	EVMExtraWeiDecimals int64 `json:"evm_extra_wei_decimals"`
}

// SpotAssetContext contains current market data for one spot coin
type SpotAssetContext struct {
	PreviousDayPrice  types.Number `json:"prevDayPx"`
	DayNotionalVolume types.Number `json:"dayNtlVlm"`
	MarkPrice         types.Number `json:"markPx"`
	MidPrice          types.Number `json:"midPx"`
	CirculatingSupply types.Number `json:"circulatingSupply"`
	Coin              string       `json:"coin"`
	TotalSupply       types.Number `json:"totalSupply"`
	DayBaseVolume     types.Number `json:"dayBaseVlm"`
}

// SpotMetadataAndAssetContextsResponse contains spot metadata and current market contexts
// Contexts do not align with the universe: they include coins outside it, such as outcome coins, so match them by Coin
type SpotMetadataAndAssetContextsResponse struct {
	Metadata      SpotMetadataResponse
	AssetContexts []SpotAssetContext
}

// UnmarshalJSON decodes a [metadata, contexts] tuple
func (s *SpotMetadataAndAssetContextsResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &s.Metadata, &s.AssetContexts)
}

// SpotClearinghouseStateResponse contains an account's spot token balances
// Unified and portfolio margin accounts also hold their perpetual trading balance here, list zero balances of every
// collateral token, and report how much of each collateral token is available after maintenance margin
type SpotClearinghouseStateResponse struct {
	Balances                         []SpotBalance `json:"balances"`
	TokenToAvailableAfterMaintenance []TokenAmount `json:"tokenToAvailableAfterMaintenance"`
}

// TokenAmount contains an amount of one spot token, by token index
type TokenAmount struct {
	Token  uint64
	Amount types.Number
}

// UnmarshalJSON decodes a [token, amount] tuple
func (t *TokenAmount) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &t.Token, &t.Amount)
}

// SpotBalance contains one spot token balance
type SpotBalance struct {
	Coin          currency.Code `json:"coin"`
	Token         uint64        `json:"token"`
	Total         types.Number  `json:"total"`
	Hold          types.Number  `json:"hold"`
	EntryNotional types.Number  `json:"entryNtl"`
	// SpotHold is undocumented and sent only for portfolio margin accounts, whose Hold can be negative
	SpotHold types.Number `json:"spotHold"`
}

// SpotDeployStateResponse contains a deployer's in-progress spot token deployments and the spot deploy gas auction
type SpotDeployStateResponse struct {
	States     []SpotDeployState  `json:"states"`
	GasAuction GasAuctionResponse `json:"gasAuction"`
}

// SpotDeployState contains the progress of one spot token deployment
type SpotDeployState struct {
	Token                   uint64                 `json:"token"`
	Specification           SpotTokenSpecification `json:"spec"`
	FullName                string                 `json:"fullName"`
	DeployerTradingFeeShare types.Number           `json:"deployerTradingFeeShare"`
	// Spots contains the indices of the spot pairs registered so far
	Spots                        []uint64     `json:"spots"`
	MaxSupply                    types.Number `json:"maxSupply"`
	HyperliquidityGenesisBalance types.Number `json:"hyperliquidityGenesisBalance"`
	// TotalGenesisBalanceInWei can reach the maximum uint64, which a float64 cannot hold exactly
	TotalGenesisBalanceInWei     uint64                 `json:"totalGenesisBalanceWei,string"`
	UserGenesisBalances          []UserBalance          `json:"userGenesisBalances"`
	ExistingTokenGenesisBalances []ExistingTokenBalance `json:"existingTokenGenesisBalances"`
	BlacklistUsers               []string               `json:"blacklistUsers"`
}

// SpotTokenSpecification contains the name and decimals a spot token was registered with
type SpotTokenSpecification struct {
	Name         currency.Code `json:"name"`
	SizeDecimals uint64        `json:"szDecimals"`
	WeiDecimals  uint64        `json:"weiDecimals"`
}

// UserBalance contains an address's balance of a token, in token units
type UserBalance struct {
	User    string
	Balance types.Number
}

// UnmarshalJSON decodes a [user, balance] tuple
func (u *UserBalance) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &u.User, &u.Balance)
}

// ExistingTokenBalance contains a genesis amount shared among the holders of an existing token, in units of the new token
type ExistingTokenBalance struct {
	Token   uint64
	Balance types.Number
}

// UnmarshalJSON decodes a [token, balance] tuple
func (e *ExistingTokenBalance) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &e.Token, &e.Balance)
}

// TokenDetailsResponse contains a spot token's supply, prices, genesis distribution and deployment details; protocol
// tokens such as USDC, PURR and HYPE have no deployer
type TokenDetailsResponse struct {
	Name              currency.Code `json:"name"`
	MaxSupply         types.Number  `json:"maxSupply"`
	TotalSupply       types.Number  `json:"totalSupply"`
	CirculatingSupply types.Number  `json:"circulatingSupply"`
	SizeDecimals      uint64        `json:"szDecimals"`
	WeiDecimals       uint64        `json:"weiDecimals"`
	// MidPrice is zero when the token's book is empty
	MidPrice                   types.Number  `json:"midPx"`
	MarkPrice                  types.Number  `json:"markPx"`
	PreviousDayPrice           types.Number  `json:"prevDayPx"`
	Genesis                    *TokenGenesis `json:"genesis"`
	Deployer                   string        `json:"deployer"`
	DeployGas                  types.Number  `json:"deployGas"`
	DeployTime                 ZonelessTime  `json:"deployTime"`
	SeededUSDC                 types.Number  `json:"seededUsdc"`
	NonCirculatingUserBalances []UserBalance `json:"nonCirculatingUserBalances"`
	FutureEmissions            types.Number  `json:"futureEmissions"`
}

// TokenGenesis contains a spot token's genesis distribution
type TokenGenesis struct {
	UserBalances          []UserBalance          `json:"userBalances"`
	ExistingTokenBalances []ExistingTokenBalance `json:"existingTokenBalances"`
	BlacklistUsers        []string               `json:"blacklistUsers"`
}

// OutcomeMetadataResponse contains the active outcome markets, their questions and the outcome deployers
type OutcomeMetadataResponse struct {
	Outcomes  []OutcomeSpecification `json:"outcomes"`
	Questions []OutcomeQuestion      `json:"questions"`
	Deployers []OutcomeDeployer      `json:"deployers"`
	FeeScale  types.Number           `json:"feeScale"`
}

// OutcomeSpecification contains an outcome market's specification; protocol outcomes have no venue or deployer fee scale
type OutcomeSpecification struct {
	Outcome     uint64 `json:"outcome"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// SideSpecifications contains side 0, which settles to the settle fraction, then side 1
	SideSpecifications []OutcomeSideSpecification `json:"sideSpecs"`
	QuoteToken         currency.Code              `json:"quoteToken"`
	Venue              string                     `json:"venue"`
	DeployerFeeScale   types.Number               `json:"deployerFeeScale"`
}

// OutcomeSideSpecification contains the name of one outcome side
type OutcomeSideSpecification struct {
	Name string `json:"name"`
}

// OutcomeQuestion contains a question, whose outcomes settle with exactly one Yes
type OutcomeQuestion struct {
	Question             uint64   `json:"question"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	FallbackOutcome      uint64   `json:"fallbackOutcome"`
	NamedOutcomes        []uint64 `json:"namedOutcomes"`
	SettledNamedOutcomes []uint64 `json:"settledNamedOutcomes"`
}

// OutcomeDeployer contains an outcome deployer's venue and the addresses it has delegated deployer actions to
type OutcomeDeployer struct {
	Deployer     string        `json:"deployer"`
	Venue        string        `json:"venue"`
	SubDeployers []SubDeployer `json:"subDeployers"`
}

// SettledOutcomeResponse contains a settled outcome's specification and settlement; Question is only set for an outcome
// that belongs to a question
type SettledOutcomeResponse struct {
	Specification OutcomeSpecification `json:"spec"`
	// SettleFraction is what one side 0 token paid, in quote tokens
	SettleFraction types.Number            `json:"settleFraction"`
	Details        string                  `json:"details"`
	Question       *SettledOutcomeQuestion `json:"question"`
}

// SettledOutcomeQuestion contains the question a settled outcome belongs to
type SettledOutcomeQuestion struct {
	Question    QuestionState `json:"question"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
}

// Question states
const (
	QuestionStateActive  = "active"
	QuestionStateSettled = "settled"
)

// QuestionState contains a question ID and its state, QuestionStateActive or QuestionStateSettled
type QuestionState struct {
	Question uint64
	State    string
}

// UnmarshalJSON decodes a single-key object such as {"active":823} or {"settled":390}
func (q *QuestionState) UnmarshalJSON(data []byte) error {
	var states map[string]uint64
	if err := json.Unmarshal(data, &states); err != nil {
		return fmt.Errorf("%w: %w", errInvalidQuestionState, err)
	}
	if len(states) != 1 {
		return fmt.Errorf("%w: expected one state, got %d", errInvalidQuestionState, len(states))
	}
	for state, question := range states {
		*q = QuestionState{Question: question, State: state}
	}
	return nil
}

// OutcomeDeployerLimitsResponse contains how many more outcomes a deployer may deploy today and have active at once
type OutcomeDeployerLimitsResponse struct {
	NumberOfDailyOutcomesRemaining  uint64 `json:"nDailyOutcomesRemaining"`
	NumberOfActiveOutcomesRemaining uint64 `json:"nActiveOutcomesRemaining"`
}

// OutcomeTemplate contains a template for deploying outcomes or questions; Name, Description and side names contain
// {keyword} placeholders for the values each deployment supplies
type OutcomeTemplate struct {
	ID   string              `json:"id"`
	Role OutcomeTemplateRole `json:"role"`
	Name string              `json:"name"`
	// Description ends with metadata giving the template's category, sub-category and semantic restriction, which
	// markets deployed from it must not contradict
	Description string                   `json:"description"`
	Keywords    []OutcomeTemplateKeyword `json:"keywords"`
}

// OutcomeTemplateRole contains what an outcome template deploys; exactly one field is set
type OutcomeTemplateRole struct {
	// Question is set for a template that deploys a question
	Question bool
	// StandaloneOutcome is set for a template that deploys a single market
	StandaloneOutcome *StandaloneOutcomeRole
	// QuestionOutcome is set for a template that deploys one outcome of a question
	QuestionOutcome *QuestionOutcomeRole
}

// UnmarshalJSON decodes the "question" role string or a single-key standaloneOutcome or questionOutcome object, and
// rejects any other role
func (o *OutcomeTemplateRole) UnmarshalJSON(data []byte) error {
	if string(data) == `"question"` {
		*o = OutcomeTemplateRole{Question: true}
		return nil
	}
	var roles map[string]json.RawMessage
	if err := json.Unmarshal(data, &roles); err != nil {
		return fmt.Errorf("%w: %w", errInvalidOutcomeTemplateRole, err)
	}
	if len(roles) != 1 {
		return fmt.Errorf("%w: %s", errInvalidOutcomeTemplateRole, data)
	}
	var role OutcomeTemplateRole
	for name, payload := range roles {
		var err error
		switch name {
		case "standaloneOutcome":
			err = json.Unmarshal(payload, &role.StandaloneOutcome)
		case "questionOutcome":
			err = json.Unmarshal(payload, &role.QuestionOutcome)
		default:
			return fmt.Errorf("%w: %s", errInvalidOutcomeTemplateRole, data)
		}
		if err != nil {
			return fmt.Errorf("%w: %w", errInvalidOutcomeTemplateRole, err)
		}
	}
	if role.StandaloneOutcome == nil && role.QuestionOutcome == nil {
		return fmt.Errorf("%w: %s", errInvalidOutcomeTemplateRole, data)
	}
	*o = role
	return nil
}

// StandaloneOutcomeRole contains the side names a standalone outcome template fixes
type StandaloneOutcomeRole struct {
	// SideNames contains the names of side 0 then side 1, such as Yes and No
	SideNames []string `json:"sideNames"`
}

// QuestionOutcomeRole contains the question template that a question outcome template's outcomes must belong to
type QuestionOutcomeRole struct {
	Parent string `json:"parent"`
}

// OutcomeTemplateKeyword contains a template keyword and the hint that types its values: dateTime, date, string,
// shortString, hlPerp, uInt or uDecimal
type OutcomeTemplateKeyword struct {
	Keyword string
	Hint    string
}

// UnmarshalJSON decodes a [keyword, hint] tuple
func (o *OutcomeTemplateKeyword) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, &o.Keyword, &o.Hint)
}
