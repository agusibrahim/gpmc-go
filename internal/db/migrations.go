package db

import (
	"database/sql"
)

const (
	// Current schema version
	schemaVersion = 2
)

// getSchemaVersion returns the current schema version from the database
func getSchemaVersion(db *sql.DB) int {
	var version int
	err := db.QueryRow("SELECT version FROM schema_version WHERE id = 1").Scan(&version)
	if err != nil {
		// Table doesn't exist or no row - assume version 1
		return 1
	}
	return version
}

// setSchemaVersion sets the schema version in the database
func setSchemaVersion(db *sql.DB, version int) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			version INTEGER
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT OR REPLACE INTO schema_version (id, version) VALUES (1, ?)
	`, version)
	return err
}

// migrate performs any necessary database migrations
func migrate(db *sql.DB) error {
	currentVersion := getSchemaVersion(db)

	if currentVersion < 2 {
		if err := migrateV1toV2(db); err != nil {
			return err
		}
	}

	if currentVersion < schemaVersion {
		return setSchemaVersion(db, schemaVersion)
	}

	return nil
}

// migrateV1toV2 migrates from schema version 1 to 2
// Renames state_token to sync_token and page_token to resume_token
func migrateV1toV2(db *sql.DB) error {
	// Check if state table exists
	var exists int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='state'
	`).Scan(&exists)
	if err != nil || exists == 0 {
		return nil // No state table yet, nothing to migrate
	}

	// Check if migration is needed (old column names exist)
	rows, err := db.Query("PRAGMA table_info(state)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasOldColumns := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString
		err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
		if err != nil {
			continue
		}
		if name == "state_token" || name == "page_token" {
			hasOldColumns = true
			break
		}
	}

	if !hasOldColumns {
		return nil // Already migrated
	}

	// Perform migration
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS state_new (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			sync_token TEXT,
			resume_token TEXT,
			init_complete INTEGER
		);

		INSERT OR REPLACE INTO state_new (id, sync_token, resume_token, init_complete)
		SELECT id, state_token, page_token, init_complete FROM state;

		DROP TABLE state;

		ALTER TABLE state_new RENAME TO state;
	`)

	return err
}
