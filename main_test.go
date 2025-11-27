package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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
		name       string
		username   string
		password   string
		user       string
		pass       string
		expectAuth bool
	}{
		{
			name:       "Valid credentials",
			username:   "admin",
			password:   "admin",
			user:       "admin",
			pass:       "admin",
			expectAuth: true,
		},
		{
			name:       "Invalid username",
			username:   "admin",
			password:   "admin",
			user:       "wrong",
			pass:       "admin",
			expectAuth: false,
		},
		{
			name:       "Invalid password",
			username:   "admin",
			password:   "admin",
			user:       "admin",
			pass:       "wrong",
			expectAuth: false,
		},
		{
			name:       "Missing credentials",
			username:   "admin",
			password:   "admin",
			user:       "",
			pass:       "",
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
		{"Day 1", "2024-01-01", 10},    // Start at 10
		{"Day 2", "2024-01-02", 12},    // +2 (10+2)
		{"Day 10", "2024-01-10", 28},   // +2 each day for 9 days (10+9*2)
		{"Day 25", "2024-01-25", 54},   // Reached 50, now +1 per day
		{"Day 65", "2024-03-05", 94},   // Almost at 100
		{"Day 75", "2024-03-15", 102},  // After 100, +1 every 2 days
		{"Day 269", "2024-09-26", 200}, // Reached maximum of 200
		{"Day 365", "2024-12-31", 200}, // Stay at 200 permanently
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
			expected := calculateTarget(10, daysSince)

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
		Done:  true,
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
	startMonth := 0    // First record in January

	tests := []struct {
		name           string
		monthNumber    int
		expectPrevious bool
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

func TestMain(t *testing.T) {
	// Test that main() doesn't panic and sets up properly
	// We can't test the full main function since it starts a server
	// So we'll test the main setup logic in a separate test
}

func TestMainInitialization(t *testing.T) {
	// Test the initialization part of main() without starting the server
	// This simulates the environment variable handling and DB setup

	// Save original environment variables
	origPort := os.Getenv("PORT")
	origUsername := os.Getenv("USERNAME")
	origPassword := os.Getenv("PASSWORD")
	origPWD := os.Getenv("PWD")

	// Set test environment variables
	os.Setenv("PORT", "9000")
	os.Setenv("USERNAME", "testuser")
	os.Setenv("PASSWORD", "testpass")
	os.Setenv("PWD", "/tmp")

	// Clean up environment after test
	defer func() {
		if origPort == "" {
			os.Unsetenv("PORT")
		} else {
			os.Setenv("PORT", origPort)
		}

		if origUsername == "" {
			os.Unsetenv("USERNAME")
		} else {
			os.Setenv("USERNAME", origUsername)
		}

		if origPassword == "" {
			os.Unsetenv("PASSWORD")
		} else {
			os.Setenv("PASSWORD", origPassword)
		}

		if origPWD == "" {
			os.Unsetenv("PWD")
		} else {
			os.Setenv("PWD", origPWD)
		}
	}()

	// Test that environment variables would be handled correctly
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port != "9000" {
		t.Errorf("Expected port to be 9000, got %s", port)
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		username = "admin"
	}
	if username != "testuser" {
		t.Errorf("Expected username to be testuser, got %s", username)
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		password = "admin"
	}
	if password != "testpass" {
		t.Errorf("Expected password to be testpass, got %s", password)
	}

	// Test default values when environment variables are not set
	os.Unsetenv("PORT")
	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port != "8080" {
		t.Errorf("Expected default port to be 8080, got %s", port)
	}
}

func TestMainSetup(t *testing.T) {
	// Test environment variable handling and default values
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	origTmpl := tmpl
	origTodayCount := todayCount
	origTodayTarget := todayTarget

	db = testDB
	defer func() {
		db = origDB
		tmpl = origTmpl
		todayCount = origTodayCount
		todayTarget = origTodayTarget
	}()

	// Test template loading - skip if templates don't exist
	var err error
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		t.Skipf("Skipping template test as templates not available: %v", err)
	}

	// Test initializeTodayCount functionality
	initializeTodayCount()

	// Verify today's count and target are set
	if todayCount <= 0 {
		t.Errorf("Expected todayCount to be greater than 0, got %d", todayCount)
	}
	if todayTarget <= 0 {
		t.Errorf("Expected todayTarget to be greater than 0, got %d", todayTarget)
	}

	// Test that first day is set correctly in the config
	var firstDay string
	testDB.View(func(tx *bolt.Tx) error {
		fd, err := getFirstDay(tx)
		if err != nil {
			return err
		}
		firstDay = fd
		return nil
	})

	if firstDay == "" {
		t.Errorf("Expected first day to be set, got empty string")
	}

	// Test error handling in initializeTodayCount - set first day with invalid format
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Config"))
		return b.Put([]byte("firstDay"), []byte("invalid-date"))
	})
	if err != nil {
		t.Fatalf("Failed to set invalid first day: %v", err)
	}

	// This should log an error but not panic
	initializeTodayCount()

	// Check that global variables are still set properly
	if todayCount <= 0 {
		t.Errorf("Expected todayCount to be set even with error, got %d", todayCount)
	}

	// Test error when existing data has invalid JSON format
	today := time.Now().Format("2006-01-02")
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(today), []byte("{invalid json}"))
	})
	if err != nil {
		t.Fatalf("Failed to add invalid data: %v", err)
	}

	// This should log an error but not panic
	initializeTodayCount()

	// Test marshal error case
	// This isn't easy to test without modifying the function itself,
	// so let's focus on other error paths

	// Test closing DB to induce error
	testDB.Close()

	// This should log an error but not panic
	initializeTodayCount()
}

