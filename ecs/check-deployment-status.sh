#!/bin/bash
# Script to check ECS deployment status
# Usage: ./check-deployment-status.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
CLUSTER_NAME="${PROJECT_NAME}-cluster"
BACKEND_SERVICE="${PROJECT_NAME}-backend-service"
FRONTEND_SERVICE="${PROJECT_NAME}-frontend-service"

echo "=========================================="
echo "Checking ECS Deployment Status"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Cluster: $CLUSTER_NAME"
echo ""

# 1. Check ECS Cluster Status
echo "=== 1. ECS Cluster Status ==="
aws ecs describe-clusters \
    --clusters $CLUSTER_NAME \
    --region $AWS_REGION \
    --query 'clusters[0].{Name:clusterName,Status:status,ActiveServices:activeServicesCount,RunningTasks:runningTasksCount,PendingTasks:pendingTasksCount}' \
    --output table
echo ""

# 2. Check Backend Service Status
echo "=== 2. Backend Service Status ==="
if aws ecs describe-services \
    --cluster $CLUSTER_NAME \
    --services $BACKEND_SERVICE \
    --region $AWS_REGION \
    --query 'services[0]' 2>/dev/null | grep -q "serviceName"; then
    aws ecs describe-services \
        --cluster $CLUSTER_NAME \
        --services $BACKEND_SERVICE \
        --region $AWS_REGION \
        --query 'services[0].{Name:serviceName,Status:status,RunningCount:runningCount,DesiredCount:desiredCount,PendingCount:pendingCount,TaskDefinition:taskDefinition}' \
        --output table
else
    echo "⚠️  Backend service not found. It may not be created yet."
fi
echo ""

# 3. Check Frontend Service Status
echo "=== 3. Frontend Service Status ==="
if aws ecs describe-services \
    --cluster $CLUSTER_NAME \
    --services $FRONTEND_SERVICE \
    --region $AWS_REGION \
    --query 'services[0]' 2>/dev/null | grep -q "serviceName"; then
    aws ecs describe-services \
        --cluster $CLUSTER_NAME \
        --services $FRONTEND_SERVICE \
        --region $AWS_REGION \
        --query 'services[0].{Name:serviceName,Status:status,RunningCount:runningCount,DesiredCount:desiredCount,PendingCount:pendingCount,TaskDefinition:taskDefinition}' \
        --output table
else
    echo "⚠️  Frontend service not found. It may not be created yet."
fi
echo ""

# 4. Check Running Tasks
echo "=== 4. Running Tasks ==="
TASKS=$(aws ecs list-tasks \
    --cluster $CLUSTER_NAME \
    --region $AWS_REGION \
    --query 'taskArns[]' \
    --output text)

if [ -z "$TASKS" ] || [ "$TASKS" == "None" ]; then
    echo "⚠️  No tasks found. Services may still be starting up."
else
    echo "Found $(echo $TASKS | wc -w | tr -d ' ') task(s)"
    aws ecs describe-tasks \
        --cluster $CLUSTER_NAME \
        --tasks $TASKS \
        --region $AWS_REGION \
        --query 'tasks[*].{TaskArn:taskArn,LastStatus:lastStatus,DesiredStatus:desiredStatus,HealthStatus:healthStatus,StartedAt:startedAt}' \
        --output table
fi
echo ""

# 5. Check Container Health
echo "=== 5. Container Health Status ==="
if [ ! -z "$TASKS" ] && [ "$TASKS" != "None" ]; then
    aws ecs describe-tasks \
        --cluster $CLUSTER_NAME \
        --tasks $TASKS \
        --region $AWS_REGION \
        --query 'tasks[*].containers[*].{Name:name,Status:lastStatus,Health:healthStatus,ExitCode:exitCode}' \
        --output table
else
    echo "⚠️  No tasks available to check container health."
fi
echo ""

# 6. Check ALB Target Health
echo "=== 6. ALB Target Health ==="
ALB_ARN=$(aws elbv2 describe-load-balancers \
    --region $AWS_REGION \
    --query "LoadBalancers[?contains(LoadBalancerName, '${PROJECT_NAME}')].LoadBalancerArn" \
    --output text | head -1)

