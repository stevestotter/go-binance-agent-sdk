package agent

import (
	"errors"
	"testing"

	"stevestotter/go-binance-trader/feeder"
	mock_agent "stevestotter/go-binance-trader/mocks/agent"
	mock_feeder "stevestotter/go-binance-trader/mocks/feeder"

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

// TODO: func TestAgentStartSendsTradesToStrategy(t *testing.T)
// TODO: func TestAgentStartSendsBookBidUpdatesToStrategy(t *testing.T)
// TODO: func TestAgentStartSendsBookAskUpdatesToStrategy(t *testing.T)
// TODO: func TestAgentStartDoesNotSendUnknownBookUpdatesToStrategy(t *testing.T)
