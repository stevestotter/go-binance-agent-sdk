FROM golang:1.14

WORKDIR /go/src/go-binance-agent-sdk
COPY . .

RUN make deps
RUN go install -v ./...

CMD ["go-binance-agent-sdk"]