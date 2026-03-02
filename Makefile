.PHONY: dev build test lint docker-build docker-run clean

# Binary output
BINARY := wingstation
BUILD_DIR := ./cmd/wingstation

# Go settings
GOFLAGS := -v
LDFLAGS := -s -w

## dev: Run with live reload (requires air: go install github.com/air-verse/air@latest)
dev:
	@echo "Starting WingStation in dev mode..."
	@go run $(BUILD_DIR)

## build: Build the binary
build:
	@echo "Building $(BINARY)..."
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(BUILD_DIR)

## test: Run tests with race detection
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## docker-build: Build Docker image
docker-build:
	docker build -t wingstation:dev .

## docker-run: Run Docker container locally
docker-run: docker-build
	docker run --rm -p 8080:8080 \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		--read-only \
		--security-opt no-new-privileges:true \
		wingstation:dev

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out
