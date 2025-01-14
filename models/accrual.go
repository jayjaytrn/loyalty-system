package models

type AccrualOrderStatus string

func (s AccrualOrderStatus) String() string {
	return string(s)
}

const (
	AccrualOrderRegistered AccrualOrderStatus = "REGISTERED"
	AccrualOrderProcessing AccrualOrderStatus = "PROCESSING"
	AccrualOrderInvalid    AccrualOrderStatus = "INVALID"
	AccrualOrderProcessed  AccrualOrderStatus = "PROCESSED"
)

type OrderToAccrual struct {
	UUID        string `json:"uuid"`
	OrderNumber string `json:"number"`
}

type AccrualResponse struct {
	Order   string             `json:"order"`
	Status  AccrualOrderStatus `json:"status"`
	Accrual float64            `json:"accrual"`
}
