package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	FlagRunAddr     string `env:"SERVER_ADDRESS"`
	RedirectBaseURL string `env:"BASE_URL"`
}

var serverConfig ServerConfig

func InitFlags() (*ServerConfig, error) {
	flag.StringVar(&serverConfig.FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&serverConfig.RedirectBaseURL, "b", "http://localhost:8080", "server URI prefix")
	flag.Parse()

	return &serverConfig, env.Parse(&serverConfig)
}
