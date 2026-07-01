.DEFAULT_GOAL := build
.PHONY: generate fmt vet build run debug test clean

generate:
	go generate ./...

fmt: generate
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -o ./bin/ocpp-tester ./cmd/centralsystem/main.go

run: vet
	go run ./cmd/centralsystem/main.go -debug

debug: vet
	dlv debug ./cmd/centralsystem/main.go -- -debug

clean:
	go clean ./...

test: build
	go test ./...

tidy:
	go mod tidy
