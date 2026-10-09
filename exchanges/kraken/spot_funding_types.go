package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errDepositMethodRequired         = errors.New("deposit method is required")
	errWithdrawalKeyRequired         = errors.New("withdrawal key is required")
	errWithdrawalReferenceIDRequired = errors.New("withdrawal reference ID is required")
)

// DepositMethodsRequest holds the parameters of Get Deposit Methods
type DepositMethodsRequest struct {
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// RebaseMultiplier is rebased, the default, to show xStocks in terms of the underlying equity, or base to show them
	// in SPV tokens
	RebaseMultiplier string
}

// DepositMethod is a method an asset can be deposited with
type DepositMethod struct {
	Method string `json:"method"`
	// Limit is the most that can be deposited right now, net of fees
	Limit DepositLimit `json:"limit"`
	// Fee is a flat fee; some methods, such as Lightning, charge FeePercentage instead
	Fee             types.Number `json:"fee"`
	FeePercentage   types.Number `json:"fee-percentage"`
	AddressSetupFee types.Number `json:"address-setup-fee"`
	// GenerateAddress is set when the method can generate new addresses
	GenerateAddress bool `json:"gen-address"`
	// Minimum is the least that can be deposited right now, net of fees
	Minimum types.Number `json:"minimum"`
}

// DepositLimit is a deposit method's limit
type DepositLimit struct {
	Amount types.Number
	// Unlimited is set when the method has no limit, which Kraken reports as false in place of an amount
	Unlimited bool
}

// UnmarshalJSON decodes an amount, or the false Kraken sends when there is no limit
func (d *DepositLimit) UnmarshalJSON(data []byte) error {
	if string(data) == "false" {
		*d = DepositLimit{Unlimited: true}
		return nil
	}
	*d = DepositLimit{}
	return json.Unmarshal(data, &d.Amount)
}

// DepositAddressesRequest holds the parameters of Get Deposit Addresses
type DepositAddressesRequest struct {
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// Method is a deposit method's name, as GetDepositMethods returns it
	Method string
	// New generates a new address
	New bool
	// Amount is the amount to be deposited, which only Bitcoin Lightning requires
	Amount float64
}

// DepositAddress is a deposit address
type DepositAddress struct {
	Address string `json:"address"`
	// ExpireTime is zero when the address does not expire
	ExpireTime types.Time `json:"expiretm"`
	// New is set when the address has never been used
	New bool `json:"new"`
	// Tag is an XRP address's destination tag, or an STX, XLM or EOS address's memo
	Tag string `json:"tag"`
	// Memo is an address's memo where Kraken sends it apart from Tag, as its docs example does for an EOS address
	Memo string `json:"memo"`
}

// RecentTransfersStatusRequest holds the parameters of Get Status of Recent Deposits and Get Status of Recent
// Withdrawals
type RecentTransfersStatusRequest struct {
	// Asset filters the transfers to an asset
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// Method filters the transfers to a deposit or withdrawal method's name
	Method string
	// Start excludes transfers created before it
	Start time.Time
	// End excludes transfers created after it
	End time.Time
	// Paginate requests the first page of a paginated response, whose NextCursor requests the next
	Paginate bool
	// Cursor requests the page a previous page's NextCursor names, and implies Paginate
	Cursor string
	// Limit is the number of transfers a page holds: 25 deposits or 500 withdrawals by default
	Limit uint64
	// RebaseMultiplier is rebased, the default, to show xStocks in terms of the underlying equity, or base to show them
	// in SPV tokens
	RebaseMultiplier string
}

// RecentDepositsStatusResponse holds recent deposits, newest first
type RecentDepositsStatusResponse struct {
	Deposits []RecentDeposit `json:"deposit"`
	// NextCursor requests the next page as RecentTransfersStatusRequest's Cursor, and is empty when the request was not
	// paginated
	NextCursor string `json:"next_cursor"`
}

