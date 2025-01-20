package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(UpWithdrawalsTable, DownWithdrawalsTable)
}

func UpWithdrawalsTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE withdrawals
(
    uuid UUID NOT NULL,
    order_number VARCHAR(255) PRIMARY KEY,
    sum NUMERIC DEFAULT 0,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);`)
	return err
}

func DownWithdrawalsTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE withdrawals;")
	return err
}
