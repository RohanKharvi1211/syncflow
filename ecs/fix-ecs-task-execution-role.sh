#!/bin/bash
# Script to ensure ECS Task Execution Role has CloudWatch Logs permissions
# Usage: ./fix-ecs-task-execution-role.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "=========================================="
echo "Fixing ECS Task Execution Role Permissions"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Account ID: $AWS_ACCOUNT_ID"
echo ""

# Get the execution role ARN from task definition
EXEC_ROLE_ARN=$(aws ecs describe-task-definition \
    --task-definition syncflow-backend-task \
    --region $AWS_REGION \
    --query 'taskDefinition.executionRoleArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$EXEC_ROLE_ARN" ] || [ "$EXEC_ROLE_ARN" == "None" ]; then
    echo "⚠️  Could not find execution role from task definition"
    echo "   Trying default role name..."
    EXEC_ROLE_NAME="ecsTaskExecutionRole"
    EXEC_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${EXEC_ROLE_NAME}"
else
    EXEC_ROLE_NAME=$(echo $EXEC_ROLE_ARN | cut -d'/' -f2)
fi

echo "Execution Role ARN: $EXEC_ROLE_ARN"
echo "Execution Role Name: $EXEC_ROLE_NAME"
echo ""

# Check if role exists
if ! aws iam get-role --role-name "$EXEC_ROLE_NAME" --region $AWS_REGION >/dev/null 2>&1; then
    echo "❌ Error: Role $EXEC_ROLE_NAME does not exist"
    echo "   Please create it first or check your task definition"
    exit 1
fi

echo "✓ Role exists: $EXEC_ROLE_NAME"
echo ""

# Create policy document for CloudWatch Logs
cat > /tmp/cloudwatch-logs-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:log-group:/ecs/syncflow-*:*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup"
      ],
      "Resource": "arn:aws:logs:${AWS_REGION}:${AWS_ACCOUNT_ID}:log-group:/ecs/syncflow-*"
    }
  ]
}
EOF

POLICY_NAME="ECSTaskExecutionCloudWatchLogsPolicy"
POLICY_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${POLICY_NAME}"

# Check if policy exists
if aws iam get-policy --policy-arn "$POLICY_ARN" --region $AWS_REGION >/dev/null 2>&1; then
    echo "Policy already exists: $POLICY_NAME"
    echo "Creating new version..."
    aws iam create-policy-version \
        --policy-arn "$POLICY_ARN" \
        --policy-document file:///tmp/cloudwatch-logs-policy.json \
        --set-as-default \
        --region $AWS_REGION >/dev/null 2>&1 || echo "  (Using existing version)"
else
    echo "Creating policy: $POLICY_NAME"
    aws iam create-policy \
        --policy-name "$POLICY_NAME" \
        --policy-document file:///tmp/cloudwatch-logs-policy.json \
        --region $AWS_REGION >/dev/null 2>&1 || echo "  (Policy may already exist with different name)"
fi

echo "✓ Policy ready: $POLICY_NAME"
echo ""

# Attach policy to role
echo "Attaching policy to role..."
aws iam attach-role-policy \
    --role-name "$EXEC_ROLE_NAME" \
    --policy-arn "$POLICY_ARN" \
    --region $AWS_REGION 2>/dev/null && echo "✓ Attached policy to role" || echo "⚠️  Policy may already be attached (or error occurred)"

echo ""

# Also ensure the standard ECS task execution role policy is attached
echo "Ensuring standard ECS Task Execution Role policy is attached..."
aws iam attach-role-policy \
    --role-name "$EXEC_ROLE_NAME" \
    --policy-arn "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy" \
    --region $AWS_REGION 2>/dev/null && echo "✓ Attached standard ECS execution role policy" || echo "⚠️  Standard policy may already be attached"

echo ""
echo "=========================================="
echo "✓ Permissions Updated!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "  1. Wait 30 seconds for IAM changes to propagate"
echo "  2. Force new deployment:"
echo "     aws ecs update-service --cluster syncflow-cluster --service syncflow-backend-service --force-new-deployment --region $AWS_REGION"
echo "  3. Monitor task status:"
echo "     aws ecs describe-services --cluster syncflow-cluster --services syncflow-backend-service --region $AWS_REGION --query 'services[0].events[:3]' --output table"
echo ""

