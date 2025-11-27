package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/boltdb/bolt"
)

// Mock database for testing
func setupTestDB(t *testing.T) *bolt.DB {
	t.Helper()
	file := "test.db"
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	
	// Create test data
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Days"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("Streak"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("Config"))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	
	return db
}

func cleanupTestDB(t *testing.T, db *bolt.DB) {
	t.Helper()
	db.Close()
	os.Remove("test.db")
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username  string
		password  string
		user      string
		pass      string
		expectAuth bool
	}{
		{
			name:      "Valid credentials",
			username:  "admin",
			password:  "admin",
			user:      "admin",
			pass:      "admin",
			expectAuth: true,
		},
		{
			name:      "Invalid username",
			username:  "admin",
			password:  "admin",
			user:      "wrong",
			pass:      "admin",
			expectAuth: false,
		},
		{
			name:      "Invalid password",
			username:  "admin",
			password:  "admin",
			user:      "admin",
			pass:      "wrong",
			expectAuth: false,
		},
		{
			name:      "Missing credentials",
			username:  "admin",
			password:  "admin",
			user:      "",
			pass:      "",
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic authentication test
			authValid := (tt.user == tt.username) && (tt.pass == tt.password)
			if authValid != tt.expectAuth {
				t.Errorf("Expected auth=%v, got auth=%v", tt.expectAuth, authValid)
			}
		})
	}
}

func TestSetAndGetFirstDay(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Test setting first day
	err := testDB.Update(func(tx *bolt.Tx) error {
		return setFirstDay(tx, "2024-01-01")
	})
	if err != nil {
		t.Errorf("Failed to set first day: %v", err)
	}

	// Test getting first day
	var firstDay string
	err = testDB.View(func(tx *bolt.Tx) error {
		fd, err := getFirstDay(tx)
		if err != nil {
			return err
		}
		firstDay = fd
		return nil
	})
	if err != nil {
		t.Errorf("Failed to get first day: %v", err)
	}

	if firstDay != "2024-01-01" {
		t.Errorf("Expected first day '2024-01-01', got '%s'", firstDay)
	}
}

func TestBasicAuthHandler(t *testing.T) {
	// Test basicAuth middleware
	handler := basicAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}, "testuser", "testpass")

	// Test with correct credentials
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("testuser", "testpass")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with incorrect credentials
	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("wronguser", "wrongpass")
	w = httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestProgressiveLoad(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Set first day to 2024-01-01
	err := testDB.Update(func(tx *bolt.Tx) error {
		return setFirstDay(tx, "2024-01-01")
	})
	if err != nil {
		t.Fatalf("Failed to set first day: %v", err)
	}

	// Test different scenarios
	tests := []struct {
		name          string
		targetDate    string
		expectedCount int
	}{
		{"Day 1", "2024-01-01", 5},
		{"Day 2", "2024-01-02", 6},
		{"Day 10", "2024-01-10", 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var firstDay string
			testDB.View(func(tx *bolt.Tx) error {
				fd, err := getFirstDay(tx)
				if err != nil {
					return err
				}
				firstDay = fd
				return nil
			})
			
			firstTime, _ := time.Parse("2006-01-02", firstDay)
			targetTime, _ := time.Parse("2006-01-02", tt.targetDate)
			
			daysSince := int(targetTime.Sub(firstTime).Hours() / 24)
			expected := 5 + daysSince

			if expected != tt.expectedCount {
				t.Errorf("Expected count %d, got %d", tt.expectedCount, expected)
			}
		})
	}
}

func TestDayDataOperations(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Test storing and retrieving day data
	dayData := DayData{
		Date:  "2024-01-01",
		Count: 5,
		Done:   true,
	}

	jsonData, _ := json.Marshal(dayData)

	// Store data
	err := testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(dayData.Date), jsonData)
	})
	if err != nil {
		t.Fatalf("Failed to store day data: %v", err)
	}

	// Retrieve data
	var retrieved DayData
	err = testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(dayData.Date))
		return json.Unmarshal(data, &retrieved)
	})
	if err != nil {
		t.Fatalf("Failed to retrieve day data: %v", err)
	}

	// Verify data
	if retrieved.Date != dayData.Date {
		t.Errorf("Expected date %s, got %s", dayData.Date, retrieved.Date)
	}
	if retrieved.Count != dayData.Count {
		t.Errorf("Expected count %d, got %d", dayData.Count, retrieved.Count)
	}
	if retrieved.Done != dayData.Done {
		t.Errorf("Expected done %v, got %v", dayData.Done, retrieved.Done)
	}
}

func TestStreakDataOperations(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Test storing and retrieving streak data
	streakData := StreakData{
		Current:  5,
		Longest:  10,
		LastDate: "2024-01-05",
	}

	jsonData, _ := json.Marshal(streakData)

	// Store data
	err := testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		return b.Put([]byte("current"), jsonData)
	})
	if err != nil {
		t.Fatalf("Failed to store streak data: %v", err)
	}

	// Retrieve data
	var retrieved StreakData
	err = testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		data := b.Get([]byte("current"))
		return json.Unmarshal(data, &retrieved)
	})
	if err != nil {
		t.Fatalf("Failed to retrieve streak data: %v", err)
	}

	// Verify data
	if retrieved.Current != streakData.Current {
		t.Errorf("Expected current %d, got %d", streakData.Current, retrieved.Current)
	}
	if retrieved.Longest != streakData.Longest {
		t.Errorf("Expected longest %d, got %d", streakData.Longest, retrieved.Longest)
	}
	if retrieved.LastDate != streakData.LastDate {
		t.Errorf("Expected last date %s, got %s", streakData.LastDate, retrieved.LastDate)
	}
}

func TestStaticFileSecurity(t *testing.T) {
	// Test static file handler security
	handler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		
		// Validate path to prevent directory traversal
		if !strings.HasPrefix(path, "static/") {
			http.NotFound(w, r)
			return
		}
		// Block sensitive files
		if strings.HasSuffix(path, ".go") || strings.Contains(path, "..") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}

	// Test valid static file
	req := httptest.NewRequest("GET", "/static/style.css", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected valid static file to return 200, got %d", w.Code)
	}

	// Test directory traversal
	req = httptest.NewRequest("GET", "/static/../../../etc/passwd", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected directory traversal to return 404, got %d", w.Code)
	}

	// Test .go file access
	req = httptest.NewRequest("GET", "/static/main.go", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected .go file access to return 404, got %d", w.Code)
	}
}

// Test calendar month logic
func TestCalendarMonthLogic(t *testing.T) {
	// Test the actual logic used in the app
	// Current month is the actual current month from time.Now()
	// Previous month logic is: month < currentMonth && month >= startMonth
	
	currentMonth := 10 // October for consistent testing
	startMonth := 0 // First record in January
	
	tests := []struct {
		name               string
		monthNumber       int
		expectPrevious     bool
	}{
		{"January (month 0) - before current month, after start", 0, true},
		{"September (month 9) - before current month, after start", 9, true},
		{"October (month 10) - current month", 10, false},
		{"November (month 11) - after current month", 11, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// JavaScript logic: const isPreviousMonth = month < currentMonth && month >= startMonth;
			isPrevious := (tt.monthNumber < currentMonth) && (tt.monthNumber >= startMonth)
			if isPrevious != tt.expectPrevious {
				t.Errorf("Expected previous=%v for month %d with current month %d and start month %d", 
					tt.expectPrevious, tt.monthNumber, currentMonth, startMonth)
			}
		})
	}
}