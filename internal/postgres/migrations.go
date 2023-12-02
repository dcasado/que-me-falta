package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
)

var schemaVersion = len(migrations)

var migrations = []func(tx *sql.Tx) error{
	func(tx *sql.Tx) (err error) {
		sql := `
			CREATE TABLE schema_version (
				version VARCHAR(3) NOT NULL
			);

			CREATE TABLE sessions (
				token                VARCHAR(255) PRIMARY KEY,
				expiration_timestamp TIMESTAMP NOT NULL
			);

			CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

			CREATE TABLE products (
				id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
				name        VARCHAR(255) NOT NULL,
				description VARCHAR(255),
				quantity    VARCHAR(255),
				added       BOOLEAN NOT NULL DEFAULT true
			);
		`
		_, err = tx.Exec(sql)
		return err
	},
}

func Migrate(db *sql.DB) error {
	var currentVersion int
	err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&currentVersion)
	if err != nil {
		log.Fatalf("error retrieving schema version for the migrations: %v", err)
	}

	for version := currentVersion; version < schemaVersion; version++ {
		newVersion := version + 1

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error starting transaction for migration %d: %v", newVersion, err)
		}

		if err := migrations[version](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing the migration %d: %v", newVersion, err)
		}

		if _, err := tx.Exec(`DELETE FROM schema_version`); err != nil {
			tx.Rollback()
			return fmt.Errorf("error deleting the version from schema_version table: %v", err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES ($1)`, strconv.Itoa(newVersion)); err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting the new schema version %d: %v", newVersion, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error commiting the transaction for the migration %d: %v", newVersion, err)
		}
	}

	return nil
}
