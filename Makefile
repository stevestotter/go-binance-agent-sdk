export GO111MODULE=on

GOFILES= $$(go list -f '{{join .GoFiles " "}}')

deps:
	go mod vendor

run:
	go run $(GOFILES)