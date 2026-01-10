# DBeaver Connection Settings (Now Working!)

## ✅ RDS is Now Publicly Accessible!

The RDS instance is now publicly accessible and the connection works!

---

## DBeaver Connection Settings

### Step 1: Create New Connection
- Open DBeaver
- Click **"New Database Connection"** (database icon with +)
- Select **PostgreSQL** → Click **Next**

---

### Step 2: Main Settings Tab

```
Connection name: SyncFlow RDS
Host:            syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com
Port:            5432
Database:        postgres  (or syncflow if you created it)
Authentication:  Database native
Username:        postgres
Password:        [Get from Secrets Manager - see command below]
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

### Step 3: SSL Settings Tab (CRITICAL!)

Click on the **"SSL"** tab:

```
☑ Use SSL
SSL Mode:        require  ← MUST BE THIS!
SSL Factory:     [default/leave as is]
SSL Root Cert:   [empty/leave empty]
SSL Cert:        [empty/leave empty]
SSL Key:         [empty/leave empty]
```

**Key Setting:** **SSL Mode: `require`** ← This is required!

---

### Step 4: Test Connection

1. Click **"Test Connection"** button at the bottom
2. DBeaver may prompt to download PostgreSQL driver - click **"Download"**
3. Wait for driver download to complete
4. Click **"Test Connection"** again

**Expected Result:**
- ✅ Green checkmark
- ✅ "Connected (PostgreSQL 15.15)"

---

### Step 5: Save and Connect

- Click **"Finish"** to save connection
- Double-click the connection to connect

---

## Quick Connection Test (From Terminal)

To verify connection works from command line:

```bash
cd ecs
./connect-to-rds.sh
```

This should connect successfully now!

---

## Creating the syncflow Database (If Needed)

If the `syncflow` database doesn't exist yet:

1. **Connect to `postgres` database first** (using DBeaver or terminal)
2. **Run this SQL:**
   ```sql
   CREATE DATABASE syncflow;
   ```
3. **Refresh** DBeaver connection tree (right-click connection → Refresh)
4. **You should now see `syncflow` database**
5. **Create a new connection** or **edit existing** to point to `syncflow` database

---

## Complete Connection Summary

✅ **Host:** `syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com`  
✅ **Port:** `5432`  
✅ **Database:** `postgres` (or `syncflow` after creating it)  
✅ **Username:** `postgres`  
✅ **Password:** (from Secrets Manager)  
✅ **SSL Mode:** `require` ← **This is the key setting!**  
✅ **Publicly Accessible:** Yes (fixed!)  
✅ **Security Group:** Allows your IP (already configured)

---

## What Was Fixed

1. ✅ **RDS is now publicly accessible** - This was the main issue!
2. ✅ **Security group allows your IP** - Already configured
3. ✅ **SSL mode must be `require`** - This is required for RDS
4. ✅ **Connection tested and working** - Verified via terminal

---

## If Connection Still Fails

1. **Check SSL Mode is `require`** (not `disable`, `allow`, or `prefer`)
2. **Verify password is correct:**
   ```bash
   aws secretsmanager get-secret-value \
       --secret-id syncflow/database/password \
       --region ap-south-1 \
       --query 'SecretString' \
       --output text
   ```
3. **Check security group has your current IP** (if IP changed):
   ```bash
   cd ecs
   ./setup-local-db-access.sh
   ```
4. **Verify RDS is publicly accessible:**
   ```bash
   aws rds describe-db-instances \
       --db-instance-identifier syncflow-db \
       --region ap-south-1 \
       --query 'DBInstances[0].PubliclyAccessible' \
       --output text
   # Should return: True
   ```

---

## Success!

The connection should work now in both:
- ✅ **DBeaver** (with SSL Mode: `require`)
- ✅ **Command line** (`./connect-to-rds.sh`)

Try connecting in DBeaver now - it should work! 🎉

