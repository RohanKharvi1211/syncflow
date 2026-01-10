# How to Connect to RDS Database from Local Machine

## Prerequisites

1. **PostgreSQL client installed** (`psql` command)
2. **RDS security group allows your IP** (port 5432)
3. **Database credentials** from AWS Secrets Manager

---

## Step 1: Get Database Credentials

```bash
# Get database host
DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region ap-south-1 \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

echo "Database Host: $DB_HOST"

# Get database password
DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region ap-south-1 \
    --query 'SecretString' \
    --output text)

echo "Password retrieved (hidden for security)"

# Get database name
DB_NAME=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/name \
    --region ap-south-1 \
    --query 'SecretString' \
    --output text)

echo "Database Name: $DB_NAME"

# Username is typically 'postgres'
DB_USER="postgres"
```

**Or get all at once:**
```bash
export DB_HOST=$(aws rds describe-db-instances --db-instance-identifier syncflow-db --region ap-south-1 --query 'DBInstances[0].Endpoint.Address' --output text)
export DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id syncflow/database/password --region ap-south-1 --query 'SecretString' --output text)
export DB_NAME=$(aws secretsmanager get-secret-value --secret-id syncflow/database/name --region ap-south-1 --query 'SecretString' --output text)
export DB_USER="postgres"
export DB_PORT="5432"
```

---

## Step 2: Allow Your IP in RDS Security Group

**Important:** RDS security group must allow connections from your local IP address.

### Get Your Public IP:
```bash
curl -s ifconfig.me
# or
curl -s https://api.ipify.org
```

### Update RDS Security Group:
```bash
MY_IP=$(curl -s ifconfig.me)
echo "Your IP: $MY_IP"

# Allow your IP to connect to RDS (port 5432)
aws ec2 authorize-security-group-ingress \
    --group-id sg-0d156ead6b2f3fa37 \
    --protocol tcp \
    --port 5432 \
    --cidr ${MY_IP}/32 \
    --region ap-south-1 2>&1
```

**If you get "rule already exists" error, that's fine - you already have access.**

**To check current rules:**
```bash
aws ec2 describe-security-groups \
    --group-ids sg-0d156ead6b2f3fa37 \
    --region ap-south-1 \
    --query 'SecurityGroups[0].IpPermissions[*].{Port:FromPort,CIDR:IpRanges[0].CidrIp}' \
    --output table
```

---

## Step 3: Install PostgreSQL Client (If Needed)

### macOS:
```bash
brew install postgresql
```

### Ubuntu/Debian:
```bash
sudo apt-get update
sudo apt-get install postgresql-client
```

### Windows:
Download and install PostgreSQL from: https://www.postgresql.org/download/windows/

### Verify Installation:
```bash
psql --version
```

---

## Step 4: Connect to Database

### Option A: Using psql Command Line

**Connect to default 'postgres' database:**
```bash
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p 5432 \
    -U "$DB_USER" \
    -d postgres \
    --set=sslmode=require
```

**Connect to 'syncflow' database (if it exists):**
```bash
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p 5432 \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --set=sslmode=require
```

**Interactive mode (it will prompt for password):**
```bash
psql -h "$DB_HOST" -p 5432 -U "$DB_USER" -d postgres -W
# Enter password when prompted
```

**One-line command to create database:**
```bash
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p 5432 \
    -U "$DB_USER" \
    -d postgres \
    --set=sslmode=require \
    -c "CREATE DATABASE syncflow;"
```

---

### Option B: Using Connection String

```bash
export PGPASSWORD="$DB_PASSWORD"
psql "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/postgres?sslmode=require"
```

**For syncflow database:**
```bash
psql "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=require"
```

---

### Option C: Using .pgpass File (No Password Prompts)

Create `~/.pgpass` file with your credentials:

```bash
cat > ~/.pgpass <<EOF
${DB_HOST}:5432:*:${DB_USER}:${DB_PASSWORD}
EOF

chmod 600 ~/.pgpass
```

**Then connect without password:**
```bash
psql -h "$DB_HOST" -p 5432 -U "$DB_USER" -d postgres -W
```

---

## Step 5: Common Commands After Connecting

Once connected, you can run SQL commands:

```sql
-- List all databases
\l

-- Switch to syncflow database
\c syncflow

-- List all tables
\dt

-- Describe a table
\d table_name

-- Run a query
SELECT * FROM users LIMIT 10;

-- Exit
\q
```

---

## Step 6: Create Database Manually (If Needed)

If the database doesn't exist, connect to 'postgres' database and create it:

```bash
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p 5432 \
    -U "$DB_USER" \
    -d postgres \
    --set=sslmode=require \
    -c "CREATE DATABASE syncflow;"
```

