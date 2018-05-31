package smsglobal

// Contact struct stores information related to a SMSGlobal contact
type Contact struct {
	Name    string `json:"Name"`
	Number  string `json:"Number"`
	Enabled bool   `json:"Enabled"`
}
