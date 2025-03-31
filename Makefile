# Postgres.app Installation Makefile for macOS
# Variables
POSTGRES_URL=https://github.com/PostgresApp/PostgresApp/releases/download/v2.8.1/Postgres-2.8.1-17.dmg
POSTGRES_DMG=/Users/prateekkumar/Downloads/Postgres.dmg
APP_DIR=/Applications
DB_URL=postgres://prateekkumar@localhost:5432/chat_app?sslmode=disable

.PHONY: postgres_install postgres_uninstall postgres_start postgres_stop \
	    migrate-up migrate-down migrate-reset migrate-create migrate-version migrate-force \
	    db_setup db_reset postgres_setup_complete postgres_optimize setup_pgbouncer setup_for_production help

postgres_install:
	@echo "==> Downloading Postgres.app (this might take a while)..."
	curl -L -o "$(POSTGRES_DMG)" "$(POSTGRES_URL)"
	@echo "==> Mounting disk image..."
	hdiutil attach "$(POSTGRES_DMG)" || { \
		echo "Error: Failed to mount the disk image. The download might be incomplete or corrupted."; \
		exit 1; \
	}
	@echo "==> Finding mount point..."
	@MOUNT_POINT=$$(find /Volumes -maxdepth 1 -name "Postgres*" -type d | head -n 1) && \
	if [ -z "$$MOUNT_POINT" ]; then \
		echo "Error: Could not find mounted Postgres volume"; \
		exit 1; \
	fi && \
	echo "    Mount point: $$MOUNT_POINT" && \
	echo "==> Copying Postgres.app to Applications folder..." && \
	cp -R "$$MOUNT_POINT/Postgres.app" "$(APP_DIR)/" && \
	echo "==> Cleaning up..." && \
	hdiutil detach "$$MOUNT_POINT" && \
	echo "==> Removing downloaded DMG file..." && \
	rm -f "$(POSTGRES_DMG)" && \
	echo "==> Postgres.app installation complete!"

postgres_uninstall:
	@echo "==> Uninstalling Postgres.app..."
	@echo "    First stopping any running PostgreSQL instances..."
	@if pgrep -f "Postgres.app" > /dev/null; then \
		osascript -e 'tell application "Postgres" to quit'; \
		echo "    Waiting for PostgreSQL to shut down..."; \
		sleep 3; \
		if pgrep -f "postgres:" > /dev/null; then \
			echo "    Force killing remaining PostgreSQL processes..."; \
			pkill -9 -f "postgres:"; \
		fi; \
	else \
		echo "    Postgres.app is not running."; \
	fi
	@echo "    Removing Postgres.app from Applications folder..."
	rm -rf "$(APP_DIR)/Postgres.app"
	@echo "    Cleaning up data files..."
	rm -rf "$(HOME)/Library/Application Support/Postgres"
	@echo "==> Postgres.app uninstalled successfully."

postgres_start:
	@echo "==> Starting Postgres.app..."
	@if pgrep -f "Postgres.app" > /dev/null; then \
		echo "    Postgres.app is already running."; \
	else \
		open -a Postgres.app; \
		echo "    Waiting for PostgreSQL to initialize (10 seconds)..."; \
		sleep 10; \
		if psql -c "SELECT 1;" >/dev/null 2>&1; then \
			echo "    PostgreSQL is now running and accepting connections."; \
		else \
			echo "    WARNING: PostgreSQL might not be fully initialized yet."; \
			echo "    You may need to manually initialize it through the Postgres.app interface."; \
			echo "    Once initialized, try running 'make db_setup' again."; \
		fi; \
	fi

postgres_stop:
	@echo "==> Stopping Postgres.app..."
	@if pgrep -f "Postgres.app" > /dev/null; then \
		osascript -e 'tell application "Postgres" to quit'; \
		echo "Waiting for PostgreSQL to shut down..."; \
		sleep 3; \
		if pgrep -f "postgres:" > /dev/null; then \
			echo "Force killing remaining PostgreSQL processes..."; \
			pkill -9 -f "postgres:"; \
		fi; \
		echo "PostgreSQL stopped."; \
	else \
		echo "Postgres.app is not running."; \
	fi

