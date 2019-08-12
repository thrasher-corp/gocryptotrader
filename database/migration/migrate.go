package migrations

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (m *Migrator) LoadMigrations() error {
	migration, err := filepath.Glob("./database/migration/migrations/*.sql")

	if err != nil {
		return errors.New("failed to load migrations")
	}

	sort.Strings(migration)

	for x := range migration {
		err = m.LoadMigration(migration[x])
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) LoadMigration(migration string) error {
	file, err := os.Open(migration)
	if err != nil {
		fmt.Println(err)
	}
	fileData := strings.Split(file.Name(), "/")

	fileSeq := strings.Split(fileData[3], "_")
	seq, _ := strconv.Atoi(fileSeq[0])

	b, err := ioutil.ReadAll(file)

	up := bytes.Split(b, []byte("-- up"))
	down := bytes.Split(up[1], []byte("-- down"))

	temp := Migration{
		Sequence: seq,
		Name:     fileData[3],
		UpSQL:    down[0],
		DownSQL:  down[1],
	}

	m.Migrations = append(m.Migrations, temp)

	return nil
}

func (m *Migrator) RunMigration() (err error) {
	err = m.checkVersionTableExists()

	if err != nil {
		return
	}

	v, _ := m.GetCurrentVersion()

	latestSeq := m.Migrations[len(m.Migrations)-1].Sequence

	fmt.Printf("Current database version: %v\n", v)

	if v == latestSeq {
		fmt.Println("no migrations to be run")
		return
	}

	tx, err := m.Conn.SQL.Begin()

	if err != nil {
		return
	}

	for y := v; y < len(m.Migrations); y++ {
		err = m.txBegin(tx, m.Migrations[y].UpSQL)

		if err != nil {
			return tx.Rollback()
		}

		_, err = tx.Exec("update version set version=$1", m.Migrations[y].Sequence)
		if err != nil {
			return tx.Rollback()
		}
	}

	err = tx.Commit()

	if err != nil {
		return tx.Rollback()
	}

	fmt.Println("Migration completed ")
	return
}

func (m *Migrator) txBegin(tx *sql.Tx, input []byte) error {
	_, err := tx.Exec(fmt.Sprintf("%s", input))

	if err != nil {
		return tx.Rollback()
	}

	return nil
}

func (m *Migrator) GetCurrentVersion() (v int, err error) {
	err = m.Conn.SQL.QueryRow("select version from version").Scan(&v)
	return
}

func (m *Migrator) checkVersionTableExists() error {
	query := `
		CREATE TABLE IF NOT EXISTS version(
		    version int not null
		);

	insert into version select 0 where 0=(select count(*) from version);
`

	_, err := m.Conn.SQL.Exec(query)

	if err != nil {
		return err
	}

	return nil
}

func (m *Migrator) convertSQL() error {
	return nil
}