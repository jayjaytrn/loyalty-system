package middleware

import (
	"go.uber.org/zap"
	"net/http"
)

func Conveyor(h http.Handler, sugar *zap.SugaredLogger, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h, sugar)
	}
	return h
}
