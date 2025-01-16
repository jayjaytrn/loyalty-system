package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"github.com/jayjaytrn/loyalty-system/logging"
	"time"
)

type Config struct {
	RunAddress                   string        `env:"RUN_ADDRESS,required"`
	DatabaseURI                  string        `env:"DATABASE_URI,required"`
	AccrualSystemAddress         string        `env:"ACCRUAL_SYSTEM_ADDRESS,required"`
	AccrualRequestTimeoutSeconds time.Duration `env:"ACCRUAL_REQUEST_TIMEOUT"`
}

func GetConfig() *Config {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	config := &Config{}

	flag.StringVar(&config.RunAddress, "a", "localhost:8080", "RunAddress")
	flag.StringVar(&config.DatabaseURI, "d", "postgres://admin:admin@localhost:5432/test", "DatabaseURI")
	flag.StringVar(&config.AccrualSystemAddress, "r", "test", "AccrualSystemAddress")
	flag.DurationVar(&config.AccrualRequestTimeoutSeconds, "t", 5, "AccrualRequestTimeoutSeconds")
	flag.Parse()

	err := env.Parse(config)
	if err != nil {
		logger.Debug("failed to parse environment variables:", err)
	}

	return config
}
