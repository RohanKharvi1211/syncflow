# Quick Start: Required Credentials for ECS Deployment

## Prerequisites

1. **Install AWS CLI** (if not already installed):
   ```bash
   # macOS
   brew install awscli
   
   # Verify installation
   aws --version
   ```

2. **Configure AWS CLI**:
   ```bash
   aws configure
   ```
   
   See `AWS_CLI_SETUP.md` for detailed setup instructions.

## Summary

You need **2 types of credentials**:

1. **GitHub Secrets** (for CI/CD automation)
2. **AWS Secrets Manager** (for runtime application secrets)

---

## 1. GitHub Secrets (Add in Repository Settings > Secrets)

These are used by GitHub Actions to deploy to AWS:

```
AWS_ACCESS_KEY_ID=<your-aws-access-key>
AWS_SECRET_ACCESS_KEY=<your-aws-secret-key>
ECS_TASK_EXECUTION_ROLE_ARN=arn:aws:iam::YOUR_ACCOUNT_ID:role/ecsTaskExecutionRole
ECS_TASK_ROLE_ARN=arn:aws:iam::YOUR_ACCOUNT_ID:role/ecsTaskRole
VITE_API_BASE_URL=https://api.yourdomain.com
```

**How to create AWS access keys:**
```bash
# Create IAM user
aws iam create-user --user-name syncflow-cicd

# Create access key
aws iam create-access-key --user-name syncflow-cicd

# Attach policies
aws iam attach-user-policy --user-name syncflow-cicd --policy-arn arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser
aws iam attach-user-policy --user-name syncflow-cicd --policy-arn arn:aws:iam::aws:policy/AmazonECS_FullAccess
```

---

## 2. AWS Secrets Manager Secrets

Store these in AWS Secrets Manager. The backend will pull them at runtime.

### Required Secrets:

```bash
# Run the helper script to create all secrets interactively:
cd ecs
./create-secrets.sh
```

**Or create manually:**

```bash
AWS_REGION=us-east-1

# Database
aws secretsmanager create-secret --name syncflow/database/host --secret-string "your-rds-endpoint.rds.amazonaws.com" --region $AWS_REGION
aws secretsmanager create-secret --name syncflow/database/user --secret-string "postgres" --region $AWS_REGION
aws secretsmanager create-secret --name syncflow/database/password --secret-string "YOUR_DB_PASSWORD" --region $AWS_REGION
aws secretsmanager create-secret --name syncflow/database/name --secret-string "syncflow" --region $AWS_REGION
aws secretsmanager create-secret --name syncflow/database/port --secret-string "5432" --region $AWS_REGION

# JWT Secret (generate random)
JWT_SECRET=$(openssl rand -base64 32)
aws secretsmanager create-secret --name syncflow/jwt/secret --secret-string "$JWT_SECRET" --region $AWS_REGION

# Google OAuth
aws secretsmanager create-secret --name syncflow/google/client_id --secret-string "YOUR_GOOGLE_CLIENT_ID" --region $AWS_REGION
aws secretsmanager create-secret --name syncflow/google/client_secret --secret-string "YOUR_GOOGLE_CLIENT_SECRET" --region $AWS_REGION
```

### Where to Get OAuth Credentials:

- **Google OAuth**: [Google Cloud Console > APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials)
- **QuickBooks OAuth**: [Intuit Developer Dashboard](https://developer.intuit.com/app/developer/myapps)

---

## 3. After Creating Secrets

### Get Secret ARNs:

```bash
cd ecs
./get-secret-arns.sh
```

### Update Task Definitions:

Edit `ecs/task-definition-backend.json` and replace:
- `YOUR_ACCOUNT_ID` → Your AWS account ID
- `YOUR_ECR_REGISTRY` → `YOUR_ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com`
- Secret ARNs → Use ARNs from `get-secret-arns.sh`
- Update OAuth redirect URIs with your domain

### Update OAuth Redirect URIs:

**Google Cloud Console:**
- Add: `https://api.yourdomain.com/api/oauth/google/callback`

**Intuit Developer Dashboard:**
- Add: `https://api.yourdomain.com/api/oauth/quickbooks/callback`

---

## Complete Checklist

- [ ] AWS IAM user created (`syncflow-cicd`)
- [ ] AWS access keys generated
- [ ] GitHub Secrets added (5 secrets)
- [ ] AWS Secrets Manager secrets created (8 secrets)
- [ ] Secret ARNs retrieved
- [ ] Task definitions updated with ARNs
- [ ] OAuth redirect URIs updated
- [ ] Domain configured (DNS + SSL certificate)

---

## Quick Commands

```bash
# Create all secrets interactively
cd ecs && ./create-secrets.sh

# Get all secret ARNs
cd ecs && ./get-secret-arns.sh

# Setup infrastructure (ECR, ECS cluster, IAM roles)
cd ecs && ./setup-infrastructure.sh

# Get IAM role ARNs
aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text
aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text

# Get AWS Account ID
aws sts get-caller-identity --query Account --output text
```

For detailed instructions, see `CREDENTIALS_CHECKLIST.md` and `deployment-guide.md`.

