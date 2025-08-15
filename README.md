# E-Commerce Platform

A production-ready e-commerce backend built with Go, featuring microservices architecture, event-driven communication, and comprehensive observability.

## 🚀 **Implementation Status**

### Phase 1: Core Infrastructure & Services ✅ **COMPLETED**
- [x] Project structure and configuration
- [x] Go module initialization with proper naming (github.com/kaanevranportfolio/Commercium)
- [x] Docker Compose infrastructure setup
- [x] Basic API Gateway with HTTP endpoints (/health, /readiness, /status, /metrics)
- [x] Structured logging with Zap
- [x] Prometheus metrics collection
- [x] Distributed tracing setup (OpenTelemetry)
- [x] Configuration management with Viper
- [x] Integration tests for Phase 1 functionality

**✅ Phase 1 Testing Results:**
- Build: ✅ Successful
- Service startup: ✅ Successful
- Health endpoints: ✅ Working (/health, /readiness, /status)
- Metrics endpoint: ✅ Working (/metrics with Prometheus format)
- Integration tests: ✅ All passing (4/4 test cases)

### Phase 2: Database Integration & Basic Services ✅ **COMPLETED**
- [x] PostgreSQL integration with connection pooling
- [x] Database migrations system  
- [x] Redis setup for caching and session management
- [x] User Service (authentication, JWT, user management)
- [x] Complete REST API with 15 endpoints
- [x] JWT authentication with access/refresh tokens
- [x] Password security with bcrypt hashing
- [x] Email verification and password reset flows
- [x] User profile and address management
- [x] Integration tests and error handling
- [x] Repository pattern and clean architecture

### Phase 3: Event-Driven Architecture 📋 **PLANNED**
- [ ] Kafka integration and event schemas
- [ ] RabbitMQ integration for task queuing
- [ ] Order Service with order state management
- [ ] Payment Service with payment processing simulation
- [ ] Inventory Service with real-time stock updates
- [ ] Notification Service for order updates

### Phase 4: Security Implementation 🔒 **PLANNED**
- [ ] JWT-based authentication
- [ ] HashiCorp Vault for secrets management
- [ ] TLS/mTLS between services
- [ ] Rate limiting and throttling
- [ ] Input validation and security headers

### Phase 4: Security Implementation 🔒 **PLANNED**
- [ ] JWT-based authentication
- [ ] HashiCorp Vault for secrets management
- [ ] TLS/mTLS between services
- [ ] Rate limiting and throttling
- [ ] Input validation and security headers

### Phase 5: Observability & Monitoring 📊 **PLANNED**
- [ ] ELK stack deployment
- [ ] Prometheus metrics collection
- [ ] Grafana dashboards
- [ ] Distributed tracing with Jaeger
- [ ] Alerting rules and notifications

### Phase 6: Container Orchestration ☸️ **PLANNED**
- [ ] Kubernetes deployment manifests
- [ ] Helm charts
- [ ] Health checks and readiness probes
- [ ] Horizontal Pod Autoscaling
- [ ] Service mesh consideration

### Phase 7: CI/CD Pipeline 🔄 **PLANNED**
- [ ] GitHub Actions workflows
- [ ] Automated testing pipeline
- [ ] Security and dependency scanning
- [ ] Multi-environment deployments
- [ ] GitOps workflow

### Phase 8: Testing & Load Testing 🧪 **PLANNED**
- [ ] Comprehensive test suites
- [ ] Load testing with k6
- [ ] Security testing with OWASP ZAP
- [ ] Chaos engineering
- [ ] Performance benchmarking

## Architecture

- **7 Microservices**: API Gateway, User, Product, Order, Payment, Inventory, Notification
- **Communication**: gRPC (sync), Kafka (events), RabbitMQ (tasks), GraphQL (client)
- **Databases**: PostgreSQL, Redis, Elasticsearch
- **Observability**: ELK Stack, Prometheus, Grafana, Jaeger
- **Security**: Vault, JWT, TLS/mTLS, RBAC

## 🚀 Quick Start for Development

### Prerequisites
- Go 1.21 or later
- Docker and Docker Compose
- Git

### Local Development Setup

1. **Clone and Setup**
   ```bash
   git clone <repository>
   cd ecommerce-platform
   make dev-setup
   ```

2. **Start User Service**
   ```bash
   make run-user-service
   ```
   This automatically starts PostgreSQL, Redis, Jaeger, and Prometheus, then runs the User Service.

3. **Test the API**
   ```bash
   # Register a user
   curl -X POST http://localhost:8080/api/v1/auth/register \
     -H "Content-Type: application/json" \
     -d '{"username":"testuser","email":"test@example.com","password":"Password123!"}'

   # Login
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"testuser","password":"Password123!"}'
   ```

