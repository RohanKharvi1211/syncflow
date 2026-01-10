# Deployment Workflow Explanation

## One-Time Setup (Run Once)

### Initial Setup Steps:
```bash
# 1. Create infrastructure (VPC, ECS cluster, etc.) - Terraform
cd ecs/terraform
terraform apply

# 2. Create ECS services (ONLY ONCE)
cd ..
AWS_REGION=ap-south-1 ./create-ecs-services.sh

# 3. Create secrets in AWS Secrets Manager (ONLY ONCE)
AWS_REGION=ap-south-1 ./create-secrets.sh

# 4. Create RDS database (ONLY ONCE)
AWS_REGION=ap-south-1 DB_PASSWORD=<password> ./create-rds.sh
```

## Regular Deployments (Automatic)

### Every Time You Push Code:

**GitHub Actions automatically:**
1. ✅ Builds Docker images
2. ✅ Pushes images to ECR
3. ✅ Registers new task definitions
4. ✅ Updates ECS services with new task definitions
5. ✅ Waits for service stability

**You DON'T need to:**
- ❌ Run `create-ecs-services.sh` again
- ❌ Manually update task definitions
- ❌ Manually restart services
- ❌ Manually deploy anything

## How It Works

### First Deployment Flow:
```
1. Services don't exist yet
   ↓
2. GitHub Actions builds & pushes images
   ↓
3. GitHub Actions registers task definitions
   ↓
4. GitHub Actions sees services don't exist
   ↓
5. Skips deployment, logs message: "Service doesn't exist"
   ↓
6. YOU run: ./create-ecs-services.sh (ONE TIME)
   ↓
7. Services are created and use existing task definitions
```

### Subsequent Deployments Flow:
```
1. You push code to GitHub
   ↓
2. GitHub Actions triggers automatically
   ↓
3. Builds & pushes new images
   ↓
4. Registers new task definition with new image
   ↓
5. GitHub Actions checks: Service exists? YES
   ↓
6. Updates service with new task definition
   ↓
7. ECS automatically performs rolling update:
   - Starts new task with new image
   - Waits for health checks to pass
   - Stops old task
   ↓
8. Deployment complete!
```

## When to Re-run `create-ecs-services.sh`

### You ONLY need to run it again if:
- ❌ Services were deleted (shouldn't happen normally)
- ❌ You're setting up a new environment (dev, staging, prod)
- ❌ You need to recreate services due to configuration issues

### You DON'T need to run it if:
- ✅ Just pushing code updates
- ✅ Updating application code
- ✅ Changing environment variables (use Secrets Manager)
- ✅ Normal deployments

## Summary

| Action | Frequency | Who Does It |
|--------|-----------|-------------|
| Create ECS services | **ONCE** | You (manual) |
| Create infrastructure | **ONCE** | Terraform |
| Create secrets | **ONCE** | You (manual) |
| Create RDS | **ONCE** | You (manual) |
| Build & push images | **Every push** | GitHub Actions |
| Register task definitions | **Every push** | GitHub Actions |
| Update services | **Every push** | GitHub Actions |
| Deploy to ECS | **Every push** | GitHub Actions |

## Quick Reference

**First Time Setup:**
```bash
# Run these ONCE
./setup-infrastructure.sh      # Create VPC, cluster, etc.
./create-ecs-services.sh       # Create ECS services
./create-secrets.sh            # Create AWS Secrets
./create-rds.sh                # Create database
```

**Regular Deployments:**
```bash
# Just push code - GitHub Actions does the rest!
git add .
git commit -m "Update application"
git push origin main

# That's it! Check status after a few minutes:
./check-deployment-status.sh
```

**Check Deployment Status:**
```bash
# After pushing, monitor deployment:
AWS_REGION=ap-south-1 ./check-deployment-status.sh

# Or watch GitHub Actions:
# Go to: GitHub → Actions tab → See workflow running
```

## Troubleshooting

### If GitHub Actions says "Service doesn't exist":
- ✅ This is normal on first deployment
- ✅ Run: `./create-ecs-services.sh` (one time)
- ✅ Re-run the workflow or wait for next push

### If deployment fails after services exist:
- Check CloudWatch logs for errors
- Verify task definition registered correctly
- Check service events in AWS Console
- Verify secrets are configured correctly

### If you need to recreate services:
```bash
# Delete services first (optional, usually not needed)
aws ecs delete-service --cluster syncflow-cluster \
    --service syncflow-backend-service --force --region ap-south-1
aws ecs delete-service --cluster syncflow-cluster \
    --service syncflow-frontend-service --force --region ap-south-1

# Then recreate
./create-ecs-services.sh
```

## Best Practices

1. **Never manually modify services** - Let GitHub Actions handle deployments
2. **Update secrets via AWS Console/CLI** - Don't recreate services
3. **Monitor deployments** - Check status after each push
4. **Use staging environment** - Test before production
5. **Keep infrastructure as code** - Use Terraform for infrastructure changes

