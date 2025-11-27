# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Push Up Tracker is a single-user web application for tracking daily push-ups with progressive daily targets. The app uses BoltDB for local storage and provides a web interface with calendar view, streak tracking, and basic authentication.

## Core Architecture

### Single-File Design
The entire backend is in `main.go` (~480 lines). There are no separate packages or modules - everything is in the main package. This includes:
- Database initialization and schema creation
- HTTP handlers for web interface and API endpoints
- Authentication middleware
- Push-up progression calculation logic
- Streak tracking logic

### Data Storage (BoltDB)
The app uses three BoltDB buckets in `pushups.db`:
- `Days`: Stores daily push-up records (key: "YYYY-MM-DD", value: JSON of `DayData`)
- `Streak`: Stores current and longest streak data (key: "current", value: JSON of `StreakData`)
- `Config`: Stores configuration like first record date (key: "firstDay", value: string date)

Database file location: `{WorkingDirectory}/pushups.db` (created with 0600 permissions)

### Push-up Progression System
The daily target is calculated in `calculateTarget()` based on days since database creation:
- **Days 0-19** (target 10-49): +2 push-ups per day
- **Days 20-69** (target 50-99): +1 push-up per day
- **Days 70-269** (target 100-199): +1 push-up every 2 days
- **Day 270+** (target 200): No increase, capped at 200

Starting count is always 10 on database initialization day. The `firstDay` in Config bucket tracks when progression started.

### Streak Logic
Streaks are updated in `updateStreak()` when a day is marked complete:
- If yesterday was completed: increment current streak
- If yesterday was NOT completed or doesn't exist: reset current to 1
- Always update longest streak if current exceeds it
- Streak state stored in Streak bucket with key "current"

## Development Commands

### Build and Run
```bash
# Build binary
make build

# Run with development settings (port 3000, admin/admin)
make run

# Clean build artifacts and database
make clean
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage report (generates coverage.html)
make test-coverage
```

Current test coverage is ~6% - test file `main_test.go` has comprehensive unit tests for individual functions but no integration tests for the full application.

### Dependencies
```bash
# Download and tidy dependencies
make deps
```

Dependencies:
- `github.com/boltdb/bolt v1.3.1` - Embedded key/value database
- `github.com/joho/godotenv v1.5.1` - Load .env files

### Production Installation
```bash
# Install to /opt/push_up_tracker with systemd service
make install

# Edit configuration (PORT, USERNAME, PASSWORD)
sudo nano /opt/push_up_tracker/.env

# Start/stop service
sudo systemctl start push_up_tracker
sudo systemctl stop push_up_tracker

# View logs
sudo journalctl -u push_up_tracker -f

# Uninstall everything
make uninstall
```

## API Endpoints

All endpoints require HTTP Basic Authentication.

- `GET /` - Render main web interface (uses templates/index.html)
- `GET /api/today` - Get today's push-up data (DayData JSON)
- `POST /api/today/complete` - Mark today as completed, updates streak
- `GET /api/calendar?year=YYYY` - Get calendar data for specified year
- `GET /api/streak` - Get current and longest streak information
- `GET /static/*` - Serve static files (style.css, app.js) with security validation

## Configuration

Environment variables (or .env file):
- `PORT` - Server port (default: 8080)
- `USERNAME` - Basic auth username (default: admin)
- `PASSWORD` - Basic auth password (default: admin)
- `PWD` - Working directory for database (default: current directory)

## Key Implementation Details

### Database Initialization
On startup, `initializeTodayCount()` checks if today's data exists. If not:
1. Check if `firstDay` exists in Config bucket
2. If no firstDay (first run): set firstDay to today, target to 10
3. If firstDay exists: calculate target based on days since firstDay
4. Create today's DayData with calculated target

### Static File Security
Static file handler validates paths to prevent directory traversal:
- Must start with "static/"
- Cannot contain ".."
- Cannot end with ".go"

### Template Rendering
Templates loaded from `templates/*.html` at startup using `template.ParseGlob()`. Only one template file exists: `index.html`.

### Global State
The app uses global variables (not ideal but functional for single-user app):
- `db *bolt.DB` - Database connection
- `tmpl *template.Template` - Compiled templates
- `todayCount int` - Today's target count
- `todayTarget int` - Today's target (seems redundant with todayCount)

