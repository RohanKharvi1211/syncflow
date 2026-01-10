# Quick Health Check Guide

After deployment, use these commands to verify your ECS services are running correctly.

## Quick Status Check

Run the automated status check script:

```bash
cd ecs
AWS_REGION=ap-south-1 ./check-deployment-status.sh
```

## Manual Checks

### 1. Check ECS Cluster Status

```bash
aws ecs describe-clusters \
    --clusters syncflow-cluster \
    --region ap-south-1 \
    --query 'clusters[0].{Status:status,ActiveServices:activeServicesCount,RunningTasks:runningTasksCount}' \
    --output table
```

**Expected Output:**
- Status: `ACTIVE`
- ActiveServices: `2` (backend + frontend)
- RunningTasks: `2` (or more if you have multiple replicas)

### 2. Check Service Status

**Backend Service:**
```bash
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-backend-service \
    --region ap-south-1 \
    --query 'services[0].{Status:status,RunningCount:runningCount,DesiredCount:desiredCount}' \
    --output table
```

**Frontend Service:**
```bash
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-frontend-service \
    --region ap-south-1 \
    --query 'services[0].{Status:status,RunningCount:runningCount,DesiredCount:desiredCount}' \
    --output table
```

**Expected Output:**
- Status: `ACTIVE`
- RunningCount: Should match DesiredCount (usually `1`)
- DesiredCount: `1` (or your configured replica count)

### 3. Check Running Tasks

```bash
aws ecs list-tasks \
    --cluster syncflow-cluster \
    --region ap-south-1 \
    --output table
```

Then check task details:

```bash
# Get task ARN first
TASK_ARN=$(aws ecs list-tasks \
    --cluster syncflow-cluster \
    --region ap-south-1 \
    --query 'taskArns[0]' \
    --output text)

# Check task status
aws ecs describe-tasks \
    --cluster syncflow-cluster \
    --tasks $TASK_ARN \
    --region ap-south-1 \
    --query 'tasks[0].{Status:lastStatus,DesiredStatus:desiredStatus,HealthStatus:healthStatus}' \
    --output table
```

**Expected Output:**
- Status: `RUNNING`
- DesiredStatus: `RUNNING`
- HealthStatus: `HEALTHY` (if health checks are enabled)

### 4. Check Container Status

```bash
TASK_ARN=$(aws ecs list-tasks \
    --cluster syncflow-cluster \
    --region ap-south-1 \
    --service-name syncflow-backend-service \
    --query 'taskArns[0]' \
    --output text)

aws ecs describe-tasks \
    --cluster syncflow-cluster \
    --tasks $TASK_ARN \
    --region ap-south-1 \
    --query 'tasks[0].containers[*].{Name:name,Status:lastStatus,Health:healthStatus,ExitCode:exitCode}' \
    --output table
```

**Expected Output:**
- Status: `RUNNING`
- Health: `HEALTHY` (if health checks configured)
- ExitCode: Should be `null` (not 0 or any number)

### 5. Check ALB Target Health

Get ALB DNS name:
```bash
aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].DNSName" \
    --output text
```

Check target health:
```bash
# Get ALB ARN
ALB_ARN=$(aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].LoadBalancerArn" \
    --output text | head -1)

# Get target groups
TG_ARN=$(aws elbv2 describe-target-groups \
    --load-balancer-arn $ALB_ARN \
    --region ap-south-1 \
    --query 'TargetGroups[0].TargetGroupArn' \
    --output text)

# Check health
aws elbv2 describe-target-health \
    --target-group-arn $TG_ARN \
    --region ap-south-1 \
    --output table
```

**Expected Output:**
- Health State: `healthy`
- Targets should show `healthy` status

### 6. Check Application Health

**Backend Health Endpoint:**
```bash
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].DNSName" \
    --output text | head -1)

curl http://$ALB_DNS/health
```

**Expected Output:**
```json
{"status":"ok"}
```

### 7. Check CloudWatch Logs

**Backend Logs:**
```bash
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --since 10m \
    --format short
```

**Frontend Logs:**
```bash
aws logs tail /ecs/syncflow-frontend \
    --region ap-south-1 \
    --since 10m \
    --format short
```

**Watch logs in real-time:**
```bash
# Backend
aws logs tail /ecs/syncflow-backend --follow --region ap-south-1

# Frontend
aws logs tail /ecs/syncflow-frontend --follow --region ap-south-1
```

### 8. Check Service Events (for troubleshooting)

```bash
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-backend-service \
    --region ap-south-1 \
    --query 'services[0].events[:10].{Message:message,Time:createdAt}' \
    --output table
```

**Look for:**
- ✅ `(service syncflow-backend-service) has reached a steady state`
- ✅ `(service syncflow-backend-service) has started 1 tasks`
- ⚠️ Errors like `unable to pull image`, `task failed to start`, etc.

## Troubleshooting Common Issues

### Service Status: `DRAINING` or `INACTIVE`
- Check service events for errors
- Verify task definition is correct
- Check if container images exist in ECR

### Tasks: `STOPPED`
- Check CloudWatch logs for container errors
- Verify environment variables and secrets are correct
- Check if database connection is working

### Tasks: `PENDING`
- Check if there are available resources (CPU/memory)
- Verify security groups allow necessary traffic
- Check if subnet has internet access (for pulling images)

### Health Checks: `UNHEALTHY`
- Verify health check endpoint (`/health`) is working
- Check if container is listening on the correct port
- Verify security groups allow health check traffic

### No Tasks Running
- Check service desired count: should be at least `1`
- Verify task definition is registered correctly
- Check service deployment configuration

## Quick Status Summary

```bash
# One-liner to check everything quickly
echo "Cluster:" && \
aws ecs describe-clusters --clusters syncflow-cluster --region ap-south-1 --query 'clusters[0].{Status:status,Tasks:runningTasksCount}' --output table && \
echo -e "\nServices:" && \
aws ecs describe-services --cluster syncflow-cluster --services syncflow-backend-service syncflow-frontend-service --region ap-south-1 --query 'services[*].{Name:serviceName,Status:status,Running:runningCount,Desired:desiredCount}' --output table
```

## Access Your Application

Once everything is healthy:

```bash
# Get ALB DNS
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].DNSName" \
    --output text | head -1)

echo "Your application is available at: http://$ALB_DNS"
echo "Backend API: http://$ALB_DNS/api"
echo "Health Check: http://$ALB_DNS/health"
```

## Next Steps After Successful Deployment

1. ✅ Verify all services are `ACTIVE` and `RUNNING`
2. ✅ Check target health is `healthy`
3. ✅ Test health endpoints
4. ✅ Test application functionality
5. ✅ Monitor CloudWatch logs for any errors
6. ✅ Set up CloudWatch alarms for production monitoring

