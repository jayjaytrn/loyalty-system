package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(UpUsersTable, DownUsersTable)
}

func UpUsersTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE users
(
    uuid UUID PRIMARY KEY,
    login varchar(255) NOT NULL UNIQUE,
    password varchar(255) NOT NULL
);`)
	return err
}

func DownUsersTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE users;")
	return err
}
