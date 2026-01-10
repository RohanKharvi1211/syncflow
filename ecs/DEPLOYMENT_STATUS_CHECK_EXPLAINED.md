# Deployment Status Check - What It Does

## Overview

The "Check deployment status" step in the GitHub Actions workflow provides **visibility** into the ECS deployment without blocking the workflow. It shows you what's happening with your deployment so you can monitor it separately.

## What The Step Does

### Location in Workflow
```yaml
- name: Check deployment status (backend)
  if: steps.check-service-backend.outputs.service_exists == 'true'
  timeout-minutes: 2
  run: |
    # Checks deployment status
    # Shows recent events
  continue-on-error: true
```

### What It Checks

#### 1. Service Status Overview
```bash
aws ecs describe-services \
  --cluster syncflow-cluster \
  --services syncflow-backend-service \
  --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount,...}'
```

**Output shows:**
- **Status**: Current service status (e.g., `ACTIVE`, `DRAINING`)
- **Running**: Number of tasks currently running
- **Desired**: Number of tasks that should be running
- **Deployments**: Current deployment status with running/desired counts

**Example output:**
```
Status: ACTIVE
Running: 1
Desired: 1
Deployments:
  - Status: PRIMARY
    RunningCount: 1
    DesiredCount: 1
```

#### 2. Recent Service Events
```bash
aws ecs describe-services \
  --query 'services[0].events[:5].{Message:message,Time:createdAt}'
```

**Shows the last 5 events**, such as:
- `(service syncflow-backend-service) has started 1 tasks: task abc123`
- `(service syncflow-backend-service) registered 1 instances in (target-group)`
- `(service syncflow-backend-service) deployment ecs-svc/123 completed`

### Key Features

#### ✅ Non-Blocking
- **`continue-on-error: true`**: If the status check fails, the workflow still succeeds
- **`timeout-minutes: 2`**: Quick check, won't hang the workflow
- Doesn't wait for deployment to complete - just checks current state

#### ✅ Visibility
- Shows you what's happening **right now**
- Displays recent events so you can see deployment progress
- Helps identify issues early (e.g., tasks failing to start)

#### ✅ No Impact on Deployment
- This is a **read-only** operation
- Doesn't affect the actual deployment
- Just queries AWS to see status

## Why We Don't Wait for Stability

### The Problem with `wait-for-service-stability: true`

If we waited for service stability:
- ❌ Workflow hangs if health checks are failing
- ❌ Takes 10-30+ minutes if deployment is slow
- ❌ Blocks CI/CD pipeline
- ❌ Hard to debug (just sits there waiting)

### The Solution: Status Check Instead

With status check:
- ✅ Workflow completes in 2 minutes (quick check)
- ✅ Shows deployment status immediately
- ✅ Doesn't block if health checks are failing
- ✅ You can monitor deployment separately

## What You See

### Successful Deployment Status:
```
Checking deployment status...
┌─────────────┬──────────┬──────────┐
│ Status      │ Running  │ Desired  │
├─────────────┼──────────┼──────────┤
│ ACTIVE      │ 1        │ 1        │
└─────────────┴──────────┴──────────┘

Deployments:
┌──────────┬──────────────┬──────────────┐
│ Status   │ RunningCount │ DesiredCount │
├──────────┼──────────────┼──────────────┤
│ PRIMARY  │ 1            │ 1            │
└──────────┴──────────────┴──────────────┘

Recent service events:
┌─────────────────────────────────────┬──────────────────────────┐
│ Message                             │ Time                     │
├─────────────────────────────────────┼──────────────────────────┤
│ (service ...) has started 1 tasks   │ 2026-01-10T14:00:00.000Z │
│ (service ...) registered in target  │ 2026-01-10T14:00:30.000Z │
└─────────────────────────────────────┴──────────────────────────┘
```

### Unhealthy Deployment:
```
Checking deployment status...
┌─────────────┬──────────┬──────────┐
│ Status      │ Running  │ Desired  │
├─────────────┼──────────┼──────────┤
│ ACTIVE      │ 0        │ 1        │  ← Tasks keep failing
└─────────────┴──────────┴──────────┘

Recent service events:
┌─────────────────────────────────────┬──────────────────────────┐
│ Message                             │ Time                     │
├─────────────────────────────────────┼──────────────────────────┤
│ (service ...) task stopped          │ 2026-01-10T14:00:45.000Z │
│ Task failed container health checks │ 2026-01-10T14:00:30.000Z │  ← Issue!
│ (service ...) has started 1 tasks   │ 2026-01-10T14:00:00.000Z │
└─────────────────────────────────────┴──────────────────────────┘
```

## How to Use This Information

### After CI/CD Completes:

1. **Check the workflow logs** - See the status check output
2. **If deployment looks healthy** - Monitor in AWS Console
3. **If deployment looks unhealthy** - Check CloudWatch logs

### Common Issues You Can Spot:

#### Issue 1: Tasks Not Starting
```
Running: 0
Desired: 1
```
**Action**: Check CloudWatch logs, task definition, IAM permissions

#### Issue 2: Health Check Failures
```
Events show: "Task failed container health checks"
```
**Action**: 
- Check if `/health` endpoint is working
- Verify health check command in task definition
- Check application logs for startup errors

#### Issue 3: Deployment Stuck
```
Deployments: Multiple deployments showing
RunningCount: 0 for primary deployment
```
**Action**: Check service events, verify security groups, check target group health

## Monitoring After Deployment

### Option 1: AWS Console
- Go to **ECS → Clusters → syncflow-cluster → Services → syncflow-backend-service**
- Check **Tasks** tab to see running tasks
- Check **Events** tab for recent events
- Check **Logs** tab for CloudWatch logs

### Option 2: AWS CLI
```bash
cd ecs
./check-deployment-status.sh
```

### Option 3: CloudWatch Logs
```bash
aws logs tail /ecs/syncflow-backend --follow
```

## Summary

**The "Check deployment status" step:**
- ✅ Provides **visibility** into deployment status
- ✅ **Doesn't block** the workflow
- ✅ Shows **recent events** for debugging
- ✅ **Read-only** - doesn't affect deployment
- ✅ **Quick** - completes in ~2 minutes
- ✅ **Non-critical** - workflow continues even if check fails

**It's like a "quick peek" at your deployment status** without waiting for it to complete!

