export GO111MODULE=on

GOFILES= $$(go list -f '{{join .GoFiles " "}}')

deps:
	go mod vendor

run-simple-agent:
	go run main.go

docker-stack-up:
	docker-compose up -d

docker-stack-down:
	docker-compose down

docker-stack-destroy:
	docker-compose down -v