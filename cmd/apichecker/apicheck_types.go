package main

// ShaResponse stores raw response from the queries
type ShaResponse struct {
	ShaResp string `json:"sha"`
}

// ExchangeInfo stores exchange info
type ExchangeInfo struct {
	Name      string
	CheckType string
	Data      *CheckData `json:",omitempty"`
	Disabled  bool
}

// CheckData is the necessary data required for checking updates
type CheckData struct {
	HTMLData   *HTMLScrapingData `json:",omitempty"`
	GitHubData *GithubData       `json:",omitempty"`
}

// HTMLScrapingData stores input required for extracting latest update data using HTML
type HTMLScrapingData struct {
	TokenData     string `json:",omitempty"`
	Key           string `json:",omitempty"`
	Val           string `json:",omitempty"`
	TokenDataEnd  string `json:",omitempty"`
	TextTokenData string `json:",omitempty"`
	DateFormat    string `json:",omitempty"`
	RegExp        string `json:",omitempty"`
	CheckString   string `json:",omitempty"`
	Path          string `json:",omitempty"`
}

// GithubData stores input required for extracting latest update data
type GithubData struct {
	Repo string `json:",omitempty"`
	Sha  string `json:",omitempty"`
}

// ListData stores trello lists' required data
type ListData struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IDBoard string `json:"idBoard"`
}

// CardFill contains data necessary to create a new card
type CardFill struct {
	Name      string
	Desc      string
	Pos       string
	Due       string
	ListID    string
	MembersID string
	LabelsID  string
	URLSource string
}

// ItemData stores data of items on a given checklist
type ItemData struct {
	State    string `json:"state"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int64  `json:"pos"`
}

// ChecklistItemData stores items on a given checklist
type ChecklistItemData struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CheckItems []ItemData `json:"checkItems"`
}

// MembersData saves member's data which includes the boards accessible
type MembersData struct {
	Name    string `json:"name"`
	ShortID string `json:"shortlink"`
	ID      string `json:"id"`
}

// Config is a format for storing update data
type Config struct {
	CardID              string `json:"CardID"`
	ChecklistID         string `json:"ChecklistID"`
	ListID              string `json:"ListID"`
	BoardID             string `json:"BoardID"`
	Key                 string `json:"Key"`
	Token               string `json:"Token"`
	CreateCardName      string
	CreateListName      string
	CreateChecklistName string
	Exchanges           []ExchangeInfo `json:"Exchanges"`
}

// TrelloData stores data on a given item (board, list, card)
type TrelloData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
