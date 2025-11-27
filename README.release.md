# Push Up Tracker - Release Package

A simple web application for tracking daily push-ups with progressive targets.

## Quick Start

### Linux

1. Extract the archive:
   ```bash
   tar -xzvf push_up_tracker_linux_amd64.tar.gz
   cd push_up_tracker_linux_amd64
   ```

2. Install the application:
   ```bash
   make install
   ```

3. Edit the configuration:
   ```bash
   sudo nano /opt/push_up_tracker/.env
   ```
   Change `PORT`, `USERNAME`, and `PASSWORD` as needed.

4. Start the service:
   ```bash
   sudo systemctl start push_up_tracker
   sudo systemctl status push_up_tracker
   ```

5. Access the application:
   Open your browser to `http://localhost:8080` (or your configured port)

### macOS

1. Extract the archive:
   ```bash
   tar -xzvf push_up_tracker_darwin_amd64.tar.gz
   cd push_up_tracker_darwin_amd64
   ```

2. Run manually:
   ```bash
   PORT=3000 USERNAME=admin PASSWORD=admin ./push_up_tracker
   ```

3. Access the application:
   Open your browser to `http://localhost:3000`

Note: The `make install` target uses systemd, which is not available on macOS.

### Windows

1. Extract the ZIP archive
2. Open Command Prompt in the extracted folder
3. Set environment variables and run:
   ```cmd
   set PORT=3000
   set USERNAME=admin
   set PASSWORD=admin
   push_up_tracker.exe
   ```

4. Access the application:
   Open your browser to `http://localhost:3000`

## Configuration

The application can be configured using environment variables or a `.env` file:

- `PORT` - Server port (default: 8080)
- `USERNAME` - Basic auth username (default: admin)
- `PASSWORD` - Basic auth password (default: admin)

## Uninstall (Linux)

```bash
make uninstall
```

## Package Contents

- `push_up_tracker` - Main application binary
- `templates/` - HTML templates
- `static/` - CSS and JavaScript files
- `.env.example` - Example configuration file
- `Makefile` - Installation/uninstallation script (Linux/macOS)
- `push_up_tracker.service` - systemd service file (Linux only)

## Features

- Single-user push-up tracking
- Progressive daily targets with structured progression
- Visual calendar with completion tracking
- Current and longest streak tracking
- BoltDB for local data storage
- Basic authentication support
- Responsive web interface

## Support

For issues, documentation, and source code, visit:
https://github.com/yourusername/push_up_tracker

## Security Notice

⚠️ **Important**: Change the default credentials before production deployment!

The application includes security features when deployed with systemd:
- Runs under `nobody` user (non-privileged)
- Protected system and home directories
- Restricted file access
- Additional hardening flags enabled
