#!/bin/bash
# Quick script to connect to RDS from local machine

AWS_REGION=${AWS_REGION:-ap-south-1}

echo "=========================================="
echo "Connecting to RDS Database"
echo "=========================================="

# Get database credentials
echo "Getting database credentials..."
export DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null)

if [ -z "$DB_HOST" ] || [ "$DB_HOST" == "None" ]; then
    echo "❌ Error: Could not get database host"
    exit 1
fi

export DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region $AWS_REGION \
    --query 'SecretString' \
    --output text 2>/dev/null)

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Error: Could not get database password"
    exit 1
fi

export DB_USER="postgres"
export DB_NAME="syncflow"
export DB_PORT="5432"

echo "✓ Database Host: $DB_HOST"
echo "✓ Database Name: $DB_NAME"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "❌ Error: 'psql' command not found."
    echo ""
    echo "Please install PostgreSQL client:"
    echo "  macOS:   brew install postgresql"
    echo "  Ubuntu:  sudo apt-get install postgresql-client"
    echo ""
    exit 1
fi

echo "Checking if database '$DB_NAME' exists..."
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p 5432 -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" --set=sslmode=require >/dev/null 2>&1; then
    echo "✓ Database '$DB_NAME' exists. Connecting..."
    echo ""
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p 5432 \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --set=sslmode=require
else
    echo "⚠️  Database '$DB_NAME' doesn't exist yet."
    echo "   Connecting to 'postgres' database instead."
    echo ""
    echo "To create the database, run:"
    echo "  CREATE DATABASE syncflow;"
    echo ""
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p 5432 \
        -U "$DB_USER" \
        -d postgres \
        --set=sslmode=require
fi
