#!/bin/bash
# Script to set up initial AWS infrastructure for ECS deployment
# Run this once before the first deployment

set -e

AWS_REGION=${AWS_REGION:-us-east-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "Setting up infrastructure for $PROJECT_NAME in region $AWS_REGION"
echo "AWS Account ID: $AWS_ACCOUNT_ID"

# Create ECR repositories
echo "=== Creating ECR Repositories ==="
aws ecr create-repository \
    --repository-name ${PROJECT_NAME}-backend \
    --region $AWS_REGION \
    --image-scanning-configuration scanOnPush=true \
    --encryption-configuration encryptionType=AES256 2>/dev/null || echo "Backend repository already exists"

aws ecr create-repository \
    --repository-name ${PROJECT_NAME}-frontend \
    --region $AWS_REGION \
    --image-scanning-configuration scanOnPush=true \
    --encryption-configuration encryptionType=AES256 2>/dev/null || echo "Frontend repository already exists"

# Create ECS cluster
echo "=== Creating ECS Cluster ==="
aws ecs create-cluster \
    --cluster-name ${PROJECT_NAME}-cluster \
    --region $AWS_REGION \
    --capacity-providers FARGATE FARGATE_SPOT \
    --default-capacity-provider-strategy \
        capacityProvider=FARGATE,weight=1 \
        capacityProvider=FARGATE_SPOT,weight=1 \
    --settings name=containerInsights,value=enabled 2>/dev/null || echo "Cluster already exists"

# Create CloudWatch log groups
echo "=== Creating CloudWatch Log Groups ==="
aws logs create-log-group \
    --log-group-name /ecs/${PROJECT_NAME}-backend \
    --region $AWS_REGION 2>/dev/null || echo "Backend log group already exists"

aws logs create-log-group \
    --log-group-name /ecs/${PROJECT_NAME}-frontend \
    --region $AWS_REGION 2>/dev/null || echo "Frontend log group already exists"

# Create IAM roles (if they don't exist)
echo "=== Creating IAM Roles ==="

# Task Execution Role
cat > /tmp/task-execution-role-trust.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

aws iam create-role \
    --role-name ecsTaskExecutionRole \
    --assume-role-policy-document file:///tmp/task-execution-role-trust.json 2>/dev/null || echo "Execution role already exists"

aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy 2>/dev/null || echo "Policy already attached"

# Task Role
aws iam create-role \
    --role-name ecsTaskRole \
    --assume-role-policy-document file:///tmp/task-execution-role-trust.json 2>/dev/null || echo "Task role already exists"

echo "=== Infrastructure Setup Complete ==="
echo ""
echo "Next steps:"
echo "1. Update ecs/task-definition-*.json files with your IAM role ARNs:"
echo "   - Execution Role: arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskExecutionRole"
echo "   - Task Role: arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskRole"
echo ""
echo "2. Create secrets in AWS Secrets Manager (see ecs/deployment-guide.md)"
echo ""
echo "3. Register task definitions:"
echo "   aws ecs register-task-definition --cli-input-json file://ecs/task-definition-backend.json"
echo "   aws ecs register-task-definition --cli-input-json file://ecs/task-definition-frontend.json"
echo ""
echo "4. Create ECS services (or use Terraform/CloudFormation)"
echo ""
echo "5. Configure GitHub Secrets (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, etc.)"

