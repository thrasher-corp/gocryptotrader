package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// SignAgreementRequest holds the parameters of SignAgreement; one of Category or CategoryV2 is required
type SignAgreementRequest struct {
	// Category is 2 for metals commodity contracts (XAU and XAG), which stock perpetuals share, or 3 for the crude
	// oil commodity contract; Bybit adds new categories to CategoryV2 only
	Category uint64
	// CategoryV2 is 1 for metals commodity contracts (XAU and XAG), which stock perpetuals share, or 2 for the crude
	// oil commodity contract
	CategoryV2 uint64
}

// CreateSubUserRequest holds the parameters of CreateNewSubUserID
type CreateSubUserRequest struct {
	// Username is required: 6 to 16 characters with both numbers and letters, and not used before
	Username string
	// Password is optional: 8 to 30 characters with numbers and upper and lower case letters
	Password string
	// MemberType is required: 1 normal subaccount or 6 custodial subaccount
	MemberType uint64
	// Switch turns on quick login
	Switch bool
	Note   string
}

// CreateSubUserResponse holds the sub UID created
type CreateSubUserResponse struct {
	UID        string `json:"uid"`
	Username   string `json:"username"`
	MemberType uint64 `json:"memberType"`
	// Status is 1 normal, 2 login banned or 4 frozen
	Status uint64 `json:"status"`
	Remark string `json:"remark"`
}

// CreateSubUIDAPIKeyRequest holds the parameters of CreateSubUIDAPIKey
type CreateSubUIDAPIKeyRequest struct {
	SubUID   uint64
	Note     string
	ReadOnly bool
	// IPs binds the key to these IP addresses; none, or "*", binds none, and an unbound key expires after 90 days
	IPs []string
	// Permissions is required, with at least one permission list
	Permissions CreateSubUIDAPIKeyPermissionsRequest
}

// CreateSubUIDAPIKeyPermissionsRequest holds the permissions of a new sub UID API key. A nil list is not sent; an
// empty one is sent as []
type CreateSubUIDAPIKeyPermissionsRequest struct {
	// ContractTrade takes Order and Position
	ContractTrade []string
	// Spot takes SpotTrade
	Spot []string
	// Options takes OptionsTrade
	Options []string
	// Wallet takes AccountTransfer and SubMemberTransferList; fund custodial accounts are not supported
	Wallet []string
	// Exchange takes ExchangeHistory
	Exchange []string
	// Earn takes Earn
	Earn []string
}

// CreateSubUIDAPIKeyResponse holds the API key created; Secret cannot be queried again
type CreateSubUIDAPIKeyResponse struct {
	ID          string               `json:"id"`
	Note        string               `json:"note"`
	APIKey      string               `json:"apiKey"`
	ReadOnly    types.Boolean        `json:"readOnly"`
	Secret      string               `json:"secret"`
	Permissions SubAPIKeyPermissions `json:"permissions"`
}

// SubAPIKeyPermissions holds the permissions of a sub UID API key. BlockTrade, Affiliate, FiatP2P and
// FiatConvertBroker do not apply to subaccounts, and NFT and CopyTrading are deprecated, so they are always empty
type SubAPIKeyPermissions struct {
	ContractTrade     []string `json:"ContractTrade"`
	Spot              []string `json:"Spot"`
	Wallet            []string `json:"Wallet"`
	Options           []string `json:"Options"`
	Derivatives       []string `json:"Derivatives"`
	Exchange          []string `json:"Exchange"`
	Earn              []string `json:"Earn"`
	BlockTrade        []string `json:"BlockTrade"`
	Affiliate         []string `json:"Affiliate"`
	FiatP2P           []string `json:"FiatP2P"`
	FiatConvertBroker []string `json:"FiatConvertBroker"`
	NFT               []string `json:"NFT"`
	CopyTrading       []string `json:"CopyTrading"`
}

// SubUIDListResponse holds the master account's sub UIDs
type SubUIDListResponse struct {
	SubMembers []SubMember `json:"subMembers"`
}

// SubMember holds a sub UID
type SubMember struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
	// MemberType is 1 standard, 4 copy trading, 6 custodial, 12 fund custodial, 14 general AI, 15 ByCustody or 16
	// Bybit AI subaccount
	MemberType uint64 `json:"memberType"`
	// Status is 1 normal, 2 login banned or 4 frozen
	Status uint64 `json:"status"`
	// AccountMode is 5 UTA 2.0 or 6 UTA 2.0 Pro; fund custodial subaccounts report 1 classic or 3 UTA
	AccountMode uint64 `json:"accountMode"`
	Remark      string `json:"remark"`
}

