# Makefile for Commercium E-commerce Platform

# Variables
BINARY_DIR := bin
API_GATEWAY_BINARY := $(BINARY_DIR)/api-gateway
USER_SERVICE_BINARY := $(BINARY_DIR)/user-service
CONFIG_DIR := configs
MIGRATION_DIR := migrations

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%S)"

.PHONY: all build clean test test-unit test-integration run-api-gateway run-user-service docker-build docker-up docker-down help

# Default target
all: build

# Build all services
build: build-api-gateway build-user-service

# Build API Gateway
build-api-gateway:
	@echo "Building API Gateway..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(API_GATEWAY_BINARY) ./cmd/api-gateway

# Build User Service  
build-user-service:
	@echo "Building User Service..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(USER_SERVICE_BINARY) ./cmd/user-service

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BINARY_DIR)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run all tests
test: ## Run all tests
	@echo "\nRunning all tests..."
	@go test -v -race -coverprofile=coverage.out ./... | tee test_output.log
	# Cleanup containers after tests
	-docker compose -f docker-compose.dev.yml down -v
	@echo "\n\033[1mTest Results Summary:\033[0m"
	@echo "----------------------------------------"
	@grep -e '--- PASS:' -e '--- FAIL:' -e '--- SKIP:' test_output.log | sed -E 's/--- PASS:/\033[32m| PASS |\033[0m/; s/--- FAIL:/\033[31m| FAIL |\033[0m/; s/--- SKIP:/\033[33m| SKIP |\033[0m/' | column -t -s'|'
	@echo "----------------------------------------"
	@rm -f test_output.log

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short ./...

# Run integration tests with development infrastructure
test-integration: dev-db-up
	@echo "Running integration tests..."
	$(GOTEST) -v -run Integration ./tests/integration/...

# Development Database Commands
dev-db-up:
	@echo "Starting development databases..."
	docker compose -f docker-compose.dev.yml up -d postgres redis
	@echo "Waiting for databases to be ready..."
	@sleep 5
	@echo "Databases ready for development!"

dev-db-down:
	@echo "Stopping development databases..."
	docker compose -f docker-compose.dev.yml down postgres redis

dev-db-logs:
	@echo "Showing database logs..."
	docker compose -f docker-compose.dev.yml logs -f postgres redis

# Full Development Environment
dev-up:
	@echo "Starting full development environment..."
	docker compose -f docker-compose.dev.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Development environment ready!"
	@echo "Services available:"
	@echo "  PostgreSQL: localhost:5432 (commercium_user/commercium_password)"
	@echo "  Redis: localhost:6379"
	@echo "  Jaeger UI: http://localhost:16686"
	@echo "  Prometheus: http://localhost:9090"

dev-down:
	@echo "Stopping development environment..."
	docker compose -f docker-compose.dev.yml down

dev-restart: dev-down dev-up

# Run services locally with development infrastructure
run-api-gateway: build-api-gateway dev-db-up
	@echo "Running API Gateway with development database..."
	CONFIG_PATH=$(CONFIG_DIR)/config.yaml $(API_GATEWAY_BINARY)

run-user-service: build-user-service dev-up
	@echo "Running User Service with full development environment..."
	CONFIG_PATH=$(CONFIG_DIR)/config-full.yaml $(USER_SERVICE_BINARY)

# Database migrations (requires running database)
migrate-up: build-user-service dev-db-up
	@echo "Running database migrations up..."
	@sleep 2
	$(USER_SERVICE_BINARY) migrate up || echo "Migration completed or already up to date"

migrate-down: build-user-service dev-db-up
	@echo "Running database migrations down..."
	@sleep 2
	$(USER_SERVICE_BINARY) migrate down || echo "Migration completed"

# Docker commands for full infrastructure
docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-up:
	@echo "Starting full infrastructure..."
	docker-compose up -d

docker-down:
	@echo "Stopping full infrastructure..."
	docker-compose down

docker-logs:
	@echo "Showing infrastructure logs..."
	docker-compose logs -f

# Development workflow helpers
dev-setup: deps dev-up
	@echo "Development environment setup complete!"
	@echo "Run 'make test-integration' to verify everything works"

dev-test: dev-up test-integration
	@echo "Development testing complete!"

dev-clean: dev-down clean
	@echo "Development environment cleaned up!"

