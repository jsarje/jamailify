package database

import (
	"database/sql"
	"fmt"
	"strings"

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

	if err := ensureSyncedEmailsSchema(db); err != nil {
		return nil, fmt.Errorf("ensure table: %w", err)
	}

	return &DB{db}, nil
}

func ensureSyncedEmailsSchema(db *sql.DB) error {
	columns, err := syncedEmailsColumns(db)
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return createSyncedEmailsTable(db)
	}
	if hasColumn(columns, "message_uid") {
		return nil
	}
	if hasColumn(columns, "pop3_uid") {
		return migrateSyncedEmailsTable(db)
	}

	return fmt.Errorf("synced_emails has unsupported columns: %s", strings.Join(columns, ", "))
}

func syncedEmailsColumns(db *sql.DB) ([]string, error) {
	rows, err := db.Query("PRAGMA table_info(synced_emails)")
	if err != nil {
		return nil, fmt.Errorf("inspect synced_emails schema: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return nil, fmt.Errorf("scan synced_emails schema: %w", err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate synced_emails schema: %w", err)
	}
	return columns, nil
}

func hasColumn(columns []string, name string) bool {
	for _, column := range columns {
		if column == name {
			return true
		}
	}
	return false
}

func createSyncedEmailsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS synced_emails (
			account_name TEXT,
			message_uid TEXT,
			PRIMARY KEY (account_name, message_uid)
		);
	`)
	if err != nil {
		return fmt.Errorf("create synced_emails table: %w", err)
	}
	return nil
}

func migrateSyncedEmailsTable(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin synced_emails migration: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		"ALTER TABLE synced_emails RENAME TO synced_emails_legacy",
		`CREATE TABLE synced_emails (
			account_name TEXT,
			message_uid TEXT,
			PRIMARY KEY (account_name, message_uid)
		)`,
		"INSERT INTO synced_emails (account_name, message_uid) SELECT account_name, pop3_uid FROM synced_emails_legacy",
		"DROP TABLE synced_emails_legacy",
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("migrate synced_emails table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit synced_emails migration: %w", err)
	}
	return nil
}

func (db *DB) IsSynced(accountName, uid string) (bool, error) {
	var n int
	err := db.QueryRow("SELECT 1 FROM synced_emails WHERE account_name = ? AND message_uid = ?", accountName, uid).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query isSynced: %w", err)
	}
	return n == 1, nil
}

func (db *DB) MarkSynced(accountName, uid string) error {
	_, err := db.Exec("INSERT OR IGNORE INTO synced_emails (account_name, message_uid) VALUES (?, ?)", accountName, uid)
	if err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}
	return nil
}
