#!/bin/bash
# Script to add Secrets Manager permissions to ECS Task Execution Role
# Usage: ./fix-secrets-permissions.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "=========================================="
echo "Fixing Secrets Manager Permissions"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Account ID: $AWS_ACCOUNT_ID"
echo ""

EXEC_ROLE_NAME="ecsTaskExecutionRole"
EXEC_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/${EXEC_ROLE_NAME}"

# Check if role exists
if ! aws iam get-role --role-name "$EXEC_ROLE_NAME" --region $AWS_REGION >/dev/null 2>&1; then
    echo "❌ Error: Role $EXEC_ROLE_NAME does not exist"
    exit 1
fi

echo "✓ Role exists: $EXEC_ROLE_NAME"
echo ""

# Get all secret ARNs for the policy
echo "Fetching secret ARNs..."
SECRET_ARNS=$(aws secretsmanager list-secrets \
    --region $AWS_REGION \
    --query "SecretList[?starts_with(Name, '${PROJECT_NAME}/')].ARN" \
    --output text)

if [ -z "$SECRET_ARNS" ] || [ "$SECRET_ARNS" == "None" ]; then
    echo "❌ Error: No secrets found with prefix ${PROJECT_NAME}/"
    exit 1
fi

echo "Found secrets:"
echo "$SECRET_ARNS" | tr '\t' '\n' | while read ARN; do
    echo "  - $ARN"
done
echo ""

# Create policy document
echo "Creating IAM policy for Secrets Manager access..."
cat > /tmp/secrets-manager-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": [
EOF

# Add each secret ARN to the policy
echo "$SECRET_ARNS" | tr '\t' '\n' | while read ARN; do
    if [ ! -z "$ARN" ] && [ "$ARN" != "None" ]; then
        echo "        \"$ARN\"," >> /tmp/secrets-manager-policy.json
    fi
done

# Remove trailing comma and close array
sed -i '' '$ s/,$//' /tmp/secrets-manager-policy.json 2>/dev/null || sed -i '$ s/,$//' /tmp/secrets-manager-policy.json

cat >> /tmp/secrets-manager-policy.json <<EOF
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "kms:ViaService": "secretsmanager.${AWS_REGION}.amazonaws.com"
        }
      }
    }
  ]
}
EOF

POLICY_NAME="ECSTaskExecutionSecretsManagerPolicy"
POLICY_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${POLICY_NAME}"

# Check if policy exists
if aws iam get-policy --policy-arn "$POLICY_ARN" --region $AWS_REGION >/dev/null 2>&1; then
    echo "Policy already exists: $POLICY_NAME"
    echo "Creating new version..."
    
    # Delete old versions if at limit (5 versions max)
    VERSIONS=$(aws iam list-policy-versions --policy-arn "$POLICY_ARN" --region $AWS_REGION --query 'Versions[?IsDefaultVersion==`false`].VersionId' --output text)
    if [ ! -z "$VERSIONS" ]; then
        for VERSION in $VERSIONS; do
            aws iam delete-policy-version --policy-arn "$POLICY_ARN" --version-id "$VERSION" --region $AWS_REGION 2>/dev/null || true
        done
    fi
    
    # Create new version
    aws iam create-policy-version \
        --policy-arn "$POLICY_ARN" \
        --policy-document file:///tmp/secrets-manager-policy.json \
        --set-as-default \
        --region $AWS_REGION >/dev/null 2>&1 && echo "✓ Created new policy version" || echo "⚠️  Failed to create version (may need to delete old versions first)"
else
    echo "Creating new policy: $POLICY_NAME"
    aws iam create-policy \
        --policy-name "$POLICY_NAME" \
        --policy-document file:///tmp/secrets-manager-policy.json \
        --description "Allows ECS task execution role to read secrets from Secrets Manager" \
        --region $AWS_REGION >/dev/null 2>&1 && echo "✓ Created policy" || {
        echo "❌ Failed to create policy"
        cat /tmp/secrets-manager-policy.json
        exit 1
    }
fi

echo "✓ Policy ready: $POLICY_ARN"
echo ""

# Attach policy to role
echo "Attaching policy to role..."
aws iam attach-role-policy \
    --role-name "$EXEC_ROLE_NAME" \
    --policy-arn "$POLICY_ARN" \
    --region $AWS_REGION 2>/dev/null && echo "✓ Attached policy to role" || echo "⚠️  Policy may already be attached (or error occurred)"

echo ""

# Verify attachment
echo "Verifying policy attachment..."
ATTACHED=$(aws iam list-attached-role-policies \
    --role-name "$EXEC_ROLE_NAME" \
    --region $AWS_REGION \
    --query "AttachedPolicies[?PolicyArn=='$POLICY_ARN'].PolicyArn" \
    --output text)

if [ ! -z "$ATTACHED" ] && [ "$ATTACHED" != "None" ]; then
    echo "✓ Policy is attached to role"
else
    echo "⚠️  Warning: Policy attachment verification failed"
fi

echo ""
echo "=========================================="
echo "✓ Permissions Updated!"
echo "=========================================="
echo ""
echo "Policy ARN: $POLICY_ARN"
echo "Attached to: $EXEC_ROLE_NAME"
echo ""
echo "IMPORTANT: IAM changes can take 30-60 seconds to propagate"
echo ""
echo "Next steps:"
echo "  1. Wait 30-60 seconds for IAM changes to propagate"
echo "  2. Force new deployment:"
echo "     aws ecs update-service --cluster ${PROJECT_NAME}-cluster --service ${PROJECT_NAME}-backend-service --force-new-deployment --region $AWS_REGION"
echo "  3. Monitor deployment:"
echo "     aws ecs describe-services --cluster ${PROJECT_NAME}-cluster --services ${PROJECT_NAME}-backend-service --region $AWS_REGION --query 'services[0].events[:5]' --output table"
echo "  4. Check logs:"
echo "     aws logs tail /ecs/${PROJECT_NAME}-backend --follow --region $AWS_REGION"
echo ""