// SubUIDListUnlimitedResponse holds a page of the master account's sub UIDs
type SubUIDListUnlimitedResponse struct {
	SubMembers []SubMember `json:"subMembers"`
	// NextCursor is "0" on the last page
	NextCursor string `json:"nextCursor"`
}

// FundCustodialSubAccountsResponse holds a page of fund custodial subaccounts
type FundCustodialSubAccountsResponse struct {
	SubMembers []SubMember `json:"subMembers"`
	// NextCursor is "0" on the last page
	NextCursor string `json:"nextCursor"`
}

// APIKeyInformationResponse holds the details of an API key
type APIKeyInformationResponse struct {
	ID       string        `json:"id"`
	Note     string        `json:"note"`
	APIKey   string        `json:"apiKey"`
	ReadOnly types.Boolean `json:"readOnly"`
	// Secret is always ""
	Secret      string                       `json:"secret"`
	Permissions APIKeyInformationPermissions `json:"permissions"`
	IPs         []string                     `json:"ips"`
	// Type is 1 personal or 2 connected to a third party app
	Type uint64 `json:"type"`
	// DeadlineDay and ExpiredAt only apply to keys bound to no IP address or whose account password changed;
	// DeadlineDay is the number of days left
	DeadlineDay         int64         `json:"deadlineDay"`
	ExpiredAt           time.Time     `json:"expiredAt"`
	CreatedAt           time.Time     `json:"createdAt"`
	UnifiedTradeAccount types.Boolean `json:"uta"`
	UserID              uint64        `json:"userID"`
	// InviterID is the UID that invited the account to Bybit
	InviterID        uint64 `json:"inviterID"`
	VIPLevel         string `json:"vipLevel"`
	MarketMakerLevel string `json:"mktMakerLevel"`
	// AffiliateID is 0 when the account is bound to no affiliate
	AffiliateID  uint64 `json:"affiliateID"`
	RSAPublicKey string `json:"rsaPublicKey"`
	// IsFIXAPI is whether the key has FIX API permission
	IsFIXAPI bool `json:"isFixApi"`
	IsMaster bool `json:"isMaster"`
	// ParentUID is the master account's UID, "0" when the master account makes the request
	ParentUID string `json:"parentUid"`
	// KYCLevel is LEVEL_DEFAULT, LEVEL_1 or LEVEL_2
	KYCLevel  string `json:"kycLevel"`
	KYCRegion string `json:"kycRegion"`
	// Deprecated: Unified is no longer maintained by Bybit
	Unified types.Boolean `json:"unified"`
	// UserIDInt64, InviterIDInt64 and AffiliateIDInt64 repeat the IDs as strings; Bybit's example response returns
	// them but its parameter table does not document them
	UserIDInt64      string `json:"userIDInt64"`
	InviterIDInt64   string `json:"inviterIDInt64"`
	AffiliateIDInt64 string `json:"affiliateIDInt64"`
}

// APIKeyInformationPermissions holds the permissions of an API key. FiatP2P, FiatBitPay, FiatConvertBroker, BitCard,
// ByXPost and BlockTrade do not apply to subaccounts, Affiliate only applies to affiliates, and NFT and CopyTrading are
// deprecated and always empty
type APIKeyInformationPermissions struct {
	ContractTrade     []string `json:"ContractTrade"`
	Spot              []string `json:"Spot"`
	Wallet            []string `json:"Wallet"`
	Options           []string `json:"Options"`
	Derivatives       []string `json:"Derivatives"`
	Exchange          []string `json:"Exchange"`
	Earn              []string `json:"Earn"`
	FiatP2P           []string `json:"FiatP2P"`
	FiatBitPay        []string `json:"FiatBitPay"`
	FiatConvertBroker []string `json:"FiatConvertBroker"`
	BitCard           []string `json:"BitCard"`
	ByXPost           []string `json:"ByXPost"`
	Affiliate         []string `json:"Affiliate"`
	BlockTrade        []string `json:"BlockTrade"`
	NFT               []string `json:"NFT"`
	CopyTrading       []string `json:"CopyTrading"`
	// FiatGlobalPay is returned in Bybit's example response but not documented in its parameter table
	FiatGlobalPay []string `json:"FiatGlobalPay"`
}

