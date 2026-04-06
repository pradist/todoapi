package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("please consider environment variables: %s", err)
	}

	db, err := setupDB("todo.db")
	if err != nil {
		panic("failed to connect database")
	}

	r := setupRouter(db, os.Getenv("SIGN"), ipLimiterFromEnv())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := startServer(ctx, r, ":"+os.Getenv("PORT")); err != nil {
		fmt.Printf("Server forced to shutdown: %s\n", err)
	}

	fmt.Println("Server exiting")
}
