# Todo API

A simple RESTful API for managing todo items, built with Go. It features credential-based JWT authentication, per-IP rate limiting, SQLite persistence via GORM, and graceful shutdown on system signals.

## Tech Stack

| Layer         | Library                                                                            |
| ------------- | ---------------------------------------------------------------------------------- |
| HTTP router   | [Gin](https://github.com/gin-gonic/gin)                                            |
| ORM           | [GORM](https://gorm.io) with SQLite driver                                         |
| Auth          | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) (HS256)                        |
| Password hash | [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)                            |
| Rate limiting | [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) (token bucket) |
| Config        | [godotenv](https://github.com/joho/godotenv)                                       |

## Project Structure

``` text
.
├── main.go          # Entry point — server setup, routing, graceful shutdown
├── auth/
│   ├── auth.go      # POST /tokenz handler — credential validation + JWT issuance
│   ├── protect.go   # JWT middleware for protected routes
│   └── user.go      # User GORM model, HashPassword, CheckPassword (bcrypt)
├── todo/
│   ├── todo.go      # Todo model and handler
│   └── todo_test.go # Unit tests
├── test/            # HTTP request samples (VS Code REST Client)
├── go.mod
└── .env             # Environment variables (not committed)
```

## Environment Variables

Create a `.env` file in the project root:

```env
PORT=8081
SIGN=your_jwt_secret_key
ADMIN_USER=admin
ADMIN_PASS=your_admin_password
```

| Variable     | Description                                                        |
|--------------|--------------------------------------------------------------------|
| `PORT`       | Port the server listens on                                         |
| `SIGN`       | Secret key used to sign JWT tokens (use a strong random string)    |
| `ADMIN_USER` | Username for the seeded admin account                              |
| `ADMIN_PASS` | Password for the seeded admin account (stored as bcrypt hash in DB)|

## Getting Started

**Prerequisites:** Go 1.24+

```bash
# Clone the repository
git clone https://github.com/pradist/todoapi.git
cd todoapi

# Install dependencies
go mod download

# Create environment file
cp .env.example .env   # then edit .env with your values

# Run the server
go run main.go
```

The server will start at `http://localhost:<PORT>`.

## API Endpoints

### Health Check

``` bash
GET /ping
```

Response:

```json
{ "message": "pong" }
```

### Get Access Token

``` bash
POST /tokenz
Content-Type: application/json
```

Request body:

```json
{ "username": "admin", "password": "your_admin_password" }
```

Returns a JWT token valid for **5 minutes**.

Response `200 OK`:

```json
{ "token": "<jwt_token>" }
```

Error responses:

- `400 Bad Request` — missing username or password
- `401 Unauthorized` — invalid credentials
- `429 Too Many Requests` — exceeded **5 requests per minute** per IP

### Create a Todo *(protected)*

``` bash
POST /todos
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

Request body:

```json
{ "text": "Buy books" }
```

Response `201 Created`:

```json
{ "ID": 1 }
```

## Authentication Flow

1. Call `POST /tokenz` with your `username` and `password` to obtain a short-lived JWT.
2. Include the token in subsequent requests as `Authorization: Bearer <token>`.
3. The `Protect` middleware validates the token signature and rejects expired or tampered tokens with `401 Unauthorized`.

## Rate Limiting

`POST /tokenz` is protected by a **per-IP token bucket** limiter:

- **5 requests per minute** per client IP
- Exceeding the limit returns `429 Too Many Requests`
- The limiter is in-memory and resets when the server restarts

## Running Tests

```bash
go test ./...
```

## Graceful Shutdown

The server listens for `SIGINT` and `SIGTERM` signals. On receiving either signal it stops accepting new connections and waits up to **5 seconds** for in-flight requests to complete before exiting.
