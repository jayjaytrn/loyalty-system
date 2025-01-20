package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jayjaytrn/loyalty-system/internal/accrual"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/internal/handlers"
	"github.com/jayjaytrn/loyalty-system/logging"
	"github.com/jayjaytrn/loyalty-system/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	credentials := models.Credentials{
		Login:    "newuser",
		Password: "password123",
	}

	body, err := json.Marshal(credentials)
	if err != nil {
		t.Fatalf("Ошибка маршалинга: %v", err)
	}

	req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}

	mock.ExpectExec(`INSERT INTO users \(uuid, login, password\)`).
		WithArgs(sqlmock.AnyArg(), "newuser", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ожидался статус код %d, получен %d", http.StatusOK, rr.Code)
	}

	authHeader := rr.Header().Get("Authorization")
	if authHeader == "" {
		t.Fatalf("Ожидался заголовок Authorization, но его нет")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Fatalf("Ожидался токен в формате Bearer, получен: %s", authHeader)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Не все ожидания для мока были выполнены: %v", err)
	}
}

func TestLogin(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	t.Run("SuccessLogin", func(t *testing.T) {
		credentials := models.Credentials{
			Login:    "existinguser",
			Password: "password123",
		}

		body, err := json.Marshal(credentials)
		if err != nil {
			t.Fatalf("Error marshalling credentials: %v", err)
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		mock.ExpectQuery(`SELECT uuid, login, password FROM users WHERE login = \$1`).
			WithArgs("existinguser").
			WillReturnRows(sqlmock.NewRows([]string{"uuid", "login", "password"}).
				AddRow("user-uuid", "existinguser", string(hashedPassword)))

		req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Error creating request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.Login(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		authHeader := rr.Header().Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Expected Authorization header, but it is missing")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Fatalf("Expected token in Bearer format, got: %s", authHeader)
		}

		if err = mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("Not all mock expectations were met: %v", err)
		}
	})

	t.Run("LoginDoesNotExist", func(t *testing.T) {
		credentials := models.Credentials{
			Login:    "nonexistentuser",
			Password: "password123",
		}

		body, err := json.Marshal(credentials)
		if err != nil {
			t.Fatalf("Error marshalling credentials: %v", err)
		}

		mock.ExpectQuery(`SELECT uuid, login, password FROM users WHERE login = \$1`).
			WithArgs("nonexistentuser").
			WillReturnError(fmt.Errorf("no rows in result set"))

		req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Error creating request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.Login(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}

func TestOrders(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}
	logger := zap.NewNop().Sugar()

	accrualChan := make(chan models.OrderToAccrual, 1)
	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logger,
		AccrualManager: &accrual.Manager{
			Orders: accrualChan,
		},
	}

	t.Run("WrongContentType", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", nil)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "wrong content type")
	})

	t.Run("MissingUUID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", nil)
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "user UUID not found")
	})

	t.Run("InvalidOrderNumber", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", bytes.NewBufferString("invalid-order"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("UUID", "user-uuid")
		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid order number")
	})

	t.Run("OrderAlreadyExistsSameUser", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", bytes.NewBufferString("4677951650035254"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("UUID", "user-uuid")

		mock.ExpectQuery(`SELECT uuid, order_number, order_status, accrual, uploaded_at FROM orders WHERE order_number = \$1`).
			WithArgs("4677951650035254").
			WillReturnRows(sqlmock.NewRows([]string{"order_number", "uuid", "order_status", "accrual", "uploaded_at"}).
				AddRow("user-uuid", "4677951650035254", models.OrderRegistered, 0, time.Now()))
		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "order already exists")
	})

	t.Run("OrderAlreadyExistsAnotherUser", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", bytes.NewBufferString("4677951650035254"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("UUID", "another-uuid")

		mock.ExpectQuery(`SELECT uuid, order_number, order_status, accrual, uploaded_at FROM orders WHERE order_number = \$1`).
			WithArgs("4677951650035254").
			WillReturnRows(sqlmock.NewRows([]string{"order_number", "uuid", "order_status", "accrual", "uploaded_at"}).
				AddRow("user-uuid", "4677951650035254", models.OrderRegistered, 0, time.Now()))

		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "order from another user already exists")
	})

	t.Run("SuccessfulOrderRegistration", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/orders", bytes.NewBufferString("4677951650035254"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("UUID", "user-uuid")

		mock.ExpectQuery(`SELECT uuid, order_number, order_status, accrual, uploaded_at FROM orders WHERE order_number = \$1`).
			WithArgs("4677951650035254").
			WillReturnRows(sqlmock.NewRows([]string{"order_number", "uuid", "order_status"}))

		mock.ExpectExec(`INSERT INTO orders \(uuid, order_number, order_status\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs("user-uuid", "4677951650035254", models.OrderRegistered).
			WillReturnResult(sqlmock.NewResult(1, 1))

		rec := httptest.NewRecorder()

		handler.Orders(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)

		select {
		case order := <-accrualChan:
			assert.Equal(t, "4677951650035254", order.OrderNumber)
			assert.Equal(t, "user-uuid", order.UUID)
		default:
			t.Fatal("Expected order to be sent to accrual channel")
		}
	})
}

func TestOrdersGet(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	// Create the manager with the mock DB
	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	UUID := "user-uuid"
	uploadedAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT order_number, order_status, accrual, uploaded_at FROM orders WHERE uuid = \$1 ORDER BY uploaded_at DESC`).
		WithArgs(UUID).
		WillReturnRows(sqlmock.NewRows([]string{"order_number", "order_status", "accrual", "uploaded_at"}).
			AddRow("4677951650035254", "OrderRegistered", 100.0, uploadedAt))

	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("UUID", UUID)

	rr := httptest.NewRecorder()
	handler.OrdersGet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expected := `[{"number":"4677951650035254","status":"OrderRegistered","accrual":100,"uploaded_at":"2025-01-01T12:00:00Z"}]`
	actual := rr.Body.String()

	if strings.TrimSpace(actual) != strings.TrimSpace(expected) {
		t.Fatalf("expected body %s, got %s", expected, actual)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("not all expectations were met: %v", err)
	}
}

