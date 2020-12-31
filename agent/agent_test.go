package agent

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stevestotter/go-binance-trader/feeder"
	mock_agent "github.com/stevestotter/go-binance-trader/mocks/agent"
	mock_feeder "github.com/stevestotter/go-binance-trader/mocks/feeder"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAgentStartReturnsErrWhenTradeFeederFailsToInitialise(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(nil, errors.New("trade error"))

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()

	assert.Error(t, err)
}

func TestAgentStartReturnsErrWhenBookUpdatesFeederFailsToInitialise(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan feeder.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(nil, errors.New("book updates error"))

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()

	assert.Error(t, err)
}

func TestAgentStartSendsTradesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)
	tradeChan := make(chan feeder.Trade)

	var expPrice float64 = 1.01
	var expQuantity float64 = 100
	expTradeID := "12345"

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(tradeChan, nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(make(chan feeder.Event), nil)

	mockStrategy.EXPECT().
		OnTrade(expPrice, expQuantity, expTradeID, gomock.Any()).
		Times(1)

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()
	assert.NoError(t, err)

	trade := feeder.Trade{
		BuyerOrderID:  88,
		SellerOrderID: 50,
		TradeTime:     123456785,
		Order: feeder.Order{
			ID: 12345,
			Event: feeder.Event{
				Type:      "trade",
				EventTime: 123456789,
				Price:     fmt.Sprint(expPrice),
				Quantity:  fmt.Sprint(expQuantity),
			},
		},
	}

	tradeChan <- trade

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartSendsBookBidUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)
	eventChan := make(chan feeder.Event)

	var expPrice float64 = 0.24
	var expQuantity float64 = 10

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan feeder.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(eventChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateBid(expPrice, expQuantity, gomock.Any()).
		Times(1)

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()
	assert.NoError(t, err)

	event := feeder.Event{
		Type:      "bid",
		EventTime: 123456789,
		Price:     fmt.Sprint(expPrice),
		Quantity:  fmt.Sprint(expQuantity),
	}

	eventChan <- event

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartSendsBookAskUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)
	eventChan := make(chan feeder.Event)

	var expPrice float64 = 0.24
	var expQuantity float64 = 10

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan feeder.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(eventChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateAsk(expPrice, expQuantity, gomock.Any()).
		Times(1)

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()
	assert.NoError(t, err)

	event := feeder.Event{
		Type:      "ask",
		EventTime: 123456789,
		Price:     fmt.Sprint(expPrice),
		Quantity:  fmt.Sprint(expQuantity),
	}

	eventChan <- event

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartDoesNotSendRemovalBookUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)
	eventChan := make(chan feeder.Event)

	var expPrice float64 = 0.24

	// quantity 0 indicates a removal
	var expQuantity float64 = 0

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan feeder.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(eventChan, nil)

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()
	assert.NoError(t, err)

	event := feeder.Event{
		Type:      "bid",
		EventTime: 123456789,
		Price:     fmt.Sprint(expPrice),
		Quantity:  fmt.Sprint(expQuantity),
	}

	eventChan <- event

	// Give time for mock to not be asserted in this case
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartDoesNotSendUnknownBookUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_feeder.NewMockFeeder(ctrl)
	eventChan := make(chan feeder.Event)

	var expPrice float64 = 0.24
	var expQuantity float64 = 10

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan feeder.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(eventChan, nil)

	a := Agent{mockFeeder, mockStrategy}
	err := a.Start()
	assert.NoError(t, err)

	event := feeder.Event{
		Type:      "something",
		EventTime: 123456789,
		Price:     fmt.Sprint(expPrice),
		Quantity:  fmt.Sprint(expQuantity),
	}

	eventChan <- event

	// Give time for mock to not be asserted in this case
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}
