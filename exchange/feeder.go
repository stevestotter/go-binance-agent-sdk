package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen --source=feeder.go --destination=../mocks/exchange/feeder.go

const (
	// BinanceURL is the base URL for all interactions with the Binance platform
	BinanceURL string = "stream.binance.com:9443"

	// Bid is an event type for a bid in the market
	Bid string = "bid"
	// Ask is an event type for an ask in the market
	Ask string = "ask"
)

var (
	errDialConnection = errors.New("Failed to establish a connection")
)

// Feeder is an interface for market exchange feeds
// Trades returns a channel of Trades made in the market
// BookUpdates returns a channel of batched BookUpdates indicating market order book activity
type Feeder interface {
	Trades() (<-chan Trade, error)
	BookUpdates() (<-chan BookUpdate, error)
	GetSymbol() string
}

// Taken from https://binance-docs.github.io/apidocs/spot/en/#aggregate-trade-streams
// {
//   "e": "trade",     // Event type
//   "E": 123456789,   // Event time
//   "s": "BNBBTC",    // Symbol
//   "t": 12345,       // Trade ID
//   "p": "0.001",     // Price
//   "q": "100",       // Quantity
//   "b": 88,          // Buyer order ID
//   "a": 50,          // Seller order ID
//   "T": 123456785,   // Trade time
//   "m": true,        // Is the buyer the market maker?
//   "M": true         // Ignore
// }

// Trade contains information about a completed trade
type Trade struct {
	// Have to include json:"e" even though not wanted as encoding/json Unmarshal() has a bug with case-sensitivity on named parameters
	// Issue discussed here: https://github.com/golang/go/issues/14750
	// Watching proposed change to package here: https://go-review.googlesource.com/c/go/+/224079/
	Type string `json:"e"` // Will always be "trade"

	ID            int     `json:"t"`
	BuyerOrderID  int     `json:"b"`
	SellerOrderID int     `json:"a"`
	TradeTime     int     `json:"T"`
	EventTime     int     `json:"E"`
	Price         float64 `json:"p,string"`
	Quantity      float64 `json:"q,string"`
}

// Taken from https://binance-docs.github.io/apidocs/spot/en/#diff-depth-stream
// {
//   "e": "depthUpdate", // Event type
//   "E": 123456789,     // Event time
//   "s": "BNBBTC",      // Symbol
//   "U": 157,           // First update ID in event
//   "u": 160,           // Final update ID in event
//   "b": [              // Bids to be updated
//     [
//       "0.0024",       // Price level to be updated
//       "10"            // Quantity
//     ]
//   ],
//   "a": [              // Asks to be updated
//     [
//       "0.0026",       // Price level to be updated
//       "100"           // Quantity
//     ]
//   ]
// }

// BookUpdate contains orderbook depth updates - multiple bids and asks since the last update event
type BookUpdate struct {
	// Have to include Type even though not wanted as encoding/json Unmarshal() has a bug with case-sensitivity on named parameters
	// Issue discussed here: https://github.com/golang/go/issues/14750
	// Watching proposed change to package here: https://go-review.googlesource.com/c/go/+/224079/
	Type string `json:"e"` // Will always be "depthUpdate"

	EventTime    int         `json:"E"`
	Bids         []BookEntry `json:"b"`
	Asks         []BookEntry `json:"a"`
	LastUpdateID int         `json:"u"` // Final update ID in event
}

type BookEntry struct {
	Price    float64
	Quantity float64
}

func (be *BookEntry) UnmarshalJSON(buf []byte) error {
	var tmp []json.Number

	var err error
	if err = json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if numFields := len(tmp); numFields != 2 {
		return fmt.Errorf("wrong number of fields in bookEntry: %d != 2", numFields)
	}

	if be.Price, err = tmp[0].Float64(); err != nil {
		return err
	}
	if be.Quantity, err = tmp[1].Float64(); err != nil {
		return err
	}

	return nil
}

type binanceFeeder struct {
	baseURL       string
	socketOptions *SocketConnectionOptions
	symbol        string
}

type SocketConnectionOptions struct {
	Dialer      *websocket.Dialer
	MaxRetries  int
	BackOffTime time.Duration
}

var DefaultSocketOptions = &SocketConnectionOptions{
	Dialer:      websocket.DefaultDialer,
	MaxRetries:  5,
	BackOffTime: 5 * time.Second,
}

func NewBinanceFeeder(symbol string) *binanceFeeder {
	return &binanceFeeder{
		baseURL:       BinanceURL,
		socketOptions: DefaultSocketOptions,
		symbol:        symbol,
	}
}

func (bf *binanceFeeder) GetSymbol() string {
	return bf.symbol
}

func (bf *binanceFeeder) connectAndListen(url string, mChan chan []byte, attempt int) error {
	log.Info().Msgf("connecting to %s", url)

	var err error
	var conn *websocket.Conn
	for attempt <= bf.socketOptions.MaxRetries {
		conn, _, err = bf.socketOptions.Dialer.Dial(url, nil)
		if err == nil {
			log.Info().Msgf("successfully connected to %s", url)
			break
		} else {
			log.Error().Err(err).Msg("connection error")
			attempt++
			time.Sleep(bf.socketOptions.BackOffTime)
		}
	}
	if err != nil {
		close(mChan)
		log.Error().Msg("max retries reached")
		return errDialConnection
	}

	go func() {
		defer conn.Close()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("error on read")
				if attempt == bf.socketOptions.MaxRetries {
					log.Error().Msg("max retries reached")
					close(mChan)
					return
				}
				attempt++
				time.Sleep(bf.socketOptions.BackOffTime)
				go bf.connectAndListen(url, mChan, attempt)
				return
			}

			attempt = 0
			mChan <- message
		}
	}()

	return nil
}

// Trades returns a read-only channel of trades made on the market
func (bf *binanceFeeder) Trades() (<-chan Trade, error) {
	u := url.URL{Scheme: "wss", Host: bf.baseURL, Path: fmt.Sprintf("ws/%s@trade", bf.symbol)}

	mChan := make(chan []byte)
	err := bf.connectAndListen(u.String(), mChan, 0)

	tChan := make(chan Trade)
	go func() {
		defer close(tChan)
		for message := range mChan {
			var t Trade
			if err := json.Unmarshal(message, &t); err != nil {
				log.Error().Err(err).
					Str("detail", string(message)).
					Msgf("error unmarshalling trade")
				continue
			}
			tChan <- t
		}
	}()
	return tChan, err
}

// BookUpdates returns a read-only channel of updates made on the orderbook in the market.
func (bf *binanceFeeder) BookUpdates() (<-chan BookUpdate, error) {
	u := url.URL{Scheme: "wss", Host: bf.baseURL, Path: fmt.Sprintf("ws/%s@depth@100ms", bf.symbol)}

	mChan := make(chan []byte)
	err := bf.connectAndListen(u.String(), mChan, 0)

	buChan := make(chan BookUpdate)
	go func() {
		defer close(buChan)
		for message := range mChan {
			var b BookUpdate
			if err := json.Unmarshal(message, &b); err != nil {
				log.Error().Err(err).
					Str("detail", string(message)).
					Msgf("error unmarshalling book update")
				continue
			}
			buChan <- b
		}
	}()
	return buChan, err
}
