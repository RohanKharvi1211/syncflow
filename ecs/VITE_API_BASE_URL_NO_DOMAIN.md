# VITE_API_BASE_URL Without a Domain Name

Since `VITE_API_BASE_URL` is a **build-time variable** (it's embedded into the frontend code during the Docker build), you need to provide a value **before** building the frontend.

Here are your options when you don't have a custom domain:

## Option 1: Use ALB DNS Name (Recommended)

Create an Application Load Balancer (ALB) first, then use its DNS name.

### Step 1: Create an ALB for Backend

You can create an ALB using AWS CLI:

```bash
# Get VPC and subnet IDs (if using Terraform, these are in terraform outputs)
# Or use existing VPC/subnets

# Get default VPC and subnets (if you haven't created custom ones)
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=is-default,Values=true" --query 'Vpcs[0].VpcId' --output text)
SUBNET_IDS=$(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$VPC_ID" --query 'Subnets[*].SubnetId' --output text | tr '\t' ',')

# Create security group for ALB
ALB_SG_ID=$(aws ec2 create-security-group \
    --group-name syncflow-backend-alb-sg \
    --description "Security group for backend ALB" \
    --vpc-id $VPC_ID \
    --query 'GroupId' \
    --output text)

# Allow HTTP/HTTPS traffic
aws ec2 authorize-security-group-ingress \
    --group-id $ALB_SG_ID \
    --protocol tcp \
    --port 80 \
    --cidr 0.0.0.0/0

aws ec2 authorize-security-group-ingress \
    --group-id $ALB_SG_ID \
    --protocol tcp \
    --port 443 \
    --cidr 0.0.0.0/0

# Create ALB
ALB_ARN=$(aws elbv2 create-load-balancer \
    --name syncflow-backend-alb \
    --subnets $(echo $SUBNET_IDS | tr ' ' ',') \
    --security-groups $ALB_SG_ID \
    --scheme internet-facing \
    --type application \
    --ip-address-type ipv4 \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text)

# Wait for ALB to be active
echo "Waiting for ALB to be active..."
aws elbv2 wait load-balancer-available --load-balancer-arns $ALB_ARN

# Get ALB DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns $ALB_ARN \
    --query 'LoadBalancers[0].DNSName' \
    --output text)

echo "Backend ALB DNS: $ALB_DNS"
echo "Use this for VITE_API_BASE_URL: http://$ALB_DNS"
```

**Note:** Without a custom domain, the ALB will use HTTP (not HTTPS). For production, you should add a custom domain and SSL certificate later.

### Step 2: Use the ALB DNS Name

After creating the ALB, use its DNS name:

```
http://syncflow-backend-alb-123456789.us-east-1.elb.amazonaws.com
```

**Example:**
- If ALB DNS is: `syncflow-backend-alb-123456789.us-east-1.elb.amazonaws.com`
- Use for `VITE_API_BASE_URL`: `http://syncflow-backend-alb-123456789.us-east-1.elb.amazonaws.com`

---

## Option 2: Use a Placeholder (Update Later)

If you want to proceed with deployment first and update the API URL later:

### Temporary Value:
```
http://localhost:8080
```

**⚠️ Important:** This will **NOT work** in production because:
- The frontend will try to call `http://localhost:8080/api/...` which doesn't exist on users' machines
- You'll need to update this secret and **redeploy** the frontend after getting the actual ALB URL

### After Deployment:
1. Create the ALB and get its DNS name
2. Update `VITE_API_BASE_URL` secret in GitHub
3. Trigger a new frontend deployment (push code or manually trigger workflow)
4. Frontend will rebuild with the correct API URL

---

## Option 3: Create ALB After First Deployment

If you haven't created the ALB yet and want to deploy immediately:

### For Now:
Use a placeholder that you'll recognize needs updating:
```
http://PLACEHOLDER-UPDATE-AFTER-ALB-CREATION
```

### After Creating ALB:
1. Get ALB DNS name:
   ```bash
   aws elbv2 describe-load-balancers \
       --query 'LoadBalancers[?contains(LoadBalancerName, `syncflow`) || contains(LoadBalancerName, `backend`)].DNSName' \
       --output text
   ```

2. Update GitHub Secret `VITE_API_BASE_URL` with the actual ALB DNS

3. Redeploy frontend:
   - Push a commit to trigger deployment, OR
   - Manually trigger the GitHub Actions workflow

---

## Recommended Approach

**For a production-like setup without a custom domain:**

1. **Create the ALB first** (using Option 1 script above)
2. **Get the ALB DNS name**
3. **Use that DNS name** for `VITE_API_BASE_URL`: `http://YOUR-ALB-DNS.us-east-1.elb.amazonaws.com`
4. **Add to GitHub Secrets**
5. **Deploy** - The frontend will be built with the correct API URL

**Example workflow:**
```bash
# 1. Create ALB and get DNS
./create-alb.sh  # (you'll need to create this script or use AWS Console)

# 2. Get the DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --query 'LoadBalancers[?contains(LoadBalancerName, `syncflow`)].DNSName' \
    --output text)

# 3. Add to GitHub Secrets
echo "VITE_API_BASE_URL=http://$ALB_DNS"
# Copy this value and add it to GitHub Repository > Settings > Secrets
```

---

## Quick Helper Script

Here's a script to create an ALB and get its DNS name:

```bash
#!/bin/bash
# create-alb.sh - Create ALB and get DNS name for VITE_API_BASE_URL

set -e

AWS_REGION=${AWS_REGION:-us-east-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "Creating Application Load Balancer for backend..."

# Get VPC ID (use default VPC or create custom)
VPC_ID=$(aws ec2 describe-vpcs \
    --filters "Name=tag:Name,Values=${PROJECT_NAME}-vpc" \
    --query 'Vpcs[0].VpcId' \
    --output text 2>/dev/null)

# Fallback to default VPC if custom VPC not found
if [ -z "$VPC_ID" ] || [ "$VPC_ID" == "None" ]; then
    echo "Using default VPC..."
    VPC_ID=$(aws ec2 describe-vpcs \
        --filters "Name=is-default,Values=true" \
        --query 'Vpcs[0].VpcId' \
        --output text)
fi

echo "Using VPC: $VPC_ID"

# Get subnets
SUBNET_IDS=$(aws ec2 describe-subnets \
    --filters "Name=vpc-id,Values=$VPC_ID" \
    --query 'Subnets[*].SubnetId' \
    --output text | tr '\t' ' ')

if [ -z "$SUBNET_IDS" ]; then
    echo "Error: No subnets found in VPC $VPC_ID"
    exit 1
fi

SUBNET_LIST=$(echo $SUBNET_IDS | tr ' ' ',' | cut -d',' -f1,2)  # Use first 2 subnets
echo "Using subnets: $SUBNET_LIST"

# Create or get security group
SG_NAME="${PROJECT_NAME}-backend-alb-sg"
SG_ID=$(aws ec2 describe-security-groups \
    --filters "Name=group-name,Values=$SG_NAME" "Name=vpc-id,Values=$VPC_ID" \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null)

if [ -z "$SG_ID" ] || [ "$SG_ID" == "None" ]; then
    echo "Creating security group..."
    SG_ID=$(aws ec2 create-security-group \
        --group-name "$SG_NAME" \
        --description "Security group for ${PROJECT_NAME} backend ALB" \
        --vpc-id "$VPC_ID" \
        --query 'GroupId' \
        --output text)
    
    # Allow HTTP and HTTPS
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 2>/dev/null || echo "Port 80 rule may already exist"
    
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 443 \
        --cidr 0.0.0.0/0 2>/dev/null || echo "Port 443 rule may already exist"
fi

echo "Using security group: $SG_ID"

# Create ALB
ALB_NAME="${PROJECT_NAME}-backend-alb"
echo "Creating ALB: $ALB_NAME..."

ALB_ARN=$(aws elbv2 create-load-balancer \
    --name "$ALB_NAME" \
    --subnets $(echo $SUBNET_LIST | tr ',' ' ') \
    --security-groups "$SG_ID" \
    --scheme internet-facing \
    --type application \
    --ip-address-type ipv4 \
    --region "$AWS_REGION" \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text 2>/dev/null || \
    aws elbv2 describe-load-balancers \
        --names "$ALB_NAME" \
        --region "$AWS_REGION" \
        --query 'LoadBalancers[0].LoadBalancerArn' \
        --output text)

if [ -z "$ALB_ARN" ] || [ "$ALB_ARN" == "None" ]; then
    echo "Error: Failed to create or find ALB"
    exit 1
fi

echo "Waiting for ALB to be active..."
aws elbv2 wait load-balancer-available \
    --load-balancer-arns "$ALB_ARN" \
    --region "$AWS_REGION"

# Get DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns "$ALB_ARN" \
    --region "$AWS_REGION" \
    --query 'LoadBalancers[0].DNSName' \
    --output text)

echo ""
echo "=========================================="
echo "✓ ALB Created Successfully!"
echo "=========================================="
echo ""
echo "ALB DNS Name: $ALB_DNS"
echo ""
echo "Use this value for VITE_API_BASE_URL:"
echo "  http://$ALB_DNS"
echo ""
echo "Next steps:"
echo "1. Add to GitHub Secret: VITE_API_BASE_URL = http://$ALB_DNS"
echo "2. Configure ALB listener to route to your ECS service"
echo "3. Create target group and attach to ALB"
echo ""
```

---

## Summary

**Best approach without a domain:**

1. **Create ALB first** (using script above or AWS Console)
2. **Get ALB DNS name** (e.g., `syncflow-backend-alb-123456789.us-east-1.elb.amazonaws.com`)
3. **Use for VITE_API_BASE_URL**: `http://syncflow-backend-alb-123456789.us-east-1.elb.amazonaws.com`
4. **Add to GitHub Secrets**
5. **Deploy**

**Later, when you get a domain:**
- Request SSL certificate in AWS Certificate Manager
- Add custom domain to ALB
- Update `VITE_API_BASE_URL` to `https://api.yourdomain.com`
- Redeploy frontend

---

## Important Notes

- **HTTP vs HTTPS**: Without a custom domain/SSL certificate, you'll use HTTP. This is fine for development but not recommended for production with sensitive data.
- **Build-time variable**: `VITE_API_BASE_URL` is embedded at build time, so changing the GitHub secret won't update already-built images. You must rebuild/redeploy.
- **ALB Configuration**: After creating the ALB, you'll need to:
  - Create a target group pointing to your ECS service
  - Configure ALB listener rules (port 80 → target group)
  - Associate ECS service with the target group

