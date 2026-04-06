package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pradist/todoapi/auth"
	"github.com/pradist/todoapi/middleware"
	"github.com/pradist/todoapi/todo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&todo.Todo{}, &auth.User{})
	seedAdminUser(db)
	return db, nil
}

func setupRouter(db *gorm.DB, sign string, limiter *middleware.IPLimiter) *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.POST("/tokenz", middleware.RateLimitMiddleware(limiter), auth.AccessToken(db, sign))
	protected := r.Group("", auth.Protect([]byte(sign)))
	handler := todo.NewTodoHandler(db)
	protected.POST("/todos", handler.NewTask)
	return r
}
