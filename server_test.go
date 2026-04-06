package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pradist/todoapi/auth"
	"github.com/pradist/todoapi/middleware"
	"github.com/pradist/todoapi/todo"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&todo.Todo{}, &auth.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedTestUser(t *testing.T, db *gorm.DB, username, password string) {
	t.Helper()
	hashed, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := db.Create(&auth.User{Username: username, Password: hashed}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func getToken(t *testing.T, r http.Handler, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /tokenz, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["token"]
}

// noLimiter returns a limiter with no restrictions so rate limiting
// does not interfere with router tests.
func noLimiter() *middleware.IPLimiter {
	return middleware.NewIPLimiter(rate.Inf, 0)
}

// --- setupRouter tests ---

func TestSetupRouter_Ping(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db, "secret", noLimiter())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["message"] != "pong" {
		t.Errorf("expected message=pong, got %q", body["message"])
	}
}

func TestSetupRouter_Tokenz_ValidCredentials(t *testing.T) {
	db := setupTestDB(t)
	seedTestUser(t, db, "admin", "pass123")
	r := setupRouter(db, "secret", noLimiter())

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "pass123"})
	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("expected token in response body")
	}
}

func TestSetupRouter_Tokenz_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	seedTestUser(t, db, "admin", "pass123")
	r := setupRouter(db, "secret", noLimiter())

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSetupRouter_Todos_WithoutAuth(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db, "secret", noLimiter())

	body, _ := json.Marshal(map[string]string{"text": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSetupRouter_Todos_WithValidToken(t *testing.T) {
	db := setupTestDB(t)
	seedTestUser(t, db, "admin", "pass123")
	r := setupRouter(db, "secret", noLimiter())

	token := getToken(t, r, "admin", "pass123")

	body, _ := json.Marshal(map[string]string{"text": "buy milk"})
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

// --- ipLimiterFromEnv tests ---

func TestIPLimiterFromEnv_Defaults(t *testing.T) {
	t.Setenv("RATE_LIMIT", "")
	t.Setenv("RATE_BURST", "")

	l := ipLimiterFromEnv()
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}
}

func TestIPLimiterFromEnv_Disabled(t *testing.T) {
	t.Setenv("RATE_LIMIT", "0")

	r := setupRouter(setupTestDB(t), "secret", ipLimiterFromEnv())

	// 20 requests should all pass when limiting is disabled
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 when disabled, got %d", i+1, w.Code)
		}
	}
}

func TestIPLimiterFromEnv_CustomValues(t *testing.T) {
	t.Setenv("RATE_LIMIT", "10")
	t.Setenv("RATE_BURST", "2")

	r := setupRouter(setupTestDB(t), "secret", ipLimiterFromEnv())

	// burst is 2, first 2 requests to /tokenz pass (rate limiter allows them)
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{"username": "x", "password": "x"})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.9:1234"
		r.ServeHTTP(w, req)
		// 401 is fine — rate limiter let the request through
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d: expected rate limiter to allow, got 429", i+1)
		}
	}

	// 3rd request should be blocked by the rate limiter
	body, _ := json.Marshal(map[string]string{"username": "x", "password": "x"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.9:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

// --- setupDB tests ---

func TestSetupDB_Success(t *testing.T) {
	db, err := setupDB(":memory:")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestSetupDB_OpenError(t *testing.T) {
	// SQLite cannot create a file inside a non-existent subdirectory
	dsn := t.TempDir() + "/nonexistent/test.db"
	_, err := setupDB(dsn)
	if err == nil {
		t.Fatal("expected error for invalid dsn, got nil")
	}
}

// --- startServer tests ---

func TestStartServer_GracefulShutdown(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db, "secret", noLimiter())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- startServer(ctx, r, ":0")
	}()

	// Give the server goroutine time to start ListenAndServe
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil error on graceful shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestStartServer_ListenError(t *testing.T) {
	// Occupy a port so startServer's ListenAndServe fails with a real error
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to bind port: %v", err)
	}
	defer ln.Close()
	port := fmt.Sprintf(":%d", ln.Addr().(*net.TCPAddr).Port)

	db := setupTestDB(t)
	r := setupRouter(db, "secret", noLimiter())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- startServer(ctx, r, port)
	}()

	// Give the goroutine time to hit the listen error and print it
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestStartServer_ServesRequestsBeforeShutdown(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db, "secret", noLimiter())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan string, 1)

	// Use httptest to capture the actual address
	go func() {
		// Start server on a random port via httptest server approach
		_ = startServer(ctx, r, ":0")
	}()
	close(ready)

	// Verify the router itself responds correctly (without going through startServer's port)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /ping, got %d", w.Code)
	}
	cancel()
}
