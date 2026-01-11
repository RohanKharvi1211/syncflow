#!/bin/bash
# Script to fix companies table UUID issue
# This converts the companies.id column from INTEGER to UUID if needed

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "=========================================="
echo "Fixing Companies Table UUID Schema"
echo "=========================================="
echo ""

# Get database credentials from Secrets Manager
DB_HOST=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/host" --region $AWS_REGION --query 'SecretString' --output text)
DB_USER=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/user" --region $AWS_REGION --query 'SecretString' --output text)
DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/password" --region $AWS_REGION --query 'SecretString' --output text)
DB_NAME=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/name" --region $AWS_REGION --query 'SecretString' --output text)
DB_PORT=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/port" --region $AWS_REGION --query 'SecretString' --output text)

echo "Connecting to database: $DB_NAME@$DB_HOST:$DB_PORT"
echo ""

# Check current column type
echo "Checking companies.id column type..."
COLUMN_TYPE=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -t -c "SELECT data_type FROM information_schema.columns WHERE table_name = 'companies' AND column_name = 'id';" 2>/dev/null | xargs)

if [ -z "$COLUMN_TYPE" ]; then
    echo "❌ Error: Could not connect to database or companies table doesn't exist"
    exit 1
fi

echo "Current companies.id type: $COLUMN_TYPE"
echo ""

if [ "$COLUMN_TYPE" = "uuid" ]; then
    echo "✅ Companies table already has UUID id. No conversion needed."
    exit 0
fi

if [ "$COLUMN_TYPE" != "integer" ] && [ "$COLUMN_TYPE" != "bigint" ]; then
    echo "⚠️  Warning: Unexpected column type: $COLUMN_TYPE"
    echo "   Expected: integer, bigint, or uuid"
    read -p "Continue with conversion? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "⚠️  Companies table has INTEGER id, but code expects UUID"
echo "   Running conversion migration..."
echo ""

# Run the conversion migration
MIGRATION_FILE="backend/migrations/postgres/20250101000005_convert_companies_to_uuid.up.sql"

if [ ! -f "$MIGRATION_FILE" ]; then
    echo "❌ Error: Migration file not found: $MIGRATION_FILE"
    exit 1
fi

echo "Executing migration: $MIGRATION_FILE"
echo ""

# Execute migration with error handling
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -f "$MIGRATION_FILE" 2>&1 | tee /tmp/migration-output.log

MIGRATION_EXIT_CODE=${PIPESTATUS[0]}

if [ $MIGRATION_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ Migration completed successfully!"
    
    # Verify the conversion
    NEW_TYPE=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -t -c "SELECT data_type FROM information_schema.columns WHERE table_name = 'companies' AND column_name = 'id';" 2>/dev/null | xargs)
    echo "New companies.id type: $NEW_TYPE"
    
    if [ "$NEW_TYPE" = "uuid" ]; then
        echo "✅ Conversion verified! Companies table now uses UUID."
    else
        echo "⚠️  Warning: Conversion may have failed. Type is still: $NEW_TYPE"
    fi
else
    echo ""
    echo "❌ Migration failed with exit code: $MIGRATION_EXIT_CODE"
    echo "Check /tmp/migration-output.log for details"
    exit 1
fi

echo ""
echo "=========================================="
echo "✓ Companies UUID Conversion Complete!"
echo "=========================================="

