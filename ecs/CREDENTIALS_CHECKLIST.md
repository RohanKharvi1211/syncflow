# Credentials Checklist for ECS Deployment

This document lists all the credentials and secrets you need to set up before deploying to AWS ECS.

## Prerequisites

Before starting, ensure you have:
- **AWS CLI installed and configured** - See `AWS_CLI_SETUP.md` for installation and setup instructions
- **AWS Account** - If you don't have one, sign up at https://aws.amazon.com/
- **GitHub Account** - For CI/CD pipeline

## 1. AWS Credentials (for GitHub Actions)

These are needed for GitHub Actions to deploy to AWS. Add them in **GitHub Repository > Settings > Secrets and variables > Actions**.

### Required GitHub Secrets:

| Secret Name | Description | Example |
|------------|-------------|---------|
| `AWS_ACCESS_KEY_ID` | AWS IAM user access key ID | `AKIAIOSFODNN7EXAMPLE` |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM user secret access key | `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY` |
| `ECS_TASK_EXECUTION_ROLE_ARN` | ARN of ECS task execution role | `arn:aws:iam::123456789012:role/ecsTaskExecutionRole` |
| `ECS_TASK_ROLE_ARN` | ARN of ECS task role | `arn:aws:iam::123456789012:role/ecsTaskRole` |
| `VITE_API_BASE_URL` | Frontend API base URL (for build-time) | `https://api.yourdomain.com` |

### How to Create AWS IAM User:

```bash
# 1. Create IAM user for CI/CD
aws iam create-user --user-name syncflow-cicd

# 2. Create access key
aws iam create-access-key --user-name syncflow-cicd

# 3. Attach necessary policies
aws iam attach-user-policy \
    --user-name syncflow-cicd \
    --policy-arn arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser

aws iam attach-user-policy \
    --user-name syncflow-cicd \
    --policy-arn arn:aws:iam::aws:policy/AmazonECS_FullAccess

# 4. Allow ECS service updates
aws iam put-user-policy \
    --user-name syncflow-cicd \
    --policy-name ECSServiceUpdate \
    --policy-document '{
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Action": [
                "ecs:UpdateService",
                "ecs:DescribeServices",
                "ecs:RegisterTaskDefinition",
                "ecs:DescribeTaskDefinition"
            ],
            "Resource": "*"
        }]
    }'
```

## 2. AWS Secrets Manager Secrets

Store these in **AWS Secrets Manager** and reference them in the ECS task definitions. The backend container will pull these at runtime.

### Database Secrets:

| Secret Name | Description | How to Create |
|------------|-------------|---------------|
| `syncflow/database/host` | RDS PostgreSQL endpoint | `your-db-instance.xxxxx.us-east-1.rds.amazonaws.com` |
| `syncflow/database/user` | Database username | `postgres` (or your username) |
| `syncflow/database/password` | Database password | Your RDS password |
| `syncflow/database/name` | Database name | `syncflow` |
| `syncflow/database/port` | Database port | `5432` |

### Application Secrets:

| Secret Name | Description | How to Create |
|------------|-------------|---------------|
| `syncflow/jwt/secret` | JWT signing secret (use strong random string) | Generate with: `openssl rand -base64 32` |

### OAuth Credentials:

