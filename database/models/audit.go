package models

// Event is a model of how the data is represented in a database
type Event struct {
	Type       string
	Identifier string
	Message    string
}
