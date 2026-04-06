package main

import (
	"testing"

	"github.com/pradist/todoapi/auth"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&auth.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func userCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	db.Model(&auth.User{}).Count(&n)
	return n
}

func TestEnsureAdminUser_CreatesUser(t *testing.T) {
	db := openSeedTestDB(t)
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASS", "secret123")

	ensureAdminUser(db)

	if userCount(t, db) != 1 {
		t.Fatal("expected 1 user to be created")
	}

	var u auth.User
	db.First(&u)
	if u.Username != "admin" {
		t.Errorf("expected username=admin, got %q", u.Username)
	}
	if !auth.CheckPassword("secret123", u.Password) {
		t.Error("password was not stored as a valid bcrypt hash")
	}
}

func TestEnsureAdminUser_SkipsWhenUsersExist(t *testing.T) {
	db := openSeedTestDB(t)
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASS", "secret123")

	hashed, _ := auth.HashPassword("existing")
	db.Create(&auth.User{Username: "existing", Password: hashed})

	ensureAdminUser(db)

	if userCount(t, db) != 1 {
		t.Fatal("expected seed to be skipped when users already exist")
	}
}

func TestEnsureAdminUser_SkipsWhenEnvMissing(t *testing.T) {
	db := openSeedTestDB(t)
	t.Setenv("ADMIN_USER", "")
	t.Setenv("ADMIN_PASS", "")

	ensureAdminUser(db)

	if userCount(t, db) != 0 {
		t.Fatal("expected no users when env vars are not set")
	}
}

func TestEnsureAdminUser_SkipsWhenOnlyUsernameMissing(t *testing.T) {
	db := openSeedTestDB(t)
	t.Setenv("ADMIN_USER", "")
	t.Setenv("ADMIN_PASS", "secret123")

	ensureAdminUser(db)

	if userCount(t, db) != 0 {
		t.Fatal("expected no users when ADMIN_USER is not set")
	}
}

func TestEnsureAdminUser_SkipsWhenOnlyPasswordMissing(t *testing.T) {
	db := openSeedTestDB(t)
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASS", "")

	ensureAdminUser(db)

	if userCount(t, db) != 0 {
		t.Fatal("expected no users when ADMIN_PASS is not set")
	}
}