| Secret Name | Description | Where to Get |
|------------|-------------|--------------|
| `syncflow/google/client_id` | Google OAuth Client ID | [Google Cloud Console](https://console.cloud.google.com/apis/credentials) |
| `syncflow/google/client_secret` | Google OAuth Client Secret | [Google Cloud Console](https://console.cloud.google.com/apis/credentials) |

### QuickBooks Credentials (Optional):

| Secret Name | Description | Where to Get |
|------------|-------------|--------------|
| `syncflow/quickbooks/client_id` | QuickBooks Client ID | [Intuit Developer Dashboard](https://developer.intuit.com/app/developer/myapps) |
| `syncflow/quickbooks/client_secret` | QuickBooks Client Secret | [Intuit Developer Dashboard](https://developer.intuit.com/app/developer/myapps) |

### How to Create Secrets in AWS Secrets Manager:

```bash
AWS_REGION=us-east-1

# Database secrets
aws secretsmanager create-secret \
    --name syncflow/database/host \
    --secret-string "your-rds-endpoint.rds.amazonaws.com" \
    --region $AWS_REGION \
    --description "RDS PostgreSQL host endpoint"

aws secretsmanager create-secret \
    --name syncflow/database/user \
    --secret-string "postgres" \
    --region $AWS_REGION \
    --description "Database username"

aws secretsmanager create-secret \
    --name syncflow/database/password \
    --secret-string "YOUR_SECURE_PASSWORD" \
    --region $AWS_REGION \
    --description "Database password"

aws secretsmanager create-secret \
    --name syncflow/database/name \
    --secret-string "syncflow" \
    --region $AWS_REGION \
    --description "Database name"

aws secretsmanager create-secret \
    --name syncflow/database/port \
    --secret-string "5432" \
    --region $AWS_REGION \
    --description "Database port"

# JWT Secret (generate a strong random string)
JWT_SECRET=$(openssl rand -base64 32)
aws secretsmanager create-secret \
    --name syncflow/jwt/secret \
    --secret-string "$JWT_SECRET" \
    --region $AWS_REGION \
    --description "JWT signing secret"

# Google OAuth
aws secretsmanager create-secret \
    --name syncflow/google/client_id \
    --secret-string "YOUR_GOOGLE_CLIENT_ID" \
    --region $AWS_REGION \
    --description "Google OAuth Client ID"

aws secretsmanager create-secret \
    --name syncflow/google/client_secret \
    --secret-string "YOUR_GOOGLE_CLIENT_SECRET" \
    --region $AWS_REGION \
    --description "Google OAuth Client Secret"
```

## 3. Environment Variables (Task Definition)

These are set directly in the ECS task definition (not secrets, but configuration):

### Backend Environment Variables:

| Variable | Value | Notes |
|---------|-------|-------|
| `PORT` | `8080` | Port the backend listens on |
| `ENVIRONMENT` | `production` | Environment name |
| `APP_NAME` | `syncflow-backend` | Application name |
| `GOOGLE_REDIRECT_URI` | `https://api.yourdomain.com/api/oauth/google/callback` | Your domain |
| `QUICKBOOKS_REDIRECT_URI` | `https://api.yourdomain.com/api/oauth/quickbooks/callback` | Your domain |
| `FRONTEND_URL` | `https://yourdomain.com` | Frontend domain |

### Frontend Environment Variables:

| Variable | Value | Notes |
|---------|-------|-------|
| `VITE_API_BASE_URL` | `https://api.yourdomain.com` | **Build-time only** - Must be set during Docker build |

**Note:** `VITE_API_BASE_URL` must be provided as a build argument when building the frontend Docker image. It's used during the build process, not at runtime.

## 4. Domain and DNS Configuration

You'll need:

- **Domain name**: For example, `yourdomain.com`
- **SSL Certificate**: For HTTPS (can use AWS Certificate Manager - ACM)
- **DNS Records**: Point your domain to the Application Load Balancer

### SSL Certificate (AWS Certificate Manager):

```bash
# Request SSL certificate
aws acm request-certificate \
    --domain-name yourdomain.com \
    --subject-alternative-names api.yourdomain.com \
    --validation-method DNS \
    --region us-east-1

# Validate the certificate (follow DNS validation steps in ACM console)
# Then attach to your ALB listeners
```

## 5. IAM Role ARNs

After creating IAM roles (via `setup-infrastructure.sh` or manually), get their ARNs:

```bash
# Get execution role ARN
aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text

# Get task role ARN
aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text
```

Add these ARNs to:
- GitHub Secrets: `ECS_TASK_EXECUTION_ROLE_ARN` and `ECS_TASK_ROLE_ARN`
- Task definition files: Update `executionRoleArn` and `taskRoleArn` fields

## 6. ECR Repository URLs

After creating ECR repositories, get their URLs:

```bash
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION=us-east-1

echo "Backend ECR: ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/syncflow-backend"
echo "Frontend ECR: ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/syncflow-frontend"
```

Update these in:
- Task definition files: `image` field in container definitions
- GitHub Actions workflow: `ECR_REPOSITORY_BACKEND` and `ECR_REPOSITORY_FRONTEND` env vars

## 7. OAuth Redirect URIs

Update these in your OAuth provider dashboards:

### Google Cloud Console:
1. Go to [Google Cloud Console > APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials)
2. Edit your OAuth 2.0 Client ID
3. Add authorized redirect URI: `https://api.yourdomain.com/api/oauth/google/callback`

### Intuit Developer Dashboard:
1. Go to [Intuit Developer Dashboard > My Apps](https://developer.intuit.com/app/developer/myapps)
2. Select your app
3. Add redirect URI: `https://api.yourdomain.com/api/oauth/quickbooks/callback`

## Quick Setup Script

Here's a script to create all required secrets:

```bash
#!/bin/bash
# create-secrets.sh - Script to create all AWS Secrets Manager secrets

set -e

AWS_REGION=${AWS_REGION:-us-east-1}

echo "Creating AWS Secrets Manager secrets..."

# Database secrets (update with your RDS values)
read -p "Enter RDS host endpoint: " DB_HOST
read -s -p "Enter database password: " DB_PASSWORD
echo
read -p "Enter database user [postgres]: " DB_USER
DB_USER=${DB_USER:-postgres}
read -p "Enter database name [syncflow]: " DB_NAME
DB_NAME=${DB_NAME:-syncflow}
read -p "Enter database port [5432]: " DB_PORT
DB_PORT=${DB_PORT:-5432}

# Create database secrets
aws secretsmanager create-secret --name syncflow/database/host --secret-string "$DB_HOST" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/database/host --secret-string "$DB_HOST" --region $AWS_REGION

aws secretsmanager create-secret --name syncflow/database/user --secret-string "$DB_USER" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/database/user --secret-string "$DB_USER" --region $AWS_REGION

aws secretsmanager create-secret --name syncflow/database/password --secret-string "$DB_PASSWORD" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/database/password --secret-string "$DB_PASSWORD" --region $AWS_REGION

aws secretsmanager create-secret --name syncflow/database/name --secret-string "$DB_NAME" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/database/name --secret-string "$DB_NAME" --region $AWS_REGION

aws secretsmanager create-secret --name syncflow/database/port --secret-string "$DB_PORT" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/database/port --secret-string "$DB_PORT" --region $AWS_REGION

# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 32)
aws secretsmanager create-secret --name syncflow/jwt/secret --secret-string "$JWT_SECRET" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/jwt/secret --secret-string "$JWT_SECRET" --region $AWS_REGION

# OAuth credentials
read -p "Enter Google OAuth Client ID: " GOOGLE_CLIENT_ID
read -s -p "Enter Google OAuth Client Secret: " GOOGLE_CLIENT_SECRET
echo

aws secretsmanager create-secret --name syncflow/google/client_id --secret-string "$GOOGLE_CLIENT_ID" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/google/client_id --secret-string "$GOOGLE_CLIENT_ID" --region $AWS_REGION

aws secretsmanager create-secret --name syncflow/google/client_secret --secret-string "$GOOGLE_CLIENT_SECRET" --region $AWS_REGION 2>/dev/null || \
    aws secretsmanager update-secret --secret-id syncflow/google/client_secret --secret-string "$GOOGLE_CLIENT_SECRET" --region $AWS_REGION

echo "✓ All secrets created successfully!"
echo ""
echo "Next steps:"
echo "1. Get secret ARNs and update task definitions"
echo "2. Add GitHub Secrets (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, etc.)"
echo "3. Update OAuth redirect URIs in Google/Intuit dashboards"
```

## Checklist

Use this checklist to track your setup:

- [ ] AWS IAM user created with access keys
- [ ] GitHub Secrets configured:
  - [ ] `AWS_ACCESS_KEY_ID`
  - [ ] `AWS_SECRET_ACCESS_KEY`
  - [ ] `ECS_TASK_EXECUTION_ROLE_ARN`
  - [ ] `ECS_TASK_ROLE_ARN`
  - [ ] `VITE_API_BASE_URL`
- [ ] AWS Secrets Manager secrets created:
  - [ ] `syncflow/database/host`
  - [ ] `syncflow/database/user`
  - [ ] `syncflow/database/password`
  - [ ] `syncflow/database/name`
  - [ ] `syncflow/database/port`
  - [ ] `syncflow/jwt/secret`
  - [ ] `syncflow/google/client_id`
  - [ ] `syncflow/google/client_secret`
- [ ] Task definitions updated with secret ARNs
- [ ] OAuth redirect URIs updated in provider dashboards
- [ ] SSL certificate obtained from ACM
- [ ] Domain DNS configured
- [ ] RDS database created and accessible

## Security Best Practices

1. **Never commit secrets to Git** - Use AWS Secrets Manager
2. **Use IAM roles** - Limit permissions to minimum required
3. **Rotate secrets regularly** - Especially JWT secrets and OAuth credentials
4. **Enable MFA** - For AWS console access
5. **Use separate IAM users** - One for CI/CD, different from personal access
6. **Enable CloudTrail** - For audit logging
7. **Use VPC endpoints** - For private communication with AWS services
8. **Enable encryption at rest** - For RDS and Secrets Manager
9. **Enable encryption in transit** - Use HTTPS/TLS everywhere

## Getting Help

If you need help finding any of these values:

- **AWS Account ID**: `aws sts get-caller-identity --query Account --output text`
- **AWS Region**: Check your AWS CLI config: `aws configure get region`
- **IAM Role ARNs**: `aws iam list-roles --query 'Roles[?RoleName==`ecsTaskExecutionRole`].Arn'`
- **Secret ARNs**: `aws secretsmanager list-secrets --query 'SecretList[?Name==`syncflow/database/host`].ARN'`
- **ECR Repository URLs**: `aws ecr describe-repositories --query 'repositories[*].repositoryUri'`