func TestBalance(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	UUID := "user-uuid"
	mock.ExpectQuery(`SELECT current, withdrawn FROM balances WHERE uuid = \$1`).
		WithArgs(UUID).
		WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(100, 50)) // Provide the expected balance values

	req, err := http.NewRequest("GET", "/balance", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("UUID", UUID)

	rr := httptest.NewRecorder()
	handler.Balance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expected := `{"current":100,"withdrawn":50}`
	actual := rr.Body.String()

	if strings.TrimSpace(actual) != strings.TrimSpace(expected) {
		t.Fatalf("expected body %s, got %s", expected, actual)
	}
}

func TestWithdraw(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	UUID := "user-uuid"
	withdrawRequest := models.WithdrawRequest{
		OrderNumber: "4677951650035254",
		Sum:         50,
	}

	body, err := json.Marshal(withdrawRequest)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	mock.ExpectQuery(`SELECT current, withdrawn FROM balances WHERE uuid = \$1`).
		WithArgs(UUID).
		WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).
			AddRow(100, 50))

	mock.ExpectExec(`INSERT INTO withdrawals \(uuid, order_number, sum\)`).
		WithArgs(UUID, withdrawRequest.OrderNumber, withdrawRequest.Sum).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO balances \(uuid, current, withdrawn\)`).
		WithArgs(UUID, -withdrawRequest.Sum, withdrawRequest.Sum).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, err := http.NewRequest("POST", "/withdraw", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("UUID", UUID)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Withdraw(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("not all expectations were met: %v", err)
	}
}

func TestWithdrawals(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockdb.Close()

	manager := db.Manager{Db: mockdb}

	handler := &handlers.Handler{
		Database: &manager,
		Logger:   logging.GetSugaredLogger(),
	}

	UUID := "user-uuid"
	mock.ExpectQuery(`SELECT order_number, sum, processed_at FROM withdrawals WHERE uuid = \$1 ORDER BY processed_at DESC`).
		WithArgs(UUID).
		WillReturnRows(sqlmock.NewRows([]string{"order_number", "sum", "processed_at"}).
			AddRow("4677951650035254", 50, "2025-01-01T12:00:00Z"))

	req, err := http.NewRequest("GET", "/withdrawals", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("UUID", UUID)

	rr := httptest.NewRecorder()
	handler.Withdrawals(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expected := `[{"order":"4677951650035254","sum":50,"processed_at":"2025-01-01T12:00:00Z"}]`
	actual := rr.Body.String()

	if strings.TrimSpace(actual) != strings.TrimSpace(expected) {
		t.Fatalf("expected body %s, got %s", expected, actual)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("not all expectations were met: %v", err)
	}
}
