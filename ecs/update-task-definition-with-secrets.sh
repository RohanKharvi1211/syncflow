#!/bin/bash
# Script to update ECS task definition with actual secret ARNs from AWS Secrets Manager
# Usage: ./update-task-definition-with-secrets.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "=========================================="
echo "Updating Task Definition with Secrets"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Account ID: $AWS_ACCOUNT_ID"
echo ""

# Get secret ARNs
DB_HOST_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/database/host" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

DB_USER_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/database/user" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

DB_PASSWORD_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/database/password" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

DB_NAME_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/database/name" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

DB_PORT_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/database/port" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

JWT_SECRET_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/jwt/secret" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

GOOGLE_CLIENT_ID_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/google/client_id" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

GOOGLE_CLIENT_SECRET_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/google/client_secret" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

GOOGLE_REDIRECT_URI_ARN=$(aws secretsmanager describe-secret \
    --secret-id "${PROJECT_NAME}/google/redirect_uri" \
    --region $AWS_REGION \
    --query 'ARN' \
    --output text 2>/dev/null || echo "")

# Get ALB DNS for redirect URI if secret doesn't exist
if [ -z "$GOOGLE_REDIRECT_URI_ARN" ] || [ "$GOOGLE_REDIRECT_URI_ARN" == "None" ]; then
    ALB_DNS=$(aws elbv2 describe-load-balancers --names "${PROJECT_NAME}-backend-alb" --region $AWS_REGION --query 'LoadBalancers[0].DNSName' --output text 2>/dev/null || echo "")
    if [ -n "$ALB_DNS" ]; then
        GOOGLE_REDIRECT_URI="http://${ALB_DNS}/api/oauth/google/callback"
    else
        GOOGLE_REDIRECT_URI="http://localhost:8080/api/oauth/google/callback"
    fi
else
    GOOGLE_REDIRECT_URI=""
fi

# Get execution and task role ARNs
EXEC_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskExecutionRole"
TASK_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskRole"

# Get current task definition
echo "Downloading current task definition..."
aws ecs describe-task-definition \
    --task-definition "${PROJECT_NAME}-backend-task" \
    --region $AWS_REGION \
    --query 'taskDefinition' > /tmp/current-task-def.json 2>/dev/null || {
    echo "⚠️  Task definition not found. Creating new one from template..."
    # Use minimal template if doesn't exist
    cat > /tmp/current-task-def.json <<EOF
{
  "family": "${PROJECT_NAME}-backend-task",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "$EXEC_ROLE_ARN",
  "taskRoleArn": "$TASK_ROLE_ARN",
  "containerDefinitions": [{}]
}
EOF
}

# Get latest image from ECR
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
LATEST_IMAGE="${ECR_REGISTRY}/${PROJECT_NAME}-backend:latest"

# Check if image exists
if aws ecr describe-images --repository-name "${PROJECT_NAME}-backend" --image-ids imageTag=latest --region $AWS_REGION >/dev/null 2>&1; then
    echo "✓ Found image: $LATEST_IMAGE"
else
    echo "⚠️  Warning: Image $LATEST_IMAGE not found. Task definition will use placeholder."
    LATEST_IMAGE="nginx:alpine"  # Placeholder
fi

# Build new task definition with secrets
echo "Building new task definition with secrets..."
cat > /tmp/new-task-def.json <<EOF
{
  "family": "${PROJECT_NAME}-backend-task",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "$EXEC_ROLE_ARN",
  "taskRoleArn": "$TASK_ROLE_ARN",
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "$LATEST_IMAGE",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        },
        {
          "name": "ENVIRONMENT",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DB_HOST",
          "valueFrom": "${DB_HOST_ARN}"
        },
        {
          "name": "DB_USER",
          "valueFrom": "${DB_USER_ARN}"
        },
        {
          "name": "DB_PASSWORD",
          "valueFrom": "${DB_PASSWORD_ARN}"
        },
        {
          "name": "DB_NAME",
          "valueFrom": "${DB_NAME_ARN}"
        },
        {
          "name": "DB_PORT",
          "valueFrom": "${DB_PORT_ARN}"
        },
        {
          "name": "JWT_SECRET",
          "valueFrom": "${JWT_SECRET_ARN}"
        }
EOF

