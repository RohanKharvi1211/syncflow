# Fixed: Database SSL Connection Error

## Problem
```
panic: db initialization failed: failed to connect to `user=postgres database=syncflow`: 
172.31.47.137:5432 (syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com): 
server error: FATAL: no pg_hba.conf entry for host "172.31.13.90", user "postgres", 
database "syncflow", no encryption (SQLSTATE 28000)
```

## Root Cause
- RDS PostgreSQL requires SSL/encryption connections
- Backend was connecting with `sslmode=disable` (no encryption)
- RDS `pg_hba.conf` rejects non-encrypted connections for security

## Solution Applied

### 1. Updated Database Connection Code

**Files updated:**
- ✅ `backend/internal/config/database.go`
- ✅ `backend/cmd/app/app.go`

**Changes:**
- Changed from hardcoded `sslmode=disable`
- Now automatically detects if connecting to RDS (non-localhost)
- Uses `sslmode=require` for RDS connections
- Uses `sslmode=disable` for local development

### 2. Fixed IAM Permissions

- ✅ Created Secrets Manager policy for ECS task execution role
- ✅ Added permissions: `secretsmanager:GetSecretValue`, `secretsmanager:DescribeSecret`
- ✅ Added KMS decrypt permissions for encrypted secrets

### 3. Updated Task Definition

- ✅ Registered new task definition with secrets from AWS Secrets Manager
- ✅ Task definition now includes: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, DB_PORT, JWT_SECRET

## Next Steps

### 1. Commit and Push Code Changes

```bash
git add backend/internal/config/database.go backend/cmd/app/app.go
git commit -m "Fix: Enable SSL for RDS database connections"
git push origin main
```

### 2. Monitor Deployment

After pushing, GitHub Actions will:
- Build new Docker image with SSL fix
- Push to ECR
- Register new task definition
- Update ECS service automatically

**Or manually trigger deployment:**
```bash
# The task definition is already updated, just force new deployment:
aws ecs update-service \
    --cluster syncflow-cluster \
    --service syncflow-backend-service \
    --force-new-deployment \
    --region ap-south-1
```

### 3. Verify Connection

After deployment, check logs:
```bash
aws logs tail /ecs/syncflow-backend \
    --region ap-south-1 \
    --follow \
    --format short
```

**Expected:**
- ✅ "Database connected successfully"
- ✅ No SSL errors
- ✅ Application starts normally

### 4. Test Health Endpoint

```bash
ALB_DNS=$(aws elbv2 describe-load-balancers \
    --region ap-south-1 \
    --query "LoadBalancers[?contains(LoadBalancerName, 'syncflow')].DNSName" \
    --output text | head -1)

curl http://$ALB_DNS/health
```

**Expected Response:**
```json
{"status":"ok"}
```

## Code Changes Summary

### Before:
```go
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", ...)
```

### After:
```go
sslMode := "disable" // Default for local development
if cfg.Database.Host != "localhost" && cfg.Database.Host != "127.0.0.1" {
    sslMode = "require" // Use SSL for RDS and other remote databases
}
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", ..., sslMode)
```

## Status

✅ **FIXED:**
- Database SSL connection code updated
- IAM permissions for Secrets Manager added
- Task definition updated with secrets

⏳ **IN PROGRESS:**
- Code needs to be committed and pushed
- New Docker image needs to be built and deployed

## Verification Checklist

After deployment:
- [ ] Task status is RUNNING
- [ ] Container health is HEALTHY
- [ ] Logs show "Database connected successfully"
- [ ] No SSL/encryption errors in logs
- [ ] Health endpoint returns `{"status":"ok"}`
- [ ] ALB target health is healthy

## Troubleshooting

### If Still Getting SSL Errors:

1. **Check if code was deployed:**
   ```bash
   # Verify task definition has latest image
   aws ecs describe-task-definition \
       --task-definition syncflow-backend-task \
       --region ap-south-1 \
       --query 'taskDefinition.containerDefinitions[0].image' \
       --output text
   ```

2. **Check if RDS requires SSL:**
   ```bash
   # RDS by default requires SSL, but we can verify
   aws rds describe-db-instances \
       --db-instance-identifier syncflow-db \
       --region ap-south-1 \
       --query 'DBInstances[0].{Endpoint:Endpoint.Address,Engine:Engine}' \
       --output table
   ```

3. **Test connection from local machine (if needed):**
   ```bash
   # Get RDS endpoint
   DB_HOST=$(aws secretsmanager get-secret-value --secret-id syncflow/database/host --region ap-south-1 --query 'SecretString' --output text)
   DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id syncflow/database/password --region ap-south-1 --query 'SecretString' --output text)
   
   # Test with SSL (if psql is available)
   PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U postgres -d syncflow -c "SELECT version();"
   ```

### If Getting "Permission Denied" Errors:

1. **Verify RDS Security Group allows ECS traffic:**
   ```bash
   # Get backend task security group
   BACKEND_SG=$(aws ecs describe-services \
       --cluster syncflow-cluster \
       --services syncflow-backend-service \
       --region ap-south-1 \
       --query 'services[0].networkConfiguration.awsvpcConfiguration.securityGroups[0]' \
       --output text)
   
   # Check RDS SG allows this SG
   aws ec2 describe-security-groups \
       --group-ids sg-0d156ead6b2f3fa37 \
       --region ap-south-1 \
       --query 'SecurityGroups[0].IpPermissions[*].{Port:FromPort,Source:UserIdGroupPairs[0].GroupId}' \
       --output table
   ```

2. **Update RDS Security Group if needed:**
   ```bash
   # Allow backend SG to connect to RDS on port 5432
   aws ec2 authorize-security-group-ingress \
       --group-id sg-0d156ead6b2f3fa37 \
       --protocol tcp \
       --port 5432 \
       --source-group $BACKEND_SG \
       --region ap-south-1
   ```

---

**Current Status:** ✅ Code fixed, ready to deploy. Commit and push to trigger deployment.

