package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pradist/todoapi/auth"
	"github.com/pradist/todoapi/todo"
)

// ipLimiter holds a per-IP token bucket limiter.
type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newIPLimiter() *ipLimiter {
	return &ipLimiter{limiters: make(map[string]*rate.Limiter)}
}

// get returns (or creates) a limiter for the given IP.
// Allows 5 requests per minute with a burst of 5.
func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(rate.Every(time.Minute/5), 5)
	l.limiters[ip] = lim
	return lim
}

func rateLimitMiddleware(l *ipLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
			return
		}
		c.Next()
	}
}

func seedAdminUser(db *gorm.DB) {
	username := os.Getenv("ADMIN_USER")
	password := os.Getenv("ADMIN_PASS")
	if username == "" || password == "" {
		fmt.Println("ADMIN_USER or ADMIN_PASS not set, skipping seed")
		return
	}

	var count int64
	db.Model(&auth.User{}).Count(&count)
	if count > 0 {
		return
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		fmt.Printf("failed to hash admin password: %s\n", err)
		return
	}
	db.Create(&auth.User{Username: username, Password: hashed})
	fmt.Println("Admin user seeded")
}

func setupDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&todo.Todo{}, &auth.User{})
	seedAdminUser(db)
	return db, nil
}

func setupRouter(db *gorm.DB, sign string, limiter *ipLimiter) *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.POST("/tokenz", rateLimitMiddleware(limiter), auth.AccessToken(db, sign))
	protected := r.Group("", auth.Protect([]byte(sign)))
	handler := todo.NewTodoHandler(db)
	protected.POST("/todos", handler.NewTask)
	return r
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("please consider environment variables: %s", err)
	}

	db, err := setupDB()
	if err != nil {
		panic("failed to connect database")
	}

	r := setupRouter(db, os.Getenv("SIGN"), newIPLimiter())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := &http.Server{
		Addr:           ":" + os.Getenv("PORT"),
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
	stop()
	fmt.Println("Shutting down gracefully, press Ctrl+C again to force")

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctxTimeout); err != nil {
		fmt.Printf("Server forced to shutdown: %s\n", err)
	}

	fmt.Println("Server exiting")
}
