#!/bin/bash

# Migration Runner Script
# This script runs database migrations using psql

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Database connection parameters (can be overridden by environment variables)
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-syncflow}
DB_PASSWORD=${DB_PASSWORD:-}

echo -e "${GREEN}Running database migrations...${NC}"

# Check if PostgreSQL is running
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -q; then
    echo -e "${RED}ERROR: PostgreSQL is not running or not accessible.${NC}"
    echo "Please ensure PostgreSQL is running and accessible at $DB_HOST:$DB_PORT"
    exit 1
fi

# Set PGPASSWORD if provided
if [ -n "$DB_PASSWORD" ]; then
    export PGPASSWORD="$DB_PASSWORD"
fi

# Migration files directory
MIGRATIONS_DIR="migrations/postgres"

# Check if migrations directory exists
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo -e "${RED}ERROR: Migrations directory not found: $MIGRATIONS_DIR${NC}"
    exit 1
fi

# Function to run a migration file
run_migration() {
    local file=$1
    local direction=$2
    
    if [ "$direction" = "up" ]; then
        echo -e "${YELLOW}Running migration: $(basename $file)${NC}"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file" -q
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Migration completed: $(basename $file)${NC}"
        else
            echo -e "${RED}✗ Migration failed: $(basename $file)${NC}"
            exit 1
        fi
    fi
}

# Run migrations in order
echo -e "${GREEN}Running initial schema migration...${NC}"
if [ -f "$MIGRATIONS_DIR/20250101000000_init_schema.up.sql" ]; then
    run_migration "$MIGRATIONS_DIR/20250101000000_init_schema.up.sql" "up"
fi

echo -e "${GREEN}Running legacy tables migration...${NC}"
if [ -f "$MIGRATIONS_DIR/20250101000001_legacy_tables.up.sql" ]; then
    run_migration "$MIGRATIONS_DIR/20250101000001_legacy_tables.up.sql" "up"
fi

echo -e "${GREEN}Running restructured schema migration...${NC}"
if [ -f "$MIGRATIONS_DIR/20250101000002_restructured_schema.up.sql" ]; then
    run_migration "$MIGRATIONS_DIR/20250101000002_restructured_schema.up.sql" "up"
fi

echo -e "${GREEN}Running seed apps migration...${NC}"
if [ -f "$MIGRATIONS_DIR/20250101000003_seed_apps.up.sql" ]; then
    run_migration "$MIGRATIONS_DIR/20250101000003_seed_apps.up.sql" "up"
fi

echo -e "${GREEN}All migrations completed successfully!${NC}"




