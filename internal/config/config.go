// Модуль конфигурации приложения
package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

// ServerConfig описывает структуру конфигурации приложения
type ServerConfig struct {
	FlagRunAddr     string `json:"server_address" env:"SERVER_ADDRESS"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"`
	ProfileMode     bool   `json:"profile_mode" env:"PROFILE_MODE"`
	RedirectBaseURL string `json:"base_url" env:"BASE_URL"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `json:"database_dsn" env:"DATABASE_DSN"`
	Secret          string `json:"-" env:"SECRET"`
	Config          string `json:"-" env:"CONFIG"`
}

var serverConfig ServerConfig

// ServerConfig парсит значения из переменных окружения
func ParseFlags() (*ServerConfig, error) {
	flag.StringVar(&serverConfig.FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.BoolVar(&serverConfig.EnableHTTPS, "s", false, "enable https")
	flag.BoolVar(&serverConfig.ProfileMode, "p", false, "register pprof profiler")
	flag.StringVar(&serverConfig.RedirectBaseURL, "b", "http://localhost:8080", "server URI prefix")
	flag.StringVar(&serverConfig.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&serverConfig.DatabaseDSN, "d", "", "Data Source Name (DSN)")
	flag.StringVar(&serverConfig.Secret, "s", "b4952c3809196592c026529df00774e46bfb5be0", "Secret")
	flag.StringVar(&serverConfig.Config, "c", "", "Config json file path")
	flag.Parse()

	return &serverConfig, env.Parse(&serverConfig)
}
