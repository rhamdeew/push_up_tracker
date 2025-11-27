package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/joho/godotenv"
)

var (
	db          *bolt.DB
	tmpl        *template.Template
	todayCount  int
	todayTarget int
)

type DayData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
	Done  bool   `json:"done"`
}

type StreakData struct {
	Current  int `json:"current"`
	Longest  int `json:"longest"`
	LastDate string `json:"lastDate"`
}

func main() {
	// Load .env file if it exists
	godotenvErr := godotenv.Load()
	if godotenvErr != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		password = "admin"
	}

	// Initialize BoltDB
	var err error
	dbPath := filepath.Join(".", "pushups.db")
	
	// Ensure working directory is the installation directory
	workingDir := os.Getenv("PWD")
	if workingDir == "" {
		workingDir = "."
	}
	dbPath = filepath.Join(workingDir, "pushups.db")
	
	db, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Days"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte("Streak"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte("Config"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Initialize today's count
	initializeTodayCount()

	// Load templates
	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	// Setup routes
	http.HandleFunc("/", basicAuth(handleIndex, username, password))
	http.HandleFunc("/api/today", basicAuth(handleToday, username, password))
	http.HandleFunc("/api/today/complete", basicAuth(handleTodayComplete, username, password))
	http.HandleFunc("/api/calendar", basicAuth(handleCalendar, username, password))
	http.HandleFunc("/api/streak", basicAuth(handleStreak, username, password))
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Security: Validate path to prevent directory traversal
		path := r.URL.Path[1:]
		if !strings.HasPrefix(path, "static/") {
			http.NotFound(w, r)
			return
		}
		// Additional security: Don't serve .go files or other sensitive files
		if strings.HasSuffix(path, ".go") || strings.Contains(path, "..") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initializeTodayCount() {
	today := time.Now().Format("2006-01-02")
	
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(today))
		
		if data == nil {
			// Check if this is the first day (database initialization)
			firstDay, err := getFirstDay(tx)
			if err != nil {
				return err
			}
			
			if firstDay == "" {
				// Database is empty, this is initialization day
				firstDay = today
				err = setFirstDay(tx, firstDay)
				if err != nil {
					return err
				}
				todayTarget = 5
			} else {
				// Calculate days since first day
				firstDayTime, err := time.Parse("2006-01-02", firstDay)
				if err != nil {
					return err
				}
				daysSince := int(time.Since(firstDayTime).Hours() / 24)
				todayTarget = 5 + daysSince
			}
			
			dayData := DayData{
				Date:  today,
				Count: todayTarget,
				Done:  false,
			}
			
			jsonData, err := json.Marshal(dayData)
			if err != nil {
				return err
			}
			
			err = b.Put([]byte(today), jsonData)
			if err != nil {
				return err
			}
			
			todayCount = todayTarget
		} else {
			var dayData DayData
			err := json.Unmarshal(data, &dayData)
			if err != nil {
				return err
			}
			todayCount = dayData.Count
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error initializing today count: %v", err)
	}
}

func basicAuth(next http.HandlerFunc, username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Push Up Tracker"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getFirstDay(tx *bolt.Tx) (string, error) {
	b := tx.Bucket([]byte("Config"))
	data := b.Get([]byte("firstDay"))
	if data == nil {
		return "", nil
	}
	return string(data), nil
}

func setFirstDay(tx *bolt.Tx, firstDay string) error {
	b := tx.Bucket([]byte("Config"))
	return b.Put([]byte("firstDay"), []byte(firstDay))
}

func handleToday(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(today))
		
		if data == nil {
			return fmt.Errorf("no data for today")
		}
		
		var dayData DayData
		err := json.Unmarshal(data, &dayData)
		if err != nil {
			return err
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dayData)
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleTodayComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	today := time.Now().Format("2006-01-02")
	
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		data := b.Get([]byte(today))
		
		var dayData DayData
		if data != nil {
			err := json.Unmarshal(data, &dayData)
			if err != nil {
				return err
			}
		} else {
			dayData = DayData{
				Date:  today,
				Count: todayCount,
				Done:  false,
			}
		}
		
		dayData.Done = true
		
		jsonData, err := json.Marshal(dayData)
		if err != nil {
			return err
		}
		
		err = b.Put([]byte(today), jsonData)
		if err != nil {
			return err
		}
		
		// Update streak
		updateStreak(tx, today)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dayData)
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func updateStreak(tx *bolt.Tx, today string) {
	b := tx.Bucket([]byte("Streak"))
	data := b.Get([]byte("current"))
	
	var streak StreakData
	if data != nil {
		json.Unmarshal(data, &streak)
	}
	
	todayTime, _ := time.Parse("2006-01-02", today)
	yesterday := todayTime.AddDate(0, 0, -1).Format("2006-01-02")
	
	// Check if yesterday was completed
	daysBucket := tx.Bucket([]byte("Days"))
	yesterdayData := daysBucket.Get([]byte(yesterday))
	
	if yesterdayData != nil {
		var yesterdayDayData DayData
		json.Unmarshal(yesterdayData, &yesterdayDayData)
		
		if yesterdayDayData.Done {
			streak.Current++
		} else {
			streak.Current = 1
		}
	} else {
		streak.Current = 1
	}
	
	if streak.Current > streak.Longest {
		streak.Longest = streak.Current
	}
	
	streak.LastDate = today
	
	jsonData, _ := json.Marshal(streak)
	b.Put([]byte("current"), jsonData)
}

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	if year == "" {
		year = strconv.Itoa(time.Now().Year())
	}
	
	var firstRecordDate string
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		
		cursor := b.Cursor()
		k, _ := cursor.First()
		if k != nil {
			firstRecordDate = string(k)
		}
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	var startMonth, startYear int
	if firstRecordDate != "" {
		firstDate, err := time.Parse("2006-01-02", firstRecordDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		startMonth = int(firstDate.Month() - 1) // Go months are 1-based, JS is 0-based
		startYear = firstDate.Year()
	} else {
		// No records, start from current month
		now := time.Now()
		startMonth = int(now.Month() - 1) // Convert to 0-based
		startYear = now.Year()
	}
	
	calendar := make(map[string]DayData)
	
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Days"))
		
		cursor := b.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			dateStr := string(k)
			if len(dateStr) >= 4 && dateStr[:4] == year {
				var dayData DayData
				err := json.Unmarshal(v, &dayData)
				if err != nil {
					continue
				}
				calendar[dateStr] = dayData
			}
		}
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := struct {
		Year         int                  `json:"year"`
		StartMonth   int                  `json:"startMonth"`
		StartYear    int                  `json:"startYear"`
		Days         map[string]DayData   `json:"days"`
	}{
		Year:       time.Now().Year(),
		StartMonth: startMonth,
		StartYear:  startYear,
		Days:       calendar,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStreak(w http.ResponseWriter, r *http.Request) {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Streak"))
		data := b.Get([]byte("current"))
		
		var streak StreakData
		if data != nil {
			err := json.Unmarshal(data, &streak)
			if err != nil {
				return err
			}
		} else {
			streak = StreakData{
				Current:  0,
				Longest:  0,
				LastDate: "",
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(streak)
		return nil
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}