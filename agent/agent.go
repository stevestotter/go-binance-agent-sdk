package agent

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/stevestotter/go-binance-agent-sdk/exchange"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen --source=agent.go --destination=../mocks/agent/agent.go

// MarketListener structs implement essential market functions for all agent strategies
type MarketListener interface {
	NewOrder(orderID string, price float64, quantity float64, params ...interface{}) error
	OnTrade(price float64, quantity float64, tradeID string, buyerOrderID string, sellerOrderID string, params ...interface{})
	OnBookUpdateBid(price float64, quantity float64, params ...interface{})
	OnBookUpdateAsk(price float64, quantity float64, params ...interface{})
}

// Agent is a trader in the market - either buying or selling goods
// The Strategy provided will make the intelligent decisions on what to
// do on trade & on market events
type Agent struct {
	Feed     exchange.Feeder
	Strategy MarketListener
	//TODO: Initial orders
}

// Start gets the Agent to start listening to market feeds, convert
// assignments to orders, and adjusts prices of those orders continuously
func (a *Agent) Start() error {
	// TODO: Add assignment feed

	tChan, err := a.Feed.Trades()
	if err != nil {
		log.Error().Err(err).
			Msg("error on reading trades")
		return err
	}

	buChan, err := a.Feed.BookUpdates()
	if err != nil {
		log.Error().Err(err).
			Msg("error on reading book updates")
		return err
	}

	go func() {
		for t := range tChan {
			a.onTradeEvent(t)
		}
	}()

	for b := range buChan {
		a.onBookUpdate(b)
	}

	return nil
}

func (a *Agent) onTradeEvent(t exchange.Trade) {
	a.Strategy.OnTrade(t.Price, t.Quantity, fmt.Sprint(t.ID), fmt.Sprint(t.BuyerOrderID), fmt.Sprint(t.SellerOrderID))
}

func (a *Agent) onBookUpdate(b exchange.BookUpdate) {
	// Iterate bids and asks in a concurrent fashion so that not all bids (/asks) are sent
	// to the strategy at the same time - which could skew agent mechanics
	go func(b exchange.BookUpdate) {
		for _, bid := range b.Bids {
			a.onBookUpdateBid(bid.Price, bid.Quantity)
		}
	}(b)

	go func(b exchange.BookUpdate) {
		for _, ask := range b.Asks {
			a.onBookUpdateAsk(ask.Price, ask.Quantity)
		}
	}(b)
}

func (a *Agent) onBookUpdateBid(price float64, quantity float64) {
	if quantity != 0 {
		a.Strategy.OnBookUpdateBid(price, quantity)
	} else {
		// removed off order book
		// TODO: potential additional event for strategy?
	}
}

func (a *Agent) onBookUpdateAsk(price float64, quantity float64) {
	if quantity != 0 {
		a.Strategy.OnBookUpdateAsk(price, quantity)
	} else {
		// removed off order book
		// TODO: potential additional event for strategy?
	}
}
