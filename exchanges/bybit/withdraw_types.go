package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WithdrawalAddressListRequest holds the Get Withdrawal Address List filters
type WithdrawalAddressListRequest struct {
	// Coin is a coin name for its regular addresses, or baseCoin for universal addresses
	Coin string
	// Chain alone returns the chain's regular and universal addresses, and with Coin baseCoin only its universal
	// addresses
	Chain string
	// AddressType is 1 for internal transfer addresses or 2 for on-chain and internal transfer addresses, both of which
	// ignore Coin and Chain. Zero is not sent: the API documents 0 for on-chain addresses but no default, and its
	// unfiltered example returns on-chain addresses
	AddressType uint64
	Limit       uint64
	Cursor      string
}

// WithdrawalAddressListResponse is a page of address book entries
type WithdrawalAddressListResponse struct {
	Rows           []WithdrawalAddress `json:"rows"`
	NextPageCursor string              `json:"nextPageCursor"`
}

// WithdrawalAddress is one address book entry
type WithdrawalAddress struct {
	// Coin is baseCoin for a universal address; a currency code would upper-case that marker
	Coin    string `json:"coin"`
	Chain   string `json:"chain"`
	Address string `json:"address"`
	Tag     string `json:"tag"`
	Remark  string `json:"remark"`
	// Status is 0 normal, or 1 for a new address that cannot be withdrawn to for 24 hours
	Status uint64 `json:"status"`
	// AddressType is 0 on-chain, 1 internal transfer or 2 both
	AddressType uint64        `json:"addressType"`
	Verified    types.Boolean `json:"verified"`
	// CreatedAt is the address's creation time. The response table names it createAt, but the examples, which look
	// captured from the API, return createdAt
	CreatedAt types.Time `json:"createdAt"`
}

// WithdrawalRecordsRequest holds the Get Withdrawal Records filters; a window spans less than 30 days
type WithdrawalRecordsRequest struct {
	WithdrawID    string
	TransactionID string
	Coin          currency.Code
	// WithdrawType is 0 for on-chain withdrawals, the API's default, 1 for off-chain or 2 for both
	WithdrawType uint64
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
	Cursor       string
}

