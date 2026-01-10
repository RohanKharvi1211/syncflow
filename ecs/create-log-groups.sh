#!/bin/bash
# Script to create CloudWatch log groups for ECS services
# Usage: ./create-log-groups.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "=========================================="
echo "Creating CloudWatch Log Groups"
echo "=========================================="
echo "Region: $AWS_REGION"
echo ""

# Function to create log group if it doesn't exist
create_log_group() {
    local LOG_GROUP_NAME=$1
    local RETENTION_DAYS=${2:-7}  # Default 7 days retention
    
    echo "Checking log group: $LOG_GROUP_NAME"
    
    if aws logs describe-log-groups \
        --log-group-name-prefix "$LOG_GROUP_NAME" \
        --region $AWS_REGION \
        --query "logGroups[?logGroupName=='$LOG_GROUP_NAME'].logGroupName" \
        --output text 2>/dev/null | grep -q "$LOG_GROUP_NAME"; then
        echo "✓ Log group already exists: $LOG_GROUP_NAME"
    else
        echo "Creating log group: $LOG_GROUP_NAME"
        aws logs create-log-group \
            --log-group-name "$LOG_GROUP_NAME" \
            --region $AWS_REGION
        
        echo "Setting retention policy to $RETENTION_DAYS days"
        aws logs put-retention-policy \
            --log-group-name "$LOG_GROUP_NAME" \
            --retention-in-days $RETENTION_DAYS \
            --region $AWS_REGION 2>/dev/null || echo "  (Retention policy already set or not applicable)"
        
        echo "✓ Created: $LOG_GROUP_NAME"
    fi
    echo ""
}

# Create log groups
create_log_group "/ecs/${PROJECT_NAME}-backend" 7
create_log_group "/ecs/${PROJECT_NAME}-frontend" 7

echo "=========================================="
echo "✓ Log Groups Created Successfully!"
echo "=========================================="
echo ""
echo "Created log groups:"
echo "  - /ecs/${PROJECT_NAME}-backend"
echo "  - /ecs/${PROJECT_NAME}-frontend"
echo ""
echo "Next steps:"
echo "  1. Restart ECS service or wait for next deployment"
echo "  2. Check logs: aws logs tail /ecs/${PROJECT_NAME}-backend --follow --region $AWS_REGION"
echo ""

