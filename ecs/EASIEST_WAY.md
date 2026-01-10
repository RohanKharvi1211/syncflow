# Easiest Way to Create Database

## ⚡ Quick Answer: Let the Code Handle It! (Recommended)

The code I already updated will **automatically create the database** when the backend starts. Just deploy:

```bash
git add backend/cmd/app/app.go
git commit -m "Fix: Auto-create database if it doesn't exist"
git push origin main
```

**That's it!** After 2-3 minutes, the database will be created automatically. No manual steps needed.

---

## If You Want to Do It Manually

### Option A: AWS Console (If Query Editor v2 is Available)

1. **Go to AWS Console** → **RDS** → **Databases** → Click `syncflow-db`
2. **Click "Query Editor v2"** in the left sidebar
3. **Connect** to database `postgres` with username `postgres`
4. **Get password** from Secrets Manager or run:
   ```bash
   aws secretsmanager get-secret-value \
       --secret-id syncflow/database/password \
       --region ap-south-1 \
       --query 'SecretString' \
       --output text
   ```
5. **Run SQL:**
   ```sql
   CREATE DATABASE syncflow;
   ```

**Note:** Query Editor v2 may not be available in all accounts/regions. If you don't see it, use Option B.

---

### Option B: From Your Local Machine (Requires PostgreSQL Client)

If you have `psql` installed locally:

```bash
cd ecs
./create-db-simple.sh
```

**If you don't have `psql`:**
- **macOS:** `brew install postgresql`
- **Ubuntu/Debian:** `sudo apt-get install postgresql-client`
- **Windows:** Download PostgreSQL installer

---

### Option C: Use AWS CloudShell (No Installation Needed!)

1. **Open AWS Console** → **CloudShell** (icon in top right)
2. **Install PostgreSQL client:**
   ```bash
   sudo yum install postgresql15 -y
   # or
   sudo apt-get update && sudo apt-get install -y postgresql-client
   ```
3. **Run the script:**
   ```bash
   cd /path/to/datatransfer/ecs
   chmod +x create-db-simple.sh
   ./create-db-simple.sh
   ```

---

## My Recommendation

**Just deploy the code!** It's the simplest and most reliable. The database will be created automatically on first backend startup.

You don't need to:
- ❌ Install any tools
- ❌ Connect to databases manually
- ❌ Run SQL commands
- ❌ Deal with network/security groups

Just push the code and it works! 🚀

