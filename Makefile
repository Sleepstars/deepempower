.PHONY: all build test clean lint coverage integration-test

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=deepempower
MAIN_PATH=./cmd/server/main.go

# Build flags
BUILD_FLAGS=-v

# Test flags
TEST_FLAGS=-v -race
COVERAGE_FLAGS=-coverprofile=coverage.out
INTEGRATION_TEST_FLAGS=-tags=integration

all: deps test build

build:
	$(GOBUILD) $(BUILD_FLAGS) -o ./bin/$(BINARY_NAME) $(MAIN_PATH)

test:
	$(GOTEST) $(TEST_FLAGS) ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

run:
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	./$(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	golangci-lint run

coverage:
	$(GOTEST) $(COVERAGE_FLAGS) ./...
	go tool cover -html=coverage.out

integration-test:
	$(GOTEST) $(TEST_FLAGS) $(INTEGRATION_TEST_FLAGS) ./test/integration/...

# Development helpers
mock:
	mockgen -source=internal/clients/types.go -destination=internal/mocks/model_client.go -package=mocks

fmt:
	go fmt ./...

vet:
	go vet ./...

# Docker
docker-build:
	docker build -t deepempower .

docker-run:
	docker run -p 8080:8080 deepempower

# Development environment
dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest

# Help
help:
	@echo "make - Build the project"
	@echo "make build - Build the binary"
	@echo "make test - Run unit tests"
	@echo "make clean - Clean build files"
	@echo "make run - Build and run the application"
	@echo "make deps - Download dependencies"
	@echo "make lint - Run linter"
	@echo "make coverage - Generate test coverage report"
	@echo "make integration-test - Run integration tests"
	@echo "make mock - Generate mock files"
	@echo "make fmt - Format code"
	@echo "make vet - Run go vet"
	@echo "make docker-build - Build Docker image"
	@echo "make docker-run - Run Docker container"
	@echo "make dev-deps - Install development dependencies"
