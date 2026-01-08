# Starting the Application

## Prerequisites

1. **Go** (1.19+) - For backend
2. **Node.js** (18+) and **npm** - For frontend
3. **PostgreSQL** - Database

## Quick Start

### Option 1: Start Both Servers (Recommended)

```bash
./start-all.sh
```

This will start both backend (port 8080) and frontend (port 3000) servers.

### Option 2: Start Servers Separately

**Terminal 1 - Backend:**
```bash
./start-backend.sh
# OR
cd backend && go run cmd/server/main.go
```

**Terminal 2 - Frontend:**
```bash
./start-frontend.sh
# OR
cd frontend && npm install && npm run dev
```

## Manual Start

### Backend Server

```bash
cd backend
go mod tidy  # Install dependencies
go run cmd/server/main.go
```

Backend will run on: `http://localhost:8080`

### Frontend Server

```bash
cd frontend
npm install  # First time only
npm run dev
```

Frontend will run on: `http://localhost:3000`

## Environment Variables

### Backend

Create `backend/.env`:
```env
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=syncflow
DB_PORT=5432
PORT=8080
```

### Frontend

Create `frontend/.env`:
```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_GOOGLE_CLIENT_ID=your-google-client-id
VITE_GOOGLE_REDIRECT_URI=http://localhost:8080/api/oauth/google/callback
```

## Troubleshooting

### Backend won't start
- Check PostgreSQL is running
- Verify database credentials in `.env`
- Run migrations: Check `backend/migrations/` directory

### Frontend won't start
- Install Node.js: `brew install node` (macOS) or download from nodejs.org
- Run `npm install` in frontend directory
- Check port 3000 is not in use

### Port already in use
- Backend: Change `PORT` in backend `.env`
- Frontend: Change port in `vite.config.ts`

## Access the Application

- **Frontend UI**: http://localhost:3000
- **Backend API**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/health


