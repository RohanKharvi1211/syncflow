#!/bin/bash
# Automated script to create all secrets with the RDS database details
# Usage: ./create-secrets-automated.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "=========================================="
echo "Creating AWS Secrets Manager Secrets"
echo "=========================================="
echo "Region: $AWS_REGION"
echo ""

# Get database details
DB_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier "${PROJECT_NAME}-db" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "")

DB_PORT=$(aws rds describe-db-instances \
    --db-instance-identifier "${PROJECT_NAME}-db" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "5432")

DB_PASSWORD=$(cat /tmp/${PROJECT_NAME}-db-password.txt 2>/dev/null || echo "")

if [ -z "$DB_ENDPOINT" ] || [ "$DB_ENDPOINT" == "None" ]; then
    echo "❌ Error: Could not get database endpoint. Is the database available?"
    exit 1
fi

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Error: Could not find database password in /tmp/${PROJECT_NAME}-db-password.txt"
    exit 1
fi

echo "Database Details:"
echo "  Endpoint: $DB_ENDPOINT"
echo "  Port: $DB_PORT"
echo "  Username: postgres"
echo "  Password: ****** (saved)"
echo ""

# Function to create or update secret
create_or_update_secret() {
    local secret_name=$1
    local secret_value=$2
    local description=$3
    
    echo "Creating/updating secret: $secret_name"
    aws secretsmanager create-secret \
        --name "$secret_name" \
        --secret-string "$secret_value" \
        --description "$description" \
        --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret \
        --secret-id "$secret_name" \
        --secret-string "$secret_value" \
        --region $AWS_REGION >/dev/null 2>&1
    echo "✓ $secret_name"
}

# Create database secrets
echo "=== Creating Database Secrets ==="
create_or_update_secret "${PROJECT_NAME}/database/host" "$DB_ENDPOINT" "RDS PostgreSQL host endpoint"
create_or_update_secret "${PROJECT_NAME}/database/user" "postgres" "Database username"
create_or_update_secret "${PROJECT_NAME}/database/password" "$DB_PASSWORD" "Database password"
create_or_update_secret "${PROJECT_NAME}/database/name" "syncflow" "Database name"
create_or_update_secret "${PROJECT_NAME}/database/port" "$DB_PORT" "Database port"
echo ""

# Generate JWT secret if not exists
echo "=== Creating Application Secrets ==="
JWT_SECRET=$(openssl rand -base64 32)
create_or_update_secret "${PROJECT_NAME}/jwt/secret" "$JWT_SECRET" "JWT signing secret"
echo "✓ Generated JWT secret"
echo ""

# OAuth credentials (will need to be set manually or via environment)
echo "=== OAuth Credentials ==="
echo "⚠️  OAuth secrets need to be created manually or provided via environment variables"
echo ""
echo "For Google OAuth:"
echo "  - Get Client ID from: https://console.cloud.google.com/apis/credentials"
echo "  - Get Client Secret from: https://console.cloud.google.com/apis/credentials"
echo ""
echo "To create OAuth secrets manually, run:"
echo "  aws secretsmanager create-secret --name ${PROJECT_NAME}/google/client_id --secret-string \"YOUR_CLIENT_ID\" --region $AWS_REGION"
echo "  aws secretsmanager create-secret --name ${PROJECT_NAME}/google/client_secret --secret-string \"YOUR_CLIENT_SECRET\" --region $AWS_REGION"
echo ""

# Get secret ARNs
echo "=========================================="
echo "✓ Secrets Created Successfully!"
echo "=========================================="
echo ""
echo "Secret ARNs (for task definitions):"
aws secretsmanager list-secrets \
    --region $AWS_REGION \
    --query "SecretList[?starts_with(Name, \`${PROJECT_NAME}/\`)].{Name:Name,ARN:ARN}" \
    --output table
echo ""
echo "Next steps:"
echo "1. Create OAuth secrets (if needed):"
echo "   - Google OAuth Client ID and Secret"
echo "   - QuickBooks Client ID and Secret (optional)"
echo ""
echo "2. Create 'syncflow' database (if not exists):"
echo "   psql -h $DB_ENDPOINT -U postgres -c 'CREATE DATABASE syncflow;'"
echo "   OR the backend will create it on first run with migrations"
echo ""
echo "3. Push code to trigger deployment:"
echo "   git push origin main"
echo ""