# Code quality
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Code coverage
coverage:
	@echo "Generating test coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build              - Build all services"
	@echo "  build-api-gateway  - Build API Gateway service"
	@echo "  build-user-service - Build User Service"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Download dependencies"
	@echo ""
	@echo "Testing Commands:"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests"
	@echo "  test-integration   - Run integration tests (starts dev DB automatically)"
	@echo "  coverage           - Generate test coverage report"
	@echo ""
	@echo "Development Environment:"
	@echo "  dev-up             - Start full development environment (PostgreSQL, Redis, Jaeger, Prometheus)"
	@echo "  dev-down           - Stop development environment"
	@echo "  dev-restart        - Restart development environment"
	@echo "  dev-db-up          - Start only databases (PostgreSQL, Redis)"
	@echo "  dev-db-down        - Stop only databases"
	@echo "  dev-db-logs        - Show database logs"
	@echo "  dev-setup          - Complete development setup"
	@echo "  dev-test           - Run full development test suite"
	@echo "  dev-clean          - Clean up development environment"
	@echo ""
	@echo "Service Execution:"
	@echo "  run-api-gateway    - Run API Gateway with development database"
	@echo "  run-user-service   - Run User Service with full development environment"
	@echo ""
	@echo "Database Migrations:"
	@echo "  migrate-up         - Run database migrations up (starts DB if needed)"
	@echo "  migrate-down       - Run database migrations down"
	@echo ""
	@echo "Production Infrastructure:"
	@echo "  docker-build       - Build Docker images"
	@echo "  docker-up          - Start full infrastructure stack"
	@echo "  docker-down        - Stop full infrastructure stack"
	@echo "  docker-logs        - Show infrastructure logs"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo ""
	@echo "Quick Start for Development:"
	@echo "  make dev-setup     - Set up everything for development"
	@echo "  make run-user-service - Start User Service with all dependencies"

# Development setup
setup-dev: ## Setup development environment
	@echo "Setting up development environment..."
	./scripts/setup/setup-dev.sh
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Build targets
build: ## Build all services
	@echo "Building all services..."
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$$service ./cmd/$$service; \
	done

build-docker: ## Build Docker images for all services
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -f build/docker/$$service.Dockerfile -t $(DOCKER_REPO)/$$service:$(VERSION) .; \
	done

# Testing
test: ## Run all tests
	@echo "\nRunning all tests..."
	@go test -v -race -coverprofile=coverage.out ./... | tee test_output.log
	# Cleanup containers after tests
	-docker compose -f docker-compose.dev.yml down -v
	@echo "\n\033[1mTest Results Summary:\033[0m"
	@echo "----------------------------------------"
	@grep -e '--- PASS:' -e '--- FAIL:' -e '--- SKIP:' test_output.log | sed -E 's/--- PASS:/\033[32m| PASS |\033[0m/; s/--- FAIL:/\033[31m| FAIL |\033[0m/; s/--- SKIP:/\033[33m| SKIP |\033[0m/' | column -t -s'|'
	@echo "----------------------------------------"
	@rm -f test_output.log

test-unit: ## Run unit tests only
	go test -v -short -race ./...

test-integration: ## Run integration tests
	go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	go test -v -tags=e2e ./tests/e2e/...

test-coverage: test ## Generate test coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Load testing
load-test: ## Run load tests with k6
	@echo "Running load tests..."
	docker run --rm -i grafana/k6:latest run - < tests/load/k6/user-load-test.js

load-test-all: ## Run all load test scenarios
	@echo "Running comprehensive load tests..."
	./scripts/testing/load-test.sh

# Code quality
lint: ## Run linters
	golangci-lint run --config .golangci.yml

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

