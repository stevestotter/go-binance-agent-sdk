FROM golang:1.14

WORKDIR /go/src/go-binance-trader
COPY . .

RUN make deps
RUN go install -v ./...

CMD ["go-binance-trader"]