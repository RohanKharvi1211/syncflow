# How to Get These 3 GitHub Secrets

This guide explains step-by-step how to get the values for these 3 GitHub Secrets:

1. `ECS_TASK_EXECUTION_ROLE_ARN`
2. `ECS_TASK_ROLE_ARN`
3. `VITE_API_BASE_URL`

---

## 1. ECS_TASK_EXECUTION_ROLE_ARN

### Step 1: Create the IAM Role (if not already created)

Run the infrastructure setup script which creates this role:

```bash
cd ecs
./setup-infrastructure.sh
```

Or create it manually:

```bash
# Create the role
aws iam create-role \
    --role-name ecsTaskExecutionRole \
    --assume-role-policy-document '{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "ecs-tasks.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
    }'

# Attach required policies
aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/SecretsManagerReadWrite
```

### Step 2: Get the ARN

```bash
# Get your AWS Account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Get the role ARN
aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text
```

**Example Output:**
```
arn:aws:iam::123456789012:role/ecsTaskExecutionRole
```

**Or construct it manually:**
```
arn:aws:iam::YOUR_ACCOUNT_ID:role/ecsTaskExecutionRole
```

Replace `YOUR_ACCOUNT_ID` with your actual AWS account ID from step 1.

---

## 2. ECS_TASK_ROLE_ARN

### Step 1: Create the IAM Role (if not already created)

The setup script creates this too, or create it manually:

```bash
# Create the role (uses same trust policy as execution role)
aws iam create-role \
    --role-name ecsTaskRole \
    --assume-role-policy-document '{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "ecs-tasks.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
    }'
```

**Note:** Task role doesn't need policies attached by default unless your app needs specific AWS service access.

### Step 2: Get the ARN

```bash
# Get your AWS Account ID (if not already saved)
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Get the role ARN
aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text
```

**Example Output:**
```
arn:aws:iam::123456789012:role/ecsTaskRole
```

**Or construct it manually:**
```
arn:aws:iam::YOUR_ACCOUNT_ID:role/ecsTaskRole
```

---

## 3. VITE_API_BASE_URL

This is the **backend API URL** that the frontend will call. You have two options:

### Option A: Use Your Domain (Production)

If you have a custom domain with an Application Load Balancer (ALB):

**Format:** `https://api.yourdomain.com`

**Steps:**

1. **Set up an Application Load Balancer (ALB)** in your AWS account
2. **Create an SSL certificate** in AWS Certificate Manager (ACM):
   ```bash
   aws acm request-certificate \
       --domain-name api.yourdomain.com \
       --validation-method DNS \
       --region us-east-1
   ```
3. **Point your DNS** to the ALB:
   - Create a CNAME record: `api.yourdomain.com` → `your-alb-dns-name.us-east-1.elb.amazonaws.com`
4. **Configure ALB listener** to use the certificate and route to your backend target group

**Then use:** `https://api.yourdomain.com`

### Option B: Use ALB DNS Name (Development/Testing)

If you don't have a custom domain yet, you can use the ALB DNS name temporarily:

**Format:** `http://your-alb-name-123456789.us-east-1.elb.amazonaws.com/api`

**⚠️ IMPORTANT:** The `/api` suffix is REQUIRED because the frontend code appends paths like `/oauth/google/initiate` to the base URL.

**Steps:**

1. **Create an ALB** (or use Terraform to create one)
2. **Get the ALB DNS name:**
   ```bash
   aws elbv2 describe-load-balancers \
       --query 'LoadBalancers[?contains(LoadBalancerName, `syncflow`)].DNSName' \
       --output text
   ```

**Then use:** `http://YOUR-ALB-DNS-NAME.us-east-1.elb.amazonaws.com/api` (with `/api` at the end!)

**Example:**
- ALB DNS: `syncflow-backend-alb-123456789.ap-south-1.elb.amazonaws.com`
- `VITE_API_BASE_URL`: `http://syncflow-backend-alb-123456789.ap-south-1.elb.amazonaws.com/api`
- This makes frontend call: `http://syncflow-backend-alb-123456789.ap-south-1.elb.amazonaws.com/api/oauth/google/initiate` ✅

