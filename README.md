# OrderForge

Production-grade order management backend built with Go, following clean architecture principles.

## Architecture

```
cmd/server/          Entry point, wiring, graceful shutdown
internal/
  domain/            Entities, value objects, domain errors, repository interfaces
  application/       Use cases (CreateOrder, CancelOrder, ApplyDiscount, etc.)
  infrastructure/    PostgreSQL repositories, Redis cache, NATS event publisher
  api/
    handler/         HTTP handlers (chi router)
    middleware/      Auth, logging, panic recovery
  config/            Environment-based configuration
migrations/          SQL schema
```

The dependency rule flows inward: handlers depend on use cases, use cases depend on domain interfaces, infrastructure implements those interfaces. No package imports from an outer layer.

## Stack

- **Go 1.22** with `log/slog` structured logging
- **PostgreSQL 16** for persistent storage
- **Redis 7** for product cache (15 min TTL)
- **NATS** for event publishing on order state transitions
- **OpenTelemetry** tracing on every use case
- **chi** router with middleware chain

## Order Lifecycle

```
draft -> confirmed -> processing -> shipped -> delivered
  |         |             |
  +----+----+-------------+
       |
   cancelled
```

Cancellation releases reserved inventory back to available stock.

## API

All endpoints except health checks require `Authorization: Bearer <token>`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe (checks DB) |
| POST | `/orders` | Create a draft order |
| GET | `/orders/{id}` | Get order by ID |
| POST | `/orders/{id}/cancel` | Cancel an order |
| POST | `/orders/{id}/discount` | Apply a coupon |
| POST | `/orders/{id}/transition` | Advance order status |
| GET | `/customers/{id}/orders` | Order history with pagination |

### Create Order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer mytoken" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "items": [
      {"product_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "quantity": 2}
    ]
  }'
```

### Apply Discount

```bash
curl -X POST http://localhost:8080/orders/{id}/discount \
  -H "Authorization: Bearer mytoken" \
  -H "Content-Type: application/json" \
  -d '{"coupon_code": "SAVE20"}'
```

### Transition Status

```bash
curl -X POST http://localhost:8080/orders/{id}/transition \
  -H "Authorization: Bearer mytoken" \
  -H "Content-Type: application/json" \
  -d '{"status": "confirmed"}'
```

## Running

### With Docker Compose

```bash
docker compose up --build
```

This starts the app with PostgreSQL, Redis, and NATS. The database schema is applied on first boot.

### Local Development

```bash
# Start dependencies
docker compose up postgres redis nats -d

# Build and run
make run

# Run tests
make test

# Lint
make lint
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP listen port |
| `DATABASE_URL` | `postgres://orderforge:orderforge@localhost:5432/orderforge?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection string |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `LOG_LEVEL` | info | Log verbosity |

## Tests

```bash
make test
```

Tests cover domain logic (order transitions, inventory reservation, coupon validation), use cases (create, cancel, discount application), middleware (auth extraction), and health endpoints.

## License

MIT
