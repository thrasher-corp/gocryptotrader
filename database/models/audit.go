package models

// AuditEvent is a model of how the data is represented in a database
type AuditEvent struct {
	Type       string
	Identifier string
	Message    string
}
