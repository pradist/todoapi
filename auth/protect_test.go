package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

var testSecret = []byte("test_secret")

func setupProtectRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Protect(testSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func makeValidToken(t *testing.T) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
		Subject:   "1",
	})
	ss, err := token.SignedString(testSecret)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	return ss
}

func doProtectRequest(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestProtect_ValidToken: valid token returns 200
func TestProtect_ValidToken(t *testing.T) {
	r := setupProtectRouter()
	token := makeValidToken(t)

	w := doProtectRequest(r, "Bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestProtect_MissingHeader: missing Authorization header returns 401
func TestProtect_MissingHeader(t *testing.T) {
	r := setupProtectRouter()

	w := doProtectRequest(r, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestProtect_NoBearerPrefix: token without "Bearer " prefix returns 401
func TestProtect_NoBearerPrefix(t *testing.T) {
	r := setupProtectRouter()
	token := makeValidToken(t)

	w := doProtectRequest(r, token)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestProtect_ExpiredToken: expired token returns 401
func TestProtect_ExpiredToken(t *testing.T) {
	r := setupProtectRouter()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(-1 * time.Minute).Unix(),
		Subject:   "1",
	})
	ss, _ := token.SignedString(testSecret)

	w := doProtectRequest(r, "Bearer "+ss)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestProtect_TamperedToken: token with modified signature returns 401
func TestProtect_TamperedToken(t *testing.T) {
	r := setupProtectRouter()
	token := makeValidToken(t)
	tampered := token[:len(token)-4] + "xxxx"

	w := doProtectRequest(r, "Bearer "+tampered)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestProtect_WrongSigningMethod: token signed with a non-HMAC algorithm (RS256) returns 401
func TestProtect_WrongSigningMethod(t *testing.T) {
	r := setupProtectRouter()

	// Craft a token with RS256 header so keyfunc rejects it
	// header: {"alg":"RS256","typ":"JWT"} (base64-encoded)
	fakeToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.invalid_sig"

	w := doProtectRequest(r, "Bearer "+fakeToken)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
