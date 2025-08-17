package todo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	
	err = db.AutoMigrate(&Todo{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	
	return db
}

func setupTestHandler(t *testing.T) (*TodoHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	
	db := setupTestDB(t)
	handler := NewTodoHandler(db)
	
	router := gin.New()
	
	return handler, router
}

func TestNewTask_Success(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	todo := map[string]interface{}{
		"text": "Test todo item",
	}
	
	jsonData, _ := json.Marshal(todo)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	
	if _, exists := response["ID"]; !exists {
		t.Error("expected response to contain ID field")
	}
}

func TestNewTask_EmptyTitle(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	todo := map[string]interface{}{
		"text": "",
	}
	
	jsonData, _ := json.Marshal(todo)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestNewTask_InvalidJSON(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	invalidJSON := `{"text": invalid json}`
	
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	
	if _, exists := response["error"]; !exists {
		t.Error("expected response to contain error field")
	}
}

func TestNewTask_MissingContentType(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	todo := map[string]interface{}{
		"text": "Test todo",
	}
	
	jsonData, _ := json.Marshal(todo)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Gin is lenient with content type, so this actually succeeds
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestNewTask_MultipleItems(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	testCases := []struct {
		text string
		name string
	}{
		{"First todo", "first"},
		{"Second todo", "second"},
		{"Third todo with longer description", "third"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			todo := map[string]interface{}{
				"text": tc.text,
			}
			
			jsonData, _ := json.Marshal(todo)
			req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
			}
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			
			if _, exists := response["ID"]; !exists {
				t.Error("expected response to contain ID field")
			}
		})
	}
}

func TestTodoHandler_DatabasePersistence(t *testing.T) {
	handler, router := setupTestHandler(t)
	router.POST("/todos", handler.NewTask)
	
	todo := map[string]interface{}{
		"text": "Persistent todo",
	}
	
	jsonData, _ := json.Marshal(todo)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	todoID := response["ID"]
	
	var savedTodo Todo
	err := handler.db.First(&savedTodo, todoID).Error
	if err != nil {
		t.Errorf("todo was not saved to database: %v", err)
	}
	
	if savedTodo.Title != "Persistent todo" {
		t.Errorf("expected title 'Persistent todo', got '%s'", savedTodo.Title)
	}
}

func TestNewTask_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a database connection that will fail
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	
	// Migrate normally first
	err = db.AutoMigrate(&Todo{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	
	// Close the database connection to force errors
	sqlDB, _ := db.DB()
	sqlDB.Close()
	
	handler := NewTodoHandler(db)
	router := gin.New()
	router.POST("/todos", handler.NewTask)
	
	todo := map[string]interface{}{
		"text": "This should fail",
	}
	
	jsonData, _ := json.Marshal(todo)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	
	if _, exists := response["error"]; !exists {
		t.Error("expected response to contain error field")
	}
}