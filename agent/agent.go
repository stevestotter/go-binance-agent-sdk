package agent

import (
	"fmt"
	"stevestotter/go-binance-trader/feeder"
	"strconv"

	"github.com/rs/zerolog/log"
)

type Agent struct {
	Feed feeder.Feeder
}

type MarketListener interface {
	OnTradeUpdate(feeder.Trade)
	OnBookUpdate(feeder.Event)
	//OnAssignment
	//OnTradeSuccessful
}

func (a *Agent) Start() {
	tChan, err := a.Feed.Trades()
	if err != nil {
		log.Fatal().Err(err).
			Msg("error on reading trades")
	}

	eChan, err := a.Feed.BookUpdates()
	if err != nil {
		log.Fatal().Err(err).
			Msg("error on reading book updates")
	}

	go func() {
		for t := range tChan {
			a.OnTradeUpdate(t)
		}
	}()

	for e := range eChan {
		a.OnBookUpdate(e)
	}
}

// OnTradeUpdate just logs for now
func (a *Agent) OnTradeUpdate(t feeder.Trade) {
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
}

// OnBookUpdate just logs for now
func (a *Agent) OnBookUpdate(e feeder.Event) {
	price, err := strconv.ParseFloat(e.Price, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert price to float")
	}

	quantity, err := strconv.ParseFloat(e.Quantity, 64)
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to convert quantity to float")
	}

	if quantity == 0 {
		log.Debug().
			Float64("price", price).
			Str("event_type", fmt.Sprintf("%s_removed", e.Type)).
			Msg("book removal")
	} else {
		log.Debug().
			Float64("price", price).
			Float64("quantity", quantity).
			Str("event_type", e.Type).
			Msg("book update")
	}
}
