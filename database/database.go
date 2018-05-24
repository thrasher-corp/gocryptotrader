package database

//go:generate sqlboiler postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/thrasher-/gocryptotrader/database/models"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// ORM is the overarching type across the database package that handles database
// connections and relational mapping
type ORM struct {
	Exec *sql.DB
}

// StartDB makes a connection to the database and returns a pointer
func StartDB() (*ORM, error) {
	db := ORM{}
	return &db, db.InstantiateConn()
}

// InstantiateConn starts the connection to the GoCryptoTrader database
func (o *ORM) InstantiateConn() error {
	db, err := sql.Open(
		"postgres",
		`dbname=gocryptotrader host=localhost user=gocryptotrader password=lol123`)
	if err != nil {
		return err
	}
	o.Exec = db

	return nil
}

// InsertGCTUser inserts a new user with password and returns its ID
func (o *ORM) InsertGCTUser(name, password string) (int, error) {
	u := &models.GCTUser{
		Name:     name,
		Password: password,
	}

	return u.ID, u.Insert(o.Exec)
}

// CheckGCTUserPassword is used to match username and password
func (o *ORM) CheckGCTUserPassword(username, password string) (bool, error) {
	model, err := models.GCTUsers(o.Exec, qm.Where("name = ?", username)).One()
	if err != nil {
		return false, err
	}

	if password != model.Password {
		return false, fmt.Errorf("incorrect user password")
	}

	return true, nil
}

// ChangeGCTUserPassword inserts a new password
func (o *ORM) ChangeGCTUserPassword(username, newPassword string) error {
	model, err := models.GCTUsers(o.Exec, qm.Where("name = ?", username)).One()
	if err != nil {
		return err
	}

	model.Password = newPassword

	return model.Update(o.Exec)
}

// DeleteGCTUser deletes a user by ID and returns an error
func (o *ORM) DeleteGCTUser(ID string) error {
	model, err := models.GCTUsers(o.Exec, qm.Where("id = ?", ID)).One()
	if err != nil {
		return err
	}

	return model.Delete(o.Exec)
}
