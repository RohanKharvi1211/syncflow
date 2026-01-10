# How to Check if ECS Instance is Running in AWS

## 🎯 Quick Check Methods

### 1. AWS Console (Web UI)

#### **ECS Dashboard**
1. Go to **AWS Console** → **ECS** (Elastic Container Service)
2. Click on **Clusters** in the left sidebar
3. Click on your cluster: **`syncflow-cluster`**
4. You'll see:
   - **Services** tab: Shows your services (backend, frontend)
   - **Tasks** tab: Shows running tasks (containers)
   - **Metrics** tab: Shows CPU, memory usage

**What to look for:**
- ✅ **Services Status**: Should show `ACTIVE`
- ✅ **Running Count**: Should match **Desired Count** (usually `1`)
- ✅ **Tasks Status**: Should show `RUNNING` (green)
- ✅ **Health Status**: Should show `HEALTHY` (if health checks enabled)

#### **Service Details**
1. In the cluster, click on **Services** tab
2. Click on **`syncflow-backend-service`**
3. Check:
   - **Status**: `ACTIVE`
   - **Running count**: `1` (or your desired count)
   - **Desired count**: `1`
   - **Task definition**: Latest version
   - **Events**: Recent events showing successful deployments

#### **Task Details**
1. In the cluster, click on **Tasks** tab
2. Click on a running task
3. Check:
   - **Last status**: `RUNNING`
   - **Desired status**: `RUNNING`
   - **Health status**: `HEALTHY` (if configured)
   - **Started at**: Recent timestamp
   - **Containers**: Should show `RUNNING` status

#### **CloudWatch Logs**
1. Go to **CloudWatch** → **Log groups**
2. Look for: `/ecs/syncflow-backend`
3. Click on it to see recent logs
4. Check for:
   - Application startup messages
   - Database connection logs
   - Any error messages

#### **Load Balancer (ALB)**
1. Go to **EC2** → **Load Balancers**
2. Find your ALB: **`syncflow-backend-alb`**
3. Click on it → **Target Groups** tab
4. Click on the target group
5. Check **Targets** tab:
   - **Status**: Should show `healthy` (green)
   - **Health checks**: Should be passing

---

## 🔧 AWS CLI Commands

### Quick Status Check

```bash
# Check cluster status
aws ecs describe-clusters \
    --clusters syncflow-cluster \
    --region ap-south-1 \
    --query 'clusters[0].{Status:status,ActiveServices:activeServicesCount,RunningTasks:runningTasksCount}' \
    --output table
```

### Check Service Status

```bash
# Backend service status
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-backend-service \
    --region ap-south-1 \
    --query 'services[0].{Name:serviceName,Status:status,Running:runningCount,Desired:desiredCount,TaskDefinition:taskDefinition}' \
    --output table
```

**Expected Output:**
```
Name: syncflow-backend-service
Status: ACTIVE
Running: 1
Desired: 1
TaskDefinition: syncflow-backend:1 (or latest version)
```

### Check Running Tasks

```bash
# List all tasks
aws ecs list-tasks \
    --cluster syncflow-cluster \
    --service-name syncflow-backend-service \
    --region ap-south-1 \
    --output table
```

### Check Task Details

```bash
# Get task ARN
TASK_ARN=$(aws ecs list-tasks \
    --cluster syncflow-cluster \
    --service-name syncflow-backend-service \
    --region ap-south-1 \
    --query 'taskArns[0]' \
    --output text)

# Check task status
aws ecs describe-tasks \
    --cluster syncflow-cluster \
    --tasks $TASK_ARN \
    --region ap-south-1 \
    --query 'tasks[0].{Status:lastStatus,DesiredStatus:desiredStatus,HealthStatus:healthStatus,StartedAt:startedAt}' \
    --output table
```

**Expected Output:**
```
Status: RUNNING
DesiredStatus: RUNNING
HealthStatus: HEALTHY (or null if not configured)
StartedAt: 2026-01-XX... (recent timestamp)
```

### Check Container Status

```bash
TASK_ARN=$(aws ecs list-tasks \
    --cluster syncflow-cluster \
    --service-name syncflow-backend-service \
    --region ap-south-1 \
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
```
Name: backend
Status: RUNNING
Health: HEALTHY (or null)
ExitCode: null (not exited)
```

### Check ALB Target Health

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

# Check target health
aws elbv2 describe-target-health \
    --target-group-arn $TG_ARN \
    --region ap-south-1 \
    --output table
```

**Expected Output:**
```
Target: <ip-address>:8080
Port: 8080
Health: healthy
```

### Check Recent Logs

