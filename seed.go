package main

import (
	"fmt"
	"os"

	"github.com/pradist/todoapi/auth"
	"gorm.io/gorm"
)

func seedAdminUser(db *gorm.DB, hashFn func(string) (string, error)) {
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

	hashed, err := hashFn(password)
	if err != nil {
		fmt.Printf("failed to hash admin password: %s\n", err)
		return
	}
	db.Create(&auth.User{Username: username, Password: hashed})
	fmt.Println("Admin user seeded")
}
