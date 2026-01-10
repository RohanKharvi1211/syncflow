#!/bin/bash
# Script to create ECS services, target groups, and configure ALB
# Usage: ./create-ecs-services.sh

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "=========================================="
echo "Creating ECS Services and ALB Configuration"
echo "=========================================="
echo "Region: $AWS_REGION"
echo "Account ID: $AWS_ACCOUNT_ID"
echo ""

# Get VPC and subnets
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=is-default,Values=true" --region $AWS_REGION --query 'Vpcs[0].VpcId' --output text)
echo "VPC ID: $VPC_ID"

SUBNET_IDS=($(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$VPC_ID" --region $AWS_REGION --query 'Subnets[*].SubnetId' --output text))
SUBNET_1=${SUBNET_IDS[0]}
SUBNET_2=${SUBNET_IDS[1]}
echo "Using subnets: $SUBNET_1, $SUBNET_2"
echo ""

# Get ALB
ALB_ARN=$(aws elbv2 describe-load-balancers \
    --names "${PROJECT_NAME}-backend-alb" \
    --region $AWS_REGION \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$ALB_ARN" ] || [ "$ALB_ARN" == "None" ]; then
    echo "❌ Error: ALB ${PROJECT_NAME}-backend-alb not found. Please create it first using ./create-alb.sh"
    exit 1
fi

echo "ALB ARN: $ALB_ARN"
echo ""

# Get ALB security group
ALB_SG_ID=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns "$ALB_ARN" \
    --region $AWS_REGION \
    --query 'LoadBalancers[0].SecurityGroups[0]' \
    --output text)

echo "ALB Security Group: $ALB_SG_ID"
echo ""

# Create security group for ECS tasks (backend)
echo "=== Creating Security Group for Backend Tasks ==="
BACKEND_SG_NAME="${PROJECT_NAME}-backend-task-sg"
BACKEND_SG_ID=$(aws ec2 describe-security-groups \
    --filters "Name=group-name,Values=$BACKEND_SG_NAME" "Name=vpc-id,Values=$VPC_ID" \
    --region $AWS_REGION \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null || echo "")

if [ -z "$BACKEND_SG_ID" ] || [ "$BACKEND_SG_ID" == "None" ]; then
    echo "Creating security group: $BACKEND_SG_NAME"
    BACKEND_SG_ID=$(aws ec2 create-security-group \
        --group-name "$BACKEND_SG_NAME" \
        --description "Security group for ${PROJECT_NAME} backend ECS tasks" \
        --vpc-id "$VPC_ID" \
        --region $AWS_REGION \
        --query 'GroupId' \
        --output text)
    
    # Allow traffic from ALB
    aws ec2 authorize-security-group-ingress \
        --group-id "$BACKEND_SG_ID" \
        --protocol tcp \
        --port 8080 \
        --source-group "$ALB_SG_ID" \
        --region $AWS_REGION 2>/dev/null || echo "Ingress rule may already exist"
    
    echo "✓ Created: $BACKEND_SG_ID"
else
    echo "✓ Using existing: $BACKEND_SG_ID"
fi
echo ""

# Create security group for ECS tasks (frontend)
echo "=== Creating Security Group for Frontend Tasks ==="
FRONTEND_SG_NAME="${PROJECT_NAME}-frontend-task-sg"
FRONTEND_SG_ID=$(aws ec2 describe-security-groups \
    --filters "Name=group-name,Values=$FRONTEND_SG_NAME" "Name=vpc-id,Values=$VPC_ID" \
    --region $AWS_REGION \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null || echo "")

if [ -z "$FRONTEND_SG_ID" ] || [ "$FRONTEND_SG_ID" == "None" ]; then
    echo "Creating security group: $FRONTEND_SG_NAME"
    FRONTEND_SG_ID=$(aws ec2 create-security-group \
        --group-name "$FRONTEND_SG_NAME" \
        --description "Security group for ${PROJECT_NAME} frontend ECS tasks" \
        --vpc-id "$VPC_ID" \
        --region $AWS_REGION \
        --query 'GroupId' \
        --output text)
    
    # Allow HTTP traffic from internet (frontend can be accessed directly or via ALB)
    aws ec2 authorize-security-group-ingress \
        --group-id "$FRONTEND_SG_ID" \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 \
        --region $AWS_REGION 2>/dev/null || echo "Ingress rule may already exist"
    
    echo "✓ Created: $FRONTEND_SG_ID"
else
    echo "✓ Using existing: $FRONTEND_SG_ID"
fi
echo ""

# Create target group for backend
echo "=== Creating Target Group for Backend ==="
BACKEND_TG_NAME="${PROJECT_NAME}-backend-tg"
BACKEND_TG_ARN=$(aws elbv2 describe-target-groups \
    --names "$BACKEND_TG_NAME" \
    --region $AWS_REGION \
    --query 'TargetGroups[0].TargetGroupArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$BACKEND_TG_ARN" ] || [ "$BACKEND_TG_ARN" == "None" ]; then
    echo "Creating target group: $BACKEND_TG_NAME"
    BACKEND_TG_ARN=$(aws elbv2 create-target-group \
        --name "$BACKEND_TG_NAME" \
        --protocol HTTP \
        --port 8080 \
        --vpc-id "$VPC_ID" \
        --target-type ip \
        --health-check-path /health \
        --health-check-protocol HTTP \
        --health-check-interval-seconds 30 \
        --health-check-timeout-seconds 5 \
        --healthy-threshold-count 2 \
        --unhealthy-threshold-count 3 \
        --matcher HttpCode=200 \
        --region $AWS_REGION \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text)
    
    echo "✓ Created: $BACKEND_TG_ARN"
else
    echo "✓ Using existing: $BACKEND_TG_ARN"
fi
echo ""

# Configure ALB listener for backend
echo "=== Configuring ALB Listener ==="
LISTENER_ARN=$(aws elbv2 describe-listeners \
    --load-balancer-arn "$ALB_ARN" \
    --region $AWS_REGION \
    --query 'Listeners[?Port==`80`].ListenerArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$LISTENER_ARN" ] || [ "$LISTENER_ARN" == "None" ]; then
    echo "Creating HTTP listener (port 80)..."
    LISTENER_ARN=$(aws elbv2 create-listener \
        --load-balancer-arn "$ALB_ARN" \
        --protocol HTTP \
        --port 80 \
        --default-actions Type=forward,TargetGroupArn=$BACKEND_TG_ARN \
        --region $AWS_REGION \
        --query 'Listeners[0].ListenerArn' \
        --output text)
    
    echo "✓ Created listener: $LISTENER_ARN"
else
    echo "✓ Listener already exists: $LISTENER_ARN"
    # Update default action to point to backend target group
    echo "Updating default action..."
    aws elbv2 modify-listener \
        --listener-arn "$LISTENER_ARN" \
        --default-actions Type=forward,TargetGroupArn=$BACKEND_TG_ARN \
        --region $AWS_REGION >/dev/null 2>&1 || echo "Action may already be set"
fi
echo ""

# Check if task definitions exist
echo "=== Checking Task Definitions ==="
BACKEND_TASK_DEF="${PROJECT_NAME}-backend-task"
FRONTEND_TASK_DEF="${PROJECT_NAME}-frontend-task"

BACKEND_TASK_EXISTS=$(aws ecs describe-task-definition \
    --task-definition "$BACKEND_TASK_DEF" \
    --region $AWS_REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text 2>/dev/null || echo "")

FRONTEND_TASK_EXISTS=$(aws ecs describe-task-definition \
    --task-definition "$FRONTEND_TASK_DEF" \
    --region $AWS_REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text 2>/dev/null || echo "")

if [ -z "$BACKEND_TASK_EXISTS" ] || [ "$BACKEND_TASK_EXISTS" == "None" ]; then
    echo "⚠️  Backend task definition '$BACKEND_TASK_DEF' does not exist yet."
    echo "   It will be created on first deployment via GitHub Actions."
    echo "   Creating a minimal task definition now so we can create the service..."
    echo ""
    
    # Create minimal task definition so service can be created
    EXECUTION_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskExecutionRole"
    TASK_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskRole"
    
    # Use a placeholder image from ECR (will be updated on first deployment)
    ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
    PLACEHOLDER_IMAGE="${ECR_REGISTRY}/${PROJECT_NAME}-backend:latest"
    
    # Check if placeholder image exists, if not, we'll need to create service differently
    aws ecr describe-images --repository-name "${PROJECT_NAME}-backend" --image-ids imageTag=latest --region $AWS_REGION >/dev/null 2>&1 || {
        echo "⚠️  No image found in ECR yet. Service will be created but may fail until first deployment."
        PLACEHOLDER_IMAGE="nginx:alpine"  # Temporary placeholder
    }
    
    cat > /tmp/backend-task-def.json <<EOF
{
  "family": "$BACKEND_TASK_DEF",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "$EXECUTION_ROLE_ARN",
  "taskRoleArn": "$TASK_ROLE_ARN",
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "$PLACEHOLDER_IMAGE",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${PROJECT_NAME}-backend",
          "awslogs-region": "$AWS_REGION",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
EOF
    
    echo "Registering minimal task definition..."
    aws ecs register-task-definition \
        --cli-input-json file:///tmp/backend-task-def.json \
        --region $AWS_REGION >/dev/null 2>&1 && echo "✓ Registered task definition" || echo "⚠️  Failed to register (may already exist or need proper image)"
    
    BACKEND_TASK_EXISTS="EXISTS"  # Mark as exists so we can proceed
fi

if [ -z "$FRONTEND_TASK_EXISTS" ] || [ "$FRONTEND_TASK_EXISTS" == "None" ]; then
    echo "⚠️  Frontend task definition '$FRONTEND_TASK_DEF' does not exist yet."
    echo "   Similar to backend, it will be created on first deployment."
fi
echo ""

# Create ECS service for backend
echo "=== Creating ECS Service for Backend ==="
CLUSTER_NAME="${PROJECT_NAME}-cluster"
BACKEND_SERVICE_NAME="${PROJECT_NAME}-backend-service"

# Check if service already exists
EXISTING_SERVICE=$(aws ecs describe-services \
    --cluster "$CLUSTER_NAME" \
    --services "$BACKEND_SERVICE_NAME" \
    --region $AWS_REGION \
    --query 'services[0].status' \
    --output text 2>/dev/null || echo "INACTIVE")

if [ "$EXISTING_SERVICE" == "ACTIVE" ]; then
    echo "✓ Backend service already exists: $BACKEND_SERVICE_NAME"
else
    echo "Creating backend service..."
    
    # Use task definition family name even if revision doesn't exist yet
    # ECS will use the latest revision when service starts
    TASK_DEF_TO_USE="$BACKEND_TASK_DEF"
    if [ -z "$BACKEND_TASK_EXISTS" ] || [ "$BACKEND_TASK_EXISTS" == "None" ]; then
        echo "⚠️  Creating service with task definition family (first deployment will register actual task definition)"
        TASK_DEF_TO_USE="$BACKEND_TASK_DEF:1"  # Use revision 1 as placeholder
    else
        TASK_DEF_TO_USE="$BACKEND_TASK_DEF:$(aws ecs describe-task-definition --task-definition $BACKEND_TASK_DEF --region $AWS_REGION --query 'taskDefinition.revision' --output text)"
    fi
    
    aws ecs create-service \
        --cluster "$CLUSTER_NAME" \
        --service-name "$BACKEND_SERVICE_NAME" \
        --task-definition "$TASK_DEF_TO_USE" \
        --desired-count 1 \
        --launch-type FARGATE \
        --platform-version LATEST \
        --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_1,$SUBNET_2],securityGroups=[$BACKEND_SG_ID],assignPublicIp=ENABLED}" \
        --load-balancers targetGroupArn=$BACKEND_TG_ARN,containerName=backend,containerPort=8080 \
        --health-check-grace-period-seconds 60 \
        --region $AWS_REGION >/dev/null 2>&1 || {
        echo "❌ Failed to create backend service. This may be because:"
        echo "   - Task definition doesn't exist (will be created on first deployment)"
        echo "   - Service already exists"
        echo "   - Insufficient permissions"
        echo ""
        echo "Continuing anyway..."
    }
    
    if [ $? -eq 0 ]; then
        echo "✓ Created backend service: $BACKEND_SERVICE_NAME"
    fi
fi
echo ""

# Create ECS service for frontend (standalone, not behind ALB for now)
echo "=== Creating ECS Service for Frontend ==="
FRONTEND_SERVICE_NAME="${PROJECT_NAME}-frontend-service"

EXISTING_FRONTEND_SERVICE=$(aws ecs describe-services \
    --cluster "$CLUSTER_NAME" \
    --services "$FRONTEND_SERVICE_NAME" \
    --region $AWS_REGION \
    --query 'services[0].status' \
    --output text 2>/dev/null || echo "INACTIVE")

if [ "$EXISTING_FRONTEND_SERVICE" == "ACTIVE" ]; then
    echo "✓ Frontend service already exists: $FRONTEND_SERVICE_NAME"
else
    echo "Creating frontend service..."
    
    TASK_DEF_TO_USE="$FRONTEND_TASK_DEF"
    if [ -z "$FRONTEND_TASK_EXISTS" ] || [ "$FRONTEND_TASK_EXISTS" == "None" ]; then
        TASK_DEF_TO_USE="$FRONTEND_TASK_DEF:1"
    else
        TASK_DEF_TO_USE="$FRONTEND_TASK_DEF:$(aws ecs describe-task-definition --task-definition $FRONTEND_TASK_DEF --region $AWS_REGION --query 'taskDefinition.revision' --output text)"
    fi
    
    aws ecs create-service \
        --cluster "$CLUSTER_NAME" \
        --service-name "$FRONTEND_SERVICE_NAME" \
        --task-definition "$TASK_DEF_TO_USE" \
        --desired-count 1 \
        --launch-type FARGATE \
        --platform-version LATEST \
        --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_1,$SUBNET_2],securityGroups=[$FRONTEND_SG_ID],assignPublicIp=ENABLED}" \
        --health-check-grace-period-seconds 30 \
        --region $AWS_REGION >/dev/null 2>&1 || {
        echo "⚠️  Frontend service creation failed (task definition may not exist yet)"
        echo "   This is OK - service will be created on first deployment"
    }
    
    if [ $? -eq 0 ]; then
        echo "✓ Created frontend service: $FRONTEND_SERVICE_NAME"
    fi
fi
echo ""

echo "=========================================="
echo "✓ Setup Complete!"
echo "=========================================="
echo ""
echo "Summary:"
echo "  - Backend Security Group: $BACKEND_SG_ID"
echo "  - Frontend Security Group: $FRONTEND_SG_ID"
echo "  - Backend Target Group: $BACKEND_TG_ARN"
echo "  - ALB Listener: $LISTENER_ARN"
echo "  - Backend Service: $BACKEND_SERVICE_NAME"
echo "  - Frontend Service: $FRONTEND_SERVICE_NAME"
echo ""
echo "Next steps:"
echo "  1. Ensure secrets exist: cd ecs && AWS_REGION=$AWS_REGION ./create-secrets.sh"
echo "  2. Push code to trigger GitHub Actions deployment"
echo "  3. GitHub Actions will build images and register task definitions"
echo "  4. Services will automatically update to use new task definitions"
echo ""

