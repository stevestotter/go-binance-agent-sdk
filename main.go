package main

import (
	"github.com/rs/zerolog/log"

	"stevestotter/go-binance-trader/agent"
	"stevestotter/go-binance-trader/feeder"
)

// To put in config file
const binanceURL = "stream.binance.com:9443"
const symbol = "btcusdt"

func main() {
	log.Info().Msg("started")

	bf := feeder.NewBinanceFeeder(binanceURL, symbol)

	a := agent.Agent{
		Feed: bf,
	}

	a.Start()
}