// UnmarshalJSON decodes the deposits an unpaginated request returns as an array, or the page object a paginated
// request returns
func (r *RecentDepositsStatusResponse) UnmarshalJSON(data []byte) error {
	*r = RecentDepositsStatusResponse{}
	if len(data) != 0 && data[0] == '[' {
		return json.Unmarshal(data, &r.Deposits)
	}
	type page RecentDepositsStatusResponse
	return json.Unmarshal(data, (*page)(r))
}

// RecentDeposit is a recent deposit and its status
type RecentDeposit struct {
	Method      string        `json:"method"`
	AssetClass  string        `json:"aclass"`
	Asset       currency.Code `json:"asset"`
	ReferenceID string        `json:"refid"`
	// TransactionID is the method's transaction ID, such as a blockchain transaction hash
	TransactionID string `json:"txid"`
	// Information is the method's transaction information, such as the address deposited to
	Information string       `json:"info"`
	Amount      types.Number `json:"amount"`
	Fee         types.Number `json:"fee"`
	Time        types.Time   `json:"time"`
	// Status is an IFEX financial transaction state: Initial, Pending, EarlyConfirmed, Settled, Success or Failure
	Status string `json:"status"`
	// StatusProperty is return for a return Kraken initiated, or onhold for a deposit held for review
	StatusProperty string `json:"status-prop"`
	// Originators holds the sending transaction IDs of a deposit credited by a sweeping transaction
	Originators []string `json:"originators"`
}

// WithdrawalMethodsRequest holds the parameters of Get Withdrawal Methods
type WithdrawalMethodsRequest struct {
	// Asset filters the methods to an asset
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// Network filters the methods to a network, such as Ethereum
	Network string
	// RebaseMultiplier is rebased, the default, to show xStocks in terms of the underlying equity, or base to show them
	// in SPV tokens
	RebaseMultiplier string
}

// WithdrawalMethod is a method the account can withdraw an asset with
type WithdrawalMethod struct {
	Asset     currency.Code `json:"asset"`
	Method    string        `json:"method"`
	MethodID  string        `json:"method_id"`
	Network   string        `json:"network"`
	NetworkID string        `json:"network_id"`
	// Minimum is the least that can be withdrawn right now, net of fees
	Minimum types.Number        `json:"minimum"`
	Fee     WithdrawalMethodFee `json:"fee"`
	Limits  []WithdrawalLimit   `json:"limits"`
}

// WithdrawalMethodFee is a withdrawal method's fee
type WithdrawalMethodFee struct {
	AssetClass string        `json:"aclass"`
	Asset      currency.Code `json:"asset"`
	// Fee is a flat fee; some methods, such as Lightning, charge FeePercentage instead
	Fee           types.Number `json:"fee"`
	FeePercentage types.Number `json:"fee_percentage"`
}

// WithdrawalLimit is a limit on a withdrawal method
type WithdrawalLimit struct {
	Description string `json:"description"`
	LimitType   string `json:"limit_type"`
	// Windows holds the limit over each of its time windows, keyed by the window's length in seconds
	Windows map[uint64]WithdrawalLimitWindow `json:"limits"`
}

// WithdrawalLimitWindow is a withdrawal limit's use over a time window
type WithdrawalLimitWindow struct {
	Maximum   types.Number `json:"maximum"`
	Remaining types.Number `json:"remaining"`
	Used      types.Number `json:"used"`
}

// WithdrawalAddressesRequest holds the parameters of Get Withdrawal Addresses
type WithdrawalAddressesRequest struct {
	// Asset filters the addresses to an asset
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// Method filters the addresses to a withdrawal method's name
	Method string
	// Key filters the addresses to the one set up under a withdrawal key name
	Key string
	// Verified filters the addresses to those whose email confirmation has completed when true, or has not when false;
	// every address is returned when it is nil
	Verified *bool
}

