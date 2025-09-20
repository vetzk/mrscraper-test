# Order Service

A microservice for managing orders built with Go, featuring product validation, event publishing, and caching capabilities.

## 🏗️ Architecture Overview

The Order Service follows a clean architecture pattern with the following components:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Layer     │    │  Product Client │    │   RabbitMQ      │
│  (HTTP/REST)    │    │   (External)    │    │  (Messaging)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Service Layer                               │
│  • Order Creation Logic    • Validation    • Event Publishing  │
└─────────────────────────────────────────────────────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│   Repository    │    │     Redis       │
│   (Database)    │    │    (Cache)      │
└─────────────────┘    └─────────────────┘
```

### Key Components:

- **Service Layer**: Core business logic for order management
- **Repository Pattern**: Data access abstraction
- **Product Client**: Integration with external product service
- **Event Publishing**: Asynchronous messaging via RabbitMQ
- **Caching**: Redis for product data caching
- **Clean Architecture**: Separation of concerns with dependency injection

### Features:

- ✅ Order creation with product validation
- ✅ Product caching for performance optimization
- ✅ Asynchronous event publishing
- ✅ Repository pattern for data persistence
- ✅ Comprehensive unit testing with mocks
- ✅ Docker containerization

## 🚀 Running the Stack Locally

### Prerequisites

- Docker and Docker Compose installed
- Git

### Quick Start

1. **Clone the repository**
```bash
git clone <repository-url>
cd order-service
```

2. **Start the entire stack**
```bash
docker-compose up -d
```

This will start:
- Order Service API (port 8080)
- PostgreSQL Database (port 5432)
- Redis Cache (port 6379)
- RabbitMQ Message Broker (port 5672, Management UI: 15672)
- Product Service Mock (port 8081)

3. **Verify services are running**
```bash
docker-compose ps
```

4. **View logs**
```bash
docker-compose logs -f order-service
```

5. **Stop the stack**
```bash
docker-compose down
```

### Development Mode

To run with code reloading during development:

```bash
# Run dependencies only
docker-compose up -d postgres redis rabbitmq product-service-mock

# Run the service locally
go run cmd/main.go
```

### Environment Configuration

Key environment variables (see `docker-compose.yml`):

```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=orderuser
DB_PASSWORD=orderpass
DB_NAME=orderdb

# Redis
REDIS_HOST=redis:6379

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Product Service
PRODUCT_SERVICE_URL=http://product-service-mock:8081
```

## 📚 API Documentation

### Base URL
```
http://localhost:8080
```

### Endpoints

#### 1. Create Order

Create a new order for a product.

**Request:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "total_price": 1500
  }'
```

**Response (201 Created):**
```json
{
  "id": 1,
  "product_id": 1,
  "total_price": 1500,
  "status": "pending",
  "created_at": "2025-09-20T10:30:00Z"
}
```

#### 2. Get Order by ID

Retrieve a specific order by its ID.

**Request:**
```bash
curl -X GET http://localhost:8080/orders/1
```

**Response (200 OK):**
```json
{
  "id": 1,
  "product_id": 1,
  "total_price": 1500,
  "status": "pending",
  "created_at": "2025-09-20T10:30:00Z"
}
```

**Response (404 Not Found):**
```json
{
  "error": "order not found"
}
```

#### 3. Get Orders by Product ID

Retrieve all orders for a specific product.

**Request:**
```bash
curl -X GET http://localhost:8080/orders/product/1
```

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "product_id": 1,
    "total_price": 1500,
    "status": "pending",
    "created_at": "2025-09-20T10:30:00Z"
  },
  {
    "id": 2,
    "product_id": 1,
    "total_price": 2000,
    "status": "confirmed",
    "created_at": "2025-09-20T11:00:00Z"
  }
]
```

#### 4. Health Check

Check service health and dependencies.

**Request:**
```bash
curl -X GET http://localhost:8080/health
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-20T10:30:00Z",
  "services": {
    "database": "up",
    "redis": "up",
    "product_service": "up",
    "rabbitmq": "up"
  }
}
```

### Error Responses

The API returns standard HTTP status codes and JSON error responses:

```json
{
  "error": "error description",
  "code": "ERROR_CODE",
  "timestamp": "2025-09-20T10:30:00Z"
}
```

Common status codes:
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## 🧪 Testing

### Run Unit Tests
```bash
go test ./internal/services -v
```

### Run Integration Tests
```bash
go test ./tests/integration -v
```

### Run All Tests with Coverage
```bash
go test ./... -cover
```

## 📋 Example Workflows

### Complete Order Flow

1. **Create an order:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 123,
    "total_price": 2999
  }'
```

2. **Verify order creation:**
```bash
curl -X GET http://localhost:8080/orders/1
```

3. **Check RabbitMQ for published events:**
- Visit RabbitMQ Management UI: http://localhost:15672
- Login: guest/guest
- Check queues for `order.created` events

4. **Verify product caching:**
```bash
# Connect to Redis
docker exec -it order-service-redis-1 redis-cli

# Check cached product
GET product:123
```

### Development Workflow

1. **Make changes to the code**

2. **Run tests:**
```bash
go test ./internal/services -v
```

3. **Build and restart:**
```bash
docker-compose build order-service
docker-compose up -d order-service
```

4. **Test the changes:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "total_price": 1000}'
```

## 🐳 Docker Services

The stack includes these services:

- **order-service**: Main application (Go)
- **postgres**: PostgreSQL database
- **redis**: Redis cache
- **rabbitmq**: RabbitMQ message broker
- **product-service-mock**: Mock product service for testing

### Accessing Service UIs

- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Order Service API**: http://localhost:8080
- **Product Service Mock**: http://localhost:8081

## 🔧 Configuration

### Database Migrations

Migrations run automatically on startup. Manual migration:

```bash
# Run migrations
docker exec order-service-app migrate -path ./migrations -database "postgres://orderuser:orderpass@postgres:5432/orderdb?sslmode=disable" up

# Rollback
docker exec order-service-app migrate -path ./migrations -database "postgres://orderuser:orderpass@postgres:5432/orderdb?sslmode=disable" down 1
```

### Cache Warmup

Warm up product cache for better performance:

```bash
curl -X POST http://localhost:8080/admin/cache/warmup \
  -H "Content-Type: application/json" \
  -d '{
    "product_ids": [1, 2, 3, 4, 5]
  }'
```

## 🐛 Troubleshooting

### Common Issues

1. **Service won't start:**
   - Check if ports are available: `netstat -tulpn | grep :8080`
   - View logs: `docker-compose logs order-service`

2. **Database connection failed:**
   - Ensure PostgreSQL is running: `docker-compose ps postgres`
   - Check database logs: `docker-compose logs postgres`

3. **Product service unavailable:**
   - Verify product service: `curl http://localhost:8081/products/1`
   - Check network connectivity between containers

4. **RabbitMQ connection issues:**
   - Check RabbitMQ status: `docker-compose logs rabbitmq`
   - Verify connection: `docker exec rabbitmq rabbitmqctl status`

### Logs and Debugging

```bash
# View all logs
docker-compose logs

# Follow specific service logs
docker-compose logs -f order-service

# Enter container for debugging
docker exec -it order-service-app sh
```

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.
