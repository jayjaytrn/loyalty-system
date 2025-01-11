package handlers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jayjaytrn/loyalty-system/config"
	"github.com/jayjaytrn/loyalty-system/internal/auth"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

type Handler struct {
	Database db.Manager
	Config   *config.Config
	Logger   *zap.SugaredLogger
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

	w.WriteHeader(http.StatusOK)
}
