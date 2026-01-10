#!/bin/bash
# Script to create the syncflow database in RDS
# Uses AWS RDS Data API (if enabled) or creates a temporary ECS task
# Usage: ./create-database-in-rds.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
DB_INSTANCE_ID="${PROJECT_NAME}-db"

echo "=========================================="
echo "Creating Database in RDS"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Database Instance: $DB_INSTANCE_ID"
echo ""

# Get database connection details
DB_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null || echo "")

DB_PORT=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Port' \
    --output text 2>/dev/null || echo "5432")

if [ -z "$DB_ENDPOINT" ] || [ "$DB_ENDPOINT" == "None" ]; then
    echo "❌ Error: Could not get database endpoint"
    exit 1
fi

echo "Database Endpoint: $DB_ENDPOINT"
echo "Database Port: $DB_PORT"
echo ""

# Get database password from Secrets Manager
DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id "${PROJECT_NAME}/database/password" \
    --region $AWS_REGION \
    --query 'SecretString' \
    --output text 2>/dev/null || echo "")

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Error: Could not get database password from Secrets Manager"
    exit 1
fi

echo "✓ Retrieved database credentials"
echo ""

# Get RDS security group
RDS_SG_ID=$(aws rds describe-db-instances \
    --db-instance-identifier "$DB_INSTANCE_ID" \
    --region $AWS_REGION \
    --query 'DBInstances[0].VpcSecurityGroups[0].VpcSecurityGroupId' \
    --output text 2>/dev/null || echo "")

if [ -z "$RDS_SG_ID" ] || [ "$RDS_SG_ID" == "None" ]; then
    echo "❌ Error: Could not get RDS security group"
    exit 1
fi

echo "RDS Security Group: $RDS_SG_ID"
echo ""

# Create a temporary ECS task to run psql command
echo "=== Creating Database Using Temporary ECS Task ==="
VPC_ID=$(aws ec2 describe-security-groups \
    --group-ids "$RDS_SG_ID" \
    --region $AWS_REGION \
    --query 'SecurityGroups[0].VpcId' \
    --output text)

SUBNETS=($(aws ec2 describe-subnets \
    --filters "Name=vpc-id,Values=$VPC_ID" \
    --region $AWS_REGION \
    --query 'Subnets[*].SubnetId' \
    --output text))

SUBNET_1=${SUBNETS[0]}
SUBNET_2=${SUBNETS[1]}

echo "VPC: $VPC_ID"
echo "Subnets: $SUBNET_1, $SUBNET_2"
echo ""

# Create temporary security group for the task (allows outbound to RDS)
TEMP_SG_NAME="${PROJECT_NAME}-temp-db-creator-sg"
TEMP_SG_ID=$(aws ec2 describe-security-groups \
    --filters "Name=group-name,Values=$TEMP_SG_NAME" "Name=vpc-id,Values=$VPC_ID" \
    --region $AWS_REGION \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null || echo "")

if [ -z "$TEMP_SG_ID" ] || [ "$TEMP_SG_ID" == "None" ]; then
    echo "Creating temporary security group..."
    TEMP_SG_ID=$(aws ec2 create-security-group \
        --group-name "$TEMP_SG_NAME" \
        --description "Temporary SG for database creation task" \
        --vpc-id "$VPC_ID" \
        --region $AWS_REGION \
        --query 'GroupId' \
        --output text)
    
    # Allow outbound to RDS
    aws ec2 authorize-security-group-egress \
        --group-id "$TEMP_SG_ID" \
        --protocol tcp \
        --port 5432 \
        --source-group "$RDS_SG_ID" \
        --region $AWS_REGION 2>/dev/null || echo "Egress rule may already exist"
    
    # Allow all outbound (for pulling image)
    aws ec2 authorize-security-group-egress \
        --group-id "$TEMP_SG_ID" \
        --protocol -1 \
        --cidr 0.0.0.0/0 \
        --region $AWS_REGION 2>/dev/null || echo "Egress rule may already exist"
    
    echo "✓ Created temporary SG: $TEMP_SG_ID"
else
    echo "✓ Using existing temporary SG: $TEMP_SG_ID"
fi
echo ""

# Create temporary task definition
EXEC_ROLE_ARN=$(aws ecs describe-task-definition \
    --task-definition "${PROJECT_NAME}-backend-task" \
    --region $AWS_REGION \
    --query 'taskDefinition.executionRoleArn' \
    --output text 2>/dev/null || echo "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/ecsTaskExecutionRole")