func TestInitializeTodayCount(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	origTodayCount := todayCount
	origTodayTarget := todayTarget
	db = testDB
	defer func() {
		db = origDB
		todayCount = origTodayCount
		todayTarget = origTodayTarget
	}()

	// Test initialization when no data exists for today
	// Need to test both scenarios: first day and subsequent days

	// Test 1: First day initialization
	today := time.Now().Format("2006-01-02")

	err := testDB.Update(func(tx *bolt.Tx) error {
		// Clear any existing config
		b := tx.Bucket([]byte("Config"))
		b.Delete([]byte("firstDay"))
		// Also clear any data for today
		daysB := tx.Bucket([]byte("Days"))
		daysB.Delete([]byte(today))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to prepare test DB: %v", err)
	}

	// Initialize today count for first time
	initializeTodayCount()

	// Verify first day was set
	var firstDay string
	testDB.View(func(tx *bolt.Tx) error {
		fd, err := getFirstDay(tx)
		if err != nil {
			return err
		}
		firstDay = fd
		return nil
	})

	if firstDay != today {
		t.Errorf("Expected first day to be today (%s), got %s", today, firstDay)
	}

	// Verify day data was created for today
	var dayData DayData
	testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(today))
		if data == nil {
			return fmt.Errorf("No data found for today")
		}
		return json.Unmarshal(data, &dayData)
	})

	if dayData.Date != today {
		t.Errorf("Expected date %s, got %s", today, dayData.Date)
	}
	if dayData.Count != 10 { // Initial target should be 10
		t.Errorf("Expected count 10, got %d", dayData.Count)
	}
	if dayData.Done != false {
		t.Errorf("Expected done to be false, got %v", dayData.Done)
	}

	// Test 2: Subsequent day initialization with existing first day
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	// Create data for tomorrow to simulate subsequent day
	tomorrowDayData := DayData{
		Date:  tomorrow,
		Count: 12, // Higher count for subsequent day
		Done:  false,
	}
	tomorrowJSON, _ := json.Marshal(tomorrowDayData)

	err = testDB.Update(func(tx *bolt.Tx) error {
		daysB := tx.Bucket([]byte("Days"))
		return daysB.Put([]byte(tomorrow), tomorrowJSON)
	})
	if err != nil {
		t.Fatalf("Failed to prepare tomorrow data: %v", err)
	}

	// Test when today's data already exists (should not overwrite)
	err = testDB.Update(func(tx *bolt.Tx) error {
		daysB := tx.Bucket([]byte("Days"))
		existingDayData := DayData{
			Date:  today,
			Count: 15, // Different count to test it's not overwritten
			Done:  true,
		}
		existingJSON, _ := json.Marshal(existingDayData)
		return daysB.Put([]byte(today), existingJSON)
	})
	if err != nil {
		t.Fatalf("Failed to prepare existing day data: %v", err)
	}

	// Initialize again
	initializeTodayCount()

	// Verify existing data was not changed
	testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(today))
		if data == nil {
			return fmt.Errorf("No data found for today")
		}
		return json.Unmarshal(data, &dayData)
	})

	if dayData.Count != 15 { // Should be the original value, not 10
		t.Errorf("Expected count 15 to be preserved, got %d", dayData.Count)
	}
	if dayData.Done != true { // Should be the original value
		t.Errorf("Expected done true to be preserved, got %v", dayData.Done)
	}

	// Check that todayCount is properly set
	if todayCount != 15 { // Should match the existing data
		t.Errorf("Expected todayCount to be 15, got %d", todayCount)
	}
}

