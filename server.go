package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/pradist/todoapi/auth"
	"github.com/pradist/todoapi/middleware"
	"github.com/pradist/todoapi/todo"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ipLimiterFromEnv builds an IPLimiter from environment variables.
//
//	RATE_LIMIT  - requests per minute per IP (default: 5; set to 0 to disable)
//	RATE_BURST  - maximum burst size (default: 5)
func ipLimiterFromEnv() *middleware.IPLimiter {
	limitPerMin := 5
	burst := 5

	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limitPerMin = n
		}
	}
	if v := os.Getenv("RATE_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			burst = n
		}
	}

	if limitPerMin == 0 {
		return middleware.NewIPLimiter(rate.Inf, 0)
	}
	r := rate.Every(time.Minute / time.Duration(limitPerMin))
	return middleware.NewIPLimiter(r, burst)
}

func setupDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&todo.Todo{}, &auth.User{})
	seedAdminUser(db, auth.HashPassword)
	return db, nil
}

func setupRouter(db *gorm.DB, sign string, limiter *middleware.IPLimiter) *gin.Engine {
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/tokenz", middleware.RateLimitMiddleware(limiter), auth.AccessToken(db, sign, func(token *jwt.Token, key any) (string, error) {
		return token.SignedString(key)
	}))
	protected := r.Group("", auth.Protect([]byte(sign)))
	handler := todo.NewTodoHandler(db)
	protected.POST("/todos", handler.NewTask)
	return r
}

func startServer(ctx context.Context, r http.Handler, addr string) error {
	s := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("listen: %s\n", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Shutting down gracefully, press Ctrl+C again to force")

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.Shutdown(ctxTimeout)
}
