package slack

// WebsocketResponse holds websocket response data
type WebsocketResponse struct {
	Type    string `json:"type"`
	ReplyTo int    `json:"reply_to"`
	Error   struct {
		Msg  string `json:"msg"`
		Code int    `json:"code"`
	} `json:"error"`
}

// SendMessage holds details for message information
type SendMessage struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// Message is a response type handling message data
type Message struct {
	Channel    string  `json:"channel"`
	User       string  `json:"user"`
	Text       string  `json:"text"`
	SourceTeam string  `json:"source_team"`
	Timestamp  float64 `json:"ts,string"`
	Team       string  `json:"team"`
}

// PresenceChange holds user presence data
type PresenceChange struct {
	Presence string `json:"presence"`
	User     string `json:"user"`
}

// Response is a generalised response type
type Response struct {
	Channels []struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		NameNormalized string   `json:"name_normalized"`
		PreviousNames  []string `json:"previous_names"`
	} `json:"channels"`
	Groups []struct {
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Members []string `json:"members"`
	} `json:"groups"`
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	Self  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"self"`
	Team struct {
		Domain string `json:"domain"`
		ID     string `json:"id"`
		Name   string `json:"name"`
	} `json:"team"`
	URL   string `json:"url"`
	Users []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		TeamID string `json:"team_id"`
	} `json:"users"`
}
