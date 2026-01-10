# ECS Deployment Guide for SyncFlow

This guide will help you deploy SyncFlow to Amazon ECS using Fargate.

## Prerequisites

1. AWS Account with appropriate permissions
2. AWS CLI configured (`aws configure`)
3. Docker installed locally (for testing)
4. GitHub repository with Actions enabled
5. ECR repositories created (or they will be created automatically)

## Step 1: Create ECR Repositories

```bash
# Set variables
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Create ECR repositories
aws ecr create-repository \
    --repository-name syncflow-backend \
    --region $AWS_REGION \
    --image-scanning-configuration scanOnPush=true

aws ecr create-repository \
    --repository-name syncflow-frontend \
    --region $AWS_REGION \
    --image-scanning-configuration scanOnPush=true
```

## Step 2: Create ECS Cluster

```bash
aws ecs create-cluster \
    --cluster-name syncflow-cluster \
    --region $AWS_REGION \
    --capacity-providers FARGATE FARGATE_SPOT \
    --default-capacity-provider-strategy \
        capacityProvider=FARGATE,weight=1 \
        capacityProvider=FARGATE_SPOT,weight=1
```

## Step 3: Create VPC and Networking (if not exists)

```bash
# Create VPC
aws ec2 create-vpc --cidr-block 10.0.0.0/16 --region $AWS_REGION

# Create subnets (at least 2 in different AZs)
# Create Internet Gateway
# Create Route Tables
# Create Security Groups

# Or use existing VPC - update task definitions with your VPC subnets and security groups
```

## Step 4: Create IAM Roles

### Task Execution Role (for pulling images from ECR)

```bash
# Create trust policy
cat > task-execution-role-trust.json <<EOF
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

# Create role
aws iam create-role \
    --role-name ecsTaskExecutionRole \
    --assume-role-policy-document file://task-execution-role-trust.json

# Attach policy
aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

# Allow access to Secrets Manager
aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/SecretsManagerReadWrite
```

### Task Role (for application access to AWS services)

```bash
# Create task role (minimal permissions for now)
aws iam create-role \
    --role-name ecsTaskRole \
    --assume-role-policy-document file://task-execution-role-trust.json
```

## Step 5: Create CloudWatch Log Groups

```bash
aws logs create-log-group \
    --log-group-name /ecs/syncflow-backend \
    --region $AWS_REGION

aws logs create-log-group \
    --log-group-name /ecs/syncflow-frontend \
    --region $AWS_REGION
```

## Step 6: Store Secrets in AWS Secrets Manager

```bash
# Database secrets
aws secretsmanager create-secret \
    --name syncflow/database/host \
    --secret-string "your-rds-endpoint.rds.amazonaws.com" \
    --region $AWS_REGION

aws secretsmanager create-secret \
    --name syncflow/database/user \
    --secret-string "postgres" \
    --region $AWS_REGION

aws secretsmanager create-secret \
    --name syncflow/database/password \
    --secret-string "your-db-password" \
    --region $AWS_REGION

aws secretsmanager create-secret \
    --name syncflow/database/name \
    --secret-string "syncflow" \
    --region $AWS_REGION

aws secretsmanager create-secret \
    --name syncflow/database/port \
    --secret-string "5432" \
    --region $AWS_REGION

# JWT Secret
aws secretsmanager create-secret \
    --name syncflow/jwt/secret \
    --secret-string "your-jwt-secret-key" \
    --region $AWS_REGION

# Google OAuth
aws secretsmanager create-secret \
    --name syncflow/google/client_id \
    --secret-string "your-google-client-id" \
    --region $AWS_REGION

aws secretsmanager create-secret \
    --name syncflow/google/client_secret \
    --secret-string "your-google-client-secret" \
    --region $AWS_REGION
```

## Step 7: Update Task Definitions

Update the task definition files in `ecs/` directory:

1. Replace `YOUR_ACCOUNT_ID` with your AWS account ID
2. Replace `YOUR_ECR_REGISTRY` with `$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com`
3. Replace `REGION` with your AWS region (e.g., `us-east-1`)
4. Update secret ARNs with actual secret ARNs from Secrets Manager
5. Update `FRONTEND_URL` and OAuth redirect URIs with your domain

## Step 8: Register Task Definitions

```bash
# Register backend task definition
aws ecs register-task-definition \
    --cli-input-json file://ecs/task-definition-backend.json \
    --region $AWS_REGION

# Register frontend task definition
aws ecs register-task-definition \
    --cli-input-json file://ecs/task-definition-frontend.json \
    --region $AWS_REGION
```

