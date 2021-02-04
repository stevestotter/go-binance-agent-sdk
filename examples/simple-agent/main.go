package main

import (
	"github.com/rs/zerolog/log"
	"github.com/stevestotter/go-binance-agent-sdk/agent"
	"github.com/stevestotter/go-binance-agent-sdk/feeder"
)

// Mode describes the type of agent
type Mode int

const (
	// Buyer mode instructs agents to make bid shouts in order to purchase commodities
	Buyer Mode = iota + 1
	// Seller mode instructs agents to make ask shouts in order to sell commodities
	Seller
)

// For the sake of this example, this is a const
const Market string = "btcusdt"

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
	log.Debug().
		Float64("price", price).
		Float64("quantity", quantity).
		Str("event_type", "trade").
		Str("trade_id", tradeID).
		Str("buyer_order_id", buyerOrderID).
		Str("seller_order_id", sellerOrderID).
		Msgf("trade in %s market", Market)

	//TODO
}

func (s *SimpleStrategy) OnBookUpdateBid(price float64, quantity float64, params ...interface{}) {
	log.Debug().
		Float64("price", price).
		Float64("quantity", quantity).
		Str("event_type", "bid").
		Msgf("book update in %s market", Market)

	//TODO
}

func (s *SimpleStrategy) OnBookUpdateAsk(price float64, quantity float64, params ...interface{}) {
	log.Debug().
		Float64("price", price).
		Float64("quantity", quantity).
		Str("event_type", "ask").
		Msgf("book update in %s market", Market)

	//TODO
}

func main() {
	bf := feeder.NewBinanceFeeder(Market)

	// TODO randomise sleep by +-10% to avoid synchronousity with other agents launched at same time
	s := NewSimpleStrategyBuyer()

	a := agent.Agent{
		Feed:     bf,
		Strategy: &s,
	}

	log.Info().Msg("simple-agent starting...")
	a.Start()
}
