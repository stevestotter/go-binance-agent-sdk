export GO111MODULE=on

GOFILES= $$(go list -f '{{join .GoFiles " "}}')

.PHONY: mocks

deps:
	go mod vendor

mocks:
	rm -rf mocks
	go generate -v ./...

test:
	go test -race -count=1 ./...