## Step 9: Create ECS Services

```bash
# Create backend service
aws ecs create-service \
    --cluster syncflow-cluster \
    --service-name syncflow-backend-service \
    --task-definition syncflow-backend-task \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx,subnet-yyy],securityGroups=[sg-xxx],assignPublicIp=ENABLED}" \
    --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:REGION:ACCOUNT_ID:targetgroup/backend-tg/xxx,containerName=backend,containerPort=8080" \
    --region $AWS_REGION

# Create frontend service
aws ecs create-service \
    --cluster syncflow-cluster \
    --service-name syncflow-frontend-service \
    --task-definition syncflow-frontend-task \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx,subnet-yyy],securityGroups=[sg-xxx],assignPublicIp=ENABLED}" \
    --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:REGION:ACCOUNT_ID:targetgroup/frontend-tg/xxx,containerName=frontend,containerPort=80" \
    --region $AWS_REGION
```

## Step 10: Configure GitHub Secrets

Add these secrets to your GitHub repository (Settings > Secrets and variables > Actions):

- `AWS_ACCESS_KEY_ID`: Your AWS access key
- `AWS_SECRET_ACCESS_KEY`: Your AWS secret key
- `VITE_API_BASE_URL`: Your backend API URL (e.g., `https://api.yourdomain.com`)

## Step 11: Create Application Load Balancer (ALB)

```bash
# Create ALB for backend
aws elbv2 create-load-balancer \
    --name syncflow-backend-alb \
    --subnets subnet-xxx subnet-yyy \
    --security-groups sg-xxx \
    --scheme internet-facing \
    --type application \
    --region $AWS_REGION

# Create ALB for frontend (or use same ALB with different target groups)
aws elbv2 create-load-balancer \
    --name syncflow-frontend-alb \
    --subnets subnet-xxx subnet-yyy \
    --security-groups sg-xxx \
    --scheme internet-facing \
    --type application \
    --region $AWS_REGION

# Create target groups and listeners
# (See AWS documentation for full ALB setup)
```

## Step 12: Update Environment Variables

In the GitHub Actions workflow file (`.github/workflows/deploy-ecs.yml`), update:

- `AWS_REGION`: Your AWS region
- `ECR_REPOSITORY_BACKEND`: ECR repository name (default: `syncflow-backend`)
- `ECR_REPOSITORY_FRONTEND`: ECR repository name (default: `syncflow-frontend`)
- `ECS_CLUSTER`: Your cluster name (default: `syncflow-cluster`)
- Service and task definition names as needed

## Step 13: Run Database Migrations

Before deploying, run database migrations:

```bash
# Option 1: Run migrations in a one-off ECS task
aws ecs run-task \
    --cluster syncflow-cluster \
    --task-definition syncflow-backend-task \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}" \
    --overrides '{"containerOverrides":[{"name":"backend","command":["/app/migrate"]}]}' \
    --region $AWS_REGION

# Option 2: Use a separate migration container or script
```

## Step 14: Deploy

Once everything is set up, push to the `main` branch and GitHub Actions will:

1. Build Docker images for backend and frontend
2. Push images to ECR
3. Update ECS task definitions
4. Deploy new task definitions to ECS services
5. Wait for services to stabilize

## Troubleshooting

### Check ECS Service Status
```bash
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-backend-service syncflow-frontend-service \
    --region $AWS_REGION
```

### Check Task Logs
```bash
# Backend logs
aws logs tail /ecs/syncflow-backend --follow --region $AWS_REGION

# Frontend logs
aws logs tail /ecs/syncflow-frontend --follow --region $AWS_REGION
```

### View Failed Tasks
```bash
aws ecs list-tasks \
    --cluster syncflow-cluster \
    --desired-status STOPPED \
    --region $AWS_REGION
```

## Security Considerations

1. Use AWS Secrets Manager for all sensitive values (not environment variables)
2. Use private subnets for ECS tasks and NAT Gateway for outbound internet
3. Enable VPC Flow Logs for network monitoring
4. Use AWS WAF for additional protection
5. Enable CloudTrail for audit logging
6. Regularly rotate secrets
7. Use least privilege IAM policies

## Cost Optimization

1. Use Fargate Spot for non-critical workloads (50-70% savings)
2. Right-size CPU and memory allocations
3. Use CloudWatch alarms to scale down during low usage
4. Enable ECS auto-scaling based on CPU/memory metrics
5. Use Application Load Balancer with idle timeout optimization

