# Order Service

A microservice ecosystem for managing orders and products, featuring order management built with Go and product management built with NestJS, with event-driven communication, caching, and real-time inventory management.

## 🏗️ Architecture Overview

The Order Service follows a clean architecture pattern with the following components:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Order API     │    │   Product API   │    │   RabbitMQ      │
│   (Go/REST)     │    │  (NestJS/REST)  │    │  (Messaging)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Event-Driven Architecture                     │
│  • Order Events    • Product Events    • Inventory Management  │
└─────────────────────────────────────────────────────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Order Service  │    │ Product Service │    │     Redis       │
│  (Repository)   │    │  (Repository)   │    │    (Cache)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Components:

#### Order Service (Go)
- **Service Layer**: Core business logic for order management
- **Repository Pattern**: Data access abstraction
- **Event Publishing**: Publishes `order.created` events to RabbitMQ
- **Product Client**: Integration with Product Service for validation
- **Caching**: Redis for product data caching

#### Product Service (NestJS)
- **Product Management**: CRUD operations for products
- **Inventory Management**: Automatic quantity decrementing on orders
- **Event Handling**: Listens to `order.created` events
- **Event Publishing**: Publishes `order.qty_confirmed` and `order.qty_failed` events
- **Caching**: Product data caching with Redis integration

#### Shared Infrastructure
- **Event-Driven Architecture**: Asynchronous communication via RabbitMQ
- **Clean Architecture**: Separation of concerns with dependency injection
- **Database**: MySQL for both services
- **Caching**: Shared Redis instance

### Features:

#### Order Service Features:
- ✅ Order creation with product validation
- ✅ Product caching for performance optimization
- ✅ Asynchronous event publishing
- ✅ Repository pattern for data persistence
- ✅ Comprehensive unit testing with mocks
- ✅ Docker containerization

#### Product Service Features:
- ✅ Product CRUD operations
- ✅ Real-time inventory management
- ✅ Automatic quantity decrementing on orders
- ✅ Event-driven order processing
- ✅ Product caching with Redis
- ✅ Comprehensive error handling and logging

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
- Order Service API (port 8080) - Go
- Product Service API (port 3000) - NestJS
- MySQL Database (port 3306)
- Redis Cache (port 6379)
- RabbitMQ Message Broker (port 5672, Management UI: 15672)

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
docker-compose up -d mysql redis rabbitmq product-service

# Run the order service locally for development
go run cmd/main.go

# Or run the product service locally for development
cd product-service && npm run start:dev
```

### Environment Configuration

Key environment variables (see `docker-compose.yml`):

```bash
# Database
DB_HOST=mysql
DB_PORT=3306
DB_USER=orderuser
DB_PASSWORD=orderpass
DB_NAME=orderdb

# Redis
REDIS_HOST=redis:6379

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Product Service
PRODUCT_SERVICE_URL=http://product-service:3000
```

## 📚 API Documentation

### Order Service API (Port 8080)

#### Base URL
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

---

### Product Service API (Port 3000)

#### Base URL
```
http://localhost:3000
```

#### Endpoints

#### 1. Create Product

Create a new product with initial inventory.

**Request:**
```bash
curl -X POST http://localhost:3000/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Smartphone",
    "price": 699.99,
    "qty": 100
  }'
```

**Response (201 Created):**
```json
{
  "id": 1,
  "name": "Smartphone",
  "price": 699.99,
  "qty": 100,
  "created_at": "2025-09-20T10:30:00Z",
  "updated_at": "2025-09-20T10:30:00Z"
}
```

#### 2. Get Product by ID

Retrieve a specific product by its ID (with caching).

**Request:**
```bash
curl -X GET http://localhost:3000/products/1
```

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "Smartphone",
  "price": 699.99,
  "qty": 99,
  "created_at": "2025-09-20T10:30:00Z",
  "updated_at": "2025-09-20T10:30:00Z"
}
```

**Response (404 Not Found):**
```json
{
  "statusCode": 404,
  "message": "Product 1 not found",
  "error": "Not Found"
}
```

### Event-Driven Communication

The services communicate through RabbitMQ events:

#### Events Published by Order Service:
- `order.created`: When a new order is created
  ```json
  {
    "orderId": 1,
    "productId": 123,
    "totalPrice": 699.99,
    "timestamp": "2025-09-20T10:30:00Z"
  }
  ```

#### Events Published by Product Service:
- `order.qty_confirmed`: When inventory is successfully decremented
  ```json
  {
    "orderId": 1,
    "timestamp": "2025-09-20T10:30:00Z"
  }
  ```

