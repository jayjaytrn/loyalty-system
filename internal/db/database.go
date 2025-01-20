package db

import (
	"github.com/jayjaytrn/loyalty-system/models"
)

type Database interface {
	PutUniqueUserData(userData models.User) error
	GetUserData(login string) models.User

	PutOrder(order models.Order) error
	UpdateOrder(order models.AccrualResponse) error
	GetOrdersList(UUID string) ([]*models.Order, error)
	GetOrderByOrderNumber(orderNumber string) (*models.Order, error)

	UpdateBalance(UUID string, accrual float32) error
	GetBalance(UUID string) (models.Balance, error)

	PutWithdraw(UUID string, orderNumber string, sum float32) error
	GetWithdrawals() ([]*models.WithdrawalsResponse, error)

	GetUnprocessedOrders() ([]*models.Order, error)
	Close() error
}
