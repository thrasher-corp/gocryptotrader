package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Satoshi test statuses
const (
	SatoshiTestStatusPending  = "pending"
	SatoshiTestStatusApproved = "approved"
	SatoshiTestStatusExpired  = "expired"
)

// Extended public key registration statuses
const (
	ExtendedPublicKeyRegistrationStatusPending = "PENDING"
	ExtendedPublicKeyRegistrationStatusActive  = "ACTIVE"
	ExtendedPublicKeyRegistrationStatusRevoked = "REVOKED"
	ExtendedPublicKeyRegistrationStatusFailed  = "FAILED"
)

// ContactRequest holds the details of a Travel Rule contact to create
type ContactRequest struct {
	RetailInfo    *RetailInfo    `json:"retail_info,omitempty"`
	CorporateInfo *CorporateInfo `json:"corporate_info,omitempty"`
	Description   string         `json:"description"`
}

// ContactResponse holds a Travel Rule contact
type ContactResponse struct {
	ID            string         `json:"id"`
	Description   string         `json:"description"`
	RetailInfo    *RetailInfo    `json:"retail_info"`
	CorporateInfo *CorporateInfo `json:"corporate_info"`
}

// CounterpartyAddressRequest holds Travel Rule counterparty details for a cryptocurrency address
type CounterpartyAddressRequest struct {
	Address                         string `json:"address"`
	Network                         string `json:"network"`
	MemoID                          string `json:"memo_id,omitempty"`
	DestinationTag                  string `json:"destination_tag,omitempty"`
	TransferID                      uint64 `json:"transfer_id,omitempty"`
	ContactThirdParty               bool   `json:"contact_thirdparty"`
	ContactUUID                     string `json:"contact_uuid,omitempty"`
	VirtualAssetServiceProviderUUID string `json:"vasp_uuid,omitempty"`
}

// CounterpartyAddressResponse holds the Travel Rule counterparty details registered for a cryptocurrency address
type CounterpartyAddressResponse struct {
	Address                         string `json:"address"`
	Network                         string `json:"network"`
	MemoID                          string `json:"memo_id"`
	DestinationTag                  string `json:"destination_tag"`
	TransferID                      uint64 `json:"transfer_id"`
	ContactThirdParty               bool   `json:"contact_thirdparty"`
	ContactUUID                     string `json:"contact_uuid"`
	VirtualAssetServiceProviderUUID string `json:"vasp_uuid"`
}

// SatoshiTestsRequest holds the filters for fetching Satoshi tests
type SatoshiTestsRequest struct {
	Network string
	Address string
	Status  string
}

// SatoshiTestRequest holds the parameters for creating a Satoshi test
type SatoshiTestRequest struct {
	Address  string
	Network  string
	Currency currency.Code
}

// SatoshiTestResponse holds a Satoshi test used to verify ownership of an external address
type SatoshiTestResponse struct {
	ID             string        `json:"id"`
	Network        string        `json:"network"`
	Currency       currency.Code `json:"currency"`
	Amount         types.Number  `json:"amount"`
	UserAddress    string        `json:"user_address"`
	DepositAddress string        `json:"deposit_address"`
	Status         string        `json:"status"`
	Expires        types.Time    `json:"expires"`
}

// ExtendedPublicKeyRegistrationsRequest holds the filters for fetching extended public key registrations
type ExtendedPublicKeyRegistrationsRequest struct {
	Network string
	Status  string
	Offset  uint64
	Limit   uint64
}

// ExtendedPublicKeyRegistrationRequest holds the parameters for registering an extended public key
type ExtendedPublicKeyRegistrationRequest struct {
	Network           string `json:"network"`
	ExtendedPublicKey string `json:"extended_public_key"`
	Label             string `json:"label,omitempty"`
}

// ExtendedPublicKeyRegistrationResponse holds an extended public key registration
type ExtendedPublicKeyRegistrationResponse struct {
	ID          string                             `json:"id"`
	Network     string                             `json:"network"`
	Label       string                             `json:"label"`
	Status      string                             `json:"status"`
	Proof       ExtendedPublicKeyRegistrationProof `json:"proof"`
	CreatedAt   DateTime                           `json:"created_at"`
	ActivatedAt DateTime                           `json:"activated_at"`
	RevokedAt   DateTime                           `json:"revoked_at"`
}

// ExtendedPublicKeyRegistrationProof holds the proof of control for an extended public key registration
type ExtendedPublicKeyRegistrationProof struct {
	ID     string `json:"id"`
	Method string `json:"method"`
}

// ExtendedPublicKeyRevocationResponse holds the status of a revoked extended public key registration
type ExtendedPublicKeyRevocationResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// AddressVerificationResponse holds the ownership verification status of a cryptocurrency address
type AddressVerificationResponse struct {
	Network    string   `json:"network"`
	Address    string   `json:"address"`
	Verified   bool     `json:"verified"`
	Method     string   `json:"method"`
	VerifiedAt DateTime `json:"verified_at"`
	RevokedAt  DateTime `json:"revoked_at"`
}

// satoshiTestBody is the JSON body of a Satoshi test request
type satoshiTestBody struct {
	Address  string `json:"address"`
	Network  string `json:"network"`
	Currency string `json:"currency"`
}