// WithdrawalRecordsResponse is a page of withdrawals
type WithdrawalRecordsResponse struct {
	Rows           []WithdrawalRecord `json:"rows"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// WithdrawalRecord is one withdrawal
type WithdrawalRecord struct {
	// TransactionID is empty for a failed or cancelled withdrawal
	TransactionID string        `json:"txID"`
	Coin          currency.Code `json:"coin"`
	Chain         string        `json:"chain"`
	Amount        types.Number  `json:"amount"`
	WithdrawFee   types.Number  `json:"withdrawFee"`
	// Status is SecurityCheck, Pending, success, CancelByUser, Reject, Fail, BlockchainConfirmed,
	// MoreInformationRequired or Unknown, and for Bybit Turkey also the HighValueReview statuses
	Status string `json:"status"`
	// ToAddress holds the Bybit UID of an internal transfer
	ToAddress  string     `json:"toAddress"`
	Tag        string     `json:"tag"`
	CreateTime types.Time `json:"createTime"`
	UpdateTime types.Time `json:"updateTime"`
	WithdrawID string     `json:"withdrawId"`
	// WithdrawType is 0 on-chain or 1 off-chain
	WithdrawType uint64 `json:"withdrawType"`
	// Fee is returned in the response examples but not described by the documentation
	Fee types.Number `json:"fee"`
	// Tax, TaxRate and TaxType apply to users with a Tax Centre configuration
	Tax     types.Number `json:"tax"`
	TaxRate types.Number `json:"taxRate"`
	TaxType string       `json:"taxType"`
}

// AvailableVASPsResponse lists the virtual asset service providers a Travel Rule withdrawal can name
type AvailableVASPsResponse struct {
	VASP []VASP `json:"vasp"`
}

// VASP is a virtual asset service provider that can receive withdrawals
type VASP struct {
	// VASPEntityID identifies the receiving platform; others covers platforms not listed
	VASPEntityID string `json:"vaspEntityId"`
	VASPName     string `json:"vaspName"`
}

// WithdrawalRequest holds the Withdraw parameters. Coin, Address, Amount and AccountType are required, and Chain is
// required unless ForceChain is 2
type WithdrawalRequest struct {
	Coin  currency.Code
	Chain string
	// Address is an address book entry, matching its case, or for ForceChain 2 another master account's Bybit UID
	Address string
	// Tag is required when the address book entry has one
	Tag    string
	Amount float64
	// Timestamp guards against replays; zero sends the current time
	Timestamp time.Time
	// ForceChain is 0 to transfer internally when the address is a Bybit one, 1 to force an on-chain withdrawal or 2
	// to withdraw to a Bybit UID
	ForceChain uint64
	// AccountType is FUND, UTA, EARN, or a comma separated combination such as FUND,UTA,EARN, which draws on the
	// Funding wallet first
	AccountType string
	// FeeType is 0 when Amount is the amount received, or 1 to deduct the handling fee from Amount
	FeeType uint64
	// RequestID is an optional idempotency key of 1 to 32 letters and digits
	RequestID string
	// TransactionPurpose is required for Bybit Turkey users and is at least 20 characters
	TransactionPurpose string
	// Questionnaire carries the Travel Rule information and supersedes Beneficiary and TransactionPurpose
	Questionnaire *TravelRuleQuestionnaire
	// Beneficiary is the legacy Travel Rule information
	Beneficiary *WithdrawalBeneficiary
}

// WithdrawalBeneficiary holds the legacy Travel Rule information of a withdrawal, which Bybit Korea, India, Turkey,
// Kazakhstan, Indonesia and UAE users need in some form; TravelRuleQuestionnaire supersedes it
type WithdrawalBeneficiary struct {
	// TransactionPurpose, RepresentativeFirstName and RepresentativeLastName are required when Korean users withdraw to
	// a company through the Korean CODE channel
	TransactionPurpose      string
	RepresentativeFirstName string
	RepresentativeLastName  string
	// VASPEntityID is the receiving platform from GetAvailableVASPs, required for Korean users
	VASPEntityID string
	// Name is the receiver's KYC name, individual or company
	Name string
	// LegalType is individual, the API's default, or company
	LegalType string
	// WalletType is 0 for a custodial wallet, the API's default, or 1 for a non-custodial wallet
	WalletType string
	// UnhostedWalletType is 0 for the user's own wallet or 1 for another's, for a non-custodial wallet
	UnhostedWalletType string
	// POINumber, POIType, POIIssuingCountry and POIExpiredDate describe the receiver's identity document, required for
	// Turkey and Kazakhstan users; POIIssuingCountry is an ISO 3166-1 alpha-3 code
	POINumber         string
	POIType           string
	POIIssuingCountry string
	POIExpiredDate    time.Time
	// The address, date of birth and place of birth fields are required for UAE users
	AddressCountry    string
	AddressState      string
	AddressCity       string
	AddressBuilding   string
	AddressStreet     string
	AddressPostalCode string
	DateOfBirth       string
	PlaceOfBirth      string
}

// WithdrawalResponse identifies a withdrawal
type WithdrawalResponse struct {
	ID string `json:"id"`
}

// TravelRuleQuestionnaire holds Travel Rule compliance information for withdrawals and deposits. The fields a
// compliance zone requires differ: see the Questionnaire documentation page (Questionnaire)
type TravelRuleQuestionnaire struct {
	// WalletType is 0 for a custodial (VASP) wallet or 1 for a non-custodial personal wallet
	WalletType uint64
	// LegalType is individual or company, and is required
	LegalType string
	// VASPCode is the counterparty's VASP code from GetAvailableVASPs, others when not listed; custodial wallets need it
	VASPCode string
	// IsSelfWallet says whether a non-custodial wallet belongs to the user
	IsSelfWallet *bool
	// FirstName and LastName are required for an individual
	FirstName string
	LastName  string
	// CompanyName is required for a company
	CompanyName string
	// TransactionPurpose is required in the EU and Turkey and is at least 20 characters
	TransactionPurpose string
	// POIType, POINumber, POIIssuingCountry and POIExpiredDate describe the proof of identity document Turkey and
	// Kazakhstan require. POIType is ID_CARD, PASSPORT, DRIVERS, RESIDENCE_PERMIT or OTHER, and POIIssuingCountry an
	// ISO 3166-1 alpha-3 code
	POIType           string
	POINumber         string
	POIIssuingCountry string
	POIExpiredDate    time.Time
	// RepresentativeFirstName and RepresentativeLastName name a company's representative for Korean withdrawals
	RepresentativeFirstName string
	RepresentativeLastName  string
}
