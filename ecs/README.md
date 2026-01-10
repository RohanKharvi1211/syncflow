# ECS Deployment Guide

This directory contains all the files needed to deploy SyncFlow to Amazon ECS.

## Quick Start

### 1. One-Time Setup

Run the infrastructure setup script:

```bash
cd ecs
./setup-infrastructure.sh
```

This will create:
- ECR repositories for backend and frontend
- ECS cluster
- CloudWatch log groups
- IAM roles

### 2. Configure Secrets

Store your secrets in AWS Secrets Manager (see `deployment-guide.md` for details):

- Database credentials
- JWT secret
- Google OAuth credentials
- QuickBooks OAuth credentials (optional)

### 3. Update Task Definitions

Edit `task-definition-backend.json` and `task-definition-frontend.json`:

- Replace `YOUR_ACCOUNT_ID` with your AWS account ID
- Replace `YOUR_ECR_REGISTRY` with your ECR registry URL
- Update secret ARNs with actual ARNs from Secrets Manager
- Update OAuth redirect URIs with your domain

### 4. Register Task Definitions

```bash
aws ecs register-task-definition \
    --cli-input-json file://task-definition-backend.json \
    --region us-east-1

aws ecs register-task-definition \
    --cli-input-json file://task-definition-frontend.json \
    --region us-east-1
```

### 5. Create ECS Services

Use the AWS Console, CLI, or Terraform to create ECS services. See `deployment-guide.md` for CLI commands.

### 6. Configure GitHub Actions

Add these secrets to your GitHub repository:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `ECS_TASK_EXECUTION_ROLE_ARN` (e.g., `arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole`)
- `ECS_TASK_ROLE_ARN` (e.g., `arn:aws:iam::ACCOUNT_ID:role/ecsTaskRole`)
- `VITE_API_BASE_URL` (e.g., `https://api.yourdomain.com`)

### 7. Deploy

Push to `main` branch - GitHub Actions will automatically build and deploy!

Or deploy manually:

```bash
./deploy.sh all
```

## Files

- `task-definition-backend.json` - ECS task definition for backend
- `task-definition-frontend.json` - ECS task definition for frontend
- `deployment-guide.md` - Detailed deployment instructions
- `setup-infrastructure.sh` - Script to create initial AWS resources
- `deploy.sh` - Manual deployment script
- `terraform/main.tf` - Terraform configuration (optional)

## Architecture

```
Internet
   |
   v
[Application Load Balancer (Frontend)]
   |
   v
[ECS Service - Frontend (Fargate)]
   |  (nginx serving React app)
   
[Application Load Balancer (Backend)]
   |
   v
[ECS Service - Backend (Fargate)]
   |  (Go API server)
   |
   v
[RDS PostgreSQL]
```

## Requirements

- Backend: 512 CPU units, 1024 MB memory
- Frontend: 256 CPU units, 512 MB memory
- Database: RDS PostgreSQL (separate setup required)
- Load Balancer: Application Load Balancer with SSL certificate

## Environment Variables

Backend environment variables are stored in AWS Secrets Manager and injected at runtime.

Frontend build-time variables (VITE_API_BASE_URL) should be set in GitHub Actions secrets.

## Monitoring

- CloudWatch Logs: `/ecs/syncflow-backend` and `/ecs/syncflow-frontend`
- CloudWatch Container Insights: Enabled on cluster
- Health checks: Configured on both containers

## Scaling

ECS services support auto-scaling. Configure in AWS Console or using:

```bash
aws application-autoscaling register-scalable-target \
    --service-namespace ecs \
    --scalable-dimension ecs:service:DesiredCount \
    --resource-id service/syncflow-cluster/syncflow-backend-service \
    --min-capacity 1 \
    --max-capacity 10 \
    --region us-east-1
```

## Troubleshooting

See `deployment-guide.md` for troubleshooting steps and common issues.

