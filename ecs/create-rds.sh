#!/bin/bash
# Script to create RDS PostgreSQL database for SyncFlow
# Usage: ./create-rds.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
DB_INSTANCE_ID="${PROJECT_NAME}-db"
DB_SUBNET_GROUP_NAME="${PROJECT_NAME}-db-subnet-group"
DB_SECURITY_GROUP_NAME="${PROJECT_NAME}-rds-sg"
MASTER_USERNAME=${MASTER_USERNAME:-postgres}
MASTER_PASSWORD=${MASTER_PASSWORD:-""}

echo "=========================================="
echo "Creating RDS PostgreSQL Database"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Instance ID: $DB_INSTANCE_ID"
echo ""

# Get VPC ID
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=is-default,Values=true" --region $AWS_REGION --query 'Vpcs[0].VpcId' --output text)
echo "VPC ID: $VPC_ID"
echo ""

# Get subnets for DB subnet group (need at least 2 in different AZs)
SUBNETS=$(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$VPC_ID" --region $AWS_REGION --query 'Subnets[*].SubnetId' --output text)
SUBNET_ARRAY=($SUBNETS)
if [ ${#SUBNET_ARRAY[@]} -lt 2 ]; then
    echo "❌ Error: Need at least 2 subnets for DB subnet group. Found: ${#SUBNET_ARRAY[@]}"
    exit 1
fi

SUBNET_1=${SUBNET_ARRAY[0]}
SUBNET_2=${SUBNET_ARRAY[1]}
echo "Using subnets for DB subnet group: $SUBNET_1, $SUBNET_2"
echo ""

# Create DB subnet group
echo "=== Creating DB Subnet Group ==="
EXISTING_SUBNET_GROUP=$(aws rds describe-db-subnet-groups \
    --db-subnet-group-name "$DB_SUBNET_GROUP_NAME" \
    --region $AWS_REGION \
    --query 'DBSubnetGroups[0].DBSubnetGroupName' \
    --output text 2>/dev/null || echo "")

if [ -z "$EXISTING_SUBNET_GROUP" ] || [ "$EXISTING_SUBNET_GROUP" == "None" ]; then
    echo "Creating DB subnet group: $DB_SUBNET_GROUP_NAME"
    aws rds create-db-subnet-group \
        --db-subnet-group-name "$DB_SUBNET_GROUP_NAME" \
        --db-subnet-group-description "Subnet group for ${PROJECT_NAME} RDS database" \
        --subnet-ids "$SUBNET_1" "$SUBNET_2" \
        --region $AWS_REGION >/dev/null
    
    echo "✓ Created DB subnet group"
else
    echo "✓ DB subnet group already exists: $DB_SUBNET_GROUP_NAME"
fi
echo ""

# Create security group for RDS
echo "=== Creating Security Group for RDS ==="
RDS_SG_ID=$(aws ec2 describe-security-groups \
    --filters "Name=group-name,Values=$DB_SECURITY_GROUP_NAME" "Name=vpc-id,Values=$VPC_ID" \
    --region $AWS_REGION \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null || echo "")

if [ -z "$RDS_SG_ID" ] || [ "$RDS_SG_ID" == "None" ]; then
    echo "Creating security group: $DB_SECURITY_GROUP_NAME"
    RDS_SG_ID=$(aws ec2 create-security-group \
        --group-name "$DB_SECURITY_GROUP_NAME" \
        --description "Security group for ${PROJECT_NAME} RDS database" \
        --vpc-id "$VPC_ID" \
        --region $AWS_REGION \
        --query 'GroupId' \
        --output text)
    
    # Allow PostgreSQL traffic from ECS backend security group
    BACKEND_SG_ID=$(aws ec2 describe-security-groups \
        --filters "Name=group-name,Values=${PROJECT_NAME}-backend-task-sg" "Name=vpc-id,Values=$VPC_ID" \
        --region $AWS_REGION \
        --query 'SecurityGroups[0].GroupId' \
        --output text 2>/dev/null || echo "")
    
    if [ -n "$BACKEND_SG_ID" ] && [ "$BACKEND_SG_ID" != "None" ]; then
        echo "Allowing PostgreSQL traffic from backend ECS tasks..."
        aws ec2 authorize-security-group-ingress \
            --group-id "$RDS_SG_ID" \
            --protocol tcp \
            --port 5432 \
            --source-group "$BACKEND_SG_ID" \
            --region $AWS_REGION 2>/dev/null || echo "Ingress rule may already exist"
    else
        echo "⚠️  Backend security group not found. Allowing from VPC CIDR instead..."
        VPC_CIDR=$(aws ec2 describe-vpcs --vpc-ids "$VPC_ID" --region $AWS_REGION --query 'Vpcs[0].CidrBlock' --output text)
        aws ec2 authorize-security-group-ingress \
            --group-id "$RDS_SG_ID" \
            --protocol tcp \
            --port 5432 \
            --cidr "$VPC_CIDR" \
            --region $AWS_REGION 2>/dev/null || echo "Ingress rule may already exist"
    fi
    
    echo "✓ Created security group: $RDS_SG_ID"
else
    echo "✓ Using existing security group: $RDS_SG_ID"
fi
echo ""

# Check if DB instance already exists
echo "=== Checking Existing Database ==="
EXISTING_DB=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].DBInstanceStatus' \
    --output text 2>/dev/null || echo "")

if [ -n "$EXISTING_DB" ] && [ "$EXISTING_DB" != "None" ]; then
    echo "⚠️  Database instance already exists: $DB_INSTANCE_ID"
    echo "   Status: $EXISTING_DB"
    echo ""
    echo "Getting connection details..."
    DB_ENDPOINT=$(aws rds describe-db-instances \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --region $AWS_REGION \
        --query 'DBInstances[0].Endpoint.Address' \
        --output text)
    DB_PORT=$(aws rds describe-db-instances \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --region $AWS_REGION \
        --query 'DBInstances[0].Endpoint.Port' \
        --output text)
    
    echo ""
    echo "=========================================="
    echo "✓ Database Already Exists!"
    echo "=========================================="
    echo "Endpoint: $DB_ENDPOINT"
    echo "Port: $DB_PORT"
    echo "Username: $MASTER_USERNAME"
    echo ""
    echo "Use this endpoint when creating secrets:"
    echo "  DB_HOST=$DB_ENDPOINT"
    echo "  DB_PORT=$DB_PORT"
    exit 0
fi

# Generate or prompt for master password if not set
if [ -z "$MASTER_PASSWORD" ]; then
    # Check if running in interactive mode (tty)
    if [ -t 0 ]; then
        echo "=== Database Configuration ==="
        echo "Master Username: $MASTER_USERNAME"
        echo ""
        echo "Options:"
        echo "  1. Generate a secure random password (recommended)"
        echo "  2. Enter your own password"
        read -p "Choose option [1]: " OPTION
        OPTION=${OPTION:-1}
        
        if [ "$OPTION" == "1" ]; then
            # Generate a secure random password
            MASTER_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-20)
            echo "✓ Generated secure password"
            echo ""
            echo "⚠️  IMPORTANT: Save this password!"
            echo "   Password: $MASTER_PASSWORD"
            echo ""
            read -p "Press Enter to continue with this password..." DUMMY
        else
            read -s -p "Enter master password (min 8 characters): " MASTER_PASSWORD
            echo
            if [ ${#MASTER_PASSWORD} -lt 8 ]; then
                echo "❌ Error: Password must be at least 8 characters"
                exit 1
            fi
        fi
    else
        # Non-interactive mode: auto-generate password
        MASTER_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-20)
        echo "=== Database Configuration ==="
        echo "Master Username: $MASTER_USERNAME"
        echo "✓ Auto-generated secure password (non-interactive mode)"
        echo ""
        echo "⚠️  IMPORTANT: Save this password - it will be shown below!"
    fi
fi

# Save password to file for later reference
PASSWORD_FILE="/tmp/${PROJECT_NAME}-db-password.txt"
echo "$MASTER_PASSWORD" > "$PASSWORD_FILE"
chmod 600 "$PASSWORD_FILE"
echo "⚠️  IMPORTANT: Database password saved to: $PASSWORD_FILE"
echo "   Password: $MASTER_PASSWORD"
echo "   (You'll need this when creating secrets)"
echo ""

# Create RDS instance
echo ""
echo "=== Creating RDS PostgreSQL Instance ==="
echo "This may take 5-10 minutes..."
echo ""

# Use db.t3.micro for free tier (or db.t3.small for better performance)
# For production, consider larger instance types
# Note: Free tier has restrictions on backup retention and performance insights

# Try creating RDS instance (handle free tier restrictions)
echo "Creating RDS instance..."

# Attempt creation with minimal settings for free tier compatibility
CREATE_OUTPUT=$(aws rds create-db-instance \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --db-instance-class db.t3.micro \
    --engine postgres \
    --engine-version 15.15 \
    --master-username "$MASTER_USERNAME" \
    --master-user-password "$MASTER_PASSWORD" \
    --allocated-storage 20 \
    --storage-type gp3 \
    --vpc-security-group-ids "$RDS_SG_ID" \
    --db-subnet-group-name "$DB_SUBNET_GROUP_NAME" \
    --storage-encrypted \
    --no-multi-az \
    --no-publicly-accessible \
    --region $AWS_REGION 2>&1)

if echo "$CREATE_OUTPUT" | grep -q "FreeTierRestrictionError\|backup.*retention"; then
    echo "⚠️  Free tier restrictions detected. Creating with minimal settings..."
    # Create without backup retention (free tier doesn't support it)
    aws rds create-db-instance \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --db-instance-class db.t3.micro \
        --engine postgres \
        --engine-version 15.15 \
        --master-username "$MASTER_USERNAME" \
        --master-user-password "$MASTER_PASSWORD" \
        --allocated-storage 20 \
        --vpc-security-group-ids "$RDS_SG_ID" \
        --db-subnet-group-name "$DB_SUBNET_GROUP_NAME" \
        --no-multi-az \
        --no-publicly-accessible \
        --region $AWS_REGION >/dev/null
    echo "✓ RDS instance creation initiated (free tier mode)"
elif echo "$CREATE_OUTPUT" | grep -q "already exists\|DBInstanceAlreadyExists"; then
    echo "⚠️  Database instance already exists or is being created"
elif echo "$CREATE_OUTPUT" | grep -q "DBInstance"; then
    echo "✓ RDS instance creation initiated"
else
    echo "❌ Failed to create RDS instance:"
    echo "$CREATE_OUTPUT"
    exit 1
fi

echo "✓ Database creation initiated!"
echo ""
echo "=========================================="
echo "⏳ Database is being created..."
echo "=========================================="
echo ""
echo "This process takes 5-10 minutes. Checking status..."
echo ""

# Wait for database to be available (with timeout)
MAX_WAIT=600  # 10 minutes
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS=$(aws rds describe-db-instances \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --region $AWS_REGION \
        --query 'DBInstances[0].DBInstanceStatus' \
        --output text 2>/dev/null || echo "not-found")
    
    if [ "$STATUS" == "available" ]; then
        echo ""
        echo "=========================================="
        echo "✓ Database is Available!"
        echo "=========================================="
        break
    elif [ "$STATUS" == "creating" ]; then
        echo -n "."
        sleep 10
        ELAPSED=$((ELAPSED + 10))
    else
        echo ""
        echo "⚠️  Unexpected status: $STATUS"
        echo "   Database may still be creating..."
        break
    fi
done

echo ""
echo ""

# Get database endpoint
DB_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "still-creating")

DB_PORT=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "5432")

echo "=========================================="
echo "✓ RDS Database Setup Complete!"
echo "=========================================="
echo ""
echo "Database Details:"
echo "  Instance ID: $DB_INSTANCE_ID"
echo "  Endpoint: $DB_ENDPOINT"
echo "  Port: $DB_PORT"
echo "  Username: $MASTER_USERNAME"
echo "  Engine: PostgreSQL 15.4"
echo "  Instance Class: db.t3.micro"
echo ""
echo "Next Steps:"
echo "1. Wait for database to be fully available (if still creating)"
echo "2. Create secrets using:"
echo "   cd ecs"
echo "   AWS_REGION=$AWS_REGION ./create-secrets.sh"
echo ""
echo "When prompted for database host, use:"
echo "  $DB_ENDPOINT"
echo ""
echo "Or set it as environment variable:"
echo "  export DB_HOST=$DB_ENDPOINT"
echo ""