if [ ! -z "$ALB_ARN" ] && [ "$ALB_ARN" != "None" ]; then
    TARGET_GROUPS=$(aws elbv2 describe-target-groups \
        --load-balancer-arn $ALB_ARN \
        --region $AWS_REGION \
        --query 'TargetGroups[*].TargetGroupArn' \
        --output text)
    
    if [ ! -z "$TARGET_GROUPS" ] && [ "$TARGET_GROUPS" != "None" ]; then
        for TG_ARN in $TARGET_GROUPS; do
            TG_NAME=$(aws elbv2 describe-target-groups \
                --target-group-arns $TG_ARN \
                --region $AWS_REGION \
                --query 'TargetGroups[0].TargetGroupName' \
                --output text)
            echo "Target Group: $TG_NAME"
            aws elbv2 describe-target-health \
                --target-group-arn $TG_ARN \
                --region $AWS_REGION \
                --query 'TargetHealthDescriptions[*].{Target:Target.Id,Port:Target.Port,Health:TargetHealth.State,Reason:TargetHealth.Reason}' \
                --output table
            echo ""
        done
    else
        echo "⚠️  No target groups found for ALB."
    fi
else
    echo "⚠️  ALB not found. It may not be created yet."
fi

# 7. Check Recent CloudWatch Logs
echo "=== 7. Recent Logs (Last 10 lines) ==="
LOG_GROUP_BACKEND="/ecs/${PROJECT_NAME}-backend"
LOG_GROUP_FRONTEND="/ecs/${PROJECT_NAME}-frontend"

if aws logs describe-log-groups \
    --log-group-name-prefix "/ecs/${PROJECT_NAME}" \
    --region $AWS_REGION \
    --query 'logGroups[*].logGroupName' \
    --output text 2>/dev/null | grep -q "$LOG_GROUP_BACKEND"; then
    echo "Backend Logs (last 10 lines):"
    aws logs tail $LOG_GROUP_BACKEND \
        --region $AWS_REGION \
        --since 5m \
        --format short 2>/dev/null | tail -10 || echo "No recent logs"
else
    echo "⚠️  Backend log group not found: $LOG_GROUP_BACKEND"
fi
echo ""

if aws logs describe-log-groups \
    --log-group-name-prefix "/ecs/${PROJECT_NAME}" \
    --region $AWS_REGION \
    --query 'logGroups[*].logGroupName' \
    --output text 2>/dev/null | grep -q "$LOG_GROUP_FRONTEND"; then
    echo "Frontend Logs (last 10 lines):"
    aws logs tail $LOG_GROUP_FRONTEND \
        --region $AWS_REGION \
        --since 5m \
        --format short 2>/dev/null | tail -10 || echo "No recent logs"
else
    echo "⚠️  Frontend log group not found: $LOG_GROUP_FRONTEND"
fi
echo ""

# 8. Check Service Events (Recent)
echo "=== 8. Recent Service Events ==="
if aws ecs describe-services \
    --cluster $CLUSTER_NAME \
    --services $BACKEND_SERVICE \
    --region $AWS_REGION \
    --query 'services[0].events[:5]' 2>/dev/null | grep -q "message"; then
    echo "Backend Service Events (last 5):"
    aws ecs describe-services \
        --cluster $CLUSTER_NAME \
        --services $BACKEND_SERVICE \
        --region $AWS_REGION \
        --query 'services[0].events[:5].{Message:message,Time:createdAt}' \
        --output table
fi
echo ""

echo "=========================================="
echo "✓ Deployment Status Check Complete"
echo "=========================================="
echo ""
echo "Quick Health Check URLs:"
echo "  Backend Health: http://$(aws elbv2 describe-load-balancers --region $AWS_REGION --query "LoadBalancers[?contains(LoadBalancerName, '${PROJECT_NAME}')].DNSName" --output text | head -1)/health"
echo ""
echo "To watch logs in real-time:"
echo "  Backend:  aws logs tail /ecs/${PROJECT_NAME}-backend --follow --region $AWS_REGION"
echo "  Frontend: aws logs tail /ecs/${PROJECT_NAME}-frontend --follow --region $AWS_REGION"
echo ""

