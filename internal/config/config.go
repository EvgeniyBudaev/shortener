// Модуль конфигурации приложения
package config

import (
	"bytes"
	"dario.cat/mergo"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"os"
)

// ServerConfig описывает структуру конфигурации приложения
type ServerConfig struct {
	FlagRunAddr     string `json:"server_address" env:"SERVER_ADDRESS"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"`
	RedirectBaseURL string `json:"base_url" env:"BASE_URL"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `json:"database_dsn" env:"DATABASE_DSN"`
	Seed            string `json:"-" env:"SEED"`
	Config          string `json:"-" env:"CONFIG"`
	TrustedSubnet   string `json:"trusted_subnet" env:"TRUSTED_SUBNET"`
	GRPCPort        string `json:"grpc_port" env:"GRPC_PORT"`
	ProfileMode     bool   `json:"profile_mode" env:"PROFILE_MODE"`
}

var serverConfig ServerConfig

// ServerConfig парсит значения из переменных окружения
func ParseFlags() (*ServerConfig, error) {
	flag.StringVar(&serverConfig.FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.BoolVar(&serverConfig.EnableHTTPS, "s", false, "enable https")
	flag.StringVar(&serverConfig.RedirectBaseURL, "b", "http://localhost:8080", "server URI prefix")
	flag.StringVar(&serverConfig.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&serverConfig.DatabaseDSN, "d", "", "Data Source Name (DSN)")
	flag.StringVar(&serverConfig.Seed, "e", "b4952c3809196592c026529df00774e46bfb5be0", "seed")
	flag.StringVar(&serverConfig.Config, "c", "", "Config json file path")
	flag.StringVar(&serverConfig.TrustedSubnet, "t", "", "trusted CIDR (ex. 192.168.0.0/24)")
	flag.StringVar(&serverConfig.GRPCPort, "grpc", "", "will add listener to port if specified")
	flag.BoolVar(&serverConfig.ProfileMode, "p", false, "register pprof profiler")
	flag.Parse()

	if serverConfig.Config != "" {
		data, err := os.ReadFile(serverConfig.Config)
		if err != nil {
			return nil, fmt.Errorf("error opening config file: %w", err)
		}
		var configFromFile ServerConfig
		if err := json.NewDecoder(bytes.NewReader(data)).Decode(&configFromFile); err != nil {
			return nil, fmt.Errorf("error parsing json file config: %w", err)
		}
		if err := mergo.Merge(&serverConfig, configFromFile); err != nil {
			return nil, fmt.Errorf("cannot merge configs: %w", err)
		}
	}

	return &serverConfig, env.Parse(&serverConfig)
}
