package middleware

import (
	"github.com/jayjaytrn/loyalty-system/internal/auth"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func ValidateAuth(h http.Handler, sugar *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		UUID, err := auth.ValidateJWT(tokenString)
		if err != nil {
			sugar.Errorw("Invalid token", "error", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("UUID", UUID)

		h.ServeHTTP(w, r)
	})
}
