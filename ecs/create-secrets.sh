#!/bin/bash
# Script to create all required AWS Secrets Manager secrets
# Usage: ./create-secrets.sh

set -e

AWS_REGION=${AWS_REGION:-us-east-1}

echo "=========================================="
echo "AWS Secrets Manager Setup for SyncFlow"
echo "=========================================="
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
        --region $AWS_REGION >/dev/null
    echo "✓ $secret_name"
}

# Database configuration
echo "=== Database Configuration ==="
read -p "Enter RDS host endpoint: " DB_HOST
read -p "Enter database user [postgres]: " DB_USER
DB_USER=${DB_USER:-postgres}
read -s -p "Enter database password: " DB_PASSWORD
echo
read -p "Enter database name [syncflow]: " DB_NAME
DB_NAME=${DB_NAME:-syncflow}
read -p "Enter database port [5432]: " DB_PORT
DB_PORT=${DB_PORT:-5432}

create_or_update_secret "syncflow/database/host" "$DB_HOST" "RDS PostgreSQL host endpoint"
create_or_update_secret "syncflow/database/user" "$DB_USER" "Database username"
create_or_update_secret "syncflow/database/password" "$DB_PASSWORD" "Database password"
create_or_update_secret "syncflow/database/name" "$DB_NAME" "Database name"
create_or_update_secret "syncflow/database/port" "$DB_PORT" "Database port"

echo ""
echo "=== Application Secrets ==="
read -p "Generate new JWT secret? (y/n) [y]: " GENERATE_JWT
GENERATE_JWT=${GENERATE_JWT:-y}

if [ "$GENERATE_JWT" = "y" ]; then
    JWT_SECRET=$(openssl rand -base64 32)
    echo "Generated JWT secret (saved to Secrets Manager)"
else
    read -s -p "Enter JWT secret: " JWT_SECRET
    echo
fi

create_or_update_secret "syncflow/jwt/secret" "$JWT_SECRET" "JWT signing secret"

echo ""
echo "=== Google OAuth Configuration ==="
read -p "Enter Google OAuth Client ID: " GOOGLE_CLIENT_ID
read -s -p "Enter Google OAuth Client Secret: " GOOGLE_CLIENT_SECRET
echo

create_or_update_secret "syncflow/google/client_id" "$GOOGLE_CLIENT_ID" "Google OAuth Client ID"
create_or_update_secret "syncflow/google/client_secret" "$GOOGLE_CLIENT_SECRET" "Google OAuth Client Secret"

echo ""
echo "=== QuickBooks OAuth Configuration (Optional) ==="
read -p "Configure QuickBooks OAuth? (y/n) [n]: " CONFIGURE_QB
CONFIGURE_QB=${CONFIGURE_QB:-n}

if [ "$CONFIGURE_QB" = "y" ]; then
    read -p "Enter QuickBooks Client ID: " QB_CLIENT_ID
    read -s -p "Enter QuickBooks Client Secret: " QB_CLIENT_SECRET
    echo
    
    create_or_update_secret "syncflow/quickbooks/client_id" "$QB_CLIENT_ID" "QuickBooks OAuth Client ID"
    create_or_update_secret "syncflow/quickbooks/client_secret" "$QB_CLIENT_SECRET" "QuickBooks OAuth Client Secret"
fi

echo ""
echo "=========================================="
echo "✓ All secrets created successfully!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Get secret ARNs:"
echo "   aws secretsmanager list-secrets --region $AWS_REGION --query 'SecretList[?starts_with(Name, \`syncflow\`)].{Name:Name,ARN:ARN}' --output table"
echo ""
echo "2. Update task definitions (ecs/task-definition-*.json) with secret ARNs"
echo ""
echo "3. Add GitHub Secrets:"
echo "   - AWS_ACCESS_KEY_ID"
echo "   - AWS_SECRET_ACCESS_KEY"
echo "   - ECS_TASK_EXECUTION_ROLE_ARN"
echo "   - ECS_TASK_ROLE_ARN"
echo "   - VITE_API_BASE_URL"
echo ""
echo "4. Update OAuth redirect URIs in Google/Intuit dashboards"

