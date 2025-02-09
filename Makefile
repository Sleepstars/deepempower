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
	mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) -o ./bin/$(BINARY_NAME) $(MAIN_PATH)
	$(GOBUILD) $(BUILD_FLAGS) -o ./bin/mockserver ./cmd/mockserver/main.go

test:
	$(GOTEST) $(TEST_FLAGS) ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

run:
	$(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)
	./bin/$(BINARY_NAME)

run-mock:
	$(GOBUILD) $(BUILD_FLAGS) -o bin/mockserver cmd/mockserver/main.go
	./bin/mockserver -port 8001 & P1=$$!; \
	./bin/mockserver -port 8002 & P2=$$!; \
	echo "Mock servers started on ports 8001 and 8002"; \
	wait

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

# Release targets
.PHONY: release-tag release-build

VERSION ?= $(shell git describe --tags --always --dirty)
PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

release-build:
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=bin/deepempower-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then \
			output=$$output.exe; \
		fi; \
		echo "Building for $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -v -o $$output ./cmd/server; \
	done

release-tag:
	@if [ -z "$(TAG)" ]; then \
		echo "Please provide a tag. Example: make release-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating new release tag: $(TAG)"
	@git tag -a $(TAG) -m "Release $(TAG)"
	@git push origin $(TAG)

# Docker targets
.PHONY: docker-build-multiarch docker-push

DOCKER_REGISTRY ?= ghcr.io
DOCKER_IMAGE ?= $(DOCKER_REGISTRY)/$(shell basename $$(pwd))

docker-build-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):$(VERSION) .

docker-push: docker-build-multiarch
	docker push $(DOCKER_IMAGE):$(VERSION)

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
	@echo "make release-build - Build binaries for all supported platforms"
	@echo "make release-tag TAG=v1.0.0 - Create and push a new release tag"
	@echo "make docker-build-multiarch - Build multi-arch Docker image"
	@echo "make docker-push - Push Docker image to registry"
