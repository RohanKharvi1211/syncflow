#!/bin/bash
# Simple script to create database - can be run from anywhere
# Usage: ./create-db-simple.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}

echo "=========================================="
echo "Creating Database via AWS RDS Data API"
echo "=========================================="

# Get database credentials
echo "Getting database credentials..."
DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region $AWS_REGION \
    --query 'SecretString' \
    --output text)

DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

DB_PORT=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text)

echo "✓ Retrieved database information"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo ""

# Check if we can connect (requires psql)
if ! command -v psql &> /dev/null; then
    echo "❌ Error: 'psql' command not found."
    echo ""
    echo "Please install PostgreSQL client:"
    echo "  macOS: brew install postgresql"
    echo "  Ubuntu: sudo apt-get install postgresql-client"
    echo "  Or use AWS Console (RDS Query Editor v2)"
    echo ""
    echo "Alternatively, let the code handle it automatically:"
    echo "  git add backend/cmd/app/app.go"
    echo "  git commit -m 'Fix: Auto-create database'"
    echo "  git push origin main"
    exit 1
fi

echo "Creating database 'syncflow'..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U postgres \
    -d postgres \
    -c "CREATE DATABASE syncflow;" 2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ Database 'syncflow' created successfully!"
    echo "=========================================="
else
    echo ""
    echo "⚠️  Database creation failed or database already exists."
    echo "   Check the error message above."
    echo ""
    echo "If database already exists, that's fine - you can proceed with deployment."
fi

