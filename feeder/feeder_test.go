package feeder

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

const (
	testSymbol = "test"
	tradesURL  = "/ws/test@trade"
	depthURL   = "/ws/test@depth@100ms"
)

var logBuffer *SyncBuffer

type SyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *SyncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *SyncBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}

func TestMain(m *testing.M) {
	// TODO: Look into changing this to not set a log buffer for all
	// tests, but just for the tests that need it.
	// Issue opened with zerolog about not being able to do this with
	// the global logger: https://github.com/rs/zerolog/issues/242
	logBuffer = new(SyncBuffer)
	log.Logger = log.Output(zerolog.SyncWriter(logBuffer))
	os.Exit(m.Run())
}

var (
	rawTrade = `{
		"e": "trade",
		"E": 123456789,
		"s": "BNBBTC",
		"t": 12345,
		"p": "0.001",
		"q": "100",
		"b": 88,
		"a": 50,
		"T": 123456785,
		"m": true,
		"M": true
	}`

	expectedTrade = Trade{
		BuyerOrderID:  88,
		SellerOrderID: 50,
		TradeTime:     123456785,
		Order: Order{
			ID: 12345,
			Event: Event{
				Type:      "trade",
				EventTime: 123456789,
				Price:     "0.001",
				Quantity:  "100",
			},
		},
	}
)

func TestBinanceFeederTradesReturnsWorkingChannelThatReceivesTrades(t *testing.T) {
	//arrange
	mc := make(chan string, 2)
	defer close(mc)

	ws := newTestServer(tradesURL, mc)
	defer ws.Close()

	mc <- rawTrade

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:     testDialer,
		MaxRetries: 0,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	tc, err := bf.Trades()

	//assert
	assert.NoError(t, err)

	actualTrade := <-tc
	assert.Equal(t, expectedTrade, actualTrade)
}

func TestBinanceFeederTradesRetriesOnConnectionFailure(t *testing.T) {
	//arrange

	// TODO: Should set the test to use a log buffer explicitly on a
	// test by test basis. But setting zerolog global log output isn't
	// thread-safe and creates race condition when run with -race flag.
	// Opened issue with zerolog: https://github.com/rs/zerolog/issues/242
	// log.Logger = zerolog.New(zerolog.SyncWriter(logBuffer))
	// defer t.Cleanup(func() {
	// 	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	// })

	logBuffer.Reset()

	// Start, then restart webserver (as need an address for BinanceFeeder)
	ws := newTestServer(tradesURL, make(chan string, 2))
	defer ws.Close()
	mc := ws.RestartWithDelay(500 * time.Millisecond)

	mc <- rawTrade

	testDialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 100 * time.Millisecond,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	tc, err := bf.Trades()

	//assert
	assert.NoError(t, err)

	assertContainsErrorLog(t, logBuffer.buf, "connection error")

	actualTrade := <-tc
	assert.Equal(t, expectedTrade, actualTrade)
}

func TestBinanceFeederTradesReturnsErrorOnConnectionFailureAfterMaxRetries(t *testing.T) {
	//arrange
	logBuffer.Reset()

	mc := make(chan string, 2)
	defer close(mc)

	ws := newTestServer("/doesnotexist", mc)
	defer ws.Close()

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 10 * time.Millisecond,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	_, err := bf.Trades()

	//assert
	assert.Equal(t, errDialConnection, err)
	assertContainsErrorLog(t, logBuffer.buf, "max retries reached")
}

func TestBinanceFeederTradesRetriesOnWebsocketReadError(t *testing.T) {
	//arrange
	mc := make(chan string, 3)
	ws := newTestServer(tradesURL, mc)
	defer ws.Close()

	mc <- rawTrade

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	tc, err := bf.Trades()

	// close channel and restart webserver
	close(mc)
	mc = ws.RestartWithDelay(100 * time.Millisecond)
	defer close(mc)

	mc <- rawTrade

	//assert
	assert.NoError(t, err)

	actualTrade1, ok := <-tc
	assert.True(t, ok)
	assert.Equal(t, expectedTrade, actualTrade1)

	actualTrade2, ok := <-tc
	assert.True(t, ok)
	assert.Equal(t, expectedTrade, actualTrade2)
}

func TestBinanceFeederTradesClosesChannelAfterMaxRetries(t *testing.T) {
	//arrange
	mc := make(chan string, 3)
	ws := newTestServer(tradesURL, mc)
	defer ws.Close()

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  0,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	tc, err := bf.Trades()

	// close channel and restart webserver
	close(mc)
	mc = ws.RestartWithDelay(0)
	defer close(mc)

	//assert
	assert.NoError(t, err)

	_, ok := <-tc
	assert.False(t, ok)
}

