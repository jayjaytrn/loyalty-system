package logging

import "go.uber.org/zap"

func GetSugaredLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("cannot initialize zap")
	}
	sl := logger.Sugar()

	return sl
}
