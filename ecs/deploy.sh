#!/bin/bash
# Manual deployment script for ECS
# Usage: ./deploy.sh [backend|frontend|all]

set -e

AWS_REGION=${AWS_REGION:-us-east-1}
ECR_REPOSITORY_BACKEND=${ECR_REPOSITORY_BACKEND:-syncflow-backend}
ECR_REPOSITORY_FRONTEND=${ECR_REPOSITORY_FRONTEND:-syncflow-frontend}
ECS_CLUSTER=${ECS_CLUSTER:-syncflow-cluster}
ECS_SERVICE_BACKEND=${ECS_SERVICE_BACKEND:-syncflow-backend-service}
ECS_SERVICE_FRONTEND=${ECS_SERVICE_FRONTEND:-syncflow-frontend-service}
ECS_TASK_DEFINITION_BACKEND=${ECS_TASK_DEFINITION_BACKEND:-syncflow-backend-task}
ECS_TASK_DEFINITION_FRONTEND=${ECS_TASK_DEFINITION_FRONTEND:-syncflow-frontend-task}

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

DEPLOY_TARGET=${1:-all}

echo "AWS Account ID: $AWS_ACCOUNT_ID"
echo "AWS Region: $AWS_REGION"
echo "ECR Registry: $ECR_REGISTRY"
echo "Deploy Target: $DEPLOY_TARGET"

# Login to ECR
echo "Logging in to ECR..."
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

deploy_backend() {
    echo "=== Deploying Backend ==="
    
    # Build and push backend image
    IMAGE_TAG=$(git rev-parse HEAD)
    BACKEND_IMAGE="${ECR_REGISTRY}/${ECR_REPOSITORY_BACKEND}:${IMAGE_TAG}"
    
    echo "Building backend image..."
    docker build -t $BACKEND_IMAGE -f backend/Dockerfile backend/
    docker tag $BACKEND_IMAGE "${ECR_REGISTRY}/${ECR_REPOSITORY_BACKEND}:latest"
    
    echo "Pushing backend image..."
    docker push $BACKEND_IMAGE
    docker push "${ECR_REGISTRY}/${ECR_REPOSITORY_BACKEND}:latest"
    
    # Update task definition
    echo "Updating task definition..."
    TASK_DEF=$(aws ecs describe-task-definition \
        --task-definition $ECS_TASK_DEFINITION_BACKEND \
        --query taskDefinition \
        --region $AWS_REGION 2>/dev/null || echo "{}")
    
    if [ "$TASK_DEF" = "{}" ]; then
        echo "Task definition not found. Please create it first using ecs/task-definition-backend.json"
        exit 1
    fi
    
    # Create new task definition revision
    NEW_TASK_DEF=$(echo "$TASK_DEF" | \
        jq --arg IMAGE "$BACKEND_IMAGE" \
           '.containerDefinitions[0].image = $IMAGE | .containerDefinitions[0].name = "backend" | del(.taskDefinitionArn) | del(.revision) | del(.status) | del(.requiresAttributes) | del(.compatibilities) | del(.registeredAt) | del(.registeredBy)')
    
    echo "$NEW_TASK_DEF" > /tmp/task-def-backend.json
    
    # Register new task definition
    NEW_TASK_DEF_ARN=$(aws ecs register-task-definition \
        --cli-input-json file:///tmp/task-def-backend.json \
        --region $AWS_REGION \
        --query 'taskDefinition.taskDefinitionArn' \
        --output text)
    
    echo "New task definition registered: $NEW_TASK_DEF_ARN"
    
    # Update service
    echo "Updating ECS service..."
    aws ecs update-service \
        --cluster $ECS_CLUSTER \
        --service $ECS_SERVICE_BACKEND \
        --task-definition $NEW_TASK_DEF_ARN \
        --region $AWS_REGION \
        --force-new-deployment
    
    echo "Backend deployment initiated. Waiting for service to stabilize..."
    aws ecs wait services-stable \
        --cluster $ECS_CLUSTER \
        --services $ECS_SERVICE_BACKEND \
        --region $AWS_REGION
    
    echo "✓ Backend deployed successfully!"
}

deploy_frontend() {
    echo "=== Deploying Frontend ==="
    
    # Build and push frontend image
    IMAGE_TAG=$(git rev-parse HEAD)
    FRONTEND_IMAGE="${ECR_REGISTRY}/${ECR_REPOSITORY_FRONTEND}:${IMAGE_TAG}"
    
    echo "Building frontend image..."
    docker build \
        --build-arg VITE_API_BASE_URL=${VITE_API_BASE_URL:-https://api.yourdomain.com} \
        -t $FRONTEND_IMAGE \
        -f frontend/Dockerfile frontend/
    docker tag $FRONTEND_IMAGE "${ECR_REGISTRY}/${ECR_REPOSITORY_FRONTEND}:latest"
    
    echo "Pushing frontend image..."
    docker push $FRONTEND_IMAGE
    docker push "${ECR_REGISTRY}/${ECR_REPOSITORY_FRONTEND}:latest"
    
    # Update task definition
    echo "Updating task definition..."
    TASK_DEF=$(aws ecs describe-task-definition \
        --task-definition $ECS_TASK_DEFINITION_FRONTEND \
        --query taskDefinition \
        --region $AWS_REGION 2>/dev/null || echo "{}")
    
    if [ "$TASK_DEF" = "{}" ]; then
        echo "Task definition not found. Please create it first using ecs/task-definition-frontend.json"
        exit 1
    fi
    
    # Create new task definition revision
    NEW_TASK_DEF=$(echo "$TASK_DEF" | \
        jq --arg IMAGE "$FRONTEND_IMAGE" \
           '.containerDefinitions[0].image = $IMAGE | .containerDefinitions[0].name = "frontend" | del(.taskDefinitionArn) | del(.revision) | del(.status) | del(.requiresAttributes) | del(.compatibilities) | del(.registeredAt) | del(.registeredBy)')
    
    echo "$NEW_TASK_DEF" > /tmp/task-def-frontend.json
    
    # Register new task definition
    NEW_TASK_DEF_ARN=$(aws ecs register-task-definition \
        --cli-input-json file:///tmp/task-def-frontend.json \
        --region $AWS_REGION \
        --query 'taskDefinition.taskDefinitionArn' \
        --output text)
    
    echo "New task definition registered: $NEW_TASK_DEF_ARN"
    
    # Update service
    echo "Updating ECS service..."
    aws ecs update-service \
        --cluster $ECS_CLUSTER \
        --service $ECS_SERVICE_FRONTEND \
        --task-definition $NEW_TASK_DEF_ARN \
        --region $AWS_REGION \
        --force-new-deployment
    
    echo "Frontend deployment initiated. Waiting for service to stabilize..."
    aws ecs wait services-stable \
        --cluster $ECS_CLUSTER \
        --services $ECS_SERVICE_FRONTEND \
        --region $AWS_REGION
    
    echo "✓ Frontend deployed successfully!"
}

# Main deployment logic
case $DEPLOY_TARGET in
    backend)
        deploy_backend
        ;;
    frontend)
        deploy_frontend
        ;;
    all)
        deploy_backend
        deploy_frontend
        ;;
    *)
        echo "Usage: $0 [backend|frontend|all]"
        exit 1
        ;;
esac

echo "=== Deployment Complete ==="

