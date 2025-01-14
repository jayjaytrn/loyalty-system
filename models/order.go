package models

import (
	"time"
)

type OrderStatus string

// Возможные значения статусов
const (
	OrderRegistered OrderStatus = "NEW"
	OrderProcessing OrderStatus = "PROCESSING"
	OrderInvalid    OrderStatus = "INVALID"
	OrderProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	UUID        string      `json:"uuid"`
	OrderNumber string      `json:"number"`
	OrderStatus OrderStatus `json:"status"`
	Accrual     *uint64     `json:"accrual,omitempty"`
	UploadedAt  time.Time   `json:"uploaded_at"`
}
