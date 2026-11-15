.PHONY: all build build-web build-go test bench clean run

BINARY_NAME=schemadoctor
BIN_DIR=bin

all: build

build-web:
	@echo "Building React 19 Studio..."
	cd web && npm run build

build-go:
	@echo "Building Go reverse proxy binary..."
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/schemadoctor

build: build-web build-go

test:
	@echo "Running tests..."
	go test -v -race ./internal/...

bench:
	@echo "Running internal benchmarks..."
	go test -bench=. -benchmem ./internal/ast

run: build-go
	./$(BIN_DIR)/$(BINARY_NAME) serve --mock-upstream

clean:
	@rm -rf $(BIN_DIR) web/dist data/*.db
