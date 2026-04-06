package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
