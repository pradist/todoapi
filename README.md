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
├── main.go               # Entry point — server setup, routing, graceful shutdown
├── auth/
│   ├── auth.go           # POST /tokenz handler — credential validation + JWT issuance
│   ├── auth_test.go      # Unit tests for AccessToken handler
│   ├── protect.go        # JWT middleware for protected routes
│   ├── protect_test.go   # Unit tests for Protect middleware
│   ├── user.go           # User GORM model, HashPassword, CheckPassword (bcrypt)
│   └── user_test.go      # Unit tests for password hashing helpers
├── todo/
│   ├── todo.go           # Todo model and handler
│   └── todo_test.go      # Unit tests for NewTask handler
├── test/
│   ├── *.http            # HTTP request samples (VS Code httpYac)
│   └── hurl/             # Hurl integration test files
│       ├── 01_health.hurl
│       ├── 02_auth.hurl
│       ├── 03_todos.hurl
│       ├── vars.env          # Local variables (not committed)
│       └── vars.env.example  # Variable template
├── scripts/
│   └── check-fmt.sh      # gofmt check script used by pre-commit
├── .pre-commit-config.yaml
├── .github/
│   └── workflows/
│       └── integration.yml  # GitHub Actions — integration tests via Hurl
├── go.mod
└── .env                  # Environment variables (not committed)
```

## Environment Variables

Create a `.env` file in the project root:

```env
PORT=8081
SIGN=your_jwt_secret_key
ADMIN_USER=admin
ADMIN_PASS=your_admin_password
TEST_SIGN=your_test_jwt_secret
TEST_FAKE_RS256_TOKEN=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.invalid_sig
```

| Variable                | Description                                                          |
|-------------------------|----------------------------------------------------------------------|
| `PORT`                  | Port the server listens on                                           |
| `SIGN`                  | Secret key used to sign JWT tokens (use a strong random string)      |
| `ADMIN_USER`            | Username for the seeded admin account                                |
| `ADMIN_PASS`            | Password for the seeded admin account (stored as bcrypt hash in DB)  |
| `TEST_SIGN`             | Secret key used when signing tokens in tests                         |
| `TEST_FAKE_RS256_TOKEN` | A JWT with RS256 header used in the wrong-signing-method test        |

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

### Unit tests

```bash
go test ./...
```

### Integration tests (Hurl)

Requires the server to be running and [Hurl](https://hurl.dev/) installed.

```bash
# copy and fill in your values
cp test/hurl/vars.env.example test/hurl/vars.env

# run all integration tests
hurl --variables-file test/hurl/vars.env --test test/hurl/*.hurl

# run with HTML report
hurl --variables-file test/hurl/vars.env --test --report-html report/ test/hurl/*.hurl
```

## Pre-commit Hooks

This project uses [pre-commit](https://pre-commit.com/) to enforce code quality before every commit.

**Install once:**

```bash
pip install pre-commit
pre-commit install
```

Hooks that run on every `git commit`:

| Hook | What it checks |
| ---- | -------------- |
| `go-fmt` | All files are formatted with `gofmt` |
| `go-vet` | No issues found by `go vet` |
| `go-test` | All unit tests pass |
| `go-build` | Project compiles successfully |

**Run manually against all files:**

```bash
pre-commit run --all-files
```

## Graceful Shutdown

The server listens for `SIGINT` and `SIGTERM` signals. On receiving either signal it stops accepting new connections and waits up to **5 seconds** for in-flight requests to complete before exiting.
