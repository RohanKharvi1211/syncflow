# Pre-Deployment Checklist

## ⚠️ Issues Found Before Deployment

Before pushing code, you need to fix these issues:

### 1. **Region Mismatch** ❌
- **GitHub Actions workflow**: Configured for `us-east-1`
- **Your AWS resources**: In `ap-south-1`
- **Fix**: Update workflow to use `ap-south-1` or create resources in `us-east-1`

### 2. **ECS Services Don't Exist** ❌
- **Required**: `syncflow-backend-service` and `syncflow-frontend-service`
- **Current**: These services don't exist yet
- **Fix**: Create services before deployment, or update workflow to create them

### 3. **ECR Repositories May Not Exist** ⚠️
- **Required**: `syncflow-backend` and `syncflow-frontend` in ECR
- **Current**: Not verified in `ap-south-1`
- **Fix**: Ensure repositories exist in the correct region

### 4. **Secrets Manager ARNs** ⚠️
- **Task definitions**: Have placeholder ARNs for secrets
- **Fix**: Update secret ARNs or ensure secrets exist in correct region

---

## Quick Fix Options

### Option A: Use ap-south-1 (Recommended)
Since your ALB and infrastructure are in `ap-south-1`, update everything to use that region.

### Option B: Use us-east-1
Recreate all resources in `us-east-1` to match the workflow.

---

## Step-by-Step Fix (Option A - ap-south-1)

### 1. Update GitHub Actions Workflow Region

```yaml
# Change in .github/workflows/deploy-ecs.yml
env:
  AWS_REGION: ap-south-1  # Changed from us-east-1
```

### 2. Verify/Create ECR Repositories in ap-south-1

```bash
# Check if repositories exist
aws ecr describe-repositories --region ap-south-1 --query 'repositories[*].repositoryName'

# If not, create them
aws ecr create-repository --repository-name syncflow-backend --region ap-south-1
aws ecr create-repository --repository-name syncflow-frontend --region ap-south-1
```

### 3. Verify ECS Cluster Exists in ap-south-1

```bash
# Check cluster
aws ecs describe-clusters --clusters syncflow-cluster --region ap-south-1

# If not exists, create it (or run setup-infrastructure.sh with region)
AWS_REGION=ap-south-1 ./setup-infrastructure.sh
```

### 4. Create ECS Services (Before First Deployment)

ECS services need to be created before the deployment workflow can update them. Create them manually or use Terraform:

```bash
# Create backend service (requires: cluster, task definition, target group, subnets, security groups)
# This is complex - better to use Terraform or create via Console
```

### 5. Ensure Secrets Manager Secrets Exist in ap-south-1

```bash
# Check secrets
aws secretsmanager list-secrets --region ap-south-1 --query 'SecretList[?starts_with(Name, `syncflow`)].Name'

# If not, create them in ap-south-1
AWS_REGION=ap-south-1 ./create-secrets.sh
```

---

## Easier Option: Create Services After First Task Definition

The GitHub Actions workflow will:
1. Build Docker images ✅
2. Push to ECR ✅
3. Register/update task definitions ✅
4. **Deploy to services** ❌ (services must exist first)

**Solution**: Create ECS services manually after the first successful task definition registration.

---

## Recommended: Update Workflow to Create Services If Missing

We should update the workflow to create services if they don't exist, or provide a separate script to create services.