# Database setup commands
db_setup:
	@echo "==> Setting up chat application database..."
	@echo "    Creating database 'chat_app'..."
	createdb chat_app || echo "Database might already exist"
	@echo "    Creating user 'prateekkumar'..."
	psql postgres -c "CREATE USER prateekkumar;" || echo "User might already exist"
	@echo "    Granting privileges..."
	psql postgres -c "GRANT ALL PRIVILEGES ON DATABASE chat_app TO prateekkumar;"
	@echo "    Installing uuid-ossp extension..."
	psql chat_app -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
	@echo "==> Database setup complete! You can now run migrations."

db_reset:
	@echo "==> Resetting chat application database..."
	@echo "    Dropping database 'chat_app'..."
	dropdb chat_app || echo "Database might not exist"
	@echo "    Creating database 'chat_app'..."
	createdb chat_app
	@echo "    Granting privileges..."
	psql postgres -c "GRANT ALL PRIVILEGES ON DATABASE chat_app TO prateekkumar;"
	@echo "==> Database reset complete! You can now run migrations."

# Migration commands
migrate-up:
	@echo "==> Applying database migrations..."
	migrate -path migrations -database "$(DB_URL)" -verbose up
	@echo "==> Migrations applied successfully."

migrate-down:
	@echo "==> Reverting last migration..."
	migrate -path migrations -database "$(DB_URL)" -verbose down 1
	@echo "==> Last migration reverted."

migrate-reset:
	@echo "==> Reverting all migrations..."
	migrate -path migrations -database "$(DB_URL)" -verbose down
	@echo "==> All migrations reverted."

migrate-create:
	@read -p "Enter migration name: " name; \
	echo "==> Creating new migration '$$name'..."; \
	migrate create -ext sql -dir migrations -seq $$name; \
	echo "==> Migration files created."

migrate-version:
	@echo "==> Current migration version:"
	@migrate -path migrations -database "$(DB_URL)" version

migrate-force:
	@read -p "Enter version to force: " version; \
	echo "==> Forcing migration to version $$version..."; \
	migrate -path migrations -database "$(DB_URL)" force $$version; \
	echo "==> Migration version forced to $$version."

postgres_setup_complete:
	@echo "==> Starting complete database setup process..."
	@echo "==> Step 1: Installing Postgres.app..."
	$(MAKE) postgres_install
	@echo "==> Step 2: Starting Postgres.app..."
	$(MAKE) postgres_start
	@echo "==> Step 3: Waiting for PostgreSQL to initialize (20 seconds)..."
	@sleep 20
	@echo "==> Step 4: Setting up database and user..."
	$(MAKE) db_setup
	@echo "==> Step 5: Applying all migrations..."
	$(MAKE) migrate-up
	@echo "==> Complete setup finished successfully!"
	@echo "Your chat application database is now ready to use."

help:
	@echo "Postgres.app and Migration Management Commands:"
	@echo ""
	@echo "  postgres_install   - Download and install Postgres.app"
	@echo "  postgres_uninstall - Remove Postgres.app"
	@echo "  postgres_start     - Start Postgres.app"
	@echo "  postgres_stop      - Stop Postgres.app"
	@echo ""
	@echo "  db_setup           - Create database and user for chat application"
	@echo "  db_reset           - Drop and recreate database"
	@echo ""
	@echo "  migrate-up         - Apply all pending migrations"
	@echo "  migrate-down       - Revert the last applied migration"
	@echo "  migrate-reset      - Revert all migrations"
	@echo "  migrate-create     - Create a new migration file"
	@echo "  migrate-version    - Show current migration version"
	@echo "  migrate-force      - Force migration version (use with caution)"
	@echo "  postgres_setup_complete  - Run all tasks from installing Postgres app to applying all schema migrations."
	@echo "  postgres_optimize  - Optimize PostgreSQL configuration for 500 concurrent users"
	@echo "  setup_pgbouncer    - Install and configure PgBouncer connection pooler"
	@echo "  setup_for_production - Configure PostgreSQL and connection pooling for high concurrency"
	@echo ""
	@echo "  help               - Show this help information"