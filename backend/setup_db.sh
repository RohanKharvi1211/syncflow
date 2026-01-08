#!/bin/bash

# Database Setup Script for SyncFlow Backend

echo "Setting up PostgreSQL database for SyncFlow..."

# Check if PostgreSQL is running
if ! pg_isready -q; then
    echo "ERROR: PostgreSQL is not running. Please start PostgreSQL first."
    echo "On macOS: brew services start postgresql"
    exit 1
fi

# Get current user
CURRENT_USER=$(whoami)

# Try to create database with current user first
echo "Attempting to create database with user: $CURRENT_USER"

# Create database
createdb syncflow 2>/dev/null && echo "✓ Database 'syncflow' created" || {
    echo "Database might already exist or permission issue."
    echo "Trying with postgres user..."
    
    # Try with postgres user
    psql -U postgres -c "CREATE DATABASE syncflow;" 2>/dev/null && echo "✓ Database 'syncflow' created" || {
        echo "⚠ Could not create database. You may need to:"
        echo "  1. Create it manually: createdb syncflow"
        echo "  2. Or use: psql -U postgres -c 'CREATE DATABASE syncflow;'"
    }
}

# Run migrations
echo ""
echo "Running database migrations..."
if [ -f "migrations/postgres/20250101000000_init_schema.up.sql" ]; then
    psql syncflow -f migrations/postgres/20250101000000_init_schema.up.sql && echo "✓ Initial schema migrated"
    psql syncflow -f migrations/postgres/20250101000001_legacy_tables.up.sql 2>/dev/null && echo "✓ Legacy tables migrated" || echo "⚠ Legacy tables migration skipped (optional)"
else
    echo "⚠ Migration files not found. Database will be auto-migrated on first run."
fi

echo ""
echo "Database setup complete!"
echo ""
echo "Next steps:"
echo "1. Update your .env file with correct database credentials"
echo "2. Start the backend server: go run ./cmd/server/main.go"
echo "3. Access frontend at http://localhost:3000"

