package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *DB {
	db, err := NewDB(":memory:")
	require.NoError(t, err, "Failed to create in-memory DB")
	return db
}

func TestInitializeDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if the table was created
	_, err := db.Query("SELECT 1 FROM synced_emails LIMIT 1")
	assert.NoError(t, err, "Table 'synced_emails' should have been created")
}

func TestIsSynced(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	synced, err := db.IsSynced("account1", "uid1")
	assert.NoError(t, err)
	assert.False(t, synced, "UID should not be synced yet")
}

func TestMarkSyncedAndIsSynced(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Mark a UID as synced
	err := db.MarkSynced("account1", "uid1")
	assert.NoError(t, err)

	// Check if it's synced
	synced, err := db.IsSynced("account1", "uid1")
	assert.NoError(t, err)
	assert.True(t, synced, "UID should be marked as synced")
}

func TestDuplicateMarkSynced(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.MarkSynced("account1", "uid1")
	assert.NoError(t, err)

	// Try to mark the same UID again
	err = db.MarkSynced("account1", "uid1")
	assert.NoError(t, err, "Duplicate insert should be idempotent")

	// Ensure the UID is still marked as synced
	synced, err := db.IsSynced("account1", "uid1")
	assert.NoError(t, err)
	assert.True(t, synced, "UID should remain marked as synced")
}

func TestCrossAccountIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Mark a UID as synced for account1
	err := db.MarkSynced("account1", "uid1")
	assert.NoError(t, err)

	// Check if the same UID is synced for account2
	synced, err := db.IsSynced("account2", "uid1")
	assert.NoError(t, err)
	assert.False(t, synced, "UID should not be synced for a different account")
}

func TestLegacyPop3UIDSchemaMigration(t *testing.T) {
	legacyDB, err := NewDB(":memory:")
	require.NoError(t, err)

	_, err = legacyDB.Exec("DROP TABLE synced_emails")
	require.NoError(t, err)
	_, err = legacyDB.Exec(`
		CREATE TABLE synced_emails (
			account_name TEXT,
			pop3_uid TEXT,
			PRIMARY KEY (account_name, pop3_uid)
		)
	`)
	require.NoError(t, err)
	_, err = legacyDB.Exec("INSERT INTO synced_emails (account_name, pop3_uid) VALUES (?, ?)", "account1", "uid1")
	require.NoError(t, err)
	require.NoError(t, legacyDB.Close())

	migratedDB, err := NewDB(":memory:")
	require.NoError(t, err)
	defer migratedDB.Close()

	_, err = migratedDB.Exec("DROP TABLE synced_emails")
	require.NoError(t, err)
	_, err = migratedDB.Exec(`
		CREATE TABLE synced_emails (
			account_name TEXT,
			pop3_uid TEXT,
			PRIMARY KEY (account_name, pop3_uid)
		)
	`)
	require.NoError(t, err)
	_, err = migratedDB.Exec("INSERT INTO synced_emails (account_name, pop3_uid) VALUES (?, ?)", "account1", "uid1")
	require.NoError(t, err)

	require.NoError(t, ensureSyncedEmailsSchema(migratedDB.DB))

	synced, err := migratedDB.IsSynced("account1", "uid1")
	require.NoError(t, err)
	assert.True(t, synced)

	columns, err := syncedEmailsColumns(migratedDB.DB)
	require.NoError(t, err)
	assert.Contains(t, columns, "message_uid")
	assert.NotContains(t, columns, "pop3_uid")
}
