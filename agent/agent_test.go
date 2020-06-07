package agent

import (
	"errors"
	"stevestotter/go-binance-trader/feeder"
	mock_agent "stevestotter/go-binance-trader/mocks/agent"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// TODO: swap out for Gomock?
type mockFeeder struct {
	tradeChan     <-chan feeder.Trade
	eventChan     <-chan feeder.Event
	tradeErr      error
	bookUpdateErr error
}

func newMockFeeder(tErr, bErr error) *mockFeeder {
	return &mockFeeder{
		tradeChan:     make(chan feeder.Trade),
		eventChan:     make(chan feeder.Event),
		tradeErr:      tErr,
		bookUpdateErr: bErr,
	}
}

func (mf *mockFeeder) Trades() (<-chan feeder.Trade, error) {
	return mf.tradeChan, mf.tradeErr
}

func (mf *mockFeeder) BookUpdates() (<-chan feeder.Event, error) {
	return mf.eventChan, mf.bookUpdateErr
}

func TestAgentStartReturnsErrWhenTradeFeederFailsToInitialise(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)

	a := Agent{newMockFeeder(errors.New("trade error"), nil), mockStrategy}
	err := a.Start()

	assert.Error(t, err)
}

func TestAgentStartReturnsErrWhenBookUpdatesFeederFailsToInitialise(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mock_agent.NewMockMarketListener(ctrl)

	a := Agent{newMockFeeder(nil, errors.New("book updates error")), mockStrategy}
	err := a.Start()

	assert.Error(t, err)
}

// TODO: func TestAgentStartSendsTradesToStrategy(t *testing.T)
// TODO: func TestAgentStartSendsBookBidUpdatesToStrategy(t *testing.T)
// TODO: func TestAgentStartSendsBookAskUpdatesToStrategy(t *testing.T)
// TODO: func TestAgentStartDoesNotSendUnknownBookUpdatesToStrategy(t *testing.T)
