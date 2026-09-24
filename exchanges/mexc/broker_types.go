package mexc

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BrokerUniversalTransferHistory holds a page of broker transfer records. The endpoint answers with
// an object carrying the page and its total, not with a bare array.
type BrokerUniversalTransferHistory struct {
	Result     []*BrokerAssetTransfer `json:"result"`
	TotalCount uint64                 `json:"totalCount"`
}

// BrokerAssetTransfer holds a response data after asset transfer by brokers
type BrokerAssetTransfer struct {
	TransactionID       string        `json:"tranId"`
	FromAccount         string        `json:"fromAccount"`
	ToAccount           string        `json:"toAccount"`
	ClientTransactionID string        `json:"clientTranId"`
	Asset               currency.Code `json:"asset"`
	FromAccountType     string        `json:"fromAccountType"`
	ToAccountType       string        `json:"toAccountType"`
	FromSymbol          string        `json:"fromSymbol"`
	ToSymbol            string        `json:"toSymbol"`
	Status              string        `json:"status"`
	Amount              types.Number  `json:"amount"`
	Timestamp           types.Time    `json:"timestamp"`
}

// BrokerSubAccounts represents a broker sub-accounts and their detail.
type BrokerSubAccounts struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Data    []BrokerSubAccount `json:"data"`
}

// BrokerSubAccount holds a broker sub-account
type BrokerSubAccount struct {
	SubAccount string     `json:"subAccount"`
	Note       string     `json:"note"`
	Timestamp  types.Time `json:"timestamp"`
}

// BrokerSubAccountStatus holds broker's subaccount status information
type BrokerSubAccountStatus struct {
	Status string `json:"status"`
}

// BrokerSubAccountAPIKey holds a broker subaccount API key
type BrokerSubAccountAPIKey struct {
	SubAccount  string     `json:"subAccount"`
	Permissions string     `json:"permissions"`
	Note        string     `json:"note"`
	APIkey      string     `json:"apikey"`
	SecretKey   string     `json:"secretKey"`
	CreateTime  types.Time `json:"createTime"`
	IP          string     `json:"ip"`
}

// BrokerSubAccountAPIKeyParams holds a broker subaccount API key creation parameters
type BrokerSubAccountAPIKeyParams struct {
	SubAccount  string     `json:"subAccount"`
	Permissions StringList `json:"permissions"`
	IP          StringList `json:"ip,omitempty"`
	Note        string     `json:"note"`
}

// StringList holds a list of string values that are returned as a single
// comma-separated string when marshaled.
type StringList []string

// MarshalJSON deserialises a list of string into a comma-separated string representation
func (sl StringList) MarshalJSON() ([]byte, error) {
	return append(append([]byte("\""), []byte(strings.Join(sl, ","))...), '"'), nil
}

// BrokerSubAccountAPIKeys holds a list of subaccount API keys
type BrokerSubAccountAPIKeys struct {
	SubAccount []*BrokerSubAccountAPIKey `json:"subAccount"`
}

// BrokerSubAccountAPIKeyDeletionParams holds request parameters for deleting a subaccount API key
type BrokerSubAccountAPIKeyDeletionParams struct {
	SubAccount string `json:"subAccount"`
	APIKey     string `json:"apiKey"`
}

// BrokerSubAccountCreationParams holds request parameters for creating a broker sub-account. The
// endpoint requires the sub-account name and note; the password is optional.
type BrokerSubAccountCreationParams struct {
	SubAccount string `json:"subAccount"`
	Note       string `json:"note"`
	Password   string `json:"password,omitempty"`
}

// BrokerSubAccountDepositAddress holds a broker sub-account deposit address
type BrokerSubAccountDepositAddress struct {
	Address string        `json:"address"`
	Coin    currency.Code `json:"coin"`
	Network string        `json:"network"`
	Memo    string        `json:"memo"`
}

// BrokerSubAccountDepositAddressCreationParams holds sub-account deposit address creation parameter
type BrokerSubAccountDepositAddressCreationParams struct {
	Coin    currency.Code `json:"coin"`
	Network string        `json:"network"`
}

// BrokerSubAccountDepositDetail holds a broker sub-account asset deposit history item. SubAccount is
// set by the all-sub-accounts endpoint only.
type BrokerSubAccountDepositDetail struct {
	SubAccount    string        `json:"subAccount"`
	Coin          currency.Code `json:"coin"`
	Network       string        `json:"network"`
	Address       string        `json:"address"`
	AddressTag    string        `json:"addressTag"`
	TransactionID string        `json:"txId"`
	UnlockConfirm string        `json:"unlockConfirm"`
	Amount        types.Number  `json:"amount"`
	Status        types.Number  `json:"status"`
	ConfirmTimes  types.Number  `json:"confirmTimes"`
	InsertTime    types.Time    `json:"insertTime"`
	Memo          string        `json:"memo"`
}
