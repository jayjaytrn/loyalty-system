package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(UpGophermartTable, DownGophermartTable)
}

func UpGophermartTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE gophermart
(
    uuid UUID PRIMARY KEY,
    login varchar(255) NOT NULL UNIQUE,
    password varchar(255) NOT NULL
);`)
	return err
}

func DownGophermartTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE gophermart;")
	return err
}
