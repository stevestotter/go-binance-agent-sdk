FROM golang:1.14

WORKDIR /go/src/go-binance-simple-agent
COPY . .

RUN go mod vendor
RUN go install -v ./...

CMD ["go-binance-simple-agent"]