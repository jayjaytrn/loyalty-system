package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jayjaytrn/loyalty-system/models"
	"go.uber.org/zap"
	"io"
	"net/http"
)

func ValidateCredentialsAndHashLogin(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			sugar.Error("wrong content type: " + contentType)
			http.Error(w, "wrong content type", http.StatusBadRequest)
			return
		}

		var credentials models.User

		if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
			sugar.Error("error decoding credentials.", zap.Error(err))
			http.Error(w, "error decoding credentials", http.StatusBadRequest)
			return
		}

		if credentials.Login == "" || credentials.Password == "" {
			sugar.Error("login and password are required")
			http.Error(w, "login and password are required", http.StatusBadRequest)
			return
		}

		loginBytes := sha256.Sum256([]byte(credentials.Login))
		loginBytesHex := hex.EncodeToString(loginBytes[:])

		credentials.Login = loginBytesHex

		bodyBytes, err := json.Marshal(credentials)
		if err != nil {
			sugar.Info("error serializing credentials", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.ContentLength = int64(len(bodyBytes))

		h.ServeHTTP(w, r)
	})
}
