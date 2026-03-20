package database

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

type DB struct {
	*sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db %s: %w", path, err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS synced_emails (
			account_name TEXT,
			pop3_uid TEXT,
			PRIMARY KEY (account_name, pop3_uid)
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("ensure table: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) IsSynced(accountName, uid string) (bool, error) {
	var n int
	err := db.QueryRow("SELECT 1 FROM synced_emails WHERE account_name = ? AND pop3_uid = ?", accountName, uid).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query isSynced: %w", err)
	}
	return n == 1, nil
}

func (db *DB) MarkSynced(accountName, uid string) error {
	_, err := db.Exec("INSERT OR IGNORE INTO synced_emails (account_name, pop3_uid) VALUES (?, ?)", accountName, uid)
	if err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}
	return nil
}