// WithdrawalAddress is a withdrawal address set up on the account
type WithdrawalAddress struct {
	Address string        `json:"address"`
	Asset   currency.Code `json:"asset"`
	Method  string        `json:"method"`
	// Key is the address's withdrawal key name, which withdrawals take in place of the address
	Key string `json:"key"`
	// Tag is an XRP address's destination tag, or an STX, XLM or EOS address's memo
	Tag string `json:"tag"`
	// Verified is set once the address's email confirmation has completed
	Verified bool `json:"verified"`
}

// WithdrawalInformationRequest holds the parameters of Get Withdrawal Information
type WithdrawalInformationRequest struct {
	Asset currency.Code
	// Key is the withdrawal key name of an address set up on the account
	Key    string
	Amount float64
}

// WithdrawalInformationResponse holds the fee and net amount of a potential withdrawal
type WithdrawalInformationResponse struct {
	Method string `json:"method"`
	// Limit is the most that can be withdrawn right now, net of fees
	Limit types.Number `json:"limit"`
	// Amount is the net amount that would be sent, after fees
	Amount types.Number `json:"amount"`
	Fee    types.Number `json:"fee"`
}

// WithdrawFundsRequest holds the parameters of Withdraw Funds
type WithdrawFundsRequest struct {
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	// Key is the withdrawal key name of an address set up on the account
	Key string
	// Address, when set, must be the key's address, or the withdrawal fails with an invalid withdrawal address error
	Address string
	Amount  float64
	// MaxFee fails the withdrawal with EFunding:Max fee exceeded when the fee charged would be higher
	MaxFee float64
	// RebaseMultiplier is rebased, the default, to show xStocks in terms of the underlying equity, or base to show them
	// in SPV tokens
	RebaseMultiplier string
}

// WithdrawFundsResponse holds a withdrawal's reference ID
type WithdrawFundsResponse struct {
	ReferenceID string `json:"refid"`
}

// RecentWithdrawalsStatusResponse holds recent withdrawals, newest first
type RecentWithdrawalsStatusResponse struct {
	Withdrawals []RecentWithdrawal `json:"withdrawals"`
	// NextCursor requests the next page as RecentTransfersStatusRequest's Cursor, and is empty when the request was not
	// paginated
	NextCursor string `json:"next_cursor"`
}

// UnmarshalJSON decodes the withdrawals an unpaginated request returns as an array, or the page object a paginated
// request returns
func (r *RecentWithdrawalsStatusResponse) UnmarshalJSON(data []byte) error {
	*r = RecentWithdrawalsStatusResponse{}
	if len(data) != 0 && data[0] == '[' {
		return json.Unmarshal(data, &r.Withdrawals)
	}
	type page RecentWithdrawalsStatusResponse
	return json.Unmarshal(data, (*page)(r))
}

// RecentWithdrawal is a recent withdrawal and its status
type RecentWithdrawal struct {
	Method string `json:"method"`
	// Network is the network the method withdraws on
	Network     string        `json:"network"`
	AssetClass  string        `json:"aclass"`
	Asset       currency.Code `json:"asset"`
	ReferenceID string        `json:"refid"`
	// TransactionID is the method's transaction ID, such as a blockchain transaction hash
	TransactionID string `json:"txid"`
	// Information is the method's transaction information, such as the address withdrawn to
	Information string       `json:"info"`
	Amount      types.Number `json:"amount"`
	Fee         types.Number `json:"fee"`
	Time        types.Time   `json:"time"`
	// Status is an IFEX financial transaction state: Initial, Pending, Settled, Success or Failure
	Status string `json:"status"`
	// StatusProperty is cancel-pending once a cancellation is requested, canceled once it succeeds, cancel-denied once
	// it is refused, return for a return Kraken initiated, which cannot be cancelled, or onhold for a withdrawal held for
	// review
	StatusProperty string `json:"status-prop"`
	// Key is the withdrawal key name of the address withdrawn to
	Key string `json:"key"`
}

// WalletTransferResponse holds a wallet transfer's reference ID
type WalletTransferResponse struct {
	ReferenceID string `json:"refid"`
}
