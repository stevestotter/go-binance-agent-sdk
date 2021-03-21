package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/stevestotter/go-binance-agent-sdk/exchange"
	mock_agent "github.com/stevestotter/go-binance-agent-sdk/mocks/agent"
	mock_exchange "github.com/stevestotter/go-binance-agent-sdk/mocks/exchange"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAgentStartReturnsErrWhenTradeFeederFailsToInitialise(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)

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
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan exchange.Trade), nil)

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
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)
	tradeChan := make(chan exchange.Trade)

	var expPrice float64 = 1.01
	var expQuantity float64 = 100
	expTradeID := "12345"
	expBuyerOrderID := "88"
	expSellerOrderID := "52"

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(tradeChan, nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(make(chan exchange.BookUpdate), nil)

	mockStrategy.EXPECT().
		OnTrade(expPrice, expQuantity, expTradeID, expBuyerOrderID, expSellerOrderID, gomock.Any()).
		Times(1)

	a := Agent{mockFeeder, mockStrategy}
	go a.Start()

	trade := exchange.Trade{
		BuyerOrderID:  88,
		SellerOrderID: 52,
		TradeTime:     123456785,
		ID:            12345,
		Type:          "trade",
		EventTime:     123456789,
		Price:         expPrice,
		Quantity:      expQuantity,
	}

	tradeChan <- trade

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartSendsBookBidUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)
	bookUpdateChan := make(chan exchange.BookUpdate)

	var expPrice float64 = 0.24
	var expQuantity float64 = 10

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan exchange.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(bookUpdateChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateBid(expPrice, expQuantity, gomock.Any()).
		Times(1)

	mockStrategy.EXPECT().
		OnBookUpdateAsk(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	a := Agent{mockFeeder, mockStrategy}
	go a.Start()

	bookUpdate := exchange.BookUpdate{
		Bids: []exchange.BookEntry{
			{
				Price:    expPrice,
				Quantity: expQuantity,
			},
		},
	}

	bookUpdateChan <- bookUpdate

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartSendsBookAskUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)
	bookUpdateChan := make(chan exchange.BookUpdate)

	var expPrice float64 = 0.24
	var expQuantity float64 = 10

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan exchange.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(bookUpdateChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateAsk(expPrice, expQuantity, gomock.Any()).
		Times(1)

	mockStrategy.EXPECT().
		OnBookUpdateBid(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	a := Agent{mockFeeder, mockStrategy}
	go a.Start()

	bookUpdate := exchange.BookUpdate{
		Asks: []exchange.BookEntry{
			{
				Price:    expPrice,
				Quantity: expQuantity,
			},
		},
	}

	bookUpdateChan <- bookUpdate

	// Give time for mock to be asserted
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartDoesNotSendBidRemovalBookUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)
	bookUpdateChan := make(chan exchange.BookUpdate)

	var expPrice float64 = 0.24

	// quantity 0 indicates a removal
	var expQuantity float64 = 0

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan exchange.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(bookUpdateChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateBid(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	a := Agent{mockFeeder, mockStrategy}
	go a.Start()

	bookUpdate := exchange.BookUpdate{
		Bids: []exchange.BookEntry{
			{
				Price:    expPrice,
				Quantity: expQuantity,
			},
		},
	}

	bookUpdateChan <- bookUpdate

	// Give time for mock to not be asserted in this case
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}

func TestAgentStartDoesNotSendAskRemovalBookUpdatesToStrategy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)
	mockFeeder := mock_exchange.NewMockFeeder(ctrl)
	bookUpdateChan := make(chan exchange.BookUpdate)

	var expPrice float64 = 0.24

	// quantity 0 indicates a removal
	var expQuantity float64 = 0

	mockFeeder.EXPECT().
		Trades().
		Times(1).
		Return(make(chan exchange.Trade), nil)

	mockFeeder.EXPECT().
		BookUpdates().
		Times(1).
		Return(bookUpdateChan, nil)

	mockStrategy.EXPECT().
		OnBookUpdateAsk(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	a := Agent{mockFeeder, mockStrategy}
	go a.Start()

	bookUpdate := exchange.BookUpdate{
		Asks: []exchange.BookEntry{
			{
				Price:    expPrice,
				Quantity: expQuantity,
			},
		},
	}

	bookUpdateChan <- bookUpdate

	// Give time for mock to not be asserted in this case
	time.Sleep(time.Duration(0.2 * float64(time.Second)))
}
