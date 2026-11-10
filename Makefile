.PHONY: build test lint run clean docker-up docker-down vet

APP_NAME := orderforge
BUILD_DIR := ./bin

build:
	@echo "Building $(APP_NAME)..."
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

test:
	go test -v -race -count=1 ./...

vet:
	go vet ./...

lint: vet
	@which staticcheck > /dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

run: build
	$(BUILD_DIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