// SubAccountAllAPIKeysRequest holds the parameters of GetSubAccountAllAPIKeys
type SubAccountAllAPIKeysRequest struct {
	// SubMemberID is the sub UID, and is required
	SubMemberID string
	Limit       uint64
	Cursor      string
}

// SubAccountAllAPIKeysResponse holds a page of a sub UID's API keys
type SubAccountAllAPIKeysResponse struct {
	Result         []SubAccountAPIKey `json:"result"`
	NextPageCursor string             `json:"nextPageCursor"`
}

// SubAccountAPIKey holds the details of a sub UID's API key
type SubAccountAPIKey struct {
	ID     string   `json:"id"`
	IPs    []string `json:"ips"`
	APIKey string   `json:"apiKey"`
	Note   string   `json:"note"`
	// Status is 1 permanent, 2 expired, 3 within the validity period or 4 expiring within 7 days
	Status uint64 `json:"status"`
	// ExpiredAt and DeadlineDay only apply to keys bound to no IP address or whose account password changed;
	// DeadlineDay is the number of days left
	ExpiredAt time.Time `json:"expiredAt"`
	CreatedAt time.Time `json:"createdAt"`
	// Type is 1 personal or 2 connected to a third party app
	Type        uint64                      `json:"type"`
	Permissions SubAccountAPIKeyPermissions `json:"permissions"`
	// Secret is always ""
	Secret      string `json:"secret"`
	ReadOnly    bool   `json:"readOnly"`
	DeadlineDay int64  `json:"deadlineDay"`
	// Flag is the API key type
	Flag string `json:"flag"`
}

// SubAccountAPIKeyPermissions holds the permissions of a sub UID's API key. Affiliate and BlockTrade do not apply to
// subaccounts, and NFT and CopyTrading are deprecated, so they are always empty
type SubAccountAPIKeyPermissions struct {
	ContractTrade []string `json:"ContractTrade"`
	Spot          []string `json:"Spot"`
	Wallet        []string `json:"Wallet"`
	Options       []string `json:"Options"`
	Derivatives   []string `json:"Derivatives"`
	Exchange      []string `json:"Exchange"`
	Earn          []string `json:"Earn"`
	Affiliate     []string `json:"Affiliate"`
	BlockTrade    []string `json:"BlockTrade"`
	NFT           []string `json:"NFT"`
	CopyTrading   []string `json:"CopyTrading"`
}

// UIDWalletTypeResponse holds the wallet types of each requested account
type UIDWalletTypeResponse struct {
	Accounts []UIDWalletType `json:"accounts"`
}

// UIDWalletType holds the wallet types of an account
type UIDWalletType struct {
	UID string `json:"uid"`
	// AccountType holds FUND and UNIFIED
	AccountType []string `json:"accountType"`
}

// ModifyMasterAPIKeyRequest holds the parameters of ModifyMasterAPIKey; Bybit no longer allows changing a key's IP
// addresses
type ModifyMasterAPIKeyRequest struct {
	// ReadOnly is not sent when nil
	ReadOnly *bool
	// Permissions are not sent when every list is nil
	Permissions ModifyMasterAPIKeyPermissionsRequest
}

// ModifyMasterAPIKeyPermissionsRequest holds the permissions to give a master account API key. A nil list is not sent;
// an empty one is sent as []. A read and write key cannot add or remove FiatP2P, FiatBitPay or FiatConvertBroker
type ModifyMasterAPIKeyPermissionsRequest struct {
	// ContractTrade takes Order and Position
	ContractTrade []string
	// Spot takes SpotTrade
	Spot []string
	// Wallet takes AccountTransfer and SubMemberTransfer
	Wallet []string
	// Options takes OptionsTrade
	Options []string
	// Exchange takes ExchangeHistory
	Exchange []string
	// Earn takes Earn
	Earn []string
	// FiatP2P takes FiatP2POrder and Advertising
	FiatP2P []string
	// FiatBitPay takes FaitPayOrder
	FiatBitPay []string
	// FiatConvertBroker takes FiatConvertBrokerOrder
	FiatConvertBroker []string
	// BitCard takes BitCard
	BitCard []string
	// ByXPost takes ByXPost
	ByXPost []string
	// Affiliate takes Affiliate, only applies to affiliates and needs every other permission removed
	Affiliate []string
	// Derivatives takes DerivativesTrade
	Derivatives []string
	// BlockTrade takes BlockTrade
	BlockTrade []string
}