## Testing Notes

When writing tests:
- Use `setupTestDB(t)` to create test database and `cleanupTestDB(t, db)` to clean up
- Save and restore global variables (db, tmpl, todayCount, todayTarget) in defer statements
- Test database file is `test.db` (automatically deleted after tests)
- Template tests should skip if templates not available

## Security Considerations

When deployed via `make install`:
- Runs under `nobody` user (non-privileged)
- Systemd security hardening enabled (NoNewPrivileges, ProtectSystem=strict, etc.)
- Database file owned by nobody with 0600 permissions
- Static file serving validates paths to prevent directory traversal
- All endpoints protected by basic auth
- Change default credentials before production deployment!

## File Structure

```
.
├── main.go              # Entire application backend
├── main_test.go         # Comprehensive unit tests
├── go.mod               # Go module definition
├── Makefile             # Build, test, and deployment targets
├── Makefile.release     # Minimal Makefile for release packages (install/uninstall only)
├── README.md            # Main project documentation
├── README.release.md    # Documentation for release packages
├── templates/
│   └── index.html       # Main web interface
├── static/
│   ├── app.js           # Frontend JavaScript
│   └── style.css        # Responsive CSS styling
├── .env.example         # Example configuration
├── push_up_tracker.service  # systemd service definition
└── .github/
    └── workflows/
        └── release.yml  # GitHub Actions release workflow
```

## Common Workflows

### Adding a New API Endpoint
1. Add handler function following pattern: `func handle<Name>(w http.ResponseWriter, r *http.Request)`
2. Register in `main()`: `http.HandleFunc("/api/path", basicAuth(handleName, username, password))`
3. Add corresponding test in `main_test.go`: `func TestHandle<Name>(t *testing.T)`

### Modifying Progression Logic
1. Update `calculateTarget(startCount, daysSince int)` function
2. Update corresponding tests in `TestProgressiveLoad`
3. Update README.md with new progression table
4. Note: Changes only affect NEW days, not existing records

### Debugging Database Issues
Database is created at runtime. To inspect:
```bash
# Install boltdb CLI tool
go install github.com/boltdb/bolt/cmd/bolt@latest

# Open database (if running locally)
bolt dump ./pushups.db

# Or check installed version
sudo bolt dump /opt/push_up_tracker/pushups.db
```

### Running Single Test
```bash
go test -v -run TestName
```

## Release Process

### Creating a Release

Releases are automated via GitHub Actions. To create a new release:

1. Tag a new version:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Run tests on all platforms
   - Build binaries for Linux, macOS, and Windows (amd64 and arm64)
   - Create release packages with all necessary files
   - Create a GitHub release with all artifacts

### Release Package Contents

Each release package includes:
- `push_up_tracker` (or `push_up_tracker.exe` for Windows) - Compiled binary
- `templates/` - HTML templates directory
- `static/` - CSS and JavaScript files directory
- `.env.example` - Example configuration file
- `README.md` - User documentation (from README.release.md)
- `Makefile` - Minimal Makefile with install/uninstall targets (Linux/macOS only)
- `push_up_tracker.service` - systemd service file (Linux only)

### Platform-Specific Builds

The workflow creates the following artifacts:
- `push_up_tracker_linux_amd64.tar.gz` - Linux x86_64 (includes Makefile + systemd service)
- `push_up_tracker_linux_arm64.tar.gz` - Linux ARM64 (includes Makefile + systemd service)
- `push_up_tracker_darwin_amd64.tar.gz` - macOS Intel (includes Makefile)
- `push_up_tracker_darwin_arm64.tar.gz` - macOS Apple Silicon (includes Makefile)
- `push_up_tracker_windows_amd64.zip` - Windows x86_64 (no Makefile/service)

### Modifying Release Workflow

Key files for releases:
- `.github/workflows/release.yml` - GitHub Actions workflow definition
- `Makefile.release` - Minimal Makefile copied to release packages as `Makefile`
- `README.release.md` - User-facing README copied to release packages as `README.md`

When adding new files to releases, update the "Copy assets" section in release.yml.