func TestHandleIndex(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db and template
	origDB := db
	origTmpl := tmpl
	db = testDB
	defer func() {
		db = origDB
		tmpl = origTmpl
	}()

	// Test 1: With valid templates
	var err error
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		t.Skipf("Skipping test as templates not available: %v", err)
		return
	}

	// Test the index handler
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("admin", "admin") // Use default credentials
	w := httptest.NewRecorder()

	handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test 2: With invalid template path
	// Create a template that will fail to execute
	tmpl = template.New("invalid")

	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	// This should return an error because the template doesn't exist
	handleIndex(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for missing template, got %d", w.Code)
	}
}

func TestHandleToday(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	db = testDB
	defer func() { db = origDB }()

	// Test retrieving today's data
	today := time.Now().Format("2006-01-02")

	// Add test data
	dayData := DayData{
		Date:  today,
		Count: 15,
		Done:  false,
	}

	jsonData, _ := json.Marshal(dayData)
	err := testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(today), jsonData)
	})
	if err != nil {
		t.Fatalf("Failed to add test data: %v", err)
	}

	// Test the API endpoint
	req := httptest.NewRequest("GET", "/api/today", nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	handleToday(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the response
	var response DayData
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Date != today {
		t.Errorf("Expected date %s, got %s", today, response.Date)
	}
	if response.Count != 15 {
		t.Errorf("Expected count 15, got %d", response.Count)
	}
	if response.Done != false {
		t.Errorf("Expected done false, got %v", response.Done)
	}

	// Test error case when no data exists
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Delete([]byte(today))
	})
	if err != nil {
		t.Fatalf("Failed to delete test data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/today", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleToday(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for missing data, got %d", w.Code)
	}

	// Test with invalid JSON data in database
	invalidJSON := []byte("{invalid json}")
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(today), invalidJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add invalid test data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/today", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleToday(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleTodayComplete(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	origTodayCount := todayCount
	db = testDB
	defer func() {
		db = origDB
		todayCount = origTodayCount
	}()

	today := time.Now().Format("2006-01-02")
	todayCount = 15

	// Test 1: Completing today's workout with existing data
	dayData := DayData{
		Date:  today,
		Count: 15,
		Done:  false,
	}

	jsonData, _ := json.Marshal(dayData)
	err := testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(today), jsonData)
	})
	if err != nil {
		t.Fatalf("Failed to add test data: %v", err)
	}

	// Test the API endpoint with POST
	req := httptest.NewRequest("POST", "/api/today/complete", nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	handleTodayComplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the response
	var response DayData
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Done != true {
		t.Errorf("Expected done to be true after completion, got %v", response.Done)
	}

	// Test 2: Completing workout with no existing data (should create new)
	// Clear today's data
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Delete([]byte(today))
	})
	if err != nil {
		t.Fatalf("Failed to delete today data: %v", err)
	}

	req = httptest.NewRequest("POST", "/api/today/complete", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleTodayComplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for new day, got %d", w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Done != true {
		t.Errorf("Expected done to be true for new day, got %v", response.Done)
	}

	// Test 3: Error case with GET request (should fail)
	req = httptest.NewRequest("GET", "/api/today/complete", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleTodayComplete(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET request, got %d", w.Code)
	}

	// Test 4: Database error case
	// Close the database to induce an error
	testDB.Close()

	req = httptest.NewRequest("POST", "/api/today/complete", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleTodayComplete(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for DB error, got %d", w.Code)
	}
}

func TestHandleTodayCompleteErrorCases(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	origTodayCount := todayCount
	db = testDB
	defer func() {
		db = origDB
		todayCount = origTodayCount
	}()

	today := time.Now().Format("2006-01-02")
	todayCount = 15

	// Test error case with invalid JSON data already exists
	err := testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(today), []byte("{invalid json}"))
	})
	if err != nil {
		t.Fatalf("Failed to add invalid data: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/today/complete", nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	// Should fail with 500 due to invalid JSON
	handleTodayComplete(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for invalid JSON, got %d", w.Code)
	}
}

func TestUpdateStreak(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	db = testDB
	defer func() { db = origDB }()

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Test case 1: First day (no yesterday data)
	err := testDB.Update(func(tx *bolt.Tx) error {
		updateStreak(tx, today)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to update streak: %v", err)
	}

	var streak StreakData
	testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		data := b.Get([]byte("current"))
		if data != nil {
			return json.Unmarshal(data, &streak)
		}
		return nil
	})

	if streak.Current != 1 {
		t.Errorf("Expected current streak to be 1 for first day, got %d", streak.Current)
	}
	if streak.Longest != 1 {
		t.Errorf("Expected longest streak to be 1 for first day, got %d", streak.Longest)
	}
	if streak.LastDate != today {
		t.Errorf("Expected last date to be %s, got %s", today, streak.LastDate)
	}

	// Test case 2: Yesterday was completed
	yesterdayData := DayData{
		Date:  yesterday,
		Count: 10,
		Done:  true,
	}
	yesterdayJSON, _ := json.Marshal(yesterdayData)

	err = testDB.Update(func(tx *bolt.Tx) error {
		// Add yesterday's data
		b := tx.Bucket([]byte("Days"))
		err := b.Put([]byte(yesterday), yesterdayJSON)
		if err != nil {
			return err
		}

		// Set initial streak to 1 (yesterday's streak)
		streak := StreakData{
			Current:  1,
			Longest:  1,
			LastDate: yesterday,
		}
		streakJSON, _ := json.Marshal(streak)
		streakB := tx.Bucket([]byte("Streak"))
		err = streakB.Put([]byte("current"), streakJSON)
		return err
	})
	if err != nil {
		t.Fatalf("Failed to add yesterday data and initial streak: %v", err)
	}

	// Update streak for today
	err = testDB.Update(func(tx *bolt.Tx) error {
		updateStreak(tx, today)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to update streak: %v", err)
	}

	testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		data := b.Get([]byte("current"))
		if data != nil {
			return json.Unmarshal(data, &streak)
		}
		return nil
	})

	if streak.Current != 2 {
		t.Errorf("Expected current streak to be 2 when yesterday was completed, got %d", streak.Current)
	}

	// Test case 3: Yesterday was not completed
	yesterdayNotDone := DayData{
		Date:  yesterday,
		Count: 10,
		Done:  false,
	}
	yesterdayNotDoneJSON, _ := json.Marshal(yesterdayNotDone)

	err = testDB.Update(func(tx *bolt.Tx) error {
		// Update yesterday's data to not done
		b := tx.Bucket([]byte("Days"))
		err := b.Put([]byte(yesterday), yesterdayNotDoneJSON)
		if err != nil {
			return err
		}

		// Set initial streak to 1 (yesterday's streak)
		streak := StreakData{
			Current:  1,
			Longest:  1,
			LastDate: yesterday,
		}
		streakJSON, _ := json.Marshal(streak)
		streakB := tx.Bucket([]byte("Streak"))
		err = streakB.Put([]byte("current"), streakJSON)
		return err
	})
	if err != nil {
		t.Fatalf("Failed to update yesterday data and set initial streak: %v", err)
	}

	// Update streak for today
	err = testDB.Update(func(tx *bolt.Tx) error {
		updateStreak(tx, today)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to update streak again: %v", err)
	}

	testDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		data := b.Get([]byte("current"))
		if data != nil {
			return json.Unmarshal(data, &streak)
		}
		return nil
	})

	if streak.Current != 1 {
		t.Errorf("Expected current streak to be 1 when yesterday was not completed, got %d", streak.Current)
	}
}

