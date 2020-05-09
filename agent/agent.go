package agent

import (
	"log"
	"stevestotter/go-binance-trader/feeder"
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
		log.Fatalf("error on reading trades: %s", err)
	}

	eChan, err := a.Feed.BookUpdates()
	if err != nil {
		log.Fatalf("error on reading book updates: %s", err)
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

func (a *Agent) OnTradeUpdate(t feeder.Trade) {
	log.Printf("Trade received: %+v", t)
}

func (a *Agent) OnBookUpdate(e feeder.Event) {
	log.Printf("Update received: %+v", e)
}
