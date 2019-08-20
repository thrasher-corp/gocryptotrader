package drivers

// ConnectionDetails holds DSN information
type ConnectionDetails struct {
	Host     string
	Port     uint16
	Username string
	Password string
	Database string
	SSLMode  string
}
