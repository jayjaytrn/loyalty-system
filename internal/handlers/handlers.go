package handlers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jayjaytrn/loyalty-system/config"
	"github.com/jayjaytrn/loyalty-system/internal/accrual"
	"github.com/jayjaytrn/loyalty-system/internal/auth"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	Database       *db.Manager
	Config         *config.Config
	Logger         *zap.SugaredLogger
	AccrualManager *accrual.Manager
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var credentials models.Credentials
	var userData models.User

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		h.Logger.Error("error reading decoded credentials", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(credentials.Password), 14)
	if err != nil {
		h.Logger.Info("password encryption error", zap.Error(err))
		http.Error(w, "internal error", http.StatusBadRequest)
		return
	}

	userData.Login = credentials.Login
	userData.Password = string(passwordBytes)
	userData.UUID = uuid.New().String()

	if err = h.Database.PutUniqueUserData(userData); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			h.Logger.Debug("duplicate key value violates unique constraint", zap.Error(err))
			http.Error(w, "login already exists", http.StatusConflict)
			return
		}
		h.Logger.Error("error when trying to put credentials to database", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := auth.BuildJWT(userData.UUID)
	if err != nil {
		h.Logger.Error("error building JWT", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials models.Credentials

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		h.Logger.Error("error reading decoded credentials", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	userData, err := h.Database.GetUserData(credentials.Login)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			h.Logger.Error("login does not exist", zap.Error(err))
			http.Error(w, "login does not exist", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(credentials.Password))
	if err != nil {
		h.Logger.Error("invalid login or password", zap.Error(err))
		http.Error(w, "invalid login or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.BuildJWT(userData.UUID)
	if err != nil {
		h.Logger.Error("error building JWT", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
		h.Logger.Error("wrong content type: " + contentType)
		http.Error(w, "wrong content type", http.StatusBadRequest)
		return
	}

	UUID := r.Header.Get("UUID")
	if UUID == "" {
		http.Error(w, "user UUID not found", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("failed to read body: " + err.Error())
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	orderNumber := string(body)
	if !ValidateOrderNumber(orderNumber) {
		h.Logger.Error("invalid order number: " + orderNumber)
		http.Error(w, "invalid order number: ", http.StatusUnprocessableEntity)
		return
	}

	order, err := h.Database.GetOrderByOrderNumber(orderNumber)
	if err != nil {
		h.Logger.Error("failed to get order: " + err.Error())
		http.Error(w, "failed to get order", http.StatusInternalServerError)
		return
	}

	if order != nil && order.UUID != "" {
		if order.UUID == UUID {
			h.Logger.Debug("order already exists")
			http.Error(w, "order already exists", http.StatusOK)
			return
		}
		h.Logger.Error("order from another user already exists")
		http.Error(w, "order from another user already exists", http.StatusConflict)
		return
	}

	newOrder := models.Order{
		OrderNumber: orderNumber,
		UUID:        UUID,
		OrderStatus: models.OrderRegistered,
	}

	err = h.Database.PutOrder(newOrder)
	if err != nil {
		h.Logger.Error("failed to put order: " + err.Error())
		http.Error(w, "failed to put order", http.StatusInternalServerError)
		return
	}

	toAccrual := models.OrderToAccrual{
		OrderNumber: orderNumber,
		UUID:        UUID,
	}
	h.AccrualManager.Orders <- toAccrual

	w.WriteHeader(http.StatusAccepted)
}

func ValidateOrderNumber(orderNumber string) bool {
	sum := 0
	alt := false

	for i := len(orderNumber) - 1; i >= 0; i-- {
		num, err := strconv.Atoi(string(orderNumber[i]))
		if err != nil {
			return false
		}

		if alt {
			num *= 2
			if num > 9 {
				num -= 9
			}
		}

		sum += num
		alt = !alt
	}

	return sum%10 == 0
}

func (h *Handler) OrdersGet(w http.ResponseWriter, r *http.Request) {
	UUID := r.Header.Get("UUID")
	if UUID == "" {
		http.Error(w, "user UUID not found", http.StatusUnauthorized)
		return
	}

	orders, err := h.Database.GetOrdersList(UUID)
	if err != nil {
		h.Logger.Error("failed to get orders: " + err.Error())
		http.Error(w, "failed to get orders", http.StatusInternalServerError)
		return
	}

	if orders == nil {
		h.Logger.Debug("no orders found")
		http.Error(w, "no orders found", http.StatusNoContent)
		return
	}

	if err = json.NewEncoder(w).Encode(orders); err != nil {
		h.Logger.Error("failed to encode orders to JSON: " + err.Error())
		http.Error(w, "failed to encode orders", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	UUID := r.Header.Get("UUID")
	if UUID == "" {
		http.Error(w, "user UUID not found", http.StatusUnauthorized)
		return
	}

	balance, err := h.Database.GetBalance(UUID)
	if err != nil {
		h.Logger.Error("failed to get balance: " + err.Error())
		http.Error(w, "failed to get balance", http.StatusInternalServerError)
		return
	}

	if balance == nil {
		balance = &models.Balance{
			Current:   0,
			Withdrawn: 0,
		}
	}
	if err = json.NewEncoder(w).Encode(balance); err != nil {
		http.Error(w, "failed to encode orders", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
