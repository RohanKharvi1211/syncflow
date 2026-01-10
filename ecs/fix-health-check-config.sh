#!/bin/bash
# Script to update ECS task definition with improved health check settings
# Usage: ./fix-health-check-config.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
TASK_DEFINITION="${PROJECT_NAME}-backend-task"

echo "=========================================="
echo "Updating Health Check Configuration"
echo "=========================================="
echo "Task Definition: $TASK_DEFINITION"
echo "Region: $AWS_REGION"
echo ""

# Get current task definition
echo "Downloading current task definition..."
aws ecs describe-task-definition \
    --task-definition "$TASK_DEFINITION" \
    --region $AWS_REGION \
    --query 'taskDefinition' \
    > /tmp/task-def.json 2>/dev/null || {
    echo "❌ Error: Task definition not found: $TASK_DEFINITION"
    exit 1
}

echo "✓ Task definition downloaded"
echo ""

# Update health check configuration using jq or Python
if command -v jq &> /dev/null; then
    echo "Updating health check using jq..."
    cat /tmp/task-def.json | jq '.containerDefinitions[0].healthCheck = {
        "command": ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 10,
        "retries": 3,
        "startPeriod": 120
    }' > /tmp/task-def-updated.json
elif command -v python3 &> /dev/null; then
    echo "Updating health check using Python..."
    python3 << 'PYTHON_SCRIPT'
import json
import sys

with open('/tmp/task-def.json', 'r') as f:
    task_def = json.load(f)

# Update health check
task_def['containerDefinitions'][0]['healthCheck'] = {
    "command": ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"],
    "interval": 30,
    "timeout": 10,
    "retries": 3,
    "startPeriod": 120
}

# Remove fields that shouldn't be in register-task-definition
remove_fields = ['revision', 'status', 'requiresAttributes', 'compatibilities', 'registeredAt', 'registeredBy']
for field in remove_fields:
    task_def.pop(field, None)

with open('/tmp/task-def-updated.json', 'w') as f:
    json.dump(task_def, f, indent=2)
PYTHON_SCRIPT
else
    echo "❌ Error: Neither jq nor python3 is installed"
    echo "   Please install one of them:"
    echo "     macOS: brew install jq"
    echo "     Ubuntu: sudo apt-get install jq"
    exit 1
fi

echo "✓ Health check configuration updated"
echo ""
echo "New health check settings:"
echo "  - Interval: 30 seconds"
echo "  - Timeout: 10 seconds (increased from 5)"
echo "  - Retries: 3"
echo "  - Start Period: 120 seconds (increased from 60)"
echo ""

# Register new task definition revision
echo "Registering new task definition revision..."
NEW_TASK_DEF_ARN=$(aws ecs register-task-definition \
    --cli-input-json file:///tmp/task-def-updated.json \
    --region $AWS_REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text 2>&1)

if [ $? -eq 0 ] && [ ! -z "$NEW_TASK_DEF_ARN" ]; then
    echo "✓ New task definition registered: $NEW_TASK_DEF_ARN"
    NEW_REVISION=$(echo "$NEW_TASK_DEF_ARN" | awk -F: '{print $NF}')
    echo "  Revision: $NEW_REVISION"
else
    echo "❌ Error: Failed to register task definition"
    cat /tmp/task-def-updated.json | head -50
    exit 1
fi

echo ""

# Update ECS service to use new task definition
SERVICE_NAME="${PROJECT_NAME}-backend-service"
echo "Updating ECS service to use new task definition..."
SERVICE_UPDATE=$(aws ecs update-service \
    --cluster "${PROJECT_NAME}-cluster" \
    --service "$SERVICE_NAME" \
    --task-definition "$NEW_TASK_DEF_ARN" \
    --region $AWS_REGION \
    --query 'service.{ServiceName:serviceName,Status:status,TaskDefinition:taskDefinition,RunningCount:runningCount,DesiredCount:desiredCount}' \
    --output json 2>&1)

if [ $? -eq 0 ]; then
    echo "✓ Service updated successfully"
    echo "$SERVICE_UPDATE" | python3 -m json.tool 2>/dev/null || echo "$SERVICE_UPDATE"
else
    echo "⚠️  Warning: Service update may have failed"
    echo "$SERVICE_UPDATE"
fi

echo ""
echo "=========================================="
echo "✓ Health Check Configuration Updated!"
echo "=========================================="
echo ""
echo "Summary of changes:"
echo "  ✅ Timeout: 5s → 10s (more time for health check to complete)"
echo "  ✅ Start Period: 60s → 120s (more time for app to initialize)"
echo "  ✅ Health check does NOT check database (as per best practices)"
echo ""
echo "The service will now:"
echo "  1. Wait 120 seconds before starting health checks"
echo "  2. Allow 10 seconds for each health check to complete"
echo "  3. Only verify that HTTP server is running (not DB connectivity)"
echo ""
echo "Monitor the deployment:"
echo "  aws ecs describe-services --cluster ${PROJECT_NAME}-cluster --services $SERVICE_NAME --region $AWS_REGION --query 'services[0].events[:5]' --output table"
echo ""

