package database

import (
	"log"
	"testing"
)

var o ORM
var connected bool

func TestConnectdb(t *testing.T) {
	err := o.InstantiateConn()
	if err != nil {
		log.Println("WARNING NO CONNECTION TO DATABASE!")
	}
	connected = true
}

func TestStartDB(t *testing.T) {
	if connected {
		_, err := StartDB()
		if err != nil {
			t.Error("test failed - Database StartDB() error", err)
		}
	}
}

func TestInsertGCTUser(t *testing.T) {
	if connected {
		_, err := o.InsertGCTUser("test", "test123")
		if err != nil {
			t.Error("test failed - Database InsertGCTUser() error", err)
		}
	}
}

func TestCheckGCTUserPassword(t *testing.T) {
	if connected {
		b, _ := o.CheckGCTUserPassword("test", "test123")
		if !b {
			t.Error("test failed - Database CheckGCTUserPassword() error")
		}
		b, _ = o.CheckGCTUserPassword("bra", "boy")
		if b {
			t.Error("test failed - Database CheckGCTUserPassword() error")
		}
	}
}

func TestChangeGCTUserPassword(t *testing.T) {
	if connected {
		err := o.ChangeGCTUserPassword("test", "ching chong")
		if err != nil {
			t.Error("test failed - Database ChangeGCTUserPassword() error", err)
		}
	}
}

func TestDeleteGCTUser(t *testing.T) {
	if connected {
		err := o.DeleteGCTUser("0")
		if err != nil {
			t.Error("test failed - Database DeleteGCTUser() error", err)
		}
	}
}
