# Push Up Tracker

A simple web application for tracking daily push-ups with BoltDB storage and basic authentication.

## Features

- Single-user push-up tracking
- Progressive daily targets (starts at 5 push-ups, increases by 1 each day)
- Visual calendar with completion tracking
- Current and longest streak tracking
- BoltDB for local data storage
- Basic authentication support
- Responsive web interface

## Installation

1. Clone or download the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build and run:
   ```bash
   go run main.go
   ```

## Configuration

Configure the application using environment variables:

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

## Deployment as Systemd Service

1. Build the application:
   ```bash
   go build -o push_up_tracker main.go
   ```

2. Copy files to deployment directory:
   ```bash
   sudo cp -r . /opt/push_up_tracker/
   sudo chown -R nobody:nogroup /opt/push_up_tracker
   ```

3. Install the systemd service:
   ```bash
   sudo cp push-up-tracker.service /etc/systemd/system/push_up_tracker.service
   sudo systemctl daemon-reload
   sudo systemctl enable push_up_tracker
   sudo systemctl start push_up_tracker
   ```

4. Check status:
   ```bash
   sudo systemctl status push_up_tracker
   ```

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

## Calendar Features

- Calendar only displays months starting from the first record in the database
- Previous months can be toggled on/off for viewing historical data
- Responsive layout: 4 months per row on desktop, 1 per row on mobile
- Visual indicators for completed days and current date
- Clean, modern interface with hover effects

## License

This project is open source and available under the MIT License.