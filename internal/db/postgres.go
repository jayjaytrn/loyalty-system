package db

import (
	"database/sql"
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
        INSERT INTO gophermart (uuid, login, password)
        VALUES ($1, $2, $3)
    `, user.UUID, user.Login, user.Password)
	if err != nil {
		return fmt.Errorf("failed to insert user data: %v", err)
	}

	return nil
}

func (m *Manager) GetUserData(login string) (models.User, error) {
	var user models.User

	// Запрос для получения пользователя по логину
	err := m.db.QueryRow(`
		SELECT uuid, login, password 
		FROM gophermart 
		WHERE login = $1
	`, login).Scan(&user.UUID, &user.Login, &user.Password)

	if err != nil {
		return user, fmt.Errorf("failed to get user data: %v", err)
	}

	return user, nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}