// ModifyMasterAPIKeyResponse holds the master account API key's new settings
type ModifyMasterAPIKeyResponse struct {
	ID       string        `json:"id"`
	Note     string        `json:"note"`
	APIKey   string        `json:"apiKey"`
	ReadOnly types.Boolean `json:"readOnly"`
	// Secret is always ""
	Secret      string                  `json:"secret"`
	Permissions MasterAPIKeyPermissions `json:"permissions"`
	IPs         []string                `json:"ips"`
}

// MasterAPIKeyPermissions holds the permissions of a master account API key; NFT and CopyTrading are deprecated, so
// they are always empty
type MasterAPIKeyPermissions struct {
	ContractTrade     []string `json:"ContractTrade"`
	Spot              []string `json:"Spot"`
	Wallet            []string `json:"Wallet"`
	Options           []string `json:"Options"`
	Derivatives       []string `json:"Derivatives"`
	BlockTrade        []string `json:"BlockTrade"`
	Exchange          []string `json:"Exchange"`
	Earn              []string `json:"Earn"`
	Affiliate         []string `json:"Affiliate"`
	FiatP2P           []string `json:"FiatP2P"`
	FiatBitPay        []string `json:"FiatBitPay"`
	FiatConvertBroker []string `json:"FiatConvertBroker"`
	NFT               []string `json:"NFT"`
	CopyTrading       []string `json:"CopyTrading"`
}

// ModifySubAPIKeyRequest holds the parameters of ModifySubAPIKey
type ModifySubAPIKeyRequest struct {
	// APIKey is the sub account API key to change; a master account's key must set it and a sub account's key must not
	APIKey string
	// ReadOnly is not sent when nil
	ReadOnly *bool
	// IPs binds the key to these IP addresses; none, or "*", binds none, and an unbound key expires after 90 days
	IPs []string
	// Permissions are not sent when every list is nil
	Permissions ModifySubAPIKeyPermissionsRequest
}

// ModifySubAPIKeyPermissionsRequest holds the permissions to give a sub account API key. A nil list is not sent; an
// empty one is sent as []
type ModifySubAPIKeyPermissionsRequest struct {
	// ContractTrade takes Order and Position
	ContractTrade []string
	// Spot takes SpotTrade
	Spot []string
	// Wallet takes AccountTransfer and SubMemberTransferList; fund custodial accounts are not supported
	Wallet []string
	// Options takes OptionsTrade
	Options []string
	// Derivatives takes DerivativesTrade
	Derivatives []string
	// Exchange takes ExchangeHistory
	Exchange []string
	// Earn takes Earn
	Earn []string
}

// ModifySubAPIKeyResponse holds the sub account API key's new settings
type ModifySubAPIKeyResponse struct {
	ID       string        `json:"id"`
	Note     string        `json:"note"`
	APIKey   string        `json:"apiKey"`
	ReadOnly types.Boolean `json:"readOnly"`
	// Secret is always ""
	Secret      string               `json:"secret"`
	Permissions SubAPIKeyPermissions `json:"permissions"`
	IPs         []string             `json:"ips"`
}

// FriendReferralsRequest holds the parameters of GetFriendReferrals; every field is optional
type FriendReferralsRequest struct {
	// Status is 0 alive or 1 invalid; both are returned when it is unset
	Status string
	// Size is the page size, from 1 to 100 and 20 by default
	Size   uint64
	Cursor string
}

// FriendReferralsResponse holds a page of referred users
type FriendReferralsResponse struct {
	NextCursor string           `json:"nextCursor"`
	Records    []FriendReferral `json:"records"`
}

// FriendReferral holds a referred user
type FriendReferral struct {
	ID         string `json:"id"`
	InviteeUID string `json:"inviteeUid"`
	// Status is 0 alive or 1 invalid
	Status    uint64     `json:"status"`
	CreatedAt types.Time `json:"createdAt"`
	UpdatedAt types.Time `json:"updatedAt"`
}

// ReferralCodeResponse holds the account's referral codes
type ReferralCodeResponse struct {
	ReferralCodes []ReferralCode `json:"referralCodes"`
}

// ReferralCode holds a referral code and its registration link
type ReferralCode struct {
	ReferralCode string `json:"referralCode"`
	ReferralLink string `json:"referralLink"`
	// Scene is 1 affiliate or 2 friend
	Scene uint64 `json:"scene"`
}
