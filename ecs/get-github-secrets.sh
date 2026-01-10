#!/bin/bash
# Script to get the 3 required GitHub Secret values
# Usage: ./get-github-secrets.sh

set -e

echo "=== Getting GitHub Secret Values ==="
echo ""

# Get AWS Account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION=${AWS_REGION:-us-east-1}

echo "AWS Account ID: $AWS_ACCOUNT_ID"
echo "AWS Region: $AWS_REGION"
echo ""

# 1. ECS_TASK_EXECUTION_ROLE_ARN
echo "1. ECS_TASK_EXECUTION_ROLE_ARN:"
EXECUTION_ROLE_ARN=$(aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text 2>/dev/null || echo "")
if [ -z "$EXECUTION_ROLE_ARN" ]; then
    echo "   ⚠️  Role 'ecsTaskExecutionRole' not found. Create it first:"
    echo "      cd ecs && ./setup-infrastructure.sh"
    EXECUTION_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskExecutionRole"
    echo "   Expected ARN: $EXECUTION_ROLE_ARN"
else
    echo "   ✓ $EXECUTION_ROLE_ARN"
fi
echo ""

# 2. ECS_TASK_ROLE_ARN
echo "2. ECS_TASK_ROLE_ARN:"
TASK_ROLE_ARN=$(aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text 2>/dev/null || echo "")
if [ -z "$TASK_ROLE_ARN" ]; then
    echo "   ⚠️  Role 'ecsTaskRole' not found. Create it first:"
    echo "      cd ecs && ./setup-infrastructure.sh"
    TASK_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskRole"
    echo "   Expected ARN: $TASK_ROLE_ARN"
else
    echo "   ✓ $TASK_ROLE_ARN"
fi
echo ""

# 3. VITE_API_BASE_URL
echo "3. VITE_API_BASE_URL:"
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --query 'LoadBalancers[?contains(LoadBalancerName, `syncflow`) || contains(LoadBalancerName, `backend`)].DNSName' \
    --output text \
    --region $AWS_REGION 2>/dev/null || echo "")

if [ -z "$ALB_DNS" ]; then
    echo "   ⚠️  No ALB found. Options:"
    echo "      a) If you have a custom domain: https://api.yourdomain.com"
    echo "      b) If using ALB: Get ALB DNS from AWS Console after creating ALB"
    echo "      c) Temporary placeholder: https://api.yourdomain.com"
    VITE_API_BASE_URL="https://api.yourdomain.com"
else
    echo "   ✓ Found ALB: $ALB_DNS"
    echo "   Using: http://$ALB_DNS (or configure HTTPS with custom domain)"
    VITE_API_BASE_URL="http://$ALB_DNS"
fi

echo ""
echo "========================================="
echo "Copy these values to GitHub Secrets:"
echo "========================================="
echo ""
echo "ECS_TASK_EXECUTION_ROLE_ARN=$EXECUTION_ROLE_ARN"
echo "ECS_TASK_ROLE_ARN=$TASK_ROLE_ARN"
echo "VITE_API_BASE_URL=$VITE_API_BASE_URL"
echo ""
echo "Steps to add to GitHub:"
echo "1. Go to your GitHub repository"
echo "2. Settings > Secrets and variables > Actions"
echo "3. Click 'New repository secret' for each value above"
echo ""