**Verify it was created:**
```bash
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p 5432 \
    -U "$DB_USER" \
    -d postgres \
    --set=sslmode=require \
    -c "\l" | grep syncflow
```

---

## GUI Tools (Easier Visual Interface)

### Option 1: pgAdmin (Free, Open Source)

1. **Download:** https://www.pgadmin.org/download/
2. **Install and open pgAdmin**
3. **Right-click "Servers"** → **Create** → **Server**
4. **General tab:**
   - Name: `SyncFlow RDS`
5. **Connection tab:**
   - Host: `syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com`
   - Port: `5432`
   - Maintenance database: `postgres`
   - Username: `postgres`
   - Password: (get from Secrets Manager)
   - **SSL Mode: Require** ← Important!
6. **Save** and connect

---

### Option 2: DBeaver (Free, Universal Database Tool)

1. **Download:** https://dbeaver.io/download/
2. **Open DBeaver** → **New Database Connection**
3. **Select PostgreSQL**
4. **Fill in:**
   - Host: `syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com`
   - Port: `5432`
   - Database: `postgres` (or `syncflow` if it exists)
   - Username: `postgres`
   - Password: (get from Secrets Manager)
   - **Enable SSL** → **SSL Mode: require**
5. **Test Connection** → **Finish**

---

### Option 3: TablePlus (macOS/Windows, Paid with Free Trial)

1. **Download:** https://tableplus.com/
2. **Create new connection** → **PostgreSQL**
3. **Fill in credentials** (same as above)
4. **Enable SSL** → **SSL Mode: Require**
5. **Connect**

---

## Quick Script to Connect

Save this as `connect-to-rds.sh`:

```bash
#!/bin/bash
# Quick script to connect to RDS

AWS_REGION=${AWS_REGION:-ap-south-1}

echo "Getting database credentials..."
export DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

export DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region $AWS_REGION \
    --query 'SecretString' \
    --output text)

export DB_USER="postgres"
export DB_NAME="syncflow"
export DB_PORT="5432"

echo "Connecting to: $DB_HOST"
echo "Database: $DB_NAME"
echo ""

# Check if database exists, if not connect to postgres
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p 5432 -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" --set=sslmode=require >/dev/null 2>&1; then
    echo "✓ Connecting to existing database: $DB_NAME"
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p 5432 \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --set=sslmode=require
else
    echo "⚠️  Database '$DB_NAME' doesn't exist yet. Connecting to 'postgres' database."
    echo "   To create the database, run:"
    echo "   CREATE DATABASE syncflow;"
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p 5432 \
        -U "$DB_USER" \
        -d postgres \
        --set=sslmode=require
fi
```

**Make it executable and run:**
```bash
chmod +x connect-to-rds.sh
./connect-to-rds.sh
```

---

## Troubleshooting

### Error: "Connection timed out"
- **Check:** RDS security group allows your IP (Step 2)
- **Check:** Your IP hasn't changed (dynamic IPs change)
- **Solution:** Update security group with new IP

### Error: "SSL connection required"
- **Fix:** Add `--set=sslmode=require` to psql command
- **Or:** Use connection string with `?sslmode=require`

### Error: "Password authentication failed"
- **Check:** Password is correct from Secrets Manager
- **Check:** Username is `postgres` (default master user)
- **Solution:** Re-verify credentials

### Error: "Database does not exist"
- **Solution:** Connect to `postgres` database first, then create it:
  ```sql
  CREATE DATABASE syncflow;
  ```

### Error: "psql: command not found"
- **Solution:** Install PostgreSQL client (Step 3)

---

## Summary: Quickest Way

1. **Get your IP and allow it:**
   ```bash
   MY_IP=$(curl -s ifconfig.me)
   aws ec2 authorize-security-group-ingress \
       --group-id sg-0d156ead6b2f3fa37 \
       --protocol tcp \
       --port 5432 \
       --cidr ${MY_IP}/32 \
       --region ap-south-1
   ```

2. **Connect:**
   ```bash
   DB_HOST=$(aws rds describe-db-instances --db-instance-identifier syncflow-db --region ap-south-1 --query 'DBInstances[0].Endpoint.Address' --output text)
   DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id syncflow/database/password --region ap-south-1 --query 'SecretString' --output text)
   
   PGPASSWORD="$DB_PASSWORD" psql \
       -h "$DB_HOST" \
       -p 5432 \
       -U postgres \
       -d postgres \
       --set=sslmode=require
   ```

3. **Create database (if needed):**
   ```sql
   CREATE DATABASE syncflow;
   \c syncflow
   ```

**Or use a GUI tool like pgAdmin or DBeaver for easier management!**

