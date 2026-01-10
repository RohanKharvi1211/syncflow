# How to Create Database Manually in AWS

## Option 1: AWS RDS Query Editor v2 (Recommended - Easiest)

If Query Editor v2 is available in your AWS account:

1. **Open AWS Console** → Go to **RDS** → Select your database `syncflow-db`
2. **Click on "Query Editor v2"** (if available in the left sidebar)
3. **Connect** to your database using:
   - Database: `postgres` (default database)
   - User: `postgres`
   - Password: (get from Secrets Manager)
   - SSL: Enabled
4. **Run this SQL command:**
   ```sql
   CREATE DATABASE syncflow;
   ```
5. **Click "Run"** or press Ctrl+Enter

**Note:** Query Editor v2 may not be available in all regions or account types. If it's not available, use Option 2 or 3.

---

## Option 2: Using psql from Local Machine (If you have PostgreSQL client)

If you have `psql` installed locally and can access RDS from your IP:

1. **Get database password:**
   ```bash
   aws secretsmanager get-secret-value \
       --secret-id syncflow/database/password \
       --region ap-south-1 \
       --query 'SecretString' \
       --output text
   ```

2. **Get RDS endpoint:**
   ```bash
   DB_HOST=$(aws rds describe-db-instances \
       --db-instance-identifier syncflow-db \
       --region ap-south-1 \
       --query 'DBInstances[0].Endpoint.Address' \
       --output text)
   echo "RDS Endpoint: $DB_HOST"
   ```

3. **Connect and create database:**
   ```bash
   PGPASSWORD='<your-password>' psql \
       -h $DB_HOST \
       -p 5432 \
       -U postgres \
       -d postgres \
       -c "CREATE DATABASE syncflow;"
   ```

**Note:** This requires:
- RDS security group allows your local IP (port 5432)
- PostgreSQL client (`psql`) installed on your machine
- SSL connection may be required

---

## Option 3: Using AWS Systems Manager (SSM) + EC2 Instance

If you have an EC2 instance in the same VPC:

1. **Install PostgreSQL client on EC2:**
   ```bash
   sudo yum install postgresql15 -y
   # or
   sudo apt-get install postgresql-client -y
   ```

2. **Get database credentials:**
   ```bash
   export DB_PASSWORD=$(aws secretsmanager get-secret-value \
       --secret-id syncflow/database/password \
       --region ap-south-1 \
       --query 'SecretString' \
       --output text)
   
   export DB_HOST=$(aws rds describe-db-instances \
       --db-instance-identifier syncflow-db \
       --region ap-south-1 \
       --query 'DBInstances[0].Endpoint.Address' \
       --output text)
   ```

3. **Connect and create database:**
   ```bash
   PGPASSWORD=$DB_PASSWORD psql \
       -h $DB_HOST \
       -p 5432 \
       -U postgres \
       -d postgres \
       -c "CREATE DATABASE syncflow;"
   ```

---

## Option 4: Use the Automated Script I Created

I already created a script that does this automatically via ECS task. However, it had network connectivity issues. You can try running it again now that I've fixed the security group:

```bash
cd ecs
AWS_REGION=ap-south-1 ./create-database-in-rds.sh
```

**Note:** This requires the temporary ECS task to have network access to RDS.

---

## Option 5: Simplest - Let the Code Handle It! ✅

**Actually, the BEST option is to just deploy the code I already updated!**

The backend code I modified will **automatically create the database** on first startup if it doesn't exist. This is:
- ✅ **Zero manual steps**
- ✅ **Works automatically**
- ✅ **No need for external tools**
- ✅ **Handles race conditions**

Just commit and push:
```bash
git add backend/cmd/app/app.go
git commit -m "Fix: Auto-create database if it doesn't exist"
git push origin main
```

After deployment (2-3 minutes), the backend will:
1. Try to connect to "syncflow" database
2. If it doesn't exist → automatically create it
3. Continue with normal startup

---

## Recommendation

**Use Option 5** (let the code handle it) - it's the simplest and most reliable approach. The code change I made handles database creation automatically, so you don't need to do anything manually.

If you prefer manual creation, **Option 1** (RDS Query Editor v2) is the easiest if available in your account.

---

## Verify Database Creation

After creating the database (any method), verify it exists:

```bash
# Get credentials
DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region ap-south-1 \
    --query 'SecretString' \
    --output text)

DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region ap-south-1 \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

# List databases
PGPASSWORD=$DB_PASSWORD psql \
    -h $DB_HOST \
    -p 5432 \
    -U postgres \
    -d postgres \
    -c "\l" | grep syncflow
```

Or check via backend logs after deployment:
```bash
aws logs tail /ecs/syncflow-backend --region ap-south-1 --since 5m
```

You should see: `✓ Database 'syncflow' created successfully` (first time) or `Database connected successfully` (subsequent times).

