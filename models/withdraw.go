package models

type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float32 `json:"sum"`
}

type WithdrawalsResponse struct {
	UUID        string  `json:"uuid,omitempty"`
	OrderNumber string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
