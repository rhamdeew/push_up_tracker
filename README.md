# Push Up Tracker

A simple web application for tracking daily push-ups with BoltDB storage and basic authentication.

![CI/CD Status](https://github.com/rhamdeew/push_up_tracker/actions/workflows/release.yml/badge.svg)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/rhamdeew/push_up_tracker)](https://github.com/rhamdeew/push_up_tracker/releases/latest)
[![GitHub license](https://img.shields.io/github/license/rhamdeew/push_up_tracker)](https://github.com/rhamdeew/push_up_tracker/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhamdeew/push_up_tracker)](https://goreportcard.com/report/github.com/rhamdeew/push_up_tracker)
[![GitHub stars](https://img.shields.io/github/stars/rhamdeew/push_up_tracker)](https://github.com/rhamdeew/push_up_tracker/stargazers)

![Push Up Tracker](https://github.com/user-attachments/assets/4f5ec9b2-b9c8-41d9-acb4-7458a3c725df)

## Features

- Single-user push-up tracking
- Progressive daily targets with structured progression
- Visual calendar with completion tracking
- Current and longest streak tracking
- BoltDB for local data storage
- Basic authentication support
- Responsive web interface

## Daily Target Progression

The application uses a structured progression system that increases push-up targets based on your current level:

| Phase | Target Range | Increment Rule | Description |
|-------|-------------|----------------|-------------|
| Beginner | 10-49 | +2 per day | Quick progression to build foundation |
| Intermediate | 50-99 | +1 per day | Steady progression as difficulty increases |
| Advanced | 100-199 | +1 every 2 days | Slower progression for high-volume training |
| Elite | 200+ | No increase | Maximum daily target reached |

### Progression Timeline

- **Day 1**: Start with 10 push-ups
- **Day 20-25**: Reach 50 push-ups
- **Day 70-75**: Reach 100 push-ups  
- **Day 269**: Reach maximum of 200 push-ups
- **Day 270+**: Maintain at 200 push-ups

This progression creates a balanced training program that:
1. Builds initial strength quickly with +2 daily increases
2. Provides steady progression with +1 daily increases in the intermediate phase
3. Implements recovery time with +1 every 2 days in the advanced phase
4. Prevents overtraining by capping at 200 push-ups per day

## Quick Start with Make

### Development
```bash
# Install dependencies
make deps

# Run locally with default settings (port 3000, admin/admin)
make run

# Build only
make build

# Clean build artifacts and database
make clean
```

### Production Deployment
```bash
# Step 1: Install binary and service
make install

# Step 2: Edit configuration
sudo nano /opt/push_up_tracker/.env
# Change PORT, USERNAME, PASSWORD as needed

# Step 3: Start the service
sudo systemctl start push_up_tracker

# Check status
sudo systemctl status push_up_tracker

# View logs
sudo journalctl -u push_up_tracker -f
```

### Configuration
The application can be configured using a `.env` file or environment variables:

- `PORT` - Server port (default: 8080)
- `USERNAME` - Basic auth username (default: admin)
- `PASSWORD` - Basic auth password (default: admin)

During installation, `.env.example` is copied to `/opt/push_up_tracker/.env`. Edit this file to customize your configuration.

### Uninstall
```bash
# Stop service and remove all files
make uninstall
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build `push_up_tracker` binary |
| `make run` | Build and run with default settings (port 3000, admin/admin) |
| `make clean` | Remove build artifacts and database |
| `make deps` | Download and tidy Go dependencies |
| `make test` | Run Go tests with verbose output |
| `make test-coverage` | Run tests with coverage report (generates coverage.html) - Current coverage: ~6% |
| `make lint` | Run Go linter (requires golangci-lint) |
| `make generate-service` | Generate systemd service from template |
| `make install` | Install binary and systemd service |
| `make uninstall` | Remove binary and systemd service |
| `make help` | Show all available targets |

## Manual Installation

If you prefer not to use Make:

1. Clone or download repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build and run:
   ```bash
   go run main.go
   ```

## Configuration

Configure application using environment variables:

- `PORT`: Server port (default: 8080)
- `USERNAME`: Basic auth username (default: admin)
- `PASSWORD`: Basic auth password (default: admin)

Example:
```bash
PORT=3000 USERNAME=myuser PASSWORD=mypass go run main.go
```

## Usage

1. Start the application
2. Open your browser and navigate to `http://localhost:8080`
3. Log in with the configured credentials
4. View your daily push-up target
5. Click "Complete Today's Push-ups" when done
6. Track your progress in the calendar view
7. Monitor your current and longest streaks

## API Endpoints

- `GET /`: Main web interface
- `GET /api/today`: Get today's push-up data
- `POST /api/today/complete`: Mark today as completed
- `GET /api/calendar?year=2024`: Get calendar data for specified year
- `GET /api/streak`: Get current and longest streak information

## Data Storage

The application uses BoltDB for local storage:
- Data is stored in `pushups.db` file
- Days bucket: Daily push-up records
- Streak bucket: Current and longest streak data
- Config bucket: Application configuration and first record tracking

## Development

The application consists of:
- `main.go`: Go backend with web server and API
- `templates/index.html`: Main web interface
- `static/style.css`: Responsive CSS styling
- `static/app.js`: Frontend JavaScript functionality
- `go.mod`: Go module dependencies

## Push-up Progression Logic

The application uses a progressive overload system:
- Starting point: 5 push-ups on the day the database is first initialized
- Increment: +1 push-up each subsequent day
- Formula: `daily_target = 5 + days_since_database_creation`
- The first day count is calculated from when the database is created, not a fixed date
- This ensures gradual strength improvement over time

## Security

The application includes several security features when deployed using `make install`:

### Systemd Security
- Runs under `nobody` user (non-privileged)
- `NoNewPrivileges=true` - Process cannot gain new privileges
- `ProtectSystem=strict` - Read-only system access
- `ProtectHome=true` - No access to home directories
- `ReadWritePaths=/opt/push_up_tracker` - Only app directory writable
- `PrivateTmp=true` - Isolated temporary directory
- Additional hardening flags enabled

### Application Security
- Basic authentication required for all endpoints
- Database file created with secure permissions (0600)
- Static file serving validates paths to prevent directory traversal
- Sensitive files (.go, .db) not accessible via web
- Input validation on all user data

### File Permissions
- Binary owned by root, executable by nobody
- Templates and static files owned by nobody with read permissions
- Database file created in working directory with restricted access

⚠️ **Important**: Change default credentials before production deployment!

## Calendar Features

- Calendar only displays months starting from first record in database
- Previous months can be toggled on/off for viewing historical data
- Responsive layout: 4 months per row on desktop, 1 per row on mobile
- Visual indicators for completed days and current date
- Clean, modern interface with hover effects

## License

This project is open source and available under the MIT License.
