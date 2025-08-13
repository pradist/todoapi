package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pradist/todoapi/auth"
	"github.com/pradist/todoapi/todo"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("please consider environment variables: %s", err)
	}

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&todo.Todo{})

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/tokenz", auth.AccessToken(os.Getenv("SIGN")))
	protected := r.Group("", auth.Protect([]byte(os.Getenv("SIGN"))))

	handler := todo.NewTodoHandler(db)
	protected.POST("/todos", handler.NewTask)

	r.Run()
}
