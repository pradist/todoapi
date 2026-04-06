package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func newTestRouter(limiter *IPLimiter) *gin.Engine {
	r := gin.New()
	r.GET("/", RateLimitMiddleware(limiter), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestNewIPLimiter(t *testing.T) {
	l := NewIPLimiter()
	if l == nil {
		t.Fatal("expected non-nil IPLimiter")
	}
	if l.limiters == nil {
		t.Fatal("expected limiters map to be initialized")
	}
}

func TestGet_SameIPReturnsSameLimiter(t *testing.T) {
	l := NewIPLimiter()
	first := l.get("1.2.3.4")
	second := l.get("1.2.3.4")
	if first != second {
		t.Error("expected the same limiter to be returned for the same IP")
	}
}

func TestGet_DifferentIPsReturnDifferentLimiters(t *testing.T) {
	l := NewIPLimiter()
	a := l.get("1.1.1.1")
	b := l.get("2.2.2.2")
	if a == b {
		t.Error("expected different limiters for different IPs")
	}
}

func TestRateLimitMiddleware_AllowsWithinBurst(t *testing.T) {
	r := newTestRouter(NewIPLimiter())

	// burst is 5, all 5 requests should pass
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksAfterBurst(t *testing.T) {
	r := newTestRouter(NewIPLimiter())

	// exhaust the burst of 5
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		r.ServeHTTP(w, req)
	}

	// 6th request must be rejected
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_DifferentIPsAreIndependent(t *testing.T) {
	limiter := NewIPLimiter()
	r := newTestRouter(limiter)

	// exhaust IP A
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		r.ServeHTTP(w, req)
	}

	// IP B should still be allowed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.4:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected IP B to be allowed (200), got %d", w.Code)
	}
}
