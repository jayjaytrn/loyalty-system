package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jayjaytrn/loyalty-system/config"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/models"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Manager struct {
	Database *db.Manager
	Orders   chan models.OrderToAccrual
	Config   *config.Config
	Logger   *zap.SugaredLogger
}

func NewManager(orders chan models.OrderToAccrual, database *db.Manager, config *config.Config, logger *zap.SugaredLogger) *Manager {
	return &Manager{
		Orders:   orders,
		Database: database,
		Config:   config,
		Logger:   logger,
	}
}

func (m *Manager) GetOrderInfoAndUpdateBalances(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.Logger.Info("context done")
			return
		case order, ok := <-m.Orders:
			if !ok {
				m.Logger.Info("order channel closed")
				return
			}
			orderInfo, err := m.getOrderInfo(order.OrderNumber)
			if err != nil {
				fmt.Println(err)
			}
			if orderInfo == nil {
				continue
			}
			if orderInfo.Status != models.AccrualOrderRegistered {
				m.updateOrder(orderInfo)
			}
			if orderInfo.Accrual != 0 {
				withdrawn := 0
				m.updateBalance(order.UUID, orderInfo.Accrual, float32(withdrawn))
			}
		}
	}
}

func (m *Manager) updateOrder(accrualResponse *models.AccrualResponse) {
	err := m.Database.UpdateOrder(accrualResponse)
	if err != nil {
		m.Logger.Warn(err)
	}

}

func (m *Manager) updateBalance(UUID string, accrual float32, withdraw float32) {
	err := m.Database.UpdateBalance(UUID, accrual, withdraw)
	if err != nil {
		fmt.Println(err)
	}
}

func (m *Manager) getOrderInfo(orderNumber string) (*models.AccrualResponse, error) {
	m.Logger.Info(fmt.Sprintf("getting order info: %s/api/orders/%s", m.Config.AccrualSystemAddress, orderNumber))
	url := fmt.Sprintf("%s/api/orders/%s", m.Config.AccrualSystemAddress, orderNumber)

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accrualResp models.AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &accrualResp, nil
}
