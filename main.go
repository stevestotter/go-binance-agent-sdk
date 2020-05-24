package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"stevestotter/go-binance-trader/agent"
	"stevestotter/go-binance-trader/feeder"
)

// To put in config file
const binanceURL = "stream.binance.com:9443"
const symbol = "btcusdt"
const logFile = "./logs/app.log"

func main() {
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal().Err(err).
			Msg("Fatal error opening log file")
	}
	defer file.Close()

	log.Logger = log.Output(file)
	log.Info().Msg("started")

	bf := feeder.NewBinanceFeeder(binanceURL, symbol)

	a := agent.Agent{
		Feed: bf,
	}

	a.Start()
}
