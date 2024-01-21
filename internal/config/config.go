// Модуль конфигурации приложения
package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// ServerConfig описывает структуру конфигурации приложения
type ServerConfig struct {
	FlagRunAddr     string `env:"SERVER_ADDRESS"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"`
	RedirectBaseURL string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Seed            string `env:"SEED"`
}

var serverConfig ServerConfig

// ServerConfig парсит значения из переменных окружения
func ParseFlags() (*ServerConfig, error) {
	flag.StringVar(&serverConfig.FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.BoolVar(&serverConfig.EnableHTTPS, "s", false, "enable https")
	flag.StringVar(&serverConfig.RedirectBaseURL, "b", "http://localhost:8080", "server URI prefix")
	flag.StringVar(&serverConfig.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&serverConfig.DatabaseDSN, "d", "", "Data Source Name (DSN)")
	flag.StringVar(&serverConfig.Seed, "c", "b4952c3809196592c026529df00774e46bfb5be0", "seed")
	flag.Parse()

	return &serverConfig, env.Parse(&serverConfig)
}
