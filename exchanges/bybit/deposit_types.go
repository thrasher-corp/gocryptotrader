package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// StatusResponse is the outcome of a request that reports only success or failure
type StatusResponse struct {
	// Status is true when the request succeeded
	Status types.Boolean `json:"status"`
}

// DepositRecordsRequest holds the Get Deposit Records (on-chain) filters; a window spans less than 30 days
type DepositRecordsRequest struct {
	// ID identifies one deposit and takes priority over the other filters
	ID string
	// TransactionID cannot find deposits made before 1 January 2024
	TransactionID string
	Coin          currency.Code
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
	Cursor        string
}

// DepositRecordsResponse is a page of on-chain deposits
type DepositRecordsResponse struct {
	Rows           []DepositRecord `json:"rows"`
	NextPageCursor string          `json:"nextPageCursor"`
}

// DepositRecord is one on-chain deposit
type DepositRecord struct {
	ID            string        `json:"id"`
	Coin          currency.Code `json:"coin"`
	Chain         string        `json:"chain"`
	Amount        types.Number  `json:"amount"`
	TransactionID string        `json:"txID"`
	// Status is 0 unknown, 1 to be confirmed, 2 processing, 3 success, 4 failed, 7 rollback processing, 10011 pending
	// credit to the funding pool, 10012 credited to the funding pool, 70011 rolled back, 70012 successful after
	// rollback review, or 70013 rollback failed
	Status     uint64       `json:"status"`
	ToAddress  string       `json:"toAddress"`
	Tag        string       `json:"tag"`
	DepositFee types.Number `json:"depositFee"`
	SuccessAt  types.Time   `json:"successAt"`
	// Confirmations is the number of confirmation blocks
	Confirmations    types.Number `json:"confirmations"`
	TransactionIndex string       `json:"txIndex"`
	BlockHash        string       `json:"blockHash"`
	// BatchReleaseLimit is the chain's deposit limit for the coin; -1 means no limit
	BatchReleaseLimit types.Number `json:"batchReleaseLimit"`
	// DepositType is 0 for a normal deposit, 10 when the daily deposit limit is reached, 20 for a deposit held for AML
	// review, or 50 for a deposit blocked by an account or compliance restriction, which must be withdrawn on the web
	DepositType string `json:"depositType"`
	// FromAddress is empty unless the deposit came on-chain from a single address
	FromAddress string `json:"fromAddress"`
	// TaxDepositRecordsID and TaxStatus serve Bybit EU (Austria) tax reporting; TaxStatus is 0 not required, 1 pending
	// or 2 completed
	TaxDepositRecordsID string `json:"taxDepositRecordsId"`
	TaxStatus           uint64 `json:"taxStatus"`
	// TravelRuleStatus is 0 approved, 1 counterparty information required, 2 under review or 3 rejected
	TravelRuleStatus uint64 `json:"travelRuleStatus"`
}

// SubDepositRecordsRequest holds the Get Sub Deposit Records (on-chain) filters; SubMemberID is required and a window
// spans less than 30 days
type SubDepositRecordsRequest struct {
	// ID identifies one deposit and takes priority over the other filters
	ID string
	// TransactionID cannot find deposits made before 1 January 2024
	TransactionID string
	SubMemberID   string
	Coin          currency.Code
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
	Cursor        string
}

// InternalDepositRecordsRequest holds the Get Internal Deposit Records (off-chain) filters; a window spans at most 30
// days
type InternalDepositRecordsRequest struct {
	TransactionID string
	StartTime     time.Time
	EndTime       time.Time
	Coin          currency.Code
	Cursor        string
	Limit         uint64
}

// InternalDepositRecordsResponse is a page of deposits made within Bybit
type InternalDepositRecordsResponse struct {
	Rows           []InternalDepositRecord `json:"rows"`
	NextPageCursor string                  `json:"nextPageCursor"`
}

// InternalDepositRecord is one deposit made within Bybit
type InternalDepositRecord struct {
	ID string `json:"id"`
	// Type is 1 for an internal deposit
	Type   uint64        `json:"type"`
	Coin   currency.Code `json:"coin"`
	Amount types.Number  `json:"amount"`
	// Status is 1 processing, 2 success or 3 failed
	Status uint64 `json:"status"`
	// Address is the sender's email address or phone number
	Address       string     `json:"address"`
	CreatedTime   types.Time `json:"createdTime"`
	FromMemberID  string     `json:"fromMemberId"`
	TransactionID string     `json:"txID"`
	// ComplianceStatus only applies to Bybit Turkey users: 0 not required, 1 pending, 2 collected or 3 completed
	ComplianceStatus uint64 `json:"complianceStatus"`
	// TaxDepositRecordsID and TaxStatus serve Bybit EU (Austria) tax reporting; TaxStatus is 0 not required, 1 pending
	// or 2 completed
	TaxDepositRecordsID string `json:"taxDepositRecordsId"`
	TaxStatus           uint64 `json:"taxStatus"`
}