func TestBinanceFeederTradesSkipsAndLogsTradeOnUnmarshalError(t *testing.T) {
	//arrange
	logBuffer.Reset()

	var badTrade = `{
		"e": "trade",
		"E": 123456789,
		"s": "BNBBTC",
		"t": 12345,
		"p": "0.001",
		"q": "100",
		"b": 88,
		"a": 50,
		"T": "TIMESTAMP",
		"m": true,
		"M": true
	}`

	var goodTrade = `{
		"e": "trade",
		"E": 123456789,
		"s": "BNBBTC",
		"t": 12345,
		"p": "0.001",
		"q": "100",
		"b": 88,
		"a": 50,
		"T": 123456785,
		"m": true,
		"M": true
	}`

	var expectedGoodTrade = Trade{
		BuyerOrderID:  88,
		SellerOrderID: 50,
		TradeTime:     123456785,
		Order: Order{
			ID: 12345,
			Event: Event{
				Type:      "trade",
				EventTime: 123456789,
				Price:     "0.001",
				Quantity:  "100",
			},
		},
	}

	mc := make(chan string, 3)
	defer close(mc)

	ws := newTestServer(tradesURL, mc)
	defer ws.Close()

	mc <- badTrade
	mc <- goodTrade

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	tc, err := bf.Trades()

	//assert
	assert.NoError(t, err)

	actualTrade, ok := <-tc
	assert.True(t, ok)
	assert.Equal(t, expectedGoodTrade, actualTrade)

	assertContainsErrorLog(t, logBuffer.buf, "error unmarshalling trade")
}

var (
	rawBookUpdate = `{
		"e": "depthUpdate",
		"E": 123456789,
		"s": "BNBBTC",
		"U": 157,
		"u": 160,
		"b": [
			[
			"0.0024",
			"10"
			]
		],
		"a": [
			[
			"0.0026",
			"100"
			]
		]
	}`

	expectedBidEvent = Event{
		Type:      "bid",
		EventTime: 123456789,
		Price:     "0.0024",
		Quantity:  "10",
	}

	expectedAskEvent = Event{
		Type:      "ask",
		EventTime: 123456789,
		Price:     "0.0026",
		Quantity:  "100",
	}
)

func TestBinanceFeederBookUpdatesReturnsWorkingChannelThatReturnsEvents(t *testing.T) {
	//arrange
	mc := make(chan string, 2)
	defer close(mc)

	ws := newTestServer(depthURL, mc)
	defer ws.Close()

	mc <- rawBookUpdate

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:     testDialer,
		MaxRetries: 0,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	ec, err := bf.BookUpdates()

	//assert
	assert.NoError(t, err)

	var actuals []Event

	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)

	assert.Contains(t, actuals, expectedBidEvent)
	assert.Contains(t, actuals, expectedAskEvent)
}

func TestBinanceFeederBookUpdatesRetriesOnConnectionFailure(t *testing.T) {
	//arrange
	logBuffer.Reset()

	// Start, then restart webserver (as need an address for BinanceFeeder)
	ws := newTestServer(depthURL, make(chan string, 2))
	defer ws.Close()
	mc := ws.RestartWithDelay(500 * time.Millisecond)

	mc <- rawBookUpdate

	testDialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 100 * time.Millisecond,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	ec, err := bf.BookUpdates()

	//assert
	assert.NoError(t, err)
	assertContainsErrorLog(t, logBuffer.buf, "connection error")

	var actuals []Event

	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)

	assert.Contains(t, actuals, expectedBidEvent)
	assert.Contains(t, actuals, expectedAskEvent)
}

func TestBinanceFeederBookUpdatesReturnsErrorOnConnectionFailureAfterMaxRetries(t *testing.T) {
	//arrange
	logBuffer.Reset()

	mc := make(chan string, 2)
	defer close(mc)

	ws := newTestServer("/doesnotexist", mc)
	defer ws.Close()

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 10 * time.Millisecond,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	_, err := bf.BookUpdates()

	//assert
	assertContainsErrorLog(t, logBuffer.buf, "max retries reached")
	assert.Equal(t, errDialConnection, err)
}

func TestBinanceFeederBookUpdatesRetriesOnWebsocketReadError(t *testing.T) {
	//arrange
	mc := make(chan string, 3)
	ws := newTestServer(depthURL, mc)
	defer ws.Close()

	mc <- rawBookUpdate

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	ec, err := bf.BookUpdates()

	// close channel and restart webserver
	close(mc)
	mc = ws.RestartWithDelay(100 * time.Millisecond)
	defer close(mc)

	mc <- rawBookUpdate

	//assert
	assert.NoError(t, err)

	var actuals []Event

	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)

	assert.Contains(t, actuals, expectedBidEvent)
	assert.Contains(t, actuals, expectedAskEvent)
	assert.Len(t, actuals, 4)
}

func TestBinanceFeederBookUpdatesClosesChannelAfterMaxRetries(t *testing.T) {
	//arrange
	mc := make(chan string, 3)
	ws := newTestServer(depthURL, mc)
	defer ws.Close()

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  0,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	ec, err := bf.BookUpdates()

	// close channel and restart webserver
	close(mc)
	mc = ws.RestartWithDelay(0)
	defer close(mc)

	//assert
	assert.NoError(t, err)

	_, ok := <-ec
	assert.False(t, ok)
}

