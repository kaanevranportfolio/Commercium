.PHONY: help build test clean proto-gen setup-dev run-all deploy-dev deploy-prod

# Variables
SERVICES := api-gateway user-service product-service order-service payment-service inventory-service notification-service
DOCKER_REPO := your-docker-registry
VERSION := $(shell git describe --tags --always --dirty)
NAMESPACE := ecommerce-platform

# Default target
help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

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
	go test -v -race -coverprofile=coverage.out ./...

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
	@for service in user product order payment inventory notification; do \
		echo "Migrating $$service database..."; \
		migrate -path internal/$$service-service/infrastructure/database/migrations \
			-database "postgres://postgres:password@localhost/$$service?sslmode=disable" up; \
	done

migrate-down: ## Rollback database migrations
	@for service in user product order payment inventory notification; do \
		echo "Rolling back $$service database..."; \
		migrate -path internal/$$service-service/infrastructure/database/migrations \
			-database "postgres://postgres:password@localhost/$$service?sslmode=disable" down; \
	done

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
