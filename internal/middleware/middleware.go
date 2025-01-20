package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

func Conveyor(h http.Handler, sugar *zap.SugaredLogger, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h, sugar)
	}
	return h
}
