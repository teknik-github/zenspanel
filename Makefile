.PHONY: build test run-api run-agent dev lint

build:
	go build -o bin/zenspanel-api ./cmd/api
	go build -o bin/zenspanel-agent ./cmd/agent

test:
	go test ./... -v -count=1

run-api:
	go run ./cmd/api

run-agent:
	go run ./cmd/agent

dev:
	air -c .air.toml

lint:
	golangci-lint run ./...
