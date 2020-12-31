package exchange

//go:generate go run -mod=mod github.com/golang/mock/mockgen --source=exchange.go --destination=../mocks/exchange/exchange.go

// MarketExchange structs implement the ability to manage orders and get exchange info
type MarketExchange interface {
	UpdateOrder(orderID string, newPrice float64, newQuantity float64)
	OrderFulfilled(orderID string, price float64, quantity float64)
	GetBestBid() float64
	GetBestAsk() float64
}

// TODO: implement MarketExchange interface on a new Exchange struct
