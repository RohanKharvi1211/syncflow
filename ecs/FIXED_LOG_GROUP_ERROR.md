# Fixed: CloudWatch Log Group Error

## Problem
```
ResourceInitializationError: failed to validate logger args: create stream has been retried 1 times: 
failed to create Cloudwatch log stream: operation error CloudWatch Logs: CreateLogStream, 
https response error StatusCode: 400, RequestID: ..., ResourceNotFoundException: 
The specified log group does not exist.
```

## Root Cause
1. **Log groups didn't exist** when the first task started
2. **IAM permissions missing** - ECS task execution role didn't have permissions to create log streams in CloudWatch Logs

## Solution Applied

### 1. Created CloudWatch Log Groups
```bash
# Run once (already done)
AWS_REGION=ap-south-1 ./create-log-groups.sh
```

Created:
- ✅ `/ecs/syncflow-backend`
- ✅ `/ecs/syncflow-frontend`

### 2. Fixed IAM Permissions
```bash
# Run once (already done)
AWS_REGION=ap-south-1 ./fix-ecs-task-execution-role.sh
```

Added permissions to `ecsTaskExecutionRole`:
- ✅ `logs:CreateLogStream`
- ✅ `logs:PutLogEvents`
- ✅ `logs:DescribeLogGroups`
- ✅ `logs:DescribeLogStreams`
- ✅ `logs:CreateLogGroup` (for future log groups)

### 3. Triggered New Deployment
```bash
# Force new deployment to pick up permissions
aws ecs update-service \
    --cluster syncflow-cluster \
    --service syncflow-backend-service \
    --force-new-deployment \
    --region ap-south-1
```

## Verification

### Check if Task is Running:
```bash
aws ecs describe-services \
    --cluster syncflow-cluster \
    --services syncflow-backend-service \
    --region ap-south-1 \
    --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount}' \
    --output table
```

**Expected:** Running count should match Desired count (usually `1`)

### Check Task Status:
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
    --query 'tasks[0].{Status:lastStatus,Health:healthStatus}' \
    --output table
```

**Expected:** Status should be `RUNNING`

### Check Logs (should work now):
```bash
# View recent logs
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --since 5m \
    --format short

# Watch logs in real-time
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --follow \
    --format short
```

## Prevention for Future

### Terraform Configuration
The log groups should be created via Terraform (already defined in `ecs/terraform/main.tf`):

```terraform
resource "aws_cloudwatch_log_group" "backend" {
  name              = "/ecs/syncflow-backend"
  retention_in_days = 7
}
```

### IAM Role Setup
The ECS task execution role should have CloudWatch Logs permissions. The fix script ensures this, but for future reference:

**Required IAM Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/ecs/syncflow-*:*"
    }
  ]
}
```

## Status

✅ **FIXED:**
- Log groups created
- IAM permissions added
- New deployment triggered

⏳ **In Progress:**
- Waiting for new task to start
- IAM changes propagating (can take 30-60 seconds)

## Next Steps

1. **Wait 1-2 minutes** for IAM changes to propagate
2. **Check service status** again:
   ```bash
   cd ecs
   AWS_REGION=ap-south-1 ./check-deployment-status.sh
   ```
3. **If task still fails**, check:
   - Service events for new errors
   - CloudWatch logs (if any were created)
   - Task definition execution role ARN is correct

## Troubleshooting

### If Task Still Fails After Fix:

1. **Check IAM role attached to task:**
   ```bash
   aws ecs describe-task-definition \
       --task-definition syncflow-backend-task \
       --region ap-south-1 \
       --query 'taskDefinition.executionRoleArn' \
       --output text
   ```

2. **Verify role has correct policy:**
   ```bash
   aws iam list-attached-role-policies \
       --role-name ecsTaskExecutionRole \
       --region ap-south-1
   ```

3. **Check recent service events:**
   ```bash
   aws ecs describe-services \
       --cluster syncflow-cluster \
       --services syncflow-backend-service \
       --region ap-south-1 \
       --query 'services[0].events[:5].{Message:message,Time:createdAt}' \
       --output table
   ```

### Common Issues:

- **IAM changes take time to propagate** - Wait 30-60 seconds
- **Old task definition cached** - Force new deployment
- **Log group name mismatch** - Verify task definition log configuration
- **Insufficient permissions** - Verify policy is attached to role

