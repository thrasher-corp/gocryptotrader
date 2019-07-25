package models

type Event struct {
	ID      int64
	Type    string
	UserID  string
	Message string
}
