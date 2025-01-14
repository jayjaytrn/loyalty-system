package db

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jayjaytrn/loyalty-system/config"
	_ "github.com/jayjaytrn/loyalty-system/internal/db/migrations"
	"github.com/jayjaytrn/loyalty-system/models"
	"github.com/pressly/goose/v3"
	"log"
)

type Manager struct {
	db *sql.DB
}

func NewManager(cfg *config.Config) (*Manager, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &Manager{
		db: db,
	}

	if err = goose.Up(db, "./internal/db/migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	return manager, nil
}

func (m *Manager) PutUniqueUserData(user models.User) error {
	_, err := m.db.Exec(`
        INSERT INTO users (uuid, login, password)
        VALUES ($1, $2, $3)
    `, user.UUID, user.Login, user.Password)
	if err != nil {
		return fmt.Errorf("failed to insert user data: %v", err)
	}

	return nil
}

func (m *Manager) GetUserData(login string) (models.User, error) {
	var user models.User

	err := m.db.QueryRow(`
		SELECT uuid, login, password 
		FROM users 
		WHERE login = $1
	`, login).Scan(&user.UUID, &user.Login, &user.Password)

	if err != nil {
		return user, fmt.Errorf("failed to get user data: %v", err)
	}

	return user, nil
}

func (m *Manager) PutOrder(order models.Order) error {
	_, err := m.db.Exec(`
        INSERT INTO orders (uuid, order_number, order_status)
        VALUES ($1, $2, $3)
    `, order.UUID, order.OrderNumber, order.OrderStatus)
	if err != nil {
		return fmt.Errorf("failed to insert user data: %v", err)
	}

	return nil
}

func (m *Manager) UpdateOrder(order *models.AccrualResponse) error {
	_, err := m.db.Exec(`
        UPDATE orders
        SET order_status = $1, accrual = $2
        WHERE order_number = $3
    `, order.Status, order.Accrual, order.Order)
	if err != nil {
		return fmt.Errorf("failed to update order: %v", err)
	}

	return nil
}

func (m *Manager) GetOrdersList(UUID string) ([]*models.Order, error) {
	var orders []*models.Order

	rows, err := m.db.Query(`
		SELECT uuid, order_number, order_status, accrual, uploaded_at
		FROM orders
		WHERE uuid = $1
		ORDER BY uploaded_at DESC
	`, UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.UUID, &order.OrderNumber, &order.OrderStatus, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		orders = append(orders, &order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during rows iteration: %v", err)
	}

	return orders, nil
}

func (m *Manager) GetOrderByOrderNumber(orderNumber string) (*models.Order, error) {
	var order models.Order

	err := m.db.QueryRow(`
		SELECT uuid, order_number, order_status, accrual, uploaded_at
		FROM orders
		WHERE order_number = $1
	`, orderNumber).Scan(&order.UUID, &order.OrderNumber, &order.OrderStatus, &order.Accrual, &order.UploadedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}

	return &order, nil
}

func (m *Manager) UpdateBalance(UUID string, accrual float64, withdrawn float64) error {
	_, err := m.db.Exec(`
		INSERT INTO balances (uuid, current, withdrawn)
		VALUES ($1, $2, $3)
		ON CONFLICT (uuid) DO UPDATE
		SET current = balances.current + $2,
		    withdrawn = balances.withdrawn + $3
	`, UUID, accrual, withdrawn)

	if err != nil {
		return fmt.Errorf("failed to update balance: %v", err)
	}

	return nil
}

func (m *Manager) PutWithdraw(UUID string, orderNumber string, sum float64) error {
	_, err := m.db.Exec(`
		INSERT INTO withdrawals (uuid, order_number, sum)
		VALUES ($1, $2, $3)
	`, UUID, orderNumber, sum)

	if err != nil {
		return fmt.Errorf("failed to update withdraw: %v", err)
	}

	return nil
}

func (m *Manager) GetWithdrawals(UUID string) ([]*models.WithdrawalsResponse, error) {
	rows, err := m.db.Query(`
		SELECT orderNumber, sum, processed_at
		FROM withdrawals
		WHERE uuid = $1
		ORDER BY processed_at DESC
	`, UUID)

	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %v", err)
	}
	defer rows.Close()

	var withdrawals []*models.WithdrawalsResponse
	for rows.Next() {
		var withdrawal models.WithdrawalsResponse
		if err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, fmt.Errorf("failed to scan withdrawal: %v", err)
		}
		withdrawals = append(withdrawals, &withdrawal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %v", err)
	}

	return withdrawals, nil
}

func (m *Manager) GetBalance(UUID string) (*models.Balance, error) {
	var balance models.Balance

	err := m.db.QueryRow(`
		SELECT current, withdrawn
		FROM balances
		WHERE uuid = $1
	`, UUID).Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}

	return &balance, nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}
