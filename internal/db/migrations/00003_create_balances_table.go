package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(UpBalancesTable, DownBalancesTable)
}

func UpBalancesTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE balances
(
    uuid UUID PRIMARY KEY,
    current INT DEFAULT 0,
    withdrawn INT DEFAULT 0
);`)
	return err
}

func DownBalancesTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE balances;")
	return err
}
