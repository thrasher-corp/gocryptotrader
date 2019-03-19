package database

import "testing"

func TestGetSQLite3Instance(t *testing.T) {
	db := GetSQLite3Instance()
	c := db.IsConnected()
	if c {
		t.Error("Test Failed - SQLite3 instance error")
	}
}

func TestGetPostgresInstance(t *testing.T) {
	db := GetPostgresInstance()
	c := db.IsConnected()
	if c {
		t.Error("Test Failed - PostgreSQL instance error")
	}
}