# Add Google secrets if they exist
if [ -n "$GOOGLE_CLIENT_ID_ARN" ] && [ "$GOOGLE_CLIENT_ID_ARN" != "None" ]; then
    cat >> /tmp/new-task-def.json <<EOF
        ,
        {
          "name": "GOOGLE_CLIENT_ID",
          "valueFrom": "${GOOGLE_CLIENT_ID_ARN}"
        },
        {
          "name": "GOOGLE_CLIENT_SECRET",
          "valueFrom": "${GOOGLE_CLIENT_SECRET_ARN}"
        }
EOF
fi

# Add Google Redirect URI as environment variable (not secret, but dynamic)
if [ -n "$GOOGLE_REDIRECT_URI" ]; then
    # We need to add it to environment section, not secrets
    # First, close the secrets array
    cat >> /tmp/new-task-def.json <<EOF
      ],
EOF
    # Then modify the environment section to include redirect URI
    # This is complex, so we'll use jq or sed
    # For now, let's add it as a secret if the ARN exists, otherwise as env var
    if [ -n "$GOOGLE_REDIRECT_URI_ARN" ] && [ "$GOOGLE_REDIRECT_URI_ARN" != "None" ]; then
        # Remove the last ], add the secret, then add ],
        sed -i.bak 's/      ],/      ,\
        {\
          "name": "GOOGLE_REDIRECT_URI",\
          "valueFrom": "'"${GOOGLE_REDIRECT_URI_ARN}"'"\
        }\
      ],/' /tmp/new-task-def.json 2>/dev/null || true
    fi
else
    # Close secrets array if no Google secrets
    cat >> /tmp/new-task-def.json <<EOF
      ],
EOF
fi

cat >> /tmp/new-task-def.json <<EOF
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${PROJECT_NAME}-backend",
          "awslogs-region": "$AWS_REGION",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
EOF

# Validate secrets
if [ -z "$DB_HOST_ARN" ] || [ "$DB_HOST_ARN" == "None" ]; then
    echo "❌ Error: DB_HOST secret ARN not found"
    exit 1
fi

echo "Secret ARNs configured:"
echo "  DB_HOST: ${DB_HOST_ARN}"
echo "  DB_USER: ${DB_USER_ARN}"
echo "  DB_PASSWORD: ${DB_PASSWORD_ARN}"
echo "  DB_NAME: ${DB_NAME_ARN}"
echo "  DB_PORT: ${DB_PORT_ARN}"
echo "  JWT_SECRET: ${JWT_SECRET_ARN}"
if [ -n "$GOOGLE_CLIENT_ID_ARN" ] && [ "$GOOGLE_CLIENT_ID_ARN" != "None" ]; then
    echo "  GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID_ARN} ✓"
    echo "  GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET_ARN} ✓"
else
    echo "  GOOGLE_CLIENT_ID: ⚠️  Not found (OAuth will not work)"
    echo "  GOOGLE_CLIENT_SECRET: ⚠️  Not found (OAuth will not work)"
fi
echo ""

# Register new task definition
echo "Registering new task definition revision..."
NEW_TASK_DEF_ARN=$(aws ecs register-task-definition \
    --cli-input-json file:///tmp/new-task-def.json \
    --region $AWS_REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

if [ -z "$NEW_TASK_DEF_ARN" ] || [ "$NEW_TASK_DEF_ARN" == "None" ]; then
    echo "❌ Failed to register task definition"
    exit 1
fi

echo "✓ Registered new task definition: $NEW_TASK_DEF_ARN"
echo ""

# Update service to use new task definition
echo "Updating service to use new task definition..."
aws ecs update-service \
    --cluster "${PROJECT_NAME}-cluster" \
    --service "${PROJECT_NAME}-backend-service" \
    --task-definition "$NEW_TASK_DEF_ARN" \
    --region $AWS_REGION \
    --query 'service.{Status:status,TaskDefinition:taskDefinition}' \
    --output table >/dev/null 2>&1 || {
    echo "⚠️  Service update failed (service may not exist yet)"
    echo "   Task definition is registered and ready for next deployment"
}

echo ""
echo "=========================================="
echo "✓ Task Definition Updated!"
echo "=========================================="
echo ""
echo "New task definition: $NEW_TASK_DEF_ARN"
echo ""
echo "Next steps:"
echo "  1. Wait for service to deploy new task (1-2 minutes)"
echo "  2. Check service status:"
echo "     aws ecs describe-services --cluster ${PROJECT_NAME}-cluster --services ${PROJECT_NAME}-backend-service --region $AWS_REGION --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount}' --output table"
echo "  3. Check logs:"
echo "     aws logs tail /ecs/${PROJECT_NAME}-backend --follow --region $AWS_REGION"
echo ""