func TestBinanceFeederBookUpdatesSkipsAndLogsTradeOnUnmarshalError(t *testing.T) {
	//arrange
	logBuffer.Reset()

	var badBookUpdate = `{
		"e": "depthUpdate",
		"E": "TIMESTAMP",
		"s": "BNBBTC",
		"U": 157,
		"u": 160,
		"b": [
			[
			"0.0024",
			"10"
			]
		],
		"a": [
			[
			"0.0026",
			"100"
			]
		]
	}`

	var goodBookUpdate = `{
		"e": "depthUpdate",
		"E": 123456789,
		"s": "BNBBTC",
		"U": 157,
		"u": 160,
		"b": [
			[
			"0.0024",
			"10"
			]
		],
		"a": [
			[
			"0.0026",
			"100"
			]
		]
	}`

	var expectedGoodBidEvent = Event{
		Type:      "bid",
		EventTime: 123456789,
		Price:     "0.0024",
		Quantity:  "10",
	}

	var expectedGoodAskEvent = Event{
		Type:      "ask",
		EventTime: 123456789,
		Price:     "0.0026",
		Quantity:  "100",
	}

	mc := make(chan string, 3)
	defer close(mc)

	ws := newTestServer(depthURL, mc)
	defer ws.Close()

	mc <- badBookUpdate
	mc <- goodBookUpdate

	testDialer := websocket.DefaultDialer
	testDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	opts := &SocketConnectionOptions{
		Dialer:      testDialer,
		MaxRetries:  1,
		BackOffTime: 1 * time.Second,
	}

	bf := &BinanceFeeder{
		baseURL:       strings.TrimPrefix(ws.URL, "wss://"),
		socketOptions: opts,
		Symbol:        testSymbol,
	}

	//act
	ec, err := bf.BookUpdates()

	//assert
	assert.NoError(t, err)

	var actuals []Event
	actuals = append(actuals, <-ec)
	actuals = append(actuals, <-ec)

	assert.Contains(t, actuals, expectedGoodBidEvent)
	assert.Contains(t, actuals, expectedGoodAskEvent)

	assertContainsErrorLog(t, logBuffer.buf, "error unmarshalling book update")
}

type logline struct {
	Msg   string `json:"message"`
	Level string `json:"level"`
}

func assertContainsErrorLog(t *testing.T, b bytes.Buffer, msg string) {
	logs, err := logContents(b)
	assert.NoError(t, err)

	for _, l := range logs {
		if l.Level == "error" && strings.Contains(l.Msg, msg) {
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("log buffer does not contain msg: %s", msg))
}

func logContents(b bytes.Buffer) ([]logline, error) {
	var logs []logline
	scanner := bufio.NewScanner(&b)
	for scanner.Scan() {
		var logObject logline
		fmt.Println(string(scanner.Bytes()))
		err := json.Unmarshal(scanner.Bytes(), &logObject)
		if err != nil {
			return []logline{}, err
		}
		logs = append(logs, logObject)
	}
	return logs, nil
}

type testServer struct {
	*httptest.Server
	path         string
	chanCapacity int
}

// RestartWithDelay stops the webserver for a duration before restarting
// it on the same URL and port as before with a new chan
func (ts *testServer) RestartWithDelay(duration time.Duration) chan string {
	newChan := make(chan string, ts.chanCapacity)
	go func() {
		ts.Close()

		// create a listener with the desired port
		l, err := net.Listen("tcp", strings.TrimPrefix(ts.URL, "wss://"))
		if err != nil {
			panic(err)
		}

		handler := newWebsocketHandler(ts.path, newChan)
		ts = &testServer{
			httptest.NewUnstartedServer(handler),
			ts.path,
			ts.chanCapacity,
		}

		// NewUnstartedServer creates a listener. Close that listener and replace
		// with the one we created
		ts.Listener.Close()
		ts.Listener = l

		// Pause for th delay
		time.Sleep(duration)

		// Start the server
		ts.StartTLS()
		ts.URL = "wss" + strings.TrimPrefix(ts.URL, "https")

		fmt.Printf("Server re-listening at: %s\n", ts.URL)
	}()

	return newChan
}

func newTestServer(path string, messages <-chan string) *testServer {
	handler := newWebsocketHandler(path, messages)
	ws := httptest.NewTLSServer(handler)
	ws.URL = "wss" + strings.TrimPrefix(ws.URL, "https")

	fmt.Printf("Server listening at: %s\n", ws.URL)

	ts := &testServer{
		ws,
		path,
		cap(messages),
	}

	return ts
}

func newWebsocketHandler(path string, messages <-chan string) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		c, _ := upgrader.Upgrade(w, r, nil)

		for m := range messages {
			fmt.Printf("Sending message: %s\n", m)
			err := c.WriteMessage(websocket.TextMessage, []byte(m))
			if err != nil {
				panic("Failed to write message")
			}
		}

		// close connection after messages chan is closed
		c.Close()
	})

	return router
}
