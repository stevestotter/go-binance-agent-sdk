package agent

import (
	"fmt"
	"strconv"

	"github.com/stevestotter/go-binance-trader/feeder"

	"github.com/rs/zerolog/log"
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
	Feed     feeder.Feeder
	Strategy MarketListener
	//TODO: Initial orders
}

// Start gets the Agent to start listening to market feeds, convert
// assignments to orders, and adjusts prices of those orders continuously
func (a *Agent) Start() error {
	// Make sure we get new random numbers each run
	// TODO: a.Strategy.Init() or on new function?

	// TODO: Add assignment feed

	tChan, err := a.Feed.Trades()
	if err != nil {
		log.Error().Err(err).
			Msg("error on reading trades")
		return err
	}

	eChan, err := a.Feed.BookUpdates()
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

	for e := range eChan {
		a.onBookEvent(e)
	}

	return nil
}

func (a *Agent) onTradeEvent(t feeder.Trade) {
	price, err := strconv.ParseFloat(t.Price, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert price to float")
	}

	quantity, err := strconv.ParseFloat(t.Quantity, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert quantity to float")
	}

	log.Debug().
		Float64("price", price).
		Float64("quantity", quantity).
		Str("event_type", t.Type).
		Int("trade_id", t.ID).
		Int("buyer_order_id", t.BuyerOrderID).
		Int("seller_order_id", t.SellerOrderID).
		Msg("trade")

	a.Strategy.OnTrade(price, quantity, fmt.Sprint(t.ID), fmt.Sprint(t.BuyerOrderID), fmt.Sprint(t.SellerOrderID))
}

func (a *Agent) onBookEvent(e feeder.Event) {
	price, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert price to float")
		return
	}

	quantity, err := strconv.ParseFloat(e.Quantity, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert quantity to float")
		return
	}

	if quantity != 0 {
		log.Debug().
			Float64("price", price).
			Float64("quantity", quantity).
			Str("event_type", e.Type).
			Msg("book update")

		switch e.Type {
		case feeder.Ask:
			a.Strategy.OnBookUpdateAsk(price, quantity)
		case feeder.Bid:
			a.Strategy.OnBookUpdateBid(price, quantity)
		default:
			log.Error().
				Msgf("Unknown event type on book update: %s", e.Type)
			return
		}
	} else {
		log.Debug().
			Float64("price", price).
			Str("event_type", fmt.Sprintf("%s_removed", e.Type)).
			Msg("book removal")
	}
}