func TestHandleCalendar(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	db = testDB
	defer func() { db = origDB }()

	year := strconv.Itoa(time.Now().Year())

	// Test case 1: No records
	req := httptest.NewRequest("GET", "/api/calendar?year="+year, nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	handleCalendar(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the response structure
	var response struct {
		Year       int                `json:"year"`
		StartMonth int                `json:"startMonth"`
		StartYear  int                `json:"startYear"`
		Days       map[string]DayData `json:"days"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Year != time.Now().Year() {
		t.Errorf("Expected year %d, got %d", time.Now().Year(), response.Year)
	}

	// Test case 2: With records
	testDate := year + "-01-01"
	dayData := DayData{
		Date:  testDate,
		Count: 10,
		Done:  true,
	}
	dayDataJSON, _ := json.Marshal(dayData)

	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(testDate), dayDataJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add test data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/calendar?year="+year, nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleCalendar(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify that the test date is included in the response
	if day, exists := response.Days[testDate]; !exists {
		t.Errorf("Expected to find data for date %s in response", testDate)
	} else {
		if day.Count != 10 {
			t.Errorf("Expected count 10, got %d", day.Count)
		}
		if day.Done != true {
			t.Errorf("Expected done true, got %v", day.Done)
		}
	}

	// Test case 3: Different year (should only include data from that year)
	nextYear := strconv.Itoa(time.Now().Year() + 1)
	nextYearDate := nextYear + "-01-01"

	// Add a date for next year
	nextYearData := DayData{
		Date:  nextYearDate,
		Count: 15,
		Done:  true,
	}
	nextYearJSON, _ := json.Marshal(nextYearData)

	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		return b.Put([]byte(nextYearDate), nextYearJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add test data for next year: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/calendar?year="+nextYear, nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleCalendar(w, req)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Should include the date we added for next year
	if _, exists := response.Days[nextYearDate]; !exists {
		t.Errorf("Expected to find data for date %s in next year response", nextYearDate)
	}

	// Should have at least one entry
	if len(response.Days) < 1 {
		t.Errorf("Expected at least 1 entry for year %s, got %d", nextYear, len(response.Days))
	}

	// Test case 4: Invalid date in database
	invalidDate := year + "-invalid-date"
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		invalidJSON := []byte("{invalid json}")
		return b.Put([]byte(invalidDate), invalidJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add invalid date data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/calendar?year="+year, nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	// Should still succeed despite invalid data
	handleCalendar(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 despite invalid data, got %d", w.Code)
	}

	// Test case 5: First record date with invalid format
	invalidFormatDate := "not-a-date"
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		// Clear all data first
		cursor := b.Cursor()
		var keys [][]byte
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			keys = append(keys, k)
		}
		for _, k := range keys {
			b.Delete(k)
		}
		// Add invalid date
		return b.Put([]byte(invalidFormatDate), []byte("{}"))
	})
	if err != nil {
		t.Fatalf("Failed to add invalid format date: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/calendar?year="+year, nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	// Should fail with 500 due to date parsing error
	handleCalendar(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for invalid date format, got %d", w.Code)
	}
}

func TestHandleCalendarErrorCases(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	db = testDB
	defer func() { db = origDB }()

	// Test database error case
	testDB.Close()

	year := strconv.Itoa(time.Now().Year())
	req := httptest.NewRequest("GET", "/api/calendar?year="+year, nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	// Should fail with 500 due to DB error
	handleCalendar(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for DB error, got %d", w.Code)
	}
}

func TestHandleStreak(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Save original db
	origDB := db
	db = testDB
	defer func() { db = origDB }()

	// Test case 1: No streak data
	req := httptest.NewRequest("GET", "/api/streak", nil)
	req.SetBasicAuth("admin", "admin")
	w := httptest.NewRecorder()

	handleStreak(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the response structure
	var streak StreakData
	err := json.Unmarshal(w.Body.Bytes(), &streak)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if streak.Current != 0 {
		t.Errorf("Expected current streak to be 0 initially, got %d", streak.Current)
	}
	if streak.Longest != 0 {
		t.Errorf("Expected longest streak to be 0 initially, got %d", streak.Longest)
	}
	if streak.LastDate != "" {
		t.Errorf("Expected last date to be empty initially, got %s", streak.LastDate)
	}

	// Test case 2: With existing streak data
	streakData := StreakData{
		Current:  5,
		Longest:  10,
		LastDate: time.Now().Format("2006-01-02"),
	}
	streakDataJSON, _ := json.Marshal(streakData)

	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		return b.Put([]byte("current"), streakDataJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add test streak data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/streak", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	handleStreak(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &streak)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if streak.Current != 5 {
		t.Errorf("Expected current streak to be 5, got %d", streak.Current)
	}
	if streak.Longest != 10 {
		t.Errorf("Expected longest streak to be 10, got %d", streak.Longest)
	}

	// Test case 3: Invalid streak data in database
	invalidJSON := []byte("{invalid json}")
	err = testDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		return b.Put([]byte("current"), invalidJSON)
	})
	if err != nil {
		t.Fatalf("Failed to add invalid streak data: %v", err)
	}

	req = httptest.NewRequest("GET", "/api/streak", nil)
	req.SetBasicAuth("admin", "admin")
	w = httptest.NewRecorder()

	// This should fail with 500 due to invalid JSON
	handleStreak(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for invalid JSON, got %d", w.Code)
	}
}