```bash
# View last 50 log lines
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --since 10m \
    --format short

# Watch logs in real-time
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --follow \
    --format short
```

### Check Service Events

```bash
# Recent service events (shows deployment history)
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
- ⚠️ Any errors like `unable to pull image`, `task failed to start`, etc.

---

## ✅ Health Indicators

### **Healthy Instance:**
- ✅ Service Status: `ACTIVE`
- ✅ Running Count = Desired Count
- ✅ Task Status: `RUNNING`
- ✅ Container Status: `RUNNING`
- ✅ ALB Target Health: `healthy`
- ✅ Health endpoint returns: `{"status":"ok"}`
- ✅ No errors in CloudWatch logs

### **Unhealthy Instance:**
- ❌ Service Status: `DRAINING` or `INACTIVE`
- ❌ Running Count < Desired Count
- ❌ Task Status: `STOPPED` or `PENDING`
- ❌ Container ExitCode: Not null (container crashed)
- ❌ ALB Target Health: `unhealthy`
- ❌ Errors in CloudWatch logs

---

## 🧪 Test Application Health

### Test Backend Health Endpoint

```bash
# Get ALB DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].DNSName" \
    --output text | head -1)

# Test health endpoint
curl http://$ALB_DNS/health

# Expected response:
# {"status":"ok"}
```

### Test API Endpoint

```bash
# Test a simple API endpoint
curl http://$ALB_DNS/api/health

# Or test with authentication
curl -H "Authorization: Bearer YOUR_TOKEN" http://$ALB_DNS/api/pipelines
```

---

## 🐛 Troubleshooting

### If Service Shows "DRAINING" or "INACTIVE"
1. Check **Service Events** for error messages
2. Check **CloudWatch Logs** for application errors
3. Verify **Task Definition** is correct
4. Check if **ECR image** exists and is accessible

### If Tasks Keep Stopping
1. Check **CloudWatch Logs** for container errors
2. Check **Task Definition** resource limits (CPU/memory)
3. Verify **Environment Variables** and **Secrets** are correct
4. Check **Security Groups** allow necessary traffic
5. Verify **Database connection** is working

### If ALB Target Shows "Unhealthy"
1. Check if **health check endpoint** (`/health`) is responding
2. Verify **Security Groups** allow ALB → ECS traffic
3. Check if **container port** matches target group port
4. Review **CloudWatch Logs** for application errors

### If No Tasks Running
1. Check **Service Desired Count** (should be at least `1`)
2. Verify **Task Definition** is registered
3. Check **Service Events** for deployment errors
4. Verify **ECR image** exists and is accessible
5. Check **IAM roles** have necessary permissions

---

## 📊 Quick Status Summary Command

Run this one-liner to get a complete status overview:

```bash
echo "=== Cluster Status ===" && \
aws ecs describe-clusters --clusters syncflow-cluster --region ap-south-1 \
    --query 'clusters[0].{Status:status,Tasks:runningTasksCount}' --output table && \
echo -e "\n=== Service Status ===" && \
aws ecs describe-services --cluster syncflow-cluster \
    --services syncflow-backend-service --region ap-south-1 \
    --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount}' --output table && \
echo -e "\n=== Task Status ===" && \
TASK_ARN=$(aws ecs list-tasks --cluster syncflow-cluster \
    --service-name syncflow-backend-service --region ap-south-1 \
    --query 'taskArns[0]' --output text) && \
[ ! -z "$TASK_ARN" ] && aws ecs describe-tasks --cluster syncflow-cluster \
    --tasks $TASK_ARN --region ap-south-1 \
    --query 'tasks[0].{Status:lastStatus,Health:healthStatus}' --output table || \
echo "No tasks found"
```

---

## 🎯 Recommended Check Order

1. **First**: Check service status (is it ACTIVE?)
2. **Second**: Check running tasks (are tasks RUNNING?)
3. **Third**: Check ALB target health (is it healthy?)
4. **Fourth**: Test health endpoint (does it respond?)
5. **Fifth**: Check CloudWatch logs (any errors?)

---

## 📝 Quick Reference

**AWS Console Paths:**
- ECS Services: `ECS → Clusters → syncflow-cluster → Services → syncflow-backend-service`
- Tasks: `ECS → Clusters → syncflow-cluster → Tasks`
- Logs: `CloudWatch → Log groups → /ecs/syncflow-backend`
- ALB: `EC2 → Load Balancers → syncflow-backend-alb → Target Groups`

**Key Metrics to Monitor:**
- Service running count vs desired count
- Task status (should be RUNNING)
- Container health status
- ALB target health
- Application logs for errors

