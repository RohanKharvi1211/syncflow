#!/bin/bash
# Script to make RDS public and delete the database
# This will drop the syncflow database and recreate it

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
DB_NAME=${DB_NAME:-syncflow}

echo "=========================================="
echo "Resetting RDS Database"
echo "=========================================="
echo ""

# Get RDS instance identifier
DB_INSTANCE_ID=$(aws rds describe-db-instances \
    --region $AWS_REGION \
    --query "DBInstances[?contains(DBInstanceIdentifier, 'syncflow')].DBInstanceIdentifier" \
    --output text | head -1)

if [ -z "$DB_INSTANCE_ID" ]; then
    echo "❌ Error: Could not find RDS instance"
    exit 1
fi

echo "RDS Instance: $DB_INSTANCE_ID"
echo ""

# Get database credentials from Secrets Manager
DB_HOST=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/host" --region $AWS_REGION --query 'SecretString' --output text)
DB_USER=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/user" --region $AWS_REGION --query 'SecretString' --output text)
DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/password" --region $AWS_REGION --query 'SecretString' --output text)
DB_PORT=$(aws secretsmanager get-secret-value --secret-id "${PROJECT_NAME}/database/port" --region $AWS_REGION --query 'SecretString' --output text)

echo "Step 1: Making RDS publicly accessible..."
aws rds modify-db-instance \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --publicly-accessible \
    --apply-immediately \
    --region $AWS_REGION \
    --output json > /dev/null

echo "✅ RDS modification initiated (publicly accessible)"
echo "   Waiting for modification to complete (this may take 2-5 minutes)..."
echo ""

# Wait for modification to complete
aws rds wait db-instance-available \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION

echo "✅ RDS is now publicly accessible"
echo ""

echo "Step 2: Getting RDS security group and adding current IP..."
# Get RDS security group
RDS_SG=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query "DBInstances[0].VpcSecurityGroups[0].VpcSecurityGroupId" \
    --output text)

if [ -z "$RDS_SG" ] || [ "$RDS_SG" = "None" ]; then
    echo "⚠️  Warning: Could not get RDS security group. Continuing anyway..."
else
    echo "RDS Security Group: $RDS_SG"
    
    # Get current public IP
    CURRENT_IP=$(curl -s https://api.ipify.org || curl -s https://ifconfig.me || echo "")
    if [ -z "$CURRENT_IP" ]; then
        echo "⚠️  Warning: Could not determine current IP. You may need to add it manually to security group $RDS_SG"
    else
        echo "Current IP: $CURRENT_IP"
        echo "Adding IP to security group (port 5432)..."
        
        # Add inbound rule for PostgreSQL (port 5432)
        aws ec2 authorize-security-group-ingress \
            --group-id "$RDS_SG" \
            --protocol tcp \
            --port 5432 \
            --cidr "$CURRENT_IP/32" \
            --region $AWS_REGION 2>/dev/null || echo "  (IP might already be authorized, continuing...)"
        
        echo "✅ IP added to security group"
        echo "   Waiting 5 seconds for changes to propagate..."
        sleep 5
    fi
fi

echo ""
echo "Step 3: Connecting to postgres database and dropping $DB_NAME..."
echo ""

# Connect to postgres database and drop the syncflow database
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d postgres -p "$DB_PORT" <<EOF
-- Terminate all connections to the database
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();

-- Drop the database
DROP DATABASE IF EXISTS "$DB_NAME";

-- Recreate the database
CREATE DATABASE "$DB_NAME";

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE "$DB_NAME" TO "$DB_USER";

\c "$DB_NAME"

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\q
EOF

echo ""
echo "✅ Database '$DB_NAME' has been dropped and recreated"
echo ""

echo "Step 4: Cleaning up security group (removing IP)..."
if [ ! -z "$RDS_SG" ] && [ "$RDS_SG" != "None" ] && [ ! -z "$CURRENT_IP" ]; then
    aws ec2 revoke-security-group-ingress \
        --group-id "$RDS_SG" \
        --protocol tcp \
        --port 5432 \
        --cidr "$CURRENT_IP/32" \
        --region $AWS_REGION 2>/dev/null || echo "  (Could not remove IP, you may need to do it manually)"
    echo "✅ IP removed from security group"
else
    echo "  (Skipping - no IP to remove)"
fi

echo ""
echo "Step 5: Making RDS private again..."
aws rds modify-db-instance \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --no-publicly-accessible \
    --apply-immediately \
    --region $AWS_REGION \
    --output json > /dev/null

echo "✅ RDS modification initiated (private)"
echo "   Waiting for modification to complete (this may take 2-5 minutes)..."
echo ""

# Wait for modification to complete
aws rds wait db-instance-available \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION

echo "✅ RDS is now private again"
echo ""

echo "=========================================="
echo "✓ Database Reset Complete!"
echo "=========================================="
echo ""
echo "The database '$DB_NAME' has been:"
echo "  1. Dropped (all data deleted)"
echo "  2. Recreated (empty database)"
echo "  3. UUID extension enabled"
echo ""
echo "Next steps:"
echo "  1. Deploy the backend application"
echo "  2. Migrations will run automatically on startup"
echo ""

