package exchange

//go:generate go run -mod=mod github.com/golang/mock/mockgen --source=exchange.go --destination=../mocks/exchange/exchange.go

type OrderManager interface {
	UpdateOrder(orderID int, newPrice float64, newQuantity float64)
}
