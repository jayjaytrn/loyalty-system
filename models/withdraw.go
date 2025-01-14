package models

type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
}

type WithdrawalsResponse struct {
	UUID        string  `json:"uuid"`
	OrderNumber string  `json:"orderNumber"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
