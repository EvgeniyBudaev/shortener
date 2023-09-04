package main

import "flag"

type serverConfig struct {
	flagRunAddr     string
	redirectBaseURL string
}

var config serverConfig

func initFlags() {
	flag.StringVar(&config.flagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&config.redirectBaseURL, "b", "http://localhost:8080", "server uri prefix")
	flag.Parse()
}
