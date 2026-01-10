# DBeaver Connection Settings for RDS PostgreSQL

## Step-by-Step DBeaver Setup

### 1. Create New Connection
- Open DBeaver
- Click **"New Database Connection"** (database icon with +)
- Select **PostgreSQL** → Click **Next**

---

### 2. Main Settings Tab

Fill in the following:

```
Host:     syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com
Port:     5432
Database: postgres    (use 'postgres' first, then switch to 'syncflow' after creating it)
Username: postgres
Password: [Get from Secrets Manager - see below]
```

**Get Password:**
```bash
aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region ap-south-1 \
    --query 'SecretString' \
    --output text
```

---

### 3. SSL Settings Tab (IMPORTANT!)

This is the critical part! Click on the **"SSL"** tab:

#### SSL Mode:
Select: **`require`**

(Options you might see: `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full`)
- **Use `require`** - This forces SSL connection which RDS requires

#### SSL Factory:
Leave as default: **`org.postgresql.ssl.DefaultJavaSSLFactory`**

#### SSL Root Certificate:
- **Leave empty** or select "Default" for `require` mode
- Only needed if using `verify-ca` or `verify-full` (not needed for basic RDS connection)

#### SSL Certificate:
- **Leave empty**

#### SSL Key:
- **Leave empty**

**Summary for SSL Tab:**
- ✅ **SSL Mode: `require`** ← This is the key setting!
- ✅ Leave all other SSL fields empty/default

---

### 4. Advanced Settings (Optional)

If connection still fails, check these in **"Advanced"** tab:

```
Connection timeout (ms): 5000
Query timeout (ms): 10000
```

---

### 5. Test Connection

- Click **"Test Connection"** button at bottom
- DBeaver may prompt to download PostgreSQL driver - click **"Download"**
- Wait for driver download to complete
- Click **"Test Connection"** again

**Expected Result:**
- ✅ Green checkmark: "Connected"
- ✅ "Connected (PostgreSQL 15.15)"

**If it fails:**
- Check error message
- Verify SSL Mode is set to `require`
- Verify password is correct
- Verify security group allows your IP

---

### 6. Save and Connect

- Click **"Finish"** to save connection
- Double-click the connection to connect

---

## Complete Connection String (for reference)

```
jdbc:postgresql://syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com:5432/postgres?ssl=true&sslmode=require
```

---

## Troubleshooting

### Error: "SSL is required but was not enabled"
**Solution:**
- Go to SSL tab
- Set SSL Mode to `require`

### Error: "Connection refused" or "Connection timed out"
**Solution:**
1. Verify your IP is allowed in RDS security group:
   ```bash
   cd ecs
   ./setup-local-db-access.sh
   ```

2. Check if security group rule exists:
   ```bash
   aws ec2 describe-security-groups \
       --group-ids sg-0d156ead6b2f3fa37 \
       --region ap-south-1 \
       --query 'SecurityGroups[0].IpPermissions[*].{Port:FromPort,CIDR:IpRanges[0].CidrIp}' \
       --output table
   ```

### Error: "Password authentication failed"
**Solution:**
- Get fresh password from Secrets Manager:
  ```bash
  aws secretsmanager get-secret-value \
      --secret-id syncflow/database/password \
      --region ap-south-1 \
      --query 'SecretString' \
      --output text
  ```
- Make sure username is `postgres` (not `syncflow` or anything else)

### Error: "Database does not exist"
**Solution:**
1. Connect to `postgres` database first (this always exists)
2. After connecting, create the `syncflow` database:
   ```sql
   CREATE DATABASE syncflow;
   ```
3. Then create a new connection pointing to `syncflow` database

---

## Quick Connection Checklist

- [ ] Host: `syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com`
- [ ] Port: `5432`
- [ ] Database: `postgres` (start with this)
- [ ] Username: `postgres`
- [ ] Password: (correct from Secrets Manager)
- [ ] **SSL Mode: `require`** ← Most important!
- [ ] Security group allows your IP (run `./setup-local-db-access.sh` if not sure)
- [ ] PostgreSQL driver downloaded in DBeaver

---

## After Successful Connection

### Create the syncflow database:

1. In DBeaver, with connection to `postgres` database active, open SQL Editor
2. Run:
   ```sql
   CREATE DATABASE syncflow;
   ```
3. Refresh database tree (right-click connection → Refresh)
4. You should now see `syncflow` database in the list
5. Double-click `syncflow` database to switch to it

### Or create new connection to syncflow:

1. Right-click your existing connection → **Copy**
2. Right-click → **Paste**
3. Edit connection → Change Database from `postgres` to `syncflow`
4. Test connection → Finish

---

## Visual Guide

**Main Tab:**
```
Connection name: SyncFlow RDS
Host:            syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com
Port:            5432
Database:        postgres
Authentication:  Database native
Username:        postgres
Password:        [your-password]
Show all databases: [optional checkbox]
```

**SSL Tab:**
```
☑ Use SSL
SSL Mode:        require  ← SELECT THIS!
SSL Factory:     [default]
SSL Root Cert:   [empty]
SSL Cert:        [empty]
SSL Key:         [empty]
SSL Password:    [empty]
```

---

## Summary

**The key setting you're missing is:**

**SSL Mode: `require`** 

Set this in the **SSL tab** when creating/editing the connection in DBeaver.

Once you set SSL Mode to `require` and have the correct host, port, username, and password, it should connect successfully!

