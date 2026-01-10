#!/bin/bash
# Script to list all secret ARNs for updating task definitions
# Usage: ./get-secret-arns.sh

AWS_REGION=${AWS_REGION:-us-east-1}

echo "Fetching secret ARNs from AWS Secrets Manager..."
echo ""

aws secretsmanager list-secrets \
    --region $AWS_REGION \
    --query 'SecretList[?starts_with(Name, `syncflow`)].{Name:Name,ARN:ARN}' \
    --output table

echo ""
echo "AWS Account ID: $(aws sts get-caller-identity --query Account --output text)"
echo "AWS Region: $AWS_REGION"
echo ""
echo "Copy the ARNs above and update your task definition files:"
echo "  - ecs/task-definition-backend.json"
echo "  - ecs/task-definition-frontend.json"