### Development Infrastructure

The development environment uses Docker Compose to provide:

| Service | URL | Purpose |
|---------|-----|---------|
| PostgreSQL | `localhost:5432` | Primary database (dev + test DBs) |
| Redis | `localhost:6379` | Caching and session storage |
| Jaeger | `http://localhost:16686` | Distributed tracing UI |
| Prometheus | `http://localhost:9090` | Metrics collection |

**Database Configuration:**
- **Development DB**: `commercium_db` 
- **Test DB**: `commercium_test_db` (automatically created)
- **User**: `commercium_user`
- **Password**: `commercium_password`

### Development Workflow

```bash
# Start only databases for lightweight development
make dev-db-up

# Run integration tests (automatically starts required services)
make test-integration

# Full development environment
make dev-up

# View logs
make dev-db-logs

# Clean shutdown
make dev-down
```

### Testing Approach

The project uses a **multi-tier testing strategy**:

1. **Unit Tests**: Fast tests with no external dependencies
   ```bash
   make test-unit
   ```

2. **Integration Tests**: Full API testing with real database
   ```bash
   make test-integration  # Automatically starts dev database
   ```

3. **Local Service Testing**: Run services against real infrastructure
   ```bash
   make run-user-service  # Starts service with PostgreSQL, Redis, etc.
   ```

**Test Database Strategy:**
- Integration tests use `commercium_test_db` (separate from development data)
- Tests automatically skip if infrastructure is unavailable (CI-friendly)
- Database is reset between test runs for consistency

### Configuration Management

The project supports environment-specific configurations:

- `configs/config.yaml` - Basic development config
- `configs/config-full.yaml` - Complete config with all services
- Docker Compose automatically configures service connectivity

**Environment Variables:**
```bash
CONFIG_PATH=configs/config-full.yaml  # Override config file
```

## Project Structure

```
├── cmd/                    # Application entry points
├── internal/               # Private application code
├── pkg/                    # Shared libraries
├── proto/                  # gRPC definitions
├── deployments/           # Kubernetes & Docker configs
├── monitoring/            # Observability configs
├── tests/                 # All types of tests
└── scripts/               # Utility scripts
```

## Development Workflow

### 1. Build & Test
```bash
make build          # Build all services
make test           # Run all tests
make test-coverage  # Generate coverage report
make lint          # Run linters
```

### 2. Proto Generation
```bash
make proto-gen     # Generate gRPC code from proto files
```

### 3. Database Migrations
```bash
make migrate-up    # Apply migrations
make migrate-down  # Rollback migrations
```

### 4. Load Testing
```bash
make load-test     # Run k6 load tests
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| API Gateway | 8080 | GraphQL endpoint, routing |
| User Service | 8081 | Authentication, user management |
| Product Service | 8082 | Product catalog, search |
| Order Service | 8083 | Order processing, saga orchestration |
| Payment Service | 8084 | Payment processing |
| Inventory Service | 8085 | Stock management |
| Notification Service | 8086 | Email, SMS, push notifications |

## API Documentation

- **GraphQL Schema**: `/docs/api/graphql-schema.md`
- **gRPC APIs**: `/docs/api/grpc-apis.md`

## Deployment

### Kubernetes
```bash
# Deploy to development
make deploy-dev

# Deploy to production
make deploy-prod
```

### Docker Compose
```bash
# Production-like environment
docker-compose -f docker-compose.prod.yml up
```

## Monitoring

- **Metrics**: Prometheus scrapes metrics from all services
- **Dashboards**: Pre-configured Grafana dashboards
- **Logging**: Centralized logging via ELK stack
- **Tracing**: Distributed tracing with Jaeger
- **Alerts**: Prometheus AlertManager for critical issues

## Security

- **Secrets Management**: HashiCorp Vault
- **Authentication**: JWT with Redis blacklisting
- **Authorization**: Role-based access control (RBAC)
- **Network Security**: TLS/mTLS between services
- **Scanning**: Automated security scanning in CI/CD

## Testing

- **Unit Tests**: `*_test.go` files alongside source code
- **Integration Tests**: `/tests/integration/`
- **E2E Tests**: `/tests/e2e/`
- **Load Tests**: `/tests/load/` (k6 and JMeter)
- **Security Tests**: OWASP ZAP configurations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test lint`
6. Submit a pull request

## Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=ecommerce
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Kafka
KAFKA_BROKERS=localhost:9092

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Vault
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-token

# JWT
JWT_SECRET_KEY=your-secret-key
JWT_EXPIRATION=24h
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

For questions and support:
- Check the [documentation](docs/)
- Review [troubleshooting guide](docs/runbooks/troubleshooting.md)
- Open an issue for bugs
- Start a discussion for questions