- `order.qty_failed`: When inventory is insufficient or product not found
  ```json
  {
    "orderId": 1,
    "reason": "product_not_found_or_unavailable",
    "timestamp": "2025-09-20T10:30:00Z"
  }
  ```

- `product_created`: When a new product is created
  ```json
  {
    "id": 1,
    "name": "Smartphone",
    "price": 699.99,
    "qty": 100,
    "timestamp": "2025-09-20T10:30:00Z"
  }
  ```

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

### Complete Order Flow with Inventory Management

1. **Create a product first:**
```bash
curl -X POST http://localhost:3000/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gaming Laptop",
    "price": 2999.99,
    "qty": 5
  }'
```

2. **Create an order:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "total_price": 2999.99
  }'
```

3. **Verify order creation:**
```bash
curl -X GET http://localhost:8080/orders/1
```

4. **Check product inventory (should be decremented):**
```bash
curl -X GET http://localhost:3000/products/1
# qty should now be 4
```

5. **Monitor RabbitMQ events:**
- Visit RabbitMQ Management UI: http://localhost:15672
- Login: guest/guest
- Check queues for event flow:
  - `order.created` → `order.qty_confirmed`

6. **Verify product caching:**
```bash
# Connect to Redis
docker exec -it order-service-redis-1 redis-cli

# Check cached product
GET product:1
```

### Error Handling Flow

1. **Try to order a non-existent product:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 999,
    "total_price": 1000
  }'
```

2. **Create product with no inventory:**
```bash
curl -X POST http://localhost:3000/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Out of Stock Item",
    "price": 99.99,
    "qty": 0
  }'
```

3. **Try to order out of stock product:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 2,
    "total_price": 99.99
  }'
```

4. **Check for `order.qty_failed` event in RabbitMQ**

### Development Workflow

1. **Make changes to the code**

2. **Run tests:**
```bash
# Order Service (Go)
go test ./internal/services -v

# Product Service (NestJS)
cd product-service && npm test
```

3. **Build and restart services:**
```bash
# Restart specific service
docker-compose build order-service
docker-compose up -d order-service

# Or restart product service
docker-compose build product-service
docker-compose up -d product-service
```

4. **Test the integration:**
```bash
# Create product
curl -X POST http://localhost:3000/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Product", "price": 100, "qty": 10}'

# Create order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "total_price": 100}'

# Verify inventory decreased
curl -X GET http://localhost:3000/products/1
```

## 🐳 Docker Services

The stack includes these services:

- **order-service**: Order management API (Go)
- **product-service**: Product management API (NestJS)
- **mysql**: MySQL database (shared)
- **redis**: Redis cache (shared)
- **rabbitmq**: RabbitMQ message broker

### Accessing Service UIs

- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Order Service API**: http://localhost:8080
- **Product Service API**: http://localhost:3000

## 🔧 Configuration

### Database Migrations

Migrations run automatically on startup. Manual migration:

```bash
# Run migrations
docker exec order-service-app migrate -path ./migrations -database "mysql://orderuser:orderpass@tcp(mysql:3306)/orderdb" up

# Rollback
docker exec order-service-app migrate -path ./migrations -database "mysql://orderuser:orderpass@tcp(mysql:3306)/orderdb" down 1
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
   - Ensure MySQL is running: `docker-compose ps mysql`
   - Check database logs: `docker-compose logs mysql`
   - Test connection: `docker exec mysql mysql -u orderuser -porderpass -e "SHOW DATABASES;"`

3. **Product service unavailable:**
   - Verify product service: `curl http://localhost:3000/products/1`
   - Check network connectivity between containers
   - View product service logs: `docker-compose logs product-service`

4. **RabbitMQ connection issues:**
   - Check RabbitMQ status: `docker-compose logs rabbitmq`
   - Verify connection: `docker exec rabbitmq rabbitmqctl status`

5. **Event processing issues:**
   - Check RabbitMQ queues: http://localhost:15672
   - Verify event patterns in product service logs
   - Test event publishing manually

6. **Inventory synchronization issues:**
   - Check product quantities: `curl http://localhost:3000/products/1`
   - Verify cache consistency: Redis CLI `GET product:1`
   - Check for failed order events in RabbitMQ

### Logs and Debugging

```bash
# View all logs
docker-compose logs

# Follow specific service logs
docker-compose logs -f order-service
docker-compose logs -f product-service

# Enter containers for debugging
docker exec -it order-service-app sh
docker exec -it product-service-app sh

# Check MySQL database
docker exec -it mysql mysql -u orderuser -porderpass orderdb

# Check RabbitMQ queue contents
docker exec rabbitmq rabbitmqctl list_queues

# Monitor Redis cache
docker exec -it redis-container redis-cli monitor
```

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.
