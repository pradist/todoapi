package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, username, password string) {
	t.Helper()
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := db.Create(&User{Username: username, Password: hashed}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func defaultSignFn(token *jwt.Token, key any) (string, error) {
	return token.SignedString(key)
}

func setupAuthRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tokenz", AccessToken(db, "test_secret", defaultSignFn))
	return r
}

func doTokenRequest(t *testing.T, r *gin.Engine, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAccessToken_Success: valid credentials return 200 with a token
func TestAccessToken_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, "alice", "secret123")
	r := setupAuthRouter(db)

	w := doTokenRequest(t, r, map[string]string{"username": "alice", "password": "secret123"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp["token"] == "" {
		t.Error("expected non-empty token in response")
	}
}

// TestAccessToken_WrongPassword: wrong password returns 401
func TestAccessToken_WrongPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, "alice", "secret123")
	r := setupAuthRouter(db)

	w := doTokenRequest(t, r, map[string]string{"username": "alice", "password": "wrongpass"})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestAccessToken_UserNotFound: unknown username returns 401
func TestAccessToken_UserNotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	r := setupAuthRouter(db)

	w := doTokenRequest(t, r, map[string]string{"username": "ghost", "password": "any"})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestAccessToken_MissingFields: missing or empty required fields return 400
func TestAccessToken_MissingFields(t *testing.T) {
	db := setupAuthTestDB(t)
	r := setupAuthRouter(db)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing password", map[string]string{"username": "alice"}},
		{"missing username", map[string]string{"password": "secret"}},
		{"empty fields", map[string]string{"username": "", "password": ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doTokenRequest(t, r, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

// TestAccessToken_InvalidJSON: malformed JSON body returns 400
func TestAccessToken_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tokenz", AccessToken(db, "test_secret", defaultSignFn))

	req := httptest.NewRequest(http.MethodPost, "/tokenz", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestAccessToken_ErrorMessageHidden: wrong username and wrong password must return the same error message
// to prevent user enumeration attacks
func TestAccessToken_ErrorMessageHidden(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, "alice", "secret123")
	r := setupAuthRouter(db)

	wWrongUser := doTokenRequest(t, r, map[string]string{"username": "nobody", "password": "secret123"})
	wWrongPass := doTokenRequest(t, r, map[string]string{"username": "alice", "password": "wrongpass"})

	var resp1, resp2 map[string]string
	json.Unmarshal(wWrongUser.Body.Bytes(), &resp1)
	json.Unmarshal(wWrongPass.Body.Bytes(), &resp2)

	if resp1["error"] != resp2["error"] {
		t.Errorf("error messages differ: %q vs %q -- may leak user existence", resp1["error"], resp2["error"])
	}
}

// TestAccessToken_SigningError: JWT signing failure returns 500
func TestAccessToken_SigningError(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, "alice", "secret123")

	failingSignFn := func(_ *jwt.Token, _ any) (string, error) {
		return "", errors.New("signing failed")
	}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.POST("/tokenz", AccessToken(db, "test_secret", failingSignFn))
	w := doTokenRequest(t, r, map[string]string{"username": "alice", "password": "secret123"})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
