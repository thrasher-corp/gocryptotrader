package mexc

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BrokerUniversalAssetTransfer holds a response data after universal asset transfer by brokers
type BrokerAssetTransfer struct {
	TransactionID       string       `json:"tranId"`
	FromAccount         string       `json:"fromAccount"`
	ToAccount           string       `json:"toAccount"`
	ClientTransactionID string       `json:"clientTranId"`
	Asset               string       `json:"asset"`
	FromAccountType     string       `json:"fromAccountType"`
	ToAccountType       string       `json:"toAccountType"`
	FromSymbol          string       `json:"fromSymbol"`
	ToSymbol            string       `json:"toSymbol"`
	Status              string       `json:"status"`
	Amount              types.Number `json:"amount"`
	Timestamp           types.Time   `json:"timestamp"`
}

// BrokerSubAccounts represents a broker sub-accounts and their detail.
type BrokerSubAccounts struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		SubAccount string     `json:"subAccount"`
		Note       string     `json:"note"`
		Timestamp  types.Time `json:"timestamp"`
	} `json:"data"`
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

// MarshalJSON deserializes a list of string into a comma-separated string representation
func (sl StringList) MarshalJSON() ([]byte, error) {
	joinedString := strings.Join(sl, ",")
	return append(append([]byte("\""), []byte(joinedString)...), '"'), nil
}

// BrokerSubAccountAPIKeys holds a list of subaccount API keys
type BrokerSubAccountAPIKeys struct {
	SubAccount []BrokerSubAccountAPIKey `json:"subAccount"`
}

// BrokerSubAccountAPIKeyDeletionParams holds request parameters for deleting a subaccount API key
type BrokerSubAccountAPIKeyDeletionParams struct {
	SubAccount string `json:"subAccount"`
	APIKey     string `json:"apiKey"`
}

// BrokerSubAccountDepositAddress holds a broker sub-account deposit address
type BrokerSubAccountDepositAddress struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
	Network string `json:"network"`
	Memo    string `json:"memo"`
}

// BrokerSubAccountDepositAddressCreationParams holds sub-account deposit address creation parameter
type BrokerSubAccountDepositAddressCreationParams struct {
	Coin    currency.Code `json:"code"`
	Network string        `json:"network"`
}

// BrokerSubAccountDepositDetail holds a broker sub-account asset deposit history item
type BrokerSubAccountDepositDetail struct {
	Amount        types.Number `json:"amount"`
	Coin          string       `json:"coin"`
	Network       string       `json:"network"`
	Status        types.Number `json:"status"`
	Address       string       `json:"address"`
	AddressTag    string       `json:"addressTag"`
	TransactionID string       `json:"txId"`
	UnlockConfirm string       `json:"unlockConfirm"`
	ConfirmTimes  types.Number `json:"confirmTimes"`
}