# Protocol Buffers
proto-gen: ## Generate gRPC code from proto files
	@echo "Generating gRPC code..."
	@for proto in proto/*/*.proto; do \
		protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			$$proto; \
	done

# Database operations
migrate-up: ## Apply database migrations
	@echo "Migrating user database..."
	@$(HOME)/go/bin/migrate -path migrations \
		-database "postgres://commercium_user:commercium_password@localhost:5432/commercium_db?sslmode=disable" up
	@echo "Migrating test database..."
	@$(HOME)/go/bin/migrate -path migrations \
		-database "postgres://commercium_user:commercium_password@localhost:5432/commercium_test_db?sslmode=disable" up

migrate-down: ## Rollback database migrations
	@echo "Rolling back user database..."
	@$(HOME)/go/bin/migrate -path migrations \
		-database "postgres://commercium_user:commercium_password@localhost:5432/commercium_db?sslmode=disable" down
	@echo "Rolling back test database..."
	@$(HOME)/go/bin/migrate -path migrations \
		-database "postgres://commercium_user:commercium_password@localhost:5432/commercium_test_db?sslmode=disable" down

migrate-create: ## Create new migration (usage: make migrate-create SERVICE=user NAME=create_users_table)
	@if [ -z "$(SERVICE)" ] || [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create SERVICE=user NAME=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir internal/$(SERVICE)-service/infrastructure/database/migrations -seq $(NAME)

# Local development
run-infrastructure: ## Start infrastructure services (postgres, redis, kafka, etc.)
	docker-compose up -d postgres redis kafka rabbitmq elasticsearch kibana prometheus grafana

run-api-gateway: ## Run API Gateway
	go run cmd/api-gateway/main.go

run-user-service: ## Run User Service
	go run cmd/user-service/main.go

run-product-service: ## Run Product Service
	go run cmd/product-service/main.go

run-order-service: ## Run Order Service
	go run cmd/order-service/main.go

run-payment-service: ## Run Payment Service
	go run cmd/payment-service/main.go

run-inventory-service: ## Run Inventory Service
	go run cmd/inventory-service/main.go

run-notification-service: ## Run Notification Service
	go run cmd/notification-service/main.go

run-all: ## Run all services (in separate terminals)
	@echo "Starting all services..."
	@echo "Make sure to run 'make run-infrastructure' first"
	@echo "Starting services in background..."
	@for service in $(SERVICES); do \
		echo "Starting $$service..."; \
		go run cmd/$$service/main.go > logs/$$service.log 2>&1 & \
	done
	@echo "All services started. Check logs/ directory for service logs"

# Kubernetes deployment
deploy-dev: ## Deploy to development environment
	@echo "Deploying to development..."
	kubectl apply -k deployments/kubernetes/overlays/development -n $(NAMESPACE)-dev

deploy-staging: ## Deploy to staging environment
	@echo "Deploying to staging..."
	kubectl apply -k deployments/kubernetes/overlays/staging -n $(NAMESPACE)-staging

deploy-prod: ## Deploy to production environment
	@echo "Deploying to production..."
	kubectl apply -k deployments/kubernetes/overlays/production -n $(NAMESPACE)-prod

# Helm deployment
helm-install: ## Install with Helm
	helm install ecommerce-platform deployments/helm/ecommerce-platform/ -n $(NAMESPACE)

helm-upgrade: ## Upgrade with Helm
	helm upgrade ecommerce-platform deployments/helm/ecommerce-platform/ -n $(NAMESPACE)

# Monitoring
setup-monitoring: ## Setup monitoring stack
	@echo "Setting up monitoring..."
	kubectl apply -f deployments/kubernetes/base/prometheus/
	kubectl apply -f deployments/kubernetes/base/grafana/

# Security
setup-vault: ## Setup HashiCorp Vault
	@echo "Setting up Vault..."
	./scripts/setup/setup-vault.sh

generate-certs: ## Generate TLS certificates
	@echo "Generating TLS certificates..."
	./security/tls/generate-certs.sh

# Docker Compose operations
up: ## Start all services with Docker Compose
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## View logs from all services
	docker-compose logs -f

# Cleanup
clean: ## Clean build artifacts and temporary files
	rm -rf bin/
	rm -rf logs/
	rm -f coverage.out coverage.html
	go clean -testcache
	docker system prune -f

clean-all: clean ## Deep clean (including Docker images and volumes)
	docker-compose down -v --remove-orphans
	docker system prune -af --volumes

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	go doc -all ./... > docs/api-reference.md

# Git hooks
install-hooks: ## Install git hooks
	cp scripts/git-hooks/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit

# Health checks
health-check: ## Check health of all services
	@echo "Checking service health..."
	@for port in 8080 8081 8082 8083 8084 8085 8086; do \
		echo "Checking service on port $$port..."; \
		curl -f http://localhost:$$port/health || echo "Service on port $$port is down"; \
	done

setup: ## Setup infrastructure and run all tests
	make dev-db-up
	make migrate-up
	make test
