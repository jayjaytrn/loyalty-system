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
	"sync"
	"time"
)

type Manager struct {
	Database    *db.Manager
	Orders      chan models.OrderToAccrual
	NeedToSleep chan bool
	Config      *config.Config
	Logger      *zap.SugaredLogger
}

func NewManager(orders chan models.OrderToAccrual, database *db.Manager, config *config.Config, logger *zap.SugaredLogger) *Manager {
	return &Manager{
		Orders:      orders,
		Database:    database,
		NeedToSleep: make(chan bool, 1),
		Config:      config,
		Logger:      logger,
	}
}

func (m *Manager) StartOrderProcessing(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < m.Config.WorkerCount; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			m.processOrders(ctx)
		}(i)
	}
	wg.Wait()
}

func (m *Manager) processOrders(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.Logger.Info("context done")
			return
		case <-m.NeedToSleep:
			m.Logger.Info("the number of requests to the accrual service has been exceeded; timeout")
			time.Sleep(m.Config.AccrualRequestTimeout)
			m.NeedToSleep <- false
		case order, ok := <-m.Orders:
			if !ok {
				m.Logger.Info("order channel closed")
				return
			}
			orderInfo, err := m.getOrderInfo(order.OrderNumber)
			if err != nil {
				m.Logger.Error("failed to get order info", zap.Error(err))
				continue
			}
			if orderInfo == nil {
				m.Logger.Info("order info is nil, mark it as invalid")
				_ = m.updateOrder(&models.AccrualResponse{
					Status:  models.AccrualOrderInvalid,
					Order:   order.OrderNumber,
					Accrual: 0,
				})
				continue
			}
			if orderInfo.Status != models.AccrualOrderRegistered {
				err = m.updateOrder(orderInfo)
				if err != nil {
					m.Logger.Error("failed to update order info", zap.Error(err))
					continue
				}
			}
			if orderInfo.Accrual != 0 {
				withdrawn := 0
				m.updateBalance(order.UUID, orderInfo.Accrual, float32(withdrawn))
			}
		}
	}
}

func (m *Manager) updateOrder(accrualResponse *models.AccrualResponse) error {
	err := m.Database.UpdateOrder(accrualResponse)
	if err != nil {
		m.Logger.Error("failed to update order", zap.Error(err))
		return err
	}
	return nil
}

func (m *Manager) updateBalance(UUID string, accrual float32, withdraw float32) {
	err := m.Database.UpdateBalance(UUID, accrual, withdraw)
	if err != nil {
		m.Logger.Error("failed to update balance", zap.Error(err))
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

	if resp.StatusCode == http.StatusTooManyRequests {
		select {
		case m.NeedToSleep <- true:
		default:
		}
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accrualResp models.AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &accrualResp, nil
}

func (m *Manager) processUnprocessedOrders() error {
	m.Logger.Info("processing unprocessed orders...")

	unprocessedOrders, err := m.Database.GetUnprocessedOrders()
	if err != nil {
		return fmt.Errorf("failed to get unprocessed orders: %v", err)
	}

	var ordersToAccrual []models.OrderToAccrual

	for _, u := range unprocessedOrders {
		ordersToAccrual = append(ordersToAccrual, models.OrderToAccrual{
			OrderNumber: u.OrderNumber,
			UUID:        u.UUID,
		})
	}

	for _, order := range ordersToAccrual {
		select {
		case m.Orders <- order:
			m.Logger.Info("added unprocessed order to channel", zap.String("order", order.OrderNumber))
		default:
		}
	}

	return nil
}

func (m *Manager) HandleUnprocessedOrders(ctx context.Context) {
	ticker := time.NewTicker(m.Config.RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.Logger.Info("context done")
			return
		case <-m.NeedToSleep:
			m.Logger.Info("need to sleep")
			time.Sleep(m.Config.AccrualRequestTimeout)
			m.NeedToSleep <- false
		case <-ticker.C:
			m.Logger.Info("checking unprocessed orders")

			err := m.processUnprocessedOrders()
			if err != nil {
				m.Logger.Error("failed to process unprocessed orders", zap.Error(err))
			}
		}
	}
}
