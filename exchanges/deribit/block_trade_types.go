package deribit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BlockTradeLeg contains one instrument's trade in a block trade
type BlockTradeLeg struct {
	InstrumentName string `json:"instrument_name"`
	// Direction is always the maker's, buy or sell, whichever role the caller takes
	Direction string  `json:"direction"`
	Price     float64 `json:"price"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount float64 `json:"amount"`
}

// BlockTradeApprovalRequest identifies a pending block trade to approve or reject
type BlockTradeApprovalRequest struct {
	// Timestamp and Nonce are the block trade's, as GetBlockTradeRequests returns them
	Timestamp time.Time `json:"-"`
	Nonce     string    `json:"nonce"`
	// Role is maker or taker, the opposite of the counterparty's
	Role string `json:"role"`
}

// BlockTradeApprovalParams contains a BlockTradeApprovalRequest's params, with its timestamp in Unix milliseconds
type BlockTradeApprovalParams struct {
	*BlockTradeApprovalRequest
	Timestamp uint64 `json:"timestamp"`
}

// ExecuteBlockTradeRequest executes a block trade the counterparty verified; every field but Role must match the
// counterparty's VerifyBlockTradeRequest
type ExecuteBlockTradeRequest struct {
	// Timestamp and Nonce are shared with the counterparty
	Timestamp time.Time `json:"-"`
	Nonce     string    `json:"nonce"`
	// Role is maker or taker, the opposite of the counterparty's
	Role   string          `json:"role"`
	Trades []BlockTradeLeg `json:"trades"`
	// CounterpartySignature is the signature the counterparty's VerifyBlockTrade returned
	CounterpartySignature string `json:"counterparty_signature"`
}

// ExecuteBlockTradeParams contains an ExecuteBlockTradeRequest's params, with its timestamp in Unix milliseconds
type ExecuteBlockTradeParams struct {
	*ExecuteBlockTradeRequest
	Timestamp uint64 `json:"timestamp"`
}

// BlockTradeIDRequest contains the block trade of a private/get_block_trade request
type BlockTradeIDRequest struct {
	ID string `json:"id"`
}

// BrokerCodeRequest contains the broker code of a private/get_block_trade_requests request; an empty code is left out
type BrokerCodeRequest struct {
	BrokerCode string `json:"broker_code,omitempty"`
}

// PendingBlockTradeResponse contains a block trade request awaiting the account's approval, or one approval declined or
// let expire
type PendingBlockTradeResponse struct {
	// Timestamp and Nonce identify the block trade to ApproveBlockTrade and RejectBlockTrade
	Timestamp types.Time      `json:"timestamp"`
	Nonce     string          `json:"nonce"`
	Trades    []BlockTradeLeg `json:"trades"`
	AppName   string          `json:"app_name"`
	// Username is the name of the user who initiated the block trade
	Username string `json:"username"`
	// Role is the account's role, maker or taker
	Role       string `json:"role"`
	UserID     uint64 `json:"user_id"`
	BrokerCode string `json:"broker_code"`
	BrokerName string `json:"broker_name"`
	// State is the account's approval state, and CounterpartyState the counterparty's, when Deribit reports it
	State             PendingBlockTradeState  `json:"state"`
	CounterpartyState *PendingBlockTradeState `json:"counterparty_state"`
	ComboID           string                  `json:"combo_id"`
}

// PendingBlockTradeState contains a party's approval state of a pending block trade and when it was reached
type PendingBlockTradeState struct {
	// Value is initial, accepted, rejected or executed, although Deribit's example also shows approved
	Value     string     `json:"value"`
	Timestamp types.Time `json:"timestamp"`
}

// GetBlockTradesRequest selects the account's block trades, newest first
type GetBlockTradesRequest struct {
	// Currency is left out to return block trades in every currency
	Currency currency.Code `json:"currency,omitzero"`
	// Count is the number of block trades to return, up to 101
	Count uint64 `json:"count,omitempty"`
	// StartID returns block trades older than it, and EndID, which needs StartID, is the oldest block trade returned
	StartID string `json:"start_id,omitempty"`
	EndID   string `json:"end_id,omitempty"`
	// BlockRFQID returns the block trades of one Block RFQ
	BlockRFQID uint64 `json:"block_rfq_id,omitempty"`
	// BrokerCode returns a broker client's block trades through one broker; Deribit documents "any" as returning every
	// block trade
	BrokerCode string `json:"broker_code,omitempty"`
}

// GetBrokerClientsRequest selects the clients registered under the broker account
type GetBrokerClientsRequest struct {
	// ClientID returns only that client, and is left out to return every client
	ClientID uint64 `json:"client_id,omitempty"`
	// IncludeSubaccounts includes the clients the broker's subaccounts manage
	IncludeSubaccounts bool `json:"include_subaccounts,omitempty"`
}

// BrokerClientResponse contains a client registered under the broker account and the users linked to it
type BrokerClientResponse struct {
	// ClientID identifies the client group in broker calls, together with a link's ClientLinkID
	ClientID    uint64 `json:"client_id"`
	ClientName  string `json:"client_name"`
	CompanyName string `json:"company_name"`
	// BrokerCode is the code of the broker's invitation links for this client
	BrokerCode string         `json:"broker_code"`
	MainUserID ObscuredUserID `json:"main_user_id"`
	// BrokerSubUserID is the broker subaccount managing the client, if any
	BrokerSubUserID            uint64             `json:"broker_sub_user_id"`
	LegalEntityIdentifier      string             `json:"lei"`
	BusinessRegistrationNumber string             `json:"brn"`
	Links                      []BrokerClientLink `json:"links"`
}

// BrokerClientLink contains a user linked to a broker client
type BrokerClientLink struct {
	ClientLinkID uint64         `json:"client_link_id"`
	UserID       ObscuredUserID `json:"user_id"`
	Name         string         `json:"name"`
	Note         string         `json:"note"`
	// State is connected once the user accepts the broker's invitation, pending until then, or rejected when inactive
	State string `json:"state"`
	// ConfirmationsRequired is the number of broker confirmations a block trade involving the link needs
	ConfirmationsRequired uint64 `json:"confirmations_required"`
	BrokerSubUserID       uint64 `json:"broker_sub_user_id"`
}

// PendingBrokerTradeResponse contains a broker block trade request awaiting its parties' approval, or one declined or
// let expire
type PendingBrokerTradeResponse struct {
	// Timestamp and Nonce identify the request for approval or rejection
	Timestamp types.Time       `json:"timestamp"`
	Nonce     string           `json:"nonce"`
	State     string           `json:"state"`
	Trades    []BlockTradeLeg  `json:"trades"`
	Maker     BrokerTradeParty `json:"maker"`
	Taker     BrokerTradeParty `json:"taker"`
}

// BrokerTradeParty contains a broker client's side of a broker block trade
type BrokerTradeParty struct {
	// State is initial, approved or rejected; only broker block trade requests report it
	State          string         `json:"state"`
	ClientID       uint64         `json:"client_id"`
	UserID         ObscuredUserID `json:"user_id"`
	ClientName     string         `json:"client_name"`
	ClientLinkName string         `json:"client_link_name"`
	ClientLinkID   uint64         `json:"client_link_id"`
}

// GetBrokerTradesRequest selects the broker's block trades, newest first
type GetBrokerTradesRequest struct {
	// Currency is left out to return broker block trades in every currency
	Currency currency.Code `json:"currency,omitzero"`
	// Count is the number of broker block trades to return, 20 by default and up to 1000
	Count uint64 `json:"count,omitempty"`
	// StartID returns block trades older than it, and EndID, which needs StartID, is the oldest block trade returned
	StartID string `json:"start_id,omitempty"`
	EndID   string `json:"end_id,omitempty"`
}

// BrokerTradesResponse contains a page of broker block trades
type BrokerTradesResponse struct {
	History []BrokerTrade `json:"history"`
	// NextStartID is the StartID of the next page, and zero on the last
	NextStartID uint64 `json:"next_start_id"`
}

// BrokerTrade contains a broker block trade, its legs' trades and its parties
type BrokerTrade struct {
	ID        string              `json:"id"`
	Timestamp types.Time          `json:"timestamp"`
	Trades    []UserTradeResponse `json:"trades"`
	Maker     BrokerTradeParty    `json:"maker"`
	Taker     BrokerTradeParty    `json:"taker"`
}

// BlockTradeSignatureRequest contains the signature of a private/invalidate_block_trade_signature request
type BlockTradeSignatureRequest struct {
	Signature string `json:"signature"`
}

// SimulateBlockTradeRequest checks a block trade could be executed
type SimulateBlockTradeRequest struct {
	// Role is maker or taker, and may be left out
	Role   string          `json:"role,omitempty"`
	Trades []BlockTradeLeg `json:"trades"`
}

// VerifyBlockTradeRequest verifies a block trade and signs it for the counterparty to execute
type VerifyBlockTradeRequest struct {
	// Timestamp and Nonce are shared with the counterparty
	Timestamp time.Time `json:"-"`
	Nonce     string    `json:"nonce"`
	// Role is maker or taker, the opposite of the counterparty's
	Role   string          `json:"role"`
	Trades []BlockTradeLeg `json:"trades"`
}

// VerifyBlockTradeParams contains a VerifyBlockTradeRequest's params, with its timestamp in Unix milliseconds
type VerifyBlockTradeParams struct {
	*VerifyBlockTradeRequest
	Timestamp uint64 `json:"timestamp"`
}

// VerifyBlockTradeResponse contains a block trade's signature, which the counterparty executes it with; it is valid for
// 5 minutes around the block trade's timestamp
type VerifyBlockTradeResponse struct {
	Signature string `json:"signature"`
}
