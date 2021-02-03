package main

import (
	"github.com/stevestotter/go-binance-trader/agent"
	"github.com/stevestotter/go-binance-trader/feeder"

	"github.com/rs/zerolog/log"
)

// Mode describes the type of agent
type Mode int

const (
	// Buyer mode instructs agents to make bid shouts in order to purchase commodities
	Buyer Mode = iota + 1
	// Seller mode instructs agents to make ask shouts in order to sell commodities
	Seller
)

type SimpleStrategy struct {
	Mode Mode
}

func NewSimpleStrategyBuyer() SimpleStrategy {
	return SimpleStrategy{Mode: Buyer}
}

func (s *SimpleStrategy) NewOrder(orderID string, price float64, quantity float64, params ...interface{}) error {
	//TODO
	return nil
}

func (s *SimpleStrategy) OnTrade(price float64, quantity float64, tradeID string, buyerOrderID string, sellerOrderID string, params ...interface{}) {
	//TODO
}

func (s *SimpleStrategy) OnBookUpdateBid(price float64, quantity float64, params ...interface{}) {
	//TODO
}

func (s *SimpleStrategy) OnBookUpdateAsk(price float64, quantity float64, params ...interface{}) {
	//TODO
}

func main() {
	log.Info().Msg("started")

	bf := feeder.NewBinanceFeeder("btcusdt")

	// TODO randomise sleep by +-10% to avoid synchronousity with other agents launched at same time
	s := NewSimpleStrategyBuyer()

	a := agent.Agent{
		Feed:     bf,
		Strategy: &s,
	}

	a.Start()
}
