# Todo API

A RESTful Todo/Memo API built with Go, SQLite, and JWT authentication.

## Quick Start

```bash
# Install dependencies
go mod tidy

# Run the server (default :8080)
go run cmd/main.go

# Or with custom settings
JWT_SECRET=my-secret DB_PATH=data.db ADDR=:3000 go run cmd/main.go
```

## Environment Variables

| Variable | Default | Description |
|------------|---------------------------|-----------------------------|
| JWT_SECRET | change-me-in-production | HMAC signing key for JWT |
| DB_PATH | todo.db | SQLite database file path |
| ADDR | :8080 | HTTP listen address |

## API Endpoints

### Public

| Method | Path | Description |
|--------|----------------|---------------------|
| POST | /api/register | Register a new user |
| POST | /api/login | Login, returns JWT |

### Protected (requires `Authorization: Bearer <token>`)

| Method | Path | Description |
|--------|---------------------|--------------------|
| GET | /api/todos | List your todos |
| POST | /api/todos | Create a todo |
| PUT | /api/todos/{id} | Update a todo |
| DELETE | /api/todos/{id} | Delete a todo |

## Usage Examples

```bash
# Register
curl -X POST http://localhost:8080/api/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
# Response: {"token":"eyJhbG..."}

# Create todo
curl -X POST http://localhost:8080/api/todos \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"title":"Buy milk","note":"2% fat"}'

# List todos
curl http://localhost:8080/api/todos \
  -H 'Authorization: Bearer <token>'

# Update todo
curl -X PUT http://localhost:8080/api/todos/1 \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"done":true}'

# Delete todo
curl -X DELETE http://localhost:8080/api/todos/1 \
  -H 'Authorization: Bearer <token>'
```
