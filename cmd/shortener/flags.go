package main

import "flag"

type serverConfig struct {
	flagRunAddr     string
	redirectBaseURL string
}

var flagsConfig serverConfig

func InitFlags() {
	flag.StringVar(&flagsConfig.flagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagsConfig.redirectBaseURL, "b", "http://localhost:8080", "server uri prefix")
	flag.Parse()
}
