# Setup Instructions

## Quick Start

### 1. Database Setup

**Option A: Use setup script (recommended)**
```bash
cd backend
./setup_db.sh
```

**Option B: Manual setup**
```bash
# Create database
createdb syncflow

# Or if you need to use a different user:
psql -U postgres -c "CREATE DATABASE syncflow;"
```

### 2. Configure Environment Variables

Create a `.env` file in the `backend/` directory:

```env
# Database - Use your actual PostgreSQL credentials
DB_HOST=localhost
DB_USER=your_postgres_user
DB_PASSWORD=your_password
DB_NAME=syncflow
DB_PORT=5432

# Server
PORT=8080

# Google OAuth (get from Google Cloud Console)
GOOGLE_CLIENT_ID=your_client_id
GOOGLE_CLIENT_SECRET=your_client_secret
GOOGLE_REDIRECT_URI=http://localhost:8080/api/oauth/google/callback
```

### 3. Start Backend Server

```bash
cd backend
go run ./cmd/server/main.go
```

The server will:
- Connect to PostgreSQL
- Run database migrations automatically
- Start on port 8080

### 4. Start Frontend Server

In a new terminal:
```bash
cd frontend
python3 -m http.server 3000
```

### 5. Access Application

Open your browser and go to: `http://localhost:3000`

## Troubleshooting

### "Error loading users"

**Possible causes:**
1. Backend server is not running
   - Solution: Start the backend server (step 3)

2. Database connection failed
   - Solution: Check PostgreSQL is running and credentials are correct
   - Test: `psql -U your_user -d syncflow -c "SELECT 1;"`

3. Database doesn't exist
   - Solution: Run `./setup_db.sh` or create manually

### "role postgres does not exist"

This means the PostgreSQL user "postgres" doesn't exist on your system. Options:

1. **Use your current user:**
   ```bash
   createdb syncflow
   # Then update .env: DB_USER=$(whoami)
   ```

2. **Create postgres user:**
   ```bash
   createuser -s postgres
   ```

3. **Use existing PostgreSQL user:**
   - Find your user: `psql -l`
   - Update `.env` with that user

### Database Connection Issues

Test your database connection:
```bash
psql -U your_user -d syncflow -c "SELECT version();"
```

If this works, use the same credentials in your `.env` file.

## First Time Setup Checklist

- [ ] PostgreSQL is installed and running
- [ ] Database `syncflow` is created
- [ ] `.env` file is configured with correct database credentials
- [ ] Backend server starts without errors
- [ ] Frontend server is running on port 3000
- [ ] Can access http://localhost:3000 without errors