TASK_ROLE_ARN=$(aws ecs describe-task-definition \
    --task-definition "${PROJECT_NAME}-backend-task" \
    --region $AWS_REGION \
    --query 'taskDefinition.taskRoleArn' \
    --output text 2>/dev/null || echo "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/ecsTaskRole")

cat > /tmp/create-db-task.json <<EOF
{
  "family": "${PROJECT_NAME}-temp-create-db",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "$EXEC_ROLE_ARN",
  "taskRoleArn": "$TASK_ROLE_ARN",
  "containerDefinitions": [
    {
      "name": "db-creator",
      "image": "postgres:15-alpine",
      "essential": true,
      "command": [
        "psql",
        "-h", "$DB_ENDPOINT",
        "-p", "$DB_PORT",
        "-U", "postgres",
        "-c", "CREATE DATABASE syncflow;"
      ],
      "environment": [
        {
          "name": "PGPASSWORD",
          "value": "$DB_PASSWORD"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${PROJECT_NAME}-temp",
          "awslogs-region": "$AWS_REGION",
          "awslogs-stream-prefix": "db-creator"
        }
      }
    }
  ]
}
EOF

# Create log group for temp task
aws logs create-log-group \
    --log-group-name "/ecs/${PROJECT_NAME}-temp" \
    --region $AWS_REGION 2>/dev/null || echo "Log group may already exist"

# Register task definition
echo "Registering temporary task definition..."
aws ecs register-task-definition \
    --cli-input-json file:///tmp/create-db-task.json \
    --region $AWS_REGION >/dev/null 2>&1 && echo "✓ Task definition registered" || echo "⚠️  Task definition may already exist"

# Run the task
echo "Running task to create database..."
TASK_ARN=$(aws ecs run-task \
    --cluster "${PROJECT_NAME}-cluster" \
    --task-definition "${PROJECT_NAME}-temp-create-db" \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_1,$SUBNET_2],securityGroups=[$TEMP_SG_ID],assignPublicIp=ENABLED}" \
    --region $AWS_REGION \
    --query 'tasks[0].taskArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$TASK_ARN" ] || [ "$TASK_ARN" == "None" ]; then
    echo "❌ Failed to run task"
    exit 1
fi

echo "✓ Task started: $TASK_ARN"
echo ""

# Wait for task to complete (with timeout)
echo "Waiting for database creation (this may take 30-60 seconds)..."
TIMEOUT=120
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS=$(aws ecs describe-tasks \
        --cluster "${PROJECT_NAME}-cluster" \
        --tasks "$TASK_ARN" \
        --region $AWS_REGION \
        --query 'tasks[0].lastStatus' \
        --output text 2>/dev/null || echo "UNKNOWN")
    
    if [ "$STATUS" == "STOPPED" ]; then
        EXIT_CODE=$(aws ecs describe-tasks \
            --cluster "${PROJECT_NAME}-cluster" \
            --tasks "$TASK_ARN" \
            --region $AWS_REGION \
            --query 'tasks[0].containers[0].exitCode' \
            --output text 2>/dev/null || echo "1")
        
        if [ "$EXIT_CODE" == "0" ]; then
            echo "✅ Database created successfully!"
            break
        else
            echo "⚠️  Task completed with exit code: $EXIT_CODE"
            echo "   Checking logs..."
            aws logs tail "/ecs/${PROJECT_NAME}-temp" \
                --region $AWS_REGION \
                --since 5m \
                --format short 2>&1 | tail -20
            exit 1
        fi
    fi
    
    sleep 5
    ELAPSED=$((ELAPSED + 5))
    echo "  Status: $STATUS (${ELAPSED}s elapsed)"
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "⚠️  Timeout waiting for task. Checking status..."
    aws ecs describe-tasks \
        --cluster "${PROJECT_NAME}-cluster" \
        --tasks "$TASK_ARN" \
        --region $AWS_REGION \
        --query 'tasks[0].{Status:lastStatus,Reason:stoppedReason}' \
        --output table
    exit 1
fi

echo ""
echo "=========================================="
echo "✓ Database Created Successfully!"
echo "=========================================="
echo ""
echo "Database: syncflow"
echo "Endpoint: $DB_ENDPOINT"
echo ""
echo "Next steps:"
echo "  1. Force new deployment of backend service:"
echo "     aws ecs update-service --cluster ${PROJECT_NAME}-cluster --service ${PROJECT_NAME}-backend-service --force-new-deployment --region $AWS_REGION"
echo "  2. Check logs to verify database connection:"
echo "     aws logs tail /ecs/${PROJECT_NAME}-backend --follow --region $AWS_REGION"
echo ""

