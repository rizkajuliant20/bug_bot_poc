.PHONY: build run clean docker-build docker-run install test

# Build the application
build:
	go build -o bug-bot ./cmd/bot

# Run the application
run:
	go run ./cmd/bot

# Install dependencies
install:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f bug-bot
	rm -rf logs/
	rm -f .notion-tracking.json

# Build Docker image
docker-build:
	docker build -t bug-bot:latest .

# Run Docker container
docker-run:
	docker run --env-file .env -v $(PWD)/logs:/root/logs bug-bot:latest

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Development mode with auto-reload (requires air)
dev:
	air
