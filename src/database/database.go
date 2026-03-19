package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS synced_emails (
			account_name TEXT,
			pop3_uid TEXT,
			PRIMARY KEY (account_name, pop3_uid)
		);
	`)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) IsSynced(accountName, uid string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT 1 FROM synced_emails WHERE account_name = ? AND pop3_uid = ?", accountName, uid).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

func (db *DB) MarkSynced(accountName, uid string) error {
	_, err := db.Exec("INSERT INTO synced_emails (account_name, pop3_uid) VALUES (?, ?)", accountName, uid)
	return err
}