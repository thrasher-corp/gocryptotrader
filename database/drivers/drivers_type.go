package drivers

// ConnectionDetails holds DSN information
type ConnectionDetails struct {
	Host     string `json:"host"`
	Port     uint32 `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}