**⚠️ Note:** Without a custom domain, you'll use HTTP (not HTTPS), which is not recommended for production.

### Option C: Manual Testing (Before ALB Setup)

If you haven't set up the ALB yet and just want to test the GitHub Actions workflow, you can use a placeholder:

**Temporary value:** `https://api.yourdomain.com` (or `http://localhost:8080` for local testing)

**Important:** After deploying, you'll need to:
1. Get the actual ALB DNS name or domain
2. Update this secret in GitHub
3. Rebuild/redeploy the frontend (since `VITE_API_BASE_URL` is a build-time variable)

---

## Quick Script to Get All 3 Values

Run this script to get all 3 values at once:

```bash
#!/bin/bash
# get-github-secrets.sh

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
EXECUTION_ROLE_ARN=$(aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text 2>/dev/null)
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
TASK_ROLE_ARN=$(aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text 2>/dev/null)
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
    --region $AWS_REGION 2>/dev/null)

if [ -z "$ALB_DNS" ]; then
    echo "   ⚠️  No ALB found. Options:"
    echo "      a) If you have a custom domain: https://api.yourdomain.com/api"
    echo "      b) If using ALB: Get ALB DNS from AWS Console after creating ALB"
    echo "      c) Temporary placeholder: https://api.yourdomain.com/api"
    VITE_API_BASE_URL="https://api.yourdomain.com/api"
else
    echo "   ✓ Found ALB: $ALB_DNS"
    echo "   Using: http://$ALB_DNS/api (⚠️  /api suffix required!)"
    VITE_API_BASE_URL="http://$ALB_DNS/api"
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
```

Save this as `get-github-secrets.sh`, make it executable, and run:

```bash
chmod +x get-github-secrets.sh
./get-github-secrets.sh
```

---

## Summary Checklist

- [ ] Run `./setup-infrastructure.sh` to create IAM roles
- [ ] Get `ECS_TASK_EXECUTION_ROLE_ARN` using: `aws iam get-role --role-name ecsTaskExecutionRole --query 'Role.Arn' --output text`
- [ ] Get `ECS_TASK_ROLE_ARN` using: `aws iam get-role --role-name ecsTaskRole --query 'Role.Arn' --output text`
- [ ] Determine `VITE_API_BASE_URL`:
  - [ ] Option A: Custom domain → `https://api.yourdomain.com`
  - [ ] Option B: ALB DNS name → `http://your-alb-dns.us-east-1.elb.amazonaws.com`
  - [ ] Option C: Placeholder → `https://api.yourdomain.com` (update after deployment)
- [ ] Add all 3 values to GitHub Repository > Settings > Secrets and variables > Actions

---

## After Adding to GitHub

Once you've added these secrets to GitHub:

1. **Push code to trigger GitHub Actions** (or manually trigger workflow)
2. **GitHub Actions will:**
   - Build Docker images
   - Push to ECR
   - Deploy to ECS
3. **After deployment**, update `VITE_API_BASE_URL` if needed with the actual backend URL
4. **Redeploy frontend** to pick up the new API URL

---

## Troubleshooting

### "Role not found" errors:

**Solution:** Create the roles first:
```bash
cd ecs
./setup-infrastructure.sh
```

### "Access Denied" when getting role ARN:

**Solution:** Your AWS credentials don't have IAM read permissions. Ensure you're using credentials with IAM access.

### Can't find ALB DNS:

**Solution:** ALB is created when you set up ECS services. Either:
- Set up infrastructure first (including ALB via Terraform)
- Use a placeholder URL temporarily
- Create ALB manually in AWS Console

### Need to update VITE_API_BASE_URL after deployment:

**Note:** Since `VITE_API_BASE_URL` is a build-time variable (Vite prefix), changing the GitHub secret won't update already-built images. You need to:
1. Update the secret in GitHub
2. Trigger a new build/deployment (push code or manually trigger workflow)
3. Frontend will rebuild with the new API URL

