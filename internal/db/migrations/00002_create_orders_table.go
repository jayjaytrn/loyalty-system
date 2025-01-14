package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(UpOrdersTable, DownOrdersTable)
}

func UpOrdersTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE orders
(
    uuid UUID NOT NULL,
    order_number VARCHAR(255) PRIMARY KEY,
    order_status VARCHAR(255) NOT NULL,
    accrual INT DEFAULT 0,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);`)
	return err
}

func DownOrdersTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE orders;")
	return err
}
