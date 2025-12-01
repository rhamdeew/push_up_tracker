.PHONY: build run clean all test test-coverage deps install update uninstall help

# Variables
BINARY_NAME=push_up_tracker
SERVICE_FILE=push_up_tracker.service
ENV_FILE=.env
ENV_EXAMPLE=.env.example
INSTALL_DIR=/opt/push_up_tracker
SERVICE_DIR=/etc/systemd/system

# Default target
all: help

# Build the binary
build:
	go build -o $(BINARY_NAME) main.go

# Run the application with default settings
run: build
	PORT=3000 USERNAME=admin PASSWORD=admin ./$(BINARY_NAME)

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f pushups.db

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install binary and systemd service
install: build
	@echo "Installing push_up_tracker..."
	
	# Create installation directory
	sudo mkdir -p $(INSTALL_DIR)
	
	# Copy binary and files
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	sudo cp -r templates $(INSTALL_DIR)/
	sudo cp -r static $(INSTALL_DIR)/
	sudo cp $(SERVICE_FILE) $(SERVICE_DIR)/
	
	# Copy .env.example to .env if .env doesn't exist
	@if [ ! -f $(INSTALL_DIR)/$(ENV_FILE) ]; then \
		sudo cp $(ENV_EXAMPLE) $(INSTALL_DIR)/$(ENV_FILE); \
		echo "Created $(INSTALL_DIR)/$(ENV_FILE) from $(ENV_EXAMPLE)"; \
		echo "Edit $(INSTALL_DIR)/$(ENV_FILE) to customize settings:"; \
		echo "  - PORT (default: 8080)"; \
		echo "  - USERNAME (default: admin)"; \
		echo "  - PASSWORD (default: admin)"; \
	else \
		echo "Using existing $(INSTALL_DIR)/$(ENV_FILE)"; \
	fi

	# Create database file with proper ownership
	sudo touch $(INSTALL_DIR)/pushups.db
	sudo chown nobody:nogroup $(INSTALL_DIR)/pushups.db
	sudo chmod 600 $(INSTALL_DIR)/pushups.db

	# Set permissions (more restrictive)
	sudo chown -R root:root $(INSTALL_DIR)
	sudo chown nobody:nogroup $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chown -R nobody:nogroup $(INSTALL_DIR)/templates
	sudo chown -R nobody:nogroup $(INSTALL_DIR)/static
	sudo chown nobody:nogroup $(INSTALL_DIR)/$(ENV_FILE)
	sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chmod 644 $(INSTALL_DIR)/templates/*
	sudo chmod 644 $(INSTALL_DIR)/static/*
	sudo chmod 644 $(INSTALL_DIR)/$(ENV_FILE)
	
	# Reload systemd and enable service
	sudo systemctl daemon-reload
	sudo systemctl enable $(SERVICE_FILE:.service=)
	
	@echo "Installation complete!"
	@echo "Start the service with: sudo systemctl start $(SERVICE_FILE:.service=)"
	@echo ""
	@echo "Configuration loaded from: $(INSTALL_DIR)/$(ENV_FILE)"
	@echo ""
	@echo "Security notice:"
	@echo "- Application runs under 'nobody' user with restricted permissions"
	@echo "- Only files in $(INSTALL_DIR) are writable"
	@echo "- Database file: $(INSTALL_DIR)/pushups.db (owned by nobody)"
	@echo "- Change default credentials before production use!"
	@echo "Check status with: sudo systemctl status $(SERVICE_FILE:.service=)"

# Update existing installation
update: build
	@echo "Updating push_up_tracker..."

	# Stop the service
	sudo systemctl stop $(SERVICE_FILE:.service=)

	# Update binary and files
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	sudo cp -r templates $(INSTALL_DIR)/
	sudo cp -r static $(INSTALL_DIR)/

	# Restore permissions
	sudo chown nobody:nogroup $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chown -R nobody:nogroup $(INSTALL_DIR)/templates
	sudo chown -R nobody:nogroup $(INSTALL_DIR)/static
	sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chmod 644 $(INSTALL_DIR)/templates/*
	sudo chmod 644 $(INSTALL_DIR)/static/*

	# Restart the service
	sudo systemctl start $(SERVICE_FILE:.service=)

	@echo "Update complete!"
	@echo "Service restarted."
	@echo "Check status with: sudo systemctl status $(SERVICE_FILE:.service=)"
	@echo "Note: .env configuration and database were preserved"

# Uninstall binary and systemd service
uninstall:
	# Stop and disable service
	sudo systemctl stop $(SERVICE_FILE:.service=) || true
	sudo systemctl disable $(SERVICE_FILE:.service=) || true
	
	# Remove files
	sudo rm -f $(SERVICE_DIR)/$(SERVICE_FILE)
	sudo rm -rf $(INSTALL_DIR)
	
	# Reload systemd
	sudo systemctl daemon-reload
	
	@echo "Uninstallation complete!"
	@echo "Note: Service configuration remains in your git repo"

# Show available targets
help:
	@echo "Available targets:"
	@echo "  build                   - Build the push_up_tracker binary"
	@echo "  run                     - Build and run with default settings (port 3000)"
	@echo "  clean                   - Remove build artifacts and database"
	@echo "  deps                    - Download and tidy Go dependencies"
	@echo "  test                    - Run tests"
	@echo "  test-coverage           - Run tests with coverage report"
	@echo "  install                 - Install binary and systemd service"
	@echo "  update                  - Update existing installation (preserves .env and database)"
	@echo "  uninstall               - Remove binary and systemd service"
	@echo "  help                    - Show this help message"
	@echo ""
	@echo "Configuration:"
	@echo "  .env.example            - Example environment file"
	@echo "  .env                    - Environment file (created during install)"
	@echo "  push_up_tracker.service - systemd service file"
	@echo ""
	@echo "Workflow:"
	@echo "  1. make install"
	@echo "  2. Edit /opt/push_up_tracker/.env to customize settings"
	@echo "  3. sudo systemctl start push_up_tracker"
	@echo ""
	@echo "Update workflow:"
	@echo "  1. git pull (or download new version)"
	@echo "  2. make update"
