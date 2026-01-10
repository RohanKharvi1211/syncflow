#!/bin/bash
# create-alb.sh - Create ALB and get DNS name for VITE_API_BASE_URL

set -e

# Get AWS region from AWS CLI config or use default
AWS_REGION=${AWS_REGION:-$(aws configure get region 2>/dev/null || echo "us-east-1")}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "Creating Application Load Balancer for backend..."
echo "AWS Region: $AWS_REGION"

# Get VPC ID (use default VPC or create custom)
VPC_ID=$(aws ec2 describe-vpcs \
    --filters "Name=tag:Name,Values=${PROJECT_NAME}-vpc" \
    --region "$AWS_REGION" \
    --query 'Vpcs[0].VpcId' \
    --output text 2>/dev/null)

# Fallback to default VPC if custom VPC not found
if [ -z "$VPC_ID" ] || [ "$VPC_ID" == "None" ]; then
    echo "Using default VPC..."
    VPC_ID=$(aws ec2 describe-vpcs \
        --filters "Name=is-default,Values=true" \
        --region "$AWS_REGION" \
        --query 'Vpcs[0].VpcId' \
        --output text)
fi

echo "Using VPC: $VPC_ID"

# Get subnets (need at least 2 in different AZs)
SUBNET_IDS=$(aws ec2 describe-subnets \
    --filters "Name=vpc-id,Values=$VPC_ID" \
    --region "$AWS_REGION" \
    --query 'Subnets[*].SubnetId' \
    --output text)

if [ -z "$SUBNET_IDS" ]; then
    echo "Error: No subnets found in VPC $VPC_ID"
    exit 1
fi

# Get first 2 subnets (in different AZs if possible)
SUBNET_ARRAY=($SUBNET_IDS)
if [ ${#SUBNET_ARRAY[@]} -lt 2 ]; then
    echo "Error: Need at least 2 subnets for ALB. Found: ${#SUBNET_ARRAY[@]}"
    exit 1
fi

SUBNET_1=${SUBNET_ARRAY[0]}
SUBNET_2=${SUBNET_ARRAY[1]}
echo "Using subnets: $SUBNET_1, $SUBNET_2"

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
        --region "$AWS_REGION" \
        --query 'GroupId' \
        --output text)
    
    # Allow HTTP and HTTPS
    echo "Configuring security group rules..."
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 \
        --region "$AWS_REGION" 2>/dev/null || echo "Port 80 rule may already exist"
    
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 443 \
        --cidr 0.0.0.0/0 \
        --region "$AWS_REGION" 2>/dev/null || echo "Port 443 rule may already exist"
else
    echo "Using existing security group: $SG_ID"
fi

# Check if ALB already exists
ALB_NAME="${PROJECT_NAME}-backend-alb"
EXISTING_ALB=$(aws elbv2 describe-load-balancers \
    --names "$ALB_NAME" \
    --region "$AWS_REGION" \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text 2>/dev/null || echo "")

if [ -n "$EXISTING_ALB" ] && [ "$EXISTING_ALB" != "None" ]; then
    echo "ALB already exists: $EXISTING_ALB"
    ALB_ARN="$EXISTING_ALB"
else
    echo "Creating ALB: $ALB_NAME..."
    ALB_ARN=$(aws elbv2 create-load-balancer \
        --name "$ALB_NAME" \
        --subnets "$SUBNET_1" "$SUBNET_2" \
        --security-groups "$SG_ID" \
        --scheme internet-facing \
        --type application \
        --ip-address-type ipv4 \
        --region "$AWS_REGION" \
        --query 'LoadBalancers[0].LoadBalancerArn' \
        --output text)
    
    if [ -z "$ALB_ARN" ] || [ "$ALB_ARN" == "None" ]; then
        echo "Error: Failed to create ALB"
        exit 1
    fi
    
    echo "Waiting for ALB to be active (this may take a minute)..."
    aws elbv2 wait load-balancer-available \
        --load-balancer-arns "$ALB_ARN" \
        --region "$AWS_REGION"
fi

# Get DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns "$ALB_ARN" \
    --region "$AWS_REGION" \
    --query 'LoadBalancers[0].DNSName' \
    --output text)

echo ""
echo "=========================================="
echo "✓ ALB Ready!"
echo "=========================================="
echo ""
echo "ALB ARN: $ALB_ARN"
echo "ALB DNS Name: $ALB_DNS"
echo ""
echo "Use this value for VITE_API_BASE_URL GitHub Secret:"
echo "  http://$ALB_DNS"
echo ""
echo "Note:"
echo "- Using HTTP (not HTTPS) since no SSL certificate configured"
echo "- You'll need to configure ALB listener and target group later"
echo "- After deployment, update this to point to your ECS service"
echo ""