// DepositAddressResponse holds a coin's deposit addresses
type DepositAddressResponse struct {
	Coin   currency.Code        `json:"coin"`
	Chains DepositAddressChains `json:"chains"`
}

// DepositAddressChains holds a coin's deposit address on each chain. Get Sub Deposit Address documents an array of
// chains, but its response examples show a single chain object, so both forms decode
type DepositAddressChains []DepositAddressChain

// UnmarshalJSON decodes an array of chains, or a single chain object as a one-chain list
func (d *DepositAddressChains) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '{' {
		var chain DepositAddressChain
		if err := json.Unmarshal(data, &chain); err != nil {
			return err
		}
		*d = DepositAddressChains{chain}
		return nil
	}
	var chains []DepositAddressChain
	if err := json.Unmarshal(data, &chains); err != nil {
		return err
	}
	*d = chains
	return nil
}

// DepositAddressChain is a coin's deposit address on one chain
type DepositAddressChain struct {
	ChainType      string `json:"chainType"`
	AddressDeposit string `json:"addressDeposit"`
	TagDeposit     string `json:"tagDeposit"`
	Chain          string `json:"chain"`
	// BatchReleaseLimit is the chain's deposit limit for the coin; -1 means no limit
	BatchReleaseLimit types.Number `json:"batchReleaseLimit"`
	// ContractAddress holds the last 6 characters of the coin's contract address, and is empty without one
	ContractAddress string `json:"contractAddress"`
}

// DepositOriginatorInfoRequest holds the Submit Deposit Originator Info parameters; DepositID and Questionnaire are
// required
type DepositOriginatorInfoRequest struct {
	// DepositID is the deposit record ID from the deposit query endpoints
	DepositID uint64
	// SubAccountID is a broker's target sub-account UID; zero means the API key's own account
	SubAccountID  uint64
	Questionnaire *TravelRuleQuestionnaire
}

// DepositOriginatorInfoResponse is a deposit's Travel Rule status after the originator information is submitted
type DepositOriginatorInfoResponse struct {
	// TravelRuleStatus is 0 approved, 1 counterparty information required, 2 under review or 3 rejected
	TravelRuleStatus uint64 `json:"travelRuleStatus"`
}

// UnmarshalJSON decodes the status. The docs table names it travelRuleStatus but the docs' response sample sends
// travel_rule_status, so either is read and the table's key wins when both are present. Presence decides, not a
// non-zero value, because 0 means approved
func (r *DepositOriginatorInfoResponse) UnmarshalJSON(data []byte) error {
	type keys struct {
		Documented *uint64 `json:"travelRuleStatus"`
		Sampled    *uint64 `json:"travel_rule_status"`
	}
	var aux keys
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	switch {
	case aux.Documented != nil:
		r.TravelRuleStatus = *aux.Documented
	case aux.Sampled != nil:
		r.TravelRuleStatus = *aux.Sampled
	}
	return nil
}

// AllowedDepositCoinInfoRequest holds the Get Allowed Deposit Coin Info filters; Coin and Chain filter alone or together
type AllowedDepositCoinInfoRequest struct {
	Coin  currency.Code
	Chain string
	// Limit and Cursor are sent, but the route returns every matching row on each page and a cursor that never ends
	Limit  uint64
	Cursor string
}

// AllowedDepositCoinInfoResponse is a page of the coins and chains Bybit lists for deposits, open to them or not
type AllowedDepositCoinInfoResponse struct {
	ConfigList     []AllowedDepositCoin `json:"configList"`
	NextPageCursor string               `json:"nextPageCursor"`
}

// AllowedDepositCoin describes depositing a coin on one chain
type AllowedDepositCoin struct {
	Coin         currency.Code `json:"coin"`
	Chain        string        `json:"chain"`
	CoinShowName string        `json:"coinShowName"`
	ChainType    string        `json:"chainType"`
	// BlockConfirmNumber is the number of confirmations after which a deposit is credited
	BlockConfirmNumber uint64       `json:"blockConfirmNumber"`
	MinDepositAmount   types.Number `json:"minDepositAmount"`
	// DepositSwitch is 1 when the chain accepts deposits and 0 when it is closed to them, since the route lists both, and
	// empty where the host leaves it out, as testnet does
	DepositSwitch string `json:"depositSwitch"`
}
